package work_test

import (
	"testing"

	"github.com/lovyou-ai/work"
)

func TestTaskStore_Create_DefaultPriority(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Default priority task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Priority != work.PriorityMedium {
		t.Errorf("Priority = %q; want %q (default)", task.Priority, work.PriorityMedium)
	}
}

func TestTaskStore_Create_ExplicitPriority(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Critical task", "must fix now", causes, testConv, work.PriorityCritical)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Priority != work.PriorityCritical {
		t.Errorf("Priority = %q; want %q", task.Priority, work.PriorityCritical)
	}
}

func TestTaskStore_GetPriority_FallsBackToCreationDefault(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "High priority task", "", causes, testConv, work.PriorityHigh)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	p, err := ts.GetPriority(task.ID)
	if err != nil {
		t.Fatalf("GetPriority: %v", err)
	}
	if p != work.PriorityHigh {
		t.Errorf("GetPriority = %q; want %q", p, work.PriorityHigh)
	}
}

func TestTaskStore_GetPriority_DefaultWhenNotSet(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Unspecified priority task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	p, err := ts.GetPriority(task.ID)
	if err != nil {
		t.Fatalf("GetPriority: %v", err)
	}
	if p != work.DefaultPriority {
		t.Errorf("GetPriority = %q; want %q (default)", p, work.DefaultPriority)
	}
}

func TestTaskStore_SetPriority(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task to reprioritise", "", causes, testConv, work.PriorityLow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.SetPriority(testActor, task.ID, work.PriorityCritical, causes, testConv); err != nil {
		t.Fatalf("SetPriority: %v", err)
	}
}

func TestTaskStore_GetPriority_ReturnsLatestAfterSet(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	// Create with low priority.
	task, err := ts.Create(testActor, "Escalating task", "", causes, testConv, work.PriorityLow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Override with critical.
	if err := ts.SetPriority(testActor, task.ID, work.PriorityCritical, causes, testConv); err != nil {
		t.Fatalf("SetPriority: %v", err)
	}

	p, err := ts.GetPriority(task.ID)
	if err != nil {
		t.Fatalf("GetPriority: %v", err)
	}
	if p != work.PriorityCritical {
		t.Errorf("GetPriority = %q; want %q (after override)", p, work.PriorityCritical)
	}
}

func TestTaskStore_GetPriority_MultipleOverrides_ReturnsLatest(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task with multiple overrides", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Set priority three times — last one wins.
	for _, p := range []work.TaskPriority{work.PriorityHigh, work.PriorityCritical, work.PriorityLow} {
		if err := ts.SetPriority(testActor, task.ID, p, causes, testConv); err != nil {
			t.Fatalf("SetPriority(%s): %v", p, err)
		}
	}

	p, err := ts.GetPriority(task.ID)
	if err != nil {
		t.Fatalf("GetPriority: %v", err)
	}
	if p != work.PriorityLow {
		t.Errorf("GetPriority = %q; want %q (last override)", p, work.PriorityLow)
	}
}

func TestTaskStore_List_IncludesPriority(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := ts.Create(testActor, "Critical task", "", causes, testConv, work.PriorityCritical)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	tasks, err := ts.List(20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task; got %d", len(tasks))
	}
	if tasks[0].Priority != work.PriorityCritical {
		t.Errorf("Priority = %q; want %q", tasks[0].Priority, work.PriorityCritical)
	}
}
