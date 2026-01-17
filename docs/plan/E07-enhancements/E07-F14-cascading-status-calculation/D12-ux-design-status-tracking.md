# UX Design: Enhanced Feature/Epic Status Tracking

**Feature**: E07-F14 - Cascading Status Calculation
**Document Type**: Journey Map & UX Design
**Author**: CXDesigner Agent
**Date**: 2026-01-16
**Status**: Draft

---

## Executive Summary

This document defines the user experience for enhanced status tracking in `shark feature list`, `shark feature get`, `shark epic list`, and `shark epic get` commands. The design prioritizes **at-a-glance status assessment** with minimal cognitive load, enabling users to quickly identify work state, blockers, and items needing attention.

### Key Design Principles

1. **Scannable First**: Users should understand status in < 3 seconds
2. **Progressive Disclosure**: Summary view → detailed view → JSON for automation
3. **Color as Signal**: Phase-based colors guide attention to actionable items
4. **Status Context**: Distinguish calculated vs manual, show what's driving status
5. **Actionable Insights**: Surface blockers, waiting items, and next actions

---

## User Journey: Feature Status Assessment

### Journey Stage 1: Quick Scan (List View)

**User Goal**: Understand overall project health across multiple features

**Touchpoints**: `shark feature list`, `shark list E07`

**User Actions**:
- Runs list command
- Scans table for visual signals
- Identifies features needing attention

**User Thoughts**:
- "What's in progress?"
- "What's blocked?"
- "What's waiting for my approval?"
- "How much work is left?"

**User Emotions**:
- Confident when status is clear
- Frustrated when table is cluttered
- Anxious when can't quickly spot blockers

**Pain Points (Current)**:
- Status column shows only text ("active", "draft", "completed")
- No indication of **why** status is active (in_development vs ready_for_approval)
- No visual cue for blockers or items needing human attention
- Progress % doesn't distinguish "50% done" from "100% done but waiting approval"

**Opportunities**:
- Color-code status by workflow phase
- Add visual indicator for blockers and waiting items
- Show task count breakdown (completed/total)
- Distinguish "work remaining" from "approval remaining"

---

### Journey Stage 2: Detailed Investigation (Get View)

**User Goal**: Understand feature status in detail, identify next actions

**Touchpoints**: `shark feature get E07-F01`, `shark get E07 F01`

**User Actions**:
- Views feature details
- Reads task status breakdown
- Reviews task list
- Identifies blockers or waiting items

**User Thoughts**:
- "Why is this feature active?"
- "What tasks are blocking completion?"
- "What needs my approval?"
- "What's the breakdown by phase?"

**User Emotions**:
- Relieved when status explanation is clear
- Confident when next actions are obvious
- Confused when status doesn't match expectation

**Pain Points (Current)**:
- Status breakdown shows counts but not phase grouping
- Task list is alphabetical/numeric, not workflow-ordered
- No clear indication of "human action required" items
- Progress doesn't explain "what's left to do"

**Opportunities**:
- Group tasks by workflow phase
- Highlight "ready_for_approval" items prominently
- Show status explanation ("active because 3 tasks in_development")
- Add "next actions" section

---

### Journey Stage 3: Epic-Level Assessment

**User Goal**: Understand epic health across multiple features

**Touchpoints**: `shark epic get E07`, `shark get E07`

**User Actions**:
- Views epic summary
- Scans feature breakdown
- Identifies which features need attention

**User Thoughts**:
- "Which features are progressing?"
- "Where are the bottlenecks?"
- "What's waiting for approval?"
- "Is the epic on track?"

**User Emotions**:
- Confident when feature statuses are clear
- Overwhelmed when too much detail
- Frustrated when can't identify blockers quickly

**Pain Points (Current)**:
- Epic status is single value ("active") with no explanation
- Feature list shows progress but not status breakdown
- No aggregated view of blockers or waiting items
- Can't quickly see which features need attention

**Opportunities**:
- Show epic status explanation
- Add feature status distribution
- Highlight features with blockers or waiting items
- Show aggregated task phase breakdown

---

## Design Solution: List Views

### `shark feature list` - Enhanced Design

**Before (Current)**:
```
Key       Title                Epic ID  Status  Progress  Tasks  Order
E07-F01   Authentication      7        active  60.0%     5      1
E07-F02   Authorization       7        active  25.0%     8      2
E07-F03   User Profiles       7        draft   0.0%      10     3
```

