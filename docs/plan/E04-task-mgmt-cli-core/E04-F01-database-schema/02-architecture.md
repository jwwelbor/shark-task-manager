# Architecture Design: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: backend-architect

## Purpose

This document defines the high-level architecture of the database layer, including component organization, layer separation, dependency flow, and integration patterns. It establishes the architectural principles that guide the detailed backend design.

---

## Architectural Overview

### System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                      shark Task Management CLI                      │
│                                                                  │
│  ┌─────────────────┐   ┌──────────────────┐   ┌──────────────┐ │
│  │   CLI Layer     │   │   File Layer     │   │  Sync Layer  │ │
│  │  (E04-F02)      │   │   (E04-F05)      │   │  (E04-F07)   │ │
│  └────────┬────────┘   └────────┬─────────┘   └──────┬───────┘ │
│           │                     │                     │          │
│           └─────────────────────┼─────────────────────┘          │
│                                 │                                │
│                                 ▼                                │
│                    ┌─────────────────────────┐                  │
│                    │   DATA ACCESS LAYER     │                  │
│                    │   (THIS FEATURE)        │                  │
│                    │                         │                  │
│                    │  ┌──────────────────┐   │                  │
│                    │  │  Repositories    │   │                  │
│                    │  │  (CRUD + Query)  │   │                  │
│                    │  └────────┬─────────┘   │                  │
│                    │           │             │                  │
│                    │  ┌────────▼─────────┐   │                  │
│                    │  │   ORM Models     │   │                  │
│                    │  │  (SQLAlchemy)    │   │                  │
│                    │  └────────┬─────────┘   │                  │
│                    │           │             │                  │
│                    │  ┌────────▼─────────┐   │                  │
│                    │  │ Session Manager  │   │                  │
│                    │  │ (Transactions)   │   │                  │
│                    │  └────────┬─────────┘   │                  │
│                    └───────────┼─────────────┘                  │
│                                │                                │
│                                ▼                                │
│                    ┌─────────────────────────┐                  │
│                    │   SQLite Database       │                  │
│                    │   (project.db)          │                  │
│                    └─────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Principles**:
1. **Layered Architecture**: Clear separation between CLI, data access, and persistence
2. **Repository Pattern**: Data access logic isolated in repositories
3. **ORM Abstraction**: SQLAlchemy provides database independence
4. **Session Management**: Centralized transaction handling
5. **Type Safety**: Full type hints enable IDE support and mypy validation

---

## Architectural Layers

### Layer 1: Database Layer (Physical Storage)

**Technology**: SQLite 3.35+
**File**: `project.db` (WAL mode)

**Responsibilities**:
- Physical data storage
- ACID transaction guarantees
- Constraint enforcement (foreign keys, CHECK, UNIQUE)
- Index management for query performance
- Trigger execution for automated timestamps

**Configuration**:
```python
# SQLite connection configuration
DATABASE_URL = f"sqlite:///{project_root}/project.db"

# Connection options
connect_args = {
    "check_same_thread": False,  # Allow multi-threaded access
    "timeout": 5.0,              # 5 second timeout for locks
}

# Required PRAGMAs
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
```

---

### Layer 2: ORM Layer (Object-Relational Mapping)

**Technology**: SQLAlchemy 2.0+

**Responsibilities**:
- Map database tables to Python classes
- Provide type-safe field access
- Generate SQL queries from Python expressions
- Manage relationships (epic→features→tasks)
- Handle lazy/eager loading of related objects

**Component Structure**:
```
database/
├── __init__.py              # Package exports
├── models.py                # ORM model definitions
│   ├── Base                 # Declarative base
│   ├── Epic                 # Epic model
│   ├── Feature              # Feature model
│   ├── Task                 # Task model
│   └── TaskHistory          # TaskHistory model
└── ...
```

**Key Design Decisions**:

1. **SQLAlchemy 2.0 Declarative Syntax**:
   ```python
   from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

   class Base(DeclarativeBase):
       pass

   class Epic(Base):
       __tablename__ = "epics"

       id: Mapped[int] = mapped_column(primary_key=True)
       key: Mapped[str] = mapped_column(String(10), unique=True)
       # Type hints provide IDE autocomplete and mypy validation
   ```

