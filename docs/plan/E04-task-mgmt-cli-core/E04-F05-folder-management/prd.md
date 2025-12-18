# Feature: File Path Management

## Epic

- [Epic PRD](/home/jwwelbor/projects/ai-dev-team/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

The Shark CLI maintains task metadata in a database and task markdown files in the filesystem. The database needs to store the correct file path for each task, and the system must ensure tasks are organized in a consistent, logical structure. Without a standardized file organization strategy, tasks become scattered across arbitrary locations, making it difficult for both humans and AI agents to find related tasks and understand feature context. File paths stored in the database can drift from actual file locations due to manual file moves, filesystem errors, or bugs. The system needs a simple, predictable file organization pattern that keeps tasks close to their feature's design documents (PRD, architecture, etc.) for maximum context locality.

### Solution

Implement a feature-based file organization system where task files live alongside their feature's documentation in `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`. Task files never move - they are created in their feature folder and stay there regardless of status changes. The database `file_path` field stores the absolute path to each task file. Provide utility functions to generate correct file paths based on feature location, create task directories automatically, and validate that database file paths match actual file locations. All file path operations use `pathlib` for cross-platform compatibility.

### Impact

- **Context Locality**: Tasks live next to their PRDs and architecture docs, reducing token usage for agents reading related documents
- **Simplicity**: No file movement on status changes eliminates entire classes of errors (permissions, disk full, race conditions)
- **Performance**: Status updates are 80-90% faster (10ms vs 70-90ms) since they only update the database
- **Predictability**: File paths are deterministic based on epic/feature structure
- **Foundation for Operations**: Provides the file path logic required by E04-F03 (Task Operations) and E04-F06 (Task Creation)

## User Personas

### Primary Persona: Backend Developer (implementing task operations)

**Role**: Developer building CLI commands that need to locate task files
**Environment**: Python 3.10+, cross-platform (Linux, macOS, Windows)

**Key Characteristics**:
- Needs reliable file path generation based on task metadata
- Must handle filesystem operations gracefully
- Requires cross-platform path handling
- Values simplicity and clarity

**Goals**:
- Call `get_task_file_path(task)` and get the correct absolute path
- Create task files in the right location using `create_task_directory(feature)`
- Validate file paths match database with `validate_file_paths()`
- Never worry about file movement or synchronization

**Pain Points this Feature Addresses**:
- No standard for where task files should be created
- Manual path construction is error-prone
- Cross-platform path handling is complex
- No way to detect file path inconsistencies

### Secondary Persona: AI Agent (indirect user through CLI)

**Role**: Executes CLI commands that create or read task files
**Environment**: Claude Code CLI, needs efficient file access

**Key Characteristics**:
- Reads task files and related feature documentation together
- Benefits from context locality (task + PRD + architecture in same directory)
- Needs fast file path resolution
- Appreciates predictable file organization

**Goals**:
- Find all tasks for a feature in one directory scan
- Read task file and related PRD without navigating the filesystem
- Trust that file paths are always correct
- Never encounter missing files or broken paths

**Pain Points this Feature Addresses**:
- Tasks scattered across different folders makes context gathering slow
- Unpredictable file organization wastes tokens on filesystem exploration
- No guarantee that database file_path is correct

### Tertiary Persona: Product Manager / Technical Lead

**Role**: Human developer managing projects
**Environment**: Terminal, filesystem browser, Git

**Key Characteristics**:
- Uses folders to navigate project structure
- Expects logical organization (epic → feature → tasks)
- Wants to find all artifacts for a feature in one place
- Values Git-friendly structure (clear feature boundaries)

**Goals**:
- Navigate to feature folder and see all tasks
- See task files next to PRD and architecture docs
- Trust that project structure is consistent
- Use Git to track feature changes (including tasks)

**Pain Points this Feature Addresses**:
- Tasks separated from feature documentation
- No standard project structure
- Files in unexpected locations
- Manual directory creation is tedious

## User Stories

### Must-Have User Stories

**Story 1: Generate Correct File Path**
- As a backend developer, I want to call `get_task_file_path(task)` and receive the correct absolute path based on the task's feature, so that I can create or read task files reliably.

**Story 2: Create Task Directory Automatically**
- As a backend developer, I want the system to automatically create `docs/plan/{epic}/{feature}/tasks/` when I create a task, so that I don't manually manage directories.

**Story 3: Validate File Path Consistency**
- As a user, I want to run `pm validate` to check if all task files exist at their database-recorded paths, so that I can detect inconsistencies.

**Story 4: Cross-Platform Paths**
- As a user on Windows, macOS, or Linux, I want all file path operations to work correctly, so that the tool is portable.

**Story 5: Feature-Based Organization**
- As a user, I want all tasks for a feature to live in `docs/plan/{epic}/{feature}/tasks/`, so that tasks are organized logically with their related documentation.

**Story 6: Predictable File Names**
- As a user, I want task files named using the pattern `{task-key}.md`, so that file names are unambiguous and match task identifiers.

### Should-Have User Stories

**Story 7: Detect Missing Files**
- As a user, I want `pm validate` to report tasks with missing files, so that I can identify and fix filesystem issues.

**Story 8: Handle Path Validation Errors**
- As a backend developer, I want clear exceptions if a file path is invalid or inaccessible, so that I can handle edge cases properly.

**Story 9: Support Absolute and Relative Paths**
- As a backend developer, I want file path functions to support both absolute and relative paths, so that I can use them flexibly.

### Could-Have User Stories

**Story 10: Auto-Repair Missing Files**
- As a user, I want `pm validate --repair` to offer to recreate missing task files from database metadata, so that I can recover from accidental deletions.

**Story 11: Warn About Long Paths**
- As a user on Windows, I want warnings if file paths exceed 260 characters (MAX_PATH), so that I can avoid path length issues.

## Requirements

### Functional Requirements

**File Path Pattern:**

1. Task files must follow the pattern: `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`
   - Example: `docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/T-E04-F01-001.md`

2. Epic key must match the epic directory name (e.g., `E04-task-mgmt-cli-core`)

3. Feature key must match the feature directory name (e.g., `F01-database-schema`)

4. Task key must match the task identifier (e.g., `T-E04-F01-001`)

5. File extension must always be `.md`

**Path Generation:**

6. The system must provide function `get_task_file_path(epic_key, feature_key, task_key) -> Path` that returns the absolute path for a task file

7. The function must use `pathlib.Path` for cross-platform compatibility

8. The function must return absolute paths (resolve relative to project root)

9. The system must detect project root by searching for `.git/` directory or `pyproject.toml` file

10. If project root cannot be determined, raise `ProjectRootNotFound` exception

**Directory Creation:**

11. The system must provide function `create_task_directory(epic_key, feature_key) -> Path` that creates the tasks directory if it doesn't exist

12. The function must create parent directories as needed (`mkdir -p` behavior)

13. The function must be idempotent (no error if directory already exists)

14. The function must return the created directory path

15. Directory creation must handle permission errors with clear exceptions

**Path Validation:**

16. The system must provide `validate_file_paths() -> ValidationResult` that checks all tasks:
    - Database `file_path` field matches expected pattern
    - File exists at the recorded path
    - File is readable

17. Validation must return `ValidationResult` with:
    - `total_tasks`: Total tasks checked
    - `valid`: Tasks with correct paths and existing files
    - `missing_files`: Tasks where database has path but file doesn't exist
    - `invalid_paths`: Tasks where database path doesn't match expected pattern

18. Validation must not modify any files or database records

19. The system must provide CLI command `pm validate` that runs validation and reports results

20. Exit code must be 0 if all paths valid, 1 if validation found issues

**File Name Validation:**

21. Task keys must be validated before generating file paths

22. Task keys must match pattern: `T-{epic}-{feature}-{sequence}` (e.g., `T-E04-F01-001`)

23. File names must not contain special characters invalid on Windows: `<>:"/\|?*`

24. If task key is invalid, raise `InvalidTaskKey` exception with message explaining the format

**Cross-Platform Compatibility:**

25. All path operations must use `pathlib.Path` (handles platform differences)

26. Path separators must be platform-appropriate (automatic with pathlib)

27. File operations must handle case-sensitive (Linux) and case-insensitive (macOS, Windows) filesystems

28. Long paths on Windows (>260 chars) must generate warnings but not fail

### Non-Functional Requirements

**Performance:**

- Path generation must complete in <1ms
- Directory creation must complete in <10ms
- Validation of 1,000 tasks must complete in <2 seconds
- File existence checks must use efficient filesystem APIs

**Reliability:**

- Path functions must always return valid paths (no invalid characters)
- Directory creation must be atomic (no partial creates)
- Validation must handle filesystem errors gracefully
- Operations must be idempotent (safe to call multiple times)

**Cross-Platform Compatibility:**

- Must work on Linux, macOS, Windows
- Must handle different path separators automatically
- Must handle different filesystem constraints (case sensitivity, path length limits)
- Must handle UNC paths on Windows

**Data Integrity:**

- Database file_path field must always store absolute paths
- Path validation must detect any drift between database and filesystem
- No orphaned files (files without database records) in validation scope

**Error Recovery:**

- Clear error messages for common issues (permission denied, path too long, invalid characters)
- Validation command provides actionable feedback
- System remains consistent even if filesystem operations fail

## Acceptance Criteria

### Path Generation

**Given** a task with epic="E04-task-mgmt-cli-core", feature="F01-database-schema", key="T-E04-F01-001"
**When** I call `get_task_file_path("E04-task-mgmt-cli-core", "F01-database-schema", "T-E04-F01-001")`
**Then** path is `/home/user/project/docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/T-E04-F01-001.md`
**And** path is absolute (starts with project root)
**And** operation completes in <1ms

**Given** an invalid task key "invalid-key"
**When** I call `get_task_file_path(epic, feature, "invalid-key")`
**Then** `InvalidTaskKey` exception is raised
**And** exception message explains the correct format

### Directory Creation

**Given** the directory `docs/plan/E04-task-mgmt-cli-core/F01-database-schema/tasks/` does not exist
**When** I call `create_task_directory("E04-task-mgmt-cli-core", "F01-database-schema")`
**Then** directory is created with all parent directories
**And** function returns the created directory path
**And** operation completes in <10ms

**Given** the directory already exists
**When** I call `create_task_directory("E04-task-mgmt-cli-core", "F01-database-schema")`
**Then** no error occurs (idempotent)
**And** function returns the existing directory path

**Given** I don't have write permissions on parent directory
**When** I call `create_task_directory("E04-task-mgmt-cli-core", "F01-database-schema")`
**Then** `PermissionError` is raised with clear message

### Validation

**Given** database shows task T-E04-F01-001 with file_path="/correct/path/T-E04-F01-001.md"
**And** file exists at that path
**When** I run `pm validate`
**Then** task is reported as valid
**And** exit code is 0

**Given** database shows task T-E04-F01-002 with file_path="/expected/path/T-E04-F01-002.md"
**And** file does NOT exist at that path
**When** I run `pm validate`
**Then** task is reported in "missing_files" list
**And** exit code is 1

**Given** database shows task with file_path="wrong/pattern/file.md" (doesn't match expected pattern)
**When** I run `pm validate`
**Then** task is reported in "invalid_paths" list
**And** exit code is 1

**Given** all 1,000 tasks have correct paths and existing files
**When** I run `pm validate`
**Then** report shows "All 1,000 tasks valid"
**And** validation completes in <2 seconds
**And** exit code is 0

### Cross-Platform Compatibility

**Given** I'm on Windows
**When** I call `get_task_file_path(epic, feature, task_key)`
**Then** path uses Windows path separators (or pathlib handles it transparently)
**And** path is valid on Windows filesystem

**Given** I'm on Linux (case-sensitive)
**When** I create files with different cases (T-E01 vs t-e01)
**Then** both files are distinct and operations work correctly

**Given** I'm on macOS (case-insensitive)
**When** operations reference paths with different cases
**Then** operations work correctly despite case insensitivity

**Given** a path that would exceed 260 characters on Windows
**When** I generate the path
**Then** a warning is logged
**And** path is still returned (may fail on old Windows, but that's a filesystem limitation)

## Out of Scope

### Explicitly NOT Included in This Feature

1. **File Content Management** - This feature generates paths but doesn't read, write, or edit markdown content. Content management is in E04-F06 (Task Creation).

2. **File Movement** - Tasks are created in their feature folder and never move. No file movement operations.

3. **Status-Based Folder Organization** - Status is tracked in database only, not by folder location.

4. **Git Integration** - No automatic git operations (commits, adds, etc.).

5. **File Versioning** - No history of file locations or changes.

6. **Cloud Sync** - No integration with Dropbox, Google Drive, or cloud storage.

7. **File Search** - No full-text search within task files (that's in E05 optional features).

8. **File Permissions Management** - No automatic chmod or chown operations.

9. **Symbolic Links** - No support for symlinks to task files (use real files only).

10. **Auto-Repair** - Validation detects issues but doesn't automatically fix them (manual intervention required).

11. **Orphaned File Detection** - Validation only checks database-recorded tasks, not random files in the filesystem.

12. **File Recovery** - No attempt to recover deleted files from OS trash/recycle bin.
