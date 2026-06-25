package work

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	// IssueScanWorkspace is the stable Work workspace for Civilization
	// autonomous issue-scan stage tasks.
	IssueScanWorkspace = "civilization.issue_scan"

	defaultIssueScanCell      = "cell_civilization_issue_scan"
	defaultIssueScanRiskClass = "high"
)

// IssueScanStageID is the canonical stage key for the autonomous issue-scan
// pipeline. The values are part of the Work/EventGraph contract.
type IssueScanStageID string

const (
	IssueScanStageResearch            IssueScanStageID = "research_issue_and_repo_context"
	IssueScanStageDebate              IssueScanStageID = "debate_with_correct_civic_roles"
	IssueScanStageSelectApproach      IssueScanStageID = "select_and_design_approach"
	IssueScanStageImplement           IssueScanStageID = "implement_on_branch"
	IssueScanStageAdversarialReview   IssueScanStageID = "run_adversarial_review"
	IssueScanStageDriveBlockersToZero IssueScanStageID = "drive_blockers_to_zero"
	IssueScanStageSurfaceHumanReadyPR IssueScanStageID = "surface_ready_for_human_result_pr"
)

type issueScanStageDefinition struct {
	Number          int
	Title           string
	Gate            string
	ExpectedOutputs []string
}

var issueScanStageDefinitions = map[IssueScanStageID]issueScanStageDefinition{
	IssueScanStageResearch: {
		Number: 1,
		Title:  "Research issue and repo context",
		Gate:   "research_packet_posted",
		ExpectedOutputs: []string{
			"bounded issue and repository context",
			"current target state",
			"protected-action boundary",
		},
	},
	IssueScanStageDebate: {
		Number: 2,
		Title:  "Debate with correct civic roles",
		Gate:   "role_debate_complete",
		ExpectedOutputs: []string{
			"role debate summary",
			"risk and authority assessment",
		},
	},
	IssueScanStageSelectApproach: {
		Number: 3,
		Title:  "Select and design approach",
		Gate:   "implementation_plan_selected",
		ExpectedOutputs: []string{
			"selected approach",
			"acceptance test plan",
		},
	},
	IssueScanStageImplement: {
		Number: 4,
		Title:  "Implement on branch",
		Gate:   "branch_patch_ready",
		ExpectedOutputs: []string{
			"scoped branch changes",
			"local validation evidence",
		},
	},
	IssueScanStageAdversarialReview: {
		Number: 5,
		Title:  "Run adversarial review",
		Gate:   "adversarial_review_result_recorded",
		ExpectedOutputs: []string{
			"exact-head adversarial review artifact",
			"finding disposition",
		},
	},
	IssueScanStageDriveBlockersToZero: {
		Number: 6,
		Title:  "Drive blockers to zero",
		Gate:   "blockers_zero_or_human_parked",
		ExpectedOutputs: []string{
			"resolved blocker list",
			"revalidation evidence",
		},
	},
	IssueScanStageSurfaceHumanReadyPR: {
		Number: 7,
		Title:  "Surface ready-for-Human result PR",
		Gate:   "human_action_state_clear",
		ExpectedOutputs: []string{
			"draft PR URL",
			"human action summary",
			"merge/re-enable recommendation boundary",
		},
	},
}

// IssueScanStageIDs returns the canonical stage order for issue-scan DAGs.
func IssueScanStageIDs() []IssueScanStageID {
	return []IssueScanStageID{
		IssueScanStageResearch,
		IssueScanStageDebate,
		IssueScanStageSelectApproach,
		IssueScanStageImplement,
		IssueScanStageAdversarialReview,
		IssueScanStageDriveBlockersToZero,
		IssueScanStageSurfaceHumanReadyPR,
	}
}

// IssueScanTarget identifies the GitHub issue being processed.
type IssueScanTarget struct {
	Repository  string
	IssueNumber int
}

