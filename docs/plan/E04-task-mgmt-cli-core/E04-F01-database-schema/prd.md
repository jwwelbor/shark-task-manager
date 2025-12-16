# Feature: Database Schema & Core Data Model

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

The PM CLI tool requires a robust, performant, and reliable data storage layer to serve as the single source of truth for all project state (epics, features, tasks). Without a properly designed database schema, the system cannot maintain referential integrity, calculate progress metrics efficiently, track status transitions, or support the complex queries needed by agents and developers. The schema must balance normalization (to prevent data duplication and inconsistencies) with query performance (agents need <100ms response times even with 10,000 tasks). It must also support the full task lifecycle with proper constraints, enable atomic transactions for status changes, and provide a foundation for audit trails.

### Solution

Design and implement a normalized SQLite database schema (project.db) with four core tables: epics, features, tasks, and task_history. The schema uses foreign key constraints to enforce the three-level hierarchy (epic → feature → task), enum-based status fields with CHECK constraints for validation, and automatic timestamp management via triggers. The tasks table includes a JSON field for dependency arrays, enabling flexible dependency tracking without additional join tables. The schema supports all CRUD operations through a Python data access layer with type-safe models, parameterized queries to prevent SQL injection, and transaction management for atomic operations.

### Impact

- **Data Integrity**: 100% referential integrity through foreign key constraints, preventing orphaned tasks or invalid epic/feature references
- **Query Performance**: <100ms response times for all standard queries (task listing, filtering, progress calculation) on datasets up to 10,000 tasks
- **Schema Evolution**: Database migration system enables schema changes without data loss as requirements evolve
- **Type Safety**: SQLAlchemy ORM models provide compile-time type checking and IDE autocomplete for all database operations
- **Foundation for Features**: Provides the data layer needed by all other E04 features (CLI, task operations, folder management)

## User Personas

### Primary Persona: Backend Developer (implementing other E04 features)

**Role**: Developer building the CLI, task operations, and other features on top of this data layer
**Environment**: Python 3.10+, SQLite, local development

**Key Characteristics**:
- Needs clear, type-safe APIs for database operations
- Requires reliable transactions for atomic file + database updates
- Must handle database errors gracefully
- Benefits from comprehensive documentation and examples

**Goals**:
- Query tasks efficiently with complex filters (status, epic, agent, priority)
- Perform atomic updates (status change + timestamp + history record)
- Calculate progress percentages across features and epics
- Validate data before insertion (status enums, key formats, dependency references)

**Pain Points this Feature Addresses**:
- Need for explicit SQL queries in every feature (ORM provides abstraction)
- Risk of SQL injection without parameterized queries
- Manual timestamp management
- Complex foreign key validation logic

### Secondary Persona: AI Agent (indirect user through CLI)

**Role**: Queries database through CLI commands for task discovery and status updates
**Environment**: Claude Code CLI, relies on fast query responses

**Key Characteristics**:
- Needs predictable query performance (<100ms)
- Requires consistent data (no stale or contradictory status)
- Depends on transaction atomicity for reliable state

**Goals**:
- Get next available task in <50ms
- Update task status reliably
- Trust that status is always consistent

**Pain Points this Feature Addresses**:
- Slow queries waste tokens and time
- Data inconsistencies cause agent confusion
- Failed updates without rollback leave database in bad state

## User Stories

### Must-Have User Stories

**Story 1: Create Database Schema**
- As a backend developer, I want to initialize the database schema with all tables and constraints, so that the application has a reliable data store from first run.

**Story 2: Enforce Referential Integrity**
- As a backend developer, I want foreign key constraints to prevent orphaned tasks, so that every task always belongs to a valid feature and epic.

**Story 3: Query Tasks with Filters**
- As a backend developer, I want to query tasks by status, epic, feature, agent type, and priority, so that I can implement CLI commands like `pm task list --status=todo --agent=frontend`.

**Story 4: Calculate Progress Automatically**
- As a backend developer, I want feature progress to be calculated automatically as (completed tasks / total tasks × 100), so that the CLI can display accurate progress without manual updates.

**Story 5: Record Status History**
- As a backend developer, I want every status change to be recorded in task_history with timestamp and agent, so that E05 can display audit trails.

**Story 6: Validate Task Keys**
- As a backend developer, I want task keys to be validated against the format `T-E##-F##-###`, so that invalid keys are rejected before database insertion.

