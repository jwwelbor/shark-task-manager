# Interface Contracts: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: feature-architect (coordinator)

## Purpose

This document defines the contracts between the database layer and all consuming layers (CLI, task operations, folder management, etc.). All architects and implementers MUST align their designs with these contracts to ensure consistent data access patterns and type safety across the application.

---

## Data Transfer Objects (DTOs)

### EpicData

**Purpose**: Represents an epic for external consumers (CLI, reports)

| Field | Type | Required | Nullable | Description |
|-------|------|----------|----------|-------------|
| id | int | No (auto) | No | Database primary key |
| key | str | Yes | No | Epic identifier (format: `E##`) |
| title | str | Yes | No | Epic title |
| description | str | No | Yes | Epic description (markdown) |
| status | str | Yes | No | One of: `draft`, `active`, `completed`, `archived` |
| priority | str | Yes | No | One of: `high`, `medium`, `low` |
| business_value | str | No | Yes | One of: `high`, `medium`, `low`, or null |
| created_at | datetime | No (auto) | No | UTC timestamp |
| updated_at | datetime | No (auto) | No | UTC timestamp |
| progress_pct | float | No (computed) | No | Calculated from features (0.0-100.0) |

**Serialization Format** (for JSON output):
```python
{
    "id": 1,
    "key": "E04",
    "title": "Task Management CLI - Core Functionality",
    "description": "...",
    "status": "active",
    "priority": "high",
    "business_value": "high",
    "created_at": "2025-12-14T10:30:00Z",
    "updated_at": "2025-12-14T15:45:00Z",
    "progress_pct": 42.5
}
```

### FeatureData

**Purpose**: Represents a feature for external consumers

| Field | Type | Required | Nullable | Description |
|-------|------|----------|----------|-------------|
| id | int | No (auto) | No | Database primary key |
| epic_id | int | Yes | No | Foreign key to epics.id |
| key | str | Yes | No | Feature identifier (format: `E##-F##`) |
| title | str | Yes | No | Feature title |
| description | str | No | Yes | Feature description (markdown) |
| status | str | Yes | No | One of: `draft`, `active`, `completed`, `archived` |
| progress_pct | float | No (computed) | No | Calculated from tasks (0.0-100.0) |
| created_at | datetime | No (auto) | No | UTC timestamp |
| updated_at | datetime | No (auto) | No | UTC timestamp |
| epic_key | str | No (computed) | No | Denormalized epic key for convenience |

**Serialization Format**:
```python
{
    "id": 1,
    "epic_id": 1,
    "key": "E04-F01",
    "title": "Database Schema & Core Data Model",
    "description": "...",
    "status": "active",
    "progress_pct": 15.0,
    "created_at": "2025-12-14T10:30:00Z",
    "updated_at": "2025-12-14T15:45:00Z",
    "epic_key": "E04"
}
```

### TaskData

**Purpose**: Represents a task for external consumers

| Field | Type | Required | Nullable | Description |
|-------|------|----------|----------|-------------|
| id | int | No (auto) | No | Database primary key |
| feature_id | int | Yes | No | Foreign key to features.id |
| key | str | Yes | No | Task identifier (format: `T-E##-F##-###`) |
| title | str | Yes | No | Task title |
| description | str | No | Yes | Task description (markdown) |
| status | str | Yes | No | One of: `todo`, `in_progress`, `blocked`, `ready_for_review`, `completed`, `archived` |
| agent_type | str | No | Yes | One of: `frontend`, `backend`, `api`, `testing`, `devops`, `general`, or null |
| priority | int | No (default=5) | No | 1 (highest) to 10 (lowest) |
| depends_on | list[str] | No | Yes | List of task keys (e.g., `["T-E01-F01-001"]`) |
| assigned_agent | str | No | Yes | Free text agent identifier |
| file_path | str | No | Yes | Absolute path to task markdown file |
| blocked_reason | str | No | Yes | Only populated when status=blocked |
| created_at | datetime | No (auto) | No | UTC timestamp |
| started_at | datetime | No | Yes | When status changed to in_progress |
| completed_at | datetime | No | Yes | When status changed to completed |
| blocked_at | datetime | No | Yes | When status changed to blocked |
| updated_at | datetime | No (auto) | No | UTC timestamp |
| feature_key | str | No (computed) | No | Denormalized feature key |
| epic_key | str | No (computed) | No | Denormalized epic key |

