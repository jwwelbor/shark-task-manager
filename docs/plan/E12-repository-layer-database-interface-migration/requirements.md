# Requirements: Repository Layer Database Interface Migration

**Epic**: E12
**Last Updated**: 2026-01-09

---

## Technical Requirements

### Core Repository Layer

**REQ-1: Update repository.DB Structure**
- Replace embedded `*sql.DB` with `db.Database` interface
- Update `NewDB()` constructor to accept `db.Database`
- Maintain backward compatibility during migration
- **Priority**: High

**REQ-2: Update Repository Method Signatures**
- Modify all repository methods to use interface methods
- Replace `db.QueryContext()` with `db.Query(ctx, ...)`
- Replace `db.ExecContext()` with `db.Exec(ctx, ...)`
- Replace `db.QueryRowContext()` with `db.QueryRow(ctx, ...)`
- **Priority**: High

**REQ-3: Transaction Support**
- Update transaction handling to use `db.Database.Begin()`
- Ensure `db.Tx` interface methods are used in transaction blocks
- Maintain existing transaction semantics
- **Priority**: High

### Database Initialization

**REQ-4: Unify Initialization Paths**
- Eliminate dual code paths in `internal/cli/db_init.go`
- Use single path: `InitializeDatabaseFromConfig() -> db.Database -> repository.NewDB()`
- Support both SQLite and Turso through same code path
- **Priority**: High

**REQ-5: Remove TursoDriver Workaround**
- Delete `TursoDriver.GetSQLDB()` method
- Ensure TursoDriver fully implements `db.Database` interface
- Verify no other code depends on `GetSQLDB()`
- **Priority**: Medium

### Testing

**REQ-6: Update Repository Tests**
- Create mock implementations of `db.Database` interface
- Update all repository tests to use interface mocks
- Ensure no test logic changes (only setup changes)
- Maintain 100% test coverage
- **Priority**: High

**REQ-7: Integration Testing**
- Verify both SQLite and Turso backends work correctly
- Test transaction handling with both backends
- Test concurrent access scenarios
- **Priority**: High

**REQ-8: Performance Validation**
- Benchmark interface method calls vs direct sql.DB calls
- Ensure <5% performance overhead
- No regression in query execution time
- **Priority**: Medium

---

## Non-Functional Requirements

### Maintainability
- Code should be simpler and more consistent after refactoring
- Single source of truth for database initialization
- Clear interface contracts

### Extensibility
- Easy to add new database backends (PostgreSQL, MySQL)
- No special-casing per backend in repository layer
- Backend selection through configuration only

### Backward Compatibility
- All existing CLI commands continue to work
- No changes to command interfaces or outputs
- Existing `.sharkconfig.json` files remain valid

---

## Out of Scope

The following are explicitly **not** part of this epic:

- Adding new database backends (PostgreSQL, MySQL)
- Changing database schema or migrations
- Modifying repository business logic
- Adding new repository methods or features
- Performance optimizations beyond interface overhead
- Connection pooling or advanced DB features

---

## Affected Components

### Files Requiring Changes (~20 files)

**Repository Layer** (15+ files):
- `internal/repository/repository.go` - Core DB wrapper
- `internal/repository/task_repository.go`
- `internal/repository/epic_repository.go`
- `internal/repository/feature_repository.go`
- `internal/repository/document_repository.go`
- `internal/repository/note_repository.go`
- `internal/repository/task_history_repository.go`
- All other `*_repository.go` files

**Database Layer** (3 files):
- `internal/cli/db_init.go` - Unified initialization
- `internal/db/turso.go` - Remove workaround
- `internal/db/sqlite.go` - Ensure interface compliance

**Testing** (15+ files):
- All `*_repository_test.go` files
- Test helpers in `internal/test/`

### Files NOT Requiring Changes

- Command layer (`internal/cli/commands/*.go`) - Already use `cli.GetDB()`
- Database drivers (`internal/db/database.go`) - Interface already defined
- Models (`internal/models/*.go`) - No changes needed

---

## Success Metrics

1. **Code Simplification**: Reduce database initialization code by 30+ lines
2. **Test Coverage**: Maintain 100% repository test coverage
3. **Performance**: <5% overhead from interface calls
4. **Consistency**: Single code path for all database backends

---

## Dependencies

### Prerequisites
- E07-F16 (Global Database Pattern) - ✅ Complete
- `db.Database` interface defined - ✅ Complete
- TursoDriver implements interface - ✅ Complete

### Blockers
None - this can be implemented immediately

---

## Implementation Phases

### Phase 1: Core Repository Layer
1. Update `repository.DB` struct
2. Update `NewDB()` constructor
3. Create interface adapter helpers if needed

### Phase 2: Repository Methods
1. Update TaskRepository methods
2. Update EpicRepository methods
3. Update FeatureRepository methods
4. Update remaining repositories

### Phase 3: Initialization & Cleanup
1. Unify `initDatabase()` function
2. Remove `GetSQLDB()` workaround
3. Update documentation

### Phase 4: Testing & Validation
1. Update all repository tests
2. Run integration tests
3. Performance benchmarking
4. Production validation

---

## Risk Assessment

### Low Risk
- Breaking existing functionality (high test coverage)
- Performance regression (interface calls are fast in Go)
- Breaking cloud database support (already working)

### Medium Risk
- Test setup complexity (need good interface mocks)
- Finding all direct `*sql.DB` usages
- Transaction handling edge cases

### Mitigation
- Implement in phases with tests at each step
- Use compiler to find all method signature changes
- Extensive integration testing before merge

---

*This epic completes the architectural migration started in E07-F16 by eliminating the final remaining technical debt in the database layer.*
