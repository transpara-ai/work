package work

import (
	"fmt"
	"strings"
	"sync"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"golang.org/x/sync/singleflight"
)

// foldEventTypes is the complete /tasks fold domain (packet
// WORK-TASKS-INCREMENTAL-FOLD-DESIGN-001 D2): every event type that
// contributes to a TaskSummary EXCEPT fact readiness, which is excluded from
// the memo (CFADA1-2 — factReadiness resolves via Get/Descendants causal
// queries a per-type frontier cannot see) and is recomputed fresh on every
// request, narrowed to the fact-requiring task set folded here.
func foldEventTypes() []types.EventType {
	return []types.EventType{
		EventTypeTaskCreated,
		EventTypeTaskLinked,
		EventTypeTaskAssigned,
		EventTypeTaskCompleted,
		EventTypeTaskReopened,
		EventTypeTaskDependencyAdded,
		EventTypeTaskUnblocked,
		EventTypeTaskArtifact,
		EventTypeTaskArtifactWaived,
		EventTypeTaskLifecycleTransitioned,
		EventTypeTaskFactRequired,
	}
}

// taskFoldState is the complete incrementally-maintained fold over the
// /tasks domain (D2). It is rebuilt from scratch the first time and then
// topped up by frontier increments on every request whose observed head
// differs from the held stable head.
//
// Memory bound: every map/slice here is keyed or indexed by task creation
// EventID (or references one), so total retained memory is O(N) where N is
// the count of work.task.* events actually observed across the eleven
// folded types in foldEventTypes — proportional to graph size for this
// domain, not to request volume. There is no unbounded per-request growth:
// each request either reads the held state (zero allocation growth) or
// tops it up by exactly the new events since the last fold and then either
// replaces the held generation (stable) or discards its own scratch copy
// (provisional). No TTL; entries are never evicted except by a full
// scratch rebuild on a fail-closed error path.
type taskFoldState struct {
	// createdOldestFirst holds every work.task.created event in OLDEST-FIRST
	// (natural append) order — the order events are applied during both a
	// full scratch rebuild and an increment. Appending here is O(1)
	// amortized, unlike maintaining newest-first order which would require
	// an O(n) shift per insertion (O(n^2) for a full rebuild at live
	// scale). listTasksFold walks this slice backward to reconstruct
	// List(limit)'s newest-first contract. createdByID indexes the same
	// Tasks by ID for O(1) existence checks and per-task lookups.
	createdOldestFirst []Task
	createdByID        map[types.EventID]Task

	// newestLink is the newest work.task.linked overlay per task (keyed by
	// the task's creation-event ID), applied the same way List() applies it:
	// override a linkage field only when the newest link value is non-empty.
	newestLink map[types.EventID]TaskLinkedContent

	// assigneeMap holds the current (newest) assignee per task.
	assigneeMap map[types.EventID]types.ActorID

	// completions and reopens are the RAW per-task event ID lists (IADA-3):
	// collapsing to a boolean would mis-apply a reopen that arrives for a
	// completion folded before the reopen existed. liveCompletedIDs is
	// derived from these two on every read (cheap: O(tasks with
	// completions)), never stored as a standalone boolean.
	completions map[types.EventID][]types.EventID // taskID -> completion event IDs (append-order)
	reopenRefs  map[types.EventID]bool            // completion event ID -> true if referenced by any reopen

	// dependencyPairs is every (taskID, dependsOnID) pair observed, in the
	// order folded. unblocked is the explicit unblock override set.
	dependencyPairs []dependencyPair
	unblocked       map[types.EventID]bool

	// artifactCount and gatesByTask mirror batchStatus scan 5 exactly,
	// including the D1a non-empty-body rule.
	artifactCount map[types.EventID]int
	gatesByTask   map[types.EventID]map[string]bool

	// waived is the waiver-event-observed set (scan 6).
	waived map[types.EventID]bool

	// lifecycleStatus is the newest-wins v3.9 lifecycle status per task
	// (scan 7 / latestLifecycleStatuses).
	lifecycleStatus map[types.EventID]TaskStatus

	// factRequiringTasks is the (rare) set of tasks carrying at least one
	// work.task.fact.required event (scan 8). Fact SATISFACTION is never
	// folded here — only membership, so the per-request factReadiness pass
	// knows which tasks to call.
	factRequiringTasks map[types.EventID]bool

	// frontier holds, per folded event type, the newest event ID observed
	// during the last fold (nil/absent if no event of that type has ever
	// been seen). Incrementing pages newest-first until this ID is reached.
	frontier map[types.EventType]types.EventID

	// generation is a monotonically increasing counter assigned at
	// promotion time, used ONLY to prove ordering in tests; production
	// no-promotion safety uses head TimestampMS comparison (see promote).
	generation uint64

	// stableHead is the head event ID this state is memoized under. A
	// zero EventID with headSet=true represents a validly-folded EMPTY
	// store (no events yet).
	stableHead types.EventID
	headSet    bool
}

