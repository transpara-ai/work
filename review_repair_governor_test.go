package work_test

import (
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestEvaluateReviewRepairGovernorRevisesUnderThreshold(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		SourceIssueRefs:       []string{" transpara-ai/work#67 ", ""},
		RepairRevolutions:     1,
		ConsecutiveNoProgress: 0,
		OpenBlockers:          2,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionRevise || decision.NextState != work.ReviewRepairStateRepair || decision.Terminal {
		t.Fatalf("decision = %#v; want non-terminal revise to repair", decision)
	}
	if strings.Join(decision.SourceIssueRefs, ",") != "transpara-ai/work#67" {
		t.Fatalf("source refs = %#v", decision.SourceIssueRefs)
	}
	if !containsString(decision.Reasons, "2 blocker(s) remain open") {
		t.Fatalf("reasons = %#v", decision.Reasons)
	}
}

func TestEvaluateReviewRepairGovernorCompletesAfterValidation(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		ValidationPassed: true,
		OpenBlockers:     0,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionComplete || decision.NextState != work.ReviewRepairStateComplete || !decision.Terminal {
		t.Fatalf("decision = %#v; want terminal complete", decision)
	}
}

func TestEvaluateReviewRepairGovernorEscalatesProtectedActionWithoutAuthority(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		ProtectedActionRequired:    true,
		AuthorityDecisionAvailable: false,
		OpenBlockers:               1,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionEscalate || decision.NextState != work.ReviewRepairStateEscalateHuman || !decision.Terminal {
		t.Fatalf("decision = %#v; want human escalation", decision)
	}
	if !containsString(decision.HumanEscalationConditions, "authority-reviewer") {
		t.Fatalf("escalation roles = %#v", decision.HumanEscalationConditions)
	}
	if !strings.Contains(strings.Join(decision.Reasons, "; "), "scoped authority decision") {
		t.Fatalf("reasons = %#v", decision.Reasons)
	}
}

func TestEvaluateReviewRepairGovernorRecordsCallerSuppliedAuthority(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		ProtectedActionRequired:    true,
		AuthorityDecisionAvailable: true,
		OpenBlockers:               1,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionRevise || decision.NextState != work.ReviewRepairStateRepair || decision.Terminal {
		t.Fatalf("decision = %#v; want revise under caller-supplied authority", decision)
	}
	if !strings.Contains(strings.Join(decision.Reasons, "; "), "caller-supplied authority decision") {
		t.Fatalf("reasons = %#v; want authority dependency surfaced", decision.Reasons)
	}
}

func TestEvaluateReviewRepairGovernorEscalatesHumanScopeAndPreservesRefs(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		SourceIssueRefs:    []string{"transpara-ai/work#67"},
		HumanScopeRequired: true,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionEscalate || !decision.Terminal {
		t.Fatalf("decision = %#v; want terminal human scope escalation", decision)
	}
	if strings.Join(decision.SourceIssueRefs, ",") != "transpara-ai/work#67" {
		t.Fatalf("source refs = %#v", decision.SourceIssueRefs)
	}
	if !strings.Contains(strings.Join(decision.Reasons, "; "), "human scope") {
		t.Fatalf("reasons = %#v", decision.Reasons)
	}
}

func TestEvaluateReviewRepairGovernorRequiresSplit(t *testing.T) {
	policy := work.ReviewRepairGovernorPolicy{
		MaxRepairRevolutions:     4,
		SplitAfterRevolutions:    2,
		AbandonAfterRevolutions:  4,
		MaxNoProgressRevolutions: 3,
		HumanEscalationRoles:     []string{"maintainer"},
	}

	decision, err := work.EvaluateReviewRepairGovernor(policy, work.ReviewRepairLoopState{
		RepairRevolutions: 2,
		OpenBlockers:      3,
		SplitCandidate:    true,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionSplit || decision.NextState != work.ReviewRepairStateSplitRequired || decision.Terminal {
		t.Fatalf("decision = %#v; want non-terminal split requirement", decision)
	}
	if !containsString(decision.HumanEscalationConditions, "maintainer") {
		t.Fatalf("escalation roles = %#v", decision.HumanEscalationConditions)
	}
}

func TestEvaluateReviewRepairGovernorContinuesUnderThresholds(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		RepairRevolutions:     1,
		ConsecutiveNoProgress: 1,
		OpenBlockers:          0,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionContinue || decision.NextState != work.ReviewRepairStateReview || decision.Terminal {
		t.Fatalf("decision = %#v; want non-terminal continue", decision)
	}
}

func TestEvaluateReviewRepairGovernorSplitsAfterNoProgress(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		RepairRevolutions:     1,
		ConsecutiveNoProgress: work.DefaultMaxNoProgressRevolutions,
		OpenBlockers:          1,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionSplit || decision.NextState != work.ReviewRepairStateSplitRequired {
		t.Fatalf("decision = %#v; want split for no progress", decision)
	}
}

func TestEvaluateReviewRepairGovernorAbandonsAtThreshold(t *testing.T) {
	decision, err := work.EvaluateReviewRepairGovernor(work.DefaultReviewRepairGovernorPolicy(), work.ReviewRepairLoopState{
		SourceIssueRefs:   []string{"transpara-ai/work#67"},
		RepairRevolutions: work.DefaultAbandonAfterRevolutions,
		OpenBlockers:      1,
		SplitCandidate:    true,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionAbandon || decision.NextState != work.ReviewRepairStateAbandonRequired || !decision.Terminal {
		t.Fatalf("decision = %#v; want terminal abandon requirement", decision)
	}
	if strings.Join(decision.SourceIssueRefs, ",") != "transpara-ai/work#67" {
		t.Fatalf("source refs = %#v", decision.SourceIssueRefs)
	}
}

func TestEvaluateReviewRepairGovernorAbandonsBeforeMaxWhenConfigured(t *testing.T) {
	policy := work.ReviewRepairGovernorPolicy{
		MaxRepairRevolutions:     5,
		SplitAfterRevolutions:    2,
		AbandonAfterRevolutions:  4,
		MaxNoProgressRevolutions: 3,
		HumanEscalationRoles:     []string{"maintainer"},
	}

	decision, err := work.EvaluateReviewRepairGovernor(policy, work.ReviewRepairLoopState{
		RepairRevolutions: 4,
		OpenBlockers:      1,
	})
	if err != nil {
		t.Fatalf("EvaluateReviewRepairGovernor: %v", err)
	}

	if decision.Action != work.ReviewRepairActionAbandon || !decision.Terminal {
		t.Fatalf("decision = %#v; want abandon threshold", decision)
	}
	if !strings.Contains(strings.Join(decision.Reasons, "; "), "abandon threshold 4") {
		t.Fatalf("reasons = %#v; want abandon threshold reason", decision.Reasons)
	}
}

func TestEvaluateReviewRepairGovernorRejectsInvalidPolicyAndState(t *testing.T) {
	tests := []struct {
		name    string
		policy  work.ReviewRepairGovernorPolicy
		state   work.ReviewRepairLoopState
		wantErr string
	}{
		{
			name:    "negative max",
			policy:  work.ReviewRepairGovernorPolicy{MaxRepairRevolutions: -1},
			wantErr: "max_repair_revolutions",
		},
		{
			name: "split after max",
			policy: work.ReviewRepairGovernorPolicy{
				MaxRepairRevolutions:     2,
				SplitAfterRevolutions:    3,
				AbandonAfterRevolutions:  3,
				MaxNoProgressRevolutions: 1,
				HumanEscalationRoles:     []string{"maintainer"},
			},
			wantErr: "split_after_revolutions",
		},
		{
			name: "no progress after abandon",
			policy: work.ReviewRepairGovernorPolicy{
				MaxRepairRevolutions:     5,
				SplitAfterRevolutions:    2,
				AbandonAfterRevolutions:  3,
				MaxNoProgressRevolutions: 4,
				HumanEscalationRoles:     []string{"maintainer"},
			},
			wantErr: "max_no_progress_revolutions",
		},
		{
			name:    "negative blockers",
			policy:  work.DefaultReviewRepairGovernorPolicy(),
			state:   work.ReviewRepairLoopState{OpenBlockers: -1},
			wantErr: "open_blockers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := work.EvaluateReviewRepairGovernor(tt.policy, tt.state); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("EvaluateReviewRepairGovernor error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
