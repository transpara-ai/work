package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

func newListTasksTestServer(t *testing.T) (*server, []types.EventID) {
	t.Helper()
	s := store.NewInMemoryStore()
	humanID := types.MustActorID("actor_00000000000000000000000000000001")
	if err := bootstrapGraph(s, humanID); err != nil {
		t.Fatalf("bootstrapGraph: %v", err)
	}
	head, err := s.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if head.IsNone() {
		t.Fatal("missing bootstrap head")
	}
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	return &server{
		ts:      work.NewTaskStore(s, factory, deriveSignerFromID(humanID)),
		store:   s,
		humanID: humanID,
	}, []types.EventID{head.Unwrap().ID()}
}

func TestListTasksOpenOnlyExcludesCanonicalTerminalStatuses(t *testing.T) {
	sv, causes := newListTasksTestServer(t)
	convID := types.MustConversationID("conv_00000000000000000000000000000001")

	openTask, err := sv.ts.Create(sv.humanID, "Open task", "", causes, convID)
	if err != nil {
		t.Fatalf("Create open task: %v", err)
	}

	terminalTasks := make(map[string]work.TaskStatus)
	for _, terminal := range []work.TaskStatus{work.StatusCertified, work.StatusRejected, work.StatusSuperseded} {
		task, err := sv.ts.Create(sv.humanID, "Terminal "+string(terminal), "", causes, convID)
		if err != nil {
			t.Fatalf("Create terminal task: %v", err)
		}
		switch terminal {
		case work.StatusCertified:
			for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified, work.StatusCertified} {
				if err := sv.ts.TransitionTask(sv.humanID, task.ID, state, "advance", nil, causes, convID); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
		case work.StatusRejected:
			for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified} {
				if err := sv.ts.TransitionTask(sv.humanID, task.ID, state, "advance", nil, causes, convID); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
			if err := sv.ts.RejectTask(sv.humanID, task.ID, "not accepted", nil, causes, convID); err != nil {
				t.Fatalf("RejectTask: %v", err)
			}
		case work.StatusSuperseded:
			if err := sv.ts.SupersedeTask(sv.humanID, task.ID, "tsk_replacement_"+string(terminal), "duplicate", nil, causes, convID); err != nil {
				t.Fatalf("SupersedeTask: %v", err)
			}
		}
		if legacyStatus, err := sv.ts.GetCompatibilityStatus(task.ID); err != nil || legacyStatus != work.LegacyStatusPending {
			t.Fatalf("terminal task legacy status = %q, %v; want pending", legacyStatus, err)
		}
		terminalTasks[task.ID.Value()] = terminal
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks?open=true", nil)
	sv.listTasks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d; body %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	seen := make(map[string]bool)
	for _, task := range body.Tasks {
		seen[task.ID] = true
	}
	if !seen[openTask.ID.Value()] {
		t.Fatal("open non-terminal task missing from open-only list")
	}
	for id, terminal := range terminalTasks {
		if seen[id] {
			t.Fatalf("%s task %s appeared in open-only list", terminal, id)
		}
	}
}
