# Feature: Epic & Feature Queries

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

Developers and AI agents need to understand project structure and progress at the epic and feature level, not just individual tasks. Agents need to answer questions like "Which features belong to Epic E01?" or "What is the completion percentage of Feature E01-F02?" without reading multiple markdown files and manually counting completed tasks. Developers need to see feature-level breakdowns to identify bottlenecks, understand which features are blocked or incomplete, and report progress to stakeholders. Without structured queries for epics and features, users must manually navigate folder structures, parse PRD files, and calculate progress percentages, wasting time and introducing errors. The lack of programmatic access to epic/feature metadata prevents agents from making intelligent decisions about task prioritization and prevents developers from generating accurate progress reports.

### Solution

Implement CLI commands for querying epics and features with automatic progress calculation built on E04-F01 (Database) and E04-F02 (CLI Framework). Provide commands to list all epics (`pm epic list`), get epic details with feature breakdown (`pm epic get`), list features for an epic (`pm feature list`), and get feature details with task breakdown (`pm feature get`). Each command calculates progress automatically: feature progress = (completed tasks / total tasks × 100), epic progress = weighted average of feature progress. Commands support both human-readable table output and machine-readable JSON, enabling agents to query project structure efficiently and developers to generate stakeholder reports.

### Impact

- **Project Visibility**: Single-command access to project structure and progress eliminates manual file navigation
- **Progress Accuracy**: Automatic calculation ensures progress percentages are always current (no stale data)
- **Agent Intelligence**: Agents can make informed decisions about task prioritization based on epic/feature status
- **Reporting Efficiency**: Developers can generate stakeholder reports in seconds with `pm epic get E01 --json | jq`
- **Foundation for Dashboard**: Provides the data queries needed by E05-F01 (Status Dashboard) for comprehensive project views

## User Personas

### Primary Persona: Product Manager / Technical Lead

**Role**: Human developer managing multi-epic projects
**Environment**: Terminal, managing 2-5 concurrent epics

**Key Characteristics**:
- Needs quick visibility into project health
- Tracks progress across multiple epics and features
- Generates progress reports for stakeholders
- Identifies bottlenecks and blocked features

**Goals**:
- See all epics with progress: `pm epic list`
- Drill into specific epic: `pm epic get E01`
- View feature breakdown: `pm feature list --epic=E01`
- Get feature details: `pm feature get E01-F02`

**Pain Points this Feature Addresses**:
- Manual calculation of progress percentages
- No easy way to see all features in an epic
- Must read multiple PRD files to understand structure
- Cannot programmatically generate reports

### Secondary Persona: AI Agent (Claude Code Agents)

**Role**: Autonomous code generation and task execution agent
**Environment**: Claude Code CLI, needs structured project context

**Key Characteristics**:
- Needs to understand which epic to work on
- Uses progress to prioritize work
- Requires JSON output for parsing
- Needs fast queries (<100ms)

**Goals**:
- Query epic structure: `pm epic get E01 --json`
- Find incomplete features: `pm feature list --status=active --json`
- Understand feature scope before starting tasks
- Report progress to users

**Pain Points this Feature Addresses**:
- No structured way to query project hierarchy
- Cannot determine which epic/feature needs attention
- Reading multiple markdown files wastes tokens
- No machine-readable progress data

## User Stories

### Must-Have User Stories

**Story 1: List All Epics**
- As a developer, I want to run `pm epic list`, so that I can see all epics with their status and progress percentages.

**Story 2: Get Epic Details**
- As a developer or agent, I want to run `pm epic get E01`, so that I can see the epic with all its features and their progress.

**Story 3: List Features for Epic**
- As a developer, I want to run `pm feature list --epic=E01`, so that I can see all features belonging to Epic E01.

**Story 4: Get Feature Details**
- As a developer or agent, I want to run `pm feature get E01-F02`, so that I can see the feature with all its tasks and status breakdown.

**Story 5: Calculate Feature Progress**
- As a user, I want feature progress calculated automatically as (completed tasks / total tasks × 100), so that I always see accurate progress without manual updates.

**Story 6: Calculate Epic Progress**
- As a user, I want epic progress calculated as the weighted average of feature progress, so that I understand overall epic completion.

**Story 7: JSON Output for Automation**
- As an AI agent, I want to add `--json` to any epic/feature command, so that I can parse results programmatically.

**Story 8: Handle Empty Results**
- As a user, I want empty results to show clear messages ("No features found for Epic E01") instead of errors, so that I know the query succeeded but returned nothing.

### Should-Have User Stories

**Story 9: Filter Features by Status**
- As a developer, I want to run `pm feature list --status=active`, so that I can see only active features across all epics.

**Story 10: Show Task Count by Status**
- As a developer, I want to see task breakdown in feature details (e.g., "5 completed, 2 in_progress, 3 todo"), so that I understand feature health.

**Story 11: Sort Epics by Progress**
- As a developer, I want to run `pm epic list --sort-by=progress`, so that I can prioritize low-progress epics.

