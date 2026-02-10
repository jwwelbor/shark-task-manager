package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// epicResumeCmd provides comprehensive context for resuming an epic
var epicResumeCmd = &cobra.Command{
	Use:   "resume <epic-key>",
	Short: "Get comprehensive context for resuming an epic",
	Long: `Get all context needed to resume work on an epic in a single command.

This includes:
  - Epic details (title, description, status, priority)
  - Context data (progress, decisions, questions, blockers)
  - Epic notes (chronologically ordered)
  - Feature summaries (with task counts and progress)
  - Task rollup (aggregate counts by status)

Examples:
  shark epic resume E07
  shark epic resume E07 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicResume,
}

func init() {
	epicCmd.AddCommand(epicResumeCmd)
}

func runEpicResume(cmd *cobra.Command, args []string) error {
	epicKey := args[0]

	// Get ResumeService
	resumeSvc, err := cli.GetResumeService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize resume service: %w", err)
	}

	resumeCtx, err := resumeSvc.GetEpicResume(cmd.Context(), epicKey)
	if err != nil {
		return fmt.Errorf("failed to get resume context for epic %s: %w", epicKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(resumeCtx)
	}

	printEpicResumeContext(resumeCtx)
	return nil
}

func printEpicResumeContext(ctx *services.EpicResumeContext) {
	// Header
	fmt.Printf("=================================================================\n")
	fmt.Printf("Epic Resume Context: %s\n", ctx.Epic.Key)
	fmt.Printf("=================================================================\n\n")

	// Epic Overview
	fmt.Printf("-- EPIC OVERVIEW ------------------------------------------------\n")
	fmt.Printf("  Title:    %s\n", ctx.Epic.Title)
	fmt.Printf("  Status:   %s\n", ctx.Epic.Status)
	fmt.Printf("  Priority: %s\n", ctx.Epic.Priority)
	if ctx.Epic.Description != nil && *ctx.Epic.Description != "" {
		fmt.Printf("\n  Description:\n")
		lines := strings.Split(*ctx.Epic.Description, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}
	}
	fmt.Println()

	// Progress Section (from context data)
	if ctx.ContextData != nil && ctx.ContextData.Progress != nil {
		fmt.Printf("-- PROGRESS -----------------------------------------------------\n")
		if ctx.ContextData.Progress.CurrentStep != nil {
			fmt.Printf("  Current: %s\n\n", *ctx.ContextData.Progress.CurrentStep)
		}
		if len(ctx.ContextData.Progress.CompletedSteps) > 0 {
			fmt.Printf("  Completed:\n")
			for _, step := range ctx.ContextData.Progress.CompletedSteps {
				fmt.Printf("    - %s\n", step)
			}
			fmt.Println()
		}
		if len(ctx.ContextData.Progress.RemainingSteps) > 0 {
			fmt.Printf("  Remaining:\n")
			for _, step := range ctx.ContextData.Progress.RemainingSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		fmt.Println()
	}

	// Open Questions
	if ctx.ContextData != nil && len(ctx.ContextData.OpenQuestions) > 0 {
		fmt.Printf("-- OPEN QUESTIONS -----------------------------------------------\n")
		for i, q := range ctx.ContextData.OpenQuestions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()
	}

	// Blockers
	if ctx.ContextData != nil && len(ctx.ContextData.Blockers) > 0 {
		fmt.Printf("-- BLOCKERS -----------------------------------------------------\n")
		for _, b := range ctx.ContextData.Blockers {
			fmt.Printf("  - %s (%s) since %s\n", b.Description, b.BlockerType, b.BlockedSince.Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	// Implementation Decisions
	if ctx.ContextData != nil && len(ctx.ContextData.ImplementationDecisions) > 0 {
		fmt.Printf("-- IMPLEMENTATION DECISIONS -------------------------------------\n")
		for key, value := range ctx.ContextData.ImplementationDecisions {
			fmt.Printf("  %s:\n    %s\n", key, value)
		}
		fmt.Println()
	}

	// Feature Summaries
	if len(ctx.Features) > 0 {
		fmt.Printf("-- FEATURES -----------------------------------------------------\n")
		for _, f := range ctx.Features {
			fmt.Printf("  %s: %s\n", f.Key, f.Title)
			fmt.Printf("    Status: %s | Tasks: %d | Progress: %.0f%%\n", f.Status, f.TaskCount, f.Progress)
		}
		fmt.Println()
	}

	// Task Rollup
	if ctx.TaskSummary != nil && ctx.TaskSummary.Total > 0 {
		fmt.Printf("-- TASK SUMMARY -------------------------------------------------\n")
		fmt.Printf("  Total Tasks: %d\n", ctx.TaskSummary.Total)
		for status, count := range ctx.TaskSummary.ByStatus {
			fmt.Printf("    %s: %d\n", status, count)
		}
		fmt.Println()
	}

	// Recent Notes
	if len(ctx.Notes) > 0 {
		fmt.Printf("-- RECENT NOTES -------------------------------------------------\n")
		start := 0
		if len(ctx.Notes) > 5 {
			start = len(ctx.Notes) - 5
			fmt.Printf("  Showing last 5 of %d notes\n\n", len(ctx.Notes))
		}
		for i := start; i < len(ctx.Notes); i++ {
			note := ctx.Notes[i]
			author := "unknown"
			if note.CreatedBy != nil {
				author = *note.CreatedBy
			}
			fmt.Printf("  [%s] %s (%s):\n", note.CreatedAt.Format("2006-01-02 15:04"), note.NoteType, author)
			lines := strings.Split(note.Content, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Printf("    %s\n", line)
				}
			}
			if i < len(ctx.Notes)-1 {
				fmt.Println()
			}
		}
		fmt.Println()
	}
}
