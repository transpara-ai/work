package work

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// ErrArtifactRequired is returned by Complete when the task has neither an
// artifact nor an artifact waiver. Callers can check with errors.Is.
var ErrArtifactRequired = errors.New("task has no artifacts; attach an artifact or waive the requirement")

// ErrInvalidLifecycleTransition is returned when a requested v3.9 task state
// transition is not allowed from the current replayed state.
var ErrInvalidLifecycleTransition = errors.New("invalid task lifecycle transition")

// TaskStatus represents the canonical Dark Factory v3.9 lifecycle state of a task.
type TaskStatus string

const (
	// StatusCreated means the task record exists and has not entered scheduling.
	StatusCreated TaskStatus = "created"
	// StatusReady means the task is unblocked and has enough Work evidence to be scheduled.
	StatusReady TaskStatus = "ready"
	// StatusRunning means runtime or human production work is in progress.
	StatusRunning TaskStatus = "running"
	// StatusBlocked means dependency or evidence prerequisites block the task.
	StatusBlocked TaskStatus = "blocked"
	// StatusFailed means task verification or execution failed.
	StatusFailed TaskStatus = "failed"
	// StatusRepairRequired means a failure needs an explicit repair attempt.
	StatusRepairRequired TaskStatus = "repair_required"
	// StatusRepairRunning means a repair attempt is active.
	StatusRepairRunning TaskStatus = "repair_running"
	// StatusRepaired means repair output exists and is ready for verification.
	StatusRepaired TaskStatus = "repaired"
	// StatusVerificationRunning means verification evidence is being gathered.
	StatusVerificationRunning TaskStatus = "verification_running"
	// StatusVerified means verification passed.
	StatusVerified TaskStatus = "verified"
	// StatusCertified means the task has terminal certification evidence.
	StatusCertified TaskStatus = "certified"
	// StatusRejected means the task was explicitly rejected.
	StatusRejected TaskStatus = "rejected"
	// StatusSuperseded means the task was replaced by another canonical task record.
	StatusSuperseded TaskStatus = "superseded"
	// StatusPolicyBlocked means policy denied or paused the task.
	StatusPolicyBlocked TaskStatus = "policy_blocked"
)

// LegacyTaskStatus is the compatibility-only projection for pre-v3.9 Work
// events. These names are not canonical v3.9 TaskStatus values.
type LegacyTaskStatus string

const (
	LegacyStatusPending   LegacyTaskStatus = "pending"
	LegacyStatusAssigned  LegacyTaskStatus = "assigned"
	LegacyStatusCompleted LegacyTaskStatus = "completed"
	LegacyStatusBlocked   LegacyTaskStatus = "blocked"
	LegacyStatusReady     LegacyTaskStatus = "ready"
)

// Task represents a work item derived from a work.task.created event.
type Task struct {
	ID                     types.EventID
	Title                  string
	Description            string
	CreatedBy              types.ActorID
	Priority               TaskPriority
	Workspace              string
	CanonicalTaskID        string
	FactoryOrderID         string
	RequirementIDs         []string
	AcceptanceCriterionIDs []string
	Cell                   string
	RiskClass              string
	ExpectedOutputs        []string
}

// TaskSummary extends Task with computed state fields for efficient list views.
// Status, Assignee, Blocked, ArtifactCount, and Waived are populated by
// ListSummaries using batch store scans.
type TaskSummary struct {
	Task
	Status        TaskStatus
	LegacyStatus  LegacyTaskStatus
	Assignee      types.ActorID // zero value if unassigned
	Blocked       bool
	ArtifactCount int
	Waived        bool
	Ready         bool
	MissingGates  []string
	MissingFacts  []string
}

// TaskCreateOptions carries v3.9 Tier 0 lineage and scheduling metadata for a task.
type TaskCreateOptions struct {
	Title                  string
	Description            string
	Workspace              string
	Priority               TaskPriority
	CanonicalTaskID        string
	FactoryOrderID         string
	RequirementIDs         []string
	AcceptanceCriterionIDs []string
	Cell                   string
	RiskClass              string
	ExpectedOutputs        []string
}

// TaskLinkage is the replayed FactoryOrder -> Requirement -> AcceptanceCriterion -> Task linkage.
type TaskLinkage struct {
	CanonicalTaskID        string
	FactoryOrderID         string
	RequirementIDs         []string
	AcceptanceCriterionIDs []string
}

// VerificationEvidence is the replayed verification evidence attached to a task.
type VerificationEvidence struct {
	TestCaseIDs   []string
	TestRunIDs    []string
	GateResultIDs []string
	WaiverIDs     []string
}

// FailureRepairReferences is the replayed failure/repair evidence attached to a task.
type FailureRepairReferences struct {
	FailureIDs       []string
	RepairAttemptIDs []string
	WaiverIDs        []string
}

// TaskProjection is the v3.9 replayed operational view of a Work task.
type TaskProjection struct {
	Task
	Status              TaskStatus
	Assignee            types.ActorID
	Blocked             bool
	Ready               bool
	Linkage             TaskLinkage
	ModelOverrides      []FactoryOrderModelOverride
	Verification        VerificationEvidence
	FailureRepair       FailureRepairReferences
	SupersededBy        string
	LastTransitionEvent types.EventID
}

// LegacyTaskProjection replays historical Work events without promoting
// pending/assigned/completed into the canonical v3.9 lifecycle.
type LegacyTaskProjection struct {
	TaskID   types.EventID
	Status   LegacyTaskStatus
	Assignee types.ActorID
	Blocked  bool
	Ready    bool
}

// ArtifactEvent holds the data from a work.task.artifact event.
type ArtifactEvent struct {
	ID        types.EventID
	TaskID    types.EventID
	Label     string
	MediaType string
	Body      string
	CreatedBy types.ActorID
	Timestamp time.Time
}

// CommentEvent holds the data from a work.task.comment event.
type CommentEvent struct {
	ID        types.EventID
	TaskID    types.EventID
	Body      string
	AuthorID  types.ActorID
	Timestamp time.Time
}

// ReopenEvent holds the data from a work.task.reopened event.
type ReopenEvent struct {
	ID         types.EventID
	TaskID     types.EventID
	ReopenedBy types.ActorID
	Reason     string
	Issues     []string
	Timestamp  time.Time
}

// ChildTask records a direct task dependency edge where Task depends on ParentID.
// In the Work graph this means Task is a child/subtask of ParentID.
type ChildTask struct {
	Task
	ParentID            types.EventID
	DependencyEventID   types.EventID
	DependencyAddedBy   types.ActorID
	DependencyTimestamp time.Time
}

// SupersededTask records a duplicate child task that was closed in favor of
// an earlier canonical child under the same parent.
type SupersededTask struct {
	TaskID      types.EventID
	TaskTitle   string
	CanonicalID types.EventID
}

// TaskReadiness is the replayed readiness gate state for a task.
type TaskReadiness struct {
	TaskID       types.EventID
	Ready        bool
	PresentGates []string
	MissingGates []string
	PresentFacts []string
	MissingFacts []string
}

// FactRequirement is a task readiness prerequisite satisfied by an existing
// EventGraph fact of the required type, optionally pinned to an exact event ID.
type FactRequirement struct {
	ID                types.EventID
	TaskID            types.EventID
	RequiredEventType types.EventType
	RequiredEventID   types.EventID
	Reason            string
	RequiredBy        types.ActorID
	Timestamp         time.Time
	Satisfied         bool
}

// TaskStore creates and queries tasks as auditable events on the shared graph.
type TaskStore struct {
	store   store.Store
	factory *event.EventFactory
	signer  event.Signer
}

// NewTaskStore creates a new TaskStore backed by the given event store.
func NewTaskStore(s store.Store, factory *event.EventFactory, signer event.Signer) *TaskStore {
	return &TaskStore{store: s, factory: factory, signer: signer}
}

// Create records a work.task.created event on the graph and returns the task.
// The caller must supply at least one cause (typically the current chain head).
// An optional priority may be passed as the last argument; defaults to PriorityMedium.
func (ts *TaskStore) Create(
	source types.ActorID,
	title, description string,
	causes []types.EventID,
	convID types.ConversationID,
	priority ...TaskPriority,
) (Task, error) {
	return ts.create(source, TaskCreateOptions{
		Title:       title,
		Description: description,
		Priority:    firstPriority(priority),
	}, causes, convID)
}

// CreateV39 records a Work task with v3.9 Tier 0 lineage references.
func (ts *TaskStore) CreateV39(
	source types.ActorID,
	opts TaskCreateOptions,
	causes []types.EventID,
	convID types.ConversationID,
) (Task, error) {
	return ts.create(source, opts, causes, convID)
}

