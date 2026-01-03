package discovery

// DiscoveryOptions contains configuration for discovery operations
type DiscoveryOptions struct {
	DocsRoot        string           // Root directory (e.g., "docs/plan")
	IndexPath       string           // Path to epic-index.md (optional)
	Strategy        ConflictStrategy // Conflict resolution strategy
	DryRun          bool             // Preview without database changes
	ValidationLevel ValidationLevel  // Validation strictness level
}

// DiscoveryReport contains the results of a discovery operation
type DiscoveryReport struct {
	FoldersScanned       int        `json:"folders_scanned"`
	FilesAnalyzed        int        `json:"files_analyzed"`
	EpicsDiscovered      int        `json:"epics_discovered"`
	EpicsFromIndex       int        `json:"epics_from_index"`
	EpicsFromFolders     int        `json:"epics_from_folders"`
	FeaturesDiscovered   int        `json:"features_discovered"`
	FeaturesFromIndex    int        `json:"features_from_index"`
	FeaturesFromFolders  int        `json:"features_from_folders"`
	RelatedDocsCataloged int        `json:"related_docs_cataloged"`
	ConflictsDetected    int        `json:"conflicts_detected"`
	Conflicts            []Conflict `json:"conflicts"`
	Warnings             []string   `json:"warnings"`
	Errors               []string   `json:"errors"`
}

// ConflictType defines the type of conflict detected
type ConflictType string

const (
	// ConflictTypeEpicIndexOnly indicates epic is in index but folder doesn't exist
	ConflictTypeEpicIndexOnly ConflictType = "epic_index_only"

	// ConflictTypeEpicFolderOnly indicates epic folder exists but not in index
	ConflictTypeEpicFolderOnly ConflictType = "epic_folder_only"

	// ConflictTypeFeatureIndexOnly indicates feature is in index but folder doesn't exist
	ConflictTypeFeatureIndexOnly ConflictType = "feature_index_only"

	// ConflictTypeFeatureFolderOnly indicates feature folder exists but not in index
	ConflictTypeFeatureFolderOnly ConflictType = "feature_folder_only"

	// ConflictTypeRelationshipMismatch indicates feature is in wrong epic folder
	ConflictTypeRelationshipMismatch ConflictType = "relationship_mismatch"
)

// Conflict represents a detected difference between index and folder structure
type Conflict struct {
	Type       ConflictType `json:"type"`       // Type of conflict
	Key        string       `json:"key"`        // Epic or feature key
	Path       string       `json:"path"`       // File path involved
	Resolution string       `json:"resolution"` // How conflict was resolved
	Strategy   string       `json:"strategy"`   // Strategy applied
	Suggestion string       `json:"suggestion"` // Actionable suggestion for user
}

// ConflictStrategy defines how to resolve conflicts between index and folders
type ConflictStrategy string

const (
	// ConflictStrategyIndexPrecedence uses epic-index.md as source of truth
	ConflictStrategyIndexPrecedence ConflictStrategy = "index-precedence"

	// ConflictStrategyFolderPrecedence uses folder structure as source of truth
	ConflictStrategyFolderPrecedence ConflictStrategy = "folder-precedence"

	// ConflictStrategyMerge imports from both index and folders
	ConflictStrategyMerge ConflictStrategy = "merge"
)

// ValidationLevel defines strictness of validation during discovery
type ValidationLevel string

const (
	// ValidationLevelStrict requires exact E##-F## naming conventions
	ValidationLevelStrict ValidationLevel = "strict"

	// ValidationLevelBalanced accepts patterns defined in config (default)
	ValidationLevelBalanced ValidationLevel = "balanced"

	// ValidationLevelPermissive accepts any reasonable folder structure
	ValidationLevelPermissive ValidationLevel = "permissive"
)

// DiscoverySource indicates where an epic/feature was discovered from
type DiscoverySource string

const (
	// SourceIndex indicates discovered from epic-index.md
	SourceIndex DiscoverySource = "index"

	// SourceFolder indicates discovered from folder structure
	SourceFolder DiscoverySource = "folder"

	// SourceMerged indicates merged from both sources
	SourceMerged DiscoverySource = "merged"
)

// IndexEpic represents an epic discovered from epic-index.md
type IndexEpic struct {
	Key      string         `json:"key"`      // Extracted from path (e.g., "E04")
	Title    string         `json:"title"`    // Extracted from link text
	Path     string         `json:"path"`     // Relative path from link
	Features []IndexFeature `json:"features"` // Features listed under this epic
}

// IndexFeature represents a feature discovered from epic-index.md
type IndexFeature struct {
	Key     string `json:"key"`      // Extracted from path (e.g., "E04-F07")
	EpicKey string `json:"epic_key"` // Parent epic key
	Title   string `json:"title"`    // Extracted from link text
	Path    string `json:"path"`     // Relative path from link
}

// FolderEpic represents an epic discovered from folder structure
type FolderEpic struct {
	Key        string          `json:"key"`          // Extracted from pattern (e.g., "E04")
	Slug       string          `json:"slug"`         // Extracted from pattern
	Path       string          `json:"path"`         // Full folder path
	EpicMdPath *string         `json:"epic_md_path"` // Path to epic.md if exists
	Features   []FolderFeature `json:"features"`     // Features discovered in this epic
}

// FolderFeature represents a feature discovered from folder structure
type FolderFeature struct {
	Key         string   `json:"key"`          // Extracted from pattern (e.g., "E04-F07")
	EpicKey     string   `json:"epic_key"`     // Parent epic key (from path)
	Slug        string   `json:"slug"`         // Extracted from pattern
	Path        string   `json:"path"`         // Full folder path
	PrdPath     *string  `json:"prd_path"`     // Path to prd.md (or PRD_F##-name.md)
	RelatedDocs []string `json:"related_docs"` // Paths to related documents
}

// DiscoveredEpic represents a merged epic from both sources
type DiscoveredEpic struct {
	Key         string              `json:"key"`         // Epic key (e.g., "E04")
	Title       string              `json:"title"`       // Epic title (from metadata)
	Description *string             `json:"description"` // Epic description
	FilePath    *string             `json:"file_path"`   // Path to epic.md
	Source      DiscoverySource     `json:"source"`      // Where it was discovered
	Features    []DiscoveredFeature `json:"features"`    // Features in this epic
}

// DiscoveredFeature represents a merged feature from both sources
type DiscoveredFeature struct {
	Key         string          `json:"key"`          // Feature key (e.g., "E04-F07")
	EpicKey     string          `json:"epic_key"`     // Parent epic key
	Title       string          `json:"title"`        // Feature title (from metadata)
	Description *string         `json:"description"`  // Feature description
	FilePath    *string         `json:"file_path"`    // Path to prd.md
	RelatedDocs []string        `json:"related_docs"` // Paths to related documents
	Source      DiscoverySource `json:"source"`       // Where it was discovered
}
