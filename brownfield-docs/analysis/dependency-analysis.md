# Dependency Analysis

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 7 — Code Quality Analysis

## Dependency Count

| Category | Count |
|----------|-------|
| Direct Go dependencies | 9 |
| Indirect Go dependencies | ~26 |
| Total transitive | ~100+ |
| Internal packages | 31 |
| Build tools | 3 (golangci-lint, GoReleaser, Air) |

## Dependency Freshness

| Status | Count | Dependencies |
|--------|-------|-------------|
| Current (latest or near-latest) | 8 | go-sqlite3, cobra, viper, pterm, testify, yaml, x/term, x/text |
| Dev/Unstable | 1 | libsql-client-go (v0.0.0 commit) |
| Deprecated | 0 | None |

**Freshness Score**: 89% current (8/9 on stable versions)

## Circular Dependencies

No circular dependencies detected between internal packages. The dependency graph is a clean DAG:

```
cli/commands → services → repository → db → config
                       → workflow → config
                       → models (leaf)
```

**Key constraint maintained**: Models package has no imports from other internal packages (leaf node).

## Unused Dependencies

No obviously unused dependencies detected in `go.mod`. All direct dependencies are imported somewhere in the codebase.

## Heavy Dependencies (Large Transitive Trees)

| Dependency | Direct Transitives | Total Impact |
|-----------|-------------------|-------------|
| viper | 8 (afero, cast, fsnotify, mapstructure, toml, gotenv, locafero, conc) | ~15 packages |
| pterm | 7 (cursor, keyboard, schedule, color, console, uniseg, runewidth) | ~10 packages |
| libsql-client-go | 2 (antlr, websocket) | ~5 packages |
| cobra | 2 (pflag, mousetrap) | ~3 packages |
| testify | 2 (spew, difflib) | ~3 packages |
| go-sqlite3 | 0 | CGO only |

**Heaviest**: Viper brings the most transitive dependencies but is well-justified for config management.

## Internal Dependency Health

### High Fan-In (Many Dependents)

| Package | Dependents | Risk |
|---------|-----------|------|
| models | ~20 | Low (stable, leaf node) |
| config | ~8 | Medium (changes cascade widely) |
| repository | ~6 | Medium (central data access) |
| workflow | ~5 | Low (stable interface) |

### High Fan-Out (Many Dependencies)

| Package | Dependencies | Risk |
|---------|-------------|------|
| services | ~8 | Medium (expected for orchestration layer) |
| cli/commands | ~6 | Low (entry point, expected) |
| taskcreation | ~5 | Low (creation requires many capabilities) |

**Assessment**: Dependency health is good. The `config` package has the highest blast radius for changes. Services have high fan-out but that's expected for the orchestration layer.

See also: [Code Metrics](code-metrics.md) | [Complexity Analysis](complexity-analysis.md)
