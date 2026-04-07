package main

import (
	"context"
	"net/http"
	"time"
)

// --- Structural telemetry response types ---

// telRole is the JSON shape for a role definition from telemetry_role_definitions.
type telRole struct {
	Role          string    `json:"role"`
	Name          string    `json:"name"`
	Tier          string    `json:"tier"`
	Purpose       string    `json:"purpose"`
	Model         string    `json:"model"`
	CanOperate    bool      `json:"can_operate"`
	MaxIterations int       `json:"max_iterations"`
	WatchPatterns []string  `json:"watch_patterns"`
	Phase         int       `json:"phase"`
	GraduatedAt   *string   `json:"graduated_at"`
	Status        string    `json:"status"`
	HasPrompt     bool      `json:"has_prompt"`
	HasPersona    bool      `json:"has_persona"`
	Category      string    `json:"category"`
	DependsOn     []string  `json:"depends_on"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// telActor is the JSON shape for an actor from the actors table.
type telActor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	ActorType   string `json:"actor_type"`
	Status      string `json:"status"`
}

// telLayer is the JSON shape for a layer from telemetry_layers.
type telLayer struct {
	Layer       int    `json:"layer"`
	Name        string `json:"name"`
	Focus       string `json:"focus"`
	Depth       int    `json:"depth"`
	Description string `json:"description"`
}

// telPhaseEnriched is a phase with agent membership and per-agent status.
type telPhaseEnriched struct {
	Phase        int               `json:"phase"`
	Label        string            `json:"label"`
	Status       string            `json:"status"`
	StartedAt    *time.Time        `json:"started_at"`
	CompletedAt  *time.Time        `json:"completed_at"`
	Notes        *string           `json:"notes"`
	ExitCriteria *string           `json:"exit_criteria"`
	Agents       []telPhaseAgent   `json:"agents"`
}

// telPhaseAgent is an agent within a phase, annotated with its role status.
type telPhaseAgent struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

// --- Query helpers ---

const rolesQuery = `
	SELECT role, name, tier, purpose, model, can_operate, max_iterations,
	       watch_patterns, phase, graduated_at, status, has_prompt,
	       has_persona, category, depends_on, updated_at
	FROM telemetry_role_definitions
	ORDER BY
	    CASE tier WHEN 'A' THEN 1 WHEN 'B' THEN 2 WHEN 'C' THEN 3 WHEN 'D' THEN 4 END,
	    role`

func (sv *server) queryRoles(ctx context.Context) ([]telRole, error) {
	rows, err := sv.pool.Query(ctx, rolesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []telRole
	for rows.Next() {
		var r telRole
		if err := rows.Scan(
			&r.Role, &r.Name, &r.Tier, &r.Purpose, &r.Model,
			&r.CanOperate, &r.MaxIterations, &r.WatchPatterns,
			&r.Phase, &r.GraduatedAt, &r.Status, &r.HasPrompt,
			&r.HasPersona, &r.Category, &r.DependsOn, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (sv *server) queryActors(ctx context.Context) ([]telActor, error) {
	const q = `SELECT id, display_name, actor_type, status
		FROM actors ORDER BY display_name`

	rows, err := sv.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actors []telActor
	for rows.Next() {
		var a telActor
		if err := rows.Scan(&a.ID, &a.DisplayName, &a.ActorType, &a.Status); err != nil {
			return nil, err
		}
		actors = append(actors, a)
	}
	return actors, rows.Err()
}

func (sv *server) queryLayers(ctx context.Context) ([]telLayer, error) {
	const q = `SELECT layer, name, focus, depth, description
		FROM telemetry_layers ORDER BY layer`

	rows, err := sv.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var layers []telLayer
	for rows.Next() {
		var l telLayer
		if err := rows.Scan(&l.Layer, &l.Name, &l.Focus, &l.Depth, &l.Description); err != nil {
			return nil, err
		}
		layers = append(layers, l)
	}
	return layers, rows.Err()
}

// queryEnrichedPhases returns phases joined with their agent membership and
// cross-referenced against telemetry_role_definitions for per-agent status.
func (sv *server) queryEnrichedPhases(ctx context.Context) ([]telPhaseEnriched, error) {
	const q = `
		SELECT p.phase, p.label, p.status, p.started_at, p.completed_at,
		       p.notes, p.exit_criteria,
		       array_agg(pa.agent_role ORDER BY pa.agent_role) FILTER (WHERE pa.agent_role IS NOT NULL) as agents
		FROM telemetry_phases p
		LEFT JOIN telemetry_phase_agents pa ON p.phase = pa.phase
		GROUP BY p.phase, p.label, p.status, p.started_at, p.completed_at,
		         p.notes, p.exit_criteria
		ORDER BY p.phase`

	rows, err := sv.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phases []telPhaseEnriched
	for rows.Next() {
		var p telPhaseEnriched
		var agentRoles []string
		if err := rows.Scan(
			&p.Phase, &p.Label, &p.Status, &p.StartedAt, &p.CompletedAt,
			&p.Notes, &p.ExitCriteria, &agentRoles,
		); err != nil {
			return nil, err
		}
		for _, role := range agentRoles {
			p.Agents = append(p.Agents, telPhaseAgent{Role: role, Status: "unknown"})
		}
		if p.Agents == nil {
			p.Agents = []telPhaseAgent{}
		}
		phases = append(phases, p)
	}
	return phases, rows.Err()
}

// annotatePhaseAgents cross-references phase agents with role definitions to
// set the correct status on each agent entry.
func annotatePhaseAgents(phases []telPhaseEnriched, roles []telRole) {
	statusByRole := make(map[string]string, len(roles))
	for _, r := range roles {
		statusByRole[r.Role] = r.Status
	}
	for i := range phases {
		for j := range phases[i].Agents {
			if s, ok := statusByRole[phases[i].Agents[j].Role]; ok {
				phases[i].Agents[j].Status = s
			} else {
				phases[i].Agents[j].Status = "missing"
			}
		}
	}
}

// --- Handlers ---

// telemetryRoles handles GET /telemetry/roles — all role definitions.
func (sv *server) telemetryRoles(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx, cancel := telemetryQueryCtx(r)
	defer cancel()

	roles, err := sv.queryRoles(ctx)
	if err != nil {
		if isMissingTable(err) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error":  "role definitions not available",
				"detail": "telemetry_role_definitions table not initialized",
			})
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if roles == nil {
		roles = []telRole{}
	}
	writeJSON(w, http.StatusOK, roles)
}

// telemetryRoleDetail handles GET /telemetry/roles/{name} — single role.
func (sv *server) telemetryRoleDetail(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	name := r.PathValue("name")
	ctx, cancel := telemetryQueryCtx(r)
	defer cancel()

	const singleQ = `
		SELECT role, name, tier, purpose, model, can_operate, max_iterations,
		       watch_patterns, phase, graduated_at, status, has_prompt,
		       has_persona, category, depends_on, updated_at
		FROM telemetry_role_definitions
		WHERE role = $1`

	rows, err := sv.pool.Query(ctx, singleQ, name)
	if err != nil {
		if isMissingTable(err) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error":  "role definitions not available",
				"detail": "telemetry_role_definitions table not initialized",
			})
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
		writeErr(w, http.StatusNotFound, "role not found")
		return
	}

	var role telRole
	if err := rows.Scan(
		&role.Role, &role.Name, &role.Tier, &role.Purpose, &role.Model,
		&role.CanOperate, &role.MaxIterations, &role.WatchPatterns,
		&role.Phase, &role.GraduatedAt, &role.Status, &role.HasPrompt,
		&role.HasPersona, &role.Category, &role.DependsOn, &role.UpdatedAt,
	); err != nil {
		telemetryDBErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// telemetryActors handles GET /telemetry/actors — all registered actors.
func (sv *server) telemetryActors(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx, cancel := telemetryQueryCtx(r)
	defer cancel()

	actors, err := sv.queryActors(ctx)
	if err != nil {
		if isMissingTable(err) {
			telemetryUnavailable(w)
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if actors == nil {
		actors = []telActor{}
	}
	writeJSON(w, http.StatusOK, actors)
}

// telemetryLayers handles GET /telemetry/layers — all 14 layers.
func (sv *server) telemetryLayers(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx, cancel := telemetryQueryCtx(r)
	defer cancel()

	layers, err := sv.queryLayers(ctx)
	if err != nil {
		if isMissingTable(err) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error":  "layers not available",
				"detail": "telemetry_layers table not initialized",
			})
			return
		}
		telemetryDBErr(w, err)
		return
	}
	if layers == nil {
		layers = []telLayer{}
	}
	writeJSON(w, http.StatusOK, layers)
}

// telemetryOverview handles GET /telemetry/overview — combined structural + runtime snapshot.
func (sv *server) telemetryOverview(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result := map[string]any{
		"timestamp": time.Now().UTC(),
	}

	// --- Existing runtime data (same as buildStatusSnapshot) ---

	agents, err := sv.queryAgentSnapshots(ctx)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	if agents == nil {
		agents = []telAgentSnapshot{}
	}
	result["agents"] = agents

	const hiveQ = `
		SELECT active_agents, total_actors, chain_length, chain_ok,
		       event_rate::float8, daily_cost::float8, daily_cap::float8, severity
		FROM telemetry_hive_snapshots
		ORDER BY recorded_at DESC LIMIT 1`

	hiveRows, err := sv.pool.Query(ctx, hiveQ)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	var hive *telHiveSnapshot
	if hiveRows != nil {
		defer hiveRows.Close()
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
	}
	result["hive"] = hive

	events, err := sv.queryEvents(ctx, 50)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	if events == nil {
		events = []telEvent{}
	}
	result["recent_events"] = events

	// --- Structural data (new tables, may not exist yet) ---

	roles, err := sv.queryRoles(ctx)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	if roles == nil {
		roles = []telRole{}
	}
	result["roles"] = roles

	actors, err := sv.queryActors(ctx)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	if actors == nil {
		actors = []telActor{}
	}
	result["actors"] = actors

	layers, err := sv.queryLayers(ctx)
	if err != nil && !isMissingTable(err) {
		telemetryDBErr(w, err)
		return
	}
	if layers == nil {
		layers = []telLayer{}
	}
	result["layers"] = layers

	// Enriched phases with agent membership and status cross-reference.
	enrichedPhases, err := sv.queryEnrichedPhases(ctx)
	if err != nil {
		if isMissingTable(err) {
			// Fall back to basic phases if telemetry_phase_agents doesn't exist.
			basicPhases, err2 := sv.queryPhases(ctx)
			if err2 != nil && !isMissingTable(err2) {
				telemetryDBErr(w, err2)
				return
			}
			if basicPhases == nil {
				basicPhases = []telPhase{}
			}
			result["phases"] = basicPhases
		} else {
			telemetryDBErr(w, err)
			return
		}
	} else {
		if enrichedPhases == nil {
			enrichedPhases = []telPhaseEnriched{}
		}
		annotatePhaseAgents(enrichedPhases, roles)
		result["phases"] = enrichedPhases
	}

	writeJSON(w, http.StatusOK, result)
}
