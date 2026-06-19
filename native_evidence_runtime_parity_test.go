package work_test

import (
	"strings"
	"testing"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestBuildNativeEvidenceRuntimeParityFixtureBuildsCompleteLocalEvidence(t *testing.T) {
	run, err := work.BuildNativeEvidenceRuntimeParityFixture(work.NativeEvidenceRuntimeParityOptions{})
	if err != nil {
		t.Fatalf("BuildNativeEvidenceRuntimeParityFixture: %v", err)
	}
	if run.Mode != work.NativeEvidenceRuntimeParityMode {
		t.Fatalf("mode = %q, want %q", run.Mode, work.NativeEvidenceRuntimeParityMode)
	}
	if run.ParityReport.Status != "pass" || len(run.ParityReport.Missing) != 0 {
		t.Fatalf("parity report = %#v; want pass with no missing evidence", run.ParityReport)
	}
	if len(run.ParityReport.EvaluationErrors) != 0 {
		t.Fatalf("evaluation errors = %#v; want none", run.ParityReport.EvaluationErrors)
	}
	if status, err := work.NativeEvidenceRuntimeParityStatus(run); status != "pass" || err != nil {
		t.Fatalf("status=%q err=%v; want pass nil", status, err)
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
		if got := len(run.EventGraph.ByType(typ)); got == 0 {
			t.Fatalf("record type %s count = 0; want native evidence", typ)
		}
		if run.ParityReport.TypeCounts[typ] == 0 {
			t.Fatalf("report type count %s = 0; want native evidence", typ)
		}
	}

	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.ActorIdentityID), v39.EdgeRequestedAuthority, run.AuthorityRequestID)
	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.AuthorityRequestID), v39.EdgeDecidedBy, run.AuthorityDecisionID)
	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.AuthorityDecisionID), v39.EdgeReceiptedBy, run.ExecutionReceiptID)
	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.TaskID), v39.EdgeUsedEnvelope, run.RuntimeEnvelopeID)
	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.RuntimeEnvelopeID), v39.EdgeProduced, run.RuntimeResultID)
	assertNativeParityEdge(t, run.EventGraph.EdgesFrom(run.ArtifactID), v39.EdgeModified, run.CodeChangeID)

	envelope := assertNativeParityGraphRecord(t, run, run.RuntimeEnvelopeID, v39.TypeRuntimeEnvelope).(*v39.RuntimeEnvelope)
	if envelope.NetworkPolicy != "disabled" || envelope.SecretsPolicy != "none" {
		t.Fatalf("envelope policies = %s/%s; want disabled/none", envelope.NetworkPolicy, envelope.SecretsPolicy)
	}
	if !strings.HasPrefix(envelope.WorkingDirectory, "fixture://") {
		t.Fatalf("working directory = %q; want fixture URI", envelope.WorkingDirectory)
	}
	for _, denied := range []string{"RuntimeBroker", "gh pr merge", "deploy", "secret access", "production operation", "value allocation"} {
		if !containsString(envelope.DeniedCommands, denied) {
			t.Fatalf("denied commands = %#v; want %q", envelope.DeniedCommands, denied)
		}
	}
	result := assertNativeParityGraphRecord(t, run, run.RuntimeResultID, v39.TypeRuntimeResult).(*v39.RuntimeResult)
	if len(result.NetworkAccessLog) != 0 || len(result.SecretAccessLog) != 0 {
		t.Fatalf("runtime result network=%#v secrets=%#v; want empty", result.NetworkAccessLog, result.SecretAccessLog)
	}
	for _, path := range result.ChangedFiles {
		if !containsString([]string{"native_evidence_runtime_parity.go", "native_evidence_runtime_parity_test.go", "docs/designs/native-evidence-runtime-parity.md"}, path) {
			t.Fatalf("changed file path = %q; want authorized path", path)
		}
	}
	receipt := assertNativeParityGraphRecord(t, run, run.ExecutionReceiptID, v39.TypeExecutionReceipt).(*v39.ExecutionReceipt)
	if receipt.Action != "runtime.invoke.local.native_evidence_parity_fixture" || receipt.Result != "succeeded" {
		t.Fatalf("receipt = %#v; want local fixture success only", receipt)
	}
}

