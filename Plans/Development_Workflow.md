# Self Systems — Development Workflow

> **Document Purpose:** Establishes development practices, testing strategy, and automation for the Self Systems project.
> **Status:** Confirmed decisions from Q71-Q80

**Last Updated:** April 8, 2026

---

## 1. Git Workflow & Version Control

### 1.1 Branching Strategy — GitHub Flow

**Decision:** GitHub Flow (Main-focused, feature branches)

**Branch Structure:**
```
main (production-ready, always deployable)
  ↑
  └─ feature/* (short-lived feature branches)
  └─ bugfix/* (short-lived bug fix branches)
  └─ hotfix/* (urgent production fixes)
  └─ docs/* (documentation updates)
  └─ ci/* (CI/CD improvements)
```

**Workflow:**
```
1. Create feature branch from main
   git checkout -b feature/ISSUE-123-add-graph-controls

2. Make commits with descriptive messages
   git commit -m "feat: add zoom controls to 3D graph"

3. Push and create Pull Request (self-review)
   git push origin feature/ISSUE-123-add-graph-controls

4. Automated tests run (GitHub Actions)
   - Go tests pass ✓
   - React tests pass ✓
   - Linting passes ✓
   - Build succeeds ✓

5. Merge to main (when tests pass)
   - Delete feature branch
   - Main always remains deployable

6. GitHub Actions auto-builds + releases
```

**Branch Naming Convention:**
```
feature/ISSUE-123-short-description
bugfix/ISSUE-456-short-description
docs/update-readme
ci/speed-up-tests
hotfix/ISSUE-789-critical-bug
```

**Why GitHub Flow:**
- ✅ Simple, easy to understand
- ✅ Scales from solo to team
- ✅ Continuous deployment friendly (Phase 2+)
- ✅ Minimal merge conflicts
- ✅ Works with automated releases

---

### 1.2 Code Review Process — Self-Review PR

**Decision:** Self-review PR (for accountability and documentation)

**PR Workflow:**
```
1. Create PR with template
2. Self-review your own changes
   - Check for logic errors
   - Verify naming conventions
   - Ensure no debug code left
3. Document the change (PR description)
4. Wait for automated checks
5. If all pass → merge
```

**PR Template (`.github/pull_request_template.md`):**
```markdown
## Description
Brief summary of what this PR does

## Type of Change
- [ ] Feature (new functionality)
- [ ] Bug fix (resolved issue)
- [ ] Refactor (code structure improved)
- [ ] Documentation (updated docs)
- [ ] Performance (improved speed/efficiency)
- [ ] Chore (dependencies, tooling)

## Changes
- Bullet list of specific changes made
- Include any breaking changes

## Testing Done
What tests were run to verify this works?
- [ ] Unit tests pass locally
- [ ] Integration tests pass
- [ ] Manual testing completed

## Screenshots (if UI changes)
Before/after images for UI modifications

## Checklist
- [ ] Code follows project style guide
- [ ] No console warnings/errors
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No hardcoded debug values left
- [ ] CHANGELOG entry added
```

**Self-Review Checklist:**
```
Before marking ready for merge:
- [ ] Did I test this locally?
- [ ] Would I understand this code in 6 months?
- [ ] Are variable/function names clear?
- [ ] Any TODOs left in?
- [ ] Any console.log() left in (Go/React)?
- [ ] Breaking changes documented?
- [ ] Tests adequate (new code coverage)?
```

**Scaling to Team:**
When team members added, change to:
- Require 1+ approvals before merge
- Request review from domain expert
- Add GitHub's CODEOWNERS file for automatic reviewers

---

### 1.3 Release Versioning — Semantic Versioning

**Decision:** Semantic Versioning (MAJOR.MINOR.PATCH)

**Version Format:**
```
v1.0.0
 │ │ └─ PATCH (bug fixes, internal improvements)
 │ └─── MINOR (new features, backward compatible)
 └───── MAJOR (breaking changes)

Examples:
v0.1.0 → v0.1.1 (patch: fixed bug)
v0.1.1 → v0.2.0 (minor: added feature)
v0.2.0 → v1.0.0 (major: breaking change)
```

