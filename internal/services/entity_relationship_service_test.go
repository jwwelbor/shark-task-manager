package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// MockEntityRelationshipRepository implements EntityRelationshipRepository for testing.
type MockEntityRelationshipRepository struct {
	CreateFunc                  func(ctx context.Context, rel *models.EntityRelationship) error
	DeleteFunc                  func(ctx context.Context, id int64) error
	DeleteByEntitiesAndTypeFunc func(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error
	GetByEntityFunc             func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error)
	GetOutgoingFunc             func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
	GetIncomingFunc             func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
}

func (m *MockEntityRelationshipRepository) Create(ctx context.Context, rel *models.EntityRelationship) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, rel)
	}
	return nil
}

func (m *MockEntityRelationshipRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockEntityRelationshipRepository) DeleteByEntitiesAndType(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error {
	if m.DeleteByEntitiesAndTypeFunc != nil {
		return m.DeleteByEntitiesAndTypeFunc(ctx, fromType, fromID, toType, toID, relType)
	}
	return nil
}

func (m *MockEntityRelationshipRepository) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
	if m.GetByEntityFunc != nil {
		return m.GetByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *MockEntityRelationshipRepository) GetOutgoing(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	if m.GetOutgoingFunc != nil {
		return m.GetOutgoingFunc(ctx, entityType, entityID, relTypes)
	}
	return nil, nil
}

func (m *MockEntityRelationshipRepository) GetIncoming(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	if m.GetIncomingFunc != nil {
		return m.GetIncomingFunc(ctx, entityType, entityID, relTypes)
	}
	return nil, nil
}

func TestNewEntityRelationshipService(t *testing.T) {
	t.Run("panics on nil repo", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil repo, got none")
			}
		}()
		NewEntityRelationshipService(nil, nil)
	})

	t.Run("succeeds with valid repo", func(t *testing.T) {
		svc := NewEntityRelationshipService(&MockEntityRelationshipRepository{}, nil)
		if svc == nil {
			t.Error("expected non-nil service")
		}
	})
}

func TestCreateRelationship_Success(t *testing.T) {
	var createdRel *models.EntityRelationship
	mockRepo := &MockEntityRelationshipRepository{
		CreateFunc: func(ctx context.Context, rel *models.EntityRelationship) error {
			rel.ID = 42
			createdRel = rel
			return nil
		},
		// No outgoing edges, so cycle detection finds nothing
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	rel, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	if rel == nil {
		t.Fatal("expected non-nil relationship")
	}
	if rel.ID != 42 {
		t.Errorf("expected ID 42, got %d", rel.ID)
	}
	if createdRel == nil {
		t.Fatal("expected Create to be called")
	}
	if createdRel.FromEntityType != models.EntityTypeTask {
		t.Errorf("expected from_entity_type 'task', got %s", createdRel.FromEntityType)
	}
	if createdRel.RelationshipType != models.EntityRelDependsOn {
		t.Errorf("expected relationship_type 'depends_on', got %s", createdRel.RelationshipType)
	}
}

func TestCreateRelationship_ValidationError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{}
	svc := NewEntityRelationshipService(mockRepo, nil)

	// Invalid relationship type
	_, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, "invalid_type")
	if err == nil {
		t.Fatal("expected error for invalid relationship type")
	}
}

func TestCreateRelationship_SelfReference(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{}
	svc := NewEntityRelationshipService(mockRepo, nil)

	_, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
}

func TestCreateRelationship_CycleDetected(t *testing.T) {
	// A depends_on B already exists. Trying to create B depends_on A should fail.
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			// B (id=2) has an outgoing depends_on edge to A (id=1)
			// Wait -- we're simulating existing state: A depends_on B
			// So A (id=1) has outgoing depends_on to B (id=2).
			// When trying B->A, DFS starts from A (toID=1, since from=B, to=A),
			// and follows outgoing edges from A.
			// Actually: from=B(2), to=A(1). DFS starts from to=(task,1).
			// From (task,1), GetOutgoing returns edge to (task,2) -- A depends_on B.
			// Then from (task,2) which is B, we check if (task,2)==target(task,2). Yes! Cycle.
			if entityID == 1 {
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
				}, nil
			}
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	_, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 2, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err == nil {
		t.Fatal("expected error for cycle detection")
	}
}

