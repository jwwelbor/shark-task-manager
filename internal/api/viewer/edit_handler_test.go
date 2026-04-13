package viewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ----- MockEditServicer -----

// MockEditServicer is a test double for EditServicer.
// Follows the established MockViewerServicer pattern from handler_test.go.
type MockEditServicer struct {
	WriteFileFunc func(ctx context.Context, path string, content string) (*services.WriteFileResult, error)
	// called tracks whether WriteFileFunc was invoked.
	called bool
}

func (m *MockEditServicer) WriteFile(ctx context.Context, path string, content string) (*services.WriteFileResult, error) {
	m.called = true
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(ctx, path, content)
	}
	return nil, errors.New("WriteFileFunc not set in mock")
}

// ----- helpers -----

// putFile issues a PUT /api/v1/edit/file request with the given JSON body.
func putFile(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/edit/file", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// ----- TC-F07-002: success returns 200 with JSON body -----

func TestEditHandler_PutFile_Success(t *testing.T) {
	mock := &MockEditServicer{
		WriteFileFunc: func(ctx context.Context, path string, content string) (*services.WriteFileResult, error) {
			return &services.WriteFileResult{Path: "docs/spec.md", BytesWritten: 12}, nil
		},
	}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"path":"docs/spec.md","content":"hello world\n"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var result services.WriteFileResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Path != "docs/spec.md" {
		t.Errorf("expected path %q, got %q", "docs/spec.md", result.Path)
	}
	if result.BytesWritten != 12 {
		t.Errorf("expected bytes_written 12, got %d", result.BytesWritten)
	}
}

// ----- TC-F07-004: service write error maps to 500 -----

func TestEditHandler_PutFile_WriteError(t *testing.T) {
	mock := &MockEditServicer{
		WriteFileFunc: func(ctx context.Context, path string, content string) (*services.WriteFileResult, error) {
			return nil, fmt.Errorf("write failed: permission denied")
		},
	}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"path":"docs/spec.md","content":"hello"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp["error"] == "" {
		t.Error("expected non-empty error field in response")
	}
}

// ----- TC-F07-008: SecurityError maps to 400 -----

func TestEditHandler_PutFile_PathTraversal(t *testing.T) {
	mock := &MockEditServicer{
		WriteFileFunc: func(ctx context.Context, path string, content string) (*services.WriteFileResult, error) {
			return nil, &services.SecurityError{Path: path}
		},
	}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"path":"../../etc/passwd","content":"evil"}`)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp["error"] != "Forbidden" {
		t.Errorf("expected error %q, got %q", "Forbidden", errResp["error"])
	}
}

// ----- TC-F07-009: body > 2 MiB returns 413 -----

func TestEditHandler_PutFile_BodyTooLarge(t *testing.T) {
	mock := &MockEditServicer{}
	h := NewEditHandler(mock)

	// Build a JSON body that exceeds 2 MiB.
	// Must start with valid JSON syntax so the decoder reads past the limit
	// before hitting a parse error — that way MaxBytesReader fires first.
	// {"path":"docs/spec.md","content":"XXXX...XXX"}
	contentValue := strings.Repeat("a", 2*1024*1024)
	bigJSON := fmt.Sprintf(`{"path":"docs/spec.md","content":%q}`, contentValue)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/edit/file", strings.NewReader(bigJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PutFile(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d; body: %s", rec.Code, rec.Body.String())
	}
	if mock.called {
		t.Error("WriteFile should NOT be called when body exceeds limit")
	}
}

// ----- TC-F07-013: missing `path` field returns 400 -----

func TestEditHandler_PutFile_MissingPath(t *testing.T) {
	mock := &MockEditServicer{}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"content":"hello"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
	if mock.called {
		t.Error("WriteFile should NOT be called when path is missing")
	}
}

// ----- TC-F07-014: missing `content` field returns 400 -----

func TestEditHandler_PutFile_MissingContent(t *testing.T) {
	mock := &MockEditServicer{}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"path":"docs/spec.md"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
	if mock.called {
		t.Error("WriteFile should NOT be called when content is missing")
	}
}

// ----- TC-F07-015: malformed JSON body returns 400 -----

func TestEditHandler_PutFile_MalformedJSON(t *testing.T) {
	mock := &MockEditServicer{}
	h := NewEditHandler(mock)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{bad json`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// ----- TC-F07-018: integration test — real EditService writes file -----

func TestEditHandler_Integration_RealService(t *testing.T) {
	dir := t.TempDir()

	// Create the target file.
	docDir := filepath.Join(dir, "docs")
	if err := os.Mkdir(docDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notesPath := filepath.Join(docDir, "notes.md")
	if err := os.WriteFile(notesPath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	// Wire real EditService → EditHandler.
	svc := services.NewEditService(dir)
	h := NewEditHandler(svc)

	rec := putFile(t, http.HandlerFunc(h.PutFile), `{"path":"docs/notes.md","content":"new content"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	onDisk, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(onDisk) != "new content" {
		t.Errorf("expected on-disk content %q, got %q", "new content", string(onDisk))
	}
}
