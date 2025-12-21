package reporting

import (
	"testing"
	"time"
)

func TestNewScanReport(t *testing.T) {
	// Arrange & Act
	report := NewScanReport()

	// Assert
	if report == nil {
		t.Fatal("NewScanReport() returned nil")
	}
	if report.Metadata.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	if report.Counts.Scanned != 0 {
		t.Errorf("Expected scanned count to be 0, got %d", report.Counts.Scanned)
	}
	if report.Counts.Matched != 0 {
		t.Errorf("Expected matched count to be 0, got %d", report.Counts.Matched)
	}
	if report.Counts.Skipped != 0 {
		t.Errorf("Expected skipped count to be 0, got %d", report.Counts.Skipped)
	}
}

func TestScanReport_AddScanned(t *testing.T) {
	// Arrange
	report := NewScanReport()

	// Act
	report.AddScanned("epic")
	report.AddScanned("feature")
	report.AddScanned("task")

	// Assert
	if report.Counts.Scanned != 3 {
		t.Errorf("Expected scanned count to be 3, got %d", report.Counts.Scanned)
	}
}

func TestScanReport_AddMatched(t *testing.T) {
	// Arrange
	report := NewScanReport()

	// Act
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddMatched("feature", "/path/to/feature.md")
	report.AddMatched("task", "/path/to/task.md")

	// Assert
	if report.Counts.Matched != 3 {
		t.Errorf("Expected matched count to be 3, got %d", report.Counts.Matched)
	}
	if report.Entities.Epics.Matched != 1 {
		t.Errorf("Expected epics matched to be 1, got %d", report.Entities.Epics.Matched)
	}
	if report.Entities.Features.Matched != 1 {
		t.Errorf("Expected features matched to be 1, got %d", report.Entities.Features.Matched)
	}
	if report.Entities.Tasks.Matched != 1 {
		t.Errorf("Expected tasks matched to be 1, got %d", report.Entities.Tasks.Matched)
	}
}

func TestScanReport_AddSkipped(t *testing.T) {
	// Arrange
	report := NewScanReport()
	entry := SkippedFileEntry{
		FilePath:      "/path/to/file.md",
		Reason:        "Pattern mismatch",
		SuggestedFix:  "Rename to match pattern",
		ErrorType:     "pattern_mismatch",
		LineNumber:    nil,
	}

	// Act
	report.AddSkipped("task", entry)

	// Assert
	if report.Counts.Skipped != 1 {
		t.Errorf("Expected skipped count to be 1, got %d", report.Counts.Skipped)
	}
	if report.Entities.Tasks.Skipped != 1 {
		t.Errorf("Expected tasks skipped to be 1, got %d", report.Entities.Tasks.Skipped)
	}
	if len(report.SkippedFiles) != 1 {
		t.Errorf("Expected 1 skipped file, got %d", len(report.SkippedFiles))
	}
	if report.SkippedFiles[0].FilePath != "/path/to/file.md" {
		t.Errorf("Expected file path '/path/to/file.md', got '%s'", report.SkippedFiles[0].FilePath)
	}
}

func TestScanReport_AddError(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum := 5
	entry := SkippedFileEntry{
		FilePath:      "/path/to/error.md",
		Reason:        "Parse error",
		SuggestedFix:  "Fix YAML syntax",
		ErrorType:     "parse_error",
		LineNumber:    &lineNum,
	}

	// Act
	report.AddError(entry)

	// Assert
	if len(report.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(report.Errors))
	}
	if report.Errors[0].FilePath != "/path/to/error.md" {
		t.Errorf("Expected file path '/path/to/error.md', got '%s'", report.Errors[0].FilePath)
	}
	if report.Errors[0].LineNumber == nil {
		t.Error("Expected line number to be set")
	} else if *report.Errors[0].LineNumber != 5 {
		t.Errorf("Expected line number 5, got %d", *report.Errors[0].LineNumber)
	}
}

func TestScanReport_AddWarning(t *testing.T) {
	// Arrange
	report := NewScanReport()
	entry := SkippedFileEntry{
		FilePath:      "/path/to/warning.md",
		Reason:        "Missing optional metadata",
		SuggestedFix:  "Add description field",
		ErrorType:     "validation_warning",
		LineNumber:    nil,
	}

	// Act
	report.AddWarning(entry)

	// Assert
	if len(report.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(report.Warnings))
	}
	if report.Warnings[0].FilePath != "/path/to/warning.md" {
		t.Errorf("Expected file path '/path/to/warning.md', got '%s'", report.Warnings[0].FilePath)
	}
}

