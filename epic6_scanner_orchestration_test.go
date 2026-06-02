package work_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic6RealScannerOrchestrationLocalEvidence(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationLocalEvidence)
	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.GateGValidation.Status != "pass" || len(run.GateGValidation.Missing) != 0 {
		t.Fatalf("Gate G validation = %#v; want pass", run.GateGValidation)
	}
	if run.CertificationResult.Blocked {
		t.Fatalf("certification result blocked: %#v", run.CertificationResult)
	}
	if run.WorkProjection.Status != work.StatusCertified {
		t.Fatalf("work status = %q; want certified", run.WorkProjection.Status)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessPassed || !run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want completed pass", run.TraceCompleteness)
	}
	assertEpic6EvidenceShape(t, run)
	assertEpic6NoExecutionReceipt(t, run)
}

func TestEpic6RealScannerOrchestrationMissingScannerBlocksGateG(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationMissingScanner)
	assertEpic6Rejected(t, run)
	if !containsString(run.GateGValidation.Missing, "sast scanner evidence is missing") {
		t.Fatalf("missing = %#v; want SAST scanner evidence blocker", run.GateGValidation.Missing)
	}
}

func TestEpic6RealScannerOrchestrationOpenCriticalBlocksEvenWithWaiver(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationOpenCritical)
	assertEpic6Rejected(t, run)
	if len(run.CertificationResult.BlockingFindings) == 0 || run.CertificationResult.BlockingFindings[0].Severity != work.FindingSeverityCritical {
		t.Fatalf("blocking findings = %#v; want critical finding", run.CertificationResult.BlockingFindings)
	}
}

func TestEpic6RealScannerOrchestrationOpenHighRequiresWaiver(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationOpenHigh)
	assertEpic6Rejected(t, run)
	if len(run.CertificationResult.BlockingFindings) == 0 || run.CertificationResult.BlockingFindings[0].Severity != work.FindingSeverityHigh {
		t.Fatalf("blocking findings = %#v; want high finding", run.CertificationResult.BlockingFindings)
	}
}

func TestEpic6RealScannerOrchestrationValidHighWaiverAllowsLocalCertification(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationHighWaived)
	if run.Certification == nil || run.CertificationResult.Blocked || run.GateGValidation.Status != "pass" {
		t.Fatalf("waived high result certification=%#v result=%#v validation=%#v; want pass", run.Certification, run.CertificationResult, run.GateGValidation)
	}
	if len(run.Waivers) != 1 || !containsString(run.Waivers[0].NotValidFor, "production_deployment") || !containsString(run.Waivers[0].NotValidFor, "gate_h") {
		t.Fatalf("waivers = %#v; want bounded non-production waiver", run.Waivers)
	}
}

func TestEpic6RealScannerOrchestrationCommittedSecretBlocksEvenWhenWaived(t *testing.T) {
	run := runEpic6(t, work.Epic6ScannerOrchestrationCommittedSecret)
	assertEpic6Rejected(t, run)
	foundSecret := false
	for _, finding := range run.CertificationResult.BlockingFindings {
		if finding.SecretHit {
			foundSecret = true
		}
	}
	if !foundSecret {
		t.Fatalf("blocking findings = %#v; want committed secret blocker", run.CertificationResult.BlockingFindings)
	}
}

func TestEpic6RealScannerOrchestrationRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	_, err := work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), CommandRunner: epic6FakeRunner})
	if err == nil || !strings.Contains(err.Error(), "source actor is required") {
		t.Fatalf("missing source err = %v; want source requirement", err)
	}
	_, err = work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{Source: testActor, Causes: causes, WorkingDir: t.TempDir(), CommandRunner: epic6FakeRunner})
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("missing conversation err = %v; want conversation requirement", err)
	}
	_, err = work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{Source: testActor, ConversationID: testConv, Causes: causes, CommandRunner: epic6FakeRunner})
	if err == nil || !strings.Contains(err.Error(), "working directory is required") {
		t.Fatalf("missing working dir err = %v; want local working dir requirement", err)
	}
	_, err = work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{Source: testActor, ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), Mode: work.Epic6ScannerOrchestrationMode("gate_h"), CommandRunner: epic6FakeRunner})
	if err == nil || !strings.Contains(err.Error(), "unsupported Epic 6 fixture mode") {
		t.Fatalf("unsupported mode err = %v; want mode rejection", err)
	}
}

func TestEpic6RealScannerOrchestrationRealTools(t *testing.T) {
	if os.Getenv("EPIC6_REAL_SCANNER_TOOLS") != "1" {
		t.Skip("set EPIC6_REAL_SCANNER_TOOLS=1 and EPIC6_* scanner paths to run real Gate G scanner validation")
	}
	paths := work.Epic6ScannerToolPaths{
		Gitleaks:   os.Getenv("EPIC6_GITLEAKS"),
		OSVScanner: os.Getenv("EPIC6_OSV_SCANNER"),
		Semgrep:    os.Getenv("EPIC6_SEMGREP"),
		Trivy:      os.Getenv("EPIC6_TRIVY"),
	}
	if paths.Gitleaks == "" || paths.OSVScanner == "" || paths.Semgrep == "" {
		t.Fatalf("EPIC6_GITLEAKS, EPIC6_OSV_SCANNER, and EPIC6_SEMGREP are required for real scanner validation")
	}
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	workingDir := os.Getenv("EPIC6_REAL_SCANNER_WORKDIR")
	if workingDir == "" {
		workingDir = t.TempDir()
	} else if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("create EPIC6_REAL_SCANNER_WORKDIR: %v", err)
	}
	run, err := work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     workingDir,
		Mode:           work.Epic6ScannerOrchestrationLocalEvidence,
		ToolPaths:      paths,
		Timeout:        2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("RunEpic6RealScannerOrchestrationTrial real tools: %v", err)
	}
	if run.GateGValidation.Status != "pass" || run.Certification == nil || run.CertificationResult.Blocked {
		t.Fatalf("real scanner validation status=%#v certification=%#v result=%#v report=%s proof=%s; want pass", run.GateGValidation, run.Certification, run.CertificationResult, run.ReportPath, run.ProofPath)
	}
	for _, path := range []string{run.ReportPath, run.ProofPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("real scanner evidence artifact %s: %v", path, err)
		}
	}
	assertEpic6EvidenceShape(t, run)
	assertEpic6NoExecutionReceipt(t, run)
}

