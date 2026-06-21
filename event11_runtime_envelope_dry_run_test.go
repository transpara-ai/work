package work_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

const event11ExpectedRuntimeEnvelopeHash = "sha256:86f035c19b12ae6cb08cde11d0622b0db83bf1675893b7544d577efe50d38a51"

func TestEvent11RuntimeEnvelopeDryRunFixtureBuildsCompleteEvidence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, work.Event11RuntimeEnvelopeDryRunOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunEvent11RuntimeEnvelopeDryRunFixture: %v", err)
	}
	if run.Mode != work.Event11RuntimeEnvelopeDryRunMode {
		t.Fatalf("mode = %q, want %q", run.Mode, work.Event11RuntimeEnvelopeDryRunMode)
	}
	if run.Report.Status != "pass" || len(run.Report.Missing) != 0 {
		t.Fatalf("report = %#v; want pass with no missing evidence", run.Report)
	}
	if status, err := work.Event11RuntimeEnvelopeDryRunStatus(run); status != "pass" || err != nil {
		t.Fatalf("status=%q err=%v; want pass nil", status, err)
	}
	if run.RuntimeRun.Result.Result.Status != work.RuntimeStatusSucceeded {
		t.Fatalf("runtime status = %q; want succeeded", run.RuntimeRun.Result.Result.Status)
	}
	if !run.Report.EnvelopeImmutable || !strings.HasPrefix(run.Report.EnvelopeHash, "sha256:") {
		t.Fatalf("envelope immutable=%v hash=%q; want immutable sha256", run.Report.EnvelopeImmutable, run.Report.EnvelopeHash)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessPassed || !run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want completed pass", run.TraceCompleteness)
	}
	if !run.AuthorityPath.Completed {
		t.Fatalf("authority path = %#v; want complete", run.AuthorityPath)
	}
	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "complete" || run.AuditReport.TraceScore != 1 {
		t.Fatalf("audit report = %#v; want complete score 1", run.AuditReport)
	}
	if run.WorkProjection.Status != work.StatusCertified {
		t.Fatalf("work status = %q; want certified", run.WorkProjection.Status)
	}
	if run.WorkProjection.Linkage.FactoryOrderID != run.FactoryOrderID {
		t.Fatalf("work linkage factory order = %q; want %q", run.WorkProjection.Linkage.FactoryOrderID, run.FactoryOrderID)
	}
	if len(run.WorkProjection.Verification.GateResultIDs) != 1 || run.WorkProjection.Verification.GateResultIDs[0] != run.GateResultID {
		t.Fatalf("work verification = %#v; want gate result %s", run.WorkProjection.Verification, run.GateResultID)
	}

	for _, typ := range []string{
		v39.TypeFactoryOrder,
		v39.TypeRequirement,
		v39.TypeAcceptanceCriterion,
		v39.TypeTask,
		v39.TypeActorIdentity,
		v39.TypeActorInvocation,
		v39.TypeAuthorityRequest,
		v39.TypeAuthorityDecision,
		v39.TypeExecutionReceipt,
		v39.TypeRuntimeEnvelope,
		v39.TypeRuntimeResult,
		v39.TypeArtifact,
		v39.TypeCodeChange,
		v39.TypeTestCase,
		v39.TypeTestRun,
		v39.TypeGateResult,
		v39.TypeFactoryRuntimeVersion,
		v39.TypeReleaseCandidate,
		v39.TypeAuditReport,
	} {
		if run.Report.TypeCounts[typ] == 0 {
			t.Fatalf("type count %s = 0; want evidence", typ)
		}
	}

	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.ActorIdentityID), v39.EdgeRequestedAuthority, run.AuthorityRequestID)
	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.AuthorityRequestID), v39.EdgeDecidedBy, run.AuthorityDecisionID)
	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.AuthorityDecisionID), v39.EdgeReceiptedBy, run.ExecutionReceiptID)
	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.TaskID), v39.EdgeUsedEnvelope, run.RuntimeEnvelopeID)
	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.RuntimeEnvelopeID), v39.EdgeProduced, run.RuntimeResultID)
	assertEvent11Edge(t, run.EventGraph.EdgesFrom(run.ArtifactID), v39.EdgeModified, run.CodeChangeID)

	envelope := assertEvent11GraphRecord(t, run, run.RuntimeEnvelopeID, v39.TypeRuntimeEnvelope).(*v39.RuntimeEnvelope)
	if envelope.RuntimeAdapterID != "local_deterministic" || envelope.NetworkPolicy != "disabled" || envelope.SecretsPolicy != "none" {
		t.Fatalf("envelope = %#v; want local deterministic disabled/none", envelope)
	}
	if envelope.WorkingDirectory != "fixture://work/event11-runtime-envelope-dry-run" {
		t.Fatalf("working directory = %q; want canonical fixture URI", envelope.WorkingDirectory)
	}
	if envelope.EnvelopeHash != run.Report.EnvelopeHash {
		t.Fatalf("envelope hash = %q; report hash = %q; want same hash", envelope.EnvelopeHash, run.Report.EnvelopeHash)
	}
	hashEnvelope := *envelope
	hashEnvelope.EnvelopeHash = ""
	encodedEnvelope, err := json.Marshal(hashEnvelope)
	if err != nil {
		t.Fatalf("marshal envelope for hash: %v", err)
	}
	sum := sha256.Sum256(encodedEnvelope)
	recomputedHash := "sha256:" + hex.EncodeToString(sum[:])
	if recomputedHash != envelope.EnvelopeHash {
		t.Fatalf("recomputed envelope hash = %q; want recorded hash %q", recomputedHash, envelope.EnvelopeHash)
	}
	if envelope.EnvelopeHash != event11ExpectedRuntimeEnvelopeHash {
		t.Fatalf("envelope hash = %q; want deterministic hash %q", envelope.EnvelopeHash, event11ExpectedRuntimeEnvelopeHash)
	}
	for _, denied := range []string{"shell", "network_attempt", "secret_attempt", "gh pr merge", "git push origin main", "deploy", "production operation", "value allocation"} {
		if !containsString(envelope.DeniedCommands, denied) {
			t.Fatalf("denied commands = %#v; want %q", envelope.DeniedCommands, denied)
		}
	}
	for _, deniedFile := range []string{"secret.txt", ".env", ".git/", "../", "go.mod", "go.sum", "production/**"} {
		if !containsString(envelope.DeniedFiles, deniedFile) {
			t.Fatalf("denied files = %#v; want %q", envelope.DeniedFiles, deniedFile)
		}
	}
	result := assertEvent11GraphRecord(t, run, run.RuntimeResultID, v39.TypeRuntimeResult).(*v39.RuntimeResult)
	if len(result.NetworkAccessLog) != 0 || len(result.SecretAccessLog) != 0 {
		t.Fatalf("runtime result network=%#v secrets=%#v; want empty", result.NetworkAccessLog, result.SecretAccessLog)
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != "report.txt" {
		t.Fatalf("runtime changed files = %#v; want report.txt only", result.ChangedFiles)
	}
	receipt := assertEvent11GraphRecord(t, run, run.ExecutionReceiptID, v39.TypeExecutionReceipt).(*v39.ExecutionReceipt)
	if receipt.Action != "runtime.invoke.local.event11_dry_run_fixture" || receipt.Result != "succeeded" {
		t.Fatalf("receipt = %#v; want local fixture success only", receipt)
	}
	for _, forbidden := range run.Report.ForbiddenActions {
		if forbidden.Action == "Gate U closure" {
			if forbidden.Status != "not_claimed" {
				t.Fatalf("forbidden action = %#v; want Gate U not_claimed", forbidden)
			}
			continue
		}
		if forbidden.Status != "not_run" {
			t.Fatalf("forbidden action = %#v; want not_run", forbidden)
		}
	}
	for _, residual := range run.Report.ResidualRisks {
		if residual.Status != "unresolved_excluded" {
			t.Fatalf("residual = %#v; want unresolved_excluded", residual)
		}
	}
}

