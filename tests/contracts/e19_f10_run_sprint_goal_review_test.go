// Package contracts checks that the shipped shark-rider run-sprint workflow
// text never calls `shark sprint close` without first walking the operator
// through a Sprint Goal Review submission (REQ-F10-006). CloseSprintWithCarryover
// enforces the accepted-review gate server-side, but the automation script
// itself must not blindly call close and hit that rejection on every sprint —
// see finding #4 of the E19 code-review fix plan.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// runSprintWorkflowPaths lists every shipped copy of the run-sprint workflow
// that gates `shark sprint close` on user confirmation. Both the embedded
// canonical bundle (internal/sharkdata/default_data) and the repo-tracked
// skills/ copy must agree — see feedback_embedded_sharkdata_canonical
// memory: the embedded copy is the source of truth, the repo copy is a
// deployed mirror kept identical.
func runSprintWorkflowPaths(root string) []string {
	return []string{
		filepath.Join(root, "..", "..", "internal", "sharkdata", "default_data", "skills", "sprint-execution", "workflows", "run-sprint.md"),
		filepath.Join(root, "..", "..", "skills", "shark-rider", "skills", "sprint-execution", "workflows", "run-sprint.md"),
	}
}

func TestRunSprintWorkflowRequiresGoalReviewBeforeEveryClose(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range runSprintWorkflowPaths(root) {
		path := path
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(path)))+"/"+filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			source := string(content)

			if !strings.Contains(source, "shark sprint goal-review") {
				t.Fatalf("%s: workflow never mentions `shark sprint goal-review`; a close attempt with no accepted review returns the sprint to active and creates no completion row (REQ-F10-006)", path)
			}

			// Every fenced ```bash invocation of `shark sprint close`
			// (the actual close call sites, as opposed to prose mentions
			// elsewhere in the doc — e.g. the "no" branch or the error
			// table) must be preceded, somewhere earlier in the file, by a
			// goal-review mention.
			reviewIndex := strings.Index(source, "shark sprint goal-review")
			if reviewIndex == -1 {
				t.Fatalf("%s: `shark sprint goal-review` not found", path)
			}
			closeInvocations := 0
			cursor := 0
			for {
				fenceStart := strings.Index(source[cursor:], "```bash")
				if fenceStart == -1 {
					break
				}
				fenceStart += cursor
				blockEnd := len(source)
				if closeFence := strings.Index(source[fenceStart+len("```bash"):], "```"); closeFence != -1 {
					blockEnd = fenceStart + len("```bash") + closeFence
				}
				block := source[fenceStart:blockEnd]
				if strings.Contains(block, "shark sprint close") {
					closeInvocations++
					if fenceStart < reviewIndex {
						t.Fatalf("%s: found a `shark sprint close` invocation not preceded by `shark sprint goal-review` in the workflow text", path)
					}
				}
				cursor = fenceStart + len("```bash")
			}
			if closeInvocations == 0 {
				t.Fatalf("%s: workflow no longer calls `shark sprint close` in a bash block at all; test is stale", path)
			}
		})
	}
}

func TestRunSprintWorkflowChecksTheUnfilteredAssignmentSet(t *testing.T) {
	for _, path := range sprintWorkflowCopies(t) {
		branch := sprintEmptySelectionBranch(t, path)
		for _, required := range []string{
			"shark sprint backlog {SPRINT_KEY} --view=ordered --all --json",
			"omits AGENT_FILTER",
			"configured terminal status",
			"AGENT_FILTER is set",
			"role-filtered slice",
			"nonterminal assignments remain",
		} {
			if !strings.Contains(branch, required) {
				t.Errorf("%s: empty-selector branch must include %q", path, required)
			}
		}
	}
}

