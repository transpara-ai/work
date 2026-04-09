package work

import (
	"errors"
	"fmt"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// ErrArtifactRequired is returned by Complete when the task has neither an
// artifact nor an artifact waiver. Callers can check with errors.Is.
var ErrArtifactRequired = errors.New("task has no artifacts; attach an artifact or waive the requirement")

// TaskStatus represents the current lifecycle state of a task.
type TaskStatus string

const (
	// StatusPending means the task has been created but not yet assigned.
	StatusPending TaskStatus = "pending"
	// StatusAssigned means the task has been assigned to an actor but not yet completed.
	StatusAssigned TaskStatus = "assigned"
	// StatusCompleted means the task has been marked complete.
	StatusCompleted TaskStatus = "completed"
)

// Task represents a work item derived from a work.task.created event.
type Task struct {
	ID          types.EventID
	Title       string
	Description string
	CreatedBy   types.ActorID
	Priority    TaskPriority
	Workspace   string
}

// TaskSummary extends Task with computed state fields for efficient list views.
// Status, Assignee, Blocked, ArtifactCount, and Waived are populated by
// ListSummaries using batch store scans.
type TaskSummary struct {
	Task
	Status        TaskStatus
	Assignee      types.ActorID // zero value if unassigned
	Blocked       bool
	ArtifactCount int
	Waived        bool
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
	if title == "" {
		return Task{}, fmt.Errorf("title is required")
	}
	p := DefaultPriority
	if len(priority) > 0 && priority[0] != "" {
		p = priority[0]
	}
	content := TaskCreatedContent{
		Title:       title,
		Description: description,
		CreatedBy:   source,
		Priority:    p,
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
		ID:          stored.ID(),
		Title:       title,
		Description: description,
		CreatedBy:   source,
		Priority:    p,
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
			ID:          ev.ID(),
			Title:       c.Title,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			Priority:    p,
			Workspace:   c.Workspace,
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
	if title == "" {
		return Task{}, fmt.Errorf("title is required")
	}
	p := DefaultPriority
	if len(priority) > 0 && priority[0] != "" {
		p = priority[0]
	}
	content := TaskCreatedContent{
		Title:       title,
		Description: description,
		CreatedBy:   source,
		Priority:    p,
		Workspace:   workspace,
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
		ID:          stored.ID(),
		Title:       title,
		Description: description,
		CreatedBy:   source,
		Priority:    p,
		Workspace:   workspace,
	}, nil
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
			ID:          ev.ID(),
			Title:       c.Title,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			Priority:    p,
			Workspace:   c.Workspace,
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

// GetStatus reconstructs the current status of a task by scanning
// work.task.completed and work.task.assigned events for the given task ID.
// Returns StatusCompleted if a completed event exists, StatusAssigned if an
// assigned event exists, and StatusPending otherwise.
func (ts *TaskStore) GetStatus(taskID types.EventID) (TaskStatus, error) {
	// Check for completed event first.
	completedPage, err := ts.store.ByType(EventTypeTaskCompleted, 1000, types.None[types.Cursor]())
	if err != nil {
		return StatusPending, fmt.Errorf("fetch completed events: %w", err)
	}
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(TaskCompletedContent)
		if ok && c.TaskID == taskID {
			return StatusCompleted, nil
		}
	}

	// Check for assigned event.
	assignedPage, err := ts.store.ByType(EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err != nil {
		return StatusPending, fmt.Errorf("fetch assigned events: %w", err)
	}
	for _, ev := range assignedPage.Items() {
		c, ok := ev.Content().(TaskAssignedContent)
		if ok && c.TaskID == taskID {
			return StatusAssigned, nil
		}
	}

	return StatusPending, nil
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
func (ts *TaskStore) GetDependencies(taskID types.EventID) ([]types.EventID, error) {
	page, err := ts.store.ByType(EventTypeTaskDependencyAdded, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch dependency events: %w", err)
	}
	var deps []types.EventID
	for _, ev := range page.Items() {
		c, ok := ev.Content().(TaskDependencyContent)
		if ok && c.TaskID == taskID {
			deps = append(deps, c.DependsOnID)
		}
	}
	return deps, nil
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

	// Collect all completed task IDs once.
	completedPage, err := ts.store.ByType(EventTypeTaskCompleted, 1000, types.None[types.Cursor]())
	if err != nil {
		return false, fmt.Errorf("fetch completed events: %w", err)
	}
	completedIDs := make(map[types.EventID]bool, len(completedPage.Items()))
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(TaskCompletedContent)
		if ok {
			completedIDs[c.TaskID] = true
		}
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

// ListOpen returns all tasks that do not have a matching work.task.completed event
// and are not blocked by an incomplete dependency.
// It fetches up to 1000 tasks and filters out completed and blocked tasks.
func (ts *TaskStore) ListOpen() ([]Task, error) {
	// Collect all completed task IDs.
	completedPage, err := ts.store.ByType(EventTypeTaskCompleted, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch completed events: %w", err)
	}
	completedIDs := make(map[types.EventID]bool, len(completedPage.Items()))
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(TaskCompletedContent)
		if ok {
			completedIDs[c.TaskID] = true
		}
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
			ID:          created.ID(),
			Title:       cc.Title,
			Description: cc.Description,
			CreatedBy:   cc.CreatedBy,
			Priority:    cp,
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

	// Scan 1: completed events → completedIDs set.
	completedPage, err := ts.store.ByType(EventTypeTaskCompleted, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch completed events: %w", err)
	}
	completedIDs := make(map[types.EventID]bool, len(completedPage.Items()))
	for _, ev := range completedPage.Items() {
		if c, ok := ev.Content().(TaskCompletedContent); ok {
			completedIDs[c.TaskID] = true
		}
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

	// Scan 5: artifact events → count per task.
	artifactPage, err := ts.store.ByType(EventTypeTaskArtifact, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch artifact events: %w", err)
	}
	artifactCount := make(map[types.EventID]int)
	for _, ev := range artifactPage.Items() {
		if c, ok := ev.Content().(TaskArtifactContent); ok {
			artifactCount[c.TaskID]++
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
		status := StatusPending
		if completedIDs[t.ID] {
			status = StatusCompleted
		} else if _, assigned := assigneeMap[t.ID]; assigned {
			status = StatusAssigned
		}
		summaries = append(summaries, TaskSummary{
			Task:          t,
			Status:        status,
			Assignee:      assigneeMap[t.ID],
			Blocked:       blockedMap[t.ID] && !unblockedMap[t.ID],
			ArtifactCount: artifactCount[t.ID],
			Waived:        waivedMap[t.ID],
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

// GetArtifactBody returns the body of a work.task.artifact event by its event ID.
// Returns empty string if not found.
func (ts *TaskStore) GetArtifactBody(artifactID types.EventID) string {
	page, err := ts.store.ByType(EventTypeTaskArtifact, 1000, types.None[types.Cursor]())
	if err != nil {
		return ""
	}
	for _, ev := range page.Items() {
		if ev.ID() == artifactID {
			if c, ok := ev.Content().(TaskArtifactContent); ok {
				return c.Body
			}
		}
	}
	return ""
}

// HasWaiver returns true if a work.task.artifact.waived event exists for the task.
func (ts *TaskStore) HasWaiver(taskID types.EventID) (bool, error) {
	_, found, err := ts.findEventForTask(EventTypeTaskArtifactWaived, taskID)
	return found, err
}

// findEventForTask returns the event ID and true if at least one event of
// the given type references the specified taskID. Returns a zero EventID
// and false if no matching event is found.
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
