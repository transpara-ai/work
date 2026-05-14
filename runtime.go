package work

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

var (
	// EventTypeRuntimeEnvelopeRecorded records the policy and deterministic work plan for a runtime attempt.
	EventTypeRuntimeEnvelopeRecorded = types.MustEventType("work.runtime.envelope.recorded")
	// EventTypeRuntimeResultRecorded records the append-only result evidence for a runtime attempt.
	EventTypeRuntimeResultRecorded = types.MustEventType("work.runtime.result.recorded")
)

var (
	// ErrRuntimePolicyBlocked is returned when local deterministic runtime policy denies a requested action.
	ErrRuntimePolicyBlocked = errors.New("runtime policy blocked")
	// ErrRuntimeTimedOut is returned when the deterministic runtime exceeds its timeout policy.
	ErrRuntimeTimedOut = errors.New("runtime timed out")
)

// RuntimeStatus is the terminal state of a deterministic local runtime attempt.
type RuntimeStatus string

const (
	RuntimeStatusSucceeded        RuntimeStatus = "succeeded"
	RuntimeStatusFailed           RuntimeStatus = "failed"
	RuntimeStatusPolicyBlocked    RuntimeStatus = "policy_blocked"
	RuntimeStatusTimedOut         RuntimeStatus = "timed_out"
	RuntimeStatusValidationFailed RuntimeStatus = "validation_failed"
)

const localDeterministicWorker = "local_deterministic"

// RuntimeResourceLimits records practical local worker limits. The deterministic
// worker records these limits and enforces timeout directly; CPU and memory are
// represented for policy evidence rather than OS sandboxing.
type RuntimeResourceLimits struct {
	MaxFilesChanged int   `json:"MaxFilesChanged,omitempty"`
	MaxOutputBytes  int64 `json:"MaxOutputBytes,omitempty"`
	MaxMemoryBytes  int64 `json:"MaxMemoryBytes,omitempty"`
}

// RuntimeCommand is one deterministic operation. It is not a general shell command.
type RuntimeCommand struct {
	Name string   `json:"Name"`
	Args []string `json:"Args,omitempty"`
}

// RuntimeEnvelope is the append-only policy envelope for one local deterministic attempt.
type RuntimeEnvelope struct {
	TaskID           types.EventID         `json:"TaskID"`
	Worker           string                `json:"Worker"`
	WorkingDirectory string                `json:"WorkingDirectory"`
	AllowedCommands  []string              `json:"AllowedCommands,omitempty"`
	DeniedCommands   []string              `json:"DeniedCommands,omitempty"`
	AllowedFiles     []string              `json:"AllowedFiles,omitempty"`
	DeniedFiles      []string              `json:"DeniedFiles,omitempty"`
	NetworkPolicy    string                `json:"NetworkPolicy,omitempty"`
	SecretsPolicy    string                `json:"SecretsPolicy,omitempty"`
	TimeoutMillis    int                   `json:"TimeoutMillis,omitempty"`
	ResourceLimits   RuntimeResourceLimits `json:"ResourceLimits,omitempty"`
	ExpectedOutputs  []string              `json:"ExpectedOutputs,omitempty"`
	Commands         []RuntimeCommand      `json:"Commands,omitempty"`
}

// RuntimeFileArtifact records a captured file artifact or changed file.
type RuntimeFileArtifact struct {
	Path   string `json:"Path"`
	Size   int64  `json:"Size"`
	SHA256 string `json:"SHA256"`
}

// RuntimeCommandLog records one deterministic operation decision and output.
type RuntimeCommandLog struct {
	Index  int      `json:"Index"`
	Name   string   `json:"Name"`
	Args   []string `json:"Args,omitempty"`
	Status string   `json:"Status"`
	Output string   `json:"Output,omitempty"`
	Error  string   `json:"Error,omitempty"`
}

