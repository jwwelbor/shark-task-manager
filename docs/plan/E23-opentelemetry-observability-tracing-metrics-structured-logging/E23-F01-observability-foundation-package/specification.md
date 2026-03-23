# E23-F01: Observability Foundation Package — Technical Specification

**Feature Key:** E23-F01-observability-foundation-package
**Epic:** E23 — OpenTelemetry Observability (tracing, metrics, structured logging)
**Date:** 2026-03-22
**Status:** Approved for Development

---

## 1. Feature Overview and Purpose

### Problem Statement

Shark Task Manager currently has no unified observability story. Diagnostic output is scattered across 23+ `log.Print*` calls and 18+ `fmt.Fprintf(os.Stderr, ...)` calls, none of which are structured, filterable, or machine-readable. There is no distributed tracing, no metrics collection, and no way to integrate with APM backends (Jaeger, Grafana Tempo, Datadog, etc.). AI agents and CI pipelines consume `shark --json` output from stdout; any unstructured diagnostic leakage to stdout breaks these integrations.

### Solution

F01 creates the `internal/observability/` package — a self-contained foundation that all subsequent E23 features build upon. It installs the OpenTelemetry Go SDK providers (or no-ops when disabled), configures the global `slog` structured logger, defines the metric instrument helpers, and extends `internal/config/config.go` with the `ObservabilityConfig` struct.

This feature ships as **purely net-new code**. No existing file is modified except for one struct field addition and one parsing block addition in `internal/config/`. No behavior changes for existing users: the zero value of `ObservabilityConfig` disables all telemetry.

### Purpose within the Epic

F01 is the mandatory first step. Its exported types and functions define the contracts that F02 (CLI lifecycle integration), F03 (structured logging migration), F04 (service tracing), F05 (repository tracing), and F06 (metrics) all depend on. It cannot be skipped, reordered, or merged with another feature.

### Impact

- Establishes a zero-cost abstraction for tracing: when disabled, all OTel calls become no-ops with less than 1% overhead.
- Provides a single initialization entry point (`observability.InitProvider`) that F02 wires into the Cobra lifecycle.
- Eliminates the need for future features to understand OTel SDK internals — they call the helper types defined here.
- Binary size increase: approximately 10-15MB (within the 15MB budget, per ADR-2).

---

## 2. Detailed Requirements

### 2.1 Functional Requirements

#### Category: Package Initialization

**REQ-F-001: Provider Initialization**
- Description: `InitProvider(cfg config.ObservabilityConfig)` must initialize an OTel `TracerProvider` and `MeterProvider` based on the supplied config. When `cfg.Enabled` is `false`, it must delegate to `NoopProvider()` with no SDK initialization and no network connections attempted.
- Priority: Must-Have
- Acceptance Criteria:
  - When `cfg.Enabled = false`, calling `InitProvider` returns a non-nil `ShutdownFunc` and no error, and the global OTel TracerProvider is set to a no-op provider.
  - When `cfg.Enabled = true` and `cfg.Exporter = "stdout"`, calling `InitProvider` returns a functioning TracerProvider writing to `os.Stderr` (never stdout).
  - When `cfg.Enabled = true` and `cfg.Exporter = "otlp"`, calling `InitProvider` creates a gRPC OTLP exporter pointed at `cfg.OTLPEndpoint`. Connection is lazy (does not fail at init if the collector is unreachable).
  - The returned `ShutdownFunc` flushes all pending spans and metrics when called with a context, and returns any flush error.
  - Calling `ShutdownFunc` twice is safe (idempotent).

**REQ-F-002: Logger Initialization**
- Description: `InitLogger(cfg config.ObservabilityConfig)` must configure the global `slog` default logger. When disabled, output must be silently discarded. All log output must go to `os.Stderr` exclusively.
- Priority: Must-Have
- Acceptance Criteria:
  - When `cfg.Enabled = false`, `slog.Info("msg")` produces no output (handler level above `slog.LevelError`).
  - When `cfg.LogFormat = "json"`, log lines are valid JSON objects.
  - When `cfg.LogFormat = "text"`, log lines are human-readable key=value format.
  - Log output never appears on stdout (verified by capturing stdout in tests).
  - The `service.name` attribute is always present in every log record when enabled.
  - `cfg.LogLevel = "debug"` enables debug-level log entries; `"error"` suppresses info and warn.

**REQ-F-003: No-Op Provider**
- Description: `NoopProvider()` must configure the OTel global TracerProvider to the `noop.TracerProvider` and return a `ShutdownFunc` that is a no-op. This allows `otel.Tracer("pkg")` calls throughout the codebase to function without panicking even when observability is disabled.
- Priority: Must-Have
- Acceptance Criteria:
  - After calling `NoopProvider()`, `otel.Tracer("test").Start(ctx, "span")` returns without panicking.
  - The returned `ShutdownFunc` returns `nil` when called.
  - No goroutines are started, no file handles opened, no network connections attempted.

