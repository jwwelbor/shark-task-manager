# Architecture Document: E04-F05 File Path Management

**Feature**: E04-F05-file-path-management
**Epic**: E04-task-mgmt-cli-core
**Status**: Design
**Last Updated**: 2025-12-15

---

## 1. Overview

This document defines the architecture for feature-based file path management in the task management CLI. The system provides utilities for generating file paths, creating task directories, and validating path consistency between the database and filesystem.

### 1.1 Design Principles

**Feature-Based Organization**: Tasks live in `docs/plan/{epic}/{feature}/tasks/` alongside their PRD and architecture documents for context locality.

**Database as Source of Truth**: Task status and metadata live in the database. File paths are recorded but never change based on status.

**Simplicity**: No file movement operations, no complex synchronization, no rollback logic. Path generation is deterministic and fast.

**Cross-Platform**: All path operations use `pathlib.Path` for transparent handling of platform differences.

---

## 2. Architecture Decisions

### ADR-001: Feature-Based File Organization

**Decision**: Task files are organized under their feature: `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`

**Rationale**:
- Context locality: Tasks live next to PRDs, architecture docs, and other feature artifacts
- Predictability: File location is determined by feature structure, not runtime state
- Simplicity: No file movement means no synchronization, no race conditions, no rollback logic
- Performance: Status updates are 80-90% faster (10ms vs 70-90ms) - database-only operation
- Agent efficiency: Reading task + PRD + architecture requires fewer filesystem navigation operations

**Example Structure**:
```
docs/plan/
├── E04-task-mgmt-cli-core/
│   ├── epic.md
│   ├── F01-database-schema/
│   │   ├── prd.md
│   │   ├── 02-architecture.md
│   │   ├── 03-data-design.md
│   │   └── tasks/
│   │       ├── T-E04-F01-001.md
│   │       ├── T-E04-F01-002.md
│   │       └── T-E04-F01-003.md
│   └── F02-cli-infrastructure/
│       └── tasks/
│           ├── T-E04-F02-001.md
│           └── T-E04-F02-002.md
```

**Trade-offs**:
- Cannot use `ls docs/tasks/todo/` to see todo tasks (must use `shark task list --status=todo`)
- But: Database queries are faster and more capable than filesystem scans

---

### ADR-002: Static File Paths (No Movement)

**Decision**: Task files are created in their feature folder and never move, regardless of status changes.

**Rationale**:
- Eliminates entire classes of errors: permission denied, disk full, file conflicts, race conditions
- Dramatically simplifies implementation: no file locking, no compensating transactions, no rollback
- Improves performance: status updates are pure database operations
- Reduces complexity: no validation/repair logic for file movement
- Better for Git: files don't move around, cleaner diffs

**Status Tracking**:
```python
# Status is tracked purely in database
def update_task_status(task_key: str, new_status: str):
    with db.transaction():
        task = task_repo.get_by_key(task_key)
        task.status = new_status
        if new_status == 'in_progress' and not task.started_at:
            task.started_at = datetime.utcnow()
        db.commit()  # <10ms, no file operations
```

**Trade-offs**:
- No visual indication of status via folder location
- But: `shark task list --status=X` provides instant filtered views

---

### ADR-003: Validation Without Auto-Repair

**Decision**: `shark validate` detects inconsistencies but does not automatically fix them.

**Rationale**:
- Missing files likely indicate larger issues (accidental deletion, corrupted sync)
- Automatic file creation could mask underlying problems
- Manual intervention ensures developer understands what went wrong
- Simplifies implementation: validation is read-only

**Validation Checks**:
1. Database `file_path` matches expected pattern
2. File exists at recorded path
3. File is readable

**Manual Repair Process**:
```bash
# User discovers missing file
shark validate  # Reports: "T-E04-F01-001: missing file"

# User investigates (Git history, backups, etc.)
git log -- docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/T-E04-F01-001.md

# User recovers file manually or recreates it
shark task create E04-F01 "Database Foundation" --key=T-E04-F01-001

# Verify fix
shark validate  # Reports: "All tasks valid"
```

