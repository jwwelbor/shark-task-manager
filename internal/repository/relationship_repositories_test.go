package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureRelationshipRepository_Create tests creating a feature relationship
// Scenario 11: Feature with Related Features (Database Table)
func TestFeatureRelationshipRepository_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Clean up any existing relationships from previous tests
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")

	// Seed test data
	epicID, _ := test.SeedTestData()

	// Create additional test features
	desc1 := "Test feature for relationships"
	feature1 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F97",
		Title: "Test Feature 1",

		Description: &desc1}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create test feature 1: %v", err)
	}

	desc2 := "Related test feature"
	feature2 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F98",
		Title: "Test Feature 2",

		Description: &desc2}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create test feature 2: %v", err)
	}

	// Create a dependency relationship between features
	rel := &models.FeatureRelationship{
		FromFeatureID:    feature1.ID,
		ToFeatureID:      feature2.ID,
		RelationshipType: models.RelationshipDependsOn,
	}

	err := relRepo.Create(ctx, rel)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	if rel.ID == 0 {
		t.Error("Expected relationship ID to be set after creation")
	}

	// Verify relationship was created in database
	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM feature_relationships WHERE id = ?", rel.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query feature_relationships: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 relationship in database, got %d", count)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E98-F29', 'E98-F30')")
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE id = ?", rel.ID)
}

// TestFeatureRelationshipRepository_ListRelatedFeatures tests listing related features
// Scenario 11: Feature with Related Features - verify all relationships are returned
func TestFeatureRelationshipRepository_ListRelatedFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Clean up any existing test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-F9%'")

	// Seed test data
	epicID, _ := test.SeedTestData()

	// Create test features
	features := make([]*models.Feature, 3)
	for i := 0; i < 3; i++ {
		desc := "Test feature for relationships"
		fnum := 91 + i // F91, F92, F93
		fkey := "E99-F" + string(rune(48+(fnum/10))) + string(rune(48+(fnum%10)))
		features[i] = &models.Feature{BaseEntity: models.BaseEntity{Key: fkey,
			Title: "Test Feature " + string(rune(49+i)),

			Description: &desc}, EpicID: epicID,

			Status: "todo",
		}
		if err := featureRepo.Create(ctx, features[i]); err != nil {
			t.Fatalf("Failed to create test feature: %v", err)
		}
	}

	// Create relationships: Feature 0 depends on Feature 1, Feature 0 related to Feature 2
	rel1 := &models.FeatureRelationship{
		FromFeatureID:    features[0].ID,
		ToFeatureID:      features[1].ID,
		RelationshipType: models.RelationshipDependsOn,
	}
	if err := relRepo.Create(ctx, rel1); err != nil {
		t.Fatalf("Failed to create relationship 1: %v", err)
	}

	rel2 := &models.FeatureRelationship{
		FromFeatureID:    features[0].ID,
		ToFeatureID:      features[2].ID,
		RelationshipType: models.RelationshipRelatedTo,
	}
	if err := relRepo.Create(ctx, rel2); err != nil {
		t.Fatalf("Failed to create relationship 2: %v", err)
	}

	// List relationships for Feature 0
	relationships, err := relRepo.ListRelatedFeatures(ctx, features[0].ID)
	if err != nil {
		t.Fatalf("Failed to list related features: %v", err)
	}

	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(relationships))
	}

	// Verify relationship types
	relationshipTypes := make(map[models.RelationshipType]bool)
	for _, rel := range relationships {
		relationshipTypes[rel.RelationshipType] = true
	}

	if !relationshipTypes[models.RelationshipDependsOn] {
		t.Error("Expected 'depends_on' relationship not found")
	}
	if !relationshipTypes[models.RelationshipRelatedTo] {
		t.Error("Expected 'related_to' relationship not found")
	}

	// Cleanup
	for _, feature := range features {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE id IN (?, ?)", rel1.ID, rel2.ID)
}

