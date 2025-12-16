# Epic & Feature Commands - Quick Reference

Fast reference guide for epic and feature query commands.

## Commands at a Glance

| Command | Description | Example |
|---------|-------------|---------|
| `pm epic list` | List all epics | `pm epic list` |
| `pm epic get <key>` | Get epic details | `pm epic get E04` |
| `pm feature list` | List all features | `pm feature list` |
| `pm feature list --epic=<key>` | List features in epic | `pm feature list --epic=E04` |
| `pm feature list --status=<status>` | Filter by status | `pm feature list --status=active` |
| `pm feature get <key>` | Get feature details | `pm feature get E04-F04` |

## Common Use Cases

### Check Project Progress

```bash
# See all epics
pm epic list

# Drill into specific epic
pm epic get E04

# See active features
pm feature list --status=active
```

### Find Next Work Item

```bash
# List active features
pm feature list --status=active

# Get feature details to see tasks
pm feature get E04-F04

# Get next available task
pm task next --feature=E04-F04
```

### Generate Report for Stakeholders

```bash
# Get all epics with progress (JSON)
pm epic list --json > report.json

# Extract summary
pm epic list --json | jq '.results[] | {key, title, progress_pct}'

# Get detailed epic breakdown
pm epic get E04 --json | jq '.features[] | {key, title, progress_pct, task_count}'
```

### AI Agent Workflow

```bash
# 1. Find active epic
EPIC=$(pm epic list --json | jq -r '.results[] | select(.status == "active") | .key' | head -1)

# 2. Get features in epic
pm feature list --epic=$EPIC --json

# 3. Find incomplete feature
FEATURE=$(pm feature list --epic=$EPIC --json | jq -r '.results[] | select(.progress_pct < 100) | .key' | head -1)

# 4. Get next task
pm task next --feature=$FEATURE --json
```

## Flags Reference

### Global Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--json` | JSON output | `pm --json epic list` |
| `--no-color` | Disable colors | `pm --no-color epic list` |
| `--verbose` | Verbose output | `pm --verbose epic list` |
| `--db <path>` | Database file | `pm --db=custom.db epic list` |

### Command-Specific Flags

| Command | Flag | Values | Example |
|---------|------|--------|---------|
| `pm feature list` | `--epic` | Epic key (e.g., E04) | `pm feature list --epic=E04` |
| `pm feature list` | `--status` | draft, active, completed, archived | `pm feature list --status=active` |

## Exit Codes

| Code | Meaning | Example |
|------|---------|---------|
| 0 | Success | Command completed |
| 1 | User error | Non-existent epic/feature, invalid status |
| 2 | System error | Database error, I/O error |

## Progress Calculation

### Feature Progress
```
progress = (completed_tasks / total_tasks) × 100
```

Only tasks with status `completed` or `archived` count as complete.

### Epic Progress
```
epic_progress = Σ(feature_progress × feature_task_count) / Σ(feature_task_count)
```

Weighted average of feature progress, weighted by task count.

## Example Outputs

### Epic List

```
┌─────┬──────────────────────────────────────┬───────────┬──────────┬──────────┐
│ Key │ Title                                │ Status    │ Progress │ Priority │
├─────┼──────────────────────────────────────┼───────────┼──────────┼──────────┤
│ E04 │ Task Management CLI Core             │ active    │    65.3% │ high     │
│ E05 │ Task Management CLI Capabilities     │ draft     │     0.0% │ medium   │
└─────┴──────────────────────────────────────┴───────────┴──────────┴──────────┘
```

### Feature List

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F01   │ Database Foundation                  │ E04  │ completed │   100.0% │     6 │
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

### Epic Details

```
Epic: E04 - Task Management CLI Core
Status: active
Priority: high
Overall Progress: 65.3%

Features:
┌───────────┬──────────────────────────────────────┬───────────┬──────────┬────────┐
│ Key       │ Title                                │ Status    │ Progress │ Tasks  │
├───────────┼──────────────────────────────────────┼───────────┼──────────┼────────┤
│ E04-F01   │ Database Foundation                  │ completed │   100.0% │     6  │
│ E04-F04   │ Epic & Feature Queries               │ active    │    33.3% │     6  │
└───────────┴──────────────────────────────────────┴───────────┴──────────┴────────┘
```

