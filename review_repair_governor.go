package work

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxRepairRevolutions      = 3
	DefaultSplitAfterRevolutions     = 2
	DefaultAbandonAfterRevolutions   = 3
	DefaultMaxNoProgressRevolutions  = 2
	ReviewRepairStateReview          = "review"
	ReviewRepairStateRepair          = "repair"
	ReviewRepairStateSplitRequired   = "split_required"
	ReviewRepairStateAbandonRequired = "abandon_required"
	ReviewRepairStateEscalateHuman   = "human_escalation_required"
	ReviewRepairStateComplete        = "complete"
)

// ReviewRepairGovernorPolicy scopes the bounded review/repair loop before a
// caller mutates Work state or performs protected actions.
type ReviewRepairGovernorPolicy struct {
	MaxRepairRevolutions     int      `json:"max_repair_revolutions"`
	SplitAfterRevolutions    int      `json:"split_after_revolutions"`
	AbandonAfterRevolutions  int      `json:"abandon_after_revolutions"`
	MaxNoProgressRevolutions int      `json:"max_no_progress_revolutions"`
	HumanEscalationRoles     []string `json:"human_escalation_roles,omitempty"`
}

// ReviewRepairLoopState is caller-supplied evidence for one loop evaluation.
type ReviewRepairLoopState struct {
	SourceIssueRefs            []string `json:"source_issue_refs,omitempty"`
	CurrentState               string   `json:"current_state"`
	RepairRevolutions          int      `json:"repair_revolutions"`
	ConsecutiveNoProgress      int      `json:"consecutive_no_progress"`
	OpenBlockers               int      `json:"open_blockers"`
	ValidationPassed           bool     `json:"validation_passed"`
	SplitCandidate             bool     `json:"split_candidate"`
	ProtectedActionRequired    bool     `json:"protected_action_required"`
	HumanScopeRequired         bool     `json:"human_scope_required"`
	AuthorityDecisionAvailable bool     `json:"authority_decision_available"`
}

type ReviewRepairGovernorAction string

const (
	ReviewRepairActionContinue ReviewRepairGovernorAction = "continue"
	ReviewRepairActionRevise   ReviewRepairGovernorAction = "revise"
	ReviewRepairActionSplit    ReviewRepairGovernorAction = "split_required"
	ReviewRepairActionAbandon  ReviewRepairGovernorAction = "abandon_required"
	ReviewRepairActionEscalate ReviewRepairGovernorAction = "human_escalation_required"
	ReviewRepairActionComplete ReviewRepairGovernorAction = "complete"
)

// ReviewRepairGovernorDecision is a pure recommendation. It carries no side
// effects and does not authorize protected work by itself.
type ReviewRepairGovernorDecision struct {
	Action                    ReviewRepairGovernorAction `json:"action"`
	NextState                 string                     `json:"next_state"`
	Terminal                  bool                       `json:"terminal"`
	Reasons                   []string                   `json:"reasons"`
	SourceIssueRefs           []string                   `json:"source_issue_refs,omitempty"`
	HumanEscalationConditions []string                   `json:"human_escalation_conditions,omitempty"`
}

func DefaultReviewRepairGovernorPolicy() ReviewRepairGovernorPolicy {
	return ReviewRepairGovernorPolicy{
		MaxRepairRevolutions:     DefaultMaxRepairRevolutions,
		SplitAfterRevolutions:    DefaultSplitAfterRevolutions,
		AbandonAfterRevolutions:  DefaultAbandonAfterRevolutions,
		MaxNoProgressRevolutions: DefaultMaxNoProgressRevolutions,
		HumanEscalationRoles:     []string{"maintainer", "authority-reviewer"},
	}
}

