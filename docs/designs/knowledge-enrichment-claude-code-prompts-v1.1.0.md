# Knowledge Enrichment Infrastructure — Claude Code Task Prompts

**Version:** 1.1.0
**Last Updated:** 2026-04-06
**Status:** Active
**Versioning:** Independent of all other documents. Major version increments reflect fundamental restructuring of the implementation approach; minor versions reflect adjustments from reconnaissance or implementation feedback; patch versions reflect corrections and clarifications.
**Author:** Claude Opus 4.6
**Owner:** Michael Saucier

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-04-06 | Initial prompt sequence: 10 prompts (recon + 9 implementation) |
| 1.1.0 | 2026-04-06 | Post-recon: System actor registration added (Prompt 1 expanded). Boot replay uses 3x ByType() calls (no prefix query needed). Bus subscription confirmed as glob pattern. Exact insertion point at loop.go:398. Stabilization uses l.iteration check (no constant exists). Event types go in eventgraph following hive_event_types.go pattern. Knowledge store is first boot-replay component in hive — new territory. |

---

## Usage

Feed these to Claude Code **ONE AT A TIME**, in order. Wait for each to
complete and verify before moving to the next. Do not skip ahead. Do not
combine.

## Prerequisites

- Design spec `docs/designs/knowledge-enrichment-infrastructure-v1.0.0.md`
  is committed to `lovyou-ai-hive`
- You are in the `lovyou-ai-hive` repo root
- You have access to `lovyou-ai-eventgraph` and `lovyou-ai-agent` as
  sibling directories (or via Go module replace directives)

## Cross-Repository Coordination

This implementation touches three repositories in sequence:

```
Prompt 1:    lovyou-ai-eventgraph  (event types + content structs)
Prompt 2:    lovyou-ai-agent       (emit methods)
Prompts 3-9: lovyou-ai-hive        (everything else)
```

After Prompts 1 and 2, run `go mod tidy` in lovyou-ai-hive to pick up
the new event types and emit methods.

## Key Recon Findings (from Prompt 0)

These findings constrain all subsequent prompts:

1. **Store query:** `store.ByType(eventType, limit, after)` is the only
   type-based query. No prefix/wildcard. Boot replay needs 3 separate
   ByType() calls — one per knowledge event type.

2. **Bus:** `bus.Subscribe("knowledge.*", handler)` works — glob pattern
   matching supports prefix wildcards. Non-agent consumers subscribe
   identically to the telemetry writer.

3. **System actor:** `ActorTypeSystem` exists in the enum but NO System
   actor is registered in production. The runtime signs infrastructure
   events (hive.run.started, etc.) using the human operator's actor ID.
   The distiller needs a real System actor registered at boot.

4. **Loop.Config:** 16 fields, passed by value. Adding `KnowledgeStore`
   follows the BudgetRegistry pattern: add pointer field, nil-check at
   point of use.

5. **Goroutine lifecycle:** Fire-and-forget with context. Telemetry writer
   and watchForApprovedRoles both use `go start(ctx)` with no WaitGroup.
   Match this pattern.

6. **Boot replay is NEW TERRITORY.** BudgetRegistry, cooldowns, and
   spawnerState all start fresh every boot. The knowledge store is the
   first hive component to reconstruct state from the chain. This pattern
   should work cleanly but has no precedent to copy from.

7. **Event types go in eventgraph**, following the `hive_event_types.go` +
   `hive_content.go` + `AllHiveEventTypes()` collector pattern.

8. **Enrichment insertion:** After `enrichSpawnObservation` at loop.go:398,
   before `return enriched, nil` at line 399. Guard:
   `if l.config.KnowledgeStore == nil { return obs }`.

9. **enrichReviewObservation does not exist yet.** Reviewer implementation
   is in parallel. The knowledge enrichment call goes after whatever the
   last enrichment is at the time of implementation.

10. **`knowledge.*` namespace is clean.** No registered event types. Layer 5
    cognitive primitives already subscribe to `knowledge.*` — our events
    will activate them automatically.

---

## Prompt 0 — Reconnaissance (COMPLETE)

Key findings summarized above. All 10 recon questions answered.

---

## Prompt 1 — Event Types + System Actor (lovyou-ai-eventgraph)