---

## 3. System Architecture

### 3.1 Component Diagram

```
┌─────────────────────────────────────────┐
│         CLI Commands (E04-F02)          │
│   shark task create/get/list               │
│   shark validate                           │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│   Task Operations (E04-F03)             │
│   Task Creation (E04-F06)               │
│  - create_task()                        │
│  - get_task()                           │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│   File Path Service (THIS)              │
│  - get_task_file_path()                 │
│  - create_task_directory()              │
│  - validate_file_paths()                │
└──────────┬───────────────┬──────────────┘
           │               │
┌──────────▼──────┐  ┌─────▼──────────────┐
│ pathlib.Path    │  │ Task Repository    │
│ (Python stdlib) │  │ (E04-F01)          │
└─────────────────┘  └────────────────────┘
```

### 3.2 File Path Pattern

**Pattern**: `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`

**Components**:
- **Project Root**: Detected by searching for `.git/` or `pyproject.toml`
- **Epic Key**: Matches epic directory (e.g., `E04-task-mgmt-cli-core`)
- **Feature Key**: Matches feature directory (e.g., `F01-database-schema`)
- **Task Key**: Matches task identifier (e.g., `T-E04-F01-001`)

**Full Example**:
```
/home/jwwelbor/projects/ai-dev-team/docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/T-E04-F01-001.md
```

---

## 4. API Specification

### 4.1 Path Generation

#### `get_task_file_path(epic_key: str, feature_key: str, task_key: str) -> Path`

Generate the absolute file path for a task.

**Parameters**:
- `epic_key`: Epic identifier (e.g., "E04-task-mgmt-cli-core")
- `feature_key`: Feature identifier (e.g., "F01-database-schema")
- `task_key`: Task identifier (e.g., "T-E04-F01-001")

**Returns**: Absolute `Path` object

**Raises**:
- `ProjectRootNotFound`: Cannot determine project root
- `InvalidTaskKey`: Task key doesn't match expected pattern

**Example**:
```python
>>> path = get_task_file_path(
...     "E04-task-mgmt-cli-core",
...     "F01-database-schema",
...     "T-E04-F01-001"
... )
>>> print(path)
/home/jwwelbor/projects/ai-dev-team/docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/T-E04-F01-001.md
>>> path.exists()
True
```

**Implementation**:
```python
from pathlib import Path

def get_task_file_path(epic_key: str, feature_key: str, task_key: str) -> Path:
    """Generate absolute file path for a task."""
    # Validate task key format
    if not _is_valid_task_key(task_key):
        raise InvalidTaskKey(f"Invalid task key: {task_key}. Expected format: T-{epic}-{feature}-{seq}")

    # Find project root
    project_root = find_project_root()

    # Construct path
    return project_root / "docs" / "plan" / epic_key / feature_key / "tasks" / f"{task_key}.md"

def find_project_root() -> Path:
    """Find project root by searching for .git or pyproject.toml."""
    current = Path.cwd()
    while current != current.parent:
        if (current / ".git").exists() or (current / "pyproject.toml").exists():
            return current
        current = current.parent
    raise ProjectRootNotFound("Cannot find project root (.git or pyproject.toml not found)")
```

**Performance**: <1ms (simple path construction + filesystem check)

---

### 4.2 Directory Creation

#### `create_task_directory(epic_key: str, feature_key: str) -> Path`

