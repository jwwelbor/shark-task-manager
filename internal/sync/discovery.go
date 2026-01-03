package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/discovery"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// runDiscovery orchestrates the discovery workflow
func (e *SyncEngine) runDiscovery(ctx context.Context, opts SyncOptions) (*DiscoveryReport, error) {
	report := &DiscoveryReport{
		Warnings: []string{},
	}

	// Build discovery options
	discoveryOpts := discovery.DiscoveryOptions{
		DocsRoot:        e.docsRoot,
		IndexPath:       filepath.Join(e.docsRoot, opts.FolderPath, "epic-index.md"),
		Strategy:        mapDiscoveryStrategy(opts.DiscoveryStrategy),
		DryRun:          opts.DryRun,
		ValidationLevel: mapValidationLevel(opts.ValidationLevel),
	}

	// Step 1: Parse epic-index.md if it exists
	var indexEpics []discovery.IndexEpic
	var indexFeatures []discovery.IndexFeature
	var parseErr error
	indexFileExists := false

	if _, err := os.Stat(discoveryOpts.IndexPath); err == nil {
		indexFileExists = true
		parser := discovery.NewIndexParser()
		indexEpics, indexFeatures, parseErr = parser.Parse(discoveryOpts.IndexPath)
		if parseErr != nil {
			// If index parsing fails and strategy is index-only, fail
			if opts.DiscoveryStrategy == DiscoveryStrategyIndexOnly {
				return nil, fmt.Errorf("failed to parse epic-index.md (required for index-only strategy): %w", parseErr)
			}
			// Otherwise warn and continue
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to parse epic-index.md: %v", parseErr))
		}
	} else {
		// Index file doesn't exist
		if opts.DiscoveryStrategy == DiscoveryStrategyIndexOnly {
			return nil, fmt.Errorf("epic-index.md not found at %s (required for index-only strategy)", discoveryOpts.IndexPath)
		}
		report.Warnings = append(report.Warnings, fmt.Sprintf("epic-index.md not found at %s", discoveryOpts.IndexPath))
	}

	// Step 2: Scan folder structure
	scanner := discovery.NewFolderScanner()
	folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to scan folders: %w", err)
	}

	// Step 3: Convert to DiscoveredEpic/DiscoveredFeature
	discoveredIndexEpics := convertIndexEpics(indexEpics)
	discoveredFolderEpics := convertFolderEpics(folderEpics)
	discoveredIndexFeatures := convertIndexFeatures(indexFeatures)
	discoveredFolderFeatures := convertFolderFeatures(folderFeatures)

	// Step 4: Detect conflicts
	detector := discovery.NewConflictDetector()
	conflicts := detector.Detect(discoveredIndexEpics, discoveredFolderEpics, discoveredIndexFeatures, discoveredFolderFeatures)
	report.ConflictsDetected = len(conflicts)

	// Step 5: Resolve conflicts
	resolver := discovery.NewConflictResolver()
	resolvedEpics, resolvedFeatures, warnings, err := resolver.Resolve(
		discoveredIndexEpics,
		discoveredFolderEpics,
		discoveredIndexFeatures,
		discoveredFolderFeatures,
		conflicts,
		mapDiscoveryStrategy(opts.DiscoveryStrategy),
		indexFileExists,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve conflicts: %w", err)
	}
	report.Warnings = append(report.Warnings, warnings...)
	report.ConflictsResolved = len(conflicts)

	// Update counts
	report.EpicsDiscovered = len(resolvedEpics)
	report.FeaturesDiscovered = len(resolvedFeatures)

	// Step 6: Import into database
	if !opts.DryRun {
		epicsImported, featuresImported, warnings, err := e.importDiscoveredEntities(ctx, resolvedEpics, resolvedFeatures)
		if err != nil {
			return nil, fmt.Errorf("failed to import discovered entities: %w", err)
		}
		report.EpicsImported = epicsImported
		report.FeaturesImported = featuresImported
		report.Warnings = append(report.Warnings, warnings...)
	}

	return report, nil
}

