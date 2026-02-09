# BA Review: E16-F01 Core Workflow Engine

**Reviewer**: Business Analyst
**Date**: 2026-02-08
**Status**: Review Complete -- Ready for Technical Architecture with Noted Gaps
**Feature PRD**: `docs/plan/E16-multi-level-workflow/E16-F01-core-workflow-engine/feature.md`
**Epic PRD**: `docs/plan/E16-multi-level-workflow/epic.md`

---

## 1. Requirements Completeness Assessment

### 1.1 Epic Requirements Coverage

The epic defines 10 functional requirements (FR-1 through FR-10). The feature PRD for F01 claims to cover FR-1, FR-2, FR-3, FR-4, and FR-7. This section assesses whether those claims hold up and whether the scoping is correct.

| Epic FR | Description | Covered in F01? | Assessment |
|---------|-------------|-----------------|------------|
| FR-1 | Level-specific workflow configuration | Yes (REQ-F-001 through REQ-F-005) | **Fully covered.** Config parsing, backward compatibility, and independent validation are all addressed. |
| FR-2 | Epic status transitions with validation | Yes (REQ-F-006) | **Partially covered.** Transition validation is addressed, but audit trail/history recording (FR-2.4) is mentioned without resolution. See Finding 2.1. |
| FR-3 | Feature status transitions with validation | Yes (REQ-F-007) | **Same gap as FR-2** regarding history recording. |
| FR-4 | `next-status` for epic and feature | Yes (REQ-F-009 through REQ-F-011) | **Covered, with one ambiguity.** The JSON response structure in the feature PRD includes `orchestrator_action`, but the "Out of Scope" section says orchestrator actions are handled by F02. See Finding 2.2. |
| FR-5 | Orchestrator actions for epic/feature | Explicitly out of scope (F02) | **Correct scoping.** However, the `next-status` JSON response example in F01 includes `orchestrator_action`. This is contradictory. |
| FR-6 | Aggregation threshold behavior | Explicitly out of scope (F03) | **Correct scoping.** |
| FR-7 | Workflow validation for all levels | Yes (REQ-F-012) | **Covered.** |
| FR-8 | Workflow list for all levels | Explicitly out of scope (F06) | **Correct scoping.** |
| FR-9 | Upward escalation | Explicitly out of scope (F05) | **Correct scoping.** |
| FR-10 | Notes and context for epic/feature | Explicitly out of scope (F04) | **Correct scoping.** |

### 1.2 Findings

**Finding 2.1: Audit trail/history for epic and feature status changes is undefined.**

The epic PRD states in FR-2.4: "History/audit trail records epic status changes (if history system supports it)." The current database has a `task_history` table but no `epic_history` or `feature_history` tables. The feature PRD does not address this at all.

- The current `task_history` table schema is task-specific (has `task_id` foreign key).
- Open Question 4 in the epic asks whether to add an `entity_type` column or create separate tables.
- This question must be answered before F01 can be fully specified, because the `next-status` and `update --status` commands need to know whether they should record history.

**Recommendation**: Either (a) define history recording as out of scope for F01 and create a separate follow-up task, or (b) resolve Open Question 4 and include history schema in F01. Option (a) is preferred since F01 is already complex enough.

**Finding 2.2: Orchestrator action in `next-status` JSON response contradicts scoping.**

The feature PRD's Acceptance Criteria Scenario 4 and the JSON response example (REQ-F-011) show `orchestrator_action` being returned. However, the "Out of Scope" section explicitly states: "Orchestrator action responses -- Handled by E16-F02."

This is a contradiction. The `next-status` response structure needs to be defined in F01 (it is part of the JSON contract), but the actual population of `orchestrator_action` data should be deferred to F02.

**Recommendation**: F01 should define the JSON response structure with `orchestrator_action: null` always. F02 will populate it. Update the acceptance criteria to clarify this boundary.

**Finding 2.3: `is_planning` and `aggregates_from` metadata fields are referenced but not defined in F01.**

The epic PRD introduces two new `StatusMetadata` fields: `is_planning` (boolean) and `aggregates_from` (string). These fields appear in the proposed config JSON but are not in the current `StatusMetadata` struct (`internal/config/workflow_schema.go`). The feature PRD for F01 does not explicitly list adding these fields to the struct.

F03 (Display & Aggregation) is where these fields are consumed, but F01 needs to parse them from the config file. If F01 does not add these fields to the struct, F03 will need to modify the parser, creating an unnecessary coupling.

