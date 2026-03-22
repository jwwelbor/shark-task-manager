package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// taskSessionsCmd displays all work sessions for a task
var taskSessionsCmd = &cobra.Command{
	Use:   "sessions <task-key>",
	Short: "View all work sessions for a task",
	Long: `View all work sessions for a task with durations and outcomes.

Shows:
  - Session start/end times
  - Duration for each session
  - Session outcome (completed, paused, blocked)
  - Session notes
  - Total time spent
  - Average session duration

Examples:
  shark task sessions T-E10-F05-001
  shark task sessions T-E10-F05-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskSessions,
}

func init() {
	taskCmd.AddCommand(taskSessionsCmd)
}

// runTaskSessions displays work sessions for a task
func runTaskSessions(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey := args[0]

	// Step 2: Call service
	svc := cli.GetTaskServiceWithDocs()
	result, err := svc.GetWorkSessions(cmd.Context(), taskKey)
	if err != nil {
		return fmt.Errorf("failed to get work sessions for %s: %w", taskKey, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	printSessions(result)
	return nil
}

// printSessions prints human-readable session information
func printSessions(result *services.TaskWorkSessions) {
	fmt.Printf("Task %s: %s\n", result.TaskKey, result.TaskTitle)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	if len(result.Sessions) == 0 {
		fmt.Println("No work sessions found for this task.")
		return
	}

	stats := result.Stats

	// Summary stats
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Sessions:   %d\n", stats.TotalSessions)
	if stats.TotalDuration > 0 {
		fmt.Printf("  Total Time:       %s\n", formatDuration(stats.TotalDuration))
	}
	if stats.AverageDuration > 0 {
		fmt.Printf("  Average Session:  %s\n", formatDuration(stats.AverageDuration))
	}
	if stats.ActiveSession {
		fmt.Printf("  Active Session:   Yes\n")
	}
	fmt.Println()

	// Session list
	fmt.Printf("Session History:\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")

	sessions := result.Sessions
	for i, session := range sessions {
		sessionNum := len(sessions) - i
		startTime := session.StartedAt.Format("2006-01-02 15:04")

		if isActiveSession(session) {
			// Active session
			duration := formatDuration(session.Duration())
			fmt.Printf("\nSession %d: %s - Active (%s)\n", sessionNum, startTime, duration)
			if session.AgentID != nil {
				fmt.Printf("  Agent: %s\n", *session.AgentID)
			}
		} else {
			// Completed session
			endTime := session.EndedAt.Time.Format("2006-01-02 15:04")
			duration := formatDuration(session.Duration())
			outcome := "unknown"
			if session.Outcome != nil {
				outcome = string(*session.Outcome)
			}

			fmt.Printf("\nSession %d: %s - %s (%s) → %s\n", sessionNum, startTime, endTime, duration, outcome)
			if session.AgentID != nil {
				fmt.Printf("  Agent: %s\n", *session.AgentID)
			}
			if session.SessionNotes != nil && *session.SessionNotes != "" {
				fmt.Printf("  Note: %s\n", *session.SessionNotes)
			}
		}
	}

	fmt.Printf("\n───────────────────────────────────────────────────────────────\n")
}

// isActiveSession checks if a work session is currently active
func isActiveSession(session *models.WorkSession) bool {
	return session.IsActive()
}
