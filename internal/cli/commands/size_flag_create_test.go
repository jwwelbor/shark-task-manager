package commands

// size_flag_create_test.go — E07-F42 T-006
//
// Tests for the --size flag on all 6 create commands.
// Test plan: TC-F004-A through TC-F004-E.
//
// Rules:
//   - All tests use mocked services (never real DB) per
//     .claude/rules/testing/cli-tests.md.
//   - Flag is StringVar (not IntVar) so it accepts "5" and "L" per Decision D4.
//   - ParseSize is called in the handler; error returned before calling service.
//   - Empty flag → Size=nil in DTO (not specified).

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// TC-F004-A: --size flag registered on all 6 create commands
// ---------------------------------------------------------------------------

func TestSizeFlag_RegisteredOnAllCreateCommands(t *testing.T) {
	cmds := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"epic create", epicCreateCmd},
		{"feature create", featureCreateCmd},
		{"task create", taskCreateCmd},
		{"bug create", bugCreateCmd},
		{"change create", changeCreateCmd},
		{"idea create", ideaCreateCmd},
		{"td create", tdCreateCmd},
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
// TC-F004-B + TC-F004-C + TC-F004-D + TC-F004-E: bug create
// (bug uses an injectable service so we can capture the DTO)
// ---------------------------------------------------------------------------

func TestBugCreate_SizeFlag_LabelForm_PassesParsedSizeToService(t *testing.T) {
	var capturedInput services.CreateBugInput
	stub := &mockBugServiceForTags{
		createBugFn: func(_ context.Context, input services.CreateBugInput) (*models.Bug, error) {
			capturedInput = input
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: input.Title}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=L", "Test bug"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size == nil {
		t.Fatal("expected Size to be non-nil")
	}
	if *capturedInput.Size != 5 {
		t.Errorf("expected Size=5 (L), got %d", *capturedInput.Size)
	}
}

func TestBugCreate_SizeFlag_NumericForm_PassesParsedSizeToService(t *testing.T) {
	var capturedInput services.CreateBugInput
	stub := &mockBugServiceForTags{
		createBugFn: func(_ context.Context, input services.CreateBugInput) (*models.Bug, error) {
			capturedInput = input
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: input.Title}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=8", "Test bug"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size == nil {
		t.Fatal("expected Size to be non-nil")
	}
	if *capturedInput.Size != 8 {
		t.Errorf("expected Size=8, got %d", *capturedInput.Size)
	}
}