**Recommendation**: F01 should add `is_planning` and `aggregates_from` to the `StatusMetadata` struct definition even though they are not consumed until F03. This follows the principle that the parser should handle all fields in the config it is designed to read.

### 1.3 Requirements within F01 that are Complete and Well-Defined

- REQ-F-001, REQ-F-002: Config parsing is well-specified with clear JSON structure from the epic.
- REQ-F-003: Backward compatibility is explicitly addressed with the existing top-level `status_flow` remaining as the task workflow.
- REQ-F-004: Default fallback to `draft`, `active`, `completed`, `archived` is clearly specified.
- REQ-F-005: Independent validation per level is stated.
- REQ-F-008: Force override is specified and mirrors existing task behavior.
- REQ-F-012: Workflow validation extension is specified.

---

## 2. Edge Cases Identified

### 2.1 Configuration Edge Cases

**EC-1: Both `epic_workflow` and top-level `status_flow` define overlapping statuses.**
- The epic and task workflows can have identically named statuses (e.g., both have `completed`, `blocked`). The parser must ensure these are kept independent and do not collide in the global workflow cache.
- **Current risk**: The existing `LoadWorkflowConfig` uses a single global cache (`workflowCache`). A multi-level parser must cache per-level or use a different caching strategy.

**EC-2: `epic_workflow` is present but `status_flow` is empty or missing within it.**
- What happens if the config contains `"epic_workflow": {}` (present but empty)?
- **Expected behavior**: Treat as "not configured" and fall back to defaults.
- **Not specified in PRD.**

**EC-3: `epic_workflow.status_flow` references a status not in `epic_workflow.status_metadata`.**
- The existing task workflow validator (`ValidateWorkflow`) checks for unreachable statuses and missing terminal states, but does not enforce that all `status_flow` keys have corresponding `status_metadata` entries.
- **Recommendation**: The epic states in NFR-3 that "Missing required fields produce clear warnings." This should be acceptance criteria for F01: a warning (not an error) if a status is in `status_flow` but not in `status_metadata`.

**EC-4: User provides `status_flow` under `epic_workflow` but misspells the key (e.g., `statusFlow`).**
- JSON parsing will silently ignore the misspelled key and treat the workflow as unconfigured.
- **Recommendation**: Add a validation step that warns about unknown keys within `epic_workflow` and `feature_workflow` sections.

**EC-5: Task-level workflow is absent but epic/feature workflows are present.**
- Current behavior: if `status_flow` is missing at top level, `LoadWorkflowConfig` returns `nil` (no error), and callers use `DefaultWorkflow()`.
- With multi-level workflows, the absence of top-level `status_flow` should not prevent `epic_workflow` or `feature_workflow` from loading.
- **Current parser will return `nil` and stop**, because it checks `_, hasStatusFlow := rawConfig["status_flow"]` first. This is a blocking implementation concern.

### 2.2 Transition Edge Cases

**EC-6: Entity already at the "next" status when `next-status` is called.**
- Example: Epic is at `active`, user runs `shark epic next-status E16`. The first entry in `status_flow["active"]` is `completed`. But what if the user calls it again when already at `completed` (a terminal status)?
- The existing task `next-status` handles this with: "Task is in terminal status -- no transitions available." Same behavior should apply.
- **Partially covered** by Acceptance Criteria but should be explicit for epic/feature.

**EC-7: Entity has a status that is not recognized by the configured workflow.**
- Example: An epic was created with status `active` (default workflow). User then adds a custom `epic_workflow` that does not include `active` as a status.
- **Expected behavior**: The `next-status` command should produce a clear error. The `update --status` command should still work with `--force`.
- **Not specified in PRD.** This is a real migration scenario.

**EC-8: `--force` overrides validation, but the target status does not exist in the workflow at all.**
- With `--force`, should any arbitrary string be accepted as a status? Or should `--force` only bypass transition validation while still requiring the target status to exist?
- The existing task behavior (`UpdateStatusForced`) accepts any status with `--force`. The same behavior should apply, but this should be documented.

**EC-9: Case sensitivity in status names.**
- The existing `workflow.Service.IsValidTransition` uses `strings.EqualFold` for case-insensitive comparison.
- The `status_flow` map keys are case-sensitive in Go. If a user types `--status Draft` vs `--status draft`, the behavior should be consistent.
- **The existing implementation handles this** via `NormalizeStatus()`, but the feature PRD should confirm this extends to epic/feature workflows.