**Serialization Format**:
```python
{
    "id": 1,
    "feature_id": 1,
    "key": "T-E04-F01-001",
    "title": "Create SQLAlchemy models",
    "description": "...",
    "status": "in_progress",
    "agent_type": "backend",
    "priority": 3,
    "depends_on": [],
    "assigned_agent": "claude-backend-specialist",
    "file_path": "/path/to/tasks/active/T-E04-F01-001.md",
    "blocked_reason": null,
    "created_at": "2025-12-14T10:30:00Z",
    "started_at": "2025-12-14T11:00:00Z",
    "completed_at": null,
    "blocked_at": null,
    "updated_at": "2025-12-14T11:00:00Z",
    "feature_key": "E04-F01",
    "epic_key": "E04"
}
```

**Note on `depends_on`**:
- Stored in database as JSON string: `'["T-E01-F01-001", "T-E01-F02-003"]'`
- Deserialized to Python list for DTO: `["T-E01-F01-001", "T-E01-F02-003"]`
- Empty dependencies: `[]` (not null)

### TaskHistoryData

**Purpose**: Represents a task status change record

| Field | Type | Required | Nullable | Description |
|-------|------|----------|----------|-------------|
| id | int | No (auto) | No | Database primary key |
| task_id | int | Yes | No | Foreign key to tasks.id |
| old_status | str | No | Yes | Previous status (null for task creation) |
| new_status | str | Yes | No | New status |
| agent | str | No | Yes | Who made the change (user, agent name, etc.) |
| notes | str | No | Yes | Optional change notes |
| timestamp | datetime | No (auto) | No | UTC timestamp |
| task_key | str | No (computed) | No | Denormalized task key |

**Serialization Format**:
```python
{
    "id": 1,
    "task_id": 1,
    "old_status": "todo",
    "new_status": "in_progress",
    "agent": "claude-backend-specialist",
    "notes": "Started implementation",
    "timestamp": "2025-12-14T11:00:00Z",
    "task_key": "T-E04-F01-001"
}
```

---

## Repository Interface Contracts

All repositories follow the Repository Pattern (not Active Record). Repositories manage persistence, models represent domain entities.

### EpicRepository

**Purpose**: CRUD operations and queries for epics

```python
from typing import Protocol
from datetime import datetime

class EpicRepository(Protocol):
    """Epic data access interface"""

    def create(self, key: str, title: str, status: str, priority: str,
               description: str | None = None,
               business_value: str | None = None) -> EpicData:
        """
        Create a new epic.

        Args:
            key: Epic identifier (format: E##)
            title: Epic title
            status: One of: draft, active, completed, archived
            priority: One of: high, medium, low
            description: Optional markdown description
            business_value: Optional: high, medium, low

        Returns:
            Created epic with auto-generated id and timestamps

        Raises:
            ValidationError: If key format invalid or status/priority invalid
            IntegrityError: If key already exists
        """
        ...

    def get_by_id(self, epic_id: int) -> EpicData | None:
        """Get epic by primary key. Returns None if not found."""
        ...

    def get_by_key(self, key: str) -> EpicData | None:
        """Get epic by key (e.g., 'E04'). Returns None if not found."""
        ...

    def list_all(self, status: str | None = None) -> list[EpicData]:
        """
        List all epics, optionally filtered by status.

        Args:
            status: Optional status filter

        Returns:
            List of epics ordered by created_at DESC
        """
        ...

    def update(self, epic_id: int, **fields) -> EpicData:
        """
        Update epic fields.

        Args:
            epic_id: Epic to update
            **fields: Fields to update (title, description, status, priority, business_value)

        Returns:
            Updated epic

        Raises:
            ValidationError: If field values invalid
            IntegrityError: If key uniqueness violated
        """
        ...

    def delete(self, epic_id: int) -> None:
        """
        Delete epic and cascade delete all features, tasks, task_history.

        Args:
            epic_id: Epic to delete

        Raises:
            IntegrityError: If epic doesn't exist
        """
        ...

    def calculate_progress(self, epic_id: int) -> float:
        """
        Calculate epic progress as weighted average of feature progress.

        Args:
            epic_id: Epic to calculate progress for

        Returns:
            Progress percentage (0.0-100.0)
            Returns 0.0 if epic has no features
        """
        ...
```

