package contracts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestB081RunSprintWorkflowMirrorsAndValidatesSelection(t *testing.T) {
	canonicalPath := "internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint.md"
	hostPath := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	canonicalWorkflow := readF12Authored(t, canonicalPath)
	hostWorkflow := readF12Authored(t, hostPath)

	require.Equal(t, canonicalWorkflow, hostWorkflow, "embedded and host sprint workflows must stay aligned")
	requireF12Contains(t, canonicalWorkflow, canonicalPath,
		"NEXT_CMD = \"shark plan {SPRINT_KEY} --json\"",
		"NEXT_CMD = \"shark plan {SPRINT_KEY} --agent={AGENT_FILTER} --json\"",
		"SPRINT_SELECTION.action == \"select_item\"",
		"SPRINT_SELECTION.action == \"parallel_candidates\"",
		"SELECTED_ENTITY = SPRINT_SELECTION.entities[0]  # first item in selector order",
		"SELECTED_ENTITY.key is missing, empty, or whitespace-only")
	requireF12Excludes(t, canonicalWorkflow, canonicalPath, "shark sprint next {SPRINT_KEY}")
}

func TestB081RunSprintMalformedAndPausedSelectionStopBeforeDispatch(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)
	dispatch := "/shark-rider run {ENTITY_KEY}"

	requireF12Ordered(t, workflow, path,
		"if the response is malformed or SPRINT_SELECTION.sprint_key is missing:",
		"stop the pull-loop; do not dispatch an entity",
		dispatch)
	requireF12Ordered(t, workflow, path,
		"if SPRINT_SELECTION.action == \"pause\":",
		"No dispatchable entity was selected; sprint drain was not verified.",
		"SELECTION_PAUSED = true",
		"break",
		dispatch)
}

func TestB081RunSprintInvalidSelectionShapesStopBeforeDispatch(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)
	dispatch := "/shark-rider run {ENTITY_KEY}"
	cases := []struct {
		name    string
		markers []string
	}{
		{
			name: "empty parallel candidates",
			markers: []string{
				"if SPRINT_SELECTION.entities is empty:",
				"report unexpected output, include \"sprint drain was not verified\", and stop the pull-loop; do not dispatch an entity",
				dispatch,
			},
		},
		{
			name: "unknown action",
			markers: []string{
				"report the unexpected selection action and stop the pull-loop; do not dispatch an entity",
				dispatch,
			},
		},
		{
			name: "missing selected entity",
			markers: []string{
				"if SELECTED_ENTITY is null:",
				"report null selected entity and stop the pull-loop; do not dispatch an entity",
				dispatch,
			},
		},
		{
			name: "missing selected entity key",
			markers: []string{
				"if SELECTED_ENTITY.key is missing, empty, or whitespace-only:",
				"report the selected entity key is invalid and stop the pull-loop; do not dispatch an entity",
				dispatch,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			requireF12Ordered(t, workflow, path, testCase.markers...)
		})
	}
}

func TestB081RunSprintIdentityMismatchesStopBeforeDispatch(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)
	dispatch := "/shark-rider run {ENTITY_KEY}"

	requireF12Ordered(t, workflow, path,
		"if SPRINT_SELECTION.sprint_key != SPRINT_KEY:",
		"report the selected sprint key mismatch and stop the pull-loop without dispatching an entity",
		dispatch)
	requireF12Ordered(t, workflow, path,
		"if SELECTED_ENTITY.sprint_key != SPRINT_KEY:",
		"report the selected entity sprint key mismatch and stop the pull-loop without dispatching an entity",
		dispatch)
}

func TestB081RunSprintSummaryDoesNotClaimCompletionWithoutProof(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)

	requireF12Contains(t, workflow, path,
		"[Sprint {SPRINT_KEY}] Pull-loop stopped.",
		"No dispatchable entity was selected; sprint drain was not verified.")
	requireF12Excludes(t, workflow, path, "[Sprint {SPRINT_KEY}] Pull-loop complete.")
}

func TestB081RunSprintSkillsAndPromptUseScopedSelector(t *testing.T) {
	for _, path := range []string{
		"internal/sharkdata/default_data/skills/sprint-execution/SKILL.md",
		"skills/shark-rider/skills/sprint-execution/SKILL.md",
	} {
		requireF12Contains(t, readF12Authored(t, path), path, "shark plan S### --json")
	}

	promptPath := "internal/sharkdata/default_data/prompts/sprint/active.md"
	requireF12Contains(t, readF12Authored(t, promptPath), promptPath, "shark plan {{.id}} --json")
	verbPath := "skills/shark-rider/verbs/run-sprint.md"
	requireF12Contains(t, readF12Authored(t, verbPath), verbPath, "shark plan S###")
}

func TestB081RunSprintCLIReferenceUsesKeylessSyntax(t *testing.T) {
	path := "docs/cli-reference/sprint-commands.md"
	cliReference := readF12Authored(t, path)
	requireF12Excludes(t, cliReference, path, "shark sprint next [sprint-key]", "shark sprint next S024")
	requireF12Contains(t, cliReference, path, "shark sprint next [flags]")
}
