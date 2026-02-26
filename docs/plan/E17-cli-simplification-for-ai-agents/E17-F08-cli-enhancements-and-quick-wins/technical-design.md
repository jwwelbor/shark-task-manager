# E17-F08 Technical Design: CLI Enhancements and Quick Wins

**Status**: Ready for Development
**Date**: 2026-02-25
**Feature**: E17-F08-cli-enhancements-and-quick-wins

---

## Overview

This document specifies the implementation design for four targeted CLI improvements. All changes are confined to `internal/cli/root.go`, `internal/cli/commands/`, and `cmd/shark/main.go`. No service layer, repository, or model changes are required. The tasks are ordered by implementation dependency: T-002 first (enables testing all others), then T-001 (mechanical, no logic change), then T-003 (error infrastructure), then T-004 (builds on stable JSON output).

---

## T-E17-F08-001: Flag Normalization

### Objective

Add `--order` as an alias for `--execution-order` on feature commands (task commands already have `--order` as the primary flag). Add `--all` as an alias for `--show-all` on all list commands. Mark the old flag names as deprecated so Cobra prints a deprecation warning when they are used.

### Cobra MarkDeprecated Pattern

`MarkDeprecated(flagName, message)` causes Cobra to:
1. Print a warning to stderr: `Flag --<name> has been deprecated, <message>`
2. Still accept the flag and pass through its value normally
3. Hide the flag from `--help` output (use `MarkShorthandDeprecated` for shorthands)

The pattern is always applied after the flag is registered, never before.

```go
_ = cmd.Flags().MarkDeprecated("execution-order", "use --order instead")
```

### Sub-task 1a: `--execution-order` on Task Commands

**Current state**: `task_helpers.go:registerCreateFlags()` registers both `--order` (primary) and `--execution-order` (alias), and `parseCreateTaskInput()` already merges them (lines 137-139). The alias logic already works correctly.

**Change required**: Mark `--execution-order` as deprecated immediately after its registration in `registerCreateFlags()`.

**File**: `internal/cli/commands/task_helpers.go`

```go
// In registerCreateFlags(), after line 220:
cmd.Flags().Int("execution-order", 0, "Execution order (alias for --order)")
_ = cmd.Flags().MarkDeprecated("execution-order", "use --order instead")
```

No changes to `parseCreateTaskInput()` - the existing merge logic at lines 137-139 remains unchanged.

### Sub-task 1b: `--execution-order` on Feature Commands

**Current state**: Feature commands only have `--execution-order`. There is no `--order` alias.

**Changes required**:

In `feature.go` `init()`:

1. For `featureCreateCmd` (line 145): add `--order` as a new flag alongside the existing `--execution-order`, then mark `--execution-order` as deprecated.

2. For `featureUpdateCmd` (line 165): add `--order` as a new flag alongside the existing `--execution-order`, then mark `--execution-order` as deprecated.

**File**: `internal/cli/commands/feature.go`

```go
// In init(), after the featureCreateCmd --execution-order registration:
featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order")
featureCreateCmd.Flags().Int("order", 0, "Execution order (shorthand for --execution-order)")
_ = featureCreateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")

// In init(), after the featureUpdateCmd --execution-order registration:
featureUpdateCmd.Flags().Int("execution-order", -1, "New execution order (-1 = no change)")
featureUpdateCmd.Flags().Int("order", -1, "New execution order (shorthand for --execution-order)")
_ = featureUpdateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
```

The read logic in `feature_helpers.go` for `featureCreateCmd` reads the package-level `featureCreateExecutionOrder` variable which is bound to `--execution-order`. Update it to also check `--order`:

```go
// In parseFeatureCreateInput() or equivalent in feature_helpers.go:
execOrder := featureCreateExecutionOrder  // from --execution-order IntVar binding
if orderFlag, _ := cmd.Flags().GetInt("order"); orderFlag != 0 && execOrder == 0 {
    execOrder = orderFlag
}
```

For `featureUpdateCmd`, the existing code reads `cmd.Flags().GetInt("execution-order")`. Add a merge check:

