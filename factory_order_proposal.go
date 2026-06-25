package work

import (
	"fmt"
	"strings"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
)

const (
	FactoryOrderProposalTargetRepo = "transpara-ai/work"

	factoryOrderProposalStatusProposalOnly  = "proposal_only"
	factoryOrderProposalStatusUnavailable   = "unavailable"
	factoryOrderProposalStatusNotAuthorized = "not_authorized"
)

// FactoryOrderDevelopmentProposalOptions is the pure input contract for a
// FactoryOrder-linked development proposal evidence packet.
type FactoryOrderDevelopmentProposalOptions struct {
	SourceIntentRef       string
	Requester             string
	TargetRepo            string
	TargetHead            string
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	IssueSourceRecords    []FactoryOrderProposalIssueSourceRecord
	ChangedFileIntent     []FactoryOrderChangedFileIntent
	ValidationPlan        []string
	AuthorityBoundary     []FactoryOrderProtectedActionBoundary

	// Negative-test seams. Supplying any of these means the input is no longer a
	// pure proposal evidence request.
	RuntimeInvocationID      string
	ExecutionReceiptID       string
	NativeEventGraphWriteRef string
	ProtectedActionClaims    []FactoryOrderProtectedActionClaim
}

// FactoryOrderChangedFileIntent records proposed file-level intent without
// carrying an applied patch.
type FactoryOrderChangedFileIntent struct {
	Repo         string `json:"repo"`
	Path         string `json:"path"`
	ChangeType   string `json:"change_type"`
	Summary      string `json:"summary"`
	ProposedOnly bool   `json:"proposed_only"`
	Applied      bool   `json:"applied"`
}

// FactoryOrderProtectedActionBoundary records an action that remains outside
// the proposal builder until separate authority exists.
type FactoryOrderProtectedActionBoundary struct {
	Action            string `json:"action"`
	Status            string `json:"status"`
	RequiredAuthority string `json:"required_authority,omitempty"`
	Summary           string `json:"summary,omitempty"`
}

// FactoryOrderProtectedActionClaim is rejected by the builder because a
// proposal packet cannot claim a protected side effect completed.
type FactoryOrderProtectedActionClaim struct {
	Action  string `json:"action"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

// FactoryOrderProposalIssueSourceRecord is caller-supplied GitHub issue source
// evidence. The builder normalizes it into proposal evidence; it never fetches
// GitHub itself.
type FactoryOrderProposalIssueSourceRecord struct {
	Repo               string   `json:"repo"`
	Number             int      `json:"number"`
	URL                string   `json:"url,omitempty"`
	Title              string   `json:"title"`
	Goal               string   `json:"goal,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Assumptions        []string `json:"assumptions,omitempty"`
	Ambiguities        []string `json:"ambiguities,omitempty"`
	RiskNotes          []string `json:"risk_notes,omitempty"`
	Labels             []string `json:"labels,omitempty"`
	SourceRefs         []string `json:"source_refs,omitempty"`
}

// FactoryOrderDevelopmentProposal is the structured proposal evidence returned
// by BuildFactoryOrderDevelopmentProposal.
type FactoryOrderDevelopmentProposal struct {
	FactoryOrder       FactoryOrderProposalSource              `json:"factory_order"`
	IssueSourceRecords []FactoryOrderProposalIssueSourceRecord `json:"issue_source_records,omitempty"`
	Requirements       []FactoryOrderProposalRequirement       `json:"requirements"`
	AcceptanceCriteria []FactoryOrderProposalAcceptance        `json:"acceptance_criteria"`
	TaskDrafts         []FactoryOrderProposalTaskDraft         `json:"task_drafts"`
	ChangedFileIntent  []FactoryOrderChangedFileIntent         `json:"changed_file_intent"`
	ProposalArtifact   FactoryOrderProposalArtifact            `json:"proposal_artifact"`
	ValidationPlan     []string                                `json:"validation_plan"`
	ValidationResult   FactoryOrderProposalAvailability        `json:"validation_result"`
	ProofOfWorkPacket  FactoryOrderProposalProofOfWork         `json:"proof_of_work_packet"`
	AuditReport        FactoryOrderProposalAuditReport         `json:"audit_report"`
	AuthorityBoundary  []FactoryOrderProtectedActionBoundary   `json:"authority_boundary"`
}