// RuntimeResult records the append-only result evidence for a runtime attempt.
type RuntimeResult struct {
	EnvelopeID       types.EventID         `json:"EnvelopeID"`
	TaskID           types.EventID         `json:"TaskID"`
	Worker           string                `json:"Worker"`
	Status           RuntimeStatus         `json:"Status"`
	ExitCode         int                   `json:"ExitCode"`
	Error            string                `json:"Error,omitempty"`
	StartedAt        time.Time             `json:"StartedAt"`
	FinishedAt       time.Time             `json:"FinishedAt"`
	TimedOut         bool                  `json:"TimedOut,omitempty"`
	PolicyBlocked    bool                  `json:"PolicyBlocked,omitempty"`
	ResourceLimits   RuntimeResourceLimits `json:"ResourceLimits,omitempty"`
	CommandLog       []RuntimeCommandLog   `json:"CommandLog,omitempty"`
	ChangedFiles     []RuntimeFileArtifact `json:"ChangedFiles,omitempty"`
	Artifacts        []RuntimeFileArtifact `json:"Artifacts,omitempty"`
	ValidationErrors []string              `json:"ValidationErrors,omitempty"`
}

// RuntimeEnvelopeRecordedContent is the event content for a RuntimeEnvelope.
type RuntimeEnvelopeRecordedContent struct {
	workContent
	Envelope   RuntimeEnvelope `json:"Envelope"`
	RecordedBy types.ActorID   `json:"RecordedBy"`
}

func (c RuntimeEnvelopeRecordedContent) EventTypeName() string {
	return "work.runtime.envelope.recorded"
}

// RuntimeResultRecordedContent is the event content for a RuntimeResult.
type RuntimeResultRecordedContent struct {
	workContent
	Result     RuntimeResult `json:"Result"`
	RecordedBy types.ActorID `json:"RecordedBy"`
}

func (c RuntimeResultRecordedContent) EventTypeName() string {
	return "work.runtime.result.recorded"
}

// RuntimeEnvelopeRecord is a replayable envelope event.
type RuntimeEnvelopeRecord struct {
	ID       types.EventID
	Envelope RuntimeEnvelope
}

// RuntimeResultRecord is a replayable result event.
type RuntimeResultRecord struct {
	ID     types.EventID
	Result RuntimeResult
}

// RuntimeRun returns the two append-only evidence events produced by RunLocalRuntime.
type RuntimeRun struct {
	Envelope RuntimeEnvelopeRecord
	Result   RuntimeResultRecord
}

// RecordRuntimeEnvelope appends the deterministic runtime policy envelope.
func (ts *TaskStore) RecordRuntimeEnvelope(source types.ActorID, envelope RuntimeEnvelope, causes []types.EventID, convID types.ConversationID) (RuntimeEnvelopeRecord, error) {
	envelope = normalizeRuntimeEnvelope(envelope)
	if err := validateRuntimeEnvelope(envelope); err != nil {
		return RuntimeEnvelopeRecord{}, err
	}
	content := RuntimeEnvelopeRecordedContent{Envelope: envelope, RecordedBy: source}
	ev, err := ts.factory.Create(EventTypeRuntimeEnvelopeRecorded, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return RuntimeEnvelopeRecord{}, fmt.Errorf("create runtime envelope event: %w", err)
	}
	stored, err := ts.store.Append(ev)
	if err != nil {
		return RuntimeEnvelopeRecord{}, fmt.Errorf("append runtime envelope event: %w", err)
	}
	return RuntimeEnvelopeRecord{ID: stored.ID(), Envelope: envelope}, nil
}

