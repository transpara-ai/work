package work

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Epic8CapabilityMonitoringLocalEvidence Epic8CapabilityMonitoringMode = "local_capability_monitoring_evidence"
)

const (
	epic8FixtureActorID             = "act_epic8_capability_monitor"
	epic8FixtureHumanActorID        = "act_epic8_operator"
	epic8CapabilityReleaseActorID   = "act_epic8_capability_release"
	epic8KnowledgeSourceRef         = "knowledge:dark-factory/v3.9/implementation/epics/epic-08-gate-i-capability-monitoring-runtime-selection/01-work-capability-monitoring-runtime-selection-implementation-authorization-v3.9.md"
	epic8DocsPRRef                  = "transpara-ai/docs#89"
	epic8DocsMergeSHA               = "2a9797faf627ce5b62f8add1968bc5eb9e53a63b"
	epic8DocsReviewedHead           = "fdbfacba4e9804cfbae5818d7860bf9500c17c2d"
	epic8CandidateRuntimeVersion    = "3.9.14-epic8-candidate-canary"
	epic8PostRollbackRuntimeVersion = "3.9.14-epic8-post-rollback-baseline"
)

// Epic8CapabilityMonitoringMode selects the authorized Gate I fixture mode.
type Epic8CapabilityMonitoringMode string

// Epic8CapabilityMonitoringOptions keeps the Gate I fixture local and bounded.
type Epic8CapabilityMonitoringOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic8CapabilityMonitoringMode

	// Negative-test seams. They remove evidence or request a forbidden scope;
	// they do not perform live activation, GitHub mutation, or protected execution.
	OmitMonitoringWindow          bool
	OmitCapabilityVersionEvidence bool
	OmitRollbackTrigger           bool
	OmitOperatorRollbackAuthority bool
	UseGlobalActivationScope      bool
	SkipCandidateReselectionProbe bool
}

// Epic8CapabilityMonitoringRun is the local evidence packet for Gate I.
type Epic8CapabilityMonitoringRun struct {
	Mode                          Epic8CapabilityMonitoringMode
	WorkTask                      Task
	WorkProjection                TaskProjection
	EventGraph                    *v39.InMemoryStore
	FactoryOrderID                string
	RequirementID                 string
	AcceptanceCriterionID         string
	TaskID                        string
	ActorInvocationID             string
	RuntimeEnvelopeID             string
	RuntimeResultID               string
	CapabilityArtifactID          string
	BaselineCapabilityVersionID   string
	CandidateCapabilityVersionID  string
	ActivationPolicyID            string
	PreRollbackRuntimeVersionID   string
	PostRollbackRuntimeVersionID  string
	RollbackRecordID              string
	OperatorAuthorityRequestID    string
	OperatorAuthorityDecisionID   string
	OperatorHumanApprovalID       string
	PromotionAuthorityRequestID   string
	PromotionAuthorityDecisionID  string
	PromotionExecutionReceiptID   string
	KnowledgeReferenceID          string
	MonitoringWindowArtifactID    string
	ProofOfWorkArtifactID         string
	RollbackDecisionArtifactID    string
	TestCaseID                    string
	TestRunID                     string
	GateResultID                  string
	FailureID                     string
	ReleaseCandidateID            string
	CertificationID               string
	RejectionID                   string
	AuditReportID                 string
	TraceCompleteness             v39.TraceCompletenessGateResult
	CapabilityUsagePath           v39.RequiredPath
	KnowledgePath                 v39.RequiredPath
	GateIValidation               Epic8GateIValidation
	Certification                 *v39.Certification
	Rejection                     *v39.Rejection
	AuditReport                   *v39.AuditReport
	Projection                    Epic8CapabilityMonitoringProjection
	LocalArtifacts                Epic8LocalArtifacts
	MonitoringWindow              Epic8MonitoringWindow
	CandidateReselectionBlocked   bool
	CandidateReselectionError     string
	GlobalActivationRejected      bool
	GlobalActivationError         string
	PromotionReceiptLocalEvidence bool
}

type Epic8LocalArtifacts struct {
	Root             string `json:"root"`
	MonitoringWindow string `json:"monitoring_window"`
	ProofOfWork      string `json:"proof_of_work"`
	RollbackDecision string `json:"rollback_decision"`
}

type Epic8CapabilityMonitoringProjection struct {
	GeneratedAt                   string                        `json:"generated_at"`
	Source                        string                        `json:"source"`
	Mode                          Epic8CapabilityMonitoringMode `json:"mode"`
	MonitoringWindow              Epic8MonitoringWindow         `json:"monitoring_window"`
	GateIValidation               Epic8GateIValidation          `json:"gate_i_validation"`
	AuditReport                   Epic8AuditEvidence            `json:"audit_report"`
	ProofOfWorkPacket             Epic8ProofOfWorkPacket        `json:"proof_of_work_packet"`
	CandidateReselectionBlocked   bool                          `json:"candidate_reselection_blocked"`
	CandidateReselectionError     string                        `json:"candidate_reselection_error,omitempty"`
	GlobalActivationRejected      bool                          `json:"global_activation_rejected"`
	GlobalActivationError         string                        `json:"global_activation_error,omitempty"`
	PromotionReceiptLocalEvidence bool                          `json:"promotion_receipt_local_evidence"`
	Errors                        []string                      `json:"errors,omitempty"`
}

type Epic8GateIValidation struct {
	Status  string   `json:"status"`
	Missing []string `json:"missing,omitempty"`
}

type Epic8AuditEvidence struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type Epic8ProofOfWorkPacket struct {
	ID                         string                 `json:"id"`
	Status                     string                 `json:"status"`
	Summary                    string                 `json:"summary"`
	TrialRefs                  []string               `json:"trial_refs"`
	Metrics                    Epic8MonitoringMetrics `json:"metrics"`
	RuntimeSelection           Epic8RuntimeSelection  `json:"runtime_selection"`
	RollbackDecision           Epic8RollbackDecision  `json:"rollback_decision"`
	NoGlobalActivationProof    Epic8ProofOfWorkItem   `json:"no_global_activation_proof"`
	ResidualRisks              []Epic8ProofOfWorkItem `json:"residual_risks"`
	EventGraphRefs             []string               `json:"event_graph_refs"`
	LocalPromotionReceiptScope string                 `json:"local_promotion_receipt_scope"`
}

type Epic8MonitoringWindow struct {
	ID          string                  `json:"id"`
	Status      string                  `json:"status"`
	Runs        []Epic8MonitoringRunRow `json:"runs"`
	Metrics     Epic8MonitoringMetrics  `json:"metrics"`
	EvidenceRef string                  `json:"evidence_ref,omitempty"`
}

type Epic8MonitoringRunRow struct {
	TrialID              string             `json:"trial_id"`
	Status               string             `json:"status"`
	CapabilityVersionRef string             `json:"capability_version_ref"`
	RuntimeVersionRef    string             `json:"runtime_version_ref"`
	Selection            string             `json:"selection"`
	Success              bool               `json:"success"`
	Regression           bool               `json:"regression"`
	RollbackTriggered    bool               `json:"rollback_triggered"`
	EvidenceRefs         []string           `json:"evidence_refs"`
	Metrics              map[string]float64 `json:"metrics"`
}

