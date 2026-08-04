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

	// T-E38-F09-024 retired communicate.md, escalate.md, and execute.md in
	// favor of council.md (T-E38-F09-013) and coordinate.md/direct.md/
	// batch.md/execute-wave.md (T-E38-F09-023). This asserts their
	// successors alongside resume.md — six files, not a shrink from the
	// four retired/kept ones above.
	for _, name := range []string{"council.md", "coordinate.md", "direct.md", "batch.md", "execute-wave.md", "resume.md"} {
		content, readErr := os.ReadFile(filepath.Join(filepath.Dir(setupPath), name))
		require.NoErrorf(t, readErr, "installed shark-attack workflow %s must exist", name)
		assert.NotEmptyf(t, content, "installed shark-attack workflow %s must be documented", name)
	}
}

// TestTC105_SharkAttackCommunicationWorkflowKeepsBoundedInboxRules verifies
// the installed council workflow keeps the durable-artifact-before-ack
// contract and never grants lease or workflow authority.
//
// T-E38-F09-024 retired communicate.md; its bounded inbox and
// acknowledgement rule is carried forward verbatim by council.md
// (T-E38-F09-013), the file this test now reads.
func TestTC105_SharkAttackCommunicationWorkflowKeepsBoundedInboxRules(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	council, err := os.ReadFile(filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows", "council.md"))
	require.NoError(t, err)
	content := strings.Join(strings.Fields(string(council)), " ")
	for _, want := range []string{
		"docs/council/inbox/<member-id>/",
		"After acting, write the result as a bounded artifact",
		"Acknowledge or remove the inbox message only after the durable artifact is present",
		"Reuse an artifact ID only for byte-equivalent content",
		"does not claim work, release a root lease, or advance a root workflow state",
	} {
		assert.Contains(t, content, want)
	}
}

// TestTC008_EmbeddedDistributionRetainsExecutionGuidance verifies installation
// exposes the chair-led coordination procedure without adding a team command
// or runtime boundary.
//
// T-E38-F09-024 retired workflows/execute.md, whose single-file chair-led
// procedure is now split across T-E38-F09-023's coordinate.md/direct.md/
// batch.md/execute-wave.md by coordination level and topology (D-010). This
// reads their concatenation instead. The pull-by-role.md citation this test
// used to require is dropped per T-E38-F09-009's TC-007-02, which already
// owns the "no other corpus file sanctions pull-by-role.md as a normal path"
// assertion — duplicating it here would drift independently of that check.
func TestTC008_EmbeddedDistributionRetainsExecutionGuidance(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowRoot := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows")
	var content string
	for _, name := range []string{"coordinate.md", "direct.md", "batch.md", "execute-wave.md"} {
		body, readErr := os.ReadFile(filepath.Join(workflowRoot, name))
		require.NoErrorf(t, readErr, "read installed %s", name)
		content += string(body) + "\n"
	}
	for _, want := range []string{
		"`skills/shark-rider/verbs/run.md`",
		"`council.md`",
		"claim,",
		"release.",
	} {
		assert.Contains(t, content, want)
	}
	for _, forbidden := range []string{"shark team ", "provider configuration", "chair workflow authority"} {
		assert.NotContains(t, content, forbidden)
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

// TestTC104_TC109_ResumeProcedureLoadsBoundedDurableContext verifies the
// installed resume.md still pins the 7 load-bearing phrases from its
// original "load bounded durable context on worker refresh" procedure,
// after T-E38-F09-014's rewrite added the same-worker-follow-up /
// bounded-replacement / capability-discovery-ordering procedure ahead of
// it. This is a deliberate adjudication, not an oversight: council.md
// ("It also carries forward two procedures this restructure supersedes:
// the bounded council inbox and acknowledgement rule, and the
// material-question routing procedure") and context/message-schema.md's
// own "Resume work" section ("load the scoped decisions, handoffs,
// unresolved escalations, resolutions, and your inbox") both keep
// `docs/council/escalations/` and `docs/council/inbox/<member-id>/` alive
// under T-E38-F09-024's retirement of escalate.md/communicate.md — those
// two files retire, the directories and the durable-context-restore
// procedure that reads them do not. All 7 phrases are therefore still
// pinned here at the installed-bundle level (`Init(root)` +
// `os.ReadFile`), distinct from `tests/contracts`'s
// `sharkdata.ReadEmbedded`-based TC-010 coverage of the new procedure.
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
//
// T-E38-F09-024 retired workflows/escalate.md; council.md (T-E38-F09-013) is
// its successor and carries the same unresolved-material-question routing
// rule forward verbatim.
func TestTC106_TC110_EscalationAndFallbackProceduresPreserveReviewBoundaries(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowRoot := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "workflows")
	council, err := os.ReadFile(filepath.Join(workflowRoot, "council.md"))
	require.NoError(t, err)
	for _, want := range []string{
		"docs/product/escalation_triggers.md",
		"council-review",
		"pause/review",
		"fixed human destination",
		"unresolved",
	} {
		assert.Contains(t, string(council), want)
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
