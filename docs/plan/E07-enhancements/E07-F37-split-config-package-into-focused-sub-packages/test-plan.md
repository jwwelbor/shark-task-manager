# Test Plan: E07-F37 — Split config package into focused sub-packages

**Feature**: E07-F37-split-config-package-into-focused-sub-packages
**Date**: 2026-03-24
**Precedent**: E07-F36 (repository package split — same alias shim pattern)
**Total existing test files**: 15 files, 11,452 LOC (2.3:1 test-to-code ratio)

---

## 1. AC Test Matrix

### AC-001: `make fmt && make lint && make test` passes with zero failures

**Test Case 1.1 — Full build quality gate after Phase 1 (leaf packages)**
- Setup: `validation/` and `template/` sub-packages created; `aliases.go` populated for those symbols
- Input: `make fmt && make lint && make test`
- Expected: Exit 0, zero failures, zero lint warnings
- Edge case: `gofmt` must produce no diff on new sub-package files (check `make fmt` is idempotent)

**Test Case 1.2 — Full build quality gate after Phase 2 (action package)**
- Setup: `action/` sub-package created; `aliases.go` covers all `action/` symbols
- Input: `make fmt && make lint && make test`
- Expected: Exit 0, zero failures
- Edge case: `golangci-lint` must not flag unused imports in `aliases.go` (type aliases count as usage)

**Test Case 1.3 — Full build quality gate after Phase 4 (workflow package)**
- Setup: `workflow/` sub-package created; `aliases.go` complete for all symbols
- Input: `make fmt && make lint && make test`
- Expected: Exit 0, zero failures
- Edge case: Constants re-declared in `aliases.go` must match source values exactly (grep check)

**Test Case 1.4 — Full build quality gate after Phase 5 (final cleanup)**
- Setup: All phases complete, original flat files removed
- Input: `make fmt && make lint && make test`
- Expected: Exit 0, zero failures, no references to deleted files in source

---

### AC-002: Zero import path changes in any file outside `internal/config/`

**Test Case 2.1 — No sub-package imports in consuming files (grep)**
- Setup: After all phases complete
- Input: `grep -r "internal/config/workflow\|internal/config/action\|internal/config/template\|internal/config/validation" --include="*.go" . | grep -v "internal/config/"`
- Expected: Zero matches (no file outside `internal/config/` imports sub-packages directly)
- Edge case: Test files outside `internal/config/` must also show zero matches

**Test Case 2.2 — All 97 consuming files still compile via aliases**
- Setup: After all phases complete
- Input: `go build ./...`
- Expected: Zero compile errors; all 97 external importers resolve symbols via `internal/config` root
- Edge case: Files that use `config.WorkflowConfig` continue working (type alias is transparent to Go type system)

**Test Case 2.3 — Type alias transparency: no type assertion failures**
- Setup: After all phases complete
- Input: Run existing tests in packages that consume `internal/config` (services, CLI, status, repository)
- Expected: All tests pass; no "cannot use X as type Y" errors
- Edge case: Interface satisfaction checks — a value of `workflow.WorkflowConfig` must satisfy `config.WorkflowConfig` (they are the same type via alias)

---

### AC-003: `go test ./internal/config/workflow/...` passes independently

**Test Case 3.1 — workflow sub-package compiles in isolation**
- Setup: `internal/config/workflow/` populated with schema.go, parser.go, validator.go, defaults.go, multilevel.go
- Input: `go build ./internal/config/workflow/`
- Expected: Compiles with no errors; only stdlib imports (encoding/json, fmt, os, sync, strings, log/slog)
- Edge case: No accidental import of `internal/config` root (would create circular dependency)

**Test Case 3.2 — workflow sub-package tests run independently**
- Setup: Test files migrated to `workflow/` (schema_test.go, parser_test.go, validator_test.go, multilevel_test.go, integration_test.go)
- Input: `go test ./internal/config/workflow/...`
- Expected: All tests pass; same pass count as original workflow test files
- Edge case: Tests that relied on unexported symbols from the same package — these move with their production code and must still compile

