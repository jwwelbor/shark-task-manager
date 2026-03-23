# CI/CD Pipeline

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## CI Pipeline (`.github/workflows/ci.yml`)

```mermaid
flowchart TD
    subgraph Trigger["Triggers"]
        PUSH["Push (any branch)"]
        PR["PR to main"]
    end

    subgraph TestJob["Test Job (ubuntu-latest)"]
        CHECKOUT1["Checkout code"]
        SETUP1["Setup Go 1.23"]
        CACHE1["Cache Go modules"]
        RUN_TEST["go test -v -p=1 ./..."]
        COVERAGE["Upload to Codecov"]
    end

    subgraph BuildJob["Build Job (Matrix)"]
        direction TB
        B_LINUX["Linux amd64"]
        B_MACOS["macOS amd64"]
        B_WIN["Windows amd64"]
        VERIFY["Verify binaries"]
    end

    subgraph LintJob["Lint Job (ubuntu-latest)"]
        CHECKOUT3["Checkout code"]
        SETUP3["Setup Go 1.23"]
        GOLINT["golangci-lint (latest)"]
    end

    subgraph Gate["CI Success Gate"]
        ALL_PASS{All jobs passed?}
        SUCCESS["CI Success"]
        FAILURE["CI Failure"]
    end

    PUSH & PR --> TestJob & BuildJob & LintJob
    TestJob --> ALL_PASS
    BuildJob --> ALL_PASS
    LintJob --> ALL_PASS
    ALL_PASS -->|Yes| SUCCESS
    ALL_PASS -->|No| FAILURE

    CHECKOUT1 --> SETUP1 --> CACHE1 --> RUN_TEST --> COVERAGE
    B_LINUX & B_MACOS & B_WIN --> VERIFY
```

## Release Pipeline (`.github/workflows/release.yml`)

```mermaid
flowchart TD
    subgraph Trigger["Trigger"]
        TAG["Push tag v*<br/>(e.g., v1.0.0)"]
    end

    subgraph QualityGate["Quality Gate"]
        TEST["Full Test Suite"]
    end

    subgraph Release["Release Job"]
        CHECKOUT["Checkout code"]
        SETUP_GO["Setup Go 1.23"]
        SETUP_CC["Setup cross-compilers<br/>(arm64-linux-gnu-gcc)"]
        GORELEASER["GoReleaser v2"]
    end

    subgraph Artifacts["Output"]
        LINUX_AMD["shark_*_linux_amd64.tar.gz"]
        LINUX_ARM["shark_*_linux_arm64.tar.gz"]
        MACOS_AMD["shark_*_darwin_amd64.tar.gz"]
        MACOS_ARM["shark_*_darwin_arm64.tar.gz"]
        WIN["shark_*_windows_amd64.zip"]
        CHECKSUMS["checksums.txt"]
        GH_RELEASE["GitHub Release (draft)"]
    end

    TAG --> TEST
    TEST -->|Pass| Release
    CHECKOUT --> SETUP_GO --> SETUP_CC --> GORELEASER
    GORELEASER --> LINUX_AMD & LINUX_ARM & MACOS_AMD & MACOS_ARM & WIN & CHECKSUMS
    GORELEASER --> GH_RELEASE
```

## Build Configuration Details

| Setting | Value |
|---------|-------|
| Go version | 1.23 |
| Test parallelism | `-p=1` (sequential for DB safety) |
| Build tags | `fts5` (SQLite full-text search) |
| LDFLAGS | `-s -w -X main.BuildDate -X main.GitCommit` |
| Linter | golangci-lint v2.9.0 |
| Release tool | GoReleaser v2.x |
| Release type | Draft (manual publish) |
=======
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## GitHub Actions CI Pipeline

```mermaid
graph LR
    subgraph "Trigger"
        PUSH["Push to any branch"]
        PR["PR to main"]
    end

    subgraph "CI Jobs (parallel)"
        TEST["Tests<br/>ubuntu-latest<br/>Go 1.23"]
        BUILD["Build Matrix<br/>linux/mac/win<br/>amd64"]
        LINT["Lint<br/>golangci-lint<br/>latest"]
    end

    subgraph "Gate"
        GATE["CI Passed<br/>(requires all 3)"]
    end

    PUSH --> TEST & BUILD & LINT
    PR --> TEST & BUILD & LINT
    TEST --> GATE
    BUILD --> GATE
    LINT --> GATE
```

### CI Job Details

| Job | Runner | Go Version | Steps |
|-----|--------|------------|-------|
| **Test** | ubuntu-latest | 1.23 | checkout → setup-go → download deps → run tests → upload coverage (codecov) |
| **Build** | ubuntu/macOS/windows | 1.23 | checkout → setup-go → build shark binary → verify binary |
| **Lint** | ubuntu-latest | 1.23 | checkout → setup-go → golangci-lint |
| **CI Passed** | ubuntu-latest | — | Gate job: fails if any above fails |

Source: `.github/workflows/ci.yml`

## Release Pipeline

```mermaid
graph TD
    TAG["Git Tag Push<br/>(v*)"] --> RELEASE["GoReleaser<br/>(release.yml)"]

    RELEASE --> BUILD_LINUX["Build<br/>linux/amd64<br/>(CGO=1)"]
    RELEASE --> BUILD_LINUX_ARM["Build<br/>linux/arm64<br/>(CGO=1, cross-compiler)"]
    RELEASE --> BUILD_MAC["Build<br/>darwin/amd64+arm64<br/>(CGO=0)"]
    RELEASE --> BUILD_WIN["Build<br/>windows/amd64<br/>(CGO=0)"]

    BUILD_LINUX --> ARCHIVE["Archives<br/>.tar.gz / .zip"]
    BUILD_LINUX_ARM --> ARCHIVE
    BUILD_MAC --> ARCHIVE
    BUILD_WIN --> ARCHIVE

    ARCHIVE --> CHECKSUM["SHA256 Checksums"]
    ARCHIVE --> DRAFT["GitHub Release<br/>(draft)"]
    CHECKSUM --> DRAFT

    DRAFT --> REVIEW["Manual Review"]
    REVIEW --> PUBLISH["Publish Release"]
```

### Release Configuration

| Setting | Value | Source |
|---------|-------|--------|
| **Tool** | GoReleaser v2 | `.goreleaser.yml` |
| **Platforms** | linux, darwin, windows | amd64 + arm64 (except win/arm64) |
| **CGO** | Enabled for linux only | macOS/Windows: CGO=0 |
| **Binary name** | `shark` | |
| **Archive format** | .tar.gz (unix), .zip (windows) | GoReleaser default |
| **Includes** | LICENSE, README.md | |
| **Release type** | Draft (manual publish) | |
| **Homebrew** | Disabled (tap not set up) | |
| **Scoop** | Disabled (bucket not set up) | |

### Build Flags

```
-s -w                           # Strip debug symbols
-X main.Version={{.Version}}    # Inject version from git tag
-X main.BuildDate=YYYY-MM-DD   # Build date (Makefile only)
-X main.GitCommit=SHORT_HASH   # Git commit (Makefile only)
```

---
>>>>>>> Stashed changes

See also: [Deployment Diagram](deployment-diagram.md) | [Infrastructure](../../specialized/infrastructure/cicd-pipeline.md)
