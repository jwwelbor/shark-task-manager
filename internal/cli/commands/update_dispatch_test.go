package commands

// update_dispatch_test.go
//
// Regression tests for the unified `shark update <KEY>` dispatch command.
//
// The dispatch was historically missing a `--size` flag registration even
// though every per-entity update subcommand (epic/feature/task/bug/change/idea
// update) registered its own. Because Cobra rejects unknown flags before
// invoking RunE, `shark update E07 --size=L` returned `unknown flag: --size`.
// The dispatch handler delegates to `runEpicUpdate(cmd, args)` etc. with the
// dispatch's own `cmd`, so registering the flag here is sufficient — every
// entity runner reads the size value via `parseSizeUpdateFlag(cmd)`.

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// TestUpdateDispatch_SizeFlagRegistered asserts that --size is registered on
// the unified `update` dispatch command as a string-typed flag with no default.
// Mirrors the per-entity check in TestSizeFlag_RegisteredOnAllUpdateCommands.
func TestUpdateDispatch_SizeFlagRegistered(t *testing.T) {
	flag := updateCmd.Flags().Lookup("size")
	if flag == nil {
		t.Fatal("--size flag not registered on updateCmd (dispatch)")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected --size to be string type on updateCmd, got %s", flag.Value.Type())
	}
	if flag.DefValue != "" {
		t.Errorf("expected --size default \"\" on updateCmd, got %q", flag.DefValue)
	}
}

// TestUpdateDispatch_SizeFlagInLongHelp asserts that the dispatch's Long
// description mentions the `--size` flag so `shark update --help` documents it.
func TestUpdateDispatch_SizeFlagInLongHelp(t *testing.T) {
	if updateCmd.Long == "" {
		t.Fatal("updateCmd.Long is empty")
	}
	// We only assert the flag name appears; the surrounding wording is
	// allowed to evolve.
	if !containsAll(updateCmd.Long, "--size") {
		t.Errorf("updateCmd.Long does not mention --size:\n%s", updateCmd.Long)
	}
}

// TestUpdateDispatch_BugSizeFlow runs the full dispatch path for a bug key
// with `--size=L` and verifies the bug service receives Size=ptr(5). This
// exercises the wiring from updateCmd flag → DetectEntityType → runBugUpdate
// → parseSizeUpdateFlag → BugUpdates DTO.
func TestUpdateDispatch_BugSizeFlow(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "ok"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildIsolatedDispatchCmd(t)
	cmd.SetArgs([]string{"B001", "--size=L"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dispatch execute error: %v", err)
	}

	if capturedUpdates.Size == nil {
		t.Fatal("expected BugUpdates.Size to be set after --size=L via dispatch")
	}
	if *capturedUpdates.Size != 5 {
		t.Errorf("expected Size=5 (L), got %d", *capturedUpdates.Size)
	}
	if capturedUpdates.ClearSize {
		t.Error("expected ClearSize=false for --size=L")
	}
}

// TestUpdateDispatch_BugSizeClearFlow verifies that `--size=clear` propagates
// through the dispatch to BugUpdates.ClearSize=true.
func TestUpdateDispatch_BugSizeClearFlow(t *testing.T) {
	var capturedUpdates services.BugUpdates
	stub := &mockBugServiceForTags{
		updateBugFn: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedUpdates = updates
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "ok"}}, nil
		},
	}
	withBugSvcOverride(t, stub)

	cmd := buildIsolatedDispatchCmd(t)
	cmd.SetArgs([]string{"B001", "--size=clear"})
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dispatch execute error: %v", err)
	}

	if !capturedUpdates.ClearSize {
		t.Error("expected ClearSize=true for --size=clear via dispatch")
	}
	if capturedUpdates.Size != nil {
		t.Errorf("expected Size=nil when ClearSize=true, got %d", *capturedUpdates.Size)
	}
}

// buildIsolatedDispatchCmd builds an isolated cobra command that mirrors the
// real `updateCmd` flag registration and dispatches via runUpdate. Building
// in isolation avoids polluting the package-level command tree across tests.
func buildIsolatedDispatchCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update <KEY> [flags]",
		Args: cobra.ExactArgs(1),
		RunE: runUpdate,
	}
	// Mirror the flags registered on the real updateCmd so runUpdate's
	// downstream entity runners can read them via cmd.Flags().
	cmd.Flags().String("title", "", "title")
	cmd.Flags().StringP("description", "d", "", "description")
	cmd.Flags().Int("order", -1, "order")
	cmd.Flags().String("key", "", "key")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().String("filename", "", "alias for --file")
	cmd.Flags().String("path", "", "alias for --file")
	cmd.Flags().Bool("force", false, "force")
	cmd.Flags().StringP("priority", "p", "", "priority")
	cmd.Flags().StringP("agent", "a", "", "agent")
	cmd.Flags().String("depends-on", "", "depends-on")
	cmd.Flags().String("business-value", "", "business-value")
	cmd.Flags().String("severity", "", "severity")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag")
	cmd.Flags().String("size", "", "size")
	return cmd
}

// containsAll is a tiny helper that returns true iff `haystack` contains every
// substring in `needles`. Avoids a strings.Contains import collision and keeps
// the test self-contained.
func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for i := 0; i+len(n) <= len(haystack); i++ {
			if haystack[i:i+len(n)] == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
