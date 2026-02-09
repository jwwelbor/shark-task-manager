# Technical Architecture: E16-F01 Core Workflow Engine

**Feature**: E16-F01-core-workflow-engine
**Epic**: E16 Multi-Level Workflow System
**Author**: Architect Agent
**Date**: 2026-02-08
**Status**: Draft

---

## 1. Architecture Overview

### Summary

This feature extends Shark's configurable workflow engine from task-only to three independent workflow levels: epic, feature, and task. Each level gets its own status flow, metadata, and validation in `.sharkconfig.json`. The existing task workflow remains at the top level of the config for backward compatibility. New `epic_workflow` and `feature_workflow` sections are parsed alongside it.

### Design Principles

1. **Appropriate**: Reuse the existing `WorkflowConfig` struct and `workflow.Service` API surface. Do not create a parallel type system.
2. **Proven**: Follow the exact same patterns established by E11 (configurable task workflows) and E13 (next-status commands).
3. **Simple**: The workflow engine is level-agnostic; the only difference between levels is which `WorkflowConfig` instance is loaded. No polymorphism, no generics, no interface hierarchies.

### High-Level Data Flow

```
.sharkconfig.json
       |
       v
config.LoadMultiLevelWorkflow(configPath)
       |
       +---> config.WorkflowConfig  (epic_workflow)
       +---> config.WorkflowConfig  (feature_workflow)
       +---> config.WorkflowConfig  (task workflow, top-level -- unchanged)
       |
       v
workflow.Service (level-aware)
       |
       +--- ForLevel("epic")    --> returns Service wrapping epic WorkflowConfig
       +--- ForLevel("feature") --> returns Service wrapping feature WorkflowConfig
       +--- ForLevel("task")    --> returns Service wrapping task WorkflowConfig (default)
       |
       v
CLI Commands / Services
       |
       +--- shark epic next-status E16
       +--- shark feature next-status E16-F01
       +--- shark epic update E16 --status ready_for_research
       +--- shark feature update E16-F01 --status ready_for_refinement_ba
```

### Entity Level Constants

A new set of string constants will be used throughout the codebase to identify workflow levels:

```go
// internal/workflow/levels.go (NEW FILE)
package workflow

// EntityLevel identifies which workflow level to use
const (
    LevelEpic    = "epic"
    LevelFeature = "feature"
    LevelTask    = "task"
)
```

---

## 2. Config Schema Changes

### 2.1 New Config Sections

The `.sharkconfig.json` gains two new top-level keys: `epic_workflow` and `feature_workflow`. Each uses the exact same structure as the existing top-level task workflow (`status_flow`, `status_metadata`, `special_statuses`, etc.). The existing top-level fields remain the task workflow, unchanged.

**Example config with all three levels:**

```json
{
  "epic_workflow": {
    "status_flow_version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { "_start_": ["draft"], "_complete_": ["completed", "cancelled"] }
  },
  "feature_workflow": {
    "status_flow_version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { "_start_": ["draft"], "_complete_": ["completed", "cancelled"] }
  },
  "status_flow_version": "1.0",
  "status_flow": { ... },
  "status_metadata": { ... },
  "special_statuses": { ... }
}
```

### 2.2 New StatusMetadata Fields

Two new fields are added to `StatusMetadata` for downstream features (E16-F03 Display & Aggregation). F01 parses them but does not act on them:

```go
// internal/config/workflow_schema.go -- additions to StatusMetadata struct

type StatusMetadata struct {
    // ... existing fields unchanged ...

    // IsPlanning indicates this status is a planning phase status.
    // When true, the entity has its own workflow status (not aggregating children).
    // When false (or omitted), the entity may aggregate progress from children.
    // Used by E16-F03 to control display behavior.
    IsPlanning bool `json:"is_planning,omitempty"`

    // AggregatesFrom indicates this status derives progress from children.
    // Values: "features" (epic aggregates features), "tasks" (feature aggregates tasks), "" (none).
    // Used by E16-F03 to switch between workflow display and progress display.
    AggregatesFrom string `json:"aggregates_from,omitempty"`
}
```

### 2.3 New Special Status Key

A new special status key constant for the aggregation threshold:

```go
// internal/config/workflow_schema.go -- addition to constants

const (
    StartStatusKey       = "_start_"
    CompleteStatusKey    = "_complete_"
    AggregationStatusKey = "_aggregation_"  // NEW: identifies aggregation threshold statuses
)
```

### 2.4 Reuse of WorkflowConfig

The `WorkflowConfig` struct in `internal/config/workflow_schema.go` is NOT changed structurally (aside from the two new `StatusMetadata` fields above). Epic, feature, and task workflows all use the same `WorkflowConfig` type. This is a deliberate design choice: the workflow engine is level-agnostic.

### 2.5 MultiLevelWorkflow Container

A new container struct holds all three workflow configs:

```go
// internal/config/workflow_multilevel.go (NEW FILE)
package config

// MultiLevelWorkflow holds workflow configurations for all entity levels.
// Any level may be nil, meaning "use default workflow for that level."
type MultiLevelWorkflow struct {
    Epic    *WorkflowConfig
    Feature *WorkflowConfig
    Task    *WorkflowConfig
}

// GetWorkflowForLevel returns the workflow config for the given level.
// Falls back to the appropriate default if nil.
//
// Parameters:
//   - level: one of "epic", "feature", "task"
//
// Returns:
//   - *WorkflowConfig: never nil (falls back to defaults)
func (m *MultiLevelWorkflow) GetWorkflowForLevel(level string) *WorkflowConfig {
    switch level {
    case "epic":
        if m.Epic != nil {
            return m.Epic
        }
        return DefaultEpicWorkflow()
    case "feature":
        if m.Feature != nil {
            return m.Feature
        }
        return DefaultFeatureWorkflow()
    case "task":
        if m.Task != nil {
            return m.Task
        }
        return DefaultWorkflow()
    default:
        return DefaultWorkflow()
    }
}
```

---

## 3. Service Layer Changes

### 3.1 workflow.Service Extension

The `workflow.Service` struct gains level awareness while maintaining backward compatibility. All existing methods continue to work unchanged (they operate on the task workflow by default).

