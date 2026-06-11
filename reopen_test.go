package work_test

import (
	"strings"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

// The review→fix return edge (run findings v12-F1): a request_changes verdict
// must be able to return a completed task to the open state so the producer's
// existing pickup machinery re-engages. These tests pin the store semantics:
// a completion is live unless a reopen references it — pure set algebra over
// CompletionRefs, no event-order comparison anywhere.

func listOpenIDs(t *testing.T, ts *work.TaskStore) map[types.EventID]bool {
	t.Helper()
	open, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	ids := make(map[types.EventID]bool, len(open))
	for _, task := range open {
		ids[task.ID] = true
	}
	return ids
}

func TestReopenRequiresLiveCompletion(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = ts.Reopen(testActor, task.ID, "review found defects", []string{"issue"}, causes, testConv)
	if err == nil {
		t.Fatal("Reopen on a never-completed task must refuse")
	}
}

func TestReopenRequiresReason(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)

	if err := ts.Reopen(testActor, task.ID, "   ", nil, causes, testConv); err == nil {
		t.Fatal("Reopen with a blank reason must refuse")
	}
}

func TestReopenReturnsTaskToOpen(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Assign(testActor, task.ID, testActor, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)

	if listOpenIDs(t, ts)[task.ID] {
		t.Fatal("completed task must not be open")
	}
	proj, err := ts.ProjectLegacyTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectLegacyTask: %v", err)
	}
	if proj.Status != work.LegacyStatusCompleted {
		t.Fatalf("pre-reopen status = %q, want completed", proj.Status)
	}

	if err := ts.Reopen(testActor, task.ID, "review found defects", []string{"fix the Tier citation"}, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	if !listOpenIDs(t, ts)[task.ID] {
		t.Fatal("reopened task must be open again")
	}
	proj, err = ts.ProjectLegacyTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectLegacyTask after reopen: %v", err)
	}
	if proj.Status == work.LegacyStatusCompleted {
		t.Fatal("reopened task must not project as completed")
	}
	if proj.Status != work.LegacyStatusAssigned {
		t.Fatalf("reopened assigned task status = %q, want assigned (sticky assignment is the return edge)", proj.Status)
	}
}

func TestReopenedDependencyReblocksAggregate(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	parent, err := ts.Create(testActor, "order", "aggregate", causes, testConv)
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	child, err := ts.Create(testActor, "subtask", "piece", causes, testConv)
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	// Corrected decomposition direction (v11-F1): parent depends_on child.
	if err := ts.AddDependency(testActor, parent.ID, child.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	if listOpenIDs(t, ts)[parent.ID] {
		t.Fatal("parent must be hidden while its piece is open")
	}

	completeWithArtifact(t, ts, testActor, child.ID, "done", causes, testConv)
	if !listOpenIDs(t, ts)[parent.ID] {
		t.Fatal("parent must surface once its piece completes")
	}

	if err := ts.Reopen(testActor, child.ID, "review found defects", nil, causes, testConv); err != nil {
		t.Fatalf("Reopen child: %v", err)
	}

	if listOpenIDs(t, ts)[parent.ID] {
		t.Fatal("parent must re-hide while the reopened piece is being fixed")
	}
	blocked, err := ts.IsBlocked(parent.ID)
	if err != nil {
		t.Fatalf("IsBlocked: %v", err)
	}
	if !blocked {
		t.Fatal("parent must read blocked while the reopened piece is open")
	}
}

func TestRecompletionAfterReopenIsCompleted(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)
	if err := ts.Reopen(testActor, task.ID, "review found defects", nil, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	// The fix: a NEW completion event, never referenced by the prior reopen.
	if err := ts.Complete(testActor, task.ID, "fixed", causes, testConv); err != nil {
		t.Fatalf("re-Complete: %v", err)
	}

	if listOpenIDs(t, ts)[task.ID] {
		t.Fatal("re-completed task must not be open")
	}
	proj, err := ts.ProjectLegacyTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectLegacyTask: %v", err)
	}
	if proj.Status != work.LegacyStatusCompleted {
		t.Fatalf("re-completed status = %q, want completed", proj.Status)
	}
}

func TestDoubleReopenRefused(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)
	if err := ts.Reopen(testActor, task.ID, "round 1", nil, causes, testConv); err != nil {
		t.Fatalf("first Reopen: %v", err)
	}
	if err := ts.Reopen(testActor, task.ID, "round 2", nil, causes, testConv); err == nil {
		t.Fatal("second Reopen without re-completion must refuse (no live completion)")
	}
}

func TestListReopensReturnsFeedback(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)
	issues := []string{"fix the Tier citation", "reword the Notes claim"}
	if err := ts.Reopen(testActor, task.ID, "request_changes: Tier rows wrong", issues, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	reopens, err := ts.ListReopens(task.ID)
	if err != nil {
		t.Fatalf("ListReopens: %v", err)
	}
	if len(reopens) != 1 {
		t.Fatalf("got %d reopens, want 1", len(reopens))
	}
	r := reopens[0]
	if r.TaskID != task.ID || r.ReopenedBy != testActor {
		t.Fatalf("reopen identity mismatch: %+v", r)
	}
	if !strings.Contains(r.Reason, "Tier rows wrong") {
		t.Fatalf("reason = %q, want the review summary", r.Reason)
	}
	if len(r.Issues) != 2 || r.Issues[0] != issues[0] || r.Issues[1] != issues[1] {
		t.Fatalf("issues = %v, want %v", r.Issues, issues)
	}
}

func TestListReopensChronological(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)
	if err := ts.Reopen(testActor, task.ID, "round 1", nil, causes, testConv); err != nil {
		t.Fatalf("Reopen 1: %v", err)
	}
	if err := ts.Complete(testActor, task.ID, "fixed once", causes, testConv); err != nil {
		t.Fatalf("re-Complete: %v", err)
	}
	if err := ts.Reopen(testActor, task.ID, "round 2", nil, causes, testConv); err != nil {
		t.Fatalf("Reopen 2: %v", err)
	}

	reopens, err := ts.ListReopens(task.ID)
	if err != nil {
		t.Fatalf("ListReopens: %v", err)
	}
	// The Operate instruction numbers these as rounds — chronological order
	// is load-bearing (ByType pages newest-first; ListReopens must reverse).
	if len(reopens) != 2 || reopens[0].Reason != "round 1" || reopens[1].Reason != "round 2" {
		t.Fatalf("reopens must be chronological (oldest first); got %+v", reopens)
	}
}

func TestBatchStatusReflectsReopen(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "subtask", "do the thing", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completeWithArtifact(t, ts, testActor, task.ID, "done", causes, testConv)
	if err := ts.Reopen(testActor, task.ID, "review found defects", nil, causes, testConv); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	summaries, err := ts.ListSummaries(100)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	for _, sum := range summaries {
		if sum.ID == task.ID && sum.LegacyStatus == work.LegacyStatusCompleted {
			t.Fatal("batch status must not report a reopened task as completed")
		}
	}
}
