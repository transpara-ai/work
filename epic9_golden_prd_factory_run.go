package work

import (
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
	Epic9GoldenPRDLocalDryRun Epic9GoldenPRDMode = "local_golden_prd_dry_run"
)

const (
	epic9FixtureActorID       = "act_epic9_golden_prd_factory"
	epic9FixtureHumanActorID  = "act_epic9_release_reviewer"
	epic9KnowledgeSourceRef   = "knowledge:dark-factory/v3.9/implementation/epics/epic-09-gate-j-golden-prd-product-factory-run/01-work-golden-prd-product-factory-run-implementation-authorization-v3.9.md"
	epic9DocsMergeSHA         = "b5efbba0706bb31b9703a645032a91b1c54a103c"
	epic9DocsReviewedHead     = "0534d4b58860926e03b8000248802f0dc4852675"
	epic9GoldenPRDName        = "simple CRUD tracker"
	epic9GoldenPRDSourceRef   = "dark-factory/v3.9/04-production-workflow-and-runtime-v3.9.md#golden-prds-simple-crud-tracker"
	epic9FactoryRuntime       = "frv_epic9_golden_prd_local_dry_run"
	epic9FactoryRuntimeString = "3.9.16-epic9-golden-prd-local-dry-run"
)

// Epic9GoldenPRDMode selects the authorized Gate J fixture mode.
type Epic9GoldenPRDMode string

// Epic9GoldenPRDOptions keeps the Gate J fixture local, dry-run, and caller-bounded.
type Epic9GoldenPRDOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic9GoldenPRDMode

	// Negative-test seams. These remove or poison local evidence only; they
	// never create live PRs, push branches, deploy, or run protected execution.
	OmitFactoryOrder               bool
	OmitSourceIntent               bool
	OmitAcceptanceEvidence         bool
	OmitGeneratedArtifactEvidence  bool
	OmitSecurityGateEvidence       bool
	AddOpenCriticalSecurityFinding bool
	AddOpenHighSecurityFinding     bool
	AddValidHighWaiver             bool
	OmitFactoryRuntimeVersion      bool
	OmitReleaseAuthority           bool
	OmitAuditReport                bool
}

// Epic9GoldenPRDRun is the local evidence packet for the bounded Gate J run.
type Epic9GoldenPRDRun struct {
	Mode                    Epic9GoldenPRDMode
	WorkTask                Task
	WorkProjection          TaskProjection
	EventGraph              *v39.InMemoryStore
	FactoryOrderID          string
	RequirementID           string
	AcceptanceCriterionID   string
	TaskID                  string
	ActorInvocationID       string
	RuntimeEnvelopeID       string
	RuntimeResultID         string
	CapabilityArtifactID    string
	KnowledgeReferenceID    string
	SourceIntentArtifactID  string
	GeneratedManifestID     string
	SecurityReportID        string
	RuntimeBOMArtifactID    string
	DeployPreviewID         string
	ProofOfWorkArtifactID   string
	TestCaseID              string
	TestRunID               string
	GateResultID            string
	FailureID               string
	FactoryRuntimeVersionID string
	AuthorityRequestID      string
	AuthorityDecisionID     string
	HumanApprovalID         string
	ReleaseCandidateID      string
	CertificationID         string
	RejectionID             string
	AuditReportID           string
	TraceCompleteness       v39.TraceCompletenessGateResult
	CapabilityUsagePath     v39.RequiredPath
	KnowledgePath           v39.RequiredPath
	GateJValidation         Epic9GateJValidation
	Certification           *v39.Certification
	Rejection               *v39.Rejection
	AuditReport             *v39.AuditReport
	Projection              Epic9GoldenPRDProjection
	LocalArtifacts          Epic9LocalArtifacts
	GeneratedManifest       Epic9GeneratedManifest
	SecurityGateReport      Epic9SecurityGateReport
}

type Epic9LocalArtifacts struct {
	Root                 string `json:"root"`
	GeneratedTemplateDir string `json:"generated_template_dir"`
	GeneratedManifest    string `json:"generated_manifest"`
	SecurityGateReport   string `json:"security_gate_report"`
	DeployPreviewDryRun  string `json:"deploy_preview_dry_run"`
	ProofOfWork          string `json:"proof_of_work"`
	AuditReport          string `json:"audit_report"`
}

type Epic9GoldenPRDProjection struct {
	GeneratedAt       string                  `json:"generated_at"`
	Source            string                  `json:"source"`
	Mode              Epic9GoldenPRDMode      `json:"mode"`
	GoldenPRD         Epic9GoldenPRDSource    `json:"golden_prd"`
	GeneratedManifest Epic9GeneratedManifest  `json:"generated_manifest"`
	SecurityGates     Epic9SecurityGateReport `json:"security_gates"`
	GateJValidation   Epic9GateJValidation    `json:"gate_j_validation"`
	AuditReport       Epic9AuditEvidence      `json:"audit_report"`
	ProofOfWorkPacket Epic9ProofOfWorkPacket  `json:"proof_of_work_packet"`
	Errors            []string                `json:"errors,omitempty"`
}

type Epic9GoldenPRDSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourceRef   string `json:"source_ref"`
	LocatorHash string `json:"locator_hash"`
}

type Epic9GeneratedManifest struct {
	TemplateID  string   `json:"template_id"`
	Root        string   `json:"root"`
	Files       []string `json:"files"`
	FileCount   int      `json:"file_count"`
	ContentHash string   `json:"content_hash"`
}

type Epic9SecurityGateReport struct {
	ID                  string                      `json:"id"`
	Status              string                      `json:"status"`
	ArtifactRef         string                      `json:"artifact_ref"`
	Gates               []Epic9SecurityGateEvidence `json:"gates"`
	CertificationResult Epic9SecurityDecision       `json:"certification_result"`
	Waivers             []Epic9SecurityWaiver       `json:"waivers,omitempty"`
}

type Epic9SecurityGateEvidence struct {
	Gate           string                 `json:"gate"`
	Status         string                 `json:"status"`
	ScannerTool    string                 `json:"scanner_tool"`
	ScannerVersion string                 `json:"scanner_version"`
	Findings       []Epic9SecurityFinding `json:"findings,omitempty"`
}

