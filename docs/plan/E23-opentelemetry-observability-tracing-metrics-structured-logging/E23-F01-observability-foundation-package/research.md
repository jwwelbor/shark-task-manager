# E23-F01 Observability Foundation Package - Feature Research Report

**Date:** 2026-03-22
**Feature:** E23-F01 - Observability Foundation Package
**Researcher:** Researcher Agent

---

## Executive Summary

E23-F01 creates the `internal/observability/` package from scratch — six new Go files with zero modifications to existing code. The entire feature is net-new, additive, and non-breaking. The `internal/config/config.go` `Config` struct receives one new field (`Observability ObservabilityConfig`) and a companion struct. The config manager's `Load()` method requires one new `rawData` field parse block following the exact pattern of every other field already parsed there. All integration points (lifecycle hooks, global singleton pattern, service constructor injection, test reset) are already present in the codebase and precisely documented here.

---

## 1. Config Loading Pattern — Exact Implementation

### File: `internal/config/config.go`

**Location:** `/home/jwwel/projects/shark-task-manager/internal/config/config.go`

The `Config` struct is defined at **line 14**. The complete struct as it currently stands:

```go
type Config struct {
    LastSyncTime           *time.Time             `json:"last_sync_time,omitempty"`
    Database               *db.DatabaseConfig     `json:"database,omitempty"`
    ColorEnabled           *bool                  `json:"color_enabled,omitempty"`
    JSONOutput             *bool                  `json:"json_output,omitempty"`
    InteractiveMode        *bool                  `json:"interactive_mode,omitempty"`
    RequireRejectionReason bool                   `json:"require_rejection_reason,omitempty"`
    Viewer                 *string                `json:"viewer,omitempty"`
    TemplateDirectory      *string                `json:"template_directory,omitempty"`
    WorkflowConfig         *string                `json:"workflow_config,omitempty"`
    RawData                map[string]interface{} `json:"-"`
    statusMetadata         map[string]*StatusMetadata `json:"-"`
}
```

**Required change:** Add one field before `RawData`:
```go
Observability ObservabilityConfig `json:"observability,omitempty"`
```

The new `ObservabilityConfig` struct to add to the same file:
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

### File: `internal/config/manager.go`

**Location:** `/home/jwwel/projects/shark-task-manager/internal/config/manager.go`

The `Load()` function is at **line 28**. It parses `.sharkconfig.json` in two passes:
1. **Line 43-46**: `json.Unmarshal` into `rawData map[string]interface{}` — preserves ALL JSON fields
2. **Lines 53-86**: Individual field extraction from `rawData` using type assertions

The `ObservabilityConfig` field parsing must be added between line 86 and the `m.config = config` assignment at line 86. Pattern to follow (mirrors existing fields):

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

**Note:** Because `observability` is `omitempty` and `Enabled` defaults to `false`, existing `.sharkconfig.json` files without the `observability` key will result in `Config.Observability` being the zero value (`Enabled: false`) — meaning all telemetry stays off. No migration needed.

**Important observation:** The manager's `Load()` function currently uses the `log` package (line 58: `log.Printf("Warning: Invalid last_sync_time format...")`). This is one of the calls to be replaced in F03 (Structured Logging Migration). F01 should NOT change this call — it is out of scope for the foundation package.

---

## 2. `sync.Once` Global Singleton Pattern

### Reference File: `internal/cli/db_global.go`

**Location:** `/home/jwwel/projects/shark-task-manager/internal/cli/db_global.go`

The complete pattern (lines 1-75):

