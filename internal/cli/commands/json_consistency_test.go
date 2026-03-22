package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestJSONConsistency_CommonEnrichmentFields verifies that all three entity JSON builders
// include the common enrichment fields: valid_transitions, orchestrator_action,
// related_documents, notes, context_data.
// Per ADR-005: additive only - never remove or rename existing fields.
func TestJSONConsistency_CommonEnrichmentFields(t *testing.T) {
	commonEnrichmentFields := []string{
		"valid_transitions",
		"orchestrator_action",
		"notes",
		"context_data",
		"related_documents",
	}

	t.Run("task JSON has common enrichment fields", func(t *testing.T) {
		task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
			Key:   "T-E01-F01-001",
			Title: "Test task",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, Status: "todo",
			Priority: 5,
		}

		result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)
		parsed := marshalAndParse(t, result)

		for _, field := range commonEnrichmentFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("task JSON missing common enrichment field %q", field)
			}
		}
	})

	t.Run("feature JSON has common enrichment fields", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

			Key:   "E01-F01",
			Title: "Test feature",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, EpicID: 1,

			Status: "active",
		}
		data := &FeatureGetData{}
		result := buildFeatureGetJSON(feature, data, nil)
		parsed := marshalAndParse(t, result)

		for _, field := range commonEnrichmentFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("feature JSON missing common enrichment field %q", field)
			}
		}
	})

	t.Run("epic JSON has common enrichment fields", func(t *testing.T) {
		epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
			Key:   "E01",
			Title: "Test epic",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, Status: "active",
		}
		data := &EpicGetData{}
		result := buildEpicGetJSON(epic, data, nil)
		parsed := marshalAndParse(t, result)

		for _, field := range commonEnrichmentFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("epic JSON missing common enrichment field %q", field)
			}
		}
	})
}

// TestJSONConsistency_FeatureMissingModelFields verifies that buildFeatureGetJSON
// includes all Feature model fields that are available but were previously omitted.
func TestJSONConsistency_FeatureMissingModelFields(t *testing.T) {
	slug := "test-feature"
	filePath := "docs/plan/E01/E01-F01/E01-F01.md"
	execOrder := 3

	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

		Key:   "E01-F01",
		Title: "Test feature",
		Slug:  &slug,

		FilePath: &filePath,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, EpicID: 1,

		Status: "active",

		ExecutionOrder: &execOrder,
	}

	data := &FeatureGetData{
		DirPath:  "docs/plan/E01/E01-F01",
		Filename: "E01-F01.md",
	}

	result := buildFeatureGetJSON(feature, data, nil)
	parsed := marshalAndParse(t, result)

	t.Run("includes slug when present", func(t *testing.T) {
		val, ok := parsed["slug"]
		if !ok {
			t.Fatal("feature JSON missing 'slug' field")
		}
		if val != "test-feature" {
			t.Errorf("expected slug 'test-feature', got %v", val)
		}
	})

	t.Run("includes file_path when present", func(t *testing.T) {
		val, ok := parsed["file_path"]
		if !ok {
			t.Fatal("feature JSON missing 'file_path' field")
		}
		if val != filePath {
			t.Errorf("expected file_path %q, got %v", filePath, val)
		}
	})

	t.Run("includes execution_order when present", func(t *testing.T) {
		val, ok := parsed["execution_order"]
		if !ok {
			t.Fatal("feature JSON missing 'execution_order' field")
		}
		// JSON numbers unmarshal as float64
		if val != float64(3) {
			t.Errorf("expected execution_order 3, got %v", val)
		}
	})
}

// TestJSONConsistency_FeatureOmitsNilOptionals verifies that nil optional fields
// are omitted from feature JSON (matching task builder behavior).
func TestJSONConsistency_FeatureOmitsNilOptionals(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

		Key:   "E01-F01",
		Title: "Test feature",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, EpicID: 1,

		Status: "active",

		// Slug, FilePath, ExecutionOrder are nil
	}

	data := &FeatureGetData{}
	result := buildFeatureGetJSON(feature, data, nil)
	parsed := marshalAndParse(t, result)

	// Nil optional fields should NOT be present (matching task builder pattern)
	for _, field := range []string{"slug", "file_path", "execution_order"} {
		if _, ok := parsed[field]; ok {
			t.Errorf("feature JSON should omit nil optional field %q, but it was present", field)
		}
	}
}

