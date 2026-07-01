package worklifecycle

import (
	"fmt"
	"reflect"
	"strings"
)

type MacroState string

const (
	MacroIntake       MacroState = "Intake"
	MacroDesigning    MacroState = "Designing"
	MacroDesignAudit  MacroState = "DesignAudit"
	MacroAwaitingAuth MacroState = "AwaitingAuth"
	MacroCoding       MacroState = "Coding"
	MacroCodeReview   MacroState = "CodeReview"
	MacroReady        MacroState = "Ready"
	MacroMerged       MacroState = "Merged"
	MacroDenied       MacroState = "Denied"
)

type ExecState string

const (
	ExecStateImplementing ExecState = "implementing"
	ExecStateSelfReviewed ExecState = "self_reviewed"
	ExecStateVerified     ExecState = "verified"
	ExecStateCertified    ExecState = "certified"
)

type GovernanceClass string

const (
	GovernanceClassProtected GovernanceClass = "governed_protected"
	GovernanceClassStandard  GovernanceClass = "governed_standard"
	GovernanceClassPrototype GovernanceClass = "non_governed_prototype"
	GovernanceClassUnknown   GovernanceClass = "unknown"
)

type BlockReason string

const (
	BlockReasonGate       BlockReason = "gate"
	BlockReasonResource   BlockReason = "resource"
	BlockReasonAuthority  BlockReason = "authority"
	BlockReasonRecovery   BlockReason = "recovery"
	BlockReasonDependency BlockReason = "dependency"
)

type Head string

type Gate string

const (
	GateIADA      Gate = "IADA"
	GateCFADA     Gate = "CFADA"
	GateAuthorize Gate = "authorize"
	GateIAR       Gate = "IAR"
	GateCFAR      Gate = "CFAR"
)

type GatePolicy string

const (
	GatePolicyUnknown  GatePolicy = "unknown"
	GatePolicyOptional GatePolicy = "optional"
	GatePolicyRequired GatePolicy = "required"
)

type GateState string

const (
	GateStateUnknown GateState = "unknown"
	GateStateFailed  GateState = "failed"
	GateStatePassed  GateState = "passed"
)

type GateProjection string

const (
	GateProjectionFailed  GateProjection = "failed"
	GateProjectionPassed  GateProjection = "passed"
	GateProjectionSkipped GateProjection = "skipped"
)

type GateRecord struct {
	policy     GatePolicy
	state      GateState
	skipped    bool
	skipReason string
}

func (g GateRecord) Policy() GatePolicy { return g.policy }
func (g GateRecord) State() GateState   { return g.state }
func (g GateRecord) Projection() GateProjection {
	if g.skipped {
		return GateProjectionSkipped
	}
	if g.state == GateStatePassed {
		return GateProjectionPassed
	}
	return GateProjectionFailed
}
func (g GateRecord) SkipReason() (string, bool) {
	if !g.skipped {
		return "", false
	}
	return g.skipReason, true
}

func defaultGateRecords() map[Gate]GateRecord {
	return map[Gate]GateRecord{
		GateIADA:      gateRecord(GatePolicyOptional, GateStateFailed, false, ""),
		GateCFADA:     gateRecord(GatePolicyOptional, GateStateFailed, false, ""),
		GateAuthorize: gateRecord(GatePolicyOptional, GateStateFailed, false, ""),
		GateIAR:       gateRecord(GatePolicyOptional, GateStateFailed, false, ""),
		GateCFAR:      gateRecord(GatePolicyRequired, GateStateFailed, false, ""),
	}
}

func gateRecord(policy GatePolicy, state GateState, skipped bool, reason string) GateRecord {
	return GateRecord{
		policy:     policy,
		state:      state,
		skipped:    skipped,
		skipReason: strings.TrimSpace(reason),
	}
}

