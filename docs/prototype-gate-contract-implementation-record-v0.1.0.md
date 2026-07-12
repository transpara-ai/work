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
`1724ecaf487c7b6d4ab2437f545168254cd86f53b09fd78a757bd5458f0cf9a6`.
That contract pins Docs standard 4.4.0 at blob
`419ab7339863923dd1f3bc4e814d9f64f29c08ba`. The Work fixture has SHA-256
`497fcc74c57b17be9c6646cf82ac7b50f495fabf97b41a19f4c459ac004c41b0`.

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

Validation on 2026-07-12:

- `go test ./pkg/worklifecycle/...` — pass;
- `go test ./...` — pass;
- `GOWORK=<temporary-work93-hive-workspace> go test ./pkg/hive` from the
  canonical Hive checkout — pass against this Work worktree; neither repo was
  modified;
- local canonical Hive Postgres inventory — zero persisted
  `authority.skipped` occurrences, with 45 `agent.authority.granted` events
  as a positive-control category.

External event stores were not inventoried. Any deployment with a historical
`authority.skipped` occurrence requires an explicit migration design before
adopting this projection. This record grants no merge, runtime migration,
EventGraph mutation, deploy, autonomy, or issue-closure authority.