// dependencyPair is one work.task.dependency.added edge: taskID depends on dependsOnID.
type dependencyPair struct {
	taskID      types.EventID
	dependsOnID types.EventID
}

// newTaskFoldState returns an empty fold state ready to be topped up from
// scratch (i.e. as if every frontier were unset).
func newTaskFoldState() *taskFoldState {
	return &taskFoldState{
		createdByID:        make(map[types.EventID]Task),
		newestLink:         make(map[types.EventID]TaskLinkedContent),
		assigneeMap:        make(map[types.EventID]types.ActorID),
		completions:        make(map[types.EventID][]types.EventID),
		reopenRefs:         make(map[types.EventID]bool),
		unblocked:          make(map[types.EventID]bool),
		artifactCount:      make(map[types.EventID]int),
		gatesByTask:        make(map[types.EventID]map[string]bool),
		waived:             make(map[types.EventID]bool),
		lifecycleStatus:    make(map[types.EventID]TaskStatus),
		factRequiringTasks: make(map[types.EventID]bool),
		frontier:           make(map[types.EventType]types.EventID),
	}
}

// clone deep-copies the fold state so a provisional (non-memoized) fold can
// be assembled from a stable base without mutating the held generation that
// concurrent readers may still be observing.
func (f *taskFoldState) clone() *taskFoldState {
	out := newTaskFoldState()
	out.createdOldestFirst = append([]Task(nil), f.createdOldestFirst...)
	for k, v := range f.createdByID {
		out.createdByID[k] = v
	}
	for k, v := range f.newestLink {
		out.newestLink[k] = v
	}
	for k, v := range f.assigneeMap {
		out.assigneeMap[k] = v
	}
	for k, v := range f.completions {
		out.completions[k] = append([]types.EventID(nil), v...)
	}
	for k, v := range f.reopenRefs {
		out.reopenRefs[k] = v
	}
	out.dependencyPairs = append([]dependencyPair(nil), f.dependencyPairs...)
	for k, v := range f.unblocked {
		out.unblocked[k] = v
	}
	for k, v := range f.artifactCount {
		out.artifactCount[k] = v
	}
	for k, v := range f.gatesByTask {
		gc := make(map[string]bool, len(v))
		for label, ok := range v {
			gc[label] = ok
		}
		out.gatesByTask[k] = gc
	}
	for k, v := range f.waived {
		out.waived[k] = v
	}
	for k, v := range f.lifecycleStatus {
		out.lifecycleStatus[k] = v
	}
	for k, v := range f.factRequiringTasks {
		out.factRequiringTasks[k] = v
	}
	for k, v := range f.frontier {
		out.frontier[k] = v
	}
	out.generation = f.generation
	out.stableHead = f.stableHead
	out.headSet = f.headSet
	return out
}

