package services

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDTOJSONSerialization verifies that all DTOs can be serialized to/from JSON.
// This is a requirement for HTTP API compatibility.
func TestDTOJSONSerialization(t *testing.T) {
	t.Run("CreateTaskInput", func(t *testing.T) {
		input := CreateTaskInput{
			EpicKey:    "E07",
			FeatureKey: "F01",
			Title:      "Test Task",
			AgentType:  "backend",
			Priority:   5,
		}

		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("Failed to marshal CreateTaskInput: %v", err)
		}

		var unmarshaled CreateTaskInput
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CreateTaskInput: %v", err)
		}

		if unmarshaled.Title != input.Title {
			t.Errorf("Expected title %s, got %s", input.Title, unmarshaled.Title)
		}
	})

	t.Run("TaskFilters", func(t *testing.T) {
		filters := TaskFilters{
			EpicKey: "E07",
			Status:  "todo",
		}

		data, err := json.Marshal(filters)
		if err != nil {
			t.Fatalf("Failed to marshal TaskFilters: %v", err)
		}

		var unmarshaled TaskFilters
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal TaskFilters: %v", err)
		}

		if unmarshaled.EpicKey != filters.EpicKey {
			t.Errorf("Expected epic key %s, got %s", filters.EpicKey, unmarshaled.EpicKey)
		}
	})

	t.Run("DependencyTree", func(t *testing.T) {
		tree := DependencyTree{
			Task: &TaskNode{
				Key:       "E07-F01-001",
				Title:     "Test Task",
				Status:    "todo",
				UpdatedAt: time.Now(),
			},
			Blocked:  false,
			CanStart: true,
			Depth:    0,
		}

		data, err := json.Marshal(tree)
		if err != nil {
			t.Fatalf("Failed to marshal DependencyTree: %v", err)
		}

		var unmarshaled DependencyTree
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal DependencyTree: %v", err)
		}

		if unmarshaled.Task.Key != tree.Task.Key {
			t.Errorf("Expected task key %s, got %s", tree.Task.Key, unmarshaled.Task.Key)
		}
	})

	t.Run("TaskUpdates", func(t *testing.T) {
		title := "Updated Title"
		priority := 8
		updates := TaskUpdates{
			Title:    &title,
			Priority: &priority,
		}

		data, err := json.Marshal(updates)
		if err != nil {
			t.Fatalf("Failed to marshal TaskUpdates: %v", err)
		}

		var unmarshaled TaskUpdates
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal TaskUpdates: %v", err)
		}

		if *unmarshaled.Title != *updates.Title {
			t.Errorf("Expected title %s, got %s", *updates.Title, *unmarshaled.Title)
		}
	})
}