func TestCreateRelationship_NoCycleCheckForNonCyclicType(t *testing.T) {
	createCalled := false
	mockRepo := &MockEntityRelationshipRepository{
		CreateFunc: func(ctx context.Context, rel *models.EntityRelationship) error {
			createCalled = true
			rel.ID = 1
			return nil
		},
		// GetOutgoing is always called for duplicate detection, but non-cyclic
		// types still skip the DFS cycle walk.
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	// related_to is not a cyclic type
	rel, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelRelatedTo)
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	if rel == nil {
		t.Fatal("expected non-nil relationship")
	}
	if !createCalled {
		t.Error("expected Create to be called")
	}
}

func TestCreateRelationship_CycleCheckForCrossEntityType(t *testing.T) {
	createCalled := false
	mockRepo := &MockEntityRelationshipRepository{
		CreateFunc: func(ctx context.Context, rel *models.EntityRelationship) error {
			createCalled = true
			rel.ID = 1
			return nil
		},
		// GetOutgoing IS called for cross-entity type depends_on (cycle detection is universal)
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			return nil, nil // No outgoing edges, so no cycle
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	// depends_on across different entity types should still check for cycles
	rel, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeFeature, 2, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	if rel == nil {
		t.Fatal("expected non-nil relationship")
	}
	if !createCalled {
		t.Error("expected Create to be called")
	}
}

func TestDeleteRelationship(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deletedID := int64(0)
		mockRepo := &MockEntityRelationshipRepository{
			DeleteFunc: func(ctx context.Context, id int64) error {
				deletedID = id
				return nil
			},
		}

		svc := NewEntityRelationshipService(mockRepo, nil)
		err := svc.DeleteRelationship(ctx(), 42)
		if err != nil {
			t.Fatalf("DeleteRelationship() error = %v", err)
		}
		if deletedID != 42 {
			t.Errorf("expected Delete called with id 42, got %d", deletedID)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		mockRepo := &MockEntityRelationshipRepository{
			DeleteFunc: func(ctx context.Context, id int64) error {
				return fmt.Errorf("not found")
			},
		}

		svc := NewEntityRelationshipService(mockRepo, nil)
		err := svc.DeleteRelationship(ctx(), 999)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestUnlinkEntities(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		called := false
		mockRepo := &MockEntityRelationshipRepository{
			DeleteByEntitiesAndTypeFunc: func(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error {
				called = true
				if fromType != models.EntityTypeTask || fromID != 1 || toType != models.EntityTypeTask || toID != 2 || relType != models.EntityRelDependsOn {
					t.Error("unexpected parameters")
				}
				return nil
			},
		}

		svc := NewEntityRelationshipService(mockRepo, nil)
		err := svc.UnlinkEntities(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelDependsOn)
		if err != nil {
			t.Fatalf("UnlinkEntities() error = %v", err)
		}
		if !called {
			t.Error("expected DeleteByEntitiesAndType to be called")
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		mockRepo := &MockEntityRelationshipRepository{
			DeleteByEntitiesAndTypeFunc: func(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error {
				return fmt.Errorf("not found")
			},
		}

		svc := NewEntityRelationshipService(mockRepo, nil)
		err := svc.UnlinkEntities(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelDependsOn)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetRelationships(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			if entityType != models.EntityTypeTask || entityID != 1 {
				t.Errorf("unexpected parameters: %s, %d", entityType, entityID)
			}
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
				{ID: 2, FromEntityType: models.EntityTypeFeature, FromEntityID: 5, ToEntityType: models.EntityTypeTask, ToEntityID: 1, RelationshipType: models.EntityRelRelatedTo},
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)
	rels, err := svc.GetRelationships(ctx(), models.EntityTypeTask, 1)
	if err != nil {
		t.Fatalf("GetRelationships() error = %v", err)
	}
	if len(rels) != 2 {
		t.Fatalf("expected 2 relationships, got %d", len(rels))
	}
}

func TestGetOutgoing(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			if entityType != models.EntityTypeTask || entityID != 1 {
				t.Errorf("unexpected parameters")
			}
			if len(relTypes) != 1 || relTypes[0] != models.EntityRelDependsOn {
				t.Errorf("unexpected relTypes: %v", relTypes)
			}
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)
	rels, err := svc.GetOutgoing(ctx(), models.EntityTypeTask, 1, []models.EntityRelationshipType{models.EntityRelDependsOn})
	if err != nil {
		t.Fatalf("GetOutgoing() error = %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
}

func TestGetIncoming(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetIncomingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			if entityType != models.EntityTypeTask || entityID != 2 {
				t.Errorf("unexpected parameters")
			}
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)
	rels, err := svc.GetIncoming(ctx(), models.EntityTypeTask, 2, nil)
	if err != nil {
		t.Fatalf("GetIncoming() error = %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
}

func TestDetectCycle_Simple(t *testing.T) {
	// A(1) depends_on B(2) exists. Check if B(2) depends_on A(1) creates cycle.
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			// Starting DFS from A(1) (the toID). A has outgoing depends_on to B(2).
			if entityID == 1 {
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
				}, nil
			}
			// B(2) has no outgoing depends_on
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	// Trying to add B(2) depends_on A(1). DFS starts from A(1), follows to B(2).
	// B(2) is the target (fromID). Cycle detected!
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 2, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if !hasCycle {
		t.Error("expected cycle to be detected")
	}
}

func TestDetectCycle_Transitive(t *testing.T) {
	// A(1) depends_on B(2), B(2) depends_on C(3). Check if C(3) depends_on A(1) creates cycle.
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			switch entityID {
			case 1:
				// A depends_on B
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
				}, nil
			case 2:
				// B depends_on C
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 2, ToEntityType: models.EntityTypeTask, ToEntityID: 3, RelationshipType: models.EntityRelDependsOn},
				}, nil
			default:
				return nil, nil
			}
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	// Trying to add C(3) depends_on A(1). DFS starts from A(1), follows A->B->C.
	// C(3) is the target (fromID=3). When DFS reaches C(3), it matches target. Cycle!
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 3, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if !hasCycle {
		t.Error("expected transitive cycle to be detected")
	}
}

func TestDetectCycle_NoCycle(t *testing.T) {
	// A(1) depends_on B(2). Check if C(3) depends_on A(1) -- no cycle.
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			if entityID == 1 {
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
				}, nil
			}
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	// Trying to add C(3) depends_on A(1). DFS starts from A(1), follows A->B.
	// Never reaches C(3). No cycle.
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 3, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if hasCycle {
		t.Error("expected no cycle")
	}
}

func TestDetectCycle_NonCyclicRelType(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{}
	svc := NewEntityRelationshipService(mockRepo, nil)

	// related_to is not cyclic, should return false immediately
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelRelatedTo)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if hasCycle {
		t.Error("expected no cycle for non-cyclic relationship type")
	}
}

func TestDetectCycle_CrossEntityType_NoCycle(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			// No outgoing edges
			return nil, nil
		},
	}
	svc := NewEntityRelationshipService(mockRepo, nil)

	// Cross-entity type depends_on now runs DFS; no outgoing edges means no cycle
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 1, models.EntityTypeFeature, 2, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if hasCycle {
		t.Error("expected no cycle when no outgoing edges exist")
	}
}

