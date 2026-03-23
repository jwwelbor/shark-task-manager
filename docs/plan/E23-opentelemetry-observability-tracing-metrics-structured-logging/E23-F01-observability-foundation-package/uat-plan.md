# E23-F01: Observability Foundation Package — UAT Plan

**Feature Key:** E23-F01-observability-foundation-package
**Date:** 2026-03-22
**Status:** Draft

---

## Overview

This UAT plan defines specific, executable test scenarios for verifying the Observability Foundation Package. Each scenario includes preconditions, exact steps, expected outcomes, and pass/fail criteria. Scenarios are grouped by testing concern.

UAT does not require an OTLP collector for most scenarios. Scenarios that involve OTLP are marked and provide a Docker-based collector setup for those who wish to run them.

---

## Setup

### Prerequisites

- Go 1.23.4 or later
- The project is built: `make build`
- A clean working directory (no leftover test configs)
- Baseline `shark` binary built from main before F01 is merged (for performance comparison)

### Environment Reset Between Scenarios

After any scenario that modifies global OTel state or environment variables:

```bash
unset SHARK_OTEL_ENABLED
unset OTEL_EXPORTER_OTLP_ENDPOINT
unset OTEL_EXPORTER_OTLP_PROTOCOL
unset OTEL_SERVICE_NAME
unset SHARK_LOG_LEVEL
unset SHARK_LOG_FORMAT
```

---

## Group 1: Default Behavior (Disabled Path)

These scenarios confirm that nothing changes for users who do not opt in to observability.

### UAT-01: Existing config without observability key loads cleanly

**Preconditions:**
- `.sharkconfig.json` does not contain an `"observability"` key
- `SHARK_OTEL_ENABLED` is not set

**Steps:**
1. Run `./bin/shark config show`
2. Inspect output for any observability-related fields or errors

**Expected Outcome:**
- Command completes with exit code 0
- No error mentioning observability
- No additional output on stderr related to OTel initialization

**Pass Criteria:** Exit code 0, no observability-related errors in output.

---

### UAT-02: Disabled observability produces zero stdout contamination

**Preconditions:**
- No `"observability"` key in `.sharkconfig.json`
- `SHARK_OTEL_ENABLED` is not set

**Steps:**
1. Run `./bin/shark get E01 --json 2>/dev/null`
2. Pipe output to `jq .` to validate it is valid JSON

**Expected Outcome:**
- `jq .` exits 0
- Output is valid JSON representing the epic

**Command:**
```bash
./bin/shark get E01 --json 2>/dev/null | jq . >/dev/null && echo "PASS: stdout is clean JSON"
```

**Pass Criteria:** Command chain exits 0 and prints "PASS: stdout is clean JSON".

---

### UAT-03: Disabled observability produces no stderr output

**Preconditions:**
- No `"observability"` key in `.sharkconfig.json`
- `SHARK_OTEL_ENABLED` is not set

**Steps:**
1. Run `./bin/shark get E01 --json 2>/tmp/shark-stderr.txt`
2. Check stderr capture file

**Expected Outcome:**
- `/tmp/shark-stderr.txt` is empty (zero bytes)

**Command:**
```bash
./bin/shark get E01 --json 2>/tmp/shark-stderr.txt
if [ -s /tmp/shark-stderr.txt ]; then
    echo "FAIL: unexpected stderr output:"
    cat /tmp/shark-stderr.txt
else
    echo "PASS: no stderr output"
fi
```

**Pass Criteria:** Prints "PASS: no stderr output".

---

### UAT-04: Unit test suite passes unchanged

**Preconditions:** F01 is merged to the development branch.

**Steps:**
1. Run `make test`

**Expected Outcome:**
- All tests pass
- No new test failures introduced

**Command:**
```bash
make test && echo "PASS: all tests pass"
```

**Pass Criteria:** `make test` exits 0.

---

## Group 2: Config Loading

These scenarios verify that the `ObservabilityConfig` struct is correctly parsed from `.sharkconfig.json`.

### UAT-05: Config with enabled=false loads without error

**Preconditions:** A `.sharkconfig.json` exists in the project root.

**Steps:**
1. Add the following to `.sharkconfig.json`:
   ```json
   "observability": {
     "enabled": false
   }
   ```
2. Run `./bin/shark config show`

**Expected Outcome:**
- Command exits 0
- No error output

**Pass Criteria:** Exit code 0.

---

### UAT-06: Config with all observability fields parses correctly

**Steps:**
1. Set `.sharkconfig.json` `observability` block to:
   ```json
   "observability": {
     "enabled": true,
     "tracing_enabled": true,
     "metrics_enabled": true,
     "log_level": "debug",
     "log_format": "text",
     "exporter": "stdout",
     "otlp_endpoint": "localhost:4317",
     "otlp_protocol": "grpc",
     "service_name": "test-shark"
   }
   ```