// importDiscoveredEntities imports discovered epics and features into database
// Returns epics imported, features imported, warnings, and error
func (e *SyncEngine) importDiscoveredEntities(ctx context.Context, epics []discovery.DiscoveredEpic, features []discovery.DiscoveredFeature) (int, int, []string, error) {
	// Begin transaction
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	epicsImported := 0
	featuresImported := 0
	warnings := []string{}

	// Import epics
	for _, epic := range epics {
		// Validate epic key format - skip if invalid
		if err := models.ValidateEpicKey(epic.Key); err != nil {
			warnings = append(warnings, fmt.Sprintf("Skipping epic with invalid key format: %s", epic.Key))
			continue
		}

		// Check if epic already exists
		existingEpic, err := e.epicRepo.GetByKey(ctx, epic.Key)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, nil, fmt.Errorf("failed to check epic %s: %w", epic.Key, err)
		}

		if existingEpic != nil {
			// Epic exists - update if needed
			needsUpdate := false
			if epic.Title != "" && existingEpic.Title != epic.Title {
				existingEpic.Title = epic.Title
				needsUpdate = true
			}
			if epic.Description != nil && existingEpic.Description != epic.Description {
				existingEpic.Description = epic.Description
				needsUpdate = true
			}
			if needsUpdate {
				if err := e.epicRepo.Update(ctx, existingEpic); err != nil {
					return 0, 0, nil, fmt.Errorf("failed to update epic %s: %w", epic.Key, err)
				}
			}
		} else {
			// Create new epic
			newEpic := &models.Epic{
				Key:      epic.Key,
				Title:    epic.Title,
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			}
			if epic.Description != nil {
				newEpic.Description = epic.Description
			}
			if err := e.epicRepo.Create(ctx, newEpic); err != nil {
				return 0, 0, nil, fmt.Errorf("failed to create epic %s: %w", epic.Key, err)
			}
			epicsImported++
		}
	}

	// Import features
	for _, feature := range features {
		// Validate feature key format - skip if invalid
		if err := models.ValidateFeatureKey(feature.Key); err != nil {
			warnings = append(warnings, fmt.Sprintf("Skipping feature with invalid key format: %s", feature.Key))
			continue
		}

		// Get parent epic
		epic, err := e.epicRepo.GetByKey(ctx, feature.EpicKey)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("failed to get epic %s for feature %s: %w", feature.EpicKey, feature.Key, err)
		}

		// Check if feature already exists
		existingFeature, err := e.featureRepo.GetByKey(ctx, feature.Key)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, nil, fmt.Errorf("failed to check feature %s: %w", feature.Key, err)
		}

		if existingFeature != nil {
			// Feature exists - update if needed
			needsUpdate := false
			if feature.Title != "" && existingFeature.Title != feature.Title {
				existingFeature.Title = feature.Title
				needsUpdate = true
			}
			if feature.Description != nil && existingFeature.Description != feature.Description {
				existingFeature.Description = feature.Description
				needsUpdate = true
			}
			if needsUpdate {
				if err := e.featureRepo.Update(ctx, existingFeature); err != nil {
					return 0, 0, nil, fmt.Errorf("failed to update feature %s: %w", feature.Key, err)
				}
			}
		} else {
			// Create new feature
			newFeature := &models.Feature{
				EpicID: epic.ID,
				Key:    feature.Key,
				Title:  feature.Title,
				Status: models.FeatureStatusActive,
			}
			if feature.Description != nil {
				newFeature.Description = feature.Description
			}
			if err := e.featureRepo.Create(ctx, newFeature); err != nil {
				return 0, 0, nil, fmt.Errorf("failed to create feature %s: %w", feature.Key, err)
			}
			featuresImported++
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return epicsImported, featuresImported, warnings, nil
}

// mapDiscoveryStrategy maps sync.DiscoveryStrategy to discovery.ConflictStrategy
func mapDiscoveryStrategy(s DiscoveryStrategy) discovery.ConflictStrategy {
	switch s {
	case DiscoveryStrategyIndexOnly:
		return discovery.ConflictStrategyIndexPrecedence
	case DiscoveryStrategyFolderOnly:
		return discovery.ConflictStrategyFolderPrecedence
	case DiscoveryStrategyMerge:
		return discovery.ConflictStrategyMerge
	default:
		return discovery.ConflictStrategyMerge
	}
}

// mapValidationLevel maps sync.ValidationLevel to discovery.ValidationLevel
func mapValidationLevel(v ValidationLevel) discovery.ValidationLevel {
	switch v {
	case ValidationLevelStrict:
		return discovery.ValidationLevelStrict
	case ValidationLevelBalanced:
		return discovery.ValidationLevelBalanced
	case ValidationLevelPermissive:
		return discovery.ValidationLevelPermissive
	default:
		return discovery.ValidationLevelBalanced
	}
}

// convertIndexEpics converts IndexEpic to DiscoveredEpic
func convertIndexEpics(indexEpics []discovery.IndexEpic) []discovery.DiscoveredEpic {
	result := make([]discovery.DiscoveredEpic, len(indexEpics))
	for i, epic := range indexEpics {
		result[i] = discovery.DiscoveredEpic{
			Key:    epic.Key,
			Title:  epic.Title,
			Source: discovery.SourceIndex,
		}
	}
	return result
}

// convertFolderEpics converts FolderEpic to DiscoveredEpic
func convertFolderEpics(folderEpics []discovery.FolderEpic) []discovery.DiscoveredEpic {
	result := make([]discovery.DiscoveredEpic, len(folderEpics))
	for i, epic := range folderEpics {
		result[i] = discovery.DiscoveredEpic{
			Key:      epic.Key,
			Title:    epic.Slug,
			FilePath: epic.EpicMdPath,
			Source:   discovery.SourceFolder,
		}
	}
	return result
}

// convertIndexFeatures converts IndexFeature to DiscoveredFeature
func convertIndexFeatures(indexFeatures []discovery.IndexFeature) []discovery.DiscoveredFeature {
	result := make([]discovery.DiscoveredFeature, len(indexFeatures))
	for i, feature := range indexFeatures {
		result[i] = discovery.DiscoveredFeature{
			Key:     feature.Key,
			EpicKey: feature.EpicKey,
			Title:   feature.Title,
			Source:  discovery.SourceIndex,
		}
	}
	return result
}

// convertFolderFeatures converts FolderFeature to DiscoveredFeature
func convertFolderFeatures(folderFeatures []discovery.FolderFeature) []discovery.DiscoveredFeature {
	result := make([]discovery.DiscoveredFeature, len(folderFeatures))
	for i, feature := range folderFeatures {
		result[i] = discovery.DiscoveredFeature{
			Key:         feature.Key,
			EpicKey:     feature.EpicKey,
			Title:       feature.Slug,
			FilePath:    feature.PrdPath,
			RelatedDocs: feature.RelatedDocs,
			Source:      discovery.SourceFolder,
		}
	}
	return result
}
