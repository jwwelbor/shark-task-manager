# E20 Research Report: Shark Templates

## Executive Summary

E20 proposes externalizing workflow and template configuration from `.sharkconfig.json` into a dedicated `.sharkworkflow.json` file and standardizing the task workflow block structure. Codebase analysis confirms the problem is real (1,759-line config file, task workflow uses legacy top-level keys while 4 other entities use `{entity}_workflow` blocks), the solution is technically feasible with moderate effort, and no showstopper risks exist. The existing config loading infrastructure in `internal/config/workflow_parser.go` already supports multi-level workflow parsing, providing a strong foundation. The recommendation is to proceed with implementation.

---

## (1) Market and Competitive Landscape

### Configuration Externalization Patterns in CLI Tools

The approach proposed in E20 follows well-established patterns in the CLI tooling ecosystem:

**Separation of concerns in config files** is a standard practice. Tools that grow beyond a single-purpose config file routinely split into multiple files:
- **ESLint**: Splits `.eslintrc` (linting rules) from `tsconfig.json` (compiler settings)
- **Docker Compose**: Separates `docker-compose.yml` (service definitions) from `.env` (runtime variables)
- **Kubernetes**: Uses separate ConfigMaps for different configuration domains, with fallback chains

**Workflow-as-configuration** is common in CI/CD and project management tools:
- **GitHub Actions**: Workflow definitions live in `.github/workflows/` as separate YAML files
- **Taskfile**: Uses `Taskfile.yml` separate from project config for workflow definitions
- **Makefile**: Workflow targets are always separate from build configuration

**Fallback chains with precedence** are standard:
- **Git**: `.gitconfig` (global) < `.gitconfig` (local) < environment variables
- **npm**: `package.json` defaults < `.npmrc` < environment variables
- **Viper** (used by Shark): Supports config file + environment variable + flag precedence natively

### Relevance to E20 Requirements

The proposed `.sharkworkflow.json` approach aligns with industry norms:
- Single-file JSON workflow config (vs. directory of files) is appropriate for Shark's scope
- Precedence chain (workflow file > inline config > defaults) follows the established pattern
- Optional adoption with backward-compatible fallback is the standard migration approach

No competing pattern or anti-pattern was identified that would argue against the proposed approach.

---

## (2) Feasibility Assessment by Requirement Area

### Must Have Requirements

**REQ-F-001 (Dedicated Workflow Configuration File)** -- FEASIBLE, Low Risk

The config loading infrastructure already supports JSON parsing of workflow blocks. The change requires:
1. Adding a file-check step in `workflow_parser.go`'s `LoadMultiLevelWorkflow()` before reading from `.sharkconfig.json`
2. The JSON structure for workflow blocks (`status_flow`, `status_metadata`, `orchestrator_actions`, `special_statuses`) is already defined in `WorkflowConfig` struct (`workflow_schema.go`)

Estimated complexity: Small (S). The `parseWorkflowSection()` function already handles individual workflow block parsing.

**REQ-F-002 (Backward-Compatible Fallback)** -- FEASIBLE, Low Risk

The current system already handles missing config gracefully:
- `LoadMultiLevelWorkflow()` returns empty `MultiLevelWorkflow{}` when config is missing
- `GetWorkflowForLevel()` falls back to built-in defaults when a level is nil
- The existing fallback chain (config > defaults) only needs one additional layer (workflow file > config > defaults)

No behavioral change for existing users. The current code paths remain the fallback.

**REQ-F-003 (Configuration Precedence Chain)** -- FEASIBLE, Low Risk

Per-entity precedence is straightforward because the existing `MultiLevelWorkflow` struct already holds separate pointers for each entity level. The loading logic can check the workflow file first, then fill in any nil levels from `.sharkconfig.json`. This is a natural extension of the current nullable-pointer pattern.

**REQ-F-004 (Configurable Workflow File Path)** -- FEASIBLE, Trivial

The `Config` struct in `config.go` already supports adding new fields. Adding a `WorkflowConfig *string` field with `omitempty` is a one-line struct change. The `Manager.Load()` method already parses raw JSON and can extract this field.

**REQ-F-005 (Consistent Task Workflow Block)** -- FEASIBLE, Moderate Risk

