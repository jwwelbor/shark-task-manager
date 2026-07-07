package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// TestParseListArgs tests the parsing of positional arguments for the list command
func TestParseListArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string // "epic", "feature", or "task"
		wantEpic    *string
		wantFeature *string
		wantErr     bool
	}{
		{
			name:        "no args - list epics",
			args:        []string{},
			wantCommand: "epic",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "epic key - list features in epic",
			args:        []string{"E10"},
			wantCommand: "feature",
			wantEpic:    listStringPtr("E10"),
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "feature key combined - list tasks in feature",
			args:        []string{"E10-F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "epic and feature separate - list tasks",
			args:        []string{"E10", "F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "epic and feature full key - list tasks",
			args:        []string{"E10", "E10-F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "change_card keyword alias - list changes",
			args:        []string{"change_card"},
			wantCommand: "change",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "change-cards keyword alias - list changes",
			args:        []string{"change-cards"},
			wantCommand: "change",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "invalid epic format",
			args:        []string{"E1"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
		{
			name:        "invalid feature format",
			args:        []string{"E10-F1"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
		{
			name:        "too many args",
			args:        []string{"E10", "F01", "extra"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, epic, feature, err := ParseListArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseListArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if command != tt.wantCommand {
				t.Errorf("ParseListArgs() command = %v, want %v", command, tt.wantCommand)
			}

			if !listStringPtrEqual(epic, tt.wantEpic) {
				t.Errorf("ParseListArgs() epic = %v, want %v", listStringPtrValue(epic), listStringPtrValue(tt.wantEpic))
			}

			if !listStringPtrEqual(feature, tt.wantFeature) {
				t.Errorf("ParseListArgs() feature = %v, want %v", listStringPtrValue(feature), listStringPtrValue(tt.wantFeature))
			}
		})
	}
}

// Helper functions for testing
func listStringPtr(s string) *string {
	return &s
}

func listStringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func listStringPtrValue(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestRunList_ForwardsStatusToStandaloneTypes(t *testing.T) {
	t.Run("bug", func(t *testing.T) {
		var gotStatus *models.BugStatus
		stub := &mockBugServiceForTags{}
		stub.listBugsFn = func(_ context.Context, filters services.BugFilters) ([]*models.Bug, error) {
			gotStatus = filters.Status
			return []*models.Bug{}, nil
		}
		withBugSvcOverride(t, stub)

		cmd := buildListTestCommand("triaged")
		err := runList(cmd, []string{"bug"})
		if err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
		if gotStatus == nil || *gotStatus != models.BugStatus("triaged") {
			t.Fatalf("status filter = %#v, want triaged", gotStatus)
		}
	})

	t.Run("change", func(t *testing.T) {
		var gotStatus string
		withChangeCardSvcOverride(t, &MockChangeCardService{
			ListChangeCardsFunc: func(_ context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
				gotStatus = filters.Status
				return []*models.ChangeCard{}, nil
			},
		})

		cmd := buildListTestCommand("approved")
		err := runList(cmd, []string{"change"})
		if err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
		if gotStatus != "approved" {
			t.Fatalf("status filter = %q, want approved", gotStatus)
		}
	})

	t.Run("tech debt", func(t *testing.T) {
		var gotStatus *string
		withTechDebtSvcOverride(t, &MockTechDebtService{
			ListTechDebtsFunc: func(_ context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
				gotStatus = filters.Status
				return []*models.TechDebt{}, nil
			},
		})

		cmd := buildListTestCommand("identified")
		err := runList(cmd, []string{"tech_debt"})
		if err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
		if gotStatus == nil || *gotStatus != "identified" {
			t.Fatalf("status filter = %#v, want identified", gotStatus)
		}
	})
}

func buildListTestCommand(status string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("sort-by", "", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().StringSlice("tag", nil, "")
	_ = cmd.Flags().Set("status", status)
	return cmd
}

func withTechDebtSvcOverride(t *testing.T, svc techDebtServicer) {
	t.Helper()
	orig := tdSvcOverride
	tdSvcOverride = svc
	t.Cleanup(func() { tdSvcOverride = orig })
}
