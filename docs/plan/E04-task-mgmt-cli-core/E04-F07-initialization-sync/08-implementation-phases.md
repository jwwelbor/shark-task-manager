# Implementation Phases: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-14
**Author**: feature-architect (coordinator)

## Overview

This document defines the implementation approach for the Initialization & Synchronization feature, breaking down the work into logical phases with clear deliverables, dependencies, and success criteria. Implementation follows a bottom-up approach: foundational components first (config, templates), then initialization service, then synchronization service.

---

## Dependencies

This feature depends on:

| Feature | Components Needed | Status |
|---------|------------------|--------|
| **E04-F01: Database Schema** | `init_database()`, `TaskRepository`, `FeatureRepository`, bulk operations, transactions | ✅ Implemented |
| **E04-F05: Folder Management** | `create_folder_structure()`, `TransactionalFileOperation`, folder-status mapping | 🔄 In Progress |
| **E04-F02: CLI Framework** | Click command registration, argument parsing, session management | 🔄 In Progress |

**Critical Dependencies**:
- E04-F01 must be complete for database operations
- E04-F05 must provide folder creation and file move operations
- E04-F02 must provide CLI command scaffolding

---

## Implementation Phases

### Phase 1: Configuration & Template Management

**Goal**: Implement configuration file and template management infrastructure

**Tasks**:
1. Create `ConfigManager` class (`pm/config/config_manager.py`)
   - `create_default_config()` - Write `.pmconfig.json`
   - `read_config()` - Parse and validate config
   - `update_config()` - Merge config updates
   - Atomic file writes (temp file + rename pattern)
   - JSON validation and error handling
2. Create `TemplateManager` class (`pm/templates/template_manager.py`)
   - `install_templates()` - Copy templates from package to project
   - `get_template()` - Load template by name
   - `render_template()` - Replace placeholders with values
3. Create template files (stored in `pm/templates/`)
   - `task.md` - Task/PRP template
   - `epic.md` - Epic PRD template
   - `feature.md` - Feature PRD template
4. Write unit tests
   - Test config create/read/update cycle
   - Test config validation (invalid JSON, wrong types)
   - Test template installation (new project, existing templates)
   - Test template rendering with placeholders

**Dependencies**: None (foundational phase)

**Deliverables**:
- `pm/config/__init__.py` - Package init
- `pm/config/config_manager.py` (~200 lines)
- `pm/config/models.py` - Config dataclass (~50 lines)
- `pm/templates/template_manager.py` (~150 lines)
- `pm/templates/task.md` - Task template
- `pm/templates/epic.md` - Epic template
- `pm/templates/feature.md` - Feature template
- `tests/unit/config/test_config_manager.py` (~150 lines)
- `tests/unit/templates/test_template_manager.py` (~100 lines)

**Success Criteria**:
- [ ] ConfigManager creates valid `.pmconfig.json`
- [ ] Config validation catches invalid JSON and wrong types
- [ ] Config updates preserve existing fields
- [ ] TemplateManager copies templates to project directory
- [ ] Template rendering replaces all placeholders
- [ ] All unit tests pass
- [ ] >80% test coverage for config and template modules

**Estimated Effort**: 4-6 hours

---

### Phase 2: Initialization Service

**Goal**: Implement `pm init` command logic

**Tasks**:
1. Create `InitService` class (`pm/services/init_service.py`)
   - `initialize_project()` - Orchestrate all init steps
   - `check_database_exists()` - Verify DB file exists
   - `check_folders_exist()` - Verify folder structure
   - `check_config_exists()` - Verify config file
   - Rollback logic for partial failures
   - Idempotency checks (safe to re-run)
2. Implement initialization steps
   - **Step 1**: Database creation (call E04-F01 `init_database()`)
   - **Step 2**: Folder creation (call E04-F05 `create_folder_structure()`)
   - **Step 3**: Config creation (call `ConfigManager.create_default_config()`)
   - **Step 4**: Template installation (call `TemplateManager.install_templates()`)
3. Create `InitResult` dataclass for reporting
   - Track what was created vs skipped
   - Collect warnings
   - Format text output with next steps