### 2.3 Command Edge Cases

**EC-10: `shark epic next-status` on an epic that does not exist.**
- Should return a clear "Epic not found" error with exit code 1.
- **Not explicitly in acceptance criteria** but implied by existing patterns.

**EC-11: `shark feature next-status` with a feature key that matches multiple epics.**
- Example: `shark feature next-status F01` -- which epic's F01?
- The existing feature key resolution handles this, but the PRD should confirm the `next-status` command accepts the same key formats (both `E16-F01` and `F01`).

**EC-12: Concurrent `next-status` calls on the same entity.**
- Two orchestrator agents might call `shark epic next-status E16` simultaneously.
- The database update is atomic, so one will succeed and the other will see a stale status and may fail or produce an unexpected transition.
- **Recommendation**: Document that callers should check the return value and handle stale-state errors gracefully.

---

## 3. Risk Analysis

### 3.1 Technical Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Global workflow cache is task-only.** The `workflowCache` variable in `workflow_parser.go` caches a single `WorkflowConfig`. Multi-level workflows need either three separate caches or a composite struct. | High | Certain | Design the composite `MultiLevelWorkflowConfig` struct in technical architecture. |
| **`ValidateEpicStatus` and `ValidateFeatureStatus` in `models/validation.go` are hardcoded to 4 statuses.** These functions are called by `Epic.Validate()` and `Feature.Validate()`. With configurable statuses, these validators will reject custom statuses. | High | Certain | These validators must be changed to accept any non-empty string (like `ValidateTaskStatus` already does). This is a **breaking change** to the model validation layer. |
| **`EpicStatus` and `FeatureStatus` types are Go enums with const values.** Code throughout the codebase compares against `models.EpicStatusDraft`, `models.EpicStatusActive`, etc. Configurable statuses break these type-safe comparisons. | High | Certain | Must refactor `EpicStatus`/`FeatureStatus` types to `string` (like `TaskStatus` already is -- it is `type TaskStatus string` but without hardcoded const restrictions). |
| **Database CHECK constraint on epic status.** The `epics` table does NOT have a CHECK constraint on status (confirmed in `db.go`). The `features` table also has no CHECK on status. This is actually favorable -- no schema migration needed. | Low | N/A | No action needed. |
| **`ParseEpicStatus` function in CLI commands is hardcoded.** The `epic.go` and `feature.go` CLI commands use `ParseEpicStatus()` and `ParseFeatureStatus()` which only accept the 4 hardcoded statuses. | High | Certain | These parse functions must become workflow-aware. |
| **`PopulateTemplate` only supports `{task_id}` placeholder.** The epic PRD uses `{id}` as the placeholder. The existing `PopulateTemplate` method in `orchestrator_action.go` replaces `{task_id}` only. | Medium | Certain | Either (a) keep `{task_id}` for backward compatibility and add `{id}` as a new placeholder, or (b) use `{id}` universally and deprecate `{task_id}`. This is an F02 concern but the design decision affects F01's JSON response structure. |

### 3.2 Business Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Scope creep from orchestrator actions.** The boundary between F01 and F02 is unclear around the `orchestrator_action` field in the `next-status` response. | Medium | High | Explicitly define F01 returns `orchestrator_action: null`. F02 populates it. |
| **Backward compatibility regression.** Users with existing `.sharkconfig.json` files could experience unexpected behavior if the parser changes. | High | Low | REQ-NF-003 covers this. Integration tests with real-world config files should be required. |
| **Complexity of maintaining three workflow definitions.** Each feature going forward must consider epic, feature, AND task workflows. This triples the testing surface for any workflow-related change. | Medium | Medium | The architectural decision to keep workflows independent (no cross-references) mitigates this. Each level is self-contained. |

### 3.3 Dependency Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **E15 (Service Layer Refactoring) is not complete.** The architecture guidelines state that new commands should use the service layer, but the existing epic/feature commands use the legacy direct-repo pattern. F01 needs to decide: follow legacy pattern or create new services? | Medium | Certain | Recommendation: Create `EpicWorkflowService` and `FeatureWorkflowService` (or extend the existing `workflow.Service`) following the service layer pattern. Do not add more fat controller commands. |
| **`init update --workflow` command generates workflow profiles.** If F01 adds new config keys (`epic_workflow`, `feature_workflow`), the `init update --workflow=advanced` command needs to include them. | Medium | High | Coordinate with the workflow profiles system. F01 should provide default advanced-profile definitions for epic and feature workflows. |

