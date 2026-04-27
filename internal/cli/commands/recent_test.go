package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Mock RecentService (function-field pattern per .claude/rules/services/testing.md)
// ---------------------------------------------------------------------------

// recentServicer is the minimal interface consumed by recent.go.
// Defined locally in the test file so we can inject a lightweight mock.
type recentServicer interface {
	ListRecent(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error)
}

// mockRecentService is a test double for RecentService.
type mockRecentService struct {
	ListRecentFunc func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error)
}

func (m *mockRecentService) ListRecent(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
	if m.ListRecentFunc != nil {
		return m.ListRecentFunc(ctx, filters)
	}
	return nil, fmt.Errorf("ListRecent not implemented in mock")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// captureRecentOutput redirects stdout and returns what was printed.
// (capturingOutput already defined in analytics_test.go for other tests.)
func captureRecentOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// buildRecentCmd returns a fresh cobra command wired to parseRecentFiltersWithConfig
// and runRecentWithSvc, enabling per-test mock injection without touching global
// state or the real database.
func buildRecentCmd(svc recentServicer, cfgLimit int) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "recent [N]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filters, err := parseRecentFiltersWithConfig(cmd, args, cfgLimit)
			if err != nil {
				return err
			}
			return runRecentWithSvc(cmd.Context(), filters, svc)
		},
	}
	cmd.Flags().Int("limit", 0, "Limit results")
	cmd.Flags().Bool("tasks", false, "Show only tasks")
	cmd.Flags().Bool("features", false, "Show only features")
	cmd.Flags().Bool("epics", false, "Show only epics")
	cmd.Flags().Bool("bugs", false, "Show only bugs")
	cmd.Flags().Bool("changes", false, "Show only change-cards")
	cmd.Flags().Bool("ideas", false, "Show only ideas")
	cmd.Flags().Bool("tech-debt", false, "Show only tech-debt items")
	return cmd
}

// ---------------------------------------------------------------------------
// IS-3: CLI argument parsing to service call (mocked service)
// ---------------------------------------------------------------------------

// TestRunRecent_ParsesPositionalLimit verifies that a positional integer
// argument sets filters.Limit correctly.
func TestRunRecent_ParsesPositionalLimit(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 5)
	cmd.SetArgs([]string{"10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFilters.Limit != 10 {
		t.Errorf("expected filters.Limit=10, got %d", capturedFilters.Limit)
	}
}

// TestRunRecent_LimitFlagOverridesPositional verifies that --limit=N wins over
// the positional argument (REQ-F-004).
func TestRunRecent_LimitFlagOverridesPositional(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	// Suppress warning output in this test
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	// Redirect stdout to suppress table output
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = old
	}()

	cmd := buildRecentCmd(mock, 5)
	cmd.SetArgs([]string{"5", "--limit=20"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFilters.Limit != 20 {
		t.Errorf("expected filters.Limit=20 (flag wins), got %d", capturedFilters.Limit)
	}
}

// TestRunRecent_FallsBackToConfigDefault verifies that when no args/flags are
// given, the config default is used (REQ-F-002).
func TestRunRecent_FallsBackToConfigDefault(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 7) // config says 7
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFilters.Limit != 7 {
		t.Errorf("expected filters.Limit=7 (config default), got %d", capturedFilters.Limit)
	}
}

// TestRunRecent_FallsBackToBuiltInDefault verifies that when config returns
// 0 or negative, the built-in default of 5 is used (REQ-F-002).
func TestRunRecent_FallsBackToBuiltInDefault(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 0) // config returns 0 → built-in default
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFilters.Limit != 5 {
		t.Errorf("expected filters.Limit=5 (built-in default), got %d", capturedFilters.Limit)
	}
}

// TestRunRecent_InvalidLimitReturnsExit3 verifies that a non-integer positional
// argument returns an error (mapped to exit code 3) and the service is not called.
func TestRunRecent_InvalidLimitReturnsExit3(t *testing.T) {
	serviceCallCount := 0

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			serviceCallCount++
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 5)
	cmd.SetArgs([]string{"abc"})
	err := cmd.Execute()

	if err == nil {
		t.Fatal("expected error for invalid limit 'abc', got nil")
	}
	if !strings.Contains(err.Error(), "exit code 3:") {
		t.Errorf("expected exit code 3 prefix in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Errorf("expected error message to name the offending argument 'abc', got: %v", err)
	}
	if serviceCallCount > 0 {
		t.Errorf("expected service NOT to be called on invalid input, but it was called %d times", serviceCallCount)
	}
}

