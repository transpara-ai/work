# Tasks Batch + Incremental Fold — Design Packet

- **doc_id:** WORK-TASKS-INCREMENTAL-FOLD-DESIGN-001
- **version:** v0.1.0
- **status:** IADA applied → CFADA
- **issue:** https://github.com/transpara-ai/work/issues/82
- **base:** work main @ 5804038
- **scope:** `store.go` list-path restructure + a new fold-cache layer + `cmd/work-server` wiring; no schema/DB changes, no writes, single-task endpoints untouched.

## 1. Problem

`GET /tasks` = 11.9-14.0s measured. `batchStatus` (store.go:1432) does 6 batch scans then calls `GetStatus` + `GetCompatibilityStatus` + `factReadiness` **per task** (store.go:1516-1528) — ~300 full event-type scans per request at limit=100 over ~22k events. Site's 5s client budget makes the console Kanban permanently honest-unavailable.

## 2. Decisions

### D1 — Batch the per-task trio; original methods stay as the oracle

One `latestLifecycleStatuses()` call replaces per-task `GetStatus` (identical fold + `StatusCreated` default). New batch folds replicate `ProjectLegacyTask`'s status derivation and `factReadiness` exactly — the implementer enumerates every event type those methods read and reproduces their newest-wins/ordering/reopen semantics bit-for-bit. `GetStatus`/`GetCompatibilityStatus`/`factReadiness` REMAIN (single-task endpoints use them) and serve as the **oracle in the golden equivalence test**: seeded full-domain store (lifecycle transitions incl. reopens, legacy completions, assignments, dependencies satisfied/unsatisfied, unblocked, artifacts with/without required gates, waivers, facts present/missing, issue-scan certified stage tasks) → batched `ListSummaries` deep-equals per-task-assembled summaries for EVERY task and EVERY field. Any divergence fails. (Fix-the-class: the equivalence is proven over the whole input domain, not the reported symptom.)

### D2 — Head-keyed memoized fold with frontier-based increments (IADA-1)

- Fold state: the batch maps + the raw per-task event aggregates needed for correct incremental application (completion/reopen event lists per task, not collapsed booleans — IADA-3) + a **frontier** of the newest seen event ID per folded event type.
- On request: read the store head. Head == held head → serve held state (zero scans). Else, for each folded event type, page newest-first ONLY until an event already in the frontier is reached — the prefix is exactly the new events (IADA-1: this is robust regardless of ByType cursor direction semantics, which must be verified but not relied on for forward iteration). Apply new events oldest→newest with last-write-wins (equals newest-wins — IADA-2). Update frontier + head.
- **Equivalence test:** interleave appends (including a reopen for a task completed BEFORE the previous fold, and an unblock for a pre-fold dependency) with requests; after each request, the served state must deep-equal a from-scratch fold at that head.
- **Fail-closed:** any error reading head or new pages → discard held state entirely, rebuild from scratch; scratch failure → request errors (today's behavior). The memo key is the exact head — a served response is always the provable fold at a real observed head; NO TTL, no staleness introduced. Race note: events landing between head-read and serve appear in the next request, same snapshot semantics as today (IADA-7).
- **Stampede control:** rebuilds wrapped in singleflight (x/sync) — concurrent requests share one fold; each writes its own response.

### D3 — Measurement

Timed test (~5,000-event seeded store, skip in -short, race-aware budget via build-tag flag as in hive PR #241): cold fold < 2s, warm (unchanged head) < 50ms. Post-fix LIVE `GET /tasks` measured and recorded in the PR. Growth headroom: cold fold is O(total events) with K single passes (K = folded event types, ~10); warm is O(1); incremental is O(new events).

## 3. Non-goals

No DB materialization, no schema/index changes (even though a `type` index would help — out of scope), no /tasks JSON shape changes, no site-side changes (its 5s timeout becomes sufficient), no changes to write paths or single-task reads.

## 4. TDD plan

1. Golden equivalence test (D1 oracle) — written FIRST against the existing code (passes trivially with per-task calls), then drives the batch restructure.
2. Batch restructure of `batchStatus`; stale doc comment on `ListSummaries` corrected.
3. Fold-cache layer + frontier increments + interleaved-equivalence test + fail-closed tests (head-read error → rebuild; page error → rebuild; both fail → error).
4. Singleflight collapse test (bounded, statistical assertion pattern from hive PR #241 — computations < N with bounded retry).
5. Timed test + live measurement + Kanban visual evidence.

## 5. IADA record (v0.1.0, 2026-07-02)

- **IADA-1 (cursor-direction hazard):** `ByType` returns newest-first; assuming forward "after cursor" iteration for increments would be fragile. Resolved: frontier-based increments (page newest-first until a seen event ID), correct under either cursor semantic.
- **IADA-2 (newest-wins under increments):** applying new events oldest→newest with last-write-wins preserves newest-wins map semantics; mandated explicitly.
- **IADA-3 (reopen pairing across the fold boundary):** live-completion semantics pair completions with reopens; collapsed booleans in the fold state would mis-apply a reopen arriving for a pre-fold completion. Resolved: the state retains per-task completion/reopen aggregates; the interleaved test includes exactly this case.
- **IADA-4 (oracle drift):** keeping the per-task methods as the golden-test oracle means the batch path can never silently diverge — any future change to per-task semantics fails the equivalence test until both paths agree.
- **IADA-7 (head race):** head-read → serve race yields at-least-as-fresh-as-request-start snapshots, identical to today's semantics; documented, not "fixed".
