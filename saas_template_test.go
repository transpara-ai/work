package work_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestSaaSTemplateV1FilesCoverRequiredD1Surface(t *testing.T) {
	files := work.SaaSTemplateV1Files()
	byPath := map[string]string{}
	for _, file := range files {
		byPath[file.Path] = file.Content
	}

	requiredFiles := []string{
		"README.md",
		".env.example",
		"docker-compose.yml",
		"Makefile",
		"frontend/package.json",
		"frontend/playwright.config.ts",
		"frontend/app/api/login/route.ts",
		"frontend/app/login/page.tsx",
		"frontend/app/logout/route.ts",
		"frontend/app/dashboard/page.tsx",
		"backend/pyproject.toml",
		"backend/app/main.py",
		"backend/alembic/versions/0001_create_tracker_items.py",
		"backend/tests/test_auth_and_tracker.py",
		"scripts/migration-check.sh",
		"scripts/deploy-preview-dry-run.sh",
	}
	for _, path := range requiredFiles {
		if _, ok := byPath[path]; !ok {
			t.Fatalf("missing required generated file %s", path)
		}
	}

	assertContains(t, byPath["frontend/package.json"], `"next"`)
	assertContains(t, byPath["backend/pyproject.toml"], `"fastapi`)
	assertContains(t, byPath["docker-compose.yml"], "postgres:")
	assertContains(t, byPath["backend/alembic/versions/0001_create_tracker_items.py"], "create_table")
	assertContains(t, byPath[".env.example"], "DATABASE_URL=")
	assertContains(t, byPath["frontend/app/login/page.tsx"], "Login")
	assertContains(t, byPath["frontend/app/api/login/route.ts"], `response.cookies.set("session"`)
	assertContains(t, byPath["frontend/app/api/login/route.ts"], "process.env.SESSION_SECRET")
	assertContains(t, byPath["frontend/app/logout/route.ts"], "cookies.delete")
	assertContains(t, byPath["frontend/playwright.config.ts"], "baseURL")
	assertContains(t, byPath["frontend/playwright.config.ts"], "docker compose up --build")
	assertContains(t, byPath["frontend/lib/api.ts"], `fetch("/api/login"`)
	assertContains(t, byPath["frontend/app/dashboard/page.tsx"], "requireSession")
	assertContains(t, byPath["backend/app/main.py"], `Depends(require_session)`)
	assertContains(t, byPath["backend/app/main.py"], `@app.post("/api/tracker-items")`)
	assertContains(t, byPath["backend/app/main.py"], `@app.patch("/api/tracker-items/{item_id}")`)
	assertContains(t, byPath["backend/app/main.py"], `@app.delete("/api/tracker-items/{item_id}")`)
	assertContains(t, byPath["backend/app/auth.py"], `os.environ.get("SESSION_SECRET"`)
	assertContains(t, byPath["backend/tests/test_auth_and_tracker.py"], "test_tracker_crud_lifecycle_requires_session")
	assertContains(t, byPath["backend/tests/test_auth_and_tracker.py"], `client.post("/api/tracker-items", json={"title": "Nope"}).status_code == 401`)
	assertContains(t, byPath["Makefile"], "build:")
	assertContains(t, byPath["scripts/migration-check.sh"], ". ./.env")
	assertContains(t, byPath["scripts/migration-check.sh"], "alembic upgrade head --sql")
	assertContains(t, byPath["backend/alembic/env.py"], "context.is_offline_mode()")
	assertContains(t, byPath["scripts/deploy-preview-dry-run.sh"], "dry run only")
}

func TestSaaSTemplateV1ExcludesOutOfScopeProductionFeatures(t *testing.T) {
	joined := strings.ToLower(joinTemplateFiles(work.SaaSTemplateV1Files()))
	for _, forbidden := range []string{
		"stripe",
		"paypal",
		"chargebee",
		"terraform apply",
		"kubectl apply",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("generated template contains out-of-scope feature %q", forbidden)
		}
	}
	assertContains(t, joined, "no production deploy")
	assertContains(t, joined, "no external service is provisioned")
}

func TestGenerateSaaSTemplateV1WritesDeterministicRepo(t *testing.T) {
	dir := t.TempDir()
	manifest, err := work.GenerateSaaSTemplateV1(dir)
	if err != nil {
		t.Fatalf("GenerateSaaSTemplateV1: %v", err)
	}
	if manifest.TemplateID != work.SaaSTemplateV1ID {
		t.Fatalf("template ID = %q; want %q", manifest.TemplateID, work.SaaSTemplateV1ID)
	}
	if len(manifest.Files) != len(work.SaaSTemplateV1Files()) {
		t.Fatalf("manifest files = %d; want %d", len(manifest.Files), len(work.SaaSTemplateV1Files()))
	}

	for _, path := range manifest.Files {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path))); err != nil {
			t.Fatalf("generated file %s: %v", path, err)
		}
	}
	info, err := os.Stat(filepath.Join(dir, "scripts", "deploy-preview-dry-run.sh"))
	if err != nil {
		t.Fatalf("stat dry-run script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("dry-run script is not executable: mode %s", info.Mode().Perm())
	}

	otherDir := t.TempDir()
	otherManifest, err := work.GenerateSaaSTemplateV1(otherDir)
	if err != nil {
		t.Fatalf("GenerateSaaSTemplateV1 second run: %v", err)
	}
	if !slices.Equal(manifest.Files, otherManifest.Files) {
		t.Fatalf("second manifest differs\nfirst: %#v\nsecond: %#v", manifest.Files, otherManifest.Files)
	}
	for _, path := range manifest.Files {
		first, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read first generated %s: %v", path, err)
		}
		second, err := os.ReadFile(filepath.Join(otherDir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read second generated %s: %v", path, err)
		}
		if string(first) != string(second) {
			t.Fatalf("generated content for %s differs across runs", path)
		}
	}
}

func TestGenerateSaaSTemplateV1RejectsEmptyTarget(t *testing.T) {
	if _, err := work.GenerateSaaSTemplateV1(" "); err == nil {
		t.Fatal("GenerateSaaSTemplateV1 accepted an empty target")
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected content to contain %q", needle)
	}
}

func joinTemplateFiles(files []work.SaaSTemplateFile) string {
	var b strings.Builder
	for _, file := range files {
		b.WriteString(file.Path)
		b.WriteByte('\n')
		b.WriteString(file.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
