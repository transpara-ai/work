package work_test

import (
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestBuildFactoryOrderDevelopmentProposalPreservesEvidence(t *testing.T) {
	opts := validFactoryOrderDevelopmentProposalOptions()

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}

	if proposal.FactoryOrder.ID != opts.FactoryOrderID {
		t.Fatalf("factory order id = %q, want %q", proposal.FactoryOrder.ID, opts.FactoryOrderID)
	}
	if proposal.FactoryOrder.TargetRepo != work.FactoryOrderProposalTargetRepo || proposal.FactoryOrder.TargetHead != opts.TargetHead {
		t.Fatalf("target = %s@%s, want %s@%s", proposal.FactoryOrder.TargetRepo, proposal.FactoryOrder.TargetHead, work.FactoryOrderProposalTargetRepo, opts.TargetHead)
	}
	if len(proposal.Requirements) != 1 || proposal.Requirements[0].ID != opts.RequirementID || proposal.Requirements[0].FactoryOrderID != opts.FactoryOrderID {
		t.Fatalf("requirements = %#v; want linked requirement", proposal.Requirements)
	}
	if len(proposal.AcceptanceCriteria) != 1 || proposal.AcceptanceCriteria[0].ID != opts.AcceptanceCriterionID || proposal.AcceptanceCriteria[0].RequirementID != opts.RequirementID {
		t.Fatalf("acceptance criteria = %#v; want linked acceptance criterion", proposal.AcceptanceCriteria)
	}
	if len(proposal.TaskDrafts) != 1 || proposal.TaskDrafts[0].ID != opts.TaskID || proposal.TaskDrafts[0].FactoryOrderID != opts.FactoryOrderID {
		t.Fatalf("task drafts = %#v; want linked task draft", proposal.TaskDrafts)
	}
	if len(proposal.ChangedFileIntent) != 3 {
		t.Fatalf("changed file intent count = %d, want 3", len(proposal.ChangedFileIntent))
	}
	for _, intent := range proposal.ChangedFileIntent {
		if intent.Repo != work.FactoryOrderProposalTargetRepo || !intent.ProposedOnly || intent.Applied {
			t.Fatalf("intent = %#v; want Work proposed-only unapplied intent", intent)
		}
	}
	if !proposal.ProposalArtifact.ProposedOnly || proposal.ProposalArtifact.Applied {
		t.Fatalf("proposal artifact flags = %#v; want proposed-only unapplied", proposal.ProposalArtifact)
	}
	assertUnavailable(t, proposal.ValidationResult, "validation")
	assertUnavailable(t, proposal.ProofOfWorkPacket.Branch, "branch")
	assertUnavailable(t, proposal.ProofOfWorkPacket.PullRequest, "pull request")
	assertUnavailable(t, proposal.ProofOfWorkPacket.CI, "ci")
	assertUnavailable(t, proposal.ProofOfWorkPacket.RuntimeInvocation, "runtime")
	assertUnavailable(t, proposal.ProofOfWorkPacket.ExecutionReceipt, "execution receipt")
	assertUnavailable(t, proposal.ProofOfWorkPacket.NativeEventGraphWrite, "EventGraph")
	if proposal.AuditReport.Status != "defer" || !containsString(proposal.AuditReport.ResidualRisks, "R-001 unresolved") {
		t.Fatalf("audit report = %#v; want defer with residual risks", proposal.AuditReport)
	}
	for _, claim := range proposal.AuditReport.ForbiddenClaims {
		if strings.Contains(strings.ToLower(claim), "level 1") {
			return
		}
	}
	t.Fatalf("forbidden claims = %#v; want Level 1 achievement exclusion", proposal.AuditReport.ForbiddenClaims)
}

