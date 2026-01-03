# Status Calculation Rules

**Feature**: E07-F14 - Cascading Status Calculation
**Document Type**: Technical Reference

---

## Quick Reference

### Task Status Values
| Status | Description | Counts As |
|--------|-------------|-----------|
| `todo` | Not started | Incomplete |
| `in_progress` | Work ongoing | Active |
| `blocked` | Waiting on dependency | Active |
| `ready_for_review` | Awaiting approval | Active |
| `completed` | Approved and done | Complete |
| `archived` | No longer relevant | Complete |

### Feature Status Calculation Matrix

| Task Composition | Calculated Feature Status |
|-----------------|---------------------------|
| No tasks | `draft` |
| All `todo` | `draft` |
| Any `in_progress` | `active` |
| Any `ready_for_review` | `active` |
| Any `blocked` | `active` |
| All `completed` | `completed` |
| All `archived` | `completed` |
| Mix of `completed` + `archived` | `completed` |
| Mix of `todo` + `completed` | `active`* |

*Note: Mix of todo + completed without any active tasks is edge case - treated as active since work is partially done.

### Epic Status Calculation Matrix

| Feature Composition | Calculated Epic Status |
|---------------------|------------------------|
| No features | `draft` |
| All `draft` | `draft` |
| Any `active` | `active` |
| Any `blocked` | `active` |
| All `completed` | `completed` |
| All `archived` | `completed` |
| Mix of `completed` + `archived` | `completed` |

---

## Calculation Algorithms

### Feature Status Algorithm (Pseudocode)

```
function calculateFeatureStatus(feature):
    tasks = getTasksForFeature(feature.id)
    
    if tasks.isEmpty():
        return "draft"
    
    activeCount = count(tasks where status in [in_progress, ready_for_review, blocked])
    completedCount = count(tasks where status in [completed, archived])
    totalCount = tasks.length
    
    if completedCount == totalCount:
        return "completed"
    
    if activeCount > 0:
        return "active"
    
    // All tasks are in todo (no active, no completed)
    return "draft"
```

### Epic Status Algorithm (Pseudocode)

```
function calculateEpicStatus(epic):
    features = getFeaturesForEpic(epic.id)
    
    if features.isEmpty():
        return "draft"
    
    activeCount = count(features where status in [active, blocked])
    completedCount = count(features where status in [completed, archived])
    totalCount = features.length
    
    if completedCount == totalCount:
        return "completed"
    
    if activeCount > 0:
        return "active"
    
    // All features are in draft (no active, no completed)
    return "draft"
```

---

## SQL Queries for Calculation

### Feature Status Calculation Query

```sql
SELECT
    CASE
        -- No tasks = draft
        WHEN COUNT(*) = 0 THEN 'draft'
        
        -- All completed/archived = completed
        WHEN COUNT(*) = SUM(
            CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END
        ) THEN 'completed'
        
        -- Any active status = active
        WHEN SUM(
            CASE WHEN status IN ('in_progress', 'ready_for_review', 'blocked') THEN 1 ELSE 0 END
        ) > 0 THEN 'active'
        
        -- All todo = draft
        ELSE 'draft'
    END as calculated_status
FROM tasks
WHERE feature_id = ?;
```

### Epic Status Calculation Query

```sql
SELECT
    CASE
        -- No features = draft
        WHEN COUNT(*) = 0 THEN 'draft'
        
        -- All completed/archived = completed
        WHEN COUNT(*) = SUM(
            CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END
        ) THEN 'completed'
        
        -- Any active status = active
        WHEN SUM(
            CASE WHEN status IN ('active', 'blocked') THEN 1 ELSE 0 END
        ) > 0 THEN 'active'
        
        -- All draft = draft
        ELSE 'draft'
    END as calculated_status
FROM features
WHERE epic_id = ?;
```

---

## Cascade Trigger Points

### Task Status Change Cascade

```
shark task start <key>
  |
  +-> Task status: todo -> in_progress
  |
  +-> Trigger: RecalculateFeatureStatus(task.feature_id)
        |
        +-> Feature status updated if changed
        |
        +-> Trigger: RecalculateEpicStatus(feature.epic_id)
              |
              +-> Epic status updated if changed
```

### Task Create Cascade

```
shark task create <title> --epic=E## --feature=F##
  |
  +-> Task created with status: todo
  |
  +-> Trigger: RecalculateFeatureStatus(feature_id)
        |
        +-> If feature was "completed", changes to "active"
        |
        +-> Trigger: RecalculateEpicStatus(epic_id)
              |
              +-> If epic was "completed", changes to "active"
```

### Task Delete Cascade

```
shark task delete <key>
  |
  +-> Task deleted
  |
  +-> Trigger: RecalculateFeatureStatus(feature_id)
        |
        +-> Recalculate based on remaining tasks
        |
        +-> Trigger: RecalculateEpicStatus(epic_id)
              |
              +-> Recalculate based on updated features
```

### Feature Status Change Cascade

```
shark feature update <key> --status=<status>
  |
  +-> IF status != "auto":
  |     +-> Set status_override = true
  |     +-> Set status = <status>
  |
  +-> ELSE (status == "auto"):
  |     +-> Set status_override = false
  |     +-> RecalculateFeatureStatus(feature_id)
  |
  +-> Trigger: RecalculateEpicStatus(epic_id)
        |
        +-> Epic status updated based on all feature statuses
```

---

## Override Behavior

### Status Override Field

| status_override | status_source | Behavior |
|-----------------|---------------|----------|
| false (default) | "calculated" | Status auto-calculated from children |
| true | "manual" | Status manually set, children don't affect it |

### Override Commands

```bash
# Set manual override
shark feature update E07-F14 --status=blocked
shark epic update E07 --status=archived

# Clear manual override (return to calculated)
shark feature update E07-F14 --status=auto
shark epic update E07 --status=auto
```

### Override + Cascade Interaction

When a parent has status_override=true:
1. Child status changes still trigger recalculation
2. Calculated status is computed but NOT applied
3. Override status remains in effect
4. User must explicitly clear override to see calculated status

---

## Special Cases Reference

### Blocked Status Handling

- **Task blocked**: Counts as "active work" for feature calculation
- **Feature blocked**: Manual override, not auto-calculated
- **Rationale**: Blocked means work started but paused, not "not started"

### Archived Status Handling

- **Task archived**: Counts as "completed" for feature calculation
- **Feature archived**: Manual override, not auto-calculated
- **Epic archived**: Manual override only
- **Rationale**: Archived is a form of completion (decided not to do it)

### Progress vs Status

| Scenario | Progress | Status |
|----------|----------|--------|
| 2/4 tasks completed, 2 todo | 50% | draft |
| 2/4 tasks completed, 1 in_progress, 1 todo | 50% | active |
| 4/4 tasks completed | 100% | completed |
| 0/4 tasks completed, 1 in_progress | 0% | active |

Progress and status are related but independent:
- Progress = completed_count / total_count
- Status = derived from task state composition
