package work

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/store"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// PhaseGateStatus is the replayed approval state for a phase gate.
type PhaseGateStatus string

const (
	PhaseGatePending  PhaseGateStatus = "pending"
	PhaseGateApproved PhaseGateStatus = "approved"
	PhaseGateRejected PhaseGateStatus = "rejected"

	// PhaseGateTLC51LinkArtifactLabel identifies task artifacts that join a
	// replayed Work phase gate to one TLC 5.1 obligation history.
	PhaseGateTLC51LinkArtifactLabel = "factory_tlc51_phase_gate_link"
	// PhaseGateTLC51LinkArtifactMediaType keeps the link machine-readable and
	// distinct from intent or authority-bearing Human records.
	PhaseGateTLC51LinkArtifactMediaType = "application/vnd.transpara.factory-tlc51-phase-gate-link+json"
	// PhaseGateTLC51LinkSchemaVersion closes the durable Work link shape.
	PhaseGateTLC51LinkSchemaVersion = "work-factory-tlc51-phase-gate-link/v1"
)

// PhaseGateTLC51Link binds an existing Work phase gate to the exact first
// EventGraph/Work twin for one TLC 5.1 obligation. Later obligation events are
// joined by their immutable order/change-series/plan/subject/obligation
// identity. The link does not turn a generic Work approval into TLC evidence.
type PhaseGateTLC51Link struct {
	SchemaVersion       string `json:"schema_version"`
	GateID              string `json:"gate_id"`
	Phase               string `json:"phase"`
	FactoryOrderID      string `json:"factory_order_id"`
	ChangeSeriesID      string `json:"change_series_id"`
	PlanDigest          string `json:"plan_digest"`
	SubjectDigest       string `json:"subject_digest"`
	ObligationID        string `json:"obligation_id"`
	LinkedEventOrdinal  uint64 `json:"linked_event_ordinal"`
	LinkedEventType     string `json:"linked_event_type"`
	LinkedPayloadSHA256 string `json:"linked_payload_sha256"`
}

// PhaseGateTLC51Projection is a non-authoritative joined replay. Even when a
// Work phase gate is approved and its Work twin says the obligation passed,
// TLCSatisfactionCredited remains false until Hive reconciles the source entry
// against EventGraph and the TLC evaluator validates exact evidence.
type PhaseGateTLC51Projection struct {
	Link                           PhaseGateTLC51Link `json:"link"`
	Gate                           *PhaseGateState    `json:"gate,omitempty"`
	LatestEventOrdinal             uint64             `json:"latest_event_ordinal,omitempty"`
	LatestEventType                string             `json:"latest_event_type,omitempty"`
	ReportedObligationOutcome      string             `json:"reported_obligation_outcome"`
	EventGraphVerificationRequired bool               `json:"eventgraph_verification_required"`
	TLCSatisfactionCredited        bool               `json:"tlc_satisfaction_credited"`
	AuthorityGranted               bool               `json:"authority_granted"`
	Quarantined                    bool               `json:"quarantined"`
	HumanInterventionRequired      bool               `json:"human_intervention_required"`
	Reason                         string             `json:"reason,omitempty"`
}

// PhaseGateState is the current state of a declared phase gate.
type PhaseGateState struct {
	ID         types.EventID
	Phase      string
	Title      string
	Criteria   []string
	Status     PhaseGateStatus
	DeclaredBy types.ActorID
	ApprovedBy types.ActorID
	RejectedBy types.ActorID
	Summary    string
	Reason     string
	DeclaredAt time.Time
	UpdatedAt  time.Time
}

// PhaseGateStore records and replays auditable phase gate decisions.
type PhaseGateStore struct {
	store   store.Store
	factory *event.EventFactory
	signer  event.Signer
}

// NewPhaseGateStore creates a phase gate store backed by the given event store.
func NewPhaseGateStore(s store.Store, factory *event.EventFactory, signer event.Signer) *PhaseGateStore {
	return &PhaseGateStore{store: s, factory: factory, signer: signer}
}

