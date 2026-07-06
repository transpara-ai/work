package work_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
	"github.com/transpara-ai/work"
)

func TestIssueScanDAGReplayCreatesOneCanonicalStageChainPerTarget(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	targets := []work.IssueScanTarget{
		{Repository: "transpara-ai/docs", IssueNumber: 172},
		{Repository: "transpara-ai/site", IssueNumber: 115},
	}

	for attempt := 0; attempt < 2; attempt++ {
		for _, target := range targets {
			result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
				RunID:  "2026-06-25-docs-172-site-115-dry-run",
				Target: target,
			}, causes, testConv)
			if err != nil {
				t.Fatalf("EnsureIssueScanDAG attempt %d target %s: %v", attempt, target.Ref(), err)
			}
			if len(result.Stages) != len(work.IssueScanStageIDs()) {
				t.Fatalf("stage count = %d; want %d", len(result.Stages), len(work.IssueScanStageIDs()))
			}
			if attempt == 0 {
				if result.CreatedTasks != 7 || result.CreatedDependencies != 6 {
					t.Fatalf("first replay created tasks=%d deps=%d; want 7/6", result.CreatedTasks, result.CreatedDependencies)
				}
				continue
			}
			if result.CreatedTasks != 0 || result.CreatedDependencies != 0 {
				t.Fatalf("second replay created tasks=%d deps=%d; want 0/0", result.CreatedTasks, result.CreatedDependencies)
			}
			for _, stage := range result.Stages {
				if stage.Created {
					t.Fatalf("stage %s was recreated on replay", stage.Stage)
				}
				if stage.DuplicateOf != stage.Task.ID {
					t.Fatalf("stage %s duplicate_of = %s; want existing task %s", stage.Stage, stage.DuplicateOf.Value(), stage.Task.ID.Value())
				}
			}
		}
	}

	tasks, err := ts.List(1000)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 14 {
		t.Fatalf("task count = %d; want 14", len(tasks))
	}
	seenCanonical := map[string]bool{}
	for _, task := range tasks {
		if task.Workspace != work.IssueScanWorkspace {
			t.Fatalf("workspace = %q; want %q", task.Workspace, work.IssueScanWorkspace)
		}
		if seenCanonical[task.CanonicalTaskID] {
			t.Fatalf("duplicate canonical task id %q", task.CanonicalTaskID)
		}
		seenCanonical[task.CanonicalTaskID] = true
	}
	depPage, err := s.ByType(work.EventTypeTaskDependencyAdded, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType dependencies: %v", err)
	}
	if len(depPage.Items()) != 12 {
		t.Fatalf("dependency edge count = %d; want 12", len(depPage.Items()))
	}
}

func TestIssueScanCanonicalIDsSeparatePunctuationVariantTargets(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	first, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/foo.bar", IssueNumber: 172},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG first target: %v", err)
	}
	second, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/foo-bar", IssueNumber: 172},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG second target: %v", err)
	}
	if first.CreatedTasks != 1 || second.CreatedTasks != 1 {
		t.Fatalf("created tasks first=%d second=%d; want one task for each distinct target", first.CreatedTasks, second.CreatedTasks)
	}
	if first.Stages[0].Task.ID == second.Stages[0].Task.ID {
		t.Fatalf("punctuation-variant targets shared task id %s", first.Stages[0].Task.ID.Value())
	}
	if first.Stages[0].Task.CanonicalTaskID == second.Stages[0].Task.CanonicalTaskID {
		t.Fatalf("punctuation-variant targets shared canonical task id %q", first.Stages[0].Task.CanonicalTaskID)
	}

	ref := first.Stages[0].Ref()
	ref.Target = work.IssueScanTarget{Repository: "transpara-ai/foo-bar", IssueNumber: 172}
	_, err = ts.BlockIssueScanStage(testActor, ref, work.IssueScanBlocker{
		Reason: work.IssueScanBlockerStaleTarget,
		Detail: "punctuation collision must not pass typed telemetry validation",
	}, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("BlockIssueScanStage collision ref = %v; want invalid transition", err)
	}
}

