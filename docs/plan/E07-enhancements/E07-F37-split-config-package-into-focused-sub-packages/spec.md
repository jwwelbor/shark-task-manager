# E07-F37: Split config package into focused sub-packages

**Feature Key**: E07-F37
**Complexity**: STANDARD (11/27)
**Precedent**: E07-F36 (repository package split -- proven type alias shim pattern)

---

## 1. Requirements

### 1.1 Functional Requirements

| ID | Requirement | Traces To |
|----|-------------|-----------|
| REQ-F-001 | Split `internal/config` into 4 sub-packages: `workflow/`, `action/`, `template/`, `validation/` | Feature goal |
| REQ-F-002 | Create `aliases.go` in `internal/config/` that re-exports all moved types, functions, and variables via type aliases and var assignments | Feature goal (zero-breakage) |
| REQ-F-003 | All 97 files importing `internal/config` must compile without any import changes | AC Scenario 2 |
| REQ-F-004 | Each sub-package must be independently testable via `go test ./internal/config/<sub>/...` | AC Scenario 3 |
| REQ-F-005 | Root `internal/config/` retains: `Config` struct, `Manager`, `ObservabilityConfig`, core I/O (config.go, manager.go) | Feature target structure |
| REQ-F-006 | Move existing test files alongside their production code in each sub-package | Feature goal (test isolation) |

### 1.2 Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| REQ-NF-001 | No measurable performance regression (workflow caching, Manager.Load) |
| REQ-NF-002 | No new external dependencies introduced |
| REQ-NF-003 | `make fmt && make lint && make test` passes after each phase |

### 1.3 Acceptance Criteria

| ID | Criterion | Testable? |
|----|-----------|-----------|
| AC-001 | `make fmt && make lint && make test` passes with zero failures | Yes: CI |
| AC-002 | Zero import path changes in any file outside `internal/config/` | Yes: grep for sub-package imports in non-config code |
| AC-003 | `go test ./internal/config/workflow/...` passes independently | Yes: direct execution |
| AC-004 | `go test ./internal/config/action/...` passes independently | Yes: direct execution |
| AC-005 | `go test ./internal/config/template/...` passes independently | Yes: direct execution |
| AC-006 | `go test ./internal/config/validation/...` passes independently | Yes: direct execution |
| AC-007 | `go test ./internal/config/...` passes (root + all sub-packages) | Yes: direct execution |
| AC-008 | Aliases in `internal/config/aliases.go` cover all previously-exported symbols | Yes: compile-time verification |

### 1.4 Out of Scope

1. Changing the public API (all 52+ exports preserved via aliases)
2. Updating imports in consuming packages (they use aliases)
3. Adding new functionality
4. Config file format changes to `.sharkconfig.json`
5. Changing global workflow cache behavior

---

## 2. Architecture

### 2.1 Target Directory Structure

```
internal/config/
├── config.go                          # STAYS: Config, ObservabilityConfig, GetTemplateDirectoryFromConfig
├── manager.go                         # STAYS: Manager (Config file I/O)
├── aliases.go                         # NEW: backward-compat re-exports
│
├── workflow/                          # NEW sub-package
│   ├── schema.go                      # FROM workflow_schema.go: WorkflowConfig, StatusMetadata, constants
│   ├── parser.go                      # FROM workflow_parser.go: LoadWorkflowConfig, cache, multi-level loading
│   ├── validator.go                   # FROM workflow_validator.go: ValidateWorkflow, ValidateTransition, findings
│   ├── defaults.go                    # FROM workflow_default.go: DefaultWorkflow, DefaultEpicWorkflow, etc.
│   ├── multilevel.go                  # FROM workflow_multilevel.go: MultiLevelWorkflow
│   ├── schema_test.go                 # FROM workflow_test.go, workflow_metadata_test.go
│   ├── parser_test.go                 # FROM workflow_file_loading_test.go, workflow_walk_test.go
│   ├── validator_test.go              # FROM workflow_validation_dx_test.go
│   ├── multilevel_test.go             # FROM workflow_multilevel_test.go
│   └── integration_test.go            # FROM e18_f01_workflow_integration_test.go
│
├── action/                            # NEW sub-package
│   ├── orchestrator.go                # FROM orchestrator_action.go: OrchestratorAction, PopulatedAction, constants
│   ├── service.go                     # FROM action_service.go: ActionService interface, DefaultActionService
│   ├── orchestrator_test.go           # FROM orchestrator_action_test.go, orchestrator_action_validation_test.go
│   ├── service_test.go                # FROM action_service_test.go
│   └── mock_service.go               # FROM mock_action_service.go
│
├── template/                          # NEW sub-package
│   ├── helpers.go                     # FROM template_helpers.go: all placeholder functions, enrichment types
│   └── helpers_test.go               # FROM template_helpers_test.go
│
└── validation/                        # NEW sub-package
    └── error.go                       # FROM validation_error.go: OrchestratorValidationError, ValidationError alias
```

