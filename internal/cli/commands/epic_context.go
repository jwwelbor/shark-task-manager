package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// epicContextCmd represents the epic context command group
var epicContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage epic context data",
	Long: `Manage structured resume context data for epics.

Context data includes progress tracking, implementation decisions, open questions,
blockers, acceptance criteria status, and related tasks.

Examples:
  shark epic context set E07 --field current_step "Architecture review"
  shark epic context get E07
  shark epic context clear E07`,
}

// epicContextSetCmd sets a context field
var epicContextSetCmd = &cobra.Command{
	Use:   "set <epic-key>",
	Short: "Set or update epic context field",
	Long: `Set or update a specific field in epic context data.

Supported fields:
  - current_step: String describing current work step
  - completed_steps: JSON array of completed steps
  - remaining_steps: JSON array of remaining steps
  - implementation_decisions: JSON object with decision key-value pairs
  - open_questions: JSON array of question strings
  - blockers: JSON array of blocker objects
  - acceptance_criteria_status: JSON array of criterion objects
  - related_tasks: JSON array of task keys

Examples:
  shark epic context set E07 --field current_step "Defining epic scope"
  shark epic context set E07 --field completed_steps '["Research","Design"]'
  shark epic context set E07 --field open_questions '["What API version?"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicContextSet,
}

// epicContextGetCmd gets epic context
var epicContextGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic context data",
	Long:  `Display the current context data for an epic.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEpicContextGet,
}

// epicContextClearCmd clears epic context
var epicContextClearCmd = &cobra.Command{
	Use:   "clear <epic-key>",
	Short: "Clear epic context data",
	Long:  `Remove all context data from an epic.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEpicContextClear,
}

func runEpicContextSet(cmd *cobra.Command, args []string) error {
	epicKey := args[0]
	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Get ContextService (service layer pattern)
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeEpic, epicKey, field, value); err != nil {
		return fmt.Errorf("failed to set context field for epic %s: %w", epicKey, err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"epic_key": epicKey,
			"field":    field,
			"success":  true,
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Updated context field '%s' for epic %s", field, epicKey))
	return nil
}

func runEpicContextGet(cmd *cobra.Command, args []string) error {
	epicKey := args[0]

	// Get ContextService
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeEpic, epicKey)
	if err != nil {
		return fmt.Errorf("failed to get context for epic %s: %w", epicKey, err)
	}

	if contextData == nil {
		contextData = &models.ContextData{}
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"epic_key":     epicKey,
			"context_data": contextData,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output - reuse the same display pattern as task context
	fmt.Printf("Context for Epic %s\n\n", epicKey)
	printContextData(contextData)
	return nil
}

func runEpicContextClear(cmd *cobra.Command, args []string) error {
	epicKey := args[0]

	// Get ContextService
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeEpic, epicKey); err != nil {
		return fmt.Errorf("failed to clear context for epic %s: %w", epicKey, err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"epic_key": epicKey,
			"success":  true,
			"message":  "Context data cleared",
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Cleared context data for epic %s", epicKey))
	return nil
}

// printContextData prints human-readable context data (shared by epic and feature context commands)
func printContextData(contextData *models.ContextData) {
	hasContent := false

	// Progress
	if contextData.Progress != nil {
		hasContent = true
		fmt.Println("Progress:")
		if contextData.Progress.CurrentStep != nil {
			fmt.Printf("  Current Step: %s\n", *contextData.Progress.CurrentStep)
		}
		if len(contextData.Progress.CompletedSteps) > 0 {
			fmt.Println("  Completed Steps:")
			for _, step := range contextData.Progress.CompletedSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		if len(contextData.Progress.RemainingSteps) > 0 {
			fmt.Println("  Remaining Steps:")
			for _, step := range contextData.Progress.RemainingSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		fmt.Println()
	}

	// Implementation Decisions
	if len(contextData.ImplementationDecisions) > 0 {
		hasContent = true
		fmt.Println("Implementation Decisions:")
		for k, v := range contextData.ImplementationDecisions {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Println()
	}

	// Open Questions
	if len(contextData.OpenQuestions) > 0 {
		hasContent = true
		fmt.Println("Open Questions:")
		for _, q := range contextData.OpenQuestions {
			fmt.Printf("  - %s\n", q)
		}
		fmt.Println()
	}

	// Blockers
	if len(contextData.Blockers) > 0 {
		hasContent = true
		fmt.Println("Blockers:")
		for _, b := range contextData.Blockers {
			fmt.Printf("  - %s (%s) - blocked since %s\n", b.Description, b.BlockerType, b.BlockedSince.Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	// Acceptance Criteria
	if len(contextData.AcceptanceCriteriaStatus) > 0 {
		hasContent = true
		fmt.Println("Acceptance Criteria:")
		for _, ac := range contextData.AcceptanceCriteriaStatus {
			fmt.Printf("  [%s] %s\n", ac.Status, ac.Criterion)
		}
		fmt.Println()
	}

	// Related Tasks
	if len(contextData.RelatedTasks) > 0 {
		hasContent = true
		fmt.Println("Related Tasks:")
		for _, t := range contextData.RelatedTasks {
			fmt.Printf("  - %s\n", t)
		}
		fmt.Println()
	}

	if !hasContent {
		fmt.Println("No context data available.")
	}
}

func init() {
	// Add context subcommands to epic
	epicCmd.AddCommand(epicContextCmd)
	epicContextCmd.AddCommand(epicContextSetCmd)
	epicContextCmd.AddCommand(epicContextGetCmd)
	epicContextCmd.AddCommand(epicContextClearCmd)

	// Flags for set command
	epicContextSetCmd.Flags().String("field", "", "Context field to update (required)")
	epicContextSetCmd.Flags().String("value", "", "Field value (required)")
	_ = epicContextSetCmd.MarkFlagRequired("field")
	_ = epicContextSetCmd.MarkFlagRequired("value")
}