type UnitState struct {
	macro           MacroState
	class           GovernanceClass
	exec            ExecState
	hasExec         bool
	blocked         BlockReason
	hasBlocked      bool
	reviewedHead    Head
	hasReviewedHead bool
	superseded      bool
	gates           map[Gate]GateRecord
	events          []Event
}

type unitStateFields struct {
	Macro        MacroState
	Class        GovernanceClass
	Exec         *ExecState
	Blocked      *BlockReason
	ReviewedHead *Head
	Superseded   bool
	Gates        map[Gate]GateRecord
	Events       []Event
}

func newUnitState(fields unitStateFields) (UnitState, error) {
	class := fields.Class
	if class == "" {
		class = GovernanceClassUnknown
	}
	if !validMacro(fields.Macro) {
		return UnitState{}, fmt.Errorf("invalid macro state %q", fields.Macro)
	}
	if !validClass(class) {
		return UnitState{}, fmt.Errorf("invalid governance class %q", class)
	}
	hasExec := fields.Exec != nil
	if hasExec != (fields.Macro == MacroCoding) {
		return UnitState{}, fmt.Errorf("exec presence invariant violated for macro %s", fields.Macro)
	}
	var exec ExecState
	if hasExec {
		exec = *fields.Exec
		if !validExec(exec) {
			return UnitState{}, fmt.Errorf("invalid exec state %q", exec)
		}
	}
	hasBlocked := fields.Blocked != nil
	var blocked BlockReason
	if hasBlocked {
		blocked = *fields.Blocked
		if !validBlockReason(blocked) {
			return UnitState{}, fmt.Errorf("invalid block reason %q", blocked)
		}
		if !activeMacro(fields.Macro) {
			return UnitState{}, fmt.Errorf("blocked overlay cannot be set on macro %s", fields.Macro)
		}
	}
	hasReviewedHead := fields.ReviewedHead != nil
	var reviewedHead Head
	if hasReviewedHead {
		reviewedHead = *fields.ReviewedHead
	}

	gates := mergeGateRecords(fields.Gates)
	if err := validateGates(class, fields.Macro, hasReviewedHead, reviewedHead, gates); err != nil {
		return UnitState{}, err
	}

	events := make([]Event, len(fields.Events))
	copy(events, fields.Events)

	return UnitState{
		macro:           fields.Macro,
		class:           class,
		exec:            exec,
		hasExec:         hasExec,
		blocked:         blocked,
		hasBlocked:      hasBlocked,
		reviewedHead:    reviewedHead,
		hasReviewedHead: hasReviewedHead,
		superseded:      fields.Superseded,
		gates:           gates,
		events:          events,
	}, nil
}

func initialState(evidence GovernanceEvidence) (UnitState, error) {
	return newUnitState(unitStateFields{
		Macro: MacroIntake,
		Class: ClassifyGovernance(evidence),
		Gates: defaultGateRecords(),
	})
}

func (s UnitState) Macro() MacroState      { return s.macro }
func (s UnitState) Class() GovernanceClass { return s.class }
func (s UnitState) Exec() (ExecState, bool) {
	if !s.hasExec {
		return "", false
	}
	return s.exec, true
}
func (s UnitState) Blocked() (BlockReason, bool) {
	if !s.hasBlocked {
		return "", false
	}
	return s.blocked, true
}
func (s UnitState) ReviewedHead() (Head, bool) {
	if !s.hasReviewedHead {
		return "", false
	}
	return s.reviewedHead, true
}
func (s UnitState) Superseded() bool { return s.superseded }
func (s UnitState) Gate(gate Gate) (GateRecord, bool) {
	record, ok := s.gates[gate]
	return record, ok
}
func (s UnitState) EventCount() int { return len(s.events) }
func (s UnitState) Events() []Event {
	events := make([]Event, len(s.events))
	copy(events, s.events)
	return events
}
func (s UnitState) Equal(other UnitState) bool {
	return reflect.DeepEqual(s, other)
}