**When to Increment:**

| Increment | When | Example |
|---|---|---|
| **PATCH** | Bug fixes, internal refactors | v1.0.0 → v1.0.1 |
| **MINOR** | New features, backward compatible | v1.0.0 → v1.1.0 |
| **MAJOR** | Breaking changes, incompatible | v1.0.0 → v2.0.0 |

**Release Process:**
```
1. Update version in code
   go/internal/version/version.go: const Version = "1.2.0"
   package.json: "version": "1.2.0"

2. Update CHANGELOG.md with all changes

3. Commit: git commit -m "chore: bump version to v1.2.0"

4. Tag: git tag v1.2.0

5. Push: git push origin main --tags

6. GitHub Actions automatically:
   - Builds binaries (Windows + Linux)
   - Creates GitHub Release
   - Generates checksums
   - Uploads artifacts
   - Notifies users
```

**CHANGELOG.md Format:**
```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [1.2.0] - 2026-04-15

### Added
- 3D graph zoom controls
- WebSocket real-time sync
- Search filtering system

### Fixed
- Memory leak in graph renderer
- Incorrect category count display
- Race condition in background processing

### Changed
- Improved graph performance with LOD rendering
- Updated dependencies

### Deprecated
- Old API format (use new format instead)

### Removed
- Legacy classification logic

### Security
- Updated to patched OpenSSL version

## [1.1.0] - 2026-03-20
...
```

---

## 2. Development Environment

### 2.1 Local Setup — Dev Container

**Decision:** VS Code Dev Container (one-click setup)

**Dev Container Configuration (`.devcontainer/devcontainer.json`):**
```json
{
  "name": "Self Systems",
  "image": "mcr.microsoft.com/devcontainers/go:1.22",
  "features": {
    "ghcr.io/devcontainers/features/node:18": {},
    "ghcr.io/devcontainers/features/docker-in-docker:latest": {}
  },
  "customizations": {
    "vscode": {
      "extensions": [
        "golang.go",
        "esbenp.prettier-vscode",
        "dbaeumer.vscode-eslint",
        "ms-azuretools.vscode-docker"
      ]
    }
  },
  "postCreateCommand": "make dev-setup",
  "forwardPorts": [8080, 3000, 6379, 8000, 9080],
  "portsAttributes": {
    "8080": { "label": "Go Backend", "onAutoForward": "notify" },
    "3000": { "label": "React Dev", "onAutoForward": "notify" },
    "6379": { "label": "Redis", "onAutoForward": "ignore" }
  }
}
```

**Setup Process:**
```
1. Clone repo
2. Open in VS Code
3. VS Code prompts: "Reopen in Container"
4. Click button → waits for container build
5. Container starts with:
   - Go 1.22 pre-installed
   - Node.js 18+ pre-installed
   - Docker daemon access
   - All extensions installed
   - Environment variables loaded
6. Run: make dev (starts all services)
7. Done!
```

**Makefile (simplifies tasks):**
```makefile
.PHONY: dev dev-setup test build clean lint

dev-setup:
	@echo "Installing dependencies..."
	go mod download
	npm install
	docker compose up -d

dev:
	@echo "Starting development environment..."
	docker compose up -d
	go run ./cmd/server &
	cd frontend && npm run dev

test:
	@echo "Running tests..."
	go test -v ./...
	npm test

lint:
	@echo "Running linters..."
	golangci-lint run
	npm run lint

build:
	@echo "Building for production..."
	wails build
	npm run build

clean:
	@echo "Cleaning up..."
	docker compose down
	rm -rf dist/
	rm -rf build/
```

**For Future Solo Setup (no container):**
- Just follow `README.md` manual setup
- Still works fine for Phase 1

---

### 2.2 Configuration Management — Hybrid Approach

**Decision:** Config files + Environment variable overrides