type Epic9SecurityFinding struct {
	ID       string `json:"id"`
	Gate     string `json:"gate"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
	WaiverID string `json:"waiver_id,omitempty"`
	Summary  string `json:"summary"`
}

type Epic9SecurityWaiver struct {
	ID                   string   `json:"id"`
	FindingID            string   `json:"finding_id"`
	ApproverRole         string   `json:"approver_role"`
	ExpiresAt            string   `json:"expires_at"`
	Reason               string   `json:"reason"`
	CompensatingControls string   `json:"compensating_controls"`
	NotValidFor          []string `json:"not_valid_for"`
}

type Epic9SecurityDecision struct {
	Blocked          bool     `json:"blocked"`
	MissingEvidence  []string `json:"missing_evidence,omitempty"`
	BlockingFindings []string `json:"blocking_findings,omitempty"`
	BlockingReasons  []string `json:"blocking_reasons,omitempty"`
}

type Epic9GateJValidation struct {
	Status  string            `json:"status"`
	Missing []string          `json:"missing,omitempty"`
	Metrics Epic9GateJMetrics `json:"metrics"`
}

type Epic9GateJMetrics struct {
	GoldenPRDRef             string   `json:"golden_prd_ref"`
	GoldenPRDLocatorHash     string   `json:"golden_prd_locator_hash"`
	FactoryOrderID           string   `json:"factory_order_id"`
	GeneratedTemplateID      string   `json:"generated_template_id"`
	GeneratedFileCount       int      `json:"generated_file_count"`
	FactoryRuntimeVersionRef string   `json:"factory_runtime_version_ref"`
	SecurityGateCount        int      `json:"security_gate_count"`
	SecurityBlockingCount    int      `json:"security_blocking_count"`
	TraceScore               float64  `json:"trace_score"`
	RequiredPathsTotal       int      `json:"required_paths_total"`
	RequiredPathsPresent     int      `json:"required_paths_present"`
	ReleaseDecision          string   `json:"release_decision"`
	ReleaseDecisionRef       string   `json:"release_decision_ref"`
	AuditReportRef           string   `json:"audit_report_ref"`
	ResidualGapRefs          []string `json:"residual_gap_refs"`
}

type Epic9AuditEvidence struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type Epic9ProofOfWorkPacket struct {
	ID                 string                  `json:"id"`
	Status             string                  `json:"status"`
	Summary            string                  `json:"summary"`
	GoldenPRD          Epic9GoldenPRDSource    `json:"golden_prd"`
	GeneratedArtifacts Epic9GeneratedManifest  `json:"generated_artifacts"`
	SecurityGates      Epic9SecurityGateReport `json:"security_gates"`
	TraceGates         []Epic9ProofOfWorkItem  `json:"trace_gates"`
	AuthorityRecords   Epic9AuthorityEvidence  `json:"authority_records"`
	ReleaseEvidence    Epic9ReleaseEvidence    `json:"release_evidence"`
	ResidualRisks      []Epic9ProofOfWorkItem  `json:"residual_risks"`
	EventGraphRefs     []string                `json:"event_graph_refs"`
	ForbiddenActions   []Epic9ProofOfWorkItem  `json:"forbidden_actions"`
}

type Epic9ProofOfWorkItem struct {
	Label       string   `json:"label"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	ArtifactRef string   `json:"artifact_ref,omitempty"`
	Refs        []string `json:"refs,omitempty"`
}

type Epic9AuthorityEvidence struct {
	AuthorityRequestID  string   `json:"authority_request_id,omitempty"`
	AuthorityDecisionID string   `json:"authority_decision_id,omitempty"`
	HumanApprovalID     string   `json:"human_approval_id,omitempty"`
	Decision            string   `json:"decision"`
	Scope               []string `json:"scope"`
	Summary             string   `json:"summary"`
}

type Epic9ReleaseEvidence struct {
	ReleaseCandidateID string   `json:"release_candidate_id,omitempty"`
	Decision           string   `json:"decision"`
	DecisionRef        string   `json:"decision_ref,omitempty"`
	AuditReportRef     string   `json:"audit_report_ref,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs"`
}

// RunEpic9GoldenPRDProductFactoryRun executes the authorized Gate J local/dry-run fixture.
func RunEpic9GoldenPRDProductFactoryRun(ts *TaskStore, opts Epic9GoldenPRDOptions) (Epic9GoldenPRDRun, error) {
	if ts == nil {
		return Epic9GoldenPRDRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic9GoldenPRDRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic9GoldenPRDRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic9GoldenPRDRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic9GoldenPRDLocalDryRun
	}
	if opts.Mode != Epic9GoldenPRDLocalDryRun {
		return Epic9GoldenPRDRun{}, fmt.Errorf("unsupported Epic 9 fixture mode %q", opts.Mode)
	}

	ids := epic9IDs()
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 9 Golden PRD Product Factory Run",
		Description:            "Run the bounded Gate J local/dry-run simple CRUD tracker product-factory path from FactoryOrder to AuditReport.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic9_golden_prd_factory",
		RiskClass:              "high",
		ExpectedOutputs:        []string{"generated-saas-template-v1/**", "artifacts/golden-prd/security-gates/report.json", "artifacts/golden-prd/proof-of-work.json", "artifacts/golden-prd/audit-report.json"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic9GoldenPRDRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 9 golden PRD local dry-run lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	}

	localArtifacts := epic9LocalArtifacts(opts.WorkingDir)
	manifest := Epic9GeneratedManifest{TemplateID: SaaSTemplateV1ID, Root: localArtifacts.GeneratedTemplateDir}
	if !opts.OmitGeneratedArtifactEvidence {
		generated, err := GenerateSaaSTemplateV1(localArtifacts.GeneratedTemplateDir)
		if err != nil {
			return Epic9GoldenPRDRun{}, err
		}
		manifest = epic9GeneratedManifest(localArtifacts.GeneratedTemplateDir, generated)
		if err := epic7WriteJSON(localArtifacts.GeneratedManifest, manifest); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
		if err := epic9WriteText(localArtifacts.DeployPreviewDryRun, "Deployment preview dry run only.\nNo production deploy is performed.\nNo external service is provisioned.\n"); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	}

	securityEvidence, securityWaivers := epic9SecurityInputs(opts)
	securityResult := EvaluateSecurityGateCertification(securityEvidence, securityWaivers, epic9FixtureTime())
	securityReport := epic9SecurityReport(ids, securityEvidence, securityWaivers, securityResult, opts)
	if !opts.OmitSecurityGateEvidence {
		if err := epic7WriteJSON(localArtifacts.SecurityGateReport, securityReport); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	}

	validation := epic9EvaluateGateJ(ids, opts, manifest, securityReport, securityResult)
	graph, graphRun, err := epic9RecordEventGraph(ids, opts, manifest, securityReport, securityResult, validation)
	if err != nil {
		return Epic9GoldenPRDRun{}, err
	}

	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 9 Gate J golden PRD evidence attached", causes, opts.ConversationID); err != nil {
		return Epic9GoldenPRDRun{}, err
	}
	if graphRun.FailureID != "" {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{FailureIDs: []string{graphRun.FailureID}}, "Epic 9 negative Gate J fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 9 Gate J local/dry-run evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic9GoldenPRDRun{}, err
	}
	if validation.Status == "pass" && graphRun.Certification != nil {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 9 golden PRD local/dry-run certified", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 9 negative Gate J fixture rejected", []string{ids.gateResult, graphRun.FailureID}, causes, opts.ConversationID); err != nil {
		return Epic9GoldenPRDRun{}, err
	}

	projection := epic9BuildProjection(ids, opts, manifest, securityReport, validation, graphRun)
	if err := epic7WriteJSON(localArtifacts.ProofOfWork, projection.ProofOfWorkPacket); err != nil {
		return Epic9GoldenPRDRun{}, err
	}
	if graphRun.AuditReport != nil && !opts.OmitAuditReport {
		if err := epic7WriteJSON(localArtifacts.AuditReport, projection.AuditReport); err != nil {
			return Epic9GoldenPRDRun{}, err
		}
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic9GoldenPRDRun{}, err
	}

	return Epic9GoldenPRDRun{
		Mode:                    opts.Mode,
		WorkTask:                task,
		WorkProjection:          workProjection,
		EventGraph:              graph,
		FactoryOrderID:          ids.factoryOrder,
		RequirementID:           ids.requirement,
		AcceptanceCriterionID:   ids.acceptanceCriterion,
		TaskID:                  ids.task,
		ActorInvocationID:       ids.actorInvocation,
		RuntimeEnvelopeID:       ids.runtimeEnvelope,
		RuntimeResultID:         ids.runtimeResult,
		CapabilityArtifactID:    ids.capabilityArtifact,
		KnowledgeReferenceID:    ids.knowledgeReference,
		SourceIntentArtifactID:  ids.sourceIntentArtifact,
		GeneratedManifestID:     ids.generatedManifestArtifact,
		SecurityReportID:        ids.securityReportArtifact,
		RuntimeBOMArtifactID:    ids.runtimeBOMArtifact,
		DeployPreviewID:         ids.deployPreviewArtifact,
		ProofOfWorkArtifactID:   ids.proofOfWorkArtifact,
		TestCaseID:              ids.testCase,
		TestRunID:               ids.testRun,
		GateResultID:            ids.gateResult,
		FailureID:               graphRun.FailureID,
		FactoryRuntimeVersionID: ids.factoryRuntime,
		AuthorityRequestID:      ids.authorityRequest,
		AuthorityDecisionID:     ids.authorityDecision,
		HumanApprovalID:         ids.humanApproval,
		ReleaseCandidateID:      ids.releaseCandidate,
		CertificationID:         epic9CertificationID(graphRun.Certification),
		RejectionID:             graphRun.RejectionID,
		AuditReportID:           ids.auditReport,
		TraceCompleteness:       graphRun.Trace,
		CapabilityUsagePath:     graphRun.CapabilityUsagePath,
		KnowledgePath:           graphRun.KnowledgePath,
		GateJValidation:         projection.GateJValidation,
		Certification:           graphRun.Certification,
		Rejection:               graphRun.Rejection,
		AuditReport:             graphRun.AuditReport,
		Projection:              projection,
		LocalArtifacts:          localArtifacts,
		GeneratedManifest:       manifest,
		SecurityGateReport:      securityReport,
	}, nil
}

