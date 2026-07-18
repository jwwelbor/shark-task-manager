package sharkdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTC107_TC003_RolePullGuidanceUsesWorkflowAndClaimAuthorities verifies the
// installed public procedure keeps role selection and leasing in the existing
// SprintService and ClaimService ownership paths.
func TestTC107_TC003_RolePullGuidanceUsesWorkflowAndClaimAuthorities(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	pull, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "pull-by-role.md"))
	require.NoError(t, err)
	normalized := strings.Join(strings.Fields(string(pull)), " ")

	for _, want := range []string{
		"workflow-resolved `agent_type`",
		"SprintService.GetNextTask(ctx, agentType)",
		"shark sprint next --agent=<type>",
		"priority/dependency order",
		"ClaimService.Claim",
		"`/shark-rider run <selected-key>`",
		"`response.entity_key`",
		"claims or executes the returned `BacklogItemView` directly",
		"legacy `agent` assignment",
		"`model_tier`",
		"does not grant claim or status authority",
	} {
		assert.Contains(t, normalized, want)
	}
}

// TestTC108_ChildWorkerOwnershipGuidanceProtectsTheRoot verifies the installed
// worker handoff guidance permits bounded child work and evidence while keeping
// the root lease and workflow transition with the parent coordinator.
func TestTC108_ChildWorkerOwnershipGuidanceProtectsTheRoot(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	ownership, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "context", "worker-ownership.md"))
	require.NoError(t, err)
	normalized := strings.Join(strings.Fields(string(ownership)), " ")

	for _, want := range []string{
		"Rider parent owns the dispatched entity's lease",
		"workflow transition from selection through release",
		"bounded evidence and a semantic outcome",
		"semantic outcome and bounded evidence pointer",
		"status set",
		"force-claim",
		"rendered prompts",
		"credentials",
	} {
		assert.Contains(t, normalized, want)
	}
}

// TestTC110_RolePullGuidanceKeepsFallbackAndOrdinaryRunBoundaries verifies
// missing gates or team capability result in an explicit safe recommendation,
// never a silent replacement of ordinary Rider dispatch.
func TestTC110_RolePullGuidanceKeepsFallbackAndOrdinaryRunBoundaries(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	pull, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "pull-by-role.md"))
	require.NoError(t, err)

	normalized := strings.ToLower(strings.Join(strings.Fields(string(pull)), " "))
	for _, want := range []string{
		"missing product gates",
		"bootstrap or escalation",
		"explicit sequential fallback",
		"do not guess product decisions",
		"ordinary `/shark-rider run` routing",
	} {
		assert.Contains(t, normalized, strings.ToLower(want))
	}
}
