// Package contracts verifies the published prompt and skill contracts for
// E38-F12. These tests deliberately read the real embedded and authored
// content; E38-F12 adds no scheduler, claim, Question, or git runtime.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

func readF12Embedded(t *testing.T, path string) string {
	t.Helper()
	content, err := sharkdata.ReadEmbedded(path)
	if err != nil {
		t.Fatalf("read embedded %s: %v", path, err)
	}
	return string(content)
}

func readF12Authored(t *testing.T, path string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func requireF12Contains(t *testing.T, content, surface string, markers ...string) {
	t.Helper()
	normalized := strings.Join(strings.Fields(content), " ")
	for _, marker := range markers {
		if !strings.Contains(normalized, strings.Join(strings.Fields(marker), " ")) {
			t.Errorf("%s must contain %q", surface, marker)
		}
	}
}

func requireF12Excludes(t *testing.T, content, surface string, forbidden ...string) {
	t.Helper()
	normalized := strings.Join(strings.Fields(content), " ")
	for _, marker := range forbidden {
		if strings.Contains(normalized, strings.Join(strings.Fields(marker), " ")) {
			t.Errorf("%s must not contain retired or prohibited contract %q", surface, marker)
		}
	}
}

// requireF12Ordered proves that a guard is reached before a later action. A
// bag of matching phrases is insufficient for published control procedures:
// an invocation placed before a no-run branch would still be unsafe.
func requireF12Ordered(t *testing.T, content, surface string, markers ...string) {
	t.Helper()
	normalized := strings.Join(strings.Fields(content), " ")
	previous := -1
	for _, marker := range markers {
		needle := strings.Join(strings.Fields(marker), " ")
		index := strings.Index(normalized, needle)
		if index < 0 {
			t.Errorf("%s must contain ordered marker %q", surface, marker)
			continue
		}
		if index <= previous {
			t.Errorf("%s must place %q after its preceding guard", surface, marker)
		}
		previous = index
	}
}

func f12ParallelTeam(t *testing.T) string {
	t.Helper()
	return readF12Embedded(t, "skills/shark-attack/workflows/parallel-team.md")
}

// TC-001 proves the selection-to-keyed-parent boundary. The coordinator
// delegates only a key; the ordinary Rider loop remains the sole claimant and
// prompt carrier.
func TestTC001KeyedSelectionAndTeammateParentContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	rider := readF12Authored(t, "skills/shark-rider/verbs/run-agent-team.md")

	requireF12Contains(t, parallel, "parallel-team.md",
		"shark plan <root> --json",
		"Treat its `select_task` or `parallel_candidates` response as\nselection, not dispatch.",
		"shark next <key> --json",
		"dispatch the exact `response.prompt`",
		"A selector supplies a key only; it never supplies a claimable prompt.",
		"It never claims, advances, or releases a delivery entity.")
	requireF12Contains(t, rider, "run-agent-team.md",
		"Do not construct a worker prompt, calculate a DAG, claim a delivery entity, or advance or release its status here.")
	requireF12Excludes(t, rider, "run-agent-team.md", "shark claim <response.entity_key>", "spawn host agent with response.prompt")
}

// TC-002 proves rolling refill cannot mistake a temporary selection gap for
// terminal completion.
func TestTC002ClaimAwareRefillAndNonterminalReportingContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"not already in flight",
		"On each teammate completion, not only when a wave drains, repeat the\nnon-terminal direct-child precheck before selection and refill an idle teammate",
		"`shark plan\n<feature>` is claim-aware; in-flight dedup remains a race defense.",
		"A no-candidate result is never itself success.",
		"/shark-rider query: list <epic> <feature>\n--all --json",
		"paused, blocked, claimed, or Question-gated work")
}

