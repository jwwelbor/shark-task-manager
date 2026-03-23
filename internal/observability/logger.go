package observability

import (
	"log/slog"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// InitLogger configures the global slog default logger based on cfg.
// All output is directed to os.Stderr exclusively.
// When cfg.Enabled is false, installs a discard handler (level above LevelError,
// so no output is produced). Must be called once during startup before any slog.* calls.
func InitLogger(cfg config.ObservabilityConfig) {
	if !cfg.Enabled {
		// Install a handler that discards everything by setting level above LevelError.
		// slog.LevelError is 8; level 9 effectively discards all standard levels.
		opts := &slog.HandlerOptions{
			Level: slog.Level(slog.LevelError + 1),
		}
		handler := slog.NewTextHandler(os.Stderr, opts)
		slog.SetDefault(slog.New(handler))
		return
	}

	level := parseLogLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	svcName := cfg.ServiceName
	if svcName == "" {
		svcName = defaultServiceName
	}

	var handler slog.Handler
	format := strings.ToLower(cfg.LogFormat)
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		// Default to JSON format
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	logger := slog.New(handler).With("service.name", svcName)
	slog.SetDefault(logger)
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