**After (Enhanced)**:
```
Key       Title                Status                Progress      Tasks      Notes
E07-F01   Authentication      active (dev)          60% (3/5)     3 done
E07-F02   Authorization       active (waiting) ⏳    75% (6/8)     2 waiting
E07-F03   User Profiles       draft                 0% (0/10)     0 done
E07-F04   Session Mgmt        active (blocked) ⚠️    40% (2/5)     1 blocked
E07-F05   Password Reset      completed ✓           100% (4/4)    4 done
```

**Key Changes**:

1. **Status Column**: Shows phase context in parentheses
   - `active (dev)` = in_development phase
   - `active (waiting) ⏳` = ready_for_approval/review (human action needed)
   - `active (blocked) ⚠️` = blocked tasks present
   - `completed ✓` = all tasks done

2. **Progress Column**: Shows ratio alongside percentage
   - `60% (3/5)` = 3 of 5 tasks completed
   - Distinguishes "50% done" from "50% of 5 tasks" vs "50% of 50 tasks"

3. **Tasks Column**: Shows completed count
   - `3 done` instead of just `5` (total)
   - Aligns with progress ratio

4. **Notes Column**: Highlights actionable items
   - `2 waiting` = 2 tasks ready_for_approval
   - `1 blocked` = 1 task blocked
   - Empty when feature is progressing normally

5. **Visual Indicators**:
   - ⏳ = waiting for human action
   - ⚠️ = blocked
   - ✓ = completed

**Color Coding (Terminal)**:
- **Planning phase** (draft, ready_for_refinement, in_refinement): Gray/Cyan
- **Development phase** (ready_for_development, in_development): Yellow
- **Review phase** (ready_for_code_review, in_code_review): Magenta
- **QA phase** (ready_for_qa, in_qa): Green
- **Approval phase** (ready_for_approval, in_approval): Purple
- **Done** (completed, cancelled): White/Gray
- **Any phase** (blocked, on_hold): Red/Orange

---

### `shark epic list` - Enhanced Design

**Before (Current)**:
```
Key   Title                  Status    Progress  Priority
E07   Enhancements          active    42.5%     1
E10   Documentation         draft     0.0%      5
```

**After (Enhanced)**:
```
Key   Title                  Status          Progress      Features      Notes
E07   Enhancements          active (mixed)   42% (10/24)   3 waiting
E10   Documentation         draft           0% (0/5)      5 draft
E11   Performance           active (dev)    65% (13/20)   2 in progress
E12   Security              completed ✓     100% (8/8)    8 done
```

**Key Changes**:

1. **Status Column**: Shows status with feature distribution hint
   - `active (mixed)` = features in multiple phases
   - `active (dev)` = most features in development
   - `draft` = no features started

2. **Progress Column**: Shows task completion ratio
   - `42% (10/24)` = 10 of 24 total tasks completed across all features
   - Aggregated from all features in epic

3. **Features Column**: Shows feature status summary
   - `3 waiting` = 3 features have waiting tasks
   - `5 draft` = 5 features not started
   - `2 in progress` = 2 features actively being worked

4. **Notes Column**: Highlights issues
   - Same as feature list (waiting, blocked items)

---

## Design Solution: Get Views

### `shark feature get E07-F01` - Enhanced Design

**Before (Current)**:
```
Feature: E07-F01
Title: Authentication
Status: active
Progress: 60.0%
Path: docs/plan/E07-enhancements/E07-F01-authentication/

Task Status Breakdown:
Status              Count
completed           3
in_development      1
ready_for_approval  1

Tasks:
Key              Title                      Status
T-E07-F01-001   Implement JWT validation   completed
T-E07-F01-002   Add token refresh          completed
T-E07-F01-003   Build login endpoint       completed
T-E07-F01-004   Add password hashing       in_development
T-E07-F01-005   Write API tests            ready_for_approval
```