2. **Relationship Definitions**:
   ```python
   class Epic(Base):
       # One-to-many: epic has many features
       features: Mapped[list["Feature"]] = relationship(
           back_populates="epic",
           cascade="all, delete-orphan"
       )

   class Feature(Base):
       # Many-to-one: feature belongs to epic
       epic: Mapped["Epic"] = relationship(back_populates="features")
   ```

3. **Explicit Loading** (no lazy loading surprises):
   - Use `selectinload()` for eager loading when needed
   - Avoid N+1 query problems
   - Make loading strategy explicit in queries

---

### Layer 3: Session Management Layer

**Responsibilities**:
- Create and manage database sessions
- Handle transaction lifecycle (begin, commit, rollback)
- Provide context managers for automatic cleanup
- Enforce isolation levels
- Connection pooling (SQLite: single connection)

**Component Structure**:
```
database/
├── session.py               # Session factory and context managers
│   ├── SessionFactory       # Creates sessions
│   ├── get_db_session()     # Context manager for transactions
│   └── init_database()      # Initialize database + check version
```

**Session Lifecycle**:
```python
# Automatic session management
with get_db_session() as session:
    # Session created
    task = session.query(Task).get(1)
    task.status = "completed"
    # Automatic commit on success
# Session closed, resources freed

# Manual rollback on exception
with get_db_session() as session:
    task.status = "invalid"
    raise Exception("Validation failed")
    # Automatic rollback
```

**Transaction Isolation**:
- SQLite default: SERIALIZABLE (strictest)
- Write operations use `BEGIN IMMEDIATE` to prevent lock escalation
- Read operations use `BEGIN DEFERRED` (default)

---

### Layer 4: Repository Layer (Data Access Logic)

**Responsibilities**:
- Implement CRUD operations
- Provide domain-specific query methods
- Encapsulate complex query logic
- Handle validation and error translation
- Manage business logic (e.g., progress calculation)

**Component Structure**:
```
database/
├── repositories.py          # Repository implementations
│   ├── EpicRepository       # Epic CRUD and queries
│   ├── FeatureRepository    # Feature CRUD and queries
│   ├── TaskRepository       # Task CRUD and complex queries
│   └── TaskHistoryRepository # History access
└── exceptions.py            # Custom database exceptions
```

**Repository Pattern Benefits**:
1. **Separation of Concerns**: Domain logic separate from persistence
2. **Testability**: Easy to mock repositories for unit tests
3. **Query Centralization**: All queries in one place
4. **Error Handling**: Translate SQLAlchemy exceptions to domain exceptions
5. **Reusability**: Repositories used by CLI, sync, and file management

**Example Repository Method**:
```python
class TaskRepository:
    def __init__(self, session: Session):
        self.session = session

    def filter_combined(self,
                       status: str | None = None,
                       epic_key: str | None = None,
                       agent_type: str | None = None) -> list[Task]:
        """Complex query with multiple filters"""
        query = select(Task)

        if status:
            query = query.where(Task.status == status)
        if agent_type:
            query = query.where(Task.agent_type == agent_type)
        if epic_key:
            query = query.join(Feature).join(Epic).where(Epic.key == epic_key)

        return self.session.execute(query).scalars().all()
```

---

