package work_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

func countLifecycleTransitions(t *testing.T, s *store.InMemoryStore) int {
	t.Helper()
	page, err := s.ByType(work.EventTypeTaskLifecycleTransitioned, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType lifecycle: %v", err)
	}
	return len(page.Items())
}

func pathToState(target work.TaskStatus) []work.TaskStatus {
	switch target {
	case work.StatusCreated:
		return nil
	case work.StatusReady:
		return []work.TaskStatus{work.StatusReady}
	case work.StatusRunning:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning}
	case work.StatusBlocked:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusBlocked}
	case work.StatusFailed:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed}
	case work.StatusRepairRequired:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired}
	case work.StatusRepairRunning:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired, work.StatusRepairRunning}
	case work.StatusRepaired:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired, work.StatusRepairRunning, work.StatusRepaired}
	case work.StatusVerificationRunning:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired, work.StatusRepairRunning, work.StatusRepaired, work.StatusVerificationRunning}
	case work.StatusVerified:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified}
	case work.StatusCertified:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified, work.StatusCertified}
	case work.StatusRejected:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified}
	case work.StatusSuperseded:
		return []work.TaskStatus{work.StatusSuperseded}
	case work.StatusPolicyBlocked:
		return []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusPolicyBlocked}
	default:
		return nil
	}
}

func TestTaskStoreV39_LinkageProjectsFactoryOrderRequirementAcceptanceCriterionTask(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:                  "Implement deterministic artifact task",
		Description:            "Base Slice 0 work item",
		CanonicalTaskID:        "tsk_hello_artifact_task",
		FactoryOrderID:         "fo_hello_artifact_order",
		RequirementIDs:         []string{"req_hello_artifact"},
		AcceptanceCriterionIDs: []string{"ac_hello_artifact"},
		Cell:                   "implementation",
		RiskClass:              "low",
		ExpectedOutputs:        []string{"hello.txt"},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39: %v", err)
	}

	projection, err := ts.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask: %v", err)
	}
	if projection.Linkage.FactoryOrderID != "fo_hello_artifact_order" {
		t.Fatalf("FactoryOrderID = %q", projection.Linkage.FactoryOrderID)
	}
	if !slices.Equal(projection.Linkage.RequirementIDs, []string{"req_hello_artifact"}) {
		t.Fatalf("RequirementIDs = %#v", projection.Linkage.RequirementIDs)
	}
	if !slices.Equal(projection.Linkage.AcceptanceCriterionIDs, []string{"ac_hello_artifact"}) {
		t.Fatalf("AcceptanceCriterionIDs = %#v", projection.Linkage.AcceptanceCriterionIDs)
	}
	if projection.Linkage.CanonicalTaskID != "tsk_hello_artifact_task" {
		t.Fatalf("CanonicalTaskID = %q", projection.Linkage.CanonicalTaskID)
	}
}

func TestTaskStoreV39_LinkTaskReplaysLatestLinkage(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Link later", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	linkage := work.TaskLinkage{
		CanonicalTaskID:        "tsk_linked_task",
		FactoryOrderID:         "fo_linked_order",
		RequirementIDs:         []string{"req_linked_requirement"},
		AcceptanceCriterionIDs: []string{"ac_linked_acceptance"},
	}
	if err := ts.LinkTask(testActor, task.ID, linkage, causes, testConv); err != nil {
		t.Fatalf("LinkTask: %v", err)
	}

	replayed := newTaskStore(t, s)
	projection, err := replayed.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask replayed: %v", err)
	}
	if projection.Linkage.FactoryOrderID != linkage.FactoryOrderID {
		t.Fatalf("FactoryOrderID = %q", projection.Linkage.FactoryOrderID)
	}
}

func TestTaskStoreV39_LifecycleReplayReconstructsCanonicalCertificationPath(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Lifecycle task", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified, work.StatusCertified} {
		if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}
	replayed := newTaskStore(t, s)
	status, err := replayed.GetStatus(task.ID)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != work.StatusCertified {
		t.Fatalf("status = %q; want certified", status)
	}

	err = ts.TransitionTask(testActor, task.ID, work.StatusRunning, "restart completed task", nil, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("certified -> running error = %v; want ErrInvalidLifecycleTransition", err)
	}
}

