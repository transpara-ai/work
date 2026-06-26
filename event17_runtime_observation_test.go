package work_test

import (
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestEvent17GovernedRuntimeObservationFixtureBuildsCompleteEvidence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEvent17GovernedRuntimeObservationFixture(ts, work.Event17GovernedRuntimeObservationOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunEvent17GovernedRuntimeObservationFixture: %v", err)
	}
	if run.Mode != work.Event17GovernedRuntimeObservationMode {
		t.Fatalf("mode = %q, want %q", run.Mode, work.Event17GovernedRuntimeObservationMode)
	}
	if status, err := work.Event17GovernedRuntimeObservationStatus(run); status != "pass" || err != nil {
		t.Fatalf("status=%q err=%v; want pass nil", status, err)
	}
	report := run.Report
	if report.Status != "pass" || len(report.Missing) != 0 {
		t.Fatalf("report = %#v; want pass with no missing", report)
	}
	if report.Envelope.Status != "recorded" || !report.Envelope.Immutable || !strings.HasPrefix(report.Envelope.EnvelopeHash, "sha256:") {
		t.Fatalf("envelope = %#v; want recorded immutable sha256", report.Envelope)
	}
	if report.Envelope.NetworkPolicy != "disabled" || report.Envelope.SecretsPolicy != "none" {
		t.Fatalf("envelope policy = %#v; want disabled/none", report.Envelope)
	}
	if report.Result.Status != "recorded" || report.Result.SideEffectClaimed {
		t.Fatalf("result = %#v; want recorded without side-effect claim", report.Result)
	}
	if len(report.Result.NetworkAccessLog) != 0 || len(report.Result.SecretAccessLog) != 0 {
		t.Fatalf("result logs network=%#v secrets=%#v; want empty", report.Result.NetworkAccessLog, report.Result.SecretAccessLog)
	}
	if report.Policy.Status != "recorded" || !report.Policy.NetworkDisabled || !report.Policy.SecretsDenied ||
		report.Policy.ExternalAdapterClaimed || report.Policy.ShellCommandClaimed {
		t.Fatalf("policy = %#v; want local disabled/denied policy", report.Policy)
	}
	if report.Trace.Status != "recorded" || !report.Trace.TraceCompleted ||
		report.Trace.TestRunID == "" || report.Trace.GateResultID == "" || report.Trace.AuditReportID == "" {
		t.Fatalf("trace = %#v; want complete TestRun/GateResult/AuditReport trace", report.Trace)
	}
	if report.TestRun.ID == "" || report.GateResult.ID == "" || report.AuditReport.ID == "" {
		t.Fatalf("evidence observations missing: test=%#v gate=%#v audit=%#v", report.TestRun, report.GateResult, report.AuditReport)
	}
	if !report.EventGraphHandoff.DescriptorOnly ||
		report.EventGraphHandoff.PersistentWriteStatus != "not_written" ||
		report.EventGraphHandoff.PersistentWriteClaimed ||
		report.EventGraphHandoff.ProductionTruthClaimed {
		t.Fatalf("handoff = %#v; want descriptor-only not-written", report.EventGraphHandoff)
	}
	presence := report.CivilizationPresence
	if presence.Status != "monitoring_only" || !presence.MonitoringOnly ||
		presence.CivilizationRuntimeReady || presence.HiveActive ||
		presence.HiveWakeStartClaimed || presence.IssueClosureAuthorityClaimed ||
		presence.ProductionTruthClaimed || presence.AutonomyIncreaseClaimed {
		t.Fatalf("civilization presence = %#v; want monitoring-only non-claim", presence)
	}
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" && action.Status != "not_claimed" {
			t.Fatalf("forbidden action = %#v; want fail-closed status", action)
		}
	}
	for _, residual := range report.ResidualRisks {
		if residual.Status != "unresolved_excluded" {
			t.Fatalf("residual = %#v; want unresolved_excluded", residual)
		}
	}
}

