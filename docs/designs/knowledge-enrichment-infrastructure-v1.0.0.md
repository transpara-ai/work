# Knowledge Enrichment Infrastructure — Framework Design

**Version:** 1.0.0
**Last Updated:** 2026-04-06
**Status:** Draft
**Versioning:** Independent of all other documents.
**Author:** Claude Opus 4.6
**Owner:** Michael Saucier

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-04-06 | Initial design: problem statement, three-layer architecture (produce → store → enrich), event types, knowledge store, relevance engine, enrichment pipe, automated distillers, integration with existing agents, recon items, implementation sequence. |

---

## 0. The Problem

The civilization has amnesia.

Every agent boots fresh. The event chain records everything that happened,
but no mechanism distills what was *learned*. SysMon has flagged memory
pressure 47 times — but no agent knows "memory pressure correlates with
CTO's Opus iterations." The Reviewer will reject code for missing error
handling — but no agent will notice "the implementer forgets error handling
on database calls 60% of the time." The Allocator adjusts budgets — but
nobody tracks whether those adjustments actually improved throughput.

The chain is an audit log, not a nervous system. It answers "what happened"
but never "what does that mean" or "what should we do differently."

This document designs the infrastructure that closes that gap. Not the
agents that produce knowledge (MemoryKeeper, Analyst, Historian — those
are Tier B and should emerge from the growth loop). The *pipe* — the
framework-level mechanism by which knowledge, once produced, flows back
into every agent's decision-making context.

### What Exists Today

```
Agent Observation Pipeline (per tick):

  1. Collect bus events matching WatchPatterns    ← generic, all agents
  2. Format as text observation                   ← generic, all agents
  3. Role-specific enrichment:                    ← bespoke, per-role
     - SysMon:    enrichHealthObservation()       === HEALTH METRICS ===
     - Allocator: enrichBudgetObservation()       === BUDGET METRICS ===
     - CTO:       enrichCTOObservation()          === LEADERSHIP BRIEFING ===
     - Spawner:   enrichSpawnObservation()        === SPAWN CONTEXT ===
     - Reviewer:  enrichReviewObservation()       === CODE REVIEW CONTEXT ===
  4. Inject SystemPrompt                          ← generic, all agents
  5. Send to LLM                                  ← generic, all agents
```

Every enrichment function is role-gated (`if l.agentDef.Role != "xxx"
{ return obs }`). No enrichment is universal. There is no shared layer
where accumulated knowledge can enter the observation for *all* agents.

### What Should Exist

```
Agent Observation Pipeline (per tick):

  1. Collect bus events matching WatchPatterns     ← generic, all agents
  2. Format as text observation                    ← generic, all agents
  3. Role-specific enrichment                      ← bespoke, per-role (unchanged)
  4. ★ Knowledge enrichment                        ← NEW: universal, all agents
  5. Inject SystemPrompt                           ← generic, all agents
  6. Send to LLM                                   ← generic, all agents
```

Step 4 is the subject of this document.

---

## 1. Design Principles

### Universal, Not Role-Gated

Every previous enrichment runs only for its designated role. Knowledge
enrichment runs for *all* agents. The SysMon should know "budget
adjustments to CTO improved throughput 40% last session." The CTO should
know "the implementer consistently struggles with concurrent Go patterns."
The Guardian should know "three consecutive SysMon health reports preceded
each of the last two chain integrity violations."

This is the architectural departure. Role-specific enrichment gives an
agent context about *its job*. Knowledge enrichment gives an agent context
about *the civilization's accumulated experience*.

### Relevance-Filtered, Not Firehosed

The civilization will accumulate hundreds of knowledge artifacts over time.
Injecting all of them into every observation is token suicide — and worse,
it drowns the signal the agent actually needs. The enrichment pipe must
filter knowledge by relevance to the agent's current role, current context,
and current pending events.

### Additive, Not Replacing

Knowledge enrichment appends a `=== INSTITUTIONAL KNOWLEDGE ===` block
after the role-specific enrichment. It never replaces or modifies the
role-specific context. An Allocator tick looks like:

```
[bus events]
=== BUDGET METRICS ===
[allocator-specific budget data]
===
=== INSTITUTIONAL KNOWLEDGE ===
[2-5 relevant knowledge items]
===
```

### Producer-Agnostic

The pipe doesn't care *who* produces knowledge. Initially, automated
distillers (Go code running in the framework) will produce basic pattern
detection. Later, the MemoryKeeper agent (Tier B, growth-loop-spawned)
will produce richer, LLM-synthesized insights. Even later, human operators
might inject knowledge directly. The pipe consumes knowledge events
regardless of source.

### Budget-Aware

Knowledge enrichment adds tokens to every observation. The infrastructure
must respect iteration budgets by enforcing a hard cap on the knowledge
block size. More knowledge is not always better — the best 3 insights are
worth more than 20 marginally relevant ones.

---

