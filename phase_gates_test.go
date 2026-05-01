package work_test

import (
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
