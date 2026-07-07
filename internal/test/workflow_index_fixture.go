package test

import (
	"os"
	"path/filepath"
	"testing"
)

// WorkflowIndexFixture is a generated route-based workflow bundle for tests
// that need a workflow index with an explicit prompt root.
type WorkflowIndexFixture struct {
	BundleRoot         string
	WorkflowIndexPath  string
	ExpectedPromptsDir string
}

// WriteWorkflowIndexFixture writes a minimal multi-entity workflow index bundle
// with a dedicated prompts/ tree. Tests use it to verify that workflow index
// template_directory wiring is honored without depending on repo example files.
func WriteWorkflowIndexFixture(t *testing.T) WorkflowIndexFixture {
	t.Helper()

	bundleRoot := t.TempDir()
	workflowsDir := filepath.Join(bundleRoot, "workflow")
	promptsDir := filepath.Join(bundleRoot, "prompts")

	files := map[string]string{
		filepath.Join(bundleRoot, "workflow.yaml"): `template_directory: prompts
entities:
  epic: workflow/epic.yaml
  feature: workflow/feature.yaml
  task: workflow/task.yaml
  sprint: workflow/sprint.yaml
  bug: workflow/bug.yaml
  change: workflow/change.yaml
  tech_debt: workflow/tech_debt.yaml
`,
		filepath.Join(workflowsDir, "epic.yaml"): `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: assessment }
  assessment:
    phase: triage
    action: spawn_agent
    agent: researcher
    prompt: epic/assessment.tmpl
    outcomes: { pass: completed, fail: draft, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "feature.yaml"): `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: assessment }
  assessment:
    phase: triage
    action: spawn_agent
    agent: product-manager
    prompt: feature/assessment.tmpl
    outcomes: { pass: completed, fail: draft, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "task.yaml"): `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: development }
  development:
    phase: development
    action: spawn_agent
    agent: developer
    prompt: task/development.tmpl
    outcomes: { pass: completed, fail: draft, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "sprint.yaml"): `version: "1.0"
start: planning
steps:
  planning:
    phase: planning
    action: spawn_agent
    agent: planner
    prompt: sprint/planning.tmpl
    outcomes: { pass: active }
  active:
    phase: execution
    action: spawn_agent
    agent: facilitator
    prompt: sprint/active.tmpl
    outcomes: { pass: closing, blocked: blocked }
  closing:
    phase: review
    action: spawn_agent
    agent: facilitator
    prompt: sprint/closing.tmpl
    outcomes: { pass: completed }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "bug.yaml"): `version: "1.0"
start: draft
steps:
  draft:
    phase: triage
    action: advance_status
    outcomes: { pass: development }
  development:
    phase: development
    action: spawn_agent
    agent: developer
    prompt: bug/development.tmpl
    outcomes: { pass: code_review, fail: draft, blocked: blocked }
  code_review:
    phase: review
    action: spawn_agent
    agent: reviewer
    prompt: bug/code_review.tmpl
    outcomes: { pass: qa, fail: development, blocked: blocked }
  qa:
    phase: qa
    action: spawn_agent
    agent: qa
    prompt: bug/qa.tmpl
    outcomes: { pass: completed, fail: development, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "change.yaml"): `version: "1.0"
start: draft
steps:
  draft:
    phase: triage
    action: advance_status
    outcomes: { pass: development }
  development:
    phase: development
    action: spawn_agent
    agent: developer
    prompt: change/development.tmpl
    outcomes: { pass: code_review, fail: draft, blocked: blocked }
  code_review:
    phase: review
    action: spawn_agent
    agent: reviewer
    prompt: change/code_review.tmpl
    outcomes: { pass: completed, fail: development, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(workflowsDir, "tech_debt.yaml"): `version: "1.0"
start: identified
steps:
  identified:
    phase: triage
    action: advance_status
    outcomes: { pass: in_progress }
  in_progress:
    phase: development
    action: spawn_agent
    agent: developer
    prompt: tech_debt/in_progress.tmpl
    outcomes: { pass: code_review, fail: identified, blocked: blocked }
  code_review:
    phase: review
    action: spawn_agent
    agent: reviewer
    prompt: tech_debt/code_review.tmpl
    outcomes: { pass: resolved, fail: in_progress, blocked: blocked }
  blocked:
    phase: blocked
    action: pause
    parking: true
  resolved:
    phase: done
    action: archive
    terminal: true
`,
		filepath.Join(promptsDir, "epic", "assessment.tmpl"):       "ROUTE FIXTURE EPIC ASSESSMENT\n",
		filepath.Join(promptsDir, "feature", "assessment.tmpl"):    "ROUTE FIXTURE FEATURE ASSESSMENT\n",
		filepath.Join(promptsDir, "task", "development.tmpl"):      "ROUTE FIXTURE TASK DEVELOPMENT\n",
		filepath.Join(promptsDir, "sprint", "planning.tmpl"):       "ROUTE FIXTURE SPRINT PLANNING\n",
		filepath.Join(promptsDir, "sprint", "active.tmpl"):         "ROUTE FIXTURE SPRINT ACTIVE\n",
		filepath.Join(promptsDir, "sprint", "closing.tmpl"):        "ROUTE FIXTURE SPRINT CLOSING\n",
		filepath.Join(promptsDir, "bug", "development.tmpl"):       "ROUTE FIXTURE BUG DEVELOPMENT\n",
		filepath.Join(promptsDir, "bug", "code_review.tmpl"):       "ROUTE FIXTURE BUG CODE REVIEW\n",
		filepath.Join(promptsDir, "bug", "qa.tmpl"):                "ROUTE FIXTURE BUG QA\n",
		filepath.Join(promptsDir, "change", "development.tmpl"):    "ROUTE FIXTURE CHANGE DEVELOPMENT\n",
		filepath.Join(promptsDir, "change", "code_review.tmpl"):    "ROUTE FIXTURE CHANGE CODE REVIEW\n",
		filepath.Join(promptsDir, "tech_debt", "in_progress.tmpl"): "ROUTE FIXTURE TECH DEBT IN PROGRESS\n",
		filepath.Join(promptsDir, "tech_debt", "code_review.tmpl"): "ROUTE FIXTURE TECH DEBT CODE REVIEW\n",
	}

	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	return WorkflowIndexFixture{
		BundleRoot:         bundleRoot,
		WorkflowIndexPath:  filepath.Join(bundleRoot, "workflow.yaml"),
		ExpectedPromptsDir: promptsDir,
	}
}
