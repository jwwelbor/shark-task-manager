# E17-F08 Research Report: CLI Enhancements and Quick Wins

## Executive Summary

This report provides a tactical codebase analysis for implementing the four tasks in E17-F08. All four tasks are straightforward changes concentrated in `internal/cli/root.go` and `internal/cli/commands/`. No service layer changes are required. The two highest-impact tasks (T-001 and T-002) together require fewer than 30 lines of code changes. T-003 (structured JSON errors) is the most architecturally significant: it requires a new `CLIError` struct, hooking into `cli.Error()`, and deciding where in the command dispatch chain to intercept errors. T-004 (`--field`) is the most technically subtle because it operates on already-serialized JSON, requiring a post-serialization field extraction step.

---

## 1. T-E17-F08-001: Flag Normalization

### Current State Audit

**`--execution-order` flag locations:**

| File | Flag Definition | Notes |
|------|----------------|-------|
| `internal/cli/commands/task_helpers.go:220` | `cmd.Flags().Int("execution-order", 0, "Execution order (alias for --order)")` | `registerCreateFlags()` for `taskCreateCmd` and `taskUpdateCmd` |
| `internal/cli/commands/create.go:130-131` | `Int("execution-order", 0, ...)` and `Int("order", 0, ...)` separately | `createTaskCmd` (the `shark create task` alias) |
| `internal/cli/commands/feature.go:145` | `IntVar(&featureCreateExecutionOrder, "execution-order", 0, ...)` | `featureCreateCmd` |
| `internal/cli/commands/feature.go:165` | `Int("execution-order", -1, "New execution order (-1 = no change)")` | `featureUpdateCmd` |
| `internal/cli/commands/create.go:108` | `IntVar(&featureCreateExecutionOrder, "execution-order", 0, ...)` | `createFeatureCmd` alias |

**Key finding**: For task commands, `--order` is already the PRIMARY flag and `--execution-order` is the alias (`task_helpers.go:219-220`). The alias behavior is already implemented in `task_helpers.go:137-139`:

```go
order, _ := cmd.Flags().GetInt("order")
if execOrder, _ := cmd.Flags().GetInt("execution-order"); execOrder > 0 && order == 0 {
    order = execOrder
}
```

So for task commands, the work is only to mark `--execution-order` as deprecated using Cobra's `MarkDeprecated()`. For **feature** commands, `--execution-order` is the only flag (no `--order` alias exists yet).

**`--show-all` flag locations:**

| File | Flag Definition | Context |
|------|----------------|---------|
| `internal/cli/commands/task_helpers.go:208` | `cmd.Flags().Bool("show-all", false, "Show all tasks including completed")` | `registerListFlags()` for `taskListCmd` |
| `internal/cli/commands/feature.go:140` | `featureListCmd.Flags().Bool("show-all", false, "Show all features including completed")` | `featureListCmd` |
| `internal/cli/commands/list.go:36` | `listCmd.Flags().Bool("show-all", false, "Show all items...")` | Smart list dispatcher |

**Read locations** (where the flag value is consumed):

| File | Line | Context |
|------|------|---------|
| `internal/cli/commands/task.go:105` | `showAll, _ := cmd.Flags().GetBool("show-all")` | `runTaskList` |
| `internal/cli/commands/feature_helpers.go:1106` | `showAll, _ = cmd.Flags().GetBool("show-all")` | `runFeatureList` via helpers |
| `internal/cli/commands/list.go:50` | `showAllFlag, _ := cmd.Flags().GetBool("show-all")` | `runList` smart dispatcher |
| `internal/cli/commands/list.go:89,101` | `featureListCmd.Flags().Set("show-all", ...)` and `taskListCmd.Flags().Set("show-all", ...)` | Propagation from list dispatcher |

### Implementation Approach

**For `--execution-order` on task commands** (already have `--order` as primary):
```go
// In registerCreateFlags() and registerTransitionFlags() in task_helpers.go:
_ = cmd.Flags().MarkDeprecated("execution-order", "use --order instead")
```

**For `--execution-order` on feature commands** (need to add `--order` as alias):
1. Add `--order` alias alongside `--execution-order` in `feature.go:145` and `feature.go:165`
2. Read both in `feature_helpers.go` (same pattern as task_helpers.go:137-139)
3. Mark `--execution-order` as deprecated

