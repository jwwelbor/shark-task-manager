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
	sb.WriteString(fmt.Sprintf("  Tasks imported:     %d\n", report.TasksImported))
	sb.WriteString(fmt.Sprintf("  Tasks updated:      %d\n", report.TasksUpdated))
	sb.WriteString(fmt.Sprintf("  Conflicts resolved: %d\n", report.ConflictsResolved))

	if report.TasksDeleted > 0 {
		sb.WriteString(fmt.Sprintf("  Tasks deleted:      %d\n", report.TasksDeleted))
	}

	sb.WriteString(fmt.Sprintf("  Warnings:           %d\n", len(report.Warnings)))
	sb.WriteString(fmt.Sprintf("  Errors:             %d\n", len(report.Errors)))

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
