package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// articleEntity returns "a <name>" or "an <name>" depending on the first letter.
func articleEntity(name string) string {
	if len(name) > 0 && (name[0] == 'a' || name[0] == 'e' || name[0] == 'i' || name[0] == 'o' || name[0] == 'u') {
		return "an " + name
	}
	return "a " + name
}

// entityTypeFromName maps entity name strings to models.EntityType constants.
func entityTypeFromName(name string) models.EntityType {
	switch name {
	case "epic":
		return models.EntityTypeEpic
	case "feature":
		return models.EntityTypeFeature
	case "task":
		return models.EntityTypeTask
	case "bug":
		return models.EntityTypeBug
	case "change":
		return models.EntityTypeChange
	default:
		return models.EntityType(name)
	}
}

// makeNoteCmd creates a "note" parent command for the given entity type.
// It includes an "add" subcommand that uses a generic handler.
func makeNoteCmd(entityName string) *cobra.Command {
	entityType := entityTypeFromName(entityName)

	noteCmd := &cobra.Command{
		Use:   "note",
		Short: fmt.Sprintf("Manage %s notes", entityName),
		Long:  fmt.Sprintf("Add, view, and manage typed notes for %ss.", entityName),
	}

	addCmd := &cobra.Command{
		Use:   fmt.Sprintf("add <%s-key> --type <type> <content>", entityName),
		Short: fmt.Sprintf("Add a typed note to %s", articleEntity(entityName)),
		Long: fmt.Sprintf(`Add a typed note to a %s for context, decisions, and documentation.

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
  shark %s note add <key> --type decision "Rationale for approach"
  shark %s note add <key> --type blocker "Waiting on dependency" --created-by alice`, entityName, entityName, entityName),
		Args: cobra.ExactArgs(2),
		RunE: makeRunNoteAdd(entityName, entityType),
	}

	addCmd.Flags().StringP("type", "t", "", "Note type (required): comment, decision, blocker, solution, reference, implementation, testing, future, question")
	addCmd.Flags().StringP("created-by", "c", "", "Creator name (optional)")
	_ = addCmd.MarkFlagRequired("type")

	noteCmd.AddCommand(addCmd)
	return noteCmd
}

// makeNotesCmd creates a "notes" list command for the given entity type.
func makeNotesCmd(entityName string) *cobra.Command {
	entityType := entityTypeFromName(entityName)

	notesCmd := &cobra.Command{
		Use:   fmt.Sprintf("notes <%s-key>", entityName),
		Short: fmt.Sprintf("List notes for a %s", entityName),
		Long: fmt.Sprintf(`List all notes for a %s, optionally filtered by type.

Examples:
  shark %s notes <key>                    List all notes
  shark %s notes <key> --type decision    List decision notes only
  shark %s notes <key> --json             Output as JSON`, entityName, entityName, entityName, entityName),
		Args: cobra.ExactArgs(1),
		RunE: makeRunNotesList(entityName, entityType),
	}

	notesCmd.Flags().StringP("type", "t", "", "Filter by note type (comma-separated for multiple)")
	return notesCmd
}

// makeRunNoteAdd returns a generic note-add handler for the given entity type.
func makeRunNoteAdd(entityName string, entityType models.EntityType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key := args[0]
		content := args[1]

		noteTypeStr, _ := cmd.Flags().GetString("type")
		createdBy, _ := cmd.Flags().GetString("created-by")

		if noteTypeStr == "" {
			return fmt.Errorf("--type flag is required")
		}

		noteSvc, err := cli.GetNoteService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get note service: %w", err)
		}

		note, err := noteSvc.AddNote(cmd.Context(), entityType, key, noteTypeStr, content, createdBy)
		if err != nil {
			return fmt.Errorf("failed to add note to %s %s: %w", entityName, key, err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(note)
		}

		creator := "unknown"
		if note.CreatedBy != nil {
			creator = *note.CreatedBy
		}

		ts := note.CreatedAt
		if ts.IsZero() {
			ts = time.Now()
		}

		fmt.Printf("Note added to %s %s\n\n", entityName, key)
		fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(noteTypeStr), ts.Format("2006-01-02 15:04"), creator)
		fmt.Println(content)

		return nil
	}
}

// makeRunNotesList returns a generic notes-list handler for the given entity type.
func makeRunNotesList(entityName string, entityType models.EntityType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key := args[0]
		noteTypesStr, _ := cmd.Flags().GetString("type")

		var noteTypes []string
		if noteTypesStr != "" {
			noteTypes = strings.Split(noteTypesStr, ",")
			for i := range noteTypes {
				noteTypes[i] = strings.TrimSpace(noteTypes[i])
			}
		}

		noteSvc, err := cli.GetNoteService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get note service: %w", err)
		}

		notes, err := noteSvc.ListNotes(cmd.Context(), entityType, key, noteTypes)
		if err != nil {
			return fmt.Errorf("failed to get notes for %s %s: %w", entityName, key, err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(notes)
		}

		if len(notes) == 0 {
			fmt.Printf("No notes found for %s %s\n", entityName, key)
			return nil
		}

		// Capitalize first letter of entity name for display
		displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
		fmt.Printf("%s %s (%d notes)\n\n", displayName, key, len(notes))

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
}
