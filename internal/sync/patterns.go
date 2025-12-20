package sync

import (
	"fmt"
	"regexp"
)

// PatternType represents the type of file pattern for task files
type PatternType string

const (
	// PatternTypeTask matches standard task files: T-E##-F##-###.md
	PatternTypeTask PatternType = "task"

	// PatternTypePRP matches legacy PRP (Product Requirement Prompt) files: ##-*.md
	PatternTypePRP PatternType = "prp"
)

// FilePattern defines a pattern for matching task files
type FilePattern struct {
	Name    PatternType
	Regex   *regexp.Regexp
	Enabled bool
}

// PatternRegistry manages multiple file patterns
type PatternRegistry struct {
	patterns map[PatternType]*FilePattern
}

// NewPatternRegistry creates a new pattern registry with default patterns
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{
		patterns: map[PatternType]*FilePattern{
			PatternTypeTask: {
				Name:    PatternTypeTask,
				Regex:   regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}\.md$`),
				Enabled: true, // Task pattern enabled by default
			},
			PatternTypePRP: {
				Name:    PatternTypePRP,
				Regex:   regexp.MustCompile(`^\d{2}-.*\.md$`),
				Enabled: false, // PRP pattern disabled by default for backward compatibility
			},
		},
	}
}

// EnablePattern enables a specific pattern type
func (r *PatternRegistry) EnablePattern(pt PatternType) error {
	pattern, exists := r.patterns[pt]
	if !exists {
		return fmt.Errorf("unknown pattern type: %s", pt)
	}
	pattern.Enabled = true
	return nil
}

// DisablePattern disables a specific pattern type
func (r *PatternRegistry) DisablePattern(pt PatternType) error {
	pattern, exists := r.patterns[pt]
	if !exists {
		return fmt.Errorf("unknown pattern type: %s", pt)
	}
	pattern.Enabled = false
	return nil
}

// GetPattern retrieves a specific pattern by type
func (r *PatternRegistry) GetPattern(pt PatternType) (*FilePattern, error) {
	pattern, exists := r.patterns[pt]
	if !exists {
		return nil, fmt.Errorf("unknown pattern type: %s", pt)
	}
	return pattern, nil
}

// GetEnabledPatterns returns all enabled patterns
func (r *PatternRegistry) GetEnabledPatterns() []*FilePattern {
	enabled := make([]*FilePattern, 0, len(r.patterns))
	for _, pattern := range r.patterns {
		if pattern.Enabled {
			enabled = append(enabled, pattern)
		}
	}
	return enabled
}

// SetActivePatterns enables specific patterns and disables all others
func (r *PatternRegistry) SetActivePatterns(patterns []PatternType) error {
	// First, disable all patterns
	for _, pattern := range r.patterns {
		pattern.Enabled = false
	}

	// Then enable requested patterns
	for _, pt := range patterns {
		if err := r.EnablePattern(pt); err != nil {
			return err
		}
	}

	return nil
}

// MatchesAnyPattern checks if a filename matches any enabled pattern
// Returns the matching pattern type or empty string if no match
func (r *PatternRegistry) MatchesAnyPattern(filename string) (PatternType, bool) {
	for _, pattern := range r.GetEnabledPatterns() {
		if pattern.Regex.MatchString(filename) {
			return pattern.Name, true
		}
	}
	return "", false
}
