---
epic_key: E23
title: OpenTelemetry Observability (tracing, metrics, structured logging)
description: Add production-grade observability to Shark Task Manager by replacing ad-hoc log.Println/fmt.Printf logging with structured JSON logging via Go's log/slog, instrumenting CLI commands, HTTP handlers, service methods, and database operations with OpenTelemetry distributed tracing spans, and collecting key operational metrics (command execution counts, durations, error rates, DB query latency). All telemetry is configurable with pluggable exporters -- stdout/console for development, OTLP for production collectors like Grafana, Jaeger, or Datadog.
---

# OpenTelemetry Observability (tracing, metrics, structured logging)

**Epic Key**: E23

---

## 1. Problem Statement and Business Justification

### Problem

Shark Task Manager currently has no structured observability. Logging across the codebase is a mix of `log.Println`, `log.Printf`, `log.Fatal`, and direct `fmt.Printf`/`fmt.Fprintf` calls scattered across 21+ source files (91 occurrences of `log.*` calls and 194+ occurrences of `fmt.Print*` in `internal/`). These calls produce unstructured plaintext output with no consistent format, no severity levels, no contextual fields (entity key, operation name, duration), and no way to correlate related operations.

There is zero tracing instrumentation. When a CLI command executes -- flowing from command handler through service layer to repository to SQLite and back -- there is no way to see the full call chain, identify which layer is slow, or understand the sequence of operations that led to an error. The HTTP API server (`cmd/server/main.go`) has no request tracing, no request-id propagation, and no middleware for observability.

There are no operational metrics. There is no data on which commands are used most frequently, how long database queries take on average, what the error rate is for status transitions, or whether Turso cloud database latency differs from local SQLite. Operators and developers are blind to system behavior in production.

This lack of observability has concrete consequences:
- Debugging production issues requires reproducing them locally with `--verbose` flag, which only toggles a boolean and does not produce structured diagnostic output.
- Performance regressions in database queries or service logic go undetected until users report slowness.
- The HTTP API (under development in `cmd/server/`) cannot be operated in production without request-level telemetry.
- AI agent orchestration via `shark run` produces run logs, but the internal shark operations during those runs are opaque.

### Business Justification

Shark is evolving from a local development tool into a team-facing system with cloud database support (Turso), an HTTP API server, and AI agent orchestration (`shark run`). Each of these transitions increases the operational surface area and makes observability a prerequisite rather than a nice-to-have.

OpenTelemetry is the industry-standard, vendor-neutral observability framework adopted by CNCF and supported by every major APM vendor (Datadog, Grafana, New Relic, Honeycomb, AWS X-Ray). Investing in OTel now establishes the telemetry foundation that all future features (HTTP API expansion, multi-user support, webhook integrations) will build on, rather than each feature implementing ad-hoc instrumentation.

The Go standard library introduced `log/slog` in Go 1.21 (shark uses Go 1.23.4), providing a structured logging API that integrates cleanly with OpenTelemetry. Using `slog` avoids third-party logging library dependencies while aligning with the Go ecosystem's direction.

---

## 2. Goals and Success Criteria

### Goals

1. **Structured logging**: Replace all ad-hoc `log.Print*` and diagnostic `fmt.Print*` calls with `log/slog` structured logging, producing JSON-formatted output with consistent fields (timestamp, level, message, component, entity_key, operation, duration_ms, error).
2. **Distributed tracing**: Instrument the full request path -- CLI command entry, HTTP handler, service method, repository query, database operation -- with OpenTelemetry spans, enabling end-to-end trace visualization in any OTel-compatible backend.
3. **Operational metrics**: Collect and export key metrics via OpenTelemetry Metrics API: command execution counters, operation duration histograms, error counters by type, and database query performance histograms.
4. **Configurable exporters**: Support stdout/console exporter for local development (zero infrastructure) and OTLP exporter for production (sends to Jaeger, Grafana Tempo, Datadog, etc.), selectable via `.sharkconfig.json` or environment variables.
5. **Minimal performance impact**: Observability instrumentation adds less than 5% overhead to command execution time, and the no-op path (telemetry disabled) adds less than 1%.

### Measurable Success Criteria

