# E18 Epic Research Report: Bug and Change-Card Management System

**Date**: 2026-03-02
**Researcher**: Researcher Agent
**Status**: Complete

---

## Executive Summary

E18 proposes adding two new first-class entity types -- bugs (B###) and change-cards (C###) -- to Shark Task Manager. This research finds the epic is **technically feasible** with no showstoppers, leveraging proven architectural patterns already established in the codebase. The design correctly supersedes E12 by simplifying the approach and aligning with post-E15/E16/E17 architecture. Key risks center on the scope of cross-cutting changes required (key detection, workflow engine, dashboard, analytics, search) rather than on any single component. The recommendation is to proceed, with careful attention to dependency on E16 (multi-level workflow) for workflow profile integration and ensuring the new entity types do not fragment the codebase into entity-specific silos.

---

## Section 1: Market and Competitive Landscape

### Relevance to E18 Requirements

The E18 PRD positions Shark as a "single source of truth" for all project work items. This section assesses whether the proposed bug and change-card designs align with industry patterns and user expectations.

### Competitive Analysis

| Tool | Bug Tracking | Change Request / Enhancement | CLI-First | Local-First | Integrated Workflow |
|------|-------------|------------------------------|-----------|-------------|---------------------|
| **Jira** | Full (Issues + Bug type) | Full (Story/Enhancement types) | No (API, some CLI wrappers) | No (cloud) | Yes (configurable) |
| **GitHub Issues** | Labels-based (no dedicated type) | Labels-based | Yes (`gh issue`) | No (cloud) | Limited (Projects) |
| **Linear** | Issue types with triage | Issue types | Yes (`linear` CLI) | No (cloud) | Yes (cycles, states) |
| **Bugzilla** | Dedicated bug type | Enhancement type | Limited | Self-hosted | Yes (fixed workflow) |
| **ClickUp** | Bug task type | Task subtypes | No | No | Yes (custom statuses) |
| **Plane** | Issue types | Issue types | Limited | Self-hosted option | Yes |
| **Shark (E18)** | Dedicated bug entity (B###) | Dedicated change-card (C###) | Yes (primary) | Yes (SQLite) | Yes (configurable per entity type) |

### Key Market Insights

1. **Dedicated entity types are the industry norm**. Jira, Linear, and Bugzilla all distinguish bugs from features/enhancements at the entity level, not just via labels. E18's decision to create separate entity types rather than adding a `type` field to tasks aligns with industry best practices.

2. **CLI-first bug tracking is a niche**. Most tools are web-first. GitHub Issues (`gh issue create`) and Linear are the closest competitors for CLI-native workflows. Shark's CLI-first approach with `--json` output for AI agents is a genuine differentiator.

3. **Severity tracking is universal for bugs**. Every bug tracker supports severity/priority classification. E18's four-level severity (critical/high/medium/low) is standard.

4. **Dedicated triage workflows are common**. Jira's triage board, Bugzilla's UNCONFIRMED->NEW transition, and Linear's triage queue all mirror E18's `reported -> triaged` workflow step.

5. **Change request / enhancement request tracking is underdeveloped in CLI tools**. Most CLI tools do not distinguish enhancement requests from regular issues. E18's change-card concept fills a real gap.

6. **Linking bugs to features is standard**. All major tools support relating bugs to features or epics. E18's `--link` flag matches this expectation.

### Market Assessment

E18's design is well-aligned with market expectations. The combination of CLI-first, local-first, and dedicated entity types with configurable workflows is unique in the market. No direct competitor offers all of these together.

---

## Section 2: Feasibility Assessment

### Assessment by Requirement Area

#### REQ-F-001 through REQ-F-003: Bug Data Model -- FEASIBLE (Low Risk)

**Evidence from codebase:**
- The `ideas` table (`/home/jwwel/projects/shark-task-manager/internal/db/db.go`, line 337) demonstrates that Shark can add standalone entity tables with unique key formats, auto-increment, and optional linking fields.
- The `Idea` model (`/home/jwwel/projects/shark-task-manager/internal/models/idea.go`) shows the pattern for standalone entities with key validation, status enums, and optional fields.
- Creating a `bugs` table follows this proven pattern. The B### key format requires a new auto-increment sequence, which is straightforward in SQLite (`INTEGER PRIMARY KEY AUTOINCREMENT`).
- The optional `--link` field maps to a nullable `linked_entity_type` + `linked_entity_key` column pair, similar to the idea's `converted_to_type`/`converted_to_key` pattern.

**Effort estimate**: Small. Direct reuse of existing patterns.

#### REQ-F-004 through REQ-F-005: Bug Workflow -- FEASIBLE (Medium Risk)

**Evidence from codebase:**
- The workflow engine (`/home/jwwel/projects/shark-task-manager/internal/config/workflow_multilevel.go`) already supports per-level workflow definitions via `GetWorkflowForLevel()`. Currently supports levels: `task`, `epic`, `feature`.
- Adding `bug` and `change` levels to the multi-level workflow system is architecturally straightforward but requires:
  1. Extending `GetWorkflowForLevel()` to return bug/change-card workflows
  2. Adding `bug_workflow` and `change_workflow` sections to `.sharkconfig.json`
  3. Extending the `workflow.Service.ForLevel()` method to accept `LevelBug` and `LevelChange` constants

**Dependency**: This integrates with E16 (Multi-Level Workflow System). E16 established the pattern for per-level workflows. E18 follows the same pattern for two additional levels.

**Effort estimate**: Medium. Requires workflow engine extension but pattern is established.

#### REQ-F-006, REQ-F-011: Bug and Change-Card CRUD -- FEASIBLE (Low Risk)

**Evidence from codebase:**
- Full CRUD pattern is well-established across epic, feature, task, and idea entities.
- Each entity has: model (`/home/jwwel/projects/shark-task-manager/internal/models/`), repository (`/home/jwwel/projects/shark-task-manager/internal/repository/`), service (`/home/jwwel/projects/shark-task-manager/internal/services/`), and CLI commands (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/`).
- New files needed: `bug.go` (model), `bug_repository.go` (repo), `bug_service.go` (service), `bug.go` (CLI commands), and equivalents for change-cards.
- The `fileops` package (`/home/jwwel/projects/shark-task-manager/internal/fileops/writer.go`) supports entity-agnostic file writing, so bug/change-card markdown generation follows existing patterns.

**Effort estimate**: Medium. Significant volume of code but entirely pattern-based.

#### REQ-F-007 through REQ-F-010: Change-Card Data Model and Workflow -- FEASIBLE (Low Risk)

Same feasibility assessment as bugs. The change-card is a simpler entity (fewer fields, shorter workflow) and follows identical patterns.

**Effort estimate**: Small to Medium. Simpler than bugs.

#### REQ-F-012: Core Command Auto-Detection -- FEASIBLE (Medium Risk)

**Evidence from codebase:**
- The `KeyService` (`/home/jwwel/projects/shark-task-manager/internal/keys/service.go`, line 169) has a `DetectEntityType()` method that uses regex patterns to identify entity type from key format (E##, E##-F##, T-E##-F##-###).
- A secondary `DetectEntityType()` exists in CLI helpers (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/helpers.go`, line 568).
- Adding B### and C### detection requires:
  1. New regex patterns in the `KeyService`
  2. New `EntityTypeBug` and `EntityTypeChange` constants
  3. Updates to all dispatch functions that switch on entity type: `update_dispatch.go`, `delete_dispatch.go`, `status_group.go`, `context.go`, `helpers.go`
- The dispatch functions in `status_group.go`, `delete_dispatch.go`, and `update_dispatch.go` all use entity type switches that must be extended.

**Risk**: Cross-cutting change. Every entity-type switch statement in the codebase must be updated. Missing any one creates a silent failure where a command silently rejects B### or C### keys.

**Mitigation**: Grep for all `EntityType` switch statements and create a checklist. Add a compile-time exhaustive check if Go supports it for the EntityType constants.

**Effort estimate**: Medium. Many files to touch, but each change is small.

#### REQ-F-013 through REQ-F-014: Dashboard and Analytics -- FEASIBLE (Medium Risk)

**Evidence from codebase:**
- The `status` package (`/home/jwwel/projects/shark-task-manager/internal/status/calculation_service.go`) provides dashboard data for features and epics.
- The analytics service (`/home/jwwel/projects/shark-task-manager/internal/services/epic_analytics_service.go`) computes metrics.
- Adding bug/change-card sections requires new query methods and dashboard rendering sections.
- The `shark status` dashboard command and `shark analytics` command both need new output sections.

**Risk**: Dashboard complexity increases. The project dashboard must now show 5 entity types (epics, features, tasks, bugs, change-cards) without becoming overwhelming.

**Effort estimate**: Medium.

#### REQ-F-015 through REQ-F-018: Notes, Context, Filtering -- FEASIBLE (Low Risk)

**Evidence from codebase:**
- The `EntityNoteRepository` (`/home/jwwel/projects/shark-task-manager/internal/repository/entity_note_repository.go`) already supports notes for multiple entity types via an `entity_type` field.
- The `ContextService` (`/home/jwwel/projects/shark-task-manager/internal/services/context_service.go`) is entity-type-aware.
- Bug and change-card notes/context slot directly into these existing systems.

**Effort estimate**: Small. Existing infrastructure supports this.

#### REQ-NF-001 through REQ-NF-006: Non-Functional Requirements -- FEASIBLE (Low Risk)

- **Performance**: SQLite CRUD for new tables will match existing entity performance. No new performance patterns required.
- **Atomic operations**: The `fileops` package handles atomic file+DB operations. Bugs and change-cards can use the same pattern.
- **Key uniqueness**: SQLite UNIQUE constraint on the key column, matching existing pattern.
- **CLI consistency**: Following established CLI patterns guarantees consistency.
- **Workflow profile integration**: Depends on E16 multi-level workflow system being complete or at least having the `ForLevel()` infrastructure (which it does).

### Feasibility Summary

| Requirement Area | Feasibility | Risk | Key Dependency |
|-----------------|-------------|------|----------------|
| Bug Data Model | HIGH | Low | None |
| Bug Workflow | HIGH | Medium | E16 (workflow engine) |
| Bug CRUD CLI | HIGH | Low | None |
| Change-Card Data Model | HIGH | Low | None |
| Change-Card Workflow | HIGH | Medium | E16 (workflow engine) |
| Change-Card CRUD CLI | HIGH | Low | None |
| Auto-Detection (B###/C###) | HIGH | Medium | Key service extension |
| Dashboard Integration | HIGH | Medium | Status/analytics services |
| Notes and Context | HIGH | Low | EntityNote system |
| Non-Functional | HIGH | Low | None |

**No feasibility blockers identified.**

---

## Section 3: System-Wide Impact -- Epic Interactions

### E12: Bug Tracker System (SUPERSEDED)

- **Interaction**: E18 explicitly supersedes E12. The E12 design was created on 2026-01-04 with a more complex data model (28+ fields) and emphasis on CI/CD automated reporting.
- **Impact**: E18 simplifies the approach by using context fields for optional metadata rather than dedicated columns. E12 design documents at `dev-artifacts/2026-01-04-bug-tracker-design/` serve as prior art but are not binding.
- **Action needed**: Mark E12 as superseded/cancelled once E18 work begins.

### E08: Idea Capture and Conversion System (COMPLEMENTARY)

- **Interaction**: E08 established the pattern for standalone entities with promotion/conversion (idea -> epic/feature/task). E18's Could Have requirements (REQ-F-019, REQ-F-020) propose similar promotion paths (bug -> task, change-card -> feature).
- **Impact**: The promotion pattern from E08 (`converted_to_type`, `converted_to_key`, `converted_at` fields on the idea model) can be directly reused for bug and change-card promotion.
- **Risk**: None. Complementary pattern reuse.

### E15: Service Layer Architecture Refactoring (FOUNDATION)

- **Interaction**: E15 established the service layer pattern (service -> repository -> DB) that E18 must follow. Bug and change-card services must follow the `NewXxxService(repo, workflowSvc, ...)` constructor pattern.
- **Impact**: E18 adds 2 new services (`BugService`, `ChangeCardService`), 2 new repositories, 2 new models, and corresponding CLI commands. All must follow E15 patterns.
- **Risk**: Low. E15 patterns are mature and well-documented in `.claude/rules/services/`.

### E16: Multi-Level Workflow System (DEPENDENCY)

- **Interaction**: E16 extended the workflow engine from task-only to epic/feature/task. E18 needs to extend it further to bug and change-card levels.
- **Impact**: E18 requires the `ForLevel()` infrastructure that E16 built. The `.sharkconfig.json` must support `bug_workflow` and `change_workflow` sections alongside the existing `epic_workflow`, `feature_workflow`, and task-level sections.
- **Risk**: Medium. If E16 is not fully complete, E18 may need to finalize the multi-level workflow infrastructure as part of its own scope.
- **Current state**: E16's `GetWorkflowForLevel()` function exists and works for task/epic/feature levels. Adding bug/change levels is additive.

### E17: CLI Simplification for AI Agents (EXTENSION)

- **Interaction**: E17 established the unified CLI patterns (`shark get`, `shark status`, `shark search`) with entity auto-detection. E18 must extend these patterns to bugs and change-cards.
- **Impact**: Every unified command that dispatches by entity type must add bug and change-card cases. The `--type` filter flag for search must add `bug` and `change` values.
- **Risk**: Medium (cross-cutting). Same auto-detection risk noted in REQ-F-012 feasibility.

### E11: Configurable Status Workflow System (FOUNDATION)

- **Interaction**: E11 delivered the configurable workflow engine for tasks. E16 extended it to epic/feature. E18 follows the same extension pattern for bugs and change-cards.
- **Impact**: Bug and change-card workflows are defined in `.sharkconfig.json` using the same `status_flow` + `status_metadata` structure.
- **Risk**: None. Pure extension of established patterns.

### Summary of Interactions

| Epic | Relationship | Action Required |
|------|-------------|-----------------|
| E12 | Superseded by E18 | Cancel/archive E12 |
| E08 | Complementary (promotion pattern) | Reuse conversion pattern |
| E15 | Foundation (service layer) | Follow E15 patterns |
| E16 | Dependency (workflow engine) | Extend ForLevel() |
| E17 | Extension (unified CLI) | Add B###/C### to dispatch |
| E11 | Foundation (workflow config) | Follow config patterns |

---

## Section 4: Existing Capability Overlap with Defined Scope

### What Already Exists and Can Be Reused

| Capability | Existing Implementation | Reuse for E18 |
|-----------|------------------------|---------------|
| **Entity model pattern** | `models/epic.go`, `models/feature.go`, `models/task.go`, `models/idea.go` | Template for `models/bug.go`, `models/change_card.go` |
| **Repository CRUD pattern** | `repository/epic_repository.go`, `repository/idea_repository.go` | Template for bug/change-card repositories |
| **Service layer pattern** | `services/task_service.go`, `services/idea_service.go` | Template for bug/change-card services |
| **CLI command pattern** | `cli/commands/task.go`, `cli/commands/idea.go` | Template for `shark bug` and `shark change` commands |
| **Entity notes system** | `repository/entity_note_repository.go` with `entity_type` field | Directly supports bug/change-card notes (add "bug"/"change" as entity_type values) |
| **Context service** | `services/context_service.go` | Supports bugs/changes via entity_type switch |
| **Resume service** | `services/resume_service.go` | Extensible to bugs/changes |
| **Key detection** | `keys/service.go` with `DetectEntityType()` | Extend with B###/C### patterns |
| **File operations** | `fileops/writer.go` with entity-agnostic `WriteEntityFile()` | Direct reuse for bug/change markdown files |
| **Workflow engine** | `config/workflow_multilevel.go` with `GetWorkflowForLevel()` | Extend with bug/change levels |
| **Search** | `repository/search_repository.go` | Extend to search bugs/changes |
| **Status history/audit** | `repository/task_history_repository.go` | Pattern reuse; may need entity_type column or separate tables |
| **Slug generation** | Auto-slug from title (slug architecture) | Direct reuse for B###-slug and C###-slug keys |
| **Markdown templates** | `templates/` package | Add bug and change-card templates |
| **Database migrations** | Auto-migration system in `db/db.go` | Add bug/change tables via migration |

### What Must Be Built New

| Component | Reason | Complexity |
|-----------|--------|------------|
| `bugs` database table | New entity type | Small |
| `change_cards` database table | New entity type | Small |
| `models/bug.go` | New model with severity, linked entity fields | Small |
| `models/change_card.go` | New model with linked entity fields | Small |
| `repository/bug_repository.go` | CRUD + severity/link filtering | Medium |
| `repository/change_card_repository.go` | CRUD + link filtering | Medium |
| `services/bug_service.go` | Triage command logic, workflow integration | Medium |
| `services/change_card_service.go` | Approval command logic, workflow integration | Medium |
| `cli/commands/bug.go` | `shark bug create/get/list/update/delete/triage` | Medium |
| `cli/commands/change.go` | `shark change create/get/list/update/delete/approve` | Medium |
| Bug workflow definition in `.sharkconfig.json` | New workflow level | Small |
| Change-card workflow definition in `.sharkconfig.json` | New workflow level | Small |
| Dashboard sections for bugs/changes | New rendering in status command | Medium |
| Analytics for bugs/changes | New metrics computation | Medium |
| Bug markdown template | Reproduction steps, environment, etc. | Small |
| Change-card markdown template | Description, justification | Small |

### Overlap Assessment

Approximately 60% of E18 is pattern reuse from existing entity implementations. The remaining 40% is new code for entity-specific logic (triage, approval, severity filtering, dashboard sections, analytics metrics). This is a favorable ratio that reduces implementation risk.

---

## Section 5: Risk Assessment

### Risk 1: Cross-Cutting Entity Type Changes

- **Probability**: HIGH (certain to occur)
- **Impact**: MEDIUM
- **Description**: Adding new entity types requires updating every switch/dispatch point in the codebase that operates on entity type. The `DetectEntityType()` function in `keys/service.go`, the dispatch functions in `update_dispatch.go`, `delete_dispatch.go`, `status_group.go`, `context.go`, and `helpers.go` all must be updated. Missing any dispatch point creates a silent failure.
- **Mapped Requirements**: REQ-F-012 (auto-detection), REQ-F-006 (bug CRUD), REQ-F-011 (change-card CRUD)
- **Mitigation**: Before implementation, create a comprehensive grep-based inventory of all `EntityType` switch/dispatch points. Consider adding a linter or compile-time check that flags incomplete entity type switches. Use the existing test suite to verify all dispatch paths.

### Risk 2: Workflow Engine Extension Complexity

- **Probability**: MEDIUM
- **Impact**: MEDIUM
- **Description**: Adding two new workflow levels (bug, change-card) to the multi-level workflow engine doubles the number of workflow configurations in `.sharkconfig.json`. The workflow validation, `show-actions`, `list`, and `validate` commands all must handle 5 levels (epic, feature, task, bug, change-card). Configuration file complexity increases.
- **Mapped Requirements**: REQ-F-004 (bug workflow), REQ-F-009 (change-card workflow), REQ-NF-006 (workflow profile integration)
- **Mitigation**: The `ForLevel()` abstraction already handles multi-level dispatch cleanly. Adding two more levels is linear complexity, not exponential. Validate workflows independently per level (no cross-level references needed for bugs/changes).

### Risk 3: Dashboard Information Overload

- **Probability**: MEDIUM
- **Impact**: LOW
- **Description**: The `shark status` dashboard currently shows epics, features, and tasks. Adding bug counts, severity breakdowns, and change-card counts increases information density. Users may find the dashboard overwhelming.
- **Mapped Requirements**: REQ-F-013 (dashboard integration)
- **Mitigation**: Use collapsible sections or conditional display (only show bug/change-card sections when entities exist). Keep bug/change-card sections concise (counts by status, not full lists). This is a UX design decision, not a technical blocker.

### Risk 4: E16 Dependency Incomplete

- **Probability**: LOW
- **Impact**: MEDIUM
- **Description**: E18's workflow integration depends on E16's multi-level workflow infrastructure. If E16 is not fully complete, E18 may need to build or finalize missing workflow engine pieces.
- **Mapped Requirements**: REQ-F-004 (bug workflow), REQ-F-009 (change-card workflow)
- **Mitigation**: The core `ForLevel()` infrastructure already exists and works. E18 only needs to register new levels, not redesign the engine. If E16 has gaps, they are incremental and can be addressed as part of E18's scope.

### Risk 5: Key Format Collision (Future)

- **Probability**: LOW
- **Impact**: LOW
- **Description**: The B### and C### key formats use single-letter prefixes. Future entity types may need prefixes that conflict (e.g., "C" for "Change" vs. a future "Component" entity). The current design uses B and C which are clear, but the single-letter prefix space is limited (26 letters).
- **Mapped Requirements**: REQ-NF-004 (key uniqueness)
- **Mitigation**: Document the B### and C### prefix allocation in the architecture docs. If future entity types need similar formats, use multi-letter prefixes (e.g., "CP###" for components) or a different scheme. This is a future concern, not a current blocker.

### Risk 6: Test Suite Expansion

- **Probability**: HIGH (certain)
- **Impact**: LOW
- **Description**: Two new entity types mean two new sets of tests across model, repository, service, and CLI layers. The test suite will grow significantly. Repository tests require database cleanup patterns. Service tests require new mock definitions.
- **Mapped Requirements**: All functional requirements
- **Mitigation**: Follow established test patterns. Reuse `MockWorkflowService` and mock patterns from existing service tests. Use table-driven tests for CRUD operations. The testing architecture in `.claude/rules/testing/` provides clear guidance.

### Risk Summary Matrix

| Risk | Probability | Impact | Overall | Mitigation Available |
|------|-------------|--------|---------|---------------------|
| Cross-cutting entity type changes | HIGH | MEDIUM | HIGH | Yes (grep inventory, exhaustive tests) |
| Workflow engine extension | MEDIUM | MEDIUM | MEDIUM | Yes (linear extension of existing pattern) |
| Dashboard overload | MEDIUM | LOW | LOW | Yes (conditional display) |
| E16 dependency incomplete | LOW | MEDIUM | LOW | Yes (core infrastructure exists) |
| Key format collision (future) | LOW | LOW | LOW | Yes (document prefix allocation) |
| Test suite expansion | HIGH | LOW | LOW | Yes (follow patterns) |

**No risks are rated as blockers.**

---

## Section 6: Recommendations

### Recommendation 1: Proceed with E18 as Designed

**Rationale**: The feasibility assessment shows no technical blockers. The design aligns with industry best practices, leverages proven codebase patterns, and addresses a genuine gap in Shark's capabilities. The decision to create separate entity types (rather than task subtypes) is correct given the need for independent workflows and semantic clarity.

### Recommendation 2: Phase Implementation by Priority

**Rationale**: The PRD's MoSCoW prioritization is well-structured. Implement in this order:

1. **Phase 1** (Must Have): Bug entity (data model + workflow + CRUD) and change-card entity (data model + workflow + CRUD). This delivers the core value proposition.
2. **Phase 2** (Must Have): Unified CLI integration (auto-detection for B###/C###), dashboard integration, analytics integration. This makes bugs and change-cards first-class citizens.
3. **Phase 3** (Should Have): Notes, context, markdown templates, linked-entity filtering. This adds depth to the entity types.
4. **Phase 4** (Could Have): Promotion (bug->task, change->feature), duplicate detection hints. These are valuable but deferrable.

### Recommendation 3: Create Entity Type Registry Pattern

**Rationale**: With 5 entity types (epic, feature, task, bug, change-card) plus ideas, the number of dispatch/switch points in the codebase is becoming a maintenance burden. Consider introducing an entity type registry pattern that:
- Registers entity types at startup with their key patterns, services, and repositories
- Provides a central dispatch function that eliminates scattered switch statements
- Makes adding future entity types a single registration call rather than N file modifications

This is not required for E18 but would significantly reduce the cross-cutting change risk identified in Risk 1.

### Recommendation 4: Cancel E12

**Rationale**: E18 explicitly supersedes E12. The E12 design documents should be archived and the epic marked as cancelled to avoid confusion. E12's design documents at `dev-artifacts/2026-01-04-bug-tracker-design/` remain valuable as reference material.

### Recommendation 5: Validate Workflow Engine Extension Early

**Rationale**: The workflow engine extension (adding `bug` and `change` levels to `ForLevel()`) is a foundational dependency for almost all E18 features. Implement and test this first to validate the approach before building CRUD and CLI layers.

### Recommendation 6: Coordinate with E16 on Workflow Profile Updates

**Rationale**: When `shark init update --workflow=advanced` is run, it should include default bug and change-card workflow definitions. This requires coordination with E16's profile system to add bug/change-card workflow profiles alongside epic/feature/task profiles.

---

## References

- E18 PRD files: `/home/jwwel/projects/shark-task-manager/docs/plan/E18-bug-and-change-card-management-system/`
- E12 epic doc: `/home/jwwel/projects/shark-task-manager/docs/plan/E12-bug-tracker-system/epic.md`
- E08 idea model: `/home/jwwel/projects/shark-task-manager/internal/models/idea.go`
- E16 multi-level workflow: `/home/jwwel/projects/shark-task-manager/docs/plan/E16-multi-level-workflow/epic.md`
- Key service: `/home/jwwel/projects/shark-task-manager/internal/keys/service.go`
- Workflow multilevel: `/home/jwwel/projects/shark-task-manager/internal/config/workflow_multilevel.go`
- Entity note repository: `/home/jwwel/projects/shark-task-manager/internal/repository/entity_note_repository.go`
- File operations: `/home/jwwel/projects/shark-task-manager/internal/fileops/writer.go`
- Database schema: `/home/jwwel/projects/shark-task-manager/internal/db/db.go`
- [Top Bug Tracking Tools (2026)](https://www.featurebase.app/blog/bug-tracking-tools)
- [Bug Tracking Software Market Analysis](https://marker.io/blog/bug-tracking-tools)
- [CLI Bug Tracking Comparison](https://clickup.com/blog/bug-tracking-software/)

---

*Research complete. All 6 sections addressed. No unresolved feasibility blockers. Overlap with sibling epics explicitly documented.*
