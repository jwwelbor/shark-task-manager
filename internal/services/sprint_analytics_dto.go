package services

import "time"

// SprintSummaryResult is the return value of GetSprintSummary (E19-F04).
// It contains velocity metrics, carryover details, and optionally detailed
// cycle-time and size-band distribution data when detailed=true is requested.
//
// All "detailed" fields are pointer types so they can be nil when detailed=false.
// The JSON tags deliberately omit `omitempty` on these pointer fields so that nil
// marshals as JSON null rather than being omitted (AC-S-4 contract: callers can
// distinguish "no data" from "field not computed"). See TC-S-07 and TC-S-08.
type SprintSummaryResult struct {
	SprintKey           string  `json:"sprint_key"`
	SprintName          string  `json:"sprint_name"`
	PlannedSize         int     `json:"planned_size"`
	CompletedSize       int     `json:"completed_size"`
	CompletionPctBySize float64 `json:"completion_pct_by_size"`
	PlannedCount        int     `json:"planned_count"`
	CompletedCount      int     `json:"completed_count"`
	VelocityThisSprint  int     `json:"velocity_this_sprint"`
	TrailingAvgVelocity float64 `json:"trailing_avg_velocity"`
	VelocityDelta       float64 `json:"velocity_delta"`
	VelocityDeltaPct    float64 `json:"velocity_delta_pct"`
	UnsizedPlanned      int     `json:"unsized_planned"`
	UnsizedCompleted    int     `json:"unsized_completed"`

	// Detailed fields -- nil when detailed=false.
	// Must NOT use omitempty so nil renders as JSON null (AC-S-4).
	AddedMidSprintCount   *int              `json:"added_mid_sprint_count"`
	AddedMidSprintSize    *int              `json:"added_mid_sprint_size"`
	RemovedMidSprintCount *int              `json:"removed_mid_sprint_count"`
	RemovedMidSprintSize  *int              `json:"removed_mid_sprint_size"`
	CycleTimeByPhase      []PhaseTime       `json:"cycle_time_by_phase"`
	AvgCompletedSize      *float64          `json:"avg_completed_size"`
	SizeBandDistribution  []SizeBand        `json:"size_band_distribution"`
	CarryoverEntities     []CarryoverEntity `json:"carryover_entities"`
}

// PhaseTime records the average cycle time in days for a workflow phase.
// Used by SprintSummaryResult.CycleTimeByPhase (E19-F04).
type PhaseTime struct {
	Phase       string  `json:"phase"`
	AverageDays float64 `json:"average_days"`
}

// SizeBand represents a bucket in the size-band distribution histogram.
// Label maps to story-point labels (XS, S, M, L, XL, XXL); Count is the
// number of entities completed in this sprint with that size label (E19-F04).
type SizeBand struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// CarryoverEntity identifies an entity that was not completed at sprint close
// and was either moved to the next sprint or dropped to the backlog.
// Used by SprintSummaryResult.CarryoverEntities (E19-F04).
type CarryoverEntity struct {
	Key        string `json:"key"`
	EntityType string `json:"entity_type"`
	Size       *int   `json:"size,omitempty"`
}

// BurndownDataPoint represents a single day's data on the sprint burndown chart.
// IdealRemaining is always populated (deterministic linear burn from total size).
// ActualRemaining is nil for future dates (not yet known) and uses omitempty so
// future-day entries do not carry an actual_remaining key in JSON (AC-B-8).
// UnsizedRemaining counts entities with no size that are still incomplete.
type BurndownDataPoint struct {
	Date             time.Time `json:"date"`
	IdealRemaining   float64   `json:"ideal_remaining"`
	ActualRemaining  *float64  `json:"actual_remaining,omitempty"` // nil = future date; omitted from JSON
	UnsizedRemaining int       `json:"unsized_remaining"`
}

// BurndownResult is the return value of GetSprintBurndown (E19-F04).
type BurndownResult struct {
	SprintKey    string              `json:"sprint_key"`
	SprintName   string              `json:"sprint_name"`
	TotalSize    int                 `json:"total_size"`
	UnsizedTotal int                 `json:"unsized_total"`
	DataPoints   []BurndownDataPoint `json:"data_points"`
}

// VelocitySprint is a single-sprint entry in the VelocityResult.
type VelocitySprint struct {
	Key              string `json:"key"`
	Name             string `json:"name"`
	CompletedSize    int    `json:"completed_size"`
	UnsizedCompleted int    `json:"unsized_completed"`
}

// VelocityResult is the return value of GetVelocityHistory (E19-F04).
// InsufficientData is true when fewer than the requested number of sprints
// have completion records (e.g., trailing-3 requested but only 1 sprint closed).
type VelocityResult struct {
	Sprints          []VelocitySprint `json:"sprints"`
	TrailingAverage  float64          `json:"trailing_average"`
	SprintCount      int              `json:"sprint_count"`
	InsufficientData bool             `json:"insufficient_data"`
}
