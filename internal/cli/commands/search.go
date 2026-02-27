package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// searchCmd is the parent command for search operations
var searchCmd = &cobra.Command{
	Use:     "search",
	Short:   "Search tasks by various criteria",
	GroupID: "inspect",
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
	// Step 1: Parse arguments
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("--file parameter is required")
	}

	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	status, _ := cmd.Flags().GetString("status")
	filters := services.TaskFilters{EpicKey: epicKey, FeatureKey: featureKey, Status: status}

	// Step 2: Call service
	tasks, err := cli.GetTaskService().SearchByFile(cmd.Context(), filePath, filters)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(tasks)
	}
	return printSearchResults(tasks, filePath)
}

// printSearchResults prints human-readable search results.
func printSearchResults(tasks []*models.Task, filePath string) error {
	if len(tasks) == 0 {
		fmt.Printf("No tasks found matching file: %s\n", filePath)
		return nil
	}
	fmt.Printf("Found %d task(s) matching file \"%s\":\n\n", len(tasks), filePath)
	for _, task := range tasks {
		fmt.Printf("%s: %s (%s)\n", task.Key, task.Title, task.Status)
		if task.CompletedAt.Valid {
			fmt.Printf("  Completed: %s\n", task.CompletedAt.Time.Format("2006-01-02 15:04:05"))
		}
		printTaskFilesChanged(task)
		fmt.Println()
	}
	return nil
}

// printTaskFilesChanged prints the files changed for a task if available
func printTaskFilesChanged(task *models.Task) {
	if task.FilesChanged == nil || *task.FilesChanged == "" {
		return
	}

	metadata := models.NewCompletionMetadata()
	if err := metadata.FromJSON(*task.FilesChanged); err != nil {
		return
	}

	if len(metadata.FilesChanged) > 0 {
		fmt.Print("  Files: ")
		for i, file := range metadata.FilesChanged {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(file)
		}
		fmt.Println()
	}
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
