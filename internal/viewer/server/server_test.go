package server_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	viewerserver "github.com/jwwelbor/shark-task-manager/internal/viewer/server"
)

// newTestDB creates an in-memory SQLite database suitable for server tests.
// It initializes the schema and returns a *repository.DB and a cleanup func.
func newTestDB(t *testing.T) *repository.DB {
	t.Helper()

	sqlDB, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory test DB: %v", err)
	}
	return repository.NewDB(sqlDB)
}

// listenRandom binds a random localhost port and returns the listener.
// The caller is responsible for closing it if not passed to StartServer.
func listenRandom(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	return ln
}

// TC-012: StartServer closes opts.Ready before blocking on srv.Serve.
// The ready signal must be received without timing out, and the server must
// be accepting connections once the signal fires.
func TestStartServer_TC012_ReadySignal(t *testing.T) {
	repoDB := newTestDB(t)
	ln := listenRandom(t)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			Listener: ln,
			DB:       repoDB,
			Ready:    ready,
		})
	}()

	// Wait for the ready signal — must arrive within 500ms (TC-018 budget).
	select {
	case <-ready:
		// Ready signal received — success.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TC-012: ready signal not received within 500ms")
	}

	// Verify the server is actually accepting connections.
	addr := ln.Addr().String()
	resp, err := http.Get("http://" + addr + "/health") //nolint:noctx
	if err != nil {
		t.Fatalf("TC-012: health check failed after ready signal: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("TC-012: expected /health 200, got %d", resp.StatusCode)
	}

	// Cleanly shut the server down.
	cancel()
	select {
	case err := <-srvDone:
		if err != nil {
			t.Errorf("TC-012: StartServer returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("TC-012: server did not stop within 5s after ctx cancel")
	}
}

// TC-013: Context cancellation triggers graceful shutdown returning nil.
func TestStartServer_TC013_GracefulShutdown(t *testing.T) {
	repoDB := newTestDB(t)
	ln := listenRandom(t)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			Listener: ln,
			DB:       repoDB,
			Ready:    ready,
		})
	}()

	// Wait for server to be ready.
	select {
	case <-ready:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TC-013: server not ready within 500ms")
	}

	// Cancel the context to trigger shutdown.
	cancel()

	// StartServer must return nil within a reasonable time.
	select {
	case err := <-srvDone:
		if err != nil {
			t.Errorf("TC-013: expected nil error on graceful shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("TC-013: server did not complete graceful shutdown within 5s")
	}
}

// TC-014b: StartServer callable with pre-bound DB; does NOT call db.Close()
// on caller-provided DB (the caller retains ownership).
//
// We verify this indirectly: after StartServer returns, the DB must still be
// usable (Ping succeeds). If StartServer had closed it, Ping would fail.
func TestStartServer_TC014b_CallerProvidedDB(t *testing.T) {
	repoDB := newTestDB(t)
	ln := listenRandom(t)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			Listener: ln,
			DB:       repoDB,
			Ready:    ready,
		})
	}()

	// Wait for the ready signal.
	select {
	case <-ready:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TC-014b: server not ready within 500ms")
	}

	// Trigger graceful shutdown.
	cancel()
	select {
	case err := <-srvDone:
		if err != nil {
			t.Errorf("TC-014b: unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TC-014b: server did not stop within 5s")
	}

	// The caller-provided DB must still be usable after StartServer returns.
	if err := repoDB.Ping(); err != nil {
		t.Errorf("TC-014b: caller-provided DB was closed by StartServer (Ping failed): %v", err)
	}

	// Clean up.
	_ = repoDB.Close()
}

// TC-018: Ready signal received within 500ms of StartServer invocation.
func TestStartServer_TC018_ReadyWithin500ms(t *testing.T) {
	repoDB := newTestDB(t)
	ln := listenRandom(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan struct{})
	start := time.Now()

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			Listener: ln,
			DB:       repoDB,
			Ready:    ready,
		})
	}()

	select {
	case <-ready:
		elapsed := time.Since(start)
		if elapsed > 500*time.Millisecond {
			t.Errorf("TC-018: ready signal took %v, expected < 500ms", elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TC-018: ready signal not received within 500ms")
	}

	cancel()
	<-srvDone
}

// TC-F01-006 / TC-F01-008: StartServer mounts the viewer mutation route on the
// local viewer surface and applies the localhost-only CORS wrapper.
func TestStartServer_MutationRoute_ReachableWithLocalCORS(t *testing.T) {
	repoDB := newTestDB(t)
	ln := listenRandom(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan struct{})
	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			Listener: ln,
			DB:       repoDB,
			Ready:    ready,
		})
	}()

	select {
	case <-ready:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("mutation route test: server not ready within 500ms")
	}

	req, err := http.NewRequest(http.MethodPatch, "http://"+ln.Addr().String()+"/api/v1/viewer/epics/E07", nil)
	if err != nil {
		t.Fatalf("mutation route test: failed to create request: %v", err)
	}
	origin := "http://localhost:3000"
	req.Header.Set("Origin", origin)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("mutation route test: PATCH request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		t.Fatalf("mutation route test: expected non-5xx response, got %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("mutation route test: expected 400 for empty PATCH body, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("mutation route test: expected Access-Control-Allow-Origin %q, got %q", origin, got)
	}

	noteReq, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/api/v1/viewer/epics/E07/notes", nil)
	if err != nil {
		t.Fatalf("mutation route test: failed to create POST note request: %v", err)
	}
	noteReq.Header.Set("Origin", origin)
	noteResp, err := client.Do(noteReq)
	if err != nil {
		t.Fatalf("mutation route test: POST note request failed: %v", err)
	}
	defer noteResp.Body.Close()
	if noteResp.StatusCode == http.StatusInternalServerError {
		t.Fatalf("mutation route test: expected non-5xx response for note route, got %d", noteResp.StatusCode)
	}
	if got := noteResp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("mutation route test: expected note ACAO %q, got %q", origin, got)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, "http://"+ln.Addr().String()+"/api/v1/viewer/tasks/T-E07-F01-001/relationships/depends_on/E07-F01", nil)
	if err != nil {
		t.Fatalf("mutation route test: failed to create DELETE request: %v", err)
	}
	deleteReq.Header.Set("Origin", origin)
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("mutation route test: DELETE request failed: %v", err)
	}
	defer deleteResp.Body.Close()
	if deleteResp.StatusCode == http.StatusInternalServerError {
		t.Fatalf("mutation route test: expected non-5xx response for delete route, got %d", deleteResp.StatusCode)
	}
	if got := deleteResp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("mutation route test: expected delete ACAO %q, got %q", origin, got)
	}

	cancel()
	select {
	case err := <-srvDone:
		if err != nil {
			t.Fatalf("mutation route test: StartServer returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("mutation route test: server did not stop within 5s")
	}
}

// TestStartServer_DefaultAddr verifies that when no Listener and no Addr is
// provided, StartServer binds to ":8080" by default.
// NOTE: This test is skipped if port 8080 is already in use to avoid flakiness
// in CI environments.
func TestStartServer_DefaultAddr(t *testing.T) {
	// Pre-check: if :8080 is already in use, skip.
	probe, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Skip("port 8080 already in use; skipping default-addr test")
	}
	probe.Close()

	repoDB := newTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- viewerserver.StartServer(ctx, viewerserver.Options{
			DB:    repoDB,
			Ready: ready,
		})
	}()

	select {
	case <-ready:
		// Server bound to default addr.
	case <-time.After(1 * time.Second):
		t.Fatal("server did not become ready within 1s")
	}

	cancel()
	<-srvDone
}
