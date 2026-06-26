package work_test

import (
	"testing"

	"github.com/transpara-ai/work"
)

func TestRuntimeBrokerExternalAdapterEligibilityFixtureBuildsCandidateOnlyEvidence(t *testing.T) {
	run := work.RunRuntimeBrokerExternalAdapterEligibilityFixture(work.RuntimeBrokerExternalAdapterEligibilityOptions{})
	if run.Mode != work.RuntimeBrokerExternalAdapterEligibilityMode {
		t.Fatalf("mode = %q, want %q", run.Mode, work.RuntimeBrokerExternalAdapterEligibilityMode)
	}
	if status, err := work.RuntimeBrokerExternalAdapterEligibilityStatus(run); status != "pass" || err != nil {
		t.Fatalf("status=%q err=%v; want pass nil", status, err)
	}
	report := run.Report
	if report.Status != "pass" || len(report.Missing) != 0 {
		t.Fatalf("report=%#v; want pass with no missing", report)
	}
	if report.Authorization.Status != "none" || !report.Authorization.RequiresSeparateAuthorityDecision {
		t.Fatalf("authorization=%#v; want no authorization and separate decision required", report.Authorization)
	}
	if report.Candidate.Status != "candidate_only" || !report.Candidate.EvidenceCompleteOnly ||
		report.Candidate.AdapterEnabled || report.Candidate.AdapterInvoked || report.Candidate.RuntimeBrokerExecuted {
		t.Fatalf("candidate=%#v; want candidate-only without enable/invoke/run", report.Candidate)
	}
	for _, ref := range []string{
		"DF-V4.0-EPIC-017-AUTHORITY-DECISION",
		"transpara-ai/docs#207",
		"ad7ecdf69bf6c7f599264c216014c3f2f8ed2f8c",
		"transpara-ai/work#64",
		"transpara-ai/work@040cf03af8336107cb15aaa9d2a3f6c45031011e",
	} {
		if !containsString(report.AuthorityRefs, ref) {
			t.Fatalf("authority refs=%#v; want %q", report.AuthorityRefs, ref)
		}
	}
	if !containsString(report.SourceRefs, "transpara-ai/work#64") {
		t.Fatalf("source refs=%#v; want work#64", report.SourceRefs)
	}
	if report.PolicyBundle.Status != "recorded" || report.PolicyBundle.BundleID == "" || report.PolicyBundle.BundleHash == "" {
		t.Fatalf("policy bundle=%#v; want recorded id/hash", report.PolicyBundle)
	}
	if report.FileBoundary.Status != "bounded" || !report.FileBoundary.PathTraversalDenied || !report.FileBoundary.ProtectedPathsDenied {
		t.Fatalf("file boundary=%#v; want bounded", report.FileBoundary)
	}
	if report.CommandBoundary.Status != "bounded" || !report.CommandBoundary.ShellGeneralCommandsDenied || !report.CommandBoundary.ProcessEscapeDenied {
		t.Fatalf("command boundary=%#v; want bounded", report.CommandBoundary)
	}
	if report.NetworkBoundary.Status != "bounded" || report.NetworkBoundary.LiveNetworkAllowed || report.NetworkBoundary.ValidationNetworkAllowed {
		t.Fatalf("network boundary=%#v; want bounded without live/validation network", report.NetworkBoundary)
	}
	if report.SecretBoundary.Status != "bounded" || report.SecretBoundary.SecretMaterialIncluded || report.SecretBoundary.SecretLogsAllowed || !report.SecretBoundary.RedactionRequired {
		t.Fatalf("secret boundary=%#v; want bounded without secret material/logs", report.SecretBoundary)
	}
	if report.TimeoutBoundary.Status != "bounded" || report.TimeoutBoundary.Timeout == "" || !report.TimeoutBoundary.CancellationRequired ||
		len(report.TimeoutBoundary.ResourceLimits) == 0 || !report.TimeoutBoundary.RetryRequiresReceipt {
		t.Fatalf("timeout boundary=%#v; want bounded timeout/cancellation/resource/receipt", report.TimeoutBoundary)
	}
	if report.ArtifactContract.Status != "bounded" || !report.ArtifactContract.HashRequired ||
		!report.ArtifactContract.ContentTypeRequired || !report.ArtifactContract.SizeBoundsRequired ||
		report.ArtifactContract.PartialArtifactsAllowed || report.ArtifactContract.ProductionArtifacts {
		t.Fatalf("artifact contract=%#v; want bounded artifact requirements", report.ArtifactContract)
	}
	if report.ExitCodeMapping.Status != "bounded" || report.ExitCodeMapping.AmbiguousExitAllowed || len(report.ExitCodeMapping.ExitCodes) == 0 {
		t.Fatalf("exit mapping=%#v; want bounded exit mapping", report.ExitCodeMapping)
	}
	if report.ReceiptEvidence.Status != "schema_defined" || !report.ReceiptEvidence.HashRequired ||
		report.ReceiptEvidence.StaleReceiptAllowed || report.ReceiptEvidence.ReceiptBeforeResult {
		t.Fatalf("receipt evidence=%#v; want schema-defined hash-bound receipt", report.ReceiptEvidence)
	}
	if report.ValidationPlan.Status != "bounded" || !report.ValidationPlan.OfflineOnly || !report.ValidationPlan.Bounded {
		t.Fatalf("validation plan=%#v; want bounded offline validation", report.ValidationPlan)
	}
	if report.ReplayPlan.Status != "bounded" || !report.ReplayPlan.Deterministic ||
		report.ReplayPlan.RequiresNetworkSecrets || report.ReplayPlan.WritesProductionState {
		t.Fatalf("replay plan=%#v; want deterministic offline replay", report.ReplayPlan)
	}
	if !report.EventGraphHandoff.DescriptorOnly || report.EventGraphHandoff.PersistentWriteStatus != "not_written" ||
		report.EventGraphHandoff.PersistentWriteClaimed || report.EventGraphHandoff.ProductionTruthClaimed {
		t.Fatalf("eventgraph handoff=%#v; want descriptor-only not-written", report.EventGraphHandoff)
	}
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" {
			t.Fatalf("forbidden action=%#v; want not_run", action)
		}
	}
	if len(report.CommandLog) != 0 || len(report.NetworkAccessLog) != 0 || len(report.SecretAccessLog) != 0 {
		t.Fatalf("logs command=%#v network=%#v secret=%#v; want none", report.CommandLog, report.NetworkAccessLog, report.SecretAccessLog)
	}
}

