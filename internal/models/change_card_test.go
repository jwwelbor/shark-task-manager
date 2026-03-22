package models

import (
	"testing"
)

func TestChangeCard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		card    ChangeCard
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid change-card",
			card:    ChangeCard{BaseEntity: BaseEntity{Title: "Add dark mode toggle"}, Status: ChangeCardStatus("proposed")},
			wantErr: false,
		},
		{
			name:    "empty title",
			card:    ChangeCard{BaseEntity: BaseEntity{Title: ""}, Status: ChangeCardStatus("proposed")},
			wantErr: true,
			errMsg:  "change-card title cannot be empty",
		},
		{
			name:    "whitespace title",
			card:    ChangeCard{BaseEntity: BaseEntity{Title: "   "}, Status: ChangeCardStatus("proposed")},
			wantErr: true,
			errMsg:  "change-card title cannot be empty",
		},
		{
			name:    "empty status",
			card:    ChangeCard{BaseEntity: BaseEntity{Title: "Valid Title"}, Status: ChangeCardStatus("")},
			wantErr: true,
			errMsg:  "change-card status cannot be empty",
		},
		{
			name:    "whitespace status",
			card:    ChangeCard{BaseEntity: BaseEntity{Title: "Valid Title"}, Status: ChangeCardStatus("   ")},
			wantErr: true,
			errMsg:  "change-card status cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.card.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestEntityTypeChange(t *testing.T) {
	if EntityTypeChange != "change" {
		t.Errorf("EntityTypeChange = %q, want %q", EntityTypeChange, "change")
	}

	if !ValidEntityTypes[EntityTypeChange] {
		t.Error("EntityTypeChange should be in ValidEntityTypes")
	}
}
