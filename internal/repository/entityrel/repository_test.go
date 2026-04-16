package entityrel

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// cleanupEntityRelationships removes all test entity relationships before each test.
func cleanupEntityRelationships(ctx context.Context) {
	database := test.GetTestDB()
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships")
}

func TestEntityRelationshipRepository_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	t.Run("happy path", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelRelatedTo,
		}

		err := repo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if rel.ID == 0 {
			t.Error("expected ID to be set after creation")
		}

		// Verify in database
		var count int
		err = database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_relationships WHERE id = ?", rel.ID).Scan(&count)
		if err != nil {
			t.Fatalf("failed to query entity_relationships: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 relationship in database, got %d", count)
		}
	})

	t.Run("duplicate detection", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelDependsOn,
		}

		err := repo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		// Try to create duplicate
		rel2 := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelDependsOn,
		}

		err = repo.Create(ctx, rel2)
		if err == nil {
			t.Fatal("expected error for duplicate relationship, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("validation - zero from_entity_id", func(t *testing.T) {
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     0,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       1,
			RelationshipType: models.RelDependsOn,
		}

		err := repo.Create(ctx, rel)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(err.Error(), "from_entity_id") {
			t.Errorf("expected from_entity_id error, got: %v", err)
		}
	})

	t.Run("validation - self-relationship", func(t *testing.T) {
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     42,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       42,
			RelationshipType: models.RelDependsOn,
		}

		err := repo.Create(ctx, rel)
		if err == nil {
			t.Fatal("expected validation error for self-relationship, got nil")
		}
		if !strings.Contains(err.Error(), "itself") {
			t.Errorf("expected 'itself' error, got: %v", err)
		}
	})

	t.Run("validation - invalid relationship type", func(t *testing.T) {
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     1,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       2,
			RelationshipType: "invalid_type",
		}

		err := repo.Create(ctx, rel)
		if err == nil {
			t.Fatal("expected validation error for invalid type, got nil")
		}
		if !strings.Contains(err.Error(), "invalid relationship_type") {
			t.Errorf("expected 'invalid relationship_type' error, got: %v", err)
		}
	})
}

func TestEntityRelationshipRepository_Delete(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	t.Run("happy path", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelRelatedTo,
		}
		if err := repo.Create(ctx, rel); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		err := repo.Delete(ctx, rel.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deleted
		var count int
		err = database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_relationships WHERE id = ?", rel.ID).Scan(&count)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 relationships after delete, got %d", count)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.Delete(ctx, 999999)
		if err == nil {
			t.Fatal("expected error for non-existent ID, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestEntityRelationshipRepository_DeleteByEntitiesAndType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	t.Run("happy path", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelDependsOn,
		}
		if err := repo.Create(ctx, rel); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		err := repo.DeleteByEntitiesAndType(ctx,
			models.EntityTypeEpic, epicID,
			models.EntityTypeFeature, featureID,
			models.RelDependsOn,
		)
		if err != nil {
			t.Fatalf("DeleteByEntitiesAndType() error = %v", err)
		}

		// Verify deleted
		var count int
		err = database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_relationships WHERE id = ?", rel.ID).Scan(&count)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 relationships after delete, got %d", count)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.DeleteByEntitiesAndType(ctx,
			models.EntityTypeTask, 999999,
			models.EntityTypeTask, 999998,
			models.RelDependsOn,
		)
		if err == nil {
			t.Fatal("expected error for non-existent relationship, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestEntityRelationshipRepository_GetByEntity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	// Create outgoing relationship: epic -> feature (depends_on)
	rel1 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeEpic,
		FromEntityID:     epicID,
		ToEntityType:     models.EntityTypeFeature,
		ToEntityID:       featureID,
		RelationshipType: models.RelDependsOn,
	}
	if err := repo.Create(ctx, rel1); err != nil {
		t.Fatalf("Create rel1 error = %v", err)
	}

	// Create incoming relationship: feature -> epic (related_to)
	rel2 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeFeature,
		FromEntityID:     featureID,
		ToEntityType:     models.EntityTypeEpic,
		ToEntityID:       epicID,
		RelationshipType: models.RelRelatedTo,
	}
	if err := repo.Create(ctx, rel2); err != nil {
		t.Fatalf("Create rel2 error = %v", err)
	}

	t.Run("bidirectional results for epic", func(t *testing.T) {
		rels, err := repo.GetByEntity(ctx, models.EntityTypeEpic, epicID)
		if err != nil {
			t.Fatalf("GetByEntity() error = %v", err)
		}

		// Should find both: one where epic is from, one where epic is to
		if len(rels) != 2 {
			t.Errorf("expected 2 relationships, got %d", len(rels))
		}
	})

	t.Run("bidirectional results for feature", func(t *testing.T) {
		rels, err := repo.GetByEntity(ctx, models.EntityTypeFeature, featureID)
		if err != nil {
			t.Fatalf("GetByEntity() error = %v", err)
		}

		if len(rels) != 2 {
			t.Errorf("expected 2 relationships, got %d", len(rels))
		}
	})

	t.Run("no results for unknown entity", func(t *testing.T) {
		rels, err := repo.GetByEntity(ctx, models.EntityTypeTask, 999999)
		if err != nil {
			t.Fatalf("GetByEntity() error = %v", err)
		}

		if len(rels) != 0 {
			t.Errorf("expected 0 relationships, got %d", len(rels))
		}
	})
}

