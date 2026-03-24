# E23 Architecture: OpenTelemetry Observability

**Date:** 2026-03-22
**Epic:** E23 - OpenTelemetry Observability (tracing, metrics, structured logging)
**Status:** Approved

---

## 1. Component Overview

### What Changes

| Component | Change Type | Description |
|-----------|------------|-------------|
| `internal/observability/` | **NEW** | Observability foundation package (provider, logger, noop, middleware) |
| `internal/cli/observability_global.go` | **NEW** | Global OTel provider singleton following `db_global.go` pattern |
| `internal/config/config.go` | **EXTEND** | Add `ObservabilityConfig` struct and field |
| `internal/cli/root.go` | **EXTEND** | OTel init in `PersistentPreRunE`, shutdown in `PersistentPostRunE` |
| `internal/cli/services_global.go` | **EXTEND** | Pass tracer to service constructors; add OTel reset to `ResetServices()` |
| `internal/services/*.go` | **EXTEND** | Accept `trace.Tracer` via constructor; add spans to public methods |
| `internal/repository/*.go` | **EXTEND** | Add spans to read/write methods using package-level tracer |
| `cmd/server/main.go` | **EXTEND** | OTel provider init, `otelhttp` middleware |
| All files with `log.Print*` | **EXTEND** | Replace with `slog.*` equivalents (mechanical migration) |

### What Stays Unchanged

| Component | Reason |
|-----------|--------|
| `cli.Success/Error/Warning/Info/OutputJSON/OutputTable` | User-facing presentation, not logging |
| `internal/models/` | No observability concerns in data models |
| `internal/db/db.go` (schema) | No schema changes for observability |
| Test files | Tests use no-op providers by default; no modifications needed |
| `--verbose` flag | Remains independent of structured logging; unchanged behavior |
| `internal/sync/`, `internal/discovery/`, `internal/fileops/` | Explicitly out of scope per PRD |

---

## 2. Architecture Decision Records

### ADR-1: Use `log/slog` from Go Standard Library (Not zap/zerolog)

**Date:** 2026-03-22
**Status:** Accepted

**Context:** Shark needs structured logging to replace 91+ ad-hoc `log.Print*` calls. Third-party options (zap, zerolog, logrus) and the standard library `log/slog` (Go 1.21+) are all viable.

**Decision:** Use `log/slog` exclusively. No third-party logging libraries.

**Rationale:**
- Shark uses Go 1.23.4; `slog` is stable and mature at this version.
- Zero additional dependency -- reduces binary bloat and supply-chain surface.
- `slog` supports JSON and text handlers, level filtering, structured key-value fields, and custom handlers.
- The Go ecosystem is converging on `slog`; OTel Go SDK provides a `slog`-compatible bridge.
- Simpler onboarding for contributors (standard library, not a third-party API).

**Consequences:**
- (+) No new dependency for logging.
- (+) Future Go versions will continue improving `slog`.
- (-) Slightly fewer features than zap (no caller-site optimization, no sugared logger). These features are not needed for a CLI tool.

### ADR-2: OpenTelemetry Go SDK for Tracing and Metrics

**Date:** 2026-03-22
**Status:** Accepted

**Context:** Need distributed tracing and metrics collection with pluggable backends.

**Decision:** Use the official `go.opentelemetry.io/otel` SDK (v1.x) for tracing and metrics. Use `otelhttp` contrib package for HTTP middleware.

**Rationale:**
- CNCF graduated project; vendor-neutral; supported by every major APM vendor.
- Stable v1.x API with semantic versioning guarantees.
- Native Go SDK with excellent context propagation support.
- Supports both stdout (development) and OTLP/gRPC (production) exporters.
- `otelhttp` provides automatic HTTP request tracing with W3C `traceparent` propagation.

