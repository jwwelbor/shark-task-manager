package reporting

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

// FormatCLI formats the scan report for CLI output
func FormatCLI(report *ScanReport, useColor bool) string {
	var sb strings.Builder

	// Helper functions for coloring
	colorize := func(text, color string) string {
		if useColor {
			return color + text + colorReset
		}
		return text
	}

	bold := func(text string) string {
		if useColor {
			return colorBold + text + colorReset
		}
		return text
	}

	// Header
	sb.WriteString(bold("Shark Scan Report"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", 17))
	sb.WriteString("\n")

	// Dry run indicator
	if report.DryRun {
		sb.WriteString(colorize("DRY RUN MODE: No database changes will be committed", colorYellow))
		sb.WriteString("\n\n")
	}

	// Metadata
	sb.WriteString(fmt.Sprintf("Scan completed at %s\n", report.Metadata.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Duration: %.1f seconds\n", report.Metadata.DurationSeconds))
	sb.WriteString(fmt.Sprintf("Validation level: %s\n", report.Metadata.ValidationLevel))
	sb.WriteString(fmt.Sprintf("Documentation root: %s\n", report.Metadata.DocumentationRoot))
	sb.WriteString("\n")

	// Summary
	sb.WriteString(bold("Summary"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 7))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Total files scanned: %d\n", report.Counts.Scanned))

	if report.Counts.Matched > 0 {
		sb.WriteString(fmt.Sprintf("  %s Matched: %d\n",
			colorize("✓", colorGreen), report.Counts.Matched))
	}
	if report.Counts.Skipped > 0 {
		sb.WriteString(fmt.Sprintf("  %s Skipped: %d\n",
			colorize("✗", colorRed), report.Counts.Skipped))
	}
	sb.WriteString("\n")

	// Breakdown by Type
	sb.WriteString(bold("Breakdown by Type"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 17))
	sb.WriteString("\n")

	formatEntitySection := func(name string, counts EntityCounts) {
		sb.WriteString(fmt.Sprintf("%s:\n", name))
		if counts.Matched > 0 {
			sb.WriteString(fmt.Sprintf("  %s Matched: %d\n",
				colorize("✓", colorGreen), counts.Matched))
		}
		if counts.Skipped > 0 {
			sb.WriteString(fmt.Sprintf("  %s Skipped: %d\n",
				colorize("✗", colorRed), counts.Skipped))
		}
		sb.WriteString("\n")
	}

	formatEntitySection("Epics", report.Entities.Epics)
	formatEntitySection("Features", report.Entities.Features)
	formatEntitySection("Tasks", report.Entities.Tasks)
	formatEntitySection("Related Docs", report.Entities.RelatedDocs)

	// Errors and Warnings
	if len(report.Errors) > 0 || len(report.Warnings) > 0 {
		sb.WriteString(bold("Errors and Warnings"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("-", 19))
		sb.WriteString("\n\n")

		// Group errors by type
		if len(report.Errors) > 0 {
			errorsByType := groupByErrorType(report.Errors)

			for _, errorType := range sortedErrorTypes(errorsByType) {
				entries := errorsByType[errorType]
				typeName := formatErrorTypeName(errorType)

				sb.WriteString(fmt.Sprintf("%s (%d):\n", typeName, len(entries)))

				for _, entry := range entries {
					sb.WriteString(fmt.Sprintf("  %s: %s",
						colorize("ERROR", colorRed), entry.FilePath))
					if entry.LineNumber != nil {
						sb.WriteString(fmt.Sprintf(":%d", *entry.LineNumber))
					}
					sb.WriteString("\n")
					sb.WriteString(fmt.Sprintf("    %s\n", entry.Reason))
					sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", entry.SuggestedFix))
					sb.WriteString("\n")
				}
			}
		}

		// Group warnings by type
		if len(report.Warnings) > 0 {
			warningsByType := groupByErrorType(report.Warnings)

			for _, errorType := range sortedErrorTypes(warningsByType) {
				entries := warningsByType[errorType]
				typeName := formatErrorTypeName(errorType)

				sb.WriteString(fmt.Sprintf("%s (%d):\n", typeName, len(entries)))

				for _, entry := range entries {
					sb.WriteString(fmt.Sprintf("  %s: %s",
						colorize("WARNING", colorYellow), entry.FilePath))
					if entry.LineNumber != nil {
						sb.WriteString(fmt.Sprintf(":%d", *entry.LineNumber))
					}
					sb.WriteString("\n")
					sb.WriteString(fmt.Sprintf("    %s\n", entry.Reason))
					sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", entry.SuggestedFix))
					sb.WriteString("\n")
				}
			}
		}
	}

	// Final summary
	sb.WriteString(bold("Scan Complete"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 13))
	sb.WriteString("\n")

	if len(report.Errors) == 0 {
		sb.WriteString(fmt.Sprintf("Successfully imported %d items:\n", report.Counts.Matched))
		if report.Entities.Epics.Matched > 0 {
			sb.WriteString(fmt.Sprintf("  - %d epics\n", report.Entities.Epics.Matched))
		}
		if report.Entities.Features.Matched > 0 {
			sb.WriteString(fmt.Sprintf("  - %d features\n", report.Entities.Features.Matched))
		}
		if report.Entities.Tasks.Matched > 0 {
			sb.WriteString(fmt.Sprintf("  - %d tasks\n", report.Entities.Tasks.Matched))
		}
		sb.WriteString("\n")
		sb.WriteString("Run 'shark validate' to verify database integrity.\n")
	} else {
		sb.WriteString(colorize(fmt.Sprintf("Scan completed with %d errors\n", len(report.Errors)), colorRed))
		sb.WriteString("Please fix the errors above and re-run the scan.\n")
	}

	return sb.String()
}

// FormatJSON formats the scan report as JSON
func FormatJSON(report *ScanReport) string {
	// Update status based on errors
	report.Status = report.GetStatus()

	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		// Fallback to error JSON if marshaling fails
		errorJSON := fmt.Sprintf(`{
  "schema_version": "1.0",
  "status": "failure",
  "error": "Failed to marshal report: %s"
}`, err.Error())
		return errorJSON
	}

	return string(jsonBytes)
}

// groupByErrorType groups error entries by their error type
func groupByErrorType(entries []SkippedFileEntry) map[string][]SkippedFileEntry {
	grouped := make(map[string][]SkippedFileEntry)
	for _, entry := range entries {
		grouped[entry.ErrorType] = append(grouped[entry.ErrorType], entry)
	}
	return grouped
}

// sortedErrorTypes returns error types sorted alphabetically
func sortedErrorTypes(grouped map[string][]SkippedFileEntry) []string {
	types := make([]string, 0, len(grouped))
	for errorType := range grouped {
		types = append(types, errorType)
	}
	sort.Strings(types)
	return types
}

// formatErrorTypeName converts error type to human-readable name
func formatErrorTypeName(errorType string) string {
	switch errorType {
	case "parse_error":
		return "Parse Errors"
	case "validation_failure":
		return "Validation Errors"
	case "validation_warning":
		return "Validation Warnings"
	case "pattern_mismatch":
		return "Pattern Mismatch Warnings"
	case "file_access_error":
		return "File Access Errors"
	default:
		return cases.Title(language.English).String(strings.ReplaceAll(errorType, "_", " "))
	}
}
