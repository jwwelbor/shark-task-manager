package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// analyticsCmd represents the analytics command group
var analyticsCmd = &cobra.Command{
	Use:     "analytics",
	Short:   "Analyze work session patterns and metrics",
	GroupID: "status",
	Long: `Analyze work session patterns across epics, features, and tasks.

Provides insights into:
  - Session duration patterns
  - Pause frequency
  - Time investment
  - Agent productivity

Examples:
  shark analytics --session-duration --epic E10
  shark analytics --pause-frequency --epic E10 --feature F05
  shark analytics --session-duration --epic E10 --agent-type backend`,
}

func init() {
	cli.RootCmd.AddCommand(analyticsCmd)

	// Flags
	analyticsCmd.Flags().Bool("session-duration", false, "Analyze session duration metrics")
	analyticsCmd.Flags().Bool("pause-frequency", false, "Analyze pause frequency patterns")
	analyticsCmd.Flags().String("epic", "", "Filter by epic key")
	analyticsCmd.Flags().String("feature", "", "Filter by feature key")
	analyticsCmd.Flags().String("agent-type", "", "Filter by agent type")
}

// runAnalytics executes the analytics command
func runAnalytics(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionDuration, _ := cmd.Flags().GetBool("session-duration")
	pauseFrequency, _ := cmd.Flags().GetBool("pause-frequency")
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	agentType, _ := cmd.Flags().GetString("agent-type")

	// Validate: at least one analysis type must be selected
	if !sessionDuration && !pauseFrequency {
		cli.Error("Please specify at least one analysis type: --session-duration or --pause-frequency")
		os.Exit(3)
	}

	// Validate: epic or feature must be specified
	if epicKey == "" && featureKey == "" {
		cli.Error("Please specify --epic or --feature for analysis scope")
		os.Exit(3)
	}

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Create repositories
	dbConn := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(dbConn)
	featureRepo := repository.NewFeatureRepository(dbConn)
	sessionRepo := repository.NewWorkSessionRepository(dbConn)

	var analytics *repository.SessionAnalytics
	var scopeDescription string

	// Get analytics data based on scope
	if featureKey != "" {
		// Feature-level analytics
		feature, err := featureRepo.GetByKey(ctx, featureKey)
		if err != nil {
			cli.Error(fmt.Sprintf("Feature %s not found", featureKey))
			os.Exit(1)
		}

		var agentTypePtr *string
		if agentType != "" {
			agentTypePtr = &agentType
		}

		analytics, err = sessionRepo.GetSessionAnalyticsByFeature(ctx, feature.ID, agentTypePtr)
		if err != nil {
			return fmt.Errorf("failed to get session analytics: %w", err)
		}

		scopeDescription = fmt.Sprintf("Feature %s", featureKey)
		if agentType != "" {
			scopeDescription += fmt.Sprintf(" (Agent: %s)", agentType)
		}

	} else if epicKey != "" {
		// Epic-level analytics
		epic, err := epicRepo.GetByKey(ctx, epicKey)
		if err != nil {
			cli.Error(fmt.Sprintf("Epic %s not found", epicKey))
			os.Exit(1)
		}

		var agentTypePtr *string
		if agentType != "" {
			agentTypePtr = &agentType
		}

		analytics, err = sessionRepo.GetSessionAnalyticsByEpic(ctx, epic.ID, agentTypePtr)
		if err != nil {
			return fmt.Errorf("failed to get session analytics: %w", err)
		}

		scopeDescription = fmt.Sprintf("Epic %s", epicKey)
		if agentType != "" {
			scopeDescription += fmt.Sprintf(" (Agent: %s)", agentType)
		}
	}

	// Output
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"scope":            scopeDescription,
			"session_duration": sessionDuration,
			"pause_frequency":  pauseFrequency,
			"analytics":        analytics,
		}
		return cli.OutputJSON(output)
	} else {
		if sessionDuration {
			printSessionDurationAnalytics(scopeDescription, analytics)
		}
		if pauseFrequency {
			printPauseFrequencyAnalytics(scopeDescription, analytics)
		}
	}

	return nil
}

