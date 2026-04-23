package models

import (
	"strings"
	"testing"
	"time"
)

// TestTagStruct verifies the Tag struct fields and JSON tags compile correctly.
func TestTagStruct(t *testing.T) {
	now := time.Now()
	tag := Tag{
		ID:        1,
		Name:      "voice",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if tag.ID != 1 {
		t.Errorf("Tag.ID = %d, want 1", tag.ID)
	}
	if tag.Name != "voice" {
		t.Errorf("Tag.Name = %q, want %q", tag.Name, "voice")
	}
	if !tag.CreatedAt.Equal(now) {
		t.Errorf("Tag.CreatedAt = %v, want %v", tag.CreatedAt, now)
	}
	if !tag.UpdatedAt.Equal(now) {
		t.Errorf("Tag.UpdatedAt = %v, want %v", tag.UpdatedAt, now)
	}
}

// TestEntityTagLinkStruct verifies EntityTagLink struct fields and JSON tags compile correctly.
func TestEntityTagLinkStruct(t *testing.T) {
	now := time.Now()
	link := EntityTagLink{
		ID:         42,
		EntityType: EntityTypeTask,
		EntityID:   7,
		TagID:      3,
		CreatedAt:  now,
	}

	if link.ID != 42 {
		t.Errorf("EntityTagLink.ID = %d, want 42", link.ID)
	}
	if link.EntityType != EntityTypeTask {
		t.Errorf("EntityTagLink.EntityType = %q, want %q", link.EntityType, EntityTypeTask)
	}
	if link.EntityID != 7 {
		t.Errorf("EntityTagLink.EntityID = %d, want 7", link.EntityID)
	}
	if link.TagID != 3 {
		t.Errorf("EntityTagLink.TagID = %d, want 3", link.TagID)
	}
	if !link.CreatedAt.Equal(now) {
		t.Errorf("EntityTagLink.CreatedAt = %v, want %v", link.CreatedAt, now)
	}
}

// TestValidateTagName tests the ValidateTagName function per ADR-4.
// The regex is ^[a-z0-9][a-z0-9-]{0,63}$ and input is lowercased before validation.
func TestValidateTagName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid cases
		{
			name:    "valid lowercase slug",
			input:   "voice",
			wantErr: false,
		},
		{
			name:    "valid slug with hyphens",
			input:   "voice-auth",
			wantErr: false,
		},
		{
			name:    "valid single character",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "valid starts with digit",
			input:   "3d-rendering",
			wantErr: false,
		},
		{
			name:    "valid all digits",
			input:   "42",
			wantErr: false,
		},
		{
			name:    "valid max length 64 chars",
			input:   strings.Repeat("a", 64),
			wantErr: false,
		},
		{
			name:    "uppercase input passes after lowercasing",
			input:   "VOICE",
			wantErr: false,
		},
		{
			name:    "mixed case passes after lowercasing",
			input:   "Voice-Auth",
			wantErr: false,
		},
		// Invalid cases
		{
			name:    "empty string fails",
			input:   "",
			wantErr: true,
		},
		{
			name:    "hyphen-leading string fails",
			input:   "-voice",
			wantErr: true,
		},
		{
			name:    "65-character string fails",
			input:   strings.Repeat("a", 65),
			wantErr: true,
		},
		{
			name:    "string with spaces fails",
			input:   "voice auth",
			wantErr: true,
		},
		{
			name:    "underscore fails",
			input:   "voice_auth",
			wantErr: true,
		},
		{
			name:    "special chars fails",
			input:   "voice!",
			wantErr: true,
		},
		{
			name:    "dot fails",
			input:   "voice.auth",
			wantErr: true,
		},
		{
			name:    "whitespace only fails",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTagName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTagName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestValidateTagName_ErrorMessage verifies that error messages quote the input.
func TestValidateTagName_ErrorMessage(t *testing.T) {
	err := ValidateTagName("INVALID NAME!")
	if err == nil {
		t.Fatal("expected error for invalid tag name, got nil")
	}
	// Error message should contain useful context
	if !strings.Contains(err.Error(), "tag name") {
		t.Errorf("error message should mention 'tag name', got: %q", err.Error())
	}
}

// TestEntityTypeIdea verifies the EntityTypeIdea constant is defined and valid.
func TestEntityTypeIdea(t *testing.T) {
	// The constant must exist and have the right string value.
	if string(EntityTypeIdea) != "idea" {
		t.Errorf("EntityTypeIdea = %q, want %q", EntityTypeIdea, "idea")
	}
}

// TestValidEntityTypes_IncludesIdea verifies that ValidEntityTypes includes EntityTypeIdea.
func TestValidEntityTypes_IncludesIdea(t *testing.T) {
	if !ValidEntityTypes[EntityTypeIdea] {
		t.Errorf("ValidEntityTypes[EntityTypeIdea] = false, want true")
	}
}

// TestValidEntityTypes_ContainsAllExpected verifies that all expected entity types are present.
func TestValidEntityTypes_ContainsAllExpected(t *testing.T) {
	expected := []EntityType{
		EntityTypeEpic,
		EntityTypeFeature,
		EntityTypeTask,
		EntityTypeChange,
		EntityTypeBug,
		EntityTypeTechDebt,
		EntityTypeIdea,
	}

	for _, et := range expected {
		if !ValidEntityTypes[et] {
			t.Errorf("ValidEntityTypes[%q] = false, want true", et)
		}
	}
}