func TestDetectCycle_CrossEntityType_CycleDetected(t *testing.T) {
	// Task(1) depends_on Bug(10) already exists. Trying Bug(10) depends_on Task(1).
	// DFS starts from Task(1) (toID), follows outgoing to Bug(10), which is the target (fromID=10).
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			if entityType == models.EntityTypeTask && entityID == 1 {
				return []*models.EntityRelationship{
					{FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeBug, ToEntityID: 10, RelationshipType: models.EntityRelDependsOn},
				}, nil
			}
			return nil, nil
		},
	}
	svc := NewEntityRelationshipService(mockRepo, nil)

	// Trying to add Bug(10) depends_on Task(1). DFS starts from Task(1), follows to Bug(10).
	// Bug(10) == target node{bug, 10}. Cycle detected!
	hasCycle, err := svc.DetectCycle(ctx(), models.EntityTypeBug, 10, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err != nil {
		t.Fatalf("DetectCycle() error = %v", err)
	}
	if !hasCycle {
		t.Error("expected cross-entity cycle to be detected")
	}
}

func TestDetectCycle_RepoError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	_, err := svc.DetectCycle(ctx(), models.EntityTypeTask, 2, models.EntityTypeTask, 1, models.EntityRelDependsOn)
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestCreateRelationship_RepoError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		CreateFunc: func(ctx context.Context, rel *models.EntityRelationship) error {
			return fmt.Errorf("UNIQUE constraint failed")
		},
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			return nil, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)

	_, err := svc.CreateRelationship(ctx(), models.EntityTypeTask, 1, models.EntityTypeTask, 2, models.EntityRelDependsOn)
	if err == nil {
		t.Fatal("expected error from repo Create")
	}
}