func (ts *TaskStore) create(
	source types.ActorID,
	opts TaskCreateOptions,
	causes []types.EventID,
	convID types.ConversationID,
) (Task, error) {
	title := opts.Title
	if title == "" {
		return Task{}, fmt.Errorf("title is required")
	}
	if err := validateTaskCreateOptions(opts); err != nil {
		return Task{}, err
	}
	p := opts.Priority
	if p == "" {
		p = DefaultPriority
	}
	content := TaskCreatedContent{
		Title:                  title,
		Description:            opts.Description,
		CreatedBy:              source,
		Priority:               p,
		Workspace:              opts.Workspace,
		CanonicalTaskID:        opts.CanonicalTaskID,
		FactoryOrderID:         opts.FactoryOrderID,
		RequirementIDs:         cloneStrings(opts.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(opts.AcceptanceCriterionIDs),
		Cell:                   opts.Cell,
		RiskClass:              opts.RiskClass,
		ExpectedOutputs:        cloneStrings(opts.ExpectedOutputs),
	}
	ev, err := ts.factory.Create(EventTypeTaskCreated, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return Task{}, fmt.Errorf("create task event: %w", err)
	}
	stored, err := ts.store.Append(ev)
	if err != nil {
		return Task{}, fmt.Errorf("append task event: %w", err)
	}
	return Task{
		ID:                     stored.ID(),
		Title:                  title,
		Description:            opts.Description,
		CreatedBy:              source,
		Priority:               p,
		Workspace:              opts.Workspace,
		CanonicalTaskID:        opts.CanonicalTaskID,
		FactoryOrderID:         opts.FactoryOrderID,
		RequirementIDs:         cloneStrings(opts.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(opts.AcceptanceCriterionIDs),
		Cell:                   opts.Cell,
		RiskClass:              opts.RiskClass,
		ExpectedOutputs:        cloneStrings(opts.ExpectedOutputs),
	}, nil
}

// List returns up to limit work.task.created events as Tasks.
func (ts *TaskStore) List(limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 20
	}
	page, err := ts.store.ByType(EventTypeTaskCreated, limit, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	tasks := make([]Task, 0, len(page.Items()))
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskCreatedContent)
		if !ok {
			continue
		}
		p := c.Priority
		if p == "" {
			p = DefaultPriority
		}
		tasks = append(tasks, Task{
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
		})
	}
	return tasks, nil
}

// CreateInWorkspace records a work.task.created event with a workspace label.
// The caller must supply at least one cause (typically the current chain head).
// An optional priority may be passed as the last argument; defaults to PriorityMedium.
func (ts *TaskStore) CreateInWorkspace(
	source types.ActorID,
	title, description, workspace string,
	causes []types.EventID,
	convID types.ConversationID,
	priority ...TaskPriority,
) (Task, error) {
	return ts.create(source, TaskCreateOptions{
		Title:       title,
		Description: description,
		Workspace:   workspace,
		Priority:    firstPriority(priority),
	}, causes, convID)
}

// ListByWorkspace returns up to limit tasks whose Workspace field matches the given workspace.
func (ts *TaskStore) ListByWorkspace(workspace string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 20
	}
	// Fetch a broad page; filter in-process since ByType has no predicate support.
	page, err := ts.store.ByType(EventTypeTaskCreated, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("list tasks by workspace: %w", err)
	}
	tasks := make([]Task, 0)
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskCreatedContent)
		if !ok || c.Workspace != workspace {
			continue
		}
		p := c.Priority
		if p == "" {
			p = DefaultPriority
		}
		tasks = append(tasks, Task{
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
		})
		if len(tasks) >= limit {
			break
		}
	}
	return tasks, nil
}

// ListSummariesByWorkspace returns up to limit workspace-scoped tasks with Status,
// Assignee, and Blocked populated via batch store scans.
func (ts *TaskStore) ListSummariesByWorkspace(workspace string, limit int) ([]TaskSummary, error) {
	tasks, err := ts.ListByWorkspace(workspace, limit)
	if err != nil {
		return nil, err
	}
	return ts.batchStatus(tasks)
}

// Assign records a work.task.assigned event on the graph.
// source is the actor performing the assignment (may equal assignee for self-assignment).
func (ts *TaskStore) Assign(
	source types.ActorID,
	taskID types.EventID,
	assignee types.ActorID,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	content := TaskAssignedContent{
		TaskID:     taskID,
		AssignedTo: assignee,
		AssignedBy: source,
	}
	ev, err := ts.factory.Create(EventTypeTaskAssigned, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create assign event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append assign event: %w", err)
	}
	return nil
}

// Complete records a work.task.completed event on the graph.
// source is the actor completing the task (typically the assignee).
//
// The artifact gate requires at least one work.task.artifact or
// work.task.artifact.waived event for the task. If neither exists,
// Complete returns ErrArtifactRequired.
func (ts *TaskStore) Complete(
	source types.ActorID,
	taskID types.EventID,
	summary string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	// --- Artifact gate (captures the event ID for ArtifactRef) ---
	artifactRef, hasArtifact, err := ts.findEventForTask(EventTypeTaskArtifact, taskID)
	if err != nil {
		return fmt.Errorf("check artifacts: %w", err)
	}
	if !hasArtifact {
		waiverRef, hasWaiver, err := ts.findEventForTask(EventTypeTaskArtifactWaived, taskID)
		if err != nil {
			return fmt.Errorf("check waivers: %w", err)
		}
		if !hasWaiver {
			return ErrArtifactRequired
		}
		artifactRef = waiverRef
	}

	content := TaskCompletedContent{
		TaskID:      taskID,
		CompletedBy: source,
		Summary:     summary,
		ArtifactRef: artifactRef,
	}
	ev, err := ts.factory.Create(EventTypeTaskCompleted, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create complete event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append complete event: %w", err)
	}
	return nil
}

// Reopen records a work.task.reopened event, returning a completed task to the
// open state — the review→fix return edge (run findings v12-F1). The event
// references every live completion for the task, so the supersede fold is pure
// set algebra over explicit CompletionRefs: no event-order comparison anywhere
// (ByType page order differs across store backends), duplicate reopens are
// structurally idempotent, and a re-completion is live again by construction.
//
// Fail-closed: an unreadable completion state refuses (it cannot be proven the
// task is completed), and a task with no live completion refuses (only
// completed work can be reopened — reopening open work would let a reopen
// masquerade as a no-op and desync callers that emit feedback alongside it).
// Reason is required: a reopen exists to carry actionable feedback to the
// producer's next Operate instruction.
func (ts *TaskStore) Reopen(
	source types.ActorID,
	taskID types.EventID,
	reason string,
	issues []string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reopen reason is required")
	}
	live, err := ts.liveCompletionsByTask()
	if err != nil {
		return fmt.Errorf("reopen %s: %w", taskID.Value(), err)
	}
	refs := live[taskID]
	if len(refs) == 0 {
		return fmt.Errorf("reopen %s: no live completion — only a completed task can be reopened", taskID.Value())
	}
	content := TaskReopenedContent{
		TaskID:         taskID,
		ReopenedBy:     source,
		Reason:         reason,
		Issues:         issues,
		CompletionRefs: refs,
	}
	ev, err := ts.factory.Create(EventTypeTaskReopened, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create reopen event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append reopen event: %w", err)
	}
	return nil
}

// TransitionTask records a v3.9 lifecycle transition after validating it
// against the current replayed Work projection.
func (ts *TaskStore) TransitionTask(
	source types.ActorID,
	taskID types.EventID,
	to TaskStatus,
	reason string,
	evidenceRefs []string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	return ts.transitionTask(source, taskID, "", to, reason, evidenceRefs, "", causes, convID)
}

// RejectTask records a terminal rejected lifecycle state.
func (ts *TaskStore) RejectTask(
	source types.ActorID,
	taskID types.EventID,
	reason string,
	evidenceRefs []string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("rejection reason is required")
	}
	return ts.transitionTask(source, taskID, "", StatusRejected, reason, evidenceRefs, "", causes, convID)
}

// SupersedeTask records a terminal superseded lifecycle state and the canonical replacement.
func (ts *TaskStore) SupersedeTask(
	source types.ActorID,
	taskID types.EventID,
	supersededBy string,
	reason string,
	evidenceRefs []string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if strings.TrimSpace(supersededBy) == "" {
		return fmt.Errorf("superseded_by is required")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("supersession reason is required")
	}
	return ts.transitionTask(source, taskID, "", StatusSuperseded, reason, evidenceRefs, supersededBy, causes, convID)
}

