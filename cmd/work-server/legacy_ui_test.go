package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyBrowserUIHeadersAndNotice(t *testing.T) {
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
			if got := resp.Header.Get(legacyUIStatusHeader); got != "legacy" {
				t.Fatalf("%s = %q, want legacy", legacyUIStatusHeader, got)
			}
			if got := resp.Header.Get(legacyUIReplacementHeader); got != tc.wantReplacement {
				t.Fatalf("%s = %q, want %q", legacyUIReplacementHeader, got, tc.wantReplacement)
			}
			body := rec.Body.String()
			if !strings.Contains(body, "Legacy Work browser UI") {
				t.Fatalf("body missing legacy notice")
			}
			if !strings.Contains(body, tc.wantReplacement) {
				t.Fatalf("body missing replacement URL %q", tc.wantReplacement)
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

func TestLegacyBrowserUIEnabled(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WORK_LEGACY_BROWSER_UI", value)
			if !legacyBrowserUIEnabled() {
				t.Fatal("legacyBrowserUIEnabled() = false, want true")
			}
		})
	}

	t.Setenv("WORK_LEGACY_BROWSER_UI", "")
	if legacyBrowserUIEnabled() {
		t.Fatal("legacyBrowserUIEnabled() = true, want false")
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
