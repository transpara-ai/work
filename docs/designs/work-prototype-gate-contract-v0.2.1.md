---
doc_id: WORK-PROTOTYPE-GATE-CONTRACT-DESIGN
title: Work Prototype Gate-Contract Enforcement Design
doc_type: design
status: review
version: 0.2.1
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
  - transpara-ai/work#93
  - transpara-ai/docs#272
  - transpara-ai/platform#66
canonical: false
---

<!-- df:artifact id=WORK-PROTOTYPE-GATE-CONTRACT-DESIGN type=design version=0.2.1 status=review -->
<!-- df:scope worklifecycle prototype governance-class iada cfada cfar compatibility no-reviewer-runtime no-deploy no-merge -->

# Work Prototype Gate-Contract Enforcement Design

## 1. Current behavior

`pkg/worklifecycle/types.go` defines four governance classes and five gates.
The public `GatePolicyOptional` value is misleading if read without the
transition rules:

- governed/unknown work reaches `DesignAudit` only through a passing internal
  design result and reaches `AwaitingAuth` only through passing CFADA;
- the direct `cfada.skipped` transition is accepted only for
  `non_governed_prototype` with a non-empty reason;
- the legacy `authority.skipped` transition also accepts the prototype class,
  which conflicts with the adopted contract and must be retired;
- CFAR is always `required`, cannot be skipped, and Ready/Merged require the
  matching reviewed head.

The behavior is intentionally stricter than the serialized word `optional`.
The defect is missing explicit compatibility semantics and incomplete skip
projection, not a demonstrated ordinary-work bypass.

## 2. Design decisions

### D1 - Preserve wire compatibility

Keep the serialized gate-policy values in this slice. Reinterpreting or
renaming `optional` on the wire could break stored events/projections and is
unnecessary to enforce the accepted matrix. Add comments and exported
compatibility documentation stating that policy is conditional on governance
class and transition validity.

The external `authority.skipped` event kind remains parseable for replay
compatibility but has no legal successor under Platform contract
`two-axis-prototype-gates/v1`, contract version `1.0.0`. Historical
streams containing it fail closed and require an explicit migration decision;
they are not silently upgraded into Human approval.

A read-only query of the local canonical Hive Postgres event store on
2026-07-12 returned zero rows where `event_type = 'authority.skipped'` or
`content_json` contained that token (the store contained 45
`agent.authority.granted` events). Repository/data scans found only source
and doctrine references, not persisted occurrences. This proves the current
local store has no migration subjects; it does not authorize deleting the event
type or infer absence in any external store. Any other deployment must run the
same inventory before adopting the new projection.

### D2 - Bind a versioned contract fixture

Vendor a generated/read-only JSON fixture containing the approved docs artifact
id/version/blob reference, platform contract version/hash, and decision vectors.
Work tests verify its content hash against a compile-time/test constant. Runtime
does not load another repository or call the network.

The fixture is an enforcement projection, not a second semantic authority.
Changing it requires coordinated docs/platform evidence.

### D3 - Keep maturity out of exemption logic

Work's gate transition consumes governance class and reason, not maturity.
Tests introduce maturity as an external decision-vector input solely to prove
that both `prototype` and `established` maturities yield the same result for a
given governance class. If a future Work API stores maturity, it remains a
separate field and cannot feed `skipAllowed`.

### D4 - Make the design skip projection unambiguous

The existing prototype path jumps from Designing to AwaitingAuth on
`CFADASkipped`, leaving IADA failed rather than explicitly omitted. The
implementation shall represent the path so projection consumers can distinguish
an intentional approved prototype omission from a failed/unperformed internal
gate.

When the existing reason-bearing `CFADASkipped` prototype transition is
accepted, record both IADA and CFADA gate records as intentionally skipped with
the same reason while preserving their optional policy values. This is one
atomic state transition and introduces no new external event type. Golden,
replay, and Hive-consumer tests must treat the changed IADA projection from
failed to skipped as an intentional compatibility delta. If that delta breaks a
documented external contract, implementation stops and returns to design/HDR;
it must not silently choose a different representation.

### D5 - Protected precedence remains absolute

Classification continues to fail closed:

```text
unreadable/uncertain/conflicting -> unknown
protected trigger              -> governed_protected
governed-standard trigger      -> governed_standard
explicit bounded class+reason  -> non_governed_prototype
otherwise                      -> unknown
```

No maturity flag or permissive supplied class overrides protected evidence.

### D6 - Only IADA and CFADA are prototype-optional

