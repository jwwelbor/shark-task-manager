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
	Long: `Manage related documents linked to entities, including tech-debt items.

Examples:
  shark related-docs add "Design Doc" docs/design.md --epic=E01
  shark related-docs list --epic=E01
  shark related-docs delete "Design Doc" --epic=E01
  shark related-docs add "Research report" docs/report.md --tech-debt=TD-001`,
}

// relatedDocsAddCmd adds a document to a parent entity
var relatedDocsAddCmd = &cobra.Command{
	Use:   "add <title> <path>",
	Short: "Add a related document",
	Long: `Add a related document to an epic, feature, task, bug, change-card, question, or tech-debt item.

The document is created or retrieved if it already exists with the same title and path.
The document is then linked to exactly one parent entity.

Examples:
  shark related-docs add "OAuth Specification" docs/oauth.md --epic=E01
  shark related-docs add "Implementation Notes" docs/notes.md --feature=E01-F01
  shark related-docs add "Task Details" docs/details.md --task=T-E01-F01-001
  shark related-docs add "Research report" docs/report.md --tech-debt=TD-001`,
	Args: cobra.ExactArgs(2),
	RunE: runRelatedDocsAdd,
}

// relatedDocsDeleteCmd removes a document from a parent entity
var relatedDocsDeleteCmd = &cobra.Command{
	Use:   "delete <title>",
	Short: "Delete a related document link",
	Long: `Remove a document link from an epic, feature, task, bug, change-card, question, or tech-debt item.

The document itself is not deleted from the database, only the link is removed.
Delete succeeds when the selected entity has no link with this title. If multiple
linked documents have this title, the command returns an ambiguity error and keeps all links.

Examples:
  shark related-docs delete "OAuth Specification" --epic=E01
  shark related-docs delete "Implementation Notes" --feature=E01-F01
  shark related-docs delete "Task Details" --task=T-E01-F01-001
  shark related-docs delete "Research report" --tech-debt=TD-001`,
	Args: cobra.ExactArgs(1),
	RunE: runRelatedDocsDelete,
}

