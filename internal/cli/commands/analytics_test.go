package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ---------------------------------------------------------------------------
// Mock DashboardAnalyticsService
// ---------------------------------------------------------------------------

// mockDashboardAnalyticsService is a test double for DashboardAnalyticsService.
// It uses the function-field pattern consistent with other mocks in this codebase.
type mockDashboardAnalyticsService struct {
	GetBugAnalyticsFunc        func(ctx context.Context) (*services.BugAnalyticsResult, error)
	GetChangeCardAnalyticsFunc func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error)
	GetTechDebtAnalyticsFunc   func(ctx context.Context) (*services.TechDebtAnalyticsResult, error)
}

func (m *mockDashboardAnalyticsService) GetBugAnalytics(ctx context.Context) (*services.BugAnalyticsResult, error) {
	if m.GetBugAnalyticsFunc != nil {
		return m.GetBugAnalyticsFunc(ctx)
	}
	return nil, fmt.Errorf("GetBugAnalytics not implemented in mock")
}

func (m *mockDashboardAnalyticsService) GetChangeCardAnalytics(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
	if m.GetChangeCardAnalyticsFunc != nil {
		return m.GetChangeCardAnalyticsFunc(ctx)
	}
	return nil, fmt.Errorf("GetChangeCardAnalytics not implemented in mock")
}

func (m *mockDashboardAnalyticsService) GetTechDebtAnalytics(ctx context.Context) (*services.TechDebtAnalyticsResult, error) {
	if m.GetTechDebtAnalyticsFunc != nil {
		return m.GetTechDebtAnalyticsFunc(ctx)
	}
	return nil, fmt.Errorf("GetTechDebtAnalytics not implemented in mock")
}

