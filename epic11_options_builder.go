package work

import (
	"time"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// Epic11OptionsInput carries the caller-supplied values needed to construct a
// fully-populated Epic11DocsDraftPROptions with internally consistent evidence.
// Every evidence ID, hash, and cross-reference is derived from these inputs
// using the same derivation logic as the internal validEpic11Options test helper.
type Epic11OptionsInput struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string
	Client         Epic11PullRequestCreator
	Now            time.Time

	Target Epic11DraftPullRequestTarget
	// ActorRole is the role of the REQUESTING actor (e.g. "implementer" or
	// "codex"). It is copied into AuthorityRequestEvidence.ActorRole and
	// identifies the agent submitting the mutation — NOT the human approver.
	ActorRole string
	// DeciderActorID identifies the HUMAN authority who approved this mutation.
	// Distinct from Source (the agent's ActorID) and ActorRole (the agent's role).
	DeciderActorID string
	// DeciderRole is the role of the human authority who approved this mutation
	// (e.g. "human"). Paired with DeciderActorID; distinct from ActorRole.
	DeciderRole string
	// SingleUseNonce must be unique per mutation invocation. It is used to derive
	// the three evidence IDs (auth_req_<nonce>, auth_dec_<nonce>, padc_<nonce>)
	// and is checked by the single-use replay guard to prevent re-execution.
	SingleUseNonce string
}

// BuildEpic11DocsDraftPROptions constructs a fully-populated
// Epic11DocsDraftPROptions whose authority-request, authority-decision, and
// policy-decision evidence fields are internally consistent and will satisfy
// the RunEpic11DocsDraftPRLiveMutation fail-closed validators.
//
// The caller must set opts.Causes after this call if causes are required.
func BuildEpic11DocsDraftPROptions(in Epic11OptionsInput) Epic11DocsDraftPROptions {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	target := in.Target
	titleHash := epic7Hash(target.Title)
	bodyHash := epic7Hash(target.Body)
	bundleHash := Epic11DocsDraftPRPolicyBundleHash()

	req := Epic11AuthorityRequestEvidence{
		ID:                     "auth_req_" + in.SingleUseNonce,
		ActorID:                in.Source.Value(),
		ActorRole:              in.ActorRole,
		Action:                 Epic11ActionPullRequestCreate,
		TargetRepository:       target.Repository,
		BaseRef:                target.BaseRef,
		BaseSHA:                target.BaseSHA,
		HeadRef:                target.HeadRef,
		HeadSHA:                target.HeadSHA,
		TitleHash:              titleHash,
		BodyHash:               bodyHash,
		ChangedFiles:           append([]string(nil), target.ChangedFiles...),
		ValidationEvidenceRefs: append([]string(nil), target.ValidationEvidenceRefs...),
		PolicyBundleID:         Epic11PolicyBundleID,
		PolicyBundleHash:       bundleHash,
		RollbackInstructions:   target.RollbackInstructions,
		SingleUseNonce:         in.SingleUseNonce,
		RequestedAt:            now.Add(-time.Minute),
		ExpiresAt:              now.Add(time.Hour),
	}

	decision := Epic11AuthorityDecisionEvidence{
		ID:                     "auth_dec_" + in.SingleUseNonce,
		AuthorityRequestID:     req.ID,
		ActorID:                req.ActorID,
		ActorRole:              req.ActorRole,
		DeciderActorID:         in.DeciderActorID,
		DeciderRole:            in.DeciderRole,
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

	policy := Epic11PolicyDecisionEvidence{
		DecisionID:           "padc_" + in.SingleUseNonce,
		AdapterID:            Epic11PolicyAdapterID,
		AdapterVersion:       "1.0.0",
		PolicyBundleID:       Epic11PolicyBundleID,
		PolicyBundleHash:     bundleHash,
		ProtectedActionType:  Epic11ActionPullRequestCreate,
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

	return Epic11DocsDraftPROptions{
		Source:            in.Source,
		ConversationID:    in.ConversationID,
		Causes:            in.Causes,
		WorkingDir:        in.WorkingDir,
		Client:            in.Client,
		Now:               now,
		Target:            target,
		AuthorityRequest:  req,
		AuthorityDecision: decision,
		PolicyDecision:    policy,
	}
}
