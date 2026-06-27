package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ---------------------------------------------------------------------------
// AC-12 / test-plan.md §12.1 — GetTagService smoke test against a real TempDir DB
// ---------------------------------------------------------------------------

// TestGetTagService_Smoke verifies that GetTagService() returns a working
// *TagService whose ListTags method operates against a real initialized DB.
//
// Spec reference: spec.md REQ-F-012, AC-12, test-plan.md 12.1.
func TestGetTagService_Smoke(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal .sharkconfig.json (no maintainer block required for ListTags).
	writeTagGlobalSharkConfig(t, tmpDir, map[string]interface{}{
		"workflow_config": "shark-data/workflow/",
	})

	// Change to tmpDir so FindProjectRoot() locates the config.
	origWd := chdirForTagTest(t, tmpDir)
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
		cli.ResetDB()
	}()

	// GetTagService() should not panic and should return a non-nil *TagService.
	var svc *services.TagService
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("GetTagService() panicked: %v", r)
			}
		}()
		svc = cli.GetTagService()
	}()

	if svc == nil {
		t.Fatal("GetTagService() returned nil")
	}

	// ListTags should work against the (empty) initialized DB.
	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags(ctx) returned unexpected error: %v", err)
	}

	// Empty vocabulary is valid — nil or an empty slice.
	if len(tags) != 0 {
		t.Errorf("ListTags(ctx) returned %d tags, expected 0 in empty project", len(tags))
	}
}

// ---------------------------------------------------------------------------
// AC-12 / test-plan.md §12.2 — Two successive calls return distinct pointers
// ---------------------------------------------------------------------------

// TestGetTagService_ReturnsNewInstanceEachCall verifies AC-T4: successive calls
// to GetTagService() return distinct *TagService pointers (not a shared singleton).
//
// Spec reference: spec.md AC-T4, test-plan.md 12.2.
func TestGetTagService_ReturnsNewInstanceEachCall(t *testing.T) {
	tmpDir := t.TempDir()

	writeTagGlobalSharkConfig(t, tmpDir, map[string]interface{}{
		"workflow_config": "shark-data/workflow/",
	})

	origWd := chdirForTagTest(t, tmpDir)
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
		cli.ResetDB()
	}()

	svc1 := cli.GetTagService()
	svc2 := cli.GetTagService()

	if svc1 == svc2 {
		t.Error("GetTagService() returned the same pointer on two calls; expected new instances (AC-T4)")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// writeTagGlobalSharkConfig writes a minimal .sharkconfig.json to projectRoot.
// Uses a distinct name to avoid collisions with writeSharkConfig in
// maintainer_global_test.go (both are in package cli_test).
func writeTagGlobalSharkConfig(t *testing.T, projectRoot string, cfg map[string]interface{}) {
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

// chdirForTagTest changes the working directory to dir and returns the original
// working directory so the caller can restore it via defer.
func chdirForTagTest(t *testing.T, dir string) string {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	return origWd
}
