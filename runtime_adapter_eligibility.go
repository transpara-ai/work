package work

import "strings"

const RuntimeBrokerExternalAdapterEligibilityMode = "event17_runtimebroker_external_adapter_eligibility"

const (
	runtimeAdapterEligibilityWorkIssue       = "transpara-ai/work#64"
	runtimeAdapterEligibilityProtectedAction = "repo.work.runtimebroker_external_adapter_eligibility_evidence.implement"
	runtimeAdapterEligibilityPolicyBundleID  = "policy.runtimebroker.external_adapter.eligibility.event17"
	runtimeAdapterEligibilityPolicyHash      = "sha256:runtimebroker-external-adapter-eligibility-policy"
	runtimeAdapterEligibilityWork59Merge     = "transpara-ai/work@040cf03af8336107cb15aaa9d2a3f6c45031011e"
)

type RuntimeBrokerExternalAdapterEligibilityOptions struct {
	OmitAuthority          bool
	WidenAuthorityClaim    string
	StaleAuthorityRef      bool
	OmitCandidateIdentity  bool
	OmitPolicyBundle       bool
	OmitSourceIssueRef     bool
	MismatchSourceIssueRef bool

	OmitFileBoundary             bool
	WidenFileBoundary            bool
	OmitCommandBoundary          bool
	ShellCommandClaim            bool
	ProcessEscapeClaim           bool
	RuntimeBrokerRunCommandClaim bool
	DeploymentCommandClaim       bool
	GitHubMutationClaim          bool
	HiveActionAPIClaim           bool

	OmitNetworkBoundary    bool
	UnscopedNetworkClaim   bool
	WidenNetworkHostScope  bool
	LiveNetworkClaim       bool
	ValidationNetworkClaim bool

	OmitSecretBoundary       bool
	UnscopedSecretClaim      bool
	CredentialMaterialClaim  bool
	SecretLogClaim           bool
	MissingRedactionRequired bool

	OmitTimeoutBoundary   bool
	UnboundedTimeoutClaim bool
	MissingCancellation   bool
	MissingResourceLimits bool
	RetryWithoutReceipt   bool

	OmitArtifactContract     bool
	PartialArtifactAllowance bool
	MissingArtifactHash      bool
	MissingArtifactContent   bool
	MissingArtifactSizeBound bool
	ProductionArtifactClaim  bool

	OmitExitCodeMapping bool
	AmbiguousExitStatus bool

	OmitReceiptEvidence   bool
	StaleReceiptClaim     bool
	MismatchedReceiptHash bool
	ReceiptBeforeResult   bool

	OmitValidationPlan      bool
	UnboundedValidation     bool
	OmitReplayPlan          bool
	NonDeterministicReplay  bool
	ReplayRequiresNetSecret bool
	ReplayWritesProduction  bool

	AdapterEnablementClaim      bool
	AdapterInvocationClaim      bool
	RuntimeBrokerExecutionClaim bool
	ProductionEventGraphWrite   bool
	ProductionTruthClaim        bool
	RuntimeSideEffectClaim      bool
	ProtectedSettingsClaim      bool
	Test001GreenClaim           bool
	Docs172ClosureClaim         bool
	AutonomyIncreaseClaim       bool
	ValueAllocationClaim        bool
	ResidualRiskClosureClaim    bool
}

type RuntimeBrokerExternalAdapterEligibilityRun struct {
	Mode   string
	Report RuntimeBrokerExternalAdapterEligibilityReport
}

type RuntimeBrokerExternalAdapterEligibilityReport struct {
	Status            string                                   `json:"status"`
	Missing           []string                                 `json:"missing,omitempty"`
	Authorization     RuntimeAdapterEligibilityAuthorization   `json:"authorization"`
	Candidate         RuntimeAdapterEligibilityCandidate       `json:"candidate"`
	AuthorityRefs     []string                                 `json:"authority_refs"`
	SourceRefs        []string                                 `json:"source_refs"`
	PolicyBundle      RuntimeAdapterEligibilityPolicyBundle    `json:"policy_bundle"`
	FileBoundary      RuntimeAdapterEligibilityFileBoundary    `json:"file_boundary"`
	CommandBoundary   RuntimeAdapterEligibilityCommandBoundary `json:"command_boundary"`
	NetworkBoundary   RuntimeAdapterEligibilityNetworkBoundary `json:"network_boundary"`
	SecretBoundary    RuntimeAdapterEligibilitySecretBoundary  `json:"secret_boundary"`
	TimeoutBoundary   RuntimeAdapterEligibilityTimeoutBoundary `json:"timeout_boundary"`
	ArtifactContract  RuntimeAdapterEligibilityArtifacts       `json:"artifact_contract"`
	ExitCodeMapping   RuntimeAdapterEligibilityExitMapping     `json:"exit_code_mapping"`
	ReceiptEvidence   RuntimeAdapterEligibilityReceipt         `json:"receipt_evidence"`
	ValidationPlan    RuntimeAdapterEligibilityValidationPlan  `json:"validation_plan"`
	ReplayPlan        RuntimeAdapterEligibilityReplayPlan      `json:"replay_plan"`
	EventGraphHandoff Event17EventGraphHandoff                 `json:"eventgraph_handoff"`
	ForbiddenActions  []Event17ForbiddenAction                 `json:"forbidden_actions"`
	ResidualRisks     []Event17ResidualRiskState               `json:"residual_risks"`
	CommandLog        []string                                 `json:"command_log,omitempty"`
	NetworkAccessLog  []string                                 `json:"network_access_log,omitempty"`
	SecretAccessLog   []string                                 `json:"secret_access_log,omitempty"`
}

