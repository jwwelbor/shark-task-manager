package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a small test helper.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

const taskStepsYAML = `
version: "1.0"
start: todo
steps:
  todo:
    phase: planning
    action: advance_status
    outcomes: { pass: in_dev }
  in_dev:
    phase: development
    action: spawn_agent
    agent: developer
    skills: [implementation]
    prompt: task/in_dev.md
    outcomes: { pass: done, fail: todo, blocked: blocked }
  blocked:
    phase: blocked
    parking: true
  done:
    phase: done
    terminal: true
`

const featureStepsYAML = `
version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: active }
  active:
    phase: development
    aggregates_from: tasks
    outcomes: { pass: completed, fail: draft, blocked: on_hold }
  on_hold:
    phase: paused
    parking: true
  completed:
    phase: done
    terminal: true
`

func TestIsWorkflowIndex(t *testing.T) {
	idx, ok := isWorkflowIndex([]byte("entities:\n  task: workflow/task.yaml\n"))
	if !ok {
		t.Fatal("expected index detection")
	}
	if idx.Entities["task"] != "workflow/task.yaml" {
		t.Errorf("entities = %v", idx.Entities)
	}

	// A regular per-entity workflow file is NOT an index.
	if _, ok := isWorkflowIndex([]byte(taskStepsYAML)); ok {
		t.Error("steps workflow misdetected as index")
	}
}

func TestLoadWorkflowIndexFile_RelativePaths(t *testing.T) {
	bundle := t.TempDir()
	writeFile(t, filepath.Join(bundle, "workflow", "task.yaml"), taskStepsYAML)
	writeFile(t, filepath.Join(bundle, "workflow", "feature.yaml"), featureStepsYAML)
	indexPath := filepath.Join(bundle, "workflow.yaml")
	writeFile(t, indexPath, "entities:\n  task: workflow/task.yaml\n  feature: workflow/feature.yaml\n")

	mlw, isIndex, err := LoadWorkflowIndexFile(indexPath)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if !isIndex {
		t.Fatal("expected isIndex=true")
	}
	if mlw.Task == nil || mlw.Feature == nil {
		t.Fatalf("task/feature not loaded: task=%v feature=%v", mlw.Task, mlw.Feature)
	}
	if !mlw.Task.HasSteps() {
		t.Error("task config should be route-based")
	}
	// Bundle root drives prompt resolution.
	if mlw.TemplateDirectory == nil || *mlw.TemplateDirectory != filepath.Join(bundle, "prompts") {
		t.Errorf("TemplateDirectory = %v, want %s/prompts", mlw.TemplateDirectory, bundle)
	}
	// Derived routing works.
	if target, ok := mlw.Task.ResolveOutcome("in_dev", "fail"); !ok || target != "todo" {
		t.Errorf("ResolveOutcome(in_dev,fail) = (%q,%v)", target, ok)
	}
}

func TestLoadWorkflowIndexFile_AbsolutePath(t *testing.T) {
	// Shared bundle lives in a separate directory; an absolute entity path
	// points at a workflow file outside the index's own tree.
	shared := t.TempDir()
	writeFile(t, filepath.Join(shared, "task.yaml"), taskStepsYAML)

	bundle := t.TempDir()
	indexPath := filepath.Join(bundle, "workflow.yaml")
	writeFile(t, indexPath, "entities:\n  task: "+filepath.Join(shared, "task.yaml")+"\n")

	mlw, isIndex, err := LoadWorkflowIndexFile(indexPath)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if !isIndex || mlw.Task == nil {
		t.Fatalf("absolute-path entity not loaded: isIndex=%v task=%v", isIndex, mlw.Task)
	}
}

func TestLoadWorkflowIndexFile_OverrideWins(t *testing.T) {
	bundle := t.TempDir()
	writeFile(t, filepath.Join(bundle, "workflow", "task.yaml"), taskStepsYAML)
	// Override changes the start step to prove it takes precedence.
	override := `
version: "1.0"
start: backlog
steps:
  backlog:
    phase: planning
    action: advance_status
    outcomes: { pass: done }
  done:
    phase: done
    terminal: true
`
	writeFile(t, filepath.Join(bundle, "overrides", "workflow", "task.yaml"), override)
	indexPath := filepath.Join(bundle, "workflow.yaml")
	writeFile(t, indexPath, "entities:\n  task: workflow/task.yaml\n")

	mlw, _, err := LoadWorkflowIndexFile(indexPath)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if _, ok := mlw.Task.GetStep("backlog"); !ok {
		t.Error("override workflow should have replaced base (expected 'backlog' step)")
	}
	if _, ok := mlw.Task.GetStep("in_dev"); ok {
		t.Error("base 'in_dev' step should be gone after override replacement")
	}
}

func TestLoadWorkflowIndexFile_NotAnIndex(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "task.yaml")
	writeFile(t, p, taskStepsYAML)
	_, isIndex, err := LoadWorkflowIndexFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isIndex {
		t.Error("a plain workflow file should not be treated as an index")
	}
}

func TestLoadWorkflowIndexFile_UnknownEntity(t *testing.T) {
	bundle := t.TempDir()
	indexPath := filepath.Join(bundle, "workflow.yaml")
	writeFile(t, indexPath, "entities:\n  gadget: workflow/gadget.yaml\n")
	_, isIndex, err := LoadWorkflowIndexFile(indexPath)
	if !isIndex {
		t.Error("expected isIndex=true even on error")
	}
	if err == nil {
		t.Error("expected error for unknown entity")
	}
}
