package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/spf13/cobra"
)

var (
	backfillDryRun  bool
	backfillVerbose bool
)

var backfillSlugsCmd = &cobra.Command{
	Use:   "backfill-slugs",
	Short: "Backfill slug columns from existing file paths",
	Long: `Extract slugs from existing file_path values and populate the slug columns
for epics, features, and tasks.

This migration uses a three-phase approach for maximum coverage:
  Phase 1: Extract epic/feature slugs from task paths (highest coverage)
  Phase 2: Extract epic slugs from feature paths (fill gaps)
  Phase 3: Extract slugs from entity's own file_path

The migration is idempotent - it can be run multiple times safely.
Only records with NULL or empty slugs will be updated.`,
	Example: `  # Preview changes without applying them
  shark migrate backfill-slugs --dry-run

  # Apply migration
  shark migrate backfill-slugs

  # Apply with verbose output
  shark migrate backfill-slugs --verbose

  # Get JSON output for automation
  shark migrate backfill-slugs --json`,
	RunE: runBackfillSlugs,
}

func init() {
	migrateCmd.AddCommand(backfillSlugsCmd)

	backfillSlugsCmd.Flags().BoolVar(&backfillDryRun, "dry-run", false,
		"Preview changes without applying them")
	backfillSlugsCmd.Flags().BoolVarP(&backfillVerbose, "verbose", "v", false,
		"Show detailed output")
}

func runBackfillSlugs(cmd *cobra.Command, args []string) error {
	// Get database path
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Open database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Show dry-run notice
	if backfillDryRun && !cli.GlobalConfig.JSON {
		cli.Info("DRY RUN MODE - No changes will be applied")
		fmt.Println()
	}

	// Run backfill migration
	if backfillVerbose && !cli.GlobalConfig.JSON {
		cli.Info("Backfilling slugs from file paths...")
	}

	stats, err := db.BackfillSlugsFromFilePaths(database, backfillDryRun)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		return outputJSON(stats, backfillDryRun)
	}

	return outputTable(stats, backfillDryRun, backfillVerbose)
}

func outputJSON(stats *db.MigrationStats, dryRun bool) error {
	output := map[string]interface{}{
		"dry_run": dryRun,
		"epics": map[string]int{
			"total":       stats.EpicsTotal,
			"with_slugs":  stats.EpicsWithSlugs,
			"updated":     stats.EpicsUpdated,
			"after_count": stats.EpicsWithSlugs + stats.EpicsUpdated,
		},
		"features": map[string]int{
			"total":       stats.FeaturesTotal,
			"with_slugs":  stats.FeaturesWithSlugs,
			"updated":     stats.FeaturesUpdated,
			"after_count": stats.FeaturesWithSlugs + stats.FeaturesUpdated,
		},
		"tasks": map[string]int{
			"total":       stats.TasksTotal,
			"with_slugs":  stats.TasksWithSlugs,
			"updated":     stats.TasksUpdated,
			"after_count": stats.TasksWithSlugs + stats.TasksUpdated,
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputTable(stats *db.MigrationStats, dryRun bool, verbose bool) error {
	if dryRun {
		fmt.Println("Preview of slug backfill:")
	} else {
		if verbose {
			fmt.Println()
		}
		fmt.Println("Results:")
	}

	// Calculate before/after counts
	epicsBefore := stats.EpicsWithSlugs
	epicsAfter := stats.EpicsWithSlugs + stats.EpicsUpdated
	featuresBefore := stats.FeaturesWithSlugs
	featuresAfter := stats.FeaturesWithSlugs + stats.FeaturesUpdated
	tasksBefore := stats.TasksWithSlugs
	tasksAfter := stats.TasksWithSlugs + stats.TasksUpdated

	// Header
	if dryRun {
		fmt.Printf("  %-12s  %-15s  %-15s\n", "Entity", "Current", "Will Update")
		fmt.Printf("  %-12s  %-15s  %-15s\n", "------", "-------", "-----------")
	} else {
		fmt.Printf("  %-12s  %-15s  %-15s  %-10s\n", "Entity", "Before", "After", "Updated")
		fmt.Printf("  %-12s  %-15s  %-15s  %-10s\n", "------", "------", "-----", "-------")
	}

	// Data rows
	if dryRun {
		fmt.Printf("  %-12s  %-15s  %-15d\n",
			"Epics",
			fmt.Sprintf("%d/%d", epicsBefore, stats.EpicsTotal),
			stats.EpicsUpdated)
		fmt.Printf("  %-12s  %-15s  %-15d\n",
			"Features",
			fmt.Sprintf("%d/%d", featuresBefore, stats.FeaturesTotal),
			stats.FeaturesUpdated)
		fmt.Printf("  %-12s  %-15s  %-15d\n",
			"Tasks",
			fmt.Sprintf("%d/%d", tasksBefore, stats.TasksTotal),
			stats.TasksUpdated)
	} else {
		fmt.Printf("  %-12s  %-15s  %-15s  %-10d\n",
			"Epics",
			fmt.Sprintf("%d/%d", epicsBefore, stats.EpicsTotal),
			fmt.Sprintf("%d/%d", epicsAfter, stats.EpicsTotal),
			stats.EpicsUpdated)
		fmt.Printf("  %-12s  %-15s  %-15s  %-10d\n",
			"Features",
			fmt.Sprintf("%d/%d", featuresBefore, stats.FeaturesTotal),
			fmt.Sprintf("%d/%d", featuresAfter, stats.FeaturesTotal),
			stats.FeaturesUpdated)
		fmt.Printf("  %-12s  %-15s  %-15s  %-10d\n",
			"Tasks",
			fmt.Sprintf("%d/%d", tasksBefore, stats.TasksTotal),
			fmt.Sprintf("%d/%d", tasksAfter, stats.TasksTotal),
			stats.TasksUpdated)
	}

	fmt.Println()

	// Footer message
	if dryRun {
		cli.Warning("Run without --dry-run to apply changes")
	} else {
		totalUpdated := stats.EpicsUpdated + stats.FeaturesUpdated + stats.TasksUpdated
		if totalUpdated > 0 {
			cli.Success(fmt.Sprintf("Migration completed successfully! Updated %d records.", totalUpdated))
		} else {
			cli.Info("No updates needed - all slugs already populated.")
		}
	}

	return nil
}
