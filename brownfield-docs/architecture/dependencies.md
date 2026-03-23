# Dependencies

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
=======
> Generated: 2026-03-20
>>>>>>> Stashed changes
> Phase: 2 — Architecture Analysis

## Internal Dependencies

<<<<<<< Updated upstream
```mermaid
graph TD
    CLI["cli/commands"] --> SVC["services"]
    CLI --> CLIFW["cli (framework)"]
    CLIFW --> REPO["repository"]
    CLIFW --> WF["workflow"]
    CLIFW --> CFG["config"]
    SVC --> REPO
    SVC --> WF
    SVC --> MODELS["models"]
    SVC --> CREATOR["taskcreation"]
    SVC --> STATUS["status"]
    REPO --> DB["db"]
    REPO --> MODELS
    DB --> CFG
    CREATOR --> TMPL["templates"]
    CREATOR --> FOPS["fileops"]
    CREATOR --> KEYGEN["keygen"]
    CREATOR --> SLUG["slug"]
    WF --> CFG
    STATUS --> CFG
    STATUS --> REPO
    DISC["discovery"] --> PARSER["parser"]
    DISC --> PATTERNS["patterns"]
    SYNC["sync"] --> DISC
    SYNC --> REPO
=======
### Package Dependency Graph

```mermaid
graph TD
    CLI["cli/commands"] --> CLI_FW["cli (framework)"]
    CLI --> SVC["services"]
    CLI --> MODELS["models"]
    CLI --> REPO["repository"]
    CLI_FW --> SVC
    CLI_FW --> REPO
    CLI_FW --> WF["workflow"]
    CLI_FW --> CFG["config"]
    CLI_FW --> DB["db"]

    SVC --> MODELS
    SVC --> REPO
    SVC --> WF
    SVC --> CFG
    SVC --> TC["taskcreation"]
    SVC --> FILEOPS["fileops"]

    REPO --> MODELS
    REPO --> DB

    WF --> CFG
    CFG --> MODELS

    TC --> MODELS
    TC --> KEYGEN["keygen"]
    TC --> SLUG["slug"]
    TC --> TMPL["templates"]

    STATUS["status"] --> MODELS
    STATUS --> REPO
    STATUS --> CFG

    INIT["init"] --> CFG
    INIT --> DB
    INIT --> PROFILES["init/profiles"]

    DISC["discovery"] --> MODELS
    DISC --> PATTERNS["patterns"]
    DISC --> PARSER["parser"]

    TASKFILE["taskfile"] --> MODELS
    TASKFILE --> PARSER

    VALID["validation"] --> MODELS
    VALID --> WF

    REPORT["reporting"] --> MODELS
    REPORT --> REPO

    subgraph "Core Domain"
        MODELS
        SVC
        REPO
    end

    subgraph "Infrastructure"
        DB
        WF
        CFG
    end

    subgraph "Entry Points"
        CLI
        CLI_FW
    end

    subgraph "Utilities"
        TC
        KEYGEN
        SLUG
        FILEOPS
        TMPL
        PARSER
        PATTERNS
        VALID
    end