func TestEvent11RuntimeEnvelopeDryRunPolicyCases(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, work.Event11RuntimeEnvelopeDryRunOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunEvent11RuntimeEnvelopeDryRunFixture: %v", err)
	}
	cases := map[string]work.Event11PolicyCaseResult{}
	for _, policyCase := range run.PolicyCases {
		cases[policyCase.Name] = policyCase
		if !policyCase.SideEffectFree {
			t.Fatalf("policy case %#v did not prove side-effect boundary", policyCase)
		}
	}
	for _, name := range []string{"denied_command", "path_traversal", "network_attempt", "secret_attempt", "timeout", "validation_failure"} {
		if _, ok := cases[name]; !ok {
			t.Fatalf("policy case %q missing from %#v", name, cases)
		}
	}
	for _, name := range []string{"denied_command", "path_traversal", "network_attempt", "secret_attempt"} {
		if cases[name].Status != work.RuntimeStatusPolicyBlocked || !cases[name].PolicyBlocked {
			t.Fatalf("case %s = %#v; want policy_blocked", name, cases[name])
		}
	}
	if cases["timeout"].Status != work.RuntimeStatusTimedOut || !cases["timeout"].TimedOut {
		t.Fatalf("timeout case = %#v; want timed_out", cases["timeout"])
	}
	if cases["validation_failure"].Status != work.RuntimeStatusValidationFailed || !cases["validation_failure"].ValidationError {
		t.Fatalf("validation case = %#v; want validation_failed", cases["validation_failure"])
	}
}