func (p Epic9GoldenPRDProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

type epic9FixtureIDs struct {
	factoryOrder              string
	requirement               string
	acceptanceCriterion       string
	task                      string
	actorIdentity             string
	humanActorIdentity        string
	actorInvocation           string
	runtimeEnvelope           string
	runtimeResult             string
	capabilityArtifact        string
	knowledgeReference        string
	sourceIntentArtifact      string
	generatedManifestArtifact string
	securityReportArtifact    string
	runtimeBOMArtifact        string
	deployPreviewArtifact     string
	proofOfWorkArtifact       string
	testCase                  string
	testRun                   string
	gateResult                string
	failure                   string
	factoryRuntime            string
	authorityRequest          string
	authorityDecision         string
	humanApproval             string
	releaseCandidate          string
	certification             string
	rejection                 string
	auditReport               string
	proofPacket               string
}

type epic9GraphRun struct {
	DecisionID          string
	RejectionID         string
	FailureID           string
	Trace               v39.TraceCompletenessGateResult
	CapabilityUsagePath v39.RequiredPath
	KnowledgePath       v39.RequiredPath
	Certification       *v39.Certification
	Rejection           *v39.Rejection
	AuditReport         *v39.AuditReport
}

func epic9IDs() epic9FixtureIDs {
	return epic9FixtureIDs{
		factoryOrder:              "fo_epic9_simple_crud_tracker",
		requirement:               "req_epic9_simple_crud_tracker_functional_trace",
		acceptanceCriterion:       "ac_epic9_simple_crud_tracker_factory_order_to_audit_report",
		task:                      "tsk_epic9_golden_prd_factory_run",
		actorIdentity:             "actor_epic9_golden_prd_factory",
		humanActorIdentity:        "actor_epic9_release_reviewer",
		actorInvocation:           "invoke_epic9_golden_prd_factory_run",
		runtimeEnvelope:           "env_epic9_golden_prd_factory_run",
		runtimeResult:             "result_epic9_golden_prd_factory_run",
		capabilityArtifact:        "cap_art_epic9_saas_template_generator",
		knowledgeReference:        "know_epic9_authorization_packet",
		sourceIntentArtifact:      "art_epic9_golden_prd_source_intent",
		generatedManifestArtifact: "art_epic9_generated_manifest",
		securityReportArtifact:    "art_epic9_security_gate_report",
		runtimeBOMArtifact:        "art_epic9_runtime_bom",
		deployPreviewArtifact:     "art_epic9_deploy_preview_dry_run",
		proofOfWorkArtifact:       "art_epic9_proof_of_work",
		testCase:                  "tc_epic9_golden_prd_factory_run",
		testRun:                   "tr_epic9_golden_prd_factory_run",
		gateResult:                "gate_epic9_golden_prd_factory_run",
		failure:                   "fail_epic9_golden_prd_factory_run",
		factoryRuntime:            epic9FactoryRuntime,
		authorityRequest:          "auth_req_epic9_release_decision",
		authorityDecision:         "auth_dec_epic9_release_decision",
		humanApproval:             "human_approval_epic9_release_decision",
		releaseCandidate:          "rc_epic9_simple_crud_tracker",
		certification:             "cert_epic9_simple_crud_tracker",
		rejection:                 "reject_epic9_simple_crud_tracker",
		auditReport:               "audit_epic9_simple_crud_tracker",
		proofPacket:               "proof_epic9_simple_crud_tracker",
	}
}

func epic9LocalArtifacts(dir string) Epic9LocalArtifacts {
	root := filepath.Join(dir, "artifacts", "golden-prd")
	return Epic9LocalArtifacts{
		Root:                 root,
		GeneratedTemplateDir: filepath.Join(dir, "generated-saas-template-v1"),
		GeneratedManifest:    filepath.Join(root, "generated-manifest.json"),
		SecurityGateReport:   filepath.Join(root, "security-gates", "report.json"),
		DeployPreviewDryRun:  filepath.Join(root, "deploy-preview-dry-run.txt"),
		ProofOfWork:          filepath.Join(root, "proof-of-work.json"),
		AuditReport:          filepath.Join(root, "audit-report.json"),
	}
}

func epic9GeneratedManifest(root string, manifest SaaSTemplateManifest) Epic9GeneratedManifest {
	files := append([]string(nil), manifest.Files...)
	return Epic9GeneratedManifest{
		TemplateID:  manifest.TemplateID,
		Root:        root,
		Files:       files,
		FileCount:   len(files),
		ContentHash: epic7Hash(manifest.TemplateID + ":" + strings.Join(files, "\n")),
	}
}

func epic9SecurityInputs(opts Epic9GoldenPRDOptions) ([]SecurityGateEvidence, []SecurityWaiver) {
	if opts.OmitSecurityGateEvidence {
		return nil, nil
	}
	scanners := SaaSTemplateV1SecurityScanners()
	evidence := make([]SecurityGateEvidence, 0, len(scanners))
	for _, scanner := range scanners {
		gate := SecurityGateEvidence{
			Gate:           scanner.Gate,
			Status:         SecurityGateStatusPass,
			ScannerTool:    scanner.Tool,
			ScannerVersion: scanner.Version,
		}
		if opts.AddOpenCriticalSecurityFinding && scanner.Gate == GateSAST {
			gate.Status = SecurityGateStatusFail
			gate.Findings = append(gate.Findings, SecurityFinding{ID: "finding_epic9_open_critical_sast", Gate: scanner.Gate, Severity: FindingSeverityCritical, Status: FindingStatusOpen, Summary: "Critical SAST finding intentionally injected by the Gate J negative seam."})
		}
		if opts.AddOpenHighSecurityFinding && scanner.Gate == GateConfigurationSecurityCheck {
			if !opts.AddValidHighWaiver {
				gate.Status = SecurityGateStatusFail
			}
			gate.Findings = append(gate.Findings, SecurityFinding{ID: "finding_epic9_open_high_config", Gate: scanner.Gate, Severity: FindingSeverityHigh, Status: FindingStatusOpen, WaiverID: "waiver_epic9_high_config", Summary: "High configuration finding intentionally injected by the Gate J negative seam."})
		}
		evidence = append(evidence, gate)
	}
	var waivers []SecurityWaiver
	if opts.AddOpenHighSecurityFinding && opts.AddValidHighWaiver {
		waivers = append(waivers, SecurityWaiver{
			ID:                   "waiver_epic9_high_config",
			FindingID:            "finding_epic9_open_high_config",
			ApproverRole:         "SecurityReviewer",
			ExpiresAt:            epic9FixtureTime().Add(24 * time.Hour),
			Reason:               "Local dry-run waiver for non-production configuration finding.",
			CompensatingControls: "Gate J remains local/dry-run and cannot deploy.",
			NotValidFor:          []string{"production_deployment"},
		})
	}
	return evidence, waivers
}

func epic9SecurityReport(ids epic9FixtureIDs, evidence []SecurityGateEvidence, waivers []SecurityWaiver, result SecurityGateCertificationResult, opts Epic9GoldenPRDOptions) Epic9SecurityGateReport {
	status := "pass"
	if opts.OmitSecurityGateEvidence {
		status = "missing"
	} else if result.Blocked {
		status = "fail"
	}
	return Epic9SecurityGateReport{
		ID:                  "security_epic9_simple_crud_tracker",
		Status:              status,
		ArtifactRef:         ids.securityReportArtifact,
		Gates:               epic9SecurityGateRecords(evidence),
		CertificationResult: epic9SecurityDecision(result),
		Waivers:             epic9SecurityWaiverRecords(waivers),
	}
}

func epic9SecurityGateRecords(evidence []SecurityGateEvidence) []Epic9SecurityGateEvidence {
	records := make([]Epic9SecurityGateEvidence, 0, len(evidence))
	for _, gate := range evidence {
		records = append(records, Epic9SecurityGateEvidence{
			Gate:           string(gate.Gate),
			Status:         string(gate.Status),
			ScannerTool:    gate.ScannerTool,
			ScannerVersion: gate.ScannerVersion,
			Findings:       epic9SecurityFindingRecords(gate.Findings),
		})
	}
	return records
}

func epic9SecurityFindingRecords(findings []SecurityFinding) []Epic9SecurityFinding {
	records := make([]Epic9SecurityFinding, 0, len(findings))
	for _, finding := range findings {
		records = append(records, Epic9SecurityFinding{ID: finding.ID, Gate: string(finding.Gate), Severity: string(finding.Severity), Status: string(finding.Status), WaiverID: finding.WaiverID, Summary: finding.Summary})
	}
	return records
}

func epic9SecurityWaiverRecords(waivers []SecurityWaiver) []Epic9SecurityWaiver {
	records := make([]Epic9SecurityWaiver, 0, len(waivers))
	for _, waiver := range waivers {
		records = append(records, Epic9SecurityWaiver{ID: waiver.ID, FindingID: waiver.FindingID, ApproverRole: waiver.ApproverRole, ExpiresAt: waiver.ExpiresAt.Format(time.RFC3339), Reason: waiver.Reason, CompensatingControls: waiver.CompensatingControls, NotValidFor: append([]string(nil), waiver.NotValidFor...)})
	}
	return records
}

func epic9SecurityDecision(result SecurityGateCertificationResult) Epic9SecurityDecision {
	return Epic9SecurityDecision{
		Blocked:          result.Blocked,
		MissingEvidence:  epic9SecurityGateStrings(result.MissingEvidence),
		BlockingFindings: epic9SecurityFindingIDs(result.BlockingFindings),
		BlockingReasons:  append([]string(nil), result.BlockingReasons...),
	}
}

func epic9SecurityGateStrings(gates []SecurityGateID) []string {
	out := make([]string, 0, len(gates))
	for _, gate := range gates {
		out = append(out, string(gate))
	}
	return out
}

func epic9SecurityFindingIDs(findings []SecurityFinding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.ID)
	}
	return out
}

