package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// dashboardAnalyticsServicer is the minimal interface that the analytics command
// requires from DashboardAnalyticsService. Defined at the point of use so that
// tests can inject lightweight mocks without depending on the concrete service type.
type dashboardAnalyticsServicer interface {
	GetBugAnalytics(ctx context.Context) (*services.BugAnalyticsResult, error)
	GetChangeCardAnalytics(ctx context.Context) (*services.ChangeCardAnalyticsResult, error)
	GetTechDebtAnalytics(ctx context.Context) (*services.TechDebtAnalyticsResult, error)
}

// analyticsCmd represents the analytics command group
var analyticsCmd = &cobra.Command{
	Use:     "analytics",
	Short:   "Analyze work session patterns and metrics",
	GroupID: "advanced",
	Long: `Analyze work session patterns across epics, features, and tasks.

Provides insights into:
  - Session duration patterns
  - Pause frequency
  - Time investment
  - Agent productivity
  - Bug analytics (--type=bug)
  - Change-card analytics (--type=change)
  - Tech-debt analytics (--type=tech_debt)

Examples:
  shark analytics --session-duration --epic E10
  shark analytics --pause-frequency --epic E10 --feature F05
  shark analytics --type=bug
  shark analytics --type=change
  shark analytics --type=tech_debt
  shark analytics`,
	RunE: runAnalytics,
}

func init() {
	cli.RootCmd.AddCommand(analyticsCmd)

	// Flags
	analyticsCmd.Flags().Bool("session-duration", false, "Analyze session duration metrics")
	analyticsCmd.Flags().Bool("pause-frequency", false, "Analyze pause frequency patterns")
	analyticsCmd.Flags().String("epic", "", "Filter by epic key")
	analyticsCmd.Flags().String("feature", "", "Filter by feature key")
	analyticsCmd.Flags().String("agent-type", "", "Filter by agent type")
	analyticsCmd.Flags().String("type", "", "Entity type analytics: bug, change, tech_debt")
}

// runAnalytics executes the analytics command
func runAnalytics(cmd *cobra.Command, args []string) error {
	entityType, _ := cmd.Flags().GetString("type")
	sessionDuration, _ := cmd.Flags().GetBool("session-duration")
	pauseFrequency, _ := cmd.Flags().GetBool("pause-frequency")

	// Route to entity-specific analytics when --type is provided
	if entityType == "bug" || entityType == "change" || entityType == "tech_debt" {
		return runEntityAnalytics(cmd, entityType)
	}

	// Reject unknown --type values
	if entityType != "" {
		return fmt.Errorf("unknown entity type %q: valid values are 'bug', 'change', 'tech_debt'", entityType)
	}

	// No session flags and no --type: show combined analytics
	if !sessionDuration && !pauseFrequency {
		return runCombinedAnalytics(cmd)
	}

	// Session analytics path (existing behaviour)
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	agentType, _ := cmd.Flags().GetString("agent-type")

	// Validate: epic or feature must be specified for session analytics
	if epicKey == "" && featureKey == "" {
		return fmt.Errorf("please specify --epic or --feature for analysis scope")
	}

	// Build scope description
	scopeDescription := ""
	if featureKey != "" {
		scopeDescription = fmt.Sprintf("Feature %s", featureKey)
	} else {
		scopeDescription = fmt.Sprintf("Epic %s", epicKey)
	}
	if agentType != "" {
		scopeDescription += fmt.Sprintf(" (Agent: %s)", agentType)
	}

	// Call service
	svc := cli.GetTaskServiceWithDocs()
	input := services.SessionAnalyticsInput{
		EpicKey:    epicKey,
		FeatureKey: featureKey,
		AgentType:  agentType,
	}

	analytics, err := svc.GetSessionAnalytics(cmd.Context(), input)
	if err != nil {
		return fmt.Errorf("failed to get session analytics: %w", err)
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
	}

	if sessionDuration {
		printSessionDurationAnalytics(scopeDescription, analytics)
	}
	if pauseFrequency {
		printPauseFrequencyAnalytics(scopeDescription, analytics)
	}

	return nil
}