// Declare records a pending phase gate and returns its replayed state.
func (pg *PhaseGateStore) Declare(
	source types.ActorID,
	phase, title string,
	criteria []string,
	causes []types.EventID,
	convID types.ConversationID,
) (PhaseGateState, error) {
	phase = strings.TrimSpace(phase)
	title = strings.TrimSpace(title)
	if phase == "" {
		return PhaseGateState{}, fmt.Errorf("phase is required")
	}
	if title == "" {
		return PhaseGateState{}, fmt.Errorf("title is required")
	}
	content := PhaseGateDeclaredContent{
		Phase:      phase,
		Title:      title,
		Criteria:   cleanCriteria(criteria),
		DeclaredBy: source,
	}
	ev, err := pg.factory.Create(EventTypePhaseGateDeclared, source, content, causes, convID, pg.store, pg.signer)
	if err != nil {
		return PhaseGateState{}, fmt.Errorf("create phase gate event: %w", err)
	}
	stored, err := pg.store.Append(ev)
	if err != nil {
		return PhaseGateState{}, fmt.Errorf("append phase gate event: %w", err)
	}
	return PhaseGateState{
		ID:         stored.ID(),
		Phase:      phase,
		Title:      title,
		Criteria:   content.Criteria,
		Status:     PhaseGatePending,
		DeclaredBy: source,
		DeclaredAt: stored.Timestamp().Value(),
		UpdatedAt:  stored.Timestamp().Value(),
	}, nil
}

// Approve records approval for a declared phase gate.
func (pg *PhaseGateStore) Approve(
	source types.ActorID,
	gateID types.EventID,
	summary string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	state, ok, err := pg.Get(gateID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("phase gate not found: %s", gateID.Value())
	}
	content := PhaseGateApprovedContent{
		GateID:     gateID,
		Phase:      state.Phase,
		ApprovedBy: source,
		Summary:    strings.TrimSpace(summary),
	}
	ev, err := pg.factory.Create(EventTypePhaseGateApproved, source, content, causes, convID, pg.store, pg.signer)
	if err != nil {
		return fmt.Errorf("create phase gate approval event: %w", err)
	}
	if _, err := pg.store.Append(ev); err != nil {
		return fmt.Errorf("append phase gate approval event: %w", err)
	}
	return nil
}

// Reject records rejection for a declared phase gate.
func (pg *PhaseGateStore) Reject(
	source types.ActorID,
	gateID types.EventID,
	reason string,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	state, ok, err := pg.Get(gateID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("phase gate not found: %s", gateID.Value())
	}
	content := PhaseGateRejectedContent{
		GateID:     gateID,
		Phase:      state.Phase,
		RejectedBy: source,
		Reason:     strings.TrimSpace(reason),
	}
	ev, err := pg.factory.Create(EventTypePhaseGateRejected, source, content, causes, convID, pg.store, pg.signer)
	if err != nil {
		return fmt.Errorf("create phase gate rejection event: %w", err)
	}
	if _, err := pg.store.Append(ev); err != nil {
		return fmt.Errorf("append phase gate rejection event: %w", err)
	}
	return nil
}

// Get returns the replayed state for a declared phase gate.
func (pg *PhaseGateStore) Get(gateID types.EventID) (PhaseGateState, bool, error) {
	gates, err := pg.List(1000)
	if err != nil {
		return PhaseGateState{}, false, err
	}
	for _, gate := range gates {
		if gate.ID == gateID {
			return gate, true, nil
		}
	}
	return PhaseGateState{}, false, nil
}

// List returns replayed phase gates, newest declarations first.
func (pg *PhaseGateStore) List(limit int) ([]PhaseGateState, error) {
	if limit <= 0 {
		limit = 20
	}
	gates := map[types.EventID]*PhaseGateState{}
	declared, err := pg.store.ByType(EventTypePhaseGateDeclared, 1000, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("list phase gate declarations: %w", err)
	}
	for _, ev := range declared.Items() {
		c, ok := ev.Content().(PhaseGateDeclaredContent)
		if !ok {
			continue
		}
		gates[ev.ID()] = &PhaseGateState{
			ID:         ev.ID(),
			Phase:      c.Phase,
			Title:      c.Title,
			Criteria:   append([]string(nil), c.Criteria...),
			Status:     PhaseGatePending,
			DeclaredBy: c.DeclaredBy,
			DeclaredAt: ev.Timestamp().Value(),
			UpdatedAt:  ev.Timestamp().Value(),
		}
	}
	if err := pg.applyApprovals(gates); err != nil {
		return nil, err
	}
	if err := pg.applyRejections(gates); err != nil {
		return nil, err
	}
	out := make([]PhaseGateState, 0, len(gates))
	for _, gate := range gates {
		out = append(out, *gate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].DeclaredAt.After(out[j].DeclaredAt)
	})
	if len(out) > limit {
		return out[:limit], nil
	}
	return out, nil
}