`skipAllowed` accepts only IADA/CFADA for a reason-bearing
`non_governed_prototype`. Human Design Review (`authorize`), IAR, CFAR,
Human Review, exact-head binding, and required checks cannot be skipped. The
existing `authority.skipped` transition is removed from the legal transition
table. Construction/replay validation rejects skipped Authorize/IAR/CFAR
records even for the prototype class.

## 3. Implementation surface after HDR

- `pkg/worklifecycle/types.go`: comments, bounded skip recording/projection,
  and any helper required to evaluate the canonical decision vector;
- `pkg/worklifecycle/*_test.go`: complete matrix, invalid construction, replay,
  and CFAR invariants;
- `pkg/worklifecycle/testdata/`: pinned compatibility fixture;
- package documentation naming the docs/platform source pins.

No Hive, Agent, EventGraph, API, persistence migration, or reviewer runner
mutation is in scope. A read-only Hive consumer compile/test is mandatory as a
separate implementation-stage local verification gate, not part of Work CI and
not part of the hermetic decision-vector suite in Requirement 7. It runs from
the Hive checkout through a temporary Go workspace that binds Hive to this Work
worktree; it modifies neither repository and proves the current consumer
compiles and observes the explicit IADA+CFADA skipped projection.

## 4. Test matrix

| ID | Class | Maturity | Reason | Expected design skip | CFAR |
|---|---|---|---|---:|---|
| T1 | non_governed_prototype | prototype | non-empty | allowed and explicit | required |
| T2 | non_governed_prototype | established | non-empty | allowed and explicit | required |
| T3 | non_governed_prototype | any | blank | denied | required |
| T4 | governed_standard | prototype | any | denied | required |
| T5 | governed_protected | prototype | any | denied | required |
| T6 | unknown | prototype | any | denied | required |
| T7 | contradictory protected + prototype evidence | prototype | non-empty | class unknown/protected; denied | required |
| T8 | unreadable evidence | any | any | denied | required |
| T9 | accepted prototype skip replayed | any | non-empty | identical explicit projection | required |
| T10 | contract fixture hash/version mismatch | any | any | test/build failure | unchanged |
| T11 | prototype `authority.skipped` event or skipped Authorize/IAR record | any | any | denied/fail closed | required |
| T12 | skipped CFAR construction; Human Review shortcut | any | any | denied/fail closed | required |
| T13 | missing vs unreadable vs uncertain classification | any | any | each resolves unknown; denied | required |
| T14 | Hive consumer test against IADA+CFADA skipped projection | any | non-empty | compile and projection assertion pass | required |

Existing transition, invalid-state, canonical-state, and merged-head tests must
remain green.

## 5. Satisfied-only-when predicate

```text
wire compatibility is preserved or explicitly migrated
AND maturity cannot affect skipAllowed
AND only reason-bearing non-governed class can skip
AND only IADA/CFADA may carry a skipped projection
AND Human Design Review and IAR cannot be skipped
AND protected/standard/unknown/contradictory inputs deny skip
AND IADA+CFADA omission is projected unambiguously
AND CFAR remains required and exact-head-bound
AND Work pins and executes the canonical decision vectors
AND a read-only Hive consumer compile/projection assertion passes
AND full tests, IAR, and exact-head CFAR pass
```

## 6. Risks

- Marking IADA skipped may change projection expectations; golden and replay
  tests must make the intended delta explicit.
- A copied fixture can drift; version/hash checks and coordinated PR evidence
  prevent silent changes.
- `optional` remains easy to misread; comments and exported projection semantics
  mitigate this without a wire migration.
- Hive consumes `CanonicalWorkState`; a read-only Hive compile/consumer check
  is a mandatory acceptance gate for the projection delta.

## 7. IADA record

The author-side assessment is recorded under
`.adversarial-design/20260712-work-93-iada/iada.result.json`. It is valid only
when that artifact names this packet's exact final blob SHA.

Version 0.1.1 repairs the assessment ambiguity by choosing one atomic skip
projection, naming its compatibility delta, and requiring a return to HDR if a
documented consumer contract rejects that delta.

Version 0.2.0 repairs the contract mismatch discovered during pre-code
inspection: prototype classification cannot skip Human Design Review or IAR.
The legacy event remains recognizable but becomes an invalid transition and
historical occurrences require an explicit migration decision. The local
canonical store inventory found zero occurrences, and Hive consumer validation
is mandatory.

Version 0.2.1 resolves final CFADA consistency findings by defining the
Platform contract identifier/version, specifying Hive verification as a
separate non-mutating local gate, and naming that gate in the
satisfied-only-when predicate.

## 8. Authority boundary

This design does not authorize implementation before HDR, Hive/runtime wiring,
external model invocation, deploy, EventGraph mutation, autonomy increase,
merge, or closure of platform#62.
