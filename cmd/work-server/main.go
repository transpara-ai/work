// Command work-server is an HTTP REST API server for the Work Graph (Layer 1).
// It exposes task management as signed, auditable events on the shared event graph.
//
// Environment variables:
//
//	WORK_HUMAN                — display name of the human operator (required)
//	WORK_API_KEY              — API key for auth; callers pass Authorization: Bearer <key> (required)
//	WORK_API_TOKEN            — bearer token for workspace-scoped external API; falls back to WORK_API_KEY if unset
//	DATABASE_URL              — Postgres DSN (optional; defaults to in-memory)
//	WORK_SIGNING_KEY_FILE     — owner-only base64 Ed25519 seed/private key (required with Postgres)
//	PORT                      — HTTP port to listen on (optional; defaults to 8080)
//	WORK_BIND_HOST            — optional listen host; set 127.0.0.1 for loopback-only operation
//	SITE_UI_BASE_URL          — canonical Site UI base URL for legacy UI notices (optional; derived from request host)
//
// Endpoints:
//
//	GET  /                                      legacy read-only dashboard (HTML, no auth required)
//	POST /tasks                                 create a task
//	GET  /tasks                                 list tasks (?open=true, ?priority=high, ?assignee=<actor_id>)
//	GET  /tasks/{id}                            get full task details (title, description, priority, status, assignee, blocked)
//	GET  /tasks/{id}/status                     get task status
//	GET  /tasks/{id}/events                     get audit trail (ordered work.task.* events for this task, including comments)
//	POST /tasks/{id}/assign                     assign task (body: {"assignee":"..."})
//	POST /tasks/{id}/unblock                    mark task blockers resolved (body: {})
//	POST /tasks/{id}/complete                   complete task (body: {"summary":"..."})
//	POST /tasks/{id}/comment                    add a comment (body: {"body":"..."})
//	GET  /tasks/{id}/comments                   list comments for a task
//	POST /tasks/{id}/artifacts                  attach an artifact (body: {"label":"...", "media_type":"...", "body":"..."})
//	GET  /tasks/{id}/artifacts                  list artifacts for a task
//	POST /tasks/{id}/waive-artifact             waive the artifact requirement (body: {"reason":"..."})
//	POST /tasks/{id}/fact-requirements          require a Phase 3 EventGraph fact for readiness
//
// Workspace-scoped routes (authenticated via WORK_API_TOKEN):
//
//	GET  /w/{workspace}                         legacy workspace task dashboard (HTML, no auth required)
//	POST /w/{workspace}/tasks                   create a task in the workspace
//	GET  /w/{workspace}/tasks                   list tasks in the workspace
//	POST /w/{workspace}/tasks/{id}/assign       assign a workspace task
//	POST /w/{workspace}/tasks/{id}/complete     complete a workspace task
//	POST /w/{workspace}/tasks/{id}/comment      add a comment to a workspace task
//	POST /w/{workspace}/tasks/{id}/artifacts    attach an artifact to a workspace task
//	GET  /w/{workspace}/tasks/{id}/artifacts    list artifacts for a workspace task
//	POST /w/{workspace}/tasks/{id}/waive-artifact  waive artifact requirement for a workspace task
//	POST /w/{workspace}/tasks/{id}/fact-requirements  require a Phase 3 EventGraph fact for readiness
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/transpara-ai/eventgraph/go/pkg/actor"
	"github.com/transpara-ai/eventgraph/go/pkg/actor/pgactor"
	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/store/pgstore"
	"github.com/transpara-ai/eventgraph/go/pkg/types"

	"github.com/transpara-ai/work"
	"github.com/transpara-ai/work/runtimeidentity"
)