func TestTaskStoreV39_InvalidTransitionsDoNotMutateState(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Invalid transition", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	beforeEvents := countLifecycleTransitions(t, s)
	err = ts.TransitionTask(testActor, task.ID, work.StatusRunning, "skip ready", nil, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("created -> running error = %v; want ErrInvalidLifecycleTransition", err)
	}
	afterEvents := countLifecycleTransitions(t, s)
	if afterEvents != beforeEvents {
		t.Fatalf("lifecycle events = %d; want %d after invalid transition", afterEvents, beforeEvents)
	}
	if status, err := ts.GetStatus(task.ID); err != nil || status != work.StatusCreated {
		t.Fatalf("status = %q, %v; want created", status, err)
	}
}

func TestTaskStoreV39_CompatibilityMappingFromLegacyEvents(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	parent, err := ts.Create(testActor, "Parent", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	child, err := ts.Create(testActor, "Child", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	if status, err := ts.GetStatus(child.ID); err != nil || status != work.StatusCreated {
		t.Fatalf("canonical initial status = %q, %v; want created", status, err)
	}
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusPending {
		t.Fatalf("initial status = %q, %v; want pending", status, err)
	}
	addRequiredGateArtifacts(t, ts, child.ID, causes)
	if status, err := ts.GetStatus(child.ID); err != nil || status != work.StatusCreated {
		t.Fatalf("canonical status after readiness = %q, %v; want created", status, err)
	}
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusReady {
		t.Fatalf("ready status = %q, %v; want ready", status, err)
	}
	if err := ts.Assign(testActor, child.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusAssigned {
		t.Fatalf("assigned status = %q, %v; want assigned", status, err)
	}
	if err := ts.AddDependency(testActor, child.ID, parent.ID, causes, testConv); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}
	if status, err := ts.GetStatus(child.ID); err != nil || status != work.StatusCreated {
		t.Fatalf("canonical blocked-by-legacy status = %q, %v; want created", status, err)
	}
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusBlocked {
		t.Fatalf("blocked status = %q, %v; want blocked", status, err)
	}
	completeWithArtifact(t, ts, testActor, parent.ID, "done", causes, testConv)
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusAssigned {
		t.Fatalf("unblocked-by-dependency status = %q, %v; want assigned", status, err)
	}
	completeWithArtifact(t, ts, testActor, child.ID, "done", causes, testConv)
	if status, err := ts.GetCompatibilityStatus(child.ID); err != nil || status != work.LegacyStatusCompleted {
		t.Fatalf("completed status = %q, %v; want completed", status, err)
	}
	if status, err := ts.GetStatus(child.ID); err != nil || status != work.StatusCreated {
		t.Fatalf("canonical status after legacy completion = %q, %v; want created", status, err)
	}
}

func TestTaskStoreV39_RejectedOnlyFromVerificationRunningOrVerified(t *testing.T) {
	for _, tc := range []struct {
		name string
		path []work.TaskStatus
	}{
		{name: "verification_running", path: []work.TaskStatus{
			work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired,
			work.StatusRepairRunning, work.StatusRepaired, work.StatusVerificationRunning,
		}},
		{name: "verified", path: []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task, err := ts.Create(testActor, "Reject valid", "", causes, testConv)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			for _, state := range tc.path {
				if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
			if err := ts.RejectTask(testActor, task.ID, "verification rejected", []string{"gate_reject"}, causes, testConv); err != nil {
				t.Fatalf("RejectTask: %v", err)
			}
			if status, err := ts.GetStatus(task.ID); err != nil || status != work.StatusRejected {
				t.Fatalf("status = %q, %v; want rejected", status, err)
			}
		})
	}

	for _, target := range []work.TaskStatus{work.StatusCreated, work.StatusReady, work.StatusRunning, work.StatusBlocked, work.StatusFailed, work.StatusRepairRequired, work.StatusRepairRunning, work.StatusRepaired, work.StatusPolicyBlocked} {
		t.Run("invalid_from_"+string(target), func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task, err := ts.Create(testActor, "Reject invalid", "", causes, testConv)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			path := pathToState(target)
			for _, state := range path {
				if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
			err = ts.RejectTask(testActor, task.ID, "not allowed here", []string{"gate_reject"}, causes, testConv)
			if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
				t.Fatalf("%s -> rejected error = %v; want invalid transition", target, err)
			}
		})
	}
}

func TestTaskStoreV39_SupersededFromAnyNonTerminalAndTerminalAfterward(t *testing.T) {
	nonTerminalStates := []work.TaskStatus{
		work.StatusCreated, work.StatusReady, work.StatusRunning, work.StatusBlocked, work.StatusFailed,
		work.StatusRepairRequired, work.StatusRepairRunning, work.StatusRepaired, work.StatusVerificationRunning,
		work.StatusVerified, work.StatusPolicyBlocked,
	}
	for _, target := range nonTerminalStates {
		t.Run(string(target), func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task, err := ts.Create(testActor, "Supersede from "+string(target), "", causes, testConv)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			for _, state := range pathToState(target) {
				if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
			if err := ts.SupersedeTask(testActor, task.ID, "tsk_replacement_"+string(target), "duplicate", nil, causes, testConv); err != nil {
				t.Fatalf("%s -> superseded: %v", target, err)
			}
			if err := ts.TransitionTask(testActor, task.ID, work.StatusReady, "terminal check", nil, causes, testConv); !errors.Is(err, work.ErrInvalidLifecycleTransition) {
				t.Fatalf("superseded -> ready error = %v; want invalid transition", err)
			}
		})
	}

	for _, terminal := range []work.TaskStatus{work.StatusCertified, work.StatusRejected, work.StatusSuperseded} {
		t.Run("terminal_"+string(terminal), func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)
			task, err := ts.Create(testActor, "Terminal "+string(terminal), "", causes, testConv)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			for _, state := range pathToState(terminal) {
				if state == work.StatusSuperseded {
					if err := ts.SupersedeTask(testActor, task.ID, "tsk_terminal_replacement", "duplicate", nil, causes, testConv); err != nil {
						t.Fatalf("SupersedeTask: %v", err)
					}
					continue
				}
				if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
					t.Fatalf("TransitionTask to %s: %v", state, err)
				}
			}
			if terminal == work.StatusRejected {
				if err := ts.RejectTask(testActor, task.ID, "reject", nil, causes, testConv); err != nil {
					t.Fatalf("RejectTask: %v", err)
				}
			}
			err = ts.SupersedeTask(testActor, task.ID, "tsk_after_terminal", "too late", nil, causes, testConv)
			if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
				t.Fatalf("%s -> superseded error = %v; want invalid transition", terminal, err)
			}
		})
	}
}

