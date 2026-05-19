package work_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
	"github.com/transpara-ai/work"
)

func TestEpic2ThinFactoryVerticalSliceCertified(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic2ThinFactoryCertified,
	})
	if err != nil {
		t.Fatalf("RunEpic2ThinFactoryVerticalSlice: %v", err)
	}

	if run.Certification == nil || run.Rejection != nil {
		t.Fatalf("decision certification=%#v rejection=%#v; want certification only", run.Certification, run.Rejection)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessPassed || !run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want completed pass", run.TraceCompleteness)
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
	if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "complete" || run.AuditReport.TraceScore != 1 {
		t.Fatalf("audit report = %#v; want complete score 1", run.AuditReport)
	}
	if run.Projection.ProofOfWorkPacket == nil || run.Projection.ProofOfWorkPacket.Status != "pass" {
		t.Fatalf("proof packet = %#v; want pass", run.Projection.ProofOfWorkPacket)
	}
	if len(run.Projection.MissingProvenance) != 0 {
		t.Fatalf("missing provenance = %#v; want none", run.Projection.MissingProvenance)
	}
	assertEpic2NoCapabilityEvidence(t, run)
	assertEpic2NoProtectedSideEffects(t, run)
	assertEpic2ProjectionJSONShape(t, run, "certification")
}

func TestEpic2ThinFactoryVerticalSliceRejected(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	run, err := work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic2ThinFactoryRejected,
	})
	if err != nil {
		t.Fatalf("RunEpic2ThinFactoryVerticalSlice: %v", err)
	}

	if run.Rejection == nil || run.Certification != nil {
		t.Fatalf("decision rejection=%#v certification=%#v; want rejection only", run.Rejection, run.Certification)
	}
	if run.TraceCompleteness.Status != v39.TraceCompletenessFailed || run.TraceCompleteness.Completed {
		t.Fatalf("trace = %#v; want incomplete failure", run.TraceCompleteness)
	}
	if !containsString(run.TraceCompleteness.Missing, "REPAIRED_BY from Failure "+run.FailureID) {
		t.Fatalf("trace missing = %#v; want repair evidence gap", run.TraceCompleteness.Missing)
	}
	if run.WorkProjection.Status != work.StatusRejected {
		t.Fatalf("work status = %q; want rejected", run.WorkProjection.Status)
	}
	if len(run.WorkProjection.FailureRepair.FailureIDs) != 1 || run.WorkProjection.FailureRepair.FailureIDs[0] != run.FailureID {
		t.Fatalf("work failure refs = %#v; want %s", run.WorkProjection.FailureRepair, run.FailureID)
	}
	if run.AuditReport == nil || statusValue(run.AuditReport.CommonNode.Status) != "incomplete" {
		t.Fatalf("audit report = %#v; want incomplete", run.AuditReport)
	}
	if len(run.Projection.MissingProvenance) == 0 {
		t.Fatalf("missing provenance = %#v; want repair gap", run.Projection.MissingProvenance)
	}
	if run.Projection.ProofOfWorkPacket == nil || len(run.Projection.ProofOfWorkPacket.KnownFailures) != 1 {
		t.Fatalf("proof packet known failures = %#v; want one failure", run.Projection.ProofOfWorkPacket)
	}
	assertEpic2NoCapabilityEvidence(t, run)
	assertEpic2NoProtectedSideEffects(t, run)
	assertEpic2ProjectionJSONShape(t, run, "rejection")

	_, err = run.EventGraph.CertifyReleaseCandidate(&v39.Certification{
		CommonNode: v39.CommonNode{
			ID:             "cert_epic2_rejected_attempt",
			Type:           v39.TypeCertification,
			CreatedAt:      time.Date(2026, 5, 19, 12, 1, 0, 0, time.UTC),
			CreatedBy:      "act_human",
			Status:         strPtr("certified"),
			IdempotencyKey: "idem_cert_epic2_rejected_attempt",
			CorrelationID:  "corr_epic2_rejected_attempt",
		},
		ReleaseCandidateID: run.ReleaseCandidateID,
		CertifierActorID:   "act_human",
		Reason:             "should not certify negative fixture",
		EvidenceRefs:       []string{run.GateResultID},
	})
	if !errors.Is(err, v39.ErrRequiredPathMissing) {
		t.Fatalf("CertifyReleaseCandidate err = %v; want required path missing", err)
	}
}

