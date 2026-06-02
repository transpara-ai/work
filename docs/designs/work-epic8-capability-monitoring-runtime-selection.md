# Work Epic 8 Capability Monitoring Runtime Selection

This design note documents the bounded Gate I fixture implemented by
`RunEpic8CapabilityMonitoringRuntimeSelectionTrials`.

The fixture is local, Work-owned, and evidence-only. It writes deterministic
monitoring-window, rollback-decision, and proof-of-work JSON artifacts into the
caller-provided working directory, records EventGraph v3.9 evidence with the
existing schema and helper semantics, and certifies only the bounded
`Epic8CapabilityMonitoringLocalEvidence` mode.

The five monitoring-window runs are:

- `trial_1_docs_only_proposal_candidate_success`
- `trial_2_bounded_code_change_candidate_success`
- `trial_3_bug_fix_repair_candidate_success`
- `trial_4_intentional_regression_triggers_rollback`
- `trial_5_post_rollback_baseline_selection_success`

The positive path records:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- local monitoring `ActorInvocation`, `RuntimeEnvelope`, and `RuntimeResult`
- `CapabilityArtifact` with `usage_logging_required=true`
- baseline and candidate `CapabilityVersion` records
- candidate promotion evidence through `EvolutionOrder`, `EvalDataset`,
  `OptimizationRun`, `CandidateVariant`, `BenchmarkResult`, `HumanReview`, and
  local side-effect-free `capability.promote` authority evidence
- canary `ActivationPolicy` and candidate `FactoryRuntimeVersion` before
  rollback
- `RollbackRecord` after the configured regression trigger
- post-rollback `FactoryRuntimeVersion` that omits the rolled-back candidate
- governed operator rollback `AuthorityRequest`, `AuthorityDecision`, and
  `HumanApproval`
- `KnowledgeReference`, `TestCase`, `TestRun`, `GateResult`,
  `ReleaseCandidate`, `Certification`, and `AuditReport`
- a proof-of-work packet that displays monitoring counters, runtime selection,
  rollback decision, residual-risk exclusions, candidate reselection blocking,
  and no-global-activation proof

The negative test seams prove Gate I rejection behavior for missing monitoring
window evidence, missing candidate `CapabilityVersion` promotion evidence,
missing rollback trigger evidence, missing operator rollback authority,
forbidden `scope=global`, and missing post-rollback candidate reselection
probe.

The fixture does not call GitHub APIs, create live pull requests, push
branches, merge, deploy, execute protected runner/worktree work, perform real
protected side effects, rely on `PolicyEngineAdapterDecision` or policy-bundle
evidence, change EventGraph/Site/Hive/Agent/docs code, perform global
activation, implement production autonomy, or advance Gate J.

The EventGraph promotion helper requires an `ExecutionReceipt` for
`capability.promote` authority evidence. The fixture records that receipt only
as side-effect-free local EventGraph evidence for candidate promotion; it does
not claim an `ExecutionReceipt` production path or real protected-action
execution.
