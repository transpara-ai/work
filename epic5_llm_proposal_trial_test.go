package work_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic5BoundedLLMProposalTrialReviewOnly(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic5LLMProposalReviewOnly,
	})
	if err != nil {
		t.Fatalf("RunEpic5BoundedLLMProposalTrial: %v", err)
	}

	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessPassed || !run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want completed pass", run.TraceCompleteness)
	}
	if run.GateFValidation.Status != "pass" || len(run.GateFValidation.Missing) != 0 {
		t.Fatalf("Gate F validation = %#v; want pass with no missing evidence", run.GateFValidation)
	}
	if run.WorkProjection.Status != work.StatusCertified {
		t.Fatalf("work status = %q; want certified", run.WorkProjection.Status)
	}
	if run.PromptHash == "" || run.ResponseHash == "" || run.InputContractHash == "" || run.OutputContractHash == "" {
		t.Fatalf("hashes missing: prompt=%q response=%q input=%q output=%q", run.PromptHash, run.ResponseHash, run.InputContractHash, run.OutputContractHash)
	}
	if run.Projection.LLMInvocation == nil {
		t.Fatal("projection LLMInvocation is nil; want recorded invocation")
	}
	if run.Projection.LLMInvocation.ModelLabel == "" || run.Projection.LLMInvocation.ProviderLabel != "recorded_fixture_provider" {
		t.Fatalf("llm labels = %#v; want recorded fixture provider", run.Projection.LLMInvocation)
	}
	assertFileExists(t, run.LocalArtifacts.PromptPath)
	assertFileExists(t, run.LocalArtifacts.ResponsePath)
	assertFileExists(t, run.LocalArtifacts.ProposalPath)
	assertFileExists(t, run.LocalArtifacts.PatchPath)
	assertEpic5CapabilityAndKnowledgeEvidence(t, run)
	assertEpic5ProposedOnlyBoundary(t, run)
	assertEpic5ProofOfWorkJSONShape(t, run, "pass")
}

func TestEpic5BoundedLLMProposalTrialMissingInvocationFailsGateF(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic5LLMProposalMissingInvocation,
	})
	if err != nil {
		t.Fatalf("RunEpic5BoundedLLMProposalTrial missing invocation: %v", err)
	}

	if run.Certification != nil || run.Rejection == nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want rejection only", run.Certification, run.Rejection)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if records := run.EventGraph.ByType(v39.TypeActorInvocation); len(records) != 0 {
		t.Fatalf("ActorInvocation records = %#v; want none for missing invocation mode", records)
	}
	gate := getEpic5GateResult(t, run)
	if statusValue(gate.CommonNode.Status) != "fail" {
		t.Fatalf("gate status = %q; want fail", statusValue(gate.CommonNode.Status))
	}
	if run.GateFValidation.Status != "fail" {
		t.Fatalf("Gate F validation status = %q; want fail", run.GateFValidation.Status)
	}
	if !containsString(run.GateFValidation.Missing, "USED_ENVELOPE from tsk_epic5_llm_proposal_missing_invocation") {
		t.Fatalf("Gate F missing evidence = %#v; want trace-derived missing runtime envelope", run.GateFValidation.Missing)
	}
	if !containsString(run.GateFValidation.Missing, "recorded LLM ActorInvocation evidence") {
		t.Fatalf("Gate F missing evidence = %#v; want missing invocation evidence", run.GateFValidation.Missing)
	}
	if !containsString(run.Projection.Errors, "recorded LLM invocation missing") {
		t.Fatalf("errors = %#v; want missing invocation", run.Projection.Errors)
	}
	assertEpic5NoExecutionReceipt(t, run)
}

