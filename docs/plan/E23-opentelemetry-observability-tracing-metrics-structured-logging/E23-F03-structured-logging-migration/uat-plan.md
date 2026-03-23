# E23-F03: UAT Plan — Structured Logging Migration

**Feature**: E23-F03 Structured Logging Migration
**Date**: 2026-03-22

---

## UAT Objective

Verify that the structured logging migration is complete, produces no regressions in user-facing behavior, and emits correct structured log output when observability is enabled.

---

## Prerequisites

- Build: `make build` succeeds
- Tests: `make test` passes
- Binary: `./bin/shark` available

---

## UAT Scenarios

### UAT-001: Silent operation (default config)

**Goal**: Confirm no new stderr output when observability is disabled.

**Steps**:
1. Ensure no `observability` section in `.sharkconfig.json` (or `enabled: false`)
2. Run: `./bin/shark list 2>stderr.txt`
3. Run: `./bin/shark get E23-F03 2>>stderr.txt`
4. Inspect `stderr.txt`

**Expected**: `stderr.txt` is empty (no slog output with discard handler active).

---

### UAT-002: Structured JSON log output when enabled

**Goal**: Confirm slog calls produce structured JSON when observability is enabled.

**Steps**:
1. Temporarily set in `.sharkconfig.json`:
   ```json
   {
     "observability": {
       "enabled": true,
       "log_format": "json",
       "log_level": "debug"
     }
   }
   ```
2. Run: `./bin/shark list 2>slog-output.txt`
3. Inspect `slog-output.txt`

**Expected**: Each line is valid JSON with `level`, `msg`, `time`, and `service.name` fields. No unstructured `log.*` format strings visible.

---

### UAT-003: Normal CLI output unchanged

**Goal**: Confirm user-facing output (stdout) is identical pre/post migration.

**Steps**:
1. Run: `./bin/shark list` — verify normal table output on stdout
2. Run: `./bin/shark status` — verify dashboard output
3. Run: `./bin/shark get E23-F03 --json` — verify JSON output structure unchanged

**Expected**: All user-facing output identical to pre-migration behavior.

---

### UAT-004: No legacy log calls remain

**Goal**: Confirm mechanical verification that migration is complete.

**Steps**:
1. Run: `grep -r "log\.Print\|log\.Fatal\|log\.Println" internal/ cmd/ --include="*.go" | grep -v "_test.go" | grep -v "//"`
2. Run: `grep -r 'fmt\.Fprintf(os\.Stderr' internal/ cmd/ --include="*.go" | grep -v "_test.go"`

**Expected**: Both commands return zero output.

---

### UAT-005: Build and lint clean

**Steps**:
1. Run: `make fmt && make lint && make build`

**Expected**: All pass with exit code 0.

---

## UAT Sign-Off Criteria

- [ ] UAT-001 passed: no stderr output by default
- [ ] UAT-002 passed: structured JSON on stderr when enabled
- [ ] UAT-003 passed: stdout output unchanged
- [ ] UAT-004 passed: zero legacy log calls
- [ ] UAT-005 passed: build and lint clean