```go
execOrder, _ := cmd.Flags().GetInt("execution-order")
if orderFlag, _ := cmd.Flags().GetInt("order"); orderFlag != -1 && execOrder == -1 {
    execOrder = orderFlag
}
```

### Sub-task 1c: `--show-all` → `--all` Alias

**Current state**: `--show-all` is registered independently on three commands: `registerListFlags()` in `task_helpers.go`, `featureListCmd` in `feature.go`, and `listCmd` in `list.go`. It is read in `task.go`, `feature_helpers.go`, and `list.go`.

**Design decision**: Add `--all` as a separate boolean flag bound to a local variable, read both flags, and combine with OR. This avoids Cobra's lack of built-in flag aliasing for non-shorthand flags. Mark `--show-all` as deprecated.

**Pattern for each registration site**:

```go
// Register both flags
cmd.Flags().Bool("show-all", false, "Show all tasks including completed")
cmd.Flags().Bool("all", false, "Show all items including completed")
_ = cmd.Flags().MarkDeprecated("show-all", "use --all instead")

// Read both flags and combine
showAll, _ := cmd.Flags().GetBool("show-all")
allFlag, _ := cmd.Flags().GetBool("all")
showAll = showAll || allFlag
```

**File changes and specific locations**:

`internal/cli/commands/task_helpers.go` - `registerListFlags()`:
```go
cmd.Flags().Bool("show-all", false, "Show all tasks including completed")
cmd.Flags().Bool("all", false, "Show all tasks including completed")
_ = cmd.Flags().MarkDeprecated("show-all", "use --all instead")
```

`internal/cli/commands/feature.go` - `featureListCmd` flags in `init()`:
```go
featureListCmd.Flags().Bool("show-all", false, "Show all features including completed")
featureListCmd.Flags().Bool("all", false, "Show all features including completed")
_ = featureListCmd.Flags().MarkDeprecated("show-all", "use --all instead")
```

`internal/cli/commands/list.go` - `listCmd` flags in `init()`:
```go
listCmd.Flags().Bool("show-all", false, "Show all items including completed")
listCmd.Flags().Bool("all", false, "Show all items including completed")
_ = listCmd.Flags().MarkDeprecated("show-all", "use --all instead")
```

**Read locations** - apply the OR pattern in each:

`internal/cli/commands/task.go` `runTaskList()`:
```go
showAll, _ := cmd.Flags().GetBool("show-all")
if allFlag, _ := cmd.Flags().GetBool("all"); allFlag {
    showAll = true
}
```

`internal/cli/commands/feature_helpers.go` `parseFeatureListFlags()`:
```go
showAll, _ = cmd.Flags().GetBool("show-all")
if allFlag, _ := cmd.Flags().GetBool("all"); allFlag {
    showAll = true
}
```

`internal/cli/commands/list.go` `runList()`:
```go
showAllFlag, _ := cmd.Flags().GetBool("show-all")
if allFlag, _ := cmd.Flags().GetBool("all"); allFlag {
    showAllFlag = true
}
```

The propagation functions `runFeatureListWithFlags()` and `runTaskListWithFlags()` pass `showAll bool` directly. No change needed there since the bool is already merged before calling them.

### Complete File List for T-001

| File | Changes |
|------|---------|
| `internal/cli/commands/task_helpers.go` | `MarkDeprecated("execution-order")` in `registerCreateFlags()`; add `--all` + `MarkDeprecated("show-all")` in `registerListFlags()` |
| `internal/cli/commands/task.go` | Read `--all` in `runTaskList()` |
| `internal/cli/commands/feature.go` | Add `--order` + `MarkDeprecated("execution-order")` for featureCreateCmd and featureUpdateCmd; add `--all` + `MarkDeprecated("show-all")` for featureListCmd |
| `internal/cli/commands/feature_helpers.go` | Read `--order` fallback for execution order; read `--all` alongside `--show-all` |
| `internal/cli/commands/list.go` | Add `--all` + `MarkDeprecated("show-all")` in `init()`; read `--all` in `runList()` |
| `internal/cli/commands/create.go` | Mark `--execution-order` deprecated on `createTaskCmd` and `createFeatureCmd` (these are aliases of the feature/task create commands and register the same flag independently) |

