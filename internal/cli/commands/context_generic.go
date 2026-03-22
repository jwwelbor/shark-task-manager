package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// printContextData prints context data in human-readable format.
func printContextData(contextData *models.ContextData) {
	hasContent := false

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

	if len(contextData.ImplementationDecisions) > 0 {
		hasContent = true
		fmt.Println("Implementation Decisions:")
		for k, v := range contextData.ImplementationDecisions {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Println()
	}

	if len(contextData.OpenQuestions) > 0 {
		hasContent = true
		fmt.Println("Open Questions:")
		for _, q := range contextData.OpenQuestions {
			fmt.Printf("  - %s\n", q)
		}
		fmt.Println()
	}

	if len(contextData.Blockers) > 0 {
		hasContent = true
		fmt.Println("Blockers:")
		for _, b := range contextData.Blockers {
			fmt.Printf("  - %s (%s) - blocked since %s\n", b.Description, b.BlockerType, b.BlockedSince.Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	if !hasContent {
		fmt.Println("No context data available.")
	}
}

// makeContextCmd creates a "context" parent command with get/set/clear subcommands
// for the given entity type.
func makeContextCmd(entityName string) *cobra.Command {
	entityType := entityTypeFromName(entityName)

	ctxCmd := &cobra.Command{
		Use:   "context",
		Short: fmt.Sprintf("Manage %s context data", entityName),
		Long:  fmt.Sprintf("Get, set, and clear structured context data for %ss.", entityName),
	}

	setCmd := &cobra.Command{
		Use:   fmt.Sprintf("set <%s-key> --field <field> --value <value>", entityName),
		Short: fmt.Sprintf("Set a context field on %s", articleEntity(entityName)),
		Long: fmt.Sprintf(`Set or update a specific field in %s context data.

Supported fields:
  - current_step: String describing current work step
  - completed_steps: JSON array of completed steps
  - remaining_steps: JSON array of remaining steps
  - implementation_decisions: JSON object with decision key-value pairs
  - open_questions: JSON array of question strings
  - blockers: JSON array of blocker objects

Examples:
  shark %s context set <key> --field current_step --value "Working on implementation"
  shark %s context set <key> --field completed_steps --value '["Design","Review"]'`, entityName, entityName, entityName),
		Args: cobra.ExactArgs(1),
		RunE: makeRunContextSet(entityName, entityType),
	}
	setCmd.Flags().String("field", "", "Context field name (required)")
	setCmd.Flags().String("value", "", "Context field value (required)")
	_ = setCmd.MarkFlagRequired("field")
	_ = setCmd.MarkFlagRequired("value")

	getCmd := &cobra.Command{
		Use:   fmt.Sprintf("get <%s-key>", entityName),
		Short: fmt.Sprintf("Get %s context data", entityName),
		Long:  fmt.Sprintf("Display the current context data for %s.", articleEntity(entityName)),
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunContextGet(entityName, entityType),
	}

	clearCmd := &cobra.Command{
		Use:   fmt.Sprintf("clear <%s-key>", entityName),
		Short: fmt.Sprintf("Clear %s context data", entityName),
		Long:  fmt.Sprintf("Remove all context data from %s.", articleEntity(entityName)),
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunContextClear(entityName, entityType),
	}

	ctxCmd.AddCommand(setCmd)
	ctxCmd.AddCommand(getCmd)
	ctxCmd.AddCommand(clearCmd)
	return ctxCmd
}

func makeRunContextSet(entityName string, entityType models.EntityType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key := args[0]
		field, _ := cmd.Flags().GetString("field")
		value, _ := cmd.Flags().GetString("value")

		ctxSvc := cli.GetContextService()

		if err := ctxSvc.SetContextField(cmd.Context(), entityType, key, field, value); err != nil {
			return fmt.Errorf("failed to set context field for %s %s: %w", entityName, key, err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"entity_type": entityName,
				"entity_key":  key,
				"field":       field,
				"success":     true,
			})
		}

		cli.Success(fmt.Sprintf("Updated context field '%s' for %s %s", field, entityName, key))
		return nil
	}
}

func makeRunContextGet(entityName string, entityType models.EntityType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key := args[0]

		ctxSvc := cli.GetContextService()

		contextData, err := ctxSvc.GetContext(cmd.Context(), entityType, key)
		if err != nil {
			return fmt.Errorf("failed to get context for %s %s: %w", entityName, key, err)
		}

		if contextData == nil {
			contextData = &models.ContextData{}
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"entity_type":  entityName,
				"entity_key":   key,
				"context_data": contextData,
			})
		}

		displayName := articleEntity(entityName)
		displayName = string(displayName[0]-32) + displayName[1:] // capitalize "A" or "An"
		fmt.Printf("Context for %s %s\n\n", displayName, key)
		printContextData(contextData)
		return nil
	}
}

func makeRunContextClear(entityName string, entityType models.EntityType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key := args[0]

		ctxSvc := cli.GetContextService()

		if err := ctxSvc.ClearContext(cmd.Context(), entityType, key); err != nil {
			return fmt.Errorf("failed to clear context for %s %s: %w", entityName, key, err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"entity_type": entityName,
				"entity_key":  key,
				"success":     true,
			})
		}

		cli.Success(fmt.Sprintf("Cleared context data for %s %s", entityName, key))
		return nil
	}
}
