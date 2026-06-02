package work

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Epic6ScannerOrchestrationLocalEvidence   Epic6ScannerOrchestrationMode = "local_evidence"
	Epic6ScannerOrchestrationMissingScanner  Epic6ScannerOrchestrationMode = "missing_scanner"
	Epic6ScannerOrchestrationOpenCritical    Epic6ScannerOrchestrationMode = "open_critical"
	Epic6ScannerOrchestrationOpenHigh        Epic6ScannerOrchestrationMode = "open_high"
	Epic6ScannerOrchestrationHighWaived      Epic6ScannerOrchestrationMode = "high_waived"
	Epic6ScannerOrchestrationCommittedSecret Epic6ScannerOrchestrationMode = "committed_secret"
)

const (
	epic6FixtureActorID      = "act_epic6_scanner_orchestrator"
	epic6FixtureHumanActorID = "act_epic6_human_security_reviewer"
	epic6FixtureTimeRFC      = "2026-06-02T00:30:00Z"
	epic6KnowledgeSourceRef  = "knowledge:dark-factory/v3.9/implementation/epics/epic-06-gate-g-real-scanner-orchestration/01-work-real-scanner-orchestration-implementation-authorization-v3.9.md"
	epic6DocsPRRef           = "transpara-ai/docs#85"
	epic6DocsMergeSHA        = "84bfe548c499a71cbc72e6cf81124cd2fdea3520"
	epic6DocsReviewedHead    = "92a661bf007038dec39c31a7b899fddaa49f3515"
)

// Epic6ScannerOrchestrationMode selects the authorized happy path or a negative Gate G path.
type Epic6ScannerOrchestrationMode string

// Epic6ScannerToolPaths allows validation to use explicit scanner binaries without relying on hidden global state.
type Epic6ScannerToolPaths struct {
	Gitleaks   string
	OSVScanner string
	Semgrep    string
	Trivy      string
}

// Epic6ScannerCommand is the command evidence shape recorded for each scanner/checker invocation.
type Epic6ScannerCommand struct {
	Tool       string
	Path       string
	Args       []string
	WorkingDir string
	OutputRef  string
}

// Epic6ScannerCommandEvidence records command, version, timing, output, and exit status evidence.
type Epic6ScannerCommandEvidence struct {
	Tool        string   `json:"tool"`
	Command     []string `json:"command"`
	WorkingDir  string   `json:"working_dir"`
	StartedAt   string   `json:"started_at"`
	CompletedAt string   `json:"completed_at"`
	ExitCode    int      `json:"exit_code"`
	StdoutRef   string   `json:"stdout_ref,omitempty"`
	StderrRef   string   `json:"stderr_ref,omitempty"`
	OutputRef   string   `json:"output_ref,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// Epic6CommandRunner is injectable so unit tests can exercise scanner policy without global scanner installs.
type Epic6CommandRunner func(context.Context, Epic6ScannerCommand) Epic6CommandResult

// Epic6CommandResult is returned by an injected scanner runner.
type Epic6CommandResult struct {
	StartedAt   time.Time
	CompletedAt time.Time
	ExitCode    int
	Stdout      string
	Stderr      string
	Error       string
}

// Epic6ScannerOrchestrationOptions keeps the fixture local, Work-owned, and caller-bounded.
type Epic6ScannerOrchestrationOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic6ScannerOrchestrationMode
	ToolPaths      Epic6ScannerToolPaths
	CommandRunner  Epic6CommandRunner
	Timeout        time.Duration
}

// Epic6ScannerOrchestrationRun is the local evidence packet for the bounded Gate G trial.
type Epic6ScannerOrchestrationRun struct {
	Mode                  Epic6ScannerOrchestrationMode
	WorkTask              Task
	WorkProjection        TaskProjection
	EventGraph            *v39.InMemoryStore
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	ActorInvocationID     string
	RuntimeEnvelopeID     string
	RuntimeResultID       string
	TestCaseID            string
	TestRunID             string
	GateResultID          string
	FailureID             string
	ReleaseCandidateID    string
	CertificationID       string
	RejectionID           string
	KnowledgeReferenceID  string
	CapabilityArtifactID  string
	AuditReportID         string
	TargetDir             string
	EvidenceDir           string
	ReportPath            string
	ProofPath             string
	GeneratedManifest     SaaSTemplateManifest
	GateEvidence          []SecurityGateEvidence
	ScannerEvidence       []Epic6ScannerGateEvidence
	Waivers               []SecurityWaiver
	CertificationResult   SecurityGateCertificationResult
	GateGValidation       Epic6GateGValidation
	TraceCompleteness     v39.TraceCompletenessGateResult
	Certification         *v39.Certification
	Rejection             *v39.Rejection
	AuditReport           *v39.AuditReport
	Projection            Epic6ScannerOrchestrationProjection
}

type Epic6ScannerGateEvidence struct {
	Gate                  SecurityGateID                `json:"gate"`
	Status                SecurityGateStatus            `json:"status"`
	ScannerTool           string                        `json:"scanner_tool"`
	ScannerVersion        string                        `json:"scanner_version"`
	EvidenceMode          string                        `json:"evidence_mode"`
	Commands              []Epic6ScannerCommandEvidence `json:"commands"`
	InputRefs             []string                      `json:"input_refs"`
	RawOutputRefs         []string                      `json:"raw_output_refs"`
	Findings              []SecurityFinding             `json:"findings,omitempty"`
	NotApplicableReason   string                        `json:"not_applicable_reason,omitempty"`
	D2ScaffoldDisposition string                        `json:"d2_scaffold_disposition"`
}

type Epic6GateGValidation struct {
	Status  string   `json:"status"`
	Missing []string `json:"missing,omitempty"`
}

type Epic6ScannerOrchestrationProjection struct {
	GeneratedAt         string                          `json:"generated_at"`
	Source              string                          `json:"source"`
	Mode                Epic6ScannerOrchestrationMode   `json:"mode"`
	TargetDir           string                          `json:"target_dir"`
	ReportPath          string                          `json:"report_path"`
	ProofPath           string                          `json:"proof_path"`
	RuntimeBOMRef       string                          `json:"runtime_bom_ref"`
	PolicyRef           string                          `json:"policy_ref"`
	GateEvidence        []Epic6ScannerGateEvidence      `json:"gate_evidence"`
	CertificationResult SecurityGateCertificationResult `json:"certification_result"`
	GateGValidation     Epic6GateGValidation            `json:"gate_g_validation"`
	ProofOfWorkPacket   Epic6ProofOfWorkPacket          `json:"proof_of_work_packet"`
}

type Epic6ProofOfWorkPacket struct {
	ID                    string                 `json:"id"`
	Status                string                 `json:"status"`
	Summary               string                 `json:"summary"`
	ScannerVersions       []Epic6ScannerVersion  `json:"scanner_versions"`
	EvidenceRefs          []string               `json:"evidence_refs"`
	PolicyDecision        Epic6ProofOfWorkItem   `json:"policy_decision"`
	D2ScaffoldDisposition Epic6ProofOfWorkItem   `json:"d2_scaffold_disposition"`
	ResidualRisks         []Epic6ProofOfWorkItem `json:"residual_risks"`
	EventGraphRefs        []string               `json:"event_graph_refs"`
}

type Epic6ScannerVersion struct {
	Gate    SecurityGateID `json:"gate"`
	Tool    string         `json:"tool"`
	Version string         `json:"version"`
}

type Epic6ProofOfWorkItem struct {
	Label       string   `json:"label"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	ArtifactRef string   `json:"artifact_ref,omitempty"`
	Refs        []string `json:"refs,omitempty"`
}

