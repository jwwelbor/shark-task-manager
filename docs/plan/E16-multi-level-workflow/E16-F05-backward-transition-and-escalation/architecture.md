# Technical Architecture: E16-F05 Backward Transition and Escalation

**Feature**: E16-F05-backward-transition-and-escalation
**Epic**: E16 Multi-Level Workflow System
**Author**: Architect Agent
**Date**: 2026-02-11
**Status**: Draft

---

## 1. Architecture Overview

### Summary

E16-F05 adds `shark epic set-status` and `shark feature set-status` commands with backward transition guards. When moving an entity backward in workflow phase (e.g., `active` -> `ready_for_refinement_tech`), the command requires a `--reason` flag and logs the reason as an audit note. Child entities remain unchanged, with a warning displayed about their current states.

### Design Principle: Consolidate and Reuse

Research identified **extensive existing infrastructure** that E16-F05 must reuse rather than rebuild:

| Existing Component | Location | How E16-F05 Uses It |
|---|---|---|
| `WorkflowConfig.IsBackwardTransition()` | `internal/config/workflow_schema.go:263` | Phase comparison logic - expose via workflow.Service |
| `validation.ValidateReasonForStatusTransition()` | `internal/validation/workflow.go:115` | Reason requirement enforcement |
| `validation.ErrReasonRequired` | `internal/validation/workflow.go:10` | Sentinel error for missing reason |
| `EpicService.TransitionStatus()` | `internal/services/epic_service.go:45` | Extend with reason/backward detection |
| `FeatureService.TransitionStatus()` | `internal/services/feature_service.go:45` | Extend with reason/backward detection |
| `EntityNoteRepository.CreateRejectionNote()` | `internal/repository/entity_note_repository.go` | Audit trail for backward transitions (supports all entity types) |
| `NoteService.AddNote()` | `internal/services/note_service.go:52` | Alternative note creation path |
| `entityTransitioner` interface | `internal/cli/commands/epic_next_status.go:164` | Shared CLI transition helper |
| `performEntityTransition()` | `internal/cli/commands/epic_next_status.go:169` | Shared CLI helper for executing transitions |
| `TransitionResult` | `internal/services/transition_types.go:9` | Extend with backward/reason fields |
| `RequireRejectionReason` config flag | `internal/config/workflow_schema.go:83` | Config-driven reason requirement |

### What Does NOT Need to Be Built

- **Backward transition detection algorithm**: Already exists in `config.WorkflowConfig.IsBackwardTransition()`
- **Phase ordering**: Already defined in `config.getPhaseOrder()` and `validation.PhaseOrder`
- **Rejection note storage**: `entity_notes` table already supports all entity types with `note_type=rejection`
- **Rejection note metadata**: `RejectionNoteMetadata` struct already stores from_status, to_status, document_path
- **CLI output patterns**: `EntityNextStatusResult`, `buildNextStatusResult()`, `printEntityTransitions()` all exist

### High-Level Data Flow

```
User: shark feature set-status E16-F01 ready_for_refinement_tech --reason "API conflict"
       |
       v
CLI: Parse args (key, target_status, --reason, --force)
       |
       v
FeatureService.TransitionStatus(ctx, key, target, opts)
       |
       +--- repo.GetByKey() -> feature at current status
       +--- workflowSvc.IsBackwardTransition(current, target) -> true
       +--- opts.Force? No -> Is reason provided? Yes -> proceed
       +--- workflowSvc.ValidateTransition(current, target) -> valid
       +--- repo.Update(ctx, feature) -> status updated
       +--- noteRepo.CreateRejectionNote() -> audit note created
       +--- Count child tasks -> 15 tasks
       |
       v
TransitionResult { IsBackward: true, Reason: "API conflict", ChildCount: 15 }
       |
       v
CLI: "Feature E16-F01 moved backward: active -> ready_for_refinement_tech"
     "Reason: API conflict"
     "Warning: 15 tasks remain in current states."
```

---

## 2. Service Layer Changes

### 2.1 TransitionOptions Struct (New)

Replace the growing parameter lists with a single options struct. This is the **primary new type** for E16-F05.

