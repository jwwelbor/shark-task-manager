package discovery

import (
	"fmt"
)

// ConflictResolver applies resolution strategies to conflicts
type ConflictResolver struct{}

// NewConflictResolver creates a new ConflictResolver instance
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// Resolve applies the specified strategy to resolve conflicts and returns final epic/feature lists
// Returns resolved epics, resolved features, warnings, and error
func (r *ConflictResolver) Resolve(
	indexEpics []DiscoveredEpic,
	folderEpics []DiscoveredEpic,
	indexFeatures []DiscoveredFeature,
	folderFeatures []DiscoveredFeature,
	conflicts []Conflict,
	strategy ConflictStrategy,
	indexFileExists bool,
) ([]DiscoveredEpic, []DiscoveredFeature, []string, error) {

	switch strategy {
	case ConflictStrategyIndexPrecedence:
		return r.resolveIndexPrecedence(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts, indexFileExists)

	case ConflictStrategyFolderPrecedence:
		return r.resolveFolderPrecedence(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts, indexFileExists)

	case ConflictStrategyMerge:
		return r.resolveMerge(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts, indexFileExists)

	default:
		return nil, nil, nil, fmt.Errorf("unknown conflict resolution strategy: %s", strategy)
	}
}

// resolveIndexPrecedence uses epic-index.md as source of truth
// Fails if index references items without folders
// Warns and skips folder-only items
func (r *ConflictResolver) resolveIndexPrecedence(
	indexEpics []DiscoveredEpic,
	folderEpics []DiscoveredEpic,
	indexFeatures []DiscoveredFeature,
	folderFeatures []DiscoveredFeature,
	conflicts []Conflict,
	indexFileExists bool,
) ([]DiscoveredEpic, []DiscoveredFeature, []string, error) {

	warnings := []string{}

	// Check for index-only conflicts (these are errors for index-precedence)
	for _, conflict := range conflicts {
		if conflict.Type == ConflictTypeEpicIndexOnly {
			return nil, nil, warnings, fmt.Errorf("epic %s in index but folder missing (index-precedence requires folders)", conflict.Key)
		}
		if conflict.Type == ConflictTypeFeatureIndexOnly {
			return nil, nil, warnings, fmt.Errorf("feature %s in index but folder missing (index-precedence requires folders)", conflict.Key)
		}
	}

	// Warn about folder-only items (will be skipped) - only if index file exists
	if indexFileExists {
		for _, conflict := range conflicts {
			if conflict.Type == ConflictTypeEpicFolderOnly {
				warnings = append(warnings, fmt.Sprintf("Epic %s in folders but not in index (skipped)", conflict.Key))
			}
			if conflict.Type == ConflictTypeFeatureFolderOnly {
				warnings = append(warnings, fmt.Sprintf("Feature %s in folders but not in index (skipped)", conflict.Key))
			}
		}
	}

	// Only include epics from index (they must all have matching folders)
	resultEpics := make([]DiscoveredEpic, len(indexEpics))
	copy(resultEpics, indexEpics)

	// Only include features from index
	resultFeatures := make([]DiscoveredFeature, len(indexFeatures))
	copy(resultFeatures, indexFeatures)

	return resultEpics, resultFeatures, warnings, nil
}

// resolveFolderPrecedence uses folder structure as source of truth
// Warns and skips index-only items
func (r *ConflictResolver) resolveFolderPrecedence(
	indexEpics []DiscoveredEpic,
	folderEpics []DiscoveredEpic,
	indexFeatures []DiscoveredFeature,
	folderFeatures []DiscoveredFeature,
	conflicts []Conflict,
	indexFileExists bool,
) ([]DiscoveredEpic, []DiscoveredFeature, []string, error) {

	warnings := []string{}

	// Warn about index-only items (will be skipped) - only if index file exists
	if indexFileExists {
		for _, conflict := range conflicts {
			if conflict.Type == ConflictTypeEpicIndexOnly {
				warnings = append(warnings, fmt.Sprintf("Epic %s in index but folder missing (skipped)", conflict.Key))
			}
			if conflict.Type == ConflictTypeFeatureIndexOnly {
				warnings = append(warnings, fmt.Sprintf("Feature %s in index but folder missing (skipped)", conflict.Key))
			}
		}
	}

	// Only include epics from folders
	resultEpics := make([]DiscoveredEpic, len(folderEpics))
	copy(resultEpics, folderEpics)

	// Only include features from folders
	resultFeatures := make([]DiscoveredFeature, len(folderFeatures))
	copy(resultFeatures, folderFeatures)

	return resultEpics, resultFeatures, warnings, nil
}