func ProtectedActionAuthorized(state UnitState) bool {
	record, ok := state.Gate(GateAuthorize)
	return ok && record.State() == GateStatePassed
}

func Derive(state UnitState, currentHead Head) (MacroState, error) {
	if err := validateState(state); err != nil {
		return "", err
	}
	if state.macro != MacroCodeReview && state.macro != MacroReady {
		return state.macro, nil
	}
	cfar, ok := state.Gate(GateCFAR)
	if !ok || cfar.State() != GateStatePassed {
		return MacroCodeReview, nil
	}
	if state.hasBlocked || state.superseded {
		return MacroCodeReview, nil
	}
	if state.hasReviewedHead && state.reviewedHead != "" && currentHead == state.reviewedHead {
		return MacroReady, nil
	}
	return MacroCodeReview, nil
}

func Apply(state UnitState, event Event) (UnitState, error) {
	if err := validateState(state); err != nil {
		return UnitState{}, err
	}
	successors := transitionSuccessors(state, event)
	if len(successors) == 0 {
		return UnitState{}, fmt.Errorf("work lifecycle transition denied: state=%s event=%s", state.macro, event.kind)
	}
	if len(successors) > 1 {
		return UnitState{}, fmt.Errorf("work lifecycle transition table overlap: state=%s event=%s matches=%d", state.macro, event.kind, len(successors))
	}
	next := successors[0].state
	next.events = append(copyEvents(state.events), event)
	return next, nil
}

func Fold(evidence GovernanceEvidence, events []Event) (UnitState, error) {
	state, err := initialState(evidence)
	if err != nil {
		return UnitState{}, err
	}
	for i, event := range events {
		state, err = Apply(state, event)
		if err != nil {
			return UnitState{}, fmt.Errorf("fold event %d (%s): %w", i, event.kind, err)
		}
	}
	return state, nil
}

func transitionMatches(state UnitState, event Event) []transitionMatch {
	successors := transitionSuccessors(state, event)
	matches := make([]transitionMatch, len(successors))
	for i, successor := range successors {
		matches[i] = transitionMatch{Name: successor.name}
	}
	return matches
}

type transitionMatch struct {
	Name string
}

type transitionSuccessor struct {
	name  string
	state UnitState
}

type transitionRow struct {
	name  string
	apply func(UnitState, Event) (UnitState, bool)
}

