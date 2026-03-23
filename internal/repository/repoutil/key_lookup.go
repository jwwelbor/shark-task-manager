// Package repoutil provides shared utilities for repository sub-packages.
// It contains key parsing, order resequencing, and tracing helpers that
// all entity-specific sub-packages (task, epic, feature, note, etc.) can
// import without creating circular dependencies.
package repoutil

import (
	"strings"
)

// ContainsHyphen checks if a string contains a hyphen character.
// Used to determine whether a key might contain a slug suffix.
func ContainsHyphen(s string) bool {
	return strings.ContainsRune(s, '-')
}

// IsNumeric checks if a string consists entirely of digit characters.
// Returns false for empty strings.
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// SplitAtFirstHyphen splits a string at the first hyphen into two parts.
// Returns the part before the hyphen and the part after it.
// If there is no hyphen, returns the original string and an empty string
// with ok=false.
//
// Examples:
//
//	SplitAtFirstHyphen("E04-epic-name") -> ("E04", "epic-name", true)
//	SplitAtFirstHyphen("E04")           -> ("E04", "", false)
func SplitAtFirstHyphen(key string) (prefix, suffix string, ok bool) {
	idx := strings.IndexByte(key, '-')
	if idx == -1 {
		return key, "", false
	}
	return key[:idx], key[idx+1:], true
}

// SplitAtNthHyphen splits a string at the Nth hyphen (1-indexed).
// Returns the part before the Nth hyphen and the part after it.
// If there are fewer than n hyphens, returns the original string and empty
// string with ok=false.
// If the Nth hyphen is found, ok=true even if the suffix is empty (trailing hyphen).
//
// Examples:
//
//	SplitAtNthHyphen("T-E07-F01-001-slug-text", 4) -> ("T-E07-F01-001", "slug-text", true)
//	SplitAtNthHyphen("T-E07-F01-001", 4)           -> ("T-E07-F01-001", "", false)
//	SplitAtNthHyphen("T-E07-F01-001-", 4)          -> ("T-E07-F01-001", "", true)
func SplitAtNthHyphen(key string, n int) (prefix, suffix string, ok bool) {
	count := 0
	for i, ch := range key {
		if ch == '-' {
			count++
			if count == n {
				return key[:i], key[i+1:], true
			}
		}
	}
	return key, "", false
}
