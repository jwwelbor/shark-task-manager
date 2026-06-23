package workflow

import (
	"strings"
	"testing"
)

func validRouteBasedConfig() *WorkflowConfig {
	cfg := &WorkflowConfig{
		Start: "todo",
		Steps: map[string]*Step{
			"todo": {Phase: "planning", Action: "advance_status",
				Outcomes: map[string]string{"pass": "qa", "fail": "todo", "blocked": "blocked"}},
			"qa": {Phase: "qa", Action: "spawn_agent", Agent: "qa", Skills: []string{"quality"}, Prompt: "task/qa.md",
				Outcomes: map[string]string{"pass": "done", "fail": "todo", "blocked": "blocked"}},
			"blocked": {Phase: "blocked", Parking: true},
			"done":    {Phase: "done", Terminal: true},
		},
	}
	cfg.DeriveLegacy()
	return cfg
}

func TestValidateWorkflow_RouteBased_Valid(t *testing.T) {
	if err := ValidateWorkflow(validRouteBasedConfig()); err != nil {
		t.Errorf("expected valid route-based workflow, got error: %v", err)
	}
}

func TestValidateWorkflow_RouteBased_MissingStart(t *testing.T) {
	cfg := validRouteBasedConfig()
	cfg.Start = "ghost"
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "start step") {
		t.Errorf("expected missing-start-step error, got %v", err)
	}
}

func TestValidateWorkflow_RouteBased_MissingCoreOutcome(t *testing.T) {
	cfg := &WorkflowConfig{
		Start: "todo",
		Steps: map[string]*Step{
			// 'todo' is workable but only defines 'pass' — missing fail/blocked.
			"todo": {Phase: "planning", Outcomes: map[string]string{"pass": "done"}},
			"done": {Phase: "done", Terminal: true},
		},
	}
	cfg.DeriveLegacy()
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "outcome") {
		t.Errorf("expected missing-core-outcome error, got %v", err)
	}
}

func TestValidateWorkflow_RouteBased_AliasCollision(t *testing.T) {
	cfg := &WorkflowConfig{
		Start: "todo",
		Steps: map[string]*Step{
			"todo": {Phase: "planning", Aliases: []string{"dup"},
				Outcomes: map[string]string{"pass": "qa", "fail": "todo", "blocked": "blocked"}},
			"qa": {Phase: "qa", Aliases: []string{"dup"},
				Outcomes: map[string]string{"pass": "done", "fail": "todo", "blocked": "blocked"}},
			"blocked": {Phase: "blocked", Parking: true},
			"done":    {Phase: "done", Terminal: true},
		},
	}
	cfg.DeriveLegacy()
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "alias") {
		t.Errorf("expected alias-collision error, got %v", err)
	}
}

func TestValidateWorkflow_LegacyUnaffected(t *testing.T) {
	// A legacy status_flow workflow must not trip route-based rules.
	cfg := &WorkflowConfig{
		StatusFlow: map[string][]string{
			"todo": {"done"},
			"done": {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"todo": {Phase: "planning"},
			"done": {Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:    {"todo"},
			CompleteStatusKey: {"done"},
		},
	}
	if err := ValidateWorkflow(cfg); err != nil {
		t.Errorf("legacy workflow should pass; got %v", err)
	}
}
