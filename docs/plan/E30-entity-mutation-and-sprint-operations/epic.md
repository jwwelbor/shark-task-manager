---
epic_key: E30
title: Entity Mutation and Sprint Operations
description: Provide editable APIs and UI flows for core Shark entities, notes, dependencies, and Sprint actions.
# size: optional but recommended. Fibonacci 1|2|3|5|8|13 or t-shirt XS|S|M|L|XL|XXL.
# Pass via the CLI: `shark create epic "<title>" --size=<S>`.
# Epics typically score 8/XL or 13/XXL — they describe multi-sprint work and exist to be decomposed.
size:
---

## 1. Problem Statement and Business Justification

Shark can already inspect epics, features, tasks, and Sprint state, but routine maintenance still depends on external CLI or direct database edits. That creates avoidable context switching for the most common operational changes: correcting metadata, adding notes, maintaining dependencies, and adjusting status. Sprint mode has a similar gap: it shows planning context but does not yet support the planning actions needed to move work forward in the same surface.

This epic closes that gap by making Shark the primary place to perform small, high-frequency work-item mutations and Sprint planning actions. The business value is operational efficiency: fewer manual edits outside the product, lower error risk from disconnected workflows, and a more useful Sprint surface for maintainers, contributors, and planners.

## 2. Goals and Success Criteria

- At least 70% of routine note, dependency, and status changes are completed in the viewer rather than via CLI or database edits within one release cycle after launch.
- Median time from opening an entity to saving a routine metadata or status update is under 60 seconds for common single-item edits.
- At least 90% of active Sprint planning sessions complete staging actions without leaving Sprint mode.
- 100% of successful mutations produce a visible history entry or equivalent audit trail.
- Fewer than 20% of routine edits require the user to leave the viewer to finish the work.

## 3. Scope: In-Scope and Out-of-Scope Boundaries

### In Scope

- Mutation APIs for Shark entity types already supported by the viewer and workflow stack.
- Editable routine fields such as title, description, priority, agent assignment, and execution order where applicable.
- Note create, update, and delete flows.
- Dependency add and remove flows with explicit validation.
- Workflow-driven status transitions that preserve validation and history.
- Sprint planning actions such as stage, remove, and ready operations.
- Viewer integration for inline editing and jump-back navigation from Sprint surfaces to entity detail views.

### Out of Scope

- Replacing existing CLI workflows.
- Arbitrary schema editing or custom user-defined fields.
- Bulk import/export tooling.
- Authentication or authorization redesign.
- Cross-project permissions or collaboration workflow changes.
- Rebuilding the viewer in a new frontend framework.
- Automatic mutation suggestions or AI-assisted edits.
- Retroactive editing of historical status records.

## 4. Constraints and Assumptions

- Status updates must remain workflow-driven; they may not become a generic unrestricted field patch.
- Dependency changes must remain explicit relations so workflow readiness and auditability stay visible.
- Mutations must be user-initiated and must not occur as a side effect of opening a view.
- The existing local-only viewer and service posture remains in place; this epic does not introduce new deployment or auth assumptions.
- The first release focuses on common maintenance paths, not advanced analytics, bulk operations, or rich-text editing.
- The supporting persona, journey, requirement, and metric documents already define the detailed downstream behavior for features in this epic.

## 5. Stakeholder Impact

- Maintainers gain a faster path for correcting data without leaving the viewer.
- Sprint planners gain explicit control over planning actions while keeping capacity and context visible.
- Contributors gain a clearer path for attaching notes, updating dependencies, and advancing work after implementation.
- Reviewers and operators gain better auditability because changes stay in the same workflow and history model as existing Shark data.
- The product itself becomes more useful for daily maintenance, which should reduce dependence on ad hoc external tooling.

## 6. High-Level Acceptance Criteria (UAT Scenarios)

- A maintainer opens an entity, edits a mutable field, saves successfully, and sees the updated value and history entry immediately in the viewer.
- A contributor adds a note to a work item, then verifies the note is attached to the entity and visible in the item history.
- A user adds and removes a dependency, and the system validates the target and preserves workflow rules without allowing invalid or duplicate relations.
- A Sprint planner stages candidate work from Sprint mode, confirms the action is recorded, and returns to the entity view without losing context.
- A user attempts an invalid mutation, receives a clear validation error, and no partial write is saved.
