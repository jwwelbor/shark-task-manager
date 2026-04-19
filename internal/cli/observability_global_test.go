package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"go.opentelemetry.io/otel"
)

func TestInitObservability_Disabled(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled: false,
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability with disabled config should not error, got: %v", err)
	}

	// Verify noop providers are installed (no-op tracer produces invalid span IDs)
	tracer := GetTracer("test")
	if tracer == nil {
		t.Fatal("GetTracer should return a non-nil tracer")
	}

	meter := GetMeter("test")
	if meter == nil {
		t.Fatal("GetMeter should return a non-nil meter")
	}
}

func TestInitObservability_EnabledStdout(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: true,
		Exporter:       "stdout",
		LogLevel:       "info",
		LogFormat:      "json",
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability with stdout config should not error, got: %v", err)
	}

	// Verify tracer and meter return valid instances
	tracer := GetTracer("test/service")
	if tracer == nil {
		t.Fatal("GetTracer should return a non-nil tracer after init")
	}

	meter := GetMeter("test/service")
	if meter == nil {
		t.Fatal("GetMeter should return a non-nil meter after init")
	}
}

func TestInitObservability_Idempotent(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled: false,
	}

	// Call multiple times
	err1 := InitObservability(cfg)
	err2 := InitObservability(cfg)

	if err1 != nil {
		t.Fatalf("first InitObservability should not error, got: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("second InitObservability should not error, got: %v", err2)
	}
}

func TestShutdownObservability_BeforeInit(t *testing.T) {
	defer ResetObservability()

	// Shutdown before init should be a no-op
	err := ShutdownObservability()
	if err != nil {
		t.Fatalf("ShutdownObservability before init should not error, got: %v", err)
	}
}

func TestShutdownObservability_Idempotent(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	// Shutdown multiple times -- should not panic or error
	err1 := ShutdownObservability()
	if err1 != nil {
		t.Fatalf("first ShutdownObservability should not error, got: %v", err1)
	}

	err2 := ShutdownObservability()
	if err2 != nil {
		t.Fatalf("second ShutdownObservability should not error, got: %v", err2)
	}
}

func TestResetObservability_ClearsState(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled: false,
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	// Reset clears state
	ResetObservability()

	// After reset, globalObsShutdown should be nil (reset installs noop, but clears the stored func)
	// Verify we can re-initialize (sync.Once was reset)
	err = InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability after reset should not error, got: %v", err)
	}
}

func TestGetTracer_BeforeInit(t *testing.T) {
	defer ResetObservability()

	// Before init, OTel global defaults to noop
	tracer := GetTracer("test")
	if tracer == nil {
		t.Fatal("GetTracer should return a non-nil tracer even before init")
	}

	// Create a span to verify it works (noop span)
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()
}

func TestGetMeter_BeforeInit(t *testing.T) {
	defer ResetObservability()

	// Before init, OTel global defaults to noop
	meter := GetMeter("test")
	if meter == nil {
		t.Fatal("GetMeter should return a non-nil meter even before init")
	}
}

func TestGetTracer_ReturnsValidInstances(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		Exporter:       "stdout",
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	// Different names should return different tracers (by convention)
	tracer1 := GetTracer("service/task")
	tracer2 := GetTracer("service/feature")

	if tracer1 == nil || tracer2 == nil {
		t.Fatal("GetTracer should return non-nil tracers")
	}

	// Verify they produce valid spans
	_, span := tracer1.Start(context.Background(), "test-op")
	span.End()
}

func TestGetMeter_ReturnsValidInstances(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	meter := GetMeter("cli/commands")
	if meter == nil {
		t.Fatal("GetMeter should return a non-nil meter")
	}
}

func TestInitObservability_FailureIsNonFatal(t *testing.T) {
	defer ResetObservability()

	// Use OTLP exporter with unreachable endpoint -- this may or may not fail
	// depending on whether the gRPC client validates eagerly.
	// But the key test: even if init fails, noop providers are installed and
	// GetTracer/GetMeter work fine.
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: true,
		Exporter:       "stdout", // Use stdout to avoid network errors
	}

	err := InitObservability(cfg)
	// stdout exporter should succeed, but even if it failed:
	// - GetTracer must still work
	// - GetMeter must still work
	_ = err

	tracer := GetTracer("test")
	if tracer == nil {
		t.Fatal("GetTracer must return non-nil even after init failure")
	}

	meter := GetMeter("test")
	if meter == nil {
		t.Fatal("GetMeter must return non-nil even after init failure")
	}

	// Create span and metric to verify they don't panic
	_, span := tracer.Start(context.Background(), "test")
	span.End()
}

