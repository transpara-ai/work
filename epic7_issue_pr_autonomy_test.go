package work_test

import (
	"encoding/json"
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic7IssueToPRProposalTrialsLocalEvidence(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{})

	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.GateHValidation.Status != "pass" || len(run.GateHValidation.Missing) != 0 {
		t.Fatalf("Gate H validation = %#v; want pass", run.GateHValidation)
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
	if len(run.Projection.Trials) != 5 {
		t.Fatalf("trials = %d; want 5", len(run.Projection.Trials))
	}
	for _, trial := range run.Projection.Trials {
		assertEpic7TrialProposedOnly(t, trial)
		assertFileExists(t, trial.IssueFixtureRef)
		assertFileExists(t, trial.ProposalPacketRef)
		assertFileExists(t, trial.ProofPacketRef)
		assertFileExists(t, trial.PatchRef)
		assertFileExists(t, trial.PRBodyRef)
		assertFileExists(t, trial.BranchPlanRef)
		assertFileExists(t, trial.ValidationPlanRef)
	}
	assertEpic7MultiRepoAuthorityEvidence(t, run)
	assertEpic7SelfImprovementEvidence(t, run)
	assertEpic7ProofJSONShape(t, run)
	assertEpic7NoExecutionReceipt(t, run)
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingIssueFixture(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{OmitIssueFixture: "trial_1_docs_only_issue_to_pr_proposal"})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "missing issue fixture trial_1_docs_only_issue_to_pr_proposal") {
		t.Fatalf("missing = %#v; want missing issue fixture", run.GateHValidation.Missing)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingProposalPacket(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{OmitProposalPacket: "trial_2_bounded_code_change_issue_to_pr_proposal"})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "missing proposal packet trial_2_bounded_code_change_issue_to_pr_proposal") {
		t.Fatalf("missing = %#v; want missing proposal packet", run.GateHValidation.Missing)
	}
	if !containsString(run.GateHValidation.Missing, "missing proof-of-work packet trial_2_bounded_code_change_issue_to_pr_proposal") {
		t.Fatalf("missing = %#v; want missing proof packet", run.GateHValidation.Missing)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsAppliedPatch(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{AppliedPatchTrial: "trial_3_bug_fix_with_tests_and_repair_proposal"})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "proposed-only boundary failed for trial_3_bug_fix_with_tests_and_repair_proposal") {
		t.Fatalf("missing = %#v; want proposed-only failure", run.GateHValidation.Missing)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsForbiddenActions(t *testing.T) {
	for _, action := range []work.Epic7ProtectedAction{
		work.Epic7ActionPullRequestCreate,
		work.Epic7ActionBranchPush,
		work.Epic7ActionDefaultBranchPush,
		work.Epic7ActionPullRequestMerge,
		work.Epic7ActionProductionDeploy,
		work.Epic7ActionProtectedExecutionRun,
	} {
		t.Run(string(action), func(t *testing.T) {
			run := runEpic7(t, work.Epic7IssueToPROptions{CompletedForbiddenActions: []work.Epic7ProtectedAction{action}})
			assertEpic7Rejected(t, run)
			if !containsString(run.GateHValidation.Missing, "forbidden action completed: "+string(action)) {
				t.Fatalf("missing = %#v; want forbidden action %s", run.GateHValidation.Missing, action)
			}
		})
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsExecutionReceipt(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{RecordExecutionReceipt: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "ExecutionReceipt recorded") {
		t.Fatalf("missing = %#v; want ExecutionReceipt blocker", run.GateHValidation.Missing)
	}
	if records := run.EventGraph.ByType(v39.TypeExecutionReceipt); len(records) != 1 {
		t.Fatalf("ExecutionReceipt records = %d; want injected forbidden evidence", len(records))
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingMultiRepoAuthority(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{MissingMultiRepoAuthority: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "multi-repo proposal without explicit authority was not rejected") {
		t.Fatalf("missing = %#v; want multi-repo authority blocker", run.GateHValidation.Missing)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingSelfImprovementReviewOrRollback(t *testing.T) {
	for name, opts := range map[string]work.Epic7IssueToPROptions{
		"review":   {MissingSelfImprovementReview: true},
		"rollback": {MissingSelfImprovementRollback: true},
	} {
		t.Run(name, func(t *testing.T) {
			run := runEpic7(t, opts)
			assertEpic7Rejected(t, run)
			if !containsString(run.GateHValidation.Missing, "self-improvement proposal with review and rollback evidence did not remain proposed-only") &&
				!containsString(run.GateHValidation.Missing, "self-improvement proposal without human review was not rejected") {
				t.Fatalf("missing = %#v; want self-improvement blocker", run.GateHValidation.Missing)
			}
		})
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	_, err := work.RunEpic7IssueToPRProposalTrials(ts, work.Epic7IssueToPROptions{ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "source actor is required") {
		t.Fatalf("missing source err = %v; want source requirement", err)
	}
	_, err = work.RunEpic7IssueToPRProposalTrials(ts, work.Epic7IssueToPROptions{Source: testActor, Causes: causes, WorkingDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("missing conversation err = %v; want conversation requirement", err)
	}
	_, err = work.RunEpic7IssueToPRProposalTrials(ts, work.Epic7IssueToPROptions{Source: testActor, ConversationID: testConv, Causes: causes})
	if err == nil || !strings.Contains(err.Error(), "working directory is required") {
		t.Fatalf("missing working dir err = %v; want local working dir requirement", err)
	}
	_, err = work.RunEpic7IssueToPRProposalTrials(ts, work.Epic7IssueToPROptions{Source: testActor, ConversationID: testConv, Causes: causes, WorkingDir: t.TempDir(), Mode: work.Epic7IssueToPRMode("live_pr")})
	if err == nil || !strings.Contains(err.Error(), "unsupported Epic 7 fixture mode") {
		t.Fatalf("unsupported mode err = %v; want mode rejection", err)
	}
}

func runEpic7(t *testing.T, opts work.Epic7IssueToPROptions) work.Epic7IssueToPRRun {
	t.Helper()
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	opts.Source = testActor
	opts.ConversationID = testConv
	opts.Causes = causes
	opts.WorkingDir = t.TempDir()
	run, err := work.RunEpic7IssueToPRProposalTrials(ts, opts)
	if err != nil {
		t.Fatalf("RunEpic7IssueToPRProposalTrials: %v", err)
	}
	return run
}

func assertEpic7Rejected(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	if run.Certification != nil || run.Rejection == nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want rejection only", run.Certification, run.Rejection)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if run.GateHValidation.Status != "fail" || len(run.GateHValidation.Missing) == 0 {
		t.Fatalf("Gate H validation = %#v; want fail with missing evidence", run.GateHValidation)
	}
}

func assertEpic7TrialProposedOnly(t *testing.T, trial work.Epic7TrialEvidence) {
	t.Helper()
	if trial.Status != "pass" || len(trial.Missing) != 0 {
		t.Fatalf("%s status=%s missing=%#v; want pass", trial.TrialID, trial.Status, trial.Missing)
	}
	if !trial.Proposal.ProposedOnly || trial.Proposal.Applied {
		t.Fatalf("%s proposal = %#v; want proposed-only", trial.TrialID, trial.Proposal)
	}
	if !trial.Checks.NoRepositoryMutation || !trial.Checks.NoExecutionReceipt {
		t.Fatalf("%s checks = %#v; want no mutation and no receipt", trial.TrialID, trial.Checks)
	}
	seen := map[work.Epic7ProtectedAction]string{}
	for _, action := range trial.AuthorityBoundary {
		seen[action.Action] = action.Status
	}
	if seen[work.Epic7ActionPullRequestPropose] != "proposed" {
		t.Fatalf("%s pull_request.propose status = %q; want proposed", trial.TrialID, seen[work.Epic7ActionPullRequestPropose])
	}
	for _, action := range []work.Epic7ProtectedAction{work.Epic7ActionPullRequestCreate, work.Epic7ActionBranchPush, work.Epic7ActionDefaultBranchPush, work.Epic7ActionPullRequestMerge, work.Epic7ActionProductionDeploy, work.Epic7ActionProtectedExecutionRun, work.Epic7ActionCapabilityActivate} {
		if seen[action] != "forbidden" {
			t.Fatalf("%s %s status = %q; want forbidden", trial.TrialID, action, seen[action])
		}
	}
}

func assertEpic7MultiRepoAuthorityEvidence(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	trial := findEpic7Trial(t, run, "trial_4_multi_repo_proposal_requires_explicit_authority")
	if !trial.Checks.MultiRepoWithoutAuthorityRejected || !trial.Checks.MultiRepoWithExplicitAuthorityOnly {
		t.Fatalf("multi-repo checks = %#v; want rejected-without-authority and proposed-only-with-authority", trial.Checks)
	}
	if len(trial.Proposal.ChangedFileIntent) < 2 {
		t.Fatalf("multi-repo intents = %#v; want at least two repos", trial.Proposal.ChangedFileIntent)
	}
}

func assertEpic7SelfImprovementEvidence(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	trial := findEpic7Trial(t, run, "trial_5_self_improvement_proposal_human_reviewed_rollback_bound")
	if !trial.Checks.SelfImprovementWithoutReviewRejected || !trial.Checks.SelfImprovementWithReviewRollbackOnly {
		t.Fatalf("self-improvement checks = %#v; want review and rollback boundary", trial.Checks)
	}
	if trial.RollbackEvidenceRef == "" {
		t.Fatalf("rollback ref is empty; want rollback evidence")
	}
	assertFileExists(t, trial.RollbackEvidenceRef)
}

func assertEpic7ProofJSONShape(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	payload, err := run.Projection.JSON()
	if err != nil {
		t.Fatalf("projection JSON: %v", err)
	}
	var decoded struct {
		Source          string `json:"source"`
		GateHValidation struct {
			Status string `json:"status"`
		} `json:"gate_h_validation"`
		ProofOfWorkPacket struct {
			Status           string `json:"status"`
			ForbiddenActions []struct {
				Action string `json:"action"`
				Status string `json:"status"`
			} `json:"forbidden_actions"`
			ResidualRisks []struct {
				Label  string `json:"label"`
				Status string `json:"status"`
			} `json:"residual_risks"`
		} `json:"proof_of_work_packet"`
		Trials []struct {
			TrialID  string `json:"trial_id"`
			Proposal struct {
				ProposedPRTitle    string `json:"proposed_pr_title"`
				ProposedPRBody     string `json:"proposed_pr_body"`
				ProposedBranchName string `json:"proposed_branch_name"`
				ProposedOnly       bool   `json:"proposed_only"`
				Applied            bool   `json:"applied"`
			} `json:"proposal"`
			ProofOfWorkPacket struct {
				IssueFixture struct {
					ArtifactRef string `json:"artifact_ref"`
				} `json:"issue_fixture"`
				PRProposal struct {
					Summary string `json:"summary"`
				} `json:"pr_proposal"`
				ValidationPlan struct {
					ArtifactRef string `json:"artifact_ref"`
				} `json:"validation_plan"`
				ForbiddenActionSeparation []struct {
					Action string `json:"action"`
					Status string `json:"status"`
				} `json:"forbidden_action_separation"`
			} `json:"proof_of_work_packet"`
		} `json:"trials"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal projection JSON: %v", err)
	}
	if decoded.Source != "work-epic7-issue-to-pr-proposal-fixture" || decoded.GateHValidation.Status != "pass" || decoded.ProofOfWorkPacket.Status != "pass" {
		t.Fatalf("decoded projection = %#v; want pass from Epic 7 source", decoded)
	}
	if len(decoded.Trials) != 5 {
		t.Fatalf("decoded trials = %d; want 5", len(decoded.Trials))
	}
	for _, trial := range decoded.Trials {
		if trial.Proposal.ProposedPRTitle == "" || trial.Proposal.ProposedPRBody == "" || trial.Proposal.ProposedBranchName == "" {
			t.Fatalf("%s proposal missing title/body/branch: %#v", trial.TrialID, trial.Proposal)
		}
		if !trial.Proposal.ProposedOnly || trial.Proposal.Applied {
			t.Fatalf("%s proposal flags = %#v; want proposed-only", trial.TrialID, trial.Proposal)
		}
		if trial.ProofOfWorkPacket.IssueFixture.ArtifactRef == "" || trial.ProofOfWorkPacket.ValidationPlan.ArtifactRef == "" || !strings.Contains(trial.ProofOfWorkPacket.PRProposal.Summary, "Gate H proposal:") {
			t.Fatalf("%s proof packet = %#v; want visible issue, PR, branch, and validation refs", trial.TrialID, trial.ProofOfWorkPacket)
		}
		if len(trial.ProofOfWorkPacket.ForbiddenActionSeparation) != 7 {
			t.Fatalf("%s forbidden separation count = %d; want 7", trial.TrialID, len(trial.ProofOfWorkPacket.ForbiddenActionSeparation))
		}
	}
	if len(decoded.ProofOfWorkPacket.ResidualRisks) != 3 {
		t.Fatalf("residual risks = %#v; want R-001/R-002/R-003", decoded.ProofOfWorkPacket.ResidualRisks)
	}
	for _, action := range decoded.ProofOfWorkPacket.ForbiddenActions {
		if action.Status != "forbidden" {
			t.Fatalf("forbidden action %s status = %q; want forbidden", action.Action, action.Status)
		}
	}
}

func assertEpic7NoExecutionReceipt(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	if records := run.EventGraph.ByType(v39.TypeExecutionReceipt); len(records) != 0 {
		t.Fatalf("ExecutionReceipt records = %#v; want none", records)
	}
}

func findEpic7Trial(t *testing.T, run work.Epic7IssueToPRRun, id string) work.Epic7TrialEvidence {
	t.Helper()
	for _, trial := range run.Projection.Trials {
		if trial.TrialID == id {
			return trial
		}
	}
	t.Fatalf("trial %s not found", id)
	return work.Epic7TrialEvidence{}
}