**REQ-F-004: Metric Instrument Helpers**
- Description: `NewCommandMetrics(meter)` and `NewDBMetrics(meter)` must create and return instrument structs holding pre-registered OTel metric instruments.
- Priority: Must-Have
- Acceptance Criteria:
  - `NewCommandMetrics` creates a `Float64Histogram` named `shark.cli.command.duration` (unit: `ms`), an `Int64Counter` named `shark.cli.command.invocations`, and an `Int64Counter` named `shark.cli.command.errors`.
  - `NewDBMetrics` creates a `Float64Histogram` named `shark.db.query.duration` (unit: `ms`) and an `Int64Counter` named `shark.db.query.errors`.
  - Both constructors return an error if instrument registration fails.
  - `CommandMetrics.RecordDuration(ctx, command, duration, err)` records a histogram observation with `command` and `status` (ok/error) attributes.
  - All methods are safe to call when the meter is a no-op meter.

**REQ-F-005: Config Struct Extension**
- Description: `internal/config/config.go` must gain an `Observability ObservabilityConfig` field and the `ObservabilityConfig` struct definition. `internal/config/manager.go` must parse the `"observability"` JSON key from `rawData`.
- Priority: Must-Have
- Acceptance Criteria:
  - Existing `.sharkconfig.json` files without an `"observability"` key load without error; `Config.Observability` is the zero value.
  - A `.sharkconfig.json` with `{"observability": {"enabled": true, "log_format": "json"}}` produces `Config.Observability.Enabled = true` and `Config.Observability.LogFormat = "json"`.
  - All nine `ObservabilityConfig` fields are parsed correctly.
  - `omitempty` on the field means JSON marshaling of a zero-value `ObservabilityConfig` omits the key.

**REQ-F-006: Environment Variable Overrides**
- Description: `applyEnvOverrides` (called inside `InitProvider` and `InitLogger`) must apply environment variables on top of the config struct values.
- Priority: Must-Have
- Acceptance Criteria:
  - `SHARK_OTEL_ENABLED=true` sets `cfg.Enabled = true` even if config file has `false`.
  - `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317` overrides `cfg.OTLPEndpoint`.
  - `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` overrides `cfg.OTLPProtocol`.
  - `OTEL_SERVICE_NAME=myservice` overrides `cfg.ServiceName`.
  - `SHARK_LOG_LEVEL=debug` overrides `cfg.LogLevel`.
  - `SHARK_LOG_FORMAT=text` overrides `cfg.LogFormat`.
  - Unset environment variables do not override config file values.

#### Category: Package Design Constraints

**REQ-F-007: No Import Cycles**
- Description: `internal/observability/` must never import `internal/cli/`, `internal/services/`, or `internal/repository/`. The allowed import graph is: `internal/cli/` -> `internal/observability/` -> `internal/config/` -> `internal/db/`.
- Priority: Must-Have
- Acceptance Criteria:
  - `go build ./internal/observability/...` succeeds with no import cycle errors.
  - `go vet ./internal/observability/...` passes.

**REQ-F-008: No Behavior Change for Existing Users**
- Description: When `observability` is absent from `.sharkconfig.json` and no `SHARK_OTEL_ENABLED` environment variable is set, all existing commands must behave identically to before this feature was merged.
- Priority: Must-Have
- Acceptance Criteria:
  - `make test` passes without modification.
  - `shark get E01` execution time is within 1% of baseline (measured by `go test -bench`).
  - No additional output appears on stdout or stderr when observability is disabled.

**REQ-F-009: Stdout Protection**
- Description: All OTel and slog output must be directed to `os.Stderr`, never `os.Stdout`.
- Priority: Must-Have
- Acceptance Criteria:
  - In unit tests that capture stdout, calling `InitProvider` + `InitLogger` with `Enabled: true` and `Exporter: "stdout"` produces zero bytes on stdout.
  - Verified by redirecting stdout to a buffer in test and asserting it remains empty.

### 2.2 Non-Functional Requirements

**REQ-NF-001: Performance — Disabled Path Overhead**
- Description: When observability is disabled, the overhead introduced by this feature must be negligible.
- Target: Less than 1% increase in `shark get E01` wall-clock execution time compared to the baseline before this feature.
- Measurement: `go test -bench=BenchmarkSharkGet -count=5` before and after the feature branch.

