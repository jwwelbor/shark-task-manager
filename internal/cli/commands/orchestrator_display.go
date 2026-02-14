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
//nolint:unused // Used by task.go (implemented in T-E07-F28-001)
func resolveTaskAction(task *models.Task) *config.PopulatedAction {
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return nil
	}
	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
	wf := multi.GetWorkflowForLevel("task")
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
