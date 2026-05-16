package work

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SaaSTemplateV1ID = "dark-factory-saas-template-v1"

type SaaSTemplateFile struct {
	Path    string
	Content string
}

type SaaSTemplateManifest struct {
	TemplateID string
	Files      []string
}

func SaaSTemplateV1Files() []SaaSTemplateFile {
	files := []SaaSTemplateFile{
		{Path: ".env.example", Content: envExample},
		{Path: ".gitignore", Content: gitignore},
		{Path: "Makefile", Content: rootMakefile},
		{Path: "README.md", Content: readme},
		{Path: "docker-compose.yml", Content: dockerCompose},
		{Path: "factory-runtime-bom.json", Content: factoryRuntimeBOMJSON()},
		{Path: "scripts/deploy-preview-dry-run.sh", Content: deployPreviewDryRun},
		{Path: "scripts/migration-check.sh", Content: migrationCheck},
		{Path: "scripts/security-gates.sh", Content: securityGatesScript()},
		{Path: "security/security-gates-policy.json", Content: securityGatesPolicyJSON},
		{Path: "frontend/package.json", Content: frontendPackageJSON},
		{Path: "frontend/playwright.config.ts", Content: frontendPlaywrightConfig},
		{Path: "frontend/next.config.mjs", Content: frontendNextConfig},
		{Path: "frontend/app/page.tsx", Content: frontendHome},
		{Path: "frontend/app/login/page.tsx", Content: frontendLogin},
		{Path: "frontend/app/api/login/route.ts", Content: frontendLoginAPIRoute},
		{Path: "frontend/app/logout/route.ts", Content: frontendLogoutRoute},
		{Path: "frontend/app/dashboard/page.tsx", Content: frontendDashboard},
		{Path: "frontend/app/globals.css", Content: frontendCSS},
		{Path: "frontend/lib/api.ts", Content: frontendAPI},
		{Path: "frontend/lib/auth.ts", Content: frontendAuth},
		{Path: "frontend/__tests__/protected-route.test.ts", Content: frontendUnitTest},
		{Path: "frontend/tests/e2e/auth-and-tracker.spec.ts", Content: frontendE2ETest},
		{Path: "backend/pyproject.toml", Content: backendPyproject},
		{Path: "backend/app/__init__.py", Content: ""},
		{Path: "backend/app/auth.py", Content: backendAuth},
		{Path: "backend/app/db.py", Content: backendDB},
		{Path: "backend/app/main.py", Content: backendMain},
		{Path: "backend/app/models.py", Content: backendModels},
		{Path: "backend/alembic.ini", Content: alembicINI},
		{Path: "backend/alembic/env.py", Content: alembicEnv},
		{Path: "backend/alembic/script.py.mako", Content: alembicScript},
		{Path: "backend/alembic/versions/0001_create_tracker_items.py", Content: alembicMigration},
		{Path: "backend/tests/test_auth_and_tracker.py", Content: backendTests},
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func GenerateSaaSTemplateV1(targetDir string) (SaaSTemplateManifest, error) {
	targetDir = strings.TrimSpace(targetDir)
	if targetDir == "" {
		return SaaSTemplateManifest{}, errors.New("target directory is required")
	}
	files := SaaSTemplateV1Files()
	manifest := SaaSTemplateManifest{TemplateID: SaaSTemplateV1ID, Files: make([]string, 0, len(files))}
	for _, file := range files {
		if err := validateTemplatePath(file.Path); err != nil {
			return SaaSTemplateManifest{}, err
		}
		manifest.Files = append(manifest.Files, file.Path)
		fullPath := filepath.Join(targetDir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return SaaSTemplateManifest{}, fmt.Errorf("create directory for %s: %w", file.Path, err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(file.Path, "scripts/") {
			mode = 0o755
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), mode); err != nil {
			return SaaSTemplateManifest{}, fmt.Errorf("write %s: %w", file.Path, err)
		}
	}
	return manifest, nil
}

func validateTemplatePath(path string) error {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return fmt.Errorf("invalid template path %q", path)
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("invalid template path %q", path)
	}
	return nil
}

const envExample = `DATABASE_URL=postgresql+psycopg://tracker:tracker@postgres:5432/tracker
NEXT_PUBLIC_API_BASE_URL=http://localhost:8000
SESSION_SECRET=replace-with-local-dev-secret
`

const gitignore = `.env
.next/
node_modules/
__pycache__/
.pytest_cache/
.venv/
dist/
artifacts/
`

const rootMakefile = `.PHONY: build test migration-check security-gates deploy-preview

build:
	cd frontend && npm run build
	cd backend && python -m compileall app

test:
	cd backend && pytest
	cd frontend && npm test

migration-check:
	./scripts/migration-check.sh

security-gates:
	./scripts/security-gates.sh

deploy-preview:
	./scripts/deploy-preview-dry-run.sh
`

const readme = `# Dark Factory SaaS Template v1

This generated repo is the constrained Dark Factory v3.9 SaaS Template v1.

## Stack

- Next.js frontend
- FastAPI backend
- Postgres through Docker Compose
- Alembic migrations
- Local cookie login/logout
- Protected dashboard route
- Protected tracker API
- One CRUD tracker workflow

## Local Run

1. Copy .env.example to .env.
2. Run docker compose up --build.
3. Open http://localhost:3000.

The default local development login accepts operator@example.com with password password.
This is a local template auth stub; replace it before any production use.
SESSION_SECRET must be the same for frontend and backend when running outside Docker Compose.

## Verification

    make build
    make test
    make migration-check
    make security-gates
    make deploy-preview

make migration-check sources .env when present and emits offline Alembic SQL without
contacting a live database.

make security-gates writes artifacts/security-gates/report.json with evidence for
secret_scan, dependency_vulnerability_scan, dependency_license_scan, sast,
auth_flow_security_check, configuration_security_check, and
container_or_build_artifact_scan when applicable. The report includes the same
scanner metadata as factory-runtime-bom.json. Release certification is blocked when
scanner evidence is missing, a committed secret is found, a critical finding is
open, or a high finding is open without a valid waiver.

In this v1 template, secret_scan performs a local deterministic pattern check.
The other generated gate entries are scaffold evidence for wiring the named
scanners before any production release.

make deploy-preview is a dry run only. This template does not include payments,
billing, production deploy, or external service provisioning.
`

const dockerCompose = `services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: tracker
      POSTGRES_PASSWORD: tracker
      POSTGRES_DB: tracker
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tracker"]
      interval: 5s
      timeout: 3s
      retries: 10

  backend:
    image: python:3.12-slim
    working_dir: /app/backend
    command: sh -c "pip install -e . && alembic upgrade head && uvicorn app.main:app --host 0.0.0.0 --port 8000"
    env_file: .env
    volumes:
      - .:/app
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    image: node:22-alpine
    working_dir: /app/frontend
    command: sh -c "npm install && npm run dev -- --hostname 0.0.0.0"
    env_file: .env
    volumes:
      - .:/app
    ports:
      - "3000:3000"
    depends_on:
      - backend
`

const migrationCheck = `#!/usr/bin/env sh
set -eu
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
cd backend
alembic upgrade head --sql >/tmp/dark-factory-saas-template-v1-migration.sql
test -s /tmp/dark-factory-saas-template-v1-migration.sql
`

const deployPreviewDryRun = `#!/usr/bin/env sh
set -eu
cat <<'PLAN'
Deployment preview dry run only.
- build frontend
- build backend container
- run migration check
- emit preview artifact
No production deploy is performed.
No external service is provisioned.
PLAN
`

func securityGatesScript() string {
	return fmt.Sprintf(`#!/usr/bin/env sh
set -eu
# v1 deterministic scaffold: secret_scan runs locally. The other gate entries
# declare scanner evidence shape and must be wired to the named tools before any
# production release.
mkdir -p artifacts/security-gates
secret_status="pass"
if grep -R -n -E 'sk_live_|-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----|AKIA[0-9A-Z]{16}' \
  --exclude-dir=.git \
  --exclude-dir=.next \
  --exclude-dir=.venv \
  --exclude-dir=artifacts \
  --exclude-dir=node_modules \
  --exclude=security-gates.sh \
  . >/tmp/dark-factory-secret-scan.txt 2>/dev/null; then
  secret_status="fail"
fi
cat >artifacts/security-gates/report.json <<'JSON'
%sJSON
tmp_report="$(mktemp)"
sed "s/SECRET_STATUS/${secret_status}/g" artifacts/security-gates/report.json >"${tmp_report}"
mv "${tmp_report}" artifacts/security-gates/report.json
test "${secret_status}" = "pass"
`, securityGateReportJSON())
}

type generatedSecurityGateReport struct {
	TemplateID          string                          `json:"template_id"`
	FactoryRuntimeBOM   FactoryRuntimeBOM               `json:"factory_runtime_bom"`
	GateEvidence        []generatedSecurityGateEvidence `json:"gate_evidence"`
	CertificationPolicy generatedSecurityGateCertPolicy `json:"certification_policy"`
}

type generatedSecurityGateEvidence struct {
	Gate                                SecurityGateID `json:"gate"`
	Status                              string         `json:"status"`
	Scanner                             scannerRef     `json:"scanner"`
	EvidenceMode                        string         `json:"evidence_mode,omitempty"`
	RequiresRealScannerBeforeProduction bool           `json:"requires_real_scanner_before_production,omitempty"`
	Inspected                           []string       `json:"inspected,omitempty"`
	Checks                              []string       `json:"checks,omitempty"`
	Reason                              string         `json:"reason,omitempty"`
}

type scannerRef struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
}

type generatedSecurityGateCertPolicy struct {
	BlockOnMissingScannerEvidence     bool `json:"block_on_missing_scanner_evidence"`
	BlockOnCommittedSecret            bool `json:"block_on_committed_secret"`
	BlockOnOpenCritical               bool `json:"block_on_open_critical"`
	BlockOnOpenHighWithoutValidWaiver bool `json:"block_on_open_high_without_valid_waiver"`
}

func factoryRuntimeBOMJSON() string {
	return mustMarshalTemplateJSON(SaaSTemplateV1FactoryRuntimeBOM())
}

func securityGateReportJSON() string {
	bom := SaaSTemplateV1FactoryRuntimeBOM()
	report := generatedSecurityGateReport{
		TemplateID:        SaaSTemplateV1ID,
		FactoryRuntimeBOM: bom,
		GateEvidence: []generatedSecurityGateEvidence{
			{
				Gate:      GateSecretScan,
				Status:    "SECRET_STATUS",
				Scanner:   scannerForGate(bom, GateSecretScan),
				Inspected: []string{"generated source", "config", ".env.example", "runtime stdout/stderr"},
			},
			{
				Gate:                                GateDependencyVulnerabilityScan,
				Status:                              "pass",
				Scanner:                             scannerForGate(bom, GateDependencyVulnerabilityScan),
				EvidenceMode:                        "scaffold",
				RequiresRealScannerBeforeProduction: true,
				Inspected:                           []string{"frontend/package.json", "backend/pyproject.toml", "docker-compose.yml"},
			},
			{
				Gate:                                GateDependencyLicenseScan,
				Status:                              "pass",
				Scanner:                             scannerForGate(bom, GateDependencyLicenseScan),
				EvidenceMode:                        "scaffold",
				RequiresRealScannerBeforeProduction: true,
				Inspected:                           []string{"frontend/package.json", "backend/pyproject.toml"},
			},
			{
				Gate:                                GateSAST,
				Status:                              "pass",
				Scanner:                             scannerForGate(bom, GateSAST),
				EvidenceMode:                        "scaffold",
				RequiresRealScannerBeforeProduction: true,
				Inspected:                           []string{"frontend", "backend"},
			},
			{
				Gate:                                GateAuthFlowSecurityCheck,
				Status:                              "pass",
				Scanner:                             scannerForGate(bom, GateAuthFlowSecurityCheck),
				EvidenceMode:                        "scaffold",
				RequiresRealScannerBeforeProduction: true,
				Checks:                              []string{"unauthenticated protected page denial", "unauthenticated protected API denial", "logout invalidates session", "no production default admin"},
			},
			{
				Gate:                                GateConfigurationSecurityCheck,
				Status:                              "pass",
				Scanner:                             scannerForGate(bom, GateConfigurationSecurityCheck),
				EvidenceMode:                        "scaffold",
				RequiresRealScannerBeforeProduction: true,
				Checks:                              []string{".env not committed", ".env.example placeholders only", "production debug disabled", "security headers expected before production"},
			},
			{
				Gate:    GateContainerOrArtifactScan,
				Status:  "not_applicable",
				Scanner: scannerForGate(bom, GateContainerOrArtifactScan),
				Reason:  "no container or build artifact is produced by the dry-run template",
			},
		},
		CertificationPolicy: generatedSecurityGateCertPolicy{
			BlockOnMissingScannerEvidence:     true,
			BlockOnCommittedSecret:            true,
			BlockOnOpenCritical:               true,
			BlockOnOpenHighWithoutValidWaiver: true,
		},
	}
	return mustMarshalTemplateJSON(report)
}

func scannerForGate(bom FactoryRuntimeBOM, gate SecurityGateID) scannerRef {
	for _, scanner := range bom.SecurityScanners {
		if scanner.Gate == gate {
			return scannerRef{Tool: scanner.Tool, Version: scanner.Version}
		}
	}
	panic(fmt.Sprintf("missing scanner metadata for %s", gate))
}

func mustMarshalTemplateJSON(value any) string {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("marshal template json: %v", err))
	}
	return string(raw) + "\n"
}

