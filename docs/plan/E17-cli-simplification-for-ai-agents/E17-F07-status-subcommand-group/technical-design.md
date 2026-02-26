# Technical Design: E17-F07 Status Subcommand Group

**Feature**: E17-F07 Status Subcommand Group
**Status**: Ready for Development
**Last Updated**: 2026-02-25

---

## 1. Architecture Overview

### 1.1 How `status_group.go` Integrates with the Existing Cobra Structure

The existing `statusCmd` in `status.go` is defined as a top-level Cobra command with `RunE: runStatus` (dashboard display). It is registered with `cli.RootCmd` in `status.go`'s `init()` function. This existing definition and behaviour remain completely unchanged.

`status_group.go` is a new file in `internal/cli/commands/`. Its `init()` function calls `statusCmd.AddCommand(...)` four times, adding the subcommands `set`, `advance`, `options`, and `history` as children of the existing `statusCmd`. Because both files are in `package commands`, `statusCmd` (unexported) is accessible from `status_group.go` without any changes to `status.go`.

Cobra's command resolution algorithm handles the coexistence transparently:

- `shark status E07-F01` — `E07-F01` does not match any registered subcommand name, so Cobra routes to `statusCmd.RunE = runStatus` (dashboard, unchanged).
- `shark status set E07-F01-001 in_progress` — `set` is a registered subcommand, so Cobra routes to `statusSetCmd.RunE = runStatusSet`.

No `DisableFlagParsing`, special disambiguation, or changes to `statusCmd`'s definition are required. The subcommand token names (`set`, `advance`, `options`, `history`) are unambiguous relative to entity key patterns (`E07`, `E07-F01`, `E07-F01-001`) — they never start with a letter followed by a digit.

### 1.2 Data Flow

```
shark status set E17-F07-001 in_development
    │
    ▼
statusSetCmd.RunE = runStatusSet(cmd, args)
    │
    ├─ ParseGetArgs([]string{"E17-F07-001"}) → ("task", "E17-F07-001", nil)
    │
    ├─ dispatchTransition(ctx, "task", "E17-F07-001", "in_development", opts)
    │       │
    │       └─ cli.GetTaskService().TransitionStatus(ctx, key, targetStatus, opts)
    │               │
    │               └─ TaskService (service layer — validates, transitions, persists)
    │
    └─ format output (JSON or human-readable)
```

---

## 2. Command Implementations

### 2.1 `shark status set <key> <status>`

**Purpose**: Set an entity to a specific workflow status. Idempotent — returns success if already at target status.

**Cobra definition**:

```go
var statusSetCmd = &cobra.Command{
    Use:   "set <key> <status>",
    Short: "Set entity to a specific workflow status",
    Long: `Set a task, feature, or epic to the specified workflow status.

Entity type is auto-detected from the key format:
  E07          → epic
  E07-F01      → feature
  E07-F01-001  → task

This command is idempotent: if the entity is already in the target status,
it returns success (exit 0) with "changed": false in JSON output.

Backward transitions require --reason. Use --force to bypass workflow validation.

Examples:
  shark status set E07-F01-001 in_development
  shark status set E07-F01-001 in_development --agent=backend-1
  shark status set E07 active --reason="Reactivating"
  shark status set E07-F01 in_review --force --reason="Admin override"
  shark status set E07-F01-001 in_development --json`,
    Args: cobra.ExactArgs(2),
    RunE: runStatusSet,
}
```

**Flags**:
- `--reason string` — Reason for backward or forced transitions (required for backward)
- `--force bool` — Bypass workflow validation (requires `--reason`)
- `--agent string` — Agent performing the transition
- `--notes string` — Transition notes (stored in history)

**Handler skeleton**:

