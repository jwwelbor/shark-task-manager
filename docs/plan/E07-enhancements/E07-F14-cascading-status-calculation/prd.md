# PRD: Cascading Status Calculation

**Epic**: E07 - Enhancements
**Feature**: E07-F14 - Cascading Status Calculation
**Status**: Draft
**Priority**: High
**Complexity**: L (Large - requires database triggers, repository changes, CLI updates)

## Overview

Implement automatic status calculation where parent entities (Features, Epics) derive their status from their children (Tasks, Features respectively). Status changes cascade upward through the hierarchy, reducing manual status management while preserving the ability to override status when needed.

### Problem Statement

Currently, Epic and Feature statuses must be managed manually. This leads to:
- Stale status values that don't reflect actual work state
- Inconsistent status across the hierarchy (e.g., completed tasks but "draft" feature)
- Extra cognitive load for users to update parent statuses
- Reduced trust in dashboard/reporting accuracy

### Solution

Automatic status calculation that:
1. Derives Feature status from Task statuses
2. Derives Epic status from Feature statuses
3. Triggers recalculation on any child status change
4. Allows manual override of calculated status
5. Re-evaluates status when children are added/removed

---

## User Stories

### Story 1: Automatic Feature Status from Tasks

**As a** project manager
**I want** feature status to automatically update based on task progress
**So that** I always see accurate feature status without manual updates

**Acceptance Criteria**:

| ID | Given | When | Then |
|----|-------|------|------|
| AC1.1 | A feature with all tasks in "todo" status | I view the feature | Status shows "draft" |
| AC1.2 | A feature with at least one task "in_progress" or "ready_for_review" | Any task starts work | Status shows "active" |
| AC1.3 | A feature with all tasks "completed" or "archived" | Last task completes | Status shows "completed" |
| AC1.4 | A completed feature | A new task is added | Status returns to "active" |
| AC1.5 | A completed feature | An existing task is reopened | Status returns to "active" |
| AC1.6 | A feature with no tasks | I view the feature | Status shows "draft" |

---

### Story 2: Automatic Epic Status from Features

**As a** project manager
**I want** epic status to automatically update based on feature statuses
**So that** I can see overall epic progress at a glance

**Acceptance Criteria**:

| ID | Given | When | Then |
|----|-------|------|------|
| AC2.1 | An epic with all features in "draft" status | I view the epic | Status shows "draft" |
| AC2.2 | An epic with at least one feature "active" | Any feature becomes active | Status shows "active" |
| AC2.3 | An epic with all features "completed" or "archived" | Last feature completes | Status shows "completed" |
| AC2.4 | A completed epic | A new feature is added | Status returns to "active" |
| AC2.5 | A completed epic | Any feature status changes to non-completed | Status returns to "active" |
| AC2.6 | An epic with no features | I view the epic | Status shows "draft" |

---

### Story 3: Manual Status Override

**As a** project manager
**I want to** manually override calculated status when needed
**So that** I can mark work as blocked, archived, or adjust for special circumstances

**Acceptance Criteria**:

| ID | Given | When | Then |
|----|-------|------|------|
| AC3.1 | A feature with calculated "active" status | I run `shark feature update --status=blocked` | Status changes to "blocked" |
| AC3.2 | A feature with manual override | A task status changes | Calculated status is shown but override is noted |
| AC3.3 | A feature with manual override | I run `shark feature update --status=auto` | Override is cleared, calculated status applies |
| AC3.4 | An epic with manual override | I run `shark epic update --status=archived` | Status changes to "archived" |
| AC3.5 | A feature/epic in any state | I use `--force` with status update | Override is applied regardless of calculated value |

---

### Story 4: Cascading Updates on Child Changes

**As a** developer
**I want** parent status to update automatically when I change task status
**So that** I don't need to manually update feature and epic status

**Acceptance Criteria**:

| ID | Given | When | Then |
|----|-------|------|------|
| AC4.1 | A feature with one in_progress task | I complete the last task | Feature recalculates (may become "completed") |
| AC4.2 | A feature in epic E07 | Feature status changes to "active" | Epic E07 recalculates (may become "active") |
| AC4.3 | An epic with 3 active features | One feature completes | Epic remains "active" (not all completed) |
| AC4.4 | An epic with 3 features all completed | I view the epic | Epic shows "completed" |
| AC4.5 | A task is deleted | Parent feature recalculates | Status reflects remaining tasks |
| AC4.6 | A feature is deleted | Parent epic recalculates | Status reflects remaining features |

---

### Story 5: Status Visibility in CLI Output