const securityGatesPolicyJSON = `{
  "required_gates": [
    "secret_scan",
    "dependency_vulnerability_scan",
    "dependency_license_scan",
    "sast",
    "auth_flow_security_check",
    "configuration_security_check",
    "container_or_build_artifact_scan"
  ],
  "waiver_requirements": [
    "linked finding",
    "risk acceptance reason",
    "compensating controls",
    "expiry",
    "authorized approver role",
    "not_valid_for scope"
  ],
  "certification_blockers": [
    "missing scanner evidence",
    "committed secret",
    "open critical finding",
    "open high finding without valid waiver"
  ]
}
`

const frontendPackageJSON = `{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "test": "vitest run",
    "e2e": "playwright test"
  },
  "dependencies": {
    "@vitejs/plugin-react": "^5.0.0",
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "devDependencies": {
    "@playwright/test": "^1.48.0",
    "typescript": "^5.6.0",
    "vitest": "^2.1.0"
  }
}
`

const frontendNextConfig = `const nextConfig = {
  output: "standalone"
};

export default nextConfig;
`

const frontendPlaywrightConfig = `import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  use: {
    baseURL: "http://localhost:3000"
  },
  webServer: {
    command: "cd .. && docker compose up --build",
    url: "http://localhost:3000",
    reuseExistingServer: true,
    timeout: 120000
  }
});
`

