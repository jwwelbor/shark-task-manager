package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type mockPlanHierarchySnapshotReader struct {
	ReadDirectChildrenFunc func(
		ctx context.Context,
		parentType, parentKey string,
		claimTTL time.Duration,
		evaluatedAt time.Time,
	) (planhierarchyrepo.Snapshot, error)
}

func (m *mockPlanHierarchySnapshotReader) ReadDirectChildren(
	ctx context.Context,
	parentType, parentKey string,
	claimTTL time.Duration,
	evaluatedAt time.Time,
) (planhierarchyrepo.Snapshot, error) {
	return m.ReadDirectChildrenFunc(ctx, parentType, parentKey, claimTTL, evaluatedAt)
}

type mockPlanHierarchyClaimPolicy struct {
	ttl time.Duration
}

func (m mockPlanHierarchyClaimPolicy) TTL() time.Duration { return m.ttl }

const asymmetricPlanHierarchyWorkflowConfig = `{
  "task_workflow": {
    "statuses": ["task_ready", "task_done"],
    "status_flow": {"task_ready": ["task_done"], "task_done": []},
    "special_statuses": {"_start_": ["task_ready"], "_complete_": ["task_done"]},
    "status_metadata": {
      "task_ready": {"color": "blue", "phase": "development"},
      "task_done": {"color": "green", "phase": "done"}
    }
  },
  "feature_workflow": {
    "statuses": ["feature_ready", "feature_done"],
    "status_flow": {"feature_ready": ["feature_done"], "feature_done": []},
    "special_statuses": {"_start_": ["feature_ready"], "_complete_": ["feature_done"]},
    "status_metadata": {
      "feature_ready": {"color": "blue", "phase": "development"},
      "feature_done": {"color": "green", "phase": "done"}
    }
  },
  "epic_workflow": {
    "statuses": ["epic_ready", "epic_done"],
    "status_flow": {"epic_ready": ["epic_done"], "epic_done": []},
    "special_statuses": {"_start_": ["epic_ready"], "_complete_": ["epic_done"]},
    "status_metadata": {
      "epic_ready": {"color": "blue", "phase": "development"},
      "epic_done": {"color": "green", "phase": "done"}
    }
  }
}`