const (
	legacyUIStatusHeader      = "X-Transpara-UI-Status"
	legacyUIReplacementHeader = "X-Transpara-Replacement-UI"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	humanName := os.Getenv("WORK_HUMAN")
	if humanName == "" {
		return fmt.Errorf("WORK_HUMAN env var is required (display name of the human operator)")
	}
	apiKey := os.Getenv("WORK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("WORK_API_KEY env var is required")
	}
	apiToken := os.Getenv("WORK_API_TOKEN")
	if apiToken == "" {
		apiToken = apiKey
	}
	dsn := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Open shared pool for Postgres, or nil for in-memory.
	var pool *pgxpool.Pool
	if dsn != "" {
		fmt.Fprintln(os.Stderr, "Postgres: configured")
		poolCfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return fmt.Errorf("postgres config: %w", err)
		}
		// The pool is shared between the event store (which holds connections
		// during advisory-locked writes) and telemetry read queries. Default
		// MaxConns (≈4 in containers) is too small: just a few concurrent
		// event writes exhaust the pool and starve telemetry reads.
		// Only override defaults — respect any values set via DSN parameters
		// (e.g. pool_max_conns, pool_min_conns) so operators can tune for
		// their Postgres/PgBouncer limits.
		if !strings.Contains(dsn, "pool_max_conns") {
			poolCfg.MaxConns = 20
		}
		if !strings.Contains(dsn, "pool_min_conns") {
			poolCfg.MinConns = 2
		}
		if !strings.Contains(dsn, "pool_max_conn_lifetime") {
			poolCfg.MaxConnLifetime = 30 * time.Minute
		}
		if !strings.Contains(dsn, "pool_max_conn_idle_time") {
			poolCfg.MaxConnIdleTime = 5 * time.Minute
		}
		if !strings.Contains(dsn, "pool_health_check_period") {
			poolCfg.HealthCheckPeriod = 30 * time.Second
		}
		// connect_timeout is parsed by pgx (not pgxpool) into ConnConfig,
		// so we check the DSN the same way as pool params for consistency.
		if !strings.Contains(dsn, "connect_timeout") {
			poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second
		}
		pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		defer pool.Close()
	}

	s, err := openStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "store close: %v\n", err)
		}
	}()

	actors, err := openActorStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("actor store: %w", err)
	}

	identity, err := runtimeidentity.Resolve(actors, humanName, os.Getenv("WORK_SIGNING_KEY_FILE"), pool != nil)
	if err != nil {
		return fmt.Errorf("load runtime identity: %w", err)
	}
	humanID := identity.ActorID

	// Register work event type unmarshalers before any store reads —
	// Head() deserializes the latest event which may be a work type.
	// Enable raw fallback so unknown event types (hive, agent, membrane)
	// don't break deserialization when sharing a Postgres store.
	event.SetFallbackUnmarshaler(event.RawFallback)
	work.RegisterEventTypes()

	// Bootstrap the event graph if it has no genesis event.
	if err := bootstrapGraphWithSigner(s, humanID, identity.Signer); err != nil {
		return fmt.Errorf("bootstrap graph: %w", err)
	}

	// Build factory and signer for work events.
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	signer := identity.Signer

	ts := work.NewTaskStore(s, factory, signer)
	phaseGates := work.NewPhaseGateStore(s, factory, signer)

	srv := &server{
		ts:         ts,
		phaseGates: phaseGates,
		store:      s,
		humanID:    humanID,
		apiKey:     apiKey,
		apiToken:   apiToken,
		pool:       pool,
		fanout:     newEventFanout(),
	}

	// Tail hive's telemetry_event_stream and republish to SSE subscribers.
	// MVP polling bridge — hive and work-server are separate binaries so they
	// cannot share an in-process bus. See events.go:runEventPoller.
	go srv.runEventPoller(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.dashboard)
	mux.HandleFunc("GET /health", srv.health)
	mux.HandleFunc("POST /tasks", srv.auth(srv.createTask))
	mux.HandleFunc("GET /tasks", srv.auth(srv.listTasks))
	mux.HandleFunc("GET /tasks/{id}", srv.auth(srv.getTask))
	mux.HandleFunc("GET /tasks/{id}/status", srv.auth(srv.getTaskStatus))
	mux.HandleFunc("GET /tasks/{id}/events", srv.auth(srv.getTaskEvents))
	mux.HandleFunc("POST /tasks/{id}/assign", srv.auth(srv.assignTask))
	mux.HandleFunc("POST /tasks/{id}/unblock", srv.auth(srv.unblockTask))
	mux.HandleFunc("POST /tasks/{id}/complete", srv.auth(srv.completeTask))
	mux.HandleFunc("POST /tasks/{id}/comment", srv.auth(srv.addComment))
	mux.HandleFunc("GET /tasks/{id}/comments", srv.auth(srv.listComments))
	mux.HandleFunc("POST /tasks/{id}/artifacts", srv.auth(srv.addArtifact))
	mux.HandleFunc("GET /tasks/{id}/artifacts", srv.auth(srv.listArtifacts))
	mux.HandleFunc("POST /tasks/{id}/waive-artifact", srv.auth(srv.waiveArtifact))
	mux.HandleFunc("POST /tasks/{id}/fact-requirements", srv.auth(srv.addFactRequirement))
	mux.HandleFunc("POST /phase-gates", srv.auth(srv.declarePhaseGate))
	mux.HandleFunc("GET /phase-gates", srv.auth(srv.listPhaseGates))
	mux.HandleFunc("POST /phase-gates/{id}/approve", srv.auth(srv.approvePhaseGate))
	mux.HandleFunc("POST /phase-gates/{id}/reject", srv.auth(srv.rejectPhaseGate))

	// Telemetry routes — reads from hive-postgres via the shared pool.
	mux.HandleFunc("GET /telemetry/status", srv.auth(srv.telemetryStatus))
	mux.HandleFunc("GET /telemetry/agents", srv.auth(srv.telemetryAgents))
	mux.HandleFunc("GET /telemetry/agents/history", srv.auth(srv.telemetryAgentHistory))
	mux.HandleFunc("GET /telemetry/agents/{role}", srv.auth(srv.telemetryAgentDetail))
	mux.HandleFunc("GET /telemetry/stream", srv.auth(srv.telemetryStream))
	mux.HandleFunc("GET /telemetry/phases", srv.auth(srv.telemetryPhases))
	mux.HandleFunc("GET /telemetry/pipeline/report", srv.auth(srv.telemetryPipelineReport))
	mux.HandleFunc("POST /telemetry/phases/{phase}", srv.auth(srv.updatePhase))
	mux.HandleFunc("GET /telemetry/health", srv.auth(srv.telemetryHealth))
	// SSE endpoints are consumed through Site's server-side proxy. Keeping
	// authentication header-only prevents credentials from entering URLs,
	// browser cookies, access logs, and referrer metadata.
	mux.HandleFunc("GET /telemetry/sse", srv.authSSE(srv.telemetrySSE))
	mux.HandleFunc("GET /events/subscribe", srv.authSSE(srv.eventsSubscribe))
	mux.HandleFunc("GET /telemetry/roles", srv.auth(srv.telemetryRoles))
	mux.HandleFunc("GET /telemetry/roles/{name}", srv.auth(srv.telemetryRoleDetail))
	mux.HandleFunc("GET /telemetry/actors", srv.auth(srv.telemetryActors))
	mux.HandleFunc("GET /telemetry/layers", srv.auth(srv.telemetryLayers))
	mux.HandleFunc("GET /telemetry/overview", srv.auth(srv.telemetryOverview))
	mux.HandleFunc("GET /telemetry/", srv.telemetryDashboard)

	// Workspace-scoped routes — isolated namespace per team, auth via WORK_API_TOKEN.
	mux.HandleFunc("GET /w/{workspace}", srv.workspaceDashboard)
	mux.HandleFunc("POST /w/{workspace}/tasks", srv.tokenAuth(srv.createWorkspaceTask))
	mux.HandleFunc("GET /w/{workspace}/tasks", srv.tokenAuth(srv.listWorkspaceTasks))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/assign", srv.tokenAuth(srv.assignTask))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/complete", srv.tokenAuth(srv.completeTask))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/comment", srv.tokenAuth(srv.addComment))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/artifacts", srv.tokenAuth(srv.addArtifact))
	mux.HandleFunc("GET /w/{workspace}/tasks/{id}/artifacts", srv.tokenAuth(srv.listArtifacts))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/waive-artifact", srv.tokenAuth(srv.waiveArtifact))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/fact-requirements", srv.tokenAuth(srv.addFactRequirement))

	addr, err := workServerListenAddress(os.Getenv, port)
	if err != nil {
		return fmt.Errorf("listen address: %w", err)
	}
	fmt.Fprintf(os.Stderr, "work-server listening on %s\n", addr)
	httpSrv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background()) //nolint:errcheck
	}()
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func workServerListenAddress(getenv func(string) string, port string) (string, error) {
	host := strings.TrimSpace(getenv("WORK_BIND_HOST"))
	if strings.ContainsAny(host, "[]") {
		if !strings.HasPrefix(host, "[") || !strings.HasSuffix(host, "]") {
			return "", errors.New("WORK_BIND_HOST has unmatched IPv6 brackets")
		}
		inside := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
		if inside == "" || strings.ContainsAny(inside, "[]") {
			return "", errors.New("WORK_BIND_HOST has malformed IPv6 brackets")
		}
		address, err := netip.ParseAddr(inside)
		if err != nil || !address.Is6() {
			return "", errors.New("bracketed WORK_BIND_HOST must contain a valid IPv6 address")
		}
		host = inside
	}
	return net.JoinHostPort(host, port), nil
}

