package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	sprint "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
)

// SprintAnalyticsService orchestrates sprint analytics queries (E19-F04).
// It is read-only and has no workflow validation dependencies; it is kept
// separate from SprintService to maintain single responsibility
// (Decision 3 in spec §5).
type SprintAnalyticsService struct {
	analyticsRepo SprintAnalyticsRepository // required
	sprintRepo    SprintRepository          // required for burndown / summary (may be nil for velocity-only use)
	now           func() time.Time          // injectable time source; defaults to time.Now
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
		now:           time.Now,
	}
}

// newSprintAnalyticsServiceWithClock creates a SprintAnalyticsService with an
// injectable time source. Used in tests to control "today" deterministically.
// See TC-B-12 (nil ActualRemaining for future days).
func newSprintAnalyticsServiceWithClock(
	analyticsRepo SprintAnalyticsRepository,
	sprintRepo SprintRepository,
	now func() time.Time,
) *SprintAnalyticsService {
	return &SprintAnalyticsService{
		analyticsRepo: analyticsRepo,
		sprintRepo:    sprintRepo,
		now:           now,
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
// If sprintKey is empty, the current active sprint is used (AC-B-1, TC-B-01).
//
// The burndown is reconstructed from task_history events via
// analyticsRepo.GetCompletionEvents (Decision 1 in spec §5). Non-task entities
// use point-in-time current status — this limitation is documented in the DTO.
//
// ActualRemaining is nil for calendar days that fall after today's date
// (AC-B-8, TC-B-12). IdealRemaining is always populated. The ideal line
// resets piecewise whenever entities are added or removed mid-sprint
// (AC-B-3, TC-B-06).
func (s *SprintAnalyticsService) GetBurndown(ctx context.Context, sprintKey string) (*BurndownResult, error) {
	// Step 1: Resolve the target sprint (AC-B-1, TC-B-01/02)
	var sp *models.Sprint
	var err error
	if sprintKey == "" {
		// No key provided → look up the active sprint
		filters := &sprint.SprintListFilters{Status: sprintStatusPtr("active")}
		sprints, listErr := s.sprintRepo.List(ctx, filters)
		if listErr != nil {
			return nil, fmt.Errorf("failed to find active sprint: %w", listErr)
		}
		if len(sprints) == 0 {
			return nil, fmt.Errorf("no active sprint found")
		}
		sp = sprints[0]
	} else {
		sp, err = s.sprintRepo.GetByKey(ctx, sprintKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get sprint %q: %w", sprintKey, err)
		}
	}

	// Step 2: Validate sprint status.
	// Planning sprints have no meaningful burndown data (AC-B-2, TC-B-04).
	if sp.Status == models.SprintStatus("planning") {
		return &BurndownResult{
			SprintKey:  sp.Key,
			SprintName: sp.Name,
			DataPoints: []BurndownDataPoint{},
		}, nil
	}

	// Step 3: Load all entity assignments (assigned + removed)
	entities, err := s.analyticsRepo.GetSprintAssignedEntities(ctx, sp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned entities for sprint %q: %w", sp.Key, err)
	}

	// Step 4: Load completion events for task entities (within sprint window)
	completionEvents, err := s.analyticsRepo.GetCompletionEvents(ctx, sp.ID, sp.StartDate, sp.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion events for sprint %q: %w", sp.Key, err)
	}

	// Build a map of entity completion timestamps.
	// Key: (entityType, entityID) → earliest completion timestamp within the sprint.
	completedAt := make(map[burndownEntityKey]time.Time)
	for _, ev := range completionEvents {
		if isTerminalStatus(ev.NewStatus) {
			k := burndownEntityKey{entityType: ev.EntityType, entityID: ev.EntityID}
			if existing, ok := completedAt[k]; !ok || ev.Timestamp.Before(existing) {
				completedAt[k] = ev.Timestamp
			}
		}
	}

	// Step 5: Build day-by-day data points.
	today := truncateToDay(s.now())
	startDay := truncateToDay(sp.StartDate)
	endDay := truncateToDay(sp.EndDate)

	// Total sprint duration in days (inclusive: endDay-startDay+1)
	durationDays := int(endDay.Sub(startDay).Hours()/24) + 1
	if durationDays < 1 {
		durationDays = 1
	}

	// Pre-compute total sized and unsized at sprint start for initial ideal line.
	// Initial total = Σ sizes of entities assigned at or before start_date and
	// not removed before start_date.
	initialTotal := computeSizeAtDay(entities, startDay)

	// unsized count: count of currently-active unsized entities
	unsizedTotal := countUnsizedAtDay(entities, startDay)

	dataPoints := make([]BurndownDataPoint, 0, durationDays)

	// idealCurrentSize tracks the remaining size at the start of the current
	// ideal segment. It resets when entity adds/removes change the net total.
	idealCurrentSize := float64(initialTotal)
	idealSegmentStart := startDay

	for dayOffset := 0; dayOffset < durationDays; dayOffset++ {
		currentDay := startDay.AddDate(0, 0, dayOffset)

		// Detect net size change: recompute total on this day and check if it
		// changed versus yesterday (for piecewise ideal reset — AC-B-3, TC-B-06).
		if dayOffset > 0 {
			prevDay := startDay.AddDate(0, 0, dayOffset-1)
			prevTotal := computeSizeAtDay(entities, prevDay)
			currTotal := computeSizeAtDay(entities, currentDay)
			if currTotal != prevTotal {
				// Net size change: reset ideal segment from this day forward.
				idealCurrentSize = float64(currTotal)
				idealSegmentStart = currentDay
			}
		}

		// Ideal remaining: linear burn from idealCurrentSize at idealSegmentStart
		// to 0 at endDay. Formula: idealCurrentSize * (segmentDays - daysElapsed) / segmentDays
		// which gives: day0=total, day(segmentDays-1) ≈ 0 (overridden to 0 exactly).
		// This matches AC-B-7 example: total=42, day0=42, day1=42*13/14=39 (14-day sprint).
		segmentDays := int(endDay.Sub(idealSegmentStart).Hours()/24) + 1
		daysElapsed := int(currentDay.Sub(idealSegmentStart).Hours() / 24)
		var idealRemaining float64
		if segmentDays > 0 {
			idealRemaining = idealCurrentSize * float64(segmentDays-daysElapsed) / float64(segmentDays)
		}
		// On the last day of the sprint (endDay), ideal = 0.0 exactly.
		if currentDay.Equal(endDay) {
			idealRemaining = 0.0
		}

		// UnsizedRemaining: count of currently-active unsized entities on this day.
		unsizedRemaining := countUnsizedAtDay(entities, currentDay)

		dp := BurndownDataPoint{
			Date:             currentDay,
			IdealRemaining:   idealRemaining,
			UnsizedRemaining: unsizedRemaining,
		}

		// ActualRemaining: only for today or past days (nil for future — AC-B-8, TC-B-12).
		if !currentDay.After(today) {
			activeSize := computeActualRemainingAtDay(entities, completedAt, currentDay)
			f := float64(activeSize)
			dp.ActualRemaining = &f
		}

		dataPoints = append(dataPoints, dp)
	}

	return &BurndownResult{
		SprintKey:    sp.Key,
		SprintName:   sp.Name,
		TotalSize:    initialTotal,
		UnsizedTotal: unsizedTotal,
		DataPoints:   dataPoints,
	}, nil
}

// GetSummary returns a sprint summary report for the given sprint key.
// When detailed is true, additional fields are populated: cycle time by phase,
// size-band distribution, carryover entity list, mid-sprint add/remove counts.
//
// Spec reference: §4.4 — Summary calculation logic (5 numbered steps).
//
// Status validation: sprint must be in "completed" or "archived" status.
// Any other status returns an error describing the restriction (informational;
// the CLI layer renders this as a message rather than an error exit code — TC-S-02).
//
// Cycle-time graceful degradation (TC-S-06, AC-S-3): when GetCycleTimeByPhase
// returns an empty slice, CycleTimeByPhase is set to nil (not empty slice) so
// JSON callers can distinguish "no data" from "field not computed" (AC-S-4).
//
// Division-by-zero safety (TC-S-04): when planned_size=0, CompletionPctBySize
// returns 0.0 without panicking.
func (s *SprintAnalyticsService) GetSummary(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
	// Step 1: Load sprint.
	sp, err := s.sprintRepo.GetByKey(ctx, sprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint %s for summary: %w", sprintKey, err)
	}

	// Step 2: Validate status is completed or archived (AC-S-1, TC-S-02).
	status := string(sp.Status)
	if status != "completed" && status != "archived" {
		return nil, fmt.Errorf(
			"sprint summary is available for completed or archived sprints only; sprint %s is in status %q",
			sprintKey, status,
		)
	}

	// Step 3: Load all assigned entities.
	entities, err := s.analyticsRepo.GetSprintAssignedEntities(ctx, sp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned entities for sprint %s: %w", sprintKey, err)
	}

	// Step 4: Compute planned Σ size and counts.
	// "Planned" = entities assigned at or before start_date (assigned_at <= start_date).
	// Entities added after start_date are "mid-sprint adds".
	var (
		plannedSize         int
		plannedCount        int
		unsizedPlanned      int
		addedMidSprintCount int
		addedMidSprintSize  int
		removedCount        int
		removedSize         int
	)

	for _, e := range entities {
		isPlanned := !e.AssignedAt.After(sp.StartDate)
		isRemoved := e.RemovedAt != nil

		if isPlanned {
			plannedCount++
			if e.Size != nil {
				plannedSize += *e.Size
			} else {
				unsizedPlanned++
			}
		} else {
			// Mid-sprint add
			addedMidSprintCount++
			if e.Size != nil {
				addedMidSprintSize += *e.Size
			}
		}

		if isRemoved {
			removedCount++
			if e.Size != nil {
				removedSize += *e.Size
			}
		}
	}

	// Step 4b: Load completion events within the sprint window.
	completionEvents, err := s.analyticsRepo.GetCompletionEvents(ctx, sp.ID, sp.StartDate, sp.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion events for sprint %s: %w", sprintKey, err)
	}

	// Build a set of completed entity keys for O(1) lookup.
	completedSet := make(map[summaryEntityKey]bool, len(completionEvents))
	for _, ev := range completionEvents {
		completedSet[summaryEntityKey{ev.EntityType, ev.EntityID}] = true
	}

	// Compute completed Σ size and counts.
	var (
		completedSize    int
		completedCount   int
		unsizedCompleted int
	)
	for _, e := range entities {
		if completedSet[summaryEntityKey{e.EntityType, e.EntityID}] {
			completedCount++
			if e.Size != nil {
				completedSize += *e.Size
			} else {
				unsizedCompleted++
			}
		}
	}

	// Step 5: Compute trailing average velocity.
	// Spec §4.4 step 5: call GetVelocityData(ctx, 6) to fetch up to 6 completed
	// sprints, then exclude the current sprint from the rows and compute the
	// average over the remaining ≤5 prior sprints. We do NOT use
	// velocityResult.TrailingAverage because it includes the current sprint in
	// its denominator.
	var trailingAvgVelocity float64
	if priorRows, velErr := s.analyticsRepo.GetVelocityData(ctx, 6); velErr == nil {
		var priorTotal int
		var priorCount int
		for _, row := range priorRows {
			if row.SprintKey == sp.Key {
				continue // exclude the sprint being summarised
			}
			priorTotal += row.CompletedSize
			priorCount++
		}
		if priorCount > 0 {
			trailingAvgVelocity = float64(priorTotal) / float64(priorCount)
		}
	}

	velocityThisSprint := completedSize
	velocityDelta := float64(velocityThisSprint) - trailingAvgVelocity
	var velocityDeltaPct float64
	if trailingAvgVelocity != 0 {
		velocityDeltaPct = velocityDelta / trailingAvgVelocity * 100
	}

	// CompletionPctBySize: completed_size / planned_size * 100.
	// When planned_size = 0 → 0.0; no divide-by-zero panic (TC-S-04).
	var completionPctBySize float64
	if plannedSize > 0 {
		completionPctBySize = float64(completedSize) / float64(plannedSize) * 100
	}

	result := &SprintSummaryResult{
		SprintKey:           sp.Key,
		SprintName:          sp.Name,
		PlannedSize:         plannedSize,
		CompletedSize:       completedSize,
		CompletionPctBySize: completionPctBySize,
		PlannedCount:        plannedCount,
		CompletedCount:      completedCount,
		VelocityThisSprint:  velocityThisSprint,
		TrailingAvgVelocity: trailingAvgVelocity,
		VelocityDelta:       velocityDelta,
		VelocityDeltaPct:    velocityDeltaPct,
		UnsizedPlanned:      unsizedPlanned,
		UnsizedCompleted:    unsizedCompleted,
		// All detailed pointer fields stay nil unless detailed=true (AC-S-4, TC-S-07).
	}

	// Step 6: Populate detailed fields when requested (TC-S-05, AC-S-3).
	if detailed {
		result.AddedMidSprintCount = &addedMidSprintCount
		result.AddedMidSprintSize = &addedMidSprintSize
		result.RemovedMidSprintCount = &removedCount
		result.RemovedMidSprintSize = &removedSize

		// Cycle-time by phase (Decision 4 in spec §5).
		// Empty slice → nil in DTO (TC-S-06, AC-S-3).
		phaseRows, ctErr := s.analyticsRepo.GetCycleTimeByPhase(ctx, sp.ID)
		if ctErr == nil && len(phaseRows) > 0 {
			phases := make([]PhaseTime, len(phaseRows))
			for i, row := range phaseRows {
				phases[i] = PhaseTime(row)
			}
			result.CycleTimeByPhase = phases
		}
		// else: CycleTimeByPhase stays nil (empty slice or error → nil per spec).

		// Average completed size and size-band distribution.
		sizeLabelMap := map[int]string{1: "XS", 2: "S", 3: "M", 5: "L", 8: "XL", 13: "XXL"}
		sizeBands := map[string]int{}
		var totalCompletedSized int
		var completedSizedCount int

		for _, e := range entities {
			if completedSet[summaryEntityKey{e.EntityType, e.EntityID}] && e.Size != nil {
				totalCompletedSized += *e.Size
				completedSizedCount++
				if label, ok := sizeLabelMap[*e.Size]; ok {
					sizeBands[label]++
				}
			}
		}

		if completedSizedCount > 0 {
			avg := float64(totalCompletedSized) / float64(completedSizedCount)
			result.AvgCompletedSize = &avg
		}

		// Emit size bands with non-zero counts in canonical order.
		// Result stays nil when no recognized size labels are present so that
		// JSON serialises as null rather than [] (BUG-002 fix).
		bandOrder := []string{"XS", "S", "M", "L", "XL", "XXL"}
		var bands []SizeBand
		for _, label := range bandOrder {
			if count := sizeBands[label]; count > 0 {
				bands = append(bands, SizeBand{Label: label, Count: count})
			}
		}
		if len(bands) > 0 {
			result.SizeBandDistribution = bands
		}

		// Carryover entities: assigned (not removed) and not completed.
		var carryover []CarryoverEntity
		for _, e := range entities {
			if e.RemovedAt == nil && !completedSet[summaryEntityKey{e.EntityType, e.EntityID}] {
				carryover = append(carryover, CarryoverEntity{
					Key:        fmt.Sprintf("%s-%d", e.EntityType, e.EntityID),
					EntityType: e.EntityType,
					Size:       e.Size,
				})
			}
		}
		result.CarryoverEntities = carryover
	}

	return result, nil
}

// =============================================================================
// Private helpers
// =============================================================================

// burndownEntityKey identifies an entity in the completion-event map used
// during burndown reconstruction.
type burndownEntityKey struct {
	entityType string
	entityID   int64
}

// summaryEntityKey identifies an entity in the completed-entity set used during
// GetSummary calculations.
type summaryEntityKey struct {
	entityType string
	entityID   int64
}

// sprintStatusPtr returns a pointer to a SprintStatus — used to populate filter structs.
func sprintStatusPtr(s string) *models.SprintStatus {
	status := models.SprintStatus(s)
	return &status
}

// truncateToDay truncates a time to midnight UTC for day-boundary comparisons.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// isTerminalStatus returns true when the given status represents task completion.
// The list of terminal statuses is service-layer knowledge (per spec Decision 1
// and spec §4.3 note: "terminal states determined by the caller").
// Using a map for O(1) lookup.
func isTerminalStatus(status string) bool {
	terminals := map[string]bool{
		"completed": true,
		"done":      true,
		"closed":    true,
		"approved":  true,
		"cancelled": true,
		"archived":  true,
		"resolved":  true,
		"wont_fix":  true,
		"wont_do":   true,
		"dismissed": true,
	}
	return terminals[status]
}

// computeSizeAtDay returns the total sized value of all entities that are
// "active" on the given day: assigned on or before the day AND not removed
// before the day (removed_at IS NULL or removed_at > day end-of-day).
//
// Unsized entities (Size == nil) contribute 0 to the sum (per Decision 5 in spec).
func computeSizeAtDay(entities []AnalyticsAssignedEntity, day time.Time) int {
	dayEnd := day.Add(24*time.Hour - time.Nanosecond)
	total := 0
	for _, e := range entities {
		if !isActiveOnDay(e, day, dayEnd) {
			continue
		}
		if e.Size != nil {
			total += *e.Size
		}
	}
	return total
}

// countUnsizedAtDay counts entities active on the given day that have no size.
func countUnsizedAtDay(entities []AnalyticsAssignedEntity, day time.Time) int {
	dayEnd := day.Add(24*time.Hour - time.Nanosecond)
	count := 0
	for _, e := range entities {
		if isActiveOnDay(e, day, dayEnd) && e.Size == nil {
			count++
		}
	}
	return count
}

// isActiveOnDay returns true when an entity is active (assigned but not removed)
// within the given day window.
func isActiveOnDay(e AnalyticsAssignedEntity, dayStart, dayEnd time.Time) bool {
	// Entity must be assigned on or before the end of the day.
	if e.AssignedAt.After(dayEnd) {
		return false
	}
	// If removed, the removal must not have happened before the start of the day.
	if e.RemovedAt != nil && e.RemovedAt.Before(dayStart) {
		return false
	}
	return true
}

// computeActualRemainingAtDay computes the actual remaining Σ size on the
// given calendar day. An entity is "remaining" if it has not reached a terminal
// status as of end-of-day on that day.
//
// Only task entities have history events (per spec Decision 1); non-task entities
// are treated as "completed on all days" if they appear in the completedAt map
// (service layer applies point-in-time current status for non-task types).
func computeActualRemainingAtDay(
	entities []AnalyticsAssignedEntity,
	completedAt map[burndownEntityKey]time.Time,
	day time.Time,
) int {
	dayStart := day
	dayEnd := day.Add(24*time.Hour - time.Nanosecond)

	total := 0
	for _, e := range entities {
		if !isActiveOnDay(e, dayStart, dayEnd) {
			continue
		}
		k := burndownEntityKey{entityType: e.EntityType, entityID: e.EntityID}
		if completionTime, ok := completedAt[k]; ok && !completionTime.After(dayEnd) {
			// Completed on or before end of this day -- does not count.
			continue
		}
		if e.Size != nil {
			total += *e.Size
		}
	}
	return total
}
