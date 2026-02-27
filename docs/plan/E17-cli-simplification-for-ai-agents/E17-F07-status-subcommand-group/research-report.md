# Feature Research Report: E17-F07 Status Subcommand Group

## Executive Summary

Feature E17-F07 adds `shark status set`, `shark status advance`, `shark status options`, and `shark status history` subcommands to the existing `shark status` smart dispatcher. All required service-layer infrastructure exists and is production-ready. The primary implementation challenge is the Cobra namespace coexistence strategy: the existing `statusCmd` has `RunE: runStatus` (dashboard display) and must simultaneously support new subcommands. This is a supported Cobra pattern with no structural changes to the parent command required. Implementation requires one new file (`status_group.go`), additions to `status.go` to register the subcommands, and no service-layer changes.

## Research Questions

1. How do the existing status commands work and what can be reused?
2. How does `get.go` implement entity auto-detection and can we reuse that pattern?
3. Which service methods exist for transitions, options, and history?
4. How do we safely add subcommands to `statusCmd` without breaking the existing `RunE`?
5. What files are affected and what is the integration scope?
6. What are the key risks?

## Methodology

Codebase analysis of: `status.go`, `epic_next_status.go`, `epic_set_status.go`, `history.go`, `get.go`, `root.go`, `task_service.go`, `feature_service.go`, `epic_service.go`, `transition_types.go`, `services_global.go`, `service_accessors.go`.

---

## Findings

### Finding 1: Current Implementation Analysis

**Summary:** The existing `shark status <key>` command is a dashboard/progress display smart dispatcher. Four separate entity-specific commands implement transitions. The feature consolidates these into one `status` subcommand namespace.

**Current state of `statusCmd` (internal/cli/commands/status.go):**
```go
var statusCmd = &cobra.Command{
    Use:     "status [EPIC] [FEATURE]",
    GroupID: "status",
    RunE:    runStatus,  // Dashboard display handler
}
```
`runStatus` parses optional epic/feature positional args, calls `cli.GetStatusService().GetDashboard()`, and renders a progress dashboard. This is the existing behavior that MUST be preserved.

**Four pre-existing entity-specific transition commands:**

| Command | File | Service Call |
|---------|------|--------------|
| `shark epic set-status <key> <status>` | `epic_set_status.go` | `epicSvc.TransitionStatus(ctx, epicKey, targetStatus, opts)` |
| `shark epic next-status <key>` | `epic_next_status.go` | `epicSvc.GetNextStatus(ctx, epicKey)` |
| `shark feature set-status <key> <status>` | `feature_set_status.go` | `featureSvc.TransitionStatus(...)` |
| `shark feature next-status <key>` | `feature_next_status.go` | `featureSvc.GetNextStatus(...)` |

These commands are registered under `epicCmd` and `featureCmd` respectively, not under `statusCmd`. E17-F07 creates entity-agnostic equivalents under `statusCmd`.

**Key reusable patterns from `epic_next_status.go`:**
```go
// entityTransitioner interface - central to E17-F07 reuse
type entityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string,
        opts services.TransitionOptions) (*services.TransitionResult, error)
}

// performEntityTransition - reusable for status advance and status set
func performEntityTransition(ctx context.Context, svc entityTransitioner, entityKey, targetStatus string,
    opts services.TransitionOptions, result *EntityNextStatusResult) error

// buildNextStatusResult - reusable for status options
func buildNextStatusResult(entityType string, info *services.NextStatusInfo) EntityNextStatusResult

// EntityNextStatusResult - JSON output struct (reuse or extend for status set/advance)
type EntityNextStatusResult struct {
    EntityType         string                 `json:"entity_type"`
    EntityKey          string                 `json:"entity_key"`
    CurrentStatus      string                 `json:"current_status"`
    AvailableTransitions []EntityTransitionChoice `json:"available_transitions"`
    IsTerminal         bool                   `json:"is_terminal"`
    // ... transition result fields
}
```

**Implications:** New `status_group.go` should import and call these helpers rather than re-implementing transition logic. The `entityTransitioner` interface is already satisfied by all three services.

---

### Finding 2: Entity Auto-Detection Pattern Analysis

**Summary:** `get.go` provides the canonical entity auto-detection pattern via `ParseGetArgs()`. The same pattern applies to `status_group.go`.

