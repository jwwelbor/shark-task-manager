# E17-F06 Feature Research Report: Progress Command

**Date**: 2026-02-25
**Feature**: E17-F06 Progress Command
**Researcher**: Researcher Agent

---

## Executive Summary

The `shark progress <id>` command is a CLI routing change, not a logic change. The status package already provides all computation infrastructure (`StatusService.GetDashboard`, `GetActionItems`, `GetStatusContext`, `CalculateProgress`). The new command is a direct copy/adapt of `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` with a new `Use` field and the same service call signature. The hidden alias strategy is a one-liner: set `statusCmd.Hidden = true` after F07 migrates `shark status` to a subcommand group. Risk is low. File impact is exactly 2 new files (`progress.go`, `progress_test.go`) with modifications to `status.go`.

---

## 1. Current `shark status <id>` Implementation Analysis

### File Location
`/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` (128 lines)

### What the Current Command Does

The current `shark status` command is a **project-level dashboard**, not a per-entity progress viewer. It accepts an optional epic filter, not a required entity key:

```
shark status                   -- Full project dashboard (all epics)
shark status E05               -- Dashboard filtered to epic E05
shark status E05 F02           -- Dashboard filtered to feature E05-F02 (still dashboard, not per-entity)
shark status E05-F02           -- Same, combined format
```

**Key insight**: The current `shark status` does NOT auto-detect entity types. It uses `ParseListArgs` (not `ParseGetArgs`) and always produces a `StatusDashboard` regardless of the key format. There is no per-entity progress view in the current `status.go`.

### Current Call Chain

```
runStatus
  -> parseStatusRequest(cmd, args)     -- calls ParseListArgs, extracts epicKey
  -> cli.GetStatusService().GetDashboard(ctx, req)   -- StatusService.GetDashboard()
  -> enrichEpicSummaries(dashboard.Epics, cli.GetDisplayService())
  -> outputStatusJSON(dashboard) OR outputStatusTerminal(dashboard)
```

### Service Used
- `cli.GetStatusService()` returns `*status.StatusService` (see `/home/jwwel/projects/shark-task-manager/internal/cli/service_accessors.go` line 221)
- `StatusService.GetDashboard(ctx, *StatusRequest)` returns `*StatusDashboard`
- `status.FormatDashboard(dashboard, noColor)` for terminal rendering

### Existing Test Coverage
`status_test.go` contains a single test that is `t.Skip`ped with a comment explaining it cannot test the command because `status.go` calls `db.InitDB()` directly (bypasses test injection). The progress command should avoid this same pitfall -- the test can be written against `parseProgressRequest` and the output formatters directly.

---

## 2. `status.CalculationService` Capabilities and API

### What `CalculationService` Does (NOT what `shark progress` needs)

Located at `/home/jwwel/projects/shark-task-manager/internal/status/calculation_service.go`.

`CalculationService` is the **cascade recalculation engine** -- it derives feature/epic statuses from their children's statuses and writes those derived statuses back to the database. It is used by task lifecycle commands (start, complete, approve) to propagate status changes up the hierarchy.

**This is NOT the service for `shark progress`.** The `shark progress` command reads dashboards, not runs recalculation.

### What `shark progress` Actually Needs

| Capability | Provider | Location |
|---|---|---|
| Project/epic/feature dashboard data | `StatusService.GetDashboard()` | `/home/jwwel/projects/shark-task-manager/internal/status/status.go` |
| Epic summary with progress % and health | `StatusService.getEpics()` (private via GetDashboard) | Same |
| Active tasks grouped by agent type | `StatusService.getActiveTasks()` | Same |
| Blocked tasks | `StatusService.getBlockedTasks()` | Same |
| Action items categorized by phase | `status.GetActionItems(tasks, cfg)` | `/home/jwwel/projects/shark-task-manager/internal/status/action_items.go` |
| Status context string ("active (development)") | `status.GetStatusContext(statusCounts, cfg)` | `/home/jwwel/projects/shark-task-manager/internal/status/context.go` |
| Weighted/completion progress from status counts | `status.CalculateProgress(statusCounts, cfg)` | `/home/jwwel/projects/shark-task-manager/internal/status/progress.go` |
| Dashboard terminal formatting | `status.FormatDashboard(dashboard, noColor)` | `/home/jwwel/projects/shark-task-manager/internal/status/formatter.go` |
| Epic display mode enrichment | `DisplayService.DetermineEpicDisplayModeByStatus()` | `/home/jwwel/projects/shark-task-manager/internal/cli/service_accessors.go` |