**Configuration Strategy:**
```
┌──────────────────────────────────────────────┐
│  1. Load defaults from config.yml            │
│         ↓ (database, AI, features defaults)  │
│  2. Override with .env (if exists)           │
│         ↓ (local secrets, API keys)          │
│  3. Override with environment variables      │
│         ↓ (production VPS overrides)         │
│  Result: Final configuration                 │
└──────────────────────────────────────────────┘
```

**Config Files:**

**`config.default.yml` (committed, in repo):**
```yaml
app:
  name: Self Systems
  version: 1.0.0
  debug: false

database:
  type: sqlite
  path: ./data.db

dgraph:
  host: localhost
  port: 8080

redis:
  host: localhost
  port: 6379

ai:
  providers:
    openai:
      enabled: true
      model: gpt-4o-mini
    anthropic:
      enabled: false

features:
  notifications: true
  behavioral_model: false
  proactive_discovery: false
```

**`.env` (gitignored, local development):**
```
# .env (in .gitignore)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
DEBUG_MODE=true
DATABASE_PATH=./data.db
LOG_LEVEL=debug
```

**Production Override (VPS environment):**
```bash
# On VPS, set before running:
export DATABASE_TYPE=postgresql
export DATABASE_URL=postgres://user:pass@db.example.com:5432/self_systems
export OPENAI_API_KEY=sk-prod-...
export DEBUG_MODE=false
export LOG_LEVEL=error
```

**Go Implementation:**
```go
import (
    "github.com/spf13/viper"
)

func LoadConfig() {
    // Set defaults from file
    viper.SetConfigName("config.default")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./config")
    viper.ReadInConfig()

    // Load .env overrides
    viper.SetConfigName(".env")
    viper.SetConfigType("ini")
    viper.MergeInConfig()

    // Env var overrides (prefix: APP_)
    viper.SetEnvPrefix("APP")
    viper.AutomaticEnv()

    // Access config
    dbPath := viper.GetString("database.path")
    apiKey := viper.GetString("ai.providers.openai.key")
}
```

**Directory Structure:**
```
self-systems/
├── .env.example         ← Template (committed)
├── .env                 ← Actual (gitignored)
├── config/
│   └── config.default.yml
├── config.prod.yml      ← Only on VPS
└── ...
```

---

## 3. Testing Strategy

### 3.1 Test Organization — Unit Adjacent + Integration Separate

**Decision:** Co-locate unit tests with source, separate integration/E2E

**Directory Structure:**
```
cmd/
└── server/
    ├── main.go
    └── main_test.go          ← Unit test for main

internal/
├── api/
│   ├── handlers.go
│   ├── handlers_test.go      ← Unit test
│   ├── middleware.go
│   └── middleware_test.go
├── service/
│   ├── classification.go
│   ├── classification_test.go
│   └── behavior_model.go
└── ...

test/
├── integration/              ← Tests requiring DB/service setup
│   ├── resource_flow_test.go
│   ├── classification_test.go
│   └── helpers.go
├── e2e/                       ← Full user flow tests
│   ├── add_resource_test.go
│   └── search_test.go
└── fixtures/                  ← Test data
    ├── resources.json
    └── categories.json

frontend/src/
├── components/
│   ├── Graph.tsx
│   └── Graph.test.tsx        ← Unit test
├── hooks/
│   ├── useNotifications.ts
│   └── useNotifications.test.ts
└── stores/
    ├── notificationStore.ts
    └── notificationStore.test.ts
```

**Running Tests by Type:**
```bash
# All tests
go test ./...
npm test

# Unit tests only (fast, just Unit Adjacent)
go test -short ./...
npm test --testPathPattern="(?<!integration)\.test\.tsx?$"

# Integration tests (requires containers)
go test -run TestIntegration ./test/integration
npm test test/integration

# E2E tests (requires running app)
npx playwright test test/e2e
```

---

### 3.2 Testing Pyramid — Complete Coverage

**Decision:** Complete Testing Pyramid (70% unit, 20% integration, 10% E2E)

**Test Statistics Target:**

