# Task Artifacts — Design Spec

**Date:** 2026-04-09
**Status:** Draft
**Scope:** lovyou-ai-work (Layer 1 event model, API, dashboard) + hive runtime integration points

## Problem

When a hive agent completes a task, the only structured output is the `Summary` field on `work.task.completed` — a short string. Rich deliverables (answers, code, analysis, shell output) exist only in `agent.evaluated` reasoning traces, which are internal monologue not linked to the task graph. Users cannot retrieve what an agent actually produced for a given task.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Artifact cardinality | Multi-artifact per task | A single task may produce code + tests + summary |
| Storage model | Immutable events on the graph | Consistent with event-sourcing; no new storage layer |
| Completion gate | Required by default, waivable | Prevents empty completions while allowing legitimate no-artifact tasks |
| Waiver mechanism | Explicit `work.task.artifact.waived` event | Auditable, self-documenting, queryable — turns a negative into a positive |

## New Event Types

### `work.task.artifact`

Appended by an agent to attach a deliverable to a task. Multiple artifacts per task are allowed and expected. Must be appended **before** the `work.task.completed` event for the same task (enforced by the completion gate).

```go
EventTypeTaskArtifact = types.MustEventType("work.task.artifact")

type TaskArtifactContent struct {
    workContent
    TaskID    types.EventID `json:"TaskID"`
    Label     string        `json:"Label"`               // human-readable name, e.g. "Rayleigh scattering explanation"
    MediaType string        `json:"MediaType"`            // MIME type: "text/plain", "text/markdown", "application/json", etc.
    Body      string        `json:"Body"`                 // the artifact content (text)
    CreatedBy types.ActorID `json:"CreatedBy"`
}
```

**Field notes:**
- `Label` is required. It appears in dashboard listings and API responses.
- `MediaType` defaults to `"text/markdown"` if empty. Enables future rendering decisions.
- `Body` holds the full artifact content inline. No external references for v1.
- There is no size limit enforced at the event level. Postgres `jsonb` handles large text. If binary artifacts become necessary later, a `"Reference"` field can be added in v2 without breaking the schema.

### `work.task.artifact.waived`

Appended by an agent to explicitly exempt a task from the artifact requirement. The completion gate accepts either an artifact OR a waiver.

```go
EventTypeTaskArtifactWaived = types.MustEventType("work.task.artifact.waived")

type TaskArtifactWaivedContent struct {
    workContent
    TaskID   types.EventID `json:"TaskID"`
    Reason   string        `json:"Reason"`    // why no artifact is needed, e.g. "operational task — restart completed"
    WaivedBy types.ActorID `json:"WaivedBy"`
}
```

**Field notes:**
- `Reason` is required. Empty reasons are rejected. This is the audit trail.
- Waivers are visible in the dashboard and queryable via the API, enabling the CTO/auditor agents to review exemption patterns over time.

## Completion Gate

### Logic

When an agent calls `TaskStore.Complete()`, the store checks:

1. Does at least one `work.task.artifact` event exist for this `TaskID`? → allow completion.
2. Does at least one `work.task.artifact.waived` event exist for this `TaskID`? → allow completion.
3. Neither exists → return error: `"task has no artifacts; attach an artifact or waive the requirement"`.

### Implementation

```go
func (ts *TaskStore) Complete(
    source types.ActorID,
    taskID types.EventID,
    summary string,
    causes []types.EventID,
    convID types.ConversationID,
) error {
    // --- Artifact gate ---
    hasArtifact, err := ts.hasEventForTask(EventTypeTaskArtifact, taskID)
    if err != nil {
        return fmt.Errorf("check artifacts: %w", err)
    }
    if !hasArtifact {
        hasWaiver, err := ts.hasEventForTask(EventTypeTaskArtifactWaived, taskID)
        if err != nil {
            return fmt.Errorf("check waivers: %w", err)
        }
        if !hasWaiver {
            return fmt.Errorf("task has no artifacts; attach an artifact or waive the requirement")
        }
    }
    // --- existing completion logic ---
    ...
}
```

The helper `hasEventForTask` scans `ByType` for the given event type and checks if any event's `TaskID` matches. This is consistent with the existing pattern used by `GetStatus`, `IsBlocked`, etc.

## TaskStore Methods

### New methods

```go
// AddArtifact records a work.task.artifact event on the graph.
func (ts *TaskStore) AddArtifact(
    source types.ActorID,
    taskID types.EventID,
    label, mediaType, body string,
    causes []types.EventID,
    convID types.ConversationID,
) error

// WaiveArtifact records a work.task.artifact.waived event on the graph.
func (ts *TaskStore) WaiveArtifact(
    source types.ActorID,
    taskID types.EventID,
    reason string,
    causes []types.EventID,
    convID types.ConversationID,
) error

// ListArtifacts returns all artifacts for a given task in chronological order.
func (ts *TaskStore) ListArtifacts(taskID types.EventID) ([]ArtifactEvent, error)
```

### New types

```go
type ArtifactEvent struct {
    ID        types.EventID
    TaskID    types.EventID
    Label     string
    MediaType string
    Body      string
    CreatedBy types.ActorID
    Timestamp time.Time
}
```

## HTTP API

