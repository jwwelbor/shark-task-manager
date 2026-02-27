package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// notesCmd is the parent command for note search operations
var notesCmd = &cobra.Command{
	Use:     "notes",
	Short:   "Search notes across all tasks",
	GroupID: "manage",
	Long:    `Search for notes across all tasks with optional filtering.`,
}

// notesSearchCmd searches notes across all entities
var notesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search note content across all entities (epics, features, tasks)",
	Long: `Search for notes containing the specified query across all entities.

The search is case-insensitive and supports filtering by entity type, epic, feature, note type, and time period.

Examples:
  shark notes search "singleton pattern"
  shark notes search "dark mode" --entity-type epic
  shark notes search "API" --entity-type feature
  shark notes search "database" --entity-type task --epic E10
  shark notes search "API" --feature E10-F01
  shark notes search "singleton" --type decision
  shark notes search "bug" --type decision,solution --epic E10
  shark notes search "missing error" --type rejection --since 2026-01-01
  shark notes search "performance" --until 2026-01-15 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runNotesSearch,
}

// NoteSearchResult represents a search result with entity context
type NoteSearchResult struct {
	EntityType string             `json:"entity_type"`
	EntityKey  string             `json:"entity_key"`
	EntityName string             `json:"entity_name"`
	Note       *models.EntityNote `json:"note"`
	// Legacy fields for backward compatibility (task-specific)
	TaskKey   string `json:"task_key,omitempty"`
	TaskTitle string `json:"task_title,omitempty"`
}

// runNotesSearch handles the notes search command
func runNotesSearch(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments and flags
	query := args[0]

	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	noteTypesStr, _ := cmd.Flags().GetString("type")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	entityTypeStr, _ := cmd.Flags().GetString("entity-type")

	// Parse note types filter (validation is handled in service)
	var noteTypes []string
	if noteTypesStr != "" {
		parts := strings.Split(noteTypesStr, ",")
		for _, nt := range parts {
			noteTypes = append(noteTypes, strings.TrimSpace(nt))
		}
	}

	// Parse entity type filter (validation is handled in service)
	var entityType *models.EntityType
	if entityTypeStr != "" {
		et := models.EntityType(strings.ToLower(strings.TrimSpace(entityTypeStr)))
		entityType = &et
	}

	// Step 2: Call service
	svc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	var notes []*models.EntityNote
	if since != "" || until != "" {
		// Validate note types before passing to service
		for _, nt := range noteTypes {
			if err := models.ValidateNoteType(nt); err != nil {
				return err
			}
		}
		notes, err = svc.SearchNotesWithTimePeriod(cmd.Context(), query, noteTypes, epicKey, featureKey, since, until)
	} else {
		// Validate entity type before passing to service
		if entityType != nil && !models.ValidEntityTypes[*entityType] {
			return fmt.Errorf("invalid entity-type: %s (must be one of: epic, feature, task)", entityTypeStr)
		}
		// Validate note types before passing to service
		for _, nt := range noteTypes {
			if err := models.ValidateNoteType(nt); err != nil {
				return err
			}
		}
		notes, err = svc.SearchNotes(cmd.Context(), query, noteTypes, entityType, epicKey, featureKey)
	}
	if err != nil {
		return fmt.Errorf("failed to search notes: %w", err)
	}

	if len(notes) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON([]NoteSearchResult{})
		}
		fmt.Printf("No results found for %q\n", query)
		return nil
	}

	// Build results with entity details (delegated to service)
	results := make([]NoteSearchResult, 0, len(notes))
	for _, note := range notes {
		details := svc.GetEntityDetails(cmd.Context(), note.EntityType, note.EntityID)
		if details == nil {
			// Skip notes where entity can't be found
			continue
		}

		result := NoteSearchResult{
			EntityType: string(note.EntityType),
			EntityKey:  details.Key,
			EntityName: details.Name,
			Note:       note,
		}

		// Set legacy fields for backward compatibility (task-specific)
		if note.EntityType == models.EntityTypeTask {
			result.TaskKey = details.Key
			result.TaskTitle = details.Name
		}

		results = append(results, result)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(results)
	}

	// Human-readable output
	fmt.Printf("Found %d result", len(results))
	if len(results) != 1 {
		fmt.Print("s")
	}
	fmt.Printf(" for %q:\n\n", query)

	for _, result := range results {
		creator := "unknown"
		if result.Note.CreatedBy != nil {
			creator = *result.Note.CreatedBy
		}

		fmt.Printf("[%s] %s: %s\n", strings.ToUpper(result.EntityType), result.EntityKey, result.EntityName)
		fmt.Printf("  [%s] %s (%s)\n", strings.ToUpper(string(result.Note.NoteType)), result.Note.CreatedAt.Format("2006-01-02 15:04"), creator)

		// Indent the content
		lines := strings.Split(result.Note.Content, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}

	return nil
}

func init() {
	// Add notes command to root
	cli.RootCmd.AddCommand(notesCmd)

	// Add search subcommand
	notesCmd.AddCommand(notesSearchCmd)

	// Flags for search
	notesSearchCmd.Flags().StringP("epic", "e", "", "Filter by epic key (e.g., E10)")
	notesSearchCmd.Flags().StringP("feature", "f", "", "Filter by feature key (e.g., E10-F01)")
	notesSearchCmd.Flags().StringP("type", "t", "", "Filter by note type (comma-separated for multiple)")
	notesSearchCmd.Flags().String("entity-type", "", "Filter by entity type (epic, feature, or task)")
	notesSearchCmd.Flags().String("since", "", "Filter notes created after date (YYYY-MM-DD format)")
	notesSearchCmd.Flags().String("until", "", "Filter notes created before date (YYYY-MM-DD format)")
}
