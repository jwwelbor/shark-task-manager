package commands

import (
	"testing"
)

// TestTaskListFeatureKeyConstruction tests that the CLI correctly constructs
// full feature keys when combining epic and feature positional arguments
func TestTaskListFeatureKeyConstruction(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedEpic       *string
		expectedFeature    *string
		expectedFeatureKey string // The key that should be passed to featureRepo.GetByKey
		shouldError        bool
	}{
		{
			name:               "No arguments",
			args:               []string{},
			expectedEpic:       nil,
			expectedFeature:    nil,
			expectedFeatureKey: "",
		},
		{
			name:               "Epic only",
			args:               []string{"E04"},
			expectedEpic:       strPtr("E04"),
			expectedFeature:    nil,
			expectedFeatureKey: "",
		},
		{
			name:               "Epic and feature suffix",
			args:               []string{"E04", "F01"},
			expectedEpic:       strPtr("E04"),
			expectedFeature:    strPtr("F01"),
			expectedFeatureKey: "E04-F01", // Should construct full key
		},
		{
			name:               "Combined feature key",
			args:               []string{"E04-F01"},
			expectedEpic:       strPtr("E04"),
			expectedFeature:    strPtr("F01"),
			expectedFeatureKey: "E04-F01", // Should construct full key
		},
		{
			name:               "Epic and full feature key",
			args:               []string{"E04", "E04-F01"},
			expectedEpic:       strPtr("E04"),
			expectedFeature:    strPtr("F01"),
			expectedFeatureKey: "E04-F01", // Should construct full key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the arguments
			epicKey, featureKey, err := ParseTaskListArgs(tt.args)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify epic key
			if tt.expectedEpic == nil {
				if epicKey != nil {
					t.Errorf("Expected nil epic key, got %q", *epicKey)
				}
			} else {
				if epicKey == nil {
					t.Errorf("Expected epic key %q, got nil", *tt.expectedEpic)
				} else if *epicKey != *tt.expectedEpic {
					t.Errorf("Expected epic key %q, got %q", *tt.expectedEpic, *epicKey)
				}
			}

			// Verify feature key (suffix only at this stage)
			if tt.expectedFeature == nil {
				if featureKey != nil {
					t.Errorf("Expected nil feature key, got %q", *featureKey)
				}
			} else {
				if featureKey == nil {
					t.Errorf("Expected feature key %q, got nil", *tt.expectedFeature)
				} else if *featureKey != *tt.expectedFeature {
					t.Errorf("Expected feature key %q, got %q", *tt.expectedFeature, *featureKey)
				}
			}

			// Now test the key construction logic that happens in runTaskList
			if tt.expectedFeatureKey != "" {
				var constructedKey string
				if epicKey != nil && featureKey != nil {
					constructedKey = *featureKey
					// Apply the same logic as in runTaskList
					if *epicKey != "" && IsFeatureKeySuffix(constructedKey) {
						constructedKey = *epicKey + "-" + constructedKey
					}
				}

				if constructedKey != tt.expectedFeatureKey {
					t.Errorf("Expected constructed feature key %q, got %q", tt.expectedFeatureKey, constructedKey)
				}
			}
		})
	}
}

// TestFeatureFilteringLogic tests the feature filtering logic
// This test validates that when a feature key is provided, the correct
// filter is applied (epic is passed to FilterCombined, feature is used for post-filtering)
func TestFeatureFilteringLogic(t *testing.T) {
	tests := []struct {
		name                  string
		positionalEpic        *string
		positionalFeature     *string
		flagEpic              string
		flagFeature           string
		expectedEpicFilter    *string // What should be passed to FilterCombined
		expectedFeatureFilter string  // What should be passed to featureRepo.GetByKey
	}{
		{
			name:                  "Positional epic and feature",
			positionalEpic:        strPtr("E04"),
			positionalFeature:     strPtr("F01"),
			expectedEpicFilter:    strPtr("E04"),
			expectedFeatureFilter: "E04-F01",
		},
		{
			name:                  "Positional combined key",
			positionalEpic:        strPtr("E04"),
			positionalFeature:     strPtr("F01"), // Parser extracts suffix
			expectedEpicFilter:    strPtr("E04"),
			expectedFeatureFilter: "E04-F01",
		},
		{
			name:                  "Flag-based epic and feature suffix",
			flagEpic:              "E04",
			flagFeature:           "F01", // Feature suffix
			expectedEpicFilter:    strPtr("E04"),
			expectedFeatureFilter: "E04-F01", // Should construct full key
		},
		{
			name:                  "Flag-based epic and full feature key",
			flagEpic:              "E04",
			flagFeature:           "E04-F01", // Full feature key
			expectedEpicFilter:    strPtr("E04"),
			expectedFeatureFilter: "E04-F01", // Should stay as is
		},
		{
			name:                  "Positional overrides flags",
			positionalEpic:        strPtr("E07"),
			positionalFeature:     strPtr("F06"),
			flagEpic:              "E04",
			flagFeature:           "E04-F01",
			expectedEpicFilter:    strPtr("E07"),
			expectedFeatureFilter: "E07-F06",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from runTaskList
			epicKey := tt.flagEpic
			featureKey := tt.flagFeature

			// Positional arguments take priority over flags
			if tt.positionalEpic != nil {
				epicKey = *tt.positionalEpic
			}
			if tt.positionalFeature != nil {
				featureKey = *tt.positionalFeature
			}

			// If we have both epic and a feature suffix (F##), construct the full key
			// This applies to both flag-based and positional argument syntax
			if epicKey != "" && featureKey != "" && IsFeatureKeySuffix(featureKey) {
				featureKey = epicKey + "-" + featureKey
			}

			// Verify epicKey for FilterCombined
			if tt.expectedEpicFilter == nil {
				if epicKey != "" {
					t.Errorf("Expected no epic filter, got %q", epicKey)
				}
			} else {
				if epicKey != *tt.expectedEpicFilter {
					t.Errorf("Expected epic filter %q, got %q", *tt.expectedEpicFilter, epicKey)
				}
			}

			// Verify featureKey for GetByKey
			if featureKey != tt.expectedFeatureFilter {
				t.Errorf("Expected feature filter %q, got %q", tt.expectedFeatureFilter, featureKey)
			}
		})
	}
}