**Story 7: Handle Dependencies as JSON**
- As a backend developer, I want task dependencies stored as JSON arrays, so that tasks can reference multiple prerequisite tasks without complex join tables.

**Story 8: Atomic Transactions**
- As a backend developer, I want to wrap multi-step operations (update task + insert history + update feature progress) in transactions, so that failures don't leave partial updates.

### Should-Have User Stories

**Story 9: Database Migration System**
- As a backend developer, I want a migration framework (Alembic) to version the schema, so that I can evolve the database structure without losing data.

**Story 10: Database Backup**
- As a developer or user, I want to create timestamped backups of project.db, so that I can recover from accidental data corruption.

**Story 11: Validate Dependency References**
- As a backend developer, I want to validate that task dependencies reference existing task IDs, so that broken dependency chains are prevented at write time.

### Could-Have User Stories

**Story 12: Read Replicas for Queries**
- As a system with high query load, I want read-only database connections for queries, so that writes don't block reads.

**Story 13: Database Optimization Hints**
- As a backend developer, I want indexes on frequently queried columns (status, epic_id, feature_id, agent_type), so that filters remain fast as data grows.

## Requirements

### Functional Requirements

**Database Tables:**

1. The system must create an `epics` table with columns:
   - `id` (INTEGER PRIMARY KEY AUTOINCREMENT)
   - `key` (TEXT NOT NULL UNIQUE) - Format: E##
   - `title` (TEXT NOT NULL)
   - `description` (TEXT)
   - `status` (TEXT NOT NULL) - ENUM: draft, active, completed, archived
   - `priority` (TEXT NOT NULL) - ENUM: high, medium, low
   - `business_value` (TEXT) - ENUM: high, medium, low
   - `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)
   - `updated_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

2. The system must create a `features` table with columns:
   - `id` (INTEGER PRIMARY KEY AUTOINCREMENT)
   - `epic_id` (INTEGER NOT NULL, FOREIGN KEY → epics.id ON DELETE CASCADE)
   - `key` (TEXT NOT NULL UNIQUE) - Format: E##-F##
   - `title` (TEXT NOT NULL)
   - `description` (TEXT)
   - `status` (TEXT NOT NULL) - ENUM: draft, active, completed, archived
   - `progress_pct` (REAL DEFAULT 0.0) - Calculated field
   - `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)
   - `updated_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

3. The system must create a `tasks` table with columns:
   - `id` (INTEGER PRIMARY KEY AUTOINCREMENT)
   - `feature_id` (INTEGER NOT NULL, FOREIGN KEY → features.id ON DELETE CASCADE)
   - `key` (TEXT NOT NULL UNIQUE) - Format: T-E##-F##-###
   - `title` (TEXT NOT NULL)
   - `description` (TEXT)
   - `status` (TEXT NOT NULL) - ENUM: todo, in_progress, blocked, ready_for_review, completed, archived
   - `agent_type` (TEXT) - ENUM: frontend, backend, api, testing, devops, general
   - `priority` (INTEGER DEFAULT 5) - 1 (highest) to 10 (lowest)
   - `depends_on` (TEXT) - JSON array of task IDs, e.g., `["T-E01-F01-001", "T-E01-F02-003"]`
   - `assigned_agent` (TEXT) - Free text agent identifier
   - `file_path` (TEXT) - Absolute path to task markdown file
   - `blocked_reason` (TEXT) - Only populated when status=blocked
   - `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)
   - `started_at` (TIMESTAMP NULL)
   - `completed_at` (TIMESTAMP NULL)
   - `blocked_at` (TIMESTAMP NULL)
   - `updated_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

4. The system must create a `task_history` table with columns:
   - `id` (INTEGER PRIMARY KEY AUTOINCREMENT)
   - `task_id` (INTEGER NOT NULL, FOREIGN KEY → tasks.id ON DELETE CASCADE)
   - `old_status` (TEXT)
   - `new_status` (TEXT NOT NULL)
   - `agent` (TEXT) - Who made the change
   - `notes` (TEXT) - Optional change notes
   - `timestamp` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

**Constraints and Validation:**

5. The system must enforce CHECK constraints on all ENUM fields to reject invalid values (e.g., `CHECK (status IN ('draft', 'active', 'completed', 'archived'))`)

6. The system must create a UNIQUE index on epic.key, feature.key, and task.key to prevent duplicate keys