```
Switch to the lovyou-ai-eventgraph repository.

Read the knowledge enrichment infrastructure design spec at
[path to spec in hive docs/designs/].

TASK 1: Create knowledge event types following the EXACT pattern used by
hive event types (hive_event_types.go + hive_content.go).

1a. Create go/pkg/event/knowledge_event_types.go:

   Three event type constants:

   var (
       EventTypeKnowledgeInsightRecorded   = NewEventType("knowledge.insight.recorded")
       EventTypeKnowledgeInsightSuperseded = NewEventType("knowledge.insight.superseded")
       EventTypeKnowledgeInsightExpired    = NewEventType("knowledge.insight.expired")
   )

   func AllKnowledgeEventTypes() []EventType {
       return []EventType{
           EventTypeKnowledgeInsightRecorded,
           EventTypeKnowledgeInsightSuperseded,
           EventTypeKnowledgeInsightExpired,
       }
   }

   Follow the exact pattern in hive_event_types.go — use the same
   NewEventType constructor, the same var block style, the same
   All*EventTypes() collector function.

1b. Create go/pkg/event/knowledge_content.go:

   Three content structs:

   // KnowledgeInsightContent represents a distilled insight from
   // pattern analysis. Emitted by automated distillers or knowledge agents.
   type KnowledgeInsightContent struct {
       InsightID     string   `json:"insight_id"`
       Domain        string   `json:"domain"`
       Summary       string   `json:"summary"`
       RelevantRoles []string `json:"relevant_roles"`
       Confidence    float64  `json:"confidence"`
       EvidenceCount int      `json:"evidence_count"`
       Source        string   `json:"source"`
       TTL           int      `json:"ttl"`
       SupersedesID  string   `json:"supersedes_id,omitempty"`
   }

   func (c KnowledgeInsightContent) EventTypeName() string {
       return EventTypeKnowledgeInsightRecorded.Value()
   }

   // KnowledgeSupersessionContent records when an insight is replaced
   // by a newer version.
   type KnowledgeSupersessionContent struct {
       OldInsightID string `json:"old_insight_id"`
       NewInsightID string `json:"new_insight_id"`
       Reason       string `json:"reason"`
   }

   func (c KnowledgeSupersessionContent) EventTypeName() string {
       return EventTypeKnowledgeInsightSuperseded.Value()
   }

   // KnowledgeExpirationContent records when an insight's TTL expires.
   type KnowledgeExpirationContent struct {
       InsightID string `json:"insight_id"`
       Reason    string `json:"reason"`
   }

   func (c KnowledgeExpirationContent) EventTypeName() string {
       return EventTypeKnowledgeInsightExpired.Value()
   }

   Follow the exact struct and EventTypeName() pattern from
   hive_content.go (GapDetectedContent, DirectiveContent, etc.).

1c. Register in DefaultRegistry():

   Find where AllHiveEventTypes() is called in the DefaultRegistry()
   function (likely in content.go or a registration file). Add
   AllKnowledgeEventTypes() in the same location, following the same
   registration pattern.

1d. Add unmarshalers in content_unmarshal.go (if the existing event types
   have unmarshalers registered there). Follow the exact pattern for the
   three new content types.

TASK 2: Verify no collisions.

   grep -r "knowledge\." across the event type files. Confirm no existing
   types use the knowledge.* prefix. (Recon confirmed this, but verify.)

Run all tests in the eventgraph repo. Nothing should break — these are
purely additive.

Commit with: "feat: add knowledge insight event types for civilization learning

- knowledge.insight.recorded: distilled insight from pattern analysis
- knowledge.insight.superseded: insight replaced by newer version
- knowledge.insight.expired: insight TTL elapsed
- Content structs with EventTypeName() methods
- Registered in DefaultRegistry via AllKnowledgeEventTypes()

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 2 — Emit Methods (lovyou-ai-agent)

```
Switch to the lovyou-ai-agent repository.

Read budget.go for the EmitBudgetAdjusted pattern, and spawn.go for the
EmitRoleProposed pattern. These are the templates.

Create knowledge.go (new file) with three emit methods:

1. EmitKnowledgeInsight(content event.KnowledgeInsightContent) error
   - checkCanEmit() guard
   - recordAndTrack(event.EventTypeKnowledgeInsightRecorded.Value(), content)
   - Error prefix: "knowledge insight"

2. EmitKnowledgeSupersession(content event.KnowledgeSupersessionContent) error
   - checkCanEmit() guard
   - recordAndTrack(event.EventTypeKnowledgeInsightSuperseded.Value(), content)
   - Error prefix: "knowledge supersession"

3. EmitKnowledgeExpiration(content event.KnowledgeExpirationContent) error
   - checkCanEmit() guard
   - recordAndTrack(event.EventTypeKnowledgeInsightExpired.Value(), content)
   - Error prefix: "knowledge expiration"

Each follows the identical pattern:
   func (a *Agent) EmitXxx(content event.XxxContent) error {
       if err := a.checkCanEmit(); err != nil {
           return fmt.Errorf("xxx: %w", err)
       }
       _, err := a.recordAndTrack(event.EventTypeXxx.Value(), content)
       if err != nil {
           return fmt.Errorf("xxx: %w", err)
       }
       return nil
   }

Run go mod tidy to pick up the new event types from eventgraph.
Run go vet and any existing tests.

Commit with: "feat: add knowledge insight emit methods

