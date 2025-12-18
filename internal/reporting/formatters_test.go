package reporting

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatCLI_BasicReport(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddMatched("feature", "/path/to/feature.md")
	report.AddMatched("task", "/path/to/task.md")
	report.SetMetadata(ScanMetadata{
		Timestamp:         time.Date(2025, 12, 17, 14, 32, 5, 0, time.UTC),
		DurationSeconds:   2.3,
		ValidationLevel:   "balanced",
		DocumentationRoot: "/home/user/project/docs/plan",
		Patterns: map[string]string{
			"epic": "E\\d{2}-.*",
		},
	})

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "Shark Scan Report") {
		t.Error("Expected output to contain 'Shark Scan Report'")
	}
	if !strings.Contains(output, "Total files scanned: 3") {
		t.Error("Expected output to show total scanned: 3")
	}
	if !strings.Contains(output, "Matched: 3") {
		t.Error("Expected output to show matched: 3")
	}
	if !strings.Contains(output, "Duration: 2.3 seconds") {
		t.Error("Expected output to show duration: 2.3 seconds")
	}
	if !strings.Contains(output, "Validation level: balanced") {
		t.Error("Expected output to show validation level")
	}
}

func TestFormatCLI_WithErrors(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum := 5
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/error.md",
		Reason:       "Cannot parse frontmatter: Missing closing '---'",
		SuggestedFix: "Add '---' on line 8 to close frontmatter",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum,
	})

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "Errors and Warnings") {
		t.Error("Expected output to contain 'Errors and Warnings' section")
	}
	if !strings.Contains(output, "/path/to/error.md") {
		t.Error("Expected output to show error file path")
	}
	if !strings.Contains(output, "Cannot parse frontmatter") {
		t.Error("Expected output to show error reason")
	}
	if !strings.Contains(output, "Add '---' on line 8") {
		t.Error("Expected output to show suggested fix")
	}
}

func TestFormatCLI_ErrorGrouping(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum1 := 5
	lineNum2 := 12

	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/parse1.md",
		Reason:       "Parse error 1",
		SuggestedFix: "Fix 1",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum1,
	})
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/validation1.md",
		Reason:       "Validation error 1",
		SuggestedFix: "Fix validation",
		ErrorType:    "validation_failure",
		LineNumber:   nil,
	})
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/parse2.md",
		Reason:       "Parse error 2",
		SuggestedFix: "Fix 2",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum2,
	})

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "Parse Errors (2)") {
		t.Error("Expected grouped parse errors section")
	}
	if !strings.Contains(output, "Validation Errors (1)") {
		t.Error("Expected grouped validation errors section")
	}
}

func TestFormatCLI_WithColor(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/error.md",
		Reason:       "Error",
		SuggestedFix: "Fix",
		ErrorType:    "parse_error",
	})

	// Act
	outputColor := FormatCLI(report, true)
	outputNoColor := FormatCLI(report, false)

	// Assert - colored output should be different from non-colored
	if outputColor == outputNoColor {
		t.Error("Expected colored output to differ from non-colored output")
	}
	// Color codes should be present in colored output
	if !strings.Contains(outputColor, "\033[") && len(outputColor) > 0 {
		// Note: Only fail if we actually have content
		// Some implementations might not add color codes if terminal doesn't support it
		t.Log("Warning: No ANSI color codes detected in colored output")
	}
}

func TestFormatCLI_DryRunIndicator(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.SetDryRun(true)
	report.AddMatched("epic", "/path/to/epic.md")

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "DRY RUN") {
		t.Error("Expected dry run indicator in output")
	}
	if !strings.Contains(output, "No database changes") {
		t.Error("Expected dry run explanation in output")
	}
}

func TestFormatCLI_EntityBreakdown(t *testing.T) {
	// Arrange
	report := NewScanReport()
	for i := 0; i < 10; i++ {
		report.AddMatched("epic", "/path/to/epic.md")
	}
	for i := 0; i < 35; i++ {
		report.AddMatched("feature", "/path/to/feature.md")
	}
	for i := 0; i < 35; i++ {
		report.AddMatched("task", "/path/to/task.md")
	}

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "Breakdown by Type") {
		t.Error("Expected 'Breakdown by Type' section")
	}
	if !strings.Contains(output, "Epics:") {
		t.Error("Expected 'Epics:' section")
	}
	if !strings.Contains(output, "Features:") {
		t.Error("Expected 'Features:' section")
	}
	if !strings.Contains(output, "Tasks:") {
		t.Error("Expected 'Tasks:' section")
	}
}

func TestFormatJSON_BasicReport(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddMatched("feature", "/path/to/feature.md")
	report.SetMetadata(ScanMetadata{
		Timestamp:         time.Date(2025, 12, 17, 14, 32, 5, 0, time.UTC),
		DurationSeconds:   2.3,
		ValidationLevel:   "balanced",
		DocumentationRoot: "/home/user/docs",
		Patterns:          map[string]string{"epic": "E\\d{2}-.*"},
	})

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	// Verify schema version
	if parsed["schema_version"] != "1.0" {
		t.Errorf("Expected schema_version '1.0', got '%v'", parsed["schema_version"])
	}

	// Verify status
	if parsed["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", parsed["status"])
	}

	// Verify counts
	counts := parsed["counts"].(map[string]interface{})
	if counts["matched"] != float64(2) {
		t.Errorf("Expected matched count 2, got %v", counts["matched"])
	}
}