// TestRunRecent_ExtendedTypeFlagsSetCorrectly verifies that --bugs, --changes,
// --ideas, and --tech-debt flags populate RecentFilters correctly so all entity
// types behave consistently with the original task/feature/epic flags.
func TestRunRecent_ExtendedTypeFlagsSetCorrectly(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want func(f services.RecentFilters) bool
	}{
		{"bugs flag", "--bugs", func(f services.RecentFilters) bool { return f.IncludeBugs }},
		{"changes flag", "--changes", func(f services.RecentFilters) bool { return f.IncludeChanges }},
		{"ideas flag", "--ideas", func(f services.RecentFilters) bool { return f.IncludeIdeas }},
		{"tech-debt flag", "--tech-debt", func(f services.RecentFilters) bool { return f.IncludeTechDebt }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedFilters services.RecentFilters

			mock := &mockRecentService{
				ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
					capturedFilters = filters
					return []services.RecentItem{}, nil
				},
			}

			cmd := buildRecentCmd(mock, 5)
			cmd.SetArgs([]string{tt.flag})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.want(capturedFilters) {
				t.Errorf("expected %s to be set in filters, got: %+v", tt.flag, capturedFilters)
			}
		})
	}
}

// TestRunRecent_AllEntityFlagsCombineCorrectly verifies that multiple entity-type
// flags can be combined and all are propagated to the service layer.
func TestRunRecent_AllEntityFlagsCombineCorrectly(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 5)
	cmd.SetArgs([]string{"--tasks", "--features", "--epics", "--bugs", "--changes", "--ideas", "--tech-debt"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedFilters.IncludeTasks || !capturedFilters.IncludeFeatures || !capturedFilters.IncludeEpics ||
		!capturedFilters.IncludeBugs || !capturedFilters.IncludeChanges ||
		!capturedFilters.IncludeIdeas || !capturedFilters.IncludeTechDebt {
		t.Errorf("expected all Include* fields to be true, got: %+v", capturedFilters)
	}
}

// TestRunRecent_TypeFlagsSetCorrectly verifies that --tasks and --epics flags
// populate RecentFilters correctly (REQ-F-005).
func TestRunRecent_TypeFlagsSetCorrectly(t *testing.T) {
	var capturedFilters services.RecentFilters

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			capturedFilters = filters
			return []services.RecentItem{}, nil
		},
	}

	cmd := buildRecentCmd(mock, 5)
	cmd.SetArgs([]string{"--tasks", "--epics"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedFilters.IncludeTasks {
		t.Error("expected IncludeTasks=true")
	}
	if !capturedFilters.IncludeEpics {
		t.Error("expected IncludeEpics=true")
	}
	if capturedFilters.IncludeFeatures {
		t.Error("expected IncludeFeatures=false (flag not set)")
	}
}

// TestRunRecent_JSONOutputEmitsArray verifies that --json mode emits a valid
// JSON array of the items returned by the service (REQ-F-008).
func TestRunRecent_JSONOutputEmitsArray(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	items := []services.RecentItem{
		{Type: "task", Key: "E07-F01-001", Title: "Task 1", CreatedAt: now, Status: "todo"},
		{Type: "epic", Key: "E07", Title: "Epic 1", CreatedAt: now.Add(-time.Minute), Status: "in_progress"},
	}

	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			return items, nil
		},
	}

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	output := captureRecentOutput(func() {
		cmd := buildRecentCmd(mock, 5)
		cmd.SetArgs([]string{})
		if err := cmd.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	var parsed []services.RecentItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v\noutput: %s", err, output)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 items in JSON array, got %d", len(parsed))
	}
	if parsed[0].Type != "task" {
		t.Errorf("expected first item type 'task', got %q", parsed[0].Type)
	}
	if parsed[1].Type != "epic" {
		t.Errorf("expected second item type 'epic', got %q", parsed[1].Type)
	}
}

// TestRunRecent_EmptyStateMessageInTableMode verifies that when the service
// returns an empty slice, table mode prints "No recent items found." (REQ-F-009).
func TestRunRecent_EmptyStateMessageInTableMode(t *testing.T) {
	mock := &mockRecentService{
		ListRecentFunc: func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error) {
			return []services.RecentItem{}, nil
		},
	}

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	output := captureRecentOutput(func() {
		cmd := buildRecentCmd(mock, 5)
		cmd.SetArgs([]string{})
		if err := cmd.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "No recent items found.") {
		t.Errorf("expected 'No recent items found.' in output, got:\n%s", output)
	}
}
