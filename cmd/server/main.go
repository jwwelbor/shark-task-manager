package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/observability"
	viewerserver "github.com/jwwelbor/shark-task-manager/internal/viewer/server"
)

// serverAddr returns the loopback TCP address to bind. It checks the PORT
// environment variable first, then falls back to 127.0.0.1:8080.
func serverAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return "127.0.0.1:" + port
	}
	return "127.0.0.1:8080"
}

func main() {
	// Initialize observability (tracing, metrics, structured logging).
	// Loads config from .sharkconfig.json; defaults to disabled if missing.
	cfg := loadObservabilityConfig()
	shutdown, err := observability.InitProvider(cfg)
	if err != nil {
		slog.Warn("Failed to initialize observability provider", "error", err)
	} else {
		defer func() {
			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				slog.Error("Observability provider shutdown error", "error", shutdownErr)
			}
		}()
	}
	logCloser := observability.InitLoggerWithRoot(cfg, ".")
	if logCloser != nil {
		defer func() {
			if closeErr := logCloser.Close(); closeErr != nil {
				slog.Error("Log file close error", "error", closeErr)
			}
		}()
	}

	// Build a context that is cancelled on SIGINT/SIGTERM so StartServer can
	// initiate graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := viewerserver.StartServer(ctx, viewerserver.Options{Addr: serverAddr()}); err != nil {
		slog.Error("Server stopped with error", "error", err)
		os.Exit(1)
	}
}

// loadObservabilityConfig loads the observability configuration from .sharkconfig.json.
// Returns a zero-value config (disabled) if the file is missing or unreadable.
func loadObservabilityConfig() config.ObservabilityConfig {
	mgr := config.NewManager(".sharkconfig.json")
	cfg, err := mgr.Load()
	if err != nil {
		return config.ObservabilityConfig{}
	}
	return cfg.GetObservability()
}
