# Deployment Diagram

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## Deployment Topology

```mermaid
graph TB
    subgraph DevMachine["Developer Machine"]
        SHARK["shark binary<br/>(compiled Go, ~25MB)"]
        SQLDB["shark-tasks.db<br/>(SQLite + WAL)"]
        CONFIG[".sharkconfig.json"]
        DOCS["docs/plan/<br/>(Markdown files)"]
        TEMPLATES["shark-templates/<br/>(80+ .md templates)"]
    end

    subgraph GitHub["GitHub"]
        REPO["Repository<br/>(source code)"]
        ACTIONS["GitHub Actions<br/>(CI/CD)"]
        RELEASES["GitHub Releases<br/>(binaries)"]
    end

    subgraph TursoCloud["Turso Cloud (Optional)"]
        TURSO["libsql://shark-tasks-*.turso.io<br/>(cloud SQLite)"]
    end

    SHARK --> SQLDB
    SHARK --> CONFIG
    SHARK --> DOCS
    SHARK --> TEMPLATES
    SHARK -.->|"Optional<br/>(multi-machine sync)"| TURSO

    REPO --> ACTIONS
    ACTIONS -->|"GoReleaser"| RELEASES
    RELEASES -->|"Download"| SHARK

    style TURSO stroke-dasharray: 5 5
```

## Build & Release Pipeline

```mermaid
flowchart LR
    subgraph Trigger["Triggers"]
        PUSH["Push to branch"]
        TAG["Push tag v*"]
        PR["Pull Request"]
    end

    subgraph CI["CI Pipeline"]
        TEST["Test<br/>(go test -p=1)"]
        BUILD["Build<br/>(3 platforms)"]
        LINT["Lint<br/>(golangci-lint)"]
    end

    subgraph Release["Release Pipeline"]
        RTEST["Test Gate"]
        GORELEASER["GoReleaser v2<br/>(cross-compile)"]
        GHRELEASE["GitHub Release<br/>(draft)"]
    end

    subgraph Artifacts["Release Artifacts"]
        LIN["Linux amd64/arm64<br/>(.tar.gz)"]
        MAC["macOS amd64/arm64<br/>(.tar.gz)"]
        WIN["Windows amd64<br/>(.zip)"]
        CHECKSUMS["checksums.txt<br/>(SHA256)"]
    end

    PUSH & PR --> TEST & BUILD & LINT
    TAG --> RTEST --> GORELEASER --> GHRELEASE
    GHRELEASE --> LIN & MAC & WIN & CHECKSUMS
```

## Platform Support Matrix

| Platform | Architecture | CGO | Build Status |
|----------|-------------|-----|-------------|
| Linux | amd64 | Yes (native) | Primary |
| Linux | arm64 | Yes (cross-compiler) | Supported |
| macOS | amd64 | No (CGO_ENABLED=0) | Supported |
| macOS | arm64 | No (CGO_ENABLED=0) | Supported |
| Windows | amd64 | No (CGO_ENABLED=0) | Supported |
| Windows | arm64 | - | Not supported |

See also: [CI/CD Pipeline](cicd-pipeline.md) | [System Overview](../../architecture/system-overview.md)