type FactoryOrderProposalSource struct {
	ID              string `json:"id"`
	SourceIntentRef string `json:"source_intent_ref"`
	Requester       string `json:"requester"`
	TargetRepo      string `json:"target_repo"`
	TargetHead      string `json:"target_head"`
}

type FactoryOrderProposalRequirement struct {
	ID             string `json:"id"`
	FactoryOrderID string `json:"factory_order_id"`
	Source         string `json:"source"`
	Text           string `json:"text"`
}

type FactoryOrderProposalAcceptance struct {
	ID                   string `json:"id"`
	RequirementID        string `json:"requirement_id"`
	VerificationMethod   string `json:"verification_method"`
	RequiredEvidenceType string `json:"required_evidence_type"`
	Text                 string `json:"text"`
}

type FactoryOrderProposalTaskDraft struct {
	ID                     string   `json:"id"`
	FactoryOrderID         string   `json:"factory_order_id"`
	RequirementIDs         []string `json:"requirement_ids"`
	AcceptanceCriterionIDs []string `json:"acceptance_criterion_ids"`
	Cell                   string   `json:"cell"`
	TargetRepo             string   `json:"target_repo"`
	TargetHead             string   `json:"target_head"`
	SourceIssueRefs        []string `json:"source_issue_refs,omitempty"`
	Assumptions            []string `json:"assumptions,omitempty"`
	Ambiguities            []string `json:"ambiguities,omitempty"`
	RiskNotes              []string `json:"risk_notes,omitempty"`
	ImplementationStarted  bool     `json:"implementation_started"`
	WorkMutationStatus     string   `json:"work_mutation_status"`
}

type FactoryOrderProposalArtifact struct {
	ID                string                          `json:"id"`
	TargetRepo        string                          `json:"target_repo"`
	TargetHead        string                          `json:"target_head"`
	ProposedOnly      bool                            `json:"proposed_only"`
	Applied           bool                            `json:"applied"`
	ChangedFileIntent []FactoryOrderChangedFileIntent `json:"changed_file_intent"`
}

type FactoryOrderProposalAvailability struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Ref    string `json:"ref,omitempty"`
}

type FactoryOrderProposalProofOfWork struct {
	WorkItem              FactoryOrderProposalProofItem           `json:"work_item"`
	ChangedFiles          []FactoryOrderChangedFileIntent         `json:"changed_files"`
	Validation            FactoryOrderProposalProofItem           `json:"validation"`
	Branch                FactoryOrderProposalAvailability        `json:"branch"`
	PullRequest           FactoryOrderProposalAvailability        `json:"pull_request"`
	CI                    FactoryOrderProposalAvailability        `json:"ci"`
	RuntimeInvocation     FactoryOrderProposalAvailability        `json:"runtime_invocation"`
	ExecutionReceipt      FactoryOrderProposalAvailability        `json:"execution_receipt"`
	NativeEventGraphWrite FactoryOrderProposalAvailability        `json:"native_eventgraph_write"`
	IssueSourceRecords    []FactoryOrderProposalIssueSourceRecord `json:"issue_source_records,omitempty"`
	AuthorityBoundary     []FactoryOrderProtectedActionBoundary   `json:"authority_boundary"`
	TraceGap              []string                                `json:"trace_gap"`
	ResidualRisks         []string                                `json:"residual_risks"`
}

type FactoryOrderProposalProofItem struct {
	Label   string   `json:"label"`
	Status  string   `json:"status"`
	Summary string   `json:"summary"`
	Refs    []string `json:"refs,omitempty"`
}

type FactoryOrderProposalAuditReport struct {
	Status          string   `json:"status"`
	Recommendation  string   `json:"recommendation"`
	Summary         string   `json:"summary"`
	ResidualRisks   []string `json:"residual_risks"`
	ForbiddenClaims []string `json:"forbidden_claims"`
}

