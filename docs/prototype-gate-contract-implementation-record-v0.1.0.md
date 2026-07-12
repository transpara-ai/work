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
`bc021d916c5026305d92f9b19fcff61389215e09d0822ae2fd402ae06f958950`.
That contract pins Docs standard 4.4.0 at blob
`419ab7339863923dd1f3bc4e814d9f64f29c08ba`. The Work fixture has SHA-256
`2fa4f79d5db9359aa962f6a2aaacee856d9c864f4b8d0aa1f7cde2ad086d80a6`.

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
- the prototype exemption remains inert until Docs PR #274 default-branch
  canonicality is separately proven.

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