## 2. Three-Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    LAYER 3: ENRICHMENT PIPE                  │
│  enrichKnowledgeObservation() — universal, all agents       │
│  Queries KnowledgeStore → filters by relevance → formats    │
│  → appends === INSTITUTIONAL KNOWLEDGE === to observation   │
│  Location: pkg/loop/knowledge.go                            │
└──────────────────────────┬──────────────────────────────────┘
                           │ reads from
┌──────────────────────────▼──────────────────────────────────┐
│                    LAYER 2: KNOWLEDGE STORE                  │
│  In-memory index of active knowledge artifacts              │
│  Populated from chain events at boot + live updates         │
│  Queryable by domain, role relevance, recency, confidence   │
│  Location: pkg/knowledge/store.go                           │
└──────────────────────────┬──────────────────────────────────┘
                           │ populated by
┌──────────────────────────▼──────────────────────────────────┐
│                    LAYER 1: KNOWLEDGE PRODUCERS              │
│  Automated distillers (Go code, framework-level)            │
│  + MemoryKeeper agent (future, Tier B)                      │
│  + Human operator injection (future)                        │
│  All emit knowledge.* events on the chain                   │
│  Location: pkg/knowledge/distill.go (automated)             │
│            agents/memorykeeper.md (future agent)             │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Event Types

### New Event Types in EventGraph

```go
// knowledge.insight.recorded — A distilled insight from pattern analysis.
// Emitted by automated distillers or the MemoryKeeper agent.
EventTypeKnowledgeInsightRecorded = EventType{Value: "knowledge.insight.recorded"}

// knowledge.insight.superseded — An older insight replaced by a newer one.
// Prevents stale knowledge from persisting indefinitely.
EventTypeKnowledgeInsightSuperseded = EventType{Value: "knowledge.insight.superseded"}

// knowledge.insight.expired — An insight aged out without supersession.
// Automatic TTL-based expiration.
EventTypeKnowledgeInsightExpired = EventType{Value: "knowledge.insight.expired"}
```

### Content Structs

```go
// KnowledgeInsightContent represents a single distilled insight.
type KnowledgeInsightContent struct {
    // InsightID is a unique identifier for this insight (UUID).
    InsightID string `json:"insight_id"`

    // Domain is the ontological domain this insight belongs to.
    // Maps to Layer numbers or cross-cutting concerns.
    // Examples: "health", "budget", "quality", "architecture",
    //           "process", "performance", "patterns"
    Domain string `json:"domain"`

    // Summary is the human/agent-readable insight text.
    // Must be concise: 1-3 sentences max.
    // Example: "Budget increases to CTO beyond 60 iterations show
    //           diminishing returns — gap detection rate plateaus
    //           while token cost increases linearly."
    Summary string `json:"summary"`

    // RelevantRoles lists which roles this insight is most useful for.
    // Empty means relevant to all roles.
    // Example: ["allocator", "cto"] or ["implementer", "reviewer"]
    RelevantRoles []string `json:"relevant_roles"`

    // Confidence is how well-supported this insight is.
    // 0.0-1.0, where 1.0 means derived from extensive evidence.
    Confidence float64 `json:"confidence"`

    // EvidenceCount is how many events support this insight.
    // Higher count → higher confidence in the pattern.
    EvidenceCount int `json:"evidence_count"`

    // Source identifies what produced this insight.
    // "distiller:pattern" for automated, "memorykeeper" for agent,
    // "operator" for human-injected.
    Source string `json:"source"`

    // TTL is the time-to-live in hours. After this duration without
    // supersession, the insight expires automatically.
    // 0 means no expiration (permanent knowledge).
    TTL int `json:"ttl"`

    // SupersedesID optionally references an older insight this replaces.
    SupersedesID string `json:"supersedes_id,omitempty"`
}

// KnowledgeSupersessionContent records when an insight is replaced.
type KnowledgeSupersessionContent struct {
    OldInsightID string `json:"old_insight_id"`
    NewInsightID string `json:"new_insight_id"`
    Reason       string `json:"reason"`
}
```

### Why Three Event Types, Not One

An insight needs a lifecycle. Without supersession and expiration, the
knowledge store grows without bound and stale insights pollute agent
observations forever. The three events form a clean lifecycle:

```
recorded ──→ (active, queryable, injected into observations)
         │
         ├── superseded ──→ (inactive, replaced by newer insight)
         │
         └── expired ──→ (inactive, TTL elapsed, auto-pruned)
```

---

## 4. The Knowledge Store

### Purpose

An in-memory, queryable index of active knowledge artifacts. Populated
from chain events at boot (replay) and updated live as new knowledge
events arrive on the bus.

### Location

`pkg/knowledge/store.go`

### Interface

