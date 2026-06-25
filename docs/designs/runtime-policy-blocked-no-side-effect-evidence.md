# Runtime Policy-Blocked No-Side-Effect Evidence

Source issue: `transpara-ai/work#65`

## Purpose

RuntimeBroker policy-blocked paths need deterministic evidence that a blocked
attempt did not create local side effects before any protected or external
runtime work is considered. Work records this as a pure projection over an
already-recorded `RuntimeResult`.

## Evidence Contract

`BuildRuntimePolicyBlockedNoSideEffectEvidence` returns `pass` only when:

- the runtime status is `policy_blocked`;
- the `policy_blocked` flag is true;
- the exit code is `126`;
- no timeout occurred;
- no changed files were recorded;
- no artifacts were recorded;
- no validation errors were recorded;
- at least one command log entry has status `policy_blocked`.

The projection reports counts, blocked command log entries, and failure reasons
so a proof packet or AuditReport can distinguish a clean policy block from a
partial side-effect case.

## Validation

The focused validation is:

```bash
go test ./... -run 'TestRuntimeBroker_(BuildsPolicyBlockedNoSideEffectEvidence|PolicyBlockedEvidenceFailsWhenSideEffectsExist)' -count=1
```

The repo gate remains:

```bash
make verify
```

## Authority Boundary

The evidence builder does not execute runtime commands, mutate Work state, write
EventGraph records, call Hive, access networks or secrets, deploy, increase
autonomy, allocate value, or close residual risks. It inspects caller-supplied
runtime result evidence only.
