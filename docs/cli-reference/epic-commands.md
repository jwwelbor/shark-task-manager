# Epic Commands

Commands for managing epics.

## `shark epic create`

Create a new epic.

**Required Flags:**
- `--title <string>`: Epic title

**Optional Flags:**
- `--file <path>`: Custom file path (relative to root, must include .md)
- `--force`: Reassign file if already claimed by another epic or feature
- `--priority <1-10>`: Priority (1 = highest, 10 = lowest)
- `--business-value <1-10>`: Business value score
- `--json`: Output in JSON format

**Examples:**

```bash
# Create epic with default file path
shark epic create --title="User Management System"
# Creates: docs/plan/E07-user-management-system/epic.md

# Create epic with custom file path
shark epic create --title="Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Create epic with priority and business value
shark epic create --title="Payment Integration" --priority=1 --business-value=10 --json

# Force reassign file (if already claimed)
shark epic create --title="Legacy Migration" --file="docs/legacy/epic.md" --force
```

---

## `shark epic list`

List all epics with progress information.

**Flags:**
- `--json`: Output in JSON format

**Examples:**

```bash
# List all epics (table format)
shark epic list

# List all epics (JSON format)
shark epic list --json
```

**JSON Output:**
```json
[
  {
    "id": 7,
    "key": "E07",
    "slug": "user-management-system",
    "title": "User Management System",
    "description": "Complete user management infrastructure",
    "file_path": "docs/plan/E07-user-management-system/epic.md",
    "priority": 1,
    "business_value": 10,
    "progress": 42.5,
    "created_at": "2026-01-02T10:00:00Z",
    "updated_at": "2026-01-02T15:30:00Z"
  }
]
```

---

## `shark epic get`

Get detailed information about a specific epic.

**Usage:**
```bash
shark epic get <epic-key> [--json]
```

**Supports:**
- Numeric keys: `E07`
- Slugged keys: `E07-user-management-system`
- Case insensitive: `e07`, `e07-user-management-system`

**Examples:**

```bash
# Get epic details (table format)
shark epic get E07

# Get epic details (JSON format)
shark epic get E07 --json

# Using slugged key
shark epic get E07-user-management-system --json
```

### Epic Rollups

Epic get includes comprehensive rollup information for visibility across all features:

**Feature Status Rollup:**
Shows the distribution of features by status across the epic:
- Counts features in each workflow status
- Helps understand feature progression
- Example: `In Planning: 2, In Development: 3, In Review: 1, Completed: 4`

**Task Status Rollup:**
Aggregates task counts across all features:
- Total task count across entire epic
- Breakdown by status (todo, in_progress, ready_for_*, completed, blocked)
- Provides full workflow visibility
- Example: `Total: 47 tasks | Todo: 10 | In Progress: 15 | Ready for Review: 8 | Completed: 12 | Blocked: 2`

**Impediments:**
Lists all blocked tasks that are impeding progress:
- Blocked task key and title
- Parent feature
- Blocker reason
- Age of blockage
- Enables quick identification and resolution of blockers

**Example Table Output:**
```
Feature Status Rollup
  In Planning: 1 | In Development: 2 | In Review: 1 | Completed: 3

Task Status Rollup
  Total: 47 tasks
  Todo: 10 | In Progress: 15 | Ready for Review: 8 | Completed: 12 | Blocked: 2

Impediments
  🔴 T-E07-F01-005 "Setup OAuth providers" (Feature: Authentication, 2 days)
  🔴 T-E07-F03-012 "Configure Postgres replication" (Feature: Database, 5 days)
```

**JSON Output:** Enhanced with rollup data:
```json
{
  "id": 7,
  "key": "E07",
  "title": "User Management System",
  "progress": 55.0,
  "feature_status_rollup": {
    "in_planning": 1,
    "in_development": 2,
    "in_review": 1,
    "completed": 3
  },
  "task_status_rollup": {
    "total": 47,
    "todo": 10,
    "in_progress": 15,
    "ready_for_approval": 5,
    "ready_for_qa": 3,
    "completed": 12,
    "blocked": 2
  },
  "impediments": [
    {
      "task_key": "T-E07-F01-005",
      "task_title": "Setup OAuth providers",
      "feature_key": "E07-F01",
      "feature_title": "Authentication",
      "reason": "Waiting for OAuth provider approval",
      "blocked_since": "2026-01-14T10:00:00Z",
      "age_days": 2
    }
  ]
}
```

## Related Documentation

- [Feature Commands](feature-commands.md)
- [Task Commands](task-commands.md)
- [Key Formats](key-formats.md) - Case insensitive and slugged keys
- [File Paths](file-paths.md) - Custom file path organization
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
