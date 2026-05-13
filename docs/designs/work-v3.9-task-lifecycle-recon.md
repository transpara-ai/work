# Work v3.9 Task Lifecycle Recon

Design-only recon for Dark Factory v3.9 Stage 3, based on `transpara-ai/docs#44`, the v3.9 kernel/runtime docs, and the current Work task/event store. No lifecycle implementation is included here.

Required sources reviewed:

- `~/transpara-ai/repos/docs/dark-factory/v3.9/02-kernel-schema-and-state-v3.9.md`
- `~/transpara-ai/repos/docs/dark-factory/v3.9/04-production-workflow-and-runtime-v3.9.md`
- `~/transpara-ai/repos/docs/dark-factory/v3.9/08-implementation-workflow-checklist-v3.9.md`
- current Work `go.mod`
- current Work `events.go`, `store.go`, and task/store/event tests
- `transpara-ai/docs#44` and its comments

## 1. Current Work model

The current Work module is `github.com/transpara-ai/work` and depends on `github.com/transpara-ai/eventgraph/go` through a local `replace ../eventgraph/go`. Work already treats tasks as EventGraph events: `TaskStore` creates signed events through `event.EventFactory`, appends them to the shared store, and derives state by replaying event types.

The current task entity is centered on `work.task.created`. Its durable creation payload contains title, description, creator, priority, and an optional workspace string. Task identity is the event ID of the creation event. There is no first-class v3.9 `Task` node schema yet, and no stored fields for FactoryOrder, EvolutionOrder, cell, requirement references, risk class, attempt count, expected outputs, or acceptance criteria.

Current operational projections include:

- list tasks globally or by workspace;
- assign tasks to actors;
- complete tasks after artifact or waiver evidence exists;
- add dependency edges between tasks;
- derive blocked/open state from incomplete dependencies;
- explicitly unblock a task with an override event;
- add comments;
- attach artifacts and artifact waivers;
- record external fact requirements and derive readiness from causal descendants or pinned event IDs;
- derive summary fields for status, assignee, blocked, artifact count, waiver, readiness, missing readiness gates, and missing facts.

Readiness is local to Work. It requires artifacts with labels `definition_of_done`, `acceptance_criteria`, and `test_plan`, plus any declared fact requirements. Fact requirements are generic EventGraph event-type requirements; Work can require, for example, an authority or lifecycle event without owning that authority decision.

## 2. Current event types and task states

Current Work event types:

- `work.task.created`
- `work.task.assigned`
- `work.task.completed`
- `work.task.dependency.added`
- `work.task.priority.set`
- `work.task.comment`
- `work.task.unblocked`
- `work.task.artifact`
- `work.task.artifact.waived`
- `work.task.fact.required`
- `work.phase.gate.declared`
- `work.phase.gate.approved`
- `work.phase.gate.rejected`

Current task states are a derived `TaskStatus` enum:

- `pending`: created but no assignment event found.
- `assigned`: assignment event found and no completion event found.
- `completed`: completion event found.

Blocked state is not part of `TaskStatus`. It is derived separately from dependency edges whose dependency task has not completed, unless a `work.task.unblocked` event exists for the blocked task. Completion currently requires either a `work.task.artifact` or `work.task.artifact.waived` event, and `work.task.completed` stores an `ArtifactRef` to one satisfying artifact or waiver.

The current `SupersedeDuplicateDirectChildren` helper does not create a `superseded` state. It adds an audit comment, waives the duplicate's artifact requirement, and completes the duplicate task. That preserves append-only history but collapses supersession into `completed`.

The phase gate events are legacy Work-local gate events. They should not become the v3.9 gate model.

## 3. Gaps versus v3.9

### FactoryOrder linkage

v3.9 requires every product task to link back to a FactoryOrder or EvolutionOrder, and every downstream artifact to link back to a FactoryOrder. Current tasks have no FactoryOrder, EvolutionOrder, FactoryOrder version, source intent, release policy, or order transition awareness. Workspace is not a replacement for FactoryOrder linkage.

### Requirement linkage

v3.9 product tasks require requirement references. Current Work has task-to-task dependencies, but no links from Task to Requirement. The existing `TaskFactRequiredContent` can require an external EventGraph fact, but it is a readiness prerequisite, not a durable requirement edge.

### AcceptanceCriterion linkage

v3.9 acceptance evidence flows through `TestRun -> TestCase -> AcceptanceCriterion -> Requirement -> FactoryOrder`. Current Work stores acceptance criteria only as an artifact label/body on a task. That is useful as operator-facing readiness text, but it is not a first-class AcceptanceCriterion reference and cannot satisfy trace completeness.

### Task lifecycle states

v3.9 Task state is:

```text
created | ready | running | blocked | failed | repair_required | repair_running | repaired | verification_running | verified | certified | rejected | superseded | policy_blocked
```

Current Work only derives `pending | assigned | completed`, plus separate booleans for blocked and ready. There is no append-only task transition event, no transition validator, no terminal-state enforcement, and no explicit evidence requirement for high-risk transitions.

### repair states

