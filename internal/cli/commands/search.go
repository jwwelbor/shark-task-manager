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

// searchCmd is the parent command for search operations
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search tasks by various criteria",
	Long: `Search for tasks using completion metadata like files changed.

Supports partial filename matching. Results are ordered by completion date (most recent first).

Examples:
  shark search --file="useTheme.ts"
  shark search --file="task_repository" --epic E10
  shark search --file="completion" --feature E10-F02
  shark search --file="models/task.go" --json`,
	RunE: runSearchFile,
}

// runSearchFile handles the file search command
func runSearchFile(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get file parameter
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("--file parameter is required")
	}

	// Get optional filters
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	status, _ := cmd.Flags().GetString("status")

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

	// Create repository
	repoDb := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Search by file changed
	tasks, err := taskRepo.FindByFileChanged(ctx, filePath)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Filter by epic if specified
	if epicKey != "" {
		filtered := []*models.Task{}
		for _, task := range tasks {
			// Need to check if task belongs to epic
			// Get feature to check epic
			featureRepo := repository.NewFeatureRepository(repoDb)
			feature, err := featureRepo.GetByID(ctx, task.FeatureID)
			if err != nil {
				continue
			}

			epicRepo := repository.NewEpicRepository(repoDb)
			epic, err := epicRepo.GetByID(ctx, feature.EpicID)
			if err != nil {
				continue
			}

			if epic.Key == epicKey {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}

	// Filter by feature if specified
	if featureKey != "" {
		filtered := []*models.Task{}
		featureRepo := repository.NewFeatureRepository(repoDb)
		feature, err := featureRepo.GetByKey(ctx, featureKey)
		if err == nil {
			for _, task := range tasks {
				if task.FeatureID == feature.ID {
					filtered = append(filtered, task)
				}
			}
			tasks = filtered
		}
	}

	// Filter by status if specified
	if status != "" {
		filtered := []*models.Task{}
		for _, task := range tasks {
			if string(task.Status) == status {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}

	// Output results
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(tasks)
	}

	// Human-readable output
	if len(tasks) == 0 {
		fmt.Printf("No tasks found matching file: %s\n", filePath)
		return nil
	}

	fmt.Printf("Found %d task(s) matching file \"%s\":\n\n", len(tasks), filePath)

	for _, task := range tasks {
		fmt.Printf("%s: %s (%s)\n", task.Key, task.Title, task.Status)

		// Display completion date if available
		if task.CompletedAt.Valid {
			fmt.Printf("  Completed: %s\n", task.CompletedAt.Time.Format("2006-01-02 15:04:05"))
		}

		// Display files changed if available
		if task.FilesChanged != nil && *task.FilesChanged != "" {
			// Parse and display files
			var files []string
			metadata := models.NewCompletionMetadata()
			if err := metadata.FromJSON(*task.FilesChanged); err == nil {
				files = metadata.FilesChanged
			}

			if len(files) > 0 {
				fmt.Print("  Files: ")
				for i, file := range files {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Print(file)
				}
				fmt.Println()
			}
		}

		fmt.Println()
	}

	return nil
}

func init() {
	// Add search command to root
	cli.RootCmd.AddCommand(searchCmd)

	// Add flags for file search
	searchCmd.Flags().String("file", "", "File name or path to search for (required)")
	searchCmd.Flags().String("epic", "", "Filter by epic key")
	searchCmd.Flags().String("feature", "", "Filter by feature key")
	searchCmd.Flags().String("status", "", "Filter by task status")

	// Mark file as required
	_ = searchCmd.MarkFlagRequired("file")
}