// applyEvent applies a single event to the fold state (last-write-wins per
// IADA-2 semantics: this is only correct when events of a given type are
// applied in OLDEST-TO-NEWEST order, which is the contract every caller in
// this file honors — see foldIncrement / foldFromScratch).
func (f *taskFoldState) applyEvent(ev event.Event) {
	switch c := ev.Content().(type) {
	case TaskCreatedContent:
		p := c.Priority
		if p == "" {
			p = DefaultPriority
		}
		t := Task{
			ID:                     ev.ID(),
			Title:                  c.Title,
			Description:            c.Description,
			CreatedBy:              c.CreatedBy,
			Priority:               p,
			Workspace:              c.Workspace,
			CanonicalTaskID:        c.CanonicalTaskID,
			FactoryOrderID:         c.FactoryOrderID,
			RequirementIDs:         cloneStrings(c.RequirementIDs),
			AcceptanceCriterionIDs: cloneStrings(c.AcceptanceCriterionIDs),
			Cell:                   c.Cell,
			RiskClass:              c.RiskClass,
			ExpectedOutputs:        cloneStrings(c.ExpectedOutputs),
			CreatedAt:              ev.Timestamp().Value(),
		}
		if _, exists := f.createdByID[t.ID]; !exists {
			// Append: applyEvent is always called oldest->newest (see the
			// doc comment above), so appending preserves oldest-first order
			// in O(1) amortized time.
			f.createdOldestFirst = append(f.createdOldestFirst, t)
		}
		f.createdByID[t.ID] = t

	case TaskLinkedContent:
		// Overlay is newest-wins; applying oldest->newest means the LAST
		// applied write for a task is retained, which is newest — correct.
		f.newestLink[c.TaskID] = c

	case TaskAssignedContent:
		f.assigneeMap[c.TaskID] = c.AssignedTo

	case TaskCompletedContent:
		f.completions[c.TaskID] = append(f.completions[c.TaskID], ev.ID())

	case TaskReopenedContent:
		for _, ref := range c.CompletionRefs {
			f.reopenRefs[ref] = true
		}

	case TaskDependencyContent:
		f.dependencyPairs = append(f.dependencyPairs, dependencyPair{taskID: c.TaskID, dependsOnID: c.DependsOnID})

	case TaskUnblockedContent:
		f.unblocked[c.TaskID] = true

	case TaskArtifactContent:
		f.artifactCount[c.TaskID]++
		label := normalizeGateLabel(c.Label)
		if isRequiredGateLabel(label) && strings.TrimSpace(c.Body) != "" {
			if f.gatesByTask[c.TaskID] == nil {
				f.gatesByTask[c.TaskID] = make(map[string]bool)
			}
			f.gatesByTask[c.TaskID][label] = true
		}

	case TaskArtifactWaivedContent:
		f.waived[c.TaskID] = true

	case TaskLifecycleTransitionContent:
		// Newest-wins: applying oldest->newest, the last write wins — correct.
		f.lifecycleStatus[c.TaskID] = c.ToState

	case TaskFactRequiredContent:
		f.factRequiringTasks[c.TaskID] = true
	}
}

// liveCompletedIDsFold derives the reopen-aware legacy-completion set from
// the fold's raw completion/reopen aggregates (mirrors
// TaskStore.liveCompletedIDs exactly, but reads from held/incremented state
// instead of re-scanning the store).
func (f *taskFoldState) liveCompletedIDsFold() map[types.EventID]bool {
	ids := make(map[types.EventID]bool, len(f.completions))
	for taskID, refs := range f.completions {
		for _, ref := range refs {
			if !f.reopenRefs[ref] {
				ids[taskID] = true
				break
			}
		}
	}
	return ids
}

// dependencySatisfiedIDsFold mirrors TaskStore.dependencySatisfiedIDs: legacy
// live completion OR certified issue-scan task.
func (f *taskFoldState) dependencySatisfiedIDsFold() map[types.EventID]bool {
	ids := f.liveCompletedIDsFold()
	for taskID, status := range f.lifecycleStatus {
		if status == StatusCertified && isIssueScanTaskContent(taskContentOf(f.createdByID[taskID])) {
			ids[taskID] = true
		}
	}
	return ids
}

// taskContentOf reconstructs the minimal TaskCreatedContent fields
// isIssueScanTaskContent inspects (CanonicalTaskID/FactoryOrderID prefixes)
// from a folded Task, so the fold layer can reuse the exact oracle
// predicate without re-deriving its logic.
func taskContentOf(t Task) TaskCreatedContent {
	return TaskCreatedContent{CanonicalTaskID: t.CanonicalTaskID, FactoryOrderID: t.FactoryOrderID}
}

// blockedFold mirrors batchStatus scan 3+4: a task is blocked if it has at
// least one dependency edge whose target is not dependency-satisfied, AND
// the task has not been explicitly unblocked.
func (f *taskFoldState) blockedFold(satisfiedIDs map[types.EventID]bool) map[types.EventID]bool {
	blocked := make(map[types.EventID]bool)
	for _, pair := range f.dependencyPairs {
		if !satisfiedIDs[pair.dependsOnID] {
			blocked[pair.taskID] = true
		}
	}
	return blocked
}