// BuildFactoryOrderDevelopmentProposal returns structured proposal evidence
// only. It performs no file writes, command execution, GitHub calls, EventGraph
// writes, RuntimeBroker calls, or TaskStore mutation.
func BuildFactoryOrderDevelopmentProposal(opts FactoryOrderDevelopmentProposalOptions) (FactoryOrderDevelopmentProposal, error) {
	normalized, err := normalizeFactoryOrderDevelopmentProposalOptions(opts)
	if err != nil {
		return FactoryOrderDevelopmentProposal{}, err
	}

	changedFiles := cloneChangedFileIntent(normalized.ChangedFileIntent)
	validationPlan := cloneStrings(normalized.ValidationPlan)
	authorityBoundary := cloneAuthorityBoundary(normalized.AuthorityBoundary)
	issueRecords := cloneIssueSourceRecords(normalized.IssueSourceRecords)

	proposal := FactoryOrderDevelopmentProposal{
		FactoryOrder: FactoryOrderProposalSource{
			ID:              normalized.FactoryOrderID,
			SourceIntentRef: normalized.SourceIntentRef,
			Requester:       normalized.Requester,
			TargetRepo:      normalized.TargetRepo,
			TargetHead:      normalized.TargetHead,
		},
		IssueSourceRecords: issueRecords,
		Requirements: []FactoryOrderProposalRequirement{
			{
				ID:             normalized.RequirementID,
				FactoryOrderID: normalized.FactoryOrderID,
				Source:         factoryOrderRequirementSource(issueRecords),
				Text:           factoryOrderRequirementText(issueRecords),
			},
		},
		AcceptanceCriteria: []FactoryOrderProposalAcceptance{
			{
				ID:                   normalized.AcceptanceCriterionID,
				RequirementID:        normalized.RequirementID,
				VerificationMethod:   "test",
				RequiredEvidenceType: factoryOrderRequiredEvidenceType(issueRecords),
				Text:                 factoryOrderAcceptanceText(issueRecords),
			},
		},
		TaskDrafts: []FactoryOrderProposalTaskDraft{
			{
				ID:                     normalized.TaskID,
				FactoryOrderID:         normalized.FactoryOrderID,
				RequirementIDs:         []string{normalized.RequirementID},
				AcceptanceCriterionIDs: []string{normalized.AcceptanceCriterionID},
				Cell:                   factoryOrderTaskDraftCell(issueRecords),
				TargetRepo:             normalized.TargetRepo,
				TargetHead:             normalized.TargetHead,
				SourceIssueRefs:        issueSourceRefs(issueRecords),
				Assumptions:            issueNotes(issueRecords, func(record FactoryOrderProposalIssueSourceRecord) []string { return record.Assumptions }),
				Ambiguities:            issueNotes(issueRecords, func(record FactoryOrderProposalIssueSourceRecord) []string { return record.Ambiguities }),
				RiskNotes:              issueNotes(issueRecords, func(record FactoryOrderProposalIssueSourceRecord) []string { return record.RiskNotes }),
				ImplementationStarted:  false,
				WorkMutationStatus:     "none",
			},
		},
		ChangedFileIntent: changedFiles,
		ProposalArtifact: FactoryOrderProposalArtifact{
			ID:                "artifact_" + strings.TrimPrefix(normalized.TaskID, "tsk_") + "_proposal",
			TargetRepo:        normalized.TargetRepo,
			TargetHead:        normalized.TargetHead,
			ProposedOnly:      true,
			Applied:           false,
			ChangedFileIntent: cloneChangedFileIntent(changedFiles),
		},
		ValidationPlan: validationPlan,
		ValidationResult: FactoryOrderProposalAvailability{
			Status: factoryOrderProposalStatusUnavailable,
			Reason: "validation is unavailable until an authorized implementation patch exists",
		},
		AuthorityBoundary: authorityBoundary,
	}
	proposal.ProofOfWorkPacket = FactoryOrderProposalProofOfWork{
		WorkItem: FactoryOrderProposalProofItem{
			Label:   "FactoryOrder development proposal",
			Status:  factoryOrderProposalStatusProposalOnly,
			Summary: "Proposal evidence built without applying a patch or executing protected actions.",
			Refs:    []string{normalized.FactoryOrderID, normalized.RequirementID, normalized.AcceptanceCriterionID, normalized.TaskID},
		},
		ChangedFiles:          cloneChangedFileIntent(changedFiles),
		Validation:            FactoryOrderProposalProofItem{Label: "validation", Status: factoryOrderProposalStatusUnavailable, Summary: strings.Join(validationPlan, "; ")},
		Branch:                unavailable("branch evidence is outside this pure builder"),
		PullRequest:           unavailable("pull request evidence is outside this pure builder"),
		CI:                    unavailable("CI evidence is outside this pure builder"),
		RuntimeInvocation:     unavailable("RuntimeBroker execution is not authorized"),
		ExecutionReceipt:      unavailable("protected execution did not occur"),
		NativeEventGraphWrite: unavailable("native EventGraph truth requires separate authority"),
		IssueSourceRecords:    cloneIssueSourceRecords(issueRecords),
		AuthorityBoundary:     cloneAuthorityBoundary(authorityBoundary),
		TraceGap:              []string{"native EventGraph truth remains unavailable"},
		ResidualRisks:         []string{"R-001 unresolved", "R-002 unresolved", "R-003 unresolved"},
	}
	proposal.AuditReport = FactoryOrderProposalAuditReport{
		Status:         "defer",
		Recommendation: "defer_protected_actions_until_separate_authority",
		Summary:        "Accept proposal evidence only; do not treat it as runtime, EventGraph, production, autonomy, value, v3.9, or residual-risk closure evidence.",
		ResidualRisks:  []string{"R-001 unresolved", "R-002 unresolved", "R-003 unresolved"},
		ForbiddenClaims: []string{
			"production readiness",
			"go-live",
			"Level 1 achievement",
			"autonomy increase",
			"value allocation",
			"v3.9 mutation/archive",
			"R-001/R-002/R-003 closure",
		},
	}
	return proposal, nil
}

