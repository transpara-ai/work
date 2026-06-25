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
	if len(proposal.ProofOfWorkPacket.SourceIssueRefs) != 0 {
		t.Fatalf("proof source refs = %#v; want empty without issue records", proposal.ProofOfWorkPacket.SourceIssueRefs)
	}
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

func TestBuildFactoryOrderDevelopmentProposalDerivesIssueEvidence(t *testing.T) {
	opts := validFactoryOrderDevelopmentProposalOptions()
	opts.SourceIntentRef = "github:transpara-ai/work#61"
	opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{
		{
			Repo:   " transpara-ai/work ",
			Number: 61,
			URL:    " https://github.com/transpara-ai/work/issues/61 ",
			Title:  " Requirements and task-draft derivation from issue records ",
			Goal:   " Derive requirements, acceptance criteria, assumptions, ambiguities, risk notes, and production-cell task drafts from issue records. ",
			AcceptanceCriteria: []string{
				"source issue refs are preserved",
				"no implementation starts",
			},
			Assumptions: []string{"issue body is caller-supplied scanner evidence"},
			Ambiguities: []string{"authority-sensitive issues still require separate scope"},
			RiskNotes:   []string{"issue records are source intent, not authority"},
			Labels:      []string{"cc:intake", "cc:civilization-presence"},
		},
	}

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}

	if len(proposal.IssueSourceRecords) != 1 {
		t.Fatalf("issue records = %#v; want one normalized source issue", proposal.IssueSourceRecords)
	}
	issue := proposal.IssueSourceRecords[0]
	if issue.Repo != "transpara-ai/work" || issue.Number != 61 || issue.SourceRefs[0] != "transpara-ai/work#61" {
		t.Fatalf("issue source = %#v; want normalized work#61 source", issue)
	}
	if proposal.Requirements[0].Source != "github_issue" || !strings.Contains(proposal.Requirements[0].Text, "transpara-ai/work#61") {
		t.Fatalf("requirement = %#v; want github issue-derived text", proposal.Requirements[0])
	}
	if proposal.AcceptanceCriteria[0].RequiredEvidenceType != "github_issue_derived_factory_order_proposal_evidence" {
		t.Fatalf("required evidence = %q", proposal.AcceptanceCriteria[0].RequiredEvidenceType)
	}
	if !strings.Contains(proposal.AcceptanceCriteria[0].Text, "no implementation starts") {
		t.Fatalf("acceptance text = %q; want issue acceptance criteria", proposal.AcceptanceCriteria[0].Text)
	}
	if len(proposal.TaskDrafts) != 1 {
		t.Fatalf("task drafts = %#v", proposal.TaskDrafts)
	}
	draft := proposal.TaskDrafts[0]
	if draft.Cell != "production_cell_draft" {
		t.Fatalf("task draft cell = %q, want production_cell_draft", draft.Cell)
	}
	if draft.ImplementationStarted || draft.WorkMutationStatus != "none" {
		t.Fatalf("task draft start/mutation flags = %#v; want no start/no mutation", draft)
	}
	if strings.Join(draft.SourceIssueRefs, ",") != "transpara-ai/work#61" {
		t.Fatalf("task source refs = %#v", draft.SourceIssueRefs)
	}
	if !containsString(draft.Assumptions, "transpara-ai/work#61: issue body is caller-supplied scanner evidence") {
		t.Fatalf("assumptions = %#v", draft.Assumptions)
	}
	if !containsString(draft.Ambiguities, "transpara-ai/work#61: authority-sensitive issues still require separate scope") {
		t.Fatalf("ambiguities = %#v", draft.Ambiguities)
	}
	if !containsString(draft.RiskNotes, "transpara-ai/work#61: issue records are source intent, not authority") {
		t.Fatalf("risk notes = %#v", draft.RiskNotes)
	}
	if len(proposal.ProofOfWorkPacket.IssueSourceRecords) != 1 || proposal.ProofOfWorkPacket.IssueSourceRecords[0].Number != 61 {
		t.Fatalf("proof issue source records = %#v", proposal.ProofOfWorkPacket.IssueSourceRecords)
	}
	if strings.Join(proposal.ProofOfWorkPacket.SourceIssueRefs, ",") != "transpara-ai/work#61" {
		t.Fatalf("proof source refs = %#v", proposal.ProofOfWorkPacket.SourceIssueRefs)
	}
	assertUnavailable(t, proposal.ProofOfWorkPacket.RuntimeInvocation, "runtime")
	assertUnavailable(t, proposal.ProofOfWorkPacket.NativeEventGraphWrite, "EventGraph")
}

