package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// displayOrchestratorAction displays the orchestrator action summary in human-readable format
func displayOrchestratorAction(action *config.PopulatedAction) {
	if action == nil {
		fmt.Println("Next Action: None configured")
		return
	}

	fmt.Println("\nNext Action:")
	fmt.Printf("  Type: %s\n", action.Action)
	if action.AgentType != "" {
		fmt.Printf("  Agent: %s\n", action.AgentType)
	}
	if len(action.Skills) > 0 {
		fmt.Printf("  Skills: %s\n", strings.Join(action.Skills, ", "))
	}
	fmt.Printf("\nInstruction: %s\n", action.Instruction)
}

// resolveTaskAction looks up the OrchestratorAction for a task's current status
// and populates the template with the task's data. Returns nil if no action is configured.
//
// This function uses TaskPlaceholdersWithRelated to include related documents and tasks.
// However, since this is in the CLI commands layer without access to repositories,
// it falls back to basic placeholders. Full integration with related docs/tasks
// happens in the service layer (e.g., TaskService) which has repository access.
//
// To implement the full feature with related docs/tasks visible to users:
// 1. Move orchestrator action resolution to a service that has repo access
// 2. Use TaskPlaceholdersWithRelated in the service instead
//
//nolint:unused // Used by task.go (implemented in T-E07-F28-001)
func resolveTaskAction(task *models.Task) *config.PopulatedAction {
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return nil
	}
	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
	wf := multi.GetWorkflowForLevel("task")
	// Use TaskPlaceholders (basic) since we don't have repository access in CLI layer.
	// Future: Move to service layer for TaskPlaceholdersWithRelated support
	return resolveAction(wf, string(task.Status), config.TaskPlaceholders(task))
}

// resolveEpicAction looks up the OrchestratorAction for an epic's current status
// and populates the template with the epic's data. Returns nil if no action is configured.
func resolveEpicAction(epic *models.Epic) *config.PopulatedAction {
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return nil
	}
	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
	wf := multi.GetWorkflowForLevel("epic")
	return resolveAction(wf, string(epic.Status), config.EpicPlaceholders(epic))
}

// resolveFeatureAction looks up the OrchestratorAction for a feature's current status
// and populates the template with the feature's data. Returns nil if no action is configured.
func resolveFeatureAction(feature *models.Feature) *config.PopulatedAction {
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return nil
	}
	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
	wf := multi.GetWorkflowForLevel("feature")
	return resolveAction(wf, string(feature.Status), config.FeaturePlaceholders(feature))
}

// resolveAction is the shared logic for looking up and populating an OrchestratorAction.
func resolveAction(wf *config.WorkflowConfig, status string, placeholders map[string]string) *config.PopulatedAction {
	if wf == nil {
		return nil
	}
	meta, found := wf.GetStatusMetadata(status)
	if !found || meta.OrchestratorAction == nil {
		return nil
	}
	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
	}
}
