package patterns

// GetDefaultPatterns returns the default pattern configuration
// These patterns match common documentation conventions and include
// named capture groups required for relationship inference
func GetDefaultPatterns() *PatternConfig {
	return &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{
				// Standard E##-slug format with named capture groups
				`^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
				// Special epic types (tech-debt, bugs, change-cards)
				`^(?P<epic_id>tech-debt|bugs|change-cards)$`,
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
				// Nested format: F##-slug (when features are nested under intermediate folders)
				`^F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
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
			Folder: []string{
				// Tasks are typically files, not folders
			},
			File: []string{
				// Full task key format: T-E##-F##-###.md
				`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`,
				// Number-based format: ###-task-name.md
				`^(?P<number>\d{3})-(?P<slug>.+)\.md$`,
				// Legacy PRP format: task-name.prp.md
				`^(?P<slug>.+)\.prp\.md$`,
			},
			Generation: GenerationFormat{
				Format: "T-E{epic:02d}-F{feature:02d}-{number:03d}.md",
			},
		},
	}
}