**Consequences:**
- (+) Works with Jaeger, Grafana Tempo, Datadog, Honeycomb, AWS X-Ray out of the box.
- (+) Single instrumentation, multiple backends.
- (-) Adds ~15-25 transitive Go module dependencies and ~10-15MB to binary size.
- (-) OTel SDK initialization adds complexity to the startup path (mitigated by no-op default).

### ADR-3: Disabled by Default with No-Op Provider Pattern

**Date:** 2026-03-22
**Status:** Accepted

**Context:** Shark is primarily a CLI tool used by developers. Most users do not have an OTel collector running. Observability must not degrade the default experience.

**Decision:** All telemetry is disabled by default. When `observability.enabled` is `false` or absent, the OTel SDK is not initialized. Services receive a no-op `trace.Tracer`, the global `slog` logger uses a discard handler at `slog.LevelError+1` (effectively discarded), and no metrics are collected. The no-op path adds less than 1% overhead.

**Rationale:**
- Preserves identical behavior for existing users.
- No network connections, goroutines, or file I/O when disabled.
- The `noop.TracerProvider` from OTel SDK is purpose-built for this pattern.
- CLI command output on stdout remains clean.

**Consequences:**
- (+) Zero impact on users who do not opt in.
- (+) Tests run with no-op providers by default -- no test changes needed.
- (-) Two code paths (enabled/disabled) to maintain. Mitigated by the no-op pattern being a single branch at initialization time; all instrumented code is identical regardless.

### ADR-4: Constructor Injection for Tracers in Services; Package-Level Tracer in Repositories

**Date:** 2026-03-22
**Status:** Accepted

**Context:** Services are tested with mocked repositories. Repositories are tested with real databases. The tracer injection pattern should match each layer's testing strategy.

**Decision:**
- **Services** receive a `trace.Tracer` via constructor injection (alongside existing repository and workflow dependencies). When nil, a no-op tracer is used. This makes services testable with mock tracers.
- **Repositories** use a package-level tracer obtained via `otel.Tracer("internal/repository")`. Since repositories are tested with real databases (not mocked), injecting a tracer adds no testability value and would inflate constructor signatures.

**Rationale:**
- Aligns with existing testing architecture: services use mocks, repositories use real DB.
- Constructor injection for services is the established pattern (see `NewTaskService`).
- Package-level `otel.Tracer()` is the standard OTel Go idiom for library code.
- The global `otel.SetTracerProvider()` call in `PersistentPreRunE` makes package-level tracers work correctly.

**Consequences:**
- (+) Services remain fully testable with mock tracers.
- (+) Repository instrumentation is simple (2-3 lines per method, no constructor changes).
- (-) Repository tracers cannot be independently mocked. Acceptable because repository tests focus on data access correctness, not tracing behavior.

### ADR-5: stderr for All Telemetry Output; stdout Reserved for CLI Data

**Date:** 2026-03-22
**Status:** Accepted

**Context:** AI agents and scripts parse `shark --json` output from stdout. Any telemetry leaking to stdout breaks these integrations.

**Decision:** All slog output and OTel stdout exporter output are directed to `os.Stderr`. This is enforced at the `internal/observability/logger.go` level. The `cli.OutputJSON()` and `cli.OutputTable()` functions continue writing to `os.Stdout` exclusively.

**Rationale:**
- Hard constraint from the existing CLI architecture.
- `slog.NewJSONHandler(os.Stderr, ...)` makes this trivial.
- OTel stdout exporter accepts an `io.Writer` parameter; pass `os.Stderr`.

**Consequences:**
- (+) Zero risk of stdout contamination.
- (+) Developers can capture diagnostics via `2>/tmp/shark.log` without affecting primary output.
- (-) Users must redirect stderr to see structured logs (this is expected and standard for CLI tools).

### ADR-6: Batch Span Processor for OTLP; Simple Span Processor for stdout

**Date:** 2026-03-22
**Status:** Accepted

**Context:** CLI commands are short-lived processes (50-500ms). The span processor choice affects whether telemetry is flushed before exit.