**For `--show-all`** (need to add `--all` as alias everywhere):
Adding a true alias with the same bound variable is the cleanest approach. Cobra supports this with `Flags().Bool()` + `MarkDeprecated()` on the old name, but the simplest implementation is:
1. Keep `--show-all` registered as-is
2. Add `--all` as a new flag bound to the same variable (use `BoolP` with the same destination)
3. Mark `--show-all` as deprecated pointing to `--all`

The complication is that `task_helpers.go` uses `registerListFlags(cmd)` which is called for both `taskListCmd` and `taskNextCmd`. The flag must be added in `registerListFlags()`.

For the `list.go` smart dispatcher, the `--show-all` flag is registered separately and propagated by setting the flag on child commands -- this propagation code must also handle `--all`.

### File Impact List

- `internal/cli/commands/task_helpers.go` - Mark `--execution-order` deprecated; add `--all` alias in `registerListFlags()`
- `internal/cli/commands/feature.go` - Add `--order` alias for featureCreateCmd and featureUpdateCmd; mark `--execution-order` deprecated; add `--all` alias for featureListCmd
- `internal/cli/commands/create.go` - Mark `--execution-order` deprecated on createTaskCmd and createFeatureCmd; add `--all` alias for list operations
- `internal/cli/commands/list.go` - Add `--all` alias; update propagation in `runFeatureListWithFlags` and `runTaskListWithFlags`
- `internal/cli/commands/task.go` - Read `--all` in addition to `--show-all` in `runTaskList`
- `internal/cli/commands/feature_helpers.go` - Read `--all` alongside `--show-all`

---

## 2. T-E17-F08-002: SHARK_OUTPUT Environment Variable

### Current State

The `initConfig()` function in `internal/cli/root.go` (lines 235-287) handles all environment variable configuration. The relevant section:

```go
// Read environment variables with PM_ prefix
viper.SetEnvPrefix("PM")
viper.AutomaticEnv()

// ...

// Update GlobalConfig from viper
GlobalConfig.JSON = viper.GetBool("json")
GlobalConfig.NoColor = viper.GetBool("no-color")
GlobalConfig.Verbose = viper.GetBool("verbose")
```

The `PM` prefix means `PM_JSON=true` would set JSON mode via Viper's `AutomaticEnv()`. However, `SHARK_OUTPUT` is a completely separate env var not covered by the `PM_` prefix convention. It requires explicit `os.Getenv()` handling.

**Why not use Viper for this**: `SHARK_OUTPUT=json` maps a string value to a boolean behavior. Viper's `AutomaticEnv()` would only handle `PM_JSON=true` (boolean), not `SHARK_OUTPUT=json` (string-to-bool mapping). Custom handling is required.

### Implementation Approach

In `initConfig()` in `internal/cli/root.go`, add after the Viper config update block (after line 284):

```go
// Check SHARK_OUTPUT environment variable for session-wide output mode
// This is checked AFTER --json flag processing so the flag takes precedence
// if the flag was explicitly set (Cobra flag parsing happens before PersistentPreRunE
// calls initConfig(), so flag values are already in GlobalConfig).
if sharkOutput := os.Getenv("SHARK_OUTPUT"); sharkOutput == "json" {
    // Only override if --json was NOT explicitly set on the command line.
    // Viper already set GlobalConfig.JSON from the flag; if it's still false,
    // the env var can set it.
    if !GlobalConfig.JSON {
        GlobalConfig.JSON = true
    }
}
```

**Ordering note**: The `PersistentPreRunE` in `root.go` calls `initConfig()`. Cobra parses flags before calling `PersistentPreRunE`, which means by the time `initConfig()` runs, `GlobalConfig.JSON` already reflects the `--json` flag. The env var should only activate if the flag was NOT set.

However, there is a subtlety: Viper's `AutomaticEnv()` with prefix `PM` maps `PM_JSON` to overwrite `GlobalConfig.JSON` after Cobra parsing. The env var check must come after the Viper block, and it must NOT overwrite a `true` value (which would come from either `--json` flag or `PM_JSON=true`).

The correct implementation: check `!GlobalConfig.JSON` before setting, because if `GlobalConfig.JSON` is already `true` from any source (`--json` flag or `PM_JSON=true`), the env var adds no value.

### File Impact List

- `internal/cli/root.go` - Add `SHARK_OUTPUT` check in `initConfig()` (~5 lines)

---

## 3. T-E17-F08-003: Structured JSON Error Output

### Current State Analysis

