package config

import (
	"encoding/json"
	"testing"
)

// TestDatabaseConfig_Marshaling tests that DatabaseConfig can be marshaled and unmarshaled
func TestDatabaseConfig_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "config with turso backend",
			config: Config{
				Database: &DatabaseConfig{
					Backend:         "turso",
					URL:             "libsql://shark-tasks.turso.io",
					AuthTokenFile:   "~/.shark/turso-token",
					EmbeddedReplica: true,
				},
			},
			expected: `{"database":{"backend":"turso","url":"libsql://shark-tasks.turso.io","auth_token_file":"~/.shark/turso-token","embedded_replica":true}}`,
		},
		{
			name: "config with local backend",
			config: Config{
				Database: &DatabaseConfig{
					Backend: "local",
					URL:     "./shark-tasks.db",
				},
			},
			expected: `{"database":{"backend":"local","url":"./shark-tasks.db"}}`,
		},
		{
			name:     "config without database (backward compat)",
			config:   Config{},
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("marshaled JSON mismatch\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			// Unmarshal back
			var unmarshaled Config
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			// Verify fields
			if tt.config.Database != nil {
				if unmarshaled.Database == nil {
					t.Error("database config was lost during unmarshal")
					return
				}
				if unmarshaled.Database.Backend != tt.config.Database.Backend {
					t.Errorf("backend mismatch: got %q, want %q", unmarshaled.Database.Backend, tt.config.Database.Backend)
				}
				if unmarshaled.Database.URL != tt.config.Database.URL {
					t.Errorf("url mismatch: got %q, want %q", unmarshaled.Database.URL, tt.config.Database.URL)
				}
			}
		})
	}
}

// TestDatabaseConfig_DefaultValues tests that nil database config is handled gracefully
func TestDatabaseConfig_DefaultValues(t *testing.T) {
	config := Config{}

	if config.Database != nil {
		t.Error("expected Database to be nil by default")
	}

	// Should be safe to check nil database config
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config with nil database: %v", err)
	}

	expected := `{}`
	if string(jsonData) != expected {
		t.Errorf("expected empty object, got: %s", string(jsonData))
	}
}

// TestDatabaseConfig_ValidationBackend tests backend validation
func TestDatabaseConfig_ValidationBackend(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		url     string
		valid   bool
	}{
		{"turso backend", "turso", "libsql://db.turso.io", true},
		{"local backend", "local", "./shark-tasks.db", true},
		{"sqlite backend (alias for local)", "sqlite", "./shark-tasks.db", true},
		{"empty backend (auto-detect)", "", "./shark-tasks.db", true},
		{"invalid backend", "postgres", "./shark-tasks.db", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DatabaseConfig{
				Backend: tt.backend,
				URL:     tt.url,
			}
			err := config.Validate()

			if tt.valid && err != nil {
				t.Errorf("expected backend %q to be valid, got error: %v", tt.backend, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected backend %q to be invalid, but validation passed", tt.backend)
			}
		})
	}
}

// TestDatabaseConfig_ValidationURL tests URL validation
func TestDatabaseConfig_ValidationURL(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		url     string
		valid   bool
	}{
		{"turso with libsql URL", "turso", "libsql://shark-tasks.turso.io", true},
		{"turso with https URL", "turso", "https://shark-tasks.turso.io", true},
		{"local with file path", "local", "./shark-tasks.db", true},
		{"local with absolute path", "local", "/home/user/shark-tasks.db", true},
		{"turso with empty URL", "turso", "", false},
		{"local with empty URL", "local", "", false},
		{"turso with file path", "turso", "./shark-tasks.db", false},
		{"local with libsql URL", "local", "libsql://db.turso.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DatabaseConfig{
				Backend: tt.backend,
				URL:     tt.url,
			}
			err := config.Validate()

			if tt.valid && err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected validation error, but validation passed")
			}
		})
	}
}

// TestDetectBackend tests automatic backend detection from URL
func TestDetectBackend(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"libsql URL", "libsql://shark-tasks.turso.io", "turso"},
		{"https URL", "https://shark-tasks.turso.io", "turso"},
		{"relative file path", "./shark-tasks.db", "local"},
		{"absolute file path", "/home/user/shark-tasks.db", "local"},
		{"relative path", "data/shark.db", "local"},
		{"empty string", "", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := DetectBackend(tt.url)
			if backend != tt.expected {
				t.Errorf("DetectBackend(%q) = %q; want %q", tt.url, backend, tt.expected)
			}
		})
	}
}