func TestEvent11RuntimeEnvelopeDryRunFixtureFailsClosedForMissingEvidence(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*work.Event11RuntimeEnvelopeDryRunOptions)
		wantMissing string
	}{
		{name: "missing receipt", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitAuthorityReceipt = true }, wantMissing: "ExecutionReceipt unavailable"},
		{name: "missing runtime result", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitRuntimeResult = true }, wantMissing: "RuntimeResult unavailable"},
		{name: "missing code change", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitCodeChange = true }, wantMissing: "CodeChange unavailable"},
		{name: "missing audit report", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitAuditReport = true }, wantMissing: "AuditReport unavailable"},
		{name: "missing policy cases", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitPolicyCases = true }, wantMissing: "policy cases unavailable"},
		{name: "missing envelope hash", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.OmitEnvelopeHash = true }, wantMissing: "RuntimeEnvelope immutable hash unavailable"},
		{name: "widened network policy", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.UnsafeNetworkPolicy = "allowed" }, wantMissing: "RuntimeEnvelope network_policy widened"},
		{name: "widened secrets policy", mutate: func(opts *work.Event11RuntimeEnvelopeDryRunOptions) { opts.UnsafeSecretsPolicy = "scoped" }, wantMissing: "RuntimeEnvelope secrets_policy widened"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			opts := work.Event11RuntimeEnvelopeDryRunOptions{
				Source:         testActor,
				ConversationID: testConv,
				Causes:         causes,
				WorkingDir:     t.TempDir(),
			}
			tt.mutate(&opts)
			run, err := work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, opts)
			if err != nil {
				t.Fatalf("RunEvent11RuntimeEnvelopeDryRunFixture: %v", err)
			}
			if run.Report.Status != "fail" {
				t.Fatalf("report status = %q; want fail", run.Report.Status)
			}
			if !containsString(run.Report.Missing, tt.wantMissing) {
				t.Fatalf("missing = %#v; want %q", run.Report.Missing, tt.wantMissing)
			}
			if _, err := work.Event11RuntimeEnvelopeDryRunStatus(run); err == nil {
				t.Fatalf("Event11RuntimeEnvelopeDryRunStatus err = nil; want incomplete error")
			}
			if run.Certification != nil {
				t.Fatalf("certification = %#v; want nil", run.Certification)
			}
			if run.Rejection == nil {
				t.Fatal("rejection = nil; want fail-closed rejection")
			}
			if opts.OmitAuditReport {
				if run.AuditReport != nil {
					t.Fatalf("audit report = %#v; want omitted", run.AuditReport)
				}
			} else if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "incomplete" {
				t.Fatalf("audit report = %#v; want incomplete", run.AuditReport)
			}
			if run.WorkProjection.Status != work.StatusRejected {
				t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
			}
		})
	}
}

func TestEvent11RuntimeEnvelopeDryRunFixtureRejectsUnsafeOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, work.Event11RuntimeEnvelopeDryRunOptions{
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "source actor") {
		t.Fatalf("missing source err = %v; want source actor error", err)
	}

	_, err = work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, work.Event11RuntimeEnvelopeDryRunOptions{
		Source:     testActor,
		Causes:     causes,
		WorkingDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "conversation ID") {
		t.Fatalf("missing conversation err = %v; want conversation ID error", err)
	}

	_, err = work.RunEvent11RuntimeEnvelopeDryRunFixture(ts, work.Event11RuntimeEnvelopeDryRunOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
	})
	if err == nil || !strings.Contains(err.Error(), "working directory") {
		t.Fatalf("missing working dir err = %v; want working directory error", err)
	}
}

func assertEvent11GraphRecord(t *testing.T, run work.Event11RuntimeEnvelopeDryRunRun, id, typ string) v39.Record {
	t.Helper()
	record, err := run.EventGraph.Get(id)
	if err != nil {
		t.Fatalf("graph.Get(%q): %v", id, err)
	}
	if got := record.GetCommon().Type; got != typ {
		t.Fatalf("record %q type = %q, want %q", id, got, typ)
	}
	return record
}

func assertEvent11Edge(t *testing.T, edges []v39.CommonEdge, typ, toID string) {
	t.Helper()
	for _, edge := range edges {
		if edge.Type == typ && edge.ToID == toID {
			return
		}
	}
	t.Fatalf("edges = %#v; want %s -> %s", edges, typ, toID)
}
