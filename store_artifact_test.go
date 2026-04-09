package work_test

import (
	"errors"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
	"github.com/lovyou-ai/work"
)

func TestTaskStore_AddArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Research task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "Analysis report", "text/markdown", "# Results\nAll good.", causes, testConv); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}
}

func TestTaskStore_AddArtifact_RequiresLabel(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Some task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "", "text/plain", "body", causes, testConv); err == nil {
		t.Fatal("expected error for empty label, got nil")
	}
}

func TestTaskStore_AddArtifact_DefaultMediaType(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "Result", "", "body", causes, testConv); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact; got %d", len(artifacts))
	}
	if artifacts[0].MediaType != "text/markdown" {
		t.Errorf("MediaType = %q; want %q", artifacts[0].MediaType, "text/markdown")
	}
}

func TestTaskStore_ListArtifacts_Empty(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "No artifacts", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) != 0 {
		t.Errorf("expected 0 artifacts; got %d", len(artifacts))
	}
}

func TestTaskStore_ListArtifacts_Multiple(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Multi-artifact task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "Code", "text/plain", "func main(){}", causes, testConv); err != nil {
		t.Fatalf("AddArtifact 1: %v", err)
	}
	if err := ts.AddArtifact(testActor, task.ID, "Tests", "text/plain", "func TestMain(){}", causes, testConv); err != nil {
		t.Fatalf("AddArtifact 2: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts; got %d", len(artifacts))
	}
	// ByType returns newest-first; verify both labels are present.
	labels := map[string]bool{artifacts[0].Label: true, artifacts[1].Label: true}
	if !labels["Code"] || !labels["Tests"] {
		t.Errorf("expected labels {Code, Tests}; got {%q, %q}", artifacts[0].Label, artifacts[1].Label)
	}
}

func TestTaskStore_ListArtifacts_FiltersByTask(t *testing.T) {
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

	if err := ts.AddArtifact(testActor, taskA.ID, "A result", "text/plain", "a", causes, testConv); err != nil {
		t.Fatalf("AddArtifact A: %v", err)
	}
	if err := ts.AddArtifact(testActor, taskB.ID, "B result", "text/plain", "b", causes, testConv); err != nil {
		t.Fatalf("AddArtifact B: %v", err)
	}

	artifacts, err := ts.ListArtifacts(taskA.ID)
	if err != nil {
		t.Fatalf("ListArtifacts A: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact for A; got %d", len(artifacts))
	}
	if artifacts[0].Label != "A result" {
		t.Errorf("Label = %q; want %q", artifacts[0].Label, "A result")
	}
}

func TestTaskStore_WaiveArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Operational task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "service restart — no deliverable", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}
}

func TestTaskStore_WaiveArtifact_RequiresReason(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "", causes, testConv); err == nil {
		t.Fatal("expected error for empty reason, got nil")
	}
}

// --- Completion gate tests ---

func TestTaskStore_Complete_RequiresArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Gated task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = ts.Complete(testActor, task.ID, "done", causes, testConv)
	if err == nil {
		t.Fatal("expected error from artifact gate, got nil")
	}
	if !errors.Is(err, work.ErrArtifactRequired) {
		t.Errorf("error = %v; want ErrArtifactRequired", err)
	}
}

func TestTaskStore_Complete_WithArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task with artifact", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "Output", "text/plain", "result", causes, testConv); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}

	if err := ts.Complete(testActor, task.ID, "done", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusCompleted {
		t.Errorf("status = %q; want %q", status, work.StatusCompleted)
	}
}

func TestTaskStore_Complete_WithWaiver(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Waived task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "operational — no output", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}

	if err := ts.Complete(testActor, task.ID, "done", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusCompleted {
		t.Errorf("status = %q; want %q", status, work.StatusCompleted)
	}
}

// --- ArtifactRef verification ---

