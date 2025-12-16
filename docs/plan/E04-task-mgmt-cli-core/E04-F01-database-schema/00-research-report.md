# Research Report: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: project-research-agent

## Executive Summary

This is a **greenfield project** with comprehensive planning documentation but no existing codebase. The research focuses on identifying industry best practices, Python ecosystem standards, and architectural patterns for building a SQLite-backed task management system with SQLAlchemy ORM.

**Key Findings**:
- No existing code to analyze - this is the foundational feature
- Project documentation follows a structured epic/feature breakdown
- Python 3.10+ target enables modern type hints and pattern matching
- Industry standard stack: SQLAlchemy 2.0+, Alembic, Click, Rich

---

## Project Context

### Current State

**Documentation Analyzed**:
- `/docs/plan/E04-task-mgmt-cli-core/epic.md` - Epic-level PRD
- `/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/prd.md` - Feature PRD
- `/docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/prd.md` - Related CLI feature

**Existing Code**: None - this is Feature 01 of Epic 04 (the foundation)

**Related Features** (dependencies):
1. **E04-F02**: CLI Infrastructure (depends on this database layer)
2. **E04-F03**: Task Lifecycle Operations (depends on this database layer)
3. **E04-F04**: Epic & Feature Queries (depends on this database layer)
4. **E04-F05**: Folder Management (parallel - integrates with this layer)
5. **E04-F06**: Task Creation (depends on this database layer)
6. **E04-F07**: Initialization & Sync (depends on this database layer)

All downstream features depend on the database schema and data access layer defined here.

### Business Requirements Summary

From the PRD, this feature must provide:

1. **Data Storage**: SQLite database (`project.db`) with 4 tables
2. **Data Integrity**: Foreign key constraints, CHECK constraints, unique indexes
3. **ORM Layer**: SQLAlchemy models with type hints
4. **Data Access**: CRUD operations, filtering, progress calculations
5. **Transactions**: Atomic multi-step operations with rollback
6. **Migrations**: Alembic-based schema versioning
7. **Performance**: <100ms queries, <50ms inserts for 10K task datasets

---

## Technology Stack Analysis

### Recommended Stack (Industry Standard)

Based on PRD requirements and Python ecosystem best practices:

| Component | Recommended | Version | Rationale |
|-----------|-------------|---------|-----------|
| **Database** | SQLite | 3.35+ | Embedded, zero-config, ACID-compliant, portable |
| **ORM** | SQLAlchemy | 2.0+ | Industry standard, type-safe, excellent docs |
| **Migrations** | Alembic | 1.13+ | Official SQLAlchemy migration tool |
| **Type Checking** | mypy | 1.0+ | Static type validation for Python 3.10+ |
| **Testing** | pytest | 7.0+ | Standard Python testing framework |
| **Date/Time** | Python stdlib | datetime + timezone | UTC timestamps, no external dependency |
| **JSON** | Python stdlib | json module | SQLite JSON1 extension for queries |

### SQLAlchemy 2.0 Key Features

**Why SQLAlchemy 2.0+**:
- Native Python type hints integration
- Modern declarative base syntax
- Improved session management
- Better async support (future-proofing)
- Explicit relationship loading (no lazy loading surprises)

**Pattern to use**:
```python
# SQLAlchemy 2.0 declarative syntax
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

class Base(DeclarativeBase):
    pass

class Task(Base):
    __tablename__ = "tasks"

    id: Mapped[int] = mapped_column(primary_key=True)
    title: Mapped[str] = mapped_column(String(255))
    status: Mapped[str] = mapped_column(String(50))
    # Type hints provide IDE autocomplete and mypy validation
```

### SQLite Configuration

**Required PRAGMA settings** (from PRD and best practices):

```python
# Enable foreign key constraints (not default in SQLite)
PRAGMA foreign_keys = ON;

# Use WAL mode for better concurrency (from PRD Non-Functional Requirements)
PRAGMA journal_mode = WAL;

# Use IMMEDIATE isolation to prevent lock escalation (from PRD)
# Applied per transaction, not per connection

# Verify database integrity on startup (from PRD)
PRAGMA integrity_check;
```

**WAL Mode Benefits**:
- Readers don't block writers
- Better crash recovery
- Faster in most scenarios
- Industry standard for SQLite applications

---

## Naming Conventions

Since this is a new project, we need to establish conventions:

### Database Naming

