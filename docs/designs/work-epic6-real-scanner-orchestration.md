# Work Epic 6 Real Scanner Orchestration

This design note documents the bounded Gate G fixture implemented by
`RunEpic6RealScannerOrchestrationTrial`.

The fixture is local and Work-owned. It generates SaaS Template v1 into the
caller-provided working directory, invokes real scanner/checker commands through
an explicit command runner, writes normalized security gate evidence, records
EventGraph v3.9 evidence, and certifies only the bounded local fixture when the
critical/high policy passes.

The positive path records:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- local scanner `ActorInvocation`, `RuntimeEnvelope`, and `RuntimeResult`
- generated target, runtime BOM, security policy, report, and proof artifacts
- `CapabilityArtifact` usage for the scanner orchestration fixture
- `KnowledgeReference` evidence for merged docs PR #85 and reviewed head
- `TestCase`, `TestRun`, `GateResult`, `ReleaseCandidate`, `Certification`,
  and `AuditReport`
- command evidence for `gitleaks`, `osv-scanner` over the generated
  `frontend/package-lock.json` and `backend/requirements.lock.txt`, `semgrep`,
  local `license-policy`, local `auth-flow-check`, local
  `config-security-check`, and explicit `trivy` not-applicable proof when no
  container/build artifact exists

The negative modes and focused tests prove Gate G failure behavior:

- `Epic6ScannerOrchestrationMissingScanner` rejects a missing real scanner
  binary.
- A scanner command that exits successfully without producing non-empty raw
  output still rejects Gate G.
- `Epic6ScannerOrchestrationOpenCritical` rejects critical findings even with a
  waiver.
- `Epic6ScannerOrchestrationOpenHigh` rejects high findings without a valid
  waiver.
- `Epic6ScannerOrchestrationHighWaived` permits only local non-production
  certification for a high finding with a valid waiver.
- `Epic6ScannerOrchestrationCommittedSecret` rejects committed secret findings
  even when marked waived.

The fixture does not implement Gate H, Gate I, Gate J, protected execution,
protected side effects, runner/worktree protected execution, policy-adapter
reliance, production autonomy, production deployment, auto-merge, default-branch
push, upstream push, or non-Work repository behavior.
