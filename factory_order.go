package work

import (
	"fmt"
	"strings"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

// OrderKind selects the terminal action and authority path for an order. The
// FactoryOrder abstraction is general — NOT all orders are software. Slice 1
// implements OrderSoftwarePR end-to-end; the other kinds are defined now so the
// type generalizes (council/governance and research orders are later slices).
type OrderKind string

const (
	// OrderSoftwarePR terminates in an Epic 11 draft PR (Slice 1 implements this).
	OrderSoftwarePR OrderKind = "software_pr"
	// OrderGovernanceDeliberation routes to the council/guardian flow and emits a
	// governance artifact / decision record (a human injects a topic for the
	// Civilization to ponder/debate/council). Terminal action defined later.
	OrderGovernanceDeliberation OrderKind = "governance_deliberation"
	// OrderResearch terminates in a research-report artifact. Terminal action defined later.
	OrderResearch OrderKind = "research"
)

// FactoryOrder is the order request that enters the civilization as a Work task.
// It is a plain input value (distinct from the eventgraph graph record
// v39.FactoryOrder); SeedFactoryOrder maps it onto a readiness-gated task. The
// terminal action is selected by Kind (Slice 1 wires only OrderSoftwarePR).
//
// Required v3.9 linkage fields:
//   - ID must carry the "fo_" prefix (validated by the store).
//   - RequirementIDs, if empty, defaults to ["req_<id-suffix>"].
//   - AcceptanceCriterionIDs, if empty, defaults to ["ac_<id-suffix>"].
//   - Cell, if empty, defaults to "implementation".
type FactoryOrder struct {
	Kind                   OrderKind // defaults to OrderSoftwarePR
	ID                     string
	Title                  string
	Intent                 string
	Cell                   string // v3.9 cell; defaults to "implementation"
	RiskClass              string // low|medium|high|critical; defaults to "low"
	DefinitionOfDone       string
	AcceptanceCriteria     string
	TestPlan               string
	RequirementIDs         []string // v3.9 req_ IDs; derived from ID if empty
	AcceptanceCriterionIDs []string // v3.9 ac_ IDs; derived from ID if empty
	ExpectedOutputs        []string
}

// idSuffix strips the "fo_" prefix (or any prefix before the first underscore)
// and returns the remaining suffix for synthesizing sibling record IDs.
func idSuffix(id string) string {
	if idx := strings.IndexByte(id, '_'); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

// SeedFactoryOrder creates the order's seed task and writes the three required
// readiness gate artifacts so the Planner's contract is satisfied up front and
// the task is assignable to the Implementer. Coordination thereafter is via the
// civic roles on the shared graph.
func SeedFactoryOrder(ts *TaskStore, source types.ActorID, order FactoryOrder, causes []types.EventID, convID types.ConversationID) (Task, error) {
	// Readiness checks gate-label presence, not body content, so a blank gate
	// would let an order go ready with no contract. Reject empty gates up front.
	if strings.TrimSpace(order.DefinitionOfDone) == "" {
		return Task{}, fmt.Errorf("factory order %q: definition_of_done is required", order.ID)
	}
	if strings.TrimSpace(order.AcceptanceCriteria) == "" {
		return Task{}, fmt.Errorf("factory order %q: acceptance_criteria is required", order.ID)
	}
	if strings.TrimSpace(order.TestPlan) == "" {
		return Task{}, fmt.Errorf("factory order %q: test_plan is required", order.ID)
	}

	risk := order.RiskClass
	if risk == "" {
		risk = "low"
	}
	kind := order.Kind
	if kind == "" {
		kind = OrderSoftwarePR
	}
	cell := order.Cell
	if cell == "" {
		cell = "implementation"
	}

	// Synthesize v3.9 sibling IDs from the order ID suffix when callers omit them.
	// This keeps FactoryOrder lean: callers only need to set ID and domain fields.
	suffix := idSuffix(order.ID)
	reqIDs := order.RequirementIDs
	if len(reqIDs) == 0 {
		reqIDs = []string{"req_" + suffix}
	}
	acIDs := order.AcceptanceCriterionIDs
	if len(acIDs) == 0 {
		acIDs = []string{"ac_" + suffix}
	}

	task, err := ts.CreateV39(source, TaskCreateOptions{
		Title:                  order.Title,
		Description:            order.Intent,
		FactoryOrderID:         order.ID,
		RequirementIDs:         reqIDs,
		AcceptanceCriterionIDs: acIDs,
		Cell:                   cell,
		RiskClass:              risk,
		ExpectedOutputs:        order.ExpectedOutputs,
	}, causes, convID)
	if err != nil {
		return Task{}, err
	}
	artifactCauses := append(append([]types.EventID(nil), causes...), task.ID)
	// The three readiness gate artifacts (kind-agnostic), plus a queryable
	// order_kind marker so the terminal-action selector can route by kind.
	gates := []struct{ label, mime, body string }{
		{"order_kind", "text/plain", string(kind)},
		{GateDefinitionOfDone, "text/markdown", order.DefinitionOfDone},
		{GateAcceptanceCriteria, "text/markdown", order.AcceptanceCriteria},
		{GateTestPlan, "text/markdown", order.TestPlan},
	}
	for _, g := range gates {
		if err := ts.AddArtifact(source, task.ID, g.label, g.mime, g.body, artifactCauses, convID); err != nil {
			return Task{}, err
		}
	}
	return task, nil
}
