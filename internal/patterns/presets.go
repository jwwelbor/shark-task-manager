package patterns

import (
	"fmt"
)

// PresetInfo contains metadata about a pattern preset
type PresetInfo struct {
	Name        string
	Description string
}

// presetRegistry stores all available presets
var presetRegistry = map[string]*presetDefinition{
	"standard": {
		info: PresetInfo{
			Name:        "standard",
			Description: "Default E##-slug conventions (shipped by default)",
		},
		config: func() *PatternConfig {
			return &PatternConfig{
				Epic: EntityPatterns{
					Folder: []string{
						// Standard E##-slug format with named capture groups
						`^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
					},
					File: []string{
						// Standard epic.md file
						`^epic\.md$`,
					},
					Generation: GenerationFormat{
						Format: "E{number:02d}-{slug}",
					},
				},
				Feature: EntityPatterns{
					Folder: []string{
						// Standard E##-F##-slug format
						`^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
					},
					File: []string{
						// Priority: prd.md (most common)
						`^prd\.md$`,
						// Alternative: PRD_F##-name.md format
						`^PRD_F(?P<number>\d{2})-(?P<slug>.+)\.md$`,
						// Fallback: any markdown file with a slug
						`^(?P<slug>[a-z0-9-]+)\.md$`,
					},
					Generation: GenerationFormat{
						Format: "E{epic:02d}-F{number:02d}-{slug}",
					},
				},
				Task: EntityPatterns{
					Folder: []string{},
					File: []string{
						// Full task key format: T-E##-F##-###.md
						`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`,
						// Number-based format: ###-task-name.md
						`^(?P<number>\d{3})-(?P<slug>.+)\.md$`,
					},
					Generation: GenerationFormat{
						Format: "T-E{epic:02d}-F{feature:02d}-{number:03d}.md",
					},
				},
			}
		},
	},
	"special-epics": {
		info: PresetInfo{
			Name:        "special-epics",
			Description: "Patterns for tech-debt, bugs, change-cards",
		},
		config: func() *PatternConfig {
			return &PatternConfig{
				Epic: EntityPatterns{
					Folder: []string{
						// Special epic types
						`^(?P<epic_id>tech-debt|bugs|change-cards)$`,
					},
					File:       []string{},
					Generation: GenerationFormat{},
				},
				Feature: EntityPatterns{},
				Task:    EntityPatterns{},
			}
		},
	},
	"numeric-only": {
		info: PresetInfo{
			Name:        "numeric-only",
			Description: "E001, F001, T001 numbering without slugs",
		},
		config: func() *PatternConfig {
			return &PatternConfig{
				Epic: EntityPatterns{
					Folder: []string{
						// Numeric-only epic format: E001
						`^E(?P<number>\d{3})$`,
					},
					File: []string{
						`^epic\.md$`,
					},
					Generation: GenerationFormat{
						Format: "E{number:03d}",
					},
				},
				Feature: EntityPatterns{
					Folder: []string{
						// Numeric-only feature format: E001-F001
						`^E(?P<epic_num>\d{3})-F(?P<number>\d{3})$`,
					},
					File: []string{
						`^prd\.md$`,
						`^feature\.md$`,
					},
					Generation: GenerationFormat{
						Format: "E{epic:03d}-F{number:03d}",
					},
				},
				Task: EntityPatterns{
					Folder: []string{},
					File: []string{
						// Numeric-only task format: T-E001-F001-001.md
						`^T-E(?P<epic_num>\d{3})-F(?P<feature_num>\d{3})-(?P<number>\d{3})\.md$`,
					},
					Generation: GenerationFormat{
						Format: "T-E{epic:03d}-F{feature:03d}-{number:03d}.md",
					},
				},
			}
		},
	},
	"legacy-prp": {
		info: PresetInfo{
			Name:        "legacy-prp",
			Description: "Support for .prp.md files in prps/ subfolder",
		},
		config: func() *PatternConfig {
			return &PatternConfig{
				Epic:    EntityPatterns{},
				Feature: EntityPatterns{},
				Task: EntityPatterns{
					Folder: []string{},
					File: []string{
						// Legacy PRP format: ##-task-name.prp.md
						`^(?P<number>\d{2})-(?P<slug>.+)\.prp\.md$`,
						// Alternative: task-name.prp.md (without number)
						`^(?P<slug>[a-z0-9-]+)\.prp\.md$`,
					},
					Generation: GenerationFormat{
						Format: "{number:02d}-{slug}.prp.md",
					},
				},
			}
		},
	},
}

// presetDefinition stores preset metadata and config factory
type presetDefinition struct {
	info   PresetInfo
	config func() *PatternConfig
}

// GetPresetNames returns a list of all available preset names
func GetPresetNames() []string {
	names := make([]string, 0, len(presetRegistry))
	for name := range presetRegistry {
		names = append(names, name)
	}
	return names
}

// GetPresetInfo returns metadata for a specific preset
func GetPresetInfo(name string) (*PresetInfo, error) {
	preset, exists := presetRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown preset: %s", name)
	}

	info := preset.info
	return &info, nil
}

// GetPreset returns the pattern configuration for a specific preset
func GetPreset(name string) (*PatternConfig, error) {
	preset, exists := presetRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown preset: %s", name)
	}

	// Call the config factory to get a fresh copy
	return preset.config(), nil
}

