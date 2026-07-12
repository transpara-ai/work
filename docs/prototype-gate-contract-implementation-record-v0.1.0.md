---
doc_id: WORK-PROTOTYPE-GATE-CONTRACT-IMPLEMENTATION
title: Work Prototype Gate Contract Implementation Record
doc_type: implementation-record
status: implemented
version: 0.1.0
created: 2026-07-12
updated: 2026-07-12
owner: Michael Saucier
steward: Codex
project: work
maturity: prototype
governance_class: governed_protected
prototype_reason: ""
related_docs:
  - FO-WORK-93-PROTOTYPE-GATE-CONTRACT
  - WORK-PROTOTYPE-GATE-CONTRACT-DESIGN
  - transpara-ai/work#93
  - transpara-ai/platform#66
  - transpara-ai/docs#272
canonical: false
---

# Work Prototype Gate Contract Implementation Record

Work vendors Platform contract `two-axis-prototype-gates/v1` version `1.0.0`
at SHA-256
`e5b838a3c60efc9fdc1c23c8e3a908d58f1774af95b653813f6b031500d6ec28`.
That contract pins Docs standard 4.4.0 at blob
`c9975c6d5bf703e58dd28ecbe9f7cf91b5ae6b96`. The Work fixture has SHA-256
`42801a522730b4578b26e91c4e8fc4c537e8a916b8fcccc5eda131c259409910`.

Implemented behavior:

- only a reason-bearing `non_governed_prototype` may use the design-gate
  exemption;
- one `cfada.skipped` transition records both IADA and CFADA as skipped with
  the same reason;
- partial/mismatched design-skip projections fail construction and replay;
- Authorize, IAR, and CFAR skipped projections are invalid for every class;
- `authority.skipped` remains recognizable on the wire but has no legal
  transition and fails replay closed;
- maturity is absent from `skipAllowed` and cannot affect the decision;
- CFAR remains required and exact-head bound.
- the contract declares that the prototype exemption is externally gated on
  Docs PR #274 default-branch canonicality. Work does not query GitHub at
  runtime; merge order and the governed controller enforce activation. The
  existing narrow prototype transition remains locally callable, so this is a
  process gate rather than a runtime Boolean claim.

Validation on 2026-07-12:

- `go test ./pkg/worklifecycle/...` — pass;
- `go test ./...` — pass;
- `GOWORK=<temporary-work93-hive-workspace> go test ./pkg/hive` from the
  canonical Hive checkout — pass against this Work worktree; neither repo was
  modified;
- local canonical Hive Postgres inventory — zero persisted
  `authority.skipped` occurrences, with 45 `agent.authority.granted` events
  as a positive-control category.

The local database has no Work lifecycle state/snapshot table, and repository
search found `UnitState` only inside `pkg/worklifecycle` and its tests. The
type has unexported fields and no JSON/storage adapter; lifecycle state is
derived by `Fold` from events. For this implementation, zero
`authority.skipped` events therefore implies zero persisted skipped-Authorize
`UnitState` snapshots. External deployments must still inventory any custom
snapshot store separately.

External event stores were not inventoried. Any deployment with a historical
`authority.skipped` occurrence requires an explicit migration design before
adopting this projection. This record grants no merge, runtime migration,
EventGraph mutation, deploy, autonomy, or issue-closure authority.
