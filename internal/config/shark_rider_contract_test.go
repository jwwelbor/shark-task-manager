package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharkRiderHostContract(t *testing.T) {
	root := findRepoRootForAudit(t)

	_, err := os.Stat(filepath.Join(root, "skills", "shark"))
	require.True(t, os.IsNotExist(err), "legacy host skill directory must not remain")

	skill := readContractFile(t, root, "skills", "shark-rider", "SKILL.md")
	assertContainsAll(t, skill, "host skill identity", []string{
		"name: shark-rider",
		"# Shark Rider",
		"/shark-rider project bootstrap",
		"shark <command>",
		"CLI owns, Rider drives",
	})

	project := readContractFile(t, root, "skills", "shark-rider", "verbs", "project.md")
	assertContainsAll(t, project, "bootstrap coordinator", []string{
		"shark admin init --non-interactive",
		"shark admin validate",
		"/shark-rider project product-design",
		"/shark-rider project brownfield-analysis",
		"docs/product/progress.md",
		"schema_version: 2",
		"[~]",
		"D07",
		"feasible as described",
	})
	if strings.Contains(project, "shark skill get product-design") {
		t.Error("bootstrap must load the product-design Rider procedure instead of owning its bundle retrieval")
	}

	product := readContractFile(t, root, "skills", "shark-rider", "verbs", "product-design.md")
	assertContainsAll(t, product, "product-design adapter", []string{
		"shark skill get product-design",
		"After each D01–D14 artifact",
		"same coordination run",
		"docs/product/progress.md",
	})

	template := readContractFile(t, root, "file_templates", "progress.md")
	assertContainsAll(t, template, "progress template", []string{
		"schema_version: 2",
		"estate:",
		"initiative_posture:",
		"product_design_scope:",
		"brownfield_depth:",
		"architecture_state:",
		"stack_summary:",
		"artifact_paths:",
		"last_refreshed:",
		"[~]",
		"## Cross-Epic Integration Map",
		"## Decision Log",
	})
}

func TestSharkRiderRunAliasRoutesToCoreLoop(t *testing.T) {
	root := findRepoRootForAudit(t)
	skill := readContractFile(t, root, "skills", "shark-rider", "SKILL.md")

	assert.Contains(t, skill, "`/run <key>` is an alias for `/shark-rider run <key>`")
	assert.Contains(t, skill, "**`/shark-rider run <key>`** (alias `/run <key>`) — the core loop")
	assert.Contains(t, skill, "`/shark-rider run` and `/run` both route to `verbs/run.md`")
	assert.NotContains(t, skill, "`/shark-rider run <key>` is an alias for `/shark-rider run <key>`")
}

func TestSharkRiderConsultModesAreRegistered(t *testing.T) {
	root := findRepoRootForAudit(t)
	skill := readContractFile(t, root, "skills", "shark-rider", "SKILL.md")

	assertContainsAll(t, skill, "consult mode registration", []string{
		"`/shark-rider consult <agent> [referent]`",
		"`/shark-rider consult-agent <agent> [referent]`",
		"`consult-agent`",
		"`verbs/consult.md`",
		"`verbs/consult-agent.md`",
	})
	assert.Contains(t, skill, "| Consult an agent inline | "+
		"`/shark-rider consult <agent> [referent]` → resolve with `shark agent list --json`, "+
		"load via `shark agent get`, and continue the conversation in the persona's voice; "+
		"read-only by default → `verbs/consult.md` |")
	assert.Contains(t, skill, "| Request delegated agent advice | "+
		"`/shark-rider consult-agent <agent> [referent]` → resolve the shared request, "+
		"launch one bounded host subagent only with enforced read-only permissions, then review and synthesize its advice; otherwise direct to inline consult → "+
		"`verbs/consult-agent.md` |")
	assert.Contains(t, skill, "A failed\npersona lookup must list `shark agent list --all` and stop before persona\nadoption or dispatch.")
}