// server holds shared dependencies for HTTP handlers.
type server struct {
	ts         *work.TaskStore
	phaseGates *work.PhaseGateStore
	store      store.Store
	humanID    types.ActorID
	apiKey     string
	apiToken   string
	pool       *pgxpool.Pool // nil when running in-memory; telemetry handlers check this

	// fanout broadcasts bus-sourced events to SSE subscribers. A background
	// goroutine (runEventPoller) tails hive's telemetry_event_stream at 500ms
	// and republishes each new row on this fanout.
	fanout *eventFanout
}

// telemetryDashboard permanently redirects the retired browser surface to Site.
func (sv *server) telemetryDashboard(w http.ResponseWriter, r *http.Request) {
	replacement := siteUIURL(r, "/ops/telemetry")
	redirectLegacyBrowserUI(w, r, replacement)
}

// dashboard permanently redirects the retired browser surface to Site. Work
// credentials are never placed in HTML or cookies.
func (sv *server) dashboard(w http.ResponseWriter, r *http.Request) {
	replacement := siteUIURL(r, "/ops/work")
	redirectLegacyBrowserUI(w, r, replacement)
}

// workspaceDashboard permanently redirects the retired workspace UI to Site.
func (sv *server) workspaceDashboard(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	replacement := siteUIURL(r, "/ops/work?workspace="+url.QueryEscape(workspace))
	redirectLegacyBrowserUI(w, r, replacement)
}

