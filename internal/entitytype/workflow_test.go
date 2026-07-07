package entitytype

import "testing"

func TestNormalizeWorkflowLevel_Aliases(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"epic", WorkflowEpic},
		{"EPICS", WorkflowEpic},
		{"feature", WorkflowFeature},
		{"tasks", WorkflowTask},
		{"sprint", WorkflowSprint},
		{"bugs", WorkflowBug},
		{"change", WorkflowChange},
		{"changes", WorkflowChange},
		{"change_card", WorkflowChange},
		{"change-card", WorkflowChange},
		{"change_cards", WorkflowChange},
		{"change-cards", WorkflowChange},
		{"changecard", WorkflowChange},
		{"tech_debt", WorkflowTechDebt},
		{"tech-debt", WorkflowTechDebt},
		{"tech_debts", WorkflowTechDebt},
		{"tech-debts", WorkflowTechDebt},
		{"techdebt", WorkflowTechDebt},
		{"td", WorkflowTechDebt},
		{" TASKS ", WorkflowTask},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := NormalizeWorkflowLevel(tt.raw)
			if !ok {
				t.Fatalf("NormalizeWorkflowLevel(%q) returned ok=false", tt.raw)
			}
			if got != tt.want {
				t.Fatalf("NormalizeWorkflowLevel(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNormalizeWorkflowLevel_Unknown(t *testing.T) {
	if got, ok := NormalizeWorkflowLevel("idea"); ok {
		t.Fatalf("NormalizeWorkflowLevel(idea) = (%q, true), want ok=false because idea has no workflow slot", got)
	}
	if got := WorkflowLevelOrSelf("idea"); got != "idea" {
		t.Fatalf("WorkflowLevelOrSelf(idea) = %q, want idea", got)
	}
}
