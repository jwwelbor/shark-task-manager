# Requirements

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)

---

## Overview

This epic adds controlled mutation workflows to Shark's existing read-first viewer and Sprint surfaces. Requirements are grouped by entity editing, relation management, and Sprint operations.

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for the epic to be useful
- **Should Have**: Strong value, but can follow in a later feature if needed
- **Could Have**: Valuable extensions that should not block launch
- **Won't Have**: Explicitly out of scope for this epic

---

### Must Have Requirements

#### REQ-F-001 - Provide explicit mutation APIs for core entities

The system must expose mutation endpoints for the entity types already managed by Shark's viewer and workflow stack, with explicit, validated update operations rather than ad hoc direct writes.

**Must support**:
- partial field updates
- create and delete flows where the entity type already supports them
- validation against required fields and workflow rules

#### REQ-F-002 - Support note management as a first-class subresource

Notes must be addable, editable where supported, and removable through explicit endpoints or equivalent service operations. Notes must stay attached to the target entity and be visible in the viewer.

#### REQ-F-003 - Support dependency management as a first-class relation

Dependencies must be addable and removable through explicit relation operations. Dependency changes must be validated so the system can reject invalid targets, duplicates, or workflow-breaking combinations.

#### REQ-F-004 - Preserve explicit status transitions and workflow validation

Status updates must continue to use Shark's workflow rules rather than bypass them through a generic field patch. The UI and API must surface the next-status/action model where available.

#### REQ-F-005 - Expose editable fields in the viewer for routine maintenance

The viewer must allow users to update the mutable fields relevant to the selected entity type, such as title, description, priority, agent assignment, and execution order when applicable.

#### REQ-F-006 - Make Sprint planning actions actionable

Sprint mode must support explicit stage, remove, and ready-style planning actions for candidate work and keep capacity visible during those operations.

#### REQ-F-007 - Keep Sprint tree context visible while editing

The Sprint tree must continue to show active, upcoming, and archived sprint scopes alongside per-sprint buckets such as ready, in progress, blocked, and done.

#### REQ-F-008 - Preserve jump-back navigation from Sprint to entity view

Selecting an item in Sprint mode must keep a clear path back to the entity-level view so users can inspect or edit the underlying record without losing context.

---

### Should Have Requirements

#### REQ-F-009 - Preserve history on every mutation

Every successful write should create or preserve the relevant history trail so the viewer can show who changed what and when.

#### REQ-F-010 - Surface mutation feedback inline

Validation errors and success states should be visible in the viewer so users do not need to inspect raw API responses to understand what happened.

#### REQ-F-011 - Allow future extension to additional entity-specific fields

The mutation model should be extensible enough to support additional editable fields later without reworking the API shape.

---

### Non-Functional Requirements

#### REQ-NF-001 - Keep the embedded single-file viewer architecture

No new frontend framework or bundler should be introduced for this epic.

#### REQ-NF-002 - Keep the security posture local-only

The viewer and its mutation APIs must continue to respect the localhost-only posture already used by the embedded dashboard.

#### REQ-NF-003 - Avoid silent writes

Mutations must be explicit user actions. Opening Sprint mode or an entity view must not cause writes on its own.

#### REQ-NF-004 - Validate partial updates

Partial updates must reject invalid field combinations and should not silently drop unknown or malformed data.

#### REQ-NF-005 - Maintain acceptable interactive performance

The editor and Sprint planning surfaces must remain responsive within the existing viewer's performance envelope.

---

## Requirement Traceability

- REQ-F-001 through REQ-F-005 map primarily to [Journey 1](./user-journeys.md) and [Journey 2](./user-journeys.md).
- REQ-F-006 through REQ-F-008 map primarily to [Journey 3](./user-journeys.md) and [Journey 4](./user-journeys.md).
- REQ-NF-001 through REQ-NF-005 apply across all journeys.

---

## Notes On Scope

The epic is intentionally explicit about status transitions and dependency management because those are workflow-sensitive operations, not generic field edits.

*See also*: [Scope Boundaries](./scope.md)
