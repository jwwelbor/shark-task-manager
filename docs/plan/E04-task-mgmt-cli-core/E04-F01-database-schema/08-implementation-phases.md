# Implementation Phases: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: feature-architect (coordinator)

## Overview

This document defines the implementation approach for the database layer, breaking down the work into logical phases with clear deliverables, dependencies, and success criteria. Implementation follows a bottom-up approach: foundational components first, then data access layer, then integration support.

---

## Documents Created

| Document | Status | Agent/Author |
|----------|--------|--------------|
| 00-research-report.md | ✅ Created | project-research-agent |
| 01-interface-contracts.md | ✅ Created | coordinator |
| 02-architecture.md | ✅ Created | backend-architect |
| 03-data-design.md | ✅ Created | db-admin |
| 04-backend-design.md | ✅ Created | backend-architect |
| 05-frontend-design.md | ❌ Skipped | N/A (no UI) |
| 06-security-design.md | ✅ Created | security-architect |
| 07-performance-design.md | ✅ Created | coordinator |
| 09-test-criteria.md | ✅ Created | tdd-agent |

**Total Documents**: 8 design documents

---

## Implementation Phases

### Phase 1: Foundation - Models and Schema

**Goal**: Create database schema, ORM models, and basic infrastructure

**Tasks**:
1. Create project structure (`pm/database/` package)
2. Implement ORM models (`models.py`)
   - Define `Base` class
   - Implement `Epic` model with relationships
   - Implement `Feature` model with relationships
   - Implement `Task` model with JSON dependency handling
   - Implement `TaskHistory` model
   - Add `to_dict()` serialization methods to all models
3. Implement validation module (`validation.py`)
   - Key format validation (regex patterns)
   - Enum validation (status, priority, agent_type)
   - Range validation (priority, progress_pct)
   - JSON validation (depends_on field)
4. Implement custom exceptions (`exceptions.py`)
   - DatabaseError base class
   - IntegrityError, ValidationError, DatabaseNotFound, SchemaVersionMismatch
5. Implement database configuration (`config.py`)
   - DatabaseConfig class with environment-specific settings
6. Create initial Alembic migration (`001_initial_schema.py`)
   - All 4 tables with constraints
   - All 10 indexes
   - All 3 triggers (updated_at auto-update)
   - Up and down migrations

**Dependencies**: None (foundational phase)

**Deliverables**:
- `pm/database/models.py` (~900 lines)
- `pm/database/validation.py` (~300 lines)
- `pm/database/exceptions.py` (~100 lines)
- `pm/database/config.py` (~100 lines)
- `pm/database/migrations/versions/001_initial_schema.py` (~200 lines)
- `alembic.ini` (configuration)
- `pm/database/migrations/env.py` (Alembic environment)

**Success Criteria**:
- [ ] All 4 models defined with type hints
- [ ] All relationships configured (cascade delete)
- [ ] All validation functions implemented
- [ ] All custom exceptions defined
- [ ] Alembic migration creates schema successfully
- [ ] `alembic upgrade head` creates all tables
- [ ] `alembic downgrade base` removes all tables
- [ ] No mypy type errors

**Estimated Effort**: 6-8 hours

---

### Phase 2: Session Management

**Goal**: Implement database connection management and session lifecycle

**Tasks**:
1. Implement session factory (`session.py`)
   - `SessionFactory` class with engine creation
   - `get_session()` context manager (auto-commit/rollback)
   - `create_all_tables()` for initialization
   - `check_integrity()` for corruption detection
   - `get_schema_version()` for Alembic version checking
2. Implement `init_database()` function
   - Database initialization with validation
   - Schema version verification
   - Integrity checking
   - File permission setting (Unix only)
3. Configure SQLite PRAGMAs
   - `foreign_keys=ON` (via event listener)
   - `journal_mode=WAL` (via event listener)
   - `busy_timeout=5000` (via event listener)
4. Write unit tests for session management
   - Test session creation and cleanup
   - Test transaction commit
   - Test transaction rollback on exception
   - Test integrity check
   - Test schema version retrieval

**Dependencies**: Phase 1 (models must exist)

**Deliverables**:
- `pm/database/session.py` (~200 lines)
- `tests/unit/database/test_session.py` (~150 lines)