func TestBuildFactoryOrderDevelopmentProposalRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*work.FactoryOrderDevelopmentProposalOptions)
		wantErr string
	}{
		{
			name: "missing factory order",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.FactoryOrderID = ""
			},
			wantErr: "factory_order_id is required",
		},
		{
			name: "bad requirement prefix",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.RequirementID = "requirement_bad"
			},
			wantErr: "requirement_id",
		},
		{
			name: "non Work target",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.TargetRepo = "transpara-ai/docs"
			},
			wantErr: "target_repo must be transpara-ai/work",
		},
		{
			name: "empty changed file intent",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent = nil
			},
			wantErr: "changed_file_intent must be non-empty",
		},
		{
			name: "non Work changed file intent",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].Repo = "transpara-ai/site"
			},
			wantErr: "changed_file_intent[0].repo",
		},
		{
			name: "proposal not marked proposed only",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].ProposedOnly = false
			},
			wantErr: "proposed_only must be true",
		},
		{
			name: "applied patch intent",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].Applied = true
			},
			wantErr: "applied must be false",
		},
		{
			name: "runtime invocation",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.RuntimeInvocationID = "run_df_v40_e7"
			},
			wantErr: "runtime_invocation_id is not allowed",
		},
		{
			name: "execution receipt",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ExecutionReceiptID = "exec_df_v40_e7"
			},
			wantErr: "execution_receipt_id is not allowed",
		},
		{
			name: "native EventGraph write",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.NativeEventGraphWriteRef = "eventgraph:write:df-v40-e7"
			},
			wantErr: "native_eventgraph_write_ref is not allowed",
		},
		{
			name: "protected action claim",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ProtectedActionClaims = []work.FactoryOrderProtectedActionClaim{{Action: "pull_request.merge", Status: "completed"}}
			},
			wantErr: "protected_action_claims are not allowed",
		},
		{
			name: "authority boundary claims authorization",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary[0].Status = "authorized"
			},
			wantErr: "claims protected action authority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := validFactoryOrderDevelopmentProposalOptions()
			tt.mutate(&opts)
			_, err := work.BuildFactoryOrderDevelopmentProposal(opts)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildFactoryOrderDevelopmentProposalCopiesInputs(t *testing.T) {
	opts := validFactoryOrderDevelopmentProposalOptions()

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}
	opts.ChangedFileIntent[0].Path = "mutated.go"
	opts.ValidationPlan[0] = "mutated"
	opts.AuthorityBoundary[0].Action = "mutated"

	if proposal.ChangedFileIntent[0].Path == "mutated.go" || proposal.ProposalArtifact.ChangedFileIntent[0].Path == "mutated.go" || proposal.ProofOfWorkPacket.ChangedFiles[0].Path == "mutated.go" {
		t.Fatalf("proposal retained caller changed-file slice alias: %#v", proposal)
	}
	if proposal.ValidationPlan[0] == "mutated" || proposal.ProofOfWorkPacket.Validation.Summary == "mutated" {
		t.Fatalf("proposal retained caller validation slice alias: %#v", proposal.ValidationPlan)
	}
	if proposal.AuthorityBoundary[0].Action == "mutated" || proposal.ProofOfWorkPacket.AuthorityBoundary[0].Action == "mutated" {
		t.Fatalf("proposal retained caller authority boundary alias: %#v", proposal.AuthorityBoundary)
	}
}

func validFactoryOrderDevelopmentProposalOptions() work.FactoryOrderDevelopmentProposalOptions {
	return work.FactoryOrderDevelopmentProposalOptions{
		SourceIntentRef:       "DF-V4.0-EPIC-006-TRIAL-EVIDENCE",
		Requester:             "Michael Saucier",
		TargetRepo:            work.FactoryOrderProposalTargetRepo,
		TargetHead:            "33750f7ca7aaab9bd0f8ba83e8835b99343164b7",
		FactoryOrderID:        "fo_df_v40_e6_work_proposal_001",
		RequirementID:         "req_df_v40_e6_proposal_evidence_path",
		AcceptanceCriterionID: "ac_df_v40_e6_proposal_evidence_path",
		TaskID:                "tsk_df_v40_e6_work_proposal_001",
		ChangedFileIntent: []work.FactoryOrderChangedFileIntent{
			{Repo: work.FactoryOrderProposalTargetRepo, Path: "factory_order_proposal.go", ChangeType: "add", Summary: "Define pure proposal evidence builder.", ProposedOnly: true},
			{Repo: work.FactoryOrderProposalTargetRepo, Path: "factory_order_proposal_test.go", ChangeType: "add", Summary: "Test linkage and forbidden evidence rejection.", ProposedOnly: true},
			{Repo: work.FactoryOrderProposalTargetRepo, Path: "docs/designs/factory-order-proposal-evidence-path.md", ChangeType: "add", Summary: "Document proposal evidence contract.", ProposedOnly: true},
		},
		ValidationPlan: []string{"go test ./...", "go vet ./...", "make verify"},
		AuthorityBoundary: []work.FactoryOrderProtectedActionBoundary{
			{Action: "runtime.execute", Status: "not_authorized", RequiredAuthority: "separate RuntimeBroker authority"},
			{Action: "native_eventgraph.write", Status: "not_authorized", RequiredAuthority: "separate EventGraph authority"},
			{Action: "pull_request.merge", Status: "not_authorized", RequiredAuthority: "Event 7 Work merge precondition"},
		},
	}
}

func assertUnavailable(t *testing.T, availability work.FactoryOrderProposalAvailability, label string) {
	t.Helper()
	if availability.Status != "unavailable" {
		t.Fatalf("%s availability = %#v; want unavailable", label, availability)
	}
	if strings.TrimSpace(availability.Reason) == "" {
		t.Fatalf("%s availability has empty reason: %#v", label, availability)
	}
}