| Criteria | Metric | Target |
|----------|--------|--------|
| Structured log coverage | Percentage of `log.Print*` and diagnostic `fmt.Print*` calls replaced with `slog` | 100% of `log.Print*` calls; 100% of diagnostic `fmt.Print*` in non-CLI-output code |
| Log format consistency | All log lines from shark processes parseable as JSON when JSON handler is configured | 100% |
| Trace span coverage | Percentage of service methods (TaskService, FeatureService, EpicService, BugService, ChangeCardService) with trace spans | 100% |
| HTTP trace coverage | Percentage of HTTP endpoints with request-level trace spans and trace-id headers | 100% |
| DB query tracing | Percentage of repository methods with trace spans recording query duration | 100% of read/write methods in TaskRepository, FeatureRepository, EpicRepository |
| Metric collection | Number of core metrics exported | At least 6: command_executions_total, command_duration_seconds, command_errors_total, db_query_duration_seconds, db_query_errors_total, http_request_duration_seconds |
| Performance overhead | CLI command execution time increase with tracing enabled vs disabled | Less than 5% (measured on `shark get E01` benchmark) |
| No-op overhead | CLI command execution time increase with telemetry fully disabled | Less than 1% |
| Exporter configurability | Number of supported exporter backends | 2 minimum: stdout (dev), OTLP/gRPC (production) |
| Existing test suite | All existing tests pass without modification (no test regressions) | 100% pass rate |
| Backward compatibility | CLI output for human users is unchanged when structured logging is not enabled | No visible change to default `shark` command output |

---

## 3. Scope

### In Scope

