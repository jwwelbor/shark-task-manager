# Testing Guide - Shark Task Manager

This guide explains how to test the database foundation we've implemented.

## Quick Start

### 1. **Run the Interactive Demo** ⭐ RECOMMENDED

The demo creates sample data and shows all features in action:

```bash
make demo
```

This will:
- Create an epic, feature, and 5 tasks
- Simulate task progress (mark some as completed)
- Calculate feature and epic progress
- Test various query filters
- Display a nice summary with emoji status icons

**Output Example:**
```
📊 Shark Task Manager - Database Demo
=====================================

1️⃣  Creating Epic...
   ✓ Created Epic: E04 - Task Management CLI - Core Functionality

2️⃣  Creating Feature...
   ✓ Created Feature: E04-F01 - Database Schema & Core Data Model

3️⃣  Creating Tasks...
   ✓ Created: T-E04-F01-001 - Create ORM Models
   ✓ Created: T-E04-F01-002 - Implement Validation
   ...

5️⃣  Current State:
   Epic Progress: 60.0%
   Feature: 60.0% complete

   Tasks:
     ✅ T-E04-F01-001 - Create ORM Models [completed]
     ✅ T-E04-F01-002 - Implement Validation [completed]
     ⭕ T-E04-F01-004 - Build Repository Layer [todo]
```

### 2. **Run Integration Tests**

Comprehensive tests that verify all CRUD operations:

```bash
make test-db
```

This tests:
- ✅ Epic CRUD operations
- ✅ Feature CRUD operations
- ✅ Task CRUD operations
- ✅ Task history tracking
- ✅ Atomic status updates (task + history in one transaction)
- ✅ Progress calculations
- ✅ Cascade deletes

**Expected Output:**
```
✓ Database initialized successfully
✓ Created epic with ID: 1
✓ Updated task status to in_progress
✓ Feature progress updated: 100.0%
✓ Cascade delete verified
✅ All tests passed!
```

### 3. **Start the Server**

Run the actual API server:

```bash
make run
```

The server will:
- Initialize the database at `shark-tasks.db`
- Run integrity checks
- Start HTTP server on port 8080

**Test endpoints:**
```bash
# Welcome message
curl http://localhost:8080/

# Health check (includes database ping)
curl http://localhost:8080/health
```

## Available Commands

```bash
make help          # Show all available commands
make clean         # Remove database and binaries (fresh start)
make demo          # Run interactive demo
make test-db       # Run integration tests
make build         # Build all binaries
make run           # Start the server
make fmt           # Format code
make vet           # Run static analysis
```

## What Gets Tested

### Database Schema
- ✅ 4 tables: epics, features, tasks, task_history
- ✅ 10 indexes for performance
- ✅ Foreign key constraints with CASCADE DELETE
- ✅ CHECK constraints for enums and ranges
- ✅ Auto-update triggers for timestamps

### Validation
- ✅ Epic keys: `E04` (pattern: `^E\d{2}$`)
- ✅ Feature keys: `E04-F01` (pattern: `^E\d{2}-F\d{2}$`)
- ✅ Task keys: `T-E04-F01-001` (pattern: `^T-E\d{2}-F\d{2}-\d{3}$`)
- ✅ Status enums (draft, active, completed, archived)
- ✅ Priority ranges (1-10 for tasks, high/medium/low for epics)
- ✅ Progress percentage (0.0-100.0)
- ✅ JSON validation for task dependencies

### Repository Operations
- ✅ Create: Epic, Feature, Task
- ✅ Read: By ID, by key, list all, list filtered
- ✅ Update: With validation
- ✅ Delete: With cascade verification
- ✅ Atomic status updates: Task status + history in one transaction
- ✅ Progress calculations: Feature and epic progress

### Query Filters
- ✅ Filter by status
- ✅ Filter by agent type
- ✅ Filter by epic (JOIN across tables)
- ✅ Combined filters (status + epic + agent + priority)

## Database File

After running the demo or server, you'll have:

```
shark-tasks.db       # Main database file
shark-tasks.db-shm   # Shared memory file (WAL mode)
shark-tasks.db-wal   # Write-ahead log (WAL mode)
```

**To reset everything:**
```bash
make clean
```

## Manual Testing

### Create Custom Test Data

You can modify `cmd/demo/main.go` to create your own test scenarios:

```go
// Create a custom epic
epic := &models.Epic{
    Key:      "E99",
    Title:    "My Custom Epic",
    Status:   models.EpicStatusActive,
    Priority: models.PriorityHigh,
}
epicRepo.Create(epic)

// Create a custom task
agentType := models.AgentTypeFrontend
task := &models.Task{
    FeatureID: feature.ID,
    Key:       "T-E99-F01-001",
    Title:     "My Custom Task",
    Status:    models.TaskStatusTodo,
    AgentType: &agentType,
    Priority:  1,
}
taskRepo.Create(task)
```

### Run Custom Test

```bash
make build
./bin/demo
```

## Testing Checklist

Before considering the database foundation complete, verify:

- [x] Database initializes without errors
- [x] All 4 tables created with correct schema
- [x] Foreign keys enabled (PRAGMA foreign_keys = ON)
- [x] WAL mode enabled (PRAGMA journal_mode = WAL)
- [x] Integrity check passes
- [x] Epic CRUD operations work
- [x] Feature CRUD operations work
- [x] Task CRUD operations work
- [x] Task history automatically created on status change
- [x] Atomic status updates (rollback on error)
- [x] Progress calculations accurate
- [x] Cascade deletes work (epic → features → tasks → history)
- [x] All validation rules enforced
- [x] Query filters work correctly
- [x] Indexes improve query performance

## Troubleshooting

### Database locked error
If you see "database is locked":
1. Stop any running servers: `Ctrl+C`
2. Clean and restart: `make clean && make demo`

### Foreign key violation
If you see foreign key errors:
- Verify foreign keys are enabled: Check logs for "foreign_keys not enabled"
- Make sure parent records exist before creating children

### Validation errors
All validation happens before database operations:
```
invalid epic key format: got "E1"
  → Must be 2 digits: E01

invalid task status: got "invalid"
  → Must be: todo, in_progress, blocked, ready_for_review, completed, archived

invalid priority: 15
  → Must be between 1 and 10
```

## Next Steps

After testing the database foundation:

1. **Build CLI** - Create commands to interact with the database
2. **Add API Endpoints** - REST API for task management
3. **Implement Task Lifecycle** - Workflow for task status transitions
4. **Add File Management** - Sync markdown task files with database
5. **Build Sync** - Import existing task files into database

## Performance Notes

The demo and tests use the same performance optimizations as production:

- **WAL mode** - Better concurrency for reads/writes
- **Indexes** - Fast lookups by key, status, agent type
- **Composite indexes** - Optimized for common queries (status + priority)
- **Connection pooling** - Reuse database connections
- **Prepared statements** - Query optimization

Expected performance (with 10K tasks):
- Single INSERT: < 50ms
- Query by key: < 10ms
- Filtered query: < 100ms
- Cascade delete: < 500ms (1000 tasks)
