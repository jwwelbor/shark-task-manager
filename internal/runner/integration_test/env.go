// Package integration_test provides a self-contained test environment for
// end-to-end testing of the shark run loop, status transitions, and agent dispatch
// without real LLM calls or touching the production database.
//
// The environment stands up an isolated SQLite database, a temporary
// .sharkconfig.json, and mock agent dispatchers that return canned responses
// instantly, enabling deterministic, hermetic integration tests.
package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Env is a self-contained integration test environment. Each call to NewEnv
// creates an isolated temporary directory with its own SQLite database,
// .sharkconfig.json, and mock agent dispatchers.
//
// Always call Env.Cleanup() (or defer it) to remove the temporary directory.
type Env struct {
	// Dir is the root temporary directory for this environment.
	Dir string

	// DBPath is the absolute path to the isolated SQLite database.
	DBPath string

	// ConfigPath is the absolute path to the .sharkconfig.json file.
	ConfigPath string

	// DB is the open database connection. Do not close it directly; use Cleanup.
	DB *repository.DB

	// WorkflowSvc is a workflow service scoped to the environment's config.
	WorkflowSvc *workflow.Service

	// Dispatchers is the map of mock agent dispatchers. Tests may replace
	// individual entries to customise per-provider canned responses.
	Dispatchers map[string]runner.AgentDispatcher

	// t is stored for helper assertions.
	t *testing.T
}

// minimalSharkConfig is the JSON written to .sharkconfig.json inside the
// temporary directory. It uses local SQLite, disables interactive mode,
// and defines a minimal two-status workflow with a spawn_agent action so the
// run loop has something to exercise.
const minimalSharkConfig = `{
  "color_enabled": false,
  "interactive_mode": false,
  "require_rejection_reason": false
}
`

// minimalTaskWorkflowYAML is written to shark-data/workflow/task.yaml.
// It defines a tiny linear task workflow:
//
//	todo → in_progress → completed
//
// "in_progress" uses spawn_agent so the mock dispatcher is exercised.
// "completed" is a terminal status with no further transitions.
//
// Shark 2.0 loads per-entity workflows from YAML files under the directory
// pointed to by .sharkconfig.json's `workflow_config` field. Each file is the
// workflow itself — no per-entity wrapper key.
const minimalTaskWorkflowYAML = `version: "1.0"
status_flow:
  todo: ["in_progress"]
  in_progress: ["completed"]
  completed: []
status_metadata:
  todo:
    color: gray
    phase: planning
    progress_weight: 0.0
    orchestrator_action:
      action: advance_status
      instruction_template: "advance entity {{entity_key}} from todo"
  in_progress:
    color: blue
    phase: development
    progress_weight: 0.5
    orchestrator_action:
      action: spawn_agent
      agent_type: developer
      instruction_template: "implement task {{entity_key}}"
  completed:
    color: green
    phase: done
    progress_weight: 1.0
special_statuses:
  _start_: ["todo"]
  _complete_: ["completed"]
`

