package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// taskResumeCmd provides comprehensive context for resuming a task
var taskResumeCmd = &cobra.Command{
	Use:   "resume <task-key>",
	Short: "Get comprehensive context for resuming a task",
	Long: `Get all context needed to resume work on a task in a single command.

This includes:
  - Task details (title, description, status, priority, dependencies)
  - Context data (progress, decisions, questions, blockers, acceptance criteria)
  - Task notes (chronologically ordered)
  - Completion metadata (if task is completed)
  - Work sessions (all sessions with durations and outcomes)

Examples:
  shark task resume T-E10-F05-001
  shark task resume T-E10-F05-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskResume,
}

func init() {
	taskCmd.AddCommand(taskResumeCmd)
}

// ResumeContext aggregates all context needed to resume a task
type ResumeContext struct {
	Task           *models.Task               `json:"task"`
	ContextData    *models.ContextData        `json:"context_data,omitempty"`
	Notes          []*models.TaskNote         `json:"notes,omitempty"`
	WorkSessions   []*models.WorkSession      `json:"work_sessions,omitempty"`
	SessionStats   *repository.SessionStats   `json:"session_stats,omitempty"`
	ActiveSession  *models.WorkSession        `json:"active_session,omitempty"`
	Dependencies   []string                   `json:"dependencies,omitempty"`
	CompletionMeta *models.CompletionMetadata `json:"completion_metadata,omitempty"`
}

// runTaskResume retrieves and displays comprehensive task context
func runTaskResume(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repositories
	dbConn := repoDb
	taskRepo := repository.NewTaskRepository(dbConn)
	noteRepo := repository.NewTaskNoteRepository(dbConn)
	sessionRepo := repository.NewWorkSessionRepository(dbConn)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		os.Exit(1)
	}

	// Build resume context
	resumeCtx := &ResumeContext{
		Task: task,
	}

	// Parse context data
	if task.ContextData != nil && *task.ContextData != "" && *task.ContextData != "{}" {
		contextData, err := models.FromJSON(*task.ContextData)
		if err == nil {
			resumeCtx.ContextData = contextData
		}
	}

	// Get notes
	notes, err := noteRepo.GetByTaskID(ctx, task.ID)
	if err == nil {
		resumeCtx.Notes = notes
	}

	// Get work sessions
	sessions, err := sessionRepo.GetByTaskID(ctx, task.ID)
	if err == nil {
		resumeCtx.WorkSessions = sessions
	}

	// Get session stats
	stats, err := sessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
	if err == nil {
		resumeCtx.SessionStats = stats
	}

	// Get active session
	activeSession, err := sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
	if err == nil && err != sql.ErrNoRows {
		resumeCtx.ActiveSession = activeSession
	}

	// Parse dependencies
	if task.DependsOn != nil && *task.DependsOn != "" {
		deps := strings.Split(strings.Trim(*task.DependsOn, "[]"), ",")
		for i, dep := range deps {
			deps[i] = strings.Trim(strings.Trim(dep, "\""), " ")
		}
		resumeCtx.Dependencies = deps
	}

	// Build completion metadata if task is completed
	if task.Status == models.TaskStatus("completed") || task.Status == models.TaskStatus("ready_for_review") {
		completionMeta := &models.CompletionMetadata{
			CompletedBy:      task.CompletedBy,
			CompletionNotes:  task.CompletionNotes,
			TestsPassed:      task.TestsPassed,
			TimeSpentMinutes: task.TimeSpentMinutes,
		}
		if task.VerificationStatus != nil {
			completionMeta.VerificationStatus = *task.VerificationStatus
		}
		if task.FilesChanged != nil && *task.FilesChanged != "" {
			if err := completionMeta.FromJSON(*task.FilesChanged); err == nil {
				resumeCtx.CompletionMeta = completionMeta
			}
		} else {
			resumeCtx.CompletionMeta = completionMeta
		}
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(resumeCtx)
	}

	printResumeContext(resumeCtx)
	return nil
}

// printResumeContext prints human-readable resume context
func printResumeContext(ctx *ResumeContext) {
	// Header
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Task Resume Context: %s\n", ctx.Task.Key)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	// Task Overview
	fmt.Printf("┌─ TASK OVERVIEW ─────────────────────────────────────────────\n")
	fmt.Printf("│ Title:    %s\n", ctx.Task.Title)
	fmt.Printf("│ Status:   %s\n", ctx.Task.Status)
	fmt.Printf("│ Priority: %d/10\n", ctx.Task.Priority)
	if ctx.Task.AgentType != nil {
		fmt.Printf("│ Agent:    %s\n", *ctx.Task.AgentType)
	}
	if ctx.Task.Description != nil && *ctx.Task.Description != "" {
		fmt.Printf("│\n│ Description:\n")
		lines := strings.Split(*ctx.Task.Description, "\n")
		for _, line := range lines {
			fmt.Printf("│   %s\n", line)
		}
	}
	if len(ctx.Dependencies) > 0 {
		fmt.Printf("│\n│ Dependencies:\n")
		for _, dep := range ctx.Dependencies {
			if dep != "" {
				fmt.Printf("│   - %s\n", dep)
			}
		}
	}
	fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")

	// Progress Section (from context data)
	if ctx.ContextData != nil && ctx.ContextData.Progress != nil {
		fmt.Printf("┌─ PROGRESS ──────────────────────────────────────────────────\n")
		if ctx.ContextData.Progress.CurrentStep != nil {
			fmt.Printf("│ ➤ CURRENT: %s\n│\n", *ctx.ContextData.Progress.CurrentStep)
		}
		if len(ctx.ContextData.Progress.CompletedSteps) > 0 {
			fmt.Printf("│ ✓ COMPLETED:\n")
			for _, step := range ctx.ContextData.Progress.CompletedSteps {
				fmt.Printf("│   • %s\n", step)
			}
			fmt.Printf("│\n")
		}
		if len(ctx.ContextData.Progress.RemainingSteps) > 0 {
			fmt.Printf("│ ☐ REMAINING:\n")
			for _, step := range ctx.ContextData.Progress.RemainingSteps {
				fmt.Printf("│   • %s\n", step)
			}
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Open Questions (highlighted)
	if ctx.ContextData != nil && len(ctx.ContextData.OpenQuestions) > 0 {
		fmt.Printf("┌─ ⚠ OPEN QUESTIONS ⚠ ───────────────────────────────────────\n")
		for i, q := range ctx.ContextData.OpenQuestions {
			fmt.Printf("│ %d. %s\n", i+1, q)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Blockers (highlighted)
	if ctx.ContextData != nil && len(ctx.ContextData.Blockers) > 0 {
		fmt.Printf("┌─ 🚧 BLOCKERS 🚧 ────────────────────────────────────────────\n")
		for _, b := range ctx.ContextData.Blockers {
			fmt.Printf("│ • %s\n", b.Description)
			fmt.Printf("│   Type: %s | Since: %s\n", b.BlockerType, b.BlockedSince.Format("2006-01-02 15:04"))
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Implementation Decisions
	if ctx.ContextData != nil && len(ctx.ContextData.ImplementationDecisions) > 0 {
		fmt.Printf("┌─ IMPLEMENTATION DECISIONS ──────────────────────────────────\n")
		for key, value := range ctx.ContextData.ImplementationDecisions {
			fmt.Printf("│ %s:\n│   %s\n", key, value)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Acceptance Criteria Status
	if ctx.ContextData != nil && len(ctx.ContextData.AcceptanceCriteriaStatus) > 0 {
		fmt.Printf("┌─ ACCEPTANCE CRITERIA ───────────────────────────────────────\n")
		for _, ac := range ctx.ContextData.AcceptanceCriteriaStatus {
			status := ac.Status
			symbol := "☐"
			switch status {
			case "complete":
				symbol = "✓"
			case "in_progress":
				symbol = "➤"
			case "failed":
				symbol = "✗"
			case "na":
				symbol = "–"
			}
			fmt.Printf("│ [%s] %s (%s)\n", symbol, ac.Criterion, status)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Work Sessions
	if len(ctx.WorkSessions) > 0 {
		fmt.Printf("┌─ WORK SESSIONS ─────────────────────────────────────────────\n")
		fmt.Printf("│ Total Sessions: %d\n", len(ctx.WorkSessions))
		if ctx.SessionStats != nil {
			fmt.Printf("│ Total Time:     %s\n", formatDuration(ctx.SessionStats.TotalDuration))
			if ctx.SessionStats.AverageDuration > 0 {
				fmt.Printf("│ Average:        %s\n", formatDuration(ctx.SessionStats.AverageDuration))
			}
		}
		if ctx.ActiveSession != nil {
			fmt.Printf("│\n│ ⏱ ACTIVE SESSION:\n")
			fmt.Printf("│   Started: %s\n", ctx.ActiveSession.StartedAt.Format("2006-01-02 15:04"))
			fmt.Printf("│   Duration: %s\n", formatDuration(ctx.ActiveSession.Duration()))
		}
		fmt.Printf("│\n│ Session History:\n")
		for i, session := range ctx.WorkSessions {
			if i >= 5 {
				fmt.Printf("│   ... (%d more sessions)\n", len(ctx.WorkSessions)-5)
				break
			}
			startTime := session.StartedAt.Format("01/02 15:04")
			var endTime, duration, outcome string
			if session.EndedAt.Valid {
				endTime = session.EndedAt.Time.Format("15:04")
				duration = formatDuration(session.Duration())
				if session.Outcome != nil {
					outcome = string(*session.Outcome)
				} else {
					outcome = "unknown"
				}
				fmt.Printf("│   %d. %s - %s (%s) → %s\n", i+1, startTime, endTime, duration, outcome)
			} else {
				fmt.Printf("│   %d. %s - active (%s)\n", i+1, startTime, formatDuration(session.Duration()))
			}
			if session.SessionNotes != nil && *session.SessionNotes != "" {
				fmt.Printf("│      Note: %s\n", *session.SessionNotes)
			}
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Recent Notes
	if len(ctx.Notes) > 0 {
		fmt.Printf("┌─ RECENT NOTES ──────────────────────────────────────────────\n")
		// Show last 5 notes
		start := 0
		if len(ctx.Notes) > 5 {
			start = len(ctx.Notes) - 5
			fmt.Printf("│ Showing last 5 of %d notes\n│\n", len(ctx.Notes))
		}
		for i := start; i < len(ctx.Notes); i++ {
			note := ctx.Notes[i]
			timestamp := note.CreatedAt.Format("2006-01-02 15:04")
			author := "unknown"
			if note.CreatedBy != nil {
				author = *note.CreatedBy
			}
			fmt.Printf("│ [%s] %s (%s):\n", timestamp, note.NoteType, author)
			lines := strings.Split(note.Content, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Printf("│   %s\n", line)
				}
			}
			if i < len(ctx.Notes)-1 {
				fmt.Printf("│\n")
			}
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Related Tasks
	if ctx.ContextData != nil && len(ctx.ContextData.RelatedTasks) > 0 {
		fmt.Printf("┌─ RELATED TASKS ─────────────────────────────────────────────\n")
		for _, taskKey := range ctx.ContextData.RelatedTasks {
			fmt.Printf("│ • %s\n", taskKey)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Completion Metadata (if completed)
	if ctx.CompletionMeta != nil {
		fmt.Printf("┌─ COMPLETION DETAILS ────────────────────────────────────────\n")
		if ctx.CompletionMeta.CompletedBy != nil {
			fmt.Printf("│ Completed By: %s\n", *ctx.CompletionMeta.CompletedBy)
		}
		fmt.Printf("│ Tests Passed: %t\n", ctx.CompletionMeta.TestsPassed)
		fmt.Printf("│ Verification: %s\n", ctx.CompletionMeta.VerificationStatus)
		if ctx.CompletionMeta.TimeSpentMinutes != nil {
			fmt.Printf("│ Time Spent:   %d minutes\n", *ctx.CompletionMeta.TimeSpentMinutes)
		}
		if len(ctx.CompletionMeta.FilesChanged) > 0 {
			fmt.Printf("│ Files Changed: %d\n", len(ctx.CompletionMeta.FilesChanged))
		}
		if ctx.CompletionMeta.CompletionNotes != nil && *ctx.CompletionMeta.CompletionNotes != "" {
			fmt.Printf("│\n│ Notes:\n│   %s\n", *ctx.CompletionMeta.CompletionNotes)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
	}

	// Next Steps (derived from context)
	fmt.Printf("┌─ NEXT STEPS ────────────────────────────────────────────────\n")
	if ctx.ContextData != nil && ctx.ContextData.Progress != nil {
		if ctx.ContextData.Progress.CurrentStep != nil {
			fmt.Printf("│ Continue: %s\n", *ctx.ContextData.Progress.CurrentStep)
		}
		if len(ctx.ContextData.Progress.RemainingSteps) > 0 {
			fmt.Printf("│ Then:\n")
			for i, step := range ctx.ContextData.Progress.RemainingSteps {
				if i < 3 {
					fmt.Printf("│   %d. %s\n", i+1, step)
				} else if i == 3 {
					fmt.Printf("│   ... and %d more steps\n", len(ctx.ContextData.Progress.RemainingSteps)-3)
					break
				}
			}
		}
	} else {
		switch ctx.Task.Status {
		case models.TaskStatus("todo"):
			fmt.Printf("│ Run: shark task start %s\n", ctx.Task.Key)
		case models.TaskStatus("in_progress"):
			fmt.Printf("│ Continue implementation\n")
			fmt.Printf("│ When done: shark task complete %s\n", ctx.Task.Key)
		case models.TaskStatus("ready_for_review"):
			fmt.Printf("│ Awaiting review\n│ To approve: shark task approve %s\n", ctx.Task.Key)
		case models.TaskStatus("completed"):
			fmt.Printf("│ Task completed\n")
		case models.TaskStatus("blocked"):
			fmt.Printf("│ Resolve blocker, then: shark task unblock %s\n", ctx.Task.Key)
		}
	}
	fmt.Printf("└─────────────────────────────────────────────────────────────\n\n")
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