**Success Criteria**:
- [ ] Sessions created and closed automatically
- [ ] Transactions commit on success
- [ ] Transactions rollback on exception
- [ ] Foreign keys enabled (verified)
- [ ] WAL mode enabled (verified)
- [ ] Integrity check works
- [ ] Schema version check works
- [ ] File permissions set to 600 on Unix
- [ ] All tests pass
- [ ] 100% test coverage for session.py

**Estimated Effort**: 4-6 hours

---

### Phase 3: Repository Layer - CRUD Operations

**Goal**: Implement data access repositories with CRUD operations

**Tasks**:
1. Implement `EpicRepository` (repositories.py)
   - `create()` - with key validation
   - `get_by_id()` - by primary key
   - `get_by_key()` - by business key
   - `list_all()` - with optional status filter
   - `update()` - with field validation
   - `delete()` - with cascade
   - `calculate_progress()` - epic progress from features
2. Implement `FeatureRepository`
   - `create()` - with key validation, epic_id FK check
   - `get_by_id()`
   - `get_by_key()`
   - `list_by_epic()` - features for an epic
   - `list_all()` - with optional status filter
   - `update()`
   - `delete()` - with cascade
   - `calculate_progress()` - feature progress from tasks
   - `update_progress()` - recalculate and cache progress_pct
3. Implement `TaskRepository`
   - `create()` - with comprehensive validation
   - `get_by_id()`
   - `get_by_key()`
   - `list_all()`
   - `list_by_feature()`
   - `list_by_epic()` - with JOIN through features
   - `filter_by_status()`
   - `filter_by_agent_type()`
   - `filter_by_priority()`
   - `filter_combined()` - multi-criteria filtering
   - `update()` - with field validation
   - `update_status()` - atomic status + history + timestamps
   - `delete()` - with cascade
4. Implement `TaskHistoryRepository`
   - `create()` - record status change
   - `list_by_task()` - history for a task
   - `list_recent()` - recent changes across all tasks
5. Write unit tests for each repository
   - Test all CRUD operations
   - Test validation error handling
   - Test foreign key constraint enforcement
   - Test cascade deletes
   - Test filtering and querying
   - Test progress calculations

**Dependencies**: Phase 2 (session management must exist)

**Deliverables**:
- `pm/database/repositories.py` (~1500 lines)
- `tests/unit/database/test_repositories.py` (~800 lines)

**Success Criteria**:
- [ ] All repository methods implemented
- [ ] All validation errors raised correctly
- [ ] Foreign key violations caught and translated
- [ ] Unique constraint violations caught and translated
- [ ] Cascade deletes work correctly
- [ ] Progress calculations accurate
- [ ] Filtered queries use indexes (verified with EXPLAIN QUERY PLAN)
- [ ] All tests pass
- [ ] >80% test coverage for repositories.py

**Estimated Effort**: 12-16 hours

---

### Phase 4: Integration Tests

**Goal**: Validate end-to-end database functionality

**Tasks**:
1. Create integration test fixtures
   - Temporary database fixture (file-based, not in-memory)
   - Populated database fixture (sample epics, features, tasks)
   - Large dataset fixture (10,000 tasks for performance testing)
2. Write integration tests
   - Test complete epic → feature → task creation
   - Test cascade delete (epic deletes all children)
   - Test foreign key constraints prevent orphans
   - Test transaction rollback on multi-step failures
   - Test progress calculation accuracy
   - Test atomic status update (task + history)
   - Test concurrent access (if applicable)
3. Write performance benchmarks
   - Single INSERT <50ms
   - get_by_key <10ms
   - Filter queries <100ms (10,000 tasks)
   - Bulk INSERT <2,000ms (100 tasks)
   - CASCADE DELETE <500ms
   - Progress calculation <200ms

**Dependencies**: Phase 3 (repositories must exist)

**Deliverables**:
- `tests/integration/test_database_operations.py` (~400 lines)
- `tests/performance/test_database_performance.py` (~300 lines)
- `tests/fixtures/sample_data.py` (~200 lines)

**Success Criteria**:
- [ ] All integration tests pass
- [ ] All performance benchmarks meet targets
- [ ] Foreign key enforcement verified
- [ ] Cascade delete verified
- [ ] Transaction rollback verified
- [ ] Progress calculations verified
- [ ] No database corruption after tests
- [ ] Tests can run in parallel (isolated databases)

