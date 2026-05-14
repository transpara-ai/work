package work_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

func runtimeEnvelope(taskID types.EventID, dir string, commands ...work.RuntimeCommand) work.RuntimeEnvelope {
	return work.RuntimeEnvelope{
		TaskID:           taskID,
		Worker:           "local_deterministic",
		WorkingDirectory: dir,
		AllowedCommands:  []string{"write_file", "append_file", "copy_file", "checksum_file", "fail", "sleep", "network_attempt", "secret_attempt"},
		DeniedCommands:   []string{"denied_command"},
		AllowedFiles:     []string{"out.txt", "copy.txt", "input.txt"},
		DeniedFiles:      []string{"secret.txt"},
		NetworkPolicy:    "disabled",
		SecretsPolicy:    "none",
		TimeoutMillis:    1000,
		ResourceLimits: work.RuntimeResourceLimits{
			MaxFilesChanged: 4,
			MaxOutputBytes:  4096,
			MaxMemoryBytes:  1024 * 1024,
		},
		ExpectedOutputs: []string{"out.txt"},
		Commands:        commands,
	}
}

func createRuntimeTask(t *testing.T, ts *work.TaskStore, causes []types.EventID) work.Task {
	t.Helper()
	task, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:           "Run deterministic runtime",
		ExpectedOutputs: []string{"out.txt"},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39: %v", err)
	}
	return task
}

func countRuntimeEvents(t *testing.T, s *store.InMemoryStore) (int, int) {
	t.Helper()
	envelopes, err := s.ByType(work.EventTypeRuntimeEnvelopeRecorded, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType envelopes: %v", err)
	}
	results, err := s.ByType(work.EventTypeRuntimeResultRecorded, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType results: %v", err)
	}
	return len(envelopes.Items()), len(results.Items())
}

func hasChangedFile(files []work.RuntimeFileArtifact, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

type failLifecycleAppendStore struct {
	store.Store
	err error
}

func (s failLifecycleAppendStore) Append(ev event.Event) (event.Event, error) {
	if ev.Type() == work.EventTypeTaskLifecycleTransitioned {
		return event.Event{}, s.err
	}
	return s.Store.Append(ev)
}

func TestRuntimeBroker_RecordsEnvelopeAndResultAllowedCommandArtifactsAndCommandLog(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	run, err := ts.RunLocalRuntime(testActor, runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "hello"}},
		work.RuntimeCommand{Name: "checksum_file", Args: []string{"out.txt"}},
	), causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Envelope.ID.IsZero() {
		t.Fatal("runtime envelope event ID is zero")
	}
	if run.Result.ID.IsZero() {
		t.Fatal("runtime result event ID is zero")
	}
	if run.Result.Result.EnvelopeID != run.Envelope.ID {
		t.Fatalf("result envelope ID = %s; want %s", run.Result.Result.EnvelopeID.Value(), run.Envelope.ID.Value())
	}
	if run.Result.Result.Status != work.RuntimeStatusSucceeded {
		t.Fatalf("status = %q; want succeeded", run.Result.Result.Status)
	}
	if len(run.Result.Result.CommandLog) != 2 {
		t.Fatalf("command log len = %d; want 2", len(run.Result.Result.CommandLog))
	}
	if len(run.Result.Result.Artifacts) != 1 || run.Result.Result.Artifacts[0].Path != "out.txt" {
		t.Fatalf("artifacts = %#v; want out.txt", run.Result.Result.Artifacts)
	}
	if !hasChangedFile(run.Result.Result.ChangedFiles, "out.txt") {
		t.Fatalf("changed files = %#v; want out.txt", run.Result.Result.ChangedFiles)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "out.txt")); err != nil || string(got) != "hello" {
		t.Fatalf("out.txt = %q, %v; want hello", got, err)
	}
	envelopeCount, resultCount := countRuntimeEvents(t, s)
	if envelopeCount != 1 || resultCount != 1 {
		t.Fatalf("runtime events envelopes=%d results=%d; want 1,1", envelopeCount, resultCount)
	}
}

func TestRuntimeBroker_DeniedCommandBlockedAndHasNoSideEffects(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "denied_command", Args: []string{"out.txt", "nope"}},
	)
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.txt")); !os.IsNotExist(err) {
		t.Fatalf("out.txt side effect exists or unexpected stat err: %v", err)
	}
}

func TestRuntimeBroker_UnlistedCommandBlocked(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "not_allowlisted", Args: []string{"out.txt", "nope"}},
	)
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
}

func TestRuntimeBroker_DeniedFileAndPathTraversalBlocked(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "denied_file", path: "secret.txt"},
		{name: "traversal", path: "../escape.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task := createRuntimeTask(t, ts, causes)
			dir := t.TempDir()
			run, err := ts.RunLocalRuntime(testActor, runtimeEnvelope(task.ID, dir,
				work.RuntimeCommand{Name: "write_file", Args: []string{tc.path, "nope"}},
			), causes, testConv)
			if err != nil {
				t.Fatalf("RunLocalRuntime: %v", err)
			}
			if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
				t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
			}
			if _, err := os.Stat(filepath.Join(dir, "secret.txt")); !os.IsNotExist(err) {
				t.Fatalf("secret.txt side effect exists or unexpected stat err: %v", err)
			}
		})
	}
}

