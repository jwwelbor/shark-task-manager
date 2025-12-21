package reporting

import (
	"time"
)

// ScanReport represents a comprehensive scan report
type ScanReport struct {
	SchemaVersion string             `json:"schema_version"`
	Status        string             `json:"status"`
	DryRun        bool               `json:"dry_run"`
	Metadata      ScanMetadata       `json:"metadata"`
	Counts        ScanCounts         `json:"counts"`
	Entities      EntityBreakdown    `json:"entities"`
	SkippedFiles  []SkippedFileEntry `json:"skipped_files,omitempty"`
	Errors        []SkippedFileEntry `json:"errors,omitempty"`
	Warnings      []SkippedFileEntry `json:"warnings,omitempty"`
}

// ScanMetadata contains scan execution metadata
type ScanMetadata struct {
	Timestamp         time.Time         `json:"timestamp"`
	DurationSeconds   float64           `json:"duration_seconds"`
	ValidationLevel   string            `json:"validation_level"`
	DocumentationRoot string            `json:"documentation_root"`
	Patterns          map[string]string `json:"patterns"`
}

// ScanCounts contains overall scan statistics
type ScanCounts struct {
	Scanned int `json:"scanned"`
	Matched int `json:"matched"`
	Skipped int `json:"skipped"`
}

// EntityBreakdown contains per-entity-type statistics
type EntityBreakdown struct {
	Epics       EntityCounts `json:"epics"`
	Features    EntityCounts `json:"features"`
	Tasks       EntityCounts `json:"tasks"`
	RelatedDocs EntityCounts `json:"related_docs"`
}

// EntityCounts contains matched/skipped counts for an entity type
type EntityCounts struct {
	Matched int `json:"matched"`
	Skipped int `json:"skipped"`
}

// SkippedFileEntry represents a file that was skipped during scanning
type SkippedFileEntry struct {
	FilePath     string `json:"file_path"`
	Reason       string `json:"reason"`
	SuggestedFix string `json:"suggested_fix"`
	ErrorType    string `json:"error_type"`
	LineNumber   *int   `json:"line_number,omitempty"`
}

// NewScanReport creates a new ScanReport with initialized fields
func NewScanReport() *ScanReport {
	return &ScanReport{
		SchemaVersion: "1.0",
		Status:        "success",
		DryRun:        false,
		Metadata: ScanMetadata{
			Timestamp: time.Now(),
			Patterns:  make(map[string]string),
		},
		Counts: ScanCounts{},
		Entities: EntityBreakdown{
			Epics:       EntityCounts{},
			Features:    EntityCounts{},
			Tasks:       EntityCounts{},
			RelatedDocs: EntityCounts{},
		},
		SkippedFiles: []SkippedFileEntry{},
		Errors:       []SkippedFileEntry{},
		Warnings:     []SkippedFileEntry{},
	}
}

// AddScanned increments the scanned count
func (r *ScanReport) AddScanned(entityType string) {
	r.Counts.Scanned++
}

// AddMatched increments the matched count for the specified entity type
func (r *ScanReport) AddMatched(entityType string, filePath string) {
	r.Counts.Scanned++
	r.Counts.Matched++

	switch entityType {
	case "epic":
		r.Entities.Epics.Matched++
	case "feature":
		r.Entities.Features.Matched++
	case "task":
		r.Entities.Tasks.Matched++
	case "related":
		r.Entities.RelatedDocs.Matched++
	}
}

// AddSkipped adds a skipped file entry and increments counts
func (r *ScanReport) AddSkipped(entityType string, entry SkippedFileEntry) {
	r.Counts.Scanned++
	r.Counts.Skipped++
	r.SkippedFiles = append(r.SkippedFiles, entry)

	switch entityType {
	case "epic":
		r.Entities.Epics.Skipped++
	case "feature":
		r.Entities.Features.Skipped++
	case "task":
		r.Entities.Tasks.Skipped++
	case "related":
		r.Entities.RelatedDocs.Skipped++
	}
}

// AddError adds an error entry to the report
func (r *ScanReport) AddError(entry SkippedFileEntry) {
	r.Errors = append(r.Errors, entry)
}

// AddWarning adds a warning entry to the report
func (r *ScanReport) AddWarning(entry SkippedFileEntry) {
	r.Warnings = append(r.Warnings, entry)
}

// SetMetadata sets the scan metadata
func (r *ScanReport) SetMetadata(metadata ScanMetadata) {
	r.Metadata = metadata
}

// SetDryRun sets the dry run flag
func (r *ScanReport) SetDryRun(dryRun bool) {
	r.DryRun = dryRun
}

// GetStatus returns the current status based on errors
func (r *ScanReport) GetStatus() string {
	if len(r.Errors) > 0 {
		return "failure"
	}
	return "success"
}
