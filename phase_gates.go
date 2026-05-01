package work

import (
	"fmt"
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
)

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
