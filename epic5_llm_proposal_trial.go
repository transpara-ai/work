package work

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	Epic5LLMProposalReviewOnly        Epic5LLMProposalMode = "review_only"
	Epic5LLMProposalMissingInvocation Epic5LLMProposalMode = "missing_invocation"
	Epic5LLMProposalAppliedPatch      Epic5LLMProposalMode = "applied_patch"
)

const (
	epic5FixtureActorID        = "act_epic5_recorded_llm"
	epic5FixtureHumanActorID   = "act_epic5_human_reviewer"
	epic5FixtureTimeRFC        = "2026-06-01T18:45:00Z"
	epic5KnowledgeSourceRef    = "knowledge:dark-factory/v3.9/implementation/epics/epic-05-gate-f-bounded-llm-proposal-trial/01-work-llm-proposal-implementation-authorization-v3.9.md"
	epic5ProposalTargetPath    = "docs/designs/epic5-llm-proposal-output.md"
	epic5RecordedProviderLabel = "recorded_fixture_provider"
	epic5RecordedModelLabel    = "recorded-gate-f-proposal-model"
)

// Epic5LLMProposalMode selects the authorized happy path or a negative Gate F path.
type Epic5LLMProposalMode string

// Epic5LLMProposalOptions keeps the fixture local and bounded to caller-provided storage.
type Epic5LLMProposalOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Mode           Epic5LLMProposalMode
}

// Epic5LLMProposalRun is the local evidence packet for the bounded Gate F trial.
type Epic5LLMProposalRun struct {
	Mode                    Epic5LLMProposalMode
	WorkTask                Task
	WorkProjection          TaskProjection
	EventGraph              *v39.InMemoryStore
	FactoryOrderID          string
	RequirementID           string
	AcceptanceCriterionID   string
	TaskID                  string
	ActorInvocationID       string
	PromptArtifactID        string
	ResponseArtifactID      string
	ProposalArtifactID      string
	ProposedPatchArtifactID string
	CodeChangeID            string
	CapabilityArtifactID    string
	KnowledgeReferenceID    string
	TestRunID               string
	GateResultID            string
	FailureID               string
	ReleaseCandidateID      string
	CertificationID         string
	RejectionID             string
	AuthorityRequestID      string
	AuthorityDecisionID     string
	HumanApprovalID         string
	AuditReportID           string
	PromptHash              string
	ResponseHash            string
	InputContractHash       string
	OutputContractHash      string
	ProposalApplied         bool
	TraceCompleteness       v39.TraceCompletenessGateResult
	CapabilityUsagePath     v39.RequiredPath
	KnowledgePath           v39.RequiredPath
	GateFValidation         Epic5GateFValidation
	Certification           *v39.Certification
	Rejection               *v39.Rejection
	HumanApproval           *v39.HumanApproval
	AuditReport             *v39.AuditReport
	Projection              Epic5LLMProposalProjection
	LocalArtifacts          Epic5LocalArtifacts
}

type Epic5LocalArtifacts struct {
	PromptPath   string
	ResponsePath string
	ProposalPath string
	PatchPath    string
}

type Epic5LLMProposalProjection struct {
	GeneratedAt       string                      `json:"generated_at"`
	Source            string                      `json:"source"`
	Mode              Epic5LLMProposalMode        `json:"mode"`
	LLMInvocation     *Epic5LLMInvocationEvidence `json:"llm_invocation,omitempty"`
	Proposal          Epic5ProposalEvidence       `json:"proposal"`
	Authority         Epic5AuthorityEvidence      `json:"authority"`
	Capability        Epic5InfluenceEvidence      `json:"capability"`
	Knowledge         Epic5InfluenceEvidence      `json:"knowledge"`
	GateEvidence      []Epic5GateEvidence         `json:"gate_evidence"`
	AuditReport       Epic5AuditEvidence          `json:"audit_report"`
	ProofOfWorkPacket Epic5ProofOfWorkPacket      `json:"proof_of_work_packet"`
	NegativeEvidence  []Epic5ProofOfWorkItem      `json:"negative_evidence"`
	Errors            []string                    `json:"errors,omitempty"`
}

type Epic5LLMInvocationEvidence struct {
	ActorInvocationID   string `json:"actor_invocation_id"`
	ActorID             string `json:"actor_id"`
	ModelLabel          string `json:"model_label"`
	ProviderLabel       string `json:"provider_label"`
	PromptHash          string `json:"prompt_hash"`
	ResponseHash        string `json:"response_hash"`
	InputContractHash   string `json:"input_contract_hash"`
	OutputContractHash  string `json:"output_contract_hash"`
	RecordedAt          string `json:"recorded_at"`
	PromptArtifactRef   string `json:"prompt_artifact_ref"`
	ResponseArtifactRef string `json:"response_artifact_ref"`
}

type Epic5ProposalEvidence struct {
	ProposalArtifactRef string `json:"proposal_artifact_ref"`
	CodeChangeID        string `json:"code_change_id,omitempty"`
	TargetRepo          string `json:"target_repo"`
	TargetPath          string `json:"target_path"`
	ProposedDiffRef     string `json:"proposed_diff_ref"`
	ProposedOnly        bool   `json:"proposed_only"`
	Applied             bool   `json:"applied"`
	Summary             string `json:"summary"`
}

type Epic5AuthorityEvidence struct {
	AuthorityRequestID  string   `json:"authority_request_id"`
	AuthorityDecisionID string   `json:"authority_decision_id"`
	HumanApprovalID     string   `json:"human_approval_id"`
	RequestedAction     string   `json:"requested_action"`
	Decision            string   `json:"decision"`
	HumanDecision       string   `json:"human_decision"`
	Scope               []string `json:"scope"`
	Summary             string   `json:"summary"`
}

