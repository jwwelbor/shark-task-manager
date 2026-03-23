# E23-F03: Test Plan — Structured Logging Migration

**Feature**: E23-F03 Structured Logging Migration
**Date**: 2026-03-22

---

## Test Strategy

This feature is a mechanical code transformation. The primary test approach is:

1. **Static verification** (grep-based): Confirm zero legacy `log.*` / `fmt.Fprintf(os.Stderr)` calls remain
2. **Regression testing**: Existing test suite (`make test`) must pass without modification
3. **Build verification**: `make fmt && make lint && make build` must pass

No new unit tests are required for the migration itself since the calls are one-liners and slog behavior is tested by the stdlib. However, one integration test (Task 6) verifies completeness via a script.

---

## Test Cases

### TC-001: Static analysis — no legacy log calls

**Type**: Static / automated
**Command**:
```bash
count=$(grep -r "log\.Print\|log\.Fatal\|log\.Println" internal/ cmd/ --include="*.go" | grep -v "_test.go" | grep -v "//" | wc -l)
[ "$count" -eq 0 ] && echo "PASS" || echo "FAIL: $count calls remain"
```
**Expected**: PASS

### TC-002: Static analysis — no fmt.Fprintf(os.Stderr diagnostic calls

**Type**: Static / automated
**Command**:
```bash
count=$(grep -r 'fmt\.Fprintf(os\.Stderr' internal/ cmd/ --include="*.go" | grep -v "_test.go" | wc -l)
[ "$count" -eq 0 ] && echo "PASS" || echo "FAIL: $count calls remain"
```
**Expected**: PASS

### TC-003: Build passes

**Type**: Build
**Command**: `make build`
**Expected**: Exit code 0

### TC-004: Lint passes

**Type**: Static analysis
**Command**: `make fmt && make lint`
**Expected**: Exit code 0, no unused import warnings, no formatting diffs

### TC-005: Full test suite passes

**Type**: Regression
**Command**: `make test`
**Expected**: All tests pass; no new failures introduced by import changes

### TC-006: slog output is structurally valid JSON when enabled

**Type**: Integration / manual
**Steps**:
1. Set `observability.enabled: true`, `log_format: "json"`, `log_level: "debug"` in `.sharkconfig.json`
2. Trigger a warning path (e.g., an operation on a non-existent entity that hits a warn path) and capture stderr
3. Each stderr line parses as valid JSON with `level`, `msg`, `time` keys

**Expected**: Lines are valid JSON; `level` values are `"INFO"`, `"WARN"`, or `"ERROR"` (no raw format strings from old `log.Printf` calls)

---

## Regression Risk Assessment

**Low risk**: This migration only changes which logging function is called. No business logic, no data model, no API contracts change. The slog stdlib package is part of Go 1.21+ standard library — no new dependencies. Import cleanup (removing `"log"` package) is the highest-risk change; caught by `make build`.

**Mitigations**:
- Run `make test` after each task group (not just at end)
- Verify `make build` after each file group

---

## Test Execution Order

1. After Task 1 (internal/cli/): run `make build && make test`
2. After Task 2 (internal/config/): run `make build && make test`
3. After Task 3 (internal/services/): run `make build && make test`
4. After Task 4 (cmd/server/): run `make build`
5. After Task 5 (remaining packages + cmd/ utilities): run `make build && make test`
6. Task 6: run TC-001, TC-002, TC-003, TC-004, TC-005 as final gate