Create the tasks directory for a feature (if it doesn't exist).

**Parameters**:
- `epic_key`: Epic identifier
- `feature_key`: Feature identifier

**Returns**: Absolute `Path` to created/existing tasks directory

**Raises**:
- `PermissionError`: Insufficient permissions to create directory
- `ProjectRootNotFound`: Cannot determine project root

**Example**:
```python
>>> tasks_dir = create_task_directory("E04-task-mgmt-cli-core", "F01-database-schema")
>>> print(tasks_dir)
/home/jwwelbor/projects/ai-dev-team/docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks
>>> tasks_dir.exists()
True
```

**Implementation**:
```python
def create_task_directory(epic_key: str, feature_key: str) -> Path:
    """Create tasks directory for a feature (idempotent)."""
    project_root = find_project_root()
    tasks_dir = project_root / "docs" / "plan" / epic_key / feature_key / "tasks"

    # Create directory and all parents (mkdir -p)
    tasks_dir.mkdir(parents=True, exist_ok=True)

    return tasks_dir
```

**Performance**: <10ms (filesystem operation)

---

### 4.3 Validation

#### `validate_file_paths() -> ValidationResult`

Validate that all task file paths in database match filesystem reality.

**Returns**: `ValidationResult` dataclass:
```python
@dataclass
class ValidationResult:
    total_tasks: int
    valid: int
    missing_files: List[TaskValidation]
    invalid_paths: List[TaskValidation]

@dataclass
class TaskValidation:
    task_key: str
    expected_path: Path
    actual_path: Optional[Path]
    issue: str  # Human-readable description
```

**Example**:
```python
>>> result = validate_file_paths()
>>> print(f"{result.valid}/{result.total_tasks} tasks valid")
95/100 tasks valid
>>> for issue in result.missing_files:
...     print(f"Missing: {issue.task_key} at {issue.expected_path}")
Missing: T-E04-F01-002 at /home/user/project/docs/plan/.../T-E04-F01-002.md
```

**Implementation**:
```python
def validate_file_paths() -> ValidationResult:
    """Validate all task file paths match database records."""
    tasks = task_repo.get_all()

    valid = []
    missing_files = []
    invalid_paths = []

    for task in tasks:
        # Generate expected path
        expected = get_task_file_path(task.epic.key, task.feature.key, task.key)

        # Check if database path matches expected
        db_path = Path(task.file_path) if task.file_path else None

        if db_path != expected:
            invalid_paths.append(TaskValidation(
                task_key=task.key,
                expected_path=expected,
                actual_path=db_path,
                issue=f"Database path {db_path} doesn't match expected {expected}"
            ))
            continue

        # Check if file exists
        if not expected.exists():
            missing_files.append(TaskValidation(
                task_key=task.key,
                expected_path=expected,
                actual_path=None,
                issue=f"File does not exist at {expected}"
            ))
            continue

        # Check if file is readable
        try:
            expected.read_text()
            valid.append(task.key)
        except PermissionError:
            missing_files.append(TaskValidation(
                task_key=task.key,
                expected_path=expected,
                actual_path=expected,
                issue=f"File exists but is not readable at {expected}"
            ))

    return ValidationResult(
        total_tasks=len(tasks),
        valid=len(valid),
        missing_files=missing_files,
        invalid_paths=invalid_paths
    )
```

**Performance**: <2 seconds for 1,000 tasks

---

## 5. Data Model

### 5.1 Database Schema (E04-F01)

**No schema changes required**. Uses existing `tasks` table:

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    file_path TEXT,  -- Stores absolute path, never changes after creation
    -- ... other fields
    FOREIGN KEY (feature_id) REFERENCES features(id)
);
```

**File Path Invariant**: Once set during task creation, `file_path` never changes.

---

## 6. Integration Points

### 6.1 Integration with E04-F06 (Task Creation)

**Task creation must set file_path correctly on first creation.**

```python
# E04-F06: task_creation.py
from file_path_service import get_task_file_path, create_task_directory