func TestBuildNativeEvidenceRuntimeParityFixtureFailsClosedForMissingEvidence(t *testing.T) {
	tests := []struct {
		name        string
		opts        work.NativeEvidenceRuntimeParityOptions
		wantMissing string
	}{
		{name: "missing execution receipt", opts: work.NativeEvidenceRuntimeParityOptions{OmitExecutionReceipt: true}, wantMissing: "ExecutionReceipt missing"},
		{name: "missing runtime result", opts: work.NativeEvidenceRuntimeParityOptions{OmitRuntimeResult: true}, wantMissing: "RuntimeResult missing"},
		{name: "missing code change", opts: work.NativeEvidenceRuntimeParityOptions{OmitCodeChange: true}, wantMissing: "CodeChange missing"},
		{name: "missing audit report", opts: work.NativeEvidenceRuntimeParityOptions{OmitAuditReport: true}, wantMissing: "AuditReport missing"},
		{name: "widened network policy", opts: work.NativeEvidenceRuntimeParityOptions{UnsafeNetworkPolicy: "allowed"}, wantMissing: "RuntimeEnvelope network_policy is not disabled"},
		{name: "widened forbidden action status", opts: work.NativeEvidenceRuntimeParityOptions{UnsafeForbiddenActionStatus: "claimed"}, wantMissing: "forbidden action status not fail-closed: RuntimeBroker execution"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run, err := work.BuildNativeEvidenceRuntimeParityFixture(tt.opts)
			if err != nil {
				t.Fatalf("BuildNativeEvidenceRuntimeParityFixture: %v", err)
			}
			if run.ParityReport.Status != "fail" {
				t.Fatalf("parity status = %q; want fail", run.ParityReport.Status)
			}
			if run.TraceCompleteness.Completed {
				t.Fatalf("trace = %#v; want incomplete for fail-closed fixture", run.TraceCompleteness)
			}
			if !containsString(run.ParityReport.Missing, tt.wantMissing) {
				t.Fatalf("missing = %#v; want %q", run.ParityReport.Missing, tt.wantMissing)
			}
			if _, err := work.NativeEvidenceRuntimeParityStatus(run); err == nil {
				t.Fatalf("NativeEvidenceRuntimeParityStatus err = nil; want incomplete error")
			}
			if run.Certification != nil {
				t.Fatalf("certification = %#v; want no certification", run.Certification)
			}
			if run.Rejection == nil {
				t.Fatalf("rejection = nil; want fail-closed rejection")
			}
			if tt.opts.OmitAuditReport {
				if run.AuditReport != nil {
					t.Fatalf("audit report = %#v; want omitted", run.AuditReport)
				}
			} else if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "incomplete" {
				t.Fatalf("audit report = %#v; want incomplete", run.AuditReport)
			}
		})
	}
}

func TestNativeEvidenceRuntimeParityFixtureSurfacesEvaluationErrors(t *testing.T) {
	receiptRun, err := work.BuildNativeEvidenceRuntimeParityFixture(work.NativeEvidenceRuntimeParityOptions{OmitExecutionReceipt: true})
	if err != nil {
		t.Fatalf("BuildNativeEvidenceRuntimeParityFixture receipt: %v", err)
	}
	if !containsDiagnostic(receiptRun.ParityReport.EvaluationErrors, "AuthorityRequestDecisionReceipt") {
		t.Fatalf("receipt evaluation errors = %#v; want authority-path diagnostic", receiptRun.ParityReport.EvaluationErrors)
	}

	runtimeRun, err := work.BuildNativeEvidenceRuntimeParityFixture(work.NativeEvidenceRuntimeParityOptions{OmitRuntimeResult: true})
	if err != nil {
		t.Fatalf("BuildNativeEvidenceRuntimeParityFixture runtime: %v", err)
	}
	if !containsDiagnostic(runtimeRun.ParityReport.EvaluationErrors, "TraceCompletenessGate") {
		t.Fatalf("runtime evaluation errors = %#v; want trace diagnostic", runtimeRun.ParityReport.EvaluationErrors)
	}
}

func TestNativeEvidenceRuntimeParityStatusRejectsNilGraph(t *testing.T) {
	if status, err := work.NativeEvidenceRuntimeParityStatus(work.NativeEvidenceRuntimeParityRun{}); status != "fail" || err == nil {
		t.Fatalf("status=%q err=%v; want fail with error", status, err)
	}
}

func TestNativeEvidenceRuntimeParityFixturePreservesProposalOnlyBoundary(t *testing.T) {
	run, err := work.BuildNativeEvidenceRuntimeParityFixture(work.NativeEvidenceRuntimeParityOptions{})
	if err != nil {
		t.Fatalf("BuildNativeEvidenceRuntimeParityFixture: %v", err)
	}
	boundary := run.ParityReport.ProposalBoundary
	if boundary.PredecessorBuilder != "BuildFactoryOrderDevelopmentProposal" || !boundary.NativeParityFixture || !boundary.DocsHeldProposalOnly {
		t.Fatalf("proposal boundary = %#v; want Event 7 predecessor and native fixture", boundary)
	}
	if boundary.ProductionTruthClaimed || boundary.PersistentWriteClaimed || boundary.RuntimeExecutionClaimed {
		t.Fatalf("proposal boundary = %#v; want no production/persistent/runtime claims", boundary)
	}
	for _, residual := range run.ParityReport.ResidualRisks {
		if residual.Status != "unresolved_excluded" {
			t.Fatalf("residual = %#v; want unresolved excluded", residual)
		}
	}
	for _, forbidden := range run.ParityReport.ForbiddenActions {
		if forbidden.Status != "not_run" {
			t.Fatalf("forbidden action = %#v; want not_run", forbidden)
		}
	}
}

func assertNativeParityGraphRecord(t *testing.T, run work.NativeEvidenceRuntimeParityRun, id, typ string) v39.Record {
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

func assertNativeParityEdge(t *testing.T, edges []v39.CommonEdge, typ, toID string) {
	t.Helper()
	for _, edge := range edges {
		if edge.Type == typ && edge.ToID == toID {
			return
		}
	}
	t.Fatalf("edges = %#v; want %s -> %s", edges, typ, toID)
}

func containsDiagnostic(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
