package work_test

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

// noiseEventContent is a minimal, non-work event type used to pad the
// store with events the /tasks fold domain must correctly ignore (D3:
// "mixed: many tasks ... + noise events of non-work types").
type noiseEventContent struct {
	eventType string
}

func (c noiseEventContent) EventTypeName() string            { return c.eventType }
func (c noiseEventContent) Accept(event.EventContentVisitor) {}

// seedLiveScaleStore populates s with approximately targetEvents events: a
// mixed domain of work.task.* activity (creation, assignment, artifacts
// with required gates, completion, occasional reopen, dependencies,
// lifecycle transitions) representative of live Dark Factory traffic, plus
// interleaved noise events of non-work types. Returns the number of tasks
// created and the total event count actually appended (including the
// bootstrap event).
func seedLiveScaleStore(tb testing.TB, s *store.InMemoryStore, factory *event.EventFactory, causes []types.EventID, targetEvents int) (taskCount int, totalEvents int) {
	tb.Helper()

	registry := event.DefaultRegistry()
	noiseType := types.MustEventType("bench.noise.recorded")
	registry.Register(noiseType, nil)
	noiseFactory := event.NewEventFactory(registry)

	ts := work.NewTaskStore(s, factory, testSigner{})

	// Budget: each "full" task lifecycle costs roughly 7 events (created,
	// assigned, 3 artifacts, completed, lifecycle transition); every 5th
	// task additionally gets a dependency edge and every 11th a reopen +
	// recompletion. One noise event is interleaved per task. This mix
	// approximates live Dark Factory traffic without requiring exact event
	// accounting — the loop stops once targetEvents is reached.
	var lastTaskID types.EventID
	for n := 0; totalEvents < targetEvents; n++ {
		task, err := ts.Create(testActor, fmt.Sprintf("bench task %d", n), "bench seed", causes, testConv)
		if err != nil {
			tb.Fatalf("seed Create %d: %v", n, err)
		}
		taskCount++
		totalEvents++ // created

		if err := ts.Assign(testActor, task.ID, testActor, causes, testConv); err != nil {
			tb.Fatalf("seed Assign %d: %v", n, err)
		}
		totalEvents++

		for _, label := range work.RequiredReadinessGateLabels() {
			if err := ts.AddArtifact(testActor, task.ID, label, "text/markdown", "gate body for "+label, causes, testConv); err != nil {
				tb.Fatalf("seed AddArtifact %d/%s: %v", n, label, err)
			}
			totalEvents++
		}

		if err := ts.Complete(testActor, task.ID, "bench completion", causes, testConv); err != nil {
			tb.Fatalf("seed Complete %d: %v", n, err)
		}
		totalEvents++

		if err := ts.TransitionTask(testActor, task.ID, work.StatusReady, "advance", nil, causes, testConv); err != nil {
			tb.Fatalf("seed TransitionTask %d: %v", n, err)
		}
		totalEvents++

		if n%5 == 0 && !lastTaskID.IsZero() {
			if err := ts.AddDependency(testActor, task.ID, lastTaskID, causes, testConv); err != nil {
				tb.Fatalf("seed AddDependency %d: %v", n, err)
			}
			totalEvents++
		}

		if n%11 == 0 {
			if err := ts.Reopen(testActor, task.ID, "bench reopen", []string{"fix it"}, causes, testConv); err != nil {
				tb.Fatalf("seed Reopen %d: %v", n, err)
			}
			totalEvents++
			if err := ts.AddArtifact(testActor, task.ID, "result", "text/plain", "fixed", causes, testConv); err != nil {
				tb.Fatalf("seed re-artifact %d: %v", n, err)
			}
			totalEvents++
			if err := ts.Complete(testActor, task.ID, "bench recompletion", causes, testConv); err != nil {
				tb.Fatalf("seed re-Complete %d: %v", n, err)
			}
			totalEvents++
		}

		// Noise event: not a work.task.* type, must be ignored by the fold.
		noiseEv, err := noiseFactory.Create(noiseType, testActor, noiseEventContent{eventType: noiseType.Value()}, causes, testConv, s, testSigner{})
		if err != nil {
			tb.Fatalf("seed noise create %d: %v", n, err)
		}
		if _, err := s.Append(noiseEv); err != nil {
			tb.Fatalf("seed noise append %d: %v", n, err)
		}
		totalEvents++

		lastTaskID = task.ID
	}
	return taskCount, totalEvents
}

