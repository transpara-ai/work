# RuntimeBroker External Adapter Eligibility Evidence

Source issue: `transpara-ai/work#64`

## Purpose

`RunRuntimeBrokerExternalAdapterEligibilityFixture` defines the Work-side
evidence required before a future external RuntimeBroker adapter can be
considered by a separate authority decision.

This fixture is evidence-only. Its passing state means:

```text
authorization: none
requires_separate_authority_decision: true
```

It does not enable, register, invoke, smoke-test, configure, or deploy any
external adapter. It does not call `TaskStore.RunLocalRuntime`, execute shell or
general commands, use live network, use secrets, write production EventGraph
truth, wake Hive, change settings, increase autonomy, allocate value, close Test
001, close `docs#172`, or close residual risk.

## Authority

- Event 17 / Gate AA docs authority: `DF-V4.0-EPIC-017-AUTHORITY-DECISION`
- Docs PR: `transpara-ai/docs#207`
- Docs merge commit: `ad7ecdf69bf6c7f599264c216014c3f2f8ed2f8c`
- Work issue: `transpara-ai/work#64`
- Work dependency: `transpara-ai/work@040cf03af8336107cb15aaa9d2a3f6c45031011e`
- External Committee packet: `work#64` issue comment recorded after CFADA
  returned zero blockers

The authority unlocks one bounded Work PR lifecycle for adapter-eligibility
evidence only.

## Evidence Contract

The report carries typed fields for:

- authorization state;
- adapter candidate identity;
- authority refs;
- source refs;
- policy bundle refs;
- file boundary;
- command and process boundary;
- network boundary;
- secret boundary;
- timeout, cancellation, retry, and resource limits;
- artifact contract;
- exit-code mapping;
- execution receipt schema;
- validation plan;
- replay plan;
- EventGraph handoff descriptor;
- forbidden actions;
- residual risks.

`EventGraphHandoff` is a non-executing descriptor. It must report
`persistent_write_status: not_written`, `persistent_write_claimed: false`, and
`production_truth_claimed: false` on the passing path.

## Fail-Closed Rules

The report fails when any of these conditions are present:

- missing or stale authority;
- widened authority;
- missing candidate identity;
- missing or mismatched source issue ref;
- missing policy bundle;
- missing or widened file boundary;
- missing or widened command/process boundary;
- shell or general command execution claim;
- process escape claim;
- deploy, GitHub mutation, Hive action API, or RuntimeBroker execution command
  escape;
- missing or unscoped network boundary;
- widened network host scope;
- live network or validation-network claim;
- missing or unscoped secret boundary;
- credential material claim;
- secret log claim;
- missing redaction requirement;
- missing or unbounded timeout;
- missing cancellation;
- missing resource limits;
- retry without receipt;
- missing artifact contract;
- partial artifact allowance;
- missing artifact hash, content type, or size bounds;
- production artifact claim;
- missing exit-code mapping;
- ambiguous exit status;
- missing receipt schema;
- stale receipt;
- mismatched receipt hash;
- receipt recorded before result;
- missing or unbounded validation plan;
- missing replay plan;
- non-deterministic replay;
- replay requiring network or secrets;
- replay writing production state;
- adapter enablement claim;
- adapter invocation claim;
- RuntimeBroker execution claim;
- production EventGraph write claim;
- production truth claim;
- runtime side-effect claim;
- protected settings claim;
- Test 001 GREEN claim;
- `docs#172` closure claim;
- autonomy increase claim;
- value allocation claim;
- residual-risk closure claim.

## Validation

Required local validation:

```text
git diff --check
go test ./... -run 'Event17|AdapterEligibility|RuntimeBroker'
go test ./...
go vet ./...
make verify
```

Validation must remain repository-local and offline deterministic. Before merge,
the PR must also have exact-head CFAR, PR-visible finding disposition,
exact-head autonomous approval, and a merge commit.
