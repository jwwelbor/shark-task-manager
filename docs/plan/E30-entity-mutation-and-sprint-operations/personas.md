# User Personas

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)

---

## Overview

This epic serves the people who keep Shark data current during day-to-day work: they need to change entity metadata, maintain dependency links, and move items through Sprint planning without falling back to external tools.

---

## Primary Personas

### Persona 1: Shark Maintainer

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Repository maintainer or internal operator responsible for keeping Shark data accurate
- **Experience Level**: High familiarity with Shark workflows, moderate-to-high technical proficiency
- **Key Characteristics**:
  - Needs quick correction of entity data
  - Values consistency and auditability
  - Comfortable with explicit workflow transitions

**Goals Related to This Epic**:
1. Update titles, descriptions, priorities, and other metadata without leaving the viewer.
2. Add notes and dependency links as part of the normal maintenance flow.
3. Preserve status history and workflow validation while making edits.

**Pain Points This Epic Addresses**:
- Routine updates require too many context switches today.
- Dependency management is too detached from the entity surfaces where it matters.
- Status corrections are safer when the UI surfaces the workflow rather than hiding it.

**Success Looks Like**:
The maintainer can inspect an item, make the needed change, and confirm the update from the same surface in one flow. The change is validated, recorded, and visible in history.

### Persona 2: Sprint Planner

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Person responsible for preparing and adjusting sprint scope
- **Experience Level**: Familiar with task state, capacity, and sprint planning concepts
- **Key Characteristics**:
  - Needs to compare candidate work quickly
  - Wants explicit control over planning actions
  - Prefers to keep readiness and capacity visible

**Goals Related to This Epic**:
1. Stage items into the sprint with clear, explicit actions.
2. Review readiness, capacity, and work-in-progress while planning.
3. Move between Sprint overview, planning, and reporting without losing context.

**Pain Points This Epic Addresses**:
- Planning controls exist only as placeholders today.
- The planner cannot currently use the Sprint surface to actually shape the backlog.
- Report data and plan data live in different mental models unless the UI connects them.

**Success Looks Like**:
The planner can use Sprint mode as the main working surface for shaping sprint scope and understanding whether the team is over capacity or blocked.

### Persona 3: Contributor

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Developer or contributor making a small update to a task or feature
- **Experience Level**: Varies from new contributor to regular contributor
- **Key Characteristics**:
  - Wants the smallest safe edit path
  - Benefits from clear validation and history
  - Often needs to update notes, dependencies, or status after implementation

**Goals Related to This Epic**:
1. Make local edits to work items without having to navigate away from the viewer.
2. See how their change affects status and dependency flow.
3. Keep the change traceable for reviewers and maintainers.

**Pain Points This Epic Addresses**:
- Small updates are disproportionately expensive today.
- The path from implementation to status update is not unified.
- Reviewers need clear history, not silent mutation.

**Success Looks Like**:
The contributor can finish implementation, attach notes, and advance status from a single workflow that makes the audit trail obvious.

---

## Secondary Personas

- **Automation / AI Agent**: Can use the API for scripted updates and sprint maintenance once mutation endpoints are available.

---

## Persona Validation Notes

These personas are inferred from the current Shark workflow and the viewer's existing use cases, not from a dedicated user research program. Confidence is highest for the maintainer/contributor roles because they map directly to the existing CLI and viewer workflows; the sprint planner role is slightly more inferred but still grounded in the current Sprint mode design.

*See also*: [User Journeys](./user-journeys.md)
