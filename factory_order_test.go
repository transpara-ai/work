package work_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestSeedFactoryOrderCreatesReadinessGatedTask(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	order := work.FactoryOrder{
		Kind:               work.OrderSoftwarePR,
		ID:                 "fo_civic_roles",
		Title:              "Document the civic roles",
		Intent:             "Produce dark-factory/civic-roles.md describing the civic roles.",
		RiskClass:          "low",
		DefinitionOfDone:   "civic-roles.md exists and names strategist, planner, implementer, reviewer, guardian, cto, spawner, allocator, sysmon.",
		AcceptanceCriteria: "Each civic role has a one-line responsibility; prose is reviewer-approved.",
		TestPlan:           "Reviewer confirms each role is named and described; markdown lints clean.",
		ExpectedOutputs:    []string{"dark-factory/civic-roles.md"},
	}

	task, err := work.SeedFactoryOrder(ts, testActor, order, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}
	if task.FactoryOrderID != order.ID {
		t.Fatalf("FactoryOrderID = %q, want %q", task.FactoryOrderID, order.ID)
	}
	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("task not ready; missing gates: %v", readiness.MissingGates)
	}
}

func TestSeedFactoryOrderSynthesizesDefaults(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	order := work.FactoryOrder{
		// Kind, Cell, RiskClass, RequirementIDs, AcceptanceCriterionIDs all omitted
		ID:                 "fo_defaults_check",
		Title:              "Defaults check",
		Intent:             "Verify synthesis defaults.",
		DefinitionOfDone:   "d",
		AcceptanceCriteria: "a",
		TestPlan:           "t",
	}
	task, err := work.SeedFactoryOrder(ts, testActor, order, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder (defaults): %v", err)
	}
	if task.Cell != "implementation" {
		t.Errorf("Cell = %q, want implementation", task.Cell)
	}
	if task.RiskClass != "low" {
		t.Errorf("RiskClass = %q, want low", task.RiskClass)
	}
	if len(task.RequirementIDs) != 1 || task.RequirementIDs[0] != "req_defaults_check" {
		t.Errorf("RequirementIDs = %v, want [req_defaults_check]", task.RequirementIDs)
	}
	if len(task.AcceptanceCriterionIDs) != 1 || task.AcceptanceCriterionIDs[0] != "ac_defaults_check" {
		t.Errorf("AcceptanceCriterionIDs = %v, want [ac_defaults_check]", task.AcceptanceCriterionIDs)
	}
	if task.FactoryOrderID != order.ID {
		t.Errorf("FactoryOrderID = %q, want %q", task.FactoryOrderID, order.ID)
	}
	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("defaults task not ready: %v", readiness.MissingGates)
	}
}

func TestSeedFactoryOrderAllowsAbsentGates(t *testing.T) {
	// D: gate bodies are optional at seed. The planner attaches any that are
	// absent, and Readiness (not the seed) enforces non-empty. So seeding an
	// order without definition_of_done/acceptance_criteria/test_plan must
	// succeed and yield a NOT-ready task whose missing gates are exactly the
	// three readiness gates.
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	order := work.FactoryOrder{
		ID:     "fo_absent_gates",
		Title:  "Absent gates",
		Intent: "Seed without gate bodies; the planner fills them later.",
		// DefinitionOfDone / AcceptanceCriteria / TestPlan intentionally empty.
	}
	task, err := work.SeedFactoryOrder(ts, testActor, order, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder with absent gates: %v", err)
	}
	readiness, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if readiness.Ready {
		t.Fatalf("task should not be ready with absent gates")
	}
	if len(readiness.MissingGates) != 3 {
		t.Fatalf("MissingGates = %v, want all 3 readiness gates", readiness.MissingGates)
	}
}