### Tests for T-001

- `task_helpers_test.go` or `task_test.go`: verify `--all` is accepted and produces same result as `--show-all`; verify `--execution-order` produces a deprecation warning but still works
- `feature_test.go`: verify `--order` on `feature create` and `feature update`; verify `--all` on `feature list`
- Existing test in `create_test.go` that checks for `execution-order` flag name: update to also check that `--order` works

---

## T-E17-F08-002: SHARK_OUTPUT Environment Variable

### Objective

Support `SHARK_OUTPUT=json` as a session-wide way to enable JSON mode, so AI agents can set it once in their environment rather than passing `--json` to every command.

### Location in root.go

The `initConfig()` function (lines 235-287) is the correct insertion point. It is called via `PersistentPreRunE` on every command after Cobra has already parsed flags. The insertion goes after the Viper update block (after line 284 `GlobalConfig.Verbose = viper.GetBool("verbose")`).

### Precedence Model

```
Flag --json (highest)  →  PM_JSON=true env var  →  SHARK_OUTPUT=json  →  default (false)
```

Viper's `AutomaticEnv()` with `SetEnvPrefix("PM")` maps `PM_JSON=true` to override `GlobalConfig.JSON` after Cobra flag parsing (line 277). After the Viper block, `GlobalConfig.JSON` is `true` if either `--json` was passed OR `PM_JSON=true` is set.

The `SHARK_OUTPUT` check must come after the Viper block and must only activate when `GlobalConfig.JSON` is still `false` (i.e., neither `--json` nor `PM_JSON=true` was set). This preserves the "flag > env > default" precedence.

### Implementation

**File**: `internal/cli/root.go`

Add to `initConfig()` after line 284 (`GlobalConfig.Verbose = viper.GetBool("verbose")`):

```go
// Check SHARK_OUTPUT environment variable.
// Only activates if --json flag and PM_JSON env var have not already enabled JSON mode.
// Precedence: --json flag > PM_JSON=true > SHARK_OUTPUT=json > default (false)
if !GlobalConfig.JSON {
    if sharkOutput := os.Getenv("SHARK_OUTPUT"); sharkOutput == "json" {
        GlobalConfig.JSON = true
    }
}
```

`os` is already imported in `root.go`.

### Behavior Notes

- `SHARK_OUTPUT=json shark task list` - JSON mode active
- `shark task list --json` - JSON mode active (flag wins, env var irrelevant)
- `SHARK_OUTPUT=json shark task list --json=false` - Flag explicitly set to false takes precedence; JSON mode inactive (Cobra sets the flag value, Viper reads it as `false`)
- `SHARK_OUTPUT=table` - No effect; only `json` value is recognized
- `SHARK_OUTPUT=` (empty) - No effect

### Tests for T-002

In a test file for root config behavior:
```go
func TestInitConfig_SHARKOutputEnvVar(t *testing.T) {
    t.Setenv("SHARK_OUTPUT", "json")
    // Reset GlobalConfig before test
    GlobalConfig.JSON = false
    // Call initConfig (or test the relevant portion)
    // Assert GlobalConfig.JSON == true
}

func TestInitConfig_SHARKOutputDoesNotOverrideFlag(t *testing.T) {
    t.Setenv("SHARK_OUTPUT", "json")
    GlobalConfig.JSON = true  // simulates --json flag having been set
    // Call initConfig
    // Assert GlobalConfig.JSON remains true (env var adds no harm)
}
```

**File**: `internal/cli/root.go` (change ~5 lines)

---

## T-E17-F08-003: Structured JSON Error Output

### Objective

When `--json` mode is active (including via `SHARK_OUTPUT=json`), all error output must be structured JSON on stdout rather than human-readable text on stderr. This allows AI agents to parse a single stream for both success and error data.

### Phase 1 (Required): Generic JSON Errors