func TestIssueScanStageCertificationUnblocksNextStage(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	first := result.Stages[0]
	second := result.Stages[1]
	if blocked, err := ts.IsBlocked(second.Task.ID); err != nil || !blocked {
		t.Fatalf("stage 2 initial blocked = %v, %v; want true", blocked, err)
	}

	if status, err := ts.StartIssueScanStage(testActor, first.Ref(), "begin research", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage = %s, %v; want running", status, err)
	}
	if gate, err := ts.SatisfyIssueScanStageGate(testActor, first.Ref(), first.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil || !gate.Created || gate.Status != work.StatusCertified {
		t.Fatalf("SatisfyIssueScanStageGate = %+v, %v; want created certified", gate, err)
	}
	if blocked, err := ts.IsBlocked(second.Task.ID); err != nil || blocked {
		t.Fatalf("stage 2 blocked after stage 1 certified = %v, %v; want false", blocked, err)
	}
	if status, err := ts.StartIssueScanStage(testActor, second.Ref(), "begin debate", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage stage 2 = %s, %v; want running", status, err)
	}
	open, err := ts.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	for _, task := range open {
		if task.ID == first.Task.ID {
			t.Fatalf("certified issue-scan stage %s remained open", first.Stage)
		}
	}
	if gate, err := ts.SatisfyIssueScanStageGate(testActor, first.Ref(), first.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil || gate.Created || gate.Status != work.StatusCertified {
		t.Fatalf("repeat gate = %+v, %v; want no-op certified", gate, err)
	}
	gatePage, err := s.ByType(work.EventTypeIssueScanStageGateSatisfied, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType gate events: %v", err)
	}
	if len(gatePage.Items()) != 1 {
		t.Fatalf("gate event count = %d; want 1", len(gatePage.Items()))
	}
}

func TestIssueScanStageGateRejectsNonCanonicalGate(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	if status, err := ts.StartIssueScanStage(testActor, stage.Ref(), "begin research", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage = %s, %v; want running", status, err)
	}
	_, err = ts.SatisfyIssueScanStageGate(testActor, stage.Ref(), "wrong_gate", []string{"artifact:research-packet"}, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("SatisfyIssueScanStageGate wrong gate = %v; want invalid transition", err)
	}
	if status, err := ts.GetStatus(stage.Task.ID); err != nil || status != work.StatusRunning {
		t.Fatalf("status after wrong gate = %s, %v; want running", status, err)
	}
	gatePage, err := s.ByType(work.EventTypeIssueScanStageGateSatisfied, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType gate events: %v", err)
	}
	if len(gatePage.Items()) != 0 {
		t.Fatalf("gate event count = %d; want 0", len(gatePage.Items()))
	}
}

func TestIssueScanStageCertificationUnblocksNextStageWithCustomWorkspace(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:     "2026-06-25-docs-172-site-115-dry-run",
		Target:    work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
		Workspace: "custom.scan.workspace",
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
			work.IssueScanStageDebate,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	first := result.Stages[0]
	second := result.Stages[1]
	if first.Task.Workspace != "custom.scan.workspace" {
		t.Fatalf("workspace = %q; want custom.scan.workspace", first.Task.Workspace)
	}
	if status, err := ts.StartIssueScanStage(testActor, first.Ref(), "begin research", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage = %s, %v; want running", status, err)
	}
	if gate, err := ts.SatisfyIssueScanStageGate(testActor, first.Ref(), first.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil || !gate.Created || gate.Status != work.StatusCertified {
		t.Fatalf("SatisfyIssueScanStageGate = %+v, %v; want created certified", gate, err)
	}
	if blocked, err := ts.IsBlocked(second.Task.ID); err != nil || blocked {
		t.Fatalf("stage 2 blocked after stage 1 certified = %v, %v; want false", blocked, err)
	}
}

func TestIssueScanTypedEventsRejectNonIssueScanTasks(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	task, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:                  "Non issue-scan task",
		CanonicalTaskID:        "tsk_non_issue_scan_task",
		FactoryOrderID:         "fo_non_issue_scan_task",
		RequirementIDs:         []string{"req_non_issue_scan_task"},
		AcceptanceCriterionIDs: []string{"ac_non_issue_scan_task"},
		Cell:                   "implementation",
		RiskClass:              "low",
		ExpectedOutputs:        []string{"ordinary evidence"},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39: %v", err)
	}
	ref := work.IssueScanStageRef{
		TaskID: task.ID,
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
		Stage:  work.IssueScanStageResearch,
	}
	_, err = ts.BlockIssueScanStage(testActor, ref, work.IssueScanBlocker{
		Reason: work.IssueScanBlockerStaleTarget,
		Detail: "ordinary task must not receive issue-scan telemetry",
	}, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("BlockIssueScanStage non issue-scan task = %v; want invalid transition", err)
	}
	blockerPage, err := s.ByType(work.EventTypeIssueScanStageBlocked, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType blocker events: %v", err)
	}
	if len(blockerPage.Items()) != 0 {
		t.Fatalf("blocker event count = %d; want 0", len(blockerPage.Items()))
	}
}

func TestIssueScanTypedEventsRejectMismatchedStageRef(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	ref := result.Stages[0].Ref()
	ref.Target = work.IssueScanTarget{Repository: "transpara-ai/site", IssueNumber: 115}

	_, err = ts.BlockIssueScanStage(testActor, ref, work.IssueScanBlocker{
		Reason: work.IssueScanBlockerStaleTarget,
		Detail: "mismatched typed telemetry must not be recorded",
	}, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("BlockIssueScanStage mismatched ref = %v; want invalid transition", err)
	}
	_, err = ts.SatisfyIssueScanStageGate(testActor, ref, result.Stages[0].Gate, []string{"artifact:research-packet"}, causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("SatisfyIssueScanStageGate mismatched ref = %v; want invalid transition", err)
	}
	blockerPage, err := s.ByType(work.EventTypeIssueScanStageBlocked, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType blocker events: %v", err)
	}
	if len(blockerPage.Items()) != 0 {
		t.Fatalf("blocker event count = %d; want 0", len(blockerPage.Items()))
	}
	gatePage, err := s.ByType(work.EventTypeIssueScanStageGateSatisfied, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType gate events: %v", err)
	}
	if len(gatePage.Items()) != 0 {
		t.Fatalf("gate event count = %d; want 0", len(gatePage.Items()))
	}
}

func TestIssueScanGateCanCertifyVerifiedStage(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	if status, err := ts.StartIssueScanStage(testActor, stage.Ref(), "begin research", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage = %s, %v; want running", status, err)
	}
	if err := ts.TransitionTask(testActor, stage.Task.ID, work.StatusVerified, "external issue-scan evidence verified", []string{"artifact:verification"}, causes, testConv); err != nil {
		t.Fatalf("TransitionTask verified: %v", err)
	}
	if gate, err := ts.SatisfyIssueScanStageGate(testActor, stage.Ref(), stage.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil || !gate.Created || gate.Status != work.StatusCertified {
		t.Fatalf("SatisfyIssueScanStageGate verified stage = %+v, %v; want created certified", gate, err)
	}
}

func TestIssueScanBlockerParksStagesWithoutRepeatedEvents(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	docs, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG docs: %v", err)
	}
	site, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/site", IssueNumber: 115},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG site: %v", err)
	}

	docsImplement := docs.Stages[3]
	humanScope := work.IssueScanBlocker{
		Reason:       work.IssueScanBlockerNeedsHumanScope,
		Detail:       "docs#172 requires human approval before protected PR/merge action",
		EvidenceRefs: []string{"github:transpara-ai/docs#172"},
	}
	firstBlock, err := ts.BlockIssueScanStage(testActor, docsImplement.Ref(), humanScope, causes, testConv)
	if err != nil {
		t.Fatalf("BlockIssueScanStage docs: %v", err)
	}
	if !firstBlock.Created || firstBlock.Status != work.StatusPolicyBlocked {
		t.Fatalf("docs block = %+v; want created policy_blocked", firstBlock)
	}
	secondBlock, err := ts.BlockIssueScanStage(testActor, docsImplement.Ref(), humanScope, causes, testConv)
	if err != nil {
		t.Fatalf("BlockIssueScanStage docs repeat: %v", err)
	}
	if secondBlock.Created || secondBlock.Status != work.StatusPolicyBlocked {
		t.Fatalf("repeat docs block = %+v; want no-op policy_blocked", secondBlock)
	}

	siteResearch := site.Stages[0]
	staleTarget := work.IssueScanBlocker{
		Reason:       work.IssueScanBlockerStaleTarget,
		Detail:       "site#115 is closed or no longer matches the scan target head",
		EvidenceRefs: []string{"github:transpara-ai/site#115"},
	}
	siteBlock, err := ts.BlockIssueScanStage(testActor, siteResearch.Ref(), staleTarget, causes, testConv)
	if err != nil {
		t.Fatalf("BlockIssueScanStage site: %v", err)
	}
	if !siteBlock.Created || siteBlock.Status != work.StatusBlocked {
		t.Fatalf("site block = %+v; want created blocked", siteBlock)
	}
	projection, err := ts.ProjectTask(docsImplement.Task.ID)
	if err != nil {
		t.Fatalf("ProjectTask docs implement: %v", err)
	}
	if projection.Status != work.StatusPolicyBlocked || !projection.Blocked {
		t.Fatalf("docs projection status=%s blocked=%v; want policy_blocked blocked", projection.Status, projection.Blocked)
	}
	blockerPage, err := s.ByType(work.EventTypeIssueScanStageBlocked, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType blocker events: %v", err)
	}
	if len(blockerPage.Items()) != 2 {
		t.Fatalf("blocker event count = %d; want 2", len(blockerPage.Items()))
	}
}

