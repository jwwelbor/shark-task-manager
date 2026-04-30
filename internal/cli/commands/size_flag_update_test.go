package commands

// size_flag_update_test.go — E07-F42 T-007
//
// Tests for the --size flag on all 6 update commands.
// Test plan: TC-F005-A through TC-F005-E.
//
// Rules:
//   - All tests use mocked services (never real DB) per
//     .claude/rules/testing/cli-tests.md.
//   - Flag is StringVar (not IntVar) per Decision D4.
//   - "clear" literal (exact lowercase) → ClearSize=true, Size=nil (TC-F005-C).
//   - "Clear" / "CLEAR" go through ParseSize and return an error (TC-F005-E, AC-T1).
//   - Empty flag → no-op: Size=nil, ClearSize=false (TC-F005-D, AC-T2).
//   - Valid value → Size=ptr(n), ClearSize=false (TC-F005-B).

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// TC-F005-A: --size flag registered on all 6 update commands
// ---------------------------------------------------------------------------

func TestSizeFlag_RegisteredOnAllUpdateCommands(t *testing.T) {
	cmds := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"epic update", epicUpdateCmd},
		{"feature update", featureUpdateCmd},
		{"task update", taskUpdateCmd},
		{"bug update", bugUpdateCmd},
		{"change update", changeUpdateCmd},
		{"idea update", ideaUpdateCmd},
		{"td update", tdUpdateCmd},
		{"update (dispatch)", updateCmd},
	}

	for _, tc := range cmds {
		t.Run(tc.name, func(t *testing.T) {
			flag := tc.cmd.Flags().Lookup("size")
			if flag == nil {
				t.Errorf("--size flag not registered on %s", tc.name)
				return
			}
			if flag.DefValue != "" {
				t.Errorf("--size flag on %s: expected default \"\", got %q", tc.name, flag.DefValue)
			}
			// Confirm it is a string flag (StringVar, not IntVar) per Decision D4.
			if flag.Value.Type() != "string" {
				t.Errorf("--size flag on %s: expected type string, got %s", tc.name, flag.Value.Type())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseSizeUpdateFlag unit tests — shared helper tested in isolation
// (TC-F005-B/C/D/E coverage)
// ---------------------------------------------------------------------------

// parseSizeUpdateFlag is the core logic for all 6 update commands (extracted
// to task_helpers.go by T-007).  These tests verify the three-way dispatch:
//   - empty → (nil, false, nil)
//   - "clear" → (nil, true, nil)
//   - valid → (ptr(n), false, nil)
//   - invalid → (nil, false, error wrapping ErrInvalidSize)

func TestParseSizeUpdateFlag_EmptyFlag_NoOp(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("size", "", "size")
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sizePtr, clearSize, err := parseSizeUpdateFlag(cmd)
	if err != nil {
		t.Fatalf("expected no error for empty flag, got: %v", err)
	}
	if sizePtr != nil {
		t.Errorf("expected nil Size for empty flag, got %d", *sizePtr)
	}
	if clearSize {
		t.Error("expected ClearSize=false for empty flag")
	}
}

func TestParseSizeUpdateFlag_ClearLiteral_SetsClearSize(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("size", "", "size")
	_ = cmd.Flags().Set("size", "clear")

	sizePtr, clearSize, err := parseSizeUpdateFlag(cmd)
	if err != nil {
		t.Fatalf("expected no error for 'clear', got: %v", err)
	}
	if sizePtr != nil {
		t.Errorf("expected nil Size when clear, got %d", *sizePtr)
	}
	if !clearSize {
		t.Error("expected ClearSize=true for 'clear' literal")
	}
}

func TestParseSizeUpdateFlag_ValidLabel_SetsSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"XS", 1}, {"S", 2}, {"M", 3}, {"L", 5}, {"XL", 8}, {"XXL", 13},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("size", "", "size")
			_ = cmd.Flags().Set("size", tt.input)

			sizePtr, clearSize, err := parseSizeUpdateFlag(cmd)
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", tt.input, err)
			}
			if sizePtr == nil {
				t.Fatalf("expected Size non-nil for %q", tt.input)
			}
			if *sizePtr != tt.expected {
				t.Errorf("expected Size=%d for %q, got %d", tt.expected, tt.input, *sizePtr)
			}
			if clearSize {
				t.Errorf("expected ClearSize=false for valid label %q", tt.input)
			}
		})
	}
}