**Decision:**
- **stdout exporter:** Use `sdktrace.NewSimpleSpanProcessor` (synchronous, immediate write). Acceptable for development because stdout writes are fast.
- **OTLP exporter:** Use `sdktrace.NewBatchSpanProcessor` (asynchronous, batched network writes). Call `provider.Shutdown(ctx)` with a 5-second timeout in `PersistentPostRunE` to flush pending spans.

**Rationale:**
- Simple processor for stdout avoids the complexity of batch flushing for a local-only exporter.
- Batch processor for OTLP avoids blocking on network I/O during command execution.
- The 5-second shutdown timeout ensures spans are flushed even for fast commands.

**Consequences:**
- (+) No visible latency increase for CLI commands with OTLP exporter.
- (+) stdout exporter shows spans immediately (useful for debugging).
- (-) If the process is killed (SIGKILL), pending OTLP spans may be lost. Acceptable for a CLI tool.

---

## 3. Data Model Changes

**None.** Observability is a cross-cutting concern that does not modify the database schema, entity models, or API contracts. No migration is needed.

---

## 4. Integration Approach

### 4.1 CLI Lifecycle Integration

```
PersistentPreRunE (root.go)
  |
  +-- initConfig()                        [existing]
  +-- templates.SetConfiguredTemplateDir() [existing]
  +-- pterm color/debug config             [existing]
  +-- observability.Init(cfg)              [NEW - returns Shutdown func]
  |     +-- Configure slog.SetDefault()
  |     +-- Initialize TracerProvider (or no-op)
  |     +-- Initialize MeterProvider (or no-op)
  |     +-- Record command start time for metrics
  |
  v
Command Execution
  |
  v
PersistentPostRunE (root.go)
  |
  +-- Record command duration metric       [NEW]
  +-- observability.Shutdown(ctx)          [NEW - flushes spans/metrics]
  +-- CloseDB()                            [existing]
```

The `observability.Init()` call returns a `Shutdown` function stored in the global singleton. Both init and shutdown are guarded by `sync.Once` in `observability_global.go`, matching the `db_global.go` pattern exactly.

### 4.2 Service Layer Integration

Services gain a `trace.Tracer` field. The tracer is passed via constructor or setter method (matching the existing `SetFeatureRepo`, `SetHistoryRepo` pattern for optional dependencies).

```go
// In task_service.go
type TaskService struct {
    // ... existing fields ...
    tracer trace.Tracer  // NEW: defaults to noop.Tracer{} when nil
}

// Setter follows existing pattern (SetFeatureRepo, SetHistoryRepo, etc.)
func (s *TaskService) SetTracer(t trace.Tracer) {
    s.tracer = t
}

// Usage in methods (2-line addition per method):
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := s.tracer.Start(ctx, "TaskService.GetTask",
        trace.WithAttributes(attribute.String("entity.key", key)))
    defer span.End()

    // ... existing implementation unchanged ...
}
```

The `GetTaskService()` accessor in `services_global.go` wires the tracer from the global OTel provider:

```go
func GetTaskService() *services.TaskService {
    // ... existing wiring ...
    svc.SetTracer(GetTracer("services/task"))
    return svc
}
```

### 4.3 Repository Layer Integration

Repositories use a package-level tracer. No constructor changes.

```go
// In task_repository.go (package level)
var repoTracer = otel.Tracer("internal/repository")

// Usage in methods:
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := repoTracer.Start(ctx, "TaskRepository.GetByKey",
        trace.WithAttributes(
            attribute.String("db.operation", "SELECT"),
            attribute.String("db.table", "tasks"),
        ))
    defer span.End()

    // ... existing query unchanged; ctx carries span ...
}
```

When observability is disabled, `otel.Tracer()` returns a no-op tracer (because the global TracerProvider was never set to a real provider). The span creation becomes a no-op with near-zero overhead.

