package patterns

// PatternConfig contains regex patterns for epic/feature/task discovery
type PatternConfig struct {
	Epic    EntityPatterns `json:"epic"`
	Feature EntityPatterns `json:"feature"`
	Task    EntityPatterns `json:"task"`
}

// EntityPatterns contains patterns for a specific entity type
type EntityPatterns struct {
	Folder     []string         `json:"folder"`
	File       []string         `json:"file"`
	Generation GenerationFormat `json:"generation"`
}

// GenerationFormat defines the template for creating new items
type GenerationFormat struct {
	Format string `json:"format"`
}
