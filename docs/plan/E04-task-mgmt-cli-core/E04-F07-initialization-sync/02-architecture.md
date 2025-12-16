# Architecture: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-14
**Author**: feature-architect

## Overview

This document defines the system architecture for the Initialization & Synchronization feature, which provides two critical commands: `pm init` for project setup and `pm sync` for filesystem-database synchronization. The architecture integrates with E04-F01 (Database Schema), E04-F05 (Folder Management), and E04-F02 (CLI Framework) to deliver reliable, transactional initialization and sync operations.

---

## Architectural Overview

### System Context

```
┌─────────────────────────────────────────────────────────────┐
│                     PM CLI Ecosystem                         │
│                                                              │
│  ┌──────────────────┐          ┌──────────────────┐        │
│  │   E04-F02: CLI   │◄─────────┤  E04-F07: Init   │        │
│  │   Framework      │          │  & Sync          │        │
│  └──────────────────┘          └──────────────────┘        │
│           ▲                             │                   │
│           │                             ▼                   │
│  ┌──────────────────┐          ┌──────────────────┐        │
│  │  E04-F01: DB     │◄─────────┤  E04-F05: Folder │        │
│  │  Schema          │          │  Management      │        │
│  └──────────────────┘          └──────────────────┘        │
└─────────────────────────────────────────────────────────────┘
         │                               │
         ▼                               ▼
    SQLite DB                    Filesystem (Markdown)
```

**Key Dependencies**:
- **E04-F01**: Provides `init_database()`, bulk operations, repositories
- **E04-F05**: Provides folder structure creation, file operations
- **E04-F02**: Provides CLI command framework, argument parsing

---

## Component Architecture

### High-Level Components

```
┌───────────────────────────────────────────────────────────────┐
│                       pm init / pm sync                        │
│                     (CLI Command Layer)                        │
└───────────────────────┬───────────────────────────────────────┘
                        │
        ┌───────────────┴───────────────┐
        │                               │
        ▼                               ▼
┌─────────────────┐           ┌─────────────────┐
│  InitService    │           │  SyncService    │
│  - Database     │           │  - File Scanner │
│  - Folders      │           │  - Parser       │
│  - Config       │           │  - Comparator   │
│  - Templates    │           │  - Reconciler   │
└─────────────────┘           └─────────────────┘
        │                               │
        └───────────────┬───────────────┘
                        │
        ┌───────────────┴───────────────┐
        │                               │
        ▼                               ▼
┌─────────────────┐           ┌─────────────────┐
│  Database Layer │           │  Filesystem     │
│  (E04-F01)      │           │  (E04-F05)      │
│  - Repos        │           │  - File I/O     │
│  - Transactions │           │  - Moves        │
└─────────────────┘           └─────────────────┘
```

---

## Component Specifications

### 1. InitService

**Responsibility**: Orchestrate project initialization

**Location**: `pm/services/init_service.py`

**Public Interface**:
```python
class InitService:
    def initialize_project(
        self,
        non_interactive: bool = False,
        force: bool = False
    ) -> InitResult:
        """Initialize PM CLI infrastructure"""
```

**Operations** (in order):
1. Check if database exists → skip creation if present
2. Create database schema (E04-F01 `init_database()`)
3. Check if folders exist → skip creation if present
4. Create folder structure (E04-F05 folder creation)
5. Check if config exists → prompt or skip based on flags
6. Create `.pmconfig.json` with defaults
7. Copy templates to `templates/` folder
8. Return success summary

**Error Handling**:
- Database creation failure: raise `InitializationError`
- Folder creation failure: rollback DB, raise error
- Config write failure: warn but don't fail (non-critical)

**Idempotency**: Safe to run multiple times; skips existing resources.

---

### 2. SyncService

**Responsibility**: Synchronize filesystem markdown files with database

**Location**: `pm/services/sync_service.py`

