# CI/CD Pipeline Documentation

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 9 — Specialized Documentation

## Pipeline Overview

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