```go
func runStatusSet(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    entityType, key, err := ParseGetArgs(args[:1])
    if err != nil {
        return fmt.Errorf("invalid key format: %w", err)
    }
    targetStatus := strings.ToLower(strings.TrimSpace(args[1]))

    reason, _ := cmd.Flags().GetString("reason")
    force, _ := cmd.Flags().GetBool("force")
    agent, _ := cmd.Flags().GetString("agent")
    notes, _ := cmd.Flags().GetString("notes")

    opts := services.TransitionOptions{
        Force:  force,
        Reason: reason,
        Agent:  agent,
        Notes:  notes,
    }

    // Step 2: Dispatch to service
    result, err := dispatchTransition(cmd.Context(), entityType, key, targetStatus, opts)
    if err != nil {
        return handleStatusTransitionError(err, key, entityType)
    }

    // Step 3: Format output
    // Idempotent case: already at target status
    if !result.Transitioned {
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "changed":    false,
                "entity_type": result.EntityType,
                "entity_key":  result.EntityKey,
                "status":      result.ToStatus,
                "message":     fmt.Sprintf("%s %s is already in status '%s'", result.EntityType, result.EntityKey, result.ToStatus),
            })
        }
        cli.Info(fmt.Sprintf("%s %s is already in status '%s' (no change)", result.EntityType, result.EntityKey, result.ToStatus))
        return nil
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "changed":     true,
            "entity_type": result.EntityType,
            "entity_key":  result.EntityKey,
            "from_status": result.FromStatus,
            "to_status":   result.ToStatus,
            "is_backward": result.IsBackward,
            "is_forced":   result.IsForced,
            "reason":      result.Reason,
            "child_count": result.ChildCount,
            "orchestrator_action": result.OrchestratorAction,
        })
    }

    cli.Success(fmt.Sprintf("%s %s: %s → %s", result.EntityType, result.EntityKey, result.FromStatus, result.ToStatus))
    if result.IsBackward && result.Reason != "" {
        cli.Info(fmt.Sprintf("Reason: %s", result.Reason))
    }
    if result.IsForced {
        cli.Warning("Workflow validation was bypassed with --force")
    }
    if result.ChildCount > 0 {
        cli.Warning(fmt.Sprintf("%d child entities remain in current states.", result.ChildCount))
    }
    displayOrchestratorAction(result.OrchestratorAction)
    return nil
}
```

### 2.2 `shark status advance <key>`

**Purpose**: Move an entity to its next status in the workflow. When multiple transitions are valid, uses the first (primary) one unless `--to` overrides.

**Cobra definition**:

```go
var statusAdvanceCmd = &cobra.Command{
    Use:   "advance <key>",
    Short: "Advance entity to the next workflow status",
    Long: `Advance a task, feature, or epic to its next workflow status.

When multiple next statuses are valid, the first (primary) transition is used.
Use --to to specify which next status when the workflow is ambiguous.

Examples:
  shark status advance E07-F01-001
  shark status advance E07-F01-001 --to=ready_for_code_review
  shark status advance E07-F01-001 --agent=backend-1
  shark status advance E07 --force --reason="Admin advance"
  shark status advance E07-F01-001 --json`,
    Args: cobra.ExactArgs(1),
    RunE: runStatusAdvance,
}
```

**Flags**:
- `--to string` — Target status when multiple transitions are valid
- `--force bool` — Bypass workflow validation
- `--reason string` — Reason for backward or forced transitions
- `--agent string` — Agent performing the transition
- `--notes string` — Transition notes

**Handler skeleton**:

```go
func runStatusAdvance(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    entityType, key, err := ParseGetArgs(args)
    if err != nil {
        return fmt.Errorf("invalid key format: %w", err)
    }

    to, _ := cmd.Flags().GetString("to")
    force, _ := cmd.Flags().GetBool("force")
    reason, _ := cmd.Flags().GetString("reason")
    agent, _ := cmd.Flags().GetString("agent")
    notes, _ := cmd.Flags().GetString("notes")

    opts := services.TransitionOptions{
        Force:  force,
        Reason: reason,
        Agent:  agent,
        Notes:  notes,
    }

    // Step 2: Get available next statuses from service
    info, err := dispatchNextStatus(cmd.Context(), entityType, key)
    if err != nil {
        return handleStatusTransitionError(err, key, entityType)
    }

    nextStatusResult := buildNextStatusResult(entityType, info)

    // Terminal status check
    if info.IsTerminal {
        nextStatusResult.Message = fmt.Sprintf("Entity is in terminal status '%s' — no transitions available", info.CurrentStatus)
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(nextStatusResult)
        }
        cli.Warning(nextStatusResult.Message)
        return nil
    }

    // No transitions available
    if len(info.AvailableTransitions) == 0 {
        nextStatusResult.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(nextStatusResult)
        }
        cli.Warning(nextStatusResult.Message)
        return nil
    }

    // Resolve target status
    targetStatus := info.AvailableTransitions[0].TargetStatus
    if to != "" {
        targetStatus = strings.ToLower(strings.TrimSpace(to))
        if !force {
            valid := false
            for _, t := range info.AvailableTransitions {
                if strings.EqualFold(t.TargetStatus, targetStatus) {
                    valid = true
                    targetStatus = t.TargetStatus
                    break
                }
            }
            if !valid {
                cli.Error(fmt.Sprintf("'%s' is not a valid transition from '%s'", targetStatus, info.CurrentStatus))
                fmt.Println("Valid transitions:")
                printEntityTransitions(nextStatusResult.AvailableTransitions)
                fmt.Println("Use --force to bypass workflow validation")
                return fmt.Errorf("invalid transition target: %s", targetStatus)
            }
        }
    }

    // Step 3: Perform transition via entity-routing helper
    return performEntityTransition(cmd.Context(), dispatchableService(entityType), key, targetStatus, opts, nextStatusResult)
}
```

