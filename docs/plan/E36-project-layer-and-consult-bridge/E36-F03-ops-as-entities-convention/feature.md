---
feature_key: E36-F03-ops-as-entities-convention
epic_key: E36
title: Ops-as-entities convention
description: Document the convention that recurring ops (deploy/devops) are regular shark entities (tasks/change-cards under an optional Ops epic), not project activities. No new mechanism.
---

# Ops-as-entities convention

**Feature Key**: E36-F03-ops-as-entities-convention | **Size**: S

**Epic**: [Project Layer and Consult Bridge](../../epic.md) | **Design**: [plan.md §3](../../../../../dev-artifacts/2026-06-29-project-entity-design/plan.md)

---

## Goal

### Problem

The `/shark project` namespace (E36-F02) introduces a clear boundary: one-time, pre-epic, human-driven setup. Without explicit guidance, users may place recurring operational work (deploys, devops, monitoring) into the project checklist — a checkbox cannot model something you do 200 times, and a static checklist item becomes stale and meaningless for recurring work. Unguided, recurring ops become invisible, untracked, or inconsistently modeled.

### Solution

Document the convention: recurring operational work is modeled as regular shark entities — tasks or change-cards, optionally grouped under an "Ops" epic — where it keeps history, status, and queryability through existing shark infrastructure. Specify the membership rule for the `project` namespace so the boundary stays clear. No new mechanism is required.

### Impact

- Project checklists stay clean and accurate (one-time setup only).
- Recurring ops have history, status, and queryability via the entity model that already exists.
- The `project` namespace is protected from scope creep by an explicit, documented rule.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want documented guidance on how to model recurring ops (deploys, devops tasks) so that I know to track them as shark tasks or change-cards rather than adding them to the project checklist.

**Acceptance Criteria**:
- [ ] Documentation explicitly states that recurring operational work belongs as shark tasks or change-cards (optionally under an "Ops" epic), not as project activities.
- [ ] The membership rule for the `project` namespace is stated: pre-epic, one-time, human-driven, produces durable docs.
- [ ] The reasoning is explained (checklists can't model recurrence; entities provide history and status).

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Document the ops-as-entities convention
   - **Description**: Guidance — in the skill bundle (e.g., within `verbs/project.md` or a dedicated `docs/guides/` location accessible to agents) — states that recurring operational work (deploy runs, devops tasks, infrastructure changes) is tracked as shark tasks or change-cards, optionally grouped under an "Ops" epic. It does not appear on the project checklist. No new shark entity type, no new command, no new mechanism.
   - **Priority**: Must-Have

2. **REQ-F-002**: State the `project` namespace membership rule
   - **Description**: The same documentation explicitly states the membership rule that keeps the namespace from accreting unrelated commands: activities belong under `/shark project` only if they are pre-epic, one-time, human-driven, and produce durable docs.
   - **Priority**: Must-Have

3. **REQ-F-003**: Explain the rationale
   - **Description**: Documentation briefly states why: a checkbox cannot model recurrence; entities give history, status, and queryability that a static checklist entry cannot.
   - **Priority**: Should-Have

### Non-Functional Requirements

None specific to this feature beyond the epic-level constraints (no schema changes, no new mechanism).

---

## Acceptance Criteria

**Scenario 1: Convention is documented and findable**
- **Given** the E36 epic is complete
- **When** a developer or agent reads the ops-as-entities guidance
- **Then** they can clearly determine that deploy/devops work should be tracked as shark tasks or change-cards (under an "Ops" epic if grouping is desired), not as project activities
- **And** the `project` namespace membership rule (pre-epic, one-time, human-driven, produces durable docs) is explicitly stated
- **And** the rationale (checklists can't model recurrence; entities give history/status) is included

---

## Out of Scope

- **An "Ops epic" scaffold command** — convention is sufficient; no new command or generator is needed.
- **Enforcement in the CLI or skill layer** — the rule is documented, not enforced; enforcing it would require modeling ops intent, which is overcomplicated for a naming convention.
- **A new shark entity type for ops work** — existing tasks and change-cards are the right model.

---

## Success Metrics

N/A — internal tooling; see epic scope.md exclusions.

---

*Last Updated*: 2026-06-29