**After (Enhanced)**:
```
════════════════════════════════════════════════════════
Feature: E07-F01 - Authentication
════════════════════════════════════════════════════════

Status:   active (waiting for approval) ⏳
          Calculated from task statuses

Progress: 60% complete (3 of 5 tasks)
          1 task ready for approval
          1 task in development

Path:     docs/plan/E07-enhancements/E07-F01-authentication/

────────────────────────────────────────────────────────
Status Distribution (by workflow phase)
────────────────────────────────────────────────────────

Phase            Status                   Count  Notes
═══════════════════════════════════════════════════════
✓ Done           completed                3
⏳ Approval       ready_for_approval       1      ← ACTION NEEDED
⚙ Development    in_development           1

────────────────────────────────────────────────────────
Action Items
────────────────────────────────────────────────────────

• 1 task ready for approval (ready_for_approval)
  ⏳ T-E07-F01-005: Write API tests

• 1 task in development (in_development)
  ⚙ T-E07-F01-004: Add password hashing

────────────────────────────────────────────────────────
Task List (grouped by workflow phase)
────────────────────────────────────────────────────────

⏳ APPROVAL PHASE
  T-E07-F01-005  Write API tests            ready_for_approval

⚙ DEVELOPMENT PHASE
  T-E07-F01-004  Add password hashing       in_development

✓ COMPLETED
  T-E07-F01-001  Implement JWT validation   completed
  T-E07-F01-002  Add token refresh          completed
  T-E07-F01-003  Build login endpoint       completed
```

**Key Changes**:

1. **Status Section**: Explains why feature has current status
   - Shows calculated vs manual
   - Explains driving factors ("waiting for approval")
   - Adds visual indicator

2. **Progress Section**: Rich progress explanation
   - Completion ratio
   - Breakdown by phase
   - Clear "what's left" summary

3. **Status Distribution**: Workflow-ordered breakdown
   - Grouped by phase
   - Color-coded by workflow
   - "Action Needed" markers for human attention

4. **Action Items Section**: NEW - highlights next steps
   - Separate section for items needing attention
   - Groups by action type
   - Shows specific tasks

5. **Task List**: Workflow-ordered instead of alphabetical
   - Grouped by phase (approval → development → completed)
   - Visual phase indicators
   - Most urgent items at top

---

### `shark epic get E07` - Enhanced Design

**Before (Current)**:
```
Epic: E07
Title: Enhancements
Status: active
Progress: 42.5%
Path: docs/plan/E07-enhancements/

Features:
Key        Title                Status    Progress  Tasks
E07-F01   Authentication      active    60.0%     5
E07-F02   Authorization       active    25.0%     8
E07-F03   User Profiles       draft     0.0%      10
```

**After (Enhanced)**:
```
════════════════════════════════════════════════════════
Epic: E07 - Enhancements
════════════════════════════════════════════════════════

Status:   active (mixed phases)
          3 features in progress
          Calculated from feature statuses

Progress: 43% complete (10 of 24 total tasks)
          3 features in development
          2 features waiting for approval
          1 feature blocked

Path:     docs/plan/E07-enhancements/

────────────────────────────────────────────────────────
Feature Status Distribution
────────────────────────────────────────────────────────

Phase            Features  Tasks (done/total)  Notes
═══════════════════════════════════════════════════════
⏳ Approval       2         6/8                 ← ACTION NEEDED
⚙ Development    3         4/10
📝 Planning       1         0/6
✓ Completed      1         4/4

────────────────────────────────────────────────────────
Action Items
────────────────────────────────────────────────────────

⏳ Features waiting for approval (2):
  • E07-F01: Authentication (1 task ready_for_approval)
  • E07-F04: Session Management (2 tasks ready_for_approval)

⚠️ Features with blockers (1):
  • E07-F02: Authorization (1 task blocked: "Waiting on API design")

────────────────────────────────────────────────────────
Feature List (ordered by urgency)
────────────────────────────────────────────────────────

⏳ WAITING FOR APPROVAL
  E07-F01  Authentication          active (waiting)   60% (3/5)   1 waiting
  E07-F04  Session Management      active (waiting)   80% (4/5)   2 waiting

⚠️ BLOCKED
  E07-F02  Authorization           active (blocked)   25% (2/8)   1 blocked

⚙ IN DEVELOPMENT
  E07-F03  User Profiles          active (dev)       30% (3/10)  3 in dev
  E07-F05  Password Reset         active (dev)       50% (2/4)   2 in dev
  E07-F06  Email Verification     active (dev)       40% (2/5)   1 in dev

📝 PLANNING
  E07-F07  2FA Support            draft             0% (0/6)    0 done

✓ COMPLETED
  E07-F08  Session Timeout        completed ✓        100% (4/4)  4 done
```

