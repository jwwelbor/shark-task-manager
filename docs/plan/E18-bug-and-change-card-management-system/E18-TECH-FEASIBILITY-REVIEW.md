# E18 Technical Feasibility Review: Bug and Change-Card Management System

**Date**: 2026-03-02
**Reviewer**: Architect Agent
**Epic**: E18 -- Bug and Change-Card Management System
**Overall Assessment**: **APPROVED**

---

## 1. Technical Feasibility by Requirement Area

### 1.1 Bug Data Model (REQ-F-001 through REQ-F-003) -- FEASIBLE

**Assessment**: Low risk. Direct pattern reuse.

**Evidence**:
- The `ideas` table in `internal/db/db.go` (lines 337-370) demonstrates a standalone entity with auto-increment key, optional linking fields (`converted_to_type`, `converted_to_key`), and standard timestamps. The `bugs` table follows this exact pattern.
- The `Idea` model in `internal/models/idea.go` provides a template for the `Bug` model struct with key validation, status enums, and optional pointer fields.
- The B### key format requires a simple integer counter column, not the date-based I-YYYY-MM-DD-xx format used by ideas. This is simpler -- a single `AUTOINCREMENT` integer column formatted as `B%03d`.

**Proposed Schema** (high-level, detailed design in tech spec phase):

```sql
CREATE TABLE IF NOT EXISTS bugs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,                    -- B001, B002, ...
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'reported',
    severity TEXT NOT NULL DEFAULT 'medium'
        CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    slug TEXT,
    linked_entity_type TEXT,                     -- 'epic', 'feature', 'task', or NULL
    linked_entity_key TEXT,                      -- E07, E07-F01, E07-F01-001, or NULL
    context_data TEXT,                           -- JSON for arbitrary metadata
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Key Decision**: The `severity` field is a dedicated column (not a context field) because it is used for filtering in list queries (`shark bug list --severity=critical`). Making it a top-level column allows indexed queries. Other optional metadata (environment, component, etc.) use the `context_data` JSON column, following the established pattern.

**Technical Concern -- Link Validation**: The `--link` flag requires validating that the linked entity exists. This is a cross-entity lookup (bug references epic/feature/task by key). The service layer must perform this validation since SQLite cannot enforce foreign keys to polymorphic targets. This is the same pattern used by the idea entity's `converted_to_key` field -- no new patterns needed.

---

### 1.2 Change-Card Data Model (REQ-F-007, REQ-F-008) -- FEASIBLE

**Assessment**: Low risk. Simpler than bugs.

**Evidence**: Same pattern as bugs but without the `severity` column.

**Proposed Schema** (high-level):

```sql
CREATE TABLE IF NOT EXISTS change_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,                    -- C001, C002, ...
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'proposed',
    slug TEXT,
    linked_entity_type TEXT,
    linked_entity_key TEXT,
    context_data TEXT,
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**No additional concerns** beyond what is documented for bugs.

---

### 1.3 Bug Workflow (REQ-F-004, REQ-F-005) -- FEASIBLE

**Assessment**: Medium risk. Depends on E16 infrastructure, but the infrastructure exists.

**Evidence**:
- `internal/config/workflow_multilevel.go` defines `MultiLevelWorkflow` with `Epic`, `Feature`, and `Task` fields. Adding `Bug *WorkflowConfig` and `Change *WorkflowConfig` fields is a struct extension.
- `GetWorkflowForLevel()` uses a switch statement that currently handles `"epic"`, `"feature"`, `"task"`. Adding `"bug"` and `"change"` cases is two additional switch branches.
- The `workflow.Service.ForLevel()` method (used by `TaskService`, `FeatureService`, `EpicService`) scopes workflow validation to a specific level. Bug and change-card services use the same pattern: `workflowSvc.ForLevel(workflow.LevelBug)`.

**Bug Workflow Definition**:
```
reported -> triaged -> in_fix -> in_verification -> resolved
                                                 -> wont_fix
            -> duplicate
```

**Change-Card Workflow Definition**:
```
proposed -> approved -> in_progress -> completed
         -> declined
```

**Technical Concern -- Default Workflow Functions**: Each workflow level needs a `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` function that returns the default status flow, metadata, and configuration. These follow the exact pattern of `DefaultEpicWorkflow()` and `DefaultFeatureWorkflow()`.

