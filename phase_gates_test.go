package work_test

import (
	"strings"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/work"
)

func newPhaseGateStore(t *testing.T, s *store.InMemoryStore) *work.PhaseGateStore {
	t.Helper()
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	return work.NewPhaseGateStore(s, factory, testSigner{})
}

func TestPhaseGateStoreDeclareApproveReject(t *testing.T) {
	s, causes := setupStore(t)
	gates := newPhaseGateStore(t, s)

	gate, err := gates.Declare(testActor, "design", "Approve design gate", []string{"brief accepted", "risks named"}, causes, testConv)
	if err != nil {
		t.Fatalf("Declare: %v", err)
	}
	if gate.Status != work.PhaseGatePending {
		t.Fatalf("Status = %q, want pending", gate.Status)
	}
	if len(gate.Criteria) != 2 {
		t.Fatalf("Criteria = %#v, want 2 entries", gate.Criteria)
	}

	if err := gates.Approve(testActor, gate.ID, "design accepted", causes, testConv); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	got, ok, err := gates.Get(gate.ID)
	if err != nil || !ok {
		t.Fatalf("Get after approve: ok=%v err=%v", ok, err)
	}
	if got.Status != work.PhaseGateApproved || got.Summary != "design accepted" {
		t.Fatalf("approved state = %#v", got)
	}

	if err := gates.Reject(testActor, gate.ID, "missing proof", causes, testConv); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	got, ok, err = gates.Get(gate.ID)
	if err != nil || !ok {
		t.Fatalf("Get after reject: ok=%v err=%v", ok, err)
	}
	if got.Status != work.PhaseGateRejected || got.Reason != "missing proof" || got.Summary != "" {
		t.Fatalf("rejected state = %#v", got)
	}
}

func TestPhaseGateStoreRequiresPhaseAndTitle(t *testing.T) {
	s, causes := setupStore(t)
	gates := newPhaseGateStore(t, s)

	if _, err := gates.Declare(testActor, "", "title", nil, causes, testConv); err == nil {
		t.Fatal("Declare accepted empty phase")
	}
	if _, err := gates.Declare(testActor, "design", "", nil, causes, testConv); err == nil {
		t.Fatal("Declare accepted empty title")
	}
}

func TestPhaseGateTLC51LinkReplaysButNeverPromotesGenericApproval(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	pg := newPhaseGateStore(t, s)
	plan := makeTLC51TestArtifact(t, "fo_phase_tlc51", "series-1", "factory.tlc51.plan.recorded", 1, 0, nil)
	ready := makeTLC51TestArtifact(t, "fo_phase_tlc51", "series-1", "factory.tlc51.obligation.ready", 2, 1, map[string]any{
		"obligation_id": "O-DESIGN",
		"ready_at":      "2026-08-27T12:00:00Z",
	})
	task, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
		ID: "fo_phase_tlc51", Title: "phase TLC link", TLC51EventArtifacts: []work.FactoryOrderTLC51EventArtifact{plan, ready},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}
	gate, err := pg.Declare(testActor, "design", "Human design review", []string{"exact design accepted"}, causes, testConv)
	if err != nil {
		t.Fatalf("Declare: %v", err)
	}
	link := work.PhaseGateTLC51Link{
		SchemaVersion:       work.PhaseGateTLC51LinkSchemaVersion,
		GateID:              gate.ID.Value(),
		Phase:               gate.Phase,
		FactoryOrderID:      "fo_phase_tlc51",
		ChangeSeriesID:      "series-1",
		PlanDigest:          tlc51TestPlanDigest,
		SubjectDigest:       tlc51TestSubjectDigest,
		ObligationID:        "O-DESIGN",
		LinkedEventOrdinal:  ready.EventOrdinal,
		LinkedEventType:     ready.EventType,
		LinkedPayloadSHA256: ready.PayloadSHA256,
	}
	if err := ts.AttachPhaseGateTLC51Link(pg, testActor, task.ID, link, causes, testConv); err != nil {
		t.Fatalf("AttachPhaseGateTLC51Link: %v", err)
	}
	if err := pg.Approve(testActor, gate.ID, "generic Work approval only", causes, testConv); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	terminal := makeTLC51TestArtifact(t, "fo_phase_tlc51", "series-1", "factory.tlc51.obligation.terminal", 3, 1, map[string]any{
		"obligation_id": "O-DESIGN",
		"outcome":       "passed",
		"reason":        "reported by Work twin",
		"terminal_at":   "2026-08-27T12:01:00Z",
	})
	if err := ts.AttachFactoryOrderTLC51EventArtifact(testActor, task.ID, terminal, causes, testConv); err != nil {
		t.Fatalf("attach terminal event: %v", err)
	}

	// Recreate both stores to prove the join comes from event replay.
	replayedTasks := newTaskStore(t, s)
	replayedGates := newPhaseGateStore(t, s)
	projection, err := replayedTasks.ProjectPhaseGateTLC51Links(task.ID, replayedGates)
	if err != nil {
		t.Fatalf("ProjectPhaseGateTLC51Links: %v", err)
	}
	if len(projection) != 1 || projection[0].Gate == nil || projection[0].Gate.Status != work.PhaseGateApproved {
		t.Fatalf("phase-gate replay = %+v, want one approved Work gate", projection)
	}
	got := projection[0]
	if got.ReportedObligationOutcome != "passed" || got.LatestEventOrdinal != 3 {
		t.Fatalf("reported obligation replay = %+v, want terminal passed at ordinal 3", got)
	}
	if !got.EventGraphVerificationRequired || got.TLCSatisfactionCredited || got.AuthorityGranted {
		t.Fatalf("generic approval or Work twin promoted into TLC credit: %+v", got)
	}
	if got.Quarantined || got.HumanInterventionRequired {
		t.Fatalf("clean Work-local linkage quarantined: %+v", got)
	}
}

func TestPhaseGateTLC51LinkFailsClosedOnMissingTwin(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	pg := newPhaseGateStore(t, s)
	task, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{ID: "fo_phase_missing", Title: "missing twin"}, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}
	gate, err := pg.Declare(testActor, "implementation", "Implementation", nil, causes, testConv)
	if err != nil {
		t.Fatalf("Declare: %v", err)
	}
	missing := makeTLC51TestArtifact(t, "fo_phase_missing", "series-1", "factory.tlc51.obligation.ready", 1, 1, map[string]any{"obligation_id": "O-IMPL"})
	link := work.PhaseGateTLC51Link{
		SchemaVersion:       work.PhaseGateTLC51LinkSchemaVersion,
		GateID:              gate.ID.Value(),
		Phase:               gate.Phase,
		FactoryOrderID:      "fo_phase_missing",
		ChangeSeriesID:      "series-1",
		PlanDigest:          tlc51TestPlanDigest,
		SubjectDigest:       tlc51TestSubjectDigest,
		ObligationID:        "O-IMPL",
		LinkedEventOrdinal:  missing.EventOrdinal,
		LinkedEventType:     missing.EventType,
		LinkedPayloadSHA256: missing.PayloadSHA256,
	}
	if err := ts.AttachPhaseGateTLC51Link(pg, testActor, task.ID, link, causes, testConv); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing Work twin accepted: %v", err)
	}
}