4. Write unit tests
   - Test first-time init (nothing exists)
   - Test re-init (everything exists) - verify idempotency
   - Test partial init (DB exists, folders don't)
   - Test rollback on database failure
   - Test rollback on folder failure
   - Test non-interactive mode (skip prompts)
   - Test force mode (overwrite existing)
5. Write integration tests
   - Full init flow on empty directory
   - Verify all artifacts created
   - Verify re-init doesn't error

**Dependencies**: Phase 1 (ConfigManager, TemplateManager)

**Deliverables**:
- `pm/services/__init__.py` - Package init
- `pm/services/init_service.py` (~300 lines)
- `pm/services/models.py` - InitResult dataclass (~50 lines)
- `tests/unit/services/test_init_service.py` (~250 lines)
- `tests/integration/test_init_integration.py` (~150 lines)

**Success Criteria**:
- [ ] Init creates all required artifacts (DB, folders, config, templates)
- [ ] Init is idempotent (safe to re-run)
- [ ] Init skips existing artifacts correctly
- [ ] Init prompts for config overwrite (interactive mode)
- [ ] Init skips prompts in non-interactive mode
- [ ] Init overwrites with --force flag
- [ ] Init rolls back on failures
- [ ] InitResult accurately reports what was created
- [ ] All unit and integration tests pass
- [ ] >80% test coverage for init_service.py

**Estimated Effort**: 6-8 hours

---

### Phase 3: Sync Foundation - File Scanning & Parsing

**Goal**: Implement file discovery and frontmatter parsing

**Tasks**:
1. Create `FileScanner` class (`pm/sync/file_scanner.py`)
   - `scan_folders()` - Recursively find *.md files
   - Support folder filtering (`--folder=todo`)
   - Return list of file paths
2. Create `FrontmatterParser` class (`pm/sync/frontmatter_parser.py`)
   - `parse_file()` - Extract YAML frontmatter
   - Use PyYAML safe loader
   - Validate required fields
   - Validate key matches filename
   - Return `TaskMetadata` dataclass or None
3. Create `TaskMetadata` dataclass (`pm/sync/models.py`)
   - All frontmatter fields typed
   - Validation methods
   - Conversion to DB format
4. Create validation utilities
   - `validate_task_key()` - Regex check
   - `validate_status()` - Enum check
   - `validate_agent_type()` - Enum check
   - `validate_dependencies()` - Array format
5. Write unit tests
   - Test scanning with/without folder filter
   - Test parsing valid frontmatter
   - Test parsing invalid YAML (returns None)
   - Test parsing missing required fields (returns None)
   - Test key-filename mismatch detection
   - Test all validation functions
6. Create test fixtures
   - Sample markdown files with valid frontmatter
   - Sample files with invalid YAML
   - Sample files with missing fields
   - Sample files with key mismatches

**Dependencies**: None (can run in parallel with Phase 2)

**Deliverables**:
- `pm/sync/__init__.py` - Package init
- `pm/sync/file_scanner.py` (~100 lines)
- `pm/sync/frontmatter_parser.py` (~200 lines)
- `pm/sync/models.py` - TaskMetadata and related dataclasses (~150 lines)
- `pm/sync/validation.py` - Validation utilities (~100 lines)
- `tests/unit/sync/test_file_scanner.py` (~100 lines)
- `tests/unit/sync/test_frontmatter_parser.py` (~200 lines)
- `tests/unit/sync/test_validation.py` (~150 lines)
- `tests/fixtures/sync_test_files/` - Sample markdown files

**Success Criteria**:
- [ ] FileScanner finds all *.md files in sync folders
- [ ] FileScanner respects folder filter
- [ ] FrontmatterParser extracts valid frontmatter
- [ ] FrontmatterParser returns None for invalid YAML
- [ ] FrontmatterParser returns None for missing fields
- [ ] Parser detects key-filename mismatches
- [ ] All validation functions work correctly
- [ ] All unit tests pass
- [ ] >80% test coverage for sync modules

**Estimated Effort**: 6-8 hours

---

### Phase 4: Sync Core - Comparison & Reconciliation

**Goal**: Implement metadata comparison and conflict resolution

**Tasks**:
1. Create `MetadataComparator` class (`pm/sync/comparator.py`)
   - `compare()` - Compare file metadata with DB task
   - Detect new tasks (no DB record)
   - Detect changes (field differences)
   - Detect conflicts (which fields differ)
   - Return `ComparisonResult` with conflict details
2. Create `ConflictReconciler` class (`pm/sync/reconciler.py`)
   - `reconcile()` - Apply conflict resolution strategy
   - Implement `FILE_WINS` strategy
   - Implement `DATABASE_WINS` strategy
   - Implement `NEWER_WINS` strategy (timestamp comparison)
   - Return `ReconciliationAction` (what to do)
3. Create `ConflictStrategy` enum
   - `FILE_WINS` - File metadata overwrites DB
   - `DATABASE_WINS` - DB is authoritative
   - `NEWER_WINS` - Most recent wins
4. Create action dataclasses
   - `ReconciliationAction` - What action to take
   - `ActionType` enum (CREATE_TASK, UPDATE_DATABASE, MOVE_FILE, SKIP)
   - Include conflict details for reporting
5. Implement folder-status validation
   - `validate_folder_status_match()` - Check file location matches status
   - Detect mismatches (e.g., file in `todo/` but status=`in_progress`)
   - Generate MOVE_FILE actions for mismatches
6. Write unit tests
   - Test comparison: new task (no DB record)
   - Test comparison: identical task (no changes)
   - Test comparison: conflicting fields
   - Test FILE_WINS strategy
   - Test DATABASE_WINS strategy
   - Test NEWER_WINS strategy
   - Test folder-status mismatch detection
   - Test action generation for each scenario

**Dependencies**: Phase 3 (FileScanner, FrontmatterParser, TaskMetadata)

**Deliverables**:
- `pm/sync/comparator.py` (~200 lines)
- `pm/sync/reconciler.py` (~250 lines)
- `pm/sync/models.py` (extended with comparison/action classes) (~100 lines added)
- `tests/unit/sync/test_comparator.py` (~200 lines)
- `tests/unit/sync/test_reconciler.py` (~250 lines)

**Success Criteria**:
- [ ] Comparator detects new tasks correctly
- [ ] Comparator detects conflicts on all fields
- [ ] Comparator reports identical tasks as NO_CHANGE
- [ ] FILE_WINS strategy generates UPDATE_DATABASE actions
- [ ] DATABASE_WINS strategy generates no DB actions
- [ ] NEWER_WINS compares timestamps correctly
- [ ] Folder-status mismatches generate MOVE_FILE actions
- [ ] All unit tests pass
- [ ] >80% test coverage

**Estimated Effort**: 8-10 hours

---

### Phase 5: Sync Service Integration

**Goal**: Orchestrate sync operations end-to-end

**Tasks**:
1. Create `SyncService` class (`pm/services/sync_service.py`)
   - `sync_filesystem()` - Main sync orchestration
   - Phase 1: Scan files
   - Phase 2: Parse frontmatter
   - Phase 3: Compare with DB
   - Phase 4: Reconcile conflicts
   - Phase 5: Apply actions (transactional)
   - Generate `SyncReport`
2. Implement bulk database operations
   - Query all tasks once (index by key for O(1) lookups)
   - Batch insert new tasks
   - Batch update existing tasks
   - Single transaction for all DB changes
3. Implement file operations
   - Move files to correct folders (after DB commit)
   - Log failures, don't rollback DB
   - Update file frontmatter if needed (database-wins mode)
4. Implement dry-run mode
   - Execute all analysis steps
   - Generate report of what would change
   - Don't modify DB or files
5. Create `SyncReport` dataclass
   - Track counts (scanned, imported, updated, conflicts)
   - Collect warnings and errors
   - Support text and JSON output formats
6. Implement feature auto-creation (`--create-missing` flag)
   - Parse feature key from frontmatter path
   - Query if feature exists
   - Create minimal feature record if missing
7. Write unit tests
   - Test sync with no files
   - Test sync with new files (import)
   - Test sync with changed files (update)
   - Test sync with conflicts
   - Test dry-run mode (no changes)
   - Test transactional rollback on DB error
   - Test feature auto-creation
8. Write integration tests
   - Full sync flow: scan → parse → compare → reconcile → apply
   - Verify DB updated correctly
   - Verify files moved correctly
   - Verify task history created
   - Test sync after Git pull (new files added)

**Dependencies**: Phase 4 (MetadataComparator, ConflictReconciler)

**Deliverables**:
- `pm/services/sync_service.py` (~400 lines)
- `pm/services/models.py` (extended with SyncReport) (~100 lines added)
- `tests/unit/services/test_sync_service.py` (~350 lines)
- `tests/integration/test_sync_integration.py` (~300 lines)

**Success Criteria**:
- [ ] Sync imports new tasks correctly
- [ ] Sync updates existing tasks with file changes
- [ ] Sync resolves conflicts per strategy
- [ ] Sync moves files to correct folders
- [ ] Sync creates task history records
- [ ] Sync uses single transaction for DB changes
- [ ] Sync rolls back on DB errors
- [ ] Dry-run mode doesn't modify anything
- [ ] SyncReport accurately tracks all operations
- [ ] Feature auto-creation works
- [ ] All unit and integration tests pass
- [ ] >80% test coverage

**Estimated Effort**: 10-12 hours

---

### Phase 6: CLI Command Integration

**Goal**: Integrate init and sync services with CLI commands

**Tasks**:
1. Create `pm init` command (`pm/cli/commands/init.py`)
   - Register with Click
   - Add `--non-interactive` flag
   - Add `--force` flag
   - Call `InitService.initialize_project()`
   - Display `InitResult` output
   - Handle errors, set exit codes
2. Create `pm sync` command (`pm/cli/commands/sync.py`)
   - Register with Click
   - Add `--folder` option
   - Add `--strategy` option (file-wins, database-wins, newer-wins)
   - Add `--dry-run` flag
   - Add `--create-missing` flag
   - Add `--json` flag
   - Call `SyncService.sync_filesystem()`
   - Display `SyncReport` (text or JSON)
   - Handle errors, set exit codes
3. Create output formatters
   - `format_init_result()` - Human-readable init output
   - `format_sync_report()` - Human-readable sync output
   - Color support (use Click.style if config.color_enabled)
4. Wire up dependency injection
   - Pass SessionFactory to services
   - Pass ConfigManager to services
   - Use Click context for shared state
5. Write CLI tests
   - Test `pm init` with all flag combinations
   - Test `pm sync` with all flag combinations
   - Test error scenarios (DB not found, permission denied)
   - Test JSON output format
   - Test exit codes (0 for success, 2 for errors)
6. Update CLI documentation
   - Add `pm init` to command reference
   - Add `pm sync` to command reference
   - Document all flags and options
   - Provide usage examples

**Dependencies**: Phase 2 (InitService), Phase 5 (SyncService), E04-F02 (CLI Framework)

**Deliverables**:
- `pm/cli/commands/init.py` (~150 lines)
- `pm/cli/commands/sync.py` (~200 lines)
- `pm/cli/formatters.py` - Output formatting (~150 lines)
- `tests/cli/test_init_command.py` (~200 lines)
- `tests/cli/test_sync_command.py` (~250 lines)
- `docs/cli-commands.md` (updated with init/sync docs)

**Success Criteria**:
- [ ] `pm init` command registered and callable
- [ ] `pm init` creates all required artifacts
- [ ] `pm init --non-interactive` skips prompts
- [ ] `pm init --force` overwrites existing files
- [ ] `pm sync` command registered and callable
- [ ] `pm sync` imports and updates tasks correctly
- [ ] `pm sync --dry-run` previews changes
- [ ] `pm sync --json` outputs valid JSON
- [ ] `pm sync --folder=todo` syncs only todo folder
- [ ] `pm sync --strategy=database-wins` keeps DB authoritative
- [ ] Error messages are clear and actionable
- [ ] Exit codes correct (0 success, 2 error)
- [ ] All CLI tests pass
- [ ] Documentation complete and accurate

**Estimated Effort**: 6-8 hours

---

### Phase 7: Performance Optimization & Testing

**Goal**: Validate performance targets and optimize if needed

**Tasks**:
1. Write performance benchmarks
   - Init performance: <5 seconds
   - Sync 100 files: <10 seconds
   - YAML parsing: <10ms per file
   - Bulk insert: <2 seconds for 100 tasks
2. Profile sync operations
   - Identify bottlenecks (file I/O, DB, parsing)
   - Use cProfile or similar
3. Optimize as needed
   - Batch DB queries (fetch all tasks once)
   - Use bulk insert for new tasks
   - Minimize file system operations
   - Cache parsed metadata if scanning twice
4. Write stress tests
   - Sync 1000 files
   - Init on slow filesystem (NFS)
   - Concurrent sync operations (if applicable)
5. Validate error handling
   - Test disk full scenario
   - Test permission denied
   - Test corrupt YAML
   - Test database locked
   - Verify rollback works in all cases

**Dependencies**: Phase 6 (full integration)

**Deliverables**:
- `tests/performance/test_init_performance.py` (~100 lines)
- `tests/performance/test_sync_performance.py` (~150 lines)
- `tests/stress/test_large_sync.py` (~100 lines)
- Performance report documenting benchmark results

**Success Criteria**:
- [ ] Init completes in <5 seconds
- [ ] Sync 100 files in <10 seconds
- [ ] YAML parsing <10ms per file
- [ ] All performance benchmarks pass
- [ ] No performance regressions
- [ ] Error handling robust under stress
- [ ] Rollback works reliably

**Estimated Effort**: 4-6 hours

---

## Total Effort Estimate

| Phase | Effort (hours) |
|-------|---------------|
| Phase 1: Config & Templates | 4-6 |
| Phase 2: Init Service | 6-8 |
| Phase 3: File Scanning & Parsing | 6-8 |
| Phase 4: Comparison & Reconciliation | 8-10 |
| Phase 5: Sync Service Integration | 10-12 |
| Phase 6: CLI Command Integration | 6-8 |
| Phase 7: Performance & Testing | 4-6 |
| **Total** | **44-58 hours** |

**Recommended Schedule**: 6-8 business days (assuming 8-hour days)

---

## Dependency Graph

```
Phase 1: Config & Templates
    ↓
Phase 2: Init Service
    ↓
    ├──────────────────┐
    ↓                  ↓
Phase 3: File Scan   Phase 6: CLI Commands
    ↓                  (depends on 2 & 5)
Phase 4: Comparison
    ↓
Phase 5: Sync Service
    ↓
Phase 6: CLI Commands (completion)
    ↓
Phase 7: Performance
```

**Critical Path**: Phases 1 → 3 → 4 → 5 → 6 (must be sequential)

**Parallel Opportunities**:
- Phase 2 and Phase 3 can run in parallel (independent components)
- Phase 6 can start after Phase 2 completes (init command), then finish after Phase 5 (sync command)

---

## Risk Mitigation

### Risk 1: YAML Parsing Edge Cases

**Impact**: Medium (malformed frontmatter could crash parser)

**Mitigation**:
- Use PyYAML safe loader (no arbitrary code execution)
- Wrap all parsing in try/except
- Return None for invalid YAML, log warning
- Test with diverse YAML formats (lists, nested objects, multiline strings)

### Risk 2: File System Permissions

**Impact**: High (permission errors could block sync)

**Mitigation**:
- Check permissions before operations
- Provide clear error messages ("Permission denied: /path/to/file")
- Rollback DB changes if file operations fail
- Test on read-only filesystems

### Risk 3: Sync Performance on Large Repos

**Impact**: Medium (1000+ files could be slow)

**Mitigation**:
- Use bulk DB operations
- Profile early and optimize bottlenecks
- Consider parallel file scanning if needed
- Add progress indicator for long operations

### Risk 4: Conflict Resolution Complexity

**Impact**: Medium (edge cases in conflict logic)

**Mitigation**:
- Start with simple FILE_WINS strategy
- Add comprehensive unit tests for all strategies
- Document conflict resolution behavior clearly
- Provide dry-run mode for users to preview

---

## Validation Checklist

Before declaring Phase 7 complete, verify:

### Code Quality
- [ ] All modules have docstrings
- [ ] All public functions have docstrings
- [ ] mypy passes with no errors
- [ ] pylint score >8.0
- [ ] No TODO or FIXME comments left

### Functionality
- [ ] Init creates all required artifacts
- [ ] Init is idempotent
- [ ] Sync imports new tasks
- [ ] Sync updates existing tasks
- [ ] Sync resolves conflicts per strategy
- [ ] Sync moves files to correct folders
- [ ] Dry-run mode doesn't modify data
- [ ] Error handling robust

### Testing
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All performance benchmarks pass
- [ ] Test coverage >80%
- [ ] No test warnings or failures

### Performance
- [ ] Init completes in <5 seconds
- [ ] Sync 100 files in <10 seconds
- [ ] YAML parsing <10ms per file
- [ ] Bulk operations efficient

### Security
- [ ] YAML safe loader used
- [ ] File paths validated (no path traversal)
- [ ] Config validated before use
- [ ] No sensitive data logged

### Documentation
- [ ] CLI commands documented
- [ ] All flags and options explained
- [ ] Usage examples provided
- [ ] Error messages clear

---

## Handoff to Dependent Features

### E04-F03: Task Lifecycle Operations

**What they need from E04-F07**:
- Initialized database (via `pm init`)
- Task files synced to DB (via `pm sync`)
- Config file with defaults

**Integration pattern**:
- User runs `pm init` once
- User creates tasks via E04-F03 commands
- User optionally runs `pm sync` after external file edits

### E04-F06: Task Queries & Filtering

**What they need from E04-F07**:
- Populated database (tasks imported via sync)
- Config file for output preferences (JSON vs text, colors)

### Future Features

**What they might need**:
- `pm init` for project setup
- `pm sync` to maintain consistency with Git operations
- Config file for feature flags and preferences

---

## Success Metrics

**Definition of Done for E04-F07**:
1. All 7 phases complete
2. All validation checklist items checked
3. All tests passing (unit, integration, performance)
4. `pm init` and `pm sync` commands working end-to-end
5. Documentation complete

**Quality Gates**:
- Code review by tech lead
- Performance benchmarks verified
- User acceptance testing (run init and sync on real project)
- Security checklist reviewed

---

**Implementation Phases Complete**: 2025-12-14
**Ready for Development**: All specifications defined
**Next Step**: Begin Phase 1 implementation (Config & Templates)