// runEntityAnalytics handles --type=bug or --type=change by calling the
// DashboardAnalyticsService and formatting the result.
func runEntityAnalytics(cmd *cobra.Command, entityType string) error {
	svc := cli.GetDashboardAnalyticsService()
	return runEntityAnalyticsWithSvc(cmd.Context(), entityType, svc)
}

// runEntityAnalyticsWithSvc is the testable core of runEntityAnalytics.
// It accepts the service as an interface so tests can inject a mock.
func runEntityAnalyticsWithSvc(ctx context.Context, entityType string, svc dashboardAnalyticsServicer) error {
	switch entityType {
	case "bug":
		result, err := svc.GetBugAnalytics(ctx)
		if err != nil {
			return fmt.Errorf("failed to get bug analytics: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}
		printBugAnalytics(result)
		return nil

	case "change":
		result, err := svc.GetChangeCardAnalytics(ctx)
		if err != nil {
			return fmt.Errorf("failed to get change-card analytics: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}
		printChangeCardAnalytics(result)
		return nil

	case "tech_debt":
		result, err := svc.GetTechDebtAnalytics(ctx)
		if err != nil {
			return fmt.Errorf("failed to get tech-debt analytics: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}
		printTechDebtAnalytics(result)
		return nil

	default:
		return fmt.Errorf("unknown entity type %q: valid values are 'bug', 'change', 'tech_debt'", entityType)
	}
}

// runCombinedAnalytics shows combined bug + change-card analytics when the user
// runs `shark analytics` without --type or session flags.
func runCombinedAnalytics(cmd *cobra.Command) error {
	svc := cli.GetDashboardAnalyticsService()
	return runCombinedAnalyticsWithSvc(cmd.Context(), svc)
}

// runCombinedAnalyticsWithSvc is the testable core of runCombinedAnalytics.
// It accepts the service as an interface so tests can inject a mock.
// Sections that fail gracefully (e.g. repo not configured) are omitted rather
// than returning an error.
func runCombinedAnalyticsWithSvc(ctx context.Context, svc dashboardAnalyticsServicer) error {
	bugResult, bugErr := svc.GetBugAnalytics(ctx)
	ccResult, ccErr := svc.GetChangeCardAnalytics(ctx)
	tdResult, tdErr := svc.GetTechDebtAnalytics(ctx)

	if cli.GlobalConfig.JSON {
		combined := &services.DashboardAnalyticsResult{}
		if bugErr == nil {
			combined.Bugs = bugResult
		}
		if ccErr == nil {
			combined.ChangeCards = ccResult
		}
		if tdErr == nil {
			combined.TechDebts = tdResult
		}
		return cli.OutputJSON(combined)
	}

	if bugErr == nil && bugResult != nil {
		printBugAnalytics(bugResult)
	}
	if ccErr == nil && ccResult != nil {
		printChangeCardAnalytics(ccResult)
	}
	if tdErr == nil && tdResult != nil {
		printTechDebtAnalytics(tdResult)
	}

	return nil
}

// printBugAnalytics renders a human-readable bug analytics report to stdout.
// It is nil-safe: pointer fields such as AvgResolutionTimeSecs are printed
// as "N/A" when nil.
func printBugAnalytics(result *services.BugAnalyticsResult) {
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Bug Analytics\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	fmt.Printf("Overview:\n")
	fmt.Printf("  Total Bugs:            %d\n", result.TotalBugs)
	fmt.Printf("  Resolved:              %d\n", result.ResolvedCount)

	if result.AvgResolutionTimeSecs != nil {
		fmt.Printf("  Avg Resolution Time:   %s\n", formatDurationFromSecs(*result.AvgResolutionTimeSecs))
	} else {
		fmt.Printf("  Avg Resolution Time:   N/A\n")
	}

	if len(result.BugsByStatus) > 0 {
		fmt.Printf("\nBy Status:\n")
		for status, count := range result.BugsByStatus {
			fmt.Printf("  %-20s %d\n", status+":", count)
		}
	}

	if len(result.BugsBySeverity) > 0 {
		fmt.Printf("\nBy Severity:\n")
		for severity, count := range result.BugsBySeverity {
			fmt.Printf("  %-20s %d\n", severity+":", count)
		}
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n\n")
}

// printChangeCardAnalytics renders a human-readable change-card analytics report
// to stdout. It is nil-safe: pointer fields are printed as "N/A" when nil.
func printChangeCardAnalytics(result *services.ChangeCardAnalyticsResult) {
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Change Card Analytics\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	fmt.Printf("Overview:\n")
	fmt.Printf("  Total Change Cards:    %d\n", result.TotalChangeCards)
	fmt.Printf("  Decided:               %d\n", result.DecidedCount)
	fmt.Printf("  Completed:             %d\n", result.CompletedCount)

	if result.ApprovalRate != nil {
		fmt.Printf("  Approval Rate:         %.0f%%\n", *result.ApprovalRate*100)
	} else {
		fmt.Printf("  Approval Rate:         N/A\n")
	}

	if result.AvgCompletionTimeSecs != nil {
		fmt.Printf("  Avg Completion Time:   %s\n", formatDurationFromSecs(*result.AvgCompletionTimeSecs))
	} else {
		fmt.Printf("  Avg Completion Time:   N/A\n")
	}

	if len(result.ChangeCardsByStatus) > 0 {
		fmt.Printf("\nBy Status:\n")
		for status, count := range result.ChangeCardsByStatus {
			fmt.Printf("  %-20s %d\n", status+":", count)
		}
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n\n")
}

// printTechDebtAnalytics renders a human-readable tech-debt analytics report to stdout.
func printTechDebtAnalytics(result *services.TechDebtAnalyticsResult) {
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Tech Debt Analytics\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	fmt.Printf("Overview:\n")
	fmt.Printf("  Total Tech Debts:      %d\n", result.TotalTechDebts)

	if len(result.TechDebtsByStatus) > 0 {
		fmt.Printf("\nBy Status:\n")
		for status, count := range result.TechDebtsByStatus {
			fmt.Printf("  %-20s %d\n", status+":", count)
		}
	}

	if len(result.TechDebtsByCategory) > 0 {
		fmt.Printf("\nBy Category:\n")
		for category, count := range result.TechDebtsByCategory {
			fmt.Printf("  %-20s %d\n", category+":", count)
		}
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n\n")
}

// formatDurationFromSecs converts a duration expressed in floating-point seconds
// into a concise human-readable string such as "2d 3h", "45m 30s", "30s".
// It is used to display avg_resolution_time_seconds and avg_completion_time_seconds
// from the analytics DTOs.
func formatDurationFromSecs(secs float64) string {
	if secs <= 0 {
		return "0s"
	}

	totalSecs := int64(secs)
	days := totalSecs / 86400
	remainder := totalSecs % 86400
	hours := remainder / 3600
	remainder = remainder % 3600
	mins := remainder / 60
	sec := remainder % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins > 0 && days == 0 { // omit minutes when days are shown
		parts = append(parts, fmt.Sprintf("%dm", mins))
	}
	if sec > 0 && days == 0 && hours == 0 {
		parts = append(parts, fmt.Sprintf("%ds", sec))
	}

	if len(parts) == 0 {
		return "0s"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// printSessionDurationAnalytics prints session duration analysis
func printSessionDurationAnalytics(scope string, analytics *services.SessionAnalytics) {
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
func printPauseFrequencyAnalytics(scope string, analytics *services.SessionAnalytics) {
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
