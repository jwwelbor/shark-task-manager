package observability

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger_Disabled_NoOutput(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled: false,
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	InitLogger(cfg)
	slog.Info("this should not appear")
	slog.Warn("this should not appear either")
	slog.Error("even errors should be suppressed")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = origStderr

	assert.Empty(t, buf.String(), "disabled logger should produce no output")
}

func TestInitLogger_Enabled_JSONFormat(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:     true,
		LogFormat:   "json",
		LogLevel:    "info",
		ServiceName: "test-svc",
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	InitLogger(cfg)
	slog.Info("test message")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = origStderr

	output := buf.String()
	assert.NotEmpty(t, output, "enabled logger should produce output")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "service.name")
	assert.Contains(t, output, "test-svc")
	// Should be valid JSON (contains opening brace)
	assert.Contains(t, output, "{")
}

func TestInitLogger_Enabled_TextFormat(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:     true,
		LogFormat:   "text",
		LogLevel:    "info",
		ServiceName: "test-svc",
	}

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	InitLogger(cfg)
	slog.Info("text message")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = origStderr

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "text message")
	assert.Contains(t, output, "service.name")
}

func TestInitLogger_NoStdoutOutput(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:     true,
		LogFormat:   "json",
		LogLevel:    "debug",
		ServiceName: "test-svc",
	}

	// Capture stdout
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	InitLogger(cfg)
	slog.Info("test stdout protection")
	slog.Debug("debug message")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = origStdout

	assert.Empty(t, buf.String(), "logger must not write to stdout")
}

func TestInitLogger_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name         string
		level        string
		infoVisible  bool
		debugVisible bool
		warnVisible  bool
	}{
		{"debug level shows all", "debug", true, true, true},
		{"info level shows info+", "info", true, false, true},
		{"warn level shows warn+", "warn", false, false, true},
		{"error level shows error only", "error", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ObservabilityConfig{
				Enabled:     true,
				LogFormat:   "text",
				LogLevel:    tt.level,
				ServiceName: "test",
			}

			origStderr := os.Stderr
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stderr = w

			InitLogger(cfg)
			slog.Debug("DEBUG_MARKER")
			slog.Info("INFO_MARKER")
			slog.Warn("WARN_MARKER")

			w.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			os.Stderr = origStderr

			output := buf.String()

			if tt.debugVisible {
				assert.Contains(t, output, "DEBUG_MARKER")
			} else {
				assert.NotContains(t, output, "DEBUG_MARKER")
			}
			if tt.infoVisible {
				assert.Contains(t, output, "INFO_MARKER")
			} else {
				assert.NotContains(t, output, "INFO_MARKER")
			}
			if tt.warnVisible {
				assert.Contains(t, output, "WARN_MARKER")
			} else {
				assert.NotContains(t, output, "WARN_MARKER")
			}
		})
	}
}

func TestInitLogger_DefaultServiceName(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogFormat: "json",
		LogLevel:  "info",
		// ServiceName left empty
	}

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	InitLogger(cfg)
	slog.Info("default name test")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = origStderr

	output := buf.String()
	assert.Contains(t, output, "shark-task-manager")
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseLogLevel(tt.input))
		})
	}
}
