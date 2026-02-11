package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// featureNoteCmd is the parent command for feature note operations
var featureNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage feature notes",
	Long:  `Add, view, and manage typed notes for features.`,
}

// featureNoteAddCmd adds a note to a feature
var featureNoteAddCmd = &cobra.Command{
	Use:   "add <feature-key> --type <type> <content>",
	Short: "Add a typed note to a feature",
	Long: `Add a typed note to a feature for context, decisions, and documentation.

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
  shark feature note add E07-F01 --type decision "Using JWT for auth"
  shark feature note add E07-F01 --type blocker "API spec incomplete" --created-by bob
  shark feature note add E07-F01 --type reference "https://example.com/spec"`,
	Args: cobra.ExactArgs(2),
	RunE: runFeatureNoteAdd,
}

// featureNotesCmd lists notes for a feature
var featureNotesCmd = &cobra.Command{
	Use:   "notes <feature-key>",
	Short: "List notes for a feature",
	Long: `List all notes for a feature, optionally filtered by type.

Examples:
  shark feature notes E07-F01                    List all notes
  shark feature notes E07-F01 --type decision    List decision notes only
  shark feature notes E07-F01 --type decision,solution  List multiple types
  shark feature notes E07-F01 --json             Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureNotes,
}

func runFeatureNoteAdd(cmd *cobra.Command, args []string) error {
	featureKey := args[0]
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

	note, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeFeature, featureKey, noteTypeStr, content, createdBy)
	if err != nil {
		return fmt.Errorf("failed to add note to feature %s: %w", featureKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	// Human-readable output
	creator := "unknown"
	if note.CreatedBy != nil {
		creator = *note.CreatedBy
	}

	fmt.Printf("Note added to feature %s\n\n", featureKey)
	fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(noteTypeStr), note.CreatedAt.Format("2006-01-02 15:04"), creator)
	fmt.Println(content)

	return nil
}

func runFeatureNotes(cmd *cobra.Command, args []string) error {
	featureKey := args[0]
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

	notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeFeature, featureKey, noteTypes)
	if err != nil {
		return fmt.Errorf("failed to get notes for feature %s: %w", featureKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	// Human-readable output
	if len(notes) == 0 {
		fmt.Printf("No notes found for feature %s\n", featureKey)
		return nil
	}

	fmt.Printf("Feature %s (%d notes)\n\n", featureKey, len(notes))

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
	// Add note subcommand to feature command
	featureCmd.AddCommand(featureNoteCmd)
	featureCmd.AddCommand(featureNotesCmd)

	// Add subcommands to note command
	featureNoteCmd.AddCommand(featureNoteAddCmd)

	// Flags for note add
	featureNoteAddCmd.Flags().StringP("type", "t", "", "Note type (required): comment, decision, blocker, solution, reference, implementation, testing, future, question")
	featureNoteAddCmd.Flags().StringP("created-by", "c", "", "Creator name (optional)")
	_ = featureNoteAddCmd.MarkFlagRequired("type")

	// Flags for notes list
	featureNotesCmd.Flags().StringP("type", "t", "", "Filter by note type (comma-separated for multiple)")
}
