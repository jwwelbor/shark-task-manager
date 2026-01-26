package init

import (
	"testing"
)

// TestGetProfile_Basic tests retrieval of basic profile
func TestGetProfile_Basic(t *testing.T) {
	profile, err := GetProfile("basic")
	if err != nil {
		t.Fatalf("GetProfile('basic') error = %v, want nil", err)
	}

	if profile == nil {
		t.Fatal("GetProfile('basic') returned nil")
	}

	if profile.Name != "basic" {
		t.Errorf("profile.Name = %q, want %q", profile.Name, "basic")
	}

	if len(profile.StatusMetadata) != 5 {
		t.Errorf("basic profile status count = %d, want 5", len(profile.StatusMetadata))
	}

	// Verify required statuses exist
	requiredStatuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
	for _, status := range requiredStatuses {
		if _, exists := profile.StatusMetadata[status]; !exists {
			t.Errorf("basic profile missing status: %s", status)
		}
	}
}

// TestGetProfile_Advanced tests retrieval of advanced profile
func TestGetProfile_Advanced(t *testing.T) {
	profile, err := GetProfile("advanced")
	if err != nil {
		t.Fatalf("GetProfile('advanced') error = %v, want nil", err)
	}

	if profile == nil {
		t.Fatal("GetProfile('advanced') returned nil")
	}

	if profile.Name != "advanced" {
		t.Errorf("profile.Name = %q, want %q", profile.Name, "advanced")
	}

	if len(profile.StatusMetadata) != 19 {
		t.Errorf("advanced profile status count = %d, want 19", len(profile.StatusMetadata))
	}

	// Verify some key statuses exist
	keyStatuses := []string{"draft", "in_development", "in_code_review", "in_qa", "completed", "blocked"}
	for _, status := range keyStatuses {
		if _, exists := profile.StatusMetadata[status]; !exists {
			t.Errorf("advanced profile missing status: %s", status)
		}
	}
}

