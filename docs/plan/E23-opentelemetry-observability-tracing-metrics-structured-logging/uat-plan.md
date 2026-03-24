# E23 UAT Plan: OpenTelemetry Observability

**Date:** 2026-03-22
**Epic:** E23 - OpenTelemetry Observability (tracing, metrics, structured logging)
**Status:** Approved

---

## 1. UAT Scenarios

Each scenario is derived from the epic PRD success criteria (Section 6) and mapped to the architecture features.

### UAT-1: Structured Logging Replaces All Ad-Hoc Logging

**Source:** Epic PRD UAT-1, Success Criterion "Structured log coverage: 100%"

**Precondition:** E23 is fully merged into the codebase.

**Verification:**
1. Run a codebase search for `log.Print`, `log.Fatal`, `log.Println` across `internal/` and `cmd/` directories. Zero matches must be returned. (Note: `log/slog` imports are expected and correct.)
2. Run a codebase search for diagnostic `fmt.Fprintf(os.Stderr, ...)` calls in `internal/` (excluding `cli.Error`, `cli.Warning`, and `cli.OutputTable` error handlers). Zero matches must be returned.
3. Configure `observability.enabled: true`, `log_format: "json"`, `log_level: "debug"` in `.sharkconfig.json`.
4. Run `shark get E01 2>/tmp/shark.log`. Verify that `/tmp/shark.log` contains one or more lines, each of which is valid JSON with at minimum the fields: `time`, `level`, `msg`.
5. Verify that stdout output (the entity details) is unchanged and contains no log lines.

**Acceptance:** All five checks pass.

---

### UAT-2: CLI Command Tracing (stdout Exporter)

**Source:** Epic PRD UAT-2, Success Criterion "Trace span coverage: 100% of service methods"

**Precondition:** Observability enabled with `exporter: "stdout"`.

**Verification:**
1. Configure `.sharkconfig.json` with `observability.enabled: true`, `exporter: "stdout"`.
2. Run `shark get E01-F01-001 2>/tmp/spans.txt` (redirect stderr to file).
3. Verify `/tmp/spans.txt` contains span output including:
   - A root span named `shark.command/get` with attribute `command=get`.
   - A child span for the service method (e.g., containing `TaskService` or `EntityService` in the span name).
   - A child span for the repository query (e.g., containing `Repository.GetByKey` in the span name).
   - Each span has a non-zero duration.
4. Verify that stdout output (the task/entity details) is uncontaminated by span data.

**Acceptance:** Span hierarchy is visible in stderr; stdout is clean.

---

### UAT-3: HTTP Request Tracing

**Source:** Epic PRD UAT-3, Success Criterion "HTTP trace coverage: 100%"

**Precondition:** HTTP server running with observability enabled, OTLP or stdout exporter configured.

**Verification:**
1. Start the HTTP server with `observability.enabled: true`.
2. Send `GET /health` and verify the response includes an HTTP status header but no `traceparent` (health check may be excluded).
3. Send `GET /api/v1/tasks/E01-F01-001` (or equivalent endpoint once API is built) with header `traceparent: 00-<trace-id>-<span-id>-01`.
4. Verify the response includes `tracestate` or `traceparent` header in the response.
5. If using stdout exporter, verify stderr shows spans for: HTTP handler, service method, repository query.
6. If using OTLP exporter, verify the trace appears in the configured backend (Jaeger, Grafana Tempo).

**Acceptance:** Request-level traces are produced with correct parent-child nesting; trace context is propagated via W3C headers.

---

### UAT-4: Database Query Metrics

**Source:** Epic PRD UAT-4, Success Criterion "At least 6 core metrics exported"

**Precondition:** Observability enabled with metrics collection active. Multiple commands have been executed.

**Verification:**
1. Configure `observability.enabled: true`, `metrics_enabled: true`.
2. Execute at least 10 shark commands that involve database access (e.g., `shark get`, `shark list`, `shark create`, `shark status advance`).
3. If using stdout metric exporter, verify stderr contains metric output including `shark.db.query.duration` with labels `operation` and `table`.
4. Verify that `shark.db.query.errors` counter is present (value may be 0 if no errors occurred).
5. If using OTLP exporter, verify the metrics appear in the configured backend with the expected label dimensions.

**Acceptance:** DB query duration histogram and error counter metrics are exported with correct labels.

---

### UAT-5: Command Execution Metrics

**Source:** Epic PRD UAT-5, Success Criterion "command_executions_total counter exists"

**Precondition:** Observability enabled.

**Verification:**
1. Configure `observability.enabled: true`, `metrics_enabled: true`, `exporter: "stdout"`.
2. Run `shark get E01`, `shark list`, `shark status E01`, `shark create task E01 F01 "test"` (4 different commands).
3. Verify stderr metric output includes:
   - `shark.cli.command.invocations` counter with `command` label for each command name.
   - `shark.cli.command.duration` histogram with `command` label.
