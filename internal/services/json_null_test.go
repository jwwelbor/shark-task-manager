package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSprintSummaryResult_NilDetailedFields_JSONNull verifies AC-S-4:
// nil detailed pointer/slice fields must appear as JSON null (not omitted).
// Counter-factual: if omitempty were added to these tags, the fields would
// vanish from JSON output, breaking callers that check field == null.
func TestSprintSummaryResult_NilDetailedFields_JSONNull(t *testing.T) {
	result := &SprintSummaryResult{
		SprintKey:           "S024",
		SprintName:          "Sprint S024",
		PlannedSize:         10,
		CompletedSize:       8,
		CompletionPctBySize: 80.0,
		PlannedCount:        2,
		CompletedCount:      2,
		VelocityThisSprint:  8,
		TrailingAvgVelocity: 7.5,
		VelocityDelta:       0.5,
		VelocityDeltaPct:    6.67,
		UnsizedPlanned:      0,
		UnsizedCompleted:    0,
		// All detailed fields nil:
		AddedMidSprintCount:   nil,
		AddedMidSprintSize:    nil,
		RemovedMidSprintCount: nil,
		RemovedMidSprintSize:  nil,
		CycleTimeByPhase:      nil,
		AvgCompletedSize:      nil,
		SizeBandDistribution:  nil,
		CarryoverEntities:     nil,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Each of these must be PRESENT (not missing) and its value must be null.
	requiredNullFields := []string{
		"added_mid_sprint_count",
		"added_mid_sprint_size",
		"removed_mid_sprint_count",
		"removed_mid_sprint_size",
		"cycle_time_by_phase",
		"avg_completed_size",
		"size_band_distribution",
		"carryover_entities",
	}

	for _, field := range requiredNullFields {
		val, present := parsed[field]
		assert.True(t, present, "field %q must be present in JSON (not omitted)", field)
		assert.Nil(t, val, "field %q must be JSON null, got %v", field, val)
	}
}
