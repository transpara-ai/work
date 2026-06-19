package work

import (
	"errors"
	"fmt"
	"strings"
	"time"

	v39 "github.com/transpara-ai/eventgraph/go/pkg/darkfactory/v39"
)

const (
	NativeEvidenceRuntimeParityMode = "deterministic_local_in_memory"

	nativeParityActorID          = "act_df_v40_event8_native_parity_fixture"
	nativeParityExternalActorID  = "act_michael_saucier_external_committee"
	nativeParityAuthorityDoc     = "DF-V4.0-EPIC-008-AUTHORITY-DECISION"
	nativeParityDocsPR           = "transpara-ai/docs#162"
	nativeParityDocsMergeSHA     = "6e18eb7df0a879ad02b1dc6bb53628d918f06377"
	nativeParityDocsReviewedHead = "d452af416f65bad2bc40f4b5be7f905963a491ba"
	nativeParityCorrelationID    = "corr_df_v40_event8_native_evidence_runtime_parity"
	nativeParityFixtureTimeRFC   = "2026-06-19T10:30:00Z"
)

// NativeEvidenceRuntimeParityOptions provides negative-test seams for the local
// deterministic parity fixture. Omitted evidence is represented as unavailable
// native evidence and must fail closed.
type NativeEvidenceRuntimeParityOptions struct {
	OmitExecutionReceipt bool
	OmitRuntimeResult    bool
	OmitCodeChange       bool
	OmitAuditReport      bool

	// Negative-test seams for boundary enforcement. Any non-empty value widens
	// the local-only boundary and must prevent certification.
	UnsafeNetworkPolicy         string
	UnsafeForbiddenActionStatus string
}

// NativeEvidenceRuntimeParityRun is the bounded Work-owned Event 8 evidence
// packet. All EventGraph records are local in-memory fixture records.
type NativeEvidenceRuntimeParityRun struct {
	Mode                  string
	EventGraph            *v39.InMemoryStore
	FactoryOrderID        string
	RequirementID         string
	AcceptanceCriterionID string
	TaskID                string
	ActorIdentityID       string
	ActorInvocationID     string
	AuthorityRequestID    string
	AuthorityDecisionID   string
	ExecutionReceiptID    string
	RuntimeEnvelopeID     string
	RuntimeResultID       string
	ArtifactID            string
	CodeChangeID          string
	TestCaseID            string
	TestRunID             string
	GateResultID          string
	FailureID             string
	FactoryRuntimeID      string
	ReleaseCandidateID    string
	CertificationID       string
	RejectionID           string
	AuditReportID         string
	TraceCompleteness     v39.TraceCompletenessGateResult
	AuthorityPath         v39.RequiredPath
	Certification         *v39.Certification
	Rejection             *v39.Rejection
	AuditReport           *v39.AuditReport
	ParityReport          NativeEvidenceRuntimeParityReport
}

type NativeEvidenceRuntimeParityReport struct {
	Status                 string                            `json:"status"`
	Missing                []string                          `json:"missing,omitempty"`
	RequiredFamilies       []NativeEvidenceFamilyCheck       `json:"required_families"`
	TypeCounts             map[string]int                    `json:"type_counts"`
	TraceCompleted         bool                              `json:"trace_completed"`
	TraceStatus            v39.TraceCompletenessStatus       `json:"trace_status"`
	AuthorityPathCompleted bool                              `json:"authority_path_completed"`
	EvaluationErrors       []string                          `json:"evaluation_errors,omitempty"`
	LocalFixtureOnly       bool                              `json:"local_fixture_only"`
	ProposalBoundary       NativeEvidenceProposalBoundary    `json:"proposal_boundary"`
	ForbiddenActions       []NativeEvidenceForbiddenAction   `json:"forbidden_actions"`
	ResidualRisks          []NativeEvidenceResidualRiskState `json:"residual_risks"`
	EvidenceRefs           []string                          `json:"evidence_refs"`
}

type NativeEvidenceFamilyCheck struct {
	Family string `json:"family"`
	Type   string `json:"type"`
	Count  int    `json:"count"`
	Status string `json:"status"`
}

type NativeEvidenceProposalBoundary struct {
	PredecessorBuilder      string   `json:"predecessor_builder"`
	PredecessorDesign       string   `json:"predecessor_design"`
	Status                  string   `json:"status"`
	NativeParityFixture     bool     `json:"native_parity_fixture"`
	DocsHeldProposalOnly    bool     `json:"docs_held_proposal_only"`
	ProductionTruthClaimed  bool     `json:"production_truth_claimed"`
	PersistentWriteClaimed  bool     `json:"persistent_write_claimed"`
	RuntimeExecutionClaimed bool     `json:"runtime_execution_claimed"`
	Notes                   []string `json:"notes"`
}

