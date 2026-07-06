# Source-Issue Marker Work Refs

## Purpose

`docs#256` requires GitHub source-issue marker comments to cite canonical Work
state without becoming canonical workflow state themselves. Work owns the
FactoryOrder/task/stage lifecycle refs that Hive, EventGraph, and Site can
project.

## Contract

`TaskStore.ProjectIssueScanMarkerWorkRef(ref)` returns a
`work.issue_scan.source_marker_ref` packet derived from replayed Work events.
`TaskStore.ProjectIssueScanMarkerWorkRefJSON(ref)` emits the same packet as
stable JSON for fixtures or downstream artifacts.
The schema constants are exported as `IssueScanMarkerSchemaVersion` and
`IssueScanMarkerProjectionKind`. `verification_refs` and
`failure_repair_refs` are always present in JSON, as empty objects when no such
refs exist, so consumers do not infer meaning from key absence.

The packet includes:

- issue-scan `run_id`, target repo/issue as `target.repository` and
  `target.issue_number`, stage id, stage number, and gate;
- Work task id, canonical task id, FactoryOrder id, requirement ids, and
  acceptance criterion ids;
- lifecycle state, ready/blocked state, missing readiness gates/facts, latest
  blocker, latest gate, and supersession target;
- verification, failure, repair, waiver, and source-issue refs;
- explicit authority exclusions.

## Boundaries

Work remains the canonical execution, readiness, blocking, and lifecycle source.
GitHub comments and labels are projection outputs only. Consumers must not parse
GitHub marker comments or labels as Work truth.
`latest_blocker` is historical context: consumers must use `blocked` and
`lifecycle_state` to decide whether a blocker is currently active.
`ready` and `missing_gates` describe start-readiness only; consumers must not
treat them as completion, certification, or merge-validity signals.

This contract does not authorize live GitHub mutation, EventGraph production
writes, Hive action APIs, deploy, value allocation, autonomy increase, Test 001
GREEN, merge, or issue closure.

## Consumers

- Hive should pass this packet or its fields into the source-issue marker bridge.
- EventGraph should use it as the Work-side input for provenance/projection
  records.
- Site should render it read-only as operator evidence.
- Wiki is intentionally omitted from this slice.