### FeatureRepository

**Purpose**: CRUD operations and queries for features

```python
class FeatureRepository(Protocol):
    """Feature data access interface"""

    def create(self, epic_id: int, key: str, title: str, status: str,
               description: str | None = None) -> FeatureData:
        """
        Create a new feature.

        Args:
            epic_id: Parent epic ID
            key: Feature identifier (format: E##-F##)
            title: Feature title
            status: One of: draft, active, completed, archived
            description: Optional markdown description

        Returns:
            Created feature with auto-generated id and timestamps

        Raises:
            ValidationError: If key format invalid or status invalid
            IntegrityError: If key already exists or epic_id doesn't exist
        """
        ...

    def get_by_id(self, feature_id: int) -> FeatureData | None:
        """Get feature by primary key. Returns None if not found."""
        ...

    def get_by_key(self, key: str) -> FeatureData | None:
        """Get feature by key (e.g., 'E04-F01'). Returns None if not found."""
        ...

    def list_by_epic(self, epic_id: int, status: str | None = None) -> list[FeatureData]:
        """
        List features for an epic, optionally filtered by status.

        Args:
            epic_id: Parent epic ID
            status: Optional status filter

        Returns:
            List of features ordered by key ASC
        """
        ...

    def list_all(self, status: str | None = None) -> list[FeatureData]:
        """List all features, optionally filtered by status."""
        ...

    def update(self, feature_id: int, **fields) -> FeatureData:
        """
        Update feature fields.

        Args:
            feature_id: Feature to update
            **fields: Fields to update (title, description, status)

        Returns:
            Updated feature

        Raises:
            ValidationError: If field values invalid
        """
        ...

    def delete(self, feature_id: int) -> None:
        """
        Delete feature and cascade delete all tasks, task_history.

        Args:
            feature_id: Feature to delete

        Raises:
            IntegrityError: If feature doesn't exist
        """
        ...

    def calculate_progress(self, feature_id: int) -> float:
        """
        Calculate feature progress as (completed tasks / total tasks × 100).

        Args:
            feature_id: Feature to calculate progress for

        Returns:
            Progress percentage (0.0-100.0)
            Returns 0.0 if feature has no tasks
        """
        ...

    def update_progress(self, feature_id: int) -> float:
        """
        Recalculate and update the cached progress_pct field.

        Args:
            feature_id: Feature to update

        Returns:
            New progress percentage (0.0-100.0)
        """
        ...
```

### TaskRepository

**Purpose**: CRUD operations and complex queries for tasks

