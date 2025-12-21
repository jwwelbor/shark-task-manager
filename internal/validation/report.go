package validation

import (
	"encoding/json"
	"fmt"
	"io"
)

// ValidationResult holds the complete validation results
type ValidationResult struct {
	BrokenFilePaths []ValidationFailure `json:"broken_file_paths,omitempty"`
	OrphanedRecords []ValidationFailure `json:"orphaned_records,omitempty"`
	Summary         ValidationSummary   `json:"summary"`
	DurationMs      int64               `json:"duration_ms"`
}

// ValidationFailure describes a specific validation failure
type ValidationFailure struct {
	EntityType        string `json:"entity_type"`
	EntityKey         string `json:"entity_key"`
	FilePath          string `json:"file_path,omitempty"`
	MissingParentType string `json:"missing_parent_type,omitempty"`
	MissingParentID   int64  `json:"missing_parent_id,omitempty"`
	Issue             string `json:"issue"`
	SuggestedFix      string `json:"suggested_fix"`
}

// ValidationSummary provides high-level validation statistics
type ValidationSummary struct {
	TotalChecked    int `json:"total_checked"`
	TotalIssues     int `json:"total_issues"`
	BrokenFilePaths int `json:"broken_file_paths"`
	OrphanedRecords int `json:"orphaned_records"`
}

// IsSuccess returns true if validation found no issues
func (r *ValidationResult) IsSuccess() bool {
	return r.Summary.TotalIssues == 0
}

// FormatJSON outputs the validation result as JSON
func (r *ValidationResult) FormatJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// FormatHuman outputs the validation result in human-readable format
func (r *ValidationResult) FormatHuman(w io.Writer) error {
	// Header
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Shark Validation Report")
	fmt.Fprintln(w, "=======================")
	fmt.Fprintln(w, "")

	// Summary
	fmt.Fprintln(w, "Summary")
	fmt.Fprintln(w, "-------")
	fmt.Fprintf(w, "Total entities validated: %d\n", r.Summary.TotalChecked)
	fmt.Fprintf(w, "  - Issues found: %d\n", r.Summary.TotalIssues)
	fmt.Fprintf(w, "  - Broken file paths: %d\n", r.Summary.BrokenFilePaths)
	fmt.Fprintf(w, "  - Orphaned records: %d\n", r.Summary.OrphanedRecords)
	fmt.Fprintf(w, "Duration: %dms\n", r.DurationMs)
	fmt.Fprintln(w, "")

	// Broken file paths
	if len(r.BrokenFilePaths) > 0 {
		fmt.Fprintln(w, "Broken File Paths")
		fmt.Fprintln(w, "-----------------")
		for _, failure := range r.BrokenFilePaths {
			fmt.Fprintf(w, "  ✗ %s [%s]\n", failure.EntityKey, failure.EntityType)
			fmt.Fprintf(w, "    Path: %s\n", failure.FilePath)
			fmt.Fprintf(w, "    Issue: %s\n", failure.Issue)
			fmt.Fprintf(w, "    Suggestion: %s\n", failure.SuggestedFix)
			fmt.Fprintln(w, "")
		}
	}

	// Orphaned records
	if len(r.OrphanedRecords) > 0 {
		fmt.Fprintln(w, "Orphaned Records")
		fmt.Fprintln(w, "----------------")
		for _, failure := range r.OrphanedRecords {
			fmt.Fprintf(w, "  ✗ %s [%s]\n", failure.EntityKey, failure.EntityType)
			fmt.Fprintf(w, "    Missing parent: %s (ID: %d)\n", failure.MissingParentType, failure.MissingParentID)
			fmt.Fprintf(w, "    Issue: %s\n", failure.Issue)
			fmt.Fprintf(w, "    Suggestion: %s\n", failure.SuggestedFix)
			fmt.Fprintln(w, "")
		}
	}

	// Final status
	fmt.Fprintln(w, "Validation Result")
	fmt.Fprintln(w, "-----------------")
	if r.IsSuccess() {
		fmt.Fprintln(w, "✓ All validations passed!")
	} else {
		fmt.Fprintf(w, "✗ VALIDATION FAILED: Found %d issue(s)\n", r.Summary.TotalIssues)
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Next Steps:")
		if r.Summary.BrokenFilePaths > 0 {
			fmt.Fprintln(w, "  - Run 'shark sync --incremental' to update file paths")
		}
		if r.Summary.OrphanedRecords > 0 {
			fmt.Fprintln(w, "  - Review orphaned records and create missing parents or delete orphans")
		}
	}
	fmt.Fprintln(w, "")

	return nil
}
