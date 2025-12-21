package commands

import (
	"fmt"
	"regexp"
)

// Compiled regex patterns for pattern matching
var (
	epicKeyPattern       = regexp.MustCompile(`^E\d{2}$`)
	featureKeyPattern    = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
	featureSuffixPattern = regexp.MustCompile(`^F\d{2}$`)
)

// IsEpicKey validates if a string is a valid epic key format (E##)
// Returns true for valid epic keys like E01, E04, E99
// Returns false for invalid formats like E1, e04, E001, etc.
func IsEpicKey(s string) bool {
	return epicKeyPattern.MatchString(s)
}

// IsFeatureKey validates if a string is a valid feature key format (E##-F##)
// Returns true for valid feature keys like E04-F01, E01-F99
// Returns false for invalid formats like E04F01, E4-F01, etc.
func IsFeatureKey(s string) bool {
	return featureKeyPattern.MatchString(s)
}

// IsFeatureKeySuffix validates if a string is a valid feature key suffix (F##)
// Returns true for valid suffixes like F01, F99
// Returns false for invalid formats like F1, f01, etc.
func IsFeatureKeySuffix(s string) bool {
	return featureSuffixPattern.MatchString(s)
}

// ParseFeatureKey parses a combined feature key format (E##-F##) into epic and feature parts
// Returns (epic, feature, nil) for valid input like "E04-F01"
// Returns ("", "", error) for invalid input with clear error message
func ParseFeatureKey(s string) (epic, feature string, err error) {
	if !IsFeatureKey(s) {
		return "", "", fmt.Errorf("invalid feature key format: %q (expected E##-F##)", s)
	}

	// Split on hyphen - we know it's valid format at this point
	// Epic is first 3 chars (E##), Feature is last 3 chars (F##)
	epic = s[:3]
	feature = s[4:7]

	return epic, feature, nil
}

// Deprecated: Use IsEpicKey instead
// isValidEpicKey validates epic key format (E##)
func isValidEpicKey(key string) bool {
	return IsEpicKey(key)
}