def pm_task_create(feature_key: str, title: str, epic_key: str) -> Task:
    """Create a new task."""
    # Generate next task key
    task_key = generate_next_task_key(epic_key, feature_key)

    # Generate file path
    file_path = get_task_file_path(epic_key, feature_key, task_key)

    # Create database record
    task = Task(
        feature_key=feature_key,
        key=task_key,
        title=title,
        status="todo",
        file_path=str(file_path)  # Store absolute path
    )
    task_repo.save(task)

    # Create tasks directory if needed
    create_task_directory(epic_key, feature_key)

    # Create task file with template
    create_task_file(file_path, task)  # E04-F06 responsibility

    return task
```

---

### 6.2 Integration with E04-F03 (Task Operations)

**Task operations DO NOT change file_path.**

```python
# E04-F03: task_operations.py

def pm_task_start(task_key: str):
    """Start a task (change status to in_progress)."""
    with db.transaction():
        task = task_repo.get_by_key(task_key)

        # Validate state transition
        if task.status != "todo":
            raise InvalidStateTransition(f"Cannot start task with status {task.status}")

        # Update status (NO file operations)
        task.status = "in_progress"
        task.started_at = datetime.utcnow()

        db.commit()  # Fast: <10ms

def pm_task_complete(task_key: str):
    """Complete a task (change status to completed)."""
    with db.transaction():
        task = task_repo.get_by_key(task_key)

        # Validate state transition
        if task.status not in ["in_progress", "ready_for_review"]:
            raise InvalidStateTransition(f"Cannot complete task with status {task.status}")

        # Update status (NO file operations)
        task.status = "completed"
        task.completed_at = datetime.utcnow()

        db.commit()  # Fast: <10ms
```

**Performance Impact**: Status updates are 80-90% faster than old approach (10ms vs 70-90ms).

---

### 6.3 Integration with E04-F07 (Sync/Import)

**After importing tasks, validate file paths.**

```python
# E04-F07: sync.py
from file_path_service import validate_file_paths

def pm_sync():
    """Import existing task files into database."""
    # Scan filesystem for task files
    task_files = find_all_task_files("docs/plan/")

    # Import each task
    for task_file in task_files:
        # Parse frontmatter
        task_data = parse_task_frontmatter(task_file)

        # Create database record with correct file_path
        task = Task(
            key=task_data.key,
            title=task_data.title,
            status=task_data.status,
            file_path=str(task_file.absolute())  # Store absolute path
        )
        task_repo.save(task)

    # Validate all paths after import
    result = validate_file_paths()

    if result.missing_files or result.invalid_paths:
        print(f"⚠️  Found {len(result.missing_files + result.invalid_paths)} issues")
        print("Run 'shark validate' for details")
    else:
        print(f"✅ All {result.total_tasks} tasks valid")
```

---

## 7. Error Handling

### 7.1 Error Scenarios

#### Scenario 1: Project Root Not Found

```python
# User runs command outside project directory
shark task create ...

# Error:
ProjectRootNotFound: Cannot find project root (.git or pyproject.toml not found)
Please run this command from within your project directory.
```

**Recovery**: User changes to project directory.

---

#### Scenario 2: Invalid Task Key

```python
# Invalid task key format
get_task_file_path("E04", "F01", "invalid-key")

# Error:
InvalidTaskKey: Invalid task key: invalid-key
Expected format: T-{epic}-{feature}-{sequence} (e.g., T-E04-F01-001)
```

**Recovery**: Use correct task key format.

---

#### Scenario 3: Permission Denied Creating Directory

```python
# Insufficient permissions
create_task_directory("E04", "F01")

# Error:
PermissionError: Permission denied: cannot create directory
/home/user/project/docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/
Please check filesystem permissions.
```

**Recovery**: Fix filesystem permissions.

---

#### Scenario 4: Missing File Detected

```python
# Database has path, file doesn't exist
result = validate_file_paths()

# Result:
ValidationResult(
    total_tasks=100,
    valid=99,
    missing_files=[
        TaskValidation(
            task_key="T-E04-F01-002",
            expected_path=Path("/home/user/.../T-E04-F01-002.md"),
            actual_path=None,
            issue="File does not exist at /home/user/.../T-E04-F01-002.md"
        )
    ],
    invalid_paths=[]
)
```

**Recovery**: Investigate (check Git history, backups), then manually recreate file or update database.

---

### 7.2 Error Classes

```python
class FilePathError(Exception):
    """Base exception for file path operations"""
    pass

