package commands

import (
	"fmt"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// statusMigrationTarget maps an entity table to the workflow level whose alias
// map governs it (E35-F05).
type statusMigrationTarget struct {
	table string
	level string
}

var statusMigrationTargets = []statusMigrationTarget{
	{"epics", "epic"},
	{"features", "feature"},
	{"tasks", "task"},
	{"bugs", "bug"},
	{"change_cards", "change"},
	{"tech_debts", "tech_debt"},
	{"sprints", "sprint"},
}

var migrateStatusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "Rewrite legacy status values to route-based step names (one-shot)",
	Long: `Rewrite each entity's live status column from old status names
(ready_for_X / in_X) to the consolidated route-based step names, using the
per-step aliases: defined in the active workflow (E35-F05).

This is a ONE-SHOT migration and is DESTRUCTIVE to the status column. It is
NOT run automatically — you must opt in with --apply. By default it runs in
dry-run mode and only reports what would change.

task_history is intentionally left untouched: audit trails record what actually
happened, and old names are alias-resolved on read instead.

Examples:
  shark admin migrate statuses              Dry-run: report what would change
  shark admin migrate statuses --apply      Execute the rewrite
  shark admin migrate statuses --json       Machine-readable report`,
	Args: cobra.NoArgs,
	RunE: runMigrateStatuses,
}

func init() {
	migrateStatusesCmd.Flags().Bool("apply", false, "Execute the rewrite (default is dry-run)")
	migrateCmd.AddCommand(migrateStatusesCmd)
}

// statusRewrite describes one old->new status change in one table.
type statusRewrite struct {
	Table string `json:"table"`
	Level string `json:"level"`
	Old   string `json:"old_status"`
	New   string `json:"new_status"`
	Count int    `json:"count"`
}

func runMigrateStatuses(cmd *cobra.Command, args []string) error {
	apply, _ := cmd.Flags().GetBool("apply")

	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	wfSvc := cli.GetWorkflowService()

	var planned []statusRewrite
	for _, tgt := range statusMigrationTargets {
		aliasMap := wfSvc.ForLevel(tgt.level).StatusAliasMap()
		if len(aliasMap) == 0 {
			continue // not a route-based workflow for this level
		}
		// Deterministic ordering of old names.
		olds := make([]string, 0, len(aliasMap))
		for old := range aliasMap {
			olds = append(olds, old)
		}
		sort.Strings(olds)

		for _, old := range olds {
			newStep := aliasMap[old]
			if old == newStep {
				continue
			}
			var count int
			countQ := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE status = ?", tgt.table) //nolint:gosec // table from fixed allowlist
			if err := repoDb.QueryRowContext(cmd.Context(), countQ, old).Scan(&count); err != nil {
				return fmt.Errorf("count %s.status=%q: %w", tgt.table, old, err)
			}
			if count > 0 {
				planned = append(planned, statusRewrite{Table: tgt.table, Level: tgt.level, Old: old, New: newStep, Count: count})
			}
		}
	}

	if !apply {
		return reportStatusMigration(planned, false)
	}

	// Apply in a single transaction.
	tx, err := repoDb.BeginTx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range planned {
		updQ := fmt.Sprintf("UPDATE %s SET status = ? WHERE status = ?", r.Table) //nolint:gosec // table from fixed allowlist
		if _, err := tx.ExecContext(cmd.Context(), updQ, r.New, r.Old); err != nil {
			return fmt.Errorf("rewrite %s %q->%q: %w", r.Table, r.Old, r.New, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return reportStatusMigration(planned, true)
}

func reportStatusMigration(planned []statusRewrite, applied bool) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"applied":  applied,
			"rewrites": planned,
		})
	}
	if len(planned) == 0 {
		cli.Info("No legacy status values to migrate (workflows are not route-based, or all statuses are already current).")
		return nil
	}
	total := 0
	headers := []string{"Table", "Old Status", "New Step", "Rows"}
	rows := make([][]string, 0, len(planned))
	for _, r := range planned {
		rows = append(rows, []string{r.Table, r.Old, r.New, fmt.Sprintf("%d", r.Count)})
		total += r.Count
	}
	cli.OutputTable(headers, rows)
	if applied {
		cli.Success(fmt.Sprintf("Rewrote %d row(s) across %d change(s). task_history left untouched.", total, len(planned)))
	} else {
		cli.Warning(fmt.Sprintf("DRY-RUN: %d row(s) across %d change(s) would be rewritten. Re-run with --apply to execute.", total, len(planned)))
	}
	return nil
}