### 2.2 Files Modified/Created

**New files created:**

| File | Source | Content |
|------|--------|---------|
| `internal/config/aliases.go` | New | Type aliases + var re-exports for all moved symbols |
| `internal/config/workflow/schema.go` | `workflow_schema.go` | `WorkflowConfig`, `StatusMetadata`, constants |
| `internal/config/workflow/parser.go` | `workflow_parser.go` | `LoadWorkflowConfig`, `LoadMultiLevelWorkflow`, cache globals |
| `internal/config/workflow/validator.go` | `workflow_validator.go` | `ValidateWorkflow`, `ValidateTransition`, findings |
| `internal/config/workflow/defaults.go` | `workflow_default.go` | `DefaultWorkflow`, `DefaultEpicWorkflow`, etc. |
| `internal/config/workflow/multilevel.go` | `workflow_multilevel.go` | `MultiLevelWorkflow` struct |
| `internal/config/action/orchestrator.go` | `orchestrator_action.go` | `OrchestratorAction`, `PopulatedAction`, action constants |
| `internal/config/action/service.go` | `action_service.go` | `ActionService` interface, `DefaultActionService` |
| `internal/config/action/mock_service.go` | `mock_action_service.go` | Mock implementation |
| `internal/config/template/helpers.go` | `template_helpers.go` | All placeholder functions, enrichment types/interfaces |
| `internal/config/validation/error.go` | `validation_error.go` | `OrchestratorValidationError` |

**Files removed (after aliases are verified):**

| File | Moved To |
|------|----------|
| `internal/config/workflow_schema.go` | `workflow/schema.go` |
| `internal/config/workflow_parser.go` | `workflow/parser.go` |
| `internal/config/workflow_validator.go` | `workflow/validator.go` |
| `internal/config/workflow_default.go` | `workflow/defaults.go` |
| `internal/config/workflow_multilevel.go` | `workflow/multilevel.go` |
| `internal/config/orchestrator_action.go` | `action/orchestrator.go` |
| `internal/config/action_service.go` | `action/service.go` |
| `internal/config/mock_action_service.go` | `action/mock_service.go` |
| `internal/config/template_helpers.go` | `template/helpers.go` |
| `internal/config/validation_error.go` | `validation/error.go` |

**Files staying in root `internal/config/`:**

| File | Reason |
|------|--------|
| `config.go` | Core `Config` struct, `ObservabilityConfig` -- root identity of the package |
| `manager.go` | `Manager` for config file I/O -- depends on `Config` + `ActionService` (interface) |

**Test files migrated with their production code** -- each sub-package gets its own `*_test.go` files.

**Files NOT modified (zero-breakage guarantee):**

All 97 files outside `internal/config/` that import `internal/config` remain unchanged. They continue using `config.WorkflowConfig`, `config.LoadWorkflowConfig`, etc. via the aliases.

### 2.3 Data Model Changes

None. This is a pure code reorganization. No database schema, migration, or config file format changes.

### 2.4 Key Technical Decisions

**Decision 1: Type alias shim (aliases.go) for backward compatibility**

- **Rationale**: Follows the proven pattern from E07-F36 (repository split). See `internal/repository/aliases.go` for the exact pattern. Type aliases (`type X = subpkg.X`) are transparent to Go's type system -- callers see no difference.
- **Alternative considered**: Update all 97 import sites. Rejected: high risk, massive diff, no incremental value.

**Decision 2: `workflow/` sub-package is a single unit (schema + parser + validator + defaults + multilevel)**

- **Rationale**: These files have tight three-way coupling. `workflow_parser.go` returns `*WorkflowConfig` (from schema), calls `ValidateWorkflow` (from validator), and references `MultiLevelWorkflow` (from multilevel). Splitting them across packages would require circular dependency resolution via interfaces, adding complexity with no benefit. Grouping them in one sub-package preserves the existing coupling within a clear boundary.
- **Alternative considered**: Separate `workflow/schema/` and `workflow/parser/` sub-sub-packages. Rejected: would require interfaces to break circular dependency between schema types and parser functions, violating the "Simple" principle.

