package maintainer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// ---- Test helpers -------------------------------------------------------

// sha256hexHelper computes a SHA-256 hex digest for use in test assertions.
func sha256hexHelper(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

// fakeClock is an injectable clock for tests that need to control time.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

func (f *fakeClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

// newTestGate builds a FileGate with isolated temp dirs and an optional fake clock.
// If fc is nil, a realClock is used.
func newTestGate(t *testing.T, cfg *config.MaintainerConfig, window time.Duration, fc *fakeClock) *FileGate {
	t.Helper()
	projectRoot := t.TempDir()
	xdgRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdgRoot)

	g := NewFileGate(projectRoot, cfg, window)
	if fc != nil {
		g = g.withClock(fc)
	}
	return g
}

// writeFakeSessionFile writes arbitrary bytes to the expected session file path,
// creating parent directories as needed. Used for AC-9 malformed cache tests.
func writeFakeSessionFile(t *testing.T, g *FileGate, content string) {
	t.Helper()
	path, err := sessionPath(g.projectRoot)
	if err != nil {
		t.Fatalf("sessionPath: %v", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// newRecordingGate creates a FileGate using a test OTel tracer provider that
// records spans. Returns the gate and the span recorder.
// The global OTel provider is temporarily replaced and restored after the test.
func newRecordingGate(t *testing.T, cfg *config.MaintainerConfig, window time.Duration) (*FileGate, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// Save and restore the global tracer provider.
	// Since tracer() calls otel.Tracer() lazily, setting the global provider
	// before the test ensures our recording provider is used.
	origProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(origProvider)
		_ = tp.Shutdown(context.Background())
	})

	gate := newTestGate(t, cfg, window, nil)
	return gate, sr
}

// ---- AC-1: Correct password, empty cache --------------------------------

func TestFileGate_Authorize_CorrectPassword_EmptyCache(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate := newTestGate(t, cfg, 60*time.Second, nil)

	ctx := context.Background()

	t.Run("correct password returns nil", func(t *testing.T) {
		if err := gate.Authorize(ctx, "hunter2"); err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("single byte password", func(t *testing.T) {
		h := sha256hexHelper("x")
		g2 := newTestGate(t, &config.MaintainerConfig{PasswordHash: h}, 60*time.Second, nil)
		if err := g2.Authorize(ctx, "x"); err != nil {
			t.Errorf("expected nil for single-byte password, got: %v", err)
		}
	})

	t.Run("unicode password", func(t *testing.T) {
		h := sha256hexHelper("pässwörd!")
		g2 := newTestGate(t, &config.MaintainerConfig{PasswordHash: h}, 60*time.Second, nil)
		if err := g2.Authorize(ctx, "pässwörd!"); err != nil {
			t.Errorf("expected nil for unicode password, got: %v", err)
		}
	})

	t.Run("case sensitive — wrong case fails", func(t *testing.T) {
		err := gate.Authorize(ctx, "Hunter2")
		if err == nil {
			t.Error("expected error for wrong-case password")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Errorf("expected *UnauthorizedError, got %T", err)
		}
	})

	t.Run("shell metacharacters in password", func(t *testing.T) {
		h := sha256hexHelper("'; DROP TABLE")
		g2 := newTestGate(t, &config.MaintainerConfig{PasswordHash: h}, 60*time.Second, nil)
		if err := g2.Authorize(ctx, "'; DROP TABLE"); err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})
}

// ---- AC-2: Wrong password, empty cache ----------------------------------

func TestFileGate_Authorize_WrongPassword_EmptyCache(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate := newTestGate(t, cfg, 60*time.Second, nil)
	ctx := context.Background()

	t.Run("wrong password returns UnauthorizedError", func(t *testing.T) {
		err := gate.Authorize(ctx, "wrong")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Fatalf("expected *UnauthorizedError, got %T: %v", err, err)
		}
		if uaErr.Reason != "wrong_password" {
			t.Errorf("expected reason 'wrong_password', got %q", uaErr.Reason)
		}
		if uaErr.Error() == "" {
			t.Error("Error() should not be empty")
		}
	})

	t.Run("empty string with configured hash and no cache", func(t *testing.T) {
		err := gate.Authorize(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty password with no cache")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Errorf("expected *UnauthorizedError, got %T", err)
		}
	})

	t.Run("providing the hash string itself as password fails", func(t *testing.T) {
		err := gate.Authorize(ctx, hash)
		if err == nil {
			t.Error("expected error: providing the hash is not the password")
		}
	})
}

// ---- AC-3: Nil config returns *UnauthorizedError with set-password hint -

func TestFileGate_Authorize_NilConfig_ReturnsHint(t *testing.T) {
	gate := newTestGate(t, nil, 0, nil)
	ctx := context.Background()

	t.Run("nil config returns UnauthorizedError with hint", func(t *testing.T) {
		err := gate.Authorize(ctx, "anything")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Fatalf("expected *UnauthorizedError, got %T", err)
		}
		if uaErr.Reason != "missing_config" {
			t.Errorf("expected reason 'missing_config', got %q", uaErr.Reason)
		}
		msg := uaErr.Error()
		if !strings.Contains(msg, "shark admin maintainer set-password") {
			t.Errorf("Error() must contain 'shark admin maintainer set-password', got: %q", msg)
		}
	})

	t.Run("non-nil config with empty PasswordHash returns missing_config", func(t *testing.T) {
		g2 := newTestGate(t, &config.MaintainerConfig{PasswordHash: ""}, 0, nil)
		err := g2.Authorize(ctx, "anything")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Fatalf("expected *UnauthorizedError, got %T", err)
		}
		if uaErr.Reason != "missing_config" {
			t.Errorf("expected reason 'missing_config', got %q", uaErr.Reason)
		}
		if !strings.Contains(uaErr.Error(), "shark admin maintainer set-password") {
			t.Errorf("Error() must contain set-password hint, got: %q", uaErr.Error())
		}
	})

	t.Run("multiple calls are deterministic", func(t *testing.T) {
		err1 := gate.Authorize(ctx, "a")
		err2 := gate.Authorize(ctx, "b")
		if err1.Error() != err2.Error() {
			t.Errorf("messages differ: %q vs %q", err1.Error(), err2.Error())
		}
	})
}

// ---- AC-4: Cache hit within window with empty password returns nil -------

func TestFileGate_Authorize_CacheHit_WithinWindow(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fc := newFakeClock(t0)
	gate := newTestGate(t, cfg, 60*time.Second, fc)
	ctx := context.Background()

	// Authorize with correct password and record success.
	if err := gate.Authorize(ctx, "hunter2"); err != nil {
		t.Fatalf("initial Authorize failed: %v", err)
	}
	if err := gate.RecordSuccess(ctx); err != nil {
		t.Fatalf("RecordSuccess failed: %v", err)
	}

	t.Run("within window — 59s — returns nil", func(t *testing.T) {
		fc.Set(t0.Add(59 * time.Second))
		if err := gate.Authorize(ctx, ""); err != nil {
			t.Errorf("expected nil within window, got: %v", err)
		}
	})

	t.Run("at exactly 60s boundary — fails (strict >)", func(t *testing.T) {
		fc.Set(t0.Add(60 * time.Second))
		err := gate.Authorize(ctx, "")
		if err == nil {
			t.Error("expected error at exact boundary (window is strict >)")
		}
	})

	t.Run("1ns within window — succeeds", func(t *testing.T) {
		fc.Set(t0.Add(1 * time.Nanosecond))
		if err := gate.Authorize(ctx, ""); err != nil {
			t.Errorf("expected nil at 1ns, got: %v", err)
		}
	})

	t.Run("empty password with valid cache succeeds", func(t *testing.T) {
		fc.Set(t0.Add(30 * time.Second))
		if err := gate.Authorize(ctx, ""); err != nil {
			t.Errorf("expected nil with empty password and valid cache, got: %v", err)
		}
	})
}

// ---- AC-5: Cache entry expired at 61s returns *UnauthorizedError ---------

func TestFileGate_Authorize_CacheExpired(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fc := newFakeClock(t0)
	gate := newTestGate(t, cfg, 60*time.Second, fc)
	ctx := context.Background()

	// Authorize and record success.
	if err := gate.Authorize(ctx, "hunter2"); err != nil {
		t.Fatalf("initial Authorize: %v", err)
	}
	if err := gate.RecordSuccess(ctx); err != nil {
		t.Fatalf("RecordSuccess: %v", err)
	}

	t.Run("61s after success — expired", func(t *testing.T) {
		fc.Set(t0.Add(61 * time.Second))
		err := gate.Authorize(ctx, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Fatalf("expected *UnauthorizedError, got %T", err)
		}
		if uaErr.Reason != "expired_cache" {
			t.Errorf("expected 'expired_cache', got %q", uaErr.Reason)
		}
	})

	t.Run("exactly 60s — expired (strict boundary)", func(t *testing.T) {
		fc.Set(t0.Add(60 * time.Second))
		err := gate.Authorize(ctx, "")
		if err == nil {
			t.Error("expected error at exact 60s boundary")
		}
	})

	t.Run("expired cache but correct explicit password — password wins", func(t *testing.T) {
		fc.Set(t0.Add(120 * time.Second))
		// Explicit correct password should still succeed despite expired cache.
		if err := gate.Authorize(ctx, "hunter2"); err != nil {
			t.Errorf("expected nil with explicit correct password, got: %v", err)
		}
	})
}

// ---- AC-6: Rotated password_hash invalidates existing cache -------------

func TestFileGate_Authorize_PasswordRotation_InvalidatesCache(t *testing.T) {
	oldHash := sha256hexHelper("oldpass")
	newHash := sha256hexHelper("newpass")
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fc := newFakeClock(t0)

	// Build gate with old config.
	xdgRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdgRoot)
	projectRoot := t.TempDir()

	oldCfg := &config.MaintainerConfig{PasswordHash: oldHash}
	oldGate := NewFileGate(projectRoot, oldCfg, 60*time.Second)
	oldGate = oldGate.withClock(fc)

	ctx := context.Background()

	// Write cache with old hash.
	if err := oldGate.Authorize(ctx, "oldpass"); err != nil {
		t.Fatalf("old gate Authorize: %v", err)
	}
	if err := oldGate.RecordSuccess(ctx); err != nil {
		t.Fatalf("old gate RecordSuccess: %v", err)
	}

	// Rotate password: build new gate (same project root, new hash, same clock).
	newCfg := &config.MaintainerConfig{PasswordHash: newHash}
	newGate := NewFileGate(projectRoot, newCfg, 60*time.Second)
	newGate = newGate.withClock(fc)

	// Advance clock by 5s (well within window, only rotation explains failure).
	fc.Advance(5 * time.Second)

	t.Run("old cache no longer valid after rotation", func(t *testing.T) {
		err := newGate.Authorize(ctx, "")
		if err == nil {
			t.Fatal("expected error after password rotation, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Fatalf("expected *UnauthorizedError, got %T", err)
		}
		if uaErr.Reason != "hash_mismatch_after_rotation" {
			t.Errorf("expected 'hash_mismatch_after_rotation', got %q", uaErr.Reason)
		}
	})

	t.Run("new password works explicitly", func(t *testing.T) {
		if err := newGate.Authorize(ctx, "newpass"); err != nil {
			t.Errorf("expected nil with new password, got: %v", err)
		}
	})
}

// ---- AC-7: Cache file at correct XDG path with correct permissions ------

func TestFileGate_CachePath_XDGOverride_Permissions(t *testing.T) {
	xdgRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdgRoot)

	projectRoot := t.TempDir()
	// Use absolute path.
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate := NewFileGate(absRoot, cfg, 60*time.Second)
	ctx := context.Background()

	// Trigger cache write.
	if err := gate.RecordSuccess(ctx); err != nil {
		t.Fatalf("RecordSuccess: %v", err)
	}

	expectedHash := projectHash(absRoot)
	expectedDir := filepath.Join(xdgRoot, "shark", expectedHash)
	expectedFile := filepath.Join(expectedDir, "maintainer.session")

	t.Run("session file exists at correct path", func(t *testing.T) {
		if _, err := os.Stat(expectedFile); err != nil {
			t.Errorf("session file not found at %s: %v", expectedFile, err)
		}
	})

	t.Run("session file mode is 0600", func(t *testing.T) {
		info, err := os.Stat(expectedFile)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		mode := info.Mode().Perm()
		if mode != 0o600 {
			t.Errorf("session file mode = %o, want 0600", mode)
		}
	})

	t.Run("per-project directory mode is 0700", func(t *testing.T) {
		info, err := os.Stat(expectedDir)
		if err != nil {
			t.Fatalf("Stat dir: %v", err)
		}
		mode := info.Mode().Perm()
		if mode != 0o700 {
			t.Errorf("directory mode = %o, want 0700", mode)
		}
	})
}