**Pattern from `internal/cli/commands/get.go`:**
```go
func ParseGetArgs(args []string) (string, string, error) {
    // Returns: (command_type, normalized_key, error)
    // command_type is one of: "epic", "feature", "task"
    // Detection is based on key format:
    //   "E07"         -> epic
    //   "E07-F01"     -> feature
    //   "F01"         -> feature (short format)
    //   "E07-F01-001" -> task
    //   "T-E07-F01-001" -> task (traditional prefix)
}
```

The detection logic is positional: segment count determines entity type. Key normalization (uppercasing, prefix stripping) happens inside the parser.

**Service accessor map for status_group.go:**
```go
switch entityType {
case "epic":
    svc := cli.GetEpicService()    // service_accessors.go:136
    result, err = svc.TransitionStatus(ctx, key, targetStatus, opts)
case "feature":
    svc := cli.GetFeatureService() // service_accessors.go:167
    result, err = svc.TransitionStatus(ctx, key, targetStatus, opts)
case "task":
    svc := cli.GetTaskService()    // services_global.go:157
    result, err = svc.TransitionStatus(ctx, key, targetStatus, opts)
}
```

All three service types satisfy the `entityTransitioner` interface since all implement `TransitionStatus(ctx, key, targetStatus, opts)` with the same signature.

**For `status history`:** Only `TaskService` has history support currently. The feature spec explicitly limits history to task entities. The command should detect the entity type and return an error if a non-task key is passed (or restrict `status history` to task keys only).

---

### Finding 3: Service Layer Readiness

**Summary:** All required service methods exist across all three entity types. No service-layer changes are needed for E17-F07 implementation.

**TransitionStatus — confirmed present:**

| Service | Method Signature | Notes |
|---------|-----------------|-------|
| `TaskService` | `TransitionStatus(ctx, key, targetStatus, opts TransitionOptions) (*TransitionResult, error)` | Full-featured, includes force/backward checks |
| `EpicService` | `TransitionStatus(ctx, epicKey, targetStatus, opts TransitionOptions) (*TransitionResult, error)` | Used by `epic_set_status.go` |
| `FeatureService` | `TransitionStatus(ctx, featureKey, targetStatus, opts TransitionOptions) (*TransitionResult, error)` | Used by `feature_set_status.go` |

**GetNextStatus — confirmed present:**

| Service | Method Signature | Returns |
|---------|-----------------|---------|
| `TaskService` | `GetNextStatus(ctx, key) (*NextStatusInfo, error)` | `NextStatusInfo` with `AvailableTransitions` |
| `EpicService` | `GetNextStatus(ctx, epicKey) (*NextStatusInfo, error)` | Same struct |
| `FeatureService` | `GetNextStatus(ctx, featureKey) (*NextStatusInfo, error)` | Same struct |

**History — confirmed present for tasks only:**

| Service | Method Signature | Notes |
|---------|-----------------|-------|
| `TaskService` | `GetTaskHistory(ctx, taskKey) ([]*models.TaskHistory, error)` | Requires `historyRepo` set at construction |
| `TaskService` | `ListHistory(ctx, filters HistoryFilters) ([]*models.TaskHistory, error)` | Used by `history.go` global command |

Epic and feature history are not tracked at the service layer. The feature spec limits `status history` to task entities, which aligns with current service capabilities.

**Accessor functions — confirmed present (no additions needed):**

```go
cli.GetEpicService()    // internal/cli/service_accessors.go:136
cli.GetFeatureService() // internal/cli/service_accessors.go:167
cli.GetTaskService()    // internal/cli/services_global.go:157
cli.GetTaskServiceWithHistory() // needed for status history subcommand
```

**Shared types in `transition_types.go`:**
- `TransitionOptions{Force bool, Reason string, DocumentPath string, Agent string}` - input DTO for set/advance
- `TransitionResult{EntityType, EntityKey, FromStatus, ToStatus, Transitioned bool, Message, OrchestratorAction, IsBackward, IsForced, Reason, ChildCount}` - output for set/advance
- `NextStatusInfo{EntityType, EntityKey, CurrentStatus, CurrentPhase, AvailableTransitions []TransitionChoice, IsTerminal bool}` - output for options
- `ErrReasonRequired`, `ErrForceReasonRequired` - sentinel errors for backward transitions

---

### Finding 4: Cobra Namespace Collision Strategy

