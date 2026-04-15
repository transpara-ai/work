# Time-Windowed Agent Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Now/1h/24h time window controls to the telemetry dashboard with per-agent state timeline tracking, visual differentiation (active/stale/stuck/terminated), and a paused-mode banner for historical views.

**Architecture:** New `GET /telemetry/agents/history?window=1h|24h` endpoint computes state transitions from existing `telemetry_agent_snapshots` rows using `LAG()` window functions. Dashboard JS adds time window buttons, freezes SSE rendering in historical mode, and renders agent cards with left-border color coding and a state timeline bar. No schema changes.

**Tech Stack:** Go 1.23 stdlib `net/http` + `pgx/v5`, inline JS/CSS in embedded HTML const.

**Spec:** `docs/designs/2026-04-15-time-windowed-agent-dashboard-design.md`

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `cmd/work-server/telemetry_history.go` | Create | Types (`telAgentHistory`, `telStateSpan`), `queryAgentHistory()`, `telemetryAgentHistory` handler |
| `cmd/work-server/telemetry_history_test.go` | Create | Unit tests for `buildAgentHistories()` pure function |
| `cmd/work-server/telemetry_dashboard.go` | Modify | CSS for time window buttons, card borders, state timeline bar, stuck pulse. JS for window switching, paused mode, `renderAgents` rewrite, `buildAgentCard` rewrite |
| `cmd/work-server/main.go` | Modify | Register `GET /telemetry/agents/history` route (line ~746) |

---

### Task 1: Agent History Types and Pure Computation Function

**Files:**
- Create: `cmd/work-server/telemetry_history.go`

- [ ] **Step 1: Create the file with types and the pure computation function**

The core logic takes raw snapshot rows (sorted by actor_id, recorded_at) and produces agent history objects. This is a pure function — no DB dependency — so it's testable without Postgres.

```go
package main

import "time"

// telStateSpan represents a contiguous period an agent spent in one FSM state.
type telStateSpan struct {
	State     string    `json:"state"`
	EnteredAt time.Time `json:"entered_at"`
	Duration  float64   `json:"duration_seconds"`
}

// telAgentHistory is the JSON shape for an agent's lifecycle within a time window.
type telAgentHistory struct {
	Role          string     `json:"role"`
	ActorID       string     `json:"actor_id"`
	CurrentState  string     `json:"current_state"`
	Model         string     `json:"model"`
	Iteration     int        `json:"iteration"`
	MaxIterations int        `json:"max_iterations"`
	TokensUsed    int64      `json:"tokens_used"`
	CostUSD       float64    `json:"cost_usd"`
	TrustScore    *float64   `json:"trust_score"`
	Errors        int        `json:"errors"`
	FirstSeen     time.Time  `json:"first_seen"`
	LastSeen      time.Time  `json:"last_seen"`
	States        []telStateSpan `json:"states"`
}

// snapshotRow is a single row from telemetry_agent_snapshots, used as input
// to the pure buildAgentHistories function.
type snapshotRow struct {
	ActorID    string
	Role       string
	State      string
	Model      string
	Iteration  int
	MaxIter    int
	TokensUsed int64
	CostUSD    float64
	TrustScore *float64
	Errors     int
	RecordedAt time.Time
}

// stuckThreshold is the duration after which a non-terminal agent is
// considered stuck. If consecutive snapshots for the same actor are spaced
// further apart than this and the state did not change to a terminal state,
// a synthetic "stuck" span is inserted.
//
// Future: this could be made configurable or derived from the telemetry
// write interval. For now 2 minutes matches the dashboard design spec.
const stuckThreshold = 2 * time.Minute

// terminalStates lists FSM states where a snapshot gap does NOT indicate
// the agent is stuck — it simply finished.
var terminalStates = map[string]bool{
	"retired":   true,
	"suspended": true,
	"idle":      true,
}

// buildAgentHistories converts a time-ordered slice of snapshot rows into
// per-actor history objects with state spans and stuck detection.
//
// Precondition: rows MUST be sorted by (actor_id ASC, recorded_at ASC).
// The caller (queryAgentHistory) guarantees this via ORDER BY.
func buildAgentHistories(rows []snapshotRow) []telAgentHistory {
	if len(rows) == 0 {
		return []telAgentHistory{}
	}

	type accumulator struct {
		latest   snapshotRow    // most recent row for summary fields
		first    time.Time      // first_seen
		spans    []telStateSpan // state spans built so far
		curState string         // current span's state
		curStart time.Time      // current span's start time
		prevAt   time.Time      // previous row's recorded_at
	}

	actors := make(map[string]*accumulator)
	order := []string{} // preserve insertion order

	for _, r := range rows {
		acc, ok := actors[r.ActorID]
		if !ok {
			acc = &accumulator{
				latest:   r,
				first:    r.RecordedAt,
				curState: r.State,
				curStart: r.RecordedAt,
				prevAt:   r.RecordedAt,
			}
			actors[r.ActorID] = acc
			order = append(order, r.ActorID)
			continue
		}

		gap := r.RecordedAt.Sub(acc.prevAt)

		// Stuck detection: if the gap exceeds the threshold and the
		// previous state was not terminal, insert a synthetic "stuck" span.
		if gap > stuckThreshold && !terminalStates[acc.curState] {
			// Close the current normal span at prevAt.
			acc.spans = append(acc.spans, telStateSpan{
				State:     acc.curState,
				EnteredAt: acc.curStart,
				Duration:  acc.prevAt.Sub(acc.curStart).Seconds(),
			})
			// Insert the stuck span covering the gap.
			acc.spans = append(acc.spans, telStateSpan{
				State:     "stuck",
				EnteredAt: acc.prevAt,
				Duration:  gap.Seconds(),
			})
			// New normal span starts at this row.
			acc.curState = r.State
			acc.curStart = r.RecordedAt
		} else if r.State != acc.curState {
			// Normal state transition — close current span, start new one.
			acc.spans = append(acc.spans, telStateSpan{
				State:     acc.curState,
				EnteredAt: acc.curStart,
				Duration:  r.RecordedAt.Sub(acc.curStart).Seconds(),
			})
			acc.curState = r.State
			acc.curStart = r.RecordedAt
		}

		acc.latest = r
		acc.prevAt = r.RecordedAt
	}

	// Finalize: close the last open span for each actor.
	result := make([]telAgentHistory, 0, len(order))
	for _, id := range order {
		acc := actors[id]

		// Close final span. Duration is from span start to last snapshot.
		acc.spans = append(acc.spans, telStateSpan{
			State:     acc.curState,
			EnteredAt: acc.curStart,
			Duration:  acc.latest.RecordedAt.Sub(acc.curStart).Seconds(),
		})

		result = append(result, telAgentHistory{
			Role:          acc.latest.Role,
			ActorID:       id,
			CurrentState:  acc.latest.State,
			Model:         acc.latest.Model,
			Iteration:     acc.latest.Iteration,
			MaxIterations: acc.latest.MaxIter,
			TokensUsed:    acc.latest.TokensUsed,
			CostUSD:       acc.latest.CostUSD,
			TrustScore:    acc.latest.TrustScore,
			Errors:        acc.latest.Errors,
			FirstSeen:     acc.first,
			LastSeen:      acc.latest.RecordedAt,
			States:        acc.spans,
		})
	}

	return result
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success (no errors)

- [ ] **Step 3: Commit**

```bash
git add cmd/work-server/telemetry_history.go
git commit -m "feat(telemetry): add agent history types and state transition computation"
```

---

### Task 2: Unit Tests for buildAgentHistories

**Files:**
- Create: `cmd/work-server/telemetry_history_test.go`

- [ ] **Step 1: Write the test file**

```go
package main

