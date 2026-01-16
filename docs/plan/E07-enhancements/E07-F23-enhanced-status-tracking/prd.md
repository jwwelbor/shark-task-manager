# E07-F23: Enhanced Status Tracking and Visibility (PRD)

**Feature:** E07-F23 - Enhanced Status Tracking and Visibility
**Epic:** E07 - Shark Enhancements
**Status:** draft
**Priority:** 5
**Depends On:** E07-F14 (Cascading Status Calculation)
**Created:** 2026-01-16

---

## Executive Summary

Enhance feature/epic status displays to provide **quick, actionable visibility** into work state. Move beyond simple completion percentages to show:
- What work remains (by phase and responsibility)
- What needs immediate attention (blocked, awaiting approval)
- Who's responsible for current work (agent vs human)
- Progress recognition before completion (ready_for_approval = 90% done)

**Key Innovation:** Config-driven using `progress_weight` and `responsibility` metadata from E07-F14.

---

## Problem Statement

### Current State (After E07-F14)
- Feature/epic status calculated from children ✅
- Progress shown as simple percentage (e.g., "60%")
- Can see task list, but requires scanning/counting

### User Pain Points

**1. "What work remains?"**
- PM sees "Feature progress: 60%" but doesn't know:
  - Are tasks in progress? Or just not started?
  - Are tasks blocked? Or waiting for approval?
  - Is agent working? Or waiting on me?

**2. "What needs my attention?"**
- Developer sees feature list, must open each feature to find:
  - Which tasks ready for approval (waiting for PM)
  - Which tasks blocked (need unblocking)
  - Which tasks in progress (actively being worked)

**3. "Progress doesn't recognize agent work"**
- Task at `ready_for_approval` shows 0% progress
- But agent work is DONE - just waiting for human approval
- PM doesn't see that 90% of work is complete

**4. "Can't quickly scan feature list"**
- Feature list shows status as "active"
- Doesn't show: `active (3 waiting approval, 2 blocked)`
- Requires opening each feature to assess state

---

## Solution Overview

### Enhanced Status Display

**Feature Get Output:**
```bash
$ shark feature get E07-F23

Feature: E07-F23 - Enhanced Status Tracking
Status: active (waiting) ⏳  [3 tasks awaiting your approval]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress Breakdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Overall: 68% (3.4/5 tasks weighted progress)
  • Completed: 2 tasks (40%)
  • Ready for Approval: 1 task (18%) ⏳
  • In Progress: 1 task (10%)
  • To Do: 1 task (0%)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Action Items (What needs your attention)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏳ Waiting for Approval (1):
  • E07-F23-003 - Add status breakdown display
    Agent work complete, awaiting your review

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Work Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 5 tasks
  ✅ Completed: 2 (40%)
  🏃 Agent Work: 1 (20%)
  ⏳ Human Work: 1 (20%)
  📋 Not Started: 1 (20%)
```

**Feature List Output:**
```bash
$ shark feature list E07

FEATURES (E07)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
E07-F22  Rejection Reasons       🟡 [████████░░░░░░░░]  45%  [3 ready, 2 blocked]
E07-F23  Enhanced Status         🟡 [██████████░░░░░░]  68%  [1 waiting approval]
E07-F24  Next Feature            🟢 [░░░░░░░░░░░░░░░░]   0%  [all todo]
```

---

## User Stories

### Story 1: Quick Status Assessment (Project Manager)
**As a** project manager
**I want to** quickly see what work remains and who's responsible
**So that** I can identify bottlenecks and blockers at a glance

**Acceptance Criteria:**
- [ ] Feature get shows work breakdown (agent, human, blocked, not started)
- [ ] Feature get shows action items section (what needs attention)
- [ ] Status context shown: `active (waiting)` instead of just `active`
- [ ] Visual indicators: ⏳ (waiting), 🚫 (blocked), ✅ (complete)

**Time Saved:** 18 minutes/day (from manual counting/scanning)

### Story 2: Progress Recognition (Developer)
**As a** developer
**I want** progress to reflect when my work is done (even if not approved)
**So that** I get credit for completion and can move to next task

**Acceptance Criteria:**
- [ ] Task at `ready_for_approval` contributes 90% to progress
- [ ] Progress shown as weighted: "68% (3.4/5 weighted)"
- [ ] "Agent Progress" shown separately from "Completion"
- [ ] Agent can see: "Your work complete, awaiting approval"