**Note on `dispatchableService`**: The `performEntityTransition` helper in `epic_next_status.go` accepts an `entityTransitioner` interface. A small adapter or a direct call to `dispatchTransition` that returns the `*EntityNextStatusResult` mutation will be used. See Section 4.

### 2.3 `shark status options <key>`

**Purpose**: Show current status and valid transitions for any entity. Read-only. Replaces the `--preview` pattern.

**Cobra definition**:

```go
var statusOptionsCmd = &cobra.Command{
    Use:   "options <key>",
    Short: "Show available status transitions for an entity",
    Long: `Display the current status and all valid workflow transitions for a task, feature, or epic.

This is a read-only command. No changes are made.

Examples:
  shark status options E07-F01-001
  shark status options E07-F01
  shark status options E07
  shark status options E07-F01-001 --json`,
    Args: cobra.ExactArgs(1),
    RunE: runStatusOptions,
}
```

**Flags**: None (read-only command). `--json` is inherited from root.

**Handler skeleton**:

```go
func runStatusOptions(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    entityType, key, err := ParseGetArgs(args)
    if err != nil {
        return fmt.Errorf("invalid key format: %w", err)
    }

    // Step 2: Get status info from service (read-only)
    info, err := dispatchNextStatus(cmd.Context(), entityType, key)
    if err != nil {
        return handleStatusTransitionError(err, key, entityType)
    }

    // Step 3: Format output
    result := buildNextStatusResult(entityType, info)

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "entity_type":          result.EntityType,
            "entity_key":           result.EntityKey,
            "current_status":       result.CurrentStatus,
            "current_phase":        result.CurrentPhase,
            "is_terminal":          info.IsTerminal,
            "available_transitions": result.AvailableTransitions,
        })
    }

    // Human-readable output
    fmt.Printf("Entity:   %s (%s)\n", result.EntityKey, result.EntityType)
    fmt.Printf("Status:   %s", result.CurrentStatus)
    if result.CurrentPhase != "" {
        fmt.Printf(" (phase: %s)", result.CurrentPhase)
    }
    fmt.Println()

    if info.IsTerminal {
        cli.Info("This is a terminal status — no further transitions are available.")
        return nil
    }

    if len(result.AvailableTransitions) == 0 {
        cli.Warning(fmt.Sprintf("No valid transitions from '%s'", result.CurrentStatus))
        return nil
    }

    fmt.Println("\nAvailable transitions:")
    printEntityTransitions(result.AvailableTransitions)
    fmt.Printf("\nUse 'shark status advance %s' to advance, or\n", result.EntityKey)
    fmt.Printf("    'shark status set %s <status>' to target a specific status.\n", result.EntityKey)
    return nil
}
```

### 2.4 `shark status history <key>`

**Purpose**: Show the status change log for a task. Task-only due to service-layer limitations.

**Cobra definition**:

```go
var statusHistoryCmd = &cobra.Command{
    Use:   "history <task-key>",
    Short: "Show status change history for a task",
    Long: `Display the status change log for a task.

Note: history is only supported for tasks. Passing an epic or feature key
will return an error with a clear message.

Examples:
  shark status history E07-F01-001
  shark status history T-E07-F01-001
  shark status history E07-F01-001 --json`,
    Args: cobra.ExactArgs(1),
    RunE: runStatusHistory,
}
```

**Flags**: `--json` inherited from root.

**Handler skeleton**:

```go
func runStatusHistory(cmd *cobra.Command, args []string) error {
    // Step 1: Parse and validate entity type — tasks only
    entityType, key, err := ParseGetArgs(args)
    if err != nil {
        return fmt.Errorf("invalid key format: %w", err)
    }
    if entityType != "task" {
        return fmt.Errorf("status history is only supported for tasks; got %s key '%s'", entityType, args[0])
    }

    // Step 2: Call service
    svc := cli.GetTaskServiceWithHistory()
    histories, err := svc.GetTaskHistory(cmd.Context(), key)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            cli.Error(fmt.Sprintf("Task %s not found", key))
            os.Exit(1)
        }
        return fmt.Errorf("failed to retrieve history for task %s: %w", key, err)
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(histories)
    }

    if len(histories) == 0 {
        cli.Info(fmt.Sprintf("No history records found for task %s", key))
        return nil
    }

    headers := []string{"Timestamp", "Old Status", "New Status", "Agent", "Notes"}
    var rows [][]string
    for _, h := range histories {
        oldStatus := "(initial)"
        if h.OldStatus != nil {
            oldStatus = *h.OldStatus
        }
        agent := ""
        if h.Agent != nil {
            agent = *h.Agent
        }
        notes := ""
        if h.Notes != nil {
            notes = *h.Notes
        }
        rows = append(rows, []string{
            h.Timestamp.Format("2006-01-02 15:04:05"),
            oldStatus,
            h.NewStatus,
            agent,
            notes,
        })
    }

    cli.OutputTable(headers, rows)
    cli.Info(fmt.Sprintf("Showing %d history records for task %s", len(histories), key))
    return nil
}
```

