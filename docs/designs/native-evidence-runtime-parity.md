# Native Evidence Runtime Parity Fixture

This design records the Work-side implementation for Dark Factory v4.0 Event 8.
It is a deterministic local parity fixture. It does not run RuntimeBroker, call
GitHub, execute external workers, write persistent EventGraph truth, deploy,
change protected settings, access secrets, allocate value, or claim production
readiness.

## Source Authority

- Docs authority: `DF-V4.0-EPIC-008-AUTHORITY-DECISION`
- Merged docs PR: `transpara-ai/docs#162`
- Docs merge commit: `6e18eb7df0a879ad02b1dc6bb53628d918f06377`
- Reviewed docs PR head: `d452af416f65bad2bc40f4b5be7f905963a491ba`
- Allowed Work paths:
  - `native_evidence_runtime_parity.go`
  - `native_evidence_runtime_parity_test.go`
  - `docs/designs/native-evidence-runtime-parity.md`

The authority is single-use for one Work PR lifecycle. Gate R remains open until
this Work implementation evidence exists, passes exact-head review, merges only
with explicit PR-visible External Committee approval on the exact Work PR head,
and a separate governed docs evidence-decision PR records the result.

## Contract

`BuildNativeEvidenceRuntimeParityFixture` builds a Work-owned in-memory
`github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39` store with native
records for the Event 8 evidence families:

- FactoryOrder, Requirement, AcceptanceCriterion, and Task
- ActorIdentity, ActorInvocation, AuthorityRequest, AuthorityDecision, and
  ExecutionReceipt
- RuntimeEnvelope and RuntimeResult
- Artifact and CodeChange
- TestCase, TestRun, and GateResult
- FactoryRuntimeVersion, ReleaseCandidate, TraceCompletenessGate result, and
  AuditReport

The fixture records the Event 7 `BuildFactoryOrderDevelopmentProposal` boundary
as the predecessor docs-held proposal-evidence path. That predecessor remains
proposal-only. The new fixture demonstrates that the evidence families can be
represented natively in v3.9 records; it does not call the proposal builder and
does not convert proposal evidence into production truth.

## Local-Only Boundary

The RuntimeEnvelope records:

- `network_policy: disabled`
- `secrets_policy: none`
- `working_directory: fixture://work/native-evidence-runtime-parity`
- allowed files limited to the three authorized Work paths
- denied commands including RuntimeBroker, merge, deploy, secret access,
  production operation, and value allocation

The RuntimeResult records empty network and secret access logs. The
ExecutionReceipt action is limited to
`runtime.invoke.local.native_evidence_parity_fixture`; it is local fixture
evidence only and cannot receipt a protected action.

## Fail-Closed Rules

The fixture exposes negative-test seams that omit individual native evidence
families or widen local-only boundary fields. Missing ExecutionReceipt,
RuntimeResult, CodeChange, or AuditReport evidence causes the parity report to
fail, prevents certification, records a Rejection, and leaves the AuditReport
incomplete when an AuditReport is present.

The parity report also fails if:

- TraceCompletenessGate is incomplete
- AuthorityRequest, AuthorityDecision, and ExecutionReceipt are not linked
- required v3.9 record families are absent
- local-only runtime policy fields are widened
- forbidden action statuses are anything other than `not_run`

## Explicit Non-Claims

This Work fixture does not close Gate R and does not claim:

- production readiness or go-live
- RuntimeBroker execution
- external worker execution
- persistent EventGraph writes
- protected settings mutation
- Level 1 achievement
- autonomy increase
- value allocation
- v3.9 mutation/archive
- R-001, R-002, or R-003 closure

Those remain governed future events requiring separate authority.

## Validation

The PR must record:

```text
git diff --check
git diff --name-status
go test ./...
go vet ./...
make verify
exact-head draft cross-family adversarial review
ready-state exact-head cross-family adversarial review
PR-visible finding disposition
cross-family-adversarial-review status on the exact reviewed head
explicit PR-visible External Committee approval before merge
```