func TestTaskStoreV39_PolicyBlockedRejectedAndSupersededBehavior(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	policyTask, err := ts.Create(testActor, "Policy blocked", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create policy task: %v", err)
	}
	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusPolicyBlocked} {
		if err := ts.TransitionTask(testActor, policyTask.ID, state, "advance", []string{"gate_policy"}, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}
	projection, err := ts.ProjectTask(policyTask.ID)
	if err != nil {
		t.Fatalf("ProjectTask policy: %v", err)
	}
	if projection.Status != work.StatusPolicyBlocked || !projection.Blocked {
		t.Fatalf("policy projection status=%q blocked=%v", projection.Status, projection.Blocked)
	}
	if err := ts.TransitionTask(testActor, policyTask.ID, work.StatusRunning, "should not run", nil, causes, testConv); !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("policy_blocked -> running error = %v; want invalid transition", err)
	}
	if err := ts.TransitionTask(testActor, policyTask.ID, work.StatusReady, "should not retry", nil, causes, testConv); !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("policy_blocked -> ready error = %v; want invalid transition", err)
	}

	rejected, err := ts.Create(testActor, "Reject me", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create rejected: %v", err)
	}
	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusVerified} {
		if err := ts.TransitionTask(testActor, rejected.ID, state, "advance", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask rejected seed to %s: %v", state, err)
		}
	}
	if err := ts.RejectTask(testActor, rejected.ID, "not accepted", []string{"fail_rejected"}, causes, testConv); err != nil {
		t.Fatalf("RejectTask: %v", err)
	}
	if err := ts.Assign(testActor, rejected.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("legacy Assign remains append-only compatibility event: %v", err)
	}
	if status, err := ts.GetStatus(rejected.ID); err != nil || status != work.StatusRejected {
		t.Fatalf("rejected status = %q, %v; want rejected", status, err)
	}

	superseded, err := ts.Create(testActor, "Supersede me", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create superseded: %v", err)
	}
	if err := ts.SupersedeTask(testActor, superseded.ID, "tsk_replacement", "duplicate work", nil, causes, testConv); err != nil {
		t.Fatalf("SupersedeTask: %v", err)
	}
	supProjection, err := ts.ProjectTask(superseded.ID)
	if err != nil {
		t.Fatalf("ProjectTask superseded: %v", err)
	}
	if supProjection.Status != work.StatusSuperseded || supProjection.SupersededBy != "tsk_replacement" {
		t.Fatalf("superseded projection status=%q by=%q", supProjection.Status, supProjection.SupersededBy)
	}
}

