package observability

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// InitLoggerWithRoot configures the global slog default logger based on cfg.
// If cfg.LogFile is set, writes to that file (relative paths resolved against
// projectRoot). Returns an io.Closer for the opened file, or nil if writing
// to stderr.
//
// When cfg.Enabled is false, installs a discard handler and returns nil without
// opening any file. When cfg.LogFile is empty, behaves identically to the
// original InitLogger. On file open failure, falls back to stderr and prints
// exactly one raw warning line via fmt.Fprintln (not via slog).
func InitLoggerWithRoot(cfg config.ObservabilityConfig, projectRoot string) io.Closer {
	if !cfg.Enabled {
		// Install a handler that discards everything by setting level above LevelError.
		// slog.LevelError is 8; level 9 effectively discards all standard levels.
		opts := &slog.HandlerOptions{
			Level: slog.Level(slog.LevelError + 1),
		}
		handler := slog.NewTextHandler(os.Stderr, opts)
		slog.SetDefault(slog.New(handler))
		return nil
	}

	level := parseLogLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	svcName := cfg.ServiceName
	if svcName == "" {
		svcName = defaultServiceName
	}

	// Determine the writer: file or stderr.
	writer, closer := resolveWriter(cfg.LogFile, projectRoot)

	var handler slog.Handler
	format := strings.ToLower(cfg.LogFormat)
	switch format {
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		// Default to JSON format
		handler = slog.NewJSONHandler(writer, opts)
	}

	logger := slog.New(handler).With("service.name", svcName)
	slog.SetDefault(logger)

	return closer
}

// InitLogger configures the global slog default logger based on cfg.
// When cfg.Enabled is false, installs a discard handler (level above LevelError,
// so no output is produced). Must be called once during startup before any slog.* calls.
//
// This function preserves the original signature for backward compatibility and
// delegates to InitLoggerWithRoot with projectRoot == "".
//
// Deprecated for use with file logging: InitLogger discards the io.Closer
// returned by InitLoggerWithRoot. When cfg.LogFile is set, the opened file
// handle cannot be closed by the caller, which will leak a file descriptor
// per invocation. Callers that configure cfg.LogFile MUST use
// InitLoggerWithRoot and close the returned io.Closer on shutdown.
func InitLogger(cfg config.ObservabilityConfig) {
	InitLoggerWithRoot(cfg, "")
}

// resolveWriter opens a log file when logFile is non-empty, returning the file
// as io.Writer plus an io.Closer. Relative paths are resolved against projectRoot.
// Parent directories are created with MkdirAll(0755). On any failure, falls back
// to os.Stderr with a single raw warning line and returns (os.Stderr, nil).
func resolveWriter(logFile, projectRoot string) (io.Writer, io.Closer) {
	if logFile == "" {
		return os.Stderr, nil
	}

	// Resolve relative paths against projectRoot.
	path := logFile
	if !filepath.IsAbs(path) {
		if projectRoot != "" {
			path = filepath.Join(projectRoot, path)
		}
		// If projectRoot is empty and path is relative, use it as-is (CWD).
	}

	// Create parent directory if missing.
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "shark warn: failed to create log directory, falling back to stderr:", err)
		return os.Stderr, nil
	}

	// Open file in append mode; create if missing.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shark warn: failed to open log file, falling back to stderr:", err)
		return os.Stderr, nil
	}

	return f, f
}

// parseLogLevel converts a string log level to slog.Level.
// Defaults to slog.LevelInfo for unrecognized values.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