```go
package cli

import (
    "context"
    "sync"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

var (
    globalDB    *repository.DB   // the resource
    dbInitOnce  sync.Once        // initialization guard
    dbInitErr   error            // stored init error
)

// GetDB returns the global resource, initializing lazily on first call.
func GetDB(ctx context.Context) (*repository.DB, error) {
    if ctx == nil {
        ctx = context.Background()
    }
    dbInitOnce.Do(func() {
        globalDB, dbInitErr = initDatabase(ctx)
    })
    if dbInitErr != nil {
        return nil, dbInitErr
    }
    return globalDB, nil
}

// CloseDB closes the resource and resets state (called from PersistentPostRunE).
func CloseDB() error {
    if globalDB != nil {
        err := globalDB.Close()
        globalDB = nil
        dbInitErr = nil
        dbInitOnce = sync.Once{}
        return err
    }
    return nil
}

// ResetDB clears state. For testing only.
func ResetDB() {
    if globalDB != nil {
        globalDB.Close()
    }
    globalDB = nil
    dbInitErr = nil
    dbInitOnce = sync.Once{}
}
```

**For E23-F01**, the analog file is `internal/cli/observability_global.go` (created in F02, not F01). However, F01 must design the `observability.Init()` and `observability.Shutdown()` function signatures that `observability_global.go` will call, since F02 depends directly on what F01 exports.

### `ResetServices()` function

**Location:** `internal/cli/services_global.go`, **lines 461-485**

```go
func ResetServices() {
    globalActionService = nil
    actionServiceErr = nil
    actionServiceOnce = sync.Once{}

    globalNoteService = nil
    noteServiceErr = nil
    noteServiceOnce = sync.Once{}

    globalContextService = nil
    contextServiceOnce = sync.Once{}

    globalResumeService = nil
    resumeServiceErr = nil
    resumeServiceOnce = sync.Once{}

    globalRegistry = nil
    registryOnce = sync.Once{}

    globalEntityService = nil
    entityServiceOnce = sync.Once{}
}
```

**Required addition in F02** (not F01): `ResetServices()` must call `ResetObservability()` from `observability_global.go` so tests can tear down OTel global state alongside service state. F01 does not touch this file.

---

## 3. Go Module Current State

### File: `go.mod`

**Location:** `/home/jwwel/projects/shark-task-manager/go.mod`

**Go version:** `go 1.23.4` — supports `log/slog` natively (available since Go 1.21). No Go version bump needed.

**Current direct dependencies:**
```
github.com/mattn/go-sqlite3 v1.14.32
github.com/pterm/pterm v0.12.82
github.com/spf13/cobra v1.10.2
github.com/spf13/viper v1.21.0
github.com/stretchr/testify v1.11.1
github.com/tursodatabase/libsql-client-go v0.0.0-20251219100830-236aa1ff8acc
golang.org/x/term v0.32.0
golang.org/x/text v0.28.0
gopkg.in/yaml.v3 v3.0.1
```

**Zero OTel dependencies currently.** No `go.opentelemetry.io/*` entries anywhere in go.mod.

**OTel dependencies to add for E23-F01** (foundation package only, not HTTP middleware):

```
go.opentelemetry.io/otel                                              # Core API
go.opentelemetry.io/otel/trace                                        # Trace API (Span, SpanContext)
go.opentelemetry.io/otel/metric                                       # Metric API (Counter, Histogram)
go.opentelemetry.io/otel/attribute                                    # Attribute key-value pairs
go.opentelemetry.io/otel/sdk/trace                                    # TracerProvider SDK
go.opentelemetry.io/otel/sdk/metric                                   # MeterProvider SDK
go.opentelemetry.io/otel/sdk/resource                                 # Service resource
go.opentelemetry.io/otel/propagation                                  # W3C trace context
go.opentelemetry.io/otel/exporters/stdout/stdouttrace                 # Stdout trace exporter
go.opentelemetry.io/otel/exporters/stdout/stdoutmetric                # Stdout metric exporter
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc       # OTLP gRPC trace exporter
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc     # OTLP gRPC metric exporter
```

**Deferred to F07 (HTTP Server Instrumentation):**
```
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp          # HTTP middleware
```

**Recommended versions** (current stable as of 2026-03):
- `go.opentelemetry.io/otel v1.34.0` (or latest v1.x)
- `go.opentelemetry.io/otel/sdk v1.34.0`
- Matching versions for all sub-packages

**Install command:**
```bash
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/exporters/stdout/stdouttrace@latest
go get go.opentelemetry.io/otel/exporters/stdout/stdoutmetric@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@latest
```