func TestTaskStoreV39_RepairPathWithEvidence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Repair path", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning, work.StatusFailed, work.StatusRepairRequired} {
		if err := ts.TransitionTask(testActor, task.ID, state, "advance", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}
	if err := ts.AttachFailureRepairReferences(testActor, task.ID, work.FailureRepairReferences{
		FailureIDs:       []string{"fail_unit_tests"},
		RepairAttemptIDs: []string{"rep_unit_tests"},
	}, "seed repair evidence", causes, testConv); err != nil {
		t.Fatalf("AttachFailureRepairReferences: %v", err)
	}
	for _, state := range []work.TaskStatus{work.StatusRepairRunning, work.StatusRepaired, work.StatusVerificationRunning} {
		if err := ts.TransitionTask(testActor, task.ID, state, "repair flow", nil, causes, testConv); err != nil {
			t.Fatalf("TransitionTask to %s: %v", state, err)
		}
	}
	if err := ts.AttachVerificationEvidence(testActor, task.ID, work.VerificationEvidence{
		TestCaseIDs:   []string{"tc_unit"},
		TestRunIDs:    []string{"tr_unit"},
		GateResultIDs: []string{"gate_unit"},
	}, "tests pass", causes, testConv); err != nil {
		t.Fatalf("AttachVerificationEvidence: %v", err)
	}
	if err := ts.TransitionTask(testActor, task.ID, work.StatusVerified, "verified", []string{"tr_unit", "gate_unit"}, causes, testConv); err != nil {
		t.Fatalf("TransitionTask verified: %v", err)
	}

	projection, err := ts.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask: %v", err)
	}
	if projection.Status != work.StatusVerified {
		t.Fatalf("status = %q; want verified", projection.Status)
	}
	if !slices.Equal(projection.FailureRepair.FailureIDs, []string{"fail_unit_tests"}) {
		t.Fatalf("FailureIDs = %#v", projection.FailureRepair.FailureIDs)
	}
	if !slices.Equal(projection.FailureRepair.RepairAttemptIDs, []string{"rep_unit_tests"}) {
		t.Fatalf("RepairAttemptIDs = %#v", projection.FailureRepair.RepairAttemptIDs)
	}
	if !slices.Equal(projection.Verification.TestRunIDs, []string{"tr_unit"}) {
		t.Fatalf("TestRunIDs = %#v", projection.Verification.TestRunIDs)
	}
	if !slices.Equal(projection.Verification.GateResultIDs, []string{"gate_unit"}) {
		t.Fatalf("GateResultIDs = %#v", projection.Verification.GateResultIDs)
	}
}