**Tables**: Lowercase, plural, snake_case
- `epics`, `features`, `tasks`, `task_history`

**Columns**: Lowercase, snake_case
- `epic_id`, `created_at`, `progress_pct`, `depends_on`

**Enums**: Lowercase, snake_case values
- Status: `draft`, `active`, `completed`, `archived`, `in_progress`, `ready_for_review`
- Priority: `high`, `medium`, `low`
- Agent type: `frontend`, `backend`, `api`, `testing`, `devops`, `general`

**Keys**: Uppercase prefix, zero-padded numbers
- Epic: `E##` (e.g., `E04`)
- Feature: `E##-F##` (e.g., `E04-F01`)
- Task: `T-E##-F##-###` (e.g., `T-E04-F01-001`)

### Python Naming

**Modules**: Lowercase, snake_case
- `database/models.py`
- `database/session.py`
- `database/repositories.py`

**Classes**: PascalCase
- `Epic`, `Feature`, `Task`, `TaskHistory`
- `DatabaseSession`, `TaskRepository`

**Functions/Methods**: Lowercase, snake_case
- `create_task()`, `get_by_id()`, `filter_by_status()`

**Type Hints**: Use modern Python 3.10+ syntax
- `list[Task]` instead of `List[Task]`
- `dict[str, Any]` instead of `Dict[str, Any]`
- `| None` instead of `Optional[...]`

---

## File Organization Patterns

### Recommended Project Structure

Based on Python best practices and PRD requirements:

```
pm/                              # Root package
├── __init__.py
├── __main__.py                  # Entry point: python -m pm
├── database/                    # E04-F01: Database layer
│   ├── __init__.py
│   ├── models.py                # SQLAlchemy ORM models
│   ├── session.py               # Session factory, connection management
│   ├── repositories.py          # CRUD operations, business queries
│   ├── migrations/              # Alembic migration scripts
│   │   ├── alembic.ini
│   │   ├── env.py
│   │   └── versions/
│   │       └── 001_initial_schema.py
│   └── exceptions.py            # Custom database exceptions
├── cli/                         # E04-F02: CLI commands (future)
├── models/                      # Domain models (if separating from ORM)
└── utils/                       # Shared utilities

tests/
├── unit/
│   └── database/
│       ├── test_models.py
│       ├── test_repositories.py
│       └── test_session.py
└── integration/
    └── test_database_operations.py

alembic.ini                      # Alembic configuration (root level)
pyproject.toml                   # Modern Python packaging
```

**Rationale**:
- **Separation of concerns**: Database layer isolated in `database/` package
- **Testability**: Clear separation of unit vs integration tests
- **Modern packaging**: `pyproject.toml` instead of `setup.py`
- **Migration location**: Alembic migrations within `database/` for cohesion

### Alternative Considered: Repository Pattern

**Option 1: Active Record Pattern** (simpler, but couples domain to persistence)
```python
# Models have their own CRUD methods
task = Task(title="Example")
task.save()  # Model knows how to persist itself
```

**Option 2: Repository Pattern** (recommended - separates concerns)
```python
# Repository handles persistence
task = Task(title="Example")
task_repository.create(task)  # Repository manages persistence
```

**Recommendation**: Use **Repository Pattern** for:
- Better testability (can mock repositories)
- Clear separation of domain models from persistence
- Easier to swap persistence layers if needed
- Aligns with E04-F02 CLI requirement for dependency injection

---

## Testing Standards

From PRD Non-Functional Requirements and Python best practices:

### Coverage Target

- **Minimum**: >80% unit test coverage (from Epic NFR)
- **Critical paths**: 100% coverage for CRUD operations, constraints, transactions

### Testing Layers

1. **Unit Tests** (`tests/unit/database/`)
   - Test models in isolation (no database)
   - Test repository logic with in-memory SQLite
   - Test constraint validation
   - Test enum validation

2. **Integration Tests** (`tests/integration/`)
   - Test full database operations with file-based SQLite
   - Test transaction rollback
   - Test foreign key cascade deletes
   - Test migration up/down

3. **Performance Tests** (separate suite)
   - Benchmark queries against 10,000 task dataset
   - Verify <100ms query performance (from PRD)
   - Verify <50ms insert performance (from PRD)

### Test Database Strategy