### 4.4 HTTP Server Integration

```go
// In cmd/server/main.go
func main() {
    // Initialize OTel provider
    cfg := loadConfig()
    shutdown := observability.Init(cfg.Observability)
    defer shutdown(context.Background())

    // ... existing DB init ...

    // Wrap mux with OTel HTTP middleware
    mux := http.NewServeMux()
    // ... register handlers ...

    handler := otelhttp.NewHandler(mux, "shark-server",
        otelhttp.WithTracerProvider(otel.GetTracerProvider()),
        otelhttp.WithMeterProvider(otel.GetMeterProvider()),
    )

    slog.Info("starting server", "port", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

The `otelhttp` middleware automatically:
- Creates a span per request with HTTP method, path, status code
- Propagates W3C `traceparent` headers
- Records `http.server.request.duration` histogram metric

### 4.5 Structured Logging Migration

The migration is mechanical. All 23 `log.Print*` calls in `internal/` and 68 in `cmd/` are replaced with `slog` equivalents. The global `slog.SetDefault()` call in `PersistentPreRunE` means no import changes beyond replacing `log` with `log/slog`.

**Migration rules:**
- `log.Printf("Warning: %s", msg)` becomes `slog.Warn(msg, "key", value)`
- `log.Printf("Error: %v", err)` becomes `slog.Error("description", "error", err)`
- `log.Fatal(...)` becomes `slog.Error(...); os.Exit(1)` (slog has no Fatal; explicit exit is clearer)
- `log.Println(msg)` becomes `slog.Info(msg)`
- `fmt.Fprintf(os.Stderr, "warning: %s\n", msg)` becomes `slog.Warn(msg)`
- `cli.Success/Error/Warning/Info` are **NOT** touched (presentation, not logging)

### 4.6 Metrics Collection Points

| Metric Name | Type | Labels | Collection Point |
|------------|------|--------|-----------------|
| `shark.cli.command.duration` | Histogram | `command`, `status` | `PersistentPostRunE` |
| `shark.cli.command.invocations` | Counter | `command`, `status` | `PersistentPostRunE` |
| `shark.cli.command.errors` | Counter | `command`, `error_type` | `PersistentPostRunE` |
| `shark.db.query.duration` | Histogram | `operation`, `table` | Repository span end |
| `shark.db.query.errors` | Counter | `operation`, `table` | Repository error path |
| `shark.http.request.duration` | Histogram | `method`, `path`, `status` | `otelhttp` middleware (auto) |

CLI command metrics are recorded in `PersistentPostRunE` using the command name and execution result stored during `PersistentPreRunE` (start time) and the command's return error.

---

## 5. Package Structure

```
internal/observability/
    provider.go         # OTel SDK bootstrap: InitProvider(cfg) -> (ShutdownFunc, error)
                        #   - Creates TracerProvider with configured exporter
                        #   - Creates MeterProvider with configured exporter
                        #   - Configures W3C trace context propagator
                        #   - Returns shutdown function for graceful flush

    logger.go           # Structured logging setup: InitLogger(cfg)
                        #   - Creates slog.JSONHandler or slog.TextHandler
                        #   - Writes to os.Stderr (never stdout)
                        #   - Sets log level from config
                        #   - Calls slog.SetDefault() for global logger
                        #   - Adds default attributes (service.name, service.version)

    noop.go             # No-op providers: NoopProvider() -> (ShutdownFunc)
                        #   - Returns noop.TracerProvider
                        #   - Returns noop.MeterProvider
                        #   - Sets slog to discard handler
                        #   - ShutdownFunc is a no-op

    metrics.go          # Metric instrument definitions:
                        #   - NewCommandMetrics(meter) -> CommandMetrics
                        #   - CommandMetrics.RecordDuration(command, duration, err)
                        #   - CommandMetrics.RecordInvocation(command, err)
                        #   - NewDBMetrics(meter) -> DBMetrics
                        #   - DBMetrics.RecordQueryDuration(operation, table, duration, err)

    config.go           # ObservabilityConfig struct definition
                        #   (or placed in internal/config/config.go as extension)

    middleware.go        # HTTP middleware helpers (optional; may use otelhttp directly)