type Epic5InfluenceEvidence struct {
	ID             string   `json:"id"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	EventGraphRefs []string `json:"event_graph_refs"`
}

type Epic5GateEvidence struct {
	GateName     string   `json:"gate_name"`
	Status       string   `json:"status"`
	GateResultID string   `json:"gate_result_id"`
	EvidenceRefs []string `json:"evidence_refs"`
	MissingRefs  []string `json:"missing_refs"`
}

type Epic5GateFValidation struct {
	Status  string   `json:"status"`
	Missing []string `json:"missing,omitempty"`
}

type Epic5AuditEvidence struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type Epic5ProofOfWorkPacket struct {
	ID                string                 `json:"id"`
	Status            string                 `json:"status"`
	Summary           string                 `json:"summary"`
	LLMContribution   *Epic5ProofOfWorkItem  `json:"llm_contribution,omitempty"`
	Proposal          Epic5ProofOfWorkItem   `json:"proposal"`
	Validation        Epic5ProofOfWorkItem   `json:"validation"`
	ReviewEvidence    Epic5ProofOfWorkItem   `json:"review_evidence"`
	AuditEvidence     Epic5ProofOfWorkItem   `json:"audit_evidence"`
	AuthorityDecision Epic5ProofOfWorkItem   `json:"authority_decision"`
	NonExecutionProof []Epic5ProofOfWorkItem `json:"non_execution_proof"`
	InfluenceEvidence []Epic5ProofOfWorkItem `json:"influence_evidence"`
	EventGraphRefs    []string               `json:"event_graph_refs"`
}

type Epic5ProofOfWorkItem struct {
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	ArtifactRef    string   `json:"artifact_ref"`
	EventGraphRefs []string `json:"event_graph_refs"`
}

// RunEpic5BoundedLLMProposalTrial executes the authorized recorded-LLM proposal fixture.
func RunEpic5BoundedLLMProposalTrial(ts *TaskStore, opts Epic5LLMProposalOptions) (Epic5LLMProposalRun, error) {
	if ts == nil {
		return Epic5LLMProposalRun{}, errors.New("task store is required")
	}
	if opts.Source.IsZero() {
		return Epic5LLMProposalRun{}, errors.New("source actor is required")
	}
	if opts.ConversationID.Value() == "" {
		return Epic5LLMProposalRun{}, errors.New("conversation ID is required")
	}
	if strings.TrimSpace(opts.WorkingDir) == "" {
		return Epic5LLMProposalRun{}, errors.New("working directory is required")
	}
	if opts.Mode == "" {
		opts.Mode = Epic5LLMProposalReviewOnly
	}
	if opts.Mode != Epic5LLMProposalReviewOnly && opts.Mode != Epic5LLMProposalMissingInvocation && opts.Mode != Epic5LLMProposalAppliedPatch {
		return Epic5LLMProposalRun{}, fmt.Errorf("unsupported Epic 5 fixture mode %q", opts.Mode)
	}

	ids := epic5IDs(opts.Mode)
	task, err := ts.CreateV39(opts.Source, TaskCreateOptions{
		Title:                  "Epic 5 Bounded LLM Proposal Trial",
		Description:            "Run the bounded recorded-LLM proposal fixture without applying the proposed diff.",
		CanonicalTaskID:        ids.task,
		FactoryOrderID:         ids.factoryOrder,
		RequirementIDs:         []string{ids.requirement},
		AcceptanceCriterionIDs: []string{ids.acceptanceCriterion},
		Cell:                   "cell_epic5_llm_proposal",
		RiskClass:              "medium",
		ExpectedOutputs:        []string{"prompt.md", "response.md", "proposal.md", "proposed.patch"},
	}, opts.Causes, opts.ConversationID)
	if err != nil {
		return Epic5LLMProposalRun{}, err
	}
	causes := append(append([]types.EventID(nil), opts.Causes...), task.ID)
	for _, status := range []TaskStatus{StatusReady, StatusRunning} {
		if err := ts.TransitionTask(opts.Source, task.ID, status, "Epic 5 recorded LLM proposal fixture lifecycle", nil, causes, opts.ConversationID); err != nil {
			return Epic5LLMProposalRun{}, err
		}
	}

	transcript := epic5RecordedTranscript()
	localArtifacts, err := epic5WriteLocalArtifacts(opts.WorkingDir, transcript)
	if err != nil {
		return Epic5LLMProposalRun{}, err
	}
	graph, graphRun, err := epic5RecordEventGraph(ids, opts.Mode, transcript)
	if err != nil {
		return Epic5LLMProposalRun{}, err
	}

	if err := ts.AttachVerificationEvidence(opts.Source, task.ID, VerificationEvidence{
		TestCaseIDs:   []string{ids.testCase},
		TestRunIDs:    []string{ids.testRun},
		GateResultIDs: []string{ids.gateResult},
	}, "Epic 5 Gate F recorded LLM proposal evidence attached", causes, opts.ConversationID); err != nil {
		return Epic5LLMProposalRun{}, err
	}
	if graphRun.FailureID != "" {
		if err := ts.AttachFailureRepairReferences(opts.Source, task.ID, FailureRepairReferences{
			FailureIDs: []string{graphRun.FailureID},
		}, "Epic 5 negative Gate F fixture failure attached", causes, opts.ConversationID); err != nil {
			return Epic5LLMProposalRun{}, err
		}
	}
	if err := ts.TransitionTask(opts.Source, task.ID, StatusVerified, "Epic 5 Gate F evidence recorded", []string{ids.testRun, ids.gateResult}, causes, opts.ConversationID); err != nil {
		return Epic5LLMProposalRun{}, err
	}
	if opts.Mode == Epic5LLMProposalReviewOnly {
		if err := ts.TransitionTask(opts.Source, task.ID, StatusCertified, "Epic 5 recorded LLM proposal trial certified for review-only evidence", []string{graphRun.DecisionID}, causes, opts.ConversationID); err != nil {
			return Epic5LLMProposalRun{}, err
		}
	} else if err := ts.RejectTask(opts.Source, task.ID, "Epic 5 negative Gate F fixture rejected", []string{ids.gateResult, graphRun.FailureID}, causes, opts.ConversationID); err != nil {
		return Epic5LLMProposalRun{}, err
	}

	projection, err := epic5BuildProjection(graph, ids, opts.Mode, transcript, graphRun)
	if err != nil {
		return Epic5LLMProposalRun{}, err
	}
	workProjection, err := ts.ProjectTask(task.ID)
	if err != nil {
		return Epic5LLMProposalRun{}, err
	}

	return Epic5LLMProposalRun{
		Mode:                    opts.Mode,
		WorkTask:                task,
		WorkProjection:          workProjection,
		EventGraph:              graph,
		FactoryOrderID:          ids.factoryOrder,
		RequirementID:           ids.requirement,
		AcceptanceCriterionID:   ids.acceptanceCriterion,
		TaskID:                  ids.task,
		ActorInvocationID:       epic5ActorInvocationID(ids, opts.Mode),
		PromptArtifactID:        ids.promptArtifact,
		ResponseArtifactID:      ids.responseArtifact,
		ProposalArtifactID:      ids.proposalArtifact,
		ProposedPatchArtifactID: ids.patchArtifact,
		CodeChangeID:            epic5CodeChangeID(ids, opts.Mode),
		CapabilityArtifactID:    ids.capabilityArtifact,
		KnowledgeReferenceID:    ids.knowledgeReference,
		TestRunID:               ids.testRun,
		GateResultID:            ids.gateResult,
		FailureID:               graphRun.FailureID,
		ReleaseCandidateID:      ids.releaseCandidate,
		CertificationID:         epic5CertificationID(graphRun.Certification),
		RejectionID:             graphRun.RejectionID,
		AuthorityRequestID:      ids.authorityRequest,
		AuthorityDecisionID:     ids.authorityDecision,
		HumanApprovalID:         ids.humanApproval,
		AuditReportID:           ids.auditReport,
		PromptHash:              transcript.PromptHash,
		ResponseHash:            transcript.ResponseHash,
		InputContractHash:       transcript.InputContractHash,
		OutputContractHash:      transcript.OutputContractHash,
		ProposalApplied:         opts.Mode == Epic5LLMProposalAppliedPatch,
		TraceCompleteness:       graphRun.Trace,
		CapabilityUsagePath:     graphRun.CapabilityUsagePath,
		KnowledgePath:           graphRun.KnowledgePath,
		GateFValidation:         graphRun.GateFValidation,
		Certification:           graphRun.Certification,
		Rejection:               graphRun.Rejection,
		HumanApproval:           graphRun.HumanApproval,
		AuditReport:             graphRun.AuditReport,
		Projection:              projection,
		LocalArtifacts:          localArtifacts,
	}, nil
}

func (p Epic5LLMProposalProjection) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

type epic5FixtureIDs struct {
	suffix              string
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	llmActorIdentity    string
	humanActorIdentity  string
	actorInvocation     string
	capabilityArtifact  string
	knowledgeReference  string
	promptArtifact      string
	responseArtifact    string
	proposalArtifact    string
	patchArtifact       string
	planningProposal    string
	codeChange          string
	authorityRequest    string
	authorityDecision   string
	humanApproval       string
	runtimeEnvelope     string
	runtimeResult       string
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

type epic5RecordedTranscriptEvidence struct {
	Prompt             string
	Response           string
	Proposal           string
	ProposedPatch      string
	PromptHash         string
	ResponseHash       string
	ProposalHash       string
	ProposedPatchHash  string
	InputContractHash  string
	OutputContractHash string
}

type epic5GraphRun struct {
	DecisionID          string
	RejectionID         string
	FailureID           string
	Trace               v39.TraceCompletenessGateResult
	CapabilityUsagePath v39.RequiredPath
	KnowledgePath       v39.RequiredPath
	GateFValidation     Epic5GateFValidation
	Certification       *v39.Certification
	Rejection           *v39.Rejection
	HumanApproval       *v39.HumanApproval
	AuditReport         *v39.AuditReport
}

type epic5GraphStatuses struct {
	FactoryStatus string
	TaskStatus    string
	TestRunStatus string
	GateStatus    string
}

type epic5GraphEvidence struct {
	Graph               *v39.InMemoryStore
	Trace               v39.TraceCompletenessGateResult
	CapabilityUsagePath v39.RequiredPath
	KnowledgePath       v39.RequiredPath
}

func epic5IDs(mode Epic5LLMProposalMode) epic5FixtureIDs {
	suffix := string(mode)
	return epic5FixtureIDs{
		suffix:              suffix,
		factoryOrder:        "fo_epic5_llm_proposal_" + suffix,
		requirement:         "req_epic5_llm_proposal_" + suffix,
		acceptanceCriterion: "ac_epic5_llm_proposal_" + suffix,
		task:                "tsk_epic5_llm_proposal_" + suffix,
		llmActorIdentity:    "actor_identity_epic5_llm_" + suffix,
		humanActorIdentity:  "actor_identity_epic5_human_" + suffix,
		actorInvocation:     "invoke_epic5_llm_proposal_" + suffix,
		capabilityArtifact:  "cap_art_epic5_llm_prompt",
		knowledgeReference:  "know_ref_epic5_authorization_" + suffix,
		promptArtifact:      "art_epic5_llm_prompt_" + suffix,
		responseArtifact:    "art_epic5_llm_response_" + suffix,
		proposalArtifact:    "art_epic5_llm_proposal_" + suffix,
		patchArtifact:       "art_epic5_llm_proposed_patch_" + suffix,
		planningProposal:    "plan_epic5_llm_proposal_" + suffix,
		codeChange:          "change_epic5_llm_proposed_patch_" + suffix,
		authorityRequest:    "auth_req_epic5_llm_proposal_" + suffix,
		authorityDecision:   "auth_dec_epic5_llm_proposal_" + suffix,
		humanApproval:       "human_app_epic5_llm_proposal_" + suffix,
		runtimeEnvelope:     "env_epic5_llm_proposal_" + suffix,
		runtimeResult:       "rr_epic5_llm_proposal_" + suffix,
		testCase:            "tc_epic5_llm_proposal_" + suffix,
		testRun:             "tr_epic5_llm_proposal_" + suffix,
		gateResult:          "gate_epic5_llm_proposal_" + suffix,
		failure:             "fail_epic5_llm_proposal_" + suffix,
		factoryRuntime:      "frv_epic5_llm_proposal_" + suffix,
		releaseCandidate:    "rc_epic5_llm_proposal_" + suffix,
		certification:       "cert_epic5_llm_proposal_" + suffix,
		rejection:           "rej_epic5_llm_proposal_" + suffix,
		auditReport:         "aud_epic5_llm_proposal_" + suffix,
		proofPacket:         "pow_epic5_llm_proposal_" + suffix,
	}
}

func epic5RecordedTranscript() epic5RecordedTranscriptEvidence {
	prompt := strings.Join([]string{
		"System: Produce a review-only proposal for Dark Factory Gate F.",
		"Scope: transpara-ai/work only; do not apply a diff or mutate a repository.",
		"Target artifact: docs/designs/epic5-llm-proposal-output.md.",
		"Required output: concise proposal plus a proposed patch.",
	}, "\n")
	response := strings.Join([]string{
		"Recorded LLM response:",
		"Create a short design note that explains the recorded proposal trial, evidence links, and human review boundary.",
		"Protected actions must remain proposed-only; no push, merge, deploy, secret access, or cross-repo mutation is allowed.",
	}, "\n")
	proposal := "Proposal: add a review-only design note for the bounded Gate F recorded LLM proposal trial."
	patch := strings.Join([]string{
		"diff --git a/docs/designs/epic5-llm-proposal-output.md b/docs/designs/epic5-llm-proposal-output.md",
		"new file mode 100644",
		"--- /dev/null",
		"+++ b/docs/designs/epic5-llm-proposal-output.md",
		"@@ -0,0 +1,4 @@",
		"+# Epic 5 LLM Proposal Output",
		"+",
		"+This is a proposed review-only design note generated by the recorded Gate F fixture.",
		"+Human review remains the final authority; this patch is not applied by the fixture.",
		"",
	}, "\n")
	return epic5RecordedTranscriptEvidence{
		Prompt:             prompt,
		Response:           response,
		Proposal:           proposal,
		ProposedPatch:      patch,
		PromptHash:         epic5Hash(prompt),
		ResponseHash:       epic5Hash(response),
		ProposalHash:       epic5Hash(proposal),
		ProposedPatchHash:  epic5Hash(patch),
		InputContractHash:  epic5Hash(strings.Join([]string{"epic5-input-contract:v1", prompt, epic5ProposalTargetPath}, "\n")),
		OutputContractHash: epic5Hash(strings.Join([]string{"epic5-output-contract:v1", response, proposal, patch}, "\n")),
	}
}

func epic5WriteLocalArtifacts(dir string, transcript epic5RecordedTranscriptEvidence) (Epic5LocalArtifacts, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Epic5LocalArtifacts{}, err
	}
	files := map[string]string{
		"prompt.md":      transcript.Prompt,
		"response.md":    transcript.Response,
		"proposal.md":    transcript.Proposal,
		"proposed.patch": transcript.ProposedPatch,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			return Epic5LocalArtifacts{}, err
		}
	}
	return Epic5LocalArtifacts{
		PromptPath:   filepath.Join(dir, "prompt.md"),
		ResponsePath: filepath.Join(dir, "response.md"),
		ProposalPath: filepath.Join(dir, "proposal.md"),
		PatchPath:    filepath.Join(dir, "proposed.patch"),
	}, nil
}

func epic5RecordEventGraph(ids epic5FixtureIDs, mode Epic5LLMProposalMode, transcript epic5RecordedTranscriptEvidence) (*v39.InMemoryStore, epic5GraphRun, error) {
	hasInvocation := mode != Epic5LLMProposalMissingInvocation
	proposalApplied := mode == Epic5LLMProposalAppliedPatch

	preflight, err := epic5BuildEventGraphEvidence(ids, mode, transcript, epic5GraphStatuses{
		FactoryStatus: "verification",
		TaskStatus:    "verification_running",
		TestRunStatus: "skipped",
		GateStatus:    "skipped",
	})
	if err != nil {
		return nil, epic5GraphRun{}, err
	}
	gateValidation := epic5EvaluateGateF(preflight.Trace, preflight.CapabilityUsagePath, preflight.KnowledgePath, hasInvocation, proposalApplied)
	evidence, err := epic5BuildEventGraphEvidence(ids, mode, transcript, epic5StatusesFromGateFValidation(gateValidation))
	if err != nil {
		return nil, epic5GraphRun{}, err
	}
	graph := evidence.Graph
	gate, err := epic5GateResultFromGraph(graph, ids.gateResult)
	if err != nil {
		return nil, epic5GraphRun{}, err
	}
	if gate.CommonNode.Status == nil || *gate.CommonNode.Status != gateValidation.Status {
		return nil, epic5GraphRun{}, fmt.Errorf("gate F status %q does not match evaluated status %q", statusString(gate.CommonNode.Status), gateValidation.Status)
	}
	if mode == Epic5LLMProposalReviewOnly && gateValidation.Status != "pass" {
		return nil, epic5GraphRun{}, fmt.Errorf("%w: gate F validation incomplete: %v", v39.ErrRequiredPathMissing, gateValidation.Missing)
	}
	if mode != Epic5LLMProposalReviewOnly && gateValidation.Status != "fail" {
		return nil, epic5GraphRun{}, errors.New("negative Epic 5 fixture unexpectedly passed Gate F validation")
	}
	approval, err := epic5HumanApprovalFromGraph(graph, ids.humanApproval)
	if err != nil {
		return nil, epic5GraphRun{}, err
	}

	if mode == Epic5LLMProposalReviewOnly {
		cert, err := graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: epic5Common(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: epic5FixtureHumanActorID, Reason: "Gate F recorded LLM proposal evidence is complete for review-only certification; the proposed patch remains unapplied.", EvidenceRefs: []string{ids.gateResult, ids.humanApproval, ids.authorityDecision, ids.codeChange}})
		if err != nil {
			return nil, epic5GraphRun{}, err
		}
		audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic5Common(ids.auditReport, v39.TypeAuditReport, "complete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, epic5GraphRun{}, err
		}
		return graph, epic5GraphRun{DecisionID: cert.CommonNode.ID, Trace: evidence.Trace, CapabilityUsagePath: evidence.CapabilityUsagePath, KnowledgePath: evidence.KnowledgePath, GateFValidation: gateValidation, Certification: cert, HumanApproval: approval, AuditReport: audit}, nil
	}

	rejection, err := graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: epic5Common(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: epic5FixtureHumanActorID, Reason: epic5FailureSummary(mode), EvidenceRefs: []string{ids.gateResult, ids.failure}})
	if err != nil {
		return nil, epic5GraphRun{}, err
	}
	audit, err := graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: epic5Common(ids.auditReport, v39.TypeAuditReport, "incomplete"), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
	if err != nil {
		return nil, epic5GraphRun{}, err
	}
	return graph, epic5GraphRun{DecisionID: rejection.CommonNode.ID, RejectionID: rejection.CommonNode.ID, FailureID: ids.failure, Trace: evidence.Trace, CapabilityUsagePath: evidence.CapabilityUsagePath, KnowledgePath: evidence.KnowledgePath, GateFValidation: gateValidation, Rejection: rejection, HumanApproval: approval, AuditReport: audit}, nil
}

func epic5BuildEventGraphEvidence(ids epic5FixtureIDs, mode Epic5LLMProposalMode, transcript epic5RecordedTranscriptEvidence, statuses epic5GraphStatuses) (epic5GraphEvidence, error) {
	graph := v39.NewInMemoryStore()
	createdAt := epic5FixtureTime()
	hasInvocation := mode != Epic5LLMProposalMissingInvocation

	taskCommon := epic5Common(ids.task, v39.TypeTask, statuses.TaskStatus)
	taskCommon.SourceRefs = []string{ids.capabilityArtifact, epic5KnowledgeSourceRef}

	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: epic5Common(ids.factoryOrder, v39.TypeFactoryOrder, statuses.FactoryStatus), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-83-gate-f-authorization", SourceIntentRef: "transpara-ai/docs#83", RiskClass: "medium", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: epic5Common(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Prove one recorded LLM proposal path with traceable influence, validation, proof-of-work display, and human authority.", Source: "explicit", RiskClass: "medium"},
		&v39.AcceptanceCriterion{CommonNode: epic5Common(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, "verified"), RequirementID: ids.requirement, Text: "Gate F passes only when a recorded LLM invocation, proposed-only diff, capability/knowledge influence, validation, audit, and human approval evidence are present.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "eventgraph_trace", OwnerRole: "maintainer", RiskClass: "medium"},
		&v39.Task{CommonNode: taskCommon, FactoryOrderID: &ids.factoryOrder, Cell: "cell_epic5_llm_proposal", State: statuses.TaskStatus, Priority: 1, RiskClass: "medium", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: epic5Common(ids.llmActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic5FixtureActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorIdentity{CommonNode: epic5Common(ids.humanActorIdentity, v39.TypeActorIdentity, "active"), ActorID: epic5FixtureHumanActorID, ActorType: "human", IdentityMode: "fixture"},
		&v39.CapabilityArtifact{CommonNode: epic5Common(ids.capabilityArtifact, v39.TypeCapabilityArtifact, "active"), ArtifactID: ids.capabilityArtifact, ArtifactType: "prompt_section", Name: "Epic 5 Gate F recorded LLM prompt", ArtifactVersion: "v1", SourceRepoOrOrigin: "transpara-ai/work", ContentHash: transcript.PromptHash, Owner: "work", RiskClass: "medium", ActivationScope: "fixture_only", EvalRefs: []string{ids.testCase}, HumanReviewRef: ids.humanApproval, RollbackRef: "not_applicable_review_only_fixture", UsageLoggingRequired: true},
		&v39.Artifact{CommonNode: epic5Common(ids.promptArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "document", Path: strPtr("fixture://epic5/prompt.md"), ContentHash: &transcript.PromptHash},
		&v39.Artifact{CommonNode: epic5Common(ids.responseArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "document", Path: strPtr("fixture://epic5/response.md"), ContentHash: &transcript.ResponseHash},
		&v39.Artifact{CommonNode: epic5Common(ids.proposalArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "document", Path: strPtr("fixture://epic5/proposal.md"), ContentHash: &transcript.ProposalHash},
		&v39.Artifact{CommonNode: epic5Common(ids.patchArtifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "code", Path: strPtr("fixture://epic5/proposed.patch"), ContentHash: &transcript.ProposedPatchHash},
		&v39.PlanningProposal{CommonNode: epic5Common(ids.planningProposal, v39.TypePlanningProposal, "proposed"), FactoryOrderID: ids.factoryOrder, FactoryOrderVersion: 1, Requirements: []string{ids.requirement}, AcceptanceCriteria: []string{ids.acceptanceCriterion}, Assumptions: []string{"Recorded fixture transcript stands in for the LLM call.", "The proposed patch is evidence only and is not applied."}, ArchitectureOptions: []string{"review_only_recorded_llm_proposal"}, RecommendedOptionID: strPtr("review_only_recorded_llm_proposal"), TaskDrafts: []string{ids.task}, RequiresHumanReview: true},
		&v39.AuthorityRequest{CommonNode: epic5Common(ids.authorityRequest, v39.TypeAuthorityRequest, "open"), ActorID: epic5FixtureActorID, ActorRole: "recorded_llm", Action: "repo.mutate.proposed_patch", TargetType: "artifact", TargetID: ids.patchArtifact, RiskClass: "medium", Reason: "The LLM proposed a patch; the fixture records it for review only and does not apply it.", ProposedCommand: strPtr("apply proposed.patch"), EvidenceRefs: []string{ids.proposalArtifact, ids.patchArtifact}},
		&v39.AuthorityDecision{CommonNode: epic5Common(ids.authorityDecision, v39.TypeAuthorityDecision, epic5AuthorityDecisionStatus(mode)), AuthorityRequestID: ids.authorityRequest, DeciderActorID: epic5FixtureHumanActorID, DeciderRole: "maintainer", Decision: epic5AuthorityDecision(mode), Reason: epic5AuthorityReason(mode), Scope: []string{"repo.review.proposed_patch"}, Conditions: []string{"do not apply patch", "do not push", "do not merge", "do not mutate another repo"}},
		&v39.HumanApproval{CommonNode: epic5Common(ids.humanApproval, v39.TypeHumanApproval, epic5HumanDecision(mode)), RequestRef: ids.authorityRequest, ApproverActorID: epic5FixtureHumanActorID, ApproverRole: "maintainer", Decision: epic5HumanDecision(mode), Reason: epic5HumanReason(mode)},
		&v39.TestCase{CommonNode: epic5Common(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Epic 5 recorded LLM proposal Gate F evidence", TestType: "unit", Path: strPtr("work/epic5_llm_proposal_trial_test.go")},
		&v39.TestRun{CommonNode: epic5Common(ids.testRun, v39.TypeTestRun, statuses.TestRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: epic5OptionalInvocation(ids, hasInvocation), Command: "go test ./..."},
		&v39.GateResult{CommonNode: epic5Common(ids.gateResult, v39.TypeGateResult, statuses.GateStatus), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_f_recorded_llm_proposal", EvidenceRefs: epic5GateEvidenceRefs(ids, hasInvocation)},
	}
	if hasInvocation {
		records = append(records,
			&v39.ActorInvocation{CommonNode: epic5Common(ids.actorInvocation, v39.TypeActorInvocation, "succeeded"), TaskID: ids.task, Runtime: "local", ActorID: epic5FixtureActorID, InputContractHash: transcript.InputContractHash, OutputContractHash: &transcript.OutputContractHash},
			&v39.CodeChange{CommonNode: epic5Common(ids.codeChange, v39.TypeCodeChange, epic5CodeChangeStatus(mode)), ArtifactID: ids.patchArtifact, ActorInvocationID: ids.actorInvocation, Repo: "transpara-ai/work", Path: epic5ProposalTargetPath, BeforeHash: strPtr("sha256:empty"), AfterHash: transcript.ProposedPatchHash, ChangeType: "create"},
			&v39.RuntimeEnvelope{CommonNode: epic5Common(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "recorded_llm_fixture", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: epic5FixtureActorID, AuthorityDecisionRef: ids.authorityDecision, AllowedFiles: []string{"prompt.md", "response.md", "proposal.md", "proposed.patch"}, DeniedFiles: []string{".git", "../", "secrets.env"}, AllowedCommands: []string{"record_prompt", "record_response", "record_proposed_patch"}, DeniedCommands: []string{"apply_patch", "git_push", "git_merge", "network_attempt", "secret_attempt"}, NetworkPolicy: "disabled", SecretsPolicy: "none", WorkingDirectory: "fixture://epic5-recorded-llm", Timeout: "1s", ResourceLimits: map[string]any{"max_files_changed": 0, "max_output_bytes": 8192}, ExpectedOutputs: []string{"prompt.md", "response.md", "proposal.md", "proposed.patch"}, OutputContract: map[string]any{"mode": "review_only_proposal"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "Task -> ActorInvocation", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"verify prompt/response hashes", "verify proposed-only authority"}, EnvelopeHash: transcript.InputContractHash},
			&v39.RuntimeResult{CommonNode: epic5Common(ids.runtimeResult, v39.TypeRuntimeResult, "recorded"), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "recorded_llm_fixture", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: "succeeded", ArtifactRefs: []string{ids.promptArtifact, ids.responseArtifact, ids.proposalArtifact, ids.patchArtifact}, ChangedFiles: []string{}, CommandLog: []string{"0:record_prompt:succeeded", "1:record_response:succeeded", "2:record_proposed_patch:succeeded", "3:apply_patch:denied"}, NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{ids.authorityDecision, ids.humanApproval}, PostRunValidationRefs: []string{ids.testRun}},
		)
	}
	if statuses.GateStatus == "fail" {
		records = append(records, &v39.Failure{CommonNode: epic5Common(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: epic5FailureClass(mode), Severity: "high", Summary: epic5FailureSummary(mode)})
	}
	if err := epic5AppendRecords(graph, records...); err != nil {
		return epic5GraphEvidence{}, err
	}
	if _, err := graph.RecordCapabilityUsage(ids.task, ids.capabilityArtifact, epic5Common("edge_epic5_used_capability_"+ids.suffix, v39.TypeCapabilityArtifact, "recorded")); err != nil {
		return epic5GraphEvidence{}, err
	}
	if _, err := graph.RecordKnowledgeReference(&v39.KnowledgeReference{AdvisoryReference: v39.AdvisoryReference{
		CommonNode:                   epic5Common(ids.knowledgeReference, v39.TypeKnowledgeReference, "recorded"),
		ReferenceCreatedAt:           createdAt,
		SourceSystem:                 "transpara-ai/docs",
		SourceRef:                    epic5KnowledgeSourceRef,
		SourceHashOrImmutableLocator: "sha256:docs-pr-83-merged-97ce7706e7047e829e3aea321fa8a1afad4f62e4",
		RetrievedAt:                  createdAt,
		UsedByActor:                  epic5FixtureActorID,
		UsedInTask:                   ids.task,
		InfluenceSummary:             "planning: Gate F authorization shaped the prompt, review criteria, and proposed-only boundary.",
		RiskScope:                    "medium",
		TrustLevel:                   "reviewed",
		FreshnessStatus:              "current",
		RedactionState:               "none",
	}}); err != nil {
		return epic5GraphEvidence{}, err
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: epic5Common(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-epic5-recorded-llm", CapabilityVersionRefs: []string{}, RuntimeRefs: []string{"work.recorded_llm_fixture@1"}}); err != nil {
		return epic5GraphEvidence{}, err
	}
	if err := epic5AppendEdges(graph, ids, createdAt, hasInvocation, statuses.GateStatus == "fail"); err != nil {
		return epic5GraphEvidence{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: epic5Common(ids.releaseCandidate, v39.TypeReleaseCandidate, statuses.FactoryStatus), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.patchArtifact}})
	if err != nil {
		return epic5GraphEvidence{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	capabilityPath, _ := graph.CapabilityUsageEvidencePath(rc.CommonNode.ID)
	knowledgePath, _ := graph.AdvisoryReferenceEvidencePath(rc.CommonNode.ID)
	if statuses.GateStatus == "pass" && traceErr != nil {
		return epic5GraphEvidence{}, traceErr
	}
	return epic5GraphEvidence{Graph: graph, Trace: trace, CapabilityUsagePath: capabilityPath, KnowledgePath: knowledgePath}, nil
}

func epic5EvaluateGateF(trace v39.TraceCompletenessGateResult, capabilityPath, knowledgePath v39.RequiredPath, hasInvocation, proposalApplied bool) Epic5GateFValidation {
	seen := map[string]bool{}
	var missing []string
	if !trace.Completed {
		missing = appendUniqueStrings(missing, trace.Missing, seen)
	}
	if !capabilityPath.Completed {
		missing = appendUniqueStrings(missing, prefixMissing("capability evidence", capabilityPath.Missing), seen)
	}
	if !knowledgePath.Completed {
		missing = appendUniqueStrings(missing, prefixMissing("knowledge evidence", knowledgePath.Missing), seen)
	}
	if !hasInvocation {
		missing = appendUniqueStrings(missing, []string{"recorded LLM ActorInvocation evidence"}, seen)
	}
	if proposalApplied {
		missing = appendUniqueStrings(missing, []string{"proposed-only boundary: CodeChange status is applied"}, seen)
	}
	status := "pass"
	if len(missing) > 0 {
		status = "fail"
	}
	return Epic5GateFValidation{Status: status, Missing: missing}
}

func epic5StatusesFromGateFValidation(validation Epic5GateFValidation) epic5GraphStatuses {
	if validation.Status == "pass" {
		return epic5GraphStatuses{FactoryStatus: "certified", TaskStatus: "certified", TestRunStatus: "pass", GateStatus: "pass"}
	}
	return epic5GraphStatuses{FactoryStatus: "rejected", TaskStatus: "rejected", TestRunStatus: "fail", GateStatus: "fail"}
}

func prefixMissing(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, prefix+": "+value)
	}
	return out
}

func epic5AppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func epic5AppendEdges(graph *v39.InMemoryStore, ids epic5FixtureIDs, createdAt time.Time, hasInvocation, includeFailure bool) error {
	edges := []v39.CommonEdge{
		epic5Edge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		epic5Edge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		epic5Edge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		epic5Edge("task_prompt", v39.EdgeProduced, ids.task, ids.promptArtifact, createdAt),
		epic5Edge("task_response", v39.EdgeProduced, ids.task, ids.responseArtifact, createdAt),
		epic5Edge("task_proposal", v39.EdgeProduced, ids.task, ids.proposalArtifact, createdAt),
		epic5Edge("task_patch", v39.EdgeProduced, ids.task, ids.patchArtifact, createdAt),
		epic5Edge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		epic5Edge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		epic5Edge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
		epic5Edge("proposal_patch", v39.EdgeProduced, ids.planningProposal, ids.patchArtifact, createdAt),
		epic5Edge("auth_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
		epic5Edge("auth_human", v39.EdgeApprovedBy, ids.authorityRequest, ids.humanApproval, createdAt),
	}
	if hasInvocation {
		edges = append(edges,
			epic5Edge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
			epic5Edge("invocation_auth", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
			epic5Edge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
			epic5Edge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt),
			epic5Edge("change_patch", v39.EdgeModified, ids.codeChange, ids.patchArtifact, createdAt),
		)
	}
	if includeFailure {
		edges = append(edges, epic5Edge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func epic5BuildProjection(graph *v39.InMemoryStore, ids epic5FixtureIDs, mode Epic5LLMProposalMode, transcript epic5RecordedTranscriptEvidence, graphRun epic5GraphRun) (Epic5LLMProposalProjection, error) {
	gate, err := epic5GateResultFromGraph(graph, ids.gateResult)
	if err != nil {
		return Epic5LLMProposalProjection{}, err
	}
	authorityDecision, err := epic5AuthorityDecisionFromGraph(graph, ids.authorityDecision)
	if err != nil {
		return Epic5LLMProposalProjection{}, err
	}
	humanApproval, err := epic5HumanApprovalFromGraph(graph, ids.humanApproval)
	if err != nil {
		return Epic5LLMProposalProjection{}, err
	}
	audit := graphRun.AuditReport
	status := statusString(gate.CommonNode.Status)
	packetStatus := "pass"
	if mode != Epic5LLMProposalReviewOnly {
		packetStatus = "fail"
	}
	projection := Epic5LLMProposalProjection{
		GeneratedAt: epic5FixtureTime().Format(time.RFC3339),
		Source:      "work-epic5-recorded-llm-proposal-fixture",
		Mode:        mode,
		Proposal: Epic5ProposalEvidence{
			ProposalArtifactRef: ids.proposalArtifact,
			CodeChangeID:        epic5CodeChangeID(ids, mode),
			TargetRepo:          "transpara-ai/work",
			TargetPath:          epic5ProposalTargetPath,
			ProposedDiffRef:     ids.patchArtifact,
			ProposedOnly:        mode == Epic5LLMProposalReviewOnly,
			Applied:             mode == Epic5LLMProposalAppliedPatch,
			Summary:             "Recorded LLM proposed a review-only design note patch; the fixture records the diff as evidence.",
		},
		Authority: Epic5AuthorityEvidence{
			AuthorityRequestID:  ids.authorityRequest,
			AuthorityDecisionID: ids.authorityDecision,
			HumanApprovalID:     ids.humanApproval,
			RequestedAction:     "repo.mutate.proposed_patch",
			Decision:            authorityDecision.Decision,
			HumanDecision:       humanApproval.Decision,
			Scope:               append([]string(nil), authorityDecision.Scope...),
			Summary:             authorityDecision.Reason,
		},
		Capability: Epic5InfluenceEvidence{
			ID:             ids.capabilityArtifact,
			Status:         requiredPathStatus(graphRun.CapabilityUsagePath),
			Summary:        "CapabilityArtifact and USED_CAPABILITY evidence recorded for the prompt section that shaped the proposal.",
			EventGraphRefs: []string{egRef(v39.TypeCapabilityArtifact, ids.capabilityArtifact)},
		},
		Knowledge: Epic5InfluenceEvidence{
			ID:             ids.knowledgeReference,
			Status:         requiredPathStatus(graphRun.KnowledgePath),
			Summary:        "KnowledgeReference records the merged Gate F authorization packet as material review criteria.",
			EventGraphRefs: []string{egRef(v39.TypeKnowledgeReference, ids.knowledgeReference)},
		},
		GateEvidence: []Epic5GateEvidence{
			{GateName: gate.GateName, Status: status, GateResultID: gate.CommonNode.ID, EvidenceRefs: append([]string(nil), gate.EvidenceRefs...), MissingRefs: append([]string(nil), graphRun.GateFValidation.Missing...)},
		},
		AuditReport: Epic5AuditEvidence{
			ID:           audit.CommonNode.ID,
			TargetType:   audit.TargetType,
			TargetID:     audit.TargetID,
			Status:       statusString(audit.CommonNode.Status),
			TraceScore:   audit.TraceScore,
			MissingLinks: append([]string(nil), audit.MissingLinks...),
		},
		ProofOfWorkPacket: Epic5ProofOfWorkPacket{
			ID:      ids.proofPacket,
			Status:  packetStatus,
			Summary: "Epic 5 Gate F recorded LLM proposal proof-of-work packet; proposal remains review-only.",
			Proposal: Epic5ProofOfWorkItem{
				Label:          "Proposed patch",
				Status:         epic5CodeChangeStatus(mode),
				Summary:        "Target " + epic5ProposalTargetPath + "; diff recorded but not applied by the happy-path fixture.",
				ArtifactRef:    ids.patchArtifact,
				EventGraphRefs: []string{egRef(v39.TypeArtifact, ids.patchArtifact), egRef(v39.TypeCodeChange, epic5CodeChangeID(ids, mode))},
			},
			Validation: Epic5ProofOfWorkItem{
				Label:          "Gate F validation",
				Status:         status,
				Summary:        "Unit fixture validates invocation hashes, proposal-only boundary, capability/knowledge influence, and human authority evidence.",
				ArtifactRef:    ids.testRun,
				EventGraphRefs: []string{egRef(v39.TypeTestRun, ids.testRun), egRef(v39.TypeGateResult, ids.gateResult)},
			},
			ReviewEvidence: Epic5ProofOfWorkItem{
				Label:          "Human review boundary",
				Status:         humanApproval.Decision,
				Summary:        humanApproval.Reason,
				ArtifactRef:    ids.humanApproval,
				EventGraphRefs: []string{egRef(v39.TypeHumanApproval, ids.humanApproval)},
			},
			AuditEvidence: Epic5ProofOfWorkItem{
				Label:          "Audit report",
				Status:         statusString(audit.CommonNode.Status),
				Summary:        "Audit report reconstructs trace and influence evidence for the recorded proposal trial.",
				ArtifactRef:    audit.CommonNode.ID,
				EventGraphRefs: []string{egRef(v39.TypeAuditReport, audit.CommonNode.ID)},
			},
			AuthorityDecision: Epic5ProofOfWorkItem{
				Label:          "Authority decision",
				Status:         authorityDecision.Decision,
				Summary:        authorityDecision.Reason,
				ArtifactRef:    ids.authorityDecision,
				EventGraphRefs: []string{egRef(v39.TypeAuthorityDecision, ids.authorityDecision), egRef(v39.TypeHumanApproval, ids.humanApproval)},
			},
			NonExecutionProof: []Epic5ProofOfWorkItem{
				{Label: "No ExecutionReceipt", Status: "pass", Summary: "The fixture records no ExecutionReceipt production path for repo mutation or protected side effects.", ArtifactRef: "", EventGraphRefs: []string{}},
				{Label: "Network and secrets", Status: "pass", Summary: "Runtime envelope disables network and secrets; command log records apply_patch as denied.", ArtifactRef: ids.runtimeEnvelope, EventGraphRefs: []string{egRef(v39.TypeRuntimeEnvelope, ids.runtimeEnvelope)}},
			},
			InfluenceEvidence: []Epic5ProofOfWorkItem{
				{Label: "CapabilityArtifact usage", Status: requiredPathStatus(graphRun.CapabilityUsagePath), Summary: "Prompt section capability use is linked by USED_CAPABILITY.", ArtifactRef: ids.capabilityArtifact, EventGraphRefs: []string{egRef(v39.TypeCapabilityArtifact, ids.capabilityArtifact)}},
				{Label: "KnowledgeReference", Status: requiredPathStatus(graphRun.KnowledgePath), Summary: "Merged Gate F authorization packet influenced prompt and review criteria.", ArtifactRef: ids.knowledgeReference, EventGraphRefs: []string{egRef(v39.TypeKnowledgeReference, ids.knowledgeReference)}},
			},
			EventGraphRefs: []string{egRef(v39.TypeFactoryOrder, ids.factoryOrder), egRef(v39.TypeTask, ids.task), egRef(v39.TypeGateResult, ids.gateResult), egRef(v39.TypeAuditReport, ids.auditReport)},
		},
	}
	if mode != Epic5LLMProposalMissingInvocation {
		projection.LLMInvocation = &Epic5LLMInvocationEvidence{
			ActorInvocationID:   ids.actorInvocation,
			ActorID:             epic5FixtureActorID,
			ModelLabel:          epic5RecordedModelLabel,
			ProviderLabel:       epic5RecordedProviderLabel,
			PromptHash:          transcript.PromptHash,
			ResponseHash:        transcript.ResponseHash,
			InputContractHash:   transcript.InputContractHash,
			OutputContractHash:  transcript.OutputContractHash,
			RecordedAt:          epic5FixtureTime().Format(time.RFC3339),
			PromptArtifactRef:   ids.promptArtifact,
			ResponseArtifactRef: ids.responseArtifact,
		}
		projection.ProofOfWorkPacket.LLMContribution = &Epic5ProofOfWorkItem{
			Label:          "Recorded LLM contribution",
			Status:         "recorded",
			Summary:        "Recorded LLM model " + epic5RecordedModelLabel + " via " + epic5RecordedProviderLabel + " produced proposal: " + transcript.Proposal,
			ArtifactRef:    ids.responseArtifact,
			EventGraphRefs: []string{egRef(v39.TypeActorInvocation, ids.actorInvocation), egRef(v39.TypeArtifact, ids.responseArtifact)},
		}
	}
	if mode == Epic5LLMProposalMissingInvocation {
		projection.Errors = append(projection.Errors, "recorded LLM invocation missing")
		projection.NegativeEvidence = append(projection.NegativeEvidence, Epic5ProofOfWorkItem{Label: "Missing invocation", Status: graphRun.GateFValidation.Status, Summary: "Gate F cannot pass without ActorInvocation evidence.", ArtifactRef: ids.gateResult, EventGraphRefs: []string{egRef(v39.TypeGateResult, ids.gateResult)}})
	}
	if mode == Epic5LLMProposalAppliedPatch {
		projection.Errors = append(projection.Errors, "proposal marked applied")
		projection.NegativeEvidence = append(projection.NegativeEvidence, Epic5ProofOfWorkItem{Label: "Applied proposal", Status: graphRun.GateFValidation.Status, Summary: "Gate F cannot pass if the proposal is applied instead of remaining proposed-only.", ArtifactRef: ids.codeChange, EventGraphRefs: []string{egRef(v39.TypeCodeChange, ids.codeChange)}})
	}
	return projection, nil
}

func epic5GateResultFromGraph(graph *v39.InMemoryStore, id string) (*v39.GateResult, error) {
	record, err := graph.Get(id)
	if err != nil {
		return nil, err
	}
	gate, ok := record.(*v39.GateResult)
	if !ok {
		return nil, fmt.Errorf("record %s is %T, want *v39.GateResult", id, record)
	}
	return gate, nil
}

func epic5AuthorityDecisionFromGraph(graph *v39.InMemoryStore, id string) (*v39.AuthorityDecision, error) {
	record, err := graph.Get(id)
	if err != nil {
		return nil, err
	}
	decision, ok := record.(*v39.AuthorityDecision)
	if !ok {
		return nil, fmt.Errorf("record %s is %T, want *v39.AuthorityDecision", id, record)
	}
	return decision, nil
}

func epic5HumanApprovalFromGraph(graph *v39.InMemoryStore, id string) (*v39.HumanApproval, error) {
	record, err := graph.Get(id)
	if err != nil {
		return nil, err
	}
	approval, ok := record.(*v39.HumanApproval)
	if !ok {
		return nil, fmt.Errorf("record %s is %T, want *v39.HumanApproval", id, record)
	}
	return approval, nil
}

func (p Epic5ProofOfWorkPacket) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func epic5GateEvidenceRefs(ids epic5FixtureIDs, hasInvocation bool) []string {
	refs := []string{ids.testRun, ids.promptArtifact, ids.responseArtifact, ids.proposalArtifact, ids.patchArtifact, ids.authorityDecision, ids.humanApproval, ids.capabilityArtifact, ids.knowledgeReference}
	if hasInvocation {
		refs = append([]string{ids.actorInvocation, ids.codeChange}, refs...)
	}
	return refs
}

func epic5OptionalInvocation(ids epic5FixtureIDs, hasInvocation bool) *string {
	if !hasInvocation {
		return nil
	}
	return &ids.actorInvocation
}

func epic5ActorInvocationID(ids epic5FixtureIDs, mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalMissingInvocation {
		return ""
	}
	return ids.actorInvocation
}

func epic5CodeChangeID(ids epic5FixtureIDs, mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalMissingInvocation {
		return ""
	}
	return ids.codeChange
}

func epic5CertificationID(cert *v39.Certification) string {
	if cert == nil {
		return ""
	}
	return cert.CommonNode.ID
}

func epic5AuthorityDecision(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalAppliedPatch {
		return "Forbidden"
	}
	return "ApprovalRequired"
}

func epic5AuthorityDecisionStatus(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalAppliedPatch {
		return "denied"
	}
	return "review_required"
}

func epic5AuthorityReason(mode Epic5LLMProposalMode) string {
	switch mode {
	case Epic5LLMProposalAppliedPatch:
		return "Proposal application is forbidden; Gate F permits review-only evidence, not repo mutation."
	case Epic5LLMProposalMissingInvocation:
		return "More evidence is required because no recorded LLM ActorInvocation exists."
	default:
		return "Human review is required before any action; only review of the proposed patch is in scope."
	}
}

func epic5HumanDecision(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalReviewOnly {
		return "approved"
	}
	if mode == Epic5LLMProposalAppliedPatch {
		return "denied"
	}
	return "more_evidence_required"
}

func epic5HumanReason(mode Epic5LLMProposalMode) string {
	switch mode {
	case Epic5LLMProposalAppliedPatch:
		return "Human reviewer denies the path because the proposal was marked applied."
	case Epic5LLMProposalMissingInvocation:
		return "Human reviewer requires recorded LLM invocation evidence before Gate F can pass."
	default:
		return "Human reviewer approves review-only handling of the recorded proposal; no mutation is authorized."
	}
}

func epic5CodeChangeStatus(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalAppliedPatch {
		return "applied"
	}
	return "proposed"
}

func epic5FailureClass(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalAppliedPatch {
		return "proposal_applied"
	}
	return "missing_llm_invocation"
}

func epic5FailureSummary(mode Epic5LLMProposalMode) string {
	if mode == Epic5LLMProposalAppliedPatch {
		return "Gate F failed because the proposal was marked applied instead of proposed-only."
	}
	return "Gate F failed because the recorded LLM ActorInvocation evidence is missing."
}

func requiredPathStatus(path v39.RequiredPath) string {
	if path.Completed {
		return "pass"
	}
	return "fail"
}

func epic5Common(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{
		ID:             id,
		Type:           typ,
		CreatedAt:      epic5FixtureTime(),
		CreatedBy:      epic5FixtureActorID,
		Status:         &status,
		IdempotencyKey: "idem_" + id,
		CorrelationID:  "corr_epic5_llm_proposal",
	}
}

func epic5Edge(label, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_epic5_" + label + ":" + from + ":" + typ + ":" + to
	return v39.CommonEdge{
		ID:             id,
		Type:           typ,
		FromID:         from,
		ToID:           to,
		CreatedAt:      createdAt,
		CreatedBy:      epic5FixtureActorID,
		CorrelationID:  "corr_epic5_llm_proposal",
		IdempotencyKey: "idem_" + id,
	}
}

func epic5FixtureTime() time.Time {
	t, _ := time.Parse(time.RFC3339, epic5FixtureTimeRFC)
	return t
}

func epic5Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