- EmitKnowledgeInsight for recording distilled insights
- EmitKnowledgeSupersession for replacing outdated insights
- EmitKnowledgeExpiration for TTL-based pruning
- Follows EmitBudgetAdjusted/EmitRoleProposed pattern

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 3 — Knowledge Store: Types and Interface (lovyou-ai-hive)

```
Switch to the lovyou-ai-hive repository. Run go mod tidy to pick up the
new event types and emit methods from Prompts 1 and 2.

Create a new package: pkg/knowledge/

1. Create pkg/knowledge/types.go:

   a. KnowledgeInsight struct (in-memory representation):
      InsightID     string
      Domain        string
      Summary       string
      RelevantRoles []string
      Confidence    float64
      EvidenceCount int
      Source        string
      RecordedAt    time.Time
      ExpiresAt     time.Time   // zero value = never expires
      Active        bool

   b. KnowledgeFilter struct (query parameters):
      Role          string        // filter by role relevance
      Domains       []string      // filter by domain (empty = all)
      MinConfidence float64       // default 0.3
      MaxAge        time.Duration // zero = no age filter
      ExcludeIDs    []string      // already-seen insight IDs

   c. Constants:
      const (
          DomainHealth       = "health"
          DomainBudget       = "budget"
          DomainQuality      = "quality"
          DomainArchitecture = "architecture"
          DomainProcess      = "process"
          DomainPerformance  = "performance"
          DomainPatterns     = "patterns"

          SourceDistillerPrefix = "distiller:"
          SourceMemoryKeeper    = "memorykeeper"
          SourceOperator        = "operator"

          MaxActiveInsights  = 100
          MaxPerDomain       = 20
          MaxPerSource       = 50
          MaxEnrichmentItems = 5
          MaxItemChars       = 300
          MaxBlockChars      = 1800
      )

2. Create pkg/knowledge/store.go:

   a. KnowledgeStore interface:
      Record(insight KnowledgeInsight) error
      Supersede(oldID, newID string) error
      Expire(insightID string) error
      Query(filter KnowledgeFilter, maxResults int) []KnowledgeInsight
      ActiveCount() int
      PruneExpired() int

   b. In-memory implementation:
      type store struct {
          mu       sync.RWMutex
          insights map[string]*KnowledgeInsight
      }

      func NewStore() KnowledgeStore

   c. Record():
      - mu.Lock() / defer mu.Unlock()
      - Add to map with Active = true
      - If insight has SupersedesID, mark old insight Active = false
      - Enforce MaxActiveInsights: if over limit, find and deactivate
        the lowest-confidence active insights until under limit

   d. Supersede():
      - mu.Lock() / defer mu.Unlock()
      - Find old by ID, set Active = false
      - Don't error if not found (may have already expired)

   e. Expire():
      - mu.Lock() / defer mu.Unlock()
      - Find by ID, set Active = false

   f. Query():
      - mu.RLock() / defer mu.RUnlock()
      - Filter active insights by all KnowledgeFilter criteria:
        * Role: include if RelevantRoles is empty (universal) OR
          contains filter.Role. Exclude if RelevantRoles is non-empty
          and does NOT contain filter.Role.
        * Domains: include if filter.Domains is empty OR insight.Domain
          is in filter.Domains
        * MinConfidence: include if insight.Confidence >= MinConfidence
        * MaxAge: include if MaxAge == 0 OR
          time.Since(RecordedAt) <= MaxAge
        * ExcludeIDs: exclude if InsightID is in ExcludeIDs
      - Score each passing insight with relevanceScore()
      - Sort descending by score
      - Return top maxResults

   g. relevanceScore(insight KnowledgeInsight, filter KnowledgeFilter) float64:
      - Role match: 0.4 if explicitly in RelevantRoles, 0.2 if universal
        (empty RelevantRoles), 0.0 if not relevant (should have been
        filtered, but safety net)
      - Confidence: insight.Confidence * 0.3
      - Recency: 0.2 if < 1h, 0.15 if < 6h, 0.1 if < 24h, 0.05 else
      - Evidence: 0.1 if EvidenceCount >= 10, 0.07 if >= 5, 0.03 else
      - Return sum

   h. PruneExpired():
      - mu.Lock() / defer mu.Unlock()
      - Iterate all insights
      - If Active && !ExpiresAt.IsZero() && time.Now().After(ExpiresAt):
        set Active = false, increment count
      - Return count pruned

   i. ActiveCount():
      - mu.RLock() / defer mu.RUnlock()
      - Count insights where Active == true

   j. RunPruner(ctx context.Context, interval time.Duration):
      - Background goroutine with time.NewTicker(interval)
      - Each tick: call PruneExpired()
      - Exit on ctx.Done()
      - Follow the telemetry writer goroutine pattern: fire-and-forget
        with context-based exit

Run go vet. No tests yet — Prompt 4 handles those.

Commit with: "feat: add knowledge store with relevance-scored querying

- In-memory store: Record/Supersede/Expire/Query/PruneExpired
- Relevance scoring: role match + confidence + recency + evidence
- Cardinality limits: 100 active, 20 per domain, 50 per source
- TTL-based expiration with background pruner goroutine
- Thread-safe with RWMutex

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 4 — Knowledge Store Tests

```
Read pkg/knowledge/store.go and types.go. Create comprehensive tests.