### `StatusService` Accessor

```go
// In internal/cli/service_accessors.go
func GetStatusService() *status.StatusService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    return status.NewStatusService(db)
}
```

`StatusService` takes a raw `*repository.DB` -- it's a legacy service that does not use the repository interface abstraction or the service layer's thin-wrapper pattern. It directly owns all its repositories internally. This is fine for the `progress` command to reuse as-is.

---

## 3. What Can Be Reused vs. What Is New

### Directly Reusable (Zero Changes)

| Component | Location | How Used |
|---|---|---|
| `StatusService.GetDashboard()` | `internal/status/status.go` | Same call as in `status.go` -- `cli.GetStatusService().GetDashboard(ctx, req)` |
| `status.FormatDashboard()` | `internal/status/formatter.go` | Same call as in `status.go` -- `status.FormatDashboard(dashboard, noColor)` |
| `enrichEpicSummaries()` helper | `internal/cli/commands/status.go` | Copy into `progress.go` or extract into `status.go` as package-level func |
| `ParseListArgs()` | `internal/cli/commands/helpers.go` | Same argument parsing as current `status.go` |
| `cli.GetStatusService()` | `internal/cli/service_accessors.go` | No changes needed |
| `cli.GetDisplayService()` | `internal/cli/service_accessors.go` | No changes needed |
| `status.StatusRequest` struct | `internal/status/models.go` | Same DTO |
| `status.StatusDashboard` struct | `internal/status/models.go` | Same output type |
| `DetectEntityType()` | `internal/cli/commands/helpers.go` | Available for optional per-entity routing |
| `ParseGetArgs()` / `scopeInterpreterImpl` | `internal/cli/commands/helpers.go` | Available if per-entity view is added |

### What Is New (Must Be Created)

| Component | Description | Complexity |
|---|---|---|
| `internal/cli/commands/progress.go` | New command file; copy of `status.go` with `Use: "progress [EPIC] [FEATURE]"` and updated documentation | XS |
| `internal/cli/commands/progress_test.go` | Tests for `parseProgressRequest` and argument parsing | S |
| `statusCmd` modification | Mark `statusCmd` as a hidden alias in Phase 2 (after E17-F07 is implemented); in Phase 1, `statusCmd` remains unchanged | XS (1 line) |

### What Does NOT Need to Change

- `internal/status/` package: no changes of any kind
- `internal/cli/service_accessors.go`: no changes
- `internal/cli/services_global.go`: no changes
- `internal/repository/`: no changes
- `internal/services/`: no changes
- Any existing tests: no changes

---

## 4. Alias / Redirect Strategy for Backward Compatibility

### Phase 1 (Current Sprint -- F06 implementation)

Both commands coexist independently:

```
shark status [EPIC] [FEATURE]    -- existing dashboard (unchanged)
shark progress [EPIC] [FEATURE]  -- new command, identical behavior
```

Cobra allows a command to have both a `RunE` handler and subcommands simultaneously. Since E17-F07 (the status subcommand group) is a dependency of F06, F07 must be completed before this step.

**Implementation**: After F07 exists, the F06 implementation step is:

1. Create `progress.go` with `progressCmd` -- a direct copy of `statusCmd`'s `RunE` (pointing to the same `runStatus` function or a copied `runProgress` function).
2. Register `progressCmd` in `init()`.
3. Add a `Deprecated: "Use 'shark progress' instead."` string to `statusCmd` OR mark `statusCmd.Hidden = true`.

### Phase 2 (After F07 is fully complete)

Once `shark status set/advance/options/history` exist as subcommands:

```go
// In status.go init():
// Make shark status <id> a hidden alias for shark progress <id>
statusCmd.Hidden = true  // Or statusCmd.Deprecated = "Use 'shark progress' instead."
```

Cobra's `Hidden = true` keeps the command functional but removes it from `--help` output. `Deprecated` shows a warning message on use. Both preserve backward compatibility.

### Phase 3 (Future cleanup -- not in scope for F06)

Remove `statusCmd`'s `RunE` handler entirely, leaving only subcommands registered by F07.

### Recommended Alias Approach

Use `statusCmd.Deprecated = "Use 'shark progress' to view progress. Use 'shark status set/advance' for transitions."` rather than `Hidden = true`. This:
- Informs users of the migration without breaking scripts
- Shows the deprecation warning in both help and on invocation
- Is a one-line change to `status.go`

---

## 5. File Impact List

### Files Created (New)

| File | Description | Estimated Size |
|---|---|---|
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/progress.go` | New `progressCmd` command, copy/adapt of `status.go` | ~120 lines |
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/progress_test.go` | Tests for `parseProgressRequest`, argument parsing, JSON/terminal routing | ~60 lines |

### Files Modified (Existing)

| File | Change | Risk |
|---|---|---|
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` | Add `statusCmd.Deprecated` or `statusCmd.Hidden = true` in `init()` after F07 is complete; extract `enrichEpicSummaries` to avoid duplication if desired | Very Low |

### Files Unchanged

Everything else. The `internal/status/` package, all service files, all repository files, all other command files remain untouched.

---

## 6. Implementation Blueprint

### `progress.go` Structure

```go
package commands

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/services"
    "github.com/jwwelbor/shark-task-manager/internal/status"
    "github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
    Use:     "progress [EPIC] [FEATURE]",
    Short:   "Show progress, health indicators, and task breakdown",
    GroupID: "status",
    Long: `Display progress rollups, health indicators, action items, and task breakdowns.

Positional Arguments:
  (no args)       Show full project progress dashboard
  EPIC            Show progress for specific epic (e.g., E04)
  EPIC FEATURE    Show progress for specific feature (e.g., E04 F01 or E04-F01)

Examples:
  shark progress                     Show full project progress dashboard
  shark progress E05                 Show progress for epic E05
  shark progress E05 F02             Show progress for feature E05-F02
  shark progress E05-F02             Show progress for feature E05-F02 (combined format)
  shark progress --epic=E05          Flag syntax (still supported)
  shark progress --json              Output as JSON`,
    RunE: runProgress,
}

func init() {
    cli.RootCmd.AddCommand(progressCmd)
    progressCmd.Flags().String("epic", "", "Filter by epic key")
    progressCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
    progressCmd.Flags().Bool("include-archived", false, "Include archived epics/features")
}

func runProgress(cmd *cobra.Command, args []string) error {
    req, err := parseProgressRequest(cmd, args)
    if err != nil {
        return err
    }

    ctx := cmd.Context()
    if ctx == nil {
        var cancel func()
        ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
    }

    dashboard, err := cli.GetStatusService().GetDashboard(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to get progress: %w", err)
    }

    enrichEpicSummaries(dashboard.Epics, cli.GetDisplayService())

    if cli.GlobalConfig.JSON {
        return outputProgressJSON(dashboard)
    }
    return outputProgressTerminal(dashboard)
}

