package work_test

import (
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

type phase3FactContent struct {
	eventType string
}

func (c phase3FactContent) EventTypeName() string            { return c.eventType }
func (c phase3FactContent) Accept(event.EventContentVisitor) {}

func appendPhase3Fact(t *testing.T, s *store.InMemoryStore, eventType types.EventType, causes []types.EventID) types.EventID {
	t.Helper()
	registry := event.DefaultRegistry()
	registry.Register(eventType, nil)
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(eventType, testActor, phase3FactContent{eventType: eventType.Value()}, causes, testConv, s, testSigner{})
	if err != nil {
		t.Fatalf("create phase3 fact: %v", err)
	}
	stored, err := s.Append(ev)
	if err != nil {
		t.Fatalf("append phase3 fact: %v", err)
	}
	return stored.ID()
}

func addRequiredGateArtifacts(t *testing.T, ts *work.TaskStore, taskID types.EventID, causes []types.EventID) {
	t.Helper()
	for _, label := range work.RequiredReadinessGateLabels() {
		if err := ts.AddArtifact(testActor, taskID, label, "text/markdown", "gate body", causes, testConv); err != nil {
			t.Fatalf("AddArtifact %s: %v", label, err)
		}
	}
}

func TestTaskStore_ReadinessRequiresPhase3FactType(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	factType := types.MustEventType("authority.decision.recorded")

	task, err := ts.Create(testActor, "Merge approved authority work", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addRequiredGateArtifacts(t, ts, task.ID, causes)
	if err := ts.AddFactRequirement(testActor, task.ID, factType, types.EventID{}, "merge requires authority decision evidence", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}

	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if readiness.Ready {
		t.Fatal("Ready = true; want false until authority fact exists")
	}
	if len(readiness.MissingFacts) != 1 || readiness.MissingFacts[0] != factType.Value() {
		t.Fatalf("MissingFacts = %#v, want %q", readiness.MissingFacts, factType.Value())
	}

	appendPhase3Fact(t, s, factType, []types.EventID{task.ID})

	readiness, err = ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness after fact: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("Ready = false after fact; missing facts %#v", readiness.MissingFacts)
	}
	if len(readiness.PresentFacts) != 1 || readiness.PresentFacts[0] != factType.Value() {
		t.Fatalf("PresentFacts = %#v, want %q", readiness.PresentFacts, factType.Value())
	}
}

func TestTaskStore_ReadinessIgnoresUnlinkedPhase3FactType(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	factType := types.MustEventType("authority.decision.recorded")

	task, err := ts.Create(testActor, "Wait for linked authority work", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addRequiredGateArtifacts(t, ts, task.ID, causes)
	if err := ts.AddFactRequirement(testActor, task.ID, factType, types.EventID{}, "requires causally linked authority decision", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}
	appendPhase3Fact(t, s, factType, causes)

	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if readiness.Ready {
		t.Fatal("Ready = true; want false for unrelated authority fact")
	}
	if len(readiness.MissingFacts) != 1 || readiness.MissingFacts[0] != factType.Value() {
		t.Fatalf("MissingFacts = %#v, want %q", readiness.MissingFacts, factType.Value())
	}
}

func TestTaskStore_ReadinessExactFactIDMayReferenceExistingFact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	factType := types.MustEventType("authority.decision.recorded")

	task, err := ts.Create(testActor, "Pin existing authority decision", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addRequiredGateArtifacts(t, ts, task.ID, causes)
	factID := appendPhase3Fact(t, s, factType, causes)
	if err := ts.AddFactRequirement(testActor, task.ID, factType, factID, "pin approved decision", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}

	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("Ready = false for exact existing fact; missing %#v", readiness.MissingFacts)
	}
}

func TestTaskStore_ReadinessRequiresExactPhase3FactID(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	factType := types.MustEventType("agent.lifecycle.transitioned")

	task, err := ts.Create(testActor, "Begin persistent agent work", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addRequiredGateArtifacts(t, ts, task.ID, causes)
	factID := appendPhase3Fact(t, s, factType, causes)
	if err := ts.AddFactRequirement(testActor, task.ID, factType, factID, "requires active lifecycle transition", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}

	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	want := factType.Value() + "#" + factID.Value()
	if !readiness.Ready {
		t.Fatalf("Ready = false; missing facts %#v", readiness.MissingFacts)
	}
	if len(readiness.PresentFacts) != 1 || readiness.PresentFacts[0] != want {
		t.Fatalf("PresentFacts = %#v, want %q", readiness.PresentFacts, want)
	}
}

func TestTaskStore_ListSummariesIncludesMissingPhase3Facts(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	factType := types.MustEventType("agent.identity.registered")

	task, err := ts.Create(testActor, "Assign work to new agent", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addRequiredGateArtifacts(t, ts, task.ID, causes)
	if err := ts.AddFactRequirement(testActor, task.ID, factType, types.EventID{}, "agent must be registered first", causes, testConv); err != nil {
		t.Fatalf("AddFactRequirement: %v", err)
	}

	summaries, err := ts.ListSummaries(10)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].Ready {
		t.Fatal("summary Ready = true; want false while fact is missing")
	}
	if len(summaries[0].MissingFacts) != 1 || summaries[0].MissingFacts[0] != factType.Value() {
		t.Fatalf("summary MissingFacts = %#v, want %q", summaries[0].MissingFacts, factType.Value())
	}
}
