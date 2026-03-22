package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// validEntityHistory returns an EntityHistory with all required fields set to valid values.
func validEntityHistory() EntityHistory {
	return EntityHistory{
		ID:         1,
		EntityType: EntityTypeTask,
		EntityID:   1,
		ToStatus:   "in_progress",
		ChangedAt:  time.Now(),
	}
}

func TestEntityHistory_Validate_Valid(t *testing.T) {
	// AC-7, AC-8: Valid EntityHistory with FromStatus=nil should pass validation
	h := validEntityHistory()
	h.FromStatus = nil

	err := h.Validate()
	if err != nil {
		t.Errorf("Validate() returned error for valid EntityHistory: %v", err)
	}
}

func TestEntityHistory_Validate_EmptyEntityType(t *testing.T) {
	// AC-2: Empty EntityType should return error containing "entity_type"
	h := validEntityHistory()
	h.EntityType = ""

	err := h.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for empty EntityType")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "entity_type") {
		t.Errorf("error should contain 'entity_type', got: %s", err.Error())
	}
}

func TestEntityHistory_Validate_InvalidEntityType(t *testing.T) {
	// AC-3: Invalid EntityType should return error containing "invalid entity_type"
	h := validEntityHistory()
	h.EntityType = "invalid"

	err := h.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for invalid EntityType")
	}
	if !strings.Contains(err.Error(), "invalid entity_type") {
		t.Errorf("error should contain 'invalid entity_type', got: %s", err.Error())
	}
}

func TestEntityHistory_Validate_ZeroEntityID(t *testing.T) {
	// AC-4: Zero EntityID should return error containing "entity_id"
	h := validEntityHistory()
	h.EntityID = 0

	err := h.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for zero EntityID")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "entity_id") {
		t.Errorf("error should contain 'entity_id', got: %s", err.Error())
	}
}

func TestEntityHistory_Validate_NegativeEntityID(t *testing.T) {
	// AC-5: Negative EntityID should return error containing "entity_id"
	h := validEntityHistory()
	h.EntityID = -5

	err := h.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for negative EntityID")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "entity_id") {
		t.Errorf("error should contain 'entity_id', got: %s", err.Error())
	}
}

func TestEntityHistory_Validate_EmptyToStatus(t *testing.T) {
	// AC-6: Empty ToStatus should return error containing "to_status"
	h := validEntityHistory()
	h.ToStatus = ""

	err := h.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for empty ToStatus")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "to_status") {
		t.Errorf("error should contain 'to_status', got: %s", err.Error())
	}
}

func TestEntityHistory_Validate_NilFromStatusAccepted(t *testing.T) {
	// AC-7: Nil FromStatus with all other fields valid should pass
	h := validEntityHistory()
	h.FromStatus = nil

	err := h.Validate()
	if err != nil {
		t.Errorf("Validate() should accept nil FromStatus, got error: %v", err)
	}
}

func TestEntityHistory_Validate_AllEntityTypes(t *testing.T) {
	// AC-8: All valid entity types should pass validation
	entityTypes := []EntityType{
		EntityTypeEpic,
		EntityTypeFeature,
		EntityTypeTask,
		EntityTypeBug,
		EntityTypeChange,
	}

	for _, et := range entityTypes {
		t.Run(string(et), func(t *testing.T) {
			h := validEntityHistory()
			h.EntityType = et

			err := h.Validate()
			if err != nil {
				t.Errorf("Validate() should accept EntityType %q, got error: %v", et, err)
			}
		})
	}
}

func TestEntityHistory_Validate_ArbitraryStatusAccepted(t *testing.T) {
	// AC-9: Arbitrary status values should pass (no status value validation in model)
	h := validEntityHistory()
	h.ToStatus = "made_up_status_name"

	err := h.Validate()
	if err != nil {
		t.Errorf("Validate() should accept arbitrary ToStatus, got error: %v", err)
	}
}

func TestEntityHistory_ForcedField_IsBool(t *testing.T) {
	// AC-10: Forced field should be non-pointer bool with false default
	h := EntityHistory{}
	if h.Forced != false {
		t.Error("Forced field should default to false (Go zero value)")
	}

	h.Forced = true
	if h.Forced != true {
		t.Error("Forced field should be settable to true")
	}
}

func TestEntityHistory_JSONSerialization(t *testing.T) {
	// AC-1, EC-4: Verify JSON tags produce correct keys and omitempty works
	fromStatus := "todo"
	h := EntityHistory{
		ID:         1,
		EntityType: EntityTypeTask,
		EntityID:   42,
		FromStatus: &fromStatus,
		ToStatus:   "in_progress",
		Forced:     false,
		ChangedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Required fields should always be present
	requiredKeys := []string{`"id"`, `"entity_type"`, `"entity_id"`, `"to_status"`, `"forced"`, `"changed_at"`}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("JSON output should contain %s, got: %s", key, jsonStr)
		}
	}

	// FromStatus should be present when non-nil
	if !strings.Contains(jsonStr, `"from_status"`) {
		t.Errorf("JSON output should contain from_status when non-nil, got: %s", jsonStr)
	}

	// Forced=false should still appear (no omitempty on bool)
	if !strings.Contains(jsonStr, `"forced":false`) {
		t.Errorf("JSON output should contain forced:false, got: %s", jsonStr)
	}
}

func TestEntityHistory_JSONSerialization_OmitemptyNil(t *testing.T) {
	// EC-4: Nullable fields should be omitted when nil
	h := EntityHistory{
		ID:         1,
		EntityType: EntityTypeTask,
		EntityID:   1,
		ToStatus:   "todo",
		ChangedAt:  time.Now(),
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Nil pointer fields should be omitted
	omittedKeys := []string{`"from_status"`, `"changed_by"`, `"notes"`, `"rejection_reason"`}
	for _, key := range omittedKeys {
		if strings.Contains(jsonStr, key) {
			t.Errorf("JSON output should omit %s when nil, got: %s", key, jsonStr)
		}
	}
}

func TestEntityHistory_FieldCount(t *testing.T) {
	// AC-1: EntityHistory should have exactly 10 fields
	// Verify by checking all fields are accessible
	fromStatus := "old"
	changedBy := "agent"
	notes := "note"
	reason := "reason"

	h := EntityHistory{
		ID:              1,
		EntityType:      EntityTypeTask,
		EntityID:        2,
		FromStatus:      &fromStatus,
		ToStatus:        "new",
		ChangedBy:       &changedBy,
		Notes:           &notes,
		Forced:          true,
		RejectionReason: &reason,
		ChangedAt:       time.Now(),
	}

	// If any field were missing or wrong type, this wouldn't compile
	_ = h
}

func TestEntityHistory_Validate_AllNullableFieldsNil(t *testing.T) {
	// EC-3: All nullable fields nil simultaneously should be valid
	h := EntityHistory{
		EntityType: EntityTypeTask,
		EntityID:   1,
		ToStatus:   "todo",
		ChangedAt:  time.Now(),
		// FromStatus, ChangedBy, Notes, RejectionReason all nil
	}

	err := h.Validate()
	if err != nil {
		t.Errorf("Validate() should accept all nullable fields as nil, got error: %v", err)
	}
}
