# Feature: Initialization & Synchronization (E04-F07)

**Epic**: E04-task-mgmt-cli-core
**Status**: Ready for Implementation
**Created**: 2025-12-14

## Overview

This feature provides two critical commands for PM CLI project setup and maintenance:

1. **`pm init`** - Initialize new projects with database, folders, config, and templates
2. **`pm sync`** - Synchronize filesystem markdown files with database

## Documents

| Document | Purpose | Status |
|----------|---------|--------|
| [prd.md](./prd.md) | Product Requirements Document | ✅ Complete |
| [02-architecture.md](./02-architecture.md) | System architecture and integration | ✅ Complete |
| [03-data-design.md](./03-data-design.md) | Data structures and formats | ✅ Complete |
| [08-implementation-phases.md](./08-implementation-phases.md) | Implementation phases and timeline | ✅ Complete |

## Implementation Tasks

All tasks are created in `/home/jwwelbor/.claude/docs/tasks/todo/` with sequential numbering:

| Task | Description | Dependencies | Estimated Time |
|------|-------------|--------------|----------------|
| [T-E04-F07-001](../../../tasks/todo/T-E04-F07-001.md) | Configuration & Template Management | None | 6 hours |
| [T-E04-F07-002](../../../tasks/todo/T-E04-F07-002.md) | Initialization Service Implementation | T-E04-F07-001 | 8 hours |
| [T-E04-F07-003](../../../tasks/todo/T-E04-F07-003.md) | File Scanning & Frontmatter Parsing | None | 8 hours |
| [T-E04-F07-004](../../../tasks/todo/T-E04-F07-004.md) | Metadata Comparison & Conflict Resolution | T-E04-F07-003 | 10 hours |
| [T-E04-F07-005](../../../tasks/todo/T-E04-F07-005.md) | Sync Service Integration | T-E04-F07-004 | 12 hours |
| [T-E04-F07-006](../../../tasks/todo/T-E04-F07-006.md) | CLI Command Integration | T-E04-F07-002, T-E04-F07-005 | 8 hours |
| [T-E04-F07-007](../../../tasks/todo/T-E04-F07-007.md) | Performance Optimization & Validation | T-E04-F07-006 | 6 hours |

**Total Estimated Effort**: 58 hours (7-8 business days)

## Dependencies

This feature depends on:

| Feature | Required Components | Status |
|---------|-------------------|--------|
| **E04-F01: Database Schema** | `init_database()`, repositories, transactions | ✅ Implemented |
| **E04-F05: Folder Management** | `create_folder_structure()`, file operations | 🔄 In Progress |
| **E04-F02: CLI Framework** | Click commands, argument parsing | 🔄 In Progress |

## Key Features

### pm init Command

```bash
# Initialize new project
pm init

# Non-interactive mode (for automation)
pm init --non-interactive

# Force overwrite existing config
pm init --force
```

**Creates**:
- SQLite database (`project.db`)
- Folder structure (`docs/tasks/{todo,active,ready-for-review,completed,archived}`)
- Configuration file (`.pmconfig.json`)
- Task templates (`templates/task.md`, `templates/epic.md`, `templates/feature.md`)

**Features**:
- Idempotent (safe to re-run)
- Transactional (rollback on failure)
- Interactive prompts (skippable with `--non-interactive`)

### pm sync Command

```bash
# Sync all task files
pm sync

# Preview changes without applying
pm sync --dry-run

# Sync only specific folder
pm sync --folder=todo

# Choose conflict resolution strategy
pm sync --strategy=file-wins        # File is authoritative (default)
pm sync --strategy=database-wins    # Database is authoritative
pm sync --strategy=newer-wins       # Most recent change wins

# Auto-create missing features
pm sync --create-missing

# JSON output for scripting
pm sync --json
```

**Capabilities**:
- Import new task files into database
- Update existing tasks from file changes
- Resolve conflicts with configurable strategies
- Move files to correct folders based on status
- Create task history records for audit trail
- Validate frontmatter and folder locations

## Implementation Approach

### Phase Breakdown

1. **Phase 1** (6h): Config & Templates
   - ConfigManager for `.pmconfig.json`
   - TemplateManager for task/epic/feature templates

2. **Phase 2** (8h): Init Service
   - Orchestrate database, folders, config, templates
   - Idempotency and rollback logic

3. **Phase 3** (8h): File Scanning & Parsing
   - FileScanner for discovering markdown files
   - FrontmatterParser for extracting YAML metadata

4. **Phase 4** (10h): Comparison & Reconciliation
   - MetadataComparator for detecting changes
   - ConflictReconciler for resolution strategies

5. **Phase 5** (12h): Sync Service
   - End-to-end sync orchestration
   - Transactional database updates
   - Bulk operations for performance

6. **Phase 6** (8h): CLI Commands
   - `pm init` and `pm sync` command handlers
   - Output formatting (text and JSON)
   - Error handling and exit codes

7. **Phase 7** (6h): Performance & Testing
   - Validate performance targets
   - Stress testing and optimization
   - Comprehensive error handling validation

### Execution Order

```
Phase 1 (Config & Templates)
    │
    ├─→ Phase 2 (Init Service) ──┐
    │                             │
    └─→ Phase 3 (File Scan) ──────┤
            │                     │
            └─→ Phase 4 (Comparison)
                    │
                    └─→ Phase 5 (Sync Service)
                            │
                            └─→ Phase 6 (CLI Commands)
                                    │
                                    └─→ Phase 7 (Performance)
```

**Parallel Opportunities**:
- Phase 2 and Phase 3 can run in parallel
- Phase 6 can start after Phase 2 (init command), complete after Phase 5 (sync command)

## Success Criteria

### Performance Targets
- ✅ Init completes in <5 seconds
- ✅ Sync processes 100 files in <10 seconds
- ✅ YAML parsing <10ms per file

### Functionality
- ✅ `pm init` creates all required artifacts
- ✅ Init is idempotent (safe to re-run)
- ✅ `pm sync` imports new tasks
- ✅ Sync updates existing tasks
- ✅ Conflict resolution strategies work correctly
- ✅ Dry-run mode doesn't modify data
- ✅ Error handling robust with rollback

### Quality
- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Test coverage >80%
- ✅ mypy type checking passes
- ✅ Documentation complete

## Usage Examples

### Setting Up New Project

```bash
# Initialize PM CLI
pm init

# Edit configuration
vim .pmconfig.json

# Create first task
pm task create --epic=E04 --feature=F01 --title="Database setup" --agent=backend-developer

# Import existing markdown tasks
pm sync
```

### Git Workflow Integration

```bash
# After git pull (new task files added)
git pull origin main
pm sync

# Verify what will change before syncing
pm sync --dry-run

# Sync with file-wins strategy
pm sync --strategy=file-wins
```

### Automation

```bash
# CI/CD initialization
pm init --non-interactive

# Programmatic sync with JSON output
pm sync --json | jq '.imported'
```

## Next Steps

1. Review design documents for completeness
2. Validate dependencies (E04-F01, E04-F05, E04-F02)
3. Execute tasks in dependency order
4. Run integration tests after each phase
5. Document learnings and patterns

## Related Features

- **E04-F01**: Database Schema (provides data layer)
- **E04-F05**: Folder Management (provides file operations)
- **E04-F02**: CLI Framework (provides command infrastructure)
- **E04-F03**: Task Lifecycle Operations (uses init and sync)

---

**Ready for Implementation**: 2025-12-14
**Implementation Start**: Awaiting E04-F05 completion
**Target Completion**: 7-8 business days from start
