# Validation Criteria

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 8 — Migration Readiness

## Functional Acceptance Criteria

### Per-Command Migration

Each migrated command must pass:

- [ ] All existing test cases still pass
- [ ] JSON output format is unchanged (backward compatible)
- [ ] Table output format is visually equivalent
- [ ] Exit codes match existing behavior (0, 1, 2, 3, 4)
- [ ] Error messages contain same key information
- [ ] `--field` flag extracts same values
- [ ] `--verbose` flag produces debug output
- [ ] Command works with both local SQLite and Turso backends

### Service Layer

Each new service method must have:

- [ ] Happy path test with mocked repository
- [ ] Error path tests (not found, invalid transition, dependency unmet)
- [ ] Table-driven tests for multiple scenarios
- [ ] Mock interaction verification (correct parameters passed)
- [ ] Context cancellation handling
- [ ] Optional dependency degradation (nil checks)

### HTTP API (When Implemented)

Each endpoint must have:

- [ ] Success response with correct status code and body
- [ ] 404 for not-found entities
- [ ] 400 for invalid input
- [ ] 422 for workflow violations
- [ ] Content-Type: application/json
- [ ] Location header for created resources (201)

## Quality Gates

### Mandatory Before Any Merge

```bash
make fmt    # No formatting changes
make lint   # No linting errors
make test   # All tests pass
```

### Coverage Thresholds

| Layer | Minimum Coverage | Target |
|-------|-----------------|--------|
| Services | 80% | 90% |
| Repository | 70% | 80% |
| CLI Commands | 60% | 75% |
| Models | 90% | 95% |

## Data Integrity Checks

| Check | Method | When |
|-------|--------|------|
| Foreign key integrity | `PRAGMA foreign_key_check` | After migration |
| No orphaned features | `SELECT f.* FROM features f LEFT JOIN epics e ON f.epic_id = e.id WHERE e.id IS NULL` | After migration |
| No orphaned tasks | `SELECT t.* FROM tasks t LEFT JOIN features f ON t.feature_id = f.id WHERE f.id IS NULL` | After migration |
| Schema version correct | `SELECT version FROM schema_version` | After migration |
| Status values valid | Compare all task statuses against workflow config | After migration |

## Rollback Procedures

### Service Layer Rollback

If a service migration causes issues:
1. Revert the command to use direct repository access
2. Keep the service code (don't delete) for future attempt
3. Update tests to match reverted implementation
4. Document what went wrong for retry

### Database Rollback

1. Restore from backup: `cp shark-tasks.db.backup shark-tasks.db`
2. If no backup: `shark init --non-interactive` (loses data)
3. Turso: Use Turso's point-in-time recovery

### Configuration Rollback

1. Restore from auto-backup: `cp .sharkconfig.json.backup.* .sharkconfig.json`
2. Or re-apply profile: `shark init update --workflow=basic`
=======
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
>>>>>>> Stashed changes

See also: [Component Order](component-order.md) | [Test Specifications](test-specifications.md)