func (pg *PhaseGateStore) applyApprovals(gates map[types.EventID]*PhaseGateState) error {
	page, err := pg.store.ByType(EventTypePhaseGateApproved, 1000, types.None[types.Cursor]())
	if err != nil {
		return fmt.Errorf("list phase gate approvals: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(PhaseGateApprovedContent)
		if !ok {
			continue
		}
		gate, ok := gates[c.GateID]
		if !ok || ev.Timestamp().Value().Before(gate.UpdatedAt) {
			continue
		}
		gate.Status = PhaseGateApproved
		gate.ApprovedBy = c.ApprovedBy
		gate.RejectedBy = types.ActorID{}
		gate.Summary = c.Summary
		gate.Reason = ""
		gate.UpdatedAt = ev.Timestamp().Value()
	}
	return nil
}

func (pg *PhaseGateStore) applyRejections(gates map[types.EventID]*PhaseGateState) error {
	page, err := pg.store.ByType(EventTypePhaseGateRejected, 1000, types.None[types.Cursor]())
	if err != nil {
		return fmt.Errorf("list phase gate rejections: %w", err)
	}
	for _, ev := range page.Items() {
		c, ok := ev.Content().(PhaseGateRejectedContent)
		if !ok {
			continue
		}
		gate, ok := gates[c.GateID]
		if !ok || ev.Timestamp().Value().Before(gate.UpdatedAt) {
			continue
		}
		gate.Status = PhaseGateRejected
		gate.ApprovedBy = types.ActorID{}
		gate.RejectedBy = c.RejectedBy
		gate.Summary = ""
		gate.Reason = c.Reason
		gate.UpdatedAt = ev.Timestamp().Value()
	}
	return nil
}

func cleanCriteria(criteria []string) []string {
	out := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		if trimmed := strings.TrimSpace(criterion); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func validatePhaseGateTLC51Link(value PhaseGateTLC51Link, expectedFactoryOrderID string) error {
	if value.SchemaVersion != PhaseGateTLC51LinkSchemaVersion {
		return fmt.Errorf("schema_version must be %q", PhaseGateTLC51LinkSchemaVersion)
	}
	if value.GateID == "" || value.Phase == "" || value.FactoryOrderID == "" || value.ChangeSeriesID == "" || value.ObligationID == "" || value.LinkedEventOrdinal == 0 {
		return fmt.Errorf("gate, phase, order, change-series, obligation, and linked event ordinal are required")
	}
	if expectedFactoryOrderID != "" && value.FactoryOrderID != expectedFactoryOrderID {
		return fmt.Errorf("factory_order_id %q does not match task order %q", value.FactoryOrderID, expectedFactoryOrderID)
	}
	for _, field := range []string{value.GateID, value.Phase, value.FactoryOrderID, value.ChangeSeriesID, value.ObligationID, value.LinkedEventType} {
		if hasControlRune(field) {
			return fmt.Errorf("TLC 5.1 phase-gate link contains control characters")
		}
	}
	if !validFactoryOrderTLC51SHA(value.PlanDigest) || !validFactoryOrderTLC51SHA(value.SubjectDigest) || !validFactoryOrderTLC51SHA(value.LinkedPayloadSHA256) {
		return fmt.Errorf("plan, subject, and linked payload digests must be lowercase SHA-256")
	}
	if value.LinkedEventType != "factory.tlc51.obligation.ready" &&
		value.LinkedEventType != "factory.tlc51.obligation.claimed" &&
		value.LinkedEventType != "factory.tlc51.obligation.running" &&
		value.LinkedEventType != "factory.tlc51.obligation.terminal" &&
		value.LinkedEventType != "factory.tlc51.evidence.linked" {
		return fmt.Errorf("linked_event_type %q does not carry a TLC 5.1 obligation", value.LinkedEventType)
	}
	return nil
}

// PhaseGateTLC51LinkArtifactBody returns the closed JSON body used for a
// durable task artifact.
func PhaseGateTLC51LinkArtifactBody(value PhaseGateTLC51Link) (string, error) {
	if err := validatePhaseGateTLC51Link(value, ""); err != nil {
		return "", err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal TLC 5.1 phase-gate link: %w", err)
	}
	return string(body), nil
}

func parsePhaseGateTLC51LinkArtifactBody(body, expectedFactoryOrderID string) (PhaseGateTLC51Link, error) {
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	var value PhaseGateTLC51Link
	if err := decoder.Decode(&value); err != nil {
		return PhaseGateTLC51Link{}, fmt.Errorf("decode TLC 5.1 phase-gate link: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return PhaseGateTLC51Link{}, fmt.Errorf("TLC 5.1 phase-gate link must contain exactly one JSON value")
	}
	if err := validatePhaseGateTLC51Link(value, expectedFactoryOrderID); err != nil {
		return PhaseGateTLC51Link{}, err
	}
	return value, nil
}

func phaseGateTLC51LinksEqual(left, right PhaseGateTLC51Link) bool {
	return left == right
}

func (ts *TaskStore) projectPhaseGateTLC51Links(taskID types.EventID) ([]PhaseGateTLC51Link, map[string]struct{}, error) {
	task, err := ts.ProjectTask(taskID)
	if err != nil {
		return nil, nil, err
	}
	artifacts, err := ts.ListArtifacts(taskID)
	if err != nil {
		return nil, nil, err
	}
	byGate := map[string]PhaseGateTLC51Link{}
	conflicts := map[string]struct{}{}
	for _, artifact := range artifacts {
		if artifact.Label != PhaseGateTLC51LinkArtifactLabel {
			continue
		}
		if artifact.MediaType != PhaseGateTLC51LinkArtifactMediaType {
			return nil, nil, fmt.Errorf("%s artifact must use media type %s", PhaseGateTLC51LinkArtifactLabel, PhaseGateTLC51LinkArtifactMediaType)
		}
		value, err := parsePhaseGateTLC51LinkArtifactBody(artifact.Body, task.FactoryOrderID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid %s artifact %s: %w", PhaseGateTLC51LinkArtifactLabel, artifact.ID.Value(), err)
		}
		if existing, ok := byGate[value.GateID]; ok {
			if !phaseGateTLC51LinksEqual(existing, value) {
				conflicts[value.GateID] = struct{}{}
			}
			continue
		}
		byGate[value.GateID] = value
	}
	gateIDs := make([]string, 0, len(byGate))
	for gateID := range byGate {
		gateIDs = append(gateIDs, gateID)
	}
	sort.Strings(gateIDs)
	links := make([]PhaseGateTLC51Link, 0, len(gateIDs))
	for _, gateID := range gateIDs {
		links = append(links, byGate[gateID])
	}
	return links, conflicts, nil
}

func findPhaseGateByStringID(pg *PhaseGateStore, gateID string) (PhaseGateState, bool, error) {
	gates, err := pg.List(1000)
	if err != nil {
		return PhaseGateState{}, false, err
	}
	for _, gate := range gates {
		if gate.ID.Value() == gateID {
			return gate, true, nil
		}
	}
	return PhaseGateState{}, false, nil
}

func phaseGateTLC51EventMatchesLink(value FactoryOrderTLC51EventArtifact, link PhaseGateTLC51Link) (factoryOrderTLC51PayloadIdentity, bool) {
	if value.FactoryOrderID != link.FactoryOrderID || value.ChangeSeriesID != link.ChangeSeriesID ||
		value.EventOrdinal != link.LinkedEventOrdinal || value.EventType != link.LinkedEventType || value.PayloadSHA256 != link.LinkedPayloadSHA256 {
		return factoryOrderTLC51PayloadIdentity{}, false
	}
	identity, err := validateFactoryOrderTLC51EventArtifact(value, link.FactoryOrderID)
	if err != nil || identity.PlanDigest != link.PlanDigest || identity.SubjectDigest != link.SubjectDigest || identity.ObligationID != link.ObligationID {
		return factoryOrderTLC51PayloadIdentity{}, false
	}
	return identity, true
}

// AttachPhaseGateTLC51Link records a durable, idempotent Work link only after
// the gate and referenced Work event twin can both be replayed exactly. It does
// not verify EventGraph, satisfy a TLC obligation, or grant authority.
func (ts *TaskStore) AttachPhaseGateTLC51Link(
	pg *PhaseGateStore,
	source types.ActorID,
	taskID types.EventID,
	value PhaseGateTLC51Link,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	task, err := ts.ProjectTask(taskID)
	if err != nil {
		return fmt.Errorf("project FactoryOrder task: %w", err)
	}
	if err := validatePhaseGateTLC51Link(value, task.FactoryOrderID); err != nil {
		return err
	}
	gate, ok, err := findPhaseGateByStringID(pg, value.GateID)
	if err != nil {
		return err
	}
	if !ok || gate.Phase != value.Phase {
		return fmt.Errorf("phase gate %q with phase %q is not replayable", value.GateID, value.Phase)
	}
	events, err := ts.ProjectFactoryOrderTLC51EventArtifacts(taskID)
	if err != nil {
		return err
	}
	if events.Quarantined {
		return fmt.Errorf("TLC 5.1 Work linkage is quarantined; Human intervention required")
	}
	linked := false
	for _, candidate := range events.EventArtifacts {
		if _, linked = phaseGateTLC51EventMatchesLink(candidate, value); linked {
			break
		}
	}
	if !linked {
		return fmt.Errorf("referenced TLC 5.1 Work event twin is missing or does not bind obligation %q", value.ObligationID)
	}
	links, conflicts, err := ts.projectPhaseGateTLC51Links(taskID)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("TLC 5.1 phase-gate linkage is quarantined; Human intervention required")
	}
	for _, existing := range links {
		if existing.GateID != value.GateID {
			continue
		}
		if phaseGateTLC51LinksEqual(existing, value) {
			return nil
		}
		return fmt.Errorf("phase gate %q already has a conflicting TLC 5.1 link", value.GateID)
	}
	body, err := PhaseGateTLC51LinkArtifactBody(value)
	if err != nil {
		return err
	}
	return ts.AddArtifact(source, taskID, PhaseGateTLC51LinkArtifactLabel, PhaseGateTLC51LinkArtifactMediaType, body, causes, convID)
}

// ProjectPhaseGateTLC51Links joins replayed Work phase gates with Work event
// twins. It reports what the twin says while withholding satisfaction and
// authority until an external EventGraph/TLC reconciliation succeeds.
func (ts *TaskStore) ProjectPhaseGateTLC51Links(taskID types.EventID, pg *PhaseGateStore) ([]PhaseGateTLC51Projection, error) {
	links, conflicts, err := ts.projectPhaseGateTLC51Links(taskID)
	if err != nil {
		return nil, err
	}
	events, err := ts.ProjectFactoryOrderTLC51EventArtifacts(taskID)
	if err != nil {
		return nil, err
	}
	result := make([]PhaseGateTLC51Projection, 0, len(links))
	for _, link := range links {
		projection := PhaseGateTLC51Projection{
			Link:                           link,
			ReportedObligationOutcome:      "unknown",
			EventGraphVerificationRequired: true,
			TLCSatisfactionCredited:        false,
			AuthorityGranted:               false,
		}
		gate, gateExists, gateErr := findPhaseGateByStringID(pg, link.GateID)
		if gateErr != nil {
			return nil, gateErr
		}
		if gateExists {
			gateCopy := gate
			projection.Gate = &gateCopy
		}
		if _, conflict := conflicts[link.GateID]; conflict {
			projection.Quarantined = true
			projection.HumanInterventionRequired = true
			projection.Reason = "conflicting phase-gate Work twins"
		}
		if events.Quarantined {
			projection.Quarantined = true
			projection.HumanInterventionRequired = true
			projection.Reason = "conflicting EventGraph/Work event twins"
		}
		if !gateExists || gate.Phase != link.Phase {
			projection.Quarantined = true
			projection.HumanInterventionRequired = true
			projection.Reason = "linked phase gate is missing or changed"
		}
		linkedEventFound := false
		for _, candidate := range events.EventArtifacts {
			identity, err := validateFactoryOrderTLC51EventArtifact(candidate, link.FactoryOrderID)
			if err != nil || candidate.FactoryOrderID != link.FactoryOrderID || candidate.ChangeSeriesID != link.ChangeSeriesID || identity.PlanDigest != link.PlanDigest || identity.SubjectDigest != link.SubjectDigest || identity.ObligationID != link.ObligationID {
				continue
			}
			if _, matches := phaseGateTLC51EventMatchesLink(candidate, link); matches {
				linkedEventFound = true
			}
			if candidate.EventOrdinal >= projection.LatestEventOrdinal {
				projection.LatestEventOrdinal = candidate.EventOrdinal
				projection.LatestEventType = candidate.EventType
				if candidate.EventType == "factory.tlc51.obligation.terminal" && identity.Outcome != "" {
					projection.ReportedObligationOutcome = identity.Outcome
				}
			}
		}
		if !linkedEventFound {
			projection.Quarantined = true
			projection.HumanInterventionRequired = true
			projection.Reason = "referenced EventGraph/Work twin is missing or mismatched"
		}
		result = append(result, projection)
	}
	return result, nil
}
