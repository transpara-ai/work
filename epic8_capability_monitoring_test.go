package work_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic8CapabilityMonitoringRuntimeSelectionTrialsLocalEvidence(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{})

	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.GateIValidation.Status != "pass" || len(run.GateIValidation.Missing) != 0 {
		t.Fatalf("Gate I validation = %#v; want pass", run.GateIValidation)
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

	assertEpic8MonitoringWindow(t, run)
	assertEpic8RuntimeSelectionEvidence(t, run)
	assertEpic8PromotionEvidence(t, run)
	assertEpic8RollbackAuthority(t, run)
	assertEpic8LocalArtifacts(t, run)
	assertEpic8ProofJSONShape(t, run)
}

func TestEpic8CapabilityMonitoringRejectsMissingCapabilityVersionEvidence(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{OmitCapabilityVersionEvidence: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "candidate CapabilityVersion promotion evidence missing") {
		t.Fatalf("missing = %#v; want capability version evidence failure", run.GateIValidation.Missing)
	}
	if run.PromotionReceiptLocalEvidence {
		t.Fatalf("promotion receipt local evidence = true; want omitted when capability evidence is missing")
	}
}

func TestEpic8CapabilityMonitoringRejectsMissingMonitoringWindow(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{OmitMonitoringWindow: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "monitoring window artifact missing") {
		t.Fatalf("missing = %#v; want monitoring window failure", run.GateIValidation.Missing)
	}
	if _, err := os.Stat(run.LocalArtifacts.MonitoringWindow); err == nil {
		t.Fatalf("monitoring window artifact exists at %s; want omitted negative seam", run.LocalArtifacts.MonitoringWindow)
	}
}

func TestEpic8CapabilityMonitoringRejectsMissingRollbackTrigger(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{OmitRollbackTrigger: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "rollback trigger evidence missing") {
		t.Fatalf("missing = %#v; want rollback trigger failure", run.GateIValidation.Missing)
	}
	if _, err := run.EventGraph.Get(run.RollbackRecordID); err == nil {
		t.Fatalf("rollback record %s exists; want omitted rollback trigger seam", run.RollbackRecordID)
	}
	if _, err := os.Stat(run.LocalArtifacts.RollbackDecision); err == nil {
		t.Fatalf("rollback decision artifact exists at %s; want omitted rollback trigger seam", run.LocalArtifacts.RollbackDecision)
	}
	proof := readEpic8Proof(t, run)
	if proof.Metrics.RollbackTriggerCount != 0 {
		t.Fatalf("proof rollback trigger count = %d; want 0", proof.Metrics.RollbackTriggerCount)
	}
	if proof.RollbackDecision.Decision != "missing" || proof.RollbackDecision.AuthorityDecisionID != "" {
		t.Fatalf("proof rollback decision = %#v; want missing without authority refs", proof.RollbackDecision)
	}
	if !strings.Contains(proof.RollbackDecision.Summary, "Rollback trigger evidence missing") {
		t.Fatalf("proof rollback decision summary = %q; want trigger-missing explanation", proof.RollbackDecision.Summary)
	}
}

func TestEpic8CapabilityMonitoringRejectsMissingOperatorRollbackAuthority(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{OmitOperatorRollbackAuthority: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "operator rollback authority missing") {
		t.Fatalf("missing = %#v; want operator authority failure", run.GateIValidation.Missing)
	}
	if run.MonitoringWindow.Metrics.OperatorRollbackDecisionRef != "" {
		t.Fatalf("operator decision ref = %q; want empty", run.MonitoringWindow.Metrics.OperatorRollbackDecisionRef)
	}
}

func TestEpic8CapabilityMonitoringRejectsGlobalActivationScope(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{UseGlobalActivationScope: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "ActivationPolicy scope=global is forbidden") {
		t.Fatalf("missing = %#v; want global activation failure", run.GateIValidation.Missing)
	}
	if !run.GlobalActivationRejected || !strings.Contains(run.GlobalActivationError, "global activation disabled") {
		t.Fatalf("global activation rejected=%v error=%q; want EventGraph global rejection", run.GlobalActivationRejected, run.GlobalActivationError)
	}
}

func TestEpic8CapabilityMonitoringRejectsMissingCandidateReselectionProbe(t *testing.T) {
	run := runEpic8(t, work.Epic8CapabilityMonitoringOptions{SkipCandidateReselectionProbe: true})
	assertEpic8Rejected(t, run)
	if !containsString(run.GateIValidation.Missing, "post-rollback candidate reselection probe missing") {
		t.Fatalf("missing = %#v; want reselection probe failure", run.GateIValidation.Missing)
	}
}