type Epic8MonitoringMetrics struct {
	MonitoringWindowRuns                 int     `json:"monitoring_window_runs"`
	CandidateAttemptCount                int     `json:"candidate_attempt_count"`
	CandidateSuccessCount                int     `json:"candidate_success_count"`
	CandidateRegressionCount             int     `json:"candidate_regression_count"`
	CandidateSuccessRate                 float64 `json:"candidate_success_rate"`
	RollbackTriggerCount                 int     `json:"rollback_trigger_count"`
	PostRollbackSuccessCount             int     `json:"post_rollback_success_count"`
	SelectedRuntimeVersionBeforeRollback string  `json:"selected_runtime_version_before_rollback"`
	SelectedRuntimeVersionAfterRollback  string  `json:"selected_runtime_version_after_rollback"`
	ActiveCapabilityVersionRef           string  `json:"active_capability_version_ref"`
	RolledBackCapabilityVersionRef       string  `json:"rolled_back_capability_version_ref"`
	RollbackToCapabilityVersionRef       string  `json:"rollback_to_capability_version_ref"`
	OperatorRollbackDecisionRef          string  `json:"operator_rollback_decision_ref"`
}

type Epic8RuntimeSelection struct {
	BeforeRollbackFactoryRuntimeVersion string `json:"before_rollback_factory_runtime_version"`
	AfterRollbackFactoryRuntimeVersion  string `json:"after_rollback_factory_runtime_version"`
	ActiveCapabilityVersionRef          string `json:"active_capability_version_ref"`
	RolledBackCapabilityVersionRef      string `json:"rolled_back_capability_version_ref"`
	RollbackToCapabilityVersionRef      string `json:"rollback_to_capability_version_ref"`
	PostRollbackCandidatePackaged       bool   `json:"post_rollback_candidate_packaged"`
	CandidateReselectionBlocked         bool   `json:"candidate_reselection_blocked"`
}

type Epic8RollbackDecision struct {
	AuthorityRequestID  string   `json:"authority_request_id,omitempty"`
	AuthorityDecisionID string   `json:"authority_decision_id,omitempty"`
	HumanApprovalID     string   `json:"human_approval_id,omitempty"`
	Decision            string   `json:"decision"`
	Scope               []string `json:"scope"`
	Summary             string   `json:"summary"`
}