func TestRuntimeBrokerExternalAdapterEligibilityFixtureFailsClosed(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*work.RuntimeBrokerExternalAdapterEligibilityOptions)
		wantMissing string
	}{
		{name: "missing authority", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitAuthority = true }, wantMissing: "authority evidence missing"},
		{name: "widened authority", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) {
			opts.WidenAuthorityClaim = "external adapter invocation"
		}, wantMissing: "authority claim outside work#64 scope"},
		{name: "stale authority", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.StaleAuthorityRef = true }, wantMissing: "stale authority evidence"},
		{name: "missing candidate identity", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitCandidateIdentity = true }, wantMissing: "candidate identity missing"},
		{name: "missing policy bundle", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitPolicyBundle = true }, wantMissing: "policy bundle evidence missing"},
		{name: "missing source issue", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitSourceIssueRef = true }, wantMissing: "source issue ref missing or mismatched"},
		{name: "mismatched source issue", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MismatchSourceIssueRef = true }, wantMissing: "source issue ref missing or mismatched"},
		{name: "missing file boundary", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitFileBoundary = true }, wantMissing: "file boundary missing"},
		{name: "widened file boundary", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.WidenFileBoundary = true }, wantMissing: "file boundary widened"},
		{name: "missing command boundary", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitCommandBoundary = true }, wantMissing: "command/process boundary missing"},
		{name: "shell command", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ShellCommandClaim = true }, wantMissing: "shell/general command execution claim"},
		{name: "process escape", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ProcessEscapeClaim = true }, wantMissing: "process escape claim"},
		{name: "deployment command", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.DeploymentCommandClaim = true }, wantMissing: "command boundary denied operation missing: deploy"},
		{name: "github mutation command", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.GitHubMutationClaim = true }, wantMissing: "command boundary denied operation missing: gh pr merge"},
		{name: "hive action command", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.HiveActionAPIClaim = true }, wantMissing: "command boundary denied operation missing: hive.action"},
		{name: "missing network boundary", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitNetworkBoundary = true }, wantMissing: "network boundary missing"},
		{name: "unscoped network", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.UnscopedNetworkClaim = true }, wantMissing: "network scope widened"},
		{name: "widened network host", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.WidenNetworkHostScope = true }, wantMissing: "network scope widened"},
		{name: "live network", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.LiveNetworkClaim = true }, wantMissing: "live network access claim"},
		{name: "validation network", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ValidationNetworkClaim = true }, wantMissing: "validation network access claim"},
		{name: "missing secret boundary", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitSecretBoundary = true }, wantMissing: "secret boundary missing"},
		{name: "unscoped secrets", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.UnscopedSecretClaim = true }, wantMissing: "secret scope widened"},
		{name: "credential material", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.CredentialMaterialClaim = true }, wantMissing: "credential material claim"},
		{name: "secret log", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.SecretLogClaim = true }, wantMissing: "secret log claim"},
		{name: "missing redaction", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingRedactionRequired = true }, wantMissing: "secret redaction requirement missing"},
		{name: "missing timeout", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitTimeoutBoundary = true }, wantMissing: "timeout/cancellation boundary missing"},
		{name: "unbounded timeout", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.UnboundedTimeoutClaim = true }, wantMissing: "timeout unbounded"},
		{name: "missing cancellation", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingCancellation = true }, wantMissing: "cancellation evidence missing"},
		{name: "missing resource limits", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingResourceLimits = true }, wantMissing: "resource limits missing"},
		{name: "retry without receipt", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.RetryWithoutReceipt = true }, wantMissing: "retry without receipt claim"},
		{name: "missing artifacts", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitArtifactContract = true }, wantMissing: "artifact contract missing"},
		{name: "partial artifacts", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.PartialArtifactAllowance = true }, wantMissing: "partial artifact allowance"},
		{name: "missing artifact hash", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingArtifactHash = true }, wantMissing: "artifact hash requirement missing"},
		{name: "missing artifact content type", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingArtifactContent = true }, wantMissing: "artifact content-type requirement missing"},
		{name: "missing artifact size", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MissingArtifactSizeBound = true }, wantMissing: "artifact size bounds missing"},
		{name: "production artifact", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ProductionArtifactClaim = true }, wantMissing: "production artifact claim"},
		{name: "missing exit mapping", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitExitCodeMapping = true }, wantMissing: "exit-code mapping missing"},
		{name: "ambiguous exit", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.AmbiguousExitStatus = true }, wantMissing: "ambiguous exit status claim"},
		{name: "missing receipt", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitReceiptEvidence = true }, wantMissing: "execution receipt evidence missing"},
		{name: "stale receipt", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.StaleReceiptClaim = true }, wantMissing: "stale receipt claim"},
		{name: "mismatched receipt", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.MismatchedReceiptHash = true }, wantMissing: "execution receipt hash mismatch"},
		{name: "receipt before result", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ReceiptBeforeResult = true }, wantMissing: "receipt recorded before result claim"},
		{name: "missing validation", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitValidationPlan = true }, wantMissing: "validation plan missing"},
		{name: "unbounded validation", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.UnboundedValidation = true }, wantMissing: "validation plan unbounded"},
		{name: "missing replay", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.OmitReplayPlan = true }, wantMissing: "replay plan missing"},
		{name: "non deterministic replay", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.NonDeterministicReplay = true }, wantMissing: "replay plan non-deterministic"},
		{name: "replay network secrets", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ReplayRequiresNetSecret = true }, wantMissing: "replay requires network or secrets"},
		{name: "replay production", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ReplayWritesProduction = true }, wantMissing: "replay writes production state"},
		{name: "adapter enablement", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.AdapterEnablementClaim = true }, wantMissing: "external adapter enablement claim"},
		{name: "adapter invocation", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.AdapterInvocationClaim = true }, wantMissing: "external adapter invocation claim"},
		{name: "runtimebroker execution", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) {
			opts.RuntimeBrokerExecutionClaim = true
		}, wantMissing: "RuntimeBroker execution claim"},
		{name: "production eventgraph write", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ProductionEventGraphWrite = true }, wantMissing: "production EventGraph write claim"},
		{name: "production truth", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ProductionTruthClaim = true }, wantMissing: "production truth claim"},
		{name: "runtime side effect", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.RuntimeSideEffectClaim = true }, wantMissing: "forbidden action status not fail-closed: runtime side effect"},
		{name: "protected settings", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ProtectedSettingsClaim = true }, wantMissing: "forbidden action status not fail-closed: protected settings change"},
		{name: "test001", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.Test001GreenClaim = true }, wantMissing: "forbidden action status not fail-closed: Test 001 GREEN"},
		{name: "docs172", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.Docs172ClosureClaim = true }, wantMissing: "forbidden action status not fail-closed: docs#172 closure"},
		{name: "autonomy", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.AutonomyIncreaseClaim = true }, wantMissing: "forbidden action status not fail-closed: autonomy increase"},
		{name: "value", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ValueAllocationClaim = true }, wantMissing: "forbidden action status not fail-closed: value allocation"},
		{name: "residual", mutate: func(opts *work.RuntimeBrokerExternalAdapterEligibilityOptions) { opts.ResidualRiskClosureClaim = true }, wantMissing: "forbidden action status not fail-closed: residual-risk closure"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := work.RuntimeBrokerExternalAdapterEligibilityOptions{}
			tc.mutate(&opts)
			run := work.RunRuntimeBrokerExternalAdapterEligibilityFixture(opts)
			if run.Report.Status != "fail" {
				t.Fatalf("status=%q; want fail for %s", run.Report.Status, tc.name)
			}
			if !containsDiagnostic(run.Report.Missing, tc.wantMissing) {
				t.Fatalf("missing=%#v; want %q", run.Report.Missing, tc.wantMissing)
			}
			if run.Report.EventGraphHandoff.Status != "blocked" {
				t.Fatalf("eventgraph handoff=%#v; want blocked", run.Report.EventGraphHandoff)
			}
			if _, err := work.RuntimeBrokerExternalAdapterEligibilityStatus(run); err == nil {
				t.Fatalf("RuntimeBrokerExternalAdapterEligibilityStatus err=nil; want incomplete error")
			}
		})
	}
}
