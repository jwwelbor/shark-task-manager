package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHistoryCommand_EntityKeyDetection verifies that `shark history <key>`
// correctly detects entity keys and delegates to entity-specific history
// instead of treating them as epic/feature positional args (which causes errors).
//
// This was a bug where `shark history T-E21-F10-001` failed because it used
// ParseListArgs which only understands epic/feature positional args.
func TestHistoryCommand_EntityKeyDetection(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectEntity bool   // true if args represent a single entity key
		expectedType string // expected entity type when expectEntity is true
		expectedKey  string // expected key when expectEntity is true
	}{
		{
			name:         "no args - project-wide history",
			args:         []string{},
			expectEntity: false,
		},
		{
			name:         "epic key only - project-wide filtered by epic",
			args:         []string{"E07"},
			expectEntity: true,
			expectedType: "epic",
			expectedKey:  "E07",
		},
		{
			name:         "task key with T- prefix",
			args:         []string{"T-E21-F10-001"},
			expectEntity: true,
			expectedType: "task",
			expectedKey:  "T-E21-F10-001",
		},
		{
			name:         "task key short format",
			args:         []string{"E21-F10-001"},
			expectEntity: true,
			expectedType: "task",
			expectedKey:  "E21-F10-001",
		},
		{
			name:         "feature key",
			args:         []string{"E07-F01"},
			expectEntity: true,
			expectedType: "feature",
			expectedKey:  "E07-F01",
		},
		{
			name:         "bug key",
			args:         []string{"B001"},
			expectEntity: true,
			expectedType: "bug",
			expectedKey:  "B001",
		},
		{
			name:         "change-card key",
			args:         []string{"CC-001"},
			expectEntity: true,
			expectedType: "change_card",
			expectedKey:  "CC-001",
		},
		{
			name:         "epic key with slug",
			args:         []string{"E07-user-management"},
			expectEntity: true,
			expectedType: "epic",
			expectedKey:  "E07-user-management",
		},
		{
			name:         "task key with slug",
			args:         []string{"E07-F01-001-implement-auth"},
			expectEntity: true,
			expectedType: "task",
			expectedKey:  "E07-F01-001-implement-auth",
		},
		{
			name:         "two args epic feature - project-wide filtered",
			args:         []string{"E07", "F01"},
			expectEntity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityType, key, isEntity := detectHistoryEntityKey(tt.args)

			assert.Equal(t, tt.expectEntity, isEntity,
				"detectHistoryEntityKey(%v) isEntity mismatch", tt.args)

			if tt.expectEntity {
				assert.Equal(t, tt.expectedType, entityType,
					"detectHistoryEntityKey(%v) entityType mismatch", tt.args)
				assert.Equal(t, tt.expectedKey, key,
					"detectHistoryEntityKey(%v) key mismatch", tt.args)
			}
		})
	}
}
