# FactoryOrder Proposal Evidence Path

This design records the Work-side implementation for Dark Factory v4.0 Event 7.
It is intentionally a pure proposal-evidence seam. It does not run
RuntimeBroker, write native EventGraph truth, call GitHub, mutate tasks, deploy,
change protected settings, or claim production readiness.

## Source Authority

- Docs authority: `DF-V4.0-EPIC-007-AUTHORIZATION`
- Merged docs PR: `transpara-ai/docs#158`
- Authorized Work base at authoring: `33750f7ca7aaab9bd0f8ba83e8835b99343164b7`
- Allowed Work paths:
  - `factory_order_proposal.go`
  - `factory_order_proposal_test.go`
  - `docs/designs/factory-order-proposal-evidence-path.md`

The authority is single-use for one future Work PR lifecycle. Gate Q remains
open until Work implementation evidence exists and a separate governed docs PR
records the closeout. This Work PR cannot update docs state or self-close Gate
Q.

## Contract

`BuildFactoryOrderDevelopmentProposal` accepts caller-supplied source intent,
target repo/head, FactoryOrder linkage IDs, changed-file intent, validation
plan, optional GitHub issue source records, and protected-action boundaries. It
returns structured values:

- FactoryOrder source summary
- normalized GitHub issue source records, when supplied by the caller
- Requirement and AcceptanceCriterion records
- Task draft, including source issue refs, assumptions, ambiguities, risk notes,
  and explicit no-start/no-mutation flags for issue-derived proposal evidence
- proposed-only changed-file intent
- unapplied proposal artifact
- unavailable validation result
- proof-of-work packet, including full GitHub issue source records and direct
  `source_issue_refs` for deterministic issue-to-evidence traceability
- AuditReport-shaped recommendation

The builder is pure. It has no store, filesystem, GitHub, runtime, EventGraph,
or command interface. Only structured status fields are authoritative; caller
supplied `summary` text is explanatory and must not be interpreted as authority,
execution, release, or production status.

When GitHub issue source records are supplied, the builder treats them as
caller-provided scanner evidence. It does not fetch GitHub, edit issues, start
tasks, mutate Work state, or infer authority. It derives proposal-only
requirement and acceptance text from the records and marks the resulting
production-cell task draft with `implementation_started: false` and
`work_mutation_status: none`.

## Fail-Closed Rules

The builder rejects:

- missing or wrongly prefixed FactoryOrder, Requirement, AcceptanceCriterion, or
  Task IDs
- target repositories other than `transpara-ai/work`
- issue source records missing repo, positive issue number, or title
- empty changed-file intent
- changed-file intent that is not `proposed_only: true`
- changed-file intent marked `applied: true`
- runtime invocation references
- ExecutionReceipt references
- native EventGraph write references
- protected-action claims
- authority-boundary statuses outside the explicit non-authorizing allowlist:
  `not_authorized`, `deferred`, `pending`, `blocked`, `unavailable`, or
  `requires_authority`

Branch, pull request, CI, RuntimeBroker, ExecutionReceipt, and native EventGraph
write evidence are recorded as unavailable in the proof-of-work packet. A later
authorized workflow can attach those facts externally; this builder does not
create or mutate them.

## Merge Boundary

The Event 7 docs packet requires a future Work merge precondition. Before any
merge, the PR must record either:

1. live `transpara-ai/work` branch-protection evidence requiring PR review, at
   least one human approving review on the exact merge head with stale approvals
   dismissed or revalidated, and `cross-family-adversarial-review` as a required
   exact-head status, or
2. explicit PR-visible External Committee approval authorizing merge of that
   exact Work PR head.

At this PR's authoring, `transpara-ai/work` main requires `Build & Test` and
`cross-family-adversarial-review`, but has `required_approving_review_count: 0`,
so option 1 is not satisfied. Without option 1 or explicit option 2 evidence,
the Work PR must stop before merge.
