package work_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/transpara-ai/work"
)

func TestSaaSTemplateV1FilesCoverRequiredSurface(t *testing.T) {
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
		"frontend/package-lock.json",
		"frontend/playwright.config.ts",
		"frontend/app/api/login/route.ts",
		"frontend/app/login/page.tsx",
		"frontend/app/logout/route.ts",
		"frontend/app/dashboard/page.tsx",
		"backend/pyproject.toml",
		"backend/requirements.lock.txt",
		"backend/app/main.py",
		"backend/alembic/versions/0001_create_tracker_items.py",
		"backend/tests/test_auth_and_tracker.py",
		"scripts/migration-check.sh",
		"scripts/security-gates.sh",
		"scripts/deploy-preview-dry-run.sh",
		"factory-runtime-bom.json",
		"security/security-gates-policy.json",
	}
	for _, path := range requiredFiles {
		if _, ok := byPath[path]; !ok {
			t.Fatalf("missing required generated file %s", path)
		}
	}

	assertContains(t, byPath["frontend/package.json"], `"next"`)
	assertContains(t, byPath["frontend/package.json"], `"postcss": "8.5.15"`)
	assertContains(t, byPath["frontend/package-lock.json"], `"lockfileVersion": 3`)
	assertContains(t, byPath["frontend/package-lock.json"], `"postcss"`)
	assertContains(t, byPath["backend/pyproject.toml"], `"fastapi`)
	assertContains(t, byPath["backend/requirements.lock.txt"], "fastapi==0.136.3")
	assertContains(t, byPath["backend/requirements.lock.txt"], "pytest==9.0.3")
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
	assertContains(t, byPath["Makefile"], "security-gates:")
	assertContains(t, byPath["scripts/migration-check.sh"], ". ./.env")
	assertContains(t, byPath["scripts/migration-check.sh"], "alembic upgrade head --sql")
	assertContains(t, byPath["backend/alembic/env.py"], "context.is_offline_mode()")
	assertContains(t, byPath["scripts/security-gates.sh"], "artifacts/security-gates/report.json")
	assertContains(t, byPath["scripts/security-gates.sh"], `"secret_scan"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"dependency_vulnerability_scan"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"dependency_license_scan"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"sast"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"auth_flow_security_check"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"configuration_security_check"`)
	assertContains(t, byPath["scripts/security-gates.sh"], `"container_or_build_artifact_scan"`)
	assertContains(t, byPath["factory-runtime-bom.json"], `"security_scanners"`)
	assertContains(t, byPath["factory-runtime-bom.json"], `"version": "8.18.4"`)
	assertContains(t, byPath["security/security-gates-policy.json"], "open high finding without valid waiver")
	assertContains(t, byPath["README.md"], "make security-gates")
	assertContains(t, byPath["README.md"], "scaffold evidence")
	assertContains(t, byPath["scripts/deploy-preview-dry-run.sh"], "dry run only")
}

func TestSaaSTemplateV1GeneratedBOMMatchesScannerMetadata(t *testing.T) {
	byPath := map[string]string{}
	for _, file := range work.SaaSTemplateV1Files() {
		byPath[file.Path] = file.Content
	}
	var generated work.FactoryRuntimeBOM
	if err := json.Unmarshal([]byte(byPath["factory-runtime-bom.json"]), &generated); err != nil {
		t.Fatalf("unmarshal generated factory-runtime-bom.json: %v", err)
	}
	if !reflect.DeepEqual(generated, work.SaaSTemplateV1FactoryRuntimeBOM()) {
		t.Fatalf("generated BOM drifted from scanner metadata\ngot:  %#v\nwant: %#v", generated, work.SaaSTemplateV1FactoryRuntimeBOM())
	}
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

func TestGeneratedSaaSTemplateV1SecurityGatesCommandWritesEvidence(t *testing.T) {
	dir := t.TempDir()
	if _, err := work.GenerateSaaSTemplateV1(dir); err != nil {
		t.Fatalf("GenerateSaaSTemplateV1: %v", err)
	}

	cmd := exec.Command(filepath.Join(dir, "scripts", "security-gates.sh"))
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("security-gates.sh failed: %v\n%s", err, output)
	}

	reportPath := filepath.Join(dir, "artifacts", "security-gates", "report.json")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read security gate report: %v", err)
	}
	reportText := string(report)
	assertContains(t, reportText, `"gate": "secret_scan"`)
	assertContains(t, reportText, `"gate": "dependency_vulnerability_scan"`)
	assertContains(t, reportText, `"gate": "dependency_license_scan"`)
	assertContains(t, reportText, `"gate": "sast"`)
	assertContains(t, reportText, `"gate": "auth_flow_security_check"`)
	assertContains(t, reportText, `"gate": "configuration_security_check"`)
	assertContains(t, reportText, `"gate": "container_or_build_artifact_scan"`)
	assertContains(t, reportText, `"security_gate_version": "dark-factory-v3.9-d2-security-gates"`)
	assertContains(t, reportText, `"block_on_open_high_without_valid_waiver": true`)
	assertContains(t, reportText, `"evidence_mode": "scaffold"`)
	assertContains(t, reportText, `"requires_real_scanner_before_production": true`)
}

func TestGeneratedSaaSTemplateV1SecurityGatesCommandFailsOnCommittedSecret(t *testing.T) {
	dir := t.TempDir()
	if _, err := work.GenerateSaaSTemplateV1(dir); err != nil {
		t.Fatalf("GenerateSaaSTemplateV1: %v", err)
	}
	secretPath := filepath.Join(dir, "backend", "app", "leaked_key.txt")
	if err := os.WriteFile(secretPath, []byte("AWS_ACCESS_KEY_ID=AKIAABCDEFGHIJKLMNOP\n"), 0o644); err != nil {
		t.Fatalf("write planted secret: %v", err)
	}

	cmd := exec.Command(filepath.Join(dir, "scripts", "security-gates.sh"))
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("security-gates.sh passed with planted secret\n%s", output)
	}

	reportPath := filepath.Join(dir, "artifacts", "security-gates", "report.json")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read security gate report: %v", err)
	}
	assertContains(t, string(report), `"gate": "secret_scan"`)
	assertContains(t, string(report), `"status": "fail"`)
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
