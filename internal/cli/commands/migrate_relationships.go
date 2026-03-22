package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/spf13/cobra"
)

var migrateRelationshipsCmd = &cobra.Command{
	Use:   "relationships",
	Short: "Migrate legacy relationship data to entity_relationships table",
	Long: `Migrate data from legacy relationship tables into the unified entity_relationships table.

This migration runs 5 phases:
  Phase 1: task_relationships → entity_relationships
  Phase 2: depends_on JSON column → entity_relationships
  Phase 3: bug linked_entity columns → entity_relationships
  Phase 4: change_card FK columns → entity_relationships
  Phase 5: epic_relationships and feature_relationships → entity_relationships

The migration is idempotent - INSERT OR IGNORE prevents duplicate rows.
Tables that do not exist are skipped automatically.`,
	Example: `  # Run the migration
  shark admin migrate relationships

  # Get JSON output for automation
  shark admin migrate relationships --json`,
	RunE: runMigrateRelationships,
}

func init() {
	migrateCmd.AddCommand(migrateRelationshipsCmd)
}

func runMigrateRelationships(cmd *cobra.Command, args []string) error {
	// Get database (initialized via cli.GetDB which calls InitDB, ensuring schema exists)
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Access the underlying *sql.DB from the repository.DB wrapper
	database := repoDb.DB

	if !cli.GlobalConfig.JSON {
		cli.Info("Migrating legacy relationship data to entity_relationships...")
		fmt.Println()
	}

	// Run migration
	counts, err := db.MigrateDataToEntityRelationships(database)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Get total count from entity_relationships for verification
	var totalRows int64
	row := database.QueryRow("SELECT COUNT(*) FROM entity_relationships")
	if scanErr := row.Scan(&totalRows); scanErr != nil {
		return fmt.Errorf("failed to count entity_relationships: %w", scanErr)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		return outputMigrateRelationshipsJSON(counts, totalRows)
	}

	return outputMigrateRelationshipsTable(counts, totalRows)
}

func outputMigrateRelationshipsJSON(counts map[string]int64, totalRows int64) error {
	output := map[string]interface{}{
		"phases": map[string]int64{
			"task_relationships": counts["phase1_task_relationships"],
			"depends_on_json":    counts["phase2_depends_on_json"],
			"bug_linked_entity":  counts["phase3_bug_linked_entity"],
			"change_card_fks":    counts["phase4_change_card_fks"],
			"epic_feature_rels":  counts["phase5_epic_feature_rels"],
		},
		"total_rows_in_entity_relationships": totalRows,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputMigrateRelationshipsTable(counts map[string]int64, totalRows int64) error {
	fmt.Printf("  %-45s  %s\n", "Phase", "Rows Inserted")
	fmt.Printf("  %-45s  %s\n", "-----", "-------------")
	fmt.Printf("  %-45s  %d\n", "Phase 1: task_relationships", counts["phase1_task_relationships"])
	fmt.Printf("  %-45s  %d\n", "Phase 2: depends_on JSON column", counts["phase2_depends_on_json"])
	fmt.Printf("  %-45s  %d\n", "Phase 3: bug linked_entity columns", counts["phase3_bug_linked_entity"])
	fmt.Printf("  %-45s  %d\n", "Phase 4: change_card FK columns", counts["phase4_change_card_fks"])
	fmt.Printf("  %-45s  %d\n", "Phase 5: epic/feature relationship tables", counts["phase5_epic_feature_rels"])
	fmt.Println()

	totalInserted := counts["phase1_task_relationships"] +
		counts["phase2_depends_on_json"] +
		counts["phase3_bug_linked_entity"] +
		counts["phase4_change_card_fks"] +
		counts["phase5_epic_feature_rels"]

	fmt.Printf("  Total rows inserted this run: %d\n", totalInserted)
	fmt.Printf("  Total rows in entity_relationships: %d\n", totalRows)
	fmt.Println()

	if totalInserted > 0 {
		cli.Success(fmt.Sprintf("Migration completed successfully! Inserted %d rows.", totalInserted))
	} else {
		cli.Info("No new rows inserted - all data already migrated.")
	}

	return nil
}
