package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// obsContainer holds the observability state and its initialization state.
// Using a container struct makes ResetObservability() safe: we swap the
// entire container atomically instead of reassigning individual sync.Once
// values (which would be a data race if any goroutine is mid-initialization).
type obsContainer struct {
	shutdown     observability.ShutdownFunc
	initOnce     sync.Once
	initErr      error
	cmdMetrics   observability.CommandMetrics
	cmdStartTime time.Time
	// logFile holds the io.Closer returned by observability.InitLoggerWithRoot
	// when cfg.LogFile is set and the file was opened successfully. It is closed
	// by ShutdownObservability and ResetObservability, then set back to nil to
	// make repeat shutdown calls safe (no double-close).
	logFile io.Closer
}

// globalObsContainer is accessed only through loadObsContainer / storeObsContainer.
// Using atomic pointer operations ensures that a call to ResetObservability()
// is immediately visible to any goroutine that subsequently calls an
// observability function, without requiring a separate mutex.
//
//nolint:gochecknoglobals // Intentional package-level singleton for CLI entry points.
var globalObsContainer unsafe.Pointer // *obsContainer

// cmdStartTime is a package-level alias kept for test compatibility.
// Tests in command_metrics_test.go reference this variable directly.
//
//nolint:gochecknoglobals // Alias for test backward compatibility.
var cmdStartTime time.Time

func init() {
	storeObsContainer(new(obsContainer))
}

func loadObsContainer() *obsContainer {
	return (*obsContainer)(atomic.LoadPointer(&globalObsContainer))
}

func storeObsContainer(c *obsContainer) {
	atomic.StorePointer(&globalObsContainer, unsafe.Pointer(c))
}

// InitObservability initializes the global observability providers (logger + OTel).
// Idempotent: subsequent calls are no-ops (sync.Once guard).
// Non-fatal: if initialization fails, no-op providers are used and the error is returned
// so the caller can log a warning without aborting the command.
//
// Called from root.go PersistentPreRunE after initConfig().
func InitObservability(cfg config.ObservabilityConfig) error {
	c := loadObsContainer()
	c.initOnce.Do(func() {
		// Resolve env overrides once so both logger and provider see the same config
		observability.ApplyEnvOverrides(&cfg)

		// Resolve project root for relative log_file paths. Non-fatal: on error
		// (e.g., no .sharkconfig.json found), fall back to an empty string which
		// causes the logger to resolve relative paths against CWD.
		projectRoot, err := FindProjectRoot()
		if err != nil {
			projectRoot = ""
		}

		// Always initialize logger first (even on failure, we need a logger).
		// Capture the io.Closer so ShutdownObservability can close the log file.
		c.logFile = observability.InitLoggerWithRoot(cfg, projectRoot)

		shutdown, err := observability.InitProvider(cfg)
		if err != nil {
			// Fall back to noop; record error for caller
			c.shutdown = observability.NoopProvider()
			c.initErr = err
			return
		}
		c.shutdown = shutdown
	})
	return c.initErr
}

// ShutdownObservability flushes pending spans/metrics and shuts down OTel providers.
// Uses a 5-second timeout context to ensure flush completes before process exit.
// Safe to call before InitObservability (no-op if not initialized).
//
// Called from root.go PersistentPostRunE before CloseDB().
func ShutdownObservability() error {
	c := loadObsContainer()

	// Close the log file descriptor (if any) so we don't leak it. We do this
	// regardless of whether c.shutdown is set -- the logger is initialized
	// independently of the OTel provider. Clear the field after closing so
	// repeat calls don't attempt a double-close.
	if c.logFile != nil {
		if err := c.logFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			// Suppress already-closed errors (idempotent shutdown). Other errors
			// are swallowed here because shutdown must not abort; higher layers
			// can surface issues via log output.
			_ = err
		}
		c.logFile = nil
	}

	if c.shutdown == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.shutdown(ctx)
}

// ResetObservability clears the global observability state.
// For testing only -- DO NOT use in production code.
// Called from ResetServices() to ensure test isolation.
func ResetObservability() {
	c := loadObsContainer()
	if c.shutdown != nil {
		_ = c.shutdown(context.Background())
	}
	// Close any log file opened by InitLoggerWithRoot so tests don't leak
	// file descriptors between cases. Suppress os.ErrClosed to stay idempotent.
	if c.logFile != nil {
		if err := c.logFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			_ = err
		}
		c.logFile = nil
	}
	// Swap to a fresh container
	storeObsContainer(new(obsContainer))
	// Sync the package-level alias for test backward compatibility
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
	c := loadObsContainer()
	meter := GetMeter("shark.cli")
	m, err := observability.NewCommandMetrics(meter)
	if err != nil {
		// Non-fatal: zero-value CommandMetrics is safe (all methods are no-ops)
		return
	}
	c.cmdMetrics = m
	c.cmdStartTime = time.Now()
	// Sync the package-level alias for test backward compatibility
	cmdStartTime = c.cmdStartTime
}

// RecordCommandMetrics records invocation count and duration for the completed command.
// cmdName is the Cobra command path (e.g., "task get"). cmdErr is the error returned
// by the command's RunE (nil on success). Must be called BEFORE ShutdownObservability
// so metrics can be flushed.
func RecordCommandMetrics(ctx context.Context, cmdName string, cmdErr error) {
	c := loadObsContainer()
	duration := time.Since(c.cmdStartTime)
	c.cmdMetrics.RecordDuration(ctx, cmdName, duration, cmdErr)
	c.cmdMetrics.RecordInvocation(ctx, cmdName, cmdErr)
}

// GetCommandMetrics returns the current CommandMetrics instance.
// Returns the zero value if InitCommandMetrics was not called or failed.
func GetCommandMetrics() observability.CommandMetrics {
	c := loadObsContainer()
	return c.cmdMetrics
}