```go
// KnowledgeStore provides queryable access to the civilization's
// accumulated institutional knowledge.
type KnowledgeStore interface {
    // Record adds or updates an insight in the store.
    Record(insight KnowledgeInsight) error

    // Supersede marks an insight as replaced by a newer one.
    Supersede(oldID, newID string) error

    // Expire removes an insight that has exceeded its TTL.
    Expire(insightID string) error

    // Query returns insights matching the given filter, ordered by
    // relevance score (highest first), limited to maxResults.
    Query(filter KnowledgeFilter, maxResults int) []KnowledgeInsight

    // ActiveCount returns the number of currently active insights.
    ActiveCount() int

    // PruneExpired removes all insights past their TTL.
    // Called periodically by the framework.
    PruneExpired() int
}
```

### The Insight Record

```go
// KnowledgeInsight is the in-memory representation of an active insight.
type KnowledgeInsight struct {
    InsightID     string
    Domain        string
    Summary       string
    RelevantRoles []string  // empty = relevant to all
    Confidence    float64
    EvidenceCount int
    Source        string
    RecordedAt    time.Time
    ExpiresAt     time.Time // zero value = never expires
    Active        bool
}
```

### The Query Filter

```go
// KnowledgeFilter specifies which insights to retrieve.
type KnowledgeFilter struct {
    // Role filters to insights relevant to this role.
    // Matches insights where RelevantRoles contains this role
    // OR RelevantRoles is empty (universal insights).
    Role string

    // Domains filters to specific knowledge domains.
    // Empty means all domains.
    Domains []string

    // MinConfidence filters out low-confidence insights.
    // Default: 0.3 (exclude very uncertain insights).
    MinConfidence float64

    // MaxAge filters out insights older than this duration.
    // Zero means no age filter.
    MaxAge time.Duration

    // ExcludeIDs excludes specific insight IDs (e.g., already seen).
    ExcludeIDs []string
}
```

### Relevance Scoring

When `Query()` returns results, they're ordered by a composite relevance
score. This is the core of the "don't firehose" principle:

```go
// relevanceScore computes how relevant an insight is for a given query.
func relevanceScore(insight KnowledgeInsight, filter KnowledgeFilter) float64 {
    score := 0.0

    // 1. Role match (0.0 or 0.4)
    // Insights explicitly tagged for this role score higher than
    // universal insights. Both are included, but targeted ones win.
    if len(insight.RelevantRoles) == 0 {
        score += 0.2 // universal — somewhat relevant to everyone
    } else if containsRole(insight.RelevantRoles, filter.Role) {
        score += 0.4 // explicitly relevant — high signal
    } else {
        return 0.0 // not relevant to this role at all
    }

    // 2. Confidence weight (0.0-0.3)
    score += insight.Confidence * 0.3

    // 3. Recency bonus (0.0-0.2)
    // More recent insights score higher — they reflect current conditions.
    age := time.Since(insight.RecordedAt)
    if age < 1*time.Hour {
        score += 0.2
    } else if age < 6*time.Hour {
        score += 0.15
    } else if age < 24*time.Hour {
        score += 0.1
    } else {
        score += 0.05
    }

    // 4. Evidence weight (0.0-0.1)
    // Insights backed by more evidence are more trustworthy.
    if insight.EvidenceCount >= 10 {
        score += 0.1
    } else if insight.EvidenceCount >= 5 {
        score += 0.07
    } else {
        score += 0.03
    }

    return score
}
```

### Boot Replay

At hive boot, the knowledge store replays `knowledge.*` events from the
chain to reconstruct state:

```go
func (s *store) ReplayFromChain(events []event.Event) error {
    for _, e := range events {
        switch e.Type() {
        case event.EventTypeKnowledgeInsightRecorded:
            var content KnowledgeInsightContent
            // unmarshal, convert to KnowledgeInsight, call s.Record()
        case event.EventTypeKnowledgeInsightSuperseded:
            var content KnowledgeSupersessionContent
            // call s.Supersede()
        case event.EventTypeKnowledgeInsightExpired:
            // call s.Expire()
        }
    }
    return nil
}
```

This means knowledge survives reboots. The chain is the durable store;
the in-memory index is a projection. Same pattern as the BudgetRegistry
replaying budget events, or the actor store replaying actor events.

---

## 5. The Enrichment Pipe

### Location

`pkg/loop/knowledge.go`

### The Function

```go
// enrichKnowledgeObservation appends relevant institutional knowledge
// to any agent's observation. Unlike role-specific enrichment functions,
// this runs for ALL agents.
func (l *Loop) enrichKnowledgeObservation(obs string) string {
    store := l.config.KnowledgeStore
    if store == nil {
        return obs
    }

    // Don't enrich during stabilization window — let the agent
    // establish baseline behavior first.
    if l.iteration < l.config.StabilizationWindow {
        return obs
    }

    // Query for insights relevant to this agent's role.
    filter := knowledge.KnowledgeFilter{
        Role:          string(l.agent.Role()),
        MinConfidence: 0.3,
        MaxAge:        72 * time.Hour, // 3 days max
    }

    insights := store.Query(filter, maxKnowledgeItems)

    if len(insights) == 0 {
        return obs
    }

    return obs + formatKnowledgeBlock(insights)
}
```

