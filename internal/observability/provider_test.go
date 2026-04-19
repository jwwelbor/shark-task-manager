package observability

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestInitProvider_DisabledReturnsNoopShutdown(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled: false,
	}
	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown should return nil
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_DisabledTracerDoesNotPanic(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled: false,
	}
	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)
	defer func() { _ = shutdown(context.Background()) }()

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test")
	assert.NotNil(t, ctx)
	span.End()
}

func TestInitProvider_StdoutExporter_NoStdoutOutput(t *testing.T) {
	// Capture stdout to verify nothing is written there
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: false,
		Exporter:       "stdout",
		ServiceName:    "test-service",
	}

	shutdown, initErr := InitProvider(cfg)
	require.NoError(t, initErr)
	require.NotNil(t, shutdown)

	// Create a span to trigger output
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	// Shutdown and flush
	assert.NoError(t, shutdown(context.Background()))

	// Close writer and read captured stdout
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = origStdout

	// Stdout must be empty
	assert.Empty(t, buf.String(), "observability must not write to stdout")
}

func TestInitProvider_StdoutExporter_TracingAndMetrics(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: true,
		Exporter:       "stdout",
		ServiceName:    "test-service",
	}

	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Verify tracer works
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	// Verify meter works
	meter := otel.Meter("test")
	counter, mErr := meter.Int64Counter("test.counter")
	assert.NoError(t, mErr)
	counter.Add(context.Background(), 1)

	assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_ShutdownIsIdempotent(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		Exporter:       "stdout",
	}

	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)

	// Call shutdown multiple times
	assert.NoError(t, shutdown(context.Background()))
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_UnsupportedExporter(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		Exporter:       "invalid",
	}

	shutdown, err := InitProvider(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported exporter type")
	// Should still return a valid shutdown (noop fallback)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_TracingOnlyNoMetrics(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		MetricsEnabled: false,
		Exporter:       "stdout",
	}

	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitProvider_MetricsOnlyNoTracing(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: false,
		MetricsEnabled: true,
		Exporter:       "stdout",
	}

	shutdown, err := InitProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		initial  config.ObservabilityConfig
		expected config.ObservabilityConfig
	}{
		{
			name:     "SHARK_OTEL_ENABLED overrides to true",
			envVars:  map[string]string{"SHARK_OTEL_ENABLED": "true"},
			initial:  config.ObservabilityConfig{Enabled: false},
			expected: config.ObservabilityConfig{Enabled: true},
		},
		{
			name:     "SHARK_OTEL_ENABLED overrides to false",
			envVars:  map[string]string{"SHARK_OTEL_ENABLED": "false"},
			initial:  config.ObservabilityConfig{Enabled: true},
			expected: config.ObservabilityConfig{Enabled: false},
		},
		{
			name:     "OTEL_EXPORTER_OTLP_ENDPOINT overrides endpoint",
			envVars:  map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "collector:4317"},
			initial:  config.ObservabilityConfig{OTLPEndpoint: "localhost:4317"},
			expected: config.ObservabilityConfig{OTLPEndpoint: "collector:4317"},
		},
		{
			name:     "OTEL_EXPORTER_OTLP_PROTOCOL overrides protocol",
			envVars:  map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "http"},
			initial:  config.ObservabilityConfig{OTLPProtocol: "grpc"},
			expected: config.ObservabilityConfig{OTLPProtocol: "http"},
		},
		{
			name:     "OTEL_SERVICE_NAME overrides service name",
			envVars:  map[string]string{"OTEL_SERVICE_NAME": "my-service"},
			initial:  config.ObservabilityConfig{ServiceName: "default"},
			expected: config.ObservabilityConfig{ServiceName: "my-service"},
		},
		{
			name:     "SHARK_LOG_LEVEL overrides log level",
			envVars:  map[string]string{"SHARK_LOG_LEVEL": "debug"},
			initial:  config.ObservabilityConfig{LogLevel: "info"},
			expected: config.ObservabilityConfig{LogLevel: "debug"},
		},
		{
			name:     "SHARK_LOG_FORMAT overrides log format",
			envVars:  map[string]string{"SHARK_LOG_FORMAT": "text"},
			initial:  config.ObservabilityConfig{LogFormat: "json"},
			expected: config.ObservabilityConfig{LogFormat: "text"},
		},
		{
			name:     "SHARK_LOG_FILE overrides log file",
			envVars:  map[string]string{"SHARK_LOG_FILE": "/tmp/foo.log"},
			initial:  config.ObservabilityConfig{LogFile: "original.log"},
			expected: config.ObservabilityConfig{LogFile: "/tmp/foo.log"},
		},
		{
			name:     "SHARK_LOG_FILE sets log file when initial is empty",
			envVars:  map[string]string{"SHARK_LOG_FILE": "/var/log/shark.log"},
			initial:  config.ObservabilityConfig{LogFile: ""},
			expected: config.ObservabilityConfig{LogFile: "/var/log/shark.log"},
		},
		{
			name:     "empty SHARK_LOG_FILE does not override",
			envVars:  map[string]string{"SHARK_LOG_FILE": ""},
			initial:  config.ObservabilityConfig{LogFile: "keep.log"},
			expected: config.ObservabilityConfig{LogFile: "keep.log"},
		},
		{
			name:     "unset env does not override",
			envVars:  map[string]string{},
			initial:  config.ObservabilityConfig{LogLevel: "warn", ServiceName: "test"},
			expected: config.ObservabilityConfig{LogLevel: "warn", ServiceName: "test"},
		},
		{
			name:     "unset SHARK_LOG_FILE preserves cfg.LogFile",
			envVars:  map[string]string{},
			initial:  config.ObservabilityConfig{LogFile: "from-config.log"},
			expected: config.ObservabilityConfig{LogFile: "from-config.log"},
		},
		{
			name:     "invalid SHARK_OTEL_ENABLED does not change value",
			envVars:  map[string]string{"SHARK_OTEL_ENABLED": "notabool"},
			initial:  config.ObservabilityConfig{Enabled: false},
			expected: config.ObservabilityConfig{Enabled: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set env vars
			envKeys := []string{
				"SHARK_OTEL_ENABLED", "OTEL_EXPORTER_OTLP_ENDPOINT",
				"OTEL_EXPORTER_OTLP_PROTOCOL", "OTEL_SERVICE_NAME",
				"SHARK_LOG_LEVEL", "SHARK_LOG_FORMAT", "SHARK_LOG_FILE",
			}
			for _, k := range envKeys {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := tt.initial
			ApplyEnvOverrides(&cfg)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestBuildResource_DefaultServiceName(t *testing.T) {
	cfg := config.ObservabilityConfig{}
	res, err := buildResource(cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestBuildResource_CustomServiceName(t *testing.T) {
	cfg := config.ObservabilityConfig{ServiceName: "custom-svc"}
	res, err := buildResource(cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}