// RecordRuntimeResult appends deterministic runtime result evidence.
func (ts *TaskStore) RecordRuntimeResult(source types.ActorID, result RuntimeResult, causes []types.EventID, convID types.ConversationID) (RuntimeResultRecord, error) {
	content := RuntimeResultRecordedContent{Result: result, RecordedBy: source}
	ev, err := ts.factory.Create(EventTypeRuntimeResultRecorded, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return RuntimeResultRecord{}, fmt.Errorf("create runtime result event: %w", err)
	}
	stored, err := ts.store.Append(ev)
	if err != nil {
		return RuntimeResultRecord{}, fmt.Errorf("append runtime result event: %w", err)
	}
	return RuntimeResultRecord{ID: stored.ID(), Result: result}, nil
}

// RunLocalRuntime records a RuntimeEnvelope, executes only named deterministic local operations,
// records RuntimeResult evidence, and optionally drives policy_blocked lifecycle state from running.
func (ts *TaskStore) RunLocalRuntime(source types.ActorID, envelope RuntimeEnvelope, causes []types.EventID, convID types.ConversationID) (RuntimeRun, error) {
	envelopeRecord, err := ts.RecordRuntimeEnvelope(source, envelope, causes, convID)
	if err != nil {
		return RuntimeRun{}, err
	}
	result := executeLocalDeterministic(envelopeRecord.ID, envelopeRecord.Envelope)
	resultRecord, err := ts.RecordRuntimeResult(source, result, []types.EventID{envelopeRecord.ID}, convID)
	if err != nil {
		return RuntimeRun{}, err
	}
	if result.Status == RuntimeStatusPolicyBlocked {
		current, err := ts.GetStatus(result.TaskID)
		if err != nil {
			return RuntimeRun{Envelope: envelopeRecord, Result: resultRecord}, fmt.Errorf("get task status for runtime policy block: %w", err)
		}
		if current == StatusRunning {
			if err := ts.TransitionTask(source, result.TaskID, StatusPolicyBlocked, "runtime policy blocked", []string{resultRecord.ID.Value()}, []types.EventID{resultRecord.ID}, convID); err != nil {
				return RuntimeRun{Envelope: envelopeRecord, Result: resultRecord}, fmt.Errorf("transition task to policy_blocked after runtime result %s: %w", resultRecord.ID.Value(), err)
			}
		} else {
			// Runtime evidence is append-only; lifecycle is only driven from an active running task.
		}
	}
	return RuntimeRun{Envelope: envelopeRecord, Result: resultRecord}, nil
}

// ProjectRuntimeResults replays RuntimeResult evidence for a task from append-only events.
func (ts *TaskStore) ProjectRuntimeResults(taskID types.EventID) ([]RuntimeResultRecord, error) {
	page, err := ts.store.ByType(EventTypeRuntimeResultRecorded, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("fetch runtime result events: %w", err)
	}
	records := make([]RuntimeResultRecord, 0)
	for _, ev := range page.Items() {
		c, ok := ev.Content().(RuntimeResultRecordedContent)
		if ok && c.Result.TaskID == taskID {
			records = append(records, RuntimeResultRecord{ID: ev.ID(), Result: c.Result})
		}
	}
	return records, nil
}

func normalizeRuntimeEnvelope(envelope RuntimeEnvelope) RuntimeEnvelope {
	if envelope.Worker == "" {
		envelope.Worker = localDeterministicWorker
	}
	if envelope.NetworkPolicy == "" {
		envelope.NetworkPolicy = "disabled"
	}
	if envelope.SecretsPolicy == "" {
		envelope.SecretsPolicy = "none"
	}
	return envelope
}

func validateRuntimeEnvelope(envelope RuntimeEnvelope) error {
	if envelope.TaskID.IsZero() {
		return fmt.Errorf("runtime envelope task ID is required")
	}
	if envelope.Worker != localDeterministicWorker {
		return fmt.Errorf("unsupported runtime worker %q", envelope.Worker)
	}
	if strings.TrimSpace(envelope.WorkingDirectory) == "" {
		return fmt.Errorf("runtime envelope working directory is required")
	}
	if len(envelope.Commands) == 0 {
		return fmt.Errorf("runtime envelope requires at least one deterministic command")
	}
	return nil
}