func TestSharkRiderInlineConsultContract(t *testing.T) {
	root := findRepoRootForAudit(t)
	inline := readContractFile(t, root, "skills", "shark-rider", "verbs", "consult.md")
	guide := readContractFile(t, root, "skills", "shark-rider", "context", "consult-role-guide.md")
	resolution := readContractFile(t, root, "skills", "shark-rider", "context", "consult-request-resolution.md")

	assert.Contains(t, inline, "If there are **no args** or the only arg is `help`")
	assert.Contains(t, inline, "skills/shark-rider/context/consult-role-guide.md` verbatim")
	assert.Contains(t, inline, "Then **STOP**. Do not treat `help` as an agent name.")
	assert.Contains(t, inline, "skills/shark-rider/context/consult-request-resolution.md` with the arguments")
	assert.Contains(t, inline, "No subagent is spawned")
	assert.Contains(t, inline, "Adopt the persona **inline**")
	assert.Contains(t, inline, "For one-shot delegated advice, use")
	assertConsultRoleGuide(t, guide)
	assertConsultRequestResolution(t, resolution)
}

func TestSharkRiderDelegatedConsultContract(t *testing.T) {
	root := findRepoRootForAudit(t)
	delegated := readContractFile(t, root, "skills", "shark-rider", "verbs", "consult-agent.md")
	guide := readContractFile(t, root, "skills", "shark-rider", "context", "consult-role-guide.md")
	resolution := readContractFile(t, root, "skills", "shark-rider", "context", "consult-request-resolution.md")

	assert.Contains(t, delegated, "If there are **no args** or the only arg is `help`")
	assert.Contains(t, delegated, "skills/shark-rider/context/consult-role-guide.md` verbatim")
	assert.Contains(t, delegated, "Then **STOP**. Do not treat `help` as an agent name.")
	assert.Contains(t, delegated, "bounded host subagent when the host can\nenforce read-only permissions")
	assert.Contains(t, delegated, "Select and launch a suitable subagent")
	assert.Contains(t, delegated, "skills/shark-rider/context/consult-request-resolution.md` with the arguments")
	assert.Contains(t, delegated, "The requested question or decision and the resolved referent.")
	assert.Contains(t, delegated, "The Shark persona definition and relevant retrieved skill/context content.")
	assert.Contains(t, delegated, "Only the project facts and artifact content needed to advise.")
	assert.Contains(t, delegated, "A read-only scope: inspect and recommend, but do not edit files, mutate Shark")
	assert.Contains(t, delegated, "state, launch other agents, or perform implementation.")
	assert.Contains(t, delegated, "The expected response: a concrete recommendation, its reasoning and evidence,")
	assert.Contains(t, delegated, "assumptions or uncertainty, and any follow-up needed from the parent.")
	assert.Contains(t, delegated, "Require a host-provided read-only sandbox or tool profile")
	assert.Contains(t, delegated, "prevents workspace\nwrites, Shark mutations, and nested dispatch.")
	assert.Contains(t, delegated, "do not launch the subagent")
	assert.Contains(t, delegated, "A prompt instruction alone does not count as an enforced")
	assert.Contains(t, delegated, "If no suitable host subagent, model, or enforced read-only profile is available,")
	assert.Contains(t, delegated, "does **not** mutate Shark state")
	assertConsultRoleGuide(t, guide)
	assertConsultRequestResolution(t, resolution)
}