Current Work has no `Failure` or `RepairAttempt` integration and no repair lifecycle. Duplicate-child supersession uses completion and waiver, not `superseded`. v3.9 requires Work to own repair scheduling and repair state projection while EventGraph owns durable `Failure` and `RepairAttempt` records.

### verification states

Current Work has no `verification_running`, `verified`, or `certified` task states. It has readiness artifacts and generic fact requirements, but no direct `TestRun` or `GateResult` references and no verified/certified transition evidence.

### policy_blocked

Current Work has no policy-blocked task state. v3.9 RuntimeResult can return `policy_blocked`; retry policy says it must not auto-retry and requires review. Work must be able to project this state from RuntimeBroker/EventGraph evidence and prevent automatic repair or rerun scheduling.

### rejected

Current Work has no rejected task state. Completion is the only terminal state. v3.9 allows rejection from `verification_running` and `verified`, and rejected results appear in Base Slice negative outputs.

### superseded

Current Work has a duplicate-child cleanup helper, but superseded is not modeled as a task lifecycle state. v3.9 requires `any non-terminal -> superseded`, with immutable nodes superseded rather than edited.

### GateResult/TestRun/Failure/RepairAttempt references

Current Work completion refers only to a Work artifact or artifact waiver. v3.9 requires references to EventGraph-owned evidence nodes:

- `TestRun` for verification execution.
- `GateResult` for gate pass/fail/error/skipped/waived outcomes.
- `Failure` for runtime, gate, traceability, policy, and repair failures.
- `RepairAttempt` for planned/running/succeeded/failed/abandoned repair activity.

The existing `work.task.fact.required` can be retained as a generic readiness prerequisite mechanism, but it cannot replace typed v3.9 evidence links.

## 4. Proposed v3.9 event model

After EventGraph Stage 1 lands, Work should align its events with EventGraph-owned Tier 0 nodes and edges instead of inventing a parallel gate or task truth model.

Recommended Work-owned events:

- `work.task.created.v39`: operational creation event that references the EventGraph `Task` node ID, FactoryOrder or EvolutionOrder ID, requirement IDs, production cell, dependencies, priority, risk class, expected outputs, and idempotency key. If EventGraph creation already emits a canonical task creation event, this Work event should become a projection helper or be omitted.
- `work.task.transition.requested`: optional command/audit event for Work API callers before canonical EventGraph mutation.
- `work.task.state.changed`: append-only lifecycle transition mirror with task ID, from state, to state, actor, reason, evidence refs, and correlation ID. This should either be the EventGraph canonical lifecycle event or a strict Work projection of it, not an independent source of truth.
- `work.task.linked.requirement`: links a task to one or more Requirement IDs when not represented directly by EventGraph edges.
- `work.task.linked.acceptance_criterion`: links a task to AcceptanceCriterion IDs when needed for scheduler projections.
- `work.task.blocked`: records dependency, policy, authority, missing evidence, or external failure blocking reason with evidence refs.
- `work.task.unblocked`: narrows the existing override into an evidence-backed transition from `blocked` to `ready`; it should not silently ignore unresolved dependencies.
- `work.task.repair.scheduled`: schedules repair for an EventGraph Failure and references the RepairAttempt ID.
- `work.task.verification.scheduled`: schedules verification and references expected TestCase or GateResult targets.
- `work.task.evidence.attached`: replaces or supplements generic task artifacts with typed refs to Artifact, CodeChange, TestRun, GateResult, Failure, RepairAttempt, Waiver, AuthorityDecision, or ExecutionReceipt.

States should be derived from lifecycle transitions and evidence, not updated in place. Allowed task transitions should match v3.9:

```text
created -> ready -> running -> verified -> certified
running -> failed -> repair_required -> repair_running -> repaired -> verification_running -> verified
verification_running -> rejected
verified -> rejected
running -> blocked
running -> policy_blocked
blocked -> ready
any non-terminal -> superseded
```

Projection rules:

- `ready` requires valid FactoryOrder/EvolutionOrder reference, valid requirement references for product tasks, dependencies satisfied or explicitly waived by authorized evidence, required readiness artifacts or v3.9 evidence records present, and no open blocking policy/failure state.
- `running` requires an ActorInvocation/RuntimeEnvelope path when runtime execution is involved.
- `failed` requires a Failure reference.
- `repair_required`, `repair_running`, and `repaired` derive from Failure and RepairAttempt state.
- `verification_running` requires scheduled or running TestRun/GateResult evidence.
- `verified` requires passing or waived GateResult/TestRun evidence for required AcceptanceCriteria.
- `certified` should only follow release/certification evidence, not Work-local completion.
- `policy_blocked` requires RuntimeResult/ActorInvocation/Failure evidence with policy-blocked status and must not auto-retry.
- `rejected` requires verification or certification rejection evidence.
- `superseded` requires a replacement task or supersession edge/reference.

## 5. Proposed migration plan after EventGraph Stage 1 lands