// TestStatusMetadata_OrchestratorAction_Load tests loading StatusMetadata with orchestrator_action
func TestStatusMetadata_OrchestratorAction_Load(t *testing.T) {
	jsonData := []byte(`{
		"color": "yellow",
		"description": "Ready for development",
		"phase": "development",
		"orchestrator_action": {
			"action": "spawn_agent",
			"agent_type": "developer",
			"skills": ["test-driven-development", "implementation"],
			"instruction_template": "Implement task {task_id}"
		}
	}`)

	var meta StatusMetadata
	err := json.Unmarshal(jsonData, &meta)
	if err != nil {
		t.Fatalf("failed to unmarshal StatusMetadata: %v", err)
	}

	if meta.OrchestratorAction == nil {
		t.Fatal("orchestrator_action should not be nil")
	}

	if meta.OrchestratorAction.Action != ActionSpawnAgent {
		t.Errorf("action = %q, want %q", meta.OrchestratorAction.Action, ActionSpawnAgent)
	}

	if meta.OrchestratorAction.AgentType != "developer" {
		t.Errorf("agent_type = %q, want %q", meta.OrchestratorAction.AgentType, "developer")
	}

	if len(meta.OrchestratorAction.Skills) != 2 {
		t.Errorf("skills length = %d, want 2", len(meta.OrchestratorAction.Skills))
	}

	if meta.OrchestratorAction.InstructionTemplate != "Implement task {task_id}" {
		t.Errorf("instruction_template mismatch")
	}
}

// TestStatusMetadata_OrchestratorAction_Backward_Compatible tests that missing orchestrator_action is valid
func TestStatusMetadata_OrchestratorAction_Backward_Compatible(t *testing.T) {
	jsonData := []byte(`{
		"color": "blue",
		"description": "In progress",
		"phase": "development"
	}`)

	var meta StatusMetadata
	err := json.Unmarshal(jsonData, &meta)
	if err != nil {
		t.Fatalf("failed to unmarshal StatusMetadata: %v", err)
	}

	if meta.OrchestratorAction != nil {
		t.Error("orchestrator_action should be nil for backward compatibility")
	}

	if meta.Color != "blue" {
		t.Errorf("color = %q, want %q", meta.Color, "blue")
	}
}

// TestStatusMetadata_OrchestratorAction_AllActionTypes tests all action types can be loaded
func TestStatusMetadata_OrchestratorAction_AllActionTypes(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   string
		wantAction string
		validate   func(*OrchestratorAction) error
	}{
		{
			name: "spawn_agent",
			jsonData: `{
				"action": "spawn_agent",
				"agent_type": "developer",
				"skills": ["implementation"],
				"instruction_template": "Implement {task_id}"
			}`,
			wantAction: ActionSpawnAgent,
			validate:   nil,
		},
		{
			name: "pause",
			jsonData: `{
				"action": "pause",
				"instruction_template": "Task {task_id} paused"
			}`,
			wantAction: ActionPause,
			validate:   nil,
		},
		{
			name: "wait_for_triage",
			jsonData: `{
				"action": "wait_for_triage",
				"instruction_template": "Task {task_id} needs triage"
			}`,
			wantAction: ActionWaitForTriage,
			validate:   nil,
		},
		{
			name: "archive",
			jsonData: `{
				"action": "archive",
				"instruction_template": "Task {task_id} archived"
			}`,
			wantAction: ActionArchive,
			validate:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action OrchestratorAction
			err := json.Unmarshal([]byte(tt.jsonData), &action)
			if err != nil {
				t.Fatalf("failed to unmarshal action: %v", err)
			}

			if action.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", action.Action, tt.wantAction)
			}

			// Validate if validator provided
			if tt.validate != nil {
				if err := tt.validate(&action); err != nil {
					t.Errorf("validation error: %v", err)
				}
			}
		})
	}
}

// TestWorkflowConfig_OrchestratorAction_Load tests loading WorkflowConfig with orchestrator_actions
func TestWorkflowConfig_OrchestratorAction_Load(t *testing.T) {
	jsonData := []byte(`{
		"status_flow": {
			"ready_for_development": ["in_development"],
			"in_development": ["ready_for_code_review"],
			"ready_for_code_review": ["completed"],
			"completed": []
		},
		"status_metadata": {
			"ready_for_development": {
				"color": "yellow",
				"phase": "development",
				"orchestrator_action": {
					"action": "spawn_agent",
					"agent_type": "developer",
					"skills": ["tdd", "implementation"],
					"instruction_template": "Implement task {task_id}"
				}
			},
			"completed": {
				"color": "green",
				"phase": "done",
				"orchestrator_action": {
					"action": "archive",
					"instruction_template": "Task {task_id} completed"
				}
			}
		}
	}`)

	var config WorkflowConfig
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("failed to unmarshal WorkflowConfig: %v", err)
	}

	// Check ready_for_development has spawn_agent
	meta, found := config.GetStatusMetadata("ready_for_development")
	if !found {
		t.Fatal("ready_for_development status not found")
	}

	if meta.OrchestratorAction == nil {
		t.Fatal("orchestrator_action should not be nil")
	}

	if meta.OrchestratorAction.Action != ActionSpawnAgent {
		t.Errorf("action = %q, want %q", meta.OrchestratorAction.Action, ActionSpawnAgent)
	}

	// Check completed has archive
	meta, found = config.GetStatusMetadata("completed")
	if !found {
		t.Fatal("completed status not found")
	}

	if meta.OrchestratorAction == nil {
		t.Fatal("orchestrator_action should not be nil for completed")
	}

	if meta.OrchestratorAction.Action != ActionArchive {
		t.Errorf("action = %q, want %q", meta.OrchestratorAction.Action, ActionArchive)
	}
}

