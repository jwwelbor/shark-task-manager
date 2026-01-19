# Feature Commands

Commands for managing features.

## `shark feature create`

Create a new feature within an epic.

**Positional Syntax (Recommended):**
```bash
shark feature create <epic-key> "<title>" [flags]
```

**Flag Syntax (Legacy, still supported):**
```bash
shark feature create --epic=<epic-key> --title="<title>" [flags]
```

**Optional Flags:**
- `--file <path>`: Custom file path (relative to root, must include .md)
- `--force`: Reassign file if already claimed by another feature or epic
- `--execution-order <number>`: Execution order within epic
- `--json`: Output in JSON format

**Examples:**

```bash
# Create feature with positional syntax (recommended)
shark feature create E07 "Authentication"
shark feature create e07 "Authentication"  # Case insensitive
# Creates: docs/plan/E07-user-management-system/E07-F01-authentication/feature.md

# Create feature with flag syntax (legacy)
shark feature create --epic=E07 --title="Authentication"

# Create feature with custom file path
shark feature create E07 "User Profiles" --file="docs/features/profiles/feature.md"

# Create feature with execution order
shark feature create E07 "Authorization" --execution-order=2 --json

# Force reassign file
shark feature create E07 "Legacy Auth" --file="docs/legacy/auth.md" --force
```

---

## `shark feature list`

List features, optionally filtered by epic.

**Usage:**
```bash
shark feature list [EPIC] [--json]
# OR (flag syntax, backward compatible)
shark feature list [--epic=<epic-key>] [--json]
```

**Examples:**

```bash
# List all features
shark feature list

# List features in specific epic (positional argument)
shark feature list E07
shark feature list E07 --json

# List features in specific epic (flag syntax)
shark feature list --epic=E07 --json

# Using slugged epic key
shark feature list E07-user-management-system --json
```

### Health Indicators

Feature list displays health indicators in table format:

- **🔴 Red**: Feature has blocked tasks
- **🟡 Yellow**: Feature has tasks in `ready_for_approval` status for more than 3 days
- **🟢 Green**: No issues detected

**Progress Format:** Shows both weighted and completion progress:
- `70.5% | 50%` = 70.5% weighted progress, 50% completion progress
- Helps identify tasks with high weight that impact overall progress

**Notes Column:** Shows count of action items:
- Number of tasks awaiting action (in ready_for_* statuses)
- Quick indicator of workload needing attention

**Example Table Output:**
```
Epic    Feature                     Progress         Status  Notes
E07     Authentication              70.5% | 50%      🟡      4 awaiting
E07     User Management             100% | 100%      🟢      0 awaiting
E07     Permission System           45.0% | 20%      🔴      2 blocked
```

**JSON Output:** Enhanced with health indicators:
```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "epic_key": "E07",
  "progress": 70.5,
  "weighted_progress": 70.5,
  "completion_progress": 50.0,
  "health_status": "warning",
  "action_items_count": 4,
  "blocked_count": 0,
  "ready_for_approval_count": 4
}
```

---

## `shark feature get`

Get detailed information about a specific feature.

**Usage:**
```bash
shark feature get <feature-key> [--json]
```

**Supports:**
- Numeric keys: `E07-F01`, `F01`
- Slugged keys: `E07-F01-authentication`, `F01-authentication`
- Case insensitive: `e07-f01`, `f01`

**Features:**
- **Workflow-aware status display**: Task statuses are colored according to workflow config
- **Phase information**: Status breakdown includes workflow phase (planning, development, review, etc.)
- **Completion message**: Shows "All tasks completed!" when progress reaches 100%

**Examples:**

```bash
# Get feature details
shark feature get E07-F01

# Get feature details (JSON)
shark feature get E07-F01 --json

# Using partial key
shark feature get F01

# Using slugged key
shark feature get E07-F01-authentication --json
```

**Output includes:**
- Feature metadata (title, status, progress, path)
- Task status breakdown (status, count, phase) - ordered by workflow phase
- Task list with colored statuses
- Completion message if all tasks are done

### Enhanced Status Information

The feature get command includes three additional sections for improved visibility:

**Progress Breakdown:**
Shows weighted and completion progress metrics:
- **Weighted Progress**: Calculated based on configured `progress_weight` for each status
- **Completion Progress**: Raw percentage of completed tasks
- **Total Tasks**: Count of all tasks in feature

**Work Summary:**
Categorizes tasks by responsibility:
- **Completed**: Finished and approved tasks
- **Agent Work**: Tasks assigned to AI agents
- **Human Work**: Tasks requiring human engineers
- **QA Work**: Tasks for quality assurance team
- **Blocked Work**: Tasks blocked by dependencies
- **Not Started**: Todo tasks

**Action Items:**
Lists tasks awaiting action, grouped by status:
- Tasks in `ready_for_approval` status
- Tasks in `ready_for_qa` status
- Other actionable statuses from workflow config

**Example Output (Table Format):**
```
Progress Breakdown
  Weighted: 70.5% | Completion: 9.1% | Total: 11 tasks

Work Summary
  Completed: 1 | Agent Work: 0 | Human Work: 7 | QA Work: 0 | Blocked: 0 | Not Started: 3

Action Items
  Ready for Approval (4 tasks)
    - T-E07-F23-001
    - T-E07-F23-002
    - T-E07-F23-003
    - T-E07-F23-004
```

**JSON Output:**
```json
{
  "id": 1,
  "epic_id": 7,
  "key": "E07-F01",
  "title": "Authentication",
  "status": "active",
  "progress_pct": 75.0,
  "progress_info": {
    "weighted_progress_pct": 70.5,
    "completion_progress_pct": 9.1,
    "total_tasks": 11
  },
  "work_summary": {
    "total_tasks": 11,
    "completed_tasks": 1,
    "agent_work": 0,
    "human_work": 7,
    "qa_work": 0,
    "blocked_work": 0,
    "not_started": 3
  },
  "action_items": {
    "ready_for_approval": ["T-E07-F23-001", "T-E07-F23-002"],
    "ready_for_qa": []
  },
  "tasks": [...],
  "status_breakdown": [
    {"status": "completed", "count": 3, "phase": "done", "color": "green"},
    {"status": "in_progress", "count": 1, "phase": "development", "color": "blue"}
  ]
}
```

## Related Documentation

- [Epic Commands](epic-commands.md)
- [Task Commands](task-commands.md)
- [Key Formats](key-formats.md) - Case insensitive and slugged keys
- [File Paths](file-paths.md) - Custom file path organization
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
