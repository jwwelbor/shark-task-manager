# Research Findings: E07-F12 Parameter Consistency Across Create and Update Commands

**Date**: 2026-01-01
**Feature**: E07-F12 - Parameter Consistency Across Create and Update Commands
**Analyst**: Research Agent
**Purpose**: Analyze create and update commands for epic and feature to identify parameter inconsistencies

---

## Executive Summary

Analysis of `internal/cli/commands/epic.go` and `internal/cli/commands/feature.go` reveals parameter inconsistencies between create and update commands. The primary issue is missing flags in create commands that are available in update commands.

### Key Findings

1. **Epic Create** is missing `--priority` and `--business-value` flags (documented in help text but not registered)
2. **Epic Create** is missing `--status` flag (available in update)
3. **Feature Create** is missing `--status` flag (available in update)
4. Opportunity to refactor shared parameter handling logic (DRY principle violation)

---

## Current Implementation Analysis

### Epic Command Flags

#### Epic Create (`epicCreateCmd`)
**Location**: `internal/cli/commands/epic.go:234-238`

**Registered Flags**:
```go
epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
epicCreateCmd.Flags().StringVar(&epicCreatePath, "path", "", "Custom base folder path...")
epicCreateCmd.Flags().StringVar(&epicCreateKey, "key", "", "Custom key for the epic...")
epicCreateCmd.Flags().String("filename", "", "Custom filename path...")
epicCreateCmd.Flags().Bool("force", false, "Force reassignment...")
```

**Help Text Claims** (lines 148-155):
```
Flags:
  --filename string    Custom file path relative to project root (must end in .md)
  --path string        Custom base folder path for this epic and children
  --force              Force reassignment if file already claimed
  --description string Epic description
  --priority string    Priority: high, medium, low (default: medium)      ← NOT REGISTERED
  --business-value string Business value: high, medium, low                ← NOT REGISTERED
```

#### Epic Update (`epicUpdateCmd`)
**Location**: `internal/cli/commands/epic.go:243-252`

**Registered Flags**:
```go
epicUpdateCmd.Flags().String("title", "", "New title for the epic")
epicUpdateCmd.Flags().String("description", "", "New description for the epic")
epicUpdateCmd.Flags().String("status", "", "New status: draft, active, completed, archived")
epicUpdateCmd.Flags().String("priority", "", "New priority: low, medium, high")
epicUpdateCmd.Flags().String("business-value", "", "New business value: low, medium, high")
epicUpdateCmd.Flags().String("key", "", "New key for the epic...")
epicUpdateCmd.Flags().String("filename", "", "New file path...")
epicUpdateCmd.Flags().String("path", "", "New custom folder base path")
epicUpdateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
```

**Missing in Create**:
- ❌ `--priority` (mentioned in help but not registered)
- ❌ `--business-value` (mentioned in help but not registered)
- ❌ `--status` (completely missing, update supports draft/active/completed/archived)

---

### Feature Command Flags

#### Feature Create (`featureCreateCmd`)
**Location**: `internal/cli/commands/feature.go:226-233`

**Registered Flags**:
```go
featureCreateCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (required)")
featureCreateCmd.Flags().StringVar(&featureCreateDescription, "description", "", "Feature description (optional)")
featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order (optional)")
featureCreateCmd.Flags().StringVar(&featureCreatePath, "path", "", "Custom base folder path...")
featureCreateCmd.Flags().StringVar(&featureCreateKey, "key", "", "Custom key for the feature...")
featureCreateCmd.Flags().StringVar(&featureCreateFilename, "filename", "", "Custom filename path...")
featureCreateCmd.Flags().BoolVar(&featureCreateForce, "force", false, "Force reassignment...")
_ = featureCreateCmd.MarkFlagRequired("epic")
```

#### Feature Update (`featureUpdateCmd`)
**Location**: `internal/cli/commands/feature.go:242-249`

