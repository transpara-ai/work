package work

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Epic2ThinFactoryCertified Epic2ThinFactoryMode = "certified"
	Epic2ThinFactoryRejected  Epic2ThinFactoryMode = "rejected"
)

const (
	epic2FixtureActorID = "act_epic2_local_factory"
	epic2FixtureTimeRFC = "2026-05-19T12:00:00Z"
)

// Epic2ThinFactoryMode selects the authorized happy or negative local fixture path.
type Epic2ThinFactoryMode string

// Epic2ThinFactoryOptions keeps the fixture bounded to a caller-provided local work directory.
type Epic2ThinFactoryOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic2ThinFactoryMode
}

// Epic2ThinFactoryRun is the in-memory evidence packet produced by the thin slice.
type Epic2ThinFactoryRun struct {
	Mode                  Epic2ThinFactoryMode
	WorkTask              Task
	WorkProjection        TaskProjection
	RuntimeRun            RuntimeRun
	EventGraph            *v39.InMemoryStore
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	ArtifactID            string
	TestRunID             string
	GateResultID          string
	FailureID             string
	ReleaseCandidateID    string
	DecisionID            string
	AuditReportID         string
	TraceCompleteness     v39.TraceCompletenessGateResult
	Certification         *v39.Certification
	Rejection             *v39.Rejection
	AuditReport           *v39.AuditReport
	Projection            Epic2OpsEvidenceProjection
}

// Epic2OpsEvidenceProjection matches the Site /ops/evidence projection contract.
type Epic2OpsEvidenceProjection struct {
	GeneratedAt       string                           `json:"generated_at"`
	Source            string                           `json:"source"`
	FactoryOrder      *Epic2EvidenceFactoryOrder       `json:"factory_order"`
	ReleaseCandidate  *Epic2EvidenceReleaseCandidate   `json:"release_candidate"`
	Decision          *Epic2EvidenceDecision           `json:"decision"`
	AuditReport       *Epic2EvidenceAuditReport        `json:"audit_report"`
	Timeline          []Epic2EvidenceTimelineEvent     `json:"timeline"`
	GateEvidence      []Epic2EvidenceGate              `json:"gate_evidence"`
	ReleaseEvidence   []Epic2EvidenceReleaseEvidence   `json:"release_evidence"`
	FailuresRepairs   []Epic2EvidenceFailureRepair     `json:"failures_repairs"`
	MissingProvenance []Epic2EvidenceMissingProvenance `json:"missing_provenance"`
	ProofOfWorkPacket *Epic2ProofOfWorkPacket          `json:"proof_of_work_packet"`
	Errors            []string                         `json:"errors"`
}

type Epic2EvidenceFactoryOrder struct {
	ID               string `json:"id"`
	Version          int    `json:"version"`
	Status           string `json:"status"`
	SourceIntentHash string `json:"source_intent_hash"`
	SourceIntentRef  string `json:"source_intent_ref"`
	RiskClass        string `json:"risk_class"`
	ReleasePolicy    string `json:"release_policy"`
}

type Epic2EvidenceReleaseCandidate struct {
	ID                      string   `json:"id"`
	Status                  string   `json:"status"`
	FactoryOrderID          string   `json:"factory_order_id"`
	FactoryRuntimeVersionID string   `json:"factory_runtime_version_id"`
	ArtifactRefs            []string `json:"artifact_refs"`
}