func redirectLegacyBrowserUI(w http.ResponseWriter, r *http.Request, replacement string) {
	w.Header().Set(legacyUIStatusHeader, "disabled")
	w.Header().Set(legacyUIReplacementHeader, replacement)
	http.Redirect(w, r, replacement, http.StatusFound)
}

func siteUIURL(r *http.Request, path string) string {
	if base := strings.TrimSpace(os.Getenv("SITE_UI_BASE_URL")); base != "" {
		u, err := url.Parse(strings.TrimRight(base, "/"))
		if err == nil {
			ref, refErr := url.Parse(path)
			if refErr == nil {
				return u.ResolveReference(ref).String()
			}
			return u.String()
		}
	}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}
	if name == "" {
		name = "localhost"
	}
	scheme := "http"
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + name + ":8201" + path
}

// health handles GET /health — used by Fly.io and load balancers to check liveness.
func (sv *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// auth validates the API key exclusively through a Bearer header. Browser UI
// lives in Site, whose server-side proxy owns the credential.
func (sv *server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); found && secureTokenEqual(token, sv.apiKey) {
			next(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "invalid or missing API key")
	}
}

// tokenAuth is middleware for workspace routes that validates the WORK_API_TOKEN bearer token.
func (sv *server) tokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || !secureTokenEqual(token, sv.apiToken) {
			writeErr(w, http.StatusUnauthorized, "invalid or missing API token")
			return
		}
		next(w, r)
	}
}

func secureTokenEqual(candidate, expected string) bool {
	return len(candidate) == len(expected) && subtle.ConstantTimeCompare([]byte(candidate), []byte(expected)) == 1
}

// createTask handles POST /tasks
// Body: {"title":"...", "description":"...", "priority":"high"}
func (sv *server) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	task, err := sv.ts.Create(sv.humanID, body.Title, body.Description, causes, convID, work.TaskPriority(body.Priority))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "create task: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          task.ID.Value(),
		"title":       task.Title,
		"description": task.Description,
		"priority":    string(task.Priority),
		"created_by":  task.CreatedBy.Value(),
	})
}