// TestFeatureRelationshipRepository_GetRelatedFeatureKeys tests retrieving related feature keys
// Scenario 11: Feature with Related Features - verifies CSV format output
func TestFeatureRelationshipRepository_GetRelatedFeatureKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Clean up any existing test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-F9%'")

	// Seed test data
	epicID, _ := test.SeedTestData()

	// Create test features with specific keys for CSV format testing
	feature1 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F95",
		Title: "Auth Feature"}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	feature2 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F96",
		Title: "API Feature"}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	// Create relationships - bidirectional
	rel1 := &models.FeatureRelationship{
		FromFeatureID:    feature1.ID,
		ToFeatureID:      feature2.ID,
		RelationshipType: models.RelationshipDependsOn,
	}
	if err := relRepo.Create(ctx, rel1); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Get related feature keys
	keys, err := relRepo.GetRelatedFeatureKeys(ctx, feature1.ID)
	if err != nil {
		t.Fatalf("Failed to get related feature keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 related feature key, got %d", len(keys))
	}

	if len(keys) > 0 && keys[0] != "E99-F96" {
		t.Errorf("Expected key 'E99-F96', got '%s'", keys[0])
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id IN (?, ?)", feature1.ID, feature2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE id = ?", rel1.ID)
}

// TestFeatureRelationshipRepository_NoRelatedFeatures tests when feature has no relationships
// Scenario 12: Feature with No Related Features
func TestFeatureRelationshipRepository_NoRelatedFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Seed test data
	epicID, _ := test.SeedTestData()

	// Create test feature with no relationships
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F94",
		Title: "Isolated Feature"}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// List relationships - should be empty
	relationships, err := relRepo.ListRelatedFeatures(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to list related features: %v", err)
	}

	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships for isolated feature, got %d", len(relationships))
	}

	// Get related feature keys - should be empty
	keys, err := relRepo.GetRelatedFeatureKeys(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get related feature keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 related feature keys, got %d", len(keys))
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
}

// TestEpicRelationshipRepository_Create tests creating an epic relationship
// Scenario 14: Epic with Related Epics (Database Table)
func TestEpicRelationshipRepository_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	relRepo := NewEpicRelationshipRepository(db)

	// Clean up any existing relationships from previous tests
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE from_epic_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E97')")

	// Create test epics
	desc1 := "Test epic for relationships"
	epic1 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E98",
		Title: "Test Epic 1",

		Description: &desc1}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create epic1: %v", err)
	}

	desc2 := "Related test epic"
	epic2 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E97",
		Title: "Test Epic 2",

		Description: &desc2}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic2); err != nil {
		t.Fatalf("Failed to create epic2: %v", err)
	}

	// Create a dependency relationship between epics
	rel := &models.EpicRelationship{
		FromEpicID:       epic1.ID,
		ToEpicID:         epic2.ID,
		RelationshipType: models.RelationshipDependsOn,
	}

	err := relRepo.Create(ctx, rel)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	if rel.ID == 0 {
		t.Error("Expected relationship ID to be set after creation")
	}

	// Verify relationship was created in database
	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM epic_relationships WHERE id = ?", rel.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query epic_relationships: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 relationship in database, got %d", count)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E97')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE id = ?", rel.ID)
}

// TestEpicRelationshipRepository_ListRelatedEpics tests listing related epics
// Scenario 14: Epic with Related Epics - verify all relationships are returned
func TestEpicRelationshipRepository_ListRelatedEpics(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	relRepo := NewEpicRelationshipRepository(db)

	// Clean up any existing test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE from_epic_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E97', 'E96', 'E95', 'E94', 'E93', 'E92', 'E91', 'E90')")

	// Create test epics
	epics := make([]*models.Epic, 3)
	for i := 0; i < 3; i++ {
		desc := "Test epic for relationships"
		epics[i] = &models.Epic{BaseEntity: models.BaseEntity{Key: "E9" + string(rune(48+i)), // E90, E91, E92
			Title: "Test Epic " + string(rune(49+i)),

			Description: &desc}, Status: "todo",
			Priority: models.PriorityMedium,
		}
		if err := epicRepo.Create(ctx, epics[i]); err != nil {
			t.Fatalf("Failed to create test epic: %v", err)
		}
	}

	// Create relationships: Epic 0 depends on Epic 1, Epic 0 related to Epic 2
	rel1 := &models.EpicRelationship{
		FromEpicID:       epics[0].ID,
		ToEpicID:         epics[1].ID,
		RelationshipType: models.RelationshipDependsOn,
	}
	if err := relRepo.Create(ctx, rel1); err != nil {
		t.Fatalf("Failed to create relationship 1: %v", err)
	}

	rel2 := &models.EpicRelationship{
		FromEpicID:       epics[0].ID,
		ToEpicID:         epics[2].ID,
		RelationshipType: models.RelationshipRelatedTo,
	}
	if err := relRepo.Create(ctx, rel2); err != nil {
		t.Fatalf("Failed to create relationship 2: %v", err)
	}

	// List relationships for Epic 0
	relationships, err := relRepo.ListRelatedEpics(ctx, epics[0].ID)
	if err != nil {
		t.Fatalf("Failed to list related epics: %v", err)
	}

	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(relationships))
	}

	// Verify relationship types
	relationshipTypes := make(map[models.RelationshipType]bool)
	for _, rel := range relationships {
		relationshipTypes[rel.RelationshipType] = true
	}

	if !relationshipTypes[models.RelationshipDependsOn] {
		t.Error("Expected 'depends_on' relationship not found")
	}
	if !relationshipTypes[models.RelationshipRelatedTo] {
		t.Error("Expected 'related_to' relationship not found")
	}

	// Cleanup
	for _, epic := range epics {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE id IN (?, ?)", rel1.ID, rel2.ID)
}