// ---- AC-8: ConstantTimeCompare is used (import assertion) ---------------

func TestPackage_UsesConstantTimeCompare(t *testing.T) {
	// AC-T8 (task spec): This test verifies via go list -json that crypto/subtle
	// is imported by the production package (not just test code), documenting the
	// design invariant that ConstantTimeCompare is on the authorize path.
	//
	// Spec reference: spec.md REQ-F-006, REQ-NF-002; test-plan.md AC-8.

	// Static import assertion via go list -json: verify crypto/subtle is in
	// the production imports (not just in test imports).
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = packageDir()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -json failed: %v\noutput: %s", err, out)
	}
	var result struct {
		Imports []string `json:"Imports"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput: %s", err, out)
	}
	subtleFound := false
	for _, imp := range result.Imports {
		if imp == "crypto/subtle" {
			subtleFound = true
			break
		}
	}
	if !subtleFound {
		t.Errorf("crypto/subtle not found in production imports; ConstantTimeCompare may not be used on the comparison path. Imports: %v", result.Imports)
	}

	// Functional test: correct and incorrect passwords exercise the comparison path.
	// The compilation of the package verifies the import is actually used.
	hash := sha256hexHelper("testpass")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate := newTestGate(t, cfg, 60*time.Second, nil)
	ctx := context.Background()

	// Correct password should succeed.
	if err := gate.Authorize(ctx, "testpass"); err != nil {
		t.Errorf("correct password failed: %v", err)
	}
	// Wrong password should fail.
	if err := gate.Authorize(ctx, "notright"); err == nil {
		t.Error("wrong password should fail")
	}
}

// ---- AC-9: Malformed cache file treated as cache miss -------------------

func TestFileGate_Authorize_MalformedCacheFile_TreatedAsMiss(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	ctx := context.Background()

	cases := []struct {
		name    string
		content string
	}{
		{"not JSON at all", "NOT JSON AT ALL"},
		{"empty file", ""},
		{"valid JSON missing fields", "{}"},
		{"truncated JSON", `{"last_suc`},
		{"wrong types in JSON", `{"last_success": 12345, "pass_hash": []}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate := newTestGate(t, cfg, 60*time.Second, nil)
			writeFakeSessionFile(t, gate, tc.content)

			err := gate.Authorize(ctx, "")
			if err == nil {
				t.Fatal("expected error (cache miss), got nil")
			}
			var uaErr *UnauthorizedError
			if !errors.As(err, &uaErr) {
				t.Errorf("expected *UnauthorizedError, got %T: %v", err, err)
			}
		})
	}

	t.Run("file is a directory", func(t *testing.T) {
		gate := newTestGate(t, cfg, 60*time.Second, nil)
		path, err := sessionPath(gate.projectRoot)
		if err != nil {
			t.Fatalf("sessionPath: %v", err)
		}
		// Create a directory at the session file path.
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		err = gate.Authorize(ctx, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var uaErr *UnauthorizedError
		if !errors.As(err, &uaErr) {
			t.Errorf("expected *UnauthorizedError, got %T: %v", err, err)
		}
	})
}

// ---- AC-10: Authorize emits one span with bool attribute, no secret -----

func TestFileGate_Authorize_SpanAttributes_NoSecretLeakage(t *testing.T) {
	password := "super-secret-xyz"
	hash := sha256hexHelper(password)
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate, sr := newRecordingGate(t, cfg, 60*time.Second)
	ctx := context.Background()

	t.Run("successful authorize: span has authorized=true, no secret", func(t *testing.T) {
		sr.Reset()
		if err := gate.Authorize(ctx, password); err != nil {
			t.Fatalf("Authorize failed: %v", err)
		}

		spans := tracetest.SpanStubsFromReadOnlySpans(sr.Ended())
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if span.Name != "maintainer.authorize" {
			t.Errorf("span name = %q, want 'maintainer.authorize'", span.Name)
		}

		// Check that authorized=true is present.
		authorizedFound := false
		for _, a := range span.Attributes {
			if string(a.Key) == "maintainer.authorized" {
				authorizedFound = true
				if a.Value.AsBool() != true {
					t.Errorf("maintainer.authorized = %v, want true", a.Value.AsBool())
				}
			}
		}
		if !authorizedFound {
			t.Error("span missing 'maintainer.authorized' attribute")
		}

		// Assert no secret in span data.
		serialized := serializeSpanData(spans)
		if strings.Contains(serialized, password) {
			t.Errorf("password found in span data: %s", serialized)
		}
		if strings.Contains(serialized, hash) {
			t.Errorf("password hash found in span data: %s", serialized)
		}
	})

	t.Run("failed authorize: span has authorized=false", func(t *testing.T) {
		sr.Reset()
		_ = gate.Authorize(ctx, "wrongpassword")

		spans := tracetest.SpanStubsFromReadOnlySpans(sr.Ended())
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		for _, a := range span.Attributes {
			if string(a.Key) == "maintainer.authorized" {
				if a.Value.AsBool() != false {
					t.Errorf("maintainer.authorized = %v, want false", a.Value.AsBool())
				}
			}
		}
	})

	t.Run("RecordSuccess emits record_success span", func(t *testing.T) {
		sr.Reset()
		_ = gate.RecordSuccess(ctx)

		spans := tracetest.SpanStubsFromReadOnlySpans(sr.Ended())
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		if spans[0].Name != "maintainer.record_success" {
			t.Errorf("span name = %q, want 'maintainer.record_success'", spans[0].Name)
		}
		// Verify no hash in record_success span.
		serialized := serializeSpanData(spans)
		if strings.Contains(serialized, hash) {
			t.Errorf("hash found in record_success span: %s", serialized)
		}
	})
}

// serializeSpanData converts span stubs to a string for substring inspection.
func serializeSpanData(spans tracetest.SpanStubs) string {
	var sb strings.Builder
	for _, s := range spans {
		sb.WriteString(s.Name)
		sb.WriteString(" ")
		for _, a := range s.Attributes {
			sb.WriteString(fmt.Sprintf("%s=%v ", a.Key, a.Value.AsInterface()))
		}
		for _, e := range s.Events {
			sb.WriteString(e.Name)
			sb.WriteString(" ")
			for _, a := range e.Attributes {
				sb.WriteString(fmt.Sprintf("%s=%v ", a.Key, a.Value.AsInterface()))
			}
		}
	}
	return sb.String()
}

// ---- AC-14: No shark-domain imports ------------------------------------

// goListImports runs "go list -json ." in dir and returns the Imports slice.
// It fails the test if the command exits non-zero or the JSON cannot be parsed.
func goListImports(t *testing.T, dir string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -json failed in %s: %v\noutput: %s", dir, err, out)
	}
	var result struct {
		Imports     []string `json:"Imports"`
		TestImports []string `json:"TestImports"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("json.Unmarshal go list output: %v\noutput: %s", err, out)
	}
	all := append(result.Imports, result.TestImports...)
	return all
}

// packageDir returns the directory of the current test's package.
// Since tests run with the package directory as cwd, this returns ".".
// We use this to ensure the go list command targets the right package.
func packageDir() string {
	return "."
}

func TestPackage_HasNoSharkDomainImports(t *testing.T) {
	// AC-T5 (task spec): This test uses go list -json at test time to assert
	// that forbidden shark-domain import prefixes are absent from the package.
	//
	// Forbidden import prefixes (spec.md REQ-F-001, test-plan.md AC-14):
	//   - github.com/jwwelbor/shark-task-manager/internal/models
	//   - github.com/jwwelbor/shark-task-manager/internal/repository
	//   - github.com/jwwelbor/shark-task-manager/internal/services
	//   - github.com/jwwelbor/shark-task-manager/internal/workflow
	//   - github.com/jwwelbor/shark-task-manager/internal/cli
	//
	// Allowed:
	//   - github.com/jwwelbor/shark-task-manager/internal/config (for MaintainerConfig)
	//   - Standard library
	//   - go.opentelemetry.io/otel (observability)

	// Compile-time assertion: the config import is allowed and required.
	var _ *config.MaintainerConfig = nil

	// Runtime assertion via go list -json: parse actual imports and check
	// for forbidden prefixes. This guards against future additions.
	forbidden := []string{
		"github.com/jwwelbor/shark-task-manager/internal/models",
		"github.com/jwwelbor/shark-task-manager/internal/repository",
		"github.com/jwwelbor/shark-task-manager/internal/services",
		"github.com/jwwelbor/shark-task-manager/internal/workflow",
		"github.com/jwwelbor/shark-task-manager/internal/cli",
	}

	imports := goListImports(t, packageDir())
	for _, imp := range imports {
		for _, prefix := range forbidden {
			if strings.HasPrefix(imp, prefix) {
				t.Errorf("forbidden import found: %q (matches forbidden prefix %q)", imp, prefix)
			}
		}
	}
}

// ---- Section 2.1: Concurrent RecordSuccess — Race Safety ----------------

func TestFileGate_RecordSuccess_Concurrent(t *testing.T) {
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fc := newFakeClock(t0)
	gate := newTestGate(t, cfg, 60*time.Second, fc)
	ctx := context.Background()

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = gate.RecordSuccess(ctx)
		}()
	}
	wg.Wait()

	// Verify the final session file is valid JSON with plausible last_success.
	path, err := sessionPath(gate.projectRoot)
	if err != nil {
		t.Fatalf("sessionPath: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var entry sessionEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("final session file is not valid JSON: %v\ncontent: %s", err, data)
	}
	if entry.LastSuccess.IsZero() {
		t.Error("last_success should not be zero")
	}

	// Verify no leftover .tmp files.
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover tmp file: %s", e.Name())
		}
	}
}

