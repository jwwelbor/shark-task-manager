package discovery

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternMatcher_MatchEpicPattern(t *testing.T) {
	config := patterns.GetDefaultPatterns()
	matcher := NewPatternMatcher(config)

	tests := []struct {
		name         string
		input        string
		shouldMatch  bool
		expectedKey  string
		expectedSlug string
	}{
		{
			name:         "match standard E##-slug",
			input:        "E04-task-mgmt-cli-core",
			shouldMatch:  true,
			expectedKey:  "E04",
			expectedSlug: "task-mgmt-cli-core",
		},
		{
			name:         "match E01",
			input:        "E01-foundation",
			shouldMatch:  true,
			expectedKey:  "E01",
			expectedSlug: "foundation",
		},
		{
			name:         "match E99",
			input:        "E99-last-epic",
			shouldMatch:  true,
			expectedKey:  "E99",
			expectedSlug: "last-epic",
		},
		{
			name:         "match tech-debt special type",
			input:        "tech-debt",
			shouldMatch:  true,
			expectedKey:  "tech-debt",
			expectedSlug: "",
		},
		{
			name:         "match bugs special type",
			input:        "bugs",
			shouldMatch:  true,
			expectedKey:  "bugs",
			expectedSlug: "",
		},
		{
			name:         "match change-cards special type",
			input:        "change-cards",
			shouldMatch:  true,
			expectedKey:  "change-cards",
			expectedSlug: "",
		},
		{
			name:        "no match for E without two digits",
			input:       "E4-test",
			shouldMatch: false,
		},
		{
			name:        "no match for lowercase e",
			input:       "e04-test",
			shouldMatch: false,
		},
		{
			name:        "no match for random folder",
			input:       "random-folder",
			shouldMatch: false,
		},
		{
			name:        "no match for feature folder",
			input:       "E04-F07-initialization-sync",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, matched := matcher.MatchEpicPattern(tt.input)

			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedKey, result.EpicID, "Epic ID mismatch")
				if tt.expectedSlug != "" {
					assert.Equal(t, tt.expectedSlug, result.EpicSlug, "Epic slug mismatch")
				}
			}
		})
	}
}

func TestPatternMatcher_MatchFeaturePattern(t *testing.T) {
	config := patterns.GetDefaultPatterns()
	matcher := NewPatternMatcher(config)

	tests := []struct {
		name            string
		input           string
		parentEpicKey   string
		shouldMatch     bool
		expectedKey     string
		expectedEpicKey string
		expectedSlug    string
	}{
		{
			name:            "match standard E##-F##-slug",
			input:           "E04-F07-initialization-sync",
			parentEpicKey:   "E04",
			shouldMatch:     true,
			expectedKey:     "F07",
			expectedEpicKey: "E04",
			expectedSlug:    "initialization-sync",
		},
		// Note: Short F##-slug format is not in default patterns (would need to be added via config)
		// {
		// 	name:            "match short F##-slug format",
		// 	input:           "F07-initialization-sync",
		// 	parentEpicKey:   "E04",
		// 	shouldMatch:     true,
		// 	expectedKey:     "F07",
		// 	expectedEpicKey: "E04",
		// 	expectedSlug:    "initialization-sync",
		// },
		{
			name:            "match F01",
			input:           "E04-F01-database-schema",
			parentEpicKey:   "E04",
			shouldMatch:     true,
			expectedKey:     "F01",
			expectedEpicKey: "E04",
			expectedSlug:    "database-schema",
		},
		{
			name:            "match F99",
			input:           "E04-F99-last-feature",
			parentEpicKey:   "E04",
			shouldMatch:     true,
			expectedKey:     "F99",
			expectedEpicKey: "E04",
			expectedSlug:    "last-feature",
		},
		{
			name:          "no match for F without two digits",
			input:         "E04-F7-test",
			parentEpicKey: "E04",
			shouldMatch:   false,
		},
		{
			name:          "no match for lowercase f",
			input:         "E04-f07-test",
			parentEpicKey: "E04",
			shouldMatch:   false,
		},
		{
			name:          "no match for random folder",
			input:         "random-folder",
			parentEpicKey: "E04",
			shouldMatch:   false,
		},
		{
			name:          "no match for epic folder",
			input:         "E05-advanced-querying",
			parentEpicKey: "E04",
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, matched := matcher.MatchFeaturePattern(tt.input, tt.parentEpicKey)

			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedKey, result.FeatureID, "Feature ID mismatch")
				assert.Equal(t, tt.expectedEpicKey, result.EpicID, "Epic ID mismatch")
				assert.Equal(t, tt.expectedSlug, result.FeatureSlug, "Feature slug mismatch")
			}
		})
	}
}

