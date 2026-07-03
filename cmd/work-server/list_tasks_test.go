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

func TestListTasksEmitsKanbanFields(t *testing.T) {
	sv, causes := newListTasksTestServer(t)
	convID := types.MustConversationID("conv_00000000000000000000000000000002")

	// Seed a plain task first (keys must be present even when values are empty).
	_, err := sv.ts.Create(sv.humanID, "Plain task", "", causes, convID)
	if err != nil {
		t.Fatalf("Create plain task: %v", err)
	}

	// Seed a task with RiskClass and Cell populated (no factory linkage to avoid
	// the RequirementIDs requirement in validateTaskLinkage).
	_, err = sv.ts.CreateV39(sv.humanID, work.TaskCreateOptions{
		Title:     "Kanban test task",
		RiskClass: "high",
		Cell:      "cell_test",
	}, causes, convID)
	if err != nil {
		t.Fatalf("CreateV39 with risk class: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	sv.listTasks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) == 0 {
		t.Fatal("no tasks returned")
	}
	// Every task item must carry all four Kanban keys.
	for _, key := range []string{"risk_class", "cell", "factory_order_id", "created_at"} {
		if _, ok := resp.Tasks[0][key]; !ok {
			t.Fatalf("/tasks item missing key %q", key)
		}
	}
	// The most-recently seeded task (index 0 — ListSummaries returns newest-first)
	// has non-empty risk_class and cell — verify they round-trip correctly.
	newest := resp.Tasks[0]
	if v, _ := newest["risk_class"].(string); v != "high" {
		t.Fatalf("risk_class = %q, want %q", v, "high")
	}
	if v, _ := newest["cell"].(string); v != "cell_test" {
		t.Fatalf("cell = %q, want %q", v, "cell_test")
	}
	if v, _ := newest["created_at"].(string); v == "" {
		t.Fatal("created_at is empty")
	}
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

func TestTC6_AC7iiiListTasksJSONCanonicalPair(t *testing.T) {
	sv, causes := newListTasksTestServer(t)
	convID := types.MustConversationID("conv_00000000000000000000000000000003")

	known, err := sv.ts.Create(sv.humanID, "Known route task", "", causes, convID)
	if err != nil {
		t.Fatalf("Create known: %v", err)
	}
	if err := sv.ts.TransitionTask(sv.humanID, known.ID, work.StatusReady, "ready", nil, serverHeadCauses(t, sv), convID); err != nil {
		t.Fatalf("TransitionTask known: %v", err)
	}
	unknown, err := sv.ts.Create(sv.humanID, "Unknown route task", "", serverHeadCauses(t, sv), convID)
	if err != nil {
		t.Fatalf("Create unknown: %v", err)
	}
	appendServerRawLifecycleTransition(t, sv, unknown.ID, work.StatusCreated, work.TaskStatus("paused"), convID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	sv.listTasks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	knownItem := findTaskItem(t, resp.Tasks, known.ID.Value())
	unknownItem := findTaskItem(t, resp.Tasks, unknown.ID.Value())
	assertRouteCanonicalSuccess(t, "/tasks known", knownItem)
	assertRouteCanonicalError(t, "/tasks unknown", unknownItem)
	assertRouteKeys(t, "/tasks", knownItem, []string{
		"id", "title", "description", "priority", "created_by", "status",
		"legacy_status", "assignee", "blocked", "artifact_count", "waived",
		"ready", "missing_gates", "missing_facts", "risk_class", "cell",
		"factory_order_id", "created_at", "canonical",
	})
}

func TestTC6_AC7iiiWorkspaceTasksJSONCanonicalPair(t *testing.T) {
	sv, causes := newListTasksTestServer(t)
	convID := types.MustConversationID("conv_00000000000000000000000000000004")

	known, err := sv.ts.CreateInWorkspace(sv.humanID, "Known workspace task", "", "ops", causes, convID)
	if err != nil {
		t.Fatalf("CreateInWorkspace known: %v", err)
	}
	if err := sv.ts.TransitionTask(sv.humanID, known.ID, work.StatusReady, "ready", nil, serverHeadCauses(t, sv), convID); err != nil {
		t.Fatalf("TransitionTask known: %v", err)
	}
	unknown, err := sv.ts.CreateInWorkspace(sv.humanID, "Unknown workspace task", "", "ops", serverHeadCauses(t, sv), convID)
	if err != nil {
		t.Fatalf("CreateInWorkspace unknown: %v", err)
	}
	appendServerRawLifecycleTransition(t, sv, unknown.ID, work.StatusCreated, work.TaskStatus("paused"), convID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/w/ops/tasks", nil)
	req.SetPathValue("workspace", "ops")
	sv.listWorkspaceTasks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Workspace string           `json:"workspace"`
		Tasks     []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Workspace != "ops" {
		t.Fatalf("workspace = %q, want ops", resp.Workspace)
	}
	knownItem := findTaskItem(t, resp.Tasks, known.ID.Value())
	unknownItem := findTaskItem(t, resp.Tasks, unknown.ID.Value())
	assertRouteCanonicalSuccess(t, "/w/{workspace}/tasks known", knownItem)
	assertRouteCanonicalError(t, "/w/{workspace}/tasks unknown", unknownItem)
	assertRouteKeys(t, "/w/{workspace}/tasks", knownItem, []string{
		"id", "title", "description", "priority", "workspace", "created_by",
		"status", "legacy_status", "assignee", "blocked", "artifact_count",
		"waived", "ready", "missing_gates", "missing_facts", "canonical",
	})
}

func appendServerRawLifecycleTransition(t *testing.T, sv *server, taskID types.EventID, from, to work.TaskStatus, convID types.ConversationID) {
	t.Helper()
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(work.EventTypeTaskLifecycleTransitioned, sv.humanID, work.TaskLifecycleTransitionContent{
		TaskID:    taskID,
		FromState: from,
		ToState:   to,
		Reason:    "future status fixture",
		ChangedBy: sv.humanID,
	}, serverHeadCauses(t, sv), convID, sv.store, deriveSignerFromID(sv.humanID))
	if err != nil {
		t.Fatalf("create raw lifecycle transition: %v", err)
	}
	if _, err := sv.store.Append(ev); err != nil {
		t.Fatalf("append raw lifecycle transition: %v", err)
	}
}

func serverHeadCauses(t *testing.T, sv *server) []types.EventID {
	t.Helper()
	head, err := sv.store.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if head.IsNone() {
		return nil
	}
	return []types.EventID{head.Unwrap().ID()}
}

func findTaskItem(t *testing.T, items []map[string]any, id string) map[string]any {
	t.Helper()
	for _, item := range items {
		if got, _ := item["id"].(string); got == id {
			return item
		}
	}
	t.Fatalf("missing task item %s", id)
	return nil
}

func assertRouteCanonicalSuccess(t *testing.T, label string, item map[string]any) {
	t.Helper()
	canonical, ok := item["canonical"].(map[string]any)
	if !ok {
		t.Fatalf("%s missing canonical object: %#v", label, item["canonical"])
	}
	if _, ok := item["canonical_error"]; ok {
		t.Fatalf("%s has canonical_error on success: %#v", label, item["canonical_error"])
	}
	if canonical["phase"] == "" {
		t.Fatalf("%s canonical phase is empty: %#v", label, canonical)
	}
}

func assertRouteCanonicalError(t *testing.T, label string, item map[string]any) {
	t.Helper()
	if _, ok := item["canonical"]; ok {
		t.Fatalf("%s has canonical on error: %#v", label, item["canonical"])
	}
	errText, ok := item["canonical_error"].(string)
	if !ok || errText == "" {
		t.Fatalf("%s missing canonical_error string: %#v", label, item["canonical_error"])
	}
}

func assertRouteKeys(t *testing.T, label string, item map[string]any, want []string) {
	t.Helper()
	if len(item) != len(want) {
		t.Fatalf("%s key count = %d, want %d: %#v", label, len(item), len(want), item)
	}
	for _, key := range want {
		if _, ok := item[key]; !ok {
			t.Fatalf("%s missing key %q in %#v", label, key, item)
		}
	}
}