```python
class TaskRepository(Protocol):
    """Task data access interface"""

    def create(self, feature_id: int, key: str, title: str, status: str,
               description: str | None = None,
               agent_type: str | None = None,
               priority: int = 5,
               depends_on: list[str] | None = None,
               assigned_agent: str | None = None,
               file_path: str | None = None) -> TaskData:
        """
        Create a new task.

        Args:
            feature_id: Parent feature ID
            key: Task identifier (format: T-E##-F##-###)
            title: Task title
            status: One of: todo, in_progress, blocked, ready_for_review, completed, archived
            description: Optional markdown description
            agent_type: Optional: frontend, backend, api, testing, devops, general
            priority: 1 (highest) to 10 (lowest), default 5
            depends_on: Optional list of task keys
            assigned_agent: Optional agent identifier
            file_path: Optional path to task markdown file

        Returns:
            Created task with auto-generated id and timestamps

        Raises:
            ValidationError: If key format invalid, status invalid, priority out of range
            IntegrityError: If key already exists or feature_id doesn't exist
        """
        ...

    def get_by_id(self, task_id: int) -> TaskData | None:
        """Get task by primary key. Returns None if not found."""
        ...

    def get_by_key(self, key: str) -> TaskData | None:
        """Get task by key (e.g., 'T-E04-F01-001'). Returns None if not found."""
        ...

    def list_all(self) -> list[TaskData]:
        """List all tasks ordered by created_at DESC."""
        ...

    def list_by_feature(self, feature_id: int) -> list[TaskData]:
        """List all tasks for a feature ordered by key ASC."""
        ...

    def list_by_epic(self, epic_id: int) -> list[TaskData]:
        """List all tasks for an epic (across all features) ordered by key ASC."""
        ...

    def filter_by_status(self, status: str) -> list[TaskData]:
        """Filter tasks by status."""
        ...

    def filter_by_agent_type(self, agent_type: str) -> list[TaskData]:
        """Filter tasks by agent_type."""
        ...

    def filter_by_priority(self, min_priority: int, max_priority: int) -> list[TaskData]:
        """
        Filter tasks by priority range.

        Args:
            min_priority: Minimum priority (inclusive)
            max_priority: Maximum priority (inclusive)

        Returns:
            Tasks with priority in range [min_priority, max_priority]
        """
        ...

    def filter_combined(self,
                       status: str | None = None,
                       epic_key: str | None = None,
                       feature_key: str | None = None,
                       agent_type: str | None = None,
                       min_priority: int | None = None,
                       max_priority: int | None = None) -> list[TaskData]:
        """
        Filter tasks by multiple criteria (AND logic).

        All parameters are optional. Only non-None parameters are used for filtering.

        Returns:
            Filtered tasks ordered by priority ASC (highest priority first), then created_at DESC
        """
        ...

    def update(self, task_id: int, **fields) -> TaskData:
        """
        Update task fields.

        Args:
            task_id: Task to update
            **fields: Fields to update (any task field except id, feature_id, key, created_at)

        Returns:
            Updated task

        Raises:
            ValidationError: If field values invalid
        """
        ...

    def update_status(self, task_id: int, new_status: str,
                     agent: str | None = None,
                     notes: str | None = None) -> TaskData:
        """
        Update task status and record history entry atomically.

        This is a convenience method that:
        1. Updates task.status
        2. Updates relevant timestamp (started_at, completed_at, blocked_at)
        3. Creates task_history record
        All in a single transaction.

        Args:
            task_id: Task to update
            new_status: New status value
            agent: Who is making the change
            notes: Optional change notes

        Returns:
            Updated task

        Raises:
            ValidationError: If status invalid
        """
        ...

    def delete(self, task_id: int) -> None:
        """
        Delete task and cascade delete task_history.

        Args:
            task_id: Task to delete

        Raises:
            IntegrityError: If task doesn't exist
        """
        ...
```

### TaskHistoryRepository

**Purpose**: Access and query task history

```python
class TaskHistoryRepository(Protocol):
    """Task history data access interface"""

    def create(self, task_id: int, new_status: str,
               old_status: str | None = None,
               agent: str | None = None,
               notes: str | None = None) -> TaskHistoryData:
        """
        Create a task history entry.

        Args:
            task_id: Task that changed
            new_status: New status value
            old_status: Previous status (null for task creation)
            agent: Who made the change
            notes: Optional change notes

        Returns:
            Created history entry with auto-generated id and timestamp

        Raises:
            IntegrityError: If task_id doesn't exist
        """
        ...

    def list_by_task(self, task_id: int) -> list[TaskHistoryData]:
        """
        Get history for a task ordered by timestamp DESC (newest first).

        Args:
            task_id: Task to get history for

        Returns:
            List of history entries
        """
        ...

    def list_recent(self, limit: int = 50) -> list[TaskHistoryData]:
        """
        Get recent history across all tasks.

        Args:
            limit: Maximum number of entries to return

        Returns:
            Recent history ordered by timestamp DESC
        """
        ...
```

---

## Session Management Interface

### DatabaseSession

**Purpose**: Manage database connections and transactions