---

## 4. Dependency Validation

### 4.1 Upstream Dependencies

| Dependency | Status | Impact on F01 |
|------------|--------|---------------|
| **E11 - Configurable Status Workflow System** | Delivered | Foundation. F01 extends E11's config parsing, validation, and `WorkflowConfig` types. |
| **E13 - Workflow-Aware Task Command System** | Delivered | Pattern to replicate. `task next-status`, `workflow show-actions`, and `workflow validate` commands exist and work. F01 mirrors these for epic/feature. |
| **`workflow.Service` (`internal/workflow/service.go`)** | Exists, task-only | Must be extended or replaced with a level-aware version. The existing `NewService(projectRoot)` only loads task workflow. |
| **`config.LoadWorkflowConfig` (`internal/config/workflow_parser.go`)** | Exists, task-only | Must be extended to parse `epic_workflow` and `feature_workflow` sections. The current parser's early-exit on missing `status_flow` is a blocker (see EC-5). |
| **`config.WorkflowConfig` struct** | Exists | Must add `is_planning` and `aggregates_from` fields to `StatusMetadata`. May also need a container struct for multi-level configs. |
| **`validation.StatusValidator`** | Exists, task-only | Must be parameterized by entity type or accept any `WorkflowConfig`. Currently it works with any `WorkflowConfig` already (it takes it as a constructor argument), so it may just work once the correct config is passed in. |

### 4.2 Downstream Impacts

| Dependent | Impact from F01 |
|-----------|-----------------|
| **E16-F02 (Orchestrator Actions)** | F02 needs the workflow engine to load epic/feature workflow configs and return `orchestrator_action` in transition responses. F01 must provide the parsing infrastructure. |
| **E16-F03 (Display & Aggregation)** | F03 needs `is_planning` metadata to determine display mode. F01 should parse this field even if F03 consumes it. |
| **E16-F04 (Notes & Context)** | F04 needs entity-level workflow awareness. Minimal F01 impact -- F04 just needs to know which workflow to use for a given entity type. |
| **E16-F05 (Backward Transition)** | F05 needs transition validation to work at epic/feature level. F01 provides this. F05 adds `--reason` enforcement. |
| **E16-F06 (Workflow Visualization)** | F06 needs multi-level workflow data. F01 provides the parsed configs. |
| **`shark init update --workflow=advanced`** | Must include epic and feature workflow definitions in the advanced profile. |
| **`shark sync`** | Status syncing currently ignores status values from files (database is source of truth). This remains unchanged, but sync should recognize new status values as valid. |

---

## 5. Open Questions Resolution (as they relate to F01)

### Open Question 1: Should `active` be configurable or always the aggregation threshold?

**F01 Impact**: LOW. F01 only needs to parse `_aggregation_` from `special_statuses` and store it. The interpretation of aggregation behavior is F03's responsibility.

**Recommendation for F01**: Parse and validate `_aggregation_` as a special status key alongside `_start_` and `_complete_`. Do not hardcode `active` as the only valid aggregation status. Allow the config to define it. This keeps F01 flexible for future changes.

**Add to F01 scope**: Extend `ValidateWorkflow()` to validate `_aggregation_` entries (if present) exist in `status_flow`.

### Open Question 2: Should `shark feature next-status` auto-transition to `active` after `ready_to_build`?

**F01 Impact**: NONE directly. The `next-status` command always uses the first entry in the `status_flow` array. If `ready_to_build` transitions to `["active", "blocked"]`, then `next-status` will auto-select `active`. The ordering in the config array controls this.

**Recommendation**: No F01 code change needed. Document in the feature PRD that the order of entries in `status_flow` arrays determines `next-status` behavior.

### Open Question 3: Should epic completion be auto-detected?

**F01 Impact**: NONE for F01. This is an F03 concern (aggregation behavior). F01 provides the transition commands; it does not auto-trigger transitions.

**Recommendation**: Defer to F03. F01 should not implement auto-completion logic.

### Open Question 4: History table schema

**F01 Impact**: HIGH if history recording is in scope; NONE if deferred.

**Current state**: The `task_history` table has these columns:
- `id`, `task_id` (FK to tasks), `from_status`, `to_status`, `agent`, `notes`, `rejection_reason`, `rejection_reason_doc`, `created_at`

**Options**:
- (a) Add `entity_type` column to `task_history` and rename to `status_history` -- requires migration
- (b) Create separate `epic_history` and `feature_history` tables -- simpler, no migration on existing table
- (c) Defer history recording for epic/feature to a later feature