func epic9EvaluateGateJ(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions, manifest Epic9GeneratedManifest, report Epic9SecurityGateReport, securityResult SecurityGateCertificationResult) Epic9GateJValidation {
	var missing []string
	if opts.OmitFactoryOrder {
		missing = append(missing, "FactoryOrder missing")
	}
	if opts.OmitSourceIntent {
		missing = append(missing, "selected PRD/source intent evidence missing")
	}
	if opts.OmitAcceptanceEvidence {
		missing = append(missing, "acceptance evidence missing")
	}
	if opts.OmitGeneratedArtifactEvidence || manifest.FileCount == 0 {
		missing = append(missing, "generated artifact evidence missing")
	}
	if opts.OmitSecurityGateEvidence || len(report.Gates) == 0 {
		missing = append(missing, "security-gate evidence missing")
	}
	for _, gate := range securityResult.MissingEvidence {
		missing = append(missing, "security-gate evidence missing: "+string(gate))
	}
	for _, finding := range securityResult.BlockingFindings {
		switch finding.Severity {
		case FindingSeverityCritical:
			missing = append(missing, "open critical security finding: "+finding.ID)
		case FindingSeverityHigh:
			missing = append(missing, "open high security finding without valid waiver: "+finding.ID)
		default:
			missing = append(missing, "blocking security finding: "+finding.ID)
		}
	}
	if opts.OmitFactoryRuntimeVersion {
		missing = append(missing, "FactoryRuntimeVersion missing")
	}
	if opts.OmitReleaseAuthority {
		missing = append(missing, "release decision authority missing")
	}
	if opts.OmitAuditReport {
		missing = append(missing, "AuditReport missing")
	}
	status := "pass"
	if len(missing) > 0 {
		status = "fail"
	}
	return Epic9GateJValidation{
		Status:  status,
		Missing: missing,
		Metrics: Epic9GateJMetrics{
			GoldenPRDRef:             epic9GoldenPRDSourceRef,
			GoldenPRDLocatorHash:     epic9GoldenPRDLocatorHash(),
			FactoryOrderID:           ids.factoryOrder,
			GeneratedTemplateID:      manifest.TemplateID,
			GeneratedFileCount:       manifest.FileCount,
			FactoryRuntimeVersionRef: ids.factoryRuntime,
			SecurityGateCount:        len(report.Gates),
			SecurityBlockingCount:    len(securityResult.BlockingFindings) + len(securityResult.MissingEvidence),
			ResidualGapRefs:          []string{"R-001", "R-002", "R-003"},
		},
	}
}