func TestIssueScanBlockerDoesNotAppendWhenStageIsTerminal(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 172},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	if _, err := ts.StartIssueScanStage(testActor, stage.Ref(), "begin research", causes, testConv); err != nil {
		t.Fatalf("StartIssueScanStage: %v", err)
	}
	if _, err := ts.SatisfyIssueScanStageGate(testActor, stage.Ref(), stage.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil {
		t.Fatalf("SatisfyIssueScanStageGate: %v", err)
	}
	_, err = ts.BlockIssueScanStage(testActor, stage.Ref(), work.IssueScanBlocker{
		Reason: work.IssueScanBlockerStaleTarget,
		Detail: "target changed after certification",
	}, causes, testConv)
	if err == nil {
		t.Fatal("BlockIssueScanStage after certification succeeded; want invalid transition")
	}
	blockerPage, err := s.ByType(work.EventTypeIssueScanStageBlocked, 1000, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType blocker events: %v", err)
	}
	if len(blockerPage.Items()) != 0 {
		t.Fatalf("blocker event count = %d; want 0 after failed terminal block", len(blockerPage.Items()))
	}
}

func TestIssueScanBlockedStageCannotRestartWithoutRepair(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-06-25-docs-172-site-115-dry-run",
		Target: work.IssueScanTarget{Repository: "transpara-ai/site", IssueNumber: 115},
		Stages: []work.IssueScanStageID{work.IssueScanStageResearch},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	blocker := work.IssueScanBlocker{
		Reason:       work.IssueScanBlockerStaleTarget,
		Detail:       "site#115 is closed or no longer matches the scan target head",
		EvidenceRefs: []string{"github:transpara-ai/site#115"},
	}
	if block, err := ts.BlockIssueScanStage(testActor, stage.Ref(), blocker, causes, testConv); err != nil || !block.Created || block.Status != work.StatusBlocked {
		t.Fatalf("BlockIssueScanStage = %+v, %v; want created blocked", block, err)
	}

	status, err := ts.StartIssueScanStage(testActor, stage.Ref(), "retry stale target", causes, testConv)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("StartIssueScanStage blocked stage = %s, %v; want invalid transition", status, err)
	}
	if status, err := ts.GetStatus(stage.Task.ID); err != nil || status != work.StatusBlocked {
		t.Fatalf("status after restart attempt = %s, %v; want blocked", status, err)
	}
}