**Approach: Service wraps a single WorkflowConfig, factory creates level-specific instances.**

```go
// internal/workflow/service.go -- MODIFIED

type Service struct {
    workflow    *config.WorkflowConfig
    projectRoot string
    level       string // NEW: "epic", "feature", or "task"
    multiLevel  *config.MultiLevelWorkflow // NEW: holds all three configs
}

// NewService creates a new Service for the task workflow (backward compatible).
// This is the existing constructor -- signature unchanged.
func NewService(projectRoot string) *Service {
    configPath := filepath.Join(projectRoot, ".sharkconfig.json")
    multi := config.LoadMultiLevelWorkflowOrDefault(configPath)

    return &Service{
        workflow:    multi.GetWorkflowForLevel(LevelTask),
        projectRoot: projectRoot,
        level:       LevelTask,
        multiLevel:  multi,
    }
}

// ForLevel returns a Service instance configured for the specified entity level.
// The returned service shares the same parsed config but operates on the
// level-specific workflow.
//
// Parameters:
//   - level: LevelEpic, LevelFeature, or LevelTask
//
// Returns:
//   - *Service: configured for the specified level
func (s *Service) ForLevel(level string) *Service {
    return &Service{
        workflow:    s.multiLevel.GetWorkflowForLevel(level),
        projectRoot: s.projectRoot,
        level:       level,
        multiLevel:  s.multiLevel,
    }
}

// GetLevel returns the entity level this service is configured for.
func (s *Service) GetLevel() string {
    return s.level
}
```

**Key design decision:** `ForLevel()` returns a new `*Service` instance wrapping the appropriate `WorkflowConfig`. All existing methods on `Service` (like `IsValidTransition`, `GetValidTransitions`, `GetStatusMetadata`, etc.) work without modification because they operate on `s.workflow`, which is now level-specific.

### 3.2 GetInitialStatus Generalization

The existing `GetInitialStatus()` returns `models.TaskStatus`. For multi-level support, we add a level-agnostic variant:

```go
// internal/workflow/service.go -- NEW METHOD

// GetInitialStatusString returns the first entry status as a plain string.
// Level-agnostic: works for epic, feature, and task levels.
func (s *Service) GetInitialStatusString() string {
    startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
    if !exists || len(startStatuses) == 0 {
        switch s.level {
        case LevelEpic, LevelFeature:
            return "draft"
        default:
            return "todo"
        }
    }
    return startStatuses[0]
}
```

The existing `GetInitialStatus() models.TaskStatus` method remains unchanged for backward compatibility.

### 3.3 New EpicService and FeatureService

These services do NOT exist today. They are created as part of this feature to provide the service layer between CLI commands and repositories for status transition operations.

```go
// internal/services/epic_service.go (NEW FILE)
package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
    "github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// EpicService provides business logic for epic operations.
type EpicService struct {
    repo        *repository.EpicRepository
    workflowSvc *workflow.Service
}

// NewEpicService creates a new EpicService.
func NewEpicService(repo *repository.EpicRepository, workflowSvc *workflow.Service) *EpicService {
    return &EpicService{
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),
    }
}

// TransitionStatus validates and performs a status transition on an epic.
//
// Parameters:
//   - ctx: context
//   - epicKey: the epic key (e.g., "E16")
//   - targetStatus: the desired new status
//   - force: if true, bypass workflow validation
//
// Returns:
//   - *TransitionResult: details of the transition
//   - error: validation or database errors
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, force bool) (*TransitionResult, error) {
    epic, err := s.repo.GetByKey(ctx, epicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to get epic: %w", err)
    }
    if epic == nil {
        return nil, fmt.Errorf("epic not found: %s", epicKey)
    }

    currentStatus := string(epic.Status)

    // Validate transition (unless forced)
    if !force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return nil, err
        }
    }

    // Normalize target status
    targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)

    // Perform update
    epic.Status = models.EpicStatus(targetStatus)
    if err := s.repo.Update(ctx, epic); err != nil {
        return nil, fmt.Errorf("failed to update epic status: %w", err)
    }

    return &TransitionResult{
        EntityType:    "epic",
        EntityKey:     epicKey,
        FromStatus:    currentStatus,
        ToStatus:      targetStatus,
        Transitioned:  true,
    }, nil
}

// GetNextStatus returns the next logical status for an epic (first in status_flow array).
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
    epic, err := s.repo.GetByKey(ctx, epicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to get epic: %w", err)
    }
    if epic == nil {
        return nil, fmt.Errorf("epic not found: %s", epicKey)
    }

    currentStatus := string(epic.Status)
    transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
    currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

    return &NextStatusInfo{
        EntityType:           "epic",
        EntityKey:            epicKey,
        CurrentStatus:        currentStatus,
        CurrentPhase:         currentMeta.Phase,
        AvailableTransitions: transitions,
        IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
    }, nil
}

// ValidateStatus checks if a status is valid in the epic workflow.
func (s *EpicService) ValidateStatus(status string) error {
    return s.workflowSvc.ValidateStatus(status)
}
```

```go
// internal/services/feature_service.go (NEW FILE)
package services

// FeatureService follows the identical pattern as EpicService,
// substituting repository.FeatureRepository, models.FeatureStatus,
// and workflow.LevelFeature. See EpicService for the complete pattern.

type FeatureService struct {
    repo        *repository.FeatureRepository
    workflowSvc *workflow.Service
}

func NewFeatureService(repo *repository.FeatureRepository, workflowSvc *workflow.Service) *FeatureService {
    return &FeatureService{
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelFeature),
    }
}

// TransitionStatus, GetNextStatus, ValidateStatus follow the same signatures
// and logic as EpicService, substituting Feature types.
```

### 3.4 Shared Transition Types

New shared types for cross-level transition results:

