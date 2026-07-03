package work_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

// TestListSummaries_GoldenEquivalence seeds the full /tasks domain enumerated
// by packet WORK-TASKS-INCREMENTAL-FOLD-DESIGN-001 D1/D1a and deep-compares
// the batched ListSummaries output against a per-task oracle assembled from
// the single-task methods (GetStatus + GetCompatibilityStatus + factReadiness
// + Readiness for the gates/ready half). Every field must match for every
// task. This is the oracle for the Task 1 batch restructure: it must fail
// ONLY on the D1a empty-body-gate rows against pre-restructure code (the
// pre-existing list-path fail-open), proving both the bug and the oracle's
// discriminating power, then pass completely once batchStatus adopts the
// non-empty-body gate rule.
func TestListSummaries_GoldenEquivalence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	type seeded struct {
		name string
		id   types.EventID
	}
	var tasks []seeded

	mustCreate := func(title string) types.EventID {
		t.Helper()
		task, err := ts.Create(testActor, title, "desc: "+title, causes, testConv)
		if err != nil {
			t.Fatalf("Create %q: %v", title, err)
		}
		return task.ID
	}

	// 1. Plain freshly-created task: StatusCreated, LegacyStatusPending, no
	// assignee, not blocked, no artifacts, not ready (missing all gates).
	plain := mustCreate("plain freshly created task")
	tasks = append(tasks, seeded{"plain", plain})

	// 2. Task with all three required gates present with non-empty bodies:
	// ready (gates satisfied, no facts required).
	readyTask := mustCreate("task with all gates satisfied")
	for _, label := range work.RequiredReadinessGateLabels() {
		if err := ts.AddArtifact(testActor, readyTask, label, "text/markdown", "concrete "+label+" body", causes, testConv); err != nil {
			t.Fatalf("AddArtifact %s: %v", label, err)
		}
	}
	tasks = append(tasks, seeded{"ready_all_gates_nonempty", readyTask})

	// 3. Task with a required gate present but EMPTY BODY — the D1a
	// discriminating row. Old batchStatus counts this gate as present
	// (label-only); the documented Readiness contract does not.
	emptyBodyTask := mustCreate("task with empty-body required gate")
	for i, label := range work.RequiredReadinessGateLabels() {
		body := "non-empty body for " + label
		if i == 0 {
			body = "   " // whitespace-only — must NOT satisfy the gate.
		}
		if err := ts.AddArtifact(testActor, emptyBodyTask, label, "text/markdown", body, causes, testConv); err != nil {
			t.Fatalf("AddArtifact %s: %v", label, err)
		}
	}
	tasks = append(tasks, seeded{"empty_body_gate", emptyBodyTask})

	// 3b. A second empty-body case: totally empty string body (not just
	// whitespace), to widen the discriminating-row coverage.
	emptyBodyTask2 := mustCreate("task with fully empty required gate body")
	for i, label := range work.RequiredReadinessGateLabels() {
		body := "non-empty body for " + label
		if i == 1 {
			body = ""
		}
		if err := ts.AddArtifact(testActor, emptyBodyTask2, label, "text/markdown", body, causes, testConv); err != nil {
			t.Fatalf("AddArtifact %s: %v", label, err)
		}
	}
	tasks = append(tasks, seeded{"empty_body_gate_2", emptyBodyTask2})

	// 4. Task assigned to an actor (legacy status "assigned").
	assignedTask := mustCreate("task assigned to an actor")
	if err := ts.Assign(testActor, assignedTask, testActor, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	tasks = append(tasks, seeded{"assigned", assignedTask})

	// 5. Task completed via waiver (legacy status "completed").
	waivedCompleteTask := mustCreate("task completed via waiver")
	if err := ts.WaiveArtifact(testActor, waivedCompleteTask, "no artifact needed", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}
	if err := ts.Complete(testActor, waivedCompleteTask, "done via waiver", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tasks = append(tasks, seeded{"waived_completed", waivedCompleteTask})

	// 6. Task completed with an artifact (legacy status "completed").
	completedTask := mustCreate("task completed with artifact")
	completeWithArtifact(t, ts, testActor, completedTask, "done", causes, testConv)
	tasks = append(tasks, seeded{"completed", completedTask})

	// 7. Task completed then REOPENED — must read as pending/open again, not
	// completed (IADA-3: reopen pairing).
	reopenedTask := mustCreate("task completed then reopened")
	completeWithArtifact(t, ts, testActor, reopenedTask, "first pass", causes, testConv)
	if err := ts.Reopen(testActor, reopenedTask, "needs fixes", []string{"issue 1"}, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	tasks = append(tasks, seeded{"reopened_after_completion", reopenedTask})

	// 7b. Task completed, reopened, then RE-completed — must read as
	// completed again (a fresh completion event, never referenced by a
	// reopen, is live).
	recompletedTask := mustCreate("task completed, reopened, re-completed")
	completeWithArtifact(t, ts, testActor, recompletedTask, "first pass", causes, testConv)
	if err := ts.Reopen(testActor, recompletedTask, "needs fixes", []string{"issue 1"}, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if err := ts.AddArtifact(testActor, recompletedTask, "result", "text/plain", "fixed", causes, testConv); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}
	if err := ts.Complete(testActor, recompletedTask, "second pass", causes, testConv); err != nil {
		t.Fatalf("Complete (re-completion): %v", err)
	}
	tasks = append(tasks, seeded{"recompleted_after_reopen", recompletedTask})

	// 8. Dependency SATISFIED: blocker completed before the dependent is
	// created's downstream check — dependent should not be blocked.
	depSatisfiedBlocker := mustCreate("dependency blocker (will be completed)")
	completeWithArtifact(t, ts, testActor, depSatisfiedBlocker, "blocker done", causes, testConv)
	depSatisfiedDependent := mustCreate("dependent with satisfied dependency")
	if err := ts.AddDependency(testActor, depSatisfiedDependent, depSatisfiedBlocker, causes, testConv); err != nil {
		t.Fatalf("AddDependency (satisfied): %v", err)
	}
	tasks = append(tasks, seeded{"dependency_satisfied_blocker", depSatisfiedBlocker})
	tasks = append(tasks, seeded{"dependency_satisfied_dependent", depSatisfiedDependent})

	// 9. Dependency UNSATISFIED: blocker still open — dependent is blocked.
	depUnsatisfiedBlocker := mustCreate("dependency blocker (still open)")
	depUnsatisfiedDependent := mustCreate("dependent with unsatisfied dependency")
	if err := ts.AddDependency(testActor, depUnsatisfiedDependent, depUnsatisfiedBlocker, causes, testConv); err != nil {
		t.Fatalf("AddDependency (unsatisfied): %v", err)
	}
	tasks = append(tasks, seeded{"dependency_unsatisfied_blocker", depUnsatisfiedBlocker})
	tasks = append(tasks, seeded{"dependency_unsatisfied_dependent", depUnsatisfiedDependent})

	// 10. Dependency unsatisfied but explicitly UNBLOCKED — not blocked.
	unblockedBlocker := mustCreate("dependency blocker (open, but dependent unblocked)")
	unblockedDependent := mustCreate("dependent explicitly unblocked")
	if err := ts.AddDependency(testActor, unblockedDependent, unblockedBlocker, causes, testConv); err != nil {
		t.Fatalf("AddDependency (unblocked): %v", err)
	}
	if err := ts.UnblockTask(testActor, unblockedDependent, causes, testConv); err != nil {
		t.Fatalf("UnblockTask: %v", err)
	}
	tasks = append(tasks, seeded{"dependency_unblocked_blocker", unblockedBlocker})
	tasks = append(tasks, seeded{"dependency_unblocked_dependent", unblockedDependent})

	// 11. Fact requirement PRESENT/satisfied via a causal event (not an
	// exact event ID pin — Descendants-based satisfaction).
	factSatisfiedTask := mustCreate("task with satisfied fact requirement")
	factTypeSatisfied := types.MustEventType("authority.decision.recorded")
	if err := ts.AddFactRequirement(testActor, factSatisfiedTask, factTypeSatisfied, types.EventID{}, "requires authority decision", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement (satisfied): %v", err)
	}
	appendPhase3Fact(t, s, factTypeSatisfied, []types.EventID{factSatisfiedTask})
	tasks = append(tasks, seeded{"fact_present_satisfied", factSatisfiedTask})

	// 12. Fact requirement MISSING — never satisfied.
	factMissingTask := mustCreate("task with missing fact requirement")
	factTypeMissing := types.MustEventType("agent.identity.registered")
	if err := ts.AddFactRequirement(testActor, factMissingTask, factTypeMissing, types.EventID{}, "requires agent registration", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement (missing): %v", err)
	}
	tasks = append(tasks, seeded{"fact_missing", factMissingTask})

	// 13. Waiver present WITHOUT completion (waived but not yet completed):
	// Waived=true, but legacy status still not completed since Complete()
	// was never called.
	waivedOnlyTask := mustCreate("task with waiver but not completed")
	if err := ts.WaiveArtifact(testActor, waivedOnlyTask, "not needed", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact (waived only): %v", err)
	}
	tasks = append(tasks, seeded{"waived_only_not_completed", waivedOnlyTask})

	// 14. Issue-scan certified stage task: canonical/factory IDs carry the
	// deterministic issue-scan prefixes and the task is transitioned all the
	// way to StatusCertified. Per dependencySatisfiedIDs, a certified
	// issue-scan task satisfies downstream dependency edges even without a
	// legacy completion event.
	issueScanTask, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:                  "issue-scan certified stage task",
		CanonicalTaskID:        "tsk_issue_scan_run1_target1_stage1",
		FactoryOrderID:         "fo_issue_scan_run1",
		RequirementIDs:         []string{"req_issue_scan_1"},
		AcceptanceCriterionIDs: []string{"ac_issue_scan_1"},
		Cell:                   "implementation",
		RiskClass:              "low",
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39 (issue-scan): %v", err)
	}
	for _, to := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified, work.StatusCertified} {
		if err := ts.TransitionTask(testActor, issueScanTask.ID, to, "advance to "+string(to), nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask -> %s: %v", to, err)
		}
	}
	tasks = append(tasks, seeded{"issue_scan_certified", issueScanTask.ID})

	// 15. Dependent on the issue-scan certified task: should read as NOT
	// blocked (dependencySatisfiedIDs treats certified issue-scan tasks as
	// satisfying downstream deps).
	issueScanDependent := mustCreate("dependent on issue-scan certified task")
	if err := ts.AddDependency(testActor, issueScanDependent, issueScanTask.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency (issue-scan dependent): %v", err)
	}
	tasks = append(tasks, seeded{"issue_scan_dependent", issueScanDependent})

	// 16. A v3.9-transitioned but NOT certified task (e.g. StatusRunning) —
	// its own status; also confirm it does NOT satisfy downstream deps
	// (dependencySatisfiedIDs only credits certified + issue-scan).
	runningTask, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title: "v3.9 task in running state",
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39 (running): %v", err)
	}
	if err := ts.TransitionTask(testActor, runningTask.ID, work.StatusReady, "advance", nil, causes, testConv); err != nil {
		t.Fatalf("TransitionTask -> ready: %v", err)
	}
	if err := ts.TransitionTask(testActor, runningTask.ID, work.StatusRunning, "advance", nil, causes, testConv); err != nil {
		t.Fatalf("TransitionTask -> running: %v", err)
	}
	tasks = append(tasks, seeded{"v39_running_not_certified", runningTask.ID})

	// 17. Linked task: created plain, then linked with new canonical/factory
	// IDs after creation — List()/batch fold must reconcile against the
	// newest work.task.linked event.
	linkedTask := mustCreate("task linked after creation")
	if err := ts.LinkTask(testActor, linkedTask, work.TaskLinkage{
		CanonicalTaskID:        "tsk_linked_after",
		FactoryOrderID:         "fo_linked_after",
		RequirementIDs:         []string{"req_1"},
		AcceptanceCriterionIDs: []string{"ac_1"},
	}, causes, testConv); err != nil {
		t.Fatalf("LinkTask: %v", err)
	}
	tasks = append(tasks, seeded{"linked_after_creation", linkedTask})

	// Sanity: ensure the store has at least as many tasks as we seeded, and
	// List(limit) covers all of them (use a high limit).
	const limit = 1000
	summaries, err := ts.ListSummaries(limit)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) < len(tasks) {
		t.Fatalf("ListSummaries returned %d tasks, want at least %d seeded", len(summaries), len(tasks))
	}

	summaryByID := make(map[types.EventID]work.TaskSummary, len(summaries))
	for _, sum := range summaries {
		summaryByID[sum.ID] = sum
	}

	var failCount int
	var failNames []string

	for _, tk := range tasks {
		got, ok := summaryByID[tk.id]
		if !ok {
			t.Errorf("[%s] task %s missing from ListSummaries output", tk.name, tk.id.Value())
			continue
		}

		// Assemble the oracle summary from the single-task methods.
		status, err := ts.GetStatus(tk.id)
		if err != nil {
			t.Fatalf("[%s] GetStatus: %v", tk.name, err)
		}
		legacyStatus, err := ts.GetCompatibilityStatus(tk.id)
		if err != nil {
			t.Fatalf("[%s] GetCompatibilityStatus: %v", tk.name, err)
		}
		legacyProjection, err := ts.ProjectLegacyTask(tk.id)
		if err != nil {
			t.Fatalf("[%s] ProjectLegacyTask: %v", tk.name, err)
		}
		readiness, err := ts.Readiness(tk.id)
		if err != nil {
			t.Fatalf("[%s] Readiness: %v", tk.name, err)
		}
		artifacts, err := ts.ListArtifacts(tk.id)
		if err != nil {
			t.Fatalf("[%s] ListArtifacts: %v", tk.name, err)
		}
		hasWaiver, err := ts.HasWaiver(tk.id)
		if err != nil {
			t.Fatalf("[%s] HasWaiver: %v", tk.name, err)
		}

		wantMissingGates := readiness.MissingGates
		wantMissingFacts := readiness.MissingFacts
		wantReady := readiness.Ready

		diffs := diffSummaryFields(got, work.TaskSummary{
			Task:          got.Task, // Task field asserted separately below (identity, not oracle-derived here)
			Status:        status,
			LegacyStatus:  legacyStatus,
			Assignee:      legacyProjection.Assignee,
			Blocked:       legacyProjection.Blocked,
			ArtifactCount: len(artifacts),
			Waived:        hasWaiver,
			Ready:         wantReady,
			MissingGates:  wantMissingGates,
			MissingFacts:  wantMissingFacts,
		})

		if len(diffs) > 0 {
			failCount++
			failNames = append(failNames, tk.name)
			for _, d := range diffs {
				t.Errorf("[%s] task %s: %s", tk.name, tk.id.Value(), d)
			}
		}
	}

	if failCount > 0 {
		sort.Strings(failNames)
		t.Logf("GOLDEN EQUIVALENCE red-state summary: %d/%d rows diverged: %v", failCount, len(tasks), failNames)
	}
}

