# Discovery Package

The `discovery` package provides foundational types, interfaces, and data structures for the epic/feature discovery system that coordinates dual-source discovery (epic-index.md + folder scanning).

## Package Contents

### Core Types (`types.go`)

- **DiscoveryOptions**: Configuration struct for discovery operations
- **DiscoveryReport**: Results of a discovery operation with detailed metrics
- **IndexEpic/IndexFeature**: Types for epics/features discovered from epic-index.md
- **FolderEpic/FolderFeature**: Types for epics/features discovered from folder scanning
- **DiscoveredEpic/DiscoveredFeature**: Merged results from both sources
- **Conflict**: Represents conflicts between index and folder structure

### Constants (`constants.go`)

- **Capture Group Names**: Named capture group constants for regex pattern matching
- **Default Patterns**: Regex patterns for epic/feature folder and file matching
- **Related Document Patterns**: Glob patterns for identifying related documentation
- **Excluded Subfolders**: Subfolders to exclude from scanning (tasks, prps)

## Enums and Constants

### ConflictStrategy

Defines how to resolve conflicts between epic-index.md and folder structure:

- `index-precedence`: epic-index.md is source of truth (default)
- `folder-precedence`: Folder structure is source of truth
- `merge`: Import from both sources

### ValidationLevel

Defines strictness of validation during discovery:

- `strict`: Requires exact E##-F## naming conventions
- `balanced`: Accepts patterns defined in config (default)
- `permissive`: Accepts any reasonable folder structure

### ConflictType

Types of conflicts that can be detected:

- `epic_index_only`: Epic in index but folder doesn't exist
- `epic_folder_only`: Epic folder exists but not in index
- `feature_index_only`: Feature in index but folder doesn't exist
- `feature_folder_only`: Feature folder exists but not in index
- `relationship_mismatch`: Feature in wrong epic folder

### DiscoverySource

Indicates where an epic/feature was discovered:

- `index`: Discovered from epic-index.md
- `folder`: Discovered from folder structure
- `merged`: Merged from both sources

## Usage Example

```go
import "github.com/jwwelbor/shark-task-manager/internal/discovery"

// Configure discovery options
opts := discovery.DiscoveryOptions{
    DocsRoot:        "docs/plan",
    IndexPath:       "docs/plan/epic-index.md",
    Strategy:        discovery.ConflictStrategyIndexPrecedence,
    ValidationLevel: discovery.ValidationLevelBalanced,
    DryRun:          false,
}

// Discovery report structure
report := discovery.DiscoveryReport{
    FoldersScanned:    47,
    EpicsDiscovered:   15,
    FeaturesDiscovered: 87,
    Conflicts:         []discovery.Conflict{},
}
```

## Pattern Matching

The package provides default regex patterns for matching epic and feature folders:

### Epic Patterns

- Standard: `E##-epic-slug` (e.g., "E04-task-mgmt-cli-core")
- Special: `tech-debt`, `bugs`, `change-cards`

### Feature Patterns

- Full: `E##-F##-feature-slug` (e.g., "E04-F07-initialization-sync")
- Short: `F##-feature-slug` (infer epic from parent folder)

### Feature File Patterns

- Primary: `prd.md`
- Secondary: `PRD_F##-feature-slug.md`

## JSON Serialization

All report types support JSON marshaling for CLI output and API integration:

```go
report := discovery.DiscoveryReport{...}
jsonData, err := json.Marshal(report)
```

## Testing

The package includes comprehensive unit tests:

- Type validation and JSON marshaling tests
- Pattern matching and regex validation tests
- Constant definition tests
- Enum value tests

Run tests:
```bash
go test ./internal/discovery/... -v
```

## Design Philosophy

This package follows the existing shark codebase conventions:

- JSON tags for all exported struct fields
- Pointer types for optional fields
- Clear, descriptive naming
- No external dependencies beyond standard library
- Comprehensive test coverage

## Related Documentation

- Feature PRD: `docs/plan/E06-intelligent-scanning/E06-F02-epic-feature-discovery/prd.md`
- Architecture: `docs/plan/E06-intelligent-scanning/E06-F02-epic-feature-discovery/02-architecture.md`
- Task: `docs/plan/E06-intelligent-scanning/E06-F02-epic-feature-discovery/tasks/T-E06-F02-001.md`