func normalizeFactoryOrderDevelopmentProposalOptions(opts FactoryOrderDevelopmentProposalOptions) (FactoryOrderDevelopmentProposalOptions, error) {
	normalized := FactoryOrderDevelopmentProposalOptions{
		SourceIntentRef:          strings.TrimSpace(opts.SourceIntentRef),
		Requester:                strings.TrimSpace(opts.Requester),
		TargetRepo:               strings.TrimSpace(opts.TargetRepo),
		TargetHead:               strings.TrimSpace(opts.TargetHead),
		FactoryOrderID:           strings.TrimSpace(opts.FactoryOrderID),
		RequirementID:            strings.TrimSpace(opts.RequirementID),
		AcceptanceCriterionID:    strings.TrimSpace(opts.AcceptanceCriterionID),
		TaskID:                   strings.TrimSpace(opts.TaskID),
		IssueSourceRecords:       cloneIssueSourceRecords(opts.IssueSourceRecords),
		ChangedFileIntent:        cloneChangedFileIntent(opts.ChangedFileIntent),
		ValidationPlan:           cloneStrings(opts.ValidationPlan),
		AuthorityBoundary:        cloneAuthorityBoundary(opts.AuthorityBoundary),
		RuntimeInvocationID:      strings.TrimSpace(opts.RuntimeInvocationID),
		ExecutionReceiptID:       strings.TrimSpace(opts.ExecutionReceiptID),
		NativeEventGraphWriteRef: strings.TrimSpace(opts.NativeEventGraphWriteRef),
		ProtectedActionClaims:    cloneProtectedActionClaims(opts.ProtectedActionClaims),
	}
	if normalized.SourceIntentRef == "" {
		return normalized, fmt.Errorf("source_intent_ref is required")
	}
	if normalized.Requester == "" {
		return normalized, fmt.Errorf("requester is required")
	}
	if normalized.TargetRepo != FactoryOrderProposalTargetRepo {
		return normalized, fmt.Errorf("target_repo must be %s", FactoryOrderProposalTargetRepo)
	}
	if normalized.TargetHead == "" {
		return normalized, fmt.Errorf("target_head is required")
	}
	if err := validateV39Reference(v39.TypeFactoryOrder, "factory_order_id", normalized.FactoryOrderID); err != nil {
		return normalized, err
	}
	if err := validateV39Reference(v39.TypeRequirement, "requirement_id", normalized.RequirementID); err != nil {
		return normalized, err
	}
	if err := validateV39Reference(v39.TypeAcceptanceCriterion, "acceptance_criterion_id", normalized.AcceptanceCriterionID); err != nil {
		return normalized, err
	}
	if err := validateV39Reference(v39.TypeTask, "task_id", normalized.TaskID); err != nil {
		return normalized, err
	}
	for i, record := range normalized.IssueSourceRecords {
		record.Repo = strings.TrimSpace(record.Repo)
		record.URL = strings.TrimSpace(record.URL)
		record.Title = strings.TrimSpace(record.Title)
		record.Goal = strings.TrimSpace(record.Goal)
		record.AcceptanceCriteria = cloneStrings(record.AcceptanceCriteria)
		record.Assumptions = cloneStrings(record.Assumptions)
		record.Ambiguities = cloneStrings(record.Ambiguities)
		record.RiskNotes = cloneStrings(record.RiskNotes)
		record.Labels = cloneStrings(record.Labels)
		record.SourceRefs = cloneStrings(record.SourceRefs)
		if record.Repo == "" {
			return normalized, fmt.Errorf("issue_source_records[%d].repo is required", i)
		}
		if record.Number <= 0 {
			return normalized, fmt.Errorf("issue_source_records[%d].number must be positive", i)
		}
		if record.Title == "" {
			return normalized, fmt.Errorf("issue_source_records[%d].title is required", i)
		}
		if record.Goal == "" {
			record.Goal = record.Title
		}
		if len(record.SourceRefs) == 0 {
			record.SourceRefs = []string{issueSourceRef(record)}
		}
		normalized.IssueSourceRecords[i] = record
	}
	if len(normalized.ChangedFileIntent) == 0 {
		return normalized, fmt.Errorf("changed_file_intent must be non-empty")
	}
	for i, intent := range normalized.ChangedFileIntent {
		intent.Repo = strings.TrimSpace(intent.Repo)
		intent.Path = strings.TrimSpace(intent.Path)
		intent.ChangeType = strings.TrimSpace(intent.ChangeType)
		intent.Summary = strings.TrimSpace(intent.Summary)
		if intent.Repo != FactoryOrderProposalTargetRepo {
			return normalized, fmt.Errorf("changed_file_intent[%d].repo must be %s", i, FactoryOrderProposalTargetRepo)
		}
		if intent.Path == "" {
			return normalized, fmt.Errorf("changed_file_intent[%d].path is required", i)
		}
		if intent.ChangeType == "" {
			return normalized, fmt.Errorf("changed_file_intent[%d].change_type is required", i)
		}
		if intent.Summary == "" {
			return normalized, fmt.Errorf("changed_file_intent[%d].summary is required", i)
		}
		if !intent.ProposedOnly {
			return normalized, fmt.Errorf("changed_file_intent[%d].proposed_only must be true", i)
		}
		if intent.Applied {
			return normalized, fmt.Errorf("changed_file_intent[%d].applied must be false", i)
		}
		normalized.ChangedFileIntent[i] = intent
	}
	if len(normalized.ValidationPlan) == 0 {
		return normalized, fmt.Errorf("validation_plan must be non-empty")
	}
	if len(normalized.AuthorityBoundary) == 0 {
		return normalized, fmt.Errorf("authority_boundary must be non-empty")
	}
	for i, boundary := range normalized.AuthorityBoundary {
		boundary.Action = strings.TrimSpace(boundary.Action)
		boundary.Status = strings.ToLower(strings.TrimSpace(boundary.Status))
		boundary.RequiredAuthority = strings.TrimSpace(boundary.RequiredAuthority)
		boundary.Summary = strings.TrimSpace(boundary.Summary)
		if boundary.Action == "" {
			return normalized, fmt.Errorf("authority_boundary[%d].action is required", i)
		}
		if boundary.Status == "" {
			return normalized, fmt.Errorf("authority_boundary[%d].status is required", i)
		}
		if !allowedProtectedActionBoundaryStatus(boundary.Status) {
			return normalized, fmt.Errorf("authority_boundary[%d].status must be not_authorized, deferred, pending, blocked, unavailable, or requires_authority", i)
		}
		normalized.AuthorityBoundary[i] = boundary
	}
	switch {
	case normalized.RuntimeInvocationID != "":
		return normalized, fmt.Errorf("runtime_invocation_id is not allowed in proposal evidence")
	case normalized.ExecutionReceiptID != "":
		return normalized, fmt.Errorf("execution_receipt_id is not allowed in proposal evidence")
	case normalized.NativeEventGraphWriteRef != "":
		return normalized, fmt.Errorf("native_eventgraph_write_ref is not allowed in proposal evidence")
	case len(normalized.ProtectedActionClaims) > 0:
		return normalized, fmt.Errorf("protected_action_claims are not allowed in proposal evidence")
	}
	return normalized, nil
}

func allowedProtectedActionBoundaryStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "blocked", "deferred", "not_authorized", "pending", "requires_authority", "unavailable":
		return true
	default:
		return false
	}
}

func unavailable(reason string) FactoryOrderProposalAvailability {
	return FactoryOrderProposalAvailability{Status: factoryOrderProposalStatusUnavailable, Reason: reason}
}

func factoryOrderRequirementSource(records []FactoryOrderProposalIssueSourceRecord) string {
	if len(records) == 0 {
		return "explicit"
	}
	return "github_issue"
}

func factoryOrderRequirementText(records []FactoryOrderProposalIssueSourceRecord) string {
	if len(records) == 0 {
		return "Construct a pure FactoryOrder-linked development proposal evidence packet."
	}
	goals := make([]string, 0, len(records))
	for _, record := range records {
		goals = append(goals, issueSourceRef(record)+": "+record.Goal)
	}
	return "Derive Work proposal requirements from GitHub issue source records without starting implementation: " + strings.Join(goals, "; ")
}

func factoryOrderRequiredEvidenceType(records []FactoryOrderProposalIssueSourceRecord) string {
	if len(records) == 0 {
		return "factory_order_development_proposal_evidence"
	}
	return "github_issue_derived_factory_order_proposal_evidence"
}

func factoryOrderTaskDraftCell(records []FactoryOrderProposalIssueSourceRecord) string {
	if len(records) == 0 {
		return "implementation"
	}
	return "production_cell_draft"
}

