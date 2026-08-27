package work

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
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
	SourceIssueRecords     []FactoryOrderSourceIssueRecord
	ModelOverrides         []FactoryOrderModelOverride
	TLC51EventArtifacts    []FactoryOrderTLC51EventArtifact
}

const (
	// FactoryOrderTLC51EventArtifactLabel identifies Work twins of immutable
	// EventGraph factory-tlc51/v1 history entries. The artifact is linkage and
	// replay evidence only; it grants no authority and cannot replace its
	// EventGraph source.
	FactoryOrderTLC51EventArtifactLabel = "factory_tlc51_event"
	// FactoryOrderTLC51EventArtifactMediaType is closed so the exact JSON
	// payload and its digest cannot be confused with markdown intent.
	FactoryOrderTLC51EventArtifactMediaType = "application/vnd.transpara.factory-tlc51-event+json"
	// FactoryTLC51ProtocolVersion is the immutable Hive/EventGraph protocol
	// identity implemented by TLC 5.1.
	FactoryTLC51ProtocolVersion = "factory-tlc51/v1"
)

// FactoryOrderTLC51EventArtifact is the Work-side twin of an EventGraph TLC
// 5.1 history entry. Payload is retained byte-for-byte and PayloadSHA256 is
// computed over those exact bytes. The common identity inside Payload must
// match every outer field. EventGraph remains the source of truth.
type FactoryOrderTLC51EventArtifact struct {
	FactoryOrderID string          `json:"factory_order_id"`
	ChangeSeriesID string          `json:"change_series_id"`
	EventOrdinal   uint64          `json:"event_ordinal"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	PayloadSHA256  string          `json:"payload_sha256"`
}

// FactoryOrderTLC51LinkProjection is a fail-closed replay of Work's TLC 5.1
// event artifacts. EventGraphVerified is deliberately always false here:
// cross-store reconciliation must establish that separately from EventGraph.
type FactoryOrderTLC51LinkProjection struct {
	EventArtifacts            []FactoryOrderTLC51EventArtifact `json:"event_artifacts"`
	Quarantined               bool                             `json:"quarantined"`
	HumanInterventionRequired bool                             `json:"human_intervention_required"`
	ConflictKeys              []string                         `json:"conflict_keys,omitempty"`
	EventGraphVerified        bool                             `json:"eventgraph_verified"`
	AuthorityGranted          bool                             `json:"authority_granted"`
}

type factoryOrderTLC51PayloadIdentity struct {
	ProtocolVersion string `json:"protocol_version"`
	FactoryOrderID  string `json:"factory_order_id"`
	ChangeSeriesID  string `json:"change_series_id"`
	PlanDigest      string `json:"plan_digest"`
	SubjectDigest   string `json:"subject_digest"`
	EventOrdinal    uint64 `json:"event_ordinal"`
	AttemptOrdinal  uint32 `json:"attempt_ordinal"`
	ObligationID    string `json:"obligation_id,omitempty"`
	Outcome         string `json:"outcome,omitempty"`
}

var factoryOrderTLC51EventTypes = map[string]bool{
	"factory.tlc51.plan.recorded":        false,
	"factory.tlc51.plan.superseded":      false,
	"factory.tlc51.obligation.ready":     true,
	"factory.tlc51.obligation.claimed":   true,
	"factory.tlc51.obligation.running":   true,
	"factory.tlc51.obligation.terminal":  true,
	"factory.tlc51.evidence.linked":      true,
	"factory.tlc51.decision.recorded":    false,
	"factory.tlc51.decision.invalidated": false,
	"factory.tlc51.effect.proposed":      true,
	"factory.tlc51.effect.observed":      true,
	"factory.tlc51.effect.reconciled":    true,
	"factory.tlc51.effect.terminal":      true,
	"factory.tlc51.human.requested":      false,
	"factory.tlc51.human.resolved":       false,
	"factory.tlc51.cutover.recorded":     false,
}

// FactoryOrderSourceIssueRecord is caller-supplied GitHub issue source
// evidence. Work normalizes it into artifacts and projections; it never fetches
// GitHub itself and never treats issue text as authority.
type FactoryOrderSourceIssueRecord struct {
	Repo               string   `json:"repo"`
	Number             int      `json:"number"`
	URL                string   `json:"url,omitempty"`
	Title              string   `json:"title"`
	Goal               string   `json:"goal,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Assumptions        []string `json:"assumptions,omitempty"`
	Ambiguities        []string `json:"ambiguities,omitempty"`
	RiskNotes          []string `json:"risk_notes,omitempty"`
	Labels             []string `json:"labels,omitempty"`
	SourceRefs         []string `json:"source_refs,omitempty"`
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
	sourceIssuesBody, err := factoryOrderSourceIssuesArtifactBody(order.SourceIssueRecords)
	if err != nil {
		return Task{}, err
	}
	tlc51Bodies, err := normalizeFactoryOrderTLC51EventArtifactBodies(order.ID, order.TLC51EventArtifacts)
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
	if sourceIssuesBody != "" {
		gates = append(gates, struct{ label, mime, body string }{
			FactoryOrderSourceIssuesArtifactLabel,
			"application/json",
			sourceIssuesBody,
		})
	}
	for _, body := range tlc51Bodies {
		gates = append(gates, struct{ label, mime, body string }{
			FactoryOrderTLC51EventArtifactLabel,
			FactoryOrderTLC51EventArtifactMediaType,
			body,
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

// ValidateFactoryOrderTLC51EventArtifact validates Work's exact EventGraph
// twin. It intentionally validates only the cross-store identity contract;
// the EventGraph registry validates the event-kind-specific payload and remains
// authoritative for whether the source entry exists.
func ValidateFactoryOrderTLC51EventArtifact(value FactoryOrderTLC51EventArtifact, expectedFactoryOrderID string) error {
	_, err := validateFactoryOrderTLC51EventArtifact(value, expectedFactoryOrderID)
	return err
}

func validateFactoryOrderTLC51EventArtifact(value FactoryOrderTLC51EventArtifact, expectedFactoryOrderID string) (factoryOrderTLC51PayloadIdentity, error) {
	attemptRequired, known := factoryOrderTLC51EventTypes[value.EventType]
	if !known {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("event_type %q is not in the closed factory-tlc51/v1 set", value.EventType)
	}
	if value.FactoryOrderID == "" || value.ChangeSeriesID == "" || value.EventOrdinal == 0 {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("factory_order_id, change_series_id, and event_ordinal are required")
	}
	if expectedFactoryOrderID != "" && value.FactoryOrderID != expectedFactoryOrderID {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("factory_order_id %q does not match task order %q", value.FactoryOrderID, expectedFactoryOrderID)
	}
	if hasControlRune(value.FactoryOrderID) || hasControlRune(value.ChangeSeriesID) || hasControlRune(value.EventType) {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("TLC 5.1 event identity contains control characters")
	}
	if len(value.Payload) == 0 || !json.Valid(value.Payload) {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload must be one valid JSON object")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, value.Payload); err != nil || !bytes.Equal(compact.Bytes(), value.Payload) {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload must preserve compact EventGraph JSON bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(value.Payload))
	decoder.UseNumber()
	var identity factoryOrderTLC51PayloadIdentity
	if err := decoder.Decode(&identity); err != nil {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("decode payload identity: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload must contain exactly one JSON value")
	}
	if identity.ProtocolVersion != FactoryTLC51ProtocolVersion {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload protocol_version must be %q", FactoryTLC51ProtocolVersion)
	}
	if identity.FactoryOrderID != value.FactoryOrderID || identity.ChangeSeriesID != value.ChangeSeriesID || identity.EventOrdinal != value.EventOrdinal {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload and artifact event identities do not match")
	}
	if !validFactoryOrderTLC51SHA(identity.PlanDigest) || !validFactoryOrderTLC51SHA(identity.SubjectDigest) {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload plan_digest and subject_digest must be lowercase SHA-256")
	}
	if attemptRequired && identity.AttemptOrdinal == 0 {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload attempt_ordinal must be positive for %s", value.EventType)
	}
	if !attemptRequired && identity.AttemptOrdinal != 0 {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload attempt_ordinal must be zero for %s", value.EventType)
	}
	if !validFactoryOrderTLC51SHA(value.PayloadSHA256) {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload_sha256 must be lowercase SHA-256")
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(value.Payload))
	if actual != value.PayloadSHA256 {
		return factoryOrderTLC51PayloadIdentity{}, fmt.Errorf("payload_sha256 does not match exact payload bytes")
	}
	return identity, nil
}

func validFactoryOrderTLC51SHA(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

// FactoryOrderTLC51EventArtifactBody returns the closed, compact JSON body
// used by Work. Compact encoding preserves the already-compact EventGraph
// payload bytes whose digest is carried by the artifact.
func FactoryOrderTLC51EventArtifactBody(value FactoryOrderTLC51EventArtifact) (string, error) {
	if err := ValidateFactoryOrderTLC51EventArtifact(value, ""); err != nil {
		return "", err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal TLC 5.1 event artifact: %w", err)
	}
	return string(body), nil
}

func parseFactoryOrderTLC51EventArtifactBody(body, expectedFactoryOrderID string) (FactoryOrderTLC51EventArtifact, error) {
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	var value FactoryOrderTLC51EventArtifact
	if err := decoder.Decode(&value); err != nil {
		return FactoryOrderTLC51EventArtifact{}, fmt.Errorf("decode TLC 5.1 event artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return FactoryOrderTLC51EventArtifact{}, fmt.Errorf("TLC 5.1 event artifact must contain exactly one JSON value")
	}
	if err := ValidateFactoryOrderTLC51EventArtifact(value, expectedFactoryOrderID); err != nil {
		return FactoryOrderTLC51EventArtifact{}, err
	}
	value.Payload = append(json.RawMessage(nil), value.Payload...)
	return value, nil
}

func normalizeFactoryOrderTLC51EventArtifactBodies(expectedFactoryOrderID string, values []FactoryOrderTLC51EventArtifact) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	byKey := make(map[string]FactoryOrderTLC51EventArtifact, len(values))
	for index, value := range values {
		if _, err := validateFactoryOrderTLC51EventArtifact(value, expectedFactoryOrderID); err != nil {
			return nil, fmt.Errorf("tlc51_event_artifacts[%d]: %w", index, err)
		}
		key := factoryOrderTLC51ArtifactKey(value)
		if existing, ok := byKey[key]; ok {
			if !factoryOrderTLC51ArtifactsEqual(existing, value) {
				return nil, fmt.Errorf("tlc51_event_artifacts[%d]: conflicting Work twin for %s", index, key)
			}
			continue
		}
		value.Payload = append(json.RawMessage(nil), value.Payload...)
		byKey[key] = value
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	bodies := make([]string, 0, len(keys))
	for _, key := range keys {
		body, err := FactoryOrderTLC51EventArtifactBody(byKey[key])
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, body)
	}
	return bodies, nil
}

func factoryOrderTLC51ArtifactKey(value FactoryOrderTLC51EventArtifact) string {
	return fmt.Sprintf("%s/%s/%020d", value.FactoryOrderID, value.ChangeSeriesID, value.EventOrdinal)
}

func factoryOrderTLC51ArtifactsEqual(left, right FactoryOrderTLC51EventArtifact) bool {
	return left.FactoryOrderID == right.FactoryOrderID &&
		left.ChangeSeriesID == right.ChangeSeriesID &&
		left.EventOrdinal == right.EventOrdinal &&
		left.EventType == right.EventType &&
		left.PayloadSHA256 == right.PayloadSHA256 &&
		bytes.Equal(left.Payload, right.Payload)
}

// AttachFactoryOrderTLC51EventArtifact appends one validated Work twin. Exact
// repeats are idempotent; a different twin at the same history ordinal is
// rejected rather than overwriting or inventing EventGraph state.
func (ts *TaskStore) AttachFactoryOrderTLC51EventArtifact(
	source types.ActorID,
	taskID types.EventID,
	value FactoryOrderTLC51EventArtifact,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	task, err := ts.ProjectTask(taskID)
	if err != nil {
		return fmt.Errorf("project FactoryOrder task: %w", err)
	}
	if _, err := validateFactoryOrderTLC51EventArtifact(value, task.FactoryOrderID); err != nil {
		return err
	}
	projection, err := ts.ProjectFactoryOrderTLC51EventArtifacts(taskID)
	if err != nil {
		return err
	}
	if projection.Quarantined {
		return fmt.Errorf("TLC 5.1 Work linkage is quarantined; Human intervention required")
	}
	key := factoryOrderTLC51ArtifactKey(value)
	for _, existing := range projection.EventArtifacts {
		if factoryOrderTLC51ArtifactKey(existing) != key {
			continue
		}
		if factoryOrderTLC51ArtifactsEqual(existing, value) {
			return nil
		}
		return fmt.Errorf("conflicting Work twin for %s", key)
	}
	body, err := FactoryOrderTLC51EventArtifactBody(value)
	if err != nil {
		return err
	}
	return ts.AddArtifact(source, taskID, FactoryOrderTLC51EventArtifactLabel, FactoryOrderTLC51EventArtifactMediaType, body, causes, convID)
}

// ProjectFactoryOrderTLC51EventArtifacts replays all TLC 5.1 Work twins for a
// task. Conflicting valid twins are retained as evidence and quarantine the
// projection; no last-write-wins rule is allowed.
func (ts *TaskStore) ProjectFactoryOrderTLC51EventArtifacts(taskID types.EventID) (FactoryOrderTLC51LinkProjection, error) {
	task, err := ts.ProjectTask(taskID)
	if err != nil {
		return FactoryOrderTLC51LinkProjection{}, err
	}
	artifacts, err := ts.ListArtifacts(taskID)
	if err != nil {
		return FactoryOrderTLC51LinkProjection{}, err
	}
	projection := FactoryOrderTLC51LinkProjection{}
	byKey := map[string]FactoryOrderTLC51EventArtifact{}
	conflicts := map[string]struct{}{}
	for _, artifact := range artifacts {
		if artifact.Label != FactoryOrderTLC51EventArtifactLabel {
			continue
		}
		if artifact.MediaType != FactoryOrderTLC51EventArtifactMediaType {
			return FactoryOrderTLC51LinkProjection{}, fmt.Errorf("%s artifact must use media type %s", FactoryOrderTLC51EventArtifactLabel, FactoryOrderTLC51EventArtifactMediaType)
		}
		value, err := parseFactoryOrderTLC51EventArtifactBody(artifact.Body, task.FactoryOrderID)
		if err != nil {
			return FactoryOrderTLC51LinkProjection{}, fmt.Errorf("invalid %s artifact %s: %w", FactoryOrderTLC51EventArtifactLabel, artifact.ID.Value(), err)
		}
		key := factoryOrderTLC51ArtifactKey(value)
		if existing, ok := byKey[key]; ok {
			if !factoryOrderTLC51ArtifactsEqual(existing, value) {
				conflicts[key] = struct{}{}
			}
			continue
		}
		byKey[key] = value
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		projection.EventArtifacts = append(projection.EventArtifacts, byKey[key])
	}
	for key := range conflicts {
		projection.ConflictKeys = append(projection.ConflictKeys, key)
	}
	sort.Strings(projection.ConflictKeys)
	projection.Quarantined = len(projection.ConflictKeys) > 0
	projection.HumanInterventionRequired = projection.Quarantined
	// Work artifacts never grant protected-action authority, and only a
	// cross-store EventGraph reconciliation can set source verification.
	projection.EventGraphVerified = false
	projection.AuthorityGranted = false
	return projection, nil
}

func factoryOrderSourceIssuesArtifactBody(records []FactoryOrderSourceIssueRecord) (string, error) {
	normalized, err := normalizeFactoryOrderSourceIssueRecords(records, "source_issue_records")
	if err != nil {
		return "", err
	}
	if len(normalized) == 0 {
		return "", nil
	}
	body := struct {
		SourceIssueRecords  []FactoryOrderSourceIssueRecord `json:"source_issue_records"`
		AuthorityExclusions []string                        `json:"authority_exclusions"`
	}{
		SourceIssueRecords: normalized,
		AuthorityExclusions: []string{
			"github_issue_records_are_source_intent_only",
			"no_protected_action_authority",
			"no_runtime_execution",
			"no_eventgraph_write",
			"no_hive_write_action_or_authority_api",
			"no_deployment",
			"no_test_001_green",
			"no_docs_172_closure",
			"no_autonomy_increase",
			"no_value_allocation",
			"no_residual_risk_closure",
			"no_branch_pr_or_merge_authority",
		},
	}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal factory order source issues: %w", err)
	}
	return string(encoded), nil
}

func normalizeFactoryOrderSourceIssueRecords(records []FactoryOrderSourceIssueRecord, field string) ([]FactoryOrderSourceIssueRecord, error) {
	if len(records) == 0 {
		return nil, nil
	}
	out := make([]FactoryOrderSourceIssueRecord, 0, len(records))
	for i, record := range records {
		normalized := FactoryOrderSourceIssueRecord{
			Repo:               strings.TrimSpace(record.Repo),
			Number:             record.Number,
			URL:                strings.TrimSpace(record.URL),
			Title:              strings.TrimSpace(record.Title),
			Goal:               strings.TrimSpace(record.Goal),
			AcceptanceCriteria: cloneStrings(record.AcceptanceCriteria),
			Assumptions:        cloneStrings(record.Assumptions),
			Ambiguities:        cloneStrings(record.Ambiguities),
			RiskNotes:          cloneStrings(record.RiskNotes),
			Labels:             cloneStrings(record.Labels),
			SourceRefs:         cloneStrings(record.SourceRefs),
		}
		if normalized.Repo == "" {
			return nil, fmt.Errorf("%s[%d].repo is required", field, i)
		}
		if normalized.Number <= 0 {
			return nil, fmt.Errorf("%s[%d].number must be positive", field, i)
		}
		if normalized.Title == "" {
			return nil, fmt.Errorf("%s[%d].title is required", field, i)
		}
		if normalized.Goal == "" {
			normalized.Goal = normalized.Title
		}
		if len(normalized.SourceRefs) == 0 {
			normalized.SourceRefs = []string{issueSourceRef(normalized)}
		}
		if factoryOrderSourceIssueRecordHasControlRune(normalized) {
			return nil, fmt.Errorf("%s[%d] contains control characters", field, i)
		}
		out = append(out, normalized)
	}
	return out, nil
}

func factoryOrderSourceIssueRecordHasControlRune(record FactoryOrderSourceIssueRecord) bool {
	if hasControlRune(record.Repo) || hasControlRune(record.URL) || hasControlRune(record.Title) || hasControlRune(record.Goal) {
		return true
	}
	for _, values := range [][]string{
		record.AcceptanceCriteria,
		record.Assumptions,
		record.Ambiguities,
		record.RiskNotes,
		record.Labels,
		record.SourceRefs,
	} {
		for _, value := range values {
			if hasControlRune(value) {
				return true
			}
		}
	}
	return false
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
		roleKey := strings.ToLower(normalized.Role)
		if _, duplicate := seen[roleKey]; duplicate {
			return nil, fmt.Errorf("model_overrides[%d].role %q is duplicated", i, normalized.Role)
		}
		seen[roleKey] = struct{}{}
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

func (ts *TaskStore) projectFactoryOrderModelOverrides(taskID types.EventID) ([]FactoryOrderModelOverride, error) {
	artifacts, err := ts.ListArtifacts(taskID)
	if err != nil {
		return nil, err
	}
	var body string
	for _, artifact := range artifacts {
		if artifact.Label == FactoryOrderModelOverridesArtifactLabel {
			body = artifact.Body
		}
	}
	if strings.TrimSpace(body) == "" {
		return nil, nil
	}
	var decoded struct {
		ModelOverrides []FactoryOrderModelOverride `json:"model_overrides"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		return nil, fmt.Errorf("parse factory order model overrides: %w", err)
	}
	normalized, err := normalizeFactoryOrderModelOverrides(decoded.ModelOverrides)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func (ts *TaskStore) projectFactoryOrderSourceIssueRecords(taskID types.EventID) ([]FactoryOrderSourceIssueRecord, error) {
	artifacts, err := ts.ListArtifacts(taskID)
	if err != nil {
		return nil, err
	}
	var body string
	for _, artifact := range artifacts {
		if artifact.Label == FactoryOrderSourceIssuesArtifactLabel {
			body = artifact.Body
		}
	}
	if strings.TrimSpace(body) == "" {
		return nil, nil
	}
	var decoded struct {
		SourceIssueRecords []FactoryOrderSourceIssueRecord `json:"source_issue_records"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		return nil, fmt.Errorf("parse factory order source issues: %w", err)
	}
	normalized, err := normalizeFactoryOrderSourceIssueRecords(decoded.SourceIssueRecords, "source_issue_records")
	if err != nil {
		return nil, err
	}
	return normalized, nil
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
