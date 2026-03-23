package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewCommandMetrics_NoopMeter(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	cm, err := NewCommandMetrics(meter)
	require.NoError(t, err)
	assert.NotNil(t, cm.duration)
	assert.NotNil(t, cm.invocations)
	assert.NotNil(t, cm.errors)
}

func TestCommandMetrics_RecordDuration_Success(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	cm, err := NewCommandMetrics(meter)
	require.NoError(t, err)

	// Should not panic
	cm.RecordDuration(context.Background(), "task get", 100*time.Millisecond, nil)
}

func TestCommandMetrics_RecordDuration_Error(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	cm, err := NewCommandMetrics(meter)
	require.NoError(t, err)

	// Should not panic
	cm.RecordDuration(context.Background(), "task get", 100*time.Millisecond, errors.New("test error"))
}

func TestCommandMetrics_RecordInvocation_Success(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	cm, err := NewCommandMetrics(meter)
	require.NoError(t, err)

	// Should not panic
	cm.RecordInvocation(context.Background(), "task list", nil)
}

func TestCommandMetrics_RecordInvocation_Error(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	cm, err := NewCommandMetrics(meter)
	require.NoError(t, err)

	// Should not panic and should also record error count
	cm.RecordInvocation(context.Background(), "task list", errors.New("db error"))
}

func TestCommandMetrics_ZeroValue_SafeToUse(t *testing.T) {
	var cm CommandMetrics
	// All methods should be safe to call on zero value
	cm.RecordDuration(context.Background(), "test", time.Second, nil)
	cm.RecordInvocation(context.Background(), "test", nil)
	cm.RecordInvocation(context.Background(), "test", errors.New("err"))
}

func TestNewDBMetrics_NoopMeter(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	dm, err := NewDBMetrics(meter)
	require.NoError(t, err)
	assert.NotNil(t, dm.queryDuration)
	assert.NotNil(t, dm.queryErrors)
}

func TestDBMetrics_RecordQueryDuration_Success(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	dm, err := NewDBMetrics(meter)
	require.NoError(t, err)

	// Should not panic
	dm.RecordQueryDuration(context.Background(), "SELECT", "tasks", 5*time.Millisecond, nil)
}

func TestDBMetrics_RecordQueryDuration_Error(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	dm, err := NewDBMetrics(meter)
	require.NoError(t, err)

	// Should not panic and record error
	dm.RecordQueryDuration(context.Background(), "INSERT", "tasks", 10*time.Millisecond, errors.New("constraint violation"))
}

func TestDBMetrics_ZeroValue_SafeToUse(t *testing.T) {
	var dm DBMetrics
	// All methods should be safe to call on zero value
	dm.RecordQueryDuration(context.Background(), "SELECT", "tasks", time.Millisecond, nil)
	dm.RecordQueryDuration(context.Background(), "INSERT", "tasks", time.Millisecond, errors.New("err"))
}

func TestErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"short error", errors.New("timeout"), "timeout"},
		{"long error", errors.New("this is a very long error message that exceeds fifty characters in total length for testing"), "this is a very long error message that exceeds fif"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errorType(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
