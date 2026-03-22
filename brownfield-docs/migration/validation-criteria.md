# Validation Criteria

> Part of the Shark Task Manager Brownfield Analysis
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

See also: [Component Order](component-order.md) | [Test Specifications](test-specifications.md)