**REQ-NF-002: Performance — Enabled Path Overhead**
- Description: When observability is enabled with the stdout exporter, CLI command execution overhead must remain acceptable for interactive use.
- Target: Less than 5% increase in wall-clock time for typical commands.
- Measurement: `go test -bench=BenchmarkSharkGet -count=5` with `SHARK_OTEL_ENABLED=true SHARK_LOG_FORMAT=text`.

**REQ-NF-003: Binary Size**
- Description: Adding OTel SDK dependencies must not exceed the 15MB binary size budget established in ADR-2.
- Target: Binary size increase (measured by `du -sh ./bin/shark` before vs. after) must be at most 15MB.
- Measurement: Compare binary sizes on a clean build.

**REQ-NF-004: Test Isolation**
- Description: OTel uses global state (`otel.SetTracerProvider()`). Tests that call `InitProvider` or `NoopProvider` must leave no state that affects other tests.
- Target: All tests in `internal/observability/` pass when run with `go test -count=2 ./internal/observability/...` (repeated execution must be stable).
- Measurement: `go test -count=2 -race ./internal/observability/...` must pass.

**REQ-NF-005: Backward Compatibility**
- Description: No breaking changes to existing public APIs in `internal/config/`.
- Target: All existing callers of `config.Config` compile without modification.
- Measurement: `go build ./...` passes after the change.

---

## 3. Technical Design

### 3.1 Package Structure

```
internal/observability/
    provider.go       # InitProvider, ShutdownFunc, applyEnvOverrides
    logger.go         # InitLogger, parseLogLevel
    noop.go           # NoopProvider
    metrics.go        # CommandMetrics, DBMetrics, NewCommandMetrics, NewDBMetrics
```

The `ObservabilityConfig` struct lives in `internal/config/config.go`, not in `internal/observability/`, to prevent an import cycle. The `internal/observability/` package imports `internal/config/` for the struct; if the struct were defined in `internal/observability/`, the cycle `config -> observability -> config` would be created.

### 3.2 File-by-File Design

#### `internal/observability/provider.go`

Responsibilities:
- Define `ShutdownFunc` type alias.
- Implement `InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error)`.
- Implement `applyEnvOverrides(cfg *config.ObservabilityConfig)` (called at the top of `InitProvider`).
- Build `sdkresource.Resource` with `semconv.ServiceNameKey`, `semconv.ServiceVersionKey`.
- Select exporter based on `cfg.Exporter`: `"stdout"` or `"otlp"` (gRPC).
- For stdout exporter: use `sdktrace.NewSimpleSpanProcessor` (ADR-6) writing to `os.Stderr`.
- For OTLP exporter: use `sdktrace.NewBatchSpanProcessor` with a `otlptracegrpc` connection to `cfg.OTLPEndpoint`.
- Configure `MeterProvider` analogously with `stdoutmetric` or `otlpmetricgrpc`.
- Set W3C propagator via `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(...))`.
- Set global providers via `otel.SetTracerProvider(tp)` and `otel.SetMeterProvider(mp)`.
- Return a `ShutdownFunc` that calls `tp.Shutdown(ctx)` and `mp.Shutdown(ctx)` in sequence, collecting errors.

Public API:
```go
// ShutdownFunc gracefully shuts down the OTel provider,
// flushing any pending spans or metrics before returning.
type ShutdownFunc func(ctx context.Context) error

// InitProvider initializes the OpenTelemetry TracerProvider and MeterProvider.
// When cfg.Enabled is false, delegates to NoopProvider() — no SDK is initialized,
// no network connections are made, and the returned ShutdownFunc is a no-op.
func InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error)
```

Internal functions (unexported):
```go
func applyEnvOverrides(cfg *config.ObservabilityConfig)
func buildResource(cfg config.ObservabilityConfig) (*sdkresource.Resource, error)
func buildTracerProvider(cfg config.ObservabilityConfig, res *sdkresource.Resource) (*sdktrace.TracerProvider, error)
func buildMeterProvider(cfg config.ObservabilityConfig, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error)
```

#### `internal/observability/logger.go`

Responsibilities:
- Implement `InitLogger(cfg config.ObservabilityConfig)`.
- When `cfg.Enabled = false`: set global slog to a text handler on `os.Stderr` with level `slog.LevelError + 1` (effectively discards everything). This avoids panics from code that calls `slog.Info()` before observability is initialized.
- When `cfg.Enabled = true`: create `slog.JSONHandler` (default) or `slog.TextHandler` (when `cfg.LogFormat = "text"`), both writing to `os.Stderr` only.
- Apply log level via `slog.HandlerOptions.Level`.
- Decorate logger with default attributes: `"service.name"` from `cfg.ServiceName` (default: `"shark-task-manager"`).
- Call `slog.SetDefault(logger)` to install globally.
- Implement `parseLogLevel(level string) slog.Level`.