- **`internal/observability/` package**: A new package providing initialization functions for traces, metrics, and structured logging. Encapsulates OpenTelemetry SDK setup, exporter configuration, and shutdown lifecycle.
- **Structured logging migration**: Replace `log.Print*`, `log.Fatal*`, and diagnostic `fmt.Printf` calls across `internal/` and `cmd/` with `slog.Info`, `slog.Error`, `slog.Debug`, `slog.Warn` with structured key-value fields. Preserve `cli.Success`, `cli.Error`, `cli.Warning`, `cli.Info` for user-facing CLI output (these are not logging -- they are presentation).
- **CLI command tracing**: Wrap each Cobra command execution in a root span (command name, args, exit code, duration). Add spans for service method calls within each command.
- **HTTP middleware tracing**: Add OpenTelemetry HTTP middleware to `cmd/server/main.go` that creates a span per request, records HTTP method/path/status, and propagates trace context via W3C `traceparent` headers.
- **Service layer spans**: Add trace spans to all public methods on TaskService, FeatureService, EpicService, BugService, ChangeCardService, NoteService, ContextService, ResumeService. Each span records the operation name, entity key, duration, and error status.
- **Repository layer spans**: Add trace spans to repository read/write methods in TaskRepository, FeatureRepository, EpicRepository. Each span records the SQL operation type (SELECT/INSERT/UPDATE/DELETE), table name, and duration.
- **Database query metrics**: Record histogram metrics for database query durations, segmented by operation type and table. Record error counters for failed queries.
- **Command metrics**: Record counter and histogram metrics for CLI command executions (command name, duration, success/error).
- **Configuration via `.sharkconfig.json`**: Add an `observability` section to the config file supporting: `enabled` (bool), `exporter` ("stdout" | "otlp"), `otlp_endpoint` (string), `log_level` ("debug" | "info" | "warn" | "error"), `log_format` ("json" | "text").
- **Environment variable overrides**: Support `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, `SHARK_LOG_LEVEL`, `SHARK_LOG_FORMAT` environment variables for CI/CD and container environments.
- **Graceful shutdown**: Flush all pending spans and metrics on process exit via `defer` in CLI root command and HTTP server shutdown.
- **Documentation**: Update CLAUDE.md and add a `docs/guides/observability.md` guide covering configuration, local development setup, and production deployment with Jaeger/Grafana.

### Out of Scope

- **Custom OpenTelemetry collector deployment**: This epic instruments the shark binary. Deploying and configuring a collector (Jaeger, Grafana Tempo, etc.) is the operator's responsibility and is documented but not automated.
- **Alerting rules or dashboards**: No Grafana dashboards, PagerDuty integrations, or alert configurations are created. The epic produces metrics and traces; consuming them is a separate concern.
- **Profiling (pprof)**: CPU and memory profiling endpoints are not part of this epic. They may be added in a future observability extension.
- **Log aggregation infrastructure**: No ELK stack, Loki, or CloudWatch setup. Structured logs go to stdout/stderr; aggregation is the deployment environment's responsibility.
- **Distributed tracing across shark run agent processes**: Trace context propagation from `shark run` into spawned Claude/Codex CLI processes is not in scope. `shark run` spans cover the dispatch and wait; the agent's internal operations are not traced.
- **SQLite PRAGMA performance instrumentation**: PRAGMA settings like WAL mode, cache size, and mmap are not instrumented. Only application-level query performance is measured.
- **Modifying the `--verbose` flag behavior**: The existing `--verbose` flag remains unchanged. Structured logging is controlled by the `observability` config section, not by `--verbose`.
- **Tracing for sync, discovery, or file operations**: The `internal/sync/`, `internal/discovery/`, and `internal/fileops/` packages are not instrumented in this epic. They can be added incrementally in follow-up work.

---

## 4. Constraints and Assumptions

### Constraints

1. **Go standard library slog**: Structured logging must use `log/slog` from the Go standard library (available in Go 1.21+, shark uses 1.23.4). No third-party logging libraries (zap, zerolog, logrus) are introduced.
2. **OpenTelemetry Go SDK**: Tracing and metrics use the official `go.opentelemetry.io/otel` SDK packages. No vendor-specific SDKs (Datadog agent, New Relic agent) are embedded in the binary.
3. **No new required dependencies at runtime**: When observability is disabled (the default for existing users), the shark binary must behave identically to today. OpenTelemetry initialization is skipped entirely when `observability.enabled` is false or absent.
4. **CLI output preservation**: User-facing CLI output (`cli.Success`, `cli.Error`, `cli.OutputJSON`, `cli.OutputTable`) must not be contaminated with log lines. Structured logs go to stderr when running in CLI mode; stdout remains clean for piping and JSON parsing.
5. **Existing test patterns**: Service tests use mocked repositories. The observability layer must not require real OTel collectors in tests. Spans and metrics are either no-op'd or captured in-memory for assertion in tests.
6. **Binary size budget**: Adding OpenTelemetry dependencies should increase the compiled binary size by no more than 15MB (from current baseline).
7. **No breaking config changes**: The `.sharkconfig.json` `observability` section is additive. Existing config files without this section continue to work with observability disabled by default.

### Assumptions

1. **Developers have access to an OTel collector for production use**: For stdout exporter (dev mode), no infrastructure is needed. For OTLP exporter (production), the developer or operator provides a collector endpoint (Jaeger, Grafana Tempo, or similar).
2. **Context propagation through the stack**: The existing `context.Context` parameter on all service and repository methods provides the carrier for trace context. No method signature changes are needed.
3. **slog is sufficient for structured logging needs**: The `log/slog` API supports all required features: structured key-value fields, multiple handlers (JSON, text), level filtering, and handler chaining. Custom handler implementation is not required.
4. **OpenTelemetry Go SDK is stable**: The `go.opentelemetry.io/otel` v1.x SDK and `go.opentelemetry.io/otel/sdk` v1.x are production-ready and follow semantic versioning. Breaking changes are not expected within the v1 major version.
5. **Performance-sensitive code paths are limited**: Shark is a CLI tool and low-throughput HTTP API. The overhead of span creation (microseconds per span) is negligible relative to database I/O (milliseconds) and network I/O (tens of milliseconds for Turso).

---

## 5. Stakeholder Impact

### Developers Using Shark CLI

**Impact**: Low (positive). Default behavior is unchanged. Developers who opt in to structured logging get JSON-formatted diagnostic output on stderr that can be piped to `jq` or log aggregation tools. Developers who enable tracing get visibility into which operations are slow.

**Change required**: None for default usage. To enable observability, add an `observability` section to `.sharkconfig.json` or set environment variables.

### HTTP API Operators

**Impact**: High (positive). The HTTP API gains request-level tracing with trace-id headers, request duration metrics, and structured access logs. This is essential for operating the API in any environment beyond local development.

**Change required**: Configure an OTLP endpoint in `.sharkconfig.json` or set `OTEL_EXPORTER_OTLP_ENDPOINT`. Deploy an OTel-compatible collector (Jaeger, Grafana Tempo) if production-grade trace storage is needed.

### AI Agent Orchestration (shark run users)

**Impact**: Medium (positive). Each `shark run` stage produces trace spans showing the internal shark operations (status lookup, workflow validation, status advance, history recording) that surround the agent dispatch. Combined with the existing run logging from E22, this provides full visibility into both the orchestration loop and the shark internals.

**Change required**: None. Tracing is additive and does not affect `shark run` behavior.

### Shark Core Developers (contributors)

**Impact**: Medium (adjustment required). All new service methods, repository methods, and HTTP handlers must follow the instrumentation patterns established by this epic. The `internal/observability/` package provides helper functions to minimize boilerplate. Existing patterns for `log.Println` are replaced with `slog.Info` calls.

**Change required**: Learn the `slog` API and span creation pattern. Follow the instrumentation guide in `docs/guides/observability.md` for new code.

### CI/CD Pipelines

**Impact**: Low. The existing `make test`, `make lint`, and `make build` commands continue to work. Tests do not require an OTel collector. The binary size increases slightly due to the OpenTelemetry SDK dependency.

**Change required**: None.

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

### UAT-1: Structured Logging Replaces Ad-Hoc Logging

**Given** the shark codebase has been updated with structured logging
**When** a developer runs `grep -r 'log.Print' internal/ cmd/` on the codebase
**Then** zero results are returned (all `log.Print*`, `log.Fatal*`, `log.Println` calls have been replaced with `slog` equivalents), and running `shark get E01 2>/tmp/shark.log` with `log_format: "json"` configured produces parseable JSON log lines in `/tmp/shark.log` with fields: `time`, `level`, `msg`, `component`, and operation-specific attributes.

### UAT-2: CLI Command Tracing

**Given** observability is enabled with `exporter: "stdout"` in `.sharkconfig.json`
**When** a developer runs `shark get E01-F01-001`
**Then** trace spans are printed to stderr showing: a root span named `shark.command/get` with attributes `command=get`, `entity_key=E01-F01-001`, child spans for the service method call (`TaskService.GetByKey`), and a child span for the repository query (`TaskRepository.GetByKey`), each with duration recorded. The stdout output (the task details) is unaffected.

### UAT-3: HTTP Request Tracing

**Given** the HTTP API server is running with observability enabled and OTLP exporter configured
**When** a client sends `GET /api/v1/tasks/E01-F01-001` with a `traceparent` header
**Then** the response includes a `tracestate` header, the request produces a trace with spans covering: HTTP handler, service method, and repository query. The trace is exportable to the configured OTLP endpoint and visible in Jaeger or Grafana Tempo.

### UAT-4: Database Query Metrics

**Given** observability is enabled and the shark CLI executes 100 commands
**When** the operator inspects the exported metrics
**Then** a `shark.db.query.duration` histogram metric exists with labels for `operation` (select, insert, update, delete) and `table` (tasks, features, epics), showing p50, p95, and p99 latency values. A `shark.db.query.errors.total` counter records any query failures.

### UAT-5: Command Execution Metrics

**Given** observability is enabled
**When** the operator runs various shark commands (get, list, create, status advance)
**Then** a `shark.command.executions.total` counter metric exists with labels for `command` and `status` (success/error), and a `shark.command.duration.seconds` histogram records execution time per command name.

### UAT-6: Telemetry Disabled by Default

**Given** a fresh shark installation with no `observability` section in `.sharkconfig.json`
**When** a developer runs `shark get E01`
**Then** no trace spans are generated, no metrics are collected, no structured log output appears on stderr, CLI output is identical to the pre-E23 behavior, and command execution time is within 1% of the pre-E23 baseline.

### UAT-7: Configuration via Environment Variables

**Given** no `observability` section in `.sharkconfig.json` but the environment variables `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` and `SHARK_LOG_LEVEL=debug` are set
**When** a developer runs `shark get E01`
**Then** traces are exported to the OTLP endpoint at localhost:4317, and debug-level structured logs appear on stderr, demonstrating that environment variables override the absence of config file settings.

### UAT-8: Graceful Shutdown Flushes Telemetry

**Given** observability is enabled with OTLP exporter
**When** a long-running shark command completes (or the HTTP server receives SIGTERM)
**Then** all pending spans and metrics are flushed to the exporter before the process exits. No spans or metrics are lost due to premature process termination.

### UAT-9: Performance Overhead Budget

**Given** a benchmark test that runs `shark get E01` 1000 times
**When** the benchmark is run with observability enabled (stdout exporter, all spans and metrics active) and with observability disabled
**Then** the median execution time with observability enabled is no more than 5% higher than with observability disabled. With observability fully disabled (no-op), the difference is less than 1%.

### UAT-10: Existing Tests Pass Without Modification

**Given** the full E23 implementation is merged
**When** `make test` is run
**Then** all existing tests pass. No test files are modified to accommodate observability changes (observability is transparent to test code via no-op providers).

---

## Epic Components

This epic will be decomposed into features covering:

- **Observability foundation package** (`internal/observability/`) -- SDK initialization, exporter configuration, shutdown lifecycle
- **Structured logging migration** -- Replace ad-hoc logging across all packages
- **CLI command instrumentation** -- Root span creation in Cobra command lifecycle
- **Service layer instrumentation** -- Span creation in all service methods
- **Repository layer instrumentation** -- Span creation and query metrics in repository methods
- **HTTP middleware instrumentation** -- Request tracing and metrics for the API server
- **Configuration and documentation** -- Config schema, env var support, developer guide

---

## Quick Reference

**Primary Users**: Shark developers, HTTP API operators, DevOps engineers

**Key Capabilities**:
- Structured JSON logging via Go `log/slog` replacing all ad-hoc print statements
- OpenTelemetry distributed tracing from CLI entry through service to database
- Operational metrics (counters, histograms) for commands, queries, and HTTP requests
- Pluggable exporters: stdout for development, OTLP for production backends
- Zero overhead when disabled; less than 5% when enabled

**Success Criteria**:
- 100% of ad-hoc log calls replaced with structured slog
- 100% of service and repository methods instrumented with trace spans
- 6+ operational metrics exported
- All existing tests pass without modification

**Timeline**: No external deadline. Internal priority: medium.

---

## Open Questions & Assumptions

No open questions -- all epic-level decisions are resolved.

---

*Last Updated*: 2026-03-22