**Summary:** Cobra fully supports parent commands that have both `RunE` (direct invocation behavior) and registered subcommands. No structural changes to `statusCmd` are required.

**How Cobra resolves the collision:**

When a user runs `shark status E07-F01`, Cobra checks if `E07-F01` matches a known subcommand name. Since it does not match `set`, `advance`, `options`, or `history`, Cobra passes the arguments to the parent command's `RunE = runStatus`. This is the existing dashboard behavior — it continues working unchanged.

When a user runs `shark status set E07-F01-001 in_progress`, Cobra recognizes `set` as a registered subcommand and routes to `statusSetCmd.RunE`.

**No `DisableFlagParsing` or special disambiguation is needed.** The subcommand names (`set`, `advance`, `options`, `history`) are distinct from entity key formats (`E07`, `F01`, `E07-F01-001`).

**Implementation approach in `status_group.go`:**
```go
func init() {
    // Add new subcommands to the existing statusCmd defined in status.go
    statusCmd.AddCommand(statusSetCmd)
    statusCmd.AddCommand(statusAdvanceCmd)
    statusCmd.AddCommand(statusOptionsCmd)
    statusCmd.AddCommand(statusHistoryCmd)
}
```

This requires `statusCmd` in `status.go` to be exported or accessible within the `commands` package. Since both files are in `package commands`, `statusCmd` (lowercase) is accessible within the package. No export needed.

**Precedence verification:** Cobra processes subcommand matching before falling through to `RunE`. Running `shark status set E07 completed` will correctly route to `statusSetCmd`, not `runStatus`. The disambiguation is unambiguous because:
- Subcommand tokens (`set`, `advance`, `options`, `history`) are lowercase English words
- Entity key tokens always start with `E`, `F`, `T` or contain numeric segments

**`shark history` alias:** The existing `historyCmd` at `history.go` is a top-level command registered with `cli.RootCmd`. Per the feature spec, `shark status history` becomes the new primary command and `shark history` becomes a hidden alias. This is achieved by setting `historyCmd.Hidden = true` in `history.go`'s `init()` after the new `statusHistoryCmd` is stable.

---

### Finding 5: Integration Points and File Impact List

**Files requiring modification:**

| File | Change Type | Description |
|------|------------|-------------|
| `internal/cli/commands/status.go` | Minor addition | No change to existing `statusCmd` definition. The subcommands are registered from `status_group.go`. |
| `internal/cli/commands/history.go` | One-line change | Set `historyCmd.Hidden = true` in `init()` once `statusHistoryCmd` is live |

**New file to create:**

| File | Description |
|------|------------|
| `internal/cli/commands/status_group.go` | All four subcommands: `statusSetCmd`, `statusAdvanceCmd`, `statusOptionsCmd`, `statusHistoryCmd` plus their `init()` registration and handler functions |

**Files with zero changes needed:**

| File | Reason |
|------|--------|
| `internal/services/task_service.go` | All methods exist: `TransitionStatus`, `GetNextStatus`, `ListHistory` |
| `internal/services/feature_service.go` | `TransitionStatus`, `GetNextStatus` confirmed present |
| `internal/services/epic_service.go` | `TransitionStatus`, `GetNextStatus` confirmed present |
| `internal/services/transition_types.go` | All DTOs ready: `TransitionOptions`, `TransitionResult`, `NextStatusInfo` |
| `internal/cli/service_accessors.go` | `GetEpicService()`, `GetFeatureService()` already present |
| `internal/cli/services_global.go` | `GetTaskService()`, `GetTaskServiceWithHistory()` already present |
| `internal/cli/commands/epic_set_status.go` | No changes (remains as entity-specific command) |
| `internal/cli/commands/epic_next_status.go` | No changes; its helpers are reused internally |