// TestGetProfile_Invalid tests error handling for invalid profile
func TestGetProfile_Invalid(t *testing.T) {
	profile, err := GetProfile("invalid")
	if err == nil {
		t.Fatal("GetProfile('invalid') error = nil, want error")
	}

	if profile != nil {
		t.Errorf("GetProfile('invalid') returned non-nil profile: %v", profile)
	}

	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestGetProfile_CaseInsensitive tests case-insensitive profile lookup
func TestGetProfile_CaseInsensitive(t *testing.T) {
	tests := []string{"BASIC", "Basic", "bAsIc", "ADVANCED", "Advanced"}

	for _, name := range tests {
		profile, err := GetProfile(name)
		if err != nil {
			t.Errorf("GetProfile(%q) error = %v, want nil", name, err)
		}

		if profile == nil {
			t.Errorf("GetProfile(%q) returned nil", name)
		}
	}
}

// TestListProfiles tests listing available profiles
func TestListProfiles(t *testing.T) {
	profiles := ListProfiles()

	if len(profiles) != 2 {
		t.Errorf("ListProfiles() returned %d profiles, want 2", len(profiles))
	}

	// Check for both expected profiles
	hasBasic := false
	hasAdvanced := false

	for _, profile := range profiles {
		if profile == "basic" {
			hasBasic = true
		}
		if profile == "advanced" {
			hasAdvanced = true
		}
	}

	if !hasBasic {
		t.Error("ListProfiles() missing 'basic'")
	}

	if !hasAdvanced {
		t.Error("ListProfiles() missing 'advanced'")
	}
}

// TestBasicProfile_Structure validates basic profile structure
func TestBasicProfile_Structure(t *testing.T) {
	profile, _ := GetProfile("basic")

	// Verify all statuses have required metadata
	for statusName, metadata := range profile.StatusMetadata {
		if metadata == nil {
			t.Errorf("basic profile status %q has nil metadata", statusName)
			continue
		}

		if metadata.Color == "" {
			t.Errorf("basic profile status %q missing color", statusName)
		}

		if metadata.Phase == "" {
			t.Errorf("basic profile status %q missing phase", statusName)
		}

		if metadata.ProgressWeight < 0.0 || metadata.ProgressWeight > 1.0 {
			t.Errorf("basic profile status %q has invalid progress weight: %f", statusName, metadata.ProgressWeight)
		}

		if metadata.Responsibility == "" {
			t.Errorf("basic profile status %q missing responsibility", statusName)
		}
	}
}

// TestAdvancedProfile_Structure validates advanced profile structure
func TestAdvancedProfile_Structure(t *testing.T) {
	profile, _ := GetProfile("advanced")

	// Verify profile has status flow
	if len(profile.StatusFlow) == 0 {
		t.Error("advanced profile missing status flow")
	}

	// Verify profile has special statuses
	if len(profile.SpecialStatuses) == 0 {
		t.Error("advanced profile missing special statuses")
	}

	// Verify all statuses have required metadata
	for statusName, metadata := range profile.StatusMetadata {
		if metadata == nil {
			t.Errorf("advanced profile status %q has nil metadata", statusName)
			continue
		}

		if metadata.Color == "" {
			t.Errorf("advanced profile status %q missing color", statusName)
		}

		if metadata.Phase == "" {
			t.Errorf("advanced profile status %q missing phase", statusName)
		}

		if metadata.ProgressWeight < 0.0 || metadata.ProgressWeight > 1.0 {
			t.Errorf("advanced profile status %q has invalid progress weight: %f", statusName, metadata.ProgressWeight)
		}

		if metadata.Responsibility == "" {
			t.Errorf("advanced profile status %q missing responsibility", statusName)
		}
	}
}

// TestBasicProfile_ProgressWeights tests specific progress weights in basic profile
func TestBasicProfile_ProgressWeights(t *testing.T) {
	profile, _ := GetProfile("basic")

	tests := []struct {
		status       string
		expectedWt   float64
	}{
		{"todo", 0.0},
		{"in_progress", 0.5},
		{"ready_for_review", 0.75},
		{"completed", 1.0},
		{"blocked", 0.0},
	}

	for _, tt := range tests {
		metadata, exists := profile.StatusMetadata[tt.status]
		if !exists {
			t.Errorf("status %q not found", tt.status)
			continue
		}

		if metadata.ProgressWeight != tt.expectedWt {
			t.Errorf("status %q progress weight = %f, want %f", tt.status, metadata.ProgressWeight, tt.expectedWt)
		}
	}
}

// TestAdvancedProfile_StatusCount verifies advanced profile has exactly 19 statuses
func TestAdvancedProfile_StatusCount(t *testing.T) {
	profile, _ := GetProfile("advanced")

	// Advanced profile should have 19 statuses
	if len(profile.StatusMetadata) != 19 {
		t.Errorf("advanced profile status count = %d, want 19", len(profile.StatusMetadata))
	}
}

// TestAdvancedProfile_StatusFlow verifies status flow is defined
func TestAdvancedProfile_StatusFlow(t *testing.T) {
	profile, _ := GetProfile("advanced")

	// Verify status flow has entries
	if profile.StatusFlow == nil {
		t.Fatal("advanced profile StatusFlow is nil")
	}

	if len(profile.StatusFlow) != 19 {
		t.Errorf("advanced profile status flow count = %d, want 19", len(profile.StatusFlow))
	}

	// Verify all statuses have flow definitions (even if empty)
	for status := range profile.StatusMetadata {
		if _, exists := profile.StatusFlow[status]; !exists {
			t.Errorf("advanced profile status %q missing from status_flow", status)
		}
	}
}

// TestAdvancedProfile_SpecialStatuses verifies special status groups
func TestAdvancedProfile_SpecialStatuses(t *testing.T) {
	profile, _ := GetProfile("advanced")

	// Verify special statuses exist
	if len(profile.SpecialStatuses) != 3 {
		t.Errorf("advanced profile special statuses count = %d, want 3", len(profile.SpecialStatuses))
	}

	// Verify required special status groups
	requiredGroups := []string{"_start_", "_complete_", "_blocked_"}
	for _, group := range requiredGroups {
		if _, exists := profile.SpecialStatuses[group]; !exists {
			t.Errorf("advanced profile missing special status group: %s", group)
		}
	}
}

// TestProfileNames tests that profile names match registry keys
func TestProfileNames(t *testing.T) {
	profiles := ListProfiles()

	for _, name := range profiles {
		profile, err := GetProfile(name)
		if err != nil {
			t.Errorf("GetProfile(%q) error = %v", name, err)
		}

		if profile.Name != name {
			t.Errorf("profile.Name = %q, want %q", profile.Name, name)
		}
	}
}

// TestBasicProfile_AllMetadataFieldsFilled verifies all metadata fields are set
func TestBasicProfile_AllMetadataFieldsFilled(t *testing.T) {
	profile, _ := GetProfile("basic")

	for statusName, metadata := range profile.StatusMetadata {
		if metadata == nil {
			t.Errorf("basic profile status %q metadata is nil", statusName)
			continue
		}

		// All metadata fields should be filled
		if metadata.Color == "" {
			t.Errorf("basic profile status %q has empty Color", statusName)
		}
		if metadata.Phase == "" {
			t.Errorf("basic profile status %q has empty Phase", statusName)
		}
		if metadata.Responsibility == "" {
			t.Errorf("basic profile status %q has empty Responsibility", statusName)
		}
		// Description can be empty for some statuses
	}
}

// TestAdvancedProfile_AllMetadataFieldsFilled verifies all metadata fields are set
func TestAdvancedProfile_AllMetadataFieldsFilled(t *testing.T) {
	profile, _ := GetProfile("advanced")

	for statusName, metadata := range profile.StatusMetadata {
		if metadata == nil {
			t.Errorf("advanced profile status %q metadata is nil", statusName)
			continue
		}

		// All metadata fields should be filled
		if metadata.Color == "" {
			t.Errorf("advanced profile status %q has empty Color", statusName)
		}
		if metadata.Phase == "" {
			t.Errorf("advanced profile status %q has empty Phase", statusName)
		}
		if metadata.Responsibility == "" {
			t.Errorf("advanced profile status %q has empty Responsibility", statusName)
		}
	}
}

// TestBasicProfile_BlocksFeature verifies only certain statuses block features
func TestBasicProfile_BlocksFeature(t *testing.T) {
	profile, _ := GetProfile("basic")

	blockingStatuses := []string{"ready_for_review", "blocked"}

	for statusName, metadata := range profile.StatusMetadata {
		isBlocking := false
		for _, blocking := range blockingStatuses {
			if statusName == blocking {
				isBlocking = true
				break
			}
		}

		if isBlocking && !metadata.BlocksFeature {
			t.Errorf("basic profile status %q should block feature but doesn't", statusName)
		}
		if !isBlocking && metadata.BlocksFeature {
			t.Errorf("basic profile status %q should not block feature but does", statusName)
		}
	}
}

// TestGetProfile_EmptyString tests behavior with empty string
func TestGetProfile_EmptyString(t *testing.T) {
	profile, err := GetProfile("")
	if err == nil {
		t.Fatal("GetProfile(\"\") should return error")
	}

	if profile != nil {
		t.Errorf("GetProfile(\"\") returned non-nil profile: %v", profile)
	}
}

// TestBasicProfile_MinimalStatuses verifies basic profile minimalism
func TestBasicProfile_MinimalStatuses(t *testing.T) {
	profile, _ := GetProfile("basic")

	// Basic profile should be minimal (5 statuses)
	expectedCount := 5
	if len(profile.StatusMetadata) != expectedCount {
		t.Errorf("basic profile has %d statuses, want %d", len(profile.StatusMetadata), expectedCount)
	}
}

// TestAdvancedProfile_ComprehensiveWorkflow verifies advanced profile has more statuses
func TestAdvancedProfile_ComprehensiveWorkflow(t *testing.T) {
	basic, _ := GetProfile("basic")
	advanced, _ := GetProfile("advanced")

	if len(advanced.StatusMetadata) <= len(basic.StatusMetadata) {
		t.Errorf("advanced profile has %d statuses, should have more than basic's %d",
			len(advanced.StatusMetadata), len(basic.StatusMetadata))
	}
}

// TestProfileDescription tests profile descriptions are set
func TestProfileDescription(t *testing.T) {
	tests := []string{"basic", "advanced"}

	for _, name := range tests {
		profile, _ := GetProfile(name)
		if profile.Description == "" {
			t.Errorf("profile %q has empty description", name)
		}
	}
}
