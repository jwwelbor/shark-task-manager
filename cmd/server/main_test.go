package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
