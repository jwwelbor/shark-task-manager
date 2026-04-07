package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// MockBugSummaryRepository is a test double for BugSummaryRepository.
// Uses function-field pattern to allow per-test customization.
type MockBugSummaryRepository struct {
	GetStatusSummaryFunc     func(ctx context.Context) (*repository.BugStatusSummary, error)
	GetResolutionStatsFunc   func(ctx context.Context) (*repository.BugResolutionStats, error)
	GetFeatureBugSummaryFunc func(ctx context.Context, featureKey string) (*repository.BugFeatureSummary, error)
}

func (m *MockBugSummaryRepository) GetStatusSummary(ctx context.Context) (*repository.BugStatusSummary, error) {
	if m.GetStatusSummaryFunc != nil {
		return m.GetStatusSummaryFunc(ctx)
	}
	return nil, errors.New("GetStatusSummary not implemented in mock")
}

func (m *MockBugSummaryRepository) GetResolutionStats(ctx context.Context) (*repository.BugResolutionStats, error) {
	if m.GetResolutionStatsFunc != nil {
		return m.GetResolutionStatsFunc(ctx)
	}
	return nil, errors.New("GetResolutionStats not implemented in mock")
}

func (m *MockBugSummaryRepository) GetFeatureBugSummary(ctx context.Context, featureKey string) (*repository.BugFeatureSummary, error) {
	if m.GetFeatureBugSummaryFunc != nil {
		return m.GetFeatureBugSummaryFunc(ctx, featureKey)
	}
	return nil, errors.New("GetFeatureBugSummary not implemented in mock")
}

// MockChangeCardSummaryRepository is a test double for ChangeCardSummaryRepository.
type MockChangeCardSummaryRepository struct {
	GetStatusSummaryFunc   func(ctx context.Context) (*repository.ChangeCardStatusSummary, error)
	GetThroughputStatsFunc func(ctx context.Context) (*repository.ChangeCardThroughputStats, error)
}

func (m *MockChangeCardSummaryRepository) GetStatusSummary(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
	if m.GetStatusSummaryFunc != nil {
		return m.GetStatusSummaryFunc(ctx)
	}
	return nil, errors.New("GetStatusSummary not implemented in mock")
}

func (m *MockChangeCardSummaryRepository) GetThroughputStats(ctx context.Context) (*repository.ChangeCardThroughputStats, error) {
	if m.GetThroughputStatsFunc != nil {
		return m.GetThroughputStatsFunc(ctx)
	}
	return nil, errors.New("GetThroughputStats not implemented in mock")
}

// ---------------------------------------------------------------------------
// GetBugAnalytics tests
// ---------------------------------------------------------------------------

