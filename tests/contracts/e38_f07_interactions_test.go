// Package contracts exercises the published execution seams consumed by E38-F07.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

func readF07RepositoryFile(t *testing.T, relPath string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(content)
}

func readF07EmbeddedFile(t *testing.T, relPath string) string {
	t.Helper()
	content, err := sharkdata.ReadEmbedded(relPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", relPath, err)
	}
	return string(content)
}

func normalizeF07Content(content string) string {
	return strings.Join(strings.Fields(content), " ")
}

// TestTC001_X01DispatchPromptFidelity protects the only supported worker
// dispatch seam: the prompt returned by shark next is passed verbatim.
func TestTC001_X01DispatchPromptFidelity(t *testing.T) {
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))
	for _, want := range []string{
		"shark next {KEY} --json",
		"Prompt: exactly `response.prompt`",
		"Do not build prompts from `shark get ... orchestrator_action`",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("Rider dispatch contract omits %q", want)
		}
	}
}

// TestTC002_X02ConfiguredOutcomeAndStopBoundaries preserves configured
// outcomes and terminal action handling instead of fabricated statuses.
func TestTC002_X02ConfiguredOutcomeAndStopBoundaries(t *testing.T) {
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))
	for _, want := range []string{
		"### `pause`", "### `archive`", "### `error`",
		"Do not retry blindly.", "workflow-configured",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("Rider outcome contract omits %q", want)
		}
	}
}

// TestTC003_ParentOwnsDispatchedEntityMutation forbids direct state mutation
// by a Rider-dispatched worker. The parent persists the worker's bounded
// evidence and directives after the worker returns.
func TestTC003_ParentOwnsDispatchedEntityMutation(t *testing.T) {
	worker := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/context/task-execution-pattern.md"))
	for _, want := range []string{
		"never claims, heartbeats, releases, or transitions",
		"does not mutate lease or workflow state for the dispatched entity",
		"bounded evidence and parent-persistence directives",
		"The parent persists those directives",
		"write bounded notes and context",
	} {
		if !strings.Contains(worker, want) {
			t.Errorf("worker contract omits %q", want)
		}
	}

	drivenStart := strings.Index(worker, "## The universal pattern (driven agent)")
	manualStart := strings.Index(worker, "Running an entity **by hand**")
	if drivenStart == -1 || manualStart == -1 || manualStart <= drivenStart {
		t.Fatal("worker contract must delimit Rider-driven and manual execution modes")
	}
	driven := worker[drivenStart:manualStart]
	for _, forbidden := range []string{
		"shark context set",
		"shark create note",
		"shark note add",
		"set context, or create notes on its own",
		"Do not write progress, context, notes, blockers, or any other Shark state",
	} {
		if strings.Contains(driven, forbidden) {
			t.Errorf("Rider-dispatched worker instructions must not issue %q; return a parent-persistence directive instead", forbidden)
		}
	}
}

// TestTC004_FailureMissingOutcomeAndKickbackOrdering keeps parent recovery
// explicit instead of turning partial work into a fabricated success.
func TestTC004_FailureMissingOutcomeAndKickbackOrdering(t *testing.T) {
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))
	for _, want := range []string{
		"No outcome at all → treat as `blocked` and record a blocker note",
		"Apply task kickbacks.",
		"BEFORE advancing the parent",
		"A `fail` outcome whose response names no kickbacks is suspicious",
		"Release the lease, always",
		"every success, failure, or exception path",
		"If the worker fails or throws, still release the lease",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("Rider recovery contract omits %q", want)
		}
	}
}

// TestTC005_RoleAwareSelfPullReentersCanonicalRiderDispatch prevents the F04
// worker-owned child mode and a sprint-selection result from being composed
// into the F07 Rider loop. A role-aware self-pull may select by workflow role,
// but must reenter /shark-rider run so shark next supplies the canonical
// prompt and metadata before the Rider claims a concrete entity.
func TestTC005_RoleAwareSelfPullReentersCanonicalRiderDispatch(t *testing.T) {
	execute := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/execute.md"))
	pullByRole := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/pull-by-role.md"))
	workerOwnership := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/context/worker-ownership.md"))
	skill := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/SKILL.md"))
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))

	for _, want := range []string{
		"workflow-resolved role selection",
		"Roster membership, responsibility prose, legacy assignment, and `model_tier` never grant authority.",
		"`shark sprint next --agent=<type>` only to select `selected-key`",
		"invoke `/shark-rider run <selected-key>`",
		"Do not claim the `BacklogItemView` selection directly.",
	} {
		if !strings.Contains(execute, want) {
			t.Errorf("F07 Rider self-pull procedure omits %q", want)
		}
	}

	for name, content := range map[string]string{
		"role-pull workflow":        pullByRole,
		"worker ownership contract": workerOwnership,
		"Shark Attack skill":        skill,
	} {
		for _, want := range []string{
			"worker-owned child mode",
			"not `/shark-rider run`",
		} {
			if !strings.Contains(content, want) {
				t.Errorf("%s omits the non-Rider child-mode boundary %q", name, want)
			}
		}
	}

	if !strings.Contains(pullByRole, "Do not hand this child session to `/shark-rider run`.") {
		t.Error("role-pull workflow must prohibit handing a worker-owned child session to the Rider loop")
	}
	for _, want := range []string{
		"Role-aware Rider self-pull never claims or executes the returned `BacklogItemView` directly.",
		"`/shark-rider run <selected-key>`",
		"`shark next <selected-key> --json`",
		"`response.prompt`",
		"`response.provider`",
		"`response.model`",
	} {
		if !strings.Contains(pullByRole, want) {
			t.Errorf("role-pull workflow omits canonical Rider reentry boundary %q", want)
		}
	}
	for _, want := range []string{
		"Do not claim a selected `BacklogItemView` directly.",
		"Only `shark next {KEY} --json` supplies a claimable dispatch response",
		"`response.entity_key`",
		"Prompt: exactly `response.prompt`",
		"`response.provider`",
		"`response.model`",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("Rider procedure omits canonical dispatch-after-selection boundary %q", want)
		}
	}
	if !strings.Contains(workerOwnership, "A Rider-dispatched worker never claims, heartbeats, releases, or selects a replacement entity.") {
		t.Error("worker ownership contract must prohibit all worker lease and replacement-selection actions in Rider mode")
	}
}

// TestTC006_MaterialEscalationUsesBoundedCouncilContract verifies F07 invokes
// the existing I-04 artifact contract with the required routing metadata.
func TestTC006_MaterialEscalationUsesBoundedCouncilContract(t *testing.T) {
	execute := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/execute.md"))
	for _, want := range []string{
		"material scope/architecture/quality questions",
		"material question, evidence, responsible role, requested decision, route, and next owner",
		"follow `escalate.md`",
		"absent policy routes to `council-review`",
	} {
		if !strings.Contains(execute, want) {
			t.Errorf("F07 escalation procedure omits %q", want)
		}
	}
}

// TestTC007_RefreshUsesSharkAndBoundedCouncilPointers verifies the procedure
// resumes from existing state and council artifacts, not a second run ledger.
func TestTC007_RefreshUsesSharkAndBoundedCouncilPointers(t *testing.T) {
	execute := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/execute.md"))
	for _, want := range []string{
		"Shark claims, history, and `shark next` remain the operational source of truth.",
		"coordinator follows `resume.md`",
		"do not create a second resume record",
		"store prompts, credentials, transcripts, or unrestricted output",
	} {
		if !strings.Contains(execute, want) {
			t.Errorf("F07 refresh procedure omits %q", want)
		}
	}
}