func epic9RecordEventGraph(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions, manifest Epic9GeneratedManifest, report Epic9SecurityGateReport, securityResult SecurityGateCertificationResult, validation Epic9GateJValidation) (*v39.InMemoryStore, epic9GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic9FixtureTime()
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

	artifactRefs := epic9ArtifactRefs(ids, opts)
	taskCommon := epic9Common(ids.task, v39.TypeTask, taskStatus)
	taskCommon.SourceRefs = []string{"capability:" + ids.capabilityArtifact}
	if !opts.OmitSourceIntent {
		taskCommon.SourceRefs = append(taskCommon.SourceRefs, epic9KnowledgeSourceRef)
	}
	records := []v39.Record{
		&v39.Task{CommonNode: taskCommon, FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic9_golden_prd_factory", State: taskStatus, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic9Common(ids.actorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic9FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic9Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic9FixtureHumanActorID, ActorType: "human", IdentityMode: "fixture"},
		&v39.CapabilityArtifact{CommonNode: epic9Common(ids.capabilityArtifact, v39.TypeCapabilityArtifact, "active"), ArtifactID: ids.capabilityArtifact, ArtifactType: "workflow_pack", Name: "Epic 9 SaaS Template v1 local generator", ArtifactVersion: "v1", SourceRepoOrOrigin: "transpara-ai/work", ContentHash: epic7Hash(SaaSTemplateV1ID + ":" + strings.Join(manifest.Files, "\n")), Owner: "work", RiskClass: "high", ActivationScope: "fixture_only", EvalRefs: []string{ids.testCase}, HumanReviewRef: epic9DocsReviewedHead, RollbackRef: "not_applicable_local_dry_run_fixture", UsageLoggingRequired: true},
		&v39.ActorInvocation{CommonNode: epic9Common(ids.actorInvocation, v39.TypeActorInvocation, runtimeStatus), TaskID: ids.task, Runtime: "local", ActorID: epic9FixtureActorID, InputContractHash: epic9GoldenPRDLocatorHash(), OutputContractHash: strPtr(epic7Hash("epic9-output:" + strings.Join(artifactRefs, ":")))},
		&v39.RuntimeEnvelope{CommonNode: epic9Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_golden_prd_factory_fixture", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: epic9FixtureActorID, AuthorityDecisionRef: "human_authorized_in_chat_2026-06-02_docs_main_" + epic7ShortSHA(epic9DocsMergeSHA), AllowedFiles: []string{"generated-saas-template-v1/**", "artifacts/golden-prd/**"}, DeniedFiles: []string{".git", "../", ".env", "secrets.env"}, AllowedCommands: []string{"generate_saas_template_v1", "write_security_gate_report", "write_proof_packet", "write_audit_report"}, DeniedCommands: []string{"gh pr create", "git push", "git merge", "gh pr merge", "deploy", "protected_execution.run", "capability.activate.global", "PolicyEngineAdapterDecision"}, NetworkPolicy: "disabled", SecretsPolicy: "none", WorkingDirectory: opts.WorkingDir, Timeout: "1s", ResourceLimits: map[string]any{"max_live_prs_created": 0, "max_branch_pushes": 0, "max_production_deploys": 0, "max_protected_executions": 0, "max_global_activations": 0, "max_repos_mutated": 0}, ExpectedOutputs: []string{"generated-saas-template-v1/**", "artifacts/golden-prd/security-gates/report.json", "artifacts/golden-prd/proof-of-work.json", "artifacts/golden-prd/audit-report.json"}, OutputContract: map[string]any{"mode": string(opts.Mode), "gate": "gate_j_golden_prd_product_factory_run"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult", "ReleaseCandidate -> Certification or Rejection -> AuditReport"}, PostRunValidationPlan: []string{"epic9EvaluateGateJ", "go test ./... -run Epic9", "make verify"}, EnvelopeHash: epic7Hash("epic9-envelope:" + string(opts.Mode))},
		&v39.RuntimeResult{CommonNode: epic9Common(ids.runtimeResult, v39.TypeRuntimeResult, runtimeStatus), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_golden_prd_factory_fixture", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: runtimeStatus, ArtifactRefs: artifactRefs, ChangedFiles: epic9ChangedFiles(manifest, opts), CommandLog: epic9CommandLog(opts, validation), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{"local_security_gate_policy", "no_policy_engine_adapter_decision"}, PostRunValidationRefs: []string{ids.testRun}},
		&v39.TestCase{CommonNode: epic9Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: epic9AcceptanceCriterionRef(ids, opts), RequirementID: &ids.requirement, Name: "Epic 9 golden PRD Gate J evidence", TestType: "unit", Path: strPtr("work/epic9_golden_prd_factory_run_test.go")},
		&v39.TestRun{CommonNode: epic9Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./... -run Epic9"},
		&v39.GateResult{CommonNode: epic9Common(ids.gateResult, v39.TypeGateResult, validation.Status), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_j_golden_prd_product_factory_run", EvidenceRefs: append([]string{ids.testRun}, artifactRefs...)},
	}
	if !opts.OmitFactoryOrder {
		records = append([]v39.Record{
			&v39.FactoryOrder{CommonNode: epic9Common(ids.factoryOrder, v39.TypeFactoryOrder, taskStatus), FactoryOrderVersion: 1, SourceIntentHash: epic9SourceIntentHash(opts), SourceIntentRef: epic9SourceIntentRef(opts), RiskClass: "high", ReleasePolicy: "human_approval_required"},
			&v39.Requirement{CommonNode: epic9Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Prove the simple CRUD tracker golden PRD through a local/dry-run FactoryOrder-to-AuditReport product factory path.", Source: "explicit", RiskClass: "high"},
		}, records...)
		if !opts.OmitAcceptanceEvidence {
			records = append(records, &v39.AcceptanceCriterion{CommonNode: epic9Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, acceptanceStatus), RequirementID: ids.requirement, Text: "Gate J passes only when source intent, generated SaaS Template v1 artifacts, product gates, security gates, trace gates, release authority, release decision, proof packet, and AuditReport are all inspectable.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "golden_prd_factory_order_to_audit_report_trace", OwnerRole: "maintainer", RiskClass: "high"})
		}
	}
	records = append(records, epic9ArtifactRecords(ids, opts, manifest, report)...)
	if !opts.OmitReleaseAuthority {
		records = append(records, epic9AuthorityRecords(ids)...)
	}
	if validation.Status == "fail" {
		records = append(records, &v39.Failure{CommonNode: epic9Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "gate_j_golden_prd_blocked", Severity: "high", Summary: strings.Join(validation.Missing, "; ")})
	}
	if err := epic7AppendRecords(graph, records...); err != nil {
		return nil, epic9GraphRun{}, err
	}
	if !opts.OmitSourceIntent {
		if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{CommonNode: epic9Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded"), ReferenceCreatedAt: createdAt, SourceSystem: "transpara-ai/docs", SourceRef: epic9KnowledgeSourceRef, SourceHashOrImmutableLocator: "docs-pr-91-merged-" + epic9DocsMergeSHA + "-reviewed-head-" + epic9DocsReviewedHead, RetrievedAt: createdAt, UsedByActor: epic9FixtureActorID, UsedInTask: ids.task, InfluenceSummary: "Gate J authorization constrained the golden PRD, local dry-run mode, FactoryOrder-to-AuditReport evidence model, security gates, rejection paths, residual risks, and stop conditions.", RiskScope: "high", TrustLevel: "human_authorized", FreshnessStatus: "current", RedactionState: "none"}}); err != nil {
			return nil, epic9GraphRun{}, err
		}
	}
	if _, err := graph.RecordCapabilityUsage(ids.task, ids.capabilityArtifact, epic9Common("edge_epic9_used_capability", v39.TypeCapabilityArtifact, "recorded")); err != nil {
		return nil, epic9GraphRun{}, err
	}
	if !opts.OmitFactoryRuntimeVersion {
		if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic9Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: epic9FactoryRuntimeString, CapabilityVersionRefs: []string{}, RuntimeRefs: []string{SaaSTemplateV1ID, SaaSTemplateV1SecurityGateVersion, "work.local_golden_prd_factory_fixture@1"}}); err != nil {
			return nil, epic9GraphRun{}, err
		}
	}
	if err := epic9AppendEdges(graph, ids, opts, artifactRefs, validation.Status == "fail"); err != nil {
		return nil, epic9GraphRun{}, err
	}
	if opts.OmitFactoryOrder || opts.OmitAcceptanceEvidence {
		trace, _ := graph.EvaluateTraceCompletenessGate(ids.releaseCandidate)
		return graph, epic9GraphRun{FailureID: ids.failure, Trace: trace}, nil
	}

	frvID := ids.factoryRuntime
	rc := &v39.ReleaseCandidate{CommonNode: epic9Common(ids.releaseCandidate, v39.TypeReleaseCandidate, releaseStatus), FactoryOrderID: ids.factoryOrder, ArtifactRefs: artifactRefs}
	if !opts.OmitFactoryRuntimeVersion {
		rc.FactoryRuntimeVersionID = &frvID
	}
	recordedRC, err := graph.RecordReleaseCandidate(rc)
	if err != nil {
		return nil, epic9GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(recordedRC.CommonNode.ID)
	capabilityPath, _ := graph.CapabilityUsageEvidencePath(recordedRC.CommonNode.ID)
	knowledgePath, _ := graph.AdvisoryReferenceEvidencePath(recordedRC.CommonNode.ID)
	if validation.Status == "pass" && traceErr != nil {
		return nil, epic9GraphRun{}, traceErr
	}
	if validation.Status == "pass" {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic9Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: epic9FixtureHumanActorID, Reason: "Gate J golden PRD local/dry-run evidence is complete for the bounded simple CRUD tracker Work fixture.", EvidenceRefs: []string{ids.gateResult, ids.testRun, ids.authorityDecision, ids.proofOfWorkArtifact}})
		if err != nil {
			return nil, epic9GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic9Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic9GraphRun{}, err
		}
		return graph, epic9GraphRun{DecisionID: cert.CommonNode.ID, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Certification: cert, AuditReport: audit}, nil
	}
	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic9Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: epic9FixtureHumanActorID, Reason: "Gate J golden PRD evidence is incomplete or unsafe: " + strings.Join(validation.Missing, "; "), EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic9GraphRun{}, err
	}
	var audit *v39.AuditReport
	if !opts.OmitAuditReport {
		audit, err = graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic9Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic9GraphRun{}, err
		}
	}
	return graph, epic9GraphRun{DecisionID: rejection.CommonNode.ID, RejectionID: rejection.CommonNode.ID, FailureID: ids.failure, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Rejection: rejection, AuditReport: audit}, nil
}

func epic9ArtifactRefs(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions) []string {
	var refs []string
	if !opts.OmitSourceIntent {
		refs = append(refs, ids.sourceIntentArtifact)
	}
	if !opts.OmitGeneratedArtifactEvidence {
		refs = append(refs, ids.generatedManifestArtifact, ids.deployPreviewArtifact)
	}
	if !opts.OmitSecurityGateEvidence {
		refs = append(refs, ids.securityReportArtifact)
	}
	if !opts.OmitFactoryRuntimeVersion {
		refs = append(refs, ids.runtimeBOMArtifact)
	}
	refs = append(refs, ids.proofOfWorkArtifact)
	return refs
}

func epic9ArtifactRecords(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions, manifest Epic9GeneratedManifest, report Epic9SecurityGateReport) []v39.Record {
	var records []v39.Record
	if !opts.OmitSourceIntent {
		path := epic9GoldenPRDSourceRef
		records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.sourceIntentArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "document", Path: &path, ContentHash: strPtr(epic9GoldenPRDLocatorHash())})
	}
	if !opts.OmitGeneratedArtifactEvidence {
		manifestPath := "artifacts/golden-prd/generated-manifest.json"
		records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.generatedManifestArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: &manifestPath, ContentHash: strPtr(manifest.ContentHash)})
		deployPath := "artifacts/golden-prd/deploy-preview-dry-run.txt"
		records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.deployPreviewArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "deployment_bundle", Path: &deployPath, ContentHash: strPtr(epic7Hash("epic9-deploy-preview-dry-run"))})
	}
	if !opts.OmitSecurityGateEvidence {
		securityPath := "artifacts/golden-prd/security-gates/report.json"
		records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.securityReportArtifact, v39.TypeArtifact, epic9ArtifactStatus(report.Status == "pass")), TaskID: &ids.task, ArtifactType: "report", Path: &securityPath, ContentHash: strPtr(epic7Hash(report.Status + ":" + strings.Join(report.CertificationResult.BlockingReasons, "|")))})
	}
	if !opts.OmitFactoryRuntimeVersion {
		bomPath := "generated-saas-template-v1/factory-runtime-bom.json"
		records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.runtimeBOMArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "config", Path: &bomPath, ContentHash: strPtr(epic7Hash(SaaSTemplateV1ID + ":" + SaaSTemplateV1SecurityGateVersion))})
	}
	proofPath := "artifacts/golden-prd/proof-of-work.json"
	records = append(records, &v39.Artifact{CommonNode: epic9Common(ids.proofOfWorkArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: &proofPath, ContentHash: strPtr(epic7Hash("epic9-proof:" + validationHashStatus(report.Status)))})
	return records
}

func epic9AuthorityRecords(ids epic9FixtureIDs) []v39.Record {
	return []v39.Record{
		&v39.AuthorityRequest{CommonNode: epic9Common(ids.authorityRequest, v39.TypeAuthorityRequest, "recorded"), ActorID: epic9FixtureActorID, ActorRole: "local_golden_prd_factory", Action: "release.local_certify_or_reject", TargetType: v39.TypeReleaseCandidate, TargetID: ids.releaseCandidate, RiskClass: "high", Reason: "Authorize only local/dry-run Gate J certification or rejection from evidence.", ProposedCommand: strPtr("record local release decision"), EvidenceRefs: []string{ids.gateResult, ids.securityReportArtifact, ids.proofOfWorkArtifact}},
		&v39.AuthorityDecision{CommonNode: epic9Common(ids.authorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.authorityRequest, DeciderActorID: epic9FixtureHumanActorID, DeciderRole: "maintainer", Decision: "ApprovalRequired", Reason: "Approve bounded local release decision only; no production deploy, live PR mutation, protected execution, or global activation is allowed.", Scope: []string{"release.local_certify_or_reject", ids.releaseCandidate, "local_dry_run_only"}, Conditions: []string{"evidence-complete before certification", "critical findings block", "high findings require valid waiver", "no production deployment", "no live repository mutation"}},
		&v39.HumanApproval{CommonNode: epic9Common(ids.humanApproval, v39.TypeHumanApproval, "approved"), RequestRef: ids.authorityRequest, ApproverActorID: epic9FixtureHumanActorID, ApproverRole: "maintainer", Decision: "approved", Reason: "Human authority is limited to certifying or rejecting the local Gate J release candidate from inspectable evidence."},
	}
}

func epic9AppendEdges(graph *v39.InMemoryStore, ids epic9FixtureIDs, opts Epic9GoldenPRDOptions, artifactRefs []string, includeFailure bool) error {
	createdAt := epic9FixtureTime()
	var edges []v39.CommonEdge
	if !opts.OmitFactoryOrder {
		edges = append(edges, epic9Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt))
		if !opts.OmitAcceptanceEvidence {
			edges = append(edges,
				epic9Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
				epic9Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
			)
		}
	}
	edges = append(edges,
		epic9Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic9Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic9Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic9Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic9Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic9Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	)
	for _, artifact := range artifactRefs {
		edges = append(edges, epic9Edge("task_"+artifact, v39.EdgeProduced, ids.task, artifact, createdAt))
	}
	if !opts.OmitReleaseAuthority {
		edges = append(edges,
			epic9Edge("release_auth_req", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
			epic9Edge("release_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
			epic9Edge("release_human", v39.EdgeApprovedBy, ids.authorityRequest, ids.humanApproval, createdAt),
		)
	}
	if includeFailure {
		edges = append(edges, epic9Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic9BuildProjection(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions, manifest Epic9GeneratedManifest, report Epic9SecurityGateReport, validation Epic9GateJValidation, graphRun epic9GraphRun) Epic9GoldenPRDProjection {
	auditEvidence := Epic9AuditEvidence{}
	if graphRun.AuditReport != nil {
		auditEvidence = Epic9AuditEvidence{ID: graphRun.AuditReport.CommonNode.ID, TargetType: graphRun.AuditReport.TargetType, TargetID: graphRun.AuditReport.TargetID, Status: statusString(graphRun.AuditReport.CommonNode.Status), TraceScore: graphRun.AuditReport.TraceScore, MissingLinks: append([]string(nil), graphRun.AuditReport.MissingLinks...)}
	}
	validation.Metrics.TraceScore = epic9TraceScore(graphRun.Trace)
	validation.Metrics.RequiredPathsTotal = len(graphRun.Trace.RequiredPaths)
	validation.Metrics.RequiredPathsPresent = epic9RequiredPathsPresent(graphRun.Trace)
	validation.Metrics.ReleaseDecision = epic9ReleaseDecision(graphRun)
	validation.Metrics.ReleaseDecisionRef = graphRun.DecisionID
	validation.Metrics.AuditReportRef = auditEvidence.ID
	if validation.Status != "pass" && len(validation.Metrics.ResidualGapRefs) == 0 {
		validation.Metrics.ResidualGapRefs = append([]string(nil), validation.Missing...)
	}
	projection := Epic9GoldenPRDProjection{
		GeneratedAt:       epic9FixtureTime().Format(time.RFC3339),
		Source:            "work-epic9-golden-prd-product-factory-run-fixture",
		Mode:              Epic9GoldenPRDLocalDryRun,
		GoldenPRD:         epic9GoldenPRDSource(opts),
		GeneratedManifest: manifest,
		SecurityGates:     report,
		GateJValidation:   validation,
		AuditReport:       auditEvidence,
		ProofOfWorkPacket: Epic9ProofOfWorkPacket{
			ID:                 ids.proofPacket,
			Status:             validation.Status,
			Summary:            "Epic 9 Gate J proof: local/dry-run simple CRUD tracker FactoryOrder-to-AuditReport path using SaaS Template v1.",
			GoldenPRD:          epic9GoldenPRDSource(opts),
			GeneratedArtifacts: manifest,
			SecurityGates:      report,
			TraceGates:         epic9TraceGateItems(graphRun.Trace),
			AuthorityRecords:   epic9AuthorityEvidence(ids, opts),
			ReleaseEvidence:    epic9ReleaseEvidence(ids, graphRun, auditEvidence),
			ResidualRisks: []Epic9ProofOfWorkItem{
				{Label: "R-001", Status: "excluded", Summary: "No runner/worktree protected execution, branch push, live PR creation, or production runtime mutation is performed."},
				{Label: "R-002", Status: "excluded", Summary: "No real protected side effects are performed and no production ExecutionReceipt path is claimed."},
				{Label: "R-003", Status: "excluded", Summary: "No PolicyEngineAdapterDecision or policy-bundle evidence is used."},
			},
			EventGraphRefs: []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeGateResult, ids.gateResult), egRef(v39.TypeReleaseCandidate, ids.releaseCandidate), egRef(v39.TypeAuditReport, ids.auditReport)},
			ForbiddenActions: []Epic9ProofOfWorkItem{
				{Label: "live PR creation", Status: "not_run", Summary: "No gh pr create or GitHub mutation is performed."},
				{Label: "branch push or merge", Status: "not_run", Summary: "No branch push, default-branch push, merge, or auto-merge is performed."},
				{Label: "production deploy", Status: "not_run", Summary: "Deploy preview evidence is a dry-run text artifact only."},
				{Label: "protected execution", Status: "not_run", Summary: "No protected runner/worktree execution is invoked."},
				{Label: "global activation", Status: "not_run", Summary: "No capability activation or global activation is requested."},
			},
		},
	}
	if validation.Status != "pass" {
		projection.Errors = append([]string(nil), validation.Missing...)
	}
	return projection
}

func epic9TraceGateItems(trace v39.TraceCompletenessGateResult) []Epic9ProofOfWorkItem {
	items := make([]Epic9ProofOfWorkItem, 0, len(trace.RequiredPaths))
	for _, path := range trace.RequiredPaths {
		status := "fail"
		if path.Completed {
			status = "pass"
		}
		items = append(items, Epic9ProofOfWorkItem{Label: path.Name, Status: status, Summary: strings.Join(path.Missing, "; "), Refs: append([]string(nil), path.NodeIDs...)})
	}
	return items
}

func epic9AuthorityEvidence(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions) Epic9AuthorityEvidence {
	if opts.OmitReleaseAuthority {
		return Epic9AuthorityEvidence{Decision: "missing", Summary: "Release certification/rejection authority omitted by negative seam."}
	}
	return Epic9AuthorityEvidence{
		AuthorityRequestID:  ids.authorityRequest,
		AuthorityDecisionID: ids.authorityDecision,
		HumanApprovalID:     ids.humanApproval,
		Decision:            "approved",
		Scope:               []string{"release.local_certify_or_reject", ids.releaseCandidate, "local_dry_run_only"},
		Summary:             "Human authority is limited to local/dry-run certification or rejection from evidence.",
	}
}

func epic9ReleaseEvidence(ids epic9FixtureIDs, graphRun epic9GraphRun, audit Epic9AuditEvidence) Epic9ReleaseEvidence {
	evidence := Epic9ReleaseEvidence{ReleaseCandidateID: ids.releaseCandidate, Decision: epic9ReleaseDecision(graphRun), DecisionRef: graphRun.DecisionID, AuditReportRef: audit.ID}
	if graphRun.Certification != nil {
		evidence.EvidenceRefs = append([]string(nil), graphRun.Certification.EvidenceRefs...)
	}
	if graphRun.Rejection != nil {
		evidence.EvidenceRefs = append([]string(nil), graphRun.Rejection.EvidenceRefs...)
	}
	return evidence
}

func epic9GoldenPRDSource(opts Epic9GoldenPRDOptions) Epic9GoldenPRDSource {
	locatorHash := epic9GoldenPRDLocatorHash()
	sourceRef := epic9GoldenPRDSourceRef
	if opts.OmitSourceIntent {
		locatorHash = ""
		sourceRef = ""
	}
	return Epic9GoldenPRDSource{ID: "golden_prd_simple_crud_tracker_v1", Name: epic9GoldenPRDName, SourceRef: sourceRef, LocatorHash: locatorHash}
}

func epic9GoldenPRDLocatorHash() string {
	return epic7Hash(epic9GoldenPRDName + "|" + epic9GoldenPRDSourceRef)
}

func epic9SourceIntentHash(opts Epic9GoldenPRDOptions) string {
	if opts.OmitSourceIntent {
		return "source-intent-omitted-negative-seam"
	}
	return epic9GoldenPRDLocatorHash()
}

func epic9SourceIntentRef(opts Epic9GoldenPRDOptions) string {
	if opts.OmitSourceIntent {
		return "missing_source_intent_negative_seam"
	}
	return epic9GoldenPRDSourceRef
}

func epic9AcceptanceCriterionRef(ids epic9FixtureIDs, opts Epic9GoldenPRDOptions) *string {
	if opts.OmitAcceptanceEvidence {
		return nil
	}
	return &ids.acceptanceCriterion
}

func epic9ChangedFiles(manifest Epic9GeneratedManifest, opts Epic9GoldenPRDOptions) []string {
	if opts.OmitGeneratedArtifactEvidence {
		return []string{}
	}
	out := make([]string, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		out = append(out, "generated-saas-template-v1/"+file)
	}
	return out
}

func epic9ArtifactStatus(ok bool) string {
	if ok {
		return "verified"
	}
	return "rejected"
}

func epic9CommandLog(opts Epic9GoldenPRDOptions, validation Epic9GateJValidation) []string {
	log := []string{
		"0:select_golden_prd:" + missingStatus(!opts.OmitSourceIntent),
		"1:record_factory_order:" + missingStatus(!opts.OmitFactoryOrder),
		"2:generate_saas_template_v1:" + missingStatus(!opts.OmitGeneratedArtifactEvidence),
		"3:record_security_gates:" + missingStatus(!opts.OmitSecurityGateEvidence),
		"4:record_factory_runtime_version:" + missingStatus(!opts.OmitFactoryRuntimeVersion),
		"5:record_release_authority:" + missingStatus(!opts.OmitReleaseAuthority),
		"6:record_audit_report:" + missingStatus(!opts.OmitAuditReport),
		"GateJ:" + validation.Status,
		"live_pr_creation:not_run",
		"branch_push:not_run",
		"merge:not_run",
		"deploy:not_run",
		"protected_execution:not_run",
		"global_activation:not_run",
		"PolicyEngineAdapterDecision:not_used",
	}
	return log
}

func epic9ReleaseDecision(graphRun epic9GraphRun) string {
	if graphRun.Certification != nil {
		return "certified"
	}
	if graphRun.Rejection != nil {
		return "rejected"
	}
	return "blocked"
}

func epic9TraceScore(trace v39.TraceCompletenessGateResult) float64 {
	if len(trace.RequiredPaths) == 0 {
		if trace.Completed {
			return 1
		}
		return 0
	}
	return float64(epic9RequiredPathsPresent(trace)) / float64(len(trace.RequiredPaths))
}

func epic9RequiredPathsPresent(trace v39.TraceCompletenessGateResult) int {
	var present int
	for _, path := range trace.RequiredPaths {
		if path.Completed {
			present++
		}
	}
	return present
}

func epic9Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{ID: id, Type: typ, CreatedAt: epic9FixtureTime(), CreatedBy: epic9FixtureActorID, Status: &status, IdempotencyKey: "idem_" + id, CorrelationID: "corr_epic9_golden_prd_factory_run"}
}

func epic9Edge(suffix, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic9_" + suffix + "_" + from + "_" + to
	return v39.CommonEdge{ID: id, Type: typ, FromID: from, ToID: to, CreatedAt: createdAt, CreatedBy: epic9FixtureActorID, CorrelationID: "corr_epic9_golden_prd_factory_run", IdempotencyKey: "idem_" + id}
}

func epic9FixtureTime() time.Time {
	return time.Date(2026, 6, 2, 16, 30, 0, 0, time.UTC)
}

func epic9CertificationID(cert *v39.Certification) string {
	if cert == nil {
		return ""
	}
	return cert.CommonNode.ID
}

func epic9WriteText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}