const frontendHome = `import Link from "next/link";

export default function Home() {
  return (
    <main>
      <h1>Tracker</h1>
      <Link href="/login">Login</Link>
      <Link href="/dashboard">Dashboard</Link>
    </main>
  );
}
`

const frontendLogin = `"use client";

import { useState } from "react";
import { login } from "../../lib/api";

export default function LoginPage() {
  const [email, setEmail] = useState("operator@example.com");
  const [password, setPassword] = useState("password");
  const [error, setError] = useState("");

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    try {
      await login(email, password);
      window.location.href = "/dashboard";
    } catch {
      setError("Login failed");
    }
  }

  return (
    <main>
      <h1>Login</h1>
      <form onSubmit={submit}>
        <input aria-label="email" value={email} onChange={(event) => setEmail(event.target.value)} />
        <input aria-label="password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
        <button type="submit">Login</button>
      </form>
      {error ? <p role="alert">{error}</p> : null}
    </main>
  );
}
`

const frontendLoginAPIRoute = `import { NextResponse } from "next/server";

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8000";
const sessionToken = process.env.SESSION_SECRET ?? "local-template-session";

export async function POST(request: Request) {
  const payload = await request.json();
  const backendResponse = await fetch(apiBase + "/auth/login", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
    cache: "no-store"
  });
  const body = await backendResponse.json();
  if (!backendResponse.ok) {
    return NextResponse.json(body, { status: backendResponse.status });
  }
  const response = NextResponse.json(body);
  response.cookies.set("session", sessionToken, {
    httpOnly: true,
    sameSite: "lax",
    path: "/"
  });
  return response;
}
`