func TestEvent17GovernedRuntimeObservationFixtureFailsClosed(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*work.Event17GovernedRuntimeObservationOptions)
		wantMissing string
	}{
		{name: "missing authority", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitAuthority = true }, wantMissing: "authority evidence missing"},
		{name: "widened authority", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) {
			opts.WidenAuthorityClaim = "Hive wake/start"
		}, wantMissing: "authority claim outside Event 17 scope"},
		{name: "missing envelope", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitEnvelope = true }, wantMissing: "pre-run RuntimeEnvelope observation missing"},
		{name: "missing runtime result", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitRuntimeResult = true }, wantMissing: "RuntimeResult observation missing"},
		{name: "missing policy decision", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitPolicyDecision = true }, wantMissing: "policy decision observation missing"},
		{name: "missing trace", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitTrace = true }, wantMissing: "trace evidence missing"},
		{name: "widened trace", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.WidenTraceScope = true }, wantMissing: "trace scope widened"},
		{name: "missing test run", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitTestRun = true }, wantMissing: "TestRun observation missing"},
		{name: "missing gate result", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitGateResult = true }, wantMissing: "GateResult observation missing"},
		{name: "missing audit report", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitAuditReport = true }, wantMissing: "AuditReport observation missing"},
		{name: "mismatched envelope hash", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.MismatchEnvelopeHash = true }, wantMissing: "envelope hash mismatch"},
		{name: "widened network", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.UnsafeNetworkPolicy = "allowed" }, wantMissing: "network policy widened"},
		{name: "widened secrets", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.UnsafeSecretsPolicy = "explicit" }, wantMissing: "secrets policy widened"},
		{name: "external adapter claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.ExternalAdapterClaim = true }, wantMissing: "external adapter claim"},
		{name: "shell command claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.ShellCommandClaim = true }, wantMissing: "shell/general command execution claim"},
		{name: "production eventgraph write claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.ProductionEventGraphWriteClaim = true }, wantMissing: "production EventGraph write claim"},
		{name: "production truth claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.ProductionTruthClaim = true }, wantMissing: "production truth claim"},
		{name: "runtime side-effect claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.RuntimeSideEffectClaim = true }, wantMissing: "runtime side-effect claim"},
		{name: "missing civilization presence", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.OmitCivilizationPresence = true }, wantMissing: "civilization-presence boundary metadata missing"},
		{name: "malformed civilization presence", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.MalformedCivilizationPresence = true }, wantMissing: "civilization-presence boundary metadata malformed"},
		{name: "civilization runtime ready claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.CivilizationRuntimeReadyClaim = true }, wantMissing: "civilization runtime readiness claim"},
		{name: "hive activity claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.HiveActivityClaim = true }, wantMissing: "Hive activity or wake/start claim"},
		{name: "issue closure authority claim", mutate: func(opts *work.Event17GovernedRuntimeObservationOptions) { opts.IssueClosureAuthorityClaim = true }, wantMissing: "issue-closure authority claim"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			opts := work.Event17GovernedRuntimeObservationOptions{
				Source:         testActor,
				ConversationID: testConv,
				Causes:         causes,
				WorkingDir:     t.TempDir(),
			}
			tc.mutate(&opts)
			run, err := work.RunEvent17GovernedRuntimeObservationFixture(ts, opts)
			if err != nil {
				t.Fatalf("RunEvent17GovernedRuntimeObservationFixture: %v", err)
			}
			if run.Report.Status != "fail" {
				t.Fatalf("status = %q; want fail for %s", run.Report.Status, tc.name)
			}
			if !containsDiagnostic(run.Report.Missing, tc.wantMissing) {
				t.Fatalf("missing = %#v; want %q", run.Report.Missing, tc.wantMissing)
			}
			if _, err := work.Event17GovernedRuntimeObservationStatus(run); err == nil {
				t.Fatalf("Event17GovernedRuntimeObservationStatus err = nil; want incomplete error")
			}
			if run.Report.EventGraphHandoff.Status != "blocked" {
				t.Fatalf("handoff = %#v; want blocked", run.Report.EventGraphHandoff)
			}
		})
	}
}