**Error output today**: `cli.Error()` in `root.go:325-330` outputs to stderr via `pterm.Error.Println()` or `fmt.Fprintln(os.Stderr, ...)`. It always outputs human-readable text regardless of `--json` mode.

**Error functions in `errors.go`**: The file contains formatting functions that return `error` values (not print them). These return multi-line human-readable strings via `fmt.Errorf()`:
- `InvalidEpicKeyError(key string) error`
- `InvalidFeatureKeyError(key string) error`
- `InvalidTaskKeyError(key string) error`
- `MissingArgumentsError(expected, got int, examples []string) error`
- `TooManyArgumentsError(expected, got int) error`
- `InvalidPositionalArgsError(command, reason string, examples []string) error`
- `AmbiguousKeyError(key string, suggestions []string) error`
- `NotFoundError(entityType, key string) error`
- `InvalidStatusTransitionError(currentStatus, targetStatus string, allowedTransitions []string) error`

These errors are returned from command `RunE` functions and handled by Cobra's error printing. Cobra prints them to stderr as plain text.

**Error calling patterns in commands**: Many commands use `cli.Error(fmt.Sprintf("Error: %v", err))` followed by returning `nil` (to suppress Cobra's duplicate error display) or returning the error directly. This inconsistency is important to note.

**How errors currently reach the user**:
1. Commands return `error` from `RunE` → Cobra prints `"Error: <message>"` to stderr
2. Commands call `cli.Error(message)` then return `nil` → only the `cli.Error()` call appears
3. Commands call `cli.Error(message)` then `return err` → duplicate display (both `cli.Error()` and Cobra's error handler fire)

**Service-layer typed errors that carry structured data**:
- `repository.NotFoundError{Entity, Key}` - propagated up from repository
- `services.BackwardReasonError` in `transition_types.go`
- `services.ErrReasonRequired`, `services.ErrForceReasonRequired`

### Implementation Approach

**Step 1: Define `CLIError` struct** in a new file `internal/cli/errors.go` (or in `root.go`):

```go
// CLIError is the structured error format for JSON mode output.
type CLIError struct {
    Error          bool     `json:"error"`
    Code           string   `json:"code"`
    Message        string   `json:"message"`
    Entity         string   `json:"entity,omitempty"`
    EntityKey      string   `json:"entity_key,omitempty"`
    CurrentStatus  string   `json:"current_status,omitempty"`
    ValidTransitions []string `json:"valid_transitions,omitempty"`
}

// Error codes
const (
    ErrCodeNotFound          = "NOT_FOUND"
    ErrCodeInvalidTransition = "INVALID_TRANSITION"
    ErrCodeValidationError   = "VALIDATION_ERROR"
    ErrCodeDatabaseError     = "DATABASE_ERROR"
    ErrCodeInvalidArgs       = "INVALID_ARGS"
)
```

**Step 2: Modify `cli.Error()`** in `root.go` to check JSON mode:

```go
func Error(message string) {
    if GlobalConfig.JSON {
        // In JSON mode, output structured error to STDOUT (not stderr)
        // so agents only need to parse one stream.
        errObj := &CLIError{
            Error:   true,
            Code:    ErrCodeValidationError,  // default; callers can use ErrorJSON() for typed errors
            Message: message,
        }
        _ = OutputJSON(errObj)
        return
    }
    if !GlobalConfig.NoColor {
        pterm.Error.Println(message)
    } else {
        fmt.Fprintln(os.Stderr, "✗", message)
    }
}
```

**Step 3: Add `ErrorJSON()` for typed error output** with code and context:

```go
func ErrorJSON(err CLIError) {
    err.Error = true
    if GlobalConfig.JSON {
        _ = OutputJSON(err)
    } else {
        // Fall back to human-readable format
        Error(err.Message)
    }
}
```

**Step 4: Handle errors returned from `RunE`**. Currently Cobra's built-in error handling prints to stderr as plain text even in JSON mode. Two options:

- **Option A (recommended)**: Wrap `Execute()` in `cmd/shark/main.go` to intercept errors:
  ```go
  if err := cli.Execute(); err != nil {
      if cli.GlobalConfig.JSON {
          cli.OutputJSON(&cli.CLIError{Error: true, Code: "COMMAND_ERROR", Message: err.Error()})
      }
      os.Exit(1)
  }
  ```
  Set `RootCmd.SilenceErrors = true` to prevent Cobra from printing errors itself.

- **Option B**: Use `PersistentPostRunE` -- but this only runs on success. No hook fires on error after `RunE` fails.

- **Option C**: Use Cobra's `cobra.Command.SetErr()` to redirect error output, but this doesn't give structured JSON.

**Recommendation**: Option A -- set `RootCmd.SilenceErrors = true` and handle errors in `Execute()` caller (in `cmd/shark/main.go`).

**Step 5: Map typed errors to codes** in command error handlers. Where commands currently do:
```go
// Old
cli.Error(fmt.Sprintf("Feature %s not found", featureKey))
```
They should use:
```go
// New
cli.ErrorJSON(cli.CLIError{
    Code:      cli.ErrCodeNotFound,
    Message:   fmt.Sprintf("feature %s not found", featureKey),
    EntityKey: featureKey,
})
```

However, this is a significant refactor across ~90 `cli.Error()` call sites (grep shows 60+ calls across 25+ files). A pragmatic approach for the initial implementation:

- Phase 1: Modify `cli.Error()` to emit generic JSON when in JSON mode (all existing call sites get basic structured errors automatically).
- Phase 2: Selectively update high-frequency error sites (NotFound, InvalidTransition) to use richer `ErrorJSON()` with proper codes.

**The `InvalidStatusTransitionError` in `errors.go`** already contains `allowedTransitions []string`. This data can feed `valid_transitions` in the JSON error. This function currently returns an `error` that Cobra prints -- it needs to be intercepted.

### File Impact List

- `internal/cli/root.go` - Modify `cli.Error()` to emit JSON when in JSON mode; add `CLIError` struct and `ErrorJSON()` function; add `SilenceErrors = true` to root command
- `cmd/shark/main.go` - Handle error from `cli.Execute()` with JSON output when in JSON mode
- `internal/cli/commands/errors.go` - Add JSON-friendly variants of the error constructor functions (or modify existing ones to also return a `CLIError`)
- Selected high-traffic command files (optional Phase 2): `epic_helpers.go`, `feature_helpers.go`, `task.go`, etc.

---

## 4. T-E17-F08-004: --field Flag for Targeted Extraction

### Current Output Architecture

`cli.OutputJSON()` in `root.go:290-294` serializes `interface{}` to JSON using `json.NewEncoder`:

```go
func OutputJSON(data interface{}) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

The `formatters` package (`internal/formatters/json.go`) defines response types with JSON tags:
- `EpicWithProgress` - embeds `*models.Epic` + `progress_pct float64`
- `FeatureWithTaskCount` - embeds `*models.Feature` + `task_count int`
- `EpicListResponse` - `{results: [...], count: int}`
- `FeatureListResponse` - `{results: [...], count: int}`
- `EpicGetResponse`, `FeatureGetResponse`

Most commands serialize domain models (`*models.Task`, `*models.Epic`, etc.) directly, so field names match JSON tags (snake_case). Some commands use formatter types for richer output.

### Implementation Approach

**Step 1: Add `--field` to `GlobalConfig` and root command**:

```go
// In Config struct (root.go)
type Config struct {
    JSON       bool
    NoColor    bool
    Verbose    bool
    ConfigFile string
    DBPath     string
    Field      string  // NEW: field extraction filter
}

// In init() (root.go)
RootCmd.PersistentFlags().StringVar(&GlobalConfig.Field, "field", "", "Extract a single field from JSON output (e.g., --field=status)")
```

**Step 2: Create `OutputField()` function** that extracts from JSON:

```go
// OutputField extracts a named field from data and prints the raw value.
// For single objects: prints the field value.
// For arrays: prints one value per line (newline-separated).
// Returns error if field is not found.
func OutputField(data interface{}, fieldName string) error {
    // Marshal to JSON bytes first
    b, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to serialize data: %w", err)
    }

    // Unmarshal to generic map for field lookup
    var m interface{}
    if err := json.Unmarshal(b, &m); err != nil {
        return fmt.Errorf("failed to parse data: %w", err)
    }

    return extractAndPrintField(m, fieldName)
}

func extractAndPrintField(data interface{}, fieldName string) error {
    switch v := data.(type) {
    case map[string]interface{}:
        // Single object: direct field lookup with dot-notation support
        val, err := lookupField(v, fieldName)
        if err != nil {
            return err
        }
        printFieldValue(val)
        return nil
    case []interface{}:
        // Array: extract field from each element, one per line
        for _, item := range v {
            if obj, ok := item.(map[string]interface{}); ok {
                val, err := lookupField(obj, fieldName)
                if err != nil {
                    continue  // Skip items that don't have the field
                }
                printFieldValue(val)
            }
        }
        return nil
    default:
        return fmt.Errorf("unsupported data type for field extraction")
    }
}

func lookupField(obj map[string]interface{}, fieldPath string) (interface{}, error) {
    // Support dot notation: "status_breakdown.todo"
    parts := strings.SplitN(fieldPath, ".", 2)
    val, ok := obj[parts[0]]
    if !ok {
        return nil, fmt.Errorf("field %q not found (exit code 4)", fieldPath)
    }
    if len(parts) == 2 {
        nested, ok := val.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("field %q is not an object", parts[0])
        }
        return lookupField(nested, parts[1])
    }
    return val, nil
}