**Test Case 3.3 — Global cache isolation**
- Setup: `workflowCache`, `workflowCacheLock`, `multiLevelCache` live in `workflow/parser.go`
- Input: Call `workflow.ClearWorkflowCache()` then `workflow.GetWorkflowOrDefault(".")` in tests
- Expected: Cache cleared and reloaded correctly; no data races (run with `-race`)
- Edge case: Parallel test execution must not cause cache corruption (`sync.RWMutex` must be retained)

**Test Case 3.4 — Existing workflow_file_loading_test.go scenarios pass**
- Source file: `internal/config/workflow_file_loading_test.go` (1,252 LOC)
- Input: All test cases in that file, migrated to `workflow/parser_test.go`
- Expected: Same outcomes; no test regressions

---

### AC-004: `go test ./internal/config/action/...` passes independently

**Test Case 4.1 — action sub-package compiles in isolation**
- Setup: `action/orchestrator.go`, `action/service.go`, `action/mock_service.go`
- Input: `go build ./internal/config/action/`
- Expected: Compiles; imports `internal/config/workflow` (for WorkflowConfig) and `internal/config/validation` (for OrchestratorValidationError) but NOT `internal/config` root
- Edge case: `action/` must not import `internal/config` root (aliases only go from root -> sub-packages, not the reverse)

**Test Case 4.2 — action sub-package tests run independently**
- Source: `orchestrator_action_test.go` (1,438 LOC), `orchestrator_action_validation_test.go` (681 LOC), `action_service_test.go` (583 LOC)
- Input: `go test ./internal/config/action/...`
- Expected: All tests pass; same pass count as original files
- Edge case: Tests using `mock_action_service.go` — the mock file must be in the same `action` package

**Test Case 4.3 — ActionService interface remains satisfied**
- Setup: `DefaultActionService` implements `ActionService` in `action/service.go`
- Input: Compile-time interface satisfaction: `var _ ActionService = (*DefaultActionService)(nil)` in service.go
- Expected: Compiles; no "does not implement" errors
- Edge case: Manager.GetActionService() returns `action.ActionService` — aliased back as `config.ActionService` in aliases.go

---

### AC-005: `go test ./internal/config/template/...` passes independently

**Test Case 5.1 — template sub-package compiles in isolation**
- Setup: `template/helpers.go` with package `template`
- Input: `go build ./internal/config/template/`
- Expected: Compiles; imports `internal/models` for entity types; no `internal/config` root import
- Edge case: Package name `template` conflicts with stdlib `text/template` — verify no shadowing issues in imports within the package itself

**Test Case 5.2 — template sub-package tests run independently**
- Source: `template_helpers_test.go` (2,539 LOC — largest test file)
- Input: `go test ./internal/config/template/...`
- Expected: All tests pass; same pass count as original
- Edge case: Test file package declaration — must be `package template` or `package template_test` (external test package); verify after migration

**Test Case 5.3 — Package name collision guard**
- Setup: `internal/config/template/helpers.go` uses `package template`
- Input: Any file that imports both `text/template` and `internal/config/template` must use an import alias
- Expected: If such a file exists, it uses an alias (e.g., `cfgtemplate "...internal/config/template"`); verify `aliases.go` itself uses an alias
- Edge case: Check `aliases.go` import section for `template` alias to avoid conflict with stdlib

---

### AC-006: `go test ./internal/config/validation/...` passes independently

**Test Case 6.1 — validation sub-package compiles in isolation**
- Setup: `validation/error.go` with package `validation`
- Input: `go build ./internal/config/validation/`
- Expected: Compiles; only stdlib imports (fmt, strings); zero internal dependencies
- Edge case: Simplest extraction — primary risk is incorrect package declaration