**File**: `internal/services/transition_types.go` (extend existing)

```go
// TransitionOptions controls behavior of status transitions.
// Used by EpicService.TransitionStatus() and FeatureService.TransitionStatus().
type TransitionOptions struct {
    // Force bypasses workflow validation (allows any status transition)
    // but still requires Reason for audit trail
    Force bool `json:"force,omitempty"`

    // Reason is required for backward transitions and forced transitions.
    // Only optional for forward transitions within valid workflow flow.
    // Controlled by RequireRejectionReason config flag for backward transitions.
    Reason string `json:"reason,omitempty"`

    // DocumentPath is an optional file path to a detailed rejection document
    DocumentPath string `json:"document_path,omitempty"`

    // Agent identifies which agent/user is performing the transition
    Agent string `json:"agent,omitempty"`
}
```

### 2.2 Extended TransitionResult

**File**: `internal/services/transition_types.go` (modify existing)

Add three fields to the existing `TransitionResult`:

```go
type TransitionResult struct {
    EntityType         string                  `json:"entity_type"`
    EntityKey          string                  `json:"entity_key"`
    FromStatus         string                  `json:"from_status"`
    ToStatus           string                  `json:"to_status"`
    Transitioned       bool                    `json:"transitioned"`
    Message            string                  `json:"message,omitempty"`
    OrchestratorAction *config.PopulatedAction `json:"orchestrator_action"`

    // New fields for E16-F05
    IsBackward bool   `json:"is_backward,omitempty"`   // True if this was a backward transition
    IsForced   bool   `json:"is_forced,omitempty"`     // True if --force was used
    Reason     string `json:"reason,omitempty"`         // Reason for backward or forced transition
    ChildCount int    `json:"child_count,omitempty"`    // Number of child entities unchanged
}
```

### 2.3 workflow.Service.IsBackwardTransition() (New Method)

Expose the existing `WorkflowConfig.IsBackwardTransition()` via the workflow service for DRY access.

**File**: `internal/workflow/service.go` (add method)

```go
// IsBackwardTransition checks if transitioning from one status to another
// moves backward in the workflow phase ordering.
// Delegates to the underlying WorkflowConfig.IsBackwardTransition().
func (s *Service) IsBackwardTransition(fromStatus, toStatus string) (bool, error) {
    return s.workflow.IsBackwardTransition(fromStatus, toStatus)
}
```

### 2.4 EpicService.TransitionStatus() — Extended

**File**: `internal/services/epic_service.go` (modify existing)

The method signature changes from:
```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, force bool) (*TransitionResult, error)
```

To:
```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error)
```

**New logic inserted between validation and update:**

```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    epic, err := s.repo.GetByKey(ctx, epicKey)
    // ... existing nil/error checks ...

    currentStatus := string(epic.Status)

    // Validate transition (unless forced)
    if !opts.Force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return nil, err
        }
    }

    // Normalize target status
    if !opts.Force {
        targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
    }

    // NEW: Enforce reason requirement for forced and backward transitions
    if opts.Force && opts.Reason == "" {
        return nil, fmt.Errorf(
            "--force requires --reason to document why validation was bypassed")
    }

    isBackward, _ := s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
    if isBackward && !opts.Force {
        wf := s.workflowSvc.GetWorkflow()
        requireReason := wf == nil || wf.RequireRejectionReason
        if requireReason && opts.Reason == "" {
            return nil, fmt.Errorf(
                "backward transition from '%s' to '%s' requires --reason flag",
                currentStatus, targetStatus)
        }
    }

    // Perform update
    epic.Status = models.EpicStatus(targetStatus)
    if err := s.repo.Update(ctx, epic); err != nil {
        return nil, fmt.Errorf("failed to update epic status: %w", err)
    }

    // NEW: Log rejection note for backward transitions with reason
    if isBackward && opts.Reason != "" && s.noteRepo != nil {
        _ = s.noteRepo.CreateRejectionNote(ctx, models.EntityTypeEpic, epic.ID,
            0, // no history_id for epics
            currentStatus, targetStatus,
            opts.Reason, opts.Agent, opts.DocumentPath)
    }

    // NEW: Count child features for warning
    var childCount int
    if s.featureRepo != nil {
        features, err := s.featureRepo.ListByEpic(ctx, epic.ID)
        if err == nil {
            childCount = len(features)
        }
    }

    action := s.resolveAction(epic, targetStatus)

    return &TransitionResult{
        EntityType:         "epic",
        EntityKey:          epicKey,
        FromStatus:         currentStatus,
        ToStatus:           targetStatus,
        Transitioned:       true,
        OrchestratorAction: action,
        IsBackward:         isBackward,
        Reason:             opts.Reason,
        ChildCount:         childCount,
    }, nil
}
```