```python
# Use in-memory SQLite for fast unit tests
@pytest.fixture
def db_session():
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    session = Session(engine)
    yield session
    session.close()

# Use temporary file SQLite for integration tests
@pytest.fixture
def db_file(tmp_path):
    db_path = tmp_path / "test.db"
    # Test with actual file to verify WAL mode, locking, etc.
```

---

## Existing Persistence Patterns

### Analysis: No Existing Patterns

This project has no existing codebase, so we establish patterns from scratch.

**Implications**:
- Freedom to choose best practices without legacy constraints
- Must document patterns clearly for future features
- Should follow industry standards to ease onboarding

### Pattern Recommendations

**1. Session Management** (from SQLAlchemy best practices):
```python
# Context manager for automatic session cleanup
@contextmanager
def get_db_session():
    session = SessionLocal()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()
```

**2. Transaction Management** (from PRD requirement 19-21):
```python
# Explicit transaction control for multi-step operations
with get_db_session() as session:
    with session.begin():  # Explicit transaction
        # Multiple operations here
        # Automatic rollback on exception
```

**3. Query Patterns**:
```python
# Use explicit select() statements (SQLAlchemy 2.0 style)
from sqlalchemy import select

stmt = select(Task).where(Task.status == "todo")
result = session.execute(stmt).scalars().all()
```

**4. Timestamp Management** (from PRD requirements 22-25):
```python
# Automatic timestamps via server defaults and onupdate
created_at = mapped_column(DateTime, server_default=func.now())
updated_at = mapped_column(DateTime, server_default=func.now(), onupdate=func.now())
```

---

## Schema Design Patterns

### Foreign Key Constraints

From PRD Functional Requirements 1-4:

```
epics
  ↓ (ON DELETE CASCADE)
features
  ↓ (ON DELETE CASCADE)
tasks
  ↓ (ON DELETE CASCADE)
task_history
```

**Pattern**: Cascade deletes ensure referential integrity automatically.

**Example**: Deleting Epic E04 deletes all its features, all their tasks, and all task history records.

### Check Constraints for Enums

From PRD Functional Requirement 5:

```sql
-- SQLAlchemy representation
status: Mapped[str] = mapped_column(
    String(50),
    CheckConstraint("status IN ('draft', 'active', 'completed', 'archived')")
)
```

**Alternative**: Use Python Enum + database validation:
```python
from enum import Enum

class EpicStatus(str, Enum):
    DRAFT = "draft"
    ACTIVE = "active"
    COMPLETED = "completed"
    ARCHIVED = "archived"

# In model:
status: Mapped[str] = mapped_column(
    String(50),
    CheckConstraint(f"status IN {tuple(s.value for s in EpicStatus)}")
)
```

### JSON Fields

From PRD Functional Requirement 3 (depends_on field):

```python
# SQLite JSON storage
depends_on: Mapped[str | None] = mapped_column(Text, nullable=True)

# Store as JSON string: '["T-E01-F01-001", "T-E01-F02-003"]'
# Query with SQLite JSON functions if needed (SQLite 3.38+)
```

**Validation pattern**:
```python
import json

def validate_depends_on(value: str | None) -> bool:
    if value is None:
        return True
    try:
        data = json.loads(value)
        return isinstance(data, list)
    except json.JSONDecodeError:
        return False
```

---

## Alembic Migration Patterns

### Initial Migration Strategy

From PRD Functional Requirements 26-29:

**Initial migration** (`001_initial_schema.py`):
```python
def upgrade():
    # Create all 4 tables with constraints
    op.create_table('epics', ...)
    op.create_table('features', ...)
    op.create_table('tasks', ...)
    op.create_table('task_history', ...)

def downgrade():
    # Reverse order (respecting foreign keys)
    op.drop_table('task_history')
    op.drop_table('tasks')
    op.drop_table('features')
    op.drop_table('epics')
```

**Migration Best Practices**:
1. Always include both `upgrade()` and `downgrade()`
2. Use transaction-based migrations for SQLite
3. Test migrations on copy of production data
4. Never modify existing migrations (create new ones)
5. Include descriptive comments

### Schema Version Tracking

Alembic automatically creates `alembic_version` table:
```sql
CREATE TABLE alembic_version (
    version_num VARCHAR(32) NOT NULL PRIMARY KEY
);
```

**Application startup check**:
```python
from alembic import command
from alembic.config import Config

def check_schema_version():
    # Verify current schema matches expected version
    # Raise SchemaVersionMismatch if outdated (PRD Req 30)
```

