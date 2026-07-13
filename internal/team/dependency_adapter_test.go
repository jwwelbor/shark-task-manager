package team

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

type legacyDependencyMock struct {
	value string
}

func (m legacyDependencyMock) ListLegacyDependencies(context.Context, ChildIdentity) (string, error) {
	return m.value, nil
}

type relationshipDependencyMock struct {
	edges []DependencyEdge
}

func (m relationshipDependencyMock) ListRelationshipDependencies(context.Context, ChildIdentity) ([]DependencyEdge, error) {
	return append([]DependencyEdge(nil), m.edges...), nil
}

// TestDependencySources_TC003 verifies legacy JSON and normalized relationship
// edges are combined, sorted, and de-duplicated by typed canonical identity.
func TestDependencySources_TC003(t *testing.T) {
	adapter, err := NewDependencyAdapter(
		legacyDependencyMock{value: `["t-e38-f01-001", "T-E38-F01-002"]`},
		relationshipDependencyMock{edges: []DependencyEdge{
			{ChildKey: "T-E38-F01-003", DependencyKey: "T-E38-F01-001", DependencyType: models.EntityTypeTask, DependencyStatus: "completed", External: true},
			{ChildKey: "T-E38-F01-003", DependencyKey: "T-E38-F01-003", DependencyType: models.EntityTypeTask},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := adapter.ListDependencies(context.Background(), ChildIdentity{Key: "T-E38-F01-003", EntityType: models.EntityTypeTask})
	if err != nil {
		t.Fatalf("ListDependencies() error = %v", err)
	}
	if len(edges) != 3 {
		t.Fatalf("ListDependencies() returned %d edges, want 3", len(edges))
	}
	if edges[0].DependencyKey != "T-E38-F01-001" || edges[1].DependencyKey != "T-E38-F01-002" || edges[2].DependencyKey != "T-E38-F01-003" {
		t.Errorf("edges are not canonical/sorted: %+v", edges)
	}
	if !edges[0].Resolved || !edges[0].External || edges[0].DependencyStatus != "completed" || edges[0].Source != "relationship" {
		t.Fatalf("relationship metadata did not enrich duplicate legacy edge: %+v", edges[0])
	}
}

func TestDependencyAdapter_MarksRelationshipTargetsResolved_TC003(t *testing.T) {
	adapter, err := NewDependencyAdapter(nil, relationshipDependencyMock{edges: []DependencyEdge{{
		ChildKey: "T-E38-F01-003", DependencyKey: "T-E37-F01-001", DependencyType: models.EntityTypeTask, Source: "relationship",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	edges, err := adapter.ListDependencies(context.Background(), ChildIdentity{Key: "T-E38-F01-003", EntityType: models.EntityTypeTask})
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || !edges[0].Resolved {
		t.Fatalf("relationship target was not marked resolved: %+v", edges)
	}
}

// TestDependencySources_RejectMalformedLegacy_TC002 verifies malformed legacy
// JSON fails loudly instead of silently dropping a dependency edge.
func TestDependencySources_RejectMalformedLegacy_TC002(t *testing.T) {
	adapter, err := NewDependencyAdapter(legacyDependencyMock{value: `{"depends_on":`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.ListDependencies(context.Background(), ChildIdentity{Key: "T-E38-F01-003", EntityType: models.EntityTypeTask})
	if err == nil {
		t.Fatal("ListDependencies() returned nil error for malformed legacy JSON")
	}
	if !IsMalformedDependency(err) {
		t.Fatalf("ListDependencies() error = %v, want malformed dependency error", err)
	}
}
