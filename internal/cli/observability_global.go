package cli

import (
	"context"
	"sync"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	// globalObsShutdown stores the shutdown function returned by InitProvider.
	globalObsShutdown observability.ShutdownFunc

	// obsInitOnce ensures observability is initialized exactly once.
	obsInitOnce sync.Once

	// obsInitErr stores initialization error for propagation.
	obsInitErr error

	// cmdMetrics holds the CLI command metrics instruments.
	// Zero value is safe (all methods are no-ops).
	cmdMetrics observability.CommandMetrics

	// cmdStartTime records when the current command started executing.
	cmdStartTime time.Time
)

// InitObservability initializes the global observability providers (logger + OTel).
// Idempotent: subsequent calls are no-ops (sync.Once guard).
// Non-fatal: if initialization fails, no-op providers are used and the error is returned
// so the caller can log a warning without aborting the command.
//
// Called from root.go PersistentPreRunE after initConfig().
func InitObservability(cfg config.ObservabilityConfig) error {
	obsInitOnce.Do(func() {
		// Resolve env overrides once so both logger and provider see the same config
		observability.ApplyEnvOverrides(&cfg)

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
// For testing only -- DO NOT use in production code.
// Called from ResetServices() to ensure test isolation.
func ResetObservability() {
	if globalObsShutdown != nil {
		_ = globalObsShutdown(context.Background())
	}
	globalObsShutdown = nil
	obsInitErr = nil
	obsInitOnce = sync.Once{}
	// Reset command metrics state
	cmdMetrics = observability.CommandMetrics{}
	cmdStartTime = time.Time{}
	// Re-install noop providers so OTel global state is clean for next test
	_ = observability.NoopProvider()
}

// GetTracer returns a named tracer from the global OTel TracerProvider.
// If called before InitObservability, returns a noop tracer (OTel global default).
// Used by services_global.go to wire tracers into service constructors (F04).
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// GetMeter returns a named meter from the global OTel MeterProvider.
// If called before InitObservability, returns a noop meter (OTel global default).
// Used by services_global.go to wire meters into service constructors (F06).
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}

// InitCommandMetrics creates the CommandMetrics instruments using the "shark.cli" meter.
// Called from PersistentPreRunE after InitObservability. Stores the start time for
// duration calculation in PersistentPostRunE. Errors are non-fatal; on failure the
// zero-value CommandMetrics (all no-ops) is used.
func InitCommandMetrics() {
	meter := GetMeter("shark.cli")
	m, err := observability.NewCommandMetrics(meter)
	if err != nil {
		// Non-fatal: zero-value CommandMetrics is safe (all methods are no-ops)
		return
	}
	cmdMetrics = m
	cmdStartTime = time.Now()
}

// RecordCommandMetrics records invocation count and duration for the completed command.
// cmdName is the Cobra command path (e.g., "task get"). cmdErr is the error returned
// by the command's RunE (nil on success). Must be called BEFORE ShutdownObservability
// so metrics can be flushed.
func RecordCommandMetrics(ctx context.Context, cmdName string, cmdErr error) {
	duration := time.Since(cmdStartTime)
	cmdMetrics.RecordDuration(ctx, cmdName, duration, cmdErr)
	cmdMetrics.RecordInvocation(ctx, cmdName, cmdErr)
}

// GetCommandMetrics returns the current CommandMetrics instance.
// Returns the zero value if InitCommandMetrics was not called or failed.
func GetCommandMetrics() observability.CommandMetrics {
	return cmdMetrics
}