var transitionTable = []transitionRow{
	{
		name: "Intake/design.opened/Designing",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroIntake || event.kind != EventDesignOpened {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroDesigning
			})
		},
	},
	{
		name: "Designing/design.submitted/DesignAudit",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroDesigning || state.hasBlocked || event.kind != EventDesignSubmitted || !internalResultPassed(event.payload) {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroDesignAudit
				fields.Gates[GateIADA] = gateRecord(GatePolicyOptional, GateStatePassed, false, "")
			})
		},
	},
	{
		name: "Designing/cfada.skipped/AwaitingAuth",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroDesigning || state.hasBlocked || event.kind != EventCFADASkipped || !skipAllowed(state, GateCFADA, event.payload) {
				return UnitState{}, false
			}
			reason, _ := skipReason(event.payload)
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroAwaitingAuth
				fields.Gates[GateCFADA] = gateRecord(GatePolicyOptional, GateStateFailed, true, reason)
			})
		},
	},
	{
		name: "DesignAudit/cfada.passed/AwaitingAuth",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroDesignAudit || state.hasBlocked || event.kind != EventCFADAPassed || !crossFamilyPassed(event.payload, false) {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroAwaitingAuth
				fields.Gates[GateCFADA] = gateRecord(GatePolicyOptional, GateStatePassed, false, "")
			})
		},
	},
	{
		name: "DesignAudit/cfada.blocked/Designing",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroDesignAudit || state.hasBlocked || event.kind != EventCFADABlocked {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroDesigning
				fields.Gates[GateCFADA] = gateRecord(GatePolicyOptional, GateStateFailed, false, "")
			})
		},
	},
	{
		name: "AwaitingAuth/authority.granted/Coding",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroAwaitingAuth || state.hasBlocked || event.kind != EventAuthorityGranted || !authorityGrantValid(event.payload) {
				return UnitState{}, false
			}
			exec := ExecStateImplementing
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroCoding
				fields.Exec = &exec
				fields.Gates[GateAuthorize] = gateRecord(GatePolicyOptional, GateStatePassed, false, "")
			})
		},
	},
	{
		name: "AwaitingAuth/authority.denied/Denied",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroAwaitingAuth || state.hasBlocked || event.kind != EventAuthorityDenied {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroDenied
			})
		},
	},
	{
		name: "AwaitingAuth/authority.skipped/Coding",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroAwaitingAuth || state.hasBlocked || event.kind != EventAuthoritySkipped || !skipAllowed(state, GateAuthorize, event.payload) {
				return UnitState{}, false
			}
			reason, _ := skipReason(event.payload)
			exec := ExecStateImplementing
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroCoding
				fields.Exec = &exec
				fields.Gates[GateAuthorize] = gateRecord(GatePolicyOptional, GateStateFailed, true, reason)
			})
		},
	},
	{
		name: "Coding.implementing/exec.self_reviewed/Coding.self_reviewed",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroCoding || state.hasBlocked || !state.hasExec || state.exec != ExecStateImplementing || event.kind != EventExecSelfReviewed {
				return UnitState{}, false
			}
			exec := ExecStateSelfReviewed
			return mutate(state, func(fields *unitStateFields) {
				fields.Exec = &exec
			})
		},
	},
	{
		name: "Coding.self_reviewed/exec.verified/Coding.verified",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroCoding || state.hasBlocked || !state.hasExec || state.exec != ExecStateSelfReviewed || event.kind != EventExecVerified || !verificationPassed(event.payload) {
				return UnitState{}, false
			}
			exec := ExecStateVerified
			return mutate(state, func(fields *unitStateFields) {
				fields.Exec = &exec
			})
		},
	},
	{
		name: "Coding.verified/exec.certified/CodeReview",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroCoding || state.hasBlocked || !state.hasExec || state.exec != ExecStateVerified || event.kind != EventExecCertified || !internalResultPassed(event.payload) {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroCodeReview
				fields.Exec = nil
				fields.Gates[GateIAR] = gateRecord(GatePolicyOptional, GateStatePassed, false, "")
			})
		},
	},
	{
		name: "CodeReview/cfar.passed/bind-reviewed-head",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroCodeReview || state.hasBlocked || event.kind != EventCFARPassed || !crossFamilyPassed(event.payload, true) {
				return UnitState{}, false
			}
			result := event.payload.(CrossFamilyResult)
			return mutate(state, func(fields *unitStateFields) {
				fields.ReviewedHead = &result.Head
				fields.Gates[GateCFAR] = gateRecord(GatePolicyRequired, GateStatePassed, false, "")
			})
		},
	},
	{
		name: "CodeReview/cfar.blocked/Coding.implementing",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if state.macro != MacroCodeReview || state.hasBlocked || event.kind != EventCFARBlocked {
				return UnitState{}, false
			}
			exec := ExecStateImplementing
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroCoding
				fields.Exec = &exec
				fields.ReviewedHead = nil
				fields.Gates[GateIAR] = gateRecord(GatePolicyOptional, GateStateFailed, false, "")
				fields.Gates[GateCFAR] = gateRecord(GatePolicyRequired, GateStateFailed, false, "")
			})
		},
	},
	{
		name: "Ready/merged/Merged",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if event.kind != EventMerged || state.hasBlocked || state.superseded {
				return UnitState{}, false
			}
			if state.macro != MacroCodeReview && state.macro != MacroReady {
				return UnitState{}, false
			}
			payload, ok := event.payload.(mergePayload)
			if !ok || !payload.Approved || payload.Head == "" || !state.hasReviewedHead || payload.Head != state.reviewedHead {
				return UnitState{}, false
			}
			cfar, ok := state.Gate(GateCFAR)
			if !ok || cfar.State() != GateStatePassed {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Macro = MacroMerged
			})
		},
	},
	{
		name: "active/blocked.raised/same-macro",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			reason, ok := event.payload.(BlockReason)
			if event.kind != EventBlockedRaised || !ok || !validBlockReason(reason) || !activeMacro(state.macro) {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Blocked = &reason
			})
		},
	},
	{
		name: "blocked/blocked.cleared/same-macro",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if event.kind != EventBlockedCleared || !state.hasBlocked {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Blocked = nil
			})
		},
	},
	{
		name: "nonterminal/superseded/same-macro",
		apply: func(state UnitState, event Event) (UnitState, bool) {
			if event.kind != EventSuperseded || terminalMacro(state.macro) {
				return UnitState{}, false
			}
			return mutate(state, func(fields *unitStateFields) {
				fields.Superseded = true
			})
		},
	},
}