**Note on `GetTaskHistory` vs `ListHistory`**: `GetTaskHistory(ctx, taskKey)` retrieves history for a specific task by key. This is the appropriate method for `status history <task-key>`. The existing `historyCmd` uses `ListHistory(ctx, filters)` for project-wide filtered queries — that behaviour remains unchanged.

---

## 3. Entity Auto-Detection Reuse from `ParseGetArgs`

`ParseGetArgs` in `get.go` is the canonical entity detection function. It is exported (capital P) and accessible within the same package. It accepts a `[]string` of positional args and returns `(entityType string, key string, err error)`.

All four subcommands call `ParseGetArgs` for key normalization and entity type detection:

```go
entityType, key, err := ParseGetArgs(args[:1])
// entityType: "epic" | "feature" | "task"
// key: normalized uppercase key (e.g. "E07-F01-001")
```

`ParseGetArgs` handles:
- Case normalization (lowercased input → normalized output)
- `T-` prefix stripping for task keys
- Short key formats (`F01`, `E07-F01-001`)
- Slug suffixes in keys

No new entity detection logic is introduced in `status_group.go`. All detection delegates to `ParseGetArgs`.

---

## 4. Service Layer Integration

### 4.1 Entity-Routing Dispatch Helpers

Two private helper functions are defined in `status_group.go` to route service calls based on entity type:

```go
// dispatchTransition routes a status transition to the correct service.
// All three services satisfy the entityTransitioner interface defined in epic_next_status.go.
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

// dispatchNextStatus routes a GetNextStatus query to the correct service.
// Returns services.NextStatusInfo which is consumed by buildNextStatusResult.
func dispatchNextStatus(ctx context.Context, entityType, key string) (*services.NextStatusInfo, error) {
    switch entityType {
    case "epic":
        return cli.GetEpicService().GetNextStatus(ctx, key)
    case "feature":
        return cli.GetFeatureService().GetNextStatus(ctx, key)
    case "task":
        return cli.GetTaskService().GetNextStatus(ctx, key)
    default:
        return nil, fmt.Errorf("unknown entity type: %s", entityType)
    }
}
```

### 4.2 Reuse of Existing Package-Level Helpers

The following helpers defined in `epic_next_status.go` are in `package commands` and are reused without modification:

| Helper | Used By | Purpose |
|--------|---------|---------|
| `performEntityTransition(ctx, svc, key, target, opts, result)` | `runStatusAdvance` | Executes transition, logs output, handles JSON/human |
| `buildNextStatusResult(entityType, info)` | `runStatusAdvance`, `runStatusOptions` | Constructs `*EntityNextStatusResult` from `*NextStatusInfo` |
| `printEntityTransitions(transitions)` | `runStatusOptions`, `runStatusAdvance` error path | Formats transition list for human output |
| `displayOrchestratorAction(action)` | `runStatusSet`, `runStatusAdvance` | Prints orchestrator action hint |
| `EntityNextStatusResult` struct | `runStatusAdvance`, `runStatusOptions` | JSON/human output structure |
| `entityTransitioner` interface | Used by `performEntityTransition` | Already satisfied by all three services |

For `runStatusAdvance`, the `performEntityTransition` helper expects an `entityTransitioner`. Rather than passing the service directly, a small adapter is used:

```go
// entityTransitionerFunc adapts a function to the entityTransitioner interface.
// This avoids importing the concrete service types in status_group.go.
type entityTransitionerFunc func(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)

func (f entityTransitionerFunc) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
    return f(ctx, key, targetStatus, opts)
}
```

This allows `runStatusAdvance` to pass a closure to `performEntityTransition`:

```go
adapter := entityTransitionerFunc(func(ctx context.Context, k, ts string, o services.TransitionOptions) (*services.TransitionResult, error) {
    return dispatchTransition(ctx, entityType, k, ts, o)
})
return performEntityTransition(cmd.Context(), adapter, key, targetStatus, opts, nextStatusResult)
```