// Ref returns the human-readable repository issue reference.
func (t IssueScanTarget) Ref() string {
	return fmt.Sprintf("%s#%d", strings.TrimSpace(t.Repository), t.IssueNumber)
}

// IssueScanStageOptions configures one deterministic issue-scan stage task.
type IssueScanStageOptions struct {
	RunID                  string
	Target                 IssueScanTarget
	Stage                  IssueScanStageID
	Title                  string
	Description            string
	Workspace              string
	Priority               TaskPriority
	CanonicalTaskID        string
	FactoryOrderID         string
	RequirementIDs         []string
	AcceptanceCriterionIDs []string
	Cell                   string
	RiskClass              string
	ExpectedOutputs        []string
}

// IssueScanDAGOptions configures the canonical stage DAG for one target issue.
type IssueScanDAGOptions struct {
	RunID     string
	Target    IssueScanTarget
	Stages    []IssueScanStageID
	Workspace string
	Priority  TaskPriority
	Cell      string
	RiskClass string
}

// IssueScanStageRef is the stable reference callers pass when recording typed
// issue-scan stage state.
type IssueScanStageRef struct {
	TaskID types.EventID
	RunID  string
	Target IssueScanTarget
	Stage  IssueScanStageID
}

// IssueScanStageRecord is returned by idempotent stage creation.
type IssueScanStageRecord struct {
	IssueScanStageRef
	Task            Task
	StageNumber     int
	Gate            string
	Created         bool
	DuplicateOf     types.EventID
	CanonicalTaskID string
	FactoryOrderID  string
}

// IssueScanDAGResult is returned by EnsureIssueScanDAG.
type IssueScanDAGResult struct {
	Stages              []IssueScanStageRecord
	CreatedTasks        int
	CreatedDependencies int
}

// IssueScanBlockerReason is the typed reason a scan stage parked.
type IssueScanBlockerReason string

const (
	IssueScanBlockerNeedsHumanScope     IssueScanBlockerReason = "needs_human_scope"
	IssueScanBlockerProtectedAction     IssueScanBlockerReason = "protected_action"
	IssueScanBlockerStaleTarget         IssueScanBlockerReason = "stale_target"
	IssueScanBlockerDuplicateChain      IssueScanBlockerReason = "duplicate_chain"
	IssueScanBlockerMissingGateEvidence IssueScanBlockerReason = "missing_gate_evidence"
)

// IssueScanBlocker carries the structured blocker state for a parked stage.
type IssueScanBlocker struct {
	Reason       IssueScanBlockerReason
	Detail       string
	EvidenceRefs []string
}

// IssueScanStageBlockResult reports whether a new blocker event was appended.
type IssueScanStageBlockResult struct {
	Created bool
	Status  TaskStatus
}

// IssueScanStageGateResult reports whether a new gate-satisfied event was appended.
type IssueScanStageGateResult struct {
	Created bool
	Status  TaskStatus
}

func (r IssueScanStageRecord) Ref() IssueScanStageRef {
	return IssueScanStageRef{
		TaskID: r.Task.ID,
		RunID:  r.RunID,
		Target: r.Target,
		Stage:  r.Stage,
	}
}

// FindTaskByCanonicalTaskID returns the oldest task with the supplied canonical
// v3.9 task ID. If bad callers appended duplicates directly, the oldest event
// remains the canonical Work task for deterministic issue-scan replay.
func (ts *TaskStore) FindTaskByCanonicalTaskID(canonicalTaskID string) (Task, bool, error) {
	canonicalTaskID = strings.TrimSpace(canonicalTaskID)
	if canonicalTaskID == "" {
		return Task{}, false, fmt.Errorf("canonical_task_id is required")
	}
	var found Task
	hasFound := false
	after := types.None[types.Cursor]()
	for {
		page, err := ts.store.ByType(EventTypeTaskCreated, 1000, after)
		if err != nil {
			return Task{}, false, fmt.Errorf("find canonical task: %w", err)
		}
		for _, ev := range page.Items() {
			c, ok := ev.Content().(TaskCreatedContent)
			if !ok || strings.TrimSpace(c.CanonicalTaskID) != canonicalTaskID {
				continue
			}
			found = taskFromCreatedContent(ev.ID(), c)
			hasFound = true
		}
		if !page.HasMore() {
			return found, hasFound, nil
		}
		after = page.Cursor()
	}
}

