# UAT Acceptance Plan: E20 -- Shark Templates

**Epic**: E20 -- Shark Templates
**Created**: 2026-03-18
**Author**: QA Agent (Claude Opus 4.6, 1M context)
**Status**: Active

---

## Overview

This plan defines how we validate that E20 delivers on its promises: externalizing workflow and template configuration from `.sharkconfig.json` into a dedicated `.sharkworkflow.json` file, standardizing the task workflow block structure, and preserving backward compatibility. Since this is internal tooling with binary success criteria (works or does not work), the plan focuses on concrete scenario-based validation rather than user persona journeys.

---

## (1) Acceptance Scenarios by Requirement Area

### Area A: Workflow File Introduction (REQ-F-001, REQ-F-004)

**AS-A01: System detects and loads `.sharkworkflow.json` from project root**

- Preconditions: Project has both `.sharkconfig.json` (runtime settings only) and `.sharkworkflow.json` (all five entity workflow blocks) in the project root.
- Steps:
  1. Run `shark config validate`
  2. Run `shark status` (project dashboard)
  3. Run `shark status options E20-F02-001` (or any task key) to verify task workflow is loaded
  4. Run `shark epic next-status` dry check -- verify epic workflow statuses match those defined in `.sharkworkflow.json`
- Expected Outcomes:
  - Config validation passes with no errors
  - All commands operate using workflow definitions from `.sharkworkflow.json`
  - No warnings or errors about missing workflow data
- Pass/Fail: PASS if all commands return expected workflow data. FAIL if any command falls back to defaults unexpectedly or reports missing workflow config.

**AS-A02: Custom workflow file path via `workflow_config` key**

- Preconditions: `.sharkconfig.json` contains `"workflow_config": "config/my-workflow.json"` pointing to a workflow file in a subdirectory.
- Steps:
  1. Create `config/my-workflow.json` with all five entity workflow blocks
  2. Set `workflow_config` in `.sharkconfig.json`
  3. Run `shark config validate`
  4. Run `shark status options` for a task, feature, and epic
- Expected Outcomes:
  - System loads workflow from `config/my-workflow.json`, not from `.sharkworkflow.json` or inline
  - All entity workflows resolve from the configured file
- Pass/Fail: PASS if workflow data comes from the custom path. FAIL if system ignores `workflow_config` or falls back to default path.

**AS-A03: Configured workflow file path does not exist -- graceful fallback**

- Preconditions: `.sharkconfig.json` contains `"workflow_config": "nonexistent.json"`. No `.sharkworkflow.json` exists. `.sharkconfig.json` has inline workflow blocks.
- Steps:
  1. Run `shark config validate`
  2. Run `shark status options` for a task
- Expected Outcomes:
  - No error emitted about the missing file
  - System falls back to inline workflow data from `.sharkconfig.json`
  - Commands operate normally
- Pass/Fail: PASS if silent fallback works. FAIL if error is emitted or commands fail.

---

### Area B: Backward-Compatible Fallback (REQ-F-002)

**AS-B01: Project without `.sharkworkflow.json` behaves identically to pre-E20**

- Preconditions: Existing project with only `.sharkconfig.json` (no `.sharkworkflow.json`, no `workflow_config` key). Current production `.sharkconfig.json` with all inline workflow data.
- Steps:
  1. Record baseline: run `shark status options E20-F02-001 --json` and capture output
  2. Upgrade shark binary to E20-enabled version
  3. Run `shark status options E20-F02-001 --json` again
  4. Diff the two outputs
- Expected Outcomes:
  - Zero differences in workflow resolution
  - All status transitions, agent assignments, and orchestrator actions identical
  - No new warnings or errors
- Pass/Fail: PASS if outputs are identical. FAIL if any workflow behavior changes.

**AS-B02: Existing `.sharkconfig.json` with inline workflow blocks remains valid**

- Preconditions: Current production `.sharkconfig.json` with all five entity workflows inline.
- Steps:
  1. Run full test suite (`make test`)
  2. Run `shark config validate`
  3. Execute a complete task lifecycle: create task, advance through statuses, complete
  4. Execute a feature lifecycle: create feature, advance through statuses
  5. Execute an epic lifecycle: advance epic through statuses
- Expected Outcomes:
  - All tests pass
  - Config validation succeeds
  - All lifecycle operations complete without errors
  - Status flow enforcement (advanced profile) works correctly