func transitionSuccessors(state UnitState, event Event) []transitionSuccessor {
	successors := []transitionSuccessor{}
	for _, row := range transitionTable {
		next, ok := row.apply(state, event)
		if ok {
			successors = append(successors, transitionSuccessor{name: row.name, state: next})
		}
	}
	return successors
}

type EventKind string

const (
	EventDesignOpened     EventKind = "design.opened"
	EventDesignSubmitted  EventKind = "design.submitted"
	EventCFADASkipped     EventKind = "cfada.skipped"
	EventCFADAPassed      EventKind = "cfada.passed"
	EventCFADABlocked     EventKind = "cfada.blocked"
	EventAuthorityGranted EventKind = "authority.granted"
	EventAuthorityDenied  EventKind = "authority.denied"
	EventAuthoritySkipped EventKind = "authority.skipped"
	EventExecSelfReviewed EventKind = "exec.self_reviewed"
	EventExecVerified     EventKind = "exec.verified"
	EventExecCertified    EventKind = "exec.certified"
	EventCFARPassed       EventKind = "cfar.passed"
	EventCFARBlocked      EventKind = "cfar.blocked"
	EventMerged           EventKind = "merged"
	EventBlockedRaised    EventKind = "blocked.raised"
	EventBlockedCleared   EventKind = "blocked.cleared"
	EventSuperseded       EventKind = "superseded"
)

type Event struct {
	kind    EventKind
	payload any
}

func (e Event) Kind() EventKind { return e.kind }

func DesignOpened() Event { return Event{kind: EventDesignOpened} }
func DesignSubmitted(result InternalResult) Event {
	return Event{kind: EventDesignSubmitted, payload: result}
}
func CFADASkipped(reason string) Event {
	return Event{kind: EventCFADASkipped, payload: strings.TrimSpace(reason)}
}
func CFADAPassed(result CrossFamilyResult) Event {
	return Event{kind: EventCFADAPassed, payload: result}
}
func CFADABlocked() Event { return Event{kind: EventCFADABlocked} }
func AuthorityGranted(decision AuthorityDecision) Event {
	return Event{kind: EventAuthorityGranted, payload: decision}
}
func AuthorityDenied() Event { return Event{kind: EventAuthorityDenied} }
func AuthoritySkipped(reason string) Event {
	return Event{kind: EventAuthoritySkipped, payload: strings.TrimSpace(reason)}
}
func ExecSelfReviewed() Event { return Event{kind: EventExecSelfReviewed} }
func ExecVerified(result VerificationResult) Event {
	return Event{kind: EventExecVerified, payload: result}
}
func ExecCertified(result InternalResult) Event {
	return Event{kind: EventExecCertified, payload: result}
}
func CFARPassed(result CrossFamilyResult) Event {
	return Event{kind: EventCFARPassed, payload: result}
}
func CFARBlocked() Event { return Event{kind: EventCFARBlocked} }
func Merged(head Head, approved bool) Event {
	return Event{kind: EventMerged, payload: mergePayload{Head: head, Approved: approved}}
}
func BlockedRaised(reason BlockReason) Event {
	return Event{kind: EventBlockedRaised, payload: reason}
}
func BlockedCleared() Event { return Event{kind: EventBlockedCleared} }
func Superseded() Event     { return Event{kind: EventSuperseded} }

