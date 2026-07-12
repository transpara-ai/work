package worklifecycle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

const prototypeContractFixtureSHA256 = "42801a522730b4578b26e91c4e8fc4c537e8a916b8fcccc5eda131c259409910"

var (
	headA = Head("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	headB = Head("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func TestTCConstructionInvariant(t *testing.T) {
	macros := []MacroState{
		MacroIntake, MacroDesigning, MacroDesignAudit, MacroAwaitingAuth, MacroCoding,
		MacroCodeReview, MacroReady, MacroMerged, MacroDenied,
	}
	execs := []struct {
		name string
		exec *ExecState
	}{
		{name: "none"},
		{name: "implementing", exec: execPtr(ExecStateImplementing)},
		{name: "self_reviewed", exec: execPtr(ExecStateSelfReviewed)},
		{name: "verified", exec: execPtr(ExecStateVerified)},
		{name: "certified", exec: execPtr(ExecStateCertified)},
		{name: "ready-illegal", exec: execPtr(ExecState("ready"))},
		{name: "blocked-illegal", exec: execPtr(ExecState("blocked"))},
	}
	blocks := []struct {
		name   string
		reason *BlockReason
	}{
		{name: "none"},
		{name: "gate", reason: blockPtr(BlockReasonGate)},
		{name: "resource", reason: blockPtr(BlockReasonResource)},
	}
	classes := []GovernanceClass{
		GovernanceClassProtected,
		GovernanceClassStandard,
		GovernanceClassPrototype,
		GovernanceClassUnknown,
		"",
	}
	heads := []struct {
		name string
		head *Head
	}{
		{name: "none"},
		{name: "reviewed", head: headPtr(headA)},
	}
	gates := []struct {
		name  string
		gates map[Gate]GateRecord
	}{
		{name: "default", gates: defaultGateRecords()},
		{name: "cfar-passed", gates: withGate(defaultGateRecords(), GateCFAR, gateRecord(GatePolicyRequired, GateStatePassed, false, ""))},
		{name: "design-gates-skipped", gates: withGate(withGate(defaultGateRecords(), GateIADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype")), GateCFADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype"))},
		{name: "iada-only-skipped-illegal", gates: withGate(defaultGateRecords(), GateIADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype"))},
		{name: "cfada-only-skipped-illegal", gates: withGate(defaultGateRecords(), GateCFADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype"))},
		{name: "authorize-skipped", gates: withGate(defaultGateRecords(), GateAuthorize, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype"))},
		{name: "iar-skipped-illegal", gates: withGate(defaultGateRecords(), GateIAR, gateRecord(GatePolicyOptional, GateStateFailed, true, "never"))},
		{name: "cfar-skipped-illegal", gates: withGate(defaultGateRecords(), GateCFAR, gateRecord(GatePolicyRequired, GateStateFailed, true, "never"))},
		{name: "skipped-and-passed-illegal", gates: withGate(defaultGateRecords(), GateCFADA, gateRecord(GatePolicyOptional, GateStatePassed, true, "bad"))},
	}

	checked := 0
	for _, macro := range macros {
		for _, exec := range execs {
			for _, block := range blocks {
				for _, class := range classes {
					for _, reviewed := range heads {
						for _, superseded := range []bool{false, true} {
							for _, gateSet := range gates {
								checked++
								fields := unitStateFields{
									Macro:        macro,
									Class:        class,
									Exec:         exec.exec,
									Blocked:      block.reason,
									ReviewedHead: reviewed.head,
									Superseded:   superseded,
									Gates:        gateSet.gates,
								}
								_, err := newUnitState(fields)
								want := constructibleByInvariant(fields)
								if (err == nil) != want {
									t.Fatalf("constructible macro=%s exec=%s block=%s class=%s head=%s gates=%s superseded=%v: err=%v wantConstructible=%v",
										macro, exec.name, block.name, class, reviewed.name, gateSet.name, superseded, err, want)
								}
							}
						}
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("construction table did not run")
	}
}

func TestTCFullDomainReducer(t *testing.T) {
	states := validDomainStates(t)
	events := fullPayloadDomainEvents()
	legalByKind := map[EventKind]int{}

	for _, state := range states {
		for _, ev := range events {
			matches := transitionMatches(state, ev)
			if len(matches) > 1 {
				t.Fatalf("overlapping successors for state=%s class=%s event=%s: %#v", state.Macro(), state.Class(), ev.Kind(), matches)
			}
			got, err := Apply(state, ev)
			if len(matches) == 0 {
				if err == nil {
					t.Fatalf("Apply silently accepted untabled triple: state=%s class=%s event=%s -> %s", state.Macro(), state.Class(), ev.Kind(), got.Macro())
				}
				continue
			}
			if err != nil {
				t.Fatalf("Apply rejected tabled triple: state=%s class=%s event=%s: %v", state.Macro(), state.Class(), ev.Kind(), err)
			}
			if got.EventCount() != state.EventCount()+1 {
				t.Fatalf("Apply did not append exactly one event for state=%s event=%s: got count %d want %d",
					state.Macro(), ev.Kind(), got.EventCount(), state.EventCount()+1)
			}
			legalByKind[ev.Kind()]++
		}
	}

	for _, kind := range []EventKind{
		EventDesignOpened, EventDesignSubmitted, EventCFADASkipped, EventCFADAPassed,
		EventCFADABlocked, EventAuthorityGranted, EventAuthorityDenied,
		EventExecSelfReviewed, EventExecVerified, EventExecCertified, EventCFARPassed,
		EventCFARBlocked, EventMerged, EventBlockedRaised, EventBlockedCleared, EventSuperseded,
	} {
		if legalByKind[kind] == 0 {
			t.Fatalf("full reducer domain never admitted legal transition for %s", kind)
		}
	}
}

func TestTCGateDefaultDeny(t *testing.T) {
	initial, err := Fold(protectedEvidence(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, gate := range []Gate{GateIADA, GateCFADA, GateAuthorize, GateIAR, GateCFAR} {
		record, ok := initial.Gate(gate)
		if !ok {
			t.Fatalf("missing default gate %s", gate)
		}
		if record.State() != GateStateFailed {
			t.Fatalf("gate %s default state=%s, want failed", gate, record.State())
		}
	}

	for _, class := range []GovernanceClass{GovernanceClassProtected, GovernanceClassStandard, GovernanceClassPrototype, GovernanceClassUnknown} {
		designing := mustState(t, unitStateFields{Macro: MacroDesigning, Class: class, Gates: defaultGateRecords()})
		for _, result := range []InternalResult{MissingInternalResult(), BlockingInternalResult(), UnreadableInternalResult()} {
			assertApplyDenied(t, designing, DesignSubmitted(result))
		}

		audit := mustState(t, unitStateFields{Macro: MacroDesignAudit, Class: class, Gates: defaultGateRecords()})
		for _, result := range []CrossFamilyResult{
			{Artifact: ArtifactMissing, AuthorFamily: "claude", ReviewerFamily: "codex"},
			{Artifact: ArtifactUnreadable, AuthorFamily: "claude", ReviewerFamily: "codex"},
			{Artifact: ArtifactPresent, AuthorFamily: "claude", ReviewerFamily: "claude"},
			{Artifact: ArtifactPresent, AuthorFamily: "claude", ReviewerFamily: "codex", BlockerCount: 1},
		} {
			assertApplyDenied(t, audit, CFADAPassed(result))
		}

		awaiting := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: class, Gates: defaultGateRecords()})
		for _, decision := range []AuthorityDecision{
			DeniedAuthority(),
			{Outcome: AuthorityOutcomeGranted},
			{Outcome: AuthorityOutcomeGranted, Scope: []string{""}},
		} {
			assertApplyDenied(t, awaiting, AuthorityGranted(decision))
		}

		codingVerified := mustState(t, unitStateFields{Macro: MacroCoding, Class: class, Exec: execPtr(ExecStateVerified), Gates: defaultGateRecords()})
		for _, result := range []InternalResult{MissingInternalResult(), BlockingInternalResult(), UnreadableInternalResult()} {
			assertApplyDenied(t, codingVerified, ExecCertified(result))
		}

		review := mustState(t, unitStateFields{Macro: MacroCodeReview, Class: class, Gates: defaultGateRecords()})
		for _, result := range []CrossFamilyResult{
			{Artifact: ArtifactMissing, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA},
			{Artifact: ArtifactUnreadable, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA},
			{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "codex", Head: headA},
			{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA, BlockerCount: 1},
			{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "claude"},
		} {
			assertApplyDenied(t, review, CFARPassed(result))
		}
		assertApplyDenied(t, review, Event{kind: EventKind("cfar.skipped"), payload: "not allowed"})
	}
}

func TestTCSkipNotPass(t *testing.T) {
	designing := mustState(t, unitStateFields{Macro: MacroDesigning, Class: GovernanceClassPrototype, Gates: defaultGateRecords()})
	skipped, err := Apply(designing, CFADASkipped("prototype-only"))
	if err != nil {
		t.Fatalf("CFADA skip should be allowed for prototype class: %v", err)
	}
	for _, gate := range []Gate{GateIADA, GateCFADA} {
		record, _ := skipped.Gate(gate)
		if record.State() != GateStateFailed || record.Projection() != GateProjectionSkipped {
			t.Fatalf("%s skipped gate state/projection=%s/%s, want failed/skipped", gate, record.State(), record.Projection())
		}
		reason, ok := record.SkipReason()
		if !ok || reason != "prototype-only" {
			t.Fatalf("%s skip reason=%q ok=%v, want prototype-only true", gate, reason, ok)
		}
		if record.Projection() == GateProjectionPassed {
			t.Fatalf("%s skipped gate projected as passed", gate)
		}
	}

	awaiting := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: GovernanceClassPrototype, Gates: defaultGateRecords()})
	assertApplyDenied(t, awaiting, AuthoritySkipped("prototype-only"))
}

func TestTCClassGatedSkip(t *testing.T) {
	for _, class := range []GovernanceClass{GovernanceClassProtected, GovernanceClassStandard, GovernanceClassUnknown, ""} {
		designing := mustState(t, unitStateFields{Macro: MacroDesigning, Class: class, Gates: defaultGateRecords()})
		assertApplyDenied(t, designing, CFADASkipped("skip"))

		awaiting := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: class, Gates: defaultGateRecords()})
		assertApplyDenied(t, awaiting, AuthoritySkipped("skip"))
	}

	designing := mustState(t, unitStateFields{Macro: MacroDesigning, Class: GovernanceClassPrototype, Gates: defaultGateRecords()})
	assertApplyAllowed(t, designing, CFADASkipped("recorded reason"))
	awaiting := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: GovernanceClassPrototype, Gates: defaultGateRecords()})
	assertApplyDenied(t, awaiting, AuthoritySkipped("recorded reason"))
}

func TestTCPrototypeSkipIsDesignGatesOnly(t *testing.T) {
	state := mustState(t, unitStateFields{
		Macro: MacroDesigning,
		Class: GovernanceClassPrototype,
		Gates: defaultGateRecords(),
	})
	for _, gate := range []Gate{GateAuthorize, GateIAR, GateCFAR} {
		if skipAllowed(state, gate, "bounded reason") {
			t.Fatalf("skipAllowed admitted non-design gate %s", gate)
		}
	}
	for _, gate := range []Gate{GateIADA, GateCFADA} {
		if !skipAllowed(state, gate, "bounded reason") {
			t.Fatalf("skipAllowed rejected design gate %s", gate)
		}
	}
}

func TestTCPrototypeSkipCannotErasePassedIADA(t *testing.T) {
	state, err := Fold(prototypeEvidence(), []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADABlocked(),
	})
	if err != nil {
		t.Fatal(err)
	}
	iada, _ := state.Gate(GateIADA)
	if iada.Projection() != GateProjectionPassed {
		t.Fatalf("IADA projection=%s, want passed", iada.Projection())
	}
	assertApplyDenied(t, state, CFADASkipped("cannot erase passed audit"))
}

func TestTCPrototypeSkipReplayAndReasonInvariant(t *testing.T) {
	events := []Event{DesignOpened(), CFADASkipped("bounded experiment")}
	first, err := Fold(prototypeEvidence(), events)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fold(prototypeEvidence(), events)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Equal(second) {
		t.Fatal("identical prototype skip streams did not replay identically")
	}

	gates := defaultGateRecords()
	gates[GateIADA] = gateRecord(GatePolicyOptional, GateStateFailed, true, "reason-a")
	gates[GateCFADA] = gateRecord(GatePolicyOptional, GateStateFailed, true, "reason-b")
	if _, err := newUnitState(unitStateFields{
		Macro: MacroAwaitingAuth,
		Class: GovernanceClassPrototype,
		Gates: gates,
	}); err == nil || !strings.Contains(err.Error(), "reasons must match") {
		t.Fatalf("mismatched design skip reasons error=%v", err)
	}
}

func TestTCTwoAxisPrototypeContractFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/two-axis-prototype-gates.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != prototypeContractFixtureSHA256 {
		t.Fatalf("prototype contract fixture sha256=%s, want %s", got, prototypeContractFixtureSHA256)
	}
	var fixture struct {
		SchemaVersion    string `json:"schema_version"`
		PlatformContract struct {
			SchemaVersion   string `json:"schema_version"`
			ContractVersion string `json:"contract_version"`
			SourceSHA256    string `json:"source_sha256"`
		} `json:"platform_contract"`
		Activation struct {
			RequiredEvidence         string `json:"required_evidence"`
			SourcePullRequest        int    `json:"source_pull_request"`
			FailClosedBeforeEvidence bool   `json:"fail_closed_before_evidence"`
		} `json:"activation"`
		DocsStandard struct {
			Version       string `json:"version"`
			SourceBlobSHA string `json:"source_blob_sha"`
		} `json:"docs_standard"`
		DecisionVectors []struct {
			ID              string          `json:"id"`
			Maturity        string          `json:"maturity"`
			GovernanceClass GovernanceClass `json:"governance_class"`
			PrototypeReason string          `json:"prototype_reason"`
			IADA            string          `json:"iada"`
			CFADA           string          `json:"cfada"`
			IAR             string          `json:"iar"`
			CFAR            string          `json:"cfar"`
			Blocking        bool            `json:"blocking"`
		} `json:"decision_vectors"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != "work-two-axis-prototype-projection/v1" ||
		fixture.PlatformContract.SchemaVersion != "two-axis-prototype-gates/v1" ||
		fixture.PlatformContract.ContractVersion != "1.0.0" ||
		fixture.PlatformContract.SourceSHA256 != "e5b838a3c60efc9fdc1c23c8e3a908d58f1774af95b653813f6b031500d6ec28" ||
		fixture.Activation.RequiredEvidence != "docs_standard_canonical_on_default_branch" ||
		fixture.Activation.SourcePullRequest != 274 ||
		!fixture.Activation.FailClosedBeforeEvidence ||
		fixture.DocsStandard.Version != "4.4.0" ||
		fixture.DocsStandard.SourceBlobSHA != "c9975c6d5bf703e58dd28ecbe9f7cf91b5ae6b96" {
		t.Fatalf("prototype contract source binding drifted: %#v", fixture)
	}
	if len(fixture.DecisionVectors) != 4 {
		t.Fatalf("decision vector count=%d, want 4", len(fixture.DecisionVectors))
	}
	expected := map[string]struct {
		iada, cfada string
		blocking    bool
	}{
		"prototype-maturity-protected":          {"required", "required", false},
		"established-non-governed-with-reason":  {"optional", "optional", false},
		"prototype-non-governed-without-reason": {"blocked", "blocked", true},
		"prototype-unknown":                     {"blocked", "blocked", true},
	}
	for _, vector := range fixture.DecisionVectors {
		want, ok := expected[vector.ID]
		if !ok {
			t.Fatalf("unexpected decision vector %s", vector.ID)
		}
		if vector.IADA != want.iada || vector.CFADA != want.cfada || vector.Blocking != want.blocking {
			t.Fatalf("%s outcomes=%s/%s blocking=%v, want %s/%s blocking=%v",
				vector.ID, vector.IADA, vector.CFADA, vector.Blocking, want.iada, want.cfada, want.blocking)
		}
		allowed := vector.GovernanceClass == GovernanceClassPrototype && strings.TrimSpace(vector.PrototypeReason) != ""
		if got := vector.IADA == "optional" && vector.CFADA == "optional"; got != allowed {
			t.Fatalf("%s design skip=%v, want %v", vector.ID, got, allowed)
		}
		if vector.IAR != "required" || vector.CFAR != "required" {
			t.Fatalf("%s weakened IAR/CFAR: %s/%s", vector.ID, vector.IAR, vector.CFAR)
		}

		var evidence GovernanceEvidence
		switch vector.GovernanceClass {
		case GovernanceClassProtected:
			evidence.Kinds = []GovernanceEvidenceKind{EvidenceProtectedAction}
		case GovernanceClassStandard:
			evidence.Kinds = []GovernanceEvidenceKind{EvidenceGovernedStandard}
		case GovernanceClassPrototype:
			evidence.Kinds = []GovernanceEvidenceKind{EvidenceNonGovernedPrototype}
			evidence.PrototypeReason = vector.PrototypeReason
		case GovernanceClassUnknown:
			evidence.Uncertain = true
		default:
			t.Fatalf("%s has unsupported governance class %s", vector.ID, vector.GovernanceClass)
		}
		classified := ClassifyGovernance(evidence)
		runtimeBlocked := classified == GovernanceClassUnknown
		if runtimeBlocked != vector.Blocking {
			t.Fatalf("%s runtime blocking=%v (class=%s), want %v",
				vector.ID, runtimeBlocked, classified, vector.Blocking)
		}
		state := mustState(t, unitStateFields{
			Macro: MacroDesigning,
			Class: classified,
			Gates: defaultGateRecords(),
		})
		runtimeOptional := skipAllowed(state, GateIADA, vector.PrototypeReason) &&
			skipAllowed(state, GateCFADA, vector.PrototypeReason)
		declaredOptional := vector.IADA == "optional" && vector.CFADA == "optional"
		if runtimeOptional != declaredOptional {
			t.Fatalf("%s runtime optional=%v, want %v", vector.ID, runtimeOptional, declaredOptional)
		}
		_ = vector.Maturity // maturity is evidence output only; it never enters allowed.
	}
}

func TestTCRetiredAuthoritySkipFailsClosedAtFoldBoundary(t *testing.T) {
	events := []Event{
		DesignOpened(),
		CFADASkipped("bounded experiment"),
		AuthoritySkipped("historical omission"),
	}
	for i := 0; i < 2; i++ {
		_, err := Fold(prototypeEvidence(), events)
		if err == nil || !strings.Contains(err.Error(), "authority.skipped") {
			t.Fatalf("Fold retired authority.skip attempt %d error=%v", i, err)
		}
	}
}

func TestTCClassDerivation(t *testing.T) {
	tests := []struct {
		name string
		in   GovernanceEvidence
		want GovernanceClass
	}{
		{name: "protected action", in: protectedEvidence(), want: GovernanceClassProtected},
		{name: "human gated", in: GovernanceEvidence{Kinds: []GovernanceEvidenceKind{EvidenceHumanAuthorityGated}}, want: GovernanceClassProtected},
		{name: "governed standard", in: GovernanceEvidence{Kinds: []GovernanceEvidenceKind{EvidenceGovernedStandard}}, want: GovernanceClassStandard},
		{name: "prototype with reason", in: GovernanceEvidence{Kinds: []GovernanceEvidenceKind{EvidenceNonGovernedPrototype}, PrototypeReason: "mechanical fixture"}, want: GovernanceClassPrototype},
		{name: "absent", in: GovernanceEvidence{}, want: GovernanceClassUnknown},
		{name: "unreadable", in: GovernanceEvidence{Unreadable: true, Kinds: []GovernanceEvidenceKind{EvidenceProtectedAction}}, want: GovernanceClassUnknown},
		{name: "conflicting", in: GovernanceEvidence{Conflicting: true, Kinds: []GovernanceEvidenceKind{EvidenceProtectedAction, EvidenceNonGovernedPrototype}, PrototypeReason: "conflict"}, want: GovernanceClassUnknown},
		{name: "uncertain", in: GovernanceEvidence{Uncertain: true, Kinds: []GovernanceEvidenceKind{EvidenceGovernedStandard}}, want: GovernanceClassUnknown},
		{name: "unqualified supplied prototype", in: GovernanceEvidence{SuppliedClass: GovernanceClassPrototype}, want: GovernanceClassUnknown},
		{name: "prototype missing reason", in: GovernanceEvidence{Kinds: []GovernanceEvidenceKind{EvidenceNonGovernedPrototype}}, want: GovernanceClassUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyGovernance(tc.in); got != tc.want {
				t.Fatalf("ClassifyGovernance()=%s, want %s", got, tc.want)
			}
		})
	}
}

func TestTCReadyNoStaleCFAR(t *testing.T) {
	events := []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		AuthorityGranted(GrantedAuthority("implement SP0 only")),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
		ExecCertified(PassingInternalResult()),
		CFARPassed(PassingCrossFamilyResult("codex", "claude", headA)),
	}
	state, err := Fold(protectedEvidence(), events)
	if err != nil {
		t.Fatalf("Fold to CFAR pass: %v", err)
	}
	if state.Macro() != MacroCodeReview {
		t.Fatalf("folded macro=%s, want stored CodeReview; Ready must be derived", state.Macro())
	}
	if got, err := Derive(state, headA); err != nil || got != MacroReady {
		t.Fatalf("Derive matching head=%s err=%v, want Ready nil", got, err)
	}
	if got, err := Derive(state, headB); err != nil || got != MacroCodeReview {
		t.Fatalf("Derive stale head=%s err=%v, want CodeReview nil", got, err)
	}

	blocked, err := Apply(state, BlockedRaised(BlockReasonGate))
	if err != nil {
		t.Fatalf("raise block after CFAR pass: %v", err)
	}
	if got, err := Derive(blocked, headA); err != nil || got == MacroReady {
		t.Fatalf("blocked reviewed state derived %s err=%v, want not Ready", got, err)
	}
}

func TestTCInternalZeroBlockerFailclosed(t *testing.T) {
	designing := mustState(t, unitStateFields{Macro: MacroDesigning, Class: GovernanceClassProtected, Gates: defaultGateRecords()})
	for _, tc := range []struct {
		name    string
		result  InternalResult
		allowed bool
	}{
		{name: "absent", result: MissingInternalResult()},
		{name: "blockers", result: BlockingInternalResult()},
		{name: "unreadable", result: UnreadableInternalResult()},
		{name: "zero", result: PassingInternalResult(), allowed: true},
	} {
		t.Run("iada-"+tc.name, func(t *testing.T) {
			assertApplyAllowedState(t, designing, DesignSubmitted(tc.result), tc.allowed)
		})
	}

	coding := mustState(t, unitStateFields{Macro: MacroCoding, Class: GovernanceClassProtected, Exec: execPtr(ExecStateVerified), Gates: defaultGateRecords()})
	for _, tc := range []struct {
		name    string
		result  InternalResult
		allowed bool
	}{
		{name: "absent", result: MissingInternalResult()},
		{name: "blockers", result: BlockingInternalResult()},
		{name: "unreadable", result: UnreadableInternalResult()},
		{name: "zero", result: PassingInternalResult(), allowed: true},
	} {
		t.Run("iar-"+tc.name, func(t *testing.T) {
			assertApplyAllowedState(t, coding, ExecCertified(tc.result), tc.allowed)
		})
	}
}

func TestTCAuthTypedGrant(t *testing.T) {
	awaiting := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: GovernanceClassProtected, Gates: defaultGateRecords()})
	granted, err := Apply(awaiting, AuthorityGranted(GrantedAuthority("implement SP0 exactly")))
	if err != nil {
		t.Fatalf("scoped grant rejected: %v", err)
	}
	if granted.Macro() != MacroCoding {
		t.Fatalf("grant macro=%s, want Coding", granted.Macro())
	}
	if exec, ok := granted.Exec(); !ok || exec != ExecStateImplementing {
		t.Fatalf("grant exec=%s ok=%v, want implementing", exec, ok)
	}
	if !ProtectedActionAuthorized(granted) {
		t.Fatal("scoped grant did not authorize protected action")
	}

	for _, ev := range []Event{
		AuthorityGranted(AuthorityDecision{Outcome: AuthorityOutcomeGranted}),
		AuthorityGranted(DeniedAuthority()),
		{kind: EventAuthorityGranted, payload: true},
	} {
		assertApplyDenied(t, awaiting, ev)
	}

	prototype := mustState(t, unitStateFields{Macro: MacroAwaitingAuth, Class: GovernanceClassPrototype, Gates: defaultGateRecords()})
	assertApplyDenied(t, prototype, AuthoritySkipped("prototype lane"))
	assertApplyDenied(t, awaiting, AuthoritySkipped("governed skip denied"))
}

func TestTCExecSubFSMUnique(t *testing.T) {
	implementing := mustState(t, unitStateFields{Macro: MacroCoding, Class: GovernanceClassProtected, Exec: execPtr(ExecStateImplementing), Gates: defaultGateRecords()})
	selfReviewed, err := Apply(implementing, ExecSelfReviewed())
	if err != nil {
		t.Fatalf("self review: %v", err)
	}
	if exec, _ := selfReviewed.Exec(); exec != ExecStateSelfReviewed {
		t.Fatalf("exec=%s, want self_reviewed", exec)
	}
	assertApplyDenied(t, implementing, ExecVerified(PassingVerificationResult()))
	assertApplyDenied(t, implementing, ExecCertified(PassingInternalResult()))

	verified, err := Apply(selfReviewed, ExecVerified(PassingVerificationResult()))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if exec, _ := verified.Exec(); exec != ExecStateVerified {
		t.Fatalf("exec=%s, want verified", exec)
	}
	assertApplyDenied(t, selfReviewed, ExecCertified(PassingInternalResult()))

	review, err := Apply(verified, ExecCertified(PassingInternalResult()))
	if err != nil {
		t.Fatalf("certify: %v", err)
	}
	if review.Macro() != MacroCodeReview {
		t.Fatalf("certify macro=%s, want CodeReview", review.Macro())
	}
	if exec, ok := review.Exec(); ok {
		t.Fatalf("CodeReview retained exec=%s", exec)
	}

	for _, state := range []UnitState{implementing, selfReviewed, verified} {
		for _, ev := range fullPayloadDomainEvents() {
			got, err := Apply(state, ev)
			if err != nil {
				continue
			}
			if got.Macro() != MacroCoding && ev.Kind() != EventExecCertified {
				t.Fatalf("event %s left Coding from exec state; exec.certified must be unique Coding-leaving edge", ev.Kind())
			}
		}
	}

	reworked, err := Apply(review, CFARBlocked())
	if err != nil {
		t.Fatalf("cfar blocked: %v", err)
	}
	if reworked.Macro() != MacroCoding {
		t.Fatalf("cfar.blocked macro=%s, want Coding", reworked.Macro())
	}
	if exec, ok := reworked.Exec(); !ok || exec != ExecStateImplementing {
		t.Fatalf("cfar.blocked exec=%s ok=%v, want implementing", exec, ok)
	}
}

func TestTCFoldDeterminismAndNoWritableState(t *testing.T) {
	events := []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		AuthorityGranted(GrantedAuthority("bounded implementation")),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
	}
	a, err := Fold(protectedEvidence(), events)
	if err != nil {
		t.Fatalf("fold a: %v", err)
	}
	b, err := Fold(protectedEvidence(), events)
	if err != nil {
		t.Fatalf("fold b: %v", err)
	}
	if !a.Equal(b) || !reflect.DeepEqual(a, b) {
		t.Fatalf("identical event sequences did not fold equal:\na=%#v\nb=%#v", a, b)
	}

	typ := reflect.TypeOf(UnitState{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			t.Fatalf("UnitState field %s is exported/writable", field.Name)
		}
		if strings.EqualFold(field.Name, "currenthead") || strings.EqualFold(field.Name, "current_head") {
			t.Fatalf("UnitState stores current head field %s; readiness must be derived", field.Name)
		}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		if strings.HasPrefix(typ.Method(i).Name, "Set") {
			t.Fatalf("UnitState exposes setter %s", typ.Method(i).Name)
		}
	}

	copied := a.Events()
	if len(copied) == 0 {
		t.Fatal("expected folded event history")
	}
	copied[0] = Superseded()
	again := a.Events()
	if again[0].Kind() != EventDesignOpened {
		t.Fatalf("Events() returned writable backing storage; first event now %s", again[0].Kind())
	}
}

func TestTCSupersedeAppendOnly(t *testing.T) {
	state, err := Fold(protectedEvidence(), []Event{DesignOpened(), DesignSubmitted(PassingInternalResult())})
	if err != nil {
		t.Fatalf("fold setup: %v", err)
	}
	before := state.Events()
	superseded, err := Apply(state, Superseded())
	if err != nil {
		t.Fatalf("superseded: %v", err)
	}
	if !superseded.Superseded() {
		t.Fatal("superseded overlay not set")
	}
	if superseded.EventCount() != len(before)+1 {
		t.Fatalf("event count=%d, want %d", superseded.EventCount(), len(before)+1)
	}
	after := superseded.Events()
	for i := range before {
		if before[i].Kind() != after[i].Kind() {
			t.Fatalf("prior event %d changed from %s to %s", i, before[i].Kind(), after[i].Kind())
		}
	}
}

func TestTCZeroBehaviorChangeAndDeps(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse package imports: %v", err)
	}
	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			for _, spec := range file.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				if path == "github.com/transpara-ai/work" {
					t.Fatalf("%s imports root work package; SP0 must not depend on work.TaskStatus or create a cycle", fileName)
				}
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "TaskStatus" {
					t.Fatalf("%s references TaskStatus; SP1 owns the adapter", fileName)
				}
				return true
			})
		}
	}
}

func validDomainStates(t *testing.T) []UnitState {
	t.Helper()
	macros := []MacroState{
		MacroIntake, MacroDesigning, MacroDesignAudit, MacroAwaitingAuth, MacroCoding,
		MacroCodeReview, MacroReady, MacroMerged, MacroDenied,
	}
	execs := []*ExecState{nil, execPtr(ExecStateImplementing), execPtr(ExecStateSelfReviewed), execPtr(ExecStateVerified), execPtr(ExecStateCertified)}
	blocks := []*BlockReason{nil, blockPtr(BlockReasonGate)}
	classes := []GovernanceClass{GovernanceClassProtected, GovernanceClassStandard, GovernanceClassPrototype, GovernanceClassUnknown}
	heads := []*Head{nil, headPtr(headA)}
	gateSets := []map[Gate]GateRecord{
		defaultGateRecords(),
		withGate(defaultGateRecords(), GateCFAR, gateRecord(GatePolicyRequired, GateStatePassed, false, "")),
		withGate(withGate(defaultGateRecords(), GateCFAR, gateRecord(GatePolicyRequired, GateStatePassed, false, "")), GateAuthorize, gateRecord(GatePolicyOptional, GateStatePassed, false, "")),
		withGate(withGate(defaultGateRecords(), GateIADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype")), GateCFADA, gateRecord(GatePolicyOptional, GateStateFailed, true, "prototype")),
	}

	var states []UnitState
	for _, macro := range macros {
		for _, exec := range execs {
			for _, blocked := range blocks {
				for _, class := range classes {
					for _, head := range heads {
						for _, superseded := range []bool{false, true} {
							for _, gates := range gateSets {
								state, err := newUnitState(unitStateFields{
									Macro:        macro,
									Class:        class,
									Exec:         exec,
									Blocked:      blocked,
									ReviewedHead: head,
									Superseded:   superseded,
									Gates:        gates,
								})
								if err == nil {
									states = append(states, state)
								}
							}
						}
					}
				}
			}
		}
	}
	if len(states) == 0 {
		t.Fatal("no valid domain states generated")
	}
	return states
}

func fullPayloadDomainEvents() []Event {
	return []Event{
		DesignOpened(),
		DesignSubmitted(PassingInternalResult()),
		DesignSubmitted(MissingInternalResult()),
		DesignSubmitted(BlockingInternalResult()),
		DesignSubmitted(UnreadableInternalResult()),
		CFADASkipped("prototype reason"),
		CFADASkipped(""),
		CFADAPassed(PassingCrossFamilyResult("claude", "codex", "")),
		CFADAPassed(CrossFamilyResult{Artifact: ArtifactPresent, AuthorFamily: "claude", ReviewerFamily: "claude"}),
		CFADAPassed(CrossFamilyResult{Artifact: ArtifactMissing, AuthorFamily: "claude", ReviewerFamily: "codex"}),
		CFADAPassed(CrossFamilyResult{Artifact: ArtifactUnreadable, AuthorFamily: "claude", ReviewerFamily: "codex"}),
		CFADAPassed(CrossFamilyResult{Artifact: ArtifactPresent, AuthorFamily: "claude", ReviewerFamily: "codex", BlockerCount: 1}),
		CFADABlocked(),
		AuthorityGranted(GrantedAuthority("bounded")),
		AuthorityGranted(AuthorityDecision{Outcome: AuthorityOutcomeGranted}),
		AuthorityGranted(DeniedAuthority()),
		AuthorityDenied(),
		AuthoritySkipped("prototype reason"),
		AuthoritySkipped(""),
		ExecSelfReviewed(),
		ExecVerified(PassingVerificationResult()),
		ExecVerified(FailingVerificationResult()),
		ExecCertified(PassingInternalResult()),
		ExecCertified(MissingInternalResult()),
		ExecCertified(BlockingInternalResult()),
		ExecCertified(UnreadableInternalResult()),
		CFARPassed(PassingCrossFamilyResult("codex", "claude", headA)),
		CFARPassed(CrossFamilyResult{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "codex", Head: headA}),
		CFARPassed(CrossFamilyResult{Artifact: ArtifactMissing, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA}),
		CFARPassed(CrossFamilyResult{Artifact: ArtifactUnreadable, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA}),
		CFARPassed(CrossFamilyResult{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "claude", Head: headA, BlockerCount: 1}),
		CFARPassed(CrossFamilyResult{Artifact: ArtifactPresent, AuthorFamily: "codex", ReviewerFamily: "claude"}),
		CFARBlocked(),
		Merged(headA, true),
		Merged(headB, true),
		Merged(headA, false),
		BlockedRaised(BlockReasonGate),
		BlockedRaised(""),
		BlockedCleared(),
		Superseded(),
		Event{kind: EventKind("unknown.event"), payload: nil},
	}
}

func constructibleByInvariant(fields unitStateFields) bool {
	class := fields.Class
	if class == "" {
		class = GovernanceClassUnknown
	}
	if !validMacro(fields.Macro) || !validClass(class) {
		return false
	}
	if fields.Exec != nil {
		if fields.Macro != MacroCoding || !validExec(*fields.Exec) {
			return false
		}
	} else if fields.Macro == MacroCoding {
		return false
	}
	if fields.Blocked != nil {
		if !validBlockReason(*fields.Blocked) || !activeMacro(fields.Macro) {
			return false
		}
	}
	if fields.Macro == MacroReady {
		if fields.ReviewedHead == nil || *fields.ReviewedHead == "" {
			return false
		}
		record := mergedGates(fields.Gates)[GateCFAR]
		if record.State() != GateStatePassed {
			return false
		}
	}
	for gate, record := range mergedGates(fields.Gates) {
		if record.Projection() == GateProjectionSkipped {
			if (gate != GateIADA && gate != GateCFADA) || class != GovernanceClassPrototype || record.State() != GateStateFailed {
				return false
			}
			if reason, ok := record.SkipReason(); !ok || strings.TrimSpace(reason) == "" {
				return false
			}
		}
		if record.State() == GateStatePassed && record.Projection() == GateProjectionSkipped {
			return false
		}
	}
	merged := mergedGates(fields.Gates)
	iada, cfada := merged[GateIADA], merged[GateCFADA]
	if (iada.Projection() == GateProjectionSkipped) != (cfada.Projection() == GateProjectionSkipped) {
		return false
	}
	if iada.Projection() == GateProjectionSkipped {
		iadaReason, _ := iada.SkipReason()
		cfadaReason, _ := cfada.SkipReason()
		if iadaReason != cfadaReason {
			return false
		}
	}
	return true
}

func mergedGates(in map[Gate]GateRecord) map[Gate]GateRecord {
	out := defaultGateRecords()
	for gate, record := range in {
		out[gate] = record
	}
	return out
}

func mustState(t *testing.T, fields unitStateFields) UnitState {
	t.Helper()
	state, err := newUnitState(fields)
	if err != nil {
		t.Fatalf("newUnitState(%#v): %v", fields, err)
	}
	return state
}

func assertApplyDenied(t *testing.T, state UnitState, ev Event) {
	t.Helper()
	if got, err := Apply(state, ev); err == nil {
		t.Fatalf("Apply(%s,%s) allowed -> %s; want denied", state.Macro(), ev.Kind(), got.Macro())
	}
}

func assertApplyAllowed(t *testing.T, state UnitState, ev Event) UnitState {
	t.Helper()
	got, err := Apply(state, ev)
	if err != nil {
		t.Fatalf("Apply(%s,%s) denied: %v", state.Macro(), ev.Kind(), err)
	}
	return got
}

func assertApplyAllowedState(t *testing.T, state UnitState, ev Event, allowed bool) {
	t.Helper()
	if allowed {
		assertApplyAllowed(t, state, ev)
		return
	}
	assertApplyDenied(t, state, ev)
}

func protectedEvidence() GovernanceEvidence {
	return GovernanceEvidence{Kinds: []GovernanceEvidenceKind{EvidenceProtectedAction}}
}

func prototypeEvidence() GovernanceEvidence {
	return GovernanceEvidence{
		Kinds:           []GovernanceEvidenceKind{EvidenceNonGovernedPrototype},
		PrototypeReason: "bounded experiment",
	}
}

func withGate(in map[Gate]GateRecord, gate Gate, record GateRecord) map[Gate]GateRecord {
	out := map[Gate]GateRecord{}
	for k, v := range in {
		out[k] = v
	}
	out[gate] = record
	return out
}

func execPtr(exec ExecState) *ExecState        { return &exec }
func blockPtr(reason BlockReason) *BlockReason { return &reason }
func headPtr(head Head) *Head                  { return &head }
