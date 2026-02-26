# E17-F06 Technical Design: Progress Command

**Feature**: E17-F06 Progress Command
**Author**: Architect Agent
**Date**: 2026-02-25
**Status**: Ready for Development

---

## 1. Architecture Overview

### Summary

`shark progress` is a new CLI command that exposes the existing project dashboard under a dedicated, unambiguous name. It resolves the namespace collision that will exist once E17-F07 introduces `shark status set/advance/options/history` as a subcommand group for status transitions.

The implementation is a **near-copy of `status.go`** with updated command metadata and renamed internal functions. No new service logic is needed. All computation is already provided by:

- `StatusService.GetDashboard()` -- aggregates project, epic, feature, and task data
- `status.FormatDashboard()` -- terminal rendering
- `enrichEpicSummaries()` -- display mode enrichment (already in `commands` package, accessible without import)

The command follows the thin-wrapper pattern: parse arguments, call `cli.GetStatusService().GetDashboard()`, format output. There is no business logic in the command file.

### Layered Call Chain

```
shark progress [EPIC] [FEATURE] [--json] [--recent=...] [--include-archived]
       |
       v
runProgress(cmd, args)                      -- commands/progress.go
       |
       v
parseProgressRequest(cmd, args)             -- argument parsing helper
       |
       v
cli.GetStatusService().GetDashboard(ctx, req)  -- service_accessors.go -> status/status.go
       |
       v
enrichEpicSummaries(dashboard.Epics, ...)   -- commands/status.go (same package)
       |
       v
outputProgressJSON(dashboard)               -- JSON: json.MarshalIndent
  OR outputProgressTerminal(dashboard)      -- Terminal: status.FormatDashboard()
```

### Dependency Graph (No New Dependencies)

```
progress.go (new)
├── cli.GetStatusService()     -- internal/cli/service_accessors.go (no change)
│   └── status.NewStatusService(*repository.DB)
│       ├── repository.EpicRepository
│       ├── repository.FeatureRepository
│       ├── repository.TaskRepository
│       └── repository.TaskHistoryRepository
├── cli.GetDisplayService()    -- internal/cli/service_accessors.go (no change)
├── enrichEpicSummaries()      -- internal/cli/commands/status.go (same package, no import)
├── status.FormatDashboard()   -- internal/status/formatter.go (no change)
├── status.StatusRequest       -- internal/status/models.go (no change)
├── status.StatusDashboard     -- internal/status/models.go (no change)
└── ParseListArgs()            -- internal/cli/commands/helpers.go (no change)
```

---

## 2. Command Implementation

### Command Definition (`progress.go`)

```go
package commands

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/status"
    "github.com/spf13/cobra"
)

// progressCmd is the dedicated command for viewing entity progress rollups,
// health indicators, task breakdowns, and action items.
//
// Once E17-F07 is complete, "shark status <id>" becomes a deprecated alias
// for this command, resolving the namespace collision with status transitions.
var progressCmd = &cobra.Command{
    Use:     "progress [EPIC] [FEATURE]",
    Short:   "Show progress, health indicators, and task breakdown",
    GroupID: "status",
    Long: `Display a progress dashboard showing project progress, health indicators,
active tasks, and blocked items.

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
  shark progress --recent=7d         Include recent completions (7 days)
  shark progress --json              Output as JSON`,
    RunE: runProgress,
}

func init() {
    cli.RootCmd.AddCommand(progressCmd)
    progressCmd.Flags().String("epic", "", "Filter by epic key")
    progressCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
    progressCmd.Flags().Bool("include-archived", false, "Include archived epics/features")
}
```

### Command Handler

```go
// runProgress executes the progress command.
// Pattern: parse -> call service -> format output. No business logic here.
func runProgress(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    req, err := parseProgressRequest(cmd, args)
    if err != nil {
        return err
    }

    // Ensure a usable context with timeout when none is provided (e.g., in tests)
    ctx := cmd.Context()
    if ctx == nil {
        var cancel func()
        ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
    }

    // Step 2: Call service
    dashboard, err := cli.GetStatusService().GetDashboard(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to get progress: %w", err)
    }

    // Enrich with display mode metadata (uses DisplayService, no additional DB queries)
    enrichEpicSummaries(dashboard.Epics, cli.GetDisplayService())

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return outputProgressJSON(dashboard)
    }
    return outputProgressTerminal(dashboard)
}
```

