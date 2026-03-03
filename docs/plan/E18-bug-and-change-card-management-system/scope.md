# Scope Boundaries

**Epic**: [Bug and Change-Card Management System](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in this epic. Clear scope boundaries prevent scope creep and set expectations for future work.

---

## Out of Scope

### Explicitly Excluded Features

**1. Automated Bug Detection / Test Runner Integration**
- **Why It's Out of Scope**: While the E12 design emphasized automated bug reporting from CI/CD pipelines, E18 focuses on establishing the core entity types, workflows, and CLI commands first. Automated integration requires webhook endpoints, test runner adapters, and event parsing that are substantial features on their own.
- **Future Consideration**: A follow-on epic can add `shark bug create --reporter-type=ci_pipeline --source=github-actions` flags and webhook endpoints for automated submission.
- **Workaround**: CI/CD pipelines can use the standard CLI (`shark bug create "<title>" --severity=high --json`) in their scripts today once E18 is complete.

**2. Web UI / Dashboard Frontend**
- **Why It's Out of Scope**: Shark is a CLI-first tool. Building a web dashboard for bug triage and change-card management is a separate initiative that depends on an HTTP API layer.
- **Future Consideration**: Once the HTTP API is stable (E15 service layer), a web frontend can render bug lists, severity charts, and change-card approval queues.
- **Workaround**: All information is accessible via CLI commands with `--json` output, which can be piped to external visualization tools.

**3. Email / Slack Notifications**
- **Why It's Out of Scope**: Notification systems require integration with external services, user preference management, and message formatting -- all orthogonal to the core bug/change-card entity types.
- **Future Consideration**: A notifications epic can add alerts for bug assignment, status changes, and change-card approvals.
- **Workaround**: Users check bug and change-card status via CLI. Scripts can poll `shark bug list --status=reported --json` for monitoring.

**4. Bug / Change-Card Attachments (Images, Logs)**
- **Why It's Out of Scope**: File attachment management (upload, storage, retrieval, cleanup) is a significant subsystem. Bug details are captured in markdown files and context fields.
- **Future Consideration**: A future epic can add `shark bug attach B001 screenshot.png` for binary attachments.
- **Workaround**: Developers can reference file paths or URLs in the bug's markdown file or notes.

**5. SLA Tracking and Escalation Rules**
- **Why It's Out of Scope**: SLA configuration (e.g., "critical bugs must be triaged within 4 hours") requires timer-based automation, alerting infrastructure, and policy management. This is a separate operational concern.
- **Future Consideration**: An SLA epic can add time-based rules to the workflow system.
- **Workaround**: Teams can manually monitor bug age via `shark bug list --status=reported` sorted by creation date.

**6. Custom Fields Beyond Context**
- **Why It's Out of Scope**: The existing context field system (key-value pairs) provides sufficient metadata storage for bug and change-card attributes. A custom field definition system with types, validation, and UI rendering is a platform-level feature.
- **Future Consideration**: A custom fields epic can add `shark config add-field bug.component --type=enum --values="ui,api,database"`.
- **Workaround**: Use context fields: `shark bug context set B001 --field component --value "api"`.

---

### Edge Cases & Scenarios Not Covered

**1. Bulk Bug Import from External Tools**
- **Impact**: Low -- affects only initial migration from external bug trackers
- **Rationale**: One-time migration scripts are project-specific and do not justify CLI commands
- **Mitigation**: Users can write shell scripts that call `shark bug create` in a loop, or directly import via SQLite

**2. Bug Dependencies (Bug A blocks Bug B)**
- **Impact**: Low -- bugs rarely have formal dependencies on each other
- **Rationale**: The existing dependency system is designed for tasks. Extending it to bugs adds complexity without clear value at this stage.
- **Mitigation**: Add a note to the bug referencing related bugs: `shark bug note add B002 --type=reference "Blocked by B001"`

**3. Cross-Project Bug Tracking**
- **Impact**: Low -- Shark currently operates as a single-project tool
- **Rationale**: Multi-project support is a platform-level concern that would require database federation
- **Mitigation**: Each project has its own Shark database with independent bug tracking

**4. Bug Severity Auto-Escalation**
- **Impact**: Medium -- critical bugs that sit in `reported` status for too long should escalate
- **Rationale**: Time-based automation requires a background process or polling mechanism that does not exist in Shark's CLI-first architecture
- **Mitigation**: QA engineers manually review `shark bug list --status=reported --severity=critical` during triage

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Bugs as Task Subtypes**
- **Description**: Instead of creating new entity types, add a `type` field to tasks and allow bugs to be a task subtype with a different workflow.
- **Pros**: Reuses all existing task infrastructure (CLI, repository, service layer, tests); minimal new code
- **Cons**: Overloads the task model with type-conditional behavior; task queries become more complex; workflows become harder to reason about (which flow applies to which task type?); keys would remain in T-E##-F##-### format, losing the semantic clarity of B###
- **Decision Rationale**: Rejected because semantic clarity and workflow isolation are more valuable than code reuse. The B### and C### key formats make bugs and change-cards instantly recognizable. Separate entity types allow independent workflow evolution without risk to the task system.

**Alternative 2: Bugs as Feature-Level Items**
- **Description**: Track bugs as features with a "bug" label, using the existing feature workflow.
- **Pros**: Zero new entity types needed; features already have status flows and CLI commands
- **Cons**: Features are designed for planned work with task decomposition; bugs are reactive items that do not decompose into subtasks; the feature workflow (refinement -> development -> review) does not match bug lifecycle (triage -> fix -> verify); metrics would conflate planned and reactive work
- **Decision Rationale**: Rejected because the feature workflow does not match bug semantics, and mixing bugs into feature metrics would undermine the goal of distinguishing planned from reactive work.

**Alternative 3: Single "Item" Entity (Bugs + Change-Cards Combined)**
- **Description**: Create a single new entity type "item" with a type discriminator for bug vs. change-card.
- **Pros**: Simpler data model; fewer CLI commands to learn; single table
- **Cons**: Bugs and change-cards have fundamentally different workflows (triage/verify vs. approve/implement); combining them forces conditional workflow logic; key format (I###) loses type semantics
- **Decision Rationale**: Rejected because separate entity types with separate workflows are clearer for users and simpler to implement than a single entity with conditional workflow branching.

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| Automated Bug Reporting (CI/CD integration) | Medium | Depends on E18 (core bug entity) |
| Bug/Change-Card Web Dashboard | Low | Depends on E18 + HTTP API (E15) |
| SLA Tracking and Escalation Rules | Low | Depends on E18 (core workflows) |
| Custom Entity Fields System | Medium | Independent (platform-level) |
| Bug-to-Task and Change-to-Feature Promotion | Medium | Depends on E18 (REQ-F-019, REQ-F-020) |

---

*See also*: [Requirements](./requirements.md)
