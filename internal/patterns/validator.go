package patterns

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a pattern validation error with context
type ValidationError struct {
	PatternName string
	Pattern     string
	EntityType  string
	Message     string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s pattern] %s: %s", e.EntityType, e.PatternName, e.Message)
}

// ValidationWarning represents a non-critical pattern issue
type ValidationWarning struct {
	PatternName string
	Pattern     string
	EntityType  string
	Message     string
}

func (w *ValidationWarning) String() string {
	return fmt.Sprintf("[%s pattern warning] %s: %s", w.EntityType, w.PatternName, w.Message)
}

// ValidatePattern validates a single pattern for correctness
// Returns error if pattern is invalid, nil if valid
func ValidatePattern(pattern, entityType string) error {
	// 1. Validate regex syntax
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return &ValidationError{
			Pattern:    pattern,
			EntityType: entityType,
			Message:    fmt.Sprintf("invalid regex syntax: %v", err),
		}
	}

	// 2. Check for catastrophic backtracking potential
	if hasCatastrophicBacktracking(pattern) {
		return &ValidationError{
			Pattern:    pattern,
			EntityType: entityType,
			Message:    "pattern has catastrophic backtracking potential (nested quantifiers like (a+)+ are not allowed)",
		}
	}

	// 3. Extract capture group names
	groups := extractCaptureGroupNames(pattern)

	// 4. Validate required capture groups based on entity type
	if err := validateRequiredCaptureGroups(groups, entityType, pattern); err != nil {
		return err
	}

	// 5. Store compiled regex for caching (in actual implementation)
	_ = compiled

	return nil
}

