package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// relatedDocsCmd represents the related-docs command group
var relatedDocsCmd = &cobra.Command{
	Use:     "related-docs",
	Short:   "Manage related documents",
	GroupID: "manage",
	Aliases: []string{"docs"},
	Long: `Manage related documents linked to epics, features, or tasks.

Examples:
  shark related-docs add "Design Doc" docs/design.md --epic=E01
  shark related-docs list --epic=E01
  shark related-docs delete "Design Doc" --epic=E01`,
}

// relatedDocsAddCmd adds a document to a parent entity
var relatedDocsAddCmd = &cobra.Command{
	Use:   "add <title> <path>",
	Short: "Add a related document",
	Long: `Add a related document to an epic, feature, or task.

The document is created or retrieved if it already exists with the same title and path.
The document is then linked to exactly one parent entity (epic, feature, or task).

Examples:
  shark related-docs add "OAuth Specification" docs/oauth.md --epic=E01
  shark related-docs add "Implementation Notes" docs/notes.md --feature=E01-F01
  shark related-docs add "Task Details" docs/details.md --task=T-E01-F01-001`,
	Args: cobra.ExactArgs(2),
	RunE: runRelatedDocsAdd,
}

// relatedDocsDeleteCmd removes a document from a parent entity
var relatedDocsDeleteCmd = &cobra.Command{
	Use:   "delete <title>",
	Short: "Delete a related document link",
	Long: `Remove a document link from an epic, feature, or task.

The document itself is not deleted from the database, only the link is removed.
Delete is idempotent - it succeeds even if the document is not linked to the parent.

Examples:
  shark related-docs delete "OAuth Specification" --epic=E01
  shark related-docs delete "Implementation Notes" --feature=E01-F01
  shark related-docs delete "Task Details" --task=T-E01-F01-001`,
	Args: cobra.ExactArgs(1),
	RunE: runRelatedDocsDelete,
}

// relatedDocsListCmd lists documents for a parent entity
var relatedDocsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List related documents",
	Long: `List all documents linked to an epic, feature, or task.

Requires exactly one of --epic, --feature, or --task flags.

Examples:
  shark related-docs list --epic=E01
  shark related-docs list --feature=E01-F01 --json
  shark related-docs list --task=T-E01-F01-001`,
	RunE: runRelatedDocsListList,
}

// dispatchAddDoc links a document to an epic, feature, task, bug, or change-card based on which key is set.
func dispatchAddDoc(ctx context.Context, epic, feature, task, bug, change, title, path string) error {
	if epic != "" {
		if err := cli.GetEpicService().LinkDocument(ctx, epic, title, path); err != nil {
			return fmt.Errorf("failed to link document to epic: %w", err)
		}
		return printDocLinked(title, path, "epic", epic, 0)
	}
	if feature != "" {
		if err := cli.GetFeatureService().LinkDocument(ctx, feature, title, path); err != nil {
			return fmt.Errorf("failed to link document to feature: %w", err)
		}
		return printDocLinked(title, path, "feature", feature, 0)
	}
	if bug != "" {
		if err := cli.GetBugService().LinkDocument(ctx, bug, title, path); err != nil {
			return fmt.Errorf("failed to link document to bug: %w", err)
		}
		return printDocLinked(title, path, "bug", bug, 0)
	}
	if change != "" {
		if err := cli.GetChangeCardService().LinkDocument(ctx, change, title, path); err != nil {
			return fmt.Errorf("failed to link document to change-card: %w", err)
		}
		return printDocLinked(title, path, "change-card", change, 0)
	}
	doc, err := cli.GetTaskServiceWithDocs().LinkDocument(ctx, task, title, path)
	if err != nil {
		return fmt.Errorf("failed to link document to task: %w", err)
	}
	return printDocLinked(doc.Title, doc.FilePath, "task", task, doc.ID)
}

// runRelatedDocsAdd handles adding a document
func runRelatedDocsAdd(cmd *cobra.Command, args []string) error {
	title, path := args[0], args[1]
	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")
	bug, _ := cmd.Flags().GetString("bug")
	change, _ := cmd.Flags().GetString("change")

	count := boolInt(epic != "") + boolInt(feature != "") + boolInt(task != "") + boolInt(bug != "") + boolInt(change != "")
	if count != 1 {
		_ = cmd.Usage()
		return nil
	}

	return dispatchAddDoc(cmd.Context(), epic, feature, task, bug, change, title, path)
}

// boolInt converts a bool to 0 or 1.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// printDocLinked outputs the result of linking a document (JSON or human-readable).
func printDocLinked(title, path, entityType, parentKey string, docID int64) error {
	if cli.GlobalConfig.JSON {
		out := map[string]interface{}{
			"title": title, "path": path, "linked_to": entityType, "parent_key": parentKey,
		}
		if docID != 0 {
			out["document_id"] = docID
		}
		return cli.OutputJSON(out)
	}
	fmt.Printf("Document linked to %s %s\n", entityType, parentKey)
	return nil
}