### 2.5 FeatureService.TransitionStatus() — Extended

**File**: `internal/services/feature_service.go` (modify existing)

Same signature change and logic as EpicService, substituting:
- `models.EntityTypeFeature` for entity type
- `s.taskRepo.ListByFeature()` for child count (tasks instead of features)
- `models.FeatureStatus()` for status type

### 2.6 Repository Dependencies on Services

The EpicService and FeatureService need new repository dependencies for:
1. `EntityNoteRepository` — to create rejection notes
2. Child entity counting — `FeatureRepository.ListByEpic()` for epic, `TaskRepository.ListByFeature()` for feature

**EpicService** changes:

```go
type EpicService struct {
    repo        EpicRepository
    workflowSvc *workflow.Service
    noteRepo    EpicNoteRepository    // NEW: for rejection notes
    featureRepo EpicFeatureCounter    // NEW: for child count
}

// EpicNoteRepository defines the note repo interface needed by EpicService.
type EpicNoteRepository interface {
    CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64,
        historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}

// EpicFeatureCounter defines the feature counting interface needed by EpicService.
type EpicFeatureCounter interface {
    ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

func NewEpicService(repo EpicRepository, workflowSvc *workflow.Service, noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter) *EpicService
```

**FeatureService** changes follow the same pattern with `TaskRepository.ListByFeature()`.

**Backward compatibility**: The constructor signature changes. All callers (service_accessors.go, tests) must pass the new dependencies. The note repo and child repo can be `nil` — the service checks before using them (graceful degradation).

### 2.7 Service Accessor Updates

**File**: `internal/cli/service_accessors.go` (modify)

```go
func GetEpicService() *services.EpicService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    noteRepo := repository.NewEntityNoteRepository(db)
    projectRoot, _ := FindProjectRoot()
    workflowSvc := workflow.NewService(projectRoot)
    return services.NewEpicService(epicRepo, workflowSvc, noteRepo, featureRepo)
}

func GetFeatureService() *services.FeatureService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    noteRepo := repository.NewEntityNoteRepository(db)
    projectRoot, _ := FindProjectRoot()
    workflowSvc := workflow.NewService(projectRoot)
    return services.NewFeatureService(featureRepo, workflowSvc, noteRepo, taskRepo)
}
```

---

## 3. CLI Command Changes

### 3.1 New: `shark epic set-status <key> <status>` Command

**File**: `internal/cli/commands/epic_set_status.go` (NEW)

```go
var epicSetStatusCmd = &cobra.Command{
    Use:   "set-status <epic-key> <status>",
    Short: "Set epic status with workflow validation",
    Long: `Set an epic to any valid status with workflow validation.

Backward transitions (moving to an earlier workflow phase) require --reason.