// GetPatternWarnings returns warnings for a pattern (non-fatal issues)
func GetPatternWarnings(pattern, entityType string) []string {
	var warnings []string

	// Extract capture group names
	groups := extractCaptureGroupNames(pattern)

	// Check for unrecognized capture group names
	recognizedGroups := map[string]bool{
		"epic_id":      true,
		"epic_slug":    true,
		"epic_num":     true,
		"feature_id":   true,
		"feature_slug": true,
		"feature_num":  true,
		"task_id":      true,
		"task_slug":    true,
		"task_num":     true,
		"number":       true,
		"slug":         true,
	}

	for _, group := range groups {
		if !recognizedGroups[group] {
			// Suggest similar names
			suggestions := findSimilarNames(group, recognizedGroups)
			warning := fmt.Sprintf("capture group '%s' is not recognized and will be ignored", group)
			if len(suggestions) > 0 {
				warning += fmt.Sprintf(". Did you mean '%s'?", strings.Join(suggestions, "' or '"))
			}
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

// ValidatePatternConfig validates all patterns in a configuration
func ValidatePatternConfig(config *PatternConfig) error {
	var errors []string

	// Validate epic patterns
	for i, pattern := range config.Epic.Folder {
		if err := ValidatePattern(pattern, "epic"); err != nil {
			errors = append(errors, fmt.Sprintf("epic folder pattern #%d: %v", i+1, err))
		}
	}
	// File patterns for epic don't need capture groups (they're in parent folder context)
	for i, pattern := range config.Epic.File {
		if err := ValidatePatternSyntaxOnly(pattern); err != nil {
			errors = append(errors, fmt.Sprintf("epic file pattern #%d: %v", i+1, err))
		}
	}

	// Validate feature patterns
	for i, pattern := range config.Feature.Folder {
		if err := ValidatePattern(pattern, "feature"); err != nil {
			errors = append(errors, fmt.Sprintf("feature folder pattern #%d: %v", i+1, err))
		}
	}
	// File patterns for feature don't need capture groups (they're in parent folder context)
	for i, pattern := range config.Feature.File {
		if err := ValidatePatternSyntaxOnly(pattern); err != nil {
			errors = append(errors, fmt.Sprintf("feature file pattern #%d: %v", i+1, err))
		}
	}

	// Validate task patterns
	for i, pattern := range config.Task.Folder {
		if err := ValidatePattern(pattern, "task"); err != nil {
			errors = append(errors, fmt.Sprintf("task folder pattern #%d: %v", i+1, err))
		}
	}
	// Task file patterns need full validation since tasks are typically files
	for i, pattern := range config.Task.File {
		// Task files can be either full task keys (with all identifiers) or simple slugs (in parent context)
		// So we validate syntax but relax capture group requirements
		if err := ValidatePatternSyntaxOnly(pattern); err != nil {
			errors = append(errors, fmt.Sprintf("task file pattern #%d: %v", i+1, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("pattern validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ValidatePatternSyntaxOnly validates only regex syntax and catastrophic backtracking
// Used for file patterns that don't require capture groups
func ValidatePatternSyntaxOnly(pattern string) error {
	// 1. Validate regex syntax
	_, err := regexp.Compile(pattern)
	if err != nil {
		return &ValidationError{
			Pattern: pattern,
			Message: fmt.Sprintf("invalid regex syntax: %v", err),
		}
	}

	// 2. Check for catastrophic backtracking potential
	if hasCatastrophicBacktracking(pattern) {
		return &ValidationError{
			Pattern: pattern,
			Message: "pattern has catastrophic backtracking potential (nested quantifiers like (a+)+ are not allowed)",
		}
	}

	return nil
}

// extractCaptureGroupNames extracts named capture group names from a regex pattern
func extractCaptureGroupNames(pattern string) []string {
	// Match named groups like (?P<name>...)
	groupRegex := regexp.MustCompile(`\(\?P<([^>]+)>`)
	matches := groupRegex.FindAllStringSubmatch(pattern, -1)

	var groups []string
	for _, match := range matches {
		if len(match) > 1 {
			groups = append(groups, match[1])
		}
	}

	return groups
}

// validateRequiredCaptureGroups checks if pattern has required capture groups for entity type
func validateRequiredCaptureGroups(groups []string, entityType, pattern string) error {
	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[g] = true
	}

	switch entityType {
	case "epic":
		// Epic patterns must include at least one of: epic_id, epic_slug, number, slug
		if !groupSet["epic_id"] && !groupSet["epic_slug"] && !groupSet["number"] && !groupSet["slug"] {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include at least one of 'epic_id', 'epic_slug', 'number', or 'slug'",
			}
		}

	case "feature":
		// Feature patterns must include (epic_id OR epic_num) AND (feature_id OR feature_slug OR number OR slug)
		hasEpic := groupSet["epic_id"] || groupSet["epic_num"]
		hasFeature := groupSet["feature_id"] || groupSet["feature_slug"] || groupSet["number"] || groupSet["slug"]

		if !hasEpic {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include 'epic_id' or 'epic_num' to identify parent epic",
			}
		}
		if !hasFeature {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include at least one of 'feature_id', 'feature_slug', 'number', or 'slug'",
			}
		}

	case "task":
		// Task patterns must include (epic_id OR epic_num) AND (feature_id OR feature_num) AND (task_id OR number OR task_slug OR slug)
		hasEpic := groupSet["epic_id"] || groupSet["epic_num"]
		hasFeature := groupSet["feature_id"] || groupSet["feature_num"]
		hasTask := groupSet["task_id"] || groupSet["number"] || groupSet["task_slug"] || groupSet["slug"]

		if !hasEpic {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include 'epic_id' or 'epic_num' to identify parent epic",
			}
		}
		if !hasFeature {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include 'feature_id' or 'feature_num' to identify parent feature",
			}
		}
		if !hasTask {
			return &ValidationError{
				Pattern:    pattern,
				EntityType: entityType,
				Message:    "missing required capture group: must include at least one of 'task_id', 'number', 'task_slug', or 'slug'",
			}
		}
	}

	return nil
}

// hasCatastrophicBacktracking detects patterns with potential catastrophic backtracking
func hasCatastrophicBacktracking(pattern string) bool {
	// Detect nested quantifiers like (a+)+, (a*)*, etc.
	// This is a simplified check - more sophisticated analysis could be added

	// Remove escaped characters and named groups to simplify analysis
	simplified := regexp.MustCompile(`\\.`).ReplaceAllString(pattern, "")
	simplified = regexp.MustCompile(`\(\?P<[^>]+>`).ReplaceAllString(simplified, "(")

	// Check for patterns like (...)+ or (...)* where ... contains + or *
	nestedQuantifierPattern := regexp.MustCompile(`\([^)]*[+*][^)]*\)[+*]`)
	if nestedQuantifierPattern.MatchString(simplified) {
		return true
	}

	return false
}

// findSimilarNames finds similar names to suggest as alternatives
func findSimilarNames(name string, recognized map[string]bool) []string {
	var suggestions []string

	// Simple similarity: check if recognized name contains part of input or vice versa
	nameLower := strings.ToLower(name)

	for recognizedName := range recognized {
		recognizedLower := strings.ToLower(recognizedName)

		// Check if one contains the other
		if strings.Contains(recognizedLower, nameLower) || strings.Contains(nameLower, recognizedLower) {
			suggestions = append(suggestions, recognizedName)
		}
	}

	// Limit to 2 suggestions
	if len(suggestions) > 2 {
		suggestions = suggestions[:2]
	}

	return suggestions
}

// ValidateWithTimeout validates a pattern with a timeout to prevent DoS
func ValidateWithTimeout(pattern, entityType string, timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- ValidatePattern(pattern, entityType)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return &ValidationError{
			Pattern:    pattern,
			EntityType: entityType,
			Message:    fmt.Sprintf("validation timed out after %v (possible regex DoS pattern)", timeout),
		}
	}
}
