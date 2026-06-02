package work_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	assertEpic7RepairEvidence(t, run)
	assertEpic7CodeChangeEvidence(t, run)
	assertEpic7LocalFilesystemBoundary(t, run)
	assertEpic7RuntimeChangedFiles(t, run, nil)
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
	assertEpic7RuntimeChangedFiles(t, run, []string{"transpara-ai/work:proposal_validator.go", "transpara-ai/work:proposal_validator_test.go"})
}

func TestEpic7IssueToPRProposalTrialsRejectsForbiddenActions(t *testing.T) {
	for _, action := range []work.Epic7ProtectedAction{
		work.Epic7ActionPullRequestCreate,
		work.Epic7ActionBranchPush,
		work.Epic7ActionDefaultBranchPush,
		work.Epic7ActionPullRequestMerge,
		work.Epic7ActionProductionDeploy,
		work.Epic7ActionProtectedExecutionRun,
		work.Epic7ActionCapabilityActivate,
	} {
		t.Run(string(action), func(t *testing.T) {
			run := runEpic7(t, work.Epic7IssueToPROptions{CompletedForbiddenActions: []work.Epic7ProtectedAction{action}})
			assertEpic7Rejected(t, run)
			if !containsString(run.GateHValidation.Missing, "forbidden action completed: "+string(action)) {
				t.Fatalf("missing = %#v; want forbidden action %s", run.GateHValidation.Missing, action)
			}
			for _, trial := range run.Projection.Trials {
				if !containsString(run.GateHValidation.Missing, "forbidden action separation failed for "+trial.TrialID) {
					t.Fatalf("missing = %#v; want forbidden action propagated to %s", run.GateHValidation.Missing, trial.TrialID)
				}
			}
		})
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingProtectedActionBoundary(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{OmitProtectedAction: work.Epic7ActionCapabilityActivate})
	assertEpic7Rejected(t, run)
	for _, trial := range run.Projection.Trials {
		if !containsString(run.GateHValidation.Missing, "forbidden action separation failed for "+trial.TrialID) {
			t.Fatalf("missing = %#v; want missing protected action propagated to %s", run.GateHValidation.Missing, trial.TrialID)
		}
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
	if !containsString(run.GateHValidation.Missing, "multi-repo proposal authority evidence is missing") {
		t.Fatalf("missing = %#v; want multi-repo authority blocker", run.GateHValidation.Missing)
	}
	trial := findEpic7Trial(t, run, "trial_4_multi_repo_proposal_requires_explicit_authority")
	if trial.MultiRepoAuthority != nil || trial.ProofOfWorkPacket.MultiRepoAuthority != nil {
		t.Fatalf("multi-repo authority = %#v proof=%#v; want absent evidence", trial.MultiRepoAuthority, trial.ProofOfWorkPacket.MultiRepoAuthority)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingRepairEvidence(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{MissingRepairEvidence: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "repair evidence missing for trial_3_bug_fix_with_tests_and_repair_proposal") {
		t.Fatalf("missing = %#v; want repair evidence blocker", run.GateHValidation.Missing)
	}
	trial := findEpic7Trial(t, run, "trial_3_bug_fix_with_tests_and_repair_proposal")
	if trial.RepairEvidenceRef != "" {
		t.Fatalf("repair evidence ref = %q; want absent", trial.RepairEvidenceRef)
	}
	if trial.ProofOfWorkPacket.RepairEvidence == nil || trial.ProofOfWorkPacket.RepairEvidence.Status != "missing" {
		t.Fatalf("repair proof = %#v; want missing proof item", trial.ProofOfWorkPacket.RepairEvidence)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingRepairTestUpdateIntent(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{MissingRepairTestUpdateIntent: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "repair proposed test update missing for trial_3_bug_fix_with_tests_and_repair_proposal") {
		t.Fatalf("missing = %#v; want repair test update blocker", run.GateHValidation.Missing)
	}
	trial := findEpic7Trial(t, run, "trial_3_bug_fix_with_tests_and_repair_proposal")
	for _, intent := range trial.Proposal.ChangedFileIntent {
		if strings.HasSuffix(intent.Path, "_test.go") {
			t.Fatalf("changed-file intents = %#v; want no test update intent in negative seam", trial.Proposal.ChangedFileIntent)
		}
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingSelfImprovementReview(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{MissingSelfImprovementReview: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "self-improvement human review evidence is missing") {
		t.Fatalf("missing = %#v; want human review blocker", run.GateHValidation.Missing)
	}
	trial := findEpic7Trial(t, run, "trial_5_self_improvement_proposal_human_reviewed_rollback_bound")
	if trial.HumanReviewEvidenceRef != "" {
		t.Fatalf("human review ref = %q; want absent", trial.HumanReviewEvidenceRef)
	}
	if trial.ProofOfWorkPacket.HumanReviewEvidence == nil || trial.ProofOfWorkPacket.HumanReviewEvidence.Status != "missing" {
		t.Fatalf("human review proof = %#v; want missing proof item", trial.ProofOfWorkPacket.HumanReviewEvidence)
	}
	if trial.ProofOfWorkPacket.RollbackEvidence == nil || trial.ProofOfWorkPacket.RollbackEvidence.Status != "recorded" {
		t.Fatalf("rollback proof = %#v; want rollback still recorded", trial.ProofOfWorkPacket.RollbackEvidence)
	}
	if records := run.EventGraph.ByType(v39.TypeHumanReview); len(records) != 0 {
		t.Fatalf("HumanReview records = %#v; want none when review evidence is missing", records)
	}
}

func TestEpic7IssueToPRProposalTrialsRejectsMissingSelfImprovementRollback(t *testing.T) {
	run := runEpic7(t, work.Epic7IssueToPROptions{MissingSelfImprovementRollback: true})
	assertEpic7Rejected(t, run)
	if !containsString(run.GateHValidation.Missing, "self-improvement rollback evidence is missing") {
		t.Fatalf("missing = %#v; want rollback blocker", run.GateHValidation.Missing)
	}
	trial := findEpic7Trial(t, run, "trial_5_self_improvement_proposal_human_reviewed_rollback_bound")
	if trial.RollbackEvidenceRef != "" {
		t.Fatalf("rollback ref = %q; want absent", trial.RollbackEvidenceRef)
	}
	if trial.ProofOfWorkPacket.RollbackEvidence == nil || trial.ProofOfWorkPacket.RollbackEvidence.Status != "missing" {
		t.Fatalf("rollback proof = %#v; want missing proof item", trial.ProofOfWorkPacket.RollbackEvidence)
	}
	if trial.ProofOfWorkPacket.HumanReviewEvidence == nil || trial.ProofOfWorkPacket.HumanReviewEvidence.Status != "recorded" {
		t.Fatalf("human review proof = %#v; want human review still recorded", trial.ProofOfWorkPacket.HumanReviewEvidence)
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
	if !trial.Checks.ExplicitMultiRepoAuthorityRecorded || !trial.Checks.MultiRepoProposalRemainsProposedOnly {
		t.Fatalf("multi-repo checks = %#v; want explicit authority and proposed-only-with-authority", trial.Checks)
	}
	if len(trial.Proposal.ChangedFileIntent) < 2 {
		t.Fatalf("multi-repo intents = %#v; want at least two repos", trial.Proposal.ChangedFileIntent)
	}
	if trial.MultiRepoAuthority == nil {
		t.Fatalf("multi-repo authority is nil; want explicit authority grant")
	}
	if !containsString(trial.MultiRepoAuthority.Scope, "transpara-ai/work:proposal-only") || !containsString(trial.MultiRepoAuthority.Scope, "transpara-ai/docs:proposal-only") {
		t.Fatalf("multi-repo scope = %#v; want Work and docs proposal-only", trial.MultiRepoAuthority.Scope)
	}
	assertEpic7GraphRecord(t, run, trial.MultiRepoAuthority.AuthorityRequestID, v39.TypeAuthorityRequest)
	assertEpic7GraphRecord(t, run, trial.MultiRepoAuthority.AuthorityDecisionID, v39.TypeAuthorityDecision)
	assertEpic7GraphRecord(t, run, trial.MultiRepoAuthority.HumanApprovalID, v39.TypeHumanApproval)
	assertEpic7Edge(t, run.EventGraph.EdgesFrom(run.ActorInvocationID), v39.EdgeRequestedAuthority, trial.MultiRepoAuthority.AuthorityRequestID)
	assertEpic7Edge(t, run.EventGraph.EdgesFrom(trial.MultiRepoAuthority.AuthorityRequestID), v39.EdgeDecidedBy, trial.MultiRepoAuthority.AuthorityDecisionID)
	assertEpic7Edge(t, run.EventGraph.EdgesFrom(trial.MultiRepoAuthority.AuthorityRequestID), v39.EdgeApprovedBy, trial.MultiRepoAuthority.HumanApprovalID)
}

func assertEpic7SelfImprovementEvidence(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	trial := findEpic7Trial(t, run, "trial_5_self_improvement_proposal_human_reviewed_rollback_bound")
	if !trial.Checks.SelfImprovementHumanReviewPresent || !trial.Checks.SelfImprovementRollbackEvidencePresent || !trial.Checks.SelfImprovementProposalRemainsProposedOnly {
		t.Fatalf("self-improvement checks = %#v; want review and rollback boundary", trial.Checks)
	}
	if trial.HumanReviewEvidenceRef == "" || trial.RollbackEvidenceRef == "" {
		t.Fatalf("human review ref = %q rollback ref = %q; want both evidence refs", trial.HumanReviewEvidenceRef, trial.RollbackEvidenceRef)
	}
	assertFileExists(t, trial.HumanReviewEvidenceRef)
	assertFileExists(t, trial.RollbackEvidenceRef)
	assertEpic7GraphRecord(t, run, "review_epic7_"+trial.TrialID, v39.TypeHumanReview)
	assertEpic7GraphRecord(t, run, "art_epic7_human_review_"+trial.TrialID, v39.TypeArtifact)
}

func assertEpic7RepairEvidence(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	trial := findEpic7Trial(t, run, "trial_3_bug_fix_with_tests_and_repair_proposal")
	if !trial.Checks.RepairEvidencePresent || !trial.Checks.RepairTestUpdateIntentPresent {
		t.Fatalf("repair checks = %#v; want evidence and test update intent", trial.Checks)
	}
	if trial.RepairEvidenceRef == "" {
		t.Fatalf("repair ref is empty; want repair evidence")
	}
	assertFileExists(t, trial.RepairEvidenceRef)
	if trial.ProofOfWorkPacket.RepairEvidence == nil || trial.ProofOfWorkPacket.RepairEvidence.Status != "recorded" {
		t.Fatalf("repair proof = %#v; want recorded proof item", trial.ProofOfWorkPacket.RepairEvidence)
	}
	foundTestIntent := false
	for _, intent := range trial.Proposal.ChangedFileIntent {
		if intent.Path == "proposal_validator_test.go" && intent.ChangeType == "update" {
			foundTestIntent = true
		}
	}
	if !foundTestIntent {
		t.Fatalf("changed-file intents = %#v; want proposed test update", trial.Proposal.ChangedFileIntent)
	}
	assertEpic7GraphRecord(t, run, "art_epic7_repair_"+trial.TrialID, v39.TypeArtifact)
}

func assertEpic7CodeChangeEvidence(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	records := run.EventGraph.ByType(v39.TypeCodeChange)
	if len(records) != 7 {
		t.Fatalf("CodeChange records = %d; want one per changed-file intent", len(records))
	}
	seen := map[string]bool{}
	for _, record := range records {
		codeChange := record.(*v39.CodeChange)
		seen[codeChange.Repo+":"+codeChange.Path] = true
	}
	for _, want := range []string{
		"transpara-ai/docs:dark-factory/v3.9/implementation/operators/gate-h-handoff.md",
		"transpara-ai/work:proposal_validator.go",
		"transpara-ai/work:proposal_validator_test.go",
		"transpara-ai/docs:dark-factory/v3.9/implementation/operators/gate-h-followup.md",
		"transpara-ai/work:issue_pr_proposal_generator.go",
	} {
		if !seen[want] {
			t.Fatalf("CodeChange refs = %#v; want %s", seen, want)
		}
	}
}

func assertEpic7LocalFilesystemBoundary(t *testing.T, run work.Epic7IssueToPRRun) {
	t.Helper()
	root := filepath.Dir(filepath.Dir(filepath.Dir(run.LocalArtifacts.IssueDir)))
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			switch {
			case rel == "fixtures", rel == "fixtures/epic7", rel == "fixtures/epic7/issues":
				return nil
			case rel == "artifacts", rel == "artifacts/issue-pr":
				return nil
			case strings.HasPrefix(rel, "artifacts/issue-pr/"):
				return nil
			default:
				t.Fatalf("unexpected local directory %s under fixture root", rel)
			}
		}
		if !strings.HasPrefix(rel, "fixtures/epic7/issues/") && !strings.HasPrefix(rel, "artifacts/issue-pr/") {
			t.Fatalf("unexpected local file %s under fixture root", rel)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk local fixture root: %v", err)
	}
}

func assertEpic7RuntimeChangedFiles(t *testing.T, run work.Epic7IssueToPRRun, want []string) {
	t.Helper()
	record, err := run.EventGraph.Get(run.RuntimeResultID)
	if err != nil {
		t.Fatalf("get runtime result: %v", err)
	}
	runtimeResult, ok := record.(*v39.RuntimeResult)
	if !ok {
		t.Fatalf("runtime result type = %T; want RuntimeResult", record)
	}
	if len(want) == 0 {
		if len(runtimeResult.ChangedFiles) != 0 {
			t.Fatalf("changed files = %#v; want none for proposed-only run", runtimeResult.ChangedFiles)
		}
		return
	}
	for _, item := range want {
		if !containsString(runtimeResult.ChangedFiles, item) {
			t.Fatalf("changed files = %#v; want %s", runtimeResult.ChangedFiles, item)
		}
	}
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
				RepairEvidence *struct {
					Status      string `json:"status"`
					ArtifactRef string `json:"artifact_ref"`
				} `json:"repair_evidence"`
				RollbackEvidence *struct {
					Status      string `json:"status"`
					ArtifactRef string `json:"artifact_ref"`
				} `json:"rollback_evidence"`
				HumanReviewEvidence *struct {
					Status      string `json:"status"`
					ArtifactRef string `json:"artifact_ref"`
				} `json:"human_review_evidence"`
				MultiRepoAuthority *struct {
					Decision string   `json:"decision"`
					Scope    []string `json:"scope"`
				} `json:"multi_repo_authority"`
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
		switch trial.TrialID {
		case "trial_3_bug_fix_with_tests_and_repair_proposal":
			if trial.ProofOfWorkPacket.RepairEvidence == nil || trial.ProofOfWorkPacket.RepairEvidence.Status != "recorded" || trial.ProofOfWorkPacket.RepairEvidence.ArtifactRef == "" {
				t.Fatalf("%s repair proof = %#v; want recorded artifact", trial.TrialID, trial.ProofOfWorkPacket.RepairEvidence)
			}
		case "trial_4_multi_repo_proposal_requires_explicit_authority":
			if trial.ProofOfWorkPacket.MultiRepoAuthority == nil || trial.ProofOfWorkPacket.MultiRepoAuthority.Decision != "ApprovalRequired" || len(trial.ProofOfWorkPacket.MultiRepoAuthority.Scope) != 2 {
				t.Fatalf("%s multi-repo proof = %#v; want explicit authority", trial.TrialID, trial.ProofOfWorkPacket.MultiRepoAuthority)
			}
		case "trial_5_self_improvement_proposal_human_reviewed_rollback_bound":
			if trial.ProofOfWorkPacket.HumanReviewEvidence == nil || trial.ProofOfWorkPacket.HumanReviewEvidence.Status != "recorded" || trial.ProofOfWorkPacket.HumanReviewEvidence.ArtifactRef == "" {
				t.Fatalf("%s human review proof = %#v; want recorded artifact", trial.TrialID, trial.ProofOfWorkPacket.HumanReviewEvidence)
			}
			if trial.ProofOfWorkPacket.RollbackEvidence == nil || trial.ProofOfWorkPacket.RollbackEvidence.Status != "recorded" || trial.ProofOfWorkPacket.RollbackEvidence.ArtifactRef == "" {
				t.Fatalf("%s rollback proof = %#v; want recorded artifact", trial.TrialID, trial.ProofOfWorkPacket.RollbackEvidence)
			}
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

func assertEpic7GraphRecord(t *testing.T, run work.Epic7IssueToPRRun, id, typ string) v39.Record {
	t.Helper()
	record, err := run.EventGraph.Get(id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	if record.GetCommon().Type != typ {
		t.Fatalf("%s type = %s; want %s", id, record.GetCommon().Type, typ)
	}
	return record
}

func assertEpic7Edge(t *testing.T, edges []v39.CommonEdge, typ, toID string) {
	t.Helper()
	for _, edge := range edges {
		if edge.Type == typ && edge.ToID == toID {
			return
		}
	}
	t.Fatalf("edges = %#v; want %s edge to %s", edges, typ, toID)
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
