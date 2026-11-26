package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// Downstream admission compares pointer and summary exactly, so the worker must
// receive the approved entries verbatim.
func TestAppendApprovedBugPlanEmitsApprovedEvidence(t *testing.T) {
	out := AppendApprovedBugPlan("prompt", ApprovedPlan{
		Pointer: "docs/plan/bugs/B1.md",
		Bytes:   []byte("plan"),
		Evidence: []workercontrol.EvidenceRef{
			{Kind: "bug_repair_plan", Pointer: "docs/plan/bugs/B1.md", Summary: "sha256:ABC"},
			{Kind: "bug_review_baseline", Pointer: "0123456789012345678901234567890123456789"},
		},
	})
	assert.Contains(t, out, `{"kind":"bug_repair_plan","pointer":"docs/plan/bugs/B1.md","summary":"sha256:ABC"}`)
	assert.Contains(t, out, `{"kind":"bug_review_baseline","pointer":"0123456789012345678901234567890123456789"}`)

	withoutEvidence := AppendApprovedBugPlan("prompt", ApprovedPlan{
		Pointer: "docs/plan/bugs/B1.md",
		Bytes:   []byte("plan"),
	})
	assert.NotContains(t, withoutEvidence, "For the outer `evidence` array")
}