Public API:
```go
// InitLogger configures the global slog default logger based on cfg.
// All output is directed to os.Stderr exclusively.
// When cfg.Enabled is false, installs a discard handler.
func InitLogger(cfg config.ObservabilityConfig)
```

Internal functions (unexported):
```go
func parseLogLevel(level string) slog.Level
```

#### `internal/observability/noop.go`

Responsibilities:
- Implement `NoopProvider() ShutdownFunc`.
- Call `otel.SetTracerProvider(noop.NewTracerProvider())` to install a no-op globally.
- Call `otel.SetMeterProvider(noop_metric.NewMeterProvider())` to install a no-op meter globally.
- Return a `ShutdownFunc` that returns `nil` unconditionally.

This function is safe to call multiple times. It is the default state for tests.

Public API:
```go
// NoopProvider installs no-op OTel providers globally and returns a no-op ShutdownFunc.
// Used when observability.enabled is false or in test environments.
// Safe to call multiple times.
func NoopProvider() ShutdownFunc
```

#### `internal/observability/metrics.go`

Responsibilities:
- Define `CommandMetrics` struct with private instrument fields: `duration metric.Float64Histogram`, `invocations metric.Int64Counter`, `errors metric.Int64Counter`.
- Implement `NewCommandMetrics(meter metric.Meter) (CommandMetrics, error)`.
- Implement `(m CommandMetrics) RecordDuration(ctx context.Context, command string, duration time.Duration, err error)`.
- Implement `(m CommandMetrics) RecordInvocation(ctx context.Context, command string, err error)`.
- Define `DBMetrics` struct with: `queryDuration metric.Float64Histogram`, `queryErrors metric.Int64Counter`.
- Implement `NewDBMetrics(meter metric.Meter) (DBMetrics, error)`.
- Implement `(m DBMetrics) RecordQueryDuration(ctx context.Context, operation, table string, duration time.Duration, err error)`.

Metric instrument names (matching architecture ADR):

| Instrument | Name | Type | Unit | Attributes |
|------------|------|------|------|------------|
| `CommandMetrics.duration` | `shark.cli.command.duration` | `Float64Histogram` | `ms` | `command`, `status` |
| `CommandMetrics.invocations` | `shark.cli.command.invocations` | `Int64Counter` | — | `command`, `status` |
| `CommandMetrics.errors` | `shark.cli.command.errors` | `Int64Counter` | — | `command`, `error_type` |
| `DBMetrics.queryDuration` | `shark.db.query.duration` | `Float64Histogram` | `ms` | `operation`, `table` |
| `DBMetrics.queryErrors` | `shark.db.query.errors` | `Int64Counter` | — | `operation`, `table` |

Attribute values for `status`: `"ok"` when `err == nil`, `"error"` when `err != nil`.

Public API:
```go
type CommandMetrics struct { /* private fields */ }

func NewCommandMetrics(meter metric.Meter) (CommandMetrics, error)
func (m CommandMetrics) RecordDuration(ctx context.Context, command string, duration time.Duration, err error)
func (m CommandMetrics) RecordInvocation(ctx context.Context, command string, err error)

type DBMetrics struct { /* private fields */ }

func NewDBMetrics(meter metric.Meter) (DBMetrics, error)
func (m DBMetrics) RecordQueryDuration(ctx context.Context, operation, table string, duration time.Duration, err error)
```

#### Config Schema Extension: `internal/config/config.go`

Add the following struct definition (placed after the `Config` struct, before any `statusMetadata` methods):

