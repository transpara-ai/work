package work

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

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
	ModelOverrides         []FactoryOrderModelOverride
}

// FactoryOrderModelOverride is structured, durable model-selection policy for
// a FactoryOrder. Hive validates these fields against modelconfig before
// seeding an order; Work records them without treating markdown intent as policy.
type FactoryOrderModelOverride struct {
	Role                 string   `json:"role"`
	Model                string   `json:"model,omitempty"`
	Provider             string   `json:"provider,omitempty"`
	Profile              string   `json:"profile,omitempty"`
	RequestedAuthMode    string   `json:"requested_auth_mode,omitempty"`
	PreferredTier        string   `json:"preferred_tier,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	MaxCostPerCallUSD    *float64 `json:"max_cost_per_call_usd,omitempty"`
	ResolvedModel        string   `json:"resolved_model,omitempty"`
	ResolvedProvider     string   `json:"resolved_provider,omitempty"`
	AuthMode             string   `json:"auth_mode,omitempty"`
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
	// Gate bodies are OPTIONAL at seed: the planner attaches any that are absent,
	// and Readiness — not the seed — enforces that each required gate has a
	// non-empty body before the task can be assigned. So empty gates are not
	// rejected here; the empty ones are simply not written (see the gates loop).
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
	modelOverrideBody, err := factoryOrderModelOverridesArtifactBody(order.ModelOverrides)
	if err != nil {
		return Task{}, err
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
	if modelOverrideBody != "" {
		gates = append(gates, struct{ label, mime, body string }{
			FactoryOrderModelOverridesArtifactLabel,
			"application/json",
			modelOverrideBody,
		})
	}
	for _, g := range gates {
		// A required gate with no body is left unwritten — the planner attaches it
		// later, and Readiness keeps the task not-ready until a non-empty body
		// exists. (order_kind is not a readiness gate, so it is always written.)
		if isRequiredGateLabel(g.label) && strings.TrimSpace(g.body) == "" {
			continue
		}
		if err := ts.AddArtifact(source, task.ID, g.label, g.mime, g.body, artifactCauses, convID); err != nil {
			return Task{}, err
		}
	}
	return task, nil
}

func factoryOrderModelOverridesArtifactBody(overrides []FactoryOrderModelOverride) (string, error) {
	normalized, err := normalizeFactoryOrderModelOverrides(overrides)
	if err != nil {
		return "", err
	}
	if len(normalized) == 0 {
		return "", nil
	}
	body := struct {
		ModelOverrides []FactoryOrderModelOverride `json:"model_overrides"`
	}{ModelOverrides: normalized}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal factory order model overrides: %w", err)
	}
	return string(encoded), nil
}

func normalizeFactoryOrderModelOverrides(overrides []FactoryOrderModelOverride) ([]FactoryOrderModelOverride, error) {
	if len(overrides) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(overrides))
	out := make([]FactoryOrderModelOverride, 0, len(overrides))
	for i, override := range overrides {
		normalized := FactoryOrderModelOverride{
			Role:              strings.TrimSpace(override.Role),
			Model:             strings.TrimSpace(override.Model),
			Provider:          strings.TrimSpace(override.Provider),
			Profile:           strings.TrimSpace(override.Profile),
			RequestedAuthMode: strings.TrimSpace(override.RequestedAuthMode),
			PreferredTier:     strings.TrimSpace(override.PreferredTier),
			ResolvedModel:     strings.TrimSpace(override.ResolvedModel),
			ResolvedProvider:  strings.TrimSpace(override.ResolvedProvider),
			AuthMode:          strings.TrimSpace(override.AuthMode),
		}
		if normalized.Role == "" {
			return nil, fmt.Errorf("model_overrides[%d].role is required", i)
		}
		if hasControlRune(normalized.Role) || hasControlRune(normalized.Model) || hasControlRune(normalized.Provider) ||
			hasControlRune(normalized.Profile) || hasControlRune(normalized.RequestedAuthMode) || hasControlRune(normalized.PreferredTier) ||
			hasControlRune(normalized.ResolvedModel) || hasControlRune(normalized.ResolvedProvider) || hasControlRune(normalized.AuthMode) {
			return nil, fmt.Errorf("model_overrides[%d] contains control characters", i)
		}
		if _, duplicate := seen[normalized.Role]; duplicate {
			return nil, fmt.Errorf("model_overrides[%d].role %q is duplicated", i, normalized.Role)
		}
		seen[normalized.Role] = struct{}{}
		if !validFactoryOrderAuthMode(normalized.RequestedAuthMode) {
			return nil, fmt.Errorf("model_overrides[%d].requested_auth_mode must be subscription, api-key, or local", i)
		}
		if !validFactoryOrderAuthMode(normalized.AuthMode) {
			return nil, fmt.Errorf("model_overrides[%d].auth_mode must be subscription, api-key, or local", i)
		}
		if override.MaxCostPerCallUSD != nil {
			if *override.MaxCostPerCallUSD < 0 {
				return nil, fmt.Errorf("model_overrides[%d].max_cost_per_call_usd must be zero or greater", i)
			}
			maxCost := *override.MaxCostPerCallUSD
			normalized.MaxCostPerCallUSD = &maxCost
		}
		normalized.RequiredCapabilities = normalizeFactoryOrderCapabilities(override.RequiredCapabilities)
		if len(normalized.RequiredCapabilities) != len(override.RequiredCapabilities) {
			return nil, fmt.Errorf("model_overrides[%d].required_capabilities contains empty values", i)
		}
		for _, cap := range normalized.RequiredCapabilities {
			if hasControlRune(cap) {
				return nil, fmt.Errorf("model_overrides[%d].required_capabilities contains control characters", i)
			}
		}
		hasOverride := normalized.Model != "" || normalized.Provider != "" || normalized.Profile != "" ||
			normalized.RequestedAuthMode != "" || normalized.PreferredTier != "" ||
			len(normalized.RequiredCapabilities) > 0 || normalized.MaxCostPerCallUSD != nil
		if !hasOverride {
			return nil, fmt.Errorf("model_overrides[%d] must set model, profile, provider, requested_auth_mode, preferred_tier, required_capabilities, or max_cost_per_call_usd", i)
		}
		out = append(out, normalized)
	}
	return out, nil
}

func validFactoryOrderAuthMode(value string) bool {
	switch value {
	case "", "subscription", "api-key", "local":
		return true
	default:
		return false
	}
}

func normalizeFactoryOrderCapabilities(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return out
		}
		out = append(out, trimmed)
	}
	return out
}

func hasControlRune(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
