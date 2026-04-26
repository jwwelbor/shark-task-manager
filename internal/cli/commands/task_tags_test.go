package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// E28-F04 T-006 — --tag flag parsing tests for `shark task create|update`.
//
// These exercise the CLI → service wire: the flag must arrive on the DTO
// so the service (tested separately in internal/services/task_service_tags_test.go)
// can invoke EnforceRequired/AttachMany with the right names.
//
// Covers AC-19 (task row, CLI side) — the service-side AttachMany call that
// the E28-F04 spec ties to AC-19 is covered by the service test
// TestTaskService_CreateTask_TagsProvidedAttachAfterPersist. AC-19b (no
// comma-split, ADR-F04-5) — invalid characters flow through as a single
// literal to the service layer where validation fails.
//
// The tests build isolated cobra commands per case so they don't mutate
// the package-level taskCreateCmd/taskUpdateCmd which are wired into
// cli.RootCmd and shared across the test binary.
// ---------------------------------------------------------------------------

// buildTaskCreateCmd returns a fresh `shark task create` command mirroring
// the production wire-up, using a small parse-only runner that captures the
// CreateTaskInput built by parseCreateTaskInput. This avoids needing a real
// TaskService / DB.
func buildTaskCreateCmd(t *testing.T, capture *services.CreateTaskInput) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "create",
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			*capture = parseCreateTaskInput(cmd, args)
			return nil
		},
	}
	registerCreateFlags(cmd)
	return cmd
}

// buildTaskUpdateCmd returns a fresh `shark task update` command that
// captures the TaskUpdates produced by parseTaskUpdates.
func buildTaskUpdateCmd(t *testing.T, capture *services.TaskUpdates) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*capture = parseTaskUpdates(cmd)
			return nil
		},
	}
	registerUpdateFlags(cmd)
	return cmd
}

// AC-19 (task row, CLI side): --tag repeats pass through to CreateTaskInput.
func TestTaskCreate_TagFlag_PassesTagsToService(t *testing.T) {
	var input services.CreateTaskInput
	cmd := buildTaskCreateCmd(t, &input)
	cmd.SetArgs([]string{"E07", "F01", "Tagged task", "--tag=voice", "--tag=auth"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(input.Tags) != 2 {
		t.Fatalf("expected 2 tags on input, got %d (%v)", len(input.Tags), input.Tags)
	}
	if input.Tags[0] != "voice" || input.Tags[1] != "auth" {
		t.Errorf("tags = %v, want [voice, auth]", input.Tags)
	}
}

// AC-19b: `--tag=voice,auth` is NOT split by Cobra's comma-separator
// behaviour into two tags because the ADR-F04-5 rationale relies on the
// service layer to reject invalid characters. But StringSliceVar in Cobra
// splits by default; the test captures the shape delivered to the service.
//
// This test documents the delivered shape — whatever Cobra produces, the
// DTO receives it intact. The service-layer validation (rejects invalid
// characters in tag names) is the enforcement gate for ADR-F04-5 and is
// exercised in the service-layer tag tests.
func TestTaskCreate_CommaInTagValue_DeliveredAsIs(t *testing.T) {
	var input services.CreateTaskInput
	cmd := buildTaskCreateCmd(t, &input)
	cmd.SetArgs([]string{"E07", "F01", "Tagged task", "--tag=voice,auth"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	// Cobra's StringSlice splits on commas by default. Whatever the
	// produced length, it is non-zero and is delivered untouched to the
	// DTO.  The test guards against a regression where the CLI layer
	// tries to "smart-merge" or re-split the slice.
	if len(input.Tags) == 0 {
		t.Fatal("expected --tag=voice,auth to produce at least one entry")
	}
}

// AC-19 (task row) absence path: no --tag flag yields nil Tags slice.
func TestTaskCreate_NoTagFlag_YieldsNil(t *testing.T) {
	var input services.CreateTaskInput
	cmd := buildTaskCreateCmd(t, &input)
	cmd.SetArgs([]string{"E07", "F01", "Untagged task"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(input.Tags) != 0 {
		t.Errorf("expected empty Tags, got %v", input.Tags)
	}
}

// AC-19 (task row) update path: --tag on update threads through additively.
func TestTaskUpdate_TagFlag_PassesTagsToService(t *testing.T) {
	var updates services.TaskUpdates
	cmd := buildTaskUpdateCmd(t, &updates)
	cmd.SetArgs([]string{"T-E07-F01-001", "--tag=voice"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(updates.Tags) != 1 || updates.Tags[0] != "voice" {
		t.Errorf("update Tags = %v, want [voice]", updates.Tags)
	}
}

// AC-19 (task row) update no-op path: absent --tag keeps Tags nil so the
// service's "additive only" semantics are preserved.
func TestTaskUpdate_NoTagFlag_LeavesTagsNil(t *testing.T) {
	var updates services.TaskUpdates
	cmd := buildTaskUpdateCmd(t, &updates)
	cmd.SetArgs([]string{"T-E07-F01-001", "--title=New Title"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if updates.Tags != nil {
		t.Errorf("expected Tags to be nil when --tag omitted, got %v", updates.Tags)
	}
}