- Pass/Fail: PASS if all lifecycle operations succeed. FAIL if any status transition fails or produces unexpected behavior.

**AS-B03: No error or warning when workflow file is absent**

- Preconditions: Project with `.sharkconfig.json` only. `workflow_config` key absent.
- Steps:
  1. Run `shark status --json` and capture stderr
  2. Run `shark task list --json` and capture stderr
  3. Run `shark config show` and capture stderr
- Expected Outcomes:
  - Zero output on stderr related to workflow file
  - No deprecation warnings about missing `.sharkworkflow.json`
  - All commands function normally
- Pass/Fail: PASS if stderr is clean of workflow-file-related messages. FAIL if any warning or error about missing workflow file appears.

---

### Area C: Configuration Precedence Chain (REQ-F-003)

**AS-C01: Workflow file takes precedence over `.sharkconfig.json` inline data**

- Preconditions: Both files exist. `.sharkworkflow.json` defines `task_workflow` with a modified status (e.g., custom status "testing_custom" added to task flow). `.sharkconfig.json` has the standard task workflow inline.
- Steps:
  1. Run `shark status options <task-key> --json`
  2. Check if "testing_custom" appears in the available status options
- Expected Outcomes:
  - The custom status from `.sharkworkflow.json` is present
  - The inline `.sharkconfig.json` task workflow is ignored for task status resolution
- Pass/Fail: PASS if workflow file definitions win. FAIL if inline definitions are used instead.

**AS-C02: Per-entity precedence -- partial workflow file**

- Preconditions: `.sharkworkflow.json` defines only `epic_workflow` and `task_workflow`. `.sharkconfig.json` defines all five entity workflows inline.
- Steps:
  1. Run `shark status options <epic-key> --json` -- should use workflow file
  2. Run `shark status options <feature-key> --json` -- should use inline `.sharkconfig.json`
  3. Run `shark status options <task-key> --json` -- should use workflow file
  4. Verify bug and change workflows also use inline `.sharkconfig.json`
- Expected Outcomes:
  - Epic and task: workflow from `.sharkworkflow.json`
  - Feature, bug, change: workflow from `.sharkconfig.json` inline
- Pass/Fail: PASS if per-entity precedence works correctly. FAIL if all-or-nothing precedence is applied.

**AS-C03: Full precedence chain -- workflow file > inline config > built-in defaults**

- Preconditions: `.sharkworkflow.json` defines `epic_workflow` only. `.sharkconfig.json` defines `feature_workflow` only (no other entity workflows). No bug, change, or task workflow in either file.
- Steps:
  1. Run `shark status options <epic-key>` -- from workflow file
  2. Run `shark status options <feature-key>` -- from inline config
  3. Run `shark status options <task-key>` -- from built-in defaults
- Expected Outcomes:
  - Each entity resolves from the correct source in the precedence chain
  - Built-in defaults provide a usable workflow for entities not configured anywhere
- Pass/Fail: PASS if three-tier precedence resolves correctly. FAIL if any tier is skipped.

---

### Area D: Task Workflow Standardization (REQ-F-005)

**AS-D01: `task_workflow` block is parsed identically to other entity blocks**

- Preconditions: `.sharkworkflow.json` with `task_workflow` block containing `version`, `status_flow`, `status_metadata`, `orchestrator_actions`, `special_statuses` sub-keys. Same structure as `epic_workflow`.
- Steps:
  1. Run `shark config validate`
  2. Run `shark status options <task-key> --json`
  3. Compare task status flow against the `task_workflow.status_flow` in the file
  4. Run `shark task next-status <task-key>` and verify the transition matches the defined flow
- Expected Outcomes:
  - Validation passes
  - Status options match the `task_workflow` block exactly
  - Status transitions follow the defined `status_flow`
- Pass/Fail: PASS if task workflow block is fully functional. FAIL if parsing errors or incorrect status resolution.

**AS-D02: Legacy top-level keys still work as fallback**

- Preconditions: `.sharkconfig.json` with legacy top-level `status_flow` and `status_metadata` keys for task workflow. No `task_workflow` block anywhere.
- Steps:
  1. Run `shark status options <task-key> --json`
  2. Verify statuses match the legacy top-level `status_flow`
  3. Run a task through its lifecycle
- Expected Outcomes:
  - Legacy format works identically to pre-E20 behavior
  - No deprecation errors (warnings are acceptable per REQ-F-020)