func runEpic6(t *testing.T, mode work.Epic6ScannerOrchestrationMode) work.Epic6ScannerOrchestrationRun {
	t.Helper()
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	run, err := work.RunEpic6RealScannerOrchestrationTrial(ts, work.Epic6ScannerOrchestrationOptions{Source: testActor, ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), Mode: mode, CommandRunner: epic6FakeRunner, Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunEpic6RealScannerOrchestrationTrial(%s): %v", mode, err)
	}
	return run
}

func epic6FakeRunner(_ context.Context, command work.Epic6ScannerCommand) work.Epic6CommandResult {
	started := time.Date(2026, 6, 2, 0, 30, 0, 0, time.UTC)
	stdout := "{}"
	if len(command.Args) > 0 && (strings.Contains(strings.Join(command.Args, " "), "version") || command.Args[0] == "--version") {
		stdout = map[string]string{"gitleaks": "8.18.4", "osv-scanner": "1.9.1", "semgrep": "1.96.0"}[command.Tool]
		if stdout == "" {
			stdout = "1.0.0"
		}
	}
	if command.OutputRef != "" {
		_ = os.WriteFile(command.OutputRef, []byte(`{"results":[]}`+"\n"), 0o644)
	}
	return work.Epic6CommandResult{StartedAt: started, CompletedAt: started.Add(time.Second), ExitCode: 0, Stdout: stdout}
}

func assertEpic6Rejected(t *testing.T, run work.Epic6ScannerOrchestrationRun) {
	t.Helper()
	if run.Certification != nil || run.Rejection == nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want rejection only", run.Certification, run.Rejection)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if run.GateGValidation.Status != "fail" {
		t.Fatalf("Gate G status = %q; want fail", run.GateGValidation.Status)
	}
	assertEpic6NoExecutionReceipt(t, run)
}

func assertEpic6EvidenceShape(t *testing.T, run work.Epic6ScannerOrchestrationRun) {
	t.Helper()
	if len(run.ScannerEvidence) != len(work.RequiredSaaSTemplateV1SecurityGates()) {
		t.Fatalf("scanner evidence count = %d; want required gate count", len(run.ScannerEvidence))
	}
	byGate := map[work.SecurityGateID]work.Epic6ScannerGateEvidence{}
	for _, item := range run.ScannerEvidence {
		byGate[item.Gate] = item
		if item.EvidenceMode == "scaffold" {
			t.Fatalf("%s still uses scaffold evidence", item.Gate)
		}
		if item.D2ScaffoldDisposition == "" {
			t.Fatalf("%s missing D2 disposition", item.Gate)
		}
	}
	for _, gate := range []work.SecurityGateID{work.GateSecretScan, work.GateDependencyVulnerabilityScan, work.GateSAST} {
		item := byGate[gate]
		if item.ScannerVersion == "" || len(item.Commands) < 2 {
			t.Fatalf("%s evidence = %#v; want version and command evidence", gate, item)
		}
	}
	for _, gate := range []work.SecurityGateID{work.GateDependencyLicenseScan, work.GateAuthFlowSecurityCheck, work.GateConfigurationSecurityCheck} {
		item := byGate[gate]
		if item.EvidenceMode != "real_local_checker" || len(item.Commands) == 0 {
			t.Fatalf("%s evidence = %#v; want real local checker command evidence", gate, item)
		}
	}
	if byGate[work.GateContainerOrArtifactScan].Status != work.SecurityGateStatusNotApplicable || byGate[work.GateContainerOrArtifactScan].NotApplicableReason == "" {
		t.Fatalf("container evidence = %#v; want explicit not_applicable proof", byGate[work.GateContainerOrArtifactScan])
	}
	payload, err := run.Projection.JSON()
	if err != nil {
		t.Fatalf("projection JSON: %v", err)
	}
	var decoded struct {
		ProofOfWorkPacket struct {
			Status                string `json:"status"`
			D2ScaffoldDisposition struct {
				Summary string `json:"summary"`
			} `json:"d2_scaffold_disposition"`
			ResidualRisks []struct {
				Status string `json:"status"`
			} `json:"residual_risks"`
		} `json:"proof_of_work_packet"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	if decoded.ProofOfWorkPacket.Status != "pass" || !strings.Contains(decoded.ProofOfWorkPacket.D2ScaffoldDisposition.Summary, "real external scanner/checker") {
		t.Fatalf("proof packet = %#v; want visible scanner/scaffold disposition", decoded.ProofOfWorkPacket)
	}
	if len(decoded.ProofOfWorkPacket.ResidualRisks) != 3 {
		t.Fatalf("residual risks = %#v; want R-001/R-002/R-003 exclusions", decoded.ProofOfWorkPacket.ResidualRisks)
	}
}

func assertEpic6NoExecutionReceipt(t *testing.T, run work.Epic6ScannerOrchestrationRun) {
	t.Helper()
	if records := run.EventGraph.ByType(v39.TypeExecutionReceipt); len(records) != 0 {
		t.Fatalf("ExecutionReceipt records = %#v; want none", records)
	}
}
