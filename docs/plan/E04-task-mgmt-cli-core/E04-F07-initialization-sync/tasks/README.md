# Implementation Tasks: Initialization & Synchronization

## Overview

This folder contains agent-executable tasks that implement the E04-F07-initialization-sync feature in logical phases. The feature adds `pm init` and `pm sync` commands to set up the CLI infrastructure and synchronize task files with the database.

## Active Tasks

| Task | Status | Assigned Agent | Dependencies | Estimated Time |
|------|--------|----------------|--------------|----------------|
| [T-E04-F07-001](./T-E04-F07-001.md) | created | api-developer | None | 8 hours |
| [T-E04-F07-002](./T-E04-F07-002.md) | created | api-developer | None | 10 hours |
| [T-E04-F07-003](./T-E04-F07-003.md) | created | api-developer | None | 8 hours |
| [T-E04-F07-004](./T-E04-F07-004.md) | created | api-developer | T-E04-F07-001 | 8 hours |
| [T-E04-F07-005](./T-E04-F07-005.md) | created | api-developer | T-E04-F07-001, T-E04-F07-003, T-E04-F07-004 | 12 hours |
| [T-E04-F07-006](./T-E04-F07-006.md) | created | api-developer | T-E04-F07-002, T-E04-F07-005 | 10 hours |

**Total Estimated Time**: 56 hours (7 developer-days)

## Workflow

### Execution Order

Tasks can be executed in the following order, with some parallelization opportunities:

**Phase 1: Foundation (Parallel Possible)**
1. **Task 001**: Repository Extensions (8 hours) - Can run independently
2. **Task 002**: Init Command (10 hours) - Can run independently
3. **Task 003**: File Scanner (8 hours) - Can run independently

**Phase 2: Conflict Handling (Depends on Task 001)**
4. **Task 004**: Conflict Detection & Resolution (8 hours) - Depends on Task 001

**Phase 3: Sync Integration (Depends on Previous)**
5. **Task 005**: Sync Engine & Command (12 hours) - Depends on Tasks 001, 003, 004

**Phase 4: Validation (Depends on All Previous)**
6. **Task 006**: Testing & Documentation (10 hours) - Depends on Tasks 002, 005

### Critical Path

```
Task 001 (8h) → Task 004 (8h) → Task 005 (12h) → Task 006 (10h) = 38 hours
```

### Parallel Execution

Tasks 001, 002, and 003 can be executed in parallel, reducing calendar time:
- **Sequential**: 56 hours
- **With Parallelization**: ~38 hours (assuming 3 parallel developers for Phase 1)

## Task Descriptions

### Task 001: Repository Extensions for Bulk Operations
Extends existing repository layer with methods optimized for sync operations: BulkCreate for efficient multi-task insertion, GetByKeys for bulk lookups, UpdateMetadata for preserving database-only fields, and CreateIfNotExists for idempotent epic/feature creation.

**Key Deliverables**:
- `TaskRepository.BulkCreate()`, `GetByKeys()`, `UpdateMetadata()`
- `EpicRepository.CreateIfNotExists()`, `GetByKey()`
- `FeatureRepository.CreateIfNotExists()`, `GetByKey()`
- Comprehensive unit tests
- Performance benchmarks

### Task 002: Initialization Command Implementation
Implements `pm init` command to set up PM CLI infrastructure with database schema, folder structure, config file, and templates. Emphasizes idempotency and atomic operations for safety.

**Key Deliverables**:
- `internal/init/` package with orchestrator and sub-components
- `pm init` CLI command with flags
- Embedded templates (go:embed)
- Unit and integration tests

### Task 003: File Scanner and Parser Implementation
Implements file discovery and metadata extraction for sync engine. Recursively scans directories, filters task files, infers epic/feature context from structure, and validates file paths for security.

**Key Deliverables**:
- `internal/sync/scanner.go` with FileScanner
- Epic/feature inference logic
- Path validation and size limits
- Unit and integration tests

### Task 004: Conflict Detection and Resolution
Implements conflict detection and resolution logic to handle discrepancies between files and database. Supports three strategies (file-wins, database-wins, newer-wins) while preserving database-only fields.

**Key Deliverables**:
- `internal/sync/conflict.go` - ConflictDetector
- `internal/sync/resolver.go` - ConflictResolver
- Strategy implementations
- Comprehensive unit tests

### Task 005: Sync Engine and Command Implementation
Implements complete sync orchestration and `pm sync` CLI command. Coordinates all sync phases, manages transactions, handles errors, and generates detailed reports.