**Technical Concern -- Configuration File Growth**: `.sharkconfig.json` already holds `status_metadata`, `status_flow`, `epic_workflow`, `feature_workflow`, and `task_workflow` sections. Adding `bug_workflow` and `change_workflow` sections increases the file size. This is acceptable because the config file is machine-generated (via `shark init update --workflow=...`) and not hand-edited. The workflow profile system (basic/advanced) needs to include bug and change-card defaults.

**Dependency Confirmation**: Inspected `internal/config/workflow_multilevel.go` -- the `ForLevel()` infrastructure is implemented and working. The extension is additive with no modifications to existing code paths.

---

### 1.4 Bug and Change-Card CRUD CLI (REQ-F-006, REQ-F-011) -- FEASIBLE

**Assessment**: Low risk. Volume of code is medium but entirely pattern-based.

**Evidence**:
- The `idea.go` CLI command file demonstrates the complete pattern for a standalone entity: `create`, `get`, `list`, `update`, `delete` subcommands, all registered via `init()`.
- The service layer pattern is established: `IdeaService` in `internal/services/idea_service.go` with `CreateIdea()`, `GetIdea()`, `ListIdeas()`, etc.
- The repository pattern is established: `IdeaRepository` in `internal/repository/` with standard CRUD.

**New Files Required**:

| Layer | Bug Files | Change-Card Files |
|-------|-----------|-------------------|
| Model | `internal/models/bug.go` | `internal/models/change_card.go` |
| Repository | `internal/repository/bug_repository.go` | `internal/repository/change_card_repository.go` |
| Service | `internal/services/bug_service.go` | `internal/services/change_card_service.go` |
| CLI Commands | `internal/cli/commands/bug.go` | `internal/cli/commands/change.go` |
| Service DTOs | `internal/services/bug_dto.go` | `internal/services/change_dto.go` |
| Tests (model) | `internal/models/bug_test.go` | `internal/models/change_card_test.go` |
| Tests (repo) | `internal/repository/bug_repository_test.go` | `internal/repository/change_card_repository_test.go` |
| Tests (service) | `internal/services/bug_service_test.go` | `internal/services/change_card_service_test.go` |
| Templates | `templates/bug.md.tmpl` | `templates/change_card.md.tmpl` |

**Total: ~18 new files**. Each follows an established template. The primary development effort is in the service layer (triage logic, approval logic, link validation) and CLI commands (flag parsing, output formatting).

**Technical Concern -- Service Accessor**: `internal/cli/services_global.go` needs `GetBugService()` and `GetChangeCardService()` functions following the existing `GetTaskService()` / `GetIdeaService()` pattern.

---

### 1.5 Core Command Auto-Detection (REQ-F-012) -- FEASIBLE

**Assessment**: Medium risk. Cross-cutting but each change is small.

**Evidence**:
- `internal/keys/service.go` defines `EntityType` constants and `DetectEntityType()` with regex patterns. Adding `EntityTypeBug = "bug"` and `EntityTypeChange = "change"` constants plus B### and C### regex patterns is straightforward.
- `internal/cli/commands/helpers.go` has a secondary `DetectEntityType()` function (line 568) that mirrors the key service. Both must be updated.

**Dispatch Points Inventory** (from grep analysis):

| File | Dispatch Pattern | Action Required |
|------|-----------------|-----------------|
| `keys/service.go` | `DetectEntityType()` regex patterns | Add B###, C### patterns |
| `cli/commands/helpers.go` | `DetectEntityType()` function | Add "bug", "change" cases |
| `cli/commands/update_dispatch.go` | Entity type switch for `shark update` | Add bug, change cases |
| `cli/commands/delete_dispatch.go` | Entity type switch for `shark delete` | Add bug, change cases |
| `cli/commands/status_group.go` | Entity type switch for `shark status` | Add bug, change cases |
| `cli/commands/context.go` | Entity type switch for `shark context` | Add bug, change cases |
| `cli/commands/render_common.go` | Entity type switch for display rendering | Add bug, change cases |
| `cli/commands/notes_search.go` | Entity type switch for notes/search | Add bug, change cases |
| `services/context_service.go` | Entity type switch for context operations | Add bug, change cases |
| `services/resume_service.go` | Entity type switch for resume context | Add bug, change cases |
| `services/note_service.go` | Entity type switch for note operations | Add bug, change cases |
| `services/display_service.go` | Entity type switch for display | Add bug, change cases |