// relatedDocsListCmd lists documents for a parent entity
var relatedDocsListCmd = &cobra.Command{
	Use:   "list [<entity-key>]",
	Short: "List related documents",
	Long: `List all documents linked to an epic, feature, task, bug, change-card, question, or tech-debt item.

Pass an entity key positionally to infer its type, or use exactly one explicit
entity-type flag.

Examples:
  shark related-docs list E01
  shark related-docs list B001 --json
  shark related-docs list --epic=E01
  shark related-docs list --feature=E01-F01 --json
  shark related-docs list --task=T-E01-F01-001
  shark related-docs list TD-001 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRelatedDocsListList,
}

// relatedDocService is the document API shared by epic, feature, bug, change and tech-debt.
type relatedDocService interface {
	LinkDocument(ctx context.Context, key, title, path string) error
	UnlinkDocument(ctx context.Context, key, title string) error
	ListRelatedDocumentsByKey(ctx context.Context, key string) ([]*models.Document, error)
}

var techDebtDocService = func() relatedDocService { return cli.GetTechDebtService() }

func relatedDocsService(kind string) (relatedDocService, error) {
	switch kind {
	case "epic":
		return cli.GetEpicService(), nil
	case "feature":
		return cli.GetFeatureService(), nil
	case "bug":
		return cli.GetBugService(), nil
	case "change-card":
		return cli.GetChangeCardService(), nil
	case "tech-debt":
		return techDebtDocService(), nil
	default:
		return nil, fmt.Errorf("unsupported related-docs entity type %q", kind)
	}
}

func runRelatedDocsAdd(cmd *cobra.Command, args []string) error {
	kind, key, err := resolveRelatedDocsSelection(cmd, nil)
	if err != nil {
		return err
	}
	title, path := args[0], args[1]
	ctx := cmd.Context()
	var doc *models.Document
	switch kind {
	case "task":
		doc, err = cli.GetTaskServiceWithDocs().LinkDocument(ctx, key, title, path)
	case "question":
		doc, err = cli.GetQuestionDocumentService(ctx).LinkDocumentByKey(ctx, key, title, path)
	default:
		svc, err := relatedDocsService(kind)
		if err != nil {
			return err
		}
		if err := svc.LinkDocument(ctx, key, title, path); err != nil {
			return fmt.Errorf("link document to %s %s: %w", kind, key, err)
		}
		return printDocLinked(title, path, kind, key, 0)
	}
	if err != nil {
		return fmt.Errorf("link document to %s %s: %w", kind, key, err)
	}
	return printDocLinked(doc.Title, doc.FilePath, kind, key, doc.ID)
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
	kind, key, err := resolveRelatedDocsSelection(cmd, nil)
	if err != nil {
		return err
	}
	title := args[0]
	ctx := cmd.Context()
	switch kind {
	case "task":
		err = cli.GetTaskServiceWithDocs().UnlinkDocument(ctx, key, title)
	case "question":
		err = cli.GetQuestionDocumentService(ctx).UnlinkDocumentByKey(ctx, key, title)
	default:
		svc, serviceErr := relatedDocsService(kind)
		if serviceErr != nil {
			return serviceErr
		}
		err = svc.UnlinkDocument(ctx, key, title)
	}
	if err != nil {
		return fmt.Errorf("unlink document from %s %s: %w", kind, key, err)
	}
	return printDocUnlinked(title, kind, key)
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

// The CLI uses hyphenated names in its output. Key inference uses Shark entity types.
var relatedDocsSelectors = []struct {
	flag, entityType, label, usage string
}{
	{"epic", "epic", "epic", "Epic key (e.g., E01)"},
	{"feature", "feature", "feature", "Feature key (e.g., E01-F01)"},
	{"task", "task", "task", "Task key (e.g., T-E01-F01-001)"},
	{"bug", "bug", "bug", "Bug key (e.g., B001)"},
	{"change", "change", "change-card", "Change-card key (e.g., CC-001)"},
	{"question", "question", "question", "Question key (e.g., Q001)"},
	{"tech-debt", "tech_debt", "tech-debt", "Tech-debt key (e.g., TD-001)"},
}

func resolveRelatedDocsSelection(cmd *cobra.Command, positional []string) (string, string, error) {
	if len(positional) > 1 {
		return "", "", fmt.Errorf("related-docs list accepts at most one positional entity key")
	}

	var kind, key string
	for _, selector := range relatedDocsSelectors {
		if !cmd.Flags().Changed(selector.flag) {
			continue
		}
		if len(positional) == 1 {
			return "", "", fmt.Errorf("cannot combine positional entity key %q with an explicit entity-type flag", positional[0])
		}
		if kind != "" {
			return "", "", fmt.Errorf("exactly one of the entity-type flags must be specified")
		}
		kind = selector.label
		// Changed confirms the flag exists, and init registers every selector as a string.
		key, _ = cmd.Flags().GetString(selector.flag)
	}
	if len(positional) == 1 {
		key = NormalizeKey(positional[0])
		entityType := DetectEntityType(key)
		for _, selector := range relatedDocsSelectors {
			if selector.entityType == entityType {
				return selector.label, key, nil
			}
		}
		return "", "", fmt.Errorf("cannot infer a supported entity type from key %q", key)
	}
	if key == "" {
		return "", "", fmt.Errorf("exactly one of --epic, --feature, --task, --bug, --change, --question, or --tech-debt must specify an entity key")
	}
	return kind, key, nil
}

func runRelatedDocsListList(cmd *cobra.Command, args []string) error {
	kind, key, err := resolveRelatedDocsSelection(cmd, args)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	var docs []*models.Document
	switch kind {
	case "task":
		docs, err = cli.GetTaskServiceWithDocs().ListRelatedDocuments(ctx, key)
	case "question":
		docs, err = cli.GetQuestionDocumentService(ctx).ListDocumentsByKey(ctx, key)
	default:
		svc, serviceErr := relatedDocsService(kind)
		if serviceErr != nil {
			return serviceErr
		}
		docs, err = svc.ListRelatedDocumentsByKey(ctx, key)
	}
	if err != nil {
		return fmt.Errorf("list documents for %s %s: %w", kind, key, err)
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")
	return printRelatedDocs(docs, jsonOutput || cli.GlobalConfig.JSON)
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

	for _, cmd := range []*cobra.Command{relatedDocsAddCmd, relatedDocsDeleteCmd, relatedDocsListCmd} {
		for _, selector := range relatedDocsSelectors {
			cmd.Flags().String(selector.flag, "", selector.usage)
		}
	}
	relatedDocsListCmd.Flags().Bool("json", false, "Output in JSON format")
}