// listTasks handles GET /tasks
// Query params: ?open=true, ?priority=high, ?assignee=<actor_id>
func (sv *server) listTasks(w http.ResponseWriter, r *http.Request) {
	openOnly := r.URL.Query().Get("open") == "true"
	priorityFilter := r.URL.Query().Get("priority")
	assigneeFilter := r.URL.Query().Get("assignee")

	summaries, err := sv.ts.ListSummariesCached(100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list tasks: "+err.Error())
		return
	}

	if openOnly {
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.LegacyStatus != work.LegacyStatusCompleted && !isTerminalTaskStatus(s.Status) && !s.Blocked {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	if assigneeFilter != "" {
		aid, err := types.NewActorID(assigneeFilter)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid assignee: "+err.Error())
			return
		}
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.Assignee == aid {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	if priorityFilter != "" {
		p := work.TaskPriority(priorityFilter)
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.Task.Priority == p {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	items := make([]map[string]any, 0, len(summaries))
	for _, s := range summaries {
		item := map[string]any{
			"id":               s.Task.ID.Value(),
			"title":            s.Task.Title,
			"description":      s.Task.Description,
			"priority":         string(s.Task.Priority),
			"created_by":       s.Task.CreatedBy.Value(),
			"status":           string(s.Status),
			"legacy_status":    string(s.LegacyStatus),
			"assignee":         s.Assignee.Value(),
			"blocked":          s.Blocked,
			"artifact_count":   s.ArtifactCount,
			"waived":           s.Waived,
			"ready":            s.Ready,
			"missing_gates":    s.MissingGates,
			"missing_facts":    s.MissingFacts,
			"risk_class":       s.Task.RiskClass,
			"cell":             s.Task.Cell,
			"factory_order_id": s.Task.FactoryOrderID,
			"created_at":       s.Task.CreatedAt.UTC().Format(time.RFC3339),
		}
		addCanonicalTaskSummaryFields(item, s)
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": items})
}

func isTerminalTaskStatus(status work.TaskStatus) bool {
	switch status {
	case work.StatusCertified, work.StatusRejected, work.StatusSuperseded:
		return true
	default:
		return false
	}
}

func (sv *server) declarePhaseGate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phase    string   `json:"phase"`
		Title    string   `json:"title"`
		Criteria []string `json:"criteria"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	gate, err := sv.phaseGates.Declare(sv.humanID, body.Phase, body.Title, body.Criteria, causes, convID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "declare phase gate: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, phaseGateResponse(gate))
}

func (sv *server) listPhaseGates(w http.ResponseWriter, r *http.Request) {
	gates, err := sv.phaseGates.List(100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list phase gates: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(gates))
	for _, gate := range gates {
		items = append(items, phaseGateResponse(gate))
	}
	writeJSON(w, http.StatusOK, map[string]any{"gates": items})
}

func (sv *server) approvePhaseGate(w http.ResponseWriter, r *http.Request) {
	gateID, ok := parsePhaseGateID(w, r)
	if !ok {
		return
	}
	var body struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.phaseGates.Approve(sv.humanID, gateID, body.Summary, causes, convID); err != nil {
		writeErr(w, http.StatusBadRequest, "approve phase gate: "+err.Error())
		return
	}
	gate, _, _ := sv.phaseGates.Get(gateID)
	writeJSON(w, http.StatusOK, phaseGateResponse(gate))
}

func (sv *server) rejectPhaseGate(w http.ResponseWriter, r *http.Request) {
	gateID, ok := parsePhaseGateID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.phaseGates.Reject(sv.humanID, gateID, body.Reason, causes, convID); err != nil {
		writeErr(w, http.StatusBadRequest, "reject phase gate: "+err.Error())
		return
	}
	gate, _, _ := sv.phaseGates.Get(gateID)
	writeJSON(w, http.StatusOK, phaseGateResponse(gate))
}

func phaseGateResponse(gate work.PhaseGateState) map[string]any {
	return map[string]any{
		"id":          gate.ID.Value(),
		"phase":       gate.Phase,
		"title":       gate.Title,
		"criteria":    gate.Criteria,
		"status":      string(gate.Status),
		"declared_by": gate.DeclaredBy.Value(),
		"approved_by": gate.ApprovedBy.Value(),
		"rejected_by": gate.RejectedBy.Value(),
		"summary":     gate.Summary,
		"reason":      gate.Reason,
		"declared_at": gate.DeclaredAt.Format(time.RFC3339Nano),
		"updated_at":  gate.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// getTaskStatus handles GET /tasks/{id}/status
func (sv *server) getTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	status, err := sv.ts.GetStatus(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get status: "+err.Error())
		return
	}
	priority, err := sv.ts.GetPriority(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get priority: "+err.Error())
		return
	}
	blocked, err := sv.ts.IsBlocked(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "blocked check: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       taskID.Value(),
		"status":   string(status),
		"priority": string(priority),
		"blocked":  blocked,
	})
}

// assignTask handles POST /tasks/{id}/assign
// Body: {"assignee":"actor_id"} — omit assignee to assign to the human operator.
func (sv *server) assignTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Assignee string `json:"assignee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	assignee := sv.humanID
	if body.Assignee != "" {
		aid, err := types.NewActorID(body.Assignee)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid assignee: "+err.Error())
			return
		}
		assignee = aid
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.Assign(sv.humanID, taskID, assignee, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "assign: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":  taskID.Value(),
		"assignee": assignee.Value(),
	})
}

// unblockTask handles POST /tasks/{id}/unblock
// Emits a work.task.unblocked event, explicitly marking the task's blockers resolved.
// Body: {} (no fields required; actor is the authenticated human operator)
func (sv *server) unblockTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.UnblockTask(sv.humanID, taskID, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "unblock: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": taskID.Value(),
		"blocked": false,
	})
}

// completeTask handles POST /tasks/{id}/complete
// Body: {"summary":"..."}
func (sv *server) completeTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.Complete(sv.humanID, taskID, body.Summary, causes, convID); err != nil {
		if errors.Is(err, work.ErrArtifactRequired) {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "complete: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":       taskID.Value(),
		"status":        string(work.StatusCreated),
		"legacy_status": "completed",
	})
}

// addComment handles POST /tasks/{id}/comment
// Body: {"body":"..."}
func (sv *server) addComment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Body == "" {
		writeErr(w, http.StatusBadRequest, "body is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.AddComment(taskID, body.Body, sv.humanID, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "add comment: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id": taskID.Value(),
		"status":  "commented",
	})
}