### Feature Details

```
Feature: E04-F04 - Epic & Feature Queries
Epic: E04 - Task Management CLI Core
Status: active
Progress: 33.3%

Task Status Breakdown:
  Completed: 2
  Todo: 4
  In Progress: 0
  Blocked: 0

Tasks:
┌────────────────┬──────────────────────────────────────┬───────────┬──────────┐
│ Key            │ Title                                │ Status    │ Priority │
├────────────────┼──────────────────────────────────────┼───────────┼──────────┤
│ T-E04-F04-001  │ Progress Calculation Service         │ completed │ high     │
│ T-E04-F04-002  │ Epic Query Commands                  │ todo      │ high     │
└────────────────┴──────────────────────────────────────┴───────────┴──────────┘
```

## JSON Output Examples

### Epic List JSON

```bash
pm epic list --json
```

```json
{
  "results": [
    {
      "key": "E04",
      "title": "Task Management CLI Core",
      "status": "active",
      "progress_pct": 65.3,
      "priority": "high"
    }
  ],
  "count": 1
}
```

### Feature Get JSON

```bash
pm feature get E04-F04 --json
```

```json
{
  "key": "E04-F04",
  "title": "Epic & Feature Queries",
  "epic_id": 1,
  "status": "active",
  "progress_pct": 33.3,
  "task_count": 6,
  "task_breakdown": {
    "completed": 2,
    "in_progress": 0,
    "todo": 4,
    "blocked": 0
  },
  "tasks": [...]
}
```

## JQ Recipes

### Filter Active Epics

```bash
pm epic list --json | jq '.results[] | select(.status == "active")'
```

### Get Epic with Lowest Progress

```bash
pm epic list --json | jq '.results | sort_by(.progress_pct) | .[0]'
```

### Extract Feature Keys from Epic

```bash
pm epic get E04 --json | jq -r '.features[].key'
```

### Count Tasks by Status

```bash
pm feature get E04-F04 --json | jq '.task_breakdown'
```

### Find Incomplete Active Features

```bash
pm feature list --status=active --json | jq '.results[] | select(.progress_pct < 100)'
```

### Get High Priority Epics

```bash
pm epic list --json | jq '.results[] | select(.priority == "high")'
```

## Common Errors

### Epic Not Found

```
Error: Epic E99 does not exist
Use 'pm epic list' to see available epics
```

**Fix:** Check epic key with `pm epic list`

### Feature Not Found

```
Error: Feature E04-F99 does not exist
Use 'pm feature list' to see available features
```

**Fix:** Check feature key with `pm feature list`

### Invalid Status

```
Error: Invalid status. Must be one of: draft, active, completed, archived
```

**Fix:** Use valid status value

### Database Error

```
Error: Database error. Run with --verbose for details.
```

**Fix:** Run with `--verbose` flag, check database file exists

## Troubleshooting

### No results but data exists

```bash
# Check database file location
pm --db=shark-tasks.db epic list

# Verify database content
sqlite3 shark-tasks.db "SELECT * FROM epics;"
```

### Progress seems wrong

```bash
# Check task status breakdown
pm feature get E04-F04

# Verify task statuses
pm task list --feature=E04-F04

# Remember: only "completed" and "archived" count
```

### JSON parsing fails

```bash
# Don't use --verbose with --json
pm --json epic list  # Good
pm --verbose --json epic list  # Bad (mixes output)

# Check exit code first
if pm epic get E04 --json > out.json; then
  jq '.' out.json
fi
```

## Tips

1. **Start broad, drill down**: `epic list` → `epic get E04` → `feature get E04-F04`
2. **Use filters**: More efficient than filtering JSON with jq
3. **Check exit codes**: In scripts, always check command success
4. **Cache results**: If querying repeatedly, save JSON output
5. **Use --json for automation**: Easier to parse than table output

## See Also

- [Full Documentation](EPIC_FEATURE_QUERIES.md) - Complete guide with examples
- [CLI Documentation](CLI.md) - All CLI commands
- [Database Schema](DATABASE_IMPLEMENTATION.md) - Database structure
- [PRD](../plan/E04-task-mgmt-cli-core/E04-F04-epic-feature-queries/prd.md) - Requirements

---

**Quick Reference Version:** 1.0.0
