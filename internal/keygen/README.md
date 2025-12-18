# Package keygen

The `keygen` package provides automatic task key generation for PRP (Product Requirement Prompt) files that lack explicit task keys. It handles path parsing, database sequence querying, key formatting, and atomic frontmatter updates.

## Features

- **Path Parsing**: Extracts epic and feature keys from file directory structure
- **Sequence Generation**: Queries database for next available task sequence number
- **Key Formatting**: Generates properly formatted task keys (T-E##-F##-###)
- **Frontmatter Updates**: Atomically writes task_key to file frontmatter
- **Validation**: Detects orphaned files (missing epic/feature in database)
- **Error Handling**: Provides clear, actionable error messages

## Package Structure

```
internal/keygen/
├── path_parser.go           # File path parsing and epic/feature extraction
├── path_parser_test.go      # Path parser unit tests
├── frontmatter_writer.go    # Atomic YAML frontmatter updates
├── frontmatter_writer_test.go # Frontmatter writer unit tests
├── generator.go             # Main task key generation orchestration
├── generator_test.go        # Generator unit tests
├── integration_test.go      # End-to-end integration tests
└── README.md               # This file
```

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/keygen"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Create generator
generator := keygen.NewTaskKeyGenerator(
    taskRepo,
    featureRepo,
    epicRepo,
    "/path/to/project/root",
)

// Generate key for a file
ctx := context.Background()
result, err := generator.GenerateKeyForFile(ctx, "/path/to/docs/plan/E04-epic/E04-F02-feature/tasks/file.prp.md")
if err != nil {
    log.Fatalf("Failed to generate key: %v", err)
}

fmt.Printf("Generated key: %s\n", result.TaskKey)
fmt.Printf("Epic: %s, Feature: %s\n", result.EpicKey, result.FeatureKey)
fmt.Printf("Written to file: %v\n", result.WrittenToFile)
```

### Path Parsing

```go
parser := keygen.NewPathParser("/project/root")

components, err := parser.ParsePath("/project/root/docs/plan/E04-epic/E04-F02-feature/tasks/file.prp.md")
if err != nil {
    log.Fatalf("Failed to parse path: %v", err)
}

fmt.Printf("Epic: %s\n", components.EpicKey)        // E04
fmt.Printf("Feature: %s\n", components.FeatureKey)  // E04-F02
```

### Frontmatter Writing

```go
writer := keygen.NewFrontmatterWriter()

// Write task key to file
err := writer.WriteTaskKey("/path/to/file.md", "T-E04-F02-001")
if err != nil {
    log.Fatalf("Failed to write task key: %v", err)
}

// Check if file has task key
hasKey, taskKey, err := writer.HasTaskKey("/path/to/file.md")
if err != nil {
    log.Fatalf("Failed to check task key: %v", err)
}

if hasKey {
    fmt.Printf("File has task key: %s\n", taskKey)
}
```

### File Validation

```go
// Validate file before generation
err := generator.ValidateFile(ctx, "/path/to/file.prp.md")
if err != nil {
    fmt.Printf("Validation failed: %v\n", err)
    // Handle validation error
}
```

## Expected Directory Structure

The path parser expects files to be organized as follows:

```
{project_root}/
└── docs/
    └── plan/
        └── {epic_folder}/           # E##-epic-slug
            └── {feature_folder}/    # E##-F##-feature-slug or E##-P##-F##-feature-slug
                └── tasks/           # or prps/
                    └── file.prp.md
```

Examples:
- `docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/auth.prp.md`
- `docs/plan/E09-game-engine/E09-P02-F01-character-mgmt/tasks/create-character.prp.md`

## Task Key Format

Generated task keys follow the format: `T-<epic>-<feature>-<sequence>`

Where:
- `<epic>`: Epic key (E##)
- `<feature>`: Feature key (F## or P##-F##)
- `<sequence>`: Zero-padded 3-digit sequence number (001-999)

Examples:
- `T-E04-F02-001` - First task in feature E04-F02
- `T-E04-F02-015` - Fifteenth task in feature E04-F02
- `T-E09-P02-F01-003` - Third task in project feature E09-P02-F01

## Error Handling

### Orphaned Files

If a file references an epic or feature that doesn't exist in the database:

```
orphaned file: feature 'E04-F99' not found in database.
Suggestion: Create feature 'E04-F99' first or move file to existing feature folder
```

### Invalid Paths

If a file path doesn't match the expected structure:

```
cannot infer epic/feature from path '/random/path/file.md':
expected directory structure like docs/plan/{E##-epic-slug}/{E##-F##-feature-slug}/tasks/{file}
```

### Frontmatter Write Failures

If the frontmatter cannot be written (permissions, disk full, etc.):

```go
result, err := generator.GenerateKeyForFile(ctx, filePath)
if err == nil && result.Error != nil {
    // Key was generated but not written to file
    log.Printf("Warning: %v", result.Error)
    // Can still use result.TaskKey for current sync
}
```

## Atomic File Updates

The frontmatter writer uses atomic file operations to ensure data integrity:

1. **Read**: Read current file contents
2. **Update**: Parse and update frontmatter with task_key
3. **Write Temp**: Write updated content to temporary file
4. **Rename**: Atomically rename temp file to original (POSIX guarantee)

This ensures:
- No partial writes
- No data corruption
- No temp files left behind on success
- Original file preserved on error

## Performance

Typical performance per file:
- Path parsing: < 1ms
- Database query: < 5ms (with indexes)
- Frontmatter update: < 10ms
- **Total: ~15ms per file**

Batch optimization is possible when processing multiple files from the same feature (query max sequence once).

## Testing

Run all tests:
```bash
go test ./internal/keygen/... -v
```

Run with coverage:
```bash
go test ./internal/keygen/... -cover
```

Run integration tests only:
```bash
go test ./internal/keygen/... -v -run TestEndToEnd
```

## Dependencies

- `github.com/jwwelbor/shark-task-manager/internal/models` - Data models
- `github.com/jwwelbor/shark-task-manager/internal/repository` - Database repositories
- `gopkg.in/yaml.v3` - YAML parsing for frontmatter

## Integration with Sync Engine

The `keygen` package integrates with the sync engine through `internal/sync/keygen_integration.go`:

```go
// In sync engine
keyGen := sync.NewKeyGenerator(taskRepo, featureRepo, epicRepo, docsRoot)

// Generate key for file
taskKey, err := keyGen.GenerateKeyForFile(ctx, filePath)
if err != nil {
    // Handle error
}

// Use generated key for task import
taskFile.Metadata.TaskKey = taskKey
```

## Thread Safety

The package is designed for concurrent use:
- Path parsing is stateless (thread-safe)
- Frontmatter writing uses atomic file operations
- Database queries use context.Context for cancellation
- Repository layer handles transaction isolation

For high-concurrency scenarios, consider:
- Batch processing files by feature
- Using database transactions for multiple operations
- Implementing retry logic for contention

## Future Enhancements

1. **Batch Optimization**: Query max sequence once per feature for multiple files
2. **Custom Formats**: Support configurable key formats
3. **Caching**: Cache max sequence numbers for frequently-accessed features
4. **Parallel Processing**: Process multiple files concurrently with proper synchronization

## License

Part of the shark-task-manager project.