// TC-F004-D: flag absent → Size nil
func TestBugCreate_SizeFlag_Absent_SizeIsNil(t *testing.T) {
	var capturedInput services.CreateBugInput
	stub := &mockBugServiceForTags{
		createBugFn: func(_ context.Context, input services.CreateBugInput) (*models.Bug, error) {
			capturedInput = input
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: input.Title}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugCreateCmdWithSize(t)
	cmd.SetArgs([]string{"Test bug"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size != nil {
		t.Errorf("expected Size=nil when flag absent, got %d", *capturedInput.Size)
	}
}

// TC-F004-E: invalid value returns error before calling service
func TestBugCreate_SizeFlag_Invalid_ReturnsErrorBeforeCallingService(t *testing.T) {
	stub := &mockBugServiceForTags{
		createBugFn: func(_ context.Context, input services.CreateBugInput) (*models.Bug, error) {
			t.Error("service should not have been called for invalid --size")
			return nil, nil
		},
	}
	withBugSvcOverride(t, stub)

	invalidInputs := []string{"4", "XXXL", "medium", "0", "14", "-1"}
	for _, bad := range invalidInputs {
		t.Run(bad, func(t *testing.T) {
			cmd := buildBugCreateCmdWithSize(t)
			cmd.SetArgs([]string{"--size=" + bad, "Test bug"})
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for --size=%q, got nil", bad)
				return
			}
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Errorf("expected error to wrap models.ErrInvalidSize for --size=%q, got: %v", bad, err)
			}
		})
	}
}

// AC-T3: --size clear on a create command must return an error
func TestBugCreate_SizeFlag_Clear_ReturnsError(t *testing.T) {
	stub := &mockBugServiceForTags{
		createBugFn: func(_ context.Context, input services.CreateBugInput) (*models.Bug, error) {
			t.Error("service should not have been called for --size clear on create")
			return nil, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildBugCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=clear", "Test bug"})
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --size=clear on create command, got nil")
		return
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize for --size=clear, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TC-F004-A/B/C/D/E: change create — injectable service
// ---------------------------------------------------------------------------

func TestChangeCreate_SizeFlag_LabelForm(t *testing.T) {
	var capturedInput services.CreateChangeCardInput
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			capturedInput = input
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=M", "Test change"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size == nil {
		t.Fatal("expected Size to be non-nil")
	}
	if *capturedInput.Size != 3 {
		t.Errorf("expected Size=3 (M), got %d", *capturedInput.Size)
	}
}

func TestChangeCreate_SizeFlag_NumericForm(t *testing.T) {
	var capturedInput services.CreateChangeCardInput
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			capturedInput = input
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=13", "Test change"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size == nil {
		t.Fatal("expected Size to be non-nil")
	}
	if *capturedInput.Size != 13 {
		t.Errorf("expected Size=13, got %d", *capturedInput.Size)
	}
}

func TestChangeCreate_SizeFlag_Absent_SizeIsNil(t *testing.T) {
	var capturedInput services.CreateChangeCardInput
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			capturedInput = input
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := buildChangeCreateCmdWithSize(t)
	cmd.SetArgs([]string{"Test change"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedInput.Size != nil {
		t.Errorf("expected Size=nil when flag absent, got %d", *capturedInput.Size)
	}
}

func TestChangeCreate_SizeFlag_Invalid_ReturnsError(t *testing.T) {
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			t.Error("service should not have been called for invalid --size")
			return nil, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	for _, bad := range []string{"4", "XXXL"} {
		t.Run(bad, func(t *testing.T) {
			cmd := buildChangeCreateCmdWithSize(t)
			cmd.SetArgs([]string{"--size=" + bad, "Test change"})
			cmd.SilenceErrors = true
			if err := cmd.Execute(); err == nil {
				t.Errorf("expected error for --size=%q", bad)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-F004-A/B/C/D/E: task create — injectable taskCreateServicer
// ---------------------------------------------------------------------------

func TestTaskCreate_SizeFlag_RegisteredAndIsString(t *testing.T) {
	flag := taskCreateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on taskCreateCmd")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string flag type, got %s", flag.Value.Type())
	}
}

func TestTaskCreate_SizeFlag_LabelL(t *testing.T) {
	var capturedInput services.CreateTaskInput
	stub := &mockTaskCreateService{
		createFn: func(_ context.Context, input services.CreateTaskInput) (*models.Task, error) {
			capturedInput = input
			return &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: input.Title}}, nil
		},
	}
	withTaskCreateSvcOverride(t, stub)

	cmd := buildTaskCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=L", "E07", "F01", "My task"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if capturedInput.Size == nil {
		t.Fatal("expected Size non-nil after --size L")
	}
	if *capturedInput.Size != 5 {
		t.Errorf("expected 5 (L), got %d", *capturedInput.Size)
	}
}

func TestTaskCreate_SizeFlag_Numeric8(t *testing.T) {
	var capturedInput services.CreateTaskInput
	stub := &mockTaskCreateService{
		createFn: func(_ context.Context, input services.CreateTaskInput) (*models.Task, error) {
			capturedInput = input
			return &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: input.Title}}, nil
		},
	}
	withTaskCreateSvcOverride(t, stub)

	cmd := buildTaskCreateCmdWithSize(t)
	cmd.SetArgs([]string{"--size=8", "E07", "F01", "My task"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if capturedInput.Size == nil {
		t.Fatal("expected Size non-nil after --size 8")
	}
	if *capturedInput.Size != 8 {
		t.Errorf("expected 8, got %d", *capturedInput.Size)
	}
}

func TestTaskCreate_SizeFlag_Absent_SizeIsNil(t *testing.T) {
	var capturedInput services.CreateTaskInput
	stub := &mockTaskCreateService{
		createFn: func(_ context.Context, input services.CreateTaskInput) (*models.Task, error) {
			capturedInput = input
			return &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: input.Title}}, nil
		},
	}
	withTaskCreateSvcOverride(t, stub)

	cmd := buildTaskCreateCmdWithSize(t)
	cmd.SetArgs([]string{"E07", "F01", "My task"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if capturedInput.Size != nil {
		t.Errorf("expected Size=nil when flag absent, got %d", *capturedInput.Size)
	}
}

func TestTaskCreate_SizeFlag_Invalid_ReturnsError(t *testing.T) {
	stub := &mockTaskCreateService{
		createFn: func(_ context.Context, input services.CreateTaskInput) (*models.Task, error) {
			t.Error("service should not have been called for invalid --size")
			return nil, nil
		},
	}
	withTaskCreateSvcOverride(t, stub)

	for _, bad := range []string{"4", "XXXL", "medium"} {
		t.Run(bad, func(t *testing.T) {
			cmd := buildTaskCreateCmdWithSize(t)
			cmd.SetArgs([]string{"--size=" + bad, "E07", "F01", "My task"})
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Errorf("expected error for --size=%q, got nil", bad)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// idea create — size flag registration and parseCreateIdeaInput tests
// ---------------------------------------------------------------------------

func TestIdeaCreate_SizeFlag_RegisteredAndIsString(t *testing.T) {
	flag := ideaCreateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on ideaCreateCmd")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %s", flag.Value.Type())
	}
}

// TestParseCreateIdeaInput_WithSizeXXL verifies that parseCreateIdeaInput
// reads ideaCreateSizeFlag and stores Size=ptr(13).
func TestParseCreateIdeaInput_WithSizeXXL(t *testing.T) {
	orig := ideaCreateSizeFlag
	ideaCreateSizeFlag = "XXL"
	defer func() { ideaCreateSizeFlag = orig }()

	input, err := parseCreateIdeaInput("My idea")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if input.Size == nil {
		t.Fatal("expected Size non-nil after size=XXL")
	}
	if *input.Size != 13 {
		t.Errorf("expected 13 (XXL), got %d", *input.Size)
	}
}

func TestParseCreateIdeaInput_SizeAbsent_SizeIsNil(t *testing.T) {
	orig := ideaCreateSizeFlag
	ideaCreateSizeFlag = ""
	defer func() { ideaCreateSizeFlag = orig }()

	input, err := parseCreateIdeaInput("My idea")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if input.Size != nil {
		t.Errorf("expected Size=nil, got %d", *input.Size)
	}
}

func TestParseCreateIdeaInput_InvalidSize_ReturnsError(t *testing.T) {
	orig := ideaCreateSizeFlag
	ideaCreateSizeFlag = "4"
	defer func() { ideaCreateSizeFlag = orig }()

	_, err := parseCreateIdeaInput("My idea")
	if err == nil {
		t.Error("expected error for invalid size \"4\"")
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize, got %v", err)
	}
}

func TestParseCreateIdeaInput_ClearIsInvalid_ReturnsError(t *testing.T) {
	orig := ideaCreateSizeFlag
	ideaCreateSizeFlag = "clear"
	defer func() { ideaCreateSizeFlag = orig }()

	_, err := parseCreateIdeaInput("My idea")
	if err == nil {
		t.Error("expected error for --size=clear on create")
	}
	if !errors.Is(err, models.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize for 'clear', got %v", err)
	}
}

// ---------------------------------------------------------------------------
// epic create / feature create — flag registration
// ---------------------------------------------------------------------------

func TestEpicCreateCmd_SizeFlagIsStringType(t *testing.T) {
	flag := epicCreateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on epicCreateCmd")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %s", flag.Value.Type())
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
}

func TestFeatureCreateCmd_SizeFlagIsStringType(t *testing.T) {
	flag := featureCreateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on featureCreateCmd")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %s", flag.Value.Type())
	}
	if flag.DefValue != "" {
		t.Errorf("expected default \"\", got %q", flag.DefValue)
	}
}

// ---------------------------------------------------------------------------
// command builders
// ---------------------------------------------------------------------------

func buildBugCreateCmdWithSize(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "create <title>",
		Args: cobraExactArgs(1),
		RunE: runBugCreate,
	}
	cmd.Flags().String("severity", "", "severity")
	cmd.Flags().String("link", "", "link")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().Bool("force", false, "force")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	cmd.Flags().String("size", "", "size")
	return cmd
}

func buildChangeCreateCmdWithSize(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "create <title>",
		Args: cobraExactArgs(1),
		RunE: runChangeCreate,
	}
	cmd.Flags().String("link", "", "link")
	cmd.Flags().String("description", "", "description")
	cmd.Flags().Int("priority", 0, "priority")
	cmd.Flags().String("requested-by", "", "requested-by")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().Bool("force", false, "force")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	cmd.Flags().String("size", "", "size")
	return cmd
}

func buildTaskCreateCmdWithSize(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "create [EPIC] [FEATURE] <title>",
		Args: cobra.RangeArgs(1, 3),
		RunE: runTaskCreate,
	}
	registerCreateFlags(cmd) // already registers --size flag
	return cmd
}