// TestPlanHierarchyServiceFiltersInMemoryAfterOneSetRead pins the one-query
// contract for `shark plan <epic|feature>`: exactly one snapshot read per
// call, with terminal/claimed/blocked children filtered in memory.
func TestPlanHierarchyServiceFiltersInMemoryAfterOneSetRead(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	order := 1
	calls := 0
	reader := &mockPlanHierarchySnapshotReader{
		ReadDirectChildrenFunc: func(
			_ context.Context,
			parentType, parentKey string,
			claimTTL time.Duration,
			_ time.Time,
		) (planhierarchyrepo.Snapshot, error) {
			calls++
			if parentType != "feature" || parentKey != "E07-F01" {
				t.Fatalf("read target = %s %s", parentType, parentKey)
			}
			if claimTTL != 20*time.Minute {
				t.Fatalf("claim TTL = %v, want 20m", claimTTL)
			}
			return planhierarchyrepo.Snapshot{
				ParentFound: true,
				Children: []planhierarchyrepo.Child{
					{
						Key: "T-E07-F01-001", Title: "Ready", Status: "todo",
						EntityType: models.EntityTypeTask, ExecutionOrder: &order,
					},
					{
						Key: "T-E07-F01-002", Title: "Claimed", Status: "todo",
						EntityType: models.EntityTypeTask, Claimed: true,
					},
					{
						Key: "T-E07-F01-003", Title: "Blocked", Status: "todo",
						EntityType: models.EntityTypeTask,
						Dependencies: []planhierarchyrepo.Dependency{{
							Key: "T-E07-F01-099", Status: "in_progress",
						}},
					},
					{
						Key: "T-E07-F01-004", Title: "Done", Status: "shipped",
						EntityType: models.EntityTypeTask,
					},
				},
			}, nil
		},
	}
	service := services.NewPlanHierarchyService(
		reader,
		wf,
		mockPlanHierarchyClaimPolicy{ttl: 20 * time.Minute},
		services.PlanHierarchyEdgeReaders{},
	)

	state, err := service.DescribeChildren(
		context.Background(),
		"feature",
		"E07-F01",
	)
	if err != nil {
		t.Fatalf("DescribeChildren() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("snapshot reads = %d, want exactly one", calls)
	}
	if state.TotalChildren != 4 || state.NonTerminalChildren != 3 {
		t.Fatalf("state counts = %#v", state)
	}
	if len(state.Children) != 1 || state.Children[0].Key != "T-E07-F01-001" {
		t.Fatalf("claimable children = %#v, want only direct ready task", state.Children)
	}
}

func TestPlanHierarchyServiceUsesChildWorkflowForTerminalFiltering(t *testing.T) {
	tests := []struct {
		name        string
		parentType  string
		parentKey   string
		entityType  models.EntityType
		terminal    string
		notTerminal string
		wantKey     string
	}{
		{
			name: "epic children use feature workflow", parentType: "epic", parentKey: "E07",
			entityType: models.EntityTypeFeature,
			terminal:   "feature_done", notTerminal: "task_done", wantKey: "E07-F02",
		},
		{
			name: "feature children use task workflow", parentType: "feature", parentKey: "E07-F01",
			entityType: models.EntityTypeTask,
			terminal:   "task_done", notTerminal: "feature_done", wantKey: "T-E07-F01-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &mockPlanHierarchySnapshotReader{
				ReadDirectChildrenFunc: func(
					_ context.Context,
					parentType string,
					parentKey string,
					_ time.Duration,
					_ time.Time,
				) (planhierarchyrepo.Snapshot, error) {
					if parentType != tt.parentType || parentKey != tt.parentKey {
						t.Fatalf(
							"read target = %s %s, want %s %s",
							parentType, parentKey, tt.parentType, tt.parentKey,
						)
					}
					return planhierarchyrepo.Snapshot{
						ParentFound: true,
						Children: []planhierarchyrepo.Child{
							{Key: "terminal", Status: tt.terminal, EntityType: tt.entityType},
							{Key: tt.wantKey, Status: tt.notTerminal, EntityType: tt.entityType},
						},
					}, nil
				},
			}
			service := services.NewPlanHierarchyService(
				reader,
				newWorkflowService(t, asymmetricPlanHierarchyWorkflowConfig),
				mockPlanHierarchyClaimPolicy{},
				services.PlanHierarchyEdgeReaders{},
			)

			state, err := service.DescribeChildren(
				context.Background(),
				tt.parentType,
				tt.parentKey,
			)
			if err != nil {
				t.Fatalf("DescribeChildren() error = %v", err)
			}
			if state.NonTerminalChildren != 1 {
				t.Fatalf("NonTerminalChildren = %d, want 1", state.NonTerminalChildren)
			}
			if len(state.Children) != 1 || state.Children[0].Key != tt.wantKey {
				t.Fatalf("children = %#v, want only %s", state.Children, tt.wantKey)
			}
		})
	}
}

func TestPlanHierarchyServiceParentNotFoundErrors(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	reader := &mockPlanHierarchySnapshotReader{
		ReadDirectChildrenFunc: func(context.Context, string, string, time.Duration, time.Time) (planhierarchyrepo.Snapshot, error) {
			return planhierarchyrepo.Snapshot{ParentFound: false}, nil
		},
	}
	service := services.NewPlanHierarchyService(reader, wf, mockPlanHierarchyClaimPolicy{}, services.PlanHierarchyEdgeReaders{})
	if _, err := service.DescribeChildren(context.Background(), "epic", "E99"); err == nil {
		t.Fatal("DescribeChildren() error = nil, want not-found error")
	}
}
