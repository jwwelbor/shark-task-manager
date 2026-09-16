package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func completeLiveCollection(planned []ownership) []techDebt {
	live := make([]techDebt, 0, len(planned))
	for _, item := range planned {
		live = append(live, techDebt{Key: item.Key, Status: item.ExpectedStatus})
	}
	return live
}

func TestReconcileResidualTechDebt(t *testing.T) {
	planned, err := loadPlanned()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		mutate      func([]techDebt) []techDebt
		valid       bool
		duplicates  []string
		invalid     []string
		missing     []string
		extra       []string
		statusDrift []string
		active      int
		planned     int
	}{
		{name: "clean live collection", mutate: func(live []techDebt) []techDebt { return live }, valid: true, active: 50, planned: 50},
		{name: "duplicate active key", mutate: func(live []techDebt) []techDebt {
			return append(live, techDebt{Key: "TD-212", Status: statusInProgress})
		}, duplicates: []string{"TD-212"}, active: 50, planned: 50},
		{name: "duplicate with conflicting status", mutate: func(live []techDebt) []techDebt {
			return append(live, techDebt{Key: "TD-212", Status: statusIdentified})
		}, duplicates: []string{"TD-212"}, active: 50, planned: 50},
		{name: "blank key", mutate: func(live []techDebt) []techDebt { return append(live, techDebt{Status: statusIdentified}) }, invalid: []string{"record[50]: blank key"}, active: 50, planned: 50},
		{name: "status mismatch", mutate: func(live []techDebt) []techDebt { live[0].Status = statusIdentified; return live }, statusDrift: []string{"TD-212"}, active: 50, planned: 50},
		{name: "unplanned live key is extra", mutate: func(live []techDebt) []techDebt {
			return append(live, techDebt{Key: "TD-999", Status: statusIdentified})
		}, extra: []string{"TD-999"}, active: 51, planned: 50},
		{name: "planned key absent from live collection is missing", mutate: func(live []techDebt) []techDebt { return live[1:] }, missing: []string{"TD-212"}, active: 49, planned: 50},
		{name: "terminal record from all-status input is extra", mutate: func(live []techDebt) []techDebt { return append(live, techDebt{Key: "TD-999", Status: "resolved"}) }, extra: []string{"TD-999"}, active: 51, planned: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := reconcile(tt.mutate(completeLiveCollection(planned)), planned)
			require.Equal(t, tt.valid, report.valid())
			require.ElementsMatch(t, tt.duplicates, report.Duplicates)
			require.ElementsMatch(t, tt.invalid, report.InvalidRecords)
			require.ElementsMatch(t, tt.missing, report.Missing)
			require.ElementsMatch(t, tt.extra, report.Extra)
			require.ElementsMatch(t, tt.statusDrift, report.StatusMismatched)
			require.Equal(t, tt.active, report.Active)
			require.Equal(t, tt.planned, report.Planned)
		})
	}
}

func TestLoadLive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"key":"TD-212","status":"in_progress"}]`), 0o600))
	live, err := loadLive(path)
	require.NoError(t, err)
	require.Len(t, live, 1)
	_, err = loadLive(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, []byte(`not-json`), 0o600))
	_, err = loadLive(path)
	require.Error(t, err)
}