**Binary size impact:** Adding OTel SDK will increase binary size by approximately 10-15MB due to transitive gRPC and protobuf dependencies. This is within the 15MB budget from the architecture ADR.

---

## 4. Existing Logging Patterns — Complete Inventory

### `log.*` calls in `internal/` (23 occurrences across source files)

| File | Line | Call |
|------|------|------|
| `internal/config/manager.go` | 58 | `log.Printf("Warning: Invalid last_sync_time format in config: %v", err)` |
| `internal/config/template_helpers.go` | 330 | `log.Printf("WARNING: Failed to fetch related docs for feature %s: %v", ...)` |
| `internal/config/template_helpers.go` | 343 | `log.Printf("WARNING: Failed to fetch related features for %s: %v", ...)` |
| `internal/config/template_helpers.go` | 441 | `log.Printf("WARNING: Failed to fetch related docs for task %s: %v", ...)` |
| `internal/config/template_helpers.go` | 450 | `log.Printf("WARNING: Failed to fetch related tasks for task %s: %v", ...)` |
| `internal/config/template_helpers.go` | 516 | `log.Printf("WARNING: Failed to fetch related docs for epic %s: %v", ...)` |
| `internal/config/template_helpers.go` | 529 | `log.Printf("WARNING: Failed to fetch related epics for %s: %v", ...)` |
| `internal/config/orchestrator_action.go` | 182 | `log.Printf("template rendering failed for %s: %v", ...)` |
| `internal/status/derivation.go` | 30 | `log.Println("WARN: No workflow config provided...")` |
| `internal/status/derivation.go` | 46 | `log.Printf("WARN: Status %q not found in workflow config...")` |
| `internal/status/derivation.go` | 64 | `log.Printf("WARN: Unrecognized phase %q for status %q...")` |
| `internal/services/display_service.go` | 449 | `log.Printf("WARNING: Failed to fetch enrichment data for epic %s: %v", ...)` |
| `internal/services/display_service.go` | 634 | `log.Printf("WARNING: Failed to fetch enrichment data for feature %s: %v", ...)` |
| `internal/services/display_service.go` | 668 | `log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", ...)` |
| `internal/services/entity_service.go` | 58 | `log.Printf("warning: failed to record entity history for %s: %v", ...)` |
| `internal/services/feature_service.go` | 238 | `log.Printf("WARNING: Failed to fetch enrichment data for feature %s: %v", ...)` |
| `internal/services/feature_service.go` | 750 | `log.Printf("warning: auto-reopen of epic %s failed: %v", ...)` |
| `internal/services/task_service.go` | 631 | `log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", ...)` |
| `internal/services/task_service.go` | 976 | `log.Printf("warning: auto-reopen check for feature %s failed: %v", ...)` |
| `internal/services/task_service.go` | 996 | `log.Printf("warning: auto-reopen of feature %s failed: %v", ...)` |
| `internal/services/epic_service.go` | 224 | `log.Printf("WARNING: Failed to fetch enrichment data for epic %s: %v", ...)` |

**These are ALL out of scope for F01.** Migration of these calls is F03 (Structured Logging Migration).

### `fmt.Fprintf(os.Stderr, ...)` calls in `internal/`

