# CI/CD Pipeline Documentation

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
=======
> Generated: 2026-03-20
>>>>>>> Stashed changes
> Phase: 9 — Specialized Documentation

## Pipeline Overview

<<<<<<< Updated upstream
| Workflow | Trigger | Purpose | File |
|----------|---------|---------|------|
| CI | Push/PR | Test, build, lint | `.github/workflows/ci.yml` |
| Release | Tag `v*` | Cross-platform release | `.github/workflows/release.yml` |
| Release Test | Manual/PR | Validate release config | `.github/workflows/release-test.yml` |

## CI Pipeline Details

### Test Job
- **Runner**: ubuntu-latest
- **Timeout**: 10 minutes
- **Go version**: 1.23
- **Command**: `go test -v -p=1 ./...`
- **Coverage**: Uploaded to Codecov
- **Sequential**: `-p=1` prevents DB contention

### Build Job (Matrix)
- **Platforms**: ubuntu-latest, macos-latest, windows-latest
- **Architecture**: amd64
- **CGO**: Platform-specific (required for SQLite)
- **Tags**: `fts5` (Full-Text Search)
- **Verification**: Binary existence check post-build

### Lint Job
- **Tool**: golangci-lint (latest via GitHub Action)
- **Config**: `.golangci.yml`

### CI Success Gate
- **Requires**: All 3 jobs pass
- **Purpose**: Safe merge indicator for PRs

## Release Pipeline Details

### Prerequisites
- Tag matching `v*` pattern (e.g., `v1.0.0`, `v0.5.0-beta`)
- Full test suite passes (quality gate)

### GoReleaser Configuration

**Builds**:
```yaml
builds:
  - main: ./cmd/shark
    binary: shark
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
    tags: [fts5]
    ldflags:
      - -s -w
      - -X main.BuildDate={{.Date}}
      - -X main.GitCommit={{.ShortCommit}}
```

**Cross-Compilation**:
- Linux amd64: Native CGO
- Linux arm64: Cross-compiler (`aarch64-linux-gnu-gcc`)
- macOS: `CGO_ENABLED=0` (pure Go SQLite fallback)
- Windows: `CGO_ENABLED=0`

**Archives**:
- Format: `shark_{Version}_{Os}_{Arch}`
- Linux/macOS: `.tar.gz`
- Windows: `.zip`
- Includes: `LICENSE`, `README.md`

**Release**:
- Type: Draft (manual publish)
- Changelog: Auto-generated from commits
- Excludes: docs, test, chore, ci commits

### Planned (Disabled)
- **Homebrew tap**: `jwwelbor/homebrew-tap` (needs `HOMEBREW_TAP_TOKEN`)
- **Scoop bucket**: `jwwelbor/scoop-shark` (needs `SCOOP_BUCKET_TOKEN`)

## Environment Configuration

| Environment | Purpose | Database | CI |
|-------------|---------|----------|-----|
| Development | Local dev | Local SQLite | N/A |
| CI | Automated testing | Temp SQLite (in-memory) | GitHub Actions |
| Release | Binary distribution | N/A | GoReleaser |
| Production (local) | User's machine | Local SQLite | N/A |
| Production (cloud) | Multi-machine | Turso cloud | N/A |

## Build Artifacts

| Artifact | Size | Format |
|----------|------|--------|
| shark (Linux amd64) | ~25MB | ELF binary |
| shark (Linux arm64) | ~25MB | ELF binary |
| shark (macOS amd64) | ~20MB | Mach-O binary |
| shark (macOS arm64) | ~20MB | Mach-O binary |
| shark.exe (Windows) | ~20MB | PE binary |
| checksums.txt | ~1KB | SHA256 hashes |

## Quality Gate Enforcement

```
Push/PR → CI Pipeline
           ├── Tests pass? (go test -p=1)
           ├── Builds compile? (3 platforms)
           └── Lint clean? (golangci-lint)
                    ↓
              All pass → Merge allowed

Tag push → Release Pipeline
           ├── Tests pass? (quality gate)
           └── GoReleaser → Draft release
```

