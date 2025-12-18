# Backend Design: Import Reporting & Validation

**Feature**: E06-F05-import-reporting-validation
**Epic**: E06-intelligent-scanning
**Status**: Draft
**Last Updated**: 2025-12-17

---

## Table of Contents

1. [Overview](#overview)
2. [Component Design](#component-design)
3. [Data Structures](#data-structures)
4. [Algorithms](#algorithms)
5. [Implementation Details](#implementation-details)
6. [Error Handling](#error-handling)
7. [Testing Approach](#testing-approach)

---

## Overview

### Purpose

This document describes the backend implementation details for the import reporting and validation feature. It provides concrete specifications for developers implementing the Reporter, Validator, and Formatter components.

### Design Principles

1. **Extend, Don't Replace**: Build on existing sync engine patterns
2. **Event-Driven Reporting**: Collect events during scan, generate report at end
3. **Read-Only Validation**: Never modify data during validation
4. **Multi-Format Output**: Support both human (CLI) and machine (JSON) consumption

### Component Summary

| Component | File | Responsibility |
|-----------|------|----------------|
| Reporter | `internal/sync/reporter.go` | Collect scan events, generate reports |
| Validator | `internal/sync/validator.go` | Database integrity validation |
| Formatter | `internal/sync/formatter.go` | Multi-format output rendering |
| Types | `internal/sync/types.go` | Extended data structures |

---

## Component Design

### 1. Reporter Component

**Location**: `internal/sync/reporter.go`

**Purpose**: Collect events during scan workflow and produce comprehensive reports.

#### Interface Design

```go
package sync

import (
    "time"
)

// Reporter collects events during scan and generates reports
type Reporter struct {
    // Metadata
    startTime       time.Time
    documentRoot    string
    validationLevel string
    patterns        PatternConfig

    // Counters
    filesScanned    int
    filesMatched    map[EntityType]int
    filesSkipped    map[EntityType]int
    entitiesImported map[EntityType]int
    entitiesUpdated  map[EntityType]int

    // Event collections
    errors   []ErrorDetail
    warnings []ErrorDetail
    conflicts []Conflict

    // State
    isDryRun bool
}

// NewReporter creates a new reporter
func NewReporter(options SyncOptions) *Reporter {
    return &Reporter{
        startTime:        time.Now(),
        documentRoot:     options.FolderPath,
        validationLevel:  options.ValidationLevel,
        patterns:         options.Patterns,
        filesMatched:     make(map[EntityType]int),
        filesSkipped:     make(map[EntityType]int),
        entitiesImported: make(map[EntityType]int),
        entitiesUpdated:  make(map[EntityType]int),
        errors:           []ErrorDetail{},
        warnings:         []ErrorDetail{},
        conflicts:        []Conflict{},
        isDryRun:         options.DryRun,
    }
}

// Event methods (called by sync engine)
func (r *Reporter) ScanStarted(timestamp time.Time, config SyncOptions)
func (r *Reporter) FileDiscovered(path string)
func (r *Reporter) FileMatched(path string, entityType EntityType, key string)
func (r *Reporter) FileSkipped(path string, entityType EntityType, reason SkipReason, suggestion string)
func (r *Reporter) ParseError(path string, line int, errorType ErrorType, message string, suggestion string)
func (r *Reporter) ValidationWarning(path string, issue string, severity Severity, suggestion string)
func (r *Reporter) EntityImported(entityType EntityType, key string)
func (r *Reporter) EntityUpdated(entityType EntityType, key string, fields []string)
func (r *Reporter) ConflictDetected(conflict Conflict)
func (r *Reporter) ConflictResolved(conflict Conflict, resolution string)

// Report generation
func (r *Reporter) GenerateReport() *ScanReport
```

#### Implementation Details

**Event Collection Pattern**:

```go
func (r *Reporter) FileSkipped(path string, entityType EntityType, reason SkipReason, suggestion string) {
    r.filesSkipped[entityType]++
    r.filesScanned++

    r.errors = append(r.errors, ErrorDetail{
        Severity:      SeverityError,
        ErrorType:     ErrorType(reason),
        FilePath:      path,
        LineNumber:    nil,
        Message:       reason.Message(),
        SuggestedFix:  suggestion,
    })
}

func (r *Reporter) ParseError(path string, line int, errorType ErrorType, message string, suggestion string) {
    r.filesScanned++

    r.errors = append(r.errors, ErrorDetail{
        Severity:      SeverityError,
        ErrorType:     errorType,
        FilePath:      path,
        LineNumber:    &line,
        Message:       message,
        SuggestedFix:  suggestion,
    })
}

func (r *Reporter) ValidationWarning(path string, issue string, severity Severity, suggestion string) {
    warning := ErrorDetail{
        Severity:      severity,
        ErrorType:     ErrorTypeValidationFailure,
        FilePath:      path,
        LineNumber:    nil,
        Message:       issue,
        SuggestedFix:  suggestion,
    }

    if severity == SeverityWarning {
        r.warnings = append(r.warnings, warning)
    } else {
        r.errors = append(r.errors, warning)
    }
}
```

**Report Generation**:

```go
func (r *Reporter) GenerateReport() *ScanReport {
    duration := time.Since(r.startTime).Seconds()

    report := &ScanReport{
        SchemaVersion: "1.0",
        Status:        r.determineStatus(),
        DryRun:        r.isDryRun,
        Metadata: ScanMetadata{
            Timestamp:         r.startTime,
            DurationSeconds:   duration,
            ValidationLevel:   r.validationLevel,
            DocumentationRoot: r.documentRoot,
            Patterns:          r.patterns,
        },
        Counts: ScanCounts{
            Scanned: r.filesScanned,
            Matched: r.sumMatched(),
            Skipped: r.sumSkipped(),
        },
        Entities: EntityBreakdown{
            Epics:        EntityCount{Matched: r.filesMatched[EntityTypeEpic], Skipped: r.filesSkipped[EntityTypeEpic]},
            Features:     EntityCount{Matched: r.filesMatched[EntityTypeFeature], Skipped: r.filesSkipped[EntityTypeFeature]},
            Tasks:        EntityCount{Matched: r.filesMatched[EntityTypeTask], Skipped: r.filesSkipped[EntityTypeTask]},
            RelatedDocs:  EntityCount{Matched: r.filesMatched[EntityTypeRelatedDoc], Skipped: r.filesSkipped[EntityTypeRelatedDoc]},
        },
        Errors:   r.errors,
        Warnings: r.warnings,
        Conflicts: r.conflicts,
        Summary: ScanSummary{
            Imported: r.sumImported(),
            Errors:   len(r.errors),
            Warnings: len(r.warnings),
        },
    }

    return report
}

func (r *Reporter) determineStatus() string {
    if len(r.errors) > 0 {
        return "failure"
    }
    if len(r.warnings) > 0 {
        return "success_with_warnings"
    }
    return "success"
}

func (r *Reporter) sumMatched() int {
    sum := 0
    for _, count := range r.filesMatched {
        sum += count
    }
    return sum
}

func (r *Reporter) sumSkipped() int {
    sum := 0
    for _, count := range r.filesSkipped {
        sum += count
    }
    return sum
}

func (r *Reporter) sumImported() int {
    sum := 0
    for _, count := range r.entitiesImported {
        sum += count
    }
    return sum
}
```

### 2. Validator Component

**Location**: `internal/sync/validator.go`

**Purpose**: Verify database integrity by checking file paths and relationships.

#### Interface Design

```go
package sync

import (
    "context"
    "os"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Validator performs database integrity checks
type Validator struct {
    epicRepo    *repository.EpicRepository
    featureRepo *repository.FeatureRepository
    taskRepo    *repository.TaskRepository
}

// NewValidator creates a new validator
func NewValidator(epicRepo *repository.EpicRepository, featureRepo *repository.FeatureRepository, taskRepo *repository.TaskRepository) *Validator {
    return &Validator{
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        taskRepo:    taskRepo,
    }
}

// Validate performs all validation checks
func (v *Validator) Validate(ctx context.Context) (*ValidationReport, error)

// Individual check methods
func (v *Validator) ValidateFilePaths(ctx context.Context) ([]ValidationFailure, error)
func (v *Validator) ValidateRelationships(ctx context.Context) ([]ValidationFailure, error)
func (v *Validator) ValidateBrokenReferences(ctx context.Context) ([]ValidationFailure, error)
```

#### Implementation Details

**File Path Validation**:

```go
func (v *Validator) ValidateFilePaths(ctx context.Context) ([]ValidationFailure, error) {
    failures := []ValidationFailure{}

    // Check epics
    epics, err := v.epicRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list epics: %w", err)
    }

    for _, epic := range epics {
        if epic.FilePath != nil && *epic.FilePath != "" {
            if _, err := os.Stat(*epic.FilePath); os.IsNotExist(err) {
                failures = append(failures, ValidationFailure{
                    CheckType:   "file_path_existence",
                    EntityType:  "epic",
                    EntityKey:   epic.Key,
                    FilePath:    *epic.FilePath,
                    Issue:       "File not found (may have been moved or deleted)",
                    SuggestedFix: fmt.Sprintf("Re-scan to update file paths or check if file was moved"),
                })
            }
        }
    }

    // Check features
    features, err := v.featureRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list features: %w", err)
    }

    for _, feature := range features {
        if feature.FilePath != nil && *feature.FilePath != "" {
            if _, err := os.Stat(*feature.FilePath); os.IsNotExist(err) {
                failures = append(failures, ValidationFailure{
                    CheckType:   "file_path_existence",
                    EntityType:  "feature",
                    EntityKey:   feature.Key,
                    FilePath:    *feature.FilePath,
                    Issue:       "File not found (may have been moved or deleted)",
                    SuggestedFix: "Re-scan to update file paths or verify file location",
                })
            }
        }
    }

    // Check tasks
    tasks, err := v.taskRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list tasks: %w", err)
    }

    for _, task := range tasks {
        if task.FilePath != nil && *task.FilePath != "" {
            if _, err := os.Stat(*task.FilePath); os.IsNotExist(err) {
                failures = append(failures, ValidationFailure{
                    CheckType:   "file_path_existence",
                    EntityType:  "task",
                    EntityKey:   task.Key,
                    FilePath:    *task.FilePath,
                    Issue:       "File not found (may have been moved or deleted)",
                    SuggestedFix: fmt.Sprintf("Update path or delete stale task: shark task delete %s", task.Key),
                })
            }
        }
    }

    return failures, nil
}
```

**Relationship Validation**:

```go
func (v *Validator) ValidateRelationships(ctx context.Context) ([]ValidationFailure, error) {
    failures := []ValidationFailure{}

    // Build epic key map for O(1) lookups
    epics, err := v.epicRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list epics: %w", err)
    }

    epicKeys := make(map[string]bool, len(epics))
    for _, epic := range epics {
        epicKeys[epic.Key] = true
    }

    // Validate features reference existing epics
    features, err := v.featureRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list features: %w", err)
    }

    featureKeys := make(map[string]bool, len(features))
    for _, feature := range features {
        featureKeys[feature.Key] = true

        if !epicKeys[feature.EpicKey] {
            failures = append(failures, ValidationFailure{
                CheckType:        "relationship_integrity",
                EntityType:       "feature",
                EntityKey:        feature.Key,
                MissingParentType: "epic",
                MissingParentKey:  feature.EpicKey,
                Issue:            "Orphaned feature: parent epic does not exist",
                SuggestedFix:     fmt.Sprintf("Create epic %s or reassign feature to existing epic", feature.EpicKey),
            })
        }
    }

    // Validate tasks reference existing features
    tasks, err := v.taskRepo.List(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list tasks: %w", err)
    }

    for _, task := range tasks {
        if !featureKeys[task.FeatureKey] {
            failures = append(failures, ValidationFailure{
                CheckType:        "relationship_integrity",
                EntityType:       "task",
                EntityKey:        task.Key,
                MissingParentType: "feature",
                MissingParentKey:  task.FeatureKey,
                Issue:            "Orphaned task: parent feature does not exist",
                SuggestedFix:     fmt.Sprintf("Create feature %s or delete orphaned task: shark task delete %s", task.FeatureKey, task.Key),
            })
        }
    }

    return failures, nil
}
```

**Report Generation**:

```go
func (v *Validator) Validate(ctx context.Context) (*ValidationReport, error) {
    startTime := time.Now()

    // Collect all failures
    allFailures := []ValidationFailure{}

    // File path checks
    filePathFailures, err := v.ValidateFilePaths(ctx)
    if err != nil {
        return nil, err
    }
    allFailures = append(allFailures, filePathFailures...)

    // Relationship checks
    relationshipFailures, err := v.ValidateRelationships(ctx)
    if err != nil {
        return nil, err
    }
    allFailures = append(allFailures, relationshipFailures...)

    // Count totals
    epics, _ := v.epicRepo.List(ctx)
    features, _ := v.featureRepo.List(ctx)
    tasks, _ := v.taskRepo.List(ctx)
    totalValidated := len(epics) + len(features) + len(tasks)

    // Count failures by type
    brokenPaths := 0
    orphanedFeatures := 0
    orphanedTasks := 0

    for _, failure := range allFailures {
        if failure.CheckType == "file_path_existence" {
            brokenPaths++
        } else if failure.CheckType == "relationship_integrity" {
            if failure.EntityType == "feature" {
                orphanedFeatures++
            } else if failure.EntityType == "task" {
                orphanedTasks++
            }
        }
    }

    // Build report
    report := &ValidationReport{
        SchemaVersion: "1.0",
        Status:        determineValidationStatus(allFailures),
        Metadata: ValidationMetadata{
            Timestamp:    startTime,
            DatabasePath: v.getDatabasePath(),
        },
        ValidationChecks: ValidationChecks{
            FilePaths: FilePathCheck{
                TotalChecked: totalValidated,
                BrokenPaths:  brokenPaths,
            },
            Relationships: RelationshipCheck{
                TotalChecked:     len(features) + len(tasks),
                OrphanedFeatures: orphanedFeatures,
                OrphanedTasks:    orphanedTasks,
            },
        },
        Failures: allFailures,
        Summary: ValidationSummary{
            TotalValidated:   totalValidated,
            TotalIssues:      len(allFailures),
            BrokenFilePaths:  brokenPaths,
            OrphanedRecords:  orphanedFeatures + orphanedTasks,
        },
    }

    return report, nil
}

func determineValidationStatus(failures []ValidationFailure) string {
    if len(failures) > 0 {
        return "failure"
    }
    return "success"
}
```

### 3. Formatter Component

**Location**: `internal/sync/formatter.go`

**Purpose**: Format reports for different output modes (CLI, JSON).

#### Interface Design

```go
package sync

import (
    "encoding/json"
    "fmt"
    "strings"
)

// Formatter provides multi-format output
type Formatter struct {
    useColor bool
}

// NewFormatter creates a new formatter
func NewFormatter(useColor bool) *Formatter {
    return &Formatter{
        useColor: useColor,
    }
}

// Format scan report
func (f *Formatter) FormatScanCLI(report *ScanReport) string
func (f *Formatter) FormatScanJSON(report *ScanReport) (string, error)

// Format validation report
func (f *Formatter) FormatValidationCLI(report *ValidationReport) string
func (f *Formatter) FormatValidationJSON(report *ValidationReport) (string, error)
```

#### Implementation Details

**CLI Formatting for Scan Report**:

```go
func (f *Formatter) FormatScanCLI(report *ScanReport) string {
    var sb strings.Builder

    // Header
    if report.DryRun {
        sb.WriteString(f.colorize("DRY RUN MODE: No database changes will be committed\n\n", colorYellow))
    }

    sb.WriteString("Shark Scan Report\n")
    sb.WriteString("=================\n")
    sb.WriteString(fmt.Sprintf("Scan completed at %s\n", report.Metadata.Timestamp.Format("2006-01-02 15:04:05")))
    sb.WriteString(fmt.Sprintf("Duration: %.1f seconds\n", report.Metadata.DurationSeconds))
    sb.WriteString(fmt.Sprintf("Validation level: %s\n", report.Metadata.ValidationLevel))
    sb.WriteString(fmt.Sprintf("Documentation root: %s\n", report.Metadata.DocumentationRoot))
    sb.WriteString("\n")

    // Summary
    sb.WriteString("Summary\n")
    sb.WriteString("-------\n")
    sb.WriteString(fmt.Sprintf("Total files scanned: %d\n", report.Counts.Scanned))
    sb.WriteString(fmt.Sprintf("  %s Matched: %d\n", f.symbol("✓", colorGreen), report.Counts.Matched))
    sb.WriteString(fmt.Sprintf("  %s Skipped: %d\n", f.symbol("✗", colorRed), report.Counts.Skipped))
    sb.WriteString("\n")

    // Breakdown by type
    sb.WriteString("Breakdown by Type\n")
    sb.WriteString("-----------------\n")
    f.formatEntityBreakdown(&sb, "Epics", report.Entities.Epics)
    f.formatEntityBreakdown(&sb, "Features", report.Entities.Features)
    f.formatEntityBreakdown(&sb, "Tasks", report.Entities.Tasks)
    if report.Entities.RelatedDocs.Matched > 0 || report.Entities.RelatedDocs.Skipped > 0 {
        f.formatEntityBreakdown(&sb, "Related Docs", report.Entities.RelatedDocs)
    }
    sb.WriteString("\n")

    // Errors and warnings
    if len(report.Errors) > 0 || len(report.Warnings) > 0 {
        sb.WriteString("Errors and Warnings\n")
        sb.WriteString("-------------------\n")
        f.formatErrorsByType(&sb, report.Errors, report.Warnings)
        sb.WriteString("\n")
    }

    // Scan complete
    sb.WriteString("Scan Complete\n")
    sb.WriteString("-------------\n")
    sb.WriteString(fmt.Sprintf("Successfully imported %d items:\n", report.Summary.Imported))
    sb.WriteString(fmt.Sprintf("  - %d epics\n", report.Entities.Epics.Matched))
    sb.WriteString(fmt.Sprintf("  - %d features\n", report.Entities.Features.Matched))
    sb.WriteString(fmt.Sprintf("  - %d tasks\n", report.Entities.Tasks.Matched))
    sb.WriteString("\n")

    if !report.DryRun {
        sb.WriteString("Run 'shark validate' to verify database integrity.\n")
    }

    return sb.String()
}

func (f *Formatter) formatEntityBreakdown(sb *strings.Builder, label string, count EntityCount) {
    sb.WriteString(fmt.Sprintf("%s:\n", label))
    sb.WriteString(fmt.Sprintf("  %s Matched: %d\n", f.symbol("✓", colorGreen), count.Matched))
    if count.Skipped > 0 {
        sb.WriteString(fmt.Sprintf("  %s Skipped: %d\n", f.symbol("✗", colorRed), count.Skipped))
    }
}

func (f *Formatter) formatErrorsByType(sb *strings.Builder, errors []ErrorDetail, warnings []ErrorDetail) {
    // Group errors by type
    errorsByType := make(map[ErrorType][]ErrorDetail)
    for _, err := range errors {
        errorsByType[err.ErrorType] = append(errorsByType[err.ErrorType], err)
    }

    // Format parse errors
    if parseErrors, ok := errorsByType[ErrorTypeParseError]; ok {
        sb.WriteString(fmt.Sprintf("\nParse Errors (%d):\n", len(parseErrors)))
        for _, err := range parseErrors {
            f.formatError(sb, err)
        }
    }

    // Format pattern mismatches
    if patternErrors, ok := errorsByType[ErrorTypePatternMismatch]; ok {
        sb.WriteString(fmt.Sprintf("\nPattern Mismatch Warnings (%d):\n", len(patternErrors)))
        for i, err := range patternErrors {
            if i >= 10 {
                sb.WriteString(fmt.Sprintf("  ... (%d more pattern mismatches)\n", len(patternErrors)-10))
                break
            }
            f.formatError(sb, err)
        }
    }

    // Format validation warnings
    if len(warnings) > 0 {
        sb.WriteString(fmt.Sprintf("\nValidation Warnings (%d):\n", len(warnings)))
        for i, warn := range warnings {
            if i >= 10 {
                sb.WriteString(fmt.Sprintf("  ... (%d more warnings)\n", len(warnings)-10))
                break
            }
            f.formatError(sb, warn)
        }
    }
}

func (f *Formatter) formatError(sb *strings.Builder, err ErrorDetail) {
    sb.WriteString(fmt.Sprintf("  %s: ", err.Severity))

    if err.LineNumber != nil {
        sb.WriteString(fmt.Sprintf("Cannot parse in %s:%d\n", err.FilePath, *err.LineNumber))
    } else {
        sb.WriteString(fmt.Sprintf("%s\n", err.FilePath))
    }

    sb.WriteString(fmt.Sprintf("    %s\n", err.Message))
    sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", err.SuggestedFix))
}

func (f *Formatter) symbol(symbol string, color string) string {
    if f.useColor {
        return f.colorize(symbol, color)
    }
    return symbol
}

func (f *Formatter) colorize(text string, color string) string {
    if !f.useColor {
        return text
    }
    return fmt.Sprintf("%s%s%s", color, text, colorReset)
}

// Color constants
const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
)
```

**JSON Formatting**:

```go
func (f *Formatter) FormatScanJSON(report *ScanReport) (string, error) {
    data, err := json.MarshalIndent(report, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal scan report to JSON: %w", err)
    }
    return string(data), nil
}

func (f *Formatter) FormatValidationJSON(report *ValidationReport) (string, error) {
    data, err := json.MarshalIndent(report, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal validation report to JSON: %w", err)
    }
    return string(data), nil
}
```

---

## Data Structures

### Extended SyncReport Structure

**Location**: `internal/sync/types.go`

```go
// ScanReport contains comprehensive scan results
type ScanReport struct {
    SchemaVersion string          `json:"schema_version"`
    Status        string          `json:"status"` // "success", "failure", "success_with_warnings"
    DryRun        bool            `json:"dry_run"`
    Metadata      ScanMetadata    `json:"metadata"`
    Counts        ScanCounts      `json:"counts"`
    Entities      EntityBreakdown `json:"entities"`
    Errors        []ErrorDetail   `json:"errors"`
    Warnings      []ErrorDetail   `json:"warnings"`
    Conflicts     []Conflict      `json:"conflicts,omitempty"`
    Summary       ScanSummary     `json:"summary"`
}

// ScanMetadata contains scan configuration and timing
type ScanMetadata struct {
    Timestamp         time.Time     `json:"timestamp"`
    DurationSeconds   float64       `json:"duration_seconds"`
    ValidationLevel   string        `json:"validation_level"`
    DocumentationRoot string        `json:"documentation_root"`
    Patterns          PatternConfig `json:"patterns"`
}

// PatternConfig describes patterns used for matching
type PatternConfig struct {
    EpicFolder   string `json:"epic_folder"`
    FeatureFolder string `json:"feature_folder"`
    TaskFile     string `json:"task_file"`
}

// ScanCounts provides high-level counts
type ScanCounts struct {
    Scanned int `json:"scanned"`
    Matched int `json:"matched"`
    Skipped int `json:"skipped"`
}

// EntityBreakdown shows counts by entity type
type EntityBreakdown struct {
    Epics       EntityCount `json:"epics"`
    Features    EntityCount `json:"features"`
    Tasks       EntityCount `json:"tasks"`
    RelatedDocs EntityCount `json:"related_docs"`
}

// EntityCount shows matched vs skipped for an entity type
type EntityCount struct {
    Matched int `json:"matched"`
    Skipped int `json:"skipped"`
}

// ErrorDetail provides structured error information
type ErrorDetail struct {
    Severity     Severity   `json:"severity"`      // "ERROR", "WARNING", "INFO"
    ErrorType    ErrorType  `json:"error_type"`    // "parse_error", "pattern_mismatch", etc.
    FilePath     string     `json:"file_path"`     // Absolute path
    LineNumber   *int       `json:"line_number,omitempty"` // Optional line number
    Message      string     `json:"message"`       // Human-readable description
    SuggestedFix string     `json:"suggested_fix"` // Actionable suggestion
}

// ScanSummary provides final totals
type ScanSummary struct {
    Imported int `json:"imported"`
    Errors   int `json:"errors"`
    Warnings int `json:"warnings"`
}

// EntityType identifies entity types
type EntityType string

const (
    EntityTypeEpic       EntityType = "epic"
    EntityTypeFeature    EntityType = "feature"
    EntityTypeTask       EntityType = "task"
    EntityTypeRelatedDoc EntityType = "related_doc"
)

// ErrorType categorizes errors
type ErrorType string

const (
    ErrorTypePatternMismatch     ErrorType = "pattern_mismatch"
    ErrorTypeValidationFailure   ErrorType = "validation_failure"
    ErrorTypeParseError          ErrorType = "parse_error"
    ErrorTypeFrontmatterError    ErrorType = "frontmatter_error"
    ErrorTypeFileAccessError     ErrorType = "file_access_error"
    ErrorTypeMissingMetadata     ErrorType = "missing_metadata"
    ErrorTypeInvalidKey          ErrorType = "invalid_key"
)

// Severity categorizes error severity
type Severity string

const (
    SeverityError   Severity = "ERROR"   // Blocks import
    SeverityWarning Severity = "WARNING" // Imports with issues
    SeverityInfo    Severity = "INFO"    // Informational
)

// SkipReason explains why a file was skipped
type SkipReason string

const (
    SkipReasonPatternMismatch SkipReason = "pattern_mismatch"
    SkipReasonParseError      SkipReason = "parse_error"
    SkipReasonValidationError SkipReason = "validation_error"
    SkipReasonFileTooLarge    SkipReason = "file_too_large"
)

func (s SkipReason) Message() string {
    switch s {
    case SkipReasonPatternMismatch:
        return "File does not match any recognized pattern"
    case SkipReasonParseError:
        return "Failed to parse file content"
    case SkipReasonValidationError:
        return "File content failed validation"
    case SkipReasonFileTooLarge:
        return "File exceeds maximum size limit"
    default:
        return "Unknown reason"
    }
}
```

### ValidationReport Structure

```go
// ValidationReport contains validation results
type ValidationReport struct {
    SchemaVersion    string             `json:"schema_version"`
    Status           string             `json:"status"` // "success", "failure"
    Metadata         ValidationMetadata `json:"metadata"`
    ValidationChecks ValidationChecks   `json:"validation_checks"`
    Failures         []ValidationFailure `json:"failures"`
    Summary          ValidationSummary  `json:"summary"`
}

// ValidationMetadata contains validation timing
type ValidationMetadata struct {
    Timestamp    time.Time `json:"timestamp"`
    DatabasePath string    `json:"database_path"`
}

// ValidationChecks summarizes check results
type ValidationChecks struct {
    FilePaths     FilePathCheck     `json:"file_paths"`
    Relationships RelationshipCheck `json:"relationships"`
}

// FilePathCheck summarizes file path validation
type FilePathCheck struct {
    TotalChecked int `json:"total_checked"`
    BrokenPaths  int `json:"broken_paths"`
}

// RelationshipCheck summarizes relationship validation
type RelationshipCheck struct {
    TotalChecked     int `json:"total_checked"`
    OrphanedFeatures int `json:"orphaned_features"`
    OrphanedTasks    int `json:"orphaned_tasks"`
}

// ValidationFailure describes a single validation issue
type ValidationFailure struct {
    CheckType        string `json:"check_type"`         // "file_path_existence", "relationship_integrity"
    EntityType       string `json:"entity_type"`        // "epic", "feature", "task"
    EntityKey        string `json:"entity_key"`         // Entity key
    FilePath         string `json:"file_path,omitempty"` // For file path checks
    MissingParentType string `json:"missing_parent_type,omitempty"` // For orphan checks
    MissingParentKey  string `json:"missing_parent_key,omitempty"`  // For orphan checks
    Issue            string `json:"issue"`              // Human-readable issue
    SuggestedFix     string `json:"suggested_fix"`      // Actionable suggestion
}

// ValidationSummary provides final totals
type ValidationSummary struct {
    TotalValidated  int `json:"total_validated"`
    TotalIssues     int `json:"total_issues"`
    BrokenFilePaths int `json:"broken_file_paths"`
    OrphanedRecords int `json:"orphaned_records"`
}
```

---

## Algorithms

### Report Generation Algorithm

**Time Complexity**: O(n) where n = total events collected
**Space Complexity**: O(n) for storing events

```
Algorithm: GenerateScanReport
Input: Reporter with collected events
Output: ScanReport

1. Calculate duration = current_time - start_time

2. Sum matched entities:
   matched_count = 0
   for each entity_type in filesMatched:
       matched_count += filesMatched[entity_type]

3. Sum skipped entities:
   skipped_count = 0
   for each entity_type in filesSkipped:
       skipped_count += filesSkipped[entity_type]

4. Determine status:
   if len(errors) > 0:
       status = "failure"
   else if len(warnings) > 0:
       status = "success_with_warnings"
   else:
       status = "success"

5. Build report structure:
   report = {
       schema_version: "1.0",
       status: status,
       dry_run: isDryRun,
       metadata: { timestamp, duration, validation_level, doc_root, patterns },
       counts: { scanned: filesScanned, matched: matched_count, skipped: skipped_count },
       entities: { breakdown by type },
       errors: errors,
       warnings: warnings,
       summary: { imported, error_count, warning_count }
   }

6. Return report
```

### Validation Algorithm

**Time Complexity**: O(e + f + t) where e=epics, f=features, t=tasks
**Space Complexity**: O(e + f) for key maps

```
Algorithm: ValidateDatabaseIntegrity
Input: Epic, Feature, Task repositories
Output: ValidationReport

1. File Path Validation:
   failures = []

   for each epic in epics:
       if epic.file_path is not null:
           if file does not exist:
               add failure: "epic file not found"

   for each feature in features:
       if feature.file_path is not null:
           if file does not exist:
               add failure: "feature file not found"

   for each task in tasks:
       if task.file_path is not null:
           if file does not exist:
               add failure: "task file not found"

2. Relationship Validation:
   epic_keys = build_set(epics.key)  // O(e)

   feature_keys = {}
   for each feature in features:
       add feature.key to feature_keys
       if feature.epic_key not in epic_keys:
           add failure: "orphaned feature"

   for each task in tasks:
       if task.feature_key not in feature_keys:
           add failure: "orphaned task"

3. Count failures by type:
   broken_paths = count(failures where check_type = "file_path_existence")
   orphaned_features = count(failures where entity_type = "feature" and check_type = "relationship_integrity")
   orphaned_tasks = count(failures where entity_type = "task" and check_type = "relationship_integrity")

4. Build report:
   report = {
       schema_version: "1.0",
       status: "failure" if len(failures) > 0 else "success",
       metadata: { timestamp, database_path },
       validation_checks: { file_paths, relationships },
       failures: failures,
       summary: { total_validated, total_issues, broken_paths, orphaned_records }
   }

5. Return report
```

---

## Implementation Details

### Sync Engine Integration

**Modification**: Update `internal/sync/engine.go` to use Reporter.

```go
package sync

import (
    "context"
    "database/sql"
)

type SyncEngine struct {
    scanner  *Scanner
    parser   *Parser
    repos    *Repositories
    reporter *Reporter
    options  SyncOptions
}

func NewEngine(repos *Repositories, options SyncOptions) *SyncEngine {
    return &SyncEngine{
        scanner:  NewScanner(),
        parser:   NewParser(),
        repos:    repos,
        reporter: nil, // Created in Sync()
        options:  options,
    }
}

func (e *SyncEngine) Sync(ctx context.Context) (*ScanReport, error) {
    // Initialize reporter
    e.reporter = NewReporter(e.options)
    e.reporter.ScanStarted(time.Now(), e.options)

    // Discovery phase
    files, err := e.scanner.Discover(e.options.FolderPath)
    if err != nil {
        return nil, fmt.Errorf("discovery failed: %w", err)
    }

    for _, file := range files {
        e.reporter.FileDiscovered(file.Path)
    }

    // Parsing phase
    metadataList := []TaskMetadata{}
    for _, file := range files {
        metadata, err := e.parser.Parse(file)
        if err != nil {
            parseErr, ok := err.(*ParseError)
            if ok {
                e.reporter.ParseError(file.Path, parseErr.Line, parseErr.ErrorType, parseErr.Message, parseErr.SuggestedFix)
            } else {
                e.reporter.FileSkipped(file.Path, file.EntityType, SkipReasonParseError, "Fix parse error and re-scan")
            }
            continue
        }

        e.reporter.FileMatched(file.Path, file.EntityType, metadata.Key)
        metadataList = append(metadataList, metadata)
    }

    // Import phase (with transaction for dry-run support)
    tx, err := e.repos.DB.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Safe to call even after commit

    for _, metadata := range metadataList {
        if err := e.importEntity(ctx, tx, metadata); err != nil {
            e.reporter.ImportError(metadata.Key, err)
            continue
        }
        e.reporter.EntityImported(metadata.Type, metadata.Key)
    }

    // Commit or rollback based on dry-run mode
    if e.options.DryRun {
        tx.Rollback()
    } else {
        if err := tx.Commit(); err != nil {
            return nil, fmt.Errorf("failed to commit transaction: %w", err)
        }
    }

    // Generate and return report
    report := e.reporter.GenerateReport()
    return report, nil
}

func (e *SyncEngine) importEntity(ctx context.Context, tx *sql.Tx, metadata TaskMetadata) error {
    // Import logic (using transaction)
    // ... implementation
    return nil
}
```

### CLI Command Updates

**File**: `internal/cli/commands/sync.go`

```go
package commands

import (
    "context"
    "fmt"
    "os"

    "github.com/jwwelbor/shark-task-manager/internal/sync"
    "github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
    Use:   "sync [flags]",
    Short: "Synchronize task files to database",
    Long: `Scan documentation folder and sync to database.

Supports dry-run mode for previewing changes and JSON output for programmatic consumption.

Examples:
  shark sync                    # Sync with default settings
  shark sync --dry-run          # Preview changes without committing
  shark sync --output=json      # JSON output for scripts
  shark sync --dry-run --output=json  # Combine flags`,
    RunE: runSync,
}

var (
    syncDryRun bool
    syncOutput string
)

func init() {
    syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview changes without committing to database")
    syncCmd.Flags().StringVar(&syncOutput, "output", "cli", "Output format: cli or json")
}

func runSync(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Initialize repositories
    repos, err := initializeRepositories()
    if err != nil {
        return err
    }
    defer repos.Close()

    // Configure sync options
    options := sync.SyncOptions{
        DBPath:        getDBPath(),
        FolderPath:    getDocsPath(),
        DryRun:        syncDryRun,
        ValidationLevel: "balanced",
        // ... other options
    }

    // Execute sync
    engine := sync.NewEngine(repos, options)
    report, err := engine.Sync(ctx)
    if err != nil {
        return fmt.Errorf("sync failed: %w", err)
    }

    // Format and output report
    useColor := isTerminal(os.Stdout)
    formatter := sync.NewFormatter(useColor)

    var output string
    if syncOutput == "json" {
        output, err = formatter.FormatScanJSON(report)
        if err != nil {
            return err
        }
    } else {
        output = formatter.FormatScanCLI(report)
    }

    fmt.Println(output)

    // Exit with error if scan had errors
    if report.Status == "failure" {
        return fmt.Errorf("sync completed with %d error(s)", report.Summary.Errors)
    }

    return nil
}
```

**File**: `internal/cli/commands/validate.go`

```go
package commands

import (
    "context"
    "fmt"
    "os"

    "github.com/jwwelbor/shark-task-manager/internal/sync"
    "github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
    Use:   "validate [flags]",
    Short: "Validate database integrity",
    Long: `Verify database integrity by checking:
  - File path existence (all entities)
  - Relationship integrity (features → epics, tasks → features)
  - Broken references

Examples:
  shark validate              # Human-readable output
  shark validate --output=json  # JSON output for scripts`,
    RunE: runValidate,
}

var validateOutput string

func init() {
    validateCmd.Flags().StringVar(&validateOutput, "output", "cli", "Output format: cli or json")
}

func runValidate(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Initialize repositories
    repos, err := initializeRepositories()
    if err != nil {
        return err
    }
    defer repos.Close()

    // Create validator
    validator := sync.NewValidator(repos.Epic, repos.Feature, repos.Task)

    // Run validation
    report, err := validator.Validate(ctx)
    if err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Format and output report
    useColor := isTerminal(os.Stdout)
    formatter := sync.NewFormatter(useColor)

    var output string
    if validateOutput == "json" {
        output, err = formatter.FormatValidationJSON(report)
        if err != nil {
            return err
        }
    } else {
        output = formatter.FormatValidationCLI(report)
    }

    fmt.Println(output)

    // Exit with error if validation failed
    if report.Status == "failure" {
        return fmt.Errorf("validation found %d issue(s)", report.Summary.TotalIssues)
    }

    return nil
}
```

---

## Error Handling

### Error Classification

| Error Type | Severity | Behavior | Recovery |
|-----------|----------|----------|----------|
| Parse error | ERROR | Skip file, continue scan | Fix file, re-scan |
| Pattern mismatch | WARNING | Skip file, continue scan | Rename file or ignore |
| Validation error | ERROR | Skip file, continue scan | Fix validation issue |
| File access error | ERROR | Skip file, continue scan | Check permissions |
| Database error | FATAL | Abort scan | Fix database, retry |

### Graceful Degradation

**Principle**: Individual file failures don't halt entire scan.

```go
func (e *SyncEngine) processFiles(files []FileInfo) {
    for _, file := range files {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    e.reporter.FileError(file.Path, fmt.Errorf("panic: %v", r))
                }
            }()

            if err := e.processFile(file); err != nil {
                e.reporter.FileSkipped(file.Path, file.EntityType, SkipReasonParseError, "Fix error and re-scan")
            }
        }()
    }
}
```

### Transaction Safety

**Dry-run implementation**:

```go
func (e *SyncEngine) executeWithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
    tx, err := e.repos.DB.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Always safe to call

    if err := fn(tx); err != nil {
        return err
    }

    if e.options.DryRun {
        // Rollback for dry-run (report already generated)
        return nil
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

---

## Testing Approach

### Unit Tests

**File**: `internal/sync/reporter_test.go`

```go
package sync

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestReporter_FileSkipped(t *testing.T) {
    options := SyncOptions{
        FolderPath:      "/docs/plan",
        ValidationLevel: "balanced",
        DryRun:          false,
    }
    reporter := NewReporter(options)

    reporter.FileSkipped("/docs/plan/notes.md", EntityTypeTask, SkipReasonPatternMismatch, "Rename to match pattern")

    report := reporter.GenerateReport()

    assert.Equal(t, 1, report.Counts.Scanned)
    assert.Equal(t, 0, report.Counts.Matched)
    assert.Equal(t, 1, report.Counts.Skipped)
    assert.Len(t, report.Errors, 1)
    assert.Equal(t, ErrorTypePatternMismatch, report.Errors[0].ErrorType)
    assert.Equal(t, "Rename to match pattern", report.Errors[0].SuggestedFix)
}

func TestReporter_ParseError(t *testing.T) {
    reporter := NewReporter(SyncOptions{})

    reporter.ParseError("/docs/plan/task.md", 5, ErrorTypeFrontmatterError, "Missing closing '---'", "Add '---' on line 8")

    report := reporter.GenerateReport()

    assert.Len(t, report.Errors, 1)
    err := report.Errors[0]
    assert.Equal(t, ErrorTypeFrontmatterError, err.ErrorType)
    assert.Equal(t, "/docs/plan/task.md", err.FilePath)
    assert.NotNil(t, err.LineNumber)
    assert.Equal(t, 5, *err.LineNumber)
    assert.Contains(t, err.Message, "Missing closing '---'")
}
```

**File**: `internal/sync/validator_test.go`

```go
package sync

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestValidator_ValidateFilePaths(t *testing.T) {
    // Setup: create test database with stale file paths
    repos := setupTestRepositories(t)
    defer cleanupTestRepositories(t, repos)

    // Create task with non-existent file path
    task := models.Task{
        Key:       "E01-F01-T001",
        FeatureKey: "E01-F01",
        Title:     "Test task",
        FilePath:  stringPtr("/nonexistent/path.md"),
    }
    repos.Task.Create(context.Background(), &task)

    // Execute validation
    validator := NewValidator(repos.Epic, repos.Feature, repos.Task)
    failures, err := validator.ValidateFilePaths(context.Background())

    // Verify
    assert.NoError(t, err)
    assert.Len(t, failures, 1)
    assert.Equal(t, "file_path_existence", failures[0].CheckType)
    assert.Equal(t, "E01-F01-T001", failures[0].EntityKey)
}

func TestValidator_ValidateRelationships(t *testing.T) {
    // Setup: create orphaned task
    repos := setupTestRepositories(t)
    defer cleanupTestRepositories(t, repos)

    // Create task without parent feature
    task := models.Task{
        Key:       "E01-F99-T001",
        FeatureKey: "E01-F99", // Does not exist
        Title:     "Orphaned task",
    }
    repos.Task.Create(context.Background(), &task)

    // Execute validation
    validator := NewValidator(repos.Epic, repos.Feature, repos.Task)
    failures, err := validator.ValidateRelationships(context.Background())

    // Verify
    assert.NoError(t, err)
    assert.Len(t, failures, 1)
    assert.Equal(t, "relationship_integrity", failures[0].CheckType)
    assert.Equal(t, "task", failures[0].EntityType)
    assert.Equal(t, "E01-F99", failures[0].MissingParentKey)
}
```

### Integration Tests

**File**: `internal/sync/sync_integration_test.go`

```go
func TestSyncEngine_DryRun(t *testing.T) {
    // Setup: create test files
    testDir := setupTestFiles(t)
    defer os.RemoveAll(testDir)

    repos := setupTestRepositories(t)
    defer cleanupTestRepositories(t, repos)

    // Execute sync in dry-run mode
    options := SyncOptions{
        FolderPath: testDir,
        DryRun:     true,
    }
    engine := NewEngine(repos, options)
    report, err := engine.Sync(context.Background())

    // Verify report generated
    assert.NoError(t, err)
    assert.True(t, report.DryRun)
    assert.Equal(t, 3, report.Counts.Matched)

    // Verify database unchanged
    tasks, _ := repos.Task.List(context.Background())
    assert.Empty(t, tasks, "Dry-run should not modify database")
}

func TestSyncEngine_JSONOutput(t *testing.T) {
    testDir := setupTestFiles(t)
    defer os.RemoveAll(testDir)

    repos := setupTestRepositories(t)
    defer cleanupTestRepositories(t, repos)

    options := SyncOptions{FolderPath: testDir}
    engine := NewEngine(repos, options)
    report, _ := engine.Sync(context.Background())

    // Format as JSON
    formatter := NewFormatter(false)
    jsonOutput, err := formatter.FormatScanJSON(report)

    // Verify valid JSON
    assert.NoError(t, err)

    var parsed map[string]interface{}
    err = json.Unmarshal([]byte(jsonOutput), &parsed)
    assert.NoError(t, err)

    // Verify schema fields
    assert.Equal(t, "1.0", parsed["schema_version"])
    assert.Contains(t, []string{"success", "failure", "success_with_warnings"}, parsed["status"])
}
```

---

*This backend design provides POC-level implementation specifications for developers building the import reporting and validation feature.*