// listTasksFold reconstructs List(limit) from the fold state:
// createdOldestFirst is maintained oldest-first (see its doc comment), so
// this walks backward from the end to reproduce ByType's newest-first page
// order, with the newestLink overlay applied exactly as List() applies it.
func (f *taskFoldState) listTasksFold(limit int) []Task {
	if limit <= 0 {
		limit = 20
	}
	n := limit
	if n > len(f.createdOldestFirst) {
		n = len(f.createdOldestFirst)
	}
	out := make([]Task, 0, n)
	for i := len(f.createdOldestFirst) - 1; i >= 0 && len(out) < n; i-- {
		t := f.createdOldestFirst[i]
		if lc, ok := f.newestLink[t.ID]; ok {
			if lc.CanonicalTaskID != "" {
				t.CanonicalTaskID = lc.CanonicalTaskID
			}
			if lc.FactoryOrderID != "" {
				t.FactoryOrderID = lc.FactoryOrderID
			}
			if len(lc.RequirementIDs) > 0 {
				t.RequirementIDs = cloneStrings(lc.RequirementIDs)
			}
			if len(lc.AcceptanceCriterionIDs) > 0 {
				t.AcceptanceCriterionIDs = cloneStrings(lc.AcceptanceCriterionIDs)
			}
		}
		out = append(out, t)
	}
	return out
}

// summariesFromFold assembles TaskSummary values for the given tasks
// (typically listTasksFold's output) purely from held/incremented fold
// state — no store reads except the per-request fact pass the caller
// performs separately (facts are excluded from the memo, CFADA1-2).
func (f *taskFoldState) summariesFromFold(tasks []Task) []TaskSummary {
	if len(tasks) == 0 {
		return nil
	}
	satisfiedIDs := f.dependencySatisfiedIDsFold()
	blockedMap := f.blockedFold(satisfiedIDs)

	summaries := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		status, ok := f.lifecycleStatus[t.ID]
		if !ok {
			status = StatusCreated
		}
		assignee := f.assigneeMap[t.ID]
		blocked := blockedMap[t.ID] && !f.unblocked[t.ID]
		missing := missingRequiredGates(f.gatesByTask[t.ID])

		summaries = append(summaries, TaskSummary{
			Task:          t,
			Status:        status,
			Assignee:      assignee,
			Blocked:       blocked,
			ArtifactCount: f.artifactCount[t.ID],
			Waived:        f.waived[t.ID],
			MissingGates:  missing,
			// MissingFacts must be an EMPTY (non-nil) slice for JSON shape
			// stability: the pre-fold code always got a non-nil slice from
			// factReadiness, serializing as []; nil would serialize as null.
			MissingFacts: []string{},
			// LegacyStatus, Ready are finished by the caller once the
			// per-request fact pass has run (facts are excluded from the
			// fold — see foldEventTypes doc comment).
		})
	}
	return summaries
}

// finalizeLegacyStatusAndReadiness fills LegacyStatus/Ready/MissingFacts on
// each summary AFTER the caller has computed MissingFacts per task (the
// narrowed per-request fact pass), replicating batchStatus's legacy-status
// switch and Ready derivation exactly.
func (f *taskFoldState) finalizeLegacyStatusAndReadiness(summaries []TaskSummary) {
	liveCompletedIDs := f.liveCompletedIDsFold()
	for i := range summaries {
		s := &summaries[i]
		ready := len(s.MissingGates) == 0 && len(s.MissingFacts) == 0
		s.Ready = ready
		switch {
		case liveCompletedIDs[s.Task.ID]:
			s.LegacyStatus = LegacyStatusCompleted
		case s.Blocked:
			s.LegacyStatus = LegacyStatusBlocked
		case !s.Assignee.IsZero():
			s.LegacyStatus = LegacyStatusAssigned
		case ready:
			s.LegacyStatus = LegacyStatusReady
		default:
			s.LegacyStatus = LegacyStatusPending
		}
	}
}