**Public Interface**:
```python
class SyncService:
    def sync_filesystem(
        self,
        folder: Optional[str] = None,
        strategy: ConflictStrategy = ConflictStrategy.FILE_WINS,
        dry_run: bool = False,
        create_missing: bool = False
    ) -> SyncReport:
        """Sync filesystem with database"""
```

**Sub-Components**:

#### 2.1 FileScanner
**Responsibility**: Discover markdown files in sync folders

```python
class FileScanner:
    def scan_folders(self, folder_filter: Optional[str]) -> List[Path]:
        """Recursively scan for *.md files"""
```

**Folders scanned**:
- `docs/tasks/todo/`
- `docs/tasks/active/`
- `docs/tasks/ready-for-review/`
- `docs/tasks/completed/`
- `docs/tasks/archived/`

#### 2.2 FrontmatterParser
**Responsibility**: Parse YAML frontmatter from markdown files

```python
class FrontmatterParser:
    def parse_file(self, file_path: Path) -> Optional[TaskMetadata]:
        """Parse frontmatter, return None if invalid"""
```

**Validation**:
- YAML structure valid
- Required fields present (key, title)
- Key matches filename
- Epic/feature keys exist in DB (or --create-missing)

#### 2.3 MetadataComparator
**Responsibility**: Compare file metadata with database records

```python
class MetadataComparator:
    def compare(
        self,
        file_meta: TaskMetadata,
        db_task: Optional[Task]
    ) -> ComparisonResult:
        """Detect new, updated, or conflicting tasks"""
```

**Comparison Logic**:
- Task doesn't exist in DB → NEW
- Task exists, no differences → NO_CHANGE
- Task exists, differences found → CONFLICT

**Conflict Detection**: Compare these fields:
- status
- priority
- title
- description
- agent_type
- depends_on

#### 2.4 ConflictReconciler
**Responsibility**: Apply conflict resolution strategy

```python
class ConflictReconciler:
    def reconcile(
        self,
        conflict: Conflict,
        strategy: ConflictStrategy
    ) -> ReconciliationAction:
        """Determine action based on strategy"""
```

**Strategies**:
- `FILE_WINS`: Update DB from file
- `DATABASE_WINS`: Keep DB, optionally update file
- `NEWER_WINS`: Compare timestamps, apply most recent

**Actions**:
- `UPDATE_DATABASE`: Write file metadata to DB
- `UPDATE_FILE`: Write DB metadata to file
- `MOVE_FILE`: Relocate file to match status
- `CREATE_TASK`: Insert new task
- `SKIP`: No action (dry-run or no change)

---

### 3. ConfigManager

**Responsibility**: Manage `.pmconfig.json` file

**Location**: `pm/config/config_manager.py`

**Public Interface**:
```python
class ConfigManager:
    def create_default_config(self, overwrite: bool = False) -> None:
        """Create .pmconfig.json with defaults"""

    def read_config(self) -> Config:
        """Read and parse config file"""
```

**Default Configuration**:
```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false
}
```

---

### 4. TemplateManager

**Responsibility**: Copy task templates to templates/ folder

**Location**: `pm/templates/template_manager.py`

**Public Interface**:
```python
class TemplateManager:
    def install_templates(self, force: bool = False) -> None:
        """Copy templates from package to project"""
```

**Templates Included**:
- `task.md` - Default task template
- `epic.md` - Epic PRD template
- `feature.md` - Feature PRD template

---

## Data Flow

### pm init Flow

```
User runs: pm init [--non-interactive] [--force]
    │
    ▼
InitService.initialize_project()
    │
    ├─► Check DB exists? → No: init_database() (E04-F01)
    │                   → Yes: Skip
    │
    ├─► Check folders exist? → No: create_folder_structure() (E04-F05)
    │                        → Yes: Skip
    │
    ├─► Check config exists? → No: create_default_config()
    │                        → Yes: Prompt (skip if --non-interactive)
    │                                Overwrite if --force
    │
    ├─► Install templates → template_manager.install_templates()
    │
    └─► Return InitResult(
            database_created: bool,
            folders_created: bool,
            config_created: bool,
            templates_installed: bool
        )
    │
    ▼
Display success message + next steps
```

