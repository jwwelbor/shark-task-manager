package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// relatedDocsCmd represents the related-docs command group
var relatedDocsCmd = &cobra.Command{
	Use:     "related-docs",
	Short:   "Manage related documents",
	GroupID: "details",
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

// runRelatedDocsAdd handles adding a document
func runRelatedDocsAdd(cmd *cobra.Command, args []string) error {
	title := args[0]
	path := args[1]

	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")

	// Validate exactly one parent is specified
	count := 0
	if epic != "" {
		count++
	}
	if feature != "" {
		count++
	}
	if task != "" {
		count++
	}

	if count != 1 {
		_ = cmd.Usage()
		return nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Wrap database with repository.DB
	dbWrapper := repository.NewDB(database)
	docRepo := repository.NewDocumentRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Handle epic parent
	if epic != "" {
		e, err := epicRepo.GetByKey(ctx, epic)
		if err != nil {
			return fmt.Errorf("epic not found: %w", err)
		}

		doc, err := docRepo.CreateOrGet(ctx, title, path)
		if err != nil {
			return fmt.Errorf("failed to create or get document: %w", err)
		}

		if err := docRepo.LinkToEpic(ctx, e.ID, doc.ID); err != nil {
			return fmt.Errorf("failed to link document to epic: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"document_id": doc.ID,
				"title":       doc.Title,
				"path":        doc.FilePath,
				"linked_to":   "epic",
				"parent_key":  epic,
			})
		}

		fmt.Printf("Document linked to epic %s\n", epic)
		return nil
	}

	// Handle feature parent
	if feature != "" {
		f, err := featureRepo.GetByKey(ctx, feature)
		if err != nil {
			return fmt.Errorf("feature not found: %w", err)
		}

		doc, err := docRepo.CreateOrGet(ctx, title, path)
		if err != nil {
			return fmt.Errorf("failed to create or get document: %w", err)
		}

		if err := docRepo.LinkToFeature(ctx, f.ID, doc.ID); err != nil {
			return fmt.Errorf("failed to link document to feature: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"document_id": doc.ID,
				"title":       doc.Title,
				"path":        doc.FilePath,
				"linked_to":   "feature",
				"parent_key":  feature,
			})
		}

		fmt.Printf("Document linked to feature %s\n", feature)
		return nil
	}

	// Handle task parent
	if task != "" {
		t, err := taskRepo.GetByKey(ctx, task)
		if err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		doc, err := docRepo.CreateOrGet(ctx, title, path)
		if err != nil {
			return fmt.Errorf("failed to create or get document: %w", err)
		}

		if err := docRepo.LinkToTask(ctx, t.ID, doc.ID); err != nil {
			return fmt.Errorf("failed to link document to task: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"document_id": doc.ID,
				"title":       doc.Title,
				"path":        doc.FilePath,
				"linked_to":   "task",
				"parent_key":  task,
			})
		}

		fmt.Printf("Document linked to task %s\n", task)
		return nil
	}

	return nil
}