// ---- Section 2.2: XDG Precedence Over UserCacheDir ---------------------

func TestCachePath_XDGOverrides_HomeCache(t *testing.T) {
	xdgRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdgRoot)

	projectRoot := t.TempDir()
	hash := sha256hexHelper("x")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	gate := NewFileGate(projectRoot, cfg, 60*time.Second)
	ctx := context.Background()

	if err := gate.RecordSuccess(ctx); err != nil {
		t.Fatalf("RecordSuccess: %v", err)
	}

	path, err := sessionPath(gate.projectRoot)
	if err != nil {
		t.Fatalf("sessionPath: %v", err)
	}
	if !strings.HasPrefix(path, xdgRoot) {
		t.Errorf("session path %q does not start with XDG_CACHE_HOME %q", path, xdgRoot)
	}
}

// TestCachePath_XDGEmptyString_FallsBackToUserCacheDir verifies the AC-7 edge case:
// when XDG_CACHE_HOME is set to an empty string it is treated as unset and the
// fallback os.UserCacheDir() path is used instead.
//
// Spec reference: spec.md REQ-F-004; test-plan.md AC-7 edge cases.
func TestCachePath_XDGEmptyString_FallsBackToUserCacheDir(t *testing.T) {
	// Set XDG_CACHE_HOME to empty string — must behave as if unset.
	t.Setenv("XDG_CACHE_HOME", "")

	projectRoot := t.TempDir()

	path, err := sessionPath(projectRoot)
	if err != nil {
		// On CI or sandboxed environments os.UserCacheDir may fail; skip in that case.
		t.Skipf("sessionPath returned error (UserCacheDir unavailable?): %v", err)
	}

	// The path must NOT start with an empty string prefix (which would trivially match
	// anything). It must start with the actual UserCacheDir value.
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("os.UserCacheDir not available: %v", err)
	}

	if !strings.HasPrefix(path, userCacheDir) {
		t.Errorf("with XDG_CACHE_HOME='', session path %q should start with UserCacheDir %q", path, userCacheDir)
	}
}

