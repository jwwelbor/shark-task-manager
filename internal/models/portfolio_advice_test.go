package models

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestNewPortfolioAdviceEnvelopeMarshalsAllocatedEmptyCollections(t *testing.T) {
	envelope := NewPortfolioAdviceEnvelope()
	envelope.Epics = append(envelope.Epics, PortfolioEpicEvidence{
		Key:                "E01",
		Title:              "First epic",
		Status:             "active",
		Priority:           "high",
		BusinessValue:      nil,
		Eligibility:        PortfolioEligibilityEligible,
		EligibilityReasons: []string{},
		BlockedItems:       []PortfolioBlockedItem{},
		ActiveWork:         []PortfolioActiveWork{},
	})

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := decoded["mode"]; got != "portfolio_advice" {
		t.Errorf("mode = %v, want portfolio_advice", got)
	}
	assertJSONArray(t, decoded, "epics")
	assertJSONArray(t, decoded, "relationships")
	assertJSONArray(t, decoded, "warnings")

	ordering := assertJSONObject(t, decoded, "ordering")
	assertEmptyJSONArray(t, ordering, "dependency_layers")
	assertEmptyJSONArray(t, ordering, "roadmap_layers")
	assertEmptyJSONArray(t, ordering, "unlayered_epics")
	assertEmptyJSONArray(t, ordering, "warnings")

	epics := assertJSONArray(t, decoded, "epics")
	if len(epics) != 1 {
		t.Fatalf("len(epics) = %d, want 1", len(epics))
	}
	epic, ok := epics[0].(map[string]any)
	if !ok {
		t.Fatalf("epics[0] type = %T, want object", epics[0])
	}
	assertEmptyJSONArray(t, epic, "eligibility_reasons")
	assertEmptyJSONArray(t, epic, "blocked_items")
	assertEmptyJSONArray(t, epic, "active_work")
	assertJSONNull(t, epic, "business_value")
}

func TestPortfolioAdviceNullableFieldsRemainPresent(t *testing.T) {
	value := struct {
		Epic         PortfolioEpicEvidence     `json:"epic"`
		ActiveWork   PortfolioActiveWork       `json:"active_work"`
		Relationship PortfolioEpicRelationship `json:"relationship"`
	}{
		Epic: PortfolioEpicEvidence{
			EligibilityReasons: []string{},
			BlockedItems:       []PortfolioBlockedItem{},
			ActiveWork:         []PortfolioActiveWork{},
		},
		ActiveWork: PortfolioActiveWork{},
		Relationship: PortfolioEpicRelationship{
			RelationshipType: EntityRelFollows,
		},
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	assertJSONNull(t, assertJSONObject(t, decoded, "epic"), "business_value")
	assertJSONNull(t, assertJSONObject(t, decoded, "active_work"), "progress")
	assertJSONNull(t, assertJSONObject(t, decoded, "relationship"), "satisfied")
}

func TestPortfolioActiveWorkJSONExcludesSensitiveClaimFields(t *testing.T) {
	progress := 0.5
	activeWork := PortfolioActiveWork{
		EntityType:    "task",
		EntityKey:     "T-E01-F01-001",
		ClaimedBy:     "developer",
		LastHeartbeat: time.Date(2026, 7, 20, 15, 4, 5, 0, time.UTC),
		Progress:      &progress,
	}

	encoded, err := json.Marshal(activeWork)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	gotKeys := make([]string, 0, len(decoded))
	for key := range decoded {
		gotKeys = append(gotKeys, key)
	}
	sort.Strings(gotKeys)
	wantKeys := []string{"claimed_by", "entity_key", "entity_type", "last_heartbeat", "progress"}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Errorf("JSON keys = %v, want %v", gotKeys, wantKeys)
	}
}