7. The system must enforce NOT NULL constraints on all required fields (title, status, priority, etc.)

8. The system must validate task key format using regex pattern `^T-E\d{2}-F\d{2}-\d{3}$` before insertion

9. The system must validate epic key format using regex pattern `^E\d{2}$` before insertion

10. The system must validate feature key format using regex pattern `^E\d{2}-F\d{2}$` before insertion

**Data Access Layer:**

11. The system must provide a SQLAlchemy ORM model for each table (Epic, Feature, Task, TaskHistory)

12. All ORM models must include Python type hints for all fields

13. The system must provide a database connection factory that returns session objects with transaction support

14. All database writes must use parameterized queries to prevent SQL injection

15. The system must provide CRUD methods for each model: create(), get_by_id(), get_by_key(), update(), delete(), list_all()

16. The system must provide filter methods for tasks: filter_by_status(), filter_by_epic(), filter_by_feature(), filter_by_agent(), filter_by_priority()

17. The system must provide a method to calculate feature progress: `calculate_feature_progress(feature_id)` returning percentage (0.0-100.0)

18. The system must provide a method to calculate epic progress: `calculate_epic_progress(epic_id)` returning weighted average of feature progress

**Transactions and Atomicity:**

19. The system must support transaction contexts for multi-step operations (e.g., update task + insert history record)

20. Failed transactions must automatically rollback all changes

21. The system must provide a `with_transaction()` context manager for explicit transaction control

**Timestamps:**

22. The system must automatically set `created_at` to current UTC timestamp on INSERT

23. The system must automatically update `updated_at` to current UTC timestamp on every UPDATE

24. The system must use UTC timezone for all timestamps

25. The system must provide helper methods to convert timestamps to local timezone for display

**Migrations:**

26. The system must use Alembic for database migrations

27. The initial migration must create all four tables with constraints

28. The system must track the current schema version in the database

29. Migrations must be reversible (up/down) to support rollback

**Error Handling:**

30. The system must raise specific exceptions for: DatabaseNotFound, SchemaVersionMismatch, IntegrityError, ValidationError

31. Foreign key violations must raise `IntegrityError` with clear message indicating parent/child relationship

32. Unique constraint violations must raise `IntegrityError` with clear message indicating which key is duplicated

### Non-Functional Requirements

**Performance:**

- Database initialization (schema creation) must complete in <500ms
- Task INSERT operations must complete in <50ms
- Task SELECT queries with filters must return in <100ms for datasets up to 10,000 tasks
- Progress calculation queries must complete in <200ms for epics with 50 features
- Batch INSERT operations (100 tasks) must complete in <2 seconds

**Security:**

- All queries must use parameterized statements (no string concatenation)
- Database file permissions must be 600 (read/write owner only) on Unix systems
- No sensitive data logging (passwords, tokens) in SQL logs

**Reliability:**

- Foreign key constraints must be enabled (PRAGMA foreign_keys = ON)
- Database must use WAL mode (Write-Ahead Logging) for better concurrency
- Transactions must use IMMEDIATE isolation level to prevent lock escalation
- Database corruption must be detectable via integrity_check pragma

**Data Integrity:**

- CASCADE deletes must propagate: deleting epic deletes all features and tasks
- Orphaned records must be impossible (enforced by foreign keys)
- Status transitions must be validated before UPDATE
- JSON dependency arrays must be valid JSON (validated before INSERT)

**Maintainability:**

- All ORM models must have docstrings explaining fields and relationships
- Complex queries must have inline comments explaining JOINs and filters
- Schema must be documented in a separate schema.md file with ER diagram (ASCII art)
- Migration files must include descriptive comments

**Compatibility:**

- SQLite version 3.35+ required (for RETURNING clause and JSON functions)
- Python 3.10+ required (for type hints and match/case syntax)
- Cross-platform path handling (use pathlib for file_path field)
- Database file must be portable across operating systems

## Acceptance Criteria

### Database Schema Creation

**Given** the PM CLI is run for the first time in a new project
**When** the database initialization code executes
**Then** the project.db file is created with all four tables (epics, features, tasks, task_history)
**And** all foreign key constraints are enabled
**And** all CHECK constraints are present on ENUM fields
**And** all UNIQUE indexes are created on key columns

### Referential Integrity Enforcement

