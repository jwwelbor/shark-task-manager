package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readE36F04Artifact(t *testing.T, relativePath string) string {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read production artifact %s: %v", relativePath, err)
	}
	return string(content)
}

func normalizeE36F04Artifact(content string) string {
	return strings.Join(strings.Fields(content), " ")
}

func e36F04Section(t *testing.T, content, start, end string) string {
	t.Helper()
	return normalizeE36F04Artifact(e36F04RawSection(t, content, start, end))
}

func e36F04RawSection(t *testing.T, content, start, end string) string {
	t.Helper()
	startIndex := strings.Index(content, start)
	if startIndex == -1 {
		t.Fatalf("production artifact omits section %q", start)
	}
	content = content[startIndex:]
	if end == "" {
		return content
	}
	endIndex := strings.Index(content, end)
	if endIndex == -1 {
		t.Fatalf("production artifact section %q is not bounded by %q", start, end)
	}
	return content[:endIndex]
}

func assertE36F04Clauses(t *testing.T, branch string, content string, required, forbidden []string) {
	t.Helper()
	for _, clause := range required {
		if !strings.Contains(content, clause) {
			t.Errorf("%s omits required production clause %q", branch, clause)
		}
	}
	for _, clause := range forbidden {
		if strings.Contains(content, clause) {
			t.Errorf("%s retains forbidden production clause %q", branch, clause)
		}
	}
}

func countE36F04CommandLines(content, command string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == command {
			count++
		}
	}
	return count
}

// TestTC017_RiderHelpSelectsOnlyInStateAwareMode reads the production Rider
// procedure and checks each decision-table branch independently. Bare
// `/shark-rider help` now calls bare `shark plan` (work selection), not bare
// `shark next` (which is invalid — see next.go's requireNextEntityKey).
func TestTC017_RiderHelpSelectsOnlyInStateAwareMode(t *testing.T) {
	help := readE36F04Artifact(t, "skills/shark-rider/verbs/help.md")

	t.Run("fast and commands stay static", func(t *testing.T) {
		branch := e36F04Section(t, help, "## Static help: `--fast` and `commands`", "## Specific command help")
		assertE36F04Clauses(t, "fast/commands help", branch, []string{
			"Treat `commands` as a static alias for `--fast`.",
			"Do not run any `shark` command.",
		}, []string{
			"Run bare `shark plan`",
			"Follow the returned `prompt` inline",
		})
	})

	t.Run("known verb stays static", func(t *testing.T) {
		branch := e36F04Section(t, help, "### Known verbs", "### Unknown verbs")
		assertE36F04Clauses(t, "known-verb help", branch, []string{
			"Print the matching static entry.",
			"Do not run any `shark` command.",
			"Do not fall through to state-aware help.",
		}, []string{"Run bare `shark plan`"})
	})

	t.Run("unknown verb stays bounded and static", func(t *testing.T) {
		branch := e36F04Section(t, help, "### Unknown verbs", "## Bare `/shark-rider help` (state-aware)")
		assertE36F04Clauses(t, "unknown-verb help", branch, []string{
			"Show `/shark-rider help commands`",
			"Do not run any `shark` command.",
			"Do not fall through to state-aware help.",
		}, []string{"Run bare `shark plan`"})
	})

	t.Run("bare help selects epic inline", func(t *testing.T) {
		rawBranch := e36F04RawSection(t, help, "## Bare `/shark-rider help` (state-aware)", "")
		branch := normalizeE36F04Artifact(rawBranch)
		assertE36F04Clauses(t, "state-aware help", branch, []string{
			"Run bare `shark plan` exactly once:",
			"Treat the Shark selection as the live workflow and relationship authority.",
			"it is a bounded selection, not advice to relay verbatim",
			"`select_epic` — report the returned `entity` as the recommended next root",
			"`parallel_candidates` — report the tied epics from `entities`.",
			"`pause` — report the `reason`",
			"state the next evidence or relationship fix",
			"Stop at the recommendation.",
			"The operator must separately invoke `/shark-rider run <recommended-key>`",
			"Do not call keyed `shark next <key>` from help.",
			"Do not spawn a subagent.",
			"Do not claim, advance, or automatically run any recommended epic.",
		}, []string{
			"Follow the returned `prompt` inline in the current agent turn.",
			"shark status # overall dashboard",
			"shark task list --blocked",
			"shark claims # active leases",
		})
		if calls := countE36F04CommandLines(rawBranch, "shark plan --json"); calls != 1 {
			t.Errorf("state-aware production help contains %d executable bare `shark plan --json` lines, want exactly 1", calls)
		}
		if calls := countE36F04CommandLines(rawBranch, "shark next"); calls != 0 {
			t.Errorf("state-aware production help contains %d executable bare `shark next` lines, want exactly 0", calls)
		}
	})
}

