---
doc_id: FO-WORK-93-PROTOTYPE-GATE-CONTRACT
title: Factory Order - Work Prototype Gate-Contract Enforcement
doc_type: factory_order
status: review
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
  - transpara-ai/platform#63
  - transpara-ai/platform#66
  - transpara-ai/docs#272
  - transpara-ai/work#93
canonical: false
---

# Factory Order - Work Prototype Gate Contract

## Source intent

- Audit and REC-5: https://github.com/transpara-ai/platform/issues/62
- Operator decisions: https://github.com/transpara-ai/platform/issues/62#issuecomment-4951166769
- Coordination: https://github.com/transpara-ai/platform/issues/63
- Canonical-standard work: https://github.com/transpara-ai/docs/issues/272
- Work child: https://github.com/transpara-ai/work/issues/93

The Work baseline is commit
`759d2ca` (`origin/main` on 2026-07-12). The relevant state machine currently
serializes IADA/CFADA/IAR as `optional`, CFAR as `required`, permits the CFADA
skip transition only for class `non_governed_prototype`, and denies CFAR skips.

## Order

Mechanically bind Work's lifecycle to the accepted two-axis prototype contract
without turning maturity into a gate bypass or silently breaking serialized
state compatibility.

## Requirements

1. Prototype maturity is represented independently from governance class or is
   proven to remain outside Work's gate-decision input.
2. Only `non_governed_prototype` with non-empty reason-bearing classification
   may take the IADA/CFADA prototype design-gate skip.
3. `governed_standard`, `governed_protected`, `unknown`, missing, unreadable,
   uncertain, and contradictory inputs cannot skip.
4. CFAR remains required and exact-head-bound for all classes.
5. Existing serialized `GatePolicyOptional` compatibility is preserved unless
   the approved design provides an explicit migration.
6. The projected skip evidence unambiguously states which gates were omitted
   and why; a failed gate must not masquerade as an intentional skip.
7. Work pins the canonical contract version/hash and executes the shared
   decision vectors in CI without network or cross-checkout dependency.

## Acceptance criteria and verification

Table-driven Go tests cover every class, maturity-only input, blank reason,
contradiction, prototype success, unknown failure, CFAR invariants, replay, and
contract version/hash. Full Work tests pass. The exact design and implementation
complete IADA, CFADA, HDR, IAR, CFAR, and Human Review.

## Non-goals

- invoking a reviewer runtime;
- changing Hive or EventGraph;
- weakening CFAR;
- dynamically fetching docs/platform during runtime or CI;
- deployment, autonomy, merge, or issue closure.

## Authority statement

This Factory Order states intent and grants nothing. Runtime code changes begin
only after exact-blob IADA, independent CFADA, and Human Design Review.