**Key Changes**:

1. **Status Section**: Epic-level status explanation
   - Shows feature distribution
   - Explains calculated status
   - Highlights issues

2. **Progress Section**: Task-level aggregation
   - Total tasks across all features
   - Feature-level breakdown
   - Identifies bottlenecks

3. **Feature Status Distribution**: NEW - aggregated view
   - Groups features by phase
   - Shows task completion per phase
   - Highlights action items

4. **Action Items Section**: NEW - epic-level actions
   - Features waiting for approval
   - Features with blockers
   - Specific task references

5. **Feature List**: Urgency-ordered
   - Waiting → Blocked → In Progress → Planning → Completed
   - Shows detailed status context
   - Easy to identify next steps

---

## JSON Output Schema

### Enhanced Feature JSON Output

```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "status": "active",
  "status_source": "calculated",
  "status_explanation": {
    "primary_phase": "approval",
    "reason": "waiting_for_approval",
    "details": "1 task ready for approval, 1 task in development"
  },
  "progress_pct": 60.0,
  "progress_breakdown": {
    "completed": 3,
    "total": 5,
    "by_phase": {
      "done": 3,
      "approval": 1,
      "development": 1,
      "planning": 0
    }
  },
  "status_distribution": [
    {
      "phase": "approval",
      "statuses": [
        {"status": "ready_for_approval", "count": 1}
      ],
      "action_required": true
    },
    {
      "phase": "development",
      "statuses": [
        {"status": "in_development", "count": 1}
      ],
      "action_required": false
    },
    {
      "phase": "done",
      "statuses": [
        {"status": "completed", "count": 3}
      ],
      "action_required": false
    }
  ],
  "action_items": {
    "waiting_for_approval": [
      {"key": "T-E07-F01-005", "title": "Write API tests"}
    ],
    "blocked": [],
    "in_development": [
      {"key": "T-E07-F01-004", "title": "Add password hashing"}
    ]
  },
  "health": {
    "overall": "good",
    "has_blockers": false,
    "awaiting_approval": true,
    "in_progress": true
  }
}
```

### Enhanced Epic JSON Output

```json
{
  "id": 7,
  "key": "E07",
  "title": "Enhancements",
  "status": "active",
  "status_source": "calculated",
  "status_explanation": {
    "primary_phase": "mixed",
    "reason": "multiple_phases",
    "details": "3 features in development, 2 waiting for approval, 1 blocked"
  },
  "progress_pct": 42.5,
  "progress_breakdown": {
    "completed_tasks": 10,
    "total_tasks": 24,
    "by_phase": {
      "done": 10,
      "approval": 3,
      "development": 7,
      "planning": 4
    }
  },
  "feature_distribution": {
    "total": 8,
    "by_status": {
      "active_waiting": 2,
      "active_development": 3,
      "active_blocked": 1,
      "draft": 1,
      "completed": 1
    },
    "by_phase": {
      "approval": {"count": 2, "tasks_done": 6, "tasks_total": 8},
      "development": {"count": 3, "tasks_done": 4, "tasks_total": 10},
      "planning": {"count": 1, "tasks_done": 0, "tasks_total": 6},
      "done": {"count": 1, "tasks_done": 4, "tasks_total": 4}
    }
  },
  "action_items": {
    "features_waiting_approval": [
      {"key": "E07-F01", "title": "Authentication", "waiting_count": 1},
      {"key": "E07-F04", "title": "Session Management", "waiting_count": 2}
    ],
    "features_blocked": [
      {"key": "E07-F02", "title": "Authorization", "blocked_count": 1, "reason": "Waiting on API design"}
    ]
  },
  "health": {
    "overall": "caution",
    "has_blockers": true,
    "awaiting_approval": true,
    "completion_estimate": "70% of work done, 30% remaining (mostly planning)"
  }
}
```

---

## Color Coding Strategy

### Terminal Output Colors

**Phase-Based Color Scheme**:

| Phase | Statuses | Color | Purpose |
|-------|----------|-------|---------|
| Planning | draft, ready_for_refinement, in_refinement | Gray/Cyan | Low urgency, preparation |
| Development | ready_for_development, in_development | Yellow | Active work, medium urgency |
| Review | ready_for_code_review, in_code_review | Magenta | Technical validation, higher urgency |
| QA | ready_for_qa, in_qa | Green | Testing phase, progress indicator |
| Approval | ready_for_approval, in_approval | Purple | Human action needed, HIGH urgency |
| Done | completed, cancelled | White/Gray | Finished, informational |
| Any | blocked, on_hold | Red/Orange | Critical attention needed |

**Visual Indicators**:
- ⏳ = Waiting for human action (approval, review)
- ⚠️ = Blocked or critical issue
- ⚙ = Active development work
- 📝 = Planning/refinement
- ✓ = Completed

**Attention Hierarchy**:
1. **Red** (blocked) = immediate attention
2. **Purple** (ready_for_approval) = human action needed
3. **Yellow** (in_development) = progressing normally
4. **Gray** (draft/planning) = low priority

---

## Implementation Priority

### Phase 1: Core Status Enhancement (T-E07-F14-009, T-E07-F14-010)
- Add status explanation to feature/epic get output
- Add progress breakdown with ratios
- Add status distribution grouped by phase

### Phase 2: Action Items (T-E07-F14-011)
- Add "Action Items" section to get output
- Highlight waiting_for_approval items
- Show blocked items with reasons

### Phase 3: Enhanced List Views (T-E07-F14-012)
- Update feature/epic list with status context
- Add visual indicators (⏳, ⚠️, ✓)
- Add Notes column

### Phase 4: JSON Schema Enhancement (T-E07-F14-013+)
- Add status_explanation to JSON
- Add progress_breakdown to JSON
- Add action_items to JSON
- Add health indicators

---

## Success Metrics

### Usability Goals

1. **Scan Time**: User can identify feature health in < 3 seconds
2. **Recognition Time**: User can identify action items in < 5 seconds
3. **Cognitive Load**: User doesn't need to mentally calculate "what's left"
4. **Error Reduction**: Zero confusion about calculated vs manual status

### User Satisfaction Goals

1. **Clarity**: "I understand why the status is X" → 100% agreement
2. **Actionability**: "I know what to do next" → 100% agreement
3. **Confidence**: "I trust this status information" → 100% agreement
4. **Efficiency**: "This saves me time" → 100% agreement

### Technical Goals

1. **Performance**: All queries < 100ms for epics with 50+ features
2. **Consistency**: Status calculations always reflect actual task states
3. **Backward Compatibility**: Existing JSON parsers continue working
4. **Extensibility**: Easy to add new phases or statuses

---

## Edge Cases & Exceptional Scenarios

### Edge Case 1: Mixed Phase Features
**Scenario**: Feature has tasks in multiple phases (e.g., 2 completed, 1 in_development, 1 ready_for_approval)

**UX Treatment**:
- Status shows `active (mixed)`
- Status explanation shows breakdown: "1 waiting approval, 1 in development"
- Action items section highlights waiting task
- Task list groups by phase

### Edge Case 2: 100% Progress but Waiting Approval
**Scenario**: Feature at 100% progress but has tasks in ready_for_approval

**UX Treatment**:
- Status shows `active (waiting) ⏳` NOT `completed`
- Progress shows `100% (4/4) - 1 waiting approval`
- Action items section prominently shows waiting task
- Status explanation: "All development complete, awaiting approval"

### Edge Case 3: Feature with All Blocked Tasks
**Scenario**: Feature has 3 tasks, all blocked

**UX Treatment**:
- Status shows `active (blocked) ⚠️`
- Progress shows current completion
- Action items section lists all blockers with reasons
- Epic-level view shows feature in "blocked" group

### Edge Case 4: Epic with No Tasks
**Scenario**: Epic has 3 features, but no tasks created yet

**UX Treatment**:
- Status shows `draft (no tasks)`
- Progress shows `0% (0/0)`
- Feature distribution shows all features as draft
- No action items section

### Edge Case 5: Manual Status Override
**Scenario**: Feature manually set to `archived` despite incomplete tasks

**UX Treatment**:
- Status shows `archived (manual override)`
- Progress shows actual task completion
- Status explanation: "Status manually overridden (calculated: active)"
- Warning indicator in list view

---

## Open Design Questions

### Question 1: Should we show estimated time to completion?
**Current**: No time estimates
**Option A**: Add "X tasks remaining → Est. Y days"
**Option B**: Add "Based on completion rate, X% likely by DATE"
**Recommendation**: Defer to future enhancement (out of scope for F14)