// EnsureIssueScanStage creates or returns the deterministic task for one
// issue-scan stage. Repeated calls for the same run, target, and stage do not
// append duplicate task events.
func (ts *TaskStore) EnsureIssueScanStage(
	source types.ActorID,
	opts IssueScanStageOptions,
	causes []types.EventID,
	convID types.ConversationID,
) (IssueScanStageRecord, error) {
	normalized, def, err := normalizeIssueScanStageOptions(opts)
	if err != nil {
		return IssueScanStageRecord{}, err
	}
	if existing, ok, err := ts.FindTaskByCanonicalTaskID(normalized.CanonicalTaskID); err != nil {
		return IssueScanStageRecord{}, err
	} else if ok {
		return issueScanStageRecord(existing, normalized, def, false, existing.ID), nil
	}
	task, err := ts.CreateV39(source, TaskCreateOptions{
		Title:                  normalized.Title,
		Description:            normalized.Description,
		Workspace:              normalized.Workspace,
		Priority:               normalized.Priority,
		CanonicalTaskID:        normalized.CanonicalTaskID,
		FactoryOrderID:         normalized.FactoryOrderID,
		RequirementIDs:         normalized.RequirementIDs,
		AcceptanceCriterionIDs: normalized.AcceptanceCriterionIDs,
		Cell:                   normalized.Cell,
		RiskClass:              normalized.RiskClass,
		ExpectedOutputs:        normalized.ExpectedOutputs,
	}, causes, convID)
	if err != nil {
		return IssueScanStageRecord{}, err
	}
	return issueScanStageRecord(task, normalized, def, true, types.EventID{}), nil
}

// EnsureIssueScanDAG creates the canonical stage tasks and linear dependencies
// for one target issue. Replaying the same run is idempotent for both task nodes
// and dependency edges.
func (ts *TaskStore) EnsureIssueScanDAG(
	source types.ActorID,
	opts IssueScanDAGOptions,
	causes []types.EventID,
	convID types.ConversationID,
) (IssueScanDAGResult, error) {
	stages := opts.Stages
	if len(stages) == 0 {
		stages = IssueScanStageIDs()
	}
	result := IssueScanDAGResult{Stages: make([]IssueScanStageRecord, 0, len(stages))}
	var previous IssueScanStageRecord
	for i, stage := range stages {
		record, err := ts.EnsureIssueScanStage(source, IssueScanStageOptions{
			RunID:     opts.RunID,
			Target:    opts.Target,
			Stage:     stage,
			Workspace: opts.Workspace,
			Priority:  opts.Priority,
			Cell:      opts.Cell,
			RiskClass: opts.RiskClass,
		}, causes, convID)
		if err != nil {
			return result, err
		}
		if record.Created {
			result.CreatedTasks++
		}
		if i > 0 {
			created, err := ts.EnsureDependency(source, record.Task.ID, previous.Task.ID, causes, convID)
			if err != nil {
				return result, err
			}
			if created {
				result.CreatedDependencies++
			}
		}
		result.Stages = append(result.Stages, record)
		previous = record
	}
	return result, nil
}

// EnsureDependency records taskID -> dependsOnID only if that edge is absent.
func (ts *TaskStore) EnsureDependency(
	source types.ActorID,
	taskID, dependsOnID types.EventID,
	causes []types.EventID,
	convID types.ConversationID,
) (bool, error) {
	deps, err := ts.GetDependencies(taskID)
	if err != nil {
		return false, err
	}
	for _, dep := range deps {
		if dep == dependsOnID {
			return false, nil
		}
	}
	if err := ts.AddDependency(source, taskID, dependsOnID, causes, convID); err != nil {
		return false, err
	}
	return true, nil
}

