# Tasks Batch + Incremental Fold — Design Packet

- **doc_id:** WORK-TASKS-INCREMENTAL-FOLD-DESIGN-001
- **version:** v0.3.0 (CFADA PASS)
- **status:** CFADA PASS — building under TDD
- **issue:** https://github.com/transpara-ai/work/issues/82
- **base:** work main @ 5804038
- **scope:** `store.go` list-path restructure + a new fold-cache layer + `cmd/work-server` wiring; no schema/DB changes, no writes, single-task endpoints untouched.

## 1. Problem

`GET /tasks` = 11.9-14.0s measured. `batchStatus` (store.go:1432) does 6 batch scans then calls `GetStatus` + `GetCompatibilityStatus` + `factReadiness` **per task** (store.go:1516-1528) — ~300 full event-type scans per request at limit=100 over ~22k events. Site's 5s client budget makes the console Kanban permanently honest-unavailable.

## 2. Decisions

### D1 — Batch the per-task trio; original methods stay as the oracle

One `latestLifecycleStatuses()` call replaces per-task `GetStatus` (identical fold + `StatusCreated` default). New batch folds replicate `ProjectLegacyTask`'s status derivation exactly — the implementer enumerates every event type it reads and reproduces its newest-wins/ordering/reopen semantics bit-for-bit. Fact readiness keeps the ORIGINAL `factReadiness` code path, narrowed to fact-requiring tasks (D2). `GetStatus`/`GetCompatibilityStatus`/`factReadiness` REMAIN (single-task endpoints use them) and serve as the **oracle in the golden equivalence test**:

**D1a — Gate-presence divergence resolved toward the documented contract (CFADA1-3):** today `batchStatus` counts a required gate as present on label alone (`store.go:1483-1499`) while the single-task `Readiness` contract requires a NON-EMPTY body (`store.go:1880-1907`: "a label-only (empty) artifact does not satisfy readiness"). That is a pre-existing fail-open divergence in the LIST path (a whitespace-body gate yields `ready=true`). This packet DELIBERATELY aligns the list path to the documented non-empty-body contract — an intentional, declared behavior correction (fail-open → fail-closed), called out in the PR body with its own test; the oracle for `Ready`/`MissingGates` is the single-task `Readiness` method (gates half), NOT the old batch fold. This is the one intentional `/tasks` output change; everything else is byte-identical.

**Golden equivalence test:** seeded full-domain store (lifecycle transitions incl. reopens, legacy completions, assignments, dependencies satisfied/unsatisfied, unblocked, artifacts with required gates including EMPTY-BODY gates, waivers, facts present/missing, issue-scan certified stage tasks) → batched `ListSummaries` deep-equals per-task-assembled summaries (oracle: `GetStatus` + `GetCompatibilityStatus` + `factReadiness` + `Readiness` for the gates/ready half) for EVERY task and EVERY field. Any divergence fails. (Fix-the-class: the equivalence is proven over the whole input domain, not the reported symptom.)

### D2 — Fold-generation memoization with frontier-based increments (IADA-1, CFADA1-1/2/4/5)

