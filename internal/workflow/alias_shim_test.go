package workflow

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func aliasService() *Service {
	cfg := &config.WorkflowConfig{
		Start: "refinement",
		Steps: map[string]*config.Step{
			"refinement": {Phase: "refinement", Aliases: []string{"ready_for_refinement", "in_refinement"},
				Outcomes: map[string]string{"pass": "qa", "fail": "refinement", "blocked": "blocked"}},
			"qa": {Phase: "qa", Aliases: []string{"ready_for_qa", "in_qa"},
				Outcomes: map[string]string{"pass": "completed", "fail": "refinement", "blocked": "blocked"}},
			"blocked":   {Phase: "blocked", Parking: true},
			"completed": {Phase: "done", Terminal: true},
		},
	}
	// The loader derives legacy maps in real use; do it here so the legacy-map
	// readers in the service (StatusFlow lookups) see the steps.
	cfg.DeriveLegacy()
	return &Service{workflow: cfg, level: LevelTask}
}

func TestService_NormalizeStatus_AliasShim(t *testing.T) {
	svc := aliasService()
	cases := []struct{ in, want string }{
		{"ready_for_qa", "qa"},
		{"in_refinement", "refinement"},
		{"qa", "qa"},
	}
	for _, c := range cases {
		if got := svc.NormalizeStatus(c.in); got != c.want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestService_IsValidStatus_AcceptsAliases(t *testing.T) {
	svc := aliasService()
	if !svc.IsValidStatus("ready_for_qa") {
		t.Error("expected old alias 'ready_for_qa' to be accepted as valid")
	}
	if !svc.IsValidStatus("qa") {
		t.Error("expected step 'qa' to be valid")
	}
	if svc.IsValidStatus("totally_unknown") {
		t.Error("unknown status should be invalid")
	}
}

func TestService_IsValidTransition_AliasAware(t *testing.T) {
	svc := aliasService()
	// refinement -> qa is valid (pass outcome target). Old names on both sides
	// must resolve and still validate.
	if !svc.IsValidTransition("in_refinement", "ready_for_qa") {
		t.Error("expected alias-resolved transition in_refinement -> ready_for_qa to be valid")
	}
}

func TestService_ValidateTransition_AliasAware(t *testing.T) {
	svc := aliasService()
	if err := svc.ValidateTransition("IN_REFINEMENT", "READY_FOR_QA"); err != nil {
		t.Fatalf("ValidateTransition should normalize aliases and case variants: %v", err)
	}
}

func TestService_ResolveAlias(t *testing.T) {
	svc := aliasService()
	if got := svc.ResolveAlias("ready_for_refinement"); got != "refinement" {
		t.Errorf("ResolveAlias = %q, want refinement", got)
	}
}
