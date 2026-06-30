# Work /tasks Kanban-fields Enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface the per-task fields the Mission Control Kanban needs — `risk_class`, `cell`, `factory_order_id`, and a creation timestamp (`created_at`, for card aging) — on the work-server `GET /tasks` response.

**Architecture:** `work` is the data provider; `site` is the consumer (the Kanban). `risk_class`/`cell`/`factory_order_id` already live on the `Task` struct (built from `TaskCreatedContent`); they just aren't emitted by the `/tasks` handler. The creation time is the timestamp of the `work.task.created` event (the task ID *is* that event's ID), so we add a `CreatedAt time.Time` to `Task`, populate it in `List` (which `ListSummaries` feeds through `batchStatus`), and emit all four fields.

**Tech Stack:** Go (event-sourced TaskStore on the eventgraph framework), `net/http` work-server. No new dependencies.

**Plan scope:** This is the data-provider half of Mission Control Plan 2 (Kanban). The `site` Kanban consumer is a separate plan. Design source: `transpara-ai/site:docs/designs/mission-control-console-design-v0.1.0.md`.

## Global Constraints

- **Additive, backward-compatible.** Only ADD fields; do not rename or remove existing `/tasks` JSON keys. Existing consumers must keep working.
- **No fabrication.** `created_at` is the real creation-event timestamp; never a synthesized value.
- **Go:** handle every error explicitly (no `_ = err`); table-driven tests; `*_test.go` in the same package.
- **Commits:** conventional, lowercase imperative subject, no trailing period. On branch `feat/work-tasks-kanban-fields` (never commit to `main`).
- **Verify:** `go build ./cmd/...` · `go vet ./...` · `go test ./...` (in-memory store; no DB needed for these tests).

---

### Task 1: Add `CreatedAt` to `Task` and populate it from the creation event

**Files:**
- Modify: `store.go` (the `Task` struct ~line 71; the `List` builder ~line 358)
- Test: `store_test.go` (same package `work`)

**Interfaces:**
- Consumes: `ev.Timestamp().Value()` returns `time.Time` (confirmed: `types.Timestamp.Value()` in eventgraph).
- Produces: `Task.CreatedAt time.Time`, populated for every `Task` returned by `List` (and therefore `ListSummaries` → `batchStatus`, which embeds `Task: t`).

- [ ] **Step 1: Write the failing test**

Add to `store_test.go` (package `work`). Use the store's existing test constructor — match how other tests in this file build a `TaskStore` and create a task (look at an existing `TestList…`/`Create` test for the exact constructor and `Create`/`CreateV39` call; reuse that idiom verbatim):

```go
func TestListPopulatesCreatedAt(t *testing.T) {
	ts := newTestStore(t) // use whatever this file's existing tests use to build a TaskStore
	before := time.Now().UTC().Add(-time.Second)

	task, err := ts.Create(testSource(t), "kanban aging task", "desc") // match the existing Create signature in this file
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	tasks, err := ts.List(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *Task
	for i := range tasks {
		if tasks[i].ID == task.ID {
			found = &tasks[i]
		}
	}
	if found == nil {
		t.Fatal("created task not returned by List")
	}
	if found.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero; want the creation-event timestamp")
	}
	if found.CreatedAt.Before(before) {
		t.Fatalf("CreatedAt %v is before test start %v", found.CreatedAt, before)
	}
}
```

NOTE to implementer: the exact `Create`/source helpers vary in this file — read one existing test in `store_test.go` first and copy its setup verbatim (constructor, source/actor, create call). Keep the assertions above.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./ -run TestListPopulatesCreatedAt -v`
Expected: FAIL — `found.CreatedAt` is zero (field doesn't exist yet → compile error, then zero after the struct field is added but before `List` populates it).

- [ ] **Step 3: Implement**

In the `Task` struct (`store.go`), add the field (keep it with the other value fields):

```go
	RiskClass              string
	ExpectedOutputs        []string
	CreatedAt              time.Time // timestamp of the work.task.created event
