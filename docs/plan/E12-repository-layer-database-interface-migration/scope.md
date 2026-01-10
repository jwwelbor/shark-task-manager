# Scope Boundaries: Repository Layer Database Interface Migration

**Epic**: E12
**Last Updated**: 2026-01-09

---

## In Scope

### Primary Deliverables

1. **Repository Layer Refactoring**
   - Update `repository.DB` to use `db.Database` interface
   - Modify all repository methods to use interface methods
   - Update all 15+ repository files for consistency
   - Maintain existing business logic unchanged

2. **Unified Database Initialization**
   - Simplify `internal/cli/db_init.go` to single code path
   - Remove special-casing for SQLite vs Turso
   - Use `db.Database` interface throughout initialization

3. **Remove Workarounds**
   - Delete `TursoDriver.GetSQLDB()` method
   - Eliminate type assertions and unwrapping
   - Clean up temporary compatibility code

4. **Test Updates**
   - Create interface mocks for testing
   - Update all repository test files
   - Ensure test coverage remains at 100%
   - Validate with integration tests

### Technical Scope

**Modified Files (~20-25 files)**:
- Repository layer implementations
- Database initialization code
- Repository test files
- Database driver implementations

**Code Changes**:
- Interface adoption in repository layer
- Method signature updates
- Test setup refactoring
- Documentation updates

---

## Out of Scope

### Not Included in This Epic

1. **New Database Backends**
   - NOT adding PostgreSQL support
   - NOT adding MySQL support
   - NOT adding other database types
   - *Reason*: Focus on interface adoption, not expansion

2. **Schema Changes**
   - NO modifications to database schema
   - NO new tables or columns
   - NO migration scripts
   - *Reason*: Pure refactoring, no feature changes

3. **Performance Optimizations**
   - NO connection pooling changes
   - NO query optimization
   - NO caching improvements
   - *Exception*: Must not regress performance >5%
   - *Reason*: Separate concern from interface adoption

4. **New Repository Features**
   - NO new repository methods
   - NO new query capabilities
   - NO business logic changes
   - *Reason*: Refactoring only, feature freeze

5. **Command Layer Changes**
   - NO changes to CLI commands
   - NO changes to command interfaces
   - NO changes to output formatting
   - *Reason*: Already uses `cli.GetDB()`, no changes needed

6. **Configuration Changes**
   - NO new config options
   - NO changes to `.sharkconfig.json` format
   - NO changes to Turso connection strings
   - *Reason*: Configuration layer already correct

7. **Cloud Features**
   - NO new Turso-specific features
   - NO embedded replica changes
   - NO sync strategy modifications
   - *Reason*: Cloud support already complete

---

## Dependencies & Prerequisites

### Required Before Starting

✅ **Already Complete**:
- E07-F16: Global database pattern implementation
- `db.Database` interface defined and stable
- TursoDriver implements full interface
- All tests passing with current architecture

⏳ **No Blockers**: Can start immediately

### Not Required

❌ Don't need to wait for:
- Other epics to complete
- Schema migrations
- New features
- Performance tuning

---

## Future Considerations

Items that may become separate epics later:

### Potential Follow-Up Work

1. **PostgreSQL Backend** (Future Epic)
   - After interface is adopted, adding PostgreSQL will be trivial
   - Would reuse same interface pattern
   - Estimated effort: Small (1-2 weeks)

2. **MySQL Backend** (Future Epic)
   - Similar to PostgreSQL
   - Interface already supports it
   - Estimated effort: Small (1-2 weeks)

3. **Connection Pool Optimization** (Future Epic)
   - Interface supports custom connection management
   - Could improve performance for high-concurrency scenarios
   - Estimated effort: Medium (2-3 weeks)

4. **Multi-Database Support** (Future Epic)
   - Read replicas, write/read splitting
   - Interface flexible enough to support
   - Estimated effort: Large (4-6 weeks)

### Why Not Now?

These are deferred because:
- Interface adoption provides foundation first
- Each adds complexity that should be evaluated separately
- Current architecture works well for existing needs
- Can be prioritized based on actual demand

---

## Acceptance Criteria

### Definition of Done

This epic is complete when:

1. ✅ All repositories accept `db.Database` interface
2. ✅ Single initialization path in `db_init.go`
3. ✅ `GetSQLDB()` workaround removed
4. ✅ All tests updated and passing
5. ✅ No performance regression >5%
6. ✅ Both SQLite and Turso work correctly
7. ✅ Documentation updated
8. ✅ Code review approved
9. ✅ Merged to main branch

### Not Required for Completion

❌ Don't need:
- New database backends implemented
- Performance improvements beyond baseline
- New features or capabilities
- Configuration format changes

---

## Boundary Examples

### ✅ In Scope Examples

**Example 1**: Changing this method signature
```go
// Before
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    row := r.db.DB.QueryRowContext(ctx, query, id)
    // ...
}

// After
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    row := r.db.QueryRow(ctx, query, id)
    // ...
}
```

**Example 2**: Simplifying initialization
```go
// Before: Dual paths
if backend == "sqlite" {
    sqlDB := db.InitDB(path)
    return repository.NewDB(sqlDB)
} else {
    turso := InitTurso(url)
    sqlDB := turso.GetSQLDB()  // Workaround!
    return repository.NewDB(sqlDB)
}

// After: Single path
database := InitializeDatabaseFromConfig(config)
return repository.NewDB(database)  // Works for both!
```

### ❌ Out of Scope Examples

**Example 1**: Adding new backend (NOT in scope)
```go
// This is a separate epic
case "postgres":
    return NewPostgresDriver(config)
```

**Example 2**: New repository methods (NOT in scope)
```go
// This is a feature addition, not refactoring
func (r *TaskRepository) GetTasksWithCache(ctx context.Context) ([]*models.Task, error) {
    // New functionality
}
```

**Example 3**: Schema changes (NOT in scope)
```go
// No schema modifications
ALTER TABLE tasks ADD COLUMN new_field TEXT;
```

---

## Communication

### Stakeholders

**Must Review**:
- Backend developers (implementation)
- DevOps engineers (deployment)
- QA team (testing strategy)

**Should Inform**:
- Frontend developers (no impact, but awareness)
- Documentation team (update guides)

**No Impact**:
- End users (transparent refactoring)
- Product managers (no feature changes)

---

*This epic has a well-defined, achievable scope focused on eliminating technical debt without expanding functionality.*
