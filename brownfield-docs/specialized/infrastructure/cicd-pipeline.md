# CI/CD Pipeline Documentation

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 9 — Specialized Documentation

## Pipeline Overview

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
