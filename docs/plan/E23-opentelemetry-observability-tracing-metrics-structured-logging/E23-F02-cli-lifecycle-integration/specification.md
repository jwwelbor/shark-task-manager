# E23-F02 Specification: CLI Lifecycle Integration

**Feature Key**: E23-F02
**Epic**: E23 - OpenTelemetry Observability
**Status**: In Specification
**Date**: 2026-03-22
**Complexity**: STANDARD (score: 7/27)

---

## Context

See E23 epic PRD and [architecture doc](../../architecture.md) for full business context, ADR rationale,
and system-level design decisions. This document covers only what is incremental to F01.

**Dependency**: Requires F01 (Observability Foundation Package) to be complete. F01 provides:
- `observability.InitProvider(cfg)` → `(ShutdownFunc, error)`
- `observability.InitLogger(cfg)`
- `observability.NoopProvider()` → `ShutdownFunc`
- `config.ObservabilityConfig` struct and `cfg.GetObservability()` accessor

---

## 1. Functional Requirements

### REQ-F-001: Observability Global Singleton

**Description**: Create `internal/cli/observability_global.go` following the exact `sync.Once` singleton
pattern of `internal/cli/db_global.go`. The file provides:
- `InitObservability(cfg config.ObservabilityConfig) error` — called from PersistentPreRunE
- `ShutdownObservability(ctx context.Context) error` — called from PersistentPostRunE
- `ResetObservability()` — test-only teardown, called from ResetServices()
- `GetTracer(name string) trace.Tracer` — returns tracer from global OTel provider

**Acceptance Criteria**:
- [ ] `InitObservability` is idempotent (sync.Once guards initialization)
- [ ] `InitObservability` stores the ShutdownFunc returned by `InitProvider`
- [ ] `InitObservability` calls both `InitProvider` and `InitLogger` with the same cfg
- [ ] `ShutdownObservability` calls the stored ShutdownFunc with a 5-second timeout context
- [ ] `ShutdownObservability` is safe to call before `InitObservability` (no-op)
- [ ] `ResetObservability` resets all globals including `sync.Once` (mirrors `ResetDB` pattern)
- [ ] `GetTracer(name)` returns `otel.Tracer(name)` from the global OTel provider

### REQ-F-002: CLI PersistentPreRunE Hook Integration

**Description**: Extend `internal/cli/root.go` `PersistentPreRunE` to initialize observability after
config is loaded. The observability config is read from the loaded viper config via
`config.LoadConfig(cfgPath).GetObservability()`.

**Acceptance Criteria**:
- [ ] `InitObservability` is called after `initConfig()` completes in PersistentPreRunE
- [ ] The ObservabilityConfig is sourced from the loaded config file (not hardcoded)
- [ ] If `InitObservability` returns an error, a warning is logged to stderr but the command proceeds
  (observability failure must not abort CLI commands)
- [ ] When `observability.enabled=false` (default), no SDK is initialized; command executes with no overhead beyond a sync.Once check

### REQ-F-003: CLI PersistentPostRunE Hook Integration

**Description**: Extend `internal/cli/root.go` `PersistentPostRunE` to shut down observability
**before** `CloseDB()`, matching the teardown ordering in the architecture doc.

**Acceptance Criteria**:
- [ ] `ShutdownObservability` is called before `CloseDB()` in PersistentPostRunE
- [ ] Shutdown uses a context with 5-second timeout to flush pending spans/metrics
- [ ] Shutdown errors are non-fatal (logged as warning when `--verbose`, silently ignored otherwise)
- [ ] The PersistentPostRunE ordering is: ShutdownObservability → CloseDB

### REQ-F-004: Test Teardown via ResetServices

**Description**: Add `ResetObservability()` to `ResetServices()` in `internal/cli/services_global.go`
so test cleanup resets observability state alongside other global service state.

**Acceptance Criteria**:
- [ ] `ResetServices()` calls `ResetObservability()` at the end of its body
- [ ] After `ResetServices()`, a subsequent `InitObservability()` call re-initializes from scratch
- [ ] Tests that call `defer cli.ResetDB()` also get observability reset via `ResetServices()`

### REQ-F-005: GetTracer Accessor for Downstream Features