Create pkg/knowledge/store_test.go:

RECORD AND QUERY:

1. TestRecord_Basic — record 1 insight, verify ActiveCount, query by
   matching role returns it, query by non-matching role returns empty

2. TestRecord_UniversalInsight — empty RelevantRoles, query with any
   role returns it

3. TestQuery_MinConfidence — 3 insights (0.2, 0.5, 0.8 confidence),
   MinConfidence 0.3 returns only 0.5 and 0.8

4. TestQuery_MaxAge — 2 insights (now and 48h ago), MaxAge 24h returns
   only the recent one

5. TestQuery_DomainFilter — 3 domains, filter for 2 returns 2

6. TestQuery_ExcludeIDs — 3 insights, exclude 1, get 2

7. TestQuery_MaxResults — 10 insights, maxResults=3 returns 3

8. TestQuery_RelevanceOrder — verify descending order by relevance score

RELEVANCE SCORING:

9. TestRelevanceScore_RoleMatch — explicit=0.4, universal=0.2

10. TestRelevanceScore_Confidence — 1.0→0.3, 0.5→0.15

11. TestRelevanceScore_Recency — 30min→0.2, 3h→0.15, 12h→0.1, 2d→0.05

12. TestRelevanceScore_Evidence — 15→0.1, 7→0.07, 2→0.03

LIFECYCLE:

13. TestSupersede — A superseded by B, query returns only B

14. TestExpire — set ExpiresAt in past, PruneExpired() deactivates

15. TestPruneExpired_MixedTTL — permanent + expired + not-yet-expired,
    prune returns 1, ActiveCount = 2

16. TestRecord_WithSupersession — new insight with SupersedesID auto-
    deactivates old one

CARDINALITY:

17. TestCardinality_MaxActive — exceed MaxActiveInsights, lowest
    confidence pruned

THREAD SAFETY:

18. TestConcurrentAccess — 10 goroutines (5 recording, 5 querying),
    no panics with -race flag

Run all tests with -race flag.

Commit with: "test: add comprehensive knowledge store tests

- 18 test cases: record, query, filter, relevance scoring
- Lifecycle: supersede, expire, prune
- Cardinality limits, thread safety with -race

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 5 — System Actor + Chain Replay + Runtime Wiring (lovyou-ai-hive)

