package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestB081RunSprintUsesSprintScopedSelection(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs(filepath.Join(workingDir, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(repoRoot, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	canonicalWorkflowPath := "internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint.md"
	hostWorkflowPath := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	canonicalWorkflow := read(canonicalWorkflowPath)
	hostWorkflow := read(hostWorkflowPath)
	if canonicalWorkflow != hostWorkflow {
		t.Fatalf("embedded canonical %s and host mirror %s diverged", canonicalWorkflowPath, hostWorkflowPath)
	}

	for _, required := range []string{
		`NEXT_CMD = "shark plan {SPRINT_KEY} --json"`,
		`NEXT_CMD = "shark plan {SPRINT_KEY} --agent={AGENT_FILTER} --json"`,
		`SPRINT_SELECTION.sprint_key != SPRINT_KEY`,
		`SPRINT_SELECTION.action == "select_item"`,
		`SPRINT_SELECTION.action == "parallel_candidates"`,
		`SELECTED_ENTITY.sprint_key != SPRINT_KEY`,
		`No dispatchable entity was selected; sprint drain was not verified.`,
	} {
		if !strings.Contains(canonicalWorkflow, required) {
			t.Errorf("%s: missing sprint-scoped selection contract %q", canonicalWorkflowPath, required)
		}
	}
	if strings.Contains(canonicalWorkflow, "shark sprint next {SPRINT_KEY}") {
		t.Errorf("%s still passes an unsupported sprint key to `shark sprint next`", canonicalWorkflowPath)
	}

	canonicalSkill := read("internal/sharkdata/default_data/skills/sprint-execution/SKILL.md")
	hostSkill := read("skills/shark-rider/skills/sprint-execution/SKILL.md")
	for path, skill := range map[string]string{
		"internal/sharkdata/default_data/skills/sprint-execution/SKILL.md": canonicalSkill,
		"skills/shark-rider/skills/sprint-execution/SKILL.md":              hostSkill,
	} {
		if !strings.Contains(skill, "shark plan S### --json") {
			t.Errorf("%s does not direct /run-sprint to the sprint-scoped plan selector", path)
		}
	}

	activePrompt := read("internal/sharkdata/default_data/prompts/sprint/active.md")
	if !strings.Contains(activePrompt, "shark plan {{.id}} --json") {
		t.Error("embedded active-sprint prompt does not use the sprint-scoped selector")
	}

	verb := read("skills/shark-rider/verbs/run-sprint.md")
	if !strings.Contains(verb, "shark plan S###") {
		t.Error("/run-sprint verb summary does not describe the sprint-scoped selector")
	}

	cliReference := read("docs/cli-reference/sprint-commands.md")
	if strings.Contains(cliReference, "shark sprint next [sprint-key]") || strings.Contains(cliReference, "shark sprint next S024") {
		t.Error("CLI reference still documents the unsupported positional sprint key")
	}
	if !strings.Contains(cliReference, "shark sprint next [flags]") {
		t.Error("CLI reference does not show the keyless `shark sprint next` syntax")
	}
}