// StartIssueScanStage moves an unblocked stage deterministically to running.
func (ts *TaskStore) StartIssueScanStage(
	source types.ActorID,
	ref IssueScanStageRef,
	reason string,
	causes []types.EventID,
	convID types.ConversationID,
) (TaskStatus, error) {
	if err := validateIssueScanStageRef(ref); err != nil {
		return "", err
	}
	blocked, err := ts.IsBlocked(ref.TaskID)
	if err != nil {
		return "", err
	}
	if blocked {
		return "", fmt.Errorf("%w: issue-scan stage %s is blocked by an incomplete predecessor", ErrInvalidLifecycleTransition, ref.TaskID.Value())
	}
	if strings.TrimSpace(reason) == "" {
		reason = "issue-scan stage started"
	}
	return ts.transitionIssueScanStageTo(source, ref.TaskID, StatusRunning, reason, nil, causes, convID)
}

// BlockIssueScanStage records a typed blocker and parks the task in blocked or
// policy_blocked. Repeating the same blocker while already parked is a no-op.
func (ts *TaskStore) BlockIssueScanStage(
	source types.ActorID,
	ref IssueScanStageRef,
	blocker IssueScanBlocker,
	causes []types.EventID,
	convID types.ConversationID,
) (IssueScanStageBlockResult, error) {
	if err := validateIssueScanStageRef(ref); err != nil {
		return IssueScanStageBlockResult{}, err
	}
	if err := validateIssueScanBlocker(blocker); err != nil {
		return IssueScanStageBlockResult{}, err
	}
	targetStatus := blocker.Reason.taskStatus()
	current, err := ts.GetStatus(ref.TaskID)
	if err != nil {
		return IssueScanStageBlockResult{}, err
	}
	if latest, ok, err := ts.latestIssueScanBlocker(ref.TaskID); err != nil {
		return IssueScanStageBlockResult{}, err
	} else if ok && current == targetStatus && latest.same(blocker) {
		return IssueScanStageBlockResult{Created: false, Status: current}, nil
	}
	content := IssueScanStageBlockedContent{
		TaskID:            ref.TaskID,
		RunID:             strings.TrimSpace(ref.RunID),
		TargetRepo:        strings.TrimSpace(ref.Target.Repository),
		TargetIssueNumber: ref.Target.IssueNumber,
		StageID:           ref.Stage,
		BlockerReason:     blocker.Reason,
		Detail:            strings.TrimSpace(blocker.Detail),
		EvidenceRefs:      cloneStrings(blocker.EvidenceRefs),
		BlockedBy:         source,
	}
	ev, err := ts.factory.Create(EventTypeIssueScanStageBlocked, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return IssueScanStageBlockResult{}, fmt.Errorf("create issue-scan blocker event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return IssueScanStageBlockResult{}, fmt.Errorf("append issue-scan blocker event: %w", err)
	}
	status, err := ts.transitionIssueScanStageTo(source, ref.TaskID, targetStatus, "issue-scan blocked: "+string(blocker.Reason), blocker.EvidenceRefs, causes, convID)
	if err != nil {
		return IssueScanStageBlockResult{}, err
	}
	return IssueScanStageBlockResult{Created: true, Status: status}, nil
}

// SatisfyIssueScanStageGate records a typed gate completion and certifies the
// issue-scan stage. Repeating the same gate after certification is a no-op.
func (ts *TaskStore) SatisfyIssueScanStageGate(
	source types.ActorID,
	ref IssueScanStageRef,
	gate string,
	evidenceRefs []string,
	causes []types.EventID,
	convID types.ConversationID,
) (IssueScanStageGateResult, error) {
	if err := validateIssueScanStageRef(ref); err != nil {
		return IssueScanStageGateResult{}, err
	}
	gate = strings.TrimSpace(gate)
	if gate == "" {
		return IssueScanStageGateResult{}, fmt.Errorf("issue-scan gate is required")
	}
	if len(cloneStrings(evidenceRefs)) == 0 {
		return IssueScanStageGateResult{}, fmt.Errorf("at least one issue-scan gate evidence reference is required")
	}
	current, err := ts.GetStatus(ref.TaskID)
	if err != nil {
		return IssueScanStageGateResult{}, err
	}
	if latest, ok, err := ts.latestIssueScanGate(ref.TaskID); err != nil {
		return IssueScanStageGateResult{}, err
	} else if ok && current == StatusCertified && latest.Gate == gate {
		return IssueScanStageGateResult{Created: false, Status: current}, nil
	}
	switch current {
	case StatusRunning, StatusVerified:
	case StatusCertified:
		return IssueScanStageGateResult{Created: false, Status: current}, nil
	default:
		return IssueScanStageGateResult{}, fmt.Errorf("%w: issue-scan gate can only be satisfied from running or verified, got %s", ErrInvalidLifecycleTransition, current)
	}
	content := IssueScanStageGateSatisfiedContent{
		TaskID:            ref.TaskID,
		RunID:             strings.TrimSpace(ref.RunID),
		TargetRepo:        strings.TrimSpace(ref.Target.Repository),
		TargetIssueNumber: ref.Target.IssueNumber,
		StageID:           ref.Stage,
		Gate:              gate,
		EvidenceRefs:      cloneStrings(evidenceRefs),
		SatisfiedBy:       source,
	}
	ev, err := ts.factory.Create(EventTypeIssueScanStageGateSatisfied, source, content, causes, convID, ts.store, ts.signer)
	if err != nil {
		return IssueScanStageGateResult{}, fmt.Errorf("create issue-scan gate event: %w", err)
	}
	if _, err := ts.store.Append(ev); err != nil {
		return IssueScanStageGateResult{}, fmt.Errorf("append issue-scan gate event: %w", err)
	}
	status, err := ts.transitionIssueScanStageTo(source, ref.TaskID, StatusCertified, "issue-scan gate satisfied: "+gate, evidenceRefs, causes, convID)
	if err != nil {
		return IssueScanStageGateResult{}, err
	}
	return IssueScanStageGateResult{Created: true, Status: status}, nil
}

func normalizeIssueScanStageOptions(opts IssueScanStageOptions) (IssueScanStageOptions, issueScanStageDefinition, error) {
	opts.RunID = strings.TrimSpace(opts.RunID)
	if opts.RunID == "" {
		return opts, issueScanStageDefinition{}, fmt.Errorf("issue-scan run_id is required")
	}
	opts.Target.Repository = strings.TrimSpace(opts.Target.Repository)
	if opts.Target.Repository == "" {
		return opts, issueScanStageDefinition{}, fmt.Errorf("issue-scan target repository is required")
	}
	if opts.Target.IssueNumber <= 0 {
		return opts, issueScanStageDefinition{}, fmt.Errorf("issue-scan target issue number must be positive")
	}
	def, ok := issueScanStageDefinitions[opts.Stage]
	if !ok {
		return opts, issueScanStageDefinition{}, fmt.Errorf("unknown issue-scan stage %q", opts.Stage)
	}
	base := issueScanBaseID(opts.RunID, opts.Target, opts.Stage)
	if opts.CanonicalTaskID == "" {
		opts.CanonicalTaskID = "tsk_" + base
	}
	if opts.FactoryOrderID == "" {
		opts.FactoryOrderID = "fo_issue_scan_" + issueScanIDPart(opts.RunID)
	}
	if len(opts.RequirementIDs) == 0 {
		opts.RequirementIDs = []string{"req_" + base}
	}
	if len(opts.AcceptanceCriterionIDs) == 0 {
		opts.AcceptanceCriterionIDs = []string{"ac_" + base}
	}
	if strings.TrimSpace(opts.Workspace) == "" {
		opts.Workspace = IssueScanWorkspace
	}
	if opts.Priority == "" {
		opts.Priority = PriorityHigh
	}
	if strings.TrimSpace(opts.Cell) == "" {
		opts.Cell = defaultIssueScanCell
	}
	if strings.TrimSpace(opts.RiskClass) == "" {
		opts.RiskClass = defaultIssueScanRiskClass
	}
	if len(opts.ExpectedOutputs) == 0 {
		opts.ExpectedOutputs = cloneStrings(def.ExpectedOutputs)
	}
	if strings.TrimSpace(opts.Title) == "" {
		opts.Title = fmt.Sprintf("Issue-scan stage %d: %s (%s)", def.Number, def.Title, opts.Target.Ref())
	}
	if strings.TrimSpace(opts.Description) == "" {
		opts.Description = fmt.Sprintf("Issue-scan run: %s\nTarget: %s\nStage %d: %s\nGate: %s", opts.RunID, opts.Target.Ref(), def.Number, def.Title, def.Gate)
	}
	return opts, def, nil
}

func issueScanStageRecord(task Task, opts IssueScanStageOptions, def issueScanStageDefinition, created bool, duplicateOf types.EventID) IssueScanStageRecord {
	return IssueScanStageRecord{
		IssueScanStageRef: IssueScanStageRef{
			TaskID: task.ID,
			RunID:  opts.RunID,
			Target: opts.Target,
			Stage:  opts.Stage,
		},
		Task:            task,
		StageNumber:     def.Number,
		Gate:            def.Gate,
		Created:         created,
		DuplicateOf:     duplicateOf,
		CanonicalTaskID: opts.CanonicalTaskID,
		FactoryOrderID:  opts.FactoryOrderID,
	}
}

func issueScanBaseID(runID string, target IssueScanTarget, stage IssueScanStageID) string {
	return "issue_scan_" + issueScanIDPart(runID) + "_" + issueScanIDPart(target.Repository) + "_" + strconv.Itoa(target.IssueNumber) + "_" + issueScanIDPart(string(stage))
}

func issueScanIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastUnderscore := true
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "unknown"
	}
	return out
}

