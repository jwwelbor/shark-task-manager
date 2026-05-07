package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-S-07 / TC-S-08: JSON marshalling — detailed pointer fields render as null, not omitted.
// Per spec AC-S-4: missing work_sessions data is represented as null in JSON (not omitted),
// allowing callers to distinguish "no data" from "field not computed".
func TestSprintSummaryResult_JSONNullVsOmit(t *testing.T) {
	t.Run("detailed pointer fields render as null when nil (not omitted)", func(t *testing.T) {
		// Arrange: SprintSummaryResult with all detailed pointer fields nil (detailed=false scenario).
		result := &SprintSummaryResult{
			SprintKey:           "S024",
			SprintName:          "Sprint 24",
			PlannedSize:         50,
			CompletedSize:       40,
			CompletionPctBySize: 80.0,
			PlannedCount:        10,
			CompletedCount:      8,
			VelocityThisSprint:  40,
			TrailingAvgVelocity: 38.0,
			VelocityDelta:       2.0,
			VelocityDeltaPct:    5.26,
			UnsizedPlanned:      1,
			UnsizedCompleted:    0,
			// All detailed fields are nil (detailed=false)
			AddedMidSprintCount:   nil,
			AddedMidSprintSize:    nil,
			RemovedMidSprintCount: nil,
			RemovedMidSprintSize:  nil,
			CycleTimeByPhase:      nil,
			AvgCompletedSize:      nil,
			SizeBandDistribution:  nil,
			CarryoverEntities:     nil,
		}

		// Act: marshal to JSON.
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Assert: JSON string has null values for pointer fields, NOT missing keys.
		jsonStr := string(data)

		// Base fields must be present and non-null.
		assert.Contains(t, jsonStr, `"sprint_key":"S024"`)
		assert.Contains(t, jsonStr, `"sprint_name":"Sprint 24"`)
		assert.Contains(t, jsonStr, `"planned_size":50`)
		assert.Contains(t, jsonStr, `"completed_size":40`)
		assert.Contains(t, jsonStr, `"completion_pct_by_size":80`)
		assert.Contains(t, jsonStr, `"planned_count":10`)
		assert.Contains(t, jsonStr, `"completed_count":8`)
		assert.Contains(t, jsonStr, `"velocity_this_sprint":40`)
		assert.Contains(t, jsonStr, `"trailing_avg_velocity":38`)
		assert.Contains(t, jsonStr, `"velocity_delta":2`)
		assert.Contains(t, jsonStr, `"unsized_planned":1`)
		assert.Contains(t, jsonStr, `"unsized_completed":0`)

		// Detailed pointer fields must be present as null, NOT absent.
		// This is the key contract: omitempty must NOT be used on detailed fields.
		assert.Contains(t, jsonStr, `"added_mid_sprint_count":null`, "added_mid_sprint_count must be null, not omitted")
		assert.Contains(t, jsonStr, `"added_mid_sprint_size":null`, "added_mid_sprint_size must be null, not omitted")
		assert.Contains(t, jsonStr, `"removed_mid_sprint_count":null`, "removed_mid_sprint_count must be null, not omitted")
		assert.Contains(t, jsonStr, `"removed_mid_sprint_size":null`, "removed_mid_sprint_size must be null, not omitted")
		assert.Contains(t, jsonStr, `"cycle_time_by_phase":null`, "cycle_time_by_phase must be null, not omitted")
		assert.Contains(t, jsonStr, `"avg_completed_size":null`, "avg_completed_size must be null, not omitted")
		assert.Contains(t, jsonStr, `"size_band_distribution":null`, "size_band_distribution must be null, not omitted")
		assert.Contains(t, jsonStr, `"carryover_entities":null`, "carryover_entities must be null, not omitted")
	})

	t.Run("detailed fields populated when detailed=true", func(t *testing.T) {
		// Arrange: SprintSummaryResult with detailed fields populated.
		avgSize := 8.5
		addedCount := 2
		addedSize := 10
		removedCount := 1
		removedSize := 5
		result := &SprintSummaryResult{
			SprintKey:             "S024",
			SprintName:            "Sprint 24",
			AddedMidSprintCount:   &addedCount,
			AddedMidSprintSize:    &addedSize,
			RemovedMidSprintCount: &removedCount,
			RemovedMidSprintSize:  &removedSize,
			CycleTimeByPhase: []PhaseTime{
				{Phase: "in_progress", AverageDays: 2.5},
				{Phase: "in_review", AverageDays: 1.2},
			},
			AvgCompletedSize: &avgSize,
			SizeBandDistribution: []SizeBand{
				{Label: "S", Count: 3},
				{Label: "M", Count: 2},
			},
			CarryoverEntities: []CarryoverEntity{
				{Key: "T-E07-F01-010", EntityType: "task", Size: nil},
			},
		}

		// Act: marshal to JSON.
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Assert: detailed fields are populated in JSON.
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"added_mid_sprint_count":2`)
		assert.Contains(t, jsonStr, `"added_mid_sprint_size":10`)
		assert.Contains(t, jsonStr, `"removed_mid_sprint_count":1`)
		assert.Contains(t, jsonStr, `"removed_mid_sprint_size":5`)
		assert.Contains(t, jsonStr, `"cycle_time_by_phase":[`)
		assert.Contains(t, jsonStr, `"avg_completed_size":8.5`)
		assert.Contains(t, jsonStr, `"size_band_distribution":[`)
		assert.Contains(t, jsonStr, `"carryover_entities":[`)
	})

	t.Run("cycle_time_by_phase null when nil even in detailed mode", func(t *testing.T) {
		// Arrange: detailed=true but GetCycleTimeByPhase returned empty (work_sessions unavailable).
		// Per TC-S-08: CycleTimeByPhase=nil must marshal as null, not [].
		result := &SprintSummaryResult{
			SprintKey:        "S024",
			SprintName:       "Sprint 24",
			CycleTimeByPhase: nil, // nil because work_sessions data unavailable
		}

		// Act.
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Assert: null, not [] and not missing.
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"cycle_time_by_phase":null`,
			"CycleTimeByPhase nil must marshal as null, not [] and not omitted")
		assert.NotContains(t, jsonStr, `"cycle_time_by_phase":[]`,
			"CycleTimeByPhase nil must not marshal as empty array []")
	})
}

// TC-B-12: BurndownDataPoint with nil ActualRemaining marshals without actual_remaining key.
// Per spec AC-B-8: future days are represented as nil in ActualRemaining (*float64),
// and the JSON output omits the actual_remaining key (not null, not 0).
func TestBurndownDataPoint_NilActualRemainingOmittedFromJSON(t *testing.T) {
	t.Run("nil ActualRemaining omits actual_remaining key from JSON", func(t *testing.T) {
		// Arrange: future day data point — ActualRemaining is nil.
		dp := BurndownDataPoint{
			Date:             time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			IdealRemaining:   21.0,
			ActualRemaining:  nil, // future date — not yet known
			UnsizedRemaining: 3,
		}

		// Act.
		data, err := json.Marshal(dp)
		require.NoError(t, err)

		// Assert: actual_remaining is NOT present in JSON at all (omitempty behavior).
		jsonStr := string(data)
		assert.NotContains(t, jsonStr, `"actual_remaining"`,
			"nil ActualRemaining must be omitted from JSON, not rendered as null or 0")
		assert.Contains(t, jsonStr, `"ideal_remaining":21`)
		assert.Contains(t, jsonStr, `"unsized_remaining":3`)
	})

	t.Run("non-nil ActualRemaining is present in JSON", func(t *testing.T) {
		// Arrange: past day data point — ActualRemaining is known.
		actual := 40.0
		dp := BurndownDataPoint{
			Date:             time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
			IdealRemaining:   39.0,
			ActualRemaining:  &actual,
			UnsizedRemaining: 3,
		}

		// Act.
		data, err := json.Marshal(dp)
		require.NoError(t, err)

		// Assert: actual_remaining IS present with correct value.
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"actual_remaining":40`)
		assert.Contains(t, jsonStr, `"ideal_remaining":39`)
	})

	t.Run("boundary: zero ActualRemaining (0.0) is present, not omitted", func(t *testing.T) {
		// Arrange: day with all tasks completed — remaining is 0.0, but not nil.
		zero := 0.0
		dp := BurndownDataPoint{
			Date:             time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			IdealRemaining:   0.0,
			ActualRemaining:  &zero,
			UnsizedRemaining: 0,
		}

		// Act.
		data, err := json.Marshal(dp)
		require.NoError(t, err)

		// Assert: actual_remaining=0 IS present (zero value of non-nil pointer must not be omitted).
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"actual_remaining":0`,
			"zero ActualRemaining (*float64 = 0.0) must be present in JSON, not omitted")
	})
}

// TestVelocityResult_JSONSchema validates the VelocityResult JSON schema matches AC-V-6.
func TestVelocityResult_JSONSchema(t *testing.T) {
	t.Run("JSON schema matches AC-V-6 spec", func(t *testing.T) {
		// Arrange: VelocityResult with 2 sprints.
		result := &VelocityResult{
			Sprints: []VelocitySprint{
				{Key: "S001", Name: "Sprint 1", CompletedSize: 18, UnsizedCompleted: 2},
				{Key: "S002", Name: "Sprint 2", CompletedSize: 21, UnsizedCompleted: 0},
			},
			TrailingAverage:  19.5,
			SprintCount:      2,
			InsufficientData: false,
		}

		// Act.
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Assert: round-trip and field presence.
		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &parsed))

		sprints, ok := parsed["sprints"].([]interface{})
		assert.True(t, ok, "sprints must be an array")
		assert.Len(t, sprints, 2)

		trailingAvg, ok := parsed["trailing_average"].(float64)
		assert.True(t, ok, "trailing_average must be float64")
		assert.InDelta(t, 19.5, trailingAvg, 0.001)

		count, ok := parsed["sprint_count"].(float64)
		assert.True(t, ok, "sprint_count must be numeric")
		assert.Equal(t, float64(2), count)

		// Per-sprint keys must include unsized_completed.
		sprint0, ok := sprints[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "S001", sprint0["key"])
		assert.Equal(t, "Sprint 1", sprint0["name"])
		assert.Equal(t, float64(18), sprint0["completed_size"])
		assert.Equal(t, float64(2), sprint0["unsized_completed"])
	})

	t.Run("insufficient_data field present when InsufficientData=true", func(t *testing.T) {
		result := &VelocityResult{
			Sprints:          []VelocitySprint{},
			TrailingAverage:  0,
			SprintCount:      0,
			InsufficientData: true,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"insufficient_data":true`)
	})
}

// TestBurndownResult_JSONSchema validates the BurndownResult JSON schema matches AC-B-7.
func TestBurndownResult_JSONSchema(t *testing.T) {
	t.Run("JSON schema matches AC-B-7 spec", func(t *testing.T) {
		// Arrange: BurndownResult with two past-day data points.
		actual0 := 42.0
		actual1 := 40.0
		result := &BurndownResult{
			SprintKey:    "S024",
			SprintName:   "Sprint 24",
			TotalSize:    42,
			UnsizedTotal: 3,
			DataPoints: []BurndownDataPoint{
				{
					Date:             time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
					IdealRemaining:   42.0,
					ActualRemaining:  &actual0,
					UnsizedRemaining: 3,
				},
				{
					Date:             time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
					IdealRemaining:   39.0,
					ActualRemaining:  &actual1,
					UnsizedRemaining: 3,
				},
			},
		}

		// Act.
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Assert: top-level fields.
		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &parsed))

		assert.Equal(t, "S024", parsed["sprint_key"])
		assert.Equal(t, "Sprint 24", parsed["sprint_name"])
		assert.Equal(t, float64(42), parsed["total_size"])
		assert.Equal(t, float64(3), parsed["unsized_total"])

		// Data points.
		dataPoints, ok := parsed["data_points"].([]interface{})
		assert.True(t, ok, "data_points must be an array")
		assert.Len(t, dataPoints, 2)

		dp0, ok := dataPoints[0].(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, dp0["date"])
		assert.Equal(t, float64(42), dp0["ideal_remaining"])
		assert.Equal(t, float64(42), dp0["actual_remaining"])
		assert.Equal(t, float64(3), dp0["unsized_remaining"])
	})
}
