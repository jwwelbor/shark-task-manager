package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// BugSummaryRepository defines the aggregate query methods that DashboardAnalyticsService
// requires from the bug repository. The concrete implementation is *repository.BugRepository,
// created by E18-F02.
//
// Interfaces are defined at the point of use (consumer side) per the project's service design
// principles. This allows the service tests to use lightweight function-field mocks without
// depending on the concrete repository package.
type BugSummaryRepository interface {
	// GetStatusSummary returns aggregate status and severity counts for all bugs.
	GetStatusSummary(ctx context.Context) (*repository.BugStatusSummary, error)
	// GetResolutionStats returns resolution count and average resolution time for bugs.
	GetResolutionStats(ctx context.Context) (*repository.BugResolutionStats, error)
	// GetFeatureBugSummary returns aggregate bug counts linked to a specific feature key.
	GetFeatureBugSummary(ctx context.Context, featureKey string) (*repository.BugFeatureSummary, error)
}

// ChangeCardSummaryRepository defines the aggregate query methods that DashboardAnalyticsService
// requires from the change-card repository. The concrete implementation is
// *repository.ChangeCardRepository, created by E18-F03.
type ChangeCardSummaryRepository interface {
	// GetStatusSummary returns aggregate status counts for all change-cards.
	GetStatusSummary(ctx context.Context) (*repository.ChangeCardStatusSummary, error)
	// GetThroughputStats returns throughput metrics (approval rate, completion time) for change-cards.
	GetThroughputStats(ctx context.Context) (*repository.ChangeCardThroughputStats, error)
}

// DashboardAnalyticsService provides bug and change-card analytics data for the
// `shark analytics` command. Per ADR-F07-002, this is a focused sub-service separate
// from EpicAnalyticsService to maintain single responsibility.
//
// Both repository dependencies are optional (can be nil). When nil, the corresponding
// analytics method returns a descriptive error rather than panicking. This allows the
// service to be used during phased rollout before all entity tables exist.
type DashboardAnalyticsService struct {
	bugRepo        BugSummaryRepository        // optional, nil-safe
	changeCardRepo ChangeCardSummaryRepository // optional, nil-safe
}

// NewDashboardAnalyticsService creates a DashboardAnalyticsService with the given
// optional repository dependencies. Either or both may be nil; the service degrades
// gracefully and returns descriptive errors when a nil repository is called.
func NewDashboardAnalyticsService(
	bugRepo BugSummaryRepository,
	changeCardRepo ChangeCardSummaryRepository,
) *DashboardAnalyticsService {
	return &DashboardAnalyticsService{
		bugRepo:        bugRepo,
		changeCardRepo: changeCardRepo,
	}
}

// GetBugAnalytics assembles bug analytics from the bug repository's aggregate query methods.
//
// Returns an error when:
//   - bugRepo is nil (bug analytics not configured)
//   - GetStatusSummary fails (error propagated with business context)
//   - GetResolutionStats fails (error propagated with business context)
func (s *DashboardAnalyticsService) GetBugAnalytics(ctx context.Context) (*BugAnalyticsResult, error) {
	if s.bugRepo == nil {
		return nil, fmt.Errorf("bug analytics not available: bug repository not configured")
	}

	summary, err := s.bugRepo.GetStatusSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug status summary: %w", err)
	}

	resolution, err := s.bugRepo.GetResolutionStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug resolution stats: %w", err)
	}

	return &BugAnalyticsResult{
		TotalBugs:             summary.Total,
		BugsByStatus:          summary.ByStatus,
		BugsBySeverity:        summary.BySeverity,
		ResolvedCount:         resolution.ResolvedCount,
		AvgResolutionTimeSecs: resolution.AvgResolutionSecs,
	}, nil
}

// GetChangeCardAnalytics assembles change-card analytics from the change-card repository's
// aggregate query methods.
//
// Returns an error when:
//   - changeCardRepo is nil (change-card analytics not configured)
//   - GetStatusSummary fails (error propagated with business context)
//   - GetThroughputStats fails (error propagated with business context)
func (s *DashboardAnalyticsService) GetChangeCardAnalytics(ctx context.Context) (*ChangeCardAnalyticsResult, error) {
	if s.changeCardRepo == nil {
		return nil, fmt.Errorf("change-card analytics not available: repository not configured")
	}

	summary, err := s.changeCardRepo.GetStatusSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card status summary: %w", err)
	}

	throughput, err := s.changeCardRepo.GetThroughputStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card throughput stats: %w", err)
	}

	return &ChangeCardAnalyticsResult{
		TotalChangeCards:      summary.Total,
		ChangeCardsByStatus:   summary.ByStatus,
		ApprovalRate:          throughput.ApprovalRate,
		DecidedCount:          throughput.DecidedCount,
		CompletedCount:        throughput.CompletedCount,
		AvgCompletionTimeSecs: throughput.AvgCompletionSecs,
	}, nil
}
