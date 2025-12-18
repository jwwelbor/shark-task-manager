package discovery

// Named capture group constants for pattern matching
const (
	// Epic pattern capture groups
	CaptureGroupEpicID   = "epic_id"   // Epic ID (e.g., "E04", "tech-debt")
	CaptureGroupEpicSlug = "epic_slug" // Epic slug (e.g., "task-mgmt-cli-core")
	CaptureGroupEpicNum  = "epic_num"  // Epic number (e.g., "04")

	// Feature pattern capture groups
	CaptureGroupFeatureID   = "feature_id"   // Feature ID (e.g., "F07")
	CaptureGroupFeatureSlug = "feature_slug" // Feature slug (e.g., "initialization-sync")
	CaptureGroupFeatureNum  = "feature_num"  // Feature number (e.g., "07")
)

// Default pattern definitions for epic and feature discovery
const (
	// DefaultEpicFolderPattern matches standard E##-epic-slug format
	DefaultEpicFolderPattern = `(?P<epic_id>E\d{2})-(?P<epic_slug>[a-z0-9-]+)`

	// DefaultEpicSpecialPattern matches special epic types (tech-debt, bugs, change-cards)
	DefaultEpicSpecialPattern = `(?P<epic_id>tech-debt|bugs|change-cards)`

	// DefaultFeatureFolderPattern matches E##-F##-feature-slug format
	DefaultFeatureFolderPattern = `(?P<epic_id>E(?P<epic_num>\d{2}))-(?P<feature_id>F(?P<feature_num>\d{2}))-(?P<feature_slug>[a-z0-9-]+)`

	// DefaultFeatureFolderShortPattern matches F##-feature-slug format (infer epic from parent)
	DefaultFeatureFolderShortPattern = `(?P<feature_id>F(?P<feature_num>\d{2}))-(?P<feature_slug>[a-z0-9-]+)`

	// DefaultFeatureFilePattern matches prd.md (highest priority)
	DefaultFeatureFilePattern = `^prd\.md$`

	// DefaultFeatureFileLongPattern matches PRD_F##-name.md format
	DefaultFeatureFileLongPattern = `^PRD_(?P<feature_id>F\d{2})-(?P<feature_slug>[a-z0-9-]+)\.md$`
)

// DefaultDocsRoot is the default documentation root directory
const DefaultDocsRoot = "docs/plan"

// DefaultIndexFileName is the default name for the epic index file
const DefaultIndexFileName = "epic-index.md"

// RelatedDocPatterns defines glob patterns for related documents
var RelatedDocPatterns = []string{
	"02-*.md",
	"03-*.md",
	"04-*.md",
	"05-*.md",
	"06-*.md",
	"07-*.md",
	"08-*.md",
	"09-*.md",
	"architecture.md",
	"database-design.md",
	"security-design.md",
	"performance-design.md",
	"test-criteria.md",
	"implementation-phases.md",
}

// ExcludedSubfolders lists subfolders to exclude from related document scanning
var ExcludedSubfolders = []string{
	"tasks",
	"prps",
}
