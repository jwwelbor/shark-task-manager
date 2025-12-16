# Task Index: Folder Management & File Operations (E04-F05)

## Overview

This index tracks all implementation tasks for the Folder Management & File Operations feature (E04-F05).

**Feature Location**: `/home/jwwelbor/projects/ai-dev-team/docs/plan/E04-task-mgmt-cli-core/E04-F05-folder-management/`
**Total Tasks**: 6
**Estimated Total Time**: 34 hours (4-5 business days)

## Task List

| Task Key | Title | Status | Assigned Agent | Dependencies | Est. Time | Current Location |
|----------|-------|--------|----------------|--------------|-----------|------------------|
| T-E04-F05-001 | Core Folder Structure & Path Management | todo | general-purpose | None | 4 hours | docs/tasks/todo/ |
| T-E04-F05-002 | Atomic File Operations with Error Handling | todo | general-purpose | T-E04-F05-001 | 6 hours | docs/tasks/todo/ |
| T-E04-F05-003 | Validation & Repair System | todo | general-purpose | T-E04-F05-002 | 8 hours | docs/tasks/todo/ |
| T-E04-F05-004 | Database-File Synchronization with Rollback | todo | general-purpose | T-E04-F05-002 | 6 hours | docs/tasks/todo/ |
| T-E04-F05-005 | CLI Commands for Validation & Repair | todo | general-purpose | T-E04-F05-003, T-E04-F05-004 | 6 hours | docs/tasks/todo/ |
| T-E04-F05-006 | Integration with Task Operations | todo | general-purpose | T-E04-F05-004 | 4 hours | docs/tasks/todo/ |

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

### Phase 1: Core Infrastructure (4 hours)

**Task**: T-E04-F05-001 - Core Folder Structure & Path Management
**Dependencies**: None

Build the foundation for folder-based workflow:
- Define six-folder structure (todo/, active/, blocked/, ready-for-review/, completed/, archived/)
- Implement automatic folder creation (idempotent)
- Create STATUS_FOLDER_MAP constant mapping statuses to folders
- Implement `get_file_path_for_status()` for path resolution
- Project root detection (.git or .pmconfig.json)
- Cross-platform path handling with pathlib.Path

**Success Gates**:
- All six folders created automatically on initialization
- Status-to-folder mapping works for all six statuses
- Cross-platform path handling verified (Windows, Linux, macOS)
- Folder creation completes in <100ms
- Unit tests cover all path mapping scenarios

**Key Deliverable**: `pm/file_management/folders.py` with folder utilities

### Phase 2A: File Operations (6 hours)

**Task**: T-E04-F05-002 - Atomic File Operations with Error Handling
**Dependencies**: T-E04-F05-001

Implement reliable file move operations:
- `move_task_file()` function using shutil.move()
- Source file verification before move
- Destination file verification after move
- Parent directory creation if needed
- Custom exception hierarchy (FileOperationError, FileMoveError)
- Comprehensive error handling (FileNotFoundError, FileExistsError, PermissionError)

**Success Gates**:
- File moves are atomic (no partial moves)
- All error scenarios properly handled with clear messages
- Source file deleted after successful move
- Destination file exists and is readable after move
- Operations complete in <50ms on SSD
- 100% test coverage for file operations

**Key Deliverable**: `pm/file_management/operations.py` with atomic file operations

### Phase 2B: Validation System (8 hours) - Can run parallel with 2A

**Task**: T-E04-F05-003 - Validation & Repair System
**Dependencies**: T-E04-F05-002

Build the safety net for consistency checking:
- `validate_folder_structure()` with optimized folder scanning
- Detect two mismatch types: MISSING_FILE, WRONG_FOLDER
- `repair_mismatches()` for automated repair
- Dry-run support (preview repairs without changes)
- ValidationResult and RepairResult dataclasses
- Optimized algorithm: scan folders once, not per-task

**Success Gates**:
- Validation of 1,000 tasks completes in <5 seconds
- Repair of 100 mismatches completes in <10 seconds
- Both mismatch types detected correctly
- Dry-run shows preview without making changes
- WRONG_FOLDER mismatches auto-repaired
- MISSING_FILE mismatches reported for manual intervention