This is the most impactful change. Currently:
- **Epic, feature, bug, change**: Use `{entity}_workflow` blocks with `version`, `status_flow`, `status_metadata`, `special_statuses` sub-keys, parsed by `parseWorkflowSection()`
- **Task**: Uses legacy top-level keys (`status_flow`, `status_metadata`, `special_statuses`, `status_flow_version`), parsed by `parseTopLevelTaskWorkflow()`

The migration path:
1. Add `task_workflow` block parsing alongside existing entity blocks in `LoadMultiLevelWorkflow()`
2. Give `task_workflow` block precedence over legacy top-level keys
3. Keep `parseTopLevelTaskWorkflow()` as fallback during transition

Risk: The `LoadWorkflowConfig()` function (legacy single-level API) is still called in some code paths. Must ensure it delegates to multi-level loading consistently. Current code already does this via `GetWorkflowOrDefault()` -> `LoadMultiLevelWorkflowOrDefault()`.

**REQ-F-006 (Automatic Migration via `shark init update`)** -- FEASIBLE, Moderate Effort

`internal/cli/commands/init.go` already handles workflow profile generation. The change requires:
1. Writing the generated workflow data to `.sharkworkflow.json` instead of (or in addition to) `.sharkconfig.json`
2. Converting task workflow output from top-level keys to `task_workflow` block format
3. Backup logic already exists (timestamped backups are created by the current init update)

**REQ-F-007 (Unified Config Loading Path)** -- FEASIBLE, Key Architectural Change

This requirement maps directly to modifying `LoadMultiLevelWorkflow()` in `workflow_parser.go`. The function currently:
1. Reads `.sharkconfig.json`
2. Parses entity workflow blocks
3. Parses legacy task workflow from top-level keys

The updated flow would be:
1. Read `.sharkconfig.json` for runtime settings + `workflow_config` key
2. If workflow file path exists, load and parse it
3. For any entity level not found in workflow file, fall back to `.sharkconfig.json` inline data
4. For task workflow, check `task_workflow` block before legacy top-level keys

The consumer interface (`MultiLevelWorkflow.GetWorkflowForLevel()`) remains unchanged. REQ-NF-003 (single code path) is achievable because only `workflow_parser.go` needs file-source awareness.

### Should Have Requirements

**REQ-F-010 (Config Validation)** -- FEASIBLE, Low Effort. The existing `workflow_validator.go` provides validation logic that can be applied to either file source.

**REQ-F-011 (Config Show Source)** -- FEASIBLE, Low Effort. Adding a `_source` metadata field is a display-layer change.

### Could Have Requirements

**REQ-F-020 (Deprecation Warnings)** -- FEASIBLE, Trivial. A single check in the loading path.

**REQ-F-021 (Workflow Export Command)** -- FEASIBLE, Low Effort. Reads existing config, restructures task data, writes new file.

### Non-Functional Requirements

**REQ-NF-001 (Zero Breaking Changes)** -- ACHIEVABLE. The fallback design ensures existing configs work without modification.

**REQ-NF-002 (Performance <5ms overhead)** -- ACHIEVABLE. JSON parsing a ~1,500-line file takes ~1-2ms on modern hardware. The caching mechanism already in place (`multiLevelCache`) means the overhead is only on first invocation.

**REQ-NF-003 (Single Code Path)** -- ACHIEVABLE. Only `workflow_parser.go` needs file-source logic. The `MultiLevelWorkflow` struct serves as the abstraction boundary.

**REQ-NF-004 (Test Isolation)** -- ACHIEVABLE. Existing workflow tests in `workflow_parser.go`, `workflow_multilevel_test.go`, and `workflow_test.go` already use temp files and in-memory fixtures.

---

## (3) System-Wide Impact: Interactions with Other Epics

### E16 -- Multi-Level Workflow (Completed, Direct Dependency)

E16 established the per-entity workflow block pattern (`epic_workflow`, `feature_workflow`, `bug_workflow`, `change_workflow`) and the `MultiLevelWorkflow` struct. E20 extends this by:
- Adding `task_workflow` block support (standardization)
- Moving all workflow blocks to an external file (separation)

