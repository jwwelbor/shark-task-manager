# Code Metrics

> Part of the Shark Task Manager Brownfield Analysis
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

See also: [Complexity Analysis](complexity-analysis.md) | [Dependency Analysis](dependency-analysis.md)
