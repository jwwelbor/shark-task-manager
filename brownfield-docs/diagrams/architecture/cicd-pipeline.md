# CI/CD Pipeline

> Part of the Shark Task Manager Brownfield Analysis
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

See also: [Deployment Diagram](deployment-diagram.md) | [Infrastructure](../../specialized/infrastructure/cicd-pipeline.md)
