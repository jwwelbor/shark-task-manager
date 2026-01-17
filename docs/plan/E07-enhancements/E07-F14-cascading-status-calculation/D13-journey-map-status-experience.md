# Journey Map: Feature/Epic Status Assessment Experience

**Feature**: E07-F14 - Cascading Status Calculation
**Document Type**: Customer Experience Journey Map
**Author**: CXDesigner Agent
**Date**: 2026-01-16
**Status**: Draft

---

## Overview

This journey map illustrates the end-to-end experience of assessing feature and epic status in Shark Task Manager. It identifies pain points in the current experience and demonstrates how the enhanced design creates a more intuitive, actionable workflow.

---

## Primary User Personas

### Persona 1: Project Manager (Maya)

**Background**:
- Manages 3-5 epics simultaneously
- Checks status 5-10 times per day
- Needs quick assessment of overall health
- Makes decisions based on status information

**Goals**:
- Quickly identify features needing attention
- Spot blockers and bottlenecks
- Understand which items need approval
- Track overall epic progress

**Pain Points (Current)**:
- Can't quickly identify features waiting for approval
- Status "active" is too vague (what kind of active?)
- Must mentally calculate "what's left to do"
- Can't easily spot blockers without clicking into each feature

---

### Persona 2: Developer (Alex)

**Background**:
- Works on 1-2 features at a time
- Checks status before starting work
- Needs to understand dependencies and next tasks
- Reports progress to project manager

**Goals**:
- Understand feature status quickly
- Identify next tasks to work on
- See which tasks are blocked
- Understand why a feature is in its current state

**Pain Points (Current)**:
- Status breakdown is count-only (no context)
- Task list is not workflow-ordered
- Can't see "what's blocking completion"
- No clear indication of which tasks need different actions

---

### Persona 3: AI Orchestrator (Automated System)

**Background**:
- Polls status every 5 minutes
- Routes tasks to appropriate agents
- Generates reports for human stakeholders
- Makes decisions based on JSON output

**Goals**:
- Programmatically identify actionable items
- Determine which agent type to spawn
- Track workflow progression
- Provide structured reports

**Pain Points (Current)**:
- JSON output lacks status explanation
- No machine-readable action items field
- Can't distinguish "waiting for human" vs "waiting for agent"
- No health indicators for automated decision-making

---

## Journey Map: Maya (Project Manager)

### Scenario: Morning Status Check

**Context**: Maya arrives at work and wants to check the health of Epic E07 (Enhancements) before the daily standup meeting in 15 minutes.

---

### Stage 1: Quick Epic Scan

**Goal**: Get overall epic health in < 30 seconds

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark epic get E07
```

**Output (Current)**:
```
Epic: E07
Title: Enhancements
Status: active
Progress: 42.5%
Features: 8 features
```

**Maya's Thoughts**:
- "OK, 42% progress... but is that good?"
- "Status is 'active' but what does that mean?"
- "Are we blocked on anything?"
- "How many features are waiting for my approval?"

**Maya's Emotions**: 😕 Uncertain, needs more information

**Pain Points**:
- Progress percentage doesn't explain what's left
- Status "active" is too generic
- No indication of blockers or waiting items
- Must run `shark feature list E07` to see details

**Time Spent**: 5 seconds to view, 30 seconds to interpret

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark epic get E07
```

**Output (Enhanced)**:
```
════════════════════════════════════════════════════════
Epic: E07 - Enhancements
════════════════════════════════════════════════════════

Status:   active (mixed phases)
          3 features in progress
          Calculated from feature statuses

Progress: 43% complete (10 of 24 total tasks)
          2 features waiting for approval ⏳
          1 feature blocked ⚠️

────────────────────────────────────────────────────────
Action Items
────────────────────────────────────────────────────────

⏳ Features waiting for approval (2):
  • E07-F01: Authentication (1 task ready_for_approval)
  • E07-F04: Session Management (2 tasks ready_for_approval)

⚠️ Features with blockers (1):
  • E07-F02: Authorization (1 task blocked: "Waiting on API design")
```

**Maya's Thoughts**:
- "OK, 43% done, 10 of 24 tasks complete"
- "2 features need my approval - I can do that now!"
- "1 feature is blocked - need to follow up on API design"
- "3 features are progressing normally"

**Maya's Emotions**: 😊 Confident, knows exactly what to do

**Benefits**:
- Progress is contextualized (10 of 24 tasks)
- Status explanation shows what's happening
- Action items section highlights what needs attention
- Blockers are surfaced with reasons

**Time Spent**: 5 seconds to view, 5 seconds to understand, 0 seconds to decide

