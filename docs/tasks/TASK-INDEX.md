# Task Index: Database Schema & Core Data Model (E04-F01)

## Overview

This index tracks all implementation tasks for the Database Schema & Core Data Model feature (E04-F01).

**Feature Location**: `/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/`
**Total Tasks**: 6
**Estimated Total Time**: 48 hours (6 business days)

## Task List

| Task Key | Title | Status | Assigned Agent | Dependencies | Est. Time | Current Location |
|----------|-------|--------|----------------|--------------|-----------|------------------|
| T-E04-F01-001 | Database Foundation - Models & Schema | todo | general-purpose | None | 8 hours | docs/tasks/todo/ |
| T-E04-F01-002 | Session Management | todo | general-purpose | T-E04-F01-001 | 6 hours | docs/tasks/todo/ |
| T-E04-F01-003 | Repository Layer - CRUD Operations | todo | general-purpose | T-E04-F01-002 | 16 hours | docs/tasks/todo/ |
| T-E04-F01-004 | Integration Tests & Performance Validation | todo | general-purpose | T-E04-F01-003 | 8 hours | docs/tasks/todo/ |
| T-E04-F01-005 | Documentation & Usage Examples | todo | general-purpose | T-E04-F01-004 | 6 hours | docs/tasks/todo/ |
| T-E04-F01-006 | Package Export & CLI Integration Prep | todo | general-purpose | T-E04-F01-005 | 4 hours | docs/tasks/todo/ |

## Task Workflow

Tasks move through these folders based on their status:

```
docs/tasks/todo/           → New tasks created here
    ↓
docs/tasks/active/         → Moved when work begins
    ↓
docs/tasks/ready-for-review/ → Moved when implementation complete
    ↓
docs/tasks/completed/      → Moved after successful review
    ↓
docs/tasks/archived/       → Moved when no longer needed
```

## Status Definitions

- **todo**: Not started, waiting for dependencies or assignment
- **active**: Currently being worked on by an agent
- **blocked**: Waiting on external dependency or decision
- **ready-for-review**: Implementation complete, awaiting code review
- **completed**: Reviewed, validated, and merged
- **archived**: No longer active or superseded

## Execution Order

### Phase 1: Foundation - Models and Schema (8 hours)

**Task**: T-E04-F01-001
**Dependencies**: None

Create the foundational database layer including:
- ORM models (Epic, Feature, Task, TaskHistory) with full SQLAlchemy 2.0+ type hints
- Validation module with key format, enum, and JSON validation
- Custom exception hierarchy
- Database configuration
- Initial Alembic migration with all tables, constraints, indexes, and triggers

**Success Gates**:
- All 4 models defined with relationships
- Validation functions for all key formats and enums
- Migration creates complete schema
- No mypy type errors

### Phase 2: Session Management (6 hours)

**Task**: T-E04-F01-002
**Dependencies**: T-E04-F01-001

Implement database connection and session lifecycle:
- SessionFactory with SQLite configuration
- Context managers for automatic transaction management
- SQLite PRAGMA configuration (foreign_keys, WAL mode, busy_timeout)
- Database initialization and validation utilities
- File permission enforcement (Unix)

**Success Gates**:
- Transactions commit on success, rollback on exception
- Foreign keys enabled and verified
- File permissions set to 600 (Unix)
- 100% test coverage

### Phase 3: Repository Layer - CRUD Operations (16 hours)

**Task**: T-E04-F01-003
**Dependencies**: T-E04-F01-002

Build comprehensive data access layer:
- EpicRepository with CRUD and progress calculation
- FeatureRepository with filtering and progress tracking
- TaskRepository with complex filtering and atomic status updates
- TaskHistoryRepository for audit trail
- Input validation before database operations
- Error translation (SQLAlchemy → domain exceptions)

**Success Gates**:
- All CRUD operations implemented
- Multi-criteria filtering works
- Atomic status update + history creation
- Progress calculations accurate
- >90% test coverage

### Phase 4: Integration Tests & Performance Validation (8 hours)

**Task**: T-E04-F01-004
**Dependencies**: T-E04-F01-003

Validate end-to-end functionality and performance:
- Integration tests for complete epic → feature → task workflows
- Cascade delete verification
- Foreign key constraint enforcement tests
- Transaction rollback behavior tests
- Performance benchmarks against PRD targets:
  - Database init <500ms
  - Single INSERT <50ms
  - Query with filters <100ms (10K tasks)
  - Progress calculation <200ms (50 features)
  - Bulk INSERT <2s (100 tasks)

**Success Gates**:
- All integration tests passing
- All performance benchmarks meet targets
- Tests run successfully in parallel
- No database corruption after tests

### Phase 5: Documentation & Usage Examples (6 hours)

**Task**: T-E04-F01-005
**Dependencies**: T-E04-F01-004

Create comprehensive developer documentation:
- Database schema documentation with ASCII ER diagram
- Repository usage examples and patterns
- Migration guide (create, apply, rollback)
- Complete docstrings for all public APIs
- Working code examples for common use cases

