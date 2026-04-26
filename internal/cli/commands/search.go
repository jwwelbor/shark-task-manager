package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// validSearchTypes lists all accepted values for the --type flag.
var validSearchTypes = []string{"epic", "feature", "task", "bug", "change", "idea", "tech_debt"}

// searchCmd is the parent command for search operations.
// It supports two modes:
//   - Full-text query mode: `shark search "query" [--type=TYPE]`
//   - File search mode: `shark search --file="pattern" [--epic] [--feature] [--status]`
var searchCmd = &cobra.Command{
	Use:     "search [query]",
	Short:   "Search tasks and entities by query or file",
	GroupID: "inspect",
	Long: `Search for entities using full-text query or file-based search.

Full-text query mode (positional argument):
  shark search "login"                     Search all entity types
  shark search "login" --type=bug          Search only bugs
  shark search "dark mode" --type=change   Search only change-cards
  shark search "auth" --type=task          Search only tasks
  shark search "auth" --json               JSON output for all types

File search mode (--file flag):
  shark search --file="useTheme.ts"
  shark search --file="task_repository" --epic E10
  shark search --file="completion" --feature E10-F02
  shark search --file="models/task.go" --json

Valid --type values: epic, feature, task, bug, change, idea, tech_debt`,
	RunE: runSearch,
}

// runSearch dispatches to query mode or file mode based on arguments/flags.
func runSearch(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")

	if filePath != "" {
		// File search mode (existing behaviour)
		return runSearchFile(cmd, args)
	}

	// Full-text query mode
	return runSearchQuery(cmd, args)
}

// runSearchFile handles the legacy file-based search.
func runSearchFile(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("--file parameter is required when not providing a query argument")
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

// runSearchQuery handles the new full-text query search mode.
func runSearchQuery(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	query, err := parseSearchQuery(args)
	if err != nil {
		return err
	}

	entityTypeFlag, _ := cmd.Flags().GetString("type")
	if err := validateSearchType(entityTypeFlag); err != nil {
		return err
	}

	// Step 2: Call service.
	// E28-F05 REQ-F-011 / REQ-F-018: read the repeatable --tag flag.
	// nil when no --tag flags were supplied (AC-T2).
	var searchTags []string
	if rawTags, tagErr := cmd.Flags().GetStringSlice("tag"); tagErr == nil && len(rawTags) > 0 {
		searchTags = rawTags
	}
	results, err := cli.GetSearchService().SearchAll(cmd.Context(), query, entityTypeFlag, searchTags)
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "search", "")
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(results)
	}
	return printEntitySearchResults(results, query)
}

// parseSearchQuery extracts the search query from positional args.
func parseSearchQuery(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("search query required: provide a positional argument (e.g., shark search \"login\") or use --file for file search")
	}
	return strings.Join(args, " "), nil
}

// validateSearchType validates the --type flag value.
// An empty string is valid (means "all types").
func validateSearchType(entityType string) error {
	if entityType == "" {
		return nil
	}
	for _, valid := range validSearchTypes {
		if entityType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid type %q: valid types are %s", entityType, strings.Join(validSearchTypes, ", "))
}

// printEntitySearchResults prints human-readable cross-entity search results.
func printEntitySearchResults(results []*repository.EntitySearchResult, query string) error {
	if len(results) == 0 {
		fmt.Printf("No results found for: %s\n", query)
		return nil
	}
	fmt.Printf("Found %d result(s) for \"%s\":\n\n", len(results), query)
	for _, r := range results {
		if r.EntityType == "bug" && r.Severity != "" {
			fmt.Printf("[%s] %s: %s (%s, severity: %s)\n", r.EntityType, r.Key, r.Title, r.Status, r.Severity)
		} else {
			fmt.Printf("[%s] %s: %s (%s)\n", r.EntityType, r.Key, r.Title, r.Status)
		}
	}
	return nil
}

// printSearchResults prints human-readable file search results.
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

// printTaskFilesChanged prints the files changed for a task if available.
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

	// Flags for file search mode (legacy)
	searchCmd.Flags().String("file", "", "File name or path to search for (file search mode)")
	searchCmd.Flags().String("epic", "", "Filter by epic key (file search mode)")
	searchCmd.Flags().String("feature", "", "Filter by feature key (file search mode)")
	searchCmd.Flags().String("status", "", "Filter by task status (file search mode)")

	// Flags for full-text query mode
	searchCmd.Flags().String("type", "", "Filter by entity type: epic, feature, task, bug, change, idea")
	// E28-F05 REQ-F-011 / REQ-F-018: repeatable --tag flag with AND semantics.
	searchCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")
}
