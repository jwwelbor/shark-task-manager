# User Acceptance Testing Guide - E17 CLI Simplification (F06 + F08 Build)

## Feature Overview

This UAT covers the remaining E17 tasks built in the autonomous build session:

**E17-F06 (Progress Command):**
- T-E17-F06-001: `shark progress` command for feature/epic progress display
- T-E17-F06-003: Progress command test suite

**E17-F08 (JSON API for AI Agents):**
- T-E17-F08-002: `SHARK_OUTPUT=json` environment variable for session-wide JSON mode
- T-E17-F08-003: Structured JSON error output (`CLIError` struct with error codes)
- T-E17-F08-004: `--field` flag for targeted JSON field extraction

## Prerequisites

- Shark CLI built: `make shark` or `make build`
- A project with shark-tasks.db containing epics, features, and tasks
- Shell access for environment variable testing

## Test Scenarios

### Scenario 1: SHARK_OUTPUT Environment Variable (T-E17-F08-002)

**Steps:**
1. Run without env var (human output):
   ```bash
   ./bin/shark task list E17
   ```
   Expected: Human-formatted table output

2. Set env var and run same command:
   ```bash
   SHARK_OUTPUT=json ./bin/shark task list E17
   ```
   Expected: JSON output without needing `--json` flag

3. Verify `--json` flag still works:
   ```bash
   ./bin/shark task list E17 --json
   ```
   Expected: JSON output

4. Verify case sensitivity (should NOT enable JSON):
   ```bash
   SHARK_OUTPUT=JSON ./bin/shark task list E17
   ```
   Expected: Human-formatted table (only lowercase "json" is accepted)

**Success Criteria:**
- [ ] `SHARK_OUTPUT=json` enables JSON mode for any command
- [ ] Human mode is default without env var
- [ ] Case-sensitive: only `json` (lowercase) works
- [ ] `--json` flag overrides regardless of env var

### Scenario 2: Structured JSON Error Output (T-E17-F08-003)

**Steps:**
1. Trigger a not-found error in JSON mode:
   ```bash
   ./bin/shark task get NONEXISTENT-999 --json
   ```
   Expected: JSON with `{"error": true, "code": "NOT_FOUND", "message": "..."}`

2. Trigger a command error in JSON mode:
   ```bash
   ./bin/shark task start --json
   ```
   Expected: JSON with `{"error": true, "code": "COMMAND_ERROR", "message": "..."}`

3. Verify human mode errors go to stderr:
   ```bash
   ./bin/shark task get NONEXISTENT-999 2>/dev/null
   ```
   Expected: No stdout output (error on stderr only)

**Success Criteria:**
- [ ] JSON errors have `error`, `code`, and `message` fields
- [ ] Error codes are SCREAMING_SNAKE_CASE strings
- [ ] `error` field is always `true`
- [ ] Optional fields (`entity`, `entity_key`) omitted when empty
- [ ] Human mode errors go to stderr, not stdout

### Scenario 3: --field Flag for Targeted Extraction (T-E17-F08-004)

**Steps:**
1. Extract a single field from an entity:
   ```bash
   ./bin/shark task get T-E17-F08-004 --field status
   ```
   Expected: Just the status value (e.g., `ready_for_approval`), no JSON wrapping

2. Extract field from a list (one value per line):
   ```bash
   ./bin/shark task list E17 --field status
   ```
   Expected: One status per line for each task

3. Extract nested field with dot notation:
   ```bash
   ./bin/shark feature get E17-F08 --field progress.weighted_pct
   ```
   Expected: Just the numeric value

4. Request non-existent field:
   ```bash
   ./bin/shark task get T-E17-F08-004 --field nonexistent
   echo "Exit code: $?"
   ```
   Expected: Error output, exit code 4

5. Verify --field implies JSON mode:
   ```bash
   ./bin/shark task get T-E17-F08-004 --field key
   ```
   Expected: Just the key value (no table formatting, no --json needed)

6. Verify CLIError bypasses field extraction:
   ```bash
   ./bin/shark task get NONEXISTENT --field status
   echo "Exit code: $?"
   ```
   Expected: Full JSON error object (not just the `status` field from the error), exit code 4 or 1

**Success Criteria:**
- [ ] `--field` extracts bare values (strings without quotes)
- [ ] Arrays produce one value per line
- [ ] Dot notation works for nested fields
- [ ] Missing field returns exit code 4
- [ ] `--field` implies `--json` mode automatically
- [ ] Error objects bypass field extraction

### Scenario 4: Progress Command (T-E17-F06-001, T-E17-F06-003)

**Steps:**
1. Get feature progress:
   ```bash
   ./bin/shark progress E17-F08
   ```
   Expected: Human-readable progress display with weighted/completion percentages

2. Get epic progress:
   ```bash
   ./bin/shark progress E17
   ```
   Expected: Epic-level progress with feature rollup

3. JSON output:
   ```bash
   ./bin/shark progress E17-F08 --json
   ```
   Expected: JSON with progress metrics

4. Combined with --field:
   ```bash
   ./bin/shark progress E17-F08 --field weighted_pct
   ```
   Expected: Just the weighted percentage number

**Success Criteria:**
- [ ] Feature progress shows weighted and completion percentages
- [ ] Epic progress shows feature-level rollup
- [ ] JSON output includes structured progress data
- [ ] `--field` works with progress output

## Rollback Plan

If issues found:
- All changes are on branch E17, not merged to main
- Revert specific files if needed:
  - `internal/cli/cli_error.go` - CLIError struct
  - `internal/cli/field.go` - Field extraction
  - `internal/cli/root.go` - Config changes (Field, SHARK_OUTPUT, OutputJSON)
  - `cmd/shark/main.go` - Error handling changes
  - `internal/cli/commands/progress.go` - Progress command

## Sign-off

- [ ] All scenarios pass
- [ ] No critical bugs
- [ ] Error codes are consistent and parseable
- [ ] AI agent workflow (SHARK_OUTPUT=json + --field) works end-to-end
- [ ] Ready for approval