func printFieldValue(val interface{}) {
    switch v := val.(type) {
    case string:
        fmt.Println(v)
    case float64:
        if v == float64(int64(v)) {
            fmt.Println(int64(v))
        } else {
            fmt.Printf("%g\n", v)
        }
    case bool:
        fmt.Println(v)
    case nil:
        fmt.Println("null")
    default:
        // Objects and arrays: re-serialize as JSON
        b, _ := json.Marshal(v)
        fmt.Println(string(b))
    }
}
```

**Step 3: Modify command output paths** to check `GlobalConfig.Field`. The cleanest approach is to modify `OutputJSON()` to be field-aware:

```go
func OutputJSON(data interface{}) error {
    if GlobalConfig.Field != "" {
        return OutputField(data, GlobalConfig.Field)
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

This way, every command that calls `cli.OutputJSON(data)` automatically gets `--field` support without any per-command changes. `--field` implicitly enables JSON mode (field extraction requires JSON serialization internally), so add to `initConfig()`:

```go
if GlobalConfig.Field != "" {
    GlobalConfig.JSON = true  // --field implies --json
}
```

**Step 4: Exit code for field not found**. Based on the existing convention, exit code 1 is "entity not found" and is the only candidate for "field not found." The research report recommends using exit code 4 to distinguish the two cases. This requires the `main.go` error handler to check for a `FieldNotFoundError` type and exit with 4.

**Dot notation support**: Field paths like `status_breakdown.todo` and `progress.weighted_pct` are common in the richer response types. The implementation above supports this with recursive `lookupField()`.

**Behavior with list commands**: For arrays, `--field` extracts the field from each element and prints one value per line. This matches `gh pr list --json title --jq '.[].title'` behavior and is the most useful behavior for agent scripts.

### File Impact List

- `internal/cli/root.go` - Add `Field string` to `Config`; register `--field` persistent flag; modify `OutputJSON()` to be field-aware; add `OutputField()`, `extractAndPrintField()`, `lookupField()`, `printFieldValue()` functions; set `GlobalConfig.JSON = true` when `Field != ""`
- `cmd/shark/main.go` - Handle `FieldNotFoundError` exit code 4
- `internal/cli/commands/` - No per-command changes required if `OutputJSON()` is field-aware

---

## 5. Cross-Task Concerns

### Interaction: T-002 + T-003 (SHARK_OUTPUT + JSON Errors)

When `SHARK_OUTPUT=json` is set:
- `GlobalConfig.JSON` is `true` after `initConfig()`
- `cli.Error()` will emit JSON to stdout
- All normal output also goes to stdout as JSON
- This is the intended behavior (one stream for agents to parse)

### Interaction: T-003 + T-004 (JSON Errors + --field)

When `--field=status` is used and an error occurs:
- The error should be output as JSON (since `--field` implies `--json`)
- The error JSON should not be run through the field extractor -- it should output as-is
- Implementation: check for `CLIError` in `OutputJSON()` and skip field extraction for errors

### Cobra Error Handling Gap

Currently, `internal/cli/commands/errors.go` functions (`NotFoundError()`, `InvalidStatusTransitionError()`, etc.) return `error` values that Cobra captures and prints via its own error handler. Cobra uses `os.Stderr` for this. With T-003:
- `RootCmd.SilenceErrors = true` prevents Cobra from printing returned errors
- The `Execute()` caller in `main.go` must print the error (as JSON if in JSON mode)
- Commands that call `cli.Error()` then return `nil` continue to work as before

### Test Implications

- T-001 (flag normalization): Update `create_test.go:157` which checks for the `execution-order` flag name, and `feature_update_test.go:31` which checks for `--execution-order`; add tests for `--all` alias behavior
- T-002 (SHARK_OUTPUT): Add test that sets env var and verifies `GlobalConfig.JSON = true`
- T-003 (JSON errors): Add tests for `cli.Error()` in JSON mode outputting to stdout as JSON
- T-004 (`--field`): Add unit tests for `OutputField()`, `lookupField()`, `extractAndPrintField()`; table-driven tests for dot notation, arrays, missing fields

---

## 6. Implementation Recommendations

### Recommended Order

1. **T-002 (SHARK_OUTPUT)**: 5-10 lines in `root.go`. Ship this first -- it enables testing all subsequent features with `SHARK_OUTPUT=json` without `--json` on every command.

2. **T-001 (Flag normalization)**: Mechanical changes using `MarkDeprecated()`. No logic changes required. Cobra's deprecation support is well-tested. Order within T-001: task flags first (they already have `--order`), then feature flags.

3. **T-003 (JSON errors)**: Implement the `CLIError` struct and modify `cli.Error()` first for the generic case. Then address the Cobra error interception in `main.go`. Do NOT attempt to update all 60+ `cli.Error()` call sites in the first pass -- the generic modification to `cli.Error()` handles them all automatically.

4. **T-004 (`--field`)**: Implement last, as it builds on the stable JSON output infrastructure. The key design decision (field-aware `OutputJSON()`) means zero per-command changes.

### Key Design Decisions Required

1. **`--field` + error output**: Should `--field` suppress errors or format them as JSON? Recommendation: errors always output as JSON when in JSON/field mode, bypassing field extraction.

2. **`--field` on list commands returning `{results: [...], count: N}`**: The top-level object has `results` and `count` fields. Should `--field=results` return the full array? Recommendation: yes, return raw JSON of the array. Should `--field=title` auto-traverse into `results[*].title`? Recommendation: no auto-traversal for clarity; agents should use `--field=results` and process the array.

3. **Exit code for field not found**: Use exit code 4 (`FIELD_NOT_FOUND`) to distinguish from exit code 1 (`ENTITY_NOT_FOUND`). Requires `main.go` to check for a typed `FieldNotFoundError`.

4. **`SHARK_OUTPUT` vs `PM_JSON`**: `SHARK_OUTPUT=json` is the specified interface. The existing `PM_JSON=true` continues to work via Viper's `AutomaticEnv()`. Both set `GlobalConfig.JSON = true`. Document both in help text.

---

## 7. File Impact Summary Per Task

### T-001: Flag Normalization
- `internal/cli/commands/task_helpers.go` (registerListFlags, registerCreateFlags)
- `internal/cli/commands/feature.go` (featureListCmd, featureCreateCmd, featureUpdateCmd flags)
- `internal/cli/commands/create.go` (createTaskCmd, createFeatureCmd flags)
- `internal/cli/commands/list.go` (listCmd flag, propagation functions)
- `internal/cli/commands/task.go` (runTaskList: read `--all`)
- `internal/cli/commands/feature_helpers.go` (read `--all`)

### T-002: SHARK_OUTPUT
- `internal/cli/root.go` (initConfig: ~8 lines)

### T-003: Structured JSON Errors
- `internal/cli/root.go` (CLIError struct, Error() modification, ErrorJSON(), SilenceErrors)
- `cmd/shark/main.go` (Execute() error handler)
- `internal/cli/commands/errors.go` (add JSON-returning variants, optional Phase 2)

### T-004: --field Flag
- `internal/cli/root.go` (Config struct, flag registration, OutputJSON modification, OutputField implementation)
- `cmd/shark/main.go` (FieldNotFoundError exit code 4)

---

## 8. References

- `/home/jwwel/projects/shark-task-manager/internal/cli/root.go` - GlobalConfig, OutputJSON, initConfig, cli.Error
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/task_helpers.go` - registerListFlags, registerCreateFlags, parseCreateTaskInput
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/task.go` - runTaskList (show-all reader)
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/feature.go` - featureListCmd/CreateCmd/UpdateCmd flag definitions
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/feature_helpers.go` - show-all reader, runFeatureList
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/list.go` - Smart list dispatcher, show-all propagation
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/create.go` - createTaskCmd, createFeatureCmd flag definitions
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/errors.go` - Error formatting functions
- `/home/jwwel/projects/shark-task-manager/internal/formatters/json.go` - Response types used in JSON output
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/research-report.md` - Epic research context

*Last Updated: 2026-02-25*