func TestSharkRiderDelegatedConsultAcceptanceIsSpecified(t *testing.T) {
	root := findRepoRootForAudit(t)
	feature := readContractFile(t, root, "docs", "plan", "E36-project-layer-and-consult-bridge", "E36-F01-consult-bridge", "feature.md")
	requirements := readContractFile(t, root, "docs", "plan", "E36-project-layer-and-consult-bridge", "requirements.md")
	requirements = strings.Join(strings.Fields(requirements), " ")
	procedureTask := readContractFile(t, root, "docs", "plan", "E36-project-layer-and-consult-bridge", "E36-F01-consult-bridge", "tasks", "T-E36-F01-002.md")
	requestResolution := readContractFile(t, root, "skills", "shark-rider", "context", "consult-request-resolution.md")
	routingTask := readContractFile(t, root, "docs", "plan", "E36-project-layer-and-consult-bridge", "E36-F01-consult-bridge", "tasks", "T-E36-F01-003.md")
	epic := readContractFile(t, root, "docs", "plan", "E36-project-layer-and-consult-bridge", "epic.md")

	assertContainsAll(t, feature, "delegated consultation acceptance scenario", []string{
		"**Scenario 7: Successful delegated consultation**",
		"`/shark-rider consult-agent architect review this design`",
		"`docs/design.md` is the unambiguous design artifact in the conversation",
		"`docs/design.md` is identified and read",
		"one bounded host-subagent request with the enforced read-only profile",
		"parent reviews the advice and synthesizes a response in its own voice",
		"no Shark state, workspace files, or additional agents are changed or launched",
		"**Scenario 8: No agents are installed**",
		"advised to run `shark admin install-shark-data`",
		"**Scenario 9: Host cannot enforce delegated read-only permissions**",
		"no subagent is launched",
	})
	assert.Contains(t, procedureTask, "loads the persona and referent, sends one bounded host-subagent request, then has the parent review and synthesize the advice")
	assert.Contains(t, procedureTask, "A delegated consultation does not mutate Shark state or workspace files and does not launch nested agents")
	assert.Contains(t, requestResolution, "If the list is empty, say no Shark agents are installed, suggest\n  `shark admin install-shark-data`, and **STOP**.")
	assert.Contains(t, procedureTask, "If the host cannot enforce read-only permissions for workspace writes, Shark mutations, and nested dispatch, delegated mode does not launch")
	assertContainsAll(t, requirements, "delegated consultation requirements", []string{
		"**REQ-F4** — Consultation modes:",
		"Both modes use `skills/shark-rider/context/consult-request-resolution.md`",
		"suggest `shark admin install-shark-data` and stop.",
		"`/shark-rider consult-agent <agent> [referent]` resolves the persona and",
		"sends one bounded request only when a host-enforced read-only profile prevents",
		"The parent checks and synthesizes the advice.",
		"This is a separate, explicit",
	})
	assertContainsAll(t, procedureTask, "delegated consultation task plan", []string{
		"### 2. Delegated one-shot consultation",
		"one bounded host-subagent request",
		"Have the parent inspect the result and synthesize the advice in its own voice.",
		"Require an enforced host permission/tool profile",
		"skills/shark-rider/verbs/consult-agent.md",
		"skills/shark-rider/context/consult-role-guide.md",
		"skills/shark-rider/context/consult-request-resolution.md",
	})
	assert.Contains(t, routingTask, "`/shark-rider consult-agent <agent> [referent]` is separately registered and routes to `verbs/consult-agent.md`")
	assert.Contains(t, routingTask, "consult-agent` verb remains a separate explicit one-shot mode")
	assert.Contains(t, epic, "shared `context/consult-role-guide.md` and `context/consult-request-resolution.md`")
	assert.Contains(t, requirements, "If the host cannot enforce these permissions, do not launch and direct the user to inline `consult`.")
}

func TestSharkRiderConsultQueryRoutingContract(t *testing.T) {
	root := findRepoRootForAudit(t)
	query := readContractFile(t, root, "skills", "shark-rider", "verbs", "query.md")

	assert.Contains(t, query, "routing** — instead `Read skills/shark-rider/verbs/consult.md` and follow it with\n`agent=<agent>` and `referent=<referent>`.")
	assert.Contains(t, query, "Natural-language requests use inline")
	assert.Contains(t, query, "If the request clearly asks to consult an agent but `<agent>` does not resolve,\nstill `Read skills/shark-rider/verbs/consult.md` with the extracted `agent` and\n`referent`.")
	assert.Contains(t, query, "Use `/shark-rider consult-agent` for one-shot subagent advice")
}

func assertConsultRoleGuide(t *testing.T, guide string) {
	t.Helper()
	assertContainsAll(t, guide, "shared consultation role guide", []string{
		"Choose the role that matches the question:",
		"`tech-director`:",
		"`architect`:",
		"`product-manager`:",
		"`qa`:",
		"`cx-designer`:",
		"`ux-designer`:",
		"If specialist advice conflicts",
		"run `shark agent list --json`",
	})
}

