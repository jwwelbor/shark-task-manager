package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func init() {
	epicCmd.AddCommand(makeNextStatusCmd("epic", func() nextStatusGetter {
		return cli.GetEpicService()
	}))
}

// GetWorkflowServiceForLevel is a helper to get a level-specific workflow service.
func GetWorkflowServiceForLevel(level string) *workflow.Service {
	return cli.GetWorkflowService().ForLevel(level)
}