// TestJSONConsistency_FieldNaming verifies all JSON builders use snake_case consistently.
func TestJSONConsistency_FieldNaming(t *testing.T) {
	t.Run("task fields are snake_case", func(t *testing.T) {
		agentType := "backend"
		execOrder := 1
		task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
			Key:   "T-E01-F01-001",
			Title: "Test",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, Status: "todo",
			Priority:       5,
			AgentType:      &agentType,
			ExecutionOrder: &execOrder,
		}
		result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)
		parsed := marshalAndParse(t, result)
		assertSnakeCase(t, parsed, "task")
	})

	t.Run("feature fields are snake_case", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

			Key:   "E01-F01",
			Title: "Test",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, EpicID: 1,

			Status: "active",
		}
		data := &FeatureGetData{}
		result := buildFeatureGetJSON(feature, data, nil)
		parsed := marshalAndParse(t, result)
		assertSnakeCase(t, parsed, "feature")
	})

	t.Run("epic fields are snake_case", func(t *testing.T) {
		epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
			Key:   "E01",
			Title: "Test",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now()}, Status: "active",
		}
		data := &EpicGetData{}
		result := buildEpicGetJSON(epic, data, nil)
		parsed := marshalAndParse(t, result)
		assertSnakeCase(t, parsed, "epic")
	})
}

// TestJSONConsistency_FieldFlagAccess verifies that common fields accessible via
// --field flag work across all entity types.
func TestJSONConsistency_FieldFlagAccess(t *testing.T) {
	// Fields that should exist in all entity JSON outputs
	universalFields := []string{
		"id", "key", "title", "status",
		"created_at", "updated_at",
		"valid_transitions", "orchestrator_action",
		"notes", "context_data",
	}

	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "Test",

		CreatedAt: time.Now(), UpdatedAt: time.Now()}, Status: "todo", Priority: 5,
	}
	taskJSON := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)
	taskParsed := marshalAndParse(t, taskJSON)

	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01", Title: "Test",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}, EpicID: 1,
		Status: "active",
	}
	featureJSON := buildFeatureGetJSON(feature, &FeatureGetData{}, nil)
	featureParsed := marshalAndParse(t, featureJSON)

	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Test",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}, Status: "active",
	}
	epicJSON := buildEpicGetJSON(epic, &EpicGetData{}, nil)
	epicParsed := marshalAndParse(t, epicJSON)

	for _, field := range universalFields {
		t.Run("field_"+field, func(t *testing.T) {
			if _, ok := taskParsed[field]; !ok {
				t.Errorf("task JSON missing universal field %q (needed for --field flag)", field)
			}
			if _, ok := featureParsed[field]; !ok {
				t.Errorf("feature JSON missing universal field %q (needed for --field flag)", field)
			}
			if _, ok := epicParsed[field]; !ok {
				t.Errorf("epic JSON missing universal field %q (needed for --field flag)", field)
			}
		})
	}
}

