package work_test

import (
	"fmt"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// TestGetDependencies_PaginatesPastSinglePage locks in run-findings v11-F1
// follow-up (hive#153 round-2 review): GetDependencies is load-bearing for the
// reverse-edge deadlock guard, so it must fold EVERY dependency event, not just
// the newest 1000-event page ByType returns. An edge older than the newest page
// must still be visible — a bounded read under a safety guard is a fail-open.
func TestGetDependencies_PaginatesPastSinglePage(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	parent, err := ts.Create(testActor, "parent order", "aggregate", causes, testConv)
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	sub, err := ts.Create(testActor, "implementation subtask", "piece", causes, testConv)
	if err != nil {
		t.Fatalf("Create subtask: %v", err)
	}
	// The edge under test, appended FIRST so it is the oldest dependency event.
	if err := ts.AddDependency(testActor, parent.ID, sub.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency parent->sub: %v", err)
	}

	// Bury it under more than one full ByType page of newer dependency events
	// between synthetic task pairs.
	for i := 0; i < 1005; i++ {
		a := types.MustEventID(fmt.Sprintf("01900000-0000-7000-8000-%012x", i*2))
		b := types.MustEventID(fmt.Sprintf("01900000-0000-7000-8000-%012x", i*2+1))
		if err := ts.AddDependency(testActor, a, b, causes, testConv); err != nil {
			t.Fatalf("AddDependency synthetic %d: %v", i, err)
		}
	}

	deps, err := ts.GetDependencies(parent.ID)
	if err != nil {
		t.Fatalf("GetDependencies: %v", err)
	}
	if len(deps) != 1 || deps[0] != sub.ID {
		t.Fatalf("GetDependencies(parent) = %v; want exactly the buried edge to %s — the fold missed an event beyond the first page", deps, sub.ID.Value())
	}
}
