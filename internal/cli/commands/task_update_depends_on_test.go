package commands

// task_update_depends_on_test.go — B048
//
// Regression tests for "task update --depends-on is a no-op — reports
// SUCCESS but never writes tasks.depends_on".
//
// Root cause (per docs/plan/bugs/B048.research-report.md): parseTaskUpdates
// had no code block reading the --depends-on flag at all, so the value was
// silently discarded before it ever reached TaskUpdates or the service.
// These tests exercise parseTaskUpdates directly (the production entry
// point used by `shark task update` and the unified `shark update`), so
// they fail against the pre-fix code where TaskUpdates.DependsOn didn't
// exist.
//
// Rules:
//   - All tests use mocked services (never real DB) per
//     .claude/rules/testing/cli-tests.md.
//   - Mirrors the --size three-way dispatch pattern in
//     size_flag_update_test.go: flag absent -> no-op; flag present empty ->
//     clear; flag present with keys -> set the JSON-encoded list.

import (
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

func TestReadStringSliceFromFlagNormalizesSupportedDependsOnFlags(t *testing.T) {
	tests := []struct {
		name     string
		flagType string
		value    string
		want     []string
	}{
		{name: "string ordinary", flagType: "string", value: "T-E07-F01-002,T-E07-F01-003", want: []string{"T-E07-F01-002", "T-E07-F01-003"}},
		{name: "string trims whitespace and empty members", flagType: "string", value: " T-E07-F01-002 , , T-E07-F01-003 ", want: []string{"T-E07-F01-002", "T-E07-F01-003"}},
		{name: "string comma only", flagType: "string", value: " , , ", want: []string{}},
		{name: "string empty", flagType: "string", value: "", want: []string{}},
		{name: "string slice", flagType: "stringSlice", value: "T-E07-F01-002,T-E07-F01-003", want: []string{"T-E07-F01-002", "T-E07-F01-003"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "update"}
			if tc.flagType == "stringSlice" {
				cmd.Flags().StringSlice("depends-on", nil, "dependency keys")
			} else {
				cmd.Flags().String("depends-on", "", "dependency keys")
			}
			if err := cmd.Flags().Set("depends-on", tc.value); err != nil {
				t.Fatalf("set --depends-on: %v", err)
			}

			got, err := readStringSliceFromFlag(cmd, "depends-on")
			if err != nil {
				t.Fatalf("readStringSliceFromFlag() error = %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("readStringSliceFromFlag() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// --depends-on with keys sets TaskUpdates.DependsOn to the JSON-encoded list.
func TestTaskUpdate_DependsOnFlag_SetsDependsOn(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--depends-on=T-E07-F01-002"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.DependsOn == nil {
		t.Fatal("expected DependsOn non-nil after --depends-on=T-E07-F01-002")
	}
	if *capturedUpdates.DependsOn != `["T-E07-F01-002"]` {
		t.Errorf("expected DependsOn=%q, got %q", `["T-E07-F01-002"]`, *capturedUpdates.DependsOn)
	}
	if capturedUpdates.ClearDependsOn {
		t.Error("expected ClearDependsOn=false")
	}
}

// --depends-on with a comma-separated list encodes all keys.
func TestTaskUpdate_DependsOnFlag_MultipleKeys_SetsDependsOn(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--depends-on=T-E07-F01-002,T-E07-F01-003"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.DependsOn == nil {
		t.Fatal("expected DependsOn non-nil after --depends-on with two keys")
	}
	want := `["T-E07-F01-002","T-E07-F01-003"]`
	if *capturedUpdates.DependsOn != want {
		t.Errorf("expected DependsOn=%q, got %q", want, *capturedUpdates.DependsOn)
	}
}

// --depends-on="" (explicit empty value) clears dependencies.
func TestTaskUpdate_DependsOnFlag_Empty_ClearsDependsOn(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001", "--depends-on="})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if !capturedUpdates.ClearDependsOn {
		t.Error("expected ClearDependsOn=true when --depends-on=\"\"")
	}
	if capturedUpdates.DependsOn != nil {
		t.Errorf("expected DependsOn=nil when ClearDependsOn=true, got %q", *capturedUpdates.DependsOn)
	}
}

// Flag absent -> no dependency mutation at all (matches --size's TC-F005-D).
func TestTaskUpdate_DependsOnFlag_Absent_NoMutation(t *testing.T) {
	var capturedUpdates services.TaskUpdates
	cmd := buildTaskUpdateCmdCapture(t, &capturedUpdates)
	cmd.SetArgs([]string{"E07-F01-001"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if capturedUpdates.DependsOn != nil {
		t.Errorf("expected DependsOn=nil when --depends-on absent, got %q", *capturedUpdates.DependsOn)
	}
	if capturedUpdates.ClearDependsOn {
		t.Error("expected ClearDependsOn=false when --depends-on absent")
	}
}

// TestUnifiedTaskUpdate_DependsOnFlag_ThreeWayDispatch verifies the unified
// command's string-typed flag reaches the same parser contract as `task
// update`: set, explicit clear, and absent no-op remain distinct.
func TestUnifiedTaskUpdate_DependsOnFlag_ThreeWayDispatch(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		wantJSON  string
		wantClear bool
	}{
		{name: "set", args: []string{"E07-F01-001", "--depends-on=T-E07-F01-002"}, wantJSON: `["T-E07-F01-002"]`},
		{name: "clear", args: []string{"E07-F01-001", "--depends-on=   ,  "}, wantClear: true},
		{name: "absent", args: []string{"E07-F01-001"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "update <key>",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, _ []string) error {
					updates, err := parseTaskUpdates(cmd)
					if err != nil {
						return err
					}
					if updates.ClearDependsOn != tc.wantClear {
						t.Errorf("ClearDependsOn = %v, want %v", updates.ClearDependsOn, tc.wantClear)
					}
					if tc.wantJSON == "" {
						if updates.DependsOn != nil {
							t.Errorf("DependsOn = %q, want nil", *updates.DependsOn)
						}
					} else if updates.DependsOn == nil || *updates.DependsOn != tc.wantJSON {
						t.Errorf("DependsOn = %v, want %q", updates.DependsOn, tc.wantJSON)
					}
					return nil
				},
			}
			cmd.Flags().String("depends-on", "", "New dependency keys, comma-separated (task & idea)")
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute returned error: %v", err)
			}
		})
	}
}

// TestUpdateCmd_DependsOnFlag_IsStringType guards the unified `shark update`
// entry point against a flag-registration regression: the dependency parser
// deliberately reads --depends-on as a string and returns an error for any
// other flag kind. Pin the type so the command keeps its intended contract
// rather than rejecting valid task updates after an unrelated flag change.
func TestUpdateCmd_DependsOnFlag_IsStringType(t *testing.T) {
	f := updateCmd.Flags().Lookup("depends-on")
	if f == nil {
		t.Fatal("expected --depends-on flag registered on the unified update command")
	}
	if f.Value.Type() != "string" {
		t.Errorf("expected unified update --depends-on flag type=string (parseTaskUpdates uses GetString), got %q", f.Value.Type())
	}
}

// TestRegisterUpdateFlags_DependsOnFlag_IsStringType is the entity-specific
// `shark task update` counterpart to TestUpdateCmd_DependsOnFlag_IsStringType.
func TestRegisterUpdateFlags_DependsOnFlag_IsStringType(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	registerUpdateFlags(cmd)
	f := cmd.Flags().Lookup("depends-on")
	if f == nil {
		t.Fatal("expected --depends-on flag registered by registerUpdateFlags")
	}
	if f.Value.Type() != "string" {
		t.Errorf("expected --depends-on flag type=string (parseTaskUpdates uses GetString), got %q", f.Value.Type())
	}
}
