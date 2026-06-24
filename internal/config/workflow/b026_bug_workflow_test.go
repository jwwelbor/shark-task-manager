// Regression tests for B026: shark status advance rejects valid bug transitions.
//
// When .sharkconfig.json's workflow_config points at a per-entity YAML directory
// (Shark 2.0 layout, e.g. "shark-data/workflow/"), the bug.yaml in that
// directory must drive the bug workflow. If LoadMultiLevelWorkflow silently
// falls back to DefaultBugWorkflow when the YAML loader fails (e.g. because
// one of the sibling entity YAML files contains a parse error), bug entities
// stored with statuses defined only in bug.yaml (such as "draft" or
// "ready_for_development") will be unrecognized, and `shark status advance`
// / `shark status transitions` will report "No valid transitions" even though
// the user-authored bug.yaml clearly defines them.
//
// These tests pin the contract:
//   - bug.yaml in the per-entity YAML directory is loaded into the Bug slot
//   - a parse error in one sibling YAML must NOT cause the bug YAML to be
//     silently dropped
package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bugYAMLWithDraft is a minimal bug.yaml that mirrors the shipping
// shark-data/workflow/bug.yaml — it starts at "draft" and transitions to
// "ready_for_development". The B026 symptom was that this status flow was
// not being loaded, causing the workflow service to report status "draft"
// as undefined.
const bugYAMLWithDraft = `version: '1.0'
status_flow:
  draft:
    - ready_for_development
    - cancelled
    - on_hold
  ready_for_development:
    - in_development
    - blocked
    - on_hold
  in_development:
    - ready_for_code_review
    - blocked
    - on_hold
  ready_for_code_review:
    - in_code_review
  in_code_review:
    - completed
  completed: []
  cancelled: []
  blocked:
    - ready_for_development
  on_hold:
    - ready_for_development
status_metadata:
  draft:
    description: Bug reported, not yet started
    phase: planning
    color: gray
  ready_for_development:
    description: Ready for bug reproduction and fix
    phase: development
    color: yellow
  in_development:
    description: Bug fix in progress
    phase: development
    color: yellow
  ready_for_code_review:
    description: Ready for code review
    phase: code_review
    color: magenta
  in_code_review:
    description: Code review in progress
    phase: code_review
    color: magenta
  completed:
    description: Bug fixed and verified
    phase: done
    color: green
  cancelled:
    description: Bug cancelled
    phase: done
    color: red
  blocked:
    description: Blocked
    phase: blocked
    color: red
  on_hold:
    description: On hold
    phase: paused
    color: gray
special_statuses:
  _start_:
    - draft
  _complete_:
    - completed
    - cancelled
`

// writeB026Config writes a .sharkconfig.json pointing workflow_config at the
// given workflow directory (relative to the config file's parent).
func writeB026Config(t *testing.T, configPath, workflowDir string) {
	t.Helper()
	cfg := map[string]interface{}{
		"workflow_config": workflowDir,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0o644))
}

// TestB026_BugYAMLDraftStatusIsLoaded verifies the basic happy path: when
// workflow_config points at a directory containing a bug.yaml that defines a
// "draft" status with valid transitions, those transitions must be reachable
// via the MultiLevelWorkflow / WorkflowConfig API used by the status service.
//
// This is the contract `shark status advance B###` (draft -> ready_for_development)
// depends on. If it fails, B026 has regressed.
func TestB026_BugYAMLDraftStatusIsLoaded(t *testing.T) {
	ClearWorkflowCache()
	defer ClearWorkflowCache()

	root := t.TempDir()
	workflowDir := filepath.Join(root, "shark-data", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "bug.yaml"),
		[]byte(bugYAMLWithDraft),
		0o644,
	))

	configPath := filepath.Join(root, ".sharkconfig.json")
	writeB026Config(t, configPath, "shark-data/workflow/")

	multi, err := LoadMultiLevelWorkflow(configPath)
	require.NoError(t, err, "LoadMultiLevelWorkflow must not error on a valid YAML layout")
	require.NotNil(t, multi)

	bugWf := multi.GetWorkflowForLevel("bug")
	require.NotNil(t, bugWf, "bug workflow must not be nil")

	// "draft" must be a defined status (presence in StatusFlow keys).
	transitions, ok := bugWf.StatusFlow["draft"]
	if !ok {
		t.Fatalf("bug workflow does not define 'draft' — silently fell back to DefaultBugWorkflow. "+
			"StatusFlow keys: %v", keysOf(bugWf.StatusFlow))
	}
	assert.Contains(t, transitions, "ready_for_development",
		"draft must transition to ready_for_development per bug.yaml")

	// ValidateTransition (the function backing `shark status advance`) must
	// accept draft -> ready_for_development without an error.
	require.NoError(t, ValidateTransition(bugWf, "draft", "ready_for_development"),
		"ValidateTransition must not reject the user-authored bug.yaml transition")

	// Sources tracking must show YAML-driven load, not "default".
	src := multi.Sources["bug"]
	assert.NotEqual(t, "default", src,
		"Sources['bug'] should reference the YAML path; 'default' means silent fallback (B026)")
}

// TestB026_PartialYAMLLoadOnSiblingError verifies that a parse error in one
// sibling entity YAML (e.g. change.yaml) must NOT cause bug.yaml to be silently
// discarded. The pre-fix behavior swallowed YAML loader errors via
// `if mlw, yamlErr := ...; yamlErr == nil && mlw != nil`, dropping a partially
// loaded MultiLevelWorkflow on any sibling failure.
//
// The fix keeps loaded slots populated when YAML errors occur for other files.
func TestB026_PartialYAMLLoadOnSiblingError(t *testing.T) {
	ClearWorkflowCache()
	defer ClearWorkflowCache()

	root := t.TempDir()
	workflowDir := filepath.Join(root, "shark-data", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))

	// Valid bug.yaml — should be loaded.
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "bug.yaml"),
		[]byte(bugYAMLWithDraft),
		0o644,
	))

	// Malformed change.yaml — must not block bug.yaml from loading.
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "change.yaml"),
		[]byte("this is: : not valid\nyaml: [unclosed"),
		0o644,
	))

	configPath := filepath.Join(root, ".sharkconfig.json")
	writeB026Config(t, configPath, "shark-data/workflow/")

	multi, _ := LoadMultiLevelWorkflow(configPath)
	// Whether LoadMultiLevelWorkflow returns an error here is acceptable
	// (surfacing the bad YAML is fine), but the returned MultiLevelWorkflow
	// — when non-nil — must still expose a usable bug workflow.
	require.NotNil(t, multi, "MultiLevelWorkflow must not be nil on partial YAML failure")

	bugWf := multi.GetWorkflowForLevel("bug")
	require.NotNil(t, bugWf)

	if _, ok := bugWf.StatusFlow["draft"]; !ok {
		t.Fatalf("bug workflow lost 'draft' status due to sibling YAML parse error — "+
			"this is the B026 regression. StatusFlow keys: %v", keysOf(bugWf.StatusFlow))
	}
}

// keysOf returns a sorted-by-insertion slice of map keys for diagnostic output.
func keysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
