package work

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

const (
	// IncidentFollowUpSchemaRef is the operation contract Work accepts
	// for incident execution follow-up tasks.
	IncidentFollowUpSchemaRef = "operation/docs/operations/work-incident-follow-up-schema.md"
	// IncidentFollowUpSchemaVersion tracks the first accepted pre-live contract.
	IncidentFollowUpSchemaVersion = "2026-06-16"
	// IncidentFollowUpArtifactLabel labels work.task.artifact events carrying the
	// incident follow-up contract payload.
	IncidentFollowUpArtifactLabel = "incident_follow_up"
	// IncidentFollowUpMediaType is the artifact media type for the contract payload.
	IncidentFollowUpMediaType = "application/json"
)

const legacyIncidentFollowUpSchemaRef = "civilization-operation/docs/operations/work-incident-follow-up-schema.md"

// IncidentFollowUpTaskType is the primary type for a routed incident follow-up.
type IncidentFollowUpTaskType string

const (
	IncidentFollowUpEvidence      IncidentFollowUpTaskType = "EVIDENCE"
	IncidentFollowUpContainment   IncidentFollowUpTaskType = "CONTAINMENT"
	IncidentFollowUpCorrection    IncidentFollowUpTaskType = "CORRECTION"
	IncidentFollowUpAuthorization IncidentFollowUpTaskType = "AUTHORIZATION"
	IncidentFollowUpValidation    IncidentFollowUpTaskType = "VALIDATION"
	IncidentFollowUpReview        IncidentFollowUpTaskType = "REVIEW"
	IncidentFollowUpCommunication IncidentFollowUpTaskType = "COMMUNICATION"
	IncidentFollowUpCleanup       IncidentFollowUpTaskType = "CLEANUP"
)

// IncidentFollowUpStatus is the contract status for a routed incident follow-up.
type IncidentFollowUpStatus string

const (
	IncidentFollowUpProposed   IncidentFollowUpStatus = "PROPOSED"
	IncidentFollowUpReady      IncidentFollowUpStatus = "READY"
	IncidentFollowUpInProgress IncidentFollowUpStatus = "IN_PROGRESS"
	IncidentFollowUpBlocked    IncidentFollowUpStatus = "BLOCKED"
	IncidentFollowUpDone       IncidentFollowUpStatus = "DONE"
	IncidentFollowUpDeclined   IncidentFollowUpStatus = "DECLINED"
	IncidentFollowUpSuperseded IncidentFollowUpStatus = "SUPERSEDED"
)

// IncidentFollowUp is Work's accepted shape for
// operation/docs/operations/work-incident-follow-up-schema.md.
// Work stores this as an immutable task artifact; the incident record remains
// the cross-repository source of truth.
type IncidentFollowUp struct {
	TaskID                string                   `json:"task_id"`
	IncidentID            string                   `json:"incident_id"`
	IncidentRecord        string                   `json:"incident_record"`
	TaskType              IncidentFollowUpTaskType `json:"task_type"`
	OwningRepo            string                   `json:"owning_repo"`
	RequestedBy           string                   `json:"requested_by"`
	AssignedTo            string                   `json:"assigned_to"`
	Status                IncidentFollowUpStatus   `json:"status"`
	Severity              string                   `json:"severity"`
	Summary               string                   `json:"summary"`
	RequiredEvidence      []string                 `json:"required_evidence"`
	AuthorizationRequired bool                     `json:"authorization_required"`
	AuthorizationEvidence string                   `json:"authorization_evidence"`
	ValidationRequired    bool                     `json:"validation_required"`
	ValidationEvidence    []string                 `json:"validation_evidence"`
	BlockingDependencies  []string                 `json:"blocking_dependencies"`
	AcceptanceCriteria    []string                 `json:"acceptance_criteria"`
	ClosureLink           string                   `json:"closure_link"`
}

// IncidentFollowUpArtifact is a parsed incident follow-up artifact event.
type IncidentFollowUpArtifact struct {
	ID        types.EventID
	TaskID    types.EventID
	FollowUp  IncidentFollowUp
	CreatedBy types.ActorID
	Timestamp time.Time
}

