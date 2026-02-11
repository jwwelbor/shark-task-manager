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