```go
// ObservabilityConfig holds configuration for the observability subsystem.
// All fields have sensible defaults; the zero value means "disabled".
// When Enabled is false, no OTel SDK is initialized and no network connections
// are made. Existing users without this key in their config are unaffected.
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

Add one field to the `Config` struct immediately before the `RawData` field:

```go
Observability ObservabilityConfig `json:"observability,omitempty"`
```

#### Config Parsing Extension: `internal/config/manager.go`

Add the following block in the `Load()` function, after all existing field parsing and before the `m.config = config` assignment (approximately after line 86):

```go
// Parse observability config if present
if obsRaw, ok := rawData["observability"].(map[string]interface{}); ok {
    var obs ObservabilityConfig
    if enabled, ok := obsRaw["enabled"].(bool); ok {
        obs.Enabled = enabled
    }
    if tracingEnabled, ok := obsRaw["tracing_enabled"].(bool); ok {
        obs.TracingEnabled = tracingEnabled
    }
    if metricsEnabled, ok := obsRaw["metrics_enabled"].(bool); ok {
        obs.MetricsEnabled = metricsEnabled
    }
    if logLevel, ok := obsRaw["log_level"].(string); ok {
        obs.LogLevel = logLevel
    }
    if logFormat, ok := obsRaw["log_format"].(string); ok {
        obs.LogFormat = logFormat
    }
    if exporter, ok := obsRaw["exporter"].(string); ok {
        obs.Exporter = exporter
    }
    if otlpEndpoint, ok := obsRaw["otlp_endpoint"].(string); ok {
        obs.OTLPEndpoint = otlpEndpoint
    }
    if otlpProtocol, ok := obsRaw["otlp_protocol"].(string); ok {
        obs.OTLPProtocol = otlpProtocol
    }
    if serviceName, ok := obsRaw["service_name"].(string); ok {
        obs.ServiceName = serviceName
    }
    config.Observability = obs
}
```

This follows the exact same pattern as every other field already parsed in `Load()`. No changes to the `Load()` function signature or any other behavior.

### 3.3 Config Schema

`.sharkconfig.json` extension (all fields optional; entire block optional):

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

Field definitions:

| Field | Type | Default (zero value) | Description |
|-------|------|---------------------|-------------|
| `enabled` | `bool` | `false` | Master switch. When false, all telemetry is off (no-op providers). |
| `tracing_enabled` | `bool` | `false` | Enable tracing (only effective when `enabled=true`). |
| `metrics_enabled` | `bool` | `false` | Enable metrics (only effective when `enabled=true`). |
| `log_level` | `string` | `""` (treated as `"info"`) | Minimum log level: `"debug"`, `"info"`, `"warn"`, `"error"`. |
| `log_format` | `string` | `""` (treated as `"json"`) | Log output format: `"json"` or `"text"`. |
| `exporter` | `string` | `""` (treated as `"stdout"`) | Exporter type: `"stdout"` (development) or `"otlp"` (production). |
| `otlp_endpoint` | `string` | `""` | OTLP collector endpoint. Only used when `exporter="otlp"`. |
| `otlp_protocol` | `string` | `""` (treated as `"grpc"`) | OTLP transport: `"grpc"` or `"http"`. |
| `service_name` | `string` | `""` (treated as `"shark-task-manager"`) | OTel service name resource attribute. |

### 3.4 Public API Surface

The complete exported API for `internal/observability/`:

```go
package observability

// --- Types ---

// ShutdownFunc gracefully shuts down the OTel provider,
// flushing any pending spans or metrics.
// Must be called before process exit when observability is enabled.
// Safe to call when observability is disabled (returns nil immediately).
type ShutdownFunc func(ctx context.Context) error

// CommandMetrics holds OTel metric instruments for CLI command tracking.
// Obtain via NewCommandMetrics(). Zero value is safe to use (all methods are no-ops).
type CommandMetrics struct { /* unexported fields */ }

// DBMetrics holds OTel metric instruments for database query tracking.
// Obtain via NewDBMetrics(). Zero value is safe to use (all methods are no-ops).
type DBMetrics struct { /* unexported fields */ }

// --- Provider Functions ---

// InitProvider initializes the OTel TracerProvider and MeterProvider.
// When cfg.Enabled is false, equivalent to NoopProvider().
// Must be called once during application startup (before any span/metric creation).
// Returns a ShutdownFunc that must be deferred or called before process exit.
func InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error)

// NoopProvider installs no-op OTel providers globally and returns a no-op ShutdownFunc.
// Safe to call multiple times. Used by tests and when Enabled=false.
func NoopProvider() ShutdownFunc

// --- Logger Function ---

// InitLogger configures the global slog.Default() logger.
// All output goes to os.Stderr. When cfg.Enabled is false, installs a discard handler.
// Must be called once during startup before any slog.* calls.
func InitLogger(cfg config.ObservabilityConfig)

// --- Metric Constructor Functions ---

// NewCommandMetrics creates CLI command metric instruments against the given meter.
// Returns an error if any instrument registration fails.
func NewCommandMetrics(meter metric.Meter) (CommandMetrics, error)

// NewDBMetrics creates database query metric instruments against the given meter.
// Returns an error if any instrument registration fails.
func NewDBMetrics(meter metric.Meter) (DBMetrics, error)

// --- Metric Recording Methods ---

// RecordDuration records the execution duration of a CLI command.
// command is the cobra command use string (e.g., "task get").
// err is nil on success; non-nil sets the status attribute to "error".
func (m CommandMetrics) RecordDuration(ctx context.Context, command string, duration time.Duration, err error)

// RecordInvocation increments the command invocation counter.
func (m CommandMetrics) RecordInvocation(ctx context.Context, command string, err error)