### Output Format

```
=== INSTITUTIONAL KNOWLEDGE ===
The following insights are distilled from the civilization's accumulated
experience. Consider them when making decisions this iteration.

[1] (domain: budget, confidence: 0.85, evidence: 23 events)
    Budget increases to CTO beyond 60 iterations show diminishing
    returns — gap detection rate plateaus while token cost increases
    linearly.

[2] (domain: quality, confidence: 0.72, evidence: 11 events)
    The implementer's most common review rejection reason is unchecked
    error returns on database operations. Three of the last five
    request_changes verdicts cited this pattern.

[3] (domain: health, confidence: 0.91, evidence: 47 events)
    Memory pressure warnings correlate strongly with CTO Opus
    iterations. SysMon health reports show pressure spikes within
    2 iterations of CTO processing.
===
```

### Token Budget

The knowledge block has a hard ceiling to prevent observation bloat:

| Constraint | Value | Rationale |
|------------|-------|-----------|
| Max items | 5 | More than 5 insights is noise, not signal |
| Max chars per item | 300 | Forces concise summaries |
| Max total block size | 1,800 chars | ~450 tokens — about 5% of a typical observation |
| Min confidence | 0.3 | Below this, the insight isn't worth the tokens |

The `formatKnowledgeBlock()` function enforces these limits. If an insight
summary exceeds 300 chars, it's truncated with "..." — a signal to the
distiller/MemoryKeeper that the insight needs to be more concise.

### Where It Plugs In

In `pkg/loop/loop.go`, the observation pipeline currently looks like:

```go
// Pseudocode of current flow
obs := l.collectBusEvents()
obs = l.enrichHealthObservation(obs)      // SysMon only
obs = l.enrichBudgetObservation(obs, iter) // Allocator only
obs = l.enrichCTOObservation(obs)          // CTO only
obs = l.enrichSpawnObservation(obs)        // Spawner only
obs = l.enrichReviewObservation(obs)       // Reviewer only
// → send to LLM
```

Knowledge enrichment slots in after all role-specific enrichments:

```go
obs := l.collectBusEvents()
obs = l.enrichHealthObservation(obs)
obs = l.enrichBudgetObservation(obs, iter)
obs = l.enrichCTOObservation(obs)
obs = l.enrichSpawnObservation(obs)
obs = l.enrichReviewObservation(obs)
obs = l.enrichKnowledgeObservation(obs)    // ★ NEW — universal
// → send to LLM
```

This ordering is intentional: role-specific context first (the agent's
immediate job), institutional knowledge second (accumulated wisdom). The
LLM sees its job context before the civilization's context, which biases
attention correctly.

---

## 6. Knowledge Producers (Layer 1)

### Phase 1: Automated Distillers (Framework Code)

Before the MemoryKeeper agent exists, basic pattern detection runs as Go
code in the framework. These are simple, deterministic distillers that
scan the event chain for known patterns and emit knowledge events.

Location: `pkg/knowledge/distill.go`

```go
// Distiller runs periodic pattern detection over the event chain
// and emits knowledge.insight.recorded events when patterns are found.
type Distiller struct {
    store     event.Store
    actor     actor.Actor // system actor for signing knowledge events
    interval  time.Duration
    known     map[string]bool // insight IDs already emitted (dedup)
}
```

#### Distiller 1: Review Pattern Detector

Scans `code.review.submitted` events for recurring rejection reasons.

```go
// detectReviewPatterns looks for repeated rejection patterns.
//
// Pattern: if the same issue category appears in 3+ rejections
// within the last 50 review events, emit an insight.
//
// Example output:
//   domain: "quality"
//   summary: "The implementer's most common rejection reason is
//            unchecked error returns (4 of last 7 rejections)."
//   relevant_roles: ["implementer", "reviewer", "cto"]
//   confidence: 0.72
//   evidence_count: 7
//   source: "distiller:review-patterns"
//   ttl: 168 (7 days)
func (d *Distiller) detectReviewPatterns() []KnowledgeInsightContent
```

#### Distiller 2: Health Correlation Detector

Scans `health.report` events and correlates severity spikes with agent
activity.

```go
// detectHealthCorrelations finds agent-activity patterns that
// correlate with health degradation.
//
// Pattern: if severity=warning or severity=critical appears within
// 3 iterations of a specific agent's processing, and this pattern
// repeats 3+ times, emit an insight.
//
// Example output:
//   domain: "health"
//   summary: "Memory pressure warnings correlate with CTO Opus
//            iterations (3 of 4 warning events within 2 ticks)."
//   relevant_roles: ["sysmon", "allocator", "cto"]
//   confidence: 0.85
//   evidence_count: 12
//   source: "distiller:health-correlation"
//   ttl: 168 (7 days)
func (d *Distiller) detectHealthCorrelations() []KnowledgeInsightContent
```

#### Distiller 3: Budget Effectiveness Tracker

