# User Journeys

**Epic**: [Sprint Management & Planning System](./epic.md)

---

## Overview

This document maps the key user workflows enabled by sprint management. There are four primary journeys covering the full sprint lifecycle: creation, planning, monitoring, and closure/retrospective. Each journey is documented for the PM persona; the AI orchestrator follows the same steps programmatically with `--json` output.

---

## Journey 1: Sprint Creation & Configuration

**Persona**: PM / Scrum Master

**Goal**: Create a new sprint with appropriate dates, capacity targets, and a sprint goal

**Preconditions**:
- Shark project is initialized with the advanced workflow profile
- At least one epic with features and tasks exists in the backlog
- Agent capacity defaults are configured (or will be set per sprint)

### Happy Path

1. **Create Sprint**
   - User action: `shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01`
   - System response: Creates sprint record with `planning` status, generates key `S024`, displays confirmation with sprint details
   - Expected outcome: Sprint exists in planning status, ready to accept task assignments

2. **Set Sprint Goal**
   - User action: `shark sprint update S024 --goal="Complete E07-F34 template enrichment and start E19 sprint infra"`
   - System response: Updates sprint goal text, confirms update
   - Expected outcome: Sprint has a documented goal visible in `shark sprint get S024`

3. **Configure Agent Capacity**
   - User action: `shark sprint capacity set S024 --agent=backend --points=21` (repeat per agent type)
   - System response: Records capacity target per agent type for this sprint
   - Expected outcome: `shark sprint capacity show S024` displays capacity targets for all configured agent types

4. **Verify Sprint Configuration**
   - User action: `shark sprint get S024`
   - System response: Displays sprint name, dates, goal, status, and capacity summary
   - Expected outcome: PM confirms all sprint parameters are correct before proceeding to planning

**Success Outcome**: A fully configured sprint in `planning` status with dates, goal, and capacity targets defined.

### Alternative Paths

**Alt Path A: Sprint with Default Capacity**
- **Trigger**: Team has stable capacity that does not change sprint-to-sprint
- **Branch Point**: After Step 1
- **Flow**:
  1. System applies default capacity from configuration (if configured)
  2. PM skips capacity set commands
- **Outcome**: Sprint uses default capacity values; PM can override specific agent types if needed

**Alt Path B: Overlapping Sprint Dates**
- **Trigger**: PM creates a sprint with dates overlapping an existing active sprint
- **Branch Point**: Step 1
- **Flow**:
  1. System warns about date overlap with the active sprint
  2. PM confirms or adjusts dates
- **Outcome**: Overlap is allowed (teams may run parallel sprints during transition) but warning is logged

---

## Journey 2: Sprint Planning

**Persona**: PM / Scrum Master

**Goal**: Populate the sprint backlog with prioritized tasks that fit within agent capacity

**Preconditions**:
- Sprint exists in `planning` status (Journey 1 complete)
- Backlog contains tasks in `ready_for_development` or earlier statuses
- Capacity targets are set for the sprint

### Happy Path

1. **View Available Backlog**
   - User action: `shark sprint plan S024`
   - System response: Displays planning view with: (a) unassigned backlog tasks sorted by priority and dependency order, (b) current sprint assignment count and capacity utilization per agent type, (c) sprint readiness score
   - Expected outcome: PM sees a clear picture of what can be added to the sprint

2. **Assign Individual Tasks**
   - User action: `shark sprint add S024 E07-F01-001`
   - System response: Assigns task to sprint, updates capacity utilization display, warns if agent type capacity exceeded
   - Expected outcome: Task appears in sprint backlog; capacity meter updates

3. **Bulk Assign from Feature**
   - User action: `shark sprint add S024 --bulk E07-F34`
   - System response: Assigns all `ready_for_development` tasks from E07-F34 to the sprint, displays count added and updated capacity utilization
   - Expected outcome: Multiple tasks added efficiently; PM can remove individual tasks if needed

4. **Review Sprint Readiness**
   - User action: `shark sprint readiness S024`
   - System response: Displays readiness score (0-100) based on: capacity utilization (not over/under), dependency satisfaction (no blocked dependencies outside sprint), task estimation coverage, and balanced agent allocation
   - Expected outcome: PM sees quantified readiness and specific items to address

5. **Remove Over-Allocated Tasks**
   - User action: `shark sprint remove S024 E07-F01-003`
   - System response: Removes task from sprint, updates capacity utilization
   - Expected outcome: Capacity is within targets; readiness score improves

6. **Start Sprint**
   - User action: `shark sprint start S024`
   - System response: Changes sprint status to `active`, validates no other sprint is currently active (blocks if so), confirms sprint start
   - Expected outcome: Sprint is active; tasks are visible in `shark sprint backlog S024`

**Success Outcome**: Sprint has a well-scoped backlog within capacity constraints and is activated for execution.

### Alternative Paths

**Alt Path A: AI Orchestrator Auto-Planning**
- **Trigger**: Orchestrator receives instruction to plan next sprint
- **Branch Point**: Replaces Steps 1-5
- **Flow**:
  1. Orchestrator runs `shark sprint plan S024 --json` to get backlog and capacity data
  2. Orchestrator selects tasks by priority order, respecting dependency chains, up to capacity limits per agent type
  3. Orchestrator runs `shark sprint add S024 <task>` for each selected task
  4. Orchestrator runs `shark sprint readiness S024 --json` and reports score to PM
  5. PM reviews and approves or adjusts
