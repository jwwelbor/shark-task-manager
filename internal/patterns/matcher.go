package patterns

import (
	"fmt"
	"regexp"
	"time"
)

// CompiledPattern represents a compiled regex pattern with metadata
type CompiledPattern struct {
	Original string
	Compiled *regexp.Regexp
	Index    int // Position in original pattern array
}

// MatchResult represents the result of pattern matching
type MatchResult struct {
	Matched       bool
	PatternIndex  int
	PatternString string
	CaptureGroups map[string]string
}

// PatternMatcher handles pattern matching with first-match-wins semantics
type PatternMatcher struct {
	epicFolderPatterns   []*CompiledPattern
	epicFilePatterns     []*CompiledPattern
	featureFolderPatterns []*CompiledPattern
	featureFilePatterns   []*CompiledPattern
	taskFolderPatterns    []*CompiledPattern
	taskFilePatterns      []*CompiledPattern
	verbose               bool
}

// NewPatternMatcher creates a new pattern matcher with compiled patterns cached
func NewPatternMatcher(config *PatternConfig, verbose bool) (*PatternMatcher, error) {
	if config == nil {
		return nil, fmt.Errorf("pattern config cannot be nil")
	}

	matcher := &PatternMatcher{
		verbose: verbose,
	}

	var err error

	// Compile epic patterns
	matcher.epicFolderPatterns, err = compilePatterns(config.Epic.Folder)
	if err != nil {
		return nil, fmt.Errorf("failed to compile epic folder patterns: %w", err)
	}
	matcher.epicFilePatterns, err = compilePatterns(config.Epic.File)
	if err != nil {
		return nil, fmt.Errorf("failed to compile epic file patterns: %w", err)
	}

	// Compile feature patterns
	matcher.featureFolderPatterns, err = compilePatterns(config.Feature.Folder)
	if err != nil {
		return nil, fmt.Errorf("failed to compile feature folder patterns: %w", err)
	}
	matcher.featureFilePatterns, err = compilePatterns(config.Feature.File)
	if err != nil {
		return nil, fmt.Errorf("failed to compile feature file patterns: %w", err)
	}

	// Compile task patterns
	matcher.taskFolderPatterns, err = compilePatterns(config.Task.Folder)
	if err != nil {
		return nil, fmt.Errorf("failed to compile task folder patterns: %w", err)
	}
	matcher.taskFilePatterns, err = compilePatterns(config.Task.File)
	if err != nil {
		return nil, fmt.Errorf("failed to compile task file patterns: %w", err)
	}

	return matcher, nil
}

// compilePatterns compiles a slice of regex patterns with caching
func compilePatterns(patterns []string) ([]*CompiledPattern, error) {
	compiled := make([]*CompiledPattern, 0, len(patterns))

	for i, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("pattern #%d (%s): %w", i+1, pattern, err)
		}

		compiled = append(compiled, &CompiledPattern{
			Original: pattern,
			Compiled: regex,
			Index:    i,
		})
	}

	return compiled, nil
}

// MatchEpicFolder matches a folder name against epic folder patterns
// Uses first-match-wins semantics: returns after first successful match
func (m *PatternMatcher) MatchEpicFolder(folderName string) *MatchResult {
	return m.matchWithTimeout(folderName, m.epicFolderPatterns, "epic folder", 100*time.Millisecond)
}

// MatchEpicFile matches a file name against epic file patterns
func (m *PatternMatcher) MatchEpicFile(fileName string) *MatchResult {
	return m.matchWithTimeout(fileName, m.epicFilePatterns, "epic file", 100*time.Millisecond)
}

// MatchFeatureFolder matches a folder name against feature folder patterns
func (m *PatternMatcher) MatchFeatureFolder(folderName string) *MatchResult {
	return m.matchWithTimeout(folderName, m.featureFolderPatterns, "feature folder", 100*time.Millisecond)
}

// MatchFeatureFile matches a file name against feature file patterns
func (m *PatternMatcher) MatchFeatureFile(fileName string) *MatchResult {
	return m.matchWithTimeout(fileName, m.featureFilePatterns, "feature file", 100*time.Millisecond)
}