**Estimated Effort**: 6-8 hours

---

### Phase 5: Documentation and Examples

**Goal**: Document usage patterns and provide examples

**Tasks**:
1. Create developer documentation
   - Database schema documentation (ASCII ER diagram)
   - Repository usage examples
   - Common query patterns
   - Transaction patterns
   - Error handling examples
2. Create migration guide
   - How to create new migrations
   - How to apply migrations
   - How to rollback migrations
   - Migration best practices
3. Add docstrings to all public APIs
   - Module-level docstrings
   - Class docstrings
   - Method docstrings with Args/Returns/Raises
4. Create usage examples
   - Example: Create epic with features and tasks
   - Example: Query tasks with filters
   - Example: Update task status atomically
   - Example: Calculate progress
   - Example: Export data to JSON

**Dependencies**: Phase 4 (all code must be complete)

**Deliverables**:
- `pm/database/README.md` (~200 lines)
- `docs/database-schema.md` (~300 lines)
- `docs/migration-guide.md` (~150 lines)
- `examples/database_usage.py` (~200 lines)

**Success Criteria**:
- [ ] All public APIs documented
- [ ] All examples run successfully
- [ ] Documentation covers common use cases
- [ ] Migration guide clear and complete
- [ ] Schema diagram accurate

**Estimated Effort**: 4-6 hours

---

### Phase 6: Database Package Export and CLI Integration Preparation

**Goal**: Finalize package exports and prepare for CLI integration

**Tasks**:
1. Create `pm/database/__init__.py` with public API exports
   ```python
   # Public API
   from .models import Epic, Feature, Task, TaskHistory
   from .session import SessionFactory, init_database, get_db_session
   from .repositories import (
       EpicRepository,
       FeatureRepository,
       TaskRepository,
       TaskHistoryRepository
   )
   from .exceptions import (
       DatabaseError,
       IntegrityError,
       ValidationError,
       DatabaseNotFound,
       SchemaVersionMismatch
   )
   from .config import DatabaseConfig

   __all__ = [...]
   ```
2. Create smoke test for package imports
3. Verify type hints work with mypy
4. Document integration points for E04-F02 (CLI Infrastructure)
   - How CLI commands will receive repositories
   - How sessions will be managed per-request
   - How to serialize models to JSON for output
5. Create integration checklist for downstream features
   - E04-F02: CLI Infrastructure (session injection pattern)
   - E04-F03: Task Lifecycle Operations (use TaskRepository)
   - E04-F05: Folder Management (atomic file + DB updates)
   - E04-F07: Initialization & Sync (bulk imports)

**Dependencies**: Phase 5 (documentation must exist)

**Deliverables**:
- `pm/database/__init__.py` (~50 lines)
- `docs/cli-integration-guide.md` (~100 lines)
- `tests/smoke/test_package_imports.py` (~50 lines)

**Success Criteria**:
- [ ] All public APIs exported from `__init__.py`
- [ ] Smoke test passes
- [ ] mypy validation passes
- [ ] Integration guide complete
- [ ] Ready for E04-F02 CLI integration

**Estimated Effort**: 2-4 hours

---

## Total Effort Estimate

| Phase | Effort (hours) |
|-------|---------------|
| Phase 1: Models and Schema | 6-8 |
| Phase 2: Session Management | 4-6 |
| Phase 3: Repository Layer | 12-16 |
| Phase 4: Integration Tests | 6-8 |
| Phase 5: Documentation | 4-6 |
| Phase 6: Package Export | 2-4 |
| **Total** | **34-48 hours** |

**Recommended Schedule**: 5-7 business days (assuming 8-hour days)

---

## Dependency Graph

```
Phase 1: Foundation
    ↓
Phase 2: Session Management
    ↓
Phase 3: Repository Layer
    ↓
Phase 4: Integration Tests
    ↓
Phase 5: Documentation
    ↓
Phase 6: Package Export
    ↓
Ready for E04-F02 (CLI Infrastructure)
```

**Critical Path**: Phases 1 → 2 → 3 (must be sequential)

**Parallel Opportunities**: Phases 4 and 5 can partially overlap.

---

## Risk Mitigation

### Risk 1: SQLAlchemy 2.0 Learning Curve

