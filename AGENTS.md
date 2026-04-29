# AGENTS.md

## Purpose
Layer 1 Work Graph: event-sourced task management for humans and agents on the same causal graph.

## Commands
- Build: `make build`
- Test: `make test`
- Coverage: `make cover`
- Vet: `make vet`
- Verify: `make verify`

## Rules
- State is derived from events; do not introduce hidden mutable state.
- Preserve task dependency, blocked, unblocked, assignment, comment, and completion semantics.
- Keep workspace-scoped behavior and authorization paths explicit.
- Do not push to `upstream`; `origin` is the writable fork.

## Exit Criteria
- `make verify` passes, or the blocker is explicit.
- Event behavior has tests for replay and derived state.
- API or CLI behavior changes are documented.