```
This prompt handles three related tasks: creating the system actor,
implementing chain replay, and wiring the knowledge store into the
runtime.

RECON CONTEXT:
- ActorTypeSystem exists in the eventgraph enum but is never registered
  in production. Infrastructure events currently use the human operator's
  actor ID — a workaround.
- BudgetRegistry does NOT replay events at boot — it starts fresh.
- The knowledge store will be the FIRST hive component to reconstruct
  state from the event chain. This is new territory.
- store.ByType(eventType, limit, after) is the only type query. No prefix
  query. We need 3 separate calls.

TASK 1: System Actor Registration

In the runtime bootstrap (wherever the human actor and AI agent actors
are registered — look at runtime.go or the boot sequence):

a. Register a System actor alongside the existing human and AI actors:

   systemActor := register a new actor with:
   - Type: ActorTypeSystem (from eventgraph constants)
   - Name: "system" (or "hive-system")
   - Deterministic key derivation: Ed25519(SHA256("system:hive"))
     following the same pattern agents use with "agent:" prefix

   Store the system actor reference on the Runtime struct so it can
   be passed to the distiller later.

b. Actor registration must be idempotent (the Postgres actor store
   checks for existing by public key). On reboot, the same system actor
   should be found, not duplicated.

c. Look at how agent actors are registered (likely in agent.New() or
   the boot ceremony). The system actor follows the same registration
   mechanism but with ActorTypeSystem instead of ActorTypeAI.

TASK 2: Chain Replay

Create pkg/knowledge/replay.go:

a. func ReplayFromStore(s store.Store, ks KnowledgeStore) error

   The store interface provides ByType(eventType, limit, after). Call it
   3 times — once per knowledge event type:

   types := []event.EventType{
       event.EventTypeKnowledgeInsightRecorded,
       event.EventTypeKnowledgeInsightSuperseded,
       event.EventTypeKnowledgeInsightExpired,
   }

   For each type, page through ALL events (use a large limit like 1000,
   loop if more exist using the cursor/after parameter).

   Collect all events, sort by timestamp (they come reverse-chrono from
   ByType), then process in chronological order:

   - For knowledge.insight.recorded: unmarshal content to
     event.KnowledgeInsightContent, convert to KnowledgeInsight, call
     ks.Record()
   - For knowledge.insight.superseded: unmarshal to
     event.KnowledgeSupersessionContent, call ks.Supersede()
   - For knowledge.insight.expired: unmarshal to
     event.KnowledgeExpirationContent, call ks.Expire()

b. func ConvertFromEventContent(
       content event.KnowledgeInsightContent,
       recordedAt time.Time,
   ) KnowledgeInsight

   - Maps event content to in-memory insight
   - Sets RecordedAt from the event timestamp parameter
   - Computes ExpiresAt: if TTL > 0, ExpiresAt = RecordedAt + time.Duration(TTL) * time.Hour
   - Sets Active = true

TASK 3: Runtime Wiring

In Runtime.Run() (runtime.go), after the store is open and before agents
boot:

a. Create the knowledge store:
   knowledgeStore := knowledge.NewStore()

b. Replay from chain:
   if err := knowledge.ReplayFromStore(r.graph.Store(), knowledgeStore); err != nil {
       // Log warning but don't fail startup — empty store is acceptable
   }

c. Start the pruner goroutine (fire-and-forget with context, matching
   telemetry writer pattern):
   go knowledgeStore.RunPruner(ctx, 15*time.Minute)

d. Subscribe to bus for live updates:
   r.graph.Bus().Subscribe(types.SubscriptionPattern("knowledge.*"),
       func(ev event.Event) {
           // Route to appropriate store method based on event type
       })

   In the handler:
   - knowledge.insight.recorded → unmarshal, convert, ks.Record()
   - knowledge.insight.superseded → unmarshal, ks.Supersede()
   - knowledge.insight.expired → unmarshal, ks.Expire()

e. Store the knowledgeStore reference on the Runtime struct so it can
   be passed to Loop.Config for each agent.

f. Add KnowledgeStore to Loop.Config:
   In pkg/loop/loop.go, add to the Config struct:
   KnowledgeStore interface{} // knowledge.KnowledgeStore — use interface{}
                              // to avoid circular import, or use the actual
                              // type if the import path works cleanly

   IMPORTANT: Check if pkg/loop can import pkg/knowledge without creating
   a circular dependency. If it can, use *knowledge.KnowledgeStore directly.
   If it can't, define a minimal interface in pkg/loop that knowledge.Store
   satisfies:
   type KnowledgeQuerier interface {
       Query(filter interface{}, maxResults int) []interface{}
   }
   ... or find another clean approach. Document what you chose.

g. When creating each agent's Loop config in the runtime (wherever
   Config{} structs are assembled), add:
   KnowledgeStore: knowledgeStore

h. Store the system actor reference so the distiller (Prompt 8) can use it.

TASK 4: Tests

Create pkg/knowledge/replay_test.go:

1. TestReplayFromStore_Empty — empty store, no errors
2. TestReplayFromStore_RecordedInsight — record event → insight in store
3. TestReplayFromStore_SupersededInsight — record + supersede → old inactive
4. TestReplayFromStore_ExpiredInsight — record + expire → inactive
5. TestConvertFromEventContent — fields map correctly, TTL→ExpiresAt computed

For store tests, use the in-memory event store from eventgraph (the same
one used in other hive tests). Check how existing hive tests create test
stores — follow that pattern.

Run all tests. Run linter. Nothing breaks.

Commit with: "feat: add system actor, chain replay, and runtime wiring for knowledge store

- Register ActorTypeSystem actor at runtime boot (first system actor)
- ReplayFromStore reconstructs knowledge from 3x ByType() chain queries
- Knowledge store wired into runtime: create, replay, prune, bus subscribe
- KnowledgeStore added to Loop.Config for universal enrichment access
- Survives hive reboot via chain replay (first boot-replay component)

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 6 — Enrichment Pipe

```
RECON CONTEXT:
- Insertion point: loop.go:398, after enrichSpawnObservation, before return
- Guard: l.config.KnowledgeStore == nil (NOT role-gated)
- Existing enrichments all follow: guard → collect → format with
  strings.Builder → return obs + sb.String()
- No stabilization constant exists — each agent checks l.iteration
  against a hardcoded number. Use 10 for knowledge enrichment.
- enrichReviewObservation does NOT exist yet. Place after whatever the
  last enrichment call is at the time you implement this.

Create pkg/loop/knowledge.go:

