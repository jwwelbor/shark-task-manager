package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestMapDetectedTypeToEntityType(t *testing.T) {
	tests := []struct {
		name     string
		detected string
		want     models.EntityType
		wantErr  bool
	}{
		{
			name:     "epic",
			detected: "epic",
			want:     models.EntityTypeEpic,
		},
		{
			name:     "feature",
			detected: "feature",
			want:     models.EntityTypeFeature,
		},
		{
			name:     "task",
			detected: "task",
			want:     models.EntityTypeTask,
		},
		{
			name:     "bug",
			detected: "bug",
			want:     models.EntityTypeBug,
		},
		{
			name:     "change",
			detected: "change",
			want:     models.EntityTypeChange,
		},
		{
			name:     "change_card",
			detected: "change_card",
			want:     models.EntityTypeChange,
		},
		{
			name:     "unknown type returns error",
			detected: "unknown",
			wantErr:  true,
		},
		{
			name:     "empty string returns error",
			detected: "",
			wantErr:  true,
		},
		{
			name:     "idea returns error (not a relationship entity)",
			detected: "idea",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapDetectedTypeToEntityType(tt.detected)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapDetectedTypeToEntityType(%q) error = %v, wantErr %v", tt.detected, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("mapDetectedTypeToEntityType(%q) = %q, want %q", tt.detected, got, tt.want)
			}
		})
	}
}

func TestMapDetectedTypeToEntityType_AllValidEntityTypes(t *testing.T) {
	// Ensure all entity types that have registered repositories can be mapped.
	// This is a smoke test to catch if new entity types are added without updating the mapping.
	requiredMappings := map[string]models.EntityType{
		"epic":    models.EntityTypeEpic,
		"feature": models.EntityTypeFeature,
		"task":    models.EntityTypeTask,
		"bug":     models.EntityTypeBug,
		"change":  models.EntityTypeChange,
	}

	for detected, want := range requiredMappings {
		got, err := mapDetectedTypeToEntityType(detected)
		if err != nil {
			t.Errorf("mapDetectedTypeToEntityType(%q) returned unexpected error: %v", detected, err)
			continue
		}
		if got != want {
			t.Errorf("mapDetectedTypeToEntityType(%q) = %q, want %q", detected, got, want)
		}
	}
}

func TestLinkCommandRelationshipTypeValidation(t *testing.T) {
	// Verify that all valid relationship types are accepted
	for relType := range models.ValidEntityRelationshipTypeSet {
		if !models.ValidEntityRelationshipTypeSet[relType] {
			t.Errorf("expected relationship type %q to be valid", relType)
		}
	}

	// Verify that an invalid type is rejected
	invalid := models.EntityRelationshipType("invalid_type")
	if models.ValidEntityRelationshipTypeSet[invalid] {
		t.Error("expected 'invalid_type' to be rejected as invalid relationship type")
	}
}