Scans `agent.budget.adjusted` events and correlates with subsequent
performance.

```go
// detectBudgetEffectiveness tracks whether budget adjustments
// produced the intended effect.
//
// Pattern: after Allocator increases an agent's budget, did that
// agent's output (events emitted, tasks completed) increase
// proportionally? If budget increases consistently fail to improve
// output for a specific agent, emit a diminishing-returns insight.
//
// Example output:
//   domain: "budget"
//   summary: "Budget increases to CTO beyond 60 iterations show
//            diminishing returns — gap detection rate plateaus."
//   relevant_roles: ["allocator", "cto"]
//   confidence: 0.78
//   evidence_count: 15
//   source: "distiller:budget-effectiveness"
//   ttl: 336 (14 days)
func (d *Distiller) detectBudgetEffectiveness() []KnowledgeInsightContent
```

#### Distiller 4: Gap Resolution Tracker

Scans `hive.gap.detected` → `hive.role.proposed` → `hive.role.approved` →
`hive.agent.spawned` sequences to track growth loop effectiveness.

```go
// detectGapResolutionPatterns tracks how effectively the growth
// loop resolves detected gaps.
//
// Patterns detected:
// - Average time from gap detection to agent spawn
// - Rejection rate and common rejection reasons
// - Whether spawned agents actually address the original gap
//
// Example output:
//   domain: "process"
//   summary: "Growth loop average resolution time is 12 iterations
//            from gap to running agent. Guardian rejection rate: 20%,
//            most common reason: overly broad watch patterns."
//   relevant_roles: ["cto", "spawner", "guardian"]
//   confidence: 0.90
//   evidence_count: 5
//   source: "distiller:gap-resolution"
//   ttl: 336 (14 days)
func (d *Distiller) detectGapResolutionPatterns() []KnowledgeInsightContent
```

#### Distiller Execution

The distiller runs as a background goroutine, similar to the telemetry
writer:

```go
func (d *Distiller) Run(ctx context.Context) {
    ticker := time.NewTicker(d.interval) // every 5 minutes
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            d.runAllDetectors()
        }
    }
}

func (d *Distiller) runAllDetectors() {
    detectors := []func() []KnowledgeInsightContent{
        d.detectReviewPatterns,
        d.detectHealthCorrelations,
        d.detectBudgetEffectiveness,
        d.detectGapResolutionPatterns,
    }

    for _, detect := range detectors {
        insights := detect()
        for _, insight := range insights {
            if d.known[insight.InsightID] {
                continue // dedup — already emitted
            }
            d.emitInsight(insight)
            d.known[insight.InsightID] = true
        }
    }
}
```

### Phase 2: MemoryKeeper Agent (Tier B — Growth Loop Spawned)

The MemoryKeeper is a *reader* of the chain and a *writer* of knowledge.
It uses LLM reasoning (Sonnet) to produce richer, more nuanced insights
than the deterministic distillers can.

The MemoryKeeper agent design is NOT part of this document — it's a Tier B
agent that should emerge from the growth loop when the CTO detects
"knowledge loss" as a gap. This document only specifies the infrastructure
the MemoryKeeper will use.

What the MemoryKeeper adds beyond automated distillers:

| Capability | Distiller | MemoryKeeper |
|-----------|-----------|--------------|
| Pattern detection | Hardcoded patterns | LLM-discovered patterns |
| Narrative synthesis | Counts and correlations | Contextual narratives |
| Cross-domain reasoning | Single-domain detectors | Multi-domain synthesis |
| Insight quality | Formulaic | Nuanced, contextual |
| Supersession | Manual dedup | Active insight curation |

The MemoryKeeper would use the same event types and knowledge store
interface. It would produce `knowledge.insight.recorded` events with
`source: "memorykeeper"` and potentially higher confidence (because
LLM reasoning over evidence is more reliable than regex pattern matching).

### Phase 3: Operator Injection (Future)

Human operators should be able to inject knowledge directly. This could
be a CLI command, a site UI, or a hive API endpoint:

```bash
hive knowledge add \
  --domain "architecture" \
  --summary "The implementer should use the repository pattern for all database access" \
  --roles "implementer,reviewer" \
  --confidence 1.0 \
  --ttl 0
```

This emits a `knowledge.insight.recorded` event with `source: "operator"`.
Operator-injected knowledge has implicit high confidence (the human said it)
and optionally no TTL (permanent policy).

---

## 7. Insight Lifecycle Management

### Supersession Protocol