// runRelatedDocsDelete handles deleting a document link
func runRelatedDocsDelete(cmd *cobra.Command, args []string) error {
	title := args[0]
	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")
	bug, _ := cmd.Flags().GetString("bug")
	change, _ := cmd.Flags().GetString("change")
	ctx := cmd.Context()

	if epic != "" {
		// Idempotent: ignore "not found" errors
		_ = cli.GetEpicService().UnlinkDocument(ctx, epic, title)
		return printDocUnlinked(title, "epic", epic)
	}
	if feature != "" {
		_ = cli.GetFeatureService().UnlinkDocument(ctx, feature, title)
		return printDocUnlinked(title, "feature", feature)
	}
	if task != "" {
		_ = cli.GetTaskServiceWithDocs().UnlinkDocument(ctx, task, title)
		return printDocUnlinked(title, "task", task)
	}
	if bug != "" {
		_ = cli.GetBugService().UnlinkDocument(ctx, bug, title)
		return printDocUnlinked(title, "bug", bug)
	}
	if change != "" {
		_ = cli.GetChangeCardService().UnlinkDocument(ctx, change, title)
		return printDocUnlinked(title, "change-card", change)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{"status": "unlinked", "title": title})
	}
	return nil
}

// printDocUnlinked outputs the result of unlinking a document.
func printDocUnlinked(title, entityType, parentKey string) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "unlinked", "title": title, "parent": entityType,
		})
	}
	fmt.Printf("Document unlinked from %s %s\n", entityType, parentKey)
	return nil
}

// dispatchListDocs fetches related documents for the first non-empty entity key.
func dispatchListDocs(ctx context.Context, epic, feature, task, bug, change string) ([]*models.Document, error) {
	if epic != "" {
		docs, err := cli.GetEpicService().ListRelatedDocumentsByKey(ctx, epic)
		if err != nil {
			return nil, fmt.Errorf("epic not found: %w", err)
		}
		return docs, nil
	}
	if feature != "" {
		docs, err := cli.GetFeatureService().ListRelatedDocumentsByKey(ctx, feature)
		if err != nil {
			return nil, fmt.Errorf("feature not found: %w", err)
		}
		return docs, nil
	}
	if bug != "" {
		docs, err := cli.GetBugService().ListRelatedDocumentsByKey(ctx, bug)
		if err != nil {
			return nil, fmt.Errorf("bug not found: %w", err)
		}
		return docs, nil
	}
	if change != "" {
		docs, err := cli.GetChangeCardService().ListRelatedDocumentsByKey(ctx, change)
		if err != nil {
			return nil, fmt.Errorf("change-card not found: %w", err)
		}
		return docs, nil
	}
	docs, err := cli.GetTaskServiceWithDocs().ListRelatedDocuments(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	return docs, nil
}

// runRelatedDocsListList handles listing documents
func runRelatedDocsListList(cmd *cobra.Command, args []string) error {
	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")
	bug, _ := cmd.Flags().GetString("bug")
	change, _ := cmd.Flags().GetString("change")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	useJSON := jsonOutput || cli.GlobalConfig.JSON

	if epic == "" && feature == "" && task == "" && bug == "" && change == "" {
		return fmt.Errorf("one of --epic, --feature, --task, --bug, or --change must be specified")
	}

	docs, err := dispatchListDocs(cmd.Context(), epic, feature, task, bug, change)
	if err != nil {
		return err
	}
	return printRelatedDocs(docs, useJSON)
}

// printRelatedDocs outputs a list of related documents.
func printRelatedDocs(docs []*models.Document, useJSON bool) error {
	if useJSON {
		return cli.OutputJSON(docs)
	}
	if len(docs) == 0 {
		fmt.Println("No documents found")
		return nil
	}
	fmt.Println("Related Documents:")
	for _, doc := range docs {
		fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
	}
	return nil
}

func init() {
	// Add subcommands
	relatedDocsCmd.AddCommand(relatedDocsAddCmd)
	relatedDocsCmd.AddCommand(relatedDocsDeleteCmd)
	relatedDocsCmd.AddCommand(relatedDocsListCmd)

	// Add flags for add command
	relatedDocsAddCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsAddCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsAddCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsAddCmd.Flags().String("bug", "", "Bug key (e.g., B001)")
	relatedDocsAddCmd.Flags().String("change", "", "Change-card key (e.g., CC-001)")

	// Add flags for delete command
	relatedDocsDeleteCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsDeleteCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsDeleteCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsDeleteCmd.Flags().String("bug", "", "Bug key (e.g., B001)")
	relatedDocsDeleteCmd.Flags().String("change", "", "Change-card key (e.g., CC-001)")

	// Add flags for list command
	relatedDocsListCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsListCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsListCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsListCmd.Flags().String("bug", "", "Bug key (e.g., B001)")
	relatedDocsListCmd.Flags().String("change", "", "Change-card key (e.g., CC-001)")
	relatedDocsListCmd.Flags().Bool("json", false, "Output in JSON format")
}