// capturingOutput redirects stdout for the duration of fn and returns what was printed.
func capturingOutput(fn func()) string {
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

// ---------------------------------------------------------------------------
// Helper builders for test fixtures
// ---------------------------------------------------------------------------

func float64Ptr(v float64) *float64 { return &v }

func bugAnalyticsFixture() *services.BugAnalyticsResult {
	return &services.BugAnalyticsResult{
		TotalBugs: 10,
		BugsByStatus: map[string]int{
			"open":     5,
			"resolved": 3,
			"closed":   2,
		},
		BugsBySeverity: map[string]int{
			"critical": 1,
			"high":     3,
			"medium":   4,
			"low":      2,
		},
		ResolvedCount:         5,
		AvgResolutionTimeSecs: float64Ptr(86400.0), // 1 day
	}
}

func changeCardAnalyticsFixture() *services.ChangeCardAnalyticsResult {
	return &services.ChangeCardAnalyticsResult{
		TotalChangeCards: 8,
		ChangeCardsByStatus: map[string]int{
			"pending":   3,
			"approved":  4,
			"completed": 1,
		},
		ApprovalRate:          float64Ptr(0.80),
		DecidedCount:          5,
		CompletedCount:        3,
		AvgCompletionTimeSecs: float64Ptr(172800.0), // 2 days
	}
}

// ---------------------------------------------------------------------------
// TC-F07-011: Bug analytics human-readable output contains all expected metrics
// ---------------------------------------------------------------------------

func TestPrintBugAnalytics_AllMetrics(t *testing.T) {
	result := bugAnalyticsFixture()

	output := capturingOutput(func() {
		printBugAnalytics(result)
	})

	// Verify key sections are present
	if !strings.Contains(output, "Bug Analytics") {
		t.Errorf("expected 'Bug Analytics' header, got:\n%s", output)
	}
	if !strings.Contains(output, "10") {
		t.Errorf("expected total bug count 10, got:\n%s", output)
	}
	if !strings.Contains(output, "5") {
		t.Errorf("expected resolved count 5, got:\n%s", output)
	}
	// Avg resolution time should be formatted (not raw seconds)
	if !strings.Contains(output, "1d") && !strings.Contains(output, "24h") && !strings.Contains(output, "1 day") {
		t.Errorf("expected formatted avg resolution time, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// TC-F07-013: Bug analytics JSON output matches contract
// ---------------------------------------------------------------------------

func TestRunEntityAnalytics_Bug_JSON(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return bugAnalyticsFixture(), nil
		},
	}

	// Enable JSON output mode for this test
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	output := capturingOutput(func() {
		err := runEntityAnalyticsWithSvc(context.Background(), "bug", mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// JSON output should include bug-specific fields
	if !strings.Contains(output, `"total_bugs"`) {
		t.Errorf("expected 'total_bugs' in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"bugs_by_status"`) {
		t.Errorf("expected 'bugs_by_status' in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"resolved_count"`) {
		t.Errorf("expected 'resolved_count' in JSON output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// TC-F07-014: --type=bug excludes change-card data
// ---------------------------------------------------------------------------

func TestRunEntityAnalytics_Bug_DoesNotCallChangeCard(t *testing.T) {
	changeCardCalled := false
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return bugAnalyticsFixture(), nil
		},
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			changeCardCalled = true
			return changeCardAnalyticsFixture(), nil
		},
	}

	_ = runEntityAnalyticsWithSvc(context.Background(), "bug", mock)

	if changeCardCalled {
		t.Error("expected GetChangeCardAnalytics NOT to be called when --type=bug")
	}
}

// ---------------------------------------------------------------------------
// TC-F07-016: Change-card analytics human-readable output contains all expected metrics
// ---------------------------------------------------------------------------

func TestPrintChangeCardAnalytics_AllMetrics(t *testing.T) {
	result := changeCardAnalyticsFixture()

	output := capturingOutput(func() {
		printChangeCardAnalytics(result)
	})

	if !strings.Contains(output, "Change") {
		t.Errorf("expected 'Change' in header, got:\n%s", output)
	}
	if !strings.Contains(output, "8") {
		t.Errorf("expected total change card count 8, got:\n%s", output)
	}
	// Approval rate 80% should appear somewhere
	if !strings.Contains(output, "80") {
		t.Errorf("expected approval rate 80, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// TC-F07-019: Change-card analytics JSON output matches contract
// ---------------------------------------------------------------------------

func TestRunEntityAnalytics_Change_JSON(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			return changeCardAnalyticsFixture(), nil
		},
	}

	// Enable JSON output mode for this test
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	output := capturingOutput(func() {
		err := runEntityAnalyticsWithSvc(context.Background(), "change", mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, `"total_change_cards"`) {
		t.Errorf("expected 'total_change_cards' in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"approval_rate"`) {
		t.Errorf("expected 'approval_rate' in JSON output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// TC-F07-020: Combined analytics (no --type) includes both sections
// ---------------------------------------------------------------------------

func TestRunCombinedAnalytics_IncludesBoth(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return bugAnalyticsFixture(), nil
		},
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			return changeCardAnalyticsFixture(), nil
		},
	}

	output := capturingOutput(func() {
		err := runCombinedAnalyticsWithSvc(context.Background(), mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Should contain both bug and change card sections
	if !strings.Contains(output, "Bug") {
		t.Errorf("expected bug section in combined output, got:\n%s", output)
	}
	if !strings.Contains(output, "Change") {
		t.Errorf("expected change-card section in combined output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// TC-F07-021: Combined analytics omits absent sections (graceful degradation)
// ---------------------------------------------------------------------------

func TestRunCombinedAnalytics_OmitsAbsentSection(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return bugAnalyticsFixture(), nil
		},
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			// Simulate change-card repo not configured
			return nil, fmt.Errorf("change-card analytics not available: repository not configured")
		},
	}

	var runErr error
	output := capturingOutput(func() {
		runErr = runCombinedAnalyticsWithSvc(context.Background(), mock)
	})

	// Should not return an error (graceful degradation)
	if runErr != nil {
		t.Errorf("expected no error on graceful degradation, got: %v", runErr)
	}
	// Bug section should still appear
	if !strings.Contains(output, "Bug") {
		t.Errorf("expected bug section even when change-card unavailable, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Test nil safety: printBugAnalytics with nil avg resolution time
// ---------------------------------------------------------------------------

func TestPrintBugAnalytics_NilAvgResolutionTime(t *testing.T) {
	result := &services.BugAnalyticsResult{
		TotalBugs:             3,
		BugsByStatus:          map[string]int{"open": 3},
		BugsBySeverity:        map[string]int{"low": 3},
		ResolvedCount:         0,
		AvgResolutionTimeSecs: nil, // no resolved bugs
	}

	// Should not panic
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		printBugAnalytics(result)
	}()

	if panicked {
		t.Error("printBugAnalytics panicked on nil AvgResolutionTimeSecs")
	}
}

// ---------------------------------------------------------------------------
// Test nil safety: printChangeCardAnalytics with nil approval rate
// ---------------------------------------------------------------------------

func TestPrintChangeCardAnalytics_NilApprovalRate(t *testing.T) {
	result := &services.ChangeCardAnalyticsResult{
		TotalChangeCards:      2,
		ChangeCardsByStatus:   map[string]int{"pending": 2},
		ApprovalRate:          nil, // no decided cards
		DecidedCount:          0,
		CompletedCount:        0,
		AvgCompletionTimeSecs: nil,
	}

	// Should not panic
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		printChangeCardAnalytics(result)
	}()

	if panicked {
		t.Error("printChangeCardAnalytics panicked on nil ApprovalRate")
	}
}

// ---------------------------------------------------------------------------
// Test formatDurationFromSecs helper
// ---------------------------------------------------------------------------

func TestFormatDurationFromSecs(t *testing.T) {
	tests := []struct {
		name     string
		secs     float64
		expected string // substring that must appear in output
	}{
		{"zero seconds", 0, "0s"},
		{"one minute", 60, "1m"},
		{"one hour", 3600, "1h"},
		{"one day", 86400, "1d"},
		{"two days", 172800, "2d"},
		{"mixed hours and minutes", 3660, "1h"},
		{"fractional seconds", 45.5, "45s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDurationFromSecs(tt.secs)
			if !strings.Contains(got, tt.expected) {
				t.Errorf("formatDurationFromSecs(%v) = %q, expected to contain %q", tt.secs, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test runEntityAnalytics error propagation
// ---------------------------------------------------------------------------

func TestRunEntityAnalytics_Bug_ServiceError(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}

	err := runEntityAnalyticsWithSvc(context.Background(), "bug", mock)
	if err == nil {
		t.Error("expected error when service returns error, got nil")
	}
	if !strings.Contains(err.Error(), "database connection failed") {
		t.Errorf("expected original error message in wrapped error, got: %v", err)
	}
}

func TestRunEntityAnalytics_Change_ServiceError(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			return nil, fmt.Errorf("change card repo unavailable")
		},
	}

	err := runEntityAnalyticsWithSvc(context.Background(), "change", mock)
	if err == nil {
		t.Error("expected error when service returns error, got nil")
	}
	if !strings.Contains(err.Error(), "change card repo unavailable") {
		t.Errorf("expected original error message in wrapped error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test invalid entity type
// ---------------------------------------------------------------------------

func TestRunEntityAnalytics_InvalidType(t *testing.T) {
	mock := &mockDashboardAnalyticsService{}

	err := runEntityAnalyticsWithSvc(context.Background(), "unknown", mock)
	if err == nil {
		t.Error("expected error for invalid entity type, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tech-debt analytics tests
// ---------------------------------------------------------------------------

func techDebtAnalyticsFixture() *services.TechDebtAnalyticsResult {
	return &services.TechDebtAnalyticsResult{
		TotalTechDebts: 12,
		TechDebtsByStatus: map[string]int{
			"identified":  5,
			"triaged":     3,
			"in_progress": 2,
			"resolved":    2,
		},
		TechDebtsByCategory: map[string]int{
			"code-quality": 4,
			"architecture": 3,
			"testing":      3,
			"dependency":   2,
		},
	}
}

func TestPrintTechDebtAnalytics_AllMetrics(t *testing.T) {
	result := techDebtAnalyticsFixture()

	output := capturingOutput(func() {
		printTechDebtAnalytics(result)
	})

	if !strings.Contains(output, "Tech Debt Analytics") {
		t.Errorf("expected 'Tech Debt Analytics' header, got:\n%s", output)
	}
	if !strings.Contains(output, "12") {
		t.Errorf("expected total tech debt count 12, got:\n%s", output)
	}
	if !strings.Contains(output, "By Status") {
		t.Errorf("expected 'By Status' section, got:\n%s", output)
	}
	if !strings.Contains(output, "By Category") {
		t.Errorf("expected 'By Category' section, got:\n%s", output)
	}
}

func TestRunEntityAnalytics_TechDebt_JSON(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetTechDebtAnalyticsFunc: func(ctx context.Context) (*services.TechDebtAnalyticsResult, error) {
			return techDebtAnalyticsFixture(), nil
		},
	}

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	output := capturingOutput(func() {
		err := runEntityAnalyticsWithSvc(context.Background(), "tech_debt", mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, `"total_tech_debts"`) {
		t.Errorf("expected 'total_tech_debts' in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"tech_debts_by_status"`) {
		t.Errorf("expected 'tech_debts_by_status' in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"tech_debts_by_category"`) {
		t.Errorf("expected 'tech_debts_by_category' in JSON output, got:\n%s", output)
	}
}

func TestRunEntityAnalytics_TechDebt_ServiceError(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetTechDebtAnalyticsFunc: func(ctx context.Context) (*services.TechDebtAnalyticsResult, error) {
			return nil, fmt.Errorf("tech-debt repo unavailable")
		},
	}

	err := runEntityAnalyticsWithSvc(context.Background(), "tech_debt", mock)
	if err == nil {
		t.Error("expected error when service returns error, got nil")
	}
	if !strings.Contains(err.Error(), "tech-debt repo unavailable") {
		t.Errorf("expected original error message in wrapped error, got: %v", err)
	}
}

func TestRunCombinedAnalytics_IncludesTechDebt(t *testing.T) {
	mock := &mockDashboardAnalyticsService{
		GetBugAnalyticsFunc: func(ctx context.Context) (*services.BugAnalyticsResult, error) {
			return nil, fmt.Errorf("not configured")
		},
		GetChangeCardAnalyticsFunc: func(ctx context.Context) (*services.ChangeCardAnalyticsResult, error) {
			return nil, fmt.Errorf("not configured")
		},
		GetTechDebtAnalyticsFunc: func(ctx context.Context) (*services.TechDebtAnalyticsResult, error) {
			return techDebtAnalyticsFixture(), nil
		},
	}

	output := capturingOutput(func() {
		err := runCombinedAnalyticsWithSvc(context.Background(), mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Tech Debt") {
		t.Errorf("expected tech-debt section in combined output, got:\n%s", output)
	}
}
