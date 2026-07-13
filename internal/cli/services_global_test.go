package cli

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/team"
	"github.com/stretchr/testify/require"
)

// TestGetTeamServices_Wiring_TC010 verifies that the established global
// accessor seam exposes the already-implemented planner and ledger services.
// The accessors must construct the real injected services without adding a
// team-specific command or bypassing the service layer.
func TestGetTeamServices_Wiring_TC010(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".sharkconfig.json"), []byte(`{}`), 0o644))
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { require.NoError(t, os.Chdir(originalWD)) }()
	defer ResetDB()
	ResetServices()
	defer ResetServices()

	planner := GetTeamPlanner()
	require.NotNil(t, planner)
	var _ team.Planner = planner

	ledger := GetTeamLedger()
	require.NotNil(t, ledger)
	var _ team.Ledger = ledger
}

// TestResetServices_ReplacesContainer verifies that ResetServices() replaces
// the container with a fresh one, so subsequent Get* calls re-initialize.
func TestResetServices_ReplacesContainer(t *testing.T) {
	// Capture the original container pointer.
	before := loadContainer()
	if before == nil {
		t.Fatal("expected non-nil container before reset")
	}

	ResetServices()
	defer ResetServices() // ensure cleanup

	after := loadContainer()
	if after == nil {
		t.Fatal("expected non-nil container after reset")
	}

	if before == after {
		t.Error("ResetServices() should have replaced the container with a new one, but pointer is unchanged")
	}
}

// TestResetServices_FreshContainerIsEmpty verifies that after ResetServices()
// the container holds no initialized services (all fields are zero/nil).
func TestResetServices_FreshContainerIsEmpty(t *testing.T) {
	ResetServices()
	defer ResetServices()

	c := loadContainer()

	if c.registry != nil {
		t.Error("expected registry to be nil in fresh container")
	}
	if c.entityService != nil {
		t.Error("expected entityService to be nil in fresh container")
	}
	if c.actionService != nil {
		t.Error("expected actionService to be nil in fresh container")
	}
	if c.actionServiceErr != nil {
		t.Error("expected actionServiceErr to be nil in fresh container")
	}
	if c.noteService != nil {
		t.Error("expected noteService to be nil in fresh container")
	}
	if c.noteServiceErr != nil {
		t.Error("expected noteServiceErr to be nil in fresh container")
	}
	if c.contextService != nil {
		t.Error("expected contextService to be nil in fresh container")
	}
	if c.resumeService != nil {
		t.Error("expected resumeService to be nil in fresh container")
	}
	if c.resumeServiceErr != nil {
		t.Error("expected resumeServiceErr to be nil in fresh container")
	}
}

// TestServiceContainer_ConcurrentSwap verifies that concurrent swaps of the
// service container via storeContainer / loadContainer do not cause data races.
// This is the core safety property of the ServiceContainer pattern: resets are
// atomic pointer swaps rather than non-atomic reassignment of individual
// sync.Once values.
//
// Run with -race to detect any remaining issues: go test -race ./internal/cli/
func TestServiceContainer_ConcurrentSwap(t *testing.T) {
	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Directly exercise the atomic swap helpers, not ResetServices()
			// (which calls ResetObservability that has its own pre-existing state).
			storeContainer(new(serviceContainer))
			_ = loadContainer()
		}()
	}

	wg.Wait()

	// After concurrent swaps, container must still be non-nil and valid.
	c := loadContainer()
	if c == nil {
		t.Fatal("container must not be nil after concurrent swaps")
	}
}

// TestLoadStoreContainer_RoundTrip verifies the atomic load/store helpers
// maintain pointer identity correctly.
func TestLoadStoreContainer_RoundTrip(t *testing.T) {
	original := loadContainer()

	fresh := new(serviceContainer)
	storeContainer(fresh)
	defer storeContainer(original) // restore

	got := loadContainer()
	if got != fresh {
		t.Errorf("loadContainer() = %p, want %p", got, fresh)
	}
}

// TestServiceContainer_SyncOnceIsolation verifies that each serviceContainer
// has independent sync.Once instances, so resetting the container truly
// allows re-initialization rather than being a no-op.
func TestServiceContainer_SyncOnceIsolation(t *testing.T) {
	c1 := new(serviceContainer)
	c2 := new(serviceContainer)

	// Track how many times the Do function runs in each container.
	count1 := 0
	count2 := 0

	c1.registryOnce.Do(func() { count1++ })
	c1.registryOnce.Do(func() { count1++ }) // should not run again

	c2.registryOnce.Do(func() { count2++ }) // should run independently
	c2.registryOnce.Do(func() { count2++ }) // should not run again

	if count1 != 1 {
		t.Errorf("c1 sync.Once ran %d times, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("c2 sync.Once ran %d times, want 1", count2)
	}
}
