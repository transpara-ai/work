# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build both binaries
go build ./cmd/...

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a single test file
go test -run TestName ./...

# Static analysis
go vet ./...
```

## Architecture

This is **Layer 1 of the Work Graph** — an event-sourced task management system built on the `eventgraph` framework (a sibling repo at `../eventgraph/go`, loaded via `go.mod` `replace` directive).

### Three layers

1. **Core package** (`events.go`, `store.go`) — domain model: tasks, events, the `TaskStore` interface, and its implementations (in-memory and Postgres-backed)
2. **CLI** (`cmd/work`) — human-facing task operations (create, list, assign, complete, status)
3. **HTTP server** (`cmd/work-server`) — REST API + embedded HTML dashboard + telemetry API reading from Postgres

### Event sourcing model

State is never mutated. All state changes are appended as typed events:
- `work.task.created`, `work.task.assigned`, `work.task.completed`
- `work.task.dependency.added`, `work.task.priority.set`, `work.task.comment`, `work.task.unblocked`

`TaskSummary` (task + computed status/assignee/blocked flag) is derived by replaying events. The `TaskStore` interface has methods for both raw `Task` and computed `TaskSummary`.

### Blocked state

A task is blocked if it has unresolved dependencies (incomplete dependency tasks), unless explicitly unblocked via `UnblockTask`. `IsBlocked(taskID)` computes this at query time.

### Workspaces

Tasks can be scoped to a named workspace string. The server exposes `/w/{workspace}/...` routes with a separate `WORK_API_TOKEN` credential.

### Telemetry

`cmd/work-server/telemetry.go` queries four Postgres tables written by the `lovyou-ai-hive` process:
- `telemetry_agent_snapshots` (24h retention)
- `telemetry_hive_snapshots` (7d retention)
- `telemetry_event_stream` (1,000-row ring buffer)
- `telemetry_phases` (permanent)

The dashboard at `cmd/work-server/telemetry_dashboard.go` is an embedded SPA served at `/` and `/w/{workspace}`.

### Server environment variables

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres DSN; falls back to in-memory if absent |
| `WORK_HUMAN` | Display name of the operator |
| `WORK_API_KEY` | Bearer token for global endpoints |
| `WORK_API_TOKEN` | Bearer token for workspace-scoped endpoints (falls back to `WORK_API_KEY`) |
| `PORT` | HTTP listen port (default 8080) |