type ResultStatus string

const (
	ResultPresent    ResultStatus = "present"
	ResultMissing    ResultStatus = "missing"
	ResultUnreadable ResultStatus = "unreadable"
)

type InternalResult struct {
	Status       ResultStatus
	BlockerCount int
}

func PassingInternalResult() InternalResult {
	return InternalResult{Status: ResultPresent}
}
func MissingInternalResult() InternalResult {
	return InternalResult{Status: ResultMissing}
}
func UnreadableInternalResult() InternalResult {
	return InternalResult{Status: ResultUnreadable}
}
func BlockingInternalResult() InternalResult {
	return InternalResult{Status: ResultPresent, BlockerCount: 1}
}

type ArtifactStatus string

const (
	ArtifactPresent    ArtifactStatus = "present"
	ArtifactMissing    ArtifactStatus = "missing"
	ArtifactUnreadable ArtifactStatus = "unreadable"
)

type CrossFamilyResult struct {
	Artifact       ArtifactStatus
	AuthorFamily   string
	ReviewerFamily string
	BlockerCount   int
	Head           Head
}

func PassingCrossFamilyResult(author, reviewer string, head Head) CrossFamilyResult {
	return CrossFamilyResult{
		Artifact:       ArtifactPresent,
		AuthorFamily:   author,
		ReviewerFamily: reviewer,
		Head:           head,
	}
}

type VerificationResult struct {
	Passed bool
}

func PassingVerificationResult() VerificationResult { return VerificationResult{Passed: true} }
func FailingVerificationResult() VerificationResult { return VerificationResult{} }

type AuthorityOutcome string

const (
	AuthorityOutcomeGranted AuthorityOutcome = "granted"
	AuthorityOutcomeDenied  AuthorityOutcome = "denied"
)

type AuthorityDecision struct {
	Outcome        AuthorityOutcome
	Scope          []string
	Exclusions     []string
	StopConditions []string
	ResidualRisk   []string
}

func GrantedAuthority(scope ...string) AuthorityDecision {
	return AuthorityDecision{Outcome: AuthorityOutcomeGranted, Scope: scope}
}

func DeniedAuthority() AuthorityDecision {
	return AuthorityDecision{Outcome: AuthorityOutcomeDenied}
}

type GovernanceEvidenceKind string

const (
	EvidenceProtectedAction      GovernanceEvidenceKind = "cc:protected-action"
	EvidenceHumanAuthorityGated  GovernanceEvidenceKind = "human-authority-gated"
	EvidenceGovernedStandard     GovernanceEvidenceKind = "governed-standard"
	EvidenceNonGovernedPrototype GovernanceEvidenceKind = "non-governed-prototype"
)

type GovernanceEvidence struct {
	Kinds           []GovernanceEvidenceKind
	PrototypeReason string
	Unreadable      bool
	Conflicting     bool
	Uncertain       bool
	SuppliedClass   GovernanceClass
}