**Total dispatch points: ~12 files, ~15 switch statements**. Each change is 2-4 lines (add `case "bug":` and `case "change":` branches).

**Mitigation Strategy**:
1. Before implementation: run `grep -rn 'EntityType\|entityType\|entity_type' internal/` to create an exhaustive inventory.
2. Add test cases that exercise `shark get B001` and `shark get C001` to catch missing dispatch points.
3. Consider adding a `//go:generate` comment or compile-time check to flag incomplete switches. Go's `exhaustive` linter can enforce this if `EntityType` is changed from `string` to an enum-like iota pattern. However, this is a refactoring decision (entity type registry, Research Recommendation 3) that is out of scope for E18 and should be tracked as a future improvement.

---

### 1.6 Dashboard and Analytics Integration (REQ-F-013, REQ-F-014) -- FEASIBLE

**Assessment**: Medium risk. New sections in established commands.

**Evidence**:
- The `shark status` dashboard command renders output sections for epics, features, and tasks. Adding "Bugs" and "Change Cards" sections requires:
  1. New repository query methods: `CountBugsByStatus()`, `CountBugsBySeverity()`, `CountChangeCardsByStatus()`
  2. New rendering sections in the dashboard command
  3. Conditional display: sections only shown when bugs/change-cards exist

- The `shark analytics` command uses `EpicAnalyticsService`. Bug and change-card analytics can follow the same pattern or be added as sections within the existing analytics output.

**Technical Concern -- Dashboard Complexity**: With 5 entity types, the dashboard output could become overwhelming. The design principle is: show sections conditionally (only when entities of that type exist) and use summary counts rather than full lists. The feature status display already shows linked bug count if bugs are linked (REQ-F-013 acceptance criteria 4).

---

### 1.7 Notes, Context, and Markdown Templates (REQ-F-015 through REQ-F-018) -- FEASIBLE

**Assessment**: Low risk. Existing infrastructure directly supports this.

**Evidence**:
- `entity_notes` table has a CHECK constraint: `entity_type IN ('epic', 'feature', 'task')`. This must be relaxed via a database migration to include `'bug'` and `'change'`. The migration is straightforward:
  ```sql
  -- SQLite does not support ALTER TABLE ... ALTER COLUMN for CHECK constraints.
  -- Approach: Create new table, copy data, drop old, rename.
  -- OR: Since this is a migration, just add the new values via a new table creation
  -- that replaces the CHECK constraint.
  ```

  **Important**: SQLite does not support modifying CHECK constraints on existing tables. The migration must either:
  - (a) Recreate the table with the updated constraint (copy data, drop old, rename), OR
  - (b) Remove the CHECK constraint entirely and enforce entity_type validation at the application/service layer instead.

  Option (b) is recommended. The service layer already validates entity types before inserting notes. The CHECK constraint is defense-in-depth, not the primary validation point. Removing it simplifies future entity type additions. This is a one-time migration that the auto-migration system in `db.go` handles.

- `EntityNoteRepository` in `internal/repository/entity_note_repository.go` uses `entity_type` as a parameter in queries. Adding `"bug"` and `"change"` as valid entity type values requires no repository code changes -- the queries are parameterized.

- `ContextService` in `internal/services/context_service.go` switches on entity type to find the correct repository. Adding bug and change-card cases to this switch is a 2-line change each.

- Markdown templates follow the `templates/` package pattern. Bug templates include sections for Reproduction Steps, Expected Behavior, Actual Behavior, Environment, and Additional Context. Change-card templates are simpler (Description, Justification, Linked Entity).

---

### 1.8 Non-Functional Requirements (REQ-NF-001 through REQ-NF-006) -- FEASIBLE

**Assessment**: Low risk. No new performance patterns required.

| Requirement | Assessment |
|-------------|------------|
| **Performance** (< 500ms local, < 2s cloud) | SQLite CRUD for a simple table will match existing entity performance. No concern. |
| **Atomic Operations** | The `fileops` package handles atomic file+DB operations. Bug/change-card creation uses `WriteEntityFile()` with `UseAtomicWrite: true`. Same pattern as tasks. |
| **Key Uniqueness** | `UNIQUE` constraint on the `key` column. Auto-increment prevents collisions. |
| **CLI Consistency** | Following established CLI patterns (from `.claude/rules/cli/commands.md`) guarantees consistency. |
| **Workflow Profile Integration** | Depends on `ForLevel()` infrastructure, which exists. Bug/change-card workflow profiles added to `shark init update --workflow=advanced`. |

