package workflow

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canonicalWorkflowDir resolves the embedded canonical workflow directory by
// walking up from this test file. The path is internal/sharkdata/default_data/workflow.
func canonicalWorkflowDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: <repo>/internal/config/workflow/canonical_provider_test.go
	// target:   <repo>/internal/sharkdata/default_data/workflow
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "workflow")
}

// TestCanonicalWorkflows_SpawnAndResumeActionsSetProvider guards against the
// Phase 3 regression: every spawn_agent / check_or_resume action shipped in
// the canonical workflow YAMLs must declare a provider so `shark next`
// returns a populated provider field for AI-DLC dispatch.
func TestCanonicalWorkflows_SpawnAndResumeActionsSetProvider(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(canonicalWorkflowDir(t), "")
	if err != nil {
		t.Fatalf("failed to load canonical workflows: %v", err)
	}

	slots := map[string]any{
		"epic":     mlw.Epic,
		"feature":  mlw.Feature,
		"task":     mlw.Task,
		"bug":      mlw.Bug,
		"change":   mlw.Change,
		"sprint":   mlw.Sprint,
		"techDebt": mlw.TechDebt,
	}

	loaded := 0
	for slot, cfg := range slots {
		if cfg == nil {
			continue
		}
		wf, ok := cfg.(*WorkflowConfig)
		if !ok || wf == nil {
			continue
		}
		loaded++

		for status, meta := range wf.StatusMetadata {
			oa := meta.OrchestratorAction
			if oa == nil {
				continue
			}
			dispatches := oa.Action == "spawn_agent" || oa.Action == "check_or_resume"
			if !dispatches {
				continue
			}
			if strings.TrimSpace(oa.Provider) == "" {
				t.Errorf("canonical %s workflow: status %q action %q has empty provider — set provider (anthropic/openai/...) so `shark next` populates .provider",
					slot, status, oa.Action)
			}
		}
	}

	if loaded == 0 {
		t.Fatalf("no canonical workflows loaded from %s; check fixture layout", canonicalWorkflowDir(t))
	}
}

// TestCanonicalOpenAIWorkflowDefaultsDoNotPinModel keeps built-in Codex routes
// on the installed CLI's current default model rather than a stale static ID.
func TestCanonicalOpenAIWorkflowDefaultsDoNotPinModel(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(canonicalWorkflowDir(t), "")
	require.NoError(t, err, "failed to load canonical workflows")

	openAIRoutes := []struct {
		workflow string
		step     string
		config   *WorkflowConfig
	}{
		{workflow: "epic", step: "feature_review", config: mlw.Epic},
		{workflow: "epic", step: "integration_review", config: mlw.Epic},
		{workflow: "feature", step: "approval", config: mlw.Feature},
		{workflow: "feature", step: "task_review", config: mlw.Feature},
	}
	for _, route := range openAIRoutes {
		route := route
		t.Run(route.workflow+"/"+route.step, func(t *testing.T) {
			assertCanonicalOpenAIRouteDoesNotPinModel(t, route.workflow, route.step, route.config)
		})
	}
}

func assertCanonicalOpenAIRouteDoesNotPinModel(
	t *testing.T, workflowName, stepName string, config *WorkflowConfig,
) {
	t.Helper()
	if config == nil {
		assert.Fail(t, "canonical workflow is unavailable", "canonical %s workflow is unavailable", workflowName)
		return
	}
	step := config.Steps[stepName]
	if step == nil {
		assert.Fail(t, "canonical workflow route is missing", "canonical %s workflow is missing OpenAI route %q", workflowName, stepName)
		return
	}
	assert.Equal(t, action.ActionSpawnAgent, step.Action,
		"canonical %s workflow route %q action", workflowName, stepName)
	assert.Equal(t, "openai", step.Provider,
		"canonical %s workflow route %q provider", workflowName, stepName)
	assert.Empty(t, step.Model,
		"canonical %s workflow OpenAI route %q should not pin a model", workflowName, stepName)
}
