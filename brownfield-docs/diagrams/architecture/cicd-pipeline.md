# CI/CD Pipeline

> Part of the Shark Task Manager Brownfield Analysis
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

See also: [Deployment Diagram](deployment-diagram.md) | [Infrastructure](../../specialized/infrastructure/cicd-pipeline.md)