func (ts *TaskStore) transitionTask(
	source types.ActorID,
	taskID types.EventID,
	from TaskStatus,
	to TaskStatus,
	reason string,
	evidenceRefs []string,
	supersededBy string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if !isKnownTaskStatus(to) {
		return fmt.Errorf("%w: unknown target state %q", ErrInvalidLifecycleTransition, to)
	}
	current, err := ts.GetStatus(taskID)
	if err != nil {
		return err
	}
	if from != "" && from != current {
		return fmt.Errorf("%w: current state %q does not match requested from_state %q", ErrInvalidLifecycleTransition, current, from)
	}
	if !canTransitionTask(current, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidLifecycleTransition, current, to)
	}
	content := TaskLifecycleTransitionContent{
		TaskID:       taskID,
		FromState:    current,
		ToState:      to,
		Reason:       strings.TrimSpace(reason),
		EvidenceRefs: cloneStrings(evidenceRefs),
		SupersededBy: strings.TrimSpace(supersededBy),
		ChangedBy:    source,
	}
	ev, err := ts.factory.Create(EventTypeTaskLifecycleTransitioned, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create lifecycle transition event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append lifecycle transition event: %w", err)
	}
	return nil
}

// LinkTask attaches FactoryOrder, Requirement, AcceptanceCriterion, and
// canonical Task record references to an existing Work task.
func (ts *TaskStore) LinkTask(
	source types.ActorID,
	taskID types.EventID,
	linkage TaskLinkage,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if err := validateTaskLinkage(linkage); err != nil {
		return err
	}
	content := TaskLinkedContent{
		TaskID:                 taskID,
		CanonicalTaskID:        strings.TrimSpace(linkage.CanonicalTaskID),
		FactoryOrderID:         strings.TrimSpace(linkage.FactoryOrderID),
		RequirementIDs:         cloneStrings(linkage.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(linkage.AcceptanceCriterionIDs),
		LinkedBy:               source,
	}
	ev, err := ts.factory.Create(EventTypeTaskLinked, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create task link event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append task link event: %w", err)
	}
	return nil
}

// AttachVerificationEvidence attaches TestCase, TestRun, GateResult, and Waiver refs.
func (ts *TaskStore) AttachVerificationEvidence(
	source types.ActorID,
	taskID types.EventID,
	evidence VerificationEvidence,
	summary string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if len(evidence.TestCaseIDs) == 0 && len(evidence.TestRunIDs) == 0 && len(evidence.GateResultIDs) == 0 && len(evidence.WaiverIDs) == 0 {
		return fmt.Errorf("at least one verification evidence reference is required")
	}
	if err := validateVerificationEvidence(evidence); err != nil {
		return err
	}
	content := TaskVerificationAttachedContent{
		TaskID:        taskID,
		TestCaseIDs:   cloneStrings(evidence.TestCaseIDs),
		TestRunIDs:    cloneStrings(evidence.TestRunIDs),
		GateResultIDs: cloneStrings(evidence.GateResultIDs),
		WaiverIDs:     cloneStrings(evidence.WaiverIDs),
		Summary:       strings.TrimSpace(summary),
		AttachedBy:    source,
	}
	ev, err := ts.factory.Create(EventTypeTaskVerificationAttached, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create verification evidence event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append verification evidence event: %w", err)
	}
	return nil
}

// AttachFailureRepairReferences attaches Failure, RepairAttempt, and Waiver refs.
func (ts *TaskStore) AttachFailureRepairReferences(
	source types.ActorID,
	taskID types.EventID,
	refs FailureRepairReferences,
	summary string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if len(refs.FailureIDs) == 0 && len(refs.RepairAttemptIDs) == 0 && len(refs.WaiverIDs) == 0 {
		return fmt.Errorf("at least one failure, repair, or waiver reference is required")
	}
	if err := validateFailureRepairReferences(refs); err != nil {
		return err
	}
	content := TaskFailureRepairAttachedContent{
		TaskID:           taskID,
		FailureIDs:       cloneStrings(refs.FailureIDs),
		RepairAttemptIDs: cloneStrings(refs.RepairAttemptIDs),
		WaiverIDs:        cloneStrings(refs.WaiverIDs),
		Summary:          strings.TrimSpace(summary),
		AttachedBy:       source,
	}
	ev, err := ts.factory.Create(EventTypeTaskFailureRepairAttached, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create failure repair event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append failure repair event: %w", err)
	}
	return nil
}

// GetStatus reconstructs the current canonical v3.9 status of a task by
// scanning explicit lifecycle events. Legacy Work events are exposed through
// ProjectLegacyTask and GetCompatibilityStatus instead of being promoted into
// canonical v3.9 lifecycle state.
func (ts *TaskStore) GetStatus(taskID types.EventID) (TaskStatus, error) {
	after := types.None[types.Cursor]()
	for {
		transitionPage, err := ts.store.ByType(EventTypeTaskLifecycleTransitioned, 1000, after)
		if err != nil {
			return StatusCreated, fmt.Errorf("fetch lifecycle transition events: %w", err)
		}
		for _, ev := range transitionPage.Items() {
			c, ok := ev.Content().(TaskLifecycleTransitionContent)
			if ok && c.TaskID == taskID {
				return c.ToState, nil
			}
		}
		if !transitionPage.HasMore() {
			return StatusCreated, nil
		}
		after = transitionPage.Cursor()
	}
}

// GetCompatibilityStatus returns the legacy Work task status projection for
// callers that still depend on pending/assigned/completed operational flow.
func (ts *TaskStore) GetCompatibilityStatus(taskID types.EventID) (LegacyTaskStatus, error) {
	projection, err := ts.ProjectLegacyTask(taskID)
	if err != nil {
		return "", err
	}
	return projection.Status, nil
}

// ProjectLegacyTask reconstructs pre-v3.9 Work task state without changing the
// canonical v3.9 lifecycle. It keeps old created/assigned/completed events
// replayable as an operational compatibility view.
func (ts *TaskStore) ProjectLegacyTask(taskID types.EventID) (LegacyTaskProjection, error) {
	assignee, err := ts.projectAssignee(taskID)
	if err != nil {
		return LegacyTaskProjection{}, err
	}
	readiness, err := ts.Readiness(taskID)
	if err != nil {
		return LegacyTaskProjection{}, err
	}
	completedIDs, err := ts.liveCompletedIDs()
	if err != nil {
		return LegacyTaskProjection{}, err
	}
	if completedIDs[taskID] {
		return LegacyTaskProjection{
			TaskID:   taskID,
			Status:   LegacyStatusCompleted,
			Assignee: assignee,
			Ready:    readiness.Ready,
		}, nil
	}

	blocked, err := ts.IsBlocked(taskID)
	if err != nil {
		return LegacyTaskProjection{}, err
	}
	if blocked {
		return LegacyTaskProjection{
			TaskID:   taskID,
			Status:   LegacyStatusBlocked,
			Assignee: assignee,
			Blocked:  true,
			Ready:    readiness.Ready,
		}, nil
	}

	if !assignee.IsZero() {
		return LegacyTaskProjection{
			TaskID:   taskID,
			Status:   LegacyStatusAssigned,
			Assignee: assignee,
			Ready:    readiness.Ready,
		}, nil
	}

	if readiness.Ready {
		return LegacyTaskProjection{
			TaskID: taskID,
			Status: LegacyStatusReady,
			Ready:  true,
		}, nil
	}

	return LegacyTaskProjection{
		TaskID: taskID,
		Status: LegacyStatusPending,
	}, nil
}

// AddDependency records a work.task.dependency.added event, declaring that taskID
// depends on dependsOnID — taskID is blocked until dependsOnID completes.
func (ts *TaskStore) AddDependency(
	source types.ActorID,
	taskID, dependsOnID types.EventID,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if taskID == dependsOnID {
		return fmt.Errorf("task %s cannot depend on itself", taskID.Value())
	}
	content := TaskDependencyContent{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
		AddedBy:     source,
	}
	ev, err := ts.factory.Create(EventTypeTaskDependencyAdded, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create dependency event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append dependency event: %w", err)
	}
	return nil
}

// GetDependencies returns all task IDs that the given taskID depends on.
// It folds EVERY dependency event, paging until exhaustion: this read is
// load-bearing for the reverse-edge deadlock guard (run findings v11-F1,
// hive#153), and a bounded read under a safety guard is a fail-open — an edge
// older than the newest page would become invisible to the guard.
func (ts *TaskStore) GetDependencies(taskID types.EventID) ([]types.EventID, error) {
	var deps []types.EventID
	after := types.None[types.Cursor]()
	for {
		page, err := ts.store.ByType(EventTypeTaskDependencyAdded, 1000, after)
		if err != nil {
			return nil, fmt.Errorf("fetch dependency events: %w", err)
		}
		for _, ev := range page.Items() {
			c, ok := ev.Content().(TaskDependencyContent)
			if ok && c.TaskID == taskID {
				deps = append(deps, c.DependsOnID)
			}
		}
		if !page.HasMore() {
			return deps, nil
		}
		after = page.Cursor()
	}
}

// DirectChildren returns tasks that directly depend on parentID, sorted by the
// dependency event timestamp from oldest to newest. The oldest child is treated
// as canonical when duplicate child titles are discovered.
func (ts *TaskStore) DirectChildren(parentID types.EventID) ([]ChildTask, error) {
	page, err := ts.store.ByType(EventTypeTaskDependencyAdded, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch dependency events: %w", err)
	}
	children := make([]ChildTask, 0)
	seenChildren := make(map[types.EventID]bool)
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskDependencyContent)
		if !ok || c.DependsOnID != parentID {
			continue
		}
		if seenChildren[c.TaskID] {
			continue
		}
		seenChildren[c.TaskID] = true
		created, err := ts.store.Get(c.TaskID)
		if err != nil {
			continue
		}
		cc, ok := created.Content().(TaskCreatedContent)
		if !ok {
			continue
		}
		p := cc.Priority
		if p == "" {
			p = DefaultPriority
		}
		children = append(children, ChildTask{
			Task: Task{
				ID:                     created.ID(),
				Title:                  cc.Title,
				Description:            cc.Description,
				CreatedBy:              cc.CreatedBy,
				Priority:               p,
				Workspace:              cc.Workspace,
				CanonicalTaskID:        cc.CanonicalTaskID,
				FactoryOrderID:         cc.FactoryOrderID,
				RequirementIDs:         cloneStrings(cc.RequirementIDs),
				AcceptanceCriterionIDs: cloneStrings(cc.AcceptanceCriterionIDs),
				Cell:                   cc.Cell,
				RiskClass:              cc.RiskClass,
				ExpectedOutputs:        cloneStrings(cc.ExpectedOutputs),
			},
			ParentID:            parentID,
			DependencyEventID:   ev.ID(),
			DependencyAddedBy:   c.AddedBy,
			DependencyTimestamp: ev.Timestamp().Value(),
		})
	}
	sort.SliceStable(children, func(i, j int) bool {
		return children[i].DependencyTimestamp.Before(children[j].DependencyTimestamp)
	})
	return children, nil
}