### 4.3 Service Methods Referenced

| Service | Method | Used By |
|---------|--------|---------|
| `TaskService` | `TransitionStatus(ctx, key, targetStatus, opts) (*TransitionResult, error)` | `dispatchTransition` case "task" |
| `FeatureService` | `TransitionStatus(ctx, key, targetStatus, opts) (*TransitionResult, error)` | `dispatchTransition` case "feature" |
| `EpicService` | `TransitionStatus(ctx, key, targetStatus, opts) (*TransitionResult, error)` | `dispatchTransition` case "epic" |
| `TaskService` | `GetNextStatus(ctx, key) (*NextStatusInfo, error)` | `dispatchNextStatus` case "task" |
| `FeatureService` | `GetNextStatus(ctx, key) (*NextStatusInfo, error)` | `dispatchNextStatus` case "feature" |
| `EpicService` | `GetNextStatus(ctx, key) (*NextStatusInfo, error)` | `dispatchNextStatus` case "epic" |
| `TaskService` | `GetTaskHistory(ctx, taskKey) ([]*models.TaskHistory, error)` | `runStatusHistory` |

**Service accessor functions used** (all pre-existing):
- `cli.GetTaskService()` — `internal/cli/services_global.go`
- `cli.GetTaskServiceWithHistory()` — `internal/cli/services_global.go`
- `cli.GetFeatureService()` — `internal/cli/service_accessors.go`
- `cli.GetEpicService()` — `internal/cli/service_accessors.go`

No service methods are added or modified. No service accessor functions are added or modified.

---

## 5. Idempotency Contract Design

### 5.1 Specification

`shark status set` MUST be idempotent: if the entity is already at the target status, the command returns exit code 0 with `"changed": false` in JSON output. No history record is created.

### 5.2 Implementation

The `TransitionResult.Transitioned` boolean field is the mechanism. Research identified that `TransitionStatus` sets `Transitioned: false` when the entity is already at the target status (no-op path) rather than returning an error.

**During implementation, the exact path through `TaskService.executeStatusTransition()` must be verified.** Two possible behaviours:

**Behaviour A** (expected): `TransitionStatus` returns `(*TransitionResult{Transitioned: false, ...}, nil)`:
```go
result, err := dispatchTransition(cmd.Context(), entityType, key, targetStatus, opts)
if err != nil {
    return handleStatusTransitionError(err, key, entityType)
}
if !result.Transitioned {
    // idempotent no-op path — return success with changed: false
    ...
    return nil
}
```

**Behaviour B** (fallback): `TransitionStatus` returns an error containing "already" or "same status":
```go
result, err := dispatchTransition(cmd.Context(), entityType, key, targetStatus, opts)
if err != nil {
    if isAlreadyAtStatusError(err) {
        // treat as no-op, return success with changed: false
        ...
        return nil
    }
    return handleStatusTransitionError(err, key, entityType)
}
```

The `isAlreadyAtStatusError` helper would check `strings.Contains(err.Error(), "already")` or a typed sentinel error if one exists.

The developer must verify which behaviour is in effect during implementation and choose the matching path. Both are handled without modifying the service layer.

### 5.3 JSON Output Contract

**No-op (already at target)**:
```json
{
  "changed": false,
  "entity_type": "task",
  "entity_key": "E07-F01-001",
  "status": "in_development",
  "message": "task E07-F01-001 is already in status 'in_development'"
}
```

**Successful transition**:
```json
{
  "changed": true,
  "entity_type": "task",
  "entity_key": "E07-F01-001",
  "from_status": "todo",
  "to_status": "in_development",
  "is_backward": false,
  "is_forced": false,
  "reason": "",
  "child_count": 0,
  "orchestrator_action": null
}
```

The `changed` field is the key discriminator for AI agents. Exit code is always 0 for both cases.

---

## 6. Cobra Namespace Strategy: Coexistence with Existing `statusCmd`

### 6.1 Mechanism

Cobra supports parent commands that have both `RunE` (direct invocation) and registered subcommands. The disambiguation is:

1. Cobra checks if the first positional argument matches a registered subcommand name.
2. If yes, route to subcommand.
3. If no, fall through to parent command's `RunE`.

**Routing table**:

| Input | Cobra Routes To |
|-------|----------------|
| `shark status` | `runStatus` (dashboard, no args) |
| `shark status E07` | `runStatus` (E07 is not a subcommand name) |
| `shark status E07-F01` | `runStatus` (E07-F01 is not a subcommand name) |
| `shark status set ...` | `runStatusSet` |
| `shark status advance ...` | `runStatusAdvance` |
| `shark status options ...` | `runStatusOptions` |
| `shark status history ...` | `runStatusHistory` |