```go
// internal/services/transition_types.go (NEW FILE)
package services

import "github.com/jwwelbor/shark-task-manager/internal/workflow"

// TransitionResult represents the outcome of a status transition.
type TransitionResult struct {
    EntityType   string `json:"entity_type"`   // "epic", "feature", "task"
    EntityKey    string `json:"entity_key"`
    FromStatus   string `json:"from_status"`
    ToStatus     string `json:"to_status"`
    Transitioned bool   `json:"transitioned"`
    Message      string `json:"message,omitempty"`
}

// NextStatusInfo contains the available transitions for an entity.
type NextStatusInfo struct {
    EntityType           string                   `json:"entity_type"`
    EntityKey            string                   `json:"entity_key"`
    CurrentStatus        string                   `json:"current_status"`
    CurrentPhase         string                   `json:"current_phase,omitempty"`
    AvailableTransitions []workflow.TransitionInfo `json:"available_transitions"`
    IsTerminal           bool                     `json:"is_terminal"`
}
```

### 3.5 ValidateTransition on Service

The existing `workflow.Service` has `IsValidTransition()` but no method that returns a structured error with valid transitions listed. Add:

```go
// internal/workflow/service.go -- NEW METHOD

// ValidateTransition checks if a transition is valid and returns a descriptive error if not.
// Delegates to config.ValidateTransition for the actual check.
func (s *Service) ValidateTransition(fromStatus, toStatus string) error {
    return config.ValidateTransition(s.workflow, fromStatus, toStatus)
}
```

---

## 4. Repository Layer Changes

### 4.1 No Schema Changes Required

The `epics` and `features` tables already have a `status` TEXT column. No database migration is needed. The `CHECK` constraint on the status column (if any) should be verified to not be overly restrictive. Current SQLite schema uses TEXT without CHECK constraints for status, so any string value is accepted at the database level.

### 4.2 Epic Repository

The existing `repository.EpicRepository` already has `GetByKey(ctx, key)` and `Update(ctx, epic)` methods. No new repository methods are required for basic status transitions.

If the `Update` method does not currently accept arbitrary status values (i.e., if it validates against hardcoded statuses), that validation must be removed from the repository. Status validation belongs in the service layer.

### 4.3 Feature Repository

Same as epic: the existing `repository.FeatureRepository` already has the needed CRUD methods. No new methods required.

### 4.4 Repository Verification Checklist

Before implementation, verify these repository methods exist and accept arbitrary status strings:

- [ ] `EpicRepository.GetByKey(ctx context.Context, key string) (*models.Epic, error)`
- [ ] `EpicRepository.Update(ctx context.Context, epic *models.Epic) error`
- [ ] `FeatureRepository.GetByKey(ctx context.Context, key string) (*models.Feature, error)`
- [ ] `FeatureRepository.Update(ctx context.Context, feature *models.Feature) error`

---

## 5. CLI Command Changes

### 5.1 New Commands

#### 5.1.1 `shark epic next-status <key>`

**File**: `internal/cli/commands/epic_next_status.go` (NEW FILE)

```go
var epicNextStatusCmd = &cobra.Command{
    Use:   "next-status <epic-key>",
    Short: "Progress epic to next workflow status",
    Long: `Progress an epic through its configured workflow.
Same behavior as 'shark task next-status' but for epics.

Examples:
  shark epic next-status E16              Advance to next status
  shark epic next-status E16 --preview    Show available transitions
  shark epic next-status E16 --status=ready_for_research  Direct transition
  shark epic next-status E16 --json       JSON output`,
    Args: cobra.ExactArgs(1),
    RunE: runEpicNextStatus,
}

func init() {
    epicNextStatusCmd.Flags().String("status", "", "Target status (non-interactive)")
    epicNextStatusCmd.Flags().Bool("preview", false, "Show transitions without changes")
    epicNextStatusCmd.Flags().Bool("force", false, "Bypass workflow validation")
    // Register under epic parent command
    epicCmd.AddCommand(epicNextStatusCmd)
}
```

The `runEpicNextStatus` function follows the same pattern as `runTaskNextStatus` in `internal/cli/commands/task_next_status.go`:

1. Parse the epic key from args
2. Create `EpicService` via `cli.GetEpicService()`
3. Call `svc.GetNextStatus(ctx, epicKey)` to get available transitions
4. If `--preview`: display and return
5. If `--status`: validate target and call `svc.TransitionStatus(ctx, epicKey, targetStatus, force)`
6. Otherwise: auto-select first transition (non-interactive) or prompt (interactive)
7. Format output as JSON or human-readable

The JSON output structure:

```json
{
  "entity_type": "epic",
  "entity_key": "E16",
  "current_status": "draft",
  "current_phase": "planning",
  "available_transitions": [
    { "target_status": "ready_for_research", "description": "...", "phase": "research" },
    { "target_status": "active", "description": "...", "phase": "execution" },
    { "target_status": "cancelled", "description": "...", "phase": "done" }
  ],
  "new_status": "ready_for_research",
  "transitioned": true,
  "message": "Epic E16: draft -> ready_for_research"
}
```

#### 5.1.2 `shark feature next-status <key>`

**File**: `internal/cli/commands/feature_next_status.go` (NEW FILE)

Identical pattern to `epic_next_status.go`, substituting:
- `FeatureService` for `EpicService`
- `featureCmd.AddCommand(featureNextStatusCmd)` for registration
- Feature key parsing instead of epic key parsing

### 5.2 Modified Commands

#### 5.2.1 `shark epic update <key> --status <new>`

**File**: `internal/cli/commands/epic.go` -- modify `runEpicUpdate`

**Current behavior**: Calls `ParseEpicStatus()` which validates against hardcoded `draft/active/completed/archived`, then calls `epicRepo.Update()` directly.

**New behavior**:

1. Remove the `ParseEpicStatus()` call
2. Create `EpicService` via `cli.GetEpicService()`
3. Call `svc.TransitionStatus(ctx, epicKey, newStatus, force)`
4. The service handles workflow validation internally

```go
// Before (current code in runEpicUpdate):
newStatus, err := ParseEpicStatus(statusFlag)
// ...
epic.Status = newStatus
err = epicRepo.Update(ctx, epic)

// After (new code):
epicSvc := cli.GetEpicService()
result, err := epicSvc.TransitionStatus(ctx, epicKey, statusFlag, forceFlag)
```

#### 5.2.2 `shark feature update <key> --status <new>`

**File**: `internal/cli/commands/feature.go` -- modify `runFeatureUpdate`

