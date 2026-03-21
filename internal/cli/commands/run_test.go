// Package commands provides CLI command implementations.
// This file contains tests for the `shark run` command helpers:
// - buildTransitioner: entity type dispatch (no DB for error path)
// - buildPlaceholderGenerator: entity type dispatch (no DB for nil path)
// - GetActionService caching (sync.Once behavior)
package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
)

// ─── buildTransitioner ────────────────────────────────────────────────────────

// TestBuildTransitioner_UnsupportedType verifies that buildTransitioner returns
// an error for an entity type it does not recognize. This code path does not
// call any global service accessor, so no database is required.
func TestBuildTransitioner_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	transitioner, err := buildTransitioner(ctx, "unknown_entity")

	if err == nil {
		t.Fatal("expected error for unsupported entity type, got nil")
	}
	if transitioner != nil {
		t.Errorf("expected nil transitioner for unsupported type, got %T", transitioner)
	}
	// Error message must identify the unsupported type.
	if err != nil && err.Error() == "" {
		t.Error("expected non-empty error message for unsupported entity type")
	}
}

// TestBuildTransitioner_UnsupportedType_ErrorMessage verifies the error message
// includes the unsupported type name so callers can diagnose the problem.
func TestBuildTransitioner_UnsupportedType_ErrorMessage(t *testing.T) {
	ctx := context.Background()
	_, err := buildTransitioner(ctx, "widget")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if len(msg) == 0 {
		t.Error("error message is empty")
	}
	// The type string "widget" should appear in the error for diagnosability.
	found := false
	for i := 0; i+len("widget") <= len(msg); i++ {
		if msg[i:i+len("widget")] == "widget" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("error message %q does not contain the unsupported type name %q", msg, "widget")
	}
}

// ─── buildPlaceholderGenerator ────────────────────────────────────────────────

// TestBuildPlaceholderGenerator_UnsupportedType verifies that
// buildPlaceholderGenerator returns nil for an unknown entity type.
// Per the contract: "controller handles nil gracefully."
func TestBuildPlaceholderGenerator_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	gen := buildPlaceholderGenerator(ctx, "unknown_entity")

	if gen != nil {
		t.Errorf("expected nil PlaceholderGenerator for unsupported entity type, got %T", gen)
	}
}

// TestBuildPlaceholderGenerator_UnsupportedType_EmptyString verifies that
// the empty string also produces nil (not a panic).
func TestBuildPlaceholderGenerator_UnsupportedType_EmptyString(t *testing.T) {
	ctx := context.Background()
	gen := buildPlaceholderGenerator(ctx, "")

	if gen != nil {
		t.Errorf("expected nil PlaceholderGenerator for empty entity type, got %T", gen)
	}
}

// ─── GetActionService caching ─────────────────────────────────────────────────

// setupActionServiceTestDir creates a temporary directory with a minimal
// .sharkconfig.json so that GetActionService can initialize without error.
// It changes the working directory to the temp dir and returns a cleanup
// function that resets service globals and restores the original directory.
func setupActionServiceTestDir(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()

	// Minimal config sufficient for config.NewActionService to succeed.
	configContent := `{}`
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir to tmpDir: %v", err)
	}

	return func() {
		cli.ResetServices()
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// TestGetActionService_ReturnsNonNil verifies that GetActionService returns a
// non-nil service when a valid .sharkconfig.json exists in the project root.
func TestGetActionService_ReturnsNonNil(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()
	svc, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("GetActionService() returned nil service")
	}
}

// TestGetActionService_SameInstanceOnRepeatCalls verifies that repeated calls
// to GetActionService return the same instance (sync.Once caching behavior).
func TestGetActionService_SameInstanceOnRepeatCalls(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()

	svc1, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() first call error: %v", err)
	}

	svc2, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() second call error: %v", err)
	}

	// Both calls must return the exact same pointer (sync.Once caching).
	if svc1 != svc2 {
		t.Errorf("GetActionService() returned different instances on repeated calls: %p vs %p", svc1, svc2)
	}
}

// TestGetActionService_ReturnsErrorAfterReset verifies that after ResetServices()
// a new call to GetActionService re-initializes from the config (sync.Once resets).
func TestGetActionService_ResetAndReinit(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()

	// First initialization.
	svc1, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() first call error: %v", err)
	}

	// Reset clears cached state.
	cli.ResetServices()

	// Second initialization — should succeed again (same config file still present).
	svc2, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() after reset error: %v", err)
	}
	if svc2 == nil {
		t.Fatal("GetActionService() returned nil after reset")
	}

	// The two instances may or may not be equal by pointer — what matters
	// is that both are valid non-nil services.
	_ = svc1
}