// MatchTaskFolder matches a folder name against task folder patterns
func (m *PatternMatcher) MatchTaskFolder(folderName string) *MatchResult {
	return m.matchWithTimeout(folderName, m.taskFolderPatterns, "task folder", 100*time.Millisecond)
}

// MatchTaskFile matches a file name against task file patterns
func (m *PatternMatcher) MatchTaskFile(fileName string) *MatchResult {
	return m.matchWithTimeout(fileName, m.taskFilePatterns, "task file", 100*time.Millisecond)
}

// matchWithTimeout performs pattern matching with timeout protection
func (m *PatternMatcher) matchWithTimeout(input string, patterns []*CompiledPattern, entityType string, timeout time.Duration) *MatchResult {
	done := make(chan *MatchResult, 1)

	go func() {
		done <- m.match(input, patterns, entityType)
	}()

	select {
	case result := <-done:
		return result
	case <-time.After(timeout):
		// Timeout occurred - return no match with error indication
		if m.verbose {
			fmt.Printf("[TIMEOUT] Pattern matching for %s '%s' exceeded %v\n", entityType, input, timeout)
		}
		return &MatchResult{
			Matched: false,
		}
	}
}

// match performs the actual pattern matching with first-match-wins semantics
func (m *PatternMatcher) match(input string, patterns []*CompiledPattern, entityType string) *MatchResult {
	attemptedPatterns := make([]string, 0, len(patterns))

	// Iterate patterns in order - first match wins
	for _, pattern := range patterns {
		attemptedPatterns = append(attemptedPatterns, pattern.Original)

		// Check if pattern matches
		if !pattern.Compiled.MatchString(input) {
			// No match, continue to next pattern
			continue
		}

		// Match found - extract capture groups
		match := pattern.Compiled.FindStringSubmatch(input)
		if match == nil {
			// Shouldn't happen since MatchString returned true, but be defensive
			continue
		}

		captureGroups := make(map[string]string)
		groupNames := pattern.Compiled.SubexpNames()

		for i, name := range groupNames {
			if i == 0 || name == "" {
				continue // Skip whole match and unnamed groups
			}
			if i < len(match) {
				captureGroups[name] = match[i]
			}
		}

		if m.verbose {
			fmt.Printf("[MATCH] %s '%s' matched pattern #%d: %s\n", entityType, input, pattern.Index+1, pattern.Original)
			fmt.Printf("  Captured groups: %v\n", captureGroups)
		}

		// First match found - return immediately (short-circuit)
		return &MatchResult{
			Matched:       true,
			PatternIndex:  pattern.Index,
			PatternString: pattern.Original,
			CaptureGroups: captureGroups,
		}
	}

	// No patterns matched
	if m.verbose {
		fmt.Printf("[NO MATCH] %s '%s' did not match any patterns. Attempted %d patterns:\n", entityType, input, len(attemptedPatterns))
		for i, p := range attemptedPatterns {
			fmt.Printf("  %d. %s\n", i+1, p)
		}
	}

	return &MatchResult{
		Matched: false,
	}
}

// SetVerbose enables or disables verbose logging
func (m *PatternMatcher) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// GetAttemptedPatterns returns the list of patterns that would be attempted for an entity type
func (m *PatternMatcher) GetAttemptedPatterns(entityType, subType string) []string {
	var patterns []*CompiledPattern

	switch entityType {
	case "epic":
		if subType == "folder" {
			patterns = m.epicFolderPatterns
		} else {
			patterns = m.epicFilePatterns
		}
	case "feature":
		if subType == "folder" {
			patterns = m.featureFolderPatterns
		} else {
			patterns = m.featureFilePatterns
		}
	case "task":
		if subType == "folder" {
			patterns = m.taskFolderPatterns
		} else {
			patterns = m.taskFilePatterns
		}
	default:
		return []string{}
	}

	result := make([]string, len(patterns))
	for i, p := range patterns {
		result[i] = p.Original
	}
	return result
}