| File | Line | Call |
|------|------|------|
| `internal/cli/root.go` | 406 | `fmt.Fprintf(os.Stderr, "Failed to render table: %v\n", err)` |
| `internal/cli/commands/epic_helpers.go` | 379 | `fmt.Fprintf(os.Stderr, "Warning: Failed to calculate progress...")` |
| `internal/cli/commands/epic_helpers.go` | 790 | `fmt.Fprintf(os.Stderr, "Warning: Failed to update progress...")` |
| `internal/cli/commands/feature_helpers.go` | 215 | `fmt.Fprintf(os.Stderr, "Warning: Failed to batch fetch status breakdowns...")` |
| `internal/cli/commands/feature_helpers.go` | 224 | `fmt.Fprintf(os.Stderr, "Warning: Failed to get config path...")` |
| `internal/cli/commands/feature_helpers.go` | 228 | `fmt.Fprintf(os.Stderr, "Warning: Failed to load config...")` |
| `internal/cli/commands/feature_helpers.go` | 750 | `fmt.Fprintf(os.Stderr, "Warning: Failed to get task counts...")` |
| `internal/cli/commands/epic.go` | 182 | `fmt.Fprintf(os.Stderr, "Failed to list epics: %v\n", err)` |
| `internal/cli/commands/epic.go` | 245 | `fmt.Fprintf(os.Stderr, "Failed to build epic data: %v\n", err)` |
| `internal/cli/commands/helpers.go` | 820 | `fmt.Fprintf(os.Stderr, "Details: %v\n", err)` |
| `internal/config/workflow_parser.go` | 393 | `fmt.Fprintf(os.Stderr, "Warning: Failed to load workflow config: %v\n", err)` |
| `internal/db/db.go` | 1064 | `fmt.Fprintf(os.Stderr, "Warning: Failed to backup WAL file %s: %v\n", ...)` |
| `internal/services/bug_service.go` | 164 | `fmt.Fprintf(os.Stderr, "warning: failed to write bug file %s: %v\n", ...)` |
| `internal/services/change_card_service.go` | 169 | `fmt.Fprintf(os.Stderr, "warning: failed to write change-card file %s: %v\n", ...)` |
| `internal/services/change_card_service.go` | 289 | `fmt.Fprintf(os.Stderr, "warning: failed to delete change-card file %s: %v\n", ...)` |
| `internal/init/profile_service.go` | 113 | `fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)` |
| `internal/init/profile_service.go` | 127 | `fmt.Fprintf(os.Stderr, "Warning: failed to create workflow file backup: %v\n", err)` |
| `internal/taskcreation/creator.go` | 308 | `fmt.Fprintf(os.Stderr, "[task-creator] %s\n", msg)` |

**These are also out of scope for F01.** Migration is F03.

**Key constraint confirmed:** `fmt.Fprintf(os.Stderr, ...)` at `internal/cli/root.go:406` confirms the existing pattern of writing diagnostic messages to stderr — never stdout. All OTel and slog output in F01 must follow this pattern (`os.Stderr` only).

---

## 5. CLI Lifecycle Hooks

### File: `internal/cli/root.go`

**Location:** `/home/jwwel/projects/shark-task-manager/internal/cli/root.go`

**`PersistentPreRunE`** — lines 42-65:
```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // Initialize configuration
    if err := initConfig(); err != nil {
        return fmt.Errorf("failed to initialize config: %w", err)
    }

    // Configure template directory from .sharkconfig.json
    cfgPath, cfgErr := GetConfigPath()
    if cfgErr == nil {
        templates.SetConfiguredTemplateDir(config.GetTemplateDirectoryFromConfig(cfgPath))
    }

    // Disable color output if requested
    if GlobalConfig.NoColor {
        pterm.DisableColor()
    }

    // Set verbose logging if requested
    if GlobalConfig.Verbose {
        pterm.EnableDebugMessages()
    }

    return nil
},
```

**`PersistentPostRunE`** — lines 66-75:
```go
PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
    // Close database connection if it was opened
    if err := CloseDB(); err != nil {
        if GlobalConfig.Verbose {
            pterm.Warning.Printf("Failed to close database: %v\n", err)
        }
    }
    return nil
},
```

**F01 does NOT touch root.go.** These hooks are listed here because F02 will inject OTel init into `PersistentPreRunE` (after the existing code) and OTel shutdown into `PersistentPostRunE` (before `CloseDB()`). F01 must design the exported functions (`Init`, `Shutdown`) that F02 will call.

**Critical observation:** The `PersistentPostRunE` hook currently calls `CloseDB()` as the only shutdown action. When F02 adds OTel shutdown, it must be called before `CloseDB()` since span flushing may need an active database context for certain span processors. The ordering in `PersistentPostRunE` should be:
1. Record command metrics (F06)
2. `observability.Shutdown(ctx)` — flush spans/metrics (F02)
3. `CloseDB()` — close database (existing)