func TestEpic5BoundedLLMProposalTrialAppliedPatchFailsGateF(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic5LLMProposalAppliedPatch,
	})
	if err != nil {
		t.Fatalf("RunEpic5BoundedLLMProposalTrial applied patch: %v", err)
	}

	if run.Certification != nil || run.Rejection == nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want rejection only", run.Certification, run.Rejection)
	}
	if !run.ProposalApplied || !run.Projection.Proposal.Applied || run.Projection.Proposal.ProposedOnly {
		t.Fatalf("proposal applied/proposed-only flags = run:%v projection:%#v; want applied failure", run.ProposalApplied, run.Projection.Proposal)
	}
	records := run.EventGraph.ByType(v39.TypeCodeChange)
	if len(records) != 1 {
		t.Fatalf("CodeChange records = %d; want 1", len(records))
	}
	codeChange := records[0].(*v39.CodeChange)
	if statusValue(codeChange.CommonNode.Status) != "applied" {
		t.Fatalf("code change status = %q; want applied", statusValue(codeChange.CommonNode.Status))
	}
	gate := getEpic5GateResult(t, run)
	if statusValue(gate.CommonNode.Status) != "fail" {
		t.Fatalf("gate status = %q; want fail", statusValue(gate.CommonNode.Status))
	}
	if run.GateFValidation.Status != "fail" {
		t.Fatalf("Gate F validation status = %q; want fail", run.GateFValidation.Status)
	}
	if !containsString(run.GateFValidation.Missing, "proposed-only boundary: CodeChange status is applied") {
		t.Fatalf("Gate F missing evidence = %#v; want applied-proposal boundary failure", run.GateFValidation.Missing)
	}
	assertEpic5NoExecutionReceipt(t, run)
}

func TestEpic5BoundedLLMProposalTrialRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "source actor is required") {
		t.Fatalf("missing source err = %v; want source requirement", err)
	}

	_, err = work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:     testActor,
		Causes:     causes,
		WorkingDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("missing conversation err = %v; want conversation requirement", err)
	}

	_, err = work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		Mode:           work.Epic5LLMProposalReviewOnly,
	})
	if err == nil || !strings.Contains(err.Error(), "working directory is required") {
		t.Fatalf("missing working dir err = %v; want local working dir requirement", err)
	}

	_, err = work.RunEpic5BoundedLLMProposalTrial(ts, work.Epic5LLMProposalOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic5LLMProposalMode("live_provider"),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported Epic 5 fixture mode") {
		t.Fatalf("unsupported mode err = %v; want mode rejection", err)
	}
}