Subcommand names (`set`, `advance`, `options`, `history`) are pure lowercase English words. Entity keys always contain uppercase letters and digits (`E07`, `E07-F01-001`). There is no possible collision.

### 6.2 Registration

All registration happens inside `status_group.go`'s `init()` function. No changes to `status.go`:

```go
func init() {
    // Register flags for statusSetCmd
    statusSetCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
    statusSetCmd.Flags().Bool("force", false, "Bypass workflow validation")
    statusSetCmd.Flags().String("agent", "", "Agent performing the transition")
    statusSetCmd.Flags().String("notes", "", "Transition notes")

    // Register flags for statusAdvanceCmd
    statusAdvanceCmd.Flags().String("to", "", "Target status (when multiple transitions are valid)")
    statusAdvanceCmd.Flags().Bool("force", false, "Bypass workflow validation")
    statusAdvanceCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
    statusAdvanceCmd.Flags().String("agent", "", "Agent performing the transition")
    statusAdvanceCmd.Flags().String("notes", "", "Transition notes")

    // statusOptionsCmd has no additional flags (read-only)
    // statusHistoryCmd has no additional flags

    // Add subcommands to the existing statusCmd (defined in status.go, same package)
    statusCmd.AddCommand(statusSetCmd)
    statusCmd.AddCommand(statusAdvanceCmd)
    statusCmd.AddCommand(statusOptionsCmd)
    statusCmd.AddCommand(statusHistoryCmd)
}
```

### 6.3 `shark history` Hidden Alias

The existing `historyCmd` in `history.go` is a project-wide history command (epic/feature filter support, `--limit`, `--offset`, `--format=csv`). It is NOT equivalent to `shark status history` — the latter is task-key-specific.

Per the feature spec, `shark history` becomes a hidden alias once `shark status history` is stable. This is a one-line change to `history.go` done as a separate tracked task:

```go
func init() {
    historyCmd.Hidden = true  // becomes hidden alias for shark status history
    // ... rest of init unchanged
}
```

This is a follow-on change and is NOT part of the initial `status_group.go` implementation. It is tracked as a separate task in the feature's task list.

---

## 7. Error Handling and Exit Codes

### 7.1 Standard Exit Codes

| Exit Code | Meaning | Trigger |
|-----------|---------|---------|
| 0 | Success | Transition completed, or already at target status (idempotent) |
| 1 | Not found | Entity key does not exist in database |
| 2 | System error | Database failure, unexpected internal error |
| 3 | Invalid state | Workflow validation failed, backward transition without reason |

### 7.2 Error Handler Helper

A private helper `handleStatusTransitionError` handles the common error cases for `set` and `advance`:

```go
func handleStatusTransitionError(err error, key, entityType string) error {
    if strings.Contains(err.Error(), "not found") {
        cli.Error(fmt.Sprintf("%s %s not found", entityType, key))
        os.Exit(1)
    }
    if errors.Is(err, services.ErrReasonRequired) || errors.Is(err, services.ErrForceReasonRequired) {
        cli.Error(err.Error())
        cli.Info("Use --reason=<text> to provide a reason for this transition.")
        cli.Info("Use --force --reason=<text> to bypass workflow validation.")
        os.Exit(3)
    }
    if strings.Contains(err.Error(), "invalid transition") || strings.Contains(err.Error(), "not allowed") {
        cli.Error(fmt.Sprintf("Invalid transition: %v", err))
        cli.Info(fmt.Sprintf("Use 'shark status options %s' to see valid transitions.", key))
        os.Exit(3)
    }
    return fmt.Errorf("failed to transition %s %s: %w", entityType, key, err)
}
```

**Design note on `os.Exit`**: The existing codebase uses `os.Exit()` in command handlers (see `epic_set_status.go` lines 73-74, 76-78 and `epic_next_status.go` lines 69-70). This is the established pattern. New commands follow the same pattern for consistency, while acknowledging the documented preference to return errors. If the project adopts the return-error pattern in a future refactoring, these commands should be updated then.

### 7.3 Error Messages for Common Cases

| Scenario | Message (non-JSON) | JSON Field |
|----------|-------------------|------------|
| Task not found | `task E07-F01-001 not found` | N/A (exit 1) |
| Invalid transition | `Invalid transition: ...` + `Use 'shark status options E07-F01-001' to see valid transitions.` | N/A (exit 3) |
| Backward without reason | `backward transition requires --reason` | N/A (exit 3) |
| Already at status | `task E07-F01-001 is already in status 'in_development' (no change)` | `{"changed": false, ...}` |
| history on non-task | `status history is only supported for tasks; got feature key 'E07-F01'` | N/A (exit 2) |