## Component Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                    DATABASE PACKAGE                             │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   repositories.py                          │ │
│  │                                                            │ │
│  │  ┌─────────────────┐  ┌──────────────────┐               │ │
│  │  │ EpicRepository  │  │ FeatureRepository│               │ │
│  │  │                 │  │                  │               │ │
│  │  │ - create()      │  │ - create()       │               │ │
│  │  │ - get_by_key()  │  │ - get_by_key()   │               │ │
│  │  │ - list_all()    │  │ - list_by_epic() │               │ │
│  │  │ - update()      │  │ - update()       │               │ │
│  │  │ - delete()      │  │ - calculate_...()│               │ │
│  │  └────────┬────────┘  └────────┬─────────┘               │ │
│  │           │                    │                          │ │
│  │  ┌────────▼────────┐  ┌────────▼─────────┐               │ │
│  │  │ TaskRepository  │  │ HistoryRepository│               │ │
│  │  │                 │  │                  │               │ │
│  │  │ - create()      │  │ - create()       │               │ │
│  │  │ - get_by_key()  │  │ - list_by_task() │               │ │
│  │  │ - filter_...()  │  │ - list_recent()  │               │ │
│  │  │ - update_...()  │  │                  │               │ │
│  │  └────────┬────────┘  └────────┬─────────┘               │ │
│  └───────────┼──────────────────────┼─────────────────────────┘ │
│              │                      │                          │
│              │  Uses                │                          │
│              ▼                      ▼                          │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                      models.py                             │ │
│  │                                                            │ │
│  │  ┌──────┐  ┌─────────┐  ┌──────┐  ┌─────────────┐        │ │
│  │  │ Epic │  │ Feature │  │ Task │  │ TaskHistory │        │ │
│  │  └───┬──┘  └────┬────┘  └───┬──┘  └──────┬──────┘        │ │
│  │      │          │           │            │               │ │
│  │      └──────────┴───────────┴────────────┘               │ │
│  │                     │                                     │ │
│  │              Inherits from                                │ │
│  │                     │                                     │ │
│  │                ┌────▼────┐                                │ │
│  │                │  Base   │  (DeclarativeBase)             │ │
│  │                └─────────┘                                │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                      session.py                            │ │
│  │                                                            │ │
│  │  ┌─────────────────┐  ┌────────────────────┐             │ │
│  │  │ SessionFactory  │  │ get_db_session()   │             │ │
│  │  │                 │  │ (context manager)  │             │ │
│  │  │ - engine        │  └────────────────────┘             │ │
│  │  │ - SessionLocal  │                                     │ │
│  │  │ - init_db()     │  ┌────────────────────┐             │ │
│  │  │ - check_...()   │  │ init_database()    │             │ │
│  │  └─────────────────┘  └────────────────────┘             │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   exceptions.py                            │ │
│  │                                                            │ │
│  │  DatabaseError → IntegrityError                            │ │
│  │               → ValidationError                            │ │
│  │               → DatabaseNotFound                           │ │
│  │               → SchemaVersionMismatch                      │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   migrations/                              │ │
│  │                                                            │ │
│  │  alembic.ini                                               │ │
│  │  env.py                                                    │ │
│  │  versions/                                                 │ │
│  │    └── 001_initial_schema.py                              │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## Dependency Flow

### Compile-Time Dependencies

```
repositories.py
    ↓ imports
models.py
    ↓ imports
session.py (for Base, engine)

session.py
    ↓ imports
sqlalchemy
```

**No Circular Dependencies**:
- Models don't import repositories
- Repositories import models (one direction)
- Session factory is independent

### Runtime Dependencies

```
CLI Command
    ↓ receives
Repository Instance
    ↓ uses
Session (from context manager)
    ↓ queries
ORM Models
    ↓ translates to
SQL Queries
    ↓ executes on
SQLite Database
```

---

## Data Flow Patterns

### Create Operation (Task Creation)

```
1. CLI Layer
   └─> TaskRepository.create(...)

2. Repository Layer
   ├─> Validate input (key format, enums)
   ├─> Create ORM object: Task(...)
   ├─> session.add(task)
   └─> session.commit()

3. ORM Layer
   └─> Translate to INSERT SQL

4. Database Layer
   ├─> Execute INSERT
   ├─> Enforce constraints
   ├─> Fire triggers (updated_at)
   └─> Return auto-generated ID

5. Response Flow (reversed)
   ├─> ORM refreshes object with DB values
   ├─> Repository returns Task object
   └─> CLI formats output (JSON or table)
```

### Complex Query (Filter Tasks)

```
1. CLI Layer
   └─> TaskRepository.filter_combined(status="todo", epic_key="E04")

2. Repository Layer
   ├─> Build query: select(Task).where(...)
   ├─> Add JOINs: .join(Feature).join(Epic)
   ├─> Add filters: .where(Task.status == "todo")
   └─> session.execute(query)

3. ORM Layer
   ├─> Translate to SQL SELECT with JOINs
   └─> Map result rows to Task objects

4. Database Layer
   ├─> Use indexes: idx_tasks_status, idx_features_epic_id
   ├─> Execute JOINs
   └─> Return result set

5. Response Flow
   ├─> ORM creates list[Task]
   ├─> Repository returns list
   └─> CLI formats as table or JSON
```

### Atomic Multi-Step Operation (Update Status)

