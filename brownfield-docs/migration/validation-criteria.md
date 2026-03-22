# Validation Criteria

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 8 — Migration Readiness

## Per-Component Acceptance Criteria

### Service Layer Migration (E15)

| Criterion | Verification |
|-----------|-------------|
| CLI command contains no business logic | Code review: command file < 40 LOC, only parse → call → format |
| CLI command does not import repository | No `repository` import in command file |
| Service method has unit tests with mocks | Test file exists, uses mock repos, covers happy + error paths |
| Service method wraps errors with context | `fmt.Errorf("...: %w", err)` pattern in all error returns |
| Existing CLI tests still pass | `make test` passes |
| JSON output format unchanged | Integration test or manual verification |
| Exit codes unchanged | Error type → exit code mapping preserved |

### Entity Polymorphism (E21)

| Criterion | Verification |
|-----------|-------------|
| Entity interface implemented by all types | Compile-time: `var _ Entity = (*Epic)(nil)` |
| EntityRegistry handles all entity types | Test: registry.Get(key) for each type |
| Unified status transitions work | Test: EntityService.SetStatus for each entity type |
| Duplicate command files eliminated | File count reduction measurable |
| No regression in entity-specific behavior | Full test suite passes |

### Schema Split

| Criterion | Verification |
|-----------|-------------|
| db.go < 500 LOC after split | `wc -l internal/db/db.go` |
| All tests pass unchanged | `make test` |
| Schema version still checked | `ApplySchemaIfNeeded` still works |
| Turso path still works | Manual test with Turso backend |

## Performance Benchmarks

| Metric | Baseline | Acceptable | Verification |
|--------|----------|------------|-------------|
| `shark get` latency | <100ms | <200ms | `time shark get E07-F01-001` |
| `shark list` (100 tasks) | <200ms | <500ms | `time shark list E07 F01` |
| `shark status advance` | <150ms | <300ms | `time shark status advance E07-F01-001` |
| Database init (cold start) | <500ms | <1s | `time shark list` (first run) |
| Database init (skip migrations) | <50ms | <100ms | `time shark list` (warm) |

## Data Integrity Checks

| Check | Command |
|-------|---------|
| Foreign key integrity | `PRAGMA integrity_check;` |
| All tasks have valid features | `SELECT * FROM tasks WHERE feature_id NOT IN (SELECT id FROM features)` |
| All features have valid epics | `SELECT * FROM features WHERE epic_id NOT IN (SELECT id FROM epics)` |
| Schema version correct | `SELECT version FROM schema_version ORDER BY version DESC LIMIT 1` |
| No orphaned task history | `SELECT * FROM task_history WHERE task_id NOT IN (SELECT id FROM tasks)` |

## Rollback Procedures

### Service Layer Migration
- **Rollback**: Revert command file to pre-migration state
- **Risk**: Low — services are additive, commands can still call repos
- **Data impact**: None — no schema changes

### Schema Split
- **Rollback**: Git revert the split commits
- **Risk**: Very low — no functional changes
- **Data impact**: None

### Turso Client Update
- **Rollback**: Revert go.mod to previous commit hash
- **Risk**: Low — Go module system handles versioning
- **Data impact**: None — data in cloud database unchanged

---

See also: [Component Order](component-order.md) | [Test Specifications](test-specifications.md)