func taskFromCreatedContent(id types.EventID, c TaskCreatedContent) Task {
	p := c.Priority
	if p == "" {
		p = DefaultPriority
	}
	return Task{
		ID:                     id,
		Title:                  c.Title,
		Description:            c.Description,
		CreatedBy:              c.CreatedBy,
		Priority:               p,
		Workspace:              c.Workspace,
		CanonicalTaskID:        c.CanonicalTaskID,
		FactoryOrderID:         c.FactoryOrderID,
		RequirementIDs:         cloneStrings(c.RequirementIDs),
		AcceptanceCriterionIDs: cloneStrings(c.AcceptanceCriterionIDs),
		Cell:                   c.Cell,
		RiskClass:              c.RiskClass,
		ExpectedOutputs:        cloneStrings(c.ExpectedOutputs),
	}
}

func validateIssueScanStageRef(ref IssueScanStageRef) error {
	if ref.TaskID.IsZero() {
		return fmt.Errorf("issue-scan task_id is required")
	}
	if strings.TrimSpace(ref.RunID) == "" {
		return fmt.Errorf("issue-scan run_id is required")
	}
	if strings.TrimSpace(ref.Target.Repository) == "" {
		return fmt.Errorf("issue-scan target repository is required")
	}
	if ref.Target.IssueNumber <= 0 {
		return fmt.Errorf("issue-scan target issue number must be positive")
	}
	if _, ok := issueScanStageDefinitions[ref.Stage]; !ok {
		return fmt.Errorf("unknown issue-scan stage %q", ref.Stage)
	}
	return nil
}

