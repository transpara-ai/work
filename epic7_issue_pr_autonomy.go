package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Epic7IssueToPRLocalProposalEvidence Epic7IssueToPRMode = "local_proposal_evidence"
)

const (
	Epic7ActionPullRequestPropose    Epic7ProtectedAction = "pull_request.propose"
	Epic7ActionPullRequestCreate     Epic7ProtectedAction = "pull_request.create"
	Epic7ActionBranchPush            Epic7ProtectedAction = "branch.push"
	Epic7ActionDefaultBranchPush     Epic7ProtectedAction = "default_branch.push"
	Epic7ActionPullRequestMerge      Epic7ProtectedAction = "pull_request.merge"
	Epic7ActionProductionDeploy      Epic7ProtectedAction = "production.deploy"
	Epic7ActionProtectedExecutionRun Epic7ProtectedAction = "protected_execution.run"
	Epic7ActionCapabilityActivate    Epic7ProtectedAction = "capability.activate"
)

const (
	Epic11DocsDraftPRLiveMutationMode Epic11DocsDraftPRMode = "docs_draft_pr_live_mutation"
)

const (
	Epic11ActionPullRequestCreate = "pull_request.create"
)

const (
	Epic11TargetRepository = "transpara-ai/docs"
	Epic11TargetBaseRef    = "main"
	Epic11HeadRefPrefix    = "codex/"
	Epic11PolicyBundleID   = "df-v3.9.20-docs-draft-pr-create-only"
	Epic11PolicyAdapterID  = "work-live-github-pr-policy-adapter"
)

const (
	epic7FixtureActorID      = "act_epic7_issue_pr_proposer"
	epic7FixtureHumanActorID = "act_epic7_human_reviewer"
	epic7KnowledgeSourceRef  = "knowledge:dark-factory/v3.9/implementation/epics/epic-07-gate-h-issue-to-pr-autonomy-trials/01-work-issue-to-pr-autonomy-implementation-authorization-v3.9.md"
	epic7DocsPRRef           = "transpara-ai/docs#87"
	epic7DocsMergeSHA        = "b2f09a3b70ccfac124d3ab8e5e0bb21523860c29"
	epic7DocsReviewedHead    = "08c413f754c48ef647f5972d100452788206ee63"

	epic11FixtureActorID     = "act_epic11_docs_draft_pr_creator"
	epic11HumanActorID       = "act_epic11_human_authorizer"
	epic11KnowledgeSourceRef = "knowledge:dark-factory/v3.9/implementation/epics/epic-11-docs-draft-pr-live-mutation/01-work-docs-draft-pr-live-mutation-implementation-authorization-v3.9.md"
	epic11DocsPRRef          = "transpara-ai/docs#95"
	epic11DocsMergeSHA       = "b21e2eca5ce547eebef83a1a392f5ca790c3e44d"
	epic11DocsReviewedHead   = "b4f9844ecad41a8dc1298e3ac19df3a4e7ac9071"

	epic11AuthorityReservationArtifactLabel = "epic11_docs_draft_pr_authority_reservation"
	epic11ExecutionReceiptArtifactLabel     = "epic11_docs_draft_pr_execution_receipt"
)

var epic11AuthorityReservationMu sync.Mutex

// Epic7IssueToPRMode selects the authorized Gate H fixture mode.
type Epic7IssueToPRMode string

// Epic7ProtectedAction names the protected-action boundary exercised by Gate H.
type Epic7ProtectedAction string

// Epic11DocsDraftPRMode selects the authorized Epic 11 live-mutation seam.
type Epic11DocsDraftPRMode string

// Epic7IssueToPROptions keeps the fixture local, proposed-only, and caller-bounded.
type Epic7IssueToPROptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic7IssueToPRMode

	// Negative-test seams. These never execute protected actions; they only
	// inject forbidden evidence so Gate H can prove rejection behavior.
	OmitIssueFixture               string
	OmitProposalPacket             string
	AppliedPatchTrial              string
	CompletedForbiddenActions      []Epic7ProtectedAction
	RecordExecutionReceipt         bool
	MissingMultiRepoAuthority      bool
	MissingRepairEvidence          bool
	MissingRepairTestUpdateIntent  bool
	MissingSelfImprovementReview   bool
	MissingSelfImprovementRollback bool
	OmitProtectedAction            Epic7ProtectedAction
}

// Epic11PullRequestCreator is the only live side-effect interface for Epic 11.
// Production callers provide a GitHub-backed implementation; tests provide a
// fake. Work validates authority and policy evidence before this interface is called.
// Epic11RemoteHeadState is the remote state of the head ref observed at
// preflight time. It binds the approved head SHA and the dark-factory/ diff
// scope to the actual remote before the irreversible draft-PR creation.
type Epic11RemoteHeadState struct {
	HeadSHA      string   `json:"head_sha"`
	ChangedFiles []string `json:"changed_files"`
}

type Epic11PullRequestCreator interface {
	// PreflightHead reports the remote head SHA and the base...head changed-file
	// list, so the caller can verify them against the approved target before any
	// mutation. It performs no mutation.
	PreflightHead(context.Context, Epic11DraftPullRequestMutation) (Epic11RemoteHeadState, error)
	CreateDraftPullRequest(context.Context, Epic11DraftPullRequestMutation) (Epic11DraftPullRequestResult, error)
}

// Epic11DocsDraftPROptions keeps the live mutation tightly scoped to docs#95.
type Epic11DocsDraftPROptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic11DocsDraftPRMode

	Client Epic11PullRequestCreator
	Now    time.Time

	Target            Epic11DraftPullRequestTarget
	AuthorityRequest  Epic11AuthorityRequestEvidence
	AuthorityDecision Epic11AuthorityDecisionEvidence
	PolicyDecision    Epic11PolicyDecisionEvidence

	// PriorExecutionReceiptRefs is an optional caller-provided defense-in-depth
	// check. Work also scans durable receipt artifacts before any GitHub call.
	PriorExecutionReceiptRefs []string
}

// Epic7IssueToPRRun is the local evidence packet for the bounded Gate H trials.
type Epic7IssueToPRRun struct {
	Mode                  Epic7IssueToPRMode
	WorkTask              Task
	WorkProjection        TaskProjection
	EventGraph            *v39.InMemoryStore
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	ActorInvocationID     string
	RuntimeEnvelopeID     string
	RuntimeResultID       string
	CapabilityArtifactID  string
	KnowledgeReferenceID  string
	TestCaseID            string
	TestRunID             string
	GateResultID          string
	FailureID             string
	ReleaseCandidateID    string
	CertificationID       string
	RejectionID           string
	AuditReportID         string
	TraceCompleteness     v39.TraceCompletenessGateResult
	CapabilityUsagePath   v39.RequiredPath
	KnowledgePath         v39.RequiredPath
	GateHValidation       Epic7GateHValidation
	Certification         *v39.Certification
	Rejection             *v39.Rejection
	AuditReport           *v39.AuditReport
	Projection            Epic7IssueToPRProjection
	LocalArtifacts        Epic7LocalArtifacts
}

// Epic11DocsDraftPRRun is the evidence packet for a guarded docs draft PR creation.
type Epic11DocsDraftPRRun struct {
	Mode              Epic11DocsDraftPRMode
	WorkTask          Task
	WorkProjection    TaskProjection
	EventGraph        *v39.InMemoryStore
	Target            Epic11DraftPullRequestTarget
	PolicyBundleID    string
	PolicyBundleHash  string
	AuthorityRequest  v39.AuthorityRequest
	AuthorityDecision v39.AuthorityDecision
	PolicyDecision    v39.PolicyEngineAdapterDecision
	AuthorityReserve  Epic11AuthorityReservationEvidence
	ExecutionReceipt  v39.ExecutionReceipt
	ReceiptEvidence   Epic11ExecutionReceiptEvidence
	MutationResult    Epic11DraftPullRequestResult
	Projection        Epic11DocsDraftPRProjection
}

type Epic7LocalArtifacts struct {
	IssueDir       string
	ProposalDir    string
	ProofDir       string
	PatchDir       string
	PRBodyDir      string
	BranchPlanDir  string
	ValidationDir  string
	RepairDir      string
	RollbackDir    string
	HumanReviewDir string
}

type Epic11DraftPullRequestTarget struct {
	Repository             string   `json:"repository"`
	BaseRef                string   `json:"base_ref"`
	BaseSHA                string   `json:"base_sha"`
	HeadRef                string   `json:"head_ref"`
	HeadSHA                string   `json:"head_sha"`
	HeadExistsOnOrigin     bool     `json:"head_exists_on_origin"`
	Title                  string   `json:"title"`
	Body                   string   `json:"body"`
	ChangedFiles           []string `json:"changed_files"`
	ValidationEvidenceRefs []string `json:"validation_evidence_refs"`
	Draft                  bool     `json:"draft"`
	MaintainerCanModify    bool     `json:"maintainer_can_modify"`
	RollbackInstructions   string   `json:"rollback_instructions"`
}

type Epic11AuthorityRequestEvidence struct {
	ID                     string    `json:"id"`
	ActorID                string    `json:"actor_id"`
	ActorRole              string    `json:"actor_role"`
	Action                 string    `json:"action"`
	TargetRepository       string    `json:"target_repository"`
	BaseRef                string    `json:"base_ref"`
	BaseSHA                string    `json:"base_sha"`
	HeadRef                string    `json:"head_ref"`
	HeadSHA                string    `json:"head_sha"`
	TitleHash              string    `json:"title_hash"`
	BodyHash               string    `json:"body_hash"`
	ChangedFiles           []string  `json:"changed_files"`
	ValidationEvidenceRefs []string  `json:"validation_evidence_refs"`
	PolicyBundleID         string    `json:"policy_bundle_id"`
	PolicyBundleHash       string    `json:"policy_bundle_hash"`
	RollbackInstructions   string    `json:"rollback_instructions"`
	SingleUseNonce         string    `json:"single_use_nonce"`
	RequestedAt            time.Time `json:"requested_at"`
	ExpiresAt              time.Time `json:"expires_at"`
}

type Epic11AuthorityDecisionEvidence struct {
	ID                     string    `json:"id"`
	AuthorityRequestID     string    `json:"authority_request_id"`
	ActorID                string    `json:"actor_id"`
	ActorRole              string    `json:"actor_role"`
	DeciderActorID         string    `json:"decider_actor_id"`
	DeciderRole            string    `json:"decider_role"`
	Decision               string    `json:"decision"`
	Action                 string    `json:"action"`
	TargetRepository       string    `json:"target_repository"`
	BaseRef                string    `json:"base_ref"`
	BaseSHA                string    `json:"base_sha"`
	HeadRef                string    `json:"head_ref"`
	HeadSHA                string    `json:"head_sha"`
	TitleHash              string    `json:"title_hash"`
	BodyHash               string    `json:"body_hash"`
	ChangedFiles           []string  `json:"changed_files"`
	ValidationEvidenceRefs []string  `json:"validation_evidence_refs"`
	PolicyBundleID         string    `json:"policy_bundle_id"`
	PolicyBundleHash       string    `json:"policy_bundle_hash"`
	RollbackInstructions   string    `json:"rollback_instructions"`
	SingleUseNonce         string    `json:"single_use_nonce"`
	ExpiresAt              time.Time `json:"expires_at"`
}

type Epic11PolicyDecisionEvidence struct {
	DecisionID           string         `json:"decision_id"`
	AdapterID            string         `json:"adapter_id"`
	AdapterVersion       string         `json:"adapter_version"`
	PolicyBundleID       string         `json:"policy_bundle_id"`
	PolicyBundleHash     string         `json:"policy_bundle_hash"`
	ProtectedActionType  string         `json:"protected_action_type"`
	ActorID              string         `json:"actor_id"`
	ResourceRefs         []string       `json:"resource_refs"`
	InputFacts           map[string]any `json:"input_facts"`
	RawDecision          string         `json:"raw_decision"`
	CanonicalDecision    string         `json:"canonical_decision"`
	ReasonCodes          []string       `json:"reason_codes"`
	EvidenceRefs         []string       `json:"evidence_refs"`
	LatencyMS            float64        `json:"latency_ms"`
	AuthorityDecisionRef string         `json:"authority_decision_ref"`
}

type Epic11DraftPullRequestMutation struct {
	Repository          string `json:"repository"`
	BaseRef             string `json:"base_ref"`
	BaseSHA             string `json:"base_sha"`
	HeadRef             string `json:"head_ref"`
	HeadSHA             string `json:"head_sha"`
	Title               string `json:"title"`
	Body                string `json:"body"`
	TitleHash           string `json:"title_hash"`
	BodyHash            string `json:"body_hash"`
	Draft               bool   `json:"draft"`
	MaintainerCanModify bool   `json:"maintainer_can_modify"`
}

type Epic11DraftPullRequestResult struct {
	Repository                   string    `json:"repository"`
	Number                       int       `json:"number"`
	URL                          string    `json:"url"`
	GitHubResponseIDOrEquivalent string    `json:"github_response_id_or_equivalent"`
	BaseRef                      string    `json:"base_ref"`
	BaseSHA                      string    `json:"base_sha"`
	HeadRef                      string    `json:"head_ref"`
	HeadSHA                      string    `json:"head_sha"`
	Draft                        bool      `json:"draft"`
	State                        string    `json:"state"`
	CreatedAt                    time.Time `json:"created_at"`
}

type Epic11ExecutionReceiptEvidence struct {
	ReceiptID                      string    `json:"receipt_id"`
	AuthorityRequestRef            string    `json:"authority_request_ref"`
	AuthorityDecisionRef           string    `json:"authority_decision_ref"`
	PolicyEngineAdapterDecisionRef string    `json:"policy_engine_adapter_decision_ref"`
	SingleUseNonce                 string    `json:"single_use_nonce"`
	ActorID                        string    `json:"actor_id"`
	ActorRole                      string    `json:"actor_role"`
	Action                         string    `json:"action"`
	TargetRepository               string    `json:"target_repository"`
	BaseRef                        string    `json:"base_ref"`
	BaseSHA                        string    `json:"base_sha"`
	HeadRef                        string    `json:"head_ref"`
	HeadSHA                        string    `json:"head_sha"`
	Draft                          bool      `json:"draft"`
	PRNumber                       int       `json:"pr_number"`
	PRURL                          string    `json:"pr_url"`
	TitleHash                      string    `json:"title_hash"`
	BodyHash                       string    `json:"body_hash"`
	GitHubResponseIDOrEquivalent   string    `json:"github_response_id_or_equivalent"`
	Result                         string    `json:"result"`
	Timestamp                      time.Time `json:"timestamp"`
	ValidationEvidenceRefs         []string  `json:"validation_evidence_refs"`
	RollbackInstructions           string    `json:"rollback_instructions"`
}

type Epic11AuthorityReservationEvidence struct {
	ReservationID                  string    `json:"reservation_id"`
	TaskRef                        string    `json:"task_ref"`
	AuthorityRequestRef            string    `json:"authority_request_ref"`
	AuthorityDecisionRef           string    `json:"authority_decision_ref"`
	PolicyEngineAdapterDecisionRef string    `json:"policy_engine_adapter_decision_ref"`
	SingleUseNonce                 string    `json:"single_use_nonce"`
	ActorID                        string    `json:"actor_id"`
	ActorRole                      string    `json:"actor_role"`
	Action                         string    `json:"action"`
	TargetRepository               string    `json:"target_repository"`
	BaseRef                        string    `json:"base_ref"`
	BaseSHA                        string    `json:"base_sha"`
	HeadRef                        string    `json:"head_ref"`
	HeadSHA                        string    `json:"head_sha"`
	Draft                          bool      `json:"draft"`
	TitleHash                      string    `json:"title_hash"`
	BodyHash                       string    `json:"body_hash"`
	Result                         string    `json:"result"`
	Timestamp                      time.Time `json:"timestamp"`
	RollbackInstructions           string    `json:"rollback_instructions"`
}

type Epic7IssueToPRProjection struct {
	GeneratedAt       string                    `json:"generated_at"`
	Source            string                    `json:"source"`
	Mode              Epic7IssueToPRMode        `json:"mode"`
	Trials            []Epic7TrialEvidence      `json:"trials"`
	GateHValidation   Epic7GateHValidation      `json:"gate_h_validation"`
	AuditReport       Epic7AuditEvidence        `json:"audit_report"`
	ProofOfWorkPacket Epic7ProofOfWorkAggregate `json:"proof_of_work_packet"`
	Errors            []string                  `json:"errors,omitempty"`
}