Modify `cli.Error()` to emit a generic structured JSON error when in JSON mode. This handles all 60+ existing `cli.Error()` call sites automatically with no per-command changes.

Intercept errors returned from `RunE` by setting `RootCmd.SilenceErrors = true` and handling errors in `cmd/shark/main.go`.

### CLIError Struct

Define in a new file `internal/cli/cli_error.go`:

```go
package cli

// CLIError is the structured error format emitted in JSON mode.
// The "error" field always being true allows agents to detect errors
// without checking HTTP status codes.
type CLIError struct {
    Error            bool     `json:"error"`
    Code             string   `json:"code"`
    Message          string   `json:"message"`
    Entity           string   `json:"entity,omitempty"`
    EntityKey        string   `json:"entity_key,omitempty"`
    CurrentStatus    string   `json:"current_status,omitempty"`
    ValidTransitions []string `json:"valid_transitions,omitempty"`
}

// Error code constants
const (
    ErrCodeNotFound          = "NOT_FOUND"
    ErrCodeInvalidTransition = "INVALID_TRANSITION"
    ErrCodeValidationError   = "VALIDATION_ERROR"
    ErrCodeDatabaseError     = "DATABASE_ERROR"
    ErrCodeInvalidArgs       = "INVALID_ARGS"
    ErrCodeCommandError      = "COMMAND_ERROR"
)
```

The `Error bool` field is structurally redundant with the JSON key name, but it provides a reliable sentinel: agents can check `.error == true` regardless of which other fields are present.

### Modified cli.Error()

**File**: `internal/cli/root.go`

Replace the existing `Error()` function (lines 325-330):

```go
// Error prints an error message. In JSON mode, outputs structured JSON to stdout.
// All errors go to stdout in JSON mode so agents parse a single stream.
func Error(message string) {
    if GlobalConfig.JSON {
        _ = OutputJSON(&CLIError{
            Error:   true,
            Code:    ErrCodeCommandError,
            Message: message,
        })
        return
    }
    if !GlobalConfig.NoColor {
        pterm.Error.Println(message)
    } else {
        fmt.Fprintln(os.Stderr, "✗", message)
    }
}
```

### ErrorJSON() for Typed Errors (Phase 1)

Add alongside `Error()` in `root.go`:

```go
// ErrorJSON outputs a structured CLIError. Use this when richer error context
// (entity key, valid transitions) is available. Falls back to Error() in human mode.
func ErrorJSON(e CLIError) {
    e.Error = true // Ensure always set
    if GlobalConfig.JSON {
        _ = OutputJSON(&e)
        return
    }
    // Human-readable fallback
    Error(e.Message)
}
```

### Cobra Error Interception

**Problem**: When a `RunE` function returns an `error`, Cobra prints it to stderr as plain text. With `SilenceErrors = true`, Cobra suppresses this but the error is still returned from `Execute()`.

**Solution**: Set `RootCmd.SilenceErrors = true` in `root.go init()` and handle the error in `cmd/shark/main.go`.

**File**: `internal/cli/root.go` - add to `init()`:

```go
RootCmd.SilenceErrors = true
// SilenceUsage prevents Cobra from printing usage on every error.
// Errors from RunE are semantic failures, not usage errors.
RootCmd.SilenceUsage = true
```

**File**: `cmd/shark/main.go` - update `main()`:

```go
func main() {
    // ... version setup ...
    cli.SetVersion(version)

    if err := cli.RootCmd.Execute(); err != nil {
        // In JSON mode, output structured error to stdout
        if cli.GlobalConfig.JSON {
            _ = cli.OutputJSON(&cli.CLIError{
                Error:   true,
                Code:    cli.ErrCodeCommandError,
                Message: err.Error(),
            })
        } else {
            // Human mode: print to stderr (Cobra silenced its own output)
            fmt.Fprintln(os.Stderr, "Error:", err)
        }
        os.Exit(1)
    }
}
```

