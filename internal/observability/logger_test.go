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

// --- File destination tests (Test Plan cases 1-6) ---

// TestInitLogger_FileDestination_WritesToFile verifies that when LogFile is set,
// slog output is written to the file in the configured format (JSON by default).
func TestInitLogger_FileDestination_WritesToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/shark.log"

	cfg := config.ObservabilityConfig{
		Enabled:     true,
		LogFormat:   "json",
		LogLevel:    "info",
		ServiceName: "file-dest-test",
		LogFile:     logPath,
	}

	closer := InitLoggerWithRoot(cfg, tmpDir)
	if closer != nil {
		defer closer.Close()
	}

	slog.Info("file-test-message")

	// Flush by closing
	if closer != nil {
		closer.Close()
	}

	data, err := os.ReadFile(logPath)
	require.NoError(t, err, "log file should exist after logger initialization")
	assert.Contains(t, string(data), "file-test-message", "log file should contain emitted record")
}

// TestInitLogger_FileDestination_Appends verifies that calling InitLoggerWithRoot
// twice with the same path appends records rather than truncating.
func TestInitLogger_FileDestination_Appends(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/append.log"

	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogFormat: "json",
		LogLevel:  "info",
		LogFile:   logPath,
	}

	// First call
	closer1 := InitLoggerWithRoot(cfg, tmpDir)
	slog.Info("first-record")
	if closer1 != nil {
		closer1.Close()
	}

	// Second call to same path
	closer2 := InitLoggerWithRoot(cfg, tmpDir)
	slog.Info("second-record")
	if closer2 != nil {
		closer2.Close()
	}

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "first-record", "first record should be present after append")
	assert.Contains(t, content, "second-record", "second record should be present after append")
}

// TestInitLogger_FileDestination_CreatesParentDir verifies that missing parent
// directories are created with mode 0755.
func TestInitLogger_FileDestination_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := tmpDir + "/nested/deep/shark.log"

	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogFormat: "json",
		LogLevel:  "info",
		LogFile:   nestedPath,
	}

	closer := InitLoggerWithRoot(cfg, tmpDir)
	if closer != nil {
		defer closer.Close()
	}

	// The parent directory must have been created
	info, err := os.Stat(tmpDir + "/nested/deep")
	require.NoError(t, err, "parent directory should be created")
	assert.True(t, info.IsDir(), "created path should be a directory")
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm(), "directory should have 0755 permissions")
}

// TestInitLogger_FileDestination_BadPath_FallsBackToStderr verifies that when the
// log file cannot be opened, the logger falls back to stderr with exactly one raw
// warning line and subsequent records land on stderr.
func TestInitLogger_FileDestination_BadPath_FallsBackToStderr(t *testing.T) {
	// Use a path whose parent is a file (not a directory) to force open failure.
	tmpDir := t.TempDir()
	blockingFile := tmpDir + "/blocker"
	require.NoError(t, os.WriteFile(blockingFile, []byte("x"), 0644))
	badPath := blockingFile + "/shark.log" // parent exists as a file

	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogFormat: "json",
		LogLevel:  "info",
		LogFile:   badPath,
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	closer := InitLoggerWithRoot(cfg, tmpDir)
	if closer != nil {
		defer closer.Close()
	}
	slog.Info("fallback-marker")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = origStderr

	output := buf.String()
	// Must contain a raw warning (not via slog — no JSON wrapper required)
	assert.Contains(t, output, "warn", "stderr should contain a fallback warning")
	// The slog record goes to stderr (fallback)
	assert.Contains(t, output, "fallback-marker", "fallback slog record should appear on stderr")
	// Bad path file must NOT have been created
	assert.NoFileExists(t, badPath, "bad path must not result in partial file creation")
}

// TestInitLogger_FileDestination_RelativePath_ResolvedToRoot verifies that a
// relative LogFile path is resolved against projectRoot, not the process CWD.
func TestInitLogger_FileDestination_RelativePath_ResolvedToRoot(t *testing.T) {
	tmpDir := t.TempDir()
	// Change working dir to somewhere unrelated to confirm resolution uses tmpDir
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(os.TempDir()))

	cfg := config.ObservabilityConfig{
		Enabled:   true,
		LogFormat: "json",
		LogLevel:  "info",
		LogFile:   "relative.log",
	}

	closer := InitLoggerWithRoot(cfg, tmpDir)
	if closer != nil {
		defer closer.Close()
	}

	slog.Info("relative-path-record")
	if closer != nil {
		closer.Close()
	}

	// File should exist under tmpDir, not CWD
	expectedPath := tmpDir + "/relative.log"
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "log file should be created at projectRoot/relative.log")

	// Should NOT exist under the changed CWD
	cwdPath := os.TempDir() + "/relative.log"
	if cwdPath != expectedPath { // guard against tmpDir == os.TempDir() edge case
		_, cwdErr := os.Stat(cwdPath)
		assert.True(t, os.IsNotExist(cwdErr), "log file must not be created in CWD")
	}
}

// TestInitLogger_FileDestination_Disabled_NoFileOpened verifies that when
// observability is disabled, setting LogFile does not create any file.
func TestInitLogger_FileDestination_Disabled_NoFileOpened(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/disabled.log"

	cfg := config.ObservabilityConfig{
		Enabled: false,
		LogFile: logPath,
	}

	closer := InitLoggerWithRoot(cfg, tmpDir)
	if closer != nil {
		closer.Close()
	}

	_, err := os.Stat(logPath)
	assert.True(t, os.IsNotExist(err), "no log file should be created when observability is disabled")
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
