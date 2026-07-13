package contracts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/team"
	"github.com/stretchr/testify/require"
)

// TestTC001_I01TeamRunResultContract is the single shared I-01 contract test
// for E38 consumers. It deliberately converts the domain result through the
// producer boundary and verifies all durable fields without persisting prompt
// text, secrets, or unrestricted worker output.
func TestTC001_I01TeamRunResultContract(t *testing.T) {
	started := time.Date(2026, time.July, 13, 15, 0, 0, 0, time.UTC)
	completed := started.Add(3 * time.Minute)
	agentType := "developer"
	provider := "anthropic"
	model := "claude-sonnet"
	effort := "medium"
	claimSession := "claim-session-001"
	workerSession := "worker-session-001"
	outcome := "passed"
	skipReason := ""
	evidence := "artifact recorded"

	run := &team.TeamRun{
		ID:               41,
		RootKey:          "E38-F01-fixture",
		RootType:         models.EntityTypeFeature,
		Status:           team.RunStatusCompleted,
		ExecutionMode:    team.ExecutionModeParallel,
		ConcurrencyLimit: 2,
		PlanHash:         "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AggregateOutcome: &outcome,
		NextAction:       stringPtr("report"),
		RootSessionID:    &claimSession,
		StartedAt:        &started,
		CompletedAt:      &completed,
	}
	items := []*team.TeamRunItem{{
		ID:               101,
		TeamRunID:        run.ID,
		ChildKey:         "T-E38-F01-001",
		ChildType:        models.EntityTypeTask,
		Wave:             1,
		ExecutionOrder:   2,
		DependencyKeys:   []string{"T-E38-F01-000"},
		PlannedRole:      &agentType,
		PlannedAction:    stringPtr("spawn_agent"),
		PlannedAgentType: &agentType,
		PlannedProvider:  &provider,
		PlannedModel:     &model,
		PlannedEffort:    &effort,
		ItemStatus:       team.ItemStatusCompleted,
		ClaimSessionID:   &claimSession,
		WorkerSessionID:  &workerSession,
		Outcome:          &outcome,
		SkipReason:       &skipReason,
		Evidence:         evidence,
		ArtifactRefs:     []string{"docs/result.md"},
		Attempt:          1,
		StartedAt:        &started,
		CompletedAt:      &completed,
	}}

	result, err := team.NewTeamRunResult(run, items)
	require.NoError(t, err)
	require.Equal(t, run.ID, result.RunID)
	require.Equal(t, run.RootKey, result.RootKey)
	require.Equal(t, run.RootType, result.RootType)
	require.Equal(t, run.Status, result.Status)
	require.Equal(t, run.ExecutionMode, result.ExecutionMode)
	require.Equal(t, run.ConcurrencyLimit, result.ConcurrencyLimit)
	require.Equal(t, run.PlanHash, result.PlanHash)
	require.Equal(t, run.AggregateOutcome, result.AggregateOutcome)
	require.Equal(t, run.NextAction, result.NextAction)
	require.Len(t, result.Items, 1)
	require.Equal(t, items[0], result.Items[0])

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	serialized := string(encoded)
	require.NotContains(t, serialized, "prompt")
	require.NotContains(t, serialized, "secret")
	require.Contains(t, serialized, "claim_session_id")
	require.Contains(t, serialized, "worker_session_id")
	require.Contains(t, serialized, "artifact_refs")
}

func stringPtr(value string) *string { return &value }
