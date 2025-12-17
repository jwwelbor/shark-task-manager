# Implementation Phases: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: feature-architect (coordinator)

## Overview

This feature implements initialization (`pm init`) and synchronization (`pm sync`) commands for a Go-based CLI task manager. Implementation is organized into 5 phases with clear dependencies and success criteria.

---

## Documents Created

| Document | Created | Agent/Author |
|----------|---------|--------------|
| 00-research-report | ✅ | project-research-agent |
| 01-interface-contracts | ✅ | coordinator |
| 02-architecture | ✅ | backend-architect |
| 03-data-design | ✅ | db-admin |
| 04-backend-design | ✅ | backend-architect |
| 05-frontend-design | ✅ | frontend-architect |
| 06-security-design | ✅ | security-architect |
| 07-performance-design | ❌ | (not needed - no specific requirements) |
| 09-test-criteria | Pending | tdd-agent |

---

## Phase 1: Foundation - Repository Extensions

**Goals**: Extend existing repository layer with bulk operations and idempotent creation methods.

**Tasks**:
- [ ] Extend `TaskRepository` with `BulkCreate(ctx, tasks) (int, error)`
  - Use prepared statements for efficiency
  - Single transaction for all inserts
  - Validate all tasks before inserting
  - Reference: 04-backend-design.md, Section "Repository Extensions"

- [ ] Extend `TaskRepository` with `GetByKeys(ctx, keys) (map[string]*Task, error)`
  - Use IN clause for bulk lookup
  - Return map for O(1) access
  - Handle partial results (missing keys OK)
  - Reference: 04-backend-design.md, Section "Repository Extensions"

- [ ] Extend `TaskRepository` with `UpdateMetadata(ctx, task) error`
  - Update only: title, description, file_path
  - Preserve: status, priority, agent_type, depends_on
  - Reference: 01-interface-contracts.md, Section "Repository Extension Contracts"

- [ ] Extend `EpicRepository` with `CreateIfNotExists(ctx, epic) (*Epic, bool, error)`
  - Check existence before inserting (idempotent)
  - Return existing epic if already exists
  - Use transaction to prevent race conditions
  - Reference: 01-interface-contracts.md, Section "EpicRepository Extensions"

- [ ] Extend `EpicRepository` with `GetByKey(ctx, key) (*Epic, error)`
  - Query by epic key (not ID)
  - Return error if not found

- [ ] Extend `FeatureRepository` with `CreateIfNotExists(ctx, feature) (*Feature, bool, error)`
  - Check existence before inserting (idempotent)
  - Return existing feature if already exists
  - Validate epic_id foreign key

- [ ] Extend `FeatureRepository` with `GetByKey(ctx, key) (*Feature, error)`
  - Query by feature key (not ID)
  - Return error if not found

**Dependencies**: None (extends existing repositories)

**Success Criteria**:
- All new repository methods have unit tests
- BulkCreate can insert 100 tasks in <1 second
- GetByKeys can lookup 100 tasks in <100ms
- CreateIfNotExists is idempotent (tested with concurrent calls)
- All methods use context.Context for cancellation
- All methods return appropriate errors

**Estimated Effort**: 2-3 days

---

## Phase 2: Initialization - pm init Command

**Goals**: Implement `pm init` command to set up database, folders, config, and templates.

**Tasks**:
- [ ] Create `internal/init/` package
  - `initializer.go`: Main orchestrator
  - `database.go`: Database creation logic
  - `folders.go`: Folder creation logic
  - `config.go`: Config file generation
  - `templates.go`: Template handling
  - Reference: 04-backend-design.md, Section "Package: internal/init"

- [ ] Implement `Initializer.Initialize(ctx, opts) (*InitResult, error)`
  - Step 1: Create database (delegate to db.InitDB)
  - Step 2: Create folders (docs/plan, templates)
  - Step 3: Create config file (.pmconfig.json)
  - Step 4: Copy templates
  - Reference: 02-architecture.md, Section "Initializer"

- [ ] Implement database file permissions setting (Unix: 600)
  - Use os.Chmod on Unix systems
  - Skip on Windows
  - Reference: 06-security-design.md, Section "Database File Permissions"

- [ ] Implement atomic config file writing
  - Write to temp file
  - Sync to disk
  - Atomic rename
  - Reference: 06-security-design.md, Section "Atomic File Writes"