### Could-Have User Stories

**Story 12: Filter Epics by Priority**
- As a developer, I want to run `pm epic list --priority=high`, so that I can focus on high-priority work.

**Story 13: Show Blocked Feature Count**
- As a developer, I want to see how many features have blocked tasks, so that I can identify workflow issues.

## Requirements

### Functional Requirements

**Epic Listing (pm epic list):**

1. The system must provide `pm epic list` command that returns all epics from the database

2. Human-readable output must display epics as a table with columns: Key | Title | Status | Progress | Priority

3. Progress must be displayed as percentage with one decimal place: "45.3%"

4. JSON output must return: `{"results": [<epic objects>], "count": <integer>}`

5. Each epic object must include: id, key, title, description, status, priority, business_value, progress_pct, created_at, updated_at

6. The system must calculate progress_pct for each epic before returning results

7. Empty results must return message "No epics found" (exit code 0, not an error)

**Epic Details (pm epic get):**

8. The system must provide `pm epic get <epic-key>` command that returns a single epic by key

9. If epic does not exist, exit with code 1 and message "Error: Epic <key> does not exist"

10. Human-readable output must display:
    - Epic metadata (key, title, status, priority, business_value, progress)
    - Table of all features with columns: Key | Title | Status | Progress | Task Count

11. JSON output must return epic object with nested `features` array containing full feature objects

12. The system must calculate feature progress for all features in the epic

13. The system must display overall epic progress prominently

**Feature Listing (pm feature list):**

14. The system must provide `pm feature list` command that returns all features

15. The system must support `--epic` filter to show features for specific epic only

16. The system must support `--status` filter to show features with specific status (draft, active, completed, archived)

17. Human-readable output must display features as a table with columns: Key | Title | Epic | Status | Progress | Tasks

18. JSON output must return: `{"results": [<feature objects>], "count": <integer>}`

19. Each feature object must include: id, epic_id, key, title, description, status, progress_pct, task_count, created_at, updated_at

20. The system must calculate progress_pct for each feature before returning results

**Feature Details (pm feature get):**

21. The system must provide `pm feature get <feature-key>` command that returns a single feature by key

22. If feature does not exist, exit with code 1 and message "Error: Feature <key> does not exist"

23. Human-readable output must display:
    - Feature metadata (key, title, epic, status, progress)
    - Task status breakdown (e.g., "Completed: 5, In Progress: 2, Todo: 3, Blocked: 1")
    - Table of all tasks with columns: Key | Title | Status | Priority | Agent

24. JSON output must return feature object with nested `tasks` array containing full task objects

25. The system must include task status breakdown as part of the output

**Progress Calculation:**

26. Feature progress must be calculated as: (count of tasks with status="completed" OR status="archived") / (count of all tasks in feature) × 100

27. If feature has zero tasks, progress must be 0.0 (not an error, not null)

28. Epic progress must be calculated as: weighted average of all feature progress values, weighted by task count per feature

29. Formula: epic_progress = Σ(feature_progress × feature_task_count) / Σ(feature_task_count)

30. If epic has zero features, progress must be 0.0

31. If epic has features with zero tasks, those features contribute 0% to weighted average

32. Progress must be stored as REAL (floating point) with precision to one decimal place in output

**Integration with Database:**

33. All commands must query E04-F01 database using ORM models (Epic, Feature, Task)

34. Progress calculations must use efficient SQL queries (not loading all data into memory)

35. JOIN queries should be used to minimize database round trips

**Integration with CLI Framework:**

36. All commands must use E04-F02 CLI framework (Click decorators, Rich formatting, error handling)

37. All commands must support global `--json` and `--no-color` flags

38. All commands must return appropriate exit codes (0=success, 1=user error, 2=system error)

**Error Handling:**

39. Non-existent epic/feature keys must return clear error messages with exit code 1

40. Database connection errors must return exit code 2 with user-friendly message

41. Invalid filter values (e.g., `--status=invalid`) must return exit code 1 with message listing valid values

### Non-Functional Requirements

**Performance:**

- `pm epic list` must return results in <100ms for 100 epics
- `pm epic get` with feature details must return in <200ms for epics with 50 features
- `pm feature get` with task details must return in <200ms for features with 100 tasks
- Progress calculations must not cause N+1 query problems (use JOINs or aggregations)

**Usability:**

- Table output must fit in 80-column terminal width (truncate long titles with "...")
- Progress percentages must be right-aligned for easy scanning
- Empty results must show helpful messages, not blank output
- Error messages must suggest related commands: "Use 'pm epic list' to see available epics"

**Accuracy:**

- Progress calculations must always reflect current database state (no caching)
- Percentages must be mathematically correct (no rounding errors that cause >100%)
- Task counts must match actual database counts

**Reliability:**

- Queries must handle missing data gracefully (epics with no features, features with no tasks)
- Division by zero in progress calculation must be handled (return 0.0, not error)
- Database errors during progress calculation must not crash (return partial results with warning)

## Acceptance Criteria

### Epic Listing

