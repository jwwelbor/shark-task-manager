package discovery

import (
	"regexp"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

// PatternMatcher handles pattern matching for epics and features
type PatternMatcher struct {
	config *patterns.PatternConfig

	// Cached compiled regexes
	epicFolderRegexes    []*regexp.Regexp
	featureFolderRegexes []*regexp.Regexp
	featureFileRegexes   []*regexp.Regexp

	// Mutex for lazy initialization
	mu sync.Once
}

// NewPatternMatcher creates a new pattern matcher with given configuration
func NewPatternMatcher(config *patterns.PatternConfig) *PatternMatcher {
	matcher := &PatternMatcher{
		config: config,
	}
	matcher.compilePatterns()
	return matcher
}

// compilePatterns compiles all regex patterns once for performance
func (m *PatternMatcher) compilePatterns() {
	m.mu.Do(func() {
		// Compile epic folder patterns
		m.epicFolderRegexes = make([]*regexp.Regexp, 0, len(m.config.Epic.Folder))
		for _, pattern := range m.config.Epic.Folder {
			re, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid patterns (log warning in production)
				continue
			}
			m.epicFolderRegexes = append(m.epicFolderRegexes, re)
		}

		// Compile feature folder patterns
		m.featureFolderRegexes = make([]*regexp.Regexp, 0, len(m.config.Feature.Folder))
		for _, pattern := range m.config.Feature.Folder {
			re, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid patterns
				continue
			}
			m.featureFolderRegexes = append(m.featureFolderRegexes, re)
		}

		// Compile feature file patterns
		m.featureFileRegexes = make([]*regexp.Regexp, 0, len(m.config.Feature.File))
		for _, pattern := range m.config.Feature.File {
			re, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid patterns
				continue
			}
			m.featureFileRegexes = append(m.featureFileRegexes, re)
		}
	})
}

// EpicMatchResult contains extracted components from epic pattern match
type EpicMatchResult struct {
	EpicID   string
	EpicSlug string
	EpicNum  string
}

// FeatureMatchResult contains extracted components from feature pattern match
type FeatureMatchResult struct {
	EpicID      string
	FeatureID   string
	FeatureSlug string
	EpicNum     string
	FeatureNum  string
}

// MatchEpicPattern tries to match input against epic patterns (first match wins)
func (m *PatternMatcher) MatchEpicPattern(input string) (EpicMatchResult, bool) {
	for _, re := range m.epicFolderRegexes {
		matches := re.FindStringSubmatch(input)
		if matches == nil {
			continue
		}

		// Extract named capture groups
		result := EpicMatchResult{}
		names := re.SubexpNames()

		for i, name := range names {
			if i == 0 || name == "" {
				continue
			}
			if i < len(matches) {
				switch name {
				case "epic_id":
					result.EpicID = matches[i]
				case "epic_slug", "slug":
					result.EpicSlug = matches[i]
				case "epic_num", "number":
					result.EpicNum = matches[i]
				}
			}
		}

		// Build EpicID from number if not set by epic_id
		if result.EpicID == "" && result.EpicNum != "" {
			result.EpicID = "E" + result.EpicNum
		}

		// Validate required fields
		if result.EpicID == "" {
			continue
		}

		return result, true
	}

	return EpicMatchResult{}, false
}

// MatchFeaturePattern tries to match input against feature patterns (first match wins)
func (m *PatternMatcher) MatchFeaturePattern(input, parentEpicKey string) (FeatureMatchResult, bool) {
	for _, re := range m.featureFolderRegexes {
		matches := re.FindStringSubmatch(input)
		if matches == nil {
			continue
		}

		// Extract named capture groups
		result := FeatureMatchResult{}
		names := re.SubexpNames()

		for i, name := range names {
			if i == 0 || name == "" {
				continue
			}
			if i < len(matches) {
				switch name {
				case "epic_id":
					result.EpicID = matches[i]
				case "feature_id":
					result.FeatureID = matches[i]
				case "feature_slug", "slug":
					result.FeatureSlug = matches[i]
				case "epic_num":
					result.EpicNum = matches[i]
				case "feature_num", "number":
					result.FeatureNum = matches[i]
				}
			}
		}

		// Build EpicID from epic_num if not set
		if result.EpicID == "" && result.EpicNum != "" {
			result.EpicID = "E" + result.EpicNum
		}

		// Build FeatureID from number if not set
		if result.FeatureID == "" && result.FeatureNum != "" {
			result.FeatureID = "F" + result.FeatureNum
		}

		// If epic_id still not captured, use parent epic key
		if result.EpicID == "" {
			result.EpicID = parentEpicKey
		}

		// Validate required fields
		if result.EpicID == "" || result.FeatureID == "" {
			continue
		}

		return result, true
	}

	return FeatureMatchResult{}, false
}

// MatchFeatureFilePattern checks if filename matches any feature file pattern
func (m *PatternMatcher) MatchFeatureFilePattern(filename string) bool {
	for _, re := range m.featureFileRegexes {
		if re.MatchString(filename) {
			return true
		}
	}
	return false
}
