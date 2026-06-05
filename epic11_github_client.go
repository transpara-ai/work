package work

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultEpic11GitHubBaseURL = "https://api.github.com"

// Epic11GitHubPullRequestCreator is the production implementation of
// Epic11PullRequestCreator. It calls the GitHub REST API to create one
// draft pull request. Base URL is injectable so tests can use httptest.
type Epic11GitHubPullRequestCreator struct {
	token   string
	baseURL string
	http    *http.Client
}

var _ Epic11PullRequestCreator = (*Epic11GitHubPullRequestCreator)(nil)

// Epic11GitHubOption is a functional option for Epic11GitHubPullRequestCreator.
type Epic11GitHubOption func(*Epic11GitHubPullRequestCreator)

// WithEpic11GitHubBaseURL overrides the GitHub API base URL (for testing).
func WithEpic11GitHubBaseURL(u string) Epic11GitHubOption {
	return func(c *Epic11GitHubPullRequestCreator) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithEpic11GitHubHTTPClient overrides the HTTP client (for testing).
func WithEpic11GitHubHTTPClient(h *http.Client) Epic11GitHubOption {
	return func(c *Epic11GitHubPullRequestCreator) { c.http = h }
}

// NewEpic11GitHubPullRequestCreator returns a production Epic11PullRequestCreator
// backed by the GitHub REST API.
func NewEpic11GitHubPullRequestCreator(token string, opts ...Epic11GitHubOption) *Epic11GitHubPullRequestCreator {
	c := &Epic11GitHubPullRequestCreator{
		token:   token,
		baseURL: defaultEpic11GitHubBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// CreateDraftPullRequest implements Epic11PullRequestCreator by calling
// POST /repos/{owner}/{repo}/pulls with draft:true.
func (c *Epic11GitHubPullRequestCreator) CreateDraftPullRequest(ctx context.Context, m Epic11DraftPullRequestMutation) (Epic11DraftPullRequestResult, error) {
	if c.token == "" {
		return Epic11DraftPullRequestResult{}, fmt.Errorf("epic11 github client: empty token")
	}
	owner, repo, ok := strings.Cut(m.Repository, "/")
	if !ok {
		return Epic11DraftPullRequestResult{}, fmt.Errorf("epic11 github client: repository %q is not owner/repo", m.Repository)
	}

	payload := map[string]any{
		"title":                 m.Title,
		"body":                  m.Body,
		"head":                  m.HeadRef,
		"base":                  m.BaseRef,
		"draft":                 m.Draft,
		"maintainer_can_modify": m.MaintainerCanModify,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Epic11DraftPullRequestResult{}, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Epic11DraftPullRequestResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Epic11DraftPullRequestResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var ghErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&ghErr)
		return Epic11DraftPullRequestResult{}, fmt.Errorf("epic11 github client: github returned %s: %s", resp.Status, ghErr.Message)
	}

	// github response shape — nested base/head structs with lowercase json keys.
	// Go's json decoder matches case-insensitively, so Ref→"ref" and SHA→"sha"
	// decode correctly without explicit tags. Explicit tags included for clarity.
	var gh struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		NodeID  string `json:"node_id"`
		Draft   bool   `json:"draft"`
		State   string `json:"state"`
		Base    struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return Epic11DraftPullRequestResult{}, fmt.Errorf("epic11 github client: decode response: %w", err)
	}

	return Epic11DraftPullRequestResult{
		Repository:                   m.Repository,
		Number:                       gh.Number,
		URL:                          gh.HTMLURL,
		GitHubResponseIDOrEquivalent: gh.NodeID,
		BaseRef:                      gh.Base.Ref,
		BaseSHA:                      gh.Base.SHA,
		HeadRef:                      gh.Head.Ref,
		HeadSHA:                      gh.Head.SHA,
		Draft:                        gh.Draft,
		State:                        gh.State,
		CreatedAt:                    gh.CreatedAt,
	}, nil
}

// PreflightHead fetches the remote head ref SHA and the base...head changed-file
// list via the GitHub REST API, so the caller can bind the approved head SHA and
// the dark-factory/ diff scope to the live remote before creating the PR. It
// performs no mutation.
func (c *Epic11GitHubPullRequestCreator) PreflightHead(ctx context.Context, m Epic11DraftPullRequestMutation) (Epic11RemoteHeadState, error) {
	if c.token == "" {
		return Epic11RemoteHeadState{}, fmt.Errorf("epic11 github client: empty token")
	}
	owner, repo, ok := strings.Cut(m.Repository, "/")
	if !ok {
		return Epic11RemoteHeadState{}, fmt.Errorf("epic11 github client: repository %q is not owner/repo", m.Repository)
	}

	var commit struct {
		SHA string `json:"sha"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s/commits/%s", c.baseURL, owner, repo, m.HeadRef), &commit); err != nil {
		return Epic11RemoteHeadState{}, fmt.Errorf("epic11 github client: head ref: %w", err)
	}

	var cmp struct {
		Files []struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s/compare/%s...%s", c.baseURL, owner, repo, m.BaseRef, m.HeadRef), &cmp); err != nil {
		return Epic11RemoteHeadState{}, fmt.Errorf("epic11 github client: compare: %w", err)
	}
	files := make([]string, 0, len(cmp.Files))
	for _, f := range cmp.Files {
		files = append(files, f.Filename)
	}
	return Epic11RemoteHeadState{HeadSHA: commit.SHA, ChangedFiles: files}, nil
}

// getJSON performs an authenticated GET and decodes a 200 response body into v.
func (c *Epic11GitHubPullRequestCreator) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var ghErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&ghErr)
		return fmt.Errorf("github returned %s: %s", resp.Status, ghErr.Message)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
