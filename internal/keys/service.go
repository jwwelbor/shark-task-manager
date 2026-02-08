package keys

import (
	"fmt"
	"regexp"
	"strings"
)

// EntityType represents the type of entity identified from a key.
type EntityType string

const (
	// EntityTypeEpic identifies an epic key (e.g., E07, E07-user-management)
	EntityTypeEpic EntityType = "epic"

	// EntityTypeFeature identifies a feature key (e.g., E07-F01, F01, E07-F01-auth)
	EntityTypeFeature EntityType = "feature"

	// EntityTypeTask identifies a task key (e.g., T-E07-F01-001, E07-F01-001)
	EntityTypeTask EntityType = "task"

	// EntityTypeUnknown indicates the key format could not be recognized
	EntityTypeUnknown EntityType = "unknown"
)

// ParsedKey contains the decomposed components of an entity key.
type ParsedKey struct {
	// Raw is the original input string before any processing
	Raw string

	// Normalized is the canonical uppercase form without slug.
	// For tasks, this includes the T- prefix.
	// Examples: "E07", "E07-F01", "T-E07-F01-001"
	Normalized string

	// EntityType is the detected entity type
	EntityType EntityType

	// EpicNum is the numeric part of the epic key (e.g., "07" from E07).
	// Empty for feature-suffix-only keys like "F01".
	EpicNum string

	// FeatureNum is the numeric part of the feature key (e.g., "01" from F01).
	// Empty for epic-only keys.
	FeatureNum string

	// TaskNum is the numeric part of the task number (e.g., "001").
	// Empty for non-task keys.
	TaskNum string

	// Slug is the optional human-readable suffix (e.g., "user-management" from E07-user-management).
	// Empty if no slug is present. Stored in lowercase.
	Slug string
}

// Compiled regex patterns for slug-aware key parsing.
// These extend the base patterns in validation.go with optional slug support.
var (
	// epicSlugPattern matches E## with optional slug: E07, E07-user-management
	epicSlugPattern = regexp.MustCompile(`^E(\d{2})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`)

	// featureFullSlugPattern matches E##-F## with optional slug: E07-F01, E07-F01-auth-module
	featureFullSlugPattern = regexp.MustCompile(`^E(\d{2})-F(\d{2})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`)

	// featureSuffixSlugPattern matches F## with optional slug: F01, F01-some-feature
	featureSuffixSlugPattern = regexp.MustCompile(`^F(\d{2})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`)

	// taskFullSlugPattern matches T-E##-F##-### with optional slug: T-E07-F01-001, T-E07-F01-001-impl-jwt
	taskFullSlugPattern = regexp.MustCompile(`^T-E(\d{2})-F(\d{2})-(\d{3})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`)

	// taskShortSlugPattern matches E##-F##-### with optional slug: E07-F01-001, E07-F01-001-impl-jwt
	taskShortSlugPattern = regexp.MustCompile(`^E(\d{2})-F(\d{2})-(\d{3})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`)
)

// KeyService provides centralized entity key parsing and normalization.
// It consolidates all key-related logic that was previously scattered across
// validation.go, helper functions in CLI commands, and the scope interpreter.
type KeyService struct{}

// NewKeyService creates a new KeyService instance.
func NewKeyService() *KeyService {
	return &KeyService{}
}

// Parse detects the entity type and extracts all components from any key format.
// It handles all supported formats including slugged keys.
//
// Supported formats:
//
//	Epic:    E07, e07, E07-user-management
//	Feature: E07-F01, F01, e07-f01, E07-F01-auth, F01-my-feature
//	Task:    T-E07-F01-001, E07-F01-001, e07-f01-001, E07-F01-001-task-name
//
// For unrecognized formats, returns a ParsedKey with EntityType == EntityTypeUnknown.
func (ks *KeyService) Parse(key string) ParsedKey {
	result := ParsedKey{
		Raw:        key,
		EntityType: EntityTypeUnknown,
	}

	if key == "" {
		return result
	}

	upper := strings.ToUpper(key)

	// Try task patterns first (most specific - avoids misidentifying E07-F01-001 as feature)

	// Full task key: T-E##-F##-### with optional slug
	if m := taskFullSlugPattern.FindStringSubmatch(upper); m != nil {
		result.EntityType = EntityTypeTask
		result.EpicNum = m[1]
		result.FeatureNum = m[2]
		result.TaskNum = m[3]
		result.Slug = toLowerSlug(m[4])
		result.Normalized = fmt.Sprintf("T-E%s-F%s-%s", m[1], m[2], m[3])
		return result
	}

	// Short task key: E##-F##-### with optional slug
	if m := taskShortSlugPattern.FindStringSubmatch(upper); m != nil {
		result.EntityType = EntityTypeTask
		result.EpicNum = m[1]
		result.FeatureNum = m[2]
		result.TaskNum = m[3]
		result.Slug = toLowerSlug(m[4])
		result.Normalized = fmt.Sprintf("T-E%s-F%s-%s", m[1], m[2], m[3])
		return result
	}

	// Feature full key: E##-F## with optional slug
	if m := featureFullSlugPattern.FindStringSubmatch(upper); m != nil {
		result.EntityType = EntityTypeFeature
		result.EpicNum = m[1]
		result.FeatureNum = m[2]
		result.Slug = toLowerSlug(m[3])
		result.Normalized = fmt.Sprintf("E%s-F%s", m[1], m[2])
		return result
	}

	// Feature suffix: F## with optional slug
	if m := featureSuffixSlugPattern.FindStringSubmatch(upper); m != nil {
		result.EntityType = EntityTypeFeature
		result.FeatureNum = m[1]
		result.Slug = toLowerSlug(m[2])
		result.Normalized = fmt.Sprintf("F%s", m[1])
		return result
	}

	// Epic key: E## with optional slug
	if m := epicSlugPattern.FindStringSubmatch(upper); m != nil {
		// Guard against matching feature suffixes that start with F##
		// The regex already handles this since it requires E prefix, but double check
		// that the slug doesn't accidentally match a feature pattern
		result.EntityType = EntityTypeEpic
		result.EpicNum = m[1]
		result.Slug = toLowerSlug(m[2])
		result.Normalized = fmt.Sprintf("E%s", m[1])
		return result
	}

	return result
}