```
Total Tests: ~250 tests
├── Unit Tests: 175 tests (70%)
│   ├── Go unit: 110 tests
│   └── React unit: 65 tests
├── Integration Tests: 50 tests (20%)
│   ├── Go (API + DB): 30 tests
│   └── React (with mock API): 20 tests
└── E2E Tests: 25 tests (10%)
    ├── Desktop app flows: 20 tests
    └── Critical paths: 5 tests

Coverage Targets:
├── Go unit + integration: 80%
├── React: 75%
└── E2E: Critical user flows (100%)
```

**Test Types:**

| Type | Framework | Examples | When |
|---|---|---|---|
| **Unit (Go)** | `testing` + `testify` | Handlers, services, utilities | Every commit |
| **Unit (React)** | Vitest + React Testing Library | Components, hooks, stores | Every commit |
| **Integration (Go)** | testify + Docker | API endpoints with real DB | Before release |
| **Integration (React)** | Vitest + MSW (mock API) | Multi-component flows | Before release |
| **E2E (Desktop)** | Playwright | User workflows in actual app | Before release |

**Testing Tools:**

**Go:**
```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

func TestExampleFunction(t *testing.T) {
    result := ExampleFunction()
    assert.Equal(t, expected, result)
}

// Integration test with Docker
type IntegrationTestSuite struct {
    suite.Suite
    db *sql.DB
}

func (suite *IntegrationTestSuite) SetupSuite() {
    // Start Docker container, create DB
}

func (suite *IntegrationTestSuite) TestAPIEndpoint() {
    // Test with real DB
}
```

**React:**
```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Graph } from './Graph';

describe('Graph Component', () => {
  it('should render nodes from data', () => {
    render(<Graph data={mockData} />);
    expect(screen.getByText('AI')).toBeInTheDocument();
  });

  it('should call onNodeClick when node clicked', async () => {
    const onClick = vi.fn();
    render(<Graph data={mockData} onNodeClick={onClick} />);
    
    await userEvent.click(screen.getByText('AI'));
    expect(onClick).toHaveBeenCalled();
  });
});
```

**E2E (Playwright):**
```typescript
import { test, expect } from '@playwright/test';

test('should allow user to add and search resource', async ({ page }) => {
  // Start with app open
  await page.goto('file://./dist/Self Systems.exe'); // or Linux binary

  // Add resource
  await page.click('button:has-text("+ Add")');
  await page.fill('[placeholder="URL"]', 'https://example.com');
  await page.fill('[placeholder="Title"]', 'Example');
  await page.selectOption('[name="category"]', 'AI');
  await page.click('button:has-text("Save")');

  // Search
  await page.fill('[placeholder="Search"]', 'Example');
  
  // Verify
  expect(await page.locator('[data-testid="resource-item"]').count()).toBe(1);
});
```

**Coverage Reporting:**
```bash
# Go
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# React
npm test -- --coverage

# Upload to code coverage service (optional)
# codecov -f coverage.out
```

---

## 4. CI/CD Automation

### 4.1 Continuous Integration Pipeline — Standard

**Decision:** Go tests + React tests + Lint + Build (GitHub Actions)

**GitHub Actions Workflow (`.github/workflows/ci.yml`):**
```yaml
name: CI

on:
  push:
    branches: [ main, develop, feature/*, bugfix/* ]
    paths-ignore:
      - "**.md"
      - "docs/**"
  pull_request:
    branches: [ main, develop ]

jobs:
  # Job 1: Check code formatting
  format:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: "1.22"
      - run: go fmt ./...
      - run: git diff --exit-code

  # Job 2: Lint Go code
  lint-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: amannn/action-semantic-pull-request@v5

      # Lint Go
      - uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  # Job 3: Lint React/TypeScript
  lint-react:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v3
        with:
          node-version: 20
      - run: npm ci
      - run: npm run lint

  # Job 4: Test Go
  test-go:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: "1.22"
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  # Job 5: Test React
  test-react:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v3
        with:
          node-version: 20
      - run: npm ci
      - run: npm test -- --coverage
      - uses: codecov/codecov-action@v3

  # Job 6: Build binaries
  build:
    runs-on: ubuntu-latest
    needs: [format, lint-go, lint-react, test-go, test-react]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: "1.22"
      - uses: actions/setup-node@v3
        with:
          node-version: 20
      - run: npm ci
      - run: wails build
      - uses: actions/upload-artifact@v3
        with:
          name: self-systems-windows
          path: build/bin/Self Systems.exe
      - run: GOOS=linux GOARCH=amd64 wails build
      - uses: actions/upload-artifact@v3
        with:
          name: self-systems-linux
          path: build/bin/self-systems

  # Job 7: Summarize
  results:
    if: always()
    needs: [format, lint-go, lint-react, test-go, test-react, build]
    runs-on: ubuntu-latest
    steps:
      - name: Check results
        run: |
          if [ "${{ needs.test-go.result }}" == "failure" ] || [ "${{ needs.test-react.result }}" == "failure" ]; then
            echo "Tests failed!"
            exit 1
          fi
          echo "All checks passed! ✓"
```