**As a** user
**I want to** see calculated status vs. manual override clearly
**So that** I understand why a status has its current value

**Acceptance Criteria**:

| ID | Given | When | Then |
|----|-------|------|------|
| AC5.1 | A feature with calculated status | I run `shark feature get E07-F01` | Shows "status: active (calculated)" |
| AC5.2 | A feature with manual override | I run `shark feature get E07-F01` | Shows "status: blocked (manual override)" |
| AC5.3 | A feature with calculated status | I run `shark feature list --json` | JSON includes `status_source: "calculated"` |
| AC5.4 | A feature with manual override | I run `shark feature list --json` | JSON includes `status_source: "manual"` |

---

## Status Calculation Rules

### Task Status Values (Existing)
- `todo` - Not started
- `in_progress` - Work ongoing
- `blocked` - Waiting on external dependency
- `ready_for_review` - Awaiting approval
- `completed` - Approved and done
- `archived` - No longer relevant

### Feature Status Calculation

```
Feature Status = f(task_statuses)

IF no tasks:
  status = "draft"

ELSE IF all tasks in {completed, archived}:
  status = "completed"

ELSE IF any task in {in_progress, ready_for_review, blocked}:
  status = "active"

ELSE (all tasks in todo):
  status = "draft"
```

**Special Cases:**
- `blocked` tasks count as "active" work (the feature is actively being worked, just stuck)
- `archived` tasks are treated as "completed" (work is done, just deprecated)

### Epic Status Calculation

```
Epic Status = f(feature_statuses)

IF no features:
  status = "draft"

ELSE IF all features in {completed, archived}:
  status = "completed"

ELSE IF any feature in {active, blocked}:
  status = "active"

ELSE (all features in draft):
  status = "draft"
```

### Cascade Trigger Points

Status recalculation MUST occur when:

1. **Task level changes:**
   - Task created
   - Task deleted
   - Task status updated (start, complete, block, unblock, reopen)

2. **Feature level changes:**
   - Feature created
   - Feature deleted
   - Feature status manually overridden

3. **Manual override scenarios:**
   - `shark feature update --status=<status>` sets manual override
   - `shark feature update --status=auto` clears override
   - `shark epic update --status=<status>` sets manual override
   - `shark epic update --status=auto` clears override

---

## Edge Cases and Exceptional Scenarios

### Edge Case 1: Empty Containers
| Scenario | Expected Behavior |
|----------|-------------------|
| Epic with no features | Status = "draft" |
| Feature with no tasks | Status = "draft" |
| Epic with features but no tasks in any feature | Status derived from feature statuses (likely "draft") |

### Edge Case 2: Blocked Tasks/Features
| Scenario | Expected Behavior |
|----------|-------------------|
| Feature with all tasks blocked | Status = "active" (blocked is active work) |
| Feature with mix of blocked + completed | Status = "active" |
| Epic with one blocked feature | Status = "active" |

### Edge Case 3: Archived Items
| Scenario | Expected Behavior |
|----------|-------------------|
| Feature with all tasks archived | Status = "completed" (archived = done) |
| Epic with all features archived | Status = "completed" |
| Mix of completed + archived | Status = "completed" |

### Edge Case 4: Status Transitions
| Scenario | Expected Behavior |
|----------|-------------------|
| Completed feature, task added | Status changes to "active" |
| Completed feature, task reopened | Status changes to "active" |
| Completed epic, feature added | Status changes to "active" |
| Active epic, all features complete | Status changes to "completed" |

### Edge Case 5: Manual Override Interactions
| Scenario | Expected Behavior |
|----------|-------------------|
| Override + child change | Calculated status updates, override preserved |
| Override to "archived" | Override applied, no cascade up |
| Override cleared | Recalculate from children immediately |
| Force override to invalid state | Error: "Cannot set completed status with incomplete children" (unless --force) |

### Edge Case 6: Concurrent Updates
| Scenario | Expected Behavior |
|----------|-------------------|
| Two tasks complete simultaneously | Both trigger recalc, last one determines final state |
| Status update during recalculation | Use optimistic locking or last-write-wins |
| Database triggers + application logic | Prefer database triggers for atomicity |

### Edge Case 7: Progress vs Status Mismatch
| Scenario | Expected Behavior |
|----------|-------------------|
| Feature 50% progress, all active tasks | Status = "active", progress = 50% |
| Feature 100% progress, manual "draft" override | Status = "draft" (override), progress = 100% |
| Epic 75% progress, one active feature | Status = "active", progress = 75% |

---

## Technical Requirements

### Functional Requirements