type incidentFollowUpEnvelope struct {
	Schema        string           `json:"schema"`
	SchemaVersion string           `json:"schema_version"`
	FollowUp      IncidentFollowUp `json:"follow_up"`
}

// NormalizeIncidentFollowUp trims and validates an incident follow-up payload.
func NormalizeIncidentFollowUp(followUp IncidentFollowUp) (IncidentFollowUp, error) {
	normalized := IncidentFollowUp{
		TaskID:                strings.TrimSpace(followUp.TaskID),
		IncidentID:            strings.TrimSpace(followUp.IncidentID),
		IncidentRecord:        strings.TrimSpace(followUp.IncidentRecord),
		TaskType:              IncidentFollowUpTaskType(strings.TrimSpace(string(followUp.TaskType))),
		OwningRepo:            strings.TrimSpace(followUp.OwningRepo),
		RequestedBy:           strings.TrimSpace(followUp.RequestedBy),
		AssignedTo:            strings.TrimSpace(followUp.AssignedTo),
		Status:                IncidentFollowUpStatus(strings.TrimSpace(string(followUp.Status))),
		Severity:              strings.TrimSpace(followUp.Severity),
		Summary:               strings.TrimSpace(followUp.Summary),
		AuthorizationRequired: followUp.AuthorizationRequired,
		AuthorizationEvidence: strings.TrimSpace(followUp.AuthorizationEvidence),
		ValidationRequired:    followUp.ValidationRequired,
		ClosureLink:           strings.TrimSpace(followUp.ClosureLink),
	}
	var err error
	normalized.RequiredEvidence, err = normalizeIncidentFollowUpList("required_evidence", followUp.RequiredEvidence, true)
	if err != nil {
		return IncidentFollowUp{}, err
	}
	normalized.ValidationEvidence, err = normalizeIncidentFollowUpList("validation_evidence", followUp.ValidationEvidence, false)
	if err != nil {
		return IncidentFollowUp{}, err
	}
	normalized.BlockingDependencies, err = normalizeIncidentFollowUpList("blocking_dependencies", followUp.BlockingDependencies, false)
	if err != nil {
		return IncidentFollowUp{}, err
	}
	normalized.AcceptanceCriteria, err = normalizeIncidentFollowUpList("acceptance_criteria", followUp.AcceptanceCriteria, true)
	if err != nil {
		return IncidentFollowUp{}, err
	}
	if err := validateIncidentFollowUpScalars(&normalized); err != nil {
		return IncidentFollowUp{}, err
	}
	switch normalized.Status {
	case IncidentFollowUpDone:
		if normalized.ClosureLink == "" {
			return IncidentFollowUp{}, fmt.Errorf("closure_link is required when status is DONE")
		}
		if normalized.AuthorizationRequired && normalized.AuthorizationEvidence == "" {
			return IncidentFollowUp{}, fmt.Errorf("authorization_evidence is required when status is DONE and authorization_required is true")
		}
		if normalized.ValidationRequired && len(normalized.ValidationEvidence) == 0 {
			return IncidentFollowUp{}, fmt.Errorf("validation_evidence is required when status is DONE and validation_required is true")
		}
	case IncidentFollowUpDeclined, IncidentFollowUpSuperseded:
		if normalized.ClosureLink == "" {
			return IncidentFollowUp{}, fmt.Errorf("closure_link is required when status is %s", normalized.Status)
		}
	case IncidentFollowUpBlocked:
		if len(normalized.BlockingDependencies) == 0 {
			return IncidentFollowUp{}, fmt.Errorf("blocking_dependencies is required when status is BLOCKED")
		}
	}
	return normalized, nil
}