**Test Case 6.2 — validation types accessible via root alias**
- Setup: `aliases.go` includes `type OrchestratorValidationError = validation.OrchestratorValidationError`
- Input: Existing code using `config.OrchestratorValidationError` must compile unchanged
- Expected: Compiles; `errors.As(err, &config.OrchestratorValidationError{})` works identically
- Edge case: `ValidationError` is already an alias of `OrchestratorValidationError` in the original — the alias chain must resolve correctly after the split

---

### AC-007: `go test ./internal/config/...` passes (root + all sub-packages)

**Test Case 7.1 — Full config package tree passes**
- Setup: All sub-packages created; aliases.go complete
- Input: `go test ./internal/config/...`
- Expected: Tests pass for: `config` (root), `config/workflow`, `config/action`, `config/template`, `config/validation`
- Edge case: Root package tests (`config_test.go`, `manager_test.go`, `integration_test.go`) must still pass; they test root-level behavior through `config.` API

**Test Case 7.2 — Existing integration tests pass**
- Source: `e18_f01_workflow_integration_test.go`, `integration_test.go`
- Input: `go test ./internal/config/... -run Integration`
- Expected: All integration tests pass; migration target is `workflow/integration_test.go`
- Edge case: Integration tests that reference both workflow and action types must work via aliases

**Test Case 7.3 — Race condition detection**
- Setup: All phases complete
- Input: `go test -race ./internal/config/...`
- Expected: No data race warnings; global caches use `sync.RWMutex` correctly
- Edge case: Tests that call `ClearWorkflowCache()` in parallel goroutines

---

### AC-008: Aliases in `internal/config/aliases.go` cover all previously-exported symbols

**Test Case 8.1 — Symbol count verification (workflow sub-package)**
- Setup: After Phase 4 complete
- Input: `grep -c "^type\|^var\|^const\|^func" internal/config/workflow/schema.go internal/config/workflow/parser.go internal/config/workflow/validator.go internal/config/workflow/defaults.go internal/config/workflow/multilevel.go` vs count of re-exports in `aliases.go`
- Expected: All exported symbols from spec Section 3 appear in `aliases.go`
- Symbols to verify: `WorkflowConfig`, `StatusMetadata`, `MultiLevelWorkflow`, `WorkflowValidationError`, `WorkflowValidationFinding`, `StartStatusKey`, `CompleteStatusKey`, `AggregationStatusKey`, `DefaultWorkflowVersion`, `LoadWorkflowConfig`, `ClearWorkflowCache`, `GetWorkflowOrDefault`, `LoadMultiLevelWorkflow`, `LoadMultiLevelWorkflowOrDefault`, `ValidateWorkflow`, `ValidateWorkflowFiles`, `ValidateTransition`, `DefaultWorkflow`, `DefaultEpicWorkflow`, `DefaultFeatureWorkflow`, `DefaultBugWorkflow`, `DefaultChangeCardWorkflow`

**Test Case 8.2 — Symbol count verification (action sub-package)**
- Symbols to verify: `OrchestratorAction`, `PopulatedAction`, `ActionService`, `DefaultActionService`, `ValidationResult`, `InvalidAction`, `StatusNotFoundError`, `ActionSpawnAgent`, `ActionPause`, `ActionWaitForTriage`, `ActionArchive`, `ActionAdvanceStatus`, `ActionCheckOrResume`, `ActionCascade`, `ValidActionTypes`, `NewActionService`, `ValidateAllOrchestratorActions`

**Test Case 8.3 — Symbol count verification (template sub-package)**
- Symbols to verify: `TemplateEnrichmentData`, `TemplateEnrichmentRepository`, `DocumentRepository`, `FeatureRelationshipRepository`, `EpicRelationshipRepository`, `TaskRelationshipRepository`, `EntityPlaceholders`, `TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders`, `BugPlaceholders`, `ChangeCardPlaceholders`, `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`, `EpicPlaceholdersWithRelated`, `ApplyEnrichmentData`, `ParseEpicKeyFromEntityKey`, `ParseFeatureKeyFromTaskKey`

