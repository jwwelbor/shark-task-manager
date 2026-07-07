package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ============================================================================
// TD-020 — Per-invocation adapter cache for resolveNext
// ============================================================================
//
// These tests verify the nextAdapterCache eliminates redundant adapter
// construction during cascade recursion. The cache is the resolution for
// TD-020: previously resolveNext rebuilt transitioner / placeholder
// generator / narrowed action service on every recursion hop (3 redundant
// constructions per hop in an epic → feature → task chain).
//
// The cache is exercised through the package-level indirection hooks
// (nextBuildTransitioner / nextBuildPlaceholderGenerator) so tests can swap
// in counting wrappers without needing a real database.

// stubTransitioner is a no-op runner.EntityTransitioner used as the cached
// adapter's payload — the test asserts on construction count, not behavior.
type stubTransitioner struct{}

func (stubTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, nil
}

func (stubTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return nil, nil
}

// stubPlaceholderGenerator is a no-op runner.PlaceholderGenerator used as the
// cached adapter's payload.
type stubPlaceholderGenerator struct{}

func (stubPlaceholderGenerator) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	return map[string]string{}, nil
}

// installCountingBuilders swaps the package-level builder hooks for counting
// wrappers and returns a cleanup function plus pointers to the counters.
// The action-service-root construction is also tracked indirectly: the cache
// receives a single action.ActionService at construction time, and every
// ForEntity call on that root increments the per-entity narrowing count.
func installCountingBuilders(t *testing.T) (transCount, genCount *int, restore func()) {
	t.Helper()

	tc := 0
	gc := 0

	origT := nextBuildTransitioner
	origG := nextBuildPlaceholderGenerator

	nextBuildTransitioner = func(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
		tc++
		return stubTransitioner{}, nil
	}
	nextBuildPlaceholderGenerator = func(_ context.Context, entityType string) runner.PlaceholderGenerator {
		gc++
		return stubPlaceholderGenerator{}
	}

	return &tc, &gc, func() {
		nextBuildTransitioner = origT
		nextBuildPlaceholderGenerator = origG
	}
}

// countingActionService wraps a stub action service and counts ForEntity
// calls per entity type. The cache should narrow at most once per type.
type countingActionService struct {
	*action.MockActionService
	forEntityCounts map[string]int
}

func newCountingActionService() *countingActionService {
	c := &countingActionService{
		MockActionService: &action.MockActionService{},
		forEntityCounts:   map[string]int{},
	}
	c.MockActionService.ForEntityFunc = func(entityType string) action.ActionService {
		c.forEntityCounts[entityType]++
		// Return the mock itself; the cache only stores the reference.
		return c.MockActionService
	}
	return c
}

// TestNextAdapterCache_BuildsEachAdapterOncePerEntityType is the headline
// TD-020 test: a single cache instance should call each builder exactly once
// per entity type, regardless of how many times get() is called for that
// type. This is what makes cascade recursion O(1) in adapter construction
// instead of O(depth).
func TestNextAdapterCache_BuildsEachAdapterOncePerEntityType(t *testing.T) {
	tc, gc, restore := installCountingBuilders(t)
	defer restore()

	actionSvc := newCountingActionService()
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc.MockActionService,
	}

	ctx := context.Background()

	// Simulate an epic → feature → task cascade with one auto-advance
	// recursion on the task. That's 4 logical resolveNext invocations across
	// 3 distinct entity types. Without the cache this would be 4 builds of
	// each adapter (12 total). With the cache it must be exactly 3 (one per
	// entity type).
	for _, entityType := range []string{"epic", "feature", "task", "task"} {
		_, err := cache.get(ctx, entityType)
		require.NoError(t, err, "cache.get(%q) failed", entityType)
	}

	assert.Equal(t, 3, *tc, "transitioner should be built once per entity type, not per recursion hop")
	assert.Equal(t, 3, *gc, "placeholder generator should be built once per entity type, not per recursion hop")
	assert.Equal(t, 1, actionSvc.forEntityCounts["epic"], "action service narrowed once for epic")
	assert.Equal(t, 1, actionSvc.forEntityCounts["feature"], "action service narrowed once for feature")
	assert.Equal(t, 1, actionSvc.forEntityCounts["task"], "action service narrowed once for task")
}

