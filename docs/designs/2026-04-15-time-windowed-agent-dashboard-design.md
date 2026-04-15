# Time-Windowed Agent Dashboard with State Tracking

**Date:** 2026-04-15
**Status:** Approved
**Scope:** `cmd/work-server` (telemetry API + dashboard)

## Problem

The telemetry dashboard shows only the latest snapshot per agent with no historical context. Pipeline mode spawns multiple agents that share the same role — some terminate while others start. The dashboard provides no way to see agents that ran earlier, how long they ran, what states they passed through, or whether any got stuck. Operators have no time-based view into hive behavior.

## Solution

Add time window controls (Now / 1h / 24h) to the dashboard. "Now" behaves as today with live SSE updates. Historical views freeze the display and show agent lifecycle data including time spent in each FSM state. A new server-side endpoint computes state transitions from existing snapshot rows using SQL window functions.

## Design

### 1. Time Window Buttons

A button bar at the top of the agent section: **Now** | **1h** | **24h**.

- **Now** (default): SSE streams live, cards update every 10s. Identical to current behavior plus new visual differentiation and time-in-current-state display.
- **1h / 24h**: Freezes the display. SSE connection stays alive but rendering stops. A banner appears: _"Viewing last hour — paused at 14:32:05"_ with "Now" as the resume action. Dashboard fetches from the new history endpoint once on click.

Switching back to "Now" resumes SSE rendering immediately (no reconnect delay).

### 2. Agent State History Endpoint

```
GET /telemetry/agents/history?window=1h|24h
```

Computes state transitions from `telemetry_agent_snapshots` using `LAG()` window functions. No schema changes — reads the existing append-only table.

**Query approach:**
- Partition by `actor_id`, order by `recorded_at`
- Compare consecutive rows to detect state changes (where `state` column differs from previous row)
- Compute duration as time between transitions
- Return the agent's latest snapshot fields plus a `states` array and `first_seen`/`last_seen` timestamps

**Response shape:**

```json
{
  "agents": [
    {
      "role": "builder",
      "actor_id": "ac3f...",
      "current_state": "retired",
      "model": "claude-sonnet-4-6",
      "iteration": 5,
      "max_iterations": 10,
      "tokens_used": 45000,
      "cost_usd": 0.234,
      "trust_score": 0.95,
      "errors": 0,
      "first_seen": "2026-04-15T14:02:00Z",
      "last_seen": "2026-04-15T14:47:00Z",
      "states": [
        {"state": "idle", "entered_at": "2026-04-15T14:02:00Z", "duration_seconds": 12},
        {"state": "processing", "entered_at": "2026-04-15T14:02:12Z", "duration_seconds": 2400},
        {"state": "waiting", "entered_at": "2026-04-15T14:42:12Z", "duration_seconds": 180},
        {"state": "retired", "entered_at": "2026-04-15T14:45:12Z", "duration_seconds": 108}
      ]
    }
  ],
  "window": "1h",
  "computed_at": "2026-04-15T15:00:00Z"
}
```

**"Now" mode does not use this endpoint** — it continues using the existing SSE stream. The client-side JS computes time-in-current-state from `last_event_at` against wall clock.

**Validation:** The `window` parameter must be `1h` or `24h`. Any other value returns HTTP 400. This is intentionally restrictive — when longer windows are supported, new values will be added.

**Performance:** With 24h retention and ~10s write interval, the table holds ~8,640 rows per agent. A `LAG()` window function over a bounded partition is well within the 5s query timeout. The existing index on `(agent_role, recorded_at DESC)` partially helps; a future index on `(actor_id, recorded_at)` would be ideal but is not required at this scale.

### 3. Agent Card Rendering

Unified grid, all cards same size. Visual differentiation via left border color and opacity:

| Condition | Left border | Opacity | Display |
|---|---|---|---|
| Active (snapshot < 20s, non-terminal state) | Green | 1.0 | State badge + "running Xm Ys" (live ticking) |
| Stale (snapshot 30s–2min, non-terminal state) | Amber | 1.0 | Amber glow + "stale Xs" |
| Stuck (snapshot > 2min, non-terminal state) | Red | 1.0 | Red pulse + "stuck Xm Ys" |
| Terminated/Retired | Gray | 0.7 | Start → End timestamps |

**State timeline bar:** Each card includes a compact horizontal bar segmented by FSM state with proportional widths and state-colored fills.
- In "Now" mode: shows time-in-current-state (ticking in JS)
- In historical modes: shows full lifecycle from first_seen to last_seen

**State color mapping** (reuses existing badge colors from dashboard):
- Idle: green
- Processing: blue
- Waiting: amber
- Escalating/Refusing: red
- Suspended: gray
- Retiring/Retired: muted gray

### 4. Stuck Detection

Applies in **all** views — "Now" and historical.

**Now mode:** Computed client-side by comparing `last_event_at` against wall clock.
- Stale: `now - last_event_at > 30s` AND state not in `{retired, suspended, idle}`
- Stuck: `now - last_event_at > 2min` AND state not in `{retired, suspended, idle}`

**Historical views:** Computed server-side from gaps in snapshot timestamps. If consecutive snapshots for the same `actor_id` are spaced > 2min apart and the state didn't transition to a terminal state, that period is flagged as stuck. The `states` array includes these stuck periods as entries so the timeline bar renders them with red segments.

### 5. Paused Indicator

When viewing 1h or 24h:
- A banner replaces the SSE connection status: _"Viewing last hour — paused at HH:MM:SS"_
- The "Now" button in the time window bar acts as the resume control
- No time-in-state ticking — values are frozen at fetch time
- The banner uses the same slot as the existing connection status indicator to avoid layout shift

## Future: Week View

The "Week" button is not included in this implementation. The `telemetry_agent_snapshots` table has 24h retention, so data beyond that is not available.

**Path to support Week:**
1. **Option A:** Extend retention to 7 days. Simple but increases table size ~7x. May need more aggressive pruning of non-transition rows (keep only rows where state changed).
2. **Option B:** Add a `telemetry_agent_state_changes` materialized table. The hive's telemetry writer inserts a row only when an agent's state changes. Compact, query-efficient, and decoupled from snapshot frequency. This is the recommended long-term approach.

Both options require changes to the hive repo's telemetry writer, which is out of scope for this work.

## Files Modified

- `cmd/work-server/telemetry.go` — new `queryAgentHistory()` function and `/telemetry/agents/history` handler
- `cmd/work-server/telemetry_dashboard.go` — time window buttons, card rendering changes, paused banner, state timeline bar, stuck detection JS
- `cmd/work-server/main.go` — register new route

## Files Not Modified

- No schema changes (reads existing `telemetry_agent_snapshots`)
- No changes to the hive repo's telemetry writer
- No changes to the event sourcing model
