package cli

import (
	"context"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/team"
	"github.com/stretchr/testify/require"
)

// TestGetTeamServices_Wiring_TC010 verifies the construction helpers used by
// the established global accessors. It uses injected seams so CLI wiring tests
// do not open a real database while preserving production-shaped construction.
func TestGetTeamServices_Wiring_TC010(t *testing.T) {
	planner := newTeamPlanner(team.PlannerDeps{
		Children: testTeamPlannerDeps{},
		Dispatch: testTeamPlannerDeps{},
	})
	require.NotNil(t, planner)
	var _ team.Planner = planner

	ledger := newTeamLedger(&testTeamLedgerRepository{}, t.TempDir())
	require.NotNil(t, ledger)
	var _ team.Ledger = ledger
}

type testTeamPlannerDeps struct{}

func (testTeamPlannerDeps) ListChildren(context.Context, models.EntityType, string) ([]team.ChildSnapshot, error) {
	return nil, nil
}

func (testTeamPlannerDeps) Resolve(context.Context, models.EntityType, string) (dispatch.DispatchStep, error) {
	return dispatch.DispatchStep{}, nil
}

type testTeamLedgerRepository struct{}

func (*testTeamLedgerRepository) FindRunByRoot(context.Context, string, string) (*teamrunrepo.TeamRun, error) {
	return nil, team.ErrRepositoryNotFound
}
func (*testTeamLedgerRepository) CreateRunWithItems(context.Context, *teamrunrepo.TeamRun, []*teamrunrepo.TeamRunItem) error {
	return nil
}
func (*testTeamLedgerRepository) CreateRunWithItemsIfAbsent(context.Context, *teamrunrepo.TeamRun, []*teamrunrepo.TeamRunItem) (*teamrunrepo.TeamRun, bool, error) {
	return nil, false, nil
}
func (*testTeamLedgerRepository) GetRun(context.Context, int64) (*teamrunrepo.TeamRun, error) {
	return nil, team.ErrRepositoryNotFound
}
func (*testTeamLedgerRepository) ListItems(context.Context, int64) ([]*teamrunrepo.TeamRunItem, error) {
	return nil, nil
}
func (*testTeamLedgerRepository) UpdateRun(context.Context, *teamrunrepo.TeamRun) error { return nil }
func (*testTeamLedgerRepository) CompareAndSetItem(context.Context, *teamrunrepo.TeamRunItem, string, int) (bool, error) {
	return false, nil
}

type testRelationshipReader struct{}

func (testRelationshipReader) GetOutgoing(context.Context, models.EntityType, int64, []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	return []*models.EntityRelationship{{
		FromEntityType:   models.EntityTypeTask,
		FromEntityID:     1,
		ToEntityType:     models.EntityTypeTask,
		ToEntityID:       2,
		RelationshipType: models.EntityRelDependsOn,
	}}, nil
}

type testEntityRegistry struct{}

func (testEntityRegistry) GetRepository(models.EntityType) (services.EntityRepository, error) {
	return testEntityRepository{}, nil
}

type testEntityRepository struct{}

func (testEntityRepository) GetByKey(context.Context, string) (models.Entity, error) {
	return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "T-E38-F01-001"}, Status: "todo"}, nil
}
func (testEntityRepository) GetByID(context.Context, int64) (models.Entity, error) {
	return &models.Task{BaseEntity: models.BaseEntity{ID: 2, Key: "T-E37-F01-001"}, Status: "completed"}, nil
}
func (testEntityRepository) UpdateStatus(context.Context, int64, string) error       { return nil }
func (testEntityRepository) Update(context.Context, models.Entity) error             { return nil }
func (testEntityRepository) GetContextData(context.Context, int64) (*string, error)  { return nil, nil }
func (testEntityRepository) UpdateContextData(context.Context, int64, *string) error { return nil }

func TestTeamRelationshipDependencySource_ReportsSatisfiedExternalMetadata(t *testing.T) {
	source := teamRelationshipDependencySource{repo: testRelationshipReader{}, registry: testEntityRegistry{}}
	edges, err := source.ListRelationshipDependencies(context.Background(), team.ChildIdentity{Key: "T-E38-F01-001", EntityType: models.EntityTypeTask})
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.True(t, edges[0].Resolved)
	require.True(t, edges[0].Satisfied)
	require.Equal(t, "completed", edges[0].DependencyStatus)
	require.Equal(t, "T-E37-F01-001", edges[0].DependencyKey)
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
