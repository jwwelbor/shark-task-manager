package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupAccessorTestDB sets up a temporary database and returns a cleanup function.
// This pattern mirrors db_global_test.go for CLI-layer tests that need a real DB.
func setupAccessorTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test-accessor.db")

	configContent := `{
		"database": {
			"backend": "local",
			"url": "` + testDB + `"
		}
	}`
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
		ResetDB()
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// GetEpicService() integration tests
// ---------------------------------------------------------------------------

// TestGetEpicService_ReturnsNonNil verifies that GetEpicService returns a
// non-nil *services.EpicService and that it can be called without panicking.
func TestGetEpicService_ReturnsNonNil(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	svc := GetEpicService()
	if svc == nil {
		t.Fatal("Expected non-nil *services.EpicService, got nil")
	}

	// Call GetEpic — may return NotFoundError for a nonexistent key,
	// but must not nil-pointer panic.
	ctx := context.Background()
	_, err := svc.GetEpic(ctx, "E99-nonexistent")
	// Error is acceptable (entity not found); nil pointer panic is not.
	_ = err
}

// TestGetEpicService_WiresTaskRepo verifies that GetEpicService wires the
// task repository so that GetImpediments does not nil-pointer panic.
func TestGetEpicService_WiresTaskRepo(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	svc := GetEpicService()
	if svc == nil {
		t.Fatal("GetEpicService() returned nil")
	}

	// GetImpediments relies on taskRepo being wired; if not wired it panics.
	ctx := context.Background()
	_, err := svc.GetImpediments(ctx, "E99-nonexistent")
	// Error is expected (no such epic), but no nil-pointer panic.
	_ = err
}

// TestGetEpicService_WiresDocRepo verifies that SetDocRepo was called during
// GetEpicService so that GetRelatedDocuments does not nil-pointer panic.
func TestGetEpicService_WiresDocRepo(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	svc := GetEpicService()
	if svc == nil {
		t.Fatal("GetEpicService() returned nil")
	}

	// GetRelatedDocuments relies on docRepo being set via SetDocRepo.
	// If not wired it will panic.
	ctx := context.Background()
	_, err := svc.GetRelatedDocuments(ctx, 0)
	// Error acceptable; nil pointer panic is not.
	_ = err
}

// ---------------------------------------------------------------------------
// GetFeatureService() integration tests
// ---------------------------------------------------------------------------

// TestGetFeatureService_ReturnsNonNil verifies that GetFeatureService returns
// a non-nil *services.FeatureService and that it is callable without panicking.
func TestGetFeatureService_ReturnsNonNil(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	svc := GetFeatureService()
	if svc == nil {
		t.Fatal("Expected non-nil *services.FeatureService, got nil")
	}

	// Call GetFeature — may return NotFoundError, but must not panic.
	ctx := context.Background()
	_, err := svc.GetFeature(ctx, "E99-F99-nonexistent")
	_ = err
}

// TestGetFeatureService_UsesConstructorWithSetters verifies that
// GetFeatureService produces a service wired with all relationship repos
// (taskRepo, epicRepo) by exercising a method that requires them.
// If the constructor + setters are not used, the service would be
// missing taskRepo/epicRepo and would panic or return an error indicating nil.
func TestGetFeatureService_UsesConstructorWithSetters(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	svc := GetFeatureService()
	if svc == nil {
		t.Fatal("GetFeatureService() returned nil")
	}

	// GetProgress calls the task counting interface which is only
	// available when the constructor wires taskRepo via the constructor parameter.
	// If it were not wired, this would panic.
	ctx := context.Background()
	_, err := svc.GetProgress(ctx, "E99-F99-nonexistent")
	// Error is expected (not found), panic is not.
	_ = err
}

// ---------------------------------------------------------------------------
// Workflow wiring regression tests
//
// These tests verify that buildTaskServiceDeps() correctly initializes the
// workflow service from .sharkconfig.json configuration.
//
// Historical context: Originally these tests verified workflow propagation to
// the TaskRepository (which stored workflow internally for validation). Since
// E15-F13, workflow validation moved entirely to the service layer, so the repo
// no longer stores workflow config. The workflow service is now the sole source
// of truth for status validation.
// ---------------------------------------------------------------------------