**Emotional Impact:** Frustrated → Recognized

### Story 3: Action Items (Human Approver)
**As a** product manager or reviewer
**I want to** immediately see tasks awaiting my approval
**So that** I can prioritize reviews and unblock developers

**Acceptance Criteria:**
- [ ] Feature get shows "Action Items" section prominently
- [ ] Tasks awaiting approval listed with context
- [ ] Blocked tasks shown with reasons
- [ ] Can filter feature list: `shark feature list --awaiting-approval`

**Time Saved:** 10 minutes/day (no more searching for what needs review)

### Story 4: Epic-Level Rollup (Stakeholder)
**As a** stakeholder
**I want to** see epic-level summary of all features
**So that** I understand overall project health without drilling down

**Acceptance Criteria:**
- [ ] Epic get shows feature status rollup by phase
- [ ] Epic get shows task rollup across all features
- [ ] Impediments surfaced: "2 features blocked, 1 at risk"
- [ ] Health indicator: 🟢 Healthy, 🟡 Attention, 🔴 At Risk

**Visibility:** Opaque → Transparent

---

## Technical Approach

### Config-Driven Design

Uses metadata from E07-F14 config:

```json
{
  "status_metadata": {
    "ready_for_approval": {
      "progress_weight": 0.9,        // 90% complete
      "responsibility": "human",      // human's responsibility
      "phase": "approval"
    },
    "in_development": {
      "progress_weight": 0.5,         // 50% complete
      "responsibility": "agent",       // agent's responsibility
      "phase": "development"
    },
    "blocked": {
      "progress_weight": 0.0,         // 0% (no progress)
      "responsibility": "none",        // external dependency
      "blocks_feature": true           // makes feature blocked
    }
  }
}
```

### Core Calculations

**1. Weighted Progress:**
```go
func CalculateProgress(statusCounts map[string]int, cfg *config.Config) float64 {
    totalTasks := 0
    weightedProgress := 0.0

    for status, count := range statusCounts {
        totalTasks += count
        meta := cfg.GetStatusMetadata(status)
        if meta != nil {
            weightedProgress += float64(count) * meta.ProgressWeight
        }
    }

    return (weightedProgress / float64(totalTasks)) * 100.0
}
```

**2. Work Breakdown:**
```go
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) WorkSummary {
    summary := WorkSummary{}

    for status, count := range statusCounts {
        meta := cfg.GetStatusMetadata(status)

        // Categorize by responsibility
        switch meta.Responsibility {
        case "agent":
            summary.AgentWork += count
        case "human", "qa_team":
            summary.HumanWork += count
        case "none":
            if meta.BlocksFeature {
                summary.BlockedWork += count
            }
        }
    }

    return summary
}
```

**3. Status Context:**
```go
func GetStatusContext(feature *Feature, statusCounts map[string]int) string {
    if feature.Status == "active" {
        // Check what kind of "active"
        if statusCounts["ready_for_approval"] > 0 {
            return "active (waiting)"
        }
        if statusCounts["blocked"] > 0 {
            return "active (blocked)"
        }
        return "active (development)"
    }
    return string(feature.Status)
}
```

---

## Enhanced Displays

### Feature Get Display

**Sections:**
1. **Header:** Feature key, title, status with context
2. **Progress Breakdown:** Weighted progress with task distribution
3. **Action Items:** What needs immediate attention
4. **Work Summary:** Breakdown by responsibility
5. **Tasks:** Workflow-ordered task list