```python
from typing import Protocol, ContextManager

class DatabaseSession(Protocol):
    """Database session management interface"""

    def get_session(self) -> ContextManager:
        """
        Get a database session with automatic transaction management.

        Usage:
            with db.get_session() as session:
                task = task_repo.create(session, ...)

        Session is automatically committed on success, rolled back on exception.
        """
        ...

    def create_all_tables(self) -> None:
        """
        Create all database tables.

        This is used for initial setup and testing.
        Production should use Alembic migrations.
        """
        ...

    def check_integrity(self) -> bool:
        """
        Run SQLite PRAGMA integrity_check.

        Returns:
            True if database is intact, False if corrupted
        """
        ...

    def get_schema_version(self) -> str | None:
        """
        Get current Alembic schema version.

        Returns:
            Version string (e.g., "001") or None if not initialized
        """
        ...
```

---

## Validation Contracts

### Key Format Validation

All repositories MUST validate key formats before database operations.

**Epic Key Format**:
- Pattern: `^E\d{2}$`
- Examples: `E01`, `E04`, `E99`
- Invalid: `E1`, `E001`, `Epic01`

**Feature Key Format**:
- Pattern: `^E\d{2}-F\d{2}$`
- Examples: `E04-F01`, `E04-F12`, `E99-F99`
- Invalid: `E4-F1`, `E04-F1`, `F01`

**Task Key Format**:
- Pattern: `^T-E\d{2}-F\d{2}-\d{3}$`
- Examples: `T-E04-F01-001`, `T-E04-F01-042`, `T-E99-F99-999`
- Invalid: `E04-F01-001`, `T-E4-F1-1`, `T-E04-F01-1`

**Implementation**:
```python
import re

EPIC_KEY_PATTERN = re.compile(r'^E\d{2}$')
FEATURE_KEY_PATTERN = re.compile(r'^E\d{2}-F\d{2}$')
TASK_KEY_PATTERN = re.compile(r'^T-E\d{2}-F\d{2}-\d{3}$')

def validate_epic_key(key: str) -> None:
    if not EPIC_KEY_PATTERN.match(key):
        raise ValidationError(f"Invalid epic key format: {key}. Expected: E##")

def validate_feature_key(key: str) -> None:
    if not FEATURE_KEY_PATTERN.match(key):
        raise ValidationError(f"Invalid feature key format: {key}. Expected: E##-F##")

def validate_task_key(key: str) -> None:
    if not TASK_KEY_PATTERN.match(key):
        raise ValidationError(f"Invalid task key format: {key}. Expected: T-E##-F##-###")
```

### Enum Validation

**Epic Status**: `draft`, `active`, `completed`, `archived`

**Epic Priority**: `high`, `medium`, `low`

**Epic Business Value**: `high`, `medium`, `low`, `null`

**Feature Status**: `draft`, `active`, `completed`, `archived`

**Task Status**: `todo`, `in_progress`, `blocked`, `ready_for_review`, `completed`, `archived`

**Task Agent Type**: `frontend`, `backend`, `api`, `testing`, `devops`, `general`, `null`

**Task Priority**: Integer 1-10 (1 = highest, 10 = lowest)

**Implementation**:
```python
from enum import Enum

class EpicStatus(str, Enum):
    DRAFT = "draft"
    ACTIVE = "active"
    COMPLETED = "completed"
    ARCHIVED = "archived"

class TaskStatus(str, Enum):
    TODO = "todo"
    IN_PROGRESS = "in_progress"
    BLOCKED = "blocked"
    READY_FOR_REVIEW = "ready_for_review"
    COMPLETED = "completed"
    ARCHIVED = "archived"

def validate_task_priority(priority: int) -> None:
    if not (1 <= priority <= 10):
        raise ValidationError(f"Priority must be 1-10, got: {priority}")
```

---

## Error Contracts

### Exception Hierarchy

All database operations may raise these exceptions:

```python
class DatabaseError(Exception):
    """Base class for all database errors"""
    pass

class DatabaseNotFound(DatabaseError):
    """Database file doesn't exist"""
    pass

class SchemaVersionMismatch(DatabaseError):
    """Database schema version doesn't match application"""
    def __init__(self, current: str, expected: str):
        self.current = current
        self.expected = expected
        super().__init__(f"Schema version mismatch: {current} (current) vs {expected} (expected)")

class IntegrityError(DatabaseError):
    """Foreign key or unique constraint violation"""
    def __init__(self, message: str, constraint: str | None = None):
        self.constraint = constraint
        super().__init__(message)

class ValidationError(DatabaseError):
    """Data validation failed"""
    def __init__(self, message: str, field: str | None = None):
        self.field = field
        super().__init__(message)
```

