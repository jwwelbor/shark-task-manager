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