**Decision 3: `action/` depends on `workflow/` via interface, not concrete types**

- **Rationale**: `action/service.go` (`DefaultActionService.Reload`) calls `GetWorkflowOrDefault()` which returns `*WorkflowConfig`. After the split, `action/` imports `workflow/` for the `WorkflowConfig` type. This is a one-way dependency (action -> workflow), no circularity. The `ActionService` interface itself remains in `action/` and `Manager` references it via the interface.

**Decision 4: `template/helpers.go` depends on `models` only (external) + `action/` for `OrchestratorAction` (internal)**

- **Rationale**: `template_helpers.go` imports `internal/models` for `Entity`, `Task`, `Feature`, `Epic`, `Bug`, `ChangeCard`, and `Document` types. It also references `OrchestratorAction` indirectly via `StatusMetadata.OrchestratorAction`. After the split, `template/` imports `models` directly (unchanged) and does NOT need to import `action/` because template helpers work with placeholder maps (`map[string]string`), not `OrchestratorAction` directly. The `TemplateEnrichmentData` and related interfaces/types live in `template/` since they are template-specific.

**Decision 5: `validation/error.go` has zero internal dependencies**

- **Rationale**: `OrchestratorValidationError` only imports `fmt` and `strings` from stdlib. It is the perfect leaf-node extraction target with zero risk. Follows feature.md Phase 1 guidance.

**Decision 6: `config.go` references `StatusMetadata` -- resolved via import**

- **Rationale**: `Config` struct has a field `statusMetadata map[string]*StatusMetadata`. After the split, `StatusMetadata` lives in `workflow/`. The root `config.go` imports `workflow/` for this type. This is a standard parent-to-child dependency (root config imports its own sub-package). The `Config.GetStatusMetadata()` and `Config.SetStatusMetadata()` methods use `*workflow.StatusMetadata` directly, aliased back via `aliases.go` for external callers.

**Decision 7: `Manager` references `ActionService` -- resolved via interface in `action/`**

- **Rationale**: `Manager.GetActionService()` returns `ActionService` (interface) and calls `NewActionService(configPath)`. After the split, `ActionService` and `NewActionService` live in `action/`. `Manager` imports `action/` for this. One-way dependency, no circularity.

### 2.5 Dependency Graph (Post-Split)

```
internal/config/ (root)
├── config.go    imports: internal/db, internal/config/workflow (for StatusMetadata)
├── manager.go   imports: internal/config/action (for ActionService, NewActionService)
├── aliases.go   imports: workflow/, action/, template/, validation/
│
├── workflow/    imports: (stdlib only: encoding/json, fmt, os, sync, strings, log/slog)
│   ├── schema.go
│   ├── parser.go     uses: schema types, validator functions, multilevel types
│   ├── validator.go  uses: schema types
│   ├── defaults.go   uses: schema types
│   └── multilevel.go uses: schema types
│
├── action/      imports: internal/config/workflow (for WorkflowConfig, GetWorkflowOrDefault)
│   │                     internal/config/validation (for OrchestratorValidationError)
│   │                     internal/templates (for OrchestratorEngine)
│   ├── orchestrator.go
│   └── service.go
│
├── template/    imports: internal/models (for Entity, Task, Feature, etc.)
│   └── helpers.go
│
└── validation/  imports: (stdlib only: fmt, strings)
    └── error.go
```

**Dependency direction**: `root -> workflow, action, template, validation` (via aliases.go). Sub-packages may depend on each other one-way: `action -> workflow`, `action -> validation`. No circular dependencies.

### 2.6 Alias Strategy (aliases.go)

Following the pattern established in `internal/repository/aliases.go`:

