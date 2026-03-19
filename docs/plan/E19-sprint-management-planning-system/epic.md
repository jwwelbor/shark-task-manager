---
epic_key: E19
title: Sprint Management & Planning System
description: Add comprehensive sprint management features for project planning, iteration tracking, velocity analytics, and agent capacity management
---

# Sprint Management & Planning System

**Epic Key**: E19

---

## Goal

### Problem

Shark Task Manager has no concept of sprints or iterations. All work items exist in a flat timeline with no grouping by time-boxed planning periods. This creates several concrete problems:

1. **No sprint planning workflow.** PMs and AI orchestrators cannot define a time-boxed iteration, assign tasks to it, and track progress against that iteration's scope. Work planning happens informally through manual status scanning and mental bookkeeping.

2. **No velocity or capacity data.** Without sprint boundaries, there is no way to calculate team velocity (story points or task count completed per iteration), predict future capacity, or identify trends in delivery performance. Planning relies on intuition rather than historical data.

3. **No sprint-level analytics.** Retrospective discussions lack data. There is no burndown chart, no completed-vs-planned comparison, no sprint summary report. The `shark analytics` command only operates at epic and project levels.

4. **No agent capacity tracking.** The AI orchestrator cannot determine how many tasks an agent type can handle in a given period. There is no capacity model to balance workload across agent types (backend, frontend, qa, ba) or detect overallocation.

5. **Manual date-range filtering for reporting.** Any time-scoped analysis requires manual date filters. There is no semantic grouping of "the work we committed to this iteration" versus "the work that carried over."

### Solution

Introduce **sprints** as a first-class entity in Shark with dedicated CLI commands, database schema, and analytics. A sprint is a named, time-boxed iteration (typically 1-4 weeks) to which tasks are assigned. The system provides:

- **Sprint lifecycle management**: Create, plan, start, close, and archive sprints via `shark sprint` commands.
- **Sprint planning**: A planning view that shows available backlog, agent capacity, and sprint readiness scoring to help PMs and AI orchestrators build sprint scope.
- **Task-to-sprint assignment**: Tasks can be assigned to sprints individually or in bulk. Tasks carry over automatically when a sprint closes if not completed.
- **Sprint analytics**: Velocity calculation from historical sprints, burndown visualization for the active sprint, and sprint summary reports for retrospectives.
- **Agent capacity model**: Define capacity per agent type per sprint. The system tracks allocation against capacity and flags overcommitment.

### Impact

- Sprint planning time reduced from ad-hoc scanning to a structured workflow that surfaces the right data for planning decisions
- Historical velocity data enables evidence-based sprint scoping, reducing both overcommitment and underutilization
- Sprint burndown and summary analytics replace manual date-range filtering, saving time during standups and retrospectives
- Agent capacity tracking prevents overallocation and enables the AI orchestrator to auto-plan sprints within defined constraints

---

## Business Value

**Rating**: High

Sprint management is a foundational capability for any iterative development workflow. Without it, Shark is a task tracker but not a planning tool. This epic transforms Shark into a complete iteration management system, enabling both human PMs and AI orchestrators to plan, execute, and review time-boxed iterations with data-driven decisions. It directly addresses the largest functional gap identified in the AI PM Journey Map analysis (stages 1 and 5), and unblocks the AI orchestrator's ability to autonomously plan and manage sprints.

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - PM/Scrum Master and AI Orchestrator profiles
- **[User Journeys](./user-journeys.md)** - Sprint lifecycle workflows from creation through retrospective
- **[Requirements](./requirements.md)** - Functional and non-functional requirements catalog
- **[Success Metrics](./success-metrics.md)** - KPIs for planning efficiency, velocity accuracy, and adoption
- **[Scope Boundaries](./scope.md)** - Explicit exclusions and future considerations

---

## Quick Reference

**Primary Users**: PM/Scrum Master (human), AI Orchestrator Agent

**Key Features**:
- Sprint entity with create, plan, start, close, archive lifecycle commands
- Sprint planning view with backlog, capacity, and readiness scoring
- Task-to-sprint assignment with bulk operations and automatic carryover
- Velocity calculation and burndown analytics
- Agent capacity model with overallocation detection

**Success Criteria**:
- Sprints can be created, planned, executed, and closed entirely within Shark CLI
- Velocity calculated from 3+ historical sprints with less than 15% variance from actual
- Sprint planning view surfaces all data needed for scoping decisions in a single command

**Dependencies**:
- Builds on existing task status and session tracking (E13 workflow-aware commands)
- Leverages task_sessions table for phase duration analysis
- Integrates with existing `shark status`, `shark analytics`, and `shark get` command patterns

---

## Entity Design Summary

### Sprint (S###)

A **sprint** is a time-boxed iteration with a defined start date, end date, and set of assigned tasks.

**Lifecycle:**
```
planning -> active -> closing -> completed
                   -> cancelled
```

**Key Characteristics:**
- Named entity with start and end dates (e.g., "Sprint 24", 2026-03-18 to 2026-04-01)
- Key format: `S###` (e.g., `S001`, `S024`)
- Only one sprint can be `active` at a time
- Tasks are assigned to sprints; a task can belong to at most one active sprint
- Unfinished tasks carry over to the next sprint on close (configurable)
- Stores sprint goal text for planning context

### CLI Commands

```bash
# Sprint lifecycle
shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01
shark sprint list [--status=active]
shark sprint get S024
shark sprint start S024
shark sprint close S024 [--carryover=next|backlog]
shark sprint archive S024

# Sprint planning
shark sprint plan S024                    # Interactive planning view
shark sprint add S024 E07-F01-001         # Assign task to sprint
shark sprint add S024 --bulk E07-F01      # Assign all ready tasks from feature
shark sprint remove S024 E07-F01-001      # Remove task from sprint
shark sprint backlog S024                 # View sprint backlog

# Sprint analytics
shark sprint velocity                     # Historical velocity chart
shark sprint burndown [S024]              # Current or specified sprint burndown
shark sprint summary S024                 # Sprint retrospective summary

# Capacity management
shark sprint capacity set S024 --agent=backend --points=21
shark sprint capacity show S024           # Show capacity vs. allocation
shark sprint readiness S024               # Sprint readiness score
```

### Database Schema (High-Level)

```
sprints table:
  id, key, name, goal, start_date, end_date, status, created_at, updated_at

sprint_assignments table:
  id, sprint_id, task_id, assigned_at, removed_at

sprint_capacity table:
  id, sprint_id, agent_type, capacity_points, allocated_points
```

---

## Open Questions & Assumptions

No open questions -- all epic-level decisions are resolved.

**Resolved Assumptions:**
1. Sprint key format is `S###` (globally unique, not scoped to epics). This matches the standalone entity pattern established by bugs (`B###`) and change-cards (`CC-###`).
2. Only one sprint can be `active` at a time. This enforces focus and simplifies the "current sprint" concept. Multiple sprints can be in `planning` status simultaneously.
3. Task-to-sprint assignment is a many-to-one relationship (a task belongs to at most one active/planning sprint). Historical assignments are preserved for velocity calculations.
4. Sprint capacity is measured in abstract "points" that map to story points or task count, configurable per team. The system does not enforce hours-based tracking.
5. Automatic carryover on sprint close is the default behavior. Incomplete tasks move to the next planning sprint or back to the backlog, configurable via `--carryover` flag.
6. Sprint analytics (velocity, burndown) require at least 3 completed sprints to produce meaningful predictions. The system operates in "data collection" mode until this threshold is met.

---

*Last Updated*: 2026-03-18
