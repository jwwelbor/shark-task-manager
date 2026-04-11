package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/api"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/dbinit"
	"github.com/jwwelbor/shark-task-manager/internal/observability"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// shutdownTimeout is the maximum time the server waits for in-flight requests
// to complete before forcibly closing connections on shutdown.
const shutdownTimeout = 30 * time.Second

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

	// Initialize database (cloud-aware: reads .sharkconfig.json for backend selection).
	repoDB, err := dbinit.Init(context.Background(), dbinit.Options{})
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer repoDB.Close()

	slog.Info("Database initialized successfully")

	// Run integrity check on the underlying *sql.DB.
	if err := db.CheckIntegrity(repoDB.DB); err != nil {
		slog.Error("Database integrity check failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database integrity check passed")

	// Wire up services.
	svcs := WireServices(repoDB, ".")

	// Set up routes
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Shark Task Manager API - Database Ready")
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := repoDB.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Database unavailable: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Register CRUD handlers for tasks, features, and epics.
	api.NewTaskHandler(svcs.TaskService).RegisterRoutes(mux)
	api.NewFeatureHandler(svcs.FeatureService).RegisterRoutes(mux)
	api.NewEpicHandler(svcs.EpicService).RegisterRoutes(mux)

	slog.Info("API routes registered", "prefix", "/api/v1")

	// Wrap the mux with otelhttp middleware for automatic span creation
	// and request metrics on all routes.
	handler := otelhttp.NewHandler(mux, "shark-api")

	// Build the HTTP server struct so we can call Shutdown later.
	port := "8080"
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	// Start server in a goroutine so the main goroutine can listen for signals.
	srvErr := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	// Block until an OS signal or a server startup error is received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-srvErr:
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	// Give in-flight requests up to shutdownTimeout to complete.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")
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
