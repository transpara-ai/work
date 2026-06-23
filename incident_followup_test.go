package work_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func validIncidentFollowUp() work.IncidentFollowUp {
	return work.IncidentFollowUp{
		TaskID:                "INC-001-work-add-incident-follow-up-schema",
		IncidentID:            "INC-001",
		IncidentRecord:        "operation/docs/incidents/INC-001-pre-live-operation.md",
		TaskType:              work.IncidentFollowUpCorrection,
		OwningRepo:            "transpara-ai/work",
		RequestedBy:           "incident-lead",
		AssignedTo:            "work-maintainer",
		Status:                work.IncidentFollowUpReady,
		Severity:              "medium",
		Summary:               "Accept the incident follow-up schema as a Work task artifact.",
		RequiredEvidence:      []string{"typed schema artifact can be attached to a Work task"},
		AuthorizationRequired: false,
		ValidationRequired:    true,
		BlockingDependencies:  []string{},
		AcceptanceCriteria:    []string{"Work can validate and list incident follow-up artifacts"},
	}
}

func TestIncidentFollowUpArtifactBodyValidatesAndPreservesContractFields(t *testing.T) {
	body, err := work.IncidentFollowUpArtifactBody(validIncidentFollowUp())
	if err != nil {
		t.Fatalf("IncidentFollowUpArtifactBody: %v", err)
	}

	var decoded struct {
		Schema        string                `json:"schema"`
		SchemaVersion string                `json:"schema_version"`
		FollowUp      work.IncidentFollowUp `json:"follow_up"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("body is not JSON: %v\n%s", err, body)
	}
	if decoded.Schema != work.IncidentFollowUpSchemaRef {
		t.Fatalf("schema = %q, want %q", decoded.Schema, work.IncidentFollowUpSchemaRef)
	}
	if decoded.SchemaVersion != work.IncidentFollowUpSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", decoded.SchemaVersion, work.IncidentFollowUpSchemaVersion)
	}
	if decoded.FollowUp.TaskType != work.IncidentFollowUpCorrection {
		t.Fatalf("task_type = %q", decoded.FollowUp.TaskType)
	}
	if decoded.FollowUp.Status != work.IncidentFollowUpReady {
		t.Fatalf("status = %q", decoded.FollowUp.Status)
	}
	if decoded.FollowUp.IncidentRecord != "operation/docs/incidents/INC-001-pre-live-operation.md" {
		t.Fatalf("incident_record = %q", decoded.FollowUp.IncidentRecord)
	}
	if decoded.FollowUp.ClosureLink != "" {
		t.Fatalf("closure_link = %q, want empty while task is open", decoded.FollowUp.ClosureLink)
	}
	if decoded.FollowUp.ValidationEvidence == nil {
		t.Fatal("validation_evidence is nil; want an empty list field in the contract payload")
	}

	githubURL := validIncidentFollowUp()
	githubURL.IncidentRecord = "https://github.com/transpara-ai/operation/blob/main/docs/incidents/INC-001-pre-live-operation.md"
	body, err = work.IncidentFollowUpArtifactBody(githubURL)
	if err != nil {
		t.Fatalf("github incident record URL rejected: %v", err)
	}
	if !strings.Contains(body, "https://github.com/transpara-ai/operation/blob/main/docs/incidents/INC-001-pre-live-operation.md") {
		t.Fatalf("github incident record was not preserved canonically:\n%s", body)
	}

	legacyRelative := validIncidentFollowUp()
	legacyRelative.IncidentRecord = "civilization-operation/docs/incidents/INC-001-pre-live-operation.md"
	body, err = work.IncidentFollowUpArtifactBody(legacyRelative)
	if err != nil {
		t.Fatalf("legacy relative incident record rejected: %v", err)
	}
	if strings.Contains(body, "civilization-operation") || !strings.Contains(body, "operation/docs/incidents/INC-001-pre-live-operation.md") {
		t.Fatalf("legacy relative incident record was not canonicalized:\n%s", body)
	}

	legacyGitHubURL := validIncidentFollowUp()
	legacyGitHubURL.IncidentRecord = "https://github.com/transpara-ai/civilization-operation/blob/main/docs/incidents/INC-001-pre-live-operation.md"
	body, err = work.IncidentFollowUpArtifactBody(legacyGitHubURL)
	if err != nil {
		t.Fatalf("legacy github incident record URL rejected: %v", err)
	}
	if strings.Contains(body, "civilization-operation") || !strings.Contains(body, "https://github.com/transpara-ai/operation/blob/main/docs/incidents/INC-001-pre-live-operation.md") {
		t.Fatalf("legacy github incident record URL was not canonicalized:\n%s", body)
	}

	wrongOrg := validIncidentFollowUp()
	wrongOrg.IncidentRecord = "https://github.com/someone/civilization-operation/blob/main/docs/incidents/INC-001-pre-live-operation.md"
	if _, err := work.IncidentFollowUpArtifactBody(wrongOrg); err == nil || !strings.Contains(err.Error(), "operation docs/incidents") {
		t.Fatalf("expected wrong-org github URL rejection, got %v", err)
	}

	wrongScheme := validIncidentFollowUp()
	wrongScheme.IncidentRecord = "ftp://github.com/transpara-ai/civilization-operation/blob/main/docs/incidents/INC-001-pre-live-operation.md"
	if _, err := work.IncidentFollowUpArtifactBody(wrongScheme); err == nil || !strings.Contains(err.Error(), "operation docs/incidents") {
		t.Fatalf("expected wrong-scheme github URL rejection, got %v", err)
	}

	fileURL := validIncidentFollowUp()
	fileURL.IncidentRecord = "file:///civilization-operation/docs/incidents/INC-001-pre-live-operation.md"
	if _, err := work.IncidentFollowUpArtifactBody(fileURL); err == nil || !strings.Contains(err.Error(), "operation docs/incidents") {
		t.Fatalf("expected file URL rejection, got %v", err)
	}
}

func TestIncidentFollowUpRejectsInvalidContractValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*work.IncidentFollowUp)
		wantErr string
	}{
		{
			name: "missing incident record",
			mutate: func(f *work.IncidentFollowUp) {
				f.IncidentRecord = ""
			},
			wantErr: "incident_record is required",
		},
		{
			name: "incident record outside operation incidents",
			mutate: func(f *work.IncidentFollowUp) {
				f.IncidentRecord = "work/docs/incidents/INC-001.md"
			},
			wantErr: "operation docs/incidents",
		},
		{
			name: "incident record bare operation incidents directory",
			mutate: func(f *work.IncidentFollowUp) {
				f.IncidentRecord = "operation/docs/incidents"
			},
			wantErr: "operation docs/incidents",
		},
		{
			name: "false positive incident record path",
			mutate: func(f *work.IncidentFollowUp) {
				f.IncidentRecord = "not-civilization-operation/docs/incidents/INC-001.md"
			},
			wantErr: "operation docs/incidents",
		},
		{
			name: "unknown task type",
			mutate: func(f *work.IncidentFollowUp) {
				f.TaskType = work.IncidentFollowUpTaskType("OPS")
			},
			wantErr: "task_type",
		},
		{
			name: "unknown status",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpStatus("CLOSED")
			},
			wantErr: "status",
		},
		{
			name: "empty required evidence",
			mutate: func(f *work.IncidentFollowUp) {
				f.RequiredEvidence = []string{""}
			},
			wantErr: "required_evidence",
		},
		{
			name: "empty acceptance criteria",
			mutate: func(f *work.IncidentFollowUp) {
				f.AcceptanceCriteria = nil
			},
			wantErr: "acceptance_criteria",
		},
		{
			name: "control character",
			mutate: func(f *work.IncidentFollowUp) {
				f.Summary = "bad\nsummary"
			},
			wantErr: "control characters",
		},
		{
			name: "authorization evidence control character",
			mutate: func(f *work.IncidentFollowUp) {
				f.AuthorizationEvidence = "bad\nevidence"
			},
			wantErr: "control characters",
		},
		{
			name: "closure link control character",
			mutate: func(f *work.IncidentFollowUp) {
				f.ClosureLink = "bad\nlink"
			},
			wantErr: "control characters",
		},
		{
			name: "blocked status requires dependency",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpBlocked
				f.BlockingDependencies = nil
			},
			wantErr: "blocking_dependencies",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followUp := validIncidentFollowUp()
			tt.mutate(&followUp)

			_, err := work.IncidentFollowUpArtifactBody(followUp)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestIncidentFollowUpDoneRequiresClosureAndEvidence(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*work.IncidentFollowUp)
		wantErr string
	}{
		{
			name: "done requires closure link",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpDone
			},
			wantErr: "closure_link",
		},
		{
			name: "done authorization requires evidence",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpDone
				f.ClosureLink = "https://github.com/transpara-ai/work/pull/51"
				f.AuthorizationRequired = true
			},
			wantErr: "authorization_evidence",
		},
		{
			name: "done validation requires evidence",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpDone
				f.ClosureLink = "https://github.com/transpara-ai/work/pull/51"
				f.ValidationRequired = true
			},
			wantErr: "validation_evidence",
		},
		{
			name: "declined requires closure link",
			mutate: func(f *work.IncidentFollowUp) {
				f.Status = work.IncidentFollowUpDeclined
			},
			wantErr: "closure_link",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followUp := validIncidentFollowUp()
			tt.mutate(&followUp)

			_, err := work.IncidentFollowUpArtifactBody(followUp)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}

	done := validIncidentFollowUp()
	done.Status = work.IncidentFollowUpDone
	done.ClosureLink = "https://github.com/transpara-ai/work/pull/51"
	done.AuthorizationRequired = true
	done.AuthorizationEvidence = "Michael approved the follow-up route in the incident record."
	done.ValidationRequired = true
	done.ValidationEvidence = []string{"make verify passed"}
	if _, err := work.IncidentFollowUpArtifactBody(done); err != nil {
		t.Fatalf("valid DONE follow-up rejected: %v", err)
	}
}

func TestTaskStore_AddIncidentFollowUpArtifactRecordsContractArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Adopt incident follow-up schema", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ts.AddIncidentFollowUpArtifact(testActor, task.ID, validIncidentFollowUp(), causes, testConv); err != nil {
		t.Fatalf("AddIncidentFollowUpArtifact: %v", err)
	}
	if err := ts.AddArtifact(testActor, task.ID, "other", "text/plain", "ignore me", causes, testConv); err != nil {
		t.Fatalf("AddArtifact other: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifacts len = %d, want 2", len(artifacts))
	}
	var artifactLabel, mediaType string
	for _, artifact := range artifacts {
		if artifact.Label == work.IncidentFollowUpArtifactLabel {
			artifactLabel = artifact.Label
			mediaType = artifact.MediaType
		}
	}
	if artifactLabel == "" {
		t.Fatalf("missing %s artifact in %+v", work.IncidentFollowUpArtifactLabel, artifacts)
	}
	if mediaType != work.IncidentFollowUpMediaType {
		t.Fatalf("artifact media type = %q, want %q", mediaType, work.IncidentFollowUpMediaType)
	}

	replayed := newTaskStore(t, s)
	followUps, err := replayed.ListIncidentFollowUps(task.ID)
	if err != nil {
		t.Fatalf("ListIncidentFollowUps: %v", err)
	}
	if len(followUps) != 1 {
		t.Fatalf("followUps len = %d, want 1", len(followUps))
	}
	if followUps[0].FollowUp.TaskID != "INC-001-work-add-incident-follow-up-schema" {
		t.Fatalf("follow-up task_id = %q", followUps[0].FollowUp.TaskID)
	}
	if followUps[0].CreatedBy != testActor {
		t.Fatalf("CreatedBy = %s, want %s", followUps[0].CreatedBy.Value(), testActor.Value())
	}
}

func TestTaskStore_AddArtifactAcceptsLegacyIncidentFollowUpSchema(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Replay legacy incident follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body, err := work.IncidentFollowUpArtifactBody(validIncidentFollowUp())
	if err != nil {
		t.Fatalf("IncidentFollowUpArtifactBody: %v", err)
	}
	body = strings.Replace(body,
		`"schema": "operation/docs/operations/work-incident-follow-up-schema.md"`,
		`"schema": "civilization-operation/docs/operations/work-incident-follow-up-schema.md"`,
		1,
	)
	body = strings.Replace(body,
		`"incident_record": "operation/docs/incidents/INC-001-pre-live-operation.md"`,
		`"incident_record": "civilization-operation/docs/incidents/INC-001-pre-live-operation.md"`,
		1,
	)

	if err := ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, body, causes, testConv); err != nil {
		t.Fatalf("AddArtifact legacy incident follow-up: %v", err)
	}

	followUps, err := ts.ListIncidentFollowUps(task.ID)
	if err != nil {
		t.Fatalf("ListIncidentFollowUps: %v", err)
	}
	if len(followUps) != 1 {
		t.Fatalf("followUps len = %d, want 1", len(followUps))
	}
	if got, want := followUps[0].FollowUp.IncidentRecord, "operation/docs/incidents/INC-001-pre-live-operation.md"; got != want {
		t.Fatalf("incident_record = %q, want %q", got, want)
	}
}

func TestTaskStore_LatestIncidentFollowUpReturnsCurrentContractArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Advance incident follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ready := validIncidentFollowUp()
	if err := ts.AddIncidentFollowUpArtifact(testActor, task.ID, ready, causes, testConv); err != nil {
		t.Fatalf("AddIncidentFollowUpArtifact ready: %v", err)
	}

	done := validIncidentFollowUp()
	done.Status = work.IncidentFollowUpDone
	done.ClosureLink = "https://github.com/transpara-ai/work/pull/51"
	done.ValidationEvidence = []string{"make verify passed"}
	if err := ts.AddIncidentFollowUpArtifact(testActor, task.ID, done, causes, testConv); err != nil {
		t.Fatalf("AddIncidentFollowUpArtifact done: %v", err)
	}

	latest, ok, err := ts.LatestIncidentFollowUp(task.ID)
	if err != nil {
		t.Fatalf("LatestIncidentFollowUp: %v", err)
	}
	if !ok {
		t.Fatal("LatestIncidentFollowUp returned ok=false")
	}
	if latest.FollowUp.Status != work.IncidentFollowUpDone {
		t.Fatalf("latest status = %q, want DONE", latest.FollowUp.Status)
	}
}

func TestTaskStore_LatestIncidentFollowUpNoFollowUps(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "No incident follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	latest, ok, err := ts.LatestIncidentFollowUp(task.ID)
	if err != nil {
		t.Fatalf("LatestIncidentFollowUp: %v", err)
	}
	if ok {
		t.Fatalf("LatestIncidentFollowUp ok = true with latest %+v, want false", latest)
	}
}

func TestTaskStore_AddArtifactRejectsMalformedIncidentFollowUpArtifact(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Malformed follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, "{", causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "artifact is invalid") {
		t.Fatalf("expected invalid artifact error, got %v", err)
	}
}

func TestTaskStore_AddArtifactRejectsWrongIncidentFollowUpSchema(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Wrong schema follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body := `{"schema":"other","schema_version":"2026-06-16","follow_up":{}}`
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, body, causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestTaskStore_AddArtifactRejectsWrongIncidentFollowUpSchemaVersion(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Wrong schema version follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body, err := work.IncidentFollowUpArtifactBody(validIncidentFollowUp())
	if err != nil {
		t.Fatalf("IncidentFollowUpArtifactBody: %v", err)
	}
	body = strings.Replace(body, `"schema_version": "2026-06-16"`, `"schema_version": "2026-07-01"`, 1)
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, body, causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("expected schema_version error, got %v", err)
	}
}

func TestTaskStore_AddArtifactRejectsIncidentFollowUpPayloadThatDoesNotValidate(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Invalid stored follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body := `{"schema":"civilization-operation/docs/operations/work-incident-follow-up-schema.md","schema_version":"2026-06-16","follow_up":{}}`
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, body, causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "artifact is invalid") {
		t.Fatalf("expected validate error, got %v", err)
	}
}

func TestTaskStore_AddArtifactRejectsUnknownIncidentFollowUpFields(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Unknown field follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body, err := work.IncidentFollowUpArtifactBody(validIncidentFollowUp())
	if err != nil {
		t.Fatalf("IncidentFollowUpArtifactBody: %v", err)
	}
	body = strings.Replace(body, `"closure_link": ""`, `"closure_link": "", "unexpected_field": "drift"`, 1)
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, work.IncidentFollowUpMediaType, body, causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestTaskStore_AddArtifactRejectsIncidentFollowUpWrongMediaType(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := ts.Create(testActor, "Wrong media type follow-up", "", causes, testConv)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body, err := work.IncidentFollowUpArtifactBody(validIncidentFollowUp())
	if err != nil {
		t.Fatalf("IncidentFollowUpArtifactBody: %v", err)
	}
	err = ts.AddArtifact(testActor, task.ID, work.IncidentFollowUpArtifactLabel, "text/plain", body, causes, testConv)
	if err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("expected media type error, got %v", err)
	}
}
