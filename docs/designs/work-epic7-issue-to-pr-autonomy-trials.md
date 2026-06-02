# Work Epic 7 Issue-to-PR Autonomy Trials

This design note documents the bounded Gate H fixture implemented by
`RunEpic7IssueToPRProposalTrials`.

The fixture is local, Work-owned, and proposal-only. It reads deterministic
issue fixtures, writes local pull-request proposal packets and proof-of-work
packets into the caller-provided working directory, records EventGraph v3.9
evidence with the existing schema, and certifies only the bounded
`pull_request.propose` behavior.

The five trial paths are:

- `trial_1_docs_only_issue_to_pr_proposal`
- `trial_2_bounded_code_change_issue_to_pr_proposal`
- `trial_3_bug_fix_with_tests_and_repair_proposal`
- `trial_4_multi_repo_proposal_requires_explicit_authority`
- `trial_5_self_improvement_proposal_human_reviewed_rollback_bound`

The positive path records:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- local proposal `ActorInvocation`, `RuntimeEnvelope`, and `RuntimeResult`
- issue fixture, proposal packet, proof packet, PR body, branch plan, patch,
  validation plan, repair, and rollback artifacts
- proposed-only `CodeChange`
- `CapabilityArtifact` and `USED_CAPABILITY` evidence for the local proposal
  generator
- `KnowledgeReference` evidence for merged docs PR #87 and reviewed head
- `AuthorityRequest`, `AuthorityDecision`, and `HumanApproval` records for
  `pull_request.propose` plus the separated forbidden actions
- `TestCase`, `TestRun`, `GateResult`, `ReleaseCandidate`, `Certification`,
  and `AuditReport`
- a proof-of-work projection that displays issue fixture, proposed PR
  title/body/branch, changed-file intent, diff refs, validation plan,
  repair/rollback evidence, authority boundary, forbidden-action separation,
  and residual-risk exclusions

The negative test seams prove Gate H failure behavior for missing issue
fixtures, missing proposal/proof packets, applied patches, live PR creation,
branch push, default-branch push, PR merge, deploy, protected execution,
`ExecutionReceipt`, missing multi-repo authority, and missing self-improvement
human review or rollback evidence.

The fixture does not call GitHub APIs, create live pull requests, push branches,
merge, deploy, execute protected actions, access secrets, rely on
`PolicyEngineAdapterDecision`, produce an `ExecutionReceipt` in the certified
path, mutate another repository, implement capability monitoring, activate a
capability, or implement Gate I / Gate J behavior.