func validateIssueScanBlocker(blocker IssueScanBlocker) error {
	if !blocker.Reason.known() {
		return fmt.Errorf("unknown issue-scan blocker reason %q", blocker.Reason)
	}
	if strings.TrimSpace(blocker.Detail) == "" {
		return fmt.Errorf("issue-scan blocker detail is required")
	}
	return nil
}

func (r IssueScanBlockerReason) known() bool {
	switch r {
	case IssueScanBlockerNeedsHumanScope, IssueScanBlockerProtectedAction, IssueScanBlockerStaleTarget,
		IssueScanBlockerDuplicateChain, IssueScanBlockerMissingGateEvidence:
		return true
	default:
		return false
	}
}

func (r IssueScanBlockerReason) taskStatus() TaskStatus {
	switch r {
	case IssueScanBlockerNeedsHumanScope, IssueScanBlockerProtectedAction:
		return StatusPolicyBlocked
	default:
		return StatusBlocked
	}
}

func (b IssueScanStageBlockedContent) same(blocker IssueScanBlocker) bool {
	return b.BlockerReason == blocker.Reason &&
		strings.TrimSpace(b.Detail) == strings.TrimSpace(blocker.Detail) &&
		strings.Join(cloneStrings(b.EvidenceRefs), "\x00") == strings.Join(cloneStrings(blocker.EvidenceRefs), "\x00")
}