func TestPortfolioAdviceDTOJSONFieldNames(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{
			name:  "envelope",
			value: PortfolioAdviceEnvelope{},
			want:  []string{"epics", "evidence_complete", "mode", "ordering", "prompt", "relationships", "warnings"},
		},
		{
			name:  "epic evidence",
			value: PortfolioEpicEvidence{},
			want:  []string{"active_work", "blocked_items", "business_value", "eligibility", "eligibility_reasons", "key", "priority", "progress_pct", "status", "title"},
		},
		{
			name:  "blocked item",
			value: PortfolioBlockedItem{},
			want:  []string{"entity_key", "entity_type", "kind", "status", "title"},
		},
		{
			name:  "active work",
			value: PortfolioActiveWork{},
			want:  []string{"claimed_by", "entity_key", "entity_type", "last_heartbeat", "progress"},
		},
		{
			name:  "relationship",
			value: PortfolioEpicRelationship{},
			want:  []string{"from_key", "from_status", "hard", "relationship_type", "satisfied", "to_key", "to_status"},
		},
		{
			name:  "warning",
			value: PortfolioWarning{},
			want:  []string{"code", "epic_keys", "message"},
		},
		{
			name:  "ordering",
			value: PortfolioOrdering{},
			want:  []string{"dependency_layers", "roadmap_layers", "unlayered_epics", "warnings"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			got := make([]string, 0, len(decoded))
			for key := range decoded {
				got = append(got, key)
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSON keys = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortfolioAdviceVocabulary(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "mode", got: string(PortfolioAdviceModePortfolioAdvice), want: "portfolio_advice"},
		{name: "eligible", got: string(PortfolioEligibilityEligible), want: "eligible"},
		{name: "ineligible", got: string(PortfolioEligibilityIneligible), want: "ineligible"},
		{name: "unknown", got: string(PortfolioEligibilityUnknown), want: "unknown"},
		{name: "workflow blocker", got: string(PortfolioBlockerWorkflowBlocked), want: "workflow_blocked"},
		{name: "hard dependency", got: string(PortfolioBlockerHardDependency), want: "hard_dependency"},
		{name: "incoming block", got: string(PortfolioBlockerIncomingBlock), want: "incoming_block"},
		{name: "hard cycle", got: string(PortfolioWarningHardOrderCycle), want: "HARD_ORDER_CYCLE"},
		{name: "roadmap cycle", got: string(PortfolioWarningRoadmapOrderCycle), want: "ROADMAP_ORDER_CYCLE"},
		{name: "contradictory order", got: string(PortfolioWarningContradictoryOrder), want: "CONTRADICTORY_ORDER"},
		{name: "missing order", got: string(PortfolioWarningMissingOrdering), want: "MISSING_ORDERING"},
		{name: "claim state unavailable", got: string(PortfolioWarningClaimStateUnavailable), want: "CLAIM_STATE_UNAVAILABLE"},
		{name: "unknown workflow status", got: string(PortfolioWarningUnknownWorkflowStatus), want: "UNKNOWN_WORKFLOW_STATUS"},
		{name: "dangling relationship", got: string(PortfolioWarningDanglingRelationship), want: "DANGLING_RELATIONSHIP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("value = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func assertJSONObject(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("JSON field %q is absent", key)
	}
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON field %q type = %T, want object", key, value)
	}
	return result
}

func assertJSONArray(t *testing.T, object map[string]any, key string) []any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("JSON field %q is absent", key)
	}
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("JSON field %q type = %T, want array", key, value)
	}
	return result
}

func assertEmptyJSONArray(t *testing.T, object map[string]any, key string) {
	t.Helper()
	if got := assertJSONArray(t, object, key); len(got) != 0 {
		t.Errorf("JSON field %q = %v, want empty array", key, got)
	}
}

func assertJSONNull(t *testing.T, object map[string]any, key string) {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("JSON field %q is absent", key)
	}
	if value != nil {
		t.Errorf("JSON field %q = %v, want null", key, value)
	}
}