func TestEntityRelationshipRepository_GetOutgoing(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	// Create two outgoing relationships from epic
	rel1 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeEpic,
		FromEntityID:     epicID,
		ToEntityType:     models.EntityTypeFeature,
		ToEntityID:       featureID,
		RelationshipType: models.RelDependsOn,
	}
	if err := repo.Create(ctx, rel1); err != nil {
		t.Fatalf("Create rel1 error = %v", err)
	}

	rel2 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeEpic,
		FromEntityID:     epicID,
		ToEntityType:     models.EntityTypeFeature,
		ToEntityID:       featureID,
		RelationshipType: models.RelRelatedTo,
	}
	if err := repo.Create(ctx, rel2); err != nil {
		t.Fatalf("Create rel2 error = %v", err)
	}

	// Create an incoming relationship (should not appear in outgoing)
	rel3 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeFeature,
		FromEntityID:     featureID,
		ToEntityType:     models.EntityTypeEpic,
		ToEntityID:       epicID,
		RelationshipType: models.RelFollows,
	}
	if err := repo.Create(ctx, rel3); err != nil {
		t.Fatalf("Create rel3 error = %v", err)
	}

	t.Run("without type filter", func(t *testing.T) {
		rels, err := repo.GetOutgoing(ctx, models.EntityTypeEpic, epicID, nil)
		if err != nil {
			t.Fatalf("GetOutgoing() error = %v", err)
		}

		if len(rels) != 2 {
			t.Errorf("expected 2 outgoing relationships, got %d", len(rels))
		}

		// All should have epic as from
		for _, r := range rels {
			if r.FromEntityType != models.EntityTypeEpic || r.FromEntityID != epicID {
				t.Errorf("unexpected from entity: %s(%d)", r.FromEntityType, r.FromEntityID)
			}
		}
	})

	t.Run("with type filter - single type", func(t *testing.T) {
		rels, err := repo.GetOutgoing(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelDependsOn})
		if err != nil {
			t.Fatalf("GetOutgoing() error = %v", err)
		}

		if len(rels) != 1 {
			t.Errorf("expected 1 outgoing depends_on relationship, got %d", len(rels))
		}
		if len(rels) > 0 && rels[0].RelationshipType != models.EntityRelDependsOn {
			t.Errorf("expected depends_on type, got %s", rels[0].RelationshipType)
		}
	})

	t.Run("with type filter - multiple types", func(t *testing.T) {
		rels, err := repo.GetOutgoing(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelDependsOn, models.EntityRelRelatedTo})
		if err != nil {
			t.Fatalf("GetOutgoing() error = %v", err)
		}

		if len(rels) != 2 {
			t.Errorf("expected 2 outgoing relationships, got %d", len(rels))
		}
	})

	t.Run("with type filter - no matches", func(t *testing.T) {
		rels, err := repo.GetOutgoing(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelLinkedTo})
		if err != nil {
			t.Fatalf("GetOutgoing() error = %v", err)
		}

		if len(rels) != 0 {
			t.Errorf("expected 0 relationships, got %d", len(rels))
		}
	})
}