---

## Performance Considerations

### Indexing Strategy

From PRD Non-Functional Requirement (query performance <100ms):

**Automatic indexes** (from UNIQUE constraints):
- `epics.key` (UNIQUE → automatic index)
- `features.key` (UNIQUE → automatic index)
- `tasks.key` (UNIQUE → automatic index)

**Additional indexes needed** (from PRD "Could-Have" Story 13):
```sql
-- Frequent query: filter tasks by status
CREATE INDEX idx_tasks_status ON tasks(status);

-- Frequent query: filter tasks by epic (via feature)
CREATE INDEX idx_features_epic_id ON features(epic_id);

-- Frequent query: filter tasks by feature
CREATE INDEX idx_tasks_feature_id ON tasks(feature_id);

-- Frequent query: filter tasks by agent_type
CREATE INDEX idx_tasks_agent_type ON tasks(agent_type);

-- Frequent query: task history lookup
CREATE INDEX idx_task_history_task_id ON task_history(task_id);

-- Composite index for common query: status + priority
CREATE INDEX idx_tasks_status_priority ON tasks(status, priority);
```

**Index Tradeoffs**:
- **Pro**: Faster SELECT queries
- **Con**: Slower INSERT/UPDATE (index maintenance)
- **Decision**: Add indexes (PRD targets <100ms queries, writes are less frequent)

### Query Optimization

**Progress calculation** (from PRD Functional Requirement 17-18):

```python
# Efficient query: COUNT with filter
def calculate_feature_progress(feature_id: int) -> float:
    total = session.query(Task).filter_by(feature_id=feature_id).count()
    if total == 0:
        return 0.0
    completed = session.query(Task).filter_by(
        feature_id=feature_id,
        status="completed"
    ).count()
    return (completed / total) * 100.0

# More efficient: single query with conditional aggregation
from sqlalchemy import func, case

def calculate_feature_progress_optimized(feature_id: int) -> float:
    result = session.query(
        func.count(Task.id).label("total"),
        func.sum(case((Task.status == "completed", 1), else_=0)).label("completed")
    ).filter_by(feature_id=feature_id).first()

    if result.total == 0:
        return 0.0
    return (result.completed / result.total) * 100.0
```

**Recommendation**: Use optimized single-query version (reduces database round trips).

---

## Security Patterns

From PRD Non-Functional Requirements (Security):

### SQL Injection Prevention

**Required**: All queries use parameterized statements (PRD NFR)

```python
# GOOD: Parameterized query (SQLAlchemy handles this)
session.query(Task).filter(Task.status == status_value)

# BAD: String concatenation (NEVER do this)
session.execute(f"SELECT * FROM tasks WHERE status = '{status_value}'")
```

**SQLAlchemy automatically parameterizes** all queries, so this is handled by using the ORM correctly.

### File Permissions

From PRD NFR: "Database file permissions must be 600 (read/write owner only) on Unix systems"

```python
import os
import stat

def create_database(db_path: str):
    # Create database
    engine = create_engine(f"sqlite:///{db_path}")
    Base.metadata.create_all(engine)

    # Set permissions (Unix only)
    if os.name != 'nt':  # Not Windows
        os.chmod(db_path, stat.S_IRUSR | stat.S_IWUSR)  # 600
```

### No Sensitive Data Logging

From PRD NFR: "No sensitive data logging (passwords, tokens) in SQL logs"

**Implementation**:
```python
# Disable SQLAlchemy echo in production
engine = create_engine(
    f"sqlite:///{db_path}",
    echo=False,  # Never echo SQL in production
    echo_pool=False
)

# For debugging, use logging configuration
import logging
logging.getLogger('sqlalchemy.engine').setLevel(logging.WARNING)
```

---

## Error Handling Patterns

From PRD Functional Requirements 30-32:

### Custom Exception Hierarchy

```python
class DatabaseError(Exception):
    """Base exception for all database errors"""
    pass

class DatabaseNotFound(DatabaseError):
    """Raised when database file doesn't exist"""
    pass

class SchemaVersionMismatch(DatabaseError):
    """Raised when database schema version doesn't match application version"""
    pass

class IntegrityError(DatabaseError):
    """Raised for foreign key or unique constraint violations"""
    def __init__(self, message: str, constraint: str | None = None):
        super().__init__(message)
        self.constraint = constraint

class ValidationError(DatabaseError):
    """Raised when data fails validation (e.g., invalid key format)"""
    pass
```