When a newer insight replaces an older one (e.g., "implementer error
handling improved from 40% to 85% catch rate"), the producer:

1. Emits a new `knowledge.insight.recorded` event with
   `supersedes_id` pointing to the old insight
2. The knowledge store marks the old insight as inactive
3. A `knowledge.insight.superseded` event is recorded for audit

The chain preserves the full history (both insights exist as events).
The in-memory store only serves the active one.

### TTL Expiration

Every insight has a TTL (time-to-live in hours). The distiller framework
runs a periodic pruner:

```go
func (d *Distiller) pruneExpired() {
    count := d.knowledgeStore.PruneExpired()
    if count > 0 {
        // Emit knowledge.insight.expired events for audit
        for _, expired := range expiredInsights {
            d.emitExpiration(expired)
        }
    }
}
```

TTL guidelines:

| Source | Default TTL | Rationale |
|--------|------------|-----------|
| Automated distiller | 168h (7 days) | Patterns may shift; re-detect regularly |
| MemoryKeeper | 336h (14 days) | LLM synthesis is higher quality, longer-lived |
| Operator | 0 (permanent) | Human-injected policy until explicitly removed |

### Insight Cardinality

The knowledge store enforces a soft cap on active insights:

| Constraint | Value | Enforcement |
|-----------|-------|-------------|
| Max active insights | 100 | Lowest-confidence insights pruned when exceeded |
| Max per domain | 20 | Prevents single-domain domination |
| Max per source | 50 | Prevents any one producer from flooding |

These limits ensure the query response stays fast and the enrichment pipe
stays lean. If the civilization has 100 active insights but a given agent
only sees the 5 most relevant, the relevance engine is doing its job.

---

## 8. Integration with Existing Agents

### All Agents: Knowledge-Aware Observation

Every agent's prompt should include awareness that the
`=== INSTITUTIONAL KNOWLEDGE ===` block exists and what to do with it.

**Add to every agent's prompt (as a shared section):**

```markdown
## Institutional Knowledge

Each iteration, your observation may include an
=== INSTITUTIONAL KNOWLEDGE === block containing insights distilled from
the civilization's accumulated experience. These insights are patterns
detected across many events — things the civilization has learned.

Use them as context for your decisions. They are not commands — they are
evidence-based observations about how the civilization behaves. If an
insight is relevant to your current task, factor it in. If it's not
relevant this iteration, ignore it.

You may disagree with an insight. If you observe evidence that contradicts
a listed insight, note it in your reasoning. The insight may be stale or
based on different conditions.
```

This is a shared block that goes into every `agents/*.md` file. It doesn't
change agent behavior — it teaches agents that the block exists and how
to interpret it.

### Guardian: Knowledge Integrity

Guardian should watch `knowledge.*` events and verify:
- Insights are well-formed (valid JSON, required fields present)
- No insight claims confidence > 1.0 or < 0.0
- Supersession chains are valid (old insight exists and was active)
- No producer is flooding (rate check on knowledge emission)

Add to Guardian's WatchPatterns: `knowledge.*` (already covered by `*`
but should be explicitly noted in the prompt).

Add to Guardian prompt:

```markdown
## Knowledge Integrity

The civilization accumulates institutional knowledge via
knowledge.insight.recorded events. Monitor these for:
- Malformed insights (missing required fields)
- Unreasonable confidence claims (> 1.0)
- Flooding (any source emitting > 10 insights per session)
- Contradictory insights that should supersede each other

Knowledge integrity is not about whether insights are "correct" — that's
subjective. It's about whether they're well-formed, properly attributed,
and not polluting the knowledge store.
```

### CTO: Knowledge-Informed Gap Detection

CTO should consume knowledge insights to improve gap detection. If the
knowledge store shows "the implementer consistently struggles with
concurrent patterns," the CTO might detect a gap for a concurrency
mentor or a specialist reviewer.

CTO already watches `*` and will see `knowledge.insight.recorded` events.
The enrichment pipe ensures relevant insights appear in every CTO tick.
No CTO prompt changes needed — the leadership briefing + knowledge block
gives the CTO everything it needs.

### Allocator: Knowledge-Informed Budget Decisions

If the knowledge store shows "budget increases to CTO beyond 60 iterations
have diminishing returns," the Allocator can factor this into budget
decisions. No Allocator prompt changes needed — the budget metrics +
knowledge block provides the context.

---

## 9. The Knowledge Store in the Runtime

### Initialization

The KnowledgeStore is created at hive boot and passed to every Loop via
`Loop.Config`:

```go
// In pkg/hive/runtime.go or wherever agents are bootstrapped:

// Create knowledge store
knowledgeStore := knowledge.NewStore()

// Replay existing knowledge events from chain
knowledgeEvents := graph.Query(/* filter for knowledge.* events */)
knowledgeStore.ReplayFromChain(knowledgeEvents)

// Start automated distiller
distiller := knowledge.NewDistiller(graph, knowledgeStore, systemActor)
go distiller.Run(ctx)

// Start TTL pruner
go knowledgeStore.RunPruner(ctx, 15*time.Minute)

// Pass to each agent's Loop config
config := loop.Config{
    // ... existing fields ...
    KnowledgeStore: knowledgeStore,
}
```

### Bus Integration

The knowledge store also needs live updates from the event bus. When a
new `knowledge.insight.recorded` event arrives on the bus, the store
should update immediately (not wait for the next distiller tick):