**Corrected memo-key claim (CFADA1-1):** `Head()` and `ByType()` are separate, unsnapshotted reads — a fold can never be pinned to an exact head. The honest invariant: a fold observes `headBefore`, scans, then observes `headAfter`. If `headBefore == headAfter` the fold is **stable** and memoized under that head; if they differ the fold is **provisional** — it is still served (it is at-least-as-fresh as `headBefore`; cross-type skew is bounded by fold duration, identical to today's per-request semantics) but NOT memo-hit: the next request's head check misses and tops up via frontiers, so skew self-heals within one request cycle. No response is ever staler than the head observed at its fold start.

**Fold scope (CFADA1-4, adv1) — the COMPLETE `/tasks` domain:** created (ordered, for `List(limit)`), linked (newest-wins overlay), assigned, completed + reopened (paired aggregates), dependencies, unblocked, artifacts (count + required-gate labels **with the non-empty-body rule** — D1a), waivers, lifecycle transitions, issue-scan certification membership. Verification/failure-repair/comment events are declared out of scope (they do not affect `/tasks` today). Warm requests are O(new events) across ALL of these — including new `work.task.created` entering the top-N and `work.task.linked` overlay updates.

**Fact readiness is excluded from the memo (CFADA1-2):** `factReadiness` resolves satisfaction via `Get(requiredEventID)` + `Descendants(taskID)` — causal-graph queries that per-type frontiers cannot see (an external `authority.decision.recorded` caused by the task satisfies a fact without any `work.task.*` event changing). Resolution: one batch scan of `work.task.fact.required` identifies the (rare) tasks carrying fact requirements; ONLY those tasks call the existing `factReadiness` per request (unchanged code path — perfect fidelity); tasks without requirements short-circuit to no-missing-facts. Fact results are never cached. If fact-requiring tasks ever become numerous, that is a future packet (documented limit).

- Fold state: the batch maps + the raw per-task event aggregates needed for correct incremental application (completion/reopen event lists per task, not collapsed booleans — IADA-3) + a **frontier** of the newest seen event ID per folded event type.
- On request: read the store head. Head == held (stable) head → serve held state (zero scans, minus the per-request fact pass above). Else, for each folded event type, page newest-first ONLY until an event already in the frontier is reached — the prefix is exactly the new events. Apply new events oldest→newest with last-write-wins (equals newest-wins — IADA-2). Update frontier; set stable/provisional per the corrected claim above.
- **Supported-store boundary (CFADA1-adv2):** frontier paging requires working newest-first cursor pagination. Supported backends: pgstore (deployed) and InMemory (tests) — a conformance test pins the pagination contract on both. SQLite's `ByType` ignores cursors and is explicitly out of scope for the cache layer (the batch restructure alone still works there).
- **Equivalence test:** interleave appends (including a reopen for a task completed BEFORE the previous fold, and an unblock for a pre-fold dependency) with requests; after each request, the served state must deep-equal a from-scratch fold at that head.
- **Fail-closed (CFADA1-adv4, enumerated):** head-read error, page-read error, frontier event not found within the paged window, or any decode error → discard held state entirely, rebuild from scratch; scratch failure → request errors (today's behavior). Empty store/zero head → fold runs, memoizes under the zero head. Process restart → cold fold (state is process-memory only). Head changed during rebuild → provisional fold (served, not memoized) per the corrected claim. NO TTL anywhere. Memory bound: retained aggregates are O(created tasks + link/assign/completion/reopen/dependency/unblock/artifact/waiver refs) — proportional to work.task.* event count, documented in the fold-state struct comment.
- **Stampede control (CFADA1-5):** rebuilds wrapped in singleflight KEYED BY THE OBSERVED HEAD — requests that observed different heads never share a flight, so no caller can receive a fold older than the head it observed. Each caller writes its own response.

### D3 — Measurement (CFADA1-adv3)

Timed test on a **~25,000-event seeded store** (matching live scale; skip in -short; race-aware budget via build-tag flag as in hive PR #241): cold fold < 2s, warm (unchanged head) < 100ms including the narrowed fact pass. A second seeding at ~50k events asserts cold-fold scaling stays near-linear (< 2.2x the 25k time). Post-fix LIVE `GET /tasks` measured and recorded in the PR. Growth: cold O(total events, K single passes); warm O(new events + fact-requiring tasks).

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

## 6. CFADA record

### Round 1 (codex, 2026-07-02) — VERDICT: BLOCKERS (5) → all resolved in v0.2.0

- **CFADA1-1 (exact-head memoization unimplementable):** Head/ByType are unsnapshotted; resolved with stable/provisional fold generations — memoize only when headBefore==headAfter; provisional folds serve (at-least-as-fresh) but never memo-hit, self-healing skew next request.
- **CFADA1-2 (factReadiness is causal, not per-type):** Descendants-based satisfaction can change without any work.task.* event; resolved by EXCLUDING facts from the memo — one batch scan finds fact-requiring tasks, only those call the original factReadiness per request.
- **CFADA1-3 (gate-body divergence + oracle hole):** the list path's label-only gate counting is a pre-existing fail-open vs the documented non-empty-body Readiness contract; resolved by deliberately aligning list to the contract (declared behavior change, own test, Readiness as oracle for gates/ready).
- **CFADA1-4 (created/linked outside the fold):** the fold scope now enumerates the complete /tasks domain incl. created ordering and linked overlays with frontiers.
- **CFADA1-5 (singleflight head mixing):** flights keyed by observed head; no caller can receive a fold older than the head it observed.
- Advisories adopted: complete domain enumeration incl. declared out-of-scope event types (adv1); supported-store boundary pgstore+InMemory with a pagination conformance test, SQLite excluded (adv2); measurement rescaled to live scale ~25k + 50k near-linear scaling assertion (adv3); fail-closed cases enumerated incl. restart/empty-head/frontier-miss + memory bound documented (adv4).

### Round 2 (codex, 2026-07-02) — VERDICT: PASS (0 blockers)

Advisories adopted into the build plan: (1) concurrent different-head flight test — an older finishing flight must never promote over a newer stable generation; (2) fact-requirement pre-scan conformance-tested against factReadiness (codex verified requirements are only created via work.task.fact.required today); (3) PR body must call out D1a's visible Site impact (ready/missing_gates may shift fail-closed; codex found no automation consumer of the old fail-open list behavior). Codex confirmed the frontier algorithm is implementable on pgstore + InMemory cursor semantics.