**Impact**: High (unfamiliarity could delay Phase 1-3)

**Mitigation**:
- Review SQLAlchemy 2.0 documentation before Phase 1
- Start with simple Epic model to learn patterns
- Use 04-backend-design.md as reference implementation
- Consult SQLAlchemy examples for complex patterns

### Risk 2: Performance Benchmarks Not Met

**Impact**: Medium (may require query optimization)

**Mitigation**:
- Use indexes defined in 03-data-design.md
- Follow query patterns from 07-performance-design.md
- Profile queries with EXPLAIN QUERY PLAN
- Optimize only if benchmarks fail (don't premature optimize)

### Risk 3: Alembic Migration Issues

**Impact**: Low (migrations are well-defined)

**Mitigation**:
- Follow exact DDL from 03-data-design.md
- Test upgrade and downgrade on temporary database
- Keep migration simple (all tables in one migration)

### Risk 4: Test Coverage Gaps

**Impact**: Medium (insufficient tests could miss bugs)

**Mitigation**:
- Aim for >80% coverage (PRD requirement)
- Focus on critical paths (CRUD, constraints, transactions)
- Use coverage.py to measure and report gaps
- Write tests before implementing (TDD where possible)

---

## Validation Checklist

Before declaring Phase 6 complete, verify:

### Code Quality
- [ ] All modules have docstrings
- [ ] All public functions have docstrings
- [ ] mypy passes with no errors
- [ ] pylint score >8.0
- [ ] No TODO or FIXME comments left

### Functionality
- [ ] All CRUD operations work
- [ ] All constraints enforced (foreign keys, CHECK, UNIQUE)
- [ ] All validations work (key formats, enums, ranges)
- [ ] All progress calculations accurate
- [ ] Cascade deletes work correctly
- [ ] Transactions rollback on errors

### Testing
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All performance benchmarks pass
- [ ] Test coverage >80%
- [ ] No test warnings or failures

### Performance
- [ ] Database init <500ms
- [ ] Single INSERT <50ms
- [ ] get_by_key <10ms
- [ ] Filter queries <100ms (10,000 tasks)
- [ ] Bulk INSERT <2,000ms (100 tasks)

### Security
- [ ] All queries parameterized (no SQL injection)
- [ ] File permissions set (600 on Unix)
- [ ] No sensitive data logged
- [ ] Foreign keys enabled and verified

### Documentation
- [ ] README.md complete
- [ ] Schema diagram accurate
- [ ] Migration guide complete
- [ ] Usage examples work
- [ ] Integration guide written

---

## Handoff to Next Features

### E04-F02: CLI Infrastructure

**What they need from E04-F01**:
- `SessionFactory` for dependency injection
- Repository classes for each entity
- `to_dict()` methods for JSON serialization
- Custom exceptions for error handling

**Integration pattern**:
```python
# CLI command receives repositories via context
@cli.command()
@click.pass_context
def task_list(ctx):
    task_repo = ctx.obj['task_repo']  # Injected
    with ctx.obj['session_factory'].get_session():
        tasks = task_repo.list_all()
        for task in tasks:
            print(json.dumps(task.to_dict()))
```

### E04-F03: Task Lifecycle Operations

**What they need from E04-F01**:
- `TaskRepository.update_status()` for atomic status changes
- `TaskHistoryRepository` for audit trail
- Validation functions for status transitions

### E04-F05: Folder Management

**What they need from E04-F01**:
- Transaction support for atomic file + DB updates
- `Task.file_path` field
- `TaskRepository.update()` to update file paths

### E04-F07: Initialization & Sync

**What they need from E04-F01**:
- Bulk insert support (single transaction)
- `init_database()` for setup
- Alembic migrations for schema creation

---

## Success Metrics

**Definition of Done for E04-F01**:
1. All 6 phases complete
2. All validation checklist items checked
3. All tests passing (unit, integration, performance)
4. All design documents followed
5. Ready for CLI integration (E04-F02)

**Quality Gates**:
- Code review by tech lead
- Performance benchmarks verified
- Security checklist reviewed
- Documentation reviewed

---

**Implementation Phases Complete**: 2025-12-14
**Ready for Development**: All specifications defined
**Next Step**: Begin Phase 1 implementation or dispatch to tdd-agent for 09-test-criteria.md