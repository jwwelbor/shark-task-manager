package action

import (
	"context"
	"testing"
)

// TestForEntity_ResolvesStatusInCorrectWorkflow proves the per-entity lookup
// disambiguates a status name (here, "completed") that exists in every
// entity's workflow but has different actions per entity.
func TestForEntity_ResolvesStatusInCorrectWorkflow(t *testing.T) {
	loader := func(_ string) (map[string]map[string]StatusActionData, error) {
		return map[string]map[string]StatusActionData{
			"task": {
				"completed": {OrchestratorAction: &OrchestratorAction{
					Action:              "archive",
					InstructionTemplate: "task/completed.md",
				}},
			},
			"feature": {
				"completed": {OrchestratorAction: &OrchestratorAction{
					Action:              "archive",
					InstructionTemplate: "feature/completed.md",
				}},
				"active": {OrchestratorAction: &OrchestratorAction{
					Action:              "cascade",
					InstructionTemplate: "feature/active.md",
				}},
			},
			"epic": {
				"active": {OrchestratorAction: &OrchestratorAction{
					Action:              "cascade",
					InstructionTemplate: "epic/active.md",
				}},
			},
		}, nil
	}

	svc, err := NewActionService("/tmp/test", loader)
	if err != nil {
		t.Fatalf("NewActionService: %v", err)
	}

	ctx := context.Background()

	// Bare service must default to task entity.
	a, err := svc.GetStatusAction(ctx, "completed")
	if err != nil {
		t.Fatalf("bare GetStatusAction: %v", err)
	}
	if a == nil || a.InstructionTemplate != "task/completed.md" {
		t.Errorf("bare service expected task/completed.md, got %+v", a)
	}

	// ForEntity("feature") sees the feature-scoped action.
	featureSvc := svc.ForEntity("feature")
	a, err = featureSvc.GetStatusAction(ctx, "completed")
	if err != nil {
		t.Fatalf("feature GetStatusAction: %v", err)
	}
	if a == nil || a.InstructionTemplate != "feature/completed.md" {
		t.Errorf("feature view expected feature/completed.md, got %+v", a)
	}

	// ForEntity("feature").GetStatusActionPopulated returns the feature's
	// "active" action (cascade) — this is exactly the case the previous flat
	// namespace got wrong: "active" exists for both feature and epic but
	// "completed" is what would have collided.
	pop, err := featureSvc.GetStatusActionPopulated(ctx, "active", nil)
	if err != nil {
		t.Fatalf("feature GetStatusActionPopulated: %v", err)
	}
	if pop == nil || pop.Action != "cascade" {
		t.Errorf("feature active expected cascade action, got %+v", pop)
	}

	// Status absent in the entity workflow → StatusNotFoundError.
	_, err = svc.ForEntity("epic").GetStatusAction(ctx, "completed")
	if err == nil {
		t.Errorf("epic 'completed' should be NotFound (not present), got nil err")
	}
	var notFound *StatusNotFoundError
	if !errorsAs(err, &notFound) {
		t.Errorf("epic missing-status error should be *StatusNotFoundError, got %T: %v", err, err)
	}
}

func TestNormalizeEntityType_ChangeCardAliases(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		want       string
	}{
		{name: "singular_cli_alias", entityType: "change_card", want: "change"},
		{name: "hyphen_cli_alias", entityType: "change-card", want: "change"},
		{name: "plural_storage_alias", entityType: "change_cards", want: "change"},
		{name: "plural_hyphen_alias", entityType: "change-cards", want: "change"},
		{name: "canonical_change", entityType: "change", want: "change"},
		{name: "tech_debt_hyphen_alias", entityType: "tech-debt", want: "tech_debt"},
		{name: "other_entity", entityType: "feature", want: "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEntityType(tt.entityType); got != tt.want {
				t.Fatalf("NormalizeEntityType(%q) = %q, want %q", tt.entityType, got, tt.want)
			}
		})
	}
}

// TestForEntity_NormalizesChangeCardToChangeWorkflow covers the B034
// regression: key detection returns "change_card" for CC-###, but workflow
// action data is loaded under the canonical "change" slot.
func TestForEntity_NormalizesChangeCardToChangeWorkflow(t *testing.T) {
	loader := func(_ string) (map[string]map[string]StatusActionData, error) {
		return map[string]map[string]StatusActionData{
			"change": {
				"development": {OrchestratorAction: &OrchestratorAction{
					Action:              "spawn_agent",
					InstructionTemplate: "change/development.md",
				}},
			},
		}, nil
	}

	svc, err := NewActionService("/tmp/test", loader)
	if err != nil {
		t.Fatalf("NewActionService: %v", err)
	}

	a, err := svc.ForEntity("change_card").GetStatusAction(context.Background(), "development")
	if err != nil {
		t.Fatalf("change_card alias GetStatusAction: %v", err)
	}
	if a == nil || a.InstructionTemplate != "change/development.md" {
		t.Fatalf("change_card alias expected change/development.md, got %+v", a)
	}
}

// errorsAs is a tiny dependency-free shim so this file doesn't pull in errors.As
// (kept inlined to mirror the test style of the surrounding package).
func errorsAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	if e, ok := target.(**StatusNotFoundError); ok {
		if nf, ok := err.(*StatusNotFoundError); ok {
			*e = nf
			return true
		}
	}
	return false
}