func TestGetRelationships_RepoError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	svc := NewEntityRelationshipService(mockRepo, nil)
	_, err := svc.GetRelationships(ctx(), models.EntityTypeTask, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// MockTaskByIDResolver implements TaskByIDResolver for testing.
type MockTaskByIDResolver struct {
	GetByIDFunc func(ctx context.Context, id int64) (*models.Task, error)
}

func (m *MockTaskByIDResolver) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mock")
}

func TestGetTaskRelationships_NoRelationships(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return []*models.EntityRelationship{}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	result, err := svc.GetTaskRelationships(ctx(), 10, nil)
	if err != nil {
		t.Fatalf("GetTaskRelationships() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestGetTaskRelationships_WithTypeFilter(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 20, RelationshipType: models.EntityRelDependsOn},
				{ID: 2, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 30, RelationshipType: models.EntityRelDependsOn},
				{ID: 3, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 40, RelationshipType: models.EntityRelBlocks},
			}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: id, Key: fmt.Sprintf("T-MOCK-%03d", id), Title: fmt.Sprintf("Task %d", id)},
				Status:     "todo",
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	result, err := svc.GetTaskRelationships(ctx(), 10, []string{"depends_on"})
	if err != nil {
		t.Fatalf("GetTaskRelationships() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	for _, r := range result {
		if r.RelationshipType != "depends_on" {
			t.Errorf("expected relationship_type 'depends_on', got '%s'", r.RelationshipType)
		}
	}
}

func TestGetTaskRelationships_SkipsNonTaskRelationships(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return []*models.EntityRelationship{
				// task-to-task: should be included
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 20, RelationshipType: models.EntityRelDependsOn},
				// epic-to-task: should be excluded
				{ID: 2, FromEntityType: models.EntityTypeEpic, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 10, RelationshipType: models.EntityRelRelatedTo},
				// task-to-feature: should be excluded
				{ID: 3, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeFeature, ToEntityID: 5, RelationshipType: models.EntityRelRelatedTo},
			}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: id, Key: fmt.Sprintf("T-MOCK-%03d", id), Title: fmt.Sprintf("Task %d", id)},
				Status:     "todo",
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	result, err := svc.GetTaskRelationships(ctx(), 10, nil)
	if err != nil {
		t.Fatalf("GetTaskRelationships() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (task-to-task only), got %d", len(result))
	}
	if result[0].TaskKey != "T-MOCK-020" {
		t.Errorf("expected task key T-MOCK-020, got %s", result[0].TaskKey)
	}
}

