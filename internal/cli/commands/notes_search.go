package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// notesCmd is the parent command for note search operations
var notesCmd = &cobra.Command{
	Use:     "notes",
	Short:   "Search notes across all tasks",
	GroupID: "details",
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
	query := args[0]

	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	noteTypesStr, _ := cmd.Flags().GetString("type")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	entityTypeStr, _ := cmd.Flags().GetString("entity-type")

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	dbWrapper := repoDb
	noteRepo := repository.NewEntityNoteRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)

	// Parse note types filter
	var noteTypes []string
	if noteTypesStr != "" {
		noteTypes = strings.Split(noteTypesStr, ",")
		// Validate each note type
		for i, nt := range noteTypes {
			noteTypes[i] = strings.TrimSpace(nt)
			if err := models.ValidateNoteType(noteTypes[i]); err != nil {
				return err
			}
		}
	}

	// Parse entity type filter
	var entityType *models.EntityType
	if entityTypeStr != "" {
		entityTypeStr = strings.ToLower(strings.TrimSpace(entityTypeStr))
		et := models.EntityType(entityTypeStr)
		if !models.ValidEntityTypes[et] {
			return fmt.Errorf("invalid entity-type: %s (must be one of: epic, feature, task)", entityTypeStr)
		}
		entityType = &et
	}

	// Search notes with time period filtering
	var notes []*models.EntityNote
	if since != "" || until != "" {
		notes, err = noteRepo.SearchWithTimePeriod(ctx, query, noteTypes, epicKey, featureKey, since, until)
	} else {
		notes, err = noteRepo.Search(ctx, query, noteTypes, entityType, epicKey, featureKey)
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

	// Get entity details for each note
	results := make([]NoteSearchResult, 0, len(notes))
	taskCache := make(map[int64]*models.Task)
	epicCache := make(map[int64]*models.Epic)
	featureCache := make(map[int64]*models.Feature)

	for _, note := range notes {
		result := NoteSearchResult{
			EntityType: string(note.EntityType),
			Note:       note,
		}

		switch note.EntityType {
		case models.EntityTypeTask:
			// Check cache first
			task, ok := taskCache[note.EntityID]
			if !ok {
				task, err = taskRepo.GetByID(ctx, note.EntityID)
				if err != nil {
					// Skip this note if we can't get the task
					continue
				}
				taskCache[note.EntityID] = task
			}
			result.EntityKey = task.Key
			result.EntityName = task.Title
			// Set legacy fields for backward compatibility
			result.TaskKey = task.Key
			result.TaskTitle = task.Title

		case models.EntityTypeEpic:
			// Check cache first
			epic, ok := epicCache[note.EntityID]
			if !ok {
				epic, err = epicRepo.GetByID(ctx, note.EntityID)
				if err != nil {
					// Skip this note if we can't get the epic
					continue
				}
				epicCache[note.EntityID] = epic
			}
			result.EntityKey = epic.Key
			result.EntityName = epic.Title

		case models.EntityTypeFeature:
			// Check cache first
			feature, ok := featureCache[note.EntityID]
			if !ok {
				feature, err = featureRepo.GetByID(ctx, note.EntityID)
				if err != nil {
					// Skip this note if we can't get the feature
					continue
				}
				featureCache[note.EntityID] = feature
			}
			result.EntityKey = feature.Key
			result.EntityName = feature.Title
		}

		results = append(results, result)
	}

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