**Recommendation**: Option (c). Defer history to a follow-up. F01 is already large enough. Status transitions will still work; they just will not have an audit trail initially. Add a TODO in the code marking where history recording should be added.

### Open Question 5: Feature status in `shark task list` output

**F01 Impact**: NONE. This is a display concern (F03 or a separate enhancement). F01 does not modify the `task list` output.

---

## 6. Refined Acceptance Criteria

The following acceptance criteria should be added to or refined in the feature PRD.

### AC-ADD-1: Config parser handles missing top-level `status_flow` with epic/feature workflows present

**Given** a `.sharkconfig.json` that contains `epic_workflow` and/or `feature_workflow` sections but does NOT contain a top-level `status_flow` key
**When** any shark command initializes
**Then** the epic/feature workflows are loaded successfully and the task workflow falls back to the default (basic 5-status workflow)

*Rationale*: The current parser exits early if `status_flow` is missing, which would prevent epic/feature workflow loading. This is a critical path that must work.

### AC-ADD-2: Hardcoded status validators accept configurable statuses

**Given** a custom `epic_workflow` with statuses like `ready_for_research`, `in_research`, etc.
**When** `shark epic update E16 --status ready_for_research` is executed
**Then** the status is accepted without error (the hardcoded `ValidateEpicStatus` function does not reject it)

*Rationale*: The current `ValidateEpicStatus()` in `models/validation.go` only accepts `draft`, `active`, `completed`, `archived`. This will reject any custom status.

### AC-ADD-3: Default workflow behavior preserved exactly

**Given** a fresh shark installation with default `.sharkconfig.json` (no `epic_workflow` or `feature_workflow`)
**When** `shark epic update E16 --status active` is executed
**Then** the behavior is identical to the current codebase (no new validation, no rejection, same output)
**And** `shark epic get E16 --json` returns status as `"active"` with no additional workflow-related fields

*Rationale*: NFR-003 requires backward compatibility, but the acceptance criteria should be testable.

### AC-ADD-4: Workflow cache handles multi-level configs

**Given** a `.sharkconfig.json` with all three workflow levels configured
**When** `shark epic next-status E16` and then `shark feature next-status E16-F01` and then `shark task next-status E16-F01-001` are run in sequence
**Then** each command uses the correct level-specific workflow (no cross-contamination between cached configs)

*Rationale*: The global workflow cache (`workflowCache`) currently stores a single config. Multi-level support requires isolation.

### AC-ADD-5: Entity not found error handling

**Given** an epic key `E99` that does not exist in the database
**When** `shark epic next-status E99` is executed
**Then** error message: "Epic E99 not found" with exit code 1

### AC-ADD-6: Terminal status handling for `next-status`

**Given** an epic at status `completed` (a terminal status with empty transitions array)
**When** `shark epic next-status E16` is executed
**Then** message: "Epic is in terminal status 'completed' -- no transitions available" (matching task `next-status` behavior)

### AC-ADD-7: `--force` bypass for epic/feature transitions

**Given** an epic at status `draft` with a configured workflow where `draft` cannot transition to `completed`
**When** `shark epic update E16 --status completed --force` is executed
**Then** the status is updated to `completed` without validation error
**And** a warning is displayed: "Workflow validation bypassed with --force"

### AC-ADD-8: Invalid configuration produces actionable error

**Given** an `epic_workflow` section with a `status_flow` entry referencing an undefined status (e.g., `"draft": ["nonexistent_status"]`)
**When** `shark workflow validate` is executed
**Then** error: "Epic workflow: undefined status references in transitions: draft -> nonexistent_status"
**And** fix suggestion is provided

### AC-REFINE Scenario 3 (Invalid Transition Rejected):

Current wording:
> "Cannot transition from 'draft' to 'in_refinement'. Valid: ready_for_research, active, cancelled"

Should explicitly state the entity type in the error message:
> "Cannot transition **epic** E16 from 'draft' to 'in_refinement'. Valid transitions: ready_for_research, active, cancelled. Use --force to override."

---

## 7. Recommendation

### Verdict: Ready for Technical Architecture -- with the following conditions:

1. **Resolve the orchestrator_action boundary.** Clarify in the feature PRD that F01 defines the JSON response structure but always returns `orchestrator_action: null`. F02 populates it. Update Scenario 4 and the JSON examples accordingly.

