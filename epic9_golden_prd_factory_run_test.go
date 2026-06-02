package work_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic9GoldenPRDProductFactoryRunLocalDryRunCertifiesFromEvidence(t *testing.T) {
	run := runEpic9(t, work.Epic9GoldenPRDOptions{})

	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.GateJValidation.Status != "pass" || len(run.GateJValidation.Missing) != 0 {
		t.Fatalf("Gate J validation = %#v; want pass", run.GateJValidation)
	}
	if run.WorkProjection.Status != work.StatusCertified {
		t.Fatalf("work status = %q; want certified", run.WorkProjection.Status)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessPassed || !run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want completed pass", run.TraceCompleteness)
	}
	if !run.CapabilityUsagePath.Completed || !run.KnowledgePath.Completed {
		t.Fatalf("influence paths capability=%#v knowledge=%#v; want complete", run.CapabilityUsagePath, run.KnowledgePath)
	}
	if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "complete" {
		t.Fatalf("audit report = %#v; want complete", run.AuditReport)
	}

	assertEpic9GoldenPRD(t, run)
	assertEpic9GeneratedTemplate(t, run)
	assertEpic9SecurityGates(t, run)
	assertEpic9AuthorityAndRelease(t, run)
	assertEpic9ProofPacket(t, run)
}

func TestEpic9GoldenPRDProductFactoryRunAllowsHighFindingWithValidLocalWaiver(t *testing.T) {
	run := runEpic9(t, work.Epic9GoldenPRDOptions{AddOpenHighSecurityFinding: true, AddValidHighWaiver: true})
	if run.Certification == nil || run.Rejection != nil || run.GateJValidation.Status != "pass" {
		t.Fatalf("run certification=%#v rejection=%#v validation=%#v; want local certification with valid high waiver", run.Certification, run.Rejection, run.GateJValidation)
	}
	if len(run.SecurityGateReport.Waivers) != 1 || !containsString(run.SecurityGateReport.Waivers[0].NotValidFor, "production_deployment") {
		t.Fatalf("waivers = %#v; want production-deployment-invalid local waiver", run.SecurityGateReport.Waivers)
	}
}

func TestEpic9GoldenPRDProductFactoryRunRejectsRequiredMissingEvidence(t *testing.T) {
	tests := []struct {
		name        string
		opts        work.Epic9GoldenPRDOptions
		wantMissing string
	}{
		{name: "missing FactoryOrder", opts: work.Epic9GoldenPRDOptions{OmitFactoryOrder: true}, wantMissing: "FactoryOrder missing"},
		{name: "missing source intent", opts: work.Epic9GoldenPRDOptions{OmitSourceIntent: true}, wantMissing: "selected PRD/source intent evidence missing"},
		{name: "missing acceptance evidence", opts: work.Epic9GoldenPRDOptions{OmitAcceptanceEvidence: true}, wantMissing: "acceptance evidence missing"},
		{name: "missing generated artifact", opts: work.Epic9GoldenPRDOptions{OmitGeneratedArtifactEvidence: true}, wantMissing: "generated artifact evidence missing"},
		{name: "missing security gate evidence", opts: work.Epic9GoldenPRDOptions{OmitSecurityGateEvidence: true}, wantMissing: "security-gate evidence missing"},
		{name: "open critical finding", opts: work.Epic9GoldenPRDOptions{AddOpenCriticalSecurityFinding: true}, wantMissing: "open critical security finding: finding_epic9_open_critical_sast"},
		{name: "open high finding without waiver", opts: work.Epic9GoldenPRDOptions{AddOpenHighSecurityFinding: true}, wantMissing: "open high security finding without valid waiver: finding_epic9_open_high_config"},
		{name: "missing runtime version", opts: work.Epic9GoldenPRDOptions{OmitFactoryRuntimeVersion: true}, wantMissing: "FactoryRuntimeVersion missing"},
		{name: "missing release authority", opts: work.Epic9GoldenPRDOptions{OmitReleaseAuthority: true}, wantMissing: "release decision authority missing"},
		{name: "missing audit report", opts: work.Epic9GoldenPRDOptions{OmitAuditReport: true}, wantMissing: "AuditReport missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := runEpic9(t, tt.opts)
			assertEpic9Rejected(t, run)
			if !containsString(run.GateJValidation.Missing, tt.wantMissing) {
				t.Fatalf("missing = %#v; want %q", run.GateJValidation.Missing, tt.wantMissing)
			}
			if tt.opts.OmitAuditReport && run.AuditReport != nil {
				t.Fatalf("audit report = %#v; want omitted", run.AuditReport)
			}
			if tt.opts.OmitFactoryOrder && run.Rejection != nil {
				t.Fatalf("rejection = %#v; want no release decision when FactoryOrder is absent", run.Rejection)
			}
			if tt.opts.OmitSourceIntent {
				order := assertEpic9GraphRecord(t, run, run.FactoryOrderID, v39.TypeFactoryOrder).(*v39.FactoryOrder)
				if strings.HasPrefix(order.SourceIntentHash, "sha256:") {
					t.Fatalf("source-intent sentinel = %q; want no digest prefix", order.SourceIntentHash)
				}
			}
		})
	}
}

func TestEpic9GoldenPRDProductFactoryRunRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	_, err := work.RunEpic9GoldenPRDProductFactoryRun(ts, work.Epic9GoldenPRDOptions{ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil {
		t.Fatalf("missing source actor err = nil; want error")
	}
	_, err = work.RunEpic9GoldenPRDProductFactoryRun(ts, work.Epic9GoldenPRDOptions{Source: testActor, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil {
		t.Fatalf("missing conversation err = nil; want error")
	}
	_, err = work.RunEpic9GoldenPRDProductFactoryRun(ts, work.Epic9GoldenPRDOptions{Source: testActor, ConversationID: testConv, Causes: causes})
	if err == nil {
		t.Fatalf("missing working dir err = nil; want error")
	}
	_, err = work.RunEpic9GoldenPRDProductFactoryRun(ts, work.Epic9GoldenPRDOptions{Source: testActor, ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), Mode: work.Epic9GoldenPRDMode("production")})
	if err == nil {
		t.Fatalf("unsupported mode err = nil; want error")
	}
}

func runEpic9(t *testing.T, opts work.Epic9GoldenPRDOptions) work.Epic9GoldenPRDRun {
	t.Helper()
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	opts.Source = testActor
	opts.ConversationID = testConv
	opts.Causes = causes
	opts.WorkingDir = t.TempDir()
	run, err := work.RunEpic9GoldenPRDProductFactoryRun(ts, opts)
	if err != nil {
		t.Fatalf("RunEpic9GoldenPRDProductFactoryRun: %v", err)
	}
	return run
}

func assertEpic9Rejected(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	if run.Certification != nil {
		t.Fatalf("certification=%#v; want no certification", run.Certification)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if run.GateJValidation.Status != "fail" || len(run.GateJValidation.Missing) == 0 {
		t.Fatalf("Gate J validation = %#v; want fail with missing evidence", run.GateJValidation)
	}
}

func assertEpic9GoldenPRD(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	golden := run.Projection.GoldenPRD
	if golden.Name != "simple CRUD tracker" || golden.SourceRef == "" || golden.LocatorHash == "" {
		t.Fatalf("golden PRD = %#v; want named source ref and locator hash", golden)
	}
	if run.GateJValidation.Metrics.GoldenPRDRef != golden.SourceRef || run.GateJValidation.Metrics.GoldenPRDLocatorHash != golden.LocatorHash {
		t.Fatalf("metrics = %#v; want golden PRD ref/locator hash", run.GateJValidation.Metrics)
	}
	knowledge := assertEpic9GraphRecord(t, run, run.KnowledgeReferenceID, v39.TypeKnowledgeReference).(*v39.KnowledgeReference)
	if strings.HasPrefix(knowledge.SourceHashOrImmutableLocator, "sha256:") {
		t.Fatalf("knowledge locator = %q; want locator without digest prefix", knowledge.SourceHashOrImmutableLocator)
	}
	if !strings.Contains(knowledge.SourceHashOrImmutableLocator, "docs-pr-91-merged-") {
		t.Fatalf("knowledge locator = %q; want docs#91 merge locator", knowledge.SourceHashOrImmutableLocator)
	}
}

func assertEpic9GeneratedTemplate(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	assertFileExists(t, run.LocalArtifacts.GeneratedManifest)
	assertFileExists(t, run.LocalArtifacts.DeployPreviewDryRun)
	assertFileExists(t, run.LocalArtifacts.ProofOfWork)
	assertFileExists(t, run.LocalArtifacts.AuditReport)
	assertFileExists(t, run.LocalArtifacts.GeneratedTemplateDir+"/frontend/app/dashboard/page.tsx")
	assertFileExists(t, run.LocalArtifacts.GeneratedTemplateDir+"/backend/app/main.py")
	if run.GeneratedManifest.TemplateID != work.SaaSTemplateV1ID {
		t.Fatalf("template id = %q; want %q", run.GeneratedManifest.TemplateID, work.SaaSTemplateV1ID)
	}
	if run.GeneratedManifest.FileCount != len(work.SaaSTemplateV1Files()) {
		t.Fatalf("generated file count = %d; want %d", run.GeneratedManifest.FileCount, len(work.SaaSTemplateV1Files()))
	}
	if !containsString(run.GeneratedManifest.Files, "frontend/tests/e2e/auth-and-tracker.spec.ts") {
		t.Fatalf("generated files missing auth-and-tracker e2e test: %#v", run.GeneratedManifest.Files)
	}
}

func assertEpic9SecurityGates(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	assertFileExists(t, run.LocalArtifacts.SecurityGateReport)
	if run.SecurityGateReport.Status != "pass" {
		t.Fatalf("security report status = %q; want pass", run.SecurityGateReport.Status)
	}
	if len(run.SecurityGateReport.Gates) != len(work.RequiredSaaSTemplateV1SecurityGates()) {
		t.Fatalf("security gates = %d; want %d", len(run.SecurityGateReport.Gates), len(work.RequiredSaaSTemplateV1SecurityGates()))
	}
	if run.SecurityGateReport.CertificationResult.Blocked {
		t.Fatalf("security result = %#v; want unblocked", run.SecurityGateReport.CertificationResult)
	}
	if run.GateJValidation.Metrics.SecurityGateCount != len(work.RequiredSaaSTemplateV1SecurityGates()) || run.GateJValidation.Metrics.SecurityBlockingCount != 0 {
		t.Fatalf("security metrics = %#v; want all gates and no blockers", run.GateJValidation.Metrics)
	}
}

func assertEpic9AuthorityAndRelease(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	assertEpic9GraphRecord(t, run, run.FactoryOrderID, v39.TypeFactoryOrder)
	assertEpic9GraphRecord(t, run, run.RequirementID, v39.TypeRequirement)
	assertEpic9GraphRecord(t, run, run.AcceptanceCriterionID, v39.TypeAcceptanceCriterion)
	assertEpic9GraphRecord(t, run, run.TaskID, v39.TypeTask)
	assertEpic9GraphRecord(t, run, run.FactoryRuntimeVersionID, v39.TypeFactoryRuntimeVersion)
	assertEpic9GraphRecord(t, run, run.AuthorityRequestID, v39.TypeAuthorityRequest)
	assertEpic9GraphRecord(t, run, run.AuthorityDecisionID, v39.TypeAuthorityDecision)
	assertEpic9GraphRecord(t, run, run.HumanApprovalID, v39.TypeHumanApproval)
	assertEpic9GraphRecord(t, run, run.ReleaseCandidateID, v39.TypeReleaseCandidate)
	assertEpic9GraphRecord(t, run, run.CertificationID, v39.TypeCertification)
	assertEpic9GraphRecord(t, run, run.AuditReportID, v39.TypeAuditReport)
	if run.Projection.ProofOfWorkPacket.AuthorityRecords.Decision != "approved" {
		t.Fatalf("authority = %#v; want approved", run.Projection.ProofOfWorkPacket.AuthorityRecords)
	}
	if run.Projection.ProofOfWorkPacket.ReleaseEvidence.Decision != "certified" {
		t.Fatalf("release evidence = %#v; want certified", run.Projection.ProofOfWorkPacket.ReleaseEvidence)
	}
}

func assertEpic9ProofPacket(t *testing.T, run work.Epic9GoldenPRDRun) {
	t.Helper()
	proof := readEpic9Proof(t, run)
	if proof.Status != "pass" || proof.ReleaseEvidence.Decision != "certified" {
		t.Fatalf("proof = %#v; want pass/certified", proof)
	}
	if len(proof.TraceGates) == 0 || len(proof.EventGraphRefs) == 0 {
		t.Fatalf("proof trace refs=%#v event refs=%#v; want inspectable refs", proof.TraceGates, proof.EventGraphRefs)
	}
	for _, label := range []string{"R-001", "R-002", "R-003"} {
		found := false
		for _, risk := range proof.ResidualRisks {
			if risk.Label == label && risk.Status == "excluded" {
				found = true
			}
		}
		if !found {
			t.Fatalf("residual risks = %#v; want %s excluded", proof.ResidualRisks, label)
		}
	}
	for _, action := range proof.ForbiddenActions {
		if action.Status != "not_run" {
			t.Fatalf("forbidden action = %#v; want not_run", action)
		}
	}
	runtimeResult := assertEpic9GraphRecord(t, run, run.RuntimeResultID, v39.TypeRuntimeResult).(*v39.RuntimeResult)
	for _, command := range runtimeResult.CommandLog {
		if strings.Contains(command, "gh pr create:run") || strings.Contains(command, "git push:run") || strings.Contains(command, "deploy:run") {
			t.Fatalf("command log contains live mutation command: %#v", runtimeResult.CommandLog)
		}
	}
}

func readEpic9Proof(t *testing.T, run work.Epic9GoldenPRDRun) work.Epic9ProofOfWorkPacket {
	t.Helper()
	raw, err := os.ReadFile(run.LocalArtifacts.ProofOfWork)
	if err != nil {
		t.Fatalf("read proof: %v", err)
	}
	var proof work.Epic9ProofOfWorkPacket
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatalf("decode proof: %v", err)
	}
	return proof
}

func assertEpic9GraphRecord(t *testing.T, run work.Epic9GoldenPRDRun, id, typ string) v39.Record {
	t.Helper()
	record, err := run.EventGraph.Get(id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	if record.GetCommon().Type != typ {
		t.Fatalf("record %s type = %q; want %q", id, record.GetCommon().Type, typ)
	}
	return record
}
