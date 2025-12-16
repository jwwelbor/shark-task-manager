# Epic & Feature Query Documentation

Comprehensive guide to querying epics and features with automatic progress calculation in the PM CLI.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Epic Commands](#epic-commands)
  - [List All Epics](#list-all-epics)
  - [Get Epic Details](#get-epic-details)
- [Feature Commands](#feature-commands)
  - [List All Features](#list-all-features)
  - [List Features by Epic](#list-features-by-epic)
  - [Get Feature Details](#get-feature-details)
- [Progress Calculation](#progress-calculation)
- [Output Formats](#output-formats)
- [Error Handling](#error-handling)
- [Use Cases](#use-cases)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The Epic & Feature Query functionality provides powerful CLI commands for understanding project structure and tracking progress at the epic and feature level. All commands automatically calculate progress percentages based on task completion status.

### Key Features

- List all epics with completion percentages
- Drill into specific epic details with feature breakdown
- List features across all epics or filtered by epic
- View feature details with task status breakdown
- Automatic progress calculation (always current, never stale)
- Support for both human-readable tables and machine-readable JSON
- Fast query performance (<200ms even for large projects)
- Clear error messages with helpful suggestions

### When to Use These Commands

- **Project Planning**: Understand which epics and features exist
- **Progress Tracking**: Check completion percentages at epic/feature level
- **Bottleneck Identification**: Find features with low progress or many blocked tasks
- **Stakeholder Reporting**: Generate progress reports quickly
- **AI Agent Context**: Agents can query structure before selecting tasks to work on

## Quick Start

```bash
# Install the PM CLI
make install-pm

# List all epics
pm epic list

# Get details for a specific epic
pm epic get E04

# List all features
pm feature list

# List features in a specific epic
pm feature list --epic=E04

# Get details for a specific feature
pm feature get E04-F04

# Get JSON output for programmatic use
pm epic list --json
pm feature get E04-F02 --json
```

## Epic Commands

### List All Epics

Display all epics in your project with status and progress.

#### Command

```bash
pm epic list
```

#### Output Format

The command displays a table with the following columns:

| Column   | Description                                      |
|----------|--------------------------------------------------|
| Key      | Epic identifier (e.g., E04)                     |
| Title    | Epic title (truncated if >40 characters)        |
| Status   | Current epic status (draft, active, completed)  |
| Progress | Completion percentage with one decimal place    |
| Priority | Epic priority (low, medium, high, critical)     |

#### Example Output

```
┌─────┬──────────────────────────────────────┬───────────┬──────────┬──────────┐
│ Key │ Title                                │ Status    │ Progress │ Priority │
├─────┼──────────────────────────────────────┼───────────┼──────────┼──────────┤
│ E04 │ Task Management CLI Core             │ active    │    65.3% │ high     │
│ E05 │ Task Management CLI Capabilities     │ draft     │     0.0% │ medium   │
│ E06 │ Advanced Workflow Automation         │ draft     │     0.0% │ low      │
└─────┴──────────────────────────────────────┴───────────┴──────────┴──────────┘
```

#### JSON Output

```bash
pm epic list --json
```

```json
{
  "results": [
    {
      "id": 1,
      "key": "E04",
      "title": "Task Management CLI Core",
      "description": "Core CLI functionality for task management",
      "status": "active",
      "priority": "high",
      "business_value": "High developer productivity",
      "progress_pct": 65.3,
      "created_at": "2025-12-15T10:30:00Z",
      "updated_at": "2025-12-15T18:45:00Z"
    },
    {
      "id": 2,
      "key": "E05",
      "title": "Task Management CLI Capabilities",
      "description": "Advanced CLI features and dashboards",
      "status": "draft",
      "priority": "medium",
      "business_value": "Enhanced project visibility",
      "progress_pct": 0.0,
      "created_at": "2025-12-15T11:00:00Z",
      "updated_at": "2025-12-15T11:00:00Z"
    }
  ],
  "count": 2
}
```

#### Empty Results

If no epics exist in the database:

```bash
pm epic list
```

```
No epics found
```

Exit code: 0 (this is not an error, just empty results)

### Get Epic Details

Display detailed information about a specific epic including all its features.

#### Command

```bash
pm epic get <epic-key>
```

#### Arguments

- `<epic-key>`: The epic identifier (e.g., E04, E05)

#### Output Format

The command displays:

1. **Epic Metadata**: Key information about the epic
2. **Overall Progress**: Epic completion percentage prominently displayed
3. **Features Table**: All features in the epic with their progress

#### Example Output

```bash
pm epic get E04
```

```
Epic: E04 - Task Management CLI Core
Status: active
Priority: high
Business Value: High developer productivity
Overall Progress: 65.3%

Description:
Core CLI functionality for task management including database operations,
CLI framework, task lifecycle, and epic/feature queries.

Features:
┌───────────┬──────────────────────────────────────┬───────────┬──────────┬────────┐
│ Key       │ Title                                │ Status    │ Progress │ Tasks  │
├───────────┼──────────────────────────────────────┼───────────┼──────────┼────────┤
│ E04-F01   │ Database Foundation                  │ completed │   100.0% │     6  │
│ E04-F02   │ CLI Infrastructure                   │ completed │   100.0% │     6  │
│ E04-F03   │ Task Lifecycle Operations            │ completed │   100.0% │     4  │
│ E04-F04   │ Epic & Feature Queries               │ active    │    33.3% │     6  │
│ E04-F05   │ File Path Management                 │ draft     │     0.0% │     6  │
│ E04-F06   │ Task Creation & Templating           │ draft     │     0.0% │     4  │
│ E04-F07   │ Database Initialization & Sync       │ draft     │     0.0% │     7  │
└───────────┴──────────────────────────────────────┴───────────┴──────────┴────────┘

Total Features: 7
Total Tasks: 39
```

#### JSON Output

```bash
pm epic get E04 --json
```

```json
{
  "id": 1,
  "key": "E04",
  "title": "Task Management CLI Core",
  "description": "Core CLI functionality for task management including database operations, CLI framework, task lifecycle, and epic/feature queries.",
  "status": "active",
  "priority": "high",
  "business_value": "High developer productivity",
  "progress_pct": 65.3,
  "created_at": "2025-12-15T10:30:00Z",
  "updated_at": "2025-12-15T18:45:00Z",
  "features": [
    {
      "id": 1,
      "epic_id": 1,
      "key": "E04-F01",
      "title": "Database Foundation",
      "description": "SQLite database with Epic, Feature, Task models",
      "status": "completed",
      "progress_pct": 100.0,
      "task_count": 6,
      "created_at": "2025-12-15T10:30:00Z",
      "updated_at": "2025-12-15T14:00:00Z"
    },
    {
      "id": 4,
      "epic_id": 1,
      "key": "E04-F04",
      "title": "Epic & Feature Queries",
      "description": "CLI commands for querying epics and features with progress calculation",
      "status": "active",
      "progress_pct": 33.3,
      "task_count": 6,
      "created_at": "2025-12-15T10:30:00Z",
      "updated_at": "2025-12-15T18:45:00Z"
    }
  ]
}
```

#### Error Handling

If the epic does not exist:

```bash
pm epic get E99
```

```
Error: Epic E99 does not exist
Use 'pm epic list' to see available epics
```

Exit code: 1

## Feature Commands

### List All Features

Display all features across all epics.

#### Command

```bash
pm feature list
```

#### Output Format

The command displays a table with the following columns:

| Column   | Description                                      |
|----------|--------------------------------------------------|
| Key      | Feature identifier (e.g., E04-F04)              |
| Title    | Feature title (truncated if >40 characters)     |
| Epic     | Parent epic key                                 |
| Status   | Current feature status                          |
| Progress | Completion percentage with one decimal place    |
| Tasks    | Total number of tasks in the feature            |

#### Example Output

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F01   │ Database Foundation                  │ E04  │ completed │   100.0% │     6 │
│ E04-F02   │ CLI Infrastructure                   │ E04  │ completed │   100.0% │     6 │
│ E04-F03   │ Task Lifecycle Operations            │ E04  │ completed │   100.0% │     4 │
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
│ E04-F05   │ File Path Management                 │ E04  │ draft     │     0.0% │     6 │
│ E05-F01   │ Status Dashboard                     │ E05  │ draft     │     0.0% │     5 │
│ E05-F02   │ Dependency Management                │ E05  │ draft     │     0.0% │     4 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

#### JSON Output

```bash
pm feature list --json
```

```json
{
  "results": [
    {
      "id": 1,
      "epic_id": 1,
      "key": "E04-F01",
      "title": "Database Foundation",
      "description": "SQLite database with Epic, Feature, Task models",
      "status": "completed",
      "progress_pct": 100.0,
      "task_count": 6,
      "created_at": "2025-12-15T10:30:00Z",
      "updated_at": "2025-12-15T14:00:00Z"
    }
  ],
  "count": 7
}
```

### List Features by Epic

Filter features to show only those belonging to a specific epic.

#### Command

```bash
pm feature list --epic=<epic-key>
```

#### Flags

- `--epic=<epic-key>` or `-e <epic-key>`: Filter by epic key (e.g., E04)

#### Example Output

```bash
pm feature list --epic=E04
```

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F01   │ Database Foundation                  │ E04  │ completed │   100.0% │     6 │
│ E04-F02   │ CLI Infrastructure                   │ E04  │ completed │   100.0% │     6 │
│ E04-F03   │ Task Lifecycle Operations            │ E04  │ completed │   100.0% │     4 │
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
│ E04-F05   │ File Path Management                 │ E04  │ draft     │     0.0% │     6 │
│ E04-F06   │ Task Creation & Templating           │ E04  │ draft     │     0.0% │     4 │
│ E04-F07   │ Database Initialization & Sync       │ E04  │ draft     │     0.0% │     7 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

#### Empty Results

If no features exist for the specified epic:

```bash
pm feature list --epic=E99
```

```
No features found for Epic E99
```

Exit code: 0 (not an error)

### Filter Features by Status

Filter features to show only those with a specific status.

#### Command

```bash
pm feature list --status=<status>
```

#### Flags

- `--status=<status>` or `-s <status>`: Filter by feature status

Valid status values:
- `draft` - Feature is in planning/draft state
- `active` - Feature is being actively developed
- `completed` - Feature is complete
- `archived` - Feature is archived

#### Example Output

```bash
pm feature list --status=active
```

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

#### Error Handling

If an invalid status value is provided:

```bash
pm feature list --status=invalid
```

```
Error: Invalid status. Must be one of: draft, active, completed, archived
```

Exit code: 1

### Get Feature Details

Display detailed information about a specific feature including all its tasks.

#### Command

```bash
pm feature get <feature-key>
```

#### Arguments

- `<feature-key>`: The feature identifier (e.g., E04-F04, E05-F01)

#### Output Format

The command displays:

1. **Feature Metadata**: Key information about the feature
2. **Progress**: Feature completion percentage
3. **Task Status Breakdown**: Count of tasks by status
4. **Tasks Table**: All tasks in the feature with their status

#### Example Output

```bash
pm feature get E04-F04
```

```
Feature: E04-F04 - Epic & Feature Queries
Epic: E04 - Task Management CLI Core
Status: active
Progress: 33.3%

Description:
Implement CLI commands for querying epics and features with automatic
progress calculation built on E04-F01 (Database) and E04-F02 (CLI Framework).

Task Status Breakdown:
  Completed: 2
  Todo: 4
  In Progress: 0
  Blocked: 0

Tasks:
┌────────────────┬──────────────────────────────────────┬───────────┬──────────┬──────────┐
│ Key            │ Title                                │ Status    │ Priority │ Agent    │
├────────────────┼──────────────────────────────────────┼───────────┼──────────┼──────────┤
│ T-E04-F04-001  │ Progress Calculation Service         │ completed │ high     │ backend  │
│ T-E04-F04-002  │ Epic Query Commands                  │ todo      │ high     │ backend  │
│ T-E04-F04-003  │ Feature Query Commands               │ todo      │ high     │ backend  │
│ T-E04-F04-004  │ Unit Tests - Progress Calculation    │ completed │ medium   │ qa       │
│ T-E04-F04-005  │ Integration Tests - CLI Commands     │ todo      │ medium   │ qa       │
│ T-E04-F04-006  │ Documentation                        │ todo      │ low      │ docs     │
└────────────────┴──────────────────────────────────────┴───────────┴──────────┴──────────┘

Total Tasks: 6
```

#### JSON Output

```bash
pm feature get E04-F04 --json
```

```json
{
  "id": 4,
  "epic_id": 1,
  "key": "E04-F04",
  "title": "Epic & Feature Queries",
  "description": "Implement CLI commands for querying epics and features with automatic progress calculation built on E04-F01 (Database) and E04-F02 (CLI Framework).",
  "status": "active",
  "progress_pct": 33.3,
  "task_count": 6,
  "created_at": "2025-12-15T10:30:00Z",
  "updated_at": "2025-12-15T18:45:00Z",
  "task_breakdown": {
    "completed": 2,
    "in_progress": 0,
    "todo": 4,
    "blocked": 0
  },
  "tasks": [
    {
      "id": 19,
      "feature_id": 4,
      "key": "T-E04-F04-001",
      "title": "Progress Calculation Service",
      "description": "Implement service for calculating epic and feature progress percentages",
      "status": "completed",
      "priority": "high",
      "assigned_agent": "backend",
      "estimated_time": "4 hours",
      "created_at": "2025-12-15T10:30:00Z",
      "updated_at": "2025-12-15T15:20:00Z"
    }
  ]
}
```

#### Error Handling

If the feature does not exist:

```bash
pm feature get E04-F99
```

```
Error: Feature E04-F99 does not exist
Use 'pm feature list' to see available features
```

Exit code: 1

## Progress Calculation

Progress is automatically calculated based on task completion status. Understanding how progress is computed helps interpret the results correctly.

### Feature Progress Calculation

Feature progress is calculated as:

```
feature_progress = (completed_task_count / total_task_count) × 100
```

**What counts as "completed":**
- Tasks with status `completed`
- Tasks with status `archived`

**Example:**

```
Feature E04-F04 has 6 tasks:
- 2 completed
- 1 in_progress
- 3 todo

Progress = (2 / 6) × 100 = 33.3%
```

### Epic Progress Calculation

Epic progress is a weighted average of all feature progress values, weighted by the number of tasks in each feature.

```
epic_progress = Σ(feature_progress × feature_task_count) / Σ(feature_task_count)
```

**Example:**

```
Epic E04 has 3 features:
- Feature F01: 100% complete, 6 tasks → contributes 600 to numerator
- Feature F02: 50% complete, 4 tasks → contributes 200 to numerator
- Feature F03: 0% complete, 10 tasks → contributes 0 to numerator

Total tasks = 6 + 4 + 10 = 20
Epic progress = (600 + 200 + 0) / 20 = 40.0%
```

### Special Cases

#### Features with Zero Tasks

If a feature has no tasks:
- Feature progress = 0.0% (not null, not an error)
- Feature contributes 0% to epic progress (weighted by 0 tasks)

#### Epics with Zero Features

If an epic has no features:
- Epic progress = 0.0% (not null, not an error)

#### Features with Only Non-Completed Tasks

If all tasks are todo, in_progress, or blocked:
- Feature progress = 0.0%

### Precision

- Progress percentages are stored as floating-point numbers
- Output displays progress with one decimal place (e.g., "45.3%")
- No rounding errors that would cause progress >100%

## Output Formats

All epic and feature commands support two output formats: human-readable tables and machine-readable JSON.

### Human-Readable Format (Default)

The default output format uses formatted tables that fit in an 80-column terminal.

**Features:**
- Color-coded status values (when terminal supports colors)
- Right-aligned progress percentages for easy scanning
- Long titles truncated with "..." ellipsis
- Clear section headers and metadata display
- Consistent table styling across all commands

**Example:**

```bash
pm epic list
```

### JSON Format

Use the global `--json` flag for machine-readable output.

**Features:**
- Valid JSON that can be parsed by tools like `jq`
- All fields included (no truncation)
- Nested structures for related data
- ISO 8601 timestamps
- Consistent schema across commands

**Example:**

```bash
pm epic list --json | jq '.results[] | select(.status == "active")'
```

### Disabling Colors

Use the global `--no-color` flag to disable colored output:

```bash
pm --no-color epic list
```

This is useful for:
- Non-terminal environments (scripts, CI/CD)
- Terminals without color support
- Screen readers
- Log files

## Error Handling

The CLI provides clear error messages with helpful suggestions and follows standard Unix exit code conventions.

### Exit Codes

| Code | Meaning        | Examples                                      |
|------|----------------|-----------------------------------------------|
| 0    | Success        | Command completed successfully                |
| 1    | User Error     | Non-existent epic/feature, invalid status     |
| 2    | System Error   | Database connection failure, I/O error        |

### Common Errors

#### Non-Existent Epic

```bash
pm epic get E99
```

```
Error: Epic E99 does not exist
Use 'pm epic list' to see available epics
```

Exit code: 1

#### Non-Existent Feature

```bash
pm feature get E04-F99
```

```
Error: Feature E04-F99 does not exist
Use 'pm feature list' to see available features
```

Exit code: 1

#### Invalid Status Filter

```bash
pm feature list --status=invalid
```

```
Error: Invalid status. Must be one of: draft, active, completed, archived
```

Exit code: 1

#### Database Connection Error

```bash
pm epic list
```

```
Error: Database error. Run with --verbose for details.
```

Exit code: 2

**With verbose flag:**

```bash
pm --verbose epic list
```

```
Error: Database error
Details: unable to open database file: shark-tasks.db: no such file or directory
Suggestion: Run 'pm init' to create the database
```

Exit code: 2

### Error Suggestions

Errors include helpful suggestions when appropriate:

- "Use 'pm epic list' to see available epics"
- "Use 'pm feature list' to see available features"
- "Run 'pm init' to create the database"
- "Must be one of: draft, active, completed, archived"

## Use Cases

### Use Case 1: Daily Standup Preparation

As a developer, check what you worked on yesterday:

```bash
# See all active features
pm feature list --status=active

# Get details on the feature you're working on
pm feature get E04-F04

# Check overall epic progress
pm epic get E04
```

### Use Case 2: Stakeholder Progress Report

Generate a progress report for stakeholders:

```bash
# Get all epics with progress
pm epic list --json > epics-report.json

# Extract high-level summary
pm epic list --json | jq '.results[] | {key, title, progress_pct, status}'

# Get detailed breakdown for a specific epic
pm epic get E04 --json | jq '.features[] | {key, title, progress_pct, task_count}'
```

### Use Case 3: Identifying Bottlenecks

Find features that need attention:

```bash
# List all active features to see their progress
pm feature list --status=active

# Get details on low-progress features
pm feature get E04-F05

# Check task breakdown to identify blockers
pm feature get E04-F05 --json | jq '.task_breakdown'
```

### Use Case 4: AI Agent Context Gathering

As an AI agent, determine which epic/feature to work on:

```bash
# Get all epics with progress
EPICS=$(pm epic list --json)

# Find active epic with lowest progress
ACTIVE_EPIC=$(echo $EPICS | jq -r '.results[] | select(.status == "active") | .key' | head -1)

# Get features in that epic
pm feature list --epic=$ACTIVE_EPIC --json

# Find incomplete features
pm feature list --epic=$ACTIVE_EPIC --json | jq '.results[] | select(.progress_pct < 100)'

# Get next task to work on
pm task next --feature=E04-F04 --json
```

### Use Case 5: Project Health Check

Quickly assess overall project health:

```bash
# List all epics sorted by progress
pm epic list

# Check for stalled features (active but 0% progress)
pm feature list --status=active --json | jq '.results[] | select(.progress_pct == 0)'

# Get epic with most incomplete work
pm epic list --json | jq '.results | sort_by(.progress_pct) | .[0]'
```

## Best Practices

### For Human Users

1. **Use Human-Readable Format for Exploration**
   ```bash
   # Default table format is easier to scan
   pm epic list
   pm feature get E04-F04
   ```

2. **Use JSON for Analysis**
   ```bash
   # Pipe to jq for filtering and analysis
   pm epic list --json | jq '.results[] | select(.priority == "high")'
   ```

3. **Check Progress Regularly**
   ```bash
   # Daily check of active features
   pm feature list --status=active
   ```

4. **Drill Down from Epic to Feature to Task**
   ```bash
   # Start broad, then narrow
   pm epic list
   pm epic get E04
   pm feature get E04-F04
   pm task list --feature=E04-F04
   ```

5. **Use Filters to Reduce Noise**
   ```bash
   # Only show what's relevant
   pm feature list --epic=E04 --status=active
   ```

### For AI Agents

1. **Always Use JSON Output**
   ```bash
   # Easier to parse programmatically
   pm epic list --json
   pm feature get E04-F04 --json
   ```

2. **Check Exit Codes**
   ```bash
   # Handle errors appropriately
   if ! pm epic get E04 --json > epic.json; then
     echo "Epic not found"
     exit 1
   fi
   ```

3. **Query Before Acting**
   ```bash
   # Understand context before selecting tasks
   FEATURE=$(pm feature get E04-F04 --json)
   PROGRESS=$(echo $FEATURE | jq -r '.progress_pct')
   if (( $(echo "$PROGRESS < 50" | bc -l) )); then
     echo "Feature needs attention"
   fi
   ```

4. **Use Filters for Efficient Queries**
   ```bash
   # Don't fetch all data if you only need active features
   pm feature list --status=active --epic=E04 --json
   ```

5. **Cache Results When Appropriate**
   ```bash
   # If querying multiple times, cache the result
   EPICS=$(pm epic list --json)
   # Use $EPICS multiple times
   ```

### Performance Considerations

1. **Progress is Always Current**
   - Progress is calculated on every query
   - No need to refresh or invalidate cache
   - Always reflects current database state

2. **Efficient Queries**
   - All commands use optimized SQL with JOINs
   - No N+1 query problems
   - Fast even for large projects (100+ epics, 1000+ tasks)

3. **Filter Early**
   - Use `--epic` and `--status` filters to reduce data transfer
   - More efficient than filtering JSON output with `jq`

## Troubleshooting

### Problem: "No epics found" but I know they exist

**Possible Causes:**
- Database not initialized
- Wrong database file being queried
- Database corruption

**Solutions:**

```bash
# Check if database exists
ls -la shark-tasks.db

# Verify you're using the correct database
pm --db=shark-tasks.db epic list

# Check database content directly
sqlite3 shark-tasks.db "SELECT * FROM epics;"

# Re-initialize if needed
pm init
```

### Problem: Progress percentages seem incorrect

**Possible Causes:**
- Tasks status not updated correctly
- Misunderstanding of progress calculation
- Features with zero tasks showing 0% (expected)

**Solutions:**

```bash
# Verify task status breakdown
pm feature get E04-F04

# Check task statuses directly
pm task list --feature=E04-F04

# Verify which tasks count as complete
pm task list --feature=E04-F04 --status=completed

# Remember: only "completed" and "archived" tasks count
```

### Problem: "Database error" when running commands

**Possible Causes:**
- Database file not found
- Database file permissions
- Database corruption
- Database locked by another process

**Solutions:**

```bash
# Run with verbose flag to see details
pm --verbose epic list

# Check database file exists and is readable
ls -la shark-tasks.db

# Check file permissions
chmod 644 shark-tasks.db

# Check if database is locked
lsof shark-tasks.db

# Re-initialize database if corrupted
mv shark-tasks.db shark-tasks.db.backup
pm init
```

### Problem: Table output doesn't fit in terminal

**Possible Causes:**
- Terminal width <80 columns
- Very long epic/feature titles
- Font size too large

**Solutions:**

```bash
# Use JSON output instead
pm epic list --json | jq

# Increase terminal width
# (resize terminal window)

# Use smaller font
# (terminal settings)

# Titles are automatically truncated to fit 80 columns
# This is expected behavior
```

### Problem: JSON output is not valid JSON

**Possible Causes:**
- Using `--json` with `--verbose` (mixes output)
- Error message mixed with JSON output

**Solutions:**

```bash
# Don't use --verbose with --json
pm --json epic list

# Check exit code before parsing
if pm epic get E04 --json > output.json; then
  jq '.' output.json
else
  echo "Command failed"
fi

# Validate JSON before parsing
pm epic list --json | jq empty
```

### Problem: Progress is 0% but tasks are completed

**Possible Causes:**
- Tasks marked as "completed" in different feature
- Tasks have status "done" instead of "completed"
- Looking at wrong feature

**Solutions:**

```bash
# Verify task status exactly
pm task list --feature=E04-F04 --json | jq '.results[] | {key, status}'

# Check valid status values
pm task list --help

# Only "completed" and "archived" count as complete
# "done", "finished", etc. do NOT count
```

## Additional Resources

- [CLI Documentation](CLI.md) - Complete CLI reference
- [Database Implementation](DATABASE_IMPLEMENTATION.md) - Database schema and design
- [Task Lifecycle](../plan/E04-task-mgmt-cli-core/E04-F03-task-lifecycle/prd.md) - Task status and lifecycle
- [Progress Calculation PRD](../plan/E04-task-mgmt-cli-core/E04-F04-epic-feature-queries/prd.md) - Detailed requirements

## Feedback and Support

If you encounter issues or have suggestions:

1. Check the troubleshooting section above
2. Run commands with `--verbose` flag for detailed error messages
3. Check the database directly with `sqlite3 shark-tasks.db`
4. Review the PRD for expected behavior
5. File an issue with detailed reproduction steps

---

**Version:** 1.0.0
**Last Updated:** 2025-12-15
**Feature:** E04-F04 - Epic & Feature Queries