Same refactoring as epic update: replace `ParseFeatureStatus()` with `FeatureService.TransitionStatus()`.

#### 5.2.3 `shark workflow validate`

**File**: `internal/cli/commands/workflow_validate.go` (existing or new)

Extend to validate all three levels:

```
$ shark workflow validate

Epic workflow: valid (12 statuses, custom)
Feature workflow: valid (13 statuses, custom)
Task workflow: valid (19 statuses, custom)

All workflows valid.
```

If any level uses defaults, show:

```
Epic workflow: valid (4 statuses, default)
```

### 5.3 Service Accessors

New global accessors for the service layer, following the existing pattern in `internal/cli/`:

```go
// internal/cli/service_accessors.go (NEW FILE or added to existing db_global.go)

// GetEpicService returns an EpicService instance.
func GetEpicService() *services.EpicService {
    db, err := GetDB(context.Background())
    if err != nil {
        // Fatal: cannot proceed without DB
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    epicRepo := repository.NewEpicRepository(db)
    projectRoot, _ := FindProjectRoot()
    workflowSvc := workflow.NewService(projectRoot)
    return services.NewEpicService(epicRepo, workflowSvc)
}

// GetFeatureService returns a FeatureService instance.
func GetFeatureService() *services.FeatureService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    featureRepo := repository.NewFeatureRepository(db)
    projectRoot, _ := FindProjectRoot()
    workflowSvc := workflow.NewService(projectRoot)
    return services.NewFeatureService(featureRepo, workflowSvc)
}
```

**Note**: The `panic` approach matches the existing `cli.GetDB()` failure pattern. In future, these should return errors, but that is outside the scope of F01.

---

## 6. Validation Changes

### 6.1 Model Layer: Remove Hardcoded Status Validation

**File**: `internal/models/validation.go`

**Current**: `ValidateEpicStatus()` and `ValidateFeatureStatus()` hardcode `draft/active/completed/archived`.

**New**: Follow the same pattern as `ValidateTaskStatus()` -- only check for non-empty string. Workflow-aware validation happens at the service layer.

```go
// ValidateEpicStatus performs basic structural validation on an epic status string.
// It only checks that the status is non-empty after trimming whitespace.
//
// For workflow-aware validation, use EpicService.ValidateStatus(status) or
// workflow.Service.ForLevel("epic").ValidateStatus(status).
func ValidateEpicStatus(status string) error {
    if strings.TrimSpace(status) == "" {
        return fmt.Errorf("%w: status cannot be empty", ErrInvalidEpicStatus)
    }
    return nil
}

// ValidateFeatureStatus performs basic structural validation on a feature status string.
func ValidateFeatureStatus(status string) error {
    if strings.TrimSpace(status) == "" {
        return fmt.Errorf("%w: status cannot be empty", ErrInvalidFeatureStatus)
    }
    return nil
}
```

**Impact**: `ErrInvalidEpicStatus` and `ErrInvalidFeatureStatus` error variables must have their messages updated:

```go
// Before:
ErrInvalidEpicStatus    = errors.New("invalid epic status: must be draft, active, completed, or archived")
ErrInvalidFeatureStatus = errors.New("invalid feature status: must be draft, active, completed, or archived")

// After:
ErrInvalidEpicStatus    = errors.New("invalid epic status")
ErrInvalidFeatureStatus = errors.New("invalid feature status")
```

### 6.2 Model Layer: Remove Hardcoded Status Constants

The following constants in `internal/models/epic.go` and `internal/models/feature.go` should be **kept but deprecated** with comments. They remain valid as default status values but are no longer the sole valid set:

```go
// internal/models/epic.go
// EpicStatus constants represent the DEFAULT epic statuses.
// When a custom epic_workflow is configured, additional statuses are valid.
// Use workflow.Service.ForLevel("epic").ValidateStatus() for runtime validation.
const (
    EpicStatusDraft     EpicStatus = "draft"
    EpicStatusActive    EpicStatus = "active"
    EpicStatusCompleted EpicStatus = "completed"
    EpicStatusArchived  EpicStatus = "archived"
)
```

### 6.3 CLI Layer: Update ValidateStatus Function

**File**: `internal/cli/commands/validators.go`

**Current**: `ValidateStatus(status, entityType)` delegates to `models.ValidateEpicStatus()` / `models.ValidateFeatureStatus()`.

**New**: Delegate to the workflow service for workflow-aware validation:

```go
// ValidateStatus validates a status using the workflow engine.
// For epic and feature, uses the level-specific workflow.
// For task, uses the task workflow.
func ValidateStatus(status string, entityType string) error {
    if status == "" {
        return fmt.Errorf("%s status cannot be empty", entityType)
    }

    projectRoot, err := cli.FindProjectRoot()
    if err != nil {
        // Fallback to basic non-empty check
        return nil
    }

    workflowSvc := workflow.NewService(projectRoot)

    switch entityType {
    case "epic":
        return workflowSvc.ForLevel(workflow.LevelEpic).ValidateStatus(status)
    case "feature":
        return workflowSvc.ForLevel(workflow.LevelFeature).ValidateStatus(status)
    default:
        return workflowSvc.ValidateStatus(status)
    }
}
```

### 6.4 Config Validation Extension

**File**: `internal/config/workflow_validator.go` -- no structural changes needed.

The existing `ValidateWorkflow(workflow *WorkflowConfig) error` function is already level-agnostic. It takes a `*WorkflowConfig` and validates it. To validate all levels, the caller simply calls it three times:

```go
// In shark workflow validate command:
multi := config.LoadMultiLevelWorkflowOrDefault(configPath)

if err := config.ValidateWorkflow(multi.Epic); err != nil {
    fmt.Printf("Epic workflow: INVALID - %s\n", err)
}
if err := config.ValidateWorkflow(multi.Feature); err != nil {
    fmt.Printf("Feature workflow: INVALID - %s\n", err)
}
if err := config.ValidateWorkflow(multi.Task); err != nil {
    fmt.Printf("Task workflow: INVALID - %s\n", err)
}
```

---

## 7. Data Flow Diagrams

### 7.1 Config Parsing Flow