**Key Deliverable**: `pm/file_management/validation.py` with validation and repair logic

### Phase 3: Database Integration (6 hours)

**Task**: T-E04-F05-004 - Database-File Synchronization with Rollback
**Dependencies**: T-E04-F05-002

Implement atomic DB-file synchronization:
- `update_task_with_file_move()` integration service
- Sequential pattern: update DB first, then move file
- Compensating transaction on file move failure
- Rollback database changes if file move fails
- Integration with E04-F01 transaction context

**Success Gates**:
- Database and file stay synchronized (no split-brain)
- File move failure triggers DB rollback
- All error scenarios properly rolled back
- Integration tests verify end-to-end synchronization
- Rollback works for all failure scenarios

**Key Deliverable**: `pm/file_management/integration.py` with DB-file synchronization

### Phase 4: CLI Commands (6 hours)

**Task**: T-E04-F05-005 - CLI Commands for Validation & Repair
**Dependencies**: T-E04-F05-003, T-E04-F05-004

Implement user-facing validation commands:
- `pm validate` command to check consistency
- `pm validate --repair` to fix mismatches
- `pm validate --dry-run` to preview repairs
- `pm validate --json` for agent consumption
- Human-readable table output
- Clear exit codes (0=valid, 1=mismatches)

**Success Gates**:
- All command variations work correctly
- Table output fits in 80-column terminal
- JSON output is valid and parseable
- Exit codes correct for all scenarios
- Error messages are clear and actionable

**Key Deliverable**: `pm/cli/validate.py` with validation CLI commands

### Phase 5: Task Operations Integration (4 hours)

**Task**: T-E04-F05-006 - Integration with Task Operations
**Dependencies**: T-E04-F05-004

Integrate with task lifecycle operations:
- Update all E04-F03 task operations to use `update_task_with_file_move()`
- Automatic file moves for all status changes
- Error handling for file operation failures
- Integration tests for all status transitions

**Success Gates**:
- All task operations move files automatically
- Start task: todo/ → active/
- Complete task: active/ → ready-for-review/
- Block task: any/ → blocked/
- Approve task: ready-for-review/ → completed/
- Archive task: completed/ → archived/
- File move failures roll back status changes

**Key Deliverable**: Updated task operations with automatic file management

## Design Documentation

All tasks reference these design documents:

- **[PRD](../../../docs/plan/E04-task-mgmt-cli-core/E04-F05-folder-management/prd.md)** - Complete requirements and acceptance criteria
- **[Architecture](../../../docs/plan/E04-task-mgmt-cli-core/E04-F05-folder-management/02-architecture.md)** - System design and technical decisions

## Dependencies & Critical Path

```
E04-F01 (Database Schema) [COMPLETE]
E04-F02 (CLI Infrastructure) [COMPLETE]
E04-F03 (Task Operations) [REQUIRED FOR T-E04-F05-006]
    ↓
T-E04-F05-001 (Folder Structure) [4h] ← FOUNDATION
    ↓
T-E04-F05-002 (File Operations) [6h] ← CRITICAL PATH
    ↓ (parallel split)
T-E04-F05-003 (Validation) [8h] ← CRITICAL PATH (longest task)
    ↓ AND
T-E04-F05-004 (DB Integration) [6h]
    ↓ (parallel join)
T-E04-F05-005 (CLI Commands) [6h]
    ↓ AND
T-E04-F05-006 (Task Ops Integration) [4h]
    ↓
FEATURE COMPLETE
```

**Critical Path**: T-E04-F05-001 → T-E04-F05-002 → T-E04-F05-003 → T-E04-F05-005 (24 hours total)

**Parallelization Opportunities**:
- T-E04-F05-003 and T-E04-F05-004 can run in parallel after T-E04-F05-002 completes
- T-E04-F05-005 and T-E04-F05-006 can run in parallel after their dependencies complete

**Minimum Duration**: ~24 hours with parallelization (vs 34 hours sequential)

## Downstream Feature Dependencies

Once all tasks are completed, these features depend on folder management:

### E04-F06: Task Creation
**What they need**:
- `get_file_path_for_status()` to determine initial file location
- Folder structure for placing new task files