- Pass/Fail: PASS if legacy path remains functional. FAIL if legacy keys are ignored or produce errors.

**AS-D03: `task_workflow` block takes precedence over legacy top-level keys**

- Preconditions: Both `task_workflow` block AND legacy top-level `status_flow`/`status_metadata` present in `.sharkconfig.json`. The `task_workflow` block has a different flow than the legacy keys.
- Steps:
  1. Run `shark status options <task-key> --json`
  2. Determine which flow is used
- Expected Outcomes:
  - The `task_workflow` block flow is used
  - Legacy top-level keys are ignored
- Pass/Fail: PASS if `task_workflow` block wins. FAIL if legacy keys take precedence.

---

### Area E: Automatic Migration via `shark init update` (REQ-F-006)

**AS-E01: `shark init update --workflow=advanced` generates `.sharkworkflow.json`**

- Preconditions: Project initialized with basic profile. No `.sharkworkflow.json` exists.
- Steps:
  1. Run `shark init update --workflow=advanced`
  2. Verify `.sharkworkflow.json` was created in the project root
  3. Verify the file contains all five `{entity}_workflow` blocks
  4. Verify `task_workflow` uses block format (not legacy top-level keys)
  5. Run `shark config validate`
- Expected Outcomes:
  - File is created with valid JSON
  - All five entity workflows present
  - Task workflow uses `task_workflow` block structure
  - Validation passes
- Pass/Fail: PASS if file is generated correctly. FAIL if any entity workflow is missing or task uses legacy format.

**AS-E02: `shark init update --workflow=basic` generates basic profile**

- Preconditions: Project with advanced profile.
- Steps:
  1. Run `shark init update --workflow=basic`
  2. Verify `.sharkworkflow.json` contains basic profile for all entity types
  3. Run `shark status options <task-key>` and verify basic statuses (todo, in_progress, ready_for_review, completed, blocked)
- Expected Outcomes:
  - Basic profile applied to all entities
  - Status options reflect basic workflow
- Pass/Fail: PASS if basic profile is correctly generated. FAIL if advanced statuses leak into basic profile.

**AS-E03: `shark init update --dry-run` previews without writing**

- Preconditions: No `.sharkworkflow.json` exists.
- Steps:
  1. Run `shark init update --workflow=advanced --dry-run`
  2. Check that `.sharkworkflow.json` was NOT created
  3. Verify the preview output shows what would be generated
- Expected Outcomes:
  - No file written
  - Preview shows workflow file contents
- Pass/Fail: PASS if no file created and preview is shown. FAIL if file is created or no preview output.

**AS-E04: Update in place with backup**

- Preconditions: Existing `.sharkworkflow.json` with basic profile.
- Steps:
  1. Run `shark init update --workflow=advanced`
  2. Check for timestamped backup of the previous `.sharkworkflow.json`
  3. Verify new file has advanced profile
- Expected Outcomes:
  - Backup file exists (e.g., `.sharkworkflow.json.backup.YYYYMMDD-HHMMSS`)
  - New file has advanced profile content
- Pass/Fail: PASS if backup created and update applied. FAIL if no backup or overwrite failure.

---

### Area F: Unified Config Loading Path (REQ-F-007)

**AS-F01: Consumer code is unaware of file source**

- Preconditions: `.sharkworkflow.json` exists with workflow data.
- Steps:
  1. Run `shark task resume <key>` -- exercises template rendering, workflow service, status resolution
  2. Run `shark status advance <key>` -- exercises workflow transition validation
  3. Run `shark status` -- exercises project dashboard, status aggregation
- Expected Outcomes:
  - All consumer paths (workflow.Service, status.CalculationService, template renderer) operate correctly
  - No "file source" awareness leaks into user-facing output
- Pass/Fail: PASS if all commands work transparently. FAIL if any command exposes file-source details or fails.

---

### Area G: Config Validation and Show (REQ-F-010, REQ-F-011) -- Should Have

**AS-G01: `shark config validate` checks workflow file**

- Preconditions: `.sharkworkflow.json` exists with an intentional structural error (e.g., `status_flow` is a string instead of an object).
- Steps:
  1. Run `shark config validate`
- Expected Outcomes:
  - Validation reports the structural error in `.sharkworkflow.json`
  - Error message identifies the file and the offending field
- Pass/Fail: PASS if error is caught and clearly reported. FAIL if validation misses the error or reports wrong file.

**AS-G02: `shark config validate` warns about duplicate definitions**

