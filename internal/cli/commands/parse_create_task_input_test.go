package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// newTestTaskCreateCmd builds a cobra.Command with all the flags that
// parseCreateTaskInput reads.  This mirrors registerCreateFlags without
// calling the real init/registration logic.
func newTestTaskCreateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().StringP("epic", "e", "", "Epic key")
	cmd.Flags().StringP("feature", "f", "", "Feature key")
	cmd.Flags().StringP("agent", "a", "", "Agent type")
	cmd.Flags().StringP("description", "d", "", "Description")
	cmd.Flags().Int("order", 0, "Execution order")
	cmd.Flags().Int("execution-order", 0, "Execution order alias")
	cmd.Flags().IntP("priority", "p", 5, "Priority 1-10")
	cmd.Flags().String("depends-on", "", "Comma-separated dependency keys")
	cmd.Flags().Bool("force", false, "Force")
	cmd.Flags().Bool("create", false, "Create file if missing")
	cmd.Flags().String("file", "", "File path")
	cmd.Flags().String("filename", "", "Alias for --file")
	cmd.Flags().String("path", "", "Alias for --file")
	cmd.Flags().String("key", "", "Custom task key")
	return cmd
}

// TestParseCreateTaskInput_ThreeArgFormat verifies the 3-arg positional form:
// args = ["E07", "F01", "Task Title"]
func TestParseCreateTaskInput_ThreeArgFormat(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	args := []string{"E07", "F01", "My Task Title"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "EpicKey", "E07", got.EpicKey)
	assertCreateTaskInputField(t, "FeatureKey", "F01", got.FeatureKey)
	assertCreateTaskInputField(t, "Title", "My Task Title", got.Title)
}

// TestParseCreateTaskInput_TwoArgCombinedFormat verifies the 2-arg form where the
// first argument is a combined "E##-F##" key:
// args = ["E07-F01", "Task Title"]
func TestParseCreateTaskInput_TwoArgCombinedFormat(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	args := []string{"E07-F01", "My Task Title"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "EpicKey", "E07", got.EpicKey)
	// featureKey keeps the full combined value (that is the implementation behaviour)
	assertCreateTaskInputField(t, "FeatureKey", "E07-F01", got.FeatureKey)
	assertCreateTaskInputField(t, "Title", "My Task Title", got.Title)
}

// TestParseCreateTaskInput_TwoArgEpicOnly verifies 2-arg form where first arg is
// just an epic key (no dash):
// args = ["E07", "Task Title"]
func TestParseCreateTaskInput_TwoArgEpicOnly(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	args := []string{"E07", "My Task Title"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "EpicKey", "E07", got.EpicKey)
	// No feature in args; featureKey stays empty unless flag is set.
	assertCreateTaskInputField(t, "FeatureKey", "", got.FeatureKey)
	assertCreateTaskInputField(t, "Title", "My Task Title", got.Title)
}

// TestParseCreateTaskInput_FlagFormat verifies fallback to --epic / --feature flags
// when only a title is provided as a positional arg:
// args = ["Task Title"], --epic=E07, --feature=F01
func TestParseCreateTaskInput_FlagFormat(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("epic", "E07"); err != nil {
		t.Fatalf("failed to set epic flag: %v", err)
	}
	if err := cmd.Flags().Set("feature", "F01"); err != nil {
		t.Fatalf("failed to set feature flag: %v", err)
	}
	args := []string{"My Task Title"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "EpicKey", "E07", got.EpicKey)
	assertCreateTaskInputField(t, "FeatureKey", "F01", got.FeatureKey)
	assertCreateTaskInputField(t, "Title", "My Task Title", got.Title)
}

// TestParseCreateTaskInput_OptionalFlags verifies that optional flags are read
// correctly when present.
func TestParseCreateTaskInput_OptionalFlags(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("agent", "backend"); err != nil {
		t.Fatalf("failed to set agent flag: %v", err)
	}
	if err := cmd.Flags().Set("priority", "3"); err != nil {
		t.Fatalf("failed to set priority flag: %v", err)
	}
	if err := cmd.Flags().Set("order", "2"); err != nil {
		t.Fatalf("failed to set order flag: %v", err)
	}
	if err := cmd.Flags().Set("file", "docs/custom/task.md"); err != nil {
		t.Fatalf("failed to set file flag: %v", err)
	}
	if err := cmd.Flags().Set("force", "true"); err != nil {
		t.Fatalf("failed to set force flag: %v", err)
	}
	if err := cmd.Flags().Set("create", "true"); err != nil {
		t.Fatalf("failed to set create flag: %v", err)
	}
	args := []string{"E07", "F01", "Flagged Task"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "EpicKey", "E07", got.EpicKey)
	assertCreateTaskInputField(t, "FeatureKey", "F01", got.FeatureKey)
	assertCreateTaskInputField(t, "Title", "Flagged Task", got.Title)
	assertCreateTaskInputField(t, "AgentType", "backend", got.AgentType)
	assertCreateTaskInputInt(t, "Priority", 3, got.Priority)
	assertCreateTaskInputInt(t, "ExecutionOrder", 2, got.ExecutionOrder)
	assertCreateTaskInputField(t, "FilePath", "docs/custom/task.md", got.FilePath)
	assertCreateTaskInputBool(t, "Force", true, got.Force)
	assertCreateTaskInputBool(t, "CreateFile", true, got.CreateFile)
}

