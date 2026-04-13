package viewer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// innerCalled is a sentinel handler that records whether it was invoked.
func innerHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

// TC-006: localhost origin is echoed in ACAO header; inner handler is called;
// Allow-Methods and Allow-Headers are set on non-OPTIONS requests too.
func TestWithLocalCORS_LocalhostOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected ACAO = %q, got %q", "http://localhost:3000", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary = %q, got %q", "Origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, PUT, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods = %q, got %q", "GET, PUT, OPTIONS", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers = %q, got %q", "Content-Type", got)
	}
}

// TC-007: 127.0.0.1 origin is echoed in ACAO header; Allow-Methods, Allow-Headers,
// and Vary are also set on non-OPTIONS requests.
func TestWithLocalCORS_127001Origin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:8080" {
		t.Errorf("expected ACAO = %q, got %q", "http://127.0.0.1:8080", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary = %q, got %q", "Origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, PUT, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods = %q, got %q", "GET, PUT, OPTIONS", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers = %q, got %q", "Content-Type", got)
	}
}

// TC-008: HTTPS localhost origin is echoed.
func TestWithLocalCORS_HTTPSLocalhostOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://localhost")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://localhost" {
		t.Errorf("expected ACAO = %q, got %q", "https://localhost", got)
	}
}

// TC-009: External origin — no ACAO header set; inner handler still called.
func TestWithLocalCORS_ExternalOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called for external origin")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for external origin, got %q", got)
	}
}

// TC-010b: OPTIONS with no Origin header — 204, no CORS headers, inner handler NOT called.
func TestWithLocalCORS_OptionsNoOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler must NOT be called for OPTIONS request")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header when Origin absent, got %q", got)
	}
}

// TC-010: Empty origin — no ACAO header set; inner handler called.
func TestWithLocalCORS_EmptyOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called when no Origin header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header when Origin absent, got %q", got)
	}
}

// TC-011: OPTIONS preflight from localhost origin — 204, correct headers, inner handler NOT called.
func TestWithLocalCORS_PreflightLocalhost(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler must NOT be called for OPTIONS preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected ACAO = %q, got %q", "http://localhost:3000", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods header to be set")
	}
}

// TC-F07-011: PUT from localhost origin — inner handler called; Allow-Methods includes "PUT".
func TestWithLocalCORS_PUTLocalhost(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called for PUT request")
	}
	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowMethods, "PUT") {
		t.Errorf("expected Access-Control-Allow-Methods to contain %q, got %q", "PUT", allowMethods)
	}
}

// TC-F07-012: OPTIONS preflight for PUT from localhost — 204; Allow-Methods contains "PUT"; inner NOT called.
func TestWithLocalCORS_PreflightPUT(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "PUT")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler must NOT be called for OPTIONS preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowMethods, "PUT") {
		t.Errorf("expected Access-Control-Allow-Methods to contain %q, got %q", "PUT", allowMethods)
	}
}

// TC-011b: OPTIONS preflight from external origin — 204, no ACAO header, inner handler NOT called.
func TestWithLocalCORS_PreflightExternalOrigin(t *testing.T) {
	called := false
	handler := WithLocalCORS(innerHandler(&called))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler must NOT be called for external-origin OPTIONS preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for external OPTIONS preflight, got %q", got)
	}
}
