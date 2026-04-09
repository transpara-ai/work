package work_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
	"github.com/lovyou-ai/work"
)

var (
	testActor = types.MustActorID("actor_00000000000000000000000000000001")
	testConv  = types.MustConversationID("conv_00000000000000000000000000000001")
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

// setupStore bootstraps an in-memory graph and returns the store and genesis causes.
func setupStore(t *testing.T) (*store.InMemoryStore, []types.EventID) {
	t.Helper()
	s := store.NewInMemoryStore()
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	bf := event.NewBootstrapFactory(registry)
	boot, err := bf.Init(testActor, testSigner{})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	stored, err := s.Append(boot)
	if err != nil {
		t.Fatalf("append genesis: %v", err)
	}
	return s, []types.EventID{stored.ID()}
}

// newTaskStore creates a TaskStore against the given store.
func newTaskStore(t *testing.T, s *store.InMemoryStore) *work.TaskStore {
	t.Helper()
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	return work.NewTaskStore(s, factory, testSigner{})
}

// completeWithArtifact is a test helper that adds a default artifact and then
// completes the task, satisfying the artifact gate.
func completeWithArtifact(t *testing.T, ts *work.TaskStore, actor types.ActorID, taskID types.EventID, summary string, causes []types.EventID, convID types.ConversationID) {
	t.Helper()
	if err := ts.AddArtifact(actor, taskID, "result", "text/plain", "done", causes, convID); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}
	if err := ts.Complete(actor, taskID, summary, causes, convID); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestTaskStore_Create(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Fix the auth bug", "login fails on mobile", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Title != "Fix the auth bug" {
		t.Errorf("Title = %q; want %q", task.Title, "Fix the auth bug")
	}
	if task.Description != "login fails on mobile" {
		t.Errorf("Description = %q; want %q", task.Description, "login fails on mobile")
	}
	if task.CreatedBy != testActor {
		t.Errorf("CreatedBy = %v; want %v", task.CreatedBy, testActor)
	}
}

func TestTaskStore_Create_RequiresTitle(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := ts.Create(testActor, "", "no title", causes, testConv)
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestTaskStore_List_Empty(t *testing.T) {
	s, _ := setupStore(t)
	ts := newTaskStore(t, s)

	tasks, err := ts.List(20)
	if err != nil {
		t.Fatalf("List (empty): %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks; got %d", len(tasks))
	}
}

func TestTaskStore_List(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := ts.Create(testActor, "Task Alpha", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create Alpha: %v", err)
	}
	_, err = ts.Create(testActor, "Task Beta", "some detail", causes, testConv)
	if err != nil {
		t.Fatalf("Create Beta: %v", err)
	}

	tasks, err := ts.List(20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks; got %d", len(tasks))
	}
}

func TestTaskStore_List_RespectsLimit(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	for i := 0; i < 5; i++ {
		if _, err := ts.Create(testActor, "Task", "", causes, testConv); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	tasks, err := ts.List(3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks (limit=3); got %d", len(tasks))
	}
}

var testAssignee = types.MustActorID("actor_00000000000000000000000000000002")

func TestTaskStore_Assign(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Do the thing", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.Assign(testActor, task.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
}

func TestTaskStore_Complete(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Ship the feature", "needs tests", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	completeWithArtifact(t, ts, testActor, task.ID, "shipped in PR #42", causes, testConv)
}

func TestTaskStore_GetByAssignee_Empty(t *testing.T) {
	s, _ := setupStore(t)
	ts := newTaskStore(t, s)

	tasks, err := ts.GetByAssignee(testAssignee)
	if err != nil {
		t.Fatalf("GetByAssignee (empty): %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks; got %d", len(tasks))
	}
}

func TestTaskStore_GetByAssignee(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Assign A to testAssignee, B to testActor (not the target assignee).
	if err := ts.Assign(testActor, taskA.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign A: %v", err)
	}
	if err := ts.Assign(testActor, taskB.ID, testActor, causes, testConv); err != nil {
		t.Fatalf("Assign B: %v", err)
	}

	tasks, err := ts.GetByAssignee(testAssignee)
	if err != nil {
		t.Fatalf("GetByAssignee: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task for testAssignee; got %d", len(tasks))
	}
	if tasks[0].Title != "Task A" {
		t.Errorf("Title = %q; want %q", tasks[0].Title, "Task A")
	}
}

func TestTaskStore_GetStatus_Pending(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Pending task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusPending {
		t.Errorf("status = %q; want %q", status, work.StatusPending)
	}
}

func TestTaskStore_GetStatus_Assigned(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Assigned task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Assign(testActor, task.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusAssigned {
		t.Errorf("status = %q; want %q", status, work.StatusAssigned)
	}
}

func TestTaskStore_GetStatus_Completed(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Completed task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Assign(testActor, task.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)

	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusCompleted {
		t.Errorf("status = %q; want %q", status, work.StatusCompleted)
	}
}

func TestTaskStore_ListOpen_Empty(t *testing.T) {
	s, _ := setupStore(t)
	ts := newTaskStore(t, s)

	tasks, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen (empty): %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 open tasks; got %d", len(tasks))
	}
}

func TestTaskStore_ListOpen_FiltersCompleted(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	// Create three tasks.
	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	_, err = ts.Create(testActor, "Task C", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create C: %v", err)
	}

	// Complete A and B; leave C open.
	completeWithArtifact(t, ts, testActor, taskA.ID, "done A", causes, testConv)
	completeWithArtifact(t, ts, testActor, taskB.ID, "done B", causes, testConv)

	open, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("expected 1 open task; got %d", len(open))
	}
	if open[0].Title != "Task C" {
		t.Errorf("open task title = %q; want %q", open[0].Title, "Task C")
	}
}

func TestTaskStore_AddDependency(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}
}