**Key Deliverables**:
- `internal/sync/engine.go` - SyncEngine orchestrator
- `pm sync` CLI command with all flags
- Transaction management
- Integration tests

### Task 006: Testing and Documentation
Completes test coverage, adds performance benchmarks, tests edge cases, and creates user documentation with examples and troubleshooting guides.

**Key Deliverables**:
- >80% code coverage
- Performance benchmarks
- Edge case tests
- User guides and troubleshooting docs

## Status Management

Task status is tracked in the database via `pm` CLI. Use these commands:

```bash
# List tasks by status
pm task list --status=created
pm task list --status=todo
pm task list --status=in_progress

# Update task status
pm task start T-E04-F07-001        # Mark as in_progress
pm task complete T-E04-F07-001     # Mark as ready_for_review
pm task approve T-E04-F07-001      # Mark as completed

# Block/unblock tasks
pm task block T-E04-F07-002 --reason="Waiting for design approval"
pm task unblock T-E04-F07-002
```

Task files remain in this `/docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/tasks/` directory regardless of status.

## Design Documentation

All tasks reference these design documents in the feature folder:

- [PRD](../prd.md) - Product Requirements Document
- [00-Research Report](../00-research-report.md) - Codebase analysis findings
- [01-Interface Contracts](../01-interface-contracts.md) - Component contracts and interfaces
- [02-Architecture](../02-architecture.md) - System architecture and data flows
- [03-Data Design](../03-data-design.md) - Database schema and data patterns
- [04-Backend Design](../04-backend-design.md) - Go package structure and implementation
- [05-Frontend Design](../05-frontend-design.md) - N/A (backend-only feature)
- [06-Security Design](../06-security-design.md) - Security measures and validations
- [08-Implementation Phases](../08-implementation-phases.md) - Phased implementation plan
- [09-Test Criteria](../09-test-criteria.md) - Testing requirements and acceptance tests

## Key Architectural Decisions

1. **Single Transaction**: All database operations in sync occur within one transaction for consistency
2. **Status is Database-Only**: File frontmatter does not contain status; database is source of truth
3. **Idempotent Operations**: Both init and sync are safe to run multiple times
4. **Bulk Operations**: Use prepared statements and IN clauses for performance
5. **Atomic File Writes**: Config files written atomically (temp + rename pattern)
6. **Context Propagation**: All operations accept context.Context for cancellation

## Performance Targets

From PRD requirements:
- `pm init`: <5 seconds
- `pm sync` with 100 files: <10 seconds
- YAML parsing: <10ms per file
- BulkCreate: 100 tasks in <1 second
- GetByKeys: 100 lookups in <100ms

## Testing Strategy

### Unit Tests
- Repository extensions (bulk operations, idempotent creation)
- Initializer components (database, folders, config, templates)
- File scanner (recursive traversal, pattern matching, inference)
- Conflict detection and resolution (all strategies)
- Sync engine components (isolated)

### Integration Tests
- Full `pm init` command execution
- Full `pm sync` command execution
- Transaction rollback on errors
- Epic/feature auto-creation with --create-missing
- Orphaned task cleanup with --cleanup

### Performance Tests
- Init speed (<5s)
- Sync speed with 100 files (<10s)
- Bulk operations benchmarks

### Edge Cases
- Empty database + empty filesystem
- Large frontmatter files (>1MB rejected)
- Concurrent sync operations
- Context cancellation (Ctrl+C)
- File permission errors
- Database locking (WAL mode)

## Getting Started

To begin implementation:

1. **Review design documents** in the feature folder
2. **Start with Task 001, 002, or 003** (they can run in parallel)
3. **Use `pm task start <key>`** to mark task as in-progress
4. **Reference design docs** for implementation details (tasks provide WHAT, not HOW)
5. **Write tests first** for critical paths (TDD approach recommended)
6. **Complete validation gates** before marking task ready for review
7. **Use `pm task complete <key>`** when done

## Notes for Implementers

- **Follow existing patterns**: Study existing repository and CLI code
- **Error handling**: Wrap errors with context using `fmt.Errorf` with `%w`
- **Logging**: Use structured logging for warnings and debug info
- **Security**: Validate file paths, enforce size limits, set correct permissions
- **Performance**: Profile if benchmarks don't meet targets
- **Documentation**: Update help text and examples as you implement

---

**Feature**: E04-F07-initialization-sync
**Total Tasks**: 6
**Status**: Ready for implementation
**Created**: 2025-12-16
