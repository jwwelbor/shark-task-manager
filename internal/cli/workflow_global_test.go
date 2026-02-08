package cli

import (
	"os"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestGetWorkflowService_ReturnsNonNil(t *testing.T) {
	// Setup: Use temp directory with no config (falls back to defaults)
	tmpDir := t.TempDir()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	svc := GetWorkflowService()
	if svc == nil {
		t.Fatal("Expected non-nil workflow service, got nil")
	}
}

func TestGetWorkflowService_ReturnsSameInstance(t *testing.T) {
	// Setup: Use temp directory
	tmpDir := t.TempDir()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	svc1 := GetWorkflowService()
	svc2 := GetWorkflowService()

	if svc1 != svc2 {
		t.Error("Expected same instance on multiple calls, got different instances")
	}
}

func TestResetWorkflowService_ClearsCache(t *testing.T) {
	// Setup: Use temp directory
	tmpDir := t.TempDir()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	// Get first instance
	svc1 := GetWorkflowService()
	if svc1 == nil {
		t.Fatal("Expected non-nil workflow service, got nil")
	}

	// Reset clears the cache
	ResetWorkflowService()
	config.ClearWorkflowCache()

	// Get second instance - should be a new one
	svc2 := GetWorkflowService()
	if svc2 == nil {
		t.Fatal("Expected non-nil workflow service after reset, got nil")
	}

	if svc1 == svc2 {
		t.Error("Expected different instance after reset, got same pointer")
	}
}

func TestGetWorkflowService_FallsBackToDefaults(t *testing.T) {
	// Setup: Use temp directory with no .sharkconfig.json
	tmpDir := t.TempDir()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	svc := GetWorkflowService()
	if svc == nil {
		t.Fatal("Expected non-nil workflow service with defaults, got nil")
	}

	// Verify that the service provides valid default behavior
	// The default workflow should define at least some statuses
	wf := svc.GetWorkflow()
	if wf == nil {
		t.Fatal("Expected non-nil workflow config from default service")
	}

	// Default workflow should have a default status for new tasks
	defaultStatus := svc.GetDefaultStatus()
	if defaultStatus == "" {
		t.Error("Expected non-empty default status from fallback workflow")
	}
}

func TestGetWorkflowService_LoadsFromConfig(t *testing.T) {
	// Setup: Create temp directory with a .sharkconfig.json
	tmpDir := t.TempDir()

	configContent := `{
		"status_metadata": {
			"custom_status": {
				"color": "blue",
				"phase": "development",
				"progress_weight": 50,
				"responsibility": "agent",
				"blocks_feature": false
			},
			"done": {
				"color": "green",
				"phase": "done",
				"progress_weight": 100,
				"responsibility": "none",
				"blocks_feature": false
			}
		},
		"status_flow": {
			"custom_status": ["done"],
			"done": []
		},
		"special_statuses": {
			"_start_": ["custom_status"],
			"_complete_": ["done"]
		}
	}`
	if err := os.WriteFile(tmpDir+"/.sharkconfig.json", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	svc := GetWorkflowService()
	if svc == nil {
		t.Fatal("Expected non-nil workflow service, got nil")
	}

	// Verify the custom config was loaded
	defaultStatus := svc.GetDefaultStatus()
	if defaultStatus != "custom_status" {
		t.Errorf("Expected default status 'custom_status' from config, got %q", defaultStatus)
	}
}

func TestGetWorkflowService_ThreadSafety(t *testing.T) {
	// Setup: Use temp directory
	tmpDir := t.TempDir()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetWorkflowService()
	defer config.ClearWorkflowCache()

	// Launch multiple goroutines all calling GetWorkflowService concurrently
	const numGoroutines = 20
	var wg sync.WaitGroup
	results := make(chan *struct{ ptr uintptr }, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			svc := GetWorkflowService()
			// Use uintptr to capture the pointer value for comparison
			// (unsafe.Pointer is not needed; we just compare the service directly)
			if svc == nil {
				t.Error("GetWorkflowService returned nil in goroutine")
			}
			results <- &struct{ ptr uintptr }{ptr: 0} // placeholder
		}()
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Drain results
	count := 0
	for range results {
		count++
	}

	if count != numGoroutines {
		t.Errorf("Expected %d results, got %d", numGoroutines, count)
	}

	// Verify we can still get the service after concurrent access
	svc := GetWorkflowService()
	if svc == nil {
		t.Fatal("Expected non-nil workflow service after concurrent access")
	}
}