func TestRunSprintWorkflowPausesForEachTerminalityEvidenceFailure(t *testing.T) {
	guards := []struct{ name, start, end string }{
		{
			name:  "failed or malformed backlog read",
			start: "if the command fails or its JSON is malformed:",
			end:   "UNFINISHED = []",
		},
		{
			name:  "assignment count mismatch",
			start: "if BACKLOG.total_count does not match the returned item count:",
			end:   "if TERMINALITY_UNVERIFIED is true:",
		},
		{
			name:  "unresolved terminality",
			start: "if TERMINALITY_UNVERIFIED is true:",
			end:   "if UNFINISHED is not empty:",
		},
		{
			name:  "unfinished assignments",
			start: "if UNFINISHED is not empty:",
			end:   "A successful all-status read",
		},
	}
	for _, path := range sprintWorkflowCopies(t) {
		branch := sprintEmptySelectionBranch(t, path)
		unresolved := sprintTextBetween(
			t,
			path,
			branch,
			"if its workflow, current status, or terminal marker cannot be resolved:",
			"else if its current status is not marked terminal",
		)
		if !strings.Contains(unresolved, "TERMINALITY_UNVERIFIED = true") {
			t.Errorf("%s: unresolved status or terminal marker must set TERMINALITY_UNVERIFIED", path)
		}
		for _, guard := range guards {
			t.Run(filepath.Base(filepath.Dir(filepath.Dir(path)))+"/"+guard.name, func(t *testing.T) {
				assertSprintGuardPauses(t, path, branch, guard.start, guard.end)
			})
		}
	}
}

func TestRunSprintWorkflowDrainsOnlyAfterVerifiedTerminality(t *testing.T) {
	for _, path := range sprintWorkflowCopies(t) {
		branch := sprintEmptySelectionBranch(t, path)
		allTerminal := strings.Index(branch, "A successful all-status read with no unfinished assignments proves")
		emptyInventory := strings.Index(branch, "including an empty assignment set")
		drained := strings.Index(branch, `LOOP_OUTCOME = "drained"`)
		require.NotEqual(t, -1, allTerminal, "%s: all-terminal proof marker is missing", path)
		require.NotEqual(t, -1, emptyInventory, "%s: an empty assignment set must be a valid drained case", path)
		require.NotEqual(t, -1, drained, "%s: drained outcome marker is missing", path)
		require.Less(t, allTerminal, drained, "%s: only a verified all-terminal read may report drained", path)
		require.Less(t, emptyInventory, drained, "%s: empty inventory must remain within the verified terminality proof", path)
		if strings.Contains(branch, "break  # backlog drained") {
			t.Errorf("%s: empty selector result must not directly report drained", path)
		}
		for _, check := range []string{"if UNFINISHED is not empty:", "if TERMINALITY_UNVERIFIED is true:", "BACKLOG.total_count does not match"} {
			if index := strings.Index(branch, check); index == -1 || index > drained {
				t.Errorf("%s: %q must be checked before the drained outcome", path, check)
			}
		}
	}
}

func TestRunSprintWorkflowListsOnlyVerifiedAssignmentsAtClose(t *testing.T) {
	for _, path := range sprintWorkflowCopies(t) {
		source := strings.Join(strings.Fields(sprintWorkflowText(t, path)), " ")
		closeStart := strings.Index(source, "## Step 4: Close Gate")
		require.NotEqual(t, -1, closeStart, "%s: close gate section is missing", path)
		closeGate := source[closeStart:]
		for _, required := range []string{
			"complete, successful all-status read",
			"inventory is unavailable or incomplete",
			"do not imply the full remaining set is known",
		} {
			if !strings.Contains(closeGate, required) {
				t.Errorf("%s: close gate must state %q", path, required)
			}
		}
	}
}

func TestRunSprintWorkflowKeepsCappedAndMalformedSelectorOutcomesUnverified(t *testing.T) {
	for _, path := range sprintWorkflowCopies(t) {
		source := sprintWorkflowText(t, path)

		capBranch := sprintTextBetween(
			t,
			path,
			source,
			`if LOOP_OUTCOME == "running" and ITERATION >= MAX_ITERATIONS and SELECTED_ENTITY is not null:`,
			"\n```",
		)
		require.Contains(t, capBranch, `LOOP_OUTCOME = "capped"`, "%s: iteration cap must set the capped outcome", path)
		require.Contains(t, capBranch, "Proceed to Step 3 (still report and prompt; do not skip close gate)", "%s: capped outcome must continue to reporting and the close gate", path)
		require.NotContains(t, capBranch, `LOOP_OUTCOME = "drained"`, "%s: reaching the iteration cap cannot report drained", path)

		errorHandling := sprintTextBetween(t, path, source, "## Error Handling", "## Idempotency")
		malformedNext := sprintTextBetween(
			t,
			path,
			errorHandling,
			"| `shark plan {SPRINT_KEY}` returns malformed JSON, a non-object JSON root, an unknown action, or an invalid selection shape |",
			"| Sprint selection or selected entity has a different `sprint_key` than `SPRINT_KEY` |",
		)
		require.Contains(t, malformedNext, "Print the raw response", "%s: malformed selector output must be surfaced", path)
		require.Contains(t, malformedNext, `LOOP_OUTCOME = "paused"`, "%s: malformed selector output must pause", path)
		require.Contains(t, malformedNext, "proceed to Step 3 without treating the sprint as drained", "%s: malformed selector output must not be treated as a drained backlog", path)
		require.NotContains(t, malformedNext, `LOOP_OUTCOME = "drained"`, "%s: malformed selector output cannot report drained", path)

		closeGate := strings.Join(strings.Fields(sprintTextBetween(t, path, source, "## Step 4: Close Gate", "Ask the user:")), " ")
		require.Contains(t, closeGate, "If `LOOP_OUTCOME` is `paused` or `capped`", "%s: close gate must cover paused and capped outcomes", path)
		require.Contains(t, closeGate, "the sprint is not verified drained", "%s: close gate must not imply paused or capped sprints are drained", path)
		require.Contains(t, closeGate, "List assignment keys and statuses only when they come from a complete, successful all-status read", "%s: close gate must condition assignment claims on a complete inventory", path)
		require.Contains(t, closeGate, "do not imply the full remaining set is known", "%s: close gate must not imply an incomplete inventory is complete", path)
	}
}