```go
// In the event bus subscription:
bus.Subscribe("knowledge.*", func(e event.Event) {
    switch e.Type() {
    case "knowledge.insight.recorded":
        knowledgeStore.Record(/* convert event content to insight */)
    case "knowledge.insight.superseded":
        knowledgeStore.Supersede(/* ... */)
    case "knowledge.insight.expired":
        knowledgeStore.Expire(/* ... */)
    }
})
```

This ensures that when the MemoryKeeper (or any producer) emits a new
insight, all agents see it in their next iteration — not after a
multi-minute distiller cycle.

---

## 10. The Compound Loop

With all three layers in place, the civilization's learning cycle works:

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  Agents make decisions → produce events on chain        │
│         ↓                                               │
│  Distillers/MemoryKeeper scan events for patterns       │
│         ↓                                               │
│  Patterns become knowledge.insight.recorded events      │
│         ↓                                               │
│  Knowledge store indexes insights, scores relevance     │
│         ↓                                               │
│  Enrichment pipe injects relevant insights into         │
│  agent observations (=== INSTITUTIONAL KNOWLEDGE ===)   │
│         ↓                                               │
│  Agents make BETTER decisions (informed by patterns)    │
│         ↓                                               │
│  Better decisions produce better events                 │
│         ↓                                               │
│  Distillers detect IMPROVED patterns                    │
│         ↓                                               │
│  Old insights superseded by new ones                    │
│         ↓                                               │
│  Knowledge quality COMPOUNDS over time                  │
│                                                         │
│  ═══════════════════════════════════════════════         │
│  THIS IS HOW THE CIVILIZATION GETS SMARTER              │
│  ═══════════════════════════════════════════════         │
└─────────────────────────────────────────────────────────┘
```

The critical property: each cycle through the loop produces *higher
quality* knowledge than the previous cycle, because agents informed by
knowledge make better decisions, which produce more coherent patterns,
which distill into sharper insights. The civilization doesn't just
accumulate data — it refines understanding.

### The Compounding Effect Over Time

| Timeframe | Knowledge State | Agent Behavior |
|-----------|----------------|----------------|
| Boot (0h) | Empty store, no insights | Agents operate on prompt alone |
| 1 hour | 5-10 automated insights from distillers | Basic pattern awareness |
| 1 day | 20-40 insights, first supersessions | Agents adapt to observed patterns |
| 1 week | 50-80 curated insights (stale ones expired) | Institutional memory forms |
| 1 month | 80-100 high-quality, evidence-rich insights | Civilization has genuine expertise |

---

## 11. What This Does NOT Cover

This document designs the infrastructure. These are explicitly deferred:

| Component | Why Deferred |
|-----------|-------------|
| **MemoryKeeper agent** | Tier B — should emerge from growth loop |
| **Analyst agent** | Tier B — quantitative analysis over history |
| **Historian agent** | Tier B — narrative institutional memory |
| **Librarian agent** | Tier B — knowledge organization and indexing |
| **Cross-session memory** | Different problem — warm sessions / persistent memory |
| **Human-facing knowledge UI** | Site feature, not framework infrastructure |
| **Knowledge base search** | Product feature for Layer 6, not agent infrastructure |

The pipe is the plumbing. The agents that push interesting water through
it come later.

---

## 12. Open Questions for Recon (Prompt 0)

1. **Event chain query mechanism.** The distillers need to query recent
   events by type (e.g., "all code.review.submitted in the last 50 events").
   What query interface does the Store provide? Is there a
   `store.QueryByType()` or similar? CTO and Spawner enrichment pull from
   `l.pendingEvents` — is that sufficient, or do distillers need direct
   store access for historical queries?

2. **Bus subscription mechanism.** The knowledge store needs live event
   updates from the bus. How does the current bus work? Is there a
   `bus.Subscribe(pattern, callback)` interface, or does each agent pull
   from a channel? How do the existing agents receive events?

3. **Loop.Config extensibility.** Adding `KnowledgeStore` to Loop.Config
   requires modifying the config struct. How many fields does it currently
   have? Is there a pattern for adding new framework services (similar to
   how BudgetRegistry was added)?

4. **Background goroutine lifecycle.** The distiller and pruner run as
   background goroutines. How does the runtime manage goroutine lifecycle?
   Is there a context.Context passed through for graceful shutdown? How
   does the telemetry writer's goroutine lifecycle work (same pattern)?

5. **System actor for distiller events.** Automated distillers emit
   events on the chain. They need an ActorID to sign with. Is there a
   system actor already registered, or does each infrastructure component
   create its own? The telemetry writer — does it emit signed events?

6. **Event type registration location.** Following the Spawner and
   Reviewer patterns: where should `knowledge.insight.recorded` etc.
   be registered? In eventgraph's type registry, in hive's `events.go`,
   or both?

7. **Chain replay at boot.** To reconstruct the knowledge store on reboot,
   we need to replay `knowledge.*` events from the chain. How does the
   chain store support filtered replay? Is there an efficient way to
   query "all events of type knowledge.*" without scanning the entire
   chain?

8. **Observation pipeline code location.** Where exactly in `loop.go`
   does the observation get built? What's the function signature? Need
   the exact insertion point for `enrichKnowledgeObservation()`.

---

## 13. Implementation Sequence

```
Prompt 0:  Reconnaissance (answer the 8 questions above)
Prompt 1:  Event types in eventgraph + emit methods in agent
           (knowledge.insight.recorded, .superseded, .expired)
