# Deployment Diagram

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## Deployment Topology

```mermaid
graph TB
    subgraph "Developer Machine"
        SHARK["shark CLI binary<br/>(~15-20MB Go binary)"]
        SQLITE[("shark-tasks.db<br/>SQLite + WAL")]
        CONFIG[".sharkconfig.json"]
        DOCS["docs/plan/**/*.md<br/>(entity files)"]
        TEMPLATES["shark-templates/<br/>(embedded in binary)"]
    end

    subgraph "Cloud (Optional)"
        TURSO[("Turso Cloud<br/>libsql://...turso.io")]
    end

    subgraph "GitHub"
        REPO["github.com/jwwelbor/<br/>shark-task-manager"]
        ACTIONS["GitHub Actions<br/>(CI + Release)"]
        RELEASES["GitHub Releases<br/>(multi-platform binaries)"]
    end

    SHARK --> SQLITE
    SHARK --> CONFIG
    SHARK --> DOCS
    SHARK -.->|optional| TURSO
    REPO --> ACTIONS
    ACTIONS --> RELEASES

    style TURSO stroke-dasharray: 5 5
```

## Distribution Channels

| Channel | Status | Platform |
|---------|--------|----------|
| GitHub Releases | Active | All (linux/darwin/windows, amd64/arm64) |
| `make install-shark` | Active | Local (builds from source) |
| `go install` | Available | Any with Go toolchain |
| Homebrew | Planned (disabled) | macOS |
| Scoop | Planned (disabled) | Windows |

## Runtime Requirements

| Requirement | Local SQLite | Turso Cloud |
|-------------|-------------|-------------|
| Network | None | Internet access |
| Filesystem | Read/write to project dir | Read/write to project dir |
| C compiler | Linux only (CGO) | Linux only (CGO) |
| Memory | ~50MB typical | ~50MB typical |
| Disk | ~20MB binary + DB | ~20MB binary |

---

See also: [CI/CD Pipeline](cicd-pipeline.md) | [System Overview](../../architecture/system-overview.md)