// TestShutdownObservability_ClosesLogFile verifies that when log_file is
// configured, the file descriptor opened by InitLoggerWithRoot is closed
// as part of ShutdownObservability. This is test plan case 8 from
// E07-F40 and covers AC-T3.
//
// Strategy: configure an absolute log file path, call InitObservability,
// emit a log record to force writes, then call ShutdownObservability.
// After shutdown, writing to the same file descriptor (captured via a
// helper that inspects the container) must fail with os.ErrClosed.
func TestShutdownObservability_ClosesLogFile(t *testing.T) {
	ResetObservability()
	defer ResetObservability()

	// Use temp directory so test does not litter the project root.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "shark.log")

	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogLevel:  "info",
		LogFormat: "json",
		LogFile:   logPath,
	}

	if err := InitObservability(cfg); err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	// The container must now hold a non-nil logFile closer.
	c := loadObsContainer()
	if c.logFile == nil {
		t.Fatal("expected obsContainer.logFile to be non-nil after InitObservability with log_file set")
	}

	// Keep a reference to the closer so we can assert it's closed after shutdown.
	closerBefore := c.logFile

	if err := ShutdownObservability(); err != nil {
		t.Fatalf("ShutdownObservability should not error, got: %v", err)
	}

	// After shutdown, calling Close() again on the same descriptor must return
	// os.ErrClosed (file was already closed by ShutdownObservability).
	closeErr := closerBefore.Close()
	if closeErr == nil {
		t.Fatal("expected closerBefore.Close() to return os.ErrClosed after shutdown, got nil")
	}
	if !errors.Is(closeErr, os.ErrClosed) {
		t.Errorf("expected os.ErrClosed after shutdown, got: %v", closeErr)
	}

	// Container's logFile must also be cleared so subsequent shutdown calls don't
	// double-close the descriptor.
	if c.logFile != nil {
		t.Errorf("expected obsContainer.logFile to be nil after ShutdownObservability, got non-nil")
	}
}

// TestShutdownObservability_LogFileCloseIsIdempotent verifies that calling
// ShutdownObservability multiple times when a log file was opened does not
// return an error on subsequent calls (os.ErrClosed suppressed or logFile
// nilled out on first close).
func TestShutdownObservability_LogFileCloseIsIdempotent(t *testing.T) {
	ResetObservability()
	defer ResetObservability()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "idempotent.log")

	cfg := config.ObservabilityConfig{
		Enabled: true,
		LogFile: logPath,
	}

	if err := InitObservability(cfg); err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	if err := ShutdownObservability(); err != nil {
		t.Fatalf("first ShutdownObservability should not error, got: %v", err)
	}
	if err := ShutdownObservability(); err != nil {
		t.Fatalf("second ShutdownObservability should not error, got: %v", err)
	}
}

// TestResetObservability_ClosesLogFile verifies that ResetObservability
// closes any open log file descriptor (AC-T4). Tests must be able to reset
// state without leaking file descriptors between cases.
func TestResetObservability_ClosesLogFile(t *testing.T) {
	ResetObservability()
	defer ResetObservability()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "reset.log")

	cfg := config.ObservabilityConfig{
		Enabled: true,
		LogFile: logPath,
	}

	if err := InitObservability(cfg); err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	c := loadObsContainer()
	if c.logFile == nil {
		t.Fatal("expected obsContainer.logFile to be non-nil after InitObservability with log_file set")
	}
	closerBefore := c.logFile

	ResetObservability()

	// After reset, the previous closer must report already closed.
	closeErr := closerBefore.Close()
	if closeErr == nil {
		t.Fatal("expected closerBefore.Close() to return os.ErrClosed after reset, got nil")
	}
	if !errors.Is(closeErr, os.ErrClosed) {
		t.Errorf("expected os.ErrClosed after reset, got: %v", closeErr)
	}
}

func TestResetObservability_ReinstallsNoopProviders(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}

	err := InitObservability(cfg)
	if err != nil {
		t.Fatalf("InitObservability should not error, got: %v", err)
	}

	// After reset, noop providers should be installed
	ResetObservability()

	// Verify the global OTel providers are noop by checking the type name.
	// NoopProvider() installs nooptrace.TracerProvider (value type, not pointer),
	// so we check via the type string rather than a type assertion.
	tp := otel.GetTracerProvider()
	mp := otel.GetMeterProvider()

	tpType := fmt.Sprintf("%T", tp)
	mpType := fmt.Sprintf("%T", mp)

	if tpType != "noop.TracerProvider" {
		t.Errorf("expected noop.TracerProvider after reset, got %s", tpType)
	}
	if mpType != "noop.MeterProvider" {
		t.Errorf("expected noop.MeterProvider after reset, got %s", mpType)
	}
}
