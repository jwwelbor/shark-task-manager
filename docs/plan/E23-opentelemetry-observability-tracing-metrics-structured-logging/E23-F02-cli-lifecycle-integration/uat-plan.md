# E23-F02 UAT Plan: CLI Lifecycle Integration

**Feature Key**: E23-F02
**Date**: 2026-03-22
**Tester**: Product Owner / QA

---

## Prerequisites

- F01 (Observability Foundation Package) merged and passing tests
- `make build` succeeds
- `make test` passes
- `.sharkconfig.json` is present in project root

---

## UAT Scenarios

### UAT-01: Default Config — No Observability Overhead

**Goal**: Verify the default (disabled) path adds no visible overhead and no output.

**Steps**:
1. Ensure `.sharkconfig.json` has no `observability` key (or `"enabled": false`)
2. Run: `time ./bin/shark get E01 --json > /tmp/out.txt 2>/tmp/err.txt`
3. Inspect `/tmp/out.txt` — must contain valid JSON
4. Inspect `/tmp/err.txt` — must be empty (zero bytes)
5. Repeat 5 times and verify wall-clock time is within 5% of pre-E23 baseline

**Pass Criteria**:
- [ ] `/tmp/out.txt` contains valid JSON task output
- [ ] `/tmp/err.txt` is empty (no observability output)
- [ ] Wall-clock time within 5% of baseline

---

### UAT-02: Observability Enabled — stdout Exporter

**Goal**: Verify that when enabled, slog and span output appears on stderr only.

**Steps**:
1. Add to `.sharkconfig.json`:
   ```json
   "observability": {
     "enabled": true,
     "exporter": "stdout",
     "log_level": "info",
     "log_format": "json"
   }
   ```
2. Run: `./bin/shark get E01 --json > /tmp/out.txt 2>/tmp/err.txt`
3. Inspect `/tmp/out.txt` — must contain valid JSON (no slog/span contamination)
4. Inspect `/tmp/err.txt` — must contain slog JSON lines and OTel span JSON

**Pass Criteria**:
- [ ] `/tmp/out.txt` is valid JSON matching the entity structure
- [ ] `/tmp/err.txt` is non-empty and contains `"service.name"` in slog lines
- [ ] No OTel span data appears in `/tmp/out.txt`

---

### UAT-03: Span Shutdown Before DB Close

**Goal**: Verify teardown ordering: observability shuts down before DB closes.

**Steps**:
1. Enable observability with `exporter=stdout` as in UAT-02
2. Run: `./bin/shark list --json 2>/tmp/err.txt`
3. Inspect `/tmp/err.txt` for span output appearing before any DB close warnings

**Pass Criteria**:
- [ ] Command completes without error
- [ ] If `--verbose` is added, no "failed to close database" errors appear after spans

---

### UAT-04: OTLP Failure Degrades Gracefully

**Goal**: Verify that an unreachable OTLP endpoint does not abort the command.

**Steps**:
1. Add to `.sharkconfig.json`:
   ```json
   "observability": {
     "enabled": true,
     "exporter": "otlp",
     "otlp_endpoint": "localhost:9999"
   }
   ```
2. Run: `./bin/shark get E01 --json > /tmp/out.txt 2>/tmp/err.txt`
3. Check exit code: `echo $?` — must be 0
4. Inspect `/tmp/out.txt` — must contain valid JSON
5. Inspect `/tmp/err.txt` — may contain a warning about OTLP connection failure

**Pass Criteria**:
- [ ] Exit code is 0 (command succeeded)
- [ ] `/tmp/out.txt` contains valid JSON
- [ ] No panic or fatal error

---

### UAT-05: Test Suite Passes with Reset

**Goal**: Verify test suite passes with the new ResetObservability integration.

**Steps**:
1. Run: `make test`
2. Verify all tests pass
3. Run: `make lint`
4. Verify no lint errors

**Pass Criteria**:
- [ ] `make test` exits with code 0
- [ ] `make lint` exits with code 0
- [ ] No test failures related to OTel global state leaking between tests

---

### UAT-06: Config Not Present — Zero Error

**Goal**: Verify that missing `.sharkconfig.json` does not cause observability panic.

**Steps**:
1. Run from a directory with no `.sharkconfig.json` ancestor:
   `./bin/shark --help`

**Pass Criteria**:
- [ ] Help output displays normally
- [ ] No panic
- [ ] No observability errors printed

---

### UAT-07: Verbose Mode Shows Shutdown Warning (If Any)

**Goal**: Verify that `--verbose` surfaces observability warnings.

**Steps**:
1. Configure `observability.enabled=true` with `exporter=otlp` and bad endpoint
2. Run: `./bin/shark get E01 --verbose --json > /tmp/out.txt 2>/tmp/err.txt`
3. Inspect `/tmp/err.txt`

**Pass Criteria**:
- [ ] Warning about observability init or shutdown appears in stderr when `--verbose`
- [ ] Command still exits with code 0

---

## Regression Checklist

The following must remain unchanged after E23-F02 merges:

- [ ] `shark get E01 --json` stdout output is identical to pre-E23 output
- [ ] `shark list --json` stdout output is identical to pre-E23 output
- [ ] `shark status advance E01-F01-001` behaves identically
- [ ] `shark task create E01 F01 "Test Task"` behaves identically
- [ ] All existing tests pass: `make test`
- [ ] `make lint` passes

---

## Sign-off

| Scenario | Status | Tester | Date |
|----------|--------|--------|------|
| UAT-01 | Pending | | |
| UAT-02 | Pending | | |
| UAT-03 | Pending | | |
| UAT-04 | Pending | | |
| UAT-05 | Pending | | |
| UAT-06 | Pending | | |
| UAT-07 | Pending | | |
| Regression | Pending | | |

---

*Last Updated: 2026-03-22*