>>>>>>> Stashed changes
```

### Dependency Matrix

<<<<<<< Updated upstream
| Source | Target | Type | Reason |
|--------|--------|------|--------|
| cli/commands | services | Compile | Business logic calls |
| cli | repository | Compile | Legacy direct access (being migrated) |
| cli | workflow | Compile | Workflow service accessor |
| cli | config | Compile | Configuration loading |
| services | repository | Compile | Data access via interfaces |
| services | workflow | Compile | Status transition validation |
| services | models | Compile | Domain types |
| services | taskcreation | Compile | Task file/key creation |
| services | status | Compile | Progress calculation |
| repository | db | Compile | Database connection |
| repository | models | Compile | Entity types |
| db | config | Compile | Cloud backend selection |
| workflow | config | Compile | Workflow configuration |
| taskcreation | templates | Compile | Entity template rendering |
| taskcreation | fileops | Compile | Atomic file writing |
| taskcreation | keygen | Compile | Key generation |
| discovery | parser | Compile | Markdown parsing |
| discovery | patterns | Compile | Entity key patterns |
| sync | discovery | Compile | Filesystem scanning |
| sync | repository | Compile | Database updates |

## External Dependencies

### Direct Dependencies (9)

| Dependency | Version | Purpose | Health |
|-----------|---------|---------|--------|
| `github.com/mattn/go-sqlite3` | v1.14.32 | SQLite driver (CGO) | Active, well-maintained |
| `github.com/tursodatabase/libsql-client-go` | v0.0.0-20251219 | Turso cloud DB client | Dev commit, not stable release |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework | Active, industry standard |
| `github.com/spf13/viper` | v1.21.0 | Configuration management | Active, widely used |
| `github.com/pterm/pterm` | v0.12.82 | Terminal UI/output | Active, feature-rich |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | Active, standard Go testing |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing | Stable, widely used |
| `golang.org/x/term` | v0.32.0 | Terminal control | Official Go extension |
| `golang.org/x/text` | v0.28.0 | Unicode handling | Official Go extension |

### Notable Indirect Dependencies

| Dependency | Via | Purpose |
|-----------|-----|---------|
| `github.com/antlr4-go/antlr/v4` | libsql-client-go | SQL parsing for Turso |
| `github.com/coder/websocket` | libsql-client-go | WebSocket for Turso |
| `github.com/spf13/afero` | viper | Filesystem abstraction |
| `github.com/fsnotify/fsnotify` | viper | Config file watching |
| `atomicgo.dev/cursor` | pterm | Terminal cursor control |
| `atomicgo.dev/keyboard` | pterm | Keyboard input |
| `github.com/gookit/color` | pterm | ANSI color output |

### Dependency Concerns

| Concern | Dependency | Severity | Detail |
|---------|-----------|----------|--------|
| Unstable version | libsql-client-go | Medium | Using dev commit (v0.0.0-...), not a tagged release |
| CGO requirement | go-sqlite3 | Low | Requires C compiler for cross-compilation |
| Large transitive tree | viper | Low | Pulls in afero, fsnotify, mapstructure, etc. |

## Build Dependencies

| Tool | Version | Purpose |
|------|---------|---------|
| golangci-lint | v2.9.0 | Static analysis (auto-installed) |
| GoReleaser | v2.x | Cross-platform release builds |
| Air | latest | Hot-reload development server |
=======
| Source Package | Depends On |
|---------------|-----------|
| **cli/commands** | cli, services, models, repository, config, workflow, formatters, pterm |
| **cli (framework)** | services, repository, workflow, config, db, pathresolver |
| **services** | models, repository (interfaces), workflow, config, taskcreation, fileops |
| **repository** | models, db |
| **workflow** | config |
| **config** | models, viper |
| **status** | models, repository, config |
| **init** | config, db, init/profiles |
| **discovery** | models, patterns, parser |
| **taskcreation** | models, keygen, slug, templates |
| **taskfile** | models, parser |
| **validation** | models, workflow |
| **templates** | models, embed.FS |

### Notable Dependency Characteristics

1. **No circular dependencies** — Dependency graph is a strict DAG
2. **models has zero internal dependencies** — Pure domain types, imports only stdlib
3. **repository depends only on models + db** — Clean data access layer
4. **services depend on interfaces, not concrete repos** — Repository interfaces defined in service package
5. **CLI commands import services AND repositories** — Legacy fat controller pattern being refactored (E15)

---

## External Dependencies

### Direct Dependencies (from `go.mod`)

| Package | Version | Purpose | Maintenance Status |
|---------|---------|---------|-------------------|
| `github.com/mattn/go-sqlite3` | v1.14.32 | SQLite driver (CGO) | Active, widely used |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework | Active, industry standard |
| `github.com/spf13/viper` | v1.21.0 | Configuration management | Active |
| `github.com/pterm/pterm` | v0.12.82 | Terminal output (tables, colors) | Active |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | Active, industry standard |
| `github.com/tursodatabase/libsql-client-go` | v0.0.0-20251219 | Turso cloud SQLite | Active (pre-release hash) |
| `golang.org/x/term` | v0.32.0 | Terminal interaction | Go team maintained |
| `golang.org/x/text` | v0.28.0 | Unicode text processing | Go team maintained |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing | Stable |

### Indirect Dependencies (Notable)

| Package | Version | Pulled By | Notes |
|---------|---------|-----------|-------|
| `github.com/antlr4-go/antlr/v4` | v4.13.0 | libsql-client-go | Parser generator for SQL |
| `github.com/coder/websocket` | v1.8.12 | libsql-client-go | WebSocket for Turso protocol |
| `github.com/fsnotify/fsnotify` | v1.9.0 | viper | File watching |
| `github.com/gookit/color` | v1.5.4 | pterm | Color output |
| `github.com/lithammer/fuzzysearch` | v1.1.8 | pterm | Fuzzy search in interactive menus |
| `github.com/spf13/afero` | v1.15.0 | viper | Filesystem abstraction |

### Dependency Health Assessment

| Dependency | Health | Risk | Notes |
|------------|--------|------|-------|
| go-sqlite3 | Green | Low | Mature, stable, v1.14.32 is recent |
| cobra | Green | Low | v1.10.2, widely adopted |
| viper | Green | Low | v1.21.0, actively maintained |
| pterm | Green | Low | v0.12.82, active development |
| testify | Green | Low | v1.11.1, industry standard |
| libsql-client-go | Yellow | Medium | Pre-release (commit hash), API may change |
| x/term | Green | Low | Go team maintained |
| x/text | Green | Low | Go team maintained |
| yaml.v3 | Green | Low | Stable, v3.0.1 |

### Dependency Concerns

1. **libsql-client-go** uses a commit hash version (`v0.0.0-20251219100830-236aa1ff8acc`), not a tagged release. This means:
   - API may change without semantic versioning guarantees
   - Harder to track changelogs and security updates
   - Should be pinned to a tagged release when available

2. **CGO dependency** via go-sqlite3 — Requires C compiler for native builds, complicates cross-compilation. macOS/Windows builds disable CGO (no FTS5 support on those platforms).

3. **Total dependency count**: 9 direct + ~15 indirect — Relatively lean for a Go project.

---
>>>>>>> Stashed changes

See also: [System Overview](system-overview.md) | [Components](components.md) | [Patterns](patterns.md)