type Epic2EvidenceDecision struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	ActorID      string   `json:"actor_id"`
	Reason       string   `json:"reason"`
	EvidenceRefs []string `json:"evidence_refs"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

type Epic2EvidenceAuditReport struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type Epic2EvidenceTimelineEvent struct {
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	NodeID    string `json:"node_id"`
	CreatedAt string `json:"created_at"`
	Summary   string `json:"summary"`
}

type Epic2EvidenceGate struct {
	GateName     string   `json:"gate_name"`
	Status       string   `json:"status"`
	GateResultID string   `json:"gate_result_id"`
	EvidenceRefs []string `json:"evidence_refs"`
	WaiverRef    string   `json:"waiver_ref"`
	MissingRefs  []string `json:"missing_refs"`
}

type Epic2EvidenceReleaseEvidence struct {
	Label            string   `json:"label"`
	Status           string   `json:"status"`
	ArtifactRefs     []string `json:"artifact_refs"`
	RuntimeRefs      []string `json:"runtime_refs"`
	BOMRefs          []string `json:"bom_refs"`
	RequiredPathRefs []string `json:"required_path_refs"`
	MissingRefs      []string `json:"missing_refs"`
}

type Epic2EvidenceFailureRepair struct {
	FailureID         string `json:"failure_id"`
	FailureClass      string `json:"failure_class"`
	Severity          string `json:"severity"`
	Summary           string `json:"summary"`
	TaskID            string `json:"task_id"`
	GateResultID      string `json:"gate_result_id"`
	TestRunID         string `json:"test_run_id"`
	RepairID          string `json:"repair_id"`
	RepairStatus      string `json:"repair_status"`
	ActorInvocationID string `json:"actor_invocation_id"`
}

type Epic2EvidenceMissingProvenance struct {
	PathName  string   `json:"path_name"`
	NodeIDs   []string `json:"node_ids"`
	EdgeIDs   []string `json:"edge_ids"`
	Missing   []string `json:"missing"`
	Completed bool     `json:"completed"`
}

type Epic2ProofOfWorkPacket struct {
	ID                     string                 `json:"id"`
	Status                 string                 `json:"status"`
	Summary                string                 `json:"summary"`
	WorkItem               *Epic2ProofOfWorkItem  `json:"work_item"`
	RuntimeInvocation      *Epic2ProofOfWorkItem  `json:"runtime_invocation"`
	ChangedFiles           []Epic2ProofOfWorkItem `json:"changed_files"`
	TestsRun               []Epic2ProofOfWorkItem `json:"tests_run"`
	CIStatus               *Epic2ProofOfWorkItem  `json:"ci_status"`
	ReviewFeedback         []Epic2ProofOfWorkItem `json:"review_feedback"`
	SecurityScanResults    []Epic2ProofOfWorkItem `json:"security_scan_results"`
	ScreenshotsWalkthrough []Epic2ProofOfWorkItem `json:"screenshots_walkthrough_artifacts"`
	KnownFailures          []Epic2ProofOfWorkItem `json:"known_failures"`
	OperatorDecision       *Epic2ProofOfWorkItem  `json:"operator_decision"`
	EventGraphRefs         []string               `json:"event_graph_refs"`
}

type Epic2ProofOfWorkItem struct {
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	ArtifactRef    string   `json:"artifact_ref"`
	EventGraphRefs []string `json:"event_graph_refs"`
}

// RunEpic2ThinFactoryVerticalSlice executes the authorized Epic 2 local/dry-run fixture.
func RunEpic2ThinFactoryVerticalSlice(ts *TaskStore, opts Epic2ThinFactoryOptions) (Epic2ThinFactoryRun, error) {
	if ts == nil {
		return Epic2ThinFactoryRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic2ThinFactoryRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic2ThinFactoryRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic2ThinFactoryRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic2ThinFactoryCertified
	}
	if opts.Mode != Epic2ThinFactoryCertified && opts.Mode != Epic2ThinFactoryRejected {
		return Epic2ThinFactoryRun{}, fmt.Errorf("unsupported Epic 2 fixture mode %q", opts.Mode)
	}

	ids := epic2IDs(opts.Mode)
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 2 Thin Factory Vertical Slice",
		Description:            "Run the bounded local/dry-run deterministic text artifact fixture.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic2_local_factory",
		RiskClass:              "low",
		ExpectedOutputs:        []string{"out.txt"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 2 local fixture lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic2ThinFactoryRun{}, err
		}
	}

	content := epic2RuntimeContent(ids, opts.Mode)
	runtimeRun, err := ts.RunLocalRuntime(opts.Source, RuntimeEnvelope{
		TaskID:           task.ID,
		Worker:           "local_deterministic",
		WorkingDirectory: opts.WorkingDir,
		AllowedCommands:  []string{"write_file", "checksum_file"},
		DeniedCommands:   []string{"network_attempt", "secret_attempt"},
		AllowedFiles:     []string{"out.txt"},
		DeniedFiles:      []string{"secret.txt", ".git", "../"},
		NetworkPolicy:    "disabled",
		SecretsPolicy:    "none",
		TimeoutMillis:    1000,
		ResourceLimits: RuntimeResourceLimits{
			MaxFilesChanged: 1,
			MaxOutputBytes:  4096,
			MaxMemoryBytes:  1024 * 1024,
		},
		ExpectedOutputs: []string{"out.txt"},
		Commands: []RuntimeCommand{
			{Name: "write_file", Args: []string{"out.txt", content}},
			{Name: "checksum_file", Args: []string{"out.txt"}},
		},
	}, causes, opts.ConversationID)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}
	if runtimeRun.Result.Result.Status != RuntimeStatusSucceeded {
		return Epic2ThinFactoryRun{}, fmt.Errorf("Epic 2 runtime status %s: %s", runtimeRun.Result.Result.Status, runtimeRun.Result.Result.Error)
	}
	artifactHash, err := epic2RuntimeArtifactHash(runtimeRun.Result.Result)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}

	graph, graphRun, err := epic2RecordEventGraph(ids, opts.Mode, artifactHash, runtimeRun)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}

	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 2 EventGraph fixture evidence attached", causes, opts.ConversationID); err != nil {
		return Epic2ThinFactoryRun{}, err
	}
	if opts.Mode == Epic2ThinFactoryRejected {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{
			FailureIDs: []string{ids.failure},
		}, "Epic 2 negative fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic2ThinFactoryRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 2 verification evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic2ThinFactoryRun{}, err
	}
	if opts.Mode == Epic2ThinFactoryCertified {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 2 fixture certified", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic2ThinFactoryRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 2 negative fixture rejected", []string{ids.gateResult, ids.failure}, causes, opts.ConversationID); err != nil {
		return Epic2ThinFactoryRun{}, err
	}

	projection, err := epic2BuildProjection(graph, ids, opts.Mode, graphRun, task, runtimeRun)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic2ThinFactoryRun{}, err
	}

	return Epic2ThinFactoryRun{
		Mode:                  opts.Mode,
		WorkTask:              task,
		WorkProjection:        workProjection,
		RuntimeRun:            runtimeRun,
		EventGraph:            graph,
		FactoryOrderID:        ids.factoryOrder,
		RequirementID:         ids.requirement,
		AcceptanceCriterionID: ids.acceptanceCriterion,
		TaskID:                ids.task,
		ArtifactID:            ids.artifact,
		TestRunID:             ids.testRun,
		GateResultID:          ids.gateResult,
		FailureID:             ids.failure,
		ReleaseCandidateID:    ids.releaseCandidate,
		DecisionID:            graphRun.DecisionID,
		AuditReportID:         ids.auditReport,
		TraceCompleteness:     graphRun.Trace,
		Certification:         graphRun.Certification,
		Rejection:             graphRun.Rejection,
		AuditReport:           graphRun.AuditReport,
		Projection:            projection,
	}, nil
}

func (p Epic2OpsEvidenceProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

type epic2FixtureIDs struct {
	suffix              string
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	actorInvocation     string
	authorityRequest    string
	authorityDecision   string
	executionReceipt    string
	runtimeEnvelope     string
	runtimeResult       string
	artifact            string
	testCase            string
	testRun             string
	gateResult          string
	failure             string
	factoryRuntime      string
	releaseCandidate    string
	certification       string
	rejection           string
	auditReport         string
	proofPacket         string
}

type epic2GraphRun struct {
	DecisionID    string
	Trace         v39.TraceCompletenessGateResult
	Certification *v39.Certification
	Rejection     *v39.Rejection
	AuditReport   *v39.AuditReport
}

func epic2IDs(mode Epic2ThinFactoryMode) epic2FixtureIDs {
	suffix := "certified"
	if mode == Epic2ThinFactoryRejected {
		suffix = "rejected"
	}
	return epic2FixtureIDs{
		suffix:              suffix,
		factoryOrder:        "fo_epic2_" + suffix,
		requirement:         "req_epic2_" + suffix,
		acceptanceCriterion: "ac_epic2_" + suffix,
		task:                "tsk_epic2_" + suffix,
		actorInvocation:     "invoke_epic2_" + suffix,
		authorityRequest:    "auth_req_epic2_" + suffix,
		authorityDecision:   "auth_dec_epic2_" + suffix,
		executionReceipt:    "exec_epic2_" + suffix,
		runtimeEnvelope:     "env_epic2_" + suffix,
		runtimeResult:       "rr_epic2_" + suffix,
		artifact:            "art_epic2_" + suffix,
		testCase:            "tc_epic2_" + suffix,
		testRun:             "tr_epic2_" + suffix,
		gateResult:          "gate_epic2_" + suffix,
		failure:             "fail_epic2_" + suffix,
		factoryRuntime:      "frv_epic2_" + suffix,
		releaseCandidate:    "rc_epic2_" + suffix,
		certification:       "cert_epic2_" + suffix,
		rejection:           "rej_epic2_" + suffix,
		auditReport:         "aud_epic2_" + suffix,
		proofPacket:         "pow_epic2_" + suffix,
	}
}

func epic2RuntimeContent(ids epic2FixtureIDs, mode Epic2ThinFactoryMode) string {
	outcome := "certified"
	if mode == Epic2ThinFactoryRejected {
		outcome = "rejected"
	}
	return strings.Join([]string{
		"Epic 2 Thin Factory Vertical Slice",
		"factory_order: " + ids.factoryOrder,
		"task: " + ids.task,
		"runtime: local_deterministic",
		"network_policy: disabled",
		"secrets_policy: none",
		"capability_evidence: none",
		"expected_outcome: " + outcome,
		"",
	}, "\n")
}

func epic2RuntimeArtifactHash(result RuntimeResult) (string, error) {
	for _, artifact := range result.Artifacts {
		if artifact.Path == "out.txt" && artifact.SHA256 != "" {
			return "sha256:" + artifact.SHA256, nil
		}
	}
	return "", errors.New("runtime artifact out.txt was not captured")
}

func epic2RecordEventGraph(ids epic2FixtureIDs, mode Epic2ThinFactoryMode, artifactHash string, runtimeRun RuntimeRun) (*v39.InMemoryStore, epic2GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic2FixtureTime()
	traceStartedAt := createdAt
	traceCompletedAt := createdAt.Add(time.Second)
	// The EventGraph envelope uses a deterministic placeholder; the Work runtime
	// evidence retains the actual caller-provided temp directory.
	eventGraphWorkingDirectory := "fixture://local-dry-run"
	status := "certified"
	gateStatus := "pass"
	testRunStatus := "pass"
	if mode == Epic2ThinFactoryRejected {
		status = "rejected"
		gateStatus = "fail"
		testRunStatus = "fail"
	}

	if err := epic2AppendRecords(graph,
		&v39.FactoryOrder{CommonNode: epic2Common(ids.factoryOrder, v39.TypeFactoryOrder, status), FactoryOrderVersion: 1, SourceIntentHash: "sha256:epic2-human-selection", SourceIntentRef: "docs#64", RiskClass: "low", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic2Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Produce one deterministic local dry-run artifact with append-only evidence.", Source: "explicit", RiskClass: "low"},
		&v39.AcceptanceCriterion{CommonNode: epic2Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, "accepted"), RequirementID: ids.requirement, Text: "Evidence must prove local runtime, artifact, test, gate, release decision, audit, and proof-of-work projection without capability usage.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "eventgraph_trace", OwnerRole: "maintainer", RiskClass: "low"},
		&v39.Task{CommonNode: epic2Common(ids.task, v39.TypeTask, status), FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic2_local_factory", State: status, Priority: 1, RiskClass: "low", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic2Common("actor_identity_"+ids.suffix, v39.TypeActorIdentity, "active"), ActorID: epic2FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorInvocation{CommonNode: epic2Common(ids.actorInvocation, v39.TypeActorInvocation, "succeeded"), TaskID: ids.task, Runtime: "local", ActorID: epic2FixtureActorID, InputContractHash: "sha256:epic2-input", OutputContractHash: strPtr("sha256:epic2-output")},
		&v39.AuthorityRequest{CommonNode: epic2Common(ids.authorityRequest, v39.TypeAuthorityRequest, "open"), ActorID: epic2FixtureActorID, ActorRole: "agent", Action: "runtime.invoke.local", TargetType: "task", TargetID: ids.task, RiskClass: "low", Reason: "Run only the authorized local deterministic dry-run fixture."},
		&v39.AuthorityDecision{CommonNode: epic2Common(ids.authorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.authorityRequest, DeciderActorID: "act_human", DeciderRole: "maintainer", Decision: "Autonomous", Reason: "Local deterministic fixture only; no protected side effects.", Scope: []string{"runtime.invoke.local"}},
		&v39.ExecutionReceipt{CommonNode: epic2Common(ids.executionReceipt, v39.TypeExecutionReceipt, "recorded"), AuthorityDecisionID: ids.authorityDecision, ActorInvocationID: &ids.actorInvocation, Action: "runtime.invoke.local", TargetID: ids.task, Result: "succeeded", EvidenceRefs: []string{ids.runtimeResult}},
		&v39.RuntimeEnvelope{CommonNode: epic2Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_deterministic", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: epic2FixtureActorID, AuthorityDecisionRef: ids.authorityDecision, AllowedFiles: []string{"out.txt"}, DeniedFiles: []string{"secret.txt", ".git", "../"}, AllowedCommands: []string{"write_file", "checksum_file"}, DeniedCommands: []string{"network_attempt", "secret_attempt"}, NetworkPolicy: "disabled", SecretsPolicy: "none", WorkingDirectory: eventGraphWorkingDirectory, Timeout: "1s", ResourceLimits: map[string]any{"max_files_changed": 1, "max_output_bytes": 4096}, ExpectedOutputs: []string{"out.txt"}, OutputContract: map[string]any{"format": "text"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"checksum_file out.txt"}, EnvelopeHash: "sha256:epic2-envelope"},
		&v39.RuntimeResult{CommonNode: epic2Common(ids.runtimeResult, v39.TypeRuntimeResult, "recorded"), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_deterministic", StartedAt: traceStartedAt, CompletedAt: traceCompletedAt, ExitStatus: "succeeded", ArtifactRefs: []string{ids.artifact}, ChangedFiles: []string{"out.txt"}, CommandLog: epic2CommandLog(runtimeRun.Result.Result.CommandLog), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{ids.authorityDecision}, PostRunValidationRefs: []string{ids.testRun}},
		&v39.Artifact{CommonNode: epic2Common(ids.artifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "document", Path: strPtr("out.txt"), ContentHash: &artifactHash},
		&v39.TestCase{CommonNode: epic2Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 2 deterministic local fixture acceptance", TestType: "unit", Path: strPtr("work/epic2_thin_factory_test.go")},
		&v39.TestRun{CommonNode: epic2Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: epic2Common(ids.gateResult, v39.TypeGateResult, gateStatus), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "trace_completeness", EvidenceRefs: []string{ids.testRun}},
	); err != nil {
		return nil, epic2GraphRun{}, err
	}
	if mode == Epic2ThinFactoryRejected {
		if err := epic2AppendRecords(graph, &v39.Failure{CommonNode: epic2Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "traceability_gap", Severity: "high", Summary: "Negative fixture intentionally omits repair evidence so GateResult cannot certify."}); err != nil {
			return nil, epic2GraphRun{}, err
		}
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic2Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-epic2-local", CapabilityVersionRefs: []string{}, RuntimeRefs: []string{"work.local_deterministic@1"}}); err != nil {
		return nil, epic2GraphRun{}, err
	}
	if err := epic2AppendEdges(graph, ids, mode, createdAt); err != nil {
		return nil, epic2GraphRun{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: epic2Common(ids.releaseCandidate, v39.TypeReleaseCandidate, status), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.artifact}})
	if err != nil {
		return nil, epic2GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	if mode == Epic2ThinFactoryCertified && traceErr != nil {
		return nil, epic2GraphRun{}, traceErr
	}
	if mode == Epic2ThinFactoryCertified && !trace.Completed {
		return nil, epic2GraphRun{}, errors.New("certified fixture trace completeness was not completed")
	}
	if mode == Epic2ThinFactoryRejected && traceErr == nil {
		return nil, epic2GraphRun{}, errors.New("negative fixture unexpectedly passed TraceCompletenessGate")
	}

	if mode == Epic2ThinFactoryCertified {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic2Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: "act_human", Reason: "Epic 2 local fixture evidence is complete and capability-free.", EvidenceRefs: []string{ids.gateResult}})
		if err != nil {
			return nil, epic2GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic2Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic2GraphRun{}, err
		}
		return graph, epic2GraphRun{DecisionID: cert.CommonNode.ID, Trace: trace, Certification: cert, AuditReport: audit}, nil
	}

	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic2Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: "act_human", Reason: "Negative fixture exposes incomplete trace evidence and must not certify.", EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic2GraphRun{}, err
	}
	audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic2Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
	if err != nil {
		return nil, epic2GraphRun{}, err
	}
	return graph, epic2GraphRun{DecisionID: rejection.CommonNode.ID, Trace: trace, Rejection: rejection, AuditReport: audit}, nil
}

func epic2AppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func epic2AppendEdges(graph *v39.InMemoryStore, ids epic2FixtureIDs, mode Epic2ThinFactoryMode, createdAt time.Time) error {
	edges := []v39.CommonEdge{
		epic2Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic2Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic2Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic2Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic2Edge("auth_request", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
		epic2Edge("auth_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
		epic2Edge("auth_receipt", v39.EdgeReceiptedBy, ids.authorityDecision, ids.executionReceipt, createdAt),
		epic2Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic2Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic2Edge("task_artifact", v39.EdgeProduced, ids.task, ids.artifact, createdAt),
		epic2Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic2Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic2Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	if mode == Epic2ThinFactoryRejected {
		edges = append(edges, epic2Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic2BuildProjection(graph *v39.InMemoryStore, ids epic2FixtureIDs, mode Epic2ThinFactoryMode, graphRun epic2GraphRun, task Task, runtimeRun RuntimeRun) (Epic2OpsEvidenceProjection, error) {
	foRecord, err := graph.Get(ids.factoryOrder)
	if err != nil {
		return Epic2OpsEvidenceProjection{}, err
	}
	rcRecord, err := graph.Get(ids.releaseCandidate)
	if err != nil {
		return Epic2OpsEvidenceProjection{}, err
	}
	gateRecord, err := graph.Get(ids.gateResult)
	if err != nil {
		return Epic2OpsEvidenceProjection{}, err
	}
	runtimeRecord, err := graph.Get(ids.runtimeResult)
	if err != nil {
		return Epic2OpsEvidenceProjection{}, err
	}
	audit := graphRun.AuditReport
	fo := foRecord.(*v39.FactoryOrder)
	rc := rcRecord.(*v39.ReleaseCandidate)
	gate := gateRecord.(*v39.GateResult)
	runtimeResult := runtimeRecord.(*v39.RuntimeResult)
	status := statusString(gate.CommonNode.Status)
	decision := epic2ProjectionDecision(graphRun)
	packetStatus := "pass"
	if graphRun.Rejection != nil {
		packetStatus = "fail"
	}
	timeline := []Epic2EvidenceTimelineEvent{
		{Label: "Factory order", Kind: v39.TypeFactoryOrder, Status: statusString(fo.CommonNode.Status), NodeID: fo.CommonNode.ID, CreatedAt: fo.CommonNode.CreatedAt.Format(time.RFC3339), Summary: "Epic 2 human-selected thin factory fixture."},
		{Label: "Runtime", Kind: v39.TypeRuntimeResult, Status: string(runtimeRun.Result.Result.Status), NodeID: runtimeResult.CommonNode.ID, CreatedAt: runtimeResult.StartedAt.Format(time.RFC3339), Summary: "Local deterministic runtime produced out.txt."},
	}
	if graphRun.Rejection != nil {
		failureRecord, err := graph.Get(ids.failure)
		if err != nil {
			return Epic2OpsEvidenceProjection{}, err
		}
		failure := failureRecord.(*v39.Failure)
		timeline = append(timeline, Epic2EvidenceTimelineEvent{Label: "Failure", Kind: v39.TypeFailure, Status: statusString(failure.CommonNode.Status), NodeID: failure.CommonNode.ID, CreatedAt: failure.CommonNode.CreatedAt.Format(time.RFC3339), Summary: failure.Summary})
	}
	timeline = append(timeline, Epic2EvidenceTimelineEvent{Label: "Decision", Kind: decision.Kind, Status: decision.Status, NodeID: decision.ID, CreatedAt: decision.CreatedAt, Summary: decision.Reason})
	projection := Epic2OpsEvidenceProjection{
		GeneratedAt: epic2FixtureTime().Format(time.RFC3339),
		Source:      "work-epic2-thin-factory-fixture",
		FactoryOrder: &Epic2EvidenceFactoryOrder{
			ID:               fo.CommonNode.ID,
			Version:          fo.FactoryOrderVersion,
			Status:           statusString(fo.CommonNode.Status),
			SourceIntentHash: fo.SourceIntentHash,
			SourceIntentRef:  fo.SourceIntentRef,
			RiskClass:        fo.RiskClass,
			ReleasePolicy:    fo.ReleasePolicy,
		},
		ReleaseCandidate: &Epic2EvidenceReleaseCandidate{
			ID:                      rc.CommonNode.ID,
			Status:                  statusString(rc.CommonNode.Status),
			FactoryOrderID:          rc.FactoryOrderID,
			FactoryRuntimeVersionID: derefString(rc.FactoryRuntimeVersionID),
			ArtifactRefs:            append([]string(nil), rc.ArtifactRefs...),
		},
		Decision: decision,
		AuditReport: &Epic2EvidenceAuditReport{
			ID:           audit.CommonNode.ID,
			TargetType:   audit.TargetType,
			TargetID:     audit.TargetID,
			Status:       statusString(audit.CommonNode.Status),
			TraceScore:   audit.TraceScore,
			MissingLinks: append([]string(nil), audit.MissingLinks...),
		},
		Timeline: timeline,
		GateEvidence: []Epic2EvidenceGate{
			{GateName: gate.GateName, Status: status, GateResultID: gate.CommonNode.ID, EvidenceRefs: append([]string(nil), gate.EvidenceRefs...), WaiverRef: derefString(gate.WaiverRef), MissingRefs: epic2MissingForGate(graphRun.Trace, gate.CommonNode.ID)},
		},
		ReleaseEvidence: []Epic2EvidenceReleaseEvidence{
			{Label: "Runtime BOM", Status: releaseEvidenceStatus(graphRun.Trace), ArtifactRefs: append([]string(nil), rc.ArtifactRefs...), RuntimeRefs: []string{"work.local_deterministic@1"}, BOMRefs: []string{ids.factoryRuntime}, RequiredPathRefs: append([]string(nil), graphRun.Trace.EvidenceRefs...), MissingRefs: append([]string(nil), graphRun.Trace.Missing...)},
		},
		MissingProvenance: epic2MissingProvenance(graphRun.Trace),
		ProofOfWorkPacket: &Epic2ProofOfWorkPacket{
			ID:      ids.proofPacket,
			Status:  packetStatus,
			Summary: "Epic 2 local/dry-run proof-of-work packet for a deterministic text artifact.",
			WorkItem: &Epic2ProofOfWorkItem{
				Label:          "Work task",
				Status:         string(taskStatusFromMode(mode)),
				Summary:        task.Title,
				ArtifactRef:    task.ID.Value(),
				EventGraphRefs: []string{egRef(v39.TypeTask, ids.task)},
			},
			RuntimeInvocation: &Epic2ProofOfWorkItem{
				Label:          "Local deterministic runtime",
				Status:         string(runtimeRun.Result.Result.Status),
				Summary:        "write_file and checksum_file completed with network disabled and secrets unavailable.",
				ArtifactRef:    runtimeRun.Result.ID.Value(),
				EventGraphRefs: []string{egRef(v39.TypeRuntimeEnvelope, ids.runtimeEnvelope), egRef(v39.TypeRuntimeResult, ids.runtimeResult)},
			},
			ChangedFiles: []Epic2ProofOfWorkItem{
				{Label: "out.txt", Status: "recorded", Summary: "Deterministic dry-run artifact.", ArtifactRef: ids.artifact, EventGraphRefs: []string{egRef(v39.TypeArtifact, ids.artifact)}},
			},
			TestsRun: []Epic2ProofOfWorkItem{
				{Label: "Trace completeness fixture", Status: status, Summary: "EventGraph path evaluation for the selected fixture.", ArtifactRef: ids.testRun, EventGraphRefs: []string{egRef(v39.TypeTestRun, ids.testRun), egRef(v39.TypeGateResult, ids.gateResult)}},
			},
			CIStatus: &Epic2ProofOfWorkItem{
				Label:          "Local validation",
				Status:         "pending",
				Summary:        "Repository validation is recorded by the PR after make verify runs.",
				ArtifactRef:    "",
				EventGraphRefs: []string{},
			},
			SecurityScanResults: []Epic2ProofOfWorkItem{
				{Label: "Protected side effects", Status: "pass", Summary: "No external runtime, network, secrets, protected repo mutation, Hive, Agent, or Site execution behavior.", ArtifactRef: ids.runtimeEnvelope, EventGraphRefs: []string{egRef(v39.TypeRuntimeEnvelope, ids.runtimeEnvelope)}},
				{Label: "Gate B capability evidence", Status: "not_applicable", Summary: "Fixture has no CapabilityArtifact, USED_CAPABILITY edge, capability source ref, or capability usage logging evidence.", ArtifactRef: "", EventGraphRefs: []string{egRef(v39.TypeFactoryRuntimeVersion, ids.factoryRuntime)}},
			},
			KnownFailures:    epic2KnownFailures(graph, ids, graphRun),
			OperatorDecision: decision.AsProofOfWorkItem(),
			EventGraphRefs:   []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeReleaseCandidate, ids.releaseCandidate), egRef(v39.TypeAuditReport, ids.auditReport)},
		},
	}
	if graphRun.Rejection != nil {
		projection.FailuresRepairs = epic2FailuresRepairs(graph, ids)
	}
	return projection, nil
}

func (d *Epic2EvidenceDecision) AsProofOfWorkItem() *Epic2ProofOfWorkItem {
	if d == nil {
		return nil
	}
	return &Epic2ProofOfWorkItem{
		Label:          d.Kind,
		Status:         d.Status,
		Summary:        d.Reason,
		ArtifactRef:    d.ID,
		EventGraphRefs: append([]string(nil), d.EventGraphRefs()...),
	}
}

func (d *Epic2EvidenceDecision) EventGraphRefs() []string {
	if d == nil || d.ID == "" {
		return nil
	}
	if d.Kind == "certification" {
		return []string{egRef(v39.TypeCertification, d.ID)}
	}
	return []string{egRef(v39.TypeRejection, d.ID)}
}

func epic2ProjectionDecision(graphRun epic2GraphRun) *Epic2EvidenceDecision {
	if graphRun.Certification != nil {
		cert := graphRun.Certification
		return &Epic2EvidenceDecision{Kind: "certification", ID: cert.CommonNode.ID, ActorID: cert.CertifierActorID, Reason: cert.Reason, EvidenceRefs: append([]string(nil), cert.EvidenceRefs...), Status: statusString(cert.CommonNode.Status), CreatedAt: cert.CommonNode.CreatedAt.Format(time.RFC3339)}
	}
	rej := graphRun.Rejection
	return &Epic2EvidenceDecision{Kind: "rejection", ID: rej.CommonNode.ID, ActorID: rej.RejectorActorID, Reason: rej.Reason, EvidenceRefs: append([]string(nil), rej.EvidenceRefs...), Status: statusString(rej.CommonNode.Status), CreatedAt: rej.CommonNode.CreatedAt.Format(time.RFC3339)}
}

func epic2ModeFromDecision(graphRun epic2GraphRun) Epic2ThinFactoryMode {
	if graphRun.Rejection != nil {
		return Epic2ThinFactoryRejected
	}
	return Epic2ThinFactoryCertified
}

func taskStatusFromMode(mode Epic2ThinFactoryMode) TaskStatus {
	if mode == Epic2ThinFactoryRejected {
		return StatusRejected
	}
	return StatusCertified
}

func releaseEvidenceStatus(trace v39.TraceCompletenessGateResult) string {
	if trace.Completed {
		return "pass"
	}
	return "fail"
}

func epic2MissingForGate(trace v39.TraceCompletenessGateResult, gateID string) []string {
	var missing []string
	for _, path := range trace.RequiredPaths {
		if stringIn(gateID, path.NodeIDs) && len(path.Missing) > 0 {
			missing = append(missing, path.Missing...)
		}
	}
	return missing
}

func epic2MissingProvenance(trace v39.TraceCompletenessGateResult) []Epic2EvidenceMissingProvenance {
	var out []Epic2EvidenceMissingProvenance
	for _, path := range trace.RequiredPaths {
		if path.Completed {
			continue
		}
		out = append(out, Epic2EvidenceMissingProvenance{
			PathName:  path.Name,
			NodeIDs:   append([]string(nil), path.NodeIDs...),
			EdgeIDs:   append([]string(nil), path.EdgeIDs...),
			Missing:   append([]string(nil), path.Missing...),
			Completed: path.Completed,
		})
	}
	return out
}

func epic2FailuresRepairs(graph *v39.InMemoryStore, ids epic2FixtureIDs) []Epic2EvidenceFailureRepair {
	record, err := graph.Get(ids.failure)
	if err != nil {
		return nil
	}
	failure := record.(*v39.Failure)
	return []Epic2EvidenceFailureRepair{
		{
			FailureID:    failure.CommonNode.ID,
			FailureClass: failure.FailureClass,
			Severity:     failure.Severity,
			Summary:      failure.Summary,
			TaskID:       derefString(failure.TaskID),
			GateResultID: derefString(failure.GateResultID),
			TestRunID:    derefString(failure.TestRunID),
		},
	}
}

func epic2KnownFailures(graph *v39.InMemoryStore, ids epic2FixtureIDs, graphRun epic2GraphRun) []Epic2ProofOfWorkItem {
	if graphRun.Rejection == nil {
		return nil
	}
	record, err := graph.Get(ids.failure)
	if err != nil {
		return nil
	}
	failure := record.(*v39.Failure)
	return []Epic2ProofOfWorkItem{
		{Label: failure.FailureClass, Status: statusString(failure.CommonNode.Status), Summary: failure.Summary, ArtifactRef: failure.CommonNode.ID, EventGraphRefs: []string{egRef(v39.TypeFailure, failure.CommonNode.ID)}},
	}
}

func epic2CommandLog(logs []RuntimeCommandLog) []string {
	out := make([]string, 0, len(logs))
	for _, log := range logs {
		out = append(out, fmt.Sprintf("%d:%s:%s", log.Index, log.Name, log.Status))
	}
	return out
}

func epic2Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{
		ID:             id,
		Type:           typ,
		CreatedAt:      epic2FixtureTime(),
		CreatedBy:      epic2FixtureActorID,
		Status:         &status,
		IdempotencyKey: "idem_" + id,
		CorrelationID:  "corr_epic2_thin_factory",
	}
}

func epic2Edge(label, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic2_" + label + ":" + from + ":" + typ + ":" + to
	return v39.CommonEdge{
		ID:             id,
		Type:           typ,
		FromID:         from,
		ToID:           to,
		CreatedAt:      createdAt,
		CreatedBy:      epic2FixtureActorID,
		CorrelationID:  "corr_epic2_thin_factory",
		IdempotencyKey: "idem_" + id,
	}
}

func epic2FixtureTime() time.Time {
	t, _ := time.Parse(time.RFC3339, epic2FixtureTimeRFC)
	return t
}

func statusString(status *string) string {
	if status == nil {
		return ""
	}
	return *status
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func strPtr(value string) *string {
	return &value
}

func egRef(typ, id string) string {
	return "eg://" + typ + "/" + id
}