func TestEpic8CapabilityMonitoringRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	_, err := work.RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts, work.Epic8CapabilityMonitoringOptions{ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil {
		t.Fatalf("missing source actor err = nil; want error")
	}
	_, err = work.RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts, work.Epic8CapabilityMonitoringOptions{Source: testActor, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil {
		t.Fatalf("missing conversation err = nil; want error")
	}
	_, err = work.RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts, work.Epic8CapabilityMonitoringOptions{Source: testActor, ConversationID: testConv, Causes: causes})
	if err == nil {
		t.Fatalf("missing working dir err = nil; want error")
	}
	_, err = work.RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts, work.Epic8CapabilityMonitoringOptions{Source: testActor, ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), Mode: work.Epic8CapabilityMonitoringMode("gate_j")})
	if err == nil {
		t.Fatalf("unsupported mode err = nil; want error")
	}
}

func runEpic8(t *testing.T, opts work.Epic8CapabilityMonitoringOptions) work.Epic8CapabilityMonitoringRun {
	t.Helper()
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	opts.Source = testActor
	opts.ConversationID = testConv
	opts.Causes = causes
	opts.WorkingDir = t.TempDir()
	run, err := work.RunEpic8CapabilityMonitoringRuntimeSelectionTrials(ts, opts)
	if err != nil {
		t.Fatalf("RunEpic8CapabilityMonitoringRuntimeSelectionTrials: %v", err)
	}
	return run
}

func assertEpic8Rejected(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	if run.Certification != nil || run.Rejection == nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want rejection only", run.Certification, run.Rejection)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if run.GateIValidation.Status != "fail" || len(run.GateIValidation.Missing) == 0 {
		t.Fatalf("Gate I validation = %#v; want fail with missing evidence", run.GateIValidation)
	}
}

func assertEpic8MonitoringWindow(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	window := run.MonitoringWindow
	if len(window.Runs) != 5 {
		t.Fatalf("runs = %d; want 5", len(window.Runs))
	}
	metrics := window.Metrics
	if metrics.MonitoringWindowRuns != len(window.Runs) {
		t.Fatalf("monitoring_window_runs = %d; want len(runs) %d", metrics.MonitoringWindowRuns, len(window.Runs))
	}
	var candidateAttempts, candidateSuccesses, candidateRegressions, rollbackTriggers, postRollbackSuccesses int
	for _, row := range window.Runs {
		if row.TrialID != "trial_5_post_rollback_baseline_selection_success" && row.CapabilityVersionRef == run.CandidateCapabilityVersionID {
			candidateAttempts++
			if row.Success {
				candidateSuccesses++
			}
			if row.Regression {
				candidateRegressions++
			}
		}
		if row.RollbackTriggered {
			rollbackTriggers++
		}
		if row.TrialID == "trial_5_post_rollback_baseline_selection_success" && row.Success {
			postRollbackSuccesses++
		}
	}
	if metrics.CandidateAttemptCount != candidateAttempts || metrics.CandidateSuccessCount != candidateSuccesses || metrics.CandidateRegressionCount != candidateRegressions {
		t.Fatalf("metrics = %#v; want attempts=%d successes=%d regressions=%d derived from rows", metrics, candidateAttempts, candidateSuccesses, candidateRegressions)
	}
	if metrics.CandidateSuccessRate != 0.75 || metrics.RollbackTriggerCount != rollbackTriggers || metrics.PostRollbackSuccessCount != postRollbackSuccesses {
		t.Fatalf("metrics = %#v; want success rate .75 rollback=%d post=%d", metrics, rollbackTriggers, postRollbackSuccesses)
	}
	if metrics.ActiveCapabilityVersionRef != run.CandidateCapabilityVersionID || metrics.RollbackToCapabilityVersionRef != run.BaselineCapabilityVersionID {
		t.Fatalf("capability refs = %#v; want active candidate and baseline rollback", metrics)
	}
}