func TestTaskStore_Complete_ArtifactRef_PointsToArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task with artifact ref", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "Output", "text/plain", "result", causes, testConv); err != nil {
		t.Fatalf("AddArtifact: %v", err)
	}

	if err := ts.Complete(testActor, task.ID, "done", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Find the completed event and check ArtifactRef.
	completedPage, err := s.ByType(work.EventTypeTaskCompleted, 100, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	var found bool
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(work.TaskCompletedContent)
		if !ok || c.TaskID != task.ID {
			continue
		}
		found = true
		if c.ArtifactRef.IsZero() {
			t.Fatal("ArtifactRef is zero; expected it to point to the artifact event")
		}
		// Verify the referenced event is actually a work.task.artifact.
		artifactPage, err := s.ByType(work.EventTypeTaskArtifact, 100, types.None[types.Cursor]())
		if err != nil {
			t.Fatalf("ByType artifact: %v", err)
		}
		var refFound bool
		for _, aev := range artifactPage.Items() {
			if aev.ID() == c.ArtifactRef {
				refFound = true
				break
			}
		}
		if !refFound {
			t.Errorf("ArtifactRef %s does not match any work.task.artifact event", c.ArtifactRef.Value())
		}

		// Verify GetArtifactBody returns the artifact body via the ref.
		body, ok := ts.GetArtifactBody(c.ArtifactRef)
		if !ok {
			t.Fatal("GetArtifactBody returned false for valid artifact ref")
		}
		if body != "result" {
			t.Errorf("GetArtifactBody = %q, want %q", body, "result")
		}
	}
	if !found {
		t.Fatal("no completed event found for task")
	}
}

func TestTaskStore_GetArtifactBody_WaiverReturnsFalse(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Waived task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "operational", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}

	if err := ts.Complete(testActor, task.ID, "done", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// ArtifactRef should point to the waiver.
	completedPage, err := s.ByType(work.EventTypeTaskCompleted, 100, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(work.TaskCompletedContent)
		if !ok || c.TaskID != task.ID {
			continue
		}
		// GetArtifactBody should return false for a waiver event.
		_, ok = ts.GetArtifactBody(c.ArtifactRef)
		if ok {
			t.Fatal("GetArtifactBody returned true for a waiver ref; expected false")
		}
		return
	}
	t.Fatal("no completed event found")
}

func TestTaskStore_Complete_ArtifactRef_PointsToWaiver(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Waived task with ref", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "operational", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}

	if err := ts.Complete(testActor, task.ID, "done", causes, testConv); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	completedPage, err := s.ByType(work.EventTypeTaskCompleted, 100, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	for _, ev := range completedPage.Items() {
		c, ok := ev.Content().(work.TaskCompletedContent)
		if !ok || c.TaskID != task.ID {
			continue
		}
		if c.ArtifactRef.IsZero() {
			t.Fatal("ArtifactRef is zero; expected it to point to the waiver event")
		}
		return
	}
	t.Fatal("no completed event found for task")
}

// --- batchStatus artifact/waiver fields ---

func TestTaskStore_ListSummaries_ArtifactCount(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddArtifact(testActor, task.ID, "A", "text/plain", "a", causes, testConv); err != nil {
		t.Fatalf("AddArtifact 1: %v", err)
	}
	if err := ts.AddArtifact(testActor, task.ID, "B", "text/plain", "b", causes, testConv); err != nil {
		t.Fatalf("AddArtifact 2: %v", err)
	}

	summaries, err := ts.ListSummaries(20)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary; got %d", len(summaries))
	}
	if summaries[0].ArtifactCount != 2 {
		t.Errorf("ArtifactCount = %d; want 2", summaries[0].ArtifactCount)
	}
	if summaries[0].Waived {
		t.Error("expected Waived = false")
	}
}

func TestTaskStore_ListSummaries_Waived(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.WaiveArtifact(testActor, task.ID, "ops task", causes, testConv); err != nil {
		t.Fatalf("WaiveArtifact: %v", err)
	}

	summaries, err := ts.ListSummaries(20)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary; got %d", len(summaries))
	}
	if summaries[0].ArtifactCount != 0 {
		t.Errorf("ArtifactCount = %d; want 0", summaries[0].ArtifactCount)
	}
	if !summaries[0].Waived {
		t.Error("expected Waived = true")
	}
}
