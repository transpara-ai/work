# Work Epic 2 Thin Factory Vertical Slice

Date: 2026-05-19

Source authorization: `transpara-ai/docs#64`, merged as the Epic 2 human-selection and Gate B disposition artifact.

## Scope

`RunEpic2ThinFactoryVerticalSlice` is the Work-owned local/dry-run fixture for Dark Factory Epic 2. It creates a Work v3.9 task, runs the existing local deterministic runtime broker, records an in-memory Dark Factory v3.9 EventGraph trace, and emits a JSON projection matching Site `/ops/evidence`.

The fixture has two modes:

- `Epic2ThinFactoryCertified`: the complete happy path, ending in a `Certification`.
- `Epic2ThinFactoryRejected`: the negative path, ending in a `Rejection` because repair evidence is intentionally absent for a failing gate.

## Boundaries

The fixture is limited to Level 0 to Level 1 behavior:

- local deterministic runtime only
- generated text artifact only
- network disabled
- secrets unavailable
- no external runtime adapter
- no protected side effects
- no Hive, Agent, or Site execution behavior
- no production deploy
- no auto-merge
- no cross-repo mutation

Gate B remains fixture-specific non-applicable because the fixture records no `CapabilityArtifact`, no `USED_CAPABILITY` edge, no capability source reference, and no capability usage logging evidence.

## Evidence

The certified mode records the required path:

```text
FactoryOrder -> Requirement -> AcceptanceCriterion -> Task
Task -> RuntimeEnvelope -> RuntimeResult
Task -> Artifact
Task -> TestCase -> TestRun -> GateResult
FactoryOrder/ReleaseCandidate -> FactoryRuntimeVersion
ReleaseCandidate -> Certification -> AuditReport
```

The rejected mode records the same local runtime and artifact evidence, then adds a failing `GateResult` and `Failure` while intentionally omitting `RepairAttempt`. `TraceCompletenessGate` therefore fails with the missing repair edge, and the release candidate is rejected instead of certified.

## Validation

The implementation tests assert:

- certified fixture reaches Work `certified`
- rejected fixture reaches Work `rejected`
- EventGraph trace completeness passes only for the certified fixture
- rejected fixture cannot be certified
- Site-compatible proof-of-work projection JSON is produced
- capability evidence remains absent and Gate B stays non-applicable only for this fixture shape
