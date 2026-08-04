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
// installed compatibility-reference procedure still documents role selection
// in the existing SprintService ownership path (T-E38-F09-009: the direct
// `ClaimService.Claim` mechanic itself was adjudicated "sanctioned claim
// route → retire" and is checked separately below, confined to the
// historical/compatibility section rather than asserted as live guidance).
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
		"`/shark-rider run <selected-key>`",
		"`response.entity_key`",
		"claims or executes the returned `BacklogItemView` directly",
		"legacy `agent` assignment",
		"`model_tier`",
		"does not grant claim or status authority",
	} {
		assert.Contains(t, normalized, want)
	}

	// T-E38-F09-009: `ClaimService.Claim` (the retired direct-claim
	// mechanic) must survive only inside the clearly marked
	// historical/compatibility section of the installed file, never as
	// live guidance above it.
	marker := "Historical reference: worker-owned child mode (compatibility only)"
	idx := strings.Index(normalized, marker)
	require.Greater(t, idx, -1, "expected historical/compatibility marker in installed pull-by-role.md")
	assert.NotContains(t, normalized[:idx], "ClaimService.Claim")
	assert.Contains(t, normalized[idx:], "ClaimService.Claim")
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
// the fallback recommendations for missing gates or team capability survive
// as a compatibility reference. T-E38-F09-009 adjudicated all five of these
// phrases "sanctioned claim route → retire": they were specific to the
// retired direct-claim decision tree, so they must no longer read as live
// guidance for the sanctioned Rider re-entry path — they must appear only
// inside the clearly marked historical/compatibility section.
func TestTC110_RolePullGuidanceKeepsFallbackAndOrdinaryRunBoundaries(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	pull, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "pull-by-role.md"))
	require.NoError(t, err)

	normalized := strings.ToLower(strings.Join(strings.Fields(string(pull)), " "))
	marker := strings.ToLower("Historical reference: worker-owned child mode (compatibility only)")
	idx := strings.Index(normalized, marker)
	require.Greater(t, idx, -1, "expected historical/compatibility marker in installed pull-by-role.md")
	before, after := normalized[:idx], normalized[idx:]

	for _, want := range []string{
		"missing product gates",
		"bootstrap or escalation",
		"explicit sequential fallback",
		"do not guess product decisions",
		"ordinary `/shark-rider run` routing",
	} {
		lw := strings.ToLower(want)
		assert.NotContains(t, before, lw, "%q must not survive as live guidance outside the historical/compatibility section", want)
		assert.Contains(t, after, lw, "%q must survive as compatibility reference inside the historical/compatibility section", want)
	}
}