```
1. CLI Layer
   └─> TaskRepository.update_status(task_id, "completed", agent="user")

2. Repository Layer (within transaction)
   ├─> BEGIN TRANSACTION
   │
   ├─> Get task: task = session.get(Task, task_id)
   ├─> Store old_status = task.status
   │
   ├─> Update task:
   │   ├─> task.status = "completed"
   │   ├─> task.completed_at = utc_now()
   │   └─> session.flush()  # Apply to DB, stay in transaction
   │
   ├─> Create history:
   │   ├─> history = TaskHistory(task_id, old_status, "completed", agent)
   │   └─> session.add(history)
   │
   └─> COMMIT (or ROLLBACK on exception)

3. Database Layer
   ├─> Execute UPDATE tasks ...
   ├─> Fire trigger: tasks_updated_at
   ├─> Execute INSERT task_history ...
   └─> Commit transaction (atomic)
```

---

## Integration Patterns

### Pattern 1: Dependency Injection (CLI Commands)

CLI commands receive repositories via context:

```python
# CLI setup (in E04-F02)
@click.group()
@click.pass_context
def cli(ctx):
    """Initialize database session and repositories"""
    session_factory = SessionFactory()
    ctx.obj = {
        'session_factory': session_factory,
        'epic_repo': EpicRepository(session_factory),
        'task_repo': TaskRepository(session_factory),
        # ...
    }

# Command usage
@cli.command()
@click.pass_context
def task_list(ctx):
    task_repo = ctx.obj['task_repo']
    with task_repo.session_factory.get_session():
        tasks = task_repo.list_all()
        # Format and display
```

### Pattern 2: Transaction Context (File + DB Updates)

Atomic file move + database update (from E04-F05):

```python
from database import get_db_session
from file_manager import move_task_file

def move_task_to_folder(task_key: str, new_status: str):
    """Atomically update file location and database"""
    with get_db_session() as session:
        task_repo = TaskRepository(session)

        # Get current task
        task = task_repo.get_by_key(task_key)
        old_path = task.file_path

        # Move file (can raise exception)
        new_path = move_task_file(old_path, new_status)

        # Update database (within same transaction)
        task_repo.update_status(task.id, new_status)
        task_repo.update(task.id, file_path=new_path)

        # Both succeed or both fail (transaction)
```

### Pattern 3: Bulk Operations (Sync Existing Files)

Import 100 existing tasks efficiently (from E04-F07):

```python
def sync_tasks_from_files(task_files: list[Path]):
    """Import tasks in bulk transaction"""
    with get_db_session() as session:
        task_repo = TaskRepository(session)

        for file_path in task_files:
            # Parse markdown frontmatter
            task_data = parse_task_file(file_path)

            # Create task in session (not committed yet)
            task_repo.create(**task_data)

        # Single commit for all tasks (fast)
        # Rollback all if any task fails validation
```

### Pattern 4: Progress Calculation (Cached vs Computed)

Feature progress is cached but recalculated on task status changes:

```python
def update_task_status(task_id: int, new_status: str):
    """Update task and recalculate feature progress"""
    with get_db_session() as session:
        task_repo = TaskRepository(session)
        feature_repo = FeatureRepository(session)

        # Update task status
        task = task_repo.update_status(task_id, new_status)

        # Recalculate and cache feature progress
        feature_repo.update_progress(task.feature_id)

        # Both updates in one transaction
```

---

## Error Handling Architecture

### Exception Translation

SQLAlchemy exceptions are translated to domain exceptions:

```python
# In repository methods
try:
    session.add(task)
    session.commit()
except SQLAlchemyIntegrityError as e:
    session.rollback()

    # Translate to domain exception
    if "FOREIGN KEY constraint failed" in str(e):
        raise IntegrityError(
            f"Task references non-existent feature (feature_id={task.feature_id})"
        )
    elif "UNIQUE constraint failed" in str(e):
        raise IntegrityError(f"Task key {task.key} already exists")
    else:
        raise DatabaseError(str(e))
```

### Error Propagation

```
Database Layer
    ↓ SQLite Error
ORM Layer
    ↓ SQLAlchemyIntegrityError
Repository Layer
    ↓ Translate to IntegrityError
CLI Layer
    ↓ Catch and format
User
    ← Error message + exit code
```

---

## Configuration Management

### Database Configuration

