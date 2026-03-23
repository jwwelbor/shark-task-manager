package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestNoopProvider_ReturnsNonNilShutdownFunc(t *testing.T) {
	shutdown := NoopProvider()
	assert.NotNil(t, shutdown)
}

func TestNoopProvider_ShutdownReturnsNil(t *testing.T) {
	shutdown := NoopProvider()
	err := shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNoopProvider_ShutdownIsIdempotent(t *testing.T) {
	shutdown := NoopProvider()
	err1 := shutdown(context.Background())
	err2 := shutdown(context.Background())
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestNoopProvider_TracerDoesNotPanic(t *testing.T) {
	NoopProvider()
	tracer := otel.Tracer("test")
	require.NotNil(t, tracer)

	ctx, span := tracer.Start(context.Background(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End() // Should not panic
}

func TestNoopProvider_MeterDoesNotPanic(t *testing.T) {
	NoopProvider()
	meter := otel.Meter("test")
	require.NotNil(t, meter)

	counter, err := meter.Int64Counter("test.counter")
	assert.NoError(t, err)
	assert.NotNil(t, counter)
	counter.Add(context.Background(), 1) // Should not panic
}

func TestNoopProvider_SafeToCallMultipleTimes(t *testing.T) {
	s1 := NoopProvider()
	s2 := NoopProvider()
	s3 := NoopProvider()

	assert.NoError(t, s1(context.Background()))
	assert.NoError(t, s2(context.Background()))
	assert.NoError(t, s3(context.Background()))
}