func TestRuntimeBroker_ChecksumDeniedFileBlocked(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("do not read"), 0o644); err != nil {
		t.Fatalf("seed secret.txt: %v", err)
	}
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "checksum_file", Args: []string{"secret.txt"}},
	)
	envelope.ExpectedOutputs = nil
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if len(run.Result.Result.Artifacts) != 0 {
		t.Fatalf("artifacts = %#v; want none for denied file", run.Result.Result.Artifacts)
	}
}

func TestRuntimeBroker_DirectoryAllowAndDenyPolicies(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "write_file", Args: []string{"logs/out.txt", "allowed"}},
		work.RuntimeCommand{Name: "write_file", Args: []string{"logs/private/secret.txt", "denied"}},
	)
	envelope.AllowedFiles = []string{"logs/"}
	envelope.DeniedFiles = []string{"logs/private/"}
	envelope.ExpectedOutputs = nil
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "logs", "out.txt")); err != nil || string(got) != "allowed" {
		t.Fatalf("logs/out.txt = %q, %v; want allowed", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "private", "secret.txt")); !os.IsNotExist(err) {
		t.Fatalf("logs/private/secret.txt side effect exists or unexpected stat err: %v", err)
	}
}

func TestRuntimeBroker_NetworkAndSecretsPolicyBlockSimulatedAttempts(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  string
	}{
		{name: "network", cmd: "network_attempt"},
		{name: "secrets", cmd: "secret_attempt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task := createRuntimeTask(t, ts, causes)
			dir := t.TempDir()
			envelope := runtimeEnvelope(task.ID, dir, work.RuntimeCommand{Name: tc.cmd})
			run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
			if err != nil {
				t.Fatalf("RunLocalRuntime: %v", err)
			}
			if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
				t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
			}
		})
	}
}

func TestRuntimeBroker_MaxFilesChangedBlockedBeforeSideEffect(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "first"}},
		work.RuntimeCommand{Name: "write_file", Args: []string{"copy.txt", "second"}},
	)
	envelope.ResourceLimits.MaxFilesChanged = 1
	envelope.ExpectedOutputs = nil
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "out.txt")); err != nil || string(got) != "first" {
		t.Fatalf("out.txt = %q, %v; want first", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "copy.txt")); !os.IsNotExist(err) {
		t.Fatalf("copy.txt side effect exists or unexpected stat err: %v", err)
	}
	if len(run.Result.Result.ChangedFiles) != 1 || run.Result.Result.ChangedFiles[0].Path != "out.txt" {
		t.Fatalf("changed files = %#v; want evidence for out.txt only", run.Result.Result.ChangedFiles)
	}
}

func TestRuntimeBroker_MaxOutputBytesBlockedBeforeWriteSideEffect(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "too large"}},
	)
	envelope.ResourceLimits.MaxOutputBytes = 3
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.txt")); !os.IsNotExist(err) {
		t.Fatalf("out.txt side effect exists or unexpected stat err: %v", err)
	}
}

func TestRuntimeBroker_TimeoutYieldsTimedOut(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir, work.RuntimeCommand{Name: "sleep", Args: []string{"50"}})
	envelope.TimeoutMillis = 1
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusTimedOut || !run.Result.Result.TimedOut {
		t.Fatalf("status=%q timedOut=%v; want timed_out true", run.Result.Result.Status, run.Result.Result.TimedOut)
	}
}

func TestRuntimeBroker_MissingExpectedOutputFailsPostRunValidation(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir, work.RuntimeCommand{Name: "write_file", Args: []string{"copy.txt", "not expected"}})
	envelope.ExpectedOutputs = []string{"out.txt"}
	run, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusValidationFailed {
		t.Fatalf("status = %q; want validation_failed", run.Result.Result.Status)
	}
	if len(run.Result.Result.ValidationErrors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestRuntimeBroker_ResultReplayProjectionRebuildsFromAppendOnlyEvents(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	run, err := ts.RunLocalRuntime(testActor, runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "hello"}},
	), causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	replayed := newTaskStore(t, s)
	records, err := replayed.ProjectRuntimeResults(task.ID)
	if err != nil {
		t.Fatalf("ProjectRuntimeResults: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d; want 1", len(records))
	}
	if records[0].ID != run.Result.ID || records[0].Result.Status != work.RuntimeStatusSucceeded {
		t.Fatalf("replayed record = %#v; want result ID %s succeeded", records[0], run.Result.ID.Value())
	}
}

func TestRuntimeBroker_RegisterWithRegistryIncludesRuntimeEventTypes(t *testing.T) {
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	if !registry.IsRegistered(work.EventTypeRuntimeEnvelopeRecorded) {
		t.Fatal("runtime envelope event type is not registered")
	}
	if !registry.IsRegistered(work.EventTypeRuntimeResultRecorded) {
		t.Fatal("runtime result event type is not registered")
	}
}