### Translating SQLAlchemy Exceptions

```python
from sqlalchemy.exc import IntegrityError as SQLAlchemyIntegrityError

def create_task(session, task: Task) -> Task:
    try:
        session.add(task)
        session.commit()
        return task
    except SQLAlchemyIntegrityError as e:
        session.rollback()
        # Parse constraint name from exception
        if "FOREIGN KEY constraint failed" in str(e):
            raise IntegrityError(
                "Task references non-existent feature",
                constraint="feature_id"
            )
        elif "UNIQUE constraint failed" in str(e):
            raise IntegrityError(
                f"Task key {task.key} already exists",
                constraint="key"
            )
        raise DatabaseError(str(e))
```

---

## Related PRD Analysis

### Dependencies from Other Features

**E04-F02: CLI Infrastructure**
- Expects database session injection via context manager
- Requires JSON-serializable model outputs
- Needs consistent error handling (custom exceptions)

**E04-F03: Task Lifecycle Operations**
- Uses CRUD methods from this data access layer
- Requires transaction support for atomic status updates
- Depends on progress calculation methods

**E04-F05: Folder Management**
- Stores `file_path` in tasks table
- Requires transaction support for atomic file move + DB update
- Uses task status to determine folder location

**E04-F07: Initialization & Sync**
- Uses Alembic migrations for schema creation
- Imports existing markdown files into database
- Requires bulk insert operations

### Integration Points

1. **Session Management**:
   - CLI commands will use `get_db_session()` context manager
   - Each command gets isolated transaction

2. **Model Serialization**:
   - Models need `to_dict()` methods for JSON output
   - CLI `--json` flag depends on this

3. **Validation**:
   - Key format validation must be centralized
   - Used by both database layer and CLI input validation

4. **Error Codes**:
   - Custom exceptions map to CLI exit codes
   - IntegrityError → exit code 1
   - ValidationError → exit code 1
   - DatabaseNotFound → exit code 2

---

## Recommendations

### Critical Implementation Decisions

1. **Use SQLAlchemy 2.0+** with modern declarative syntax and type hints
2. **Repository Pattern** for data access (not Active Record)
3. **Single-query progress calculations** for performance
4. **Explicit transaction management** via context managers
5. **Alembic migrations** in `database/migrations/` subdirectory
6. **Custom exception hierarchy** for precise error handling
7. **In-memory SQLite for unit tests**, file-based for integration tests
8. **Composite indexes** for common query patterns (status + priority)

### Open Questions for Design Phase

1. **Dependency validation**: Should `depends_on` JSON references be validated at write time or read time? (PRD defers to E04-F03)
2. **Progress caching**: Should `features.progress_pct` be cached or always calculated? (PRD shows it as a column, implies caching)
3. **Enum storage**: Store as strings or integers? (PRD examples use strings)
4. **Timezone handling**: Store as UTC, display in local? (PRD specifies UTC storage, helper methods for display)

### Next Steps

1. **01-interface-contracts.md**: Define exact method signatures for repositories
2. **03-data-design.md**: Define complete table schemas with all constraints
3. **04-backend-design.md**: Detail repository implementations and session management
4. **06-security-design.md**: Expand security considerations across layers

---

## References

### Python Ecosystem

- SQLAlchemy 2.0 Documentation: https://docs.sqlalchemy.org/en/20/
- Alembic Tutorial: https://alembic.sqlalchemy.org/en/latest/tutorial.html
- Python Type Hints (PEP 484, 585): https://peps.python.org/pep-0484/
- pytest Best Practices: https://docs.pytest.org/en/stable/

### Similar Projects (for pattern reference)

- Alembic (self-managed migrations): https://github.com/sqlalchemy/alembic
- Flask-SQLAlchemy: https://github.com/pallets/flask-sqlalchemy
- FastAPI SQLAlchemy patterns: https://fastapi.tiangolo.com/tutorial/sql-databases/

### SQLite Resources

- SQLite JSON1 Extension: https://www.sqlite.org/json1.html
- SQLite WAL Mode: https://www.sqlite.org/wal.html
- SQLite Foreign Keys: https://www.sqlite.org/foreignkeys.html

---

**Research Complete**: 2025-12-14
**Next Agent**: Coordinator creates interface contracts (01-interface-contracts.md)