func TestParseSizeUpdateFlag_ValidNumeric_SetsSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1}, {"2", 2}, {"3", 3}, {"5", 5}, {"8", 8}, {"13", 13},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("size", "", "size")
			_ = cmd.Flags().Set("size", tt.input)

			sizePtr, clearSize, err := parseSizeUpdateFlag(cmd)
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", tt.input, err)
			}
			if sizePtr == nil {
				t.Fatalf("expected non-nil Size for %q", tt.input)
			}
			if *sizePtr != tt.expected {
				t.Errorf("expected Size=%d for %q, got %d", tt.expected, tt.input, *sizePtr)
			}
			if clearSize {
				t.Errorf("expected ClearSize=false for valid numeric %q", tt.input)
			}
		})
	}
}

// TC-F005-E: "Clear" (capitalized) returns error — NOT treated as the clear sentinel.
func TestParseSizeUpdateFlag_ClearCapitalized_ReturnsError(t *testing.T) {
	for _, bad := range []string{"Clear", "CLEAR"} {
		bad := bad
		t.Run(bad, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("size", "", "size")
			_ = cmd.Flags().Set("size", bad)

			_, _, err := parseSizeUpdateFlag(cmd)
			if err == nil {
				t.Errorf("expected error for %q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected error wrapping ErrInvalidSize for %q, got: %v", bad, err)
			}
		})
	}
}

func TestParseSizeUpdateFlag_InvalidValues_ReturnError(t *testing.T) {
	for _, bad := range []string{"4", "XXXL", "medium", "0", "14", "-1", "abc"} {
		bad := bad
		t.Run(bad, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("size", "", "size")
			_ = cmd.Flags().Set("size", bad)

			_, _, err := parseSizeUpdateFlag(cmd)
			if err == nil {
				t.Errorf("expected error for %q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected ErrInvalidSize for %q, got: %v", bad, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-F005-B + TC-F005-C + TC-F005-D + TC-F005-E: bug update
// (bug uses an injectable service so we can capture the DTO)
// ---------------------------------------------------------------------------

// TC-F005-B: --size <valid> on bug update passes correct value to service.
func TestBugUpdate_SizeFlag_LabelForm_PassesParsedSizeToService(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"B001", "--size=XL"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected Size to be non-nil after --size=XL")
	}
	if *capturedUpdates.Size != 8 {
		t.Errorf("expected Size=8 (XL), got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false when --size=XL")
	}
}

func TestBugUpdate_SizeFlag_NumericForm_PassesParsedSizeToService(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"B001", "--size=5"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected Size to be non-nil after --size=5")
	}
	if *capturedUpdates.Size != 5 {
		t.Errorf("expected Size=5, got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false when --size=5")
	}
}

// TC-F005-C: --size clear on bug update sets ClearSize=true.
func TestBugUpdate_SizeFlag_Clear_SetsClearSizeTrue(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"B001", "--size=clear"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if !capturedUpdates.ClearSize {
		t.Error("expected ClearSize=true when --size=clear")
	}
	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when ClearSize=true, got %d", *capturedUpdates.Size)
	}
}

// TC-F005-D: flag absent → no size mutation (Size=nil, ClearSize=false).
func TestBugUpdate_SizeFlag_Absent_NoSizeMutation(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugUpdateCmdWithSize(t)
	// Provide a title update so the "at least one flag" check passes.
	cmd.SetArgs([]string{"B001", "--title=New title"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when --size flag absent, got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false when --size flag absent")
	}
}

// TC-F005-E: "Clear" (capitalized) returns a parse error, not the clear sentinel.
func TestBugUpdate_SizeFlag_ClearCapitalized_ReturnsError(t *testing.T) {
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			t.Error("service should not have been called for invalid --size=Clear")
			return nil, nil
		},
	}
	withBugSvcOverride(t, stub)

	for _, badClear := range []string{"Clear", "CLEAR"} {
		bad := badClear
		t.Run(bad, func(t *testing.T) {
			cmd := buildBugUpdateCmdWithSize(t)
			cmd.SetArgs([]string{"B001", "--size=" + bad})
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for --size=%q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected error wrapping ErrInvalidSize for --size=%q, got: %v", bad, err)
			}
		})
	}
}