// RecordQueryDuration records the duration of a database operation.
// operation is the SQL verb (e.g., "SELECT", "INSERT"); table is the target table name.
func (m DBMetrics) RecordQueryDuration(ctx context.Context, operation, table string, duration time.Duration, err error)
```

### 3.5 OTel Go SDK Dependencies

All dependencies are net-new additions to `go.mod`. Zero OTel entries exist today.

Install commands (run in order):

```bash
# Core OTel API (interfaces, no SDK)
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/otel/metric@latest
go get go.opentelemetry.io/otel/attribute@latest
go get go.opentelemetry.io/otel/propagation@latest

# OTel SDK (TracerProvider, MeterProvider, BatchSpanProcessor)
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/sdk/trace@latest
go get go.opentelemetry.io/otel/sdk/metric@latest
go get go.opentelemetry.io/otel/sdk/resource@latest

# Exporters
go get go.opentelemetry.io/otel/exporters/stdout/stdouttrace@latest
go get go.opentelemetry.io/otel/exporters/stdout/stdoutmetric@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@latest

go mod tidy
```

Deferred to F07 (HTTP Server Instrumentation) — do NOT add in F01:
```
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

Not required (explicitly excluded per ADR):
- `go.opentelemetry.io/otel/log` (slog bridge)
- `go.opentelemetry.io/otel/exporters/zipkin`
- `go.opentelemetry.io/otel/exporters/prometheus`

Recommended versions: `go.opentelemetry.io/otel v1.34.0` or latest v1.x stable. All sub-packages must use matching versions.

### 3.6 Span Processor Selection (ADR-6)

| Exporter | Span Processor | Rationale |
|----------|---------------|-----------|
| `stdout` | `sdktrace.NewSimpleSpanProcessor` | Synchronous; stdout writes are fast; useful for development debugging. |
| `otlp` | `sdktrace.NewBatchSpanProcessor` | Async; batched network I/O; shutdown with 5s context timeout flushes pending spans. |

---

## 4. Integration Points with Existing Code

### 4.1 Files Modified by F01

| File | Change | Scope |
|------|--------|-------|
| `internal/config/config.go` | Add `ObservabilityConfig` struct + `Observability` field to `Config` | 12 lines added |
| `internal/config/manager.go` | Add `rawData["observability"]` parsing block in `Load()` | 22 lines added |

### 4.2 Files Created by F01

| File | Purpose |
|------|---------|
| `internal/observability/provider.go` | OTel provider initialization and shutdown |
| `internal/observability/logger.go` | Global slog logger configuration |
| `internal/observability/noop.go` | No-op provider installation |
| `internal/observability/metrics.go` | Metric instrument helpers |
| `internal/observability/provider_test.go` | Unit tests for provider |
| `internal/observability/logger_test.go` | Unit tests for logger |
| `internal/observability/noop_test.go` | Unit tests for noop |
| `internal/observability/metrics_test.go` | Unit tests for metrics |

### 4.3 Files NOT Modified by F01

The following files will be modified by later features but are explicitly out of scope for F01:

| File | Modified By |
|------|------------|
| `internal/cli/root.go` | F02 (OTel init in PreRunE / shutdown in PostRunE) |
| `internal/cli/services_global.go` | F02 (add ResetObservability) |
| `internal/cli/observability_global.go` | F02 (new file: global singleton) |
| `internal/services/*.go` | F04 (constructor tracer injection) |
| `internal/repository/*.go` | F05 (package-level tracer) |
| All `log.*` call sites (23 files) | F03 (structured logging migration) |
| `cmd/server/main.go` | F07 (HTTP server instrumentation) |

### 4.4 Contract with F02

F01 defines the types and function signatures that F02 will call. F02 must not be implemented before F01 is merged. The critical contracts:

- `observability.InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error)` — F02 calls this in `PersistentPreRunE`.
- `observability.InitLogger(cfg config.ObservabilityConfig)` — F02 calls this in `PersistentPreRunE`.
- `observability.NoopProvider() ShutdownFunc` — F02 uses this for test reset (`ResetObservability()`).
- `observability.ShutdownFunc` type — F02 stores the returned value and calls it in `PersistentPostRunE`.

### 4.5 Dependency Graph (Strict)

```
internal/cli/
    imports: internal/observability/, internal/config/
                  |
                  v
internal/observability/
    imports: internal/config/, go.opentelemetry.io/otel/*
                  |
                  v
internal/config/
    imports: internal/db/ (for DatabaseConfig)

FORBIDDEN (import cycle prevention):
    internal/observability/ MUST NOT import internal/cli/
    internal/observability/ MUST NOT import internal/services/
    internal/observability/ MUST NOT import internal/repository/
    internal/config/ MUST NOT import internal/observability/
```

---

## 5. Testing Strategy

### 5.1 Guiding Principles

- All tests in `internal/observability/` are unit tests. No database, no file I/O (except stderr capture), no network calls (OTLP exporter uses lazy connection so init does not fail without a collector).
- Because OTel uses package-level global state (`otel.SetTracerProvider`), tests must call `NoopProvider()` as cleanup after any test that sets a real provider.
- Tests run with `go test -race ./internal/observability/...` to catch data races during concurrent provider initialization.

