package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// telPipelinePhase is one structured phase emitted by hive pipeline mode.
type telPipelinePhase struct {
	CycleID       string    `json:"cycle_id"`
	Phase         string    `json:"phase"`
	WorkflowStage string    `json:"workflow_stage"`
	Outcome       string    `json:"outcome"`
	Repo          *string   `json:"repo"`
	TaskID        *string   `json:"task_id"`
	TaskTitle     *string   `json:"task_title"`
	DurationSecs  *float64  `json:"duration_secs"`
	CostUSD       float64   `json:"cost_usd"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	BoardOpen     int       `json:"board_open"`
	ReviseCount   int       `json:"revise_count"`
	Summary       *string   `json:"summary"`
	Error         *string   `json:"error"`
	InputRef      *string   `json:"input_ref"`
	OutputRef     *string   `json:"output_ref"`
	RecordedAt    time.Time `json:"recorded_at"`
}

type telPipelineReport struct {
	CycleID           string             `json:"cycle_id"`
	Status            string             `json:"status"`
	CurrentStage      string             `json:"current_stage"`
	CurrentPhase      string             `json:"current_phase"`
	LastOutcome       string             `json:"last_outcome"`
	LastSummary       string             `json:"last_summary"`
	StartedAt         time.Time          `json:"started_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	DurationSecs      float64            `json:"duration_secs"`
	TotalCostUSD      float64            `json:"total_cost_usd"`
	TotalInputTokens  int                `json:"total_input_tokens"`
	TotalOutputTokens int                `json:"total_output_tokens"`
	TotalTokens       int                `json:"total_tokens"`
	OpenBoardItems    int                `json:"open_board_items"`
	ReviseCount       int                `json:"revise_count"`
	IntakeComplete    bool               `json:"intake_complete"`
	DesignComplete    bool               `json:"design_complete"`
	EmissionComplete  bool               `json:"emission_complete"`
	Phases            []telPipelinePhase `json:"phases"`
	HumanStatus       string             `json:"human_status"`
}

func (sv *server) queryPipelinePhases(ctx context.Context, limit int) ([]telPipelinePhase, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
		WITH latest AS (
			SELECT cycle_id
			FROM telemetry_pipeline_phases
			ORDER BY recorded_at DESC
			LIMIT 1
		)
		SELECT cycle_id, phase, workflow_stage, outcome, repo, task_id, task_title,
		       duration_secs::float8, cost_usd::float8, input_tokens, output_tokens,
		       board_open, revise_count, summary, error, input_ref, output_ref, recorded_at
		FROM telemetry_pipeline_phases
		WHERE cycle_id = (SELECT cycle_id FROM latest)
		ORDER BY recorded_at ASC
		LIMIT $1`

	rows, err := sv.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phases []telPipelinePhase
	for rows.Next() {
		var p telPipelinePhase
		if err := rows.Scan(
			&p.CycleID, &p.Phase, &p.WorkflowStage, &p.Outcome,
			&p.Repo, &p.TaskID, &p.TaskTitle, &p.DurationSecs, &p.CostUSD,
			&p.InputTokens, &p.OutputTokens, &p.BoardOpen, &p.ReviseCount,
			&p.Summary, &p.Error, &p.InputRef, &p.OutputRef, &p.RecordedAt,
		); err != nil {
			return nil, err
		}
		phases = append(phases, p)
	}
	return phases, rows.Err()
}

func buildPipelineReport(phases []telPipelinePhase) *telPipelineReport {
	if len(phases) == 0 {
		return nil
	}
	first := phases[0]
	last := phases[len(phases)-1]
	report := &telPipelineReport{
		CycleID:      first.CycleID,
		Status:       "running",
		CurrentStage: last.WorkflowStage,
		CurrentPhase: last.Phase,
		LastOutcome:  last.Outcome,
		StartedAt:    first.RecordedAt,
		UpdatedAt:    last.RecordedAt,
		Phases:       phases,
	}
	if last.Summary != nil {
		report.LastSummary = *last.Summary
	}
	for _, p := range phases {
		report.TotalCostUSD += p.CostUSD
		report.TotalInputTokens += p.InputTokens
		report.TotalOutputTokens += p.OutputTokens
		report.TotalTokens += p.InputTokens + p.OutputTokens
		report.OpenBoardItems = p.BoardOpen
		if p.ReviseCount > report.ReviseCount {
			report.ReviseCount = p.ReviseCount
		}
		switch p.WorkflowStage {
		case "intake":
			report.IntakeComplete = p.Outcome != "no.tasks"
		case "design":
			report.DesignComplete = p.Outcome != "no.tasks"
		case "emission":
			report.EmissionComplete = p.Outcome != "escalation"
		}
		if p.Outcome == "escalation" || p.Error != nil && *p.Error != "" {
			report.Status = "blocked"
		}
	}
	if last.WorkflowStage == "audit" && last.Outcome == "audit.done" {
		report.Status = "complete"
	}
	report.DurationSecs = report.UpdatedAt.Sub(report.StartedAt).Seconds()
	report.HumanStatus = pipelineHumanStatus(report)
	return report
}

func pipelineHumanStatus(r *telPipelineReport) string {
	if r == nil {
		return "No pipeline cycle has reported structured telemetry yet."
	}
	detail := r.LastSummary
	if detail == "" {
		detail = fmt.Sprintf("%s ended with %s", r.CurrentPhase, r.LastOutcome)
	}
	return fmt.Sprintf(
		"Cycle %s is %s. Current stage: %s/%s. %s. Board open: %d. Cost: $%.4f. Revisions: %d.",
		r.CycleID, r.Status, r.CurrentStage, r.CurrentPhase, detail,
		r.OpenBoardItems, r.TotalCostUSD, r.ReviseCount,
	)
}

// telemetryPipelineReport handles GET /telemetry/pipeline/report.
func (sv *server) telemetryPipelineReport(w http.ResponseWriter, r *http.Request) {
	if sv.pool == nil {
		telemetryUnavailable(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	phases, err := sv.queryPipelinePhases(ctx, 100)
	if err != nil {
		if isMissingTable(err) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error":  "pipeline telemetry not available",
				"detail": "telemetry_pipeline_phases table not initialized",
			})
			return
		}
		telemetryDBErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"report":      buildPipelineReport(phases),
		"computed_at": time.Now().UTC(),
	})
}
