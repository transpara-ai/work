# Work Epic 9 Golden PRD Product Factory Run

This design note documents the bounded Gate J implementation fixture in
`transpara-ai/work`.

## Scope

The fixture is `RunEpic9GoldenPRDProductFactoryRun` with mode
`Epic9GoldenPRDLocalDryRun`. It implements only the local/dry-run Gate J packet
authorized by `transpara-ai/docs#91`.

The selected golden PRD is `simple CRUD tracker`, sourced from
`dark-factory/v3.9/04-production-workflow-and-runtime-v3.9.md`. The local
fixture records this source by canonical path plus a deterministic locator hash
derived from the PRD name and source reference; it does not claim a
cross-repository document-body digest.

## Evidence Path

The fixture records:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- `ActorInvocation`, `RuntimeEnvelope`, and `RuntimeResult`
- source-intent, generated-template, security-report, runtime-BOM,
  deploy-preview dry-run, proof-of-work, and audit artifacts
- `TestCase`, `TestRun`, and `GateResult`
- `FactoryRuntimeVersion`
- release authority request, authority decision, and human approval
- `ReleaseCandidate`
- `Certification` or `Rejection`
- `AuditReport`
- `KnowledgeReference` and local capability-usage evidence required by the
  current EventGraph v3.9 certification eligibility path, with the docs#91
  merge/reviewed-head locator recorded as an immutable locator rather than a
  digest

The generated product is SaaS Template v1, written only into the caller's
working directory. The deploy preview is a dry-run text artifact only.

## Rejection Paths

Focused tests cover missing `FactoryOrder`, missing PRD/source-intent evidence,
missing acceptance evidence, missing generated artifact evidence, missing
security-gate evidence, open critical security findings, open high findings
without a valid local waiver, missing `FactoryRuntimeVersion`, missing release
authority, and missing `AuditReport`.

## Exclusions

The fixture does not create live pull requests, push branches, merge, deploy,
run protected execution, access secrets, perform real protected side effects,
claim production `ExecutionReceipt` evidence, rely on
`PolicyEngineAdapterDecision`, activate capabilities globally, or mutate
EventGraph, Site, Hive, Agent, or docs implementation code.

R-001, R-002, and R-003 remain excluded residual risks.
