# Event 17 Governed Runtime Observation

Source issue: `transpara-ai/work#59`

## Purpose

Event 17 needs Work-owned runtime-observation evidence that is machine-readable
and fail-closed, without enabling external adapters or production writes.
`RunEvent17GovernedRuntimeObservationFixture` wraps the existing Event 11 local
RuntimeBroker dry-run fixture and derives a stricter governed observation
report.

This is a local fixture contract only. It does not start Hive, execute external
adapters, open a production EventGraph store, write persistent EventGraph truth,
deploy, restart services, access secrets, change protected settings, increase
autonomy, allocate value, close Test 001, close `docs#172`, or close residual
risk.

## Authority

- Event 17 / Gate AA docs authority: `DF-V4.0-EPIC-017-AUTHORITY-DECISION`
- Docs PR: `transpara-ai/docs#207`
- Docs merge commit: `ad7ecdf69bf6c7f599264c216014c3f2f8ed2f8c`
- Work issue: `transpara-ai/work#59`
- Autonomous External Committee packet: `work#59` issue comment recorded after
  CFADA returned zero blockers

The authority unlocks one bounded Work PR lifecycle for local runtime
observation evidence only.

## Evidence Contract

The Event 17 report carries typed fields for:

- pre-run RuntimeEnvelope observation;
- post-run RuntimeResult observation;
- policy decisions and policy cases;
- exact Event 17 authority references;
- trace completeness;
- TestRun, GateResult, and AuditReport evidence;
- EventGraph handoff descriptor;
- civilization-presence monitoring metadata;
- forbidden actions;
- residual risks;
- evidence refs.

RuntimeEnvelope policy fields are observed from the Event 11 in-memory
EventGraph record. The Event 17 report must fail closed if the observed
RuntimeEnvelope adapter, network policy, secrets policy, or denied-command set
widens away from the local deterministic fixture contract.

RuntimeResult access-log fields are observed from the Event 11 in-memory
EventGraph record. The Event 17 report must fail closed if network or secret
access is observed in the recorded RuntimeResult. `ChangedFiles` and
`Artifacts` remain declarative runtime evidence for the deterministic dry-run
fixture, not a live filesystem mutation claim.

`EventGraphHandoff` is a non-executing descriptor. It must report
`persistent_write_status: not_written`, `persistent_write_claimed: false`, and
`production_truth_claimed: false` on the passing path.

`CivilizationPresence` is monitoring visibility metadata only. On the passing
path it must report `status: monitoring_only` and must not assert Civilization
runtime readiness, Hive activity, Hive wake/start, issue-closure authority,
production truth, or autonomy increase.

## Fail-Closed Rules

The report fails when any of these conditions are present:

- missing authority;
- widened or mismatched authority;
- missing Event 17 authority references;
- missing pre-run envelope;
- non-local runtime adapter;
- missing runtime result;
- RuntimeResult network access observed;
- RuntimeResult secret access observed;
- missing policy decision;
- missing trace evidence;
- widened trace scope;
- missing TestRun;
- missing GateResult;
- missing AuditReport;
- stale or mismatched envelope hash;
- widened network policy;
- widened secrets policy;
- external adapter claim;
- shell or general command execution claim;
- production EventGraph write claim;
- production truth claim;
- runtime side-effect claim;
- missing civilization-presence metadata;
- malformed civilization-presence metadata;
- Civilization runtime readiness claim;
- Hive activity, wake, or start claim;
- issue-closure authority claim.
- autonomy-increase claim;
- EventGraph handoff descriptor-only invariant loss;
- EventGraph handoff persistent write invariant loss.

## Validation

Required local validation:

```text
git diff --check
go test ./... -run Event17
go test ./...
go vet ./...
make verify
```

Before merge, the PR must also have exact-head CFAR, PR-visible finding
disposition, exact-head autonomous approval, and a merge commit.
