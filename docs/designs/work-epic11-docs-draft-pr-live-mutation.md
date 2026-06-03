# Work Epic 11 Docs Draft PR Live Mutation

This design note documents the bounded Epic 11 implementation authorized by
merged `transpara-ai/docs#95`.

The implementation is Work-owned and exposes
`RunEpic11DocsDraftPRLiveMutation`. The only live side-effect interface is
`Epic11PullRequestCreator.CreateDraftPullRequest`, which production callers
must back with a GitHub client and tests back with a fake.

## Authorized Action

```text
pull_request.create
```

Allowed target:

```text
repository: transpara-ai/docs
base: main
head: existing codex/* branch on origin
draft: true
changed files: dark-factory/** only
maximum PRs per run: 1
```

The implementation does not push branches. The head branch and both base/head
SHAs must already be known before the authority request is created.

## Fail-Closed Guards

The GitHub client is not called unless all of the following match exactly:

- target repository, base ref/SHA, and head ref/SHA
- title hash and body hash
- changed-file list and validation evidence refs
- requested actor and actor role
- single-use nonce and expiry
- `AuthorityRequest`
- approved `AuthorityDecision`
- `PolicyEngineAdapterDecision`
- `policy_bundle_id=df-v3.9.20-docs-draft-pr-create-only`
- the canonical policy bundle hash from `Epic11DocsDraftPRPolicyBundleHash`
- manual rollback instructions
- no prior durable authority reservation or `ExecutionReceipt` refs for the
  same authority decision or single-use nonce

The policy canonical decision must be `approval_required`, preserving the
human-gated action vocabulary already used by EventGraph v3.9.

Before the GitHub client is called, Work records an
`epic11_docs_draft_pr_authority_reservation` task artifact under an
in-process reservation lock. Later attempts with the same authority decision
or nonce fail closed on that reservation even if the original client call
failed before a post-confirmation receipt could be recorded.

## Receipt Boundary

`ExecutionReceipt` evidence is created only after the pull-request client
returns an open draft PR response whose repository, base ref/SHA, head ref/SHA,
state, URL, and response ID match the target. Failed guards return before the
GitHub client is called. A non-draft or mismatched GitHub response is rejected
and does not produce successful receipt evidence.

The receipt records:

- authority request ref
- authority decision ref
- policy decision ref
- actor ID and role
- action
- target repo, base ref/SHA, and head ref/SHA
- `draft=true`
- PR number and URL
- title/body hashes
- GitHub response ID or equivalent
- result
- timestamp
- validation evidence refs
- manual rollback instructions

## Forbidden Actions

This implementation does not authorize or implement:

- `pull_request.ready_for_review`
- `pull_request.merge`
- `pull_request.update`
- `pull_request.close`
- `pull_request.request_review`
- `issue.comment`
- `label.mutate`
- `branch.push`
- `repo.push.default_branch`
- `repo.merge.main`
- `worktree.merge.main`
- `production.deploy`
- `secret.access`
- `capability.activate`
- `runtime.invoke.external`
- `repo.mutate.cross_repo`
- `upstream.push`
- automatic rollback mutation

Rollback remains a manual operator instruction unless a later docs packet
selects and authorizes a separate rollback mutation class.

## Evidence Model

The run records EventGraph v3.9 evidence with the existing schema:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- `ActorIdentity` and `ActorInvocation`
- `RuntimeEnvelope` and `RuntimeResult`
- `AuthorityRequest`, `AuthorityDecision`, and `HumanApproval`
- `PolicyEngineAdapterDecision`
- post-confirmation `ExecutionReceipt`
- `FactoryRuntimeVersion`, `TestCase`, `TestRun`, `GateResult`,
  `ReleaseCandidate`, `Certification`, and `AuditReport`
- `KnowledgeReference` for merged `transpara-ai/docs#95`

Work also records a pre-call task artifact for the authority reservation. That
artifact is Work-local durable evidence; it does not require an EventGraph
schema change.

No EventGraph schema changes are required.
