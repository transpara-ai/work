package work_test

import (
	"errors"
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
