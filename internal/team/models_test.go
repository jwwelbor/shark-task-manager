package team

import (
	"errors"
	"strings"
	"testing"
)

func TestLedgerOutput_RejectsSensitiveContent_TC009(t *testing.T) {
	base := ItemResultUpdate{
		RunID:        1,
		ItemID:       1,
		Attempt:      0,
		Status:       ItemStatusCompleted,
		Outcome:      "passed",
		Evidence:     "completed the requested work",
		ArtifactRefs: []string{"artifacts/run-1/result.md"},
	}

	tests := []struct {
		name   string
		mutate func(*ItemResultUpdate)
		cause  error
	}{
		{name: "evidence over bound", mutate: func(update *ItemResultUpdate) {
			update.Evidence = strings.Repeat("x", MaxEvidenceBytes+1)
		}, cause: ErrEvidenceTooLarge},
		{name: "bearer token", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "Bearer sk-test-123"
		}, cause: ErrSensitiveEvidence},
		{name: "private key", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "-----BEGIN PRIVATE KEY-----"
		}, cause: ErrSensitiveEvidence},
		{name: "rendered prompt", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "rendered prompt: do the work"
		}, cause: ErrSensitiveEvidence},
		{name: "absolute artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"/tmp/secret"}
		}, cause: ErrInvalidArtifactPath},
		{name: "traversal artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"../secret"}
		}, cause: ErrInvalidArtifactPath},
		{name: "duplicate artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"artifacts/result.md", "artifacts/result.md"}
		}, cause: ErrInvalidArtifactPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := base
			tt.mutate(&update)
			if err := update.Validate(); !errors.Is(err, tt.cause) {
				t.Fatalf("ItemResultUpdate.Validate() error = %v, want %v", err, tt.cause)
			}
		})
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
}

func TestTeamRunResult_ContainsCompleteSafeShape_TC014(t *testing.T) {
	run := &TeamRun{
		ID:               7,
		RootKey:          "E38-F01",
		RootType:         "feature",
		Status:           RunStatusCompleted,
		ExecutionMode:    ExecutionModeParallel,
		ConcurrencyLimit: 2,
		PlanHash:         strings.Repeat("a", 64),
	}
	items := []*TeamRunItem{{
		ID:             9,
		TeamRunID:      7,
		ChildKey:       "T-E38-F01-001",
		ChildType:      "task",
		Wave:           1,
		ItemStatus:     ItemStatusCompleted,
		Attempt:        1,
		Outcome:        stringPtr("passed"),
		Evidence:       "bounded summary",
		ArtifactRefs:   []string{"artifacts/result.md"},
		ClaimSessionID: stringPtr("claim-1"),
	}}

	result, err := NewTeamRunResult(run, items)
	if err != nil {
		t.Fatalf("NewTeamRunResult() error = %v", err)
	}
	if result.RunID != 7 || result.RootKey != "E38-F01" || len(result.Items) != 1 {
		t.Fatalf("incomplete result: %+v", result)
	}
	if result.Items[0].Attempt != 1 || result.Items[0].ClaimSessionID == nil {
		t.Fatalf("item result lost lifecycle fields: %+v", result.Items[0])
	}
}

func stringPtr(value string) *string { return &value }