---

## 6. Package Structure Recommendation

### Directory

```
internal/observability/
    config.go           # ObservabilityConfig struct ONLY (or kept in internal/config/)
    provider.go         # InitProvider(cfg ObservabilityConfig) (ShutdownFunc, error)
    logger.go           # InitLogger(cfg ObservabilityConfig)
    noop.go             # NoopProvider() ShutdownFunc
    metrics.go          # CommandMetrics and DBMetrics instrument helpers
```

**Decision on config.go placement:** The `ObservabilityConfig` struct belongs in `internal/config/config.go` — NOT in `internal/observability/config.go`. This avoids an import cycle risk: `internal/observability/` imports `internal/config/` for `ObservabilityConfig`. If `ObservabilityConfig` were defined in `internal/observability/`, then `internal/config/` would need to import `internal/observability/` to use its own config type, creating a cycle.

**Dependency hierarchy (strict, no cycles):**
```
internal/cli/
    imports: internal/observability/, internal/config/
internal/observability/
    imports: internal/config/ (for ObservabilityConfig)
    imports: go.opentelemetry.io/otel/* (OTel SDK)
    NEVER imports: internal/cli/, internal/services/, internal/repository/
internal/config/
    imports: internal/db/ (for DatabaseConfig)
    NEVER imports: internal/observability/, internal/cli/
```

### File Contents Outline

#### `internal/observability/provider.go`

```go
package observability

import (
    "context"
    "fmt"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

// ShutdownFunc is a function that gracefully shuts down the OTel provider,
// flushing any pending spans or metrics.
type ShutdownFunc func(ctx context.Context) error

// InitProvider initializes the OpenTelemetry TracerProvider and MeterProvider
// based on the provided ObservabilityConfig. Returns a ShutdownFunc that must
// be called to flush and clean up resources.
//
// When cfg.Enabled is false, this is equivalent to calling NoopProvider().
// All providers are disabled and no network connections are made.
func InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error) {
    if !cfg.Enabled {
        return NoopProvider(), nil
    }
    // ... build real providers based on cfg.Exporter ...
}

// applyEnvOverrides applies OTel standard environment variable overrides
// to the config, mutating it in place.
func applyEnvOverrides(cfg *config.ObservabilityConfig) {
    if v := os.Getenv("SHARK_OTEL_ENABLED"); v == "true" {
        cfg.Enabled = true
    }
    if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
        cfg.OTLPEndpoint = v
    }
    if v := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); v != "" {
        cfg.OTLPProtocol = v
    }
    if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
        cfg.ServiceName = v
    }
    if v := os.Getenv("SHARK_LOG_LEVEL"); v != "" {
        cfg.LogLevel = v
    }
    if v := os.Getenv("SHARK_LOG_FORMAT"); v != "" {
        cfg.LogFormat = v
    }
}
```

#### `internal/observability/logger.go`

```go
package observability

import (
    "log/slog"
    "os"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

// InitLogger configures the global slog default logger based on cfg.
// Writes to os.Stderr exclusively — never stdout.
// Sets the global slog default via slog.SetDefault().
//
// When cfg.Enabled is false, sets a discard handler (level > ERROR so
// nothing is logged). This is the safe default for tests.
func InitLogger(cfg config.ObservabilityConfig) {
    if !cfg.Enabled {
        // Discard all log output when observability is disabled
        slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
            Level: slog.Level(slog.LevelError + 1),
        })))
        return
    }

    level := parseLogLevel(cfg.LogLevel)
    opts := &slog.HandlerOptions{Level: level}

    var handler slog.Handler
    if cfg.LogFormat == "text" {
        handler = slog.NewTextHandler(os.Stderr, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    }

    // Add default service attributes
    serviceName := cfg.ServiceName
    if serviceName == "" {
        serviceName = "shark-task-manager"
    }

    logger := slog.New(handler).With(
        "service.name", serviceName,
    )
    slog.SetDefault(logger)
}

func parseLogLevel(level string) slog.Level {
    switch level {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default: // "info" or empty
        return slog.LevelInfo
    }
}
```