// listComments handles GET /tasks/{id}/comments
// Returns all comments for the task in chronological order.
func (sv *server) listComments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	comments, err := sv.ts.ListComments(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list comments: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		items = append(items, map[string]any{
			"id":        c.ID.Value(),
			"task_id":   c.TaskID.Value(),
			"body":      c.Body,
			"author_id": c.AuthorID.Value(),
			"timestamp": c.Timestamp.String(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID.Value(), "comments": items})
}

// addArtifact handles POST /tasks/{id}/artifacts
// Body: {"label":"...", "media_type":"text/markdown", "body":"..."}
func (sv *server) addArtifact(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Label     string `json:"label"`
		MediaType string `json:"media_type"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Label == "" {
		writeErr(w, http.StatusBadRequest, "label is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.AddArtifact(sv.humanID, taskID, body.Label, body.MediaType, body.Body, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "add artifact: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id": taskID.Value(),
		"label":   body.Label,
		"status":  "attached",
	})
}

// listArtifacts handles GET /tasks/{id}/artifacts
func (sv *server) listArtifacts(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	artifacts, err := sv.ts.ListArtifacts(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list artifacts: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(artifacts))
	for _, a := range artifacts {
		items = append(items, map[string]any{
			"id":         a.ID.Value(),
			"label":      a.Label,
			"media_type": a.MediaType,
			"body":       a.Body,
			"created_by": a.CreatedBy.Value(),
			"timestamp":  a.Timestamp.String(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID.Value(), "artifacts": items})
}

// waiveArtifact handles POST /tasks/{id}/waive-artifact
// Body: {"reason":"..."}
func (sv *server) waiveArtifact(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Reason == "" {
		writeErr(w, http.StatusBadRequest, "reason is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.WaiveArtifact(sv.humanID, taskID, body.Reason, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "waive artifact: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id": taskID.Value(),
		"status":  "waived",
	})
}

// addFactRequirement handles POST /tasks/{id}/fact-requirements.
// Body: {"event_type":"authority.decision.recorded", "event_id":"optional-uuid-v7", "reason":"..."}
func (sv *server) addFactRequirement(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		EventType string `json:"event_type"`
		EventID   string `json:"event_id"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	requiredType, err := types.NewEventType(body.EventType)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid event_type: "+err.Error())
		return
	}
	var requiredID types.EventID
	if body.EventID != "" {
		requiredID, err = types.NewEventID(body.EventID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid event_id: "+err.Error())
			return
		}
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.AddFactRequirement(sv.humanID, taskID, requiredType, requiredID, body.Reason, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "add fact requirement: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id":    taskID.Value(),
		"event_type": requiredType.Value(),
		"event_id":   requiredID.Value(),
		"status":     "required",
	})
}

// getTask handles GET /tasks/{id}
// Returns full task details: title, description, priority, status, assignee, blocked.
func (sv *server) getTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	// Fetch the creation event for base task fields.
	ev, err := sv.store.Get(taskID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "task not found: "+err.Error())
		return
	}
	c, ok := ev.Content().(work.TaskCreatedContent)
	if !ok {
		writeErr(w, http.StatusNotFound, "event is not a task")
		return
	}

	status, err := sv.ts.GetStatus(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get status: "+err.Error())
		return
	}
	priority, err := sv.ts.GetPriority(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get priority: "+err.Error())
		return
	}
	blocked, err := sv.ts.IsBlocked(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "blocked check: "+err.Error())
		return
	}

	// Find current assignee: assigned events are returned newest-first, so the first match wins.
	var assignee string
	assignedPage, err := sv.store.ByType(work.EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err == nil {
		for _, ae := range assignedPage.Items() {
			ac, ok := ae.Content().(work.TaskAssignedContent)
			if ok && ac.TaskID == taskID {
				assignee = ac.AssignedTo.Value()
				break
			}
		}
	}

	artifacts, err := sv.ts.ListArtifacts(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list artifacts: "+err.Error())
		return
	}
	hasWaiver, err := sv.ts.HasWaiver(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "check waiver: "+err.Error())
		return
	}
	readiness, err := sv.ts.Readiness(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "check readiness: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":             taskID.Value(),
		"title":          c.Title,
		"description":    c.Description,
		"priority":       string(priority),
		"status":         string(status),
		"created_by":     c.CreatedBy.Value(),
		"assignee":       assignee,
		"blocked":        blocked,
		"artifact_count": len(artifacts),
		"waived":         hasWaiver,
		"ready":          readiness.Ready,
		"missing_gates":  readiness.MissingGates,
		"present_facts":  readiness.PresentFacts,
		"missing_facts":  readiness.MissingFacts,
	})
}

