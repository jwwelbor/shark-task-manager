package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	taskRepo := repository.NewTaskRepository(dbConn)
	sessionRepo := repository.NewWorkSessionRepository(dbConn)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		os.Exit(1)
	}

	// Get work sessions
	sessions, err := sessionRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get work sessions: %w", err)
	}

	// Get session stats
	stats, err := sessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get session stats: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":   taskKey,
			"task_title": task.Title,
			"sessions":   sessions,
			"stats":      stats,
		}
		return cli.OutputJSON(output)
	}

	printSessions(taskKey, task.Title, sessions, stats)
	return nil
}

// printSessions prints human-readable session information
func printSessions(taskKey, taskTitle string, sessions []*models.WorkSession, stats *repository.SessionStats) {
	// Header
	fmt.Printf("Task %s: %s\n", taskKey, taskTitle)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	if len(sessions) == 0 {
		fmt.Println("No work sessions found for this task.")
		return
	}

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

	for i, session := range sessions {
		sessionNum := len(sessions) - i
		startTime := session.StartedAt.Format("2006-01-02 15:04")

		if session.IsActive() {
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