**Test Case 8.4 — Symbol count verification (validation sub-package)**
- Symbols to verify: `OrchestratorValidationError`, `ValidationError` (alias chain)

**Test Case 8.5 — Compile-time completeness check**
- Setup: After all phases, run `go build ./...` from project root
- Input: Any external caller that was previously compiling still compiles
- Expected: Zero "undefined: config.X" errors across entire codebase
- Edge case: Symbols used only in test files outside `internal/config/` must also be covered (check test files in `internal/services/`, `internal/cli/` etc.)

---

## 2. Non-Functional AC Tests

### REQ-NF-001: No measurable performance regression

**Test Case NF-1.1 — Workflow config loading benchmark**
- Setup: Existing benchmark tests in workflow test files
- Input: `go test -bench=BenchmarkLoadWorkflowConfig ./internal/config/workflow/...` (before and after)
- Expected: No regression > 5% in cached path; cold load time unchanged
- Edge case: Global cache still lives in `workflow/` package — cache clearing and reloading must have same performance profile

**Test Case NF-1.2 — Manager.Load performance**
- Setup: `manager_test.go` stays in root config package
- Input: `go test -bench=. ./internal/config/` (before and after)
- Expected: No regression in Manager load time
- Edge case: `manager.go` now imports `action/` sub-package; verify no additional latency from import initialization

### REQ-NF-002: No new external dependencies

**Test Case NF-2.1 — go.mod unchanged**
- Input: `git diff go.mod go.sum` after all phases
- Expected: Zero changes to `go.mod` or `go.sum`

### REQ-NF-003: Quality gate passes after each phase

**Test Case NF-3.1 — Per-phase quality gate**
- Input: `make fmt && make lint && make test` after each of the 5 phases
- Expected: All 5 phase checkpoints produce zero failures
- Edge case: Do not proceed to next phase if quality gate fails

---

## 3. Integration Scenarios

### Scenario I-1: Downstream CLI commands still work end-to-end

**Components**: `internal/cli/commands/` -> `internal/config` root -> `aliases.go` -> sub-packages
**What to verify**: CLI commands using `config.LoadWorkflowConfig`, `config.OrchestratorAction`, `config.ActionService` still resolve correctly
**Verification**: `go build ./cmd/shark/` succeeds; `./bin/shark status options E07-F37` runs without panics
**Trace to epic**: E07 general enhancement — preserving existing CLI behavior

### Scenario I-2: Status service continues to use workflow config

**Components**: `internal/status/` -> `internal/config` (via aliases) -> `config/workflow/`
**What to verify**: `config.WorkflowConfig`, `config.StatusMetadata` resolve correctly via type alias; status calculation logic unchanged
**Verification**: `go test ./internal/status/...` passes after all phases
**Trace to**: AC-002 (zero import path changes), AC-008 (aliases cover all symbols)

### Scenario I-3: ActionService interface contract survives the split

**Components**: `internal/config/manager.go` -> `internal/config/action/service.go`; `internal/config/aliases.go` re-exports `ActionService`
**What to verify**: Code that stores a `config.ActionService` variable and calls it through the interface still compiles and runs correctly
**Verification**: `manager_test.go` passes; `go vet ./...` produces no interface warnings
**Trace to**: AC-002, AC-008

### Scenario I-4: `config/action` sub-package one-way dependency on `config/workflow`

**Components**: `internal/config/action/` imports `internal/config/workflow/`
**What to verify**: No circular dependency introduced; `workflow/` does NOT import `action/`
**Verification**: `go build ./internal/config/workflow/` with no `action` in import graph; use `go list -deps ./internal/config/workflow/` and verify absence of `action` package
**Trace to**: Spec Section 2.5 (dependency graph), REQ-NF-002

### Scenario I-5: `aliases.go` constant re-declaration correctness

