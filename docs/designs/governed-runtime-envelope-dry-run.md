# Governed Runtime Envelope Dry Run

This design records the Work-side implementation for Dark Factory v4.0 Event
11. It is a deterministic local RuntimeBroker dry-run fixture. It does not call
external runtimes, execute a shell, reach the network, read secrets, write
production EventGraph truth, deploy, change protected settings, mutate Hive,
Site, Agent, or wiki state, allocate value, close Gate U, or claim production
readiness.

## Source Authority

- Docs authority: `DF-V4.0-EPIC-011-AUTHORITY-DECISION`
- Merged docs PR: `transpara-ai/docs#180`
- Docs merge commit: `e6fc5e65305e4ef17b110c1952fc4ce91bf938ff`
- Reviewed docs PR head: `83bd5ae1cf183ff31ef67e265ba658c8617d8679`
- Allowed Work paths:
  - `event11_runtime_envelope_dry_run.go`
  - `event11_runtime_envelope_dry_run_test.go`
  - `docs/designs/governed-runtime-envelope-dry-run.md`

The authority is single-use for one Work PR lifecycle. The Work PR may only
produce Level 0 deterministic RuntimeBroker dry-run evidence. It still requires
exact-head adversarial review and explicit PR-visible External Committee
approval before merge. Gate U remains open until a separate governed docs
evidence-decision PR records the result.

## Contract

`RunEvent11RuntimeEnvelopeDryRunFixture` creates one Work task from a local
FactoryOrder, transitions it through the bounded dry-run lifecycle, executes a
local deterministic runtime envelope through `TaskStore.RunLocalRuntime`, and
projects the evidence into a local in-memory
`github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39` store.

The successful packet records:

- FactoryOrder, Requirement, AcceptanceCriterion, and Task
- ActorIdentity, ActorInvocation, AuthorityRequest, AuthorityDecision, and
  ExecutionReceipt
- RuntimeEnvelope and RuntimeResult
- Artifact and CodeChange
- TestCase, TestRun, GateResult, TraceCompletenessGate result, and
  ActorAuthorityRequestDecisionReceipt path
- FactoryRuntimeVersion, ReleaseCandidate, Certification, and AuditReport

The fixture also writes a Work task artifact named
`event11_runtime_evidence`. That artifact is JSON report evidence for this
local fixture only; it is not a production receipt.

## Output Schema And EventGraph Handoff

The `event11_runtime_evidence` JSON artifact includes explicit machine-readable
handoff fields so downstream projections do not infer state from prose:

- `trace_output` records the trace status, completion flag, FactoryOrder,
  ReleaseCandidate, TestRun, GateResult, AuditReport, required-path count,
  evidence refs, missing evidence, and output status.
- `gate_output` records output status, the Gate U fixture gate name,
  `gate_scope: fixture_only`, `gate_u_closure_claimed: false`, FactoryOrder,
  ReleaseCandidate, TestRun, GateResult, AuditReport, evidence refs, and
  missing evidence.
- `eventgraph_handoff` records the EventGraph boundary. A passing fixture uses
  `status: local_fixture_projection_complete`, `projection_scope:
  work_local_in_memory_v39_fixture`, `persistent_write_status: not_written`,
  `persistent_write_claimed: false`, `production_truth_claimed: false`,
  `runtime_execution_scope: local_deterministic_fixture_only`, EventGraph refs
  for records that actually exist in the local fixture, authority refs, and
  notes. A failing fixture uses `status: blocked`, carries the blocking missing
  evidence in `blocked_by`, and omits refs for missing record families.

The handoff is intentionally not a persistent EventGraph write. It is a typed
Work artifact projection over the local in-memory v3.9 fixture. Durable
EventGraph writes, production truth, and runtime evidence ingestion remain
separate governed EventGraph/Hive work.

The in-memory `AuthorityDecision` follows the existing v3.9 convention of
`status: approved` with `decision: ApprovalRequired` to show that local fixture
evidence may be built under the bounded Event 11 authority while the Work PR
itself still requires standalone exact-head External Committee approval before
merge.

## Local Runtime Boundary

The RuntimeEnvelope records:

- `runtime_adapter_id: local_deterministic`
- `network_policy: disabled`
- `secrets_policy: none`
- `working_directory: fixture://work/event11-runtime-envelope-dry-run`
- allowed files limited to `report.txt`
- denied commands including shell execution, network access, secret access,
  direct default-branch push, PR merge, deploy, production operation, and value
  allocation

The envelope hash is computed from the persisted v3.9 RuntimeEnvelope record
with its own `envelope_hash` field blanked. This makes the hash deterministic,
machine-independent, and recomputable from the evidence record. The live
execution still uses a real filesystem working directory; the persisted
evidence canonicalizes that path to the fixture URI above.

Callers must pass an ephemeral fixture directory as `WorkingDir`. The function
creates that directory when needed and the local deterministic runtime writes
only its allowed fixture outputs beneath it. It is not a general-purpose runner
for arbitrary project roots.

The RuntimeResult records empty network and secret access logs. The
ExecutionReceipt action is limited to
`runtime.invoke.local.event11_dry_run_fixture`. It receipts only the local
fixture task and cannot receipt a protected action.

## Negative Cases

The fixture runs policy cases for:

- denied command
- path traversal
- network attempt
- secret attempt
- timeout
- validation failure

Each case must either be policy-blocked, timed out, or validation-failed as
expected, and it must prove no unauthorized side effect outside the fixture
boundary.

## Fail-Closed Rules

The fixture exposes negative-test seams that omit or widen individual evidence
families. Missing ExecutionReceipt, RuntimeResult, CodeChange, AuditReport,
policy cases, or envelope hash evidence causes the report to fail, prevents
Certification, records a Rejection, and leaves the AuditReport incomplete when
an AuditReport is present.

The report also fails if:

- TraceCompletenessGate is incomplete
- AuthorityRequest, AuthorityDecision, and ExecutionReceipt are not linked
- required v3.9 record families are absent
- local-only runtime policy fields are widened
- RuntimeResult records network or secret access
- RuntimeResult changed files escape `report.txt`
- forbidden action statuses are anything other than `not_run` or `not_claimed`

## Explicit Non-Claims

This Work fixture does not close Gate U and does not claim:

- production readiness or go-live
- external RuntimeBroker adapter execution
- general shell execution
- persistent EventGraph writes
- protected settings mutation
- default-branch mutation
- deploy authority
- Level 1, Level 2, or Level 3 achievement
- autonomy increase
- value allocation
- v3.9 mutation or archival
- docs#172 closure
- R-001, R-002, or R-003 closure

Those remain governed future events requiring separate authority.

## Validation

The PR must record:

```text
git diff --check
git diff --name-status
go test ./... -run Event11
go test ./...
go vet ./...
make verify
negative search over changed files for external runtime, shell, network, secret,
deploy, protected action, default-branch, production, and value allocation terms
exact-head draft cross-family adversarial review
ready-state exact-head cross-family adversarial review
PR-visible finding disposition
cross-family-adversarial-review status on the exact reviewed head
explicit PR-visible External Committee approval before merge
```
