package action

import (
	"context"
	"strings"
	"testing"
)

// TestValidateEntityMap_QualifyKeys exercises the extracted validateEntityMap
// helper directly to prove that:
//   - missing actions and invalid actions are collected
//   - qualifyKeys=true prefixes non-default entity types ("feature:status")
//   - qualifyKeys=true leaves the default entity type ("task") unprefixed
//   - qualifyKeys=false always emits raw status names
//
// This is the safety net for TD-018: any future divergence between the two
// ValidateActions callers will be caught here.
func TestValidateEntityMap_QualifyKeys(t *testing.T) {
	// Shared fixture: one missing ready_for_* action plus one invalid action
	// (no InstructionTemplate -> OrchestratorAction.Validate fails).
	makeMap := func() map[string]StatusActionData {
		return map[string]StatusActionData{
			"ready_for_review": {OrchestratorAction: nil}, // missing
			"in_progress": {OrchestratorAction: &OrchestratorAction{
				Action: "implement",
				// InstructionTemplate intentionally empty -> invalid
			}},
			"completed": {OrchestratorAction: &OrchestratorAction{
				Action:              "archive",
				InstructionTemplate: "completed.md",
			}},
		}
	}

	tests := []struct {
		name            string
		entityType      string
		qualifyKeys     bool
		wantMissing     string
		wantInvalidKey  string
		wantWarningFrag string
	}{
		{
			name:            "qualifyKeys=true, non-default entity prefixes keys",
			entityType:      "feature",
			qualifyKeys:     true,
			wantMissing:     "feature:ready_for_review",
			wantInvalidKey:  "feature:in_progress",
			wantWarningFrag: "'feature:ready_for_review'",
		},
		{
			name:            "qualifyKeys=true, default entity (task) stays unqualified",
			entityType:      DefaultEntityType,
			qualifyKeys:     true,
			wantMissing:     "ready_for_review",
			wantInvalidKey:  "in_progress",
			wantWarningFrag: "'ready_for_review'",
		},
		{
			name:            "qualifyKeys=false emits raw status names even for non-default entity",
			entityType:      "feature",
			qualifyKeys:     false,
			wantMissing:     "ready_for_review",
			wantInvalidKey:  "in_progress",
			wantWarningFrag: "'ready_for_review'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:          true,
				MissingActions: []string{},
				InvalidActions: []InvalidAction{},
				Warnings:       []string{},
			}

			validateEntityMap(tt.entityType, makeMap(), tt.qualifyKeys, result)

			// Missing action key
			if len(result.MissingActions) != 1 || result.MissingActions[0] != tt.wantMissing {
				t.Errorf("MissingActions = %v, want [%q]", result.MissingActions, tt.wantMissing)
			}

			// Invalid action key
			if len(result.InvalidActions) != 1 || result.InvalidActions[0].Status != tt.wantInvalidKey {
				t.Errorf("InvalidActions[0].Status = %+v, want Status=%q",
					result.InvalidActions, tt.wantInvalidKey)
			}
			if len(result.InvalidActions) == 1 && result.InvalidActions[0].Error == "" {
				t.Error("InvalidActions[0].Error is empty, expected validation error message")
			}

			// Warning message format
			if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], tt.wantWarningFrag) {
				t.Errorf("Warnings = %v, want one entry containing %q",
					result.Warnings, tt.wantWarningFrag)
			}

			// The helper flips Valid=false on every invalid action.
			if result.Valid {
				t.Error("expected Valid=false after recording an invalid action")
			}
		})
	}
}

// TestValidateActions_BothCallersConsistent exercises the public ValidateActions
// surface on both the root service and a ForEntity view to confirm the
// extracted helper feeds both correctly.
func TestValidateActions_BothCallersConsistent(t *testing.T) {
	loader := func(_ string) (map[string]map[string]StatusActionData, error) {
		return map[string]map[string]StatusActionData{
			"task": {
				"ready_for_review": {OrchestratorAction: nil}, // missing -> warning, unprefixed
			},
			"feature": {
				"ready_for_qa": {OrchestratorAction: nil}, // missing -> warning, prefixed at root
				"completed": {OrchestratorAction: &OrchestratorAction{
					Action:              "archive",
					InstructionTemplate: "feature/completed.md",
				}},
			},
		}, nil
	}

	svc, err := NewActionService("/tmp/td018-test", loader)
	if err != nil {
		t.Fatalf("NewActionService: %v", err)
	}

	ctx := context.Background()

	t.Run("root ValidateActions qualifies non-default entities", func(t *testing.T) {
		res, err := svc.ValidateActions(ctx)
		if err != nil {
			t.Fatalf("ValidateActions: %v", err)
		}
		// We expect two missing actions: "ready_for_review" (task -> unprefixed)
		// and "feature:ready_for_qa" (feature -> prefixed).
		if len(res.MissingActions) != 2 {
			t.Fatalf("MissingActions = %v, want 2 entries", res.MissingActions)
		}
		gotTask, gotFeature := false, false
		for _, m := range res.MissingActions {
			switch m {
			case "ready_for_review":
				gotTask = true
			case "feature:ready_for_qa":
				gotFeature = true
			}
		}
		if !gotTask || !gotFeature {
			t.Errorf("missing expected keys; got %v", res.MissingActions)
		}
	})

	t.Run("ForEntity ValidateActions emits raw status names", func(t *testing.T) {
		res, err := svc.ForEntity("feature").ValidateActions(ctx)
		if err != nil {
			t.Fatalf("ValidateActions: %v", err)
		}
		if len(res.MissingActions) != 1 || res.MissingActions[0] != "ready_for_qa" {
			t.Errorf("MissingActions = %v, want [\"ready_for_qa\"]", res.MissingActions)
		}
	})
}
