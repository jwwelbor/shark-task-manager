# Enhanced JSON API Response Fields

Enhanced JSON response fields for improved status tracking and visibility.

## Feature Get Enhanced Fields

The `shark feature get` command returns additional fields for comprehensive status visibility:

### Progress Information

```json
"progress_info": {
  "weighted_progress_pct": 70.5,
  "completion_progress_pct": 9.1,
  "total_tasks": 11
}
```

- **weighted_progress_pct**: Progress calculated based on `progress_weight` configuration
- **completion_progress_pct**: Raw percentage of completed tasks
- **total_tasks**: Total count of tasks in feature

### Work Summary (by responsibility)

```json
"work_summary": {
  "total_tasks": 11,
  "completed_tasks": 1,
  "agent_work": 0,
  "human_work": 7,
  "qa_work": 0,
  "blocked_work": 0,
  "not_started": 3
}
```

- **total_tasks**: Total count of all tasks
- **completed_tasks**: Tasks in completed status
- **agent_work**: Tasks assigned to AI agents
- **human_work**: Tasks requiring human engineers
- **qa_work**: Tasks for quality assurance team
- **blocked_work**: Tasks blocked by dependencies
- **not_started**: Tasks in todo status

### Action Items (by status)

```json
"action_items": {
  "ready_for_approval": ["T-E07-F23-001", "T-E07-F23-002"],
  "ready_for_qa": ["T-E07-F23-006"]
}
```

Lists task keys awaiting action, grouped by actionable status.

### Complete Feature Response

```json
{
  "id": 1,
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
    "ready_for_qa": ["T-E07-F23-006"]
  }
}
```

---

## Feature List Enhanced Fields

The `shark feature list` command returns health indicators and dual progress metrics:

```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "progress": 70.5,
  "weighted_progress": 70.5,
  "completion_progress": 50.0,
  "health_status": "warning",
  "action_items_count": 4,
  "blocked_count": 0,
  "ready_for_approval_count": 4
}
```

### Health Status Values

- `"healthy"`: No blockers, all approval tasks < 3 days
- `"warning"`: Ready for approval tasks > 3 days, or minor blockers
- `"critical"`: Multiple blockers or high-priority tasks blocked

---

## Epic Get Enhanced Fields

The `shark epic get` command returns rollup information across all features:

### Feature Status Rollup

```json
"feature_status_rollup": {
  "in_planning": 1,
  "in_development": 2,
  "in_review": 1,
  "completed": 3
}
```

Shows the distribution of features by status across the epic.

### Task Status Rollup

```json
"task_status_rollup": {
  "total": 47,
  "todo": 10,
  "in_progress": 15,
  "ready_for_approval": 5,
  "ready_for_qa": 3,
  "completed": 12,
  "blocked": 2
}
```

Aggregates task counts across all features in the epic.

### Impediments (Blocked Tasks)

```json
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
```

Lists all blocked tasks impeding progress with:
- Task key and title
- Parent feature
- Blocker reason
- Age of blockage

### Complete Epic Response

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

---

## Configuration-Driven Calculations

All enhanced fields are calculated based on status configuration in `.sharkconfig.json`:

### Status Metadata Configuration

```json
{
  "status_metadata": {
    "completed": {
      "color": "green",
      "phase": "done",
      "progress_weight": 100,
      "responsibility": "none",
      "blocks_feature": false
    },
    "ready_for_approval": {
      "color": "magenta",
      "phase": "review",
      "progress_weight": 75,
      "responsibility": "human",
      "blocks_feature": true
    },
    "in_development": {
      "color": "yellow",
      "phase": "development",
      "progress_weight": 50,
      "responsibility": "agent",
      "blocks_feature": false
    }
  }
}
```

### Field Calculations

1. **Weighted Progress**: `(sum of progress_weight * task_count) / total_tasks * 100`
2. **Completion Progress**: `(completed_tasks / total_tasks) * 100`
3. **Work Breakdown**: Grouped by `responsibility` field (agent, human, qa_team, none)
4. **Health Status**: Based on `blocks_feature` statuses and age of approval tasks
5. **Action Items**: Tasks in statuses with `blocks_feature: true`

## Related Documentation

- [Epic Commands](epic-commands.md) - Epic rollup fields
- [Feature Commands](feature-commands.md) - Feature enhanced fields
- [Workflow Configuration](workflow-config.md) - Status metadata configuration
- [JSON Output](json-output.md) - Basic JSON response format