2. **Decide on history recording scope.** Recommend deferring to a follow-up, but the tech arch should identify where the hook points will be so that adding history later is non-disruptive.

3. **Acknowledge the model validation refactoring.** The tech arch must address the `ValidateEpicStatus`/`ValidateFeatureStatus` hardcoded validators and the `EpicStatus`/`FeatureStatus` type constants. This is a prerequisite change that affects code outside of the workflow engine.

4. **Add `is_planning` and `aggregates_from` to `StatusMetadata` struct.** Even though F03 consumes these, F01 should parse them. The tech arch should include this in the `StatusMetadata` struct definition.

5. **Design the multi-level config cache.** The current single-cache architecture in `workflow_parser.go` is insufficient. The tech arch must propose either (a) a composite struct with three workflow configs, (b) a map of entity type to config, or (c) separate cache variables.

6. **Address the parser early-exit issue (EC-5).** The current `LoadWorkflowConfig` returns `nil` when top-level `status_flow` is absent. This blocks loading epic/feature workflows when the task workflow is not customized. The tech arch must redesign the parser entry point.

### Implementation Sizing

Based on the scope analysis, this feature involves:

- Extending `config.WorkflowConfig` / parser (M)
- Extending `workflow.Service` for multi-level (M)
- Refactoring `models/validation.go` for configurable statuses (S)
- Refactoring `EpicStatus`/`FeatureStatus` types or adding workflow-aware validation bypass (M)
- New `epic next-status` CLI command (S -- mirrors existing `task next-status`)
- New `feature next-status` CLI command (S -- mirrors existing `task next-status`)
- Enhancing `epic update --status` with workflow validation (S)
- Enhancing `feature update --status` with workflow validation (S)
- Extending `workflow validate` for multi-level (S)
- Default workflow definitions for epic/feature (S)
- Integration tests and backward compatibility tests (M)

**Overall estimate**: L (multiple M-sized pieces plus integration complexity)

Recommendation is to break implementation into 5-7 tasks following the task template pattern.

---

## Appendix A: Codebase Impact Map

Files that will require modification for F01:

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/config/workflow_schema.go` | Modify | Add `IsPlanning`, `AggregatesFrom` to `StatusMetadata`. Possibly add `MultiLevelWorkflowConfig` composite struct. |
| `internal/config/workflow_parser.go` | Modify | Parse `epic_workflow` and `feature_workflow` sections. Redesign cache. Handle missing top-level `status_flow`. |
| `internal/config/workflow_default.go` | Modify | Add `DefaultEpicWorkflow()` and `DefaultFeatureWorkflow()` functions. |
| `internal/config/workflow_validator.go` | Modify | Extend `ValidateWorkflow` for epic/feature configs. Add `_aggregation_` validation. |
| `internal/workflow/service.go` | Modify | Make level-aware. Add `NewEpicService`, `NewFeatureService`, or parameterize by entity type. |
| `internal/models/validation.go` | Modify | Change `ValidateEpicStatus` and `ValidateFeatureStatus` to accept any non-empty string (like `ValidateTaskStatus`). |
| `internal/models/epic.go` | Potentially modify | May need to change `EpicStatus` type handling to be compatible with configurable statuses. |
| `internal/models/feature.go` | Potentially modify | Same as above for `FeatureStatus`. |
| `internal/cli/commands/epic.go` | Modify | Enhance `runEpicUpdate` to validate against workflow. Remove hardcoded `ParseEpicStatus`. |
| `internal/cli/commands/feature.go` | Modify | Same as above for feature. |
| `internal/cli/commands/epic_next_status.go` | **New file** | New `shark epic next-status` command, mirroring `task_next_status.go`. |
| `internal/cli/commands/feature_next_status.go` | **New file** | New `shark feature next-status` command. |
| `internal/cli/commands/validators.go` | Modify | Update `ParseEpicStatus`, `ParseFeatureStatus` to be workflow-aware. |
| `internal/validation/workflow_validator.go` | Modify | Ensure `StatusValidator` works with any workflow config (already parameterized, may just need testing). |

## Appendix B: Glossary

- **Aggregation threshold**: The status at which an entity stops using its own workflow status and begins deriving progress from child entities. Currently `active`.
- **Planning status**: Any status before the aggregation threshold where the entity has its own workflow state (e.g., `draft`, `ready_for_research`).
- **Level-specific workflow**: Independent workflow definitions for epic, feature, and task entities.
- **Next-status**: The first entry in the `status_flow` array for the current status. The "happy path" forward transition.
