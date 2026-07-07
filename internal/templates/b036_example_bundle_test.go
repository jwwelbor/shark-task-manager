package templates

import (
	"strings"
	"testing"

	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestWorkflowIndexFixture_RendersAssessmentTemplate(t *testing.T) {
	fixture := testutil.WriteWorkflowIndexFixture(t)

	renderer, err := NewOrchestratorRenderer(fixture.ExpectedPromptsDir)
	if err != nil {
		t.Fatalf("NewOrchestratorRenderer(%s): %v", fixture.ExpectedPromptsDir, err)
	}

	out, err := renderer.Render("epic/assessment.tmpl", map[string]string{
		"id":        "E01",
		"title":     "Epic prompt coverage",
		"file_path": "docs/plan/epic.md",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "ROUTE FIXTURE EPIC ASSESSMENT") {
		t.Fatalf("rendered template missing expected assessment instructions:\n%s", out)
	}
}