// ---- Section 2.3: Project Hash Isolation --------------------------------

func TestCachePath_DifferentProjects_DifferentHashes(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	hash1 := projectHash(root1)
	hash2 := projectHash(root2)

	if hash1 == hash2 {
		t.Errorf("different project roots produced the same hash: %s", hash1)
	}

	// Verify session paths are distinct.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path1, err := sessionPath(root1)
	if err != nil {
		t.Fatalf("sessionPath 1: %v", err)
	}
	path2, err := sessionPath(root2)
	if err != nil {
		t.Fatalf("sessionPath 2: %v", err)
	}
	if path1 == path2 {
		t.Errorf("different project roots produced the same session path: %s", path1)
	}
}

// ---- INT-4: Future-Consumer Pattern Compiles and Works ------------------

func TestFutureConsumerPattern_Compiles(t *testing.T) {
	// Verifies that the ten-line adoption pattern from doc.go compiles.
	// Assigns *FileGate to Gate interface variable and calls both methods.
	hash := sha256hexHelper("hunter2")
	cfg := &config.MaintainerConfig{PasswordHash: hash}
	projectRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	// Pattern from doc.go / spec §1.1 REQ-F-010:
	var gate Gate = NewFileGate(projectRoot, cfg, 60*time.Second)

	ctx := context.Background()
	// Both methods are on the Gate interface.
	_ = gate.Authorize(ctx, "hunter2")
	_ = gate.RecordSuccess(ctx)
}