```

In `List` (`store.go`), add `CreatedAt` to the `Task{...}` literal built from `ev`:

```go
			Cell:                   c.Cell,
			RiskClass:              c.RiskClass,
			ExpectedOutputs:        cloneStrings(c.ExpectedOutputs),
			CreatedAt:              ev.Timestamp().Value(),
```

(Ensure `time` is imported in `store.go` — it almost certainly already is.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./ -run TestListPopulatesCreatedAt -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add store.go store_test.go
git commit -m "feat: populate task createdat from the creation event"
```

---

### Task 2: Emit the four Kanban fields on `GET /tasks`

**Files:**
- Modify: `cmd/work-server/main.go` (`listTasks`, the per-task `map[string]any` ~line 1106-1120)
- Test: `cmd/work-server/main_test.go` (or the existing work-server handler test file — match its package and pattern)

**Interfaces:**
- Consumes: `s.Task.RiskClass`, `s.Task.Cell`, `s.Task.FactoryOrderID`, `s.Task.CreatedAt` (Task 1).
- Produces: `/tasks` JSON items gain `risk_class`, `cell`, `factory_order_id`, `created_at` (RFC3339).

- [ ] **Step 1: Write the failing test**

Find the existing work-server test that exercises `listTasks` / `GET /tasks` (search `cmd/work-server/*_test.go` for `"/tasks"`). Match its server construction and auth. Add a test asserting the new keys appear:

```go
func TestListTasksEmitsKanbanFields(t *testing.T) {
	sv := newTestServer(t) // match the existing work-server test harness
	// create a task with a risk class / cell / factory order via the store the
	// test server uses — reuse whatever the existing /tasks test does to seed a task,
	// setting RiskClass/Cell/FactoryOrderID through the same create path.

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	// add the Bearer auth header the existing /tasks test uses, if any
	w := httptest.NewRecorder()
	sv.listTasks(w, req) // or route through the mux the existing test uses

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) == 0 {
		t.Fatal("no tasks returned")
	}
	for _, key := range []string{"risk_class", "cell", "factory_order_id", "created_at"} {
		if _, ok := resp.Tasks[0][key]; !ok {
			t.Fatalf("/tasks item missing key %q", key)
		}
	}
}
```

NOTE to implementer: read the existing `/tasks` test first and mirror its server/store/auth setup exactly; only the new-key assertions are novel. If the existing test seeds tasks without risk/cell, also seed one with a non-empty `RiskClass` so the value (not just the key) is exercised.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/work-server/ -run TestListTasksEmitsKanbanFields -v`
Expected: FAIL — keys absent.

- [ ] **Step 3: Implement**

In `listTasks` (`cmd/work-server/main.go`), add four entries to the per-task map (alongside the existing keys):

```go
			"missing_facts":    s.MissingFacts,
			"risk_class":       s.Task.RiskClass,
			"cell":             s.Task.Cell,
			"factory_order_id": s.Task.FactoryOrderID,
			"created_at":       s.Task.CreatedAt.UTC().Format(time.RFC3339),
```

(Ensure `time` is imported in `cmd/work-server/main.go`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/work-server/ -run TestListTasksEmitsKanbanFields -v`
Expected: PASS.

- [ ] **Step 5: Full verify + commit**

```bash
go build ./cmd/... && go vet ./... && go test ./...
git add cmd/work-server/main.go cmd/work-server/main_test.go
git commit -m "feat: emit risk class, cell, factory order id, and created_at on /tasks"
```

(Adjust the test path in `git add` to the actual work-server test file you edited.)

---

## Self-Review

- **Spec coverage:** `risk_class`, `cell`, `factory_order_id` (Task 2) + `created_at` aging timestamp (Tasks 1-2). The site Kanban consumes these. ✓
- **Backward compatibility:** only additive map keys + one new struct field; no existing key renamed/removed. ✓
- **No fabrication:** `created_at` is the real `work.task.created` event timestamp. ✓
- **Deferred (not in this plan):** effort-to-date / predicted-effort and a clean linked-PR URL remain unexposed — they require richer runtime-evidence/release-candidate projection work and are tracked for a later enrichment. The site Kanban renders those as explicit "unavailable".
