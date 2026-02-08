package cli

import (
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

var (
	// globalWorkflowService holds the cached workflow service instance
	globalWorkflowService *workflow.Service

	// workflowInitOnce ensures the workflow service is initialized exactly once
	workflowInitOnce sync.Once
)

// GetWorkflowService returns the global workflow service, initializing it on first call.
// Uses FindProjectRoot() for project root auto-detection.
// Thread-safe via sync.Once.
//
// Unlike GetDB(), this function does not return an error because
// workflow.NewService always succeeds by falling back to default configuration
// when .sharkconfig.json is missing or invalid.
//
// Usage:
//
//	svc := cli.GetWorkflowService()
//	transitions := svc.GetValidTransitions(currentStatus)
func GetWorkflowService() *workflow.Service {
	workflowInitOnce.Do(func() {
		projectRoot, err := FindProjectRoot()
		if err != nil {
			// Fall back to current directory if project root detection fails
			projectRoot = "."
		}
		globalWorkflowService = workflow.NewService(projectRoot)
	})
	return globalWorkflowService
}

// ResetWorkflowService clears the cached workflow service.
// This is intended for testing only - DO NOT use in production code.
// It allows tests to reset state between test cases so that each test
// gets a fresh workflow service instance with its own configuration.
func ResetWorkflowService() {
	globalWorkflowService = nil
	workflowInitOnce = sync.Once{}
}