E20 is a natural successor to E16. No conflicts. The `parseWorkflowSection()` function created by E16 is reused directly.

### E21 -- Entity Polymorphism and Duplication Reduction (Draft, Interaction)

E21 proposes a shared `Entity` interface and generic `EntityService`. This interacts with E20 in two ways:
1. **Positive**: E20's consistent `{entity}_workflow` block structure makes it easier for E21 to write generic workflow-loading code. After E20, all entities follow the same pattern, eliminating the task special case that would otherwise need handling in generic code.
2. **Ordering**: E20 should ideally complete before E21 begins service-layer generalization, so E21 can assume consistent workflow structure.

No conflicts. E20 simplifies E21's work.

### E19 -- Sprint Management (Draft, No Direct Interaction)

E19 adds sprints as a new entity type. If E19 needs sprint-level workflow (e.g., `sprint_workflow`), E20's externalized workflow file provides the natural home for it. No conflicts. E20 is supportive infrastructure.

### E15 -- Service Layer Architecture Refactoring (In Progress, Minor Interaction)

E15 is migrating CLI commands to use the service layer. E20 changes config loading, which is consumed by services via `workflow.Service`. The `workflow.Service` initialization already accepts workflow data from the config layer transparently, so E15's service refactoring is compatible with E20's config changes.

### E11 -- Configurable Status Workflow System (Completed, Foundation)

E11 established the config-driven workflow pattern. E20 preserves E11's design while separating the physical storage. No regressions.

### E07-F30, E07-F34 -- Template Engine and Enrichment (Completed, Foundation)

E07-F30 built the Go template rendering engine (`internal/templates/`). E07-F34 added database-sourced template variables. E20 moves the `template_directory` setting to the workflow file, giving templates a dedicated configuration surface. The `OrchestratorRenderer` in `internal/templates/orchestrator_renderer.go` already uses `SetConfiguredTemplateDir()` for directory configuration -- this indirection means the renderer does not need to know where the setting came from.

---

## (4) Existing Capability Overlap with Defined Scope

### Already Implemented (No Work Needed)

| Capability | Location | Notes |
|---|---|---|
| Per-entity workflow block parsing (`epic_workflow`, `feature_workflow`, `bug_workflow`, `change_workflow`) | `workflow_parser.go` `parseWorkflowSection()` | Complete for 4 of 5 entity types |
| Multi-level workflow struct | `workflow_multilevel.go` `MultiLevelWorkflow` | Holds all 5 levels with nil-means-default semantics |
| Default workflows per entity | `workflow_default.go` | Built-in defaults for all 5 entity types |
| Workflow caching with double-check locking | `workflow_parser.go` | Thread-safe, path-aware cache |
| Template directory configuration | `config.go` `GetTemplateDirectory()` | Already reads from `Config` struct |
| Template directory discovery | `orchestrator_renderer.go` `findTemplateDir()` | Walks up from working directory |
| Workflow validation | `workflow_validator.go` | Validates structure of workflow blocks |
| `shark init update --workflow=` | `commands/init.go` | Generates workflow profiles |

### Partially Implemented (Extension Needed)

| Capability | Current State | E20 Extension |
|---|---|---|
| Task workflow parsing | `parseTopLevelTaskWorkflow()` reads legacy top-level keys | Add `task_workflow` block parsing with fallback to legacy |
| Config loading sequence | Reads only `.sharkconfig.json` | Add workflow file check before inline config |
| `Config` struct | Has `TemplateDirectory` field | Add `WorkflowConfig *string` field for file path |
| Init update command | Writes all config to `.sharkconfig.json` | Write workflow data to `.sharkworkflow.json` instead |

### Not Implemented (New Work)

| Capability | Scope |
|---|---|
| `.sharkworkflow.json` file detection and loading | New file I/O in `workflow_parser.go` |
| Per-entity precedence resolution (workflow file vs inline) | New logic in `LoadMultiLevelWorkflow()` |
| `shark config validate` for workflow file | Extension to existing validation command |
| `shark config show` workflow source indicator | Extension to existing show command |
| Deprecation warnings for legacy task keys | New check in loading path |
| `shark admin workflow export` command | New CLI command |

### Estimated New Code

