# Work Max-Revolutions Governor

Source issue: `transpara-ai/work#67`

## Purpose

The max-revolutions governor bounds review/repair loops before they can burn
tokens or imply authority from repeated failed attempts. It is a pure policy
evaluator: callers provide current loop evidence and receive a deterministic
next-state recommendation.

## Policy Thresholds

Default thresholds:

- `max_repair_revolutions`: 3
- `split_after_revolutions`: 2
- `abandon_after_revolutions`: 3
- `max_no_progress_revolutions`: 2

Policy validation is fail-closed:

- unset zero thresholds default to the values above;
- negative thresholds are rejected;
- split and abandon thresholds must be within the maximum repair revolution
  budget;
- split must occur no later than abandon;
- no-progress split must be reachable before abandon;
- human escalation roles must be non-empty.

## State Transitions

`EvaluateReviewRepairGovernor` returns one deterministic action:

- `complete`: validation passed and zero blockers remain.
- `human_escalation_required`: protected action or human scope is required
  before the loop can continue.
- `abandon_required`: repair revolutions reached the abandon threshold or
  maximum repair cap.
- `split_required`: no-progress or split-candidate thresholds were reached.
- `revise`: blockers remain but thresholds have not been crossed.
- `continue`: the loop remains under configured thresholds.

The decision preserves `source_issue_refs` so downstream proof and AuditReport
evidence can cite the originating GitHub issues.

If `protected_action_required` is true and
`authority_decision_available` is also true, the governor does not escalate, but
the decision reasons record that the recommendation depends on caller-supplied
authority evidence. The governor is stateless: callers must increment
`repair_revolutions`, set `consecutive_no_progress`, and supply truthful
authority and scope flags each cycle.

## Authority Boundary

The governor does not mutate Work state, open branches or PRs, execute runtime
actions, write EventGraph records, deploy, increase autonomy, allocate value,
or close residual risks. It only produces a recommendation that a separately
authorized caller may record or act on.
