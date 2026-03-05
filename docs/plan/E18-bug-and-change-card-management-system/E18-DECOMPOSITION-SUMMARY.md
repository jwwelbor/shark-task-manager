# E18 Decomposition Summary: Bug and Change-Card Management System

**Date**: 2026-03-02
**Author**: Product Manager Agent
**Epic**: E18 -- Bug and Change-Card Management System

---

## Feature Tree

```
E18: Bug and Change-Card Management System
|
+-- E18-F01: Database Schema and Workflow Engine Extension [Order: 1]
|   Foundation: tables, indexes, workflow levels, profiles, entity_notes migration
|
+-- E18-F02: Bug Entity Core (Model, Repository, Service) [Order: 2]
|   Bug model, BugRepository, BugService, triage logic, link validation, template
|
+-- E18-F03: Change-Card Entity Core (Model, Repository, Service) [Order: 3]
|   ChangeCard model, ChangeCardRepository, ChangeCardService, approval logic, template
|
+-- E18-F04: Bug CLI Commands [Order: 4]
|   shark bug create/get/list/update/delete/triage + notes/context subcommands
|
+-- E18-F05: Change-Card CLI Commands [Order: 5]
|   shark change create/get/list/update/delete/approve + notes/context subcommands
|
+-- E18-F06: Unified CLI Integration and Key Auto-Detection [Order: 6]
|   B###/C### key detection, dispatch point updates (~15 switch statements, ~12 files)
|
+-- E18-F07: Dashboard and Analytics Integration [Order: 7]
    Dashboard bug/change sections, analytics metrics, resolution time, throughput
```

---

## Dependency Graph

```
F01 (Schema + Workflow)
 |
 +---> F02 (Bug Core) -----> F04 (Bug CLI) --------+
 |                                                   |
 +---> F03 (Change Core) -> F05 (Change CLI) -------+
                                                     |
                                                     v
                                             F06 (Unified CLI)
                                                     |
                                                     v
                                             F07 (Dashboard + Analytics)
```

**Dependency details**:

| Feature | Depends On | Blocks |
|---------|-----------|--------|
| F01 | None (E16 external dependency, confirmed met) | F02, F03 |
| F02 | F01 | F04, F07 |
| F03 | F01 | F05, F07 |
| F04 | F02 | F06 |
| F05 | F03 | F06 |
| F06 | F04, F05 | F07 |
| F07 | F02, F03, F06 | None |

**Parallelization opportunities**:
- F02 and F03 can be developed in parallel (both depend only on F01)
- F04 and F05 can be developed in parallel (they depend on F02/F03 respectively)

**Acyclicity verification**: The dependency graph is a directed acyclic graph (DAG). No circular dependencies exist. The critical path is: F01 -> F02 -> F04 -> F06 -> F07.

---

## Requirements Traceability Matrix

Every epic requirement is mapped to at least one feature. No requirement is left unassigned.

### Must Have Requirements

| Requirement | Description | Feature(s) | Priority |
|-------------|------------|------------|----------|
| REQ-F-001 | Bug Entity Creation | F01 (table), F02 (service), F04 (CLI) | Must |
| REQ-F-002 | Bug Severity Tracking | F01 (column), F02 (filtering), F04 (--severity flag) | Must |
| REQ-F-003 | Bug Entity Linking | F01 (columns), F02 (validation), F04 (--link flag) | Must |
| REQ-F-004 | Bug Status Workflow | F01 (workflow level), F02 (service transitions), F06 (unified dispatch) | Must |
| REQ-F-005 | Bug Triage Command | F02 (TriageBug service), F04 (shark bug triage CLI) | Must |
| REQ-F-006 | Bug CRUD Commands | F02 (service CRUD), F04 (CLI commands) | Must |
| REQ-F-007 | Change-Card Entity Creation | F01 (table), F03 (service), F05 (CLI) | Must |
| REQ-F-008 | Change-Card Entity Linking | F01 (columns), F03 (validation), F05 (--link flag) | Must |
| REQ-F-009 | Change-Card Status Workflow | F01 (workflow level), F03 (service transitions), F06 (unified dispatch) | Must |
| REQ-F-010 | Change-Card Approval Command | F03 (ApproveChangeCard service), F05 (shark change approve CLI) | Must |
| REQ-F-011 | Change-Card CRUD Commands | F03 (service CRUD), F05 (CLI commands) | Must |
| REQ-F-012 | Core Command Auto-Detection | F06 (key detection, dispatch updates) | Must |
| REQ-F-013 | Dashboard Integration | F07 (bug/change-card dashboard sections) | Must |
| REQ-F-014 | Analytics Integration | F07 (bug/change-card metrics) | Must |

### Should Have Requirements