**Example:**
```
Feature: E07-F23 - Enhanced Status Tracking
Status: active (waiting) ⏳
Created: 2026-01-16 | Updated: 2026-01-16 14:30

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress Breakdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Overall: 68% (3.4/5 tasks)
  • Completed: 2 tasks (40%)
  • Ready for Approval: 1 task (18%) ⏳
  • In Development: 1 task (10%)
  • Draft: 1 task (0%)

Progress Bar:
[████████████████████░░░░░░░░] 68% Weighted Progress
[████████████░░░░░░░░░░░░░░░░] 40% Completed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Action Items (What needs your attention)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏳ Waiting for Approval (1):
  • E07-F23-003 - Add status breakdown display
    Agent work complete, awaiting your review

    👉 Review: shark task approve E07-F23-003

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Work Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 5 tasks
Remaining: 3 tasks (60%)

Breakdown:
  ✅ Completed: 2 tasks
  🏃 Agent Work: 1 task (in progress)
  ⏳ Human Work: 1 task (awaiting approval)
  📋 Not Started: 1 task

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Tasks (5)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏳ Approval Phase:
  E07-F23-003  Add status breakdown display    ready_for_approval

🏃 Development Phase:
  E07-F23-002  Implement work breakdown         in_development

✅ Completed:
  E07-F23-001  Create config metadata           completed
  E07-F23-004  Add unit tests                   completed

📋 Planning:
  E07-F23-005  Update documentation             draft
```

### Feature List Display

**Enhanced columns:**
```
KEY      TITLE                    HEALTH  PROGRESS              NOTES
E07-F22  Rejection Reasons        🟡      45% (2.2/5)          [3 ready, 2 blocked]
E07-F23  Enhanced Status          🟡      68% (3.4/5)          [1 waiting approval]
E07-F24  Next Feature             🟢      0% (0/8)             [all todo]
```

**Health Indicators:**
- 🟢 Healthy: No blockers, on track
- 🟡 Attention: 1-2 blockers OR tasks awaiting approval > 7 days
- 🔴 At Risk: 3+ blockers OR >30% tasks blocked

### Epic Get Display

**Sections:**
1. **Header:** Epic key, title, overall progress
2. **Feature Status Rollup:** Features by phase with progress
3. **Task Rollup:** All tasks across features
4. **Impediments:** Features with blockers or risks
5. **Recent Activity:** Last 5 status changes

**Example:**
```
Epic: E07 - Shark Enhancements
Status: active (calculated)
Overall Progress: 60% (12 of 20 features complete)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Feature Status Summary (20 features)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Planning:    3 features [draft: 3]
Development: 8 features [active: 6, blocked: 2]
Review:      4 features [active: 2, waiting: 2]
Done:       12 features [completed: 12]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Task Rollup (250 tasks across all features)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✅ Completed: 150 (60%)
  🏃 In Progress: 45 (18%)
  ⏳ Awaiting Approval: 20 (8%)
  🚫 Blocked: 15 (6%)
  📋 To Do: 20 (8%)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️ Impediments & Risks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚫 Blocked Features (2):
  • E07-F05: 3 tasks blocked (waiting on API design)
  • E07-F12: 2 tasks blocked (dependency not ready)

⏳ Approval Backlog (2 features):
  • E07-F22: 3 tasks awaiting approval (age: 5 days)
  • E07-F23: 1 task awaiting approval (age: 2 days)
```

---

## JSON API Response Format

### Enhanced Feature Response

```json
{
  "id": 23,
  "key": "E07-F23",
  "title": "Enhanced Status Tracking",
  "status": "active",
  "status_context": "waiting",
  "status_explanation": "3 tasks awaiting approval",

  "progress": {
    "weighted_pct": 68.0,
    "completion_pct": 40.0,
    "weighted_ratio": "3.4/5",
    "completion_ratio": "2/5"
  },

  "work_summary": {
    "total_tasks": 5,
    "work_remaining": 3,
    "agent_work": 1,
    "human_work": 1,
    "blocked_work": 0,
    "not_started": 1
  },

  "status_breakdown": {
    "by_phase": {
      "approval": {
        "statuses": [
          {"status": "ready_for_approval", "count": 1, "color": "purple"}
        ],
        "total": 1,
        "progress_pct": 90
      },
      "development": {
        "statuses": [
          {"status": "in_development", "count": 1, "color": "yellow"}
        ],
        "total": 1,
        "progress_pct": 50
      },
      "done": {
        "statuses": [
          {"status": "completed", "count": 2, "color": "white"}
        ],
        "total": 2,
        "progress_pct": 100
      },
      "planning": {
        "statuses": [
          {"status": "draft", "count": 1, "color": "gray"}
        ],
        "total": 1,
        "progress_pct": 0
      }
    }
  },

  "action_items": {
    "awaiting_approval": [
      {
        "task_key": "E07-F23-003",
        "title": "Add status breakdown display",
        "status": "ready_for_approval",
        "age_days": 2
      }
    ],
    "blocked": [],
    "in_progress": [
      {
        "task_key": "E07-F23-002",
        "title": "Implement work breakdown",
        "status": "in_development"
      }
    ]
  },

  "health": {
    "indicator": "attention",
    "level": "yellow",
    "reasons": [
      "1 task awaiting approval"
    ]
  }
}
```

