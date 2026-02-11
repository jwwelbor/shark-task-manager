package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// epicNoteCmd is the parent command for epic note operations
var epicNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage epic notes",
	Long:  `Add, view, and manage typed notes for epics.`,
}

// epicNoteAddCmd adds a note to an epic
var epicNoteAddCmd = &cobra.Command{
	Use:   "add <epic-key> --type <type> <content>",
	Short: "Add a typed note to an epic",
	Long: `Add a typed note to an epic for context, decisions, and documentation.

Note Types:
  comment        - General observation
  decision       - Why we chose X over Y
  blocker        - What's blocking progress
  solution       - How we solved a problem
  reference      - External links, documentation
  implementation - What we actually built
  testing        - Test results, coverage
  future         - Future improvements / TODO
  question       - Unanswered questions

Examples:
  shark epic note add E07 --type decision "Chose microservices architecture"
  shark epic note add E07 --type blocker "Waiting on infrastructure team" --created-by alice
  shark epic note add E07 --type reference "https://example.com/design-doc"`,
	Args: cobra.ExactArgs(2),
	RunE: runEpicNoteAdd,
}

// epicNotesCmd lists notes for an epic
var epicNotesCmd = &cobra.Command{
	Use:   "notes <epic-key>",
	Short: "List notes for an epic",
	Long: `List all notes for an epic, optionally filtered by type.

Examples:
  shark epic notes E07                    List all notes
  shark epic notes E07 --type decision    List decision notes only
  shark epic notes E07 --type decision,solution  List multiple types
  shark epic notes E07 --json             Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicNotes,
}

func runEpicNoteAdd(cmd *cobra.Command, args []string) error {
	epicKey := args[0]
	content := args[1]

	noteTypeStr, _ := cmd.Flags().GetString("type")
	createdBy, _ := cmd.Flags().GetString("created-by")

	if noteTypeStr == "" {
		return fmt.Errorf("--type flag is required")
	}

	// Get NoteService (service layer pattern)
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize note service: %w", err)
	}

	note, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeEpic, epicKey, noteTypeStr, content, createdBy)
	if err != nil {
		return fmt.Errorf("failed to add note to epic %s: %w", epicKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	// Human-readable output
	creator := "unknown"
	if note.CreatedBy != nil {
		creator = *note.CreatedBy
	}

	fmt.Printf("Note added to epic %s\n\n", epicKey)
	fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(noteTypeStr), note.CreatedAt.Format("2006-01-02 15:04"), creator)
	fmt.Println(content)

	return nil
}

func runEpicNotes(cmd *cobra.Command, args []string) error {
	epicKey := args[0]
	noteTypesStr, _ := cmd.Flags().GetString("type")

	// Get NoteService
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize note service: %w", err)
	}

	// Parse note types filter
	var noteTypes []string
	if noteTypesStr != "" {
		noteTypes = strings.Split(noteTypesStr, ",")
		for i := range noteTypes {
			noteTypes[i] = strings.TrimSpace(noteTypes[i])
		}
	}

	notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeEpic, epicKey, noteTypes)
	if err != nil {
		return fmt.Errorf("failed to get notes for epic %s: %w", epicKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	// Human-readable output
	if len(notes) == 0 {
		fmt.Printf("No notes found for epic %s\n", epicKey)
		return nil
	}

	fmt.Printf("Epic %s (%d notes)\n\n", epicKey, len(notes))

	for _, note := range notes {
		creator := "unknown"
		if note.CreatedBy != nil {
			creator = *note.CreatedBy
		}

		fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(string(note.NoteType)), note.CreatedAt.Format("2006-01-02 15:04"), creator)
		fmt.Println(note.Content)
		fmt.Println()
	}

	return nil
}

func init() {
	// Add note subcommand to epic command
	epicCmd.AddCommand(epicNoteCmd)
	epicCmd.AddCommand(epicNotesCmd)

	// Add subcommands to note command
	epicNoteCmd.AddCommand(epicNoteAddCmd)

	// Flags for note add
	epicNoteAddCmd.Flags().StringP("type", "t", "", "Note type (required): comment, decision, blocker, solution, reference, implementation, testing, future, question")
	epicNoteAddCmd.Flags().StringP("created-by", "c", "", "Creator name (optional)")
	_ = epicNoteAddCmd.MarkFlagRequired("type")

	// Flags for notes list
	epicNotesCmd.Flags().StringP("type", "t", "", "Filter by note type (comma-separated for multiple)")
}
