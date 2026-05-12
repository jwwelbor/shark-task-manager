package workflow

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
