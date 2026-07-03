package worklifecycle

import (
	"encoding/json"
	"strings"
	"testing"
)

type canonicalFixture struct {
	phase        PhaseKind
	exec         ExecState
	readyToRun   bool
	blocked      bool
	policyDenied bool
	failed       bool
	terminal     TerminalKind
	legacy       LegacyMark
}

func TestTC3_CanonicalWorkStateConstraintProduct(t *testing.T) {
	valid := canonicalFixture{
		phase:    PhasePreExec,
		terminal: TerminalNone,
		legacy:   LegacyNone,
	}
	if _, err := newCanonicalFixture(valid); err != nil {
		t.Fatalf("valid seeded canonical state rejected: %v", err)
	}

	tests := []struct {
		name string
		in   canonicalFixture
	}{
		{
			name: "zero value",
			in:   canonicalFixture{},
		},
		{
			name: "exec phase without exec",
			in: canonicalFixture{
				phase:    PhaseExec,
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "pre_exec with exec",
			in: canonicalFixture{
				phase:    PhasePreExec,
				exec:     ExecStateImplementing,
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "terminal with exec",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				exec:     ExecStateVerified,
				terminal: TerminalCertified,
				legacy:   LegacyNone,
			},
		},
		{
			name: "terminal phase without terminal",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "pre_exec with terminal",
			in: canonicalFixture{
				phase:    PhasePreExec,
				terminal: TerminalRejected,
				legacy:   LegacyNone,
			},
		},
		{
			name: "policy denied without blocked",
			in: canonicalFixture{
				phase:        PhaseExec,
				exec:         ExecStateImplementing,
				policyDenied: true,
				terminal:     TerminalNone,
				legacy:       LegacyNone,
			},
		},
		{
			name: "policy denied outside exec phase",
			in: canonicalFixture{
				phase:        PhasePreExec,
				blocked:      true,
				policyDenied: true,
				terminal:     TerminalNone,
				legacy:       LegacyNone,
			},
		},
		{
			name: "terminal with policy denied",
			in: canonicalFixture{
				phase:        PhaseTerminal,
				blocked:      true,
				policyDenied: true,
				terminal:     TerminalRejected,
				legacy:       LegacyNone,
			},
		},
		{
			name: "legacy outside pre_exec exec variant",
			in: canonicalFixture{
				phase:    PhaseExec,
				exec:     ExecStateImplementing,
				terminal: TerminalNone,
				legacy:   LegacyMarkLegacy,
			},
		},
		{
			name: "legacy outside pre_exec terminal variant",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				terminal: TerminalSuperseded,
				legacy:   LegacyMarkLegacy,
			},
		},
		{
			name: "legacy_completed with certified",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				terminal: TerminalCertified,
				legacy:   LegacyMarkCompleted,
			},
		},
		{
			name: "ready outside pre_exec",
			in: canonicalFixture{
				phase:      PhaseExec,
				exec:       ExecStateImplementing,
				readyToRun: true,
				terminal:   TerminalNone,
				legacy:     LegacyNone,
			},
		},
		{
			name: "failed outside exec",
			in: canonicalFixture{
				phase:    PhasePreExec,
				failed:   true,
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "terminal with blocked overlay",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				blocked:  true,
				terminal: TerminalRejected,
				legacy:   LegacyNone,
			},
		},
		{
			name: "terminal with ready overlay",
			in: canonicalFixture{
				phase:      PhaseTerminal,
				readyToRun: true,
				terminal:   TerminalRejected,
				legacy:     LegacyNone,
			},
		},
		{
			name: "terminal with failed overlay",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				failed:   true,
				terminal: TerminalRejected,
				legacy:   LegacyNone,
			},
		},
		{
			name: "invalid phase enum",
			in: canonicalFixture{
				phase:    PhaseKind("paused"),
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "invalid exec enum",
			in: canonicalFixture{
				phase:    PhaseExec,
				exec:     ExecState("running"),
				terminal: TerminalNone,
				legacy:   LegacyNone,
			},
		},
		{
			name: "invalid terminal enum",
			in: canonicalFixture{
				phase:    PhaseTerminal,
				terminal: TerminalKind("done"),
				legacy:   LegacyNone,
			},
		},
		{
			name: "invalid legacy enum",
			in: canonicalFixture{
				phase:    PhasePreExec,
				terminal: TerminalNone,
				legacy:   LegacyMark("old"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := newCanonicalFixture(tc.in); err == nil {
				t.Fatalf("NewCanonicalWorkState accepted invalid state: %#v", got)
			}

			var decoded CanonicalWorkState
			if err := json.Unmarshal(canonicalJSON(t, tc.in), &decoded); err == nil {
				t.Fatalf("UnmarshalJSON accepted invalid state: %#v", decoded)
			}
		})
	}

	var zero CanonicalWorkState
	if err := zero.Validate(); err == nil {
		t.Fatal("zero-value CanonicalWorkState Validate returned nil")
	}
	if _, err := json.Marshal(zero); err == nil {
		t.Fatal("zero-value CanonicalWorkState MarshalJSON returned nil error")
	}
}

