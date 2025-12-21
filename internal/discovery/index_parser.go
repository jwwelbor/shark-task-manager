package discovery

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// IndexParser parses epic-index.md markdown files to extract epic and feature references
type IndexParser struct {
	// Compiled regex patterns for performance
	linkPattern    *regexp.Regexp
	epicPattern    *regexp.Regexp
	specialPattern *regexp.Regexp
	featurePattern *regexp.Regexp
}

// NewIndexParser creates a new IndexParser with compiled patterns
func NewIndexParser() *IndexParser {
	return &IndexParser{
		linkPattern:    regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`),
		epicPattern:    regexp.MustCompile(`^(E\d{2})-([a-z0-9-]+)$`),
		specialPattern: regexp.MustCompile(`^(tech-debt|bugs|change-cards)$`),
		featurePattern: regexp.MustCompile(`^(E\d{2})-(F\d{2})-`),
	}
}

// Parse reads epic-index.md and extracts epic and feature links
// Returns slices of IndexEpic and IndexFeature, or error if file cannot be read
func (p *IndexParser) Parse(indexPath string) ([]IndexEpic, []IndexFeature, error) {
	// Read file
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read index file: %w", err)
	}

	// Parse all markdown links
	matches := p.linkPattern.FindAllStringSubmatch(string(content), -1)

	epics := []IndexEpic{}
	features := []IndexFeature{}

	for _, match := range matches {
		if len(match) != 3 {
			// Malformed match, skip
			continue
		}

		linkText := match[1]
		linkPath := match[2]

		// Normalize path (remove leading ./ and /, remove trailing /)
		cleanPath := p.normalizePath(linkPath)

		// Skip external links (http://, https://, etc.)
		if strings.Contains(cleanPath, "://") {
			continue
		}

		// Skip file links (ending in .md, .txt, etc.)
		if strings.HasSuffix(cleanPath, ".md") || strings.HasSuffix(cleanPath, ".txt") {
			continue
		}

		// Count path segments to determine epic vs feature
		segments := strings.Split(cleanPath, "/")

		// Remove empty segments (from leading/trailing slashes)
		nonEmptySegments := []string{}
		for _, seg := range segments {
			if seg != "" {
				nonEmptySegments = append(nonEmptySegments, seg)
			}
		}

		if len(nonEmptySegments) == 1 {
			// Epic link: ./E04-epic-slug/
			epic, err := p.parseEpicLink(linkText, cleanPath)
			if err != nil {
				// Log warning but don't fail - skip invalid link
				continue
			}
			epics = append(epics, epic)
		} else if len(nonEmptySegments) == 2 {
			// Feature link: ./E04-epic-slug/E04-F01-feature-slug/
			feature, err := p.parseFeatureLink(linkText, cleanPath)
			if err != nil {
				// Log warning but don't fail - skip invalid link
				continue
			}
			features = append(features, feature)
		}
		// Ignore deeper paths (task links, document links)
	}

	return epics, features, nil
}

// normalizePath removes leading ./ or /, and trailing /
func (p *IndexParser) normalizePath(path string) string {
	// Remove leading ./
	cleanPath := strings.TrimPrefix(path, "./")

	// Remove leading /
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	// Remove trailing /
	cleanPath = strings.TrimSuffix(cleanPath, "/")

	return cleanPath
}

// parseEpicLink extracts epic metadata from markdown link
func (p *IndexParser) parseEpicLink(linkText, path string) (IndexEpic, error) {
	// Normalize path first
	cleanPath := p.normalizePath(path)

	// Try standard pattern: E##-slug
	matches := p.epicPattern.FindStringSubmatch(cleanPath)
	if len(matches) == 3 {
		return IndexEpic{
			Key:   matches[1], // E04
			Title: linkText,
			Path:  cleanPath,
		}, nil
	}

	// Try special type pattern: tech-debt, bugs, change-cards
	matches = p.specialPattern.FindStringSubmatch(cleanPath)
	if len(matches) == 2 {
		return IndexEpic{
			Key:   matches[1], // tech-debt
			Title: linkText,
			Path:  cleanPath,
		}, nil
	}

	return IndexEpic{}, fmt.Errorf("path does not match epic patterns: %s", path)
}

// parseFeatureLink extracts feature metadata from markdown link
func (p *IndexParser) parseFeatureLink(linkText, path string) (IndexFeature, error) {
	// Normalize path first
	cleanPath := p.normalizePath(path)

	// Path format: E04-epic-slug/E04-F01-feature-slug
	segments := strings.Split(cleanPath, "/")

	// Filter out empty segments
	nonEmptySegments := []string{}
	for _, seg := range segments {
		if seg != "" {
			nonEmptySegments = append(nonEmptySegments, seg)
		}
	}

	if len(nonEmptySegments) != 2 {
		return IndexFeature{}, fmt.Errorf("invalid feature path (must have 2 segments): %s", path)
	}

	epicFolder := nonEmptySegments[0]
	featureFolder := nonEmptySegments[1]

	// Extract epic key from epic folder
	epicMatches := p.epicPattern.FindStringSubmatch(epicFolder)
	if len(epicMatches) < 2 {
		return IndexFeature{}, fmt.Errorf("cannot extract epic key from: %s", epicFolder)
	}
	epicKey := epicMatches[1]

	// Extract feature key from feature folder
	// Pattern: E04-F01-feature-slug
	featureMatches := p.featurePattern.FindStringSubmatch(featureFolder)
	if len(featureMatches) < 3 {
		return IndexFeature{}, fmt.Errorf("cannot extract feature key from: %s", featureFolder)
	}

	// Build full feature key: E04-F01
	featureKey := featureMatches[1] + "-" + featureMatches[2] // E04-F01

	// Validate epic key matches between epic folder and feature folder
	featureEpicKey := featureMatches[1]
	if featureEpicKey != epicKey {
		return IndexFeature{}, fmt.Errorf("epic key mismatch: epic folder has %s but feature has %s", epicKey, featureEpicKey)
	}

	return IndexFeature{
		Key:     featureKey,
		EpicKey: epicKey,
		Title:   linkText,
		Path:    cleanPath,
	}, nil
}