**Description**: `GetTracer(name string) trace.Tracer` in `observability_global.go` provides the
wiring point for F04 (service layer tracing) and F05 (repository tracing).

**Acceptance Criteria**:
- [ ] Returns `otel.Tracer(name)` — delegates to the OTel global provider
- [ ] Safe to call before `InitObservability` (returns no-op tracer because OTel global defaults to noop)
- [ ] Available to `services_global.go` for wiring tracers into service constructors (F04 scope)

---

## 2. Non-Functional Requirements

### REQ-NF-001: Zero Performance Impact When Disabled

**Description**: When `observability.enabled=false` (the default), CLI command execution time must be
indistinguishable from pre-E23 baseline.

**Measurement**: Benchmark `shark get E01` before and after merge.
**Target**: Less than 1% overhead (see ADR-3 in architecture doc).
**Implementation**: sync.Once check is O(1); noop providers perform no allocations.

### REQ-NF-002: No stdout Contamination

**Description**: All observability output (slog, OTel exporters) must write to os.Stderr only.

**Target**: Zero bytes written to os.Stdout by observability code paths.
**Implementation**: Enforced in `observability/logger.go` and `observability/provider.go` (F01
guarantee). F02 does not add any new output paths.

### REQ-NF-003: Graceful Degradation

**Description**: Observability initialization failures (e.g., OTLP exporter cannot connect) must not
abort CLI command execution.

**Target**: Any error from `InitObservability` results in warning-level stderr output and no-op
providers being used for the remainder of the command.
**Implementation**: PersistentPreRunE catches `InitObservability` error, logs warning, continues.

---

## 3. Acceptance Criteria (Feature-Level)

**Scenario 1: Default configuration (observability disabled)**
- Given `.sharkconfig.json` has no `observability` key or `observability.enabled=false`
- When any shark command runs
- Then `InitObservability` installs no-op providers
- And command execution time is within 1% of pre-E23 baseline
- And no bytes are written to stdout or stderr from observability paths

**Scenario 2: Observability enabled with stdout exporter**
- Given `.sharkconfig.json` has `observability.enabled=true` and `observability.exporter=stdout`
- When any shark command runs
- Then slog JSON output appears on stderr (not stdout)
- And OTel span output appears on stderr (not stdout)
- And `shark get E01 --json` stdout output is unaffected

**Scenario 3: Shutdown flushes telemetry**
- Given observability is enabled
- When a command completes (PersistentPostRunE runs)
- Then `ShutdownObservability` is called before `CloseDB`
- And a 5-second context timeout is used for flushing
- And the database is closed after observability shutdown completes

**Scenario 4: Test isolation**
- Given a test calls `cli.ResetServices()`
- When the next test calls `cli.InitObservability(cfg)`
- Then a fresh singleton is created (sync.Once was reset)
- And no state from the previous test leaks into the next

**Scenario 5: Init failure degrades gracefully**
- Given `observability.exporter=otlp` and no OTLP collector running
- When `InitObservability` is called
- Then the error is logged to stderr as a warning
- And the CLI command continues with no-op providers
- And the command succeeds normally

---

## 4. Out of Scope

1. **Command-level metrics recording** (F06 scope) — PersistentPreRunE/PostRunE will record
   `shark.cli.command.duration` and `shark.cli.command.invocations` metrics, but this is deferred to F06.
   F02 only wires init/shutdown; it does NOT call `CommandMetrics.RecordInvocation`.

2. **Service tracer injection** (F04 scope) — `GetTracer()` is provided by F02 but calling
   `svc.SetTracer(GetTracer("services/task"))` in service accessors is F04 work.

3. **Repository tracer package-level setup** (F05 scope).

4. **HTTP server observability** (F07 scope).

5. **Structured logging migration** (F03 scope) — existing `log.Print*` calls in `root.go` are not
   replaced in this feature.

---

## 5. Architecture

### 5.1 Files to Create

#### `internal/cli/observability_global.go` (NEW)

