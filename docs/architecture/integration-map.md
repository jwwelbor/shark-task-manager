# Integration Map

**Project**: Shark Task Manager
**Generated**: 2026-03-02

## Data Stores

### SQLite (Local — Default)

| Property | Value |
|----------|-------|
| **File** | `shark-tasks.db` (project root) |
| **Driver** | `github.com/mattn/go-sqlite3` v1.14.32 |
| **Mode** | WAL (Write-Ahead Logging) |
| **Config** | `internal/db/db.go` |
| **Connection** | `sqlite3://shark-tasks.db?_foreign_keys=on` |
| **Build Tag** | `-tags "fts5"` (full-text search) |

**PRAGMAs**:
- `foreign_keys = ON` — referential integrity
- `journal_mode = WAL` — concurrent reads during writes
- `busy_timeout = 5000` — 5s lock wait
- `synchronous = NORMAL` — safety/performance balance
- `cache_size = -64000` — 64MB cache
- `temp_store = MEMORY` — in-memory temp tables
- `mmap_size = 30000000000` — memory-mapped I/O

**Tables**: epics, features, tasks, task_history, entity_notes, task_relationships, documents, work_sessions, ideas

### Turso Cloud (Optional)

| Property | Value |
|----------|-------|
| **Protocol** | libsql over WebSocket |
| **Driver** | `github.com/tursodatabase/libsql-client-go` |
| **URL Format** | `libsql://shark-tasks-org.turso.io` |
| **Auth** | Token file or `TURSO_AUTH_TOKEN` env var |
| **Config** | `.sharkconfig.json` → `database.backend: "turso"` |
| **Setup** | `shark cloud init --url=... --auth-token=...` |

## File System

### Entity Files

```
docs/plan/
  └── {epic-key}/
        ├── epic.md                    (Epic description)
        └── {feature-key}/
              ├── feature.md           (Feature description)
              └── tasks/
                    └── {task-key}.md  (YAML frontmatter + Markdown)
```

- **Sync**: Filesystem → Database (unidirectional)
- **Status**: Database-only (never synced from files)
- **Format**: YAML frontmatter (`---` delimited) + Markdown body
- **Writer**: `internal/fileops/writer.go` (atomic writes)

### Configuration

| File | Purpose | Format |
|------|---------|--------|
| `.sharkconfig.json` | Project configuration | JSON |
| `shark-tasks.db` | SQLite database | Binary |
| `shark-tasks.db-wal` | Write-Ahead Log | Binary |
| `shark-tasks.db-shm` | Shared memory | Binary |

### Templates

| Location | Purpose |
|----------|---------|
| `shark-data/` | Optional editable content bundle |
| `shark-data/workflow/` | Per-entity workflow YAML |
| `shark-data/prompts/` | Status prompts by entity type |
| `shark-data/prompts/_partials/` | Shared prompt partials |
| `shark-data/file_templates/` | Markdown skeletons for created epic, feature, task, and sprint files |
| `shark-data/skills/` | Skill instructions and supporting references |
| `shark-data/agents/` | Agent role definitions |

## API Boundaries

### CLI Interface (`cmd/shark/`)

- **Entry**: `cmd/shark/main.go`
- **Root**: `internal/cli/root.go`
- **Commands**: `internal/cli/commands/` (auto-registered via `init()`)
- **Global Flags**: `--json`, `--field`, `--no-color`, `--verbose`, `--db`, `--config`
- **Exit Codes**: 0 (success), 1 (not found), 2 (DB error), 3 (invalid state)
- **Service Access**: `internal/cli/services_global.go` (lazy init)

### HTTP API (`cmd/server/`)

- **Entry**: `cmd/server/main.go`
- **Wiring**: `cmd/server/services.go` (`WireServices()`)
- **Status**: Minimal implementation (health check)
- **Design**: Service-injected handlers, JSON responses
- **Pattern**: Same services as CLI, different presentation layer

## External Tools

### Build & Quality

| Tool | Integration | Config |
|------|-------------|--------|
| **golangci-lint** v2.9.0 | `make lint` (auto-installs) | `.golangci.yml` |
| **gofmt** | `make fmt` | Go stdlib |
| **go vet** | `make vet` | Go stdlib |
| **air** | `make dev` (hot reload) | `.air.toml` |

### CI/CD (GitHub Actions)

| Workflow | Triggers | Jobs |
|----------|----------|------|
| `ci.yml` | Push, PR to main | Test, Build (matrix), Lint, CI-Success |
| `release.yml` | Release tags | Build + publish |
| `release-test.yml` | Manual | Release verification |

**Test Matrix**: Ubuntu, macOS, Windows (amd64)
**Coverage**: codecov.io integration

### Database Tools

| Tool | Purpose |
|------|---------|
| `sqlite3` CLI | Direct database inspection |
| `turso` CLI | Cloud database management (create, tokens, shell) |

## Internal Service Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│ CLI Commands / HTTP Handlers                                 │
│ (parse input, format output)                                │
├─────────────────────────────────────────────────────────────┤
│ Services (business logic, validation, orchestration)         │
│ TaskService │ FeatureService │ EpicService │ NoteService     │
│ ContextSvc  │ ResumeSvc      │ WorkflowSvc │ StatusCalcSvc   │
├─────────────────────────────────────────────────────────────┤
│ Repositories (data access only)                              │
│ TaskRepo │ FeatureRepo │ EpicRepo │ NoteRepo │ HistoryRepo   │
├─────────────────────────────────────────────────────────────┤
│ Database (SQLite / Turso)                                    │
└─────────────────────────────────────────────────────────────┘
```

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `TURSO_AUTH_TOKEN` | Turso cloud authentication | (none) |

## Project Root Detection

Shark auto-detects project root by walking up directories looking for:
1. `.sharkconfig.json` (primary)
2. `shark-tasks.db` (secondary)
3. `.git/` (fallback)

All commands work from any subdirectory within the project.