#### FR1: Database Schema Changes
- Add `status_override` boolean column to `features` table
- Add `status_override` boolean column to `epics` table
- Add triggers for status recalculation (optional, can be in application layer)

#### FR2: Repository Layer Changes
- Add `RecalculateFeatureStatus(ctx, featureID)` method
- Add `RecalculateEpicStatus(ctx, epicID)` method
- Modify existing task status change methods to trigger feature recalc
- Modify existing feature status change methods to trigger epic recalc

#### FR3: CLI Updates
- Add `--status=auto` option to clear override
- Update `feature get` and `epic get` to show status source
- Update JSON output to include `status_source` field

#### FR4: Cascading Update Chain
```
task.UpdateStatus()
  -> feature.RecalculateStatus()
  -> epic.RecalculateStatus()

feature.Create()
  -> epic.RecalculateStatus()

feature.Delete()
  -> epic.RecalculateStatus()
```

### Non-Functional Requirements

#### NFR1: Performance
- Status recalculation should complete in < 100ms for epics with 50+ features
- Use single query aggregation, not N+1 queries
- Consider caching calculated status for large hierarchies

#### NFR2: Consistency
- All cascading updates within a single transaction
- No partial state updates (all or nothing)
- Use database triggers for atomicity where possible

#### NFR3: Auditability
- Log status changes with source (calculated vs manual)
- Include triggering event in logs
- Track override history if needed

#### NFR4: Backward Compatibility
- Existing epics/features without override column treated as calculated
- Migration should not change current status values
- `status_override = false` by default

---

## Implementation Approach

### Option A: Database Triggers (Recommended)

**Pros:**
- Atomic updates guaranteed
- No application code changes for cascade
- Consistent even with direct DB access

**Cons:**
- SQLite trigger syntax complexity
- Harder to debug
- Logic split between app and DB

### Option B: Application Layer Cascade

**Pros:**
- All logic in Go code
- Easier testing and debugging
- Full control over cascade behavior

**Cons:**
- Must remember to call recalc in all places
- Risk of inconsistent state if code paths missed
- More complex transaction management

### Option C: Hybrid (Selected)

- Application layer handles main logic and cascade triggering
- Database triggers for safety net / constraint validation
- Repository methods encapsulate all cascade logic

---

## Success Criteria

- [ ] Feature status automatically updates when any task status changes
- [ ] Epic status automatically updates when any feature status changes
- [ ] Manual override can be set with `--status=<value>`
- [ ] Override can be cleared with `--status=auto`
- [ ] CLI shows "(calculated)" or "(manual override)" in output
- [ ] JSON output includes `status_source` field
- [ ] All edge cases handled per specification
- [ ] No N+1 queries in status calculation
- [ ] All existing tests pass
- [ ] New unit tests for calculation logic
- [ ] New integration tests for cascade behavior

---

## Open Questions

1. **Should blocked features/epics cascade?**
   - Current design: blocked is part of "active" calculation
   - Alternative: blocked could stop cascade (needs discussion)

2. **Should archived status be auto-calculated or manual only?**
   - Current design: archived treated as completed for calculation
   - Alternative: archived always manual, never auto-calculated

3. **Database trigger vs application layer?**
   - Current design: hybrid approach
   - Trade-off: atomicity vs debuggability

4. **Should status history be tracked?**
   - Current: task_history exists, no feature/epic history
   - Option: Add feature_history, epic_history tables

5. **Performance threshold for large projects?**
   - Need benchmarks for epics with 100+ features, 1000+ tasks
   - May need background recalculation for very large hierarchies

---

## Related Documents

- `/docs/plan/E07-enhancements/E07-F10-add-complete-method-to-epic-and-feature-commands/prd.md` - Related bulk completion feature
- `/internal/repository/feature_repository.go` - Current progress calculation implementation
- `/internal/repository/epic_repository.go` - Current progress calculation implementation
- `/internal/models/feature.go` - Feature status enum definitions
- `/internal/models/epic.go` - Epic status enum definitions

---

## Appendix: Status State Machine

### Feature Status Transitions

```
         +---> [completed] <---+
         |         ^           |
         |         |           |
[draft] -+---> [active] -------+
         |         |           |
         |         v           |
         +---> [blocked] ------+
         |                     |
         +---> [archived] <----+

Legend:
- Solid arrows: Calculated transitions
- All states can be reached via manual override
```

### Epic Status Transitions

```
[draft] <---> [active] <---> [completed]
   |             |              |
   +-------------+-------> [archived]

Legend:
- Bidirectional: Status can move back when child states change
- Archived: Typically manual override only
```