```python
# database/config.py
from pathlib import Path
from typing import Optional

class DatabaseConfig:
    """Database configuration settings"""

    def __init__(self,
                 db_path: Optional[Path] = None,
                 echo: bool = False,
                 pool_size: int = 1):
        self.db_path = db_path or Path.cwd() / "project.db"
        self.echo = echo  # SQL query logging
        self.pool_size = pool_size  # SQLite: always 1

    @property
    def database_url(self) -> str:
        return f"sqlite:///{self.db_path}"

    @property
    def connect_args(self) -> dict:
        return {
            "check_same_thread": False,
            "timeout": 5.0,
        }
```

### Environment-Specific Overrides

```python
# Development
config = DatabaseConfig(
    db_path=Path("dev_project.db"),
    echo=True  # Log SQL queries
)

# Testing
config = DatabaseConfig(
    db_path=Path(":memory:"),  # In-memory database
    echo=False
)

# Production
config = DatabaseConfig(
    db_path=Path("/var/pm/project.db"),
    echo=False  # No SQL logging
)
```

---

## Performance Architecture

### Query Optimization Strategy

1. **Index Usage**:
   - All foreign keys indexed automatically
   - Additional indexes on frequently filtered columns
   - Composite indexes for common multi-column queries

2. **Eager Loading**:
   ```python
   # Load task with related feature and epic (single query)
   stmt = select(Task).options(
       selectinload(Task.feature).selectinload(Feature.epic)
   )
   ```

3. **Batch Operations**:
   ```python
   # Bulk insert (one transaction, minimal overhead)
   with session.begin():
       for task_data in tasks:
           session.add(Task(**task_data))
   ```

4. **Cached Calculations**:
   - `features.progress_pct` cached in database
   - Recalculated only when task status changes
   - Avoids repeated aggregation queries

### Connection Pooling

SQLite uses single connection (no pool needed):
```python
engine = create_engine(
    database_url,
    poolclass=StaticPool,  # Single connection
    connect_args={"check_same_thread": False}
)
```

---

## Migration Architecture

### Alembic Integration

```
database/
└── migrations/
    ├── alembic.ini          # Alembic configuration
    ├── env.py               # Migration environment
    └── versions/
        ├── 001_initial_schema.py
        └── 002_future_migration.py (example)
```

### Migration Workflow

```
1. Developer creates migration:
   $ alembic revision -m "Add estimation fields"

2. Alembic generates template:
   versions/002_add_estimation_fields.py

3. Developer implements upgrade/downgrade:
   def upgrade(): op.add_column(...)
   def downgrade(): op.drop_column(...)

4. Apply migration:
   $ alembic upgrade head

5. Application checks version on startup:
   if current_version != expected_version:
       raise SchemaVersionMismatch(...)
```

---

## Testing Architecture

### Unit Testing (Isolated)

```python
# Test repository logic with in-memory database
@pytest.fixture
def in_memory_session():
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    session = Session(engine)
    yield session
    session.close()

def test_task_repository_create(in_memory_session):
    repo = TaskRepository(in_memory_session)
    task = repo.create(feature_id=1, key="T-E01-F01-001", ...)
    assert task.id is not None
```

### Integration Testing (Real Database)

```python
# Test with temporary file-based database
@pytest.fixture
def temp_db(tmp_path):
    db_path = tmp_path / "test.db"
    engine = create_engine(f"sqlite:///{db_path}")
    Base.metadata.create_all(engine)
    session = Session(engine)
    yield session
    session.close()

def test_cascade_delete(temp_db):
    """Verify CASCADE DELETE behavior"""
    epic_repo = EpicRepository(temp_db)
    task_repo = TaskRepository(temp_db)

    # Create epic → feature → task
    epic = epic_repo.create(key="E01", ...)
    # ... create feature and task

    # Delete epic
    epic_repo.delete(epic.id)

    # Verify cascade
    tasks = task_repo.list_all()
    assert len(tasks) == 0  # Task deleted automatically
```

---

## Security Architecture

### SQL Injection Prevention

**All queries parameterized** (enforced by ORM):
```python
# SAFE: Parameterized query
stmt = select(Task).where(Task.status == user_input)
session.execute(stmt)

# NEVER do this (not using ORM):
# UNSAFE: String concatenation
session.execute(f"SELECT * FROM tasks WHERE status = '{user_input}'")
```

### File Permission Enforcement