// TC-002 also proves that plan is never used as a harmless terminal-parent
// inspection, and that both hierarchy and sprint closing checks include the
// complete roster rather than only currently selectable work.
func TestTC002TerminalParentPreflightAndCompleteRosterContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"first run the non-mutating, all-inclusive direct-child precheck `/shark-rider query: list <root> --all --json`",
		"Call `shark plan <root> --json` only when that precheck shows at least one non-terminal direct child.",
		"This precheck is mandatory before initial selection and every refill",
		"when every child is terminal, `shark plan` auto-advances its parent and is not safe as an inspection call.",
		"all-inclusive terminal query `/shark-rider query: list <epic> <feature> --all --json`, which must show every direct task terminal.",
		"`shark sprint backlog <sprint-key> --all --json` and require every assigned task, bug, change-card, and tech-debt item to be terminal.")
	requireF12Ordered(t, parallel, "parallel-team.md",
		"all-inclusive direct-child precheck `/shark-rider query: list <root> --all --json`",
		"Call `shark plan <root> --json` only when that precheck shows at least one non-terminal direct child.")
}

// TC-003 preserves F09's independent topology evidence rules and dependency
// order instead of treating isolation as permission to reorder work.
func TestTC003EvidenceBasedTopologyDecisionTableContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"`Sequential` is the default.",
		"`Parallel with ownership` requires recorded disjoint ownership/write-scope\n evidence; `Parallel with isolation` requires recorded isolation evidence.",
		"Missing ownership evidence and missing isolation evidence independently\n degrade to `Sequential`.",
		"Producer/consumer order remains binding under either\n parallel topology")
}

// TC-004 ensures shared-worktree parallel craft is evidence-gated and that
// commits and the full quality gate remain serialized.
func TestTC004SharedWorktreeSerializationContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"parallel craft is allowed only with\n the recorded disjoint scopes",
		"one mutually exclusive\n turn for commits and for the complete `make fmt && make lint && make test`\n gate",
		"file-scoped staging discipline",
		"There is no standing merge-referee role and no concurrent shared-worktree gate run.")
}

// TC-005 preserves the existing E39 and council authority split. It tests
// published routing prose only; it does not simulate a Question lifecycle.
func TestTC005QuestionAndCouncilRouteContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	council := readF12Embedded(t, "skills/shark-attack/workflows/council.md")
	requireF12Contains(t, parallel, "parallel-team.md",
		"For a **routine** question, the entity-parent teammate mints, configures,\n  and links the scoped `Q###` through `route-question.md`. The coordinator\n  then claims that Question, routes responders, records responses, and\n  resolves it under the Question lease.",
		"For a **material** question, route directly through `council.md`. Do not\n  mint or claim a `Q###`.",
		"Ready unrelated work\n continues through rolling refill.")
	requireF12Contains(t, council, "council.md", "A routine question creates no `docs/council/` artifact.", "A material question routes through this file only")
}

// TC-005 verifies both mutually-exclusive Question branches. In particular,
// a material issue may not create a parallel Q lifecycle merely because the
// routine branch happens to mention the E39 route.
func TestTC005QuestionRouteBranchesAreExclusiveContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"classify it against the material threshold in `council.md` before minting any Question or council artifact.",
		"The branches are mutually exclusive:",
		"For a **routine** question, the entity-parent teammate mints, configures, and links the scoped `Q###` through `route-question.md`.",
		"For a **material** question, route directly through `council.md`. Do not mint or claim a `Q###`.",
		"The entity-parent teammate retains the delivery entity lease")
	requireF12Ordered(t, parallel, "parallel-team.md",
		"classify it against the material threshold in `council.md` before minting any Question or council artifact.",
		"For a **routine** question",
		"For a **material** question")
}

// TC-006 makes holds event-bounded and retains the all-parked report rather
// than introducing a fixed Question timeout.
func TestTC006QuestionHoldEventBoundaryContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"at least every ten minutes",
		"deterministically longest-held teammate",
		"At a run stop, resume boundary, or team cleanup, convert every live hold",
		"report the open `Q###` keys to the owner.",
		"Holds are event-bounded, never controlled by a fixed Question\n wait timeout")
	requireF12Excludes(t, parallel, "parallel-team.md", "Question-wait timeout:", "wait 30 minutes for Question")
}

