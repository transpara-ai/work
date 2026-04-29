package main

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPipelineReport(t *testing.T) {
	start := time.Date(2026, 4, 28, 19, 0, 0, 0, time.UTC)
	intakeSummary := "intake completed with outcome milestone.created; 1 tasks remain open"
	designSummary := "design completed with outcome tasks.created; 4 tasks remain open"
	phases := []telPipelinePhase{
		{
			CycleID:       "pipeline-20260428T190000Z",
			Phase:         "pm",
			WorkflowStage: "intake",
			Outcome:       "milestone.created",
			Summary:       &intakeSummary,
			BoardOpen:     1,
			CostUSD:       0.01,
			InputTokens:   100,
			OutputTokens:  50,
			RecordedAt:    start,
		},
		{
			CycleID:       "pipeline-20260428T190000Z",
			Phase:         "architect",
			WorkflowStage: "design",
			Outcome:       "tasks.created",
			Summary:       &designSummary,
			BoardOpen:     4,
			CostUSD:       0.02,
			InputTokens:   200,
			OutputTokens:  80,
			RecordedAt:    start.Add(2 * time.Minute),
		},
	}

	report := buildPipelineReport(phases)
	if report == nil {
		t.Fatal("report is nil")
	}
	if report.CycleID != "pipeline-20260428T190000Z" {
		t.Errorf("CycleID = %q", report.CycleID)
	}
	if report.CurrentStage != "design" || report.CurrentPhase != "architect" {
		t.Errorf("current = %s/%s", report.CurrentStage, report.CurrentPhase)
	}
	if !report.IntakeComplete || !report.DesignComplete {
		t.Errorf("completion flags: intake=%v design=%v", report.IntakeComplete, report.DesignComplete)
	}
	if report.TotalTokens != 430 {
		t.Errorf("TotalTokens = %d, want 430", report.TotalTokens)
	}
	if report.TotalInputTokens != 300 {
		t.Errorf("TotalInputTokens = %d, want 300", report.TotalInputTokens)
	}
	if report.TotalOutputTokens != 130 {
		t.Errorf("TotalOutputTokens = %d, want 130", report.TotalOutputTokens)
	}
	if report.OpenBoardItems != 4 {
		t.Errorf("OpenBoardItems = %d, want 4", report.OpenBoardItems)
	}
	if !strings.Contains(report.HumanStatus, "Current stage: design/architect") {
		t.Errorf("HumanStatus missing current stage: %q", report.HumanStatus)
	}
}

func TestBuildPipelineReportBlockedAndComplete(t *testing.T) {
	start := time.Date(2026, 4, 28, 19, 0, 0, 0, time.UTC)
	errText := "validation failed"
	blocked := buildPipelineReport([]telPipelinePhase{
		{
			CycleID:       "cycle-blocked",
			Phase:         "validator",
			WorkflowStage: "validation",
			Outcome:       "escalation",
			Error:         &errText,
			RecordedAt:    start,
		},
	})
	if blocked == nil || blocked.Status != "blocked" {
		t.Fatalf("blocked report status = %+v, want blocked", blocked)
	}

	complete := buildPipelineReport([]telPipelinePhase{
		{
			CycleID:       "cycle-complete",
			Phase:         "audit",
			WorkflowStage: "audit",
			Outcome:       "audit.done",
			RecordedAt:    start,
		},
	})
	if complete == nil || complete.Status != "complete" {
		t.Fatalf("complete report status = %+v, want complete", complete)
	}
}

func TestBuildPipelineReportEmpty(t *testing.T) {
	if got := buildPipelineReport(nil); got != nil {
		t.Fatalf("empty report = %+v, want nil", got)
	}
}