// TestEpicRelationshipRepository_GetRelatedEpicKeys tests retrieving related epic keys
// Scenario 14: Epic with Related Epics - verifies CSV format output
func TestEpicRelationshipRepository_GetRelatedEpicKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	relRepo := NewEpicRelationshipRepository(db)

	// Clean up any existing test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE from_epic_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E97', 'E96', 'E95', 'E94', 'E93', 'E92', 'E91', 'E90')")

	// Create test epics with specific keys for CSV format testing
	epic1 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E96",
		Title: "Auth Epic"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create epic1: %v", err)
	}

	epic2 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E95",
		Title: "API Epic"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic2); err != nil {
		t.Fatalf("Failed to create epic2: %v", err)
	}

	// Create relationship
	rel := &models.EpicRelationship{
		FromEpicID:       epic1.ID,
		ToEpicID:         epic2.ID,
		RelationshipType: models.RelationshipDependsOn,
	}
	if err := relRepo.Create(ctx, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Get related epic keys
	keys, err := relRepo.GetRelatedEpicKeys(ctx, epic1.ID)
	if err != nil {
		t.Fatalf("Failed to get related epic keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 related epic key, got %d", len(keys))
	}

	if len(keys) > 0 && keys[0] != "E95" {
		t.Errorf("Expected key 'E95', got '%s'", keys[0])
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id IN (?, ?)", epic1.ID, epic2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE id = ?", rel.ID)
}

// TestEpicRelationshipRepository_NoRelatedEpics tests when epic has no relationships
// Scenario 15: Epic with No Related Epics
func TestEpicRelationshipRepository_NoRelatedEpics(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	relRepo := NewEpicRelationshipRepository(db)

	// Create test epic with no relationships
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E97",
		Title: "Isolated Epic"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// List relationships - should be empty
	relationships, err := relRepo.ListRelatedEpics(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list related epics: %v", err)
	}

	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships for isolated epic, got %d", len(relationships))
	}

	// Get related epic keys - should be empty
	keys, err := relRepo.GetRelatedEpicKeys(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get related epic keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 related epic keys, got %d", len(keys))
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureRelationshipRepository_CrossEpicRelationships tests cross-epic feature relationships
// Scenario 11: Feature with Related Features (Cross-Epic Support)
func TestFeatureRelationshipRepository_CrossEpicRelationships(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")

	// Create two epics
	epic1 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E98",
		Title: "Epic 1"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create epic1: %v", err)
	}

	epic2 := &models.Epic{BaseEntity: models.BaseEntity{Key: "E94",
		Title: "Epic 2"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic2); err != nil {
		t.Fatalf("Failed to create epic2: %v", err)
	}

	// Create features in different epics
	feature1 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F95",
		Title: "Feature in Epic 1"}, EpicID: epic1.ID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	feature2 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E94-F05",
		Title: "Feature in Epic 2"}, EpicID: epic2.ID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	// Create cross-epic relationship
	rel := &models.FeatureRelationship{
		FromFeatureID:    feature1.ID,
		ToFeatureID:      feature2.ID,
		RelationshipType: models.RelationshipDependsOn,
	}
	if err := relRepo.Create(ctx, rel); err != nil {
		t.Fatalf("Failed to create cross-epic relationship: %v", err)
	}

	// Get related feature keys - should include cross-epic feature
	keys, err := relRepo.GetRelatedFeatureKeys(ctx, feature1.ID)
	if err != nil {
		t.Fatalf("Failed to get related feature keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 related feature key, got %d", len(keys))
	}

	if len(keys) > 0 && keys[0] != "E94-F05" {
		t.Errorf("Expected cross-epic key 'E94-F05', got '%s'", keys[0])
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id IN (?, ?)", feature1.ID, feature2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id IN (?, ?)", epic1.ID, epic2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE id = ?", rel.ID)
}

