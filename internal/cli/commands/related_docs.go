package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
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

type questionUnlinkService interface {
	UnlinkDocumentByKey(ctx context.Context, key, title string) error
}

type relatedDocsEntityKind string

const (
	relatedDocsEntityEpic     relatedDocsEntityKind = "epic"
	relatedDocsEntityFeature  relatedDocsEntityKind = "feature"
	relatedDocsEntityTask     relatedDocsEntityKind = "task"
	relatedDocsEntityBug      relatedDocsEntityKind = "bug"
	relatedDocsEntityChange   relatedDocsEntityKind = "change"
	relatedDocsEntityQuestion relatedDocsEntityKind = "question"
	relatedDocsEntityTechDebt relatedDocsEntityKind = "tech_debt"
)

func (kind relatedDocsEntityKind) cliLabel() string {
	switch kind {
	case relatedDocsEntityChange:
		return "change-card"
	case relatedDocsEntityTechDebt:
		return "tech-debt"
	default:
		return string(kind)
	}
}

type relatedDocsSelector struct {
	flag  string
	kind  relatedDocsEntityKind
	usage string
}

var techDebtDocService = func() relatedDocService { return cli.GetTechDebtService() }

// relatedDocsGetDB lets the delete handler return acquisition failures before
// any accessor that assumes the CLI database has already been initialized.
//
//nolint:gochecknoglobals // Test seam for the handler's pre-dispatch failure path.
var relatedDocsGetDB = cli.GetDB

// relatedDocsQuestionUnlinkService keeps the question handler's sentinel
// propagation directly testable at the registered Cobra boundary.
//
//nolint:gochecknoglobals // Test seam for the question RunE error contract.
var relatedDocsQuestionUnlinkService = func(ctx context.Context) questionUnlinkService {
	return cli.GetQuestionDocumentService(ctx)
}

func relatedDocsService(kind relatedDocsEntityKind) (relatedDocService, error) {
	switch kind {
	case relatedDocsEntityEpic:
		return cli.GetEpicService(), nil
	case relatedDocsEntityFeature:
		return cli.GetFeatureService(), nil
	case relatedDocsEntityBug:
		return cli.GetBugService(), nil
	case relatedDocsEntityChange:
		return cli.GetChangeCardService(), nil
	case relatedDocsEntityTechDebt:
		return techDebtDocService(), nil
	default:
		return nil, fmt.Errorf("unsupported related-docs entity type %q", kind.cliLabel())
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
	case relatedDocsEntityTask:
		doc, err = cli.GetTaskServiceWithDocs().LinkDocument(ctx, key, title, path)
	case relatedDocsEntityQuestion:
		doc, err = cli.GetQuestionDocumentService(ctx).LinkDocumentByKey(ctx, key, title, path)
	default:
		svc, err := relatedDocsService(kind)
		if err != nil {
			return err
		}
		if err := svc.LinkDocument(ctx, key, title, path); err != nil {
			return fmt.Errorf("link document to %s %s: %w", kind.cliLabel(), key, err)
		}
		return printDocLinked(title, path, kind, key, 0)
	}
	if err != nil {
		return fmt.Errorf("link document to %s %s: %w", kind.cliLabel(), key, err)
	}
	return printDocLinked(doc.Title, doc.FilePath, kind, key, doc.ID)
}

// printDocLinked outputs the result of linking a document (JSON or human-readable).
func printDocLinked(title, path string, kind relatedDocsEntityKind, parentKey string, docID int64) error {
	label := kind.cliLabel()
	if cli.GlobalConfig.JSON {
		out := map[string]interface{}{
			"title": title, "path": path, "linked_to": label, "parent_key": parentKey,
		}
		if docID != 0 {
			out["document_id"] = docID
		}
		return cli.OutputJSON(out)
	}
	fmt.Printf("Document linked to %s %s\n", label, parentKey)
	return nil
}

// runRelatedDocsDelete handles deleting a document link
func runRelatedDocsDelete(cmd *cobra.Command, args []string) error {
	kind, key, err := resolveRelatedDocsSelection(cmd, nil)
	if err != nil {
		return err
	}
	if _, err := relatedDocsGetDB(cmd.Context()); err != nil {
		return fmt.Errorf("exit code 2: failed to unlink document from %s %s: acquire database: %w", kind.cliLabel(), key, err)
	}
	title := args[0]
	ctx := cmd.Context()
	switch kind {
	case relatedDocsEntityTask:
		err = cli.GetTaskServiceWithDocs().UnlinkDocument(ctx, key, title)
	case relatedDocsEntityQuestion:
		err = relatedDocsQuestionUnlinkService(ctx).UnlinkDocumentByKey(ctx, key, title)
	default:
		svc, serviceErr := relatedDocsService(kind)
		if serviceErr != nil {
			return serviceErr
		}
		err = svc.UnlinkDocument(ctx, key, title)
	}
	if err != nil {
		return fmt.Errorf("exit code %d: failed to unlink document from %s %s: %w", relatedDocsErrorExitCode(err), kind.cliLabel(), key, err)
	}
	return printDocUnlinked(title, kind, key)
}

func relatedDocsErrorExitCode(err error) int {
	if repoerr.IsNotFound(err) {
		return 1
	}
	if errors.Is(err, services.ErrAmbiguousDocumentTitle) {
		return 3
	}
	return 2
}