func TestSeedFactoryOrderRecordsStructuredModelOverrides(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	maxCost := 0.25

	task, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
		ID:                 "fo_model_override",
		Title:              "Model override",
		Intent:             "Seed with structured model override.",
		DefinitionOfDone:   "d",
		AcceptanceCriteria: "a",
		TestPlan:           "t",
		ModelOverrides: []work.FactoryOrderModelOverride{
			{
				Role:                 "guardian",
				Model:                "api-sonnet",
				RequestedAuthMode:    "api-key",
				RequiredCapabilities: []string{"reasoning"},
				MaxCostPerCallUSD:    &maxCost,
				ResolvedModel:        "api-claude-sonnet-4-6",
				ResolvedProvider:     "anthropic",
				AuthMode:             "api-key",
			},
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	var body string
	for _, artifact := range artifacts {
		if artifact.Label == work.FactoryOrderModelOverridesArtifactLabel {
			if artifact.MediaType != "application/json" {
				t.Fatalf("override artifact media type = %q, want application/json", artifact.MediaType)
			}
			body = artifact.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing %s artifact in %+v", work.FactoryOrderModelOverridesArtifactLabel, artifacts)
	}

	var decoded struct {
		ModelOverrides []work.FactoryOrderModelOverride `json:"model_overrides"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("override artifact is not JSON: %v\n%s", err, body)
	}
	if len(decoded.ModelOverrides) != 1 {
		t.Fatalf("model_overrides = %+v, want one", decoded.ModelOverrides)
	}
	override := decoded.ModelOverrides[0]
	if override.Role != "guardian" || override.Model != "api-sonnet" || override.RequestedAuthMode != "api-key" {
		t.Fatalf("override = %+v, want guardian api-sonnet api-key", override)
	}
	if override.ResolvedProvider != "anthropic" || override.AuthMode != "api-key" || override.MaxCostPerCallUSD == nil || *override.MaxCostPerCallUSD != maxCost {
		t.Fatalf("resolved override = %+v, want anthropic/api-key with cost cap", override)
	}

	replayed := newTaskStore(t, s)
	projection, err := replayed.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask: %v", err)
	}
	if len(projection.ModelOverrides) != 1 {
		t.Fatalf("projected model overrides = %+v, want one", projection.ModelOverrides)
	}
	projected := projection.ModelOverrides[0]
	if projected.Role != "guardian" || projected.ResolvedProvider != "anthropic" || projected.AuthMode != "api-key" {
		t.Fatalf("projected override = %+v, want guardian anthropic api-key", projected)
	}
}

func TestSeedFactoryOrderRecordsSourceIssueRecords(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
		ID:                 "fo_source_issue",
		Title:              "Source issue",
		Intent:             "Seed with GitHub issue source intent.",
		DefinitionOfDone:   "d",
		AcceptanceCriteria: "a",
		TestPlan:           "t",
		SourceIssueRecords: []work.FactoryOrderSourceIssueRecord{
			{
				Repo:   " transpara-ai/work ",
				Number: 60,
				URL:    " https://github.com/transpara-ai/work/issues/60 ",
				Title:  " FactoryOrder source-intent ingestion from GitHub issues ",
				Goal:   " Make GitHub issues usable as source intent without treating issues as authority. ",
				AcceptanceCriteria: []string{
					" source issue refs are preserved ",
					"",
					" replay projection exposes normalized records ",
				},
				RiskNotes:  []string{" issue records are not authority ", ""},
				Labels:     []string{" cc:intake ", " cc:pr-ready "},
				SourceRefs: []string{" work#60 "},
			},
			{
				Repo:   "transpara-ai/docs",
				Number: 197,
				Title:  "Development Arc issue-source migration parent tracker",
			},
		},
	}, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}

	artifacts, err := ts.ListArtifacts(task.ID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	var body string
	for _, artifact := range artifacts {
		if artifact.Label == work.FactoryOrderSourceIssuesArtifactLabel {
			if artifact.MediaType != "application/json" {
				t.Fatalf("source issue artifact media type = %q, want application/json", artifact.MediaType)
			}
			body = artifact.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing %s artifact in %+v", work.FactoryOrderSourceIssuesArtifactLabel, artifacts)
	}

	var decoded struct {
		SourceIssueRecords  []work.FactoryOrderSourceIssueRecord `json:"source_issue_records"`
		AuthorityExclusions []string                             `json:"authority_exclusions"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("source issue artifact is not JSON: %v\n%s", err, body)
	}
	if len(decoded.SourceIssueRecords) != 2 {
		t.Fatalf("source_issue_records = %+v, want two records", decoded.SourceIssueRecords)
	}
	first := decoded.SourceIssueRecords[0]
	if first.Repo != "transpara-ai/work" || first.Number != 60 || first.URL != "https://github.com/transpara-ai/work/issues/60" {
		t.Fatalf("first issue = %+v, want normalized work#60", first)
	}
	if strings.Join(first.AcceptanceCriteria, "|") != "source issue refs are preserved|replay projection exposes normalized records" {
		t.Fatalf("acceptance criteria = %#v; want trimmed non-empty values", first.AcceptanceCriteria)
	}
	if strings.Join(first.SourceRefs, ",") != "work#60" {
		t.Fatalf("source refs = %#v; want caller-provided compact ref", first.SourceRefs)
	}
	second := decoded.SourceIssueRecords[1]
	if second.Goal != second.Title || strings.Join(second.SourceRefs, ",") != "transpara-ai/docs#197" {
		t.Fatalf("second issue = %+v, want derived goal and source ref", second)
	}
	if !containsString(decoded.AuthorityExclusions, "no_protected_action_authority") || !containsString(decoded.AuthorityExclusions, "no_eventgraph_write") {
		t.Fatalf("authority exclusions = %#v; want protected-action and EventGraph exclusions", decoded.AuthorityExclusions)
	}

	replayed := newTaskStore(t, s)
	projection, err := replayed.ProjectTask(task.ID)
	if err != nil {
		t.Fatalf("ProjectTask: %v", err)
	}
	if len(projection.SourceIssueRecords) != 2 {
		t.Fatalf("projected source issues = %+v, want two", projection.SourceIssueRecords)
	}
	if projection.SourceIssueRecords[0].Repo != "transpara-ai/work" || strings.Join(projection.SourceIssueRecords[1].SourceRefs, ",") != "transpara-ai/docs#197" {
		t.Fatalf("projected source issues = %+v, want normalized replay", projection.SourceIssueRecords)
	}
}

func TestSeedFactoryOrderRejectsInvalidSourceIssueRecords(t *testing.T) {
	tests := []struct {
		name    string
		record  work.FactoryOrderSourceIssueRecord
		wantErr string
	}{
		{
			name:    "missing repo",
			record:  work.FactoryOrderSourceIssueRecord{Number: 60, Title: "issue"},
			wantErr: "source_issue_records[0].repo is required",
		},
		{
			name:    "missing number",
			record:  work.FactoryOrderSourceIssueRecord{Repo: "transpara-ai/work", Title: "issue"},
			wantErr: "source_issue_records[0].number must be positive",
		},
		{
			name:    "missing title",
			record:  work.FactoryOrderSourceIssueRecord{Repo: "transpara-ai/work", Number: 60},
			wantErr: "source_issue_records[0].title is required",
		},
		{
			name:    "control character",
			record:  work.FactoryOrderSourceIssueRecord{Repo: "transpara-ai/work", Number: 60, Title: "issue\nrecord"},
			wantErr: "source_issue_records[0] contains control characters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)

			_, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
				ID:                 "fo_bad_source_issue",
				Title:              "Bad source issue",
				DefinitionOfDone:   "d",
				AcceptanceCriteria: "a",
				TestPlan:           "t",
				SourceIssueRecords: []work.FactoryOrderSourceIssueRecord{tt.record},
			}, causes, testConv)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected source issue error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSeedFactoryOrderRejectsInvalidModelOverrides(t *testing.T) {
	negativeCost := -0.01
	tests := []struct {
		name      string
		overrides []work.FactoryOrderModelOverride
		wantErr   string
	}{
		{
			name:      "missing role",
			overrides: []work.FactoryOrderModelOverride{{Model: "sonnet"}},
			wantErr:   "role is required",
		},
		{
			name:      "control character",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "son\nnet"}},
			wantErr:   "control characters",
		},
		{
			name: "duplicate role",
			overrides: []work.FactoryOrderModelOverride{
				{Role: "guardian", Model: "sonnet"},
				{Role: "guardian", Model: "opus"},
			},
			wantErr: "duplicated",
		},
		{
			name: "duplicate role case variant",
			overrides: []work.FactoryOrderModelOverride{
				{Role: "guardian", Model: "sonnet"},
				{Role: "Guardian", Model: "opus"},
			},
			wantErr: "duplicated",
		},
		{
			name:      "invalid requested auth mode",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "sonnet", RequestedAuthMode: "oauth"}},
			wantErr:   "requested_auth_mode",
		},
		{
			name:      "invalid resolved auth mode",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "sonnet", AuthMode: "oauth"}},
			wantErr:   "auth_mode",
		},
		{
			name:      "negative cost cap",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "sonnet", MaxCostPerCallUSD: &negativeCost}},
			wantErr:   "max_cost_per_call_usd",
		},
		{
			name:      "empty capability",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "sonnet", RequiredCapabilities: []string{"reasoning", ""}}},
			wantErr:   "required_capabilities contains empty",
		},
		{
			name:      "capability control character",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian", Model: "sonnet", RequiredCapabilities: []string{"cod\ning"}}},
			wantErr:   "required_capabilities contains control",
		},
		{
			name:      "no substantive override",
			overrides: []work.FactoryOrderModelOverride{{Role: "guardian"}},
			wantErr:   "must set model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, causes := setupStore(t)
			ts := newTaskStore(t, s)

			_, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
				ID:             "fo_bad_model_override",
				Title:          "Bad model override",
				ModelOverrides: tt.overrides,
			}, causes, testConv)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected model override error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestReadinessTreatsEmptyGateBodyAsMissing(t *testing.T) {
	// D defense-in-depth: a required gate artifact with an empty body does NOT
	// satisfy readiness — only a non-empty body does. A label-only (empty)
	// artifact must not mark a task ready; the planner must attach a real body.
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)

	task, err := work.SeedFactoryOrder(ts, testActor, work.FactoryOrder{
		ID: "fo_empty_body", Title: "Empty body", Intent: "x",
	}, causes, testConv)
	if err != nil {
		t.Fatalf("SeedFactoryOrder: %v", err)
	}

	addArtifact := func(label, body string) {
		t.Helper()
		if err := ts.AddArtifact(testActor, task.ID, label, "text/markdown", body, causes, testConv); err != nil {
			t.Fatalf("AddArtifact %s: %v", label, err)
		}
	}
	// Two real gates and one whitespace-only (empty) body.
	addArtifact(work.GateAcceptanceCriteria, "real acceptance criteria")
	addArtifact(work.GateTestPlan, "real test plan")
	addArtifact(work.GateDefinitionOfDone, "   ")

	r, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness: %v", err)
	}
	if r.Ready {
		t.Fatalf("task ready despite empty definition_of_done body; missing=%v present=%v", r.MissingGates, r.PresentGates)
	}
	missingDOD := false
	for _, g := range r.MissingGates {
		if g == work.GateDefinitionOfDone {
			missingDOD = true
		}
	}
	if !missingDOD {
		t.Fatalf("MissingGates = %v, want it to include %s", r.MissingGates, work.GateDefinitionOfDone)
	}

	// Filling the body with real content makes the task ready.
	addArtifact(work.GateDefinitionOfDone, "real definition of done")
	r2, err := ts.Readiness(task.ID)
	if err != nil {
		t.Fatalf("Readiness after fill: %v", err)
	}
	if !r2.Ready {
		t.Fatalf("task not ready after filling definition_of_done: missing %v", r2.MissingGates)
	}
}