func assertEpic8RuntimeSelectionEvidence(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	preRuntime := assertEpic8FactoryRuntime(t, run, run.PreRollbackRuntimeVersionID)
	if !containsString(preRuntime.CapabilityVersionRefs, run.CandidateCapabilityVersionID) {
		t.Fatalf("pre-rollback runtime refs = %#v; want candidate", preRuntime.CapabilityVersionRefs)
	}
	postRuntime := assertEpic8FactoryRuntime(t, run, run.PostRollbackRuntimeVersionID)
	if containsString(postRuntime.CapabilityVersionRefs, run.CandidateCapabilityVersionID) {
		t.Fatalf("post-rollback runtime refs = %#v; want candidate omitted", postRuntime.CapabilityVersionRefs)
	}
	assertEpic8GraphRecord(t, run, run.ActivationPolicyID, v39.TypeActivationPolicy)
	assertEpic8GraphRecord(t, run, run.RollbackRecordID, v39.TypeRollbackRecord)
	if !run.CandidateReselectionBlocked || !strings.Contains(run.CandidateReselectionError, "rolled back capability version cannot be activated") {
		t.Fatalf("candidate reselection blocked=%v error=%q; want rolled-back candidate blocked", run.CandidateReselectionBlocked, run.CandidateReselectionError)
	}
}

func assertEpic8PromotionEvidence(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	assertEpic8GraphRecord(t, run, run.CapabilityArtifactID, v39.TypeCapabilityArtifact)
	assertEpic8GraphRecord(t, run, run.BaselineCapabilityVersionID, v39.TypeCapabilityVersion)
	assertEpic8GraphRecord(t, run, run.CandidateCapabilityVersionID, v39.TypeCapabilityVersion)
	assertEpic8GraphRecord(t, run, "evo_epic8_issue_pr_proposer_candidate", v39.TypeEvolutionOrder)
	assertEpic8GraphRecord(t, run, "eval_epic8_monitoring_window", v39.TypeEvalDataset)
	assertEpic8GraphRecord(t, run, "opt_epic8_issue_pr_proposer_candidate", v39.TypeOptimizationRun)
	assertEpic8GraphRecord(t, run, "cand_epic8_issue_pr_proposer_candidate", v39.TypeCandidateVariant)
	assertEpic8GraphRecord(t, run, "bench_epic8_issue_pr_proposer_candidate", v39.TypeBenchmarkResult)
	assertEpic8GraphRecord(t, run, "review_epic8_issue_pr_proposer_candidate", v39.TypeHumanReview)
	assertEpic8GraphRecord(t, run, run.PromotionAuthorityRequestID, v39.TypeAuthorityRequest)
	assertEpic8GraphRecord(t, run, run.PromotionAuthorityDecisionID, v39.TypeAuthorityDecision)
	assertEpic8GraphRecord(t, run, run.PromotionExecutionReceiptID, v39.TypeExecutionReceipt)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom("actor_epic8_capability_release"), v39.EdgeRequestedAuthority, run.PromotionAuthorityRequestID)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom(run.PromotionAuthorityRequestID), v39.EdgeDecidedBy, run.PromotionAuthorityDecisionID)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom(run.PromotionAuthorityDecisionID), v39.EdgeReceiptedBy, run.PromotionExecutionReceiptID)
	if !run.PromotionReceiptLocalEvidence {
		t.Fatalf("promotion receipt local evidence = false; want true")
	}
}

func assertEpic8RollbackAuthority(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	assertEpic8GraphRecord(t, run, run.OperatorAuthorityRequestID, v39.TypeAuthorityRequest)
	assertEpic8GraphRecord(t, run, run.OperatorAuthorityDecisionID, v39.TypeAuthorityDecision)
	assertEpic8GraphRecord(t, run, run.OperatorHumanApprovalID, v39.TypeHumanApproval)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom(run.ActorInvocationID), v39.EdgeRequestedAuthority, run.OperatorAuthorityRequestID)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom(run.OperatorAuthorityRequestID), v39.EdgeDecidedBy, run.OperatorAuthorityDecisionID)
	assertEpic8Edge(t, run.EventGraph.EdgesFrom(run.OperatorAuthorityRequestID), v39.EdgeApprovedBy, run.OperatorHumanApprovalID)
	if run.MonitoringWindow.Metrics.OperatorRollbackDecisionRef != run.OperatorAuthorityDecisionID {
		t.Fatalf("operator decision ref = %q; want %q", run.MonitoringWindow.Metrics.OperatorRollbackDecisionRef, run.OperatorAuthorityDecisionID)
	}
}

func assertEpic8LocalArtifacts(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	assertFileExists(t, run.LocalArtifacts.MonitoringWindow)
	assertFileExists(t, run.LocalArtifacts.ProofOfWork)
	assertFileExists(t, run.LocalArtifacts.RollbackDecision)
	assertEpic8WriteConfinement(t, run)
}