### Argument Parsing Helper

```go
// parseProgressRequest builds a StatusRequest from command arguments and flags.
// Supports three invocation forms:
//   shark progress                      -> no filter
//   shark progress E05                  -> epic filter
//   shark progress E05 F02              -> epic + feature filter
//   shark progress E05-F02              -> combined format (feature filter)
func parseProgressRequest(cmd *cobra.Command, args []string) (*status.StatusRequest, error) {
    _, positionalEpic, _, err := ParseListArgs(args)
    if err != nil {
        return nil, err
    }

    epicKeyFlag, _ := cmd.Flags().GetString("epic")
    recentWindow, _ := cmd.Flags().GetString("recent")
    includeArchived, _ := cmd.Flags().GetBool("include-archived")

    // Positional argument takes precedence over flag (matches status.go behavior)
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
```

### Output Formatters

```go
// outputProgressJSON marshals the dashboard to indented JSON.
func outputProgressJSON(dashboard *status.StatusDashboard) error {
    data, err := json.MarshalIndent(dashboard, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal progress: %w", err)
    }
    fmt.Println(string(data))
    return nil
}

// outputProgressTerminal renders the dashboard with rich terminal formatting.
func outputProgressTerminal(dashboard *status.StatusDashboard) error {
    output := status.FormatDashboard(dashboard, cli.GlobalConfig.NoColor)
    fmt.Print(output)
    return nil
}
```

### `enrichEpicSummaries` Sharing

`enrichEpicSummaries` is already defined in `status.go` at package level (package `commands`). Since `progress.go` is in the same package, it calls `enrichEpicSummaries` directly with no import needed and no duplication. This is by design: the function must NOT be copied into `progress.go`.

---

## 3. Service Reuse

### StatusService.GetDashboard()

The entire computation workload is handled by `StatusService.GetDashboard()`, which is accessed via the existing global accessor:

```go
// In internal/cli/service_accessors.go -- no changes needed
func GetStatusService() *status.StatusService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    return status.NewStatusService(db)
}
```

`GetDashboard` assembles:
- `ProjectSummary` -- overall counts and completion percentage
- `[]*EpicSummary` -- per-epic progress %, health (healthy/warning/critical), task counts
- `map[string][]*TaskInfo` -- active tasks grouped by agent type
- `[]*BlockedTaskInfo` -- blocked tasks with reasons and age
- `[]*CompletionInfo` -- recent completions (when `--recent` window specified)

No changes to `StatusService`, `StatusRequest`, `StatusDashboard`, or any of their dependencies.

### CalculationService Clarification

`status.CalculationService` is the **cascade recalculation engine** used by task lifecycle commands (start, complete, approve) to propagate status changes up the hierarchy. It writes derived statuses back to the database. It is NOT used by the progress command. The feature description's mention of reusing `CalculationService` refers to the broader `internal/status/` package, specifically `GetActionItems`, `GetStatusContext`, and `CalculateProgress` -- all of which are already called internally by `StatusService.GetDashboard()`.

---

## 4. Backward Compatibility

### Phase 1: Coexistence (this sprint -- F06 implementation)

Both commands exist and produce identical output:

```
shark status [EPIC] [FEATURE]    -- unchanged, existing behavior
shark progress [EPIC] [FEATURE]  -- new command, same dashboard behavior
```

No changes to `status.go` in this phase. The `statusCmd` is left fully functional.

### Phase 2: Deprecation (after E17-F07 is complete and tested)

Once E17-F07 exists and `shark status set/advance/options/history` are registered as subcommands, add one line to `status.go`'s `init()`:

```go
func init() {
    cli.RootCmd.AddCommand(statusCmd)
    statusCmd.Flags().String("epic", "", "Filter by epic key")
    statusCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
    statusCmd.Flags().Bool("include-archived", false, "Include archived epics/features")

    // TODO(E17-F06/Phase2): Add after E17-F07 is complete and tested.
    // This makes "shark status [EPIC]" a deprecated alias for "shark progress [EPIC]"
    // while keeping the command functional for backward compatibility.
    // statusCmd.Deprecated = "Use 'shark progress' to view progress. Use 'shark status set/advance' for status transitions."
}
```