```go
package config

import (
    "github.com/jwwelbor/shark-task-manager/internal/config/action"
    "github.com/jwwelbor/shark-task-manager/internal/config/template"
    "github.com/jwwelbor/shark-task-manager/internal/config/validation"
    "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
)

// --- workflow/ types ---
type WorkflowConfig = workflow.WorkflowConfig
type StatusMetadata = workflow.StatusMetadata
type MultiLevelWorkflow = workflow.MultiLevelWorkflow
type WorkflowValidationError = workflow.WorkflowValidationError
type WorkflowValidationFinding = workflow.WorkflowValidationFinding

// --- workflow/ constants ---
const StartStatusKey = workflow.StartStatusKey         // Cannot alias const; re-declare
const CompleteStatusKey = workflow.CompleteStatusKey
const AggregationStatusKey = workflow.AggregationStatusKey
const DefaultWorkflowVersion = workflow.DefaultWorkflowVersion

// --- workflow/ functions ---
var LoadWorkflowConfig = workflow.LoadWorkflowConfig
var ClearWorkflowCache = workflow.ClearWorkflowCache
var GetWorkflowOrDefault = workflow.GetWorkflowOrDefault
var LoadMultiLevelWorkflow = workflow.LoadMultiLevelWorkflow
var LoadMultiLevelWorkflowOrDefault = workflow.LoadMultiLevelWorkflowOrDefault
var ValidateWorkflow = workflow.ValidateWorkflow
var ValidateWorkflowFiles = workflow.ValidateWorkflowFiles
var ValidateTransition = workflow.ValidateTransition
var DefaultWorkflow = workflow.DefaultWorkflow
var DefaultEpicWorkflow = workflow.DefaultEpicWorkflow
var DefaultFeatureWorkflow = workflow.DefaultFeatureWorkflow
var DefaultBugWorkflow = workflow.DefaultBugWorkflow
var DefaultChangeCardWorkflow = workflow.DefaultChangeCardWorkflow

// --- action/ types ---
type OrchestratorAction = action.OrchestratorAction
type PopulatedAction = action.PopulatedAction
type ActionService = action.ActionService
type DefaultActionService = action.DefaultActionService
type ValidationResult = action.ValidationResult
type InvalidAction = action.InvalidAction
type StatusNotFoundError = action.StatusNotFoundError

// --- action/ constants ---
const ActionSpawnAgent = action.ActionSpawnAgent
const ActionPause = action.ActionPause
// ... (all action constants)

// --- action/ functions ---
var NewActionService = action.NewActionService
var ValidateAllOrchestratorActions = action.ValidateAllOrchestratorActions
var ValidActionTypes = action.ValidActionTypes  // var slice, re-exported

// --- template/ types & functions ---
type TemplateEnrichmentData = template.TemplateEnrichmentData
type TemplateEnrichmentRepository = template.TemplateEnrichmentRepository
type DocumentRepository = template.DocumentRepository
type FeatureRelationshipRepository = template.FeatureRelationshipRepository
type EpicRelationshipRepository = template.EpicRelationshipRepository
type TaskRelationshipRepository = template.TaskRelationshipRepository

var EntityPlaceholders = template.EntityPlaceholders
var TaskPlaceholders = template.TaskPlaceholders
var FeaturePlaceholders = template.FeaturePlaceholders
var EpicPlaceholders = template.EpicPlaceholders
var BugPlaceholders = template.BugPlaceholders
var ChangeCardPlaceholders = template.ChangeCardPlaceholders
var TaskPlaceholdersWithRelated = template.TaskPlaceholdersWithRelated
var FeaturePlaceholdersWithRelated = template.FeaturePlaceholdersWithRelated
var EpicPlaceholdersWithRelated = template.EpicPlaceholdersWithRelated
var ApplyEnrichmentData = template.ApplyEnrichmentData
var ParseEpicKeyFromEntityKey = template.ParseEpicKeyFromEntityKey
var ParseFeatureKeyFromTaskKey = template.ParseFeatureKeyFromTaskKey

// --- validation/ types ---
type OrchestratorValidationError = validation.OrchestratorValidationError
// Note: ValidationError is already an alias of OrchestratorValidationError in the original code
```

**Note on constants**: Go does not allow `const X = pkg.X` for string constants. These must be re-declared with the same value, or the original `const` block stays in `aliases.go`. The preferred approach (following Go convention) is to keep constants in the sub-package and re-declare them in aliases.go with identical values, adding a comment pointing to the canonical definition.

### 2.7 Cross-Cutting Concern: `config.go` Field Types

After the split, `Config` struct in `config.go` references `StatusMetadata` (now in `workflow/`). Resolution:

```go
// config.go
package config

import (
    "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
    "github.com/jwwelbor/shark-task-manager/internal/db"
)

type Config struct {
    // ...existing fields...
    statusMetadata map[string]*workflow.StatusMetadata `json:"-"`
}

func (c *Config) GetStatusMetadata(status string) *workflow.StatusMetadata { ... }
func (c *Config) SetStatusMetadata(metadata map[string]*workflow.StatusMetadata) { ... }
```

External callers using `config.StatusMetadata` continue to work via the type alias in `aliases.go`.

### 2.8 Cross-Cutting Concern: `manager.go` ActionService Dependency

After the split, `Manager.GetActionService()` references `ActionService` (now in `action/`) and calls `NewActionService`. Resolution:

```go
// manager.go
package config

import (
    "github.com/jwwelbor/shark-task-manager/internal/config/action"
)

type Manager struct {
    configPath    string
    config        *Config
    actionService action.ActionService
}

func (m *Manager) GetActionService() (action.ActionService, error) {
    if m.actionService == nil {
        service, err := action.NewActionService(m.configPath)
        // ...
    }
    return m.actionService, nil
}
```

External callers using `config.ActionService` (the interface) continue to work via the type alias.

### 2.9 Cross-Cutting Concern: Global Workflow Cache

`workflow_parser.go` has package-level `var` globals for caching (`workflowCache`, `multiLevelCache` with `sync.RWMutex`). These move to `workflow/parser.go` and remain package-level globals within the `workflow` sub-package. The `ClearWorkflowCache()` function moves with them and is aliased back.

### 2.10 Implementation Phases

**Phase 1 (Task 001): Extract leaf-node packages -- `validation/` and `template/`**

These have minimal internal coupling:
- `validation/error.go`: zero internal dependencies (only stdlib)
- `template/helpers.go`: depends on `internal/models` (external), no config-internal deps

Steps:
1. Create `internal/config/validation/` directory
2. Move `validation_error.go` -> `validation/error.go`, change package to `validation`
3. Create `internal/config/template/` directory
4. Move `template_helpers.go` -> `template/helpers.go`, change package to `template`
5. Move corresponding test files
6. Add aliases for moved symbols to `aliases.go`
7. Remove original files
8. Run `make fmt && make lint && make test`

**Phase 2 (Task 002): Extract `action/` sub-package**

Dependencies: `action/` imports `workflow/` types (WorkflowConfig, StatusMetadata) and `validation/` types (OrchestratorValidationError). Since `workflow/` is not yet extracted, `action/` can temporarily import the root `config` package for workflow types -- OR we do Phase 3 first.

Recommended order change: Extract `workflow/` before `action/` to avoid temporary circular dependency.

Revised sequence:
1. Create `internal/config/action/` directory
2. Move `orchestrator_action.go` -> `action/orchestrator.go`, change package
3. Move `action_service.go` -> `action/service.go`, change package
4. Move `mock_action_service.go` -> `action/mock_service.go`, change package
5. Update internal imports to reference `workflow.WorkflowConfig` etc.
6. Move test files
7. Add aliases
8. Remove originals
9. Quality gate

**Phase 3 (Task 003): Coupling analysis for workflow files**

Analyze the three-way coupling between `workflow_schema.go`, `workflow_parser.go`, and `workflow_validator.go`. Since they all go into the same `workflow/` sub-package, no interface decoupling is needed. This phase becomes a design verification checkpoint rather than active work.

**Phase 4 (Task 004): Extract `workflow/` sub-package (core split)**

This is the largest extraction (1,521 LOC across 5 files):
1. Create `internal/config/workflow/` directory
2. Move all 5 workflow files, change package to `workflow`
3. Move all workflow test files
4. Update `config.go` and `manager.go` to import `workflow/` and `action/`
5. Complete `aliases.go` with all remaining symbols
6. Remove original files
7. Quality gate

**Phase 5 (Task 005): Cleanup and verification**

1. Verify no direct sub-package imports leaked into non-config code
2. Remove `template_helpers.go.backup` (already exists in repo)
3. Remove integration test skip files if applicable
4. Final comprehensive test run
5. Update any internal documentation if needed

### 2.11 Risk Mitigations

| Risk | Mitigation |
|------|------------|
| Circular dependency between root and sub-packages | Dependency graph analysis (Section 2.5) confirms one-way deps only |
| Constants cannot be aliased in Go | Re-declare with identical values + comment referencing canonical source |
| Global cache state in workflow parser | Cache globals move as-is to `workflow/` sub-package; behavior unchanged |
| `manager.go` imports `action/` creating tight coupling | `ActionService` is already an interface; Manager depends on interface, not impl |
| Test files reference unexported functions | Unexported helpers move with their production file to the same sub-package |

