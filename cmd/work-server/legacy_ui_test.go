package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyBrowserUICannotBeReenabledOrExposeCredentials(t *testing.T) {
	t.Setenv("WORK_LEGACY_BROWSER_UI", "1")
	sv := &server{apiKey: "test-key", apiToken: "test-token"}
	cases := []struct {
		name            string
		req             *http.Request
		handler         http.HandlerFunc
		wantReplacement string
	}{
		{
			name:            "root dashboard",
			req:             httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/", nil),
			handler:         sv.dashboard,
			wantReplacement: "http://nucbuntu:8201/ops/work",
		},
		{
			name:            "telemetry dashboard",
			req:             httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/telemetry/", nil),
			handler:         sv.telemetryDashboard,
			wantReplacement: "http://nucbuntu:8201/ops/telemetry",
		},
		{
			name:            "workspace dashboard",
			req:             workspaceDashboardRequest("http://nucbuntu:8080/w/journey-test", "journey-test"),
			handler:         sv.workspaceDashboard,
			wantReplacement: "http://nucbuntu:8201/ops/work?workspace=journey-test",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, tc.req)
			resp := rec.Result()
			if resp.StatusCode != http.StatusFound {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
			}
			if got := resp.Header.Get(legacyUIStatusHeader); got != "disabled" {
				t.Fatalf("%s = %q, want disabled", legacyUIStatusHeader, got)
			}
			if got := resp.Header.Get(legacyUIReplacementHeader); got != tc.wantReplacement {
				t.Fatalf("%s = %q, want %q", legacyUIReplacementHeader, got, tc.wantReplacement)
			}
			if strings.Contains(rec.Body.String(), "test-key") || strings.Contains(rec.Body.String(), "test-token") || len(resp.Cookies()) != 0 {
				t.Fatal("retired UI exposed a credential")
			}
		})
	}
}

func TestLegacyBrowserUIRedirectsToSiteByDefault(t *testing.T) {
	sv := &server{apiKey: "test-key", apiToken: "test-token"}
	cases := []struct {
		name            string
		req             *http.Request
		handler         http.HandlerFunc
		wantReplacement string
	}{
		{
			name:            "root dashboard",
			req:             httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/", nil),
			handler:         sv.dashboard,
			wantReplacement: "http://nucbuntu:8201/ops/work",
		},
		{
			name:            "telemetry dashboard",
			req:             httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/telemetry/", nil),
			handler:         sv.telemetryDashboard,
			wantReplacement: "http://nucbuntu:8201/ops/telemetry",
		},
		{
			name:            "workspace dashboard",
			req:             workspaceDashboardRequest("http://nucbuntu:8080/w/journey-test", "journey-test"),
			handler:         sv.workspaceDashboard,
			wantReplacement: "http://nucbuntu:8201/ops/work?workspace=journey-test",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, tc.req)
			resp := rec.Result()
			if resp.StatusCode != http.StatusFound {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
			}
			if got := resp.Header.Get("Location"); got != tc.wantReplacement {
				t.Fatalf("Location = %q, want %q", got, tc.wantReplacement)
			}
			if got := resp.Header.Get(legacyUIStatusHeader); got != "disabled" {
				t.Fatalf("%s = %q, want disabled", legacyUIStatusHeader, got)
			}
			if got := resp.Header.Get(legacyUIReplacementHeader); got != tc.wantReplacement {
				t.Fatalf("%s = %q, want %q", legacyUIReplacementHeader, got, tc.wantReplacement)
			}
		})
	}
}

func TestAPIRoutesDoNotCarryLegacyUIHeaders(t *testing.T) {
	sv := &server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/health", nil)
	sv.health(rec, req)
	resp := rec.Result()
	if got := resp.Header.Get(legacyUIStatusHeader); got != "" {
		t.Fatalf("%s = %q, want empty", legacyUIStatusHeader, got)
	}
	if got := resp.Header.Get(legacyUIReplacementHeader); got != "" {
		t.Fatalf("%s = %q, want empty", legacyUIReplacementHeader, got)
	}
}

func TestAPIRejectsRetiredCookieCredential(t *testing.T) {
	sv := &server{apiKey: "test-key"}
	handler := sv.auth(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/tasks", nil)
	req.AddCookie(&http.Cookie{Name: "ws_key", Value: "test-key"})
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie credential status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestSiteUIURLCanBeOverridden(t *testing.T) {
	t.Setenv("SITE_UI_BASE_URL", "https://ops.example.test")
	req := httptest.NewRequest(http.MethodGet, "http://nucbuntu:8080/telemetry/", nil)
	if got := siteUIURL(req, "/ops/telemetry"); got != "https://ops.example.test/ops/telemetry" {
		t.Fatalf("siteUIURL = %q", got)
	}
}

func workspaceDashboardRequest(rawURL, workspace string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	req.SetPathValue("workspace", workspace)
	return req
}