Note: `cli.GlobalConfig.JSON` is set by `PersistentPreRunE` which runs before `RunE`. If `RunE` returns an error, `GlobalConfig.JSON` is already set correctly from flag/env processing.

Edge case: If `PersistentPreRunE` itself fails (e.g., config init error), `GlobalConfig.JSON` may not yet reflect the `--json` flag. This is acceptable - the failure mode is config-level, and outputting plain text for a config failure is reasonable.

### Phase 2 (Optional, Selective): Typed Error Codes

Update high-value call sites in commands to use `ErrorJSON()` with proper codes. Start with the most common patterns:

**NotFound errors** (example from `feature_helpers.go`):
```go
// Before
cli.Error(fmt.Sprintf("Feature %s not found", featureKey))

// After
cli.ErrorJSON(cli.CLIError{
    Code:      cli.ErrCodeNotFound,
    Message:   fmt.Sprintf("feature %s not found", featureKey),
    Entity:    "feature",
    EntityKey: featureKey,
})
```

**InvalidStatusTransition errors**: The `InvalidStatusTransitionError()` function in `errors.go` already captures `allowedTransitions []string`. Callers that use this function can be updated to emit:
```go
cli.ErrorJSON(cli.CLIError{
    Code:             cli.ErrCodeInvalidTransition,
    Message:          err.Error(),
    CurrentStatus:    currentStatus,
    ValidTransitions: allowedTransitions,
})
```

Phase 2 is not required for the initial implementation. The Phase 1 generic `Error()` modification handles all existing call sites automatically.

### Stdout vs Stderr Routing Decision

In JSON mode, ALL output goes to stdout:
- Successful JSON responses: stdout (existing)
- Error JSON: stdout (new)

Rationale: AI agents reading from stdout can parse a single stream. Checking whether to read stdout or stderr based on exit code is an extra step that JSON mode should eliminate. The `"error": true` field distinguishes error responses from success responses.

In human mode, the existing routing is unchanged:
- Success/info/warning: stdout (via pterm)
- Errors: stderr (via `fmt.Fprintln(os.Stderr, ...)`)

### File List for T-003

| File | Changes |
|------|---------|
| `internal/cli/cli_error.go` | New file: `CLIError` struct and error code constants |
| `internal/cli/root.go` | Modify `Error()` for JSON mode; add `ErrorJSON()`; set `SilenceErrors = true` and `SilenceUsage = true` in `init()` |
| `cmd/shark/main.go` | Handle error from `cli.RootCmd.Execute()` with JSON output when in JSON mode |
| `internal/cli/commands/errors.go` | Phase 2 only: add `CLIError`-returning variants (not required for Phase 1) |

### Tests for T-003

```go
// Test cli.Error() in JSON mode outputs to stdout as JSON
func TestError_JSONMode(t *testing.T) {
    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    GlobalConfig.JSON = true
    Error("something went wrong")

    w.Close()
    os.Stdout = old
    var buf bytes.Buffer
    buf.ReadFrom(r)

    var result CLIError
    json.Unmarshal(buf.Bytes(), &result)
    assert.True(t, result.Error)
    assert.Equal(t, "COMMAND_ERROR", result.Code)
    assert.Equal(t, "something went wrong", result.Message)
}

// Test cli.Error() in human mode outputs to stderr
func TestError_HumanMode(t *testing.T) {
    GlobalConfig.JSON = false
    // Verify it writes to stderr, not stdout
    // (pterm.Error.Println writes to stderr by default)
}
```

---

## T-E17-F08-004: --field Flag for Targeted Field Extraction

### Objective

Add a global `--field <name>` flag that extracts a single named field from JSON output. For object responses, print the field value. For array responses, print one value per line. Dot-notation paths (e.g., `status_breakdown.todo`) are supported.

### Config Struct Addition

**File**: `internal/cli/root.go`

```go
type Config struct {
    JSON       bool
    NoColor    bool
    Verbose    bool
    ConfigFile string
    DBPath     string
    Field      string  // --field flag: extract named field from JSON output
}
```

### Flag Registration

In `init()` in `root.go`, after the existing persistent flag registrations:

```go
RootCmd.PersistentFlags().StringVar(&GlobalConfig.Field, "field", "",
    "Extract a single field from JSON output (e.g., --field=status, --field=progress.weighted_pct)")
```

No Viper binding is needed for `--field`. It is purely a CLI presentation concern.

### Implied JSON Mode

When `--field` is set, JSON mode must be active because field extraction requires serializing to JSON first. Add to `initConfig()` after the `SHARK_OUTPUT` block:

```go
// --field implies JSON mode since field extraction requires JSON serialization.
if GlobalConfig.Field != "" {
    GlobalConfig.JSON = true
}
```

### Modified OutputJSON()

The cleanest approach is to make `OutputJSON()` field-aware. Every command that calls `cli.OutputJSON(data)` gains `--field` support with zero per-command changes.

**File**: `internal/cli/root.go`

```go
// OutputJSON outputs data in JSON format.
// If GlobalConfig.Field is set, extracts and prints only that field value.
// CLIError values bypass field extraction and are always output in full.
func OutputJSON(data interface{}) error {
    // Never apply field extraction to error responses
    if _, isError := data.(*CLIError); isError {
        return outputJSONRaw(data)
    }
    if GlobalConfig.Field != "" {
        return OutputField(data, GlobalConfig.Field)
    }
    return outputJSONRaw(data)
}

// outputJSONRaw writes data as indented JSON to stdout without field filtering.
func outputJSONRaw(data interface{}) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

### OutputField() and Helper Functions

Add these functions to `root.go` (or extract to a new `internal/cli/field_output.go` file if preferred for organization):

```go
// FieldNotFoundError is returned when the requested field does not exist.
type FieldNotFoundError struct {
    Field string
}

func (e *FieldNotFoundError) Error() string {
    return fmt.Sprintf("field %q not found in output", e.Field)
}

// OutputField extracts a named field from data and prints the value.
// For objects: prints the field value.
// For arrays: prints the extracted field from each element, one per line.
// Supports dot-notation: "progress.weighted_pct" traverses nested objects.
func OutputField(data interface{}, fieldPath string) error {
    b, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to serialize data: %w", err)
    }
    var parsed interface{}
    if err := json.Unmarshal(b, &parsed); err != nil {
        return fmt.Errorf("failed to parse serialized data: %w", err)
    }
    return extractAndPrintField(parsed, fieldPath)
}

func extractAndPrintField(data interface{}, fieldPath string) error {
    switch v := data.(type) {
    case map[string]interface{}:
        val, err := lookupField(v, fieldPath)
        if err != nil {
            return err
        }
        printFieldValue(val)
        return nil
    case []interface{}:
        for _, item := range v {
            if obj, ok := item.(map[string]interface{}); ok {
                val, err := lookupField(obj, fieldPath)
                if err != nil {
                    continue // Skip array elements that lack the field
                }
                printFieldValue(val)
            }
        }
        return nil
    default:
        return &FieldNotFoundError{Field: fieldPath}
    }
}

// lookupField retrieves the value at fieldPath from obj.
// Supports one level of dot-notation: "parent.child"
func lookupField(obj map[string]interface{}, fieldPath string) (interface{}, error) {
    parts := strings.SplitN(fieldPath, ".", 2)
    val, ok := obj[parts[0]]
    if !ok {
        return nil, &FieldNotFoundError{Field: fieldPath}
    }
    if len(parts) == 2 {
        nested, ok := val.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("field %q is not an object, cannot traverse to %q", parts[0], parts[1])
        }
        return lookupField(nested, parts[1])
    }
    return val, nil
}