// TestRelationshipValidation tests validation of relationship fields
// Ensures both feature and epic relationships validate properly
func TestRelationshipValidation(t *testing.T) {
	tests := []struct {
		name        string
		feature     *models.FeatureRelationship
		epic        *models.EpicRelationship
		expectError bool
		errorType   string
	}{
		{
			name: "feature self-relationship",
			feature: &models.FeatureRelationship{
				FromFeatureID:    1,
				ToFeatureID:      1,
				RelationshipType: models.RelationshipDependsOn,
			},
			expectError: true,
			errorType:   "self",
		},
		{
			name: "epic self-relationship",
			epic: &models.EpicRelationship{
				FromEpicID:       1,
				ToEpicID:         1,
				RelationshipType: models.RelationshipDependsOn,
			},
			expectError: true,
			errorType:   "self",
		},
		{
			name: "feature zero from_id",
			feature: &models.FeatureRelationship{
				FromFeatureID:    0,
				ToFeatureID:      1,
				RelationshipType: models.RelationshipDependsOn,
			},
			expectError: true,
			errorType:   "invalid",
		},
		{
			name: "epic zero to_id",
			epic: &models.EpicRelationship{
				FromEpicID:       1,
				ToEpicID:         0,
				RelationshipType: models.RelationshipDependsOn,
			},
			expectError: true,
			errorType:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.feature != nil {
				err := tt.feature.Validate()
				if (err != nil) != tt.expectError {
					t.Errorf("FeatureRelationship.Validate() error = %v, expectError %v", err, tt.expectError)
				}
			}

			if tt.epic != nil {
				err := tt.epic.Validate()
				if (err != nil) != tt.expectError {
					t.Errorf("EpicRelationship.Validate() error = %v, expectError %v", err, tt.expectError)
				}
			}
		})
	}
}

// BenchmarkListRelatedFeatures benchmarks the feature relationship list operation
func BenchmarkListRelatedFeatures(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	relRepo := NewFeatureRelationshipRepository(db)

	// Clean up any existing test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-F9%'")

	// Seed test data
	epicID, _ := test.SeedTestData()

	// Create test feature
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "BENCH-F01",
		Title: "Benchmark Feature"}, EpicID: epicID,

		Status: "todo",
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		b.Fatalf("Failed to create feature: %v", err)
	}

	// Create 10 related features
	for i := 0; i < 10; i++ {
		relFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "BENCH-REL-" + string(rune(49+i)),
			Title: "Related Feature"}, EpicID: epicID,

			Status: "todo",
		}
		if err := featureRepo.Create(ctx, relFeature); err != nil {
			b.Fatalf("Failed to create related feature: %v", err)
		}

		rel := &models.FeatureRelationship{
			FromFeatureID:    feature.ID,
			ToFeatureID:      relFeature.ID,
			RelationshipType: models.RelationshipDependsOn,
		}
		if err := relRepo.Create(ctx, rel); err != nil {
			b.Fatalf("Failed to create relationship: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = relRepo.ListRelatedFeatures(ctx, feature.ID)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'BENCH-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM feature_relationships WHERE from_feature_id > 0")
}

// BenchmarkListRelatedEpics benchmarks the epic relationship list operation
func BenchmarkListRelatedEpics(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	relRepo := NewEpicRelationshipRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE from_epic_id > 0")

	// Create test epic
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "BENCH-E01",
		Title: "Benchmark Epic"}, Status: "todo",
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		b.Fatalf("Failed to create epic: %v", err)
	}

	// Create 10 related epics
	for i := 0; i < 10; i++ {
		relEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "BENCH-REL-" + string(rune(49+i)),
			Title: "Related Epic"}, Status: "todo",
			Priority: models.PriorityMedium,
		}
		if err := epicRepo.Create(ctx, relEpic); err != nil {
			b.Fatalf("Failed to create related epic: %v", err)
		}

		rel := &models.EpicRelationship{
			FromEpicID:       epic.ID,
			ToEpicID:         relEpic.ID,
			RelationshipType: models.RelationshipDependsOn,
		}
		if err := relRepo.Create(ctx, rel); err != nil {
			b.Fatalf("Failed to create relationship: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = relRepo.ListRelatedEpics(ctx, epic.ID)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'BENCH-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epic_relationships WHERE from_epic_id > 0")
}
