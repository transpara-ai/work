package work

import (
	"strings"
	"testing"

	"github.com/transpara-ai/work/pkg/worklifecycle"
)

type canonicalWant struct {
	phase        worklifecycle.PhaseKind
	exec         worklifecycle.ExecState
	readyToRun   bool
	blocked      bool
	policyDenied bool
	failed       bool
	terminal     worklifecycle.TerminalKind
	legacy       worklifecycle.LegacyMark
}

func TestTC1_MapTaskStatusFullDomain(t *testing.T) {
	wantByStatus := map[TaskStatus]canonicalWant{
		StatusCreated:             {phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusReady:               {phase: worklifecycle.PhasePreExec, readyToRun: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusRunning:             {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusBlocked:             {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, blocked: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusPolicyBlocked:       {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, blocked: true, policyDenied: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusFailed:              {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusRepairRequired:      {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusRepairRunning:       {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusRepaired:            {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusVerificationRunning: {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusVerified:            {phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateVerified, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone},
		StatusCertified:           {phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalCertified, legacy: worklifecycle.LegacyNone},
		StatusRejected:            {phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalRejected, legacy: worklifecycle.LegacyNone},
		StatusSuperseded:          {phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalSuperseded, legacy: worklifecycle.LegacyNone},
	}
	for _, status := range []TaskStatus{
		StatusCreated, StatusReady, StatusRunning, StatusBlocked, StatusPolicyBlocked,
		StatusFailed, StatusRepairRequired, StatusRepairRunning, StatusRepaired,
		StatusVerificationRunning, StatusVerified, StatusCertified, StatusRejected,
		StatusSuperseded,
	} {
		t.Run(string(status), func(t *testing.T) {
			got, err := MapTaskStatus(status)
			if err != nil {
				t.Fatalf("MapTaskStatus(%q): %v", status, err)
			}
			assertCanonicalWant(t, got, wantByStatus[status])
		})
	}
	for _, status := range []TaskStatus{"", "unknown", "paused"} {
		t.Run("error-"+string(status), func(t *testing.T) {
			if got, err := MapTaskStatus(status); err == nil {
				t.Fatalf("MapTaskStatus(%q) = %#v, want error", status, got)
			}
		})
	}
}

func TestTC2_MapLegacyTaskStatusFullDomain(t *testing.T) {
	wantByStatus := map[LegacyTaskStatus]canonicalWant{
		LegacyStatusPending:   {phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy},
		LegacyStatusAssigned:  {phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy},
		LegacyStatusReady:     {phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy},
		LegacyStatusBlocked:   {phase: worklifecycle.PhasePreExec, blocked: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy},
		LegacyStatusCompleted: {phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkCompleted},
	}
	for _, status := range []LegacyTaskStatus{
		LegacyStatusPending, LegacyStatusAssigned, LegacyStatusReady,
		LegacyStatusBlocked, LegacyStatusCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			got, err := MapLegacyTaskStatus(status)
			if err != nil {
				t.Fatalf("MapLegacyTaskStatus(%q): %v", status, err)
			}
			assertCanonicalWant(t, got, wantByStatus[status])
		})
	}
	for _, status := range []LegacyTaskStatus{"", "unknown"} {
		t.Run("error-"+string(status), func(t *testing.T) {
			if got, err := MapLegacyTaskStatus(status); err == nil {
				t.Fatalf("MapLegacyTaskStatus(%q) = %#v, want error", status, got)
			}
		})
	}
}

func TestTC10_LegacyCompletedNeverCertified(t *testing.T) {
	got, err := MapLegacyTaskStatus(LegacyStatusCompleted)
	if err != nil {
		t.Fatalf("MapLegacyTaskStatus(completed): %v", err)
	}
	if got.Terminal() == worklifecycle.TerminalCertified {
		t.Fatal("legacy completed mapped to canonical certification")
	}
	if got.Legacy() != worklifecycle.LegacyMarkCompleted {
		t.Fatalf("legacy mark = %s, want legacy_completed", got.Legacy())
	}
}

func TestTC4_CanonicalStateForProjectionFullProduct(t *testing.T) {
	statuses := []TaskStatus{
		"", StatusCreated, StatusReady, StatusRunning, StatusBlocked, StatusPolicyBlocked,
		StatusFailed, StatusRepairRequired, StatusRepairRunning, StatusRepaired,
		StatusVerificationRunning, StatusVerified, StatusCertified, StatusRejected,
		StatusSuperseded, TaskStatus("paused"), TaskStatus("junk"),
	}
	legacyStatuses := []LegacyTaskStatus{
		"", LegacyStatusPending, LegacyStatusCompleted, LegacyStatusBlocked,
		LegacyStatusAssigned, LegacyStatusReady, LegacyTaskStatus("legacy_junk"),
	}

	for _, status := range statuses {
		for _, legacyStatus := range legacyStatuses {
			for _, taskBlocked := range []bool{false, true} {
				for _, legacyBlocked := range []bool{false, true} {
					for _, taskReady := range []bool{false, true} {
						for _, legacyReady := range []bool{false, true} {
							name := strings.Join([]string{
								string(status), string(legacyStatus),
								boolName(taskBlocked), boolName(legacyBlocked),
								boolName(taskReady), boolName(legacyReady),
							}, "/")
							t.Run(name, func(t *testing.T) {
								projection := TaskProjection{Status: status, Blocked: taskBlocked, Ready: taskReady}
								legacy := LegacyTaskProjection{Status: legacyStatus, Blocked: legacyBlocked, Ready: legacyReady}
								got, err := canonicalStateForProjection(projection, legacy)
								want, wantErr := expectedProjectionCanonical(status, legacyStatus, taskBlocked || legacyBlocked, taskReady || legacyReady)
								if wantErr {
									if err == nil {
										t.Fatalf("canonicalStateForProjection() = %#v, want error", got)
									}
									return
								}
								if err != nil {
									t.Fatalf("canonicalStateForProjection(): %v", err)
								}
								assertCanonicalWant(t, got, want)
							})
						}
					}
				}
			}
		}
	}
}

func expectedProjectionCanonical(status TaskStatus, legacyStatus LegacyTaskStatus, anyBlocked, anyReady bool) (canonicalWant, bool) {
	var base canonicalWant
	switch {
	case status != "" && status != StatusCreated:
		var ok bool
		base, ok = expectedTaskStatusCanonical(status)
		if !ok {
			return canonicalWant{}, true
		}
	case legacyStatus == LegacyStatusCompleted || legacyStatus == LegacyStatusBlocked || legacyStatus == LegacyStatusAssigned || legacyStatus == LegacyStatusReady:
		base = expectedLegacyStatusCanonical(legacyStatus)
	case anyBlocked:
		base = canonicalWant{phase: worklifecycle.PhasePreExec, blocked: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}
	case anyReady:
		base = canonicalWant{phase: worklifecycle.PhasePreExec, readyToRun: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}
	default:
		base = canonicalWant{phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}
	}

	if base.phase != worklifecycle.PhaseTerminal {
		if anyBlocked {
			base.blocked = true
			base.readyToRun = false
		}
		if base.phase == worklifecycle.PhasePreExec && anyReady && !base.blocked {
			base.readyToRun = true
		}
	}
	return base, false
}

func expectedTaskStatusCanonical(status TaskStatus) (canonicalWant, bool) {
	switch status {
	case StatusCreated:
		return canonicalWant{phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusReady:
		return canonicalWant{phase: worklifecycle.PhasePreExec, readyToRun: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusRunning:
		return canonicalWant{phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusBlocked:
		return canonicalWant{phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, blocked: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusPolicyBlocked:
		return canonicalWant{phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, blocked: true, policyDenied: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusFailed, StatusRepairRequired, StatusRepairRunning, StatusRepaired, StatusVerificationRunning:
		return canonicalWant{phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateImplementing, failed: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusVerified:
		return canonicalWant{phase: worklifecycle.PhaseExec, exec: worklifecycle.ExecStateVerified, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyNone}, true
	case StatusCertified:
		return canonicalWant{phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalCertified, legacy: worklifecycle.LegacyNone}, true
	case StatusRejected:
		return canonicalWant{phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalRejected, legacy: worklifecycle.LegacyNone}, true
	case StatusSuperseded:
		return canonicalWant{phase: worklifecycle.PhaseTerminal, terminal: worklifecycle.TerminalSuperseded, legacy: worklifecycle.LegacyNone}, true
	default:
		return canonicalWant{}, false
	}
}

func expectedLegacyStatusCanonical(status LegacyTaskStatus) canonicalWant {
	switch status {
	case LegacyStatusBlocked:
		return canonicalWant{phase: worklifecycle.PhasePreExec, blocked: true, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy}
	case LegacyStatusCompleted:
		return canonicalWant{phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkCompleted}
	default:
		return canonicalWant{phase: worklifecycle.PhasePreExec, terminal: worklifecycle.TerminalNone, legacy: worklifecycle.LegacyMarkLegacy}
	}
}

func assertCanonicalWant(t *testing.T, got worklifecycle.CanonicalWorkState, want canonicalWant) {
	t.Helper()
	if got.Phase() != want.phase {
		t.Fatalf("phase = %s, want %s", got.Phase(), want.phase)
	}
	if got.Exec() != want.exec {
		t.Fatalf("exec = %s, want %s", got.Exec(), want.exec)
	}
	if got.ReadyToRun() != want.readyToRun {
		t.Fatalf("ready_to_run = %v, want %v", got.ReadyToRun(), want.readyToRun)
	}
	if got.Blocked() != want.blocked {
		t.Fatalf("blocked = %v, want %v", got.Blocked(), want.blocked)
	}
	if got.PolicyDenied() != want.policyDenied {
		t.Fatalf("policy_denied = %v, want %v", got.PolicyDenied(), want.policyDenied)
	}
	if got.Failed() != want.failed {
		t.Fatalf("failed = %v, want %v", got.Failed(), want.failed)
	}
	if got.Terminal() != want.terminal {
		t.Fatalf("terminal = %s, want %s", got.Terminal(), want.terminal)
	}
	if got.Legacy() != want.legacy {
		t.Fatalf("legacy = %s, want %s", got.Legacy(), want.legacy)
	}
}

func boolName(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
