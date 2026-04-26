package services

import (
	"context"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Compile-time check: *MockTagService must satisfy both TagAttacher and TagQuerier.
var _ TagAttacher = (*MockTagService)(nil)
var _ TagQuerier = (*MockTagService)(nil)

// MockTagService is the package-private, shared mock used by each
// entity-service test file (TaskService, FeatureService, EpicService,
// BugService, ChangeCardService, IdeaService). It implements the TagQuerier
// interface (which includes TagAttacher) consumed by entity services.
// Previously it only implemented TagAttacher (F04). In F05 it grows three
// new query methods to satisfy TagQuerier.
//
// Deprecated comment (pre-F05) retained for context:
// It implements the subset of the *TagService API that entity services consume:
//
//   - EnforceRequired(ctx, entityType, names) error
//   - AttachMany(ctx, entityType, entityID, names) error
//   - DetachOne(ctx, entityType, entityID, name) error  (reserved for
//     future non-create/update hooks; not required by T-E28-F04-005)
//
// Because the real *TagService is a concrete struct (not an interface) on
// the entity-service side, this mock does NOT purport to substitute for
// *TagService directly — instead entity services accept a narrow interface
// TagAttacher (defined in tag_service.go) that both *TagService and
// *MockTagService satisfy.
//
// Features:
//
//   - Function fields allow per-test error injection.
//   - Call log (Events) captures the call order across Create/Attach/etc.
//     for the AC-17 ordering assertion ("EnforceRequired before persist;
//     AttachMany after persist"). Callers push entity-service events like
//     "Create" into the same log via RecordEvent().
//   - Call counters give a concise single-variable assertion surface for
//     AC-15 / AC-16 / AC-18 ("called N times").
//
// Spec: E28-F04 test-plan.md §3.2 ("new helpers") and §1.2 ("ordering
// assertion strategy").
type MockTagService struct {
	// Function-field hooks. Callers leave these nil for the happy path.
	enforceRequiredFn       func(ctx context.Context, entityType models.EntityType, names []string) error
	attachManyFn            func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error
	detachOneFn             func(ctx context.Context, entityType models.EntityType, entityID int64, name string) error
	entityIDsByTagsFn       func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
	listTagsForEntityFn     func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)
	attachedTagNamesByIDsFn func(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)

	// Call counters. Read after the service call under test completes.
	EnforceRequiredCalls       int
	AttachManyCalls            int
	DetachOneCalls             int
	EntityIDsByTagsCalls       int
	ListTagsForEntityCalls     int
	AttachedTagNamesByIDsCalls int

	// Captured arguments from the MOST RECENT call to each method. These
	// are overwritten on repeat invocations within a single test.
	LastEnforceEntityType models.EntityType
	LastEnforceNames      []string
	LastAttachEntityType  models.EntityType
	LastAttachEntityID    int64
	LastAttachNames       []string
	LastDetachEntityType  models.EntityType
	LastDetachEntityID    int64
	LastDetachName        string

	// Events is an ordered log of tag-service operations plus any events
	// callers explicitly record via RecordEvent (e.g., "Create" from the
	// entity repo mock). The log is append-only and is protected by mu so
	// tests can safely run in parallel.
	mu     sync.Mutex
	Events []string
}

// NewMockTagService returns a zero-valued MockTagService ready for use.
// All function fields are nil (happy path). Tests that need error paths
// should set the relevant Fn field before calling the service under test.
func NewMockTagService() *MockTagService {
	return &MockTagService{}
}

// RecordEvent appends an arbitrary event string to the shared Events log.
// Entity-service repo mocks call this with "Create", "Update", etc. so
// tests can assert the interleaving of repo work with tag-service calls
// (AC-17: "Create before AttachMany").
func (m *MockTagService) RecordEvent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, name)
}

// WithEnforceRequiredFn overrides the default (nil-returning) behaviour
// of EnforceRequired. Returns the receiver so tests can chain.
func (m *MockTagService) WithEnforceRequiredFn(fn func(ctx context.Context, entityType models.EntityType, names []string) error) *MockTagService {
	m.enforceRequiredFn = fn
	return m
}