// coldFoldBudget/warmFoldBudget are the production (non-race) latency
// budgets from packet D3: cold fold < 2s, warm (unchanged head, including
// the narrowed fact pass) < 100ms. Under -race, instrumentation overhead
// (roughly 4-10x per raceEnabled's doc comment) gets 4x headroom so the
// test measures real regressions, not race-detector cost.
const (
	coldFoldBudget = 2 * time.Second
	warmFoldBudget = 100 * time.Millisecond
	raceMultiplier = 4
)

func budgetFor(base time.Duration) time.Duration {
	if raceEnabled {
		return base * raceMultiplier
	}
	return base
}

// TestListSummariesCached_LiveScaleLatencyBudget seeds a ~25,000-event
// store (D3) and asserts cold ListSummariesCached < 2s (< 8s under -race)
// and warm (unchanged head) < 100ms (< 400ms under -race). Skipped under
// -short since it is a live-scale timed test, not a unit test.
func TestListSummariesCached_LiveScaleLatencyBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live-scale latency budget test in -short mode")
	}

	s, causes := setupStore(t)
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)

	const targetEvents = 25000
	taskCount, totalEvents := seedLiveScaleStore(t, s, factory, causes, targetEvents)
	t.Logf("seeded %d tasks, %d total events (target %d)", taskCount, totalEvents, targetEvents)

	ts := work.NewTaskStore(s, factory, testSigner{})

	// Settle the heap after seeding's allocation burst so a GC pause does
	// not bleed into the timed cold-fold window.
	runtime.GC()

	coldStart := time.Now()
	cold, err := ts.ListSummariesCached(100)
	coldElapsed := time.Since(coldStart)
	if err != nil {
		t.Fatalf("cold ListSummariesCached: %v", err)
	}
	if len(cold) == 0 {
		t.Fatal("cold ListSummariesCached returned no summaries")
	}
	coldBudget := budgetFor(coldFoldBudget)
	t.Logf("cold fold: %s (budget %s, raceEnabled=%v)", coldElapsed, coldBudget, raceEnabled)
	if coldElapsed >= coldBudget {
		t.Fatalf("cold fold wall-clock = %s, want < %s (raceEnabled=%v)", coldElapsed, coldBudget, raceEnabled)
	}

	warmStart := time.Now()
	warm, err := ts.ListSummariesCached(100)
	warmElapsed := time.Since(warmStart)
	if err != nil {
		t.Fatalf("warm ListSummariesCached: %v", err)
	}
	if len(warm) != len(cold) {
		t.Fatalf("warm returned %d summaries, cold returned %d", len(warm), len(cold))
	}
	warmBudget := budgetFor(warmFoldBudget)
	t.Logf("warm fold: %s (budget %s, raceEnabled=%v)", warmElapsed, warmBudget, raceEnabled)
	if warmElapsed >= warmBudget {
		t.Fatalf("warm fold wall-clock = %s, want < %s (raceEnabled=%v)", warmElapsed, warmBudget, raceEnabled)
	}
}