// HasChildren reports whether taskID has at least one direct child/subtask.
func (ts *TaskStore) HasChildren(taskID types.EventID) (bool, error) {
	children, err := ts.DirectChildren(taskID)
	if err != nil {
		return false, err
	}
	return len(children) > 0, nil
}

// SupersedeDuplicateDirectChildren closes duplicate direct children of parentID.
// Duplicates are detected by normalized title under the same parent. The oldest
// child for each normalized title remains canonical; later open duplicates get
// an audit comment, an artifact waiver, and a completion event pointing at the
// canonical task. This preserves referential integrity instead of deleting or
// rewriting the duplicated chain.
func (ts *TaskStore) SupersedeDuplicateDirectChildren(
	parentID types.EventID,
	source types.ActorID,
	causes []types.EventID,
	convID types.ConversationID,
) ([]SupersededTask, error) {
	children, err := ts.DirectChildren(parentID)
	if err != nil {
		return nil, err
	}
	canonicalByTitle := make(map[string]ChildTask)
	superseded := make([]SupersededTask, 0)
	for _, child := range children {
		key := normalizeTaskTitle(child.Title)
		if key == "" {
			continue
		}
		canonical, exists := canonicalByTitle[key]
		if !exists {
			canonicalByTitle[key] = child
			continue
		}
		if canonical.ID == child.ID {
			continue
		}
		status, err := ts.GetCompatibilityStatus(child.ID)
		if err != nil {
			return superseded, fmt.Errorf("get duplicate status: %w", err)
		}
		if status == LegacyStatusCompleted {
			continue
		}
		canonicalStatus, err := ts.GetStatus(child.ID)
		if err != nil {
			return superseded, fmt.Errorf("get duplicate canonical status: %w", err)
		}
		if isTerminalTaskStatus(canonicalStatus) {
			continue
		}
		body := fmt.Sprintf("Superseded duplicate child task. Canonical task: %s (%s). Parent task: %s.", canonical.ID.Value(), canonical.Title, parentID.Value())
		if err := ts.AddComment(child.ID, body, source, causes, convID); err != nil {
			return superseded, fmt.Errorf("comment duplicate child: %w", err)
		}
		if err := ts.WaiveArtifact(source, child.ID, "Superseded by canonical child task "+canonical.ID.Value(), causes, convID); err != nil {
			return superseded, fmt.Errorf("waive duplicate child artifact: %w", err)
		}
		if err := ts.Complete(source, child.ID, body, causes, convID); err != nil {
			return superseded, fmt.Errorf("complete duplicate child: %w", err)
		}
		superseded = append(superseded, SupersededTask{
			TaskID:      child.ID,
			TaskTitle:   child.Title,
			CanonicalID: canonical.ID,
		})
	}
	return superseded, nil
}

func normalizeTaskTitle(title string) string {
	title = strings.ToLower(title)
	var b strings.Builder
	lastSpace := true
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case !lastSpace:
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// IsBlocked returns true if taskID has any declared dependency that is not yet completed
// and has not been explicitly unblocked via a work.task.unblocked event.
func (ts *TaskStore) IsBlocked(taskID types.EventID) (bool, error) {
	deps, err := ts.GetDependencies(taskID)
	if err != nil {
		return false, err
	}
	if len(deps) == 0 {
		return false, nil
	}

	// A work.task.unblocked event explicitly clears the blocked state.
	unblockedPage, err := ts.store.ByType(EventTypeTaskUnblocked, 1000, types.None[types.Cursor]())
	if err != nil {
		return false, fmt.Errorf("fetch unblocked events: %w", err)
	}
	for _, ev := range unblockedPage.Items() {
		if c, ok := ev.Content().(TaskUnblockedContent); ok && c.TaskID == taskID {
			return false, nil
		}
	}

	// Collect all dependency-satisfying task IDs once (reopen-aware legacy
	// completions plus v3.9 certified tasks).
	completedIDs, err := ts.dependencySatisfiedIDs()
	if err != nil {
		return false, err
	}

	for _, depID := range deps {
		if !completedIDs[depID] {
			return true, nil
		}
	}
	return false, nil
}

// UnblockTask records a work.task.unblocked event, explicitly marking the task's
// blockers as resolved. After this event, IsBlocked returns false for the task
// regardless of its dependency state.
func (ts *TaskStore) UnblockTask(
	source types.ActorID,
	taskID types.EventID,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	content := TaskUnblockedContent{
		TaskID:      taskID,
		UnblockedBy: source,
	}
	ev, err := ts.factory.Create(EventTypeTaskUnblocked, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create unblock event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append unblock event: %w", err)
	}
	return nil
}

// liveCompletionsByTask returns, per task, the completion event IDs whose
// effect stands: a work.task.completed event is live unless a
// work.task.reopened event names its EventID in CompletionRefs (the review→fix
// return edge, run findings v12-F1). Pure set algebra over explicit references
// — no event-order comparison anywhere, so the fold is identical across store
// backends regardless of ByType page order. A task with at least one live
// completion reads as completed. Page budget mirrors the file's existing
// completion scans (the ByType pagination class is routed G-2.x).
func (ts *TaskStore) liveCompletionsByTask() (map[types.EventID][]types.EventID, error) {
	completedPage, err := ts.store.ByType(EventTypeTaskCompleted, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch completed events: %w", err)
	}
	reopenedPage, err := ts.store.ByType(EventTypeTaskReopened, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch reopened events: %w", err)
	}
	superseded := make(map[types.EventID]bool)
	for _, ev := range reopenedPage.Items() {
		if c, ok := ev.Content().(TaskReopenedContent); ok {
			for _, ref := range c.CompletionRefs {
				superseded[ref] = true
			}
		}
	}
	live := make(map[types.EventID][]types.EventID)
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(TaskCompletedContent)
		if !ok || superseded[ev.ID()] {
			continue
		}
		live[c.TaskID] = append(live[c.TaskID], ev.ID())
	}
	return live, nil
}

// liveCompletedIDs returns the set of task IDs with at least one live
// completion. This is the single completion-state fold shared by every
// projection (ListOpen, IsBlocked, ProjectLegacyTask, batchStatus) so reopen
// semantics cannot drift between them.
func (ts *TaskStore) liveCompletedIDs() (map[types.EventID]bool, error) {
	live, err := ts.liveCompletionsByTask()
	if err != nil {
		return nil, err
	}
	ids := make(map[types.EventID]bool, len(live))
	for taskID := range live {
		ids[taskID] = true
	}
	return ids, nil
}

// dependencySatisfiedIDs returns task IDs that satisfy downstream dependency
// edges. Legacy Work tasks satisfy dependencies through a live completion.
// Issue-scan stage tasks also satisfy their canonical stage DAG dependencies
// once certified; other v3.9 certified tasks keep the pre-existing
// completion-only dependency semantics.
func (ts *TaskStore) dependencySatisfiedIDs() (map[types.EventID]bool, error) {
	ids, err := ts.liveCompletedIDs()
	if err != nil {
		return nil, err
	}
	statuses, err := ts.latestLifecycleStatuses()
	if err != nil {
		return nil, err
	}
	issueScanIDs, err := ts.issueScanTaskIDs()
	if err != nil {
		return nil, err
	}
	for taskID, status := range statuses {
		if status == StatusCertified && issueScanIDs[taskID] {
			ids[taskID] = true
		}
	}
	return ids, nil
}

