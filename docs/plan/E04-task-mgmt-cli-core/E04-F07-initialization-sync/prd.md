# Feature: Initialization & Synchronization

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Key Architectural Decisions

**Status Management:**
- Task status is stored ONLY in the database, NOT in file frontmatter
- Status is managed exclusively through the `pm` tool
- File frontmatter contains only: key (required), title (optional), description (optional), file_path (optional)
- When syncing, the system queries the database using the task key to get the current status

**File Organization:**
- Tasks are organized under feature folders: `docs/plan/<epic>/<feature>/T-<key>.md`
- Tasks remain in their feature folders regardless of status changes
- Status changes update the database record only, not file location
- Legacy tasks in status-based folders (docs/tasks/todo/, etc.) are supported for backward compatibility

### Problem

New projects need to set up the PM CLI infrastructure from scratch: create the database schema, set up folder structure, configure defaults, and optionally import existing task markdown files. Existing projects with task files organized under feature folders (e.g., `docs/plan/E04-epic/E04-F06-feature/T-E04-F06-001.md`) or legacy status-based folders need to migrate data into the database without losing information or manually recreating tasks. When markdown files are edited outside the CLI (direct file edits, Git pulls, manual edits to title/description), the database becomes stale and out of sync with filesystem reality. Tasks are organized under their respective feature folders and remain there regardless of status changes - status is managed solely in the database, not in files. Without initialization and sync tools, users must manually set up infrastructure, risk data loss during migration, and have no way to detect or repair database/filesystem inconsistencies caused by external changes.

### Solution

Implement `pm init` command for new project setup that creates database schema, folder structure, default configuration file, and task templates in a single operation. Implement `pm sync` command that scans feature folders recursively for task markdown files, parses frontmatter (key, title, description), queries database for status using task key, imports/updates database records to match file contents, and handles conflicts when database and file metadata disagree. Tasks remain organized under their feature folders regardless of status - status changes only update the database, not file location or frontmatter. Support dry-run mode (`--dry-run`) to preview changes before applying, force mode (`--force`) to overwrite database with file contents, and selective sync (`--folder=<path>`) to sync specific folders. Provide conflict resolution strategies: file-wins (default), database-wins, or newer-wins based on timestamps. Integrate with E04-F01 (Database), E04-F05 (Folder Management), and E04-F02 (CLI Framework) to ensure reliable setup and synchronization.

### Impact

- **Onboarding Speed**: Reduce new project setup from 30+ minutes (manual DB creation, folder setup, template copying) to <30 seconds with `pm init`
- **Migration Safety**: Enable zero-data-loss migration of existing markdown task files to database with `pm sync`
- **Ecosystem Integration**: Support Git-based workflows where files are edited in external editors or merged via Git
- **Consistency Recovery**: Provide automated repair for database/filesystem mismatches caused by external edits
- **Developer Confidence**: Dry-run mode enables safe preview of sync operations before committing changes

## User Personas

### Primary Persona: Product Manager / Technical Lead (New User)

**Role**: Developer setting up PM CLI for the first time on a new or existing project
**Environment**: Terminal, may have existing markdown task files

**Key Characteristics**:
- Wants quick setup with minimal configuration
- May have existing task files to import
- Needs confidence that migration won't lose data
- Prefers automated setup over manual steps

**Goals**:
- Run single command to set up PM CLI: `pm init`
- Import existing task files: `pm sync`
- Verify sync results before committing: `pm sync --dry-run`
- Configure defaults (epic, agent) during init

**Pain Points this Feature Addresses**:
- Manual database schema creation is complex
- No automated way to import existing task files
- Fear of data loss during migration
- Tedious folder setup and configuration

### Secondary Persona: Developer Using Git Workflow

**Role**: Developer working in team where task files are versioned in Git
**Environment**: Git repository, pulling changes from collaborators

**Key Characteristics**:
- Edits task files in text editor (not always via CLI)
- Pulls Git changes that include new/modified task files
- Needs to sync database after Git operations
- Requires conflict resolution when DB and files disagree

**Goals**:
- Sync database after `git pull`: `pm sync`
- Detect conflicts between DB and files
- Choose resolution strategy (file-wins vs database-wins)
- See what will change before syncing: `pm sync --dry-run`

**Pain Points this Feature Addresses**:
- Database becomes stale after Git pulls
- No automated sync after external file changes
- Manual conflict resolution is error-prone
- No visibility into what will change