// setupAdvancedWorkflowDB sets up a temporary database with an advanced workflow
// config containing statuses like in_approval that only exist in non-basic profiles.
// Returns a cleanup function that resets both DB and workflow service globals.
func setupAdvancedWorkflowDB(t *testing.T) func() {
	t.Helper()

	// Reset globals BEFORE setup to clear any cached state from prior tests.
	ResetDB()
	ResetWorkflowService()

	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test-wiring.db")

	// Config with advanced workflow: includes in_approval and other non-basic statuses.
	// The key invariant is that status_flow must contain statuses beyond the basic 5.
	configContent := `{
		"database": {
			"backend": "local",
			"url": "` + testDB + `"
		},
		"status_flow": {
			"todo": ["in_progress", "blocked"],
			"in_progress": ["ready_for_review", "blocked"],
			"ready_for_review": ["completed", "in_progress"],
			"completed": [],
			"blocked": ["todo", "in_progress"],
			"ready_for_approval": ["in_approval", "blocked"],
			"in_approval": ["completed", "ready_for_approval"]
		}
	}`
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
		ResetDB()
		ResetWorkflowService()
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// TestBuildTaskServiceDeps_WorkflowServiceAvailable verifies that buildTaskServiceDeps
// correctly initializes the workflow service from .sharkconfig.json.
//
// Historical note: These tests originally verified that workflow config was propagated
// to the TaskRepository (which stored it internally). Since E15-F13, workflow validation
// has moved entirely to the service layer, so the repo no longer stores workflow config.
// The workflow service is now the sole source of truth for status validation.
func TestBuildTaskServiceDeps_WorkflowServiceAvailable(t *testing.T) {
	cleanup := setupAdvancedWorkflowDB(t)
	defer cleanup()

	deps := buildTaskServiceDeps()

	// Verify entity service is initialized and has the advanced config
	svcWorkflow := deps.entitySvc.GetWorkflowService().GetWorkflow()
	if svcWorkflow == nil {
		t.Fatal("WorkflowService workflow is nil; expected non-nil workflow config")
	}
	if svcWorkflow.StatusFlow == nil {
		t.Fatal("WorkflowService StatusFlow is nil")
	}

	// These statuses exist in the advanced config but NOT in the basic 5-status default.
	advancedStatuses := []string{"in_approval", "ready_for_approval"}
	for _, status := range advancedStatuses {
		if _, exists := svcWorkflow.StatusFlow[status]; !exists {
			t.Errorf("WorkflowService missing advanced status %q from config", status)
		}
	}

	// Also verify basic statuses are present (sanity check).
	basicStatuses := []string{"todo", "in_progress", "completed", "blocked"}
	for _, status := range basicStatuses {
		if _, exists := svcWorkflow.StatusFlow[status]; !exists {
			t.Errorf("WorkflowService missing basic status %q", status)
		}
	}
}

// TestBuildTaskServiceDeps_RepoAndServiceInitialized verifies that buildTaskServiceDeps
// correctly initializes both the task repo and workflow service.
func TestBuildTaskServiceDeps_RepoAndServiceInitialized(t *testing.T) {
	cleanup := setupAdvancedWorkflowDB(t)
	defer cleanup()

	deps := buildTaskServiceDeps()

	if deps.taskRepo == nil {
		t.Fatal("TaskRepository is nil")
	}
	if deps.entitySvc == nil {
		t.Fatal("EntityService is nil")
	}
	if deps.creatorSvc == nil {
		t.Fatal("CreatorSvc is nil")
	}
}

// ---------------------------------------------------------------------------
// Architecture compliance test
// ---------------------------------------------------------------------------

// TestNoTODOsInServiceAccessors verifies that the service_accessors.go file
// contains no TODO comments, confirming the implementation is complete.
func TestNoTODOsInServiceAccessors(t *testing.T) {
	content, err := os.ReadFile("service_accessors.go")
	if err != nil {
		t.Fatalf("Failed to read service_accessors.go: %v", err)
	}

	// Count occurrences of "TODO" in the file.
	count := 0
	for i := 0; i+4 <= len(content); i++ {
		if string(content[i:i+4]) == "TODO" {
			count++
		}
	}

	if count > 0 {
		t.Errorf("service_accessors.go contains %d TODO comment(s); expected 0", count)
	}
}

func TestGetClaimService_UsesConfigClaimTTLSeconds(t *testing.T) {
	t.Run("zero disables expiry", func(t *testing.T) {
		ResetDB()
		tmpDir := t.TempDir()
		testDB := filepath.Join(tmpDir, "claim-zero.db")
		configContent := `{
			"database": {
				"backend": "local",
				"url": "` + testDB + `"
			},
			"claim_ttl_seconds": 0
		}`
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
		defer func() {
			ResetDB()
			if err := os.Chdir(origWd); err != nil {
				t.Errorf("Failed to restore working directory: %v", err)
			}
		}()

		svc := GetClaimService()
		if got := svc.TTL(); got != 0 {
			t.Fatalf("TTL() = %v, want 0", got)
		}
	})

	t.Run("config overrides env", func(t *testing.T) {
		ResetDB()
		t.Setenv("SHARK_CLAIM_TTL_SECONDS", "7")
		tmpDir := t.TempDir()
		testDB := filepath.Join(tmpDir, "claim-config.db")
		configContent := `{
			"database": {
				"backend": "local",
				"url": "` + testDB + `"
			},
			"claim_ttl_seconds": 120
		}`
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
		defer func() {
			ResetDB()
			if err := os.Chdir(origWd); err != nil {
				t.Errorf("Failed to restore working directory: %v", err)
			}
		}()

		svc := GetClaimService()
		if got := svc.TTL(); got != 120*time.Second {
			t.Fatalf("TTL() = %v, want 120s", got)
		}
	})
}
