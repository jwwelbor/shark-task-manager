package commands

import (
	"fmt"
	"strconv"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// teamRunCmd is the reachable production entrypoint for executing an already
// confirmed team-run snapshot. Planning and confirmation remain separate
// workflow steps; this command only invokes the canonical scheduler boundary.
var teamRunCmd = &cobra.Command{
	Use:   "run <run-id>",
	Short: "Execute a confirmed team run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("invalid team run id %q", args[0])
		}
		result, err := cli.RunTeamSchedulerWithPlanHash(cmd.Context(), runID, teamRootSession, teamPlanHash)
		if err != nil {
			return err
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}
		fmt.Printf("team run %d: %s (%d items)\n", runID, result.Status, len(result.Items))
		return nil
	},
}

var teamRootSession string
var teamPlanHash string

func init() {
	teamRunCmd.Flags().StringVar(&teamRootSession, "root-session", "", "confirmed root claim session")
	teamRunCmd.Flags().StringVar(&teamPlanHash, "plan-hash", "", "hash captured from the confirmed plan")
	if err := teamRunCmd.MarkFlagRequired("root-session"); err != nil {
		panic(fmt.Sprintf("mark team run root-session required: %v", err))
	}
	if err := teamRunCmd.MarkFlagRequired("plan-hash"); err != nil {
		panic(fmt.Sprintf("mark team run plan-hash required: %v", err))
	}
	teamCmd.AddCommand(teamRunCmd)
}