// TestListSummaries_MissingFactsNeverNil pins the JSON wire shape of the
// missing_facts (and missing_gates) fields: EVERY summary returned by BOTH
// ListSummaries (batch path, store.go batchStatus) and ListSummariesCached
// (fold path, store_fold_cache.go summariesFromFold) must carry a non-nil
// slice, so it serializes as [] and never as null. The golden-equivalence
// and interleaved tests normalize nil->[] before DeepEqual, which makes
// them structurally blind to exactly this regression — this test asserts
// the raw slices directly, with no normalization. It fails on a batch path
// that defaults missingFacts to a nil slice for tasks without fact
// requirements.
func TestListSummaries_MissingFactsNeverNil(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	// Task WITHOUT any fact requirement — the path where a nil default
	// would survive to the output untouched.
	if _, err := ts.Create(testActor, "task without fact requirements", "", causes, testConv); err != nil {
		t.Fatalf("Create (no facts): %v", err)
	}

	// Task WITH a (missing) fact requirement — exercises the factReadiness
	// overwrite path, which returns a non-nil slice by construction.
	withFacts, err := ts.Create(testActor, "task with fact requirement", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create (with facts): %v", err)
	}
	factType := types.MustEventType("agent.identity.registered")
	if err := ts.AddFactRequirement(testActor, withFacts.ID, factType, types.EventID{}, "requires agent registration", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}

	batch, err := ts.ListSummaries(100)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	cached, err := ts.ListSummariesCached(100)
	if err != nil {
		t.Fatalf("ListSummariesCached: %v", err)
	}

	for name, list := range map[string][]work.TaskSummary{
		"ListSummaries":       batch,
		"ListSummariesCached": cached,
	} {
		if len(list) < 2 {
			t.Fatalf("%s returned %d summaries, want at least 2", name, len(list))
		}
		for _, sum := range list {
			if sum.MissingFacts == nil {
				t.Errorf("%s: task %s MissingFacts is nil — serializes as JSON null, want [] (empty slice)", name, sum.Task.ID.Value())
			}
			if sum.MissingGates == nil {
				t.Errorf("%s: task %s MissingGates is nil — serializes as JSON null, want [] (empty slice)", name, sum.Task.ID.Value())
			}
		}
	}
}