func TestScanReport_SetMetadata(t *testing.T) {
	// Arrange
	report := NewScanReport()
	metadata := ScanMetadata{
		Timestamp:         time.Now(),
		DurationSeconds:   2.5,
		ValidationLevel:   "balanced",
		DocumentationRoot: "/home/user/docs",
		Patterns: map[string]string{
			"epic":    "E\\d{2}-.*",
			"feature": "E\\d{2}-F\\d{2}-.*",
			"task":    "T-E\\d{2}-F\\d{2}-\\d{3}\\.md",
		},
	}

	// Act
	report.SetMetadata(metadata)

	// Assert
	if report.Metadata.ValidationLevel != "balanced" {
		t.Errorf("Expected validation level 'balanced', got '%s'", report.Metadata.ValidationLevel)
	}
	if report.Metadata.DocumentationRoot != "/home/user/docs" {
		t.Errorf("Expected documentation root '/home/user/docs', got '%s'", report.Metadata.DocumentationRoot)
	}
	if report.Metadata.DurationSeconds != 2.5 {
		t.Errorf("Expected duration 2.5, got %f", report.Metadata.DurationSeconds)
	}
	if len(report.Metadata.Patterns) != 3 {
		t.Errorf("Expected 3 patterns, got %d", len(report.Metadata.Patterns))
	}
}

func TestScanReport_ErrorGrouping(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum1 := 5
	lineNum2 := 12

	// Act - Add multiple errors of different types
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/parse1.md",
		Reason:       "Missing closing frontmatter",
		SuggestedFix: "Add '---' on line 8",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum1,
	})
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/validation1.md",
		Reason:       "Invalid epic key",
		SuggestedFix: "Use format E##-slug",
		ErrorType:    "validation_failure",
		LineNumber:   nil,
	})
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/parse2.md",
		Reason:       "Invalid YAML syntax",
		SuggestedFix: "Check YAML formatting",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum2,
	})

	// Assert
	errorsByType := make(map[string][]SkippedFileEntry)
	for _, err := range report.Errors {
		errorsByType[err.ErrorType] = append(errorsByType[err.ErrorType], err)
	}

	if len(errorsByType) != 2 {
		t.Errorf("Expected 2 error types, got %d", len(errorsByType))
	}
	if len(errorsByType["parse_error"]) != 2 {
		t.Errorf("Expected 2 parse errors, got %d", len(errorsByType["parse_error"]))
	}
	if len(errorsByType["validation_failure"]) != 1 {
		t.Errorf("Expected 1 validation failure, got %d", len(errorsByType["validation_failure"]))
	}
}

func TestScanReport_EntityBreakdown(t *testing.T) {
	// Arrange
	report := NewScanReport()

	// Act - Simulate a realistic scan
	// Epics: 10 matched, 1 skipped
	for i := 0; i < 10; i++ {
		report.AddMatched("epic", "/path/to/epic.md")
	}
	report.AddSkipped("epic", SkippedFileEntry{
		FilePath:     "/path/to/skipped-epic.md",
		Reason:       "Pattern mismatch",
		SuggestedFix: "Rename to match pattern",
		ErrorType:    "pattern_mismatch",
	})

	// Features: 35 matched, 5 skipped
	for i := 0; i < 35; i++ {
		report.AddMatched("feature", "/path/to/feature.md")
	}
	for i := 0; i < 5; i++ {
		report.AddSkipped("feature", SkippedFileEntry{
			FilePath:     "/path/to/skipped-feature.md",
			Reason:       "Validation failure",
			SuggestedFix: "Fix metadata",
			ErrorType:    "validation_failure",
		})
	}

	// Tasks: 35 matched, 14 skipped
	for i := 0; i < 35; i++ {
		report.AddMatched("task", "/path/to/task.md")
	}
	for i := 0; i < 14; i++ {
		report.AddSkipped("task", SkippedFileEntry{
			FilePath:     "/path/to/skipped-task.md",
			Reason:       "Parse error",
			SuggestedFix: "Fix frontmatter",
			ErrorType:    "parse_error",
		})
	}

	// Assert
	if report.Entities.Epics.Matched != 10 {
		t.Errorf("Expected 10 epics matched, got %d", report.Entities.Epics.Matched)
	}
	if report.Entities.Epics.Skipped != 1 {
		t.Errorf("Expected 1 epic skipped, got %d", report.Entities.Epics.Skipped)
	}
	if report.Entities.Features.Matched != 35 {
		t.Errorf("Expected 35 features matched, got %d", report.Entities.Features.Matched)
	}
	if report.Entities.Features.Skipped != 5 {
		t.Errorf("Expected 5 features skipped, got %d", report.Entities.Features.Skipped)
	}
	if report.Entities.Tasks.Matched != 35 {
		t.Errorf("Expected 35 tasks matched, got %d", report.Entities.Tasks.Matched)
	}
	if report.Entities.Tasks.Skipped != 14 {
		t.Errorf("Expected 14 tasks skipped, got %d", report.Entities.Tasks.Skipped)
	}

	// Total counts
	if report.Counts.Matched != 80 {
		t.Errorf("Expected 80 total matched, got %d", report.Counts.Matched)
	}
	if report.Counts.Skipped != 20 {
		t.Errorf("Expected 20 total skipped, got %d", report.Counts.Skipped)
	}
}

