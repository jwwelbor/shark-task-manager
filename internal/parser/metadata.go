package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

// TaskMetadata contains extracted metadata from a task file
type TaskMetadata struct {
	Title       string
	Description string
	TaskKey     string
	Status      string
	Priority    int
	Feature     string
	// Add other fields as needed
}

// ExtractTitleFromFilename extracts a title from the filename based on pattern match
// Converts hyphens to spaces and applies Title Case formatting
func ExtractTitleFromFilename(filename string, patternMatch *patterns.MatchResult) string {
	if patternMatch == nil || !patternMatch.Matched {
		return ""
	}

	// Remove .md extension
	base := strings.TrimSuffix(filename, ".md")
	base = strings.TrimSuffix(base, ".prp.md")

	var descriptive string

	// Extract descriptive part based on capture groups
	if slug, ok := patternMatch.CaptureGroups["slug"]; ok {
		// PRP pattern or standard pattern with slug
		descriptive = slug
	} else if taskKey, ok := patternMatch.CaptureGroups["task_key"]; ok {
		// Standard pattern: remove task key prefix
		// "T-E04-F02-001-implement-caching" -> "implement-caching"
		if strings.HasPrefix(base, taskKey) {
			remaining := base[len(taskKey):]
			if strings.HasPrefix(remaining, "-") {
				descriptive = remaining[1:] // Remove leading hyphen
			}
		}
	} else if number, ok := patternMatch.CaptureGroups["number"]; ok {
		// Numbered pattern: remove number prefix
		// "01-research-phase" -> "research-phase"
		prefix := number + "-"
		if strings.HasPrefix(base, prefix) {
			descriptive = base[len(prefix):]
		}
	}

	// If no descriptive part found, return empty
	if descriptive == "" {
		return ""
	}

	// Convert hyphens to spaces and title-case
	return toTitleCase(descriptive)
}

// ExtractTitleFromMarkdown extracts title from first H1 heading in markdown
// Removes common prefixes like "Task:", "PRP:", "TODO:", "WIP:" (case-insensitive)
func ExtractTitleFromMarkdown(content string) string {
	lines := strings.Split(content, "\n")

	inFrontmatter := false
	pastFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track frontmatter boundaries
		if trimmed == "---" {
			if !pastFrontmatter {
				inFrontmatter = !inFrontmatter
				if !inFrontmatter {
					pastFrontmatter = true
				}
			}
			continue
		}

		// Skip frontmatter content
		if inFrontmatter {
			continue
		}

		// Look for H1 heading
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(trimmed[2:])

			// Remove common prefixes (case-insensitive)
			prefixes := []string{"Task:", "PRP:", "TODO:", "WIP:"}
			for _, prefix := range prefixes {
				if len(title) > len(prefix) &&
					strings.EqualFold(title[:len(prefix)], prefix) {
					title = strings.TrimSpace(title[len(prefix):])
					break
				}
			}

			return title
		}
	}

	return ""
}

// ExtractDescriptionFromMarkdown extracts the first paragraph after frontmatter/H1
// Limits to 500 characters maximum
func ExtractDescriptionFromMarkdown(content string) string {
	lines := strings.Split(content, "\n")

	inFrontmatter := false
	pastFrontmatter := false
	pastFirstHeading := false
	foundHeading := false
	var paragraph strings.Builder

	// Check if content starts with frontmatter
	startsWithFrontmatter := len(lines) > 0 &&
		(strings.HasPrefix(lines[0], "---"))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track frontmatter boundaries
		if trimmed == "---" {
			if !pastFrontmatter && startsWithFrontmatter {
				inFrontmatter = !inFrontmatter
				if !inFrontmatter {
					pastFrontmatter = true
				}
			}
			continue
		}

		// Skip frontmatter content
		if inFrontmatter {
			continue
		}

		// If we started with frontmatter but haven't passed it yet, skip
		if startsWithFrontmatter && !pastFrontmatter {
			continue
		}

		// Check for H1 heading
		if strings.HasPrefix(trimmed, "# ") {
			if !foundHeading {
				foundHeading = true
				pastFirstHeading = true
				continue
			}
			// Second heading - stop here
			break
		}

		// If we haven't found frontmatter and haven't found a heading,
		// we can't extract a description (need to be after heading)
		if startsWithFrontmatter && !pastFirstHeading {
			continue
		}

		// If no frontmatter and we found a heading, collect paragraph
		// If we have frontmatter and passed it and the heading, collect paragraph
		canCollect := (!startsWithFrontmatter && foundHeading) ||
			(startsWithFrontmatter && pastFrontmatter && pastFirstHeading)

		if canCollect {
			// Stop at blank line (end of paragraph)
			if trimmed == "" {
				if paragraph.Len() > 0 {
					break
				}
				continue
			}

			// Stop at next heading
			if strings.HasPrefix(trimmed, "#") {
				break
			}

			// Add line to paragraph
			if paragraph.Len() > 0 {
				paragraph.WriteString("\n")
			}
			paragraph.WriteString(trimmed)

			// Stop if we hit 500 chars
			if paragraph.Len() >= 500 {
				break
			}
		}
	}

	result := paragraph.String()
	if len(result) > 500 {
		result = result[:500]
	}

	return result
}

// ExtractMetadata extracts task metadata using multi-source priority-based fallbacks
// Returns TaskMetadata and any warnings encountered during extraction
func ExtractMetadata(content, filename string, patternMatch *patterns.MatchResult) (*TaskMetadata, []string) {
	var warnings []string

	// Parse frontmatter first
	fm, err := ParseFrontmatter(content)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to parse frontmatter: %v", err))
		fm = &Frontmatter{HasFrontmatter: false}
	}

	metadata := &TaskMetadata{
		TaskKey:  fm.TaskKey,
		Status:   fm.Status,
		Priority: fm.Priority,
		Feature:  fm.Feature,
	}

	// Extract title with priority-based fallback
	// Priority 1: Frontmatter title
	if fm.Title != "" {
		metadata.Title = fm.Title
	} else {
		// Priority 2: Filename
		metadata.Title = ExtractTitleFromFilename(filename, patternMatch)

		if metadata.Title == "" {
			// Priority 3: H1 heading
			metadata.Title = ExtractTitleFromMarkdown(content)

			if metadata.Title == "" {
				// Use placeholder and log warning
				metadata.Title = "Untitled Task"
				warnings = append(warnings, fmt.Sprintf(
					"no title found for %s. Using default title. "+
						"Suggestion: Add title to frontmatter, filename, or H1 heading.",
					filepath.Base(filename)))
			}
		}
	}

	// Extract description with priority-based fallback
	// Priority 1: Frontmatter description
	if fm.Description != "" {
		metadata.Description = fm.Description
	} else {
		// Priority 2: First paragraph from markdown body
		metadata.Description = ExtractDescriptionFromMarkdown(content)
		// Note: Empty description is acceptable, no warning needed
	}

	return metadata, warnings
}

// toTitleCase converts a hyphen-separated string to Title Case
// Example: "implement-user-authentication" -> "Implement User Authentication"
func toTitleCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// H1PrefixPattern is a regex to match common H1 prefixes
var H1PrefixPattern = regexp.MustCompile(`^(?i)(task|prp|todo|wip):\s*`)

// RemoveH1Prefix removes common H1 prefixes from a title string
func RemoveH1Prefix(title string) string {
	return H1PrefixPattern.ReplaceAllString(title, "")
}