// printSessionDurationAnalytics prints session duration analysis
func printSessionDurationAnalytics(scope string, analytics *repository.SessionAnalytics) {
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Session Duration Analysis: %s\n", scope)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	if analytics.TotalSessions == 0 {
		fmt.Println("No sessions found for analysis.")
		return
	}

	fmt.Printf("Overall Metrics:\n")
	fmt.Printf("  Total Sessions:        %d\n", analytics.TotalSessions)
	fmt.Printf("  Tasks with Sessions:   %d\n", analytics.TasksWithSessions)
	fmt.Printf("  Sessions per Task:     %.1f\n\n", analytics.AverageSessionsPerTask)

	fmt.Printf("Time Investment:\n")
	fmt.Printf("  Total Time:            %s\n", formatDuration(analytics.TotalDuration))
	if analytics.AverageDuration > 0 {
		fmt.Printf("  Average Session:       %s\n", formatDuration(analytics.AverageDuration))
		fmt.Printf("  Median Session:        %s\n\n", formatDuration(analytics.MedianDuration))
	}

	// Estimation guidance
	if analytics.AverageSessionsPerTask > 1 {
		fmt.Printf("Estimation Insights:\n")
		fmt.Printf("  • Tasks typically require %.1f sessions\n", analytics.AverageSessionsPerTask)
		if analytics.AverageDuration > 0 {
			estimatedTotal := time.Duration(float64(analytics.AverageDuration) * analytics.AverageSessionsPerTask)
			fmt.Printf("  • Estimated time per task: %s\n", formatDuration(estimatedTotal))
		}
		fmt.Printf("  • Factor this into future estimates\n")
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n\n")
}

// printPauseFrequencyAnalytics prints pause frequency analysis
func printPauseFrequencyAnalytics(scope string, analytics *repository.SessionAnalytics) {
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Pause Frequency Analysis: %s\n", scope)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	if analytics.TotalSessions == 0 {
		fmt.Println("No sessions found for analysis.")
		return
	}

	fmt.Printf("Pause Patterns:\n")
	fmt.Printf("  Total Sessions:        %d\n", analytics.TotalSessions)
	fmt.Printf("  Tasks with Sessions:   %d\n", analytics.TasksWithSessions)
	fmt.Printf("  Tasks with Pauses:     %d\n", analytics.TasksWithPauses)
	fmt.Printf("  Pause Rate:            %.1f%%\n\n", analytics.PauseRate)

	fmt.Printf("Sessions per Task:       %.1f\n\n", analytics.AverageSessionsPerTask)

	// Insights
	fmt.Printf("Insights:\n")
	if analytics.PauseRate > 50 {
		fmt.Printf("  ⚠ High pause rate (%.1f%%) suggests:\n", analytics.PauseRate)
		fmt.Printf("    • Tasks may be blocked frequently\n")
		fmt.Printf("    • Requirements may be unclear\n")
		fmt.Printf("    • External dependencies causing delays\n")
	} else if analytics.PauseRate > 20 {
		fmt.Printf("  ℹ Moderate pause rate (%.1f%%) is normal for:\n", analytics.PauseRate)
		fmt.Printf("    • Complex features requiring research\n")
		fmt.Printf("    • Tasks with external dependencies\n")
	} else {
		fmt.Printf("  ✓ Low pause rate (%.1f%%) indicates:\n", analytics.PauseRate)
		fmt.Printf("    • Clear requirements\n")
		fmt.Printf("    • Minimal blockers\n")
		fmt.Printf("    • Good task independence\n")
	}

	if analytics.AverageSessionsPerTask > 3 {
		fmt.Printf("\n  ⚠ High sessions per task (%.1f) suggests:\n", analytics.AverageSessionsPerTask)
		fmt.Printf("    • Consider breaking down tasks\n")
		fmt.Printf("    • Tasks may be too large\n")
		fmt.Printf("    • Frequent interruptions\n")
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n\n")
}

func init() {
	analyticsCmd.RunE = runAnalytics
}