**Registered Flags**:
```go
featureUpdateCmd.Flags().String("title", "", "New title for the feature")
featureUpdateCmd.Flags().String("description", "", "New description for the feature")
featureUpdateCmd.Flags().String("status", "", "New status: draft, active, completed, archived")
featureUpdateCmd.Flags().Int("execution-order", -1, "New execution order (-1 = no change)")
featureUpdateCmd.Flags().String("key", "", "New key for the feature...")
featureUpdateCmd.Flags().String("filename", "", "New file path...")
featureUpdateCmd.Flags().String("path", "", "New custom folder base path")
featureUpdateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
```

**Missing in Create**:
- ❌ `--status` (update supports draft/active/completed/archived)

---

## Code Architecture Analysis

### Current Pattern (DRY Violation)

**Issue**: Parameter handling logic is duplicated across:
1. Epic create vs epic update
2. Feature create vs feature update
3. Epic commands vs feature commands (similar patterns)

**Example Duplication**:

```go
// epic.go:1427-1442 (update)
customPath, _ := cmd.Flags().GetString("path")
if customPath != "" {
    projectRoot, err := os.Getwd()
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
        os.Exit(1)
    }
    _, relPath, err := utils.ValidateFolderPath(customPath, projectRoot)
    if err != nil {
        cli.Error(fmt.Sprintf("Error: %v", err))
        os.Exit(1)
    }
    epic.CustomFolderPath = &relPath
    changed = true
}

// feature.go:1497-1512 (update) - IDENTICAL LOGIC
customPath, _ := cmd.Flags().GetString("path")
if customPath != "" {
    projectRoot, err := os.Getwd()
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
        os.Exit(1)
    }
    _, relPath, err := utils.ValidateFolderPath(customPath, projectRoot)
    if err != nil {
        cli.Error(fmt.Sprintf("Error: %v", err))
        os.Exit(1)
    }
    feature.CustomFolderPath = &relPath
    changed = true
}
```

**Similar duplication for**:
- File path validation (`--filename`)
- Force reassignment logic (`--force`)
- Custom key validation (no spaces check)
- Database collision detection
- Backup creation before force operations

---

## Data Model Analysis

### Epic Model
**Location**: `internal/models/epic.go`

```go
type Epic struct {
    ID               int64       `json:"id"`
    Key              string      `json:"key"`
    Title            string      `json:"title"`
    Description      *string     `json:"description"`
    Status           EpicStatus  `json:"status"`      // draft, active, completed, archived
    Priority         Priority    `json:"priority"`    // low, medium, high
    BusinessValue    *Priority   `json:"business_value"` // nullable
    FilePath         *string     `json:"file_path"`
    CustomFolderPath *string     `json:"custom_folder_path"`
    Slug             string      `json:"slug"`
    CreatedAt        time.Time   `json:"created_at"`
    UpdatedAt        time.Time   `json:"updated_at"`
}
```

**Default Values on Create** (epic.go:913-922):
```go
epic := &models.Epic{
    Key:              nextKey,
    Title:            epicTitle,
    Description:      &epicCreateDescription,
    Status:           models.EpicStatusDraft,        // ← Hardcoded to "draft"
    Priority:         models.PriorityMedium,         // ← Hardcoded to "medium"
    BusinessValue:    nil,                           // ← Always nil
    FilePath:         customFilePath,
    CustomFolderPath: customFolderPath,
}
```

### Feature Model
**Location**: `internal/models/feature.go`