func assertEpic8ProofJSONShape(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	proof := readEpic8Proof(t, run)
	if proof.Status != "pass" || len(proof.TrialRefs) != 5 {
		t.Fatalf("proof status=%q trials=%#v; want pass with five trials", proof.Status, proof.TrialRefs)
	}
	if proof.Metrics.CandidateSuccessRate != 0.75 || proof.Metrics.CandidateRegressionCount != 1 || proof.Metrics.RollbackTriggerCount != 1 {
		t.Fatalf("proof metrics = %#v; want counters", proof.Metrics)
	}
	if !proof.RuntimeSelection.CandidateReselectionBlocked || proof.RuntimeSelection.ActiveCapabilityVersionRef != run.CandidateCapabilityVersionID {
		t.Fatalf("runtime selection = %#v; want blocked candidate reselection and active candidate ref", proof.RuntimeSelection)
	}
	if proof.RollbackDecision.AuthorityDecisionID != run.OperatorAuthorityDecisionID {
		t.Fatalf("rollback decision = %#v; want operator decision", proof.RollbackDecision)
	}
	if proof.NoGlobalActivationProof.Status != "pass" {
		t.Fatalf("no-global proof = %#v; want pass", proof.NoGlobalActivationProof)
	}
	if !strings.Contains(proof.LocalPromotionReceiptScope, "side-effect-free local capability.promote evidence") || strings.Contains(proof.LocalPromotionReceiptScope, "production ExecutionReceipt path claimed") {
		t.Fatalf("receipt scope = %q; want local-only non-production wording", proof.LocalPromotionReceiptScope)
	}
	assertEpic8ResidualRisk(t, proof, "R-001", "excluded")
	assertEpic8ResidualRisk(t, proof, "R-002", "excluded")
	assertEpic8ResidualRisk(t, proof, "R-003", "excluded")
	assertEpic8ResidualRisk(t, proof, "Gate J", "waiting")
}

func readEpic8Proof(t *testing.T, run work.Epic8CapabilityMonitoringRun) work.Epic8ProofOfWorkPacket {
	t.Helper()
	raw, err := os.ReadFile(run.LocalArtifacts.ProofOfWork)
	if err != nil {
		t.Fatalf("read proof file: %v", err)
	}
	var proof work.Epic8ProofOfWorkPacket
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatalf("decode proof JSON: %v", err)
	}
	return proof
}

func assertEpic8ResidualRisk(t *testing.T, proof work.Epic8ProofOfWorkPacket, label, status string) {
	t.Helper()
	for _, risk := range proof.ResidualRisks {
		if risk.Label == label {
			if risk.Status != status {
				t.Fatalf("residual risk %s status = %q; want %q", label, risk.Status, status)
			}
			return
		}
	}
	t.Fatalf("residual risks = %#v; want label %s", proof.ResidualRisks, label)
}

func assertEpic8WriteConfinement(t *testing.T, run work.Epic8CapabilityMonitoringRun) {
	t.Helper()
	workingDir := filepath.Dir(filepath.Dir(run.LocalArtifacts.Root))
	expected := map[string]bool{
		run.LocalArtifacts.MonitoringWindow: true,
		run.LocalArtifacts.ProofOfWork:      true,
		run.LocalArtifacts.RollbackDecision: true,
	}
	if err := filepath.WalkDir(workingDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !expected[path] {
			t.Fatalf("unexpected local fixture write %s under %s", path, workingDir)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk working dir %s: %v", workingDir, err)
	}
}

func assertEpic8FactoryRuntime(t *testing.T, run work.Epic8CapabilityMonitoringRun, id string) *v39.FactoryRuntimeVersion {
	t.Helper()
	record := assertEpic8GraphRecord(t, run, id, v39.TypeFactoryRuntimeVersion)
	runtime, ok := record.(*v39.FactoryRuntimeVersion)
	if !ok {
		t.Fatalf("%s type = %T; want FactoryRuntimeVersion", id, record)
	}
	return runtime
}

func assertEpic8GraphRecord(t *testing.T, run work.Epic8CapabilityMonitoringRun, id, typ string) v39.Record {
	t.Helper()
	record, err := run.EventGraph.Get(id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	if record.GetCommon().Type != typ {
		t.Fatalf("%s type = %q; want %q", id, record.GetCommon().Type, typ)
	}
	return record
}

func assertEpic8Edge(t *testing.T, edges []v39.CommonEdge, typ, toID string) {
	t.Helper()
	for _, edge := range edges {
		if edge.Type == typ && edge.ToID == toID {
			return
		}
	}
	t.Fatalf("edges = %#v; want %s to %s", edges, typ, toID)
}