```
.sharkconfig.json
       |
       v
config.LoadMultiLevelWorkflow(configPath)
       |
       +--- Read JSON file
       +--- Parse as map[string]interface{}
       |
       +--- Check for "epic_workflow" key
       |     +--- Found? Parse as WorkflowConfig
       |     +--- Missing? Set Epic = nil (will use DefaultEpicWorkflow)
       |
       +--- Check for "feature_workflow" key
       |     +--- Found? Parse as WorkflowConfig
       |     +--- Missing? Set Feature = nil (will use DefaultFeatureWorkflow)
       |
       +--- Check for "status_flow" key (top-level = task)
       |     +--- Found? Parse as WorkflowConfig (existing logic)
       |     +--- Missing? Set Task = nil (will use DefaultWorkflow)
       |
       +--- Return *MultiLevelWorkflow { Epic, Feature, Task }
```

### 7.2 Epic Next-Status Command Flow

```
User: shark epic next-status E16
       |
       v
CLI: runEpicNextStatus(cmd, args)
       |
       +--- Parse "E16" from args[0]
       +--- epicSvc := cli.GetEpicService()
       |       +--- Creates EpicRepository
       |       +--- Creates workflow.Service
       |       +--- Calls NewEpicService(repo, workflowSvc)
       |             +--- workflowSvc.ForLevel("epic") -> epic-specific Service
       |
       +--- info, err := epicSvc.GetNextStatus(ctx, "E16")
       |       +--- repo.GetByKey(ctx, "E16") -> epic at status "draft"
       |       +--- workflowSvc.GetTransitionInfo("draft")
       |             -> ["ready_for_research", "active", "cancelled"]
       |
       +--- Auto-select first: "ready_for_research"
       |
       +--- result, err := epicSvc.TransitionStatus(ctx, "E16", "ready_for_research", false)
       |       +--- workflowSvc.ValidateTransition("draft", "ready_for_research") -> valid
       |       +--- repo.Update(ctx, epic{Status: "ready_for_research"})
       |
       +--- Output: "Epic E16: draft -> ready_for_research"
```

### 7.3 Feature Status Update with Validation

```
User: shark feature update E16-F01 --status in_refinement
       |
       v
CLI: runFeatureUpdate(cmd, args)
       |
       +--- featureSvc := cli.GetFeatureService()
       +--- result, err := featureSvc.TransitionStatus(ctx, "E16-F01", "in_refinement", false)
       |       +--- repo.GetByKey(ctx, "E16-F01") -> feature at status "draft"
       |       +--- workflowSvc.ValidateTransition("draft", "in_refinement") -> ERROR
       |             "invalid transition from 'draft' to 'in_refinement'.
       |              Valid: ready_for_refinement_ba, active, cancelled.
       |              Use --force to override"
       |
       +--- Output error to user with valid transitions listed
```

---

## 8. Backward Compatibility Strategy

### 8.1 Missing Config Sections

When `.sharkconfig.json` lacks `epic_workflow` or `feature_workflow`:

- **Epic**: Uses `DefaultEpicWorkflow()` with statuses: `draft`, `active`, `completed`, `archived`
- **Feature**: Uses `DefaultFeatureWorkflow()` with the same four statuses
- **Task**: Uses existing `DefaultWorkflow()` -- completely unchanged

### 8.2 Default Workflows

```go
// internal/config/workflow_default.go -- NEW FUNCTIONS

// DefaultEpicWorkflow returns the backward-compatible default epic workflow.
// Matches the current hardcoded epic status set: draft, active, completed, archived.
func DefaultEpicWorkflow() *WorkflowConfig {
    return &WorkflowConfig{
        Version: DefaultWorkflowVersion,
        StatusFlow: map[string][]string{
            "draft":     {"active", "archived"},
            "active":    {"completed", "archived"},
            "completed": {"archived"},
            "archived":  {},
        },
        StatusMetadata: map[string]StatusMetadata{
            "draft":     {Color: "gray",  Description: "Epic created, not yet started", Phase: "planning", IsPlanning: true},
            "active":    {Color: "blue",  Description: "Epic in progress, aggregating features", Phase: "execution", AggregatesFrom: "features"},
            "completed": {Color: "green", Description: "All features complete", Phase: "done"},
            "archived":  {Color: "gray",  Description: "Epic archived", Phase: "done"},
        },
        SpecialStatuses: map[string][]string{
            StartStatusKey:       {"draft"},
            CompleteStatusKey:    {"completed", "archived"},
            AggregationStatusKey: {"active"},
        },
    }
}

// DefaultFeatureWorkflow returns the backward-compatible default feature workflow.
func DefaultFeatureWorkflow() *WorkflowConfig {
    return &WorkflowConfig{
        Version: DefaultWorkflowVersion,
        StatusFlow: map[string][]string{
            "draft":     {"active", "archived"},
            "active":    {"completed", "archived"},
            "completed": {"archived"},
            "archived":  {},
        },
        StatusMetadata: map[string]StatusMetadata{
            "draft":     {Color: "gray",  Description: "Feature created, not yet started", Phase: "planning", IsPlanning: true},
            "active":    {Color: "blue",  Description: "Feature in progress, aggregating tasks", Phase: "execution", AggregatesFrom: "tasks"},
            "completed": {Color: "green", Description: "All tasks complete", Phase: "done"},
            "archived":  {Color: "gray",  Description: "Feature archived", Phase: "done"},
        },
        SpecialStatuses: map[string][]string{
            StartStatusKey:       {"draft"},
            CompleteStatusKey:    {"completed", "archived"},
            AggregationStatusKey: {"active"},
        },
    }
}
```

### 8.3 The `draft -> active` Shortcut

Per NFR-004, `draft -> active` must always be valid at all levels. This is guaranteed by:

1. Both custom example workflows in the epic PRD include `"draft": ["...", "active", "..."]`
2. Both default workflows include `"draft": ["active", ...]`
3. The `shark workflow validate` command should warn (not error) if `draft -> active` is not in the status flow. This is a soft validation since users might choose to enforce a mandatory planning workflow.

### 8.4 Existing Command Compatibility