| Requirement | Description | Feature(s) | Priority |
|-------------|------------|------------|----------|
| REQ-F-015 | Bug Notes and Context | F01 (entity_notes migration), F02 (service support), F04 (CLI commands) | Should |
| REQ-F-016 | Bug Markdown File Template | F02 (template), F04 (shark view B001) | Should |
| REQ-F-017 | Change-Card Notes and Context | F01 (entity_notes migration), F03 (service support), F05 (CLI commands) | Should |
| REQ-F-018 | Bug List Filtering by Linked Entity | F02 (repository query), F04 (--link flag on list), F05 (--link flag on list) | Should |

### Could Have Requirements (Deferred)

| Requirement | Description | Status |
|-------------|------------|--------|
| REQ-F-019 | Bug-to-Task Promotion | Deferred to follow-on epic |
| REQ-F-020 | Change-Card-to-Feature Promotion | Deferred to follow-on epic |
| REQ-F-021 | Bug Duplicate Detection Hint | Deferred to follow-on epic |

### Non-Functional Requirements

| Requirement | Description | Feature(s) |
|-------------|------------|------------|
| REQ-NF-001 | Bug/Change Creation Speed (< 500ms local) | F02, F03 (efficient queries), F04, F05 (minimal overhead) |
| REQ-NF-002 | List Command Performance (< 1s for 1000 entities) | F02, F03 (indexed queries), F07 (efficient aggregates) |
| REQ-NF-003 | Atomic Operations | F02, F03 (fileops atomic write pattern) |
| REQ-NF-004 | Key Uniqueness | F01 (UNIQUE constraints, auto-increment) |
| REQ-NF-005 | CLI Pattern Consistency | F04, F05, F06 (follow established patterns) |
| REQ-NF-006 | Workflow Profile Integration | F01 (workflow profiles include bug/change defaults) |

---

## Sprint Sizing Estimates

| Feature | Complexity | Sprint Estimate | Notes |
|---------|-----------|----------------|-------|
| F01: Database Schema + Workflow | S-M | 1 sprint | Foundation; all pattern-based |
| F02: Bug Entity Core | M | 1-2 sprints | Most complex (triage, link validation, 3 layers + tests) |
| F03: Change-Card Entity Core | S-M | 1 sprint | Simpler than F02; parallel with F02 |
| F04: Bug CLI Commands | M | 1 sprint | Many subcommands but thin wrappers |
| F05: Change-Card CLI Commands | S-M | 1 sprint | Simpler than F04; parallel with F04 |
| F06: Unified CLI Integration | M | 1 sprint | Cross-cutting; highest risk area |
| F07: Dashboard + Analytics | M | 1-2 sprints | New rendering sections + metrics calculations |

**Total estimate**: 7-9 sprints of sequential work, or 5-6 sprints with parallelization (F02/F03 parallel, F04/F05 parallel).

---

## Phased Delivery Alignment

The feature decomposition aligns with the phased approach recommended by the research report and feasibility reviews:

| Phase | Features | Delivers |
|-------|----------|----------|
| Phase 1: Core Entities | F01, F02, F03 | Data model + workflows + service layer for both bugs and change-cards |
| Phase 2: CLI Access | F04, F05, F06 | Full CLI commands + unified command integration |
| Phase 3: Visibility | F07 | Dashboard + analytics with all metrics |

Each phase delivers independently usable functionality:
- After Phase 1: Service layer is testable; HTTP API could use services directly.
- After Phase 2: Users can manage bugs and change-cards through CLI.
- After Phase 3: Full visibility into bug/change-card data in dashboards and analytics.

---

## Scope Overlap Verification

Each feature has a distinct, non-overlapping scope:

| Feature | Scope | Does NOT include |
|---------|-------|-----------------|
| F01 | Database tables, workflow engine, migrations | Models, repos, services, CLI |
| F02 | Bug model + repo + service + template | CLI commands, dashboard |
| F03 | Change-card model + repo + service + template | CLI commands, dashboard |
| F04 | Bug CLI commands only | Change-card commands, unified dispatch |
| F05 | Change-card CLI commands only | Bug commands, unified dispatch |
| F06 | Key detection + dispatch updates only | Entity-specific commands, dashboard |
| F07 | Dashboard + analytics only | CRUD operations, workflow logic |

No two features share implementation responsibility for any component.

---

## Exit Gate Verification

| Gate Criterion | Status | Evidence |
|----------------|--------|----------|
| Every epic requirement traces to a feature | PASS | All 14 Must Have, 4 Should Have requirements mapped. 3 Could Have deferred with documentation. All 6 Non-Functional requirements mapped. See traceability matrix above. |
| No scope overlap between features | PASS | Each feature has distinct component ownership. See overlap verification above. |
| Acyclic dependencies | PASS | Dependency graph is a DAG with no cycles. Critical path: F01 -> F02 -> F04 -> F06 -> F07. |
| Each feature sized for 1-3 sprints | PASS | All features estimated at 1-2 sprints. No feature exceeds 2 sprints. |

---

*Decomposition complete. Ready for feature-level refinement and task creation.*
