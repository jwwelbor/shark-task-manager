# Complexity Analysis

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 7 — Code Quality Analysis

## Complexity Hotspots

### Files Exceeding 1,000 LOC

| File | LOC | Concern Level | Notes |
|------|-----|--------------|-------|
| `internal/db/db.go` | 2,335 | High | Schema + init + migrations + integrity check in one file |
| `internal/repository/task_repository.go` | 1,806 | Medium | Many query methods, but each is simple SQL |
| `internal/db/migrate.go` | 1,560 | Medium | Expected growth as migrations accumulate |
| `internal/services/task_service.go` | 1,371 | Medium | Central service, well-decomposed methods |
| `internal/cli/commands/feature_helpers.go` | 1,209 | High | Output formatting complexity |
| `internal/cli/commands/config.go` | 1,075 | Medium | Config subcommand handlers |
| `internal/services/feature_service.go` | 1,063 | Medium | Feature business logic |
| `internal/repository/feature_repository.go` | 995 | Low | Standard repository, approaching threshold |
| `internal/cli/commands/epic_helpers.go` | 994 | High | Output formatting complexity |
| `internal/services/epic_service.go` | 923 | Low | Epic business logic |

### Potential God Objects

1. **`internal/db/db.go` (2,335 LOC)** — Combines schema creation, SQLite configuration, migration orchestration, schema versioning, and integrity checking. Should be split into:
   - `schema.go` (table definitions)
   - `config.go` (SQLite PRAGMAs)
   - `version.go` (schema versioning)

2. **`internal/cli/commands/feature_helpers.go` (1,209 LOC)** and **`epic_helpers.go` (994 LOC)** — Output formatting helpers that grow with each new display feature. Could benefit from a shared formatting framework.

### Deeply Nested Logic

- **Workflow validation**: Multi-level config resolution (task → feature → epic level) with fallback chains
- **Key parsing**: Entity type detection from key format involves multiple regex patterns and fallback strategies
- **Status transition**: Validation chain: get entity → validate transition → check force flag → create rejection note → update status → check unblocks

### Duplication Patterns

The CLI commands layer shows significant structural duplication across entity types:
- `epic_helpers.go` (994 LOC) and `feature_helpers.go` (1,209 LOC) contain parallel output formatting logic
- `epic_context.go`, `feature_context.go`, `task_context.go` share similar patterns
- `epic_note.go`, `feature_note.go`, `task_note.go` are structurally similar
- `epic_resume.go`, `feature_resume.go`, `task_resume.go` follow same pattern
- **E21 (Entity Polymorphism)** is actively addressing this duplication

### File Count by Complexity Category

| Complexity | File Count | Description |
|------------|-----------|-------------|
| **Simple** (< 100 LOC) | ~80 | Utility files, small commands, type definitions |
| **Moderate** (100-500 LOC) | ~140 | Standard services, repositories, commands |
| **Complex** (500-1000 LOC) | ~30 | Feature-rich services, large repositories |
| **Very Complex** (1000+ LOC) | 16 | Schema, migrations, central services |

## Package Complexity Assessment

| Package | Complexity | Reasoning |
|---------|-----------|-----------|
| `db` | High | 2,335 LOC schema file, migration accumulation |
| `cli/commands` | High | 68 files, structural duplication across entity types |
| `services` | Medium | Well-factored with EntityService extraction |
| `repository` | Medium | Large but simple (SQL queries) |
| `config` | Medium | Complex workflow config parsing |
| `status` | Medium | Non-trivial calculation logic |
| `models` | Low | Pure data types, minimal logic |
| `workflow` | Low | Clean, focused responsibility |

---

See also: [Code Metrics](code-metrics.md) | [Maintenance Burden](../technical-debt/maintenance-burden.md)