---

## 8. Test Strategy

### 8.1 Test File Location

```
internal/cli/commands/status_group_test.go
```

### 8.2 Test Approach

CLI command tests use mocked services — never real databases. Since `status_group.go` calls `cli.GetEpicService()`, `cli.GetFeatureService()`, and `cli.GetTaskService()`, the test approach requires either:

- **Option A**: Interface-based overrides via package-level variables in `service_accessors.go` / `services_global.go` (if override hooks exist).
- **Option B**: Test the handler functions directly by constructing minimal Cobra commands with mock implementations injected via function-level variables.

The recommended approach for this codebase is to test the handler functions by constructing the command and verifying output, following the pattern used in other CLI test files.

### 8.3 Test Cases

**`runStatusSet` tests**:

| Test Case | Setup | Expected |
|-----------|-------|---------|
| Happy path — task set | `dispatchTransition` returns `{Transitioned: true, FromStatus: "todo", ToStatus: "in_development"}` | Exit 0, `changed: true` in JSON, success message in human |
| Idempotent — already at status | `dispatchTransition` returns `{Transitioned: false, ToStatus: "in_development"}` | Exit 0, `changed: false` in JSON, info message in human |
| Not found | `dispatchTransition` returns error containing "not found" | Exit 1 |
| Backward without reason | `dispatchTransition` returns `ErrReasonRequired` | Exit 3 |
| Invalid transition | `dispatchTransition` returns error containing "invalid transition" | Exit 3, suggests `status options` |
| Epic set | `entityType = "epic"`, routes to `GetEpicService()` | Exit 0, entity_type = "epic" |
| Feature set | `entityType = "feature"`, routes to `GetFeatureService()` | Exit 0, entity_type = "feature" |

**`runStatusAdvance` tests**:

| Test Case | Setup | Expected |
|-----------|-------|---------|
| Auto-advance | One transition available | Exit 0, transition performed |
| `--to` flag specifies valid target | Multiple transitions, `--to=ready_for_code_review` | Transitions to specified target |
| `--to` flag specifies invalid target | No matching transition | Exit 3, lists valid options |
| Terminal status | `GetNextStatus` returns `IsTerminal: true` | Exit 0, warning message |
| No transitions | `GetNextStatus` returns empty `AvailableTransitions` | Exit 0, warning message |

**`runStatusOptions` tests**:

| Test Case | Setup | Expected |
|-----------|-------|---------|
| Task with transitions | Returns `NextStatusInfo` with 2 transitions | JSON includes `available_transitions`, human table |
| Terminal status | `IsTerminal: true` | Indicates no transitions |
| Feature entity | `entityType = "feature"` | Routes to `GetFeatureService().GetNextStatus` |

**`runStatusHistory` tests**:

| Test Case | Setup | Expected |
|-----------|-------|---------|
| Task history | Returns `[]*models.TaskHistory` | Exit 0, table or JSON output |
| Empty history | Returns `[]` | Exit 0, info message |
| Non-task key (feature) | `entityType = "feature"` | Error message, no service call |
| Task not found | Service returns "not found" error | Exit 1 |

**`ParseGetArgs` routing tests** (integration with entity auto-detection):

| Input | Expected entityType |
|-------|-------------------|
| `["E07"]` | `"epic"` |
| `["E07-F01"]` | `"feature"` |
| `["E07-F01-001"]` | `"task"` |
| `["T-E07-F01-001"]` | `"task"` |
| `["e07-f01-001"]` | `"task"` (case normalized) |

### 8.4 What is NOT Tested in CLI Tests

The following is verified at the service layer (service tests, not CLI tests):
- Workflow validation logic
- Backward transition detection
- History record creation
- Database persistence

---

## 9. File List

### 9.1 New Files

| File | Description |
|------|-------------|
| `internal/cli/commands/status_group.go` | All four subcommand definitions (`statusSetCmd`, `statusAdvanceCmd`, `statusOptionsCmd`, `statusHistoryCmd`), their handler functions (`runStatusSet`, `runStatusAdvance`, `runStatusOptions`, `runStatusHistory`), private helpers (`dispatchTransition`, `dispatchNextStatus`, `handleStatusTransitionError`, `entityTransitionerFunc`), and `init()` registration |
| `internal/cli/commands/status_group_test.go` | Unit tests for all four handlers |

### 9.2 Modified Files

| File | Change | Scope |
|------|--------|-------|
| `internal/cli/commands/history.go` | Add `historyCmd.Hidden = true` to `init()` | One line, follow-on task |