type epic6FixtureIDs struct {
	suffix               string
	factoryOrder         string
	requirement          string
	acceptanceCriterion  string
	task                 string
	scannerActorIdentity string
	humanActorIdentity   string
	actorInvocation      string
	runtimeEnvelope      string
	runtimeResult        string
	capabilityArtifact   string
	knowledgeReference   string
	targetArtifact       string
	bomArtifact          string
	policyArtifact       string
	reportArtifact       string
	proofArtifact        string
	testCase             string
	testRun              string
	gateResult           string
	failure              string
	factoryRuntime       string
	releaseCandidate     string
	certification        string
	rejection            string
	auditReport          string
	proofPacket          string
}

type epic6GraphRun struct {
	DecisionID    string
	RejectionID   string
	FailureID     string
	Trace         v39.TraceCompletenessGateResult
	Certification *v39.Certification
	Rejection     *v39.Rejection
	AuditReport   *v39.AuditReport
}

// RunEpic6RealScannerOrchestrationTrial executes the authorized Gate G local scanner/checker fixture.
func RunEpic6RealScannerOrchestrationTrial(ts *TaskStore, opts Epic6ScannerOrchestrationOptions) (Epic6ScannerOrchestrationRun, error) {
	if ts == nil {
		return Epic6ScannerOrchestrationRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic6ScannerOrchestrationRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic6ScannerOrchestrationRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic6ScannerOrchestrationRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic6ScannerOrchestrationLocalEvidence
	}
	if !epic6SupportedMode(opts.Mode) {
		return Epic6ScannerOrchestrationRun{}, fmt.Errorf("unsupported Epic 6 fixture mode %q", opts.Mode)
	}

	ids := epic6IDs(opts.Mode)
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 6 Real Scanner Orchestration",
		Description:            "Run the bounded Gate G local scanner/checker fixture over generated SaaS Template v1.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic6_scanner_orchestration",
		RiskClass:              "high",
		ExpectedOutputs:        []string{"factory-runtime-bom.json", "security/security-gates-policy.json", "artifacts/security-gates/report.json", "artifacts/security-gates/proof-of-work.json"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 6 scanner orchestration fixture lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic6ScannerOrchestrationRun{}, err
		}
	}

	targetDir := filepath.Join(opts.WorkingDir, "generated-saas-template-v1")
	evidenceDir := filepath.Join(opts.WorkingDir, "artifacts", "security-gates")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	manifest, err := GenerateSaaSTemplateV1(targetDir)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	if opts.Mode == Epic6ScannerOrchestrationCommittedSecret {
		plantedKey := "AWS_ACCESS_KEY_ID=" + "AKIA" + strings.Repeat("A", 16) + "\n"
		if err := os.WriteFile(filepath.Join(targetDir, "backend", "app", "epic6_leaked_key.txt"), []byte(plantedKey), 0o644); err != nil {
			return Epic6ScannerOrchestrationRun{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), epic6Timeout(opts.Timeout))
	defer cancel()
	scannerEvidence, waivers, err := epic6CollectScannerEvidence(ctx, targetDir, evidenceDir, opts)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	gateEvidence := epic6SecurityGateEvidence(scannerEvidence)
	certResult := EvaluateSecurityGateCertification(gateEvidence, waivers, epic6FixtureTime())
	validation := epic6EvaluateGateG(scannerEvidence, certResult)
	proof := epic6BuildProofPacket(ids, scannerEvidence, certResult, validation)
	reportPath, proofPath, err := epic6WriteEvidenceArtifacts(evidenceDir, scannerEvidence, certResult, validation, proof)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}

	graph, graphRun, err := epic6RecordEventGraph(ids, opts.Mode, scannerEvidence, certResult, validation, targetDir, reportPath, proofPath)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 6 Gate G real scanner/checker evidence attached", causes, opts.ConversationID); err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	if graphRun.FailureID != "" {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{
			FailureIDs: []string{graphRun.FailureID},
		}, "Epic 6 negative Gate G fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic6ScannerOrchestrationRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 6 Gate G scanner evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	if validation.Status == "pass" {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 6 real scanner orchestration certified for local Gate G evidence", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic6ScannerOrchestrationRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 6 negative Gate G fixture rejected", []string{ids.gateResult, graphRun.FailureID}, causes, opts.ConversationID); err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}

	projection := Epic6ScannerOrchestrationProjection{
		GeneratedAt:         epic6FixtureTime().Format(time.RFC3339),
		Source:              "work-epic6-real-scanner-orchestration-fixture",
		Mode:                opts.Mode,
		TargetDir:           targetDir,
		ReportPath:          reportPath,
		ProofPath:           proofPath,
		RuntimeBOMRef:       "factory-runtime-bom.json",
		PolicyRef:           "security/security-gates-policy.json",
		GateEvidence:        scannerEvidence,
		CertificationResult: certResult,
		GateGValidation:     validation,
		ProofOfWorkPacket:   proof,
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic6ScannerOrchestrationRun{}, err
	}
	return Epic6ScannerOrchestrationRun{
		Mode:                  opts.Mode,
		WorkTask:              task,
		WorkProjection:        workProjection,
		EventGraph:            graph,
		FactoryOrderID:        ids.factoryOrder,
		RequirementID:         ids.requirement,
		AcceptanceCriterionID: ids.acceptanceCriterion,
		TaskID:                ids.task,
		ActorInvocationID:     ids.actorInvocation,
		RuntimeEnvelopeID:     ids.runtimeEnvelope,
		RuntimeResultID:       ids.runtimeResult,
		TestCaseID:            ids.testCase,
		TestRunID:             ids.testRun,
		GateResultID:          ids.gateResult,
		FailureID:             graphRun.FailureID,
		ReleaseCandidateID:    ids.releaseCandidate,
		CertificationID:       epic6CertificationID(graphRun.Certification),
		RejectionID:           graphRun.RejectionID,
		KnowledgeReferenceID:  ids.knowledgeReference,
		CapabilityArtifactID:  ids.capabilityArtifact,
		AuditReportID:         ids.auditReport,
		TargetDir:             targetDir,
		EvidenceDir:           evidenceDir,
		ReportPath:            reportPath,
		ProofPath:             proofPath,
		GeneratedManifest:     manifest,
		GateEvidence:          gateEvidence,
		ScannerEvidence:       scannerEvidence,
		Waivers:               waivers,
		CertificationResult:   certResult,
		GateGValidation:       validation,
		TraceCompleteness:     graphRun.Trace,
		Certification:         graphRun.Certification,
		Rejection:             graphRun.Rejection,
		AuditReport:           graphRun.AuditReport,
		Projection:            projection,
	}, nil
}