### Question 2: Should blocked tasks show block reason in list view?
**Current**: Just shows "1 blocked"
**Option A**: Show first 20 chars of reason: "1 blocked: Waiting on..."
**Option B**: Keep minimal, show in get view only
**Recommendation**: Option B (keep list view scannable)

### Question 3: Should we add "urgency score" to features?
**Current**: No urgency indicator
**Option A**: Add urgency column based on waiting + blocked count
**Option B**: Sort by urgency automatically
**Recommendation**: Option B (sort features by urgency in epic get view)

### Question 4: How to handle `on_hold` status?
**Current**: Part of "any" phase
**Treatment**: Same as blocked (⚠️ indicator, orange color, action items section)
**Recommendation**: Distinguish on_hold (intentional) from blocked (unintentional) with different indicator (⏸ vs ⚠️)

---

## Related Documents

- [E07-F14 Feature PRD](./prd.md) - Complete requirements
- [E07-F14 Feature Specification](./feature.md) - Feature summary
- [Workflow Configuration](./.sharkconfig.json) - Status metadata and colors
- [CLI Reference](../../../docs/CLI_REFERENCE.md) - Current CLI patterns

---

## Appendix: Visual Mockups

### Terminal Output Color Preview

```
┌─────────────────────────────────────────────────────────────────┐
│ Feature: E07-F01 - Authentication                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Status:   active (waiting for approval) ⏳ [PURPLE]             │
│           Calculated from task statuses                         │
│                                                                 │
│ Progress: 60% complete (3 of 5 tasks)                           │
│           1 task ready for approval [PURPLE]                    │
│           1 task in development [YELLOW]                        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Status Distribution (by workflow phase)                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ✓ Done           completed [WHITE]         3                   │
│ ⏳ Approval       ready_for_approval [PURPLE]  1  ← ACTION      │
│ ⚙ Development    in_development [YELLOW]      1                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Action Items                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ • 1 task ready for approval [PURPLE]                            │
│   ⏳ T-E07-F01-005: Write API tests                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Color Legend**:
- [PURPLE] = #9f7aea (ready_for_approval, in_approval)
- [YELLOW] = #f6ad55 (in_development, ready_for_development)
- [WHITE] = #ffffff (completed)
- [RED] = #fc8181 (blocked)
- [GREEN] = #68d391 (ready_for_qa, in_qa)
- [GRAY] = #a0aec0 (draft, cancelled)

---

## Appendix: Accessibility Considerations

### Color Blindness Support

**Problem**: Users with color vision deficiency may not distinguish phase colors

**Solution**:
1. **Never rely solely on color**: Always pair with text/symbols
2. **Use shape indicators**: ⏳, ⚠️, ⚙, ✓, 📝
3. **Use phase labels**: "APPROVAL PHASE", "DEVELOPMENT PHASE"
4. **Support `--no-color` flag**: All information visible in monochrome

### Screen Reader Support

**Problem**: Screen readers don't convey color or visual layout

**Solution**:
1. **Structured output**: Use clear section headers
2. **Verbose labels**: "Status: active, waiting for approval (1 task)"
3. **Action items first**: Most important info at top
4. **Consistent structure**: Same order across all commands

### Terminal Width Handling

**Problem**: Narrow terminals (< 80 columns) break table layouts

**Solution**:
1. **Responsive truncation**: Shorten titles, not data
2. **Priority columns**: Show key info first (status, progress)
3. **Fallback to vertical**: Stack columns if width < 60
4. **JSON alternative**: Always available with `--json`

---

## Conclusion

This UX design prioritizes **scannable status assessment** with **minimal cognitive load**. By using phase-based color coding, workflow-ordered grouping, and explicit action items, users can quickly understand feature/epic health and identify next steps.

The design supports three user personas:
1. **Project Manager**: Quick scan of epic health
2. **Developer**: Detailed feature status and next tasks
3. **Automation**: Rich JSON schema for programmatic access

Key innovations:
- Status explanation (why is it active?)
- Progress breakdown (what's done, what's left?)
- Action items section (what needs attention?)
- Phase-based grouping (workflow-ordered, not alphabetical)
- Health indicators (blockers, waiting items)

All changes maintain backward compatibility and follow existing Shark CLI patterns.