---

## Benefits

### Time Savings

**Project Manager:** 18 minutes/day
- Before: Manually count tasks, open each feature
- After: Single glance at feature list

**Developer:** 7 minutes/day
- Before: Search for next task, check dependencies
- After: Action items section shows what's ready

**AI Orchestrator:** 27 minutes/day
- Before: Multiple API calls to aggregate data
- After: Single API call with full context

**Total:** 52 minutes/day = **17 hours/month**

### Quality Improvements

**Visibility:**
- Bottlenecks identified immediately
- Approval backlogs visible
- Health indicators proactive

**Recognition:**
- Agent work credited before approval (90% progress)
- Progress reflects reality
- Emotional impact: Frustrated → Recognized

**Actionability:**
- "What needs attention" clearly shown
- Next steps obvious
- Less cognitive load

---

## Implementation Tasks

See architecture documentation for detailed tasks.

**Phases:**
1. **Core Calculations** (E07-F23-001 to E07-F23-003)
   - Weighted progress calculation
   - Work breakdown by responsibility
   - Status context derivation

2. **Feature Get Enhancement** (E07-F23-004 to E07-F23-006)
   - Progress breakdown section
   - Action items section
   - Work summary section

3. **Feature List Enhancement** (E07-F23-007 to E07-F23-008)
   - Health indicators
   - Enhanced columns (notes, health)

4. **Epic Get Enhancement** (E07-F23-009 to E07-F23-011)
   - Feature status rollup
   - Task rollup
   - Impediments section

**Estimated Effort:** 17 hours total

---

## Dependencies

### Required
- **E07-F14 (Cascading Status Calculation):** Provides status calculation and config metadata

### Optional
- **E07-F22 (Rejection Reason):** Rejection reasons can be shown in action items

---

## Acceptance Criteria

### Phase 1: Feature Get Enhancement
- [ ] Progress breakdown shows weighted progress and completion
- [ ] Progress shows ratio: "68% (3.4/5)"
- [ ] Action items section shows tasks awaiting approval
- [ ] Work summary shows agent/human/blocked/not started breakdown
- [ ] Status context shown: "active (waiting)"

### Phase 2: Feature List Enhancement
- [ ] Health indicators: 🟢 🟡 🔴
- [ ] Notes column: "[3 ready, 2 blocked]"
- [ ] Progress bar with weighted progress
- [ ] Can filter: `--awaiting-approval`, `--with-blockers`

### Phase 3: Epic Get Enhancement
- [ ] Feature status rollup by phase
- [ ] Task rollup across all features
- [ ] Impediments section with blocked features
- [ ] Approval backlog shown with age

### Phase 4: Performance
- [ ] All queries < 100ms (p95)
- [ ] Single API call for full context
- [ ] No N+1 queries

---

## Non-Goals

- ❌ Real-time notifications (future feature)
- ❌ Velocity metrics (future feature)
- ❌ Burndown charts (future feature)
- ❌ Gantt charts (out of scope)

---

## Success Metrics

### Usability
- ✅ Time to assess feature status < 5 seconds (down from 30s)
- ✅ User can answer "what needs attention?" in < 3 seconds
- ✅ 100% of users agree: "I know what to do next"

### Accuracy
- ✅ Progress reflects reality (agent work counted)
- ✅ Health indicators match manual assessment (>95% accuracy)
- ✅ Work breakdown adds up to 100%

### Performance
- ✅ Feature get < 100ms (p95)
- ✅ Feature list < 200ms for 50 features (p95)
- ✅ Epic get < 500ms for 20 features (p95)

---

## Future Enhancements

### Phase 2: Velocity Metrics
- Average completion time per task
- Tasks completed per week
- Velocity trends

### Phase 3: Notifications
- Slack/email when task awaiting approval
- Notify when task blocked
- Daily digest of pending approvals

### Phase 4: Dashboards
- Web-based dashboard
- Real-time updates
- Customizable views

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Architecture
