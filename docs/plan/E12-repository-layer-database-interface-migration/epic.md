---
epic_key: E12
title: Repository Layer Database Interface Migration
description: Migrate repository layer from *sql.DB to db.Database interface to eliminate dual initialization paths and complete cloud-aware architecture. Currently repositories require *sql.DB which forces TursoDriver to expose GetSQLDB() workaround. This epic will refactor ~15 repository files to use db.Database interface directly, simplifying initialization logic and improving maintainability.
---

# Repository Layer Database Interface Migration

**Epic Key**: E12

---

## Goal

### Problem
The repository layer currently requires `*sql.DB` from Go's standard library, which forces the cloud database implementation (TursoDriver) to expose a `GetSQLDB()` workaround method. This creates dual initialization paths in `internal/cli/db_init.go` - one simple path for local SQLite and one complex path for Turso that must extract the underlying `*sql.DB`. This technical debt makes the codebase harder to maintain, understand, and extend with future database backends. The infrastructure for a clean interface-based design exists (`db.Database` interface) but is not utilized by the repository layer.

### Solution
Refactor the repository layer to accept and use the `db.Database` interface instead of `*sql.DB`. This involves updating `repository.DB` struct to embed `db.Database`, modifying all repository methods to use interface methods instead of `*sql.DB` methods, and updating the 15+ repository files. Once complete, both SQLite and Turso backends will be initialized through a single unified code path, eliminating the workaround and simplifying the architecture.

### Impact
- Eliminate dual database initialization paths, reducing code complexity by ~30 lines
- Remove the `GetSQLDB()` workaround from TursoDriver, improving architecture cleanliness
- Enable easier addition of future database backends (PostgreSQL, MySQL) without code path multiplication
- Improve maintainability by having a single, consistent database initialization pattern
- Complete the cloud-aware architecture migration started in E07-F16

---

## Business Value

**Rating**: Medium

This is technical debt reduction that improves long-term maintainability and extensibility. While it doesn't directly impact end users, it significantly reduces the complexity of database initialization logic, making future database backend additions (PostgreSQL, MySQL) much easier. It eliminates architectural workarounds and completes the cloud-aware infrastructure started in E07-F16, reducing risk of bugs and simplifying onboarding for new developers working with the database layer.

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - Target user profiles and characteristics
- **[User Journeys](./user-journeys.md)** - High-level workflows and interaction patterns
- **[Requirements](./requirements.md)** - Functional and non-functional requirements
- **[Success Metrics](./success-metrics.md)** - KPIs and measurement framework
- **[Scope Boundaries](./scope.md)** - Out of scope items and future considerations

---

## Quick Reference

**Primary Users**: Backend developers, DevOps engineers, contributors working with database layer

**Key Changes**:
- Refactor `repository.DB` to use `db.Database` interface instead of `*sql.DB`
- Update 15+ repository files to use interface methods
- Unify database initialization in `internal/cli/db_init.go` (eliminate dual paths)
- Remove `TursoDriver.GetSQLDB()` workaround method
- Update all tests to work with interface-based repositories

**Success Criteria**:
- Single database initialization path in `initDatabase()` function
- All repositories accept `db.Database` interface via `NewDB()`
- All existing tests pass without modification to test logic
- No performance regression (<5% overhead from interface calls)

**Timeline**: No critical deadlines - can be implemented incrementally alongside other work

**Technical Scope**:
- `internal/repository/repository.go` - Core DB wrapper
- `internal/repository/*_repository.go` - 15+ repository implementations
- `internal/cli/db_init.go` - Unified initialization
- `internal/db/turso.go` - Remove GetSQLDB() workaround
- All repository tests - Update mocking patterns

---

*Last Updated*: 2026-01-09