See also: [Deployment Diagram](../diagrams/architecture/deployment-diagram.md) | [CI/CD Pipeline Diagram](../diagrams/architecture/cicd-pipeline.md)
=======
| Property | Value |
|----------|-------|
| **Platform** | GitHub Actions |
| **Workflows** | 3 (CI, Release, Release Test) |
| **CI Trigger** | Push to any branch + PR to main |
| **Release Trigger** | Git tag push (v*) |
| **Build Tool** | GoReleaser v2 |

## Workflow 1: CI (`.github/workflows/ci.yml`)

### Trigger
- Push to any branch
- Pull request to main

### Jobs

| Job | Runner | Purpose | Timeout |
|-----|--------|---------|---------|
| **test** | ubuntu-latest | Run full test suite + coverage | 10min |
| **build** | ubuntu/macOS/windows | Verify cross-platform build | 10min |
| **lint** | ubuntu-latest | golangci-lint static analysis | 10min |
| **ci-success** | ubuntu-latest | Gate job (requires all above) | — |

### Build Matrix
| OS | Architecture | CGO |
|----|-------------|-----|
| ubuntu-latest | amd64 | Enabled |
| macos-latest | amd64 | Disabled |
| windows-latest | amd64 | Disabled |

### Test Configuration
- Go 1.23 with module cache
- Sequential test execution (`-p=1`) for database isolation
- Coverage uploaded to Codecov (non-blocking)

## Workflow 2: Release (`.github/workflows/release.yml`)

Uses GoReleaser v2 to build and publish multi-platform binaries.

### Build Targets

| OS | Arch | CGO | Cross-Compiler |
|----|------|-----|-----------------|
| linux | amd64 | Enabled | Native |
| linux | arm64 | Enabled | aarch64-linux-gnu-gcc |
| darwin | amd64 | Disabled | — |
| darwin | arm64 | Disabled | — |
| windows | amd64 | Disabled | — |

### Release Artifacts
- Binary archives (.tar.gz for unix, .zip for windows)
- SHA256 checksums (`checksums.txt`)
- GitHub Release (created as draft)
- Includes LICENSE and README.md

### Disabled Distribution Channels
- **Homebrew tap** — Planned but repository not set up
- **Scoop bucket** — Planned but repository not set up

## Workflow 3: Release Test (`.github/workflows/release-test.yml`)

Validates GoReleaser configuration without publishing, used for testing release pipeline changes.

## Build Configuration

### Makefile Targets

| Target | Command | Description |
|--------|---------|-------------|
| `build` | `go build -tags "fts5" ...` | Build all binaries |
| `shark` | `go build -tags "fts5" ...` | Build CLI only |
| `install-shark` | Copy to go/bin or existing location | Install globally |
| `test` | `go test -tags "fts5" -v -p=1 -parallel=1 ./...` | Run tests |
| `test-coverage` | Same + `-coverprofile=coverage.out` | Tests with coverage |
| `lint` | `golangci-lint run` | Static analysis |
| `fmt` | `go fmt ./...` | Format code |
| `vet` | `go vet ./...` | Go vet checks |
| `dev` | `air` | Hot reload development |
| `clean` | `rm -rf bin/ *.db ...` | Clean all artifacts |

### Build Flags
```
-tags "fts5"                        # Enable SQLite FTS5
-ldflags "-X main.Version=..."      # Version injection
-ldflags "-X main.BuildDate=..."    # Build date
-ldflags "-X main.GitCommit=..."    # Git commit hash
-ldflags "-s -w"                    # Strip debug (release only)
```

## Quality Gates

Mandatory before any code change is considered complete:
1. `make fmt` — Format all Go code
2. `make lint` — golangci-lint static analysis
3. `make test` — Full test suite

CI enforces these same gates on every push.

---

See also: [CI/CD Pipeline Diagram](../../diagrams/architecture/cicd-pipeline.md) | [Deployment Diagram](../../diagrams/architecture/deployment-diagram.md)
>>>>>>> Stashed changes