func TestEpic2ThinFactoryRejectsUnsafeFixtureOptions(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	_, err := work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic2ThinFactoryCertified,
	})
	if err == nil || !strings.Contains(err.Error(), "source actor is required") {
		t.Fatalf("missing source err = %v; want source requirement", err)
	}

	_, err = work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		Source:     testActor,
		Causes:     causes,
		WorkingDir: t.TempDir(),
		Mode:       work.Epic2ThinFactoryCertified,
	})
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("missing conversation err = %v; want conversation requirement", err)
	}

	_, err = work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		WorkingDir:     t.TempDir(),
		Mode:           work.Epic2ThinFactoryMode("capability"),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported Epic 2 fixture mode") {
		t.Fatalf("unsupported mode err = %v; want mode rejection", err)
	}

	_, err = work.RunEpic2ThinFactoryVerticalSlice(ts, work.Epic2ThinFactoryOptions{
		Source:         testActor,
		ConversationID: testConv,
		Causes:         causes,
		Mode:           work.Epic2ThinFactoryCertified,
	})
	if err == nil || !strings.Contains(err.Error(), "working directory is required") {
		t.Fatalf("missing working dir err = %v; want local workdir requirement", err)
	}
}

func assertEpic2NoCapabilityEvidence(t *testing.T, run work.Epic2ThinFactoryRun) {
	t.Helper()
	for _, typ := range []string{v39.TypeCapabilityArtifact, v39.TypeCapabilityVersion, v39.TypeActivationPolicy, v39.TypeRollbackRecord} {
		if records := run.EventGraph.ByType(typ); len(records) != 0 {
			t.Fatalf("%s records = %#v; want none", typ, records)
		}
	}
	taskRecord, err := run.EventGraph.Get(run.TaskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	task := taskRecord.(*v39.Task)
	if len(task.CommonNode.SourceRefs) != 0 {
		t.Fatalf("task source refs = %#v; want none", task.CommonNode.SourceRefs)
	}
	frvRecord, err := run.EventGraph.Get("frv_epic2_" + string(run.Mode))
	if err != nil {
		t.Fatalf("get factory runtime version: %v", err)
	}
	frv := frvRecord.(*v39.FactoryRuntimeVersion)
	if len(frv.CapabilityVersionRefs) != 0 {
		t.Fatalf("capability version refs = %#v; want none", frv.CapabilityVersionRefs)
	}
	for _, id := range []string{run.TaskID, run.ReleaseCandidateID, run.FactoryOrderID, run.GateResultID} {
		for _, edge := range run.EventGraph.EdgesFrom(id) {
			if edge.Type == v39.EdgeUsedCapability {
				t.Fatalf("unexpected USED_CAPABILITY edge from %s: %#v", id, edge)
			}
		}
	}
	path, err := run.EventGraph.CapabilityUsageEvidencePath(run.ReleaseCandidateID)
	if err != nil {
		t.Fatalf("CapabilityUsageEvidencePath: %v", err)
	}
	if !path.Completed || len(path.Missing) != 0 {
		t.Fatalf("capability usage path = %#v; want non-applicable completed path", path)
	}
}

func assertEpic2NoProtectedSideEffects(t *testing.T, run work.Epic2ThinFactoryRun) {
	t.Helper()
	if run.RuntimeRun.Envelope.Envelope.Worker != "local_deterministic" {
		t.Fatalf("worker = %q; want local_deterministic", run.RuntimeRun.Envelope.Envelope.Worker)
	}
	if run.RuntimeRun.Envelope.Envelope.NetworkPolicy != "disabled" {
		t.Fatalf("network policy = %q; want disabled", run.RuntimeRun.Envelope.Envelope.NetworkPolicy)
	}
	if run.RuntimeRun.Envelope.Envelope.SecretsPolicy != "none" {
		t.Fatalf("secrets policy = %q; want none", run.RuntimeRun.Envelope.Envelope.SecretsPolicy)
	}
	for _, log := range run.RuntimeRun.Result.Result.CommandLog {
		if log.Name == "network_attempt" || log.Name == "secret_attempt" {
			t.Fatalf("command log includes protected attempt: %#v", log)
		}
	}
	record, err := run.EventGraph.Get("env_epic2_" + string(run.Mode))
	if err != nil {
		t.Fatalf("get runtime envelope: %v", err)
	}
	envelope := record.(*v39.RuntimeEnvelope)
	if envelope.NetworkPolicy != "disabled" || envelope.SecretsPolicy != "none" {
		t.Fatalf("eventgraph runtime policies = %q/%q; want disabled/none", envelope.NetworkPolicy, envelope.SecretsPolicy)
	}
	if !containsString(envelope.DeniedCommands, "network_attempt") || !containsString(envelope.DeniedCommands, "secret_attempt") {
		t.Fatalf("denied commands = %#v; want network and secret attempts denied", envelope.DeniedCommands)
	}
}

func assertEpic2ProjectionJSONShape(t *testing.T, run work.Epic2ThinFactoryRun, decisionKind string) {
	t.Helper()
	payload, err := run.Projection.JSON()
	if err != nil {
		t.Fatalf("projection JSON: %v", err)
	}
	var decoded struct {
		Source       string `json:"source"`
		FactoryOrder struct {
			ID string `json:"id"`
		} `json:"factory_order"`
		ReleaseCandidate struct {
			ID string `json:"id"`
		} `json:"release_candidate"`
		Decision struct {
			Kind string `json:"kind"`
		} `json:"decision"`
		ProofOfWorkPacket struct {
			ID       string `json:"id"`
			WorkItem struct {
				EventGraphRefs []string `json:"event_graph_refs"`
			} `json:"work_item"`
			SecurityScanResults []struct {
				Status  string `json:"status"`
				Summary string `json:"summary"`
			} `json:"security_scan_results"`
		} `json:"proof_of_work_packet"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal projection JSON: %v", err)
	}
	if decoded.Source != "work-epic2-thin-factory-fixture" {
		t.Fatalf("source = %q; want work fixture", decoded.Source)
	}
	if decoded.FactoryOrder.ID != run.FactoryOrderID || decoded.ReleaseCandidate.ID != run.ReleaseCandidateID {
		t.Fatalf("projection ids = %s/%s; want %s/%s", decoded.FactoryOrder.ID, decoded.ReleaseCandidate.ID, run.FactoryOrderID, run.ReleaseCandidateID)
	}
	if decoded.Decision.Kind != decisionKind {
		t.Fatalf("decision kind = %q; want %s", decoded.Decision.Kind, decisionKind)
	}
	if decoded.ProofOfWorkPacket.ID == "" || len(decoded.ProofOfWorkPacket.WorkItem.EventGraphRefs) != 1 {
		t.Fatalf("proof packet = %#v; want site-compatible work item refs", decoded.ProofOfWorkPacket)
	}
	if decisionKind == "rejection" && len(run.Projection.Timeline) < 4 {
		t.Fatalf("timeline = %#v; want rejected fixture failure entry", run.Projection.Timeline)
	}
	foundGateB := false
	for _, item := range decoded.ProofOfWorkPacket.SecurityScanResults {
		if strings.Contains(item.Summary, "CapabilityArtifact") && item.Status == "not_applicable" {
			foundGateB = true
		}
	}
	if !foundGateB {
		t.Fatalf("security scan results = %#v; want Gate B non-applicable evidence", decoded.ProofOfWorkPacket.SecurityScanResults)
	}
}

func statusValue(status *string) string {
	if status == nil {
		return ""
	}
	return *status
}

func strPtr(value string) *string {
	return &value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