// foldCache is the process-memory-only, head-keyed fold-generation memo
// layer for TaskStore.ListSummariesCached. NO TTL: entries are held
// indefinitely and only ever discarded by a fail-closed rebuild. State does
// not survive process restart (cold fold on next request after restart is
// correct and expected).
type foldCache struct {
	mu    sync.Mutex
	state *taskFoldState // nil until the first successful fold

	// group runs rebuild/increment flights keyed by the OBSERVED head
	// (EventID.Value(), or a sentinel for the empty store) — CFADA1-5:
	// requests that observed different heads never share a flight, so no
	// caller can receive a fold older than the head it itself observed.
	group singleflight.Group
}

func newFoldCache() *foldCache {
	return &foldCache{}
}

// ListSummariesCached is the incrementally-memoized equivalent of
// ListSummaries, wired ONLY into the GET /tasks handler (D2). Read order:
//
//  1. Read the store head.
//  2. If a stable held generation exists AND its head equals the observed
//     head, assemble summaries directly from held state (zero store scans
//     for the folded domain).
//  3. Otherwise, run a singleflight-deduplicated rebuild/increment KEYED BY
//     THE OBSERVED HEAD: requests that observed different heads never share
//     a flight (CFADA1-5), so no caller can receive a fold older than the
//     head it itself observed.
//  4. After folding, re-read the head. headBefore == headAfter -> the fold
//     is STABLE and is promoted into the held generation (subject to the
//     no-promotion-over-newer guarantee below). Otherwise the fold is
//     PROVISIONAL: it is still served (at least as fresh as headBefore) but
//     is never memoized, so the next request's head check misses and tops
//     up again — skew self-heals within one request cycle.
//
// Facts are ALWAYS computed fresh per request (CFADA1-2), narrowed to the
// fact-requiring task set the fold identifies — this call never caches
// fact satisfaction.
//
// Fail-closed (D2, enumerated): a head-read error, a page-read error, a
// frontier-not-found-after-exhaustion, or a decode error during folding
// discards any held state and forces a full scratch rebuild; if the
// scratch rebuild ALSO fails, ListSummariesCached returns the error (no
// stale serve — identical to today's per-request failure behavior).
func (ts *TaskStore) ListSummariesCached(limit int) ([]TaskSummary, error) {
	if ts.fold == nil {
		ts.fold = newFoldCache()
	}
	return ts.fold.listSummaries(ts.store, ts, limit)
}

// listSummaries implements the ListSummariesCached algorithm against an
// injectable taskStoreReader-compatible store.Store, so the fail-closed
// table test can pass an error-injecting fake without a full Store.
func (fc *foldCache) listSummaries(s store.Store, ts *TaskStore, limit int) ([]TaskSummary, error) {
	headBefore, err := s.Head()
	if err != nil {
		return fc.scratchRebuildAndServe(s, ts, limit, err)
	}
	headBeforeID := headEventID(headBefore)

	fc.mu.Lock()
	held := fc.state
	fc.mu.Unlock()

	if held != nil && held.headSet && held.stableHead == headBeforeID {
		// Hit: assemble directly from held state (facts computed fresh below).
		return fc.assembleAndFinishFacts(ts, held, limit)
	}

	// Miss: singleflight-deduplicated rebuild/increment keyed by the
	// OBSERVED head (CFADA1-5). types.EventID{}.Value() is "" for an empty
	// store, which is a valid, distinct flight key on its own (there is
	// only ever one "empty store" state to key against).
	flightKey := headBeforeID.Value()
	resultAny, err, _ := fc.group.Do(flightKey, func() (any, error) {
		return fc.rebuildOrIncrement(s, headBeforeID)
	})
	if err != nil {
		// The flight itself failed (page/frontier error) -> fail-closed:
		// discard held state and attempt one full scratch rebuild.
		return fc.scratchRebuildAndServe(s, ts, limit, err)
	}
	newState := resultAny.(*taskFoldState)
	return fc.assembleAndFinishFacts(ts, newState, limit)
}

