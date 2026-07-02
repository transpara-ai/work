# Tasks Batch + Incremental Fold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `GET /tasks` from 12-14s to p99 well under 5s (sub-second warm) via batch folds + a fold-generation memo, with event-sourced correctness proven by golden equivalence against the existing per-task methods.

**Architecture:** BINDING spec = `docs/designs/tasks-incremental-fold-v0.1.0.md` (internal v0.3.0, CFADA PASS — D1/D1a/D2/D3 are law; read it FIRST). Anchors: `store.go` batchStatus:1432 (per-task loop 1516-1528), GetStatus:796, factReadiness:1916, latestLifecycleStatuses:1249, ListSummaries:1548; handler `cmd/work-server/main.go:1057`.

**Tech Stack:** Go; eventgraph store via replace (../eventgraph/go); x/sync singleflight; GOFLAGS=-buildvcs=false in this worktree if VCS stamping errors appear.

## Global Constraints

- Single-task methods (`GetStatus`, `GetCompatibilityStatus`, `factReadiness`, `Readiness`, `ProjectLegacyTask`) unchanged — they are the golden oracle.
- The ONE intentional output change is D1a (list gates/ready adopt the non-empty-body contract); everything else byte-identical.
- Fail-closed per packet D2's enumerated rules; no TTL anywhere; provisional folds never memoized.
- Tests per task: `go test ./... -count=1` green; `go vet ./...` clean.
- Commits conventional, ending `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

### Task 1: Golden equivalence test + batch restructure (D1/D1a)

**Files:** `store.go` (batchStatus restructure + new batch legacy-status fold + fact-requiring pre-scan + stale ListSummaries doc comment fix), new `store_list_equivalence_test.go`.

Steps: (1) Write the golden test FIRST per packet D1's domain list (incl. EMPTY-BODY required gates and issue-scan certified stage tasks; oracle = per-task methods with `Readiness` for gates/ready) — it must FAIL against current code ONLY on the D1a rows (empty-body gate divergence) proving both the bug and the oracle's discriminating power; every other row must already pass (this validates the oracle wiring). (2) Restructure: hoist one `latestLifecycleStatuses()`; add a batch legacy-status fold reproducing `ProjectLegacyTask`'s status semantics exactly (enumerate its event reads first); replace the per-task gate map with the non-empty-body rule; fact pass narrowed to fact-requiring tasks via one `work.task.fact.required` scan (conformance-asserted against per-task `factReadiness` in the golden test — CFADA2-adv2). (3) Whole suite green; fix the stale "three batch store scans" comment. (4) Commit `perf(store): batch the per-task status/legacy/facts folds in ListSummaries`.

### Task 2: Fold-generation memo layer (D2)

**Files:** new `store_fold_cache.go` + `store_fold_cache_test.go`; `cmd/work-server/main.go` wiring (listTasks path only).

Implement per packet D2 verbatim: fold state w/ per-task aggregates + per-type frontiers; stable/provisional generations (memoize ONLY stable); frontier-based increments (page newest-first until frontier hit; apply oldest→newest); singleflight keyed by observed head; enumerated fail-closed rules (any read/decode error → discard + scratch rebuild → error passthrough); facts always computed per request outside the memo. Tests: interleaved append/request equivalence vs from-scratch fold (incl. reopen-for-pre-fold-completion and unblock-for-pre-fold-dependency); no-promotion test — an older-head flight finishing AFTER a newer stable generation must not overwrite it (CFADA2-adv1); fail-closed table (head error / page error / frontier miss / both-fail-error); pagination conformance test on InMemory (+ pgstore if a test DSN is available, else document); empty-store and zero-head cases. Commit `perf(store): head-keyed fold-generation memo with frontier increments for /tasks`.

### Task 3: Measurement + full verify (D3)

**Files:** `store_fold_bench_test.go` (or extend Task 2's test file).

Timed test per D3: ~25,000-event seeded store (mixed domain incl. non-work events as noise) — cold fold < 2s, warm < 100ms; ~50k seeding asserts cold scaling < 2.2x the 25k time; skip in -short; race-aware budget via `raceEnabled` build-tag pair (files race_enabled_test.go / race_disabled_test.go, `//go:build race` / `//go:build !race`). Full `go build ./... && go vet ./... && go test ./... -count=1`. Commit `test(store): fold latency budget at live scale`.
