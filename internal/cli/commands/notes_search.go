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

// notesSearchCmd searches notes across all tasks
var notesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search note content across all tasks",
	Long: `Search for notes containing the specified query across all tasks.

The search is case-insensitive and supports filtering by epic, feature, and note type.

Examples:
  shark notes search "singleton pattern"
  shark notes search "dark mode" --epic E10
  shark notes search "API" --feature E10-F01
  shark notes search "singleton" --type decision
  shark notes search "bug" --type decision,solution --epic E10
  shark notes search "performance" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runNotesSearch,
}

// NoteSearchResult represents a search result with task context
type NoteSearchResult struct {
	TaskKey   string           `json:"task_key"`
	TaskTitle string           `json:"task_title"`
	Note      *models.TaskNote `json:"note"`
}

// runNotesSearch handles the notes search command
func runNotesSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	noteTypesStr, _ := cmd.Flags().GetString("type")

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	dbWrapper := repoDb
	noteRepo := repository.NewTaskNoteRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

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

	// Search notes
	notes, err := noteRepo.Search(ctx, query, noteTypes, epicKey, featureKey)
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

	// Get task details for each note
	results := make([]NoteSearchResult, 0, len(notes))
	taskCache := make(map[int64]*models.Task)

	for _, note := range notes {
		// Check cache first
		task, ok := taskCache[note.TaskID]
		if !ok {
			task, err = taskRepo.GetByID(ctx, note.TaskID)
			if err != nil {
				// Skip this note if we can't get the task
				continue
			}
			taskCache[note.TaskID] = task
		}

		results = append(results, NoteSearchResult{
			TaskKey:   task.Key,
			TaskTitle: task.Title,
			Note:      note,
		})
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

		fmt.Printf("%s: %s\n", result.TaskKey, result.TaskTitle)
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
}