**Success Gates**:
- All documentation files created
- All examples run successfully
- All public APIs have docstrings
- Schema diagram matches implementation

### Phase 6: Package Export & CLI Integration Prep (4 hours)

**Task**: T-E04-F01-006
**Dependencies**: T-E04-F01-005

Finalize package for downstream integration:
- Clean public API exports in `__init__.py`
- Smoke tests verify package imports
- mypy strict mode validation passes
- CLI integration guide created
- Integration checklist for downstream features
- Final validation checklist completed

**Success Gates**:
- Package imports cleanly
- Zero mypy errors (strict mode)
- Integration guide complete
- Ready for CLI integration (E04-F02)

## Design Documentation

All tasks reference these design documents in the feature directory:

- [PRD](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/prd.md) - Product requirements and acceptance criteria
- [00-Research Report](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/00-research-report.md) - Codebase analysis and patterns
- [01-Interface Contracts](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/01-interface-contracts.md) - API contracts and interfaces
- [02-Architecture](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/02-architecture.md) - System architecture and layers
- [03-Data Design](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/03-data-design.md) - Database schema and DDL
- [04-Backend Design](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/04-backend-design.md) - Implementation specifications
- [06-Security Design](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/06-security-design.md) - Security measures
- [07-Performance Design](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/07-performance-design.md) - Performance optimization
- [08-Implementation Phases](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/08-implementation-phases.md) - Detailed phase breakdown
- [09-Test Criteria](../docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/09-test-criteria.md) - Comprehensive test requirements

## Dependencies & Critical Path

```
T-E04-F01-001 (Foundation) [8h]
    ↓
T-E04-F01-002 (Session Mgmt) [6h]
    ↓
T-E04-F01-003 (Repositories) [16h]  ← CRITICAL PATH (longest task)
    ↓
T-E04-F01-004 (Integration Tests) [8h]
    ↓
T-E04-F01-005 (Documentation) [6h]
    ↓
T-E04-F01-006 (Package Export) [4h]
    ↓
READY FOR E04-F02 (CLI Infrastructure)
```

**Critical Path**: All tasks are sequential dependencies. Longest task is T-E04-F01-003 (Repository Layer) at 16 hours.

**Parallelization**: No parallel opportunities within this feature. All phases must complete sequentially.

## Downstream Feature Dependencies

Once all tasks are completed, these features can begin:

### E04-F02: CLI Infrastructure
**What they need**:
- `SessionFactory` for dependency injection
- All repository classes (EpicRepository, FeatureRepository, TaskRepository, TaskHistoryRepository)
- `to_dict()` methods for JSON serialization
- Custom exceptions for error handling

**Integration pattern**: CLI commands receive repositories via Click context, use context managers for transactions.

### E04-F03: Task Lifecycle Operations
**What they need**:
- `TaskRepository.update_status()` for atomic status changes
- `TaskHistoryRepository` for audit trail
- Validation functions for status transitions

### E04-F05: Folder Management
**What they need**:
- Transaction support for atomic file + DB updates
- `Task.file_path` field
- `TaskRepository.update()` to update file paths

### E04-F07: Initialization & Sync
**What they need**:
- Bulk insert support (single transaction)
- `init_database()` for setup
- Alembic migrations for schema creation

## Feature Metrics

**Code Volume Estimate**:
- `models.py`: ~900 lines
- `validation.py`: ~300 lines
- `exceptions.py`: ~100 lines
- `config.py`: ~100 lines
- `session.py`: ~200 lines
- `repositories.py`: ~1500 lines
- Tests: ~1400 lines
- **Total**: ~4500 lines of production code + tests

**Test Coverage Targets**:
- `models.py`: 90%
- `validation.py`: 95%
- `session.py`: 100%
- `repositories.py`: 90%
- `exceptions.py`: 100%
- `config.py`: 80%
- **Overall**: >85% code coverage

**Performance Targets** (from PRD):
- Database initialization: <500ms
- Single task INSERT: <50ms
- get_by_key query: <10ms
- Filter queries (10K tasks): <100ms
- Progress calculation (50 features): <200ms
- Bulk INSERT (100 tasks): <2,000ms
- CASCADE DELETE: <500ms

## Notes

- **Database-Only Feature**: This is a backend-only feature with no frontend/API components, so Contract Validation phase (T-E##-F##-001) is not applicable.
- **SQLite Focus**: Implementation targets SQLite 3.35+ with specific optimizations (WAL mode, foreign keys, etc.). PostgreSQL migration possible but out of scope.
- **Type Safety Critical**: Full Python 3.10+ type hints required throughout. Mypy strict mode must pass.
- **Security Measures**: File permissions (600 on Unix), parameterized queries (no SQL injection), no sensitive data logging.
- **Documentation Required**: Comprehensive docstrings, usage examples, migration guide, and ER diagram mandatory for handoff to downstream features.

---

**Task Index Created**: 2025-12-14
**Feature Status**: Design complete, ready for implementation
**Next Action**: Begin Phase 1 (T-E04-F01-001) or assign to implementation agent