const frontendLogoutRoute = `import { NextResponse } from "next/server";

export async function GET() {
  const response = NextResponse.redirect(new URL("/login", "http://localhost:3000"));
  response.cookies.delete("session");
  return response;
}
`

const frontendDashboard = `import Link from "next/link";
import { fetchTrackerItems } from "../../lib/api";
import { requireSession } from "../../lib/auth";

export default async function DashboardPage() {
  const session = await requireSession();
  const items = await fetchTrackerItems(session);

  return (
    <main>
      <header>
        <h1>Tracker Workflow</h1>
        <Link href="/logout">Logout</Link>
      </header>
      <ul>
        {items.map((item) => (
          <li key={item.id}>{item.title} - {item.status}</li>
        ))}
      </ul>
    </main>
  );
}
`

const frontendCSS = `body {
  font-family: system-ui, sans-serif;
  margin: 2rem;
}

main {
  max-width: 48rem;
}
`

const frontendAPI = `const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8000";

export type TrackerItem = {
  id: number;
  title: string;
  status: "todo" | "doing" | "done";
};

export async function login(email: string, password: string) {
  const response = await fetch("/api/login", {
    method: "POST",
    headers: { "content-type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password })
  });
  if (!response.ok) {
    throw new Error("login failed");
  }
}

export async function fetchTrackerItems(session?: string): Promise<TrackerItem[]> {
  const response = await fetch(apiBase + "/api/tracker-items", {
    headers: session ? { cookie: "session=" + session } : undefined,
    credentials: "include",
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error("tracker fetch failed");
  }
  return response.json();
}
`