// printDocUnlinked outputs the result of unlinking a document.
func printDocUnlinked(title string, kind relatedDocsEntityKind, parentKey string) error {
	label := kind.cliLabel()
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "unlinked", "title": title, "parent": label,
		})
	}
	fmt.Printf("Document unlinked from %s %s\n", label, parentKey)
	return nil
}

// The CLI uses hyphenated names in its output. Key inference uses Shark entity types.
var relatedDocsSelectors = []relatedDocsSelector{
	{flag: "epic", kind: relatedDocsEntityEpic, usage: "Epic key (e.g., E01)"},
	{flag: "feature", kind: relatedDocsEntityFeature, usage: "Feature key (e.g., E01-F01)"},
	{flag: "task", kind: relatedDocsEntityTask, usage: "Task key (e.g., T-E01-F01-001)"},
	{flag: "bug", kind: relatedDocsEntityBug, usage: "Bug key (e.g., B001)"},
	{flag: "change", kind: relatedDocsEntityChange, usage: "Change-card key (e.g., CC-001)"},
	{flag: "question", kind: relatedDocsEntityQuestion, usage: "Question key (e.g., Q001)"},
	{flag: "tech-debt", kind: relatedDocsEntityTechDebt, usage: "Tech-debt key (e.g., TD-001)"},
}

func resolveRelatedDocsSelection(cmd *cobra.Command, positional []string) (relatedDocsEntityKind, string, error) {
	if len(positional) > 1 {
		return "", "", fmt.Errorf("related-docs list accepts at most one positional entity key")
	}
	if len(positional) == 1 {
		if hasRelatedDocsSelector(cmd) {
			return "", "", fmt.Errorf("cannot combine positional entity key %q with an explicit entity-type flag", positional[0])
		}
		return resolveRelatedDocsPositionalSelection(positional[0])
	}
	return resolveRelatedDocsExplicitSelection(cmd)
}

func hasRelatedDocsSelector(cmd *cobra.Command) bool {
	for _, selector := range relatedDocsSelectors {
		if cmd.Flags().Changed(selector.flag) {
			return true
		}
	}
	return false
}

func resolveRelatedDocsPositionalSelection(rawKey string) (relatedDocsEntityKind, string, error) {
	key := NormalizeKey(rawKey)
	if strings.HasPrefix(key, "TD-") {
		if err := validateRelatedDocsTechDebtKey(key); err != nil {
			return "", "", err
		}
		return relatedDocsEntityTechDebt, key, nil
	}
	entityType := DetectEntityType(key)
	for _, selector := range relatedDocsSelectors {
		if string(selector.kind) == entityType {
			return selector.kind, key, nil
		}
	}
	return "", "", fmt.Errorf("cannot infer a supported entity type from key %q", key)
}

func resolveRelatedDocsExplicitSelection(cmd *cobra.Command) (relatedDocsEntityKind, string, error) {
	var kind relatedDocsEntityKind
	var key string
	for _, selector := range relatedDocsSelectors {
		if !cmd.Flags().Changed(selector.flag) {
			continue
		}
		if kind != "" {
			return "", "", fmt.Errorf("exactly one of the entity-type flags must be specified")
		}
		kind = selector.kind
		var err error
		key, err = cmd.Flags().GetString(selector.flag)
		if err != nil {
			return "", "", fmt.Errorf("read --%s selector: %w", selector.flag, err)
		}
	}
	if key == "" {
		return "", "", fmt.Errorf("exactly one of --epic, --feature, --task, --bug, --change, --question, or --tech-debt must specify an entity key")
	}
	if kind == relatedDocsEntityTechDebt {
		key = strings.TrimSpace(key)
		if err := validateRelatedDocsTechDebtKey(key); err != nil {
			return "", "", err
		}
	}
	return kind, key, nil
}

func validateRelatedDocsTechDebtKey(key string) error {
	parts := strings.SplitN(strings.TrimSpace(key), "-", 3)
	if len(parts) < 2 || models.ValidateTechDebtKey(strings.ToUpper(parts[0]+"-"+parts[1])) != nil {
		return invalidRelatedDocsTechDebtKey(key)
	}
	if len(parts) == 3 {
		slug := strings.ToLower(parts[2])
		if slug == "" || utils.GenerateSlug(slug) != slug {
			return invalidRelatedDocsTechDebtKey(key)
		}
	}
	return nil
}

func invalidRelatedDocsTechDebtKey(key string) error {
	return fmt.Errorf("invalid tech-debt key %q: expected TD-### or TD-###-<slug>", key)
}

func runRelatedDocsListList(cmd *cobra.Command, args []string) error {
	kind, key, err := resolveRelatedDocsSelection(cmd, args)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	var docs []*models.Document
	switch kind {
	case relatedDocsEntityTask:
		docs, err = cli.GetTaskServiceWithDocs().ListRelatedDocuments(ctx, key)
	case relatedDocsEntityQuestion:
		docs, err = cli.GetQuestionDocumentService(ctx).ListDocumentsByKey(ctx, key)
	default:
		svc, serviceErr := relatedDocsService(kind)
		if serviceErr != nil {
			return serviceErr
		}
		docs, err = svc.ListRelatedDocumentsByKey(ctx, key)
	}
	if err != nil {
		return fmt.Errorf("list documents for %s %s: %w", kind.cliLabel(), key, err)
	}
	// The related-docs list command always registers json as a bool flag.
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