- [ ] Embed default templates in binary
  - Use go:embed for template files
  - Copy to templates/ folder during init
  - Reference: 04-backend-design.md, Section "templates.go"

- [ ] Create `internal/cli/commands/init.go`
  - Define Cobra command
  - Parse flags: --non-interactive, --force, --db, --config
  - Display results (human-readable and JSON)
  - Reference: 04-backend-design.md, Section "init.go"

- [ ] Implement idempotency
  - Check if database exists before creating
  - Check if folders exist before creating
  - Prompt before overwriting config (unless --force or --non-interactive)
  - Reference: 01-interface-contracts.md, Section "Initializer Interface"

- [ ] Write unit tests for initializer
  - Test database creation (idempotent)
  - Test folder creation (idempotent)
  - Test config creation (prompt behavior, --force flag)
  - Test template copying
  - Reference: 09-test-criteria.md (to be created)

**Dependencies**:
- Phase 1 (repository extensions not strictly required, but good to have)

**Success Criteria**:
- `pm init` completes in <5 seconds
- Init is idempotent (safe to run multiple times)
- Database created with correct schema
- Folders created with correct permissions (0755)
- Config created with correct format and permissions (0644)
- Templates copied successfully
- Non-interactive mode works (no prompts)
- JSON output works
- All acceptance criteria from PRD met

**Estimated Effort**: 3-4 days

---

## Phase 3: File Scanning & Parsing

**Goals**: Implement file scanning and frontmatter parsing for sync engine.

**Tasks**:
- [ ] Create `internal/sync/` package
  - `types.go`: Shared types (SyncOptions, SyncReport, Conflict, etc.)
  - `scanner.go`: File scanning logic
  - Reference: 04-backend-design.md, Section "Package: internal/sync"

- [ ] Implement `FileScanner.Scan(rootPath) ([]TaskFileInfo, error)`
  - Recursively walk directories using filepath.Walk
  - Filter files matching pattern T-*.md
  - Extract file metadata (path, name, modified time)
  - Infer epic and feature keys from directory structure
  - Reference: 02-architecture.md, Section "FileScanner"