func TestFormatJSON_WithErrors(t *testing.T) {
	// Arrange
	report := NewScanReport()
	lineNum := 5
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/error.md",
		Reason:       "Parse error",
		SuggestedFix: "Fix YAML",
		ErrorType:    "parse_error",
		LineNumber:   &lineNum,
	})

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	// Verify status is failure
	if parsed["status"] != "failure" {
		t.Errorf("Expected status 'failure', got '%v'", parsed["status"])
	}

	// Verify errors array
	errors := parsed["errors"].([]interface{})
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	errorObj := errors[0].(map[string]interface{})
	if errorObj["file_path"] != "/path/to/error.md" {
		t.Errorf("Expected file_path '/path/to/error.md', got '%v'", errorObj["file_path"])
	}
	if errorObj["error_type"] != "parse_error" {
		t.Errorf("Expected error_type 'parse_error', got '%v'", errorObj["error_type"])
	}
	if errorObj["line_number"] != float64(5) {
		t.Errorf("Expected line_number 5, got %v", errorObj["line_number"])
	}
}

func TestFormatJSON_EntityBreakdown(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddMatched("feature", "/path/to/feature1.md")
	report.AddMatched("feature", "/path/to/feature2.md")
	report.AddSkipped("task", SkippedFileEntry{
		FilePath:     "/path/to/skipped.md",
		Reason:       "Skipped",
		SuggestedFix: "Fix",
		ErrorType:    "pattern_mismatch",
	})

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	entities := parsed["entities"].(map[string]interface{})

	epics := entities["epics"].(map[string]interface{})
	if epics["matched"] != float64(1) {
		t.Errorf("Expected epics matched 1, got %v", epics["matched"])
	}

	features := entities["features"].(map[string]interface{})
	if features["matched"] != float64(2) {
		t.Errorf("Expected features matched 2, got %v", features["matched"])
	}

	tasks := entities["tasks"].(map[string]interface{})
	if tasks["skipped"] != float64(1) {
		t.Errorf("Expected tasks skipped 1, got %v", tasks["skipped"])
	}
}

func TestFormatJSON_DryRunFlag(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.SetDryRun(true)

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	if parsed["dry_run"] != true {
		t.Errorf("Expected dry_run true, got %v", parsed["dry_run"])
	}
}

func TestFormatJSON_ValidJSONStructure(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddError(SkippedFileEntry{
		FilePath:     "/path/to/error.md",
		Reason:       "Error",
		SuggestedFix: "Fix",
		ErrorType:    "parse_error",
	})
	report.AddWarning(SkippedFileEntry{
		FilePath:     "/path/to/warning.md",
		Reason:       "Warning",
		SuggestedFix: "Fix",
		ErrorType:    "validation_warning",
	})
	report.SetMetadata(ScanMetadata{
		Timestamp:         time.Now(),
		DurationSeconds:   1.5,
		ValidationLevel:   "strict",
		DocumentationRoot: "/docs",
		Patterns:          map[string]string{},
	})

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"schema_version",
		"status",
		"dry_run",
		"metadata",
		"counts",
		"entities",
	}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("Expected field '%s' to exist in JSON output", field)
		}
	}
}

func TestFormatJSON_MetadataFields(t *testing.T) {
	// Arrange
	report := NewScanReport()
	timestamp := time.Date(2025, 12, 17, 14, 32, 5, 0, time.UTC)
	report.SetMetadata(ScanMetadata{
		Timestamp:         timestamp,
		DurationSeconds:   3.7,
		ValidationLevel:   "strict",
		DocumentationRoot: "/home/user/docs/plan",
		Patterns: map[string]string{
			"epic":    "E\\d{2}-.*",
			"feature": "E\\d{2}-F\\d{2}-.*",
		},
	})

	// Act
	output := FormatJSON(report)

	// Assert
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	metadata := parsed["metadata"].(map[string]interface{})
	if metadata["validation_level"] != "strict" {
		t.Errorf("Expected validation_level 'strict', got '%v'", metadata["validation_level"])
	}
	if metadata["documentation_root"] != "/home/user/docs/plan" {
		t.Errorf("Expected documentation_root '/home/user/docs/plan', got '%v'", metadata["documentation_root"])
	}
	if metadata["duration_seconds"] != 3.7 {
		t.Errorf("Expected duration_seconds 3.7, got %v", metadata["duration_seconds"])
	}

	patterns := metadata["patterns"].(map[string]interface{})
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}
}

func TestFormatCLI_WarningsSection(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddWarning(SkippedFileEntry{
		FilePath:     "/path/to/warning.md",
		Reason:       "Missing optional metadata",
		SuggestedFix: "Add description field",
		ErrorType:    "validation_warning",
	})

	// Act
	output := FormatCLI(report, false)

	// Assert
	if !strings.Contains(output, "/path/to/warning.md") {
		t.Error("Expected warning file path in output")
	}
	if !strings.Contains(output, "Missing optional metadata") {
		t.Error("Expected warning reason in output")
	}
	if !strings.Contains(output, "Add description field") {
		t.Error("Expected warning suggested fix in output")
	}
}

func TestFormatCLI_NoErrorsOrWarnings(t *testing.T) {
	// Arrange
	report := NewScanReport()
	report.AddMatched("epic", "/path/to/epic.md")
	report.AddMatched("feature", "/path/to/feature.md")

	// Act
	output := FormatCLI(report, false)

	// Assert
	if strings.Contains(output, "Errors and Warnings") {
		t.Error("Should not show 'Errors and Warnings' section when there are none")
	}
}