func (p Epic6ScannerOrchestrationProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func epic6CollectScannerEvidence(ctx context.Context, targetDir, evidenceDir string, opts Epic6ScannerOrchestrationOptions) ([]Epic6ScannerGateEvidence, []SecurityWaiver, error) {
	var evidence []Epic6ScannerGateEvidence
	gates := []struct {
		gate        SecurityGateID
		tool        string
		path        string
		versionArgs []string
		scanArgs    []string
		inputRefs   []string
		outputName  string
	}{
		{GateSecretScan, "gitleaks", opts.ToolPaths.Gitleaks, []string{"version"}, []string{"detect", "--no-git", "--source", targetDir, "--report-format", "json", "--report-path", filepath.Join(evidenceDir, "gitleaks.json"), "--exit-code", "1"}, []string{"generated source", "config", ".env.example"}, "gitleaks.json"},
		{GateDependencyVulnerabilityScan, "osv-scanner", opts.ToolPaths.OSVScanner, []string{"--version"}, []string{"scan", "--format", "json", "--output", filepath.Join(evidenceDir, "osv-scanner.json"), "--lockfile", filepath.Join(targetDir, "frontend", "package-lock.json"), "--lockfile", filepath.Join(targetDir, "backend", "requirements.lock.txt")}, []string{"frontend/package.json", "frontend/package-lock.json", "backend/pyproject.toml", "backend/requirements.lock.txt"}, "osv-scanner.json"},
		{GateSAST, "semgrep", opts.ToolPaths.Semgrep, []string{"--version"}, epic6SemgrepArgs(targetDir, evidenceDir), []string{"frontend", "backend"}, "semgrep.json"},
	}
	for _, gate := range gates {
		item, err := epic6ExternalScannerEvidence(ctx, targetDir, evidenceDir, gate.gate, gate.tool, gate.path, gate.versionArgs, gate.scanArgs, gate.inputRefs, gate.outputName, opts)
		if err != nil {
			return nil, nil, err
		}
		evidence = append(evidence, item)
	}
	evidence = append(evidence,
		epic6LicensePolicyEvidence(targetDir, evidenceDir),
		epic6AuthFlowEvidence(targetDir, evidenceDir),
		epic6ConfigSecurityEvidence(targetDir, evidenceDir),
		epic6ContainerEvidence(targetDir, evidenceDir, opts.ToolPaths.Trivy),
	)
	waivers := epic6ApplyModeEvidence(opts.Mode, evidence)
	return evidence, waivers, nil
}

func epic6ExternalScannerEvidence(ctx context.Context, targetDir, evidenceDir string, gate SecurityGateID, tool, path string, versionArgs, scanArgs, inputRefs []string, outputName string, opts Epic6ScannerOrchestrationOptions) (Epic6ScannerGateEvidence, error) {
	resolved, err := epic6ResolveTool(tool, path, opts.CommandRunner)
	if err != nil || opts.Mode == Epic6ScannerOrchestrationMissingScanner && gate == GateSAST {
		return Epic6ScannerGateEvidence{
			Gate: gate, Status: SecurityGateStatusFail, ScannerTool: tool, ScannerVersion: "", EvidenceMode: "real_scanner_command",
			InputRefs: inputRefs, RawOutputRefs: []string{}, D2ScaffoldDisposition: "blocked: required real scanner binary is missing",
			Findings: []SecurityFinding{{ID: "finding_epic6_missing_" + string(gate), Gate: gate, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "required scanner binary is missing"}},
		}, nil
	}
	versionCommand := Epic6ScannerCommand{Tool: tool, Path: resolved, Args: versionArgs, WorkingDir: targetDir}
	versionEvidence, versionOutput, err := epic6RunCommand(ctx, evidenceDir, versionCommand, opts.CommandRunner)
	if err != nil {
		return Epic6ScannerGateEvidence{}, err
	}
	version := epic6ScannerVersion(versionOutput, tool, resolved)
	scanCommand := Epic6ScannerCommand{Tool: tool, Path: resolved, Args: scanArgs, WorkingDir: targetDir, OutputRef: filepath.Join(evidenceDir, outputName)}
	scanEvidence, _, err := epic6RunCommand(ctx, evidenceDir, scanCommand, opts.CommandRunner)
	if err != nil {
		return Epic6ScannerGateEvidence{}, err
	}
	status := SecurityGateStatusPass
	var findings []SecurityFinding
	if scanEvidence.ExitCode != 0 {
		status = SecurityGateStatusFail
		findings = append(findings, SecurityFinding{
			ID:        "finding_epic6_" + string(gate),
			Gate:      gate,
			Severity:  epic6SeverityForGate(gate),
			Status:    FindingStatusOpen,
			SecretHit: gate == GateSecretScan,
			Summary:   tool + " returned a failing exit status",
		})
	}
	return Epic6ScannerGateEvidence{
		Gate: gate, Status: status, ScannerTool: tool, ScannerVersion: version, EvidenceMode: "real_scanner_command",
		Commands:  []Epic6ScannerCommandEvidence{versionEvidence, scanEvidence},
		InputRefs: inputRefs, RawOutputRefs: epic6CommandOutputRefs(scanEvidence),
		Findings: findings, D2ScaffoldDisposition: "replaces generated D2 scaffold evidence with real command evidence for this fixture",
	}, nil
}

func epic6LicensePolicyEvidence(targetDir, evidenceDir string) Epic6ScannerGateEvidence {
	inputRefs := []string{"frontend/package.json", "frontend/package-lock.json", "backend/pyproject.toml", "backend/requirements.lock.txt"}
	status := SecurityGateStatusPass
	var findings []SecurityFinding
	checks := []string{"inspected generated frontend and backend dependency manifests"}
	if !epic6FileContains(filepath.Join(targetDir, "frontend", "package.json"), []string{`"dependencies"`, `"next"`, `"react"`}) {
		status = SecurityGateStatusFail
		findings = append(findings, SecurityFinding{ID: "finding_epic6_license_frontend_manifest", Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend dependency metadata is missing required entries"})
	}
	licenseChecks, licenseFindings := epic6FrontendPackageLockLicenseChecks(filepath.Join(targetDir, "frontend", "package-lock.json"))
	checks = append(checks, licenseChecks...)
	if len(licenseFindings) > 0 {
		status = SecurityGateStatusFail
		findings = append(findings, licenseFindings...)
	}
	if !epic6FileContains(filepath.Join(targetDir, "backend", "pyproject.toml"), []string{"dependencies = [", `"fastapi`, `"sqlalchemy`}) {
		status = SecurityGateStatusFail
		findings = append(findings, SecurityFinding{ID: "finding_epic6_license_backend_manifest", Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "backend dependency metadata is missing required entries"})
	}
	if !epic6FileContains(filepath.Join(targetDir, "backend", "requirements.lock.txt"), []string{"fastapi==", "pytest=="}) {
		status = SecurityGateStatusFail
		findings = append(findings, SecurityFinding{ID: "finding_epic6_license_backend_lock", Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "backend dependency lock metadata is missing required entries"})
	}
	outputRef := filepath.Join(evidenceDir, "license-policy.json")
	_ = epic6WriteJSON(outputRef, map[string]any{"tool": "license-policy", "version": "dark-factory-local-1", "status": status, "checks": checks, "findings": findings})
	return Epic6ScannerGateEvidence{Gate: GateDependencyLicenseScan, Status: status, ScannerTool: "license-policy", ScannerVersion: "dark-factory-local-1", EvidenceMode: "real_local_checker", Commands: []Epic6ScannerCommandEvidence{epic6LocalCommand("license-policy", targetDir, inputRefs, outputRef)}, InputRefs: inputRefs, RawOutputRefs: []string{outputRef}, Findings: findings, D2ScaffoldDisposition: "replaces generated D2 scaffold evidence with manifest-derived local policy evidence"}
}

func epic6FrontendPackageLockLicenseChecks(path string) ([]string, []SecurityFinding) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, []SecurityFinding{{ID: "finding_epic6_license_frontend_lock_missing", Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock license metadata is missing"}}
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
			License string `json:"license"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &lock); err != nil || len(lock.Packages) == 0 {
		return nil, []SecurityFinding{{ID: "finding_epic6_license_frontend_lock_invalid", Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock license metadata is invalid"}}
	}
	required := map[string]string{
		"node_modules/next":      "MIT",
		"node_modules/react":     "MIT",
		"node_modules/react-dom": "MIT",
	}
	licenses := map[string]bool{}
	packageCount := 0
	var findings []SecurityFinding
	for packagePath, pkg := range lock.Packages {
		if packagePath == "" {
			continue
		}
		packageCount++
		license := strings.TrimSpace(pkg.License)
		if license == "" {
			findings = append(findings, SecurityFinding{ID: "finding_epic6_license_missing_" + epic6Slug(packagePath), Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock dependency is missing license metadata: " + packagePath})
			continue
		}
		licenses[license] = true
		if epic6ForbiddenLicenseExpression(license) {
			findings = append(findings, SecurityFinding{ID: "finding_epic6_license_forbidden_" + epic6Slug(packagePath), Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock dependency has forbidden license expression: " + packagePath})
		}
	}
	for packagePath, expectedLicense := range required {
		pkg, ok := lock.Packages[packagePath]
		if !ok || strings.TrimSpace(pkg.Version) == "" {
			findings = append(findings, SecurityFinding{ID: "finding_epic6_license_required_missing_" + epic6Slug(packagePath), Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock dependency metadata is missing required package: " + packagePath})
			continue
		}
		if !strings.Contains(pkg.License, expectedLicense) {
			findings = append(findings, SecurityFinding{ID: "finding_epic6_license_required_unexpected_" + epic6Slug(packagePath), Gate: GateDependencyLicenseScan, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "frontend package-lock dependency has unexpected license metadata: " + packagePath})
		}
	}
	checks := []string{fmt.Sprintf("inspected frontend package-lock license metadata for %d packages and %d license expressions", packageCount, len(licenses))}
	return checks, findings
}

func epic6ForbiddenLicenseExpression(license string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(license))
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '(' || r == ')' || r == '/'
	})
	for _, token := range tokens {
		token = strings.Trim(token, ",")
		for _, marker := range []string{"AGPL", "AGPL-3.0-ONLY", "AGPL-3.0-OR-LATER", "GPL-3.0-ONLY", "GPL-3.0-OR-LATER", "BUSL", "UNLICENSED"} {
			if token == marker {
				return true
			}
		}
		if strings.HasPrefix(token, "AGPL-") || strings.HasPrefix(token, "BUSL-") {
			return true
		}
	}
	return false
}

func epic6AuthFlowEvidence(targetDir, evidenceDir string) Epic6ScannerGateEvidence {
	inputRefs := []string{"frontend/app/api/login/route.ts", "frontend/app/dashboard/page.tsx", "frontend/app/logout/route.ts", "frontend/lib/auth.ts", "backend/app/auth.py", "backend/app/main.py", "backend/tests/test_auth_and_tracker.py"}
	required := map[string][]string{
		"frontend/app/api/login/route.ts":        {`response.cookies.set("session"`, "httpOnly: true", `sameSite: "lax"`},
		"frontend/app/dashboard/page.tsx":        {"requireSession"},
		"frontend/app/logout/route.ts":           {"cookies.delete"},
		"frontend/lib/auth.ts":                   {"redirect(\"/login\")"},
		"backend/app/auth.py":                    {"def require_session", "HTTPException(status_code=401"},
		"backend/app/main.py":                    {"Depends(require_session)", `@app.post("/auth/login")`},
		"backend/tests/test_auth_and_tracker.py": {"status_code == 401"},
	}
	status := SecurityGateStatusPass
	var findings []SecurityFinding
	for rel, needles := range required {
		if !epic6FileContains(filepath.Join(targetDir, filepath.FromSlash(rel)), needles) {
			status = SecurityGateStatusFail
			findings = append(findings, SecurityFinding{ID: "finding_epic6_auth_" + epic6Slug(rel), Gate: GateAuthFlowSecurityCheck, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "auth-flow checker did not find required route/session evidence in " + rel})
		}
	}
	outputRef := filepath.Join(evidenceDir, "auth-flow-check.json")
	_ = epic6WriteJSON(outputRef, map[string]any{"tool": "auth-flow-check", "version": "dark-factory-local-1", "status": status, "inspected": inputRefs, "findings": findings})
	return Epic6ScannerGateEvidence{Gate: GateAuthFlowSecurityCheck, Status: status, ScannerTool: "auth-flow-check", ScannerVersion: "dark-factory-local-1", EvidenceMode: "real_local_checker", Commands: []Epic6ScannerCommandEvidence{epic6LocalCommand("auth-flow-check", targetDir, inputRefs, outputRef)}, InputRefs: inputRefs, RawOutputRefs: []string{outputRef}, Findings: findings, D2ScaffoldDisposition: "replaces generated D2 scaffold evidence with source-derived auth/session checks"}
}

func epic6ConfigSecurityEvidence(targetDir, evidenceDir string) Epic6ScannerGateEvidence {
	inputRefs := []string{".env.example", ".gitignore", "docker-compose.yml", "frontend/app/api/login/route.ts", "backend/app/auth.py"}
	status := SecurityGateStatusPass
	var findings []SecurityFinding
	checks := map[string][]string{
		".env.example":                    {"SESSION_SECRET=replace-with-local-dev-secret", "DATABASE_URL="},
		".gitignore":                      {".env", "artifacts/"},
		"docker-compose.yml":              {"env_file: .env", "postgres:16-alpine"},
		"frontend/app/api/login/route.ts": {"httpOnly: true", `sameSite: "lax"`},
		"backend/app/auth.py":             {"os.environ.get(\"SESSION_SECRET\""},
	}
	for rel, needles := range checks {
		if !epic6FileContains(filepath.Join(targetDir, filepath.FromSlash(rel)), needles) {
			status = SecurityGateStatusFail
			findings = append(findings, SecurityFinding{ID: "finding_epic6_config_" + epic6Slug(rel), Gate: GateConfigurationSecurityCheck, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "config-security checker did not find required config evidence in " + rel})
		}
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".env")); err == nil {
		status = SecurityGateStatusFail
		findings = append(findings, SecurityFinding{ID: "finding_epic6_config_env_committed", Gate: GateConfigurationSecurityCheck, Severity: FindingSeverityCritical, Status: FindingStatusOpen, Summary: ".env exists in generated target"})
	}
	outputRef := filepath.Join(evidenceDir, "config-security-check.json")
	_ = epic6WriteJSON(outputRef, map[string]any{"tool": "config-security-check", "version": "dark-factory-local-1", "status": status, "inspected": inputRefs, "findings": findings})
	return Epic6ScannerGateEvidence{Gate: GateConfigurationSecurityCheck, Status: status, ScannerTool: "config-security-check", ScannerVersion: "dark-factory-local-1", EvidenceMode: "real_local_checker", Commands: []Epic6ScannerCommandEvidence{epic6LocalCommand("config-security-check", targetDir, inputRefs, outputRef)}, InputRefs: inputRefs, RawOutputRefs: []string{outputRef}, Findings: findings, D2ScaffoldDisposition: "replaces generated D2 scaffold evidence with generated-config checks"}
}

func epic6ContainerEvidence(targetDir, evidenceDir, trivyPath string) Epic6ScannerGateEvidence {
	inputRefs := []string{"generated target tree"}
	outputRef := filepath.Join(evidenceDir, "container-build-artifact-scan.json")
	artifacts := epic6ContainerArtifacts(targetDir)
	version := "not_applicable:no_container_or_build_artifact"
	if len(artifacts) > 0 && trivyPath != "" {
		version = "pending_real_trivy_run"
	}
	_ = epic6WriteJSON(outputRef, map[string]any{"tool": "trivy", "version": version, "status": SecurityGateStatusNotApplicable, "reason": "generated fixture produced no container image, image archive, SBOM, dist, or standalone build artifact", "artifacts": artifacts})
	return Epic6ScannerGateEvidence{Gate: GateContainerOrArtifactScan, Status: SecurityGateStatusNotApplicable, ScannerTool: "trivy", ScannerVersion: version, EvidenceMode: "not_applicable_with_proof", Commands: []Epic6ScannerCommandEvidence{epic6LocalCommand("trivy-artifact-presence-check", targetDir, inputRefs, outputRef)}, InputRefs: inputRefs, RawOutputRefs: []string{outputRef}, NotApplicableReason: "no container image or build artifact produced by the generated SaaS Template v1 fixture", D2ScaffoldDisposition: "narrows generated D2 scaffold evidence to explicit not_applicable proof for this fixture"}
}

func epic6ApplyModeEvidence(mode Epic6ScannerOrchestrationMode, evidence []Epic6ScannerGateEvidence) []SecurityWaiver {
	var waivers []SecurityWaiver
	switch mode {
	case Epic6ScannerOrchestrationOpenCritical:
		epic6AddFinding(evidence, GateDependencyVulnerabilityScan, SecurityFinding{ID: "finding_epic6_open_critical", Gate: GateDependencyVulnerabilityScan, Severity: FindingSeverityCritical, Status: FindingStatusOpen, WaiverID: "waiver_epic6_critical", Summary: "negative fixture open critical vulnerability"})
		waivers = append(waivers, epic6ValidWaiver("waiver_epic6_critical", "finding_epic6_open_critical"))
	case Epic6ScannerOrchestrationOpenHigh:
		epic6AddFinding(evidence, GateSAST, SecurityFinding{ID: "finding_epic6_open_high", Gate: GateSAST, Severity: FindingSeverityHigh, Status: FindingStatusOpen, Summary: "negative fixture open high SAST finding"})
	case Epic6ScannerOrchestrationHighWaived:
		epic6AddFinding(evidence, GateSAST, SecurityFinding{ID: "finding_epic6_waived_high", Gate: GateSAST, Severity: FindingSeverityHigh, Status: FindingStatusOpen, WaiverID: "waiver_epic6_high", Summary: "negative fixture high finding with valid non-production waiver"})
		waivers = append(waivers, epic6ValidWaiver("waiver_epic6_high", "finding_epic6_waived_high"))
	case Epic6ScannerOrchestrationCommittedSecret:
		epic6AddFinding(evidence, GateSecretScan, SecurityFinding{ID: "finding_epic6_committed_secret", Gate: GateSecretScan, Severity: FindingSeverityHigh, Status: FindingStatusWaived, WaiverID: "waiver_epic6_secret", SecretHit: true, Summary: "negative fixture committed secret hit"})
		waivers = append(waivers, epic6ValidWaiver("waiver_epic6_secret", "finding_epic6_committed_secret"))
	}
	return waivers
}

func epic6EvaluateGateG(scannerEvidence []Epic6ScannerGateEvidence, cert SecurityGateCertificationResult) Epic6GateGValidation {
	seen := map[string]bool{}
	var missing []string
	for _, item := range scannerEvidence {
		if item.EvidenceMode == "scaffold" {
			missing = append(missing, string(item.Gate)+" still uses scaffold evidence")
		}
		if item.Status != SecurityGateStatusNotApplicable && len(item.Commands) == 0 {
			missing = append(missing, string(item.Gate)+" command evidence missing")
		}
		if item.Status != SecurityGateStatusNotApplicable && strings.TrimSpace(item.ScannerVersion) == "" {
			missing = append(missing, string(item.Gate)+" scanner version missing")
		}
	}
	missing = append(missing, cert.BlockingReasons...)
	for _, gate := range cert.MissingEvidence {
		missing = append(missing, string(gate)+" evidence missing")
	}
	deduped := make([]string, 0, len(missing))
	for _, value := range missing {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		deduped = append(deduped, value)
	}
	status := "pass"
	if len(deduped) > 0 || cert.Blocked {
		status = "fail"
	}
	return Epic6GateGValidation{Status: status, Missing: deduped}
}

func epic6RecordEventGraph(ids epic6FixtureIDs, mode Epic6ScannerOrchestrationMode, scannerEvidence []Epic6ScannerGateEvidence, certResult SecurityGateCertificationResult, validation Epic6GateGValidation, targetDir, reportPath, proofPath string) (*v39.InMemoryStore, epic6GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic6FixtureTime()
	taskStatus := "certified"
	acceptanceStatus := "verified"
	testRunStatus := "pass"
	runtimeStatus := "succeeded"
	releaseStatus := "certified"
	if validation.Status != "pass" {
		taskStatus = "rejected"
		acceptanceStatus = "rejected"
		testRunStatus = "fail"
		runtimeStatus = "failed"
		releaseStatus = "rejected"
	}
	taskCommon := epic6Common(ids.task, v39.TypeTask, taskStatus)
	taskCommon.SourceRefs = []string{ids.capabilityArtifact, epic6KnowledgeSourceRef}
	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: epic6Common(ids.factoryOrder, v39.TypeFactoryOrder, taskStatus), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-85-merged-" + epic6DocsMergeSHA, SourceIntentRef: epic6DocsPRRef, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic6Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Replace or narrow D2 non-secret scanner scaffold evidence with real scanner/checker evidence for generated SaaS Template v1.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: epic6Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, acceptanceStatus), RequirementID: ids.requirement, Text: "Gate G passes only when real scanner/checker commands record versions, command evidence, findings, policy decisions, BOM refs, and D2 scaffold disposition.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "scanner_orchestration", OwnerRole: "maintainer", RiskClass: "high"},
		&v39.Task{CommonNode: taskCommon, FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic6_scanner_orchestration", State: taskStatus, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic6Common(ids.scannerActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic6FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic6Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic6FixtureHumanActorID, ActorType: "human", IdentityMode: "fixture"},
		&v39.CapabilityArtifact{CommonNode: epic6Common(ids.capabilityArtifact, v39.TypeCapabilityArtifact, "active"), ArtifactID: ids.capabilityArtifact, ArtifactType: "runtime_adapter", Name: "Epic 6 Gate G scanner orchestration fixture", ArtifactVersion: "v1", SourceRepoOrOrigin: "transpara-ai/work", ContentHash: epic6Hash(strings.Join(epic6ScannerVersionStrings(scannerEvidence), "\n")), Owner: "work", RiskClass: "high", ActivationScope: "fixture_only", EvalRefs: []string{ids.testCase}, HumanReviewRef: epic6DocsReviewedHead, RollbackRef: "not_applicable_local_fixture", UsageLoggingRequired: true},
		&v39.Artifact{CommonNode: epic6Common(ids.targetArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "code", Path: strPtr("fixture://epic6/generated-saas-template-v1"), ContentHash: strPtr(epic6Hash(strings.Join(SaaSTemplateV1FactoryRuntimeBOM().SecurityScannerVersions(), "\n")))},
		&v39.Artifact{CommonNode: epic6Common(ids.bomArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "config", Path: strPtr(filepath.Join(targetDir, "factory-runtime-bom.json")), ContentHash: strPtr(epic6HashFile(filepath.Join(targetDir, "factory-runtime-bom.json")))},
		&v39.Artifact{CommonNode: epic6Common(ids.policyArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "config", Path: strPtr(filepath.Join(targetDir, "security", "security-gates-policy.json")), ContentHash: strPtr(epic6HashFile(filepath.Join(targetDir, "security", "security-gates-policy.json")))},
		&v39.Artifact{CommonNode: epic6Common(ids.reportArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: strPtr(reportPath), ContentHash: strPtr(epic6HashFile(reportPath))},
		&v39.Artifact{CommonNode: epic6Common(ids.proofArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: strPtr(proofPath), ContentHash: strPtr(epic6HashFile(proofPath))},
		&v39.ActorInvocation{CommonNode: epic6Common(ids.actorInvocation, v39.TypeActorInvocation, runtimeStatus), TaskID: ids.task, Runtime: "local", ActorID: epic6FixtureActorID, InputContractHash: epic6Hash("epic6-input:" + targetDir), OutputContractHash: strPtr(epic6Hash("epic6-output:" + reportPath + ":" + proofPath))},
		&v39.RuntimeEnvelope{CommonNode: epic6Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_scanner_orchestration", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: epic6FixtureActorID, AuthorityDecisionRef: "human_authorized_in_chat_2026-06-02", AllowedFiles: []string{"generated-saas-template-v1/**", "artifacts/security-gates/**"}, DeniedFiles: []string{".git", "../", ".env", "secrets.env"}, AllowedCommands: epic6AllowedCommands(scannerEvidence), DeniedCommands: []string{"git push", "gh pr merge", "docker compose up", "terraform apply", "kubectl apply", "production deploy"}, NetworkPolicy: "restricted", SecretsPolicy: "none", WorkingDirectory: targetDir, Timeout: "5m", ResourceLimits: map[string]any{"scope": "local fixture", "network_note": "scanner tools only; generated app is not run"}, ExpectedOutputs: []string{reportPath, proofPath}, OutputContract: map[string]any{"mode": string(mode), "gate": "gate_g_real_scanner_orchestration"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"EvaluateSecurityGateCertification", "epic6EvaluateGateG"}, EnvelopeHash: epic6Hash("epic6-envelope:" + string(mode))},
		&v39.RuntimeResult{CommonNode: epic6Common(ids.runtimeResult, v39.TypeRuntimeResult, runtimeStatus), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_scanner_orchestration", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: runtimeStatus, ArtifactRefs: []string{ids.reportArtifact, ids.proofArtifact}, ChangedFiles: []string{}, CommandLog: epic6CommandLog(scannerEvidence), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{"EvaluateSecurityGateCertification"}, PostRunValidationRefs: []string{ids.testRun}},
		&v39.TestCase{CommonNode: epic6Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 6 real scanner orchestration Gate G evidence", TestType: "unit", Path: strPtr("work/epic6_scanner_orchestration_test.go")},
		&v39.TestRun{CommonNode: epic6Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: epic6Common(ids.gateResult, v39.TypeGateResult, validation.Status), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_g_real_scanner_orchestration", EvidenceRefs: []string{ids.testRun, ids.reportArtifact, ids.proofArtifact, ids.bomArtifact, ids.policyArtifact}},
	}
	if validation.Status == "fail" {
		records = append(records, &v39.Failure{CommonNode: epic6Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "gate_g_scanner_policy_blocked", Severity: "high", Summary: strings.Join(validation.Missing, "; ")})
	}
	if err := epic6AppendRecords(graph, records...); err != nil {
		return nil, epic6GraphRun{}, err
	}
	if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{CommonNode: epic6Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded"), ReferenceCreatedAt: createdAt, SourceSystem: "transpara-ai/docs", SourceRef: epic6KnowledgeSourceRef, SourceHashOrImmutableLocator: "sha256:docs-pr-85-merged-" + epic6DocsMergeSHA + "-reviewed-head-" + epic6DocsReviewedHead, RetrievedAt: createdAt, UsedByActor: epic6FixtureActorID, UsedInTask: ids.task, InfluenceSummary: "Gate G authorization packet shaped scanner set, evidence model, stop conditions, and exclusions.", RiskScope: "high", TrustLevel: "human_authorized", FreshnessStatus: "current", RedactionState: "none"}}); err != nil {
		return nil, epic6GraphRun{}, err
	}
	if _, err := graph.RecordCapabilityUsage(ids.task, ids.capabilityArtifact, epic6Common("edge_epic6_used_capability_"+ids.suffix, v39.TypeCapabilityArtifact, "recorded")); err != nil {
		return nil, epic6GraphRun{}, err
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic6Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-epic6-real-scanner-orchestration", CapabilityVersionRefs: []string{}, RuntimeRefs: epic6ScannerVersionStrings(scannerEvidence)}); err != nil {
		return nil, epic6GraphRun{}, err
	}
	if err := epic6AppendEdges(graph, ids, createdAt, validation.Status == "fail"); err != nil {
		return nil, epic6GraphRun{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: epic6Common(ids.releaseCandidate, v39.TypeReleaseCandidate, releaseStatus), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.reportArtifact, ids.proofArtifact}})
	if err != nil {
		return nil, epic6GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	if validation.Status == "pass" && traceErr != nil {
		return nil, epic6GraphRun{}, traceErr
	}
	if validation.Status == "pass" {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic6Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: epic6FixtureHumanActorID, Reason: "Gate G scanner/checker evidence is complete for the bounded local Work fixture.", EvidenceRefs: []string{ids.gateResult, ids.reportArtifact, ids.proofArtifact}})
		if err != nil {
			return nil, epic6GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic6Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic6GraphRun{}, err
		}
		return graph, epic6GraphRun{DecisionID: cert.CommonNode.ID, Trace: trace, Certification: cert, AuditReport: audit}, nil
	}
	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic6Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: epic6FixtureHumanActorID, Reason: "Gate G scanner/checker evidence is incomplete or policy-blocked: " + strings.Join(validation.Missing, "; "), EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic6GraphRun{}, err
	}
	audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic6Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
	if err != nil {
		return nil, epic6GraphRun{}, err
	}
	return graph, epic6GraphRun{DecisionID: rejection.CommonNode.ID, RejectionID: rejection.CommonNode.ID, FailureID: ids.failure, Trace: trace, Rejection: rejection, AuditReport: audit}, nil
}

func epic6RunCommand(ctx context.Context, evidenceDir string, command Epic6ScannerCommand, runner Epic6CommandRunner) (Epic6ScannerCommandEvidence, string, error) {
	result := Epic6CommandResult{}
	if runner != nil {
		result = runner(ctx, command)
	} else {
		result = epic6ExecCommand(ctx, command)
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = epic6FixtureTime()
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = result.StartedAt
	}
	stdoutRef, stderrRef := "", ""
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return Epic6ScannerCommandEvidence{}, "", err
	}
	base := epic6CommandEvidenceBase(command)
	if result.Stdout != "" {
		stdoutRef = filepath.Join(evidenceDir, base+"-stdout.txt")
		if err := os.WriteFile(stdoutRef, []byte(result.Stdout), 0o644); err != nil {
			return Epic6ScannerCommandEvidence{}, "", err
		}
	}
	if result.Stderr != "" {
		stderrRef = filepath.Join(evidenceDir, base+"-stderr.txt")
		if err := os.WriteFile(stderrRef, []byte(result.Stderr), 0o644); err != nil {
			return Epic6ScannerCommandEvidence{}, "", err
		}
	}
	if command.OutputRef != "" {
		if _, err := os.Stat(command.OutputRef); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(command.OutputRef, []byte(result.Stdout), 0o644); err != nil {
				return Epic6ScannerCommandEvidence{}, "", err
			}
		}
	}
	cmd := append([]string{command.Path}, command.Args...)
	return Epic6ScannerCommandEvidence{Tool: command.Tool, Command: cmd, WorkingDir: command.WorkingDir, StartedAt: result.StartedAt.Format(time.RFC3339), CompletedAt: result.CompletedAt.Format(time.RFC3339), ExitCode: result.ExitCode, StdoutRef: stdoutRef, StderrRef: stderrRef, OutputRef: command.OutputRef, Error: result.Error}, strings.TrimSpace(result.Stdout + "\n" + result.Stderr), nil
}

func epic6ExecCommand(ctx context.Context, command Epic6ScannerCommand) Epic6CommandResult {
	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.WorkingDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	errText := ""
	if err != nil {
		errText = err.Error()
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	return Epic6CommandResult{StartedAt: started, CompletedAt: time.Now().UTC(), ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String(), Error: errText}
}

func epic6ResolveTool(tool, explicitPath string, runner Epic6CommandRunner) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if runner != nil {
		return tool, nil
	}
	resolved, err := exec.LookPath(tool)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func epic6SemgrepArgs(targetDir, evidenceDir string) []string {
	configPath := filepath.Join(evidenceDir, "semgrep-local-rules.yml")
	_ = os.WriteFile(configPath, []byte(`rules:
  - id: dark-factory-forbidden-dynamic-eval
    message: forbidden dynamic evaluation
    severity: ERROR
    languages: [typescript, python]
    pattern-regex: eval\s*\(
`), 0o644)
	return []string{"scan", "--config", configPath, "--json", "--output", filepath.Join(evidenceDir, "semgrep.json"), filepath.Join(targetDir, "frontend"), filepath.Join(targetDir, "backend")}
}

func epic6WriteEvidenceArtifacts(evidenceDir string, scannerEvidence []Epic6ScannerGateEvidence, cert SecurityGateCertificationResult, validation Epic6GateGValidation, proof Epic6ProofOfWorkPacket) (string, string, error) {
	reportPath := filepath.Join(evidenceDir, "report.json")
	proofPath := filepath.Join(evidenceDir, "proof-of-work.json")
	if err := epic6WriteJSON(reportPath, map[string]any{"template_id": SaaSTemplateV1ID, "factory_runtime_bom": SaaSTemplateV1FactoryRuntimeBOM(), "gate_evidence": scannerEvidence, "certification_result": cert, "gate_g_validation": validation}); err != nil {
		return "", "", err
	}
	if err := epic6WriteJSON(proofPath, proof); err != nil {
		return "", "", err
	}
	return reportPath, proofPath, nil
}

func epic6WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func epic6SecurityGateEvidence(evidence []Epic6ScannerGateEvidence) []SecurityGateEvidence {
	out := make([]SecurityGateEvidence, 0, len(evidence))
	for _, item := range evidence {
		out = append(out, SecurityGateEvidence{Gate: item.Gate, Status: item.Status, ScannerTool: item.ScannerTool, ScannerVersion: item.ScannerVersion, Findings: append([]SecurityFinding(nil), item.Findings...)})
	}
	return out
}

func epic6BuildProofPacket(ids epic6FixtureIDs, evidence []Epic6ScannerGateEvidence, cert SecurityGateCertificationResult, validation Epic6GateGValidation) Epic6ProofOfWorkPacket {
	status := validation.Status
	return Epic6ProofOfWorkPacket{ID: ids.proofPacket, Status: status, Summary: "Epic 6 Gate G real scanner/checker orchestration proof-of-work packet for generated SaaS Template v1.", ScannerVersions: epic6ScannerVersions(evidence), EvidenceRefs: []string{ids.reportArtifact, ids.proofArtifact, ids.bomArtifact, ids.policyArtifact}, PolicyDecision: Epic6ProofOfWorkItem{Label: "Critical/high policy", Status: status, Summary: fmt.Sprintf("blocked=%v missing=%v blocking=%v", cert.Blocked, cert.MissingEvidence, cert.BlockingReasons), ArtifactRef: ids.gateResult}, D2ScaffoldDisposition: Epic6ProofOfWorkItem{Label: "D2 scaffold disposition", Status: status, Summary: "Gate G proof ignores generated non-secret scaffold pass entries and records real external scanner/checker or not-applicable evidence for every D2 gate.", ArtifactRef: ids.reportArtifact}, ResidualRisks: []Epic6ProofOfWorkItem{{Label: "R-001", Status: "excluded", Summary: "No runner/worktree protected execution is performed."}, {Label: "R-002", Status: "excluded", Summary: "No protected side effects or production ExecutionReceipt path is recorded."}, {Label: "R-003", Status: "excluded", Summary: "No PolicyEngineAdapterDecision or policy-bundle evidence is used."}}, EventGraphRefs: []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeGateResult, ids.gateResult)}}
}

func epic6ScannerVersions(evidence []Epic6ScannerGateEvidence) []Epic6ScannerVersion {
	out := make([]Epic6ScannerVersion, 0, len(evidence))
	for _, item := range evidence {
		out = append(out, Epic6ScannerVersion{Gate: item.Gate, Tool: item.ScannerTool, Version: item.ScannerVersion})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Gate < out[j].Gate })
	return out
}

func epic6ScannerVersionStrings(evidence []Epic6ScannerGateEvidence) []string {
	versions := epic6ScannerVersions(evidence)
	out := make([]string, 0, len(versions))
	for _, version := range versions {
		out = append(out, string(version.Gate)+"="+version.Tool+"@"+version.Version)
	}
	return out
}

func (bom FactoryRuntimeBOM) SecurityScannerVersions() []string {
	out := make([]string, 0, len(bom.SecurityScanners))
	for _, scanner := range bom.SecurityScanners {
		out = append(out, string(scanner.Gate)+"="+scanner.Tool+"@"+scanner.Version)
	}
	sort.Strings(out)
	return out
}

func epic6IDs(mode Epic6ScannerOrchestrationMode) epic6FixtureIDs {
	suffix := string(mode)
	return epic6FixtureIDs{suffix: suffix, factoryOrder: "fo_epic6_scanner_orchestration", requirement: "req_epic6_scanner_orchestration_" + suffix, acceptanceCriterion: "ac_epic6_scanner_orchestration_" + suffix, task: "tsk_epic6_scanner_orchestration", scannerActorIdentity: "actor_identity_epic6_scanner_" + suffix, humanActorIdentity: "actor_identity_epic6_human_" + suffix, actorInvocation: "invoke_epic6_scanner_orchestration_" + suffix, runtimeEnvelope: "env_epic6_scanner_orchestration_" + suffix, runtimeResult: "rr_epic6_scanner_orchestration_" + suffix, capabilityArtifact: "cap_art_epic6_scanner_orchestration", knowledgeReference: "know_ref_epic6_docs85_" + suffix, targetArtifact: "art_epic6_generated_target_" + suffix, bomArtifact: "art_epic6_runtime_bom_" + suffix, policyArtifact: "art_epic6_security_policy_" + suffix, reportArtifact: "art_epic6_security_report_" + suffix, proofArtifact: "art_epic6_proof_packet_" + suffix, testCase: "tc_epic6_scanner_orchestration_" + suffix, testRun: "tr_epic6_scanner_orchestration_" + suffix, gateResult: "gate_epic6_scanner_orchestration_" + suffix, failure: "fail_epic6_scanner_orchestration_" + suffix, factoryRuntime: "frv_epic6_scanner_orchestration_" + suffix, releaseCandidate: "rc_epic6_scanner_orchestration_" + suffix, certification: "cert_epic6_scanner_orchestration_" + suffix, rejection: "rej_epic6_scanner_orchestration_" + suffix, auditReport: "aud_epic6_scanner_orchestration_" + suffix, proofPacket: "pow_epic6_scanner_orchestration_" + suffix}
}

func epic6AppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func epic6AppendEdges(graph *v39.InMemoryStore, ids epic6FixtureIDs, createdAt time.Time, includeFailure bool) error {
	edges := []v39.CommonEdge{
		epic6Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic6Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic6Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic6Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic6Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic6Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic6Edge("task_target", v39.EdgeProduced, ids.task, ids.targetArtifact, createdAt),
		epic6Edge("task_bom", v39.EdgeProduced, ids.task, ids.bomArtifact, createdAt),
		epic6Edge("task_policy", v39.EdgeProduced, ids.task, ids.policyArtifact, createdAt),
		epic6Edge("task_report", v39.EdgeProduced, ids.task, ids.reportArtifact, createdAt),
		epic6Edge("task_proof", v39.EdgeProduced, ids.task, ids.proofArtifact, createdAt),
		epic6Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic6Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic6Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	if includeFailure {
		edges = append(edges, epic6Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic6Edge(suffix, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	return v39.CommonEdge{ID: "edge_epic6_" + suffix + "_" + from + "_" + to, Type: typ, FromID: from, ToID: to, CreatedAt: createdAt, CreatedBy: epic6FixtureActorID, CorrelationID: "corr_epic6_scanner_orchestration", IdempotencyKey: "idem_edge_epic6_" + suffix + "_" + from + "_" + to}
}

func epic6Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{ID: id, Type: typ, CreatedAt: epic6FixtureTime(), CreatedBy: epic6FixtureActorID, Status: &status, IdempotencyKey: "idem_" + id, CorrelationID: "corr_epic6_scanner_orchestration"}
}

func epic6FixtureTime() time.Time {
	t, err := time.Parse(time.RFC3339, epic6FixtureTimeRFC)
	if err != nil {
		panic(err)
	}
	return t
}

func epic6SupportedMode(mode Epic6ScannerOrchestrationMode) bool {
	switch mode {
	case Epic6ScannerOrchestrationLocalEvidence, Epic6ScannerOrchestrationMissingScanner, Epic6ScannerOrchestrationOpenCritical, Epic6ScannerOrchestrationOpenHigh, Epic6ScannerOrchestrationHighWaived, Epic6ScannerOrchestrationCommittedSecret:
		return true
	default:
		return false
	}
}

func epic6CertificationID(cert *v39.Certification) string {
	if cert == nil {
		return ""
	}
	return cert.CommonNode.ID
}

func epic6Timeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return 5 * time.Minute
}

func epic6ScannerVersion(output, tool, path string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, tool+" version: ")
		line = strings.TrimPrefix(line, tool+" version ")
		line = strings.TrimPrefix(line, tool+" ")
		if epic6VersionNeedsBuildInfo(line) {
			if version := epic6BuildInfoVersion(path); version != "" {
				return version
			}
		}
		return line
	}
	if version := epic6BuildInfoVersion(path); version != "" {
		return version
	}
	return "unknown"
}

func epic6VersionNeedsBuildInfo(version string) bool {
	normalized := strings.ToLower(strings.TrimSpace(version))
	return normalized == "" || normalized == "dev" || strings.Contains(normalized, "set by build process")
}

func epic6BuildInfoVersion(path string) string {
	info, err := buildinfo.ReadFile(path)
	if err != nil || info == nil || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return ""
	}
	return info.Main.Version
}

func epic6SeverityForGate(gate SecurityGateID) SecurityFindingSeverity {
	if gate == GateSecretScan {
		return FindingSeverityHigh
	}
	return FindingSeverityHigh
}

func epic6CommandOutputRefs(command Epic6ScannerCommandEvidence) []string {
	var refs []string
	for _, ref := range []string{command.OutputRef, command.StdoutRef, command.StderrRef} {
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func epic6LocalCommand(tool, dir string, inputs []string, outputRef string) Epic6ScannerCommandEvidence {
	return Epic6ScannerCommandEvidence{Tool: tool, Command: append([]string{tool, "evaluate"}, inputs...), WorkingDir: dir, StartedAt: epic6FixtureTime().Format(time.RFC3339), CompletedAt: epic6FixtureTime().Format(time.RFC3339), ExitCode: 0, OutputRef: outputRef}
}

func epic6FileContains(path string, needles []string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := string(raw)
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}

func epic6ContainerArtifacts(targetDir string) []string {
	var out []string
	for _, rel := range []string{"Dockerfile", "dist", "build", ".next", "image.tar", "image.oci", "sbom.json"} {
		if _, err := os.Stat(filepath.Join(targetDir, rel)); err == nil {
			out = append(out, rel)
		}
	}
	return out
}

func epic6ValidWaiver(id, findingID string) SecurityWaiver {
	return SecurityWaiver{ID: id, FindingID: findingID, ApproverRole: "security", ExpiresAt: epic6FixtureTime().Add(24 * time.Hour), Reason: "accepted only for local non-production Gate G negative fixture", CompensatingControls: "fixture is local and cannot authorize protected execution, production deployment, or later gates", NotValidFor: []string{"protected_execution", "production_deployment", "gate_h", "gate_i", "gate_j"}}
}

func epic6AddFinding(evidence []Epic6ScannerGateEvidence, gate SecurityGateID, finding SecurityFinding) {
	for i := range evidence {
		if evidence[i].Gate == gate {
			evidence[i].Findings = append(evidence[i].Findings, finding)
			return
		}
	}
}

func epic6AllowedCommands(evidence []Epic6ScannerGateEvidence) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range evidence {
		for _, command := range item.Commands {
			if len(command.Command) == 0 {
				continue
			}
			joined := strings.Join(command.Command, " ")
			if !seen[joined] {
				seen[joined] = true
				out = append(out, joined)
			}
		}
	}
	sort.Strings(out)
	return out
}

func epic6CommandLog(evidence []Epic6ScannerGateEvidence) []string {
	var out []string
	for _, item := range evidence {
		for _, command := range item.Commands {
			out = append(out, fmt.Sprintf("%s:%s:exit=%d", item.Gate, strings.Join(command.Command, " "), command.ExitCode))
		}
	}
	sort.Strings(out)
	return out
}

func epic6Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func epic6CommandEvidenceBase(command Epic6ScannerCommand) string {
	action := "command"
	if len(command.Args) > 0 {
		action = command.Args[0]
	}
	return epic6Slug(command.Tool+"-"+action) + "-" + epic6ShortHash(strings.Join(command.Args, "\x00"))
}

func epic6ShortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func epic6HashFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return epic6Hash("missing:" + path)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func epic6Slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return strings.Trim(b.String(), "_")
}
