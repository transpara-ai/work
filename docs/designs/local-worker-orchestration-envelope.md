# Local Worker Orchestration Envelope

Source issue: `transpara-ai/work#87`

## Purpose

This design records the reference-only local-worker orchestration envelope for
the future Work runtime path. It cites `transpara-ai/work#87` as the worked
example intake record and applies the required correction: command admission is
by explicit allowlist, never by heuristic denylist.

The packet is design-only. It does not implement orchestration, run
RuntimeBroker, execute local or external adapters, mutate Work state, write
EventGraph truth, call Hive, create or edit GitHub issues, deploy, restart
services, read secrets, increase autonomy, allocate value, close Test 001, or
claim production readiness.

## Source Treatment

`transpara-ai/work#87` is a source-of-intent record for pattern learning only.
The referenced Neo cage is never imported, vendored, invoked, treated as a
runtime, or treated as a gate authority or truth store. It contributes design
pressure for bounded orchestration, readable denials, role-limited tool sets,
and crash-safe transcripts. Work remains governed by the Dark Factory
RuntimeBroker envelope model and by separately reviewed Work/EventGraph
evidence contracts.

Canonical command-policy source:
`transpara-ai/docs:dark-factory/v4.0/04-production-workflow-runtime-v4.0.md`
line 287 requires commands to be allowed by explicit binary and allowed
argument patterns. This packet applies that rule to the future local-worker
orchestration shape and does not create a new policy vocabulary.

The known bad pattern is explicitly rejected: a heuristic dangerous-command
denylist is fail-open and must not be copied. Work orchestration must admit
commands only when the immutable envelope names the binary or operation, the
allowed argument shape, the allowed file scope, the network and secret policy,
the timeout, and the output contract.

## Envelope Contract

A future local-worker orchestration implementation must bind every run to one
immutable envelope before work starts. The envelope must carry:

- FactoryOrder, Task, ActorInvocation, AuthorityDecision, and source issue refs;
- runtime adapter identity and version;
- explicit allowed operations or binaries;
- explicit allowed argument patterns for each operation;
- denied-by-default behavior for any absent operation or argument shape;
- allowed file paths, denied path classes, and output paths;
- network policy, secret policy, working directory, timeout, and resource
  limits;
- expected outputs, transcript policy, validation plan, and receipt schema;
- envelope hash computed from the canonical record, not from prose.

The local worker may constrain execution, observe outputs, and return readable
policy denials. It must not interpret policy intent beyond the envelope, invent
permission from task text, or treat an agent request as authority.

## Orchestration Bounds

The first orchestration shape should be deliberately small:

- no recursive spawn unless a later governed PR adds a specific child-task
  contract;
- no inherited write access through a child worker unless the parent envelope
  grants it explicitly;
- role-specific tool sets derived from the envelope, not from prompt prose;
- bounded depth, child count, wall-clock time, per-step time, output bytes, and
  changed-file count;
- fail-closed behavior when any bound is unreadable, missing, or stale.

Denials are normal runtime evidence. A blocked command, blocked path, exhausted
budget, missing authority decision, stale envelope hash, or unreadable policy
state must produce a structured result that a caller can re-plan around. It
must not silently retry, widen the envelope, or continue with default authority.

## Transcript And Receipt Requirements

Every worker-visible operation must append evidence in an order that can be
audited after a crash:

- operation request;
- envelope and authority refs used for admission;
- admission decision;
- operation result or policy denial;
- changed files and artifacts, when any;
- validation result;
- ExecutionReceipt candidate when and only when separately authorized.

An operation request without its result is an incomplete transcript. Future
verification must treat that as blocked or failed evidence, not as success.

## Future Readiness Criteria

Implementation work is not ready from this packet alone. A future PR becomes
ready only when it has a separate source issue or FactoryOrder that names:

- exact Work files and tests to change;
- the allowed local worker commands or operations;
- the envelope schema fields to persist or project;
- the failure cases to prove, including absent allowlist, argument mismatch,
  path escape, recursive spawn attempt, budget exhaustion, missing authority,
  and transcript interruption;
- whether the implementation is pure evaluation, local deterministic fixture,
  or a separately authorized runtime path.

## Validation Expectations

A future implementation PR must include at least:

```text
git diff --check
go test ./... -run 'Runtime|Worker|Envelope|Policy'
go test ./...
go vet ./...
make verify
```

Design-only PRs that update this packet must run `git diff --check` and the
repo validation target unless the PR body records why code validation was not
applicable.

## Authority Boundary

This design closes only the `work#87` reference-placement requirement by
citing it from the local-worker orchestration arc. It does not authorize
runtime execution, RuntimeBroker execution, external adapter invocation,
production EventGraph writes, Hive wake/start/action APIs, Work runtime writes,
GitHub mutation, deploy, service restart, private fetch, protected settings
changes, Test 001 GREEN, production go-live, value allocation, autonomy
increase, residual-risk closure, or wiki work.
