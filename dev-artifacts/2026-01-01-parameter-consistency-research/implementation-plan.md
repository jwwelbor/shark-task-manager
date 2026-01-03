# Implementation Plan: E07-F12 Parameter Consistency Across Create and Update Commands

**Date**: 2026-01-01
**Feature**: E07-F12 - Parameter Consistency Across Create and Update Commands
**Architect**: Architect Agent
**Based on**: Research Findings Document (2026-01-01)

---

## Executive Summary

This plan addresses parameter inconsistencies between create and update commands for epics and features, while implementing a DRY (Don't Repeat Yourself) architecture to eliminate code duplication. The implementation is structured in three phases to minimize risk and maintain backward compatibility.

**Key Objectives**:
1. Add missing flags to create commands (--priority, --business-value, --status)
2. Refactor shared code into reusable modules
3. Maintain 100% backward compatibility
4. Improve testability and maintainability

**Complexity**: Medium (M) - Requires careful refactoring and comprehensive testing

---

## Architecture Design

### 1. Shared Flag Registration Module

**New File**: `internal/cli/commands/shared_flags.go`

This module provides composable flag registration functions following the builder pattern.

#### API Design

```go
package commands

import "github.com/spf13/cobra"

// FlagSet represents a group of related flags that can be added to commands
type FlagSet string

const (
    FlagSetMetadata    FlagSet = "metadata"     // --title, --description
    FlagSetPath        FlagSet = "path"         // --path, --filename, --force
    FlagSetEpicStatus  FlagSet = "epic_status"  // --status, --priority, --business-value
    FlagSetFeatureStatus FlagSet = "feature_status" // --status
    FlagSetCustomKey   FlagSet = "custom_key"   // --key
)

// AddFlagSet adds a predefined set of flags to a command
// This is the primary API for composing flag groups
func AddFlagSet(cmd *cobra.Command, flagSet FlagSet, opts ...FlagOption)

// FlagOption allows customization of flag behavior
type FlagOption func(*flagConfig)

// WithRequired marks flags as required
func WithRequired(flagNames ...string) FlagOption

// WithDefaults sets default values for flags
func WithDefaults(defaults map[string]interface{}) FlagOption

// Individual flag registration functions (for granular control)
func AddMetadataFlags(cmd *cobra.Command)
func AddPathFlags(cmd *cobra.Command)
func AddEpicStatusFlags(cmd *cobra.Command, defaults map[string]string)
func AddFeatureStatusFlags(cmd *cobra.Command, defaults map[string]string)
func AddCustomKeyFlag(cmd *cobra.Command)
```

#### Usage Examples

```go
// Epic Create Command
func init() {
    // Compose flags using flag sets
    AddFlagSet(epicCreateCmd, FlagSetMetadata)
    AddFlagSet(epicCreateCmd, FlagSetPath)
    AddFlagSet(epicCreateCmd, FlagSetEpicStatus,
        WithDefaults(map[string]interface{}{
            "status":   "draft",
            "priority": "medium",
        }))
    AddFlagSet(epicCreateCmd, FlagSetCustomKey)
}

// Epic Update Command
func init() {
    AddFlagSet(epicUpdateCmd, FlagSetMetadata)
    AddFlagSet(epicUpdateCmd, FlagSetPath)
    AddFlagSet(epicUpdateCmd, FlagSetEpicStatus) // No defaults for updates
    AddFlagSet(epicUpdateCmd, FlagSetCustomKey)
}

// Feature Create Command
func init() {
    AddFlagSet(featureCreateCmd, FlagSetMetadata)
    AddFlagSet(featureCreateCmd, FlagSetPath)
    AddFlagSet(featureCreateCmd, FlagSetFeatureStatus,
        WithDefaults(map[string]interface{}{
            "status": "draft",
        }))
    AddFlagSet(featureCreateCmd, FlagSetCustomKey)

    // Feature-specific flags
    featureCreateCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (required)")
    featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order")
    _ = featureCreateCmd.MarkFlagRequired("epic")
}
```

#### Benefits
- **Composability**: Mix and match flag sets per command
- **Consistency**: Same flags behave identically across commands
- **Maintainability**: Flag changes propagate automatically
- **Testability**: Flag registration can be tested in isolation

---

### 2. Shared Validation Module

**New File**: `internal/cli/commands/validators.go`

Centralizes parameter validation logic with clear error handling.

#### API Design

```go
package commands

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
    "github.com/spf13/cobra"
)

// PathValidationResult holds validated path information
type PathValidationResult struct {
    RelativePath string
    AbsolutePath string
}

// ValidateCustomPath validates and processes the --path flag
// Returns nil if flag not provided, error if validation fails
func ValidateCustomPath(cmd *cobra.Command, flagName string) (*PathValidationResult, error)

// ValidateCustomFilename validates and processes the --filename flag
// Returns nil if flag not provided, error if validation fails
func ValidateCustomFilename(cmd *cobra.Command, flagName string, projectRoot string) (*PathValidationResult, error)

// ValidateNoSpaces ensures a key doesn't contain spaces
func ValidateNoSpaces(key string, entityType string) error

// ValidateStatus ensures status is one of: draft, active, completed, archived
func ValidateStatus(status string, entityType string) error

// ValidatePriority ensures priority is one of: low, medium, high
func ValidatePriority(priority string, entityType string) error
```

#### Implementation Pattern

Each validator follows this pattern:
1. **Early return**: If flag not provided or empty, return nil (no error)
2. **Validation**: Apply business rules and constraints
3. **Error wrapping**: Provide context in error messages
4. **Consistent errors**: Use standard error formats

```go
func ValidateCustomPath(cmd *cobra.Command, flagName string) (*PathValidationResult, error) {
    customPath, _ := cmd.Flags().GetString(flagName)
    if customPath == "" {
        return nil, nil // Not provided, not an error
    }

    projectRoot, err := os.Getwd()
    if err != nil {
        return nil, fmt.Errorf("failed to get working directory: %w", err)
    }

    absPath, relPath, err := utils.ValidateFolderPath(customPath, projectRoot)
    if err != nil {
        return nil, fmt.Errorf("invalid path %q: %w", customPath, err)
    }

    return &PathValidationResult{
        RelativePath: relPath,
        AbsolutePath: absPath,
    }, nil
}
```

---

### 3. Shared File Assignment Module

**New File**: `internal/cli/commands/file_assignment.go`

Handles file collision detection and reassignment logic.

#### API Design

```go
package commands

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// FileCollision represents a file path conflict
type FileCollision struct {
    FilePath string
    Epic     *models.Epic   // Non-nil if epic claims this file
    Feature  *models.Feature // Non-nil if feature claims this file
}

// DetectFileCollision checks if a file path is already claimed
// Returns nil if no collision exists
func DetectFileCollision(
    ctx context.Context,
    filePath string,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) (*FileCollision, error)

// HandleFileReassignment manages --force file reassignment
// Returns error if collision exists and force=false
// Reassigns file and returns nil if force=true
func HandleFileReassignment(
    ctx context.Context,
    collision *FileCollision,
    force bool,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) error

// CreateBackupIfForce creates a database backup when --force is used
// Replaces backupDatabaseOnForce and backupDatabaseOnForceFeature
func CreateBackupIfForce(force bool, dbPath string, operation string) (string, error)
```

#### Usage Example

```go
// In epic create handler
func runEpicCreate(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Detect file collision
    collision, err := DetectFileCollision(ctx, filePath, epicRepo, featureRepo)
    if err != nil {
        return fmt.Errorf("failed to check file collision: %w", err)
    }

    // Handle reassignment if needed
    force, _ := cmd.Flags().GetBool("force")
    if err := HandleFileReassignment(ctx, collision, force, epicRepo, featureRepo); err != nil {
        return err
    }

    // Create backup if force used
    if _, err := CreateBackupIfForce(force, dbPath, "epic creation"); err != nil {
        return err
    }

    // ... continue with creation ...
}
```

---

### 4. Shared Status/Priority Parsing Module

**New File**: `internal/cli/commands/status_priority.go`

Handles parsing and validation of status/priority flags with defaults.

#### API Design

```go
package commands

import (
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/spf13/cobra"
)

// ParseEpicStatus extracts and validates --status flag for epic create
// Returns default if flag not provided
func ParseEpicStatus(cmd *cobra.Command, defaultStatus models.EpicStatus) (models.EpicStatus, error)

// ParseEpicPriority extracts and validates --priority flag for epic create
// Returns default if flag not provided
func ParseEpicPriority(cmd *cobra.Command, defaultPriority models.Priority) (models.Priority, error)

// ParseEpicBusinessValue extracts and validates --business-value flag
// Returns nil if flag not provided (nullable field)
func ParseEpicBusinessValue(cmd *cobra.Command) (*models.Priority, error)

// ParseFeatureStatus extracts and validates --status flag for feature create
// Returns default if flag not provided
func ParseFeatureStatus(cmd *cobra.Command, defaultStatus models.FeatureStatus) (models.FeatureStatus, error)
```

#### Implementation Pattern

```go
func ParseEpicStatus(cmd *cobra.Command, defaultStatus models.EpicStatus) (models.EpicStatus, error) {
    statusStr, _ := cmd.Flags().GetString("status")
    if statusStr == "" {
        return defaultStatus, nil
    }

    status := models.EpicStatus(statusStr)
    if err := ValidateStatus(string(status), "epic"); err != nil {
        return "", err
    }

    return status, nil
}

func ParseEpicBusinessValue(cmd *cobra.Command) (*models.Priority, error) {
    bvStr, _ := cmd.Flags().GetString("business-value")
    if bvStr == "" {
        return nil, nil // Nullable field
    }

    bv := models.Priority(bvStr)
    if err := ValidatePriority(string(bv), "epic"); err != nil {
        return nil, err
    }

    return &bv, nil
}
```

---

## File Structure

### New Files to Create

```
internal/cli/commands/
├── shared_flags.go          # NEW: Flag registration module
├── validators.go            # NEW: Validation functions
├── file_assignment.go       # NEW: File collision handling
└── status_priority.go       # NEW: Status/priority parsing
```

### Files to Modify

```
internal/cli/commands/
├── epic.go                  # Update flag registration, refactor create/update handlers
├── feature.go               # Update flag registration, refactor create/update handlers
├── epic_create_test.go      # Add tests for new flags
├── epic_update_test.go      # Update tests for shared functions
├── feature_update_test.go   # Update tests for shared functions
└── [new test files]         # Tests for shared modules
```

### New Test Files to Create

```
internal/cli/commands/
├── shared_flags_test.go     # NEW: Test flag registration
├── validators_test.go       # NEW: Test validation functions
├── file_assignment_test.go  # NEW: Test collision detection
└── status_priority_test.go  # NEW: Test parsing functions
```

---

## Migration Strategy

### Phase 1: Add Missing Flags (Low Risk, Quick Win)

**Objective**: Add missing flags to create commands without refactoring.

**Tasks**:
1. Add `--priority`, `--business-value`, `--status` flags to `epicCreateCmd`
2. Add `--status` flag to `featureCreateCmd`
3. Update command handlers to parse and use new flags
4. Update default value assignment in create handlers
5. Add basic tests for new flags

**Files Changed**:
- `internal/cli/commands/epic.go` (flag registration + handler)
- `internal/cli/commands/feature.go` (flag registration + handler)
- `internal/cli/commands/epic_create_test.go` (new tests)
- Tests for feature create (if file exists, or create new)

**Acceptance Criteria**:
- Epic can be created with `--priority=high`
- Epic can be created with `--business-value=low`
- Epic can be created with `--status=active`
- Feature can be created with `--status=active`
- Defaults still work when flags not provided
- All existing tests pass
- New flags documented in help text

**Estimated Effort**: Small (S)

---

### Phase 2: Extract Shared Modules (Medium Risk, DRY Foundation)

**Objective**: Create shared modules without changing epic/feature commands.

**Tasks**:
1. Create `shared_flags.go` with flag registration functions
2. Create `validators.go` with validation functions
3. Create `file_assignment.go` with collision detection
4. Create `status_priority.go` with parsing functions
5. Write comprehensive tests for all shared modules
6. **Do NOT modify epic.go or feature.go yet**

**Files Changed**:
- `internal/cli/commands/shared_flags.go` (NEW)
- `internal/cli/commands/validators.go` (NEW)
- `internal/cli/commands/file_assignment.go` (NEW)
- `internal/cli/commands/status_priority.go` (NEW)
- `internal/cli/commands/shared_flags_test.go` (NEW)
- `internal/cli/commands/validators_test.go` (NEW)
- `internal/cli/commands/file_assignment_test.go` (NEW)
- `internal/cli/commands/status_priority_test.go` (NEW)

**Acceptance Criteria**:
- All shared functions have 90%+ test coverage
- Shared functions work with mocked repositories (no real DB)
- All existing tests still pass
- No behavioral changes to commands

**Estimated Effort**: Medium (M)

---

### Phase 3: Refactor Commands to Use Shared Modules (Medium Risk, Cleanup)

**Objective**: Replace duplicate code in epic.go and feature.go with shared functions.

**Tasks**:
1. Replace duplicate flag registration with `AddFlagSet` calls
2. Replace validation logic with shared validator functions
3. Replace file collision detection with shared functions
4. Remove duplicate `backupDatabaseOnForce*` functions
5. Update tests to use shared test helpers
6. Verify 100% backward compatibility

**Files Changed**:
- `internal/cli/commands/epic.go` (refactor create/update handlers)
- `internal/cli/commands/feature.go` (refactor create/update handlers)
- `internal/cli/commands/epic_create_test.go` (update for refactoring)
- `internal/cli/commands/epic_update_test.go` (update for refactoring)
- `internal/cli/commands/feature_update_test.go` (update for refactoring)

**Acceptance Criteria**:
- No duplicate validation logic in epic.go or feature.go
- Single `CreateBackupIfForce` function replaces two duplicates
- All existing tests pass
- No behavioral changes to commands
- Code coverage maintained or improved
- Lines of code reduced by 20%+ in epic.go and feature.go

**Estimated Effort**: Medium (M)

---

## API Contracts for Shared Functions

### Shared Flags Module

```go
// AddFlagSet adds a predefined set of flags to a command
// Parameters:
//   cmd: The cobra command to add flags to
//   flagSet: The flag set identifier (metadata, path, epic_status, etc.)
//   opts: Optional configuration (defaults, required flags, etc.)
//
// Returns: none (panics on invalid flagSet)
//
// Example:
//   AddFlagSet(epicCreateCmd, FlagSetEpicStatus,
//       WithDefaults(map[string]interface{}{"status": "draft"}))
func AddFlagSet(cmd *cobra.Command, flagSet FlagSet, opts ...FlagOption)
```

### Validators Module

```go
// ValidateCustomPath validates the --path flag
// Parameters:
//   cmd: The cobra command containing the flag
//   flagName: Name of the flag to validate (usually "path")
//
// Returns:
//   *PathValidationResult: Validated paths (nil if flag not provided)
//   error: Validation error or nil
//
// Errors:
//   - Directory access errors
//   - Path format errors
//   - Relative path resolution errors
func ValidateCustomPath(cmd *cobra.Command, flagName string) (*PathValidationResult, error)

// ValidateNoSpaces ensures key contains no spaces
// Parameters:
//   key: The key to validate
//   entityType: "epic", "feature", or "task" (for error messages)
//
// Returns:
//   error: Validation error or nil
//
// Example:
//   if err := ValidateNoSpaces(customKey, "epic"); err != nil {
//       return err
//   }
func ValidateNoSpaces(key string, entityType string) error
```

### File Assignment Module

```go
// DetectFileCollision checks if file is claimed by another entity
// Parameters:
//   ctx: Context for database operations
//   filePath: The file path to check (relative to project root)
//   epicRepo: Epic repository for collision checking
//   featureRepo: Feature repository for collision checking
//
// Returns:
//   *FileCollision: Collision info (nil if no collision)
//   error: Database error or nil
//
// Example:
//   collision, err := DetectFileCollision(ctx, "docs/plan/epic.md", epicRepo, featureRepo)
//   if err != nil {
//       return err
//   }
//   if collision != nil {
//       // Handle collision
//   }
func DetectFileCollision(
    ctx context.Context,
    filePath string,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) (*FileCollision, error)

// HandleFileReassignment manages file reassignment with --force
// Parameters:
//   ctx: Context for database operations
//   collision: The detected collision (from DetectFileCollision)
//   force: Whether --force flag was provided
//   epicRepo: Epic repository for reassignment
//   featureRepo: Feature repository for reassignment
//
// Returns:
//   error: Reassignment error or nil
//
// Behavior:
//   - If collision=nil: Returns nil (no-op)
//   - If collision exists and force=false: Returns error
//   - If collision exists and force=true: Clears FilePath on conflicting entity
//
// Example:
//   if err := HandleFileReassignment(ctx, collision, force, epicRepo, featureRepo); err != nil {
//       return fmt.Errorf("file reassignment failed: %w", err)
//   }
func HandleFileReassignment(
    ctx context.Context,
    collision *FileCollision,
    force bool,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
) error
```

### Status/Priority Module

```go
// ParseEpicStatus extracts and validates --status flag
// Parameters:
//   cmd: The cobra command containing the flag
//   defaultStatus: Status to return if flag not provided
//
// Returns:
//   models.EpicStatus: Parsed status or default
//   error: Validation error or nil
//
// Valid values: draft, active, completed, archived
//
// Example:
//   status, err := ParseEpicStatus(cmd, models.EpicStatusDraft)
//   if err != nil {
//       return err
//   }
func ParseEpicStatus(cmd *cobra.Command, defaultStatus models.EpicStatus) (models.EpicStatus, error)

// ParseEpicBusinessValue extracts --business-value flag (nullable)
// Parameters:
//   cmd: The cobra command containing the flag
//
// Returns:
//   *models.Priority: Parsed business value or nil
//   error: Validation error or nil
//
// Example:
//   bv, err := ParseEpicBusinessValue(cmd)
//   if err != nil {
//       return err
//   }
//   epic.BusinessValue = bv // May be nil
func ParseEpicBusinessValue(cmd *cobra.Command) (*models.Priority, error)
```

---

## Testing Strategy

### Test Architecture

Following project guidelines:
- **Repository tests**: Use real database with cleanup
- **CLI command tests**: Use mocked repositories (NO real database)
- **Shared module tests**: Pure logic tests (no database, no mocks)

### Test Coverage Requirements

| Module | Target Coverage | Test Type |
|--------|----------------|-----------|
| `shared_flags.go` | 95% | Unit tests (flag registration) |
| `validators.go` | 95% | Unit tests (validation logic) |
| `file_assignment.go` | 90% | Mock repository tests |
| `status_priority.go` | 95% | Unit tests (parsing logic) |
| Epic create handler | 85% | Mock repository tests |
| Feature create handler | 85% | Mock repository tests |

### Test Files

#### `shared_flags_test.go`
```go
func TestAddFlagSet_Metadata(t *testing.T)
func TestAddFlagSet_Path(t *testing.T)
func TestAddFlagSet_EpicStatus(t *testing.T)
func TestAddFlagSet_WithDefaults(t *testing.T)
func TestAddFlagSet_InvalidFlagSet(t *testing.T) // Should panic
```

#### `validators_test.go`
```go
func TestValidateCustomPath_Valid(t *testing.T)
func TestValidateCustomPath_Invalid(t *testing.T)
func TestValidateCustomPath_NotProvided(t *testing.T)
func TestValidateNoSpaces_Valid(t *testing.T)
func TestValidateNoSpaces_Invalid(t *testing.T)
func TestValidateStatus_AllValidValues(t *testing.T)
func TestValidateStatus_InvalidValue(t *testing.T)
func TestValidatePriority_AllValidValues(t *testing.T)
```

#### `file_assignment_test.go`
```go
func TestDetectFileCollision_NoCollision(t *testing.T)
func TestDetectFileCollision_EpicCollision(t *testing.T)
func TestDetectFileCollision_FeatureCollision(t *testing.T)
func TestHandleFileReassignment_NoCollision(t *testing.T)
func TestHandleFileReassignment_CollisionWithForce(t *testing.T)
func TestHandleFileReassignment_CollisionWithoutForce(t *testing.T)
func TestCreateBackupIfForce_ForceTrue(t *testing.T)
func TestCreateBackupIfForce_ForceFalse(t *testing.T)
```

#### `status_priority_test.go`
```go
func TestParseEpicStatus_Provided(t *testing.T)
func TestParseEpicStatus_NotProvided(t *testing.T)
func TestParseEpicStatus_Invalid(t *testing.T)
func TestParseEpicPriority_AllValues(t *testing.T)
func TestParseEpicBusinessValue_Provided(t *testing.T)
func TestParseEpicBusinessValue_NotProvided(t *testing.T)
func TestParseFeatureStatus_AllValues(t *testing.T)
```

#### `epic_create_test.go` (Enhanced)
```go
// Existing tests...

func TestEpicCreate_WithPriority(t *testing.T)
func TestEpicCreate_WithBusinessValue(t *testing.T)
func TestEpicCreate_WithStatus(t *testing.T)
func TestEpicCreate_WithAllNewFlags(t *testing.T)
func TestEpicCreate_DefaultsWhenFlagsNotProvided(t *testing.T)
```

#### Feature Create Tests (New or Enhanced)
```go
func TestFeatureCreate_WithStatus(t *testing.T)
func TestFeatureCreate_DefaultStatusWhenNotProvided(t *testing.T)
```

### Test Execution Plan

**Phase 1 Tests**:
```bash
# Test new flags in isolation
go test -v ./internal/cli/commands -run TestEpicCreate_WithPriority
go test -v ./internal/cli/commands -run TestEpicCreate_WithBusinessValue
go test -v ./internal/cli/commands -run TestEpicCreate_WithStatus
go test -v ./internal/cli/commands -run TestFeatureCreate_WithStatus

# Regression tests
go test -v ./internal/cli/commands/...
```

**Phase 2 Tests**:
```bash
# Test shared modules in isolation
go test -v ./internal/cli/commands -run TestAddFlagSet
go test -v ./internal/cli/commands -run TestValidate
go test -v ./internal/cli/commands -run TestFileAssignment
go test -v ./internal/cli/commands -run TestParse

# Full suite (should still pass)
go test -v ./internal/cli/commands/...
```

**Phase 3 Tests**:
```bash
# Test refactored commands
go test -v ./internal/cli/commands -run TestEpicCreate
go test -v ./internal/cli/commands -run TestEpicUpdate
go test -v ./internal/cli/commands -run TestFeatureCreate
go test -v ./internal/cli/commands -run TestFeatureUpdate

# Full regression
make test
```

---

## Backward Compatibility Guarantees

### 1. Default Behavior Unchanged

**Before**:
```bash
shark epic create "My Epic"
# Creates with status=draft, priority=medium, business_value=nil
```

**After**:
```bash
shark epic create "My Epic"
# Still creates with status=draft, priority=medium, business_value=nil
```

### 2. Existing Flags Still Work

**Before**:
```bash
shark epic create "My Epic" --description="desc" --path="docs/custom"
```

**After**:
```bash
shark epic create "My Epic" --description="desc" --path="docs/custom"
# Still works identically
```

### 3. Database Schema Unchanged

No migrations required. All changes use existing columns:
- `epics.status` (already exists)
- `epics.priority` (already exists)
- `epics.business_value` (already exists)
- `features.status` (already exists)

### 4. Repository Interface Unchanged

No changes to repository method signatures:
- `EpicRepository.Create(ctx, *Epic) error`
- `FeatureRepository.Create(ctx, *Feature) error`

### 5. JSON Output Format Unchanged

API consumers using `--json` flag see identical output structure.

---

## Acceptance Criteria

### Phase 1: Flag Addition

- [ ] Epic create accepts `--priority` flag with values: low, medium, high
- [ ] Epic create accepts `--business-value` flag with values: low, medium, high
- [ ] Epic create accepts `--status` flag with values: draft, active, completed, archived
- [ ] Feature create accepts `--status` flag with values: draft, active, completed, archived
- [ ] Default values match current behavior when flags not provided
- [ ] Invalid flag values produce clear error messages
- [ ] Help text accurately describes all flags
- [ ] All existing tests pass
- [ ] New tests added for each new flag
- [ ] No database schema changes required

### Phase 2: Shared Modules

- [ ] `shared_flags.go` created with flag registration functions
- [ ] `validators.go` created with validation functions
- [ ] `file_assignment.go` created with collision detection
- [ ] `status_priority.go` created with parsing functions
- [ ] All shared modules have 90%+ test coverage
- [ ] Shared module tests use mocks (no real database)
- [ ] All existing tests still pass
- [ ] No behavioral changes to existing commands

### Phase 3: Refactoring

- [ ] Epic create uses shared flag registration
- [ ] Epic create uses shared validation functions
- [ ] Epic create uses shared file collision detection
- [ ] Epic update uses shared functions
- [ ] Feature create uses shared flag registration
- [ ] Feature create uses shared validation functions
- [ ] Feature create uses shared file collision detection
- [ ] Feature update uses shared functions
- [ ] Duplicate `backupDatabaseOnForce*` functions removed
- [ ] Single `CreateBackupIfForce` function used everywhere
- [ ] All tests updated for refactored code
- [ ] All tests pass
- [ ] Code coverage maintained or improved
- [ ] Lines of code reduced by 20%+ in epic.go and feature.go

### Overall Quality Gates

- [ ] 100% backward compatibility maintained
- [ ] All existing integration tests pass
- [ ] No performance regression (command execution time)
- [ ] No new linter warnings or errors
- [ ] Documentation updated (if needed)
- [ ] Code follows project conventions and patterns

---

## Implementation Sequence

### Week 1: Phase 1 - Flag Addition

**Day 1-2**: Epic Create Flags
- Add `--priority`, `--business-value`, `--status` flags to epic create
- Update handler to parse and use flags
- Write tests for new flags

**Day 3**: Feature Create Flags
- Add `--status` flag to feature create
- Update handler to parse and use flag
- Write tests for new flag

**Day 4**: Testing & Validation
- Run full test suite
- Manual testing of create commands
- Document new flags in help text

**Deliverable**: Working create commands with new flags

---

### Week 2: Phase 2 - Shared Modules

**Day 1**: `shared_flags.go`
- Implement flag registration functions
- Write comprehensive tests

**Day 2**: `validators.go`
- Extract validation functions
- Write comprehensive tests

**Day 3**: `file_assignment.go`
- Extract collision detection
- Write comprehensive tests

**Day 4**: `status_priority.go`
- Extract parsing functions
- Write comprehensive tests

**Day 5**: Integration Testing
- Verify shared modules work together
- Run full test suite

**Deliverable**: Battle-tested shared modules ready for integration

---

### Week 3: Phase 3 - Refactoring

**Day 1-2**: Refactor Epic Commands
- Replace duplicate code with shared functions
- Update tests
- Verify backward compatibility

**Day 3-4**: Refactor Feature Commands
- Replace duplicate code with shared functions
- Update tests
- Verify backward compatibility

**Day 5**: Final Testing & Cleanup
- Full regression testing
- Code review
- Documentation updates

**Deliverable**: Clean, DRY codebase with improved maintainability

---

## Risk Assessment & Mitigation

### Risk 1: Breaking Backward Compatibility

**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Comprehensive test suite covering all existing use cases
- Default values explicitly set to match current behavior
- Phase 1 completed and tested before Phase 3 refactoring
- Manual testing of all command combinations

### Risk 2: Test Coverage Gaps

**Likelihood**: Medium
**Impact**: Medium
**Mitigation**:
- Minimum 90% coverage requirement for shared modules
- Table-driven tests covering all edge cases
- Integration tests for end-to-end workflows
- Code review focusing on test quality

### Risk 3: Refactoring Introduces Bugs

**Likelihood**: Medium
**Impact**: Medium
**Mitigation**:
- Phase 2 creates and tests shared modules BEFORE refactoring
- Incremental refactoring (epic first, then feature)
- Full test suite run after each change
- Git commits after each working phase

### Risk 4: Performance Regression

**Likelihood**: Low
**Impact**: Low
**Mitigation**:
- Shared functions should be faster (less duplicate work)
- Benchmark command execution time before/after
- Monitor for any database query changes

---

## Success Metrics

### Code Quality Metrics

| Metric | Before | Target After |
|--------|--------|--------------|
| Lines of code (epic.go) | ~1500 | <1200 (-20%) |
| Lines of code (feature.go) | ~1600 | <1300 (-20%) |
| Duplicate code blocks | 5+ | 0 |
| Test coverage (commands) | ~75% | >85% |
| Test coverage (shared modules) | N/A | >90% |

### Functional Metrics

| Metric | Target |
|--------|--------|
| Backward compatibility | 100% |
| Existing tests passing | 100% |
| New features working | 100% |
| Command execution time | No regression |

### Developer Experience Metrics

| Metric | Target |
|--------|--------|
| Time to add new flag | <5 minutes (vs 20+ before) |
| Code review time | -30% (less duplication) |
| Onboarding clarity | Improved (shared modules) |

---

## Related Documentation

**Research Artifacts**:
- `dev-artifacts/2026-01-01-parameter-consistency-research/research-findings.md`

**Source Files**:
- `internal/cli/commands/epic.go` - Epic commands
- `internal/cli/commands/feature.go` - Feature commands
- `internal/models/epic.go` - Epic model
- `internal/models/feature.go` - Feature model

**Testing References**:
- `internal/cli/commands/epic_create_test.go` - Epic create tests
- `internal/cli/commands/epic_update_test.go` - Epic update tests
- `internal/cli/commands/feature_update_test.go` - Feature update tests

**Project Guidelines**:
- `CLAUDE.md` - Testing architecture, DRY principles
- `docs/CLI_REFERENCE.md` - CLI documentation (to be updated)

---

## Questions for Product Manager

Before implementation begins, architect requests clarification on:

1. **Priority of phases**: Should we implement all three phases, or is Phase 1 sufficient for now?

2. **Default values**: Confirm default values for new flags:
   - Epic status: `draft` (current hardcoded value)
   - Epic priority: `medium` (current hardcoded value)
   - Epic business value: `nil` (current hardcoded value)
   - Feature status: `draft` (current hardcoded value)

3. **Validation strictness**: Should invalid status/priority values:
   - Error immediately (fail fast)
   - Warn and use default (forgiving)

4. **Documentation updates**: Which documents need updating after implementation?
   - CLI_REFERENCE.md
   - README.md
   - User guide (if exists)

5. **Release timeline**: Target release version for this feature?

---

## Conclusion

This implementation plan provides a phased approach to adding parameter consistency while eliminating code duplication. The architecture follows DRY principles, maintains backward compatibility, and improves long-term maintainability.

**Key Benefits**:
- **Consistency**: Create and update commands have matching parameters
- **Efficiency**: Set all properties at creation (no create-then-update needed)
- **Maintainability**: Shared code reduces duplication by 20%+
- **Testability**: Isolated shared modules are easier to test
- **Extensibility**: Adding new flags becomes trivial

**Recommended Approach**: Implement all three phases sequentially for maximum benefit. Phase 1 can be deployed independently if time constraints exist.

**Next Step**: Review with Product Manager and TechLead, then hand off to TDD agent for implementation.