| Command | Current Behavior | New Behavior | Breaking? |
|---------|-----------------|--------------|-----------|
| `shark epic update E16 --status active` | Validates against hardcoded set, updates | Validates against epic workflow, updates | No -- `active` is valid in default workflow |
| `shark feature update E16-F01 --status active` | Same | Same | No |
| `shark epic update E16 --status in_refinement` | Rejects (not in hardcoded set) | Validates against epic workflow; succeeds if custom workflow defines it | No -- expands capability |
| `shark epic list` | Works | Works -- no change | No |
| `shark task next-status` | Works | Works -- no change to task workflow | No |

### 8.5 Config Caching

**Current**: Single global cache (`workflowCache` in `workflow_parser.go`).

**New**: The `LoadMultiLevelWorkflow()` function replaces `LoadWorkflowConfig()` for the cache. The new cache stores a `*MultiLevelWorkflow` instead of a single `*WorkflowConfig`. The existing `LoadWorkflowConfig()` and `GetWorkflowOrDefault()` functions remain for backward compatibility but internally delegate to the multi-level loader (extracting just the task workflow).

```go
// internal/config/workflow_parser.go -- MODIFIED CACHE

var (
    multiLevelCache     *MultiLevelWorkflow
    multiLevelCacheLock sync.RWMutex
    multiLevelCachePath string
)

// LoadMultiLevelWorkflow loads all three workflow configs from .sharkconfig.json.
func LoadMultiLevelWorkflow(configPath string) (*MultiLevelWorkflow, error) {
    // Cache check (same pattern as existing LoadWorkflowConfig)
    // ...
    // Parse rawConfig as map[string]interface{}
    // Extract "epic_workflow" -> parse as WorkflowConfig
    // Extract "feature_workflow" -> parse as WorkflowConfig
    // Extract top-level status_flow -> parse as WorkflowConfig (existing logic)
    // Return &MultiLevelWorkflow{Epic: epicWf, Feature: featureWf, Task: taskWf}
}

// LoadMultiLevelWorkflowOrDefault loads configs or returns defaults for missing sections.
func LoadMultiLevelWorkflowOrDefault(configPath string) *MultiLevelWorkflow {
    multi, err := LoadMultiLevelWorkflow(configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Warning: Failed to load workflow config: %v\n", err)
        return &MultiLevelWorkflow{} // All nil = all defaults
    }
    if multi == nil {
        return &MultiLevelWorkflow{}
    }
    return multi
}

// GetWorkflowOrDefault remains unchanged externally but delegates internally:
func GetWorkflowOrDefault(configPath string) *WorkflowConfig {
    multi := LoadMultiLevelWorkflowOrDefault(configPath)
    return multi.GetWorkflowForLevel("task")
}
```

---

## 9. File Change Inventory

### New Files

| File | Purpose | Size Estimate |
|------|---------|---------------|
| `internal/workflow/levels.go` | Entity level constants (`LevelEpic`, `LevelFeature`, `LevelTask`) | XS (~15 lines) |
| `internal/config/workflow_multilevel.go` | `MultiLevelWorkflow` container struct and `GetWorkflowForLevel()` | S (~60 lines) |
| `internal/services/epic_service.go` | `EpicService` with `TransitionStatus()`, `GetNextStatus()`, `ValidateStatus()` | M (~120 lines) |
| `internal/services/feature_service.go` | `FeatureService` with same methods | M (~120 lines) |
| `internal/services/transition_types.go` | `TransitionResult`, `NextStatusInfo` shared types | XS (~35 lines) |
| `internal/cli/commands/epic_next_status.go` | `shark epic next-status` command | M (~200 lines, follows task_next_status.go pattern) |
| `internal/cli/commands/feature_next_status.go` | `shark feature next-status` command | M (~200 lines) |
| `internal/cli/service_accessors.go` | `GetEpicService()`, `GetFeatureService()` factory functions | S (~50 lines) |

### Modified Files

| File | Changes | Impact |
|------|---------|--------|
| `internal/config/workflow_schema.go` | Add `IsPlanning`, `AggregatesFrom` to `StatusMetadata`; add `AggregationStatusKey` constant | Low -- additive only |
| `internal/config/workflow_parser.go` | Replace single cache with multi-level cache; add `LoadMultiLevelWorkflow()`, `LoadMultiLevelWorkflowOrDefault()`; update `GetWorkflowOrDefault()` to delegate | Medium -- core parsing logic |
| `internal/config/workflow_default.go` | Add `DefaultEpicWorkflow()`, `DefaultFeatureWorkflow()` | Low -- additive |
| `internal/workflow/service.go` | Add `level` and `multiLevel` fields to `Service`; add `ForLevel()`, `GetLevel()`, `GetInitialStatusString()`, `ValidateTransition()` methods; modify `NewService()` to use multi-level loader | Medium -- preserves existing API |
| `internal/models/validation.go` | Change `ValidateEpicStatus()` and `ValidateFeatureStatus()` from hardcoded to non-empty check; update error message variables | Low -- less restrictive |
| `internal/models/epic.go` | Add deprecation comments to status constants | XS -- comments only |
| `internal/models/feature.go` | Add deprecation comments to status constants | XS -- comments only |
| `internal/cli/commands/epic.go` | Modify `runEpicUpdate` to use `EpicService.TransitionStatus()` instead of direct repo call with hardcoded validation | Medium -- refactor to service pattern |
| `internal/cli/commands/feature.go` | Modify `runFeatureUpdate` to use `FeatureService.TransitionStatus()` | Medium -- refactor to service pattern |
| `internal/cli/commands/validators.go` | Update `ValidateStatus()` to use workflow service | Low -- simple delegation change |

### Test Files (New)

| File | Tests |
|------|-------|
| `internal/config/workflow_multilevel_test.go` | Parse multi-level config, fallback to defaults, missing sections |
| `internal/workflow/service_multilevel_test.go` | `ForLevel()`, level-specific transitions, level isolation |
| `internal/services/epic_service_test.go` | `TransitionStatus`, `GetNextStatus`, validation (mocked repo) |
| `internal/services/feature_service_test.go` | Same pattern as epic |
| `internal/cli/commands/epic_next_status_test.go` | Command args, JSON output, preview mode (mocked service) |
| `internal/cli/commands/feature_next_status_test.go` | Same pattern as epic |