// diffSummaryFields deep-compares the computed fields of two TaskSummary
// values (excluding the embedded Task identity fields, which are asserted by
// map-lookup identity in the caller) and returns a human-readable diff per
// mismatched field.
func diffSummaryFields(got, want work.TaskSummary) []string {
	var diffs []string
	if got.Status != want.Status {
		diffs = append(diffs, fmt.Sprintf("Status = %q, want %q", got.Status, want.Status))
	}
	if got.LegacyStatus != want.LegacyStatus {
		diffs = append(diffs, fmt.Sprintf("LegacyStatus = %q, want %q", got.LegacyStatus, want.LegacyStatus))
	}
	if got.Assignee != want.Assignee {
		diffs = append(diffs, fmt.Sprintf("Assignee = %v, want %v", got.Assignee, want.Assignee))
	}
	if got.Blocked != want.Blocked {
		diffs = append(diffs, fmt.Sprintf("Blocked = %v, want %v", got.Blocked, want.Blocked))
	}
	if got.ArtifactCount != want.ArtifactCount {
		diffs = append(diffs, fmt.Sprintf("ArtifactCount = %d, want %d", got.ArtifactCount, want.ArtifactCount))
	}
	if got.Waived != want.Waived {
		diffs = append(diffs, fmt.Sprintf("Waived = %v, want %v", got.Waived, want.Waived))
	}
	if got.Ready != want.Ready {
		diffs = append(diffs, fmt.Sprintf("Ready = %v, want %v", got.Ready, want.Ready))
	}
	if !reflect.DeepEqual(sortedCopy(got.MissingGates), sortedCopy(want.MissingGates)) {
		diffs = append(diffs, fmt.Sprintf("MissingGates = %#v, want %#v", got.MissingGates, want.MissingGates))
	}
	if !reflect.DeepEqual(sortedCopy(got.MissingFacts), sortedCopy(want.MissingFacts)) {
		diffs = append(diffs, fmt.Sprintf("MissingFacts = %#v, want %#v", got.MissingFacts, want.MissingFacts))
	}
	return diffs
}

func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