---

## 3. Exported Symbol Catalog

Complete list of symbols that need aliasing, organized by source file and target sub-package.

### From `workflow_schema.go` -> `workflow/schema.go`

**Types**: `WorkflowConfig`, `StatusMetadata`
**Constants**: `StartStatusKey`, `CompleteStatusKey`, `AggregationStatusKey`, `DefaultWorkflowVersion`
**Methods** (on types, travel with type): `WorkflowConfig.GetStatusMetadata`, `WorkflowConfig.UnmarshalJSON`, `WorkflowConfig.GetStatusesByAgentType`, `WorkflowConfig.GetStatusesByPhase`, `WorkflowConfig.IsBackwardTransition`

### From `workflow_parser.go` -> `workflow/parser.go`

**Functions**: `LoadWorkflowConfig`, `ClearWorkflowCache`, `GetWorkflowOrDefault`, `LoadMultiLevelWorkflow`, `LoadMultiLevelWorkflowOrDefault`
**Unexported** (move, no alias needed): `workflowCache`, `workflowCacheLock`, `workflowCachePath`, `multiLevelCache`, `multiLevelCacheLock`, `multiLevelCachePath`, `parseWorkflowSection`, `parseTopLevelTaskWorkflow`, `validateWorkflowFilePath`, `resolveWorkflowFilePath`, `expandHome`, `loadWorkflowFile`

### From `workflow_validator.go` -> `workflow/validator.go`

**Types**: `WorkflowValidationError`, `WorkflowValidationFinding`
**Functions**: `ValidateWorkflow`, `ValidateWorkflowFiles`, `ValidateTransition`
**Unexported**: `validateSpecialStatuses`, `validateStatusReferences`, `validateReachability`, `validateTerminalPaths`, `readRawConfigKeys`

### From `workflow_default.go` -> `workflow/defaults.go`

**Functions**: `DefaultWorkflow`, `DefaultEpicWorkflow`, `DefaultBugWorkflow`, `DefaultChangeCardWorkflow`, `DefaultFeatureWorkflow`

### From `workflow_multilevel.go` -> `workflow/multilevel.go`

**Types**: `MultiLevelWorkflow`
**Methods**: `MultiLevelWorkflow.GetWorkflowForLevel`

### From `orchestrator_action.go` -> `action/orchestrator.go`

**Types**: `OrchestratorAction`, `PopulatedAction` (already in action_service.go, consolidate)
**Constants**: `ActionSpawnAgent`, `ActionPause`, `ActionWaitForTriage`, `ActionArchive`, `ActionAdvanceStatus`, `ActionCheckOrResume`, `ActionCascade`
**Variables**: `ValidActionTypes`
**Functions**: `ValidateAllOrchestratorActions`
**Unexported**: `stringSliceContains`, `validateTemplateSyntax`, `extractPlaceholders`

### From `action_service.go` -> `action/service.go`

**Types**: `ActionService` (interface), `PopulatedAction`, `ValidationResult`, `InvalidAction`, `StatusNotFoundError`, `DefaultActionService`
**Functions**: `NewActionService`

### From `template_helpers.go` -> `template/helpers.go`

**Types**: `TemplateEnrichmentData`, `TemplateEnrichmentRepository` (interface), `DocumentRepository` (interface), `FeatureRelationshipRepository` (interface), `EpicRelationshipRepository` (interface), `TaskRelationshipRepository` (interface)
**Functions**: `EntityPlaceholders`, `TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders`, `BugPlaceholders`, `ChangeCardPlaceholders`, `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`, `EpicPlaceholdersWithRelated`, `ApplyEnrichmentData`, `ParseEpicKeyFromEntityKey`, `ParseFeatureKeyFromTaskKey`
**Unexported**: `epicKeyPattern`, `featureKeyPattern`, `parseEpicKeyFromEntityKey`, `parseFeatureKeyFromTaskKey`, `formatDocPathsAsCSV`, `stringifyMetadataValue`, `extractContextDataFields`, `formatEpicKeysAsCSV`

### From `validation_error.go` -> `validation/error.go`

**Types**: `OrchestratorValidationError`, `ValidationError` (alias of `OrchestratorValidationError`)

---

*Last Updated*: 2026-03-24