type Epic11DocsDraftPRProjection struct {
	GeneratedAt      string                         `json:"generated_at"`
	Source           string                         `json:"source"`
	Mode             Epic11DocsDraftPRMode          `json:"mode"`
	Target           Epic11DraftPullRequestTarget   `json:"target"`
	PolicyBundleID   string                         `json:"policy_bundle_id"`
	PolicyBundleHash string                         `json:"policy_bundle_hash"`
	MutationResult   Epic11DraftPullRequestResult   `json:"mutation_result"`
	ReceiptEvidence  Epic11ExecutionReceiptEvidence `json:"receipt_evidence"`
	ForbiddenActions []string                       `json:"forbidden_actions"`
	EventGraphRefs   []string                       `json:"event_graph_refs"`
}

type Epic7GateHValidation struct {
	Status  string   `json:"status"`
	Missing []string `json:"missing,omitempty"`
}

type Epic7AuditEvidence struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type Epic7ProofOfWorkAggregate struct {
	ID               string                    `json:"id"`
	Status           string                    `json:"status"`
	Summary          string                    `json:"summary"`
	TrialRefs        []string                  `json:"trial_refs"`
	ForbiddenActions []Epic7ProtectedActionRef `json:"forbidden_actions"`
	ResidualRisks    []Epic7ProofOfWorkItem    `json:"residual_risks"`
	EventGraphRefs   []string                  `json:"event_graph_refs"`
}

type Epic7TrialEvidence struct {
	TrialID                string                      `json:"trial_id"`
	Status                 string                      `json:"status"`
	IssueFixture           Epic7IssueFixture           `json:"issue_fixture"`
	IssueFixtureRef        string                      `json:"issue_fixture_ref"`
	ProposalPacketRef      string                      `json:"proposal_packet_ref"`
	ProofPacketRef         string                      `json:"proof_packet_ref"`
	PatchRef               string                      `json:"patch_ref,omitempty"`
	PRBodyRef              string                      `json:"pr_body_ref"`
	BranchPlanRef          string                      `json:"branch_plan_ref"`
	ValidationPlanRef      string                      `json:"validation_plan_ref"`
	RepairEvidenceRef      string                      `json:"repair_evidence_ref,omitempty"`
	RollbackEvidenceRef    string                      `json:"rollback_evidence_ref,omitempty"`
	HumanReviewEvidenceRef string                      `json:"human_review_evidence_ref,omitempty"`
	Proposal               Epic7PRProposalPacket       `json:"proposal"`
	ProofOfWorkPacket      Epic7TrialProofOfWorkPacket `json:"proof_of_work_packet"`
	AuthorityBoundary      []Epic7ProtectedActionRef   `json:"authority_boundary"`
	MultiRepoAuthority     *Epic7AuthorityGrant        `json:"multi_repo_authority,omitempty"`
	Checks                 Epic7TrialChecks            `json:"checks"`
	EventGraphRefs         []string                    `json:"event_graph_refs"`
	Missing                []string                    `json:"missing,omitempty"`
}

type Epic7IssueFixture struct {
	ID                 string   `json:"id"`
	SourceRepo         string   `json:"source_repo"`
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	Labels             []string `json:"labels"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type Epic7PRProposalPacket struct {
	ID                     string                    `json:"id"`
	TrialID                string                    `json:"trial_id"`
	IssueFixtureID         string                    `json:"issue_fixture_id"`
	ProposedPRTitle        string                    `json:"proposed_pr_title"`
	ProposedPRBody         string                    `json:"proposed_pr_body"`
	ProposedBranchName     string                    `json:"proposed_branch_name"`
	ChangedFileIntent      []Epic7ChangedFileIntent  `json:"changed_file_intent"`
	ImplementationPlan     []string                  `json:"implementation_plan"`
	ProposedDiffRef        string                    `json:"proposed_diff_ref,omitempty"`
	ValidationPlanRef      string                    `json:"validation_plan_ref"`
	RepairEvidenceRef      string                    `json:"repair_evidence_ref,omitempty"`
	RollbackEvidenceRef    string                    `json:"rollback_evidence_ref,omitempty"`
	HumanReviewEvidenceRef string                    `json:"human_review_evidence_ref,omitempty"`
	AuthorityBoundary      []Epic7ProtectedActionRef `json:"authority_boundary"`
	ProposedOnly           bool                      `json:"proposed_only"`
	Applied                bool                      `json:"applied"`
}

type Epic7ChangedFileIntent struct {
	Repo       string `json:"repo"`
	Path       string `json:"path"`
	ChangeType string `json:"change_type"`
	Summary    string `json:"summary"`
}

type Epic7ProtectedActionRef struct {
	Action              Epic7ProtectedAction `json:"action"`
	Status              string               `json:"status"`
	Summary             string               `json:"summary"`
	AuthorityRequestID  string               `json:"authority_request_id,omitempty"`
	AuthorityDecisionID string               `json:"authority_decision_id,omitempty"`
	HumanApprovalID     string               `json:"human_approval_id,omitempty"`
}

type Epic7AuthorityGrant struct {
	AuthorityRequestID  string   `json:"authority_request_id"`
	AuthorityDecisionID string   `json:"authority_decision_id"`
	HumanApprovalID     string   `json:"human_approval_id"`
	Scope               []string `json:"scope"`
	Decision            string   `json:"decision"`
	Summary             string   `json:"summary"`
}

type Epic7TrialChecks struct {
	IssueFixturePresent                        bool `json:"issue_fixture_present"`
	ProposalPacketPresent                      bool `json:"proposal_packet_present"`
	ProofPacketPresent                         bool `json:"proof_packet_present"`
	ProposedOnly                               bool `json:"proposed_only"`
	NoRepositoryMutation                       bool `json:"no_repository_mutation"`
	NoExecutionReceipt                         bool `json:"no_execution_receipt"`
	ForbiddenActionsSeparated                  bool `json:"forbidden_actions_separated"`
	RepairEvidencePresent                      bool `json:"repair_evidence_present,omitempty"`
	RepairTestUpdateIntentPresent              bool `json:"repair_test_update_intent_present,omitempty"`
	ExplicitMultiRepoAuthorityRecorded         bool `json:"explicit_multi_repo_authority_recorded,omitempty"`
	MultiRepoProposalRemainsProposedOnly       bool `json:"multi_repo_proposal_remains_proposed_only,omitempty"`
	SelfImprovementHumanReviewPresent          bool `json:"self_improvement_human_review_present,omitempty"`
	SelfImprovementRollbackEvidencePresent     bool `json:"self_improvement_rollback_evidence_present,omitempty"`
	SelfImprovementProposalRemainsProposedOnly bool `json:"self_improvement_proposal_remains_proposed_only,omitempty"`
}

type Epic7TrialProofOfWorkPacket struct {
	ID                        string                    `json:"id"`
	Status                    string                    `json:"status"`
	Summary                   string                    `json:"summary"`
	IssueFixture              Epic7ProofOfWorkItem      `json:"issue_fixture"`
	PRProposal                Epic7ProofOfWorkItem      `json:"pr_proposal"`
	DiffEvidence              Epic7ProofOfWorkItem      `json:"diff_evidence"`
	ValidationPlan            Epic7ProofOfWorkItem      `json:"validation_plan"`
	RepairEvidence            *Epic7ProofOfWorkItem     `json:"repair_evidence,omitempty"`
	RollbackEvidence          *Epic7ProofOfWorkItem     `json:"rollback_evidence,omitempty"`
	HumanReviewEvidence       *Epic7ProofOfWorkItem     `json:"human_review_evidence,omitempty"`
	AuthorityBoundary         []Epic7ProtectedActionRef `json:"authority_boundary"`
	MultiRepoAuthority        *Epic7AuthorityGrant      `json:"multi_repo_authority,omitempty"`
	ForbiddenActionSeparation []Epic7ProtectedActionRef `json:"forbidden_action_separation"`
	EventGraphRefs            []string                  `json:"event_graph_refs"`
}

type Epic7ProofOfWorkItem struct {
	Label       string   `json:"label"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	ArtifactRef string   `json:"artifact_ref,omitempty"`
	Refs        []string `json:"refs,omitempty"`
}

// Epic11DocsDraftPRPolicyBundleHash returns the hash of the built-in policy
// document that constrains the docs draft-PR live mutation.
func Epic11DocsDraftPRPolicyBundleHash() string {
	return epic7Hash(epic11PolicyBundleDocument())
}

// RunEpic11DocsDraftPRLiveMutation creates one live draft PR only after the
// merged docs#95 authority, the just-in-time human decision, and the policy
// adapter decision all match the exact target.
func RunEpic11DocsDraftPRLiveMutation(ctx context.Context, ts *TaskStore, opts Epic11DocsDraftPROptions) (Epic11DocsDraftPRRun, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = epic11NormalizeOptions(opts)
	if err := epic11ValidateOptions(ts, opts); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}

	ids := epic11IDs(opts)
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 11 Docs Draft PR Live Mutation",
		Description:            "Create one draft PR in transpara-ai/docs only after exact just-in-time authority and policy evidence match.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic11_docs_draft_pr_live_mutation",
		RiskClass:              "high",
		ExpectedOutputs:        []string{"artifacts/epic11/docs-draft-pr/projection.json", "artifacts/epic11/docs-draft-pr/execution_receipt.json"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 11 docs draft PR live mutation lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic11DocsDraftPRRun{}, err
		}
	}

	authorityReservation, err := epic11ReserveAuthorityUse(ts, opts, task.ID, causes)
	if err != nil {
		if blockErr := ts.TransitionTask(opts.Source, task.ID, StatusPolicyBlocked, "Epic 11 authority reservation failed before GitHub call: "+err.Error(), []string{ids.authorityRequest, ids.authorityDecision, ids.policyDecision}, causes, opts.ConversationID); blockErr != nil {
			return Epic11DocsDraftPRRun{}, fmt.Errorf("reserve authority use: %w; policy block task: %v", err, blockErr)
		}
		return Epic11DocsDraftPRRun{}, err
	}

	mutation := epic11Mutation(opts)

	// Bind the approved head SHA and dark-factory/ diff scope to the live remote
	// before the irreversible create. A moved branch or an out-of-scope file
	// fails closed here, so the PR side-effect never fires on unapproved content.
	headState, err := opts.Client.PreflightHead(ctx, mutation)
	if err != nil {
		if failErr := ts.TransitionTask(opts.Source, task.ID, StatusFailed, "Epic 11 remote head preflight failed before GitHub mutation: "+err.Error(), []string{ids.authorityRequest, ids.authorityDecision, ids.policyDecision}, causes, opts.ConversationID); failErr != nil {
			return Epic11DocsDraftPRRun{}, fmt.Errorf("preflight head: %w; fail task: %v", err, failErr)
		}
		return Epic11DocsDraftPRRun{}, fmt.Errorf("preflight head: %w", err)
	}
	if err := epic11ValidateRemoteHead(opts.Target, headState); err != nil {
		if failErr := ts.TransitionTask(opts.Source, task.ID, StatusFailed, "Epic 11 remote head preflight rejected the mutation: "+err.Error(), []string{ids.authorityRequest, ids.authorityDecision, ids.policyDecision}, causes, opts.ConversationID); failErr != nil {
			return Epic11DocsDraftPRRun{}, fmt.Errorf("validate remote head: %w; fail task: %v", err, failErr)
		}
		return Epic11DocsDraftPRRun{}, fmt.Errorf("validate remote head: %w", err)
	}

	result, err := opts.Client.CreateDraftPullRequest(ctx, mutation)
	if err != nil {
		if failErr := ts.TransitionTask(opts.Source, task.ID, StatusFailed, "Epic 11 draft PR client returned before confirmed draft PR creation: "+err.Error(), []string{ids.authorityRequest, ids.authorityDecision, ids.policyDecision}, causes, opts.ConversationID); failErr != nil {
			return Epic11DocsDraftPRRun{}, fmt.Errorf("create draft PR: %w; fail task: %v", err, failErr)
		}
		return Epic11DocsDraftPRRun{}, fmt.Errorf("create draft PR: %w", err)
	}
	if err := epic11ValidateMutationResult(opts.Target, result); err != nil {
		if failErr := ts.TransitionTask(opts.Source, task.ID, StatusFailed, "Epic 11 draft PR response failed post-confirmation validation: "+err.Error(), []string{ids.authorityRequest, ids.authorityDecision, ids.policyDecision}, causes, opts.ConversationID); failErr != nil {
			return Epic11DocsDraftPRRun{}, fmt.Errorf("validate draft PR response: %w; fail task: %v", err, failErr)
		}
		return Epic11DocsDraftPRRun{}, err
	}

	receiptEvidence := epic11ReceiptEvidence(ids, opts, result)
	receiptArtifactBody, err := epic11ReceiptArtifactBody(receiptEvidence)
	if err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	if err := ts.AddArtifact(opts.Source, task.ID, epic11ExecutionReceiptArtifactLabel, "application/json", receiptArtifactBody, causes, opts.ConversationID); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	graph, records, err := epic11RecordEventGraph(ids, opts, result, receiptEvidence)
	if err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	projection := epic11BuildProjection(ids, opts, result, receiptEvidence)
	if err := epic11WriteEvidence(opts.WorkingDir, opts, result, receiptEvidence, projection); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 11 docs draft PR live mutation evidence attached", causes, opts.ConversationID); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 11 draft PR creation evidence recorded", []string{ids.testRun, ids.gateResult, ids.executionReceipt}, causes, opts.ConversationID); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 11 docs draft PR live mutation certified for the single authorized draft PR creation", []string{ids.certification, ids.executionReceipt}, causes, opts.ConversationID); err != nil {
		return Epic11DocsDraftPRRun{}, err
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic11DocsDraftPRRun{}, err
	}

	return Epic11DocsDraftPRRun{
		Mode:              opts.Mode,
		WorkTask:          task,
		WorkProjection:    workProjection,
		EventGraph:        graph,
		Target:            opts.Target,
		PolicyBundleID:    Epic11PolicyBundleID,
		PolicyBundleHash:  Epic11DocsDraftPRPolicyBundleHash(),
		AuthorityRequest:  records.AuthorityRequest,
		AuthorityDecision: records.AuthorityDecision,
		PolicyDecision:    records.PolicyDecision,
		AuthorityReserve:  authorityReservation,
		ExecutionReceipt:  records.ExecutionReceipt,
		ReceiptEvidence:   receiptEvidence,
		MutationResult:    result,
		Projection:        projection,
	}, nil
}

type epic11FixtureIDs struct {
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	actorIdentity       string
	humanActorIdentity  string
	actorInvocation     string
	runtimeEnvelope     string
	runtimeResult       string
	knowledgeReference  string
	authorityRequest    string
	authorityDecision   string
	humanApproval       string
	policyDecision      string
	executionReceipt    string
	testCase            string
	testRun             string
	gateResult          string
	factoryRuntime      string
	releaseCandidate    string
	certification       string
	auditReport         string
}

type epic11GraphRecords struct {
	AuthorityRequest  v39.AuthorityRequest
	AuthorityDecision v39.AuthorityDecision
	PolicyDecision    v39.PolicyEngineAdapterDecision
	ExecutionReceipt  v39.ExecutionReceipt
}