```go
type Feature struct {
    ID               int64          `json:"id"`
    EpicID           int64          `json:"epic_id"`
    Key              string         `json:"key"`
    Title            string         `json:"title"`
    Description      *string        `json:"description"`
    Status           FeatureStatus  `json:"status"`  // draft, active, completed, archived
    ProgressPct      float64        `json:"progress_pct"`
    ExecutionOrder   *int           `json:"execution_order"` // nullable
    FilePath         *string        `json:"file_path"`
    CustomFolderPath *string        `json:"custom_folder_path"`
    Slug             string         `json:"slug"`
    CreatedAt        time.Time      `json:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at"`
}
```

**Default Values on Create** (feature.go:1055-1065):
```go
feature := &models.Feature{
    EpicID:           epic.ID,
    Key:              featureKey,
    Title:            featureTitle,
    Description:      &featureCreateDescription,
    Status:           models.FeatureStatusDraft,     // ← Hardcoded to "draft"
    ProgressPct:      0.0,
    ExecutionOrder:   executionOrder,
    FilePath:         customFilePath,
    CustomFolderPath: customFolderPath,
}
```

---

## Identified Refactoring Opportunities

### 1. Shared Flag Registration

**Current**: Flags defined inline in `init()` for each command
**Proposal**: Extract common flag groups to shared functions

```go
// Proposed: internal/cli/commands/shared_flags.go

// AddPathFlags adds --path, --filename, --force flags to a command
func AddPathFlags(cmd *cobra.Command) {
    cmd.Flags().String("path", "", "Custom base folder path...")
    cmd.Flags().String("filename", "", "Custom filename path...")
    cmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
}

// AddStatusFlags adds --status, --priority flags to a command
func AddStatusFlags(cmd *cobra.Command, entity string) {
    cmd.Flags().String("status", "", fmt.Sprintf("Status: draft, active, completed, archived"))
    cmd.Flags().String("priority", "", "Priority: low, medium, high")
}

// AddMetadataFlags adds --title, --description flags
func AddMetadataFlags(cmd *cobra.Command) {
    cmd.Flags().String("title", "", "Title")
    cmd.Flags().String("description", "", "Description")
}
```

### 2. Shared Parameter Validation

**Current**: Validation logic duplicated in each command handler
**Proposal**: Extract to shared validation functions

```go
// Proposed: internal/cli/commands/validators.go

// ValidateAndProcessCustomPath validates --path flag and returns relative path
func ValidateAndProcessCustomPath(cmd *cobra.Command, flagName string) (*string, error)

// ValidateAndProcessCustomFilename validates --filename flag and handles collisions
func ValidateAndProcessCustomFilename(
    cmd *cobra.Command,
    projectRoot string,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    ctx context.Context,
) (*string, error)

// ValidateNoSpaces ensures key doesn't contain spaces
func ValidateNoSpaces(key string, entity string) error
```

### 3. Shared Backup Logic

**Current**: Duplicate backup functions
**Existing**: `backupDatabaseOnForce` (epic.go:35-50) and `backupDatabaseOnForceFeature` (feature.go:34-49) are identical

**Proposal**: Move to shared utility

```go
// Proposed: internal/cli/utils.go or internal/db/backup.go

// BackupDatabaseOnForce creates a backup when --force flag is used
func BackupDatabaseOnForce(force bool, dbPath string, operation string) (string, error)
```

### 4. Shared File Collision Detection

**Current**: Collision detection logic duplicated in epic create and feature create

**Proposal**: Extract to shared function

```go
// Proposed: internal/cli/commands/file_assignment.go

type FileCollision struct {
    Epic    *models.Epic
    Feature *models.Feature
}

// DetectFileCollision checks if a file path is already claimed
func DetectFileCollision(
    ctx context.Context,
    filePath string,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) (*FileCollision, error)

// HandleFileReassignment handles --force file reassignment
func HandleFileReassignment(
    ctx context.Context,
    collision *FileCollision,
    force bool,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) error
```

---

## Recommended Changes

### Phase 1: Add Missing Flags (Quick Win)

#### Epic Create
```go
// Add to epic.go init() after line 238
epicCreateCmd.Flags().String("priority", "medium", "Priority: low, medium, high")
epicCreateCmd.Flags().String("business-value", "", "Business value: low, medium, high")
epicCreateCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived")
```

#### Feature Create
```go
// Add to feature.go init() after line 232
featureCreateCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived")
```

### Phase 2: Update Create Handlers (Use New Flags)

#### Epic Create Handler
```go
// In runEpicCreate, before creating epic (after line 912)
priority, _ := cmd.Flags().GetString("priority")
if priority == "" {
    priority = "medium" // default
}

