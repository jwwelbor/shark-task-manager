package discovery

import (
	"fmt"
)

// ConflictDetector identifies conflicts between index and folder discoveries
type ConflictDetector struct{}

// NewConflictDetector creates a new ConflictDetector instance
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// Detect finds all conflicts between index and folder discoveries
// Returns a slice of conflicts with actionable suggestions
func (d *ConflictDetector) Detect(
	indexEpics []DiscoveredEpic,
	folderEpics []DiscoveredEpic,
	indexFeatures []DiscoveredFeature,
	folderFeatures []DiscoveredFeature,
) []Conflict {
	conflicts := []Conflict{}

	// Build key maps for efficient lookup
	indexEpicKeys := d.buildEpicKeyMap(indexEpics)
	folderEpicKeys := d.buildEpicKeyMap(folderEpics)
	indexFeatureKeys := d.buildFeatureKeyMap(indexFeatures)
	folderFeatureKeys := d.buildFeatureKeyMap(folderFeatures)

	// Detect epic conflicts
	// 1. Epics in index but not in folders (broken references)
	for key, epic := range indexEpicKeys {
		if _, existsInFolder := folderEpicKeys[key]; !existsInFolder {
			path := ""
			if epic.FilePath != nil {
				path = *epic.FilePath
			}

			conflicts = append(conflicts, Conflict{
				Type:       ConflictTypeEpicIndexOnly,
				Key:        key,
				Path:       path,
				Resolution: "",
				Strategy:   "",
				Suggestion: fmt.Sprintf("Create folder for epic %s or remove from epic-index.md", key),
			})
		}
	}

	// 2. Epics in folders but not in index (undocumented)
	for key, epic := range folderEpicKeys {
		if _, existsInIndex := indexEpicKeys[key]; !existsInIndex {
			path := ""
			if epic.FilePath != nil {
				path = *epic.FilePath
			}

			conflicts = append(conflicts, Conflict{
				Type:       ConflictTypeEpicFolderOnly,
				Key:        key,
				Path:       path,
				Resolution: "",
				Strategy:   "",
				Suggestion: fmt.Sprintf("Add epic %s to epic-index.md or use merge/folder-precedence strategy", key),
			})
		}
	}

	// Detect feature conflicts
	// 1. Features in index but not in folders (broken references)
	for key, feature := range indexFeatureKeys {
		if _, existsInFolder := folderFeatureKeys[key]; !existsInFolder {
			path := ""
			if feature.FilePath != nil {
				path = *feature.FilePath
			}

			conflicts = append(conflicts, Conflict{
				Type:       ConflictTypeFeatureIndexOnly,
				Key:        key,
				Path:       path,
				Resolution: "",
				Strategy:   "",
				Suggestion: fmt.Sprintf("Create folder for feature %s or remove from epic-index.md", key),
			})
		}
	}

	// 2. Features in folders but not in index (undocumented)
	for key, feature := range folderFeatureKeys {
		if _, existsInIndex := indexFeatureKeys[key]; !existsInIndex {
			path := ""
			if feature.FilePath != nil {
				path = *feature.FilePath
			}

			conflicts = append(conflicts, Conflict{
				Type:       ConflictTypeFeatureFolderOnly,
				Key:        key,
				Path:       path,
				Resolution: "",
				Strategy:   "",
				Suggestion: fmt.Sprintf("Add feature %s to epic-index.md or use merge/folder-precedence strategy", key),
			})
		}
	}

	// 3. Features with mismatched parent epics (relationship conflicts)
	for key, indexFeature := range indexFeatureKeys {
		if folderFeature, existsInFolder := folderFeatureKeys[key]; existsInFolder {
			// Feature exists in both, but check if parent epic matches
			if indexFeature.EpicKey != folderFeature.EpicKey {
				path := ""
				if folderFeature.FilePath != nil {
					path = *folderFeature.FilePath
				}

				conflicts = append(conflicts, Conflict{
					Type:       ConflictTypeRelationshipMismatch,
					Key:        key,
					Path:       path,
					Resolution: "",
					Strategy:   "",
					Suggestion: fmt.Sprintf("Feature %s has parent epic %s in index but %s in folder structure. Move folder or update epic-index.md", key, indexFeature.EpicKey, folderFeature.EpicKey),
				})
			}
		}
	}

	return conflicts
}

// buildEpicKeyMap creates a map of epic keys to DiscoveredEpic for fast lookup
func (d *ConflictDetector) buildEpicKeyMap(epics []DiscoveredEpic) map[string]DiscoveredEpic {
	keyMap := make(map[string]DiscoveredEpic)
	for _, epic := range epics {
		keyMap[epic.Key] = epic
	}
	return keyMap
}

// buildFeatureKeyMap creates a map of feature keys to DiscoveredFeature for fast lookup
func (d *ConflictDetector) buildFeatureKeyMap(features []DiscoveredFeature) map[string]DiscoveredFeature {
	keyMap := make(map[string]DiscoveredFeature)
	for _, feature := range features {
		keyMap[feature.Key] = feature
	}
	return keyMap
}