// TC-007 pins the environment-scoped lease policy and its prohibited
// alternatives without inventing a claim-store runtime test.
func TestTC007LeaseAndHeartbeatNegativeCorpusContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"Export `SHARK_CLAIM_TTL_SECONDS=1800` for this team session only\n when `.sharkconfig.json` has no `claim_ttl_seconds` key.",
		"at least every ten minutes while waiting.",
		"force-steal as normal recovery",
		"persistent lease/claim store")
	requireF12Excludes(t, parallel, "parallel-team.md", "SHARK_CLAIM_TTL_SECONDS=0", "claim_ttl_seconds: 1800")
}

// TC-008 keeps sprint selection bounded to the active backlog and keeps
// planning and retro as council evidence, not automatic lifecycle actions.
func TestTC008SprintModeAndCeremonyContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	council := readF12Embedded(t, "skills/shark-attack/workflows/council.md")
	requireF12Contains(t, parallel, "parallel-team.md",
		"its backlog is the sole selection universe",
		"Until\n E19-F09 provides `shark plan sprint`, enumerate the active backlog in its\n documented order (`sprint_order`, `execution_order`, `priority`, `assigned_at`)",
		"Do not use free selection or repeated `sprint next` calls to construct a wave.",
		"only the owner starts or closes a sprint.")
	requireF12Contains(t, council, "council.md", "Sprint planning and retro are council ceremonies", "The council proposes only: the owner alone starts\n the sprint.", "The owner alone closes a sprint; no ceremony automatically\n starts or closes one.")
}

// TC-009 verifies every team-sprint surface is a thin alias and that the solo
// pull loop still has its own sprint-next contract.
func TestTC009SprintTeamAliasAndSoloSeparationContract(t *testing.T) {
	for _, alias := range []struct {
		path  string
		key   string
		guard string
	}{
		{"skills/shark-rider/verbs/run-sprint-team.md", "S###", "It does not group entities by feature, create nested teams"},
		{"skills/shark-rider/skills/sprint-execution/SKILL.md", "S###", "It does not group work by feature or create nested teams."},
		{"skills/shark-rider/skills/sprint-execution/workflows/run-sprint-team.md", "{SPRINT_KEY}", "Do not group items by feature, make nested teams"},
	} {
		content := readF12Authored(t, alias.path)
		requireF12Contains(t, content, alias.path,
			"/shark-rider run-agent-team --sprint "+alias.key,
			alias.guard)
		requireF12Excludes(t, content, alias.path, "feature-grouped nested-team workflow", "nested team bootstrap")
	}
	solo := readF12Authored(t, "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md")
	requireF12Contains(t, solo, "run-sprint.md", "sprint next", "/shark-rider run")
}

// TC-009 also guards the complete public Rider execution/help corpus. The
// three alias files above own execution detail; these router, entrypoint,
// discovery, solo, and published-reference surfaces must not revive the
// retired feature-grouped topology either.
func TestTC009PublicRiderTeamReferencesUseTopologyAlias(t *testing.T) {
	for _, surface := range []struct {
		path      string
		required  string
		forbidden string
	}{
		{
			path:      "skills/shark-rider/SKILL.md",
			required:  "Thin alias for `/shark-rider run-agent-team --sprint S###`",
			forbidden: "parallel feature groups via agent teams",
		},
		{
			path:      "skills/shark-rider/verbs/run-agent-team.md",
			required:  "Run a selected root through the canonical Shark Attack team topology.",
			forbidden: "feature group",
		},
		{
			path:      "skills/shark-rider/verbs/help.md",
			required:  "run-agent-team <epic-key|feature-key> | run-sprint-team <sprint-key>",
			forbidden: "Sprint execution grouped by feature, with standalone entities run sequentially.",
		},
		{
			path:      "skills/shark-rider/verbs/run-sprint.md",
			required:  "For team topology mode, use `/shark-rider run-sprint-team`.",
			forbidden: "parallel feature groups via agent teams",
		},
		{
			path:      "docs/cli-reference/sprint-commands.md",
			required:  "Drive an active sprint through the canonical team topology — the active backlog is its sole selection universe.",
			forbidden: "Groups tasks by feature key",
		},
	} {
		content := readF12Authored(t, surface.path)
		requireF12Contains(t, content, surface.path, surface.required)
		requireF12Excludes(t, content, surface.path, surface.forbidden)
	}
}

