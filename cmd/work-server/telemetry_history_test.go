package main

import (
	"testing"
	"time"
)

func TestBuildAgentHistories(t *testing.T) {
	t0 := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)

	mkRow := func(actor, role, state string, offsetSec int) snapshotRow {
		return snapshotRow{
			ActorID:    actor,
			Role:       role,
			State:      state,
			Model:      "claude-sonnet-4-6",
			Iteration:  1,
			MaxIter:    10,
			TokensUsed: 1000,
			CostUSD:    0.01,
			Errors:     0,
			RecordedAt: t0.Add(time.Duration(offsetSec) * time.Second),
		}
	}

	tests := []struct {
		name       string
		rows       []snapshotRow
		wantAgents int
		check      func(t *testing.T, result []telAgentHistory)
	}{
		{
			name:       "empty input",
			rows:       nil,
			wantAgents: 0,
		},
		{
			name: "single agent single state",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "processing", 10),
				mkRow("a1", "builder", "processing", 20),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if h.ActorID != "a1" {
					t.Errorf("actor_id = %q, want a1", h.ActorID)
				}
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].State != "processing" {
					t.Errorf("state = %q, want processing", h.States[0].State)
				}
				if h.States[0].Duration != 20 {
					t.Errorf("duration = %f, want 20", h.States[0].Duration)
				}
			},
		},
		{
			name: "single agent multiple transitions",
			rows: []snapshotRow{
				mkRow("a1", "builder", "idle", 0),
				mkRow("a1", "builder", "processing", 10),
				mkRow("a1", "builder", "waiting", 100),
				mkRow("a1", "builder", "retired", 200),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 4 {
					t.Fatalf("states len = %d, want 4", len(h.States))
				}
				wantStates := []string{"idle", "processing", "waiting", "retired"}
				wantDurations := []float64{10, 90, 100, 0}
				for i, s := range h.States {
					if s.State != wantStates[i] {
						t.Errorf("states[%d].State = %q, want %q", i, s.State, wantStates[i])
					}
					if s.Duration != wantDurations[i] {
						t.Errorf("states[%d].Duration = %f, want %f", i, s.Duration, wantDurations[i])
					}
				}
				if h.CurrentState != "retired" {
					t.Errorf("current_state = %q, want retired", h.CurrentState)
				}
			},
		},
		{
			name: "multiple agents same role",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "retired", 60),
				mkRow("a2", "builder", "processing", 10),
				mkRow("a2", "builder", "processing", 70),
			},
			wantAgents: 2,
			check: func(t *testing.T, result []telAgentHistory) {
				if result[0].ActorID != "a1" || result[1].ActorID != "a2" {
					t.Errorf("unexpected actor order: %s, %s", result[0].ActorID, result[1].ActorID)
				}
				if len(result[0].States) != 2 {
					t.Errorf("a1 states = %d, want 2", len(result[0].States))
				}
				if len(result[1].States) != 1 {
					t.Errorf("a2 states = %d, want 1", len(result[1].States))
				}
			},
		},
		{
			name: "stuck detection - gap exceeds threshold",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "processing", 10),
				mkRow("a1", "builder", "processing", 140),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 3 {
					t.Fatalf("states len = %d, want 3", len(h.States))
				}
				if h.States[0].State != "processing" {
					t.Errorf("states[0] = %q, want processing", h.States[0].State)
				}
				if h.States[1].State != "stuck" {
					t.Errorf("states[1] = %q, want stuck", h.States[1].State)
				}
				if h.States[1].Duration != 130 {
					t.Errorf("stuck duration = %f, want 130", h.States[1].Duration)
				}
				if h.States[2].State != "processing" {
					t.Errorf("states[2] = %q, want processing", h.States[2].State)
				}
			},
		},
		{
			name: "gap after terminal state is not stuck",
			rows: []snapshotRow{
				mkRow("a1", "builder", "retired", 0),
				mkRow("a1", "builder", "retired", 200),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].State != "retired" {
					t.Errorf("state = %q, want retired", h.States[0].State)
				}
			},
		},
		{
			name: "gap before terminal transition is not stuck",
			rows: []snapshotRow{
				mkRow("a1", "builder", "processing", 0),
				mkRow("a1", "builder", "processing", 10),
				// 150s gap (> 2min) but next state is retired — legitimate shutdown
				mkRow("a1", "builder", "retired", 160),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				// Should be processing + retired, NO stuck span
				if len(h.States) != 2 {
					t.Fatalf("states len = %d, want 2", len(h.States))
				}
				if h.States[0].State != "processing" {
					t.Errorf("states[0] = %q, want processing", h.States[0].State)
				}
				if h.States[1].State != "retired" {
					t.Errorf("states[1] = %q, want retired", h.States[1].State)
				}
			},
		},
		{
			name: "single snapshot - just spawned",
			rows: []snapshotRow{
				mkRow("a1", "scout", "idle", 0),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if len(h.States) != 1 {
					t.Fatalf("states len = %d, want 1", len(h.States))
				}
				if h.States[0].Duration != 0 {
					t.Errorf("duration = %f, want 0", h.States[0].Duration)
				}
			},
		},
		{
			name: "first_seen and last_seen correct",
			rows: []snapshotRow{
				mkRow("a1", "critic", "idle", 0),
				mkRow("a1", "critic", "processing", 30),
				mkRow("a1", "critic", "retired", 300),
			},
			wantAgents: 1,
			check: func(t *testing.T, result []telAgentHistory) {
				h := result[0]
				if !h.FirstSeen.Equal(t0) {
					t.Errorf("first_seen = %v, want %v", h.FirstSeen, t0)
				}
				want := t0.Add(300 * time.Second)
				if !h.LastSeen.Equal(want) {
					t.Errorf("last_seen = %v, want %v", h.LastSeen, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentHistories(tt.rows)
			if len(result) != tt.wantAgents {
				t.Fatalf("agent count = %d, want %d", len(result), tt.wantAgents)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