// getTaskEvents handles GET /tasks/{id}/events
// Returns the ordered audit trail of all work.task.* events causally linked to this task.
func (sv *server) getTaskEvents(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	var collected []event.Event

	// Include the task creation event itself.
	if ev, err := sv.store.Get(taskID); err == nil {
		if _, ok := ev.Content().(work.TaskCreatedContent); ok {
			collected = append(collected, ev)
		}
	}

	// Scan all other work event types for events that reference this task.
	for _, et := range []types.EventType{
		work.EventTypeTaskAssigned,
		work.EventTypeTaskCompleted,
		work.EventTypeTaskDependencyAdded,
		work.EventTypeTaskPrioritySet,
		work.EventTypeTaskComment,
		work.EventTypeTaskUnblocked,
		work.EventTypeTaskArtifact,
		work.EventTypeTaskArtifactWaived,
		work.EventTypeTaskFactRequired,
	} {
		page, err := sv.store.ByType(et, 1000, types.None[types.Cursor]())
		if err != nil {
			continue
		}
		for _, ev := range page.Items() {
			if taskIDFromContent(ev.Content()) == taskID {
				collected = append(collected, ev)
			}
		}
	}

	// Sort chronologically (oldest first) for a readable audit trail.
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Timestamp().Value().Before(collected[j].Timestamp().Value())
	})

	items := make([]map[string]any, 0, len(collected))
	for _, ev := range collected {
		items = append(items, map[string]any{
			"id":        ev.ID().Value(),
			"type":      ev.Type().Value(),
			"source":    ev.Source().Value(),
			"timestamp": ev.Timestamp().String(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID.Value(), "events": items})
}

// taskIDFromContent extracts the TaskID field from a work event content struct.
// Returns zero value EventID if the content type does not reference a task ID.
func taskIDFromContent(content any) types.EventID {
	switch c := content.(type) {
	case work.TaskAssignedContent:
		return c.TaskID
	case work.TaskCompletedContent:
		return c.TaskID
	case work.TaskDependencyContent:
		return c.TaskID
	case work.TaskPrioritySetContent:
		return c.TaskID
	case work.CommentContent:
		return c.TaskID
	case work.TaskUnblockedContent:
		return c.TaskID
	case work.TaskArtifactContent:
		return c.TaskID
	case work.TaskArtifactWaivedContent:
		return c.TaskID
	case work.TaskFactRequiredContent:
		return c.TaskID
	}
	return types.EventID{}
}

// createWorkspaceTask handles POST /w/{workspace}/tasks
// Creates a task scoped to the given workspace namespace.
// Body: {"title":"...", "description":"...", "priority":"high"}
func (sv *server) createWorkspaceTask(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	task, err := sv.ts.CreateInWorkspace(sv.humanID, body.Title, body.Description, workspace, causes, convID, work.TaskPriority(body.Priority))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "create task: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          task.ID.Value(),
		"title":       task.Title,
		"description": task.Description,
		"priority":    string(task.Priority),
		"workspace":   task.Workspace,
		"created_by":  task.CreatedBy.Value(),
	})
}

// listWorkspaceTasks handles GET /w/{workspace}/tasks
// Lists tasks scoped to the given workspace namespace.
func (sv *server) listWorkspaceTasks(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	summaries, err := sv.ts.ListSummariesByWorkspace(workspace, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list tasks: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(summaries))
	for _, s := range summaries {
		item := map[string]any{
			"id":             s.Task.ID.Value(),
			"title":          s.Task.Title,
			"description":    s.Task.Description,
			"priority":       string(s.Task.Priority),
			"workspace":      s.Task.Workspace,
			"created_by":     s.Task.CreatedBy.Value(),
			"status":         string(s.Status),
			"legacy_status":  string(s.LegacyStatus),
			"assignee":       s.Assignee.Value(),
			"blocked":        s.Blocked,
			"artifact_count": s.ArtifactCount,
			"waived":         s.Waived,
			"ready":          s.Ready,
			"missing_gates":  s.MissingGates,
			"missing_facts":  s.MissingFacts,
		}
		addCanonicalTaskSummaryFields(item, s)
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace, "tasks": items})
}

// --- Helpers ---

func addCanonicalTaskSummaryFields(item map[string]any, summary work.TaskSummary) {
	if summary.Canonical != nil {
		item["canonical"] = summary.Canonical
		return
	}
	if summary.CanonicalError != "" {
		item["canonical_error"] = summary.CanonicalError
	}
}

// currentCauses fetches the current graph head to use as a cause for new events.
func (sv *server) currentCauses() ([]types.EventID, error) {
	head, err := sv.store.Head()
	if err != nil {
		return nil, err
	}
	if head.IsSome() {
		return []types.EventID{head.Unwrap().ID()}, nil
	}
	return nil, nil
}