// TC-009 keeps team execution behind its bundled canonical procedure. The
// host entrypoint cannot silently substitute an authored local copy when the
// selected bundle lacks the procedure.
func TestTC009BundledParallelTeamRetrievalAndUnavailableFallbackContract(t *testing.T) {
	rider := readF12Authored(t, "skills/shark-rider/verbs/run-agent-team.md")
	requireF12Contains(t, rider, "run-agent-team.md",
		"shark skill get shark-attack workflows/parallel-team.md",
		"The Shark Attack parallel-team procedure is unavailable in this bundle; no team run was started.",
		"Do not read, copy, or construct a host-local substitute.",
		"run-agent-team requires an epic or feature root; no team run was\n   started.",
		"For `<epic-key|feature-key>`, follow the retrieved canonical Shark Attack `parallel-team.md` procedure",
		"For `--sprint S###`, follow the retrieved procedure in sprint mode.")
	requireF12Ordered(t, rider, "run-agent-team.md",
		"shark skill get shark-attack workflows/parallel-team.md",
		"The Shark Attack parallel-team procedure is unavailable in this bundle; no team run was started.",
		"For `<epic-key|feature-key>`, follow the retrieved canonical Shark Attack `parallel-team.md` procedure")
}

// TC-009 verifies each distributed team-sprint workflow performs a real
// read-only preflight and keeps every no-run branch before topology dispatch.
func TestTC009SprintExecutionPhasePreflightAndNoRunBranchesContract(t *testing.T) {
	for _, surface := range []struct {
		path string
		read func(*testing.T, string) string
	}{
		{"skills/shark-rider/skills/sprint-execution/workflows/run-sprint-team.md", readF12Authored},
		{"skills/sprint-execution/workflows/run-sprint-team.md", readF12Embedded},
	} {
		content := surface.read(t, surface.path)
		requireF12Contains(t, content, surface.path,
			"shark sprint get {SPRINT_KEY} --json",
			"configured execution-phase status (the bundled workflow calls it `active`)",
			"follow the project's `workflow_config` to its sprint workflow",
			"status whose metadata has `phase: execution`",
			"execution phase could not be resolved;\n   no team run was started.",
			"never assume a literal status label.",
			"If the sprint is not found or the read fails",
			"no team run was started.",
			"If its status is `planning`",
			"Only the owner may start it; no team run was started.",
			"If its status is terminal, closing, on hold, or otherwise outside the execution phase",
			"/shark-rider run-agent-team --sprint {SPRINT_KEY}")
		requireF12Ordered(t, content, surface.path,
			"shark sprint get {SPRINT_KEY} --json",
			"If the sprint is not found or the read fails",
			"If its status is `planning`",
			"If its status is terminal, closing, on hold, or otherwise outside the execution phase",
			"/shark-rider run-agent-team --sprint {SPRINT_KEY}")
	}
}

// TC-010 confines the isolation integrator to reviewed git integration and
// asserts its gate, fix-forward, escalation, and closeout evidence.
func TestTC010IsolationIntegratorContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"one worktree and branch per teammate",
		"merge\n that branch into the integration branch serially",
		"`make fmt && make\n lint && make test`",
		"The integrator may resolve only\n mechanical conflicts",
		"one scoped fix-forward",
		"Two consecutive fix-forward failures\n escalate to the council; do not attempt a third.",
		"review the\n worktree contents before removal",
		"The integrator reports entity key, merge commit, and gate result",
		"It never mutates Shark state.")
}

// TC-011 keeps the closing report bounded and excludes sensitive worker data.
func TestTC011ClosingReportRequiredFieldsAndNegativeCorpusContract(t *testing.T) {
	parallel := f12ParallelTeam(t)
	requireF12Contains(t, parallel, "parallel-team.md",
		"| Entity | Teammate | Semantic outcome | Merge commit | Gate result |",
		"wave count, wall-clock duration, raised/resolved Question counts,\n and fix-forward count",
		"one bounded feature note",
		"existing council ledger",
		"Never\n persist a rendered prompt, credential, token, unrestricted worker transcript,")
}
