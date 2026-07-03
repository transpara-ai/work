package work_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

func TestTC6_AC7ProjectTaskAndSummarySurfacesCanonicalPair(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	known, err := ts.CreateInWorkspace(testActor, "Known canonical task", "", "ops", causes, testConv)
	if err != nil {
		t.Fatalf("CreateInWorkspace known: %v", err)
	}
	if err := ts.TransitionTask(testActor, known.ID, work.StatusReady, "ready", nil, headCauses(t, s), testConv); err != nil {
		t.Fatalf("TransitionTask known: %v", err)
	}

	unknown, err := ts.CreateInWorkspace(testActor, "Future status task", "", "ops", headCauses(t, s), testConv)
	if err != nil {
		t.Fatalf("CreateInWorkspace unknown: %v", err)
	}
	appendRawLifecycleTransition(t, s, unknown.ID, work.StatusCreated, work.TaskStatus("paused"))

	knownProjection, err := ts.ProjectTask(known.ID)
	if err != nil {
		t.Fatalf("ProjectTask known: %v", err)
	}
	assertCanonicalSuccess(t, "ProjectTask known", knownProjection.Canonical, knownProjection.CanonicalError)

	unknownProjection, err := ts.ProjectTask(unknown.ID)
	if err != nil {
		t.Fatalf("ProjectTask unknown: %v", err)
	}
	assertCanonicalError(t, "ProjectTask unknown", unknownProjection.Canonical, unknownProjection.CanonicalError)

	cached, err := ts.ListSummariesCached(100)
	if err != nil {
		t.Fatalf("ListSummariesCached: %v", err)
	}
	batch, err := ts.ListSummaries(100)
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	workspace, err := ts.ListSummariesByWorkspace("ops", 100)
	if err != nil {
		t.Fatalf("ListSummariesByWorkspace: %v", err)
	}

	for producer, summaries := range map[string][]work.TaskSummary{
		"ListSummariesCached":      cached,
		"ListSummaries":            batch,
		"ListSummariesByWorkspace": workspace,
	} {
		knownSummary := findSummaryByID(t, producer, summaries, known.ID)
		unknownSummary := findSummaryByID(t, producer, summaries, unknown.ID)
		assertCanonicalSuccess(t, producer+" known", knownSummary.Canonical, knownSummary.CanonicalError)
		assertCanonicalError(t, producer+" unknown", unknownSummary.Canonical, unknownSummary.CanonicalError)
		if canonicalFingerprint(t, knownSummary.Canonical) != canonicalFingerprint(t, knownProjection.Canonical) {
			t.Fatalf("%s known canonical differs from ProjectTask", producer)
		}
	}
}

func TestTC6_AC7vProjectTaskLegacyReadFailurePreservesProjection(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Projection survives enrichment failure", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.TransitionTask(testActor, task.ID, work.StatusReady, "ready", nil, headCauses(t, s), testConv); err != nil {
		t.Fatalf("TransitionTask: %v", err)
	}

	baseline, err := ts.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("baseline ProjectTask: %v", err)
	}

	fault := &canonicalLegacyReadFailureStore{
		InMemoryStore: s,
		failType:      work.EventTypeTaskCompleted,
		failOnCall:    1,
	}
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	faulty := work.NewTaskStore(fault, event.NewEventFactory(registry), testSigner{})

	got, err := faulty.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask with enrichment-only failure returned error: %v", err)
	}
	assertCanonicalError(t, "ProjectTask legacy read failure", got.Canonical, got.CanonicalError)
	if !strings.Contains(got.CanonicalError, "legacy projection") {
		t.Fatalf("CanonicalError = %q, want legacy projection read failure context", got.CanonicalError)
	}

	gotComparable := got
	gotComparable.Canonical = nil
	gotComparable.CanonicalError = ""
	baselineComparable := baseline
	baselineComparable.Canonical = nil
	baselineComparable.CanonicalError = ""
	if !reflect.DeepEqual(gotComparable, baselineComparable) {
		t.Fatalf("pre-SP1 projection fields changed under enrichment failure:\ngot  %#v\nwant %#v", gotComparable, baselineComparable)
	}
}

type canonicalLegacyReadFailureStore struct {
	*store.InMemoryStore
	mu         sync.Mutex
	failType   types.EventType
	failOnCall int
	calls      map[string]int
}

func (s *canonicalLegacyReadFailureStore) ByType(eventType types.EventType, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	s.mu.Lock()
	if s.calls == nil {
		s.calls = make(map[string]int)
	}
	s.calls[eventType.Value()]++
	call := s.calls[eventType.Value()]
	fail := eventType == s.failType && call == s.failOnCall
	s.mu.Unlock()
	if fail {
		return types.Page[event.Event]{}, fmt.Errorf("injected canonical enrichment read failure for %s", eventType.Value())
	}
	return s.InMemoryStore.ByType(eventType, limit, after)
}

func appendRawLifecycleTransition(t *testing.T, s *store.InMemoryStore, taskID types.EventID, from, to work.TaskStatus) {
	t.Helper()
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(work.EventTypeTaskLifecycleTransitioned, testActor, work.TaskLifecycleTransitionContent{
		TaskID:    taskID,
		FromState: from,
		ToState:   to,
		Reason:    "future status fixture",
		ChangedBy: testActor,
	}, headCauses(t, s), testConv, s, testSigner{})
	if err != nil {
		t.Fatalf("create raw lifecycle transition: %v", err)
	}
	if _, err := s.Append(ev); err != nil {
		t.Fatalf("append raw lifecycle transition: %v", err)
	}
}

func headCauses(t *testing.T, s *store.InMemoryStore) []types.EventID {
	t.Helper()
	head, err := s.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if head.IsNone() {
		return nil
	}
	return []types.EventID{head.Unwrap().ID()}
}

func findSummaryByID(t *testing.T, producer string, summaries []work.TaskSummary, id types.EventID) work.TaskSummary {
	t.Helper()
	for _, summary := range summaries {
		if summary.Task.ID == id {
			return summary
		}
	}
	t.Fatalf("%s summary missing task %s", producer, id.Value())
	return work.TaskSummary{}
}

func assertCanonicalSuccess(t *testing.T, label string, canonical any, canonicalError string) {
	t.Helper()
	if canonical == nil {
		t.Fatalf("%s canonical is nil", label)
	}
	if canonicalError != "" {
		t.Fatalf("%s canonical_error = %q, want empty", label, canonicalError)
	}
}

func assertCanonicalError(t *testing.T, label string, canonical any, canonicalError string) {
	t.Helper()
	if !isNilCanonical(canonical) {
		t.Fatalf("%s canonical = %#v, want nil", label, canonical)
	}
	if canonicalError == "" {
		t.Fatalf("%s canonical_error is empty", label)
	}
}

func isNilCanonical(canonical any) bool {
	if canonical == nil {
		return true
	}
	value := reflect.ValueOf(canonical)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func canonicalFingerprint(t *testing.T, canonical any) string {
	t.Helper()
	out, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal canonical fingerprint: %v", err)
	}
	return string(out)
}