class ProjectRootNotFound(FilePathError):
    """Cannot determine project root"""
    pass

class InvalidTaskKey(FilePathError):
    """Task key doesn't match expected format"""
    def __init__(self, task_key: str):
        super().__init__(
            f"Invalid task key: {task_key}. "
            f"Expected format: T-{{epic}}-{{feature}}-{{sequence}} (e.g., T-E04-F01-001)"
        )
```

---

## 8. Testing Strategy

### 8.1 Unit Tests

**Path Generation** (`test_file_path.py`):
```python
def test_get_task_file_path_returns_correct_path():
    path = get_task_file_path("E04-task-mgmt-cli-core", "F01-database-schema", "T-E04-F01-001")
    assert path.name == "T-E04-F01-001.md"
    assert "E04-task-mgmt-cli-core" in str(path)
    assert "F01-database-schema" in str(path)
    assert "tasks" in str(path)
    assert path.is_absolute()

def test_get_task_file_path_raises_on_invalid_key():
    with pytest.raises(InvalidTaskKey):
        get_task_file_path("E04", "F01", "invalid-key")

def test_find_project_root_finds_git_directory(tmp_path):
    (tmp_path / ".git").mkdir()
    os.chdir(tmp_path)
    root = find_project_root()
    assert root == tmp_path
```

---

**Directory Creation** (`test_directory_creation.py`):
```python
def test_create_task_directory_creates_all_parents(tmp_path):
    # Mock project root
    with mock_project_root(tmp_path):
        tasks_dir = create_task_directory("E04-task-mgmt-cli-core", "F01-database-schema")

        assert tasks_dir.exists()
        assert tasks_dir.is_dir()
        assert tasks_dir.name == "tasks"
        assert tasks_dir.parent.name == "F01-database-schema"

def test_create_task_directory_is_idempotent(tmp_path):
    with mock_project_root(tmp_path):
        # Create once
        dir1 = create_task_directory("E04", "F01")

        # Create again
        dir2 = create_task_directory("E04", "F01")

        # Same directory returned
        assert dir1 == dir2
        assert dir1.exists()
```

---

**Validation** (`test_validation.py`):
```python
def test_validate_detects_missing_file(db_session, tmp_path):
    with mock_project_root(tmp_path):
        # Create task in DB with file_path
        task = create_task(
            key="T-E04-F01-001",
            file_path=str(tmp_path / "docs" / "plan" / "E04" / "F01" / "tasks" / "T-E04-F01-001.md")
        )

        # Don't create the file

        # Validate
        result = validate_file_paths()

        assert result.total_tasks == 1
        assert result.valid == 0
        assert len(result.missing_files) == 1
        assert result.missing_files[0].task_key == "T-E04-F01-001"

def test_validate_detects_invalid_path(db_session):
    # Create task with wrong path pattern
    task = create_task(
        key="T-E04-F01-001",
        file_path="/wrong/location/T-E04-F01-001.md"
    )

    result = validate_file_paths()

    assert len(result.invalid_paths) == 1
    assert result.invalid_paths[0].task_key == "T-E04-F01-001"
```

---

### 8.2 Integration Tests

**CLI Commands** (`test_cli_integration.py`):
```python
def test_pm_validate_reports_all_valid(cli_runner, db_session):
    # Create tasks with valid paths and files
    for i in range(10):
        task = create_task(key=f"T-E04-F01-{i:03d}")
        create_file(task.file_path)

    # Run validation
    result = cli_runner.invoke(["validate"])

    assert result.exit_code == 0
    assert "All 10 tasks valid" in result.output