func epic11NormalizeOptions(opts Epic11DocsDraftPROptions) Epic11DocsDraftPROptions {
	if opts.Mode == "" {
		opts.Mode = Epic11DocsDraftPRLiveMutationMode
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	return opts
}

func epic11ValidateOptions(ts *TaskStore, opts Epic11DocsDraftPROptions) error {
	if ts == nil {
		return errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return errors.New("working directory is required")
	}
	if opts.Mode != Epic11DocsDraftPRLiveMutationMode {
		return fmt.Errorf("unsupported Epic 11 mode %q", opts.Mode)
	}
	if opts.Client == nil {
		return errors.New("pull request creator is required")
	}
	if len(opts.PriorExecutionReceiptRefs) > 0 {
		return fmt.Errorf("authority decision already used by receipt refs: %s", strings.Join(opts.PriorExecutionReceiptRefs, ","))
	}
	if err := epic11ValidateTarget(opts.Target); err != nil {
		return err
	}
	if err := epic11ValidateAuthorityRequest(opts); err != nil {
		return err
	}
	if err := epic11ValidateAuthorityDecision(opts); err != nil {
		return err
	}
	priorAuthorityUseRefs, err := epic11PriorAuthorityUseRefs(ts, opts.AuthorityDecision.ID, opts.AuthorityDecision.SingleUseNonce)
	if err != nil {
		return err
	}
	if len(priorAuthorityUseRefs) > 0 {
		return fmt.Errorf("authority decision already used by durable authority refs: %s", strings.Join(priorAuthorityUseRefs, ","))
	}
	if err := epic11ValidatePolicyDecision(opts); err != nil {
		return err
	}
	return nil
}

func epic11ValidateTarget(target Epic11DraftPullRequestTarget) error {
	if target.Repository != Epic11TargetRepository {
		return fmt.Errorf("target repository %q is not authorized", target.Repository)
	}
	if target.BaseRef != Epic11TargetBaseRef {
		return fmt.Errorf("target base ref %q is not authorized", target.BaseRef)
	}
	if strings.TrimSpace(target.BaseSHA) == "" {
		return errors.New("target base SHA is required")
	}
	if !strings.HasPrefix(target.HeadRef, Epic11HeadRefPrefix) || target.HeadRef == Epic11HeadRefPrefix {
		return fmt.Errorf("target head ref %q must match %s*", target.HeadRef, Epic11HeadRefPrefix)
	}
	if strings.TrimSpace(target.HeadSHA) == "" {
		return errors.New("target head SHA is required")
	}
	if !target.HeadExistsOnOrigin {
		return errors.New("head branch must already exist on origin")
	}
	if !target.Draft {
		return errors.New("draft=true is required")
	}
	if !target.MaintainerCanModify {
		return errors.New("maintainer_can_modify=true is required")
	}
	if strings.TrimSpace(target.Title) == "" {
		return errors.New("PR title is required")
	}
	if strings.TrimSpace(target.Body) == "" {
		return errors.New("PR body is required")
	}
	if len(target.ChangedFiles) == 0 {
		return errors.New("changed file list is required")
	}
	for _, path := range target.ChangedFiles {
		clean, err := cleanRuntimeRelativePath(path)
		if err != nil {
			return fmt.Errorf("changed file %q is invalid: %w", path, err)
		}
		if clean != path {
			return fmt.Errorf("changed file %q must be normalized", path)
		}
		if !strings.HasPrefix(clean, "dark-factory/") {
			return fmt.Errorf("changed file %q is outside dark-factory/", path)
		}
	}
	if len(target.ValidationEvidenceRefs) == 0 {
		return errors.New("validation evidence refs are required")
	}
	if strings.TrimSpace(target.RollbackInstructions) == "" {
		return errors.New("rollback instructions are required")
	}
	return nil
}

func epic11ValidateAuthorityRequest(opts Epic11DocsDraftPROptions) error {
	req := opts.AuthorityRequest
	target := opts.Target
	for field, value := range map[string]string{
		"authority request ID":         req.ID,
		"authority request actor ID":   req.ActorID,
		"authority request actor role": req.ActorRole,
		"authority request nonce":      req.SingleUseNonce,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if req.ActorID != opts.Source.Value() {
		return fmt.Errorf("authority request actor %q does not match source %q", req.ActorID, opts.Source.Value())
	}
	if req.Action != Epic11ActionPullRequestCreate {
		return fmt.Errorf("authority request action %q is not authorized", req.Action)
	}
	if err := epic11EvidenceMatchesTarget("authority request", req.TargetRepository, req.BaseRef, req.BaseSHA, req.HeadRef, req.HeadSHA, req.TitleHash, req.BodyHash, req.PolicyBundleID, req.PolicyBundleHash, req.RollbackInstructions, req.ChangedFiles, req.ValidationEvidenceRefs, target); err != nil {
		return err
	}
	if req.RequestedAt.IsZero() {
		return errors.New("authority request requested_at is required")
	}
	if req.ExpiresAt.IsZero() {
		return errors.New("authority request expiry is required")
	}
	if !opts.Now.Before(req.ExpiresAt) {
		return errors.New("authority request is expired")
	}
	return nil
}

func epic11ValidateAuthorityDecision(opts Epic11DocsDraftPROptions) error {
	decision := opts.AuthorityDecision
	req := opts.AuthorityRequest
	target := opts.Target
	for field, value := range map[string]string{
		"authority decision ID":               decision.ID,
		"authority decision request ID":       decision.AuthorityRequestID,
		"authority decision actor ID":         decision.ActorID,
		"authority decision actor role":       decision.ActorRole,
		"authority decision decider actor ID": decision.DeciderActorID,
		"authority decision decider role":     decision.DeciderRole,
		"authority decision nonce":            decision.SingleUseNonce,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if decision.AuthorityRequestID != req.ID {
		return fmt.Errorf("authority decision request ID %q does not match %q", decision.AuthorityRequestID, req.ID)
	}
	if decision.ActorID != req.ActorID || decision.ActorRole != req.ActorRole {
		return errors.New("authority decision actor does not match authority request actor")
	}
	if !epic11AuthorityDecisionApproved(decision.Decision) {
		return fmt.Errorf("authority decision %q is not approved for the protected action", decision.Decision)
	}
	if decision.Action != req.Action {
		return fmt.Errorf("authority decision action %q does not match request action %q", decision.Action, req.Action)
	}
	if err := epic11EvidenceMatchesTarget("authority decision", decision.TargetRepository, decision.BaseRef, decision.BaseSHA, decision.HeadRef, decision.HeadSHA, decision.TitleHash, decision.BodyHash, decision.PolicyBundleID, decision.PolicyBundleHash, decision.RollbackInstructions, decision.ChangedFiles, decision.ValidationEvidenceRefs, target); err != nil {
		return err
	}
	if decision.SingleUseNonce != req.SingleUseNonce {
		return errors.New("authority decision nonce does not match authority request nonce")
	}
	if decision.ExpiresAt.IsZero() {
		return errors.New("authority decision expiry is required")
	}
	if !opts.Now.Before(decision.ExpiresAt) {
		return errors.New("authority decision is expired")
	}
	return nil
}

func epic11ValidatePolicyDecision(opts Epic11DocsDraftPROptions) error {
	decision := opts.PolicyDecision
	for field, value := range map[string]string{
		"policy decision ID":            decision.DecisionID,
		"policy adapter version":        decision.AdapterVersion,
		"policy raw decision":           decision.RawDecision,
		"policy authority decision ref": decision.AuthorityDecisionRef,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if decision.AdapterID != Epic11PolicyAdapterID {
		return fmt.Errorf("policy adapter %q is not authorized", decision.AdapterID)
	}
	if decision.PolicyBundleID != Epic11PolicyBundleID {
		return fmt.Errorf("policy bundle ID %q is not authorized", decision.PolicyBundleID)
	}
	if decision.PolicyBundleHash != Epic11DocsDraftPRPolicyBundleHash() {
		return errors.New("policy bundle hash is missing, stale, mismatched, or placeholder")
	}
	if decision.ProtectedActionType != Epic11ActionPullRequestCreate {
		return fmt.Errorf("policy action %q is not authorized", decision.ProtectedActionType)
	}
	if decision.ActorID != opts.AuthorityRequest.ActorID {
		return errors.New("policy actor does not match authority request actor")
	}
	if decision.AuthorityDecisionRef != opts.AuthorityDecision.ID {
		return errors.New("policy decision is not linked to the authority decision")
	}
	if decision.CanonicalDecision != "approval_required" {
		return fmt.Errorf("policy canonical decision %q does not approve the required human-gated action", decision.CanonicalDecision)
	}
	if decision.LatencyMS < 0 {
		return errors.New("policy latency must be >= 0")
	}
	if len(decision.ReasonCodes) == 0 {
		return errors.New("policy reason codes are required")
	}
	if len(decision.EvidenceRefs) == 0 {
		return errors.New("policy evidence refs are required")
	}
	if len(decision.InputFacts) == 0 {
		return errors.New("policy input facts are required")
	}
	return nil
}

func epic11EvidenceMatchesTarget(label, repo, baseRef, baseSHA, headRef, headSHA, titleHash, bodyHash, policyBundleID, policyBundleHash, rollback string, changedFiles, validationRefs []string, target Epic11DraftPullRequestTarget) error {
	if repo != target.Repository {
		return fmt.Errorf("%s repository %q does not match target %q", label, repo, target.Repository)
	}
	if baseRef != target.BaseRef || baseSHA != target.BaseSHA {
		return fmt.Errorf("%s base ref/SHA does not match target", label)
	}
	if headRef != target.HeadRef || headSHA != target.HeadSHA {
		return fmt.Errorf("%s head ref/SHA does not match target", label)
	}
	if titleHash != epic7Hash(target.Title) {
		return fmt.Errorf("%s title hash does not match target title", label)
	}
	if bodyHash != epic7Hash(target.Body) {
		return fmt.Errorf("%s body hash does not match target body", label)
	}
	if !equalStringSlices(changedFiles, target.ChangedFiles) {
		return fmt.Errorf("%s changed file list does not match target", label)
	}
	if !equalStringSlices(validationRefs, target.ValidationEvidenceRefs) {
		return fmt.Errorf("%s validation evidence refs do not match target", label)
	}
	if policyBundleID != Epic11PolicyBundleID {
		return fmt.Errorf("%s policy bundle ID %q is not authorized", label, policyBundleID)
	}
	if policyBundleHash != Epic11DocsDraftPRPolicyBundleHash() {
		return fmt.Errorf("%s policy bundle hash is missing, stale, mismatched, or placeholder", label)
	}
	if rollback != target.RollbackInstructions {
		return fmt.Errorf("%s rollback instructions do not match target", label)
	}
	return nil
}

func epic11ValidateMutationResult(target Epic11DraftPullRequestTarget, result Epic11DraftPullRequestResult) error {
	if result.Repository != target.Repository {
		return fmt.Errorf("created PR repository %q does not match %q", result.Repository, target.Repository)
	}
	if result.Number <= 0 {
		return errors.New("created PR number is required")
	}
	if strings.TrimSpace(result.URL) == "" {
		return errors.New("created PR URL is required")
	}
	if strings.TrimSpace(result.GitHubResponseIDOrEquivalent) == "" {
		return errors.New("GitHub response ID is required")
	}
	if result.BaseRef != target.BaseRef || result.BaseSHA != target.BaseSHA {
		return errors.New("created PR base ref/SHA does not match target")
	}
	if result.HeadRef != target.HeadRef || result.HeadSHA != target.HeadSHA {
		return errors.New("created PR head ref/SHA does not match target")
	}
	if !result.Draft {
		return errors.New("created PR is not draft")
	}
	if result.State != "open" {
		return fmt.Errorf("created PR state %q is not open", result.State)
	}
	if result.CreatedAt.IsZero() {
		return errors.New("created PR timestamp is required")
	}
	return nil
}

func epic11Mutation(opts Epic11DocsDraftPROptions) Epic11DraftPullRequestMutation {
	target := opts.Target
	return Epic11DraftPullRequestMutation{
		Repository:          target.Repository,
		BaseRef:             target.BaseRef,
		BaseSHA:             target.BaseSHA,
		HeadRef:             target.HeadRef,
		HeadSHA:             target.HeadSHA,
		Title:               target.Title,
		Body:                target.Body,
		TitleHash:           epic7Hash(target.Title),
		BodyHash:            epic7Hash(target.Body),
		Draft:               true,
		MaintainerCanModify: target.MaintainerCanModify,
	}
}

// epic11ValidateRemoteHead fails closed unless the live remote head SHA matches
// the approved head SHA and every file in the base...head diff is under
// dark-factory/. It binds the authorized decision to the actual remote state so
// a moved branch or an out-of-scope file cannot reach the create call.
func epic11ValidateRemoteHead(target Epic11DraftPullRequestTarget, state Epic11RemoteHeadState) error {
	if strings.TrimSpace(state.HeadSHA) == "" {
		return errors.New("remote head SHA is empty")
	}
	if state.HeadSHA != target.HeadSHA {
		return fmt.Errorf("remote head SHA %q does not match approved head SHA %q", state.HeadSHA, target.HeadSHA)
	}
	if len(state.ChangedFiles) == 0 {
		return errors.New("remote head diff is empty")
	}
	for _, path := range state.ChangedFiles {
		if !strings.HasPrefix(path, "dark-factory/") {
			return fmt.Errorf("remote changed file %q is outside dark-factory/", path)
		}
	}
	return nil
}

func epic11ReserveAuthorityUse(ts *TaskStore, opts Epic11DocsDraftPROptions, taskID types.EventID, causes []types.EventID) (Epic11AuthorityReservationEvidence, error) {
	epic11AuthorityReservationMu.Lock()
	defer epic11AuthorityReservationMu.Unlock()

	priorAuthorityUseRefs, err := epic11PriorAuthorityUseRefs(ts, opts.AuthorityDecision.ID, opts.AuthorityDecision.SingleUseNonce)
	if err != nil {
		return Epic11AuthorityReservationEvidence{}, err
	}
	if len(priorAuthorityUseRefs) > 0 {
		return Epic11AuthorityReservationEvidence{}, fmt.Errorf("authority decision already used by durable authority refs: %s", strings.Join(priorAuthorityUseRefs, ","))
	}

	reservation := epic11AuthorityReservationEvidence(opts, taskID)
	body, err := epic11AuthorityReservationArtifactBody(reservation)
	if err != nil {
		return Epic11AuthorityReservationEvidence{}, err
	}
	if err := ts.AddArtifact(opts.Source, taskID, epic11AuthorityReservationArtifactLabel, "application/json", body, causes, opts.ConversationID); err != nil {
		return Epic11AuthorityReservationEvidence{}, fmt.Errorf("reserve Epic 11 authority use: %w", err)
	}
	return reservation, nil
}

func epic11PriorAuthorityUseRefs(ts *TaskStore, authorityDecisionID, singleUseNonce string) ([]string, error) {
	var refs []string
	after := types.None[types.Cursor]()
	for {
		page, err := ts.store.ByType(EventTypeTaskArtifact, 1000, after)
		if err != nil {
			return nil, fmt.Errorf("fetch Epic 11 authority use artifacts: %w", err)
		}
		for _, ev := range page.Items() {
			content, ok := ev.Content().(TaskArtifactContent)
			if !ok {
				continue
			}
			switch content.Label {
			case epic11AuthorityReservationArtifactLabel:
				var reservation Epic11AuthorityReservationEvidence
				if err := json.Unmarshal([]byte(content.Body), &reservation); err != nil {
					return nil, fmt.Errorf("decode Epic 11 authority reservation artifact %s: %w", ev.ID().Value(), err)
				}
				if reservation.AuthorityDecisionRef == authorityDecisionID || reservation.SingleUseNonce == singleUseNonce {
					refs = append(refs, ev.ID().Value()+":"+reservation.ReservationID)
				}
			case epic11ExecutionReceiptArtifactLabel:
				var receipt Epic11ExecutionReceiptEvidence
				if err := json.Unmarshal([]byte(content.Body), &receipt); err != nil {
					return nil, fmt.Errorf("decode Epic 11 execution receipt artifact %s: %w", ev.ID().Value(), err)
				}
				if receipt.AuthorityDecisionRef == authorityDecisionID || receipt.SingleUseNonce == singleUseNonce {
					refs = append(refs, ev.ID().Value()+":"+receipt.ReceiptID)
				}
			}
		}
		if !page.HasMore() {
			break
		}
		after = page.Cursor()
		if after.IsNone() {
			return nil, errors.New("fetch Epic 11 authority use artifacts: page has more results but no cursor")
		}
	}
	return refs, nil
}

func epic11AuthorityReservationArtifactBody(reservation Epic11AuthorityReservationEvidence) (string, error) {
	body, err := json.MarshalIndent(reservation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal Epic 11 authority reservation artifact: %w", err)
	}
	return string(body), nil
}

func epic11ReceiptArtifactBody(receipt Epic11ExecutionReceiptEvidence) (string, error) {
	body, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal Epic 11 execution receipt artifact: %w", err)
	}
	return string(body), nil
}

func epic11AuthorityReservationEvidence(opts Epic11DocsDraftPROptions, taskID types.EventID) Epic11AuthorityReservationEvidence {
	return Epic11AuthorityReservationEvidence{
		ReservationID:                  "reserve_epic11_docs_draft_pr_create_" + epic7Hash(opts.AuthorityDecision.ID+":"+opts.AuthorityDecision.SingleUseNonce),
		TaskRef:                        taskID.Value(),
		AuthorityRequestRef:            opts.AuthorityRequest.ID,
		AuthorityDecisionRef:           opts.AuthorityDecision.ID,
		PolicyEngineAdapterDecisionRef: opts.PolicyDecision.DecisionID,
		SingleUseNonce:                 opts.AuthorityDecision.SingleUseNonce,
		ActorID:                        opts.AuthorityRequest.ActorID,
		ActorRole:                      opts.AuthorityRequest.ActorRole,
		Action:                         Epic11ActionPullRequestCreate,
		TargetRepository:               opts.Target.Repository,
		BaseRef:                        opts.Target.BaseRef,
		BaseSHA:                        opts.Target.BaseSHA,
		HeadRef:                        opts.Target.HeadRef,
		HeadSHA:                        opts.Target.HeadSHA,
		Draft:                          true,
		TitleHash:                      epic7Hash(opts.Target.Title),
		BodyHash:                       epic7Hash(opts.Target.Body),
		Result:                         "reserved",
		Timestamp:                      opts.Now,
		RollbackInstructions:           opts.Target.RollbackInstructions,
	}
}

func epic11ReceiptEvidence(ids epic11FixtureIDs, opts Epic11DocsDraftPROptions, result Epic11DraftPullRequestResult) Epic11ExecutionReceiptEvidence {
	return Epic11ExecutionReceiptEvidence{
		ReceiptID:                      ids.executionReceipt,
		AuthorityRequestRef:            opts.AuthorityRequest.ID,
		AuthorityDecisionRef:           opts.AuthorityDecision.ID,
		PolicyEngineAdapterDecisionRef: opts.PolicyDecision.DecisionID,
		SingleUseNonce:                 opts.AuthorityDecision.SingleUseNonce,
		ActorID:                        opts.AuthorityRequest.ActorID,
		ActorRole:                      opts.AuthorityRequest.ActorRole,
		Action:                         Epic11ActionPullRequestCreate,
		TargetRepository:               opts.Target.Repository,
		BaseRef:                        opts.Target.BaseRef,
		BaseSHA:                        opts.Target.BaseSHA,
		HeadRef:                        opts.Target.HeadRef,
		HeadSHA:                        opts.Target.HeadSHA,
		Draft:                          true,
		PRNumber:                       result.Number,
		PRURL:                          result.URL,
		TitleHash:                      epic7Hash(opts.Target.Title),
		BodyHash:                       epic7Hash(opts.Target.Body),
		GitHubResponseIDOrEquivalent:   result.GitHubResponseIDOrEquivalent,
		Result:                         "succeeded",
		Timestamp:                      result.CreatedAt,
		ValidationEvidenceRefs:         cloneStrings(opts.Target.ValidationEvidenceRefs),
		RollbackInstructions:           opts.Target.RollbackInstructions,
	}
}

func epic11RecordEventGraph(ids epic11FixtureIDs, opts Epic11DocsDraftPROptions, result Epic11DraftPullRequestResult, receipt Epic11ExecutionReceiptEvidence) (*v39.InMemoryStore, epic11GraphRecords, error) {
	graph := v39.NewInMemoryStore()
	createdAt := opts.Now
	receiptRef := ids.executionReceipt
	decisionRef := opts.AuthorityDecision.ID
	targetID := epic11TargetID(opts.Target)
	authRequest := v39.AuthorityRequest{CommonNode: epic11Common(ids.authorityRequest, v39.TypeAuthorityRequest, "recorded", createdAt), ActorID: opts.AuthorityRequest.ActorID, ActorRole: opts.AuthorityRequest.ActorRole, Action: Epic11ActionPullRequestCreate, TargetType: "pull_request", TargetID: targetID, RiskClass: "high", Reason: "Create one draft PR in transpara-ai/docs under merged docs#95 and exact just-in-time authority.", ProposedCommand: strPtr("create draft pull request only; no ready, merge, push, update, close, or rollback mutation"), EvidenceRefs: append([]string{epic11DocsPRRef, ids.knowledgeReference}, opts.Target.ValidationEvidenceRefs...), ExpiresAt: &opts.AuthorityRequest.ExpiresAt}
	authDecision := v39.AuthorityDecision{CommonNode: epic11Common(ids.authorityDecision, v39.TypeAuthorityDecision, "approved", createdAt), AuthorityRequestID: opts.AuthorityRequest.ID, DeciderActorID: opts.AuthorityDecision.DeciderActorID, DeciderRole: opts.AuthorityDecision.DeciderRole, Decision: "ApprovalRequired", Reason: "Explicit human approval grants exactly one draft PR creation and no other mutation.", Scope: []string{Epic11ActionPullRequestCreate, opts.Target.Repository, opts.Target.BaseRef, opts.Target.BaseSHA, opts.Target.HeadRef, opts.Target.HeadSHA, epic7Hash(opts.Target.Title), epic7Hash(opts.Target.Body), Epic11PolicyBundleID, Epic11DocsDraftPRPolicyBundleHash(), opts.AuthorityDecision.SingleUseNonce}, Conditions: []string{"draft=true", "single-use", "head branch already exists on origin", "changed files under dark-factory/", "no PR ready/merge/update/close", "no branch push", "manual rollback instructions only"}, ExpiresAt: &opts.AuthorityDecision.ExpiresAt}
	policyDecision := v39.PolicyEngineAdapterDecision{CommonNode: epic11Common(ids.policyDecision, v39.TypePolicyEngineAdapterDecision, "recorded", createdAt), DecisionID: opts.PolicyDecision.DecisionID, AdapterID: opts.PolicyDecision.AdapterID, AdapterVersion: opts.PolicyDecision.AdapterVersion, PolicyBundleID: opts.PolicyDecision.PolicyBundleID, PolicyBundleHash: opts.PolicyDecision.PolicyBundleHash, ProtectedActionType: opts.PolicyDecision.ProtectedActionType, ActorID: opts.PolicyDecision.ActorID, ResourceRefs: cloneStrings(opts.PolicyDecision.ResourceRefs), InputFacts: cloneMap(opts.PolicyDecision.InputFacts), RawDecision: opts.PolicyDecision.RawDecision, CanonicalDecision: opts.PolicyDecision.CanonicalDecision, ReasonCodes: cloneStrings(opts.PolicyDecision.ReasonCodes), EvidenceRefs: cloneStrings(opts.PolicyDecision.EvidenceRefs), LatencyMS: opts.PolicyDecision.LatencyMS, AuthorityDecisionRef: &decisionRef, ExecutionReceiptRef: &receiptRef}
	executionReceipt := v39.ExecutionReceipt{CommonNode: epic11Common(ids.executionReceipt, v39.TypeExecutionReceipt, "recorded", createdAt), AuthorityDecisionID: opts.AuthorityDecision.ID, ActorInvocationID: &ids.actorInvocation, Action: Epic11ActionPullRequestCreate, TargetID: fmt.Sprintf("%s#%d", opts.Target.Repository, result.Number), Result: "succeeded", EvidenceRefs: []string{ids.runtimeResult, ids.policyDecision, opts.PolicyDecision.DecisionID, result.URL, result.GitHubResponseIDOrEquivalent}}
	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: epic11Common(ids.factoryOrder, v39.TypeFactoryOrder, "certified", createdAt), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-95-merged-" + epic11DocsMergeSHA, SourceIntentRef: epic11DocsPRRef, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic11Common(ids.requirement, v39.TypeRequirement, "accepted", createdAt), FactoryOrderID: ids.factoryOrder, Text: "Create exactly one draft PR in transpara-ai/docs under just-in-time authority and policy evidence.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: epic11Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, "verified", createdAt), RequirementID: ids.requirement, Text: "The GitHub call occurs only after exact AuthorityRequest, AuthorityDecision, PolicyEngineAdapterDecision, and policy bundle evidence match, and the ExecutionReceipt is recorded only after GitHub confirms draft=true.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "authority_policy_execution_receipt_trace", OwnerRole: "maintainer", RiskClass: "high"},
		&v39.Task{CommonNode: epic11Common(ids.task, v39.TypeTask, "certified", createdAt), FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic11_docs_draft_pr_live_mutation", State: "certified", Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic11Common(ids.actorIdentity, v39.TypeActorIdentity, "active", createdAt), ActorID: opts.AuthorityRequest.ActorID, ActorType: "agent", IdentityMode: "externally_managed"},
		&v39.ActorIdentity{CommonNode: epic11Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active", createdAt), ActorID: opts.AuthorityDecision.DeciderActorID, ActorType: "human", IdentityMode: "externally_managed"},
		&v39.ActorInvocation{CommonNode: epic11Common(ids.actorInvocation, v39.TypeActorInvocation, "succeeded", createdAt), TaskID: ids.task, Runtime: "codex", ActorID: opts.AuthorityRequest.ActorID, InputContractHash: epic7Hash(opts.AuthorityRequest.ID + ":" + opts.AuthorityDecision.ID), OutputContractHash: strPtr(epic7Hash(receipt.PRURL + ":" + receipt.GitHubResponseIDOrEquivalent))},
		&v39.RuntimeEnvelope{CommonNode: epic11Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded", createdAt), RuntimeAdapterID: "work_live_github_pr_creator", RuntimeAdapterVersion: opts.PolicyDecision.AdapterVersion, FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: opts.AuthorityRequest.ActorID, AuthorityDecisionRef: opts.AuthorityDecision.ID, AllowedFiles: []string{"dark-factory/**"}, DeniedFiles: []string{".git", "../", ".env", "secrets.env"}, AllowedCommands: []string{"pull_request.create:draft"}, DeniedCommands: []string{"pull_request.ready_for_review", "pull_request.merge", "pull_request.close", "pull_request.update", "pull_request.request_review", "issue.comment", "label.mutate", "branch.push", "repo.push.default_branch", "repo.merge.main", "worktree.merge.main", "production.deploy", "secret.access", "upstream.push"}, NetworkPolicy: "restricted", SecretsPolicy: "scoped", WorkingDirectory: opts.WorkingDir, Timeout: "1m", ResourceLimits: map[string]any{"max_pull_requests_created": 1, "target_repository": opts.Target.Repository, "draft_required": true, "network_scope": "github.com/transpara-ai/docs pull_request.create only", "secret_scope": "GitHub token supplied by caller; not read or stored by Work"}, ExpectedOutputs: []string{"open draft pull request URL", "ExecutionReceipt"}, OutputContract: map[string]any{"mode": string(opts.Mode), "action": Epic11ActionPullRequestCreate}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "ActorInvocation -> AuthorityRequest -> AuthorityDecision -> ExecutionReceipt", "AuthorityDecision -> PolicyEngineAdapterDecision", "Task -> RuntimeEnvelope -> RuntimeResult"}, PostRunValidationPlan: []string{"epic11ValidateOptions", "epic11ValidateMutationResult"}, EnvelopeHash: epic7Hash("epic11-envelope:" + opts.AuthorityRequest.ID + ":" + opts.AuthorityDecision.ID)},
		&v39.RuntimeResult{CommonNode: epic11Common(ids.runtimeResult, v39.TypeRuntimeResult, "succeeded", createdAt), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "work_live_github_pr_creator", StartedAt: createdAt, CompletedAt: result.CreatedAt, ExitStatus: "succeeded", ArtifactRefs: []string{ids.executionReceipt, ids.policyDecision}, ChangedFiles: []string{}, CommandLog: []string{"0:validate_authority_request:succeeded", "1:validate_authority_decision:succeeded", "2:validate_policy_engine_adapter_decision:succeeded", "3:create_draft_pull_request:succeeded", "4:record_execution_receipt:succeeded"}, NetworkAccessLog: []string{"github.com/transpara-ai/docs:pull_request.create:draft_only"}, SecretAccessLog: []string{"github token supplied by caller; not read or stored by Work"}, PolicyDecisionRefs: []string{ids.policyDecision}, PostRunValidationRefs: []string{ids.testRun}},
		&authRequest,
		&authDecision,
		&v39.HumanApproval{CommonNode: epic11Common(ids.humanApproval, v39.TypeHumanApproval, "approved", createdAt), RequestRef: opts.AuthorityRequest.ID, ApproverActorID: opts.AuthorityDecision.DeciderActorID, ApproverRole: opts.AuthorityDecision.DeciderRole, Decision: "approved", Reason: "Approve exactly one draft PR creation in transpara-ai/docs under docs#95."},
		&policyDecision,
		&executionReceipt,
		&v39.FactoryRuntimeVersion{CommonNode: epic11Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active", createdAt), RuntimeVersion: "3.9.20-epic11-docs-draft-pr-live-mutation", RuntimeRefs: []string{"work.live_github_pr_creator@" + opts.PolicyDecision.AdapterVersion}},
		&v39.TestCase{CommonNode: epic11Common(ids.testCase, v39.TypeTestCase, "active", createdAt), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 11 docs draft PR live mutation evidence", TestType: "unit", Path: strPtr("work/epic7_issue_pr_autonomy_test.go")},
		&v39.TestRun{CommonNode: epic11Common(ids.testRun, v39.TypeTestRun, "pass", createdAt), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: epic11Common(ids.gateResult, v39.TypeGateResult, "pass", createdAt), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "epic11_docs_draft_pr_live_mutation", EvidenceRefs: []string{ids.testRun, ids.authorityRequest, ids.authorityDecision, ids.policyDecision, ids.executionReceipt}},
		&v39.ReleaseCandidate{CommonNode: epic11Common(ids.releaseCandidate, v39.TypeReleaseCandidate, "certified", createdAt), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.policyDecision, ids.executionReceipt}},
		&v39.Certification{CommonNode: epic11Common(ids.certification, v39.TypeCertification, "certified", createdAt), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: opts.AuthorityDecision.DeciderActorID, Reason: "Epic 11 draft PR creation satisfied exact authority, policy, and receipt evidence for one draft PR only.", EvidenceRefs: []string{ids.gateResult, ids.executionReceipt}},
		&v39.AuditReport{CommonNode: epic11Common(ids.auditReport, v39.TypeAuditReport, "complete", createdAt), TargetType: "release_candidate", TargetID: ids.releaseCandidate, TraceScore: 1},
	}
	if err := epic7AppendRecords(graph, records...); err != nil {
		return nil, epic11GraphRecords{}, err
	}
	if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{CommonNode: epic11Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded", createdAt), ReferenceCreatedAt: createdAt, SourceSystem: "transpara-ai/docs", SourceRef: epic11KnowledgeSourceRef, SourceHashOrImmutableLocator: "sha256:docs-pr-95-merged-" + epic11DocsMergeSHA + "-reviewed-head-" + epic11DocsReviewedHead, RetrievedAt: createdAt, UsedByActor: opts.AuthorityRequest.ActorID, UsedInTask: ids.task, InfluenceSummary: "Epic 11 docs#95 authorized only draft PR creation in transpara-ai/docs with exact JIT authority, policy bundle evidence, and post-confirmation receipt.", RiskScope: "high", TrustLevel: "human_authorized", FreshnessStatus: "current", RedactionState: "none"}}); err != nil {
		return nil, epic11GraphRecords{}, err
	}
	if err := epic11AppendEdges(graph, ids, createdAt); err != nil {
		return nil, epic11GraphRecords{}, err
	}
	return graph, epic11GraphRecords{AuthorityRequest: authRequest, AuthorityDecision: authDecision, PolicyDecision: policyDecision, ExecutionReceipt: executionReceipt}, nil
}

func epic11AppendEdges(graph *v39.InMemoryStore, ids epic11FixtureIDs, createdAt time.Time) error {
	edges := []v39.CommonEdge{
		epic11Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic11Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic11Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic11Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic11Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic11Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic11Edge("invoke_auth", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
		epic11Edge("auth_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
		epic11Edge("auth_human", v39.EdgeApprovedBy, ids.authorityRequest, ids.humanApproval, createdAt),
		epic11Edge("auth_receipt", v39.EdgeReceiptedBy, ids.authorityDecision, ids.executionReceipt, createdAt),
		epic11Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic11Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic11Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic11BuildProjection(ids epic11FixtureIDs, opts Epic11DocsDraftPROptions, result Epic11DraftPullRequestResult, receipt Epic11ExecutionReceiptEvidence) Epic11DocsDraftPRProjection {
	return Epic11DocsDraftPRProjection{
		GeneratedAt:      opts.Now.Format(time.RFC3339),
		Source:           "work-epic11-docs-draft-pr-live-mutation",
		Mode:             opts.Mode,
		Target:           opts.Target,
		PolicyBundleID:   Epic11PolicyBundleID,
		PolicyBundleHash: Epic11DocsDraftPRPolicyBundleHash(),
		MutationResult:   result,
		ReceiptEvidence:  receipt,
		ForbiddenActions: []string{"pull_request.ready_for_review", "pull_request.merge", "pull_request.close", "pull_request.update", "pull_request.request_review", "issue.comment", "label.mutate", "branch.push", "repo.push.default_branch", "repo.merge.main", "worktree.merge.main", "production.deploy", "secret.access", "capability.activate", "runtime.invoke.external", "repo.mutate.cross_repo", "upstream.push"},
		EventGraphRefs:   []string{egRef(v39.TypeAuthorityRequest, ids.authorityRequest), egRef(v39.TypeAuthorityDecision, ids.authorityDecision), egRef(v39.TypePolicyEngineAdapterDecision, ids.policyDecision), egRef(v39.TypeExecutionReceipt, ids.executionReceipt), egRef(v39.TypeGateResult, ids.gateResult)},
	}
}

func (p Epic11DocsDraftPRProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func epic11WriteEvidence(dir string, opts Epic11DocsDraftPROptions, result Epic11DraftPullRequestResult, receipt Epic11ExecutionReceiptEvidence, projection Epic11DocsDraftPRProjection) error {
	root := filepath.Join(dir, "artifacts", "epic11", "docs-draft-pr")
	files := map[string]any{
		"target.json":             opts.Target,
		"authority_request.json":  opts.AuthorityRequest,
		"authority_decision.json": opts.AuthorityDecision,
		"policy_decision.json":    opts.PolicyDecision,
		"mutation_result.json":    result,
		"execution_receipt.json":  receipt,
		"projection.json":         projection,
	}
	for name, value := range files {
		if err := epic7WriteJSON(filepath.Join(root, name), value); err != nil {
			return err
		}
	}
	return nil
}

func epic11IDs(opts Epic11DocsDraftPROptions) epic11FixtureIDs {
	return epic11FixtureIDs{
		factoryOrder:        "fo_epic11_docs_draft_pr_live_mutation",
		requirement:         "req_epic11_docs_draft_pr_live_mutation",
		acceptanceCriterion: "ac_epic11_docs_draft_pr_live_mutation",
		task:                "tsk_epic11_docs_draft_pr_live_mutation",
		actorIdentity:       "actor_identity_epic11_docs_draft_pr_creator",
		humanActorIdentity:  "actor_identity_epic11_human_authorizer",
		actorInvocation:     "invoke_epic11_docs_draft_pr_create",
		runtimeEnvelope:     "env_epic11_docs_draft_pr_create",
		runtimeResult:       "rr_epic11_docs_draft_pr_create",
		knowledgeReference:  "know_ref_epic11_docs95",
		authorityRequest:    opts.AuthorityRequest.ID,
		authorityDecision:   opts.AuthorityDecision.ID,
		humanApproval:       "human_app_epic11_docs_draft_pr_create",
		policyDecision:      opts.PolicyDecision.DecisionID,
		executionReceipt:    "exec_epic11_docs_draft_pr_create",
		testCase:            "tc_epic11_docs_draft_pr_live_mutation",
		testRun:             "tr_epic11_docs_draft_pr_live_mutation",
		gateResult:          "gate_epic11_docs_draft_pr_live_mutation",
		factoryRuntime:      "frv_epic11_docs_draft_pr_live_mutation",
		releaseCandidate:    "rc_epic11_docs_draft_pr_live_mutation",
		certification:       "cert_epic11_docs_draft_pr_live_mutation",
		auditReport:         "aud_epic11_docs_draft_pr_live_mutation",
	}
}

func epic11PolicyBundleDocument() string {
	return strings.Join([]string{
		"policy_bundle_id=df-v3.9.20-docs-draft-pr-create-only",
		"adapter_id=work-live-github-pr-policy-adapter",
		"allow=pull_request.create",
		"draft_required=true",
		"target_repository=transpara-ai/docs",
		"target_base_branch=main",
		"target_head_branch_pattern=codex/*",
		"head_branch_must_already_exist_on_origin=true",
		"changed_files_prefix=dark-factory/",
		"max_pull_requests_created_per_run=1",
		"require_exact_authority_request=true",
		"require_matching_authority_decision=true",
		"require_policy_bundle_hash=true",
		"require_post_confirmation_execution_receipt=true",
		"forbid=pull_request.ready_for_review,pull_request.merge,pull_request.close,pull_request.update,pull_request.request_review,issue.comment,label.mutate,branch.push,repo.push.default_branch,repo.merge.main,worktree.merge.main,production.deploy,secret.access,capability.activate,runtime.invoke.external,repo.mutate.cross_repo,upstream.push",
	}, "\n")
}

func epic11Common(id, typ, status string, createdAt time.Time) v39.CommonNode {
	return v39.CommonNode{ID: id, Type: typ, CreatedAt: createdAt, CreatedBy: epic11FixtureActorID, Status: &status, IdempotencyKey: "idem_" + id, CorrelationID: "corr_epic11_docs_draft_pr_live_mutation", SourceRefs: []string{epic11DocsPRRef}}
}

func epic11Edge(suffix, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic11_" + suffix + "_" + from + "_" + to
	return v39.CommonEdge{ID: id, Type: typ, FromID: from, ToID: to, CreatedAt: createdAt, CreatedBy: epic11FixtureActorID, CorrelationID: "corr_epic11_docs_draft_pr_live_mutation", IdempotencyKey: "idem_" + id}
}

func epic11TargetID(target Epic11DraftPullRequestTarget) string {
	return target.Repository + ":" + target.BaseRef + "<-" + target.HeadRef
}

func epic11AuthorityDecisionApproved(decision string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(decision), "_", ""))
	return normalized == "approvalrequired" || normalized == "approved"
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// RunEpic7IssueToPRProposalTrials executes the authorized proposal-only Gate H fixture.
func RunEpic7IssueToPRProposalTrials(ts *TaskStore, opts Epic7IssueToPROptions) (Epic7IssueToPRRun, error) {
	if ts == nil {
		return Epic7IssueToPRRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic7IssueToPRRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic7IssueToPRRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic7IssueToPRRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic7IssueToPRLocalProposalEvidence
	}
	if opts.Mode != Epic7IssueToPRLocalProposalEvidence {
		return Epic7IssueToPRRun{}, fmt.Errorf("unsupported Epic 7 fixture mode %q", opts.Mode)
	}

	ids := epic7IDs()
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 7 Issue-to-PR Proposal Autonomy Trials",
		Description:            "Run five bounded Gate H local proposal trials without creating live PRs or mutating repositories.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic7_issue_pr_autonomy",
		RiskClass:              "high",
		ExpectedOutputs:        []string{"artifacts/issue-pr/proposals/*.json", "artifacts/issue-pr/proof-of-work/*.json"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic7IssueToPRRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 7 issue-to-PR proposal fixture lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic7IssueToPRRun{}, err
		}
	}

	localArtifacts := epic7LocalArtifacts(opts.WorkingDir)
	trials, err := epic7BuildTrials(opts, localArtifacts)
	if err != nil {
		return Epic7IssueToPRRun{}, err
	}
	validation := epic7EvaluateGateH(trials, opts)
	graph, graphRun, err := epic7RecordEventGraph(ids, opts, trials, validation)
	if err != nil {
		return Epic7IssueToPRRun{}, err
	}
	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 7 Gate H issue-to-PR proposal evidence attached", causes, opts.ConversationID); err != nil {
		return Epic7IssueToPRRun{}, err
	}
	if graphRun.FailureID != "" {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{FailureIDs: []string{graphRun.FailureID}}, "Epic 7 negative Gate H fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic7IssueToPRRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 7 Gate H evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic7IssueToPRRun{}, err
	}
	if validation.Status == "pass" {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 7 issue-to-PR proposal trials certified for local proposal-only evidence", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic7IssueToPRRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 7 negative Gate H fixture rejected", []string{ids.gateResult, graphRun.FailureID}, causes, opts.ConversationID); err != nil {
		return Epic7IssueToPRRun{}, err
	}

	projection := epic7BuildProjection(ids, trials, validation, graphRun)
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic7IssueToPRRun{}, err
	}
	return Epic7IssueToPRRun{
		Mode:                  opts.Mode,
		WorkTask:              task,
		WorkProjection:        workProjection,
		EventGraph:            graph,
		FactoryOrderID:        ids.factoryOrder,
		RequirementID:         ids.requirement,
		AcceptanceCriterionID: ids.acceptanceCriterion,
		TaskID:                ids.task,
		ActorInvocationID:     ids.actorInvocation,
		RuntimeEnvelopeID:     ids.runtimeEnvelope,
		RuntimeResultID:       ids.runtimeResult,
		CapabilityArtifactID:  ids.capabilityArtifact,
		KnowledgeReferenceID:  ids.knowledgeReference,
		TestCaseID:            ids.testCase,
		TestRunID:             ids.testRun,
		GateResultID:          ids.gateResult,
		FailureID:             graphRun.FailureID,
		ReleaseCandidateID:    ids.releaseCandidate,
		CertificationID:       epic7CertificationID(graphRun.Certification),
		RejectionID:           graphRun.RejectionID,
		AuditReportID:         ids.auditReport,
		TraceCompleteness:     graphRun.Trace,
		CapabilityUsagePath:   graphRun.CapabilityUsagePath,
		KnowledgePath:         graphRun.KnowledgePath,
		GateHValidation:       validation,
		Certification:         graphRun.Certification,
		Rejection:             graphRun.Rejection,
		AuditReport:           graphRun.AuditReport,
		Projection:            projection,
		LocalArtifacts:        localArtifacts,
	}, nil
}

func (p Epic7IssueToPRProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

type epic7FixtureIDs struct {
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	actorIdentity       string
	humanActorIdentity  string
	actorInvocation     string
	runtimeEnvelope     string
	runtimeResult       string
	capabilityArtifact  string
	knowledgeReference  string
	planningProposal    string
	testCase            string
	testRun             string
	gateResult          string
	failure             string
	factoryRuntime      string
	releaseCandidate    string
	certification       string
	rejection           string
	auditReport         string
	proofPacket         string
}

type epic7GraphRun struct {
	DecisionID          string
	RejectionID         string
	FailureID           string
	Trace               v39.TraceCompletenessGateResult
	CapabilityUsagePath v39.RequiredPath
	KnowledgePath       v39.RequiredPath
	Certification       *v39.Certification
	Rejection           *v39.Rejection
	AuditReport         *v39.AuditReport
}

func epic7IDs() epic7FixtureIDs {
	return epic7FixtureIDs{
		factoryOrder:        "fo_epic7_issue_pr_autonomy",
		requirement:         "req_epic7_issue_pr_autonomy",
		acceptanceCriterion: "ac_epic7_issue_pr_autonomy",
		task:                "tsk_epic7_issue_pr_autonomy",
		actorIdentity:       "actor_identity_epic7_issue_pr_proposer",
		humanActorIdentity:  "actor_identity_epic7_human_reviewer",
		actorInvocation:     "invoke_epic7_issue_pr_proposal",
		runtimeEnvelope:     "env_epic7_issue_pr_proposal",
		runtimeResult:       "rr_epic7_issue_pr_proposal",
		capabilityArtifact:  "cap_art_epic7_issue_pr_proposer",
		knowledgeReference:  "know_ref_epic7_docs87",
		planningProposal:    "plan_epic7_issue_pr_autonomy",
		testCase:            "tc_epic7_issue_pr_autonomy",
		testRun:             "tr_epic7_issue_pr_autonomy",
		gateResult:          "gate_epic7_issue_pr_autonomy",
		failure:             "fail_epic7_issue_pr_autonomy",
		factoryRuntime:      "frv_epic7_issue_pr_autonomy",
		releaseCandidate:    "rc_epic7_issue_pr_autonomy",
		certification:       "cert_epic7_issue_pr_autonomy",
		rejection:           "rej_epic7_issue_pr_autonomy",
		auditReport:         "aud_epic7_issue_pr_autonomy",
		proofPacket:         "pow_epic7_issue_pr_autonomy",
	}
}

func epic7LocalArtifacts(dir string) Epic7LocalArtifacts {
	return Epic7LocalArtifacts{
		IssueDir:       filepath.Join(dir, "fixtures", "epic7", "issues"),
		ProposalDir:    filepath.Join(dir, "artifacts", "issue-pr", "proposals"),
		ProofDir:       filepath.Join(dir, "artifacts", "issue-pr", "proof-of-work"),
		PatchDir:       filepath.Join(dir, "artifacts", "issue-pr", "patches"),
		PRBodyDir:      filepath.Join(dir, "artifacts", "issue-pr", "pr-bodies"),
		BranchPlanDir:  filepath.Join(dir, "artifacts", "issue-pr", "branch-plans"),
		ValidationDir:  filepath.Join(dir, "artifacts", "issue-pr", "validation"),
		RepairDir:      filepath.Join(dir, "artifacts", "issue-pr", "repair"),
		RollbackDir:    filepath.Join(dir, "artifacts", "issue-pr", "rollback"),
		HumanReviewDir: filepath.Join(dir, "artifacts", "issue-pr", "human-review"),
	}
}

func epic7BuildTrials(opts Epic7IssueToPROptions, dirs Epic7LocalArtifacts) ([]Epic7TrialEvidence, error) {
	defs := epic7TrialDefinitions()
	trials := make([]Epic7TrialEvidence, 0, len(defs))
	for _, def := range defs {
		trial, err := epic7BuildTrial(def, opts, dirs)
		if err != nil {
			return nil, err
		}
		trials = append(trials, trial)
	}
	return trials, nil
}

type epic7TrialDefinition struct {
	id                 string
	sourceRepo         string
	title              string
	body               string
	labels             []string
	acceptanceCriteria []string
	intents            []Epic7ChangedFileIntent
	needsRepair        bool
	needsRollback      bool
	multiRepo          bool
	selfImprovement    bool
}

func epic7TrialDefinitions() []epic7TrialDefinition {
	return []epic7TrialDefinition{
		{
			id:         "trial_1_docs_only_issue_to_pr_proposal",
			sourceRepo: "transpara-ai/docs",
			title:      "Clarify Gate H operator handoff",
			body:       "Documentation-only issue requiring a proposed PR packet and no repository mutation.",
			labels:     []string{"documentation", "dark-factory", "gate-h"},
			acceptanceCriteria: []string{
				"proposal packet includes docs diff intent",
				"proof packet confirms no live PR creation",
			},
			intents: []Epic7ChangedFileIntent{{Repo: "transpara-ai/docs", Path: "dark-factory/v3.9/implementation/operators/gate-h-handoff.md", ChangeType: "create", Summary: "Propose a docs-only handoff note."}},
		},
		{
			id:         "trial_2_bounded_code_change_issue_to_pr_proposal",
			sourceRepo: "transpara-ai/work",
			title:      "Add bounded local proposal validator",
			body:       "Code-change issue requiring a proposed patch, validation plan, and no patch application.",
			labels:     []string{"enhancement", "work", "gate-h"},
			acceptanceCriteria: []string{
				"proposal packet includes a proposed-only patch",
				"validation plan names focused Go tests",
			},
			intents: []Epic7ChangedFileIntent{{Repo: "transpara-ai/work", Path: "proposal_validator.go", ChangeType: "create", Summary: "Propose a local-only proposal validator helper."}},
		},
		{
			id:          "trial_3_bug_fix_with_tests_and_repair_proposal",
			sourceRepo:  "transpara-ai/work",
			title:       "Repair proposal packet when tests fail",
			body:        "Bug-fix issue requiring failing-test evidence, proposed fix, proposed test update, and repair rationale.",
			labels:      []string{"bug", "tests", "gate-h"},
			needsRepair: true,
			acceptanceCriteria: []string{
				"proof packet includes failing-test evidence",
				"proposal packet includes proposed fix and test update",
				"repair evidence is visible",
			},
			intents: []Epic7ChangedFileIntent{
				{Repo: "transpara-ai/work", Path: "proposal_validator.go", ChangeType: "update", Summary: "Propose the bug fix."},
				{Repo: "transpara-ai/work", Path: "proposal_validator_test.go", ChangeType: "update", Summary: "Propose the regression test update."},
			},
		},
		{
			id:         "trial_4_multi_repo_proposal_requires_explicit_authority",
			sourceRepo: "transpara-ai/work",
			title:      "Coordinate Work and docs proposal without mutation",
			body:       "Multi-repo proposal issue requiring explicit authority evidence before proposal-only certification.",
			labels:     []string{"multi-repo", "authority", "gate-h"},
			multiRepo:  true,
			acceptanceCriteria: []string{
				"proposal without multi-repo authority is rejected",
				"proposal with explicit authority remains proposed-only",
			},
			intents: []Epic7ChangedFileIntent{
				{Repo: "transpara-ai/work", Path: "proposal_validator.go", ChangeType: "update", Summary: "Propose Work-side change."},
				{Repo: "transpara-ai/docs", Path: "dark-factory/v3.9/implementation/operators/gate-h-followup.md", ChangeType: "create", Summary: "Propose docs follow-up only after explicit authority."},
			},
		},
		{
			id:              "trial_5_self_improvement_proposal_human_reviewed_rollback_bound",
			sourceRepo:      "transpara-ai/work",
			title:           "Review rollback-bound proposal generator improvement",
			body:            "Self-improvement issue requiring human review and rollback evidence before proposal-only certification.",
			labels:          []string{"self-improvement", "rollback", "gate-h"},
			selfImprovement: true,
			needsRollback:   true,
			acceptanceCriteria: []string{
				"proposal without human review is rejected",
				"proposal with review and rollback evidence remains proposed-only",
			},
			intents: []Epic7ChangedFileIntent{{Repo: "transpara-ai/work", Path: "issue_pr_proposal_generator.go", ChangeType: "update", Summary: "Propose a rollback-bound generator improvement."}},
		},
	}
}

func epic7BuildTrial(def epic7TrialDefinition, opts Epic7IssueToPROptions, dirs Epic7LocalArtifacts) (Epic7TrialEvidence, error) {
	issue := Epic7IssueFixture{ID: def.id, SourceRepo: def.sourceRepo, Title: def.title, Body: def.body, Labels: append([]string(nil), def.labels...), AcceptanceCriteria: append([]string(nil), def.acceptanceCriteria...)}
	issuePath := filepath.Join(dirs.IssueDir, def.id+".json")
	proposalPath := filepath.Join(dirs.ProposalDir, def.id+".json")
	proofPath := filepath.Join(dirs.ProofDir, def.id+".json")
	patchPath := filepath.Join(dirs.PatchDir, def.id+".patch")
	prBodyPath := filepath.Join(dirs.PRBodyDir, def.id+".md")
	branchPlanPath := filepath.Join(dirs.BranchPlanDir, def.id+".json")
	validationPath := filepath.Join(dirs.ValidationDir, def.id+".md")
	repairPath := filepath.Join(dirs.RepairDir, def.id+".md")
	rollbackPath := filepath.Join(dirs.RollbackDir, def.id+".md")
	humanReviewPath := filepath.Join(dirs.HumanReviewDir, def.id+".md")

	applied := opts.AppliedPatchTrial == def.id
	authority := epic7AuthorityBoundary(def, opts)
	intents := epic7ChangedFileIntents(def, opts)
	multiRepoAuthority := epic7MultiRepoAuthorityGrant(def, opts)
	proposal := Epic7PRProposalPacket{
		ID:                 "proposal_" + def.id,
		TrialID:            def.id,
		IssueFixtureID:     def.id,
		ProposedPRTitle:    "Gate H proposal: " + def.title,
		ProposedPRBody:     epic7PRBody(def),
		ProposedBranchName: "proposal/gate-h/" + strings.TrimPrefix(def.id, "trial_"),
		ChangedFileIntent:  intents,
		ImplementationPlan: []string{"read the issue fixture", "prepare a proposed patch packet", "record validation and authority boundaries", "stop before live PR creation or repository mutation"},
		ProposedDiffRef:    patchPath,
		ValidationPlanRef:  validationPath,
		AuthorityBoundary:  authority,
		ProposedOnly:       !applied,
		Applied:            applied,
	}
	if def.needsRepair && !opts.MissingRepairEvidence {
		proposal.RepairEvidenceRef = repairPath
	}
	if def.needsRollback && !opts.MissingSelfImprovementRollback {
		proposal.RollbackEvidenceRef = rollbackPath
	}
	if def.selfImprovement && !opts.MissingSelfImprovementReview {
		proposal.HumanReviewEvidenceRef = humanReviewPath
	}
	checks := Epic7TrialChecks{
		IssueFixturePresent:                        opts.OmitIssueFixture != def.id,
		ProposalPacketPresent:                      opts.OmitProposalPacket != def.id,
		ProofPacketPresent:                         opts.OmitProposalPacket != def.id,
		ProposedOnly:                               proposal.ProposedOnly && !proposal.Applied,
		NoRepositoryMutation:                       !proposal.Applied && len(opts.CompletedForbiddenActions) == 0,
		NoExecutionReceipt:                         !opts.RecordExecutionReceipt,
		ForbiddenActionsSeparated:                  epic7ForbiddenActionsSeparated(authority),
		RepairEvidencePresent:                      !def.needsRepair || proposal.RepairEvidenceRef != "",
		RepairTestUpdateIntentPresent:              !def.needsRepair || epic7HasTestUpdateIntent(proposal.ChangedFileIntent),
		ExplicitMultiRepoAuthorityRecorded:         !def.multiRepo || multiRepoAuthority != nil,
		MultiRepoProposalRemainsProposedOnly:       !def.multiRepo || (multiRepoAuthority != nil && proposal.ProposedOnly && !proposal.Applied),
		SelfImprovementHumanReviewPresent:          !def.selfImprovement || proposal.HumanReviewEvidenceRef != "",
		SelfImprovementRollbackEvidencePresent:     !def.selfImprovement || proposal.RollbackEvidenceRef != "",
		SelfImprovementProposalRemainsProposedOnly: !def.selfImprovement || (proposal.HumanReviewEvidenceRef != "" && proposal.RollbackEvidenceRef != "" && proposal.ProposedOnly && !proposal.Applied),
	}
	proof := epic7BuildTrialProof(def, proposal, proofPath, checks, multiRepoAuthority)
	trial := Epic7TrialEvidence{
		TrialID:                def.id,
		Status:                 "pass",
		IssueFixture:           issue,
		IssueFixtureRef:        issuePath,
		ProposalPacketRef:      proposalPath,
		ProofPacketRef:         proofPath,
		PatchRef:               patchPath,
		PRBodyRef:              prBodyPath,
		BranchPlanRef:          branchPlanPath,
		ValidationPlanRef:      validationPath,
		RepairEvidenceRef:      proposal.RepairEvidenceRef,
		RollbackEvidenceRef:    proposal.RollbackEvidenceRef,
		HumanReviewEvidenceRef: proposal.HumanReviewEvidenceRef,
		Proposal:               proposal,
		ProofOfWorkPacket:      proof,
		AuthorityBoundary:      authority,
		MultiRepoAuthority:     multiRepoAuthority,
		Checks:                 checks,
		EventGraphRefs:         []string{egRef(v39.TypePlanningProposal, "plan_epic7_"+def.id), egRef(v39.TypeArtifact, "art_epic7_proposal_"+def.id), egRef(v39.TypeArtifact, "art_epic7_proof_"+def.id)},
	}
	trial.Missing = epic7TrialMissing(trial)
	if len(trial.Missing) > 0 {
		trial.Status = "fail"
		trial.ProofOfWorkPacket.Status = "fail"
	}

	if opts.OmitIssueFixture != def.id {
		if err := epic7WriteJSON(issuePath, issue); err != nil {
			return Epic7TrialEvidence{}, err
		}
	}
	if err := epic7WriteFile(patchPath, epic7Patch(def)); err != nil {
		return Epic7TrialEvidence{}, err
	}
	if err := epic7WriteFile(prBodyPath, proposal.ProposedPRBody); err != nil {
		return Epic7TrialEvidence{}, err
	}
	if err := epic7WriteJSON(branchPlanPath, map[string]any{"branch": proposal.ProposedBranchName, "base": "main", "push": "forbidden", "live_pr_create": "forbidden"}); err != nil {
		return Epic7TrialEvidence{}, err
	}
	if err := epic7WriteFile(validationPath, epic7ValidationPlan(def)); err != nil {
		return Epic7TrialEvidence{}, err
	}
	if proposal.RepairEvidenceRef != "" {
		if err := epic7WriteFile(repairPath, "Failing-test evidence is recorded; proposed fix and proposed test update remain unapplied.\n"); err != nil {
			return Epic7TrialEvidence{}, err
		}
	}
	if proposal.RollbackEvidenceRef != "" {
		if err := epic7WriteFile(rollbackPath, "Rollback plan: discard the proposed generator change and keep the current local proposal generator active.\n"); err != nil {
			return Epic7TrialEvidence{}, err
		}
	}
	if proposal.HumanReviewEvidenceRef != "" {
		if err := epic7WriteFile(humanReviewPath, "Human review evidence: reviewer approves proposal-only self-improvement with rollback evidence; no self-apply or activation is authorized.\n"); err != nil {
			return Epic7TrialEvidence{}, err
		}
	}
	if opts.OmitProposalPacket != def.id {
		if err := epic7WriteJSON(proposalPath, proposal); err != nil {
			return Epic7TrialEvidence{}, err
		}
		if err := epic7WriteJSON(proofPath, proof); err != nil {
			return Epic7TrialEvidence{}, err
		}
	}
	return trial, nil
}

func epic7AuthorityBoundary(def epic7TrialDefinition, opts Epic7IssueToPROptions) []Epic7ProtectedActionRef {
	out := make([]Epic7ProtectedActionRef, 0, len(epic7ProtectedActions()))
	for _, action := range epic7ProtectedActions() {
		if opts.OmitProtectedAction == action {
			continue
		}
		status := "forbidden"
		summary := "Action is outside the bounded Gate H fixture and is not executed."
		if action == Epic7ActionPullRequestPropose {
			status = "proposed"
			summary = "The fixture may produce a local pull-request proposal packet only."
		}
		if epic7ActionCompleted(action, opts.CompletedForbiddenActions) {
			status = "completed"
			summary = "Injected forbidden-action evidence; Gate H must reject this trial."
		}
		ref := Epic7ProtectedActionRef{
			Action:              action,
			Status:              status,
			Summary:             summary,
			AuthorityRequestID:  "auth_req_epic7_" + def.id + "_" + epic7ActionSlug(action),
			AuthorityDecisionID: "auth_dec_epic7_" + def.id + "_" + epic7ActionSlug(action),
			HumanApprovalID:     "human_app_epic7_" + def.id + "_" + epic7ActionSlug(action),
		}
		out = append(out, ref)
	}
	return out
}

func epic7ChangedFileIntents(def epic7TrialDefinition, opts Epic7IssueToPROptions) []Epic7ChangedFileIntent {
	out := make([]Epic7ChangedFileIntent, 0, len(def.intents))
	for _, intent := range def.intents {
		if def.needsRepair && opts.MissingRepairTestUpdateIntent && strings.HasSuffix(intent.Path, "_test.go") {
			continue
		}
		out = append(out, intent)
	}
	return out
}

func epic7HasTestUpdateIntent(intents []Epic7ChangedFileIntent) bool {
	for _, intent := range intents {
		if strings.HasSuffix(intent.Path, "_test.go") && intent.ChangeType == "update" {
			return true
		}
	}
	return false
}

func epic7MultiRepoAuthorityGrant(def epic7TrialDefinition, opts Epic7IssueToPROptions) *Epic7AuthorityGrant {
	if !def.multiRepo || opts.MissingMultiRepoAuthority {
		return nil
	}
	return &Epic7AuthorityGrant{
		AuthorityRequestID:  "auth_req_epic7_" + def.id + "_multi_repo_proposal",
		AuthorityDecisionID: "auth_dec_epic7_" + def.id + "_multi_repo_proposal",
		HumanApprovalID:     "human_app_epic7_" + def.id + "_multi_repo_proposal",
		Scope:               []string{"transpara-ai/work:proposal-only", "transpara-ai/docs:proposal-only"},
		Decision:            "ApprovalRequired",
		Summary:             "Explicit human authority permits recording a multi-repo proposal packet only; no repository is mutated.",
	}
}

func epic7BuildTrialProof(def epic7TrialDefinition, proposal Epic7PRProposalPacket, proofPath string, checks Epic7TrialChecks, multiRepoAuthority *Epic7AuthorityGrant) Epic7TrialProofOfWorkPacket {
	status := "pass"
	if len(epic7ChecksMissing(def, proposal, checks)) > 0 {
		status = "fail"
	}
	var repair *Epic7ProofOfWorkItem
	if def.needsRepair {
		repairStatus := "missing"
		repairSummary := "Repair evidence is missing."
		if proposal.RepairEvidenceRef != "" {
			repairStatus = "recorded"
			repairSummary = "Failing-test evidence, proposed fix, proposed test update, and repair rationale are recorded."
		}
		repair = &Epic7ProofOfWorkItem{Label: "Repair evidence", Status: repairStatus, Summary: repairSummary, ArtifactRef: proposal.RepairEvidenceRef}
	}
	var rollback *Epic7ProofOfWorkItem
	if def.needsRollback {
		rollbackStatus := "missing"
		rollbackSummary := "Rollback evidence is missing."
		if proposal.RollbackEvidenceRef != "" {
			rollbackStatus = "recorded"
			rollbackSummary = "Human-reviewed self-improvement proposal includes rollback evidence and remains unapplied."
		}
		rollback = &Epic7ProofOfWorkItem{Label: "Rollback evidence", Status: rollbackStatus, Summary: rollbackSummary, ArtifactRef: proposal.RollbackEvidenceRef}
	}
	var humanReview *Epic7ProofOfWorkItem
	if def.selfImprovement {
		reviewStatus := "missing"
		reviewSummary := "Human review evidence is missing."
		if proposal.HumanReviewEvidenceRef != "" {
			reviewStatus = "recorded"
			reviewSummary = "Human reviewer approves only proposal evidence with rollback; no self-apply, merge, or activation is authorized."
		}
		humanReview = &Epic7ProofOfWorkItem{Label: "Human review evidence", Status: reviewStatus, Summary: reviewSummary, ArtifactRef: proposal.HumanReviewEvidenceRef}
	}
	return Epic7TrialProofOfWorkPacket{
		ID:      "pow_" + def.id,
		Status:  status,
		Summary: "Gate H local issue-to-PR proposal proof packet for " + def.id + ".",
		IssueFixture: Epic7ProofOfWorkItem{
			Label:       "Issue fixture",
			Status:      boolStatus(checks.IssueFixturePresent),
			Summary:     def.title + " from " + def.sourceRepo,
			ArtifactRef: "fixtures/epic7/issues/" + def.id + ".json",
		},
		PRProposal: Epic7ProofOfWorkItem{
			Label:       "Proposed PR title/body/branch",
			Status:      boolStatus(checks.ProposalPacketPresent),
			Summary:     proposal.ProposedPRTitle + " on " + proposal.ProposedBranchName,
			ArtifactRef: proposal.ID,
			Refs:        []string{proposal.ProposedBranchName, proofPath},
		},
		DiffEvidence: Epic7ProofOfWorkItem{
			Label:       "Proposed diff",
			Status:      boolStatus(proposal.ProposedOnly && !proposal.Applied),
			Summary:     "Changed-file intent and patch artifact are proposed-only; no patch is applied.",
			ArtifactRef: proposal.ProposedDiffRef,
		},
		ValidationPlan: Epic7ProofOfWorkItem{
			Label:       "Validation plan",
			Status:      "recorded",
			Summary:     "Focused unit tests and full Work verification are planned before PR readiness.",
			ArtifactRef: proposal.ValidationPlanRef,
		},
		RepairEvidence:            repair,
		RollbackEvidence:          rollback,
		HumanReviewEvidence:       humanReview,
		AuthorityBoundary:         append([]Epic7ProtectedActionRef(nil), proposal.AuthorityBoundary...),
		MultiRepoAuthority:        multiRepoAuthority,
		ForbiddenActionSeparation: epic7ForbiddenActionRefs(proposal.AuthorityBoundary),
		EventGraphRefs:            []string{egRef(v39.TypeGateResult, "gate_epic7_issue_pr_autonomy")},
	}
}

func epic7EvaluateGateH(trials []Epic7TrialEvidence, opts Epic7IssueToPROptions) Epic7GateHValidation {
	seen := map[string]bool{}
	var missing []string
	required := map[string]bool{}
	for _, def := range epic7TrialDefinitions() {
		required[def.id] = false
	}
	for _, trial := range trials {
		if _, ok := required[trial.TrialID]; ok {
			required[trial.TrialID] = true
		}
		missing = appendUniqueStrings(missing, trial.Missing, seen)
	}
	for trialID, found := range required {
		if !found {
			missing = appendUniqueStrings(missing, []string{"missing required trial " + trialID}, seen)
		}
	}
	for _, action := range opts.CompletedForbiddenActions {
		missing = appendUniqueStrings(missing, []string{"forbidden action completed: " + string(action)}, seen)
	}
	if opts.RecordExecutionReceipt {
		missing = appendUniqueStrings(missing, []string{"ExecutionReceipt recorded"}, seen)
	}
	status := "pass"
	if len(missing) > 0 {
		status = "fail"
	}
	return Epic7GateHValidation{Status: status, Missing: missing}
}

func epic7TrialMissing(trial Epic7TrialEvidence) []string {
	def := epic7DefinitionByID(trial.TrialID)
	return epic7ChecksMissing(def, trial.Proposal, trial.Checks)
}

func epic7ChecksMissing(def epic7TrialDefinition, proposal Epic7PRProposalPacket, checks Epic7TrialChecks) []string {
	seen := map[string]bool{}
	var missing []string
	if !checks.IssueFixturePresent {
		missing = appendUniqueStrings(missing, []string{"missing issue fixture " + def.id}, seen)
	}
	if !checks.ProposalPacketPresent {
		missing = appendUniqueStrings(missing, []string{"missing proposal packet " + def.id}, seen)
	}
	if !checks.ProofPacketPresent {
		missing = appendUniqueStrings(missing, []string{"missing proof-of-work packet " + def.id}, seen)
	}
	if !checks.ProposedOnly || proposal.Applied {
		missing = appendUniqueStrings(missing, []string{"proposed-only boundary failed for " + def.id}, seen)
	}
	if !checks.NoRepositoryMutation {
		missing = appendUniqueStrings(missing, []string{"repository mutation evidence present for " + def.id}, seen)
	}
	if !checks.NoExecutionReceipt {
		missing = appendUniqueStrings(missing, []string{"ExecutionReceipt evidence present for " + def.id}, seen)
	}
	if !checks.ForbiddenActionsSeparated {
		missing = appendUniqueStrings(missing, []string{"forbidden action separation failed for " + def.id}, seen)
	}
	if def.needsRepair {
		if !checks.RepairEvidencePresent {
			missing = appendUniqueStrings(missing, []string{"repair evidence missing for " + def.id}, seen)
		}
		if !checks.RepairTestUpdateIntentPresent {
			missing = appendUniqueStrings(missing, []string{"repair proposed test update missing for " + def.id}, seen)
		}
	}
	if def.multiRepo {
		if !checks.ExplicitMultiRepoAuthorityRecorded {
			missing = appendUniqueStrings(missing, []string{"multi-repo proposal authority evidence is missing"}, seen)
		}
		if !checks.MultiRepoProposalRemainsProposedOnly {
			missing = appendUniqueStrings(missing, []string{"multi-repo proposal with explicit authority did not remain proposed-only"}, seen)
		}
	}
	if def.selfImprovement {
		if !checks.SelfImprovementHumanReviewPresent {
			missing = appendUniqueStrings(missing, []string{"self-improvement human review evidence is missing"}, seen)
		}
		if !checks.SelfImprovementRollbackEvidencePresent {
			missing = appendUniqueStrings(missing, []string{"self-improvement rollback evidence is missing"}, seen)
		}
		if !checks.SelfImprovementProposalRemainsProposedOnly {
			missing = appendUniqueStrings(missing, []string{"self-improvement proposal with review and rollback evidence did not remain proposed-only"}, seen)
		}
	}
	return missing
}

func epic7RecordEventGraph(ids epic7FixtureIDs, opts Epic7IssueToPROptions, trials []Epic7TrialEvidence, validation Epic7GateHValidation) (*v39.InMemoryStore, epic7GraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic7FixtureTime()
	taskStatus := "certified"
	testRunStatus := "pass"
	runtimeStatus := "succeeded"
	releaseStatus := "certified"
	acceptanceStatus := "verified"
	if validation.Status != "pass" {
		taskStatus = "rejected"
		testRunStatus = "fail"
		runtimeStatus = "failed"
		releaseStatus = "rejected"
		acceptanceStatus = "rejected"
	}
	taskCommon := epic7Common(ids.task, v39.TypeTask, taskStatus)
	taskCommon.SourceRefs = []string{ids.capabilityArtifact, epic7KnowledgeSourceRef}
	proposalArtifactIDs := epic7ProposalArtifactIDs(trials)
	proofArtifactIDs := epic7ProofArtifactIDs(trials)
	firstArtifact := epic7FirstArtifactID(trials)
	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: epic7Common(ids.factoryOrder, v39.TypeFactoryOrder, taskStatus), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-87-merged-" + epic7DocsMergeSHA, SourceIntentRef: epic7DocsPRRef, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic7Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Prove local issue-to-PR proposal autonomy with auditable proposal packets and no protected action execution.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: epic7Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, acceptanceStatus), RequirementID: ids.requirement, Text: "Gate H passes only when all five local issue fixtures produce proposed-only PR packets, proof-of-work packets, authority boundaries, and no live PR/push/merge/deploy/protected execution evidence.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "issue_to_pr_proposal_trace", OwnerRole: "maintainer", RiskClass: "high"},
		&v39.Task{CommonNode: taskCommon, FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic7_issue_pr_autonomy", State: taskStatus, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic7Common(ids.actorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic7FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic7Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic7FixtureHumanActorID, ActorType: "human", IdentityMode: "fixture"},
		&v39.CapabilityArtifact{CommonNode: epic7Common(ids.capabilityArtifact, v39.TypeCapabilityArtifact, "active"), ArtifactID: ids.capabilityArtifact, ArtifactType: "workflow_pack", Name: "Epic 7 Gate H issue-to-PR proposer", ArtifactVersion: "v1", SourceRepoOrOrigin: "transpara-ai/work", ContentHash: epic7Hash(strings.Join(epic7TrialIDs(trials), "\n")), Owner: "work", RiskClass: "high", ActivationScope: "fixture_only", EvalRefs: []string{ids.testCase}, HumanReviewRef: epic7DocsReviewedHead, RollbackRef: "not_applicable_local_proposal_fixture", UsageLoggingRequired: true},
		&v39.PlanningProposal{CommonNode: epic7Common(ids.planningProposal, v39.TypePlanningProposal, "proposed"), FactoryOrderID: ids.factoryOrder, FactoryOrderVersion: 1, Requirements: []string{ids.requirement}, AcceptanceCriteria: []string{ids.acceptanceCriterion}, Assumptions: []string{"Pull requests are represented as local proposal packets because EventGraph v3.9 has no dedicated PullRequest record.", "All protected actions remain forbidden unless separately authorized outside this fixture."}, ArchitectureOptions: []string{"local_issue_to_pr_proposal_packets"}, RecommendedOptionID: strPtr("local_issue_to_pr_proposal_packets"), TaskDrafts: []string{ids.task}, RequiresHumanReview: true},
		&v39.ActorInvocation{CommonNode: epic7Common(ids.actorInvocation, v39.TypeActorInvocation, runtimeStatus), TaskID: ids.task, Runtime: "local", ActorID: epic7FixtureActorID, InputContractHash: epic7Hash("epic7-input:" + opts.WorkingDir), OutputContractHash: strPtr(epic7Hash("epic7-output:" + strings.Join(append(proposalArtifactIDs, proofArtifactIDs...), ":")))},
		&v39.RuntimeEnvelope{CommonNode: epic7Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_issue_pr_proposal_fixture", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: epic7FixtureActorID, AuthorityDecisionRef: "human_authorized_in_chat_2026-06-02_docs_main_" + epic7ShortSHA(epic7DocsMergeSHA), AllowedFiles: []string{"fixtures/epic7/issues/**", "artifacts/issue-pr/**"}, DeniedFiles: []string{".git", "../", ".env", "secrets.env"}, AllowedCommands: []string{"write_issue_fixture", "write_proposal_packet", "write_proof_packet"}, DeniedCommands: []string{"gh pr create", "git push", "git merge", "gh pr merge", "deploy", "protected_execution.run", "capability.activate"}, NetworkPolicy: "disabled", SecretsPolicy: "none", WorkingDirectory: opts.WorkingDir, Timeout: "1s", ResourceLimits: map[string]any{"max_live_prs_created": 0, "max_branch_pushes": 0, "max_repos_mutated": 0}, ExpectedOutputs: []string{"artifacts/issue-pr/proposals/*.json", "artifacts/issue-pr/proof-of-work/*.json"}, OutputContract: map[string]any{"mode": string(opts.Mode), "gate": "gate_h_issue_to_pr_proposal"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"epic7EvaluateGateH", "go test ./..."}, EnvelopeHash: epic7Hash("epic7-envelope:" + string(opts.Mode))},
		&v39.RuntimeResult{CommonNode: epic7Common(ids.runtimeResult, v39.TypeRuntimeResult, runtimeStatus), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_issue_pr_proposal_fixture", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: runtimeStatus, ArtifactRefs: append(proposalArtifactIDs, proofArtifactIDs...), ChangedFiles: epic7RuntimeChangedFiles(trials), CommandLog: epic7CommandLog(trials, opts), NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{"proposal_only_boundary"}, PostRunValidationRefs: []string{ids.testRun}},
		&v39.TestCase{CommonNode: epic7Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 7 issue-to-PR proposal Gate H evidence", TestType: "unit", Path: strPtr("work/epic7_issue_pr_autonomy_test.go")},
		&v39.TestRun{CommonNode: epic7Common(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: epic7Common(ids.gateResult, v39.TypeGateResult, validation.Status), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_h_issue_to_pr_proposal_autonomy", EvidenceRefs: append([]string{ids.testRun}, append(proposalArtifactIDs, proofArtifactIDs...)...)},
	}
	records = append(records, epic7TrialRecords(trials)...)
	if validation.Status == "fail" {
		records = append(records, &v39.Failure{CommonNode: epic7Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "gate_h_issue_pr_proposal_blocked", Severity: "high", Summary: strings.Join(validation.Missing, "; ")})
	}
	if opts.RecordExecutionReceipt {
		records = append(records, &v39.ExecutionReceipt{CommonNode: epic7Common("exec_epic7_forbidden_receipt", v39.TypeExecutionReceipt, "forbidden"), AuthorityDecisionID: "auth_dec_epic7_forbidden_receipt", ActorInvocationID: &ids.actorInvocation, Action: string(Epic7ActionPullRequestCreate), TargetID: ids.task, Result: "blocked", EvidenceRefs: []string{ids.gateResult}})
	}
	if err := epic7AppendRecords(graph, records...); err != nil {
		return nil, epic7GraphRun{}, err
	}
	if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{CommonNode: epic7Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded"), ReferenceCreatedAt: createdAt, SourceSystem: "transpara-ai/docs", SourceRef: epic7KnowledgeSourceRef, SourceHashOrImmutableLocator: "sha256:docs-pr-87-merged-" + epic7DocsMergeSHA + "-reviewed-head-" + epic7DocsReviewedHead, RetrievedAt: createdAt, UsedByActor: epic7FixtureActorID, UsedInTask: ids.task, InfluenceSummary: "Gate H authorization packet shaped issue fixtures, PR proposal packets, protected-action separation, and stop conditions.", RiskScope: "high", TrustLevel: "human_authorized", FreshnessStatus: "current", RedactionState: "none"}}); err != nil {
		return nil, epic7GraphRun{}, err
	}
	if _, err := graph.RecordCapabilityUsage(ids.task, ids.capabilityArtifact, epic7Common("edge_epic7_used_capability", v39.TypeCapabilityArtifact, "recorded")); err != nil {
		return nil, epic7GraphRun{}, err
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic7Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-epic7-issue-pr-proposal", CapabilityVersionRefs: []string{}, RuntimeRefs: []string{"work.local_issue_pr_proposal_fixture@1"}}); err != nil {
		return nil, epic7GraphRun{}, err
	}
	if err := epic7AppendEdges(graph, ids, trials, firstArtifact, createdAt, validation.Status == "fail"); err != nil {
		return nil, epic7GraphRun{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: epic7Common(ids.releaseCandidate, v39.TypeReleaseCandidate, releaseStatus), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: append(proposalArtifactIDs, proofArtifactIDs...)})
	if err != nil {
		return nil, epic7GraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	capabilityPath, _ := graph.CapabilityUsageEvidencePath(rc.CommonNode.ID)
	knowledgePath, _ := graph.AdvisoryReferenceEvidencePath(rc.CommonNode.ID)
	if validation.Status == "pass" && traceErr != nil {
		return nil, epic7GraphRun{}, traceErr
	}
	if validation.Status == "pass" {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic7Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: epic7FixtureHumanActorID, Reason: "Gate H issue-to-PR proposal evidence is complete for the bounded local Work fixture.", EvidenceRefs: []string{ids.gateResult, ids.testRun}})
		if err != nil {
			return nil, epic7GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic7Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic7GraphRun{}, err
		}
		return graph, epic7GraphRun{DecisionID: cert.CommonNode.ID, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Certification: cert, AuditReport: audit}, nil
	}
	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic7Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: epic7FixtureHumanActorID, Reason: "Gate H issue-to-PR proposal evidence is incomplete or unsafe: " + strings.Join(validation.Missing, "; "), EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic7GraphRun{}, err
	}
	audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic7Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
	if err != nil {
		return nil, epic7GraphRun{}, err
	}
	return graph, epic7GraphRun{DecisionID: rejection.CommonNode.ID, RejectionID: rejection.CommonNode.ID, FailureID: ids.failure, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath, Rejection: rejection, AuditReport: audit}, nil
}

func epic7TrialRecords(trials []Epic7TrialEvidence) []v39.Record {
	var records []v39.Record
	for _, trial := range trials {
		def := epic7DefinitionByID(trial.TrialID)
		status := trial.Status
		issueArtifact := "art_epic7_issue_" + trial.TrialID
		proposalArtifact := "art_epic7_proposal_" + trial.TrialID
		proofArtifact := "art_epic7_proof_" + trial.TrialID
		patchArtifact := "art_epic7_patch_" + trial.TrialID
		prBodyArtifact := "art_epic7_pr_body_" + trial.TrialID
		branchArtifact := "art_epic7_branch_plan_" + trial.TrialID
		validationArtifact := "art_epic7_validation_plan_" + trial.TrialID
		codeChanges := make([]v39.Record, 0, len(trial.Proposal.ChangedFileIntent))
		for index, intent := range trial.Proposal.ChangedFileIntent {
			codeChanges = append(codeChanges, &v39.CodeChange{CommonNode: epic7Common(epic7CodeChangeID(trial.TrialID, index), v39.TypeCodeChange, epic7CodeChangeStatus(trial)), ArtifactID: patchArtifact, ActorInvocationID: "invoke_epic7_issue_pr_proposal", Repo: intent.Repo, Path: intent.Path, BeforeHash: strPtr("sha256:fixture_base"), AfterHash: epic7Hash(epic7Patch(def)), ChangeType: intent.ChangeType})
		}
		records = append(records,
			&v39.Artifact{CommonNode: epic7Common(issueArtifact, v39.TypeArtifact, boolArtifactStatus(trial.Checks.IssueFixturePresent)), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "document", Path: &trial.IssueFixtureRef, ContentHash: strPtr(epic7Hash(trial.IssueFixture.Title + trial.IssueFixture.Body))},
			&v39.Artifact{CommonNode: epic7Common(proposalArtifact, v39.TypeArtifact, boolArtifactStatus(trial.Checks.ProposalPacketPresent)), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "document", Path: &trial.ProposalPacketRef, ContentHash: strPtr(epic7Hash(trial.Proposal.ProposedPRTitle + trial.Proposal.ProposedPRBody))},
			&v39.Artifact{CommonNode: epic7Common(proofArtifact, v39.TypeArtifact, boolArtifactStatus(trial.Checks.ProofPacketPresent)), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "report", Path: &trial.ProofPacketRef, ContentHash: strPtr(epic7Hash(trial.ProofOfWorkPacket.Summary + status))},
			&v39.Artifact{CommonNode: epic7Common(patchArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "code", Path: &trial.PatchRef, ContentHash: strPtr(epic7Hash(epic7Patch(def)))},
			&v39.Artifact{CommonNode: epic7Common(prBodyArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "document", Path: &trial.PRBodyRef, ContentHash: strPtr(epic7Hash(trial.Proposal.ProposedPRBody))},
			&v39.Artifact{CommonNode: epic7Common(branchArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "config", Path: &trial.BranchPlanRef, ContentHash: strPtr(epic7Hash(trial.Proposal.ProposedBranchName))},
			&v39.Artifact{CommonNode: epic7Common(validationArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "test", Path: &trial.ValidationPlanRef, ContentHash: strPtr(epic7Hash(epic7ValidationPlan(def)))},
		)
		records = append(records, codeChanges...)
		if trial.RepairEvidenceRef != "" {
			repairArtifact := "art_epic7_repair_" + trial.TrialID
			records = append(records, &v39.Artifact{CommonNode: epic7Common(repairArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "report", Path: &trial.RepairEvidenceRef, ContentHash: strPtr(epic7Hash("repair:" + trial.TrialID))})
		}
		if trial.RollbackEvidenceRef != "" {
			rollbackArtifact := "art_epic7_rollback_" + trial.TrialID
			records = append(records, &v39.Artifact{CommonNode: epic7Common(rollbackArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "report", Path: &trial.RollbackEvidenceRef, ContentHash: strPtr(epic7Hash("rollback:" + trial.TrialID))})
		}
		if trial.HumanReviewEvidenceRef != "" {
			humanReviewArtifact := "art_epic7_human_review_" + trial.TrialID
			humanReviewRecord := "review_epic7_" + trial.TrialID
			records = append(records,
				&v39.Artifact{CommonNode: epic7Common(humanReviewArtifact, v39.TypeArtifact, "verified"), TaskID: strPtr("tsk_epic7_issue_pr_autonomy"), ArtifactType: "report", Path: &trial.HumanReviewEvidenceRef, ContentHash: strPtr(epic7Hash("human-review:" + trial.TrialID))},
				&v39.HumanReview{CommonNode: epic7Common(humanReviewRecord, v39.TypeHumanReview, "approved"), ReviewerActorID: epic7FixtureHumanActorID, ReviewerRole: "maintainer", Rationale: "Human reviewer approves proposal-only self-improvement with rollback evidence; no self-apply, merge, or capability activation is authorized."},
			)
		}
		if trial.MultiRepoAuthority != nil {
			grant := trial.MultiRepoAuthority
			records = append(records,
				&v39.AuthorityRequest{CommonNode: epic7Common(grant.AuthorityRequestID, v39.TypeAuthorityRequest, "recorded"), ActorID: epic7FixtureActorID, ActorRole: "local_proposal_generator", Action: "multi_repo.proposal_packet", TargetType: "proposal_packet", TargetID: proposalArtifact, RiskClass: "high", Reason: grant.Summary, ProposedCommand: strPtr("write local multi-repo proposal packet"), EvidenceRefs: []string{proposalArtifact, proofArtifact}},
				&v39.AuthorityDecision{CommonNode: epic7Common(grant.AuthorityDecisionID, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: grant.AuthorityRequestID, DeciderActorID: epic7FixtureHumanActorID, DeciderRole: "maintainer", Decision: grant.Decision, Reason: grant.Summary, Scope: append([]string(nil), grant.Scope...), Conditions: []string{"proposal-only", "no live GitHub mutation", "no branch push", "no merge", "no deploy"}},
				&v39.HumanApproval{CommonNode: epic7Common(grant.HumanApprovalID, v39.TypeHumanApproval, "approved"), RequestRef: grant.AuthorityRequestID, ApproverActorID: epic7FixtureHumanActorID, ApproverRole: "maintainer", Decision: "approved", Reason: grant.Summary},
			)
		}
		for _, action := range trial.AuthorityBoundary {
			records = append(records,
				&v39.AuthorityRequest{CommonNode: epic7Common(action.AuthorityRequestID, v39.TypeAuthorityRequest, "recorded"), ActorID: epic7FixtureActorID, ActorRole: "local_proposal_generator", Action: string(action.Action), TargetType: "proposal_packet", TargetID: proposalArtifact, RiskClass: "high", Reason: action.Summary, ProposedCommand: strPtr(epic7ProposedCommand(action.Action)), EvidenceRefs: []string{proposalArtifact}},
				&v39.AuthorityDecision{CommonNode: epic7Common(action.AuthorityDecisionID, v39.TypeAuthorityDecision, epic7AuthorityStatus(action)), AuthorityRequestID: action.AuthorityRequestID, DeciderActorID: epic7FixtureHumanActorID, DeciderRole: "maintainer", Decision: epic7AuthorityDecision(action), Reason: action.Summary, Scope: []string{string(action.Action)}, Conditions: []string{"local proposal evidence only", "no live GitHub mutation", "no push", "no merge", "no deploy"}},
				&v39.HumanApproval{CommonNode: epic7Common(action.HumanApprovalID, v39.TypeHumanApproval, epic7HumanApprovalStatus(action)), RequestRef: action.AuthorityRequestID, ApproverActorID: epic7FixtureHumanActorID, ApproverRole: "maintainer", Decision: epic7HumanApprovalDecision(action), Reason: action.Summary},
			)
		}
	}
	return records
}

func epic7AppendEdges(graph *v39.InMemoryStore, ids epic7FixtureIDs, trials []Epic7TrialEvidence, firstArtifact string, createdAt time.Time, includeFailure bool) error {
	edges := []v39.CommonEdge{
		epic7Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic7Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic7Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic7Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		epic7Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		epic7Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
		epic7Edge("task_artifact", v39.EdgeProduced, ids.task, firstArtifact, createdAt),
		epic7Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic7Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic7Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	for _, trial := range trials {
		for _, artifactID := range []string{"art_epic7_issue_" + trial.TrialID, "art_epic7_proposal_" + trial.TrialID, "art_epic7_proof_" + trial.TrialID, "art_epic7_patch_" + trial.TrialID, "art_epic7_pr_body_" + trial.TrialID, "art_epic7_branch_plan_" + trial.TrialID, "art_epic7_validation_plan_" + trial.TrialID} {
			edges = append(edges, epic7Edge("task_"+artifactID, v39.EdgeProduced, ids.task, artifactID, createdAt))
		}
		edges = append(edges, epic7Edge("plan_proposal_"+trial.TrialID, v39.EdgeProduced, ids.planningProposal, "art_epic7_proposal_"+trial.TrialID, createdAt))
		for index := range trial.Proposal.ChangedFileIntent {
			codeChangeID := epic7CodeChangeID(trial.TrialID, index)
			edges = append(edges, epic7Edge("change_patch_"+trial.TrialID+"_"+fmt.Sprint(index), v39.EdgeModified, codeChangeID, "art_epic7_patch_"+trial.TrialID, createdAt))
		}
		if trial.RepairEvidenceRef != "" {
			edges = append(edges, epic7Edge("task_repair_"+trial.TrialID, v39.EdgeProduced, ids.task, "art_epic7_repair_"+trial.TrialID, createdAt))
		}
		if trial.RollbackEvidenceRef != "" {
			edges = append(edges, epic7Edge("task_rollback_"+trial.TrialID, v39.EdgeProduced, ids.task, "art_epic7_rollback_"+trial.TrialID, createdAt))
		}
		if trial.HumanReviewEvidenceRef != "" {
			humanReviewArtifact := "art_epic7_human_review_" + trial.TrialID
			humanReviewRecord := "review_epic7_" + trial.TrialID
			edges = append(edges,
				epic7Edge("task_human_review_artifact_"+trial.TrialID, v39.EdgeProduced, ids.task, humanReviewArtifact, createdAt),
				epic7Edge("proposal_human_review_"+trial.TrialID, v39.EdgeApprovedBy, "art_epic7_proposal_"+trial.TrialID, humanReviewRecord, createdAt),
				epic7Edge("human_review_artifact_"+trial.TrialID, v39.EdgeProduced, humanReviewRecord, humanReviewArtifact, createdAt),
			)
		}
		if trial.MultiRepoAuthority != nil {
			grant := trial.MultiRepoAuthority
			edges = append(edges,
				epic7Edge("invoke_multi_repo_auth_"+trial.TrialID, v39.EdgeRequestedAuthority, ids.actorInvocation, grant.AuthorityRequestID, createdAt),
				epic7Edge("multi_repo_auth_decision_"+trial.TrialID, v39.EdgeDecidedBy, grant.AuthorityRequestID, grant.AuthorityDecisionID, createdAt),
				epic7Edge("multi_repo_auth_human_"+trial.TrialID, v39.EdgeApprovedBy, grant.AuthorityRequestID, grant.HumanApprovalID, createdAt),
			)
		}
		for _, action := range trial.AuthorityBoundary {
			edges = append(edges,
				epic7Edge("invoke_auth_"+trial.TrialID+"_"+epic7ActionSlug(action.Action), v39.EdgeRequestedAuthority, ids.actorInvocation, action.AuthorityRequestID, createdAt),
				epic7Edge("auth_decision_"+trial.TrialID+"_"+epic7ActionSlug(action.Action), v39.EdgeDecidedBy, action.AuthorityRequestID, action.AuthorityDecisionID, createdAt),
				epic7Edge("auth_human_"+trial.TrialID+"_"+epic7ActionSlug(action.Action), v39.EdgeApprovedBy, action.AuthorityRequestID, action.HumanApprovalID, createdAt),
			)
		}
	}
	if includeFailure {
		edges = append(edges, epic7Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic7BuildProjection(ids epic7FixtureIDs, trials []Epic7TrialEvidence, validation Epic7GateHValidation, graphRun epic7GraphRun) Epic7IssueToPRProjection {
	audit := graphRun.AuditReport
	auditEvidence := Epic7AuditEvidence{}
	if audit != nil {
		auditEvidence = Epic7AuditEvidence{ID: audit.CommonNode.ID, TargetType: audit.TargetType, TargetID: audit.TargetID, Status: statusString(audit.CommonNode.Status), TraceScore: audit.TraceScore, MissingLinks: append([]string(nil), audit.MissingLinks...)}
	}
	projection := Epic7IssueToPRProjection{
		GeneratedAt:     epic7FixtureTime().Format(time.RFC3339),
		Source:          "work-epic7-issue-to-pr-proposal-fixture",
		Mode:            Epic7IssueToPRLocalProposalEvidence,
		Trials:          append([]Epic7TrialEvidence(nil), trials...),
		GateHValidation: validation,
		AuditReport:     auditEvidence,
		ProofOfWorkPacket: Epic7ProofOfWorkAggregate{
			ID:               ids.proofPacket,
			Status:           validation.Status,
			Summary:          "Epic 7 Gate H aggregate proof: five local issue-to-PR proposal trials remain proposed-only.",
			TrialRefs:        epic7TrialIDs(trials),
			ForbiddenActions: epic7AggregateForbiddenActions(trials),
			ResidualRisks: []Epic7ProofOfWorkItem{
				{Label: "R-001", Status: "excluded", Summary: "No runner/worktree protected execution or branch push is performed."},
				{Label: "R-002", Status: "excluded", Summary: "No protected side effects, live PRs, merge, deploy, or ExecutionReceipt production path is recorded."},
				{Label: "R-003", Status: "excluded", Summary: "No PolicyEngineAdapterDecision or policy-bundle evidence is used."},
			},
			EventGraphRefs: []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeGateResult, ids.gateResult), egRef(v39.TypeAuditReport, ids.auditReport)},
		},
	}
	if validation.Status != "pass" {
		projection.Errors = append([]string(nil), validation.Missing...)
	}
	return projection
}

func epic7TrialIDs(trials []Epic7TrialEvidence) []string {
	out := make([]string, 0, len(trials))
	for _, trial := range trials {
		out = append(out, trial.TrialID)
	}
	return out
}

func epic7ProposalArtifactIDs(trials []Epic7TrialEvidence) []string {
	out := make([]string, 0, len(trials))
	for _, trial := range trials {
		out = append(out, "art_epic7_proposal_"+trial.TrialID)
	}
	return out
}

func epic7ProofArtifactIDs(trials []Epic7TrialEvidence) []string {
	out := make([]string, 0, len(trials))
	for _, trial := range trials {
		out = append(out, "art_epic7_proof_"+trial.TrialID)
	}
	return out
}

func epic7FirstArtifactID(trials []Epic7TrialEvidence) string {
	if len(trials) == 0 {
		return "art_epic7_empty"
	}
	return "art_epic7_issue_" + trials[0].TrialID
}

func epic7AggregateForbiddenActions(trials []Epic7TrialEvidence) []Epic7ProtectedActionRef {
	if len(trials) == 0 {
		return nil
	}
	return epic7ForbiddenActionRefs(trials[0].AuthorityBoundary)
}

func epic7RuntimeChangedFiles(trials []Epic7TrialEvidence) []string {
	var out []string
	for _, trial := range trials {
		if !trial.Proposal.Applied {
			continue
		}
		for _, intent := range trial.Proposal.ChangedFileIntent {
			out = append(out, intent.Repo+":"+intent.Path)
		}
	}
	return out
}

func epic7ForbiddenActionRefs(actions []Epic7ProtectedActionRef) []Epic7ProtectedActionRef {
	var out []Epic7ProtectedActionRef
	for _, action := range actions {
		if action.Action != Epic7ActionPullRequestPropose {
			out = append(out, action)
		}
	}
	return out
}

func epic7ForbiddenActionsSeparated(actions []Epic7ProtectedActionRef) bool {
	seen := map[Epic7ProtectedAction]bool{}
	for _, action := range actions {
		seen[action.Action] = true
		if action.Action != Epic7ActionPullRequestPropose && action.Status == "completed" {
			return false
		}
	}
	for _, action := range epic7ProtectedActions() {
		if !seen[action] {
			return false
		}
	}
	return true
}

func epic7ProtectedActions() []Epic7ProtectedAction {
	return []Epic7ProtectedAction{Epic7ActionPullRequestPropose, Epic7ActionPullRequestCreate, Epic7ActionBranchPush, Epic7ActionDefaultBranchPush, Epic7ActionPullRequestMerge, Epic7ActionProductionDeploy, Epic7ActionProtectedExecutionRun, Epic7ActionCapabilityActivate}
}

func epic7ActionCompleted(action Epic7ProtectedAction, completed []Epic7ProtectedAction) bool {
	for _, item := range completed {
		if item == action {
			return true
		}
	}
	return false
}

func epic7DefinitionByID(id string) epic7TrialDefinition {
	for _, def := range epic7TrialDefinitions() {
		if def.id == id {
			return def
		}
	}
	return epic7TrialDefinition{id: id}
}

func epic7CommandLog(trials []Epic7TrialEvidence, opts Epic7IssueToPROptions) []string {
	log := []string{"0:load_issue_fixtures:succeeded", "1:write_local_proposal_packets:succeeded", "2:write_proof_of_work_packets:succeeded"}
	for _, action := range epic7ProtectedActions() {
		if action == Epic7ActionPullRequestPropose {
			log = append(log, "proposal:"+string(action)+":recorded")
			continue
		}
		status := "denied"
		if epic7ActionCompleted(action, opts.CompletedForbiddenActions) {
			status = "forbidden_completed_evidence"
		}
		log = append(log, string(action)+":"+status)
	}
	if opts.RecordExecutionReceipt {
		log = append(log, "ExecutionReceipt:blocked_forbidden_test_evidence")
	}
	for _, trial := range trials {
		log = append(log, "trial:"+trial.TrialID+":"+trial.Status)
	}
	return log
}

func epic7Patch(def epic7TrialDefinition) string {
	var b strings.Builder
	for _, intent := range def.intents {
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", intent.Path, intent.Path)
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", intent.Path, intent.Path)
		fmt.Fprintf(&b, "@@ proposed-only @@\n+%s\n", intent.Summary)
	}
	return b.String()
}

func epic7PRBody(def epic7TrialDefinition) string {
	return strings.Join([]string{
		"## Proposed Scope",
		def.body,
		"",
		"## Authority Boundary",
		"This is a local proposal packet only. It does not create a live PR, push a branch, merge, deploy, run protected execution, activate a capability, or mutate a repository.",
		"",
		"## Validation",
		"- proposed-only packet review",
		"- focused Gate H fixture tests",
	}, "\n")
}

func epic7ValidationPlan(def epic7TrialDefinition) string {
	return strings.Join([]string{
		"# Validation Plan",
		"",
		"- Verify issue fixture: " + def.id,
		"- Verify proposed PR title/body/branch.",
		"- Verify changed-file intent and proposed diff refs.",
		"- Verify no live GitHub API call, branch push, merge, deploy, protected execution, or ExecutionReceipt.",
	}, "\n") + "\n"
}

func epic7WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return epic7WriteFile(path, string(append(raw, '\n')))
}

func epic7WriteFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func epic7AppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func epic7Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{ID: id, Type: typ, CreatedAt: epic7FixtureTime(), CreatedBy: epic7FixtureActorID, Status: &status, IdempotencyKey: "idem_" + id, CorrelationID: "corr_epic7_issue_pr_autonomy"}
}

func epic7Edge(suffix, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic7_" + suffix + "_" + from + "_" + to
	return v39.CommonEdge{ID: id, Type: typ, FromID: from, ToID: to, CreatedAt: createdAt, CreatedBy: epic7FixtureActorID, CorrelationID: "corr_epic7_issue_pr_autonomy", IdempotencyKey: "idem_" + id}
}

func epic7FixtureTime() time.Time {
	return time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC)
}

func epic7Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func epic7CodeChangeStatus(trial Epic7TrialEvidence) string {
	if trial.Proposal.Applied {
		return "applied"
	}
	return "proposed"
}

func epic7CodeChangeID(trialID string, index int) string {
	if index == 0 {
		return "change_epic7_" + trialID
	}
	return fmt.Sprintf("change_epic7_%s_%d", trialID, index+1)
}

func epic7CertificationID(cert *v39.Certification) string {
	if cert == nil {
		return ""
	}
	return cert.CommonNode.ID
}

func epic7ShortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func epic7ActionSlug(action Epic7ProtectedAction) string {
	slug := strings.NewReplacer(".", "_", "-", "_").Replace(string(action))
	return strings.TrimSpace(slug)
}

func epic7ProposedCommand(action Epic7ProtectedAction) string {
	switch action {
	case Epic7ActionPullRequestPropose:
		return "write local proposal packet"
	case Epic7ActionPullRequestCreate:
		return "gh pr create"
	case Epic7ActionBranchPush:
		return "git push origin proposal-branch"
	case Epic7ActionDefaultBranchPush:
		return "git push origin main"
	case Epic7ActionPullRequestMerge:
		return "gh pr merge"
	case Epic7ActionProductionDeploy:
		return "deploy"
	case Epic7ActionProtectedExecutionRun:
		return "protected_execution.run"
	case Epic7ActionCapabilityActivate:
		return "capability.activate"
	default:
		return string(action)
	}
}

func epic7AuthorityStatus(action Epic7ProtectedActionRef) string {
	if action.Status == "completed" {
		return "forbidden"
	}
	if action.Action == Epic7ActionPullRequestPropose {
		return "approved"
	}
	return "review_required"
}

func epic7AuthorityDecision(action Epic7ProtectedActionRef) string {
	if action.Status == "completed" {
		return "Forbidden"
	}
	return "ApprovalRequired"
}

func epic7HumanApprovalStatus(action Epic7ProtectedActionRef) string {
	if action.Status == "completed" {
		return "denied"
	}
	if action.Action == Epic7ActionPullRequestPropose {
		return "approved"
	}
	return "more_evidence_required"
}

func epic7HumanApprovalDecision(action Epic7ProtectedActionRef) string {
	if action.Status == "completed" {
		return "denied"
	}
	if action.Action == Epic7ActionPullRequestPropose {
		return "approved"
	}
	return "more_evidence_required"
}

func boolStatus(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

func boolArtifactStatus(ok bool) string {
	if ok {
		return "verified"
	}
	return "rejected"
}