#### `internal/observability/noop.go`

```go
package observability

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace/noop"
)

// NoopProvider configures no-op OTel providers and returns a no-op ShutdownFunc.
// Used when observability.enabled is false or in test environments.
//
// Sets the global OTel TracerProvider to noop so that otel.Tracer("pkg")
// calls in repository layer return no-op tracers without panicking.
func NoopProvider() ShutdownFunc {
    // Set no-op tracer provider globally so package-level otel.Tracer() calls work
    otel.SetTracerProvider(noop.NewTracerProvider())
    return func(ctx context.Context) error {
        return nil
    }
}
```

#### `internal/observability/metrics.go`

```go
package observability

import (
    "context"
    "time"
    "go.opentelemetry.io/otel/metric"
)

// CommandMetrics holds OTel metric instruments for CLI command tracking.
type CommandMetrics struct {
    duration    metric.Float64Histogram
    invocations metric.Int64Counter
    errors      metric.Int64Counter
}

// NewCommandMetrics creates metric instruments registered against the given meter.
func NewCommandMetrics(meter metric.Meter) (CommandMetrics, error) {
    // ... create histogram and counter instruments ...
}

// RecordDuration records the duration and outcome of a CLI command.
// command is the cobra command name (e.g. "task get"), err is nil on success.
func (m CommandMetrics) RecordDuration(ctx context.Context, command string, duration time.Duration, err error) {
    // ... record histogram observation ...
}

// DBMetrics holds OTel metric instruments for database query tracking.
type DBMetrics struct {
    queryDuration metric.Float64Histogram
    queryErrors   metric.Int64Counter
}

// NewDBMetrics creates metric instruments registered against the given meter.
func NewDBMetrics(meter metric.Meter) (DBMetrics, error) {
    // ... create histogram and counter instruments ...
}
```

---

## 7. OTel Dependencies — Exact `go get` Commands

Run these in order. Each command fetches and pins the dependency in `go.mod` / `go.sum`:

```bash
# Core OTel API (no SDK, just interfaces)
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/otel/metric@latest
go get go.opentelemetry.io/otel/attribute@latest
go get go.opentelemetry.io/otel/propagation@latest

# OTel SDK (TracerProvider, MeterProvider, BatchSpanProcessor, etc.)
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/sdk/trace@latest
go get go.opentelemetry.io/otel/sdk/metric@latest
go get go.opentelemetry.io/otel/sdk/resource@latest

# Exporters needed by F01 (stdout and OTLP/gRPC)
go get go.opentelemetry.io/otel/exporters/stdout/stdouttrace@latest
go get go.opentelemetry.io/otel/exporters/stdout/stdoutmetric@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@latest
```

**After adding dependencies:**
```bash
go mod tidy  # prune unused, add missing indirect deps
make build   # verify it compiles
make test    # verify nothing broke
```

---

## 8. Integration Points Summary Table

| Integration Point | File | Line(s) | F01 Action | Future Feature |
|------------------|------|---------|------------|---------------|
| `Config` struct extension | `internal/config/config.go` | 14-35 | Add `Observability ObservabilityConfig` field and struct | F01 |
| Config load parsing | `internal/config/manager.go` | ~86 | Add `rawData["observability"]` parse block | F01 |
| OTel provider init | `internal/cli/root.go` | 42-65 (PreRunE) | Add `observability.Init()` call after existing setup | F02 |
| OTel provider shutdown | `internal/cli/root.go` | 66-75 (PostRunE) | Add `observability.Shutdown()` before `CloseDB()` | F02 |
| Global singleton | `internal/cli/observability_global.go` | NEW FILE | Create following `db_global.go` pattern | F02 |
| Test teardown | `internal/cli/services_global.go` | 461-485 | Add `ResetObservability()` to `ResetServices()` | F02 |
| Service tracer injection | `internal/services/task_service.go` et al | Constructors | Add `tracer trace.Tracer` field + `SetTracer()` | F04 |
| Repository tracer | `internal/repository/*.go` | Package-level | Add `var repoTracer = otel.Tracer("pkg")` | F05 |
| `log.*` migration | 21 locations in `internal/` | Various | Replace with `slog.Warn/Info/Error` | F03 |
| `fmt.Fprintf(stderr)` migration | 18 locations in `internal/` | Various | Replace with `slog.Warn/Error` | F03 |
| HTTP middleware | `cmd/server/main.go` | TBD | Add `otelhttp.NewHandler(mux, ...)` | F07 |

