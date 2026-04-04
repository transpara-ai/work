# Telemetry System

The telemetry system snapshots live hive state — agent FSM states, iteration counts,
token costs, hive health metrics, and a recent event stream — to dedicated Postgres
tables every 10–15 seconds. The work-server exposes this data as a JSON API; a
standalone dashboard HTML file in `lovyou-ai-summary` polls the API and renders the
mission control view. Postgres is the bridge: the hive writes, the work-server reads,
the dashboard fetches.

---

## Architecture

```
lovyou-ai-hive               lovyou-ai-work               lovyou-ai-summary
(cmd/hive)                    (cmd/work-server)            (dashboard.html)
┌──────────────────┐          ┌────────────────────┐       ┌──────────────────┐
│  Hive Runtime    │          │  Work-Server       │       │  Mission Control │
│  ├─ Agent loops  │ postgres │  ├─ /tasks/*       │ HTTP  │  (static HTML)   │
│  ├─ Writer ──────┼───→──────┼──┤ /telemetry/*    │←──────┼── fetch() polls  │
│  └─ Pruner       │ (5432)   │  └─ CORS + auth    │       │  URL params      │
└──────────────────┘          └────────────────────┘       └──────────────────┘
```

**Writer** — `lovyou-ai-hive/pkg/telemetry/writer.go`. Pure Go goroutine, no LLM.
Runs inside the hive process, writes every `TELEMETRY_INTERVAL` (default 10s).

**API** — `lovyou-ai-work/cmd/work-server`. Reads from the same `DATABASE_URL` pool
the work-server already uses for task data. No new DB connection.

**Dashboard** — `lovyou-ai-summary/dashboard.html`. Static file, served from
nucbuntu or opened directly in a browser. Configured entirely via URL parameters.

---

## Configuration

| Variable             | Default | Description                                 |
|----------------------|---------|---------------------------------------------|
| `DATABASE_URL`       | —       | Postgres DSN (required for telemetry reads) |
| `TELEMETRY_INTERVAL` | `10s`   | Writer snapshot frequency (hive process)    |

---

## Pruning

The telemetry writer runs a pruner goroutine alongside the snapshot goroutine:

| Table                       | Retention              |
|-----------------------------|------------------------|
| `telemetry_agent_snapshots` | 24 hours               |
| `telemetry_hive_snapshots`  | 7 days                 |
| `telemetry_event_stream`    | Last 1,000 rows (ring) |
| `telemetry_phases`          | Never pruned           |

---

## Dashboard

Open `dashboard.html` from the `lovyou-ai-summary` repo in any browser:

```
# Served via any static file server on nucbuntu:
open http://nucbuntu:PORT/dashboard.html?api=http://nucbuntu:8080&key=$API_KEY

# Or served directly from GitHub Pages:
open https://transpara-ai.github.io/lovyou-ai-summary/dashboard.html?api=http://nucbuntu:8080&key=$API_KEY

# Or opened as a local file (CORS allows file:// with the work-server's * origin):
open dashboard.html?api=http://nucbuntu:8080&key=$API_KEY
```

URL parameters:
- `api` — base URL of the work-server (no trailing slash)
- `key` — `WORK_API_KEY` value

---

## API Endpoints

All endpoints require `Authorization: Bearer <WORK_API_KEY>`.

```bash
export API_KEY=<your WORK_API_KEY>
BASE=http://localhost:8080
```

### Full status snapshot

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/status | jq .
```

Returns agents, hive health, phases, and last 50 events in one call.

### Per-agent snapshots

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/agents | jq .
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/agents/guardian | jq .
```

`/agents/{role}` returns the latest snapshot plus the last 20 for trend data.

### Event stream

```bash
# Default: last 50 events
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/stream | jq .

# Up to 200 events
curl -s -H "Authorization: Bearer $API_KEY" "$BASE/telemetry/stream?limit=200" | jq .
```

### Expansion phases

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/phases | jq .
```

### Hive health

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE/telemetry/health | jq .
```

---

## Phase Updates (Graduation Ceremonies)

Mark an expansion phase as started or complete:

```bash
# Phase 2 starting
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress","notes":"CTO implementation started"}' \
  $BASE/telemetry/phases/2 | jq .

# Phase 2 complete
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status":"complete","notes":"CTO and Architect graduated"}' \
  $BASE/telemetry/phases/2 | jq .
```

Valid status values: `blocked`, `in_progress`, `complete`.

Timestamp behaviour:
- `started_at` is set to `now()` on first transition to `in_progress` (preserved on subsequent calls).
- `completed_at` is set to `now()` when status becomes `complete`; cleared to `null` otherwise.
- `notes` is updated only when a non-empty value is provided; existing notes are preserved when `notes` is omitted or empty.

---

## End-to-End Verification Checklist

### 1. Verify telemetry tables exist in hive-postgres

```bash
docker exec hive-postgres-1 psql -U hive -d hive -c "\dt telemetry_*"
```

Expected: four tables — `telemetry_agent_snapshots`, `telemetry_event_stream`,
`telemetry_hive_snapshots`, `telemetry_phases`.

### 2. Verify phase seed data

```bash
docker exec hive-postgres-1 psql -U hive -d hive \
  -c "SELECT phase, label, status FROM telemetry_phases ORDER BY phase"
```

Expected: Phases 0–8, with Phase 0 and 1 showing `complete`, Phase 2 showing
`blocked` or `in_progress`.

### 3. Verify API responds

```bash
curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/telemetry/phases | jq .
```

Expected: `{"phases": [...]}` with 9 phase objects.

### 4. Verify dashboard loads

Open in browser:

```
http://nucbuntu:8080 → navigate to /telemetry/
# or directly:
open dashboard.html?api=http://nucbuntu:8080&key=$API_KEY
```

Expected: page loads, connection indicator is green, phase timeline renders.

### 5. Verify live data when hive is running

- Agent cards appear within 15 seconds (one writer interval).
- Event stream shows `health.report` events from SysMon.
- Hive health panel shows `chain_length` ≥ 1,008.
- Cost and iteration counters increment across refreshes.

### 6. Verify phase update round-trip

```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress","notes":"CTO implementation started"}' \
  http://localhost:8080/telemetry/phases/2 | jq .
```

Expected: response contains `"status": "in_progress"` and a non-null `started_at`.
On the next dashboard poll (≤10s), Phase 2 card switches from gray to blue.

### 7. Verify graceful degradation when hive is not running

```bash
# With DATABASE_URL pointing to a live postgres but no hive running:
curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/telemetry/status | jq .
```

Expected: HTTP 200, `agents: []`, `hive: null`, `phases: [...]` (phases table is
persistent), `recent_events: []`.

```bash
# With DATABASE_URL unset (in-memory mode):
curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/telemetry/status
```

Expected: HTTP 503 `{"error":"telemetry not available", ...}`.
