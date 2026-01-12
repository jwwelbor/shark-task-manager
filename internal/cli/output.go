package cli

import (
	"fmt"
	"strings"
)

// GetRequiredSectionsForEntityType returns the required sections for a given entity type
func GetRequiredSectionsForEntityType(entityType string) []string {
	switch strings.ToLower(entityType) {
	case "epic":
		return []string{"Vision & Goals", "Success Criteria", "Features"}
	case "feature":
		return []string{"Requirements", "Design", "Test Plan"}
	case "task":
		return []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"}
	default:
		// Default to task sections for unknown types
		return []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"}
	}
}

// FormatEntityCreationMessage formats a human-readable creation message for epic/feature/task
func FormatEntityCreationMessage(entityType, entityKey, entityTitle, filePath, projectRoot string, fileWasLinked bool, requiredSections []string) string {
	var sb strings.Builder

	// Success header
	sb.WriteString(fmt.Sprintf("✅ Created %s %s: %s\n", entityType, entityKey, entityTitle))

	if fileWasLinked {
		// File was linked to existing content
		sb.WriteString("📎 LINKED TO EXISTING FILE\n\n")
		sb.WriteString(fmt.Sprintf("File: %s\n\n", filePath))
		sb.WriteString("No action required - using existing file content.\n")
	} else {
		// New placeholder file was created
		sb.WriteString("⚠️  PLACEHOLDER FILE CREATED - EDITING REQUIRED\n\n")
		sb.WriteString(fmt.Sprintf("File: %s\n\n", filePath))
		sb.WriteString("REQUIRED ACTIONS:\n")
		sb.WriteString("1. Edit the task file to add implementation details\n")
		sb.WriteString("2. Fill in required sections:\n")
		for _, section := range requiredSections {
			sb.WriteString(fmt.Sprintf("   • %s\n", section))
		}
	}

	return sb.String()
}

// FormatEntityCreationJSON formats JSON output for entity creation
func FormatEntityCreationJSON(entityType, entityKey, entityTitle, filePath, projectRoot string, requiredSections []string) map[string]interface{} {
	result := make(map[string]interface{})

	// Basic fields
	result["status"] = "created"
	result["entity_type"] = entityType
	result["key"] = entityKey
	result["title"] = entityTitle
	result["file_path"] = filePath
	result["file_state"] = "placeholder"
	result["requires_editing"] = true

	// Required actions
	requiredActions := []map[string]interface{}{
		{
			"action":            "edit_file",
			"path":              filePath,
			"required_sections": requiredSections,
			"priority":          "blocking",
		},
	}
	result["required_actions"] = requiredActions

	// Next commands based on entity type
	var nextCommands []string
	switch strings.ToLower(entityType) {
	case "epic":
		nextCommands = []string{
			fmt.Sprintf("shark epic get %s", entityKey),
			fmt.Sprintf("shark feature create %s \"Feature title\"", entityKey),
		}
	case "feature":
		nextCommands = []string{
			fmt.Sprintf("shark feature get %s", entityKey),
			fmt.Sprintf("shark task create %s \"Task title\" --agent=backend", entityKey),
		}
	case "task":
		nextCommands = []string{
			fmt.Sprintf("shark task start %s", entityKey),
		}
	}
	result["next_commands"] = nextCommands

	return result
}