// NewEnv creates a new isolated test environment under t.TempDir().
// It writes .sharkconfig.json and .sharkworkflow.json, initialises a fresh
// SQLite database, and wires up mock dispatchers that succeed instantly.
//
// The returned *Env is ready to use. Call Env.Cleanup() (or defer it) to
// release resources.
func NewEnv(t *testing.T) *Env {
	t.Helper()

	dir := t.TempDir()

	configPath := filepath.Join(dir, ".sharkconfig.json")
	workflowDir := filepath.Join(dir, "shark-data", "workflow")
	taskWorkflowPath := filepath.Join(workflowDir, "task.yaml")
	dbPath := filepath.Join(dir, "shark-tasks.db")

	// Write minimal config files.
	if err := os.WriteFile(configPath, []byte(minimalSharkConfig), 0644); err != nil {
		t.Fatalf("NewEnv: write .sharkconfig.json: %v", err)
	}
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("NewEnv: mkdir shark-data/workflow: %v", err)
	}
	if err := os.WriteFile(taskWorkflowPath, []byte(minimalTaskWorkflowYAML), 0644); err != nil {
		t.Fatalf("NewEnv: write task.yaml: %v", err)
	}

	// Point config at the workflow directory (Shark 2.0 layout).
	if err := patchConfigWorkflowRef(configPath, "shark-data/workflow/"); err != nil {
		t.Fatalf("NewEnv: patch workflow_config: %v", err)
	}

	// Clear the global workflow cache so each test environment loads its own
	// isolated workflow rather than a cached version from a previous test.
	config.ClearWorkflowCache()

	// Initialise isolated SQLite database.
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("NewEnv: init db: %v", err)
	}

	wrappedDB := repository.NewDB(sqlDB)

	// Build workflow service scoped to this config.
	workflowSvc := workflow.NewService(dir)

	// Default mock dispatcher: succeeds instantly, no real subprocess.
	defaultDisp := NewMockDispatcher("default", &DispatchResult{
		ExitCode: 0,
		Stdout:   "task completed successfully (mock)",
		Stderr:   "",
	})

	dispatchers := map[string]runner.AgentDispatcher{
		"":          defaultDisp,
		"anthropic": defaultDisp,
		"codex":     NewMockDispatcher("codex", &DispatchResult{ExitCode: 0, Stdout: "codex done (mock)"}),
		"openai":    NewMockDispatcher("openai", &DispatchResult{ExitCode: 0, Stdout: "openai done (mock)"}),
	}

	return &Env{
		Dir:         dir,
		DBPath:      dbPath,
		ConfigPath:  configPath,
		DB:          wrappedDB,
		WorkflowSvc: workflowSvc,
		Dispatchers: dispatchers,
		t:           t,
	}
}

// patchConfigWorkflowRef updates the workflow_config field in the named
// .sharkconfig.json to point at relativeWorkflowPath.
func patchConfigWorkflowRef(configPath, relativeWorkflowPath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	m["workflow_config"] = relativeWorkflowPath
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

// Cleanup releases resources held by the environment. It closes the database
// connection. The temporary directory is cleaned up automatically by testing.T.
func (e *Env) Cleanup() {
	if e.DB != nil {
		_ = e.DB.Close()
	}
}

// NewEntityService returns a base EntityService backed by the environment's
// workflow configuration. It is the root service used to construct TaskService,
// EpicService, etc.
func (e *Env) NewEntityService() *services.EntityService {
	e.t.Helper()
	return services.NewEntityService(e.WorkflowSvc)
}

// NewTaskService returns a TaskService wired to the isolated database and
// workflow config. The service has no file creation side-effects (creatorSvc=nil).
func (e *Env) NewTaskService() *services.TaskService {
	e.t.Helper()
	repo := repository.NewTaskRepository(e.DB)
	entitySvc := e.NewEntityService()
	return services.NewTaskService(repo, entitySvc, nil)
}

// NewEpicService returns an EpicService wired to the isolated database.
func (e *Env) NewEpicService() *services.EpicService {
	e.t.Helper()
	repo := repository.NewEpicRepository(e.DB)
	entitySvc := e.NewEntityService()
	return services.NewEpicService(repo, entitySvc, nil, nil, nil)
}

// NewActionService returns a config.ActionService backed by the environment's
// .sharkconfig.json / .sharkworkflow.json. Returns a fatal error on failure.
func (e *Env) NewActionService() config.ActionService {
	e.t.Helper()
	svc, err := config.NewActionService(e.ConfigPath)
	if err != nil {
		e.t.Fatalf("NewActionService: %v", err)
	}
	return svc
}

// SeedTask creates an epic, feature, and task in the isolated database and
// returns the task key (e.g., "IT-E01-F01-001"). Panics on failure.
//
// This is a convenience helper: tests that need a pre-existing entity can call
// SeedTask and get a fully-wired task key ready for use with a RunController.
func (e *Env) SeedTask(ctx context.Context, epicKey, featureKey, taskKey, title, status string) {
	e.t.Helper()

	sqlDB := e.DB.DB

	// Upsert epic.
	_, err := sqlDB.ExecContext(ctx, `
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
		VALUES (?, ?, 'Integration test epic', 'active', 'medium')
	`, epicKey, "Integration Test Epic")
	if err != nil {
		e.t.Fatalf("SeedTask: insert epic %s: %v", epicKey, err)
	}

	var epicID int64
	if err := sqlDB.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = ?", epicKey).Scan(&epicID); err != nil {
		e.t.Fatalf("SeedTask: get epic id for %s: %v", epicKey, err)
	}

	// Upsert feature.
	_, err = sqlDB.ExecContext(ctx, `
		INSERT OR IGNORE INTO features (epic_id, key, title, slug, description, status)
		VALUES (?, ?, 'Integration test feature', 'integration-test-feature', 'Integration test feature', 'active')
	`, epicID, featureKey)
	if err != nil {
		e.t.Fatalf("SeedTask: insert feature %s: %v", featureKey, err)
	}

	var featureID int64
	if err := sqlDB.QueryRowContext(ctx, "SELECT id FROM features WHERE key = ?", featureKey).Scan(&featureID); err != nil {
		e.t.Fatalf("SeedTask: get feature id for %s: %v", featureKey, err)
	}

	// Insert task.
	_, err = sqlDB.ExecContext(ctx, `
		INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES (?, ?, ?, ?, 'developer', 5, '[]')
	`, featureID, taskKey, title, status)
	if err != nil {
		e.t.Fatalf("SeedTask: insert task %s: %v", taskKey, err)
	}
}