Examples:
  shark epic set-status E16 active                     Forward transition
  shark epic set-status E16 ready_for_refinement_tech --reason "API conflict"  Backward
  shark epic set-status E16 draft --force --reason "DB repair"  Force with reason`,
    Args: cobra.ExactArgs(2),
    RunE: runEpicSetStatus,
}

func init() {
    epicSetStatusCmd.Flags().String("reason", "", "Reason for backward transition (required for backward moves)")
    epicSetStatusCmd.Flags().Bool("force", false, "Bypass workflow validation and reason requirement")
    epicSetStatusCmd.Flags().String("agent", "", "Agent or user performing the transition")
    epicCmd.AddCommand(epicSetStatusCmd)
}

func runEpicSetStatus(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    epicKey := strings.ToUpper(strings.TrimSpace(args[0]))
    targetStatus := strings.TrimSpace(args[1])

    reason, _ := cmd.Flags().GetString("reason")
    force, _ := cmd.Flags().GetBool("force")
    agent, _ := cmd.Flags().GetString("agent")

    epicSvc := cli.GetEpicService()
    result, err := epicSvc.TransitionStatus(ctx, epicKey, targetStatus, services.TransitionOptions{
        Force:  force,
        Reason: reason,
        Agent:  agent,
    })
    if err != nil {
        // Check for backward transition reason requirement
        if strings.Contains(err.Error(), "requires --reason") {
            cli.Error(err.Error())
            os.Exit(3)
        }
        if strings.Contains(err.Error(), "not found") {
            cli.Error(fmt.Sprintf("Epic not found: %s", epicKey))
            os.Exit(1)
        }
        return fmt.Errorf("failed to set epic status: %w", err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    cli.Success(fmt.Sprintf("Epic %s: %s -> %s", result.EntityKey, result.FromStatus, result.ToStatus))
    if result.IsBackward {
        cli.Info(fmt.Sprintf("Reason: %s", result.Reason))
    }
    if result.ChildCount > 0 {
        cli.Warning(fmt.Sprintf("%d features remain in current states.", result.ChildCount))
    }
    displayOrchestratorAction(result.OrchestratorAction)
    return nil
}
```

### 3.2 New: `shark feature set-status <key> <status>` Command

**File**: `internal/cli/commands/feature_set_status.go` (NEW)

Identical pattern to `epic_set_status.go`, substituting:
- `cli.GetFeatureService()` for service
- "Feature" for entity type in messages
- "tasks" for child entity in warning message

### 3.3 Modified: `shark epic next-status` — Add `--reason` Flag

**File**: `internal/cli/commands/epic_next_status.go` (modify)

Changes:
1. Add `--reason` and `--agent` flags to `init()`
2. Update `runEpicNextStatus()` to pass reason via `TransitionOptions`
3. Update `performEntityTransition()` to accept options

The `performEntityTransition()` signature changes from:
```go
func performEntityTransition(ctx context.Context, svc entityTransitioner, _ interface{}, entityKey string, targetStatus string, force bool, result *EntityNextStatusResult) error
```

To:
```go
func performEntityTransition(ctx context.Context, svc entityTransitioner, entityKey string, targetStatus string, opts services.TransitionOptions, result *EntityNextStatusResult) error
```

And the `entityTransitioner` interface updates:
```go
type entityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}
```

### 3.4 Modified: `shark feature next-status` — Add `--reason` Flag

**File**: `internal/cli/commands/feature_next_status.go` (modify)

Same changes as epic next-status.

### 3.5 Display: Backward Transition Warning

When `result.IsBackward` is true and not in JSON mode, CLI commands display:
```
✓ Feature E16-F01: active -> ready_for_refinement_tech
  Reason: API contract conflicts with database schema
  Warning: 15 tasks remain in current states.
```

When in JSON mode, the full `TransitionResult` (including `is_backward`, `reason`, `child_count`) is returned.

---

## 4. No Database Changes Required

The `entity_notes` table already supports rejection notes for all entity types:
- `entity_type` IN ('epic', 'feature', 'task')
- `note_type` = 'rejection'
- `metadata` JSON blob stores `RejectionNoteMetadata` (from_status, to_status, document_path)

The `EntityNoteRepository.CreateRejectionNote()` already accepts any entity type. No schema migration needed.

---

## 5. Backward Compatibility

### 5.1 TransitionStatus Signature Change

The `TransitionStatus()` method signature changes from `(ctx, key, status, force)` to `(ctx, key, status, TransitionOptions)`. This is a **breaking change** to the service API.

**Impact**: All callers must be updated:
- `internal/cli/commands/epic_next_status.go` — change `force` to `TransitionOptions{Force: force}`
- `internal/cli/commands/feature_next_status.go` — same
- `internal/services/epic_service_test.go` — update test calls
- `internal/services/feature_service_test.go` — update test calls

### 5.2 Constructor Signature Change

