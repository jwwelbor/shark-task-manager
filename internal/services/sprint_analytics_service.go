package services

import (
	"context"
	"fmt"

	sprint "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
)

// SprintAnalyticsService orchestrates sprint analytics queries (E19-F04).
// It is read-only and has no workflow validation dependencies; it is kept
// separate from SprintService to maintain single responsibility
// (Decision 3 in spec §5).
type SprintAnalyticsService struct {
	analyticsRepo SprintAnalyticsRepository // required
	sprintRepo    SprintRepository          // required for burndown / summary (may be nil for velocity-only use)
}

// NewSprintAnalyticsService creates a SprintAnalyticsService.
// analyticsRepo is required. sprintRepo is required for GetBurndown and
// GetSummary but may be nil if only GetVelocity is called.
func NewSprintAnalyticsService(
	analyticsRepo SprintAnalyticsRepository,
	sprintRepo SprintRepository,
) *SprintAnalyticsService {
	return &SprintAnalyticsService{
		analyticsRepo: analyticsRepo,
		sprintRepo:    sprintRepo,
	}
}

// GetVelocity returns velocity data for the last n completed sprints.
//
// Validation: n must be in the range [1, 100]. Values outside this range
// return an error without calling the repository.
//
// Trailing average is the mean of CompletedSize values across all returned
// rows, including rows with zero CompletedSize (zero-velocity sprints
// contribute 0 to the numerator but are included in the denominator).
// This matches AC-V-4 and TC-V-07.
//
// InsufficientData is set to true when the repository returns fewer than 3
// rows; this is informational only and does not cause an error (AC-V-5,
// TC-V-09).
func (s *SprintAnalyticsService) GetVelocity(ctx context.Context, n int) (*VelocityResult, error) {
	// Validate n range (AC-V-2, TC-V-04)
	if n < 1 || n > 100 {
		return nil, fmt.Errorf("sprints must be between 1 and 100, got %d", n)
	}

	rows, err := s.analyticsRepo.GetVelocityData(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("failed to get velocity data: %w", err)
	}

	// Build per-sprint breakdown
	sprints := make([]VelocitySprint, len(rows))
	var totalSize int
	for i, row := range rows {
		sprints[i] = VelocitySprint{
			Key:              row.SprintKey,
			Name:             row.SprintName,
			CompletedSize:    row.CompletedSize,
			UnsizedCompleted: row.UnsizedCompleted,
		}
		totalSize += row.CompletedSize
	}

	// Compute trailing average.
	// Denominator is len(rows) — zero-velocity sprints are included (AC-V-4).
	// When len(rows) == 0 the average is 0.0; no divide-by-zero (TC-V-08).
	var trailingAverage float64
	if len(rows) > 0 {
		trailingAverage = float64(totalSize) / float64(len(rows))
	}

	result := &VelocityResult{
		Sprints:          sprints,
		TrailingAverage:  trailingAverage,
		SprintCount:      len(rows),
		InsufficientData: len(rows) < 3, // AC-V-5, TC-V-09
	}

	return result, nil
}

// GetBurndown returns burndown data for the sprint identified by sprintKey.
// If sprintKey is empty, the current active sprint is used.
// This is a stub for E19-F04 burndown tasks (T-E19-F04-005 and later).
func (s *SprintAnalyticsService) GetBurndown(ctx context.Context, sprintKey string) (*BurndownResult, error) {
	return nil, fmt.Errorf("GetBurndown not yet implemented (T-E19-F04-005)")
}

// GetSummary returns a sprint summary report for the given sprint key.
// When detailed is true, additional fields are populated (cycle time, size
// bands, carryover list).
// This is a stub for E19-F04 summary tasks (T-E19-F04-006 and later).
func (s *SprintAnalyticsService) GetSummary(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
	return nil, fmt.Errorf("GetSummary not yet implemented (T-E19-F04-006)")
}

// Ensure the sprint.VelocityRow type is used (prevents import cycle issues).
var _ = sprint.VelocityRow{}