func TestRuntimeBroker_RegisterEventTypesUnmarshalsRuntimeResult(t *testing.T) {
	work.RegisterEventTypes()
	taskID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	envelopeID := types.MustEventID("019462a0-0000-7000-8000-000000000002")
	raw, err := json.Marshal(work.RuntimeResultRecordedContent{
		Result: work.RuntimeResult{
			EnvelopeID: envelopeID,
			TaskID:     taskID,
			Worker:     "local_deterministic",
			Status:     work.RuntimeStatusPolicyBlocked,
		},
		RecordedBy: testActor,
	})
	if err != nil {
		t.Fatalf("marshal runtime result content: %v", err)
	}
	content, err := event.UnmarshalContent("work.runtime.result.recorded", raw)
	if err != nil {
		t.Fatalf("UnmarshalContent runtime result: %v", err)
	}
	runtimeContent, ok := content.(work.RuntimeResultRecordedContent)
	if !ok {
		t.Fatalf("content type = %T; want RuntimeResultRecordedContent", content)
	}
	if runtimeContent.Result.TaskID != taskID || runtimeContent.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("runtime content = %#v; want projected task policy block", runtimeContent.Result)
	}
}

func TestRuntimeBroker_UnsupportedExternalRuntimeIsRejected(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	dir := t.TempDir()
	envelope := runtimeEnvelope(task.ID, dir, work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "hello"}})
	envelope.Worker = "hermes"
	_, err := ts.RunLocalRuntime(testActor, envelope, causes, testConv)
	if err == nil {
		t.Fatal("RunLocalRuntime accepted unsupported external runtime")
	}
	envelopeCount, resultCount := countRuntimeEvents(t, s)
	if envelopeCount != 0 || resultCount != 0 {
		t.Fatalf("runtime events envelopes=%d results=%d; want none for invalid worker", envelopeCount, resultCount)
	}
}

func TestRuntimeBroker_PolicyBlockedLifecycleTransitionErrorIsReturnedAfterEvidence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning} {
		if err := ts.TransitionTask(testActor, task.ID, state, "prepare runtime", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}

	injected := errors.New("injected lifecycle append failure")
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	failingTS := work.NewTaskStore(failLifecycleAppendStore{Store: s, err: injected}, event.NewEventFactory(registry), testSigner{})
	run, err := failingTS.RunLocalRuntime(testActor, runtimeEnvelope(task.ID, t.TempDir(),
		work.RuntimeCommand{Name: "secret_attempt"},
	), causes, testConv)
	if !errors.Is(err, injected) {
		t.Fatalf("RunLocalRuntime error = %v; want injected lifecycle append failure", err)
	}
	if !strings.Contains(err.Error(), "transition task to policy_blocked") {
		t.Fatalf("RunLocalRuntime error = %q; want lifecycle transition context", err)
	}
	if run.Envelope.ID.IsZero() || run.Result.ID.IsZero() {
		t.Fatalf("RunLocalRuntime returned run = %#v; want recorded evidence IDs despite lifecycle error", run)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("runtime status = %q; want policy_blocked", run.Result.Result.Status)
	}
	envelopeCount, resultCount := countRuntimeEvents(t, s)
	if envelopeCount != 1 || resultCount != 1 {
		t.Fatalf("runtime events envelopes=%d results=%d; want recorded evidence 1,1", envelopeCount, resultCount)
	}
	status, statusErr := ts.GetStatus(task.ID)
	if statusErr != nil {
		t.Fatalf("GetStatus after failed transition: %v", statusErr)
	}
	if status != work.StatusRunning {
		t.Fatalf("task status = %q; want running after failed lifecycle transition", status)
	}
}

func TestRuntimeBroker_PolicyBlockedDoesNotRetryAndDrivesLifecycleState(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task := createRuntimeTask(t, ts, causes)
	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning} {
		if err := ts.TransitionTask(testActor, task.ID, state, "prepare runtime", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}
	dir := t.TempDir()
	run, err := ts.RunLocalRuntime(testActor, runtimeEnvelope(task.ID, dir,
		work.RuntimeCommand{Name: "secret_attempt"},
		work.RuntimeCommand{Name: "write_file", Args: []string{"out.txt", "should not run"}},
	), causes, testConv)
	if err != nil {
		t.Fatalf("RunLocalRuntime: %v", err)
	}
	if run.Result.Result.Status != work.RuntimeStatusPolicyBlocked {
		t.Fatalf("runtime status = %q; want policy_blocked", run.Result.Result.Status)
	}
	if len(run.Result.Result.CommandLog) != 1 {
		t.Fatalf("command log len = %d; want no auto-retry after policy block", len(run.Result.Result.CommandLog))
	}
	status, err := ts.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusPolicyBlocked {
		t.Fatalf("task status = %q; want policy_blocked", status)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.txt")); !os.IsNotExist(err) {
		t.Fatalf("out.txt side effect exists or unexpected stat err: %v", err)
	}
}