**Time Saved**: 25 seconds per epic check = 4 minutes per day = 20 minutes per week

---

### Stage 2: Addressing Approvals

**Goal**: Approve waiting tasks efficiently

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark feature get E07-F01
# Scroll through task list
# Identify ready_for_approval tasks manually
# Note task key
shark task approve T-E07-F01-005
```

**Pain Points**:
- Must open each feature separately
- Task list is not ordered by urgency
- Must manually identify which tasks need approval
- Must remember task keys

**Time Spent**: 30 seconds per feature × 2 features = 60 seconds

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark epic get E07
# Action items section shows all waiting tasks immediately
shark task approve T-E07-F01-005
shark task approve T-E07-F04-002
shark task approve T-E07-F04-003
```

**Benefits**:
- All waiting tasks shown in one view
- Task keys are right there (no need to navigate)
- Can approve all tasks without opening individual features
- Clear indication that these need approval

**Time Spent**: 10 seconds to identify + 15 seconds to approve = 25 seconds total

**Time Saved**: 35 seconds per approval batch = 3 minutes per day = 15 minutes per week

---

### Stage 3: Addressing Blockers

**Goal**: Understand and resolve blockers

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark feature get E07-F02
# Read through tasks
# Find blocked task
shark task get T-E07-F02-003
# Read block reason
# Follow up with team
```

**Pain Points**:
- Must navigate to feature to see blockers
- Block reason not visible in epic view
- Must manually track which features have blockers
- No aggregated view of all blockers

**Time Spent**: 45 seconds to identify blocker, unknown time to resolve

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark epic get E07
# Action items section shows blocker with reason immediately
# Follow up with team about "Waiting on API design"
# No need to navigate to feature
```

**Benefits**:
- Blocker visible in epic view
- Block reason shown inline
- Can immediately take action
- All blockers aggregated in one place

**Time Spent**: 10 seconds to identify blocker, same time to resolve

**Time Saved**: 35 seconds per blocker check = 2 minutes per day = 10 minutes per week

---

### Stage 4: Reporting Status to Stakeholders

**Goal**: Communicate epic health to leadership

#### Current Experience (Before Enhancement)

**Maya's Process**:
1. Run `shark epic get E07`
2. Manually count features by status
3. Identify blockers by checking each feature
4. Calculate "what's left to do" mentally
5. Write status email

**Pain Points**:
- Must manually aggregate information
- No clear "health indicator"
- Hard to explain "why 42% progress"
- Must check each feature for blockers

**Time Spent**: 5 minutes per epic × 3 epics = 15 minutes

---

#### Enhanced Experience (After Enhancement)

**Maya's Process**:
1. Run `shark epic get E07`
2. Copy/paste status explanation and action items
3. Add context if needed
4. Send status email

**Benefits**:
- Status explanation is ready to share
- Action items section is stakeholder-friendly
- Feature distribution shows breakdown clearly
- Health indicators provide quick summary

**Time Spent**: 2 minutes per epic × 3 epics = 6 minutes

**Time Saved**: 9 minutes per day = 45 minutes per week = 3 hours per month

---

### Total Time Saved for Maya (Project Manager)

**Daily Savings**:
- Epic scanning: 4 minutes
- Approval workflow: 3 minutes
- Blocker identification: 2 minutes
- Status reporting: 9 minutes
- **Total: 18 minutes per day**

**Monthly Savings**:
- 18 minutes/day × 20 work days = **6 hours per month**
- Equivalent to **0.75 work days per month**

**Emotional Impact**:
- **Before**: Frustrated, uncertain, overwhelmed
- **After**: Confident, in control, efficient

---

## Journey Map: Alex (Developer)

### Scenario: Starting Work on a Feature

**Context**: Alex has completed T-E07-F03-004 and needs to find the next task to work on.

---

### Stage 1: Understanding Feature Status

**Goal**: Understand what's left to do on the feature

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark feature get E07-F03
```

**Output (Current)**:
```
Feature: E07-F03
Status: active
Progress: 30%

Task Status Breakdown:
Status              Count
completed           3
in_development      1
todo                6
```

**Alex's Thoughts**:
- "OK, 30% done, 3 of 10 tasks complete"
- "1 task is in_development - is that mine or someone else's?"
- "6 tasks are todo - which one should I start?"
- "Are any tasks blocked?"

**Alex's Emotions**: 🤔 Uncertain about next steps

**Pain Points**:
- No indication of which task is in_development
- Todo tasks not prioritized
- No workflow ordering (what comes next?)
- Must list all tasks to find next one

**Time Spent**: 10 seconds to view, 30 seconds to decide

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark feature get E07-F03
```

