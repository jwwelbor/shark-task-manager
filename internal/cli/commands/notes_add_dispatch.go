package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// notesAddCmd is the verb-first unified note-add dispatch. It auto-detects the
// entity type from the key and forwards to the shared NoteService, mirroring
// the per-entity `shark <entity> note add` commands. This is the preferred
// surface; the entity-first forms remain for backward compatibility.
var notesAddCmd = &cobra.Command{
	Use:   "add <KEY> --type <type> <content>",
	Short: "Add a typed note to any entity (auto-detects type from key)",
	Long: `Add a typed note to any entity. Entity type is auto-detected from the key format.

Key format detection:
  E##                Epic
  E##-F##            Feature
  E##-F##-### or T-* Task
  B###               Bug
  CC-###             Change card (C###/CC### aliases accepted)
  TD-###             Tech-debt
  I-YYYY-MM-DD-##    Idea
  S###               Sprint

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
  shark notes add E07-F01-001 --type decision "Chose JWT over sessions"
  shark notes add E07-F01     --type blocker  "Waiting on auth schema"
  shark notes add B042        --type solution "Fixed by debouncing input handler"
  shark notes add CC-007      --type comment  "Approved by product"
  shark notes add TD-003      --type reference "See ADR-014 for context" --created-by alice`,
	Args: cobra.ExactArgs(2),
	RunE: runNotesAdd,
}

func init() {
	notesAddCmd.Flags().StringP("type", "t", "", "Note type (required): comment, decision, blocker, solution, reference, implementation, testing, future, question")
	notesAddCmd.Flags().StringP("created-by", "c", "", "Creator name (optional)")
	_ = notesAddCmd.MarkFlagRequired("type")

	notesCmd.AddCommand(notesAddCmd)
}

func runNotesAdd(cmd *cobra.Command, args []string) error {
	key := args[0]
	content := args[1]

	noteTypeStr, _ := cmd.Flags().GetString("type")
	createdBy, _ := cmd.Flags().GetString("created-by")

	entityType, entityName, err := resolveEntityFromKey(key)
	if err != nil {
		return err
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
	if note.CreatedBy != nil && *note.CreatedBy != "" {
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