`NewEpicService()` and `NewFeatureService()` gain new parameters. All callers update:
- `internal/cli/service_accessors.go` — pass note repo and child repo
- Test files — pass mock repos or nil

### 5.3 entityTransitioner Interface Change

The shared `entityTransitioner` interface changes. All users update:
- `performEntityTransition()` call sites

### 5.4 Forward Transitions Unaffected

Forward transitions (`draft` -> `active`, etc.) behave exactly as before. The `--reason` flag is optional for forward transitions and simply ignored.

### 5.5 Force Flag Behavior Change

Previously `--force` bypassed everything silently. Now `--force` still bypasses workflow validation but **requires `--reason`** for audit trail. Direct database access is the escape hatch for true emergencies (e.g., `sqlite3 shark-tasks.db "UPDATE epics SET status='draft' WHERE key='E16'"`).

---

## 6. File Change Inventory

### New Files

| File | Purpose | Size |
|------|---------|------|
| `internal/cli/commands/epic_set_status.go` | `shark epic set-status` command | S (~80 lines) |
| `internal/cli/commands/feature_set_status.go` | `shark feature set-status` command | S (~80 lines) |
| `internal/cli/commands/epic_set_status_test.go` | Tests for epic set-status | M (~150 lines) |
| `internal/cli/commands/feature_set_status_test.go` | Tests for feature set-status | M (~150 lines) |

### Modified Files

| File | Changes | Impact |
|------|---------|--------|
| `internal/services/transition_types.go` | Add `TransitionOptions` struct; add `IsBackward`, `Reason`, `ChildCount` to `TransitionResult` | Low — additive |
| `internal/workflow/service.go` | Add `IsBackwardTransition()` method | Low — additive |
| `internal/services/epic_service.go` | Extend `TransitionStatus()` with backward detection, reason validation, note creation, child count; add new repo dependencies; update constructor | Medium |
| `internal/services/feature_service.go` | Same as epic_service | Medium |
| `internal/cli/service_accessors.go` | Pass note repo and child repo to constructors | Low |
| `internal/cli/commands/epic_next_status.go` | Add `--reason` flag; update `performEntityTransition()` and `entityTransitioner` interface signature | Medium |
| `internal/cli/commands/feature_next_status.go` | Add `--reason` flag; update call to `performEntityTransition()` | Low |
| `internal/services/epic_service_test.go` | Update for new signature; add backward transition tests | Medium |
| `internal/services/feature_service_test.go` | Same as epic | Medium |
| `internal/workflow/service_new_methods_test.go` or new test file | Test `IsBackwardTransition()` wrapper | Low |

### Files NOT Modified

| File | Reason |
|------|--------|
| `internal/db/db.go` | No schema changes — entity_notes already supports all types |
| `internal/repository/entity_note_repository.go` | Already has `CreateRejectionNote()` for all entity types |
| `internal/validation/workflow.go` | Existing functions can be used, but service uses `WorkflowConfig.IsBackwardTransition()` directly |
| `internal/config/workflow_schema.go` | `IsBackwardTransition()` and `RequireRejectionReason` already exist |
| `internal/models/entity_note.go` | `EntityNote` model already supports all entity types |

---

## 7. Task Breakdown

### Task 1: TransitionOptions and Extended TransitionResult (XS)
- Add `TransitionOptions` struct to `transition_types.go`
- Add `IsBackward`, `Reason`, `ChildCount` fields to `TransitionResult`
- No behavior change yet — just types
- Tests: Verify JSON serialization of new fields

### Task 2: workflow.Service.IsBackwardTransition() Wrapper (XS)
- Add `IsBackwardTransition(fromStatus, toStatus string) (bool, error)` to `workflow.Service`
- Delegates to `s.workflow.IsBackwardTransition()`
- Tests: Verify delegation, various phase pairs

### Task 3: EpicService Backward Transition Support (M)
- Add `EpicNoteRepository` and `EpicFeatureCounter` interfaces
- Update `NewEpicService()` constructor to accept new deps
- Change `TransitionStatus()` signature to use `TransitionOptions`
- Add backward detection, reason enforcement, rejection note creation, child count
- Update `internal/cli/service_accessors.go` for `GetEpicService()`
- Tests: backward with reason (succeeds), backward without reason (fails), forward without reason (succeeds), force bypass, child count, note creation (mocked repos)

