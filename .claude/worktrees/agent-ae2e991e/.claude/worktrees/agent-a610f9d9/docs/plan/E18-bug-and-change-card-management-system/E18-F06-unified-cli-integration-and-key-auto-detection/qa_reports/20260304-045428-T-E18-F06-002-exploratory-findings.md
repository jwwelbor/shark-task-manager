# Exploratory Testing Findings: T-E18-F06-002 - Command Layer Dispatch Extension

**Date:** 2026-03-04
**Task:** T-E18-F06-002
**Session Duration:** ~45 minutes
**Charter:** Explore unified CLI dispatch to discover routing issues with B###, C###, and CC-### key formats

---

## Session Overview

This exploratory session focused on verifying the dispatch layer in the unified CLI commands after the Command Layer Dispatch Extension implementation. I examined source code directly and ran smoke tests to verify routing behavior.

---

## Findings

### Finding 1 (Critical): C### key error message is self-contradictory

**File:** `internal/cli/commands/helpers.go`
**Severity:** Critical (blocks AC-2)

The error message for C### keys reads:
```
invalid change card key format "C001" - expected C### (e.g., C001)
```

This message says "expected C### (e.g., C001)" while simultaneously rejecting C001. The error message was likely copy-pasted from or written alongside the CC-### handler without accounting for C### being a legitimate change entity key format. This is confusing to users and would be difficult to diagnose without reading the source code.

**Root cause:** The C-prefix error handler in `ParseScope` was written to catch invalid CC-### keys (e.g., "CABC" or "C-001") but accidentally catches valid C### keys as well because `IsChangeKey` is never checked first.

---

### Finding 2 (High): Missing scopeChange constant / case in ParseScope

**File:** `internal/cli/commands/helpers.go`
**Severity:** High

Examining the scope constants and ParseScope logic:
- `scopeBug` is defined and handled
- `scopeChangeCard` is defined and handled
- No `scopeChange` constant appears to exist for routing C### keys

The developer implemented B### and CC-### routing correctly but did not complete the C### routing. This may have been overlooked if the test coverage for `ParseScope` only covered B### and CC-### patterns.

---

### Finding 3 (Medium): Inconsistent "change" vs "change_card" naming in dispatch switches

**Files:** `get.go`, `delete_dispatch.go`, `status_group.go`, `update_dispatch.go`

Across the dispatch switches, there is inconsistency:
- `update_dispatch.go`: has `case "change", "change_card"` (handles both)
- `get.go`: only has `case "change_card"` (missing "change")
- `delete_dispatch.go`: only has `case "change_card"` (missing "change")
- `status_group.go`: only has `case "change_card"` (missing "change")

The `update_dispatch.go` implementation appears to be the most complete, while the others missed the "change" case. This inconsistency suggests the fix was applied to update but not propagated to get/delete/status.

---

### Finding 4 (Low): No automated test for dispatch with C### keys

**File:** `internal/cli/commands/` (test files)

All 30 test packages pass, but no automated test covers the scenario:
```
ParseGetArgs([]string{"C001"}) should return scopeChange type
```

The defect was only catchable via manual smoke testing. Adding test coverage for this scenario would catch similar regressions in the future.

---

### Finding 5 (Informational): runChangeGet() exists but is unreachable from runGet()

**File:** `internal/cli/commands/change.go`

`runChangeGet()` is implemented and available (line ~329) but `runGet()` in `get.go` has no `case "change"` to call it. This confirms the implementation is partially done - the handler exists, the dispatch is missing.

---

### Finding 6 (Informational): `GetChangeCardService()` and `GetBugService()` correctly follow lazy-init pattern

**File:** `internal/cli/services_global.go`

Both service accessors follow the established global accessor pattern correctly:
1. Call `GetDB()` to get the database connection
2. Create the repository from the DB
3. Call `GetWorkflowService()` for workflow validation
4. Construct and return the service

This is well-implemented and follows the pattern established by `GetTaskService()`, `GetEpicService()`, and `GetFeatureService()`.

---

### Finding 7 (Informational): CC-### routing works correctly end-to-end

The full routing chain for CC-### works:
1. `ParseScope("CC-001")` → `IsChangeCardKey` returns true → `scopeChangeCard`
2. `DetectEntityType("CC-001")` → returns "change_card"
3. `runGet` → `case "change_card"` → `runChangeCardGet`
4. `runChangeCardGet` → `GetChangeCardService()` → service returns "not found"

This is the reference implementation that C### routing should mirror.

---

## Suggestions for Developer

1. **Fix `helpers.go`**: Add `IsChangeKey` check before the C-prefix error handler. The check should be ordered: `IsChangeCardKey` first (more specific CC-### pattern), then `IsChangeKey` (C### pattern).

2. **Fix `get.go`**: Add `case "change": return runChangeGet(cmd, []string{key})` in the dispatch switch.

3. **Fix `delete_dispatch.go`**: Verify and add `case "change"` if change entities support delete.

4. **Fix `status_group.go`**: Add `case "change"` to `dispatchTransition`, `dispatchNextStatus`, and `dispatchAdvance` if change entities support status transitions.

5. **Add tests**: Write a table-driven test for `ParseGetArgs` covering B###, C###, and CC-### inputs to prevent regression.

6. **Fix error message**: Update the error handler message to clearly distinguish between invalid C### format (e.g., C-001 or CABC) vs. valid CC-### change card format.

---

## Environment

- Build: `make shark` (successful)
- Binary: `./bin/shark`
- Go: 1.23.4+
- Branch: E18-F02
- Working directory: shark-task-manager worktree