func TestIssueScanMarkerWorkRefProjectsCanonicalRefs(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	sourceIssueBody := `{"source_issue_records":[{"repo":"transpara-ai/docs","number":256,"title":"Factory-order acquisition marker","source_refs":["  "]},{"repo":"transpara-ai/docs","number":257,"title":"Duplicate explicit source ref must not fall back","source_refs":["transpara-ai/docs#256"]}]}`
	if err := ts.AddArtifact(testActor, stage.Task.ID, work.FactoryOrderSourceIssuesArtifactLabel, "application/json", sourceIssueBody, causes, testConv); err != nil {
		t.Fatalf("AddArtifact source issues: %v", err)
	}
	if err := ts.AttachVerificationEvidence(testActor, stage.Task.ID, work.VerificationEvidence{
		TestCaseIDs:   []string{"tc_issue_scan_marker_research"},
		TestRunIDs:    []string{"tr_issue_scan_marker_research"},
		GateResultIDs: []string{"gate_issue_scan_marker_research"},
		WaiverIDs:     []string{"waiver_issue_scan_marker_verification"},
	}, "issue-scan marker research evidence", causes, testConv); err != nil {
		t.Fatalf("AttachVerificationEvidence: %v", err)
	}
	if err := ts.AttachFailureRepairReferences(testActor, stage.Task.ID, work.FailureRepairReferences{
		FailureIDs:       []string{"fail_issue_scan_marker_research"},
		RepairAttemptIDs: []string{"rep_issue_scan_marker_research"},
		WaiverIDs:        []string{"waiver_issue_scan_marker_repair"},
	}, "issue-scan marker repair evidence", causes, testConv); err != nil {
		t.Fatalf("AttachFailureRepairReferences: %v", err)
	}
	if status, err := ts.StartIssueScanStage(testActor, stage.Ref(), "begin research", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage = %s, %v; want running", status, err)
	}
	if gate, err := ts.SatisfyIssueScanStageGate(testActor, stage.Ref(), stage.Gate, []string{"artifact:research-packet"}, causes, testConv); err != nil || !gate.Created || gate.Status != work.StatusCertified {
		t.Fatalf("SatisfyIssueScanStageGate = %+v, %v; want created certified", gate, err)
	}

	packet, err := ts.ProjectIssueScanMarkerWorkRef(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRef: %v", err)
	}
	if packet.SchemaVersion != work.IssueScanMarkerSchemaVersion || packet.ProjectionKind != work.IssueScanMarkerProjectionKind || packet.CanonicalSource != "work" || !packet.ProjectionOnly {
		t.Fatalf("packet identity = %+v", packet)
	}
	if packet.RunID != "2026-07-06-docs-256" || packet.Target.Ref() != "transpara-ai/docs#256" {
		t.Fatalf("target = %s %s; want run and docs#256", packet.RunID, packet.Target.Ref())
	}
	if packet.Stage != work.IssueScanStageResearch || packet.StageNumber != 1 || packet.Gate != stage.Gate {
		t.Fatalf("stage fields = %+v; want research stage 1 gate %s", packet, stage.Gate)
	}
	if packet.TaskID != stage.Task.ID.Value() || packet.CanonicalTaskID != stage.CanonicalTaskID || packet.FactoryOrderID != stage.FactoryOrderID {
		t.Fatalf("task linkage = %+v; want task/canonical/factory order from stage", packet)
	}
	if packet.LifecycleState != work.StatusCertified || packet.Blocked || packet.LatestGate == nil || packet.LatestGate.Gate != stage.Gate {
		t.Fatalf("lifecycle/gate = status %s blocked %v gate %+v; want certified unblocked latest gate", packet.LifecycleState, packet.Blocked, packet.LatestGate)
	}
	if !containsString(packet.SourceIssueRefs, "transpara-ai/docs#256") {
		t.Fatalf("source refs = %#v; want docs#256", packet.SourceIssueRefs)
	}
	if containsString(packet.SourceIssueRefs, "transpara-ai/docs#257") {
		t.Fatalf("source refs = %#v; duplicate explicit ref should not fall back to docs#257", packet.SourceIssueRefs)
	}
	if !containsString(packet.VerificationRefs.TestRunIDs, "tr_issue_scan_marker_research") || !containsString(packet.VerificationRefs.GateResultIDs, "gate_issue_scan_marker_research") {
		t.Fatalf("verification refs = %+v; want test run and gate result refs", packet.VerificationRefs)
	}
	if !containsString(packet.VerificationRefs.WaiverIDs, "waiver_issue_scan_marker_verification") {
		t.Fatalf("verification refs = %+v; want waiver ref", packet.VerificationRefs)
	}
	if !containsString(packet.FailureRepairRefs.FailureIDs, "fail_issue_scan_marker_research") || !containsString(packet.FailureRepairRefs.RepairAttemptIDs, "rep_issue_scan_marker_research") {
		t.Fatalf("failure/repair refs = %+v; want failure and repair refs", packet.FailureRepairRefs)
	}
	if !containsString(packet.FailureRepairRefs.WaiverIDs, "waiver_issue_scan_marker_repair") {
		t.Fatalf("failure/repair refs = %+v; want waiver ref", packet.FailureRepairRefs)
	}
	if packet.LastTransitionEvent == "" {
		t.Fatalf("last transition event is empty; want lifecycle transition ref")
	}
	for _, want := range []string{
		"github_comments_are_not_work_lifecycle_truth",
		"no_live_github_mutation_authority",
		"no_merge_authority",
		"no_issue_closure",
	} {
		if !containsString(packet.AuthorityExclusions, want) {
			t.Fatalf("authority exclusions = %#v; want %s", packet.AuthorityExclusions, want)
		}
	}
	if !containsString(packet.MissingGates, work.GateDefinitionOfDone) {
		t.Fatalf("missing gates = %#v; want readiness gate projection", packet.MissingGates)
	}

	body, err := ts.ProjectIssueScanMarkerWorkRefJSON(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRefJSON: %v", err)
	}
	for _, want := range []string{
		`"target"`,
		`"repository"`,
		`"issue_number"`,
		`"latest_gate"`,
		`"evidence_refs"`,
		`"test_run_ids"`,
		`"gate_result_ids"`,
		`"waiver_ids"`,
		`"failure_ids"`,
		`"repair_attempt_ids"`,
		`"last_transition_event"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("marker JSON missing literal key %s:\n%s", want, body)
		}
	}
	var decoded work.IssueScanMarkerWorkRef
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("marker JSON not decodable: %v\n%s", err, body)
	}
	if decoded.FactoryOrderID != packet.FactoryOrderID || decoded.LatestGate == nil || decoded.LatestGate.Gate != stage.Gate {
		t.Fatalf("decoded packet = %+v; want factory order and latest gate", decoded)
	}
}

func TestIssueScanMarkerWorkRefJSONMatchesGoldenShape(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	body, err := ts.ProjectIssueScanMarkerWorkRefJSON(result.Stages[0].Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRefJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("generated marker JSON not decodable: %v\n%s", err, body)
	}
	got["task_id"] = "<task_id>"

	golden, err := os.ReadFile("testdata/issue_scan_marker_work_ref.golden.json")
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	var want map[string]any
	if err := json.Unmarshal(golden, &want); err != nil {
		t.Fatalf("golden marker JSON not decodable: %v\n%s", err, golden)
	}
	if !reflect.DeepEqual(got, want) {
		rendered, _ := json.MarshalIndent(got, "", "  ")
		t.Fatalf("marker JSON shape drifted.\nGot:\n%s\nWant:\n%s", rendered, golden)
	}
}

func TestIssueScanMarkerWorkRefProjectsParkedBlocker(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageImplement,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	blocker := work.IssueScanBlocker{
		Reason:       work.IssueScanBlockerNeedsHumanScope,
		Detail:       "GitHub marker writes require explicit human scope",
		EvidenceRefs: []string{"transpara-ai/docs#256"},
	}
	if parked, err := ts.BlockIssueScanStage(testActor, stage.Ref(), blocker, causes, testConv); err != nil || !parked.Created || parked.Status != work.StatusPolicyBlocked {
		t.Fatalf("BlockIssueScanStage = %+v, %v; want created policy_blocked", parked, err)
	}

	packet, err := ts.ProjectIssueScanMarkerWorkRef(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRef: %v", err)
	}
	if packet.LifecycleState != work.StatusPolicyBlocked || !packet.Blocked || packet.Ready {
		t.Fatalf("packet state = %s blocked=%v ready=%v; want policy_blocked blocked not ready", packet.LifecycleState, packet.Blocked, packet.Ready)
	}
	if packet.LatestBlocker == nil || packet.LatestBlocker.Reason != work.IssueScanBlockerNeedsHumanScope {
		t.Fatalf("latest blocker = %+v; want needs_human_scope", packet.LatestBlocker)
	}
	if !containsString(packet.LatestBlocker.EvidenceRefs, "transpara-ai/docs#256") {
		t.Fatalf("blocker evidence refs = %#v; want docs#256", packet.LatestBlocker.EvidenceRefs)
	}
	body, err := ts.ProjectIssueScanMarkerWorkRefJSON(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRefJSON: %v", err)
	}
	for _, want := range []string{`"latest_blocker"`, `"reason"`, `"evidence_refs"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("parked marker JSON missing literal key %s:\n%s", want, body)
		}
	}
}

