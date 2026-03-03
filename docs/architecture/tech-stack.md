# Tech Stack

**Project**: Shark Task Manager
**Generated**: 2026-03-02
**Track**: Brownfield

## Core Language

| Component | Version | Notes |
|-----------|---------|-------|
| **Go** | 1.23.4 (go.mod) | Statically typed, compiled |
| **Go Runtime** | 1.26.0 (installed) | Runtime on dev machines |

## Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework — command hierarchy, flags, lifecycle hooks |
| `github.com/spf13/viper` | v1.21.0 | Configuration — reads `.sharkconfig.json`, env vars |
| `github.com/mattn/go-sqlite3` | v1.14.32 | SQLite driver — CGo-based, FTS5 support via build tag |
| `github.com/tursodatabase/libsql-client-go` | v0.0.0-20251219 | Turso cloud SQLite — multi-machine database access |
| `github.com/pterm/pterm` | v0.12.82 | Terminal UI — colored output, tables, progress bars |
| `github.com/stretchr/testify` | v1.11.1 | Testing — assertion library (`assert`, `require`) |
| `golang.org/x/term` | v0.32.0 | Terminal control — raw mode, window size |
| `golang.org/x/text` | v0.28.0 | Text processing — unicode, slug generation |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML — task file frontmatter parsing |

## Database

### SQLite (Local — Default)

- **File**: `shark-tasks.db` (project root)
- **WAL Mode**: Write-Ahead Logging for concurrent reads
- **Foreign Keys**: Always enabled (`PRAGMA foreign_keys = ON`)
- **Cache**: 64MB in-memory (`cache_size = -64000`)
- **Busy Timeout**: 5 seconds
- **Memory-Mapped I/O**: 30GB mmap_size
- **Build Tag**: `-tags "fts5"` for full-text search
- **Configuration**: `internal/db/db.go`

### Turso (Cloud — Optional)

- **Protocol**: libsql over WebSocket
- **URL Format**: `libsql://shark-tasks-org.turso.io`
- **Auth**: Token file or `TURSO_AUTH_TOKEN` env var
- **Configuration**: `.sharkconfig.json` `database.backend: "turso"`
- **Use Case**: Multi-machine shared state

### Schema

| Table | Purpose |
|-------|---------|
| `epics` | Top-level organizational units |
| `features` | Features within epics |
| `tasks` | Atomic work items |
| `task_history` | Status change audit trail |
| `entity_notes` | Structured notes on entities |
| `task_relationships` | Task dependency graph |
| `documents` | Related document tracking |
| `work_sessions` | Development session tracking |
| `ideas` | Idea pipeline |

## Build & Quality Tools

| Tool | Version | Purpose | Command |
|------|---------|---------|---------|
| **Make** | System | Build orchestration | `make build`, `make shark` |
| **golangci-lint** | v2.9.0 | Static analysis | `make lint` |
| **gofmt** | Stdlib | Code formatting | `make fmt` |
| **go vet** | Stdlib | Static analysis | `make vet` |
| **air** | Latest | Hot-reload dev server | `make dev` |

### Build Flags

```bash
# Standard build
go build -tags "fts5" -ldflags "-X main.BuildDate=$(date) -X main.GitCommit=$(git rev-parse HEAD)"
```

### Quality Gate (Mandatory)

```bash
make fmt && make lint && make test
```

## CI/CD

- **Platform**: GitHub Actions (`.github/workflows/`)
- **Workflows**: `ci.yml` (test/build/lint), `release.yml`, `release-test.yml`
- **Test Matrix**: Ubuntu, macOS, Windows (amd64)
- **Coverage**: Uploaded to codecov.io

## Entry Points

| Binary | Source | Purpose |
|--------|--------|---------|
| `bin/shark` | `cmd/shark/` | Main CLI tool |
| `bin/shark-task-manager` | `cmd/server/` | HTTP API server |
| `bin/demo` | `cmd/demo/` | Interactive demo |
| `bin/test-db` | `cmd/test-db/` | Database integration tests |

## Configuration

- **Primary**: `.sharkconfig.json` (JSON, project root)
- **Sections**: database, workflow, status_metadata, templates, viewer
- **Auto-detection**: Project root found by walking up to `.sharkconfig.json`, `shark-tasks.db`, or `.git/`
- **Profiles**: Basic (5 statuses) or Advanced (19 statuses)

## Rationale

| Decision | Why |
|----------|-----|
| **Go** | Fast compilation, static typing, single binary deployment, strong stdlib |
| **SQLite** | Zero-config, embedded, excellent Go support, WAL for concurrency |
| **Cobra/Viper** | Industry-standard Go CLI framework, auto-generated help, config management |
| **pterm** | Rich terminal output without heavy dependencies |
| **Turso** | Cloud SQLite for team collaboration without infrastructure overhead |
| **Constructor DI** | Compile-time safe, no reflection, explicit dependencies |