---

## 2. Architectural Concerns

### 2.1 Cross-Cutting Entity Type Dispatch -- MANAGEABLE

**Concern**: Every entity-type switch statement in the codebase (approximately 15 across 12 files) must be updated to include `"bug"` and `"change"` cases. Missing any dispatch point creates a silent failure.

**Assessment**: This is the primary architectural concern. It is inherent to the current codebase design where entity types are dispatched via string-matching switch statements.

**Mitigation**:
1. Create a pre-implementation checklist of all dispatch points (grep inventory).
2. Add integration tests that exercise `shark get B001`, `shark get C001`, `shark status advance B001`, `shark status advance C001` through the unified commands.
3. The entity type registry pattern (Research Recommendation 3) would eliminate this concern class for future entity types, but is out of scope for E18.

**Verdict**: Manageable with disciplined implementation. Not a blocker.

### 2.2 Database Migration for entity_notes CHECK Constraint -- MANAGEABLE

**Concern**: The `entity_notes` table has a CHECK constraint `entity_type IN ('epic', 'feature', 'task')` that must be relaxed to include `'bug'` and `'change'`. SQLite does not support `ALTER TABLE` for CHECK constraints.

**Assessment**: This requires a table recreation migration (create new table, copy data, drop old, rename new). This is a known SQLite limitation and the auto-migration system in `db.go` already handles similar migrations.

**Recommended Approach**: Remove the CHECK constraint entirely during the migration and rely on application-layer validation. This makes future entity type additions require zero database migrations for the notes table.

**Verdict**: One-time migration. Well-understood pattern. Not a blocker.

### 2.3 Workflow Engine Scaling -- LOW CONCERN

**Concern**: Adding two more workflow levels (bug, change) to the multi-level workflow engine increases configuration complexity.

**Assessment**: The `ForLevel()` abstraction cleanly isolates workflow levels. Adding two more levels is linear complexity, not exponential. Each level's workflow is independent (no cross-level references needed for bugs/changes). The configuration file growth is acceptable since it is machine-generated.

**Verdict**: No concern. Linear extension of proven pattern.

### 2.4 Key Format Namespace -- LOW CONCERN

**Concern**: The B### and C### key formats use single-letter prefixes from a limited namespace (26 letters).

**Assessment**: With current entity types (E, F, T, I, B, C), 6 of 26 letters are used. This leaves 20 single-letter prefixes for future entity types. If the namespace becomes exhausted, multi-letter prefixes (e.g., "CP###" for components) are possible without changing the key detection logic.

**Verdict**: Not a concern for E18. Document the prefix allocation in architecture docs as a future reference.

---

## 3. Dependency and Integration Risks

### 3.1 E16 Multi-Level Workflow System -- LOW RISK

**Status**: E16 is active. The `ForLevel()` infrastructure is implemented and working for epic/feature/task levels.

**Impact**: E18 extends `ForLevel()` with `LevelBug` and `LevelChange` constants. This is an additive change that does not modify existing behavior.

**Risk Assessment**: Low. The core dependency is met. If E16 has any remaining gaps in profile integration (e.g., `shark init update --workflow=advanced` not yet including per-level profiles), E18 can address those gaps as part of its own implementation.

### 3.2 E15 Service Layer -- NO RISK

**Status**: E15 is active. Service layer patterns are mature and documented.

**Impact**: E18 adds two new services following established patterns. No E15 modification required.

### 3.3 E17 CLI Unification -- NO RISK

**Status**: E17 is completed.

**Impact**: E18 extends the unified CLI patterns. The existing dispatch infrastructure accepts the extension.

### 3.4 Cross-Epic Technical Conflicts -- NONE FOUND

No technical conflicts with any active or completed epic. E12 (original bug tracker) is superseded and should be cancelled/archived.

---

## 4. Technical Debt Implications

### 4.1 Debt This Epic Creates -- MINIMAL

**New Technical Debt**: The scattered entity-type switch statements grow from 3 cases to 5 cases in ~15 locations. This increases the maintenance burden slightly but does not create a new debt category -- it extends an existing pattern.

