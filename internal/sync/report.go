package sync

import (
	"fmt"
	"strings"
)

// FormatReport formats a SyncReport for human-readable output
func FormatReport(report *SyncReport, dryRun bool) string {
	var sb strings.Builder

	if dryRun {
		sb.WriteString("DRY-RUN MODE: No changes will be made\n\n")
	}

	sb.WriteString("Sync Summary:\n")
	sb.WriteString(fmt.Sprintf("  Files scanned:      %d\n", report.FilesScanned))

	// Show incremental filtering statistics if applicable
	if report.FilesFiltered != report.FilesScanned || report.FilesSkipped > 0 {
		sb.WriteString(fmt.Sprintf("  Files filtered:     %d\n", report.FilesFiltered))
		sb.WriteString(fmt.Sprintf("  Files skipped:      %d (unchanged)\n", report.FilesSkipped))
	}

	sb.WriteString(fmt.Sprintf("  Tasks imported:     %d\n", report.TasksImported))
	sb.WriteString(fmt.Sprintf("  Tasks updated:      %d\n", report.TasksUpdated))
	sb.WriteString(fmt.Sprintf("  Conflicts resolved: %d\n", report.ConflictsResolved))

	if report.KeysGenerated > 0 {
		sb.WriteString(fmt.Sprintf("  Keys generated:     %d\n", report.KeysGenerated))
	}

	if report.TasksDeleted > 0 {
		sb.WriteString(fmt.Sprintf("  Tasks deleted:      %d\n", report.TasksDeleted))
	}

	sb.WriteString(fmt.Sprintf("  Warnings:           %d\n", len(report.Warnings)))
	sb.WriteString(fmt.Sprintf("  Errors:             %d\n", len(report.Errors)))

	// Show pattern match statistics
	if len(report.PatternMatches) > 0 {
		sb.WriteString("\nPattern Matches:\n")
		for pattern, count := range report.PatternMatches {
			sb.WriteString(fmt.Sprintf("  %s: %d files\n", pattern, count))
		}
	}

	// Show conflicts
	if len(report.Conflicts) > 0 {
		sb.WriteString("\nConflicts:\n")
		for _, conflict := range report.Conflicts {
			sb.WriteString(fmt.Sprintf("  %s:\n", conflict.TaskKey))
			sb.WriteString(fmt.Sprintf("    Field:    %s\n", conflict.Field))
			sb.WriteString(fmt.Sprintf("    Database: %q\n", conflict.DatabaseValue))
			sb.WriteString(fmt.Sprintf("    File:     %q\n", conflict.FileValue))
		}
	}

	// Show warnings
	if len(report.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, warning := range report.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
	}

	// Show errors
	if len(report.Errors) > 0 {
		sb.WriteString("\nErrors:\n")
		for _, err := range report.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", err))
		}
	}

	return sb.String()
}