// rebuildOrIncrement is the singleflight-wrapped body: clone-or-build,
// increment via frontiers, re-read the head, and promote if stable (subject
// to no-promotion-over-newer). Returns the state this caller should serve
// from (its own copy — never a state a concurrent promotion could still be
// mutating).
func (fc *foldCache) rebuildOrIncrement(s store.Store, headBeforeID types.EventID) (*taskFoldState, error) {
	fc.mu.Lock()
	held := fc.state
	fc.mu.Unlock()

	var working *taskFoldState
	if held == nil {
		fresh, err := foldFromScratch(s)
		if err != nil {
			return nil, err
		}
		working = fresh
	} else {
		// Clone so concurrent readers of the currently-held stable
		// generation are never mutated out from under them.
		clone := held.clone()
		incremented, err := foldIncrement(s, clone)
		if err != nil {
			return nil, err
		}
		working = incremented
	}

	headAfter, err := s.Head()
	if err != nil {
		return nil, err
	}
	headAfterID := headEventID(headAfter)

	working.stableHead = headAfterID
	working.headSet = true

	if headAfterID == headBeforeID {
		// Stable: eligible for promotion.
		fc.promote(working)
	}
	// Provisional folds (headAfterID != headBeforeID) are served but never
	// promoted — the caller gets `working` directly, which is at least as
	// fresh as headBeforeID.
	return working, nil
}

// promote installs candidate as the held generation UNLESS the currently
// held generation observes a head that is the same as or newer than
// candidate's — the no-promotion guarantee (CFADA1-1/adv1): a finished
// flight that started against an older head must never clobber a newer
// stable generation that another (faster) flight already promoted.
//
// Ordering is decided by types.EventID.TimestampMS() (UUIDv7 IDs are
// time-ordered) with an equality guard: if both heads carry the identical
// millisecond timestamp (extremely unlikely for genuinely different
// events, but not impossible under coarse clocks or concurrent appends)
// candidate is NOT promoted over an existing held generation — ties favor
// keeping the already-held state rather than risking a spurious downgrade,
// since two different heads can never be proven ordered from the tie alone.
func (fc *foldCache) promote(candidate *taskFoldState) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	current := fc.state
	if current == nil || !current.headSet {
		candidate.generation = 1
		fc.state = candidate
		return
	}
	if current.stableHead == candidate.stableHead {
		// Same head already held (e.g. a duplicate flight for the same
		// key, or a re-promotion) — still safe to replace since content is
		// identical by construction of the fold; keep generation moving
		// forward for observability.
		candidate.generation = current.generation + 1
		fc.state = candidate
		return
	}
	if candidate.stableHead.TimestampMS() > current.stableHead.TimestampMS() {
		candidate.generation = current.generation + 1
		fc.state = candidate
		return
	}
	// candidate's head is older than, or ties with (and differs from), the
	// currently held newer head: refuse to promote. The currently held
	// generation remains authoritative.
}

// scratchRebuildAndServe is the fail-closed path (D2 enumerated rules): on
// ANY error from the fast path (head read, flight/page/frontier error), the
// held state is discarded and one full scratch rebuild is attempted. If
// that ALSO fails, the original error is returned — no stale serve.
func (fc *foldCache) scratchRebuildAndServe(s store.Store, ts *TaskStore, limit int, cause error) ([]TaskSummary, error) {
	fc.mu.Lock()
	fc.state = nil
	fc.mu.Unlock()

	fresh, err := foldFromScratch(s)
	if err != nil {
		return nil, fmt.Errorf("fold rebuild after fail-closed discard (cause: %v): %w", cause, err)
	}
	headAfter, err := s.Head()
	if err != nil {
		return nil, fmt.Errorf("fold rebuild after fail-closed discard (cause: %v): re-read head: %w", cause, err)
	}
	fresh.stableHead = headEventID(headAfter)
	fresh.headSet = true
	fc.promote(fresh)

	return fc.assembleAndFinishFacts(ts, fresh, limit)
}

// assembleAndFinishFacts builds TaskSummary rows from fold state, then runs
// the narrowed per-request fact pass (CFADA1-2 — never cached) and
// finalizes LegacyStatus/Ready.
func (fc *foldCache) assembleAndFinishFacts(ts *TaskStore, state *taskFoldState, limit int) ([]TaskSummary, error) {
	tasks := state.listTasksFold(limit)
	summaries := state.summariesFromFold(tasks)
	for i := range summaries {
		if !state.factRequiringTasks[summaries[i].Task.ID] {
			continue
		}
		_, missingFacts, err := ts.factReadiness(summaries[i].Task.ID)
		if err != nil {
			return nil, err
		}
		summaries[i].MissingFacts = missingFacts
	}
	state.finalizeLegacyStatusAndReadiness(summaries)
	return summaries, nil
}