// WithAttachManyFn overrides the default (nil-returning) behaviour of
// AttachMany.
func (m *MockTagService) WithAttachManyFn(fn func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error) *MockTagService {
	m.attachManyFn = fn
	return m
}

// WithDetachOneFn overrides the default (nil-returning) behaviour of
// DetachOne.
func (m *MockTagService) WithDetachOneFn(fn func(ctx context.Context, entityType models.EntityType, entityID int64, name string) error) *MockTagService {
	m.detachOneFn = fn
	return m
}

// EnforceRequired records the call, appends an "EnforceRequired" event to
// the log, and delegates to the configured function (or returns nil on
// the happy path).
func (m *MockTagService) EnforceRequired(ctx context.Context, entityType models.EntityType, names []string) error {
	m.mu.Lock()
	m.EnforceRequiredCalls++
	m.LastEnforceEntityType = entityType
	m.LastEnforceNames = append([]string(nil), names...)
	m.Events = append(m.Events, "EnforceRequired")
	fn := m.enforceRequiredFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, names)
	}
	return nil
}

// AttachMany records the call, appends an "AttachMany" event, and
// delegates to the configured function (or returns nil).
func (m *MockTagService) AttachMany(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
	m.mu.Lock()
	m.AttachManyCalls++
	m.LastAttachEntityType = entityType
	m.LastAttachEntityID = entityID
	m.LastAttachNames = append([]string(nil), names...)
	m.Events = append(m.Events, "AttachMany")
	fn := m.attachManyFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, entityID, names)
	}
	return nil
}

// DetachOne records the call, appends a "DetachOne" event, and delegates
// to the configured function (or returns nil).
func (m *MockTagService) DetachOne(ctx context.Context, entityType models.EntityType, entityID int64, name string) error {
	m.mu.Lock()
	m.DetachOneCalls++
	m.LastDetachEntityType = entityType
	m.LastDetachEntityID = entityID
	m.LastDetachName = name
	m.Events = append(m.Events, "DetachOne")
	fn := m.detachOneFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, entityID, name)
	}
	return nil
}

// EventsCopy returns a snapshot of the event log. Useful for assertions
// that want a stable slice independent of subsequent mutations.
func (m *MockTagService) EventsCopy() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.Events...)
}

// WithEntityIDsByTagsFn overrides the default behaviour of EntityIDsByTags.
func (m *MockTagService) WithEntityIDsByTagsFn(fn func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)) *MockTagService {
	m.entityIDsByTagsFn = fn
	return m
}

// WithListTagsForEntityFn overrides the default behaviour of ListTagsForEntity.
func (m *MockTagService) WithListTagsForEntityFn(fn func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)) *MockTagService {
	m.listTagsForEntityFn = fn
	return m
}

// WithAttachedTagNamesByIDsFn overrides the default behaviour of AttachedTagNamesByIDs.
func (m *MockTagService) WithAttachedTagNamesByIDsFn(fn func(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)) *MockTagService {
	m.attachedTagNamesByIDsFn = fn
	return m
}

// EntityIDsByTags records the call, appends an "EntityIDsByTags" event, and
// delegates to the configured function (or returns nil, nil on the happy path).
func (m *MockTagService) EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
	m.mu.Lock()
	m.EntityIDsByTagsCalls++
	m.Events = append(m.Events, "EntityIDsByTags")
	fn := m.entityIDsByTagsFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, names, op)
	}
	return nil, nil
}

// ListTagsForEntity records the call, appends a "ListTagsForEntity" event,
// and delegates to the configured function (or returns an empty slice).
func (m *MockTagService) ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
	m.mu.Lock()
	m.ListTagsForEntityCalls++
	m.Events = append(m.Events, "ListTagsForEntity")
	fn := m.listTagsForEntityFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, entityID)
	}
	return []string{}, nil
}

// AttachedTagNamesByIDs records the call, appends an "AttachedTagNamesByIDs"
// event, and delegates to the configured function (or returns an empty map).
func (m *MockTagService) AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error) {
	m.mu.Lock()
	m.AttachedTagNamesByIDsCalls++
	m.Events = append(m.Events, "AttachedTagNamesByIDs")
	fn := m.attachedTagNamesByIDsFn
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, entityType, entityIDs)
	}
	return map[int64][]string{}, nil
}
