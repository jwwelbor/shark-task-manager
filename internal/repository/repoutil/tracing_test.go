package repoutil

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTracingTest configures a recording TracerProvider and returns the exporter.
// The exporter captures all spans created during the test for assertion.
// Cleanup restores the original provider.
func setupTracingTest(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	prevProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)

	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		_ = tp.Shutdown(context.Background())
	})

	return exporter
}

// TC-F001-3: NewTracer creates tracer with the given instrumentation name.
func TestNewTracer_InstrumentationName(t *testing.T) {
	exporter := setupTracingTest(t)

	tracer := NewTracer("test/package")

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	stubs := exporter.GetSpans()
	if len(stubs) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	// Verify the instrumentation scope name matches
	found := false
	for _, stub := range stubs {
		if stub.InstrumentationScope.Name == "test/package" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected span with instrumentation scope 'test/package', got: %v", stubs[0].InstrumentationScope.Name)
	}
}

// TC-F001-3: RecordSpanError sets span status to Error and records an event.
func TestRecordSpanError_WithError(t *testing.T) {
	exporter := setupTracingTest(t)

	tracer := NewTracer("test/package")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")

	err := errors.New("boom")
	RecordSpanError(span, err)
	span.End()

	stubs := exporter.GetSpans()
	if len(stubs) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	stub := stubs[0]

	// Verify error status
	if stub.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", stub.Status.Code)
	}

	// Verify error event recorded
	if len(stub.Events) == 0 {
		t.Error("expected span events from RecordError, got none")
	}
}

// TC-F001-3 edge case: RecordSpanError with nil span must not panic.
func TestRecordSpanError_NilSpan(t *testing.T) {
	// Must not panic
	RecordSpanError(nil, errors.New("some error"))
}

// TC-F001-3 edge case: RecordSpanError with nil error must not change span status.
func TestRecordSpanError_NilError(t *testing.T) {
	exporter := setupTracingTest(t)

	tracer := NewTracer("test/package")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")

	RecordSpanError(span, nil)
	span.End()

	stubs := exporter.GetSpans()
	if len(stubs) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	stub := stubs[0]

	// Status should remain unset (not Error) when err is nil
	if stub.Status.Code == codes.Error {
		t.Errorf("expected span status to remain unset, got Error")
	}

	// No events should be recorded
	if len(stub.Events) != 0 {
		t.Errorf("expected no span events, got %d", len(stub.Events))
	}
}

// TC-NF004-1 edge case: NewTracer with no provider configured returns no-op tracer (no panic).
func TestNewTracer_NoProviderNoPanic(t *testing.T) {
	// Don't configure any provider — just use the global default (no-op)
	tracer := NewTracer("some/package")

	ctx := context.Background()
	// Should not panic
	_, span := tracer.Start(ctx, "test-span")
	span.End()
}