// headEventID extracts the EventID from a Head() result, returning the
// zero EventID for an empty store (types.None case) — the documented
// "empty store/zero head" fold state (D2).
func headEventID(head types.Option[event.Event]) types.EventID {
	if head.IsNone() {
		return types.EventID{}
	}
	return head.Unwrap().ID()
}

// errFrontierNotFound is returned internally when paging newest-first for a
// folded event type never reaches a previously-seen frontier event ID
// within the paged window (i.e. paging ran to exhaustion without finding
// it). Per D2's enumerated fail-closed rules this forces a full scratch
// rebuild — it must never be treated as "no new events".
var errFrontierNotFound = fmt.Errorf("fold: frontier event not found after paging to exhaustion")

// newEventsSinceFrontier pages eventType newest-first via s.ByType until it
// reaches frontier (a previously-seen event ID for this type), or until the
// store reports no more pages. It returns the events strictly newer than
// frontier in OLDEST-TO-NEWEST order (ready for applyEvent) and the newest
// event ID seen (the new frontier for this type). If frontierSet is false,
// EVERY event of this type is "new" (first fold / scratch rebuild): paging
// runs to exhaustion and all events are returned oldest-to-newest.
//
// Returns errFrontierNotFound if frontierSet is true and paging exhausts
// the store without ever encountering the frontier ID — this can only
// happen if the store's pagination contract is violated (an event
// vanished, or newest-first ordering broke), so the caller must discard
// the held state and rebuild from scratch rather than risk a silently
// incomplete increment.
func newEventsSinceFrontier(s taskStoreReader, eventType types.EventType, frontier types.EventID, frontierSet bool) ([]event.Event, types.EventID, error) {
	var newestFirst []event.Event
	after := types.None[types.Cursor]()
	foundFrontier := false
	for {
		page, err := s.ByType(eventType, 1000, after)
		if err != nil {
			return nil, types.EventID{}, fmt.Errorf("page %s: %w", eventType.Value(), err)
		}
		stopThisPage := false
		for _, ev := range page.Items() {
			if frontierSet && ev.ID() == frontier {
				foundFrontier = true
				stopThisPage = true
				break
			}
			newestFirst = append(newestFirst, ev)
		}
		if stopThisPage {
			break
		}
		if !page.HasMore() {
			break
		}
		after = page.Cursor()
	}
	if frontierSet && !foundFrontier {
		return nil, types.EventID{}, errFrontierNotFound
	}

	newFrontier := frontier
	if len(newestFirst) > 0 {
		newFrontier = newestFirst[0].ID() // first item of newest-first slice is the newest overall
	}

	// Reverse to oldest-to-newest for correct last-write-wins application.
	oldestFirst := make([]event.Event, len(newestFirst))
	for i, ev := range newestFirst {
		oldestFirst[len(newestFirst)-1-i] = ev
	}
	return oldestFirst, newFrontier, nil
}

// taskStoreReader is the minimal store surface the fold layer reads. Kept
// as an interface (rather than depending on store.Store directly) purely so
// tests can inject read-erroring fakes for the fail-closed table without
// standing up a full Store implementation.
type taskStoreReader interface {
	Head() (types.Option[event.Event], error)
	ByType(eventType types.EventType, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error)
}

// foldFromScratch builds a brand-new taskFoldState by paging every folded
// event type to exhaustion (frontierSet=false — every event is "new").
func foldFromScratch(s taskStoreReader) (*taskFoldState, error) {
	state := newTaskFoldState()
	for _, et := range foldEventTypes() {
		events, newFrontier, err := newEventsSinceFrontier(s, et, types.EventID{}, false)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			state.applyEvent(ev)
		}
		if len(events) > 0 {
			state.frontier[et] = newFrontier
		}
	}
	return state, nil
}

// foldIncrement tops up base (which the caller must NOT mutate concurrently
// — pass a clone if base may still be read elsewhere) with events newer
// than each folded type's held frontier. Returns the incremented state.
func foldIncrement(s taskStoreReader, base *taskFoldState) (*taskFoldState, error) {
	for _, et := range foldEventTypes() {
		frontier, frontierSet := base.frontier[et]
		events, newFrontier, err := newEventsSinceFrontier(s, et, frontier, frontierSet)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			base.applyEvent(ev)
		}
		if len(events) > 0 {
			base.frontier[et] = newFrontier
		}
	}
	return base, nil
}