// TC-F005-E (other invalid): verify general invalid values still rejected.
func TestBugUpdate_SizeFlag_Invalid_ReturnsError(t *testing.T) {
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			t.Error("service should not have been called for invalid --size")
			return nil, nil
		},
	}
	withBugSvcOverride(t, stub)

	for _, bad := range []string{"4", "XXXL", "medium", "0", "14"} {
		bad := bad
		t.Run(bad, func(t *testing.T) {
			cmd := buildBugUpdateCmdWithSize(t)
			cmd.SetArgs([]string{"B001", "--size=" + bad})
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for --size=%q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected ErrInvalidSize for --size=%q, got: %v", bad, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-F005-B/C/D/E: change update — injectable service
// ---------------------------------------------------------------------------

// TC-F005-B: --size <valid> on change update passes correct value.
func TestChangeUpdate_SizeFlag_LabelForm_PassesParsedSizeToService(t *testing.T) {
	var capturedUpdates services.ChangeCardUpdates
	stub := &MockChangeCardService{
		UpdateChangeCardFunc: func(_ context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			capturedUpdates = updates
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"CC-001", "--size=L"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected Size non-nil after --size=L")
	}
	if *capturedUpdates.Size != 5 {
		t.Errorf("expected Size=5 (L), got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false")
	}
}

// TC-F005-C: --size clear on change update sets ClearSize=true.
func TestChangeUpdate_SizeFlag_Clear_SetsClearSizeTrue(t *testing.T) {
	var capturedUpdates services.ChangeCardUpdates
	stub := &MockChangeCardService{
		UpdateChangeCardFunc: func(_ context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			capturedUpdates = updates
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"CC-001", "--size=clear"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if !capturedUpdates.ClearSize {
		t.Error("expected ClearSize=true when --size=clear")
	}
	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when ClearSize=true, got %d", *capturedUpdates.Size)
	}
}

// TC-F005-D: flag absent → no size mutation.
func TestChangeUpdate_SizeFlag_Absent_NoSizeMutation(t *testing.T) {
	var capturedUpdates services.ChangeCardUpdates
	stub := &MockChangeCardService{
		UpdateChangeCardFunc: func(_ context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			capturedUpdates = updates
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"CC-001", "--title=New title"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when --size absent, got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false when --size absent")
	}
}

// TC-F005-E: "Clear" (capitalized) returns parse error, not treated as clear sentinel.
func TestChangeUpdate_SizeFlag_ClearCapitalized_ReturnsError(t *testing.T) {
	stub := &MockChangeCardService{
		UpdateChangeCardFunc: func(_ context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			t.Error("service should not be called for invalid --size=Clear")
			return nil, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeUpdateCmdWithSize(t)
	cmd.SetArgs([]string{"CC-001", "--size=Clear"})
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --size=Clear, got nil")
		return
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize for --size=Clear, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TC-F005-B/C/D/E: task update — parseTaskUpdates with size support
// ---------------------------------------------------------------------------

// TC-F005-B: --size <valid> on task update populates TaskUpdates.Size.
func TestTaskUpdate_SizeFlag_LabelForm_SetsSize(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--size=XL"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected Size non-nil after --size=XL")
	}
	if *capturedUpdates.Size != 8 {
		t.Errorf("expected Size=8 (XL), got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false")
	}
}

func TestTaskUpdate_SizeFlag_NumericForm_SetsSize(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--size=13"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected Size non-nil after --size=13")
	}
	if *capturedUpdates.Size != 13 {
		t.Errorf("expected Size=13, got %d", *capturedUpdates.Size)
	}
}

// TC-F005-C: --size clear on task update sets ClearSize=true.
func TestTaskUpdate_SizeFlag_Clear_SetsClearSizeTrue(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--size=clear"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if !capturedUpdates.ClearSize {
		t.Error("expected ClearSize=true when --size=clear")
	}
	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when ClearSize=true, got %d", *capturedUpdates.Size)
	}
}

// TC-F005-D: flag absent → no size mutation.
func TestTaskUpdate_SizeFlag_Absent_NoSizeMutation(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001"})
	cmd.SilenceErrors = true
	_ = cmd.Execute()

	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when --size absent, got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false when --size absent")
	}
}

// TC-F005-E: "Clear" capitalized returns parse error.
func TestTaskUpdate_SizeFlag_ClearCapitalized_ReturnsError(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--size=Clear"})
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --size=Clear, got nil")
		return
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize for --size=Clear, got: %v", err)
	}
}

func TestTaskUpdate_SizeFlag_Invalid_ReturnsError(t *testing.T) {
	for _, bad := range []string{"4", "XXXL", "medium"} {
		bad := bad
		t.Run(bad, func(t *testing.T) {
			var capturedUpdates services.TaskUpdates
			cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
			cmd.SetArgs([]string{"E07-F01-001", "--size=" + bad})
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for --size=%q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected ErrInvalidSize for --size=%q, got: %v", bad, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-F005-B/C/D/E: idea update — parseUpdateIdeaInput with size support
// ---------------------------------------------------------------------------

// TC-F005-B: --size <valid> on idea update populates UpdateIdeaInput.Size.
func TestIdeaUpdate_SizeFlag_LabelForm_SetsSize(t *testing.T) {
	var capturedInput services.UpdateIdeaInput
	cmd := buildIdeaUpdateCmdCapture(t, &capturedInput)
	cmd.SetArgs([]string{"I-2026-01-01-01", "--size=S"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size == nil {
		t.Fatal("expected Size non-nil after --size=S")
	}
	if *capturedInput.Size != 2 {
		t.Errorf("expected Size=2 (S), got %d", *capturedInput.Size)
	}
	if capturedInput.ClearSize {
		t.Error("expected ClearSize=false")
	}
}

// TC-F005-C: --size clear on idea update sets ClearSize=true.
func TestIdeaUpdate_SizeFlag_Clear_SetsClearSizeTrue(t *testing.T) {
	var capturedInput services.UpdateIdeaInput
	cmd := buildIdeaUpdateCmdCapture(t, &capturedInput)
	cmd.SetArgs([]string{"I-2026-01-01-01", "--size=clear"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if !capturedInput.ClearSize {
		t.Error("expected ClearSize=true when --size=clear")
	}
	if capturedInput.Size != nil {
		t.Errorf("expected Size=nil when ClearSize=true, got %d", *capturedInput.Size)
	}
}

// TC-F005-D: flag absent → no size mutation.
func TestIdeaUpdate_SizeFlag_Absent_NoSizeMutation(t *testing.T) {
	var capturedInput services.UpdateIdeaInput
	cmd := buildIdeaUpdateCmdCapture(t, &capturedInput)
	cmd.SetArgs([]string{"I-2026-01-01-01"})
	cmd.SilenceErrors = true
	_ = cmd.Execute()

	if capturedInput.Size != nil {
		t.Errorf("expected Size=nil when --size absent, got %d", *capturedInput.Size)
	}
	if capturedInput.ClearSize {
		t.Error("expected ClearSize=false when --size absent")
	}
}

// TC-F005-E: "CLEAR" capitalized returns parse error.
func TestIdeaUpdate_SizeFlag_ClearCapitalized_ReturnsError(t *testing.T) {
	var capturedInput services.UpdateIdeaInput
	cmd := buildIdeaUpdateCmdCapture(t, &capturedInput)
	cmd.SetArgs([]string{"I-2026-01-01-01", "--size=CLEAR"})
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --size=CLEAR, got nil")
		return
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize for --size=CLEAR, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TC-F005-A: flag type checks for epic update and feature update
// ---------------------------------------------------------------------------

func TestEpicUpdateCmd_SizeFlagIsStringType(t *testing.T) {
	flag := epicUpdateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on epicUpdateCmd")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %s", flag.Value.Type())
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
}

func TestFeatureUpdateCmd_SizeFlagIsStringType(t *testing.T) {
	flag := featureUpdateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on featureUpdateCmd")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %s", flag.Value.Type())
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
}

// ---------------------------------------------------------------------------
// Command builders for isolated tests
// ---------------------------------------------------------------------------

// buildBugUpdateCmdWithSize builds an isolated bug update command that uses
// runBugUpdate handler (which calls getBugService() — overrideable in tests).
func buildBugUpdateCmdWithSize(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update <key>",
		Args: cobra.ExactArgs(1),
		RunE: runBugUpdate,
	}
	cmd.Flags().String("title", "", "title")
	cmd.Flags().String("severity", "", "severity")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().String("filename", "", "alias for --file")
	cmd.Flags().String("path", "", "alias for --file")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag")
	cmd.Flags().String("size", "", "size")
	return cmd
}

// buildChangeUpdateCmdWithSize builds an isolated change update command.
func buildChangeUpdateCmdWithSize(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update <key>",
		Args: cobra.ExactArgs(1),
		RunE: runChangeUpdate,
	}
	cmd.Flags().String("title", "", "title")
	cmd.Flags().String("description", "", "description")
	cmd.Flags().Int("priority", 0, "priority")
	cmd.Flags().String("requested-by", "", "requested-by")
	cmd.Flags().String("assigned-to", "", "assigned-to")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().String("filename", "", "alias for --file")
	cmd.Flags().String("path", "", "alias for --file")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag")
	cmd.Flags().String("size", "", "size")
	return cmd
}

// buildTaskUpdateCmdCapture builds a fresh task update command that captures
// the TaskUpdates produced by parseTaskUpdates (after size support is added).
// Does NOT call the real service — purely captures the parsed DTO.
func buildTaskUpdateCmdCapture(t *testing.T, capture *services.TaskUpdates) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			updates := parseTaskUpdates(cmd)
			// Apply size three-way dispatch on top (mirrors runTaskUpdate logic).
			sizePtr, clearSize, err := parseSizeUpdateFlag(cmd)
			if err != nil {
				return err
			}
			updates.Size = sizePtr
			updates.ClearSize = clearSize
			*capture = updates
			return nil
		},
	}
	registerUpdateFlags(cmd)
	return cmd
}

// buildIdeaUpdateCmdCapture builds a fresh idea update command that captures
// the UpdateIdeaInput produced by parseUpdateIdeaInput.
func buildIdeaUpdateCmdCapture(t *testing.T, capture *services.UpdateIdeaInput) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := parseUpdateIdeaInput(cmd)
			if err != nil {
				return err
			}
			*capture = input
			return nil
		},
	}
	// Mirror ideaUpdateCmd flag registration for the flags parseUpdateIdeaInput reads.
	cmd.Flags().StringVar(&ideaStatus, "status", "", "Update status")
	cmd.Flags().IntVar(&ideaPriority, "priority", 0, "Update priority")
	cmd.Flags().StringVar(&ideaDescription, "description", "", "Update description")
	cmd.Flags().StringVar(&ideaNotes, "notes", "", "Update notes")
	cmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "related-docs")
	cmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "depends-on")
	cmd.Flags().IntVar(&ideaOrder, "order", 0, "order")
	cmd.Flags().StringVar(&ideaDescription, "title", "", "title")
	cmd.Flags().StringSliceVar(&ideaUpdateTags, "tag", nil, "tag")
	// --size is registered here for isolation testing.
	cmd.Flags().String("size", "", "size")
	return cmd
}