---

## 10. Implementation Order

### Wave 1: Config Parsing Foundation (No CLI changes)

**Tasks (in order):**

1. **Add `IsPlanning` and `AggregatesFrom` to `StatusMetadata`** in `internal/config/workflow_schema.go`
   - Add `AggregationStatusKey` constant
   - Complexity: XS
   - Tests: Verify JSON parsing of new fields

2. **Create `DefaultEpicWorkflow()` and `DefaultFeatureWorkflow()`** in `internal/config/workflow_default.go`
   - Complexity: S
   - Tests: Verify default workflows pass validation

3. **Create `MultiLevelWorkflow` container** in `internal/config/workflow_multilevel.go`
   - Implement `GetWorkflowForLevel()`
   - Complexity: S
   - Tests: Level routing, nil fallback to defaults

4. **Create `LoadMultiLevelWorkflow()` parser** in `internal/config/workflow_parser.go`
   - Replace single cache with multi-level cache
   - Backward-compatible `GetWorkflowOrDefault()` delegation
   - Complexity: M
   - Tests: Parse all three sections, missing sections, invalid JSON, cache behavior

### Wave 2: Workflow Service Extension

5. **Create `internal/workflow/levels.go`** with level constants
   - Complexity: XS

6. **Extend `workflow.Service` with `ForLevel()` and related methods**
   - Add `level`, `multiLevel` fields
   - Add `ForLevel()`, `GetLevel()`, `GetInitialStatusString()`, `ValidateTransition()`
   - Modify `NewService()` to load multi-level configs
   - Complexity: M
   - Tests: Level isolation, existing task methods unchanged

### Wave 3: Service Layer

7. **Create `internal/services/transition_types.go`** with shared types
   - Complexity: XS

8. **Create `internal/services/epic_service.go`**
   - `TransitionStatus()`, `GetNextStatus()`, `ValidateStatus()`
   - Complexity: M
   - Tests: Valid transition, invalid transition, force, not found (mocked repo)

9. **Create `internal/services/feature_service.go`**
   - Same pattern as epic
   - Complexity: M
   - Tests: Same coverage as epic

### Wave 4: Validation Refactoring

10. **Update `internal/models/validation.go`**
    - Change `ValidateEpicStatus()` and `ValidateFeatureStatus()` to non-empty check
    - Update error message variables
    - Complexity: S
    - Tests: Verify non-empty check, verify custom statuses pass

11. **Update `internal/cli/commands/validators.go`**
    - Use workflow service for status validation
    - Complexity: S
    - Tests: Verify workflow-aware validation

### Wave 5: CLI Commands

12. **Create `internal/cli/service_accessors.go`**
    - `GetEpicService()`, `GetFeatureService()` factories
    - Complexity: S

13. **Create `internal/cli/commands/epic_next_status.go`**
    - Full command implementation following `task_next_status.go` pattern
    - Complexity: M
    - Tests: Args, preview, status flag, JSON output (mocked service)

14. **Create `internal/cli/commands/feature_next_status.go`**
    - Same pattern as epic
    - Complexity: M
    - Tests: Same coverage

15. **Modify `internal/cli/commands/epic.go`** -- refactor `runEpicUpdate`
    - Use `EpicService.TransitionStatus()` instead of direct repo
    - Complexity: M
    - Tests: Valid transition, invalid transition, force flag

16. **Modify `internal/cli/commands/feature.go`** -- refactor `runFeatureUpdate`
    - Use `FeatureService.TransitionStatus()` instead of direct repo
    - Complexity: M
    - Tests: Same coverage

17. **Extend `shark workflow validate`** to validate all three levels
    - Complexity: S
    - Tests: All valid, one invalid, default workflows

---

## 11. Testing Strategy

### 11.1 Config Parsing Tests (Real data, no DB)

Location: `internal/config/workflow_multilevel_test.go`

| Test | Description |
|------|-------------|
| `TestLoadMultiLevelWorkflow_AllPresent` | Parse config with all three sections |
| `TestLoadMultiLevelWorkflow_OnlyTask` | Only top-level task workflow; epic/feature nil |
| `TestLoadMultiLevelWorkflow_MissingFile` | No config file; all nil |
| `TestLoadMultiLevelWorkflow_InvalidEpicWorkflow` | Invalid `epic_workflow` JSON; error returned |
| `TestLoadMultiLevelWorkflow_ValidEpicInvalidFeature` | Valid epic, invalid feature; error for feature |
| `TestGetWorkflowForLevel_FallbackDefaults` | Nil sections use appropriate defaults |
| `TestGetWorkflowForLevel_EpicHasCorrectStatuses` | Custom epic workflow has expected statuses |
| `TestLoadMultiLevelWorkflow_CacheBehavior` | Second call returns cached; clear invalidates |

### 11.2 Workflow Service Tests (No DB)

Location: `internal/workflow/service_multilevel_test.go`

| Test | Description |
|------|-------------|
| `TestForLevel_Epic` | `ForLevel("epic")` returns epic-specific service |
| `TestForLevel_Feature` | `ForLevel("feature")` returns feature-specific service |
| `TestForLevel_Task` | `ForLevel("task")` returns task service (same as default) |
| `TestForLevel_Isolation` | Epic transitions do not affect task transitions |
| `TestNewService_BackwardCompatible` | `NewService()` still works for tasks |
| `TestGetInitialStatusString_Epic` | Returns "draft" for epic |
| `TestGetInitialStatusString_Task` | Returns "todo" for task (unchanged) |
| `TestValidateTransition_ValidEpic` | `draft -> ready_for_research` valid |
| `TestValidateTransition_InvalidEpic` | `draft -> in_refinement` invalid |

### 11.3 Service Tests (Mocked Repository)

Location: `internal/services/epic_service_test.go`, `internal/services/feature_service_test.go`