### 9.3 Files with Zero Changes

| File | Reason |
|------|--------|
| `internal/cli/commands/status.go` | `statusCmd` is not modified; subcommands are registered from `status_group.go` |
| `internal/cli/commands/epic_set_status.go` | Remains as entity-specific command, unchanged |
| `internal/cli/commands/epic_next_status.go` | Helpers reused in-package; no changes needed |
| `internal/cli/commands/feature_set_status.go` | Remains as entity-specific command, unchanged |
| `internal/cli/commands/feature_next_status.go` | Remains as entity-specific command, unchanged |
| `internal/services/task_service.go` | All required methods exist |
| `internal/services/feature_service.go` | All required methods exist |
| `internal/services/epic_service.go` | All required methods exist |
| `internal/services/transition_types.go` | All DTOs already defined |
| `internal/cli/service_accessors.go` | All accessors already present |
| `internal/cli/services_global.go` | All accessors already present |

---

## 10. `status_group.go` Complete Structure Reference

```
package commands

import (
    "context"
    "errors"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/services"
    "github.com/spf13/cobra"
)

// ─── Command definitions ────────────────────────────────────────────────────

var statusSetCmd      = &cobra.Command{ ... }
var statusAdvanceCmd  = &cobra.Command{ ... }
var statusOptionsCmd  = &cobra.Command{ ... }
var statusHistoryCmd  = &cobra.Command{ ... }

// ─── init: register flags and subcommands ────────────────────────────────────

func init() { ... }

// ─── Handler functions ────────────────────────────────────────────────────────

func runStatusSet(cmd *cobra.Command, args []string) error { ... }
func runStatusAdvance(cmd *cobra.Command, args []string) error { ... }
func runStatusOptions(cmd *cobra.Command, args []string) error { ... }
func runStatusHistory(cmd *cobra.Command, args []string) error { ... }

// ─── Private helpers ─────────────────────────────────────────────────────────

// dispatchTransition routes to the correct service based on entity type.
func dispatchTransition(ctx context.Context, entityType, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) { ... }

// dispatchNextStatus routes GetNextStatus to the correct service.
func dispatchNextStatus(ctx context.Context, entityType, key string) (*services.NextStatusInfo, error) { ... }

// handleStatusTransitionError maps service errors to CLI exit codes.
func handleStatusTransitionError(err error, key, entityType string) error { ... }

// entityTransitionerFunc adapts a dispatch function to the entityTransitioner interface.
type entityTransitionerFunc func(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)

func (f entityTransitionerFunc) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
    return f(ctx, key, targetStatus, opts)
}
```

---

## 11. Key Design Decisions and Rationale

| Decision | Rationale |
|----------|-----------|
| Single file `status_group.go` for all four subcommands | Keeps the change contained; all group-related code is colocated; minimises file proliferation |
| Reuse `ParseGetArgs` from `get.go` | Established pattern with all edge cases handled; avoids divergence |
| Reuse `buildNextStatusResult`, `performEntityTransition` from `epic_next_status.go` | Both files are in `package commands`; helpers work across entity types; no duplication |
| `entityTransitionerFunc` adapter | Bridges the dispatch routing table to `performEntityTransition`'s interface without changing the helper |
| Idempotency via `Transitioned` field, not via error suppression | AI-agent friendly: exit 0 with `changed: false` is more composable than suppressing errors |
| `status history` restricted to tasks | Reflects current service-layer reality; clear error message guides users to correct command |
| No changes to `statusCmd` in `status.go` | Cobra's subcommand resolution handles coexistence; zero risk to existing dashboard behaviour |
| `history.go` hidden as a follow-on task | Decouples the two changes; avoids making `shark history` invisible before `shark status history` is validated |
| Context timeout set at handler level | Consistent with `epic_set_status.go` and `epic_next_status.go` patterns; 30-second timeout |

---

## 12. Implementation Checklist

Before declaring this feature complete, the developer must verify:

- [ ] `TransitionStatus` idempotency behaviour confirmed (Behaviour A or B — see Section 5.2)
- [ ] All four subcommands return correct exit codes for all error scenarios
- [ ] `shark status --help` shows all four subcommands
- [ ] `shark status set E07-F01-001 <current-status>` returns exit 0 with `changed: false`
- [ ] `shark status advance E07-F01-001` with terminal status returns exit 0 with warning
- [ ] `shark status history E07-F01` returns clear error (non-task key)
- [ ] Existing `shark status E07` (dashboard) is unaffected
- [ ] `make fmt && make lint && make test` passes green
- [ ] All tests in `status_group_test.go` pass