### Error Messages

**IntegrityError examples**:
- Foreign key: `"Task references non-existent feature (feature_id=999)"`
- Unique constraint: `"Epic key 'E04' already exists"`
- Cascade info: `"Cannot delete epic E04: would cascade delete 7 features and 42 tasks"`

**ValidationError examples**:
- Key format: `"Invalid task key format: 'T-E4-F1-1'. Expected: T-E##-F##-###"`
- Enum: `"Invalid task status: 'invalid'. Expected one of: todo, in_progress, blocked, ready_for_review, completed, archived"`
- Range: `"Priority must be 1-10, got: 15"`

---

## Timestamp Contracts

### Timezone Handling

**Storage**: All timestamps stored in UTC

**Display**: Convert to local timezone for human-readable output

**Format**:
- Database storage: `datetime` object (timezone-aware UTC)
- JSON serialization: ISO 8601 with Z suffix (e.g., `"2025-12-14T15:45:00Z"`)
- Human display: Local timezone with explicit offset (e.g., `"2025-12-14 10:45:00 -05:00"`)

**Helper Methods**:
```python
from datetime import datetime, timezone

def utc_now() -> datetime:
    """Get current UTC time as timezone-aware datetime."""
    return datetime.now(timezone.utc)

def to_utc(dt: datetime) -> datetime:
    """Convert datetime to UTC timezone."""
    return dt.astimezone(timezone.utc)

def to_local(dt: datetime) -> datetime:
    """Convert UTC datetime to local timezone."""
    return dt.astimezone()

def format_iso8601(dt: datetime) -> str:
    """Format datetime as ISO 8601 string with Z suffix."""
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")

def format_local(dt: datetime) -> str:
    """Format datetime in local timezone for display."""
    local_dt = to_local(dt)
    return local_dt.strftime("%Y-%m-%d %H:%M:%S %z")
```

### Automatic Timestamp Updates

**created_at**: Set automatically on INSERT, never modified

**updated_at**: Set automatically on INSERT, updated automatically on every UPDATE

**started_at**: Set when task status changes to `in_progress`

**completed_at**: Set when task status changes to `completed`

**blocked_at**: Set when task status changes to `blocked`

**Transition handling**:
- If task moves from `in_progress` → `blocked`, keep `started_at`, set `blocked_at`
- If task moves from `blocked` → `in_progress`, keep both timestamps
- If task moves from `in_progress` → `completed`, keep `started_at`, set `completed_at`

---

## Transaction Contracts

### Atomicity Guarantees

**Single Operation Transactions**:
- Each repository method is atomic (one database operation)
- Automatically committed on success, rolled back on exception

**Multi-Operation Transactions**:
- Use `update_status()` for atomic status + history + timestamp updates
- External code can use session context manager for custom atomic operations

**Example Multi-Step Operation**:
```python
# Atomic: update task status + create history entry + update feature progress
with db.get_session() as session:
    # All operations in one transaction
    task = task_repo.update_status(session, task_id, "completed", agent="user")
    feature_repo.update_progress(session, task.feature_id)
    # Automatic commit or rollback
```

### Isolation Level

- Default SQLite isolation: SERIALIZABLE (strictest)
- Use `BEGIN IMMEDIATE` for write transactions to prevent lock escalation
- Read-only queries can use `BEGIN DEFERRED` (default)

---

## Performance Contracts

From PRD Non-Functional Requirements:

| Operation | Maximum Latency | Dataset Size |
|-----------|----------------|--------------|
| Database initialization | 500ms | N/A |
| Single task INSERT | 50ms | N/A |
| Task SELECT with filters | 100ms | 10,000 tasks |
| Progress calculation | 200ms | 50 features |
| Batch INSERT (100 tasks) | 2,000ms | N/A |
| get_by_key() | 10ms | N/A (indexed) |
| list_all() | 100ms | 10,000 tasks |

**Query Optimization Requirements**:
- All `get_by_key()` operations use UNIQUE index (automatic)
- Filter operations use indexes on `status`, `agent_type`, `priority`
- Progress calculations use optimized single-query aggregation
- Foreign key lookups use indexes on `epic_id`, `feature_id`