```go
package cli

import (
    "context"
    "sync"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/observability"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

var (
    // globalObsShutdown stores the shutdown function returned by InitProvider.
    globalObsShutdown observability.ShutdownFunc

    // obsInitOnce ensures observability is initialized exactly once.
    obsInitOnce sync.Once

    // obsInitErr stores initialization error for propagation.
    obsInitErr error
)

// InitObservability initializes the global observability providers (logger + OTel).
// Idempotent: subsequent calls are no-ops (sync.Once guard).
// Non-fatal: if initialization fails, no-op providers are used and the error is returned
// so the caller can log a warning without aborting the command.
//
// Called from root.go PersistentPreRunE after initConfig().
func InitObservability(cfg config.ObservabilityConfig) error {
    obsInitOnce.Do(func() {
        // Always initialize logger first (even on failure, we need a logger)
        observability.InitLogger(cfg)

        shutdown, err := observability.InitProvider(cfg)
        if err != nil {
            // Fall back to noop; record error for caller
            globalObsShutdown = observability.NoopProvider()
            obsInitErr = err
            return
        }
        globalObsShutdown = shutdown
    })
    return obsInitErr
}

// ShutdownObservability flushes pending spans/metrics and shuts down OTel providers.
// Uses a 5-second timeout context to ensure flush completes before process exit.
// Safe to call before InitObservability (no-op if not initialized).
//
// Called from root.go PersistentPostRunE before CloseDB().
func ShutdownObservability() error {
    if globalObsShutdown == nil {
        return nil
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return globalObsShutdown(ctx)
}

// ResetObservability clears the global observability state.
// For testing only — DO NOT use in production code.
// Called from ResetServices() to ensure test isolation.
func ResetObservability() {
    if globalObsShutdown != nil {
        _ = globalObsShutdown(context.Background())
    }
    globalObsShutdown = nil
    obsInitErr = nil
    obsInitOnce = sync.Once{}
    // Re-install noop providers so OTel global state is clean for next test
    _ = observability.NoopProvider()
}

// GetTracer returns a named tracer from the global OTel TracerProvider.
// If called before InitObservability, returns a noop tracer (OTel global default).
// Used by services_global.go to wire tracers into service constructors (F04).
func GetTracer(name string) trace.Tracer {
    return otel.Tracer(name)
}
```

### 5.2 Files to Modify

#### `internal/cli/root.go` — PersistentPreRunE extension

Add observability initialization after `initConfig()`. Config loading requires knowing the config
path, which is available via `GetConfigPath()` already called in the hook.

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // Initialize configuration [EXISTING]
    if err := initConfig(); err != nil {
        return fmt.Errorf("failed to initialize config: %w", err)
    }

    // Configure template directory from .sharkconfig.json [EXISTING]
    cfgPath, cfgErr := GetConfigPath()
    if cfgErr == nil {
        templates.SetConfiguredTemplateDir(config.GetTemplateDirectoryFromConfig(cfgPath))
    }

    // [NEW] Initialize observability (non-fatal on error)
    obsCfg := loadObservabilityConfig(cfgPath)
    if err := InitObservability(obsCfg); err != nil {
        // Log warning to stderr — observability failure must not abort CLI commands
        fmt.Fprintf(os.Stderr, "warning: observability init failed: %v\n", err)
    }

    // Disable color output if requested [EXISTING]
    if GlobalConfig.NoColor {
        pterm.DisableColor()
    }

    // Set verbose logging if requested [EXISTING]
    if GlobalConfig.Verbose {
        pterm.EnableDebugMessages()
    }

    return nil
},
```

The helper `loadObservabilityConfig` reads from already-loaded viper config:

```go
// loadObservabilityConfig reads ObservabilityConfig from the loaded config file.
// Returns zero-value config (disabled) if config is unavailable.
func loadObservabilityConfig(cfgPath string) config.ObservabilityConfig {
    if cfgPath == "" {
        return config.ObservabilityConfig{}
    }
    cfg, err := config.LoadConfig(cfgPath)
    if err != nil {
        return config.ObservabilityConfig{}
    }
    return cfg.GetObservability()
}
```

#### `internal/cli/root.go` — PersistentPostRunE extension

```go
PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
    // [NEW] Shutdown observability first (flushes spans/metrics before DB close)
    if err := ShutdownObservability(); err != nil {
        if GlobalConfig.Verbose {
            fmt.Fprintf(os.Stderr, "warning: observability shutdown failed: %v\n", err)
        }
    }

    // Close database connection if it was opened [EXISTING]
    if err := CloseDB(); err != nil {
        if GlobalConfig.Verbose {
            pterm.Warning.Printf("Failed to close database: %v\n", err)
        }
    }
    return nil
},
```

#### `internal/cli/services_global.go` — ResetServices extension

Add `ResetObservability()` at the end of `ResetServices()`:

```go
// ResetServices clears global service state. For testing only.
func ResetServices() {
    // ... existing resets (globalActionService, globalNoteService, etc.) ...

    // Reset observability state for test isolation [NEW]
    ResetObservability()
}
```

### 5.3 Config Loading Integration

The observability config is read via `config.LoadConfig(cfgPath)`. This function already exists in
`internal/config/config.go`. The `Config.GetObservability()` method (also in config.go) returns a
zero-value `ObservabilityConfig` (disabled) when the field is nil, ensuring backward compatibility.

**No new imports** are needed in `root.go` beyond `os` (already present) for `fmt.Fprintf`.
The `observability_global.go` file imports:
- `github.com/jwwelbor/shark-task-manager/internal/config` (existing)
- `github.com/jwwelbor/shark-task-manager/internal/observability` (new, F01 provided)
- `go.opentelemetry.io/otel` (new, already in go.mod from F01)
- `go.opentelemetry.io/otel/trace` (new, already in go.mod from F01)

### 5.4 Import Graph (No Circular Dependencies)

```
internal/cli/root.go
    |
    +-> internal/cli/observability_global.go
            |
            +-> internal/observability/  (F01 package)
            |       +-> internal/config/
            +-> go.opentelemetry.io/otel