func assertEpic5CapabilityAndKnowledgeEvidence(t *testing.T, run work.Epic5LLMProposalRun) {
	t.Helper()
	if !run.CapabilityUsagePath.Completed || len(run.CapabilityUsagePath.Missing) != 0 {
		t.Fatalf("capability usage path = %#v; want complete", run.CapabilityUsagePath)
	}
	if !run.KnowledgePath.Completed || len(run.KnowledgePath.Missing) != 0 {
		t.Fatalf("knowledge path = %#v; want complete", run.KnowledgePath)
	}
	if records := run.EventGraph.ByType(v39.TypeCapabilityArtifact); len(records) != 1 {
		t.Fatalf("CapabilityArtifact records = %d; want 1", len(records))
	}
	if records := run.EventGraph.ByType(v39.TypeKnowledgeReference); len(records) != 1 {
		t.Fatalf("KnowledgeReference records = %d; want 1", len(records))
	}
	taskRecord, err := run.EventGraph.Get(run.TaskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	task := taskRecord.(*v39.Task)
	if !containsString(task.CommonNode.SourceRefs, run.CapabilityArtifactID) {
		t.Fatalf("task source refs = %#v; want capability artifact", task.CommonNode.SourceRefs)
	}
	if !containsString(task.CommonNode.SourceRefs, "knowledge:dark-factory/v3.9/implementation/epics/epic-05-gate-f-bounded-llm-proposal-trial/01-work-llm-proposal-implementation-authorization-v3.9.md") {
		t.Fatalf("task source refs = %#v; want knowledge source ref", task.CommonNode.SourceRefs)
	}
}

func assertEpic5ProposedOnlyBoundary(t *testing.T, run work.Epic5LLMProposalRun) {
	t.Helper()
	records := run.EventGraph.ByType(v39.TypeCodeChange)
	if len(records) != 1 {
		t.Fatalf("CodeChange records = %d; want 1", len(records))
	}
	codeChange := records[0].(*v39.CodeChange)
	if statusValue(codeChange.CommonNode.Status) != "proposed" {
		t.Fatalf("code change status = %q; want proposed", statusValue(codeChange.CommonNode.Status))
	}
	if codeChange.Repo != "transpara-ai/work" || codeChange.Path != "docs/designs/epic5-llm-proposal-output.md" {
		t.Fatalf("code change target = %s:%s; want Work proposal target", codeChange.Repo, codeChange.Path)
	}
	if !run.Projection.Proposal.ProposedOnly || run.Projection.Proposal.Applied {
		t.Fatalf("proposal = %#v; want proposed-only unapplied", run.Projection.Proposal)
	}
	if run.HumanApproval == nil || run.HumanApproval.Decision != "approved" {
		t.Fatalf("human approval = %#v; want review-only approval", run.HumanApproval)
	}
	assertEpic5NoExecutionReceipt(t, run)
}

func assertEpic5NoExecutionReceipt(t *testing.T, run work.Epic5LLMProposalRun) {
	t.Helper()
	if records := run.EventGraph.ByType(v39.TypeExecutionReceipt); len(records) != 0 {
		t.Fatalf("ExecutionReceipt records = %#v; want none", records)
	}
}

func assertEpic5ProofOfWorkJSONShape(t *testing.T, run work.Epic5LLMProposalRun, status string) {
	t.Helper()
	payload, err := run.Projection.JSON()
	if err != nil {
		t.Fatalf("projection JSON: %v", err)
	}
	var decoded struct {
		Source        string `json:"source"`
		LLMInvocation struct {
			ModelLabel    string `json:"model_label"`
			ProviderLabel string `json:"provider_label"`
			PromptHash    string `json:"prompt_hash"`
			ResponseHash  string `json:"response_hash"`
		} `json:"llm_invocation"`
		Proposal struct {
			ProposedOnly bool `json:"proposed_only"`
			Applied      bool `json:"applied"`
		} `json:"proposal"`
		ProofOfWorkPacket struct {
			Status          string `json:"status"`
			LLMContribution struct {
				Summary string `json:"summary"`
			} `json:"llm_contribution"`
			NonExecutionProof []struct {
				Label  string `json:"label"`
				Status string `json:"status"`
			} `json:"non_execution_proof"`
		} `json:"proof_of_work_packet"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal projection JSON: %v", err)
	}
	if decoded.Source != "work-epic5-recorded-llm-proposal-fixture" {
		t.Fatalf("source = %q; want Epic 5 fixture source", decoded.Source)
	}
	if decoded.LLMInvocation.ModelLabel == "" || decoded.LLMInvocation.ProviderLabel != "recorded_fixture_provider" {
		t.Fatalf("llm invocation = %#v; want recorded fixture provider", decoded.LLMInvocation)
	}
	if decoded.LLMInvocation.PromptHash == "" || decoded.LLMInvocation.ResponseHash == "" {
		t.Fatalf("llm hashes missing: %#v", decoded.LLMInvocation)
	}
	if !decoded.Proposal.ProposedOnly || decoded.Proposal.Applied {
		t.Fatalf("proposal flags = %#v; want proposed-only", decoded.Proposal)
	}
	if decoded.ProofOfWorkPacket.Status != status {
		t.Fatalf("proof packet status = %q; want %q", decoded.ProofOfWorkPacket.Status, status)
	}
	if !strings.Contains(decoded.ProofOfWorkPacket.LLMContribution.Summary, "Recorded LLM") {
		t.Fatalf("llm contribution summary = %q; want visible LLM contribution", decoded.ProofOfWorkPacket.LLMContribution.Summary)
	}
	foundNoReceipt := false
	for _, item := range decoded.ProofOfWorkPacket.NonExecutionProof {
		if item.Label == "No ExecutionReceipt" && item.Status == "pass" {
			foundNoReceipt = true
		}
	}
	if !foundNoReceipt {
		t.Fatalf("non-execution proof = %#v; want no ExecutionReceipt evidence", decoded.ProofOfWorkPacket.NonExecutionProof)
	}
}

func getEpic5GateResult(t *testing.T, run work.Epic5LLMProposalRun) *v39.GateResult {
	t.Helper()
	record, err := run.EventGraph.Get(run.GateResultID)
	if err != nil {
		t.Fatalf("get gate result: %v", err)
	}
	gate, ok := record.(*v39.GateResult)
	if !ok {
		t.Fatalf("gate record type = %T; want GateResult", record)
	}
	return gate
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected local artifact %s to exist", path)
		}
		t.Fatalf("stat %s: %v", path, err)
	}
}
