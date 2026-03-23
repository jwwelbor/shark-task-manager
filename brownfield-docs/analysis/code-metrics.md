# Code Metrics

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 7 — Code Quality Analysis

## Lines of Code

| Category | Files | LOC | Percentage |
|----------|-------|-----|-----------|
| Production Go | 592 | ~135,263 | 34.5% |
| Test Go | 605 | ~256,737 | 65.5% |
| **Total Go** | **1,197** | **~392,000** | **100%** |

## File Counts by Type

| Extension | Count | Purpose |
|-----------|-------|---------|
| .go | 1,197 | Source code + tests |
| .md | ~1,955 | Documentation, entity files |
| .json | 30 | Configuration |
| .yml | 10 | CI/CD, linting config |
| .sh | 28 | Scripts |
| .toml | 2 | Air, GoReleaser config |

## Package Size Distribution

| Size Category | Packages | Examples |
|--------------|----------|---------|
| Large (>10K LOC) | 3 | repository (30K), services (28K), config (14K) |
| Medium (2K-10K) | 10 | db (11K), status (7K), init (5K), cli (5K), runner (4K), discovery (4K), patterns (4K) |
| Small (<2K LOC) | 18 | models (3K), taskcreation (3K), templates (3K), workflow (2K), validation (2K), etc. |

## Test Coverage Metrics

| Metric | Value |
|--------|-------|
| Test-to-code ratio | 1.9:1 |
| Test files | 605 (51% of Go files) |
| Test LOC | ~257K (66% of total Go LOC) |
| Test framework | testify (assertions + mocks) |
| Test execution | Sequential (-p=1) |

## Module Statistics

| Metric | Value |
|--------|-------|
| Internal packages | 31 |
| Entry points (cmd/) | 11 |
| Direct dependencies | 9 |
| Transitive dependencies | ~100+ |
| Database tables | 16 |
| CLI commands | 30+ |
| Entity templates | 80+ |

## Average File Size

| Package | Avg LOC/file | Description |
|---------|-------------|-------------|
| repository | ~391 | Larger files (complex SQL queries) |
| services | ~461 | Larger files (business logic) |
| config | ~534 | Configuration parsing complexity |
| models | ~115 | Compact data structures |
| cli/commands | ~200 | Medium (thin wrappers) |
=======
> Generated: 2026-03-20
> Phase: 7 — Code Quality Analysis

## Summary Statistics

| Metric | Value |
|--------|-------|
| **Total Go files** | 1,679 |
| **Production Go files (internal/)** | 266 |
| **Test Go files (internal/)** | 288 |
| **Command entry files (cmd/)** | 12 |
| **Production LOC (internal/)** | 64,632 |
| **Test LOC (internal/)** | 121,144 |
| **Test-to-Production ratio** | 1.87:1 |
| **Internal packages** | 30 |
| **CLI command files** | 68 |
| **Markdown files** | 5,636 |
| **JSON config files** | 41 |
| **GitHub Actions workflows** | 3 |

## Files by Type

| Extension | Count | Purpose |
|-----------|-------|---------|
| `.go` | 1,679 | Source code + tests |
| `.md` | 5,636 | Documentation, task files, plans |
| `.json` | 41 | Configuration, test data |
| `.yml`/`.yaml` | ~5 | CI/CD, linter config, air config |
| `.tmpl` | ~20+ | Go templates (embedded) |

## Lines of Code by Package (Production Only)

| Package | LOC | Files | Avg LOC/File | Category |
|---------|-----|-------|-------------|----------|
| `db` | ~3,900 | 9 | 433 | Infrastructure |
| `repository` | ~6,200 | 22 | 282 | Data Access |
| `services` | ~8,500 | 38 | 224 | Business Logic |
| `cli/commands` | ~16,000 | 68 | 235 | CLI Layer |
| `cli (framework)` | ~3,500 | 14 | 250 | Framework |
| `config` | ~2,800 | 13 | 215 | Configuration |
| `models` | ~2,500 | 20 | 125 | Domain Types |
| `status` | ~2,200 | 11 | 200 | Status Calc |
| `init` | ~1,800 | 11 | 164 | Initialization |
| `workflow` | ~600 | 3 | 200 | Workflow |
| Other (16 pkgs) | ~16,500 | ~50 | ~330 | Utilities |

## Largest Files (Complexity Hotspots)

### Production Files

| File | LOC | Concern |
|------|-----|---------|
| `internal/db/db.go` | 2,335 | Schema + initialization — potential god file |
| `internal/repository/task_repository.go` | 1,806 | Task data access — many query methods |
| `internal/db/migrate.go` | 1,560 | Migration logic — expected to grow |
| `internal/services/task_service.go` | 1,371 | Task business logic — central service |
| `internal/cli/commands/feature_helpers.go` | 1,209 | Feature output formatting |
| `internal/cli/commands/config.go` | 1,075 | Config command handlers |
| `internal/services/feature_service.go` | 1,063 | Feature business logic |
| `internal/repository/feature_repository.go` | 995 | Feature data access |
| `internal/cli/commands/epic_helpers.go` | 994 | Epic output formatting |
| `internal/services/epic_service.go` | 923 | Epic business logic |

### Test Files

| File | LOC | Notes |
|------|-----|-------|
| `internal/services/feature_service_test.go` | 3,356 | Most extensively tested service |
| `internal/config/template_helpers_test.go` | 2,611 | Template helper validation |
| `internal/services/task_service_test.go` | 2,403 | Task service coverage |
| `internal/services/epic_service_test.go` | 2,148 | Epic service coverage |
| `internal/status/status_test.go` | 1,885 | Status calculation tests |

## Test Coverage Observations

- **Services**: Well-tested (task: 2,403 LOC, feature: 3,356 LOC, epic: 2,148 LOC)
- **Repository**: 50 test files — most extensive test suite
- **Config**: 14 test files — thorough configuration validation
- **CLI commands**: 100 test files — comprehensive command testing
- **Models**: 6 test files — basic structural validation
- **Workflow**: 3 test files — transition validation
- **Test-to-production ratio of 1.87:1** is excellent, indicating strong test discipline

## File Size Distribution

| Range | Production Files | Test Files |
|-------|-----------------|-----------|
| 0-100 LOC | ~80 | ~60 |
| 100-300 LOC | ~100 | ~80 |
| 300-500 LOC | ~40 | ~50 |
| 500-1000 LOC | ~30 | ~40 |
| 1000+ LOC | ~16 | ~15 |

**Observation**: 16 production files exceed 1,000 LOC. The top 3 (`db.go`, `task_repository.go`, `migrate.go`) account for 5,700 LOC — nearly 9% of all production code. These are complexity hotspots.

---
>>>>>>> Stashed changes

See also: [Complexity Analysis](complexity-analysis.md) | [Dependency Analysis](dependency-analysis.md)
