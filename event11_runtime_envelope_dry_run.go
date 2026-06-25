package work

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Event11RuntimeEnvelopeDryRunMode = "event11_runtime_envelope_dry_run"

	event11ActorID          = "act_df_v40_event11_runtime_envelope_dry_run"
	event11ExternalActorID  = "act_michael_saucier_external_committee"
	event11AuthorityDoc     = "DF-V4.0-EPIC-011-AUTHORITY-DECISION"
	event11DocsPR           = "transpara-ai/docs#180"
	event11DocsMergeSHA     = "e6fc5e65305e4ef17b110c1952fc4ce91bf938ff"
	event11DocsReviewedHead = "83bd5ae1cf183ff31ef67e265ba658c8617d8679"
	event11CorrelationID    = "corr_df_v40_event11_runtime_envelope_dry_run"
	event11FixtureTimeRFC   = "2026-06-21T12:00:00Z"
	event11FixtureWorkDir   = "fixture://work/event11-runtime-envelope-dry-run"
)

// Event11RuntimeEnvelopeDryRunOptions configures the bounded Gate U fixture.
// Negative-test seams intentionally omit or widen evidence so tests can prove
// the closeout fails closed instead of weakening Gate U.
type Event11RuntimeEnvelopeDryRunOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	// WorkingDir must be an ephemeral fixture directory owned by the caller.
	WorkingDir string

	OmitAuthorityReceipt bool
	OmitRuntimeResult    bool
	OmitCodeChange       bool
	OmitAuditReport      bool
	OmitPolicyCases      bool
	OmitEnvelopeHash     bool

	UnsafeNetworkPolicy string
	UnsafeSecretsPolicy string
}

// Event11RuntimeEnvelopeDryRunRun is the Work-side evidence packet for one
// local deterministic RuntimeBroker dry run under Event 11 authority.
type Event11RuntimeEnvelopeDryRunRun struct {
	Mode                  string
	WorkTask              Task
	WorkProjection        TaskProjection
	RuntimeRun            RuntimeRun
	PolicyCases           []Event11PolicyCaseResult
	EventGraph            *v39.InMemoryStore
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	ActorIdentityID       string
	ActorInvocationID     string
	AuthorityRequestID    string
	AuthorityDecisionID   string
	ExecutionReceiptID    string
	RuntimeEnvelopeID     string
	RuntimeResultID       string
	ArtifactID            string
	CodeChangeID          string
	TestCaseID            string
	TestRunID             string
	GateResultID          string
	FailureID             string
	FactoryRuntimeID      string
	ReleaseCandidateID    string
	CertificationID       string
	RejectionID           string
	AuditReportID         string
	TraceCompleteness     v39.TraceCompletenessGateResult
	AuthorityPath         v39.RequiredPath
	Certification         *v39.Certification
	Rejection             *v39.Rejection
	AuditReport           *v39.AuditReport
	Report                Event11RuntimeEnvelopeDryRunReport
}

type Event11RuntimeEnvelopeDryRunReport struct {
	Status                 string                      `json:"status"`
	Missing                []string                    `json:"missing,omitempty"`
	TypeCounts             map[string]int              `json:"type_counts"`
	TraceCompleted         bool                        `json:"trace_completed"`
	TraceStatus            v39.TraceCompletenessStatus `json:"trace_status"`
	AuthorityPathCompleted bool                        `json:"authority_path_completed"`
	EnvelopeHash           string                      `json:"envelope_hash,omitempty"`
	EnvelopeImmutable      bool                        `json:"envelope_immutable"`
	LocalRuntimeOnly       bool                        `json:"local_runtime_only"`
	TraceOutput            Event11RuntimeTraceOutput   `json:"trace_output"`
	GateOutput             Event11RuntimeGateOutput    `json:"gate_output"`
	EventGraphHandoff      Event11EventGraphHandoff    `json:"eventgraph_handoff"`
	PolicyCases            []Event11PolicyCaseResult   `json:"policy_cases"`
	ForbiddenActions       []Event11ForbiddenAction    `json:"forbidden_actions"`
	ResidualRisks          []Event11ResidualRiskState  `json:"residual_risks"`
	EvidenceRefs           []string                    `json:"evidence_refs"`
}

type Event11RuntimeTraceOutput struct {
	Status             string                      `json:"status"`
	TraceStatus        v39.TraceCompletenessStatus `json:"trace_status"`
	TraceCompleted     bool                        `json:"trace_completed"`
	FactoryOrderID     string                      `json:"factory_order_id"`
	ReleaseCandidateID string                      `json:"release_candidate_id"`
	TestRunID          string                      `json:"test_run_id"`
	GateResultID       string                      `json:"gate_result_id"`
	AuditReportID      string                      `json:"audit_report_id,omitempty"`
	RequiredPathCount  int                         `json:"required_path_count"`
	Missing            []string                    `json:"missing,omitempty"`
	EvidenceRefs       []string                    `json:"evidence_refs"`
}

type Event11RuntimeGateOutput struct {
	Status              string   `json:"status"`
	GateName            string   `json:"gate_name"`
	GateScope           string   `json:"gate_scope"`
	GateUClosureClaimed bool     `json:"gate_u_closure_claimed"`
	FactoryOrderID      string   `json:"factory_order_id"`
	ReleaseCandidateID  string   `json:"release_candidate_id"`
	TestRunID           string   `json:"test_run_id"`
	GateResultID        string   `json:"gate_result_id"`
	AuditReportID       string   `json:"audit_report_id,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs"`
	Missing             []string `json:"missing,omitempty"`
}

type Event11EventGraphHandoff struct {
	Status                 string   `json:"status"`
	ProjectionScope        string   `json:"projection_scope"`
	PersistentWriteStatus  string   `json:"persistent_write_status"`
	PersistentWriteClaimed bool     `json:"persistent_write_claimed"`
	ProductionTruthClaimed bool     `json:"production_truth_claimed"`
	RuntimeExecutionScope  string   `json:"runtime_execution_scope"`
	EventGraphRefs         []string `json:"eventgraph_refs"`
	AuthorityRefs          []string `json:"authority_refs"`
	BlockedBy              []string `json:"blocked_by,omitempty"`
	Notes                  []string `json:"notes"`
}

type Event11PolicyCaseResult struct {
	Name            string        `json:"name"`
	Status          RuntimeStatus `json:"status"`
	ExitCode        int           `json:"exit_code"`
	PolicyBlocked   bool          `json:"policy_blocked,omitempty"`
	TimedOut        bool          `json:"timed_out,omitempty"`
	ValidationError bool          `json:"validation_error,omitempty"`
	SideEffectFree  bool          `json:"side_effect_free"`
	Error           string        `json:"error,omitempty"`
	ChangedFiles    []string      `json:"changed_files,omitempty"`
	CommandLog      []string      `json:"command_log,omitempty"`
}

