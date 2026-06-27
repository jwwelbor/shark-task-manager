package cli_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// sha256hexMG computes the SHA-256 hex digest of a string.
// Isolated helper to avoid name collisions with other test files.
func sha256hexMG(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// writeSharkConfig writes a minimal .sharkconfig.json to projectRoot.
func writeSharkConfig(t *testing.T, projectRoot string, cfg map[string]interface{}) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	path := filepath.Join(projectRoot, ".sharkconfig.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-13 / INT-2: GetMaintainerGate returns a functional Gate for a valid project
// ---------------------------------------------------------------------------

func TestGetMaintainerGate_ReturnsWorkingGate(t *testing.T) {
	// Spec reference: test-plan.md AC-13, INT-2.
	// This test calls the real GetMaintainerGate() by pointing the project root
	// at a temp directory that contains a well-formed .sharkconfig.json.

	projectRoot := t.TempDir()
	password := "hunter2"
	hash := sha256hexMG(password)

	// Write config with maintainer section
	writeSharkConfig(t, projectRoot, map[string]interface{}{
		"maintainer": map[string]interface{}{
			"password_hash":        hash,
			"cache_window_seconds": 60,
		},
	})

	// Override project root discovery so the accessor finds our temp dir.
	// This is done by setting the --config flag equivalent via the global config override.
	origConfig := cli.GlobalConfig.ConfigFile
	cli.GlobalConfig.ConfigFile = filepath.Join(projectRoot, ".sharkconfig.json")
	defer func() { cli.GlobalConfig.ConfigFile = origConfig }()

	gate := cli.GetMaintainerGate()

	// Assert the gate is non-nil
	if gate == nil {
		t.Fatal("GetMaintainerGate() returned nil")
	}

	// Assert it is a *maintainer.FileGate (verifies AC-13 "is a *maintainer.FileGate")
	if _, ok := gate.(*maintainer.FileGate); !ok {
		t.Errorf("GetMaintainerGate() returned %T, want *maintainer.FileGate", gate)
	}

	ctx := context.Background()

	// Correct password returns nil
	if err := gate.Authorize(ctx, password); err != nil {
		t.Errorf("Authorize(correct password) = %v, want nil", err)
	}

	// Wrong password returns *UnauthorizedError
	err := gate.Authorize(ctx, "wrong")
	if err == nil {
		t.Fatal("Authorize(wrong password) = nil, want *UnauthorizedError")
	}
	var uErr *maintainer.UnauthorizedError
	if !errors.As(err, &uErr) {
		t.Errorf("Authorize(wrong password) error type = %T, want *maintainer.UnauthorizedError", err)
	}
}

// ---------------------------------------------------------------------------
// INT-2 edge case: Config missing maintainer field — gate always fails, no panic
// ---------------------------------------------------------------------------

func TestGetMaintainerGate_NilMaintainerConfig_NoFail(t *testing.T) {
	// Spec reference: test-plan.md AC-13 edge case, INT-2.
	// If config has no "maintainer" key, GetMaintainerGate should NOT panic
	// and should return a gate that always returns *UnauthorizedError.

	projectRoot := t.TempDir()

	// Write config WITHOUT maintainer section
	writeSharkConfig(t, projectRoot, map[string]interface{}{
		"workflow_config": "shark-data/workflow/",
	})

	origConfig := cli.GlobalConfig.ConfigFile
	cli.GlobalConfig.ConfigFile = filepath.Join(projectRoot, ".sharkconfig.json")
	defer func() { cli.GlobalConfig.ConfigFile = origConfig }()

	// Must not panic
	gate := cli.GetMaintainerGate()
	if gate == nil {
		t.Fatal("GetMaintainerGate() returned nil, expected a gate that always fails")
	}

	// Authorize should return *UnauthorizedError (missing_config), not panic
	ctx := context.Background()
	err := gate.Authorize(ctx, "anything")
	if err == nil {
		t.Fatal("Authorize(anything) = nil, want *UnauthorizedError for nil config")
	}
	var uErr *maintainer.UnauthorizedError
	if !errors.As(err, &uErr) {
		t.Errorf("Authorize error type = %T, want *maintainer.UnauthorizedError", err)
	}
}

// ---------------------------------------------------------------------------
// INT-2: GetMaintainerGate creates a new instance each call (not shared state)
// ---------------------------------------------------------------------------

func TestGetMaintainerGate_ReturnsNewInstanceEachCall(t *testing.T) {
	// Spec reference: test-plan.md INT-2 "GetMaintainerGate() creates a new instance each call".

	projectRoot := t.TempDir()
	writeSharkConfig(t, projectRoot, map[string]interface{}{
		"maintainer": map[string]interface{}{
			"password_hash": sha256hexMG("somepass"),
		},
	})

	origConfig := cli.GlobalConfig.ConfigFile
	cli.GlobalConfig.ConfigFile = filepath.Join(projectRoot, ".sharkconfig.json")
	defer func() { cli.GlobalConfig.ConfigFile = origConfig }()

	gate1 := cli.GetMaintainerGate()
	gate2 := cli.GetMaintainerGate()

	// Should be different pointers (new instance per call)
	if gate1 == gate2 {
		t.Error("GetMaintainerGate() returned the same pointer on two calls; expected new instances")
	}
}