**Components**: `internal/config/aliases.go` re-declares string constants from `workflow/` and `action/`
**What to verify**: Re-declared constant values are byte-for-byte identical to source values
**Verification**: Script: `grep -A1 "StartStatusKey\|CompleteStatusKey\|AggregationStatusKey\|DefaultWorkflowVersion\|ActionSpawnAgent\|ActionPause" internal/config/aliases.go internal/config/workflow/schema.go internal/config/action/orchestrator.go` — values must match
**Trace to**: AC-008

---

## 4. Test Infrastructure

### Existing infrastructure to follow (do NOT recreate)

| File | LOC | What it tests | Migration target |
|------|-----|---------------|------------------|
| `internal/config/template_helpers_test.go` | 2,539 | All placeholder functions, enrichment | `config/template/helpers_test.go` |
| `internal/config/orchestrator_action_test.go` | 1,438 | OrchestratorAction struct, constants | `config/action/orchestrator_test.go` |
| `internal/config/workflow_file_loading_test.go` | 1,252 | LoadWorkflowConfig, cache, file loading | `config/workflow/parser_test.go` |
| `internal/config/config_test.go` | 1,018 | Config struct, StatusMetadata API | STAYS in root (config_test.go) |
| `internal/config/workflow_test.go` | 1,002 | WorkflowConfig struct, schema | `config/workflow/schema_test.go` |
| `internal/config/orchestrator_action_validation_test.go` | 681 | ValidateAllOrchestratorActions | `config/action/orchestrator_test.go` (merge) |
| `internal/config/manager_test.go` | 680 | Manager, ActionService integration | STAYS in root (manager_test.go) |
| `internal/config/action_service_test.go` | 583 | DefaultActionService, NewActionService | `config/action/service_test.go` |
| `internal/config/workflow_validation_dx_test.go` | 513 | ValidateWorkflow, ValidateTransition DX | `config/workflow/validator_test.go` |
| `internal/config/workflow_multilevel_test.go` | ? | MultiLevelWorkflow | `config/workflow/multilevel_test.go` |
| `internal/config/workflow_metadata_test.go` | ? | StatusMetadata fields | `config/workflow/schema_test.go` (merge) |
| `internal/config/workflow_walk_test.go` | ? | Workflow directory walking | `config/workflow/parser_test.go` (merge) |
| `internal/config/e18_f01_workflow_integration_test.go` | ? | E18-F01 integration | `config/workflow/integration_test.go` |
| `internal/config/integration_test.go` | ? | Cross-config integration | `config/workflow/integration_test.go` (merge) |
| `internal/config/config_observability_test.go` | ? | Observability config | STAYS in root |

**Pattern**: Follow `internal/repository/aliases.go` for the exact alias syntax (type alias shim proven in E07-F36).

### New test helpers needed

**None.** This is a pure structural migration. All existing test logic moves as-is into sub-package test files. No new test helpers are required.

The only new verification needed is the grep-based symbol coverage check (Test Case 8.1–8.5) which is a shell command, not a Go test file.

### Test file package declarations after migration

Each migrated test file must update its `package` declaration:

| Original package | New package (non-external test) | New package (external test) |
|------------------|---------------------------------|------------------------------|
| `package config` | `package workflow` / `package action` / etc. | `package workflow_test` / etc. |
| `package config_test` | `package workflow_test` / etc. | same |

Verify: run `go vet ./internal/config/...` after each package declaration update.

---

## 5. Exit Gate Checklist

- [x] Every AC in spec.md has at least one test case (AC-001 through AC-008: 8 ACs, 25+ test cases)
- [x] Edge cases identified for each AC (Go type alias transparency, constant re-declaration, package name collision for `template/`, circular dependency prevention)
- [x] Integration scenarios cover cross-component boundaries (5 scenarios: CLI, status, ActionService, one-way deps, constant correctness)
- [x] Test patterns reference existing infrastructure (15 existing test files mapped to migration targets)
- [x] No orphaned tests — all test cases trace to a specific AC or NFR from spec.md