**`status_group.go` structure outline:**
```go
package commands

import (
    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/services"
    "github.com/spf13/cobra"
)

// Package-level flag vars for each subcommand
var (
    statusSetReason  string
    statusSetForce   bool
    statusSetAgent   string
    statusAdvanceForce bool
    statusAdvanceAgent string
    // history flags (reuse from history.go pattern)
)

// statusSetCmd: shark status set <key> <status>
var statusSetCmd = &cobra.Command{
    Use:     "set <key> <status>",
    Short:   "Set entity to a specific workflow status",
    Args:    cobra.ExactArgs(2),
    RunE:    runStatusSet,
}

// statusAdvanceCmd: shark status advance <key>
var statusAdvanceCmd = &cobra.Command{
    Use:     "advance <key>",
    Short:   "Advance entity to the next workflow status",
    Args:    cobra.ExactArgs(1),
    RunE:    runStatusAdvance,
}

// statusOptionsCmd: shark status options <key>
var statusOptionsCmd = &cobra.Command{
    Use:     "options <key>",
    Short:   "Show available status transitions for an entity",
    Args:    cobra.ExactArgs(1),
    RunE:    runStatusOptions,
}

// statusHistoryCmd: shark status history <key>
var statusHistoryCmd = &cobra.Command{
    Use:     "history <key>",
    Short:   "Show status change history for a task",
    Args:    cobra.ExactArgs(1),
    RunE:    runStatusHistory,
}

func init() {
    // Register flags for each subcommand
    statusSetCmd.Flags().StringVar(&statusSetReason, "reason", "", "Reason for backward or forced transitions")
    statusSetCmd.Flags().BoolVar(&statusSetForce, "force", false, "Bypass workflow validation")
    statusSetCmd.Flags().StringVar(&statusSetAgent, "agent", "", "Agent performing the transition")
    statusAdvanceCmd.Flags().BoolVar(&statusAdvanceForce, "force", false, "Bypass workflow validation")
    statusAdvanceCmd.Flags().StringVar(&statusAdvanceAgent, "agent", "", "Agent performing the transition")

    // Register subcommands with parent statusCmd (defined in status.go, same package)
    statusCmd.AddCommand(statusSetCmd)
    statusCmd.AddCommand(statusAdvanceCmd)
    statusCmd.AddCommand(statusOptionsCmd)
    statusCmd.AddCommand(statusHistoryCmd)
}
```

**`runStatusSet` implementation outline:**
```go
func runStatusSet(cmd *cobra.Command, args []string) error {
    entityType, key, err := ParseGetArgs(args[:1]) // reuse from get.go
    if err != nil {
        return err
    }
    targetStatus := strings.ToLower(strings.TrimSpace(args[1]))

    opts := services.TransitionOptions{
        Force:  statusSetForce,
        Reason: statusSetReason,
        Agent:  statusSetAgent,
    }

    result, err := dispatchTransition(cmd.Context(), entityType, key, targetStatus, opts)
    if err != nil {
        return handleTransitionError(err, key)
    }

    // Idempotent check: if already at target status
    if !result.Transitioned {
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "changed": false,
                "entity_key": result.EntityKey,
                "status": result.ToStatus,
            })
        }
        cli.Info(fmt.Sprintf("Entity %s is already in status '%s'", result.EntityKey, result.ToStatus))
        return nil
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }
    cli.Success(fmt.Sprintf("%s %s: %s -> %s", result.EntityType, result.EntityKey, result.FromStatus, result.ToStatus))
    return nil
}
```

**`dispatchTransition` helper (entity-type routing):**
```go
func dispatchTransition(ctx context.Context, entityType, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
    switch entityType {
    case "epic":
        return cli.GetEpicService().TransitionStatus(ctx, key, targetStatus, opts)
    case "feature":
        return cli.GetFeatureService().TransitionStatus(ctx, key, targetStatus, opts)
    case "task":
        return cli.GetTaskService().TransitionStatus(ctx, key, targetStatus, opts)
    default:
        return nil, fmt.Errorf("unknown entity type: %s", entityType)
    }
}
```

---

### Finding 6: Idempotent Status Set

**Summary:** The `TransitionResult.Transitioned` boolean field provides the mechanism for idempotent behavior.

From `transition_types.go`:
```go
type TransitionResult struct {
    Transitioned bool   `json:"transitioned"` // false if already at target status
    FromStatus   string `json:"from_status"`
    ToStatus     string `json:"to_status"`
    // ...
}
```

When a task is already at the target status, `TransitionStatus` should return `Transitioned: false` (or may return an error — this needs verification during implementation). The `runStatusSet` handler must check this field and return `"changed": false` with exit code 0 rather than treating it as an error. This is important for AI agent idempotency patterns.

---

## Competitive Landscape

Not applicable — this is an internal CLI command consolidation.

## Constraints Identified

