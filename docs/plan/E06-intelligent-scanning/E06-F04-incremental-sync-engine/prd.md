# Feature: Incremental Sync Engine

## Epic

- [Epic PRD](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/epic.md)
- [Epic Requirements](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/requirements.md)

## Goal

### Problem

The current synchronization system (E04-F07) performs full filesystem scans every time `shark sync` is executed, regardless of how many files have actually changed. For AI agents working in development sessions, this creates significant inefficiencies: syncing after creating or modifying 3-5 task files requires scanning the entire documentation tree (potentially 100-500+ files), taking 5-10 seconds per sync operation. When agents perform multiple sync operations per session, this overhead accumulates to 30-60 seconds of wasted time. Additionally, the system has no mechanism to detect when both the database and filesystem have been modified between syncs, leading to silent data overwrites and potential loss of manual database edits made through the CLI. As projects scale to hundreds of documentation files, full scan performance degrades linearly, making sync operations a bottleneck in agent productivity and developer workflows.

### Solution

Implement an incremental sync engine that tracks modification timestamps and only processes changed files since the last successful sync. The engine will:

1. **Track Last Sync Time**: Record `last_sync_time` in `.sharkconfig.json` after each successful sync operation
2. **Filter Changed Files**: Compare file modification time (`mtime`) against `last_sync_time` to identify files that need processing
3. **Detect Conflicts**: Identify scenarios where both database records and filesystem files have been modified since the last sync
4. **Apply Resolution Strategies**: Support configurable conflict resolution (`file-wins`, `db-wins`, `manual`) with detailed logging
5. **Maintain Transaction Safety**: Ensure all database updates are atomic with proper rollback on errors
6. **Optimize Performance**: Achieve <2 seconds for 1-10 file changes, <5 seconds for 100 file changes

This feature builds on E04-F07's synchronization foundation by adding intelligent change detection, reducing unnecessary I/O operations, and preventing data loss through conflict awareness. It directly supports the AI Agent persona (Journey 2) who needs sub-second sync operations during development sessions.

### Impact

- **Agent Session Efficiency**: Reduce incremental sync time from 5-10 seconds (full scan) to <2 seconds (changed files only), saving 8-50+ seconds per agent session
- **Sync Performance at Scale**: Maintain <5 second sync times even as documentation grows from 100 to 500+ files
- **Data Integrity**: Prevent silent data overwrites by detecting and reporting conflicts when both database and files have changed
- **Developer Confidence**: Provide detailed conflict reports with resolution strategies, enabling informed decisions about which changes to preserve
- **Token Efficiency**: Reduce filesystem I/O and processing overhead, minimizing resource consumption during agent workflows
- **Scalability Foundation**: Enable shark to handle large projects (1000+ documentation files) without performance degradation

## User Personas

### Primary Persona: AI Agent (Incremental Development Sync)

**Role**: Autonomous code generation and task execution agent (Claude Code)

**Environment**: Claude Code CLI, stateless between sessions, working in feature folders

**Key Characteristics**:
- Creates 3-7 new task files per work session
- Updates existing task file frontmatter (status, description, title)
- Runs sync at session end to update database before termination
- Cannot tolerate slow operations (time budget constraints)
- Needs atomic, reliable sync with clear success/failure indicators
- Limited ability to resolve complex conflicts interactively

**Goals Related to This Feature**:
1. Sync only changed files (new/modified) without scanning entire documentation tree
2. Complete sync operations in <2 seconds for typical session (5-7 file changes)
3. Receive clear error messages if conflicts detected
4. Rely on default conflict resolution (file-wins) for automated workflows

**Pain Points This Feature Addresses**:
- Current full scans waste 5-10 seconds per sync when only 3-5 files changed
- No visibility into what changed or why sync took so long
- Silent overwrites when database was updated between syncs
- Performance degrades as project documentation grows

**Success Looks Like**:
AI Agent creates 4 new task files and updates 2 existing task descriptions during a feature implementation session. At session end, agent runs `shark sync --incremental` which completes in 1.3 seconds, reports "6 files synced (4 created, 2 updated)", and confirms database is current. Agent session ends with confidence that next session will have accurate project state.