---

### pm sync Flow

```
User runs: pm sync [--folder=todo] [--strategy=file-wins] [--dry-run]
    │
    ▼
SyncService.sync_filesystem()
    │
    ├─► FileScanner.scan_folders(folder_filter)
    │   └─► Returns: List[Path] (*.md files)
    │
    ├─► For each file:
    │   │
    │   ├─► FrontmatterParser.parse_file(path)
    │   │   ├─► Invalid YAML? → Log warning, skip
    │   │   ├─► Missing fields? → Log warning, skip
    │   │   └─► Key mismatch? → Log warning, skip
    │   │
    │   ├─► Query database for task (by key)
    │   │
    │   ├─► MetadataComparator.compare(file_meta, db_task)
    │   │   ├─► NEW: No DB record
    │   │   ├─► NO_CHANGE: Identical
    │   │   └─► CONFLICT: Differences detected
    │   │
    │   └─► ConflictReconciler.reconcile(conflict, strategy)
    │       ├─► FILE_WINS: Update DB, maybe move file
    │       ├─► DATABASE_WINS: Keep DB, maybe update file
    │       └─► NEWER_WINS: Compare timestamps
    │
    ├─► If NOT dry_run:
    │   │
    │   ├─► Begin transaction
    │   │
    │   ├─► Apply all reconciliation actions:
    │   │   ├─► UPDATE_DATABASE → repo.update(task)
    │   │   ├─► CREATE_TASK → repo.create(task)
    │   │   ├─► MOVE_FILE → folder_manager.move(src, dest)
    │   │   └─► UPDATE_FILE → write_frontmatter(file, db_meta)
    │   │
    │   └─► Commit transaction
    │       (Rollback on any failure)
    │
    └─► Return SyncReport(
            scanned: int,
            imported: int,
            updated: int,
            conflicts: int,
            warnings: List[str],
            errors: List[str]
        )
    │
    ▼
Display sync report
```

---

## Transaction Management

### Initialization Transactions

**Goal**: Rollback all changes if any step fails

**Implementation**:
```python
def initialize_project(self):
    db_created = False
    folders_created = False

    try:
        # Step 1: Database
        if not db_exists():
            init_database()
            db_created = True

        # Step 2: Folders
        if not folders_exist():
            create_folder_structure()
            folders_created = True

        # Step 3: Config (non-critical)
        try:
            create_default_config()
        except Exception as e:
            logger.warning(f"Config creation failed: {e}")

        # Step 4: Templates (non-critical)
        try:
            install_templates()
        except Exception as e:
            logger.warning(f"Template installation failed: {e}")

        return InitResult(...)

    except Exception as e:
        # Rollback
        if folders_created:
            remove_folder_structure()
        if db_created:
            remove_database()
        raise InitializationError(f"Init failed: {e}")
```

**Critical vs Non-Critical**:
- **Critical** (rollback on failure): Database, Folders
- **Non-Critical** (warn but continue): Config, Templates

---

### Sync Transactions

**Goal**: All-or-nothing database updates; file moves are best-effort

**Implementation**:
```python
def sync_filesystem(self):
    # Phase 1: Scan and analyze (read-only)
    files = scanner.scan_folders(folder_filter)
    actions = []

    for file in files:
        meta = parser.parse_file(file)
        db_task = repo.get_by_key(meta.key)
        comparison = comparator.compare(meta, db_task)
        action = reconciler.reconcile(comparison, strategy)
        actions.append(action)

    if dry_run:
        return SyncReport(preview=True, actions=actions)

    # Phase 2: Apply actions (transactional)
    with session_factory.get_session() as session:
        try:
            # Database actions (in transaction)
            for action in actions:
                if action.type == ActionType.UPDATE_DATABASE:
                    repo.update(action.task_id, action.changes)
                elif action.type == ActionType.CREATE_TASK:
                    repo.create(action.task_data)

            # Commit DB changes
            session.commit()

            # File actions (best-effort, outside transaction)
            for action in actions:
                if action.type == ActionType.MOVE_FILE:
                    try:
                        folder_manager.move(action.src, action.dest)
                    except Exception as e:
                        logger.warning(f"File move failed: {e}")

        except Exception as e:
            session.rollback()
            raise SyncError(f"Sync failed: {e}")

    return SyncReport(...)
```