- **Technical:** `status history` is task-only due to the service-layer limitation. Epic and feature history are not tracked. The command must validate the entity type and return an explicit error for non-task keys.
- **Business:** The existing `shark status <key>` dashboard display must not break. All current `--json` output contracts remain valid.
- **Cobra architecture:** Subcommand names (`set`, `advance`, `options`, `history`) must not conflict with valid entity key patterns. Confirmed: entity keys always start with letter + digit or contain hyphens + digits, no collision possible.
- **External dependencies:** None. All service methods exist.

## Recommendations

1. **Single new file `status_group.go`**: Place all four subcommand definitions, flags, and handlers in one file. The `init()` function adds subcommands to the existing `statusCmd`. This minimizes file proliferation and keeps the group change contained.

2. **Reuse `ParseGetArgs` from `get.go`**: Do not create a new entity detection function. `ParseGetArgs` is the established pattern and already handles all edge cases (short keys, slugs, case normalization).

3. **Reuse `dispatchTransition` helper inline or as unexported function**: The entity-type routing table is needed by `set`, `advance`, and `options`. Extract it as an unexported helper function `dispatchTransition()` and a matching `dispatchNextStatus()` to avoid duplication across the four handlers.

4. **Reuse `performEntityTransition` from `epic_next_status.go`**: The existing helper function handles the full advance-and-report cycle. Import it by keeping both files in `package commands`.

5. **Handle idempotency explicitly in `runStatusSet`**: Check `result.Transitioned == false` and return `{"changed": false, ...}` with exit code 0. Do not return an error for "already at target status" — this is the AI agent friendly pattern specified in the feature.

6. **Mark `shark history` as hidden alias in a follow-on task**: Set `historyCmd.Hidden = true` in `history.go`. This is a one-line change but should be done as a separate tracked change to avoid breaking existing scripts during transition.

7. **Validate `status history` entity type early**: At the top of `runStatusHistory`, call `ParseGetArgs` and immediately return an error if `entityType != "task"`, with a clear message: `"status history is only supported for tasks; got <entityType>"`.

## Risks and Unknowns

- **Risk: `TransitionStatus` behavior when already at target status.** Probability: Medium. Impact: Medium. The `Transitioned: false` path needs to be verified — does `TransitionStatus` return an error or a result with `Transitioned: false`? The idempotency contract depends on this behavior. Mitigation: Trace through `TaskService.executeStatusTransition()` during implementation to confirm the exact path. If it returns an error, add error-type handling in `runStatusSet`.

- **Risk: Cobra arg disambiguation edge case.** Probability: Low. Impact: High. If a user runs `shark status set` with no further arguments, Cobra routes to `statusSetCmd` and reports "accepts 2 arg(s), received 0". This is correct behavior. The risk is if `set` could ever appear as an entity key — confirmed impossible given key format rules.

- **Risk: `ParseGetArgs` not exported from `get.go`.** Probability: Low (it is already used across files). Impact: Medium. Verify the function is exported (capital P). It is `ParseGetArgs`, confirmed in the research.

- **Unknown: Feature and epic `GetNextStatus` return format.** The `NextStatusInfo` struct and `EntityNextStatusResult` were confirmed in `transition_types.go` and `epic_next_status.go` respectively. However, the exact `AvailableTransitions` field format (whether it matches `TransitionChoice` or a subtype) should be verified during implementation to ensure `buildNextStatusResult()` works with feature and epic inputs.

## References

- `internal/cli/commands/status.go` - Existing `statusCmd` definition
- `internal/cli/commands/get.go` - `ParseGetArgs` entity detection pattern
- `internal/cli/commands/epic_next_status.go` - `entityTransitioner`, `performEntityTransition`, `buildNextStatusResult`
- `internal/cli/commands/epic_set_status.go` - Set-status pattern for reference
- `internal/cli/commands/history.go` - History command (future hidden alias)
- `internal/services/transition_types.go` - `TransitionOptions`, `TransitionResult`, `NextStatusInfo`
- `internal/services/task_service.go` - `TransitionStatus`, `GetNextStatus`, `ListHistory`
- `internal/cli/service_accessors.go` - `GetEpicService()`, `GetFeatureService()`
- `internal/cli/services_global.go` - `GetTaskService()`, `GetTaskServiceWithHistory()`
- `docs/plan/E17-cli-simplification-for-ai-agents/research-report.md` - Epic-level research
- `docs/plan/E17-cli-simplification-for-ai-agents/E17-F07-status-subcommand-group/feature.md` - Feature specification