```

**Dependency hierarchy:**
```
internal/cli/              (imports observability, config)
    |
    v
internal/observability/    (imports config, OTel SDK)
    |
    v
internal/config/           (imports db for DatabaseConfig)
    |
    v
(no internal imports)
```

No circular dependency risk. `internal/observability/` never imports `internal/cli/`, `internal/services/`, or `internal/repository/`.

---

## 6. Configuration Schema

### `.sharkconfig.json` Extension

```json
{
  "observability": {
    "enabled": false,
    "tracing_enabled": true,
    "metrics_enabled": true,
    "log_level": "info",
    "log_format": "json",
    "exporter": "stdout",
    "otlp_endpoint": "localhost:4317",
    "otlp_protocol": "grpc",
    "service_name": "shark-task-manager"
  }
}
```

### Field Definitions

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Master switch. When false, all telemetry is off (no-op providers). |
| `tracing_enabled` | bool | `true` | Enable tracing (only effective when `enabled=true`). |
| `metrics_enabled` | bool | `true` | Enable metrics (only effective when `enabled=true`). |
| `log_level` | string | `"info"` | Minimum log level: `"debug"`, `"info"`, `"warn"`, `"error"`. |
| `log_format` | string | `"json"` | Log output format: `"json"` (structured) or `"text"` (human-readable). |
| `exporter` | string | `"stdout"` | Exporter type: `"stdout"` (dev), `"otlp"` (production). |
| `otlp_endpoint` | string | `"localhost:4317"` | OTLP collector endpoint (only used when `exporter="otlp"`). |
| `otlp_protocol` | string | `"grpc"` | OTLP transport: `"grpc"` or `"http"`. |
| `service_name` | string | `"shark-task-manager"` | OTel service name resource attribute. |

### Go Struct

```go
// ObservabilityConfig holds configuration for the observability subsystem.
// All fields have sensible defaults; the zero value means "disabled".
type ObservabilityConfig struct {
    Enabled        bool   `json:"enabled"`
    TracingEnabled bool   `json:"tracing_enabled"`
    MetricsEnabled bool   `json:"metrics_enabled"`
    LogLevel       string `json:"log_level"`
    LogFormat      string `json:"log_format"`
    Exporter       string `json:"exporter"`
    OTLPEndpoint   string `json:"otlp_endpoint"`
    OTLPProtocol   string `json:"otlp_protocol"`
    ServiceName    string `json:"service_name"`
}
```

Added to `internal/config/config.go`:
```go
type Config struct {
    // ... existing fields ...
    Observability ObservabilityConfig `json:"observability,omitempty"`
}
```

### Environment Variable Overrides

Environment variables take precedence over `.sharkconfig.json` values. Checked during `observability.Init()`.

| Environment Variable | Overrides | Standard |
|---------------------|-----------|----------|
| `SHARK_OTEL_ENABLED` | `observability.enabled` | Shark-specific |
| `SHARK_LOG_LEVEL` | `observability.log_level` | Shark-specific |
| `SHARK_LOG_FORMAT` | `observability.log_format` | Shark-specific |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `observability.otlp_endpoint` | OTel standard |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `observability.otlp_protocol` | OTel standard |
| `OTEL_SERVICE_NAME` | `observability.service_name` | OTel standard |

Using standard OTel environment variable names (`OTEL_*`) where they exist ensures compatibility with OTel ecosystem tooling (auto-instrumentation agents, collector configs, CI/CD pipelines).

### Backward Compatibility

- The `observability` key is `omitempty` -- existing configs without it are valid.
- Unknown fields in `.sharkconfig.json` are preserved via the existing `RawData` mechanism.
- No migration or version bump needed. The config is purely additive.

---

## 7. Go Module Dependencies

### Required Packages

```
go.opentelemetry.io/otel                          # Core API (Tracer, Meter interfaces)
go.opentelemetry.io/otel/trace                    # Trace API (Span, SpanContext)
go.opentelemetry.io/otel/metric                   # Metric API (Counter, Histogram)
go.opentelemetry.io/otel/attribute                 # Attribute key-value pairs
go.opentelemetry.io/otel/sdk/trace                 # TracerProvider implementation
go.opentelemetry.io/otel/sdk/metric                # MeterProvider implementation
go.opentelemetry.io/otel/sdk/resource              # Service resource (name, version)
go.opentelemetry.io/otel/propagation               # W3C trace context propagation
go.opentelemetry.io/otel/exporters/stdout/stdouttrace   # Stdout trace exporter
go.opentelemetry.io/otel/exporters/stdout/stdoutmetric  # Stdout metric exporter
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc  # OTLP gRPC trace exporter
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc # OTLP gRPC metric exporter
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp      # HTTP middleware
```

### Not Required (Explicitly Excluded)

- `go.opentelemetry.io/otel/log` -- slog bridge is not needed; we use slog directly.
- `go.opentelemetry.io/otel/exporters/zipkin` -- Zipkin is not a target exporter.
- `go.opentelemetry.io/otel/exporters/prometheus` -- Prometheus pull model is not suited for CLI tools.
- Any vendor-specific SDK (Datadog agent, New Relic agent, etc.).

### Binary Size Impact

Estimated increase: 10-15MB (from transitive gRPC and protobuf dependencies). This is within the 15MB budget specified in the PRD constraints.

---

## 8. Feature Decomposition Guidance

The epic should be decomposed into the following features, ordered by dependency:

**F01: Observability Foundation Package** -- `internal/observability/` with provider, logger, noop, config. No integration with existing code.

**F02: CLI Lifecycle Integration** -- Wire `observability.Init()` into `PersistentPreRunE`, shutdown into `PersistentPostRunE`, global singleton in `observability_global.go`, reset in `ResetServices()`.

**F03: Structured Logging Migration** -- Replace all `log.Print*` and diagnostic `fmt.Fprintf(os.Stderr)` with `slog` calls across `internal/` and `cmd/`.

**F04: Service Layer Tracing** -- Add `trace.Tracer` to service constructors/setters, instrument all public methods with spans.

**F05: Repository Layer Tracing** -- Add package-level tracer to repositories, instrument read/write methods with spans and DB attributes.

**F06: CLI Command Metrics** -- Record command invocations, duration, and error counters in `PersistentPreRunE`/`PostRunE`.

**F07: HTTP Server Instrumentation** -- OTel provider init in `cmd/server/main.go`, `otelhttp` middleware, structured logging migration for server.

**F08: Documentation and Guide** -- `docs/guides/observability.md`, CLAUDE.md updates, developer onboarding guide.

Each feature is independently deployable. F01 must ship first; F02 depends on F01; F03-F07 can proceed in parallel after F02.

---

## 9. Risk Mitigations Summary

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| stdout contamination | High | High | Enforce `os.Stderr` in `logger.go`; code review checklist |
| Test isolation (global OTel state) | High | Medium | No-op default; `ResetObservability()` in test teardown |
| Import cycle | Low | High | Strict hierarchy: `cli` -> `observability` -> `config` |
| Binary size exceeds budget | Low | Low | Use only needed OTel packages; measure before merge |
| Performance regression (disabled) | Low | Medium | Benchmark gate: `shark get E01` must be within 1% of baseline |
| Performance regression (enabled) | Low | Low | Batch processor for OTLP; benchmark gate: within 5% of baseline |

---

*Last Updated: 2026-03-22*
