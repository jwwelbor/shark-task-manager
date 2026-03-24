package cli

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// workflowContainer holds the workflow service and its initialization state.
// Using a container struct makes ResetWorkflowService() safe: we swap the
// entire container atomically instead of reassigning individual sync.Once
// values (which would be a data race if any goroutine is mid-initialization).
type workflowContainer struct {
	svc      *workflow.Service
	initOnce sync.Once
}

// globalWorkflowContainer is accessed only through loadWorkflowContainer / storeWorkflowContainer.
// Using atomic pointer operations ensures that a call to ResetWorkflowService()
// is immediately visible to any goroutine that subsequently calls
// GetWorkflowService(), without requiring a separate mutex.
//
//nolint:gochecknoglobals // Intentional package-level singleton for CLI entry points.
var globalWorkflowContainer unsafe.Pointer // *workflowContainer

func init() {
	storeWorkflowContainer(new(workflowContainer))
}

func loadWorkflowContainer() *workflowContainer {
	return (*workflowContainer)(atomic.LoadPointer(&globalWorkflowContainer))
}

func storeWorkflowContainer(c *workflowContainer) {
	atomic.StorePointer(&globalWorkflowContainer, unsafe.Pointer(c))
}

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
	c := loadWorkflowContainer()
	c.initOnce.Do(func() {
		projectRoot, err := FindProjectRoot()
		if err != nil {
			// Fall back to current directory if project root detection fails
			projectRoot = "."
		}
		c.svc = workflow.NewService(projectRoot)
	})
	return c.svc
}

// ResetWorkflowService clears the cached workflow service.
// This is intended for testing only - DO NOT use in production code.
// It allows tests to reset state between test cases so that each test
// gets a fresh workflow service instance with its own configuration.
func ResetWorkflowService() {
	storeWorkflowContainer(new(workflowContainer))

	// EntityService depends on WorkflowService, so reset it too
	// to avoid returning a stale singleton built from the old workflow.
	resetEntityService()
}