// TestNextAdapterCache_ReturnsSameInstanceOnRepeatedGet verifies the cache
// returns the *same* adapter struct pointer on repeated lookups. This is
// the contract that makes downstream identity-based assumptions safe and
// is what allows resolveNext to treat the adapter triple as effectively
// const for the duration of a `shark next` call.
func TestNextAdapterCache_ReturnsSameInstanceOnRepeatedGet(t *testing.T) {
	_, _, restore := installCountingBuilders(t)
	defer restore()

	actionSvc := newCountingActionService()
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc.MockActionService,
	}

	ctx := context.Background()

	first, err := cache.get(ctx, "task")
	require.NoError(t, err)
	second, err := cache.get(ctx, "task")
	require.NoError(t, err)

	assert.Same(t, first, second, "repeated get() for the same entity type must return the same *nextAdapters")
}

// TestNextAdapterCache_NormalizesChangeCardEntityType is the B034 regression
// test: DetectEntityType/ParseScope hand `resolveNext` the raw entity-type
// string "change_card" for CC-### keys, but the workflow-loading subsystem
// only ever registers a "change" slot in the action service's per-entity
// map. Before the fix, cache.get() called actionSvcRoot.ForEntity("change_card")
// directly, so every change-card status lookup missed the map and degraded
// to the B022 pause path regardless of status. The cache must narrow against
// "change", never the unregistered "change_card" key.
func TestNextAdapterCache_NormalizesChangeCardEntityType(t *testing.T) {
	_, _, restore := installCountingBuilders(t)
	defer restore()

	actionSvc := newCountingActionService()
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc.MockActionService,
	}

	ctx := context.Background()

	_, err := cache.get(ctx, "change_card")
	require.NoError(t, err)

	assert.Equal(t, 1, actionSvc.forEntityCounts["change"],
		"change_card must narrow the action service against the \"change\" workflow (B034)")
	assert.Equal(t, 0, actionSvc.forEntityCounts["change_card"],
		"action service must never be narrowed against the unregistered \"change_card\" key")
}

// TestNextAdapterCache_PropagatesBuilderError verifies error handling: when
// the transitioner builder fails (e.g. unknown entity type), get() must
// surface a wrapped error and not cache a partial entry. The next call
// with a valid entity type must still succeed and the failed type must
// stay rebuildable if the caller retries.
func TestNextAdapterCache_PropagatesBuilderError(t *testing.T) {
	origT := nextBuildTransitioner
	defer func() { nextBuildTransitioner = origT }()

	attempts := 0
	nextBuildTransitioner = func(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
		attempts++
		if entityType == "unknown_entity" {
			return nil, assert.AnError
		}
		return stubTransitioner{}, nil
	}

	origG := nextBuildPlaceholderGenerator
	defer func() { nextBuildPlaceholderGenerator = origG }()
	nextBuildPlaceholderGenerator = func(_ context.Context, entityType string) runner.PlaceholderGenerator {
		return stubPlaceholderGenerator{}
	}

	actionSvc := newCountingActionService()
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc.MockActionService,
	}

	ctx := context.Background()

	_, err := cache.get(ctx, "unknown_entity")
	require.Error(t, err, "unknown entity type must propagate builder error")

	// Failed entry must NOT be cached — a retry should re-invoke the builder.
	_, _ = cache.get(ctx, "unknown_entity")
	assert.Equal(t, 2, attempts, "failed builds must not be cached; retry must hit the builder again")

	// Other entity types must still resolve normally despite the prior failure.
	_, err = cache.get(ctx, "task")
	require.NoError(t, err, "valid entity type must succeed after a prior failure")
}