**Integration pattern**: New tasks created with status=todo, file placed in docs/tasks/todo/

### E04-F07: Sync/Import
**What they need**:
- `validate_folder_structure()` to check consistency after import
- `repair_mismatches()` to fix imported files in wrong folders

**Integration pattern**: After importing tasks, run validation and optionally auto-repair

### E04-F03: Task Operations (enhanced)
**What they need**:
- `update_task_with_file_move()` for all status change operations
- Automatic file moves when status changes

**Integration pattern**: Every status change triggers file move to appropriate folder

## Feature Metrics

**Code Volume Estimate**:
- `pm/file_management/folders.py`: ~200 lines
- `pm/file_management/operations.py`: ~300 lines
- `pm/file_management/validation.py`: ~400 lines
- `pm/file_management/integration.py`: ~250 lines
- `pm/file_management/exceptions.py`: ~100 lines
- `pm/file_management/models.py`: ~100 lines
- `pm/cli/validate.py`: ~300 lines
- Task operations integration: ~100 lines (modifications)
- Tests: ~2,500 lines
- **Total**: ~1,750 lines production code + 2,500 lines tests = ~4,250 lines

**Test Coverage Targets**:
- `folders.py`: 100% (foundational utilities)
- `operations.py`: 100% (critical file operations)
- `validation.py`: 95% (complex business logic)
- `integration.py`: 100% (critical synchronization)
- `cli/validate.py`: 90% (CLI interface)
- **Overall**: >95% code coverage

**Performance Targets** (from PRD):
- Folder creation: <100ms
- Single file move: <50ms on SSD
- Validation of 1,000 tasks: <5 seconds
- Repair of 100 mismatches: <10 seconds

## Architectural Decisions

### ADR-001: Blocked Tasks Get Their Own Folder
Tasks with `status="blocked"` are stored in `docs/tasks/blocked/`.

**Rationale**: Maintains core principle that folder location always matches database status. Makes blocked tasks easily discoverable.

**Impact**: More file moves when blocking/unblocking tasks, but moves are fast (<50ms) and agent works sequentially.

### ADR-002: Sequential DB-Then-File Updates
Update database first, then move file. No concurrent access locking needed.

**Rationale**: Single agent executing CLI commands sequentially. Simple error handling: if file move fails, rollback DB.

**Impact**: Brief window (1-50ms) where DB is updated but file hasn't moved yet. If process crashes, validation will detect and repair.

### ADR-003: Validation as Safety Net
Provide `pm validate` command to detect and repair inconsistencies.

**Rationale**: Handles edge cases (process crashes, manual file moves, bugs). Agent can run validation periodically or after errors.

**Impact**: Validation is not needed during normal operations - only for recovery scenarios.

## Notes

- **Sequential Operations**: CLI executes one operation at a time - no file locking or concurrent access handling needed
- **Database is Source of Truth**: Task status lives in database. Folder location is derived state that mirrors the database.
- **Simple Error Recovery**: If file operations fail, rollback database changes. Validation detects and repairs drift.
- **Cross-Platform**: All path operations use pathlib.Path for Windows/Linux/macOS compatibility
- **Performance Focus**: Validation uses optimized algorithm (scan folders once, not per-task)
- **Agent-Friendly**: Validation provides JSON output for AI agent consumption
- **No File Locking**: Sequential execution means no need for file locks or concurrent access control
- **Compensating Transactions**: Manual rollback of DB changes when file moves fail (not ACID rollback)

## Key Design Patterns

1. **Folder-as-Status**: File location always matches database status (1:1 mapping)
2. **Compensating Transactions**: Manual DB rollback when file operations fail
3. **Optimized Validation**: Scan folders once, compare in memory (not N filesystem calls)
4. **Idempotent Operations**: Folder creation and validation can run multiple times safely
5. **Fail-Fast**: Validate preconditions before attempting operations
6. **Clear Exceptions**: All errors include full paths and clear messages

---

**Task Index Created**: 2025-12-14
**Feature Status**: PRD and Architecture complete, ready for implementation
**Next Action**: Begin Phase 1 (T-E04-F05-001) or assign to implementation agent