func ClassifyGovernance(evidence GovernanceEvidence) GovernanceClass {
	if evidence.Unreadable || evidence.Conflicting || evidence.Uncertain {
		return GovernanceClassUnknown
	}
	protected := hasEvidence(evidence, EvidenceProtectedAction) || hasEvidence(evidence, EvidenceHumanAuthorityGated)
	standard := hasEvidence(evidence, EvidenceGovernedStandard)
	prototype := hasEvidence(evidence, EvidenceNonGovernedPrototype)

	categories := 0
	for _, present := range []bool{protected, standard, prototype} {
		if present {
			categories++
		}
	}
	if categories != 1 {
		return GovernanceClassUnknown
	}
	switch {
	case protected:
		return GovernanceClassProtected
	case standard:
		return GovernanceClassStandard
	case prototype:
		if strings.TrimSpace(evidence.PrototypeReason) == "" {
			return GovernanceClassUnknown
		}
		return GovernanceClassPrototype
	default:
		return GovernanceClassUnknown
	}
}

func validMacro(macro MacroState) bool {
	switch macro {
	case MacroIntake, MacroDesigning, MacroDesignAudit, MacroAwaitingAuth, MacroCoding, MacroCodeReview, MacroReady, MacroMerged, MacroDenied:
		return true
	default:
		return false
	}
}

func validClass(class GovernanceClass) bool {
	switch class {
	case GovernanceClassProtected, GovernanceClassStandard, GovernanceClassPrototype, GovernanceClassUnknown:
		return true
	default:
		return false
	}
}

func validExec(exec ExecState) bool {
	switch exec {
	case ExecStateImplementing, ExecStateSelfReviewed, ExecStateVerified, ExecStateCertified:
		return true
	default:
		return false
	}
}

func validBlockReason(reason BlockReason) bool {
	switch reason {
	case BlockReasonGate, BlockReasonResource, BlockReasonAuthority, BlockReasonRecovery, BlockReasonDependency:
		return true
	default:
		return false
	}
}

func activeMacro(macro MacroState) bool {
	switch macro {
	case MacroDesigning, MacroDesignAudit, MacroAwaitingAuth, MacroCoding, MacroCodeReview:
		return true
	default:
		return false
	}
}

func terminalMacro(macro MacroState) bool {
	return macro == MacroMerged || macro == MacroDenied
}

func validateState(state UnitState) error {
	_, err := newUnitState(unitStateFields{
		Macro:        state.macro,
		Class:        state.class,
		Exec:         state.execPtr(),
		Blocked:      state.blockPtr(),
		ReviewedHead: state.headPtr(),
		Superseded:   state.superseded,
		Gates:        state.gates,
		Events:       state.events,
	})
	return err
}

func validateGates(class GovernanceClass, macro MacroState, hasReviewedHead bool, reviewedHead Head, gates map[Gate]GateRecord) error {
	for _, gate := range []Gate{GateIADA, GateCFADA, GateAuthorize, GateIAR, GateCFAR} {
		record, ok := gates[gate]
		if !ok {
			return fmt.Errorf("missing gate %s", gate)
		}
		if !validGatePolicy(record.policy) {
			return fmt.Errorf("invalid gate policy %q for %s", record.policy, gate)
		}
		if !validGateState(record.state) {
			return fmt.Errorf("invalid gate state %q for %s", record.state, gate)
		}
		if gate == GateCFAR && record.policy != GatePolicyRequired {
			return fmt.Errorf("CFAR gate policy must be required")
		}
		if gate != GateCFAR && record.policy == GatePolicyRequired {
			return fmt.Errorf("gate %s policy must remain optional in SP0", gate)
		}
		if record.skipped {
			if gate == GateCFAR {
				return fmt.Errorf("CFAR cannot be skipped")
			}
			if class != GovernanceClassPrototype {
				return fmt.Errorf("gate %s skip denied for class %s", gate, class)
			}
			if record.policy != GatePolicyOptional || record.state != GateStateFailed || strings.TrimSpace(record.skipReason) == "" {
				return fmt.Errorf("invalid skipped gate record for %s", gate)
			}
		}
	}
	if macro == MacroReady {
		cfar := gates[GateCFAR]
		if !hasReviewedHead || reviewedHead == "" || cfar.state != GateStatePassed {
			return fmt.Errorf("Ready requires a CFAR-passed reviewed head")
		}
	}
	return nil
}