- Preconditions: `task_workflow` defined in both `.sharkworkflow.json` AND `.sharkconfig.json`.
- Steps:
  1. Run `shark config validate`
- Expected Outcomes:
  - Warning about duplicate task workflow definition across files
  - Indicates which file takes precedence
- Pass/Fail: PASS if duplicate warning is emitted. FAIL if silently accepted.

**AS-G03: `shark config show` indicates workflow source**

- Preconditions: `.sharkworkflow.json` defines epic and task workflows. `.sharkconfig.json` defines feature workflow inline.
- Steps:
  1. Run `shark config show`
  2. Run `shark config show --json`
- Expected Outcomes:
  - Human output shows source per entity (e.g., "epic_workflow: .sharkworkflow.json")
  - JSON output includes `_source` metadata field per workflow block
- Pass/Fail: PASS if source is indicated per entity. FAIL if source is missing or incorrect.

---

### Area H: Deprecation Warnings (REQ-F-020) -- Could Have

**AS-H01: Deprecation warning when legacy keys coexist with `task_workflow` block**

- Preconditions: `.sharkconfig.json` has both legacy top-level `status_flow` and a `task_workflow` block.
- Steps:
  1. Run `shark status` (human output)
  2. Run `shark status --json` (machine output)
- Expected Outcomes:
  - Human output: deprecation warning emitted once, includes migration guidance
  - JSON output: no warning (clean machine-readable output)
- Pass/Fail: PASS if warning appears in human output only. FAIL if warning appears in JSON output or never appears.

---

### Area I: Workflow Export (REQ-F-021) -- Could Have

**AS-I01: `shark admin workflow export` creates workflow file from current config**

- Preconditions: Project with all workflow data inline in `.sharkconfig.json`. No `.sharkworkflow.json`.
- Steps:
  1. Run `shark admin workflow export`
  2. Verify `.sharkworkflow.json` was created
  3. Verify all five entity workflows are present
  4. Verify task workflow is in `task_workflow` block format (converted from legacy)
- Expected Outcomes:
  - File created with correct content
  - Legacy task keys converted to block format
- Pass/Fail: PASS if export and conversion are correct. FAIL if any entity missing or task format not converted.

---

## (2) Success Criteria Validation Plan

The epic defines four success criteria. Here is how each is verified:

### SC-1: "All entity workflows use consistent `{entity}_workflow` block structure in the new file"

**Verification Method**: Structural inspection + functional test
- Parse `.sharkworkflow.json` generated by `shark init update --workflow=advanced`
- Assert all five keys exist: `epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`
- Assert each block contains identical sub-key set: `version`, `status_flow`, `status_metadata`, `orchestrator_actions`, `special_statuses`
- Functional: Run `shark status options` for each entity type and verify status resolution
- **Pass**: All five blocks present with identical structure. **Fail**: Any block missing, structurally different, or uses legacy format.

### SC-2: "`.sharkworkflow.json` is optional -- system works identically without it via fallback"

**Verification Method**: A/B comparison test
- Configuration A: Current production setup (`.sharkconfig.json` only, no workflow file)
- Configuration B: Same setup after upgrading to E20-enabled binary
- Capture `shark status options --json` output for epic, feature, task, bug, and change entity types under both configurations
- Diff all outputs
- **Pass**: Zero differences across all entity types. **Fail**: Any difference in status flow, metadata, or behavior.

### SC-3: "Existing commands, tests, and workflows continue to function without modification"

**Verification Method**: Full regression suite
- Run `make test` against the E20-modified codebase with the UNCHANGED current `.sharkconfig.json` (no `.sharkworkflow.json`)
- All existing tests must pass without modification
- Run manual smoke test of core commands: `shark status`, `shark task list`, `shark feature list`, `shark epic list`, `shark task next-status`
- **Pass**: All tests pass, all smoke tests work. **Fail**: Any test failure or behavioral change.

### SC-4: "`shark init` and `shark init update --workflow=` generate the new file format"

**Verification Method**: Output validation
- Run `shark init update --workflow=advanced` in a clean project
- Verify `.sharkworkflow.json` is created
- Validate JSON structure matches expected schema
- Run `shark init update --workflow=basic` and verify basic profile
- Run `shark init` in a new empty directory and verify it creates appropriate config files
- **Pass**: All init variants produce correct workflow files. **Fail**: Any missing file, incorrect structure, or invalid JSON.

---