**Mitigation**: The entity type registry pattern (Research Recommendation 3) would eliminate this debt class entirely. This is a future improvement tracked as a potential follow-on epic.

### 4.2 Debt This Epic Exposes -- MODERATE (Pre-existing)

**Exposed Debt**: The entity type dispatch pattern (scattered switch statements) is a pre-existing architectural concern that becomes more visible with 5 entity types. This is not debt that E18 creates; E18 reveals it by adding two more entity types to the existing pattern.

**Recommendation**: Track the entity type registry as a follow-on improvement. Do not block E18 on this refactoring.

### 4.3 Debt This Epic Resolves -- NONE

E18 does not address existing technical debt. It is a feature epic.

---

## 5. Implementation Recommendations

### Recommendation 1: Phase Implementation by Priority

Follow the research report's phased approach:

1. **Phase 1**: Data models + workflows + CRUD for both bugs and change-cards. This delivers core value.
2. **Phase 2**: Unified CLI integration (B###/C### auto-detection), dashboard, analytics.
3. **Phase 3**: Notes, context, markdown templates, linked-entity filtering.
4. **Phase 4**: Promotion features, duplicate detection.

### Recommendation 2: Implement Workflow Engine Extension First

The workflow engine extension (adding `LevelBug` and `LevelChange` to `ForLevel()`) is a foundational dependency. Implement and validate it before building CRUD and CLI layers. This de-risks the E16 dependency early.

### Recommendation 3: Remove entity_notes CHECK Constraint

Instead of recreating the table with an updated CHECK constraint, remove the constraint entirely and rely on application-layer validation. This:
- Eliminates a migration risk
- Makes future entity type additions frictionless
- Aligns with the principle that business rules belong in the service layer, not in database constraints for polymorphic fields

### Recommendation 4: Create Entity Type Dispatch Inventory Before Implementation

Before writing any new code, run a comprehensive grep for all entity type dispatch points. Create a checklist in the first implementation task. Use this checklist as a verification gate before marking Phase 2 (unified CLI integration) complete.

### Recommendation 5: Follow E15 Service Layer Patterns Strictly

All new code must follow the service layer architecture:
- **CLI commands**: Thin wrappers (parse, call service, format)
- **Services**: All business logic (triage, approval, link validation, workflow transitions)
- **Repositories**: Pure CRUD (no business rules)
- **Models**: Structural validation only (no workflow status lists)

Since E18 is entirely new code (no legacy commands to refactor), it has the advantage of starting clean with the target architecture.

### Recommendation 6: Cancel/Archive E12

E18 explicitly supersedes E12. Archive E12 before starting E18 development to prevent confusion. Reference E18 as the superseding epic.

---

## 6. Overall Assessment

### APPROVED

E18 "Bug and Change-Card Management System" passes the technical feasibility review on all evaluation criteria.

| Criterion | Result | Summary |
|-----------|--------|---------|
| Technical Feasibility | PASS | All 21 requirements (14 Must, 4 Should, 3 Could) are implementable with existing technology and patterns. No new technology adoption required. |
| Architectural Concerns | PASS | Cross-cutting entity type dispatch is the primary concern, mitigated by pre-implementation inventory and integration tests. entity_notes CHECK constraint requires migration but is a standard operation. |
| Dependency & Integration Risks | PASS | E16 dependency confirmed met (ForLevel infrastructure exists). No cross-epic conflicts. Clean extension of E15, E17 patterns. |
| Technical Debt Implications | PASS | Minimal new debt. Exposes pre-existing dispatch pattern debt that should be tracked for future improvement (entity type registry). |

**Estimated Complexity**: Medium-Large. Approximately 18 new files, 15 modified files, 60% pattern reuse, 40% new entity-specific logic.

**Confidence Level**: High. The 60% pattern reuse ratio significantly reduces implementation risk. All new components follow proven patterns from the existing codebase. The primary risk (cross-cutting dispatch points) is well-understood and mitigable.

The epic is ready to advance to the next workflow stage.

---

*Reviewed against: E18 PRD (6 documents), E18-EPIC-RESEARCH-REPORT.md, E18-BA-FEASIBILITY-REVIEW.md, and direct codebase inspection of internal/models/, internal/repository/, internal/services/, internal/cli/commands/, internal/keys/, internal/config/, internal/db/.*