| Test | Description |
|------|-------------|
| `TestTransitionStatus_Valid` | Valid transition succeeds |
| `TestTransitionStatus_Invalid` | Invalid transition returns error with valid options |
| `TestTransitionStatus_Force` | Force bypasses validation |
| `TestTransitionStatus_NotFound` | Non-existent entity returns error |
| `TestGetNextStatus_HasTransitions` | Returns available transitions |
| `TestGetNextStatus_Terminal` | Terminal status returns empty transitions |
| `TestValidateStatus_CustomValid` | Custom workflow status passes |
| `TestValidateStatus_CustomInvalid` | Unknown status fails |

### 11.4 CLI Command Tests (Mocked Service)

Location: `internal/cli/commands/epic_next_status_test.go`, `internal/cli/commands/feature_next_status_test.go`

| Test | Description |
|------|-------------|
| `TestEpicNextStatus_Preview` | Preview mode shows transitions, no changes |
| `TestEpicNextStatus_DirectStatus` | `--status` flag transitions directly |
| `TestEpicNextStatus_AutoSelect` | No flag auto-selects first transition |
| `TestEpicNextStatus_JSONOutput` | `--json` returns structured response |
| `TestEpicNextStatus_InvalidKey` | Bad key returns error |
| `TestEpicNextStatus_Terminal` | Terminal status returns warning |

### 11.5 Integration Test (Manual)

After implementation, verify these end-to-end scenarios:

1. `shark init --non-interactive` with no `epic_workflow` in config; `shark epic update E16 --status active` works
2. Add custom `epic_workflow` to `.sharkconfig.json`; `shark epic next-status E16` advances through custom statuses
3. `shark workflow validate` reports all three levels
4. Existing `shark task next-status` behavior unchanged
5. `shark epic update E16 --status invalid_status` returns error listing valid transitions

---

## 12. Open Questions Resolution

### Q1: Should `active` be configurable or always the aggregation threshold?

**Decision**: `active` is NOT hardcoded as the aggregation threshold. The `_aggregation_` special status key in `special_statuses` controls which statuses trigger aggregation. This allows custom workflows where the aggregation status has a different name. However, both default workflows and the example configs use `active` as the aggregation status by convention.

**Rationale**: Keeping it configurable via `_aggregation_` is more flexible and follows the same pattern as `_start_` and `_complete_`. No extra implementation cost.

### Q2: Should `shark feature next-status` auto-transition to `active` after `ready_to_build`?

**Decision**: No. `next-status` always selects the **first entry** in the `status_flow` array for the current status. If the workflow config defines `"ready_to_build": ["active", "blocked"]`, then `next-status` from `ready_to_build` will auto-select `active`. This is controlled entirely by config ordering, not by special-case code.

**Rationale**: Consistent behavior across all levels. The workflow designer controls the default "next" action by ordering the status_flow arrays.

### Q3: Should epic completion be auto-detected?

**Decision**: No, not in F01 scope. Epic completion requires explicit `shark epic next-status` or `shark epic update --status completed`. Auto-completion based on child feature status is a potential future enhancement for E16-F03 (Display & Aggregation).

**Rationale**: Auto-completion introduces complexity around edge cases (what about cancelled features? features added later?). Explicit control is simpler and more predictable.

### Q4: History table schema for epic/feature status transitions

**Decision**: Out of scope for F01. The current `task_history` table is task-specific. Epic and feature status transitions will NOT be recorded in a history table in F01. This is tracked as a future enhancement. The `TransitionResult` struct returned by the service layer captures the transition details for CLI output, but does not persist them.

**Rationale**: Adding history tables requires schema migration and is orthogonal to the core workflow engine. E16-F04 (Notes & Context) or a separate feature can address this.

### Q5: Feature status in `shark task list` output

**Decision**: Out of scope for F01. This is a display concern handled by E16-F03 (Display & Aggregation). The core workflow engine provides the infrastructure to query feature status, but the task list command is not modified in F01.

**Rationale**: F01 focuses on the engine; F03 focuses on display integration.

---

## Appendix A: Complete Default Workflow Status Flows

### Default Epic Workflow

```
draft --> active --> completed --> archived
  |         |
  +-------> archived
```

### Default Feature Workflow

```
draft --> active --> completed --> archived
  |         |
  +-------> archived
```

### Example Custom Epic Workflow (from Epic PRD)

```
draft --> ready_for_research --> in_research --> ready_for_refinement -->
  |         ^                                        |
  |         |                                        v
  |       on_hold <------- active              in_refinement -->
  |         ^                |                       |
  |         |                v                       v
  +---> cancelled        completed           ready_for_decomposition -->
  |                                                  |
  +---> active                                       v
                                            in_decomposition --> active
                                                     |
                                                  blocked ---> ready_for_*
```

### Example Custom Feature Workflow (from Epic PRD)

```
draft --> ready_for_refinement_ba --> in_refinement_ba --> ready_for_refinement_tech -->
  |                                       |
  |                                    draft (backward)
  |
  +--> active
  +--> cancelled

... --> in_refinement_tech --> ready_for_task_generation --> in_task_generation -->
            |                         |
         ready_for_refinement_ba   blocked
            (backward)

... --> ready_to_build --> active --> completed
             |                |
          blocked           on_hold --> ready_for_*
```

---

## Appendix B: Dependency Graph

```
Wave 1: Config Foundation
  T-001: StatusMetadata fields
  T-002: Default workflows        (depends on T-001)
  T-003: MultiLevelWorkflow       (depends on T-002)
  T-004: Multi-level parser       (depends on T-003)

Wave 2: Service Extension
  T-005: Level constants
  T-006: workflow.Service changes  (depends on T-004, T-005)

Wave 3: Business Logic
  T-007: Transition types
  T-008: EpicService              (depends on T-006, T-007)
  T-009: FeatureService           (depends on T-006, T-007)

Wave 4: Validation
  T-010: Model validation         (depends on T-006)
  T-011: CLI validators           (depends on T-006)

Wave 5: CLI
  T-012: Service accessors        (depends on T-008, T-009)
  T-013: epic next-status cmd     (depends on T-012)
  T-014: feature next-status cmd  (depends on T-012)
  T-015: epic update refactor     (depends on T-012)
  T-016: feature update refactor  (depends on T-012)
  T-017: workflow validate ext    (depends on T-004)
```

---

*Last Updated*: 2026-02-08