**Given** the database contains 5 epics
**When** I run `pm epic list`
**Then** all 5 epics are displayed in a table
**And** each epic shows key, title, status, progress percentage, and priority
**And** progress percentages are calculated correctly

**Given** I run `pm epic list --json`
**When** the command completes
**Then** output is valid JSON with structure: `{"results": [...], "count": 5}`
**And** each epic object includes progress_pct field

### Epic Details

**Given** Epic E01 exists with 3 features: F01 (50% complete), F02 (75% complete), F03 (100% complete)
**When** I run `pm epic get E01`
**Then** the epic is displayed with overall progress = 75% (average of feature progress)
**And** all 3 features are listed with their individual progress percentages

**Given** Epic E99 does not exist
**When** I run `pm epic get E99`
**Then** error message is displayed: "Error: Epic E99 does not exist"
**And** exit code is 1

**Given** I run `pm epic get E01 --json`
**When** the command completes
**Then** JSON output includes epic object with nested "features" array
**And** each feature includes progress_pct and task_count

### Feature Listing

**Given** the database contains 10 features across 3 epics
**When** I run `pm feature list --epic=E01`
**Then** only features belonging to Epic E01 are displayed

**Given** I run `pm feature list --status=active`
**When** the command completes
**Then** only features with status="active" are displayed

**Given** no features match the filters
**When** I run `pm feature list --epic=E99`
**Then** message is displayed: "No features found for Epic E99"
**And** exit code is 0 (not an error)

### Feature Details

**Given** Feature E01-F02 exists with 10 tasks: 7 completed, 2 in_progress, 1 todo
**When** I run `pm feature get E01-F02`
**Then** feature is displayed with progress = 70% (7/10 × 100)
**And** task breakdown shows: "Completed: 7, In Progress: 2, Todo: 1"
**And** all 10 tasks are listed in a table

**Given** Feature E01-F99 does not exist
**When** I run `pm feature get E01-F99`
**Then** error message is displayed: "Error: Feature E01-F99 does not exist"
**And** exit code is 1

### Progress Calculation

**Given** Feature E01-F01 has 10 tasks: 5 completed, 5 todo
**When** progress is calculated
**Then** feature progress_pct = 50.0

**Given** Feature E01-F02 has 0 tasks
**When** progress is calculated
**Then** feature progress_pct = 0.0 (not null, not error)

**Given** Epic E01 has 2 features:
  - Feature F01: 50% complete, 10 tasks
  - Feature F02: 100% complete, 10 tasks
**When** epic progress is calculated
**Then** epic progress_pct = 75.0 (weighted average: (50×10 + 100×10) / (10+10) = 1500/20 = 75)

**Given** Epic E02 has 2 features:
  - Feature F01: 100% complete, 1 task
  - Feature F02: 0% complete, 9 tasks
**When** epic progress is calculated
**Then** epic progress_pct = 10.0 (weighted average: (100×1 + 0×9) / (1+9) = 100/10 = 10)

### JSON Output

**Given** I run `pm epic get E01 --json`
**When** the command completes
**Then** output is valid JSON
**And** I can parse it with `jq '.key'` to extract "E01"
**And** I can parse it with `jq '.features[0].progress_pct'` to extract feature progress

### Error Handling

**Given** the database connection fails
**When** I run `pm epic list`
**Then** error message is displayed: "Error: Database error. Run with --verbose for details."
**And** exit code is 2 (system error)

**Given** I run `pm feature list --status=invalid`
**When** the command executes
**Then** error message is displayed: "Error: Invalid status. Must be one of: draft, active, completed, archived"
**And** exit code is 1 (user error)

### Table Formatting

**Given** terminal width is 80 columns
**When** I run `pm epic list`
**Then** the table fits within 80 columns
**And** long titles are truncated with "..." ellipsis

**Given** I run `pm feature get E01-F01` with 50 tasks
**When** the task table is displayed
**Then** tasks are formatted as a readable table
**And** progress percentage is prominently displayed at the top

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Creating Epics/Features** - This feature is read-only. Creating epics and features is done manually via PRD files (not CLI).

2. **Updating Epic/Feature Metadata** - No commands to change epic title, status, or priority. Metadata is managed externally.

3. **Deleting Epics/Features** - No deletion commands. This would be destructive and cascade to features/tasks.

4. **Dashboard View** - The comprehensive `pm status` command with progress bars and multi-epic overview is in E05-F01 (Status Dashboard).

5. **Dependency Visualization** - Showing which features depend on others is out of scope (E05-F02).

6. **Historical Progress** - Progress over time tracking is out of scope (E05-F03).

7. **Advanced Filtering** - Complex queries like "show epics with >50% progress" are deferred to E05.

8. **Custom Fields** - Only standard epic/feature fields are supported (no custom metadata).

9. **Epic/Feature Archives** - While status can be "archived", no special commands for archival management.

10. **Exporting to External Formats** - CSV/Excel export is out of scope (manual JSON export via `--json | jq` is sufficient).