## (3) Cross-Epic Integration Test Scenarios

### INT-01: E20 + E21 (Entity Polymorphism) -- Consistent Workflow Structure

**Scenario**: E21 builds generic entity services that load workflow config for any entity type. After E20, the generic code should not need entity-specific branches for task.

- Preconditions: E20 complete. `.sharkworkflow.json` with all five entity workflows.
- Steps:
  1. Verify that `LoadMultiLevelWorkflow()` returns a `MultiLevelWorkflow` where all five levels are non-nil
  2. Verify that `GetWorkflowForLevel(level)` returns identically-structured `*WorkflowConfig` for all levels
  3. Verify there is no task-specific branching in any code path that E21 would generalize
- Expected Outcome: All entity workflows are structurally uniform, enabling generic entity service code.
- Priority: High -- E21 depends on this property.

### INT-02: E20 + E19 (Sprint Management) -- Future Entity Extensibility

**Scenario**: E19 may introduce `sprint_workflow`. The workflow file and loading code should accommodate a sixth entity type without architectural changes.

- Preconditions: E20 complete.
- Steps:
  1. Add a `sprint_workflow` block to `.sharkworkflow.json` (even if sprint entity doesn't exist yet)
  2. Verify the file parses without error (unknown keys are ignored or handled gracefully)
  3. Verify no crash or data corruption from the extra block
- Expected Outcome: Workflow file tolerates additional entity workflow blocks gracefully.
- Priority: Medium -- E19 is in draft, not imminent.

### INT-03: E20 + E15 (Service Layer Refactoring) -- Service Initialization Transparency

**Scenario**: E15 refactors CLI commands to use services. Services initialize with `workflow.Service`, which reads from the config layer. After E20, service initialization must work identically regardless of whether workflow data comes from one file or two.

- Preconditions: E20 complete. Services created via `cli.GetTaskService()`, `cli.GetFeatureService()`, etc.
- Steps:
  1. Create a task via `shark task create` (exercises service initialization)
  2. Advance task via `shark task next-status` (exercises workflow.Service transition validation)
  3. Repeat with `.sharkworkflow.json` present and absent
- Expected Outcome: Identical behavior in both configurations. Service layer is unaware of config source.
- Priority: High -- E15 is in progress.

### INT-04: E20 + Template Rendering (E07-F30, E07-F34) -- Template Directory Resolution

**Scenario**: The `template_directory` setting may move to `.sharkworkflow.json`. Template rendering must find templates regardless of where the setting lives.

- Preconditions: E20 complete. `template_directory` set in `.sharkworkflow.json` (not in `.sharkconfig.json`).
- Steps:
  1. Run `shark task resume <key>` which triggers template rendering
  2. Verify the orchestrator prompt is rendered correctly using templates from the configured directory
  3. Move `template_directory` back to `.sharkconfig.json`, remove from workflow file
  4. Verify templates still render correctly
- Expected Outcome: Template rendering works regardless of which file contains `template_directory`.
- Priority: High -- template rendering is core functionality.

---

## (4) Risk-Based Test Priorities

Based on the research risk assessment and feasibility review findings, tests are prioritized as follows:

### Priority 1 -- CRITICAL (Must pass before any other testing)

These address the highest-probability, highest-impact risks identified in the research and tech feasibility review:

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| AS-B01 (Backward compatibility -- zero diff) | Regression in workflow loading (Medium prob, High impact) | Research Risk #1 |
| AS-B02 (Existing config remains valid) | Regression in workflow loading | Research Risk #1 |
| AS-D01 (task_workflow block parsing) | Task workflow standardization is highest-value change with moderate risk | Research Risk #5, Tech Review REQ-F-005 |
| AS-D02 (Legacy task keys fallback) | Legacy LoadWorkflowConfig() callers | Research Risk #6, Tech Review concern |
| SC-3 (Full regression -- `make test`) | Regression across the board | Research, Tech Review |

### Priority 2 -- HIGH (Must pass before release)

These address the precedence chain and migration tooling:

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| AS-C01 (Workflow file precedence) | Precedence chain correctness | REQ-F-003 |
| AS-C02 (Per-entity precedence) | Partial workflow file handling | REQ-F-003 |
| AS-A01 (Workflow file detection) | Core new functionality | REQ-F-001 |
| AS-E01 (Init update generates file) | Init generates incorrect format (Low prob, Medium impact) | Research Risk #3 |
| AS-F01 (Consumer code transparency) | Abstraction boundary leaks | REQ-NF-003 |
| INT-03 (Service initialization) | Service layer integration | E15 interaction |
| INT-04 (Template directory resolution) | Template dir resolution breaks (Low prob, Medium impact) | Research Risk #5 |

### Priority 3 -- MEDIUM (Should pass before release)

| Scenario | Risk Addressed |
|----------|---------------|
| AS-A02 (Custom workflow path) | Configurable path feature |
| AS-A03 (Missing configured path fallback) | Graceful degradation |
| AS-C03 (Three-tier precedence) | Full precedence chain |
| AS-D03 (Block precedence over legacy) | Task workflow migration |
| AS-E02, AS-E03, AS-E04 (Init variants) | Migration tooling completeness |
| AS-G01, AS-G02, AS-G03 (Validation/show) | Developer experience |
| INT-01 (E21 structural uniformity) | Cross-epic enablement |

### Priority 4 -- LOW (Nice to have)

| Scenario | Risk Addressed |
|----------|---------------|
| AS-H01 (Deprecation warnings) | Could Have requirement |
| AS-I01 (Workflow export) | Could Have requirement |
| INT-02 (Sprint workflow extensibility) | Future epic support |

---

## (5) Backward Compatibility Validation

Since E20 is a config externalization, backward compatibility is the single most important quality attribute. This section consolidates all backward compatibility checks:

### BCV-1: Unchanged Config -- Zero Behavioral Change

**Method**: Before/after comparison
- Capture full workflow state for all five entity types BEFORE E20 changes
- Apply E20 changes (code only, no config file changes)
- Capture full workflow state AFTER E20 changes
- Assert zero differences

**Artifacts to capture per entity type**:
- `shark status options <key> --json` (available transitions)
- `shark config show --json` (resolved config)
- Workflow metadata (colors, phases, progress weights, responsibilities)
- Orchestrator actions per status
- Special status groups

### BCV-2: Fallback Path Coverage

**Method**: Negative testing -- ensure all fallback paths are exercised:

| Scenario | Expected Fallback |
|----------|-------------------|
| No `.sharkworkflow.json`, no `workflow_config` key | Read inline from `.sharkconfig.json` |
| `workflow_config` points to nonexistent file | Read inline from `.sharkconfig.json` |
| `.sharkworkflow.json` exists but is empty JSON (`{}`) | All entities fall back to `.sharkconfig.json` |
| `.sharkworkflow.json` has only `epic_workflow` | Epic from file, others from `.sharkconfig.json` |
| `.sharkworkflow.json` malformed JSON | Fallback to `.sharkconfig.json` (or clear error) |
| Both files absent, no inline config | Built-in defaults for all entities |

### BCV-3: Cache Behavior Under Two-File Model

**Method**: Cache invalidation test (addresses Research Risk #2)
- Load config with `.sharkworkflow.json` present -- verify cached
- Modify `.sharkworkflow.json` (add a status to `task_workflow`)
- Load config again in a new CLI invocation -- verify new data is loaded
- Delete `.sharkworkflow.json` -- verify fallback to `.sharkconfig.json`

*Note*: Since CLI commands are short-lived (one load per process), cache staleness is only a concern within a single process. This test confirms the cache keys correctly account for both file paths.

### BCV-4: Non-Functional Backward Compatibility

**REQ-NF-002 (Performance)**: Benchmark config loading before and after E20 changes.
- Measure: time for `LoadMultiLevelWorkflow()` to complete
- Target: less than 5ms additional latency with two-file path
- Method: Go benchmark test (`BenchmarkLoadMultiLevelWorkflow`) with temp files

---

## Exit Gate Checklist

- [x] Every requirement area has at least one acceptance scenario (Areas A-I cover all REQ-F-001 through REQ-F-021)
- [x] Risk areas from research and feasibility reviews have targeted scenarios (Priority 1 maps to specific research/review risks)
- [x] Plan is actionable for feature-level decomposition into test strategies (scenarios are concrete with preconditions, steps, and pass/fail criteria)
- [x] Backward compatibility has dedicated comprehensive validation section (BCV-1 through BCV-4)
- [x] Cross-epic integration points are covered (INT-01 through INT-04)
- [x] Success criteria from epic.md each have a verification method (SC-1 through SC-4)

---

*Last Updated*: 2026-03-18
*Author*: QA Agent (Claude Opus 4.6, 1M context)
