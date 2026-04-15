package main

import (
	"context"
	"net/http"
	"time"
)

// telStateSpan represents a contiguous period an agent spent in one FSM state.
type telStateSpan struct {
	State     string    `json:"state"`
	EnteredAt time.Time `json:"entered_at"`
	Duration  float64   `json:"duration_seconds"`
}

// telAgentHistory is the JSON shape for an agent's lifecycle within a time window.
type telAgentHistory struct {
	Role          string         `json:"role"`
	ActorID       string         `json:"actor_id"`
	CurrentState  string         `json:"current_state"`
	Model         string         `json:"model"`
	Iteration     int            `json:"iteration"`
	MaxIterations int            `json:"max_iterations"`
	TokensUsed    int64          `json:"tokens_used"`
	CostUSD       float64        `json:"cost_usd"`
	TrustScore    *float64       `json:"trust_score"`
	Errors        int            `json:"errors"`
	FirstSeen     time.Time      `json:"first_seen"`
	LastSeen      time.Time      `json:"last_seen"`
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
// the agent is stuck — it simply finished. Note: idle is intentionally
// excluded — agents start in idle and should transition quickly. An agent
// stuck in idle for >2min likely failed to initialize.
var terminalStates = map[string]bool{
	"retired":   true,
	"suspended": true,
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
		latest   snapshotRow
		first    time.Time
		spans    []telStateSpan
		curState string
		curStart time.Time
		prevAt   time.Time
	}

	actors := make(map[string]*accumulator)
	order := []string{}

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

		if gap > stuckThreshold && !terminalStates[acc.curState] && !terminalStates[r.State] {
			acc.spans = append(acc.spans, telStateSpan{
				State:     acc.curState,
				EnteredAt: acc.curStart,
				Duration:  acc.prevAt.Sub(acc.curStart).Seconds(),
			})
			acc.spans = append(acc.spans, telStateSpan{
				State:     "stuck",
				EnteredAt: acc.prevAt,
				Duration:  gap.Seconds(),
			})
			acc.curState = r.State
			acc.curStart = r.RecordedAt
		} else if r.State != acc.curState {
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

	result := make([]telAgentHistory, 0, len(order))
	for _, id := range order {
		acc := actors[id]

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

	rows, err := sv.pool.Query(ctx, q, window)
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

	// History queries scan the full window (up to 24h of snapshots) rather
	// than a single latest-per-actor row, so they need a longer deadline than
	// the 5s telemetryQueryCtx used for point-in-time reads.
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
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