func EvaluateReviewRepairGovernor(policy ReviewRepairGovernorPolicy, state ReviewRepairLoopState) (ReviewRepairGovernorDecision, error) {
	policy = normalizeReviewRepairGovernorPolicy(policy)
	state = normalizeReviewRepairLoopState(state)
	if err := validateReviewRepairGovernorPolicy(policy); err != nil {
		return ReviewRepairGovernorDecision{}, err
	}
	if err := validateReviewRepairLoopState(state); err != nil {
		return ReviewRepairGovernorDecision{}, err
	}

	base := ReviewRepairGovernorDecision{
		SourceIssueRefs: append([]string(nil), state.SourceIssueRefs...),
	}
	switch {
	case state.ProtectedActionRequired && !state.AuthorityDecisionAvailable:
		return withEscalation(base, "protected action requires a scoped authority decision", policy.HumanEscalationRoles), nil
	case state.HumanScopeRequired:
		return withEscalation(base, "human scope is required before the loop can continue", policy.HumanEscalationRoles), nil
	case state.ValidationPassed && state.OpenBlockers == 0:
		base.Action = ReviewRepairActionComplete
		base.NextState = ReviewRepairStateComplete
		base.Terminal = true
		base.Reasons = []string{"validation passed with zero open blockers"}
		return base, nil
	case state.RepairRevolutions >= policy.AbandonAfterRevolutions:
		base.Action = ReviewRepairActionAbandon
		base.NextState = ReviewRepairStateAbandonRequired
		base.Terminal = true
		base.Reasons = []string{fmt.Sprintf("repair revolutions %d reached abandon threshold %d", state.RepairRevolutions, policy.AbandonAfterRevolutions)}
		base.HumanEscalationConditions = policy.HumanEscalationRoles
		return base, nil
	case state.ConsecutiveNoProgress >= policy.MaxNoProgressRevolutions:
		base.Action = ReviewRepairActionSplit
		base.NextState = ReviewRepairStateSplitRequired
		base.Reasons = []string{fmt.Sprintf("no-progress revolutions %d reached split threshold %d", state.ConsecutiveNoProgress, policy.MaxNoProgressRevolutions)}
		base.HumanEscalationConditions = policy.HumanEscalationRoles
		return base, nil
	case state.SplitCandidate && state.RepairRevolutions >= policy.SplitAfterRevolutions:
		base.Action = ReviewRepairActionSplit
		base.NextState = ReviewRepairStateSplitRequired
		base.Reasons = []string{fmt.Sprintf("split candidate reached repair revolution threshold %d", policy.SplitAfterRevolutions)}
		base.HumanEscalationConditions = policy.HumanEscalationRoles
		return base, nil
	case state.OpenBlockers > 0:
		base.Action = ReviewRepairActionRevise
		base.NextState = ReviewRepairStateRepair
		base.Reasons = []string{fmt.Sprintf("%d blocker(s) remain open", state.OpenBlockers)}
		return base, nil
	default:
		base.Action = ReviewRepairActionContinue
		base.NextState = ReviewRepairStateReview
		base.Reasons = []string{"loop remains under configured thresholds"}
		return base, nil
	}
}

func normalizeReviewRepairGovernorPolicy(policy ReviewRepairGovernorPolicy) ReviewRepairGovernorPolicy {
	defaults := DefaultReviewRepairGovernorPolicy()
	if policy.MaxRepairRevolutions == 0 {
		policy.MaxRepairRevolutions = defaults.MaxRepairRevolutions
	}
	if policy.SplitAfterRevolutions == 0 {
		policy.SplitAfterRevolutions = defaults.SplitAfterRevolutions
	}
	if policy.AbandonAfterRevolutions == 0 {
		policy.AbandonAfterRevolutions = defaults.AbandonAfterRevolutions
	}
	if policy.MaxNoProgressRevolutions == 0 {
		policy.MaxNoProgressRevolutions = defaults.MaxNoProgressRevolutions
	}
	if len(policy.HumanEscalationRoles) == 0 {
		policy.HumanEscalationRoles = defaults.HumanEscalationRoles
	}
	policy.HumanEscalationRoles = cleanNonEmptyReviewRepairStrings(policy.HumanEscalationRoles)
	return policy
}

func validateReviewRepairGovernorPolicy(policy ReviewRepairGovernorPolicy) error {
	switch {
	case policy.MaxRepairRevolutions < 1:
		return fmt.Errorf("max_repair_revolutions must be positive")
	case policy.SplitAfterRevolutions < 1:
		return fmt.Errorf("split_after_revolutions must be positive")
	case policy.AbandonAfterRevolutions < 1:
		return fmt.Errorf("abandon_after_revolutions must be positive")
	case policy.MaxNoProgressRevolutions < 1:
		return fmt.Errorf("max_no_progress_revolutions must be positive")
	case policy.SplitAfterRevolutions > policy.MaxRepairRevolutions:
		return fmt.Errorf("split_after_revolutions must be less than or equal to max_repair_revolutions")
	case policy.AbandonAfterRevolutions > policy.MaxRepairRevolutions:
		return fmt.Errorf("abandon_after_revolutions must be less than or equal to max_repair_revolutions")
	case policy.SplitAfterRevolutions > policy.AbandonAfterRevolutions:
		return fmt.Errorf("split_after_revolutions must be less than or equal to abandon_after_revolutions")
	case len(policy.HumanEscalationRoles) == 0:
		return fmt.Errorf("human_escalation_roles must be non-empty")
	}
	return nil
}

func normalizeReviewRepairLoopState(state ReviewRepairLoopState) ReviewRepairLoopState {
	state.CurrentState = strings.TrimSpace(state.CurrentState)
	state.SourceIssueRefs = cleanNonEmptyReviewRepairStrings(state.SourceIssueRefs)
	return state
}

func validateReviewRepairLoopState(state ReviewRepairLoopState) error {
	switch {
	case state.RepairRevolutions < 0:
		return fmt.Errorf("repair_revolutions must be zero or greater")
	case state.ConsecutiveNoProgress < 0:
		return fmt.Errorf("consecutive_no_progress must be zero or greater")
	case state.OpenBlockers < 0:
		return fmt.Errorf("open_blockers must be zero or greater")
	}
	return nil
}

func withEscalation(base ReviewRepairGovernorDecision, reason string, roles []string) ReviewRepairGovernorDecision {
	base.Action = ReviewRepairActionEscalate
	base.NextState = ReviewRepairStateEscalateHuman
	base.Terminal = true
	base.Reasons = []string{reason}
	base.HumanEscalationConditions = append([]string(nil), roles...)
	return base
}

func cleanNonEmptyReviewRepairStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
