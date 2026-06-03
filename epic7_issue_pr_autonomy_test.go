package work_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestEpic11DocsDraftPRLiveMutationCreatesDraftPRAfterAuthorityPolicyAndRecordsReceipt(t *testing.T) {
	client := &epic11FakePRClient{}
	opts := validEpic11Options(t, client)
	run := runEpic11(t, opts)

	if client.calls != 1 {
		t.Fatalf("client calls = %d; want 1", client.calls)
	}
	if len(client.mutations) != 1 {
		t.Fatalf("mutations = %#v; want one", client.mutations)
	}
	mutation := client.mutations[0]
	if mutation.Repository != work.Epic11TargetRepository || mutation.BaseRef != work.Epic11TargetBaseRef || mutation.HeadRef != opts.Target.HeadRef || !mutation.Draft {
		t.Fatalf("mutation = %#v; want authorized draft PR target", mutation)
	}
	if mutation.TitleHash != hashForTest(opts.Target.Title) || mutation.BodyHash != hashForTest(opts.Target.Body) {
		t.Fatalf("mutation hashes = %q/%q; want title/body hashes", mutation.TitleHash, mutation.BodyHash)
	}
	if run.WorkProjection.Status != work.StatusCertified {
		t.Fatalf("work status = %q; want certified", run.WorkProjection.Status)
	}
	if run.MutationResult.Number == 0 || !run.MutationResult.Draft || run.MutationResult.State != "open" {
		t.Fatalf("mutation result = %#v; want open draft PR", run.MutationResult)
	}
	if run.ReceiptEvidence.AuthorityRequestRef != opts.AuthorityRequest.ID || run.ReceiptEvidence.AuthorityDecisionRef != opts.AuthorityDecision.ID || run.ReceiptEvidence.PolicyEngineAdapterDecisionRef != opts.PolicyDecision.DecisionID {
		t.Fatalf("receipt refs = %#v; want request/decision/policy refs", run.ReceiptEvidence)
	}
	if run.AuthorityReserve.AuthorityRequestRef != opts.AuthorityRequest.ID || run.AuthorityReserve.AuthorityDecisionRef != opts.AuthorityDecision.ID || run.AuthorityReserve.PolicyEngineAdapterDecisionRef != opts.PolicyDecision.DecisionID {
		t.Fatalf("authority reservation refs = %#v; want request/decision/policy refs", run.AuthorityReserve)
	}
	if run.AuthorityReserve.Result != "reserved" || run.AuthorityReserve.SingleUseNonce != opts.AuthorityDecision.SingleUseNonce {
		t.Fatalf("authority reservation = %#v; want reserved nonce", run.AuthorityReserve)
	}
	if run.ReceiptEvidence.SingleUseNonce != opts.AuthorityDecision.SingleUseNonce {
		t.Fatalf("receipt nonce = %q; want authority decision nonce", run.ReceiptEvidence.SingleUseNonce)
	}
	if run.ReceiptEvidence.Result != "succeeded" || run.ReceiptEvidence.PRURL == "" || !run.ReceiptEvidence.Draft {
		t.Fatalf("receipt evidence = %#v; want successful draft PR receipt", run.ReceiptEvidence)
	}
	if run.PolicyBundleID != work.Epic11PolicyBundleID || run.PolicyBundleHash != work.Epic11DocsDraftPRPolicyBundleHash() {
		t.Fatalf("policy bundle = %s/%s; want canonical bundle", run.PolicyBundleID, run.PolicyBundleHash)
	}
	for _, record := range []v39.Record{
		assertEpic7GraphRecord(t, work.Epic7IssueToPRRun{EventGraph: run.EventGraph}, opts.AuthorityRequest.ID, v39.TypeAuthorityRequest),
		assertEpic7GraphRecord(t, work.Epic7IssueToPRRun{EventGraph: run.EventGraph}, opts.AuthorityDecision.ID, v39.TypeAuthorityDecision),
		assertEpic7GraphRecord(t, work.Epic7IssueToPRRun{EventGraph: run.EventGraph}, opts.PolicyDecision.DecisionID, v39.TypePolicyEngineAdapterDecision),
		assertEpic7GraphRecord(t, work.Epic7IssueToPRRun{EventGraph: run.EventGraph}, run.ExecutionReceipt.CommonNode.ID, v39.TypeExecutionReceipt),
	} {
		if err := record.Validate(); err != nil {
			t.Fatalf("%s schema validation: %v", record.GetCommon().ID, err)
		}
	}
	assertEpic7Edge(t, run.EventGraph.EdgesFrom(opts.AuthorityRequest.ID), v39.EdgeDecidedBy, opts.AuthorityDecision.ID)
	assertEpic7Edge(t, run.EventGraph.EdgesFrom(opts.AuthorityDecision.ID), v39.EdgeReceiptedBy, run.ExecutionReceipt.CommonNode.ID)
	assertFileExists(t, filepath.Join(opts.WorkingDir, "artifacts", "epic11", "docs-draft-pr", "execution_receipt.json"))
	assertFileExists(t, filepath.Join(opts.WorkingDir, "artifacts", "epic11", "docs-draft-pr", "projection.json"))

	payload, err := run.Projection.JSON()
	if err != nil {
		t.Fatalf("projection JSON: %v", err)
	}
	var decoded struct {
		ForbiddenActions []string `json:"forbidden_actions"`
		ReceiptEvidence  struct {
			PRURL  string `json:"pr_url"`
			Result string `json:"result"`
		} `json:"receipt_evidence"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	for _, forbidden := range []string{"pull_request.merge", "repo.push.default_branch", "worktree.merge.main", "upstream.push"} {
		if !containsString(decoded.ForbiddenActions, forbidden) {
			t.Fatalf("forbidden actions = %#v; want %s", decoded.ForbiddenActions, forbidden)
		}
	}
	if decoded.ReceiptEvidence.PRURL == "" || decoded.ReceiptEvidence.Result != "succeeded" {
		t.Fatalf("decoded receipt = %#v; want succeeded URL", decoded.ReceiptEvidence)
	}
}

func TestEpic11DocsDraftPRLiveMutationBlocksBeforeGitHubCall(t *testing.T) {
	tests := []struct {
		name string
		edit func(*work.Epic11DocsDraftPROptions)
		want string
	}{
		{name: "missing authority decision", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityDecision.ID = "" }, want: "authority decision ID is required"},
		{name: "expired authority decision", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityDecision.ExpiresAt = opts.Now.Add(-time.Minute)
		}, want: "authority decision is expired"},
		{name: "different repo", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityDecision.TargetRepository = "transpara-ai/work"
		}, want: "authority decision repository"},
		{name: "different base sha", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityDecision.BaseSHA = "deadbeef" }, want: "authority decision base ref/SHA"},
		{name: "different head sha", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityDecision.HeadSHA = "deadbeef" }, want: "authority decision head ref/SHA"},
		{name: "title hash mismatch", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityRequest.TitleHash = hashForTest("different title")
		}, want: "authority request title hash"},
		{name: "body hash mismatch", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityDecision.BodyHash = hashForTest("different body")
		}, want: "authority decision body hash"},
		{name: "expired authority request", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityRequest.ExpiresAt = opts.Now.Add(-time.Minute)
		}, want: "authority request is expired"},
		{name: "actor differs from source", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.AuthorityRequest.ActorID = "act_other_epic11_actor"
		}, want: "authority request actor"},
		{name: "missing policy bundle hash", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.PolicyDecision.PolicyBundleHash = "" }, want: "policy bundle hash"},
		{name: "policy bundle ID mismatch", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.PolicyDecision.PolicyBundleID = "df-v3.9.20-wrong-bundle"
		}, want: "policy bundle ID"},
		{name: "stale policy bundle hash", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.PolicyDecision.PolicyBundleHash = "sha256:placeholder" }, want: "policy bundle hash"},
		{name: "policy forbidden", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.PolicyDecision.CanonicalDecision = "forbidden" }, want: "policy canonical decision"},
		{name: "head absent", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.Target.HeadExistsOnOrigin = false }, want: "head branch must already exist"},
		{name: "non draft", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.Target.Draft = false }, want: "draft=true is required"},
		{name: "nonce mismatch", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityDecision.SingleUseNonce = "nonce-other" }, want: "authority decision nonce"},
		{name: "forbidden merge action", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityRequest.Action = "pull_request.merge" }, want: "authority request action"},
		{name: "forbidden branch push action", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.AuthorityRequest.Action = "branch.push" }, want: "authority request action"},
		{name: "forbidden policy default branch push action", edit: func(opts *work.Epic11DocsDraftPROptions) {
			opts.PolicyDecision.ProtectedActionType = "repo.push.default_branch"
		}, want: "policy action"},
		{name: "prior receipt reuse", edit: func(opts *work.Epic11DocsDraftPROptions) { opts.PriorExecutionReceiptRefs = []string{"exec_previous"} }, want: "already used"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &epic11FakePRClient{}
			opts := validEpic11Options(t, client)
			tc.edit(&opts)
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			opts.Causes = causes
			_, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v; want containing %q", err, tc.want)
			}
			if client.calls != 0 {
				t.Fatalf("client calls = %d; want 0 before failed guard", client.calls)
			}
		})
	}
}

func TestEpic11DocsDraftPRLiveMutationBlocksDurableDecisionReuseBeforeGitHubCall(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	firstClient := &epic11FakePRClient{}
	firstOpts := validEpic11Options(t, firstClient)
	firstOpts.Causes = causes
	firstRun, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, firstOpts)
	if err != nil {
		t.Fatalf("first RunEpic11DocsDraftPRLiveMutation: %v", err)
	}
	if firstClient.calls != 1 || firstRun.ReceiptEvidence.AuthorityDecisionRef != firstOpts.AuthorityDecision.ID {
		t.Fatalf("first run calls=%d receipt=%#v; want one successful receipt", firstClient.calls, firstRun.ReceiptEvidence)
	}

	secondClient := &epic11FakePRClient{}
	secondOpts := validEpic11Options(t, secondClient)
	secondOpts.Causes = causes
	_, err = work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, secondOpts)
	if err == nil || !strings.Contains(err.Error(), "authority decision already used by durable authority refs") {
		t.Fatalf("second err = %v; want durable reuse rejection", err)
	}
	if secondClient.calls != 0 {
		t.Fatalf("second client calls = %d; want 0 before failed durable reuse guard", secondClient.calls)
	}
}

func TestEpic11DocsDraftPRLiveMutationReservationBlocksRetryAfterClientError(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	firstClient := &epic11FakePRClient{err: errors.New("github unavailable")}
	firstOpts := validEpic11Options(t, firstClient)
	firstOpts.Causes = causes
	_, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, firstOpts)
	if err == nil || !strings.Contains(err.Error(), "create draft PR") {
		t.Fatalf("first err = %v; want client failure after reservation", err)
	}
	if firstClient.calls != 1 {
		t.Fatalf("first client calls = %d; want 1", firstClient.calls)
	}

	secondClient := &epic11FakePRClient{}
	secondOpts := validEpic11Options(t, secondClient)
	secondOpts.Causes = causes
	_, err = work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, secondOpts)
	if err == nil || !strings.Contains(err.Error(), "authority decision already used by durable authority refs") {
		t.Fatalf("second err = %v; want reservation reuse rejection", err)
	}
	if secondClient.calls != 0 {
		t.Fatalf("second client calls = %d; want 0 before reservation replay", secondClient.calls)
	}
}

func TestEpic11DocsDraftPRLiveMutationMalformedDurableAuthorityArtifactsFailClosed(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{
			name:  "malformed reservation artifact",
			label: "epic11_docs_draft_pr_authority_reservation",
			want:  "decode Epic 11 authority reservation artifact",
		},
		{
			name:  "malformed receipt artifact",
			label: "epic11_docs_draft_pr_execution_receipt",
			want:  "decode Epic 11 execution receipt artifact",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task, err := ts.Create(testActor, "Malformed Epic 11 durable artifact", "", causes, testConv)
			if err != nil {
				t.Fatalf("Create poison task: %v", err)
			}
			if err := ts.AddArtifact(testActor, task.ID, tc.label, "application/json", "{not-json", causes, testConv); err != nil {
				t.Fatalf("AddArtifact poison: %v", err)
			}

			client := &epic11FakePRClient{}
			opts := validEpic11Options(t, client)
			opts.Causes = causes
			_, err = work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v; want containing %q", err, tc.want)
			}
			if client.calls != 0 {
				t.Fatalf("client calls = %d; want 0 for malformed durable artifact", client.calls)
			}
		})
	}
}

func TestEpic11DocsDraftPRLiveMutationReservationBlocksConcurrentSameNonceBeforeGitHubCall(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	firstClient := newEpic11BlockingPRClient()
	firstOpts := validEpic11Options(t, firstClient)
	firstOpts.Causes = causes
	firstErr := make(chan error, 1)
	go func() {
		_, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, firstOpts)
		firstErr <- err
	}()

	select {
	case <-firstClient.entered:
	case <-time.After(time.Second):
		t.Fatal("first client was not called after reservation")
	}

	secondClient := &epic11FakePRClient{}
	secondOpts := validEpic11Options(t, secondClient)
	secondOpts.Causes = causes
	_, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, secondOpts)
	if err == nil || !strings.Contains(err.Error(), "authority decision already used by durable authority refs") {
		t.Fatalf("second err = %v; want reservation conflict", err)
	}
	if secondClient.calls != 0 {
		t.Fatalf("second client calls = %d; want 0 while first run holds reservation", secondClient.calls)
	}

	close(firstClient.release)
	select {
	case err := <-firstErr:
		if err != nil {
			t.Fatalf("first run err = %v; want success", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first run did not finish")
	}
	if calls := firstClient.Calls(); calls != 1 {
		t.Fatalf("first client calls = %d; want 1", calls)
	}
}

func TestEpic11DocsDraftPRLiveMutationRejectsUnexpectedGitHubResponse(t *testing.T) {
	client := &epic11FakePRClient{}
	opts := validEpic11Options(t, client)
	client.result = epic11SuccessfulResult(opts)
	client.result.Draft = false
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	opts.Causes = causes
	_, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, opts)
	if err == nil || !strings.Contains(err.Error(), "created PR is not draft") {
		t.Fatalf("err = %v; want non-draft response rejection", err)
	}
	if client.calls != 1 {
		t.Fatalf("client calls = %d; want 1 for post-response validation", client.calls)
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

func runEpic11(t *testing.T, opts work.Epic11DocsDraftPROptions) work.Epic11DocsDraftPRRun {
	t.Helper()
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	opts.Causes = causes
	run, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, opts)
	if err != nil {
		t.Fatalf("RunEpic11DocsDraftPRLiveMutation: %v", err)
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

type epic11FakePRClient struct {
	calls     int
	mutations []work.Epic11DraftPullRequestMutation
	result    work.Epic11DraftPullRequestResult
	err       error
}

func (c *epic11FakePRClient) CreateDraftPullRequest(_ context.Context, mutation work.Epic11DraftPullRequestMutation) (work.Epic11DraftPullRequestResult, error) {
	c.calls++
	c.mutations = append(c.mutations, mutation)
	if c.err != nil {
		return work.Epic11DraftPullRequestResult{}, c.err
	}
	if c.result.Number != 0 || c.result.URL != "" {
		return c.result, nil
	}
	return work.Epic11DraftPullRequestResult{
		Repository:                   mutation.Repository,
		Number:                       111,
		URL:                          "https://github.com/transpara-ai/docs/pull/111",
		GitHubResponseIDOrEquivalent: "github-pr-node-111",
		BaseRef:                      mutation.BaseRef,
		BaseSHA:                      mutation.BaseSHA,
		HeadRef:                      mutation.HeadRef,
		HeadSHA:                      mutation.HeadSHA,
		Draft:                        true,
		State:                        "open",
		CreatedAt:                    time.Date(2026, 6, 3, 13, 0, 1, 0, time.UTC),
	}, nil
}

type epic11BlockingPRClient struct {
	mu      sync.Mutex
	calls   int
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func newEpic11BlockingPRClient() *epic11BlockingPRClient {
	return &epic11BlockingPRClient{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (c *epic11BlockingPRClient) CreateDraftPullRequest(ctx context.Context, mutation work.Epic11DraftPullRequestMutation) (work.Epic11DraftPullRequestResult, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	c.once.Do(func() { close(c.entered) })
	select {
	case <-c.release:
	case <-ctx.Done():
		return work.Epic11DraftPullRequestResult{}, ctx.Err()
	}
	return work.Epic11DraftPullRequestResult{
		Repository:                   mutation.Repository,
		Number:                       222,
		URL:                          "https://github.com/transpara-ai/docs/pull/222",
		GitHubResponseIDOrEquivalent: "github-pr-node-222",
		BaseRef:                      mutation.BaseRef,
		BaseSHA:                      mutation.BaseSHA,
		HeadRef:                      mutation.HeadRef,
		HeadSHA:                      mutation.HeadSHA,
		Draft:                        true,
		State:                        "open",
		CreatedAt:                    time.Date(2026, 6, 3, 13, 0, 2, 0, time.UTC),
	}, nil
}

func (c *epic11BlockingPRClient) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func validEpic11Options(t *testing.T, client work.Epic11PullRequestCreator) work.Epic11DocsDraftPROptions {
	t.Helper()
	now := time.Date(2026, 6, 3, 13, 0, 0, 0, time.UTC)
	target := work.Epic11DraftPullRequestTarget{
		Repository:             work.Epic11TargetRepository,
		BaseRef:                work.Epic11TargetBaseRef,
		BaseSHA:                "b21e2eca5ce547eebef83a1a392f5ca790c3e44d",
		HeadRef:                "codex/epic-11-docs-draft-pr-live-mutation-fixture",
		HeadSHA:                "b4f9844ecad41a8dc1298e3ac19df3a4e7ac9071",
		HeadExistsOnOrigin:     true,
		Title:                  "[codex] Epic 11 fixture draft PR",
		Body:                   "## Summary\n\nFixture body for the guarded Epic 11 draft PR creation seam.\n",
		ChangedFiles:           []string{"dark-factory/v3.9/implementation/epics/epic-11-docs-draft-pr-live-mutation/README.md"},
		ValidationEvidenceRefs: []string{"git diff --check", "go test ./...", "make verify"},
		Draft:                  true,
		MaintainerCanModify:    true,
		RollbackInstructions:   "Manual rollback only: human may close the draft PR in GitHub after a separately authorized mutation.",
	}
	req := work.Epic11AuthorityRequestEvidence{
		ID:                     "auth_req_epic11_docs_draft_pr_create",
		ActorID:                testActor.Value(),
		ActorRole:              "codex",
		Action:                 work.Epic11ActionPullRequestCreate,
		TargetRepository:       target.Repository,
		BaseRef:                target.BaseRef,
		BaseSHA:                target.BaseSHA,
		HeadRef:                target.HeadRef,
		HeadSHA:                target.HeadSHA,
		TitleHash:              hashForTest(target.Title),
		BodyHash:               hashForTest(target.Body),
		ChangedFiles:           append([]string(nil), target.ChangedFiles...),
		ValidationEvidenceRefs: append([]string(nil), target.ValidationEvidenceRefs...),
		PolicyBundleID:         work.Epic11PolicyBundleID,
		PolicyBundleHash:       work.Epic11DocsDraftPRPolicyBundleHash(),
		RollbackInstructions:   target.RollbackInstructions,
		SingleUseNonce:         "nonce-epic11-docs-draft-pr-create",
		RequestedAt:            now.Add(-time.Minute),
		ExpiresAt:              now.Add(time.Hour),
	}
	decision := work.Epic11AuthorityDecisionEvidence{
		ID:                     "auth_dec_epic11_docs_draft_pr_create",
		AuthorityRequestID:     req.ID,
		ActorID:                req.ActorID,
		ActorRole:              req.ActorRole,
		DeciderActorID:         "act_human_epic11_authorizer",
		DeciderRole:            "maintainer",
		Decision:               "ApprovalRequired",
		Action:                 req.Action,
		TargetRepository:       req.TargetRepository,
		BaseRef:                req.BaseRef,
		BaseSHA:                req.BaseSHA,
		HeadRef:                req.HeadRef,
		HeadSHA:                req.HeadSHA,
		TitleHash:              req.TitleHash,
		BodyHash:               req.BodyHash,
		ChangedFiles:           append([]string(nil), req.ChangedFiles...),
		ValidationEvidenceRefs: append([]string(nil), req.ValidationEvidenceRefs...),
		PolicyBundleID:         req.PolicyBundleID,
		PolicyBundleHash:       req.PolicyBundleHash,
		RollbackInstructions:   req.RollbackInstructions,
		SingleUseNonce:         req.SingleUseNonce,
		ExpiresAt:              now.Add(time.Hour),
	}
	policy := work.Epic11PolicyDecisionEvidence{
		DecisionID:           "padc_epic11_docs_draft_pr_create",
		AdapterID:            work.Epic11PolicyAdapterID,
		AdapterVersion:       "1.0.0",
		PolicyBundleID:       work.Epic11PolicyBundleID,
		PolicyBundleHash:     work.Epic11DocsDraftPRPolicyBundleHash(),
		ProtectedActionType:  work.Epic11ActionPullRequestCreate,
		ActorID:              req.ActorID,
		ResourceRefs:         []string{target.Repository, target.BaseRef, target.HeadRef},
		InputFacts:           map[string]any{"repository": target.Repository, "base_sha": target.BaseSHA, "head_sha": target.HeadSHA, "draft": target.Draft},
		RawDecision:          "allow draft PR creation only after exact JIT authority match",
		CanonicalDecision:    "approval_required",
		ReasonCodes:          []string{"docs95_authorized", "exact_target_match", "draft_required", "single_use_nonce"},
		EvidenceRefs:         []string{req.ID, decision.ID, "transpara-ai/docs#95"},
		LatencyMS:            1,
		AuthorityDecisionRef: decision.ID,
	}
	return work.Epic11DocsDraftPROptions{
		Source:            testActor,
		ConversationID:    testConv,
		WorkingDir:        t.TempDir(),
		Client:            client,
		Now:               now,
		Target:            target,
		AuthorityRequest:  req,
		AuthorityDecision: decision,
		PolicyDecision:    policy,
	}
}

func epic11SuccessfulResult(opts work.Epic11DocsDraftPROptions) work.Epic11DraftPullRequestResult {
	return work.Epic11DraftPullRequestResult{
		Repository:                   opts.Target.Repository,
		Number:                       111,
		URL:                          "https://github.com/transpara-ai/docs/pull/111",
		GitHubResponseIDOrEquivalent: "github-pr-node-111",
		BaseRef:                      opts.Target.BaseRef,
		BaseSHA:                      opts.Target.BaseSHA,
		HeadRef:                      opts.Target.HeadRef,
		HeadSHA:                      opts.Target.HeadSHA,
		Draft:                        true,
		State:                        "open",
		CreatedAt:                    opts.Now.Add(time.Second),
	}
}

func hashForTest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
