package models

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestSprint_Validate covers happy paths and structural validation failures
// for the Sprint model. Status validity is NOT checked here (workflow-driven).
func TestSprint_Validate(t *testing.T) {
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		sprint  Sprint
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sprint",
			sprint: Sprint{
				Key:       "S001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: false,
		},
		{
			name: "valid sprint S999",
			sprint: Sprint{
				Key:       "S999",
				Name:      "Final sprint",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("active"),
			},
			wantErr: false,
		},
		{
			name: "invalid key - lowercase",
			sprint: Sprint{
				Key:       "s001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
		},
		{
			name: "invalid key - too few digits",
			sprint: Sprint{
				Key:       "S01",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
		},
		{
			name: "invalid key - too many digits",
			sprint: Sprint{
				Key:       "S0001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
		},
		{
			name: "invalid key - wrong prefix",
			sprint: Sprint{
				Key:       "E001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
		},
		{
			name: "empty key",
			sprint: Sprint{
				Key:       "",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
		},
		{
			name: "empty name",
			sprint: Sprint{
				Key:       "S001",
				Name:      "",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "whitespace name",
			sprint: Sprint{
				Key:       "S001",
				Name:      "   ",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "end_date equals start_date - rejected",
			sprint: Sprint{
				Key:       "S001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   start,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
			errMsg:  "sprint end_date must be after start_date",
		},
		{
			name: "end_date before start_date - rejected",
			sprint: Sprint{
				Key:       "S001",
				Name:      "Sprint 1",
				StartDate: end,
				EndDate:   start,
				Status:    SprintStatus("planning"),
			},
			wantErr: true,
			errMsg:  "sprint end_date must be after start_date",
		},
		{
			name: "empty status",
			sprint: Sprint{
				Key:       "S001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus(""),
			},
			wantErr: true,
			errMsg:  "sprint status cannot be empty",
		},
		{
			name: "whitespace status",
			sprint: Sprint{
				Key:       "S001",
				Name:      "Sprint 1",
				StartDate: start,
				EndDate:   end,
				Status:    SprintStatus("   "),
			},
			wantErr: true,
			errMsg:  "sprint status cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sprint.Validate()
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

// TestSprintAssignment_Validate verifies app-layer entity_type allowlist
// (no DB CHECK per post-B018 convention) plus non-zero ID checks.
func TestSprintAssignment_Validate(t *testing.T) {
	tests := []struct {
		name       string
		assignment SprintAssignment
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid - task entity",
			assignment: SprintAssignment{SprintID: 1, EntityType: "task", EntityID: 100},
			wantErr:    false,
		},
		{
			name:       "valid - bug entity",
			assignment: SprintAssignment{SprintID: 1, EntityType: "bug", EntityID: 100},
			wantErr:    false,
		},
		{
			name:       "valid - change_card entity",
			assignment: SprintAssignment{SprintID: 1, EntityType: "change_card", EntityID: 100},
			wantErr:    false,
		},
		{
			name:       "valid - tech_debt entity",
			assignment: SprintAssignment{SprintID: 1, EntityType: "tech_debt", EntityID: 100},
			wantErr:    false,
		},
		{
			name:       "sprint_id zero",
			assignment: SprintAssignment{SprintID: 0, EntityType: "task", EntityID: 100},
			wantErr:    true,
			errMsg:     "sprint_id must be greater than 0",
		},
		{
			name:       "sprint_id negative",
			assignment: SprintAssignment{SprintID: -1, EntityType: "task", EntityID: 100},
			wantErr:    true,
			errMsg:     "sprint_id must be greater than 0",
		},
		{
			name:       "entity_id zero",
			assignment: SprintAssignment{SprintID: 1, EntityType: "task", EntityID: 0},
			wantErr:    true,
			errMsg:     "entity_id must be greater than 0",
		},
		{
			name:       "entity_id negative",
			assignment: SprintAssignment{SprintID: 1, EntityType: "task", EntityID: -1},
			wantErr:    true,
			errMsg:     "entity_id must be greater than 0",
		},
		{
			name:       "invalid entity_type - idea (not allowlisted)",
			assignment: SprintAssignment{SprintID: 1, EntityType: "idea", EntityID: 100},
			wantErr:    true,
		},
		{
			name:       "invalid entity_type - empty",
			assignment: SprintAssignment{SprintID: 1, EntityType: "", EntityID: 100},
			wantErr:    true,
		},
		{
			name:       "invalid entity_type - uppercase TASK",
			assignment: SprintAssignment{SprintID: 1, EntityType: "TASK", EntityID: 100},
			wantErr:    true,
		},
		{
			name:       "invalid entity_type - whitespace-padded ' task '",
			assignment: SprintAssignment{SprintID: 1, EntityType: " task ", EntityID: 100},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.assignment.Validate()
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

// TestSprintAssignment_AllocatedPointsNullable verifies SprintCapacity has the
// nullable AllocatedPoints field so it can be omitted by the foundational
// schema (no triggers maintain it in this feature).
func TestSprintCapacity_AllocatedPointsNullable(t *testing.T) {
	// Zero value: AllocatedPoints is *float64, defaults to nil.
	cap1 := SprintCapacity{
		SprintID:       1,
		AgentType:      "backend",
		CapacityPoints: 20.0,
	}
	if cap1.AllocatedPoints != nil {
		t.Errorf("AllocatedPoints zero value = %v, want nil", cap1.AllocatedPoints)
	}

	// Non-nil case.
	allocated := 12.5
	cap2 := SprintCapacity{
		SprintID:        1,
		AgentType:       "backend",
		CapacityPoints:  20.0,
		AllocatedPoints: &allocated,
	}
	if cap2.AllocatedPoints == nil {
		t.Fatal("AllocatedPoints = nil, want non-nil")
	}
	if *cap2.AllocatedPoints != 12.5 {
		t.Errorf("*AllocatedPoints = %v, want 12.5", *cap2.AllocatedPoints)
	}
}

// TestSprintAssignment_RemovedAtNullable verifies SprintAssignment.RemovedAt is *time.Time.
func TestSprintAssignment_RemovedAtNullable(t *testing.T) {
	sa := SprintAssignment{
		SprintID:   1,
		EntityType: "task",
		EntityID:   100,
	}
	if sa.RemovedAt != nil {
		t.Errorf("RemovedAt zero value = %v, want nil", sa.RemovedAt)
	}

	now := time.Now()
	sa.RemovedAt = &now
	if sa.RemovedAt == nil {
		t.Fatal("RemovedAt = nil, want non-nil")
	}
}

// TestValidateSprintKey_Valid covers happy-path keys for the standalone validator.
func TestValidateSprintKey_Valid(t *testing.T) {
	valid := []string{"S001", "S024", "S100", "S999"}
	for _, key := range valid {
		t.Run(key, func(t *testing.T) {
			if err := ValidateSprintKey(key); err != nil {
				t.Errorf("ValidateSprintKey(%q) = %v, want nil", key, err)
			}
		})
	}
}

// TestValidateSprintKey_Invalid covers all rejection cases. The validator
// receives an already-normalised (uppercase) key — the caller is responsible
// for normalising via keys.Normalize() upstream.
func TestValidateSprintKey_Invalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty string", ""},
		{"single S", "S"},
		{"S followed by single digit", "S0"},
		{"S followed by single non-zero digit", "S1"},
		{"two digits", "S01"},
		{"four digits", "S0001"},
		{"lowercase", "s024"},
		{"epic key", "E07"},
		{"feature key", "E07-F01"},
		{"task key", "T-E07-F01-001"},
		{"bug key", "B001"},
		{"change-card key", "CC-001"},
		{"tech-debt key", "TD-001"},
		{"S followed by letter", "Sabc"},
		{"S with extra suffix", "S001-extra"},
		{"SQL injection", "S001 OR 1=1"},
		{"semicolon injection", "S001;DROP TABLE sprints"},
		{"newline injection", "S001\n"},
		{"null byte injection", "S001\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSprintKey(tt.key)
			if err == nil {
				t.Errorf("ValidateSprintKey(%q) = nil, want error", tt.key)
				return
			}
			if !errors.Is(err, ErrInvalidSprintKey) {
				t.Errorf("ValidateSprintKey(%q) error = %v, want errors.Is(ErrInvalidSprintKey)", tt.key, err)
			}
		})
	}
}

// TestErrInvalidSprintKey_Message verifies the sentinel error message documents the format.
func TestErrInvalidSprintKey_Message(t *testing.T) {
	msg := ErrInvalidSprintKey.Error()
	if !strings.Contains(msg, "S") {
		t.Errorf("ErrInvalidSprintKey message does not mention S###/S\\d{3} format: %q", msg)
	}
}