### Tertiary Persona: AI Agent (Initialization)

**Role**: Agent setting up PM CLI in automated workflows
**Environment**: automated project setup

**Key Characteristics**:
- Needs non-interactive initialization
- Requires predictable, scriptable commands
- Cannot handle interactive prompts
- Needs JSON output for verification

**Goals**:
- Initialize: `pm init --non-interactive`
- Verify init succeeded: check exit code and JSON output
- Sync files programmatically: `pm sync --json`

**Pain Points this Feature Addresses**:
- Interactive prompts break automation
- No machine-readable output for verification
- Manual steps cannot be scripted

## User Stories

### Must-Have User Stories

**Story 1: Initialize New Project**
- As a user, I want to run `pm init` to set up database, folders, and config in one command, so that I can start using PM CLI immediately.

**Story 2: Import Existing Task Files**
- As a user, I want to run `pm sync` to import existing markdown task files into the database, so that I can migrate legacy projects without data loss.

**Story 3: Preview Sync Changes**
- As a user, I want to run `pm sync --dry-run` to see what will change without modifying data, so that I can verify sync results before committing.

**Story 8: Create Default Config**
- As a user, I want `pm init` to create `.pmconfig.json` with sensible defaults, so that I can customize configuration without starting from scratch.

**Story 9: Selective Folder Sync**
- As a user, I want to run `pm sync --folder=<path>` to sync only a specific folder, so that I can update subsets of tasks (useful for legacy task folders or specific feature folders).

**Story 12: Epic/Feature Import**
- As a user, I want `pm sync` to detect epic and feature keys from task files and create epic/feature records if they don't exist, so that I can import tasks without manually creating epics first.

### Should-Have User Stories

**Story 4: Sync After Git Pull**
- As a developer, I want to run `pm sync` after pulling Git changes, so that my database reflects newly added or modified task files.

**Story 5: Handle Frontmatter Changes**
- As a user, I want file frontmatter changes (title, description) to update the database during sync, so that external edits are reflected in PM CLI.

**Story 6: Detect Conflicts**
- As a user, I want sync to detect conflicts (DB says title="X", file says title="Y") and report them clearly, so that I can resolve inconsistencies.

**Story 7: Choose Conflict Resolution**
- As a user, I want to choose conflict resolution strategy (`--strategy=file-wins` or `--strategy=database-wins`), so that I control which source is authoritative.

**Story 10: Force Overwrite**
- As a user, I want to run `pm sync --force` to overwrite database with file contents regardless of conflicts, so that I can reset database to match filesystem.


### Could-Have User Stories

**Story 11: Sync History Recording**
- As a user, I want sync operations to create task_history records for status changes, so that I have audit trail of external edits.

**Story 13: Interactive Conflict Resolution**
- As a user, I want sync to prompt me for each conflict (file or database wins?), so that I can resolve conflicts case-by-case.

**Story 14: Backup Before Sync**
- As a user, I want `pm sync --backup` to create database backup before syncing, so that I can rollback if sync goes wrong.

## Requirements

### Functional Requirements

**Initialization Command (pm init):**

1. The system must provide `pm init` command that sets up PM CLI infrastructure

2. The command must perform these operations in order:
   - Create database schema (run E04-F01 migrations)
   - Create folder structure (use E04-F05 folder creation)
   - Create default config file `.pmconfig.json`
   - Copy task templates to `templates/` folder
   - Display success message with next steps

3. The command must be idempotent (safe to run multiple times without errors)