**Transaction Scope**:
- **In Transaction**: All DB operations (CREATE, UPDATE)
- **Outside Transaction**: File moves (log failures, don't rollback DB)

**Rationale**: File operations can fail independently (permissions, disk full). DB should remain consistent even if file moves fail.

---

## Integration Points

### With E04-F01 (Database Schema)

**Used Components**:
- `init_database()` - Create schema during `pm init`
- `TaskRepository.create()` - Bulk insert during sync
- `TaskRepository.update()` - Update tasks from file changes
- `TaskRepository.get_by_key()` - Query tasks during sync
- `TaskHistoryRepository.create()` - Record sync changes
- `FeatureRepository.create()` - Auto-create missing features

**Transaction Pattern**:
```python
from pm.database import SessionFactory, TaskRepository

with SessionFactory().get_session() as session:
    repo = TaskRepository(session)
    # Perform operations
    # Auto-commit on success, rollback on exception
```

---

### With E04-F05 (Folder Management)

**Used Components**:
- `create_folder_structure()` - Create sync folders during init
- `TransactionalFileOperation` - Move files during sync
- `get_folder_for_status()` - Determine target folder from status

**Integration Pattern**:
```python
from pm.folder_management import FolderManager

folder_mgr = FolderManager()

# During init
folder_mgr.create_folder_structure()

# During sync
if file_status != folder_location:
    target = folder_mgr.get_folder_for_status(file_status)
    folder_mgr.move(current_path, target / filename)
```

---

### With E04-F02 (CLI Framework)

**Command Registration**:
```python
@cli.command()
@click.option('--non-interactive', is_flag=True)
@click.option('--force', is_flag=True)
def init(non_interactive: bool, force: bool):
    """Initialize PM CLI infrastructure"""
    init_service = InitService()
    result = init_service.initialize_project(
        non_interactive=non_interactive,
        force=force
    )
    display_init_result(result)

@cli.command()
@click.option('--folder', type=str, default=None)
@click.option('--strategy', type=click.Choice(['file-wins', 'database-wins', 'newer-wins']))
@click.option('--dry-run', is_flag=True)
@click.option('--create-missing', is_flag=True)
@click.option('--json', is_flag=True)
def sync(folder, strategy, dry_run, create_missing, json):
    """Sync filesystem with database"""
    sync_service = SyncService()
    report = sync_service.sync_filesystem(
        folder=folder,
        strategy=ConflictStrategy[strategy.upper().replace('-', '_')],
        dry_run=dry_run,
        create_missing=create_missing
    )
    if json:
        print(report.to_json())
    else:
        display_sync_report(report)
```

---

## Error Handling Strategy

### Init Errors

| Error Condition | Handling | Exit Code |
|----------------|----------|-----------|
| Database creation fails | Rollback, display error | 2 |
| Folder creation fails | Rollback DB, display error | 2 |
| Config write fails | Warn, continue | 0 |
| Template copy fails | Warn, continue | 0 |
| Already initialized | Skip existing, success | 0 |
| Permission denied | Display error, rollback | 2 |

---

### Sync Errors

| Error Condition | Handling | Exit Code |
|----------------|----------|-----------|
| Invalid YAML | Log warning, skip file | 0 |
| Missing required fields | Log warning, skip file | 0 |
| Key mismatch | Log warning, skip file | 0 |
| Non-existent feature | Log warning, skip (or create if --create-missing) | 0 |
| DB constraint violation | Rollback, display error | 2 |
| File permission error | Rollback, display error | 2 |
| Disk full | Rollback, display error | 2 |

**Principle**: Parsing errors are warnings (skip file, continue). Database/filesystem errors are fatal (rollback, exit).

---

## Deployment Considerations

### Database Location

Default: `./project.db` (current working directory)

Override: `PM_DATABASE_PATH` environment variable

**Init Behavior**:
- Check if file exists at path
- If exists and valid: skip creation
- If exists but corrupt: error, suggest recovery
- If not exists: create new

---

### Config Location

Default: `./.pmconfig.json` (current working directory)

**Init Behavior**:
- Check if file exists
- If exists and `--non-interactive`: skip
- If exists and interactive: prompt "Overwrite? (y/N)"
- If exists and `--force`: overwrite
- If not exists: create

---

### Template Location

Source: `pm/templates/` (package resources)
Destination: `./templates/` (project directory)

**Init Behavior**:
- Copy all `.md` templates from package to project
- If destination exists and `--force`: overwrite
- If destination exists and not `--force`: skip

---

## Performance Considerations

### Sync Performance

**Target**: Process 100 files in <10 seconds

**Optimizations**:
1. **Batch DB Queries**: Fetch all tasks in one query, index by key
2. **YAML Parsing**: Use fast PyYAML C bindings
3. **File I/O**: Minimize file reads (parse once per file)
4. **Bulk Inserts**: Use SQLAlchemy bulk operations for new tasks
5. **Transaction Batching**: Single transaction for all DB changes

**Benchmark**:
```python
# Good: Single query with indexing
all_tasks = {task.key: task for task in repo.list_all()}
for file_meta in file_metas:
    db_task = all_tasks.get(file_meta.key)  # O(1) lookup

# Bad: Query per file
for file_meta in file_metas:
    db_task = repo.get_by_key(file_meta.key)  # O(n) queries
```

---

### Init Performance

**Target**: Complete in <5 seconds

**Operations**:
- Database creation: <500ms (E04-F01 target)
- Folder creation: <100ms (mkdir operations)
- Config write: <10ms (JSON serialization)
- Template copy: <100ms (copy 3 files)

**Total**: ~700ms + overhead = <5 seconds

---

## Security Considerations

See [Security Design](./06-security-design.md) for complete details.

**Key Concerns**:
1. **Path Traversal**: Validate file paths during sync
2. **YAML Injection**: Use safe YAML loader
3. **File Permissions**: Set 600 on database, 644 on config
4. **Input Validation**: Validate all frontmatter fields

---

## Testing Strategy

See [Test Criteria](./09-test-criteria.md) for complete test specifications.

**Unit Tests**:
- InitService: Each operation independently
- SyncService: Each sub-component independently
- ConfigManager: Config read/write
- TemplateManager: Template installation

**Integration Tests**:
- Full `pm init` flow
- Full `pm sync` flow with various scenarios
- Rollback on failure
- Idempotency verification

**Performance Tests**:
- Sync 100 files
- Init with large template set
- Concurrent sync operations

---

## Future Enhancements

**Out of Scope for E04-F07**:
1. **Continuous Sync**: File watching and auto-sync
2. **Interactive Conflict Resolution**: Prompt per conflict
3. **Bidirectional Sync**: Full DB → file sync
4. **Merge Conflict Resolution**: Handle Git merge conflicts
5. **Schema Migrations**: Auto-upgrade DB schema during sync

**Potential Future Features**:
- `pm sync --watch` for continuous sync
- `pm sync --interactive` for conflict prompts
- `pm export` to export DB to markdown
- `pm validate` to check DB/filesystem consistency

---

## Summary

This architecture provides:
- **Reliable Initialization**: Idempotent, transactional setup
- **Flexible Synchronization**: Multiple strategies, dry-run mode
- **Strong Integration**: Leverages E04-F01, E04-F05, E04-F02
- **Error Resilience**: Comprehensive rollback and error handling
- **Performance**: Meets <5s init, <10s sync targets

**Next Steps**:
1. Review architecture with tech lead
2. Create data design document
3. Implement backend components
4. Write comprehensive tests