// DetectEntityType returns the entity type for a key without full parsing.
// This is a convenience method equivalent to Parse(key).EntityType.
//
// Returns EntityTypeUnknown for unrecognized formats.
func (ks *KeyService) DetectEntityType(key string) EntityType {
	return ks.Parse(key).EntityType
}

// Normalize converts any key to its canonical uppercase form without slug.
// For task keys, the T- prefix is always included.
//
// Examples:
//
//	e07              -> E07
//	e07-f01          -> E07-F01
//	e07-f01-001      -> T-E07-F01-001
//	E07-F01-001-slug -> T-E07-F01-001
//	e07-user-mgmt    -> E07
//	f01              -> F01
//	hello            -> HELLO (unknown keys are uppercased)
//	""               -> ""
func (ks *KeyService) Normalize(key string) string {
	if key == "" {
		return ""
	}

	parsed := ks.Parse(key)
	if parsed.EntityType == EntityTypeUnknown {
		// For unknown keys, just uppercase the whole thing
		return strings.ToUpper(key)
	}
	return parsed.Normalized
}

// IsValid checks if a key is valid for any entity type.
// Returns true for epic, feature, or task keys (including slugged variants).
func (ks *KeyService) IsValid(key string) bool {
	return ks.Parse(key).EntityType != EntityTypeUnknown
}

// Format formats a ParsedKey back into the canonical string form.
// The result contains only the numeric parts (no slug).
//
// Examples:
//
//	{EntityTypeEpic, EpicNum: "07"}                           -> "E07"
//	{EntityTypeFeature, EpicNum: "07", FeatureNum: "01"}      -> "E07-F01"
//	{EntityTypeFeature, FeatureNum: "01"}                     -> "F01"
//	{EntityTypeTask, EpicNum: "07", FeatureNum: "01", TaskNum: "001"} -> "T-E07-F01-001"
//	{EntityTypeUnknown}                                       -> ""
func (ks *KeyService) Format(parsed ParsedKey) string {
	switch parsed.EntityType {
	case EntityTypeEpic:
		return fmt.Sprintf("E%s", parsed.EpicNum)

	case EntityTypeFeature:
		if parsed.EpicNum != "" {
			return fmt.Sprintf("E%s-F%s", parsed.EpicNum, parsed.FeatureNum)
		}
		return fmt.Sprintf("F%s", parsed.FeatureNum)

	case EntityTypeTask:
		return fmt.Sprintf("T-E%s-F%s-%s", parsed.EpicNum, parsed.FeatureNum, parsed.TaskNum)

	default:
		return ""
	}
}

// NormalizeTaskKey handles both T-E##-F##-### and E##-F##-### formats,
// returning the canonical T-prefixed form without any slug.
// For non-task keys or invalid input, returns the input uppercased.
//
// Examples:
//
//	T-E07-F01-001          -> T-E07-F01-001
//	E07-F01-001            -> T-E07-F01-001
//	e07-f01-001            -> T-E07-F01-001
//	E07-F01-001-task-name  -> T-E07-F01-001
//	T-E07-F01-001-slug     -> T-E07-F01-001
func (ks *KeyService) NormalizeTaskKey(key string) string {
	parsed := ks.Parse(key)
	if parsed.EntityType == EntityTypeTask {
		return parsed.Normalized
	}
	// Fallback: return uppercased input
	return strings.ToUpper(key)
}

// toLowerSlug converts a slug to lowercase, or returns empty string for empty input.
func toLowerSlug(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s)
}