2. Write a small Go test that calls `config.Load(configPath)` and asserts each field.

**Expected Outcome (via test):**
```go
assert.True(t, cfg.Observability.Enabled)
assert.True(t, cfg.Observability.TracingEnabled)
assert.Equal(t, "debug", cfg.Observability.LogLevel)
assert.Equal(t, "text", cfg.Observability.LogFormat)
assert.Equal(t, "stdout", cfg.Observability.Exporter)
assert.Equal(t, "localhost:4317", cfg.Observability.OTLPEndpoint)
assert.Equal(t, "grpc", cfg.Observability.OTLPProtocol)
assert.Equal(t, "test-shark", cfg.Observability.ServiceName)
```

**Pass Criteria:** All assertions pass.

---

### UAT-07: Unknown observability fields do not cause errors

**Steps:**
1. Add to `.sharkconfig.json`:
   ```json
   "observability": {
     "enabled": false,
     "future_field": "ignored"
   }
   ```
2. Run `./bin/shark config show`

**Expected Outcome:**
- Command exits 0
- No error about unknown field

**Pass Criteria:** Exit code 0.

---

### UAT-08: Config without observability produces zero-value struct

**Steps:**
1. Ensure `.sharkconfig.json` has no `"observability"` key.
2. Run the config package test `TestConfigLoad_ObservabilityAbsent`.

**Expected Outcome:**
- `Config.Observability.Enabled == false`
- All other `ObservabilityConfig` fields are zero values

**Pass Criteria:** Test passes.

---

## Group 3: Provider Initialization

These scenarios verify `InitProvider` behavior across different configurations.

### UAT-09: InitProvider disabled path returns no-op shutdown

**Steps:**
1. Run `go test -v -run TestInitProvider_DisabledReturnsNoop ./internal/observability/`

**Expected Outcome:**
- Test passes
- Log output shows "PASS"

**Pass Criteria:** `go test` exits 0.

---

### UAT-10: InitProvider stdout exporter produces stderr output, not stdout

**Steps:**
1. Write a test binary (or use the test file) that:
   - Sets `cfg.Enabled = true`, `cfg.Exporter = "stdout"`, `cfg.ServiceName = "uat-test"`
   - Calls `InitProvider(cfg)`
   - Creates a tracer: `tracer := otel.Tracer("uat")`
   - Starts and ends a span: `ctx, span := tracer.Start(ctx, "uat-span"); span.End()`
   - Calls the returned ShutdownFunc
2. Run the binary capturing both stdout and stderr separately:
   ```bash
   go test -v -run TestInitProvider_StdoutNotContaminated ./internal/observability/ \
       1>/tmp/uat-stdout.txt 2>/tmp/uat-stderr.txt
   ```
3. Inspect captured output:
   ```bash
   echo "--- stdout ---"
   cat /tmp/uat-stdout.txt
   echo "--- stderr ---"
   cat /tmp/uat-stderr.txt
   ```

**Expected Outcome:**
- Stdout contains only Go test output (`--- PASS`, `ok`, etc.)
- Stderr may contain OTel span data in stdout format (since the test redirects stderr)
- No OTel span data appears in stdout

**Pass Criteria:** `TestInitProvider_StdoutNotContaminated` passes.

---

### UAT-11: InitProvider OTLP exporter does not fail without collector

**Steps:**
1. Ensure no OTel collector is running on `localhost:4317`
2. Run `go test -v -run TestInitProvider_OTLPExporter_LazyConnect ./internal/observability/`

**Expected Outcome:**
- `InitProvider` returns no error
- `ShutdownFunc` returns without panicking (may return a timeout error if flush is attempted, which is acceptable)

**Pass Criteria:** Test passes (InitProvider does not error at init time).

---

### UAT-12: OTel global state is clean after ShutdownFunc is called

**Steps:**
1. Run `go test -count=2 -race -v ./internal/observability/`

**Expected Outcome:**
- All tests pass on both runs
- No data race detector warnings

**Pass Criteria:** `go test -count=2 -race` exits 0.

---

## Group 4: Logger Initialization

### UAT-13: JSON format produces valid JSON log lines

**Steps:**
1. Run `go test -v -run TestInitLogger_JSONFormat ./internal/observability/`

**Expected Outcome:**
- Test passes
- Captured stderr contains a line that is valid JSON with a `"msg"` key

**Pass Criteria:** Test passes.

---

### UAT-14: Text format produces human-readable output

**Steps:**
1. Run `go test -v -run TestInitLogger_TextFormat ./internal/observability/`

**Expected Outcome:**
- Test passes
- Captured stderr contains a line with `msg=` key-value text format (not JSON braces)