func executeLocalDeterministic(envelopeID types.EventID, envelope RuntimeEnvelope) RuntimeResult {
	started := time.Now().UTC()
	result := RuntimeResult{
		EnvelopeID:     envelopeID,
		TaskID:         envelope.TaskID,
		Worker:         envelope.Worker,
		Status:         RuntimeStatusSucceeded,
		StartedAt:      started,
		ResourceLimits: envelope.ResourceLimits,
	}
	deadline := time.Time{}
	if envelope.TimeoutMillis > 0 {
		deadline = started.Add(time.Duration(envelope.TimeoutMillis) * time.Millisecond)
	}
	changed := map[string]RuntimeFileArtifact{}

	for i, cmd := range envelope.Commands {
		entry := RuntimeCommandLog{Index: i, Name: cmd.Name, Args: cloneStrings(cmd.Args), Status: "started"}
		if timedOut(deadline) {
			entry.Status = string(RuntimeStatusTimedOut)
			entry.Error = ErrRuntimeTimedOut.Error()
			result.CommandLog = append(result.CommandLog, entry)
			finishRuntimeResultWithChanged(&result, RuntimeStatusTimedOut, 124, ErrRuntimeTimedOut.Error(), changed)
			return result
		}
		if err := checkRuntimeCommandPolicy(envelope, cmd.Name); err != nil {
			entry.Status = string(RuntimeStatusPolicyBlocked)
			entry.Error = err.Error()
			result.CommandLog = append(result.CommandLog, entry)
			finishRuntimeResultWithChanged(&result, RuntimeStatusPolicyBlocked, 126, err.Error(), changed)
			return result
		}
		if path, ok, err := runtimeCommandChangedFile(envelope, cmd); err != nil {
			entry.Status = string(RuntimeStatusPolicyBlocked)
			entry.Error = err.Error()
			result.CommandLog = append(result.CommandLog, entry)
			finishRuntimeResultWithChanged(&result, RuntimeStatusPolicyBlocked, 126, err.Error(), changed)
			return result
		} else if ok && envelope.ResourceLimits.MaxFilesChanged > 0 {
			if _, alreadyChanged := changed[path]; !alreadyChanged && len(changed) >= envelope.ResourceLimits.MaxFilesChanged {
				err := fmt.Errorf("%w: max files changed limit %d exceeded", ErrRuntimePolicyBlocked, envelope.ResourceLimits.MaxFilesChanged)
				entry.Status = string(RuntimeStatusPolicyBlocked)
				entry.Error = err.Error()
				result.CommandLog = append(result.CommandLog, entry)
				finishRuntimeResultWithChanged(&result, RuntimeStatusPolicyBlocked, 126, err.Error(), changed)
				return result
			}
		}
		output, files, err := executeRuntimeCommand(envelope, cmd, deadline)
		entry.Output = output
		if err != nil {
			entry.Error = err.Error()
			if errors.Is(err, ErrRuntimePolicyBlocked) {
				entry.Status = string(RuntimeStatusPolicyBlocked)
				result.CommandLog = append(result.CommandLog, entry)
				finishRuntimeResultWithChanged(&result, RuntimeStatusPolicyBlocked, 126, err.Error(), changed)
				return result
			}
			if errors.Is(err, ErrRuntimeTimedOut) {
				entry.Status = string(RuntimeStatusTimedOut)
				result.CommandLog = append(result.CommandLog, entry)
				finishRuntimeResultWithChanged(&result, RuntimeStatusTimedOut, 124, err.Error(), changed)
				return result
			}
			entry.Status = string(RuntimeStatusFailed)
			result.CommandLog = append(result.CommandLog, entry)
			finishRuntimeResultWithChanged(&result, RuntimeStatusFailed, 1, err.Error(), changed)
			return result
		}
		if err := checkRuntimeOutputLimit(envelope, output); err != nil {
			entry.Status = string(RuntimeStatusPolicyBlocked)
			entry.Error = err.Error()
			result.CommandLog = append(result.CommandLog, entry)
			finishRuntimeResultWithChanged(&result, RuntimeStatusPolicyBlocked, 126, err.Error(), changed)
			return result
		}
		entry.Status = string(RuntimeStatusSucceeded)
		result.CommandLog = append(result.CommandLog, entry)
		for _, file := range files {
			changed[file.Path] = file
		}
	}

	for _, path := range envelope.ExpectedOutputs {
		artifact, err := captureRuntimeArtifact(envelope, path)
		if err != nil {
			result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("expected output %q missing: %v", path, err))
			continue
		}
		result.Artifacts = append(result.Artifacts, artifact)
	}
	for _, path := range sortedRuntimeArtifactPaths(changed) {
		result.ChangedFiles = append(result.ChangedFiles, changed[path])
	}
	if len(result.ValidationErrors) > 0 {
		finishRuntimeResult(&result, RuntimeStatusValidationFailed, 2, strings.Join(result.ValidationErrors, "; "))
		return result
	}
	finishRuntimeResult(&result, RuntimeStatusSucceeded, 0, "")
	return result
}

