package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// --- Telemetry response types ---

// telAgentSnapshot is the JSON shape for a single agent's latest snapshot.
type telAgentSnapshot struct {
	Role          string    `json:"role"`
	ActorID       string    `json:"actor_id"`
	State         string    `json:"state"`
	Model         string    `json:"model"`
	Iteration     int       `json:"iteration"`
	MaxIterations int       `json:"max_iterations"`
	TokensUsed    int64     `json:"tokens_used"`
	CostUSD       float64   `json:"cost_usd"`
	TrustScore    *float64  `json:"trust_score"`
	LastEventType *string   `json:"last_event_type"`
	LastEventAt   time.Time `json:"last_event_at"`
	LastMessage   *string   `json:"last_message"`
	Errors        int       `json:"errors"`
}

// telHiveSnapshot is the JSON shape for the latest hive health snapshot.
type telHiveSnapshot struct {
	ActiveAgents int      `json:"active_agents"`
	TotalActors  int      `json:"total_actors"`
	ChainLength  int64    `json:"chain_length"`
	ChainOK      bool     `json:"chain_ok"`
	EventRate    *float64 `json:"event_rate"`
	DailyCost    *float64 `json:"daily_cost"`
	DailyCap     *float64 `json:"daily_cap"`
	Severity     string   `json:"severity"`
}