**Pass Criteria:** Test passes.

---

### UAT-15: Log level filtering works correctly

**Steps:**
1. Run `go test -v -run TestInitLogger_LevelFiltering ./internal/observability/`

**Expected Outcome:**
- With `LogLevel = "warn"`, `slog.Info()` produces no output
- With `LogLevel = "warn"`, `slog.Warn()` produces output

**Pass Criteria:** Test passes.

---

### UAT-16: Logger disabled path is truly silent

**Preconditions:** Default config (no observability key).

**Steps:**
1. Call `InitLogger` with `cfg.Enabled = false`
2. Call `slog.Info("message")`, `slog.Warn("warning")`, `slog.Debug("debug")`
3. Assert stderr has zero bytes

**Command:**
```bash
go test -v -run TestInitLogger_DisabledDiscards ./internal/observability/
```

**Pass Criteria:** Test passes with zero bytes captured on stderr.

---

### UAT-17: service.name attribute appears in every log line

**Steps:**
1. Run `go test -v -run TestInitLogger_ServiceNameAttribute ./internal/observability/`

**Expected Outcome:**
- Captured JSON log line contains `"service.name": "myservice"` (or whatever name was set)

**Pass Criteria:** Test passes.

---

### UAT-18: Default service name is "shark-task-manager"

**Steps:**
1. Run `go test -v -run TestInitLogger_DefaultServiceName ./internal/observability/`

**Expected Outcome:**
- When `cfg.ServiceName = ""`, output JSON contains `"service.name": "shark-task-manager"`

**Pass Criteria:** Test passes.

---

## Group 5: No-Op Provider

### UAT-19: NoopProvider does not panic on tracer calls

**Steps:**
1. Run `go test -v -run TestNoopProvider_GlobalTracerDoesNotPanic ./internal/observability/`

**Expected Outcome:**
- No panic
- `otel.Tracer("test").Start(ctx, "span")` returns a valid no-op span

**Pass Criteria:** Test passes.

---

### UAT-20: NoopProvider is idempotent

**Steps:**
1. Run `go test -v -run TestNoopProvider_IdempotentCallsSafe ./internal/observability/`

**Expected Outcome:**
- Calling `NoopProvider()` three times in succession does not panic

**Pass Criteria:** Test passes.

---

## Group 6: Metric Instruments

### UAT-21: CommandMetrics records without panicking

**Steps:**
1. Run `go test -v -run TestCommandMetrics ./internal/observability/`

**Expected Outcome:**
- `NewCommandMetrics` succeeds with a no-op meter
- `RecordDuration` with nil error sets `status=ok`
- `RecordDuration` with non-nil error sets `status=error`
- No panics

**Pass Criteria:** All `TestCommandMetrics_*` tests pass.

---

### UAT-22: DBMetrics records without panicking

**Steps:**
1. Run `go test -v -run TestDBMetrics ./internal/observability/`

**Expected Outcome:**
- `NewDBMetrics` succeeds with a no-op meter
- `RecordQueryDuration` calls succeed without panic

**Pass Criteria:** All `TestDBMetrics_*` tests pass.

---

## Group 7: Environment Variable Overrides

### UAT-23: SHARK_OTEL_ENABLED overrides config file setting

**Preconditions:**
- `.sharkconfig.json` has `"observability": {"enabled": false}`

**Steps:**
1. `export SHARK_OTEL_ENABLED=true`
2. Run `go test -v -run TestApplyEnvOverrides_Enabled ./internal/observability/`

**Expected Outcome:**
- After applying env overrides, `cfg.Enabled == true`

**Pass Criteria:** Test passes.

**Cleanup:** `unset SHARK_OTEL_ENABLED`

---

### UAT-24: OTEL_EXPORTER_OTLP_ENDPOINT overrides config

**Steps:**
1. `export OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317`
2. Run `go test -v -run TestApplyEnvOverrides_OTLPEndpoint ./internal/observability/`

**Expected Outcome:**
- `cfg.OTLPEndpoint == "collector:4317"`

**Pass Criteria:** Test passes.

**Cleanup:** `unset OTEL_EXPORTER_OTLP_ENDPOINT`

---

### UAT-25: OTEL_SERVICE_NAME overrides config

**Steps:**
1. `export OTEL_SERVICE_NAME=my-custom-service`
2. Run `go test -v -run TestApplyEnvOverrides_ServiceName ./internal/observability/`

**Expected Outcome:**
- `cfg.ServiceName == "my-custom-service"`

**Pass Criteria:** Test passes.

**Cleanup:** `unset OTEL_SERVICE_NAME`

---

### UAT-26: SHARK_LOG_LEVEL overrides config