// parseTaskID extracts and validates the {id} path parameter from the request.
func parseTaskID(w http.ResponseWriter, r *http.Request) (types.EventID, bool) {
	idStr := r.PathValue("id")
	if idStr == "" {
		writeErr(w, http.StatusBadRequest, "task id is required")
		return types.EventID{}, false
	}
	taskID, err := types.NewEventID(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid task id: "+err.Error())
		return types.EventID{}, false
	}
	return taskID, true
}

func parsePhaseGateID(w http.ResponseWriter, r *http.Request) (types.EventID, bool) {
	idStr := r.PathValue("id")
	if idStr == "" {
		writeErr(w, http.StatusBadRequest, "phase gate id is required")
		return types.EventID{}, false
	}
	gateID, err := types.NewEventID(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid phase gate id: "+err.Error())
		return types.EventID{}, false
	}
	return gateID, true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeErr writes a JSON error response.
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Infrastructure helpers (mirror of cmd/work patterns) ---

func openStore(ctx context.Context, pool *pgxpool.Pool) (store.Store, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Store: in-memory")
		return store.NewInMemoryStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Store: postgres")
	return pgstore.NewPostgresStoreFromPool(ctx, pool)
}

func openActorStore(ctx context.Context, pool *pgxpool.Pool) (actor.IActorStore, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Actor store: in-memory")
		return actor.NewInMemoryActorStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Actor store: postgres")
	return pgactor.NewPostgresActorStoreFromPool(ctx, pool)
}

// bootstrapGraph emits the genesis event if the store is empty. Idempotent.
func bootstrapGraph(s store.Store, humanID types.ActorID) error {
	return bootstrapGraphWithSigner(s, humanID, deriveSignerFromID(humanID))
}

func bootstrapGraphWithSigner(s store.Store, humanID types.ActorID, signer event.Signer) error {
	head, err := s.Head()
	if err != nil {
		return fmt.Errorf("check head: %w", err)
	}
	if head.IsSome() {
		return nil // already bootstrapped
	}
	fmt.Fprintln(os.Stderr, "Bootstrapping event graph...")
	registry := event.DefaultRegistry()
	bsFactory := event.NewBootstrapFactory(registry)
	bootstrap, err := bsFactory.Init(humanID, signer)
	if err != nil {
		return fmt.Errorf("create genesis event: %w", err)
	}
	if _, err := s.Append(bootstrap); err != nil {
		return fmt.Errorf("append genesis event: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Event graph bootstrapped.")
	return nil
}

// bootstrapSigner provides a minimal Signer for the genesis event.
type bootstrapSigner struct {
	humanID types.ActorID
}

func (b *bootstrapSigner) Sign(data []byte) (types.Signature, error) {
	h := sha256.Sum256([]byte("signer:" + b.humanID.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	sig := ed25519.Sign(priv, data)
	return types.NewSignature(sig)
}

// registerHuman bootstraps a human operator in the actor store.
// WARNING: derives key from display name — insecure for production persistent stores.
// Mirrors cmd/hive registerHuman exactly so the same name produces the same ActorID.
func registerHuman(actors actor.IActorStore, displayName string) (types.ActorID, error) {
	h := sha256.Sum256([]byte("human:" + displayName))
	priv := ed25519.NewKeyFromSeed(h[:])
	pub := priv.Public().(ed25519.PublicKey)
	pk, err := types.NewPublicKey([]byte(pub))
	if err != nil {
		return types.ActorID{}, fmt.Errorf("public key: %w", err)
	}
	a, err := actors.Register(pk, displayName, event.ActorTypeHuman)
	if err != nil {
		return types.ActorID{}, err
	}
	return a.ID(), nil
}

// ed25519Signer implements event.Signer for work-emitted events.
type ed25519Signer struct {
	key ed25519.PrivateKey
}

func (s *ed25519Signer) Sign(data []byte) (types.Signature, error) {
	sig := ed25519.Sign(s.key, data)
	return types.NewSignature(sig)
}

// deriveSignerFromID creates a deterministic Ed25519 signer from an ActorID.
// Stable across restarts — the same humanID always produces the same key.
func deriveSignerFromID(id types.ActorID) *ed25519Signer {
	h := sha256.Sum256([]byte("signer:" + id.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	return &ed25519Signer{key: priv}
}

// newConversationID generates a unique ConversationID for this HTTP request.
func newConversationID() (types.ConversationID, error) {
	id, err := types.NewEventIDFromNew()
	if err != nil {
		return types.ConversationID{}, err
	}
	return types.NewConversationID("work-server-" + id.Value())
}
