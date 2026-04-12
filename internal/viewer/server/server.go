// Package server provides the importable StartServer entry point for the
// shark web dashboard. It is used by both cmd/server/main.go (thin wrapper)
// and internal/cli/commands/web.go (shark web command).
//
// The package lives in internal/viewer/server (NOT cmd/server) so that it can
// be imported by the CLI without creating a circular dependency.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/api"
	"github.com/jwwelbor/shark-task-manager/internal/api/viewer"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/dbinit"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	viewerassets "github.com/jwwelbor/shark-task-manager/internal/viewer"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// shutdownTimeout is the maximum time the server waits for in-flight requests
// to complete before forcibly closing connections on shutdown.
const shutdownTimeout = 30 * time.Second

// Options configures StartServer behaviour.
type Options struct {
	// Listener is a pre-bound net.Listener. If non-nil, the server will use it
	// instead of binding opts.Addr. Useful for tests and for the CLI (shark web)
	// which binds a port before forking the server goroutine.
	Listener net.Listener

	// Addr is the TCP address to bind when Listener is nil.
	// Defaults to ":8080" when both Listener and Addr are unset.
	Addr string

	// ProjectRoot overrides the auto-detected project root used for loading
	// .sharkconfig.json and for wiring services.
	ProjectRoot string

	// DB is an optional pre-initialized *repository.DB. When non-nil,
	// StartServer skips dbinit.Init and does NOT call db.Close() on exit —
	// the caller retains ownership of the connection.
	DB *repository.DB

	// Ready is an optional channel. StartServer closes it after the listener
	// is bound and ready to accept connections, but BEFORE blocking on Serve.
	// Callers can use this to detect readiness without polling.
	Ready chan<- struct{}
}

// StartServer wires the full HTTP server (all routes, CORS, otelhttp) and
// serves until ctx is cancelled, then performs a graceful 30-second shutdown.
//
// Behaviour summary:
//  1. If opts.DB is nil:  call dbinit.Init to obtain *repository.DB.
//  2. Call db.CheckIntegrity on the underlying *sql.DB.
//  3. Call WireServices(db, projectRoot) to build the service graph.
//  4. Build http.ServeMux: SPA at GET /, health, CRUD routes, viewer routes.
//  5. Wrap mux in otelhttp.NewHandler.
//  6. Bind listener (opts.Listener if provided, else opts.Addr / ":8080").
//  7. Close opts.Ready (if non-nil) after binding, before blocking on Serve.
//  8. On ctx.Done(): call srv.Shutdown with 30-second timeout.
//  9. Translate http.ErrServerClosed → nil.
//
// 10. When opts.DB was non-nil (caller-provided): do NOT call db.Close().
func StartServer(ctx context.Context, opts Options) error {
	// --- Step 1: Resolve/initialize DB ----------------------------------------

	callerOwnedDB := opts.DB != nil

	repoDB := opts.DB
	if !callerOwnedDB {
		var err error
		repoDB, err = dbinit.Init(ctx, dbinit.Options{ProjectRoot: opts.ProjectRoot})
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		// Only close the DB if we opened it.
		defer repoDB.Close()
	}

	// --- Step 2: Integrity check -----------------------------------------------

	if err := db.CheckIntegrity(repoDB.DB); err != nil {
		return fmt.Errorf("database integrity check failed: %w", err)
	}
	slog.Info("Database integrity check passed")

	// --- Step 3: Wire services -------------------------------------------------

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	svcs := WireServices(repoDB, projectRoot)

	// --- Step 4: Build mux -----------------------------------------------------

	mux := http.NewServeMux()

	// SPA: serve the embedded viewer.html at GET /.
	// Use method-qualified patterns consistently to avoid conflicts with other routes.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(viewerassets.ViewerHTML)
	})

	// Health endpoint.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := repoDB.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "Database unavailable: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "OK")
	})

	// CRUD routes for tasks, features, and epics.
	api.NewTaskHandler(svcs.TaskService).RegisterRoutes(mux)
	api.NewFeatureHandler(svcs.FeatureService).RegisterRoutes(mux)
	api.NewEpicHandler(svcs.EpicService).RegisterRoutes(mux)

	// Read-only viewer dashboard routes under /api/v1/viewer/ (with CORS).
	viewer.NewViewerHandler(svcs.ViewerService).RegisterRoutes(mux, "/api/v1/viewer")

	slog.Info("API routes registered", "prefix", "/api/v1")

	// --- Step 5: Wrap with otelhttp --------------------------------------------

	handler := otelhttp.NewHandler(mux, "shark-api")

	// --- Step 6: Bind listener -------------------------------------------------

	ln := opts.Listener
	if ln == nil {
		addr := opts.Addr
		if addr == "" {
			addr = ":8080"
		}
		var err error
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
		// Only close the listener on error paths; srv.Serve takes ownership.
	}

	srv := &http.Server{Handler: handler}

	// --- Step 7: Signal readiness BEFORE blocking on Serve --------------------

	if opts.Ready != nil {
		close(opts.Ready)
	}

	// --- Step 8: Serve + graceful shutdown on ctx cancellation ----------------

	srvErr := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "addr", ln.Addr().String())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		} else {
			srvErr <- nil
		}
	}()

	select {
	case err := <-srvErr:
		// Server stopped before ctx was cancelled (likely a bind/accept error).
		return err

	case <-ctx.Done():
		slog.Info("Shutdown signal received; stopping server")
	}

	// --- Step 9: Graceful shutdown ---------------------------------------------

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// Drain srvErr so the goroutine exits cleanly.
	if err := <-srvErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error during shutdown: %w", err)
	}

	slog.Info("Server stopped gracefully")
	return nil
}
