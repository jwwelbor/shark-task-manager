package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	customfilepath "github.com/jwwelbor/shark-task-manager/internal/filepath"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate task file paths and consistency",
	Long: `Validate that all task file paths in the database are correct and that files exist.

This command checks:
- Database file_path values match the expected pattern
- Task files exist at their recorded paths
- File paths are consistent with task metadata

Examples:
  pm validate              Validate all tasks
  pm validate --json       Output results as JSON
  pm validate --verbose    Show detailed validation information`,
	RunE: runValidate,
}

// ValidationResult holds the validation results
type ValidationResult struct {
	TotalTasks    int                 `json:"total_tasks"`
	ValidTasks    int                 `json:"valid_tasks"`
	InvalidTasks  int                 `json:"invalid_tasks"`
	MissingFiles  []ValidationIssue   `json:"missing_files,omitempty"`
	InvalidPaths  []ValidationIssue   `json:"invalid_paths,omitempty"`
	DurationMs    int64               `json:"duration_ms"`
}

// ValidationIssue describes a validation problem
type ValidationIssue struct {
	TaskKey      string `json:"task_key"`
	ExpectedPath string `json:"expected_path,omitempty"`
	ActualPath   string `json:"actual_path,omitempty"`
	Issue        string `json:"issue"`
}

var (
	validateJSON    bool
	validateVerbose bool
)

func init() {
	// Register validate command with root
	cli.RootCmd.AddCommand(validateCmd)

	// Add flags
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output results as JSON")
	validateCmd.Flags().BoolVar(&validateVerbose, "verbose", false, "Show detailed validation information")
}

func runValidate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Get database path
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Create repository
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get all tasks
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasks, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve tasks: %w", err)
	}

	// Validate each task
	result := &ValidationResult{
		TotalTasks:   len(tasks),
		MissingFiles: []ValidationIssue{},
		InvalidPaths: []ValidationIssue{},
	}

	for _, task := range tasks {
		if err := validateTask(*task, result); err != nil {
			if validateVerbose {
				cli.Warning(fmt.Sprintf("Error validating task %s: %v", task.Key, err))
			}
		}
	}

	result.ValidTasks = result.TotalTasks - len(result.MissingFiles) - len(result.InvalidPaths)
	result.InvalidTasks = len(result.MissingFiles) + len(result.InvalidPaths)
	result.DurationMs = time.Since(startTime).Milliseconds()

	// Output results
	if validateJSON {
		return outputJSON(result)
	}

	return outputHuman(result)
}

func validateTask(task models.Task, result *ValidationResult) error {
	// If task has no file_path, skip (might be database-only tasks)
	if task.FilePath == nil || *task.FilePath == "" {
		if validateVerbose {
			cli.Info("Task %s has no file_path (skipping)", task.Key)
		}
		return nil
	}

	actualPath := *task.FilePath

	// Validate path pattern against task key
	if err := customfilepath.ValidateFilePath(actualPath, task.Key); err != nil {
		result.InvalidPaths = append(result.InvalidPaths, ValidationIssue{
			TaskKey:    task.Key,
			ActualPath: actualPath,
			Issue:      fmt.Sprintf("Path doesn't match expected pattern: %v", err),
		})
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, ValidationIssue{
			TaskKey:      task.Key,
			ExpectedPath: actualPath,
			Issue:        "File does not exist",
		})
		return nil
	}

	if validateVerbose {
		cli.Success(fmt.Sprintf("✓ Task %s: path valid, file exists", task.Key))
	}

	return nil
}

func outputJSON(result *ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputHuman(result *ValidationResult) error {
	fmt.Println()
	cli.Title("Validation Results")
	fmt.Println()

	// Summary
	fmt.Printf("Total tasks:   %d\n", result.TotalTasks)
	fmt.Printf("Valid tasks:   %s%d%s\n", cli.ColorGreen, result.ValidTasks, cli.ColorReset)
	fmt.Printf("Invalid tasks: %s%d%s\n", cli.ColorRed, result.InvalidTasks, cli.ColorReset)
	fmt.Printf("Duration:      %dms\n", result.DurationMs)
	fmt.Println()

	// Missing files
	if len(result.MissingFiles) > 0 {
		cli.Error(fmt.Sprintf("Missing Files (%d)", len(result.MissingFiles)))
		fmt.Println()
		for _, issue := range result.MissingFiles {
			fmt.Printf("  %s%-20s%s %s\n",
				cli.ColorYellow, issue.TaskKey, cli.ColorReset,
				issue.ExpectedPath)
			fmt.Printf("    → %s\n", issue.Issue)
		}
		fmt.Println()
	}

	// Invalid paths
	if len(result.InvalidPaths) > 0 {
		cli.Error(fmt.Sprintf("Invalid Paths (%d)", len(result.InvalidPaths)))
		fmt.Println()
		for _, issue := range result.InvalidPaths {
			fmt.Printf("  %s%-20s%s %s\n",
				cli.ColorYellow, issue.TaskKey, cli.ColorReset,
				issue.ActualPath)
			fmt.Printf("    → %s\n", issue.Issue)
		}
		fmt.Println()
	}

	// Exit code
	if result.InvalidTasks > 0 {
		return fmt.Errorf("validation found %d issue(s)", result.InvalidTasks)
	}

	cli.Success("All tasks valid!")
	return nil
}