func validGatePolicy(policy GatePolicy) bool {
	return policy == GatePolicyOptional || policy == GatePolicyRequired
}

func validGateState(state GateState) bool {
	return state == GateStateFailed || state == GateStatePassed
}

func mergeGateRecords(in map[Gate]GateRecord) map[Gate]GateRecord {
	out := defaultGateRecords()
	for gate, record := range in {
		out[gate] = record
	}
	return out
}

func mutate(state UnitState, edit func(*unitStateFields)) (UnitState, bool) {
	fields := state.fields()
	edit(&fields)
	next, err := newUnitState(fields)
	if err != nil {
		return UnitState{}, false
	}
	return next, true
}

func (s UnitState) fields() unitStateFields {
	return unitStateFields{
		Macro:        s.macro,
		Class:        s.class,
		Exec:         s.execPtr(),
		Blocked:      s.blockPtr(),
		ReviewedHead: s.headPtr(),
		Superseded:   s.superseded,
		Gates:        copyGates(s.gates),
		Events:       copyEvents(s.events),
	}
}

func (s UnitState) execPtr() *ExecState {
	if !s.hasExec {
		return nil
	}
	exec := s.exec
	return &exec
}

func (s UnitState) blockPtr() *BlockReason {
	if !s.hasBlocked {
		return nil
	}
	blocked := s.blocked
	return &blocked
}

func (s UnitState) headPtr() *Head {
	if !s.hasReviewedHead {
		return nil
	}
	head := s.reviewedHead
	return &head
}

func copyGates(in map[Gate]GateRecord) map[Gate]GateRecord {
	out := map[Gate]GateRecord{}
	for gate, record := range in {
		out[gate] = record
	}
	return out
}

func copyEvents(in []Event) []Event {
	out := make([]Event, len(in))
	copy(out, in)
	return out
}

func internalResultPassed(payload any) bool {
	result, ok := payload.(InternalResult)
	return ok && result.Status == ResultPresent && result.BlockerCount == 0
}

func crossFamilyPassed(payload any, requireHead bool) bool {
	result, ok := payload.(CrossFamilyResult)
	if !ok {
		return false
	}
	if result.Artifact != ArtifactPresent || result.BlockerCount != 0 {
		return false
	}
	if strings.TrimSpace(result.AuthorFamily) == "" || strings.TrimSpace(result.ReviewerFamily) == "" {
		return false
	}
	if result.AuthorFamily == result.ReviewerFamily {
		return false
	}
	if requireHead && result.Head == "" {
		return false
	}
	return true
}

func verificationPassed(payload any) bool {
	result, ok := payload.(VerificationResult)
	return ok && result.Passed
}

func authorityGrantValid(payload any) bool {
	decision, ok := payload.(AuthorityDecision)
	if !ok || decision.Outcome != AuthorityOutcomeGranted {
		return false
	}
	for _, scope := range decision.Scope {
		if strings.TrimSpace(scope) != "" {
			return true
		}
	}
	return false
}

func skipAllowed(state UnitState, gate Gate, payload any) bool {
	reason, ok := skipReason(payload)
	if !ok || strings.TrimSpace(reason) == "" {
		return false
	}
	if state.class != GovernanceClassPrototype || gate == GateCFAR {
		return false
	}
	record, ok := state.Gate(gate)
	return ok && record.Policy() == GatePolicyOptional
}

func skipReason(payload any) (string, bool) {
	reason, ok := payload.(string)
	if !ok {
		return "", false
	}
	reason = strings.TrimSpace(reason)
	return reason, reason != ""
}

func hasEvidence(evidence GovernanceEvidence, kind GovernanceEvidenceKind) bool {
	for _, candidate := range evidence.Kinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

type mergePayload struct {
	Head     Head
	Approved bool
}