// TestTC017_RiderRouterSeparatesPlanFromDispatch pins the corrected command
// boundary: `shark plan` selects (bare epic, hierarchy tier, or standalone
// tier); keyed `shark next <key>` dispatches. The two are no longer modes of
// one `next` command.
func TestTC017_RiderRouterSeparatesPlanFromDispatch(t *testing.T) {
	skill := normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/SKILL.md"))
	assertE36F04Clauses(t, "Rider router", skill, []string{
		"`shark plan [root|collection]` selects work and never dispatches.",
		"Bare `shark plan` selects one epic or an epic-only `parallel_candidates` tie",
		"`shark plan <epic|feature>` evaluates one hierarchy edge and returns direct",
		"children as a selection",
		"`shark plan bugs|change-cards|tech-debt` selects the",
		"next claimable standalone tier",
		"None of these claim, advance, or spawn an",
		"agent.",
		"`shark next <key> --json` resolves keyed workflow dispatch for one concrete",
		"entity, cascading internally to the first dispatchable descendant.",
		"`/shark-rider run <key>` is the explicit handoff from a selected key to",
		"keyed dispatch.",
		"`shark next <key> --json` is the only keyed dispatch API",
		"Pass `response.prompt` to the host agent unchanged.",
	}, []string{
		"Workflow engine routing + prompt assembly: shark next renders the step",
		"`shark next` is the dispatch API",
		"Repoint to `shark next` and dispatch its prompt.",
		"Bare `shark next` is a read-only portfolio-advice query",
	})
}

func TestTC017_CommandModeDocumentsScopeDispatchAndLeasesToKeyedNext(t *testing.T) {
	architecture := normalizeE36F04Artifact(readE36F04Artifact(t, "docs/architecture/shark-dispatch-prompt-assembly.md"))
	assertE36F04Clauses(t, "dispatch architecture", architecture, []string{
		"This wire contract applies only to keyed `shark next <key> --json`.",
		"It requires",
		"an entity key — bare `shark next` (no key) is invalid and errors, pointing the",
		"operator at `shark plan`.",
		"`shark plan [root|collection]` is the separate,",
		"read-only work-selection surface",
		"None of `shark plan`'s selection",
		"responses assemble a specialist dispatch prompt or claim, advance, or",
		"normalize workflow state",
		"`shark next <key> --json` returns the harness-facing wire shape",
		"pass `response.prompt` unchanged",
	}, []string{
		"`shark next` returns the harness-facing wire shape",
		"Bare `shark next` returns a read-only portfolio-advice envelope",
	})

	routeGuide := normalizeE36F04Artifact(readE36F04Artifact(t, "docs/guides/route-based-workflow.md"))
	assertE36F04Clauses(t, "route-based workflow guide", routeGuide, []string{
		"Keyed `shark next <key>` requires an entity key and never selects or leases work on its",
		"own",
		"Work selection is a separate, read-only",
		"surface: `shark plan [root|collection]` returns an epic, a one-level hierarchy",
		"tier, or a standalone-collection tier without claiming or leasing anything.",
		"entity = shark plan <root>",
	}, []string{
		"`shark next` hands out only unclaimed entities",
		"Bare `shark next` is read-only portfolio advice and does not select or lease an entity.",
		"entity = shark next <root>",
	})

	combined := strings.Join([]string{architecture, routeGuide,
		normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/SKILL.md")),
		normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/verbs/help.md"))}, " ")
	if strings.Contains(combined, "shark next --preview") {
		t.Error("production Rider/command-mode artifacts must not restore preview guidance")
	}
}
