# Scope Boundaries

**Epic**: [Sprint Management & Planning System](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in this epic. Clear boundaries prevent scope creep and set expectations for what will be addressed in future work.

---

## Out of Scope

### Explicitly Excluded Features

**1. Graphical/Web UI for Sprint Boards**
- **Why It's Out of Scope**: Shark is a CLI-first tool. Building a web-based sprint board (Kanban/Scrum board UI) is a separate epic with its own design, frontend framework, and deployment concerns. Sprint management must work entirely through CLI commands.
- **Future Consideration**: A web dashboard epic could visualize sprint data from the database. The sprint schema is designed to support future UI layers.
- **Workaround**: `shark sprint backlog` and `shark sprint burndown` provide text-based equivalents of board and chart views.

**2. Story Point Estimation Ceremonies**
- **Why It's Out of Scope**: Estimating story points requires team-level estimation ceremonies (planning poker, relative sizing discussions, collaborative re-estimation, confidence intervals) that are beyond CLI sprint management. E19 *consumes* the numeric `size` field delivered by E07-F42 — it does not provide tooling for arriving at those values, recording estimation sessions, or driving multi-agent re-estimation.
- **Future Consideration**: A separate epic could add ceremony-supporting features such as recorded estimation sessions, multi-agent re-estimation prompts, confidence intervals, or size drift analysis across re-estimations.
- **Workaround**: Teams set entity `size` manually (numeric Fibonacci or t-shirt label, both accepted by E07-F42) per their own estimation process. Sprint commands consume whatever value is set; sprints with unsized entities receive a sizing-coverage penalty in the readiness score (REQ-F-013) but are not blocked.

**3. Cross-Team Sprint Coordination**
- **Why It's Out of Scope**: Shark operates as a single-project tool with one database. Multi-team sprint coordination (program-level PI planning, cross-team dependency tracking, shared sprint calendars) requires a fundamentally different data model.
- **Future Consideration**: Could be addressed if Shark adds multi-project or multi-team support.
- **Workaround**: Each team runs independent sprints within their own Shark instance.

**4. Notification and Reminder System**
- **Why It's Out of Scope**: Automated notifications (sprint starting tomorrow, sprint ending, capacity exceeded) require an event/notification infrastructure that does not exist in Shark. Sprint commands provide the data; alerting is a separate concern.
- **Future Consideration**: A notification epic could subscribe to sprint lifecycle events and send alerts via configured channels.
- **Workaround**: PMs can run `shark sprint get` or `shark sprint backlog` on a regular cadence. AI orchestrators can poll sprint state.

**5. Time Tracking and Logging**
- **Why It's Out of Scope**: Sprint capacity uses abstract points, not hours. Detailed time tracking (start/stop timers, logged hours per task, timesheet reports) is a different feature with different privacy and compliance requirements.
- **Future Consideration**: Time tracking could integrate with sprint capacity for hours-based planning in a future epic.
- **Workaround**: Task sessions (`task_sessions` table) track phase durations automatically. Sprint summary reports include average cycle time by phase.

**6. Sprint Templates**
- **Why It's Out of Scope**: Pre-defined sprint templates (recurring task sets, standard ceremonies, boilerplate goals) add complexity to the initial sprint implementation without clear demand. The auto-creation feature (REQ-F-016, Could Have) handles cadence but not content templates.
- **Future Consideration**: Sprint templates could be added once the base sprint system is proven and teams express the need.
- **Workaround**: AI orchestrators can implement template-like behavior by scripting sprint creation and task assignment commands.

---

### Edge Cases & Scenarios Not Covered

**1. Concurrent Sprint Modification by Multiple Users**
- **Impact**: Low -- Shark is typically used by one PM or one AI orchestrator at a time
- **Rationale**: SQLite WAL mode handles concurrent reads, and sprint modifications are infrequent write operations. Full optimistic locking is unnecessary overhead.
- **Mitigation**: Last-write-wins for sprint attribute updates. Task assignment uses database constraints to prevent double-assignment.

**2. Sprints Spanning Calendar Year Boundaries**
- **Impact**: Low -- this is a valid but uncommon scenario
- **Rationale**: Date handling uses ISO 8601 format which has no year-boundary issues. No special handling needed.
- **Mitigation**: Standard date comparison operations work correctly across year boundaries.

**3. Very Long Sprints (>30 days)**
- **Impact**: Low -- unusual but not prohibited
- **Rationale**: The system does not enforce maximum sprint duration. Burndown charts may become less useful for very long sprints.
- **Mitigation**: System warns if sprint duration exceeds 30 days but does not block creation.

**4. Retrospective Deletion of Completed Sprints**
- **Impact**: Medium -- deleting completed sprints would corrupt velocity history
- **Rationale**: Completed sprint data is used for velocity calculations. Allowing deletion would silently change historical metrics.
- **Mitigation**: Delete is only allowed for sprints in `planning` status. Completed/archived sprints cannot be deleted (only archived for exclusion from active views).

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Sprints as Epic Metadata**
- **Description**: Instead of a separate sprint entity, attach sprint information (start date, end date, sprint number) as metadata on existing epics or features
- **Pros**: No new entity type; simpler schema; reuses existing infrastructure
- **Cons**: Conflates iteration planning with feature scope; sprints cut across epics and features; no standalone sprint lifecycle; impossible to track velocity independently
- **Decision Rationale**: Sprints are fundamentally cross-cutting -- a single sprint contains tasks from multiple features and epics. They need their own entity with independent lifecycle management.

**Alternative 2: Tag-Based Sprint Assignment**
- **Description**: Instead of a sprint entity, use tags (e.g., `sprint:24`) on tasks to group them into sprints
- **Pros**: Flexible; no schema changes; works with existing tag system (if one existed)
- **Cons**: No sprint lifecycle management; no capacity model; tags have no dates or status; velocity calculation requires parsing tag values; no enforcement of single-sprint-per-task
- **Decision Rationale**: Tags provide grouping but not management. Sprint management requires lifecycle states, dates, capacity, and analytics that tags cannot provide.

**Alternative 3: Integration with External Sprint Tools (Jira, Linear)**
- **Description**: Instead of building sprint management, integrate with external tools via API for sprint planning
- **Pros**: Leverages mature sprint management UIs; avoids reinventing the wheel
- **Cons**: Breaks single-source-of-truth principle; adds external dependency; requires network access; complex sync logic; AI orchestrator cannot operate autonomously
- **Decision Rationale**: Shark's core value is being a self-contained, offline-capable, AI-agent-friendly project management tool. External dependencies contradict this.

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| Web Dashboard for Sprint Visualization | Medium | Depends on E19 for sprint data model |
| Story Point Estimation Ceremonies | Low | Builds on E07-F42's `size` field; layers ceremony tooling on top |
| Sprint Retrospective Automation | Low | Depends on E19 for sprint summary data |
| Multi-Team Program Increment Planning | Low | Depends on multi-project support |
| Sprint-Aware Notification System | Low | Depends on E19 + notification infrastructure |

---

*See also*: [Requirements](./requirements.md)
