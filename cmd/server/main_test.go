package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestOtelhttpMiddlewareWrapsHandler(t *testing.T) {
	// Verify otelhttp.NewHandler wraps a standard handler and still serves requests.
	inner := http.NewServeMux()
	inner.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := otelhttp.NewHandler(inner, "test-api")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "OK" {
		t.Errorf("expected body 'OK', got %q", body)
	}
}

func TestOtelhttpMiddlewarePreservesRouting(t *testing.T) {
	// Verify the middleware does not interfere with normal routing.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("root"))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("healthy"))
	})

	handler := otelhttp.NewHandler(mux, "test-api")

	tests := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"root path", "/", "root"},
		{"health path", "/health", "healthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("path %s: expected body %q, got %q", tt.path, tt.wantBody, got)
			}
		})
	}
}

func TestLoadObservabilityConfig_MissingFile(t *testing.T) {
	// loadObservabilityConfig should return a zero-value config (disabled)
	// when no .sharkconfig.json is present. Since tests run from a temp
	// directory or the project root without the expected config, this
	// verifies graceful fallback.
	cfg := loadObservabilityConfig()
	if cfg.Enabled {
		t.Error("expected observability to be disabled when config is missing")
	}
}

// TestGracefulShutdown_StopsAcceptingNewConnections verifies that calling
// srv.Shutdown() causes the server to stop and ListenAndServe to return
// http.ErrServerClosed (not a hard error).
func TestGracefulShutdown_StopsAcceptingNewConnections(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Pick an available port by letting the OS assign one.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := &http.Server{Handler: mux}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.Serve(ln)
	}()

	// Wait for the server to accept connections by dialing.
	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("server did not start accepting connections: %v", err)
	}
	conn.Close()

	// Trigger graceful shutdown with a short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() returned unexpected error: %v", err)
	}

	// After Shutdown, Serve must have returned http.ErrServerClosed.
	select {
	case err := <-srvErr:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("expected http.ErrServerClosed from Serve, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("timed out waiting for server goroutine to exit after Shutdown")
	}
}

// TestGracefulShutdown_InFlightRequestsComplete verifies that a request that
// started before Shutdown() is called is allowed to finish before the server
// stops.
func TestGracefulShutdown_InFlightRequestsComplete(t *testing.T) {
	const requestDelay = 100 * time.Millisecond

	// requestStarted is closed once the handler begins executing, so the test
	// knows it's safe to call Shutdown.
	requestStarted := make(chan struct{})
	// requestDone records the time the handler wrote its response.
	var requestFinished time.Time
	var mu sync.Mutex

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		time.Sleep(requestDelay) // simulate work in progress
		mu.Lock()
		requestFinished = time.Now()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	// Fire the slow request in the background.
	var (
		respStatus int
		clientErr  error
		wg         sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := http.Get("http://" + addr + "/slow") //nolint:noctx
		if err != nil {
			clientErr = err
			return
		}
		_ = resp.Body.Close()
		respStatus = resp.StatusCode
	}()

	// Wait until the handler is executing before triggering shutdown.
	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for in-flight request to start")
	}

	shutdownStarted := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() returned unexpected error: %v", err)
	}

	shutdownFinished := time.Now()

	// Wait for the HTTP client goroutine to finish.
	wg.Wait()

	if clientErr != nil {
		t.Fatalf("in-flight request failed: %v", clientErr)
	}
	if respStatus != http.StatusOK {
		t.Errorf("expected response status 200, got %d", respStatus)
	}

	mu.Lock()
	rf := requestFinished
	mu.Unlock()

	// The handler must have written its response before Shutdown returned.
	if rf.IsZero() {
		t.Error("handler never finished — response was not written")
	}
	if rf.After(shutdownFinished) {
		t.Errorf("handler finished (%v) after Shutdown returned (%v) — in-flight request was not honoured",
			rf.Sub(shutdownStarted), shutdownFinished.Sub(shutdownStarted))
	}
}
