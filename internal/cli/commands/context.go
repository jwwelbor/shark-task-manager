package commands

import (
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// contextCmd is the top-level smart dispatcher for context operations.
// It auto-detects entity type from the key format (E07=epic, E07-F01=feature, E07-F01-001=task).
var contextCmd = &cobra.Command{
	Use:     "context <key>",
	Short:   "Get or manage entity context data",
	GroupID: "manage",
	Long: `Get or manage structured resume context data for any entity.
Entity type is auto-detected from the key format.

When called with just a key, displays the context data (equivalent to "shark context get <key>").

Subcommands:
  get     Get context data (default when no subcommand)
  set     Set or update a context field
  clear   Clear all context data

Key Formats:
  E07                Epic
  E07-F01            Feature
  E07-F01-001        Task

Examples:
  shark context E07                                    Get epic context
  shark context E07-F01                                Get feature context
  shark context E07-F01-001                            Get task context
  shark context get E07-F01-001                        Same as above (explicit)
  shark context set E07-F01-001 --field current_step --value "Implementing API"
  shark context clear E07-F01-001                      Clear all context data
  shark context E07-F01-001 --json                     JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runContextGet,
}

// contextGetCmd explicitly gets context data for an entity.
var contextGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get entity context data",
	Long: `Display the current context data for an entity. Entity type is auto-detected from key format.

Examples:
  shark context get E07              Get epic context
  shark context get E07-F01          Get feature context
  shark context get E07-F01-001      Get task context
  shark context get E07-F01-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runContextGet,
}

// contextSetCmd sets a context field on any entity.
var contextSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set or update entity context field",
	Long: `Set or update a specific field in entity context data. Entity type is auto-detected from key format.

Supported fields:
  - current_step: String describing current work step
  - completed_steps: JSON array of completed steps
  - remaining_steps: JSON array of remaining steps
  - implementation_decisions: JSON object with decision key-value pairs
  - open_questions: JSON array of question strings
  - blockers: JSON array of blocker objects
  - acceptance_criteria_status: JSON array of criterion objects

Examples:
  shark context set E07-F01-001 --field current_step --value "Implementing API endpoint"
  shark context set E07-F01 --field completed_steps --value '["Step 1","Step 2"]'
  shark context set E07 --field open_questions --value '["What API version?"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runContextSet,
}

// contextClearCmd clears all context data from an entity.
var contextClearCmd = &cobra.Command{
	Use:   "clear <key>",
	Short: "Clear entity context data",
	Long: `Remove all context data from an entity. Entity type is auto-detected from key format.

Examples:
  shark context clear E07-F01-001    Clear task context
  shark context clear E07-F01        Clear feature context
  shark context clear E07            Clear epic context`,
	Args: cobra.ExactArgs(1),
	RunE: runContextClear,
}

func init() {
	// Register context command with root
	cli.RootCmd.AddCommand(contextCmd)

	// Add subcommands
	contextCmd.AddCommand(contextGetCmd)
	contextCmd.AddCommand(contextSetCmd)
	contextCmd.AddCommand(contextClearCmd)

	// Flags for set command
	contextSetCmd.Flags().String("field", "", "Context field to update (required)")
	contextSetCmd.Flags().String("value", "", "Field value (required)")
	_ = contextSetCmd.MarkFlagRequired("field")
	_ = contextSetCmd.MarkFlagRequired("value")
}

// runContextGet handles both `shark context <key>` and `shark context get <key>`.
func runContextGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse key and detect entity type
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	modelEntityType, err := toModelEntityType(entityType)
	if err != nil {
		return err
	}

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), modelEntityType, key)
	if err != nil {
		cli.Error(fmt.Sprintf("%s %s not found", capitalizeEntityType(entityType), key))
		os.Exit(1)
	}

	if contextData == nil {
		contextData = &models.ContextData{}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"entity_type":  entityType,
			"entity_key":   key,
			"context_data": contextData,
		}
		return cli.OutputJSON(output)
	}

	fmt.Printf("Context for %s %s\n\n", capitalizeEntityType(entityType), key)
	printContextData(contextData)
	return nil
}

// runContextSet handles `shark context set <key> --field <field> --value <value>`.
func runContextSet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse key and flags
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	modelEntityType, err := toModelEntityType(entityType)
	if err != nil {
		return err
	}

	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), modelEntityType, key, field, value); err != nil {
		return fmt.Errorf("failed to set context field for %s %s: %w", entityType, key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"entity_type": entityType,
			"entity_key":  key,
			"field":       field,
			"success":     true,
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Updated context field '%s' for %s %s", field, entityType, key))
	return nil
}

// runContextClear handles `shark context clear <key>`.
func runContextClear(cmd *cobra.Command, args []string) error {
	// Step 1: Parse key and detect entity type
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	modelEntityType, err := toModelEntityType(entityType)
	if err != nil {
		return err
	}

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.ClearContext(cmd.Context(), modelEntityType, key); err != nil {
		return fmt.Errorf("failed to clear context for %s %s: %w", entityType, key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"entity_type": entityType,
			"entity_key":  key,
			"success":     true,
			"message":     "Context data cleared",
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Cleared context data for %s %s", entityType, key))
	return nil
}

// toModelEntityType converts a string entity type from ParseGetArgs to models.EntityType.
func toModelEntityType(entityType string) (models.EntityType, error) {
	switch entityType {
	case "epic":
		return models.EntityTypeEpic, nil
	case "feature":
		return models.EntityTypeFeature, nil
	case "task":
		return models.EntityTypeTask, nil
	default:
		return "", fmt.Errorf("unsupported entity type: %s", entityType)
	}
}