- [ ] Implement epic/feature inference logic
  - Parse parent directory for feature key (E##-F##-*)
  - Parse grandparent directory for epic key (E##-*)
  - Fall back to parsing task key from filename
  - Return empty keys if inference fails
  - Reference: 02-architecture.md, Section "FileScanner"

- [ ] Validate file paths
  - Ensure paths are within allowed directories
  - Reject symlinks or validate symlink targets
  - Reference: 06-security-design.md, Section "File Path Validation"

- [ ] Implement file size limits
  - Reject files >1MB (DoS prevention)
  - Log warning for large files
  - Reference: 06-security-design.md, Section "File Limits"

- [ ] Write unit tests for file scanner
  - Test recursive directory traversal
  - Test pattern matching (T-*.md)
  - Test epic/feature inference from paths
  - Test symlink handling
  - Test file size limits
  - Reference: 09-test-criteria.md

**Dependencies**: None (can be developed in parallel with Phase 2)

**Success Criteria**:
- Scanner correctly identifies all T-*.md files in tree
- Epic/feature inference works for standard folder structure
- Scanner handles legacy folder structure (docs/tasks/*)
- File size limits enforced
- Path validation prevents traversal attacks
- Scanner completes 100 files in <1 second

**Estimated Effort**: 2-3 days

---

## Phase 4: Conflict Detection & Resolution

**Goals**: Implement conflict detection and resolution strategies.

**Tasks**:
- [ ] Create conflict detection logic in `internal/sync/conflict.go`
  - `ConflictDetector.DetectConflicts(fileData, dbTask) []Conflict`
  - Compare title, description, file_path
  - Do NOT compare status, priority, agent_type (database-only fields)
  - Reference: 02-architecture.md, Section "ConflictDetector"

- [ ] Create conflict resolution logic in `internal/sync/resolver.go`
  - `ConflictResolver.Resolve(conflicts, fileData, dbTask, strategy) (*Task, error)`
  - Implement file-wins strategy
  - Implement database-wins strategy
  - Implement newer-wins strategy (timestamp comparison)
  - Preserve database-only fields
  - Reference: 02-architecture.md, Section "ConflictResolver"

- [ ] Implement field comparison rules
  - Title: Conflict if file has title AND differs from DB
  - Description: Conflict if both exist AND differ
  - File path: Always conflict if DB path != actual path
  - Reference: 01-interface-contracts.md, Section "ConflictResolver Interface"

- [ ] Write unit tests for conflict detection
  - Test title conflict detection
  - Test description conflict detection
  - Test file_path conflict detection
  - Test no conflict for database-only fields
  - Reference: 09-test-criteria.md

- [ ] Write unit tests for conflict resolution
  - Test file-wins strategy
  - Test database-wins strategy
  - Test newer-wins strategy (timestamp comparison)
  - Test database-only fields are preserved
  - Reference: 09-test-criteria.md

**Dependencies**: Phase 1 (needs UpdateMetadata method)

**Success Criteria**:
- Conflict detection correctly identifies all conflicts
- No false positives (database-only fields don't trigger conflicts)
- File-wins strategy updates database
- Database-wins strategy keeps database unchanged
- Newer-wins strategy uses timestamps correctly
- All strategies preserve database-only fields (status, priority, etc.)

**Estimated Effort**: 2-3 days

---

## Phase 5: Sync Engine & pm sync Command

**Goals**: Implement complete sync orchestration and CLI command.

**Tasks**:
- [ ] Create sync engine in `internal/sync/engine.go`
  - `SyncEngine.Sync(ctx, opts) (*SyncReport, error)`
  - Orchestrate: scan → parse → query → detect → resolve → update
  - Use single transaction for all database operations
  - Support dry-run mode (no database changes)
  - Reference: 02-architecture.md, Section "SyncEngine"

- [ ] Implement file parsing logic
  - For each scanned file: Parse frontmatter using taskfile.ParseTaskFile
  - Validate required fields (key)
  - Build TaskMetadata structs
  - Collect warnings for invalid files
  - Reference: 02-architecture.md, Section "SyncEngine"

- [ ] Implement database query logic
  - Extract all task keys from parsed files
  - Bulk lookup using TaskRepository.GetByKeys
  - Build map for O(1) access
  - Reference: 02-architecture.md, Section "SyncEngine"

- [ ] Implement task import logic
  - For new tasks (not in database):
  - Validate epic/feature exists (or create if --create-missing)
  - Create Task model (status=todo, file_path=actual location)
  - Insert using TaskRepository.Create
  - Create history record (agent=sync, notes="Imported from file")
  - Reference: 02-architecture.md, Section "syncTask"

- [ ] Implement task update logic
  - For existing tasks (in database):
  - Detect conflicts using ConflictDetector
  - Resolve conflicts using ConflictResolver
  - Update using TaskRepository.UpdateMetadata
  - Create history record (agent=sync, notes="Updated from file: title, file_path")
  - Reference: 02-architecture.md, Section "syncTask"

- [ ] Implement --create-missing flag logic
  - Check if epic exists using EpicRepository.GetByKey
  - If not exists: Create epic using EpicRepository.CreateIfNotExists
  - Check if feature exists using FeatureRepository.GetByKey
  - If not exists: Create feature using FeatureRepository.CreateIfNotExists
  - Use minimal metadata (key + auto-generated title)
  - Reference: 03-data-design.md, Section "Auto-Creation Strategy"

- [ ] Implement --cleanup flag logic
  - Find orphaned tasks (file_path not in scanned files)
  - Delete using TaskRepository.Delete
  - History records auto-deleted via CASCADE
  - Reference: 03-data-design.md, Section "Query Patterns"

- [ ] Implement transaction management
  - Begin transaction at start (unless --dry-run)
  - Defer rollback (safety net)
  - Commit only if all operations succeed
  - Reference: 02-architecture.md, Section "Transaction Management"

- [ ] Implement sync report generation
  - Track: files_scanned, tasks_imported, tasks_updated, conflicts_resolved
  - Collect warnings (invalid YAML, missing fields, etc.)
  - Collect conflicts (with details)
  - Reference: 04-backend-design.md, Section "types.go"

- [ ] Create `internal/cli/commands/sync.go`
  - Define Cobra command
  - Parse flags: --folder, --dry-run, --strategy, --create-missing, --cleanup
  - Display sync report (human-readable and JSON)
  - Reference: 04-backend-design.md, Section "sync.go"

- [ ] Write integration tests for sync engine
  - Test full sync with new tasks
  - Test full sync with existing tasks (conflicts)
  - Test dry-run mode (no database changes)
  - Test file-wins, database-wins, newer-wins strategies
  - Test --create-missing flag
  - Test --cleanup flag
  - Test transaction rollback on error
  - Reference: 09-test-criteria.md

**Dependencies**:
- Phase 1: Repository extensions
- Phase 3: File scanning
- Phase 4: Conflict detection/resolution

**Success Criteria**:
- `pm sync` processes 100 files in <10 seconds
- All acceptance criteria from PRD met
- Dry-run mode works (preview without changes)
- All conflict resolution strategies work correctly
- Transaction rollback works on errors
- --create-missing flag creates epics/features correctly
- --cleanup flag deletes orphaned tasks
- Sync report is accurate
- JSON output works
- Integration tests pass

**Estimated Effort**: 4-5 days

---

## Phase 6: Testing & Documentation

**Goals**: Complete test coverage and update documentation.

**Tasks**:
- [ ] Write comprehensive unit tests
  - All packages >80% coverage (per Epic NFR)
  - Critical paths: 100% coverage (CRUD, transactions, conflict resolution)
  - Reference: 09-test-criteria.md

- [ ] Write integration tests
  - Test full init command execution
  - Test full sync command execution
  - Test init + sync workflow
  - Test Git pull + sync workflow
  - Reference: 09-test-criteria.md

- [ ] Write performance tests
  - Benchmark init (target: <5 seconds)
  - Benchmark sync with 100 files (target: <10 seconds)
  - Benchmark YAML parsing (target: <10ms per file)
  - Reference: 02-architecture.md, Section "Performance Optimization"

- [ ] Test edge cases
  - Empty database + empty filesystem
  - Large frontmatter (stress test)
  - Concurrent sync operations
  - File permissions issues
  - Database locked (WAL mode)
  - Context cancellation (Ctrl+C)

- [ ] Update CLI documentation
  - Update help text for `pm init`
  - Update help text for `pm sync`
  - Add examples to README

- [ ] Create user guide
  - First-time setup workflow
  - Git pull + sync workflow
  - Conflict resolution guide
  - Troubleshooting guide

**Dependencies**: Phase 2-5 (needs all implementation complete)

**Success Criteria**:
- All unit tests pass
- All integration tests pass
- Code coverage >80%
- Performance benchmarks meet targets
- All edge cases handled gracefully
- Documentation complete and accurate

**Estimated Effort**: 3-4 days

---

## Total Estimated Effort

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Phase 1: Repository Extensions | 2-3 days | None |
| Phase 2: Init Command | 3-4 days | None |
| Phase 3: File Scanning | 2-3 days | None |
| Phase 4: Conflict Detection/Resolution | 2-3 days | Phase 1 |
| Phase 5: Sync Engine & Command | 4-5 days | Phases 1, 3, 4 |
| Phase 6: Testing & Documentation | 3-4 days | Phases 2-5 |

**Total**: 16-22 days (3-4 weeks)

**Critical Path**: Phase 1 → Phase 4 → Phase 5 → Phase 6

**Parallelizable**: Phases 2 and 3 can be developed in parallel with Phase 1

---

## Risk Mitigation

### Risk: Performance does not meet targets

**Mitigation**:
- Profile early (after Phase 3)
- Optimize hot paths (prepared statements, bulk queries)
- Consider parallel file parsing if needed

### Risk: Conflict resolution logic is complex

**Mitigation**:
- Start with simple strategies (file-wins, database-wins)
- Add newer-wins after basic strategies work
- Comprehensive unit tests for all strategies

### Risk: Transaction rollback fails

**Mitigation**:
- Use deferred rollback (safety net)
- Test rollback extensively in integration tests
- Validate WAL mode is enabled (better concurrency)

### Risk: File path validation has security holes

**Mitigation**:
- Use absolute paths for all comparisons
- Test path traversal attempts
- Code review by security expert

---

## Definition of Done

**Feature is complete when**:
- [ ] All phases completed
- [ ] All unit tests pass (>80% coverage)
- [ ] All integration tests pass
- [ ] All acceptance criteria from PRD met
- [ ] Performance benchmarks meet targets
- [ ] Security checklist completed
- [ ] Documentation complete
- [ ] Code reviewed and approved
- [ ] Merged to main branch

---

## Next Steps

1. Review this implementation plan with team
2. Create tasks in task tracker (using `pm task create`)
3. Assign phases to developers
4. Begin Phase 1 (Repository Extensions)
5. Schedule daily standups to track progress
6. Review and adjust plan as needed

---

**Document Complete**: 2025-12-16
**Next Document**: 09-test-criteria.md (tdd-agent creates)