// printFieldValue prints a field value in a script-friendly format.
// Strings are printed bare (no quotes). Numbers, bools, null use their natural form.
// Nested objects and arrays are re-serialized as compact JSON.
func printFieldValue(val interface{}) {
    switch v := val.(type) {
    case string:
        fmt.Println(v)
    case float64:
        // JSON numbers are float64; print as integer when there is no fractional part
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
        // Arrays and objects: re-serialize as compact JSON
        b, _ := json.Marshal(v)
        fmt.Println(string(b))
    }
}
```

### Exit Code for Field Not Found

The existing exit code convention: 0=success, 1=not found, 2=DB error, 3=invalid state. Add exit code 4 for field extraction failure to distinguish from entity-not-found.

**File**: `cmd/shark/main.go`

```go
if err := cli.RootCmd.Execute(); err != nil {
    // Check for field not found (exit code 4)
    var fieldErr *cli.FieldNotFoundError
    if errors.As(err, &fieldErr) {
        if cli.GlobalConfig.JSON {
            _ = cli.OutputJSON(&cli.CLIError{
                Error:   true,
                Code:    "FIELD_NOT_FOUND",
                Message: err.Error(),
            })
        } else {
            fmt.Fprintln(os.Stderr, "Error:", err)
        }
        os.Exit(4)
    }

    // Other errors
    if cli.GlobalConfig.JSON {
        _ = cli.OutputJSON(&cli.CLIError{
            Error:   true,
            Code:    cli.ErrCodeCommandError,
            Message: err.Error(),
        })
    } else {
        fmt.Fprintln(os.Stderr, "Error:", err)
    }
    os.Exit(1)
}
```

`errors` needs to be imported in `main.go`.

### Behavior with List Commands

List commands that return `{"results": [...], "count": N}` responses:

- `--field=results` returns the full JSON array
- `--field=count` returns the integer
- `--field=title` on a results array: no auto-traversal; returns `FieldNotFoundError` because the top-level object has no `title` field. Agents should use `--field=results` and process the array, or use `--json` and pipe through `jq`.

This deliberate simplicity avoids surprising auto-traversal behavior. The dot-notation handles one level: `--field=results.0` is not supported (would require JSONPath, which is out of scope).

### File List for T-004

| File | Changes |
|------|---------|
| `internal/cli/root.go` | Add `Field string` to `Config`; register `--field` persistent flag; add `FieldNotFoundError` type; modify `OutputJSON()` to be field-aware; add `outputJSONRaw()`, `OutputField()`, `extractAndPrintField()`, `lookupField()`, `printFieldValue()` functions; set `GlobalConfig.JSON = true` when `Field != ""` in `initConfig()` |
| `cmd/shark/main.go` | Handle `FieldNotFoundError` with exit code 4 |

Alternatively, if `root.go` becomes large, the field extraction functions (`OutputField`, `extractAndPrintField`, `lookupField`, `printFieldValue`, `FieldNotFoundError`) can be moved to a new `internal/cli/field_output.go` file. This is an implementation-time decision.

### Tests for T-004

Table-driven unit tests covering:

```go
func TestOutputField_SimpleField(t *testing.T) {
    data := map[string]interface{}{"status": "in_progress", "key": "E17-F08-001"}
    // Test --field=status returns "in_progress"
}

func TestOutputField_DotNotation(t *testing.T) {
    data := map[string]interface{}{
        "progress": map[string]interface{}{"weighted_pct": 75.5},
    }
    // Test --field=progress.weighted_pct returns "75.5"
}

func TestOutputField_ArrayExtraction(t *testing.T) {
    data := []interface{}{
        map[string]interface{}{"key": "T001", "status": "todo"},
        map[string]interface{}{"key": "T002", "status": "in_progress"},
    }
    // Test --field=status returns "todo\nin_progress\n" (one per line)
}

func TestOutputField_MissingField(t *testing.T) {
    data := map[string]interface{}{"status": "todo"}
    err := OutputField(data, "nonexistent")
    var fieldErr *FieldNotFoundError
    assert.True(t, errors.As(err, &fieldErr))
}

func TestOutputField_IntegerValue(t *testing.T) {
    data := map[string]interface{}{"count": float64(42)}
    // Test --field=count prints "42" (not "42.0")
}

