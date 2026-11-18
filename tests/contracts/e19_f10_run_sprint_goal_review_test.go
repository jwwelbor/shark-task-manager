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
			"omit AGENT_FILTER",
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
		drained := strings.Index(branch, `LOOP_OUTCOME = "drained"`)
		if allTerminal == -1 || drained == -1 || allTerminal > drained {
			t.Fatalf("%s: only a verified all-terminal read may report drained", path)
		}
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
		if closeStart == -1 {
			t.Fatalf("%s: close gate section is missing", path)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	return runSprintWorkflowPaths(root)
}

func sprintWorkflowText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func sprintEmptySelectionBranch(t *testing.T, path string) string {
	t.Helper()
	source := sprintWorkflowText(t, path)
	start := strings.Index(source, "if NEXT_ENTITY is null or empty")
	if start == -1 {
		t.Fatalf("%s: empty-selector branch is missing", path)
	}
	end := strings.Index(source[start:], "ENTITY_KEY = NEXT_ENTITY.key")
	if end == -1 {
		t.Fatalf("%s: empty-selector branch does not return to normal dispatch", path)
	}
	return source[start : start+end]
}

func sprintTextBetween(t *testing.T, path, source, start, end string) string {
	t.Helper()
	startAt := strings.Index(source, start)
	if startAt == -1 {
		t.Fatalf("%s: missing contract marker %q", path, start)
	}
	endAt := strings.Index(source[startAt+len(start):], end)
	if endAt == -1 {
		t.Fatalf("%s: missing end marker %q after %q", path, end, start)
	}
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