4. If database already exists, skip schema creation (don't error, just skip)

5. If folders already exist, skip folder creation

6. If config file exists, prompt user: "Config file already exists. Overwrite? (y/N)" (skip if `--non-interactive`)

7. The command must support `--non-interactive` flag for automation (skip all prompts, use defaults)

8. The command must support `--force` flag to overwrite existing config and templates

9. Default config file must include:
   ```json
   {
     "default_epic": null,
     "default_agent": null,
     "color_enabled": true,
     "json_output": false
   }
   ```

10. The command must display completion message with instructions:
    ```
    PM CLI initialized successfully!

    Next steps:
    1. Edit .pmconfig.json to set default epic and agent
    2. Create tasks with: pm task create --epic=E01 --feature=F01 --title="Task title" --agent=backend
    3. Import existing tasks with: pm sync
    ```

**Synchronization Command (pm sync):**

11. The system must provide `pm sync` command that synchronizes filesystem with database

12. The command must scan feature folders for task .md files:
    - Recursively search under `docs/plan/` for feature folders matching pattern `E##-F##-*`
    - Within each feature folder, find all task markdown files (e.g., `T-E04-F06-001.md`)
    - Also scan legacy task folders if they exist: `docs/tasks/todo/`, `docs/tasks/active/`, etc.

13. For each markdown file, the command must:
    - Parse frontmatter (YAML between `---` delimiters)
    - Extract task key from frontmatter (required field)
    - Extract optional metadata: title, description, file_path
    - Query database using task key to get current status and full task record
    - Compare file metadata with database record (if exists)

14. The command must support `--folder=<folder-path>` to sync only a specific folder (e.g., `--folder=docs/tasks/todo` or `--folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation`)

15. When scanning files, the system must attempt to infer epic and feature from:
    - Parent folder structure (e.g., `docs/plan/E04-epic/E04-F06-feature/`)
    - Task key pattern in frontmatter (e.g., `T-E04-F06-001`)
    - If epic/feature cannot be determined, prompt user: "Cannot determine feature for task <key>. Please specify --epic=<key> --feature=<key>"

16. The command must support `--dry-run` flag to preview changes without applying them

17. The command must support `--json` flag to output sync report in JSON format

**File Discovery and Parsing:**

18. The system must recursively scan feature folders under `docs/plan/` for task files matching pattern `T-*.md`

19. The system must parse YAML frontmatter using a Go YAML library (e.g., gopkg.in/yaml.v3)

20. If frontmatter is invalid YAML, log warning and skip file: "Warning: Invalid frontmatter in <file>, skipping"

21. Required field in frontmatter: `key` (task identifier)

22. Optional fields in frontmatter: `title`, `description`, `file_path`

23. Status is NOT stored in frontmatter - status is retrieved from database using the task key

24. Task key must match filename: if file is `T-E01-F02-003.md`, frontmatter key must be `T-E01-F02-003`

25. If key mismatch detected, log warning: "Warning: Key mismatch in <file>: filename=T-E01-F02-003, frontmatter=T-E01-F02-004"

26. The system may read file_path field from frontmatter to identify a canonical file reference (may differ from actual file location due to legacy naming or reorganization)

**Database Comparison:**

27. For each parsed file, the system must query database for task with matching key

28. If task does not exist in database (new file):
    - Create new database record from file metadata (key, title, description)
    - Infer epic and feature from parent folder structure or task key pattern
    - If epic/feature cannot be inferred, prompt user to specify (see requirement 15)
    - Set file_path to actual file location
    - Set initial status to "todo" (default for new tasks)
    - Create task_history record: "Task imported from file"

29. If task exists in database (existing file):
    - Compare file metadata (title, description) with database record
    - Detect conflicts (differences in title or description between file and DB)
    - Update file_path in database if actual file location differs from stored path
    - Apply conflict resolution strategy for metadata differences

30. If database has tasks not found in filesystem (orphaned DB records):
    - Log warning: "Warning: Task T-E01-F02-003 in database but file not found at <path>"
    - Optionally mark as archived or delete (based on `--cleanup` flag)

**Conflict Detection:**

31. The system must detect conflicts between file and database for these fields:
    - title (if present in frontmatter and differs from DB)
    - description (if present in frontmatter and differs from DB)
    - file_path (if actual file location differs from DB record)

32. Note: Status, priority, agent_type, and depends_on are NOT stored in frontmatter and therefore cannot conflict. These fields are managed exclusively through the pm tool and stored only in the database.

33. Conflicts must be reported in sync output:
    ```
    Conflict detected in T-E01-F02-003:
      Field: title
      Database: "Implement user authentication"
      File: "Add user authentication feature"
      Resolution: file-wins (title updated to "Add user authentication feature")
    ```

**Conflict Resolution Strategies:**

34. The system must support `--strategy=<strategy>` flag with values:
    - `file-wins` (default): File metadata (title, description) overwrites database
    - `database-wins`: Database metadata is authoritative, files unchanged
    - `newer-wins`: Use timestamp comparison (most recently updated wins)

35. With `file-wins` strategy:
    - Update database record with file metadata from frontmatter (title, description)
    - Update file_path in database to match actual file location
    - Create task_history record: "Updated from file during sync"

36. With `database-wins` strategy:
    - Keep database record unchanged
    - Optionally update file frontmatter to match database (with `--update-files` flag)
    - Log conflicts but don't modify database

37. With `newer-wins` strategy:
    - Compare file modified timestamp with database updated_at
    - If file is newer, use file-wins logic
    - If database is newer, use database-wins logic

**Epic and Feature Inference:**

38. If inferred epic/feature doesn't exist in database:
    - Log warning: "Warning: Task references non-existent feature E01-F02"
    - Skip task import (don't create orphaned tasks)
    - Suggest: "Create feature E01-F02 first or run pm sync with --create-missing"

39. With `--create-missing` flag:
    - Auto-create missing epic and feature records with minimal metadata
    - Infer feature key from parent folder name or task key pattern
    - Log: "Created feature E01-F02 (inferred from task file)"

40. Feature folder organization is preserved:
    - Tasks remain in their feature folders regardless of status changes
    - File location follows pattern: `docs/plan/<epic-folder>/<feature-folder>/T-<key>.md`
    - Status is managed in database only, not in files
    - Database file_path is updated to reflect actual file location

**Sync Report:**

41. After sync completes, the system must display summary report:
    ```
    Sync completed:
      Files scanned: 47
      New tasks imported: 5
      Existing tasks updated: 3
      Conflicts resolved: 2
      Warnings: 1
      Errors: 0
    ```

42. JSON output must include detailed report:
    ```json
    {
      "scanned": 47,
      "imported": 5,
      "updated": 3,
      "conflicts": 2,
      "warnings": ["Warning: Invalid frontmatter in docs/tasks/legacy/invalid.md"],
      "errors": []
    }
    ```

**Dry-Run Mode:**

43. With `--dry-run` flag, the system must:
    - Scan files and detect changes as normal
    - Display what would change (imported, updated, conflicts)
    - NOT modify database
    - NOT update files
    - Exit with code 0 (success)

44. Dry-run output must clearly indicate preview mode: "Dry-run mode: No changes will be made"

**Transaction Safety:**

45. All database changes during sync must use transactions (Go database/sql package)

46. If any database operation fails, the entire sync must rollback

47. With `--update-files` flag, file frontmatter updates must be atomic (all-or-nothing)

48. Database and file updates must be coordinated to prevent partial updates

**Error Handling:**

49. Invalid YAML frontmatter must not halt sync (log warning, skip file, continue)

50. Missing required field (key) must not halt sync (log warning, skip file)

51. Database errors must halt sync and rollback (exit code 2)

52. Filesystem errors (permissions) must halt sync and rollback (exit code 2)

### Non-Functional Requirements

**Performance:**

- `pm init` must complete in <5 seconds
- `pm sync` must process 100 files in <10 seconds
- YAML parsing must not be bottleneck (<10ms per file)
- Database bulk inserts should be used for efficiency

**Implementation:**

- The project is implemented in Go
- Use Go standard library packages where possible (database/sql, os, path/filepath)
- YAML parsing: gopkg.in/yaml.v3 or similar Go YAML library
- Transaction support via database/sql package
- Cross-platform file path handling using filepath package

**Usability:**

- Init completion message must guide users on next steps
- Sync report must clearly show what changed
- Dry-run output must be easy to review
- Error messages must suggest fixes

**Reliability:**

- Init must be idempotent (safe to re-run)
- Sync must be transactional (all-or-nothing)
- File parsing errors must not corrupt database
- Rollback must work reliably on failures

**Data Integrity:**

- Sync must never create orphaned tasks (without valid epic/feature)
- File parsing must validate required fields
- Key uniqueness must be enforced
- Timestamps must be preserved during import

**Compatibility:**

- YAML parsing must handle common frontmatter formats
- Init must work on Linux, macOS, Windows (Go's cross-platform support)
- File path handling must use filepath package for cross-platform compatibility
- Config file must be valid JSON

## Acceptance Criteria

### Initialization

**Given** a new project with no PM CLI infrastructure
**When** I run `pm init`
**Then** database file `project.db` is created with schema
**And** folder structure `docs/tasks/{todo,active,ready-for-review,completed,archived}` is created
**And** config file `.pmconfig.json` is created with defaults
**And** task templates are copied to `templates/` folder
**And** success message displays next steps

**Given** PM CLI is already initialized
**When** I run `pm init` again
**Then** command completes without errors (idempotent)
**And** existing database is not modified
**And** existing config is not overwritten (unless --force)

### File Scanning

**Given** I have 10 task markdown files across multiple features under docs/plan/
**When** I run `pm sync`
**Then** all 10 files are scanned and parsed
**And** sync report shows "Files scanned: 10"

**Given** I have task files in multiple folders
**When** I run `pm sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation`
**Then** only task files in the specified folder are scanned
**And** files in other folders are ignored

### New Task Import

**Given** file `docs/plan/E01-epic-name/E01-F02-feature/T-E01-F02-003.md` exists with valid frontmatter containing key, title, and description
**And** task T-E01-F02-003 does not exist in database
**When** I run `pm sync`
**Then** task T-E01-F02-003 is created in database
**And** all metadata from frontmatter (key, title, description) is imported
**And** status is set to "todo" (default for new tasks)
**And** file_path is set to actual file location
**And** sync report shows "New tasks imported: 1"

**Given** file has invalid frontmatter (bad YAML)
**When** I run `pm sync`
**Then** warning is logged: "Invalid frontmatter in <file>"
**And** file is skipped
**And** sync continues with other files

### Conflict Resolution (File-Wins)

**Given** database shows task T-E01-F02-003 has title="Implement authentication"
**And** file frontmatter shows title="Add user authentication"
**When** I run `pm sync --strategy=file-wins`
**Then** database title is updated to "Add user authentication"
**And** conflict is reported in sync output
**And** task_history record is created

### Conflict Resolution (Database-Wins)

**Given** database shows task T-E01-F02-003 has title="Implement authentication"
**And** file frontmatter shows title="Add user authentication"
**When** I run `pm sync --strategy=database-wins`
**Then** database title remains "Implement authentication"
**And** conflict is reported but database not modified

### File Path Updates

**Given** database shows task T-E01-F02-003 at file_path="docs/tasks/created/T-E01-F02-003.md"
**And** actual file is at `docs/plan/E01-epic/E01-F02-feature/T-E01-F02-003.md`
**When** I run `pm sync`
**Then** database file_path is updated to match actual location
**And** task remains in feature folder (not moved)
**And** sync report shows file path conflict resolved

### Dry-Run Mode

**Given** 5 files would be imported during sync
**When** I run `pm sync --dry-run`
**Then** sync report shows "New tasks imported: 5"
**And** message shows "Dry-run mode: No changes will be made"
**And** database is not modified
**And** files are not moved

### Missing Epic/Feature

**Given** file references feature "E99-F99" that doesn't exist in database
**When** I run `pm sync`
**Then** warning is logged: "Task references non-existent feature E99-F99"
**And** task is skipped (not imported)

**Given** file references non-existent feature
**When** I run `pm sync --create-missing`
**Then** feature E99-F99 is auto-created
**And** task is imported successfully

### Transaction Rollback

**Given** sync is processing 10 files
**And** file #5 causes database constraint violation
**When** sync fails
**Then** all database changes are rolled back
**And** tasks from files #1-4 are not in database
**And** error message explains the failure

### JSON Output

**Given** I run `pm sync --json`
**When** sync completes
**Then** output is valid JSON
**And** JSON contains: scanned, imported, updated, conflicts, warnings, errors

### Non-Interactive Init

**Given** I run `pm init --non-interactive` in CI/CD
**When** config file already exists
**Then** no prompt is shown (skip config creation)
**And** command completes successfully
**And** exit code is 0

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Status-Based File Moves** - Tasks are NOT moved between folders when status changes. Tasks remain in their feature folders permanently. Only the frontmatter status field and database record are updated.

2. **Interactive Conflict Resolution** - Prompting user for each conflict is Could-Have, deferred.

3. **Automatic Backup** - Creating database backups before sync is Could-Have.

4. **Epic/Feature PRD Import** - Syncing epic/feature metadata from PRD files is out of scope (manual epic creation only).

5. **Continuous Sync** - File watching and automatic sync on file changes is out of scope (manual sync only).

6. **Merge Conflict Resolution** - Handling Git merge conflicts in frontmatter is out of scope (users resolve manually).

7. **Selective Field Sync** - Syncing only certain fields (e.g., sync status but not priority) is out of scope.

8. **Bidirectional Sync** - Updating file frontmatter from database is only available with `--update-files` flag in database-wins mode, not a full bidirectional sync.

9. **Version Control Integration** - No automatic `git commit` after sync.

10. **Schema Migration** - Upgrading database schema during sync is out of scope (separate migration commands).

11. **Orphaned Task Cleanup** - Automatically deleting database tasks without files requires explicit `--cleanup` flag, not automatic.