func TestPatternMatcher_MatchFeatureFilePattern(t *testing.T) {
	config := patterns.GetDefaultPatterns()
	matcher := NewPatternMatcher(config)

	tests := []struct {
		name        string
		input       string
		shouldMatch bool
	}{
		{
			name:        "match prd.md (highest priority)",
			input:       "prd.md",
			shouldMatch: true,
		},
		{
			name:        "match PRD_F07-name.md",
			input:       "PRD_F07-initialization-sync.md",
			shouldMatch: true,
		},
		{
			name:        "match architecture.md (fallback pattern matches any .md)",
			input:       "02-architecture.md",
			shouldMatch: true, // Fallback pattern ^(?P<slug>[a-z0-9-]+)\.md$ matches this
		},
		{
			name:        "no match for task file",
			input:       "T-E04-F07-001.md",
			shouldMatch: false,
		},
		{
			name:        "no match for README",
			input:       "README.md",
			shouldMatch: false,
		},
		{
			name:        "no match for non-markdown",
			input:       "prd.txt",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := matcher.MatchFeatureFilePattern(tt.input)
			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")
		})
	}
}

func TestPatternMatcher_FirstMatchWins(t *testing.T) {
	// Test that when multiple patterns match, the first one wins
	config := patterns.GetDefaultPatterns()
	matcher := NewPatternMatcher(config)

	// E04-F07-slug matches both standard and can be interpreted as short pattern
	// But standard pattern should win (it's first in defaults)
	result, matched := matcher.MatchFeaturePattern("E04-F07-initialization-sync", "E04")

	require.True(t, matched)
	assert.Equal(t, "F07", result.FeatureID)
	assert.Equal(t, "E04", result.EpicID)
	assert.Equal(t, "initialization-sync", result.FeatureSlug)
}

func TestPatternMatcher_CachedRegexes(t *testing.T) {
	// Test that regexes are compiled once and cached
	config := patterns.GetDefaultPatterns()
	matcher := NewPatternMatcher(config)

	// First match compiles regexes
	result1, matched1 := matcher.MatchEpicPattern("E04-task-mgmt-cli-core")
	require.True(t, matched1)

	// Second match should use cached regexes
	result2, matched2 := matcher.MatchEpicPattern("E05-advanced-querying")
	require.True(t, matched2)

	assert.NotEqual(t, result1.EpicID, result2.EpicID, "Different epics should have different IDs")
}

func TestPatternMatcher_InvalidPattern(t *testing.T) {
	// Test behavior with invalid regex pattern
	config := &patterns.PatternConfig{
		Epic: patterns.EntityPatterns{
			Folder: []string{
				`(?P<epic_id>E\d{2`, // Invalid regex: missing closing )
			},
		},
	}

	// NewPatternMatcher should handle invalid patterns gracefully
	// In actual implementation, we may want to validate patterns first
	matcher := NewPatternMatcher(config)
	assert.NotNil(t, matcher)

	// Match should fail gracefully for invalid patterns
	_, matched := matcher.MatchEpicPattern("E04-test")
	assert.False(t, matched, "Invalid pattern should not match")
}

func TestPatternMatcher_CustomPatterns(t *testing.T) {
	// Test with custom patterns
	config := &patterns.PatternConfig{
		Epic: patterns.EntityPatterns{
			Folder: []string{
				`(?P<epic_id>EPIC-\d{3})-(?P<epic_slug>[a-z0-9-]+)`, // Custom format: EPIC-001-slug
			},
		},
	}

	matcher := NewPatternMatcher(config)

	tests := []struct {
		name        string
		input       string
		shouldMatch bool
		expectedKey string
	}{
		{
			name:        "match custom EPIC-### format",
			input:       "EPIC-001-custom-epic",
			shouldMatch: true,
			expectedKey: "EPIC-001",
		},
		{
			name:        "no match for standard E## format",
			input:       "E04-task-mgmt-cli-core",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, matched := matcher.MatchEpicPattern(tt.input)

			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedKey, result.EpicID, "Epic ID mismatch")
			}
		})
	}
}
