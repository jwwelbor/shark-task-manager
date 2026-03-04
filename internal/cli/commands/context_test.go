package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestToModelEntityType(t *testing.T) {
	tests := []struct {
		name        string
		entityType  string
		want        models.EntityType
		expectError bool
	}{
		{"epic", "epic", models.EntityTypeEpic, false},
		{"feature", "feature", models.EntityTypeFeature, false},
		{"task", "task", models.EntityTypeTask, false},
		{"change_card", "change_card", models.EntityTypeChange, false},
		{"change", "change", models.EntityTypeChange, false},
		{"bug", "bug", models.EntityTypeBug, false},
		{"invalid", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelEntityType(tt.entityType)
			if (err != nil) != tt.expectError {
				t.Errorf("toModelEntityType(%q) error = %v, expectError %v", tt.entityType, err, tt.expectError)
				return
			}
			if got != tt.want {
				t.Errorf("toModelEntityType(%q) = %v, want %v", tt.entityType, got, tt.want)
			}
		})
	}
}