func TestOutputField_ErrorBypassesFieldExtraction(t *testing.T) {
    GlobalConfig.Field = "message"
    err := &CLIError{Error: true, Code: "NOT_FOUND", Message: "task not found"}
    // Test OutputJSON(err) outputs full error JSON, not just the "message" field
}
```

---

## Cross-Task Interaction Notes

### T-002 + T-003: SHARK_OUTPUT and JSON Errors

When `SHARK_OUTPUT=json` is set, `initConfig()` sets `GlobalConfig.JSON = true`. Subsequently, `cli.Error()` emits JSON. Both mechanisms use the same `GlobalConfig.JSON` gate, so they compose correctly with no additional work.

### T-003 + T-004: JSON Errors and --field

When `--field=status` is used and a command fails, the error must NOT be run through field extraction. The `OutputJSON()` modification achieves this by checking for `*CLIError` before applying field extraction. Errors always output their full JSON body.

### T-001 and SHARK_OUTPUT: Test ergonomics

After T-002 is implemented, the `--all` and `--order` flag changes in T-001 can be verified using `SHARK_OUTPUT=json shark task list --all` rather than appending `--json` to each test command.

---

## Implementation Order

The following order minimizes risk and maximizes test ergonomics:

**1. T-002: SHARK_OUTPUT (~8 lines)**
Rationale: Smallest change. Enables all subsequent testing without `--json` on every command. Ship first, validate env var behavior.

**2. T-001: Flag Normalization (~30 lines across 6 files)**
Rationale: Purely mechanical changes using `MarkDeprecated()`. No logic changes. The alias-OR pattern for `--all` is straightforward. No dependencies on T-002, T-003, or T-004.

**3. T-003: Structured JSON Errors (~50 lines across 3 files)**
Rationale: Creates the `CLIError` struct and modifies `cli.Error()`. Do the generic Phase 1 changes only. Do not update all 60+ call sites - the modified `cli.Error()` handles them automatically. After this task, all errors in JSON mode are structured.

**4. T-004: --field Flag (~80 lines across 2 files)**
Rationale: Builds on the stable JSON output infrastructure. Field extraction uses `OutputJSON()` which is already correct after T-003. Implement last to avoid modifying `OutputJSON()` multiple times.

---

## Summary: Files Changed Per Task

| File | T-001 | T-002 | T-003 | T-004 |
|------|-------|-------|-------|-------|
| `internal/cli/root.go` | - | +8 lines | +30 lines | +70 lines |
| `internal/cli/cli_error.go` | - | - | new file | - |
| `internal/cli/commands/task_helpers.go` | +6 lines | - | - | - |
| `internal/cli/commands/task.go` | +4 lines | - | - | - |
| `internal/cli/commands/feature.go` | +10 lines | - | - | - |
| `internal/cli/commands/feature_helpers.go` | +6 lines | - | - | - |
| `internal/cli/commands/list.go` | +6 lines | - | - | - |
| `internal/cli/commands/create.go` | +4 lines | - | - | - |
| `cmd/shark/main.go` | - | - | +12 lines | +8 lines |

Estimated total code change: ~160 lines of new/modified code across 9 files.

---

## Test Strategy Summary

All tests for E17-F08 are unit tests. No real database access is required.

| Task | Test File | Test Type | Key Scenarios |
|------|-----------|-----------|---------------|
| T-001 | `task_helpers_test.go`, `feature_test.go` | Unit | `--all` is accepted; `--execution-order` still works with deprecation warning; `--order` works on feature commands |
| T-002 | `root_test.go` | Unit | `SHARK_OUTPUT=json` sets `GlobalConfig.JSON`; does not override explicit `--json=false`; `SHARK_OUTPUT=table` has no effect |
| T-003 | `root_test.go` | Unit | `cli.Error()` in JSON mode outputs to stdout as valid JSON; `ErrorJSON()` includes code and entity fields; `SilenceErrors` is set on `RootCmd` |
| T-004 | `root_test.go` or `field_output_test.go` | Unit | Simple field; dot-notation; array extraction; missing field returns `FieldNotFoundError`; integer formatting; error JSON bypasses field extraction |

The quality gate `make fmt && make lint && make test` must pass after each task.

---

*Last Updated: 2026-02-25*
