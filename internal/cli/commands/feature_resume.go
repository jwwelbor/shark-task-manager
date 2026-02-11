package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// featureResumeCmd provides comprehensive context for resuming a feature
var featureResumeCmd = &cobra.Command{
	Use:   "resume <feature-key>",
	Short: "Get comprehensive context for resuming a feature",
	Long: `Get all context needed to resume work on a feature in a single command.

This includes:
  - Feature details (title, description, status, progress)
  - Context data (progress, decisions, questions, blockers)
  - Feature notes (chronologically ordered)
  - Task summaries (with status and priority)
  - Task rollup (aggregate counts by status)

Examples:
  shark feature resume E07-F01
  shark feature resume E07-F01 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureResume,
}

func init() {
	featureCmd.AddCommand(featureResumeCmd)
}

func runFeatureResume(cmd *cobra.Command, args []string) error {
	featureKey := args[0]

	// Get ResumeService
	resumeSvc, err := cli.GetResumeService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize resume service: %w", err)
	}

	resumeCtx, err := resumeSvc.GetFeatureResume(cmd.Context(), featureKey)
	if err != nil {
		return fmt.Errorf("failed to get resume context for feature %s: %w", featureKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(resumeCtx)
	}

	printFeatureResumeContext(resumeCtx)
	return nil
}

func printFeatureResumeContext(ctx *services.FeatureResumeContext) {
	// Header
	fmt.Printf("=================================================================\n")
	fmt.Printf("Feature Resume Context: %s\n", ctx.Feature.Key)
	fmt.Printf("=================================================================\n\n")

	// Feature Overview
	fmt.Printf("-- FEATURE OVERVIEW ---------------------------------------------\n")
	fmt.Printf("  Title:    %s\n", ctx.Feature.Title)
	fmt.Printf("  Status:   %s\n", ctx.Feature.Status)
	fmt.Printf("  Progress: %.0f%%\n", ctx.Feature.ProgressPct)
	if ctx.Feature.Description != nil && *ctx.Feature.Description != "" {
		fmt.Printf("\n  Description:\n")
		lines := strings.Split(*ctx.Feature.Description, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}
	}
	fmt.Println()

	// Progress Section
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

	// Task Summaries
	if len(ctx.Tasks) > 0 {
		fmt.Printf("-- TASKS --------------------------------------------------------\n")
		for _, t := range ctx.Tasks {
			fmt.Printf("  %s: %s [%s] (P%d)\n", t.Key, t.Title, t.Status, t.Priority)
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