// IncidentFollowUpArtifactBody returns the JSON body stored in Work artifacts.
func IncidentFollowUpArtifactBody(followUp IncidentFollowUp) (string, error) {
	normalized, err := NormalizeIncidentFollowUp(followUp)
	if err != nil {
		return "", err
	}
	body := incidentFollowUpEnvelope{
		Schema:        IncidentFollowUpSchemaRef,
		SchemaVersion: IncidentFollowUpSchemaVersion,
		FollowUp:      normalized,
	}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal incident follow-up: %w", err)
	}
	return string(encoded), nil
}

func parseIncidentFollowUpArtifactBody(body string) (IncidentFollowUp, error) {
	var decoded incidentFollowUpEnvelope
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return IncidentFollowUp{}, err
	}
	if decoded.Schema != IncidentFollowUpSchemaRef && decoded.Schema != legacyIncidentFollowUpSchemaRef {
		return IncidentFollowUp{}, fmt.Errorf("schema %q, want %q", decoded.Schema, IncidentFollowUpSchemaRef)
	}
	if decoded.SchemaVersion != IncidentFollowUpSchemaVersion {
		return IncidentFollowUp{}, fmt.Errorf("schema_version %q, want %q", decoded.SchemaVersion, IncidentFollowUpSchemaVersion)
	}
	normalized, err := NormalizeIncidentFollowUp(decoded.FollowUp)
	if err != nil {
		return IncidentFollowUp{}, err
	}
	return normalized, nil
}

// AddIncidentFollowUpArtifact records the accepted incident follow-up contract
// payload on a Work task.
func (ts *TaskStore) AddIncidentFollowUpArtifact(
	source types.ActorID,
	taskID types.EventID,
	followUp IncidentFollowUp,
	causes []types.EventID,
	convID types.ConversationID,
) error {
	body, err := IncidentFollowUpArtifactBody(followUp)
	if err != nil {
		return err
	}
	artifactCauses := append(append([]types.EventID(nil), causes...), taskID)
	return ts.AddArtifact(source, taskID, IncidentFollowUpArtifactLabel, IncidentFollowUpMediaType, body, artifactCauses, convID)
}

// ListIncidentFollowUps parses incident follow-up artifacts for a Work task.
func (ts *TaskStore) ListIncidentFollowUps(taskID types.EventID) ([]IncidentFollowUpArtifact, error) {
	artifacts, err := ts.ListArtifacts(taskID)
	if err != nil {
		return nil, err
	}
	out := make([]IncidentFollowUpArtifact, 0)
	for _, artifact := range artifacts {
		if artifact.Label != IncidentFollowUpArtifactLabel {
			continue
		}
		normalized, err := parseIncidentFollowUpArtifactBody(artifact.Body)
		if err != nil {
			return nil, fmt.Errorf("validate incident follow-up artifact %s: %w", artifact.ID.Value(), err)
		}
		out = append(out, IncidentFollowUpArtifact{
			ID:        artifact.ID,
			TaskID:    artifact.TaskID,
			FollowUp:  normalized,
			CreatedBy: artifact.CreatedBy,
			Timestamp: artifact.Timestamp,
		})
	}
	return out, nil
}

// LatestIncidentFollowUp returns the most recent incident follow-up artifact for
// a Work task.
func (ts *TaskStore) LatestIncidentFollowUp(taskID types.EventID) (IncidentFollowUpArtifact, bool, error) {
	followUps, err := ts.ListIncidentFollowUps(taskID)
	if err != nil {
		return IncidentFollowUpArtifact{}, false, err
	}
	if len(followUps) == 0 {
		return IncidentFollowUpArtifact{}, false, nil
	}
	latest := followUps[0]
	for _, followUp := range followUps[1:] {
		if followUp.Timestamp.After(latest.Timestamp) ||
			(followUp.Timestamp.Equal(latest.Timestamp) && followUp.ID.Value() > latest.ID.Value()) {
			latest = followUp
		}
	}
	return latest, true, nil
}