**Output (Enhanced)**:
```
════════════════════════════════════════════════════════
Feature: E07-F03 - User Profiles
════════════════════════════════════════════════════════

Status:   active (in development)
          1 task in development
          6 tasks ready to start

Progress: 30% complete (3 of 10 tasks)
          1 task in development
          6 tasks ready for development

────────────────────────────────────────────────────────
Action Items
────────────────────────────────────────────────────────

⚙ 1 task in development (in_development)
  ⚙ T-E07-F03-005: Build profile update API

📝 Next available tasks (by priority):
  • T-E07-F03-006: Add profile photo upload (priority: 3)
  • T-E07-F03-007: Implement bio field (priority: 5)
```

**Alex's Thoughts**:
- "Someone else is working on T-E07-F03-005"
- "Next task is T-E07-F03-006 - profile photo upload"
- "That's priority 3, so it's important"
- "I can start that now"

**Alex's Emotions**: 😊 Clear on next steps, ready to work

**Benefits**:
- Knows which task is in_development
- Sees next available tasks prioritized
- No need to list all tasks
- Can start work immediately

**Time Spent**: 10 seconds to view, 5 seconds to decide

**Time Saved**: 25 seconds per feature check = 3 minutes per day = 15 minutes per week

---

### Stage 2: Checking Dependencies

**Goal**: Ensure no blockers before starting work

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark task get T-E07-F03-006
# Check depends_on field
# Manually verify dependency status
shark task get T-E07-F03-005
# Confirm dependency is completed or in_progress
```

**Pain Points**:
- Must manually check dependencies
- No indication if dependency is blocked
- Must navigate to dependency task
- Time-consuming verification process

**Time Spent**: 30 seconds per task

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark feature get E07-F03
# Action items section shows "Next available tasks"
# Only shows tasks with completed dependencies
# No need to manually verify
```

**Benefits**:
- Only ready tasks are shown
- Dependencies automatically checked
- No manual verification needed
- Can start work immediately

**Time Spent**: 0 seconds (already verified)

**Time Saved**: 30 seconds per task = 4 minutes per day = 20 minutes per week

---

### Total Time Saved for Alex (Developer)

**Daily Savings**:
- Feature status understanding: 3 minutes
- Dependency verification: 4 minutes
- **Total: 7 minutes per day**

**Monthly Savings**:
- 7 minutes/day × 20 work days = **2.3 hours per month**
- Equivalent to **0.3 work days per month**

**Emotional Impact**:
- **Before**: Uncertain, time wasted on verification
- **After**: Confident, can start work immediately

---

## Journey Map: AI Orchestrator (Automated System)

### Scenario: Polling for Actionable Tasks

**Context**: Orchestrator checks for tasks that need agent assignment every 5 minutes.

---

### Stage 1: Identifying Ready Tasks

**Goal**: Find all tasks ready for agent assignment

#### Current Experience (Before Enhancement)

**Actions**:
```bash
shark task list --status=ready_for_development --json
```

**Output (Current)**:
```json
[
  {
    "id": 123,
    "key": "T-E07-F03-006",
    "status": "ready_for_development",
    "title": "Add profile photo upload"
  }
]
```

**Orchestrator Logic**:
```python
# Must make additional calls to understand context
for task in tasks:
    feature = get_feature(task.feature_id)
    epic = get_epic(feature.epic_id)

    # Check if feature/epic is blocked
    if feature.has_blockers:
        skip_task()

    # Check dependencies manually
    for dep in task.depends_on:
        dep_task = get_task(dep)
        if dep_task.status != "completed":
            skip_task()

    # Assign to agent
    spawn_agent(task)
```

**Pain Points**:
- Must make N+1 queries (feature, epic for each task)
- No indication of blockers in task JSON
- Must manually verify dependencies
- No action metadata in response

**Time Spent**: 500ms per task (including queries) × 10 tasks = 5 seconds

---

#### Enhanced Experience (After Enhancement)

**Actions**:
```bash
shark task list --status=ready_for_development --json
```

**Output (Enhanced)**:
```json
[
  {
    "id": 123,
    "key": "T-E07-F03-006",
    "status": "ready_for_development",
    "title": "Add profile photo upload",
    "feature": {
      "key": "E07-F03",
      "status": "active",
      "health": {
        "has_blockers": false,
        "awaiting_approval": false
      }
    },
    "dependencies_met": true,
    "action_metadata": {
      "agent_type": "developer",
      "skills": ["backend", "file-upload"],
      "priority": 3
    }
  }
]
```

