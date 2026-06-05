package work_test

import (
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
