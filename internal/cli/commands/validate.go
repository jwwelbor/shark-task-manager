package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "Validate database integrity",
	GroupID: "setup",
	Long: `Validate database integrity by checking file paths and relationships.

This command checks:
- Task file paths exist on filesystem
- Features reference existing epics (relationship integrity)
- Tasks reference existing features (relationship integrity)
- Detects orphaned records (records with missing parents)

Examples:
  shark validate              Validate database integrity
  shark validate --json       Output results as JSON
  shark validate --verbose    Show detailed validation information`,
	RunE: runValidate,
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

	// Create repositories
	dbWrapper := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Create repository adapter for validation
	repoAdapter := validation.NewRepositoryAdapter(epicRepo, featureRepo, taskRepo)

	// Create validator
	validator := validation.NewValidator(repoAdapter)

	// Run validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if validateVerbose {
		cli.Info("Starting validation...")
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output results
	if validateJSON {
		if err := result.FormatJSON(os.Stdout); err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
	} else {
		if err := result.FormatHuman(os.Stdout); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	}

	// Exit with appropriate code
	if !result.IsSuccess() {
		return fmt.Errorf("validation found %d issue(s)", result.Summary.TotalIssues)
	}

	return nil
}
