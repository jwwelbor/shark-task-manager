package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	noopmetric "go.opentelemetry.io/otel/metric/noop"
)

// ShutdownFunc gracefully shuts down the OTel provider,
// flushing any pending spans or metrics before returning.
// Must be called before process exit when observability is enabled.
// Safe to call when observability is disabled (returns nil immediately).
type ShutdownFunc func(ctx context.Context) error

// NoopProvider installs no-op OTel providers globally and returns a no-op ShutdownFunc.
// Used when observability.enabled is false or in test environments.
// Safe to call multiple times. No goroutines are started, no file handles opened,
// no network connections attempted.
func NoopProvider() ShutdownFunc {
	otel.SetTracerProvider(nooptrace.NewTracerProvider())
	otel.SetMeterProvider(noopmetric.NewMeterProvider())
	return func(_ context.Context) error {
		return nil
	}
}