// TestWorkflowConfig_OrchestratorAction_Missing tests that missing orchestrator_action doesn't break config loading
func TestWorkflowConfig_OrchestratorAction_Missing(t *testing.T) {
	jsonData := []byte(`{
		"status_flow": {
			"todo": ["in_progress"],
			"in_progress": ["completed"],
			"completed": []
		},
		"status_metadata": {
			"todo": {
				"color": "gray",
				"phase": "planning"
			},
			"in_progress": {
				"color": "blue",
				"phase": "development"
			}
		}
	}`)

	var config WorkflowConfig
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("failed to unmarshal WorkflowConfig: %v", err)
	}

	meta, found := config.GetStatusMetadata("todo")
	if !found {
		t.Fatal("todo status not found")
	}

	if meta.OrchestratorAction != nil {
		t.Error("orchestrator_action should be nil for backward compatibility")
	}
}

// TestOrchestratorAction_Validate_InConfig tests that invalid orchestrator_actions are caught
func TestOrchestratorAction_Validate_InConfig(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		shouldFail  bool
		failMessage string
	}{
		{
			name: "invalid action type",
			jsonData: `{
				"action": "invalid_action",
				"instruction_template": "Test"
			}`,
			shouldFail:  true,
			failMessage: "invalid action type",
		},
		{
			name: "spawn_agent without agent_type",
			jsonData: `{
				"action": "spawn_agent",
				"skills": ["implementation"],
				"instruction_template": "Implement {task_id}"
			}`,
			shouldFail:  true,
			failMessage: "agent_type",
		},
		{
			name: "spawn_agent without skills",
			jsonData: `{
				"action": "spawn_agent",
				"agent_type": "developer",
				"instruction_template": "Implement {task_id}"
			}`,
			shouldFail:  true,
			failMessage: "skills",
		},
		{
			name: "missing instruction_template",
			jsonData: `{
				"action": "pause"
			}`,
			shouldFail:  true,
			failMessage: "instruction_template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action OrchestratorAction
			err := json.Unmarshal([]byte(tt.jsonData), &action)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			validErr := action.Validate()

			if tt.shouldFail && validErr == nil {
				t.Error("expected validation error but got nil")
			}

			if !tt.shouldFail && validErr != nil {
				t.Errorf("unexpected validation error: %v", validErr)
			}

			if tt.shouldFail && validErr != nil && !containsString(validErr.Error(), tt.failMessage) {
				t.Errorf("error message should contain %q, got: %v", tt.failMessage, validErr)
			}
		})
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContainsHelper(s, substr)))
}

func stringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestConfig_IsInteractiveModeEnabled tests the IsInteractiveModeEnabled method
func TestConfig_IsInteractiveModeEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false, // Default: non-interactive
		},
		{
			name:     "config with nil InteractiveMode",
			config:   &Config{},
			expected: false, // Default: non-interactive
		},
		{
			name: "interactive mode enabled",
			config: &Config{
				InteractiveMode: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "interactive mode disabled",
			config: &Config{
				InteractiveMode: boolPtr(false),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsInteractiveModeEnabled()
			if result != tt.expected {
				t.Errorf("IsInteractiveModeEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestConfig_InteractiveMode_Marshaling tests that InteractiveMode field can be marshaled/unmarshaled
func TestConfig_InteractiveMode_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "interactive mode enabled",
			config: Config{
				InteractiveMode: boolPtr(true),
			},
			expected: `{"interactive_mode":true}`,
		},
		{
			name: "interactive mode disabled",
			config: Config{
				InteractiveMode: boolPtr(false),
			},
			expected: `{"interactive_mode":false}`,
		},
		{
			name:     "interactive mode omitted (nil)",
			config:   Config{},
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("marshaled JSON mismatch\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			// Unmarshal back
			var unmarshaled Config
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			// Verify field
			if tt.config.InteractiveMode != nil && unmarshaled.InteractiveMode == nil {
				t.Error("interactive_mode was lost during unmarshal")
				return
			}

			if tt.config.InteractiveMode != nil && unmarshaled.InteractiveMode != nil {
				if *unmarshaled.InteractiveMode != *tt.config.InteractiveMode {
					t.Errorf("interactive_mode mismatch: got %v, want %v", *unmarshaled.InteractiveMode, *tt.config.InteractiveMode)
				}
			}
		})
	}
}

// TestConfig_InteractiveMode_DefaultBehavior tests that default behavior is non-interactive
func TestConfig_InteractiveMode_DefaultBehavior(t *testing.T) {
	// Empty config (no InteractiveMode set)
	config := Config{}

	if config.IsInteractiveModeEnabled() {
		t.Error("expected default behavior to be non-interactive (false), got true")
	}

	// Config loaded from JSON without interactive_mode field
	jsonData := []byte(`{"color_enabled": true}`)
	var loaded Config
	if err := json.Unmarshal(jsonData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if loaded.IsInteractiveModeEnabled() {
		t.Error("expected non-interactive when field is missing, got true")
	}
}

// boolPtr is a helper function to create a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
