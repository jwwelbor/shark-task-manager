package cli

import (
	"fmt"
	"strings"
)

// FormatEntityCreationMessage formats a human-readable creation message for any
// entity type. The output is intentionally generic — callers no longer pass a
// list of required sections. The message either tells the user a placeholder
// file was created (and to edit it) or that an existing file was linked.
func FormatEntityCreationMessage(entityType, entityKey, entityTitle, filePath, projectRoot string, fileWasLinked bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Created %s %s: %s\n", entityType, entityKey, entityTitle))
	if fileWasLinked {
		sb.WriteString("📎 LINKED TO EXISTING FILE\n\n")
		sb.WriteString(fmt.Sprintf("File: %s\n\n", filePath))
		sb.WriteString("No action required - using existing file content.\n")
	} else {
		sb.WriteString("⚠️  PLACEHOLDER FILE CREATED - EDITING REQUIRED\n\n")
		sb.WriteString(fmt.Sprintf("File: %s\n\n", filePath))
		sb.WriteString("Edit the file to fill in the relevant sections.\n")
	}
	return sb.String()
}

// FormatEntityCreationJSON formats JSON output for entity creation. The shape
// is intentionally minimal — callers no longer pass a list of required
// sections. `next_commands` is populated for known entity types and is empty
// (but always present) for unknown types.
func FormatEntityCreationJSON(entityType, entityKey, entityTitle, filePath, projectRoot string) map[string]interface{} {
	result := make(map[string]interface{})

	result["status"] = "created"
	result["entity_type"] = entityType
	result["key"] = entityKey
	result["title"] = entityTitle
	result["file_path"] = filePath
	result["file_state"] = "placeholder"
	result["requires_editing"] = true

	requiredActions := []map[string]interface{}{
		{
			"action":   "edit_file",
			"path":     filePath,
			"priority": "blocking",
		},
	}
	result["required_actions"] = requiredActions

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
			fmt.Sprintf("shark task next-status %s", entityKey),
		}
	case "bug":
		nextCommands = []string{
			fmt.Sprintf("shark bug get %s", entityKey),
		}
	case "change":
		nextCommands = []string{
			fmt.Sprintf("shark change get %s", entityKey),
		}
	case "tech-debt":
		nextCommands = []string{
			fmt.Sprintf("shark td get %s", entityKey),
		}
	default:
		nextCommands = []string{}
	}
	result["next_commands"] = nextCommands

	return result
}