4. Verify that a command that fails (e.g., `shark get NONEXISTENT`) produces a metric with `status=error`.

**Acceptance:** Command invocation and duration metrics are recorded per command name with success/error status.

---

### UAT-6: Telemetry Disabled by Default

**Source:** Epic PRD UAT-6, Success Criterion "No-op overhead < 1%"

**Precondition:** Fresh installation or config without `observability` section.

**Verification:**
1. Remove the `observability` section from `.sharkconfig.json` (or use a fresh config).
2. Run `shark get E01`. Verify:
   - No trace spans appear on stderr.
   - No metrics output appears on stderr.
   - No structured log output appears on stderr.
   - CLI output is identical to pre-E23 behavior.
3. Verify `shark get E01 2>/tmp/stderr.txt` produces an empty or minimal stderr file (only existing pterm debug messages if `--verbose` is used, which it is not in this test).

**Acceptance:** Zero observable telemetry output when disabled; behavior matches pre-E23 baseline.

---

### UAT-7: Configuration via Environment Variables

**Source:** Epic PRD UAT-7

**Precondition:** No `observability` section in `.sharkconfig.json`.

**Verification:**
1. Set environment variables: `SHARK_OTEL_ENABLED=true`, `SHARK_LOG_LEVEL=debug`, `SHARK_LOG_FORMAT=json`.
2. Run `shark get E01 2>/tmp/env.log`.
3. Verify `/tmp/env.log` contains JSON-formatted debug-level log lines, confirming that environment variables activated observability.
4. Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` and `OTEL_SERVICE_NAME=shark-test`.
5. Verify that the OTel resource attributes in span output reflect the custom service name.

**Acceptance:** Environment variables override absent config file settings.

---

### UAT-8: Graceful Shutdown Flushes Telemetry

**Source:** Epic PRD UAT-8

**Precondition:** Observability enabled with OTLP exporter pointing to a collector.

**Verification:**
1. Configure OTLP exporter with a running collector (e.g., Jaeger all-in-one with OTLP receiver).
2. Run `shark get E01` (a fast command, ~100ms).
3. Verify that the trace appears in the collector UI within 10 seconds of command completion.
4. Run `shark list` (another fast command).
5. Verify that all spans from the command appear in the collector (none missing due to premature exit).
6. For the HTTP server: start the server, send a request, send SIGTERM, and verify the request's trace appears in the collector.

**Acceptance:** No spans or metrics are lost due to process exit timing.

---

### UAT-9: Performance Overhead Budget

**Source:** Epic PRD UAT-9, Success Criteria "< 5% enabled, < 1% disabled"

**Precondition:** A benchmark test exists for `shark get E01`.

**Verification:**
1. Run the benchmark 1000 times with `observability.enabled: false`. Record median execution time (T_disabled).
2. Run the benchmark 1000 times with `observability.enabled: true`, `exporter: "stdout"` (all spans and metrics active). Record median execution time (T_enabled).
3. Run the benchmark 1000 times with the `observability` section removed from config entirely. Record median execution time (T_noop).
4. Calculate:
   - Enabled overhead: `(T_enabled - T_disabled) / T_disabled * 100%` -- must be < 5%.
   - No-op overhead: `(T_noop - T_disabled) / T_disabled * 100%` -- must be < 1%. (Note: T_disabled and T_noop should be nearly identical; this validates the no-op path has no cost.)

**Acceptance:** Both overhead thresholds are met.

**Note:** The benchmark should use the same database and entity to control for I/O variability. Use `go test -bench` with a dedicated benchmark test, not manual timing.

---

### UAT-10: Existing Tests Pass Without Modification

**Source:** Epic PRD UAT-10, Success Criterion "100% pass rate"

**Precondition:** Full E23 implementation merged.

**Verification:**
1. Run `make test` from the project root.
2. Verify all tests pass with exit code 0.
3. Verify no test files were modified as part of E23 (observability is transparent to tests).
4. Run `make lint` to verify no linting regressions.

**Acceptance:** `make fmt && make lint && make test` all pass with zero failures and zero test file modifications.

---

## 2. Cross-Feature Integration Scenarios

These scenarios verify that multiple E23 features work correctly together.

### Integration-1: Full CLI Trace Chain

**Features involved:** F01 (Foundation), F02 (CLI Lifecycle), F04 (Service Tracing), F05 (Repository Tracing), F06 (Metrics)

**Scenario:** Execute `shark status advance E01-F01-001` with observability enabled.

**Verify:**
- A root span `shark.command/status` exists.
- Child spans show the service method call (status validation, transition).
- Grandchild spans show the repository queries (get task, update status, create history record).
- All spans are in the same trace (same trace ID).
- Command duration metric is recorded.
- DB query duration metrics are recorded for each repository call.

---

### Integration-2: Structured Logs Contain Trace Context

**Features involved:** F01 (Foundation), F03 (Logging), F04 (Service Tracing)

**Scenario:** Execute a command that produces a warning log (e.g., a command that triggers a service-level warning).

**Verify:**
- The structured log line (JSON) contains `trace_id` and `span_id` fields matching the active span context.
- This enables correlating logs with traces in observability backends.

**Note:** This requires the slog handler to extract trace context from `context.Context` and add it as log attributes. This is a standard pattern implemented in `logger.go`.

---

### Integration-3: HTTP Server with CLI-Style Tracing

**Features involved:** F01 (Foundation), F04 (Service Tracing), F05 (Repository Tracing), F07 (HTTP Instrumentation)

**Scenario:** Send an HTTP request to the API server with a `traceparent` header.

**Verify:**
- The HTTP middleware creates a span that is a child of the incoming trace context.
- Service and repository spans are children of the HTTP span.
- The full trace (HTTP -> Service -> Repository) is visible in the collector.

---

### Integration-4: Disabled Mode Does Not Affect Any Feature

**Features involved:** All features

**Scenario:** Remove the `observability` section from `.sharkconfig.json` and run every shark command category (get, list, create, status, etc.).

**Verify:**
- No panics, errors, or behavioral changes.
- No stderr output related to observability.
- All commands produce identical output to pre-E23 baseline.

---

### Integration-5: Config + Environment Variable Precedence

**Features involved:** F01 (Foundation), F02 (CLI Lifecycle)

**Scenario:** Set `.sharkconfig.json` with `observability.enabled: false` and `log_level: "warn"`. Set environment variables `SHARK_OTEL_ENABLED=true` and `SHARK_LOG_LEVEL=debug`.

**Verify:**
- Observability is enabled (environment variable overrides config file).
- Log level is debug (environment variable overrides config file).
- This confirms the precedence order: env vars > config file > defaults.

---

## 3. Performance Considerations

### Budget Constraints

| Condition | Max Overhead | Measurement |
|-----------|-------------|-------------|
| Telemetry disabled (no config section) | < 1% | No-op provider, no spans, no metrics, no structured logging |
| Telemetry enabled, stdout exporter | < 5% | Synchronous span writes to stderr, metric recording |
| Telemetry enabled, OTLP exporter | < 5% | Async batch export, 5s shutdown flush |

### What Contributes to Overhead (Enabled)

- **Span creation:** ~1-5 microseconds per span. With 3-5 spans per command (root + service + repo), this is 5-25 microseconds total. Negligible vs. 50-500ms command execution.
- **slog JSON formatting:** ~1-2 microseconds per log line. Shark produces 0-5 log lines per command. Negligible.
- **Metric recording:** ~0.5-1 microsecond per metric observation. 3-5 metrics per command. Negligible.
- **OTLP batch flush on shutdown:** 0-100ms depending on network. Happens after command output is printed, so perceived latency is zero.
- **OTel SDK initialization:** ~1-5ms first call (provider construction, exporter connection). Subsequent calls hit `sync.Once` (free).

### What Contributes to Overhead (Disabled)

- **No-op tracer `Start()` call:** ~10-50 nanoseconds (allocates empty span struct). With 3-5 calls per command, this is 30-250 nanoseconds. Immeasurable.
- **slog discard handler:** ~5-10 nanoseconds per log call (level check short-circuits). Immeasurable.
- **`sync.Once` check:** ~2-5 nanoseconds. Immeasurable.

### Benchmark Strategy

A Go benchmark test will be added in `internal/observability/benchmark_test.go`:

```go
func BenchmarkCommandWithObservability(b *testing.B) { ... }
func BenchmarkCommandWithoutObservability(b *testing.B) { ... }
func BenchmarkNoopSpanCreation(b *testing.B) { ... }
```

The benchmark gate is run as part of the CI pipeline. A regression exceeding the budget thresholds fails the build.

---

## 4. Out-of-Scope Clarifications

The following items are explicitly excluded from UAT verification for E23:

- **Collector deployment verification:** E23 produces telemetry; consuming it is the operator's responsibility.
- **Dashboard creation or alerting rules:** No Grafana dashboards or alert configs.
- **Trace propagation into `shark run` agent processes:** Agent internal operations are not traced.
- **Instrumentation of `internal/sync/`, `internal/discovery/`, `internal/fileops/`:** These packages are not instrumented in E23.
- **Modifying `--verbose` flag behavior:** `--verbose` remains independent of observability.

---

*Last Updated: 2026-03-22*
