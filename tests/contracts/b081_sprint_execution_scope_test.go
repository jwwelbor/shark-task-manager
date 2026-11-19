package contracts

import (
	"strings"
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
		"if the parsed response root is not a JSON object:",
		"SPRINT_SELECTION.action == \"select_item\"",
		"SPRINT_SELECTION.action == \"parallel_candidates\"",
		"SELECTED_ENTITY = SPRINT_SELECTION.entities[0]  # first item in selector order",
		"SELECTED_ENTITY.key is missing, empty, or whitespace-only")
	requireF12Excludes(t, canonicalWorkflow, canonicalPath, "shark sprint next {SPRINT_KEY}")
	requireF12Ordered(t, canonicalWorkflow, canonicalPath,
		"Parse JSON response as SPRINT_SELECTION.",
		"if the parsed response root is not a JSON object:",
		"if SPRINT_SELECTION.sprint_key is missing:",
		"if SPRINT_SELECTION.action == \"select_item\":")
}

func TestB081RunSprintSelectItemAssignsAndDispatchesSelectedKey(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)

	requireF12Ordered(t, workflow, path,
		"if SPRINT_SELECTION.action == \"select_item\":",
		"SELECTED_ENTITY = SPRINT_SELECTION.entity",
		"if SELECTED_ENTITY is null:",
		"if SELECTED_ENTITY.key is missing, empty, or whitespace-only:",
		"ENTITY_KEY = SELECTED_ENTITY.key",
		"/shark-rider run {ENTITY_KEY}")
}

func TestB081RunSprintMalformedAndPausedSelectionStopBeforeDispatch(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)
	dispatch := "/shark-rider run {ENTITY_KEY}"

	requireF12Ordered(t, workflow, path,
		"if the response is malformed:",
		"stop the pull-loop; do not dispatch an entity",
		"if the parsed response root is not a JSON object:",
		"if SPRINT_SELECTION.sprint_key is missing:",
		dispatch)
	rootGuardStart := strings.Index(workflow, "if the parsed response root is not a JSON object:")
	require.NotEqual(t, -1, rootGuardStart, "workflow must reject a non-object JSON root")
	requireF12Ordered(t, workflow[rootGuardStart:], path,
		"if the parsed response root is not a JSON object:",
		"stop the pull-loop; do not dispatch an entity",
		"if SPRINT_SELECTION.sprint_key is missing:",
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

func TestB081RunSprintCanonicalizesLowercaseInputBeforeSelection(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)

	requireF12Contains(t, workflow, path, "Case-insensitive (accept `s001` as `S001`).")
	requireF12Ordered(t, workflow, path,
		"shark get {SPRINT_KEY} --json",
		"`key` — canonical sprint key (use this for all subsequent calls)",
		"Set `SPRINT_KEY = key` before continuing so later identity checks use the canonical form.",
		"NEXT_CMD = \"shark plan {SPRINT_KEY} --json\"",
		"if SPRINT_SELECTION.sprint_key != SPRINT_KEY:")
}

func TestB081RunSprintIterationCapNoticeRequiresValidSelection(t *testing.T) {
	path := "skills/shark-rider/skills/sprint-execution/workflows/run-sprint.md"
	workflow := readF12Authored(t, path)
	capGuard := "if ITERATION >= MAX_ITERATIONS and SELECTED_ENTITY is not null:"

	requireF12Ordered(t, workflow, path,
		"SELECTED_ENTITY = SPRINT_SELECTION.entity",
		"ENTITY_KEY = SELECTED_ENTITY.key",
		"/shark-rider run {ENTITY_KEY}",
		capGuard,
		"Print cap notice:")
	loopStart := strings.Index(workflow, "while ITERATION < MAX_ITERATIONS:")
	require.NotEqual(t, -1, loopStart, "workflow must define the bounded pull-loop")
	requireF12Ordered(t, workflow[loopStart:], path,
		"while ITERATION < MAX_ITERATIONS:",
		"SELECTED_ENTITY = null",
		"if SPRINT_SELECTION.action == \"pause\":",
		"SELECTION_PAUSED = true",
		"break",
		capGuard)
	invalidKeyStart := strings.Index(workflow, "if SELECTED_ENTITY.key is missing, empty, or whitespace-only:")
	require.NotEqual(t, -1, invalidKeyStart, "workflow must validate the selected entity key")
	requireF12Ordered(t, workflow[invalidKeyStart:], path,
		"if SELECTED_ENTITY.key is missing, empty, or whitespace-only:",
		"SELECTED_ENTITY = null",
		"report the selected entity key is invalid and stop the pull-loop; do not dispatch an entity",
		capGuard)
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
	riderPath := "skills/shark-rider/SKILL.md"
	requireF12Contains(t, readF12Authored(t, riderPath), riderPath, "`shark plan S###` → `/shark-rider run` per entity")
}

func TestB081RunSprintCLIReferenceUsesKeylessSyntax(t *testing.T) {
	path := "docs/cli-reference/sprint-commands.md"
	cliReference := readF12Authored(t, path)
	requireF12Excludes(t, cliReference, path, "shark sprint next [sprint-key]", "shark sprint next S024")
	requireF12Contains(t, cliReference, path, "shark sprint next [flags]")
}