// ListPresets returns a list of all available presets with their metadata
func ListPresets() []PresetInfo {
	presets := make([]PresetInfo, 0, len(presetRegistry))

	// Return in consistent order
	order := []string{"standard", "special-epics", "numeric-only", "legacy-prp"}
	for _, name := range order {
		if preset, exists := presetRegistry[name]; exists {
			presets = append(presets, preset.info)
		}
	}

	return presets
}

// MergePatterns merges a preset configuration into a base configuration
// Patterns from preset are appended to base patterns, skipping duplicates
// Generation formats from base are preserved
func MergePatterns(base, preset *PatternConfig) *PatternConfig {
	if base == nil {
		return preset
	}
	if preset == nil {
		return base
	}

	result := &PatternConfig{
		Epic: EntityPatterns{
			Folder:     mergeStringSlices(base.Epic.Folder, preset.Epic.Folder),
			File:       mergeStringSlices(base.Epic.File, preset.Epic.File),
			Generation: base.Epic.Generation,
		},
		Feature: EntityPatterns{
			Folder:     mergeStringSlices(base.Feature.Folder, preset.Feature.Folder),
			File:       mergeStringSlices(base.Feature.File, preset.Feature.File),
			Generation: base.Feature.Generation,
		},
		Task: EntityPatterns{
			Folder:     mergeStringSlices(base.Task.Folder, preset.Task.Folder),
			File:       mergeStringSlices(base.Task.File, preset.Task.File),
			Generation: base.Task.Generation,
		},
	}

	// If base doesn't have generation format but preset does, use preset's
	if result.Epic.Generation.Format == "" && preset.Epic.Generation.Format != "" {
		result.Epic.Generation = preset.Epic.Generation
	}
	if result.Feature.Generation.Format == "" && preset.Feature.Generation.Format != "" {
		result.Feature.Generation = preset.Feature.Generation
	}
	if result.Task.Generation.Format == "" && preset.Task.Generation.Format != "" {
		result.Task.Generation = preset.Task.Generation
	}

	return result
}

// mergeStringSlices merges two string slices, appending items from b to a
// Duplicates are skipped
func mergeStringSlices(a, b []string) []string {
	// Create a map to track existing items
	seen := make(map[string]bool)
	for _, item := range a {
		seen[item] = true
	}

	// Start with copy of a
	result := make([]string, len(a))
	copy(result, a)

	// Append items from b that aren't duplicates
	for _, item := range b {
		if !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}

	return result
}

// GetMergeStats returns information about what was merged
type MergeStats struct {
	Added   int
	Skipped int
	Details []string
}

// MergePatternsWithStats merges patterns and returns statistics about the merge
func MergePatternsWithStats(base, preset *PatternConfig) (*PatternConfig, *MergeStats) {
	stats := &MergeStats{
		Details: []string{},
	}

	if base == nil {
		return preset, stats
	}
	if preset == nil {
		return base, stats
	}

	// Merge epic patterns
	epicFolderAdded, epicFolderSkipped := countMerge(base.Epic.Folder, preset.Epic.Folder)
	epicFileAdded, epicFileSkipped := countMerge(base.Epic.File, preset.Epic.File)

	// Merge feature patterns
	featureFolderAdded, featureFolderSkipped := countMerge(base.Feature.Folder, preset.Feature.Folder)
	featureFileAdded, featureFileSkipped := countMerge(base.Feature.File, preset.Feature.File)

	// Merge task patterns
	taskFolderAdded, taskFolderSkipped := countMerge(base.Task.Folder, preset.Task.Folder)
	taskFileAdded, taskFileSkipped := countMerge(base.Task.File, preset.Task.File)

	// Calculate totals
	stats.Added = epicFolderAdded + epicFileAdded + featureFolderAdded + featureFileAdded + taskFolderAdded + taskFileAdded
	stats.Skipped = epicFolderSkipped + epicFileSkipped + featureFolderSkipped + featureFileSkipped + taskFolderSkipped + taskFileSkipped

	// Generate details
	if epicFolderAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d epic folder pattern(s)", epicFolderAdded))
	}
	if epicFileAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d epic file pattern(s)", epicFileAdded))
	}
	if featureFolderAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d feature folder pattern(s)", featureFolderAdded))
	}
	if featureFileAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d feature file pattern(s)", featureFileAdded))
	}
	if taskFolderAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d task folder pattern(s)", taskFolderAdded))
	}
	if taskFileAdded > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Added %d task file pattern(s)", taskFileAdded))
	}

	if stats.Skipped > 0 {
		stats.Details = append(stats.Details, fmt.Sprintf("Skipped %d duplicate pattern(s)", stats.Skipped))
	}

	// Perform the actual merge
	result := MergePatterns(base, preset)

	return result, stats
}

// countMerge counts how many items would be added vs skipped when merging b into a
func countMerge(a, b []string) (added, skipped int) {
	seen := make(map[string]bool)
	for _, item := range a {
		seen[item] = true
	}

	for _, item := range b {
		if seen[item] {
			skipped++
		} else {
			added++
			seen[item] = true
		}
	}

	return added, skipped
}
