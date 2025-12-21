package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Integration tests for feature list command with positional arguments

// TestParseFeatureListArgsIntegration verifies positional argument parsing for feature list
func TestParseFeatureListArgsIntegration(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantEpic   *string
		wantErr    bool
		errMessage string
	}{
		// No arguments - list all features
		{
			name:     "No args lists all features",
			args:     []string{},
			wantEpic: nil,
			wantErr:  false,
		},

		// Valid epic key
		{
			name:     "Valid epic E04",
			args:     []string{"E04"},
			wantEpic: strPtr("E04"),
			wantErr:  false,
		},

		// Invalid formats
		{
			name:       "Invalid format E1",
			args:       []string{"E1"},
			wantEpic:   nil,
			wantErr:    true,
			errMessage: "invalid epic key format",
		},

		{
			name:       "Invalid format e04",
			args:       []string{"e04"},
			wantEpic:   nil,
			wantErr:    true,
			errMessage: "invalid epic key format",
		},

		{
			name:       "Feature key E04-F01 not allowed",
			args:       []string{"E04-F01"},
			wantEpic:   nil,
			wantErr:    true,
			errMessage: "invalid epic key format",
		},

		// Too many arguments
		{
			name:       "Two arguments not allowed",
			args:       []string{"E04", "F01"},
			wantEpic:   nil,
			wantErr:    true,
			errMessage: "too many positional arguments",
		},

		{
			name:       "Three arguments not allowed",
			args:       []string{"E04", "F01", "extra"},
			wantEpic:   nil,
			wantErr:    true,
			errMessage: "too many positional arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, err := ParseFeatureListArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFeatureListArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMessage != "" {
				if err == nil || !contains(err.Error(), tt.errMessage) {
					t.Errorf("ParseFeatureListArgs(%v) error message should contain %q, got %v", tt.args, tt.errMessage, err)
				}
				return
			}

			if (epic == nil) != (tt.wantEpic == nil) {
				t.Errorf("ParseFeatureListArgs(%v) = %v, want %v", tt.args, epic, tt.wantEpic)
				return
			}

			if epic != nil && tt.wantEpic != nil && *epic != *tt.wantEpic {
				t.Errorf("ParseFeatureListArgs(%v) = %q, want %q", tt.args, *epic, *tt.wantEpic)
			}
		})
	}
}

// TestFeatureListQueryWithDatabase verifies feature list with real database
func TestFeatureListQueryWithDatabase(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)

	// Create test epic
	testEpicKey := "E70"
	epic := &models.Epic{
		Key:           testEpicKey,
		Title:         "Feature List Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: ptrPriority(models.PriorityHigh),
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Get epic ID for feature creation
	createdEpic, err := epicRepo.GetByKey(ctx, testEpicKey)
	if err != nil || createdEpic == nil {
		t.Fatalf("Failed to retrieve created epic: %v", err)
	}

	// Create 3 features under the epic
	for i := 1; i <= 3; i++ {
		filePath := fmt.Sprintf("docs/plan/%s/F%02d/feature.md", testEpicKey, i)
		execOrder := i
		feature := &models.Feature{
			Key:            fmt.Sprintf("%s-F%02d", testEpicKey, i),
			EpicID:         createdEpic.ID,
			Title:          fmt.Sprintf("Test Feature %d", i),
			Status:         models.FeatureStatusDraft,
			FilePath:       &filePath,
			ExecutionOrder: &execOrder,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create test feature: %v", err)
		}
	}

	// Test: List all features (no filter)
	allFeatures, err := featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list all features: %v", err)
	}
	if len(allFeatures) == 0 {
		t.Error("Expected features in database but found none")
	}

	// Test: Filter by specific epic
	epicFeatures, err := featureRepo.ListByEpic(ctx, createdEpic.ID)
	if err != nil {
		t.Fatalf("Failed to list features by epic: %v", err)
	}
	if len(epicFeatures) != 3 {
		t.Errorf("Expected 3 features for epic, got %d", len(epicFeatures))
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && s[0:len(substr)] == substr))
}

// Helper to create Priority pointer
func ptrPriority(p models.Priority) *models.Priority {
	return &p
}