import (
	"testing"
	"time"
)

func TestBuildAgentHistories(t *testing.T) {
	t0 := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)

	mkRow := func(actor, role, state string, offsetSec int) snapshotRow {
		return snapshotRow{
			ActorID:    actor,
			Role:       role,
			State:      state,
			Model:      "claude-sonnet-4-6",
			Iteration:  1,
			MaxIter:    10,
			TokensUsed: 1000,
			CostUSD:    0.01,
			Errors:     0,
			RecordedAt: t0.Add(time.Duration(offsetSec) * time.Second),
		}
	}

	tests := []struct {
		name       string
		rows       []snapshotRow
		wantAgents int
		check      func(t *testing.T, result []telAgentHistory)
	}{
		{
			name:       "empty input",
			rows:       nil,
			wantAgents: 0,
		},
		{
			name: "single agent single state",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "processing", 10),
				mkRow("a1", "builder", "processing", 20),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if h.ActorID != "a1" {
					t.Errorf("actor_id = %q, want a1", h.ActorID)
				}
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].State != "processing" {
					t.Errorf("state = %q, want processing", h.States[0].State)
				}
				if h.States[0].Duration != 20 {
					t.Errorf("duration = %f, want 20", h.States[0].Duration)
				}
			},
		},
		{
			name: "single agent multiple transitions",
			rows: []snapshotRow{
				mkRow("a1", "builder", "idle", 0),
				mkRow("a1", "builder", "processing", 10),
				mkRow("a1", "builder", "waiting", 100),
				mkRow("a1", "builder", "retired", 200),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 4 {
					t.Fatalf("states len = %d, want 4", len(h.States))
				}
				wantStates := []string{"idle", "processing", "waiting", "retired"}
				wantDurations := []float64{10, 90, 100, 0}
				for i, s := range h.States {
					if s.State != wantStates[i] {
						t.Errorf("states[%d].State = %q, want %q", i, s.State, wantStates[i])
					}
					if s.Duration != wantDurations[i] {
						t.Errorf("states[%d].Duration = %f, want %f", i, s.Duration, wantDurations[i])
					}
				}
				if h.CurrentState != "retired" {
					t.Errorf("current_state = %q, want retired", h.CurrentState)
				}
			},
		},
		{
			name: "multiple agents same role",
			rows: []snapshotRow{
				// Rows sorted by actor_id ASC, then recorded_at ASC
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "retired", 60),
				mkRow("a2", "builder", "processing", 10),
				mkRow("a2", "builder", "processing", 70),
			},
			wantAgents: 2,
			check: func(t *testing.T, result []telAgentHistory) {
				if result[0].ActorID != "a1" || result[1].ActorID != "a2" {
					t.Errorf("unexpected actor order: %s, %s", result[0].ActorID, result[1].ActorID)
				}
				// a1: processing(60s) + retired(0s)
				if len(result[0].States) != 2 {
					t.Errorf("a1 states = %d, want 2", len(result[0].States))
				}
				// a2: processing(60s)
				if len(result[1].States) != 1 {
					t.Errorf("a2 states = %d, want 1", len(result[1].States))
				}
			},
		},
		{
			name: "stuck detection - gap exceeds threshold",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "processing", 10),
				// 130s gap (> 2min threshold)
				mkRow("a1", "builder", "processing", 140),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				// Expect: processing(10s) + stuck(130s) + processing(0s)
				if len(h.States) != 3 {
					t.Fatalf("states len = %d, want 3", len(h.States))
				}
				if h.States[0].State != "processing" {
					t.Errorf("states[0] = %q, want processing", h.States[0].State)
				}
				if h.States[1].State != "stuck" {
					t.Errorf("states[1] = %q, want stuck", h.States[1].State)
				}
				if h.States[1].Duration != 130 {
					t.Errorf("stuck duration = %f, want 130", h.States[1].Duration)
				}
				if h.States[2].State != "processing" {
					t.Errorf("states[2] = %q, want processing", h.States[2].State)
				}
			},
		},
		{
			name: "gap after terminal state is not stuck",
			rows: []snapshotRow{
				mkRow("a1", "builder", "retired", 0),
				// 200s gap, but state was terminal
				mkRow("a1", "builder", "retired", 200),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				// Should be a single retired span, no stuck inserted
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].State != "retired" {
					t.Errorf("state = %q, want retired", h.States[0].State)
				}
			},
		},
		{
			name: "single snapshot - just spawned",
			rows: []snapshotRow{
				mkRow("a1", "scout", "idle", 0),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].Duration != 0 {
					t.Errorf("duration = %f, want 0", h.States[0].Duration)
				}
			},
		},
		{
			name: "first_seen and last_seen correct",
			rows: []snapshotRow{
				mkRow("a1", "critic", "idle", 0),
				mkRow("a1", "critic", "processing", 30),
				mkRow("a1", "critic", "retired", 300),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if !h.FirstSeen.Equal(t0) {
					t.Errorf("first_seen = %v, want %v", h.FirstSeen, t0)
				}
				want := t0.Add(300 * time.Second)
				if !h.LastSeen.Equal(want) {
					t.Errorf("last_seen = %v, want %v", h.LastSeen, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentHistories(tt.rows)
			if len(result) != tt.wantAgents {
				t.Fatalf("agent count = %d, want %d", len(result), tt.wantAgents)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test -v -run TestBuildAgentHistories ./cmd/work-server/...`
Expected: all 7 subtests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/work-server/telemetry_history_test.go
git commit -m "test(telemetry): unit tests for agent history state transition computation"
```

---

### Task 3: SQL Query and HTTP Handler

**Files:**
- Modify: `cmd/work-server/telemetry_history.go` (append to bottom)
- Modify: `cmd/work-server/main.go:746` (add route)

- [ ] **Step 1: Add queryAgentHistory and handler to telemetry_history.go**

Append the following to the bottom of `cmd/work-server/telemetry_history.go`:

```go
// queryAgentHistory fetches all snapshots within the given window and
// computes per-actor state timelines using buildAgentHistories.
//
// The query fetches raw rows ordered by (actor_id, recorded_at) — the
// precondition for buildAgentHistories. State transition detection and
// stuck flagging happen in Go, not SQL, keeping the query simple and the
// logic testable without a database.
//
// Future: for windows longer than 24h (e.g. "week"), the current table's
// retention is insufficient. Options:
//   - Extend telemetry_agent_snapshots retention (simple, more storage)
//   - Add a materialized telemetry_agent_state_changes table in the hive
//     repo's telemetry writer (compact, recommended long-term)
func (sv *server) queryAgentHistory(ctx context.Context, window time.Duration) ([]telAgentHistory, error) {
	const q = `
		SELECT agent_role, actor_id, state, model, iteration, max_iterations,
		       tokens_used, cost_usd::float8, trust_score::float8,
		       errors, recorded_at
		FROM telemetry_agent_snapshots
		WHERE recorded_at > now() - $1::interval
		ORDER BY actor_id, recorded_at ASC`

	rows, err := sv.pool.Query(ctx, q, window.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []snapshotRow
	for rows.Next() {
		var r snapshotRow
		if err := rows.Scan(
			&r.Role, &r.ActorID, &r.State, &r.Model,
			&r.Iteration, &r.MaxIter,
			&r.TokensUsed, &r.CostUSD, &r.TrustScore,
			&r.Errors, &r.RecordedAt,
		); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return buildAgentHistories(snapshots), nil
}

// validWindows maps accepted ?window= values to durations.
// Intentionally restrictive — add new values here when longer retention
// is available (see Future comment on queryAgentHistory).
var validWindows = map[string]time.Duration{
	"1h":  1 * time.Hour,
	"24h": 24 * time.Hour,
}

// telemetryAgentHistory handles GET /telemetry/agents/history?window=1h|24h.
func (sv *server) telemetryAgentHistory(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	windowStr := r.URL.Query().Get("window")
	window, ok := validWindows[windowStr]
	if !ok {
		writeErr(w, http.StatusBadRequest, "window must be one of: 1h, 24h")
		return
	}

	ctx, cancel := telemetryQueryCtx(r)
	defer cancel()

	agents, err := sv.queryAgentHistory(ctx, window)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agents":      agents,
		"window":      windowStr,
		"computed_at": time.Now().UTC(),
	})
}
```

- [ ] **Step 2: Register the route in main.go**

In `cmd/work-server/main.go`, add the following line after line 746 (`GET /telemetry/agents/{role}`):

```go
	mux.HandleFunc("GET /telemetry/agents/history", srv.auth(srv.telemetryAgentHistory))
```

**Important:** This must come BEFORE the `GET /telemetry/agents/{role}` route, otherwise `history` would match as a `{role}` path value. Move the new line to be between the existing `GET /telemetry/agents` and `GET /telemetry/agents/{role}` lines. The route block should read:

```go
	mux.HandleFunc("GET /telemetry/agents", srv.auth(srv.telemetryAgents))
	mux.HandleFunc("GET /telemetry/agents/history", srv.auth(srv.telemetryAgentHistory))
	mux.HandleFunc("GET /telemetry/agents/{role}", srv.auth(srv.telemetryAgentDetail))
```

- [ ] **Step 3: Add import for "context" in telemetry_history.go**

Add to the import block at the top of `telemetry_history.go`:

```go
import (
	"context"
	"net/http"
	"time"
)
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 6: Commit**

```bash
git add cmd/work-server/telemetry_history.go cmd/work-server/main.go
git commit -m "feat(telemetry): add /telemetry/agents/history endpoint with state timelines"
```

---

### Task 4: Dashboard CSS — Time Window Buttons, Card Borders, State Timeline Bar, Stuck Pulse

**Files:**
- Modify: `cmd/work-server/telemetry_dashboard.go` (CSS section, lines ~10-660)

- [ ] **Step 1: Add CSS for time window button bar**

Insert the following CSS after the `.agent-card.has-errors` rule (after line 295 in `telemetry_dashboard.go`):

```css
/* ── TIME WINDOW CONTROLS ──────────────────────── */
.time-window-bar {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.time-window-btn {
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 600;
  padding: 3px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-sec);
  cursor: pointer;
  transition: all 0.15s;
}

.time-window-btn:hover { border-color: var(--border-light); color: var(--text); }
.time-window-btn.active {
  background: rgba(59,130,246,0.15);
  border-color: var(--blue);
  color: var(--blue);
}

.paused-banner {
  display: none;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  background: rgba(245,158,11,0.08);
  border: 1px solid rgba(245,158,11,0.2);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--amber);
  margin-bottom: 0.5rem;
}

.paused-banner.visible { display: flex; }

/* ── AGENT CARD BORDER STATES ─────────────────── */
.agent-card.active-agent  { border-left: 3px solid var(--green); }
.agent-card.stale-agent   { border-left: 3px solid var(--amber); box-shadow: 0 0 6px rgba(245,158,11,0.2); }
.agent-card.stuck-agent   { border-left: 3px solid var(--red); animation: pulse-stuck 1.5s ease-in-out infinite; }
.agent-card.terminated-agent { border-left: 3px solid var(--gray); opacity: 0.7; }

@keyframes pulse-stuck {
  0%, 100% { box-shadow: 0 0 4px rgba(239,68,68,0.2); }
  50%      { box-shadow: 0 0 12px rgba(239,68,68,0.4); }
}

/* ── STATE TIMELINE BAR ───────────────────────── */
.state-timeline {
  display: flex;
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
  margin: 0.375rem 0 0.125rem;
  background: var(--border);
}

.state-timeline .seg {
  height: 100%;
  min-width: 2px;
  transition: flex 0.3s;
}

.seg-idle       { background: var(--green); }
.seg-processing { background: var(--blue); }
.seg-waiting    { background: var(--amber); }
.seg-escalating,
.seg-refusing   { background: var(--red); }
.seg-suspended  { background: var(--gray); }
.seg-retiring,
.seg-retired    { background: rgba(100,116,139,0.5); }
.seg-stuck      { background: var(--red); animation: pulse-stuck 1.5s ease-in-out infinite; }
.seg-unknown    { background: var(--border-light); }

.agent-duration {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-dim);
}

.agent-lifespan {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-dim);
  display: flex;
  justify-content: space-between;
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add cmd/work-server/telemetry_dashboard.go
git commit -m "feat(dashboard): add CSS for time window controls, agent card borders, state timeline"
```

---

### Task 5: Dashboard HTML — Time Window Buttons and Paused Banner

**Files:**
- Modify: `cmd/work-server/telemetry_dashboard.go` (HTML section, around line 694-705)

- [ ] **Step 1: Add time window buttons to the Agent Status section header**

Replace the Agent Status section (lines 694-705) with:

```html
    <!-- Agent Status -->
    <div class="section">
      <div class="section-head">
        <span class="section-label">Agent Status</span>
        <div class="time-window-bar">
          <button class="time-window-btn active" data-window="now" onclick="setTimeWindow('now')">Now</button>
          <button class="time-window-btn" data-window="1h" onclick="setTimeWindow('1h')">1h</button>
          <button class="time-window-btn" data-window="24h" onclick="setTimeWindow('24h')">24h</button>
        </div>
        <span class="section-meta" id="agent-count"></span>
      </div>
      <div class="paused-banner" id="paused-banner">
        <span id="paused-text">Viewing last hour — paused</span>
      </div>
      <div class="section-body">
        <div class="agent-grid" id="agent-grid">
          <div class="data-empty">Awaiting telemetry data…</div>
        </div>
      </div>
    </div>
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add cmd/work-server/telemetry_dashboard.go
git commit -m "feat(dashboard): add time window buttons and paused banner HTML"
```

---

### Task 6: Dashboard JS — Time Window Switching and Paused Mode

**Files:**
- Modify: `cmd/work-server/telemetry_dashboard.go` (JS section, around line 750-810)

- [ ] **Step 1: Add time window state variables and setTimeWindow function**

After the existing `var lastSuccess = null;` line (line 751), add:

```javascript
  var currentWindow  = "now";   // "now" | "1h" | "24h"
  var pausedAt       = null;    // timestamp when historical view was activated
  var lastAgents     = [];      // cached for window switching
```

Then, after the `connectSSE()` call (after line 790), add the `setTimeWindow` function:

```javascript
  // ── TIME WINDOW ────────────────────────────────────
  function setTimeWindow(win) {
    currentWindow = win;

    // Update button active states
    var btns = document.querySelectorAll(".time-window-btn");
    for (var i = 0; i < btns.length; i++) {
      btns[i].classList.toggle("active", btns[i].getAttribute("data-window") === win);
    }

    var banner = document.getElementById("paused-banner");
    var bannerText = document.getElementById("paused-text");

    if (win === "now") {
      // Resume live rendering
      pausedAt = null;
      banner.classList.remove("visible");
      // Re-render with latest SSE data
      if (lastAgents.length) renderAgents(lastAgents);
      return;
    }

    // Historical mode — freeze and fetch
    pausedAt = new Date();
    var label = win === "1h" ? "last hour" : "last 24 hours";
    bannerText.textContent = "Viewing " + label + " — paused at " + pausedAt.toLocaleTimeString();
    banner.classList.add("visible");

    fetch("/telemetry/agents/history?window=" + win)
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        return res.json();
      })
      .then(function (data) {
        renderHistoricalAgents(data.agents || []);
      })
      .catch(function (err) {
        console.error("History fetch failed:", err);
      });
  }
  window.setTimeWindow = setTimeWindow;
```

- [ ] **Step 2: Modify the SSE onmessage handler to respect paused mode**

Replace the existing `es.onmessage` handler (lines 764-777) with:

```javascript
    es.onmessage = function (evt) {
      try {
        var data = JSON.parse(evt.data);
        lastSuccess = Date.now();
        setConnStatus("connected", "Live");
        renderPhases(data.phases || []);
        // Only update agents in "now" mode — historical views are frozen
        if (currentWindow === "now") {
          lastAgents = data.agents || [];
          renderAgents(lastAgents);
        }
        renderHive(data.hive || null);
        renderEvents(data.recent_events || []);
        fetchTasks();
      } catch (err) {
        console.error("SSE parse error:", err);
      }
    };
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/work-server/telemetry_dashboard.go
git commit -m "feat(dashboard): add time window switching with paused mode JS"
```

---

### Task 7: Dashboard JS — Rewrite renderAgents and buildAgentCard for Visual Differentiation

**Files:**
- Modify: `cmd/work-server/telemetry_dashboard.go` (JS section, lines 873-1002)

- [ ] **Step 1: Rewrite renderAgents to classify agent condition**

Replace the existing `renderAgents` function (lines 873-894) with:

```javascript
  function renderAgents(agents) {
    var grid  = clearEl("agent-grid");
    var count = document.getElementById("agent-count");

    if (!agents.length) {
      grid.appendChild(el("div", { cls: "data-empty", text: "No agent data — hive may be offline" }));
      count.textContent = "";
      return;
    }

    count.textContent = agents.length + " agent" + (agents.length !== 1 ? "s" : "");

    var sorted = agents.slice().sort(function (a, b) {
      var ae = a.errors || 0, be = b.errors || 0;
      if (ae !== be) return be - ae;
      return (a.role || "").localeCompare(b.role || "");
    });

    sorted.forEach(function (a) {
      grid.appendChild(buildAgentCard(a, classifyAgent(a)));
    });
  }

  // Classify an agent's condition for "now" mode based on last_event_at.
  function classifyAgent(a) {
    var state = (a.state || "").toLowerCase();
    var terminal = { retired: 1, suspended: 1, idle: 1 };
    if (terminal[state]) return "terminated";

    if (!a.last_event_at) return "active";
    var ageSec = (Date.now() - new Date(a.last_event_at).getTime()) / 1000;
    if (ageSec > 120) return "stuck";
    if (ageSec > 30)  return "stale";
    return "active";
  }
```

- [ ] **Step 2: Rewrite buildAgentCard to use border classes and show duration**

Replace the existing `buildAgentCard` function (lines 896-1002) with:

```javascript
  function buildAgentCard(a, condition) {
    var state      = (a.state || "unknown").toLowerCase();
    var model      = shortModel(a.model || "");
    var iter       = a.iteration || 0;
    var maxIter    = a.max_iterations || 1;
    var pct        = Math.min(100, Math.round((iter / maxIter) * 100));
    var fillCls    = pct >= 90 ? "danger" : pct >= 70 ? "warn" : "";
    var cost       = fmtCost(a.cost_usd);
    var trust      = a.trust_score != null ? Math.round(a.trust_score * 100) + "%" : "\u2014";
    var errors     = a.errors || 0;
    var hasErrors  = errors > 0;
    var lastEvType = a.last_event_type || "";
    var lastEvAt   = a.last_event_at ? relTime(a.last_event_at) : "";

    var condCls = condition ? condition + "-agent" : "";
    var card = el("div", { cls: "agent-card " + condCls + (hasErrors ? " has-errors" : "") });
    card.addEventListener("click", function () { card.classList.toggle("expanded"); });

    // Head
    var head = el("div", { cls: "agent-card-head" });
    head.appendChild(el("span", { cls: "agent-role", text: a.role || "unknown" }));
    head.appendChild(el("span", { cls: "badge badge-" + state, text: a.state || "Unknown" }));
    if (model) head.appendChild(el("span", { cls: "badge badge-model", text: model }));

    // Duration indicator in head
    if (condition === "stuck" && a.last_event_at) {
      var stuckSec = Math.floor((Date.now() - new Date(a.last_event_at).getTime()) / 1000);
      head.appendChild(el("span", { cls: "agent-duration", text: "stuck " + fmtDuration(stuckSec) }));
    } else if (condition === "stale" && a.last_event_at) {
      var staleSec = Math.floor((Date.now() - new Date(a.last_event_at).getTime()) / 1000);
      head.appendChild(el("span", { cls: "agent-duration", text: "stale " + staleSec + "s" }));
    } else if (condition === "active" && a.last_event_at) {
      head.appendChild(el("span", { cls: "agent-duration", text: "running" }));
    }
    card.appendChild(head);

    // State timeline bar (now mode: single segment for current state)
    if (a.last_event_at) {
      var timeline = el("div", { cls: "state-timeline" });
      var seg = el("div", { cls: "seg seg-" + state });
      seg.style.flex = "1";
      timeline.appendChild(seg);
      card.appendChild(timeline);
    }

    // Body
    var body = el("div", { cls: "agent-card-body" });

    // Iter row
    var iterRow = el("div", { cls: "agent-row" });
    iterRow.appendChild(el("span", { cls: "agent-label", text: "Iter" }));
    var bar  = el("div", { cls: "progress-bar" });
    var fill = el("div", { cls: "progress-fill " + fillCls });
    fill.style.width = pct + "%";
    bar.appendChild(fill);
    iterRow.appendChild(bar);
    iterRow.appendChild(el("span", { cls: "agent-val", text: iter + "/" + maxIter }));
    body.appendChild(iterRow);

    // Cost / Trust
    var costRow = el("div", { cls: "agent-row" });
    costRow.appendChild(el("span", { cls: "agent-label", text: "Cost" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: cost }));
    var spacer = el("span"); spacer.style.flex = "1";
    costRow.appendChild(spacer);
    costRow.appendChild(el("span", { cls: "agent-label", text: "Trust" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: trust }));
    body.appendChild(costRow);

    // Last event
    if (lastEvType || lastEvAt) {
      var evRow = el("div", { cls: "agent-event" });
      if (lastEvType) evRow.appendChild(el("span", { cls: "event-type-pill " + evtClass(lastEvType), text: lastEvType }));
      if (lastEvAt) {
        var dim = el("span", { text: lastEvAt }); dim.style.color = "var(--text-dim)";
        evRow.appendChild(dim);
      }
      if (hasErrors) {
        var errSpan = el("span", { text: errors + " err" });
        errSpan.style.cssText = "color:var(--red);margin-left:auto";
        evRow.appendChild(errSpan);
      }
      body.appendChild(evRow);
    }

    card.appendChild(body);

    // Expand section
    var expand = el("div", { cls: "agent-expand" });
    expand.appendChild(el("div", { cls: "expand-label", text: "Last Message" }));
    expand.appendChild(el("pre", { cls: "last-message", text: a.last_message || "(no message recorded)" }));

    var meta = el("div", { cls: "expand-meta" });
    var tokItem = el("div", { cls: "expand-meta-item" });
    tokItem.appendChild(document.createTextNode("Tokens "));
    tokItem.appendChild(el("strong", { text: (a.tokens_used || 0).toLocaleString() }));
    meta.appendChild(tokItem);

    var errItem = el("div", { cls: "expand-meta-item" });
    errItem.appendChild(document.createTextNode("Errors "));
    var errStrong = el("strong", { text: errors });
    errStrong.style.color = hasErrors ? "var(--red)" : "var(--text)";
    errItem.appendChild(errStrong);
    meta.appendChild(errItem);

    if (a.actor_id) {
      var idItem = el("div", { cls: "expand-meta-item",
        text: a.actor_id.slice(0, 20) + (a.actor_id.length > 20 ? "\u2026" : "") });
      idItem.style.cssText = "font-family:var(--mono);font-size:10px;color:var(--text-dim)";
      meta.appendChild(idItem);
    }

    expand.appendChild(meta);
    card.appendChild(expand);
    return card;
  }

  function fmtDuration(totalSec) {
    if (totalSec < 60) return totalSec + "s";
    var m = Math.floor(totalSec / 60);
    var s = totalSec % 60;
    if (m < 60) return m + "m " + s + "s";
    var h = Math.floor(m / 60);
    m = m % 60;
    return h + "h " + m + "m";
  }
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/work-server/telemetry_dashboard.go
git commit -m "feat(dashboard): rewrite agent cards with visual differentiation and stuck detection"
```

---

### Task 8: Dashboard JS — Historical Agent Rendering with State Timeline

**Files:**
- Modify: `cmd/work-server/telemetry_dashboard.go` (JS section, add after buildAgentCard)

- [ ] **Step 1: Add renderHistoricalAgents function**

Add after the `fmtDuration` function:

```javascript
  // ── HISTORICAL RENDERING ────────────────────────────
  // Renders agents from the /telemetry/agents/history endpoint.
  // Each agent has a .states array with state spans and durations.
  function renderHistoricalAgents(agents) {
    var grid  = clearEl("agent-grid");
    var count = document.getElementById("agent-count");

    if (!agents.length) {
      grid.appendChild(el("div", { cls: "data-empty", text: "No agents in this time window" }));
      count.textContent = "";
      return;
    }

    count.textContent = agents.length + " agent" + (agents.length !== 1 ? "s" : "");

    // Sort: agents with errors first, then by role
    var sorted = agents.slice().sort(function (a, b) {
      var ae = a.errors || 0, be = b.errors || 0;
      if (ae !== be) return be - ae;
      return (a.role || "").localeCompare(b.role || "");
    });

    sorted.forEach(function (a) {
      grid.appendChild(buildHistoricalCard(a));
    });
  }

  function buildHistoricalCard(a) {
    var state      = (a.current_state || "unknown").toLowerCase();
    var model      = shortModel(a.model || "");
    var iter       = a.iteration || 0;
    var maxIter    = a.max_iterations || 1;
    var pct        = Math.min(100, Math.round((iter / maxIter) * 100));
    var fillCls    = pct >= 90 ? "danger" : pct >= 70 ? "warn" : "";
    var cost       = fmtCost(a.cost_usd);
    var trust      = a.trust_score != null ? Math.round(a.trust_score * 100) + "%" : "\u2014";
    var errors     = a.errors || 0;
    var hasErrors  = errors > 0;
    var hadStuck   = false;
    var states     = a.states || [];

    // Check if agent was ever stuck
    for (var i = 0; i < states.length; i++) {
      if (states[i].state === "stuck") { hadStuck = true; break; }
    }

    // Determine card condition
    var terminal = { retired: 1, suspended: 1, idle: 1 };
    var condition = terminal[state] ? "terminated" : "active";
    if (hadStuck) condition = "stuck";

    var condCls = condition + "-agent";
    var card = el("div", { cls: "agent-card " + condCls + (hasErrors ? " has-errors" : "") });
    card.addEventListener("click", function () { card.classList.toggle("expanded"); });

    // Head
    var head = el("div", { cls: "agent-card-head" });
    head.appendChild(el("span", { cls: "agent-role", text: a.role || "unknown" }));
    head.appendChild(el("span", { cls: "badge badge-" + state, text: a.current_state || "Unknown" }));
    if (model) head.appendChild(el("span", { cls: "badge badge-model", text: model }));
    card.appendChild(head);

    // State timeline bar — proportional segments
    if (states.length > 0) {
      var totalDur = 0;
      for (var j = 0; j < states.length; j++) totalDur += states[j].duration_seconds || 0;

      // Avoid division by zero for just-spawned agents
      if (totalDur === 0) totalDur = 1;

      var timeline = el("div", { cls: "state-timeline" });
      for (var k = 0; k < states.length; k++) {
        var sp = states[k];
        var seg = el("div", { cls: "seg seg-" + (sp.state || "unknown").toLowerCase() });
        var pctW = Math.max(1, (sp.duration_seconds || 0) / totalDur * 100);
        seg.style.flex = String(pctW);
        seg.title = sp.state + ": " + fmtDuration(Math.round(sp.duration_seconds || 0));
        timeline.appendChild(seg);
      }
      card.appendChild(timeline);
    }

    // Lifespan row (start -> end)
    var lifespan = el("div", { cls: "agent-lifespan" });
    var startStr = a.first_seen ? new Date(a.first_seen).toLocaleTimeString() : "\u2014";
    var endStr   = terminal[state]
      ? (a.last_seen ? new Date(a.last_seen).toLocaleTimeString() : "\u2014")
      : "running";
    lifespan.appendChild(el("span", { text: startStr }));
    lifespan.appendChild(el("span", { text: "\u2192" }));
    lifespan.appendChild(el("span", { text: endStr }));
    card.appendChild(lifespan);

    // Body
    var body = el("div", { cls: "agent-card-body" });

    // Iter row
    var iterRow = el("div", { cls: "agent-row" });
    iterRow.appendChild(el("span", { cls: "agent-label", text: "Iter" }));
    var bar  = el("div", { cls: "progress-bar" });
    var fill = el("div", { cls: "progress-fill " + fillCls });
    fill.style.width = pct + "%";
    bar.appendChild(fill);
    iterRow.appendChild(bar);
    iterRow.appendChild(el("span", { cls: "agent-val", text: iter + "/" + maxIter }));
    body.appendChild(iterRow);

    // Cost / Trust
    var costRow = el("div", { cls: "agent-row" });
    costRow.appendChild(el("span", { cls: "agent-label", text: "Cost" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: cost }));
    var spacer = el("span"); spacer.style.flex = "1";
    costRow.appendChild(spacer);
    costRow.appendChild(el("span", { cls: "agent-label", text: "Trust" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: trust }));
    body.appendChild(costRow);

    // Errors
    if (hasErrors) {
      var errRow = el("div", { cls: "agent-row" });
      errRow.appendChild(el("span", { cls: "agent-label", text: "Errors" }));
      var errVal = el("span", { cls: "agent-val", text: String(errors) });
      errVal.style.color = "var(--red)";
      errRow.appendChild(errVal);
      body.appendChild(errRow);
    }

    card.appendChild(body);

    // Expand section — state breakdown
    var expand = el("div", { cls: "agent-expand" });
    expand.appendChild(el("div", { cls: "expand-label", text: "State Breakdown" }));

    var stateTable = el("div");
    stateTable.style.cssText = "display:flex;flex-direction:column;gap:2px;font-size:11px;font-family:var(--mono)";
    for (var m = 0; m < states.length; m++) {
      var sp2 = states[m];
      var row = el("div");
      row.style.cssText = "display:flex;gap:0.5rem;align-items:center";
      var dot = el("span", { cls: "seg seg-" + (sp2.state || "unknown").toLowerCase() });
      dot.style.cssText = "width:8px;height:8px;border-radius:2px;flex-shrink:0";
      row.appendChild(dot);
      row.appendChild(el("span", { text: sp2.state || "unknown", style: { color: "var(--text-sec)", minWidth: "80px" } }));
      row.appendChild(el("span", { text: fmtDuration(Math.round(sp2.duration_seconds || 0)), style: { color: "var(--text)" } }));
      if (sp2.entered_at) {
        row.appendChild(el("span", { text: "at " + new Date(sp2.entered_at).toLocaleTimeString(), style: { color: "var(--text-dim)", marginLeft: "auto" } }));
      }
      stateTable.appendChild(row);
    }
    expand.appendChild(stateTable);

    if (a.actor_id) {
      var idItem = el("div", { cls: "expand-meta-item",
        text: a.actor_id.slice(0, 20) + (a.actor_id.length > 20 ? "\u2026" : "") });
      idItem.style.cssText = "font-family:var(--mono);font-size:10px;color:var(--text-dim);margin-top:0.5rem";
      expand.appendChild(idItem);
    }

    card.appendChild(expand);
    return card;
  }
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/work-server/...`
Expected: success

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/work-server/telemetry_dashboard.go
git commit -m "feat(dashboard): add historical agent rendering with state timeline bars"
```

---

### Task 9: Final Build, Full Test Run, Vet

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./cmd/...`
Expected: success

- [ ] **Step 2: Static analysis**

Run: `go vet ./...`
Expected: no warnings

- [ ] **Step 3: Full test suite**

Run: `go test -v ./...`
Expected: all tests pass, including the new `TestBuildAgentHistories`

- [ ] **Step 4: Commit .gitignore change**

```bash
git add .gitignore
git commit -m "chore: add .superpowers to gitignore"
```
