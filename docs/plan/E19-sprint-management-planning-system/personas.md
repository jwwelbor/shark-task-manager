# User Personas

**Epic**: [Sprint Management & Planning System](./epic.md)

---

## Overview

Sprint management serves two fundamentally different user types: a human PM/Scrum Master who makes strategic planning decisions and facilitates team ceremonies, and an AI orchestrator agent that automates sprint planning and task assignment within defined constraints. Both interact with the same sprint system but have different goals, interaction patterns, and success criteria.

---

## Primary Personas

### Persona 1: PM / Scrum Master (Human)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Project Manager or Scrum Master managing an AI-augmented development team
- **Experience Level**: Experienced with iterative development methodologies; moderate CLI proficiency
- **Key Characteristics**:
  - Plans 1-4 week sprints based on team capacity and business priorities
  - Runs sprint ceremonies (planning, standups, retrospectives) and needs data to support those discussions
  - Balances planned feature work against reactive work (bugs, change-cards)
  - Evaluates team performance trends over multiple sprints to improve forecasting

**Goals Related to This Epic**:
1. Create and scope sprints in under 10 minutes using capacity data and backlog priority ordering
2. Monitor sprint progress daily via burndown and blocking item views without switching to external tools
3. Run retrospectives with accurate data on planned-vs-completed, velocity trends, and cycle time per task phase
4. Forecast future sprint capacity using historical velocity data to set realistic commitments

**Pain Points This Epic Addresses**:
- No way to group tasks into time-boxed iterations -- sprint planning happens in spreadsheets or mentally
- Velocity is unknown because there are no sprint boundaries to measure completed work per iteration
- Retrospective data requires manual date-range filtering and cross-referencing multiple `shark analytics` outputs
- No visibility into agent workload distribution -- some agent types may be overallocated while others sit idle

**Success Looks Like**:
The PM can run an entire sprint lifecycle (plan, monitor, close, retro) using only `shark sprint` commands. Sprint planning takes under 10 minutes because the system surfaces backlog items sorted by priority, shows agent capacity vs. allocation, and provides a readiness score. Retrospectives are data-driven because sprint summary reports show velocity, completion rate, carryover items, and cycle time breakdowns.

---

### Persona 2: AI Orchestrator Agent

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Automated agent that plans and dispatches development work across specialized agent types
- **Experience Level**: Programmatic consumer of `shark` CLI commands with `--json` output
- **Key Characteristics**:
  - Operates autonomously within defined capacity constraints
  - Makes task assignment decisions based on priority, dependencies, and agent availability
  - Needs deterministic, machine-readable outputs for all sprint operations
  - Runs sprint planning algorithms that optimize for throughput and dependency resolution

**Goals Related to This Epic**:
1. Auto-create sprints on a configured cadence (e.g., every 2 weeks) without human intervention
2. Populate sprint backlog by selecting highest-priority tasks that fit within agent capacity constraints
3. Detect overallocation and rebalance work across agent types before sprint start
4. Generate sprint readiness reports for human review before sprint activation

**Pain Points This Epic Addresses**:
- No sprint entity to programmatically create and manage iterations
- No capacity model to constrain task assignment -- the orchestrator has no way to know when an agent type is overloaded
- No `--json` output for sprint planning data that the orchestrator can consume for automated decision-making
- No readiness scoring to determine if a sprint is well-formed before starting it

**Success Looks Like**:
The orchestrator can programmatically create a sprint, query the backlog sorted by priority and dependency order, assign tasks up to capacity limits per agent type, check readiness score, and report the plan to the human PM for approval -- all through `shark sprint` commands with `--json` output. Sprint planning that previously required manual PM intervention can be auto-generated and presented for review.

---

## Secondary Personas

- **Developer Agent**: Indirectly affected -- sees their assigned tasks scoped to a sprint backlog view rather than a flat list. Benefits from clearer "what to work on next" ordering within sprint scope.
- **QA Agent**: Indirectly affected -- sprint summary reports include QA phase metrics (tasks in qa_failed, average verification time) that inform QA capacity planning.

---

## Persona Validation Notes

These personas are derived from the AI PM Journey Map analysis (dev-artifacts/2026-01-10-task-command-ux-analysis/) which identified sprint planning as the largest functional gap (stages 1 and 5). The PM persona reflects the human user operating Shark as described in that journey map. The AI orchestrator persona reflects the autonomous planning capability described in the architecture documentation. Confidence is high for the PM persona (directly observed gap) and moderate for the AI orchestrator (projected capability based on architecture direction).

---

*See also*: [User Journeys](./user-journeys.md)
