package maintainer

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// gateTracerName is the name used for the maintainer gate OTel tracer.
// The tracer is retrieved lazily per-call via otel.Tracer() so that tests can
// set the global tracer provider before invoking gate methods.
const gateTracerName = "internal/auth/maintainer"

// tracer returns the OTel tracer for this package, fetched from the current
// global provider. Lazy lookup allows tests to install a recording provider
// before calling Authorize / RecordSuccess.
func tracer() trace.Tracer {
	return otel.Tracer(gateTracerName)
}

// defaultWindow is the default authorization cache window when no window is configured.
const defaultWindow = 60 * time.Second

// Gate is the authorization primitive for maintainer-gated admin commands.
// Future admin commands consume this interface. See doc.go for the ten-line
// adoption example.
//
// Spec reference: spec.md §2.2, REQ-F-001, REQ-F-010.
type Gate interface {
	// Authorize verifies either an explicit password or a live cache entry.
	// Returns nil on success or *UnauthorizedError on any authorization failure.
	// Infrastructure failures (disk I/O on the cache file) are treated as cache
	// misses per REQ-NF-003 and never surface as non-UnauthorizedError errors.
	Authorize(ctx context.Context, providedPass string) error

	// RecordSuccess extends the sudo-style authorization window. Callers typically
	// invoke it after a gated operation completes. Returns an error only on hard
	// I/O failures; callers SHOULD treat the return as best-effort.
	RecordSuccess(ctx context.Context) error
}

// FileGate is the filesystem-backed implementation of Gate.
// It verifies passwords using SHA-256 with crypto/subtle.ConstantTimeCompare,
// and caches successful authorizations in a per-project session file under the
// XDG cache directory.
//
// Spec reference: spec.md §2.2 (package shape), REQ-F-002 through REQ-F-006.
type FileGate struct {
	projectRoot string
	cfg         *config.MaintainerConfig
	window      time.Duration
	clk         clock
}

// NewFileGate constructs a FileGate. A zero window is replaced with the default
// of 60 seconds. Passing nil cfg creates a gate that always returns
// *UnauthorizedError with the "missing_config" reason.
//
// Spec reference: spec.md §2.2, REQ-F-005.
func NewFileGate(projectRoot string, cfg *config.MaintainerConfig, window time.Duration) *FileGate {
	w := window
	if w <= 0 {
		w = defaultWindow
	}
	return &FileGate{
		projectRoot: projectRoot,
		cfg:         cfg,
		window:      w,
		clk:         realClock{},
	}
}

// Authorize checks whether the provided password or a live cache entry grants
// access. It returns nil on success or *UnauthorizedError on failure.
//
// Decision order:
//  1. If no password is configured (nil cfg or empty PasswordHash), fail with "missing_config".
//  2. If providedPass is non-empty, compare its SHA-256 digest to the configured hash.
//     Return nil on match, "wrong_password" on mismatch.
//  3. Otherwise check the cache file for a live entry whose pass_hash matches the
//     configured hash and whose last_success is within the configured window.
//
// Spec reference: spec.md REQ-F-002, REQ-F-006, REQ-NF-001, REQ-NF-002.
func (g *FileGate) Authorize(ctx context.Context, providedPass string) (retErr error) {
	_, span := tracer().Start(ctx, "maintainer.authorize")
	defer func() {
		authorized := retErr == nil
		span.SetAttributes(attribute.Bool("maintainer.authorized", authorized))
		span.End()
	}()

	// Step 1: Check config exists and has a password hash.
	if g.cfg == nil || g.cfg.PasswordHash == "" {
		return &UnauthorizedError{Reason: "missing_config"}
	}

	configHash := g.cfg.PasswordHash

	// Step 2: Explicit password provided — verify by SHA-256.
	if providedPass != "" {
		digest := sha256hex(providedPass)
		// Use ConstantTimeCompare to prevent timing oracle attacks (REQ-F-006, REQ-NF-002).
		if subtle.ConstantTimeCompare([]byte(digest), []byte(configHash)) == 1 {
			return nil
		}
		return &UnauthorizedError{Reason: "wrong_password"}
	}

	// Step 3: No explicit password — check the cache file.
	path, err := sessionPath(g.projectRoot)
	if err != nil {
		// Can't determine cache path; treat as cache miss.
		return &UnauthorizedError{Reason: "expired_cache"}
	}

	entry := readSession(path)
	if entry == nil {
		// No cache or malformed cache — cache miss.
		return &UnauthorizedError{Reason: "expired_cache"}
	}

	// Verify that the cached pass_hash matches the currently configured hash.
	// This catches password rotation (AC-6, REQ-F-002, REQ-F-003).
	if subtle.ConstantTimeCompare([]byte(entry.PassHash), []byte(configHash)) != 1 {
		return &UnauthorizedError{Reason: "hash_mismatch_after_rotation"}
	}

	// Verify the cache entry is within the window (REQ-F-005).
	// An entry is expired when elapsed >= window (the window is "strictly less than"
	// the elapsed time must be for the cache to be valid — i.e. elapsed < window).
	elapsed := g.clk.Now().Sub(entry.LastSuccess)
	if elapsed >= g.window {
		return &UnauthorizedError{Reason: "expired_cache"}
	}

	return nil
}

// RecordSuccess writes a cache entry with last_success = now and pass_hash equal
// to the currently configured password_hash. The write uses the atomic
// temp-file + rename pattern (spec F02-D2).
//
// Callers SHOULD treat the returned error as best-effort; a write failure does
// not indicate an authorization problem.
//
// Spec reference: spec.md REQ-F-003, REQ-NF-001.
func (g *FileGate) RecordSuccess(ctx context.Context) (retErr error) {
	_, span := tracer().Start(ctx, "maintainer.record_success")
	defer func() { span.End() }()

	if g.cfg == nil || g.cfg.PasswordHash == "" {
		// Nothing to record if no password is configured.
		return nil
	}

	path, err := sessionPath(g.projectRoot)
	if err != nil {
		return fmt.Errorf("maintainer: resolve session path: %w", err)
	}

	entry := &sessionEntry{
		LastSuccess: g.clk.Now(),
		PassHash:    g.cfg.PasswordHash,
	}
	return writeSession(path, entry)
}

// sha256hex returns the lowercase-hex SHA-256 digest of s.
// This is the hashing function used for password comparison.
//
// Spec reference: spec.md REQ-F-006, ADR-6 (SHA-256, no salt).
func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

// Ensure FileGate satisfies the Gate interface at compile time.
var _ Gate = (*FileGate)(nil)

// withClock returns a copy of the FileGate with a different clock.
// This is package-private and used only in tests to inject a fake clock.
func (g *FileGate) withClock(c clock) *FileGate {
	cp := *g
	cp.clk = c
	return &cp
}