func (ts *TaskStore) issueScanTaskIDs() (map[types.EventID]bool, error) {
	ids := make(map[types.EventID]bool)
	after := types.None[types.Cursor]()
	for {
		page, err := ts.store.ByType(EventTypeTaskCreated, 1000, after)
		if err != nil {
			return nil, fmt.Errorf("fetch issue-scan task ids: %w", err)
		}
		for _, ev := range page.Items() {
			c, ok := ev.Content().(TaskCreatedContent)
			if ok && strings.TrimSpace(c.Workspace) == IssueScanWorkspace {
				ids[ev.ID()] = true
			}
		}
		if !page.HasMore() {
			return ids, nil
		}
		after = page.Cursor()
	}
}

func (ts *TaskStore) latestLifecycleStatuses() (map[types.EventID]TaskStatus, error) {
	statuses := make(map[types.EventID]TaskStatus)
	after := types.None[types.Cursor]()
	for {
		page, err := ts.store.ByType(EventTypeTaskLifecycleTransitioned, 1000, after)
		if err != nil {
			return nil, fmt.Errorf("fetch lifecycle transition events: %w", err)
		}
		for _, ev := range page.Items() {
			c, ok := ev.Content().(TaskLifecycleTransitionContent)
			if !ok {
				continue
			}
			if _, seen := statuses[c.TaskID]; seen {
				continue
			}
			statuses[c.TaskID] = c.ToState
		}
		if !page.HasMore() {
			return statuses, nil
		}
		after = page.Cursor()
	}
}

// ListOpen returns all tasks that do not have a matching work.task.completed event
// and are not blocked by an incomplete dependency.
// It fetches up to 1000 tasks and filters out completed and blocked tasks.
func (ts *TaskStore) ListOpen() ([]Task, error) {
	// Collect all dependency-satisfying task IDs. A legacy completion
	// superseded by a reopen does not count; certified tasks count only for
	// the issue-scan stage workspace.
	completedIDs, err := ts.dependencySatisfiedIDs()
	if err != nil {
		return nil, err
	}

	// Collect all dependency edges: taskID → dependsOnID.
	depPage, err := ts.store.ByType(EventTypeTaskDependencyAdded, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch dependency events: %w", err)
	}
	// blockedBy maps a taskID to the set of its uncompleted dependency IDs.
	blockedBy := make(map[types.EventID][]types.EventID)
	for _, ev := range depPage.Items() {
		c, ok := ev.Content().(TaskDependencyContent)
		if !ok {
			continue
		}
		if !completedIDs[c.DependsOnID] {
			blockedBy[c.TaskID] = append(blockedBy[c.TaskID], c.DependsOnID)
		}
	}

	// Collect explicitly unblocked tasks — these override the blocked state.
	unblockedPage, err := ts.store.ByType(EventTypeTaskUnblocked, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch unblocked events: %w", err)
	}
	unblockedIDs := make(map[types.EventID]bool)
	for _, ev := range unblockedPage.Items() {
		if c, ok := ev.Content().(TaskUnblockedContent); ok {
			unblockedIDs[c.TaskID] = true
		}
	}

	// List all tasks and filter out completed and blocked ones.
	all, err := ts.List(1000)
	if err != nil {
		return nil, err
	}
	open := make([]Task, 0, len(all))
	for _, t := range all {
		isBlocked := len(blockedBy[t.ID]) > 0 && !unblockedIDs[t.ID]
		if !completedIDs[t.ID] && !isBlocked {
			open = append(open, t)
		}
	}
	return open, nil
}

// GetByAssignee returns tasks assigned to the given actor.
// It scans work.task.assigned events and joins each to its work.task.created event.
func (ts *TaskStore) GetByAssignee(assignee types.ActorID) ([]Task, error) {
	// Fetch all assigned events; no SQL join available in-memory so filter in code.
	page, err := ts.store.ByType(EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch assigned events: %w", err)
	}
	var tasks []Task
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskAssignedContent)
		if !ok || c.AssignedTo != assignee {
			continue
		}
		created, err := ts.store.Get(c.TaskID)
		if err != nil {
			continue // task event missing — skip
		}
		cc, ok := created.Content().(TaskCreatedContent)
		if !ok {
			continue
		}
		cp := cc.Priority
		if cp == "" {
			cp = DefaultPriority
		}
		tasks = append(tasks, Task{
			ID:                     created.ID(),
			Title:                  cc.Title,
			Description:            cc.Description,
			CreatedBy:              cc.CreatedBy,
			Priority:               cp,
			Workspace:              cc.Workspace,
			CanonicalTaskID:        cc.CanonicalTaskID,
			FactoryOrderID:         cc.FactoryOrderID,
			RequirementIDs:         cloneStrings(cc.RequirementIDs),
			AcceptanceCriterionIDs: cloneStrings(cc.AcceptanceCriterionIDs),
			Cell:                   cc.Cell,
			RiskClass:              cc.RiskClass,
			ExpectedOutputs:        cloneStrings(cc.ExpectedOutputs),
		})
	}
	return tasks, nil
}

// SetPriority records a work.task.priority.set event, updating the priority of taskID.
func (ts *TaskStore) SetPriority(
	source types.ActorID,
	taskID types.EventID,
	priority TaskPriority,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	content := TaskPrioritySetContent{
		TaskID:   taskID,
		Priority: priority,
		SetBy:    source,
	}
	ev, err := ts.factory.Create(EventTypeTaskPrioritySet, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create priority event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append priority event: %w", err)
	}
	return nil
}

// GetPriority returns the effective priority of a task by scanning
// work.task.priority.set events for the most recent override, falling back
// to the priority recorded in the work.task.created event.
func (ts *TaskStore) GetPriority(taskID types.EventID) (TaskPriority, error) {
	// Scan all priority-set events for this task; events are returned newest-first,
	// so the first match is the most recent override.
	page, err := ts.store.ByType(EventTypeTaskPrioritySet, 1000, types.None[types.Cursor]())
	if err != nil {
		return DefaultPriority, fmt.Errorf("fetch priority events: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskPrioritySetContent)
		if ok && c.TaskID == taskID {
			return c.Priority, nil
		}
	}

	// Fall back to the creation event's priority.
	ev, err := ts.store.Get(taskID)
	if err != nil {
		return DefaultPriority, fmt.Errorf("get task event: %w", err)
	}
	c, ok := ev.Content().(TaskCreatedContent)
	if !ok {
		return DefaultPriority, nil
	}
	if c.Priority == "" {
		return DefaultPriority, nil
	}
	return c.Priority, nil
}

// batchStatus enriches a slice of Tasks with computed Status, Assignee, and Blocked
// fields using three batch store scans rather than N per-task queries.
func (ts *TaskStore) batchStatus(tasks []Task) ([]TaskSummary, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	// Scan 1: completed + reopened events plus issue-scan certification -> dependency-satisfied set.
	completedIDs, err := ts.dependencySatisfiedIDs()
	if err != nil {
		return nil, err
	}

	// Scan 2: assigned events (newest-first) → current assignee per task.
	assignedPage, err := ts.store.ByType(EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch assigned events: %w", err)
	}
	assigneeMap := make(map[types.EventID]types.ActorID, len(assignedPage.Items()))
	for _, ev := range assignedPage.Items() {
		if c, ok := ev.Content().(TaskAssignedContent); ok {
			if _, seen := assigneeMap[c.TaskID]; !seen {
				assigneeMap[c.TaskID] = c.AssignedTo
			}
		}
	}

	// Scan 3: dependency events → blocked set (reuses completedIDs from scan 1).
	depPage, err := ts.store.ByType(EventTypeTaskDependencyAdded, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch dependency events: %w", err)
	}
	blockedMap := make(map[types.EventID]bool)
	for _, ev := range depPage.Items() {
		if c, ok := ev.Content().(TaskDependencyContent); ok {
			if !completedIDs[c.DependsOnID] {
				blockedMap[c.TaskID] = true
			}
		}
	}

	// Scan 4: unblocked events → explicitly unblocked set (clears blocked state).
	unblockedPage, err := ts.store.ByType(EventTypeTaskUnblocked, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch unblocked events: %w", err)
	}
	unblockedMap := make(map[types.EventID]bool)
	for _, ev := range unblockedPage.Items() {
		if c, ok := ev.Content().(TaskUnblockedContent); ok {
			unblockedMap[c.TaskID] = true
		}
	}

	// Scan 5: artifact events → count and readiness labels per task.
	artifactPage, err := ts.store.ByType(EventTypeTaskArtifact, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch artifact events: %w", err)
	}
	artifactCount := make(map[types.EventID]int)
	gatesByTask := make(map[types.EventID]map[string]bool)
	for _, ev := range artifactPage.Items() {
		if c, ok := ev.Content().(TaskArtifactContent); ok {
			artifactCount[c.TaskID]++
			label := normalizeGateLabel(c.Label)
			if isRequiredGateLabel(label) {
				if gatesByTask[c.TaskID] == nil {
					gatesByTask[c.TaskID] = make(map[string]bool)
				}
				gatesByTask[c.TaskID][label] = true
			}
		}
	}

	// Scan 6: waiver events → waived set.
	waiverPage, err := ts.store.ByType(EventTypeTaskArtifactWaived, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch waiver events: %w", err)
	}
	waivedMap := make(map[types.EventID]bool)
	for _, ev := range waiverPage.Items() {
		if c, ok := ev.Content().(TaskArtifactWaivedContent); ok {
			waivedMap[c.TaskID] = true
		}
	}

	summaries := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		status, err := ts.GetStatus(t.ID)
		if err != nil {
			return nil, err
		}
		legacyStatus, err := ts.GetCompatibilityStatus(t.ID)
		if err != nil {
			return nil, err
		}
		missing := missingRequiredGates(gatesByTask[t.ID])
		_, missingFacts, err := ts.factReadiness(t.ID)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, TaskSummary{
			Task:          t,
			Status:        status,
			LegacyStatus:  legacyStatus,
			Assignee:      assigneeMap[t.ID],
			Blocked:       blockedMap[t.ID] && !unblockedMap[t.ID],
			ArtifactCount: artifactCount[t.ID],
			Waived:        waivedMap[t.ID],
			Ready:         len(missing) == 0 && len(missingFacts) == 0,
			MissingGates:  missing,
			MissingFacts:  missingFacts,
		})
	}
	return summaries, nil
}