### 5.2 Test File: `internal/observability/provider_test.go`

```
TestInitProvider_DisabledReturnsNoop
    - cfg.Enabled = false
    - Assert: returns non-nil ShutdownFunc, no error
    - Assert: otel.Tracer("test") does not panic
    - Assert: shutdown returns nil

TestInitProvider_StdoutExporter
    - cfg.Enabled = true, cfg.Exporter = "stdout", cfg.ServiceName = "test-service"
    - Assert: returns non-nil ShutdownFunc, no error
    - Assert: shutdown returns nil (flushes cleanly)
    - Cleanup: call NoopProvider() to reset global state

TestInitProvider_OTLPExporter_LazyConnect
    - cfg.Enabled = true, cfg.Exporter = "otlp", cfg.OTLPEndpoint = "localhost:4317"
    - No collector running — connection must be lazy
    - Assert: InitProvider returns no error (gRPC connects on first export attempt)
    - Assert: shutdown returns nil
    - Cleanup: call NoopProvider()

TestInitProvider_StdoutNotContaminated
    - Capture os.Stdout before and after InitProvider call with stdout exporter
    - Assert: zero bytes written to stdout buffer
    - Cleanup: call NoopProvider()

TestApplyEnvOverrides
    - Set SHARK_OTEL_ENABLED=true in env
    - Start with cfg.Enabled = false
    - Call InitProvider
    - Assert: cfg.Enabled is true after override
    - Also test: OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME, SHARK_LOG_LEVEL
```

### 5.3 Test File: `internal/observability/logger_test.go`

```
TestInitLogger_DisabledDiscards
    - cfg.Enabled = false
    - Capture stderr
    - Call slog.Info("should be discarded")
    - Assert: zero bytes in captured stderr

TestInitLogger_JSONFormat
    - cfg.Enabled = true, cfg.LogFormat = "json"
    - Capture stderr
    - Call slog.Info("test", "key", "val")
    - Assert: captured bytes are valid JSON
    - Assert: JSON contains "msg" key with "test"

TestInitLogger_TextFormat
    - cfg.Enabled = true, cfg.LogFormat = "text"
    - Capture stderr
    - Call slog.Info("test msg")
    - Assert: captured bytes contain "test msg" in text format (not JSON)

TestInitLogger_LevelFiltering
    - cfg.Enabled = true, cfg.LogLevel = "warn"
    - Capture stderr
    - Call slog.Info("should be filtered")
    - Assert: zero bytes in captured stderr
    - Call slog.Warn("should appear")
    - Assert: non-zero bytes in captured stderr

TestInitLogger_ServiceNameAttribute
    - cfg.Enabled = true, cfg.LogFormat = "json", cfg.ServiceName = "myservice"
    - Capture stderr, call slog.Info("msg")
    - Parse captured JSON, assert "service.name" key equals "myservice"

TestInitLogger_DefaultServiceName
    - cfg.Enabled = true, cfg.ServiceName = "" (zero value)
    - Capture stderr, call slog.Info("msg")
    - Parse JSON, assert "service.name" equals "shark-task-manager"

TestInitLogger_StdoutNotContaminated
    - Capture os.Stdout
    - cfg.Enabled = true, cfg.LogFormat = "json"
    - Call slog.Info("test")
    - Assert: zero bytes in stdout buffer
```

### 5.4 Test File: `internal/observability/noop_test.go`

```
TestNoopProvider_ReturnsShutdownFunc
    - Call NoopProvider()
    - Assert: returned ShutdownFunc is non-nil
    - Assert: calling ShutdownFunc returns nil

TestNoopProvider_GlobalTracerDoesNotPanic
    - Call NoopProvider()
    - otel.Tracer("test").Start(ctx, "span")
    - Assert: no panic

TestNoopProvider_IdempotentCallsSafe
    - Call NoopProvider() three times in succession
    - Assert: no panics, no errors
```

### 5.5 Test File: `internal/observability/metrics_test.go`

```
TestNewCommandMetrics_Success
    - Create a no-op meter from a no-op MeterProvider
    - Call NewCommandMetrics(meter)
    - Assert: no error

TestCommandMetrics_RecordDuration_Success
    - Create CommandMetrics from no-op meter
    - Call RecordDuration(ctx, "task get", 50ms, nil)
    - Assert: no panic, no error

TestCommandMetrics_RecordDuration_ErrorStatus
    - Create CommandMetrics from no-op meter
    - Call RecordDuration(ctx, "task get", 50ms, errors.New("fail"))
    - Assert: no panic, no error

TestNewDBMetrics_Success
    - Create a no-op meter
    - Call NewDBMetrics(meter)
    - Assert: no error

TestDBMetrics_RecordQueryDuration_Success
    - Create DBMetrics from no-op meter
    - Call RecordQueryDuration(ctx, "SELECT", "tasks", 5ms, nil)
    - Assert: no panic, no error
```

