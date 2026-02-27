# E17-F08 Test Plan: CLI Enhancements and Quick Wins

**Feature:** E17-F08-cli-enhancements-and-quick-wins
**Date:** 2026-02-25
**Author:** QA Agent
**Status:** Ready for Review

---

## Table of Contents

1. [Test Scope and Objectives](#1-test-scope-and-objectives)
2. [Test Approach](#2-test-approach)
3. [T-001: Flag Normalization](#3-t-001-flag-normalization)
4. [T-002: SHARK_OUTPUT Environment Variable](#4-t-002-shark_output-environment-variable)
5. [T-003: Structured JSON Error Output](#5-t-003-structured-json-error-output)
6. [T-004: --field Flag for Field Extraction](#6-t-004---field-flag-for-field-extraction)
7. [Cross-Task Interaction Tests](#7-cross-task-interaction-tests)
8. [Backward Compatibility Tests](#8-backward-compatibility-tests)
9. [UAT Scenario Mapping](#9-uat-scenario-mapping)
10. [Quality Gates](#10-quality-gates)

---

## 1. Test Scope and Objectives

### Scope

This test plan covers all four tasks in E17-F08:

| Task | Title | Key Concern |
|------|-------|-------------|
| T-E17-F08-001 | Flag Normalization | Old flags still work; new flags work; deprecation warnings on stderr |
| T-E17-F08-002 | SHARK_OUTPUT Env Var | Env var enables JSON mode; flag overrides env; unknown values ignored |
| T-E17-F08-003 | Structured JSON Errors | JSON error format; stdout vs stderr routing; exit codes |
| T-E17-F08-004 | --field Flag | Field extraction from objects and arrays; missing field exit code; error bypass |

### Out of Scope

- Phase 2 (optional) typed error codes at individual call sites (T-003 Phase 2)
- Batch operations (E17-F07)
- Status subcommands (E17-F01)
- Progress commands (E17-F06)

### Objectives

1. Verify all new CLI behaviors match the technical-design.md specification.
2. Verify no regression in existing command output, exit codes, or behavior.
3. Verify correct output stream routing (stdout vs stderr) in all modes.
4. Verify deprecation warnings appear on stderr and are suppressed in JSON mode.
5. Verify the four tasks compose correctly together.

---

## 2. Test Approach

### Test Levels

| Level | Description | Files |
|-------|-------------|-------|
| Unit tests | Pure logic functions with no DB or I/O | `root_test.go`, `field_output_test.go` |
| CLI tests | Command flag parsing, output format, stream routing | `task_test.go`, `feature_test.go`, `list_test.go`, `root_test.go` |
| Integration tests | End-to-end execution against real binary | Manual / shell scripts |

All unit and CLI tests use mocked repositories. No real database access in unit or CLI tests. Integration tests use a test project with seeded data.

### Test Data Requirements

For integration tests:
- At least one epic, feature, and task in known statuses
- A task in `todo` status for start/transition tests
- A task in `completed` status for invalid-transition error tests

### Files Under Test

| File | Tasks |
|------|-------|
| `internal/cli/root.go` | T-002, T-003, T-004 |
| `internal/cli/cli_error.go` | T-003 |
| `internal/cli/commands/task_helpers.go` | T-001 |
| `internal/cli/commands/task.go` | T-001 |
| `internal/cli/commands/feature.go` | T-001 |
| `internal/cli/commands/feature_helpers.go` | T-001 |
| `internal/cli/commands/list.go` | T-001 |
| `internal/cli/commands/create.go` | T-001 |
| `cmd/shark/main.go` | T-003, T-004 |

---

## 3. T-001: Flag Normalization

### Overview

T-001 adds `--order` as an alias for `--execution-order` on feature commands (task commands already have `--order`), and adds `--all` as an alias for `--show-all` on all list commands. Both old flags are marked deprecated via `cobra.Command.MarkDeprecated()`.

### TC-001-01: `--order` flag on `feature create`

**Test Type:** CLI unit test
**File:** `feature_test.go`

**Preconditions:** Epic E07 exists (or is mocked)

**Steps:**
1. Run `shark feature create E07 "Test Feature" --order=3`

**Expected Results:**
- Exit code 0
- Feature created with `execution_order = 3`
- No deprecation warning emitted

**Pass/Fail:** Feature created; `execution_order` field in JSON output equals `3`.

---

### TC-001-02: `--order` flag on `feature update`

**Test Type:** CLI unit test
**File:** `feature_test.go`

**Preconditions:** Feature E07-F01 exists

**Steps:**
1. Run `shark feature update E07-F01 --order=5`

**Expected Results:**
- Exit code 0
- Feature updated with `execution_order = 5`

**Pass/Fail:** Updated feature JSON shows `execution_order = 5`.

---

### TC-001-03: `--execution-order` on `feature create` produces deprecation warning

**Test Type:** CLI unit test
**File:** `feature_test.go`

**Steps:**
1. Capture stderr
2. Run `shark feature create E07 "Test Feature" --execution-order=2`

**Expected Results:**
- Exit code 0
- Feature created with `execution_order = 2` (flag still works)
- Stderr contains: `Flag --execution-order has been deprecated, use --order instead`
- Stdout contains only the feature output (no deprecation text mixed in)

**Pass/Fail:** Feature created successfully; deprecation warning on stderr only.

---

### TC-001-04: `--execution-order` on `task create` produces deprecation warning

**Test Type:** CLI unit test
**File:** `task_helpers_test.go`

**Steps:**
1. Capture stderr
2. Run `shark task create E07 F01 "Test Task" --execution-order=1`

**Expected Results:**
- Exit code 0
- Task created with `execution_order = 1`
- Stderr contains deprecation warning for `--execution-order`

**Pass/Fail:** Task created successfully; warning on stderr only.

---

### TC-001-05: `--all` flag on `task list`

**Test Type:** CLI unit test
**File:** `task_test.go`

**Preconditions:** At least one completed task exists in a feature

**Steps:**
1. Run `shark task list --all --json`

**Expected Results:**
- Exit code 0
- Completed tasks appear in the output
- No deprecation warning (new flag, not deprecated)

**Pass/Fail:** Response includes completed tasks.

---

### TC-001-06: `--show-all` on `task list` produces deprecation warning

**Test Type:** CLI unit test
**File:** `task_test.go`

**Steps:**
1. Capture stderr
2. Run `shark task list --show-all --json`

**Expected Results:**
- Exit code 0
- Completed tasks appear in the output (flag still works)
- Stderr contains: `Flag --show-all has been deprecated, use --all instead`
- Stdout is valid JSON only (no deprecation text)

**Pass/Fail:** Old flag functional; warning on stderr; JSON output clean.

---

### TC-001-07: `--all` flag on `feature list`

**Test Type:** CLI unit test
**File:** `feature_test.go`

**Steps:**
1. Run `shark feature list --all --json`

**Expected Results:**
- Exit code 0
- Completed features appear in output
- No deprecation warning

**Pass/Fail:** Response includes completed features.

---

### TC-001-08: `--show-all` on `feature list` produces deprecation warning

**Test Type:** CLI unit test
**File:** `feature_test.go`

**Steps:**
1. Capture stderr
2. Run `shark feature list --show-all --json`

**Expected Results:**
- Exit code 0
- Output includes completed features
- Stderr contains deprecation warning

**Pass/Fail:** Old flag works; warning on stderr only.

---

### TC-001-09: `--all` flag on `list` (smart dispatcher)

**Test Type:** CLI unit test
**File:** `list_test.go` or integration

**Steps:**
1. Run `shark list E07 --all --json`

**Expected Results:**
- Exit code 0
- Output includes completed items for epic E07

**Pass/Fail:** `--all` accepted on smart list dispatcher.

---

### TC-001-10: `--show-all` on `list` (smart dispatcher) produces deprecation warning

**Test Type:** CLI unit test

**Steps:**
1. Capture stderr
2. Run `shark list E07 --show-all --json`

**Expected Results:**
- Exit code 0
- Completed items appear
- Deprecation warning on stderr

**Pass/Fail:** Old flag functional; warning on stderr.

---

### TC-001-11: Both `--all` and `--show-all` together produces single result (OR semantics)

**Test Type:** CLI unit test

**Steps:**
1. Run `shark task list --all --show-all --json`

**Expected Results:**
- Exit code 0
- Completed tasks appear (same as using just `--all`)
- No errors about conflicting flags

**Pass/Fail:** Both flags accepted together; behavior same as either alone.

---

### TC-001-12: Deprecation warnings are suppressed in JSON mode

**Test Type:** CLI integration test

**Preconditions:** `SHARK_OUTPUT=json` is set

**Steps:**
1. Capture all stdout
2. Run `SHARK_OUTPUT=json shark task list --show-all`
3. Attempt to parse stdout as JSON

**Expected Results:**
- Exit code 0
- Stdout is valid JSON (no deprecation text mixed in)
- Stderr may contain the deprecation warning (acceptable) OR it may be suppressed

**Pass/Fail:** stdout parses as valid JSON without corruption from deprecation text.

**Note:** Cobra prints deprecation warnings to stderr, not stdout. This test confirms that JSON-mode stdout is not polluted.

---

### TC-001-13: `--execution-order` on `create task` (unified create command) produces deprecation warning

**Test Type:** CLI unit test
**File:** `create_test.go`

**Steps:**
1. Capture stderr
2. Run `shark create task E07-F01 "Test" --execution-order=3`

**Expected Results:**
- Exit code 0
- Task created with `execution_order = 3`
- Deprecation warning on stderr

**Pass/Fail:** Works; warning present.

---

### TC-001-14: `--execution-order` on `create feature` (unified create command) produces deprecation warning

**Test Type:** CLI unit test

**Steps:**
1. Capture stderr
2. Run `shark create feature E07 "Test" --execution-order=1`

**Expected Results:**
- Exit code 0
- Feature created with `execution_order = 1`
- Deprecation warning on stderr

**Pass/Fail:** Works; warning present.

---

## 4. T-002: SHARK_OUTPUT Environment Variable

### Overview

`SHARK_OUTPUT=json` enables JSON mode session-wide, so AI agents can set it once in the environment instead of passing `--json` to every command. Precedence: `--json` flag > `PM_JSON=true` > `SHARK_OUTPUT=json` > default (false).

### TC-002-01: `SHARK_OUTPUT=json` enables JSON mode

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `SHARK_OUTPUT=json` via `t.Setenv()`
2. Set `GlobalConfig.JSON = false`
3. Call the relevant portion of `initConfig()` or the check block directly
4. Assert `GlobalConfig.JSON == true`

**Expected Results:**
- `GlobalConfig.JSON` is `true` after config initialization

**Pass/Fail:** JSON mode is activated by env var.

---

### TC-002-02: `SHARK_OUTPUT=json` makes commands output JSON without `--json` flag

**Test Type:** CLI integration test

**Steps:**
1. Set `SHARK_OUTPUT=json` in the test environment
2. Run `shark task list` (no `--json` flag)
3. Attempt to parse stdout as JSON

**Expected Results:**
- Exit code 0
- Stdout is valid JSON
- Output format is identical to `shark task list --json`

**Pass/Fail:** JSON output produced without explicit `--json` flag.

---

### TC-002-03: `--json` flag takes precedence over `SHARK_OUTPUT=json`

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `SHARK_OUTPUT=json` via `t.Setenv()`
2. Simulate `GlobalConfig.JSON = true` (as if `--json` was passed)
3. Call `initConfig()` check block
4. Assert `GlobalConfig.JSON` remains `true`

**Expected Results:**
- `GlobalConfig.JSON` remains `true`
- No conflict or reset

**Pass/Fail:** Flag value preserved; env var does not override.

---

### TC-002-04: `--json=false` flag overrides `SHARK_OUTPUT=json`

**Test Type:** CLI integration test

**Preconditions:** `SHARK_OUTPUT=json` set in environment

**Steps:**
1. Run `shark task list --json=false` with `SHARK_OUTPUT=json` in environment (if Cobra supports this)

**Expected Results:**
- Exit code 0
- Output is human-readable table format, not JSON
- `--json=false` suppresses JSON mode even with env var set

**Pass/Fail:** Explicit flag to disable JSON overrides the environment variable.

**Note:** Per technical design, Cobra sets the flag to `false`, Viper reads it as `false`, and the `SHARK_OUTPUT` check (`if !GlobalConfig.JSON`) does not activate. Verify this behavior.

---

### TC-002-05: `SHARK_OUTPUT=table` has no effect

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `SHARK_OUTPUT=table` via `t.Setenv()`
2. Set `GlobalConfig.JSON = false`
3. Call `initConfig()` check block
4. Assert `GlobalConfig.JSON == false`

**Expected Results:**
- `GlobalConfig.JSON` remains `false`
- Unknown value is silently ignored

**Pass/Fail:** Non-json values have no effect.

---

### TC-002-06: Empty `SHARK_OUTPUT` has no effect

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `SHARK_OUTPUT=""` via `t.Setenv()`
2. Set `GlobalConfig.JSON = false`
3. Call `initConfig()` check block
4. Assert `GlobalConfig.JSON == false`

**Expected Results:**
- `GlobalConfig.JSON` remains `false`

**Pass/Fail:** Empty value has no effect.

---

### TC-002-07: `PM_JSON=true` takes precedence over `SHARK_OUTPUT=json`

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `PM_JSON=true` and `SHARK_OUTPUT=json`
2. Set `GlobalConfig.JSON = false`
3. Run full `initConfig()` initialization
4. Assert `GlobalConfig.JSON == true`

**Expected Results:**
- JSON mode enabled (either source would enable it, but Viper handles `PM_JSON`)
- No conflict

**Pass/Fail:** Both sources agree on JSON mode.

---

### TC-002-08: `SHARK_OUTPUT` variable is case-sensitive

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `SHARK_OUTPUT=JSON` (uppercase)
2. Set `GlobalConfig.JSON = false`
3. Call `initConfig()` check block
4. Assert `GlobalConfig.JSON == false`

**Expected Results:**
- `GlobalConfig.JSON` remains `false`
- Only lowercase `json` value is recognized

**Pass/Fail:** Case-insensitive match is NOT performed; only `json` works.

---

## 5. T-003: Structured JSON Error Output

### Overview

When `--json` mode is active, `cli.Error()` emits a structured JSON object to stdout instead of human-readable text to stderr. A new `CLIError` struct is defined with `error`, `code`, `message`, and optional `entity`/`entity_key`/`current_status`/`valid_transitions` fields.

### TC-003-01: `cli.Error()` in JSON mode outputs structured JSON to stdout

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `GlobalConfig.JSON = true`
2. Capture stdout (redirect `os.Stdout` to a pipe)
3. Call `cli.Error("something went wrong")`
4. Restore stdout
5. Parse captured output as JSON into `CLIError` struct

**Expected Results:**
- Captured output is valid JSON
- `result.Error == true`
- `result.Code == "COMMAND_ERROR"`
- `result.Message == "something went wrong"`

**Pass/Fail:** All three assertions pass.

---

### TC-003-02: `cli.Error()` in human mode outputs to stderr

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `GlobalConfig.JSON = false`
2. Capture stderr
3. Call `cli.Error("something went wrong")`
4. Assert captured stderr contains the error message
5. Assert stdout is empty (or not modified)

**Expected Results:**
- Error message appears on stderr
- Stdout is not polluted

**Pass/Fail:** Human-mode errors go to stderr only.

---

### TC-003-03: `cli.ErrorJSON()` outputs full structured error with all fields

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `GlobalConfig.JSON = true`
2. Capture stdout
3. Call:
   ```go
   cli.ErrorJSON(cli.CLIError{
       Code:          cli.ErrCodeNotFound,
       Message:       "feature E99-F99 not found",
       Entity:        "feature",
       EntityKey:     "E99-F99",
   })
   ```
4. Parse captured output as JSON

**Expected Results:**
- `result.Error == true` (always set by `ErrorJSON`)
- `result.Code == "NOT_FOUND"`
- `result.Message == "feature E99-F99 not found"`
- `result.Entity == "feature"`
- `result.EntityKey == "E99-F99"`

**Pass/Fail:** All fields present and correct.

---

### TC-003-04: `cli.ErrorJSON()` in human mode falls back to plain error message

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `GlobalConfig.JSON = false`
2. Capture stderr
3. Call `cli.ErrorJSON(cli.CLIError{Code: "NOT_FOUND", Message: "not found"})`
4. Assert stderr contains the message text

**Expected Results:**
- Error message readable on stderr
- No raw JSON printed to terminal

**Pass/Fail:** Human fallback works.

---

### TC-003-05: Cobra's `SilenceErrors` is set on `RootCmd`

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Access `cli.RootCmd.SilenceErrors`
2. Assert it is `true`

**Expected Results:**
- `RootCmd.SilenceErrors == true`

**Pass/Fail:** Cobra's own error printing is silenced.

---

### TC-003-06: Cobra's `SilenceUsage` is set on `RootCmd`

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Access `cli.RootCmd.SilenceUsage`
2. Assert it is `true`

**Expected Results:**
- `RootCmd.SilenceUsage == true`

**Pass/Fail:** Usage is not printed on every error.

---

### TC-003-07: `main.go` outputs structured JSON error when command fails in JSON mode

**Test Type:** Integration test

**Preconditions:** Binary is built

**Steps:**
1. Run `SHARK_OUTPUT=json ./bin/shark task start E99-F99-999` (non-existent task)
2. Capture stdout and exit code

**Expected Results:**
- Exit code is non-zero (1 for not-found, or 2 for general error)
- Stdout is valid JSON
- JSON contains `"error": true`
- JSON contains a `"code"` field
- Stderr is empty (all output on stdout in JSON mode)

**Pass/Fail:** Error on stdout as JSON; stderr clean.

---

### TC-003-08: `main.go` outputs plain text error when command fails in human mode

**Test Type:** Integration test

**Preconditions:** Binary is built, `SHARK_OUTPUT` unset

**Steps:**
1. Run `./bin/shark task start E99-F99-999` (non-existent task)
2. Capture stdout, stderr, and exit code

**Expected Results:**
- Exit code non-zero
- Error message appears on stderr
- Stdout is empty

**Pass/Fail:** Human-mode error on stderr; stdout clean.

---

### TC-003-09: Error in JSON mode does NOT print usage

**Test Type:** Integration test

**Steps:**
1. Run `SHARK_OUTPUT=json ./bin/shark task start` (missing required argument)
2. Capture stdout

**Expected Results:**
- Exit code non-zero
- Stdout is JSON error object (or empty if the error is caught before RunE)
- Stdout does NOT contain command usage/help text mixed into JSON

**Pass/Fail:** No usage text in JSON output stream.

---

### TC-003-10: `CLIError` struct has correct field names and JSON tags

**Test Type:** Unit test
**File:** `root_test.go` or `cli_error_test.go`

**Steps:**
1. Create `CLIError{Error: true, Code: "NOT_FOUND", Message: "msg", Entity: "task", EntityKey: "E01-F01-001"}`
2. Marshal to JSON
3. Unmarshal and verify JSON key names

**Expected Results:**
JSON output contains keys: `"error"`, `"code"`, `"message"`, `"entity"`, `"entity_key"`

**Pass/Fail:** JSON key names match specification exactly.

---

### TC-003-11: Error code constants are defined

**Test Type:** Unit test

**Steps:**
1. Assert the following constants exist and have correct values:
   - `cli.ErrCodeNotFound == "NOT_FOUND"`
   - `cli.ErrCodeInvalidTransition == "INVALID_TRANSITION"`
   - `cli.ErrCodeValidationError == "VALIDATION_ERROR"`
   - `cli.ErrCodeDatabaseError == "DATABASE_ERROR"`
   - `cli.ErrCodeInvalidArgs == "INVALID_ARGS"`
   - `cli.ErrCodeCommandError == "COMMAND_ERROR"`

**Pass/Fail:** All constants defined with correct string values.

---

### TC-003-12: Optional fields (`entity`, `entity_key`, `current_status`, `valid_transitions`) are omitted from JSON when empty

**Test Type:** Unit test

**Steps:**
1. Create `CLIError{Error: true, Code: "COMMAND_ERROR", Message: "msg"}`
2. Marshal to JSON
3. Verify `entity`, `entity_key`, `current_status`, `valid_transitions` are NOT present in the JSON

**Expected Results:**
- JSON does not contain null or empty fields for optional properties

**Pass/Fail:** `omitempty` tag works correctly.

---

## 6. T-004: --field Flag for Field Extraction

### Overview

`--field <name>` is a global persistent flag that extracts a single named field from JSON output. For objects, it prints the field value. For arrays, it prints one value per line. Dot-notation (`progress.weighted_pct`) is supported for one level of nesting. When `--field` is set, JSON mode is automatically implied. Missing field returns exit code 4.

### TC-004-01: `--field` extracts a simple string field from an object

**Test Type:** Unit test
**File:** `root_test.go` or `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"status": "in_progress", "key": "E17-F08-001"}, "status")`
2. Capture stdout
3. Assert output is `"in_progress\n"`

**Expected Results:**
- Output is bare string value without JSON quotes
- Trailing newline present
- No JSON wrapping

**Pass/Fail:** Raw string value printed.

---

### TC-004-02: `--field` extracts a numeric field as integer (not float)

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"count": float64(42)}, "count")`
2. Capture stdout
3. Assert output is `"42\n"` (not `"42.0\n"`)

**Expected Results:**
- Integer numbers are printed without decimal point

**Pass/Fail:** `42` not `42.0`.

---

### TC-004-03: `--field` extracts a float field with fractional part

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"progress_pct": float64(78.5)}, "progress_pct")`
2. Capture stdout
3. Assert output is `"78.5\n"`

**Expected Results:**
- Decimal numbers include fractional part

**Pass/Fail:** `78.5` printed correctly.

---

### TC-004-04: `--field` extracts a boolean field

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"is_blocked": true}, "is_blocked")`
2. Capture stdout
3. Assert output is `"true\n"`

**Expected Results:**
- Boolean printed as `true` or `false`

**Pass/Fail:** Boolean printed without quotes.

---

### TC-004-05: `--field` returns null for null value

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"notes": nil}, "notes")`
2. Capture stdout
3. Assert output is `"null\n"`

**Expected Results:**
- Null value prints as `null`

**Pass/Fail:** `null` printed.

---

### TC-004-06: `--field` with dot-notation extracts nested field

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call:
   ```go
   OutputField(map[string]interface{}{
       "progress": map[string]interface{}{"weighted_pct": float64(75.5)},
   }, "progress.weighted_pct")
   ```
2. Capture stdout
3. Assert output is `"75.5\n"`

**Expected Results:**
- Dot-notation traverses one level of nesting

**Pass/Fail:** Nested value extracted correctly.

---

### TC-004-07: `--field` with missing field returns `FieldNotFoundError`

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"status": "todo"}, "nonexistent")`
2. Assert returned error is a `*FieldNotFoundError`
3. Assert `fieldErr.Field == "nonexistent"`

**Expected Results:**
- Error is typed `*FieldNotFoundError`
- Error message contains the field name

**Pass/Fail:** Correct error type returned.

---

### TC-004-08: `--field` on an array extracts one value per element

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call:
   ```go
   OutputField([]interface{}{
       map[string]interface{}{"key": "T001", "status": "todo"},
       map[string]interface{}{"key": "T002", "status": "in_progress"},
   }, "status")
   ```
2. Capture stdout
3. Assert output is `"todo\nin_progress\n"`

**Expected Results:**
- One extracted value per line
- Order preserved

**Pass/Fail:** One value per array element printed.

---

### TC-004-09: `--field` on an array skips elements missing the field

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField` on an array where the second element lacks the `status` field:
   ```go
   []interface{}{
       map[string]interface{}{"key": "T001", "status": "todo"},
       map[string]interface{}{"key": "T002"},
   }
   ```
2. Use field `"status"`
3. Capture stdout
4. Assert output is `"todo\n"` (only first element)

**Expected Results:**
- Elements lacking the field are silently skipped
- No error returned for partial arrays

**Pass/Fail:** Skip behavior correct; no panic.

---

### TC-004-10: `--field` setting implies JSON mode

**Test Type:** Unit test
**File:** `root_test.go`

**Steps:**
1. Set `GlobalConfig.Field = "status"`
2. Set `GlobalConfig.JSON = false`
3. Call the `initConfig()` block that checks `GlobalConfig.Field`
4. Assert `GlobalConfig.JSON == true`

**Expected Results:**
- `GlobalConfig.JSON` automatically set to `true` when `Field` is non-empty

**Pass/Fail:** JSON mode implied by `--field`.

---

### TC-004-11: `--field` on a `get` command returns bare field value

**Test Type:** Integration test

**Preconditions:** A task exists with key `E07-F01-001` and status `todo`

**Steps:**
1. Run `./bin/shark get E07-F01-001 --field status`
2. Capture stdout

**Expected Results:**
- Exit code 0
- Stdout is exactly `todo\n`
- No JSON wrapping, no extra whitespace

**Pass/Fail:** Bare value returned.

---

### TC-004-12: `--field` on `task next` returns the task key

**Test Type:** Integration test

**Preconditions:** At least one task in `ready_for_development` or `todo` status exists

**Steps:**
1. Run `./bin/shark task next --field key`
2. Capture stdout

**Expected Results:**
- Exit code 0
- Output is a single task key string (e.g., `E07-F01-001`)
- No JSON wrapping

**Pass/Fail:** Task key extracted cleanly.

---

### TC-004-13: `--field` with missing field exits with code 4

**Test Type:** Integration test

**Preconditions:** Task exists

**Steps:**
1. Run `./bin/shark get E07-F01-001 --field nonexistent_field_xyz`
2. Capture exit code

**Expected Results:**
- Exit code is 4 (field not found)
- NOT exit code 1 (entity not found)
- Error message identifies the missing field

**Pass/Fail:** Exit code 4 for field-not-found; distinguishable from entity-not-found (exit 1).

---

### TC-004-14: `--field` with missing field in JSON mode outputs `FIELD_NOT_FOUND` error JSON

**Test Type:** Integration test

**Preconditions:** `SHARK_OUTPUT=json` set; task exists

**Steps:**
1. Run `SHARK_OUTPUT=json ./bin/shark get E07-F01-001 --field nonexistent_xyz`
2. Capture stdout and exit code
3. Parse stdout as JSON

**Expected Results:**
- Exit code 4
- Stdout is valid JSON
- JSON contains `"error": true`
- JSON contains `"code": "FIELD_NOT_FOUND"`

**Pass/Fail:** Structured JSON error with correct code and exit code 4.

---

### TC-004-15: CLIError responses bypass field extraction

**Test Type:** Unit test
**File:** `root_test.go` or `field_output_test.go`

**Steps:**
1. Set `GlobalConfig.JSON = true`
2. Set `GlobalConfig.Field = "message"`
3. Call `cli.OutputJSON(&cli.CLIError{Error: true, Code: "NOT_FOUND", Message: "task not found"})`
4. Capture stdout
5. Parse as JSON

**Expected Results:**
- Full `CLIError` JSON is output, not just the `message` field
- `result.Error == true` and `result.Code == "NOT_FOUND"` present in output

**Pass/Fail:** Error responses are never field-filtered.

---

### TC-004-16: `--field` on nested dot-notation where intermediate is not an object

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"name": "flat string"}, "name.child")`
2. Assert an error is returned

**Expected Results:**
- Error returned describing that `name` is not an object
- No panic

**Pass/Fail:** Graceful error on invalid traversal path.

---

### TC-004-17: `--field` on an array at the top level where field traversal would require second-level dot-notation

**Test Type:** Unit test

This tests the deliberate design decision: `--field=title` on a list response that wraps results in `{"results": [...], "count": N}` returns `FieldNotFoundError` because the top-level object has no `title` field.

**Steps:**
1. Call `OutputField(map[string]interface{}{"results": []interface{}{map[string]interface{}{"title": "T1"}}, "count": float64(1)}, "title")`
2. Assert `*FieldNotFoundError` is returned

**Expected Results:**
- Error returned for `title` not found at top level
- Auto-traversal into `results` array does NOT happen

**Pass/Fail:** No auto-traversal; error on missing top-level field.

---

### TC-004-18: `--field` on a nested object value returns compact JSON

**Test Type:** Unit test
**File:** `field_output_test.go`

**Steps:**
1. Call `OutputField(map[string]interface{}{"meta": map[string]interface{}{"a": 1, "b": 2}}, "meta")`
2. Capture stdout

**Expected Results:**
- Output is compact JSON of the nested object (e.g., `{"a":1,"b":2}`)
- Not the raw Go representation

**Pass/Fail:** Nested objects serialized as compact JSON.

---

## 7. Cross-Task Interaction Tests

These tests verify that T-002, T-003, and T-004 work correctly when combined.

### TC-CROSS-01: `SHARK_OUTPUT=json` + error → structured JSON error on stdout

**Test Type:** Integration test

**Steps:**
1. Set `SHARK_OUTPUT=json` in environment
2. Run `./bin/shark task start E99-F99-999` (non-existent task)
3. Capture stdout and exit code

**Expected Results:**
- Exit code non-zero
- Stdout is valid JSON with `"error": true`
- Stderr is empty

**Pass/Fail:** SHARK_OUTPUT activates JSON mode; Error() uses JSON path.

---

### TC-CROSS-02: `--field` + error → error JSON bypasses field extraction

**Test Type:** Integration test

**Preconditions:** Task E07-F01-001 exists

**Steps:**
1. Run `./bin/shark get E07-F01-001 --field nonexistent`
2. Capture stdout
3. Parse stdout as JSON

**Expected Results:**
- Exit code 4
- Stdout is a full `CLIError` JSON object (not just one extracted field)
- JSON contains `"error": true` and `"code": "FIELD_NOT_FOUND"`

**Pass/Fail:** Errors are never field-filtered.

---

### TC-CROSS-03: `SHARK_OUTPUT=json` + `--field` → field extracted from JSON output

**Test Type:** Integration test

**Preconditions:** Task E07-F01-001 exists with known status

**Steps:**
1. Set `SHARK_OUTPUT=json` in environment
2. Run `./bin/shark get E07-F01-001 --field status` (no explicit `--json`)
3. Capture stdout

**Expected Results:**
- Exit code 0
- Output is the bare status value (e.g., `todo`)
- Not full JSON (field extraction runs after SHARK_OUTPUT enables JSON)

**Pass/Fail:** SHARK_OUTPUT enables JSON mode, then `--field` extracts the value.

---

### TC-CROSS-04: `--all` (T-001) flag works when `SHARK_OUTPUT=json` (T-002) is set

**Test Type:** Integration test

**Steps:**
1. Set `SHARK_OUTPUT=json` in environment
2. Run `./bin/shark task list --all`
3. Parse stdout as JSON
4. Verify completed tasks are included

**Expected Results:**
- Exit code 0
- Valid JSON including completed tasks

**Pass/Fail:** Both T-001 and T-002 work together.

---

### TC-CROSS-05: Deprecation warning (`--show-all`) does not corrupt JSON output when `SHARK_OUTPUT=json`

**Test Type:** Integration test

**Steps:**
1. Set `SHARK_OUTPUT=json` in environment
2. Run `./bin/shark task list --show-all`
3. Capture stdout only (not stderr)
4. Attempt to parse stdout as JSON

**Expected Results:**
- Exit code 0
- Stdout is valid JSON (deprecation warning on stderr, not stdout)

**Pass/Fail:** Cobra deprecation warnings do not corrupt the JSON output stream.

---

## 8. Backward Compatibility Tests

These tests verify that no existing behavior is broken by F08 changes.

### TC-BC-001: `shark task list --json` output identical before and after

**Test Type:** Regression (snapshot comparison)

**Steps:**
1. Capture `shark task list --json` output before F08 changes (snapshot)
2. After F08 changes, capture again
3. Compare

**Expected Results:**
- Outputs are byte-identical

**Pass/Fail:** No regression in task list output.

---

### TC-BC-002: `shark task start` still works (no `--json`)

**Test Type:** Integration regression

**Steps:**
1. Run `shark task start <task-key-in-todo-status>`

**Expected Results:**
- Exit code 0
- Human-readable success message in terminal
- Task status changed to next status

**Pass/Fail:** Old lifecycle command unaffected.

---

### TC-BC-003: `shark feature create --execution-order` still works (functional, with warning)

**Test Type:** Integration regression

**Steps:**
1. Run `shark feature create E07 "Test" --execution-order=1`

**Expected Results:**
- Exit code 0
- Feature created with `execution_order = 1`
- Warning on stderr about deprecation (acceptable)

**Pass/Fail:** Old flag still functional.

---

### TC-BC-004: `shark task list --show-all` still works (functional, with warning)

**Test Type:** Integration regression

**Steps:**
1. Run `shark task list --show-all --json`

**Expected Results:**
- Exit code 0
- Valid JSON output including completed tasks
- Deprecation warning on stderr only

**Pass/Fail:** Old flag still functional.

---

### TC-BC-005: `make test` passes after all F08 changes

**Test Type:** Automated quality gate

**Steps:**
1. Run `make fmt && make lint && make test`

**Expected Results:**
- All three commands exit with code 0
- No test failures
- No lint errors
- No formatting changes

**Pass/Fail:** All pass.

---

### TC-BC-006: Exit codes unchanged for existing commands

**Test Type:** Regression

**Steps:**
1. Test that exit code 0 = success, 1 = not found, 2 = DB error, 3 = invalid state for existing commands
2. Verify `shark task start <non-existent>` still exits with code 1

**Expected Results:**
- Existing exit code behavior preserved

**Pass/Fail:** No exit code regressions.

---

## 9. UAT Scenario Mapping

This section maps E17-F08 test cases to the acceptance scenarios defined in the epic UAT plan (`uat-plan.md`).

### Journey 1: AI Agent Daily Task Workflow (J1)

| UAT Scenario | Covered By | F08 Task |
|--------------|------------|---------|
| J1-S01: Get next task with `--field key` | TC-004-12 | T-004 |
| J1-S02: Read task details with `SHARK_OUTPUT=json` | TC-002-02 | T-002 |
| J1-S04: Check status via `--field status` | TC-004-11, TC-CROSS-03 | T-004, T-002 |
| J1-S08: Structured error on invalid transition | TC-003-07, TC-CROSS-01 | T-003, T-002 |
| J1-S09: Structured error on entity not found | TC-003-07 | T-003 |

**Note:** J1-S03, J1-S05, J1-S06, J1-S07, J1-S10 relate to `status advance/set/options` commands which are in E17-F01, not F08. They are not covered here.

---

### Journey 3: Project Setup (J3)

| UAT Scenario | Covered By | F08 Task |
|--------------|------------|---------|
| J3-S01: Unified create with `--order` flag | TC-001-01 | T-001 |
| J3-S02: Task create with `--order` flag | TC-001-01 (task equivalent), TC-001-04 | T-001 |
| J3-S03: Deprecated flag still works with warning | TC-001-03, TC-001-04 | T-001 |

---

### Risk-Based Test Coverage

| UAT Risk Test ID | Covered By | Notes |
|------------------|------------|-------|
| BC-08: `--execution-order` still accepted | TC-001-03, TC-001-04, TC-001-13, TC-001-14, TC-BC-003 | Multiple test cases |
| BC-09: `--show-all` still accepted | TC-001-06, TC-001-08, TC-001-10, TC-BC-004 | Multiple test cases |
| ERR-01: JSON mode errors to stdout | TC-003-01, TC-003-07 | T-003 |
| ERR-02: Non-JSON mode errors to stderr | TC-003-02, TC-003-08 | T-003 |
| ERR-03: JSON error has required fields | TC-003-01, TC-003-03, TC-003-10 | T-003 |
| ERR-06: Deprecation warnings to stderr only | TC-001-03, TC-001-06, TC-001-12, TC-CROSS-05 | T-001 + T-002 |
| ERR-07: No deprecation warnings in JSON mode | TC-001-12, TC-CROSS-05 | T-001 + T-002 |
| EC-02: Field not found exit code 4 | TC-004-13 | T-004 |
| EC-03: JSON code distinguishes NOT_FOUND vs FIELD_NOT_FOUND | TC-004-14 | T-004 |

---

## 10. Quality Gates

All of the following must be satisfied before F08 is considered complete.

### Mandatory Quality Gate

```bash
make fmt && make lint && make test
```

Must exit 0 after each task. Run after T-002, T-001, T-003, T-004 in that order.

### Unit Test Coverage

| Area | Minimum Coverage | Test File |
|------|-----------------|-----------|
| `root.go` (SHARK_OUTPUT, field logic, error functions) | 80% | `root_test.go` |
| `cli_error.go` (CLIError struct, constants) | 90% | `root_test.go` |
| Field extraction functions | 90% | `field_output_test.go` |
| Flag normalization (task_helpers, feature) | 80% | Per-command test files |

### Behavioral Gates

| Gate | Pass Criteria |
|------|---------------|
| No stdout pollution in JSON mode | `stdout | python3 -c "import json,sys; json.load(sys.stdin)"` exits 0 for all commands |
| No stderr output in JSON mode (errors) | stderr is empty when command fails in JSON mode |
| Deprecation warnings on stderr only | Cobra's built-in behavior; verified by TC-001-03 and TC-001-12 |
| All old flags functional | TC-BC-003, TC-BC-004 pass |
| All existing tests still pass | `make test` exits 0 |
| Exit code 4 for field-not-found | TC-004-13 passes |
| CLIError bypasses field extraction | TC-004-15 passes |

### Test Execution Checklist

- [ ] TC-001-01 through TC-001-14: Flag normalization tests
- [ ] TC-002-01 through TC-002-08: SHARK_OUTPUT env var tests
- [ ] TC-003-01 through TC-003-12: Structured JSON error tests
- [ ] TC-004-01 through TC-004-18: `--field` flag tests
- [ ] TC-CROSS-01 through TC-CROSS-05: Cross-task interaction tests
- [ ] TC-BC-001 through TC-BC-006: Backward compatibility tests
- [ ] `make fmt && make lint && make test` passes

---

## Appendix A: Test Case Summary

| Task | Unit Tests | CLI Tests | Integration Tests | Total |
|------|-----------|-----------|-------------------|-------|
| T-001 | 8 | 6 | 2 | 14 |
| T-002 | 6 | 0 | 2 | 8 |
| T-003 | 9 | 0 | 3 | 12 |
| T-004 | 13 | 0 | 5 | 18 |
| Cross-task | 0 | 0 | 5 | 5 |
| Backward compat | 0 | 0 | 6 | 6 |
| **Total** | **36** | **6** | **23** | **63** |

---

## Appendix B: Implementation Order and Test Execution Order

Per the technical design, implementation order is: T-002 → T-001 → T-003 → T-004.

Run tests in the same order:

1. **After T-002:** Run TC-002-01 through TC-002-08 + `make test`
2. **After T-001:** Run TC-001-01 through TC-001-14 + TC-CROSS-04 + TC-CROSS-05 + TC-BC-003 + TC-BC-004 + `make test`
3. **After T-003:** Run TC-003-01 through TC-003-12 + TC-CROSS-01 + TC-003-08 + `make test`
4. **After T-004:** Run TC-004-01 through TC-004-18 + TC-CROSS-02 + TC-CROSS-03 + TC-004-13 + TC-004-14 + `make test`

Final pass: Run the full checklist from Section 10 including all backward compatibility tests.

---

*Last Updated: 2026-02-25*
*QA Agent: Ready for development team*