businessValue, _ := cmd.Flags().GetString("business-value")
var bvPtr *models.Priority
if businessValue != "" {
    bv := models.Priority(businessValue)
    bvPtr = &bv
}

status, _ := cmd.Flags().GetString("status")
if status == "" {
    status = "draft" // default
}

epic := &models.Epic{
    // ... existing fields ...
    Status:        models.EpicStatus(status),
    Priority:      models.Priority(priority),
    BusinessValue: bvPtr,
    // ... rest of fields ...
}
```

#### Feature Create Handler
```go
// In runFeatureCreate, before creating feature (after line 1054)
status, _ := cmd.Flags().GetString("status")
if status == "" {
    status = "draft" // default
}

feature := &models.Feature{
    // ... existing fields ...
    Status: models.FeatureStatus(status),
    // ... rest of fields ...
}
```

### Phase 3: Refactor for DRY (Architecture Improvement)

1. Create `internal/cli/commands/shared_flags.go` with flag registration helpers
2. Create `internal/cli/commands/validators.go` with parameter validation functions
3. Create `internal/cli/commands/file_assignment.go` with file collision logic
4. Update epic.go and feature.go to use shared functions
5. Remove duplicate `backupDatabaseOnForceFeature` function

---

## Testing Requirements

### Unit Tests Needed

1. **Flag Parsing Tests**
   - Test that new flags are correctly parsed
   - Test default values for optional flags
   - Test flag validation (e.g., invalid status values)

2. **Create Command Tests**
   - Test epic create with `--priority=high`
   - Test epic create with `--business-value=low`
   - Test epic create with `--status=active`
   - Test feature create with `--status=active`
   - Test that defaults still work when flags not provided

3. **Validation Tests**
   - Test shared validation functions
   - Test file collision detection
   - Test custom path validation

4. **Integration Tests**
   - Test end-to-end create → update workflow
   - Test that created entities have correct status/priority
   - Test file assignment with --force

---

## Files Requiring Changes

### Immediate Changes (Phase 1 & 2)
1. `internal/cli/commands/epic.go` - Add flags to epicCreateCmd, update runEpicCreate
2. `internal/cli/commands/feature.go` - Add flag to featureCreateCmd, update runFeatureCreate

### Refactoring Changes (Phase 3)
3. `internal/cli/commands/shared_flags.go` - New file for shared flag registration
4. `internal/cli/commands/validators.go` - New file for shared validation
5. `internal/cli/commands/file_assignment.go` - New file for file collision handling
6. `internal/cli/commands/epic_test.go` - Add tests for new flags
7. `internal/cli/commands/feature_test.go` - Add tests for new flags

---

## Business Impact

### User Experience Improvements

1. **Consistency**: Users can set all properties at creation time instead of create → update workflow
2. **Efficiency**: One command instead of two for setting initial state
3. **Predictability**: Create and update commands have matching parameters

### Code Quality Improvements

1. **DRY**: Eliminate duplicate parameter handling logic
2. **Maintainability**: Changes to parameter validation only need to happen once
3. **Testability**: Shared functions are easier to test in isolation

---

## Next Steps

1. ✅ Research complete - findings documented
2. ⏭️ Architect creates DRY implementation plan
3. ⏭️ TDD agent implements with tests
4. ⏭️ Code review and validation
5. ⏭️ Update related documentation

---

## Related Files

**Core Implementation**:
- `internal/cli/commands/epic.go:140-1498` - Epic commands
- `internal/cli/commands/feature.go:116-1566` - Feature commands

**Data Models**:
- `internal/models/epic.go` - Epic model definition
- `internal/models/feature.go` - Feature model definition

**Validators**:
- `internal/utils/validation.go` - Folder path validation
- `internal/taskcreation/validation.go` - Filename validation

**Repositories** (may need updates if default values change):
- `internal/repository/epic_repository.go`
- `internal/repository/feature_repository.go`
