# Technical Design: Workflow Profile Support for `shark init update`

**Feature**: E07-F24-workflow-profile-support
**Document Version**: 1.0
**Created**: 2026-01-25
**Status**: Draft
**Owner**: Architect Agent

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Data Structures](#data-structures)
4. [Component Design](#component-design)
5. [File-by-File Implementation](#file-by-file-implementation)
6. [Configuration Merge Strategy](#configuration-merge-strategy)
7. [Testing Strategy](#testing-strategy)
8. [Implementation Phases](#implementation-phases)
9. [Risk Assessment](#risk-assessment)
10. [Performance Considerations](#performance-considerations)

---

## Executive Summary

This feature adds workflow profile support to the `shark init` command, enabling users to quickly configure their task workflow from predefined templates (basic or advanced) while preserving existing configuration.

**Key Design Principles**:
- **Appropriate**: Leverages existing patterns (PathResolver, atomic writes, Cobra commands)
- **Proven**: Uses established config merge strategies from internal/config/manager.go
- **Simple**: No database schema changes, pure configuration file manipulation

**Technical Highlights**:
- Two predefined profiles: basic (5 statuses) and advanced (19 statuses)
- Smart config merging preserves user customizations
- Atomic file writes prevent corruption
- Backward compatible with existing configs
- No breaking changes to existing commands

---

## Architecture Overview

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLI Layer (Cobra)                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  init.go: `shark init update`                           │  │
│  │  - Parse flags (--workflow, --force, --dry-run)         │  │
│  │  - Validate inputs                                       │  │
│  │  - Call ProfileService                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│              Service Layer (Business Logic)                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ProfileService (internal/init/profile_service.go)      │  │
│  │  - Load profile by name                                  │  │
│  │  - Validate profile                                      │  │
│  │  - Merge profile with existing config                   │  │
│  │  - Generate change report                                │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│              Data Layer (Profiles & Config)                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ProfileRegistry (internal/init/profiles.go)            │  │
│  │  - basicProfile constant                                 │  │
│  │  - advancedProfile constant                              │  │
│  │  - GetProfile(name) -> Profile                           │  │
│  │  - ListProfiles() -> []string                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ConfigMerger (internal/init/config_merger.go)          │  │
│  │  - DeepMerge(base, overlay) -> merged                   │  │
│  │  - DetectChanges(old, new) -> ChangeReport              │  │
│  │  - PreserveFields(config, preserve) -> config           │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ConfigWriter (reuse internal/config/manager.go)        │  │
│  │  - AtomicWrite(path, data)                               │  │
│  │  - CreateBackup(path) -> backupPath                      │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
User Command
    ↓
[Parse Args] --workflow=advanced --dry-run
    ↓
[Load Current Config] .sharkconfig.json → currentConfig
    ↓
[Get Profile] ProfileRegistry.GetProfile("advanced") → profile
    ↓
[Merge Config] ConfigMerger.Merge(currentConfig, profile) → mergedConfig
    ↓
[Detect Changes] ConfigMerger.DetectChanges(current, merged) → changeReport
    ↓
[Dry Run?]
    ├─ Yes → [Display Changes] → Exit
    └─ No  → [Create Backup] → [Write Config] → [Display Report] → Exit
```

---

## Data Structures

### 1. WorkflowProfile (internal/init/types.go)

```go
// WorkflowProfile represents a predefined workflow configuration
type WorkflowProfile struct {
    Name               string                            `json:"name"`
    Description        string                            `json:"description"`
    Version            string                            `json:"version"`
    StatusMetadata     map[string]*StatusMetadata       `json:"status_metadata"`
    StatusFlow         map[string][]string              `json:"status_flow,omitempty"`
    SpecialStatuses    map[string][]string              `json:"special_statuses,omitempty"`
    StatusFlowVersion  string                            `json:"status_flow_version,omitempty"`
}

// StatusMetadata represents metadata for a single status
type StatusMetadata struct {
    Color           string   `json:"color"`                    // ANSI color name
    Phase           string   `json:"phase"`                    // Workflow phase
    ProgressWeight  float64  `json:"progress_weight"`          // 0.0 to 1.0
    Responsibility  string   `json:"responsibility"`           // agent, human, qa_team, none
    BlocksFeature   bool     `json:"blocks_feature"`           // Whether status blocks feature
    AgentTypes      []string `json:"agent_types,omitempty"`    // Allowed agent types
    Description     string   `json:"description,omitempty"`    // Human-readable description
}

// ProfileMetadata provides information about a profile
type ProfileMetadata struct {
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    StatusCount  int      `json:"status_count"`
    HasStatusFlow bool    `json:"has_status_flow"`
    HasSpecialStatuses bool `json:"has_special_statuses"`
    TargetUsers  string   `json:"target_users"`
}
```

### 2. UpdateOptions (internal/init/types.go)

```go
// UpdateOptions represents options for updating config
type UpdateOptions struct {
    ConfigPath     string  // Path to .sharkconfig.json
    WorkflowName   string  // Profile name (basic, advanced, or empty)
    Force          bool    // Overwrite existing status configurations
    DryRun         bool    // Preview changes without applying
    NonInteractive bool    // Skip prompts
    Verbose        bool    // Enable verbose logging
}
```

### 3. UpdateResult (internal/init/types.go)

```go
// UpdateResult represents the result of a config update
type UpdateResult struct {
    Success       bool             `json:"success"`
    ProfileName   string           `json:"profile_name,omitempty"`
    BackupPath    string           `json:"backup_path,omitempty"`
    Changes       *ChangeReport    `json:"changes"`
    ConfigPath    string           `json:"config_path"`
    DryRun        bool             `json:"dry_run"`
}

// ChangeReport details what changed during update
type ChangeReport struct {
    Added       []string         `json:"added"`        // Sections added
    Preserved   []string         `json:"preserved"`    // Sections kept
    Overwritten []string         `json:"overwritten"`  // Sections replaced
    Stats       *ChangeStats     `json:"stats"`        // Detailed statistics
}

// ChangeStats provides detailed change statistics
type ChangeStats struct {
    StatusesAdded     int `json:"statuses_added"`
    FlowsAdded        int `json:"flows_added"`
    GroupsAdded       int `json:"groups_added"`
    FieldsPreserved   int `json:"fields_preserved"`
}
```

### 4. ConfigMergeOptions (internal/init/config_merger.go)

```go
// ConfigMergeOptions controls merge behavior
type ConfigMergeOptions struct {
    PreserveFields   []string  // Fields to never overwrite (e.g., "database", "project_root")
    OverwriteFields  []string  // Fields to replace (e.g., "status_metadata")
    Force            bool      // If true, overwrite even protected fields
}
```

---

## Component Design

### 1. ProfileRegistry (internal/init/profiles.go)

**Purpose**: Centralized storage and retrieval of workflow profiles.

**Responsibilities**:
- Store predefined profiles as Go constants
- Provide profile lookup by name
- List available profiles
- Validate profile structure

**Key Functions**:

```go
// GetProfile retrieves a workflow profile by name
func GetProfile(name string) (*WorkflowProfile, error)

// ListProfiles returns metadata for all available profiles
func ListProfiles() []*ProfileMetadata

// ValidateProfile checks if a profile is well-formed
func ValidateProfile(profile *WorkflowProfile) error
```

**Implementation Notes**:
- Profiles stored as Go constants (compile-time validation)
- Use init() to register profiles in global map
- Validation ensures all required fields present

### 2. ProfileService (internal/init/profile_service.go)

**Purpose**: Orchestrate profile application and config updates.

**Responsibilities**:
- Load current configuration
- Apply workflow profile
- Merge configurations intelligently
- Generate change reports
- Create backups
- Write updated config

**Key Functions**:

```go
// ApplyProfile applies a workflow profile to existing config
func (s *ProfileService) ApplyProfile(opts UpdateOptions) (*UpdateResult, error)

// GetChangePreview shows what would change without applying
func (s *ProfileService) GetChangePreview(opts UpdateOptions) (*ChangeReport, error)

// AddMissingFields adds missing config fields from default profile
func (s *ProfileService) AddMissingFields(opts UpdateOptions) (*UpdateResult, error)
```

**Dependencies**:
- ProfileRegistry (for profile lookup)
- ConfigMerger (for merge logic)
- config.Manager (for config I/O)

### 3. ConfigMerger (internal/init/config_merger.go)

**Purpose**: Smart configuration merging with change tracking.

**Responsibilities**:
- Deep merge two config maps
- Preserve specified fields
- Detect and report changes
- Handle nested structures

**Key Functions**:

```go
// Merge performs a deep merge of overlay into base
// Returns merged config and change report
func Merge(base, overlay map[string]interface{}, opts ConfigMergeOptions) (map[string]interface{}, *ChangeReport, error)

// DeepMerge recursively merges overlay into base
func DeepMerge(base, overlay map[string]interface{}) map[string]interface{}

// DetectChanges compares two configs and reports differences
func DetectChanges(old, new map[string]interface{}) *ChangeReport

// PreserveFields ensures specified fields from base are kept
func PreserveFields(merged, base map[string]interface{}, fields []string) map[string]interface{}
```

**Merge Rules**:
1. **Always Preserve**: `database`, `project_root`, `viewer`, `last_sync_time`
2. **Merge If Empty**: Add missing fields from profile
3. **Overwrite (default)**: `status_metadata`, `status_flow`, `special_statuses`
4. **Force Mode**: Overwrite all except database config

### 4. Command Handler (internal/cli/commands/init.go - extended)

**Purpose**: CLI interface for config update command.

**Responsibilities**:
- Parse command-line flags
- Validate inputs
- Call ProfileService
- Display results

**Command Structure**:

```go
var initUpdateCmd = &cobra.Command{
    Use:   "update [flags]",
    Short: "Update Shark configuration",
    Long:  `Update Shark configuration with workflow profiles or add missing fields.`,
    Example: `  # Add missing fields only
  shark init update

  # Apply basic workflow
  shark init update --workflow=basic

  # Apply advanced workflow with preview
  shark init update --workflow=advanced --dry-run

  # Force overwrite existing config
  shark init update --workflow=basic --force`,
    RunE: runInitUpdate,
}

func init() {
    initCmd.AddCommand(initUpdateCmd)

    initUpdateCmd.Flags().StringVar(&workflowName, "workflow", "",
        "Apply workflow profile (basic, advanced)")
    initUpdateCmd.Flags().BoolVar(&updateForce, "force", false,
        "Overwrite existing status configurations")
    initUpdateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false,
        "Preview changes without applying")
}
```

---

## File-by-File Implementation

### New Files to Create

#### 1. `internal/init/profiles.go`

**Purpose**: Profile registry with predefined workflows.

**Contents**:
- Basic profile constant (5 statuses)
- Advanced profile constant (19 statuses)
- Profile registry map
- GetProfile(), ListProfiles(), ValidateProfile()

**Estimated Lines**: ~600 lines (mostly JSON data)

**Key Sections**:
```go
// Basic Profile
var basicProfile = &WorkflowProfile{
    Name:        "basic",
    Description: "Simple workflow for solo developers",
    Version:     "1.0",
    StatusMetadata: map[string]*StatusMetadata{
        "todo": {
            Color:          "gray",
            Phase:          "planning",
            ProgressWeight: 0.0,
            Responsibility: "none",
            BlocksFeature:  false,
            Description:    "Task not started",
        },
        // ... 4 more statuses
    },
}

// Advanced Profile
var advancedProfile = &WorkflowProfile{
    Name:        "advanced",
    Description: "Comprehensive TDD workflow for teams",
    Version:     "1.0",
    StatusMetadata: map[string]*StatusMetadata{
        // ... 19 statuses
    },
    StatusFlow: map[string][]string{
        // ... flow definitions
    },
    SpecialStatuses: map[string][]string{
        "_start_":    {"draft", "ready_for_development"},
        "_complete_": {"completed", "cancelled"},
        "_blocked_":  {"blocked", "on_hold"},
    },
    StatusFlowVersion: "1.0",
}

// Registry
var profileRegistry = map[string]*WorkflowProfile{
    "basic":    basicProfile,
    "advanced": advancedProfile,
}

func GetProfile(name string) (*WorkflowProfile, error) {
    profile, exists := profileRegistry[strings.ToLower(name)]
    if !exists {
        return nil, fmt.Errorf("profile not found: %s (available: %s)",
            name, strings.Join(ListProfileNames(), ", "))
    }
    return profile, nil
}
```

#### 2. `internal/init/profile_service.go`

**Purpose**: Business logic for applying profiles.

**Contents**:
- ProfileService struct
- ApplyProfile() method
- GetChangePreview() method
- AddMissingFields() method

**Estimated Lines**: ~250 lines

**Key Implementation**:
```go
type ProfileService struct {
    configManager *config.Manager
    merger        *ConfigMerger
}

func NewProfileService(configPath string) *ProfileService {
    return &ProfileService{
        configManager: config.NewManager(configPath),
        merger:        NewConfigMerger(),
    }
}

func (s *ProfileService) ApplyProfile(opts UpdateOptions) (*UpdateResult, error) {
    // 1. Load current config
    currentConfig, err := s.configManager.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    // 2. Get profile
    var profile *WorkflowProfile
    if opts.WorkflowName != "" {
        profile, err = GetProfile(opts.WorkflowName)
        if err != nil {
            return nil, err
        }
    } else {
        // Use default profile for adding missing fields
        profile = basicProfile
    }

    // 3. Merge configs
    mergeOpts := ConfigMergeOptions{
        PreserveFields:  []string{"database", "project_root", "viewer", "last_sync_time"},
        OverwriteFields: []string{"status_metadata", "status_flow", "special_statuses"},
        Force:           opts.Force,
    }

    mergedConfig, changeReport, err := s.merger.Merge(
        currentConfig.RawData,
        profileToMap(profile),
        mergeOpts,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to merge config: %w", err)
    }

    // 4. Dry run? Return preview
    if opts.DryRun {
        return &UpdateResult{
            Success:     true,
            ProfileName: profile.Name,
            Changes:     changeReport,
            ConfigPath:  opts.ConfigPath,
            DryRun:      true,
        }, nil
    }

    // 5. Create backup
    backupPath, err := createConfigBackup(opts.ConfigPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create backup: %w", err)
    }

    // 6. Write merged config (atomic)
    if err := s.writeConfig(opts.ConfigPath, mergedConfig); err != nil {
        return nil, fmt.Errorf("failed to write config: %w", err)
    }

    return &UpdateResult{
        Success:     true,
        ProfileName: profile.Name,
        BackupPath:  backupPath,
        Changes:     changeReport,
        ConfigPath:  opts.ConfigPath,
        DryRun:      false,
    }, nil
}
```

#### 3. `internal/init/config_merger.go`

**Purpose**: Configuration merge logic with change tracking.

**Contents**:
- ConfigMerger struct
- Merge() method with deep merge logic
- DetectChanges() method
- Helper functions for nested map operations

**Estimated Lines**: ~300 lines

**Key Implementation**:
```go
type ConfigMerger struct {
    // No state needed
}

func NewConfigMerger() *ConfigMerger {
    return &ConfigMerger{}
}

// Merge performs intelligent merge of overlay into base
func (m *ConfigMerger) Merge(base, overlay map[string]interface{}, opts ConfigMergeOptions) (map[string]interface{}, *ChangeReport, error) {
    // 1. Deep copy base to avoid mutations
    merged := deepCopy(base)

    // 2. Track changes
    report := &ChangeReport{
        Added:       []string{},
        Preserved:   []string{},
        Overwritten: []string{},
        Stats:       &ChangeStats{},
    }

    // 3. Process each field in overlay
    for key, value := range overlay {
        // Skip if in preserve list (unless force mode)
        if contains(opts.PreserveFields, key) && !opts.Force {
            report.Preserved = append(report.Preserved, key)
            continue
        }

        // Check if field exists in base
        if _, exists := merged[key]; exists {
            // Field exists - check if we should overwrite
            if contains(opts.OverwriteFields, key) || opts.Force {
                merged[key] = value
                report.Overwritten = append(report.Overwritten, key)
            } else {
                // Merge nested structures
                merged[key] = m.mergeValue(merged[key], value)
                report.Added = append(report.Added, key)
            }
        } else {
            // New field - add it
            merged[key] = value
            report.Added = append(report.Added, key)
        }
    }

    // 4. Calculate statistics
    report.Stats = m.calculateStats(base, merged, overlay)

    return merged, report, nil
}

// mergeValue handles merging of nested values
func (m *ConfigMerger) mergeValue(base, overlay interface{}) interface{} {
    // Handle maps recursively
    baseMap, baseIsMap := base.(map[string]interface{})
    overlayMap, overlayIsMap := overlay.(map[string]interface{})

    if baseIsMap && overlayIsMap {
        return m.DeepMerge(baseMap, overlayMap)
    }

    // For non-maps, overlay wins
    return overlay
}

// DeepMerge recursively merges two maps
func (m *ConfigMerger) DeepMerge(base, overlay map[string]interface{}) map[string]interface{} {
    result := deepCopy(base)

    for key, value := range overlay {
        if baseValue, exists := result[key]; exists {
            result[key] = m.mergeValue(baseValue, value)
        } else {
            result[key] = value
        }
    }

    return result
}

// calculateStats computes detailed change statistics
func (m *ConfigMerger) calculateStats(base, merged, overlay map[string]interface{}) *ChangeStats {
    stats := &ChangeStats{}

    // Count statuses added
    if statusMeta, ok := overlay["status_metadata"].(map[string]interface{}); ok {
        stats.StatusesAdded = len(statusMeta)
    }

    // Count flows added
    if statusFlow, ok := overlay["status_flow"].(map[string]interface{}); ok {
        stats.FlowsAdded = len(statusFlow)
    }

    // Count special status groups
    if specialStatuses, ok := overlay["special_statuses"].(map[string]interface{}); ok {
        stats.GroupsAdded = len(specialStatuses)
    }

    // Count preserved fields
    for key := range base {
        if _, exists := overlay[key]; !exists {
            stats.FieldsPreserved++
        }
    }

    return stats
}
```

#### 4. `internal/init/profiles_test.go`

**Purpose**: Unit tests for profile registry.

**Tests**:
- TestGetProfile_Basic
- TestGetProfile_Advanced
- TestGetProfile_NotFound
- TestListProfiles
- TestValidateProfile_Valid
- TestValidateProfile_Invalid

**Estimated Lines**: ~200 lines

#### 5. `internal/init/profile_service_test.go`

**Purpose**: Unit tests for profile service.

**Tests**:
- TestApplyProfile_Basic
- TestApplyProfile_Advanced
- TestApplyProfile_DryRun
- TestApplyProfile_Force
- TestApplyProfile_PreserveDatabase
- TestAddMissingFields
- TestApplyProfile_InvalidProfile

**Estimated Lines**: ~400 lines

#### 6. `internal/init/config_merger_test.go`

**Purpose**: Unit tests for config merger.

**Tests**:
- TestMerge_AddMissingFields
- TestMerge_PreserveFields
- TestMerge_OverwriteFields
- TestMerge_NestedMaps
- TestDeepMerge
- TestDetectChanges
- TestCalculateStats

**Estimated Lines**: ~350 lines

### Files to Modify

#### 7. `internal/cli/commands/init.go`

**Changes**:
- Add `initUpdateCmd` subcommand
- Add flags: --workflow, --force, --dry-run
- Implement `runInitUpdate()` function
- Add output formatting for update results

**New Code**:
```go
var (
    workflowName  string
    updateForce   bool
    updateDryRun  bool
)

var initUpdateCmd = &cobra.Command{
    Use:   "update [flags]",
    Short: "Update Shark configuration",
    Long:  `Update Shark configuration with workflow profiles or add missing fields.

Without --workflow flag, adds missing configuration fields while preserving
all existing values.

With --workflow flag, applies the specified workflow profile (basic or advanced).

Use --dry-run to preview changes before applying.`,
    Example: `  # Add missing fields only
  shark init update

  # Apply basic workflow (5 statuses)
  shark init update --workflow=basic

  # Apply advanced workflow (19 statuses)
  shark init update --workflow=advanced

  # Preview changes without applying
  shark init update --workflow=advanced --dry-run

  # Force overwrite existing status configurations
  shark init update --workflow=basic --force`,
    RunE: runInitUpdate,
}

func init() {
    cli.RootCmd.AddCommand(initCmd)
    initCmd.AddCommand(initUpdateCmd)

    // Existing init flags
    initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
        "Skip all prompts (use defaults)")
    initCmd.Flags().BoolVar(&initForce, "force", false,
        "Overwrite existing config and templates")

    // Update subcommand flags
    initUpdateCmd.Flags().StringVar(&workflowName, "workflow", "",
        "Apply workflow profile (basic, advanced)")
    initUpdateCmd.Flags().BoolVar(&updateForce, "force", false,
        "Overwrite existing status configurations")
    initUpdateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false,
        "Preview changes without applying")
}

func runInitUpdate(cmd *cobra.Command, args []string) error {
    // Get config path
    configPath := ".sharkconfig.json"

    // Create service
    service := init_pkg.NewProfileService(configPath)

    // Build options
    opts := init_pkg.UpdateOptions{
        ConfigPath:     configPath,
        WorkflowName:   workflowName,
        Force:          updateForce,
        DryRun:         updateDryRun,
        NonInteractive: cli.GlobalConfig.JSON,
        Verbose:        cli.GlobalConfig.Verbose,
    }

    // Apply profile (or add missing fields if no workflow specified)
    result, err := service.ApplyProfile(opts)
    if err != nil {
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "success": false,
                "error":   err.Error(),
            })
        }
        return err
    }

    // Output results
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    displayUpdateResult(result)
    return nil
}

func displayUpdateResult(result *init_pkg.UpdateResult) {
    if result.DryRun {
        cli.Info("DRY RUN - No changes applied")
        fmt.Println()
    }

    if result.ProfileName != "" {
        cli.Success(fmt.Sprintf("Applied %s workflow profile", result.ProfileName))
    } else {
        cli.Success("Updated configuration")
    }
    fmt.Println()

    if result.BackupPath != "" {
        fmt.Printf("✓ Backed up config to %s\n", result.BackupPath)
    }

    // Display changes
    changes := result.Changes
    if len(changes.Added) > 0 {
        fmt.Printf("  Added: %s\n", strings.Join(changes.Added, ", "))
    }
    if len(changes.Overwritten) > 0 {
        fmt.Printf("  Overwritten: %s\n", strings.Join(changes.Overwritten, ", "))
    }
    if len(changes.Preserved) > 0 {
        fmt.Printf("  Preserved: %s\n", strings.Join(changes.Preserved, ", "))
    }

    // Display stats
    if changes.Stats != nil {
        fmt.Println()
        fmt.Printf("  Statuses: %d added\n", changes.Stats.StatusesAdded)
        if changes.Stats.FlowsAdded > 0 {
            fmt.Printf("  Flows: %d added\n", changes.Stats.FlowsAdded)
        }
        if changes.Stats.GroupsAdded > 0 {
            fmt.Printf("  Groups: %d added\n", changes.Stats.GroupsAdded)
        }
        fmt.Printf("  Fields: %d preserved\n", changes.Stats.FieldsPreserved)
    }

    if !result.DryRun {
        fmt.Println()
        fmt.Printf("✓ Config updated: %s\n", result.ConfigPath)
    }
}
```

**Estimated Lines Added**: ~150 lines

#### 8. `internal/init/types.go`

**Changes**:
- Add WorkflowProfile struct
- Add StatusMetadata struct
- Add UpdateOptions struct
- Add UpdateResult struct
- Add ChangeReport struct
- Add ChangeStats struct
- Add ProfileMetadata struct

**Estimated Lines Added**: ~120 lines

---

## Configuration Merge Strategy

### Merge Rules

```
┌─────────────────────────────────────────────────────────────┐
│                    Configuration Merge                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ALWAYS PRESERVE (never overwrite):                         │
│  - database                                                 │
│  - project_root                                             │
│  - viewer                                                   │
│  - last_sync_time                                           │
│                                                             │
│  DEFAULT BEHAVIOR (no --workflow flag):                     │
│  - Add missing fields from default profile                  │
│  - Preserve all existing values                             │
│  - No overwrites                                            │
│                                                             │
│  WITH --workflow FLAG:                                      │
│  - Add missing fields                                       │
│  - Overwrite: status_metadata, status_flow,                │
│                special_statuses, status_flow_version        │
│  - Preserve: everything else (except with --force)          │
│                                                             │
│  WITH --force FLAG:                                         │
│  - Overwrite ALL fields except ALWAYS PRESERVE list         │
│  - Replace entire status configurations                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Merge Algorithm

```go
function Merge(base, overlay, options):
    merged = deepCopy(base)
    report = new ChangeReport()

    for each key, value in overlay:
        // Always preserve database config (unless force)
        if key in PRESERVE_LIST and not options.Force:
            report.Preserved.add(key)
            continue

        // Field exists in base
        if key in merged:
            if key in OVERWRITE_LIST or options.Force:
                // Replace entire field
                merged[key] = value
                report.Overwritten.add(key)
            else:
                // Deep merge nested structures
                merged[key] = mergeValue(merged[key], value)
                report.Added.add(key)
        else:
            // New field - add it
            merged[key] = value
            report.Added.add(key)

    report.Stats = calculateStats(base, merged, overlay)
    return merged, report
```

### Example Scenarios

#### Scenario 1: Fresh Install (Empty Config)

**Input**: `{}`
**Profile**: basic
**Output**: Complete basic profile config
**Report**: Added all fields, 5 statuses

#### Scenario 2: Partial Config (Missing Status Flow)

**Input**:
```json
{
  "database": {"backend": "local", "url": "./shark-tasks.db"},
  "status_metadata": { "todo": {...}, "in_progress": {...} }
}
```

**Profile**: advanced
**Output**:
```json
{
  "database": {"backend": "local", "url": "./shark-tasks.db"},  // PRESERVED
  "status_metadata": { ...19 statuses from advanced... },       // OVERWRITTEN
  "status_flow": { ...advanced flows... },                      // ADDED
  "special_statuses": { ...advanced groups... }                 // ADDED
}
```

**Report**:
- Preserved: database
- Overwritten: status_metadata
- Added: status_flow, special_statuses

#### Scenario 3: No --workflow Flag (Add Missing Only)

**Input**:
```json
{
  "database": {"backend": "local", "url": "./shark-tasks.db"},
  "color_enabled": true
}
```

**Output**:
```json
{
  "database": {"backend": "local", "url": "./shark-tasks.db"},  // PRESERVED
  "color_enabled": true,                                         // PRESERVED
  "status_metadata": { ...basic statuses... },                   // ADDED
  "default_agent": null,                                         // ADDED
  "default_epic": null                                           // ADDED
}
```

**Report**:
- Preserved: database, color_enabled
- Added: status_metadata, default_agent, default_epic

---

## Testing Strategy

### Unit Tests (internal/init/)

#### profiles_test.go
- **TestGetProfile_Basic**: Verify basic profile retrieval
- **TestGetProfile_Advanced**: Verify advanced profile retrieval
- **TestGetProfile_NotFound**: Error handling for invalid profile names
- **TestListProfiles**: Verify all profiles listed
- **TestValidateProfile_Valid**: Profile validation with valid data
- **TestValidateProfile_MissingFields**: Validation errors for incomplete profiles
- **TestValidateProfile_InvalidWeights**: Validation errors for invalid progress weights

#### profile_service_test.go
- **TestApplyProfile_Basic**: Apply basic profile to empty config
- **TestApplyProfile_Advanced**: Apply advanced profile to empty config
- **TestApplyProfile_DryRun**: Verify no changes in dry-run mode
- **TestApplyProfile_Force**: Force overwrite existing config
- **TestApplyProfile_PreserveDatabase**: Database config never overwritten
- **TestApplyProfile_PreserveCustomFields**: Custom fields preserved
- **TestAddMissingFields**: Add missing fields without overwriting
- **TestApplyProfile_InvalidProfile**: Error handling for invalid profile names
- **TestApplyProfile_BackupCreated**: Verify backup file created

#### config_merger_test.go
- **TestMerge_AddMissingFields**: Add new fields to config
- **TestMerge_PreserveFields**: Verify preserve list respected
- **TestMerge_OverwriteFields**: Verify overwrite list respected
- **TestMerge_NestedMaps**: Deep merge of nested structures
- **TestMerge_Force**: Force mode overwrites everything
- **TestDeepMerge_Simple**: Simple map merge
- **TestDeepMerge_Nested**: Nested map merge
- **TestDetectChanges**: Change detection accuracy
- **TestCalculateStats**: Statistics calculation
- **TestMerge_EmptyBase**: Merge into empty config
- **TestMerge_EmptyOverlay**: Merge empty overlay

### Integration Tests (internal/cli/commands/)

#### init_update_test.go
- **TestInitUpdate_BasicProfile**: Full workflow for basic profile
- **TestInitUpdate_AdvancedProfile**: Full workflow for advanced profile
- **TestInitUpdate_NoWorkflowFlag**: Add missing fields only
- **TestInitUpdate_DryRun**: Verify dry-run doesn't write
- **TestInitUpdate_Force**: Force overwrite existing
- **TestInitUpdate_MissingConfig**: Create config if doesn't exist
- **TestInitUpdate_JSONOutput**: Verify JSON output format
- **TestInitUpdate_BackupRestoration**: Verify backup can restore config

### Test Coverage Goals

- **Unit Tests**: >85% coverage
- **Integration Tests**: All command paths tested
- **Error Paths**: All error conditions tested
- **Edge Cases**: Empty configs, partial configs, corrupted configs

### Test Data

Create test fixtures in `internal/init/testdata/`:
- `empty_config.json` - Empty config file
- `partial_config.json` - Config with only database
- `full_basic_config.json` - Complete basic profile config
- `full_advanced_config.json` - Complete advanced profile config
- `custom_config.json` - Config with custom status metadata

---

## Implementation Phases

### Phase 1: Data Structures & Profile Registry (2-3 hours)

**Files**:
- `internal/init/types.go` (create structs)
- `internal/init/profiles.go` (create profiles)
- `internal/init/profiles_test.go`

**Deliverables**:
- WorkflowProfile struct
- StatusMetadata struct
- Basic profile constant
- Advanced profile constant
- Profile registry with GetProfile(), ListProfiles()
- Unit tests for profile retrieval

**Success Criteria**:
- ✅ GetProfile("basic") returns basic profile
- ✅ GetProfile("advanced") returns advanced profile
- ✅ GetProfile("invalid") returns error
- ✅ ListProfiles() returns 2 profiles
- ✅ All tests pass

### Phase 2: Config Merger (3-4 hours)

**Files**:
- `internal/init/config_merger.go`
- `internal/init/config_merger_test.go`

**Deliverables**:
- ConfigMerger struct
- Merge() function with deep merge logic
- DetectChanges() function
- Helper functions (deepCopy, contains, etc.)
- Comprehensive unit tests

**Success Criteria**:
- ✅ Merge preserves database config
- ✅ Merge overwrites status_metadata when specified
- ✅ Merge adds missing fields
- ✅ Deep merge works for nested maps
- ✅ Change detection accurate
- ✅ Statistics calculation correct
- ✅ All tests pass (>85% coverage)

### Phase 3: Profile Service (3-4 hours)

**Files**:
- `internal/init/profile_service.go`
- `internal/init/profile_service_test.go`

**Deliverables**:
- ProfileService struct
- ApplyProfile() method
- GetChangePreview() method
- AddMissingFields() method
- Config backup logic
- Unit tests

**Success Criteria**:
- ✅ ApplyProfile() applies basic profile
- ✅ ApplyProfile() applies advanced profile
- ✅ Dry-run doesn't write files
- ✅ Backup created before write
- ✅ AddMissingFields() preserves all existing values
- ✅ All tests pass (>85% coverage)

### Phase 4: CLI Command (2-3 hours)

**Files**:
- `internal/cli/commands/init.go` (modify)
- `internal/cli/commands/init_update_test.go` (create)

**Deliverables**:
- `shark init update` subcommand
- Flags: --workflow, --force, --dry-run
- runInitUpdate() function
- Output formatting (human and JSON)
- Integration tests

**Success Criteria**:
- ✅ `shark init update` adds missing fields
- ✅ `shark init update --workflow=basic` applies basic profile
- ✅ `shark init update --workflow=advanced` applies advanced profile
- ✅ `shark init update --dry-run` shows preview
- ✅ `shark init update --force` overwrites existing
- ✅ `--json` flag outputs valid JSON
- ✅ All integration tests pass

### Phase 5: Documentation (1-2 hours)

**Files**:
- `docs/CLI_REFERENCE.md` (update)
- `docs/guides/workflow-profiles.md` (create)
- `CLAUDE.md` (update with agent status routing)

**Deliverables**:
- Command reference documentation
- Workflow profiles guide
- Example configs for both profiles
- Advanced workflow diagram

**Success Criteria**:
- ✅ CLI_REFERENCE.md updated with `shark init update`
- ✅ Workflow profiles guide created
- ✅ Example configs provided
- ✅ CLAUDE.md updated

### Phase 6: Integration Testing & Polish (2-3 hours)

**Files**:
- All previous files (refinements)

**Deliverables**:
- End-to-end testing
- Error message improvements
- Edge case handling
- Performance optimization

**Success Criteria**:
- ✅ All tests pass (unit + integration)
- ✅ Test coverage >85%
- ✅ Error messages helpful and actionable
- ✅ Command executes in <100ms
- ✅ No memory leaks
- ✅ Backward compatible with existing configs

---

## Risk Assessment

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Config corruption during write** | Low | High | Atomic write (temp file + rename), automatic backup before changes |
| **Merge logic bugs** | Medium | Medium | Comprehensive unit tests, dry-run mode for preview |
| **Profile validation errors** | Low | Medium | Compile-time validation (Go constants), runtime validation before apply |
| **Backward compatibility break** | Low | High | Preserve unknown fields in RawData, never modify database config |
| **JSON marshaling errors** | Low | Medium | Use same encoder as existing code, test with various configs |
| **File permission issues** | Low | Low | Inherit permissions from existing config, clear error messages |

### Mitigation Strategies

1. **Atomic Writes**: Use temp file + rename pattern (already in codebase)
2. **Automatic Backup**: Create `.sharkconfig.json.backup` before any writes
3. **Dry-Run Mode**: Allow users to preview changes without applying
4. **Validation**: Validate profile structure before applying
5. **Preserve Unknown Fields**: Use RawData map to preserve fields we don't know about
6. **Comprehensive Testing**: >85% test coverage with unit + integration tests

### Rollback Plan

If config is corrupted:
1. Automatic backup created before write: `.sharkconfig.json.backup`
2. User can restore: `cp .sharkconfig.json.backup .sharkconfig.json`
3. Error messages guide users to restore from backup

### Edge Cases

1. **Empty Config**: Create complete config from profile
2. **Partial Config**: Merge profile into partial config
3. **Corrupted JSON**: Return clear error message, don't write
4. **Missing Config File**: Create new config with profile
5. **Permission Denied**: Return error with guidance
6. **Concurrent Writes**: Atomic rename prevents corruption

---

## Performance Considerations

### Benchmarks

| Operation | Target Time | Notes |
|-----------|------------|-------|
| Load config | <10ms | Read + parse JSON |
| Get profile | <1ms | Lookup in map |
| Merge config | <5ms | Deep copy + merge |
| Detect changes | <5ms | Compare maps |
| Write config | <10ms | Marshal + atomic write |
| **Total** | **<100ms** | End-to-end command execution |

### Optimization Strategies

1. **Lazy Profile Loading**: Profiles defined as constants (no I/O)
2. **Minimal Allocations**: Reuse existing config.Manager
3. **Atomic Write**: Single write operation (no multiple passes)
4. **Efficient JSON**: Use json.Encoder with streaming

### Memory Usage

- **Profile Registry**: ~50KB (two profiles in memory)
- **Config Files**: ~20KB (typical advanced config)
- **Merge Operation**: ~100KB peak (deep copy + merge)
- **Total**: ~200KB peak memory usage

**Acceptable**: No performance concerns for typical configs.

### Scalability

- **Profile Count**: O(1) lookup in map (scales to 100+ profiles)
- **Config Size**: O(n) merge where n = fields (typical n < 100)
- **Status Count**: O(1) per status (typical count = 5-19)

**Verdict**: No scalability concerns for foreseeable use cases.

---

## Security Considerations

### Threat Model

1. **Config Tampering**: User manually edits config with invalid JSON
   - **Mitigation**: JSON parsing errors caught and reported clearly

2. **Path Traversal**: Malicious config path (e.g., `../../etc/passwd`)
   - **Mitigation**: Use filepath.Clean(), validate path is within project

3. **Permission Escalation**: Config file created with wrong permissions
   - **Mitigation**: Inherit permissions from existing config (0644 default)

4. **Backup Overwrite**: Backup file already exists
   - **Mitigation**: Use timestamped backup names (`.sharkconfig.json.backup.{timestamp}`)

### Input Validation

```go
// Validate workflow name
func validateWorkflowName(name string) error {
    if name == "" {
        return nil // Empty is valid (means no workflow)
    }

    validNames := []string{"basic", "advanced"}
    for _, valid := range validNames {
        if strings.EqualFold(name, valid) {
            return nil
        }
    }

    return fmt.Errorf("invalid workflow name: %s (valid: %s)",
        name, strings.Join(validNames, ", "))
}

// Validate config path
func validateConfigPath(path string) error {
    cleanPath := filepath.Clean(path)

    // Must be .json file
    if !strings.HasSuffix(cleanPath, ".json") {
        return fmt.Errorf("config file must be .json: %s", path)
    }

    // Must not traverse outside project
    if strings.Contains(cleanPath, "..") {
        return fmt.Errorf("config path must not traverse directories: %s", path)
    }

    return nil
}
```

---

## Backward Compatibility

### Compatibility Matrix

| Existing Config Version | Feature Support | Notes |
|------------------------|----------------|-------|
| **No config** | ✅ Full | Create config from profile |
| **v1.0 (legacy)** | ✅ Full | Merge profile, preserve existing |
| **v2.0 (current)** | ✅ Full | Native support |
| **Custom status metadata** | ✅ Full | Preserved unless --force |
| **Cloud database (Turso)** | ✅ Full | Database config always preserved |
| **Custom viewer** | ✅ Full | Viewer config always preserved |

### Breaking Changes

**NONE**: This feature is purely additive.

- No changes to database schema
- No changes to existing command behavior
- Existing configs continue to work
- Unknown fields preserved in RawData

### Migration Path

**Not Required**: Feature is backward compatible.

Users can:
1. Continue using existing configs (no changes needed)
2. Run `shark init update` to add missing fields
3. Run `shark init update --workflow=basic` to standardize on basic workflow
4. Run `shark init update --workflow=advanced` to adopt advanced workflow

---

## Future Enhancements

### Potential Extensions (Post-MVP)

1. **Custom Profiles**
   - User-defined profiles in `~/.shark/profiles/`
   - Profile export: `shark init export --workflow=my-custom`
   - Profile import: `shark init import --file=./custom-profile.json`

2. **Profile Validation**
   - `shark init validate` command
   - Detect invalid status metadata
   - Suggest fixes for common issues

3. **Interactive Profile Selection**
   - `shark init update --interactive`
   - Guided wizard for workflow selection
   - Preview changes before applying

4. **Profile Repository**
   - Community-contributed profiles
   - `shark profile search <keyword>`
   - `shark profile install <name>`

5. **Profile Versioning**
   - Semantic versioning for profiles
   - Migration scripts for profile updates
   - Changelog for profile changes

6. **Profile Inheritance**
   - Base profiles with extensions
   - `extends: "basic"` in custom profiles
   - Override specific statuses

---

## Appendix A: Basic Profile Definition

```json
{
  "name": "basic",
  "description": "Simple workflow for solo developers",
  "version": "1.0",
  "status_metadata": {
    "todo": {
      "color": "gray",
      "phase": "planning",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": false,
      "description": "Task not started"
    },
    "in_progress": {
      "color": "yellow",
      "phase": "development",
      "progress_weight": 0.5,
      "responsibility": "agent",
      "blocks_feature": false,
      "description": "Work in progress"
    },
    "ready_for_review": {
      "color": "magenta",
      "phase": "review",
      "progress_weight": 0.75,
      "responsibility": "human",
      "blocks_feature": true,
      "description": "Awaiting human review"
    },
    "completed": {
      "color": "green",
      "phase": "done",
      "progress_weight": 1.0,
      "responsibility": "none",
      "blocks_feature": false,
      "description": "Task finished"
    },
    "blocked": {
      "color": "red",
      "phase": "any",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": true,
      "description": "Blocked by external dependency"
    }
  }
}
```

**Status Flow** (5 statuses):
```
todo → in_progress → ready_for_review → completed
                   ↘ blocked ↗
```

---

## Appendix B: Advanced Profile Definition

See current `.sharkconfig.json` in project root for full advanced profile structure:
- 16 statuses (draft, ready_for_refinement, in_refinement, ready_for_development, in_development, ready_for_code_review, in_code_review, ready_for_qa, in_qa, ready_for_approval, in_approval, completed, blocked, cancelled, on_hold)
- Status flow definitions for all transitions
- Special status groups (_start_, _complete_, _blocked_)
- Agent type assignments (ba, tech_lead, developer, qa, product_owner)
- Progress weights for granular progress tracking

**Status Flow** (19 statuses):
```
draft → ready_for_refinement → in_refinement →
ready_for_development → in_development →
ready_for_code_review → in_code_review →
ready_for_qa → in_qa →
ready_for_approval → in_approval → completed

blocked (from any status)
cancelled (terminal)
on_hold (suspended)
```

---

## Appendix C: Change Report Example

### Example 1: Apply Basic Profile to Empty Config

**Input**: `{}`

**Output**:
```json
{
  "success": true,
  "profile_name": "basic",
  "backup_path": ".sharkconfig.json.backup",
  "changes": {
    "added": ["status_metadata", "color_enabled", "default_epic", "default_agent"],
    "preserved": [],
    "overwritten": [],
    "stats": {
      "statuses_added": 5,
      "flows_added": 0,
      "groups_added": 0,
      "fields_preserved": 0
    }
  },
  "config_path": ".sharkconfig.json",
  "dry_run": false
}
```

### Example 2: Apply Advanced Profile to Existing Config

**Input**:
```json
{
  "database": {"backend": "local", "url": "./shark-tasks.db"},
  "status_metadata": {"todo": {...}, "in_progress": {...}}
}
```

**Output**:
```json
{
  "success": true,
  "profile_name": "advanced",
  "backup_path": ".sharkconfig.json.backup",
  "changes": {
    "added": ["status_flow", "special_statuses", "status_flow_version"],
    "preserved": ["database"],
    "overwritten": ["status_metadata"],
    "stats": {
      "statuses_added": 19,
      "flows_added": 17,
      "groups_added": 3,
      "fields_preserved": 1
    }
  },
  "config_path": ".sharkconfig.json",
  "dry_run": false
}
```

---

## Appendix D: File Structure Summary

```
internal/init/
├── config.go              (existing - config creation)
├── config_test.go         (existing)
├── database.go            (existing)
├── database_test.go       (existing)
├── errors.go              (existing)
├── folders.go             (existing)
├── folders_test.go        (existing)
├── initializer.go         (existing)
├── initializer_test.go    (existing)
├── templates.go           (existing)
├── templates_test.go      (existing)
├── types.go               (existing - UPDATE: add new structs)
│
├── profiles.go            (NEW - profile registry)
├── profiles_test.go       (NEW - profile tests)
├── profile_service.go     (NEW - business logic)
├── profile_service_test.go (NEW - service tests)
├── config_merger.go       (NEW - merge logic)
└── config_merger_test.go  (NEW - merger tests)

internal/cli/commands/
├── init.go                (MODIFY - add subcommand)
└── init_update_test.go    (NEW - integration tests)

docs/
├── CLI_REFERENCE.md       (UPDATE - add command docs)
└── guides/
    └── workflow-profiles.md (NEW - workflow guide)
```

**Summary**:
- **New Files**: 6 (profiles.go, profile_service.go, config_merger.go + 3 test files)
- **Modified Files**: 2 (init.go, types.go)
- **Documentation**: 2 updates
- **Total Estimated Lines**: ~2,400 lines (including tests)

---

## Glossary

- **Profile**: Predefined workflow configuration with status metadata
- **Basic Profile**: Simple 5-status workflow for solo developers
- **Advanced Profile**: Comprehensive 19-status TDD workflow for teams
- **Config Merge**: Intelligent combining of two configurations
- **Atomic Write**: Write operation that either fully succeeds or fully fails
- **Dry Run**: Preview mode that doesn't apply changes
- **Force Mode**: Override protection and overwrite existing values
- **Change Report**: Summary of what changed during config update
- **Status Metadata**: Configuration for a single status (color, phase, weight, etc.)
- **Status Flow**: Valid transitions between statuses
- **Special Status Groups**: Named groups of statuses (e.g., _start_, _complete_)

---

**End of Technical Design Document**