func finishRuntimeResult(result *RuntimeResult, status RuntimeStatus, exitCode int, message string) {
	result.Status = status
	result.ExitCode = exitCode
	result.Error = message
	result.FinishedAt = time.Now().UTC()
	result.TimedOut = status == RuntimeStatusTimedOut
	result.PolicyBlocked = status == RuntimeStatusPolicyBlocked
}

func finishRuntimeResultWithChanged(result *RuntimeResult, status RuntimeStatus, exitCode int, message string, changed map[string]RuntimeFileArtifact) {
	appendChangedRuntimeFiles(result, changed)
	finishRuntimeResult(result, status, exitCode, message)
}

func executeRuntimeCommand(envelope RuntimeEnvelope, cmd RuntimeCommand, deadline time.Time) (string, []RuntimeFileArtifact, error) {
	switch cmd.Name {
	case "write_file":
		if len(cmd.Args) != 2 {
			return "", nil, fmt.Errorf("write_file requires path and content")
		}
		path, err := resolveRuntimePath(envelope, cmd.Args[0])
		if err != nil {
			return "", nil, err
		}
		if err := checkRuntimeBytesLimit(envelope, int64(len(cmd.Args[1]))); err != nil {
			return "", nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(path, []byte(cmd.Args[1]), 0o644); err != nil {
			return "", nil, err
		}
		artifact, err := captureRuntimeArtifact(envelope, cmd.Args[0])
		return "", []RuntimeFileArtifact{artifact}, err
	case "append_file":
		if len(cmd.Args) != 2 {
			return "", nil, fmt.Errorf("append_file requires path and content")
		}
		path, err := resolveRuntimePath(envelope, cmd.Args[0])
		if err != nil {
			return "", nil, err
		}
		currentSize, err := runtimeFileSize(path)
		if err != nil {
			return "", nil, err
		}
		if err := checkRuntimeBytesLimit(envelope, currentSize+int64(len(cmd.Args[1]))); err != nil {
			return "", nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", nil, err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return "", nil, err
		}
		if _, err := f.WriteString(cmd.Args[1]); err != nil {
			_ = f.Close()
			return "", nil, err
		}
		if err := f.Close(); err != nil {
			return "", nil, err
		}
		artifact, err := captureRuntimeArtifact(envelope, cmd.Args[0])
		return "", []RuntimeFileArtifact{artifact}, err
	case "copy_file":
		if len(cmd.Args) != 2 {
			return "", nil, fmt.Errorf("copy_file requires source and destination")
		}
		src, err := resolveRuntimePath(envelope, cmd.Args[0])
		if err != nil {
			return "", nil, err
		}
		dst, err := resolveRuntimePath(envelope, cmd.Args[1])
		if err != nil {
			return "", nil, err
		}
		srcSize, err := runtimeFileSize(src)
		if err != nil {
			return "", nil, err
		}
		if err := checkRuntimeBytesLimit(envelope, srcSize); err != nil {
			return "", nil, err
		}
		in, err := os.Open(src)
		if err != nil {
			return "", nil, err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", nil, err
		}
		out, err := os.Create(dst)
		if err != nil {
			return "", nil, err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			return "", nil, err
		}
		if err := out.Close(); err != nil {
			return "", nil, err
		}
		artifact, err := captureRuntimeArtifact(envelope, cmd.Args[1])
		return "", []RuntimeFileArtifact{artifact}, err
	case "checksum_file":
		if len(cmd.Args) != 1 {
			return "", nil, fmt.Errorf("checksum_file requires path")
		}
		artifact, err := captureRuntimeArtifact(envelope, cmd.Args[0])
		if err != nil {
			return "", nil, err
		}
		return artifact.SHA256, nil, nil
	case "fail":
		msg := "deterministic failure"
		if len(cmd.Args) > 0 {
			msg = cmd.Args[0]
		}
		return "", nil, fmt.Errorf("%s", msg)
	case "sleep":
		if len(cmd.Args) != 1 {
			return "", nil, fmt.Errorf("sleep requires milliseconds")
		}
		millis, err := strconv.Atoi(cmd.Args[0])
		if err != nil || millis < 0 {
			return "", nil, fmt.Errorf("invalid sleep milliseconds %q", cmd.Args[0])
		}
		duration := time.Duration(millis) * time.Millisecond
		if !deadline.IsZero() && time.Now().UTC().Add(duration).After(deadline) {
			return "", nil, ErrRuntimeTimedOut
		}
		time.Sleep(duration)
		return "", nil, nil
	case "network_attempt":
		if envelope.NetworkPolicy != "enabled" && envelope.NetworkPolicy != "allow" {
			return "", nil, fmt.Errorf("%w: network policy is %q", ErrRuntimePolicyBlocked, envelope.NetworkPolicy)
		}
		return "network attempt simulated", nil, nil
	case "secret_attempt":
		if envelope.SecretsPolicy == "none" || envelope.SecretsPolicy == "" {
			return "", nil, fmt.Errorf("%w: secrets policy is none", ErrRuntimePolicyBlocked)
		}
		return "secret attempt simulated", nil, nil
	default:
		return "", nil, fmt.Errorf("%w: unsupported deterministic operation %q", ErrRuntimePolicyBlocked, cmd.Name)
	}
}

func checkRuntimeCommandPolicy(envelope RuntimeEnvelope, name string) error {
	if stringIn(name, envelope.DeniedCommands) {
		return fmt.Errorf("%w: command %q is denied", ErrRuntimePolicyBlocked, name)
	}
	if !stringIn(name, envelope.AllowedCommands) {
		return fmt.Errorf("%w: command %q is not allowed", ErrRuntimePolicyBlocked, name)
	}
	return nil
}

func runtimeCommandChangedFile(envelope RuntimeEnvelope, cmd RuntimeCommand) (string, bool, error) {
	var rel string
	switch cmd.Name {
	case "write_file", "append_file":
		if len(cmd.Args) < 1 {
			return "", false, nil
		}
		rel = cmd.Args[0]
	case "copy_file":
		if len(cmd.Args) < 2 {
			return "", false, nil
		}
		rel = cmd.Args[1]
	default:
		return "", false, nil
	}
	if _, err := resolveRuntimePath(envelope, rel); err != nil {
		return "", false, err
	}
	clean, err := cleanRuntimeRelativePath(rel)
	if err != nil {
		return "", false, err
	}
	return clean, true, nil
}

func resolveRuntimePath(envelope RuntimeEnvelope, rel string) (string, error) {
	clean, err := cleanRuntimeRelativePath(rel)
	if err != nil {
		return "", err
	}
	if !pathAllowed(clean, envelope.AllowedFiles) {
		return "", fmt.Errorf("%w: file %q is not allowed", ErrRuntimePolicyBlocked, clean)
	}
	if pathDenied(clean, envelope.DeniedFiles) {
		return "", fmt.Errorf("%w: file %q is denied", ErrRuntimePolicyBlocked, clean)
	}
	root, err := filepath.Abs(envelope.WorkingDirectory)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(clean))
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	relToRoot, err := filepath.Rel(root, fullAbs)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path traversal outside working directory", ErrRuntimePolicyBlocked)
	}
	return fullAbs, nil
}