func TestEntityRelationshipRepository_GetIncoming(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityRelationshipRepository(db)

	cleanupEntityRelationships(ctx)
	epicID, featureID := test.SeedTestData()

	// Create incoming relationships TO the epic
	rel1 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeFeature,
		FromEntityID:     featureID,
		ToEntityType:     models.EntityTypeEpic,
		ToEntityID:       epicID,
		RelationshipType: models.RelDependsOn,
	}
	if err := repo.Create(ctx, rel1); err != nil {
		t.Fatalf("Create rel1 error = %v", err)
	}

	rel2 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeFeature,
		FromEntityID:     featureID,
		ToEntityType:     models.EntityTypeEpic,
		ToEntityID:       epicID,
		RelationshipType: models.RelBlocks,
	}
	if err := repo.Create(ctx, rel2); err != nil {
		t.Fatalf("Create rel2 error = %v", err)
	}

	// Create an outgoing relationship FROM the epic (should not appear in incoming)
	rel3 := &models.EntityRelationship{
		FromEntityType:   models.EntityTypeEpic,
		FromEntityID:     epicID,
		ToEntityType:     models.EntityTypeFeature,
		ToEntityID:       featureID,
		RelationshipType: models.RelFollows,
	}
	if err := repo.Create(ctx, rel3); err != nil {
		t.Fatalf("Create rel3 error = %v", err)
	}

	t.Run("without type filter", func(t *testing.T) {
		rels, err := repo.GetIncoming(ctx, models.EntityTypeEpic, epicID, nil)
		if err != nil {
			t.Fatalf("GetIncoming() error = %v", err)
		}

		if len(rels) != 2 {
			t.Errorf("expected 2 incoming relationships, got %d", len(rels))
		}

		// All should have epic as to
		for _, r := range rels {
			if r.ToEntityType != models.EntityTypeEpic || r.ToEntityID != epicID {
				t.Errorf("unexpected to entity: %s(%d)", r.ToEntityType, r.ToEntityID)
			}
		}
	})

	t.Run("with type filter - single type", func(t *testing.T) {
		rels, err := repo.GetIncoming(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelBlocks})
		if err != nil {
			t.Fatalf("GetIncoming() error = %v", err)
		}

		if len(rels) != 1 {
			t.Errorf("expected 1 incoming blocks relationship, got %d", len(rels))
		}
		if len(rels) > 0 && rels[0].RelationshipType != models.EntityRelBlocks {
			t.Errorf("expected blocks type, got %s", rels[0].RelationshipType)
		}
	})

	t.Run("with type filter - multiple types", func(t *testing.T) {
		rels, err := repo.GetIncoming(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelDependsOn, models.EntityRelBlocks})
		if err != nil {
			t.Fatalf("GetIncoming() error = %v", err)
		}

		if len(rels) != 2 {
			t.Errorf("expected 2 incoming relationships, got %d", len(rels))
		}
	})

	t.Run("with type filter - no matches", func(t *testing.T) {
		rels, err := repo.GetIncoming(ctx, models.EntityTypeEpic, epicID,
			[]models.EntityRelationshipType{models.EntityRelLinkedTo})
		if err != nil {
			t.Fatalf("GetIncoming() error = %v", err)
		}

		if len(rels) != 0 {
			t.Errorf("expected 0 relationships, got %d", len(rels))
		}
	})
}