- **Outcome**: Sprint is auto-planned with human approval gate

**Alt Path B: Capacity Exceeded Warning**
- **Trigger**: Task assignment would exceed agent type capacity
- **Branch Point**: Step 2 or Step 3
- **Flow**:
  1. System displays warning: "Adding this task exceeds backend capacity (23/21 points)"
  2. Task is still assigned (capacity is advisory, not enforced)
  3. Readiness score decreases to reflect overallocation
- **Outcome**: PM is informed but not blocked; overallocation is tracked

### Critical Decision Points

- **Decision at Step 4**: If readiness score is below 60, the PM must decide whether to adjust scope (remove tasks), increase capacity estimates, or accept the risk and proceed.

---

## Journey 3: Sprint Monitoring

**Persona**: PM / Scrum Master

**Goal**: Track sprint progress during execution and identify blocking issues

**Preconditions**:
- Sprint is in `active` status
- Tasks are being worked on by agents

### Happy Path

1. **View Sprint Backlog**
   - User action: `shark sprint backlog S024`
   - System response: Displays all tasks assigned to the sprint grouped by status (in_development, ready_for_code_review, in_qa, completed, blocked), with progress percentage
   - Expected outcome: PM has a snapshot of current sprint state

2. **Check Burndown**
   - User action: `shark sprint burndown S024`
   - System response: Displays text-based burndown chart showing ideal vs. actual task completion over time, with days remaining
   - Expected outcome: PM can see if sprint is on track, ahead, or behind schedule

3. **Identify Blocked Items**
   - User action: `shark sprint backlog S024 --blocked`
   - System response: Lists blocked tasks with blocking reason and days blocked
   - Expected outcome: PM can prioritize unblocking actions

4. **Check Velocity Trend**
   - User action: `shark sprint velocity`
   - System response: Shows completed points/tasks per sprint for the last N sprints (default 5) with average velocity
   - Expected outcome: PM has context for whether current sprint pace is consistent with historical performance

**Success Outcome**: PM has full visibility into sprint health and can make data-driven decisions about interventions.

### Alternative Paths

**Alt Path A: Mid-Sprint Scope Change**
- **Trigger**: Urgent bug or change-card added mid-sprint
- **Branch Point**: During active sprint
- **Flow**:
  1. PM runs `shark sprint add S024 B042` to add urgent bug to sprint
  2. System warns about scope change and capacity impact
  3. PM optionally removes a lower-priority task to compensate
- **Outcome**: Sprint scope is adjusted; burndown reflects the change

---

## Journey 4: Sprint Close & Retrospective

**Persona**: PM / Scrum Master

**Goal**: Close the sprint, handle incomplete work, and review sprint performance data

**Preconditions**:
- Sprint is in `active` status
- Sprint end date is reached (or PM decides to close early)

### Happy Path

1. **Review Sprint State Before Close**
   - User action: `shark sprint summary S024`
   - System response: Displays pre-close summary: completed tasks (count and points), incomplete tasks, carryover candidates, velocity for this sprint, comparison to average velocity
   - Expected outcome: PM understands what was accomplished and what remains

2. **Close Sprint**
   - User action: `shark sprint close S024 --carryover=next`
   - System response: Changes sprint status to `closing`, moves incomplete tasks to the next `planning` sprint (or back to backlog if `--carryover=backlog`), generates sprint completion record, displays final summary
   - Expected outcome: Sprint is in `completed` status; carryover tasks are reassigned

3. **Review Retrospective Data**
   - User action: `shark sprint summary S024 --detailed`
   - System response: Displays detailed retrospective data: (a) planned vs. completed tasks, (b) velocity (this sprint vs. trailing average), (c) tasks added mid-sprint, (d) tasks removed mid-sprint, (e) average cycle time by phase, (f) agent utilization vs. capacity, (g) blocked time analysis
   - Expected outcome: PM has all data needed for a productive retrospective discussion

4. **Archive Sprint**
   - User action: `shark sprint archive S024`
   - System response: Moves sprint to `archived` status; sprint data is retained for velocity calculations but excluded from default listing
   - Expected outcome: Sprint history is preserved; active views are clean

**Success Outcome**: Sprint is closed with complete performance data, incomplete work is handled, and historical data is available for future planning.

### Alternative Paths

**Alt Path A: Carryover to Backlog**
- **Trigger**: No next sprint is in planning status, or PM wants manual re-prioritization
- **Branch Point**: Step 2
- **Flow**:
  1. PM runs `shark sprint close S024 --carryover=backlog`
  2. Incomplete tasks are unassigned from any sprint (return to general backlog)
  3. PM manually re-assigns in future sprint planning
- **Outcome**: Clean separation between sprints; no automatic assumptions about future scope

**Alt Path B: Sprint Extended**
- **Trigger**: PM decides to extend sprint end date rather than close
- **Branch Point**: Before Step 2
- **Flow**:
  1. PM runs `shark sprint update S024 --end=2026-04-08`
  2. Sprint remains active with new end date
  3. Burndown recalculates with extended timeline
- **Outcome**: Sprint deadline is extended; no carryover needed

---

*See also*: [Requirements](./requirements.md)