func factoryOrderAcceptanceText(records []FactoryOrderProposalIssueSourceRecord) string {
	if len(records) == 0 {
		return "Proposal evidence preserves linkage, validation plan, authority boundary, proof-of-work, and AuditReport without protected side effects."
	}
	criteria := issueNotes(records, func(record FactoryOrderProposalIssueSourceRecord) []string {
		return record.AcceptanceCriteria
	})
	if len(criteria) == 0 {
		return "Issue-derived proposal evidence preserves source issue refs, assumptions, ambiguities, risk notes, validation plan, authority boundary, proof-of-work, and AuditReport without starting implementation or mutating Work state."
	}
	return "Issue-derived proposal evidence preserves source issue refs and these acceptance criteria without starting implementation or mutating Work state: " + strings.Join(criteria, "; ")
}

func issueSourceRefs(records []FactoryOrderProposalIssueSourceRecord) []string {
	refs := make([]string, 0, len(records))
	for _, record := range records {
		refs = append(refs, issueSourceRef(record))
	}
	return refs
}

func issueSourceRef(record FactoryOrderProposalIssueSourceRecord) string {
	return fmt.Sprintf("%s#%d", strings.TrimSpace(record.Repo), record.Number)
}

func issueNotes(records []FactoryOrderProposalIssueSourceRecord, selectNotes func(FactoryOrderProposalIssueSourceRecord) []string) []string {
	notes := make([]string, 0)
	for _, record := range records {
		prefix := issueSourceRef(record)
		for _, note := range selectNotes(record) {
			note = strings.TrimSpace(note)
			if note == "" {
				continue
			}
			notes = append(notes, prefix+": "+note)
		}
	}
	return notes
}

func cloneChangedFileIntent(values []FactoryOrderChangedFileIntent) []FactoryOrderChangedFileIntent {
	if len(values) == 0 {
		return nil
	}
	out := make([]FactoryOrderChangedFileIntent, len(values))
	copy(out, values)
	return out
}

func cloneIssueSourceRecords(values []FactoryOrderProposalIssueSourceRecord) []FactoryOrderProposalIssueSourceRecord {
	if len(values) == 0 {
		return nil
	}
	out := make([]FactoryOrderProposalIssueSourceRecord, len(values))
	copy(out, values)
	for i := range out {
		out[i].AcceptanceCriteria = cloneStrings(out[i].AcceptanceCriteria)
		out[i].Assumptions = cloneStrings(out[i].Assumptions)
		out[i].Ambiguities = cloneStrings(out[i].Ambiguities)
		out[i].RiskNotes = cloneStrings(out[i].RiskNotes)
		out[i].Labels = cloneStrings(out[i].Labels)
		out[i].SourceRefs = cloneStrings(out[i].SourceRefs)
	}
	return out
}

func cloneAuthorityBoundary(values []FactoryOrderProtectedActionBoundary) []FactoryOrderProtectedActionBoundary {
	if len(values) == 0 {
		return nil
	}
	out := make([]FactoryOrderProtectedActionBoundary, len(values))
	copy(out, values)
	return out
}

func cloneProtectedActionClaims(values []FactoryOrderProtectedActionClaim) []FactoryOrderProtectedActionClaim {
	if len(values) == 0 {
		return nil
	}
	out := make([]FactoryOrderProtectedActionClaim, len(values))
	copy(out, values)
	return out
}