The deprecation line is included as a commented-out TODO so the developer implementing Phase 2 knows exactly what to do. It must not be uncommented until E17-F07 is complete.

**Why `Deprecated` over `Hidden`:**
- `Deprecated` prints a warning on invocation, informing users of the migration path
- `Hidden` silently removes the command from help but gives no guidance
- Both preserve full backward compatibility -- the command continues to work

### Phase 3: Future Cleanup (not in F06 scope)

After agents and users have migrated: remove `statusCmd.RunE`, leaving only the E17-F07 subcommands registered to `statusCmd`. Out of scope for this feature.

---

## 5. --field Integration Points

The `--field` flag infrastructure is implemented by E17-F02, which is an explicit dependency of F06. The progress command does not need any special handling to support `--field`.

### How It Works

E17-F02 registers `--field` as either a persistent root-level flag or a per-command flag. The progress command's output path is:

```
runProgress -> outputProgressJSON(dashboard)
```

When `--field progress_pct` is passed, the E17-F02 infrastructure intercepts the JSON output and extracts the requested field from the `StatusDashboard` struct before printing.

### Field Extraction Points in StatusDashboard

The `StatusDashboard` JSON structure provides these natural extraction targets:

| `--field` value        | JSON path                          | Example value |
|------------------------|------------------------------------|---------------|
| `summary.overall_progress` | `summary.overall_progress`     | `78.5`        |
| `summary.tasks.completed`  | `summary.tasks.completed`      | `42`          |
| `summary.tasks.total`      | `summary.tasks.total`          | `60`          |
| `summary.blocked_count`    | `summary.blocked_count`        | `3`           |
| `epics[0].health`          | `epics[0].health`              | `"healthy"`   |
| `epics[0].progress_percent` | `epics[0].progress_percent`  | `65.0`        |

The exact `--field` path syntax and extraction mechanism are defined by E17-F02. The progress command exposes all these fields through its standard JSON output (`--json`) and requires no additional logic to support `--field`.

### No Special Progress Command Work Needed

The progress command implementation is complete as described in Section 2. When E17-F02 is implemented, `--field` will work with `shark progress` automatically, because:
1. The JSON output is a well-structured `StatusDashboard` struct
2. E17-F02's interception mechanism applies to all commands that produce JSON output
3. No `progress.go` changes are required when E17-F02 is implemented

---

## 6. Error Handling

### Argument Parsing Errors

`parseProgressRequest` delegates to `ParseListArgs`, which returns typed errors for invalid arguments. These propagate to Cobra's error handler, which displays the error message and exits with a non-zero code. No special handling is needed in `runProgress`.

### Service Errors

```go
dashboard, err := cli.GetStatusService().GetDashboard(ctx, req)
if err != nil {
    return fmt.Errorf("failed to get progress: %w", err)
}
```

The error is wrapped with business context ("failed to get progress") and returned to Cobra. Specific error types that `GetDashboard` can return:

| Error Condition | Source | Behavior |
|---|---|---|
| Invalid epic key format | `StatusRequest.Validate()` | Returns `fmt.Errorf("invalid epic key format: ...")` |
| Invalid timeframe | `StatusRequest.Validate()` | Returns `fmt.Errorf("invalid timeframe: ...")` |
| Database query failure | SQL operations in `StatusService` | Returns `fmt.Errorf("query ...: %w", err)` |
| Context cancellation/timeout | 5-second timeout or caller cancellation | Returns `context.DeadlineExceeded` or `context.Canceled` |

All errors propagate cleanly through the `fmt.Errorf("failed to get progress: %w", err)` wrapper. No `os.Exit()` calls -- errors return through Cobra's normal error path.

### JSON Marshaling Errors

`StatusDashboard` contains only standard Go types (strings, ints, floats, slices, maps). JSON marshaling failures are theoretically impossible with this structure, but the error is checked and wrapped as a defensive measure:

