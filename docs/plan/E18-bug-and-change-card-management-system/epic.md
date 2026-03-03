---
epic_key: E18
title: Bug and Change-Card Management System
description: Add first-class bug and change-card (enhancement card) entity types to Shark with dedicated workflows, CLI commands, and reporting
---

# Bug and Change-Card Management System

**Epic Key**: E18

---

## Goal

### Problem

Shark Task Manager currently has no way to track bugs or small enhancement requests. Bugs discovered during development, testing, or production monitoring get lost or are tracked in external tools, breaking the single-source-of-truth principle that makes Shark effective. Small improvements that do not warrant a full epic/feature decomposition have no home -- they either get shoehorned into the task system (losing semantic clarity) or go untracked entirely. There is no way to distinguish bugs from planned features in dashboards or reports, making it impossible to measure defect rates, triage efficiency, or the balance between planned work and reactive work. Teams need different workflows for bugs (triage, fix, verify) versus planned features, but everything is currently forced through the same epic/feature/task pipeline.

### Solution

Introduce **bugs** and **change-cards** as new first-class entity types in Shark. Each type is a standalone item that can optionally link to existing epics, features, or tasks but does not require parent entities. Each type has its own dedicated workflow (status flow) tailored to its lifecycle. The database schema reuses the proven patterns from the existing entity system for consistency. CLI commands follow existing patterns (`shark bug create`, `shark change create`) and integrate with the unified `shark status`, `shark get`, and `shark search` commands. Reporting and analytics surfaces include bug/change-card counts, resolution metrics, and type-based filtering.

### Impact

- All project work items tracked in a single system -- bugs, enhancements, features, and tasks -- eliminating external tool dependency
- Faster bug triage and resolution through a dedicated bug workflow with severity tracking and verification gates
- Clear visibility into the balance between planned work (features/tasks) and reactive work (bugs/change-cards) via reporting
- A lightweight path for small improvements that do not need full epic/feature decomposition overhead

---

## Business Value

**Rating**: High

Bugs and change requests are fundamental to every development workflow. Without native support, teams must use external tools or misuse the task system, leading to data fragmentation and lost context. This epic closes a critical gap in Shark's completeness as a project management tool. It directly improves developer productivity by eliminating context switching between tools and enables data-driven decisions about quality investment through integrated analytics.

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - Developer, QA engineer, and product owner profiles
- **[User Journeys](./user-journeys.md)** - Bug lifecycle, change-card lifecycle, and cross-entity workflows
- **[Requirements](./requirements.md)** - Functional and non-functional requirements catalog
- **[Success Metrics](./success-metrics.md)** - KPIs for adoption, resolution efficiency, and coverage
- **[Scope Boundaries](./scope.md)** - Explicit exclusions and future considerations

---

## Quick Reference

**Primary Users**: Developers, QA engineers, Product owners

**Key Features**:
- Bug entity type with dedicated triage/fix/verify workflow and severity tracking
- Change-card entity type with lightweight propose/approve/implement workflow
- CLI commands: `shark bug create/list/get/triage`, `shark change create/list/get/approve`
- Optional linking to existing epics, features, and tasks
- Dashboard and analytics integration with type-based filtering

**Success Criteria**:
- Bugs trackable end-to-end within Shark with zero reliance on external tools
- Change-cards provide a lightweight path for small enhancements with under 60 seconds to create
- Reporting distinguishes bugs, change-cards, and features with type-based filtering

**Relationship to E12**: This epic supersedes the earlier E12 Bug Tracker System design. E18 expands scope to include change-cards, simplifies the data model approach, and aligns with the current architecture patterns established in E15-E17.

---

## Entity Design Summary

### Bug (B###)

A **bug** is a defect report -- something that was working (or expected to work) but does not. Bugs are standalone items with an optional link to the epic, feature, or task where the defect was found.

**Dedicated Workflow:**
```
reported -> triaged -> in_fix -> in_verification -> resolved
                                                 -> wont_fix
            -> duplicate
```

**Key Characteristics:**
- Standalone entity (not nested under features)
- Optional link to epic, feature, or task (where the bug was found)
- Severity tracking: critical, high, medium, low
- Key format: `B###` (e.g., `B001`, `B042`)
- Markdown file stores reproduction steps, environment info, stack traces

### Change-Card (C###)

A **change-card** is a small enhancement request -- an improvement that does not warrant a full feature/epic decomposition but needs tracking and workflow.

**Dedicated Workflow:**
```
proposed -> approved -> in_progress -> completed
         -> declined
```

**Key Characteristics:**
- Standalone entity (not nested under features)
- Optional link to epic or feature (what area it enhances)
- Lighter weight than feature -- no decomposition into tasks
- Key format: `C###` (e.g., `C001`, `C015`)

### CLI Commands

```bash
# Bug commands
shark bug create "Login fails on Safari" --severity=high --link=E07-F01
shark bug list [--severity=critical] [--status=reported]
shark bug get B001
shark bug triage B001 --severity=medium --assign=developer
shark status advance B001

# Change-card commands
shark change create "Add dark mode toggle to settings" --link=E07
shark change list [--status=proposed]
shark change get C001
shark change approve C001
shark status advance C001
```

---

## Open Questions & Assumptions

No open questions -- all epic-level decisions are resolved.

**Resolved Assumptions:**
1. Bugs and change-cards are standalone entities, not subtypes of tasks. This is intentional to keep workflows clean and avoid overloading the task model.
2. Key formats `B###` and `C###` are globally unique (not scoped to epics). This matches the standalone nature of these entities.
3. The `shark get`, `shark search`, and `shark status` commands will be extended to auto-detect bugs and change-cards by key prefix, matching the existing pattern for E##/F##/T-E## detection.
4. E12 Bug Tracker System design documents serve as prior art but are not binding. E18 may diverge where the current architecture (post E15-E17) warrants different approaches.

---

*Last Updated*: 2026-03-02