1. func (l *Loop) enrichKnowledgeObservation(obs string) string

   CRITICAL: This is NOT role-gated. It runs for ALL agents.
   There is no "if l.agent.Role() != ..." guard.

   Guards (in order):
   a. if l.config.KnowledgeStore == nil { return obs }
   b. if l.iteration < 10 { return obs }  // stabilization

   Then:
   - Access the KnowledgeStore from l.config
   - Build a KnowledgeFilter:
     Role: string(l.agent.Role())
     MinConfidence: 0.3
     MaxAge: 72 * time.Hour
   - Call Query(filter, MaxEnrichmentItems)
   - If empty: return obs (no block at all)
   - Format and return: obs + formatKnowledgeBlock(insights)

   NOTE: If you had to use an interface type for KnowledgeStore in Config
   to avoid circular imports (from Prompt 5), do the type assertion here:
   ks, ok := l.config.KnowledgeStore.(knowledge.KnowledgeStore)
   if !ok { return obs }

2. func formatKnowledgeBlock(insights []knowledge.KnowledgeInsight) string

   Use strings.Builder. Format:

   === INSTITUTIONAL KNOWLEDGE ===
   The following insights are distilled from the civilization's
   accumulated experience. Consider them when making decisions.

   [1] (domain: health, confidence: 0.85, evidence: 23 events)
       Memory pressure warnings correlate with CTO Opus iterations.

   [2] ...

   ===

   Constraints:
   - Max items: knowledge.MaxEnrichmentItems (5)
   - Per item: truncate Summary to knowledge.MaxItemChars (300) with "..."
   - Total block: truncate to knowledge.MaxBlockChars (1800)

3. Wire into loop.go:

   At the current insertion point (after the last enrichment, before return):

   // Enrich observation with institutional knowledge for ALL agents.
   enriched = l.enrichKnowledgeObservation(enriched)
   return enriched, nil

   If enrichReviewObservation has been added by the parallel Reviewer
   implementation, place after it. If not, place after
   enrichSpawnObservation.