```go
data, err := json.MarshalIndent(dashboard, "", "  ")
if err != nil {
    return fmt.Errorf("failed to marshal progress: %w", err)
}
```

### Context Nil Guard

The nil context guard matches `status.go` exactly:

```go
ctx := cmd.Context()
if ctx == nil {
    var cancel func()
    ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
}
```

This handles test invocations where `cmd.Context()` returns nil.

---

## 7. Test Strategy

### What Can Be Tested Without Database Access

The `GetStatusService()` accessor uses a global database connection that cannot be injected in unit tests (same limitation as `status_test.go`). Therefore, test coverage focuses on the parts of `progress.go` that have no database dependency:

**1. `parseProgressRequest` -- argument parsing (no DB access)**

```go
func TestParseProgressRequest(t *testing.T) {
    tests := []struct {
        name        string
        args        []string
        epicFlag    string
        wantEpicKey string
        wantErr     bool
    }{
        {"no args", []string{}, "", "", false},
        {"epic positional", []string{"E05"}, "", "E05", false},
        {"epic positional overrides flag", []string{"E05"}, "E07", "E05", false},
        {"epic flag only", []string{}, "E05", "E05", false},
        {"combined feature format", []string{"E05-F02"}, "", "E05", false},
        {"too many args", []string{"E05", "F02", "extra"}, "", "", true},
        {"lowercase epic normalized", []string{"e05"}, "", "E05", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := &cobra.Command{}
            cmd.Flags().String("epic", "", "")
            cmd.Flags().String("recent", "", "")
            cmd.Flags().Bool("include-archived", false, "")
            if tt.epicFlag != "" {
                _ = cmd.Flags().Set("epic", tt.epicFlag)
            }

            req, err := parseProgressRequest(cmd, tt.args)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.wantEpicKey, req.EpicKey)
        })
    }
}
```

**2. `outputProgressJSON` -- JSON output routing (no DB access)**

```go
func TestOutputProgressJSON(t *testing.T) {
    dashboard := &status.StatusDashboard{
        Summary: &status.ProjectSummary{
            OverallProgress: 75.0,
        },
        Epics:        []*status.EpicSummary{},
        ActiveTasks:  map[string][]*status.TaskInfo{},
        BlockedTasks: []*status.BlockedTaskInfo{},
    }

    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := outputProgressJSON(dashboard)
    assert.NoError(t, err)

    _ = w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    output := buf.String()

    assert.Contains(t, output, `"overall_progress": 75`)
    assert.Contains(t, output, `"epics"`)
}
```

**3. `runProgress` JSON routing flag (mock service, no DB)**

The routing logic `if cli.GlobalConfig.JSON { return outputProgressJSON(...) }` should be covered once a mock injection mechanism exists for `GetStatusService`. Until then, this is deferred (matching the existing `status_test.go` precedent).

### Test File: `progress_test.go`

```
internal/cli/commands/progress_test.go
```

Target coverage: `parseProgressRequest` (100%), `outputProgressJSON` (partial), `outputProgressTerminal` (partial).

Skipped with explanation (matching `status_test.go`): `runProgress` end-to-end -- blocked on mock injection for `GetStatusService`.

### Integration Testing

Manual smoke test after implementation:

```bash
./bin/shark progress                        # Full project dashboard
./bin/shark progress E17                    # Epic-filtered dashboard
./bin/shark progress E17 F06                # Feature-filtered dashboard
./bin/shark progress E17-F06                # Combined format
./bin/shark progress --json                 # JSON output
./bin/shark progress E17 --json             # Filtered JSON
./bin/shark progress --recent=7d            # With recent completions window
./bin/shark progress --invalid-epic=ZZ99    # Should return validation error
```

### Regression Gate

`make test` must pass green before and after implementation. All existing tests are unaffected because:
- `status.go` is not modified in Phase 1
- No shared mutable state is introduced
- `enrichEpicSummaries` is already package-level, no refactoring needed

---

## 8. File List

### Files Created (New)

| File | Size Estimate | Description |
|---|---|---|
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/progress.go` | ~110 lines | New `progressCmd` command, thin wrapper over `StatusService.GetDashboard()` |
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/progress_test.go` | ~80 lines | Tests for `parseProgressRequest` and output formatting helpers |