func cleanRuntimeRelativePath(path string) (string, error) {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" || path == "." {
		return "", fmt.Errorf("%w: empty runtime path", ErrRuntimePolicyBlocked)
	}
	if strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: absolute runtime path %q", ErrRuntimePolicyBlocked, path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: path traversal outside working directory", ErrRuntimePolicyBlocked)
	}
	return clean, nil
}

func captureRuntimeArtifact(envelope RuntimeEnvelope, rel string) (RuntimeFileArtifact, error) {
	full, err := resolveRuntimePath(envelope, rel)
	if err != nil {
		return RuntimeFileArtifact{}, err
	}
	clean, err := cleanRuntimeRelativePath(rel)
	if err != nil {
		return RuntimeFileArtifact{}, err
	}
	f, err := os.Open(full)
	if err != nil {
		return RuntimeFileArtifact{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return RuntimeFileArtifact{}, err
	}
	info, err := f.Stat()
	if err != nil {
		return RuntimeFileArtifact{}, err
	}
	return RuntimeFileArtifact{Path: clean, Size: info.Size(), SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

func runtimeFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

func checkRuntimeBytesLimit(envelope RuntimeEnvelope, size int64) error {
	if envelope.ResourceLimits.MaxOutputBytes <= 0 || size <= envelope.ResourceLimits.MaxOutputBytes {
		return nil
	}
	return fmt.Errorf("%w: max output bytes limit %d exceeded", ErrRuntimePolicyBlocked, envelope.ResourceLimits.MaxOutputBytes)
}

func checkRuntimeOutputLimit(envelope RuntimeEnvelope, output string) error {
	return checkRuntimeBytesLimit(envelope, int64(len(output)))
}

func sortedRuntimeArtifactPaths(artifacts map[string]RuntimeFileArtifact) []string {
	paths := make([]string, 0, len(artifacts))
	for path := range artifacts {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func appendChangedRuntimeFiles(result *RuntimeResult, changed map[string]RuntimeFileArtifact) {
	for _, path := range sortedRuntimeArtifactPaths(changed) {
		result.ChangedFiles = append(result.ChangedFiles, changed[path])
	}
}

func pathAllowed(path string, allowed []string) bool {
	for _, candidate := range allowed {
		candidate = cleanRuntimePolicyPath(candidate)
		if candidate == path || strings.HasSuffix(candidate, "/") && strings.HasPrefix(path, candidate) {
			return true
		}
	}
	return false
}

func pathDenied(path string, denied []string) bool {
	for _, candidate := range denied {
		candidate = cleanRuntimePolicyPath(candidate)
		if candidate == path || strings.HasSuffix(candidate, "/") && strings.HasPrefix(path, candidate) {
			return true
		}
	}
	return false
}

func cleanRuntimePolicyPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if strings.HasSuffix(path, "/") {
		return filepath.ToSlash(filepath.Clean(strings.TrimSuffix(path, "/"))) + "/"
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func stringIn(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func timedOut(deadline time.Time) bool {
	return !deadline.IsZero() && time.Now().UTC().After(deadline)
}
