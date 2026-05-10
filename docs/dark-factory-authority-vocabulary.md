# Dark Factory Authority Vocabulary

Date: 2026-05-08

Source of truth: `transpara-ai/docs` `dark-factory/DF-SOP-0001-authority-gated-side-effects.md`.

Work records tasks, phase gates, and approvals. It must not invent alternate protected-action names when work items, phase gates, or operator notes refer to authority-gated side effects.

## Authority Outcomes

```text
Autonomous
Notify
ApprovalRequired
Forbidden
```

## Protected Actions

```text
production.deploy
repo.create
repo.delete
repo.push.default_branch
repo.merge.main
repo.mutate.cross_repo
agent.spawn.persistent
agent.retire
agent.revoke
agent.escalate_permissions
policy.change
secret.access
external_communication.company_voice
data.delete
self_modification.activate
billing.spend_above_threshold
license.change
```

## Local Alignment Notes

- Phase gate names may describe project workflow, but authority-gated side effects must use the canonical action names above.
- `repo.merge.main` is distinct from `repo.push.default_branch`; approval for one does not approve the other.
- `repo.mutate.cross_repo` is the canonical spelling for multi-repo mutation; do not use `repo.mutate_cross_repo`.