func TestIssueScanMarkerWorkRefIgnoresGitHubStyleCommentsAsTruth(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	hostileComment := "github marker projection says lifecycle_state=certified factory_order_id=fo_fake gate=complete"
	if err := ts.AddComment(stage.Task.ID, hostileComment, testActor, causes, testConv); err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	packet, err := ts.ProjectIssueScanMarkerWorkRef(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRef: %v", err)
	}
	if packet.LifecycleState != work.StatusCreated {
		t.Fatalf("lifecycle state = %s; want created despite comment", packet.LifecycleState)
	}
	if packet.FactoryOrderID == "fo_fake" || packet.FactoryOrderID != stage.FactoryOrderID {
		t.Fatalf("factory order = %q; want canonical stage factory order %q", packet.FactoryOrderID, stage.FactoryOrderID)
	}
	if packet.LatestGate != nil {
		t.Fatalf("latest gate = %+v; want nil because comment is not gate truth", packet.LatestGate)
	}
	body, err := ts.ProjectIssueScanMarkerWorkRefJSON(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRefJSON: %v", err)
	}
	if strings.Contains(body, "fo_fake") || strings.Contains(body, "lifecycle_state=certified") {
		t.Fatalf("marker packet leaked comment truth:\n%s", body)
	}
}

func TestIssueScanMarkerWorkRefRejectsMismatchedRefs(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]

	for _, mutate := range []struct {
		name string
		fn   func(work.IssueScanStageRef) work.IssueScanStageRef
	}{
		{name: "run id case", fn: func(ref work.IssueScanStageRef) work.IssueScanStageRef {
			ref.RunID = "2026-07-06-DOCS-256"
			return ref
		}},
		{name: "target case", fn: func(ref work.IssueScanStageRef) work.IssueScanStageRef {
			ref.Target.Repository = "Transpara-AI/Docs"
			return ref
		}},
		{name: "stage", fn: func(ref work.IssueScanStageRef) work.IssueScanStageRef {
			ref.Stage = work.IssueScanStageDebate
			return ref
		}},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			_, err := ts.ProjectIssueScanMarkerWorkRef(mutate.fn(stage.Ref()))
			if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
				t.Fatalf("ProjectIssueScanMarkerWorkRef mismatched %s = %v; want invalid transition", mutate.name, err)
			}
		})
	}

	ordinary, err := ts.CreateV39(testActor, work.TaskCreateOptions{
		Title:                  "Ordinary task",
		CanonicalTaskID:        "tsk_ordinary_marker_ref",
		FactoryOrderID:         "fo_ordinary_marker_ref",
		RequirementIDs:         []string{"req_ordinary_marker_ref"},
		AcceptanceCriterionIDs: []string{"ac_ordinary_marker_ref"},
		Cell:                   "implementation",
		RiskClass:              "low",
	}, causes, testConv)
	if err != nil {
		t.Fatalf("CreateV39 ordinary task: %v", err)
	}
	ref := stage.Ref()
	ref.TaskID = ordinary.ID
	_, err = ts.ProjectIssueScanMarkerWorkRef(ref)
	if !errors.Is(err, work.ErrInvalidLifecycleTransition) {
		t.Fatalf("ProjectIssueScanMarkerWorkRef ordinary task = %v; want invalid transition", err)
	}
}