**Given** a feature exists with id=5
**When** I attempt to delete the parent epic
**Then** the feature is automatically deleted (CASCADE)
**And** all tasks belonging to that feature are also deleted

**Given** I attempt to insert a task with `feature_id=999` (non-existent)
**When** the INSERT executes
**Then** an IntegrityError is raised
**And** the transaction is rolled back
**And** no task record is created

### Task Key Validation

**Given** I attempt to create a task with key "INVALID-KEY"
**When** the create_task() method is called
**Then** a ValidationError is raised with message "Invalid task key format"
**And** no database record is created

**Given** I create a task with valid key "T-E01-F02-003"
**When** the create_task() method is called
**Then** the task is inserted successfully
**And** the key is stored exactly as provided

### Status Enum Validation

**Given** I attempt to set task status to "invalid_status"
**When** the UPDATE executes
**Then** a CHECK constraint violation occurs
**And** the database rejects the update
**And** the task status remains unchanged

**Given** I set task status to "in_progress" (valid)
**When** the UPDATE executes
**Then** the status is updated successfully
**And** updated_at timestamp is automatically updated

### Timestamp Management

**Given** I create a new task
**When** the INSERT executes
**Then** created_at is automatically set to current UTC time
**And** updated_at is automatically set to current UTC time

**Given** an existing task with created_at="2025-01-01 10:00:00"
**When** I update the task title
**Then** updated_at is automatically updated to current UTC time
**And** created_at remains unchanged

### Progress Calculation

**Given** a feature has 10 tasks: 7 completed, 2 in_progress, 1 todo
**When** I call calculate_feature_progress(feature_id)
**Then** the result is 70.0 (7/10 × 100)

**Given** a feature has 0 tasks
**When** I call calculate_feature_progress(feature_id)
**Then** the result is 0.0 (not an error)

**Given** an epic has 3 features with progress [50%, 75%, 100%]
**When** I call calculate_epic_progress(epic_id)
**Then** the result is 75.0 (average of feature progress)

### Task Filtering

**Given** the database has 100 tasks across 5 epics
**When** I call filter_by_status("todo") AND filter_by_epic("E01")
**Then** only tasks matching both conditions are returned
**And** the query completes in <100ms

**Given** I filter by agent_type="frontend" and priority <= 3
**When** the query executes
**Then** only high-priority frontend tasks are returned
**And** results are returned as ORM model instances (not raw dicts)

### Transaction Rollback

**Given** I start a transaction to update task status and insert history record
**When** the history INSERT fails (e.g., constraint violation)
**Then** the entire transaction is rolled back
**And** the task status remains unchanged
**And** no history record is created

### History Recording

**Given** I update a task status from "todo" to "in_progress"
**When** the update completes
**Then** a task_history record is created with old_status="todo", new_status="in_progress"
**And** the history timestamp matches the task.updated_at timestamp

### Database Migration

**Given** I have an existing database at schema version 1
**When** I run `alembic upgrade head` to apply migration 2
**Then** the schema is updated without data loss
**And** the version number in the database is updated to 2

## Out of Scope

### Explicitly NOT Included in This Feature

1. **CLI Commands** - This feature provides the data layer only. CLI command implementation is in E04-F02 (CLI Infrastructure) and E04-F03 (Task Lifecycle Operations).

2. **File Operations** - Creating, moving, or deleting task markdown files is handled by E04-F05 (Folder Management). This feature only stores file_path as metadata.

3. **Dependency Validation** - While the schema stores dependencies as JSON, validating that referenced task IDs exist is deferred to E04-F03 (Task Lifecycle Operations) or E05-F02 (Dependency Management).

4. **Advanced Queries** - Full-text search, complex aggregations, and saved filters are in E05-F01 (Status Dashboard) and optional E05 features.

5. **Data Export** - Exporting tasks to CSV/JSON is in E05-F03 (History & Audit Trail).

6. **Multi-Database Support** - Only SQLite is supported. PostgreSQL or MySQL support is out of scope.

7. **Database Sharding** - Single database file for all projects. Multi-database or sharding is out of scope.

8. **Real-time Sync** - No database triggers or event listeners for real-time updates. Polling-based refresh only.

9. **Custom Fields** - Schema supports only the defined columns. User-defined custom fields are out of scope.

10. **Database Encryption** - SQLite file is not encrypted. Database-level encryption (SQLCipher) is out of scope.