func assertConsultRequestResolution(t *testing.T, resolution string) {
	t.Helper()
	assertContainsAll(t, resolution, "shared consultation request resolution", []string{
		"First token → `<agent>` candidate.",
		"Remaining tokens → `<referent>`",
		"If the list is empty",
		"shark admin install-shark-data",
		"fuzzy matches",
		"No agent matching",
		"shark agent get <resolved-agent>",
		"was listed but its content could not be loaded.",
		"Path-shaped file token",
		"known file suffix",
		"sentence-final punctuation",
		"A period elsewhere in ordinary prose is not enough",
		"Shark entity key accepted by `shark get`",
		"most recent unambiguous artifact",
		"Empty or ambiguous prose",
		"If a requested file, entity, or in-context artifact cannot be identified",
		"Read persona-referenced context only when it is relevant.",
		"Get Shark-bundled\nskills through `shark skill get <name> [relative-path]`",
		"If required content is unavailable,",
	})
}

func TestSharkRiderBootstrapChildHandoffContract(t *testing.T) {
	root := findRepoRootForAudit(t)
	project := readContractFile(t, root, "skills", "shark-rider", "verbs", "project.md")
	product := readContractFile(t, root, "skills", "shark-rider", "verbs", "product-design.md")
	brownfield := readContractFile(t, root, "skills", "shark-rider", "verbs", "brownfield-analysis.md")
	execution := readContractFile(t, root, "skills", "shark-rider", "context", "project-bootstrap-execution.md")

	assert.Contains(t, project, "skills/shark-rider/verbs/product-design.md")
	assert.Contains(t, project, "skills/shark-rider/verbs/brownfield-analysis.md")
	assert.Contains(t, project, "CHILD ACTION RESULT")
	assert.Contains(t, project, "same coordination run")
	assert.NotContains(t, project, "Dispatch `/shark-rider project product-design`")
	assert.NotContains(t, project, "Dispatch `/shark-rider project brownfield-analysis`")

	assert.Contains(t, product, "shark skill get product-design")
	assert.Contains(t, product, "Return a `CHILD ACTION RESULT`")
	assert.Contains(t, product, "stack_feedback")
	assert.NotContains(t, product, "dispatch `/shark-rider project bootstrap`")
	assert.Contains(t, brownfield, "CHILD ACTION RESULT")
	assert.NotContains(t, brownfield, "dispatch another host-level Rider action")

	for _, checkpoint := range []string{
		"created before product or analysis work starts",
		"changes after the durable artifact",
		"Interrupt after one artifact",
		"bootstrap consumes it",
		"feasible as described",
	} {
		assert.Contains(t, execution, checkpoint)
	}
}

func TestSharkRiderProjectInitCompatibilityRoute(t *testing.T) {
	root := findRepoRootForAudit(t)
	alias := readContractFile(t, root, "skills", "shark-rider", "verbs", "project-init.md")

	assert.Contains(t, alias, "deprecated")
	assert.Contains(t, alias, "/shark-rider project bootstrap")
	assert.Contains(t, alias, "skills/shark-rider/verbs/project.md")
	assert.NotContains(t, alias, "shark admin init")
}

func TestActiveBundleAndRiderInstructionsUseRiderSyntax(t *testing.T) {
	root := findRepoRootForAudit(t)
	paths := []string{
		filepath.Join(root, "skills", "shark-rider"),
		filepath.Join(root, "internal", "sharkdata", "default_data", "skills"),
	}

	for _, base := range paths {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sanitized := strings.ReplaceAll(string(contents), "/shark-rider", "")
			if strings.Contains(sanitized, "/shark") {
				t.Errorf("active Rider or bundle instruction uses the legacy host syntax: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk active instruction tree %s: %v", base, err)
		}
	}
}

func readContractFile(t *testing.T, root string, parts ...string) string {
	t.Helper()
	path := filepath.Join(root, filepath.Join(parts...))
	contents, err := os.ReadFile(path)
	if err != nil {
		require.NoErrorf(t, err, "read %s", path)
	}
	return string(contents)
}

func assertContainsAll(t *testing.T, contents, description string, needles []string) {
	t.Helper()
	for _, needle := range needles {
		assert.Containsf(t, contents, needle, "%s is missing %q", description, needle)
	}
}