1. Freeze the Stage 1 EventGraph schemas and path query names for `FactoryOrder`, `Requirement`, `AcceptanceCriterion`, `Task`, `TestRun`, `GateResult`, `Failure`, `RepairAttempt`, `RuntimeEnvelope`, `RuntimeResult`, `Waiver`, `AuthorityDecision`, and `ExecutionReceipt`.
2. Add Work projections that can read the canonical EventGraph Task node and lifecycle records without changing existing APIs. Keep current `pending/assigned/completed` projection available as a compatibility view.
3. Introduce v3.9 task creation APIs behind new names or versioned payloads. Require FactoryOrder or EvolutionOrder reference, production cell, requirement references for product tasks, dependencies, risk class, priority, and expected outputs.
4. Add a transition validator that enforces the v3.9 task state machine, terminal states, evidence requirements, and audit behavior for invalid high-risk or critical transitions.
5. Replace completion-centric semantics with evidence-backed transitions. Existing `Complete` can map to a compatibility transition only when enough evidence exists to move to `verified` or `certified`; otherwise it should remain legacy.
6. Migrate artifact readiness labels into v3.9 evidence references. Keep operator-facing artifacts as notes or attachments, but make AcceptanceCriterion, TestCase, TestRun, GateResult, and Waiver the trace-completeness path.
7. Update dependency blocking so unresolved dependencies, missing authority, missing evidence, policy decisions, and open failures are represented as explicit blocked or policy-blocked reasons.
8. Implement repair scheduling only after Failure and RepairAttempt records exist. Enforce `max_repair_attempts_per_task = 3` in Work scheduling, with per-release-candidate limits read from the runtime/release layer.
9. Retire Work-local phase gate declarations from lifecycle decisions. Preserve event unmarshalling for old data, but do not route v3.9 gates through `work.phase.gate.*`.
10. Add a one-way migration/projection for old tasks: `pending -> created`, assigned open tasks with readiness satisfied -> `ready`, assigned open tasks without readiness -> `created` or `blocked` depending on blockers, completed tasks -> legacy-completed projection with no automatic `verified/certified` unless required v3.9 evidence exists.

## 6. Required tests

Required replay and derived-state tests:

- v3.9 task creation records FactoryOrder/EvolutionOrder, requirements, cell, risk, priority, expected outputs, dependencies, and idempotency key.
- product task creation without requirement references fails.
- task readiness derives `ready` only when order, requirements, dependencies, and required evidence are satisfied.
- lifecycle replay reconstructs every valid v3.9 task transition.
- invalid transitions fail without mutating derived state.
- terminal states cannot transition except where v3.9 explicitly allows supersession from non-terminal states only.
- high-risk or critical invalid transitions write audit evidence when Stage 1 audit records are available.
- dependency blocking derives `blocked`, and evidence-backed unblock returns to `ready`.
- RuntimeResult or Failure with `policy_blocked` derives `policy_blocked` and schedules no auto-retry.
- `failed -> repair_required -> repair_running -> repaired -> verification_running -> verified` replays from Failure and RepairAttempt evidence.
- repair attempts stop at `max_repair_attempts_per_task = 3`.
- verification requires TestRun/GateResult evidence tied to AcceptanceCriterion/Requirement/FactoryOrder.
- rejected verification derives `rejected`.
- supersession derives `superseded` and records replacement evidence.
- legacy task events continue to replay through the compatibility projection.

Required path tests that should sit in EventGraph or integration tests:

- `CodeChange -> Artifact -> ActorInvocation -> Task -> Requirement -> FactoryOrder`
- `TestRun -> TestCase -> AcceptanceCriterion -> Requirement -> FactoryOrder`
- `Failure -> RepairAttempt -> TestRun/GateResult`
- `ReleaseCandidate -> Certification -> GateResult`

## 7. Risks

- Implementing Work lifecycle before EventGraph Stage 1 stabilizes would create duplicate truth and migration churn.
- Treating current readiness artifacts as AcceptanceCriterion evidence would weaken trace completeness.
- Keeping `work.task.unblocked` as an unconditional override could hide unresolved dependencies or authority failures.
- Mapping old completed tasks directly to `verified` or `certified` would overstate evidence. Legacy completion should remain compatibility state unless v3.9 evidence exists.
- Reusing `work.phase.gate.*` for v3.9 gates would resurrect a parallel gate model.
- Policy-blocked handling crosses RuntimeBroker, EventGraph Failure evidence, and Work scheduling; unclear ownership could accidentally permit auto-retry.
- Repair limits need both per-task and per-release-candidate visibility. Work can enforce per-task locally, but release-candidate limits need EventGraph/release evidence.
- Broad scans over `ByType(..., 1000, ...)` are acceptable in current tests but will not scale for v3.9 projections without indexed queries or path queries.

## 8. Non-goals

- Do not resurrect `refactor/solo-phase-gates`.
- Do not implement `PhaseGateStore`.
- Do not invent a parallel Work gate model.
- Do not implement RuntimeBroker.
- Do not implement Stage 3 lifecycle code in this recon.
- Do not make Work the source of production truth; EventGraph owns durable v3.9 nodes, edges, required paths, and evidence records.
- Do not convert legacy `completed` tasks to `verified` or `certified` without required v3.9 evidence.
- Do not push, merge, deploy, create repositories, access secrets, or mutate upstream state as part of this design-only task.