```python
def init_database(db_path: Path):
    """Create database with secure permissions"""
    engine = create_engine(f"sqlite:///{db_path}")
    Base.metadata.create_all(engine)

    # Set restrictive permissions (Unix only)
    if os.name != 'nt':
        os.chmod(db_path, 0o600)  # Owner read/write only
        if db_path.with_suffix(".db-wal").exists():
            os.chmod(db_path.with_suffix(".db-wal"), 0o600)
        if db_path.with_suffix(".db-shm").exists():
            os.chmod(db_path.with_suffix(".db-shm"), 0o600)
```

---

## Scalability Considerations

### Current Scale Targets

From PRD:
- **Dataset size**: 10,000 tasks
- **Query performance**: <100ms for filtered queries
- **Insert performance**: <50ms per task
- **Concurrent users**: 1 (single developer + agents)

### Future Scaling Path

If scale requirements grow:

1. **SQLite → PostgreSQL**:
   - SQLAlchemy abstracts database (minimal code changes)
   - Update connection string and migration tool
   - No repository or model changes needed

2. **Read Replicas**:
   - Separate read/write session factories
   - Route SELECT queries to replica
   - Route INSERT/UPDATE to primary

3. **Caching Layer**:
   - Add Redis/Memcached for frequently accessed data
   - Cache task counts, progress percentages
   - Invalidate on updates

**Current decision**: SQLite sufficient for 10,000 tasks, single developer use case.

---

## Deployment Architecture

### Database File Location

**Development**:
```
project_root/
├── project.db           # Main database
├── project.db-wal       # Write-Ahead Log
└── project.db-shm       # Shared memory
```

**Production** (system-wide install):
```
/var/pm/
├── project.db
├── project.db-wal
└── project.db-shm
```

### Initialization Sequence

```
1. Application Startup
   ↓
2. Check if database exists
   ├─ No: Create database + run migrations
   └─ Yes: Check schema version
       ├─ Match: Continue
       └─ Mismatch: Raise SchemaVersionMismatch

3. Enable SQLite PRAGMAs
   ├─ PRAGMA foreign_keys = ON
   ├─ PRAGMA journal_mode = WAL
   └─ PRAGMA busy_timeout = 5000

4. Run integrity check
   └─ PRAGMA integrity_check

5. Create session factory
   └─ Ready for queries
```

---

## Monitoring and Observability

### Logging Strategy

```python
import logging

logger = logging.getLogger("pm.database")

# In repositories
def create_task(self, **data):
    logger.info(f"Creating task: {data['key']}")
    try:
        task = Task(**data)
        self.session.add(task)
        self.session.commit()
        logger.info(f"Task created: {task.key} (id={task.id})")
        return task
    except Exception as e:
        logger.error(f"Failed to create task: {e}")
        raise
```

**Log Levels**:
- DEBUG: SQL queries (only in development)
- INFO: CRUD operations
- WARNING: Validation errors, constraint violations
- ERROR: Database errors, transaction rollbacks

### Performance Metrics

```python
import time

def timed_query(func):
    """Decorator to log query execution time"""
    def wrapper(*args, **kwargs):
        start = time.time()
        result = func(*args, **kwargs)
        duration = (time.time() - start) * 1000  # ms
        logger.info(f"{func.__name__} completed in {duration:.2f}ms")
        return result
    return wrapper

@timed_query
def filter_tasks(self, status: str):
    ...
```

---

## Summary

This architecture defines:

1. **4 Layers**: Database → ORM → Session Management → Repositories
2. **Repository Pattern**: Isolation of data access logic
3. **Session Management**: Centralized transaction handling via context managers
4. **Dependency Injection**: Repositories injected into CLI commands
5. **Error Translation**: SQLAlchemy exceptions → domain exceptions
6. **Type Safety**: Full type hints enable IDE support and validation
7. **Migration Support**: Alembic for versioned schema evolution
8. **Performance**: Indexed queries, batch operations, cached calculations
9. **Security**: Parameterized queries, file permissions, no sensitive logging
10. **Testability**: In-memory testing, transaction rollback, mock repositories

**Key Architectural Decisions**:
- SQLAlchemy 2.0+ for modern ORM features
- Repository pattern for testability and maintainability
- Context managers for automatic transaction management
- Type hints throughout for IDE support and mypy validation
- Alembic for database migrations

---

**Architecture Complete**: 2025-12-14
**Next Document**: 04-backend-design.md (detailed implementation specifications)
