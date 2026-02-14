package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
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

// NOTE: The orchestrator action resolution functions (resolveTaskAction, resolveEpicAction,
// resolveFeatureAction) have been moved to DisplayService in internal/services/display_service.go.
// This change enables the use of TaskPlaceholdersWithRelated, FeaturePlaceholdersWithRelated,
// and EpicPlaceholdersWithRelated which require repository access to populate {related_docs}
// and {related_tasks} template variables.
//
// Use DisplayService.ResolveTaskAction(), .ResolveFeatureAction(), or .ResolveEpicAction()
// instead of these removed CLI-layer functions.