// resolveMerge combines both sources, with index metadata taking precedence
// Includes items from both index and folders
func (r *ConflictResolver) resolveMerge(
	indexEpics []DiscoveredEpic,
	folderEpics []DiscoveredEpic,
	indexFeatures []DiscoveredFeature,
	folderFeatures []DiscoveredFeature,
	conflicts []Conflict,
	indexFileExists bool,
) ([]DiscoveredEpic, []DiscoveredFeature, []string, error) {

	warnings := []string{}

	// Build maps for efficient lookup
	indexEpicMap := r.buildEpicMap(indexEpics)
	folderEpicMap := r.buildEpicMap(folderEpics)
	indexFeatureMap := r.buildFeatureMap(indexFeatures)
	folderFeatureMap := r.buildFeatureMap(folderFeatures)

	// Merge epics
	resultEpicMap := make(map[string]DiscoveredEpic)

	// Start with folder epics
	for key, folderEpic := range folderEpicMap {
		resultEpicMap[key] = folderEpic
	}

	// Overlay with index epics (index metadata wins)
	for key, indexEpic := range indexEpicMap {
		if folderEpic, existsInFolder := folderEpicMap[key]; existsInFolder {
			// Both sources have this epic - merge with index metadata winning
			merged := DiscoveredEpic{
				Key:         key,
				Title:       indexEpic.Title,       // Index wins
				Description: indexEpic.Description, // Index wins
				FilePath:    folderEpic.FilePath,   // Keep folder file path
				Source:      SourceMerged,
				Features:    []DiscoveredFeature{}, // Features handled separately
			}

			// If index has file path, use it
			if indexEpic.FilePath != nil {
				merged.FilePath = indexEpic.FilePath
			}

			resultEpicMap[key] = merged
		} else {
			// Index-only epic (folder missing) - only warn if index file exists
			if indexFileExists {
				warnings = append(warnings, fmt.Sprintf("Epic %s in index but folder missing (included anyway)", key))
			}
			indexEpicCopy := indexEpic
			indexEpicCopy.Source = SourceIndex
			resultEpicMap[key] = indexEpicCopy
		}
	}

	// Warn about folder-only epics (already included from first pass) - only if index file exists
	if indexFileExists {
		for key := range folderEpicMap {
			if _, existsInIndex := indexEpicMap[key]; !existsInIndex {
				warnings = append(warnings, fmt.Sprintf("Epic %s in folders but not in index (included anyway)", key))
			}
		}
	}

	// Merge features
	resultFeatureMap := make(map[string]DiscoveredFeature)

	// Start with folder features
	for key, folderFeature := range folderFeatureMap {
		resultFeatureMap[key] = folderFeature
	}

	// Overlay with index features (index metadata wins)
	for key, indexFeature := range indexFeatureMap {
		if folderFeature, existsInFolder := folderFeatureMap[key]; existsInFolder {
			// Both sources have this feature - merge with index metadata winning
			merged := DiscoveredFeature{
				Key:         key,
				EpicKey:     indexFeature.EpicKey,      // Index wins
				Title:       indexFeature.Title,        // Index wins
				Description: indexFeature.Description,  // Index wins
				FilePath:    folderFeature.FilePath,    // Keep folder file path
				RelatedDocs: folderFeature.RelatedDocs, // Keep folder related docs
				Source:      SourceMerged,
			}

			// If index has file path, use it
			if indexFeature.FilePath != nil {
				merged.FilePath = indexFeature.FilePath
			}

			// Warn if parent epic differs
			if indexFeature.EpicKey != folderFeature.EpicKey {
				warnings = append(warnings,
					fmt.Sprintf("Feature %s has parent epic %s in index but %s in folder (using index parent)",
						key, indexFeature.EpicKey, folderFeature.EpicKey))
			}

			resultFeatureMap[key] = merged
		} else {
			// Index-only feature (folder missing) - only warn if index file exists
			if indexFileExists {
				warnings = append(warnings, fmt.Sprintf("Feature %s in index but folder missing (included anyway)", key))
			}
			indexFeatureCopy := indexFeature
			indexFeatureCopy.Source = SourceIndex
			resultFeatureMap[key] = indexFeatureCopy
		}
	}

	// Warn about folder-only features (already included from first pass) - only if index file exists
	if indexFileExists {
		for key := range folderFeatureMap {
			if _, existsInIndex := indexFeatureMap[key]; !existsInIndex {
				warnings = append(warnings, fmt.Sprintf("Feature %s in folders but not in index (included anyway)", key))
			}
		}
	}

	// Convert maps to slices
	resultEpics := make([]DiscoveredEpic, 0, len(resultEpicMap))
	for _, epic := range resultEpicMap {
		resultEpics = append(resultEpics, epic)
	}

	resultFeatures := make([]DiscoveredFeature, 0, len(resultFeatureMap))
	for _, feature := range resultFeatureMap {
		resultFeatures = append(resultFeatures, feature)
	}

	return resultEpics, resultFeatures, warnings, nil
}

// buildEpicMap creates a map of epic keys to DiscoveredEpic
func (r *ConflictResolver) buildEpicMap(epics []DiscoveredEpic) map[string]DiscoveredEpic {
	epicMap := make(map[string]DiscoveredEpic)
	for _, epic := range epics {
		epicMap[epic.Key] = epic
	}
	return epicMap
}

// buildFeatureMap creates a map of feature keys to DiscoveredFeature
func (r *ConflictResolver) buildFeatureMap(features []DiscoveredFeature) map[string]DiscoveredFeature {
	featureMap := make(map[string]DiscoveredFeature)
	for _, feature := range features {
		featureMap[feature.Key] = feature
	}
	return featureMap
}