func TestIssueScanMarkerWorkRefProjectsSupersededBy(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	if err := ts.SupersedeTask(testActor, stage.Task.ID, "tsk_replacement_marker_ref", "replaced by newer source issue scan", []string{"transpara-ai/docs#256"}, causes, testConv); err != nil {
		t.Fatalf("SupersedeTask: %v", err)
	}
	packet, err := ts.ProjectIssueScanMarkerWorkRef(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRef: %v", err)
	}
	if packet.LifecycleState != work.StatusSuperseded || packet.SupersededBy != "tsk_replacement_marker_ref" || packet.LastTransitionEvent == "" {
		t.Fatalf("superseded packet = %+v; want superseded replacement and transition event", packet)
	}
}

func TestIssueScanMarkerWorkRefKeepsResolvedBlockerAsHistory(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	result, err := ts.EnsureIssueScanDAG(testActor, work.IssueScanDAGOptions{
		RunID:  "2026-07-06-docs-256",
		Target: work.IssueScanTarget{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stages: []work.IssueScanStageID{
			work.IssueScanStageResearch,
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("EnsureIssueScanDAG: %v", err)
	}
	stage := result.Stages[0]
	blocker := work.IssueScanBlocker{
		Reason:       work.IssueScanBlockerStaleTarget,
		Detail:       "source issue target was stale during acquisition",
		EvidenceRefs: []string{"transpara-ai/docs#256"},
	}
	if parked, err := ts.BlockIssueScanStage(testActor, stage.Ref(), blocker, causes, testConv); err != nil || !parked.Created || parked.Status != work.StatusBlocked {
		t.Fatalf("BlockIssueScanStage = %+v, %v; want created blocked", parked, err)
	}
	if err := ts.TransitionTask(testActor, stage.Task.ID, work.StatusReady, "stale target repaired", []string{"transpara-ai/docs#256"}, causes, testConv); err != nil {
		t.Fatalf("TransitionTask ready: %v", err)
	}
	if status, err := ts.StartIssueScanStage(testActor, stage.Ref(), "restart after stale target repair", causes, testConv); err != nil || status != work.StatusRunning {
		t.Fatalf("StartIssueScanStage after repair = %s, %v; want running", status, err)
	}

	packet, err := ts.ProjectIssueScanMarkerWorkRef(stage.Ref())
	if err != nil {
		t.Fatalf("ProjectIssueScanMarkerWorkRef: %v", err)
	}
	if packet.LifecycleState != work.StatusRunning || packet.Blocked || packet.LatestBlocker == nil {
		t.Fatalf("resolved blocker packet = %+v; want running, not blocked, with historical latest blocker", packet)
	}
	if packet.LatestBlocker.Reason != work.IssueScanBlockerStaleTarget {
		t.Fatalf("latest blocker = %+v; want stale target history", packet.LatestBlocker)
	}
}