**Pipeline Duration:**
- Lint: ~1 min
- Tests: ~3-4 min
- Build: ~2-3 min
- **Total: ~6-8 minutes**

**Branch Protection Rules:**
In GitHub repo settings:
- ✅ Require status checks passing: CI must pass before merge
- ✅ Require branches up to date: Must be rebased on main
- ✅ Dismiss stale reviews: New commits dismiss old approvals
- ✅ Require code review: At least 1 approval (future: when team added)

---

### 4.2 Automated Release Pipeline — Build & Release

**Decision:** Automated release on version tag

**GitHub Actions Workflow (`.github/workflows/release.yml`):**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set version from tag
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_ENV
      
      # Build for Windows
      - uses: actions/setup-go@v4
        with:
          go-version: "1.22"
      - uses: actions/setup-node@v3
        with:
          node-version: 20
      
      - run: npm ci && wails build
        env:
          GOOS: windows
          GOARCH: amd64
      
      - name: Create checksums
        run: |
          cd build/bin
          sha256sum "Self Systems.exe" > checksums.txt
          sha256sum "Self Systems.exe" | cut -d ' ' -f 1 > windows-sha256.txt
      
      # Build for Linux
      - run: wails build
        env:
          GOOS: linux
          GOARCH: amd64
      
      - name: Create checksums
        run: |
          cd build/bin
          sha256sum self-systems >> checksums.txt
          sha256sum self-systems | cut -d ' ' -f 1 > linux-sha256.txt
      
      # Generate changelog
      - name: Generate changelog
        run: |
          git log $(git describe --tags --abbrev=0 HEAD~1 2>/dev/null || echo "HEAD~10")..HEAD --oneline > RELEASE_NOTES.md
      
      # Create GitHub Release
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            build/bin/Self Systems.exe
            build/bin/self-systems
            build/bin/checksums.txt
          body_path: RELEASE_NOTES.md
          draft: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      
      # Optional: Notify users
      - name: Send notification
        run: |
          echo "Release ${{ env.VERSION }} published!"
          # Could add Discord/Slack webhook here
```

**Release Process:**
```bash
# 1. Create tag locally (after merging PR)
git tag v1.2.0

# 2. Push tag (triggers GitHub Actions)
git push origin v1.2.0

# 3. GitHub Actions automatically:
# - Builds Windows binary
# - Builds Linux binary (amd64, arm64)
# - Generates SHA256 checksums
# - Creates GitHub Release
# - Uploads artifacts
# - Sends notification

# Result: Users can download from:
# https://github.com/username/self-systems/releases/v1.2.0
```

---

### 4.3 Code Quality Gates — Required Checks

**Decision:** Required checks (tests + lint must pass)

**GitHub Branch Protection Settings:**
```
Settings → Branches → main → Edit

☑ Require a pull request before merging
  ☑ Require status checks to pass before merging
    Required status checks:
    ✓ ci / format
    ✓ ci / lint-go
    ✓ ci / lint-react
    ✓ ci / test-go
    ✓ ci / test-react
    ✓ ci / build
  
  ☑ Require branches to be up to date before merging
  ☑ Dismiss stale pull request approvals when new commits...
