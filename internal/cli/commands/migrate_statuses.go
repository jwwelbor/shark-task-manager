package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
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

var legacyStatusRepairMaps = map[string]map[string]string{
	"epic": {
		"in_decomposition":         "decomposition",
		"in_design":                "design",
		"in_feature_review":        "feature_review",
		"in_refinement":            "refinement",
		"in_research":              "research",
		"ready_for_decomposition":  "decomposition",
		"ready_for_design":         "design",
		"ready_for_feature_review": "feature_review",
		"ready_for_refinement":     "refinement",
		"ready_for_research":       "research",
	},
	"feature": {
		"in_approval":               "approval",
		"in_assessment":             "assessment",
		"in_code_review":            "code_review",
		"in_qa":                     "qa",
		"in_research":               "research",
		"in_specification":          "specification",
		"in_task_generation":        "task_generation",
		"in_task_review":            "task_review",
		"in_test_planning":          "test_planning",
		"ready_for_approval":        "approval",
		"ready_for_assessment":      "assessment",
		"ready_for_code_review":     "code_review",
		"ready_for_qa":              "qa",
		"ready_for_research":        "research",
		"ready_for_specification":   "specification",
		"ready_for_task_generation": "task_generation",
		"ready_for_task_review":     "task_review",
		"ready_for_test_planning":   "test_planning",
	},
	"task": {
		"in_approval":             "development",
		"in_code_review":          "development",
		"in_development":          "development",
		"in_progress":             "development",
		"in_qa":                   "development",
		"research":                "development",
		"ready_for_approval":      "development",
		"ready_for_code_review":   "development",
		"ready_for_development":   "development",
		"ready_for_qa":            "development",
		"ready_for_refinement_ba": "draft",
		"todo":                    "draft",
	},
	"bug": {
		"in_code_review":        "code_review",
		"in_development":        "development",
		"in_qa":                 "qa",
		"ready_for_code_review": "code_review",
		"ready_for_development": "development",
		"ready_for_qa":          "qa",
	},
	"change": {
		"in_code_review":         "code_review",
		"in_development":         "development",
		"in_qa":                  "qa",
		"in_verification":        "qa",
		"ready_for_code_review":  "code_review",
		"ready_for_development":  "development",
		"ready_for_qa":           "qa",
		"ready_for_verification": "qa",
	},
	"tech_debt": {
		"progress": "in_progress",
	},
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

	planned, err := collectStatusRewrites(cmd.Context(), repoDb, wfSvc)
	if err != nil {
		return err
	}

	if !apply {
		return reportStatusMigration(planned, false)
	}

	if err := applyStatusRewrites(cmd.Context(), repoDb, planned); err != nil {
		return err
	}

	return reportStatusMigration(planned, true)
}

// collectStatusRewrites computes the planned old->new status rewrites across all
// entity tables, using each level's alias map. Identity aliases (old == new) and
// statuses with no matching rows are skipped. Extracted from the command so it
// can be exercised against a real test DB (B1-F4).
func collectStatusRewrites(ctx context.Context, repoDb *repository.DB, wfSvc *workflow.Service) ([]statusRewrite, error) {
	var planned []statusRewrite
	for _, tgt := range statusMigrationTargets {
		repairMap := legacyStatusRepairMapForLevel(tgt.level, wfSvc)
		if err := failOnUnmappedLegacyStatuses(ctx, repoDb, tgt, repairMap); err != nil {
			return nil, err
		}
		// Deterministic ordering of old names.
		olds := make([]string, 0, len(repairMap))
		for old := range repairMap {
			olds = append(olds, old)
		}
		sort.Strings(olds)

		for _, old := range olds {
			newStep := repairMap[old]
			if old == newStep {
				continue
			}
			var count int
			countQ := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE status = ?", tgt.table) //nolint:gosec // table from fixed allowlist
			if err := repoDb.QueryRowContext(ctx, countQ, old).Scan(&count); err != nil {
				return nil, fmt.Errorf("count %s.status=%q: %w", tgt.table, old, err)
			}
			if count > 0 {
				planned = append(planned, statusRewrite{Table: tgt.table, Level: tgt.level, Old: old, New: newStep, Count: count})
			}
		}
	}
	return planned, nil
}

func legacyStatusRepairMapForLevel(level string, wfSvc *workflow.Service) map[string]string {
	out := map[string]string{}
	for old, newStep := range legacyStatusRepairMaps[level] {
		out[old] = newStep
	}
	if wfSvc != nil {
		for old, newStep := range wfSvc.ForLevel(level).StatusAliasMap() {
			out[old] = newStep
		}
	}
	return out
}

func failOnUnmappedLegacyStatuses(ctx context.Context, repoDb *repository.DB, tgt statusMigrationTarget, repairMap map[string]string) error {
	query := fmt.Sprintf("SELECT DISTINCT status FROM %s", tgt.table) //nolint:gosec // table from fixed allowlist
	rows, err := repoDb.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("list distinct %s statuses: %w", tgt.table, err)
	}
	defer func() { _ = rows.Close() }()

	var unmapped []string
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return fmt.Errorf("scan %s status: %w", tgt.table, err)
		}
		if _, ok := repairMap[status]; !ok && isLegacyStatusValue(status, tgt.level) {
			unmapped = append(unmapped, status)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s statuses: %w", tgt.table, err)
	}
	if len(unmapped) > 0 {
		sort.Strings(unmapped)
		return fmt.Errorf("%s.status contains legacy status value(s) with no migration mapping: %s", tgt.table, strings.Join(unmapped, ", "))
	}
	return nil
}

func isLegacyStatusValue(status, level string) bool {
	if strings.HasPrefix(status, "ready_for_") {
		return true
	}
	if !strings.HasPrefix(status, "in_") {
		return false
	}
	return !(level == "tech_debt" && status == "in_progress")
}

// applyStatusRewrites executes the planned rewrites in a single transaction,
// updating each entry's Count to the rows actually affected. task_history is
// never touched. Extracted from the command for testability (B1-F4).
func applyStatusRewrites(ctx context.Context, repoDb *repository.DB, planned []statusRewrite) error {
	tx, err := repoDb.BeginTx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for i := range planned {
		r := &planned[i]
		updQ := fmt.Sprintf("UPDATE %s SET status = ? WHERE status = ?", r.Table) //nolint:gosec // table from fixed allowlist
		res, err := tx.ExecContext(ctx, updQ, r.New, r.Old)
		if err != nil {
			return fmt.Errorf("rewrite %s %q->%q: %w", r.Table, r.Old, r.New, err)
		}
		// Report the rows actually rewritten, not the pre-flight COUNT(*) plan
		// (which can drift if the data changed between planning and apply).
		if affected, aerr := res.RowsAffected(); aerr == nil {
			r.Count = int(affected)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
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