### Task 4: FeatureService Backward Transition Support (M)
- Same as Task 3, substituting Feature types
- Child count uses `TaskRepository.ListByFeature()` instead of `FeatureRepository.ListByEpic()`
- Update `internal/cli/service_accessors.go` for `GetFeatureService()`
- Tests: Same coverage as epic

### Task 5: Update epic/feature next-status CLI Commands (S)
- Update `entityTransitioner` interface signature to use `TransitionOptions`
- Update `performEntityTransition()` to accept and pass `TransitionOptions`
- Add `--reason` and `--agent` flags to epic and feature next-status commands
- Display backward transition warning when `result.IsBackward`
- Tests: Verify flags parsed, backward warning displayed

### Task 6: Epic set-status CLI Command (S)
- Create `internal/cli/commands/epic_set_status.go`
- Parse key, status, --reason, --force, --agent
- Call `epicSvc.TransitionStatus()` with `TransitionOptions`
- Format output (JSON and human-readable)
- Display backward warning and child count
- Tests: Valid transition, backward without reason (error), backward with reason (success), force, JSON output

### Task 7: Feature set-status CLI Command (S)
- Create `internal/cli/commands/feature_set_status.go`
- Same pattern as epic set-status
- Tests: Same coverage

### Task 8: Integration Testing & Documentation (S)
- End-to-end test: backward transition with reason -> verify note created
- End-to-end test: backward transition without reason -> verify error
- End-to-end test: forward transition without reason -> succeeds
- Verify JSON output includes all new fields
- Update CLI_REFERENCE.md with new commands

---

## 8. Dependency Graph

```
Task 1: TransitionOptions & Extended TransitionResult
    |
    +---> Task 2: workflow.Service.IsBackwardTransition()
    |        |
    |        +---> Task 3: EpicService Backward Transition Support
    |        |        |
    |        |        +---> Task 5: Update next-status CLI Commands
    |        |        |        |
    |        |        |        +---> Task 6: Epic set-status CLI Command
    |        |        |
    |        |        +---> Task 6 (also depends on Task 3)
    |        |
    |        +---> Task 4: FeatureService Backward Transition Support
    |                 |
    |                 +---> Task 5 (also depends on Task 4)
    |                 |
    |                 +---> Task 7: Feature set-status CLI Command
    |
    All Tasks ---> Task 8: Integration Testing
```

**Parallelizable**: Tasks 3 and 4 can be done in parallel after Tasks 1 and 2. Tasks 6 and 7 can be done in parallel after Task 5.

---

## 9. Consolidation Opportunities Identified

### 9.1 Backward Detection: Three → One Entry Point

Currently three implementations exist:
1. `config.WorkflowConfig.IsBackwardTransition()` — phase-based, returns `(bool, error)`
2. `validation.IsBackwardTransition()` — phase-based, returns `bool` only
3. `config.Config.IsBackwardTransition()` — weight-based, returns `bool` only

**E16-F05 consolidates**: All new code uses `workflow.Service.IsBackwardTransition()` which delegates to #1 (the canonical, level-aware implementation). The task-level code in `internal/repository/task_repository.go` already uses `r.workflow.IsBackwardTransition()` (same method, accessed directly). Implementations #2 and #3 are legacy and should be deprecated in a future cleanup pass (out of scope for F05).

### 9.2 EpicService and FeatureService: Nearly Identical

The `TransitionStatus()` methods on EpicService and FeatureService differ in only:
- Entity type string ("epic" vs "feature")
- Repository type (EpicRepository vs FeatureRepository)
- Status type (models.EpicStatus vs models.FeatureStatus)
- Child counting (features vs tasks)
- Orchestrator action resolution

A shared `baseTransitionService` could be extracted, but this adds abstraction for only 2 consumers. **Decision: Keep separate for now.** If E16-F06 or future features add more entity-level transition logic, revisit extraction. The duplication is ~30 lines of transition logic, which is acceptable.

