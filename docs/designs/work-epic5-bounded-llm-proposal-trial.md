# Work Epic 5 Bounded LLM Proposal Trial

This design note documents the bounded Gate F fixture implemented by
`RunEpic5BoundedLLMProposalTrial`.

The fixture is local and review-only. It uses a committed static transcript to
represent a recorded LLM proposal, writes prompt/response/proposal artifacts only
inside the caller-provided working directory, records EventGraph v3.9 evidence,
and leaves the proposed patch unapplied.

The positive path records:

- `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, and `Task`
- recorded `ActorInvocation` with model/provider labels and prompt/response hashes
- prompt, response, proposal, and proposed patch `Artifact` records
- proposed-only `CodeChange`
- `CapabilityArtifact` and `USED_CAPABILITY` evidence for the prompt section
- `KnowledgeReference` evidence for the merged Gate F authorization packet
- `AuthorityRequest`, `AuthorityDecision`, and `HumanApproval`
- `TestCase`, `TestRun`, `GateResult`, `ReleaseCandidate`, `Certification`, and `AuditReport`
- a proof-of-work projection that displays the LLM contribution, proposed diff,
  validation, review, audit, authority, influence evidence, and negative
  non-execution evidence

The negative modes prove Gate F failure behavior:

- `Epic5LLMProposalMissingInvocation` rejects the gate when `ActorInvocation`
  evidence is absent.
- `Epic5LLMProposalAppliedPatch` rejects the gate when the proposal is marked
  applied instead of proposed-only.

The fixture does not call a live model provider, use OpenRouter/Pi/model-broker
code, access the network, read secrets, invoke an external runtime adapter,
produce an `ExecutionReceipt`, mutate another repository, push, merge, deploy,
or implement Gate G through Gate J behavior.
