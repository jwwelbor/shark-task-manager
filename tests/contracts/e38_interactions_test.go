package contracts

import (
	"encoding/json"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/team"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestTC001_I01TeamRunResultContract is the complete shared I-01 shape test.
// Persisted conversion is covered by the repository-owned TC-014 test.
func TestTC001_I01TeamRunResultContract(t *testing.T) {
	outcome, agent, provider, model, effort := "passed", "developer", "anthropic", "claude-sonnet", "medium"
	claim, worker, skip := "claim-session-001", "worker-session-001", ""
	started := time.Date(2026, time.July, 13, 15, 0, 0, 0, time.UTC)
	completed := started.Add(3 * time.Minute)
	run := &team.TeamRun{ID: 41, RootKey: "E38-F01-fixture", RootType: models.EntityTypeFeature, Status: team.RunStatusCompleted, ExecutionMode: team.ExecutionModeParallel, ConcurrencyLimit: 2, PlanHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", AggregateOutcome: &outcome, NextAction: &claim, RootSessionID: &claim, StartedAt: &started, CompletedAt: &completed}
	item := &team.TeamRunItem{ID: 101, TeamRunID: run.ID, ChildKey: "T-E38-F01-001", ChildType: models.EntityTypeTask, Wave: 1, ExecutionOrder: 2, DependencyKeys: []string{"T-E38-F01-000"}, PlannedRole: &agent, PlannedAction: &claim, PlannedAgentType: &agent, PlannedProvider: &provider, PlannedModel: &model, PlannedEffort: &effort, ItemStatus: team.ItemStatusCompleted, ClaimSessionID: &claim, WorkerSessionID: &worker, Outcome: &outcome, SkipReason: &skip, Evidence: "artifact recorded", ArtifactRefs: []string{"docs/result.md"}, Attempt: 1, StartedAt: &started, CompletedAt: &completed}
	result, err := team.NewTeamRunResult(run, []*team.TeamRunItem{item})
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
	require.Equal(t, 1, result.Counts.Total)
	require.Equal(t, 1, result.Counts.ByStatus[string(team.ItemStatusCompleted)])
	require.Equal(t, 0, result.Counts.ByStatus[string(team.ItemStatusFailed)])
	require.Len(t, result.Items, 1)
	got := result.Items[0]
	require.Equal(t, item.ID, got.ID)
	require.Equal(t, item.TeamRunID, got.TeamRunID)
	require.Equal(t, item.ChildKey, got.ChildKey)
	require.Equal(t, item.ChildType, got.ChildType)
	require.Equal(t, item.Wave, got.Wave)
	require.Equal(t, item.ExecutionOrder, got.ExecutionOrder)
	require.Equal(t, item.DependencyKeys, got.DependencyKeys)
	require.Equal(t, item.PlannedRole, got.PlannedRole)
	require.Equal(t, item.PlannedAction, got.PlannedAction)
	require.Equal(t, item.PlannedAgentType, got.PlannedAgentType)
	require.Equal(t, item.PlannedProvider, got.PlannedProvider)
	require.Equal(t, item.PlannedModel, got.PlannedModel)
	require.Equal(t, item.PlannedEffort, got.PlannedEffort)
	require.Equal(t, item.ItemStatus, got.ItemStatus)
	require.Equal(t, item.ClaimSessionID, got.ClaimSessionID)
	require.Equal(t, item.WorkerSessionID, got.WorkerSessionID)
	require.Equal(t, item.Outcome, got.Outcome)
	require.Equal(t, item.SkipReason, got.SkipReason)
	require.Equal(t, item.Evidence, got.Evidence)
	require.Equal(t, item.ArtifactRefs, got.ArtifactRefs)
	require.Equal(t, item.Attempt, got.Attempt)
	require.Equal(t, item.StartedAt, got.StartedAt)
	require.Equal(t, item.CompletedAt, got.CompletedAt)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "prompt")
	require.NotContains(t, string(encoded), "secret")
	require.Contains(t, string(encoded), "claim_session_id")
	require.Contains(t, string(encoded), "worker_session_id")
	require.Contains(t, string(encoded), "artifact_refs")
	require.Contains(t, string(encoded), "\"counts\"")
	require.Contains(t, string(encoded), "\"by_status\"")
}

// TestTC004_I04CouncilCommunicationContract is the shared serialized shape
// consumed by F02/F03/F05 from the F04 role and communication protocol. Keep
// this producer-neutral: the durable artifact may be YAML or JSON, but these
// fields and their meanings must remain stable across feature boundaries.
func TestTC004_I04CouncilCommunicationContract(t *testing.T) {
	serialized := []byte(`{
		"sender_role": "developer",
		"recipient_role": "chair",
		"root_key": "E38-F02",
		"child_key": "T-E38-F02-001",
		"subject": "implementation handoff",
		"requested_action": "review the bounded dispatch result",
		"urgency": "normal",
		"evidence_links": ["docs/council/handoffs/T-E38-F02-001.md"],
		"handoff": {
			"summary": "The child completed with bounded evidence.",
			"open_questions": ["Should the next wave proceed?"]
		},
		"decision": {
			"outcome": "proceed",
			"rationale": "All required child evidence is present."
		},
		"created_at": "2026-07-13T15:00:00Z"
	}`)

	var message map[string]any
	require.NoError(t, json.Unmarshal(serialized, &message))
	require.Equal(t, "developer", message["sender_role"])
	require.Equal(t, "chair", message["recipient_role"])
	require.Equal(t, "E38-F02", message["root_key"])
	require.Equal(t, "T-E38-F02-001", message["child_key"])
	require.Equal(t, "normal", message["urgency"])
	require.Equal(t, "implementation handoff", message["subject"])
	require.Equal(t, "review the bounded dispatch result", message["requested_action"])
	require.Equal(t, "2026-07-13T15:00:00Z", message["created_at"])
	require.Equal(t, []any{"docs/council/handoffs/T-E38-F02-001.md"}, message["evidence_links"])
	require.Equal(t, map[string]any{
		"summary":        "The child completed with bounded evidence.",
		"open_questions": []any{"Should the next wave proceed?"},
	}, message["handoff"])
	require.Equal(t, map[string]any{
		"outcome":   "proceed",
		"rationale": "All required child evidence is present.",
	}, message["decision"])

	// Communication metadata scopes and informs execution; it is not a Shark
	// mutation authority or a substitute for workflow/claim ownership.
	require.NotContains(t, message, "claim")
	require.NotContains(t, message, "status_transition")
	require.NotContains(t, message, "force")

	roundTrip, err := json.Marshal(message)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(roundTrip, &decoded))
	require.Equal(t, message, decoded)
}