// ListSummaries returns up to limit tasks with Status, Assignee, and Blocked
// populated via three batch store scans (rather than N per-task queries).
func (ts *TaskStore) ListSummaries(limit int) ([]TaskSummary, error) {
	tasks, err := ts.List(limit)
	if err != nil {
		return nil, err
	}
	return ts.batchStatus(tasks)
}

// ProjectTask rebuilds the v3.9 Work projection for a task from append-only events.
func (ts *TaskStore) ProjectTask(taskID types.EventID) (TaskProjection, error) {
	ev, err := ts.store.Get(taskID)
	if err != nil {
		return TaskProjection{}, fmt.Errorf("get task event: %w", err)
	}
	created, ok := ev.Content().(TaskCreatedContent)
	if !ok {
		return TaskProjection{}, fmt.Errorf("event %s is not a work task", taskID.Value())
	}
	p := created.Priority
	if p == "" {
		p = DefaultPriority
	}
	task := Task{
		ID:                     ev.ID(),
		Title:                  created.Title,
		Description:            created.Description,
		CreatedBy:              created.CreatedBy,
		Priority:               p,
		Workspace:              created.Workspace,
		CanonicalTaskID:        created.CanonicalTaskID,
		FactoryOrderID:         created.FactoryOrderID,
		RequirementIDs:         cloneStrings(created.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(created.AcceptanceCriterionIDs),
		Cell:                   created.Cell,
		RiskClass:              created.RiskClass,
		ExpectedOutputs:        cloneStrings(created.ExpectedOutputs),
	}
	status, err := ts.GetStatus(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	readiness, err := ts.Readiness(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	blocked, err := ts.IsBlocked(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	linkage, err := ts.projectLinkage(taskID, task)
	if err != nil {
		return TaskProjection{}, err
	}
	modelOverrides, err := ts.projectFactoryOrderModelOverrides(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	verification, err := ts.projectVerification(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	failureRepair, err := ts.projectFailureRepair(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	assignee, err := ts.projectAssignee(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	lastTransition, supersededBy, err := ts.projectLatestTransition(taskID)
	if err != nil {
		return TaskProjection{}, err
	}
	return TaskProjection{
		Task:                task,
		Status:              status,
		Assignee:            assignee,
		Blocked:             blocked || status == StatusBlocked || status == StatusPolicyBlocked,
		Ready:               status == StatusReady || (readiness.Ready && status == StatusRepaired),
		Linkage:             linkage,
		ModelOverrides:      modelOverrides,
		Verification:        verification,
		FailureRepair:       failureRepair,
		SupersededBy:        supersededBy,
		LastTransitionEvent: lastTransition,
	}, nil
}

// AddComment records a work.task.comment event on the graph, attaching a
// freeform note authored by author to the given task.
func (ts *TaskStore) AddComment(
	taskID types.EventID,
	body string,
	author types.ActorID,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if body == "" {
		return fmt.Errorf("body is required")
	}
	content := CommentContent{
		TaskID:   taskID,
		Body:     body,
		AuthorID: author,
	}
	ev, err := ts.factory.Create(EventTypeTaskComment, author, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create comment event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append comment event: %w", err)
	}
	return nil
}

// ListComments returns all comments for the given task in chronological order.
func (ts *TaskStore) ListComments(taskID types.EventID) ([]CommentEvent, error) {
	page, err := ts.store.ByType(EventTypeTaskComment, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch comment events: %w", err)
	}
	var comments []CommentEvent
	for _, ev := range page.Items() {
		c, ok := ev.Content().(CommentContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		comments = append(comments, CommentEvent{
			ID:        ev.ID(),
			TaskID:    c.TaskID,
			Body:      c.Body,
			AuthorID:  c.AuthorID,
			Timestamp: ev.Timestamp().Value(),
		})
	}
	return comments, nil
}

// AddArtifact records a work.task.artifact event on the graph.
func (ts *TaskStore) AddArtifact(
	source types.ActorID,
	taskID types.EventID,
	label, mediaType, body string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if label == "" {
		return fmt.Errorf("label is required")
	}
	if label == IncidentFollowUpArtifactLabel {
		if mediaType == "" {
			mediaType = IncidentFollowUpMediaType
		}
		if mediaType != IncidentFollowUpMediaType {
			return fmt.Errorf("%s artifacts must use media type %s", IncidentFollowUpArtifactLabel, IncidentFollowUpMediaType)
		}
		if _, err := parseIncidentFollowUpArtifactBody(body); err != nil {
			return fmt.Errorf("%s artifact is invalid: %w", IncidentFollowUpArtifactLabel, err)
		}
	}
	if mediaType == "" {
		mediaType = "text/markdown"
	}
	content := TaskArtifactContent{
		TaskID:    taskID,
		Label:     label,
		MediaType: mediaType,
		Body:      body,
		CreatedBy: source,
	}
	ev, err := ts.factory.Create(EventTypeTaskArtifact, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create artifact event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append artifact event: %w", err)
	}
	return nil
}

// WaiveArtifact records a work.task.artifact.waived event on the graph.
func (ts *TaskStore) WaiveArtifact(
	source types.ActorID,
	taskID types.EventID,
	reason string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	content := TaskArtifactWaivedContent{
		TaskID:   taskID,
		Reason:   reason,
		WaivedBy: source,
	}
	ev, err := ts.factory.Create(EventTypeTaskArtifactWaived, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create waiver event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append waiver event: %w", err)
	}
	return nil
}

// AddFactRequirement records that a task is not ready until a Phase 3
// lifecycle, authority, key, trust, decision, or audit fact exists.
func (ts *TaskStore) AddFactRequirement(
	source types.ActorID,
	taskID types.EventID,
	requiredEventType types.EventType,
	requiredEventID types.EventID,
	reason string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	if requiredEventType.Value() == "" {
		return fmt.Errorf("required event type is required")
	}
	content := TaskFactRequiredContent{
		TaskID:            taskID,
		RequiredEventType: requiredEventType,
		RequiredEventID:   requiredEventID,
		Reason:            strings.TrimSpace(reason),
		RequiredBy:        source,
	}
	ev, err := ts.factory.Create(EventTypeTaskFactRequired, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return fmt.Errorf("create fact requirement event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return fmt.Errorf("append fact requirement event: %w", err)
	}
	return nil
}

// ListFactRequirements returns task readiness fact requirements with their
// current satisfaction state derived from the shared EventGraph store.
func (ts *TaskStore) ListFactRequirements(taskID types.EventID) ([]FactRequirement, error) {
	page, err := ts.store.ByType(EventTypeTaskFactRequired, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch fact requirement events: %w", err)
	}
	requirements := make([]FactRequirement, 0)
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskFactRequiredContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		satisfied, err := ts.factRequirementSatisfied(c)
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, FactRequirement{
			ID:                ev.ID(),
			TaskID:            c.TaskID,
			RequiredEventType: c.RequiredEventType,
			RequiredEventID:   c.RequiredEventID,
			Reason:            c.Reason,
			RequiredBy:        c.RequiredBy,
			Timestamp:         ev.Timestamp().Value(),
			Satisfied:         satisfied,
		})
	}
	return requirements, nil
}

// ListArtifacts returns all artifacts for the given task in chronological order.
func (ts *TaskStore) ListArtifacts(taskID types.EventID) ([]ArtifactEvent, error) {
	page, err := ts.store.ByType(EventTypeTaskArtifact, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch artifact events: %w", err)
	}
	var artifacts []ArtifactEvent
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskArtifactContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		artifacts = append(artifacts, ArtifactEvent{
			ID:        ev.ID(),
			TaskID:    c.TaskID,
			Label:     c.Label,
			MediaType: c.MediaType,
			Body:      c.Body,
			CreatedBy: c.CreatedBy,
			Timestamp: ev.Timestamp().Value(),
		})
	}
	return artifacts, nil
}

// ListReopens returns a task's work.task.reopened events in CHRONOLOGICAL
// order (oldest first), carrying the reviewer's reason and fix list. The
// producer's Operate instruction folds these in as numbered rounds, so the
// order is load-bearing: ByType pages newest-first, hence the explicit
// reversal. Bounded in practice by the reviewer's per-task verdict cap
// (run findings v12-F1).
func (ts *TaskStore) ListReopens(taskID types.EventID) ([]ReopenEvent, error) {
	page, err := ts.store.ByType(EventTypeTaskReopened, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch reopen events: %w", err)
	}
	var reopens []ReopenEvent
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskReopenedContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		reopens = append(reopens, ReopenEvent{
			ID:         ev.ID(),
			TaskID:     c.TaskID,
			ReopenedBy: c.ReopenedBy,
			Reason:     c.Reason,
			Issues:     c.Issues,
			Timestamp:  ev.Timestamp().Value(),
		})
	}
	for i, j := 0, len(reopens)-1; i < j; i, j = i+1, j-1 {
		reopens[i], reopens[j] = reopens[j], reopens[i]
	}
	return reopens, nil
}

// Readiness reconstructs whether a task has the required implementation gates
// and any declared Phase 3 fact prerequisites.
func (ts *TaskStore) Readiness(taskID types.EventID) (TaskReadiness, error) {
	page, err := ts.store.ByType(EventTypeTaskArtifact, 1000, types.None[types.Cursor]())
	if err != nil {
		return TaskReadiness{}, fmt.Errorf("fetch artifact events: %w", err)
	}
	present := make(map[string]bool)
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskArtifactContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		label := normalizeGateLabel(c.Label)
		// A required gate counts as present only when its body is non-empty: a
		// label-only (empty) artifact does not satisfy readiness. This is where the
		// non-empty gate contract is enforced (D), regardless of whether the seed
		// or the planner attached the gate.
		if isRequiredGateLabel(label) && strings.TrimSpace(c.Body) != "" {
			present[label] = true
		}
	}
	missing := missingRequiredGates(present)
	presentFacts, missingFacts, err := ts.factReadiness(taskID)
	if err != nil {
		return TaskReadiness{}, err
	}
	out := TaskReadiness{
		TaskID:       taskID,
		Ready:        len(missing) == 0 && len(missingFacts) == 0,
		PresentGates: presentRequiredGates(present),
		MissingGates: missing,
		PresentFacts: presentFacts,
		MissingFacts: missingFacts,
	}
	return out, nil
}

func (ts *TaskStore) factReadiness(taskID types.EventID) ([]string, []string, error) {
	requirements, err := ts.ListFactRequirements(taskID)
	if err != nil {
		return nil, nil, err
	}
	present := make([]string, 0)
	missing := make([]string, 0)
	for _, req := range requirements {
		label := factRequirementLabel(req)
		if req.Satisfied {
			present = append(present, label)
			continue
		}
		missing = append(missing, label)
	}
	return present, missing, nil
}

func (ts *TaskStore) factRequirementSatisfied(req TaskFactRequiredContent) (bool, error) {
	if !req.RequiredEventID.IsZero() {
		ev, err := ts.store.Get(req.RequiredEventID)
		if err != nil {
			return false, nil
		}
		return ev.Type() == req.RequiredEventType, nil
	}
	descendants, err := ts.store.Descendants(req.TaskID, 1000)
	if err != nil {
		return false, fmt.Errorf("fetch task descendants: %w", err)
	}
	for _, ev := range descendants {
		if ev.Type() == req.RequiredEventType {
			return true, nil
		}
	}
	return false, nil
}

func factRequirementLabel(req FactRequirement) string {
	if req.RequiredEventID.IsZero() {
		return req.RequiredEventType.Value()
	}
	return req.RequiredEventType.Value() + "#" + req.RequiredEventID.Value()
}

func normalizeGateLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.ReplaceAll(label, "-", "_")
	label = strings.ReplaceAll(label, " ", "_")
	return label
}

func isRequiredGateLabel(label string) bool {
	for _, required := range RequiredReadinessGateLabels() {
		if label == required {
			return true
		}
	}
	return false
}

func missingRequiredGates(present map[string]bool) []string {
	missing := make([]string, 0)
	for _, required := range RequiredReadinessGateLabels() {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func presentRequiredGates(present map[string]bool) []string {
	out := make([]string, 0)
	for _, required := range RequiredReadinessGateLabels() {
		if present[required] {
			out = append(out, required)
		}
	}
	return out
}

// GetArtifactBody returns the body of a work.task.artifact event by its event ID.
// Returns the body and true if found, or empty string and false if not found or
// the event is not a TaskArtifactContent (e.g., it's a waiver).
func (ts *TaskStore) GetArtifactBody(artifactID types.EventID) (string, bool) {
	ev, err := ts.store.Get(artifactID)
	if err != nil {
		return "", false
	}
	c, ok := ev.Content().(TaskArtifactContent)
	if !ok {
		return "", false
	}
	return c.Body, true
}

func firstPriority(priority []TaskPriority) TaskPriority {
	if len(priority) == 0 {
		return DefaultPriority
	}
	return priority[0]
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func validateTaskCreateOptions(opts TaskCreateOptions) error {
	linkage := TaskLinkage{
		CanonicalTaskID:        opts.CanonicalTaskID,
		FactoryOrderID:         opts.FactoryOrderID,
		RequirementIDs:         opts.RequirementIDs,
		AcceptanceCriterionIDs: opts.AcceptanceCriterionIDs,
	}
	if linkage.CanonicalTaskID == "" && linkage.FactoryOrderID == "" && len(linkage.RequirementIDs) == 0 && len(linkage.AcceptanceCriterionIDs) == 0 {
		return nil
	}
	if err := validateTaskLinkage(linkage); err != nil {
		return err
	}
	if strings.TrimSpace(opts.Cell) == "" {
		return fmt.Errorf("cell is required for v3.9 task linkage")
	}
	if strings.TrimSpace(opts.RiskClass) == "" {
		return fmt.Errorf("risk_class is required for v3.9 task linkage")
	}
	if !isRiskClass(opts.RiskClass) {
		return fmt.Errorf("risk_class must be one of low, medium, high, critical")
	}
	return nil
}

func validateTaskLinkage(linkage TaskLinkage) error {
	if strings.TrimSpace(linkage.CanonicalTaskID) != "" {
		if err := validateV39Reference(v39.TypeTask, "canonical_task_id", strings.TrimSpace(linkage.CanonicalTaskID)); err != nil {
			return err
		}
	}
	if err := validateV39Reference(v39.TypeFactoryOrder, "factory_order_id", strings.TrimSpace(linkage.FactoryOrderID)); err != nil {
		return err
	}
	if len(linkage.RequirementIDs) == 0 {
		return fmt.Errorf("at least one requirement_id is required")
	}
	if len(linkage.AcceptanceCriterionIDs) == 0 {
		return fmt.Errorf("at least one acceptance_criterion_id is required")
	}
	if err := validateV39References(v39.TypeRequirement, "requirement_ids", linkage.RequirementIDs); err != nil {
		return err
	}
	return validateV39References(v39.TypeAcceptanceCriterion, "acceptance_criterion_ids", linkage.AcceptanceCriterionIDs)
}

func validateVerificationEvidence(evidence VerificationEvidence) error {
	if err := validateV39References(v39.TypeTestCase, "test_case_ids", evidence.TestCaseIDs); err != nil {
		return err
	}
	if err := validateV39References(v39.TypeTestRun, "test_run_ids", evidence.TestRunIDs); err != nil {
		return err
	}
	if err := validateV39References(v39.TypeGateResult, "gate_result_ids", evidence.GateResultIDs); err != nil {
		return err
	}
	return validateV39References(v39.TypeWaiver, "waiver_ids", evidence.WaiverIDs)
}

func validateFailureRepairReferences(refs FailureRepairReferences) error {
	if err := validateV39References(v39.TypeFailure, "failure_ids", refs.FailureIDs); err != nil {
		return err
	}
	if err := validateV39References(v39.TypeRepairAttempt, "repair_attempt_ids", refs.RepairAttemptIDs); err != nil {
		return err
	}
	return validateV39References(v39.TypeWaiver, "waiver_ids", refs.WaiverIDs)
}

func validateV39References(recordType, field string, values []string) error {
	for _, value := range values {
		if err := validateV39Reference(recordType, field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateV39Reference(recordType, field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	prefixByType := map[string]string{
		v39.TypeFactoryOrder:        "fo_",
		v39.TypeRequirement:         "req_",
		v39.TypeAcceptanceCriterion: "ac_",
		v39.TypeTask:                "tsk_",
		v39.TypeTestCase:            "tc_",
		v39.TypeTestRun:             "tr_",
		v39.TypeGateResult:          "gate_",
		v39.TypeFailure:             "fail_",
		v39.TypeRepairAttempt:       "rep_",
		v39.TypeWaiver:              "waiver_",
	}
	if prefix := prefixByType[recordType]; prefix != "" && !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("%s %q must reference %s with prefix %q", field, value, recordType, prefix)
	}
	return nil
}

func isRiskClass(value string) bool {
	switch value {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isKnownTaskStatus(status TaskStatus) bool {
	switch status {
	case StatusCreated, StatusReady, StatusRunning, StatusBlocked, StatusFailed, StatusRepairRequired,
		StatusRepairRunning, StatusRepaired, StatusVerificationRunning, StatusVerified, StatusCertified,
		StatusRejected, StatusSuperseded, StatusPolicyBlocked:
		return true
	default:
		return false
	}
}

func isTerminalTaskStatus(status TaskStatus) bool {
	switch status {
	case StatusCertified, StatusRejected, StatusSuperseded:
		return true
	default:
		return false
	}
}

func canTransitionTask(from, to TaskStatus) bool {
	if from == to {
		return false
	}
	if from == StatusCertified || from == StatusRejected || from == StatusSuperseded {
		return false
	}
	if to == StatusSuperseded {
		return isKnownTaskStatus(from)
	}
	switch from {
	case StatusCreated:
		return to == StatusReady
	case StatusReady:
		return to == StatusRunning
	case StatusRunning:
		return to == StatusVerified || to == StatusFailed || to == StatusBlocked || to == StatusPolicyBlocked
	case StatusBlocked:
		return to == StatusReady
	case StatusFailed:
		return to == StatusRepairRequired
	case StatusRepairRequired:
		return to == StatusRepairRunning
	case StatusRepairRunning:
		return to == StatusRepaired
	case StatusRepaired:
		return to == StatusVerificationRunning
	case StatusVerificationRunning:
		return to == StatusVerified || to == StatusRejected
	case StatusVerified:
		return to == StatusCertified || to == StatusRejected
	case StatusPolicyBlocked:
		return false
	default:
		return false
	}
}

func (ts *TaskStore) projectAssignee(taskID types.EventID) (types.ActorID, error) {
	page, err := ts.store.ByType(EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err != nil {
		return types.ActorID{}, fmt.Errorf("fetch assigned events: %w", err)
	}
	for _, ev := range page.Items() {
		if c, ok := ev.Content().(TaskAssignedContent); ok && c.TaskID == taskID {
			return c.AssignedTo, nil
		}
	}
	return types.ActorID{}, nil
}

func (ts *TaskStore) projectLatestTransition(taskID types.EventID) (types.EventID, string, error) {
	page, err := ts.store.ByType(EventTypeTaskLifecycleTransitioned, 1000, types.None[types.Cursor]())
	if err != nil {
		return types.EventID{}, "", fmt.Errorf("fetch lifecycle transition events: %w", err)
	}
	for _, ev := range page.Items() {
		if c, ok := ev.Content().(TaskLifecycleTransitionContent); ok && c.TaskID == taskID {
			return ev.ID(), c.SupersededBy, nil
		}
	}
	return types.EventID{}, "", nil
}

func (ts *TaskStore) projectLinkage(taskID types.EventID, task Task) (TaskLinkage, error) {
	linkage := TaskLinkage{
		CanonicalTaskID:        task.CanonicalTaskID,
		FactoryOrderID:         task.FactoryOrderID,
		RequirementIDs:         cloneStrings(task.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(task.AcceptanceCriterionIDs),
	}
	page, err := ts.store.ByType(EventTypeTaskLinked, 1000, types.None[types.Cursor]())
	if err != nil {
		return TaskLinkage{}, fmt.Errorf("fetch task link events: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskLinkedContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		if c.CanonicalTaskID != "" {
			linkage.CanonicalTaskID = c.CanonicalTaskID
		}
		if c.FactoryOrderID != "" {
			linkage.FactoryOrderID = c.FactoryOrderID
		}
		if len(c.RequirementIDs) > 0 {
			linkage.RequirementIDs = cloneStrings(c.RequirementIDs)
		}
		if len(c.AcceptanceCriterionIDs) > 0 {
			linkage.AcceptanceCriterionIDs = cloneStrings(c.AcceptanceCriterionIDs)
		}
		break
	}
	return linkage, nil
}

func (ts *TaskStore) projectVerification(taskID types.EventID) (VerificationEvidence, error) {
	page, err := ts.store.ByType(EventTypeTaskVerificationAttached, 1000, types.None[types.Cursor]())
	if err != nil {
		return VerificationEvidence{}, fmt.Errorf("fetch verification evidence events: %w", err)
	}
	var out VerificationEvidence
	seenTestCases := map[string]bool{}
	seenTestRuns := map[string]bool{}
	seenGateResults := map[string]bool{}
	seenWaivers := map[string]bool{}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskVerificationAttachedContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		out.TestCaseIDs = appendUniqueStrings(out.TestCaseIDs, c.TestCaseIDs, seenTestCases)
		out.TestRunIDs = appendUniqueStrings(out.TestRunIDs, c.TestRunIDs, seenTestRuns)
		out.GateResultIDs = appendUniqueStrings(out.GateResultIDs, c.GateResultIDs, seenGateResults)
		out.WaiverIDs = appendUniqueStrings(out.WaiverIDs, c.WaiverIDs, seenWaivers)
	}
	return out, nil
}

func (ts *TaskStore) projectFailureRepair(taskID types.EventID) (FailureRepairReferences, error) {
	page, err := ts.store.ByType(EventTypeTaskFailureRepairAttached, 1000, types.None[types.Cursor]())
	if err != nil {
		return FailureRepairReferences{}, fmt.Errorf("fetch failure repair events: %w", err)
	}
	var out FailureRepairReferences
	seenFailures := map[string]bool{}
	seenRepairs := map[string]bool{}
	seenWaivers := map[string]bool{}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskFailureRepairAttachedContent)
		if !ok || c.TaskID != taskID {
			continue
		}
		out.FailureIDs = appendUniqueStrings(out.FailureIDs, c.FailureIDs, seenFailures)
		out.RepairAttemptIDs = appendUniqueStrings(out.RepairAttemptIDs, c.RepairAttemptIDs, seenRepairs)
		out.WaiverIDs = appendUniqueStrings(out.WaiverIDs, c.WaiverIDs, seenWaivers)
	}
	return out, nil
}

func appendUniqueStrings(dst []string, src []string, seen map[string]bool) []string {
	for _, value := range src {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		dst = append(dst, value)
	}
	return dst
}

// HasWaiver returns true if a work.task.artifact.waived event exists for the task.
func (ts *TaskStore) HasWaiver(taskID types.EventID) (bool, error) {
	_, found, err := ts.findEventForTask(EventTypeTaskArtifactWaived, taskID)
	return found, err
}

// findEventForTask returns the event ID and true if at least one event of
// the given type references the specified taskID. Returns a zero EventID
// and false if no matching event is found. When multiple events match
// (e.g., multiple artifacts per task), returns the first found — iteration
// order depends on the store implementation. This is intentional: any
// artifact satisfies the gate, and ArtifactRef is a convenience pointer
// not a completeness guarantee.
func (ts *TaskStore) findEventForTask(eventType types.EventType, taskID types.EventID) (types.EventID, bool, error) {
	page, err := ts.store.ByType(eventType, 1000, types.None[types.Cursor]())
	if err != nil {
		return types.EventID{}, false, fmt.Errorf("fetch %s events: %w", eventType.Value(), err)
	}
	for _, ev := range page.Items() {
		switch c := ev.Content().(type) {
		case TaskArtifactContent:
			if c.TaskID == taskID {
				return ev.ID(), true, nil
			}
		case TaskArtifactWaivedContent:
			if c.TaskID == taskID {
				return ev.ID(), true, nil
			}
		}
	}
	return types.EventID{}, false, nil
}