def test_pm_validate_reports_missing_files(cli_runner, db_session):
    # Create task without file
    task = create_task(key="T-E04-F01-001")

    result = cli_runner.invoke(["validate"])

    assert result.exit_code == 1
    assert "missing" in result.output.lower()
    assert "T-E04-F01-001" in result.output
```

---

### 8.3 Cross-Platform Tests

```python
@pytest.mark.skipif(sys.platform != "win32", reason="Windows-specific")
def test_handles_windows_paths():
    """Ensure pathlib handles Windows paths correctly"""
    path = get_task_file_path("E04", "F01", "T-E04-F01-001")
    assert isinstance(path, Path)
    # pathlib automatically uses backslashes on Windows

@pytest.mark.skipif(sys.platform != "linux", reason="Linux-specific")
def test_handles_case_sensitive_filesystem():
    """Ensure paths are case-sensitive on Linux"""
    path1 = get_task_file_path("E04", "F01", "T-E04-F01-001")
    path2 = get_task_file_path("E04", "F01", "t-e04-f01-001")  # Different case
    assert path1 != path2
```

---

## 9. Performance Considerations

### 9.1 Path Generation Performance

**Requirement**: <1ms per call

**Implementation**: Pure string/path construction, no I/O except one-time project root discovery (cached).

```python
# Cache project root to avoid repeated filesystem searches
_project_root_cache: Optional[Path] = None

def find_project_root() -> Path:
    global _project_root_cache
    if _project_root_cache is None:
        _project_root_cache = _find_project_root_uncached()
    return _project_root_cache
```

---

### 9.2 Validation Performance

**Requirement**: <2 seconds for 1,000 tasks

**Optimized Approach**:
```python
def validate_file_paths() -> ValidationResult:
    """Efficient validation using batch queries and minimal I/O."""
    # 1. Single database query for all tasks (<100ms)
    tasks = task_repo.get_all()

    # 2. Check file existence (1000 stat calls ~500ms on SSD)
    for task in tasks:
        expected = get_task_file_path(task.epic.key, task.feature.key, task.key)
        exists = expected.exists()  # Single stat call
        # ... validation logic

    # Total: <1000ms for 1,000 tasks
```

**No Optimizations Needed**: Standard library is sufficient.

---

## 10. Implementation Checklist

### Phase 1: Core Path Functions
- [ ] Implement `find_project_root()`
- [ ] Implement `get_task_file_path()`
- [ ] Add task key validation
- [ ] Add unit tests
- [ ] Handle cross-platform paths

### Phase 2: Directory Management
- [ ] Implement `create_task_directory()`
- [ ] Add idempotency tests
- [ ] Handle permission errors
- [ ] Add integration tests

### Phase 3: Validation
- [ ] Implement `validate_file_paths()`
- [ ] Define `ValidationResult` dataclass
- [ ] Add validation tests
- [ ] Handle edge cases (missing files, invalid paths)

### Phase 4: CLI Integration
- [ ] Add `shark validate` command
- [ ] Format validation output
- [ ] Set appropriate exit codes
- [ ] Add CLI integration tests

### Phase 5: Documentation
- [ ] Add user documentation
- [ ] Document error recovery procedures
- [ ] Add examples to CLI help text

---

## 11. Success Criteria

**Functional**:
- ✅ All task files live in feature/tasks/ directories
- ✅ File paths are deterministic based on epic/feature/task keys
- ✅ Validation detects missing files and invalid paths
- ✅ Cross-platform path handling works on Linux, macOS, Windows

**Performance**:
- ✅ Path generation completes in <1ms
- ✅ Directory creation completes in <10ms
- ✅ Validation of 1,000 tasks completes in <2s

**Simplicity**:
- ✅ No file movement operations
- ✅ No file locking or synchronization logic
- ✅ No rollback or compensating transactions
- ✅ Status updates are pure database operations (<10ms)

---

**Document Status**: Ready for Task Generation
**Next Step**: Generate implementation tasks from this architecture
