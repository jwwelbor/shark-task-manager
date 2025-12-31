// Package slug provides utilities for generating URL-friendly slugs from task titles.
// Slugs are used to create human-readable task filenames like:
// T-E04-F01-001-some-task-description.md
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	// maxSlugLength is the maximum length of a generated slug to avoid overly long filenames
	maxSlugLength = 100
)

var (
	// nonAlphanumericRegex matches any character that is not alphanumeric or hyphen
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)
	// multipleHyphensRegex matches multiple consecutive hyphens
	multipleHyphensRegex = regexp.MustCompile(`-+`)
)

// Generate creates a URL-friendly slug from a task title.
//
// Rules:
// - Converts to lowercase
// - Replaces spaces and underscores with hyphens
// - Removes special characters and punctuation
// - Handles unicode by removing diacritics (é -> e)
// - Collapses multiple hyphens into one
// - Removes leading/trailing hyphens
// - Truncates to maxSlugLength characters
// - Returns empty string if no valid characters remain
//
// Examples:
//
//	Generate("Some Task Description") -> "some-task-description"
//	Generate("Fix bug: API endpoint") -> "fix-bug-api-endpoint"
//	Generate("Add émoji support") -> "add-emoji-support"
func Generate(title string) string {
	if title == "" {
		return ""
	}

	// Step 1: Normalize unicode characters (decompose then remove diacritics)
	// This converts "é" to "e", "ñ" to "n", etc.
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, title)

	// Step 2: Convert to lowercase
	slug := strings.ToLower(normalized)

	// Step 3: Replace spaces, underscores, and periods with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, ".", "-")

	// Step 4: Remove all non-alphanumeric characters except hyphens
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "")

	// Step 5: Collapse multiple hyphens into single hyphen
	slug = multipleHyphensRegex.ReplaceAllString(slug, "-")

	// Step 6: Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Step 7: Truncate to max length
	if len(slug) > maxSlugLength {
		slug = slug[:maxSlugLength]
		// Re-trim in case truncation ended on a hyphen
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// GenerateFilename creates a task filename with slug from task key and title.
//
// Format: {taskKey}-{slug}.md
// If the slug is empty (title has no valid characters), falls back to: {taskKey}.md
//
// Examples:
//
//	GenerateFilename("T-E04-F01-001", "Some Task Description") -> "T-E04-F01-001-some-task-description.md"
//	GenerateFilename("T-E04-F01-001", "!@#$") -> "T-E04-F01-001.md"
func GenerateFilename(taskKey, title string) string {
	slug := Generate(title)

	if slug == "" {
		// No valid slug, use key only
		return taskKey + ".md"
	}

	// Combine key and slug
	return taskKey + "-" + slug + ".md"
}