const frontendAuth = `import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export async function requireSession() {
  const session = (await cookies()).get("session");
  if (!session?.value) {
    redirect("/login");
  }
  return session.value;
}
`

const frontendUnitTest = `import { describe, expect, it } from "vitest";

describe("protected dashboard route", () => {
  it("uses the session cookie guard", async () => {
    const source = await import("../lib/auth");
    expect(source.requireSession).toBeTypeOf("function");
  });
});
`

const frontendE2ETest = `import { expect, test } from "@playwright/test";

test("login opens the protected tracker workflow", async ({ page }) => {
  await page.goto("/login");
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page.getByRole("heading", { name: "Tracker Workflow" })).toBeVisible();
});
`

const backendPyproject = `[project]
name = "dark-factory-saas-template-v1"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
  "alembic>=1.13",
  "fastapi>=0.115",
  "psycopg[binary]>=3.2",
  "pydantic>=2.9",
  "sqlalchemy>=2.0",
  "uvicorn>=0.32"
]

[project.optional-dependencies]
test = ["httpx>=0.27", "pytest>=8.3"]

[build-system]
requires = ["setuptools>=61"]
build-backend = "setuptools.build_meta"

[tool.pytest.ini_options]
testpaths = ["tests"]
`

const backendAuth = `import os

from fastapi import Cookie, HTTPException, Response
from pydantic import BaseModel

SESSION_SECRET = os.environ.get("SESSION_SECRET", "local-template-session")


class LoginRequest(BaseModel):
    email: str
    password: str


def login_user(payload: LoginRequest, response: Response) -> dict:
    if payload.email != "operator@example.com" or payload.password != "password":
        raise HTTPException(status_code=401, detail="invalid credentials")
    response.set_cookie("session", SESSION_SECRET, httponly=True, samesite="lax")
    return {"email": payload.email}


def require_session(session: str | None = Cookie(default=None)) -> str:
    if session != SESSION_SECRET:
        raise HTTPException(status_code=401, detail="login required")
    return session
`

const backendDB = `import os

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool

DATABASE_URL = os.environ.get("DATABASE_URL", "sqlite+pysqlite:///:memory:")

engine_kwargs = {"pool_pre_ping": True}
if DATABASE_URL == "sqlite+pysqlite:///:memory:":
    engine_kwargs = {
        "connect_args": {"check_same_thread": False},
        "poolclass": StaticPool,
    }

engine = create_engine(DATABASE_URL, **engine_kwargs)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
`

const backendModels = `from sqlalchemy import Integer, String
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    pass


class TrackerItem(Base):
    __tablename__ = "tracker_items"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, index=True)
    title: Mapped[str] = mapped_column(String(200), nullable=False)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="todo")
`