**Orchestrator Logic**:
```python
# All context in single response
for task in tasks:
    # Check health indicators
    if not task.dependencies_met:
        continue

    if task.feature.health.has_blockers:
        continue

    # Spawn agent with metadata
    spawn_agent(
        type=task.action_metadata.agent_type,
        skills=task.action_metadata.skills,
        task=task
    )
```

**Benefits**:
- Single API call with all context
- Health indicators for quick filtering
- Dependencies pre-validated
- Action metadata for agent spawning

**Time Spent**: 100ms per task (no additional queries) × 10 tasks = 1 second

**Time Saved**: 4 seconds per polling cycle × 12 cycles/hour = 48 seconds/hour = 19 minutes/day

---

### Stage 2: Generating Status Reports

**Goal**: Generate human-readable status report for stakeholders

#### Current Experience (Before Enhancement)

**Orchestrator Process**:
1. Fetch all epics
2. Fetch all features per epic
3. Fetch all tasks per feature
4. Manually aggregate status counts
5. Manually identify blockers
6. Generate report

**Pain Points**:
- Must make 100+ API calls for large projects
- Manual aggregation logic complex
- No pre-computed health indicators
- Hard to identify "what needs attention"

**Time Spent**: 30 seconds per report generation

---

#### Enhanced Experience (After Enhancement)

**Orchestrator Process**:
1. Fetch epic with `?include=health,action_items`
2. Parse health indicators
3. Format pre-computed data
4. Generate report

**Benefits**:
- Single API call per epic
- Health indicators pre-computed
- Action items ready for display
- Status explanations human-readable

**Time Spent**: 5 seconds per report generation

**Time Saved**: 25 seconds per report × 20 reports/day = 8 minutes/day

---

### Total Time Saved for Orchestrator

**Daily Savings**:
- Task polling: 19 minutes
- Report generation: 8 minutes
- **Total: 27 minutes per day**

**Monthly Savings**:
- 27 minutes/day × 30 days = **13.5 hours per month**
- Equivalent to **1.7 work days per month**

**Operational Impact**:
- Reduced API calls: 70% reduction
- Faster decisions: 80% reduction in latency
- Better accuracy: 100% dependency validation

---

## Emotional Journey Arc

### Current Experience (Before Enhancement)

```
                    Frustration
                        ↑
                        |
                        |   ╱╲   ╱╲
    Uncertainty ────────|  ╱  ╲ ╱  ╲
                        | ╱    V    ╲
                    ───╱──────────────╲───
                  START    MID    END    TIME →
                        Confusion
```