// TestParseCreateTaskInput_DependsOn verifies comma-separated dependency parsing.
func TestParseCreateTaskInput_DependsOn(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("depends-on", "E07-F01-001, E07-F01-002"); err != nil {
		t.Fatalf("failed to set depends-on flag: %v", err)
	}
	args := []string{"E07", "F01", "Dependent Task"}

	got := parseCreateTaskInput(cmd, args)

	if len(got.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d: %v", len(got.DependsOn), got.DependsOn)
		return
	}
	if got.DependsOn[0] != "T-E07-F01-001" {
		t.Errorf("expected DependsOn[0]=%q, got %q", "T-E07-F01-001", got.DependsOn[0])
	}
	if got.DependsOn[1] != "T-E07-F01-002" {
		t.Errorf("expected DependsOn[1]=%q, got %q", "T-E07-F01-002", got.DependsOn[1])
	}
}

func TestParseCreateTaskInput_DependsOnPreservesInvalidKeyForValidation(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("depends-on", "not-a-task-key"); err != nil {
		t.Fatalf("set depends-on flag: %v", err)
	}

	got := parseCreateTaskInput(cmd, []string{"E07", "F01", "Dependent Task"})
	if len(got.DependsOn) != 1 || got.DependsOn[0] != "not-a-task-key" {
		t.Errorf("DependsOn = %#v, want []string{\"not-a-task-key\"}", got.DependsOn)
	}
}

// TestParseCreateTaskInput_EmptyArgs verifies that an empty arg list returns a
// zero-value CreateTaskInput (flags also not set).
func TestParseCreateTaskInput_EmptyArgs(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	args := []string{}

	got := parseCreateTaskInput(cmd, args)

	// All fields should be empty/zero when no args and no flags provided.
	want := services.CreateTaskInput{Priority: 5} // default priority from flag default
	if got.EpicKey != want.EpicKey {
		t.Errorf("EpicKey: expected %q, got %q", want.EpicKey, got.EpicKey)
	}
	if got.FeatureKey != want.FeatureKey {
		t.Errorf("FeatureKey: expected %q, got %q", want.FeatureKey, got.FeatureKey)
	}
	if got.Title != want.Title {
		t.Errorf("Title: expected %q, got %q", want.Title, got.Title)
	}
}

// TestParseCreateTaskInput_ExecutionOrderAlias verifies that --execution-order is
// treated as an alias for --order when --order is not set.
func TestParseCreateTaskInput_ExecutionOrderAlias(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("execution-order", "5"); err != nil {
		t.Fatalf("failed to set execution-order flag: %v", err)
	}
	args := []string{"E07", "F01", "Ordered Task"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputInt(t, "ExecutionOrder", 5, got.ExecutionOrder)
}

// TestParseCreateTaskInput_OrderTakesPrecedenceOverAlias verifies that --order takes
// precedence over --execution-order when both are set.
func TestParseCreateTaskInput_OrderTakesPrecedenceOverAlias(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("order", "3"); err != nil {
		t.Fatalf("failed to set order flag: %v", err)
	}
	if err := cmd.Flags().Set("execution-order", "9"); err != nil {
		t.Fatalf("failed to set execution-order flag: %v", err)
	}
	args := []string{"E07", "F01", "Priority Order Task"}

	got := parseCreateTaskInput(cmd, args)

	// --order should win because the implementation only reads --execution-order
	// when order == 0.
	assertCreateTaskInputInt(t, "ExecutionOrder", 3, got.ExecutionOrder)
}

// TestParseCreateTaskInput_FilenameAlias verifies that --filename is treated as an
// alias for --file when --file is empty.
func TestParseCreateTaskInput_FilenameAlias(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("filename", "docs/custom/via-filename.md"); err != nil {
		t.Fatalf("failed to set filename flag: %v", err)
	}
	args := []string{"E07", "F01", "Alias Task"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "FilePath", "docs/custom/via-filename.md", got.FilePath)
}

// TestParseCreateTaskInput_CustomKey covers B063: the --key flag was
// registered on the task create command but parseCreateTaskInput never read
// it, so services.CreateTaskInput.CustomKey was always empty and a supplied
// key was silently ignored.
func TestParseCreateTaskInput_CustomKey(t *testing.T) {
	cmd := newTestTaskCreateCmd()
	if err := cmd.Flags().Set("key", "T-E07-F01-099"); err != nil {
		t.Fatalf("failed to set key flag: %v", err)
	}
	args := []string{"E07", "F01", "Custom Key Task"}

	got := parseCreateTaskInput(cmd, args)

	assertCreateTaskInputField(t, "CustomKey", "T-E07-F01-099", got.CustomKey)
}

// ---- helpers ----------------------------------------------------------------

func assertCreateTaskInputField(t *testing.T, field, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %q, got %q", field, want, got)
	}
}

func assertCreateTaskInputInt(t *testing.T, field string, want, got int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %d, got %d", field, want, got)
	}
}

func assertCreateTaskInputBool(t *testing.T, field string, want, got bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %v, got %v", field, want, got)
	}
}