func TestGetTaskRelationships_ResolverReturnsError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 99, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return nil, fmt.Errorf("task %d not found", id)
		},
	}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	// When resolver fails, the relationship is silently skipped (matching existing behavior)
	result, err := svc.GetTaskRelationships(ctx(), 10, nil)
	if err != nil {
		t.Fatalf("GetTaskRelationships() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results when resolver fails, got %d", len(result))
	}
}

func TestGetTaskRelationships_RepoError(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	mockResolver := &MockTaskByIDResolver{}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	_, err := svc.GetTaskRelationships(ctx(), 10, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Errorf("expected error to contain 'db error', got: %v", err)
	}
}

func TestGetTaskBlockedBy_ReturnsDependencies(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			if entityType != models.EntityTypeTask || entityID != 10 {
				t.Errorf("unexpected parameters: %s, %d", entityType, entityID)
			}
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 20, RelationshipType: models.EntityRelDependsOn},
				{ID: 2, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 30, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: id, Key: fmt.Sprintf("T-MOCK-%03d", id), Title: fmt.Sprintf("Task %d", id)},
				Status:     "in_progress",
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	result, err := svc.GetTaskBlockedBy(ctx(), 10)
	if err != nil {
		t.Fatalf("GetTaskBlockedBy() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	for _, r := range result {
		if r.RelationshipType != "depends_on" {
			t.Errorf("expected relationship_type 'depends_on', got '%s'", r.RelationshipType)
		}
		if r.Direction != "outgoing" {
			t.Errorf("expected direction 'outgoing', got '%s'", r.Direction)
		}
	}
}

func TestGetTaskBlocks_ReturnsIncomingAndOutgoing(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{
		GetIncomingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			// Incoming depends_on: task 40 depends on task 10
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 40, ToEntityType: models.EntityTypeTask, ToEntityID: 10, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
		GetOutgoingFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
			// Outgoing blocks: task 10 blocks task 50
			return []*models.EntityRelationship{
				{ID: 2, FromEntityType: models.EntityTypeTask, FromEntityID: 10, ToEntityType: models.EntityTypeTask, ToEntityID: 50, RelationshipType: models.EntityRelBlocks},
			}, nil
		},
	}
	mockResolver := &MockTaskByIDResolver{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: id, Key: fmt.Sprintf("T-MOCK-%03d", id), Title: fmt.Sprintf("Task %d", id)},
				Status:     "todo",
			}, nil
		},
	}

	svc := NewEntityRelationshipService(mockRepo, mockResolver)

	result, err := svc.GetTaskBlocks(ctx(), 10)
	if err != nil {
		t.Fatalf("GetTaskBlocks() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
}

func TestGetTaskRelationships_NilResolver(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{}
	svc := NewEntityRelationshipService(mockRepo, nil)

	_, err := svc.GetTaskRelationships(ctx(), 10, nil)
	if err == nil {
		t.Fatal("expected error when resolver is nil")
	}
}

func TestNewEntityRelationshipService_NilResolverAllowed(t *testing.T) {
	mockRepo := &MockEntityRelationshipRepository{}
	svc := NewEntityRelationshipService(mockRepo, nil)
	if svc == nil {
		t.Error("expected non-nil service with nil resolver")
	}
}

func TestExistingTests_UpdatedConstructor(t *testing.T) {
	// Verify existing functionality still works with nil resolver
	mockRepo := &MockEntityRelationshipRepository{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error) {
			return []*models.EntityRelationship{
				{ID: 1, FromEntityType: models.EntityTypeTask, FromEntityID: 1, ToEntityType: models.EntityTypeTask, ToEntityID: 2, RelationshipType: models.EntityRelDependsOn},
			}, nil
		},
	}
	svc := NewEntityRelationshipService(mockRepo, nil)
	rels, err := svc.GetRelationships(ctx(), models.EntityTypeTask, 1)
	if err != nil {
		t.Fatalf("GetRelationships() error = %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
}

// ctx is a helper to create a background context for tests.
func ctx() context.Context {
	return context.Background()
}