type Epic8ProofOfWorkItem struct {
	Label       string   `json:"label"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	ArtifactRef string   `json:"artifact_ref,omitempty"`
	Refs        []string `json:"refs,omitempty"`
}

// RunEpic8CapabilityMonitoringRuntimeSelectionTrials executes the authorized Gate I fixture.
func RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts *TaskStore, opts Epic8CapabilityMonitoringOptions) (Epic8CapabilityMonitoringRun, error) {
	if ts == nil {
		return Epic8CapabilityMonitoringRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic8CapabilityMonitoringRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic8CapabilityMonitoringRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic8CapabilityMonitoringRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic8CapabilityMonitoringLocalEvidence
	}
	if opts.Mode != Epic8CapabilityMonitoringLocalEvidence {
		return Epic8CapabilityMonitoringRun{}, fmt.Errorf("unsupported Epic 8 fixture mode %q", opts.Mode)
	}

	ids := epic8IDs()
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 8 Capability Monitoring Runtime Selection Trials",
		Description:            "Run five bounded Gate I local monitoring-window trials with canary runtime selection and governed rollback evidence.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic8_capability_monitoring",
		RiskClass:              "high",
		ExpectedOutputs:        []string{"artifacts/capability-monitoring/window.json", "artifacts/capability-monitoring/proof-of-work.json"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 8 capability monitoring fixture lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic8CapabilityMonitoringRun{}, err
		}
	}

	localArtifacts := epic8LocalArtifacts(opts.WorkingDir)
	monitoringWindow := epic8BuildMonitoringWindow(ids, opts)
	if !opts.OmitMonitoringWindow {
		if err := epic7WriteJSON(localArtifacts.MonitoringWindow, monitoringWindow); err != nil {
			return Epic8CapabilityMonitoringRun{}, err
		}
	}
	if !opts.OmitRollbackTrigger && !opts.OmitOperatorRollbackAuthority {
		if err := epic7WriteJSON(localArtifacts.RollbackDecision, epic8RollbackDecision(ids, opts)); err != nil {
			return Epic8CapabilityMonitoringRun{}, err
		}
	}

	validation := epic8EvaluateGateI(monitoringWindow, opts)
	graph, graphRun, err := epic8RecordEventGraph(ids, opts, monitoringWindow, validation)
	if err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	if graphRun.CandidateReselectionError != "" && !strings.Contains(graphRun.CandidateReselectionError, "rolled back capability version cannot be activated") {
		validation.Status = "fail"
		validation.Missing = appendUniqueStringLocal(validation.Missing, "post-rollback candidate reselection returned unexpected error: "+graphRun.CandidateReselectionError)
	}
	if validation.Status == "pass" && !graphRun.CandidateReselectionBlocked {
		validation.Status = "fail"
		validation.Missing = appendUniqueStringLocal(validation.Missing, "rolled-back candidate was not blocked from post-rollback selection")
	}

	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 8 Gate I capability monitoring evidence attached", causes, opts.ConversationID); err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	if graphRun.FailureID != "" {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{FailureIDs: []string{graphRun.FailureID}}, "Epic 8 negative Gate I fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic8CapabilityMonitoringRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 8 Gate I evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	if validation.Status == "pass" {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 8 capability monitoring runtime selection trials certified", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic8CapabilityMonitoringRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 8 negative Gate I fixture rejected", []string{ids.gateResult, graphRun.FailureID}, causes, opts.ConversationID); err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}

	projection := epic8BuildProjection(ids, monitoringWindow, validation, graphRun)
	if err := epic7WriteJSON(localArtifacts.ProofOfWork, projection.ProofOfWorkPacket); err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic8CapabilityMonitoringRun{}, err
	}
	return Epic8CapabilityMonitoringRun{
		Mode:                          opts.Mode,
		WorkTask:                      task,
		WorkProjection:                workProjection,
		EventGraph:                    graph,
		FactoryOrderID:                ids.factoryOrder,
		RequirementID:                 ids.requirement,
		AcceptanceCriterionID:         ids.acceptanceCriterion,
		TaskID:                        ids.task,
		ActorInvocationID:             ids.actorInvocation,
		RuntimeEnvelopeID:             ids.runtimeEnvelope,
		RuntimeResultID:               ids.runtimeResult,
		CapabilityArtifactID:          ids.capabilityArtifact,
		BaselineCapabilityVersionID:   ids.baselineCapabilityVersion,
		CandidateCapabilityVersionID:  ids.candidateCapabilityVersion,
		ActivationPolicyID:            ids.activationPolicy,
		PreRollbackRuntimeVersionID:   ids.runtimeBeforeRollback,
		PostRollbackRuntimeVersionID:  ids.runtimeAfterRollback,
		RollbackRecordID:              ids.rollbackRecord,
		OperatorAuthorityRequestID:    ids.operatorAuthorityRequest,
		OperatorAuthorityDecisionID:   ids.operatorAuthorityDecision,
		OperatorHumanApprovalID:       ids.operatorHumanApproval,
		PromotionAuthorityRequestID:   ids.promotionAuthorityRequest,
		PromotionAuthorityDecisionID:  ids.promotionAuthorityDecision,
		PromotionExecutionReceiptID:   ids.promotionExecutionReceipt,
		KnowledgeReferenceID:          ids.knowledgeReference,
		MonitoringWindowArtifactID:    ids.monitoringWindowArtifact,
		ProofOfWorkArtifactID:         ids.proofOfWorkArtifact,
		RollbackDecisionArtifactID:    ids.rollbackDecisionArtifact,
		TestCaseID:                    ids.testCase,
		TestRunID:                     ids.testRun,
		GateResultID:                  ids.gateResult,
		FailureID:                     graphRun.FailureID,
		ReleaseCandidateID:            ids.releaseCandidate,
		CertificationID:               epic8CertificationID(graphRun.Certification),
		RejectionID:                   graphRun.RejectionID,
		AuditReportID:                 ids.auditReport,
		TraceCompleteness:             graphRun.Trace,
		CapabilityUsagePath:           graphRun.CapabilityUsagePath,
		KnowledgePath:                 graphRun.KnowledgePath,
		GateIValidation:               validation,
		Certification:                 graphRun.Certification,
		Rejection:                     graphRun.Rejection,
		AuditReport:                   graphRun.AuditReport,
		Projection:                    projection,
		LocalArtifacts:                localArtifacts,
		MonitoringWindow:              monitoringWindow,
		CandidateReselectionBlocked:   graphRun.CandidateReselectionBlocked,
		CandidateReselectionError:     graphRun.CandidateReselectionError,
		GlobalActivationRejected:      graphRun.GlobalActivationRejected,
		GlobalActivationError:         graphRun.GlobalActivationError,
		PromotionReceiptLocalEvidence: graphRun.PromotionReceiptLocalEvidence,
	}, nil
}

func (p Epic8CapabilityMonitoringProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

type epic8FixtureIDs struct {
	factoryOrder               string
	requirement                string
	acceptanceCriterion        string
	task                       string
	actorIdentity              string
	humanActorIdentity         string
	capabilityReleaseIdentity  string
	actorInvocation            string
	runtimeEnvelope            string
	runtimeResult              string
	capabilityArtifact         string
	baselineCapabilityVersion  string
	candidateCapabilityVersion string
	evolutionOrder             string
	evalDataset                string
	optimizationRun            string
	candidateVariant           string
	benchmarkResult            string
	humanReview                string
	activationPolicy           string
	runtimeBeforeRollback      string
	rollbackRecord             string
	runtimeAfterRollback       string
	operatorAuthorityRequest   string
	operatorAuthorityDecision  string
	operatorHumanApproval      string
	promotionAuthorityRequest  string
	promotionAuthorityDecision string
	promotionExecutionReceipt  string
	knowledgeReference         string
	monitoringWindowArtifact   string
	proofOfWorkArtifact        string
	rollbackDecisionArtifact   string
	testCase                   string
	testRun                    string
	gateResult                 string
	failure                    string
	releaseCandidate           string
	certification              string
	rejection                  string
	auditReport                string
	proofPacket                string
}

type epic8GraphRun struct {
	DecisionID                    string
	RejectionID                   string
	FailureID                     string
	Trace                         v39.TraceCompletenessGateResult
	CapabilityUsagePath           v39.RequiredPath
	KnowledgePath                 v39.RequiredPath
	Certification                 *v39.Certification
	Rejection                     *v39.Rejection
	AuditReport                   *v39.AuditReport
	CandidateReselectionBlocked   bool
	CandidateReselectionError     string
	GlobalActivationRejected      bool
	GlobalActivationError         string
	PromotionReceiptLocalEvidence bool
}

func epic8IDs() epic8FixtureIDs {
	return epic8FixtureIDs{
		factoryOrder:               "fo_epic8_capability_monitoring",
		requirement:                "req_epic8_capability_monitoring",
		acceptanceCriterion:        "ac_epic8_capability_monitoring",
		task:                       "tsk_epic8_capability_monitoring",
		actorIdentity:              "actor_epic8_capability_monitor",
		humanActorIdentity:         "actor_epic8_operator",
		capabilityReleaseIdentity:  "actor_epic8_capability_release",
		actorInvocation:            "invoke_epic8_capability_monitoring",
		runtimeEnvelope:            "env_epic8_capability_monitoring",
		runtimeResult:              "result_epic8_capability_monitoring",
		capabilityArtifact:         "cap_art_epic8_issue_pr_proposer",
		baselineCapabilityVersion:  "cap_version_epic8_issue_pr_proposer_baseline",
		candidateCapabilityVersion: "cap_version_epic8_issue_pr_proposer_candidate",
		evolutionOrder:             "evo_epic8_issue_pr_proposer_candidate",
		evalDataset:                "eval_epic8_monitoring_window",
		optimizationRun:            "opt_epic8_issue_pr_proposer_candidate",
		candidateVariant:           "cand_epic8_issue_pr_proposer_candidate",
		benchmarkResult:            "bench_epic8_issue_pr_proposer_candidate",
		humanReview:                "review_epic8_issue_pr_proposer_candidate",
		activationPolicy:           "activation_epic8_candidate_canary",
		runtimeBeforeRollback:      "frv_epic8_candidate_canary",
		rollbackRecord:             "rollback_epic8_candidate_regression",
		runtimeAfterRollback:       "frv_epic8_post_rollback_baseline",
		operatorAuthorityRequest:   "auth_req_epic8_operator_rollback",
		operatorAuthorityDecision:  "auth_dec_epic8_operator_rollback",
		operatorHumanApproval:      "human_approval_epic8_operator_rollback",
		promotionAuthorityRequest:  "auth_req_epic8_capability_promotion",
		promotionAuthorityDecision: "auth_dec_epic8_capability_promotion",
		promotionExecutionReceipt:  "exec_epic8_capability_promotion_local_evidence",
		knowledgeReference:         "know_epic8_authorization_packet",
		monitoringWindowArtifact:   "art_epic8_monitoring_window",
		proofOfWorkArtifact:        "art_epic8_proof_of_work",
		rollbackDecisionArtifact:   "art_epic8_rollback_decision",
		testCase:                   "tc_epic8_capability_monitoring",
		testRun:                    "tr_epic8_capability_monitoring",
		gateResult:                 "gate_epic8_capability_monitoring",
		failure:                    "fail_epic8_capability_monitoring",
		releaseCandidate:           "rc_epic8_capability_monitoring",
		certification:              "cert_epic8_capability_monitoring",
		rejection:                  "reject_epic8_capability_monitoring",
		auditReport:                "audit_epic8_capability_monitoring",
		proofPacket:                "proof_epic8_capability_monitoring",
	}
}

func epic8LocalArtifacts(dir string) Epic8LocalArtifacts {
	root := filepath.Join(dir, "artifacts", "capability-monitoring")
	return Epic8LocalArtifacts{
		Root:             root,
		MonitoringWindow: filepath.Join(root, "window.json"),
		ProofOfWork:      filepath.Join(root, "proof-of-work.json"),
		RollbackDecision: filepath.Join(root, "rollback-decision.json"),
	}
}

func epic8BuildMonitoringWindow(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions) Epic8MonitoringWindow {
	runs := []Epic8MonitoringRunRow{
		epic8CandidateRun("trial_1_docs_only_proposal_candidate_success", ids, true, false, false),
		epic8CandidateRun("trial_2_bounded_code_change_candidate_success", ids, true, false, false),
		epic8CandidateRun("trial_3_bug_fix_repair_candidate_success", ids, true, false, false),
		epic8CandidateRun("trial_4_intentional_regression_triggers_rollback", ids, false, true, !opts.OmitRollbackTrigger),
		{
			TrialID:              "trial_5_post_rollback_baseline_selection_success",
			Status:               "pass",
			CapabilityVersionRef: ids.baselineCapabilityVersion,
			RuntimeVersionRef:    ids.runtimeAfterRollback,
			Selection:            "baseline_after_candidate_rollback",
			Success:              true,
			Regression:           false,
			RollbackTriggered:    false,
			EvidenceRefs:         []string{ids.runtimeAfterRollback, ids.rollbackRecord},
			Metrics:              map[string]float64{"success": 1, "regression": 0},
		},
	}
	if opts.OmitCapabilityVersionEvidence {
		for index := range runs {
			runs[index].CapabilityVersionRef = ""
			if strings.Contains(runs[index].Selection, "candidate") {
				runs[index].Selection = "candidate_selection_missing_capability_version_evidence"
			}
		}
	}
	metrics := epic8Metrics(ids, runs, opts)
	status := "pass"
	if opts.OmitMonitoringWindow {
		status = "missing"
	}
	return Epic8MonitoringWindow{ID: "window_epic8_capability_monitoring", Status: status, Runs: runs, Metrics: metrics, EvidenceRef: ids.monitoringWindowArtifact}
}

func epic8CandidateRun(id string, ids epic8FixtureIDs, success, regression, rollback bool) Epic8MonitoringRunRow {
	status := "pass"
	if regression {
		status = "regression"
	}
	return Epic8MonitoringRunRow{
		TrialID:              id,
		Status:               status,
		CapabilityVersionRef: ids.candidateCapabilityVersion,
		RuntimeVersionRef:    ids.runtimeBeforeRollback,
		Selection:            "candidate_canary",
		Success:              success,
		Regression:           regression,
		RollbackTriggered:    rollback,
		EvidenceRefs:         []string{ids.runtimeBeforeRollback, ids.activationPolicy},
		Metrics:              map[string]float64{"success": boolMetric(success), "regression": boolMetric(regression)},
	}
}

func epic8Metrics(ids epic8FixtureIDs, runs []Epic8MonitoringRunRow, opts Epic8CapabilityMonitoringOptions) Epic8MonitoringMetrics {
	var candidateAttempts, candidateSuccesses, candidateRegressions, rollbackTriggers, postRollbackSuccesses int
	for _, run := range runs {
		if run.CapabilityVersionRef == ids.candidateCapabilityVersion && run.TrialID != "trial_5_post_rollback_baseline_selection_success" {
			candidateAttempts++
			if run.Success {
				candidateSuccesses++
			}
			if run.Regression {
				candidateRegressions++
			}
		}
		if run.RollbackTriggered {
			rollbackTriggers++
		}
		if run.TrialID == "trial_5_post_rollback_baseline_selection_success" && run.Success {
			postRollbackSuccesses++
		}
	}
	successRate := 0.0
	if candidateAttempts > 0 {
		successRate = float64(candidateSuccesses) / float64(candidateAttempts)
	}
	activeRef := ids.candidateCapabilityVersion
	rolledBackRef := ids.candidateCapabilityVersion
	rollbackToRef := ids.baselineCapabilityVersion
	operatorDecision := ids.operatorAuthorityDecision
	if opts.OmitCapabilityVersionEvidence {
		activeRef = ""
		rolledBackRef = ""
		rollbackToRef = ""
	}
	if opts.OmitOperatorRollbackAuthority {
		operatorDecision = ""
	}
	return Epic8MonitoringMetrics{
		MonitoringWindowRuns:                 len(runs),
		CandidateAttemptCount:                candidateAttempts,
		CandidateSuccessCount:                candidateSuccesses,
		CandidateRegressionCount:             candidateRegressions,
		CandidateSuccessRate:                 successRate,
		RollbackTriggerCount:                 rollbackTriggers,
		PostRollbackSuccessCount:             postRollbackSuccesses,
		SelectedRuntimeVersionBeforeRollback: epic8CandidateRuntimeVersion,
		SelectedRuntimeVersionAfterRollback:  epic8PostRollbackRuntimeVersion,
		ActiveCapabilityVersionRef:           activeRef,
		RolledBackCapabilityVersionRef:       rolledBackRef,
		RollbackToCapabilityVersionRef:       rollbackToRef,
		OperatorRollbackDecisionRef:          operatorDecision,
	}
}

func epic8EvaluateGateI(window Epic8MonitoringWindow, opts Epic8CapabilityMonitoringOptions) Epic8GateIValidation {
	var missing []string
	metrics := window.Metrics
	if opts.OmitMonitoringWindow {
		missing = append(missing, "monitoring window artifact missing")
	}
	if metrics.MonitoringWindowRuns != 5 {
		missing = append(missing, "monitoring_window_runs must equal 5")
	}
	if metrics.CandidateSuccessCount != 3 {
		missing = append(missing, "candidate_success_count must equal 3")
	}
	if metrics.CandidateRegressionCount != 1 {
		missing = append(missing, "candidate_regression_count must equal 1")
	}
	if metrics.CandidateSuccessRate != 0.75 {
		missing = append(missing, "candidate_success_rate must equal 0.75")
	}
	if metrics.RollbackTriggerCount != 1 || opts.OmitRollbackTrigger {
		missing = append(missing, "rollback trigger evidence missing")
	}
	if metrics.PostRollbackSuccessCount != 1 {
		missing = append(missing, "post-rollback baseline success evidence missing")
	}
	if opts.OmitCapabilityVersionEvidence || metrics.ActiveCapabilityVersionRef == "" {
		missing = append(missing, "candidate CapabilityVersion promotion evidence missing")
	}
	if opts.OmitOperatorRollbackAuthority || metrics.OperatorRollbackDecisionRef == "" {
		missing = append(missing, "operator rollback authority missing")
	}
	if opts.UseGlobalActivationScope {
		missing = append(missing, "ActivationPolicy scope=global is forbidden")
	}
	if opts.SkipCandidateReselectionProbe {
		missing = append(missing, "post-rollback candidate reselection probe missing")
	}
	if len(missing) > 0 {
		return Epic8GateIValidation{Status: "fail", Missing: missing}
	}
	return Epic8GateIValidation{Status: "pass"}
}

func epic8RecordEventGraph(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions, window Epic8MonitoringWindow, validation Epic8GateIValidation) (*v39.InMemoryStore, epic8GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic8FixtureTime()
	taskStatus := "certified"
	testRunStatus := "pass"
	runtimeStatus := "succeeded"
	releaseStatus := "certified"
	acceptanceStatus := "verified"
	if validation.Status != "pass" {
		taskStatus = "rejected"
		testRunStatus = "fail"
		runtimeStatus = "failed"
		releaseStatus = "rejected"
		acceptanceStatus = "rejected"
	}

	artifactRefs := epic8ArtifactRefs(ids, opts)
	taskCommon := epic8Common(ids.task, v39.TypeTask, taskStatus)
	taskCommon.SourceRefs = []string{ids.capabilityArtifact, epic8KnowledgeSourceRef}
	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: epic8Common(ids.factoryOrder, v39.TypeFactoryOrder, taskStatus), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-89-merged-" + epic8DocsMergeSHA, SourceIntentRef: epic8DocsPRRef, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic8Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Prove bounded capability monitoring and runtime selection with a local five-run window, canary activation, governed rollback, and no global activation.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: epic8Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, acceptanceStatus), RequirementID: ids.requirement, Text: "Gate I passes only when local monitoring evidence yields three candidate successes, one regression-triggered rollback, one post-rollback baseline success, active CapabilityVersion runtime selection before rollback, and blocked candidate reselection after rollback.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "capability_monitoring_runtime_selection_trace", OwnerRole: "maintainer", RiskClass: "high"},
		&v39.Task{CommonNode: taskCommon, FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic8_capability_monitoring", State: taskStatus, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic8Common(ids.actorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic8FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic8Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic8FixtureHumanActorID, ActorType: "human", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic8Common(ids.capabilityReleaseIdentity, v39.TypeActorIdentity, "active"), ActorID: epic8CapabilityReleaseActorID, ActorType: "service", IdentityMode: "fixture"},
		&v39.CapabilityArtifact{CommonNode: epic8Common(ids.capabilityArtifact, v39.TypeCapabilityArtifact, "active"), ArtifactID: ids.capabilityArtifact, ArtifactType: "skill", Name: "Epic 8 monitored issue-to-PR proposer", ArtifactVersion: "v1", SourceRepoOrOrigin: "transpara-ai/work", ContentHash: epic7Hash("epic8:" + strings.Join(epic8TrialIDs(window.Runs), "\n")), Owner: "work", RiskClass: "medium", ActivationScope: "canary", EvalRefs: []string{ids.testCase}, HumanReviewRef: ids.humanReview, RollbackRef: ids.baselineCapabilityVersion, UsageLoggingRequired: true},
		&v39.CapabilityVersion{CommonNode: epic8Common(ids.baselineCapabilityVersion, v39.TypeCapabilityVersion, "approved"), CapabilityArtifactID: ids.capabilityArtifact, CapabilitySemver: "1.0.0"},
		&v39.ActorInvocation{CommonNode: epic8Common(ids.actorInvocation, v39.TypeActorInvocation, runtimeStatus), TaskID: ids.task, Runtime: "local", ActorID: epic8FixtureActorID, InputContractHash: epic7Hash("epic8-input:" + strings.Join(epic8TrialIDs(window.Runs), ":")), OutputContractHash: strPtr(epic7Hash("epic8-output:" + strings.Join(artifactRefs, ":")))},
		&v39.RuntimeEnvelope{CommonNode: epic8Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_capability_monitoring_fixture", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.runtimeAfterRollback, TaskID: ids.task, ActorID: epic8FixtureActorID, AuthorityDecisionRef: "human_authorized_in_chat_2026-06-02_docs_main_" + epic7ShortSHA(epic8DocsMergeSHA), AllowedFiles: []string{"artifacts/capability-monitoring/**"}, DeniedFiles: []string{".git", "../", ".env", "secrets.env"}, AllowedCommands: []string{"write_monitoring_window", "write_rollback_decision", "write_proof_packet"}, DeniedCommands: []string{"gh pr create", "git push", "git merge", "gh pr merge", "deploy", "protected_execution.run", "capability.activate.global"}, NetworkPolicy: "disabled", SecretsPolicy: "none", WorkingDirectory: opts.WorkingDir, Timeout: "1s", ResourceLimits: map[string]any{"max_live_prs_created": 0, "max_branch_pushes": 0, "max_production_mutations": 0, "max_global_activations": 0}, ExpectedOutputs: []string{"artifacts/capability-monitoring/window.json", "artifacts/capability-monitoring/proof-of-work.json"}, OutputContract: map[string]any{"mode": string(opts.Mode), "gate": "gate_i_capability_monitoring"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"epic8EvaluateGateI", "go test ./... -run Epic8", "make verify"}, EnvelopeHash: epic7Hash("epic8-envelope:" + string(opts.Mode))},
		&v39.RuntimeResult{CommonNode: epic8Common(ids.runtimeResult, v39.TypeRuntimeResult, runtimeStatus), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_capability_monitoring_fixture", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: runtimeStatus, ArtifactRefs: artifactRefs, ChangedFiles: []string{}, CommandLog: epic8CommandLog(window, opts), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{"no_global_activation_boundary", "no_policy_engine_adapter_decision"}, PostRunValidationRefs: []string{ids.testRun}},
		&v39.TestCase{CommonNode: epic8Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 8 capability monitoring Gate I evidence", TestType: "unit", Path: strPtr("work/epic8_capability_monitoring_test.go")},
		&v39.TestRun{CommonNode: epic8Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./... -run Epic8"},
		&v39.GateResult{CommonNode: epic8Common(ids.gateResult, v39.TypeGateResult, validation.Status), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_i_capability_monitoring_runtime_selection", EvidenceRefs: append([]string{ids.testRun}, artifactRefs...)},
	}
	records = append(records, epic8MonitoringArtifactRecords(ids, opts, window)...)
	if !opts.OmitCapabilityVersionEvidence {
		records = append(records, epic8CapabilityPromotionEvidenceRecords(ids)...)
	}
	if !opts.OmitOperatorRollbackAuthority {
		records = append(records, epic8OperatorRollbackAuthorityRecords(ids)...)
	}
	if validation.Status == "fail" {
		records = append(records, &v39.Failure{CommonNode: epic8Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "gate_i_capability_monitoring_blocked", Severity: "high", Summary: strings.Join(validation.Missing, "; ")})
	}
	if err := epic7AppendRecords(graph, records...); err != nil {
		return nil, epic8GraphRun{}, err
	}
	if err := epic8AppendBaseEdges(graph, ids, opts, artifactRefs, validation.Status == "fail"); err != nil {
		return nil, epic8GraphRun{}, err
	}
	if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{CommonNode: epic8Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded"), ReferenceCreatedAt: createdAt, SourceSystem: "transpara-ai/docs", SourceRef: epic8KnowledgeSourceRef, SourceHashOrImmutableLocator: "sha256:docs-pr-89-merged-" + epic8DocsMergeSHA + "-reviewed-head-" + epic8DocsReviewedHead, RetrievedAt: createdAt, UsedByActor: epic8FixtureActorID, UsedInTask: ids.task, InfluenceSummary: "Gate I authorization packet constrained the local monitoring window, candidate CapabilityVersion evidence, rollback authority, no-global-activation boundary, residual risks, and Gate J stop condition.", RiskScope: "high", TrustLevel: "human_authorized", FreshnessStatus: "current", RedactionState: "none"}}); err != nil {
		return nil, epic8GraphRun{}, err
	}
	if _, err := graph.RecordCapabilityUsage(ids.task, ids.capabilityArtifact, epic8Common("edge_epic8_used_capability", v39.TypeCapabilityArtifact, "recorded")); err != nil {
		return nil, epic8GraphRun{}, err
	}

	promotionReceiptLocalEvidence := false
	if !opts.OmitCapabilityVersionEvidence {
		version := epic8CandidateCapabilityVersion(ids)
		if _, err := graph.PromoteCapabilityVersion(version); err != nil {
			return nil, epic8GraphRun{}, err
		}
		promotionReceiptLocalEvidence = true
	}

	var globalActivationRejected bool
	var globalActivationError string
	if !opts.OmitCapabilityVersionEvidence {
		policy := epic8ActivationPolicy(ids, opts)
		runtime := &v39.FactoryRuntimeVersion{CommonNode: epic8Common(ids.runtimeBeforeRollback, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: epic8CandidateRuntimeVersion, CapabilityVersionRefs: []string{ids.candidateCapabilityVersion}, RuntimeRefs: []string{"work.local_issue_pr_proposer_candidate@1"}}
		if opts.UseGlobalActivationScope {
			_, _, err := graph.ActivateCapabilityVersion(policy, runtime)
			globalActivationRejected = err != nil
			if err != nil {
				globalActivationError = err.Error()
			}
		} else if _, _, err := graph.ActivateCapabilityVersion(policy, runtime); err != nil {
			return nil, epic8GraphRun{}, err
		}
	}

	if !opts.OmitCapabilityVersionEvidence && !opts.UseGlobalActivationScope && !opts.OmitRollbackTrigger {
		if _, err := graph.RecordRollbackRecord(&v39.RollbackRecord{CommonNode: epic8Common(ids.rollbackRecord, v39.TypeRollbackRecord, "completed"), CapabilityVersionID: ids.candidateCapabilityVersion, RollbackTo: ids.baselineCapabilityVersion, Trigger: "benchmark_regression", ActorID: epic8FixtureHumanActorID, FactoryRuntimeVersionID: ids.runtimeBeforeRollback}); err != nil {
			return nil, epic8GraphRun{}, err
		}
	}

	runtimeAfterRefs := []string{"work.local_issue_pr_proposer_baseline@1"}
	runtimeAfterCapabilityRefs := []string{}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic8Common(ids.runtimeAfterRollback, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: epic8PostRollbackRuntimeVersion, CapabilityVersionRefs: runtimeAfterCapabilityRefs, RuntimeRefs: runtimeAfterRefs}); err != nil {
		return nil, epic8GraphRun{}, err
	}

	var candidateBlocked bool
	var candidateBlockErr string
	if !opts.OmitCapabilityVersionEvidence && !opts.UseGlobalActivationScope && !opts.OmitRollbackTrigger && !opts.SkipCandidateReselectionProbe {
		reselectPolicy := epic8ActivationPolicy(ids, opts)
		reselectPolicy.CommonNode.ID = "activation_epic8_candidate_reselection_probe"
		reselectPolicy.ActivationPolicyID = reselectPolicy.CommonNode.ID
		reselectRuntime := &v39.FactoryRuntimeVersion{CommonNode: epic8Common("frv_epic8_candidate_reselection_probe", v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.14-epic8-candidate-reselection-probe", CapabilityVersionRefs: []string{ids.candidateCapabilityVersion}, RuntimeRefs: []string{"work.local_issue_pr_proposer_candidate@1"}}
		_, _, err := graph.ActivateCapabilityVersion(reselectPolicy, reselectRuntime)
		candidateBlocked = err != nil
		if err != nil {
			candidateBlockErr = err.Error()
		}
	}

	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: epic8Common(ids.releaseCandidate, v39.TypeReleaseCandidate, releaseStatus), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.runtimeAfterRollback, ArtifactRefs: artifactRefs})
	if err != nil {
		return nil, epic8GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	capabilityPath, _ := graph.CapabilityUsageEvidencePath(rc.CommonNode.ID)
	knowledgePath, _ := graph.AdvisoryReferenceEvidencePath(rc.CommonNode.ID)
	if validation.Status == "pass" && traceErr != nil {
		return nil, epic8GraphRun{}, traceErr
	}
	if validation.Status == "pass" {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic8Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: epic8FixtureHumanActorID, Reason: "Gate I capability monitoring evidence is complete for the bounded local Work fixture.", EvidenceRefs: []string{ids.gateResult, ids.testRun, ids.rollbackRecord}})
		if err != nil {
			return nil, epic8GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic8Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic8GraphRun{}, err
		}
		return graph, epic8GraphRun{DecisionID: cert.CommonNode.ID, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Certification: cert, AuditReport: audit, CandidateReselectionBlocked: candidateBlocked, CandidateReselectionError: candidateBlockErr, GlobalActivationRejected: globalActivationRejected, GlobalActivationError: globalActivationError, PromotionReceiptLocalEvidence: promotionReceiptLocalEvidence}, nil
	}
	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic8Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: epic8FixtureHumanActorID, Reason: "Gate I capability monitoring evidence is incomplete or unsafe: " + strings.Join(validation.Missing, "; "), EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic8GraphRun{}, err
	}
	audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic8Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
	if err != nil {
		return nil, epic8GraphRun{}, err
	}
	return graph, epic8GraphRun{DecisionID: rejection.CommonNode.ID, RejectionID: rejection.CommonNode.ID, FailureID: ids.failure, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Rejection: rejection, AuditReport: audit, CandidateReselectionBlocked: candidateBlocked, CandidateReselectionError: candidateBlockErr, GlobalActivationRejected: globalActivationRejected, GlobalActivationError: globalActivationError, PromotionReceiptLocalEvidence: promotionReceiptLocalEvidence}, nil
}

func epic8CapabilityPromotionEvidenceRecords(ids epic8FixtureIDs) []v39.Record {
	reviewCommon := epic8Common(ids.humanReview, v39.TypeHumanReview, "approved")
	reviewCommon.SourceRefs = []string{ids.candidateVariant, ids.benchmarkResult}
	return []v39.Record{
		&v39.EvolutionOrder{CommonNode: epic8Common(ids.evolutionOrder, v39.TypeEvolutionOrder, "accepted"), EvolutionOrderVersion: 1, TargetCapabilityType: "skill", TargetRepo: "transpara-ai/work", TargetPath: "epic7_issue_pr_autonomy.go", RiskClass: "medium", Motivation: "select the monitored issue-to-PR proposal capability only after local monitoring evidence", EvalSource: "local five-run Gate I monitoring window", Constraints: []string{"no_global_activation", "local_fixture_only", "rollback_target_required"}, ReviewRequirements: []string{"CapabilityReviewer approval", "operator rollback authority"}},
		&v39.EvalDataset{CommonNode: epic8Common(ids.evalDataset, v39.TypeEvalDataset, "active"), SourceType: "benchmark", TrustLevel: "reviewed", TrainCount: 0, ValidationCount: 5, HoldoutCount: 0},
		&v39.OptimizationRun{CommonNode: epic8Common(ids.optimizationRun, v39.TypeOptimizationRun, "succeeded"), EvolutionOrderID: ids.evolutionOrder, EvalDatasetID: ids.evalDataset, Engine: "manual"},
		&v39.CandidateVariant{CommonNode: epic8Common(ids.candidateVariant, v39.TypeCandidateVariant, "approved"), OptimizationRunID: ids.optimizationRun, CapabilityArtifactID: ids.capabilityArtifact},
		&v39.BenchmarkResult{CommonNode: epic8Common(ids.benchmarkResult, v39.TypeBenchmarkResult, "pass"), CandidateVariantID: ids.candidateVariant, BaselineRef: ids.baselineCapabilityVersion, MetricDeltas: map[string]float64{"candidate_success_rate": 0.75, "candidate_regression_count": 1, "post_rollback_success_count": 1}},
		&v39.HumanReview{CommonNode: reviewCommon, ReviewerActorID: epic8FixtureHumanActorID, ReviewerRole: "CapabilityReviewer", Rationale: "Local candidate may be canary-selected only with rollback target, five-run monitoring evidence, and no global activation."},
		&v39.AuthorityRequest{CommonNode: epic8Common(ids.promotionAuthorityRequest, v39.TypeAuthorityRequest, "recorded"), ActorID: epic8CapabilityReleaseActorID, ActorRole: v39.CapabilityReleaseRole, Action: v39.CapabilityPromotionAction, TargetType: v39.TypeCapabilityVersion, TargetID: ids.candidateCapabilityVersion, RiskClass: "medium", Reason: "Authorize side-effect-free local capability promotion evidence for Gate I runtime selection.", ProposedCommand: strPtr("record local capability.promote evidence"), EvidenceRefs: []string{ids.benchmarkResult, ids.humanReview}},
		&v39.AuthorityDecision{CommonNode: epic8Common(ids.promotionAuthorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.promotionAuthorityRequest, DeciderActorID: epic8FixtureHumanActorID, DeciderRole: "maintainer", Decision: "ApprovalRequired", Reason: "Approve local capability promotion evidence only; this is not protected execution or production deployment.", Scope: []string{v39.CapabilityPromotionAction, ids.candidateCapabilityVersion}, Conditions: []string{"local evidence only", "no production execution", "no global activation", "rollback target required"}},
		&v39.ExecutionReceipt{CommonNode: epic8Common(ids.promotionExecutionReceipt, v39.TypeExecutionReceipt, "recorded"), AuthorityDecisionID: ids.promotionAuthorityDecision, ActorInvocationID: nil, Action: v39.CapabilityPromotionAction, TargetID: ids.candidateCapabilityVersion, Result: "succeeded", EvidenceRefs: []string{ids.benchmarkResult, ids.humanReview, ids.promotionAuthorityDecision}},
	}
}

func epic8CandidateCapabilityVersion(ids epic8FixtureIDs) *v39.CapabilityVersion {
	return &v39.CapabilityVersion{CommonNode: epic8Common(ids.candidateCapabilityVersion, v39.TypeCapabilityVersion, "approved"), CapabilityArtifactID: ids.capabilityArtifact, EvolutionOrderID: ids.evolutionOrder, OptimizationRunID: ids.optimizationRun, CandidateVariantID: ids.candidateVariant, EvalDatasetID: ids.evalDataset, BenchmarkResultID: ids.benchmarkResult, HumanReviewID: ids.humanReview, PromoterActorID: epic8CapabilityReleaseActorID, PromoterRole: v39.CapabilityReleaseRole, CapabilitySemver: "1.1.0", RollbackTo: strPtr(ids.baselineCapabilityVersion)}
}

func epic8ActivationPolicy(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions) *v39.ActivationPolicy {
	scope := "canary"
	if opts.UseGlobalActivationScope {
		scope = "global"
	}
	canaryPercent := 20.0
	return &v39.ActivationPolicy{CommonNode: epic8Common(ids.activationPolicy, v39.TypeActivationPolicy, "approved"), ActivationPolicyID: ids.activationPolicy, CapabilityVersionID: ids.candidateCapabilityVersion, Scope: scope, AllowedProjects: []string{"dark-factory"}, AllowedFactoryOrders: []string{ids.factoryOrder}, CanaryPercent: &canaryPercent, MonitoringWindowRuns: 5, RollbackTriggers: []string{"benchmark_regression"}, ApprovedBy: []string{epic8FixtureHumanActorID}}
}

func epic8OperatorRollbackAuthorityRecords(ids epic8FixtureIDs) []v39.Record {
	return []v39.Record{
		&v39.AuthorityRequest{CommonNode: epic8Common(ids.operatorAuthorityRequest, v39.TypeAuthorityRequest, "recorded"), ActorID: epic8FixtureActorID, ActorRole: "local_capability_monitor", Action: "capability.rollback", TargetType: v39.TypeCapabilityVersion, TargetID: ids.candidateCapabilityVersion, RiskClass: "high", Reason: "Regression in Gate I monitoring window requires rollback to baseline candidate selection.", ProposedCommand: strPtr("record local rollback decision"), EvidenceRefs: []string{ids.monitoringWindowArtifact, ids.benchmarkResult}},
		&v39.AuthorityDecision{CommonNode: epic8Common(ids.operatorAuthorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.operatorAuthorityRequest, DeciderActorID: epic8FixtureHumanActorID, DeciderRole: "maintainer", Decision: "ApprovalRequired", Reason: "Approve rollback to baseline after regression; no global activation or production mutation is allowed.", Scope: []string{"capability.rollback", ids.candidateCapabilityVersion, ids.baselineCapabilityVersion}, Conditions: []string{"local rollback evidence only", "candidate must not be selected after rollback", "Gate J remains waiting"}},
		&v39.HumanApproval{CommonNode: epic8Common(ids.operatorHumanApproval, v39.TypeHumanApproval, "approved"), RequestRef: ids.operatorAuthorityRequest, ApproverActorID: epic8FixtureHumanActorID, ApproverRole: "maintainer", Decision: "approved", Reason: "Operator rollback authority is granted for the local Gate I fixture after regression evidence."},
	}
}

func epic8MonitoringArtifactRecords(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions, window Epic8MonitoringWindow) []v39.Record {
	var records []v39.Record
	if !opts.OmitMonitoringWindow {
		path := "artifacts/capability-monitoring/window.json"
		records = append(records, &v39.Artifact{CommonNode: epic8Common(ids.monitoringWindowArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: &path, ContentHash: strPtr(epic7Hash(strings.Join(epic8TrialIDs(window.Runs), ":")))})
	}
	proofPath := "artifacts/capability-monitoring/proof-of-work.json"
	records = append(records, &v39.Artifact{CommonNode: epic8Common(ids.proofOfWorkArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: &proofPath, ContentHash: strPtr(epic7Hash("proof:" + validationHashStatus(window.Status)))})
	if !opts.OmitRollbackTrigger && !opts.OmitOperatorRollbackAuthority {
		rollbackPath := "artifacts/capability-monitoring/rollback-decision.json"
		records = append(records, &v39.Artifact{CommonNode: epic8Common(ids.rollbackDecisionArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: &rollbackPath, ContentHash: strPtr(epic7Hash(ids.operatorAuthorityDecision + ":" + ids.rollbackRecord))})
	}
	return records
}

func epic8ArtifactRefs(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions) []string {
	var refs []string
	if !opts.OmitMonitoringWindow {
		refs = append(refs, ids.monitoringWindowArtifact)
	}
	refs = append(refs, ids.proofOfWorkArtifact)
	if !opts.OmitRollbackTrigger && !opts.OmitOperatorRollbackAuthority {
		refs = append(refs, ids.rollbackDecisionArtifact)
	}
	return refs
}

func epic8AppendBaseEdges(graph *v39.InMemoryStore, ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions, artifactRefs []string, includeFailure bool) error {
	createdAt := epic8FixtureTime()
	edges := []v39.CommonEdge{
		epic8Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic8Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic8Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic8Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic8Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic8Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic8Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic8Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic8Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	for _, artifact := range artifactRefs {
		edges = append(edges, epic8Edge("task_"+artifact, v39.EdgeProduced, ids.task, artifact, createdAt))
	}
	if !opts.OmitCapabilityVersionEvidence {
		edges = append(edges,
			epic8Edge("release_auth_req", v39.EdgeRequestedAuthority, ids.capabilityReleaseIdentity, ids.promotionAuthorityRequest, createdAt),
			epic8Edge("promotion_decision", v39.EdgeDecidedBy, ids.promotionAuthorityRequest, ids.promotionAuthorityDecision, createdAt),
			epic8Edge("promotion_receipt", v39.EdgeReceiptedBy, ids.promotionAuthorityDecision, ids.promotionExecutionReceipt, createdAt),
		)
	}
	if !opts.OmitOperatorRollbackAuthority {
		edges = append(edges,
			epic8Edge("operator_auth_req", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.operatorAuthorityRequest, createdAt),
			epic8Edge("operator_decision", v39.EdgeDecidedBy, ids.operatorAuthorityRequest, ids.operatorAuthorityDecision, createdAt),
			epic8Edge("operator_human", v39.EdgeApprovedBy, ids.operatorAuthorityRequest, ids.operatorHumanApproval, createdAt),
		)
	}
	if includeFailure {
		edges = append(edges, epic8Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic8BuildProjection(ids epic8FixtureIDs, window Epic8MonitoringWindow, validation Epic8GateIValidation, graphRun epic8GraphRun) Epic8CapabilityMonitoringProjection {
	auditEvidence := Epic8AuditEvidence{}
	if graphRun.AuditReport != nil {
		auditEvidence = Epic8AuditEvidence{ID: graphRun.AuditReport.CommonNode.ID, TargetType: graphRun.AuditReport.TargetType, TargetID: graphRun.AuditReport.TargetID, Status: statusString(graphRun.AuditReport.CommonNode.Status), TraceScore: graphRun.AuditReport.TraceScore, MissingLinks: append([]string(nil), graphRun.AuditReport.MissingLinks...)}
	}
	projection := Epic8CapabilityMonitoringProjection{
		GeneratedAt:                   epic8FixtureTime().Format(time.RFC3339),
		Source:                        "work-epic8-capability-monitoring-runtime-selection-fixture",
		Mode:                          Epic8CapabilityMonitoringLocalEvidence,
		MonitoringWindow:              window,
		GateIValidation:               validation,
		AuditReport:                   auditEvidence,
		CandidateReselectionBlocked:   graphRun.CandidateReselectionBlocked,
		CandidateReselectionError:     graphRun.CandidateReselectionError,
		GlobalActivationRejected:      graphRun.GlobalActivationRejected,
		GlobalActivationError:         graphRun.GlobalActivationError,
		PromotionReceiptLocalEvidence: graphRun.PromotionReceiptLocalEvidence,
		ProofOfWorkPacket: Epic8ProofOfWorkPacket{
			ID:        ids.proofPacket,
			Status:    validation.Status,
			Summary:   "Epic 8 Gate I aggregate proof: five local monitoring-window runs drive canary runtime selection, regression rollback, and baseline post-rollback selection.",
			TrialRefs: epic8TrialIDs(window.Runs),
			Metrics:   window.Metrics,
			RuntimeSelection: Epic8RuntimeSelection{
				BeforeRollbackFactoryRuntimeVersion: ids.runtimeBeforeRollback,
				AfterRollbackFactoryRuntimeVersion:  ids.runtimeAfterRollback,
				ActiveCapabilityVersionRef:          window.Metrics.ActiveCapabilityVersionRef,
				RolledBackCapabilityVersionRef:      window.Metrics.RolledBackCapabilityVersionRef,
				RollbackToCapabilityVersionRef:      window.Metrics.RollbackToCapabilityVersionRef,
				PostRollbackCandidatePackaged:       false,
				CandidateReselectionBlocked:         graphRun.CandidateReselectionBlocked,
			},
			RollbackDecision:        epic8ProjectionRollbackDecision(ids, window),
			NoGlobalActivationProof: Epic8ProofOfWorkItem{Label: "No global activation", Status: boolStatus(!graphRun.GlobalActivationRejected), Summary: "Positive fixture uses canary activation only; the negative seam proves scope=global is rejected by EventGraph activation helpers.", Refs: []string{ids.activationPolicy}},
			ResidualRisks: []Epic8ProofOfWorkItem{
				{Label: "R-001", Status: "excluded", Summary: "No runner/worktree protected execution, branch push, live PR creation, or production runtime mutation is performed."},
				{Label: "R-002", Status: "excluded", Summary: "No real protected side effects are performed and no ExecutionReceipt production path is claimed; the capability.promote receipt is side-effect-free local EventGraph evidence required by existing promotion helper semantics."},
				{Label: "R-003", Status: "excluded", Summary: "No PolicyEngineAdapterDecision or policy-bundle evidence is used."},
				{Label: "Gate J", Status: "waiting", Summary: "Golden PRD run remains outside this fixture and requires a later selected, merged, reviewed, and explicitly authorized scope."},
			},
			EventGraphRefs:             []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeActivationPolicy, ids.activationPolicy), egRef(v39.TypeRollbackRecord, ids.rollbackRecord), egRef(v39.TypeAuditReport, ids.auditReport)},
			LocalPromotionReceiptScope: "side-effect-free local capability.promote evidence only; not protected execution, deployment, or a production ExecutionReceipt path",
		},
	}
	if validation.Status != "pass" {
		projection.Errors = append([]string(nil), validation.Missing...)
	}
	return projection
}

func epic8RollbackDecision(ids epic8FixtureIDs, opts Epic8CapabilityMonitoringOptions) Epic8RollbackDecision {
	if opts.OmitOperatorRollbackAuthority {
		return Epic8RollbackDecision{Decision: "missing", Summary: "Operator rollback authority omitted by negative seam."}
	}
	return Epic8RollbackDecision{
		AuthorityRequestID:  ids.operatorAuthorityRequest,
		AuthorityDecisionID: ids.operatorAuthorityDecision,
		HumanApprovalID:     ids.operatorHumanApproval,
		Decision:            "approved",
		Scope:               []string{"capability.rollback", ids.candidateCapabilityVersion, ids.baselineCapabilityVersion},
		Summary:             "Regression in trial_4 authorizes local rollback to the baseline/disabled-candidate runtime selection.",
	}
}

func epic8ProjectionRollbackDecision(ids epic8FixtureIDs, window Epic8MonitoringWindow) Epic8RollbackDecision {
	if window.Metrics.RollbackTriggerCount == 0 || window.Metrics.RolledBackCapabilityVersionRef == "" {
		return Epic8RollbackDecision{Decision: "missing", Summary: "Rollback trigger evidence missing from the monitoring-window evidence."}
	}
	if window.Metrics.OperatorRollbackDecisionRef == "" {
		return Epic8RollbackDecision{Decision: "missing", Summary: "Operator rollback authority missing from the monitoring-window evidence."}
	}
	return epic8RollbackDecision(ids, Epic8CapabilityMonitoringOptions{})
}

func epic8CommandLog(window Epic8MonitoringWindow, opts Epic8CapabilityMonitoringOptions) []string {
	log := []string{"0:write_monitoring_window:" + missingStatus(!opts.OmitMonitoringWindow), "1:record_candidate_capability_version:" + missingStatus(!opts.OmitCapabilityVersionEvidence), "2:record_canary_activation:" + missingStatus(!opts.UseGlobalActivationScope), "3:record_rollback_trigger:" + missingStatus(!opts.OmitRollbackTrigger), "4:record_operator_rollback_authority:" + missingStatus(!opts.OmitOperatorRollbackAuthority), "5:probe_candidate_reselection:" + missingStatus(!opts.SkipCandidateReselectionProbe)}
	if opts.UseGlobalActivationScope {
		log = append(log, "ActivationPolicy(scope=global):rejected")
	}
	for _, run := range window.Runs {
		log = append(log, "trial:"+run.TrialID+":"+run.Status)
	}
	log = append(log, "GateJ:waiting", "protected_execution:not_run", "live_pr_creation:not_run", "branch_push:not_run", "deploy:not_run")
	return log
}

func epic8TrialIDs(runs []Epic8MonitoringRunRow) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, run.TrialID)
	}
	return out
}

func epic8Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{ID: id, Type: typ, CreatedAt: epic8FixtureTime(), CreatedBy: epic8FixtureActorID, Status: &status, IdempotencyKey: "idem_" + id, CorrelationID: "corr_epic8_capability_monitoring"}
}

func epic8Edge(suffix, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic8_" + suffix + "_" + from + "_" + to
	return v39.CommonEdge{ID: id, Type: typ, FromID: from, ToID: to, CreatedAt: createdAt, CreatedBy: epic8FixtureActorID, CorrelationID: "corr_epic8_capability_monitoring", IdempotencyKey: "idem_" + id}
}

func epic8FixtureTime() time.Time {
	return time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC)
}

func epic8CertificationID(cert *v39.Certification) string {
	if cert == nil {
		return ""
	}
	return cert.CommonNode.ID
}

func boolMetric(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func missingStatus(ok bool) string {
	if ok {
		return "recorded"
	}
	return "missing"
}

func validationHashStatus(status string) string {
	if status == "" {
		return "unknown"
	}
	return status
}

func appendUniqueStringLocal(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