Prompt 2:  Knowledge store (pkg/knowledge/store.go, types.go)
           - In-memory store with Record/Supersede/Expire/Query
           - Relevance scoring
           - Boot replay from chain
           - TTL pruner goroutine
Prompt 3:  Knowledge store tests
           - Record, query, supersede, expire, relevance scoring,
             cardinality limits, boot replay
Prompt 4:  Enrichment pipe (pkg/loop/knowledge.go)
           - enrichKnowledgeObservation() — universal enrichment
           - formatKnowledgeBlock() — text formatting
           - Wire into loop.go observation pipeline
Prompt 5:  Enrichment pipe tests
           - With/without knowledge, relevance filtering, token budget,
             stabilization window, non-role-gated behavior
Prompt 6:  Automated distillers (pkg/knowledge/distill.go)
           - At least 2 of the 4 distillers (review patterns + health
             correlations are highest value)
           - Distiller goroutine lifecycle
Prompt 7:  Distiller tests + integration
           - Pattern detection, dedup, emission, lifecycle
Prompt 8:  Agent prompt updates
           - Shared "Institutional Knowledge" section in all agent prompts
           - Guardian knowledge integrity section
Prompt 9:  Integration test
           - Boot → distiller runs → insight recorded → enrichment pipe
             injects into agent observation → agent receives knowledge
```

### Parallel with Reviewer

This implementation is independent of the Reviewer agent. However, the
review pattern distiller (Distiller 1) requires `code.review.submitted`
events to exist on the chain, which means the Reviewer must be operational
before Distiller 1 produces useful output. The other three distillers
(health correlation, budget effectiveness, gap resolution) can run
immediately with existing event types.

The enrichment pipe (Prompts 4-5) and the knowledge store (Prompts 2-3)
have zero dependencies on the Reviewer and can be built and tested
immediately.

---

## 14. Exit Criteria

Knowledge Enrichment Infrastructure graduation requires ALL of the
following:

- [ ] `knowledge.insight.recorded` event type registered in eventgraph
- [ ] `knowledge.insight.superseded` event type registered
- [ ] `knowledge.insight.expired` event type registered
- [ ] `EmitKnowledgeInsight()` method on Agent
- [ ] `KnowledgeStore` interface implemented with in-memory store
- [ ] Query with relevance scoring returns role-filtered, ranked results
- [ ] Boot replay reconstructs store from chain events
- [ ] TTL pruner runs as background goroutine
- [ ] Cardinality limits enforced (100 active, 20 per domain, 50 per source)
- [ ] `enrichKnowledgeObservation()` runs for ALL agents (not role-gated)
- [ ] Knowledge block respects token budget (5 items, 1800 chars max)
- [ ] Stabilization window prevents enrichment in early iterations
- [ ] At least 2 automated distillers operational
- [ ] Distiller dedup prevents duplicate insight emission
- [ ] All agent prompts updated with "Institutional Knowledge" section
- [ ] Guardian prompt updated with knowledge integrity monitoring
- [ ] Knowledge survives hive reboot (chain replay verified)
- [ ] Unit test coverage ≥ 80% on pkg/knowledge/
- [ ] Integration test: distill → record → enrich → agent sees knowledge
- [ ] Linter passes, all tests pass

---

## 15. What Comes After

Once the knowledge enrichment infrastructure is operational:

1. **MemoryKeeper emergence.** CTO detects "knowledge quality is limited
   to automated pattern matching" as a gap. Spawner proposes MemoryKeeper.
   MemoryKeeper uses the exact same event types and store interface —
   just produces richer insights via LLM reasoning.

2. **Knowledge compounding validated.** After a week of operation, compare
   agent decision quality (review approval rates, budget effectiveness,
   gap detection accuracy) before and after knowledge enrichment. The
   civilization should measurably improve.

3. **Analyst and Historian emerge.** As the knowledge store grows, the
   CTO detects needs for quantitative analysis (Analyst) and narrative
   synthesis (Historian). Both are Tier B growth loop candidates.

4. **Operator knowledge UI.** A site feature that lets humans browse,
   inject, and curate knowledge artifacts. This is a product feature,
   not agent infrastructure.

The infrastructure is the foundation. Everything else grows on top of it.

---

*This document specifies the framework-level knowledge enrichment
infrastructure for the lovyou.ai civilization. It requires Prompt 0
reconnaissance against the codebase to resolve open questions before
implementation begins.*
