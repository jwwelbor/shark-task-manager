package commands

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// TestBuildTaskGetJSON verifies that buildTaskGetJSON includes all enriched fields
// that the human-readable view displays.
func TestBuildTaskGetJSON(t *testing.T) {
	now := time.Now()
	agentType := "backend"
	filePath := "docs/plan/E07/E07-F01/tasks/T-E07-F01-001.md"
	description := "Implement JWT validation"
	blockedReason := "Waiting on API spec"
	execOrder := 2

	task := &models.Task{
		ID:             1,
		FeatureID:      10,
		Key:            "T-E07-F01-001",
		Title:          "Implement auth",
		Status:         "in_progress",
		AgentType:      &agentType,
		Priority:       5,
		ExecutionOrder: &execOrder,
		FilePath:       &filePath,
		Description:    &description,
		BlockedReason:  &blockedReason,
		CreatedAt:      now,
		UpdatedAt:      now,
		StartedAt:      sql.NullTime{Time: now, Valid: true},
	}

	deps := []*models.Task{
		{Key: "T-E07-F01-000", Status: "completed"},
	}

	blockedBy := []services.RelationshipWithTask{
		{TaskKey: "T-E07-F01-002", TaskTitle: "API Spec", TaskStatus: "in_progress", RelationshipType: "blocks", Direction: "incoming"},
	}

	blocks := []services.RelationshipWithTask{
		{TaskKey: "T-E07-F01-003", TaskTitle: "Deploy", TaskStatus: "todo", RelationshipType: "blocks", Direction: "outgoing"},
	}

	relatedDocs := []*models.Document{
		{FilePath: "docs/specs/auth.md", Title: "Auth Spec"},
	}

	validTransitions := []string{"ready_for_review", "blocked"}

	orchestratorAction := &config.PopulatedAction{
		Action:      "spawn_agent",
		AgentType:   "backend",
		Skills:      []string{"implementation"},
		Instruction: "Implement JWT token validation",
	}

	notes := []*models.EntityNote{
		{ID: 1, Content: "Review needed"},
	}

	currentStep := "Working on auth implementation"
	contextData := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep: &currentStep,
		},
	}

	result := buildTaskGetJSON(task, deps, blockedBy, blocks, relatedDocs,
		validTransitions, orchestratorAction, notes, contextData)

	// Verify it marshals to valid JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("buildTaskGetJSON result failed to marshal: %v", err)
	}

	// Parse back as generic map to verify all fields
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	t.Run("includes basic task fields", func(t *testing.T) {
		for _, field := range []string{"id", "key", "title", "status", "priority", "feature_id",
			"agent_type", "execution_order", "file_path", "description", "blocked_reason",
			"created_at", "updated_at", "started_at"} {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing expected field %q in JSON output", field)
			}
		}
	})

	t.Run("includes orchestrator_action", func(t *testing.T) {
		action, ok := parsed["orchestrator_action"]
		if !ok {
			t.Fatal("missing orchestrator_action in JSON output")
		}
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			t.Fatalf("orchestrator_action is not an object, got %T", action)
		}
		if actionMap["action"] != "spawn_agent" {
			t.Errorf("expected action=spawn_agent, got %v", actionMap["action"])
		}
		if actionMap["agent_type"] != "backend" {
			t.Errorf("expected agent_type=backend, got %v", actionMap["agent_type"])
		}
		if actionMap["instruction"] != "Implement JWT token validation" {
			t.Errorf("expected instruction text, got %v", actionMap["instruction"])
		}
	})

	t.Run("includes valid_transitions", func(t *testing.T) {
		transitions, ok := parsed["valid_transitions"]
		if !ok {
			t.Fatal("missing valid_transitions in JSON output")
		}
		transArr, ok := transitions.([]interface{})
		if !ok {
			t.Fatalf("valid_transitions is not an array, got %T", transitions)
		}
		if len(transArr) != 2 {
			t.Errorf("expected 2 transitions, got %d", len(transArr))
		}
	})

	t.Run("includes related_documents", func(t *testing.T) {
		docs, ok := parsed["related_documents"]
		if !ok {
			t.Fatal("missing related_documents in JSON output")
		}
		docsArr, ok := docs.([]interface{})
		if !ok {
			t.Fatalf("related_documents is not an array, got %T", docs)
		}
		if len(docsArr) != 1 {
			t.Errorf("expected 1 related doc, got %d", len(docsArr))
		}
	})

	t.Run("includes blocked_by", func(t *testing.T) {
		bb, ok := parsed["blocked_by"]
		if !ok {
			t.Fatal("missing blocked_by in JSON output")
		}
		bbArr, ok := bb.([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array, got %T", bb)
		}
		if len(bbArr) != 1 {
			t.Errorf("expected 1 blocked_by entry, got %d", len(bbArr))
		}
	})

	t.Run("includes blocks", func(t *testing.T) {
		b, ok := parsed["blocks"]
		if !ok {
			t.Fatal("missing blocks in JSON output")
		}
		bArr, ok := b.([]interface{})
		if !ok {
			t.Fatalf("blocks is not an array, got %T", b)
		}
		if len(bArr) != 1 {
			t.Errorf("expected 1 blocks entry, got %d", len(bArr))
		}
	})

	t.Run("includes dependencies", func(t *testing.T) {
		d, ok := parsed["dependencies"]
		if !ok {
			t.Fatal("missing dependencies in JSON output")
		}
		dArr, ok := d.([]interface{})
		if !ok {
			t.Fatalf("dependencies is not an array, got %T", d)
		}
		if len(dArr) != 1 {
			t.Errorf("expected 1 dependency, got %d", len(dArr))
		}
	})

	t.Run("includes notes", func(t *testing.T) {
		n, ok := parsed["notes"]
		if !ok {
			t.Fatal("missing notes in JSON output")
		}
		nArr, ok := n.([]interface{})
		if !ok {
			t.Fatalf("notes is not an array, got %T", n)
		}
		if len(nArr) != 1 {
			t.Errorf("expected 1 note, got %d", len(nArr))
		}
	})

	t.Run("includes context_data", func(t *testing.T) {
		cd, ok := parsed["context_data"]
		if !ok {
			t.Fatal("missing context_data in JSON output")
		}
		cdMap, ok := cd.(map[string]interface{})
		if !ok {
			t.Fatalf("context_data is not an object, got %T", cd)
		}
		progress, ok := cdMap["progress"].(map[string]interface{})
		if !ok {
			t.Fatalf("context_data.progress is not an object, got %T", cdMap["progress"])
		}
		if progress["current_step"] != "Working on auth implementation" {
			t.Errorf("expected current_step text, got %v", progress["current_step"])
		}
	})
}

// TestBuildTaskGetJSON_EmptyOptionals verifies graceful handling of nil/empty optional fields.
func TestBuildTaskGetJSON_EmptyOptionals(t *testing.T) {
	task := &models.Task{
		ID:        1,
		Key:       "T-E07-F01-001",
		Title:     "Simple task",
		Status:    "todo",
		Priority:  5,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Core fields must always be present
	if parsed["key"] != "T-E07-F01-001" {
		t.Errorf("expected key T-E07-F01-001, got %v", parsed["key"])
	}
	if parsed["title"] != "Simple task" {
		t.Errorf("expected title Simple task, got %v", parsed["title"])
	}

	// Optional fields should be nil or empty arrays, not missing
	if _, ok := parsed["orchestrator_action"]; !ok {
		t.Error("orchestrator_action field should exist (even if null)")
	}
	if _, ok := parsed["valid_transitions"]; !ok {
		t.Error("valid_transitions field should exist (even if empty)")
	}
}
