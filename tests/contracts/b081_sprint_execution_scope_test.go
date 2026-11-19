package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestB081RunSprintWorkflowMirrorsAndValidatesSelection(t *testing.T) {
	canonicalPath := "internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint.md"
	hostPath := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	canonicalWorkflow := readB081RepoFile(t, canonicalPath)
	hostWorkflow := readB081RepoFile(t, hostPath)
	if canonicalWorkflow != hostWorkflow {
		t.Fatalf("embedded canonical %s and host mirror %s diverged", canonicalPath, hostPath)
	}

	for _, required := range []string{
		`NEXT_CMD = "shark plan {SPRINT_KEY} --json"`,
		`NEXT_CMD = "shark plan {SPRINT_KEY} --agent={AGENT_FILTER} --json"`,
		`SPRINT_SELECTION.sprint_key != SPRINT_KEY`,
		`SPRINT_SELECTION.action == "select_item"`,
		`SPRINT_SELECTION.action == "parallel_candidates"`,
		`SELECTED_ENTITY = SPRINT_SELECTION.entities[0]  # first item in selector order`,
		`SELECTED_ENTITY.sprint_key != SPRINT_KEY`,
		`No dispatchable entity was selected; sprint drain was not verified.`,
	} {
		assertB081Contains(t, canonicalPath, canonicalWorkflow, required)
	}
	if strings.Contains(canonicalWorkflow, "shark sprint next {SPRINT_KEY}") {
		t.Errorf("%s still passes an unsupported sprint key to `shark sprint next`", canonicalPath)
	}

	assertB081MismatchStopsBeforeDispatch(t, canonicalWorkflow, "if SPRINT_SELECTION.sprint_key != SPRINT_KEY:")
	assertB081MismatchStopsBeforeDispatch(t, canonicalWorkflow, "if SELECTED_ENTITY.sprint_key != SPRINT_KEY:")
}

func TestB081RunSprintSkillsAndPromptUseScopedSelector(t *testing.T) {
	for _, path := range []string{
		"internal/sharkdata/default_data/skills/sprint-execution/SKILL.md",
		"skills/shark-rider/skills/sprint-execution/SKILL.md",
	} {
		assertB081Contains(t, path, readB081RepoFile(t, path), "shark plan S### --json")
	}

	promptPath := "internal/sharkdata/default_data/prompts/sprint/active.md"
	assertB081Contains(t, promptPath, readB081RepoFile(t, promptPath), "shark plan {{.id}} --json")
	verbPath := "skills/shark-rider/verbs/run-sprint.md"
	assertB081Contains(t, verbPath, readB081RepoFile(t, verbPath), "shark plan S###")
}

func TestB081RunSprintCLIReferenceUsesKeylessSyntax(t *testing.T) {
	path := "docs/cli-reference/sprint-commands.md"
	cliReference := readB081RepoFile(t, path)
	if strings.Contains(cliReference, "shark sprint next [sprint-key]") || strings.Contains(cliReference, "shark sprint next S024") {
		t.Errorf("%s still documents the unsupported positional sprint key", path)
	}
	assertB081Contains(t, path, cliReference, "shark sprint next [flags]")
}

func readB081RepoFile(t *testing.T, relative string) string {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs(filepath.Join(workingDir, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(repoRoot, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(content)
}

func assertB081Contains(t *testing.T, path, content, required string) {
	t.Helper()
	if !strings.Contains(content, required) {
		t.Errorf("%s: missing sprint-scope contract %q", path, required)
	}
}

func assertB081MismatchStopsBeforeDispatch(t *testing.T, workflow, mismatchGuard string) {
	t.Helper()
	guardAt := strings.Index(workflow, mismatchGuard)
	if guardAt < 0 {
		t.Errorf("workflow is missing mismatch guard %q", mismatchGuard)
		return
	}

	guardBody := workflow[guardAt:]
	noDispatchAt := strings.Index(guardBody, "do not dispatch an entity")
	dispatchAt := strings.Index(guardBody, "/shark-rider run {ENTITY_KEY}")
	if noDispatchAt < 0 || dispatchAt < 0 || noDispatchAt > dispatchAt {
		t.Errorf("workflow must stop without dispatch after mismatch guard %q", mismatchGuard)
	}
}
