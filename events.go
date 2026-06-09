package work

import (
	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// TaskPriority represents the urgency level of a task.
type TaskPriority string

const (
	// PriorityLow is for tasks that can wait.
	PriorityLow TaskPriority = "low"
	// PriorityMedium is the default priority when none is specified.
	PriorityMedium TaskPriority = "medium"
	// PriorityHigh is for tasks that should be addressed soon.
	PriorityHigh TaskPriority = "high"
	// PriorityCritical is for tasks that must be addressed immediately.
	PriorityCritical TaskPriority = "critical"

	// DefaultPriority is the priority assigned when none is specified at creation.
	DefaultPriority = PriorityMedium

	// GateDefinitionOfDone is the task artifact label for the concrete completion contract.
	GateDefinitionOfDone = "definition_of_done"
	// GateAcceptanceCriteria is the task artifact label for user-visible pass/fail criteria.
	GateAcceptanceCriteria = "acceptance_criteria"
	// GateTestPlan is the task artifact label for executable verification steps.
	GateTestPlan = "test_plan"
	// FactoryOrderModelOverridesArtifactLabel carries validated per-order model
	// override policy as structured JSON, not free-form markdown intent text.
	FactoryOrderModelOverridesArtifactLabel = "factory_order_model_overrides"
)

// RequiredReadinessGateLabels returns the artifact labels that must exist before
// implementation work can be assigned. The returned slice is safe to mutate.
func RequiredReadinessGateLabels() []string {
	return []string{GateDefinitionOfDone, GateAcceptanceCriteria, GateTestPlan}
}

// Work Graph event types — Layer 1 of the thirteen-product roadmap.
var (
	EventTypeTaskCreated               = types.MustEventType("work.task.created")
	EventTypeTaskAssigned              = types.MustEventType("work.task.assigned")
	EventTypeTaskCompleted             = types.MustEventType("work.task.completed")
	EventTypeTaskDependencyAdded       = types.MustEventType("work.task.dependency.added")
	EventTypeTaskPrioritySet           = types.MustEventType("work.task.priority.set")
	EventTypeTaskComment               = types.MustEventType("work.task.comment")
	EventTypeTaskUnblocked             = types.MustEventType("work.task.unblocked")
	EventTypeTaskArtifact              = types.MustEventType("work.task.artifact")
	EventTypeTaskArtifactWaived        = types.MustEventType("work.task.artifact.waived")
	EventTypeTaskFactRequired          = types.MustEventType("work.task.fact.required")
	EventTypeTaskLifecycleTransitioned = types.MustEventType("work.task.lifecycle.transitioned")
	EventTypeTaskLinked                = types.MustEventType("work.task.linked")
	EventTypeTaskVerificationAttached  = types.MustEventType("work.task.verification.attached")
	EventTypeTaskFailureRepairAttached = types.MustEventType("work.task.failure.repair.attached")
	EventTypePhaseGateDeclared         = types.MustEventType("work.phase.gate.declared")
	EventTypePhaseGateApproved         = types.MustEventType("work.phase.gate.approved")
	EventTypePhaseGateRejected         = types.MustEventType("work.phase.gate.rejected")
)

// allWorkEventTypes returns all work event types for registration.
func allWorkEventTypes() []types.EventType {
	return []types.EventType{
		EventTypeTaskCreated, EventTypeTaskAssigned, EventTypeTaskCompleted,
		EventTypeTaskDependencyAdded, EventTypeTaskPrioritySet, EventTypeTaskComment,
		EventTypeTaskUnblocked, EventTypeTaskArtifact, EventTypeTaskArtifactWaived,
		EventTypeTaskFactRequired, EventTypeTaskLifecycleTransitioned, EventTypeTaskLinked,
		EventTypeTaskVerificationAttached, EventTypeTaskFailureRepairAttached,
		EventTypePhaseGateDeclared, EventTypePhaseGateApproved, EventTypePhaseGateRejected,
		EventTypeRuntimeEnvelopeRecorded, EventTypeRuntimeResultRecorded,
	}
}

// workContent is embedded in all work content types. Work events use
// no-op Accept (same pattern as pipeline content) since they are hive-specific.
type workContent struct{}

func (workContent) Accept(event.EventContentVisitor) {}

// --- Content structs ---

// TaskCreatedContent is emitted when a new task is created.
type TaskCreatedContent struct {
	workContent
	Title                  string        `json:"Title"`
	Description            string        `json:"Description,omitempty"`
	CreatedBy              types.ActorID `json:"CreatedBy"`
	Priority               TaskPriority  `json:"Priority,omitempty"`
	Workspace              string        `json:"Workspace,omitempty"`
	CanonicalTaskID        string        `json:"CanonicalTaskID,omitempty"`
	FactoryOrderID         string        `json:"FactoryOrderID,omitempty"`
	RequirementIDs         []string      `json:"RequirementIDs,omitempty"`
	AcceptanceCriterionIDs []string      `json:"AcceptanceCriterionIDs,omitempty"`
	Cell                   string        `json:"Cell,omitempty"`
	RiskClass              string        `json:"RiskClass,omitempty"`
	ExpectedOutputs        []string      `json:"ExpectedOutputs,omitempty"`
}

func (c TaskCreatedContent) EventTypeName() string { return "work.task.created" }

// TaskAssignedContent is emitted when a task is assigned to an actor.
type TaskAssignedContent struct {
	workContent
	TaskID     types.EventID `json:"TaskID"`
	AssignedTo types.ActorID `json:"AssignedTo"`
	AssignedBy types.ActorID `json:"AssignedBy"`
}

func (c TaskAssignedContent) EventTypeName() string { return "work.task.assigned" }

// TaskCompletedContent is emitted when a task is completed.
// ArtifactRef points to the work.task.artifact or work.task.artifact.waived
// event that satisfied the completion gate. Auto-populated by Complete().
type TaskCompletedContent struct {
	workContent
	TaskID      types.EventID `json:"TaskID"`
	CompletedBy types.ActorID `json:"CompletedBy"`
	Summary     string        `json:"Summary,omitempty"`
	ArtifactRef types.EventID `json:"ArtifactRef,omitempty"`
}

func (c TaskCompletedContent) EventTypeName() string { return "work.task.completed" }

// TaskDependencyContent is emitted when a dependency is declared between two tasks.
// It records that TaskID depends on DependsOnID — TaskID cannot start until DependsOnID completes.
type TaskDependencyContent struct {
	workContent
	TaskID      types.EventID `json:"TaskID"`
	DependsOnID types.EventID `json:"DependsOnID"`
	AddedBy     types.ActorID `json:"AddedBy"`
}

func (c TaskDependencyContent) EventTypeName() string { return "work.task.dependency.added" }

// TaskPrioritySetContent is emitted when a task's priority is updated post-creation.
type TaskPrioritySetContent struct {
	workContent
	TaskID   types.EventID `json:"TaskID"`
	Priority TaskPriority  `json:"Priority"`
	SetBy    types.ActorID `json:"SetBy"`
}

func (c TaskPrioritySetContent) EventTypeName() string { return "work.task.priority.set" }

// CommentContent is emitted when a freeform note is added to a task.
type CommentContent struct {
	workContent
	TaskID   types.EventID `json:"TaskID"`
	Body     string        `json:"Body"`
	AuthorID types.ActorID `json:"AuthorID"`
}

func (c CommentContent) EventTypeName() string { return "work.task.comment" }

// TaskUnblockedContent is emitted when a task's blockers are explicitly marked resolved.
// It overrides any active dependency-based blocked state for the task.
type TaskUnblockedContent struct {
	workContent
	TaskID      types.EventID `json:"TaskID"`
	UnblockedBy types.ActorID `json:"UnblockedBy"`
}

func (c TaskUnblockedContent) EventTypeName() string { return "work.task.unblocked" }

// TaskArtifactContent is emitted when an agent attaches a deliverable to a task.
// Multiple artifacts per task are expected. Must be appended before completion.
type TaskArtifactContent struct {
	workContent
	TaskID    types.EventID `json:"TaskID"`
	Label     string        `json:"Label"`
	MediaType string        `json:"MediaType"`
	Body      string        `json:"Body"`
	CreatedBy types.ActorID `json:"CreatedBy"`
}

func (c TaskArtifactContent) EventTypeName() string { return "work.task.artifact" }

// TaskArtifactWaivedContent is emitted to explicitly exempt a task from the
// artifact requirement. The completion gate accepts either an artifact or a waiver.
type TaskArtifactWaivedContent struct {
	workContent
	TaskID   types.EventID `json:"TaskID"`
	Reason   string        `json:"Reason"`
	WaivedBy types.ActorID `json:"WaivedBy"`
}

func (c TaskArtifactWaivedContent) EventTypeName() string { return "work.task.artifact.waived" }

// TaskFactRequiredContent records a readiness prerequisite on an external
// EventGraph fact. Work replays the requirement but does not decide authority.
type TaskFactRequiredContent struct {
	workContent
	TaskID            types.EventID   `json:"TaskID"`
	RequiredEventType types.EventType `json:"RequiredEventType"`
	RequiredEventID   types.EventID   `json:"RequiredEventID,omitempty"`
	Reason            string          `json:"Reason,omitempty"`
	RequiredBy        types.ActorID   `json:"RequiredBy"`
}

func (c TaskFactRequiredContent) EventTypeName() string { return "work.task.fact.required" }

// TaskLifecycleTransitionContent records an explicit v3.9 Task lifecycle move.
type TaskLifecycleTransitionContent struct {
	workContent
	TaskID       types.EventID `json:"TaskID"`
	FromState    TaskStatus    `json:"FromState"`
	ToState      TaskStatus    `json:"ToState"`
	Reason       string        `json:"Reason,omitempty"`
	EvidenceRefs []string      `json:"EvidenceRefs,omitempty"`
	SupersededBy string        `json:"SupersededBy,omitempty"`
	ChangedBy    types.ActorID `json:"ChangedBy"`
}

func (c TaskLifecycleTransitionContent) EventTypeName() string {
	return "work.task.lifecycle.transitioned"
}

// TaskLinkedContent attaches v3.9 Tier 0 product lineage references to a Work task.
type TaskLinkedContent struct {
	workContent
	TaskID                 types.EventID `json:"TaskID"`
	CanonicalTaskID        string        `json:"CanonicalTaskID,omitempty"`
	FactoryOrderID         string        `json:"FactoryOrderID,omitempty"`
	RequirementIDs         []string      `json:"RequirementIDs,omitempty"`
	AcceptanceCriterionIDs []string      `json:"AcceptanceCriterionIDs,omitempty"`
	LinkedBy               types.ActorID `json:"LinkedBy"`
}

func (c TaskLinkedContent) EventTypeName() string { return "work.task.linked" }

// TaskVerificationAttachedContent attaches v3.9 verification evidence refs to a task.
type TaskVerificationAttachedContent struct {
	workContent
	TaskID        types.EventID `json:"TaskID"`
	TestCaseIDs   []string      `json:"TestCaseIDs,omitempty"`
	TestRunIDs    []string      `json:"TestRunIDs,omitempty"`
	GateResultIDs []string      `json:"GateResultIDs,omitempty"`
	WaiverIDs     []string      `json:"WaiverIDs,omitempty"`
	Summary       string        `json:"Summary,omitempty"`
	AttachedBy    types.ActorID `json:"AttachedBy"`
}

func (c TaskVerificationAttachedContent) EventTypeName() string {
	return "work.task.verification.attached"
}

// TaskFailureRepairAttachedContent attaches v3.9 failure and repair refs to a task.
type TaskFailureRepairAttachedContent struct {
	workContent
	TaskID           types.EventID `json:"TaskID"`
	FailureIDs       []string      `json:"FailureIDs,omitempty"`
	RepairAttemptIDs []string      `json:"RepairAttemptIDs,omitempty"`
	WaiverIDs        []string      `json:"WaiverIDs,omitempty"`
	Summary          string        `json:"Summary,omitempty"`
	AttachedBy       types.ActorID `json:"AttachedBy"`
}

func (c TaskFailureRepairAttachedContent) EventTypeName() string {
	return "work.task.failure.repair.attached"
}

// PhaseGateDeclaredContent is emitted when a phase needs explicit approval.
type PhaseGateDeclaredContent struct {
	workContent
	Phase      string        `json:"Phase"`
	Title      string        `json:"Title"`
	Criteria   []string      `json:"Criteria,omitempty"`
	DeclaredBy types.ActorID `json:"DeclaredBy"`
}

func (c PhaseGateDeclaredContent) EventTypeName() string { return "work.phase.gate.declared" }

// PhaseGateApprovedContent records approval for a declared phase gate.
type PhaseGateApprovedContent struct {
	workContent
	GateID     types.EventID `json:"GateID"`
	Phase      string        `json:"Phase,omitempty"`
	ApprovedBy types.ActorID `json:"ApprovedBy"`
	Summary    string        `json:"Summary,omitempty"`
}

func (c PhaseGateApprovedContent) EventTypeName() string { return "work.phase.gate.approved" }

// PhaseGateRejectedContent records rejection for a declared phase gate.
type PhaseGateRejectedContent struct {
	workContent
	GateID     types.EventID `json:"GateID"`
	Phase      string        `json:"Phase,omitempty"`
	RejectedBy types.ActorID `json:"RejectedBy"`
	Reason     string        `json:"Reason,omitempty"`
}

func (c PhaseGateRejectedContent) EventTypeName() string { return "work.phase.gate.rejected" }

// RegisterEventTypes registers work content unmarshalers for Postgres
// deserialization. Call this before querying work events from the store.
func RegisterEventTypes() {
	event.RegisterContentUnmarshaler("work.task.created", event.Unmarshal[TaskCreatedContent])
	event.RegisterContentUnmarshaler("work.task.assigned", event.Unmarshal[TaskAssignedContent])
	event.RegisterContentUnmarshaler("work.task.completed", event.Unmarshal[TaskCompletedContent])
	event.RegisterContentUnmarshaler("work.task.dependency.added", event.Unmarshal[TaskDependencyContent])
	event.RegisterContentUnmarshaler("work.task.priority.set", event.Unmarshal[TaskPrioritySetContent])
	event.RegisterContentUnmarshaler("work.task.comment", event.Unmarshal[CommentContent])
	event.RegisterContentUnmarshaler("work.task.unblocked", event.Unmarshal[TaskUnblockedContent])
	event.RegisterContentUnmarshaler("work.task.artifact", event.Unmarshal[TaskArtifactContent])
	event.RegisterContentUnmarshaler("work.task.artifact.waived", event.Unmarshal[TaskArtifactWaivedContent])
	event.RegisterContentUnmarshaler("work.task.fact.required", event.Unmarshal[TaskFactRequiredContent])
	event.RegisterContentUnmarshaler("work.task.lifecycle.transitioned", event.Unmarshal[TaskLifecycleTransitionContent])
	event.RegisterContentUnmarshaler("work.task.linked", event.Unmarshal[TaskLinkedContent])
	event.RegisterContentUnmarshaler("work.task.verification.attached", event.Unmarshal[TaskVerificationAttachedContent])
	event.RegisterContentUnmarshaler("work.task.failure.repair.attached", event.Unmarshal[TaskFailureRepairAttachedContent])
	event.RegisterContentUnmarshaler("work.phase.gate.declared", event.Unmarshal[PhaseGateDeclaredContent])
	event.RegisterContentUnmarshaler("work.phase.gate.approved", event.Unmarshal[PhaseGateApprovedContent])
	event.RegisterContentUnmarshaler("work.phase.gate.rejected", event.Unmarshal[PhaseGateRejectedContent])
	event.RegisterContentUnmarshaler("work.runtime.envelope.recorded", event.Unmarshal[RuntimeEnvelopeRecordedContent])
	event.RegisterContentUnmarshaler("work.runtime.result.recorded", event.Unmarshal[RuntimeResultRecordedContent])
}

// RegisterWithRegistry registers all work event types with the given registry
// and registers content unmarshalers for Postgres deserialization.
func RegisterWithRegistry(registry *event.EventTypeRegistry) {
	for _, et := range allWorkEventTypes() {
		registry.Register(et, nil)
	}
	RegisterEventTypes()
}
