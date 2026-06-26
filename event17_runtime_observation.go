package work

import (
	"errors"
	"fmt"
	"strings"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const Event17GovernedRuntimeObservationMode = "event17_governed_runtime_observation"

const (
	event17AuthorityDecisionID = "DF-V4.0-EPIC-017-AUTHORITY-DECISION"
	event17DocsPR              = "transpara-ai/docs#207"
	event17DocsMergeCommit     = "ad7ecdf69bf6c7f599264c216014c3f2f8ed2f8c"
	event17WorkIssue           = "transpara-ai/work#59"
)

type Event17GovernedRuntimeObservationOptions struct {
	Source         types.ActorID
	ConversationID types.ConversationID
	Causes         []types.EventID
	WorkingDir     string

	OmitAuthority                   bool
	WidenAuthorityClaim             string
	OmitEnvelope                    bool
	OmitRuntimeResult               bool
	OmitPolicyDecision              bool
	OmitTrace                       bool
	WidenTraceScope                 bool
	OmitTestRun                     bool
	OmitGateResult                  bool
	OmitAuditReport                 bool
	MismatchEnvelopeHash            bool
	ObservedRuntimeAdapterID        string
	ObservedNetworkAccessLog        []string
	ObservedSecretAccessLog         []string
	UnsafeNetworkPolicy             string
	UnsafeSecretsPolicy             string
	ExternalAdapterClaim            bool
	ShellCommandClaim               bool
	ProductionEventGraphWriteClaim  bool
	ProductionTruthClaim            bool
	EventGraphHandoffDescriptorLost bool
	EventGraphHandoffWriteStatus    string
	RuntimeSideEffectClaim          bool
	OmitCivilizationPresence        bool
	MalformedCivilizationPresence   bool
	CivilizationRuntimeReadyClaim   bool
	HiveActivityClaim               bool
	IssueClosureAuthorityClaim      bool
	AutonomyIncreaseClaim           bool
}

type Event17GovernedRuntimeObservationRun struct {
	Mode    string
	Event11 Event11RuntimeEnvelopeDryRunRun
	Report  Event17GovernedRuntimeObservationReport
}

type Event17GovernedRuntimeObservationReport struct {
	Status               string                      `json:"status"`
	Missing              []string                    `json:"missing,omitempty"`
	Envelope             Event17EnvelopeObservation  `json:"envelope"`
	Result               Event17ResultObservation    `json:"result"`
	Policy               Event17PolicyObservation    `json:"policy"`
	Trace                Event17TraceObservation     `json:"trace"`
	TestRun              Event17EvidenceObservation  `json:"test_run"`
	GateResult           Event17EvidenceObservation  `json:"gate_result"`
	AuditReport          Event17EvidenceObservation  `json:"audit_report"`
	EventGraphHandoff    Event17EventGraphHandoff    `json:"eventgraph_handoff"`
	CivilizationPresence Event17CivilizationPresence `json:"civilization_presence"`
	ForbiddenActions     []Event17ForbiddenAction    `json:"forbidden_actions"`
	ResidualRisks        []Event17ResidualRiskState  `json:"residual_risks"`
	EvidenceRefs         []string                    `json:"evidence_refs"`
	AuthorityRefs        []string                    `json:"authority_refs"`
}

type Event17EnvelopeObservation struct {
	Status            string   `json:"status"`
	RuntimeEnvelopeID string   `json:"runtime_envelope_id,omitempty"`
	RuntimeAdapterID  string   `json:"runtime_adapter_id,omitempty"`
	Immutable         bool     `json:"immutable"`
	EnvelopeHash      string   `json:"envelope_hash,omitempty"`
	NetworkPolicy     string   `json:"network_policy,omitempty"`
	SecretsPolicy     string   `json:"secrets_policy,omitempty"`
	DeniedCommands    []string `json:"denied_commands,omitempty"`
	DeniedFiles       []string `json:"denied_files,omitempty"`
	ObservationScope  string   `json:"observation_scope"`
}

type Event17ResultObservation struct {
	Status            string        `json:"status"`
	RuntimeResultID   string        `json:"runtime_result_id,omitempty"`
	RuntimeStatus     RuntimeStatus `json:"runtime_status,omitempty"`
	ChangedFiles      []string      `json:"changed_files,omitempty"`
	Artifacts         []string      `json:"artifacts,omitempty"`
	NetworkAccessLog  []string      `json:"network_access_log,omitempty"`
	SecretAccessLog   []string      `json:"secret_access_log,omitempty"`
	SideEffectClaimed bool          `json:"side_effect_claimed"`
}

type Event17PolicyObservation struct {
	Status                 string                    `json:"status"`
	PolicyDecisionRefs     []string                  `json:"policy_decision_refs,omitempty"`
	PolicyCases            []Event11PolicyCaseResult `json:"policy_cases,omitempty"`
	NetworkDisabled        bool                      `json:"network_disabled"`
	SecretsDenied          bool                      `json:"secrets_denied"`
	ExternalAdapterClaimed bool                      `json:"external_adapter_claimed"`
	ShellCommandClaimed    bool                      `json:"shell_command_claimed"`
}

type Event17TraceObservation struct {
	Status         string                      `json:"status"`
	TraceCompleted bool                        `json:"trace_completed"`
	TraceStatus    v39.TraceCompletenessStatus `json:"trace_status"`
	TraceScope     string                      `json:"trace_scope"`
	TestRunID      string                      `json:"test_run_id,omitempty"`
	GateResultID   string                      `json:"gate_result_id,omitempty"`
	AuditReportID  string                      `json:"audit_report_id,omitempty"`
	EvidenceRefs   []string                    `json:"evidence_refs,omitempty"`
	Missing        []string                    `json:"missing,omitempty"`
}

type Event17EvidenceObservation struct {
	Status       string   `json:"status"`
	ID           string   `json:"id,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type Event17EventGraphHandoff struct {
	Status                 string   `json:"status"`
	DescriptorOnly         bool     `json:"descriptor_only"`
	PersistentWriteStatus  string   `json:"persistent_write_status"`
	PersistentWriteClaimed bool     `json:"persistent_write_claimed"`
	ProductionTruthClaimed bool     `json:"production_truth_claimed"`
	EventGraphRefs         []string `json:"eventgraph_refs,omitempty"`
	BlockedBy              []string `json:"blocked_by,omitempty"`
	Notes                  []string `json:"notes,omitempty"`
}

type Event17CivilizationPresence struct {
	Status                       string `json:"status"`
	MonitoringOnly               bool   `json:"monitoring_only"`
	CivilizationRuntimeReady     bool   `json:"civilization_runtime_ready"`
	HiveActive                   bool   `json:"hive_active"`
	HiveWakeStartClaimed         bool   `json:"hive_wake_start_claimed"`
	IssueClosureAuthorityClaimed bool   `json:"issue_closure_authority_claimed"`
	ProductionTruthClaimed       bool   `json:"production_truth_claimed"`
	AutonomyIncreaseClaimed      bool   `json:"autonomy_increase_claimed"`
}

type Event17ForbiddenAction struct {
	Action string `json:"action"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Event17ResidualRiskState struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func RunEvent17GovernedRuntimeObservationFixture(ts *TaskStore, opts Event17GovernedRuntimeObservationOptions) (Event17GovernedRuntimeObservationRun, error) {
	if ts == nil {
		return Event17GovernedRuntimeObservationRun{}, errors.New("task store is required")
	}
	event11Run, err := RunEvent11RuntimeEnvelopeDryRunFixture(ts, Event11RuntimeEnvelopeDryRunOptions{
		Source:               opts.Source,
		ConversationID:       opts.ConversationID,
		Causes:               opts.Causes,
		WorkingDir:           opts.WorkingDir,
		OmitAuthorityReceipt: opts.OmitAuthority,
		OmitRuntimeResult:    opts.OmitRuntimeResult,
		OmitAuditReport:      opts.OmitAuditReport,
		UnsafeNetworkPolicy:  opts.UnsafeNetworkPolicy,
		UnsafeSecretsPolicy:  opts.UnsafeSecretsPolicy,
	})
	if err != nil {
		return Event17GovernedRuntimeObservationRun{}, err
	}
	report := event17ObservationReport(event11Run, opts)
	return Event17GovernedRuntimeObservationRun{
		Mode:    Event17GovernedRuntimeObservationMode,
		Event11: event11Run,
		Report:  report,
	}, nil
}

func Event17GovernedRuntimeObservationStatus(run Event17GovernedRuntimeObservationRun) (string, error) {
	if run.Mode != Event17GovernedRuntimeObservationMode {
		return "fail", errors.New("event17 governed runtime observation mode is missing")
	}
	if run.Report.Status == "" {
		return "fail", errors.New("event17 governed runtime observation report is missing")
	}
	if run.Report.Status != "pass" {
		return run.Report.Status, fmt.Errorf("event17 governed runtime observation incomplete: %s", strings.Join(run.Report.Missing, "; "))
	}
	return "pass", nil
}

func event17ObservationReport(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17GovernedRuntimeObservationReport {
	testRunID := event17MaybeOmittedID(event11Run.TestRunID, opts.OmitTestRun)
	gateResultID := event17MaybeOmittedID(event11Run.GateResultID, opts.OmitGateResult)
	auditReportID := event17MaybeOmittedID(event11Run.AuditReportID, opts.OmitAuditReport)
	report := Event17GovernedRuntimeObservationReport{
		Status:               "pass",
		Envelope:             event17EnvelopeObservation(event11Run, opts),
		Result:               event17ResultObservation(event11Run, opts),
		Policy:               event17PolicyObservation(event11Run, opts),
		Trace:                event17TraceObservation(event11Run, opts),
		TestRun:              event17EvidenceObservation("recorded", testRunID, []string{testRunID}),
		GateResult:           event17EvidenceObservation("recorded", gateResultID, []string{gateResultID, testRunID}),
		AuditReport:          event17EvidenceObservation("recorded", auditReportID, []string{auditReportID, gateResultID, testRunID}),
		EventGraphHandoff:    event17EventGraphHandoff(event11Run, opts),
		CivilizationPresence: event17CivilizationPresence(opts),
		ForbiddenActions:     event17ForbiddenActions(opts),
		ResidualRisks:        event17ResidualRisks(),
		EvidenceRefs:         event17EvidenceRefs(event11Run),
		AuthorityRefs:        event17AuthorityRefs(),
	}
	report.Missing = append(report.Missing, event17Missing(event11Run, opts, report)...)
	if len(report.Missing) > 0 {
		report.Status = "fail"
		report.EventGraphHandoff.Status = "blocked"
		report.EventGraphHandoff.BlockedBy = append([]string(nil), report.Missing...)
		report.Trace.Status = "fail"
		report.Trace.Missing = append([]string(nil), report.Missing...)
	}
	return report
}

func event17EnvelopeObservation(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17EnvelopeObservation {
	status := "recorded"
	id := event11Run.RuntimeEnvelopeID
	hash := event11Run.Report.EnvelopeHash
	immutable := event11Run.Report.EnvelopeImmutable
	observed := event17ObservedRuntimeEnvelope(event11Run)
	runtimeAdapterID := ""
	networkPolicy := ""
	secretsPolicy := ""
	var deniedCommands []string
	var deniedFiles []string
	if observed != nil {
		runtimeAdapterID = observed.RuntimeAdapterID
		networkPolicy = observed.NetworkPolicy
		secretsPolicy = observed.SecretsPolicy
		deniedCommands = append([]string(nil), observed.DeniedCommands...)
		deniedFiles = append([]string(nil), observed.DeniedFiles...)
	}
	if strings.TrimSpace(opts.ObservedRuntimeAdapterID) != "" {
		runtimeAdapterID = opts.ObservedRuntimeAdapterID
	}
	if opts.OmitEnvelope {
		status = "missing"
		id = ""
		hash = ""
		immutable = false
		runtimeAdapterID = ""
		networkPolicy = ""
		secretsPolicy = ""
		deniedCommands = nil
		deniedFiles = nil
	}
	if opts.MismatchEnvelopeHash {
		hash = "sha256:mismatched-event17-envelope"
	}
	return Event17EnvelopeObservation{
		Status:            status,
		RuntimeEnvelopeID: id,
		RuntimeAdapterID:  runtimeAdapterID,
		Immutable:         immutable,
		EnvelopeHash:      hash,
		NetworkPolicy:     networkPolicy,
		SecretsPolicy:     secretsPolicy,
		DeniedCommands:    deniedCommands,
		DeniedFiles:       deniedFiles,
		ObservationScope:  "local_fixture_observation_only",
	}
}

func event17ResultObservation(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17ResultObservation {
	status := "recorded"
	id := event11Run.RuntimeResultID
	runtimeStatus := event11Run.RuntimeRun.Result.Result.Status
	changed := event17FileArtifactPaths(event11Run.RuntimeRun.Result.Result.ChangedFiles)
	artifacts := event17FileArtifactPaths(event11Run.RuntimeRun.Result.Result.Artifacts)
	observed := event17ObservedRuntimeResult(event11Run)
	var networkLog []string
	var secretLog []string
	if observed != nil {
		networkLog = append([]string(nil), observed.NetworkAccessLog...)
		secretLog = append([]string(nil), observed.SecretAccessLog...)
	}
	networkLog = append(networkLog, opts.ObservedNetworkAccessLog...)
	secretLog = append(secretLog, opts.ObservedSecretAccessLog...)
	if opts.OmitRuntimeResult {
		status = "missing"
		id = ""
		runtimeStatus = ""
		changed = nil
		artifacts = nil
		networkLog = nil
		secretLog = nil
	}
	return Event17ResultObservation{
		Status:            status,
		RuntimeResultID:   id,
		RuntimeStatus:     runtimeStatus,
		ChangedFiles:      changed,
		Artifacts:         artifacts,
		NetworkAccessLog:  event17UniqueStrings(networkLog),
		SecretAccessLog:   event17UniqueStrings(secretLog),
		SideEffectClaimed: opts.RuntimeSideEffectClaim,
	}
}

func event17PolicyObservation(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17PolicyObservation {
	status := "recorded"
	refs := event17UniqueStrings(append([]string{event11Run.AuthorityDecisionID}, event17AuthorityRefs()...))
	cases := append([]Event11PolicyCaseResult(nil), event11Run.PolicyCases...)
	if opts.OmitPolicyDecision {
		status = "missing"
		refs = nil
	}
	return Event17PolicyObservation{
		Status:                 status,
		PolicyDecisionRefs:     refs,
		PolicyCases:            cases,
		NetworkDisabled:        event17PolicyIsDisabled(event11Run, opts),
		SecretsDenied:          event17SecretsAreDenied(event11Run, opts),
		ExternalAdapterClaimed: opts.ExternalAdapterClaim,
		ShellCommandClaimed:    opts.ShellCommandClaim,
	}
}

func event17TraceObservation(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17TraceObservation {
	status := "recorded"
	completed := event11Run.Report.TraceCompleted
	traceStatus := event11Run.Report.TraceStatus
	scope := "event17_governed_runtime_observation_local_fixture"
	refs := append([]string(nil), event11Run.Report.TraceOutput.EvidenceRefs...)
	testRunID := event11Run.TestRunID
	gateResultID := event11Run.GateResultID
	auditReportID := event11Run.AuditReportID
	if opts.OmitTrace {
		status = "missing"
		completed = false
		traceStatus = v39.TraceCompletenessFailed
		refs = nil
	}
	if opts.WidenTraceScope {
		scope = "live_runtime_or_production_eventgraph"
	}
	if opts.OmitTestRun {
		testRunID = ""
	}
	if opts.OmitGateResult {
		gateResultID = ""
	}
	if opts.OmitAuditReport {
		auditReportID = ""
	}
	return Event17TraceObservation{
		Status:         status,
		TraceCompleted: completed,
		TraceStatus:    traceStatus,
		TraceScope:     scope,
		TestRunID:      testRunID,
		GateResultID:   gateResultID,
		AuditReportID:  auditReportID,
		EvidenceRefs:   refs,
	}
}

func event17EvidenceObservation(status, id string, refs []string) Event17EvidenceObservation {
	if id == "" {
		status = "missing"
		refs = nil
	}
	return Event17EvidenceObservation{Status: status, ID: id, EvidenceRefs: event11UniqueStrings(refs)}
}

func event17EventGraphHandoff(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) Event17EventGraphHandoff {
	persistentWriteStatus := "not_written"
	if strings.TrimSpace(opts.EventGraphHandoffWriteStatus) != "" {
		persistentWriteStatus = opts.EventGraphHandoffWriteStatus
	}
	return Event17EventGraphHandoff{
		Status:                 "descriptor_only",
		DescriptorOnly:         !opts.EventGraphHandoffDescriptorLost,
		PersistentWriteStatus:  persistentWriteStatus,
		PersistentWriteClaimed: opts.ProductionEventGraphWriteClaim,
		ProductionTruthClaimed: opts.ProductionTruthClaim,
		EventGraphRefs:         append([]string(nil), event11Run.Report.EventGraphHandoff.EventGraphRefs...),
		Notes: []string{
			"handoff is a non-executing descriptor",
			"no production EventGraph write is performed",
			"production writes require separate EventGraph authority",
		},
	}
}

func event17CivilizationPresence(opts Event17GovernedRuntimeObservationOptions) Event17CivilizationPresence {
	status := "monitoring_only"
	monitoringOnly := true
	if opts.OmitCivilizationPresence {
		status = "missing"
		monitoringOnly = false
	}
	if opts.MalformedCivilizationPresence {
		status = "malformed"
		monitoringOnly = false
	}
	return Event17CivilizationPresence{
		Status:                       status,
		MonitoringOnly:               monitoringOnly,
		CivilizationRuntimeReady:     opts.CivilizationRuntimeReadyClaim,
		HiveActive:                   opts.HiveActivityClaim,
		HiveWakeStartClaimed:         opts.HiveActivityClaim,
		IssueClosureAuthorityClaimed: opts.IssueClosureAuthorityClaim,
		ProductionTruthClaimed:       opts.ProductionTruthClaim,
		AutonomyIncreaseClaimed:      opts.AutonomyIncreaseClaim,
	}
}

func event17ForbiddenActions(opts Event17GovernedRuntimeObservationOptions) []Event17ForbiddenAction {
	return []Event17ForbiddenAction{
		{Action: "live production EventGraph write", Status: event17ClaimStatus(opts.ProductionEventGraphWriteClaim), Reason: "Event 17 work#59 permits descriptor-only handoff"},
		{Action: "production truth claim", Status: event17ClaimStatus(opts.ProductionTruthClaim), Reason: "production truth requires separate authority"},
		{Action: "external adapter", Status: event17ClaimStatus(opts.ExternalAdapterClaim), Reason: "adapter eligibility is work#64 and invocation remains forbidden"},
		{Action: "shell/general command execution", Status: event17ClaimStatus(opts.ShellCommandClaim), Reason: "runtime operations remain deterministic named primitives"},
		{Action: "Hive wake/start", Status: event17ClaimStatus(opts.HiveActivityClaim), Reason: "monitoring must not wake Hive"},
		{Action: "runtime side effect", Status: event17ClaimStatus(opts.RuntimeSideEffectClaim), Reason: "observation must remain side-effect free"},
		{Action: "issue closure authority", Status: event17ClaimStatus(opts.IssueClosureAuthorityClaim), Reason: "issue closure occurs only through PR merge automation"},
	}
}

func event17ResidualRisks() []Event17ResidualRiskState {
	return []Event17ResidualRiskState{
		{ID: "R-001", Status: "unresolved_excluded", Reason: "protected branch/default-branch mutation remains forbidden"},
		{ID: "R-002", Status: "unresolved_excluded", Reason: "protected side effects and production deploy remain unauthorized"},
		{ID: "R-003", Status: "unresolved_excluded", Reason: "policy-bundle reliance remains future governed work"},
	}
}

func event17Missing(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions, report Event17GovernedRuntimeObservationReport) []string {
	var missing []string
	if event11Run.Report.Status != "pass" {
		missing = append(missing, "Event 11 source fixture incomplete")
		missing = append(missing, event11Run.Report.Missing...)
	}
	if opts.OmitAuthority {
		missing = append(missing, "authority evidence missing")
	}
	if opts.WidenAuthorityClaim != "" {
		missing = append(missing, "authority claim outside Event 17 scope: "+opts.WidenAuthorityClaim)
	}
	for _, ref := range event17AuthorityRefs() {
		if !stringIn(ref, report.AuthorityRefs) {
			missing = append(missing, "Event 17 authority ref missing: "+ref)
		}
	}
	if opts.OmitEnvelope {
		missing = append(missing, "pre-run RuntimeEnvelope observation missing")
	}
	if opts.OmitRuntimeResult {
		missing = append(missing, "RuntimeResult observation missing")
	}
	if opts.OmitPolicyDecision {
		missing = append(missing, "policy decision observation missing")
	}
	if opts.OmitTrace {
		missing = append(missing, "trace evidence missing")
	}
	if opts.WidenTraceScope {
		missing = append(missing, "trace scope widened")
	}
	if opts.OmitTestRun {
		missing = append(missing, "TestRun observation missing")
	}
	if opts.OmitGateResult {
		missing = append(missing, "GateResult observation missing")
	}
	if opts.OmitAuditReport {
		missing = append(missing, "AuditReport observation missing")
	}
	if opts.MismatchEnvelopeHash || report.Envelope.EnvelopeHash != event11Run.Report.EnvelopeHash {
		missing = append(missing, "envelope hash mismatch")
	}
	if report.Envelope.NetworkPolicy != "disabled" || opts.UnsafeNetworkPolicy != "" || !report.Policy.NetworkDisabled {
		missing = append(missing, "network policy widened")
	}
	if report.Envelope.SecretsPolicy != "none" || opts.UnsafeSecretsPolicy != "" || !report.Policy.SecretsDenied {
		missing = append(missing, "secrets policy widened")
	}
	if report.Envelope.RuntimeAdapterID != localDeterministicWorker {
		missing = append(missing, "runtime adapter not local_deterministic")
	}
	for _, denied := range event11DeniedCommands() {
		if !stringIn(denied, report.Envelope.DeniedCommands) {
			missing = append(missing, "runtime envelope denied_commands missing "+denied)
		}
	}
	if len(report.Result.NetworkAccessLog) != 0 {
		missing = append(missing, "RuntimeResult network access observed")
	}
	if len(report.Result.SecretAccessLog) != 0 {
		missing = append(missing, "RuntimeResult secret access observed")
	}
	if opts.ExternalAdapterClaim {
		missing = append(missing, "external adapter claim")
	}
	if opts.ShellCommandClaim {
		missing = append(missing, "shell/general command execution claim")
	}
	if opts.ProductionEventGraphWriteClaim {
		missing = append(missing, "production EventGraph write claim")
	}
	if opts.ProductionTruthClaim {
		missing = append(missing, "production truth claim")
	}
	if opts.RuntimeSideEffectClaim {
		missing = append(missing, "runtime side-effect claim")
	}
	if opts.OmitCivilizationPresence {
		missing = append(missing, "civilization-presence boundary metadata missing")
	}
	if opts.MalformedCivilizationPresence {
		missing = append(missing, "civilization-presence boundary metadata malformed")
	}
	if opts.CivilizationRuntimeReadyClaim {
		missing = append(missing, "civilization runtime readiness claim")
	}
	if opts.HiveActivityClaim {
		missing = append(missing, "Hive activity or wake/start claim")
	}
	if opts.IssueClosureAuthorityClaim {
		missing = append(missing, "issue-closure authority claim")
	}
	if opts.AutonomyIncreaseClaim || report.CivilizationPresence.AutonomyIncreaseClaimed {
		missing = append(missing, "autonomy increase claim")
	}
	if !report.EventGraphHandoff.DescriptorOnly {
		missing = append(missing, "EventGraph handoff descriptor-only invariant missing")
	}
	if report.EventGraphHandoff.PersistentWriteStatus != "not_written" {
		missing = append(missing, "EventGraph handoff persistent write status is not not_written")
	}
	if report.EventGraphHandoff.PersistentWriteClaimed {
		missing = append(missing, "EventGraph handoff persistent write claimed")
	}
	if report.EventGraphHandoff.ProductionTruthClaimed {
		missing = append(missing, "EventGraph handoff production truth claimed")
	}
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" {
			missing = append(missing, "forbidden action status not fail-closed: "+action.Action)
		}
	}
	if report.CivilizationPresence.Status != "monitoring_only" || !report.CivilizationPresence.MonitoringOnly {
		missing = append(missing, "civilization-presence monitoring-only status not asserted")
	}
	return event11UniqueStrings(missing)
}

func event17ObservedRuntimeEnvelope(event11Run Event11RuntimeEnvelopeDryRunRun) *v39.RuntimeEnvelope {
	if event11Run.EventGraph == nil || strings.TrimSpace(event11Run.RuntimeEnvelopeID) == "" {
		return nil
	}
	record, err := event11Run.EventGraph.Get(event11Run.RuntimeEnvelopeID)
	if err != nil {
		return nil
	}
	envelope, ok := record.(*v39.RuntimeEnvelope)
	if !ok {
		return nil
	}
	return envelope
}

func event17ObservedRuntimeResult(event11Run Event11RuntimeEnvelopeDryRunRun) *v39.RuntimeResult {
	if event11Run.EventGraph == nil || strings.TrimSpace(event11Run.RuntimeResultID) == "" {
		return nil
	}
	record, err := event11Run.EventGraph.Get(event11Run.RuntimeResultID)
	if err != nil {
		return nil
	}
	result, ok := record.(*v39.RuntimeResult)
	if !ok {
		return nil
	}
	return result
}

func event17PolicyIsDisabled(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) bool {
	if opts.UnsafeNetworkPolicy != "" {
		return false
	}
	observed := event17ObservedRuntimeEnvelope(event11Run)
	return observed != nil && observed.NetworkPolicy == "disabled"
}

func event17SecretsAreDenied(event11Run Event11RuntimeEnvelopeDryRunRun, opts Event17GovernedRuntimeObservationOptions) bool {
	if opts.UnsafeSecretsPolicy != "" {
		return false
	}
	observed := event17ObservedRuntimeEnvelope(event11Run)
	return observed != nil && observed.SecretsPolicy == "none"
}

func event17AuthorityRefs() []string {
	return []string{
		event17AuthorityDecisionID,
		event17DocsPR,
		event17DocsMergeCommit,
		event17WorkIssue,
	}
}

func event17EvidenceRefs(event11Run Event11RuntimeEnvelopeDryRunRun) []string {
	return event11UniqueStrings([]string{
		event11Run.FactoryOrderID,
		event11Run.TaskID,
		event11Run.AuthorityDecisionID,
		event11Run.RuntimeEnvelopeID,
		event11Run.RuntimeResultID,
		event11Run.TestRunID,
		event11Run.GateResultID,
		event11Run.AuditReportID,
	})
}

func event17FileArtifactPaths(files []RuntimeFileArtifact) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
}

func event17MaybeOmittedID(id string, omitted bool) string {
	if omitted {
		return ""
	}
	return id
}

func event17ClaimStatus(claimed bool) string {
	if claimed {
		return "claimed"
	}
	return "not_run"
}

func event17UniqueStrings(values []string) []string {
	return event11UniqueStrings(values)
}
