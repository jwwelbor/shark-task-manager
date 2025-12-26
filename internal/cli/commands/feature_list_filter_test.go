package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func TestFeatureList_HidesCompletedFeaturesByDefault(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create active feature
	activeFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F01",
		Title:       "Active Feature",
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	if err := featureRepo.Create(ctx, activeFeature); err != nil {
		t.Fatalf("Failed to create active feature: %v", err)
	}

	// Create completed feature
	completedFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F02",
		Title:       "Completed Feature",
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	if err := featureRepo.Create(ctx, completedFeature); err != nil {
		t.Fatalf("Failed to create completed feature: %v", err)
	}

	// Get all features (simulating list without filter)
	allFeatures, err := featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	// Convert to FeatureWithTaskCount for filtering
	featuresWithTaskCount := make([]FeatureWithTaskCount, 0, len(allFeatures))
	for _, feature := range allFeatures {
		featuresWithTaskCount = append(featuresWithTaskCount, FeatureWithTaskCount{
			Feature:   feature,
			TaskCount: 0,
		})
	}

	// Filter out completed features (default behavior)
	showAll := false
	filteredFeatures := filterFeaturesByCompletedStatus(featuresWithTaskCount, showAll, "")

	// Should only show active feature
	if len(filteredFeatures) != 1 {
		t.Errorf("Expected 1 feature after filtering, got %d", len(filteredFeatures))
	}

	if len(filteredFeatures) > 0 && filteredFeatures[0].Status == models.FeatureStatusCompleted {
		t.Error("Completed feature should be filtered out")
	}
}

func TestFeatureList_ShowsAllFeaturesWithShowAllFlag(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create active feature
	activeFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F01",
		Title:       "Active Feature",
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	if err := featureRepo.Create(ctx, activeFeature); err != nil {
		t.Fatalf("Failed to create active feature: %v", err)
	}

	// Create completed feature
	completedFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F02",
		Title:       "Completed Feature",
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	if err := featureRepo.Create(ctx, completedFeature); err != nil {
		t.Fatalf("Failed to create completed feature: %v", err)
	}

	// Get all features
	allFeatures, err := featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	// Convert to FeatureWithTaskCount for filtering
	featuresWithTaskCount := make([]FeatureWithTaskCount, 0, len(allFeatures))
	for _, feature := range allFeatures {
		featuresWithTaskCount = append(featuresWithTaskCount, FeatureWithTaskCount{
			Feature:   feature,
			TaskCount: 0,
		})
	}

	// With showAll=true, should show all features
	showAll := true
	filteredFeatures := filterFeaturesByCompletedStatus(featuresWithTaskCount, showAll, "")

	// Should show both features
	if len(filteredFeatures) != 2 {
		t.Errorf("Expected 2 features with --show-all, got %d", len(filteredFeatures))
	}
}

func TestFeatureList_ShowsCompletedFeaturesWithExplicitStatusFilter(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create completed feature
	completedFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F02",
		Title:       "Completed Feature",
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	if err := featureRepo.Create(ctx, completedFeature); err != nil {
		t.Fatalf("Failed to create completed feature: %v", err)
	}

	// Get all features
	allFeatures, err := featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	// Convert to FeatureWithTaskCount for filtering
	featuresWithTaskCount := make([]FeatureWithTaskCount, 0, len(allFeatures))
	for _, feature := range allFeatures {
		featuresWithTaskCount = append(featuresWithTaskCount, FeatureWithTaskCount{
			Feature:   feature,
			TaskCount: 0,
		})
	}

	// With explicit status filter, should not apply default filtering
	statusFilter := "completed"
	filteredFeatures := filterFeaturesByCompletedStatus(featuresWithTaskCount, false, statusFilter)

	// Should show completed feature because explicit filter is set
	if len(filteredFeatures) != 1 {
		t.Errorf("Expected 1 feature with explicit status filter, got %d", len(filteredFeatures))
	}
}
