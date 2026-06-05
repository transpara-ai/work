package work_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestEpic11GitHubClientCreatesDraftPR(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number": 111, "html_url": "https://github.com/transpara-ai/docs/pull/111",
			"node_id": "PR_node_111", "draft": true, "state": "open",
			"base":       map[string]any{"ref": "main", "sha": "basesha"},
			"head":       map[string]any{"ref": "codex/civic-roles", "sha": "headsha"},
			"created_at": "2026-06-05T12:00:00Z",
		})
	}))
	defer srv.Close()

	client := work.NewEpic11GitHubPullRequestCreator("test-token", work.WithEpic11GitHubBaseURL(srv.URL))
	res, err := client.CreateDraftPullRequest(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "transpara-ai/docs", BaseRef: "main", BaseSHA: "basesha",
		HeadRef: "codex/civic-roles", HeadSHA: "headsha",
		Title: "[codex] Document the civic roles", Body: "## Summary\n", Draft: true, MaintainerCanModify: true,
	})
	if err != nil {
		t.Fatalf("CreateDraftPullRequest: %v", err)
	}
	if res.Number != 111 || !res.Draft || res.State != "open" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.URL != "https://github.com/transpara-ai/docs/pull/111" {
		t.Fatalf("URL = %q", res.URL)
	}
	if res.GitHubResponseIDOrEquivalent != "PR_node_111" {
		t.Fatalf("GitHubResponseIDOrEquivalent = %q", res.GitHubResponseIDOrEquivalent)
	}
	if res.BaseRef != "main" {
		t.Fatalf("BaseRef = %q", res.BaseRef)
	}
	if res.BaseSHA != "basesha" {
		t.Fatalf("BaseSHA = %q", res.BaseSHA)
	}
	if res.HeadRef != "codex/civic-roles" {
		t.Fatalf("HeadRef = %q", res.HeadRef)
	}
	if res.HeadSHA != "headsha" {
		t.Fatalf("HeadSHA = %q", res.HeadSHA)
	}
	if res.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if gotPath != "/repos/transpara-ai/docs/pulls" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"draft":true`) {
		t.Fatalf("body missing draft flag: %s", gotBody)
	}
}

func TestEpic11GitHubClientPreflightHeadReturnsSHAAndFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/commits/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"sha": "remotesha"})
		case strings.Contains(r.URL.Path, "/compare/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]any{{"filename": "dark-factory/civic-roles.md"}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := work.NewEpic11GitHubPullRequestCreator("test-token", work.WithEpic11GitHubBaseURL(srv.URL))
	state, err := client.PreflightHead(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "transpara-ai/docs", BaseRef: "main", HeadRef: "codex/civic-roles", HeadSHA: "remotesha",
	})
	if err != nil {
		t.Fatalf("PreflightHead: %v", err)
	}
	if state.HeadSHA != "remotesha" {
		t.Fatalf("HeadSHA = %q; want remotesha", state.HeadSHA)
	}
	if len(state.ChangedFiles) != 1 || state.ChangedFiles[0] != "dark-factory/civic-roles.md" {
		t.Fatalf("ChangedFiles = %v; want [dark-factory/civic-roles.md]", state.ChangedFiles)
	}
}

func TestEpic11GitHubClientEmptyTokenErrors(t *testing.T) {
	client := work.NewEpic11GitHubPullRequestCreator("")
	_, err := client.CreateDraftPullRequest(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "transpara-ai/docs",
	})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestEpic11GitHubClientBadRepositoryErrors(t *testing.T) {
	client := work.NewEpic11GitHubPullRequestCreator("token")
	_, err := client.CreateDraftPullRequest(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "not-owner-slash-repo",
	})
	if err == nil {
		t.Fatal("expected error for bad repository format")
	}
}

func TestEpic11GitHubClientNon201Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "Validation Failed",
		})
	}))
	defer srv.Close()

	client := work.NewEpic11GitHubPullRequestCreator("token", work.WithEpic11GitHubBaseURL(srv.URL))
	_, err := client.CreateDraftPullRequest(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "transpara-ai/docs", Draft: true,
	})
	if err == nil {
		t.Fatal("expected error for non-201 response")
	}
	if !strings.Contains(err.Error(), "422") && !strings.Contains(err.Error(), "Unprocessable") {
		t.Fatalf("error should mention status: %v", err)
	}
	if !strings.Contains(err.Error(), "Validation Failed") {
		t.Fatalf("error should contain github message: %v", err)
	}
}

func TestEpic11GitHubClientNonJSONErrorSurfacesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>502</html>"))
	}))
	defer srv.Close()

	client := work.NewEpic11GitHubPullRequestCreator("token", work.WithEpic11GitHubBaseURL(srv.URL))
	_, err := client.CreateDraftPullRequest(context.Background(), work.Epic11DraftPullRequestMutation{
		Repository: "transpara-ai/docs", Draft: true,
	})
	if err == nil {
		t.Fatal("expected error for 502 response")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error should contain status 502: %v", err)
	}
	if !strings.Contains(err.Error(), "github returned") {
		t.Fatalf("error should say 'github returned', not a decode error: %v", err)
	}
	if strings.Contains(err.Error(), "decode response") {
		t.Fatalf("error should not mention 'decode response' (I1 fix): %v", err)
	}
}