func TestTaskStoreV39_AttachEvidenceRequiresReferences(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Evidence validation", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.AttachVerificationEvidence(testActor, task.ID, work.VerificationEvidence{}, "", causes, testConv); err == nil {
		t.Fatal("AttachVerificationEvidence accepted empty refs")
	}
	if err := ts.AttachFailureRepairReferences(testActor, task.ID, work.FailureRepairReferences{}, "", causes, testConv); err == nil {
		t.Fatal("AttachFailureRepairReferences accepted empty refs")
	}
	if err := ts.AttachVerificationEvidence(testActor, task.ID, work.VerificationEvidence{TestRunIDs: []string{"not_a_test_run"}}, "", causes, testConv); err == nil {
		t.Fatal("AttachVerificationEvidence accepted invalid test run prefix")
	}
}

func TestTaskStoreV39_ProjectTaskReplayCorrectnessAcrossStoreInstances(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:                  "Replay full projection",
		CanonicalTaskID:        "tsk_replay",
		FactoryOrderID:         "fo_replay",
		RequirementIDs:         []string{"req_replay"},
		AcceptanceCriterionIDs: []string{"ac_replay"},
		Cell:                   "verification",
		RiskClass:              "medium",
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39: %v", err)
	}
	if err := ts.Assign(testActor, task.ID, testAssignee, causes, testConv); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	for _, state := range []work.TaskStatus{work.StatusReady, work.StatusRunning} {
		if err := ts.TransitionTask(testActor, task.ID, state, "start", nil, causes, testConv); err != nil {
			t.Fatalf("Transition %s: %v", state, err)
		}
	}
	if err := ts.AttachVerificationEvidence(testActor, task.ID, work.VerificationEvidence{TestCaseIDs: []string{"tc_replay"}, TestRunIDs: []string{"tr_replay"}}, "evidence", causes, testConv); err != nil {
		t.Fatalf("AttachVerificationEvidence: %v", err)
	}

	replayed := newTaskStore(t, s)
	projection, err := replayed.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask replayed: %v", err)
	}
	if projection.Status != work.StatusRunning {
		t.Fatalf("status = %q; want running", projection.Status)
	}
	if projection.Assignee != testAssignee {
		t.Fatalf("assignee = %s; want %s", projection.Assignee, testAssignee)
	}
	if projection.Linkage.FactoryOrderID != "fo_replay" {
		t.Fatalf("FactoryOrderID = %q", projection.Linkage.FactoryOrderID)
	}
	if !slices.Equal(projection.Verification.TestCaseIDs, []string{"tc_replay"}) {
		t.Fatalf("TestCaseIDs = %#v", projection.Verification.TestCaseIDs)
	}
}

func TestTaskStoreV39_LinkageRequiresCompatibleTier0IDs(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Bad linkage", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	err = ts.LinkTask(testActor, task.ID, work.TaskLinkage{
		CanonicalTaskID:        "task_without_prefix",
		FactoryOrderID:         "fo_valid",
		RequirementIDs:         []string{"req_valid"},
		AcceptanceCriterionIDs: []string{"ac_valid"},
	}, causes, testConv)
	if err == nil {
		t.Fatal("LinkTask accepted incompatible canonical Task ID")
	}
}

func TestTaskStoreV39_TaskIDCanRemainWorkEventIDWhenCanonicalRecordSeparate(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.Create(testActor, "Separate canonical task record", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.ID.IsZero() {
		t.Fatal("task event ID is zero")
	}
	if err := ts.LinkTask(testActor, task.ID, work.TaskLinkage{
		CanonicalTaskID:        "tsk_separate_record",
		FactoryOrderID:         "fo_separate",
		RequirementIDs:         []string{"req_separate"},
		AcceptanceCriterionIDs: []string{"ac_separate"},
	}, causes, testConv); err != nil {
		t.Fatalf("LinkTask: %v", err)
	}
	projection, err := ts.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask: %v", err)
	}
	if projection.Linkage.CanonicalTaskID != "tsk_separate_record" {
		t.Fatalf("CanonicalTaskID = %q", projection.Linkage.CanonicalTaskID)
	}
	if projection.ID.Value() == projection.Linkage.CanonicalTaskID {
		t.Fatal("work task event ID should remain distinct from canonical Task record ID")
	}
}
