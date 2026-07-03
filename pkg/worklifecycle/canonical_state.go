package worklifecycle

import (
	"encoding/json"
	"fmt"
)

type PhaseKind string

const (
	PhasePreExec  PhaseKind = "pre_exec"
	PhaseExec     PhaseKind = "exec"
	PhaseTerminal PhaseKind = "terminal"
)

type TerminalKind string

const (
	TerminalNone       TerminalKind = "none"
	TerminalCertified  TerminalKind = "certified"
	TerminalRejected   TerminalKind = "rejected"
	TerminalSuperseded TerminalKind = "superseded"
)

type LegacyMark string

const (
	LegacyNone          LegacyMark = "none"
	LegacyMarkLegacy    LegacyMark = "legacy"
	LegacyMarkCompleted LegacyMark = "legacy_completed"
)

// CanonicalWorkState is the serializable projection form of the SP0 work
// lifecycle vocabulary. Its fields are deliberately unexported so every value
// is created through validating constructors or JSON decoding.
type CanonicalWorkState struct {
	phase        PhaseKind
	exec         ExecState
	readyToRun   bool
	blocked      bool
	policyDenied bool
	failed       bool
	terminal     TerminalKind
	legacy       LegacyMark
}

type canonicalWorkStateJSON struct {
	Phase        PhaseKind    `json:"phase"`
	Exec         ExecState    `json:"exec"`
	ReadyToRun   bool         `json:"ready_to_run"`
	Blocked      bool         `json:"blocked"`
	PolicyDenied bool         `json:"policy_denied"`
	Failed       bool         `json:"failed"`
	Terminal     TerminalKind `json:"terminal"`
	Legacy       LegacyMark   `json:"legacy"`
}

func NewCanonicalWorkState(
	phase PhaseKind,
	exec ExecState,
	readyToRun bool,
	blocked bool,
	policyDenied bool,
	failed bool,
	terminal TerminalKind,
	legacy LegacyMark,
) (CanonicalWorkState, error) {
	state := CanonicalWorkState{
		phase:        phase,
		exec:         exec,
		readyToRun:   readyToRun,
		blocked:      blocked,
		policyDenied: policyDenied,
		failed:       failed,
		terminal:     terminal,
		legacy:       legacy,
	}
	if err := state.Validate(); err != nil {
		return CanonicalWorkState{}, err
	}
	return state, nil
}

func (s CanonicalWorkState) Phase() PhaseKind       { return s.phase }
func (s CanonicalWorkState) Exec() ExecState        { return s.exec }
func (s CanonicalWorkState) ReadyToRun() bool       { return s.readyToRun }
func (s CanonicalWorkState) Blocked() bool          { return s.blocked }
func (s CanonicalWorkState) PolicyDenied() bool     { return s.policyDenied }
func (s CanonicalWorkState) Failed() bool           { return s.failed }
func (s CanonicalWorkState) Terminal() TerminalKind { return s.terminal }
func (s CanonicalWorkState) Legacy() LegacyMark     { return s.legacy }

func (s CanonicalWorkState) Validate() error {
	if !validPhaseKind(s.phase) {
		return fmt.Errorf("invalid canonical phase %q", s.phase)
	}
	if !validTerminalKind(s.terminal) {
		return fmt.Errorf("invalid canonical terminal %q", s.terminal)
	}
	if !validLegacyMark(s.legacy) {
		return fmt.Errorf("invalid canonical legacy mark %q", s.legacy)
	}

	hasExec := s.exec != ""
	if hasExec {
		if !validExec(s.exec) {
			return fmt.Errorf("invalid canonical exec state %q", s.exec)
		}
	}
	if (s.phase == PhaseExec) != hasExec {
		return fmt.Errorf("canonical exec presence invariant violated for phase %s", s.phase)
	}

	hasTerminal := s.terminal != TerminalNone
	if (s.phase == PhaseTerminal) != hasTerminal {
		return fmt.Errorf("canonical terminal presence invariant violated for phase %s", s.phase)
	}
	if hasTerminal && (s.blocked || s.failed || s.readyToRun) {
		return fmt.Errorf("canonical terminal state cannot carry blocked, failed, or ready overlays")
	}

	if s.policyDenied {
		if !s.blocked {
			return fmt.Errorf("canonical policy denied requires blocked")
		}
		if s.phase != PhaseExec {
			return fmt.Errorf("canonical policy denied requires exec phase")
		}
	}
	if s.readyToRun && s.phase != PhasePreExec {
		return fmt.Errorf("canonical ready_to_run requires pre_exec phase")
	}
	if s.failed && s.phase != PhaseExec {
		return fmt.Errorf("canonical failed requires exec phase")
	}
	if s.legacy != LegacyNone && s.phase != PhasePreExec {
		return fmt.Errorf("canonical legacy mark requires pre_exec phase")
	}
	if s.legacy == LegacyMarkCompleted && s.terminal != TerminalNone {
		return fmt.Errorf("canonical legacy_completed cannot imply terminal certification")
	}
	return nil
}

func (s CanonicalWorkState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(canonicalWorkStateJSON{
		Phase:        s.phase,
		Exec:         s.exec,
		ReadyToRun:   s.readyToRun,
		Blocked:      s.blocked,
		PolicyDenied: s.policyDenied,
		Failed:       s.failed,
		Terminal:     s.terminal,
		Legacy:       s.legacy,
	})
}

func (s *CanonicalWorkState) UnmarshalJSON(data []byte) error {
	var payload canonicalWorkStateJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	next, err := NewCanonicalWorkState(
		payload.Phase,
		payload.Exec,
		payload.ReadyToRun,
		payload.Blocked,
		payload.PolicyDenied,
		payload.Failed,
		payload.Terminal,
		payload.Legacy,
	)
	if err != nil {
		return err
	}
	*s = next
	return nil
}

func validPhaseKind(phase PhaseKind) bool {
	switch phase {
	case PhasePreExec, PhaseExec, PhaseTerminal:
		return true
	default:
		return false
	}
}

func validTerminalKind(terminal TerminalKind) bool {
	switch terminal {
	case TerminalNone, TerminalCertified, TerminalRejected, TerminalSuperseded:
		return true
	default:
		return false
	}
}

func validLegacyMark(legacy LegacyMark) bool {
	switch legacy {
	case LegacyNone, LegacyMarkLegacy, LegacyMarkCompleted:
		return true
	default:
		return false
	}
}