// --------------------------------------------------------
// MockDispatcher - canned response dispatcher for tests
// --------------------------------------------------------

// DispatchResult is an alias for runner.DispatchResult so callers don't need
// to import the runner package just to build a canned response.
type DispatchResult = runner.DispatchResult

// MockDispatcher is an AgentDispatcher that returns a pre-configured canned
// response without spawning any subprocess. It is safe for concurrent use.
//
// Tests can inspect DispatchCallCount and LastInput to verify the controller
// invoked the dispatcher correctly.
type MockDispatcher struct {
	// name is returned by Name().
	name string

	// result is the canned response returned by Dispatch().
	result *DispatchResult

	// err is an optional error to return instead of result.
	// If non-nil, Dispatch() returns nil, err.
	err error

	// DispatchCallCount is incremented each time Dispatch() is called.
	DispatchCallCount int

	// LastInput is the most recent DispatchInput received.
	LastInput *runner.DispatchInput

	// DispatchDelay is an optional delay to simulate latency.
	DispatchDelay time.Duration
}

// NewMockDispatcher creates a MockDispatcher that returns the given result on every Dispatch call.
// Pass result=nil to make Dispatch() return an error (via WithError).
func NewMockDispatcher(name string, result *DispatchResult) *MockDispatcher {
	return &MockDispatcher{name: name, result: result}
}

// WithError configures the dispatcher to return the given error on all Dispatch calls.
// This is useful for testing failure paths in the run controller.
func (m *MockDispatcher) WithError(err error) *MockDispatcher {
	m.err = err
	return m
}

// WithDelay adds a simulated latency to Dispatch calls.
func (m *MockDispatcher) WithDelay(d time.Duration) *MockDispatcher {
	m.DispatchDelay = d
	return m
}

// Name implements runner.AgentDispatcher.
func (m *MockDispatcher) Name() string { return m.name }

// BuildCommand implements runner.AgentDispatcher. It returns a deterministic
// command string derived from the dispatcher's name so stage.dispatch events
// carry a stable, inspectable value in tests without spawning a real subprocess.
func (m *MockDispatcher) BuildCommand(input runner.DispatchInput) (string, error) {
	return fmt.Sprintf("mock-%s <instruction>", m.name), nil
}

// Dispatch implements runner.AgentDispatcher. It stores the input for inspection,
// optionally sleeps, and returns the canned result or error.
func (m *MockDispatcher) Dispatch(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error) {
	m.DispatchCallCount++

	inputCopy := input
	m.LastInput = &inputCopy

	if m.DispatchDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.DispatchDelay):
		}
	}

	if m.err != nil {
		return nil, m.err
	}

	// Build result with Duration filled in so it matches real dispatcher shape.
	result := &runner.DispatchResult{
		ExitCode: m.result.ExitCode,
		Stdout:   m.result.Stdout,
		Stderr:   m.result.Stderr,
		Duration: m.DispatchDelay,
		Command:  fmt.Sprintf("mock-%s <instruction>", m.name),
	}
	if m.result.Duration > 0 {
		result.Duration = m.result.Duration
	}

	return result, nil
}

// Compile-time assertion: MockDispatcher must implement AgentDispatcher.
var _ runner.AgentDispatcher = (*MockDispatcher)(nil)