### 9.3 Task set-status: Future DRY Opportunity

The existing `shark task set-status` in `task.go` uses the **legacy repository pattern** (direct repo access, business logic in command handler). Ideally it should be refactored to use a `TaskService.TransitionStatus()` method with `TransitionOptions`, matching the epic/feature pattern. This is **out of scope** for F05 but should be tracked as a future E15 refactoring task.

---

## 10. Testing Strategy

### 10.1 Service Tests (Mocked Repositories)

| Test | Description |
|------|-------------|
| `TestTransitionStatus_ForwardNoReason` | Forward transition without reason succeeds |
| `TestTransitionStatus_BackwardWithReason` | Backward transition with reason succeeds |
| `TestTransitionStatus_BackwardNoReason` | Backward transition without reason returns error |
| `TestTransitionStatus_BackwardForceWithReason` | Backward transition with force + reason succeeds |
| `TestTransitionStatus_ForceNoReason` | Force without reason returns error |
| `TestTransitionStatus_BackwardReasonNotRequired` | Config `RequireRejectionReason=false` allows backward without reason |
| `TestTransitionStatus_RejectionNoteCreated` | Rejection note created for backward + reason |
| `TestTransitionStatus_ChildCount` | ChildCount populated from repo |
| `TestTransitionStatus_NilNoteRepo` | Graceful degradation when note repo is nil |
| `TestTransitionStatus_NotFound` | Entity not found returns error |
| `TestTransitionStatus_InvalidTransition` | Invalid status transition returns error |

### 10.2 CLI Command Tests (Mocked Services)

| Test | Description |
|------|-------------|
| `TestEpicSetStatus_BasicForward` | Forward transition, success output |
| `TestEpicSetStatus_BackwardWithReason` | Backward with --reason, success + warning |
| `TestEpicSetStatus_BackwardNoReason` | Error message about --reason requirement |
| `TestEpicSetStatus_ForceWithReason` | --force --reason succeeds |
| `TestEpicSetStatus_ForceNoReason` | --force without --reason returns error |
| `TestEpicSetStatus_JSONOutput` | JSON includes is_backward, reason, child_count |
| `TestEpicSetStatus_NotFound` | Exit code 1 |
| `TestFeatureSetStatus_*` | Same test matrix for feature |

### 10.3 Workflow Service Tests (No DB)

| Test | Description |
|------|-------------|
| `TestIsBackwardTransition_EpicLevel` | Epic workflow phase comparison |
| `TestIsBackwardTransition_FeatureLevel` | Feature workflow phase comparison |
| `TestIsBackwardTransition_SpecialPhases` | "any"/"blocked" phases return false |

---

## 11. Open Questions Resolution

### Q1: Should `--reason` be required at the Cobra level?

**Decision**: No. The `--reason` flag is always optional at the Cobra level. The service layer determines if it's required based on transition direction. This matches the existing task pattern and avoids Cobra limitation where you can't conditionally require flags.

### Q2: Should we create entity_history tables for epic/feature?

**Decision**: No, not in F05 scope. The `entity_notes` table with `note_type=rejection` provides sufficient audit trail for backward transitions. Full entity history tracking (all transitions, not just backward) is deferred to a future feature. The `CreateRejectionNote()` method accepts `historyID=0` when no history record exists.

### Q3: Should child entity statuses be listed in the warning?

**Decision**: No. The warning says "N tasks/features remain in current states." It does not enumerate each child entity's status. This keeps the output concise. Users can run `shark list <key>` to see child statuses if needed.

### Q4: Should the set-status command auto-select from next-status options?

**Decision**: No. The `set-status` command takes an explicit target status as a positional argument. It validates the transition via workflow but does not auto-select. `next-status` handles the interactive/auto-select flow.

### Q5: Should `--force` require `--reason`?

**Decision**: Yes. `--force` bypasses workflow validation (allows any status transition) but requires `--reason` to document why validation was bypassed. This provides accountability for all exceptional transitions. Direct database access (`sqlite3`) is the escape hatch for true emergencies where even the CLI reason requirement is impractical.

---

*Last Updated*: 2026-02-11