type NativeEvidenceForbiddenAction struct {
	Action string `json:"action"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type NativeEvidenceResidualRiskState struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type nativeParityFixtureIDs struct {
	factoryOrder        string
	requirement         string
	acceptanceCriterion string
	task                string
	actorIdentity       string
	actorInvocation     string
	authorityRequest    string
	authorityDecision   string
	executionReceipt    string
	runtimeEnvelope     string
	runtimeResult       string
	artifact            string
	codeChange          string
	testCase            string
	testRun             string
	gateResult          string
	failure             string
	factoryRuntime      string
	releaseCandidate    string
	certification       string
	rejection           string
	auditReport         string
}

type nativeParityGraphRun struct {
	trace         v39.TraceCompletenessGateResult
	authorityPath v39.RequiredPath
	traceErr      error
	authorityErr  error
	certification *v39.Certification
	rejection     *v39.Rejection
	auditReport   *v39.AuditReport
}

// BuildNativeEvidenceRuntimeParityFixture builds the bounded Event 8 fixture
// authorized by DF-V4.0-EPIC-008-AUTHORITY-DECISION. It does not call GitHub,
// RuntimeBroker, external workers, persistent EventGraph storage, deployment,
// secret access, or protected settings.
func BuildNativeEvidenceRuntimeParityFixture(opts NativeEvidenceRuntimeParityOptions) (NativeEvidenceRuntimeParityRun, error) {
	ids := nativeParityIDs()
	graph, graphRun, err := nativeParityRecordEventGraph(ids, opts)
	if err != nil {
		return NativeEvidenceRuntimeParityRun{}, err
	}
	report := nativeParityEvaluate(graph, ids, graphRun.trace, graphRun.authorityPath, graphRun.traceErr, graphRun.authorityErr, opts)

	return NativeEvidenceRuntimeParityRun{
		Mode:                  NativeEvidenceRuntimeParityMode,
		EventGraph:            graph,
		FactoryOrderID:        ids.factoryOrder,
		RequirementID:         ids.requirement,
		AcceptanceCriterionID: ids.acceptanceCriterion,
		TaskID:                ids.task,
		ActorIdentityID:       ids.actorIdentity,
		ActorInvocationID:     ids.actorInvocation,
		AuthorityRequestID:    ids.authorityRequest,
		AuthorityDecisionID:   ids.authorityDecision,
		ExecutionReceiptID:    ids.executionReceipt,
		RuntimeEnvelopeID:     ids.runtimeEnvelope,
		RuntimeResultID:       ids.runtimeResult,
		ArtifactID:            ids.artifact,
		CodeChangeID:          ids.codeChange,
		TestCaseID:            ids.testCase,
		TestRunID:             ids.testRun,
		GateResultID:          ids.gateResult,
		FailureID:             ids.failure,
		FactoryRuntimeID:      ids.factoryRuntime,
		ReleaseCandidateID:    ids.releaseCandidate,
		CertificationID:       ids.certification,
		RejectionID:           ids.rejection,
		AuditReportID:         ids.auditReport,
		TraceCompleteness:     graphRun.trace,
		AuthorityPath:         graphRun.authorityPath,
		Certification:         graphRun.certification,
		Rejection:             graphRun.rejection,
		AuditReport:           graphRun.auditReport,
		ParityReport:          report,
	}, nil
}

func nativeParityRecordEventGraph(ids nativeParityFixtureIDs, opts NativeEvidenceRuntimeParityOptions) (*v39.InMemoryStore, nativeParityGraphRun, error) {
	graph := v39.NewInMemoryStore()
	createdAt := nativeParityFixtureTime()
	status := "certified"
	taskState := "certified"
	actorStatus := "succeeded"
	gateStatus := "pass"
	testRunStatus := "pass"
	failures := nativeParityInjectedFailures(opts)
	if len(failures) > 0 {
		status = "rejected"
		taskState = "rejected"
		actorStatus = "failed"
		gateStatus = "fail"
		testRunStatus = "fail"
	}

	records := []v39.Record{
		&v39.FactoryOrder{CommonNode: nativeParityCommon(ids.factoryOrder, v39.TypeFactoryOrder, status), FactoryOrderVersion: 1, SourceIntentHash: "sha256:docs-pr-162-merged-" + nativeParityDocsMergeSHA, SourceIntentRef: nativeParityAuthorityDoc, RiskClass: "high", ReleasePolicy: "human_approval_required"},
		&v39.Requirement{CommonNode: nativeParityCommon(ids.requirement, v39.TypeRequirement, "accepted"), FactoryOrderID: ids.factoryOrder, Text: "Represent the Event 7 docs-held FactoryOrder proposal evidence path as native Dark Factory v3.9 evidence families.", Source: "explicit", RiskClass: "high"},
		&v39.AcceptanceCriterion{CommonNode: nativeParityCommon(ids.acceptanceCriterion, v39.TypeAcceptanceCriterion, "accepted"), RequirementID: ids.requirement, Text: "Native evidence includes runtime envelope/result, authority decision/receipt, artifact/code change, tests, gate result, trace completeness, and audit evidence without protected side effects.", Source: "explicit", VerificationMethod: "test", RequiredEvidenceType: "native_eventgraph_runtime_parity_fixture", OwnerRole: "External Committee", RiskClass: "high"},
		&v39.Task{CommonNode: nativeParityCommon(ids.task, v39.TypeTask, taskState), FactoryOrderID: &ids.factoryOrder, Cell: "cell_df_v40_event8_native_evidence_runtime_parity", State: taskState, Priority: 1, RiskClass: "high", AttemptCount: 1},
		&v39.ActorIdentity{CommonNode: nativeParityCommon(ids.actorIdentity, v39.TypeActorIdentity, "active"), ActorID: nativeParityActorID, ActorType: "agent", IdentityMode: "fixture"},
		&v39.ActorInvocation{CommonNode: nativeParityCommon(ids.actorInvocation, v39.TypeActorInvocation, actorStatus), TaskID: ids.task, Runtime: "local", ActorID: nativeParityActorID, InputContractHash: "sha256:event8-native-parity-input", OutputContractHash: strPtr("sha256:event8-native-parity-output")},
		&v39.AuthorityRequest{CommonNode: nativeParityCommon(ids.authorityRequest, v39.TypeAuthorityRequest, "open"), ActorID: nativeParityActorID, ActorRole: "Operator", Action: "repo.work.native_evidence_runtime_parity_fixture.implement", TargetType: "repo", TargetID: "transpara-ai/work", RiskClass: "high", Reason: "Build one deterministic Work-owned native evidence/runtime parity fixture under merged Event 8 authority.", ProposedCommand: strPtr("BuildNativeEvidenceRuntimeParityFixture"), EvidenceRefs: []string{nativeParityAuthorityDoc, nativeParityDocsPR}},
		&v39.AuthorityDecision{CommonNode: nativeParityCommon(ids.authorityDecision, v39.TypeAuthorityDecision, "approved"), AuthorityRequestID: ids.authorityRequest, DeciderActorID: nativeParityExternalActorID, DeciderRole: "External Committee", Decision: "ApprovalRequired", Reason: "Local deterministic fixture may be built; future Work PR merge still requires explicit PR-visible External Committee approval on the exact head.", Scope: []string{"transpara-ai/work/native_evidence_runtime_parity.go", "transpara-ai/work/native_evidence_runtime_parity_test.go", "transpara-ai/work/docs/designs/native-evidence-runtime-parity.md"}, Conditions: []string{"in-memory v3.9 records only", "no RuntimeBroker", "no persistent EventGraph write", "no protected settings", "no production claim", "explicit PR-visible approval required before merge"}},
		&v39.RuntimeEnvelope{CommonNode: nativeParityCommon(ids.runtimeEnvelope, v39.TypeRuntimeEnvelope, "recorded"), RuntimeAdapterID: "local_native_evidence_parity_fixture", RuntimeAdapterVersion: "1", FactoryRuntimeVersionRef: ids.factoryRuntime, TaskID: ids.task, ActorID: nativeParityActorID, AuthorityDecisionRef: ids.authorityDecision, AllowedFiles: []string{"native_evidence_runtime_parity.go", "native_evidence_runtime_parity_test.go", "docs/designs/native-evidence-runtime-parity.md"}, DeniedFiles: []string{"go.mod", "go.sum", "../eventgraph/**", "../site/**", ".github/**", ".env", "secrets.env"}, AllowedCommands: []string{"go test ./...", "go vet ./...", "make verify"}, DeniedCommands: []string{"RuntimeBroker", "gh pr merge", "git push origin main", "kubectl apply", "terraform apply", "deploy", "secret access", "production operation", "value allocation"}, NetworkPolicy: nativeParityNetworkPolicy(opts), SecretsPolicy: "none", WorkingDirectory: "fixture://work/native-evidence-runtime-parity", Timeout: "5m", ResourceLimits: map[string]any{"persistent_eventgraph_write": false, "runtimebroker_execution": false, "protected_side_effects": false}, ExpectedOutputs: []string{"native evidence parity report", "in-memory v3.9 trace completeness", "audit report"}, OutputContract: map[string]any{"mode": NativeEvidenceRuntimeParityMode, "gate": "Gate R remains open pending separate docs evidence decision"}, TraceRequiredPaths: []string{"FactoryOrder -> Requirement -> AcceptanceCriterion -> Task", "ActorIdentity / AuthorityRequest / AuthorityDecision / ExecutionReceipt", "Task -> RuntimeEnvelope -> RuntimeResult", "Task -> Artifact -> CodeChange", "Task -> TestCase -> TestRun -> GateResult"}, PostRunValidationPlan: []string{"git diff --check", "go test ./...", "go vet ./...", "make verify", "exact-head adversarial review", "explicit PR-visible External Committee approval before merge"}, EnvelopeHash: "sha256:event8-native-parity-envelope"},
		&v39.Artifact{CommonNode: nativeParityCommon(ids.artifact, v39.TypeArtifact, "verified"), TaskID: &ids.task, ArtifactType: "report", Path: strPtr("fixture://work/native-evidence-runtime-parity/native-parity-report.json"), ContentHash: strPtr("sha256:event8-native-parity-report")},
		&v39.TestCase{CommonNode: nativeParityCommon(ids.testCase, v39.TypeTestCase, "active"), AcceptanceCriterionID: &ids.acceptanceCriterion, RequirementID: &ids.requirement, Name: "Event 8 native evidence/runtime parity fixture", TestType: "unit", Path: strPtr("native_evidence_runtime_parity_test.go")},
		&v39.TestRun{CommonNode: nativeParityCommon(ids.testRun, v39.TypeTestRun, testRunStatus), TestCaseID: &ids.testCase, ActorInvocationID: &ids.actorInvocation, Command: "go test ./..."},
		&v39.GateResult{CommonNode: nativeParityCommon(ids.gateResult, v39.TypeGateResult, gateStatus), FactoryOrderID: ids.factoryOrder, ReleaseCandidateID: &ids.releaseCandidate, GateName: "gate_r_native_evidence_runtime_parity_fixture", EvidenceRefs: []string{ids.testRun, ids.artifact}},
	}
	if !opts.OmitExecutionReceipt {
		records = append(records, &v39.ExecutionReceipt{CommonNode: nativeParityCommon(ids.executionReceipt, v39.TypeExecutionReceipt, "recorded"), AuthorityDecisionID: ids.authorityDecision, ActorInvocationID: &ids.actorInvocation, Action: "runtime.invoke.local.native_evidence_parity_fixture", TargetID: ids.task, Result: "succeeded", EvidenceRefs: []string{ids.runtimeResult, ids.artifact}})
	}
	if !opts.OmitRuntimeResult {
		exitStatus := "succeeded"
		if len(failures) > 0 {
			exitStatus = "failed"
		}
		// v3.9 trace completeness models runtime results as produced by the
		// RuntimeEnvelope, so this fixture records the envelope ID here.
		records = append(records, &v39.RuntimeResult{CommonNode: nativeParityCommon(ids.runtimeResult, v39.TypeRuntimeResult, "recorded"), InvocationID: ids.runtimeEnvelope, RuntimeAdapterID: "local_native_evidence_parity_fixture", StartedAt: createdAt, CompletedAt: createdAt.Add(time.Second), ExitStatus: exitStatus, ArtifactRefs: []string{ids.artifact}, ChangedFiles: []string{"native_evidence_runtime_parity.go", "native_evidence_runtime_parity_test.go", "docs/designs/native-evidence-runtime-parity.md"}, CommandLog: []string{"build in-memory v3.9 fixture", "evaluate trace completeness", "evaluate native parity report"}, NetworkAccessLog: []string{}, SecretAccessLog: []string{}, PolicyDecisionRefs: []string{ids.authorityDecision}, PostRunValidationRefs: []string{ids.testRun}})
	}
	if !opts.OmitCodeChange {
		records = append(records, &v39.CodeChange{CommonNode: nativeParityCommon(ids.codeChange, v39.TypeCodeChange, "verified"), ArtifactID: ids.artifact, ActorInvocationID: ids.actorInvocation, Repo: "transpara-ai/work", Path: "native_evidence_runtime_parity.go", BeforeHash: strPtr("sha256:absent"), AfterHash: "sha256:event8-native-parity-code", ChangeType: "create"})
	}
	if len(failures) > 0 {
		records = append(records, &v39.Failure{CommonNode: nativeParityCommon(ids.failure, v39.TypeFailure, "open"), FactoryOrderID: &ids.factoryOrder, TaskID: &ids.task, GateResultID: &ids.gateResult, TestRunID: &ids.testRun, FailureClass: "native_evidence_unavailable", Severity: "high", Summary: strings.Join(failures, "; ")})
	}

	if err := nativeParityAppendRecords(graph, records...); err != nil {
		return nil, nativeParityGraphRun{}, err
	}
	if _, err := graph.RecordFactoryRuntimeVersionBOM(&v39.FactoryRuntimeVersion{CommonNode: nativeParityCommon(ids.factoryRuntime, v39.TypeFactoryRuntimeVersion, "active"), RuntimeVersion: "3.9.0-event8-native-parity-local", CapabilityVersionRefs: []string{}, RuntimeRefs: []string{"work.local_native_evidence_parity_fixture@1"}}); err != nil {
		return nil, nativeParityGraphRun{}, err
	}
	if err := nativeParityAppendEdges(graph, ids, opts, len(failures) > 0, createdAt); err != nil {
		return nil, nativeParityGraphRun{}, err
	}
	rc, err := graph.RecordReleaseCandidate(&v39.ReleaseCandidate{CommonNode: nativeParityCommon(ids.releaseCandidate, v39.TypeReleaseCandidate, status), FactoryOrderID: ids.factoryOrder, FactoryRuntimeVersionID: &ids.factoryRuntime, ArtifactRefs: []string{ids.artifact}})
	if err != nil {
		return nil, nativeParityGraphRun{}, err
	}
	trace, traceErr := graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	authorityPath, authorityErr := graph.ActorAuthorityRequestDecisionReceipt(ids.authorityRequest)
	preDecision := nativeParityEvaluateWithAuditRequirement(graph, ids, trace, authorityPath, traceErr, authorityErr, opts, false)
	if opts.OmitAuditReport {
		preDecision.Status = "fail"
		preDecision.Missing = append(preDecision.Missing, "AuditReport unavailable")
	}

	var cert *v39.Certification
	var rejection *v39.Rejection
	if preDecision.Status == "pass" {
		cert, err = graph.CertifyReleaseCandidate(&v39.Certification{CommonNode: nativeParityCommon(ids.certification, v39.TypeCertification, "certified"), ReleaseCandidateID: ids.releaseCandidate, CertifierActorID: nativeParityExternalActorID, Reason: "Event 8 local fixture evidence is complete for native parity only; merge still requires PR-visible External Committee approval.", EvidenceRefs: []string{ids.gateResult, ids.authorityDecision, ids.executionReceipt}})
		if err != nil {
			return nil, nativeParityGraphRun{}, err
		}
	} else {
		rejection, err = graph.RejectReleaseCandidate(&v39.Rejection{CommonNode: nativeParityCommon(ids.rejection, v39.TypeRejection, "rejected"), ReleaseCandidateID: ids.releaseCandidate, RejectorActorID: nativeParityExternalActorID, Reason: "Native evidence/runtime parity fixture is incomplete and must fail closed.", EvidenceRefs: append([]string{ids.gateResult}, preDecision.Missing...)})
		if err != nil {
			return nil, nativeParityGraphRun{}, err
		}
	}

	var audit *v39.AuditReport
	if !opts.OmitAuditReport {
		auditStatus := "complete"
		if preDecision.Status != "pass" {
			auditStatus = "incomplete"
		}
		audit, err = graph.ReconstructAuditReport(ids.releaseCandidate, &v39.AuditReport{CommonNode: nativeParityCommon(ids.auditReport, v39.TypeAuditReport, auditStatus), TargetType: "release_candidate", TargetID: ids.releaseCandidate})
		if err != nil {
			return nil, nativeParityGraphRun{}, err
		}
	}

	trace, traceErr = graph.EvaluateTraceCompletenessGate(rc.CommonNode.ID)
	authorityPath, authorityErr = graph.ActorAuthorityRequestDecisionReceipt(ids.authorityRequest)
	return graph, nativeParityGraphRun{trace: trace, authorityPath: authorityPath, traceErr: traceErr, authorityErr: authorityErr, certification: cert, rejection: rejection, auditReport: audit}, nil
}

func nativeParityEvaluate(graph *v39.InMemoryStore, ids nativeParityFixtureIDs, trace v39.TraceCompletenessGateResult, authorityPath v39.RequiredPath, traceErr error, authorityErr error, opts NativeEvidenceRuntimeParityOptions) NativeEvidenceRuntimeParityReport {
	return nativeParityEvaluateWithAuditRequirement(graph, ids, trace, authorityPath, traceErr, authorityErr, opts, true)
}

func nativeParityEvaluateWithAuditRequirement(graph *v39.InMemoryStore, ids nativeParityFixtureIDs, trace v39.TraceCompletenessGateResult, authorityPath v39.RequiredPath, traceErr error, authorityErr error, opts NativeEvidenceRuntimeParityOptions, requireAudit bool) NativeEvidenceRuntimeParityReport {
	report := NativeEvidenceRuntimeParityReport{
		Status:                 "pass",
		TypeCounts:             map[string]int{},
		TraceCompleted:         trace.Completed,
		TraceStatus:            trace.Status,
		AuthorityPathCompleted: authorityPath.Completed,
		LocalFixtureOnly:       true,
		ProposalBoundary: NativeEvidenceProposalBoundary{
			PredecessorBuilder:      "BuildFactoryOrderDevelopmentProposal",
			PredecessorDesign:       "docs/designs/factory-order-proposal-evidence-path.md",
			Status:                  "native_parity_fixture_only",
			NativeParityFixture:     true,
			DocsHeldProposalOnly:    true,
			ProductionTruthClaimed:  false,
			PersistentWriteClaimed:  false,
			RuntimeExecutionClaimed: false,
			Notes: []string{
				"Event 7 proposal evidence remains proposal-only and not production truth.",
				"Event 8 fixture stores native evidence in a Work-owned v3.9 in-memory store only.",
				"Gate R remains open pending future governed docs evidence-decision closeout.",
			},
		},
		ForbiddenActions: []NativeEvidenceForbiddenAction{
			{Action: "RuntimeBroker execution", Status: "not_run", Reason: "not authorized by Event 8 Work fixture grant"},
			{Action: "persistent EventGraph write", Status: "not_run", Reason: "fixture uses v3.9 in-memory store only"},
			{Action: "GitHub mutation", Status: "not_run", Reason: "builder has no GitHub client path"},
			{Action: "protected settings mutation", Status: "not_run", Reason: "outside authorized scope"},
			{Action: "production deploy/go-live", Status: "not_run", Reason: "Gate R remains open and production is not authorized"},
			{Action: "value allocation", Status: "not_run", Reason: "outside authorized scope"},
		},
		ResidualRisks: []NativeEvidenceResidualRiskState{
			{ID: "R-001", Status: "unresolved_excluded", Reason: "runtime authority remains separate"},
			{ID: "R-002", Status: "unresolved_excluded", Reason: "protected side effects remain unauthorized"},
			{ID: "R-003", Status: "unresolved_excluded", Reason: "policy-bundle evidence remains future work"},
		},
		EvidenceRefs: []string{ids.factoryOrder, ids.requirement, ids.acceptanceCriterion, ids.task, ids.authorityDecision, ids.runtimeEnvelope, ids.artifact, ids.testRun, ids.gateResult},
	}

	for _, typ := range nativeParityRequiredRecordTypes(requireAudit) {
		count := len(graph.ByType(typ))
		report.TypeCounts[typ] = count
		status := "pass"
		if count == 0 {
			status = "missing"
			report.Missing = append(report.Missing, typ+" missing")
		}
		report.RequiredFamilies = append(report.RequiredFamilies, NativeEvidenceFamilyCheck{Family: nativeParityFamilyLabel(typ), Type: typ, Count: count, Status: status})
	}
	if !trace.Completed || trace.Status != v39.TraceCompletenessPassed {
		report.Missing = append(report.Missing, "TraceCompletenessGate incomplete")
		report.Missing = append(report.Missing, trace.Missing...)
	}
	if traceErr != nil {
		report.EvaluationErrors = append(report.EvaluationErrors, "TraceCompletenessGate: "+traceErr.Error())
	}
	if !authorityPath.Completed {
		report.Missing = append(report.Missing, "AuthorityRequest/AuthorityDecision/ExecutionReceipt path incomplete")
		report.Missing = append(report.Missing, authorityPath.Missing...)
	}
	if authorityErr != nil {
		report.EvaluationErrors = append(report.EvaluationErrors, "AuthorityRequestDecisionReceipt: "+authorityErr.Error())
	}
	report.Missing = append(report.Missing, nativeParityLocalFixtureMissing(graph, ids)...)
	if opts.UnsafeForbiddenActionStatus != "" && len(report.ForbiddenActions) > 0 {
		report.ForbiddenActions[0].Status = opts.UnsafeForbiddenActionStatus
	}
	for _, action := range report.ForbiddenActions {
		if action.Status != "not_run" {
			report.Missing = append(report.Missing, "forbidden action status not fail-closed: "+action.Action)
		}
	}
	if len(report.Missing) > 0 {
		report.Status = "fail"
	}
	return report
}

func nativeParityLocalFixtureMissing(graph *v39.InMemoryStore, ids nativeParityFixtureIDs) []string {
	var missing []string
	envelopeRecord, err := graph.Get(ids.runtimeEnvelope)
	if err != nil {
		return []string{"RuntimeEnvelope unavailable for local fixture checks"}
	}
	envelope, ok := envelopeRecord.(*v39.RuntimeEnvelope)
	if !ok {
		return []string{"RuntimeEnvelope record has unexpected type"}
	}
	if envelope.NetworkPolicy != "disabled" {
		missing = append(missing, "RuntimeEnvelope network_policy is not disabled")
	}
	if envelope.SecretsPolicy != "none" {
		missing = append(missing, "RuntimeEnvelope secrets_policy is not none")
	}
	if !strings.HasPrefix(envelope.WorkingDirectory, "fixture://") {
		missing = append(missing, "RuntimeEnvelope working_directory is not fixture scoped")
	}
	for _, denied := range []string{"RuntimeBroker", "gh pr merge", "deploy", "secret access", "production operation", "value allocation"} {
		if !nativeParityContains(envelope.DeniedCommands, denied) {
			missing = append(missing, "RuntimeEnvelope denied_commands missing "+denied)
		}
	}
	resultRecord, err := graph.Get(ids.runtimeResult)
	if err == nil {
		result, ok := resultRecord.(*v39.RuntimeResult)
		if !ok {
			missing = append(missing, "RuntimeResult record has unexpected type")
		} else {
			if len(result.NetworkAccessLog) != 0 {
				missing = append(missing, "RuntimeResult network_access_log is not empty")
			}
			if len(result.SecretAccessLog) != 0 {
				missing = append(missing, "RuntimeResult secret_access_log is not empty")
			}
			for _, path := range result.ChangedFiles {
				if !nativeParityContains(nativeParityAllowedPaths(), path) {
					missing = append(missing, "RuntimeResult changed_files contains unauthorized path "+path)
				}
			}
		}
	}
	receiptRecord, err := graph.Get(ids.executionReceipt)
	if err == nil {
		receipt, ok := receiptRecord.(*v39.ExecutionReceipt)
		if !ok {
			missing = append(missing, "ExecutionReceipt record has unexpected type")
		} else {
			if receipt.Action != "runtime.invoke.local.native_evidence_parity_fixture" {
				missing = append(missing, "ExecutionReceipt action is outside local fixture scope")
			}
			if receipt.Result != "succeeded" {
				missing = append(missing, "ExecutionReceipt result is not succeeded")
			}
			if receipt.TargetID != ids.task {
				missing = append(missing, "ExecutionReceipt target is not the fixture task")
			}
		}
	}
	return missing
}

func nativeParityAppendRecords(graph *v39.InMemoryStore, records ...v39.Record) error {
	for _, record := range records {
		if _, err := graph.AppendRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func nativeParityAppendEdges(graph *v39.InMemoryStore, ids nativeParityFixtureIDs, opts NativeEvidenceRuntimeParityOptions, includeFailure bool, createdAt time.Time) error {
	edges := []v39.CommonEdge{
		nativeParityEdge("fo_req", v39.EdgeRequires, ids.factoryOrder, ids.requirement, createdAt),
		nativeParityEdge("req_ac", v39.EdgeRequires, ids.requirement, ids.acceptanceCriterion, createdAt),
		nativeParityEdge("ac_task", v39.EdgeDecomposedInto, ids.acceptanceCriterion, ids.task, createdAt),
		nativeParityEdge("identity_auth_request", v39.EdgeRequestedAuthority, ids.actorIdentity, ids.authorityRequest, createdAt),
		nativeParityEdge("task_invocation", v39.EdgeInvoked, ids.task, ids.actorInvocation, createdAt),
		nativeParityEdge("invocation_auth_request", v39.EdgeRequestedAuthority, ids.actorInvocation, ids.authorityRequest, createdAt),
		nativeParityEdge("auth_decision", v39.EdgeDecidedBy, ids.authorityRequest, ids.authorityDecision, createdAt),
		nativeParityEdge("task_envelope", v39.EdgeUsedEnvelope, ids.task, ids.runtimeEnvelope, createdAt),
		nativeParityEdge("task_artifact", v39.EdgeProduced, ids.task, ids.artifact, createdAt),
		nativeParityEdge("task_testcase", v39.EdgeVerifies, ids.task, ids.testCase, createdAt),
		nativeParityEdge("testcase_testrun", v39.EdgeVerifies, ids.testCase, ids.testRun, createdAt),
		nativeParityEdge("testrun_gate", v39.EdgeProduced, ids.testRun, ids.gateResult, createdAt),
	}
	if !opts.OmitExecutionReceipt {
		edges = append(edges, nativeParityEdge("auth_receipt", v39.EdgeReceiptedBy, ids.authorityDecision, ids.executionReceipt, createdAt))
	}
	if !opts.OmitRuntimeResult {
		edges = append(edges, nativeParityEdge("envelope_result", v39.EdgeProduced, ids.runtimeEnvelope, ids.runtimeResult, createdAt))
	}
	if !opts.OmitCodeChange {
		edges = append(edges, nativeParityEdge("artifact_code_change", v39.EdgeModified, ids.artifact, ids.codeChange, createdAt))
	}
	if includeFailure {
		edges = append(edges, nativeParityEdge("gate_failure", v39.EdgeFailedBy, ids.gateResult, ids.failure, createdAt))
	}
	for _, edge := range edges {
		if _, err := graph.AppendEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func nativeParityInjectedFailures(opts NativeEvidenceRuntimeParityOptions) []string {
	var missing []string
	if opts.OmitExecutionReceipt {
		missing = append(missing, "ExecutionReceipt unavailable")
	}
	if opts.OmitRuntimeResult {
		missing = append(missing, "RuntimeResult unavailable")
	}
	if opts.OmitCodeChange {
		missing = append(missing, "CodeChange unavailable")
	}
	if opts.OmitAuditReport {
		missing = append(missing, "AuditReport unavailable")
	}
	if opts.UnsafeNetworkPolicy != "" {
		missing = append(missing, "RuntimeEnvelope network_policy widened")
	}
	if opts.UnsafeForbiddenActionStatus != "" {
		missing = append(missing, "Forbidden action status widened")
	}
	return missing
}

func nativeParityNetworkPolicy(opts NativeEvidenceRuntimeParityOptions) string {
	if opts.UnsafeNetworkPolicy != "" {
		return opts.UnsafeNetworkPolicy
	}
	return "disabled"
}

func nativeParityRequiredRecordTypes(requireAudit bool) []string {
	types := []string{
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
	}
	if requireAudit {
		types = append(types, v39.TypeAuditReport)
	}
	return types
}

func nativeParityFamilyLabel(typ string) string {
	switch typ {
	case v39.TypeFactoryOrder, v39.TypeRequirement, v39.TypeAcceptanceCriterion, v39.TypeTask:
		return "factory_order_trace"
	case v39.TypeAuthorityRequest, v39.TypeAuthorityDecision, v39.TypeExecutionReceipt, v39.TypeActorIdentity:
		return "authority_trace"
	case v39.TypeRuntimeEnvelope, v39.TypeActorInvocation, v39.TypeRuntimeResult:
		return "runtime_trace"
	case v39.TypeArtifact, v39.TypeCodeChange:
		return "artifact_code_trace"
	case v39.TypeTestCase, v39.TypeTestRun, v39.TypeGateResult:
		return "validation_trace"
	case v39.TypeFactoryRuntimeVersion, v39.TypeReleaseCandidate, v39.TypeAuditReport:
		return "release_audit_trace"
	default:
		return "native_evidence"
	}
}

func nativeParityIDs() nativeParityFixtureIDs {
	return nativeParityFixtureIDs{
		factoryOrder:        "fo_df_v40_event8_native_parity_001",
		requirement:         "req_df_v40_event8_native_parity_001",
		acceptanceCriterion: "ac_df_v40_event8_native_parity_001",
		task:                "tsk_df_v40_event8_native_parity_001",
		actorIdentity:       "actor_identity_df_v40_event8_native_parity",
		actorInvocation:     "inv_df_v40_event8_native_parity_001",
		authorityRequest:    "auth_req_df_v40_event8_native_parity_001",
		authorityDecision:   "auth_dec_df_v40_event8_native_parity_001",
		executionReceipt:    "exec_df_v40_event8_native_parity_001",
		runtimeEnvelope:     "env_df_v40_event8_native_parity_001",
		runtimeResult:       "res_df_v40_event8_native_parity_001",
		artifact:            "artifact_df_v40_event8_native_parity_report",
		codeChange:          "codechange_df_v40_event8_native_parity_go",
		testCase:            "tc_df_v40_event8_native_parity_001",
		testRun:             "tr_df_v40_event8_native_parity_001",
		gateResult:          "gate_df_v40_event8_native_parity_001",
		failure:             "fail_df_v40_event8_native_parity_001",
		factoryRuntime:      "frv_df_v40_event8_native_parity_local",
		releaseCandidate:    "rc_df_v40_event8_native_parity_001",
		certification:       "cert_df_v40_event8_native_parity_001",
		rejection:           "rej_df_v40_event8_native_parity_001",
		auditReport:         "audit_df_v40_event8_native_parity_001",
	}
}

func nativeParityAllowedPaths() []string {
	return []string{
		"native_evidence_runtime_parity.go",
		"native_evidence_runtime_parity_test.go",
		"docs/designs/native-evidence-runtime-parity.md",
	}
}

func nativeParityCommon(id, typ, status string) v39.CommonNode {
	return v39.CommonNode{
		ID:             id,
		Type:           typ,
		CreatedAt:      nativeParityFixtureTime(),
		CreatedBy:      nativeParityActorID,
		Status:         strPtr(status),
		Version:        "4.0.0-event8-native-parity",
		IdempotencyKey: "idem_" + id,
		CorrelationID:  nativeParityCorrelationID,
		SourceRefs: []string{
			nativeParityAuthorityDoc,
			nativeParityDocsPR,
			"docs-pr-162-merge-" + nativeParityDocsMergeSHA,
			"docs-pr-162-reviewed-head-" + nativeParityDocsReviewedHead,
		},
	}
}

func nativeParityEdge(label, typ, from, to string, createdAt time.Time) v39.CommonEdge {
	id := "edge_df_v40_event8_" + label + ":" + from + ":" + typ + ":" + to
	return v39.CommonEdge{
		ID:             id,
		Type:           typ,
		FromID:         from,
		ToID:           to,
		CreatedAt:      createdAt,
		CreatedBy:      nativeParityActorID,
		CorrelationID:  nativeParityCorrelationID,
		IdempotencyKey: "idem_" + id,
		EvidenceRefs:   []string{nativeParityAuthorityDoc},
	}
}

func nativeParityFixtureTime() time.Time {
	t, err := time.Parse(time.RFC3339, nativeParityFixtureTimeRFC)
	if err != nil {
		panic(err)
	}
	return t
}

func nativeParityContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func NativeEvidenceRuntimeParityStatus(run NativeEvidenceRuntimeParityRun) (string, error) {
	if run.EventGraph == nil {
		return "fail", errors.New("native evidence runtime parity run has nil EventGraph")
	}
	if run.ParityReport.Status == "" {
		return "fail", errors.New("native evidence runtime parity report is missing")
	}
	if run.ParityReport.Status != "pass" {
		return run.ParityReport.Status, fmt.Errorf("native evidence runtime parity incomplete: %s", strings.Join(run.ParityReport.Missing, "; "))
	}
	return "pass", nil
}
