package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// featureContextCmd represents the feature context command group
var featureContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage feature context data",
	Long: `Manage structured resume context data for features.

Context data includes progress tracking, implementation decisions, open questions,
blockers, acceptance criteria status, and related tasks.

Examples:
  shark feature context set E07-F01 --field current_step "Implementing API"
  shark feature context get E07-F01
  shark feature context clear E07-F01`,
}

// featureContextSetCmd sets a context field
var featureContextSetCmd = &cobra.Command{
	Use:   "set <feature-key>",
	Short: "Set or update feature context field",
	Long: `Set or update a specific field in feature context data.

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
  shark feature context set E07-F01 --field current_step "Implementing auth endpoint"
  shark feature context set E07-F01 --field completed_steps '["Design","DB Schema"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureContextSet,
}

// featureContextGetCmd gets feature context
var featureContextGetCmd = &cobra.Command{
	Use:   "get <feature-key>",
	Short: "Get feature context data",
	Long:  `Display the current context data for a feature.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureContextGet,
}

// featureContextClearCmd clears feature context
var featureContextClearCmd = &cobra.Command{
	Use:   "clear <feature-key>",
	Short: "Clear feature context data",
	Long:  `Remove all context data from a feature.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureContextClear,
}

func runFeatureContextSet(cmd *cobra.Command, args []string) error {
	featureKey := args[0]
	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Get ContextService (service layer pattern)
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeFeature, featureKey, field, value); err != nil {
		return fmt.Errorf("failed to set context field for feature %s: %w", featureKey, err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"feature_key": featureKey,
			"field":       field,
			"success":     true,
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Updated context field '%s' for feature %s", field, featureKey))
	return nil
}

func runFeatureContextGet(cmd *cobra.Command, args []string) error {
	featureKey := args[0]

	// Get ContextService
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeFeature, featureKey)
	if err != nil {
		return fmt.Errorf("failed to get context for feature %s: %w", featureKey, err)
	}

	if contextData == nil {
		contextData = &models.ContextData{}
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"feature_key":  featureKey,
			"context_data": contextData,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output - reuse the shared display function from epic_context.go
	fmt.Printf("Context for Feature %s\n\n", featureKey)
	printContextData(contextData)
	return nil
}

func runFeatureContextClear(cmd *cobra.Command, args []string) error {
	featureKey := args[0]

	// Get ContextService
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeFeature, featureKey); err != nil {
		return fmt.Errorf("failed to clear context for feature %s: %w", featureKey, err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"feature_key": featureKey,
			"success":     true,
			"message":     "Context data cleared",
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Cleared context data for feature %s", featureKey))
	return nil
}

func init() {
	// Add context subcommands to feature
	featureCmd.AddCommand(featureContextCmd)
	featureContextCmd.AddCommand(featureContextSetCmd)
	featureContextCmd.AddCommand(featureContextGetCmd)
	featureContextCmd.AddCommand(featureContextClearCmd)

	// Flags for set command
	featureContextSetCmd.Flags().String("field", "", "Context field to update (required)")
	featureContextSetCmd.Flags().String("value", "", "Field value (required)")
	_ = featureContextSetCmd.MarkFlagRequired("field")
	_ = featureContextSetCmd.MarkFlagRequired("value")
}