// runRelatedDocsDelete handles deleting a document link
func runRelatedDocsDelete(cmd *cobra.Command, args []string) error {
	title := args[0]

	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Wrap database with repository.DB
	dbWrapper := repository.NewDB(database)
	docRepo := repository.NewDocumentRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Handle epic parent
	if epic != "" {
		e, err := epicRepo.GetByKey(ctx, epic)
		if err != nil {
			// Epic doesn't exist, but delete is idempotent - succeed anyway
			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(map[string]interface{}{
					"status": "unlinked",
					"parent": "epic",
				})
			}
			return nil
		}

		// Look up the document by title
		doc, err := docRepo.GetByTitle(ctx, title)
		if err == nil {
			// Document exists, actually perform the unlinking
			if err := docRepo.UnlinkFromEpic(ctx, e.ID, doc.ID); err != nil {
				return fmt.Errorf("failed to unlink document: %w", err)
			}
		}
		// If document doesn't exist, delete is idempotent - succeed anyway

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "unlinked",
				"title":  title,
				"parent": "epic",
			})
		}

		fmt.Printf("Document unlinked from epic %s\n", epic)
		return nil
	}

	// Handle feature parent
	if feature != "" {
		f, err := featureRepo.GetByKey(ctx, feature)
		if err != nil {
			// Feature doesn't exist, but delete is idempotent - succeed anyway
			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(map[string]interface{}{
					"status": "unlinked",
					"parent": "feature",
				})
			}
			return nil
		}

		// Look up the document by title
		doc, err := docRepo.GetByTitle(ctx, title)
		if err == nil {
			// Document exists, actually perform the unlinking
			if err := docRepo.UnlinkFromFeature(ctx, f.ID, doc.ID); err != nil {
				return fmt.Errorf("failed to unlink document: %w", err)
			}
		}
		// If document doesn't exist, delete is idempotent - succeed anyway

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "unlinked",
				"title":  title,
				"parent": "feature",
			})
		}

		fmt.Printf("Document unlinked from feature %s\n", feature)
		return nil
	}

	// Handle task parent
	if task != "" {
		t, err := taskRepo.GetByKey(ctx, task)
		if err != nil {
			// Task doesn't exist, but delete is idempotent - succeed anyway
			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(map[string]interface{}{
					"status": "unlinked",
					"parent": "task",
				})
			}
			return nil
		}

		// Look up the document by title
		doc, err := docRepo.GetByTitle(ctx, title)
		if err == nil {
			// Document exists, actually perform the unlinking
			if err := docRepo.UnlinkFromTask(ctx, t.ID, doc.ID); err != nil {
				return fmt.Errorf("failed to unlink document: %w", err)
			}
		}
		// If document doesn't exist, delete is idempotent - succeed anyway

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "unlinked",
				"title":  title,
				"parent": "task",
			})
		}

		fmt.Printf("Document unlinked from task %s\n", task)
		return nil
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "unlinked",
			"title":  title,
		})
	}

	return nil
}

// runRelatedDocsListList handles listing documents
func runRelatedDocsListList(cmd *cobra.Command, args []string) error {
	epic, _ := cmd.Flags().GetString("epic")
	feature, _ := cmd.Flags().GetString("feature")
	task, _ := cmd.Flags().GetString("task")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Wrap database with repository.DB
	dbWrapper := repository.NewDB(database)
	docRepo := repository.NewDocumentRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	var docs []*models.Document

	// Handle epic parent
	if epic != "" {
		e, err := epicRepo.GetByKey(ctx, epic)
		if err != nil {
			return fmt.Errorf("epic not found: %w", err)
		}

		docs, err = docRepo.ListForEpic(ctx, e.ID)
		if err != nil {
			return fmt.Errorf("failed to list documents for epic: %w", err)
		}
	} else if feature != "" {
		f, err := featureRepo.GetByKey(ctx, feature)
		if err != nil {
			return fmt.Errorf("feature not found: %w", err)
		}

		docs, err = docRepo.ListForFeature(ctx, f.ID)
		if err != nil {
			return fmt.Errorf("failed to list documents for feature: %w", err)
		}
	} else if task != "" {
		t, err := taskRepo.GetByKey(ctx, task)
		if err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		docs, err = docRepo.ListForTask(ctx, t.ID)
		if err != nil {
			return fmt.Errorf("failed to list documents for task: %w", err)
		}
	} else {
		return fmt.Errorf("one of --epic, --feature, or --task must be specified")
	}

	// Output results
	if jsonOutput || cli.GlobalConfig.JSON {
		return cli.OutputJSON(docs)
	}

	// Human-readable output
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
	// Register related-docs command with root
	cli.RootCmd.AddCommand(relatedDocsCmd)

	// Add subcommands
	relatedDocsCmd.AddCommand(relatedDocsAddCmd)
	relatedDocsCmd.AddCommand(relatedDocsDeleteCmd)
	relatedDocsCmd.AddCommand(relatedDocsListCmd)

	// Add flags for add command
	relatedDocsAddCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsAddCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsAddCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")

	// Add flags for delete command
	relatedDocsDeleteCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsDeleteCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsDeleteCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")

	// Add flags for list command
	relatedDocsListCmd.Flags().String("epic", "", "Epic key (e.g., E01)")
	relatedDocsListCmd.Flags().String("feature", "", "Feature key (e.g., E01-F01)")
	relatedDocsListCmd.Flags().String("task", "", "Task key (e.g., T-E01-F01-001)")
	relatedDocsListCmd.Flags().Bool("json", false, "Output in JSON format")
}
