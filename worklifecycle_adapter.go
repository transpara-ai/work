package work

import (
	"fmt"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work/pkg/worklifecycle"
)

func MapTaskStatus(status TaskStatus) (worklifecycle.CanonicalWorkState, error) {
	switch status {
	case StatusCreated:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusReady:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", true, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusRunning:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusBlocked:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, true, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusPolicyBlocked:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, true, true, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusFailed:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, true, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusRepairRequired:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, true, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusRepairRunning:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, true, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusRepaired:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, true, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusVerificationRunning:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateImplementing, false, false, false, true, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusVerified:
		return newCanonicalWorkState(worklifecycle.PhaseExec, worklifecycle.ExecStateVerified, false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case StatusCertified:
		return newCanonicalWorkState(worklifecycle.PhaseTerminal, "", false, false, false, false, worklifecycle.TerminalCertified, worklifecycle.LegacyNone)
	case StatusRejected:
		return newCanonicalWorkState(worklifecycle.PhaseTerminal, "", false, false, false, false, worklifecycle.TerminalRejected, worklifecycle.LegacyNone)
	case StatusSuperseded:
		return newCanonicalWorkState(worklifecycle.PhaseTerminal, "", false, false, false, false, worklifecycle.TerminalSuperseded, worklifecycle.LegacyNone)
	}
	return worklifecycle.CanonicalWorkState{}, fmt.Errorf("unknown task status %q", status)
}

func MapLegacyTaskStatus(status LegacyTaskStatus) (worklifecycle.CanonicalWorkState, error) {
	switch status {
	case LegacyStatusPending:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyMarkLegacy)
	case LegacyStatusAssigned:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyMarkLegacy)
	case LegacyStatusReady:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyMarkLegacy)
	case LegacyStatusBlocked:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, true, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyMarkLegacy)
	case LegacyStatusCompleted:
		return newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyMarkCompleted)
	}
	return worklifecycle.CanonicalWorkState{}, fmt.Errorf("unknown legacy task status %q", status)
}

func canonicalStateForProjection(projection TaskProjection, legacy LegacyTaskProjection) (worklifecycle.CanonicalWorkState, error) {
	var (
		base worklifecycle.CanonicalWorkState
		err  error
	)
	switch {
	case projection.Status != "" && projection.Status != StatusCreated:
		base, err = MapTaskStatus(projection.Status)
	case legacy.Status == LegacyStatusCompleted || legacy.Status == LegacyStatusBlocked || legacy.Status == LegacyStatusAssigned || legacy.Status == LegacyStatusReady:
		base, err = MapLegacyTaskStatus(legacy.Status)
	case projection.Blocked || legacy.Blocked:
		base, err = newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, true, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	case projection.Ready || legacy.Ready:
		base, err = newCanonicalWorkState(worklifecycle.PhasePreExec, "", true, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	default:
		base, err = newCanonicalWorkState(worklifecycle.PhasePreExec, "", false, false, false, false, worklifecycle.TerminalNone, worklifecycle.LegacyNone)
	}
	if err != nil {
		return worklifecycle.CanonicalWorkState{}, err
	}
	if base.Phase() == worklifecycle.PhaseTerminal {
		return base, nil
	}

	blocked := base.Blocked() || projection.Blocked || legacy.Blocked
	readyToRun := base.ReadyToRun()
	if blocked {
		readyToRun = false
	} else if base.Phase() == worklifecycle.PhasePreExec && (projection.Ready || legacy.Ready) {
		readyToRun = true
	}
	return newCanonicalWorkState(
		base.Phase(),
		base.Exec(),
		readyToRun,
		blocked,
		base.PolicyDenied(),
		base.Failed(),
		base.Terminal(),
		base.Legacy(),
	)
}

func newCanonicalWorkState(
	phase worklifecycle.PhaseKind,
	exec worklifecycle.ExecState,
	readyToRun bool,
	blocked bool,
	policyDenied bool,
	failed bool,
	terminal worklifecycle.TerminalKind,
	legacy worklifecycle.LegacyMark,
) (worklifecycle.CanonicalWorkState, error) {
	return worklifecycle.NewCanonicalWorkState(phase, exec, readyToRun, blocked, policyDenied, failed, terminal, legacy)
}

type taskSummaryFields struct {
	Task          Task
	Status        TaskStatus
	LegacyStatus  LegacyTaskStatus
	Assignee      types.ActorID
	Blocked       bool
	ArtifactCount int
	Waived        bool
	Ready         bool
	MissingGates  []string
	MissingFacts  []string
}

func newTaskSummary(fields taskSummaryFields) TaskSummary {
	summary := TaskSummary{
		Task:          fields.Task,
		Status:        fields.Status,
		LegacyStatus:  fields.LegacyStatus,
		Assignee:      fields.Assignee,
		Blocked:       fields.Blocked,
		ArtifactCount: fields.ArtifactCount,
		Waived:        fields.Waived,
		Ready:         fields.Ready,
		MissingGates:  fields.MissingGates,
		MissingFacts:  fields.MissingFacts,
	}
	attachCanonicalToTaskSummary(&summary)
	return summary
}

func attachCanonicalToTaskSummary(summary *TaskSummary) {
	projection := TaskProjection{
		Task:    summary.Task,
		Status:  summary.Status,
		Blocked: summary.Blocked,
		Ready:   summary.Ready,
	}
	legacy := LegacyTaskProjection{
		TaskID:   summary.Task.ID,
		Status:   summary.LegacyStatus,
		Assignee: summary.Assignee,
		Blocked:  summary.Blocked,
		Ready:    summary.Ready,
	}
	canonical, err := canonicalStateForProjection(projection, legacy)
	if err != nil {
		summary.Canonical = nil
		summary.CanonicalError = err.Error()
		return
	}
	summary.Canonical = &canonical
	summary.CanonicalError = ""
}
