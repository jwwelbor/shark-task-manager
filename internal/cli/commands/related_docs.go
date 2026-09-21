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
	Use:   "list [<entity-key>]",
	Short: "List related documents",
	Long: `List all documents linked to an epic, feature, task, bug, change-card, or question.

Pass an entity key positionally to infer its type, or use exactly one explicit
entity-type flag.

Examples:
  shark related-docs list E01
  shark related-docs list B001 --json
  shark related-docs list --epic=E01
  shark related-docs list --feature=E01-F01 --json
  shark related-docs list --task=T-E01-F01-001`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRelatedDocsListList,
}

// dispatchAddDoc links a document to an epic, feature, task, bug, or change-card based on which key is set.
func dispatchAddDoc(ctx context.Context, epic, feature, task, bug, change, question, title, path string) error {
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
	if question != "" {
		doc, err := cli.GetQuestionDocumentService(ctx).LinkDocumentByKey(ctx, question, title, path)
		if err != nil {
			return fmt.Errorf("failed to link document to question: %w", err)
		}
		return printDocLinked(doc.Title, doc.FilePath, "question", question, doc.ID)
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
	question, _ := cmd.Flags().GetString("question")

	count := boolInt(epic != "") + boolInt(feature != "") + boolInt(task != "") + boolInt(bug != "") + boolInt(change != "") + boolInt(question != "")
	if count != 1 {
		_ = cmd.Usage()
		return nil
	}

	return dispatchAddDoc(cmd.Context(), epic, feature, task, bug, change, question, title, path)
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
	question, _ := cmd.Flags().GetString("question")
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
	if question != "" {
		if err := cli.GetQuestionDocumentService(ctx).UnlinkDocumentByKey(ctx, question, title); err != nil {
			return fmt.Errorf("unlink document from question %s: %w", question, err)
		}
		return printDocUnlinked(title, "question", question)
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

// dispatchListDocs fetches related documents for the selected entity key.
func dispatchListDocs(ctx context.Context, entityType, key string) ([]*models.Document, error) {
	switch entityType {
	case "epic":
		docs, err := cli.GetEpicService().ListRelatedDocumentsByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("epic not found: %w", err)
		}
		return docs, nil
	case "feature":
		docs, err := cli.GetFeatureService().ListRelatedDocumentsByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("feature not found: %w", err)
		}
		return docs, nil
	case "bug":
		docs, err := cli.GetBugService().ListRelatedDocumentsByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("bug not found: %w", err)
		}
		return docs, nil
	case "change":
		docs, err := cli.GetChangeCardService().ListRelatedDocumentsByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("change-card not found: %w", err)
		}
		return docs, nil
	case "question":
		docs, err := cli.GetQuestionDocumentService(ctx).ListDocumentsByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("question not found: %w", err)
		}
		return docs, nil
	case "task":
		docs, err := cli.GetTaskServiceWithDocs().ListRelatedDocuments(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		return docs, nil
	default:
		return nil, fmt.Errorf("unsupported related-docs entity type %q", entityType)
	}
}

// resolveRelatedDocsListSelection resolves either a positional entity key or
// one explicit entity-type selector into the values consumed by dispatchListDocs.
func resolveRelatedDocsListSelection(args []string, epic, feature, task, bug, change, question string) (string, string, error) {
	if len(args) > 1 {
		return "", "", fmt.Errorf("related-docs list accepts at most one positional entity key")
	}
	if len(args) == 1 {
		if hasExplicitRelatedDocsSelector(epic, feature, task, bug, change, question) {
			return "", "", fmt.Errorf("cannot combine positional entity key %q with an explicit entity-type flag", args[0])
		}
		return resolvePositionalRelatedDocsSelection(args[0])
	}
	return resolveExplicitRelatedDocsSelection(epic, feature, task, bug, change, question)
}

func hasExplicitRelatedDocsSelector(keys ...string) bool {
	selected := 0
	for _, key := range keys {
		if key != "" {
			selected++
		}
	}
	return selected > 0
}

func resolvePositionalRelatedDocsSelection(key string) (string, string, error) {
	entityType := DetectEntityType(key)
	switch entityType {
	case "epic", "feature", "task", "bug", "change", "question":
		return entityType, key, nil
	default:
		return "", "", fmt.Errorf("cannot infer a supported entity type from key %q", key)
	}
}

func resolveExplicitRelatedDocsSelection(epic, feature, task, bug, change, question string) (string, string, error) {
	selectors := []struct {
		entityType string
		key        string
	}{
		{entityType: "epic", key: epic},
		{entityType: "feature", key: feature},
		{entityType: "task", key: task},
		{entityType: "bug", key: bug},
		{entityType: "change", key: change},
		{entityType: "question", key: question},
	}
	selected := 0
	var selectedType, selectedKey string
	for _, selector := range selectors {
		if selector.key == "" {
			continue
		}
		selected++
		selectedType, selectedKey = selector.entityType, selector.key
	}
	if selected == 0 {
		return "", "", fmt.Errorf("one of --epic, --feature, --task, --bug, --change, or --question must be specified")
	}
	if selected > 1 {
		return "", "", fmt.Errorf("exactly one of --epic, --feature, --task, --bug, --change, or --question must be specified")
	}
	return selectedType, selectedKey, nil
}

// runRelatedDocsListList handles listing documents
func runRelatedDocsListList(cmd *cobra.Command, args []string) error {
	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")
	bug, _ := cmd.Flags().GetString("bug")
	change, _ := cmd.Flags().GetString("change")
	question, _ := cmd.Flags().GetString("question")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	useJSON := jsonOutput || cli.GlobalConfig.JSON

	entityType, key, err := resolveRelatedDocsListSelection(args, epic, feature, task, bug, change, question)
	if err != nil {
		return err
	}

	docs, err := dispatchListDocs(cmd.Context(), entityType, key)
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
	relatedDocsAddCmd.Flags().String("question", "", "Question key (e.g., Q001)")

	// Add flags for delete command
	relatedDocsDeleteCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsDeleteCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsDeleteCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsDeleteCmd.Flags().String("bug", "", "Bug key (e.g., B001)")
	relatedDocsDeleteCmd.Flags().String("change", "", "Change-card key (e.g., CC-001)")
	relatedDocsDeleteCmd.Flags().String("question", "", "Question key (e.g., Q001)")

	// Add flags for list command
	relatedDocsListCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsListCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsListCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsListCmd.Flags().String("bug", "", "Bug key (e.g., B001)")
	relatedDocsListCmd.Flags().String("change", "", "Change-card key (e.g., CC-001)")
	relatedDocsListCmd.Flags().String("question", "", "Question key (e.g., Q001)")
	relatedDocsListCmd.Flags().Bool("json", false, "Output in JSON format")
}
