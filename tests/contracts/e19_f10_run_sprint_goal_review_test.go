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
