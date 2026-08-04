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
//
// T-E38-F09-024 retired workflows/execute.md, the file this test originally
// read for the self-pull contract: that content restated pull-by-role.md's
// own sanctioned path in different words, which is exactly the kind of
// duplication D-010's router restructure removes (AC-017). The self-pull
// contract now reads directly from pull-by-role.md, its one remaining
// canonical source, plus direct.md's pointer into it.
func TestTC005_RoleAwareSelfPullReentersCanonicalRiderDispatch(t *testing.T) {
	direct := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/direct.md"))
	pullByRole := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/pull-by-role.md"))
	workerOwnership := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/context/worker-ownership.md"))
	skill := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/SKILL.md"))
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))

	if !strings.Contains(direct, "a role-aware self-pull (`pull-by-role.md`'s sanctioned path)") {
		t.Error("Direct dispatch procedure must point a role-aware self-pull at pull-by-role.md's sanctioned path")
	}
	for _, want := range []string{
		"Start from the workflow role already resolved for the worker.",
		"Roster membership describes available expertise; it does not grant claim or status authority.",
		"`shark sprint next --agent=<type>`",
		"Invoke `/shark-rider run <selected-key>`.",
		"Role-aware Rider self-pull never claims or executes the returned `BacklogItemView` directly.",
	} {
		if !strings.Contains(pullByRole, want) {
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
//
// T-E38-F09-024 retired workflows/escalate.md; council.md (T-E38-F09-013) is
// its successor and now the sole canonical source for the material-question
// artifact contract (D-010, AC-017 — the rule may not be stated in both
// files). run.md already routes a mid-run `kind: needs_council` consultation
// through council.md rather than duplicating its threshold or procedure.
func TestTC006_MaterialEscalationUsesBoundedCouncilContract(t *testing.T) {
	council := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/council.md"))
	run := normalizeF07Content(readF07RepositoryFile(t, "skills/shark-rider/verbs/run.md"))

	for _, want := range []string{
		"specialists disagree, a cross-feature or cross-epic contract is missing or inconsistent, the change has high blast radius, the change is irreversible, or a worker cannot proceed safely from project evidence",
		"artifact ID, trigger, requested decision, evidence, sender and recipient roles, root key, optional child key, route, status, and `next_action`",
		"docs/product/escalation_triggers.md",
		"set `status: unresolved`, set `route: council-review`, and recommend `pause/review`",
	} {
		if !strings.Contains(council, want) {
			t.Errorf("F07 escalation procedure omits %q", want)
		}
	}
	if !strings.Contains(run, "workflows/council.md") {
		t.Error("Rider procedure must route a mid-run material consultation through workflows/council.md")
	}
}

// TestTC007_RefreshUsesSharkAndBoundedCouncilPointers verifies the procedure
// resumes from existing state and council artifacts, not a second run ledger.
//
// T-E38-F09-024 retired workflows/execute.md, which previously stated this
// refresh contract in its own paraphrase. The router (SKILL.md) now points a
// refresh at resume.md (T-E38-F09-014), which is the sole canonical source
// for the no-second-replacement and no-transcript-storage invariants.
func TestTC007_RefreshUsesSharkAndBoundedCouncilPointers(t *testing.T) {
	skill := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/SKILL.md"))
	resume := normalizeF07Content(readF07EmbeddedFile(t, "skills/shark-attack/workflows/resume.md"))

	if !strings.Contains(skill, "the parent follows `workflows/resume.md`") {
		t.Error("Shark Attack skill router must point worker refresh at workflows/resume.md")
	}
	for _, want := range []string{
		"without depending on prior chat history and without storing sensitive transcripts",
		"It excludes rendered prompt content, credentials, access tokens, and transcripts.",
		"Never start more than one replacement",
	} {
		if !strings.Contains(resume, want) {
			t.Errorf("F07 refresh procedure omits %q", want)
		}
	}
}