### Files Modified (Existing)

| File | Change | Phase | Risk |
|---|---|---|---|
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` | Add commented-out TODO for Phase 2 deprecation in `init()` | Phase 1 (F06) | Very Low |
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` | Uncomment `statusCmd.Deprecated = "..."` line | Phase 2 (after F07) | Very Low |

### Files Unchanged

All of the following require zero modifications:

| File | Reason Unchanged |
|---|---|
| `internal/status/status.go` | `StatusService.GetDashboard()` reused as-is |
| `internal/status/models.go` | `StatusRequest`, `StatusDashboard` reused as-is |
| `internal/status/formatter.go` | `FormatDashboard()` reused as-is |
| `internal/status/calculation_service.go` | Not used by progress command |
| `internal/status/action_items.go` | Called internally by `GetDashboard()`, no direct use |
| `internal/status/context.go` | Called internally by `GetDashboard()`, no direct use |
| `internal/status/progress.go` | Called internally by `GetDashboard()`, no direct use |
| `internal/cli/service_accessors.go` | `GetStatusService()` reused as-is |
| `internal/cli/services_global.go` | No new service wiring needed |
| `internal/cli/commands/helpers.go` | `ParseListArgs()` reused as-is |
| `internal/repository/` (all files) | No repository changes |
| `internal/services/` (all files) | No service changes |
| All existing `*_test.go` files | No existing tests touched |

---

## 9. Design Decisions

### Decision 1: Dashboard Scope (not per-entity routing)

The feature description mentions `shark progress E18-F05-001` showing task-level progress context. Research confirmed that `StatusService.GetDashboard()` does not support per-task or per-feature queries -- it always produces a project/epic-level dashboard.

**Decision**: Implement `shark progress` as a dashboard command (same scope as `shark status`) rather than adding per-entity routing. Per-entity routing would require a new `StatusRequest.FeatureKey` field, changes to `StatusService`, and new formatting logic -- all out of scope for this feature.

Rationale: The research report explicitly recommends deferring per-entity routing. The primary value of this feature is resolving the namespace collision and providing a canonical progress command, not adding new functionality.

### Decision 2: No Duplication of `enrichEpicSummaries`

`enrichEpicSummaries` is defined in `status.go` and is accessible from `progress.go` because both are in the `commands` package. It must not be copied.

### Decision 3: `statusCmd.Deprecated` vs `statusCmd.Hidden`

`Deprecated` is preferred over `Hidden` for Phase 2. It keeps the command visible and functional while printing a migration hint on every invocation. This actively informs agents and users of the recommended path without breaking scripts.

### Decision 4: Phase 2 Deprecation as TODO Comment

The `statusCmd.Deprecated` line is present in `status.go` as a commented-out TODO (not active). This documents the intent without risking breakage if F07 is delayed or reordered. The developer implementing Phase 2 has clear, in-code guidance.

### Decision 5: `--field` Deferred to E17-F02

No `--field` handling is added to `progress.go`. E17-F02 provides the infrastructure; `progress.go` will receive it automatically because its JSON output is a well-structured `StatusDashboard`. This keeps the F06 implementation scope minimal and clean.

---

## 10. Quality Checklist

- [ ] Thin wrapper: parse -> call service -> format output. No business logic in command.
- [ ] Calls service via `cli.GetStatusService()` global accessor (no direct repo access)
- [ ] `enrichEpicSummaries` called from `status.go` (same package), not duplicated
- [ ] `--json` routes to `outputProgressJSON`, terminal routes to `outputProgressTerminal`
- [ ] `--epic`, `--recent`, `--include-archived` flags registered with identical behavior to `status`
- [ ] Positional args (`EPIC`, `EPIC FEATURE`, combined `E05-F02`) handled by `ParseListArgs`
- [ ] Error from `GetDashboard` wrapped with `"failed to get progress: %w"`
- [ ] Context nil guard matches `status.go` pattern
- [ ] `statusCmd.Deprecated` is commented TODO in `status.go` (Phase 1 only)
- [ ] `make fmt && make lint && make test` pass green

---

*Last Updated*: 2026-02-25