type Event11ForbiddenAction struct {
	Action string `json:"action"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Event11ResidualRiskState struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type event11FixtureIDs struct {
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	actorIdentity       string
	actorInvocation     string
	authorityRequest    string
	authorityDecision   string
	executionReceipt    string
	runtimeEnvelope     string
	runtimeResult       string
	artifact            string
	codeChange          string
	testCase            string
	testRun             string
	gateResult          string
	failure             string
	factoryRuntime      string
	releaseCandidate    string
	certification       string
	rejection           string
	auditReport         string
}

type event11GraphRun struct {
	trace         v39.TraceCompletenessGateResult
	authorityPath v39.RequiredPath
	traceErr      error
	authorityErr  error
	envelopeHash  string
	certification *v39.Certification
	rejection     *v39.Rejection
	auditReport   *v39.AuditReport
}

// RunEvent11RuntimeEnvelopeDryRunFixture runs one local deterministic
// RuntimeBroker fixture and projects the evidence into a local in-memory v3.9
// graph. It does not call external runtimes, network, secrets, production
// stores, GitHub mutation APIs, deploy tooling, Hive, Site, Agent, or protected
// settings.
func RunEvent11RuntimeEnvelopeDryRunFixture(ts *TaskStore, opts Event11RuntimeEnvelopeDryRunOptions) (Event11RuntimeEnvelopeDryRunRun, error) {
	if ts == nil {
		return Event11RuntimeEnvelopeDryRunRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Event11RuntimeEnvelopeDryRunRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Event11RuntimeEnvelopeDryRunRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Event11RuntimeEnvelopeDryRunRun{}, errors.New("working directory is required")
	}
	workingDir, err := filepath.Abs(opts.WorkingDir)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}

	ids := event11IDs()
	task, err := SeedFactoryOrder(ts, opts.Source, event11FactoryOrder(ids), opts.Causes, opts.ConversationID)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusReady, "Event 11 FactoryOrder readiness gates recorded", []string{ids.factoryOrder, ids.requirement, ids.acceptanceCriterion}, opts.Causes, opts.ConversationID); err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusRunning, "Event 11 local deterministic RuntimeBroker dry run started", []string{ids.authorityDecision}, opts.Causes, opts.ConversationID); err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}

	envelope := event11RuntimeEnvelope(task.ID, filepath.Join(workingDir, "happy-path"))
	canonicalEnvelope := event11CanonicalRuntimeEnvelope(envelope)
	envelopeHash, err := event11EnvelopeHash(canonicalEnvelope)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	beforeHash := envelopeHash
	runtimeRun, err := ts.RunLocalRuntime(opts.Source, envelope, opts.Causes, opts.ConversationID)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	afterHash, err := event11EnvelopeHash(event11CanonicalRuntimeEnvelope(envelope))
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	envelopeImmutable := beforeHash == afterHash
	if !envelopeImmutable {
		return Event11RuntimeEnvelopeDryRunRun{}, errors.New("runtime envelope mutated during dry run")
	}

	policyCases, err := event11RunPolicyCases(ts, opts, workingDir)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	artifactHash, err := event11RuntimeArtifactHash(runtimeRun.Result.Result)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}

	graph, graphRun, err := event11RecordEventGraph(ids, runtimeRun, canonicalEnvelope, envelopeImmutable, artifactHash, policyCases, opts)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	report := event11Evaluate(graph, ids, graphRun.trace, graphRun.authorityPath, graphRun.traceErr, graphRun.authorityErr, graphRun.envelopeHash, envelopeImmutable, policyCases, opts)
	if err := ts.AddArtifact(opts.Source, task.ID, "event11_runtime_evidence", "application/json", event11WorkArtifactBody(report), append(append([]types.EventID(nil), opts.Causes...), runtimeRun.Result.ID), opts.ConversationID); err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Event 11 RuntimeBroker dry-run evidence attached", opts.Causes, opts.ConversationID); err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}
	if report.Status == "pass" {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Event 11 dry-run evidence verified", []string{ids.testRun, ids.gateResult, runtimeRun.Result.ID.Value()}, opts.Causes, opts.ConversationID); err != nil {
			return Event11RuntimeEnvelopeDryRunRun{}, err
		}
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Event 11 local deterministic dry-run fixture certified", []string{ids.certification}, opts.Causes, opts.ConversationID); err != nil {
			return Event11RuntimeEnvelopeDryRunRun{}, err
		}
	} else {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Event 11 dry-run evidence evaluated with residuals", []string{ids.testRun, ids.gateResult}, opts.Causes, opts.ConversationID); err != nil {
			return Event11RuntimeEnvelopeDryRunRun{}, err
		}
		if err := ts.RejectTask(opts.Source, task.ID, "Event 11 dry-run fixture failed closed", append([]string{ids.gateResult}, report.Missing...), opts.Causes, opts.ConversationID); err != nil {
			return Event11RuntimeEnvelopeDryRunRun{}, err
		}
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Event11RuntimeEnvelopeDryRunRun{}, err
	}

	return Event11RuntimeEnvelopeDryRunRun{
		Mode:                  Event11RuntimeEnvelopeDryRunMode,
		WorkTask:              task,
		WorkProjection:        workProjection,
		RuntimeRun:            runtimeRun,
		PolicyCases:           policyCases,
		EventGraph:            graph,
		FactoryOrderID:        ids.factoryOrder,
		RequirementID:         ids.requirement,
		AcceptanceCriterionID: ids.acceptanceCriterion,
		TaskID:                ids.task,
		ActorIdentityID:       ids.actorIdentity,
		ActorInvocationID:     ids.actorInvocation,
		AuthorityRequestID:    ids.authorityRequest,
		AuthorityDecisionID:   ids.authorityDecision,
		ExecutionReceiptID:    ids.executionReceipt,
		RuntimeEnvelopeID:     ids.runtimeEnvelope,
		RuntimeResultID:       ids.runtimeResult,
		ArtifactID:            ids.artifact,
		CodeChangeID:          ids.codeChange,
		TestCaseID:            ids.testCase,
		TestRunID:             ids.testRun,
		GateResultID:          ids.gateResult,
		FailureID:             ids.failure,
		FactoryRuntimeID:      ids.factoryRuntime,
		ReleaseCandidateID:    ids.releaseCandidate,
		CertificationID:       ids.certification,
		RejectionID:           ids.rejection,
		AuditReportID:         ids.auditReport,
		TraceCompleteness:     graphRun.trace,
		AuthorityPath:         graphRun.authorityPath,
		Certification:         graphRun.certification,
		Rejection:             graphRun.rejection,
		AuditReport:           graphRun.auditReport,
		Report:                report,
	}, nil
}

func event11FactoryOrder(ids event11FixtureIDs) FactoryOrder {
	return FactoryOrder{
		Kind:                   OrderSoftwarePR,
		ID:                     ids.factoryOrder,
		Title:                  "Event 11 Governed Runtime Envelope Dry Run",
		Intent:                 "Prove one local deterministic RuntimeBroker envelope/result path under Event 11 authority.",
		Cell:                   "cell_df_v40_event11_runtime_envelope_dry_run",
		RiskClass:              "high",
		DefinitionOfDone:       "RuntimeEnvelope is recorded before execution, RuntimeResult is append-only, policy-blocked negative cases prove no unauthorized side effects, and Gate U remains open pending docs evidence decision.",
		AcceptanceCriteria:     "Evidence includes FactoryOrder, Task, AuthorityDecision, immutable RuntimeEnvelope, RuntimeResult, Artifact/CodeChange, TestRun, GateResult, TraceCompletenessGate, Certification or Rejection, AuditReport-shaped closeout, and negative policy cases.",
		TestPlan:               "Run event11_runtime_envelope_dry_run_test.go plus go test ./..., go vet ./..., make verify, exact-head review, and standalone External Committee approval before merge.",
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		ExpectedOutputs:        []string{"report.txt"},
	}
}

func event11RuntimeEnvelope(taskID types.EventID, dir string) RuntimeEnvelope {
	return RuntimeEnvelope{
		TaskID:           taskID,
		Worker:           localDeterministicWorker,
		WorkingDirectory: dir,
		AllowedCommands:  []string{"write_file", "append_file", "checksum_file"},
		DeniedCommands:   event11DeniedCommands(),
		AllowedFiles:     []string{"report.txt"},
		DeniedFiles:      []string{"secret.txt", ".env", ".git/", "../", "go.mod", "go.sum", "production/**"},
		NetworkPolicy:    "disabled",
		SecretsPolicy:    "none",
		TimeoutMillis:    1000,
		ResourceLimits: RuntimeResourceLimits{
			MaxFilesChanged: 1,
			MaxOutputBytes:  4096,
			MaxMemoryBytes:  1024 * 1024,
		},
		ExpectedOutputs: []string{"report.txt"},
		Commands: []RuntimeCommand{
			{Name: "write_file", Args: []string{"report.txt", "Event 11 local deterministic RuntimeBroker dry-run evidence\n"}},
			{Name: "checksum_file", Args: []string{"report.txt"}},
		},
	}
}

func event11CanonicalRuntimeEnvelope(envelope RuntimeEnvelope) RuntimeEnvelope {
	canonical := envelope
	canonical.WorkingDirectory = event11FixtureWorkDir
	canonical.AllowedCommands = cloneStrings(envelope.AllowedCommands)
	canonical.DeniedCommands = cloneStrings(envelope.DeniedCommands)
	canonical.AllowedFiles = cloneStrings(envelope.AllowedFiles)
	canonical.DeniedFiles = cloneStrings(envelope.DeniedFiles)
	canonical.ExpectedOutputs = cloneStrings(envelope.ExpectedOutputs)
	canonical.Commands = append([]RuntimeCommand(nil), envelope.Commands...)
	for i := range canonical.Commands {
		canonical.Commands[i].Args = cloneStrings(canonical.Commands[i].Args)
	}
	return canonical
}

func event11RunPolicyCases(ts *TaskStore, opts Event11RuntimeEnvelopeDryRunOptions, workingDir string) ([]Event11PolicyCaseResult, error) {
	if opts.OmitPolicyCases {
		return nil, nil
	}
	cases := []struct {
		name    string
		envelop func(types.EventID, string) RuntimeEnvelope
		check   func(string, RuntimeResult) bool
	}{
		{
			name: "denied_command",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "denied_command", Args: []string{"out.txt", "blocked"}})
				env.ExpectedOutputs = nil
				return env
			},
			check: func(dir string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusPolicyBlocked && !fileExists(filepath.Join(dir, "out.txt"))
			},
		},
		{
			name: "path_traversal",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "write_file", Args: []string{"../escape.txt", "blocked"}})
				env.ExpectedOutputs = nil
				return env
			},
			check: func(dir string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusPolicyBlocked && !fileExists(filepath.Join(filepath.Dir(dir), "escape.txt"))
			},
		},
		{
			name: "network_attempt",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "network_attempt"})
				env.ExpectedOutputs = nil
				return env
			},
			check: func(_ string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusPolicyBlocked && len(result.ChangedFiles) == 0
			},
		},
		{
			name: "secret_attempt",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "secret_attempt"})
				env.ExpectedOutputs = nil
				return env
			},
			check: func(_ string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusPolicyBlocked && len(result.ChangedFiles) == 0
			},
		},
		{
			name: "timeout",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "sleep", Args: []string{"50"}})
				env.TimeoutMillis = 1
				env.ExpectedOutputs = nil
				return env
			},
			check: func(_ string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusTimedOut && result.TimedOut && len(result.ChangedFiles) == 0
			},
		},
		{
			name: "validation_failure",
			envelop: func(taskID types.EventID, dir string) RuntimeEnvelope {
				env := event11PolicyEnvelope(taskID, dir, RuntimeCommand{Name: "write_file", Args: []string{"other.txt", "not expected"}})
				env.AllowedFiles = []string{"other.txt", "report.txt"}
				env.ExpectedOutputs = []string{"report.txt"}
				return env
			},
			check: func(dir string, result RuntimeResult) bool {
				return result.Status == RuntimeStatusValidationFailed && fileExists(filepath.Join(dir, "other.txt")) && !fileExists(filepath.Join(dir, "report.txt"))
			},
		},
	}
	results := make([]Event11PolicyCaseResult, 0, len(cases))
	for _, tc := range cases {
		task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
			Title:                  "Event 11 policy case: " + tc.name,
			FactoryOrderID:         "fo_df_v40_event11_policy_" + tc.name,
			RequirementIDs:         []string{"req_df_v40_event11_policy_" + tc.name},
			AcceptanceCriterionIDs: []string{"ac_df_v40_event11_policy_" + tc.name},
			Cell:                   "cell_df_v40_event11_runtime_envelope_dry_run",
			RiskClass:              "high",
		}, opts.Causes, opts.ConversationID)
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(workingDir, "policy-"+tc.name)
		env := tc.envelop(task.ID, dir)
		run, err := ts.RunLocalRuntime(opts.Source, env, opts.Causes, opts.ConversationID)
		if err != nil {
			return nil, err
		}
		result := run.Result.Result
		results = append(results, Event11PolicyCaseResult{
			Name:            tc.name,
			Status:          result.Status,
			ExitCode:        result.ExitCode,
			PolicyBlocked:   result.PolicyBlocked,
			TimedOut:        result.TimedOut,
			ValidationError: result.Status == RuntimeStatusValidationFailed,
			SideEffectFree:  tc.check(dir, result),
			Error:           result.Error,
			ChangedFiles:    event11ChangedFilePaths(result.ChangedFiles),
			CommandLog:      event11CommandLog(result.CommandLog),
		})
	}
	return results, nil
}

func event11PolicyEnvelope(taskID types.EventID, dir string, command RuntimeCommand) RuntimeEnvelope {
	return RuntimeEnvelope{
		TaskID:           taskID,
		Worker:           localDeterministicWorker,
		WorkingDirectory: dir,
		AllowedCommands:  []string{"write_file", "sleep", "network_attempt", "secret_attempt"},
		DeniedCommands:   event11DeniedCommands(),
		AllowedFiles:     []string{"out.txt"},
		DeniedFiles:      []string{"secret.txt", ".env", ".git/", "../"},
		NetworkPolicy:    "disabled",
		SecretsPolicy:    "none",
		TimeoutMillis:    1000,
		ResourceLimits: RuntimeResourceLimits{
			MaxFilesChanged: 1,
			MaxOutputBytes:  4096,
			MaxMemoryBytes:  1024 * 1024,
		},
		ExpectedOutputs: []string{"out.txt"},
		Commands:        []RuntimeCommand{command},
	}
}

func event11RecordEventGraph(ids event11FixtureIDs, runtimeRun RuntimeRun, envelope RuntimeEnvelope, envelopeImmutable bool, artifactHash string, policyCases []Event11PolicyCaseResult, opts Event11RuntimeEnvelopeDryRunOptions) (*v39.InMemoryStore, event11GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := event11FixtureTime()
	networkPolicy := envelope.NetworkPolicy
	if opts.UnsafeNetworkPolicy != "" {
		networkPolicy = opts.UnsafeNetworkPolicy
	}
	secretsPolicy := envelope.SecretsPolicy
	if opts.UnsafeSecretsPolicy != "" {
		secretsPolicy = opts.UnsafeSecretsPolicy
	}
	runtimeEnvelopeRecord := event11RuntimeEnvelopeRecord(ids, envelope, networkPolicy, secretsPolicy)
	recordedEnvelopeHash, err := event11RuntimeEnvelopeRecordHash(runtimeEnvelopeRecord)
	if err != nil {
		return nil, event11GraphRun{}, err
	}
	reportedEnvelopeHash := recordedEnvelopeHash
	if opts.OmitEnvelopeHash {
		reportedEnvelopeHash = ""
		runtimeEnvelopeRecord.EnvelopeHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	} else {
		runtimeEnvelopeRecord.EnvelopeHash = recordedEnvelopeHash
	}
	failures := event11InjectedFailures(reportedEnvelopeHash, envelopeImmutable, policyCases, opts)
	status := "certified"
	taskState := "certified"
	actorStatus := "succeeded"
	gateStatus := "pass"
	testRunStatus := "pass"
	if len(failures) > 0 {
		status = "rejected"
		taskState = "rejected"
		actorStatus = "failed"
		gateStatus = "fail"
		testRunStatus = "fail"
	}

	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: event11Common(ids.factoryOrder, v39.TypeFactoryOrder, status), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-180-merged-" + event11DocsMergeSHA, SourceIntentRef: event11AuthorityDoc, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: event11Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Prove one local deterministic RuntimeBroker envelope/result dry run under Event 11 authority.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: event11Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, "accepted"), RequirementID: ids.requirement, Text: "Runtime evidence includes immutable envelope hash, append-only result, local-only policy, no-side-effect negative cases, trace completeness, GateResult, and AuditReport-shaped closeout.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "event11_runtime_envelope_dry_run", OwnerRole: "External Committee", RiskClass: "high"},
		&v39.Task{CommonNode: event11Common(ids.task, v39.TypeTask, taskState), FactoryOrderID: &ids.factoryOrder, Cell: "cell_df_v40_event11_runtime_envelope_dry_run", State: taskState, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: event11Common(ids.actorIdentity, v39.TypeActorIdentity, "active"), ActorID: event11ActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorInvocation{CommonNode: event11Common(ids.actorInvocation, v39.TypeActorInvocation, actorStatus), TaskID: ids.task, Runtime: "local", ActorID: event11ActorID, InputContractHash: "sha256:event11-runtime-envelope-input", OutputContractHash: strPtr("sha256:event11-runtime-envelope-output")},
		&v39.AuthorityRequest{CommonNode: event11Common(ids.authorityRequest, v39.TypeAuthorityRequest, "open"), ActorID: event11ActorID, ActorRole: "Operator", Action: "repo.work.event11_runtime_envelope_dry_run.implement", TargetType: "repo", TargetID: "transpara-ai/work", RiskClass: "high", Reason: "Build one deterministic RuntimeBroker dry-run fixture under merged Event 11 authority.", ProposedCommand: strPtr("RunEvent11RuntimeEnvelopeDryRunFixture"), EvidenceRefs: []string{event11AuthorityDoc, event11DocsPR}},
		&v39.AuthorityDecision{CommonNode: event11Common(ids.authorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.authorityRequest, DeciderActorID: event11ExternalActorID, DeciderRole: "External Committee", Decision: "ApprovalRequired", Reason: "Event 11 docs#180 grants exactly one bounded Work PR lifecycle for Level 0 local deterministic RuntimeBroker dry-run evidence; Work PR merge still requires standalone exact-head approval.", Scope: event11AllowedWorkPaths(), Conditions: []string{"local deterministic RuntimeBroker only", "in-memory v3.9 projection only", "no external runtime adapter", "no network", "no secrets", "no production EventGraph write", "no protected side effects", "no Gate U closure by Work PR", "explicit PR-visible approval required before merge"}},
		runtimeEnvelopeRecord,
		&v39.Artifact{CommonNode: event11Common(ids.artifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: strPtr("fixture://work/event11-runtime-envelope-dry-run/report.txt"), ContentHash: &artifactHash},
		&v39.TestCase{CommonNode: event11Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Event 11 local deterministic RuntimeBroker envelope dry run", TestType: "unit", Path: strPtr("event11_runtime_envelope_dry_run_test.go")},
		&v39.TestRun{CommonNode: event11Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: event11Common(ids.gateResult, v39.TypeGateResult, gateStatus), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_u_event11_runtime_envelope_dry_run_fixture", EvidenceRefs: []string{ids.testRun, ids.artifact}},
	}
	if !opts.OmitAuthorityReceipt {
		records = append(records, &v39.ExecutionReceipt{CommonNode: event11Common(ids.executionReceipt, v39.TypeExecutionReceipt, "recorded"), AuthorityDecisionID: ids.authorityDecision, ActorInvocationID: &ids.actorInvocation, Action: "runtime.invoke.local.event11_dry_run_fixture", TargetID: ids.task, Result: "succeeded", EvidenceRefs: []string{ids.runtimeResult, ids.artifact}})
	}
	if !opts.OmitRuntimeResult {
		exitStatus := event11V39RuntimeStatus(runtimeRun.Result.Result.Status)
		records = append(records, &v39.RuntimeResult{CommonNode: event11Common(ids.runtimeResult, v39.TypeRuntimeResult, "recorded"), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_deterministic", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: exitStatus, ArtifactRefs: []string{ids.artifact}, ChangedFiles: event11ChangedFilePaths(runtimeRun.Result.Result.ChangedFiles), CommandLog: event11CommandLog(runtimeRun.Result.Result.CommandLog), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{ids.authorityDecision}, PostRunValidationRefs: []string{ids.testRun}})
	}
	if !opts.OmitCodeChange {
		records = append(records, &v39.CodeChange{CommonNode: event11Common(ids.codeChange, v39.TypeCodeChange, "verified"), ArtifactID: ids.artifact, ActorInvocationID: ids.actorInvocation, Repo: "transpara-ai/work", Path: "event11_runtime_envelope_dry_run.go", BeforeHash: strPtr("sha256:absent"), AfterHash: "sha256:event11-runtime-envelope-dry-run-code", ChangeType: "create"})
	}
	if len(failures) > 0 {
		records = append(records, &v39.Failure{CommonNode: event11Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "event11_runtime_evidence_incomplete", Severity: "high", Summary: strings.Join(failures, "; ")})
	}
	if err := event11AppendRecords(graph, records...); err != nil {
		return nil, event11GraphRun{}, err
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: event11Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-event11-local-runtimebroker", CapabilityVersionRefs: []string{}, RuntimeRefs: []string{"work.local_deterministic_runtimebroker@1"}}); err != nil {
		return nil, event11GraphRun{}, err
	}
	if err := event11AppendEdges(graph, ids, opts, len(failures) > 0, createdAt); err != nil {
		return nil, event11GraphRun{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: event11Common(ids.releaseCandidate, v39.TypeReleaseCandidate, status), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.artifact}})
	if err != nil {
		return nil, event11GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	authorityPath, authorityErr := graph.ActorAuthorityRequestDecisionReceipt(ids.authorityRequest)
	preDecision := event11EvaluateWithAuditRequirement(graph, ids, trace, authorityPath, traceErr, authorityErr, reportedEnvelopeHash, envelopeImmutable, policyCases, opts, false)
	if opts.OmitAuditReport {
		preDecision.Status = "fail"
		preDecision.Missing = append(preDecision.Missing, "AuditReport unavailable")
	}

	var cert *v39.Certification
	var rejection *v39.Rejection
	if preDecision.Status == "pass" {
		cert, err = graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: event11Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: event11ExternalActorID, Reason: "Event 11 local deterministic RuntimeBroker dry-run fixture evidence is complete for Level 0 only; Gate U still requires later docs evidence decision.", EvidenceRefs: []string{ids.gateResult, ids.authorityDecision, ids.executionReceipt}})
		if err != nil {
			return nil, event11GraphRun{}, err
		}
	} else {
		rejection, err = graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: event11Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: event11ExternalActorID, Reason: "Event 11 RuntimeBroker dry-run fixture is incomplete and must fail closed.", EvidenceRefs: append([]string{ids.gateResult}, preDecision.Missing...)})
		if err != nil {
			return nil, event11GraphRun{}, err
		}
	}

	var audit *v39.AuditReport
	if !opts.OmitAuditReport {
		auditStatus := "complete"
		if preDecision.Status != "pass" {
			auditStatus = "incomplete"
		}
		audit, err = graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: event11Common(ids.auditReport, v39.TypeAuditReport, auditStatus), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, event11GraphRun{}, err
		}
	}

	trace, traceErr = graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	authorityPath, authorityErr = graph.ActorAuthorityRequestDecisionReceipt(ids.authorityRequest)
	return graph, event11GraphRun{trace: trace, authorityPath: authorityPath, traceErr: traceErr, authorityErr: authorityErr, envelopeHash: reportedEnvelopeHash, certification: cert, rejection: rejection, auditReport: audit}, nil
}

func event11Evaluate(graph *v39.InMemoryStore, ids event11FixtureIDs, trace v39.TraceCompletenessGateResult, authorityPath v39.RequiredPath, traceErr error, authorityErr error, envelopeHash string, envelopeImmutable bool, policyCases []Event11PolicyCaseResult, opts Event11RuntimeEnvelopeDryRunOptions) Event11RuntimeEnvelopeDryRunReport {
	return event11EvaluateWithAuditRequirement(graph, ids, trace, authorityPath, traceErr, authorityErr, envelopeHash, envelopeImmutable, policyCases, opts, true)
}

func event11EvaluateWithAuditRequirement(graph *v39.InMemoryStore, ids event11FixtureIDs, trace v39.TraceCompletenessGateResult, authorityPath v39.RequiredPath, traceErr error, authorityErr error, envelopeHash string, envelopeImmutable bool, policyCases []Event11PolicyCaseResult, opts Event11RuntimeEnvelopeDryRunOptions, requireAudit bool) Event11RuntimeEnvelopeDryRunReport {
	reportedEnvelopeHash := envelopeHash
	if opts.OmitEnvelopeHash {
		reportedEnvelopeHash = ""
	}
	report := Event11RuntimeEnvelopeDryRunReport{
		Status:                 "pass",
		TypeCounts:             map[string]int{},
		TraceCompleted:         trace.Completed,
		TraceStatus:            trace.Status,
		AuthorityPathCompleted: authorityPath.Completed,
		EnvelopeHash:           reportedEnvelopeHash,
		EnvelopeImmutable:      envelopeImmutable,
		LocalRuntimeOnly:       true,
		PolicyCases:            append([]Event11PolicyCaseResult(nil), policyCases...),
		ForbiddenActions: []Event11ForbiddenAction{
			{Action: "external runtime adapter", Status: "not_run", Reason: "Event 11 permits local_deterministic only"},
			{Action: "general shell execution", Status: "not_run", Reason: "Runtime commands are named deterministic operations, not shell commands"},
			{Action: "network access", Status: "not_run", Reason: "network_policy is disabled and network_attempt is policy_blocked"},
			{Action: "secret access", Status: "not_run", Reason: "secrets_policy is none and secret_attempt is policy_blocked"},
			{Action: "production EventGraph write", Status: "not_run", Reason: "projection uses a local in-memory v3.9 store only"},
			{Action: "protected side effect", Status: "not_run", Reason: "fixture writes only allowed local report.txt output"},
			{Action: "Gate U closure", Status: "not_claimed", Reason: "Work evidence requires later docs evidence-decision"},
			{Action: "value allocation", Status: "not_run", Reason: "outside Event 11 authority"},
		},
		ResidualRisks: []Event11ResidualRiskState{
			{ID: "R-001", Status: "unresolved_excluded", Reason: "default-branch/protected mutation remains forbidden"},
			{ID: "R-002", Status: "unresolved_excluded", Reason: "protected side effects and production deploy remain unauthorized"},
			{ID: "R-003", Status: "unresolved_excluded", Reason: "policy-bundle reliance remains future governed work"},
		},
		EvidenceRefs: []string{ids.factoryOrder, ids.requirement, ids.acceptanceCriterion, ids.task, ids.authorityDecision, ids.runtimeEnvelope, ids.runtimeResult, ids.artifact, ids.testRun, ids.gateResult},
	}
	report.Missing = append(report.Missing, event11InjectedFailures(reportedEnvelopeHash, envelopeImmutable, policyCases, opts)...)
	for _, typ := range event11RequiredRecordTypes(requireAudit) {
		count := len(graph.ByType(typ))
		report.TypeCounts[typ] = count
		if count == 0 {
			report.Missing = append(report.Missing, typ+" missing")
		}
	}
	if !trace.Completed || trace.Status != v39.TraceCompletenessPassed {
		report.Missing = append(report.Missing, "TraceCompletenessGate incomplete")
		report.Missing = append(report.Missing, trace.Missing...)
	}
	if traceErr != nil {
		report.Missing = append(report.Missing, "TraceCompletenessGate: "+traceErr.Error())
	}
	if !authorityPath.Completed {
		report.Missing = append(report.Missing, "AuthorityRequest/AuthorityDecision/ExecutionReceipt path incomplete")
		report.Missing = append(report.Missing, authorityPath.Missing...)
	}
	if authorityErr != nil {
		report.Missing = append(report.Missing, "AuthorityRequestDecisionReceipt: "+authorityErr.Error())
	}
	report.Missing = append(report.Missing, event11LocalEvidenceMissing(graph, ids, reportedEnvelopeHash, envelopeImmutable, policyCases, opts)...)
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" && action.Status != "not_claimed" {
			report.Missing = append(report.Missing, "forbidden action status not fail-closed: "+action.Action)
		}
	}
	if len(report.Missing) > 0 {
		report.Status = "fail"
	}
	report.TraceOutput = event11RuntimeTraceOutput(ids, trace, report)
	report.GateOutput = event11RuntimeGateOutput(ids, graph, report)
	report.EventGraphHandoff = event11EventGraphHandoff(ids, report)
	return report
}

func event11RuntimeTraceOutput(ids event11FixtureIDs, trace v39.TraceCompletenessGateResult, report Event11RuntimeEnvelopeDryRunReport) Event11RuntimeTraceOutput {
	return Event11RuntimeTraceOutput{
		Status:             report.Status,
		TraceStatus:        trace.Status,
		TraceCompleted:     trace.Completed,
		FactoryOrderID:     ids.factoryOrder,
		ReleaseCandidateID: ids.releaseCandidate,
		TestRunID:          ids.testRun,
		GateResultID:       ids.gateResult,
		AuditReportID:      event11AvailableAuditReportID(report.TypeCounts, ids),
		RequiredPathCount:  len(trace.RequiredPaths),
		Missing:            append([]string(nil), report.Missing...),
		EvidenceRefs:       event11UniqueStrings(append([]string{ids.testRun, ids.gateResult}, trace.EvidenceRefs...)),
	}
}

func event11RuntimeGateOutput(ids event11FixtureIDs, graph *v39.InMemoryStore, report Event11RuntimeEnvelopeDryRunReport) Event11RuntimeGateOutput {
	output := Event11RuntimeGateOutput{
		Status:              report.Status,
		GateName:            "gate_u_event11_runtime_envelope_dry_run_fixture",
		GateScope:           "fixture_only",
		GateUClosureClaimed: false,
		FactoryOrderID:      ids.factoryOrder,
		ReleaseCandidateID:  ids.releaseCandidate,
		TestRunID:           ids.testRun,
		GateResultID:        ids.gateResult,
		AuditReportID:       event11AvailableAuditReportID(report.TypeCounts, ids),
		EvidenceRefs:        event11UniqueStrings([]string{ids.testRun, ids.gateResult}),
		Missing:             append([]string(nil), report.Missing...),
	}
	if graph != nil {
		if record, err := graph.Get(ids.gateResult); err == nil {
			if gate, ok := record.(*v39.GateResult); ok {
				output.GateName = gate.GateName
				output.EvidenceRefs = event11UniqueStrings(append([]string{ids.gateResult}, gate.EvidenceRefs...))
			}
		}
	}
	return output
}

func event11EventGraphHandoff(ids event11FixtureIDs, report Event11RuntimeEnvelopeDryRunReport) Event11EventGraphHandoff {
	status := "local_fixture_projection_complete"
	blockedBy := []string(nil)
	if report.Status != "pass" {
		status = "blocked"
		blockedBy = append([]string(nil), report.Missing...)
	}
	return Event11EventGraphHandoff{
		Status:                 status,
		ProjectionScope:        "work_local_in_memory_v39_fixture",
		PersistentWriteStatus:  "not_written",
		PersistentWriteClaimed: false,
		ProductionTruthClaimed: false,
		RuntimeExecutionScope:  "local_deterministic_fixture_only",
		EventGraphRefs:         event11HandoffEventGraphRefs(report.TypeCounts, ids),
		AuthorityRefs:          event11UniqueStrings([]string{event11AuthorityDoc, event11DocsPR, ids.authorityDecision}),
		BlockedBy:              blockedBy,
		Notes: []string{
			"handoff is a typed Work artifact projection only",
			"no production EventGraph write is performed or claimed",
			"persistent EventGraph writes require separate EventGraph authority",
		},
	}
}

func event11AvailableAuditReportID(typeCounts map[string]int, ids event11FixtureIDs) string {
	if typeCounts[v39.TypeAuditReport] == 0 {
		return ""
	}
	return ids.auditReport
}

func event11HandoffEventGraphRefs(typeCounts map[string]int, ids event11FixtureIDs) []string {
	candidates := []struct {
		typ string
		id  string
	}{
		{v39.TypeRuntimeEnvelope, ids.runtimeEnvelope},
		{v39.TypeRuntimeResult, ids.runtimeResult},
		{v39.TypeTestRun, ids.testRun},
		{v39.TypeGateResult, ids.gateResult},
		{v39.TypeAuditReport, ids.auditReport},
	}
	refs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if typeCounts[candidate.typ] > 0 {
			refs = append(refs, egRef(candidate.typ, candidate.id))
		}
	}
	return refs
}

func event11UniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func event11LocalEvidenceMissing(graph *v39.InMemoryStore, ids event11FixtureIDs, envelopeHash string, envelopeImmutable bool, policyCases []Event11PolicyCaseResult, opts Event11RuntimeEnvelopeDryRunOptions) []string {
	var missing []string
	if strings.TrimSpace(envelopeHash) == "" || !strings.HasPrefix(envelopeHash, "sha256:") {
		missing = append(missing, "RuntimeEnvelope immutable hash missing")
	}
	if !envelopeImmutable {
		missing = append(missing, "RuntimeEnvelope changed after execution")
	}
	envelopeRecord, err := graph.Get(ids.runtimeEnvelope)
	if err != nil {
		return append(missing, "RuntimeEnvelope unavailable for local checks")
	}
	envelope, ok := envelopeRecord.(*v39.RuntimeEnvelope)
	if !ok {
		return append(missing, "RuntimeEnvelope record has unexpected type")
	}
	if envelope.NetworkPolicy != "disabled" {
		missing = append(missing, "RuntimeEnvelope network_policy is not disabled")
	}
	if envelope.SecretsPolicy != "none" {
		missing = append(missing, "RuntimeEnvelope secrets_policy is not none")
	}
	if envelope.RuntimeAdapterID != "local_deterministic" {
		missing = append(missing, "RuntimeEnvelope runtime_adapter_id is not local_deterministic")
	}
	if !strings.HasPrefix(envelope.WorkingDirectory, "fixture://") {
		missing = append(missing, "RuntimeEnvelope working_directory is not fixture scoped")
	}
	for _, denied := range []string{"shell", "network_attempt", "secret_attempt", "gh pr merge", "git push origin main", "deploy", "production operation", "value allocation"} {
		if !stringIn(denied, envelope.DeniedCommands) {
			missing = append(missing, "RuntimeEnvelope denied_commands missing "+denied)
		}
	}
	resultRecord, err := graph.Get(ids.runtimeResult)
	if err == nil {
		result, ok := resultRecord.(*v39.RuntimeResult)
		if !ok {
			missing = append(missing, "RuntimeResult record has unexpected type")
		} else {
			if result.RuntimeAdapterID != "local_deterministic" {
				missing = append(missing, "RuntimeResult runtime_adapter_id is not local_deterministic")
			}
			if len(result.NetworkAccessLog) != 0 {
				missing = append(missing, "RuntimeResult network_access_log is not empty")
			}
			if len(result.SecretAccessLog) != 0 {
				missing = append(missing, "RuntimeResult secret_access_log is not empty")
			}
			for _, path := range result.ChangedFiles {
				if path != "report.txt" {
					missing = append(missing, "RuntimeResult changed_files contains unauthorized fixture path "+path)
				}
			}
		}
	}
	receiptRecord, err := graph.Get(ids.executionReceipt)
	if err == nil {
		receipt, ok := receiptRecord.(*v39.ExecutionReceipt)
		if !ok {
			missing = append(missing, "ExecutionReceipt record has unexpected type")
		} else {
			if receipt.Action != "runtime.invoke.local.event11_dry_run_fixture" {
				missing = append(missing, "ExecutionReceipt action is outside local fixture scope")
			}
			if receipt.Result != "succeeded" {
				missing = append(missing, "ExecutionReceipt result is not succeeded")
			}
			if receipt.TargetID != ids.task {
				missing = append(missing, "ExecutionReceipt target is not the fixture task")
			}
		}
	}
	if opts.OmitPolicyCases || len(policyCases) == 0 {
		missing = append(missing, "policy_blocked negative cases unavailable")
	}
	for _, policyCase := range policyCases {
		if !policyCase.SideEffectFree {
			missing = append(missing, "policy case "+policyCase.Name+" did not prove no unauthorized side effect")
		}
	}
	return missing
}

func event11InjectedFailures(envelopeHash string, envelopeImmutable bool, policyCases []Event11PolicyCaseResult, opts Event11RuntimeEnvelopeDryRunOptions) []string {
	var missing []string
	if opts.OmitAuthorityReceipt {
		missing = append(missing, "ExecutionReceipt unavailable")
	}
	if opts.OmitRuntimeResult {
		missing = append(missing, "RuntimeResult unavailable")
	}
	if opts.OmitCodeChange {
		missing = append(missing, "CodeChange unavailable")
	}
	if opts.OmitAuditReport {
		missing = append(missing, "AuditReport unavailable")
	}
	if opts.OmitPolicyCases || len(policyCases) == 0 {
		missing = append(missing, "policy cases unavailable")
	}
	if opts.OmitEnvelopeHash || strings.TrimSpace(envelopeHash) == "" {
		missing = append(missing, "RuntimeEnvelope immutable hash unavailable")
	}
	if !envelopeImmutable {
		missing = append(missing, "RuntimeEnvelope immutability check failed")
	}
	if opts.UnsafeNetworkPolicy != "" {
		missing = append(missing, "RuntimeEnvelope network_policy widened")
	}
	if opts.UnsafeSecretsPolicy != "" {
		missing = append(missing, "RuntimeEnvelope secrets_policy widened")
	}
	for _, policyCase := range policyCases {
		if !policyCase.SideEffectFree {
			missing = append(missing, "policy case "+policyCase.Name+" side-effect check failed")
		}
	}
	return missing
}

func event11AppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func event11AppendEdges(graph *v39.InMemoryStore, ids event11FixtureIDs, opts Event11RuntimeEnvelopeDryRunOptions, includeFailure bool, createdAt time.Time) error {
	edges := []v39.CommonEdge{
		event11Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		event11Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		event11Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		event11Edge("identity_auth_request", v39.EdgeRequestedAuthority, ids.actorIdentity, ids.authorityRequest, createdAt),
		event11Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		event11Edge("invocation_auth_request", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
		event11Edge("auth_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
		event11Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		event11Edge("task_artifact", v39.EdgeProduced, ids.task, ids.artifact, createdAt),
		event11Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		event11Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		event11Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	if !opts.OmitAuthorityReceipt {
		edges = append(edges, event11Edge("auth_receipt", v39.EdgeReceiptedBy, ids.authorityDecision, ids.executionReceipt, createdAt))
	}
	if !opts.OmitRuntimeResult {
		edges = append(edges, event11Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt))
	}
	if !opts.OmitCodeChange {
		edges = append(edges, event11Edge("artifact_code_change", v39.EdgeModified, ids.artifact, ids.codeChange, createdAt))
	}
	if includeFailure {
		edges = append(edges, event11Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func event11RequiredRecordTypes(requireAudit bool) []string {
	types := []string{
		v39.TypeFactoryOrder,
		v39.TypeRequirement,
		v39.TypeAcceptanceCriterion,
		v39.TypeTask,
		v39.TypeActorIdentity,
		v39.TypeActorInvocation,
		v39.TypeAuthorityRequest,
		v39.TypeAuthorityDecision,
		v39.TypeExecutionReceipt,
		v39.TypeRuntimeEnvelope,
		v39.TypeRuntimeResult,
		v39.TypeArtifact,
		v39.TypeCodeChange,
		v39.TypeTestCase,
		v39.TypeTestRun,
		v39.TypeGateResult,
		v39.TypeFactoryRuntimeVersion,
		v39.TypeReleaseCandidate,
	}
	if requireAudit {
		types = append(types, v39.TypeAuditReport)
	}
	return types
}

func event11IDs() event11FixtureIDs {
	return event11FixtureIDs{
		factoryOrder:        "fo_df_v40_event11_runtime_envelope_001",
		requirement:         "req_df_v40_event11_runtime_envelope_001",
		acceptanceCriterion: "ac_df_v40_event11_runtime_envelope_001",
		task:                "tsk_df_v40_event11_runtime_envelope_001",
		actorIdentity:       "actor_identity_df_v40_event11_runtime_envelope",
		actorInvocation:     "inv_df_v40_event11_runtime_envelope_001",
		authorityRequest:    "auth_req_df_v40_event11_runtime_envelope_001",
		authorityDecision:   "auth_dec_df_v40_event11_runtime_envelope_001",
		executionReceipt:    "exec_df_v40_event11_runtime_envelope_001",
		runtimeEnvelope:     "env_df_v40_event11_runtime_envelope_001",
		runtimeResult:       "res_df_v40_event11_runtime_envelope_001",
		artifact:            "artifact_df_v40_event11_runtime_envelope_report",
		codeChange:          "codechange_df_v40_event11_runtime_envelope_go",
		testCase:            "tc_df_v40_event11_runtime_envelope_001",
		testRun:             "tr_df_v40_event11_runtime_envelope_001",
		gateResult:          "gate_df_v40_event11_runtime_envelope_001",
		failure:             "fail_df_v40_event11_runtime_envelope_001",
		factoryRuntime:      "frv_df_v40_event11_runtime_envelope_local",
		releaseCandidate:    "rc_df_v40_event11_runtime_envelope_001",
		certification:       "cert_df_v40_event11_runtime_envelope_001",
		rejection:           "rej_df_v40_event11_runtime_envelope_001",
		auditReport:         "audit_df_v40_event11_runtime_envelope_001",
	}
}

func event11AllowedWorkPaths() []string {
	return []string{
		"event11_runtime_envelope_dry_run.go",
		"event11_runtime_envelope_dry_run_test.go",
		"docs/designs/governed-runtime-envelope-dry-run.md",
	}
}

func event11DeniedCommands() []string {
	return []string{
		"denied_command",
		"shell",
		"exec",
		"network_attempt",
		"secret_attempt",
		"git push origin main",
		"gh pr merge",
		"docker",
		"kubectl",
		"terraform",
		"deploy",
		"production operation",
		"value allocation",
	}
}

func event11RuntimeEnvelopeRecord(ids event11FixtureIDs, envelope RuntimeEnvelope, networkPolicy, secretsPolicy string) *v39.RuntimeEnvelope {
	return &v39.RuntimeEnvelope{
		CommonNode:               event11Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"),
		RuntimeAdapterID:         envelope.Worker,
		RuntimeAdapterVersion:    "1",
		FactoryRuntimeVersionRef: ids.factoryRuntime,
		TaskID:                   ids.task,
		ActorID:                  event11ActorID,
		AuthorityDecisionRef:     ids.authorityDecision,
		AllowedFiles:             cloneStrings(envelope.AllowedFiles),
		DeniedFiles:              cloneStrings(envelope.DeniedFiles),
		AllowedCommands:          cloneStrings(envelope.AllowedCommands),
		DeniedCommands:           cloneStrings(envelope.DeniedCommands),
		NetworkPolicy:            networkPolicy,
		SecretsPolicy:            secretsPolicy,
		WorkingDirectory:         envelope.WorkingDirectory,
		Timeout:                  event11RuntimeTimeout(envelope.TimeoutMillis),
		ResourceLimits:           event11RuntimeResourceLimits(envelope.ResourceLimits),
		ExpectedOutputs:          cloneStrings(envelope.ExpectedOutputs),
		OutputContract:           map[string]any{"mode": Event11RuntimeEnvelopeDryRunMode, "gate": "Gate U remains open pending separate docs evidence decision"},
		TraceRequiredPaths:       []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "ActorIdentity / AuthorityRequest / AuthorityDecision / ExecutionReceipt", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact -> CodeChange", "Task -> TestCase -> TestRun -> GateResult", "ReleaseCandidate -> Certification or Rejection -> AuditReport"},
		PostRunValidationPlan:    []string{"go test ./...", "go vet ./...", "make verify", "exact-head adversarial review", "standalone External Committee approval before merge"},
	}
}

func event11RuntimeEnvelopeRecordHash(envelope *v39.RuntimeEnvelope) (string, error) {
	if envelope == nil {
		return "", errors.New("runtime envelope record is required")
	}
	hashEnvelope := *envelope
	hashEnvelope.EnvelopeHash = ""
	encoded, err := json.Marshal(hashEnvelope)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func event11RuntimeTimeout(timeoutMillis int) string {
	if timeoutMillis <= 0 {
		return "0s"
	}
	if timeoutMillis%1000 == 0 {
		return fmt.Sprintf("%ds", timeoutMillis/1000)
	}
	return (time.Duration(timeoutMillis) * time.Millisecond).String()
}

func event11RuntimeResourceLimits(limits RuntimeResourceLimits) map[string]any {
	return map[string]any{
		"max_files_changed": limits.MaxFilesChanged,
		"max_output_bytes":  limits.MaxOutputBytes,
		"max_memory_bytes":  limits.MaxMemoryBytes,
	}
}

func event11EnvelopeHash(envelope RuntimeEnvelope) (string, error) {
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func event11RuntimeArtifactHash(result RuntimeResult) (string, error) {
	for _, artifact := range result.Artifacts {
		if artifact.Path == "report.txt" && artifact.SHA256 != "" {
			return "sha256:" + artifact.SHA256, nil
		}
	}
	return "", errors.New("runtime artifact report.txt was not captured")
}

func event11V39RuntimeStatus(status RuntimeStatus) string {
	switch status {
	case RuntimeStatusSucceeded:
		return "succeeded"
	case RuntimeStatusTimedOut:
		return "timed_out"
	case RuntimeStatusPolicyBlocked:
		return "policy_blocked"
	default:
		return "failed"
	}
}

func event11ChangedFilePaths(files []RuntimeFileArtifact) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
}

func event11CommandLog(logs []RuntimeCommandLog) []string {
	out := make([]string, 0, len(logs))
	for _, log := range logs {
		out = append(out, fmt.Sprintf("%d:%s:%s", log.Index, log.Name, log.Status))
	}
	return out
}

func event11WorkArtifactBody(report Event11RuntimeEnvelopeDryRunReport) string {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return `{"status":"error","error":"marshal event11 report"}`
	}
	return string(encoded)
}

func event11Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{
		ID:             id,
		Type:           typ,
		CreatedAt:      event11FixtureTime(),
		CreatedBy:      event11ActorID,
		Status:         strPtr(status),
		Version:        "4.0.0-event11-runtime-envelope-dry-run",
		IdempotencyKey: "idem_" + id,
		CorrelationID:  event11CorrelationID,
		SourceRefs: []string{
			event11AuthorityDoc,
			event11DocsPR,
			"docs-pr-180-merge-" + event11DocsMergeSHA,
			"docs-pr-180-reviewed-head-" + event11DocsReviewedHead,
		},
	}
}

func event11Edge(label, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_df_v40_event11_" + label + ":" + from + ":" + typ + ":" + to
	return v39.CommonEdge{
		ID:             id,
		Type:           typ,
		FromID:         from,
		ToID:           to,
		CreatedAt:      createdAt,
		CreatedBy:      event11ActorID,
		CorrelationID:  event11CorrelationID,
		IdempotencyKey: "idem_" + id,
		EvidenceRefs:   []string{event11AuthorityDoc},
	}
}

func event11FixtureTime() time.Time {
	t, err := time.Parse(time.RFC3339, event11FixtureTimeRFC)
	if err != nil {
		panic(err)
	}
	return t
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Event11RuntimeEnvelopeDryRunStatus(run Event11RuntimeEnvelopeDryRunRun) (string, error) {
	if run.EventGraph == nil {
		return "fail", errors.New("event11 runtime envelope dry-run has nil EventGraph")
	}
	if run.Report.Status == "" {
		return "fail", errors.New("event11 runtime envelope dry-run report is missing")
	}
	if run.Report.Status != "pass" {
		return run.Report.Status, fmt.Errorf("event11 runtime envelope dry-run incomplete: %s", strings.Join(run.Report.Missing, "; "))
	}
	return "pass", nil
}
