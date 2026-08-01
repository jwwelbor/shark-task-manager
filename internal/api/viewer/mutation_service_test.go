package viewer

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type mockEpicMutationServicer struct {
	updateEpicFn     func(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	transitionEpicFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	updateCalls      int
	transitionCalls  int
}

func (m *mockEpicMutationServicer) UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
	m.updateCalls++
	if m.updateEpicFn != nil {
		return m.updateEpicFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockEpicMutationServicer) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionCalls++
	if m.transitionEpicFn != nil {
		return m.transitionEpicFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

type mockFeatureMutationServicer struct {
	updateFeatureFn     func(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	transitionFeatureFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	updateCalls         int
	transitionCalls     int
}

func (m *mockFeatureMutationServicer) UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
	m.updateCalls++
	if m.updateFeatureFn != nil {
		return m.updateFeatureFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockFeatureMutationServicer) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionCalls++
	if m.transitionFeatureFn != nil {
		return m.transitionFeatureFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

type mockTaskMutationServicer struct {
	updateTaskFn     func(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	transitionTaskFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	updateCalls      int
	transitionCalls  int
}

func (m *mockTaskMutationServicer) UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
	m.updateCalls++
	if m.updateTaskFn != nil {
		return m.updateTaskFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockTaskMutationServicer) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionCalls++
	if m.transitionTaskFn != nil {
		return m.transitionTaskFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

type mockNoteMutationServicer struct {
	addNoteFn     func(ctx context.Context, entityType models.EntityType, entityKey, noteType, content, createdBy string) (*models.EntityNote, error)
	addCalls      int
	lastType      models.EntityType
	lastKey       string
	lastNoteType  string
	lastContent   string
	lastCreatedBy string
}

func (m *mockNoteMutationServicer) AddNote(ctx context.Context, entityType models.EntityType, entityKey, noteType, content, createdBy string) (*models.EntityNote, error) {
	m.addCalls++
	m.lastType = entityType
	m.lastKey = entityKey
	m.lastNoteType = noteType
	m.lastContent = content
	m.lastCreatedBy = createdBy
	if m.addNoteFn != nil {
		return m.addNoteFn(ctx, entityType, entityKey, noteType, content, createdBy)
	}
	return nil, nil
}

type mockRelationshipMutationServicer struct {
	createFn     func(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) (*models.EntityRelationship, error)
	unlinkFn     func(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error
	createCalls  int
	unlinkCalls  int
	lastFromType models.EntityType
	lastFromID   int64
	lastToType   models.EntityType
	lastToID     int64
	lastRelType  models.EntityRelationshipType
}

func (m *mockRelationshipMutationServicer) CreateRelationship(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) (*models.EntityRelationship, error) {
	m.createCalls++
	m.lastFromType = fromType
	m.lastFromID = fromID
	m.lastToType = toType
	m.lastToID = toID
	m.lastRelType = relType
	if m.createFn != nil {
		return m.createFn(ctx, fromType, fromID, toType, toID, relType)
	}
	return nil, nil
}

func (m *mockRelationshipMutationServicer) UnlinkEntities(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error {
	m.unlinkCalls++
	m.lastFromType = fromType
	m.lastFromID = fromID
	m.lastToType = toType
	m.lastToID = toID
	m.lastRelType = relType
	if m.unlinkFn != nil {
		return m.unlinkFn(ctx, fromType, fromID, toType, toID, relType)
	}
	return nil
}

type mockEntityResolver struct {
	resolveFn func(ctx context.Context, key string) (models.Entity, error)
	calls     int
	lastKey   string
}

func (m *mockEntityResolver) Resolve(ctx context.Context, key string) (models.Entity, error) {
	m.calls++
	m.lastKey = key
	if m.resolveFn != nil {
		return m.resolveFn(ctx, key)
	}
	return nil, nil
}

func TestMutationService_UpdateDelegates(t *testing.T) {
	t.Run("epic", func(t *testing.T) {
		mock := &mockEpicMutationServicer{
			updateEpicFn: func(_ context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
				if key != "E07" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.Title == nil || *updates.Title != "Updated Epic" {
					t.Fatalf("unexpected title: %#v", updates.Title)
				}
				epic := &models.Epic{}
				epic.Key = key
				epic.Title = *updates.Title
				return epic, nil
			},
		}
		svc := NewMutationService(mock, &mockFeatureMutationServicer{}, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		title := "Updated Epic"
		result, err := svc.UpdateEpic(context.Background(), "E07", services.EpicUpdates{Title: &title})
		if err != nil {
			t.Fatalf("UpdateEpic returned error: %v", err)
		}
		if mock.updateCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateCalls)
		}
		if result == nil || result.Title != "Updated Epic" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("feature", func(t *testing.T) {
		mock := &mockFeatureMutationServicer{
			updateFeatureFn: func(_ context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
				if key != "E07-F01" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.ExecutionOrder == nil || *updates.ExecutionOrder != 7 {
					t.Fatalf("unexpected execution_order: %#v", updates.ExecutionOrder)
				}
				feature := &models.Feature{}
				feature.Key = key
				feature.ExecutionOrder = updates.ExecutionOrder
				return feature, nil
			},
		}
		svc := NewMutationService(&mockEpicMutationServicer{}, mock, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		order := 7
		result, err := svc.UpdateFeature(context.Background(), "E07-F01", services.FeatureUpdates{ExecutionOrder: &order})
		if err != nil {
			t.Fatalf("UpdateFeature returned error: %v", err)
		}
		if mock.updateCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateCalls)
		}
		if result == nil || result.ExecutionOrder == nil || *result.ExecutionOrder != 7 {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("task", func(t *testing.T) {
		mock := &mockTaskMutationServicer{
			updateTaskFn: func(_ context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
				if key != "T-E07-F01-001" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.AgentType == nil || *updates.AgentType != "backend" {
					t.Fatalf("unexpected agent type: %#v", updates.AgentType)
				}
				task := &models.Task{}
				task.Key = key
				task.AgentType = updates.AgentType
				return task, nil
			},
		}
		svc := NewMutationService(&mockEpicMutationServicer{}, &mockFeatureMutationServicer{}, mock, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		agent := "backend"
		result, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", services.TaskUpdates{AgentType: &agent})
		if err != nil {
			t.Fatalf("UpdateTask returned error: %v", err)
		}
		if mock.updateCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateCalls)
		}
		if result == nil || result.AgentType == nil || *result.AgentType != "backend" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
}

func TestMutationService_TransitionDelegates(t *testing.T) {
	t.Run("epic", func(t *testing.T) {
		mock := &mockEpicMutationServicer{
			transitionEpicFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "E07" {
					t.Fatalf("unexpected key: %s", key)
				}
				if targetStatus != "active" {
					t.Fatalf("unexpected target status: %s", targetStatus)
				}
				if !opts.Force || opts.Reason != "reason" || opts.Agent != "viewer" {
					t.Fatalf("unexpected opts: %#v", opts)
				}
				return &services.TransitionResult{EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}
		svc := NewMutationService(mock, &mockFeatureMutationServicer{}, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		result, err := svc.TransitionEpic(context.Background(), "E07", "active", services.TransitionOptions{Force: true, Reason: "reason", Agent: "viewer"})
		if err != nil {
			t.Fatalf("TransitionEpic returned error: %v", err)
		}
		if mock.transitionCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionCalls)
		}
		if result == nil || result.EntityKey != "E07" || result.ToStatus != "active" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("feature", func(t *testing.T) {
		mock := &mockFeatureMutationServicer{
			transitionFeatureFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "E07-F01" || targetStatus != "completed" {
					t.Fatalf("unexpected call: %s %s", key, targetStatus)
				}
				if opts.Reason != "done" {
					t.Fatalf("unexpected opts: %#v", opts)
				}
				return &services.TransitionResult{EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}
		svc := NewMutationService(&mockEpicMutationServicer{}, mock, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		result, err := svc.TransitionFeature(context.Background(), "E07-F01", "completed", services.TransitionOptions{Reason: "done"})
		if err != nil {
			t.Fatalf("TransitionFeature returned error: %v", err)
		}
		if mock.transitionCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionCalls)
		}
		if result == nil || result.EntityKey != "E07-F01" || result.ToStatus != "completed" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("task", func(t *testing.T) {
		mock := &mockTaskMutationServicer{
			transitionTaskFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "T-E07-F01-001" || targetStatus != "in_progress" {
					t.Fatalf("unexpected call: %s %s", key, targetStatus)
				}
				if opts.Agent != "viewer" {
					t.Fatalf("unexpected opts: %#v", opts)
				}
				return &services.TransitionResult{EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}
		svc := NewMutationService(&mockEpicMutationServicer{}, &mockFeatureMutationServicer{}, mock, &mockNoteMutationServicer{}, &mockRelationshipMutationServicer{}, &mockEntityResolver{})

		result, err := svc.TransitionTask(context.Background(), "T-E07-F01-001", "in_progress", services.TransitionOptions{Agent: "viewer"})
		if err != nil {
			t.Fatalf("TransitionTask returned error: %v", err)
		}
		if mock.transitionCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionCalls)
		}
		if result == nil || result.EntityKey != "T-E07-F01-001" || result.ToStatus != "in_progress" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
}

func TestMutationService_NoteAndRelationshipDelegates(t *testing.T) {
	t.Run("add note resolves entity and delegates", func(t *testing.T) {
		resolver := &mockEntityResolver{
			resolveFn: func(_ context.Context, key string) (models.Entity, error) {
				if key != "E07" {
					t.Fatalf("unexpected key: %s", key)
				}
				return &models.Epic{BaseEntity: models.BaseEntity{ID: 42, Key: key}}, nil
			},
		}
		noteSvc := &mockNoteMutationServicer{
			addNoteFn: func(_ context.Context, entityType models.EntityType, entityKey, noteType, content, createdBy string) (*models.EntityNote, error) {
				if entityType != models.EntityTypeEpic {
					t.Fatalf("unexpected entity type: %s", entityType)
				}
				if entityKey != "E07" || noteType != "decision" || content != "Use normalized store" || createdBy != "viewer" {
					t.Fatalf("unexpected note args: %s %s %s %s", entityKey, noteType, content, createdBy)
				}
				return &models.EntityNote{EntityType: entityType, EntityID: 42, NoteType: models.NoteType(noteType), Content: content}, nil
			},
		}

		svc := NewMutationService(&mockEpicMutationServicer{}, &mockFeatureMutationServicer{}, &mockTaskMutationServicer{}, noteSvc, &mockRelationshipMutationServicer{}, resolver)
		note, err := svc.AddNote(context.Background(), "E07", "decision", "Use normalized store", "viewer")
		if err != nil {
			t.Fatalf("AddNote returned error: %v", err)
		}
		if resolver.calls != 1 {
			t.Fatalf("expected 1 resolve call, got %d", resolver.calls)
		}
		if noteSvc.addCalls != 1 {
			t.Fatalf("expected 1 note call, got %d", noteSvc.addCalls)
		}
		if note == nil || note.EntityID != 42 || note.NoteType != models.NoteType("decision") {
			t.Fatalf("unexpected note: %#v", note)
		}
	})

	t.Run("create relationship resolves both entities and delegates", func(t *testing.T) {
		resolver := &mockEntityResolver{
			resolveFn: func(_ context.Context, key string) (models.Entity, error) {
				switch key {
				case "E07":
					return &models.Epic{BaseEntity: models.BaseEntity{ID: 10, Key: key}}, nil
				case "T-E07-F01-002":
					return &models.Task{BaseEntity: models.BaseEntity{ID: 20, Key: key}}, nil
				default:
					t.Fatalf("unexpected key: %s", key)
				}
				return nil, nil
			},
		}
		relSvc := &mockRelationshipMutationServicer{
			createFn: func(_ context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) (*models.EntityRelationship, error) {
				if fromType != models.EntityTypeEpic || fromID != 10 || toType != models.EntityTypeTask || toID != 20 || relType != models.EntityRelRelatedTo {
					t.Fatalf("unexpected relationship args: %#v %#v %#v %#v %#v", fromType, fromID, toType, toID, relType)
				}
				return &models.EntityRelationship{FromEntityType: fromType, FromEntityID: fromID, ToEntityType: toType, ToEntityID: toID, RelationshipType: relType}, nil
			},
		}

		svc := NewMutationService(&mockEpicMutationServicer{}, &mockFeatureMutationServicer{}, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, relSvc, resolver)
		rel, err := svc.CreateRelationship(context.Background(), "E07", "T-E07-F01-002", "related_to")
		if err != nil {
			t.Fatalf("CreateRelationship returned error: %v", err)
		}
		if resolver.calls != 2 {
			t.Fatalf("expected 2 resolve calls, got %d", resolver.calls)
		}
		if relSvc.createCalls != 1 {
			t.Fatalf("expected 1 create call, got %d", relSvc.createCalls)
		}
		if rel == nil || rel.RelationshipType != models.EntityRelRelatedTo {
			t.Fatalf("unexpected relationship: %#v", rel)
		}
	})

	t.Run("delete relationship resolves both entities and delegates", func(t *testing.T) {
		resolver := &mockEntityResolver{
			resolveFn: func(_ context.Context, key string) (models.Entity, error) {
				switch key {
				case "E07-F01":
					return &models.Feature{BaseEntity: models.BaseEntity{ID: 11, Key: key}}, nil
				case "T-E07-F01-002":
					return &models.Task{BaseEntity: models.BaseEntity{ID: 12, Key: key}}, nil
				default:
					t.Fatalf("unexpected key: %s", key)
				}
				return nil, nil
			},
		}
		relSvc := &mockRelationshipMutationServicer{
			unlinkFn: func(_ context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error {
				if fromType != models.EntityTypeFeature || fromID != 11 || toType != models.EntityTypeTask || toID != 12 || relType != models.EntityRelDependsOn {
					t.Fatalf("unexpected delete args: %#v %#v %#v %#v %#v", fromType, fromID, toType, toID, relType)
				}
				return nil
			},
		}

		svc := NewMutationService(&mockEpicMutationServicer{}, &mockFeatureMutationServicer{}, &mockTaskMutationServicer{}, &mockNoteMutationServicer{}, relSvc, resolver)
		if err := svc.DeleteRelationship(context.Background(), "E07-F01", "depends_on", "T-E07-F01-002"); err != nil {
			t.Fatalf("DeleteRelationship returned error: %v", err)
		}
		if resolver.calls != 2 {
			t.Fatalf("expected 2 resolve calls, got %d", resolver.calls)
		}
		if relSvc.unlinkCalls != 1 {
			t.Fatalf("expected 1 unlink call, got %d", relSvc.unlinkCalls)
		}
	})
}

// TC-302: the resolver used by the viewer relationship transport must classify
// the complete, explicitly approved Question gate target vocabulary.
func TestResolveMutationKey_QuestionBlockEligibleTargets_TC302(t *testing.T) {
	tests := []struct {
		key      string
		wantType models.EntityType
		wantKey  string
	}{
		{key: "E39", wantType: models.EntityTypeEpic, wantKey: "E39"},
		{key: "e39-f03", wantType: models.EntityTypeFeature, wantKey: "E39-F03"},
		{key: "e39-f03-003", wantType: models.EntityTypeTask, wantKey: "T-E39-F03-003"},
		{key: "b039", wantType: models.EntityTypeBug, wantKey: "B039"},
		{key: "cc-039", wantType: models.EntityTypeChange, wantKey: "CC-039"},
		{key: "td-039", wantType: models.EntityTypeTechDebt, wantKey: "TD-039"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotType, gotKey, err := resolveMutationKey(tt.key)
			if err != nil {
				t.Fatalf("TC-302 resolveMutationKey(%q) error = %v", tt.key, err)
			}
			if gotType != tt.wantType || gotKey != tt.wantKey {
				t.Fatalf("TC-302 resolveMutationKey(%q) = (%q, %q), want (%q, %q)", tt.key, gotType, gotKey, tt.wantType, tt.wantKey)
			}
		})
	}
}
