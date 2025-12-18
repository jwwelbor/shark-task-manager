package patterns

import (
	"fmt"
	"time"
)

// PatternRegistry manages pattern matching for epic, feature, and task discovery
// It provides a unified interface for loading, validating, and matching patterns
// from configuration, with support for pattern precedence and caching
type PatternRegistry struct {
	config  *PatternConfig
	matcher *PatternMatcher
	verbose bool
}

// RegistryOptions configures PatternRegistry behavior
type RegistryOptions struct {
	Verbose bool
}

// NewPatternRegistry creates a new pattern registry from configuration
// It loads patterns, validates them, compiles regex patterns, and prepares for matching
func NewPatternRegistry(config *PatternConfig, opts *RegistryOptions) (*PatternRegistry, error) {
	if config == nil {
		return nil, fmt.Errorf("pattern config cannot be nil")
	}

	verbose := false
	if opts != nil {
		verbose = opts.Verbose
	}

	// Validate configuration before creating registry
	if err := ValidatePatternConfig(config); err != nil {
		return nil, fmt.Errorf("pattern validation failed: %w", err)
	}

	// Create pattern matcher with compiled patterns cached
	matcher, err := NewPatternMatcher(config, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern matcher: %w", err)
	}

	return &PatternRegistry{
		config:  config,
		matcher: matcher,
		verbose: verbose,
	}, nil
}

// NewPatternRegistryFromDefaults creates a pattern registry using default patterns
func NewPatternRegistryFromDefaults(verbose bool) (*PatternRegistry, error) {
	config := GetDefaultPatterns()
	opts := &RegistryOptions{Verbose: verbose}
	return NewPatternRegistry(config, opts)
}

// LoadPatternRegistryFromFile loads a pattern registry from a configuration file
func LoadPatternRegistryFromFile(configPath string, verbose bool) (*PatternRegistry, error) {
	// Load config from file
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Merge with defaults if patterns are missing
	if config.Patterns == nil {
		config.Patterns = GetDefaultPatterns()
	} else {
		config.Patterns = MergeWithDefaults(config.Patterns)
	}

	// Create registry
	opts := &RegistryOptions{Verbose: verbose}
	return NewPatternRegistry(config.Patterns, opts)
}

// MatchTaskFile matches a task filename against configured task file patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchTaskFile(filename string) *MatchResult {
	return r.matcher.MatchTaskFile(filename)
}

// MatchTaskFolder matches a task folder name against configured task folder patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchTaskFolder(foldername string) *MatchResult {
	return r.matcher.MatchTaskFolder(foldername)
}

// MatchFeatureFile matches a feature filename against configured feature file patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchFeatureFile(filename string) *MatchResult {
	return r.matcher.MatchFeatureFile(filename)
}

// MatchFeatureFolder matches a feature folder name against configured feature folder patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchFeatureFolder(foldername string) *MatchResult {
	return r.matcher.MatchFeatureFolder(foldername)
}

// MatchEpicFile matches an epic filename against configured epic file patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchEpicFile(filename string) *MatchResult {
	return r.matcher.MatchEpicFile(filename)
}

// MatchEpicFolder matches an epic folder name against configured epic folder patterns
// Returns MatchResult with first matching pattern (first-match-wins)
func (r *PatternRegistry) MatchEpicFolder(foldername string) *MatchResult {
	return r.matcher.MatchEpicFolder(foldername)
}

// GetConfig returns the pattern configuration used by this registry
func (r *PatternRegistry) GetConfig() *PatternConfig {
	return r.config
}

// GetTaskPatterns returns the task file patterns configured in this registry
func (r *PatternRegistry) GetTaskPatterns() []string {
	return r.matcher.GetAttemptedPatterns("task", "file")
}

// GetFeaturePatterns returns the feature folder patterns configured in this registry
func (r *PatternRegistry) GetFeaturePatterns() []string {
	return r.matcher.GetAttemptedPatterns("feature", "folder")
}

// GetEpicPatterns returns the epic folder patterns configured in this registry
func (r *PatternRegistry) GetEpicPatterns() []string {
	return r.matcher.GetAttemptedPatterns("epic", "folder")
}

// SetVerbose enables or disables verbose logging for pattern matching
func (r *PatternRegistry) SetVerbose(verbose bool) {
	r.verbose = verbose
	r.matcher.SetVerbose(verbose)
}

// ValidatePattern validates a single pattern for the given entity type
// Returns error if pattern is invalid, nil if valid
func (r *PatternRegistry) ValidatePattern(pattern, entityType string) error {
	return ValidatePattern(pattern, entityType)
}

// ValidatePatternWithTimeout validates a pattern with a timeout to prevent DoS
func (r *PatternRegistry) ValidatePatternWithTimeout(pattern, entityType string, timeout time.Duration) error {
	return ValidateWithTimeout(pattern, entityType, timeout)
}

// GetPatternWarnings returns warnings for a pattern (non-fatal issues)
func (r *PatternRegistry) GetPatternWarnings(pattern, entityType string) []string {
	return GetPatternWarnings(pattern, entityType)
}

// GenerateTaskKey generates a task key using the configured generation format
// Takes parameters like epic number, feature number, task number, and slug
func (r *PatternRegistry) GenerateTaskKey(params map[string]interface{}) (string, error) {
	format := r.config.Task.Generation.Format
	return ApplyGenerationFormat(format, params)
}

// GenerateFeatureKey generates a feature key using the configured generation format
func (r *PatternRegistry) GenerateFeatureKey(params map[string]interface{}) (string, error) {
	format := r.config.Feature.Generation.Format
	return ApplyGenerationFormat(format, params)
}

// GenerateEpicKey generates an epic key using the configured generation format
func (r *PatternRegistry) GenerateEpicKey(params map[string]interface{}) (string, error) {
	format := r.config.Epic.Generation.Format
	return ApplyGenerationFormat(format, params)
}