// telPhase is the JSON shape for an expansion phase.
type telPhase struct {
	Phase       int        `json:"phase"`
	Label       string     `json:"label"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Notes       *string    `json:"notes"`
}

// telEvent is the JSON shape for a single event stream entry.
type telEvent struct {
	EventType string    `json:"event_type"`
	ActorRole string    `json:"actor_role"`
	Summary   *string   `json:"summary"`
	At        time.Time `json:"at"`
}

// --- Error helpers ---

// isMissingTable returns true when err is a PostgreSQL "undefined table" (42P01) error.
func isMissingTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

// telemetryUnavailable writes the standard 503 when tables are not yet initialised.
func telemetryUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":  "telemetry not available",
		"detail": "telemetry tables not initialized",
	})
}

// telemetryDBErr writes the standard 503 when the database is unreachable.
func telemetryDBErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":  "database unavailable",
		"detail": err.Error(),
	})
}

// --- Shared query helpers ---

// queryAgentSnapshots returns the latest snapshot per agent role.
func (sv *server) queryAgentSnapshots(ctx context.Context) ([]telAgentSnapshot, error) {
	const q = `
		SELECT DISTINCT ON (agent_role)
			agent_role, actor_id, state, model, iteration, max_iterations,
			tokens_used, cost_usd::float8, trust_score::float8,
			last_event_type, last_message, errors, recorded_at
		FROM telemetry_agent_snapshots
		ORDER BY agent_role, recorded_at DESC`

	rows, err := sv.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []telAgentSnapshot
	for rows.Next() {
		var a telAgentSnapshot
		if err := rows.Scan(
			&a.Role, &a.ActorID, &a.State, &a.Model,
			&a.Iteration, &a.MaxIterations,
			&a.TokensUsed, &a.CostUSD, &a.TrustScore,
			&a.LastEventType, &a.LastMessage, &a.Errors, &a.LastEventAt,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (sv *server) queryPhases(ctx context.Context) ([]telPhase, error) {
	const q = `SELECT phase, label, status, started_at, completed_at, notes
		FROM telemetry_phases ORDER BY phase ASC`

	rows, err := sv.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phases []telPhase
	for rows.Next() {
		var p telPhase
		if err := rows.Scan(&p.Phase, &p.Label, &p.Status, &p.StartedAt, &p.CompletedAt, &p.Notes); err != nil {
			return nil, err
		}
		phases = append(phases, p)
	}
	return phases, rows.Err()
}

func (sv *server) queryEvents(ctx context.Context, limit int) ([]telEvent, error) {
	const q = `SELECT event_type, actor_role, summary, recorded_at
		FROM telemetry_event_stream ORDER BY recorded_at DESC LIMIT $1`

	rows, err := sv.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []telEvent
	for rows.Next() {
		var e telEvent
		if err := rows.Scan(&e.EventType, &e.ActorRole, &e.Summary, &e.At); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// --- Handlers ---

// telemetryStatus handles GET /telemetry/status — full snapshot.
func (sv *server) telemetryStatus(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx := r.Context()

	agents, err := sv.queryAgentSnapshots(ctx)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if agents == nil {
		agents = []telAgentSnapshot{}
	}

	const hiveQ = `
		SELECT active_agents, total_actors, chain_length, chain_ok,
		       event_rate::float8, daily_cost::float8, daily_cap::float8, severity
		FROM telemetry_hive_snapshots
		ORDER BY recorded_at DESC LIMIT 1`

	hiveRows, err := sv.pool.Query(ctx, hiveQ)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	defer hiveRows.Close()

	var hive *telHiveSnapshot
	if hiveRows.Next() {
		var h telHiveSnapshot
		if err := hiveRows.Scan(
			&h.ActiveAgents, &h.TotalActors, &h.ChainLength, &h.ChainOK,
			&h.EventRate, &h.DailyCost, &h.DailyCap, &h.Severity,
		); err != nil {
			telemetryDBErr(w, err)
			return
		}
		hive = &h
	}
	if err := hiveRows.Err(); err != nil {
		telemetryDBErr(w, err)
		return
	}
	hiveRows.Close()

	phases, err := sv.queryPhases(ctx)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if phases == nil {
		phases = []telPhase{}
	}

	events, err := sv.queryEvents(ctx, 50)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if events == nil {
		events = []telEvent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC(),
		"hive":          hive,
		"agents":        agents,
		"phases":        phases,
		"recent_events": events,
	})
}

// telemetryAgents handles GET /telemetry/agents — latest snapshot per agent.
func (sv *server) telemetryAgents(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	agents, err := sv.queryAgentSnapshots(r.Context())
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if agents == nil {
		agents = []telAgentSnapshot{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

// telemetryAgentDetail handles GET /telemetry/agents/{role} — single agent with history.
func (sv *server) telemetryAgentDetail(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	role := r.PathValue("role")
	ctx := r.Context()

	const latestQ = `
		SELECT agent_role, actor_id, state, model, iteration, max_iterations,
		       tokens_used, cost_usd::float8, trust_score::float8,
		       last_event_type, last_message, errors, recorded_at
		FROM telemetry_agent_snapshots
		WHERE agent_role = $1
		ORDER BY recorded_at DESC LIMIT 1`

	latestRows, err := sv.pool.Query(ctx, latestQ, role)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	defer latestRows.Close()

	var current *telAgentSnapshot
	if latestRows.Next() {
		var a telAgentSnapshot
		if err := latestRows.Scan(
			&a.Role, &a.ActorID, &a.State, &a.Model,
			&a.Iteration, &a.MaxIterations,
			&a.TokensUsed, &a.CostUSD, &a.TrustScore,
			&a.LastEventType, &a.LastMessage, &a.Errors, &a.LastEventAt,
		); err != nil {
			telemetryDBErr(w, err)
			return
		}
		current = &a
	}
	if err := latestRows.Err(); err != nil {
		telemetryDBErr(w, err)
		return
	}
	latestRows.Close()

	const histQ = `
		SELECT agent_role, actor_id, state, model, iteration, max_iterations,
		       tokens_used, cost_usd::float8, trust_score::float8,
		       last_event_type, last_message, errors, recorded_at
		FROM telemetry_agent_snapshots
		WHERE agent_role = $1
		ORDER BY recorded_at DESC LIMIT 20`

	histRows, err := sv.pool.Query(ctx, histQ, role)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	defer histRows.Close()

	var history []telAgentSnapshot
	for histRows.Next() {
		var a telAgentSnapshot
		if err := histRows.Scan(
			&a.Role, &a.ActorID, &a.State, &a.Model,
			&a.Iteration, &a.MaxIterations,
			&a.TokensUsed, &a.CostUSD, &a.TrustScore,
			&a.LastEventType, &a.LastMessage, &a.Errors, &a.LastEventAt,
		); err != nil {
			telemetryDBErr(w, err)
			return
		}
		history = append(history, a)
	}
	if err := histRows.Err(); err != nil {
		telemetryDBErr(w, err)
		return
	}
	if history == nil {
		history = []telAgentSnapshot{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agent":   current,
		"history": history,
	})
}

// telemetryStream handles GET /telemetry/stream — recent event stream.
func (sv *server) telemetryStream(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	events, err := sv.queryEvents(r.Context(), limit)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if events == nil {
		events = []telEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// telemetryPhases handles GET /telemetry/phases — expansion phase status.
func (sv *server) telemetryPhases(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	phases, err := sv.queryPhases(r.Context())
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if phases == nil {
		phases = []telPhase{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"phases": phases})
}

// telemetryHealth handles GET /telemetry/health — latest hive health snapshot.
func (sv *server) telemetryHealth(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	const q = `
		SELECT active_agents, total_actors, chain_length, chain_ok,
		       event_rate::float8, daily_cost::float8, daily_cap::float8, severity
		FROM telemetry_hive_snapshots
		ORDER BY recorded_at DESC LIMIT 1`

	rows, err := sv.pool.Query(r.Context(), q)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	defer rows.Close()

	var hive *telHiveSnapshot
	if rows.Next() {
		var h telHiveSnapshot
		if err := rows.Scan(
			&h.ActiveAgents, &h.TotalActors, &h.ChainLength, &h.ChainOK,
			&h.EventRate, &h.DailyCost, &h.DailyCap, &h.Severity,
		); err != nil {
			telemetryDBErr(w, err)
			return
		}
		hive = &h
	}
	if err := rows.Err(); err != nil {
		telemetryDBErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"hive": hive})
}

// updatePhase handles POST /telemetry/phases/{phase} — graduation ceremony updates.
//
// Timestamp logic:
//   - started_at is set to now() when status becomes "in_progress", but only if it is
//     currently NULL (a manual timestamp set via a prior call is preserved).
//   - completed_at is set to now() when status becomes "complete".
//   - completed_at is cleared (NULL) when status becomes anything other than "complete".
func (sv *server) updatePhase(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	phase, err := strconv.Atoi(r.PathValue("phase"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid phase number")
		return
	}

	var body struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validStatuses := map[string]bool{"blocked": true, "in_progress": true, "complete": true}
	if !validStatuses[body.Status] {
		writeErr(w, http.StatusBadRequest, "status must be: blocked, in_progress, complete")
		return
	}

	// started_at: preserve existing value if set; set to now() only when
	// transitioning to in_progress and the column is still NULL.
	// completed_at: now() when complete, NULL otherwise.
	// notes: update only when a non-empty value is provided.
	const q = `
		UPDATE telemetry_phases
		SET
			status       = $2,
			started_at   = COALESCE(started_at, CASE WHEN $2 = 'in_progress' THEN now() ELSE NULL END),
			completed_at = CASE WHEN $2 = 'complete' THEN now() ELSE NULL END,
			notes        = CASE WHEN $3 <> '' THEN $3 ELSE notes END
		WHERE phase = $1
		RETURNING phase, label, status, started_at, completed_at, notes`

	rows, err := sv.pool.Query(r.Context(), q, phase, body.Status, body.Notes)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			telemetryDBErr(w, err)
			return
		}
		writeErr(w, http.StatusNotFound, "phase not found")
		return
	}

	var p telPhase
	if err := rows.Scan(&p.Phase, &p.Label, &p.Status, &p.StartedAt, &p.CompletedAt, &p.Notes); err != nil {
		telemetryDBErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"phase": p})
}
