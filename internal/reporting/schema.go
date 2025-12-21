package reporting

// GetJSONSchema returns the JSON schema for the scan report
func GetJSONSchema() string {
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Shark Scan Report",
  "description": "Schema for Shark Task Manager scan report output",
  "version": "1.0",
  "type": "object",
  "required": [
    "schema_version",
    "status",
    "dry_run",
    "metadata",
    "counts",
    "entities"
  ],
  "properties": {
    "schema_version": {
      "type": "string",
      "description": "Version of the report schema (semver format)",
      "pattern": "^\\d+\\.\\d+$",
      "example": "1.0"
    },
    "status": {
      "type": "string",
      "description": "Overall scan status",
      "enum": ["success", "failure"]
    },
    "dry_run": {
      "type": "boolean",
      "description": "Whether this was a dry run (no database changes committed)"
    },
    "metadata": {
      "type": "object",
      "description": "Scan execution metadata",
      "required": [
        "timestamp",
        "duration_seconds",
        "validation_level",
        "documentation_root",
        "patterns"
      ],
      "properties": {
        "timestamp": {
          "type": "string",
          "format": "date-time",
          "description": "ISO 8601 timestamp when scan started"
        },
        "duration_seconds": {
          "type": "number",
          "description": "Total scan duration in seconds with 1 decimal precision",
          "minimum": 0
        },
        "validation_level": {
          "type": "string",
          "description": "Validation level used during scan",
          "enum": ["strict", "balanced", "permissive"]
        },
        "documentation_root": {
          "type": "string",
          "description": "Absolute path to documentation root directory"
        },
        "patterns": {
          "type": "object",
          "description": "Regex patterns applied during scan",
          "additionalProperties": {
            "type": "string"
          }
        }
      }
    },
    "counts": {
      "type": "object",
      "description": "Overall scan statistics",
      "required": ["scanned", "matched", "skipped"],
      "properties": {
        "scanned": {
          "type": "integer",
          "description": "Total files scanned",
          "minimum": 0
        },
        "matched": {
          "type": "integer",
          "description": "Files successfully matched and imported",
          "minimum": 0
        },
        "skipped": {
          "type": "integer",
          "description": "Files skipped due to errors or validation failures",
          "minimum": 0
        }
      }
    },
    "entities": {
      "type": "object",
      "description": "Per-entity-type breakdown",
      "required": ["epics", "features", "tasks", "related_docs"],
      "properties": {
        "epics": {
          "$ref": "#/definitions/EntityCounts"
        },
        "features": {
          "$ref": "#/definitions/EntityCounts"
        },
        "tasks": {
          "$ref": "#/definitions/EntityCounts"
        },
        "related_docs": {
          "$ref": "#/definitions/EntityCounts"
        }
      }
    },
    "skipped_files": {
      "type": "array",
      "description": "List of all skipped files with details",
      "items": {
        "$ref": "#/definitions/SkippedFileEntry"
      }
    },
    "errors": {
      "type": "array",
      "description": "List of errors that prevented file import",
      "items": {
        "$ref": "#/definitions/SkippedFileEntry"
      }
    },
    "warnings": {
      "type": "array",
      "description": "List of warnings (file imported but with issues)",
      "items": {
        "$ref": "#/definitions/SkippedFileEntry"
      }
    }
  },
  "definitions": {
    "EntityCounts": {
      "type": "object",
      "description": "Matched and skipped counts for an entity type",
      "required": ["matched", "skipped"],
      "properties": {
        "matched": {
          "type": "integer",
          "description": "Number of entities successfully matched",
          "minimum": 0
        },
        "skipped": {
          "type": "integer",
          "description": "Number of entities skipped",
          "minimum": 0
        }
      }
    },
    "SkippedFileEntry": {
      "type": "object",
      "description": "Details about a skipped file",
      "required": ["file_path", "reason", "suggested_fix", "error_type"],
      "properties": {
        "file_path": {
          "type": "string",
          "description": "Absolute path to the skipped file"
        },
        "reason": {
          "type": "string",
          "description": "Human-readable description of why file was skipped"
        },
        "suggested_fix": {
          "type": "string",
          "description": "Actionable suggestion for resolving the issue"
        },
        "error_type": {
          "type": "string",
          "description": "Machine-readable error type for programmatic handling",
          "enum": [
            "parse_error",
            "validation_failure",
            "validation_warning",
            "pattern_mismatch",
            "file_access_error"
          ]
        },
        "line_number": {
          "type": "integer",
          "description": "Line number where error occurred (for parse errors)",
          "minimum": 1
        }
      }
    }
  }
}`
}