func TestTaskStore_GetDependencies(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// B depends on A.
	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	deps, err := ts.GetDependencies(taskB.ID)
	if err != nil {
		t.Fatalf("GetDependencies: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency; got %d", len(deps))
	}
	if deps[0] != taskA.ID {
		t.Errorf("dependency = %v; want %v", deps[0], taskA.ID)
	}

	// A has no dependencies.
	depsA, err := ts.GetDependencies(taskA.ID)
	if err != nil {
		t.Fatalf("GetDependencies A: %v", err)
	}
	if len(depsA) != 0 {
		t.Errorf("expected 0 dependencies for A; got %d", len(depsA))
	}
}

func TestTaskStore_IsBlocked_NoDependencies(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Standalone task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	blocked, err := ts.IsBlocked(task.ID)
	if err != nil {
		t.Fatalf("IsBlocked: %v", err)
	}
	if blocked {
		t.Error("expected task with no dependencies to be unblocked")
	}
}

func TestTaskStore_IsBlocked_BlockedByIncomplete(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// B depends on A (not yet complete).
	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	blocked, err := ts.IsBlocked(taskB.ID)
	if err != nil {
		t.Fatalf("IsBlocked: %v", err)
	}
	if !blocked {
		t.Error("expected task B to be blocked by incomplete task A")
	}
}

func TestTaskStore_IsBlocked_UnblockedWhenDepCompleted(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// B depends on A.
	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	// Complete A.
	completeWithArtifact(t, ts, testActor, taskA.ID, "done", causes, testConv)

	blocked, err := ts.IsBlocked(taskB.ID)
	if err != nil {
		t.Fatalf("IsBlocked: %v", err)
	}
	if blocked {
		t.Error("expected task B to be unblocked after task A completed")
	}
}

func TestTaskStore_ListOpen_ExcludesBlocked(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	// Create two tasks: A and B (B depends on A).
	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// B depends on A — so B is blocked.
	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	// ListOpen should return only A (B is blocked).
	open, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("expected 1 open task; got %d", len(open))
	}
	if open[0].ID != taskA.ID {
		t.Errorf("open task = %v; want %v (Task A)", open[0].ID, taskA.ID)
	}
}

func TestTaskStore_ListOpen_UnblocksAfterDepCompleted(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	taskA, err := ts.Create(testActor, "Task A", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	taskB, err := ts.Create(testActor, "Task B", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// B depends on A.
	if err := ts.AddDependency(testActor, taskB.ID, taskA.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	// Complete A — B should now be unblocked.
	completeWithArtifact(t, ts, testActor, taskA.ID, "done", causes, testConv)

	// ListOpen should return only B (A is completed, B is now unblocked).
	open, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("expected 1 open task; got %d", len(open))
	}
	if open[0].ID != taskB.ID {
		t.Errorf("open task = %v; want %v (Task B)", open[0].ID, taskB.ID)
	}
}
