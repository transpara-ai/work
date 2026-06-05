package work_test

import (
	"context"
	"testing"
	"time"

	"github.com/transpara-ai/work"
)

func TestBuildEpic11OptionsProducesValidEvidence(t *testing.T) {
	s, causes := setupStore(t)
	ts := newTaskStore(t, s)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	client := &epic11FakePRClient{}

	opts := work.BuildEpic11DocsDraftPROptions(work.Epic11OptionsInput{
		Source:         testActor,
		ConversationID: testConv,
		WorkingDir:     t.TempDir(),
		Client:         client,
		Now:            now,
		Target: work.Epic11DraftPullRequestTarget{
			Repository:             work.Epic11TargetRepository,
			BaseRef:                work.Epic11TargetBaseRef,
			BaseSHA:                "b21e2eca5ce547eebef83a1a392f5ca790c3e44d",
			HeadRef:                "codex/civic-roles",
			HeadSHA:                "b4f9844ecad41a8dc1298e3ac19df3a4e7ac9071",
			HeadExistsOnOrigin:     true,
			Title:                  "[codex] Document the civic roles",
			Body:                   "## Summary\n\nCivic roles.\n",
			ChangedFiles:           []string{"dark-factory/civic-roles.md"},
			ValidationEvidenceRefs: []string{"make verify"},
			Draft:                  true,
			MaintainerCanModify:    true,
			RollbackInstructions:   "Manual rollback only: human may close the draft PR.",
		},
		ActorRole:      "implementer",
		DeciderActorID: "act_human_authorizer",
		DeciderRole:    "human",
		SingleUseNonce: "nonce-civic-roles-pr",
	})
	opts.Causes = causes

	run, err := work.RunEpic11DocsDraftPRLiveMutation(context.Background(), ts, opts)
	if err != nil {
		t.Fatalf("RunEpic11DocsDraftPRLiveMutation rejected builder evidence: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected exactly one client call, got %d", client.calls)
	}
	if run.MutationResult.Number == 0 || !run.MutationResult.Draft {
		t.Fatalf("unexpected mutation result: %+v", run.MutationResult)
	}
}