### 5.6 Config Extension Tests (`internal/config/`)

The existing config tests must continue to pass. Add new tests in the config package:

```
TestConfigLoad_ObservabilityAbsent
    - Write a minimal .sharkconfig.json without "observability" key
    - Load config
    - Assert: Config.Observability == ObservabilityConfig{} (zero value)
    - Assert: Config.Observability.Enabled == false

TestConfigLoad_ObservabilityPresent
    - Write config with {"observability": {"enabled": true, "log_format": "json", "log_level": "debug"}}
    - Load config
    - Assert: Enabled=true, LogFormat="json", LogLevel="debug"
    - Assert: all other fields are zero value

TestConfigLoad_ObservabilityAllFields
    - Write config with all nine observability fields populated
    - Load config
    - Assert all nine fields match expected values

TestObservabilityConfig_OmitEmpty
    - Marshal Config{} to JSON
    - Assert: JSON output does not contain "observability" key (omitempty)
```

---

## 6. Acceptance Criteria

All criteria must be verified before F01 is considered done.

### AC-01: Package compiles with no errors
- `go build ./internal/observability/...` exits 0
- `go vet ./internal/observability/...` exits 0
- `go build ./...` exits 0 (no regressions to callers of `internal/config/`)

### AC-02: No import cycle
- `go build ./internal/observability/...` does not report an import cycle
- `internal/observability/` does not import any `internal/cli/`, `internal/services/`, or `internal/repository/` package

### AC-03: All unit tests pass
- `go test -race ./internal/observability/...` exits 0
- `go test -count=2 -race ./internal/observability/...` exits 0 (OTel global state is properly reset between runs)
- `go test -race ./internal/config/...` exits 0 (existing config tests pass + new config tests pass)
- `make test` exits 0 (no regressions to the full test suite)

### AC-04: Stdout is never contaminated
- Test `TestInitProvider_StdoutNotContaminated` and `TestInitLogger_StdoutNotContaminated` both pass
- Manually verified: `SHARK_OTEL_ENABLED=true ./bin/shark get E01 2>/dev/null | jq .` produces valid JSON

### AC-05: Disabled path is zero-overhead
- `make build && go test -bench=BenchmarkSharkGet -count=5 ./...` shows less than 1% increase compared to baseline
- (Baseline measured on main branch before F01 is merged)

### AC-06: Config backward compatibility
- An existing `.sharkconfig.json` without the `"observability"` key loads without error
- `Config.Observability.Enabled` is `false` in this case
- `make test` passes on the config package

### AC-07: Provider init does not fail without OTLP collector
- Calling `InitProvider` with `cfg.Exporter = "otlp"` and `cfg.OTLPEndpoint = "localhost:4317"` with no collector running returns no error
- The gRPC connection is lazy (connects on first export, not at init)

### AC-08: Shutdown is idempotent and safe
- Calling the returned `ShutdownFunc` twice does not panic and returns nil on the second call
- Test `TestNoopProvider_IdempotentCallsSafe` passes

### AC-09: Environment variable overrides work
- `SHARK_OTEL_ENABLED=true` overrides `cfg.Enabled = false`
- `OTEL_SERVICE_NAME=myservice` overrides `cfg.ServiceName`
- All six env var overrides documented in REQ-F-006 are tested

### AC-10: Binary size within budget
- `du -sh ./bin/shark` after `make build` shows no more than 15MB increase over baseline

### AC-11: Code quality gates pass
- `make fmt` produces no diff
- `make lint` exits 0
- All new exported symbols have godoc comments

---

## 7. Out of Scope for F01

The following items are explicitly deferred to later features. They must not be implemented in F01 even if the implementation seems natural or easy.

| Item | Deferred To |
|------|------------|
| Wiring `observability.Init()` into `internal/cli/root.go` | F02 |
| Creating `internal/cli/observability_global.go` | F02 |
| Adding `ResetObservability()` to `ResetServices()` | F02 |
| Adding `trace.Tracer` to service constructors | F04 |
| Adding package-level tracers to repositories | F05 |
| Recording CLI command duration metrics in `PostRunE` | F06 |
| HTTP server OTel initialization | F07 |
| Replacing `log.Print*` calls with `slog.*` | F03 |
| Replacing `fmt.Fprintf(os.Stderr, ...)` with `slog.*` | F03 |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | F07 |

---

*Last Updated: 2026-03-22*