---

### Secondary Persona: Product Manager / Technical Lead (Manual Edit Synchronization)

**Role**: Human developer managing project with manual markdown edits and CLI operations

**Environment**: Terminal, text editor (VS Code, Vim), Git workflow

**Key Characteristics**:
- Occasionally edits task markdown files directly (bypassing CLI)
- Uses CLI to update task status, priorities, and assignments
- Pulls Git changes from collaborators that include documentation updates
- Needs to sync after Git pulls without rescanning unchanged files
- Values data integrity and wants visibility into conflicts

**Goals Related to This Feature**:
1. Sync after Git pulls without waiting for full project scan
2. Detect conflicts when both CLI updates and manual file edits occurred
3. Choose resolution strategy (file-wins vs. db-wins) based on context
4. Review detailed conflict reports before applying changes

**Pain Points This Feature Addresses**:
- Full scans after Git pulls are slow when only 10-20 files changed
- No way to detect if CLI edits conflict with file changes
- Silent data loss when file changes overwrite recent CLI updates
- Cannot preview conflicts before committing to resolution strategy

**Success Looks Like**:
Product Manager uses CLI to update status of 5 tasks (database changes). Collaborator pushes Git changes that modify 3 task files, including 1 of the recently updated tasks. Product Manager runs `git pull` then `shark sync --incremental --dry-run` which detects 1 conflict (T-E04-F06-003: database status changed to "completed", file status shows "active"). Product Manager reviews conflict report, decides file should win (collaborator's edit is newer), runs `shark sync --incremental --strategy=file-wins`, and database is updated with 3 file changes including the conflict resolution. Total time: <3 seconds.

---

### Tertiary Persona: Technical Lead (Large Project Import)

**Role**: Developer importing existing project with 200+ documentation files

**Environment**: Existing documentation repository, initial shark setup

**Key Characteristics**:
- Performing initial full scan to import all documentation
- Subsequent syncs should be incremental (only changed files)
- Needs performance guarantee that sync won't become bottleneck as project grows
- May re-scan periodically to validate database consistency

**Goals Related to This Feature**:
1. Initial full scan establishes baseline `last_sync_time`
2. Subsequent syncs are fast (<5 seconds) even with 100+ files in project
3. Option to force full rescan when needed (validation, troubleshooting)
4. Performance remains consistent as project scales to 500+ files

**Pain Points This Feature Addresses**:
- Fear that sync performance will degrade as documentation grows
- No way to force full rescan when needed (validation scenarios)
- Uncertainty about what changed between syncs

**Success Looks Like**:
Technical Lead runs initial `shark scan --execute` which takes 15 seconds to process 247 files and records `last_sync_time`. Over the next week, 30 incremental sync operations complete in average of 2.1 seconds each (vs. 15 seconds for full scans), processing 3-12 changed files per sync. After 3 months, project has grown to 380 files, but incremental syncs still complete in <3 seconds. Occasional full rescan with `--force-full-scan` flag validates database consistency in 22 seconds.

## User Stories

### Must-Have User Stories

**Story 1: Track Last Sync Timestamp**
- As a user, I want the system to record the timestamp of the last successful sync in `.sharkconfig.json`, so that subsequent syncs can identify changed files.

**Story 2: Incremental Sync Based on File Modification Time**
- As an AI Agent, I want `shark sync --incremental` to only process files modified since `last_sync_time`, so that sync completes in <2 seconds for typical sessions (5-7 file changes).

**Story 3: Detect Database-File Conflicts**
- As a Product Manager, I want sync to detect when both database record and file have been modified since last sync, so that I can prevent silent data loss.

**Story 4: Apply File-Wins Conflict Resolution**
- As a user, I want `--strategy=file-wins` (default) to update database records with file contents when conflicts are detected, so that filesystem changes take precedence.

**Story 5: Apply Database-Wins Conflict Resolution**
- As a user, I want `--strategy=db-wins` to preserve database records and ignore file changes when conflicts are detected, so that CLI updates take precedence.

**Story 6: Report Conflict Details**
- As a Product Manager, I want detailed conflict reports showing old values, new values, timestamps, and resolution strategy applied, so that I can verify the correct data was preserved.

**Story 7: Dry-Run Conflict Detection**
- As a user, I want `shark sync --incremental --dry-run` to detect and report conflicts without modifying database, so that I can preview changes before committing.

**Story 8: Atomic Transaction Safety**
- As a user, I want all incremental sync database updates to be atomic (all-or-nothing), so that partial failures don't corrupt database state.

**Story 9: Force Full Scan Override**
- As a Technical Lead, I want `shark sync --force-full-scan` to ignore `last_sync_time` and scan all files, so that I can validate database consistency or recover from clock skew issues.

**Story 10: Update Last Sync Time on Success**
- As a user, I want `last_sync_time` updated only after successful sync completion, so that failed syncs can be retried without losing track of unprocessed changes.

### Should-Have User Stories

**Story 11: Performance Metrics in Sync Report**
- As a Product Manager, I want sync reports to include performance metrics (files scanned, files changed, elapsed time), so that I can verify incremental sync performance gains.

**Story 12: Selective Folder Incremental Sync**
- As a user, I want `shark sync --incremental --folder=<path>` to apply incremental filtering within a specific folder, so that I can efficiently sync subsets of documentation.

**Story 13: Handle Clock Skew Gracefully**
- As a user, I want sync to handle small clock skew (±5 seconds) gracefully, so that minor time synchronization issues don't cause false positives.

**Story 14: Fallback to Full Scan on Missing Last Sync Time**
- As a user, I want incremental sync to automatically perform full scan when `last_sync_time` is not available, so that the feature works on first use without special initialization.

**Story 15: Conflict Resolution Logging**
- As a Technical Lead, I want all conflict resolutions logged with timestamps, file paths, and resolution strategy, so that I can audit data integrity issues.

### Could-Have User Stories

**Story 16: Manual Conflict Resolution Mode**
- As a Product Manager, I want `--strategy=manual` to prompt me for each conflict (file or database wins?), so that I can resolve conflicts case-by-case interactively.

**Story 17: Git-Based Change Detection**
- As an AI Agent, I want sync to use `git diff --name-only` to detect changed files when working in Git repository, so that change detection is more accurate than mtime comparison.

**Story 18: Conflict History Tracking**
- As a user, I want conflict resolutions recorded in `task_history` table, so that I have audit trail of data integrity decisions.

**Story 19: Per-File Conflict Strategy Override**
- As a Product Manager, I want to specify per-file conflict strategy (via frontmatter or config), so that I can have different resolution rules for different file types.

**Story 20: Performance Benchmarking Mode**
- As a Technical Lead, I want `shark sync --benchmark` to compare incremental vs. full scan performance, so that I can validate performance improvements.

## Requirements

### Functional Requirements

#### Last Sync Time Tracking

1. The system must record the current timestamp in `.sharkconfig.json` under the key `last_sync_time` after each successful sync operation

2. The timestamp must be in RFC3339 format (ISO 8601) with timezone information (e.g., "2025-12-17T14:30:45-08:00")

3. If `.sharkconfig.json` does not exist, the system must create it with default configuration including `last_sync_time: null`

4. If sync fails (database error, transaction rollback, parsing error), `last_sync_time` must NOT be updated

5. The system must read `last_sync_time` from config at sync start to determine baseline for change detection

6. If `last_sync_time` is `null` or missing, incremental sync must automatically fallback to full scan mode

#### Incremental File Filtering

7. The system must provide `shark sync --incremental` flag to enable modification-based file filtering

8. When `--incremental` is specified, the system must compare each file's modification time (`mtime`) against `last_sync_time`

9. Only files where `mtime > last_sync_time` must be parsed and processed

10. Files where `mtime <= last_sync_time` must be skipped entirely (no parsing, no database queries)

11. For new files (not in database), the system must include them in changed files list regardless of `mtime` (database presence check is required)

12. The system must support `--force-full-scan` flag to override `last_sync_time` and scan all files regardless of modification time

13. When `--incremental` is specified without `last_sync_time` available, the system must log warning: "No last_sync_time found, performing full scan" and proceed with full scan

14. Directory modification times must NOT be used for filtering; only individual file `mtime` is considered

15. The system must handle filesystem timestamp precision correctly (some filesystems have 1-second precision, others nanosecond)

#### Conflict Detection

16. The system must detect conflicts by comparing both file `mtime` and database `updated_at` timestamp against `last_sync_time`

17. A conflict exists when ALL of the following are true:
    - File `mtime > last_sync_time` (file modified since last sync)
    - Database `updated_at > last_sync_time` (database record modified since last sync)
    - File metadata (title, description, status) differs from database record

18. For new files (not in database), no conflict is possible (treat as create operation)

19. For files where `mtime <= last_sync_time` but database `updated_at > last_sync_time`, no conflict exists (database update only, skip file)

20. The system must check these fields for conflicts:
    - Task: `title`, `description`, `status`, `priority`, `agent_type`
    - Feature: `title`, `description`
    - Epic: `title`, `description`

21. Fields NOT stored in file frontmatter (e.g., `depends_on`, internal IDs, timestamps) must NOT be checked for conflicts

22. The system must collect all conflicts detected during scan and report them together (not one at a time)

#### Conflict Resolution Strategies

23. The system must support `--strategy=<strategy>` flag with values: `file-wins`, `db-wins`, `manual`

24. Default strategy when `--strategy` is not specified must be `file-wins`

25. With `--strategy=file-wins`:
    - Database record must be updated with file metadata (title, description, status, etc.)
    - File `mtime` must be recorded in database for future conflict detection
    - Conflict must be logged with old values, new values, and resolution: "file-wins"
    - Operation must succeed (database updated)

26. With `--strategy=db-wins`:
    - Database record must remain unchanged
    - File is not updated (database-to-file sync is out of scope)
    - Conflict must be logged with old values, new values, and resolution: "db-wins"
    - File `mtime` must still be recorded to prevent repeated conflict detection on same file

27. With `--strategy=manual`:
    - System must present conflict to user with options: "[F]ile wins, [D]atabase wins, [S]kip"
    - User selects resolution for each conflict individually
    - Selected strategy is applied to that conflict only
    - All unresolved conflicts must be skipped (no database changes for those records)

28. Conflict resolution strategy must apply to ALL detected conflicts uniformly (cannot mix strategies in single sync unless `manual`)

29. When using `--dry-run`, conflicts must be detected and reported but no resolution strategy is applied (database unchanged)

#### Transaction Safety and Atomicity

30. All database modifications during incremental sync must be wrapped in a single transaction

31. The transaction must include:
    - New record insertions (tasks, features, epics created from new files)
    - Record updates (tasks, features, epics modified in files)
    - Conflict resolutions (database updates based on resolution strategy)
    - Metadata updates (file paths, timestamps)

32. If ANY database operation fails during sync, the entire transaction must be rolled back

33. After rollback, `last_sync_time` must NOT be updated (allows retry)

34. After rollback, the system must display error message with specific failure reason and affected file

35. On successful transaction commit, `last_sync_time` must be updated to the timestamp captured at sync start (not commit time)

36. The system must use prepared statements and parameterized queries for all database operations (prevent SQL injection)

37. Database connection must be properly closed even if transaction fails

#### Sync Reporting

38. After incremental sync completes, the system must display summary report:
    ```
    Incremental sync completed in 1.8 seconds:
      Files scanned: 287
      Files changed: 6 (4 created, 2 updated)
      Conflicts detected: 1
      Conflicts resolved: 1 (file-wins)
      Warnings: 0
      Errors: 0
    ```

39. The report must include performance metrics:
    - Total elapsed time (seconds with 1 decimal precision)
    - Total files in documentation tree (scanned for mtime comparison)
    - Files processed (mtime > last_sync_time)
    - Files created (new records inserted)
    - Files updated (existing records modified)

40. The report must include conflict summary:
    - Total conflicts detected
    - Conflicts resolved (with strategy used)
    - Conflicts skipped (if applicable in manual mode)

41. For each conflict, the system must log detailed information:
    ```
    Conflict in T-E04-F06-003 (docs/plan/.../T-E04-F06-003.md):
      Field: status
      Database: "completed" (updated 2025-12-17 14:25:00)
      File: "active" (modified 2025-12-17 14:28:00)
      Resolution: file-wins → database updated to "active"
    ```

42. JSON output (with `--json` flag) must include structured report:
    ```json
    {
      "elapsed_seconds": 1.8,
      "files_scanned": 287,
      "files_changed": 6,
      "files_created": 4,
      "files_updated": 2,
      "conflicts_detected": 1,
      "conflicts_resolved": 1,
      "resolution_strategy": "file-wins",
      "warnings": [],
      "errors": [],
      "conflicts": [
        {
          "key": "T-E04-F06-003",
          "file_path": "docs/plan/.../T-E04-F06-003.md",
          "field": "status",
          "db_value": "completed",
          "file_value": "active",
          "resolution": "file-wins"
        }
      ]
    }
    ```

#### Dry-Run Mode

43. With `--dry-run` flag, the system must:
    - Perform all file scanning and change detection as normal
    - Detect conflicts and generate conflict reports
    - Display what WOULD be changed (creates, updates, resolutions)
    - NOT modify database (skip transaction commit)
    - NOT update `last_sync_time` in config
    - Exit with code 0 (success)

44. Dry-run output must clearly indicate preview mode at start and end:
    ```
    DRY-RUN MODE: No changes will be made to database

    [... sync report ...]

    DRY-RUN MODE: Database was not modified
    ```

#### Error Handling and Edge Cases

45. If file `mtime` is in the future (clock skew), the system must:
    - Allow small skew (±60 seconds) without warning
    - Log warning for large skew (>60 seconds): "File T-E04-F06-003.md has future mtime, possible clock skew"
    - Still process the file (treat as changed)

46. If file `mtime` is significantly older than `last_sync_time` but content has changed (unlikely edge case):
    - The system will miss the change (by design - relies on mtime)
    - User must use `--force-full-scan` to detect such changes
    - This is acceptable tradeoff for performance

47. If `.sharkconfig.json` is corrupted or contains invalid `last_sync_time`:
    - Log error: "Invalid last_sync_time in config, performing full scan"
    - Proceed with full scan
    - Update config with valid timestamp on success

48. If filesystem does not support `mtime` (highly unlikely):
    - System must detect lack of mtime support
    - Fall back to full scan mode automatically
    - Log warning: "Filesystem does not support modification times, incremental sync disabled"

49. If incremental sync finds zero changed files:
    - Display message: "No files changed since last sync"
    - Do NOT update `last_sync_time` (preserve original timestamp)
    - Exit with code 0 (success, no action needed)

50. If user specifies `--incremental` but also provides `--folder=<path>`:
    - Incremental filtering applies ONLY within specified folder
    - Files outside folder are not scanned
    - Behavior is intersection of folder filter and incremental filter

#### Integration with Existing Sync

51. Incremental sync must reuse E04-F07's file parsing, pattern matching, and database insertion logic

52. Incremental sync must honor existing flags:
    - `--dry-run`: Preview changes without modification
    - `--folder=<path>`: Limit sync to specific folder
    - `--json`: Output structured JSON report
    - `--strategy=<strategy>`: Conflict resolution (extends E04-F07's basic strategy)

53. When `--incremental` is NOT specified, the system must perform full scan (E04-F07 behavior) but still update `last_sync_time` on success

54. Incremental sync must work with E06's pattern matching (REQ-F-003, REQ-F-005, REQ-F-007) to identify changed epic/feature/task files

### Non-Functional Requirements

#### Performance

- **REQ-NF-001: Incremental Sync Speed**: Incremental sync must complete in <2 seconds for 1-10 file changes, <5 seconds for 100 file changes, <30 seconds for full project scan (500+ files)

- **REQ-NF-002: Filtering Efficiency**: File modification time comparison (mtime check) must add <100ms overhead for 500 files

- **REQ-NF-003: Database Query Performance**: Conflict detection queries (checking database `updated_at`) must complete in <50ms for 100 records

- **REQ-NF-004: Transaction Commit Time**: Database transaction commit must complete in <100ms for 100 record updates

- **REQ-NF-005: Memory Efficiency**: Incremental sync must use <50MB additional memory compared to full scan (avoid loading entire file list into memory)

#### Reliability

- **REQ-NF-006: Transactional Import**: All database modifications during incremental sync must be atomic (all-or-nothing) with proper rollback on any error (E06 REQ-NF-003)

- **REQ-NF-007: Idempotent Syncs**: Running incremental sync multiple times on the same files (without changes) must produce identical database state (E06 REQ-NF-004)

- **REQ-NF-008: Error Recovery**: Parser failures on individual files must not halt entire sync; log error, skip file, continue with remaining files

- **REQ-NF-009: Clock Skew Tolerance**: System must handle ±60 seconds of clock skew gracefully without false positives or failures

- **REQ-NF-010: Database Lock Handling**: If database is locked (concurrent access), system must retry 3 times with exponential backoff before failing

#### Usability

- **REQ-NF-011: Actionable Error Messages**: All errors and warnings must include file path, field name, old/new values, and suggested fix

- **REQ-NF-012: Progress Visibility**: For large incremental syncs (>50 files changed), display progress indicator: "Processing changed files: 23/67 (34%)"

- **REQ-NF-013: Clear Conflict Reports**: Conflict reports must show file path, field, both values, timestamps, and resolution applied in human-readable format

- **REQ-NF-014: JSON Schema**: JSON output must follow documented schema with semantic versioning for breaking changes

#### Maintainability

- **REQ-NF-015: Code Reuse**: Incremental sync must reuse 80%+ of E04-F07 parsing and database logic (avoid duplication)

- **REQ-NF-016: Test Coverage**: Incremental sync logic must have >80% unit test coverage with focus on edge cases (clock skew, missing timestamps, conflicts)

- **REQ-NF-017: Integration Tests**: Automated tests must cover: first sync (no last_sync_time), incremental sync (some files changed), conflict scenarios (file-wins, db-wins, manual)

- **REQ-NF-018: Configuration Validation**: System must validate `last_sync_time` format on config load and handle invalid values gracefully

#### Security

- **REQ-NF-019: Path Validation**: File paths must be validated to prevent directory traversal attacks (no "../" or absolute paths outside docs root)

- **REQ-NF-020: Input Sanitization**: Timestamps from config and filesystem must be sanitized to prevent injection attacks or timestamp manipulation exploits

#### Compatibility

- **REQ-NF-021: Cross-Platform Timestamps**: File modification time handling must work correctly on Linux, macOS, and Windows (different mtime precision)

- **REQ-NF-022: Timezone Handling**: `last_sync_time` must be stored with timezone and correctly compared against local filesystem times

- **REQ-NF-023: Backward Compatibility**: Incremental sync must work with projects that don't have `last_sync_time` (automatic fallback to full scan)

- **REQ-NF-024: Config File Format**: Adding `last_sync_time` to `.sharkconfig.json` must not break existing config parsers

## Acceptance Criteria

### Last Sync Time Tracking

**Given** I run `shark sync` (full scan) on a new project
**When** sync completes successfully
**Then** `.sharkconfig.json` contains `last_sync_time` with current RFC3339 timestamp
**And** timestamp includes timezone information

**Given** I run `shark sync` and database operation fails
**When** transaction is rolled back
**Then** `last_sync_time` is NOT updated in config
**And** next sync will reprocess same files

### Incremental File Filtering

**Given** `last_sync_time` is set to "2025-12-17T14:00:00Z"
**And** I have 100 files in documentation tree
**And** 5 files were modified after 14:00:00
**When** I run `shark sync --incremental`
**Then** only 5 files are parsed and processed
**And** sync completes in <2 seconds
**And** sync report shows "Files scanned: 100, Files changed: 5"

**Given** I run `shark sync --incremental` and `last_sync_time` is not set
**When** sync starts
**Then** system logs warning: "No last_sync_time found, performing full scan"
**And** all files are processed
**And** `last_sync_time` is set after successful completion

### Conflict Detection

**Given** database record T-E04-F06-003 has `status="completed"` with `updated_at="2025-12-17T14:25:00Z"`
**And** file T-E04-F06-003.md has `status="active"` with `mtime="2025-12-17T14:28:00Z"`
**And** `last_sync_time="2025-12-17T14:20:00Z"` (both modified since last sync)
**When** I run `shark sync --incremental`
**Then** conflict is detected on field `status`
**And** conflict report shows database value "completed" and file value "active"

**Given** database record has `updated_at < last_sync_time` (no database change)
**And** file has `mtime > last_sync_time` (file changed)
**When** I run `shark sync --incremental`
**Then** NO conflict is detected
**And** file changes update database normally (file-wins by default)

### Conflict Resolution: File-Wins

**Given** conflict exists on T-E04-F06-003 field `status` (database="completed", file="active")
**When** I run `shark sync --incremental --strategy=file-wins`
**Then** database `status` is updated to "active"
**And** database `updated_at` is updated to current timestamp
**And** conflict is logged: "Conflict in T-E04-F06-003: status: completed → active (file-wins)"
**And** sync report shows "Conflicts resolved: 1 (file-wins)"

### Conflict Resolution: Database-Wins

**Given** conflict exists on T-E04-F06-003 field `status` (database="completed", file="active")
**When** I run `shark sync --incremental --strategy=db-wins`
**Then** database `status` remains "completed"
**And** file is NOT modified (database-to-file sync is out of scope)
**And** conflict is logged: "Conflict in T-E04-F06-003: status: completed (db-wins, file value 'active' ignored)"
**And** sync report shows "Conflicts resolved: 1 (db-wins)"

### Conflict Resolution: Manual (Interactive)

**Given** conflict exists on T-E04-F06-003 field `status`
**When** I run `shark sync --incremental --strategy=manual`
**Then** system prompts: "Conflict in T-E04-F06-003: status: DB='completed', File='active'. [F]ile wins, [D]atabase wins, [S]kip?"
**And** I enter "F"
**And** database is updated with file value "active"
**And** next conflict (if any) is presented for resolution

### Dry-Run Conflict Detection

**Given** conflict exists on T-E04-F06-003
**When** I run `shark sync --incremental --dry-run`
**Then** conflict is detected and reported with full details
**And** database is NOT modified
**And** `last_sync_time` is NOT updated
**And** output starts with "DRY-RUN MODE: No changes will be made"
**And** output ends with "DRY-RUN MODE: Database was not modified"

### Transaction Safety

**Given** incremental sync will process 10 changed files
**And** file #7 has invalid frontmatter (YAML parse error)
**When** sync executes
**Then** files #1-6 are processed successfully
**And** file #7 is skipped with warning: "Invalid frontmatter in file #7"
**And** files #8-10 are processed successfully
**And** transaction commits successfully (parser errors don't abort transaction)

**Given** incremental sync will process 10 changed files
**And** file #7 creates database constraint violation (duplicate key)
**When** sync executes
**Then** transaction is rolled back
**And** NO files are committed to database
**And** error message explains constraint violation with file path
**And** `last_sync_time` is NOT updated

### Force Full Scan Override

**Given** `last_sync_time` is set and only 3 files changed since then
**When** I run `shark sync --incremental --force-full-scan`
**Then** all files in documentation tree are scanned and processed
**And** sync report shows all files processed (not just 3 changed)
**And** `last_sync_time` is updated to current timestamp

### Performance Guarantees

**Given** I have 250 files in documentation tree
**And** 7 files were modified since `last_sync_time`
**When** I run `shark sync --incremental`
**Then** sync completes in <2 seconds
**And** only 7 files are parsed (not 250)

**Given** I have 500 files in documentation tree
**And** 150 files were modified since `last_sync_time`
**When** I run `shark sync --incremental`
**Then** sync completes in <5 seconds
**And** sync report shows accurate counts

### Clock Skew Handling

**Given** file has `mtime` 30 seconds in the future (clock skew)
**When** I run `shark sync --incremental`
**Then** file is processed normally (no warning for small skew)
**And** sync completes successfully

**Given** file has `mtime` 5 minutes in the future (significant clock skew)
**When** I run `shark sync --incremental`
**Then** warning is logged: "File T-E04-F06-003.md has future mtime, possible clock skew"
**And** file is still processed (treated as changed)

### Integration with Folder Filtering

**Given** I specify `--folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation`
**And** I specify `--incremental`
**When** I run sync
**Then** only files within specified folder AND modified since `last_sync_time` are processed
**And** files outside folder are ignored
**And** files inside folder but unchanged are skipped

### JSON Output

**Given** I run `shark sync --incremental --json`
**When** sync completes with 1 conflict
**Then** output is valid JSON
**And** JSON contains: `elapsed_seconds`, `files_scanned`, `files_changed`, `conflicts_detected`, `conflicts` array
**And** `conflicts` array includes: `key`, `file_path`, `field`, `db_value`, `file_value`, `resolution`

### Zero Changed Files

**Given** `last_sync_time` is current and no files have been modified
**When** I run `shark sync --incremental`
**Then** sync report shows "No files changed since last sync"
**And** `last_sync_time` is NOT updated (preserves original timestamp)
**And** exit code is 0 (success)

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Bidirectional Sync (Database-to-File)** - Updating file frontmatter from database changes is not included. Only file-to-database sync is supported. Database changes made via CLI do not propagate back to files (except through manual editing).

2. **Git Integration for Change Detection (REQ-F-021)** - Using `git diff --name-only` or `git status` to identify changed files is deferred to Could-Have. This feature relies solely on filesystem `mtime` comparison.

3. **Per-File Conflict Strategy** - Specifying different resolution strategies for different files (via frontmatter config or file-level settings) is out of scope. Global strategy applies to all conflicts in a single sync.

4. **Conflict History in Task History Table** - Recording conflict resolutions in `task_history` table for audit trail is Could-Have. Conflicts are logged to console/file but not stored in database history.

5. **Interactive Conflict Resolution UI** - While manual strategy is Could-Have, a rich interactive UI for reviewing conflicts (e.g., side-by-side diff, preview changes) is out of scope. Manual mode uses simple text prompts.

6. **Automatic Backup Before Sync** - Creating database snapshots before applying incremental sync changes is not included. Users must manually backup database if desired.

7. **Partial Transaction Commits** - If 10 files change and file #7 fails, the system either commits all valid files (parser errors) or rolls back everything (database errors). No "commit successful files, skip failed files" middle ground.

8. **Incremental Sync of Related Documents (REQ-F-006)** - While the feature works with epic/feature/task files, cataloging and syncing related documents (architecture.md, design specs) is out of scope for this feature (handled by other E06 features).

9. **Multi-Root Incremental Sync (REQ-F-020)** - Incremental sync works with single documentation root. Scanning multiple roots (docs/plan, docs/archived) with separate `last_sync_time` tracking is out of scope.

10. **Sync Scheduling or Automation** - No automatic sync on file change detection (inotify, fswatch). Incremental sync is always manual via CLI command.

11. **Conflict Resolution Rules Engine** - No support for complex resolution rules (e.g., "for status field, always use file; for description, use database"). Only global strategies supported.

12. **Performance Benchmarking Mode** - Built-in performance comparison between incremental and full scan is Could-Have. Users can manually time operations but no automated benchmarking.

13. **Incremental Sync of Deletions** - Detecting and handling deleted files (file existed at last sync but now missing) is not included. Full scan with orphaned record detection is required for cleanup.

14. **Timestamp-Based Conflict Resolution (newer-wins)** - While E04-F07 mentions "newer-wins" strategy, this feature focuses on file-wins and db-wins. Automatic timestamp-based resolution is out of scope (users can manually compare timestamps in conflict report).

15. **Configuration Versioning** - No schema versioning or migration for `.sharkconfig.json` when adding `last_sync_time`. Existing configs are simply extended with the new field.