func (ts *TaskStore) latestIssueScanBlocker(taskID types.EventID) (IssueScanStageBlockedContent, bool, error) {
	page, err := ts.store.ByType(EventTypeIssueScanStageBlocked, 1000, types.None[types.Cursor]())
	if err != nil {
		return IssueScanStageBlockedContent{}, false, fmt.Errorf("fetch issue-scan blocker events: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(IssueScanStageBlockedContent)
		if ok && c.TaskID == taskID {
			return c, true, nil
		}
	}
	return IssueScanStageBlockedContent{}, false, nil
}

func (ts *TaskStore) latestIssueScanGate(taskID types.EventID) (IssueScanStageGateSatisfiedContent, bool, error) {
	page, err := ts.store.ByType(EventTypeIssueScanStageGateSatisfied, 1000, types.None[types.Cursor]())
	if err != nil {
		return IssueScanStageGateSatisfiedContent{}, false, fmt.Errorf("fetch issue-scan gate events: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(IssueScanStageGateSatisfiedContent)
		if ok && c.TaskID == taskID {
			return c, true, nil
		}
	}
	return IssueScanStageGateSatisfiedContent{}, false, nil
}

func (ts *TaskStore) transitionIssueScanStageTo(
	source types.ActorID,
	taskID types.EventID,
	target TaskStatus,
	reason string,
	evidenceRefs []string,
	causes []types.EventID,
	convID types.ConversationID,
) (TaskStatus, error) {
	current, err := ts.GetStatus(taskID)
	if err != nil {
		return "", err
	}
	for current != target {
		next, ok := nextIssueScanTransition(current, target)
		if !ok {
			return "", fmt.Errorf("%w: cannot move issue-scan stage %s -> %s", ErrInvalidLifecycleTransition, current, target)
		}
		if err := ts.TransitionTask(source, taskID, next, reason, evidenceRefs, causes, convID); err != nil {
			return "", err
		}
		current = next
	}
	return current, nil
}

func nextIssueScanTransition(current, target TaskStatus) (TaskStatus, bool) {
	switch current {
	case StatusCreated:
		return StatusReady, true
	case StatusReady:
		return StatusRunning, true
	case StatusRunning:
		switch target {
		case StatusBlocked, StatusPolicyBlocked, StatusVerified, StatusCertified:
			if target == StatusCertified {
				return StatusVerified, true
			}
			return target, true
		default:
			return "", false
		}
	case StatusBlocked:
		if target == StatusReady || target == StatusRunning {
			return StatusReady, true
		}
	case StatusVerified:
		if target == StatusCertified {
			return StatusCertified, true
		}
	}
	return "", false
}
