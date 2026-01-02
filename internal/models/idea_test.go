package models

import (
	"testing"
	"time"
)

// TestIdeaValidation_ValidIdea tests that a valid idea passes validation
func TestIdeaValidation_ValidIdea(t *testing.T) {
	createdDate := time.Now()
	status := IdeaStatusNew
	title := "Test Idea"
	priority := 5
	order := 1

	idea := &Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       title,
		CreatedDate: createdDate,
		Status:      status,
		Priority:    &priority,
		Order:       &order,
	}

	err := idea.Validate()
	if err != nil {
		t.Errorf("Expected valid idea to pass validation, got error: %v", err)
	}
}

// TestIdeaValidation_EmptyKey tests that idea with empty key fails validation
func TestIdeaValidation_EmptyKey(t *testing.T) {
	createdDate := time.Now()
	status := IdeaStatusNew
	title := "Test Idea"

	idea := &Idea{
		ID:          1,
		Key:         "",
		Title:       title,
		CreatedDate: createdDate,
		Status:      status,
	}

	err := idea.Validate()
	if err == nil {
		t.Error("Expected validation to fail for empty key")
	}
}

// TestIdeaValidation_EmptyTitle tests that idea with empty title fails validation
func TestIdeaValidation_EmptyTitle(t *testing.T) {
	createdDate := time.Now()
	status := IdeaStatusNew

	idea := &Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       "",
		CreatedDate: createdDate,
		Status:      status,
	}

	err := idea.Validate()
	if err == nil {
		t.Error("Expected validation to fail for empty title")
	}
	if err != ErrEmptyTitle {
		t.Errorf("Expected ErrEmptyTitle, got: %v", err)
	}
}

// TestIdeaValidation_InvalidStatus tests that idea with invalid status fails validation
func TestIdeaValidation_InvalidStatus(t *testing.T) {
	createdDate := time.Now()
	status := IdeaStatus("invalid_status")
	title := "Test Idea"

	idea := &Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       title,
		CreatedDate: createdDate,
		Status:      status,
	}

	err := idea.Validate()
	if err == nil {
		t.Error("Expected validation to fail for invalid status")
	}
}

// TestIdeaValidation_ValidStatuses tests all valid idea status values
func TestIdeaValidation_ValidStatuses(t *testing.T) {
	validStatuses := []IdeaStatus{
		IdeaStatusNew,
		IdeaStatusOnHold,
		IdeaStatusConverted,
		IdeaStatusArchived,
	}

	createdDate := time.Now()
	title := "Test Idea"

	for _, status := range validStatuses {
		idea := &Idea{
			ID:          1,
			Key:         "I-2026-01-01-01",
			Title:       title,
			CreatedDate: createdDate,
			Status:      status,
		}

		err := idea.Validate()
		if err != nil {
			t.Errorf("Expected status %q to be valid, got error: %v", status, err)
		}
	}
}

// TestIdeaKeyFormat tests that idea key follows I-YYYY-MM-DD-xx format
func TestIdeaKeyFormat(t *testing.T) {
	testCases := []struct {
		name      string
		key       string
		shouldErr bool
	}{
		{
			name:      "valid key with single digit sequence",
			key:       "I-2026-01-01-01",
			shouldErr: false,
		},
		{
			name:      "valid key with double digit sequence",
			key:       "I-2026-01-01-99",
			shouldErr: false,
		},
		{
			name:      "invalid key missing prefix",
			key:       "2026-01-01-01",
			shouldErr: true,
		},
		{
			name:      "invalid key wrong format",
			key:       "I-01-01-2026-01",
			shouldErr: true,
		},
		{
			name:      "invalid key wrong separator",
			key:       "I_2026_01_01_01",
			shouldErr: true,
		},
	}

	createdDate := time.Now()
	status := IdeaStatusNew
	title := "Test Idea"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idea := &Idea{
				ID:          1,
				Key:         tc.key,
				Title:       title,
				CreatedDate: createdDate,
				Status:      status,
			}

			err := idea.Validate()
			if tc.shouldErr && err == nil {
				t.Errorf("Expected validation to fail for key %q", tc.key)
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected validation to pass for key %q, got error: %v", tc.key, err)
			}
		})
	}
}
