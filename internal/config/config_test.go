package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseConfig_Marshaling tests that db.DatabaseConfig can be marshaled and unmarshaled
func TestDatabaseConfig_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "config with turso backend",
			config: Config{
				Database: &db.DatabaseConfig{
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
				Database: &db.DatabaseConfig{
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
			config := db.DatabaseConfig{
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

// TestDatabaseConfigValidationURL tests URL validation
func TestDatabaseConfigValidationURL(t *testing.T) {
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
			config := db.DatabaseConfig{
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

// TestDetectBackendFromURL tests automatic backend detection from URL
func TestDetectBackendFromURL(t *testing.T) {
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
			backend := db.DetectBackendFromURL(tt.url)
			if backend != tt.expected {
				t.Errorf("db.DetectBackendFromURL(%q) = %q; want %q", tt.url, backend, tt.expected)
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

// TestConfig_RequireRejectionReason_DefaultValue tests default value is false
func TestConfig_RequireRejectionReason_DefaultValue(t *testing.T) {
	cfg := &Config{}
	if cfg.RequireRejectionReason != false {
		t.Errorf("expected default RequireRejectionReason to be false, got: %v", cfg.RequireRejectionReason)
	}
}

// TestConfig_RequireRejectionReason_Parsing tests JSON unmarshaling of the field
func TestConfig_RequireRejectionReason_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    bool
		wantErr bool
	}{
		{
			name:    "explicitly true",
			json:    `{"require_rejection_reason": true}`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "explicitly false",
			json:    `{"require_rejection_reason": false}`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "omitted (default)",
			json:    `{}`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid type (string)",
			json:    `{"require_rejection_reason": "yes"}`,
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid type (number)",
			json:    `{"require_rejection_reason": 1}`,
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := json.Unmarshal([]byte(tt.json), &cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg.RequireRejectionReason != tt.want {
				t.Errorf("RequireRejectionReason = %v, want %v", cfg.RequireRejectionReason, tt.want)
			}
		})
	}
}

// TestConfig_IsRequireRejectionReasonEnabled tests the getter method
func TestConfig_IsRequireRejectionReasonEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name:   "enabled",
			config: &Config{RequireRejectionReason: true},
			want:   true,
		},
		{
			name:   "disabled",
			config: &Config{RequireRejectionReason: false},
			want:   false,
		},
		{
			name:   "nil config",
			config: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsRequireRejectionReasonEnabled()
			if result != tt.want {
				t.Errorf("IsRequireRejectionReasonEnabled() = %v, want %v", result, tt.want)
			}
		})
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

// TestConfig_IsBackwardTransition tests whether a status transition is backward
// based on ProgressWeight values in StatusMetadata (E07-F22)
func TestConfig_IsBackwardTransition(t *testing.T) {
	// Create a config with StatusMetadata that has ProgressWeight values
	cfg := &Config{}

	// Set up the test using the status flow pattern
	// (This assumes StatusMetadata can be set internally for testing)
	tests := []struct {
		name         string
		oldStatus    string
		newStatus    string
		weights      map[string]float64
		wantBackward bool
	}{
		{
			name:         "backward: high weight to low weight",
			oldStatus:    "ready_for_code_review",
			newStatus:    "in_development",
			weights:      map[string]float64{"ready_for_code_review": 0.85, "in_development": 0.50},
			wantBackward: true,
		},
		{
			name:         "backward: completed to review",
			oldStatus:    "completed",
			newStatus:    "ready_for_code_review",
			weights:      map[string]float64{"completed": 1.0, "ready_for_code_review": 0.85},
			wantBackward: true,
		},
		{
			name:         "forward: development to review",
			oldStatus:    "in_development",
			newStatus:    "ready_for_code_review",
			weights:      map[string]float64{"in_development": 0.50, "ready_for_code_review": 0.85},
			wantBackward: false,
		},
		{
			name:         "equal: same status",
			oldStatus:    "in_development",
			newStatus:    "in_development",
			weights:      map[string]float64{"in_development": 0.50},
			wantBackward: false,
		},
		{
			name:         "unknown old status",
			oldStatus:    "unknown_status",
			newStatus:    "in_development",
			weights:      map[string]float64{"in_development": 0.50},
			wantBackward: false,
		},
		{
			name:         "unknown new status",
			oldStatus:    "in_development",
			newStatus:    "unknown_status",
			weights:      map[string]float64{"in_development": 0.50},
			wantBackward: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.IsBackwardTransition(tt.oldStatus, tt.newStatus, tt.weights)
			if result != tt.wantBackward {
				t.Errorf("IsBackwardTransition(%q, %q) = %v, want %v",
					tt.oldStatus, tt.newStatus, result, tt.wantBackward)
			}
		})
	}
}

// TestConfig_GetViewer_DefaultValue tests that GetViewer returns "cat" by default
func TestConfig_GetViewer_DefaultValue(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name:   "nil config",
			config: nil,
			want:   "cat",
		},
		{
			name:   "config with nil Viewer",
			config: &Config{},
			want:   "cat",
		},
		{
			name: "config with empty Viewer",
			config: &Config{
				Viewer: stringPtr(""),
			},
			want: "cat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetViewer()
			if result != tt.want {
				t.Errorf("GetViewer() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestConfig_GetViewer_CustomValue tests that GetViewer returns custom viewer
func TestConfig_GetViewer_CustomValue(t *testing.T) {
	tests := []struct {
		name   string
		viewer string
		want   string
	}{
		{
			name:   "glow viewer",
			viewer: "glow",
			want:   "glow",
		},
		{
			name:   "nano viewer",
			viewer: "nano",
			want:   "nano",
		},
		{
			name:   "bat viewer",
			viewer: "bat",
			want:   "bat",
		},
		{
			name:   "less viewer",
			viewer: "less",
			want:   "less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Viewer: &tt.viewer,
			}
			result := config.GetViewer()
			if result != tt.want {
				t.Errorf("GetViewer() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestConfig_Viewer_Marshaling tests that Viewer field can be marshaled/unmarshaled
func TestConfig_Viewer_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "config with glow viewer",
			config: Config{
				Viewer: stringPtr("glow"),
			},
			expected: `{"viewer":"glow"}`,
		},
		{
			name: "config with cat viewer",
			config: Config{
				Viewer: stringPtr("cat"),
			},
			expected: `{"viewer":"cat"}`,
		},
		{
			name:     "config without viewer (nil)",
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
			if tt.config.Viewer != nil && unmarshaled.Viewer == nil {
				t.Error("viewer was lost during unmarshal")
				return
			}

			if tt.config.Viewer != nil && unmarshaled.Viewer != nil {
				if *unmarshaled.Viewer != *tt.config.Viewer {
					t.Errorf("viewer mismatch: got %v, want %v", *unmarshaled.Viewer, *tt.config.Viewer)
				}
			}
		})
	}
}

// stringPtr is a helper function to create a pointer to a string value
func stringPtr(s string) *string {
	return &s
}

func TestGetTemplateDirectory(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name:     "nil config returns default",
			config:   nil,
			expected: "shark-templates",
		},
		{
			name:     "nil TemplateDirectory returns default",
			config:   &Config{},
			expected: "shark-templates",
		},
		{
			name:     "empty string TemplateDirectory returns default",
			config:   &Config{TemplateDirectory: stringPtr("")},
			expected: "shark-templates",
		},
		{
			name:     "custom directory returned",
			config:   &Config{TemplateDirectory: stringPtr("my-templates")},
			expected: "my-templates",
		},
		{
			name:     "path with subdirectory returned",
			config:   &Config{TemplateDirectory: stringPtr("custom/templates")},
			expected: "custom/templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTemplateDirectory()
			if result != tt.expected {
				t.Errorf("GetTemplateDirectory() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetTemplateDirectoryFromConfig(t *testing.T) {
	t.Run("empty config path returns default", func(t *testing.T) {
		result := GetTemplateDirectoryFromConfig("")
		if result != "shark-templates" {
			t.Errorf("GetTemplateDirectoryFromConfig(\"\") = %q, want %q", result, "shark-templates")
		}
	})

	t.Run("nonexistent config file returns default", func(t *testing.T) {
		result := GetTemplateDirectoryFromConfig("/nonexistent/path/.sharkconfig.json")
		if result != "shark-templates" {
			t.Errorf("GetTemplateDirectoryFromConfig() = %q, want %q", result, "shark-templates")
		}
	})
}

// TestConfig_Maintainer_RoundTrip tests that Config with a populated Maintainer field
// survives a JSON marshal/unmarshal round-trip with all fields preserved
// (test-plan.md §2.5, AC-T3, AC-T4, AC-T5).
func TestConfig_Maintainer_RoundTrip(t *testing.T) {
	t.Run("config with maintainer preserves all fields", func(t *testing.T) {
		// Arrange
		workflowConfig := "shark-templates/.sharkworkflow.json"
		original := Config{
			WorkflowConfig: &workflowConfig,
			Maintainer: &MaintainerConfig{
				PasswordHash:       "abc123deadbeef",
				CacheWindowSeconds: 120,
			},
		}

		// Act: Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		// Assert: maintainer key is present
		jsonStr := string(data)
		if !containsString(jsonStr, `"maintainer"`) {
			t.Errorf("marshaled JSON missing 'maintainer' key: %s", jsonStr)
		}
		if !containsString(jsonStr, `"password_hash"`) {
			t.Errorf("marshaled JSON missing 'password_hash' key: %s", jsonStr)
		}
		if !containsString(jsonStr, `"cache_window_seconds"`) {
			t.Errorf("marshaled JSON missing 'cache_window_seconds' key: %s", jsonStr)
		}
		if !containsString(jsonStr, `"workflow_config"`) {
			t.Errorf("marshaled JSON missing 'workflow_config' key: %s", jsonStr)
		}

		// Act: Unmarshal back
		var got Config
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}

		// Assert: Maintainer fields preserved
		if got.Maintainer == nil {
			t.Fatal("Maintainer is nil after round-trip")
		}
		if got.Maintainer.PasswordHash != original.Maintainer.PasswordHash {
			t.Errorf("PasswordHash = %q, want %q", got.Maintainer.PasswordHash, original.Maintainer.PasswordHash)
		}
		if got.Maintainer.CacheWindowSeconds != original.Maintainer.CacheWindowSeconds {
			t.Errorf("CacheWindowSeconds = %d, want %d", got.Maintainer.CacheWindowSeconds, original.Maintainer.CacheWindowSeconds)
		}

		// Assert: other top-level key preserved
		if got.WorkflowConfig == nil || *got.WorkflowConfig != workflowConfig {
			t.Errorf("WorkflowConfig = %v, want %q", got.WorkflowConfig, workflowConfig)
		}
	})

	t.Run("config without maintainer omits maintainer key (omitempty)", func(t *testing.T) {
		// Arrange: Config with no Maintainer field
		original := Config{}

		// Act: Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		// Assert: no "maintainer" key in JSON output
		jsonStr := string(data)
		if containsString(jsonStr, `"maintainer"`) {
			t.Errorf("marshaled JSON unexpectedly contains 'maintainer' key: %s", jsonStr)
		}
	})

	t.Run("maintainer with zero CacheWindowSeconds omits that field (omitempty)", func(t *testing.T) {
		// Arrange
		original := Config{
			Maintainer: &MaintainerConfig{
				PasswordHash:       "abc",
				CacheWindowSeconds: 0, // zero value, should be omitted
			},
		}

		// Act
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		// Assert: cache_window_seconds omitted when zero
		jsonStr := string(data)
		if containsString(jsonStr, `"cache_window_seconds"`) {
			t.Errorf("marshaled JSON unexpectedly contains 'cache_window_seconds' for zero value: %s", jsonStr)
		}
		// password_hash should still be present
		if !containsString(jsonStr, `"password_hash"`) {
			t.Errorf("marshaled JSON missing 'password_hash': %s", jsonStr)
		}
	})
}

// TestConfig_TagRequiredFor_RoundTrip tests that Config with a populated
// TagRequiredForTypes field survives a JSON marshal/unmarshal round-trip with
// the slice preserved exactly (length, order, values).
// Reference: E28-F04 spec.md §2.3, test-plan.md §1.4 AC-27.
func TestConfig_TagRequiredFor_RoundTrip(t *testing.T) {
	// Arrange
	original := Config{
		TagRequiredForTypes: []string{"task", "bug"},
	}

	// Act: Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Assert: tag_required_for key is present with expected values
	jsonStr := string(data)
	if !containsString(jsonStr, `"tag_required_for"`) {
		t.Errorf("marshaled JSON missing 'tag_required_for' key: %s", jsonStr)
	}
	if !containsString(jsonStr, `"task"`) {
		t.Errorf("marshaled JSON missing 'task' value: %s", jsonStr)
	}
	if !containsString(jsonStr, `"bug"`) {
		t.Errorf("marshaled JSON missing 'bug' value: %s", jsonStr)
	}

	// Act: Unmarshal back
	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Assert: slice preserved exactly (length, order, values)
	if len(got.TagRequiredForTypes) != len(original.TagRequiredForTypes) {
		t.Fatalf("TagRequiredForTypes length = %d, want %d",
			len(got.TagRequiredForTypes), len(original.TagRequiredForTypes))
	}
	for i, v := range original.TagRequiredForTypes {
		if got.TagRequiredForTypes[i] != v {
			t.Errorf("TagRequiredForTypes[%d] = %q, want %q",
				i, got.TagRequiredForTypes[i], v)
		}
	}

	// Act: Re-marshal after round-trip
	data2, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() second pass error = %v", err)
	}

	// Assert: re-marshal produces the same JSON
	if string(data2) != string(data) {
		t.Errorf("re-marshal produced different JSON:\n got: %s\nwant: %s",
			string(data2), string(data))
	}
}

// TestConfig_TagRequiredFor_AbsentFieldIsNilSlice verifies that JSON without
// a tag_required_for key unmarshals to a nil slice (not an empty slice) and
// that cfg.TagRequiredFor() returns nil in that case. Ensures omitempty
// correctness and nil-safe method behavior.
// Reference: E28-F04 test-plan.md §1.4 AC-27b.
func TestConfig_TagRequiredFor_AbsentFieldIsNilSlice(t *testing.T) {
	// Arrange: JSON with no tag_required_for key
	raw := []byte(`{}`)

	// Act
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Assert: underlying field is nil
	if cfg.TagRequiredForTypes != nil {
		t.Errorf("TagRequiredForTypes = %v, want nil", cfg.TagRequiredForTypes)
	}

	// Assert: accessor returns nil (not []string{})
	got := cfg.TagRequiredFor()
	if got != nil {
		t.Errorf("cfg.TagRequiredFor() = %v, want nil", got)
	}

	// Assert: absent field is omitted on re-marshal (omitempty)
	out, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if containsString(string(out), `"tag_required_for"`) {
		t.Errorf("marshaled JSON unexpectedly contains 'tag_required_for' for nil slice: %s",
			string(out))
	}
}

// TestConfig_TagRequiredFor_NilReceiver verifies that calling TagRequiredFor
// on a nil *Config receiver returns nil without panicking. Required by
// REQ-F-007 / spec.md §2.3 nil-safe semantics.
// Reference: E28-F04 test-plan.md §1.4 AC-27c, task AC-T1.
func TestConfig_TagRequiredFor_NilReceiver(t *testing.T) {
	var c *Config
	got := c.TagRequiredFor()
	if got != nil {
		t.Errorf("(*Config)(nil).TagRequiredFor() = %v, want nil", got)
	}
}

// TestConfig_TagRequiredFor_DefensiveCopy verifies that the slice returned by
// TagRequiredFor is a defensive copy: callers mutating the returned slice
// must not affect subsequent calls. Required by spec.md §2.3.
// Reference: E28-F04 test-plan.md §1.4 AC-27d, task AC-T3.
func TestConfig_TagRequiredFor_DefensiveCopy(t *testing.T) {
	cfg := &Config{
		TagRequiredForTypes: []string{"task", "bug"},
	}

	// Act: take a copy and mutate it
	first := cfg.TagRequiredFor()
	if len(first) != 2 {
		t.Fatalf("first call returned %d entries, want 2", len(first))
	}
	first[0] = "mutated"

	// Assert: a subsequent call still returns the original values
	second := cfg.TagRequiredFor()
	if len(second) != 2 {
		t.Fatalf("second call returned %d entries, want 2", len(second))
	}
	if second[0] != "task" || second[1] != "bug" {
		t.Errorf("second call = %v, want [task bug]; caller mutation leaked into backing field",
			second)
	}

	// Sanity: the backing field itself is unchanged
	if cfg.TagRequiredForTypes[0] != "task" {
		t.Errorf("TagRequiredForTypes[0] = %q, want \"task\"; backing field was mutated",
			cfg.TagRequiredForTypes[0])
	}
}

// --- E07-F17-001: RecentConfig and GetRecentDefaultLimit ---

// TestGetRecentDefaultLimit_NilConfig verifies that calling GetRecentDefaultLimit
// on a nil *Config receiver returns 5 without panicking (AC-T1).
func TestGetRecentDefaultLimit_NilConfig(t *testing.T) {
	var cfg *Config
	got := cfg.GetRecentDefaultLimit()
	if got != 5 {
		t.Errorf("GetRecentDefaultLimit() on nil receiver = %d, want 5", got)
	}
}

// TestGetRecentDefaultLimit_SectionAbsent verifies that when the "recent" section
// is absent from the config JSON, the default of 5 is returned (AC-T2).
func TestGetRecentDefaultLimit_SectionAbsent(t *testing.T) {
	configJSON := `{"color_enabled": true}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	// Recent pointer should be nil
	if cfg.Recent != nil {
		t.Errorf("expected Recent to be nil when section absent, got %+v", cfg.Recent)
	}

	got := cfg.GetRecentDefaultLimit()
	if got != 5 {
		t.Errorf("GetRecentDefaultLimit() with absent section = %d, want 5", got)
	}
}

// TestGetRecentDefaultLimit_FieldZero verifies that when recent.default_limit is 0,
// the default of 5 is returned (AC-T3).
func TestGetRecentDefaultLimit_FieldZero(t *testing.T) {
	configJSON := `{"recent": {"default_limit": 0}}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	got := cfg.GetRecentDefaultLimit()
	if got != 5 {
		t.Errorf("GetRecentDefaultLimit() with default_limit=0 = %d, want 5", got)
	}
}

// TestGetRecentDefaultLimit_FieldNegative verifies that when recent.default_limit
// is negative, the default of 5 is returned (AC-T3).
func TestGetRecentDefaultLimit_FieldNegative(t *testing.T) {
	configJSON := `{"recent": {"default_limit": -3}}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	got := cfg.GetRecentDefaultLimit()
	if got != 5 {
		t.Errorf("GetRecentDefaultLimit() with default_limit=-3 = %d, want 5", got)
	}
}

// TestGetRecentDefaultLimit_FieldPositive verifies that when recent.default_limit
// is a positive value, that configured value is returned (AC-T4).
func TestGetRecentDefaultLimit_FieldPositive(t *testing.T) {
	configJSON := `{"recent": {"default_limit": 7}}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	got := cfg.GetRecentDefaultLimit()
	if got != 7 {
		t.Errorf("GetRecentDefaultLimit() with default_limit=7 = %d, want 7", got)
	}
}

// --- E19-F05-001: SprintDefaultsConfig struct and sprint_defaults parsing ---

// TC-015-05: carryover_behavior and auto_create parsed from config.
// Verifies that when sprint_defaults section is present, all fields are parsed
// correctly. Production entrypoint: config.Manager.Load() reads real file.
func TestSprintDefaults_ParsedFromConfig(t *testing.T) {
	configJSON := `{
		"sprint_defaults": {
			"carryover_behavior": "next",
			"auto_create": false,
			"capacity": {}
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.SprintDefaults, "SprintDefaults must not be nil when section present")
	assert.Equal(t, "next", cfg.SprintDefaults.CarryoverBehavior)
	assert.False(t, cfg.SprintDefaults.AutoCreate)
}

// TC-015-05 (capacity map): capacity values in sprint_defaults.capacity parsed correctly.
func TestSprintDefaults_CapacityParsed(t *testing.T) {
	configJSON := `{
		"sprint_defaults": {
			"capacity": {"backend": 21, "frontend": 13},
			"carryover_behavior": "backlog",
			"auto_create": true
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.SprintDefaults)
	assert.Equal(t, float64(21), cfg.SprintDefaults.Capacity["backend"])
	assert.Equal(t, float64(13), cfg.SprintDefaults.Capacity["frontend"])
	assert.Equal(t, "backlog", cfg.SprintDefaults.CarryoverBehavior)
	assert.True(t, cfg.SprintDefaults.AutoCreate)
}

// TC-015-06: sprint_defaults absent — graceful defaults (nil pointer, no panic).
// Production entrypoint: config.Manager.Load() reads real file.
func TestSprintDefaults_AbsentSectionIsNil(t *testing.T) {
	configJSON := `{"color_enabled": true}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	// SprintDefaults must be nil (not present in config)
	assert.Nil(t, cfg.SprintDefaults, "SprintDefaults must be nil when section absent")
}

// TC-015-06: nil SprintDefaults causes no panic when accessed.
func TestSprintDefaults_NilSafe(t *testing.T) {
	cfg := &Config{} // SprintDefaults is nil
	// Accessing SprintDefaults.Capacity on nil pointer must not panic
	assert.Nil(t, cfg.SprintDefaults)
}

// TestRecentConfig_BackwardCompat_ExistingConfigLoadsOK verifies that an existing
// .sharkconfig.json without a "recent" key loads without error (AC-T5 / REQ-F-011).
func TestRecentConfig_BackwardCompat_ExistingConfigLoadsOK(t *testing.T) {
	// Simulate a realistic existing config without the recent section
	configJSON := `{
		"color_enabled": true,
		"require_rejection_reason": false,
		"template_directory": "shark-templates",
		"workflow_config": "shark-templates/.sharkworkflow-short.json"
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err, "loading config without 'recent' section must not return an error")

	// The Recent field should be nil (not configured)
	if cfg.Recent != nil {
		t.Errorf("expected Recent to be nil for existing config without 'recent' key, got %+v", cfg.Recent)
	}

	// GetRecentDefaultLimit falls back to built-in default of 5
	got := cfg.GetRecentDefaultLimit()
	if got != 5 {
		t.Errorf("GetRecentDefaultLimit() for existing config = %d, want 5", got)
	}

	// Other existing fields are preserved
	require.NotNil(t, cfg.ColorEnabled)
	assert.True(t, *cfg.ColorEnabled)
	require.NotNil(t, cfg.TemplateDirectory)
	assert.Equal(t, "shark-templates", *cfg.TemplateDirectory)
}