func TestBuildFactoryOrderDevelopmentProposalOrdersProofIssueRefs(t *testing.T) {
	opts := validFactoryOrderDevelopmentProposalOptions()
	opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{
		{Repo: "transpara-ai/work", Number: 61, Title: "Requirements and task-draft derivation from issue records"},
		{Repo: "transpara-ai/work", Number: 62, Title: "Proof-of-work packet linked to issue source records"},
	}

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}

	want := "transpara-ai/work#61,transpara-ai/work#62"
	if strings.Join(proposal.ProofOfWorkPacket.SourceIssueRefs, ",") != want {
		t.Fatalf("proof source refs = %#v, want %s", proposal.ProofOfWorkPacket.SourceIssueRefs, want)
	}
	if strings.Join(proposal.TaskDrafts[0].SourceIssueRefs, ",") != want {
		t.Fatalf("task source refs = %#v, want %s", proposal.TaskDrafts[0].SourceIssueRefs, want)
	}
}

func TestBuildFactoryOrderDevelopmentProposalRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*work.FactoryOrderDevelopmentProposalOptions)
		wantErr string
	}{
		{
			name: "missing source intent",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.SourceIntentRef = ""
			},
			wantErr: "source_intent_ref is required",
		},
		{
			name: "missing requester",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.Requester = ""
			},
			wantErr: "requester is required",
		},
		{
			name: "missing target head",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.TargetHead = ""
			},
			wantErr: "target_head is required",
		},
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
			name: "bad acceptance criterion prefix",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AcceptanceCriterionID = "acceptance_bad"
			},
			wantErr: "acceptance_criterion_id",
		},
		{
			name: "bad task prefix",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.TaskID = "task_bad"
			},
			wantErr: "task_id",
		},
		{
			name: "non Work target",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.TargetRepo = "transpara-ai/docs"
			},
			wantErr: "target_repo must be transpara-ai/work",
		},
		{
			name: "issue source missing repo",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{{Number: 61, Title: "issue"}}
			},
			wantErr: "issue_source_records[0].repo is required",
		},
		{
			name: "issue source missing number",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{{Repo: "transpara-ai/work", Title: "issue"}}
			},
			wantErr: "issue_source_records[0].number must be positive",
		},
		{
			name: "issue source missing title",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{{Repo: "transpara-ai/work", Number: 61}}
			},
			wantErr: "issue_source_records[0].title is required",
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
			name: "empty changed file path",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].Path = ""
			},
			wantErr: "changed_file_intent[0].path is required",
		},
		{
			name: "empty changed file change type",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].ChangeType = ""
			},
			wantErr: "changed_file_intent[0].change_type is required",
		},
		{
			name: "empty changed file summary",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ChangedFileIntent[0].Summary = ""
			},
			wantErr: "changed_file_intent[0].summary is required",
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
			name: "empty validation plan",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.ValidationPlan = nil
			},
			wantErr: "validation_plan must be non-empty",
		},
		{
			name: "empty authority boundary",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary = nil
			},
			wantErr: "authority_boundary must be non-empty",
		},
		{
			name: "authority boundary missing action",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary[0].Action = ""
			},
			wantErr: "authority_boundary[0].action is required",
		},
		{
			name: "authority boundary missing status",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary[0].Status = ""
			},
			wantErr: "authority_boundary[0].status is required",
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
			wantErr: "authority_boundary[0].status must be",
		},
		{
			name: "authority boundary claims merge",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary[0].Status = "merged"
			},
			wantErr: "authority_boundary[0].status must be",
		},
		{
			name: "authority boundary claims deploy",
			mutate: func(opts *work.FactoryOrderDevelopmentProposalOptions) {
				opts.AuthorityBoundary[0].Status = "deployed"
			},
			wantErr: "authority_boundary[0].status must be",
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
	opts.IssueSourceRecords = []work.FactoryOrderProposalIssueSourceRecord{
		{
			Repo:               "transpara-ai/work",
			Number:             61,
			Title:              "Requirements and task-draft derivation from issue records",
			AcceptanceCriteria: []string{"source issue refs are preserved"},
			Assumptions:        []string{"issue body is caller-supplied scanner evidence"},
			Ambiguities:        []string{"authority-sensitive issues still require separate scope"},
			RiskNotes:          []string{"issue records are source intent, not authority"},
			Labels:             []string{"cc:intake"},
			SourceRefs:         []string{"github:transpara-ai/work#61"},
		},
	}

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}
	opts.ChangedFileIntent[0].Path = "mutated.go"
	opts.ValidationPlan[0] = "mutated"
	opts.AuthorityBoundary[0].Action = "mutated"
	opts.IssueSourceRecords[0].Repo = "mutated"
	opts.IssueSourceRecords[0].Assumptions[0] = "mutated"
	opts.IssueSourceRecords[0].Labels[0] = "mutated"

	if proposal.ChangedFileIntent[0].Path == "mutated.go" || proposal.ProposalArtifact.ChangedFileIntent[0].Path == "mutated.go" || proposal.ProofOfWorkPacket.ChangedFiles[0].Path == "mutated.go" {
		t.Fatalf("proposal retained caller changed-file slice alias: %#v", proposal)
	}
	if proposal.ValidationPlan[0] == "mutated" || proposal.ProofOfWorkPacket.Validation.Summary == "mutated" {
		t.Fatalf("proposal retained caller validation slice alias: %#v", proposal.ValidationPlan)
	}
	if proposal.AuthorityBoundary[0].Action == "mutated" || proposal.ProofOfWorkPacket.AuthorityBoundary[0].Action == "mutated" {
		t.Fatalf("proposal retained caller authority boundary alias: %#v", proposal.AuthorityBoundary)
	}
	if proposal.IssueSourceRecords[0].Repo == "mutated" || proposal.TaskDrafts[0].Assumptions[0] == "mutated" || proposal.ProofOfWorkPacket.IssueSourceRecords[0].Labels[0] == "mutated" {
		t.Fatalf("proposal retained caller issue source alias: %#v", proposal.IssueSourceRecords)
	}
}

func TestBuildFactoryOrderDevelopmentProposalCanonicalizesBoundaryStatus(t *testing.T) {
	opts := validFactoryOrderDevelopmentProposalOptions()
	opts.AuthorityBoundary[0].Status = "Blocked"

	proposal, err := work.BuildFactoryOrderDevelopmentProposal(opts)
	if err != nil {
		t.Fatalf("BuildFactoryOrderDevelopmentProposal: %v", err)
	}
	if proposal.AuthorityBoundary[0].Status != "blocked" || proposal.ProofOfWorkPacket.AuthorityBoundary[0].Status != "blocked" {
		t.Fatalf("authority boundary status = %#v / %#v, want blocked", proposal.AuthorityBoundary[0], proposal.ProofOfWorkPacket.AuthorityBoundary[0])
	}
}

func validFactoryOrderDevelopmentProposalOptions() work.FactoryOrderDevelopmentProposalOptions {
	return work.FactoryOrderDevelopmentProposalOptions{
		// Event 7 implements the proposal accepted by Event 6, so these fixture
		// IDs intentionally preserve the Event 6 FactoryOrder linkage.
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