### New endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/tasks/{id}/artifacts` | `WORK_API_KEY` | Attach an artifact to a task |
| GET | `/tasks/{id}/artifacts` | `WORK_API_KEY` | List artifacts for a task |
| POST | `/tasks/{id}/waive-artifact` | `WORK_API_KEY` | Waive the artifact requirement |
| POST | `/w/{workspace}/tasks/{id}/artifacts` | `WORK_API_TOKEN` | Workspace-scoped artifact attach |
| POST | `/w/{workspace}/tasks/{id}/waive-artifact` | `WORK_API_TOKEN` | Workspace-scoped waiver |

### POST `/tasks/{id}/artifacts` — request body

```json
{
    "label": "Rayleigh scattering explanation",
    "media_type": "text/markdown",
    "body": "The sky appears blue because of **Rayleigh scattering**..."
}
```

### POST `/tasks/{id}/waive-artifact` — request body

```json
{
    "reason": "operational task — service restart completed, no deliverable"
}
```

### GET `/tasks/{id}/artifacts` — response

```json
{
    "artifacts": [
        {
            "id": "019d...",
            "label": "Rayleigh scattering explanation",
            "media_type": "text/markdown",
            "body": "The sky appears blue because...",
            "created_by": "actor_54c8...",
            "timestamp": "2026-04-08T18:09:35Z"
        }
    ]
}
```

### Changes to existing endpoints

**GET `/tasks/{id}`** — add `artifact_count` and `waived` fields to the response so callers can see at a glance whether a task has deliverables without fetching them.

**GET `/tasks/{id}/events`** — already returns all `work.task.*` events for a task. Artifact and waiver events will appear in this audit trail automatically once registered.

## Dashboard Changes

### Task detail view

When a task has artifacts, show them below the task metadata:

- Each artifact displayed with its `Label`, `MediaType`, and rendered `Body`
- Markdown artifacts rendered as HTML
- Plain text / JSON artifacts displayed in a code block
- Waived tasks show a notice: "Artifact waived — {reason}"

### Task list view

Add an artifact indicator column:
- Number badge showing artifact count (e.g. "2 artifacts")
- Waiver icon for waived tasks
- Empty/warning state for completed tasks with neither (legacy data)

### Telemetry overview

Add aggregate stats:
- Total artifacts produced (all time)
- Waiver rate (waivers / completions)
- Agents by artifact count (who produces the most/least)

## Hive Integration Points

These changes are in the **hive runtime** (separate repo), listed here for completeness.

### Agent prompts

Agents with `can_operate: true` should be instructed:

> When completing a task, attach your deliverable as an artifact before marking the task complete.
> Use `work.task.artifact` with a descriptive label, appropriate media type, and the full output.
> If the task has no deliverable (operational tasks, restarts, deployments), waive the artifact requirement with a clear reason.

### Completion flow

The hive's task completion call path will start receiving errors from the gate when no artifact/waiver is present. Agents must adapt by:
1. Emitting `work.task.artifact` before `work.task.completed`, or
2. Emitting `work.task.artifact.waived` before `work.task.completed`

### CTO/Auditor review

The CTO and auditor agents should periodically review waiver events to:
- Identify patterns in waived task types
- Flag agents that waive excessively
- Recommend prompt adjustments for agents that should be producing artifacts but aren't

## Migration

### Backward compatibility

- Existing completed tasks have no artifacts and no waivers. They are legacy data.
- The completion gate only applies to **new** completions after deployment.
- The dashboard should handle the legacy case gracefully (no artifacts shown, no error).

### Event registration

Add to `events.go`:

```go
EventTypeTaskArtifact       = types.MustEventType("work.task.artifact")
EventTypeTaskArtifactWaived = types.MustEventType("work.task.artifact.waived")
```

Add to `RegisterEventTypes()`:

```go
event.RegisterContentUnmarshaler("work.task.artifact", event.Unmarshal[TaskArtifactContent])
event.RegisterContentUnmarshaler("work.task.artifact.waived", event.Unmarshal[TaskArtifactWaivedContent])
```

Add to `allWorkEventTypes()`:

```go
EventTypeTaskArtifact, EventTypeTaskArtifactWaived,
```

### No database migration needed

Events are stored in the generic `events` table with `content_json`. New event types require no DDL — just content unmarshaler registration.

## Build Sequence

All changes are in the `lovyou-ai-work` repo unless noted.

1. **Event types + content structs** (`events.go`) — add the two new types and registration
2. **Store types** (`store.go`) — add `ArtifactEvent` struct
3. **Store methods** (`store.go`) — add `AddArtifact`, `WaiveArtifact`, `ListArtifacts`, `hasEventForTask`
4. **Completion gate** (`store.go`) — modify `Complete` to enforce the gate
5. **Tests** (`store_test.go`) — artifact CRUD, waiver CRUD, gate enforcement, legacy compatibility
6. **API handlers** (`cmd/work-server/main.go`) — new endpoints + route registration
7. **Dashboard** (`cmd/work-server/telemetry_dashboard.go`) — artifact display in task views
8. **Hive prompts** (separate repo, out of scope) — update agent prompts to produce artifacts

## Out of Scope

- Binary artifact storage (v2 — add `Reference` field when needed)
- Artifact size limits (monitor organically first)
- Artifact search/full-text indexing
- Artifact versioning semantics (implicit via event ordering for now)