// TestListSummariesCached_ColdScalingNearLinear seeds a second store at
// ~50,000 events and asserts the cold-fold time stays near-linear relative
// to the ~25,000-event cold-fold time (D3), not quadratic. Skipped under
// -short.
//
// Budget note (measured via pprof while building this test): the fold
// ALGORITHM ITSELF profiles as O(events) — in a 40-iteration CPU profile,
// ListSummariesCached's own cumulative cost was ~3% of total sampled time,
// with the remainder in InMemoryStore.paginateReverse's CURSOR RESOLUTION,
// which does a linear backward scan over the type-indexed slice to find
// each page's "after" cursor (go/pkg/store/memory.go paginateReverse,
// lines ~428-437) — an O(n) operation per page, making a K-page walk over
// N items of one type cost O(N*K), i.e. mildly superlinear as N grows and
// K grows with it. This is a pre-existing InMemoryStore characteristic
// (TEST/DEV BACKEND ONLY, out of scope for this packet's non-goals — no
// store-internal changes) and does NOT reproduce on the production
// backend: PostgresStore.paginateReverse resolves its cursor via
// `SELECT seq FROM events WHERE id = $1`, an indexed O(log n)-or-better
// lookup (go/pkg/store/pgstore/pgstore.go paginateReverse). Filed
// upstream as a follow-up (InMemoryStore O(n) cursor lookup) — not fixed
// here per the packet's explicit non-goal against store-internal changes.
//
// The budget below (3.0x, rather than a naive 2.0x doubling) is
// deliberately set to absorb this documented, understood, backend-local
// tax while still catching a genuine algorithmic regression in the fold
// itself (which would show a much larger deviation, not a mild ~2.2-2.3x
// bump). The near-linear CLAIM in the design packet is about the fold
// algorithm's own complexity; this test's budget is calibrated to the
// store it can actually exercise in this worktree.
//
// Flake control (bounded retry): the assertion is a RATIO of two sub-200ms
// wall-clock measurements, which is inherently contention-sensitive — on a
// busy CI host, scheduler noise on one side alone can push a clean ~2x
// ratio past 3x (observed: 1.96x-2.89x on clean runs, 3.486x once under CPU
// contention) without any code regression. The check therefore re-measures
// up to maxRatioAttempts times and PASSES if ANY attempt's ratio lands
// within budget (the bounded-retry statistical pattern from hive PR #241,
// per the design packet's TDD plan). A genuine O(n^2) regression in the
// fold fails every attempt deterministically: its ratio inflation (~2x on
// top of the linear baseline, i.e. ~4x+) exceeds the budget regardless of
// contention, so the retry loosens only the noise sensitivity, never the
// regression gate. Seeding is done once per scale; only the timed fold
// samples are repeated per attempt.
func TestListSummariesCached_ColdScalingNearLinear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cold-scaling test in -short mode")
	}

	const nearLinearBudgetMultiplier = 3.0
	const maxRatioAttempts = 3

	// seedScale seeds a fresh store once; the timed samples reuse it with a
	// fresh TaskStore (and therefore a fresh, empty foldCache) per sample.
	type seededScale struct {
		s       *store.InMemoryStore
		factory *event.EventFactory
	}
	seedScale := func(targetEvents int) seededScale {
		s, causes := setupStore(t)
		registry := event.DefaultRegistry()
		work.RegisterWithRegistry(registry)
		factory := event.NewEventFactory(registry)
		taskCount, totalEvents := seedLiveScaleStore(t, s, factory, causes, targetEvents)
		t.Logf("seeded %d-event scale: %d tasks, %d events", targetEvents, taskCount, totalEvents)
		return seededScale{s: s, factory: factory}
	}

	// measure settles the heap (runtime.GC()) so allocation bursts do not
	// bleed a GC pause into the timed window, then takes the MEDIAN of
	// several independent cold folds to absorb scheduler/GC jitter at this
	// sub-100ms absolute scale. Each sample is a genuine cold fold: no fold
	// state is shared between samples.
	const samplesPerScale = 5
	measure := func(sc seededScale, label string) time.Duration {
		samples := make([]time.Duration, 0, samplesPerScale)
		for i := 0; i < samplesPerScale; i++ {
			ts := work.NewTaskStore(sc.s, sc.factory, testSigner{})
			runtime.GC()
			start := time.Now()
			if _, err := ts.ListSummariesCached(100); err != nil {
				t.Fatalf("cold ListSummariesCached (%s, sample=%d): %v", label, i, err)
			}
			samples = append(samples, time.Since(start))
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		return samples[len(samples)/2]
	}

	scale25k := seedScale(25000)
	scale50k := seedScale(50000)

	var lastRatio float64
	for attempt := 1; attempt <= maxRatioAttempts; attempt++ {
		elapsed25k := measure(scale25k, "25k")
		elapsed50k := measure(scale50k, "50k")

		// Guard against a degenerate near-zero baseline making the ratio
		// meaningless (e.g. if the 25k fold completed in under a millisecond).
		if elapsed25k <= 0 {
			t.Fatalf("attempt %d: 25k-scale cold fold measured as non-positive duration: %s", attempt, elapsed25k)
		}

		lastRatio = float64(elapsed50k) / float64(elapsed25k)
		maxAllowed := time.Duration(float64(elapsed25k) * nearLinearBudgetMultiplier)
		t.Logf("attempt %d/%d: 25k cold fold = %s, 50k cold fold = %s, 50k/25k ratio = %.3f (max allowed %.1fx = %s)",
			attempt, maxRatioAttempts, elapsed25k, elapsed50k, lastRatio, nearLinearBudgetMultiplier, maxAllowed)
		if elapsed50k < maxAllowed {
			return // within budget — near-linear scaling confirmed
		}
		t.Logf("attempt %d/%d exceeded the %.1fx budget; re-measuring (contention-sensitive wall-clock ratio — see doc comment)",
			attempt, maxRatioAttempts, nearLinearBudgetMultiplier)
	}
	t.Fatalf("all %d attempts exceeded the %.1fx near-linear budget (last ratio %.3f); scaling looks super-linear beyond the documented InMemoryStore cursor-resolution tax",
		maxRatioAttempts, nearLinearBudgetMultiplier, lastRatio)
}
