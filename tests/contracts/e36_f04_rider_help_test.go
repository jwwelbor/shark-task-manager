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

// TestTC017_RiderHelpCallsAdviceOnlyInStateAwareMode reads the production
// Rider procedure and checks each decision-table branch independently.
func TestTC017_RiderHelpCallsAdviceOnlyInStateAwareMode(t *testing.T) {
	help := readE36F04Artifact(t, "skills/shark-rider/verbs/help.md")

	t.Run("fast and commands stay static", func(t *testing.T) {
		branch := e36F04Section(t, help, "## Static help: `--fast` and `commands`", "## Specific command help")
		assertE36F04Clauses(t, "fast/commands help", branch, []string{
			"Treat `commands` as a static alias for `--fast`.",
			"Do not run any `shark` command.",
		}, []string{
			"Run bare `shark next`",
			"Follow the portfolio advice prompt",
		})
	})

	t.Run("known verb stays static", func(t *testing.T) {
		branch := e36F04Section(t, help, "### Known verbs", "### Unknown verbs")
		assertE36F04Clauses(t, "known-verb help", branch, []string{
			"Print the matching static entry.",
			"Do not run any `shark` command.",
			"Do not fall through to state-aware help.",
		}, []string{"Run bare `shark next`"})
	})

	t.Run("unknown verb stays bounded and static", func(t *testing.T) {
		branch := e36F04Section(t, help, "### Unknown verbs", "## Bare `/shark-rider help` (state-aware)")
		assertE36F04Clauses(t, "unknown-verb help", branch, []string{
			"Show `/shark-rider help commands`",
			"Do not run any `shark` command.",
			"Do not fall through to state-aware help.",
		}, []string{"Run bare `shark next`"})
	})

	t.Run("bare help consumes advice inline", func(t *testing.T) {
		rawBranch := e36F04RawSection(t, help, "## Bare `/shark-rider help` (state-aware)", "")
		branch := normalizeE36F04Artifact(rawBranch)
		assertE36F04Clauses(t, "state-aware help", branch, []string{
			"Run bare `shark next` exactly once:",
			"Follow the returned `prompt` inline in the current agent turn.",
			"Inspect the relevant artifacts under `docs/product/` named by the prompt.",
			"recommend exactly one `eligibility=eligible` epic key",
			"why-now evidence",
			"strongest eligible alternative",
			"report the evidence or relationship gap instead of guessing",
			"The operator must separately invoke `/shark-rider run <recommended-key>`",
			"Do not call keyed `shark next <key>`",
			"Do not spawn a subagent",
			"Do not claim, advance, or automatically run the recommended epic.",
		}, []string{
			"Do not call `shark next` from state-aware help.",
			"shark status # overall dashboard",
			"shark task list --blocked",
			"shark claims # active leases",
		})
		if calls := countE36F04CommandLines(rawBranch, "shark next"); calls != 1 {
			t.Errorf("state-aware production help contains %d executable bare `shark next` lines, want exactly 1", calls)
		}
	})
}

func TestTC017_RiderRouterPreservesBothNextModes(t *testing.T) {
	skill := normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/SKILL.md"))
	assertE36F04Clauses(t, "Rider router", skill, []string{
		"Bare `shark next` is a read-only portfolio-advice query",
		"follow its returned prompt inline in the current agent turn",
		"`shark next <key> --json` is the only keyed dispatch API",
		"Pass `response.prompt` to the host agent unchanged.",
		"`/shark-rider run <key>` is the explicit handoff from advice to keyed dispatch.",
	}, []string{
		"Workflow engine routing + prompt assembly: shark next renders the step",
		"`shark next` is the dispatch API",
		"Repoint to `shark next` and dispatch its prompt.",
	})
}

func TestTC017_CommandModeDocumentsScopeDispatchAndLeasesToKeyedNext(t *testing.T) {
	architecture := normalizeE36F04Artifact(readE36F04Artifact(t, "docs/architecture/shark-dispatch-prompt-assembly.md"))
	assertE36F04Clauses(t, "dispatch architecture", architecture, []string{
		"This wire contract applies only to keyed `shark next <key> --json`.",
		"Bare `shark next` returns a read-only portfolio-advice envelope",
		"does not assemble a specialist dispatch prompt",
		"does not claim, advance, or normalize workflow state.",
		"`shark next <key> --json` returns the harness-facing wire shape",
		"pass `response.prompt` unchanged",
	}, []string{
		"`shark next` returns the harness-facing wire shape",
	})

	routeGuide := normalizeE36F04Artifact(readE36F04Artifact(t, "docs/guides/route-based-workflow.md"))
	assertE36F04Clauses(t, "route-based workflow guide", routeGuide, []string{
		"Keyed `shark next <root>` selects only unclaimed dispatchable entities.",
		"Bare `shark next` is read-only portfolio advice and does not select or lease an entity.",
		"entity = shark next <root>",
	}, []string{
		"`shark next` hands out only unclaimed entities",
	})

	combined := strings.Join([]string{architecture, routeGuide,
		normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/SKILL.md")),
		normalizeE36F04Artifact(readE36F04Artifact(t, "skills/shark-rider/verbs/help.md"))}, " ")
	if strings.Contains(combined, "shark next --preview") {
		t.Error("production Rider/command-mode artifacts must not restore preview guidance")
	}
}