func TestTC7_DeriveUnprovenHeadRows(t *testing.T) {
	reviewed, err := Fold(protectedEvidence(), []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		AuthorityGranted(GrantedAuthority("bounded implementation")),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
		ExecCertified(PassingInternalResult()),
		CFARPassed(PassingCrossFamilyResult("codex", "claude", headA)),
	})
	if err != nil {
		t.Fatalf("fold reviewed state: %v", err)
	}
	var zeroHead Head
	for _, head := range []Head{"", " ", zeroHead} {
		t.Run("reviewed-head-unproven-current-"+strings.ReplaceAll(string(head), " ", "space"), func(t *testing.T) {
			got, err := Derive(reviewed, head)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if got != MacroCodeReview {
				t.Fatalf("Derive(%q) = %s, want CodeReview", head, got)
			}
		})
	}
	if got, err := Derive(reviewed, headA); err != nil || got != MacroReady {
		t.Fatalf("Derive matching head = %s, %v; want Ready nil", got, err)
	}

	noReviewedHead, err := Fold(protectedEvidence(), []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		AuthorityGranted(GrantedAuthority("bounded implementation")),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
		ExecCertified(PassingInternalResult()),
	})
	if err != nil {
		t.Fatalf("fold no-reviewed-head state: %v", err)
	}
	for _, head := range []Head{"", " ", headA} {
		t.Run("absent-reviewed-head-"+strings.ReplaceAll(string(head), " ", "space"), func(t *testing.T) {
			got, err := Derive(noReviewedHead, head)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if got != MacroCodeReview {
				t.Fatalf("Derive(%q) = %s, want CodeReview", head, got)
			}
		})
	}

	emptyReviewedHead := CrossFamilyResult{
		Artifact:       ArtifactPresent,
		AuthorFamily:   "codex",
		ReviewerFamily: "claude",
		Head:           "",
	}
	if crossFamilyPassed(emptyReviewedHead, true) {
		t.Fatal("present-but-empty ReviewedHead unexpectedly constructable through CFARPassed")
	}
	if _, err := Apply(noReviewedHead, CFARPassed(emptyReviewedHead)); err == nil {
		t.Fatal("present-but-empty ReviewedHead row was accepted; want construction rejection")
	}
}

func TestTC8_CFARBlockedDeterministicRevisionIdentity(t *testing.T) {
	events := []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		AuthorityGranted(GrantedAuthority("bounded implementation")),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
		ExecCertified(PassingInternalResult()),
		CFARBlocked(),
	}
	a, err := Fold(protectedEvidence(), events)
	if err != nil {
		t.Fatalf("fold a: %v", err)
	}
	b, err := Fold(protectedEvidence(), events)
	if err != nil {
		t.Fatalf("fold b: %v", err)
	}
	if !a.Equal(b) {
		t.Fatalf("identical cfar.blocked histories did not fold equal:\na=%#v\nb=%#v", a, b)
	}
	if a.Macro() != MacroCoding {
		t.Fatalf("cfar.blocked macro = %s, want Coding", a.Macro())
	}
	if exec, ok := a.Exec(); !ok || exec != ExecStateImplementing {
		t.Fatalf("cfar.blocked exec = %s ok=%v, want implementing true", exec, ok)
	}
	if _, ok := a.ReviewedHead(); ok {
		t.Fatal("cfar.blocked retained reviewed head")
	}
	if iar, _ := a.Gate(GateIAR); iar.State() != GateStateFailed {
		t.Fatalf("IAR gate after cfar.blocked = %s, want failed", iar.State())
	}
	if cfar, _ := a.Gate(GateCFAR); cfar.State() != GateStateFailed {
		t.Fatalf("CFAR gate after cfar.blocked = %s, want failed", cfar.State())
	}

	supersededEvents := append([]Event{}, events[:7]...)
	supersededEvents = append(supersededEvents, Superseded(), CFARBlocked())
	superseded, err := Fold(protectedEvidence(), supersededEvents)
	if err == nil {
		if a.Equal(superseded) {
			t.Fatal("history differing by superseded certification folded equal")
		}
		return
	}
	if !strings.Contains(err.Error(), "cfar.blocked") {
		t.Fatalf("superseded-different history failed for unexpected reason: %v", err)
	}
}

func newCanonicalFixture(in canonicalFixture) (CanonicalWorkState, error) {
	return NewCanonicalWorkState(
		in.phase,
		in.exec,
		in.readyToRun,
		in.blocked,
		in.policyDenied,
		in.failed,
		in.terminal,
		in.legacy,
	)
}

func canonicalJSON(t *testing.T, in canonicalFixture) []byte {
	t.Helper()
	out, err := json.Marshal(map[string]any{
		"phase":         in.phase,
		"exec":          in.exec,
		"ready_to_run":  in.readyToRun,
		"blocked":       in.blocked,
		"policy_denied": in.policyDenied,
		"failed":        in.failed,
		"terminal":      in.terminal,
		"legacy":        in.legacy,
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return out
}
