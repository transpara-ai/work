# Work Incident Follow-Up Adoption

**Date:** 2026-06-16
**Status:** Accepted for source-repo adoption slice
**Scope:** Work task artifacts and incident execution handoff

## Source Contract

Work accepts the pre-live incident follow-up schema defined by:

```text
operation/docs/operations/work-incident-follow-up-schema.md
```

The schema is represented in Work as a typed JSON artifact on the task graph,
using the `incident_follow_up` artifact label and `application/json` media type.

## Boundary

`operation` remains the source of truth for the incident record,
cross-repository operating analysis, evidence sufficiency, and closure
criteria.

`work` owns execution state only when a follow-up is routed into Work. The Work
artifact records the incident follow-up task id, incident record link, task
type, owner fields, status, required evidence, authorization requirement,
validation requirement, dependencies, acceptance criteria, and closure link.
The source contract lists these fields as required for every follow-up task, so
the adapter rejects empty scalar fields such as `requested_by`, `assigned_to`,
and `severity` for every status, including `PROPOSED`.
The source contract defines closed values for `task_type` and `status`; it does
not define a closed severity vocabulary, so Work treats `severity` as required
free text for this slice.

The required-field excerpt from the source contract is:

```text
task_id:
incident_id:
incident_record:
task_type:
owning_repo:
requested_by:
assigned_to:
status:
severity:
summary:
required_evidence:
authorization_required:
authorization_evidence:
validation_required:
validation_evidence:
blocking_dependencies:
acceptance_criteria:
closure_link:
```

Opening or assigning a Work follow-up does not authorize a human-gated action.
Durable authorization evidence is still required before a routed follow-up can
be completed as `DONE` when `authorization_required` is true. Work validates
that evidence fields are present, but operation remains responsible for
determining whether the evidence is sufficient.

## Implementation

The first adoption slice intentionally uses Work's existing artifact event
model:

- `IncidentFollowUp` defines the accepted payload shape.
- `IncidentFollowUpArtifactBody` validates and encodes the payload.
- `TaskStore.AddIncidentFollowUpArtifact` records it as `work.task.artifact`.
- `TaskStore.ListIncidentFollowUps` replays and validates stored artifacts.
- `TaskStore.LatestIncidentFollowUp` resolves the most recent follow-up
  artifact for callers that need current contract state.

No new task lifecycle status is introduced in this slice. The follow-up status
values are contract fields inside the artifact so Work can accept the
operation schema without changing the current Work task lifecycle.

Stored artifacts stamped with the former `civilization-operation` schema
reference are still accepted and normalized on replay for compatibility.

Matching `incident_follow_up` artifacts are treated as contract records. If a
matching artifact has invalid JSON, the wrong schema, the wrong schema version,
unknown JSON fields, or a payload that no longer validates, reads fail loudly
instead of silently dropping the record. Non-matching task artifacts are ignored
by the incident follow-up projection.

The generic `TaskStore.AddArtifact` path also validates any artifact using the
reserved `incident_follow_up` label. Callers may use either
`AddIncidentFollowUpArtifact` or a pre-encoded JSON artifact, but the reserved
label cannot persist malformed, wrong-schema, wrong-version, unknown-field, or
wrong-media-type payloads.

## Test 001 Relationship

This source-repo adoption slice provides the Work-side acceptance required by
Test 001's follow-up schema gap. It does not claim that every source repository
has implemented its own incident workflow. Other source repos still need to
accept or implement the relevant incident follow-up contracts before Test 001
can move beyond a repo-specific partial pass.