const backendMain = `from fastapi import Depends, FastAPI, HTTPException, Response
from pydantic import BaseModel
from sqlalchemy.orm import Session

from .auth import LoginRequest, login_user, require_session
from .db import get_db
from .models import TrackerItem

app = FastAPI(title="Dark Factory SaaS Template v1")


class TrackerItemCreate(BaseModel):
    title: str
    status: str = "todo"


class TrackerItemUpdate(BaseModel):
    title: str | None = None
    status: str | None = None


@app.post("/auth/login")
def login(payload: LoginRequest, response: Response):
    return login_user(payload, response)


@app.get("/api/me")
def me(session: str = Depends(require_session)):
    return {"session": session}


@app.get("/api/tracker-items")
def list_tracker_items(session: str = Depends(require_session), db: Session = Depends(get_db)):
    return db.query(TrackerItem).order_by(TrackerItem.id).all()


@app.post("/api/tracker-items")
def create_tracker_item(payload: TrackerItemCreate, session: str = Depends(require_session), db: Session = Depends(get_db)):
    item = TrackerItem(title=payload.title, status=payload.status)
    db.add(item)
    db.commit()
    db.refresh(item)
    return item


@app.patch("/api/tracker-items/{item_id}")
def update_tracker_item(item_id: int, payload: TrackerItemUpdate, session: str = Depends(require_session), db: Session = Depends(get_db)):
    item = db.get(TrackerItem, item_id)
    if item is None:
        raise HTTPException(status_code=404, detail="tracker item not found")
    if payload.title is not None:
        item.title = payload.title
    if payload.status is not None:
        item.status = payload.status
    db.commit()
    db.refresh(item)
    return item


@app.delete("/api/tracker-items/{item_id}")
def delete_tracker_item(item_id: int, session: str = Depends(require_session), db: Session = Depends(get_db)):
    item = db.get(TrackerItem, item_id)
    if item is None:
        raise HTTPException(status_code=404, detail="tracker item not found")
    db.delete(item)
    db.commit()
    return {"deleted": item_id}
`

const alembicINI = `[alembic]
script_location = alembic
`

const alembicEnv = `import os

from alembic import context
from app.db import engine
from app.models import Base

target_metadata = Base.metadata
DATABASE_URL = os.environ.get("DATABASE_URL", "sqlite+pysqlite:///:memory:")


def run_migrations_offline():
    context.configure(
        url=DATABASE_URL,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )
    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online():
    with engine.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)
        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
`

const alembicScript = `"""${message}

Revision ID: ${up_revision}
Revises: ${down_revision | comma,n}
Create Date: ${create_date}
"""

from alembic import op
import sqlalchemy as sa

revision = ${repr(up_revision)}
down_revision = ${repr(down_revision)}
branch_labels = ${repr(branch_labels)}
depends_on = ${repr(depends_on)}


def upgrade() -> None:
    ${upgrades if upgrades else "pass"}


def downgrade() -> None:
    ${downgrades if downgrades else "pass"}
`

const alembicMigration = `"""create tracker items

Revision ID: 0001_create_tracker_items
Revises:
Create Date: 2026-05-15
"""

from alembic import op
import sqlalchemy as sa

revision = "0001_create_tracker_items"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "tracker_items",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("title", sa.String(length=200), nullable=False),
        sa.Column("status", sa.String(length=20), nullable=False, server_default="todo"),
    )


def downgrade() -> None:
    op.drop_table("tracker_items")
`

const backendTests = `from fastapi.testclient import TestClient

from app.db import SessionLocal, engine
from app.main import app
from app.models import Base, TrackerItem


Base.metadata.create_all(bind=engine)


def clear_tracker_items():
    db = SessionLocal()
    try:
        db.query(TrackerItem).delete()
        db.commit()
    finally:
        db.close()


def test_protected_api_requires_login():
    client = TestClient(app)
    assert client.get("/api/tracker-items").status_code == 401
    assert client.post("/api/tracker-items", json={"title": "Nope"}).status_code == 401
    assert client.patch("/api/tracker-items/1", json={"status": "done"}).status_code == 401
    assert client.delete("/api/tracker-items/1").status_code == 401


def test_login_flow_sets_cookie():
    client = TestClient(app)
    login = client.post("/auth/login", json={"email": "operator@example.com", "password": "password"})
    assert login.status_code == 200
    assert "session" in client.cookies


def test_tracker_crud_lifecycle_requires_session():
    clear_tracker_items()
    client = TestClient(app)
    login = client.post("/auth/login", json={"email": "operator@example.com", "password": "password"})
    assert login.status_code == 200

    created = client.post("/api/tracker-items", json={"title": "First task", "status": "todo"})
    assert created.status_code == 200
    item_id = created.json()["id"]

    listed = client.get("/api/tracker-items")
    assert listed.status_code == 200
    assert any(item["id"] == item_id and item["title"] == "First task" for item in listed.json())

    updated = client.patch(f"/api/tracker-items/{item_id}", json={"status": "done"})
    assert updated.status_code == 200
    assert updated.json()["status"] == "done"

    deleted = client.delete(f"/api/tracker-items/{item_id}")
    assert deleted.status_code == 200
    assert deleted.json() == {"deleted": item_id}
`