**Steps:**
1. `export SHARK_LOG_LEVEL=debug`
2. Run `go test -v -run TestApplyEnvOverrides_LogLevel ./internal/observability/`

**Expected Outcome:**
- `cfg.LogLevel == "debug"`

**Pass Criteria:** Test passes.

**Cleanup:** `unset SHARK_LOG_LEVEL`

---

### UAT-27: Unset env vars do not override config values

**Steps:**
1. Ensure all `SHARK_*` and `OTEL_*` env vars are unset
2. Set `cfg.ServiceName = "from-config"`, `cfg.LogLevel = "warn"`
3. Call `applyEnvOverrides(&cfg)` (via test)
4. Assert values are unchanged

**Expected Outcome:**
- `cfg.ServiceName` remains `"from-config"`
- `cfg.LogLevel` remains `"warn"`

**Pass Criteria:** Test passes.

---

## Group 8: No Regression to Existing Commands

### UAT-28: All existing tests pass after F01 merge

**Steps:**
1. Checkout the F01 branch
2. Run `make test`

**Expected Outcome:**
- All tests pass
- Zero new test failures

**Command:**
```bash
make test && echo "PASS: full test suite clean"
```

**Pass Criteria:** `make test` exits 0.

---

### UAT-29: shark task commands work normally after F01

**Steps:**
1. Run these commands against the development database:
   ```bash
   ./bin/shark list
   ./bin/shark get E23 --json | jq .key
   ./bin/shark task list --json | jq length
   ```

**Expected Outcome:**
- All commands exit 0
- JSON output is valid and contains expected fields
- No new warnings or errors in output

**Pass Criteria:** All three commands exit 0 and produce valid output.

---

### UAT-30: Performance within 1% of baseline (disabled path)

**Preconditions:**
- Baseline binary built from main before F01 (saved as `./bin/shark-baseline`)
- F01 binary built as `./bin/shark`

**Steps:**
1. Run benchmark on baseline:
   ```bash
   hyperfine --warmup 3 './bin/shark-baseline get E01 --json 2>/dev/null'
   ```
2. Run benchmark on F01:
   ```bash
   hyperfine --warmup 3 './bin/shark get E01 --json 2>/dev/null'
   ```
3. Compare mean times

**Alternative (if hyperfine is not installed):**
```bash
for i in $(seq 1 10); do
    time ./bin/shark get E01 --json 2>/dev/null
done
```

**Expected Outcome:**
- Mean execution time of F01 binary is within 1% of baseline

**Pass Criteria:** Performance delta is within 1%.

---

## Group 9: Optional — OTLP Collector Integration

These scenarios require a running OTLP collector. They are optional for initial UAT but should be run before the epic is closed.

### Setup

```bash
docker run -d --name otel-collector \
    -p 4317:4317 \
    otel/opentelemetry-collector:latest
```

### UAT-31: OTLP exporter sends spans to a running collector

**Preconditions:**
- OTLP collector running on `localhost:4317`
- `.sharkconfig.json` with:
  ```json
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_protocol": "grpc",
    "service_name": "uat-test"
  }
  ```

**Steps:**
1. Run `./bin/shark get E01 --json 2>/tmp/uat-otlp.log`
2. Check collector logs:
   ```bash
   docker logs otel-collector 2>&1 | grep "uat-test" | head -5
   ```

**Expected Outcome:**
- Command exits 0
- Collector logs show at least one span received from `uat-test`

**Pass Criteria:** Collector receives at least one span with `service.name = "uat-test"`.

---

### UAT-32: Shutdown flushes all pending spans

**Steps:**
1. With OTLP collector running, run 5 shark commands in rapid succession
2. Check collector received spans for all 5

**Pass Criteria:** Collector log shows spans from all 5 command invocations within a few seconds of the last command completing.

---

## Sign-Off Checklist

Before F01 is marked complete, the following must be confirmed:

- [ ] UAT-01 through UAT-04 pass (default disabled behavior unchanged)
- [ ] UAT-05 through UAT-08 pass (config loading correct)
- [ ] UAT-09 through UAT-12 pass (provider initialization)
- [ ] UAT-13 through UAT-18 pass (logger initialization)
- [ ] UAT-19 through UAT-20 pass (no-op provider)
- [ ] UAT-21 through UAT-22 pass (metric instruments)
- [ ] UAT-23 through UAT-27 pass (environment variable overrides)
- [ ] UAT-28 through UAT-30 pass (no regression)
- [ ] `make fmt && make lint && make test` all exit 0
- [ ] Binary size increase measured and documented (target: under 15MB)
- [ ] Reviewer has confirmed no stdout contamination manually

Optional (should be run before epic close):
- [ ] UAT-31 and UAT-32 pass (OTLP integration with real collector)

---

*Last Updated: 2026-03-22*
