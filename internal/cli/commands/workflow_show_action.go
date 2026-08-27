package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// workflowShowActionCmd previews the populated orchestrator action for an entity at a status
var workflowShowActionCmd = &cobra.Command{
	Use:   "show-action <entity-key> <status>",
	Short: "Preview orchestrator action for an entity at a status",
	Long: `Preview the fully populated orchestrator action for a specific entity at a given status.

Looks up the entity from the database, loads the workflow configuration for the
entity's level, and populates the action template with real entity data (key, title,
description, etc.). Use this to verify exactly what instruction an agent will receive
before it runs.

Entity type is auto-detected from the key format:
  E##          → epic
  E##-F##      → feature
  E##-F##-###  → task
  B###         → bug
  CC-###       → change-card

Examples:
  shark admin workflow show-action E07-F01-001 ready_for_development
  shark admin workflow show-action E07-F01 ready_for_feature_review
  shark admin workflow show-action E07 in_progress
  shark admin workflow show-action E07-F01-001 ready_for_development --json`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkflowShowAction,
}

func init() {
	workflowCmd.AddCommand(workflowShowActionCmd)
}

// showActionResult is the JSON output structure
type showActionResult struct {
	EntityKey  string                  `json:"entity_key"`
	EntityType string                  `json:"entity_type"`
	Status     string                  `json:"status"`
	Action     *action.PopulatedAction `json:"action"`
}

func runWorkflowShowAction(cmd *cobra.Command, args []string) error {
	entityKey := args[0]
	status := args[1]

	// Detect entity type from key format
	entityType := DetectEntityType(entityKey)
	if entityType == "unknown" {
		return fmt.Errorf("cannot detect entity type from key %q", entityKey)
	}

	// Map entity type to workflow level
	level := entityType
	if level == "change_card" {
		level = "change"
	}

	// Load workflow config for this level
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	multiWorkflow := config.LoadMultiLevelWorkflowOrDefault(configPath)
	wfConfig := multiWorkflow.GetWorkflowForLevel(level)

	// Look up the status metadata and its orchestrator action
	metadata, exists := wfConfig.StatusMetadata[status]
	if !exists {
		return fmt.Errorf("status %q not found in %s workflow configuration", status, level)
	}

	if metadata.OrchestratorAction == nil {
		result := &showActionResult{
			EntityKey:  entityKey,
			EntityType: entityType,
			Status:     status,
			Action:     nil,
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		fmt.Printf("Entity:  %s (%s)\n", entityKey, entityType)
		fmt.Printf("Status:  %s\n", status)
		fmt.Printf("\nNo orchestrator action defined for this status.\n")
		return nil
	}

	// Look up entity from database to populate template placeholders
	placeholders, lookupErr := lookupEntityPlaceholders(cmd.Context(), entityKey, entityType)
	if lookupErr != nil {
		return fmt.Errorf("failed to look up entity %s: %w", entityKey, lookupErr)
	}

	// Populate the action template
	populated := metadata.OrchestratorAction.ToPopulatedAction(placeholders)

	result := &showActionResult{
		EntityKey:  entityKey,
		EntityType: entityType,
		Status:     status,
		Action:     populated,
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	// Human-readable output
	fmt.Printf("Entity:      %s (%s)\n", entityKey, entityType)
	fmt.Printf("Status:      %s\n", status)
	fmt.Printf("Action:      %s\n", populated.Action)
	if populated.AgentType != "" {
		fmt.Printf("Agent Type:  %s\n", populated.AgentType)
	}
	if populated.Provider != "" {
		fmt.Printf("Provider:    %s\n", populated.Provider)
	}
	if populated.Model != "" {
		fmt.Printf("Model:       %s\n", populated.Model)
	}
	if len(populated.Skills) > 0 {
		fmt.Printf("Skills:      %s\n", strings.Join(populated.Skills, ", "))
	}
	if populated.Instruction != "" {
		fmt.Printf("\nInstruction:\n  %s\n", populated.Instruction)
	}

	return nil
}

// lookupEntityPlaceholders fetches an entity from the database and returns its template placeholders.
func lookupEntityPlaceholders(ctx context.Context, key string, entityType string) (map[string]string, error) {
	// fallbackPlaceholders returns minimal placeholders when an entity isn't found in the DB.
	// This allows show-action to still render the template with at least the key populated.
	fallbackPlaceholders := func(key string) map[string]string {
		return map[string]string{
			"id":  key,
			"key": key,
		}
	}

	switch entityType {
	case "task":
		taskKey, err := NormalizeTaskKey(key)
		if err != nil {
			return nil, fmt.Errorf("invalid task key: %w", err)
		}
		svc := cli.GetTaskService()
		task, err := svc.GetTask(ctx, taskKey)
		if err != nil || task == nil {
			return fallbackPlaceholders(taskKey), nil
		}
		return config.TaskPlaceholders(task), nil

	case "feature":
		svc := cli.GetFeatureService()
		feature, err := svc.GetFeature(ctx, key)
		if err != nil || feature == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.FeaturePlaceholders(feature), nil

	case "epic":
		svc := cli.GetEpicService()
		epic, err := svc.GetEpic(ctx, key)
		if err != nil || epic == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.EpicPlaceholders(epic), nil

	case "bug":
		svc := cli.GetBugService()
		bug, err := svc.GetBug(ctx, key)
		if err != nil || bug == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.BugPlaceholders(bug), nil

	case "change", "change_card":
		svc := cli.GetChangeCardService()
		card, err := svc.GetChangeCard(ctx, key)
		if err != nil || card == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.ChangeCardPlaceholders(card), nil

	case "tech_debt":
		svc := cli.GetTechDebtService()
		td, err := svc.GetTechDebt(ctx, key)
		if err != nil || td == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.TechDebtPlaceholders(td), nil

	case "question":
		question, err := getQuestionService().GetQuestion(ctx, key)
		if err != nil || question == nil {
			return fallbackPlaceholders(key), nil
		}
		state, err := models.DecodeQuestionState(question.ContextData)
		if err != nil {
			return nil, fmt.Errorf("decode Question state for placeholders: %w", err)
		}
		if state == nil || state.CurrentResponder() == "" {
			return nil, fmt.Errorf("Question %s has no current responder", key)
		}
		placeholders := config.EntityPlaceholders(question)
		placeholders["summary"] = question.Summary
		placeholders["requester"] = question.Requester
		placeholders["current_responder"] = state.CurrentResponder()
		return placeholders, nil

	case "sprint":
		svc := cli.GetSprintService()
		sprint, err := svc.GetSprint(ctx, key)
		if err != nil || sprint == nil {
			return fallbackPlaceholders(key), nil
		}
		return config.EntityPlaceholders(sprint), nil

	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}
