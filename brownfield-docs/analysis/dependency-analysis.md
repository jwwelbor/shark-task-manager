# Dependency Analysis

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 7 — Code Quality Analysis

## Dependency Count Summary

| Category | Count |
|----------|-------|
| Direct Go dependencies | 9 |
| Indirect Go dependencies | ~15 |
| Internal packages | 30 |
| Total dependency depth | 5 levels max |

## Dependency Freshness

| Dependency | Current Version | Status | Risk |
|------------|----------------|--------|------|
| mattn/go-sqlite3 | v1.14.32 | Current | Low |
| spf13/cobra | v1.10.2 | Current | Low |
| spf13/viper | v1.21.0 | Current | Low |
| pterm | v0.12.82 | Current | Low |
| testify | v1.11.1 | Current | Low |
| libsql-client-go | v0.0.0 (commit hash) | Pre-release | Medium |
| x/term | v0.32.0 | Current | Low |
| x/text | v0.28.0 | Current | Low |
| yaml.v3 | v3.0.1 | Stable | Low |

**Overall freshness**: Excellent — all tagged dependencies are at or near latest versions.

## Circular Dependencies

**None detected.** The internal package dependency graph is a strict DAG (directed acyclic graph). The layered architecture enforces this:
- Models → (no internal deps)
- Repository → models, db
- Services → models, repository interfaces, workflow
- CLI → services, models

## Unused Dependencies

No obviously unused dependencies detected in `go.mod`. All direct dependencies are imported in production code:
- `go-sqlite3`: `internal/db/db.go`
- `cobra`: `internal/cli/`
- `viper`: `internal/config/`
- `pterm`: `internal/cli/commands/`
- `testify`: `*_test.go` files
- `libsql-client-go`: `internal/db/` (Turso support)
- `x/term`, `x/text`: `internal/cli/`, `internal/slug/`
- `yaml.v3`: `internal/config/`, `internal/taskfile/`

## Heavy Transitive Dependencies

| Direct Dep | Notable Transitive | Concern |
|-----------|-------------------|---------|
| `libsql-client-go` | antlr4-go/antlr (parser), coder/websocket | Adds ~5MB to binary; pulls in ANTLR SQL parser |
| `viper` | spf13/afero, fsnotify, mapstructure | Filesystem abstraction overkill for JSON config |
| `pterm` | gookit/color, atomicgo/* (3 pkgs), fuzzysearch | Rich terminal feature set, most used |

## Internal Dependency Health

### Well-Structured Dependencies
- **`models`** — Zero internal dependencies (pure domain types)
- **`repository`** — Only depends on `models` and `db`
- **`workflow`** — Only depends on `config`

### Concerning Dependencies
- **`cli/commands`** imports **`repository`** directly — This is the legacy "fat controller" anti-pattern. ~40% of command files bypass the service layer and call repositories directly. Being addressed by E15 refactoring epic.
- **`services`** imports **`config`** — Services read workflow configuration, coupling business logic to config file format.

### Dependency Direction Violations

| From | To | Violation | Impact |
|------|----|-----------|--------|
| cli/commands | repository | Bypasses service layer | Duplicated business logic, harder to test |

This is the single most significant architectural debt item. The E15 epic is systematically extracting business logic from CLI commands into services.

## CGO Dependency

The `mattn/go-sqlite3` driver requires CGO (C compiler) for native builds with FTS5 support:
- **Linux**: CGO enabled, full SQLite features
- **macOS/Windows**: CGO disabled in releases, no FTS5
- **Impact**: Cross-compilation requires C cross-compilers (aarch64-linux-gnu-gcc for ARM64)
- **Alternative**: `modernc.org/sqlite` provides a pure-Go SQLite driver but may have different performance characteristics

---

See also: [Dependencies](../architecture/dependencies.md) | [Code Metrics](code-metrics.md)
