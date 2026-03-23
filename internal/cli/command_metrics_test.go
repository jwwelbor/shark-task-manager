package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/observability"
	"github.com/stretchr/testify/assert"
)

func TestInitCommandMetrics_CreatesInstruments(t *testing.T) {
	defer ResetObservability()

	// Initialize observability with noop/stdout exporter
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}
	err := InitObservability(cfg)
	assert.NoError(t, err)

	// Initialize command metrics
	InitCommandMetrics()

	// Verify metrics were created (non-zero value)
	cm := GetCommandMetrics()
	// CommandMetrics fields are unexported, but we can verify it works
	// by calling methods without panic
	cm.RecordDuration(context.Background(), "test", 100*time.Millisecond, nil)
	cm.RecordInvocation(context.Background(), "test", nil)
}

func TestInitCommandMetrics_SetsStartTime(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}
	err := InitObservability(cfg)
	assert.NoError(t, err)

	before := time.Now()
	InitCommandMetrics()
	after := time.Now()

	// cmdStartTime should be between before and after
	assert.False(t, cmdStartTime.Before(before), "cmdStartTime should be >= before")
	assert.False(t, cmdStartTime.After(after), "cmdStartTime should be <= after")
}

func TestRecordCommandMetrics_Success(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}
	err := InitObservability(cfg)
	assert.NoError(t, err)

	InitCommandMetrics()

	// Should not panic on success path
	RecordCommandMetrics(context.Background(), "shark task get", nil)
}

func TestRecordCommandMetrics_Error(t *testing.T) {
	defer ResetObservability()

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}
	err := InitObservability(cfg)
	assert.NoError(t, err)

	InitCommandMetrics()

	// Should not panic on error path
	RecordCommandMetrics(context.Background(), "shark task get", errors.New("not found"))
}

func TestRecordCommandMetrics_ZeroValue_NoPanic(t *testing.T) {
	defer ResetObservability()

	// Do NOT call InitCommandMetrics -- cmdMetrics is zero value
	// This tests AC-3: no panic when OTel is not initialized

	// Should not panic
	RecordCommandMetrics(context.Background(), "shark task get", nil)
	RecordCommandMetrics(context.Background(), "shark task get", errors.New("error"))
}

func TestGetCommandMetrics_ZeroValueBeforeInit(t *testing.T) {
	defer ResetObservability()

	// Before init, should return zero value
	cm := GetCommandMetrics()

	// Zero value CommandMetrics should be safe to use
	cm.RecordDuration(context.Background(), "test", time.Second, nil)
	cm.RecordInvocation(context.Background(), "test", nil)
	cm.RecordInvocation(context.Background(), "test", errors.New("err"))
}

func TestResetObservability_ClearsCommandMetrics(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}
	err := InitObservability(cfg)
	assert.NoError(t, err)

	InitCommandMetrics()
	assert.False(t, cmdStartTime.IsZero(), "cmdStartTime should be set after init")

	ResetObservability()

	// After reset, cmdStartTime should be zero
	assert.True(t, cmdStartTime.IsZero(), "cmdStartTime should be zero after reset")

	// GetCommandMetrics should return zero value
	cm := GetCommandMetrics()
	// Should be safe to call (zero value is no-op)
	cm.RecordDuration(context.Background(), "test", time.Second, nil)
	cm.RecordInvocation(context.Background(), "test", nil)
}

func TestCommandMetrics_ZeroValueStruct_AllMethodsSafe(t *testing.T) {
	// Directly test the zero-value CommandMetrics struct
	var cm observability.CommandMetrics

	// All methods should be safe to call on zero value (no panic)
	cm.RecordDuration(context.Background(), "test cmd", time.Second, nil)
	cm.RecordDuration(context.Background(), "test cmd", time.Second, errors.New("err"))
	cm.RecordInvocation(context.Background(), "test cmd", nil)
	cm.RecordInvocation(context.Background(), "test cmd", errors.New("err"))
}