// TestEntityRelFeatureKeyAdapter_ListRelatedFeatureKeys validates IS-3 and AC-3:
// The new EntityRelFeatureKeyAdapter queries entity_relationships for feature-to-feature
// relationships and returns the related feature keys.
func TestEntityRelFeatureKeyAdapter_ListRelatedFeatureKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	relRepo := NewEntityRelationshipRepository(db)
	adapter := NewEntityRelFeatureKeyAdapter(db)

	cleanupEntityRelationships(ctx)
	_, featureID := test.SeedTestData()

	// Create a second feature to relate to
	var featureID2 int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO features (key, title, status, epic_id, file_path) VALUES ('E01-F02', 'Feature 2', 'todo', (SELECT epic_id FROM features WHERE id=?), '') RETURNING id`,
		featureID,
	).Scan(&featureID2)
	if err != nil {
		// Try without RETURNING (older sqlite)
		_, insertErr := database.ExecContext(ctx,
			`INSERT INTO features (key, title, status, epic_id, file_path) VALUES ('E01-F02-TEST', 'Feature 2 Test', 'todo', (SELECT epic_id FROM features WHERE id=?), '')`,
			featureID,
		)
		if insertErr != nil {
			t.Fatalf("Failed to create second feature: %v", insertErr)
		}
		getErr := database.QueryRowContext(ctx, `SELECT id FROM features WHERE key='E01-F02-TEST'`).Scan(&featureID2)
		if getErr != nil {
			t.Fatalf("Failed to get second feature ID: %v", getErr)
		}
	}
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id=?", featureID2)

	t.Run("happy path — related feature appears in output", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		// Create feature-to-feature relationship in entity_relationships
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeFeature,
			FromEntityID:     featureID,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID2,
			RelationshipType: models.RelRelatedTo,
		}
		if err := relRepo.Create(ctx, rel); err != nil {
			t.Fatalf("Create relationship error = %v", err)
		}

		keys, err := adapter.ListRelatedFeatureKeys(ctx, featureID)
		if err != nil {
			t.Fatalf("ListRelatedFeatureKeys() error = %v", err)
		}

		if len(keys) != 1 {
			t.Errorf("expected 1 related feature key, got %d: %v", len(keys), keys)
		}
	})

	t.Run("empty result — no relationships", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		keys, err := adapter.ListRelatedFeatureKeys(ctx, featureID)
		if err != nil {
			t.Fatalf("ListRelatedFeatureKeys() error = %v", err)
		}

		if keys == nil {
			t.Error("expected non-nil slice, got nil")
		}
		if len(keys) != 0 {
			t.Errorf("expected 0 keys, got %d: %v", len(keys), keys)
		}
	})

	t.Run("bidirectional — adapter finds both incoming and outgoing", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		// featureID2 → featureID (incoming to featureID)
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeFeature,
			FromEntityID:     featureID2,
			ToEntityType:     models.EntityTypeFeature,
			ToEntityID:       featureID,
			RelationshipType: models.RelRelatedTo,
		}
		if err := relRepo.Create(ctx, rel); err != nil {
			t.Fatalf("Create relationship error = %v", err)
		}

		keys, err := adapter.ListRelatedFeatureKeys(ctx, featureID)
		if err != nil {
			t.Fatalf("ListRelatedFeatureKeys() error = %v", err)
		}

		if len(keys) != 1 {
			t.Errorf("expected 1 related feature key (incoming), got %d: %v", len(keys), keys)
		}
	})
}

// TestEntityRelEpicKeyAdapter_ListRelatedEpicKeys validates IS-3 and AC-3:
// The new EntityRelEpicKeyAdapter queries entity_relationships for epic-to-epic
// relationships and returns the related epic keys.
func TestEntityRelEpicKeyAdapter_ListRelatedEpicKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	relRepo := NewEntityRelationshipRepository(db)
	adapter := NewEntityRelEpicKeyAdapter(db)

	cleanupEntityRelationships(ctx)
	epicID, _ := test.SeedTestData()

	// Create a second epic to relate to
	var epicID2 int64
	_, insertErr := database.ExecContext(ctx,
		`INSERT INTO epics (key, title, status, priority, file_path) VALUES ('E02-TEST', 'Epic 2 Test', 'todo', 'medium', '') ON CONFLICT(key) DO NOTHING`,
	)
	if insertErr != nil {
		t.Fatalf("Failed to create second epic: %v", insertErr)
	}
	getErr := database.QueryRowContext(ctx, `SELECT id FROM epics WHERE key='E02-TEST'`).Scan(&epicID2)
	if getErr != nil {
		t.Fatalf("Failed to get second epic ID: %v", getErr)
	}
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE key='E02-TEST'")

	t.Run("happy path — related epic appears in output", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeEpic,
			FromEntityID:     epicID,
			ToEntityType:     models.EntityTypeEpic,
			ToEntityID:       epicID2,
			RelationshipType: models.RelRelatedTo,
		}
		if err := relRepo.Create(ctx, rel); err != nil {
			t.Fatalf("Create relationship error = %v", err)
		}

		keys, err := adapter.ListRelatedEpicKeys(ctx, epicID)
		if err != nil {
			t.Fatalf("ListRelatedEpicKeys() error = %v", err)
		}

		if len(keys) != 1 {
			t.Errorf("expected 1 related epic key, got %d: %v", len(keys), keys)
		}
	})

	t.Run("empty result — no relationships", func(t *testing.T) {
		cleanupEntityRelationships(ctx)

		keys, err := adapter.ListRelatedEpicKeys(ctx, epicID)
		if err != nil {
			t.Fatalf("ListRelatedEpicKeys() error = %v", err)
		}

		if keys == nil {
			t.Error("expected non-nil slice, got nil")
		}
		if len(keys) != 0 {
			t.Errorf("expected 0 keys, got %d: %v", len(keys), keys)
		}
	})
}
