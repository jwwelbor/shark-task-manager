package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/observability"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

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
	observability.InitLogger(cfg)

	// Initialize database
	database, err := db.InitDB("shark-tasks.db")
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	slog.Info("Database initialized successfully")

	// Run integrity check
	if err := db.CheckIntegrity(database); err != nil {
		slog.Error("Database integrity check failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database integrity check passed")

	// Set up routes
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Shark Task Manager API - Database Ready")
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := database.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Database unavailable: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Wrap the mux with otelhttp middleware for automatic span creation
	// and request metrics on all routes.
	handler := otelhttp.NewHandler(mux, "shark-api")

	// Start server
	port := "8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("Server failed to start", "error", err)
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