```

No new circular dependency risk. `internal/observability/` never imports `internal/cli/`.

### 5.5 Test Isolation Pattern

Tests that use global CLI state must call `ResetServices()` (which now includes `ResetObservability()`)
in their cleanup:

```go
func TestSomeCLICommand(t *testing.T) {
    defer cli.ResetDB()
    defer cli.ResetServices() // now also resets observability

    // test body
}
```

Because `ResetObservability()` re-installs noop providers via `observability.NoopProvider()`, the
OTel global state is clean for the next test.

---

## 6. Key Technical Decisions

### Decision 1: Non-Fatal Observability Initialization

Observability errors (OTLP connection refused, bad config) are never propagated as command failures.
This matches ADR-3 (disabled by default) and ensures zero user-impact for the 99% who don't configure
observability. **Pattern**: catch error from `InitObservability`, log to stderr, continue.

### Decision 2: Shutdown Before CloseDB

Spans may reference database operation durations. Shutting down OTel before closing the DB ensures
all spans that reference DB operations have a chance to complete and flush. This follows the teardown
order specified in the architecture doc Section 4.1.

### Decision 3: sync.Once Pattern from db_global.go

Exact copy of the `dbInitOnce / dbInitErr / globalDB` pattern. Benefits: thread-safe, lazy,
idempotent. Reset pattern (assign new `sync.Once{}`) matches `ResetDB()` exactly.

### Decision 4: loadObservabilityConfig Helper

Rather than adding observability config reading inline in PersistentPreRunE, a small helper function
keeps the hook readable. This follows the existing `initConfig()` helper pattern in root.go.

---

## 7. Traceability

| Requirement | Epic PRD Section | Architecture Doc Section |
|-------------|-----------------|--------------------------|
| REQ-F-001 | PRD §3 (Foundation) | §4.1 CLI Lifecycle, §5 Package Structure |
| REQ-F-002 | PRD §3 (Foundation) | §4.1 CLI Lifecycle |
| REQ-F-003 | PRD §3 (Foundation) | §4.1 CLI Lifecycle, ADR-6 |
| REQ-F-004 | PRD §6 (Testing) | §9 Risk Mitigations (Test isolation) |
| REQ-F-005 | PRD §4 (Tracing) | §4.2 Service Layer Integration |
| REQ-NF-001 | PRD §5 (Performance) | ADR-3 |
| REQ-NF-002 | PRD §5 (Performance) | ADR-5 |
| REQ-NF-003 | PRD §5 (Performance) | ADR-3 |

---

*Last Updated: 2026-03-22*
