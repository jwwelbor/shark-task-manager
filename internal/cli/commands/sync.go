package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncFolder        string
	syncDryRun        bool
	syncStrategy      string
	syncCreateMissing bool
	syncCleanup       bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize task files with database",
	Long: `Synchronize task markdown files with the database by scanning feature folders,
parsing frontmatter, detecting conflicts, and applying resolution strategies.

Status is managed exclusively in the database and is NOT synced from files.`,
	Example: `  # Sync all feature folders
  pm sync

  # Sync specific folder
  pm sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation

  # Preview changes without applying (dry-run)
  pm sync --dry-run

  # Use database-wins strategy for conflicts
  pm sync --strategy=database-wins

  # Auto-create missing epics/features
  pm sync --create-missing

  # Delete orphaned database tasks (files deleted)
  pm sync --cleanup`,
	RunE: runSync,
}

func init() {
	cli.RootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVar(&syncFolder, "folder", "",
		"Sync specific folder only (default: docs/plan)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false,
		"Preview changes without applying them")
	syncCmd.Flags().StringVar(&syncStrategy, "strategy", "file-wins",
		"Conflict resolution strategy: file-wins, database-wins, newer-wins")
	syncCmd.Flags().BoolVar(&syncCreateMissing, "create-missing", false,
		"Auto-create missing epics/features")
	syncCmd.Flags().BoolVar(&syncCleanup, "cleanup", false,
		"Delete orphaned database tasks (files deleted)")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Get database path
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Parse conflict strategy
	strategy, err := parseConflictStrategy(syncStrategy)
	if err != nil {
		return fmt.Errorf("invalid strategy: %w", err)
	}

	// Default folder path
	folderPath := syncFolder
	if folderPath == "" {
		folderPath = "docs/plan"
	}

	// Create sync options
	opts := sync.SyncOptions{
		DBPath:        dbPath,
		FolderPath:    folderPath,
		DryRun:        syncDryRun,
		Strategy:      strategy,
		CreateMissing: syncCreateMissing,
		Cleanup:       syncCleanup,
	}

	// Create sync engine
	engine, err := sync.NewSyncEngine(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create sync engine: %w", err)
	}
	defer engine.Close()

	// Run sync with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := engine.Sync(ctx, opts)
	if err != nil {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
		}
		return err
	}

	// Output report
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "success",
			"report": report,
		})
	}

	displaySyncReport(report, syncDryRun)
	return nil
}

func parseConflictStrategy(s string) (sync.ConflictStrategy, error) {
	switch s {
	case "file-wins":
		return sync.ConflictStrategyFileWins, nil
	case "database-wins":
		return sync.ConflictStrategyDatabaseWins, nil
	case "newer-wins":
		return sync.ConflictStrategyNewerWins, nil
	default:
		return "", fmt.Errorf("unknown strategy: %s (valid: file-wins, database-wins, newer-wins)", s)
	}
}

func displaySyncReport(report *sync.SyncReport, dryRun bool) {
	if dryRun {
		cli.Warning("DRY-RUN MODE: No changes will be made")
		fmt.Println()
	}

	cli.Success("Sync completed:")
	fmt.Printf("  Files scanned:      %d\n", report.FilesScanned)
	fmt.Printf("  New tasks imported: %d\n", report.TasksImported)
	fmt.Printf("  Tasks updated:      %d\n", report.TasksUpdated)
	fmt.Printf("  Conflicts resolved: %d\n", report.ConflictsResolved)

	if report.TasksDeleted > 0 {
		fmt.Printf("  Tasks deleted:      %d\n", report.TasksDeleted)
	}

	fmt.Printf("  Warnings:           %d\n", len(report.Warnings))
	fmt.Printf("  Errors:             %d\n", len(report.Errors))

	// Display conflicts
	if len(report.Conflicts) > 0 {
		fmt.Println()
		fmt.Println("Conflicts:")
		for _, conflict := range report.Conflicts {
			fmt.Printf("  %s:\n", conflict.TaskKey)
			fmt.Printf("    Field:    %s\n", conflict.Field)
			fmt.Printf("    Database: %q\n", conflict.DatabaseValue)
			fmt.Printf("    File:     %q\n", conflict.FileValue)
		}
	}

	// Display warnings
	if len(report.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	// Display errors
	if len(report.Errors) > 0 {
		fmt.Println()
		fmt.Println("Errors:")
		for _, err := range report.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}