Based on the analysis of existing patterns:
- **Core config loading changes**: ~100-150 lines in `workflow_parser.go`
- **Config struct changes**: ~10 lines in `config.go` and `manager.go`
- **Init update changes**: ~50-80 lines in `commands/init.go`
- **Validation extensions**: ~30-50 lines
- **Tests**: ~200-300 lines (following existing test patterns in `workflow_multilevel_test.go`)
- **Total**: ~400-600 lines of new/modified code

---

## (5) Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| **Regression in workflow loading** -- Changing the config loading path could break existing workflow resolution for any entity type | Medium | High | Extensive existing test suite in `workflow_multilevel_test.go` and `workflow_test.go` covers all entity levels. Add integration test that loads real `.sharkconfig.json` before and after changes, asserting identical workflow resolution. |
| **Cache invalidation bug** -- The caching layer (`multiLevelCache`, `workflowCache`) currently keys on file path. With two files, stale cache from one file could mask changes in the other. | Medium | Medium | Clear both caches when either file is loaded. Consider keying cache on hash of both file contents, or simply invalidating cache when workflow file path differs. The current `ClearWorkflowCache()` function already clears both caches. |
| **Init update generates incorrect format** -- `shark init update --workflow=advanced` must produce the new file format correctly for all 5 entity types | Low | Medium | Table-driven tests comparing generated output against known-good fixtures. The existing init update logic already generates per-entity blocks; the change is primarily where the output is written. |
| **Turso/cloud users affected** -- Cloud database users may have different config loading paths | Low | Low | The config loading is backend-agnostic. `.sharkconfig.json` is read locally regardless of database backend. No Turso-specific config loading exists. |
| **Template directory resolution breaks** -- Moving `template_directory` to the workflow file could break template discovery if the workflow file is missing and the setting was in `.sharkconfig.json` | Low | Medium | The fallback chain (workflow file > `.sharkconfig.json` > default) ensures the setting is found in either location. The `GetTemplateDirectory()` method on `Config` struct continues to work for inline configs. |
| **Legacy `LoadWorkflowConfig()` callers** -- Some code paths may still call the legacy single-level API directly | Low | Medium | Grep confirms `LoadWorkflowConfig()` is primarily called via `GetWorkflowOrDefault()`, which already delegates to the multi-level loader. Audit all call sites during implementation. |

No showstopper risks identified. All risks have clear mitigations.

---

## (6) Recommendations

### Proceed with Implementation

The research confirms E20 is well-scoped, technically feasible, and addresses a real codebase problem. The existing infrastructure handles most of the heavy lifting.

### Implementation Order

1. **Start with REQ-F-005 (task workflow standardization)** -- This is the highest-value change and the one with moderate risk. Adding `task_workflow` block support to `LoadMultiLevelWorkflow()` alongside the existing legacy fallback establishes the consistent structure before introducing file separation.

2. **Then REQ-F-001 + REQ-F-003 + REQ-F-004 (workflow file with precedence)** -- Modify `LoadMultiLevelWorkflow()` to check for an external file first. This builds on step 1 because the task workflow block is now consistent with other entities.

3. **Then REQ-F-002 (backward compatibility verification)** -- Run the full test suite against both configurations (with and without `.sharkworkflow.json`).

4. **Then REQ-F-006 + REQ-F-007 (init update and unified loading)** -- Update the init command to generate the new file.

5. **Finally, Should Have and Could Have** -- Validation extensions, deprecation warnings, and export command.

### Ordering Relative to E21

E20 should complete before E21 (Entity Polymorphism) begins service-layer changes. E20's consistent `{entity}_workflow` structure removes a special case that would complicate E21's generic entity service design. However, E20 and E21 touch different layers (E20: config loading; E21: models/services/repositories), so parallel work on non-overlapping components is possible.

### Testing Strategy

- Extend `workflow_multilevel_test.go` with two-file scenarios
- Add integration test that loads the real `.sharkconfig.json` and verifies workflow resolution matches pre-E20 behavior
- Use `internal/config/test_config_integration.go` patterns for fixture-based testing
- Performance benchmark: measure config loading time before and after (target: <5ms additional)

---

*Research completed: 2026-03-18*
*Researcher: Claude Opus 4.6 (1M context)*