func parseProgressRequest(cmd *cobra.Command, args []string) (*status.StatusRequest, error) {
    _, positionalEpic, _, err := ParseListArgs(args)
    if err != nil {
        return nil, err
    }
    epicKeyFlag, _ := cmd.Flags().GetString("epic")
    recentWindow, _ := cmd.Flags().GetString("recent")
    includeArchived, _ := cmd.Flags().GetBool("include-archived")

    epicKey := epicKeyFlag
    if positionalEpic != nil {
        epicKey = *positionalEpic
    }
    return &status.StatusRequest{
        EpicKey:         epicKey,
        RecentWindow:    recentWindow,
        IncludeArchived: includeArchived,
    }, nil
}

func outputProgressJSON(dashboard *status.StatusDashboard) error {
    data, err := json.MarshalIndent(dashboard, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal progress: %w", err)
    }
    fmt.Println(string(data))
    return nil
}

func outputProgressTerminal(dashboard *status.StatusDashboard) error {
    output := status.FormatDashboard(dashboard, cli.GlobalConfig.NoColor)
    fmt.Print(output)
    return nil
}
```

**Note**: `enrichEpicSummaries` is currently defined in `status.go`. To avoid duplication, it should be extracted to either `status_display_helpers.go` or remain in `status.go` and called from `progress.go` directly (it is package-level, so it is accessible from `progress.go` in the same package).

### `status.go` Modification (Phase 2, after F07)

Add one line to `status.go`'s `init()` function:

```go
func init() {
    cli.RootCmd.AddCommand(statusCmd)
    statusCmd.Flags().String("epic", "", "Filter by epic key")
    statusCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
    statusCmd.Flags().Bool("include-archived", false, "Include archived epics/features")

    // Phase 2: Make shark status (without subcommand) a deprecated alias for shark progress.
    // This preserves backward compatibility while directing users to the new command.
    statusCmd.Deprecated = "Use 'shark progress' to view progress. Use 'shark status set/advance' for status transitions."
}
```

---

## 7. Risks and Recommendations

### Risk 1: `enrichEpicSummaries` duplication

**Probability**: Certain (if not addressed)
**Impact**: Low -- code smell only, not a functional bug

**Description**: `enrichEpicSummaries` is defined in `status.go`. If `progress.go` is a copy, it will call this function from the same package (no issue) but both files become nearly identical.

**Recommendation**: Do NOT duplicate `enrichEpicSummaries`. Since both files are in the `commands` package, `progress.go` can call the function defined in `status.go` directly. The only duplicated code is the `progressCmd` struct definition, flag registration, and the thin `runProgress` → `parseProgressRequest` → `GetDashboard` → format chain.

### Risk 2: `statusCmd.Deprecated` timing (ordering with F07)

**Probability**: Medium (if feature order changes)
**Impact**: Low -- backward compatibility issue only if F07 is missing

**Description**: The feature specifies F06 depends on F07 (status subcommand group) existing first. If F06 is implemented before F07, marking `statusCmd.Deprecated` would break agents using `shark status <key>` before a replacement exists.

**Recommendation**: The `statusCmd.Deprecated` line should be added ONLY after F07 is complete and tested. In Phase 1, `progress.go` is created and registered but `status.go` is left unchanged. Document this in the task description.

### Risk 3: Test coverage gap for `runProgress`

**Probability**: High (existing pattern)
**Impact**: Medium -- the existing `status_test.go` is entirely skipped for the same reason

**Description**: The current `status.go` test is skipped because `GetStatusService()` uses a global DB accessor that cannot be injected in tests. `progress.go` will have the same problem.

**Recommendation**: Write tests for `parseProgressRequest` (pure argument parsing, no DB access) and the JSON/terminal routing conditionals. These tests don't require database access and provide meaningful coverage. Follow the pattern established in `helpers_test.go` and `status_test.go`. Avoid testing `runProgress` end-to-end until a mock injection mechanism exists.

### Risk 4: Feature scope creep -- per-entity progress view

**Probability**: Low
**Impact**: Medium -- additional complexity if per-entity routing is added

**Description**: The feature description mentions `shark progress E18-F05-001` showing task progress context. The current `StatusService.GetDashboard()` does NOT support per-task progress -- it always returns a project/epic-level dashboard. Supporting per-task progress would require `TaskService.GetTask()` plus formatting logic.

**Recommendation**: Implement F06 Phase 1 as a direct copy of the dashboard command. If per-entity routing is desired (epic → EpicService, feature → FeatureService, task → TaskService), that should be a separate sub-feature or a later Phase 2 enhancement. The feature description says "reuses the existing `status.CalculationService` and `status.GetStatusContext` infrastructure" which is consistent with the dashboard approach.

If per-entity routing IS in scope, the implementation would:
1. Parse the key with `ParseGetArgs(args)` → detect entity type
2. For "epic": call `GetDashboard()` with epicKey filter (works today)
3. For "feature": call `GetDashboard()` with featureKey -- **this requires a new `StatusRequest.FeatureKey` field** (does not exist today)
4. For "task": call `TaskService.GetTask()` and format as progress context

This adds complexity and likely requires changes to `StatusService`. Recommend deferring per-entity routing to a follow-up.

### Risk 5: `--field` flag dependency (E17-F02)

**Probability**: Certain
**Impact**: Low -- the feature explicitly lists F02 as a dependency

**Description**: The acceptance criteria include `shark progress E18-F05 --field progress_pct`. The `--field` flag infrastructure does not exist yet; it is implemented by F02.

**Recommendation**: Implement `progress.go` without the `--field` flag. The flag infrastructure will be added by F02 as a persistent root flag or per-command flag. `progress.go` will pick it up automatically once F02 is implemented. No special handling needed in the progress command itself.

---

## 8. Summary of Recommendations

1. **Implement `progress.go` as a near-copy of `status.go`** -- same service calls, same flags, different `Use` field and documentation. Estimated implementation time: 1 hour.

2. **Do not duplicate `enrichEpicSummaries`** -- call the existing function from `status.go` (same package, accessible without import).

3. **Do not mark `statusCmd.Deprecated` in this sprint** -- add the deprecation line only after E17-F07 is implemented and tested. Add a TODO comment in `status.go`'s `init()` noting this.

4. **Write tests only for `parseProgressRequest`** -- avoid testing `runProgress` end-to-end until mock injection is available. Focus test coverage on argument parsing (0 args, 1 arg epic, 2 args epic+feature, combined format, invalid keys).

5. **Defer per-entity routing** (`shark progress E18-F05-001` showing task context) to a follow-up. The immediate implementation scope is the dashboard view with epic/feature filtering.

6. **`--field` flag is a no-op concern** -- F02 will add it as infrastructure; `progress.go` does not need to explicitly handle it.

---

## References

- Feature description: `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F06-progress-command/feature.md`
- Epic research report: `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/research-report.md`
- Current `shark status` command: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go`
- `StatusService`: `/home/jwwel/projects/shark-task-manager/internal/status/status.go`
- `StatusService` accessor: `/home/jwwel/projects/shark-task-manager/internal/cli/service_accessors.go` (line 213-227)
- Status models and types: `/home/jwwel/projects/shark-task-manager/internal/status/models.go`
- `CalculationService`: `/home/jwwel/projects/shark-task-manager/internal/status/calculation_service.go`
- `GetActionItems`: `/home/jwwel/projects/shark-task-manager/internal/status/action_items.go`
- `GetStatusContext`: `/home/jwwel/projects/shark-task-manager/internal/status/context.go`
- `CalculateProgress`: `/home/jwwel/projects/shark-task-manager/internal/status/progress.go`
- `status.FormatDashboard`: `/home/jwwel/projects/shark-task-manager/internal/status/formatter.go`
- Key detection helpers: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/helpers.go` (`DetectEntityType`, `ParseGetArgs`, `ParseListArgs`)
- Aliases pattern: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/aliases.go`
- `enrichEpicSummaries`: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` (line 96-108)