```

**Code Quality Standards:**

| Check | Tool | Rule | Auto-fix |
|---|---|---|---|
| **Go Format** | gofmt | All code must be formatted | `go fmt ./...` |
| **Go Lint** | golangci-lint | 0 errors (warnings ok) | Manual |
| **TS/React Lint** | ESLint | No errors | `npm run lint -- --fix` |
| **Type Check** | TypeScript | 0 type errors | Manual |
| **Tests** | Go testing + Vitest | 80%+ coverage | N/A |
| **Build** | wails build | Must compile | N/A |

**Local Pre-commit Hook (optional):**
```bash
# .husky/pre-commit
#!/bin/sh

# Run linter on staged files
npm run lint -- --staged-files

# Run tests on changed files
go test -short ./...

# If anything fails, commit is blocked
```

**Installing pre-commit hooks:**
```bash
npm install husky --save-dev
npx husky install
npx husky add .husky/pre-commit "npm run lint && go test -short ./..."
```

---

## 5. Development Best Practices

### 5.1 Code Style & Conventions

**Go:**
```go
// Package comment
package classification

import (
    "context"
    
    "github.com/pkg/errors"
)

// ResourceClassifier handles resource classification
type ResourceClassifier struct {
    db ResourceRepository
}

// Classify takes a resource and returns its category
func (rc *ResourceClassifier) Classify(ctx context.Context, r *Resource) (string, error) {
    // Implementation
}
```

**React/TypeScript:**
```typescript
// components/Graph.tsx
import { memo, useCallback, useState } from 'react';
import { useGraphStore } from '@/stores/graphStore';

interface GraphProps {
  onNodeClick?: (nodeId: string) => void;
}

export const Graph = memo(function Graph({ onNodeClick }: GraphProps) {
  const [zoom, setZoom] = useState(1);
  const nodes = useGraphStore((s) => s.nodes);

  const handleZoom = useCallback((delta: number) => {
    setZoom((z) => Math.max(0.1, z + delta));
  }, []);

  return (
    <div>
      {/* Component JSX */}
    </div>
  );
});

Graph.displayName = 'Graph';
```

### 5.2 Error Handling

**Go:**
```go
// Wrap errors with context
if err != nil {
    return errors.Wrap(err, "failed to classify resource")
}

// Use custom error types for errors we can handle
type NotFoundError struct {
    ResourceID string
}

if _, err := rc.GetResource(id); err != nil {
    if _, ok := err.(*NotFoundError); ok {
        // Handle not found
    }
}
```

**React:**
```typescript
// Use error boundaries
import { ErrorBoundary } from 'react-error-boundary';

function ErrorFallback({error}) {
  return (
    <div role="alert">
      <p>Something went wrong:</p>
      <pre>{error.message}</pre>
    </div>
  )
}

<ErrorBoundary FallbackComponent={ErrorFallback}>
  <Graph />
</ErrorBoundary>
```

### 5.3 Logging

**Go:**
```go
import "github.com/go-logr/logr"

// Structured logging
log.Info("resource processed", "id", r.ID, "category", category)
log.Error(err, "failed to save resource", "resource_id", r.ID)
log.V(2).Info("debug info", "details", data) // Verbose logging
```

**React:**
```typescript
// Use console thoughtfully (not in production)
if (process.env.NODE_ENV === 'development') {
  console.log('Graph data updated:', data);
}
```

---

## 6. Documentation

**Required Documentation:**
- [ ] `README.md` — Project overview, setup instructions
- [ ] `CONTRIBUTING.md` — How to contribute
- [ ] `ARCHITECTURE.md` — System design overview
- [ ] `API.md` — Backend API documentation
- [ ] `TESTING.md` — How to run tests
- [ ] `DEPLOYMENT.md` — How to deploy

---

## 7. Future Enhancements (Phase 2+)

- [ ] Add security scanning (Snyk, Trivy)
- [ ] Performance benchmarking
- [ ] Automated dependency updates (Dependabot)
- [ ] Load testing pipeline
- [ ] Staging environment tests
- [ ] Browser compatibility testing (E2E cross-browser)

---

*Document will be updated as workflow evolves.*