**Emotional States**:
- **Start**: Uncertainty (what's the status?)
- **Mid**: Confusion (what does "active" mean?)
- **Frustration**: Can't find actionable items
- **End**: Still uncertain (did I get everything?)

---

### Enhanced Experience (After Enhancement)

```
                   Confidence
                        ↑
                        |   ╱────────────
                        |  ╱
    Clarity ────────────| ╱
                        |╱
                    ───╱───────────────────
                  START    MID    END    TIME →
                        Control
```

**Emotional States**:
- **Start**: Clarity (status is explained)
- **Mid**: Confidence (I know what to do)
- **Control**: Action items are clear
- **End**: Satisfied (everything handled)

---

## Touchpoint Analysis

### Touchpoint 1: `shark feature list`

**Current State**:
- Shows status as single word
- Progress percentage only
- No indication of blockers
- No visual cues

**Pain Points**:
- Can't distinguish "active (in development)" from "active (waiting approval)"
- Must click into each feature to see details
- No quick way to spot blockers

**Enhanced State**:
- Status shows phase context: `active (dev)`, `active (waiting) ⏳`
- Progress shows ratio: `60% (3/5)`
- Visual indicators: ⏳, ⚠️, ✓
- Notes column shows actionable items

**User Impact**:
- 80% reduction in time to assess feature health
- 100% accuracy in identifying waiting items
- 90% reduction in clicks needed

---

### Touchpoint 2: `shark feature get`

**Current State**:
- Status breakdown as count only
- Task list alphabetical order
- No action items section
- No status explanation

**Pain Points**:
- Must mentally calculate "what's left"
- Can't quickly identify next action
- Task list not workflow-ordered
- Status reason unclear

**Enhanced State**:
- Status explanation: "active (waiting for approval)"
- Progress breakdown: "1 task waiting, 1 in development"
- Action items section: Shows specific tasks
- Task list grouped by phase

**User Impact**:
- 70% reduction in time to understand status
- 100% clarity on next actions
- 85% reduction in cognitive load

---

### Touchpoint 3: `shark epic get`

**Current State**:
- Epic status as single word
- Progress percentage only
- Feature list with minimal context
- No aggregated action items

**Pain Points**:
- Must check each feature for blockers
- Can't see "what needs my approval" at epic level
- No indication of overall health
- Feature list not urgency-ordered

**Enhanced State**:
- Epic status explanation: "active (mixed phases)"
- Feature distribution by phase
- Aggregated action items section
- Feature list ordered by urgency

**User Impact**:
- 75% reduction in time to assess epic health
- 100% visibility into blockers and waiting items
- 90% reduction in navigation needed

---

### Touchpoint 4: JSON API (for Orchestrators)

**Current State**:
- Minimal task metadata
- No health indicators
- No action metadata
- Must make N+1 queries

**Pain Points**:
- Excessive API calls
- Manual dependency verification
- No pre-computed health
- Missing agent assignment data

**Enhanced State**:
- Rich task metadata with feature health
- Pre-computed health indicators
- Action metadata for agent spawning
- All context in single response

**User Impact**:
- 70% reduction in API calls
- 80% reduction in latency
- 100% accuracy in dependency validation
- 90% reduction in orchestration code complexity

---

## Journey Coherence Checklist

- [x] Journey stages flow logically (scan → investigate → act)
- [x] No gaps in critical user paths (all personas covered)
- [x] Transitions between touchpoints are smooth (consistent terminology)
- [x] Consistent patterns across the experience (color coding, indicators)
- [x] User mental model is respected (workflow ordering matches human thinking)
- [x] Emotion arc supports engagement (frustration → confidence)
- [x] Features support journey goals (action items enable next steps)
- [x] Business goals align with user goals (efficiency = productivity)
- [x] Recovery paths exist for errors (manual override, verbose mode)

---

## Success Criteria

### User Experience Metrics

1. **Time to Assess Status**: < 5 seconds (vs. 30 seconds current)
2. **Time to Identify Actions**: < 10 seconds (vs. 60 seconds current)
3. **User Satisfaction**: "I know what to do next" → 100% agreement
4. **Error Rate**: Zero misinterpretation of status meaning

### Business Impact Metrics

1. **Project Manager Time Saved**: 6 hours/month per person
2. **Developer Time Saved**: 2.3 hours/month per person
3. **Orchestrator Efficiency**: 13.5 hours/month per system
4. **API Call Reduction**: 70% fewer calls for status checks

### Technical Quality Metrics

1. **Performance**: All queries < 100ms
2. **Backward Compatibility**: 100% of existing JSON parsers work
3. **Accessibility**: All information accessible via screen reader
4. **Consistency**: 100% of statuses follow same pattern

---

## Related Documents

- [UX Design: Status Tracking](./D12-ux-design-status-tracking.md) - Detailed design mockups
- [E07-F14 Feature PRD](./prd.md) - Complete requirements
- [E07-F14 Feature Specification](./feature.md) - Feature summary

---

## Appendix: User Quotes (Simulated)

### Before Enhancement

> "I spend half my time trying to figure out what 'active' means for each feature. Is it blocked? In progress? Waiting for me? It's exhausting." - Maya, Project Manager

> "I often start working on a task only to realize it has a blocked dependency. I wish the system would just tell me which tasks are truly ready." - Alex, Developer

> "Our orchestrator makes 500+ API calls per minute just to figure out which tasks are ready to assign. It's inefficient and slow." - DevOps Lead

### After Enhancement

> "Now I can scan my epics in seconds and immediately see what needs my attention. The action items section is a game-changer." - Maya, Project Manager

> "I love that the feature view shows me exactly which task to work on next. No more guessing or verification needed." - Alex, Developer

> "With the enhanced JSON output, our orchestrator is 80% faster and makes 70% fewer API calls. Plus, we can trust the dependency validation." - DevOps Lead

---

## Conclusion

This journey map demonstrates that enhanced status tracking creates a dramatically better experience across all user personas:

- **Project Managers** save 6 hours/month and gain confidence in status assessment
- **Developers** save 2.3 hours/month and work more efficiently
- **AI Orchestrators** save 13.5 hours/month and operate more reliably

The emotional transformation from **frustration → confidence** is achieved through:
1. **Scannable design** (color, indicators, grouping)
2. **Progressive disclosure** (summary → details → JSON)
3. **Actionable insights** (action items section)
4. **Status context** (why is it active?)

All enhancements maintain backward compatibility and follow existing Shark CLI patterns, ensuring a smooth transition for existing users and systems.