// TC-F07-011: Bug analytics shows all metrics (happy path).
func TestDashboardAnalyticsService_GetBugAnalytics_HappyPath(t *testing.T) {
	avgSecs := 14400.0
	mockBug := &MockBugSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.BugStatusSummary, error) {
			return &repository.BugStatusSummary{
				Total: 10,
				ByStatus: map[string]int{
					"reported":        3,
					"triaged":         2,
					"in_fix":          1,
					"in_verification": 1,
					"resolved":        2,
					"wont_fix":        1,
				},
				BySeverity: map[string]int{
					"critical": 2,
					"high":     3,
					"medium":   4,
					"low":      1,
				},
				OpenBySeverity: map[string]int{
					"critical": 2,
					"high":     3,
					"medium":   2,
				},
			}, nil
		},
		GetResolutionStatsFunc: func(ctx context.Context) (*repository.BugResolutionStats, error) {
			return &repository.BugResolutionStats{
				ResolvedCount:     3,
				AvgResolutionSecs: &avgSecs,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(mockBug, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetBugAnalytics() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("GetBugAnalytics() returned nil result")
	}
	if result.TotalBugs != 10 {
		t.Errorf("TotalBugs = %d, want 10", result.TotalBugs)
	}
	if result.ResolvedCount != 3 {
		t.Errorf("ResolvedCount = %d, want 3", result.ResolvedCount)
	}
	if result.AvgResolutionTimeSecs == nil {
		t.Fatal("AvgResolutionTimeSecs should not be nil when resolved bugs exist")
	}
	if *result.AvgResolutionTimeSecs != avgSecs {
		t.Errorf("AvgResolutionTimeSecs = %f, want %f", *result.AvgResolutionTimeSecs, avgSecs)
	}
	if len(result.BugsByStatus) == 0 {
		t.Error("BugsByStatus should not be empty")
	}
	if len(result.BugsBySeverity) == 0 {
		t.Error("BugsBySeverity should not be empty")
	}
}

// TC-F07-012 / Section 3.1: GetBugAnalytics with nil bug repository returns error.
func TestDashboardAnalyticsService_GetBugAnalytics_NilRepo(t *testing.T) {
	svc := NewDashboardAnalyticsService(nil, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetBugAnalytics() expected error when bugRepo is nil, got nil")
	}
	if result != nil {
		t.Error("GetBugAnalytics() expected nil result when bugRepo is nil")
	}
	// Verify the error message is descriptive (not a panic)
	if err.Error() == "" {
		t.Error("Error message should be non-empty")
	}
}

// Section 3.1: Error propagated when GetStatusSummary returns error.
func TestDashboardAnalyticsService_GetBugAnalytics_StatusSummaryError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	mockBug := &MockBugSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.BugStatusSummary, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(mockBug, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetBugAnalytics() expected error when GetStatusSummary fails")
	}
	if result != nil {
		t.Error("GetBugAnalytics() expected nil result on error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// Section 3.1: Error propagated when GetResolutionStats returns error.
func TestDashboardAnalyticsService_GetBugAnalytics_ResolutionStatsError(t *testing.T) {
	expectedErr := errors.New("resolution stats unavailable")
	mockBug := &MockBugSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.BugStatusSummary, error) {
			return &repository.BugStatusSummary{
				Total:    5,
				ByStatus: map[string]int{"reported": 5},
			}, nil
		},
		GetResolutionStatsFunc: func(ctx context.Context) (*repository.BugResolutionStats, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(mockBug, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetBugAnalytics() expected error when GetResolutionStats fails")
	}
	if result != nil {
		t.Error("GetBugAnalytics() expected nil result on resolution stats error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// TC-F07-012: Bug analytics with zero resolved bugs -- AvgResolutionTimeSecs is nil.
func TestDashboardAnalyticsService_GetBugAnalytics_ZeroResolved(t *testing.T) {
	mockBug := &MockBugSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.BugStatusSummary, error) {
			return &repository.BugStatusSummary{
				Total:    3,
				ByStatus: map[string]int{"reported": 3},
			}, nil
		},
		GetResolutionStatsFunc: func(ctx context.Context) (*repository.BugResolutionStats, error) {
			return &repository.BugResolutionStats{
				ResolvedCount:     0,
				AvgResolutionSecs: nil,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(mockBug, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetBugAnalytics() unexpected error: %v", err)
	}
	if result.ResolvedCount != 0 {
		t.Errorf("ResolvedCount = %d, want 0", result.ResolvedCount)
	}
	if result.AvgResolutionTimeSecs != nil {
		t.Errorf("AvgResolutionTimeSecs should be nil when no resolved bugs, got %f", *result.AvgResolutionTimeSecs)
	}
}

// Section 3.1: Zero bugs returns result with zero total and empty maps.
func TestDashboardAnalyticsService_GetBugAnalytics_ZeroBugs(t *testing.T) {
	mockBug := &MockBugSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.BugStatusSummary, error) {
			return &repository.BugStatusSummary{
				Total:    0,
				ByStatus: map[string]int{},
			}, nil
		},
		GetResolutionStatsFunc: func(ctx context.Context) (*repository.BugResolutionStats, error) {
			return &repository.BugResolutionStats{
				ResolvedCount:     0,
				AvgResolutionSecs: nil,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(mockBug, nil, nil)
	result, err := svc.GetBugAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetBugAnalytics() unexpected error: %v", err)
	}
	if result.TotalBugs != 0 {
		t.Errorf("TotalBugs = %d, want 0", result.TotalBugs)
	}
}

// ---------------------------------------------------------------------------
// GetChangeCardAnalytics tests
// ---------------------------------------------------------------------------

// TC-F07-016: Change-card analytics shows all metrics (happy path).
func TestDashboardAnalyticsService_GetChangeCardAnalytics_HappyPath(t *testing.T) {
	approvalRate := 0.833
	avgCompletion := 259200.0

	mockCC := &MockChangeCardSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
			return &repository.ChangeCardStatusSummary{
				Total: 8,
				ByStatus: map[string]int{
					"proposed":    2,
					"approved":    1,
					"in_progress": 2,
					"completed":   2,
					"declined":    1,
				},
			}, nil
		},
		GetThroughputStatsFunc: func(ctx context.Context) (*repository.ChangeCardThroughputStats, error) {
			return &repository.ChangeCardThroughputStats{
				DecidedCount:      6,
				ApprovedCount:     5,
				DeclinedCount:     1,
				ApprovalRate:      &approvalRate,
				CompletedCount:    2,
				AvgCompletionSecs: &avgCompletion,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(nil, mockCC, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetChangeCardAnalytics() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("GetChangeCardAnalytics() returned nil result")
	}
	if result.TotalChangeCards != 8 {
		t.Errorf("TotalChangeCards = %d, want 8", result.TotalChangeCards)
	}
	if result.DecidedCount != 6 {
		t.Errorf("DecidedCount = %d, want 6", result.DecidedCount)
	}
	if result.CompletedCount != 2 {
		t.Errorf("CompletedCount = %d, want 2", result.CompletedCount)
	}
	if result.ApprovalRate == nil {
		t.Fatal("ApprovalRate should not be nil when decided cards exist")
	}
	if *result.ApprovalRate != approvalRate {
		t.Errorf("ApprovalRate = %f, want %f", *result.ApprovalRate, approvalRate)
	}
	if result.AvgCompletionTimeSecs == nil {
		t.Fatal("AvgCompletionTimeSecs should not be nil when completed cards exist")
	}
	if *result.AvgCompletionTimeSecs != avgCompletion {
		t.Errorf("AvgCompletionTimeSecs = %f, want %f", *result.AvgCompletionTimeSecs, avgCompletion)
	}
	if len(result.ChangeCardsByStatus) == 0 {
		t.Error("ChangeCardsByStatus should not be empty")
	}
}

// Section 3.1: GetChangeCardAnalytics with nil repository returns error.
func TestDashboardAnalyticsService_GetChangeCardAnalytics_NilRepo(t *testing.T) {
	svc := NewDashboardAnalyticsService(nil, nil, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetChangeCardAnalytics() expected error when changeCardRepo is nil, got nil")
	}
	if result != nil {
		t.Error("GetChangeCardAnalytics() expected nil result when repo is nil")
	}
}

// TC-F07-018: Change-card analytics with zero decided cards -- ApprovalRate is nil.
func TestDashboardAnalyticsService_GetChangeCardAnalytics_ZeroDecided(t *testing.T) {
	mockCC := &MockChangeCardSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
			return &repository.ChangeCardStatusSummary{
				Total:    2,
				ByStatus: map[string]int{"proposed": 2},
			}, nil
		},
		GetThroughputStatsFunc: func(ctx context.Context) (*repository.ChangeCardThroughputStats, error) {
			return &repository.ChangeCardThroughputStats{
				DecidedCount:      0,
				ApprovedCount:     0,
				DeclinedCount:     0,
				ApprovalRate:      nil,
				CompletedCount:    0,
				AvgCompletionSecs: nil,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(nil, mockCC, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetChangeCardAnalytics() unexpected error: %v", err)
	}
	if result.ApprovalRate != nil {
		t.Errorf("ApprovalRate should be nil when no decided cards, got %f", *result.ApprovalRate)
	}
	if result.DecidedCount != 0 {
		t.Errorf("DecidedCount = %d, want 0", result.DecidedCount)
	}
}

// TC-F07-017: Change-card analytics with zero completed cards -- AvgCompletionTimeSecs is nil.
func TestDashboardAnalyticsService_GetChangeCardAnalytics_ZeroCompleted(t *testing.T) {
	approvalRate := 1.0
	mockCC := &MockChangeCardSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
			return &repository.ChangeCardStatusSummary{
				Total:    2,
				ByStatus: map[string]int{"proposed": 1, "approved": 1},
			}, nil
		},
		GetThroughputStatsFunc: func(ctx context.Context) (*repository.ChangeCardThroughputStats, error) {
			return &repository.ChangeCardThroughputStats{
				DecidedCount:      1,
				ApprovedCount:     1,
				DeclinedCount:     0,
				ApprovalRate:      &approvalRate,
				CompletedCount:    0,
				AvgCompletionSecs: nil,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(nil, mockCC, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetChangeCardAnalytics() unexpected error: %v", err)
	}
	if result.AvgCompletionTimeSecs != nil {
		t.Errorf("AvgCompletionTimeSecs should be nil when no completed cards, got %f", *result.AvgCompletionTimeSecs)
	}
	if result.CompletedCount != 0 {
		t.Errorf("CompletedCount = %d, want 0", result.CompletedCount)
	}
}

// Section 3.1: Error propagated when GetStatusSummary returns error for change cards.
func TestDashboardAnalyticsService_GetChangeCardAnalytics_StatusSummaryError(t *testing.T) {
	expectedErr := errors.New("change card db error")
	mockCC := &MockChangeCardSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(nil, mockCC, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetChangeCardAnalytics() expected error when GetStatusSummary fails")
	}
	if result != nil {
		t.Error("GetChangeCardAnalytics() expected nil result on error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// Section 3.1: Error propagated when GetThroughputStats returns error.
func TestDashboardAnalyticsService_GetChangeCardAnalytics_ThroughputStatsError(t *testing.T) {
	expectedErr := errors.New("throughput stats unavailable")
	mockCC := &MockChangeCardSummaryRepository{
		GetStatusSummaryFunc: func(ctx context.Context) (*repository.ChangeCardStatusSummary, error) {
			return &repository.ChangeCardStatusSummary{
				Total:    3,
				ByStatus: map[string]int{"proposed": 3},
			}, nil
		},
		GetThroughputStatsFunc: func(ctx context.Context) (*repository.ChangeCardThroughputStats, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(nil, mockCC, nil)
	result, err := svc.GetChangeCardAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetChangeCardAnalytics() expected error when GetThroughputStats fails")
	}
	if result != nil {
		t.Error("GetChangeCardAnalytics() expected nil result on throughput stats error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// JSON contract tests (TC-F07-031, TC-F07-032)
// ---------------------------------------------------------------------------

// TC-F07-031: Bug analytics JSON contract -- all required keys present with correct types.
func TestBugAnalyticsResult_JSONContract(t *testing.T) {
	avgSecs := 14400.0
	result := &BugAnalyticsResult{
		TotalBugs: 10,
		BugsByStatus: map[string]int{
			"reported": 3,
			"resolved": 2,
		},
		BugsBySeverity: map[string]int{
			"critical": 2,
			"high":     3,
		},
		ResolvedCount:         3,
		AvgResolutionTimeSecs: &avgSecs,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"total_bugs", "bugs_by_status", "bugs_by_severity", "resolved_count", "avg_resolution_time_seconds"}
	for _, key := range requiredKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON missing required key: %s", key)
		}
	}

	// avg_resolution_time_seconds should be a float when set
	if avgVal, ok := parsed["avg_resolution_time_seconds"]; !ok || avgVal == nil {
		t.Error("avg_resolution_time_seconds should be non-nil when set")
	}

	// Verify total_bugs is numeric
	if _, ok := parsed["total_bugs"].(float64); !ok {
		t.Errorf("total_bugs should be numeric, got %T", parsed["total_bugs"])
	}
}

// TC-F07-031: Bug analytics JSON contract -- avg_resolution_time_seconds is null when nil.
func TestBugAnalyticsResult_JSONContract_NilAvg(t *testing.T) {
	result := &BugAnalyticsResult{
		TotalBugs:             3,
		BugsByStatus:          map[string]int{"reported": 3},
		BugsBySeverity:        map[string]int{},
		ResolvedCount:         0,
		AvgResolutionTimeSecs: nil,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// avg_resolution_time_seconds should be present but null when nil pointer
	avgVal, exists := parsed["avg_resolution_time_seconds"]
	if !exists {
		t.Error("avg_resolution_time_seconds key should be present in JSON (as null)")
	}
	if avgVal != nil {
		t.Errorf("avg_resolution_time_seconds should be null when nil, got %v", avgVal)
	}
}

// TC-F07-032: Change-card analytics JSON contract -- all required keys present.
func TestChangeCardAnalyticsResult_JSONContract(t *testing.T) {
	approvalRate := 0.833
	avgCompletion := 259200.0
	result := &ChangeCardAnalyticsResult{
		TotalChangeCards: 8,
		ChangeCardsByStatus: map[string]int{
			"proposed":  2,
			"completed": 2,
		},
		ApprovalRate:          &approvalRate,
		DecidedCount:          6,
		CompletedCount:        2,
		AvgCompletionTimeSecs: &avgCompletion,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{
		"total_change_cards",
		"change_cards_by_status",
		"approval_rate",
		"decided_count",
		"completed_count",
		"avg_completion_time_seconds",
	}
	for _, key := range requiredKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON missing required key: %s", key)
		}
	}

	// approval_rate should be a float (0.0-1.0 range)
	if rateVal, ok := parsed["approval_rate"].(float64); !ok {
		t.Errorf("approval_rate should be numeric float, got %T", parsed["approval_rate"])
	} else if rateVal < 0.0 || rateVal > 1.0 {
		t.Errorf("approval_rate should be in [0.0, 1.0] range, got %f", rateVal)
	}
}

// TC-F07-032: Change-card analytics JSON -- approval_rate is null when no decided cards.
func TestChangeCardAnalyticsResult_JSONContract_NilApprovalRate(t *testing.T) {
	result := &ChangeCardAnalyticsResult{
		TotalChangeCards:      2,
		ChangeCardsByStatus:   map[string]int{"proposed": 2},
		ApprovalRate:          nil,
		DecidedCount:          0,
		CompletedCount:        0,
		AvgCompletionTimeSecs: nil,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	rateVal, exists := parsed["approval_rate"]
	if !exists {
		t.Error("approval_rate key should be present in JSON (as null)")
	}
	if rateVal != nil {
		t.Errorf("approval_rate should be null when nil, got %v", rateVal)
	}
}

// TC-F07-020: DashboardAnalyticsResult combines both for unfiltered analytics.
func TestDashboardAnalyticsResult_JSONContract_CombinedOutput(t *testing.T) {
	avgSecs := 14400.0
	approvalRate := 0.5
	avgCompletion := 3600.0

	combined := &DashboardAnalyticsResult{
		Bugs: &BugAnalyticsResult{
			TotalBugs:             5,
			BugsByStatus:          map[string]int{"reported": 5},
			BugsBySeverity:        map[string]int{"high": 3},
			ResolvedCount:         1,
			AvgResolutionTimeSecs: &avgSecs,
		},
		ChangeCards: &ChangeCardAnalyticsResult{
			TotalChangeCards:      3,
			ChangeCardsByStatus:   map[string]int{"proposed": 3},
			ApprovalRate:          &approvalRate,
			DecidedCount:          2,
			CompletedCount:        1,
			AvgCompletionTimeSecs: &avgCompletion,
		},
	}

	data, err := json.Marshal(combined)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := parsed["bugs"]; !ok {
		t.Error("combined JSON should contain 'bugs' key")
	}
	if _, ok := parsed["change_cards"]; !ok {
		t.Error("combined JSON should contain 'change_cards' key")
	}
}

// TC-F07-021: DashboardAnalyticsResult omits nil sections with omitempty.
func TestDashboardAnalyticsResult_JSONContract_OmitEmpty(t *testing.T) {
	avgSecs := 14400.0
	combined := &DashboardAnalyticsResult{
		Bugs: &BugAnalyticsResult{
			TotalBugs:             5,
			BugsByStatus:          map[string]int{"reported": 5},
			BugsBySeverity:        map[string]int{},
			ResolvedCount:         1,
			AvgResolutionTimeSecs: &avgSecs,
		},
		ChangeCards: nil, // omitted
	}

	data, err := json.Marshal(combined)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := parsed["bugs"]; !ok {
		t.Error("combined JSON should contain 'bugs' key when bugs are set")
	}
	if _, ok := parsed["change_cards"]; ok {
		t.Error("combined JSON should NOT contain 'change_cards' key when nil (omitempty)")
	}
}

// ---------------------------------------------------------------------------
// MockTechDebtSummaryRepository
// ---------------------------------------------------------------------------

// MockTechDebtSummaryRepository is a test double for TechDebtSummaryRepository.
type MockTechDebtSummaryRepository struct {
	CountByStatusFunc   func(ctx context.Context) (map[string]int, error)
	CountByCategoryFunc func(ctx context.Context) (map[string]int, error)
}

func (m *MockTechDebtSummaryRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return nil, errors.New("CountByStatus not implemented in mock")
}

func (m *MockTechDebtSummaryRepository) CountByCategory(ctx context.Context) (map[string]int, error) {
	if m.CountByCategoryFunc != nil {
		return m.CountByCategoryFunc(ctx)
	}
	return nil, errors.New("CountByCategory not implemented in mock")
}

// ---------------------------------------------------------------------------
// GetTechDebtAnalytics tests
// ---------------------------------------------------------------------------

// Happy path: tech-debt analytics returns total, by-status, and by-category counts.
func TestDashboardAnalyticsService_GetTechDebtAnalytics_HappyPath(t *testing.T) {
	mockTD := &MockTechDebtSummaryRepository{
		CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{
				"identified":  3,
				"triaged":     2,
				"in_progress": 1,
				"resolved":    4,
			}, nil
		},
		CountByCategoryFunc: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{
				"code-quality":  4,
				"architecture":  3,
				"testing":       2,
				"documentation": 1,
			}, nil
		},
	}

	svc := NewDashboardAnalyticsService(nil, nil, mockTD)
	result, err := svc.GetTechDebtAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetTechDebtAnalytics() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("GetTechDebtAnalytics() returned nil result")
	}
	if result.TotalTechDebts != 10 {
		t.Errorf("TotalTechDebts = %d, want 10", result.TotalTechDebts)
	}
	if len(result.TechDebtsByStatus) != 4 {
		t.Errorf("TechDebtsByStatus count = %d, want 4", len(result.TechDebtsByStatus))
	}
	if len(result.TechDebtsByCategory) != 4 {
		t.Errorf("TechDebtsByCategory count = %d, want 4", len(result.TechDebtsByCategory))
	}
}

// Nil repo returns descriptive error.
func TestDashboardAnalyticsService_GetTechDebtAnalytics_NilRepo(t *testing.T) {
	svc := NewDashboardAnalyticsService(nil, nil, nil)
	result, err := svc.GetTechDebtAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetTechDebtAnalytics() expected error when techDebtRepo is nil, got nil")
	}
	if result != nil {
		t.Error("GetTechDebtAnalytics() expected nil result when repo is nil")
	}
}

// Error propagated when CountByStatus fails.
func TestDashboardAnalyticsService_GetTechDebtAnalytics_StatusCountError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	mockTD := &MockTechDebtSummaryRepository{
		CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(nil, nil, mockTD)
	result, err := svc.GetTechDebtAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetTechDebtAnalytics() expected error when CountByStatus fails")
	}
	if result != nil {
		t.Error("GetTechDebtAnalytics() expected nil result on error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// Error propagated when CountByCategory fails.
func TestDashboardAnalyticsService_GetTechDebtAnalytics_CategoryCountError(t *testing.T) {
	expectedErr := errors.New("category query failed")
	mockTD := &MockTechDebtSummaryRepository{
		CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{"identified": 3}, nil
		},
		CountByCategoryFunc: func(ctx context.Context) (map[string]int, error) {
			return nil, expectedErr
		},
	}

	svc := NewDashboardAnalyticsService(nil, nil, mockTD)
	result, err := svc.GetTechDebtAnalytics(context.Background())

	if err == nil {
		t.Fatal("GetTechDebtAnalytics() expected error when CountByCategory fails")
	}
	if result != nil {
		t.Error("GetTechDebtAnalytics() expected nil result on error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error chain should contain original error; got: %v", err)
	}
}

// Zero tech-debts returns result with zero total and empty maps.
func TestDashboardAnalyticsService_GetTechDebtAnalytics_ZeroItems(t *testing.T) {
	mockTD := &MockTechDebtSummaryRepository{
		CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{}, nil
		},
		CountByCategoryFunc: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{}, nil
		},
	}

	svc := NewDashboardAnalyticsService(nil, nil, mockTD)
	result, err := svc.GetTechDebtAnalytics(context.Background())

	if err != nil {
		t.Fatalf("GetTechDebtAnalytics() unexpected error: %v", err)
	}
	if result.TotalTechDebts != 0 {
		t.Errorf("TotalTechDebts = %d, want 0", result.TotalTechDebts)
	}
}

// TechDebtAnalyticsResult JSON contract test.
func TestTechDebtAnalyticsResult_JSONContract(t *testing.T) {
	result := &TechDebtAnalyticsResult{
		TotalTechDebts:      10,
		TechDebtsByStatus:   map[string]int{"identified": 3, "resolved": 4},
		TechDebtsByCategory: map[string]int{"code-quality": 5, "architecture": 3},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"total_tech_debts", "tech_debts_by_status", "tech_debts_by_category"}
	for _, key := range requiredKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON missing required key: %s", key)
		}
	}
}

// DashboardAnalyticsResult includes tech_debts in combined output.
func TestDashboardAnalyticsResult_JSONContract_WithTechDebts(t *testing.T) {
	combined := &DashboardAnalyticsResult{
		TechDebts: &TechDebtAnalyticsResult{
			TotalTechDebts:      5,
			TechDebtsByStatus:   map[string]int{"identified": 5},
			TechDebtsByCategory: map[string]int{"testing": 5},
		},
	}

	data, err := json.Marshal(combined)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := parsed["tech_debts"]; !ok {
		t.Error("combined JSON should contain 'tech_debts' key")
	}
}

// ---------------------------------------------------------------------------
// Interface contract (compile-time check)
// ---------------------------------------------------------------------------

// Verify that concrete repository types satisfy the service-layer interfaces.
// If F02/F03 change method signatures, this will cause a compile error.
var _ BugSummaryRepository = (*MockBugSummaryRepository)(nil)
var _ ChangeCardSummaryRepository = (*MockChangeCardSummaryRepository)(nil)
var _ TechDebtSummaryRepository = (*MockTechDebtSummaryRepository)(nil)
