package sharkdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTC103_SharkAttackSetupInstallsProtocolWorkflows verifies the embedded
// bundle supplies the setup and privacy guidance needed after installation.
func TestTC103_SharkAttackSetupInstallsProtocolWorkflows(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	setupPath := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "setup.md")
	setup, err := os.ReadFile(setupPath)
	require.NoError(t, err)
	assert.Contains(t, string(setup), "admin install-shark-data")
	assert.Contains(t, string(setup), "docs/council/")
	assert.Contains(t, string(setup), ".gitignore")
	assert.Contains(t, string(setup), "overrides/skills/")
	assert.Contains(t, string(setup), "shark-attack/")

	for _, name := range []string{"communicate.md", "escalate.md", "resume.md"} {
		content, readErr := os.ReadFile(filepath.Join(filepath.Dir(setupPath), name))
		require.NoErrorf(t, readErr, "installed shark-attack workflow %s must exist", name)
		assert.NotEmptyf(t, content, "installed shark-attack workflow %s must be documented", name)
	}
}

// TestTC103_CouncilProjectLayoutDocumentsPrivateContinuity verifies the
// project-level, durable layout is versioned without requiring private content.
func TestTC103_CouncilProjectLayoutDocumentsPrivateContinuity(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	councilRoot := filepath.Join(projectRoot, "docs", "council")

	readme, err := os.ReadFile(filepath.Join(councilRoot, "README.md"))
	require.NoError(t, err)
	for _, want := range []string{
		"decisions/",
		"handoffs/",
		"escalations/",
		"inbox/<member-id>/",
		"private",
		"refreshed worker",
	} {
		assert.Contains(t, string(readme), want)
	}

	for _, marker := range []string{
		"decisions/.gitkeep",
		"handoffs/.gitkeep",
		"escalations/.gitkeep",
		"inbox/.gitkeep",
	} {
		_, statErr := os.Stat(filepath.Join(councilRoot, filepath.FromSlash(marker)))
		assert.NoErrorf(t, statErr, "council layout marker %s must exist", marker)
	}
}

// TestTC104_TC109_ResumeProcedureLoadsBoundedDurableContext verifies a
// refreshed worker is directed to every durable source before its inbox.
func TestTC104_TC109_ResumeProcedureLoadsBoundedDurableContext(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	resume, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "resume.md"))
	require.NoError(t, err)
	for _, want := range []string{
		"docs/council/decisions/",
		"docs/council/handoffs/",
		"docs/council/escalations/",
		"docs/council/inbox/<member-id>/",
		"acknowledge",
		"bounded paths and metadata",
		"rendered prompts",
	} {
		assert.Contains(t, string(resume), want)
	}
}

// TestTC106_TC110_EscalationAndFallbackProceduresPreserveReviewBoundaries
// verifies that the protocol routes an unresolved material question to council
// review and does not silently change normal Rider routing.
func TestTC106_TC110_EscalationAndFallbackProceduresPreserveReviewBoundaries(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowRoot := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows")
	escalate, err := os.ReadFile(filepath.Join(workflowRoot, "escalate.md"))
	require.NoError(t, err)
	for _, want := range []string{
		"docs/product/escalation_triggers.md",
		"council-review",
		"pause/review",
		"fixed human destination",
		"unresolved",
	} {
		assert.Contains(t, string(escalate), want)
	}

	setup, err := os.ReadFile(filepath.Join(workflowRoot, "setup.md"))
	require.NoError(t, err)
	normalizedSetup := strings.ToLower(strings.Join(strings.Fields(string(setup)), " "))
	for _, want := range []string{
		"bootstrap",
		"sequential fallback",
		"ordinary `/shark-rider run` routing",
		"do not guess product decisions",
	} {
		assert.Contains(t, normalizedSetup, want)
	}
}