Run go vet and existing tests. Nil-safe guard means existing tests
(which don't set KnowledgeStore) pass unchanged.

Commit with: "feat: add universal knowledge enrichment pipe

- enrichKnowledgeObservation runs for ALL agents (not role-gated)
- Queries knowledge store with role-filtered relevance scoring
- Formats === INSTITUTIONAL KNOWLEDGE === block
- Token budget: 5 items, 300 chars/item, 1800 chars total
- Nil-safe: no-op when KnowledgeStore not configured
- Insertion point: after all role-specific enrichments, before return

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 7 — Enrichment Pipe Tests

```
Read pkg/loop/knowledge.go. Create comprehensive tests.

Create pkg/loop/knowledge_test.go:

Look at how existing loop tests are structured (health_test.go,
budget_test.go, spawner_test.go). Follow the same test setup pattern
for creating Loop instances with mock/test configs.

1. TestEnrichKnowledge_NilStore
   - Config with KnowledgeStore = nil
   - Returns observation unchanged

2. TestEnrichKnowledge_StabilizationWindow
   - Config with KnowledgeStore set, iteration = 3
   - Returns observation unchanged

3. TestEnrichKnowledge_NoInsights
   - Empty KnowledgeStore, iteration = 15
   - Returns observation unchanged (no block appended)

4. TestEnrichKnowledge_HasInsights
   - Record 3 insights relevant to agent's role
   - Output contains "=== INSTITUTIONAL KNOWLEDGE ==="
   - Output contains all 3 summaries
   - Output ends with "==="

5. TestEnrichKnowledge_RoleFiltering
   - 2 insights: one for "allocator", one for "cto"
   - Loop with role "allocator" → only allocator insight

6. TestEnrichKnowledge_UniversalInsights
   - 1 insight with empty RelevantRoles
   - Loop with any role → insight appears

7. TestEnrichKnowledge_MaxItems
   - 10 relevant insights, output has at most MaxEnrichmentItems

8. TestEnrichKnowledge_TruncateLongSummary
   - 500-char summary → truncated to MaxItemChars with "..."

9. TestEnrichKnowledge_TotalBlockSize
   - 5 insights with long summaries → total <= MaxBlockChars

10. TestEnrichKnowledge_AllRolesReceive
    CRITICAL TEST: This validates the "not role-gated" property.
    - Create loops for 5 different roles: "guardian", "sysmon",
      "allocator", "cto", "implementer"
    - Record a universal insight
    - Call enrichKnowledgeObservation for each
    - ALL of them receive the knowledge block

11. TestFormatKnowledgeBlock_Format
    - 2 insights → verify numbered items, domain/confidence/evidence
      metadata, header and footer

Run all tests with -race flag.

Commit with: "test: add knowledge enrichment pipe tests

- 11 cases: nil store, stabilization, filtering, token budget
- Critical: verifies ALL roles receive knowledge (not role-gated)

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 8 — Automated Distillers

```
RECON CONTEXT:
- Distillers query the chain using store.ByType(type, limit, after)
- They emit events using the system actor registered in Prompt 5
- The system actor and its signer need to be passed to the distiller
- The distiller also needs a reference to the event bus (to call
  graph.Record() or bus.Publish()) — check how the Runtime emits events
  (runtime.emit() at runtime.go:331) and follow that pattern

Create pkg/knowledge/distill.go:

1. Distiller struct:

   type Distiller struct {
       store    store.Store          // event chain (for historical queries)
       ks       KnowledgeStore       // knowledge store (for dedup + immediate recording)
       emitter  EventEmitter         // interface for emitting events to the chain
       interval time.Duration
       known    map[string]bool      // insight IDs already emitted (dedup)
       mu       sync.Mutex
   }

   The EventEmitter interface should wrap whatever mechanism the runtime
   uses to emit signed events. Look at runtime.emit() — it calls
   graph.Record() with the human actor's ID. The distiller needs the
   same but with the system actor's ID.

   Define a minimal interface:
   type EventEmitter interface {
       Emit(eventType string, content interface{}) error
   }

   Implement it in the runtime using the system actor's graph/signer.

   func NewDistiller(store store.Store, ks KnowledgeStore, emitter EventEmitter, interval time.Duration) *Distiller

2. Run(ctx context.Context):
   - time.NewTicker(d.interval)
   - Each tick: d.runAllDetectors()
   - Exit on ctx.Done()

3. runAllDetectors():
   - Call each detector
   - For each insight: check d.known[InsightID]
   - If not known: call d.emitter.Emit() + d.ks.Record(), add to d.known

4. DISTILLER 1: detectHealthCorrelations()

   - Query store.ByType("health.report", 200, "") for recent health reports
   - For each with severity warning or critical (unmarshal the content to
     check the Overall score — Overall < 1.0 means non-ok):
     * Query store.Recent(20, "") around that time window
     * Count which agent sources appear most frequently
   - If same agent correlates with 3+ severity events:
     * InsightID = deterministic: fmt.Sprintf("health-corr-%s-%s",
       agentName, time.Now().Format("2006-01-02"))
     * domain: "health"
     * confidence: min(0.5 + float64(correlationCount)*0.1, 0.95)
     * evidence_count: total health events scanned
     * source: "distiller:health-correlation"
     * ttl: 168 (7 days)
     * relevant_roles: ["sysmon", "allocator", correlated agent name]

5. DISTILLER 2: detectBudgetEffectiveness()

   - Query store.ByType("agent.budget.adjusted", 100, "")
   - For each adjustment: identify the target agent and the increase amount
     (unmarshal the AgentBudgetAdjustedContent)
   - Query store.BySource(targetAgentID, 50, "") for events before and
     after the adjustment timestamp
   - Compare output rates (events per time unit)
   - If 3+ adjustments show < 20% output increase:
     * InsightID = deterministic
     * domain: "budget"
     * confidence: based on adjustment count
     * source: "distiller:budget-effectiveness"
     * ttl: 336 (14 days)
     * relevant_roles: ["allocator", target agent name, "cto"]

6. IMPORTANT: Both distillers must handle empty query results gracefully.
   If no health.report or budget.adjusted events exist yet, return empty
   slice. No errors, no panics.

7. Start the distiller in Runtime.Run():
   distiller := knowledge.NewDistiller(
       r.graph.Store(),
       knowledgeStore,
       systemEmitter,   // wraps system actor + graph
       5 * time.Minute,
   )
   go distiller.Run(ctx)

Run go vet.

Commit with: "feat: add automated knowledge distillers

- Distiller framework: background goroutine, dedup, lifecycle
- Health correlation detector: agent activity vs severity events
- Budget effectiveness tracker: diminishing returns detection
- Uses system actor for event signing (not human operator)
- Resilient to missing data

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Prompt 9 — Tests + Agent Prompts + Integration

```
Three tasks:

TASK 1: Distiller tests

Create pkg/knowledge/distill_test.go:

1. TestDistiller_Dedup — run detector twice, second produces nothing

2. TestDistiller_HealthCorrelation_Found — chain with severity events
   clustered around one agent → insight returned

3. TestDistiller_HealthCorrelation_NoPattern — all ok severity → empty

4. TestDistiller_BudgetEffectiveness_DiminishingReturns — budget
   increases without output increase → insight returned

5. TestDistiller_BudgetEffectiveness_Effective — output scaled → empty

6. TestDistiller_EmptyChain — both detectors on empty chain → empty, no panic

7. TestDistiller_InsightEmission — detector produces insight → appears
   in KnowledgeStore AND on chain (via mock emitter)

Use in-memory event stores for test setup. Create test helper functions
for building health.report and budget.adjusted events with known content.

TASK 2: Agent prompt updates

For each agent prompt file in agents/ that corresponds to a running or
soon-to-be-running agent (guardian.md, sysmon.md, allocator.md, cto.md,
spawner.md, strategist.md, planner.md, implementer.md):

Add this section in a natural location (after the role-specific
observation context, before Anti-patterns):

## Institutional Knowledge

Each iteration, your observation may include an
=== INSTITUTIONAL KNOWLEDGE === block containing insights distilled from
the civilization's accumulated experience. These are evidence-based
patterns detected across many events.

Use them as context for your decisions. They are not commands — they are
observations about how the civilization behaves. If an insight is relevant
to your current task, factor it in. If not, ignore it. You may disagree
with an insight if you observe contradicting evidence.

Additionally, update guardian.md specifically — add:

## Knowledge Integrity

The civilization accumulates institutional knowledge via
knowledge.insight.recorded events. Monitor for:
- Malformed insights (missing required fields)
- Any source emitting more than 10 insights per hour (flooding)
- Contradictory active insights that should supersede each other
This is about structural integrity, not content correctness.

TASK 3: Integration test

Create a test (in pkg/knowledge/ or an appropriate test file) that
validates the end-to-end pipeline:

TestKnowledgeCompoundLoop:

1. Create an in-memory event store (from eventgraph)
2. Create a KnowledgeStore via NewStore()
3. Record several health.report events on the chain with varying
   severity and different agent sources (simulate a pattern where
   one agent correlates with severity spikes)
4. Create a Distiller with a mock emitter
5. Run detectHealthCorrelations()
6. Verify: insight returned with the correlated agent identified
7. Record the insight in the KnowledgeStore
8. Create a minimal Loop setup with role "allocator" and the
   KnowledgeStore
9. Call enrichKnowledgeObservation()
10. Verify: output contains === INSTITUTIONAL KNOWLEDGE ===
11. Verify: the health correlation insight appears in the output

This proves: events → distill → store → enrich → agent sees knowledge.

Run ALL tests with -race flag. Everything must pass.

Commit with: "feat: complete knowledge enrichment infrastructure

- Distiller tests: dedup, pattern detection, emission (7 cases)
- Agent prompts: all running agents aware of institutional knowledge
- Guardian: knowledge integrity monitoring
- Integration test: events → distill → store → enrich end-to-end

Co-Authored-By: transpara-ai (transpara-ai@transpara.com)"
```

---

## Post-Implementation Verification

```
Rebuild the hive binary and restart the service:

cd ~/transpara-ai/repos/lovyou-ai-hive
go build -o /home/transpara/bin/hive ./cmd/hive
systemctl --user restart lovyou-hive.service
sleep 15
systemctl --user status lovyou-hive.service
journalctl --user -u lovyou-hive.service --no-pager -n 80

Verify:
1. Hive boots without errors
2. System actor registration succeeds (look for "system" actor log)
3. Knowledge store initialized (replay from chain — may have 0 events
   on first boot since no knowledge events exist yet)
4. Distiller goroutine starts (look for log messages)
5. Pruner goroutine starts
6. All 8 existing agents boot and function normally
7. Agent observations include the enrichment call (may produce no
   === INSTITUTIONAL KNOWLEDGE === block initially if store is empty)
8. After ~5 minutes (one distiller cycle), check:
   a. Did any distillers detect patterns? (Requires sufficient chain
      history — health reports, budget adjustments)
   b. If insights were produced, do they appear as
      knowledge.insight.recorded events on the chain?
   c. Do subsequent agent observations include the
      === INSTITUTIONAL KNOWLEDGE === block with those insights?

9. Test the compound loop:
   - Let the hive run for 30+ minutes
   - Check if insights accumulate
   - Check if agents reference knowledge in their reasoning
   - The civilization is now learning from its own history

10. Verify no performance degradation:
    - Agent iteration times should not increase significantly
    - The universal enrichment adds ~450 tokens max per observation
    - Check that Haiku agents (SysMon, Allocator) aren't overwhelmed

Report back:
- Did the system actor register?
- Did replay find any existing knowledge events? (Should be 0 first time)
- Did the distillers produce insights after one cycle?
- Did the enrichment pipe inject insights into agent observations?
- Any errors, warnings, or unexpected behavior?

If the store initializes, the enrichment pipe runs for all agents, and
distillers start producing insights, the knowledge enrichment
infrastructure is graduated.
```

---

## Summary of Recon-Driven Changes from v1.0.0

| Area | v1.0.0 Assumption | v1.1.0 Reality |
|------|-------------------|----------------|
| Store query | Assumed prefix query | ByType() only — 3 separate calls |
| System actor | Assumed exists | Must create and register at boot |
| Boot replay | Assumed established pattern | First component to do this — new territory |
| Bus subscription | Assumed might not work | Confirmed: glob pattern matching works |
| enrichReviewObservation | Might exist | Does not exist yet |
| Stabilization constant | Assumed shared constant | Each agent hardcodes — use 10 |
| Event type location | In hive/events.go? | In eventgraph, following hive_event_types.go |
| Config extensibility | Might need structural changes | Simple field addition, nil-check pattern |
