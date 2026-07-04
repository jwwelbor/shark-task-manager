package workflow

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// loadEmbeddedWorkflow loads the canonical route-based workflow for the given
// entity slot ("epic", "feature", "task", "sprint", "bug", "change",
// "tech_debt") from the binary's embedded shark-data/workflow/ tree.
//
// Returns a fresh parse on every call (matches the historical fresh-instance
// contract of the Default*Workflow() factories) — callers such as
// WorkflowConfig.DeriveLegacy() mutate the returned config in place, so a
// shared/cached instance would leak mutations across callers.
//
// A missing or unparsable embedded file is a build invariant violation: every
// entity slot ships a YAML file under internal/sharkdata/default_data/workflow/,
// so failure here means the binary was built without its data bundle.
func loadEmbeddedWorkflow(entityType string) *WorkflowConfig {
	filename := EmbeddedWorkflowFilename(entityType)
	if filename == "" {
		panic(fmt.Sprintf("corrupt binary: no embedded workflow filename registered for entity type %q", entityType))
	}
	relPath := YAMLWorkflowDir + "/" + filename
	data, err := sharkdata.ReadEmbedded(relPath)
	if err != nil {
		panic(fmt.Sprintf("corrupt binary: embedded workflow %q failed to read: %v", relPath, err))
	}
	cfg, err := ParseWorkflowYAMLBytes(data, "embedded:"+relPath)
	if err != nil {
		panic(fmt.Sprintf("corrupt binary: embedded workflow %q failed to parse: %v", relPath, err))
	}
	return cfg
}

// DefaultWorkflow returns the embedded canonical task workflow (Shark 2.x,
// route-based). Used when no on-disk workflow YAML or inline config block is
// present for the "task" entity slot.
func DefaultWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("task")
}

// DefaultEpicWorkflow returns the embedded canonical epic workflow.
func DefaultEpicWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("epic")
}

// DefaultBugWorkflow returns the embedded canonical bug workflow.
func DefaultBugWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("bug")
}

// DefaultChangeCardWorkflow returns the embedded canonical change-card workflow.
func DefaultChangeCardWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("change")
}

// DefaultTechDebtWorkflow returns the embedded canonical tech-debt workflow.
func DefaultTechDebtWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("tech_debt")
}

// DefaultFeatureWorkflow returns the embedded canonical feature workflow.
func DefaultFeatureWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("feature")
}

// DefaultSprintWorkflow returns the embedded canonical sprint workflow.
func DefaultSprintWorkflow() *WorkflowConfig {
	return loadEmbeddedWorkflow("sprint")
}