type RuntimeAdapterEligibilityAuthorization struct {
	Status                            string `json:"status"`
	Reason                            string `json:"reason"`
	RequiresSeparateAuthorityDecision bool   `json:"requires_separate_authority_decision"`
}

type RuntimeAdapterEligibilityCandidate struct {
	Status                string `json:"status"`
	AdapterID             string `json:"adapter_id,omitempty"`
	AdapterVersion        string `json:"adapter_version,omitempty"`
	RuntimeClass          string `json:"runtime_class,omitempty"`
	ProtectedActionType   string `json:"protected_action_type,omitempty"`
	EvidenceCompleteOnly  bool   `json:"evidence_complete_only"`
	AdapterEnabled        bool   `json:"adapter_enabled"`
	AdapterInvoked        bool   `json:"adapter_invoked"`
	RuntimeBrokerExecuted bool   `json:"runtimebroker_executed"`
}

type RuntimeAdapterEligibilityPolicyBundle struct {
	Status       string   `json:"status"`
	BundleID     string   `json:"bundle_id,omitempty"`
	BundleHash   string   `json:"bundle_hash,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type RuntimeAdapterEligibilityFileBoundary struct {
	Status               string   `json:"status"`
	AllowedFiles         []string `json:"allowed_files,omitempty"`
	DeniedFiles          []string `json:"denied_files,omitempty"`
	PathTraversalDenied  bool     `json:"path_traversal_denied"`
	ProtectedPathsDenied bool     `json:"protected_paths_denied"`
}

type RuntimeAdapterEligibilityCommandBoundary struct {
	Status                     string   `json:"status"`
	AllowedOperations          []string `json:"allowed_operations,omitempty"`
	DeniedOperations           []string `json:"denied_operations,omitempty"`
	ShellGeneralCommandsDenied bool     `json:"shell_general_commands_denied"`
	ProcessEscapeDenied        bool     `json:"process_escape_denied"`
}

type RuntimeAdapterEligibilityNetworkBoundary struct {
	Status                   string   `json:"status"`
	Scope                    string   `json:"scope,omitempty"`
	AllowedHosts             []string `json:"allowed_hosts,omitempty"`
	LiveNetworkAllowed       bool     `json:"live_network_allowed"`
	ValidationNetworkAllowed bool     `json:"validation_network_allowed"`
}

type RuntimeAdapterEligibilitySecretBoundary struct {
	Status                 string   `json:"status"`
	SecretPolicy           string   `json:"secret_policy,omitempty"`
	AllowedSecretRefs      []string `json:"allowed_secret_refs,omitempty"`
	SecretMaterialIncluded bool     `json:"secret_material_included"`
	SecretLogsAllowed      bool     `json:"secret_logs_allowed"`
	RedactionRequired      bool     `json:"redaction_required"`
}

type RuntimeAdapterEligibilityTimeoutBoundary struct {
	Status               string         `json:"status"`
	Timeout              string         `json:"timeout,omitempty"`
	CancellationRequired bool           `json:"cancellation_required"`
	ResourceLimits       map[string]any `json:"resource_limits,omitempty"`
	RetryRequiresReceipt bool           `json:"retry_requires_receipt"`
}

type RuntimeAdapterEligibilityArtifacts struct {
	Status                  string   `json:"status"`
	ExpectedArtifacts       []string `json:"expected_artifacts,omitempty"`
	HashRequired            bool     `json:"hash_required"`
	ContentTypeRequired     bool     `json:"content_type_required"`
	SizeBoundsRequired      bool     `json:"size_bounds_required"`
	PartialArtifactsAllowed bool     `json:"partial_artifacts_allowed"`
	ProductionArtifacts     bool     `json:"production_artifacts"`
}

type RuntimeAdapterEligibilityExitMapping struct {
	Status               string         `json:"status"`
	ExitCodes            map[int]string `json:"exit_codes,omitempty"`
	AmbiguousExitAllowed bool           `json:"ambiguous_exit_allowed"`
}

type RuntimeAdapterEligibilityReceipt struct {
	Status               string   `json:"status"`
	ReceiptSchema        string   `json:"receipt_schema,omitempty"`
	ReceiptHash          string   `json:"receipt_hash,omitempty"`
	ExpectedReceiptHash  string   `json:"expected_receipt_hash,omitempty"`
	HashRequired         bool     `json:"hash_required"`
	HashMatches          bool     `json:"hash_matches"`
	StaleReceiptAllowed  bool     `json:"stale_receipt_allowed"`
	ReceiptBeforeResult  bool     `json:"receipt_before_result"`
	RequiredEvidenceRefs []string `json:"required_evidence_refs,omitempty"`
}

type RuntimeAdapterEligibilityValidationPlan struct {
	Status      string   `json:"status"`
	Steps       []string `json:"steps,omitempty"`
	OfflineOnly bool     `json:"offline_only"`
	Bounded     bool     `json:"bounded"`
}

type RuntimeAdapterEligibilityReplayPlan struct {
	Status                 string   `json:"status"`
	Steps                  []string `json:"steps,omitempty"`
	Deterministic          bool     `json:"deterministic"`
	RequiresNetworkSecrets bool     `json:"requires_network_secrets"`
	WritesProductionState  bool     `json:"writes_production_state"`
}

func RunRuntimeBrokerExternalAdapterEligibilityFixture(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeBrokerExternalAdapterEligibilityRun {
	report := runtimeBrokerExternalAdapterEligibilityReport(opts)
	return RuntimeBrokerExternalAdapterEligibilityRun{
		Mode:   RuntimeBrokerExternalAdapterEligibilityMode,
		Report: report,
	}
}

func RuntimeBrokerExternalAdapterEligibilityStatus(run RuntimeBrokerExternalAdapterEligibilityRun) (string, error) {
	if run.Mode != RuntimeBrokerExternalAdapterEligibilityMode {
		return "fail", errRuntimeAdapterEligibility("runtime adapter eligibility mode is missing")
	}
	if run.Report.Status == "" {
		return "fail", errRuntimeAdapterEligibility("runtime adapter eligibility report is missing")
	}
	if run.Report.Status != "pass" {
		return run.Report.Status, errRuntimeAdapterEligibility("runtime adapter eligibility incomplete: " + strings.Join(run.Report.Missing, "; "))
	}
	return "pass", nil
}

func runtimeBrokerExternalAdapterEligibilityReport(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeBrokerExternalAdapterEligibilityReport {
	report := RuntimeBrokerExternalAdapterEligibilityReport{
		Status:            "pass",
		Authorization:     runtimeAdapterEligibilityAuthorization(),
		Candidate:         runtimeAdapterEligibilityCandidate(opts),
		AuthorityRefs:     runtimeAdapterEligibilityAuthorityRefs(opts),
		SourceRefs:        runtimeAdapterEligibilitySourceRefs(opts),
		PolicyBundle:      runtimeAdapterEligibilityPolicyBundle(opts),
		FileBoundary:      runtimeAdapterEligibilityFileBoundary(opts),
		CommandBoundary:   runtimeAdapterEligibilityCommandBoundary(opts),
		NetworkBoundary:   runtimeAdapterEligibilityNetworkBoundary(opts),
		SecretBoundary:    runtimeAdapterEligibilitySecretBoundary(opts),
		TimeoutBoundary:   runtimeAdapterEligibilityTimeoutBoundary(opts),
		ArtifactContract:  runtimeAdapterEligibilityArtifacts(opts),
		ExitCodeMapping:   runtimeAdapterEligibilityExitMapping(opts),
		ReceiptEvidence:   runtimeAdapterEligibilityReceipt(opts),
		ValidationPlan:    runtimeAdapterEligibilityValidationPlan(opts),
		ReplayPlan:        runtimeAdapterEligibilityReplayPlan(opts),
		EventGraphHandoff: runtimeAdapterEligibilityEventGraphHandoff(opts),
		ForbiddenActions:  runtimeAdapterEligibilityForbiddenActions(opts),
		ResidualRisks:     runtimeAdapterEligibilityResidualRisks(),
	}
	report.Missing = runtimeAdapterEligibilityMissing(opts, report)
	if len(report.Missing) > 0 {
		report.Status = "fail"
		report.EventGraphHandoff.Status = "blocked"
		report.EventGraphHandoff.BlockedBy = append([]string(nil), report.Missing...)
	}
	return report
}

func runtimeAdapterEligibilityAuthorization() RuntimeAdapterEligibilityAuthorization {
	return RuntimeAdapterEligibilityAuthorization{
		Status:                            "none",
		Reason:                            "evidence complete only; external adapter execution requires a separate future AuthorityDecision",
		RequiresSeparateAuthorityDecision: true,
	}
}

func runtimeAdapterEligibilityCandidate(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityCandidate {
	candidate := RuntimeAdapterEligibilityCandidate{
		Status:                "candidate_only",
		AdapterID:             "runtimebroker.external.adapter.fixture",
		AdapterVersion:        "0.0.0-evidence-only",
		RuntimeClass:          "external_runtime_candidate",
		ProtectedActionType:   runtimeAdapterEligibilityProtectedAction,
		EvidenceCompleteOnly:  true,
		AdapterEnabled:        opts.AdapterEnablementClaim,
		AdapterInvoked:        opts.AdapterInvocationClaim,
		RuntimeBrokerExecuted: opts.RuntimeBrokerExecutionClaim,
	}
	if opts.OmitCandidateIdentity {
		candidate.Status = "missing"
		candidate.AdapterID = ""
		candidate.AdapterVersion = ""
		candidate.RuntimeClass = ""
		candidate.ProtectedActionType = ""
		candidate.EvidenceCompleteOnly = false
	}
	return candidate
}

func runtimeAdapterEligibilityAuthorityRefs(opts RuntimeBrokerExternalAdapterEligibilityOptions) []string {
	if opts.OmitAuthority {
		return nil
	}
	refs := []string{event17AuthorityDecisionID, event17DocsPR, event17DocsMergeCommit, runtimeAdapterEligibilityWorkIssue, runtimeAdapterEligibilityWork59Merge}
	if opts.StaleAuthorityRef {
		refs[2] = "stale:transpara-ai/docs#207"
	}
	return refs
}

func runtimeAdapterEligibilitySourceRefs(opts RuntimeBrokerExternalAdapterEligibilityOptions) []string {
	if opts.OmitSourceIssueRef {
		return nil
	}
	ref := runtimeAdapterEligibilityWorkIssue
	if opts.MismatchSourceIssueRef {
		ref = "transpara-ai/work#59"
	}
	return []string{ref, "docs#200", "transpara-ai/docs#207"}
}

func runtimeAdapterEligibilityPolicyBundle(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityPolicyBundle {
	if opts.OmitPolicyBundle {
		return RuntimeAdapterEligibilityPolicyBundle{Status: "missing"}
	}
	return RuntimeAdapterEligibilityPolicyBundle{
		Status:       "recorded",
		BundleID:     runtimeAdapterEligibilityPolicyBundleID,
		BundleHash:   runtimeAdapterEligibilityPolicyHash,
		EvidenceRefs: runtimeAdapterEligibilitySourceRefs(opts),
	}
}

func runtimeAdapterEligibilityFileBoundary(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityFileBoundary {
	if opts.OmitFileBoundary {
		return RuntimeAdapterEligibilityFileBoundary{Status: "missing"}
	}
	return RuntimeAdapterEligibilityFileBoundary{
		Status:               "bounded",
		AllowedFiles:         []string{"adapter-manifest.json", "artifacts/runtimebroker/eligibility/**"},
		DeniedFiles:          runtimeAdapterEligibilityDeniedFiles(opts),
		PathTraversalDenied:  !opts.WidenFileBoundary,
		ProtectedPathsDenied: !opts.WidenFileBoundary,
	}
}

func runtimeAdapterEligibilityDeniedFiles(opts RuntimeBrokerExternalAdapterEligibilityOptions) []string {
	if opts.WidenFileBoundary {
		return []string{".git/**"}
	}
	return []string{".env", "secrets.env", "../", ".git/**", "production/**", "/etc/**"}
}

func runtimeAdapterEligibilityCommandBoundary(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityCommandBoundary {
	if opts.OmitCommandBoundary {
		return RuntimeAdapterEligibilityCommandBoundary{Status: "missing"}
	}
	return RuntimeAdapterEligibilityCommandBoundary{
		Status:                     "bounded",
		AllowedOperations:          []string{"validate_manifest", "validate_receipt_schema", "validate_replay_plan"},
		DeniedOperations:           runtimeAdapterEligibilityDeniedCommands(opts),
		ShellGeneralCommandsDenied: !opts.ShellCommandClaim,
		ProcessEscapeDenied:        !opts.ProcessEscapeClaim,
	}
}

func runtimeAdapterEligibilityDeniedCommands(opts RuntimeBrokerExternalAdapterEligibilityOptions) []string {
	denied := []string{"sh", "bash", "python -c", "curl", "wget", "gh pr merge", "git push origin main", "kubectl", "terraform", "deploy", "hive.action", "RuntimeBroker.run"}
	if opts.DeploymentCommandClaim {
		return withoutRuntimeAdapterEligibilityString(denied, "deploy")
	}
	if opts.GitHubMutationClaim {
		return withoutRuntimeAdapterEligibilityString(denied, "gh pr merge")
	}
	if opts.HiveActionAPIClaim {
		return withoutRuntimeAdapterEligibilityString(denied, "hive.action")
	}
	if opts.RuntimeBrokerRunCommandClaim {
		return withoutRuntimeAdapterEligibilityString(denied, "RuntimeBroker.run")
	}
	return denied
}

func runtimeAdapterEligibilityNetworkBoundary(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityNetworkBoundary {
	if opts.OmitNetworkBoundary {
		return RuntimeAdapterEligibilityNetworkBoundary{Status: "missing"}
	}
	return RuntimeAdapterEligibilityNetworkBoundary{
		Status:                   "bounded",
		Scope:                    runtimeAdapterEligibilityNetworkScope(opts),
		AllowedHosts:             runtimeAdapterEligibilityAllowedHosts(opts),
		LiveNetworkAllowed:       opts.LiveNetworkClaim,
		ValidationNetworkAllowed: opts.ValidationNetworkClaim,
	}
}

func runtimeAdapterEligibilityNetworkScope(opts RuntimeBrokerExternalAdapterEligibilityOptions) string {
	if opts.UnscopedNetworkClaim {
		return "unscoped"
	}
	return "future_authority_named_hosts_only"
}

func runtimeAdapterEligibilityAllowedHosts(opts RuntimeBrokerExternalAdapterEligibilityOptions) []string {
	if opts.WidenNetworkHostScope {
		return []string{"*"}
	}
	return []string{"adapter.example.invalid"}
}

func runtimeAdapterEligibilitySecretBoundary(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilitySecretBoundary {
	if opts.OmitSecretBoundary {
		return RuntimeAdapterEligibilitySecretBoundary{Status: "missing"}
	}
	return RuntimeAdapterEligibilitySecretBoundary{
		Status:                 "bounded",
		SecretPolicy:           runtimeAdapterEligibilitySecretPolicy(opts),
		AllowedSecretRefs:      []string{"future_authority_named_secret_ref_only"},
		SecretMaterialIncluded: opts.CredentialMaterialClaim,
		SecretLogsAllowed:      opts.SecretLogClaim,
		RedactionRequired:      !opts.MissingRedactionRequired,
	}
}

func runtimeAdapterEligibilitySecretPolicy(opts RuntimeBrokerExternalAdapterEligibilityOptions) string {
	if opts.UnscopedSecretClaim {
		return "unscoped"
	}
	return "none_for_this_pr_scoped_future_authority_only"
}

func runtimeAdapterEligibilityTimeoutBoundary(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityTimeoutBoundary {
	if opts.OmitTimeoutBoundary {
		return RuntimeAdapterEligibilityTimeoutBoundary{Status: "missing"}
	}
	timeout := "30s"
	if opts.UnboundedTimeoutClaim {
		timeout = "unbounded"
	}
	resourceLimits := map[string]any{"max_runtime_seconds": 30, "max_retries": 0, "max_output_bytes": 65536}
	if opts.MissingResourceLimits {
		resourceLimits = nil
	}
	return RuntimeAdapterEligibilityTimeoutBoundary{
		Status:               "bounded",
		Timeout:              timeout,
		CancellationRequired: !opts.MissingCancellation,
		ResourceLimits:       resourceLimits,
		RetryRequiresReceipt: !opts.RetryWithoutReceipt,
	}
}

func runtimeAdapterEligibilityArtifacts(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityArtifacts {
	if opts.OmitArtifactContract {
		return RuntimeAdapterEligibilityArtifacts{Status: "missing"}
	}
	return RuntimeAdapterEligibilityArtifacts{
		Status:                  "bounded",
		ExpectedArtifacts:       []string{"eligibility-report.json", "execution-receipt-schema.json", "replay-plan.json"},
		HashRequired:            !opts.MissingArtifactHash,
		ContentTypeRequired:     !opts.MissingArtifactContent,
		SizeBoundsRequired:      !opts.MissingArtifactSizeBound,
		PartialArtifactsAllowed: opts.PartialArtifactAllowance,
		ProductionArtifacts:     opts.ProductionArtifactClaim,
	}
}

func runtimeAdapterEligibilityExitMapping(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityExitMapping {
	if opts.OmitExitCodeMapping {
		return RuntimeAdapterEligibilityExitMapping{Status: "missing"}
	}
	return RuntimeAdapterEligibilityExitMapping{
		Status: "bounded",
		ExitCodes: map[int]string{
			0:   "succeeded",
			1:   "failed",
			124: "timed_out",
			126: "policy_blocked",
		},
		AmbiguousExitAllowed: opts.AmbiguousExitStatus,
	}
}

func runtimeAdapterEligibilityReceipt(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityReceipt {
	if opts.OmitReceiptEvidence {
		return RuntimeAdapterEligibilityReceipt{Status: "missing"}
	}
	return RuntimeAdapterEligibilityReceipt{
		Status:               "schema_defined",
		ReceiptSchema:        "runtimebroker.external_adapter.execution_receipt.v1",
		ReceiptHash:          runtimeAdapterEligibilityReceiptHash(opts),
		ExpectedReceiptHash:  "sha256:runtimebroker-external-adapter-receipt-schema",
		HashRequired:         true,
		HashMatches:          !opts.MismatchedReceiptHash,
		StaleReceiptAllowed:  opts.StaleReceiptClaim,
		ReceiptBeforeResult:  opts.ReceiptBeforeResult,
		RequiredEvidenceRefs: []string{"authority_decision", "policy_bundle", "runtime_envelope", "runtime_result", "validation_result", "replay_result"},
	}
}

func runtimeAdapterEligibilityReceiptHash(opts RuntimeBrokerExternalAdapterEligibilityOptions) string {
	if opts.MismatchedReceiptHash {
		return "sha256:mismatched-runtimebroker-external-adapter-receipt-schema"
	}
	return "sha256:runtimebroker-external-adapter-receipt-schema"
}

func runtimeAdapterEligibilityValidationPlan(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityValidationPlan {
	if opts.OmitValidationPlan {
		return RuntimeAdapterEligibilityValidationPlan{Status: "missing"}
	}
	return RuntimeAdapterEligibilityValidationPlan{
		Status:      "bounded",
		Steps:       []string{"validate manifest schema", "validate no live invocation", "validate receipt schema", "validate replay plan"},
		OfflineOnly: !opts.ValidationNetworkClaim,
		Bounded:     !opts.UnboundedValidation,
	}
}

func runtimeAdapterEligibilityReplayPlan(opts RuntimeBrokerExternalAdapterEligibilityOptions) RuntimeAdapterEligibilityReplayPlan {
	if opts.OmitReplayPlan {
		return RuntimeAdapterEligibilityReplayPlan{Status: "missing"}
	}
	return RuntimeAdapterEligibilityReplayPlan{
		Status:                 "bounded",
		Steps:                  []string{"load candidate report", "verify hashes", "verify receipt schema", "verify no side effects"},
		Deterministic:          !opts.NonDeterministicReplay,
		RequiresNetworkSecrets: opts.ReplayRequiresNetSecret,
		WritesProductionState:  opts.ReplayWritesProduction,
	}
}

func runtimeAdapterEligibilityEventGraphHandoff(opts RuntimeBrokerExternalAdapterEligibilityOptions) Event17EventGraphHandoff {
	return Event17EventGraphHandoff{
		Status:                 "descriptor_only",
		DescriptorOnly:         true,
		PersistentWriteStatus:  "not_written",
		PersistentWriteClaimed: opts.ProductionEventGraphWrite,
		ProductionTruthClaimed: opts.ProductionTruthClaim,
		EventGraphRefs:         []string{"eg://AuthorityDecision/" + event17AuthorityDecisionID, "eg://WorkIssue/" + runtimeAdapterEligibilityWorkIssue},
		Notes: []string{
			"handoff is a non-executing descriptor",
			"adapter eligibility does not write production EventGraph truth",
			"external adapter execution requires separate future authority",
		},
	}
}

func runtimeAdapterEligibilityForbiddenActions(opts RuntimeBrokerExternalAdapterEligibilityOptions) []Event17ForbiddenAction {
	return []Event17ForbiddenAction{
		{Action: "external adapter enablement", Status: event17ClaimStatus(opts.AdapterEnablementClaim), Reason: "work#64 defines evidence only"},
		{Action: "external adapter invocation", Status: event17ClaimStatus(opts.AdapterInvocationClaim), Reason: "execution requires separate future authority"},
		{Action: "RuntimeBroker execution", Status: event17ClaimStatus(opts.RuntimeBrokerExecutionClaim), Reason: "no runtime path is called by this fixture"},
		{Action: "shell/general command execution", Status: event17ClaimStatus(opts.ShellCommandClaim), Reason: "eligibility validation is data-only"},
		{Action: "network access", Status: event17ClaimStatus(opts.LiveNetworkClaim || opts.ValidationNetworkClaim), Reason: "validation remains local/offline"},
		{Action: "secret access", Status: event17ClaimStatus(opts.CredentialMaterialClaim || opts.SecretLogClaim), Reason: "secrets require separate future authority"},
		{Action: "production EventGraph write", Status: event17ClaimStatus(opts.ProductionEventGraphWrite), Reason: "handoff is descriptor-only"},
		{Action: "production truth claim", Status: event17ClaimStatus(opts.ProductionTruthClaim), Reason: "truth write requires separate EventGraph authority"},
		{Action: "runtime side effect", Status: event17ClaimStatus(opts.RuntimeSideEffectClaim), Reason: "candidate report is in-memory evidence"},
		{Action: "protected settings change", Status: event17ClaimStatus(opts.ProtectedSettingsClaim), Reason: "settings changes are forbidden"},
		{Action: "Test 001 GREEN", Status: event17ClaimStatus(opts.Test001GreenClaim), Reason: "Test 001 is outside work#64"},
		{Action: "docs#172 closure", Status: event17ClaimStatus(opts.Docs172ClosureClaim), Reason: "docs#172 remains outside work#64"},
		{Action: "autonomy increase", Status: event17ClaimStatus(opts.AutonomyIncreaseClaim), Reason: "runtime autonomy is not changed"},
		{Action: "value allocation", Status: event17ClaimStatus(opts.ValueAllocationClaim), Reason: "value allocation is forbidden"},
		{Action: "residual-risk closure", Status: event17ClaimStatus(opts.ResidualRiskClosureClaim), Reason: "risk closure requires separate authority"},
	}
}

func runtimeAdapterEligibilityResidualRisks() []Event17ResidualRiskState {
	return []Event17ResidualRiskState{
		{ID: "R-001", Status: "unresolved_excluded", Reason: "external adapter execution remains unauthorized"},
		{ID: "R-002", Status: "unresolved_excluded", Reason: "production EventGraph writes remain unauthorized"},
		{ID: "R-003", Status: "unresolved_excluded", Reason: "future policy bundle enforcement remains separate governed work"},
	}
}

func runtimeAdapterEligibilityMissing(opts RuntimeBrokerExternalAdapterEligibilityOptions, report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	missing = append(missing, runtimeAdapterEligibilityAuthorityMissing(opts, report)...)
	missing = append(missing, runtimeAdapterEligibilityCandidateMissing(report)...)
	missing = append(missing, runtimeAdapterEligibilityPolicyMissing(report)...)
	missing = append(missing, runtimeAdapterEligibilityBoundaryMissing(report)...)
	missing = append(missing, runtimeAdapterEligibilityForbiddenMissing(report)...)
	return event11UniqueStrings(missing)
}

func runtimeAdapterEligibilityAuthorityMissing(opts RuntimeBrokerExternalAdapterEligibilityOptions, report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	if opts.OmitAuthority {
		missing = append(missing, "authority evidence missing")
	}
	if opts.WidenAuthorityClaim != "" {
		missing = append(missing, "authority claim outside work#64 scope: "+opts.WidenAuthorityClaim)
	}
	for _, ref := range []string{event17AuthorityDecisionID, event17DocsPR, event17DocsMergeCommit, runtimeAdapterEligibilityWorkIssue, runtimeAdapterEligibilityWork59Merge} {
		if !stringIn(ref, report.AuthorityRefs) {
			missing = append(missing, "authority ref missing: "+ref)
		}
	}
	if opts.StaleAuthorityRef {
		missing = append(missing, "stale authority evidence")
	}
	if !stringIn(runtimeAdapterEligibilityWorkIssue, report.SourceRefs) {
		missing = append(missing, "source issue ref missing or mismatched")
	}
	if report.Authorization.Status != "none" || !report.Authorization.RequiresSeparateAuthorityDecision {
		missing = append(missing, "authorization state is not none/requires separate authority")
	}
	return missing
}

func runtimeAdapterEligibilityCandidateMissing(report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	if report.Candidate.Status != "candidate_only" || report.Candidate.AdapterID == "" || report.Candidate.AdapterVersion == "" || report.Candidate.ProtectedActionType == "" {
		missing = append(missing, "candidate identity missing")
	}
	if !report.Candidate.EvidenceCompleteOnly {
		missing = append(missing, "candidate evidence-complete-only status missing")
	}
	if report.Candidate.AdapterEnabled {
		missing = append(missing, "external adapter enablement claim")
	}
	if report.Candidate.AdapterInvoked {
		missing = append(missing, "external adapter invocation claim")
	}
	if report.Candidate.RuntimeBrokerExecuted {
		missing = append(missing, "RuntimeBroker execution claim")
	}
	return missing
}

func runtimeAdapterEligibilityPolicyMissing(report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	if report.PolicyBundle.Status != "recorded" || report.PolicyBundle.BundleID == "" || report.PolicyBundle.BundleHash == "" {
		missing = append(missing, "policy bundle evidence missing")
	}
	if !strings.HasPrefix(report.PolicyBundle.BundleHash, "sha256:") {
		missing = append(missing, "policy bundle hash missing")
	}
	return missing
}

func runtimeAdapterEligibilityBoundaryMissing(report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	if report.FileBoundary.Status != "bounded" {
		missing = append(missing, "file boundary missing")
	}
	for _, denied := range []string{".env", "secrets.env", "../", ".git/**", "production/**"} {
		if !stringIn(denied, report.FileBoundary.DeniedFiles) {
			missing = append(missing, "file boundary denied path missing: "+denied)
		}
	}
	if !report.FileBoundary.PathTraversalDenied || !report.FileBoundary.ProtectedPathsDenied {
		missing = append(missing, "file boundary widened")
	}
	if report.CommandBoundary.Status != "bounded" {
		missing = append(missing, "command/process boundary missing")
	}
	for _, denied := range []string{"sh", "bash", "curl", "gh pr merge", "git push origin main", "deploy", "hive.action", "RuntimeBroker.run"} {
		if !stringIn(denied, report.CommandBoundary.DeniedOperations) {
			missing = append(missing, "command boundary denied operation missing: "+denied)
		}
	}
	if !report.CommandBoundary.ShellGeneralCommandsDenied {
		missing = append(missing, "shell/general command execution claim")
	}
	if !report.CommandBoundary.ProcessEscapeDenied {
		missing = append(missing, "process escape claim")
	}
	if report.NetworkBoundary.Status != "bounded" {
		missing = append(missing, "network boundary missing")
	}
	if report.NetworkBoundary.Scope == "unscoped" || stringIn("*", report.NetworkBoundary.AllowedHosts) {
		missing = append(missing, "network scope widened")
	}
	if report.NetworkBoundary.LiveNetworkAllowed {
		missing = append(missing, "live network access claim")
	}
	if report.NetworkBoundary.ValidationNetworkAllowed {
		missing = append(missing, "validation network access claim")
	}
	if report.SecretBoundary.Status != "bounded" {
		missing = append(missing, "secret boundary missing")
	}
	if report.SecretBoundary.SecretPolicy == "unscoped" {
		missing = append(missing, "secret scope widened")
	}
	if report.SecretBoundary.SecretMaterialIncluded {
		missing = append(missing, "credential material claim")
	}
	if report.SecretBoundary.SecretLogsAllowed {
		missing = append(missing, "secret log claim")
	}
	if !report.SecretBoundary.RedactionRequired {
		missing = append(missing, "secret redaction requirement missing")
	}
	if report.TimeoutBoundary.Status != "bounded" {
		missing = append(missing, "timeout/cancellation boundary missing")
	}
	if report.TimeoutBoundary.Timeout == "unbounded" || report.TimeoutBoundary.Timeout == "" {
		missing = append(missing, "timeout unbounded")
	}
	if !report.TimeoutBoundary.CancellationRequired {
		missing = append(missing, "cancellation evidence missing")
	}
	if len(report.TimeoutBoundary.ResourceLimits) == 0 {
		missing = append(missing, "resource limits missing")
	}
	if !report.TimeoutBoundary.RetryRequiresReceipt {
		missing = append(missing, "retry without receipt claim")
	}
	if report.ArtifactContract.Status != "bounded" {
		missing = append(missing, "artifact contract missing")
	}
	if report.ArtifactContract.PartialArtifactsAllowed {
		missing = append(missing, "partial artifact allowance")
	}
	if !report.ArtifactContract.HashRequired {
		missing = append(missing, "artifact hash requirement missing")
	}
	if !report.ArtifactContract.ContentTypeRequired {
		missing = append(missing, "artifact content-type requirement missing")
	}
	if !report.ArtifactContract.SizeBoundsRequired {
		missing = append(missing, "artifact size bounds missing")
	}
	if report.ArtifactContract.ProductionArtifacts {
		missing = append(missing, "production artifact claim")
	}
	if report.ExitCodeMapping.Status != "bounded" {
		missing = append(missing, "exit-code mapping missing")
	}
	if report.ExitCodeMapping.AmbiguousExitAllowed || len(report.ExitCodeMapping.ExitCodes) == 0 {
		missing = append(missing, "ambiguous exit status claim")
	}
	if report.ReceiptEvidence.Status != "schema_defined" {
		missing = append(missing, "execution receipt evidence missing")
	}
	if !report.ReceiptEvidence.HashRequired {
		missing = append(missing, "execution receipt hash requirement missing")
	}
	if !report.ReceiptEvidence.HashMatches || report.ReceiptEvidence.ReceiptHash != report.ReceiptEvidence.ExpectedReceiptHash {
		missing = append(missing, "execution receipt hash mismatch")
	}
	if report.ReceiptEvidence.StaleReceiptAllowed {
		missing = append(missing, "stale receipt claim")
	}
	if report.ReceiptEvidence.ReceiptBeforeResult {
		missing = append(missing, "receipt recorded before result claim")
	}
	if report.ValidationPlan.Status != "bounded" {
		missing = append(missing, "validation plan missing")
	}
	if !report.ValidationPlan.OfflineOnly {
		missing = append(missing, "validation plan requires network")
	}
	if !report.ValidationPlan.Bounded {
		missing = append(missing, "validation plan unbounded")
	}
	if report.ReplayPlan.Status != "bounded" {
		missing = append(missing, "replay plan missing")
	}
	if !report.ReplayPlan.Deterministic {
		missing = append(missing, "replay plan non-deterministic")
	}
	if report.ReplayPlan.RequiresNetworkSecrets {
		missing = append(missing, "replay requires network or secrets")
	}
	if report.ReplayPlan.WritesProductionState {
		missing = append(missing, "replay writes production state")
	}
	if !report.EventGraphHandoff.DescriptorOnly || report.EventGraphHandoff.PersistentWriteStatus != "not_written" {
		missing = append(missing, "EventGraph handoff is not descriptor-only")
	}
	if report.EventGraphHandoff.PersistentWriteClaimed {
		missing = append(missing, "production EventGraph write claim")
	}
	if report.EventGraphHandoff.ProductionTruthClaimed {
		missing = append(missing, "production truth claim")
	}
	return missing
}

func runtimeAdapterEligibilityForbiddenMissing(report RuntimeBrokerExternalAdapterEligibilityReport) []string {
	var missing []string
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" {
			missing = append(missing, "forbidden action status not fail-closed: "+action.Action)
		}
	}
	if len(report.CommandLog) != 0 {
		missing = append(missing, "command log produced")
	}
	if len(report.NetworkAccessLog) != 0 {
		missing = append(missing, "network log produced")
	}
	if len(report.SecretAccessLog) != 0 {
		missing = append(missing, "secret log produced")
	}
	return missing
}

func errRuntimeAdapterEligibility(msg string) error {
	return runtimeAdapterEligibilityError(msg)
}

type runtimeAdapterEligibilityError string

func (e runtimeAdapterEligibilityError) Error() string {
	return string(e)
}

func withoutRuntimeAdapterEligibilityString(values []string, remove string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != remove {
			out = append(out, value)
		}
	}
	return out
}
