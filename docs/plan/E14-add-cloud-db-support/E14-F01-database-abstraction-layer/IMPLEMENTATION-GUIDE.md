# F01 Implementation Guide: Database Abstraction Layer

## Overview

Create a database abstraction layer that allows Shark to support multiple database backends (SQLite local, Turso cloud) without changing business logic or command implementations.

## Implementation Order

Follow this sequence (per MVP roadmap):

1. **T-E14-F01-005**: Add configuration fields for database backend selection
2. **T-E14-F01-001**: Define database interface with CRUD operations
3. **T-E14-F01-002**: Refactor SQLite repository to implement database interface
4. **T-E14-F01-003**: Implement database driver registry for backend selection
5. **T-E14-F01-004**: Write unit tests for database abstraction layer

## Technical Design Reference

See feature.md for complete technical design including:
- Database interface definition (lines 128-159)
- Driver registry pattern (lines 161-192)
- Usage examples (lines 194-214)

## Reference Prototype

Working prototype available at:
- `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2026-01-05-turso-prototype/`
- See `prototype/main.go` and `prototype/test_replica.go` for Turso integration examples

## Key Requirements

### REQ-F-001: Database Interface

**File**: `internal/db/database.go` (new file)

Interface must include:
- Connection management: `Connect()`, `Close()`, `Ping()`
- Query operations: `Query()`, `QueryRow()`, `Exec()`
- Transaction support: `Begin()`, `Commit()`, `Rollback()`
- Metadata: `DriverName()`

**Acceptance Criteria**:
- Interface supports all repository patterns in use
- Compatible with existing `*sql.DB` usage
- Supports context for cancellation/timeouts
- Transaction interface matches `*sql.Tx` patterns

### REQ-F-002: SQLite Driver Implementation

**Files**:
- `internal/db/sqlite_driver.go` (new file)
- Refactor `internal/db/db.go` to use interface

Preserve existing behavior:
- WAL mode enabled
- Foreign keys enforced
- Connection pooling configured
- All PRAGMA settings maintained

**Acceptance Criteria**:
- All existing tests pass without modification
- No performance degradation (< 5ms overhead)
- Backward compatible with current usage

### REQ-F-003: Driver Registry

**File**: `internal/db/registry.go` (new file)

Registry pattern:
- `RegisterDriver(name, factory)` for backend registration
- `NewDatabase(config)` creates instance based on config
- `DetectBackend(url)` auto-detects from URL format
- SQLite is default backend

**Acceptance Criteria**:
- `libsql://` URLs detected as Turso
- File paths detected as SQLite
- Invalid backends return clear error
- Registry allows adding new backends

### REQ-F-004: Configuration Integration

**Files**:
- `internal/config/config.go` - add database config fields
- `.sharkconfig.json` schema updated

Configuration fields:
```json
{
  "database": {
    "backend": "sqlite",  // "sqlite" | "turso"
    "url": "./shark-tasks.db",
    "connection": {
      "max_open_conns": 25,
      "max_idle_conns": 5,
      "conn_max_lifetime": "5m"
    }
  }
}
```

**Acceptance Criteria**:
- Priority: CLI flag `--db` > env `SHARK_DB_URL` > config file > default
- URL format auto-detection works
- Invalid config shows helpful error
- Backward compatible with existing config

## Testing Strategy

### Unit Tests (T-E14-F01-004)

**File**: `internal/db/database_test.go`

Test coverage required:
1. **Interface compliance**: SQLite driver implements all methods
2. **Registry**:
   - Register and retrieve backends
   - Auto-detection from URLs
   - Error handling for invalid backends
3. **Configuration**:
   - Parse database config correctly
   - Priority order (flag > env > config > default)
   - Validation catches invalid configs

**Mock interface** for testing consumers:
```go
type MockDatabase struct {
    ConnectFunc func(ctx context.Context, dsn string) error
    QueryFunc   func(ctx context.Context, query string, args ...interface{}) (Rows, error)
    // ... other methods
}
```

### Integration Tests

Run existing repository tests - should pass unchanged:
```bash
make test-db
```

**Success criteria**: 100% pass rate on existing tests

## Migration Path for Existing Code

### Before (Current)
```go
// internal/repository/task_repository.go
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
    row := r.db.QueryRow("SELECT * FROM tasks WHERE id = ?", id)
    // ...
}
```

### After (With Interface)
```go
// internal/repository/task_repository.go
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
    row := r.db.QueryRow(ctx, "SELECT * FROM tasks WHERE id = ?", id)
    // ...
}
```

**Key change**: Add `ctx` parameter to all database calls

## Files to Create

1. `internal/db/database.go` - Interface definitions
2. `internal/db/sqlite_driver.go` - SQLite implementation
3. `internal/db/registry.go` - Driver registry
4. `internal/db/database_test.go` - Unit tests
5. `internal/db/mock_database.go` - Mock for testing

## Files to Modify

1. `internal/db/db.go` - Refactor to use interface
2. `internal/config/config.go` - Add database config fields
3. `internal/repository/*_repository.go` - Add ctx to DB calls (if needed)

## Validation Checklist

Before marking F01 complete:

- [ ] All 5 tasks completed
- [ ] Interface defined with all required methods
- [ ] SQLite driver implements interface
- [ ] Registry pattern working
- [ ] Configuration fields added
- [ ] Unit tests pass (new tests)
- [ ] Integration tests pass (existing repository tests)
- [ ] No performance regression (benchmark)
- [ ] Code review approved

## Success Metrics

- All existing tests pass unchanged
- Command layer has zero direct `database/sql` imports
- Mock database interface available for testing
- Benchmark shows < 5ms abstraction overhead

---

**Ready for Development**: This guide provides sufficient context to implement all F01 tasks.