func TestRunSprintWorkflowResolvesMetadataOncePerEntityType(t *testing.T) {
	for _, path := range sprintWorkflowCopies(t) {
		branch := sprintEmptySelectionBranch(t, path)
		for _, required := range []string{
			"For every distinct entity type in BACKLOG.items:",
			"workflow {ENTITY_TYPE} --json` once",
			"reuse that workflow metadata for every matching item",
		} {
			if !strings.Contains(branch, required) {
				t.Errorf("%s: terminality check must include %q", path, required)
			}
		}
		if strings.Contains(branch, "/shark-rider workflow {ITEM.key}") {
			t.Errorf("%s: workflow metadata must not be fetched once per assigned item", path)
		}
	}
}

func sprintWorkflowCopies(t *testing.T) []string {
	t.Helper()
	root, err := os.Getwd()
	require.NoError(t, err, "get working directory")
	return runSprintWorkflowPaths(root)
}

func sprintWorkflowText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "read workflow %s", path)
	return string(content)
}

func sprintEmptySelectionBranch(t *testing.T, path string) string {
	t.Helper()
	source := sprintWorkflowText(t, path)
	return sprintTextBetween(t, path, source,
		"if SPRINT_SELECTION.action == \"pause\":",
		"if SPRINT_SELECTION.action == \"select_item\":")
}

func sprintTextBetween(t *testing.T, path, source, start, end string) string {
	t.Helper()
	startAt := strings.Index(source, start)
	require.NotEqual(t, -1, startAt, "%s: missing contract marker %q", path, start)
	endAt := strings.Index(source[startAt+len(start):], end)
	require.NotEqual(t, -1, endAt, "%s: missing end marker %q after %q", path, end, start)
	return source[startAt : startAt+len(start)+endAt]
}

func assertSprintGuardPauses(t *testing.T, path, branch, guard, end string) {
	t.Helper()
	block := sprintTextBetween(t, path, branch, guard, end)
	paused := strings.Index(block, `LOOP_OUTCOME = "paused"`)
	stop := strings.Index(block, "break")
	drained := strings.Index(branch, `LOOP_OUTCOME = "drained"`)
	guardAt := strings.Index(branch, guard)
	pauseBeforeStop := paused >= 0 && stop >= 0 && paused < stop
	pauseBeforeDrain := guardAt >= 0 && drained >= 0 && guardAt+paused < drained
	if !pauseBeforeStop || !pauseBeforeDrain {
		t.Errorf("%s: %q must set paused and break before the drained outcome", path, guard)
	}
}

func TestRunSprintWorkflowCopiesStayIdentical(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	paths := runSprintWorkflowPaths(root)
	canonical, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatalf("read canonical %s: %v", paths[0], err)
	}
	mirror, err := os.ReadFile(paths[1])
	if err != nil {
		t.Fatalf("read mirror %s: %v", paths[1], err)
	}
	if string(canonical) != string(mirror) {
		t.Fatalf("embedded canonical run-sprint.md and the repo skills/ mirror have diverged; edit the embedded canonical (internal/sharkdata/default_data) and copy it over the repo mirror")
	}
}
