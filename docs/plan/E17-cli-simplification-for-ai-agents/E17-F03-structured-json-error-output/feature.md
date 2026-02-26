---
feature_key: E17-F03
epic_key: E17
title: Structured JSON Error Output
description: When JSON mode is active, output all errors as structured JSON to stdout with error codes, entity context, and valid transitions, eliminating the need for defensive 2>/dev/null error suppression.
execution_order: 3
phase: 1
complexity: S
status: draft
dependencies: []
depended_on_by:
  - E17-F04 (env var triggers JSON error mode)
epic_requirements:
  - F03 (Structured JSON Error Output)
  - NFR-1 (Backward Compatibility)
  - NFR-3 (Service Layer Integration)
  - NFR-4 (Testing)
---

# Structured JSON Error Output

**Feature Key**: E17-F03
**Phase**: 1 (Must-Have)
**Complexity**: S (M per tech review -- new error type + mapping layer)
**Execution Order**: 3 (foundational for F02 and F07; creates error infrastructure)

---

## Scope

### Problem

36% of all AI agent CLI invocations use defensive error suppression (`2>/dev/null` or `2>&1 ||`) because error output is unpredictable. Errors are human-readable text sent to stderr, which agents cannot parse programmatically. When a status transition fails, agents have no machine-readable way to know what the current status is or what transitions are valid.

### Solution

When `--json` or `SHARK_OUTPUT=json` is active, output all errors as structured JSON to stdout with consistent error codes, the entity key being operated on, the current status for status-related errors, and an array of valid transitions for transition errors.

### What This Feature Does

- Defines a `StructuredError` type with JSON-serializable fields: `error`, `code`, `message`, `entity`, `current_status`, `valid_transitions`, `details`
- Creates an error mapping layer that translates existing typed errors (`NotFoundError`, `BackwardReasonError`, validation errors) into `StructuredError` instances with appropriate error codes
- Intercepts errors in Cobra command execution when JSON mode is active and outputs structured JSON to stdout instead of stderr
- Preserves human-readable error formatting when JSON mode is NOT active
- Defines consistent exit codes: 0 (success), 1 (not found), 2 (system error), 3 (invalid state), 4 (conflict)

### What This Feature Does NOT Do

- Does not change error behavior when `--json` is not active (human-readable stderr preserved)
- Does not create new error types in the service layer (uses existing typed errors)
- Does not change success output format

---

## Acceptance Criteria

- [ ] Error output is valid JSON: `{"error": true, "code": "INVALID_TRANSITION", "message": "...", ...}`
- [ ] Error JSON includes `entity` field (the ID operated on)
- [ ] Error JSON includes `current_status` for status-related errors
- [ ] Error JSON includes `valid_transitions` array for transition errors
- [ ] Errors go to stdout (not stderr) when JSON mode is active
- [ ] Exit codes are consistent:
  - 0: Success
  - 1: Not found
  - 2: System error (DB, IO)
  - 3: Invalid state (workflow violation, validation)
  - 4: Conflict (duplicate key)
- [ ] Human-readable errors still go to stderr when NOT in JSON mode (unchanged behavior)
- [ ] All existing tests pass without modification (`make test` green)

### Error Codes

| Code | Meaning | Exit Code |
|------|---------|-----------|
| `NOT_FOUND` | Entity does not exist | 1 |
| `INVALID_TRANSITION` | Workflow does not allow this transition | 3 |
| `INVALID_STATUS` | Status value is not in workflow config | 3 |
| `VALIDATION_ERROR` | Input validation failure | 3 |
| `CONFLICT` | Duplicate key or concurrent modification | 4 |
| `SYSTEM_ERROR` | Database or IO failure | 2 |
| `FIELD_NOT_FOUND` | Requested field does not exist on entity (for --field) | 1 |

---

## Dependencies

### Depends On

None. This is standalone error infrastructure.

### Depended On By

- **E17-F04**: When `SHARK_OUTPUT=json` activates JSON mode, structured errors should be active too.
- **E17-F02**: The `--field` flag uses structured errors for field-not-found reporting.

---

## Implementation Notes

- Create `StructuredError` struct in `internal/cli/commands/errors.go` or a new `internal/cli/json_errors.go`
- Map existing typed errors:
  - `repository.NotFoundError` -> `NOT_FOUND`
  - `services.BackwardReasonError` -> `INVALID_TRANSITION`
  - `services.ErrReasonRequired` -> `VALIDATION_ERROR`
  - Generic `error` -> `SYSTEM_ERROR`
- Intercept errors in Cobra's `PersistentPostRunE` or in a wrapper around `RunE` handlers
- When `GlobalConfig.JSON` is true, serialize error to JSON and write to stdout
- Exit code determination based on error code mapping
- The existing `InvalidStatusTransitionError()` in `errors.go` already includes allowed transitions as text -- restructure this as JSON

---

## Success Metrics

- **Primary**: Defensive error suppression reduced from 36% to less than 5% of agent commands
- **Measured by**: Count of `2>/dev/null` and `2>&1 ||` patterns in agent logs
- **Backward Compatibility**: 100% -- human-readable mode unchanged

---

## UAT Scenarios

- J1-S08: Structured error on invalid transition
- J1-S09: Structured error on entity not found
- ERR-01 through ERR-07: Complete structured error test suite
- EC-01 through EC-04: Exit code validation

---

*Last Updated*: 2026-02-25