// TestBuildFeatureGetJSON_FullPopulated verifies feature JSON with all fields populated,
// matching the pattern from TestBuildTaskGetJSON.
func TestBuildFeatureGetJSON_FullPopulated(t *testing.T) {
	slug := "test-feature"
	filePath := "docs/plan/E01/E01-F01/E01-F01.md"
	execOrder := 2
	desc := "Feature description"

	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

		Key:         "E01-F01",
		Title:       "Test Feature",
		Slug:        &slug,
		Description: &desc,

		FilePath:  &filePath,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, EpicID: 10,

		Status:         "active",
		StatusOverride: true,
		ProgressPct:    65.5,
		ExecutionOrder: &execOrder,
	}

	currentStep := "Implementing core logic"
	contextData := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep: &currentStep,
		},
	}

	data := &FeatureGetData{
		Tasks:           []*models.Task{{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001"}, Status: "todo"}},
		StatusBreakdown: []workflow.StatusCount{{Status: "todo", Count: 1}},
		DirPath:         "docs/plan/E01/E01-F01",
		Filename:        "E01-F01.md",
		RelatedDocs:     []*models.Document{{FilePath: "docs/spec.md", Title: "Spec"}},
		ProgressInfo:    &status.ProgressInfo{WeightedPct: 50.0},
		WorkSummary:     &status.WorkSummary{},
		ActionItems:     &status.ActionItems{},
		Notes:           []*models.EntityNote{{ID: 1, Content: "Review needed"}},
		ContextData:     contextData,
	}

	orchestratorAction := &config.PopulatedAction{
		Action:    "spawn_agent",
		AgentType: "developer",
	}

	result := buildFeatureGetJSON(feature, data, orchestratorAction)
	parsed := marshalAndParse(t, result)

	t.Run("includes all basic feature fields", func(t *testing.T) {
		for _, field := range []string{
			"id", "epic_id", "key", "title", "description", "status",
			"status_source", "status_override", "progress_pct",
			"path", "filename", "created_at", "updated_at",
		} {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing basic field %q", field)
			}
		}
	})

	t.Run("includes optional model fields", func(t *testing.T) {
		if parsed["slug"] != "test-feature" {
			t.Errorf("expected slug 'test-feature', got %v", parsed["slug"])
		}
		if parsed["file_path"] != filePath {
			t.Errorf("expected file_path %q, got %v", filePath, parsed["file_path"])
		}
		if parsed["execution_order"] != float64(2) {
			t.Errorf("expected execution_order 2, got %v", parsed["execution_order"])
		}
	})

	t.Run("includes enrichment data", func(t *testing.T) {
		for _, field := range []string{
			"tasks", "status_breakdown", "related_documents",
			"progress", "work_summary", "action_items",
			"notes", "context_data", "orchestrator_action", "valid_transitions",
		} {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing enrichment field %q", field)
			}
		}
	})

	t.Run("status_source is manual when overridden", func(t *testing.T) {
		if parsed["status_source"] != "manual" {
			t.Errorf("expected status_source 'manual', got %v", parsed["status_source"])
		}
	})

	t.Run("orchestrator_action has expected structure", func(t *testing.T) {
		action, ok := parsed["orchestrator_action"].(map[string]interface{})
		if !ok {
			t.Fatalf("orchestrator_action is not an object, got %T", parsed["orchestrator_action"])
		}
		if action["action"] != "spawn_agent" {
			t.Errorf("expected action=spawn_agent, got %v", action["action"])
		}
	})
}

// TestBuildEpicGetJSON_FullPopulated verifies epic JSON with all fields populated.
func TestBuildEpicGetJSON_FullPopulated(t *testing.T) {
	slug := "test-epic"
	filePath := "docs/plan/E01/E01.md"
	desc := "Epic description"

	bv := models.PriorityHigh
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
		Key:         "E01",
		Title:       "Test Epic",
		Slug:        &slug,
		Description: &desc,

		FilePath:  &filePath,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "active",
		Priority:      models.PriorityHigh,
		BusinessValue: &bv,
	}

	data := &EpicGetData{
		DirPath:              "docs/plan/E01",
		Filename:             "E01.md",
		EpicProgress:         75.0,
		RelatedDocs:          []*models.Document{{FilePath: "docs/spec.md"}},
		FeatureRollup:        map[string]int{"active": 2, "completed": 1},
		TaskRollup:           map[string]int{"todo": 3, "completed": 5},
		BlockedTasks:         []*models.Task{},
		ApprovalBacklogCount: 2,
		EpicNotes:            []*models.EntityNote{{ID: 1, Content: "Note"}},
		EpicContext:          &models.ContextData{},
	}

	result := buildEpicGetJSON(epic, data, nil)
	parsed := marshalAndParse(t, result)

	t.Run("includes all basic epic fields", func(t *testing.T) {
		for _, field := range []string{
			"id", "key", "title", "description", "status", "status_source",
			"priority", "business_value", "slug", "progress_pct",
			"path", "filename", "file_path", "created_at", "updated_at",
		} {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing basic field %q", field)
			}
		}
	})

	t.Run("includes enrichment data", func(t *testing.T) {
		for _, field := range []string{
			"features", "related_documents", "feature_status_rollup",
			"task_status_rollup", "impediments", "approval_backlog_count",
			"notes", "context_data", "orchestrator_action", "valid_transitions",
		} {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing enrichment field %q", field)
			}
		}
	})
}

// marshalAndParse is a test helper that marshals a map to JSON and parses it back.
func marshalAndParse(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	return parsed
}

// assertSnakeCase verifies that all top-level field names in a JSON map use snake_case.
func assertSnakeCase(t *testing.T, parsed map[string]interface{}, entityType string) {
	t.Helper()
	for key := range parsed {
		for i, c := range key {
			if c >= 'A' && c <= 'Z' {
				t.Errorf("%s JSON field %q contains uppercase at position %d (expected snake_case)", entityType, key, i)
				break
			}
		}
	}
}