---

## 9. Test Strategy for F01

F01 creates new code with no integration into existing entry points. Tests can be straightforward unit tests:

### `internal/observability/provider_test.go`

```go
func TestInitProvider_DisabledReturnsNoop(t *testing.T) {
    cfg := config.ObservabilityConfig{Enabled: false}
    shutdown, err := observability.InitProvider(cfg)
    assert.NoError(t, err)
    assert.NotNil(t, shutdown)

    // Verify no-op: calling otel.Tracer should not panic
    tracer := otel.Tracer("test")
    assert.NotNil(t, tracer)

    // Shutdown should not error
    assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_StdoutExporter(t *testing.T) {
    cfg := config.ObservabilityConfig{
        Enabled:     true,
        Exporter:    "stdout",
        ServiceName: "test-service",
    }
    shutdown, err := observability.InitProvider(cfg)
    assert.NoError(t, err)
    assert.NoError(t, shutdown(context.Background()))
}
```

### `internal/observability/logger_test.go`

```go
func TestInitLogger_DisabledDiscards(t *testing.T) {
    cfg := config.ObservabilityConfig{Enabled: false}
    observability.InitLogger(cfg)
    // Calling slog.Info should not panic and should be silently discarded
    slog.Info("this should be discarded")
}

func TestInitLogger_JSONFormat(t *testing.T) {
    cfg := config.ObservabilityConfig{
        Enabled:   true,
        LogFormat: "json",
        LogLevel:  "info",
    }
    observability.InitLogger(cfg)
    slog.Info("test message", "key", "value")
}
```

**Test isolation:** Since OTel uses global state (`otel.SetTracerProvider()`), tests in `internal/observability/` must reset global providers between test cases. The `noop.go`'s `NoopProvider()` can be called at the end of each test as cleanup.

---

## 10. Key Risks for F01 Implementation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Import cycle: `internal/observability/` imports `internal/cli/` | Low | High | `ObservabilityConfig` stays in `internal/config/`; `observability` package only imports `config` and OTel SDK |
| OTel global state leaking between tests | High | Medium | Every test that calls `InitProvider` must call the returned `ShutdownFunc`; `NoopProvider()` resets global provider |
| stdout contamination from OTel/slog | Medium | High | Enforce `os.Stderr` in `logger.go` via `slog.NewJSONHandler(os.Stderr, ...)` — never `os.Stdout` |
| `go mod tidy` adding unexpected transitive deps | Low | Low | Review `go.mod` diff after `go get`; pin versions explicitly |
| `ShutdownFunc` signature mismatch with F02 expectations | Medium | Medium | Define `ShutdownFunc = func(ctx context.Context) error` in F01 so F02 knows the exact type to store |

---

## References

- Epic architecture: `docs/plan/E23-opentelemetry-observability-tracing-metrics-struct/architecture.md`
- Epic research: `docs/plan/E23-opentelemetry-observability-tracing-metrics-struct/research.md`
- `sync.Once` pattern: `/home/jwwel/projects/shark-task-manager/internal/cli/db_global.go` (lines 1-75)
- Config struct: `/home/jwwel/projects/shark-task-manager/internal/config/config.go` (lines 14-35)
- Config load: `/home/jwwel/projects/shark-task-manager/internal/config/manager.go` (lines 28-88)
- CLI lifecycle hooks: `/home/jwwel/projects/shark-task-manager/internal/cli/root.go` (lines 42-75)
- ResetServices: `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` (lines 461-485)
- Go module: `/home/jwwel/projects/shark-task-manager/go.mod`
- OTel SDK docs: https://pkg.go.dev/go.opentelemetry.io/otel