// withChangeCardSvcOverrideForSize installs a test override for changeCardSvcOverride.
// Named with the "ForSize" suffix to avoid a redeclaration conflict with the
// helper in change_test.go.
func withChangeCardSvcOverrideForSize(t *testing.T, svc changeCardServicer) {
	t.Helper()
	orig := changeCardSvcOverride
	changeCardSvcOverride = svc
	t.Cleanup(func() { changeCardSvcOverride = orig })
}

// ---------------------------------------------------------------------------
// mockTaskCreateService — narrow stub implementing taskCreateServicer
// ---------------------------------------------------------------------------

type mockTaskCreateService struct {
	createFn func(ctx context.Context, input services.CreateTaskInput) (*models.Task, error)
}

func (m *mockTaskCreateService) CreateTask(ctx context.Context, input services.CreateTaskInput) (*models.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: input.Title}}, nil
}

// Compile-time check.
var _ taskCreateServicer = (*mockTaskCreateService)(nil)

// withTaskCreateSvcOverride installs a test override for taskCreateSvcOverride.
func withTaskCreateSvcOverride(t *testing.T, svc taskCreateServicer) {
	t.Helper()
	orig := taskCreateSvcOverride
	taskCreateSvcOverride = svc
	t.Cleanup(func() { taskCreateSvcOverride = orig })
}
