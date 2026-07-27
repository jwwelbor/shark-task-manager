package commands

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/require"
)

func planEdgeTestChildren() []services.PlanHierarchyChild {
	order := 1
	return []services.PlanHierarchyChild{
		{
			Key: "T-E01-F01-001", Title: "First", Status: "todo",
			EntityType: models.EntityTypeTask, ExecutionOrder: &order,
		},
		{
			Key: "T-E01-F01-002", Title: "Second", Status: "todo",
			EntityType: models.EntityTypeTask, ExecutionOrder: &order,
		},
	}
}

func planCandidateJSONFields(t *testing.T, candidate any) []string {
	t.Helper()
	raw, err := json.Marshal(candidate)
	require.NoError(t, err)
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	fields := make([]string, 0, len(decoded))
	for field := range decoded {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

// TestBuildHierarchyPlanSelectionEmitsNoEdgeFields pins that `shark plan`'s
// selection output stays edge-less. buildHierarchyPlanSelection never populates
// DependsOn/Blocks/Links, and the omitempty tags must keep them out of the
// serialized envelope entirely — on both the singleton and parallel shapes.
// This is an exact field-set assertion, so an edge key appearing by accident
// fails here rather than silently widening the plan contract.
func TestBuildHierarchyPlanSelectionEmitsNoEdgeFields(t *testing.T) {
	wantCandidateFields := []string{
		"entity_key", "entity_type", "execution_order", "status", "title",
	}

	children := planEdgeTestChildren()
	parallel := buildHierarchyPlanSelection("E01-F01", "feature", children, "parallel_tie", 10)
	require.Equal(t, "parallel_candidates", parallel.Action)
	require.Len(t, parallel.Entities, 2)
	for index := range parallel.Entities {
		if got := planCandidateJSONFields(t, parallel.Entities[index]); !reflect.DeepEqual(
			got, wantCandidateFields,
		) {
			t.Fatalf("parallel candidate %d JSON fields = %#v, want %#v", index, got, wantCandidateFields)
		}
	}

	singleton := buildHierarchyPlanSelection("E01-F01", "feature", children[:1], "execution_order", 10)
	require.NotNil(t, singleton.Entity)
	if got := planCandidateJSONFields(t, singleton.Entity); !reflect.DeepEqual(
		got, wantCandidateFields,
	) {
		t.Fatalf("singleton candidate JSON fields = %#v, want %#v", got, wantCandidateFields)
	}

	// Envelope level: outputHierarchyPlanSelectionJSON marshals the whole
	// response, so the top-level field set is the wire contract. Neither the
	// edge fields nor resolved_via may appear in plan's own output.
	if got := planCandidateJSONFields(t, singleton); !reflect.DeepEqual(got, []string{
		"action", "entity", "mode", "root_key", "root_type", "selection_reason",
	}) {
		t.Fatalf("singleton envelope JSON fields = %#v", got)
	}
	if got := planCandidateJSONFields(t, parallel); !reflect.DeepEqual(got, []string{
		"action", "entities", "mode", "parallel_execution", "root_key", "root_type", "selection_reason",
	}) {
		t.Fatalf("parallel envelope JSON fields = %#v", got)
	}
}

// TestApplyCandidateEdgesPopulatesBothEnvelopeShapes covers the seam a keyed
// fork caller uses: edges are attached after buildHierarchyPlanSelection
// returns, for the singleton Entity pointer as well as the Entities slice.
func TestApplyCandidateEdgesPopulatesBothEnvelopeShapes(t *testing.T) {
	children := planEdgeTestChildren()
	edges := map[string]services.PlanHierarchyEdges{
		"T-E01-F01-001": {
			DependsOn: []services.PlanHierarchyEdge{
				{Key: "T-E01-F01-009", Status: "shipped", Type: "depends_on"},
			},
			Blocks: []services.PlanHierarchyEdge{
				{Key: "T-E01-F01-002", Status: "todo", Type: "blocks"},
			},
			Links: []services.PlanHierarchyEdge{
				{Key: "B001", Status: "open", Type: "related_to"},
			},
			Warnings: []services.PlanHierarchyEdgeWarning{{
				Code:             services.PlanHierarchyWarningDanglingRelationship,
				Direction:        "outgoing",
				RelationshipID:   42,
				EndpointType:     models.EntityTypeEpic,
				EndpointID:       99,
				RelationshipType: models.EntityRelDependsOn,
			}},
		},
	}

	parallel := buildHierarchyPlanSelection("E01-F01", "feature", children, "parallel_tie", 10)
	applyCandidateEdges(&parallel, edges)
	require.Equal(t, []CandidateEdge{
		{Key: "T-E01-F01-009", Status: "shipped", Type: "depends_on"},
	}, parallel.Entities[0].DependsOn)
	require.Equal(t, []CandidateEdge{
		{Key: "T-E01-F01-002", Status: "todo", Type: "blocks"},
	}, parallel.Entities[0].Blocks)
	require.Equal(t, []CandidateEdge{
		{Key: "B001", Status: "open", Type: "related_to"},
	}, parallel.Entities[0].Links)
	require.Equal(t, []CandidateEdgeWarning{{
		Code:             services.PlanHierarchyWarningDanglingRelationship,
		Direction:        "outgoing",
		RelationshipID:   42,
		EndpointType:     "epic",
		EndpointID:       99,
		RelationshipType: "depends_on",
	}}, parallel.Entities[0].Warnings)

	// A candidate with no map entry stays edge-less rather than being zeroed.
	require.Nil(t, parallel.Entities[1].DependsOn)
	require.Nil(t, parallel.Entities[1].Blocks)
	require.Nil(t, parallel.Entities[1].Links)
	require.Nil(t, parallel.Entities[1].Warnings)

	singleton := buildHierarchyPlanSelection("E01-F01", "feature", children[:1], "execution_order", 10)
	applyCandidateEdges(&singleton, edges)
	require.NotNil(t, singleton.Entity)
	require.Len(t, singleton.Entity.DependsOn, 1)
	if got := planCandidateJSONFields(t, singleton.Entity); !reflect.DeepEqual(got, []string{
		"blocks", "depends_on", "entity_key", "entity_type", "execution_order", "links", "status", "title",
		"warnings",
	}) {
		t.Fatalf("candidate-with-edges JSON fields = %#v", got)
	}
}