---

## Migration Contracts

### Alembic Integration

**Version Numbering**:
- Format: `001`, `002`, `003`, ... (3-digit zero-padded)
- Initial migration: `001_initial_schema.py`

**Migration Structure**:
```python
def upgrade():
    """Apply migration (forward)"""
    # Create/modify schema
    pass

def downgrade():
    """Reverse migration (backward)"""
    # Undo changes
    pass
```

**Schema Version Check**:
```python
# Application startup must verify schema version
current_version = db.get_schema_version()
expected_version = "001"  # From application config

if current_version != expected_version:
    raise SchemaVersionMismatch(current_version, expected_version)
```

---

## Naming Standards

### Python Code

**Modules**: `database/models.py`, `database/repositories.py`

**Classes**: `Epic`, `Feature`, `Task`, `TaskHistory`, `EpicRepository`

**Functions/Methods**: `create()`, `get_by_id()`, `filter_by_status()`

**Variables**: `epic_id`, `task_key`, `created_at`

### Database Schema

**Tables**: `epics`, `features`, `tasks`, `task_history`

**Columns**: `epic_id`, `created_at`, `progress_pct`

**Indexes**: `idx_tasks_status`, `idx_tasks_feature_id`

**Constraints**: `fk_tasks_feature_id`, `ck_tasks_status`

---

## Backward Compatibility

### Initial Release (v1.0.0)

This is the first release - no backward compatibility concerns.

### Future Schema Changes

When modifying schema:
1. Create new Alembic migration
2. Update ORM models
3. Update DTOs if field contracts change
4. Update repository interfaces if needed
5. Increment schema version
6. Document breaking changes in migration comments

**Breaking vs Non-Breaking Changes**:

**Non-Breaking** (safe):
- Adding nullable columns
- Adding new tables
- Adding indexes
- Relaxing constraints

**Breaking** (requires migration):
- Removing columns
- Changing column types
- Adding NOT NULL columns (without defaults)
- Removing tables
- Changing foreign key relationships

---

## Integration Points

### CLI Layer (E04-F02)

**Dependency Injection**:
```python
# CLI commands receive repositories via dependency injection
@click.command()
@click.pass_context
def task_list(ctx):
    task_repo: TaskRepository = ctx.obj['task_repo']
    tasks = task_repo.list_all()
    # Format and display
```

**JSON Output**:
```python
# DTOs serialize to JSON for --json flag
tasks = task_repo.filter_by_status("todo")
json_output = [task.to_dict() for task in tasks]
print(json.dumps(json_output, indent=2))
```

### Folder Management (E04-F05)

**Atomic File + DB Updates**:
```python
# Move file and update database atomically
with db.get_session() as session:
    # Move file
    new_path = move_task_file(old_path, new_status)

    # Update database
    task = task_repo.update_status(session, task_id, new_status)
    task_repo.update(session, task_id, file_path=new_path)

    # Both succeed or both fail
```

### Task Creation (E04-F06)

**Task + File Creation**:
```python
# Create database record first, then file
with db.get_session() as session:
    task = task_repo.create(session, ...)
    file_path = create_task_file(task.key, task.title)
    task_repo.update(session, task.id, file_path=file_path)
```

---

## Summary

This interface contract defines:

1. **4 DTOs**: EpicData, FeatureData, TaskData, TaskHistoryData
2. **4 Repositories**: EpicRepository, FeatureRepository, TaskRepository, TaskHistoryRepository
3. **Session Management**: DatabaseSession interface
4. **Validation**: Key formats, enums, ranges
5. **Error Handling**: Custom exception hierarchy
6. **Timestamps**: UTC storage, local display, automatic updates
7. **Transactions**: Atomicity guarantees for multi-step operations
8. **Performance**: Latency targets for all operations
9. **Migrations**: Alembic integration and version checking
10. **Integration**: Patterns for CLI, folder management, task creation

All implementing code MUST adhere to these contracts.

---

**Contracts Defined**: 2025-12-14
**Next Document**: 03-data-design.md (database admin creates table schemas)