func TestScanReport_SkippedFileDetails(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum := 42

	// Act
	report.AddSkipped("task", SkippedFileEntry{
		FilePath:     "/absolute/path/to/file.md",
		Reason:       "Cannot parse frontmatter: Missing closing '---' for frontmatter block",
		SuggestedFix: "Add '---' on line 45 to close frontmatter",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum,
	})

	// Assert
	if len(report.SkippedFiles) != 1 {
		t.Fatalf("Expected 1 skipped file, got %d", len(report.SkippedFiles))
	}

	skipped := report.SkippedFiles[0]
	if skipped.FilePath != "/absolute/path/to/file.md" {
		t.Errorf("Expected absolute path, got '%s'", skipped.FilePath)
	}
	if skipped.Reason == "" {
		t.Error("Expected reason to be set")
	}
	if skipped.SuggestedFix == "" {
		t.Error("Expected suggested fix to be set")
	}
	if skipped.ErrorType != "parse_error" {
		t.Errorf("Expected error type 'parse_error', got '%s'", skipped.ErrorType)
	}
	if skipped.LineNumber == nil {
		t.Error("Expected line number to be set")
	} else if *skipped.LineNumber != 42 {
		t.Errorf("Expected line number 42, got %d", *skipped.LineNumber)
	}
}

func TestScanReport_RelatedDocsSupport(t *testing.T) {
	// Arrange
	report := NewScanReport()

	// Act
	report.AddMatched("related", "/path/to/architecture.md")
	report.AddMatched("related", "/path/to/api-spec.md")
	report.AddSkipped("related", SkippedFileEntry{
		FilePath:     "/path/to/invalid.md",
		Reason:       "Invalid format",
		SuggestedFix: "Fix format",
		ErrorType:    "validation_failure",
	})

	// Assert
	if report.Entities.RelatedDocs.Matched != 2 {
		t.Errorf("Expected 2 related docs matched, got %d", report.Entities.RelatedDocs.Matched)
	}
	if report.Entities.RelatedDocs.Skipped != 1 {
		t.Errorf("Expected 1 related doc skipped, got %d", report.Entities.RelatedDocs.Skipped)
	}
}

func TestScanReport_DryRunMode(t *testing.T) {
	// Arrange
	report := NewScanReport()

	// Act
	report.SetDryRun(true)

	// Assert
	if !report.DryRun {
		t.Error("Expected dry run to be true")
	}
}

func TestScanReport_StatusTracking(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*ScanReport)
		expectedStatus string
	}{
		{
			name: "success with no errors",
			setup: func(r *ScanReport) {
				r.AddMatched("epic", "/path/to/epic.md")
				r.AddMatched("feature", "/path/to/feature.md")
			},
			expectedStatus: "success",
		},
		{
			name: "success with warnings only",
			setup: func(r *ScanReport) {
				r.AddMatched("epic", "/path/to/epic.md")
				r.AddWarning(SkippedFileEntry{
					FilePath:     "/path/to/warn.md",
					Reason:       "Missing optional metadata",
					SuggestedFix: "Add description",
					ErrorType:    "validation_warning",
				})
			},
			expectedStatus: "success",
		},
		{
			name: "failure with errors",
			setup: func(r *ScanReport) {
				r.AddMatched("epic", "/path/to/epic.md")
				r.AddError(SkippedFileEntry{
					FilePath:     "/path/to/error.md",
					Reason:       "Parse error",
					SuggestedFix: "Fix YAML",
					ErrorType:    "parse_error",
				})
			},
			expectedStatus: "failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			report := NewScanReport()

			// Act
			tt.setup(report)
			status := report.GetStatus()

			// Assert
			if status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, status)
			}
		})
	}
}