func validateIncidentFollowUpScalars(followUp *IncidentFollowUp) error {
	required := []struct {
		field string
		value string
	}{
		{field: "task_id", value: followUp.TaskID},
		{field: "incident_id", value: followUp.IncidentID},
		{field: "incident_record", value: followUp.IncidentRecord},
		{field: "task_type", value: string(followUp.TaskType)},
		{field: "owning_repo", value: followUp.OwningRepo},
		{field: "requested_by", value: followUp.RequestedBy},
		{field: "assigned_to", value: followUp.AssignedTo},
		{field: "status", value: string(followUp.Status)},
		{field: "severity", value: followUp.Severity},
		{field: "summary", value: followUp.Summary},
	}
	for _, item := range required {
		if item.value == "" {
			return fmt.Errorf("%s is required", item.field)
		}
		if hasControlRune(item.value) {
			return fmt.Errorf("%s contains control characters", item.field)
		}
	}
	if hasControlRune(followUp.AuthorizationEvidence) || hasControlRune(followUp.ClosureLink) {
		return fmt.Errorf("incident follow-up contains control characters")
	}
	canonicalIncidentRecord, ok := canonicalOperationIncidentRecord(followUp.IncidentRecord)
	if !ok {
		return fmt.Errorf("incident_record must point to operation docs/incidents/")
	}
	followUp.IncidentRecord = canonicalIncidentRecord
	if !validIncidentFollowUpTaskType(followUp.TaskType) {
		return fmt.Errorf("task_type %q is not accepted by the incident follow-up schema", followUp.TaskType)
	}
	if !validIncidentFollowUpStatus(followUp.Status) {
		return fmt.Errorf("status %q is not accepted by the incident follow-up schema", followUp.Status)
	}
	return nil
}

func canonicalOperationIncidentRecord(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		if parsed.Host == "" || parsed.Host != "github.com" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return "", false
		}
		cleaned := strings.Trim(path.Clean(parsed.Path), "/")
		parts := strings.Split(cleaned, "/")
		if len(parts) >= 4 && parts[0] == "transpara-ai" && isOperationRepoSlug(parts[1]) && strings.Contains(cleaned, "/docs/incidents/") {
			parts[1] = "operation"
			parsed.Path = "/" + strings.Join(parts, "/")
			return parsed.String(), true
		}
		return "", false
	}
	cleaned := strings.TrimPrefix(path.Clean(strings.ReplaceAll(trimmed, "\\", "/")), "/")
	parts := strings.Split(cleaned, "/")
	for i := 0; i+2 < len(parts); i++ {
		if isOperationRepoSlug(parts[i]) && parts[i+1] == "docs" && parts[i+2] == "incidents" {
			parts[i] = "operation"
			return strings.Join(parts, "/"), true
		}
	}
	return "", false
}

func isOperationRepoSlug(value string) bool {
	return value == "operation" || value == "civilization-operation"
}

func normalizeIncidentFollowUpList(field string, values []string, requireNonEmpty bool) ([]string, error) {
	out := make([]string, 0, len(values))
	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("%s[%d] is required", field, i)
		}
		if hasControlRune(trimmed) {
			return nil, fmt.Errorf("%s[%d] contains control characters", field, i)
		}
		out = append(out, trimmed)
	}
	if requireNonEmpty && len(out) == 0 {
		return nil, fmt.Errorf("%s must contain at least one value", field)
	}
	return out, nil
}

func validIncidentFollowUpTaskType(value IncidentFollowUpTaskType) bool {
	switch value {
	case IncidentFollowUpEvidence, IncidentFollowUpContainment, IncidentFollowUpCorrection,
		IncidentFollowUpAuthorization, IncidentFollowUpValidation, IncidentFollowUpReview,
		IncidentFollowUpCommunication, IncidentFollowUpCleanup:
		return true
	default:
		return false
	}
}

func validIncidentFollowUpStatus(value IncidentFollowUpStatus) bool {
	switch value {
	case IncidentFollowUpProposed, IncidentFollowUpReady, IncidentFollowUpInProgress,
		IncidentFollowUpBlocked, IncidentFollowUpDone, IncidentFollowUpDeclined,
		IncidentFollowUpSuperseded:
		return true
	default:
		return false
	}
}
