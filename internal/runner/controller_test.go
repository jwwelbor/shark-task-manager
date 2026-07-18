package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction assertions (TC-F02-001-5, TC-F02-002-1,
// TC-F02-003-1).
// If any mock fails to satisfy its interface, this file will not compile.
// ---------------------------------------------------------------------------

var _ EntityTransitioner = (*MockTransitioner)(nil)
var _ PlaceholderGenerator = (*MockPlaceholderGen)(nil)
var _ config.ActionService = (*MockActionService)(nil)
var _ AgentDispatcher = (*MockDispatcher)(nil)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

// MockTransitioner implements EntityTransitioner for testing.
type MockTransitioner struct {
	TransitionStatusFunc func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

func (m *MockTransitioner) TransitionStatus(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if m.TransitionStatusFunc != nil {
		return m.TransitionStatusFunc(ctx, key, target, opts)
	}
	return nil, fmt.Errorf("MockTransitioner.TransitionStatus not implemented")
}

func (m *MockTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	if m.GetNextStatusFunc != nil {
		return m.GetNextStatusFunc(ctx, key)
	}
	return nil, fmt.Errorf("MockTransitioner.GetNextStatus not implemented")
}

func TestGuardedTransitionOptions_BindsRunLeaseAndSourceStatus(t *testing.T) {
	next := &services.NextStatusInfo{Outcomes: map[string]string{"blocked": "waiting", "pass": "completed"}}
	opts := guardedTransitionOptions(RunOptions{SessionID: "lease-123"}, "in_progress", "completed", next)

	if !opts.GuardAdvance {
		t.Fatal("runner transitions must opt into advance-guard enforcement")
	}
	if opts.SessionID != "lease-123" || opts.FromStatus != "in_progress" || opts.Outcome != "pass" {
		t.Fatalf("guarded options = %+v, want lease, source status, and resolved outcome", opts)
	}
}

// MockPlaceholderGen implements PlaceholderGenerator for testing.
type MockPlaceholderGen struct {
	GenerateFunc func(ctx context.Context, key string) (map[string]string, error)
}

func (m *MockPlaceholderGen) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, key)
	}
	return map[string]string{}, nil
}

// MockActionService implements config.ActionService for testing.
type MockActionService struct {
	GetStatusActionFunc          func(ctx context.Context, status string) (*config.OrchestratorAction, error)
	GetStatusActionPopulatedFunc func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error)
	GetAllActionsFunc            func(ctx context.Context) (map[string]*config.OrchestratorAction, error)
	ValidateActionsFunc          func(ctx context.Context) (*config.ValidationResult, error)
	ReloadFunc                   func(ctx context.Context) error
}

func (m *MockActionService) GetStatusAction(ctx context.Context, status string) (*config.OrchestratorAction, error) {
	if m.GetStatusActionFunc != nil {
		return m.GetStatusActionFunc(ctx, status)
	}
	return nil, nil
}

func (m *MockActionService) GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
	if m.GetStatusActionPopulatedFunc != nil {
		return m.GetStatusActionPopulatedFunc(ctx, status, vars)
	}
	return nil, nil
}

func (m *MockActionService) GetAllActions(ctx context.Context) (map[string]*config.OrchestratorAction, error) {
	if m.GetAllActionsFunc != nil {
		return m.GetAllActionsFunc(ctx)
	}
	return map[string]*config.OrchestratorAction{}, nil
}

func (m *MockActionService) ValidateActions(ctx context.Context) (*config.ValidationResult, error) {
	if m.ValidateActionsFunc != nil {
		return m.ValidateActionsFunc(ctx)
	}
	return &config.ValidationResult{Valid: true}, nil
}

func (m *MockActionService) Reload(ctx context.Context) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(ctx)
	}
	return nil
}

// ForEntity returns the mock itself; tests that don't care about per-entity
// scoping see identical behavior, while tests that do can wrap a new mock
// per entity and inject it via the parent's GetStatusActionPopulatedFunc.
func (m *MockActionService) ForEntity(entityType string) config.ActionService {
	return m
}

// MockDispatcher implements AgentDispatcher for testing.
type MockDispatcher struct {
	DispatchFunc     func(ctx context.Context, input DispatchInput) (*DispatchResult, error)
	NameFunc         func() string
	BuildCommandFunc func(input DispatchInput) (string, error)
}

func (m *MockDispatcher) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	if m.DispatchFunc != nil {
		return m.DispatchFunc(ctx, input)
	}
	return &DispatchResult{ExitCode: 0}, nil
}

func (m *MockDispatcher) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

func (m *MockDispatcher) BuildCommand(input DispatchInput) (string, error) {
	if m.BuildCommandFunc != nil {
		return m.BuildCommandFunc(input)
	}
	return "mock-cmd", nil
}

// ---------------------------------------------------------------------------
// Helper: build a real workflow.Service for tests.
// NewService("") uses the default workflow config (completed/cancelled terminal).
// ---------------------------------------------------------------------------

func defaultWorkflowSvc() *workflow.Service {
	return workflow.NewService("")
}

// ---------------------------------------------------------------------------
// TC-F02-001: RunController struct and constructor injection
// ---------------------------------------------------------------------------

// TestNewRunController_ValidDeps verifies that a controller is created successfully
// when all required dependencies are provided.
func TestNewRunController_ValidDeps(t *testing.T) {
	transitioner := &MockTransitioner{}
	actionSvc := &MockActionService{}
	dispatchers := map[string]AgentDispatcher{"": &MockDispatcher{}}

	deps := RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
	}

	ctrl, err := NewRunController(deps)
	if err != nil {
		t.Fatalf("NewRunController() returned unexpected error: %v", err)
	}
	if ctrl == nil {
		t.Fatal("NewRunController() returned nil controller")
	}
}

// TestNewRunController_NilTransitioner verifies that a nil Transitioner causes panic.
func TestNewRunController_NilTransitioner(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil Transitioner, got none")
		}
	}()

	actionSvc := &MockActionService{}
	dispatchers := map[string]AgentDispatcher{"": &MockDispatcher{}}

	deps := RunControllerDeps{
		Transitioner: nil, // nil — should panic
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
	}

	_, _ = NewRunController(deps) //nolint:errcheck
}

// TestNewRunController_NilActionSvc verifies that a nil ActionSvc causes panic.
func TestNewRunController_NilActionSvc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil ActionSvc, got none")
		}
	}()

	transitioner := &MockTransitioner{}
	dispatchers := map[string]AgentDispatcher{"": &MockDispatcher{}}

	deps := RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    nil, // nil — should panic
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
	}

	_, _ = NewRunController(deps) //nolint:errcheck
}

// TestNewRunController_NilWorkflowSvc verifies that a nil WorkflowSvc causes panic.
func TestNewRunController_NilWorkflowSvc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil WorkflowSvc, got none")
		}
	}()

	transitioner := &MockTransitioner{}
	actionSvc := &MockActionService{}
	dispatchers := map[string]AgentDispatcher{"": &MockDispatcher{}}

	deps := RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  nil, // nil — should panic
		Dispatchers:  dispatchers,
	}

	_, _ = NewRunController(deps) //nolint:errcheck
}

// TestNewRunController_EmptyDispatchers verifies that an empty dispatchers map returns error.
func TestNewRunController_EmptyDispatchers(t *testing.T) {
	transitioner := &MockTransitioner{}
	actionSvc := &MockActionService{}

	deps := RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  map[string]AgentDispatcher{}, // empty
	}

	_, err := NewRunController(deps)
	if err == nil {
		t.Error("expected error for empty Dispatchers map, got nil")
	}
}

// ---------------------------------------------------------------------------
// TC-F02-002: EntityTransitioner interface shape
// ---------------------------------------------------------------------------

// TestEntityTransitioner_TransitionStatusSignature verifies that TransitionStatus
// can be called via the EntityTransitioner interface and returns the right types.
func TestEntityTransitioner_TransitionStatusSignature(t *testing.T) {
	var transitioner EntityTransitioner = &MockTransitioner{
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}

	result, err := transitioner.TransitionStatus(context.Background(), "E07-F01-001", "in_progress", services.TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("TransitionStatus() returned nil result")
	}
	if result.ToStatus != "in_progress" {
		t.Errorf("expected ToStatus=in_progress, got %s", result.ToStatus)
	}
}

// TestEntityTransitioner_GetNextStatusSignature verifies that GetNextStatus
// can be called via the EntityTransitioner interface and returns the right types.
func TestEntityTransitioner_GetNextStatusSignature(t *testing.T) {
	var transitioner EntityTransitioner = &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
	}

	info, err := transitioner.GetNextStatus(context.Background(), "E07-F01-001")
	if err != nil {
		t.Fatalf("GetNextStatus() returned error: %v", err)
	}
	if info == nil {
		t.Fatal("GetNextStatus() returned nil")
	}
	if info.CurrentStatus != "in_development" {
		t.Errorf("expected CurrentStatus=in_development, got %s", info.CurrentStatus)
	}
}

// ---------------------------------------------------------------------------
// TC-F02-003: PlaceholderGenerator interface shape
// ---------------------------------------------------------------------------

// TestPlaceholderGenerator_ReturnsMap verifies GeneratePlaceholders returns a non-nil map.
func TestPlaceholderGenerator_ReturnsMap(t *testing.T) {
	var gen PlaceholderGenerator = &MockPlaceholderGen{
		GenerateFunc: func(ctx context.Context, key string) (map[string]string, error) {
			return map[string]string{"id": key, "title": "Test Task"}, nil
		},
	}

	vars, err := gen.GeneratePlaceholders(context.Background(), "E07-F01-001")
	if err != nil {
		t.Fatalf("GeneratePlaceholders() returned error: %v", err)
	}
	if vars == nil {
		t.Fatal("GeneratePlaceholders() returned nil map")
	}
	if vars["id"] != "E07-F01-001" {
		t.Errorf("expected id=E07-F01-001, got %s", vars["id"])
	}
}

// TestPlaceholderGenerator_PropagatesError verifies that GeneratePlaceholders
// propagates errors from the underlying generator.
func TestPlaceholderGenerator_PropagatesError(t *testing.T) {
	expectedErr := errors.New("entity not found")
	var gen PlaceholderGenerator = &MockPlaceholderGen{
		GenerateFunc: func(ctx context.Context, key string) (map[string]string, error) {
			return nil, expectedErr
		},
	}

	vars, err := gen.GeneratePlaceholders(context.Background(), "MISSING")
	if err == nil {
		t.Fatal("expected error from GeneratePlaceholders, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if vars != nil {
		t.Error("expected nil vars on error")
	}
}

// ---------------------------------------------------------------------------
// RunController Run() integration tests with mocked dependencies
// ---------------------------------------------------------------------------

// makeController is a test helper that creates a RunController with sensible defaults.
func makeController(t *testing.T,
	transitioner EntityTransitioner,
	actionSvc config.ActionService,
	dispatchers map[string]AgentDispatcher,
) *RunController {
	t.Helper()
	if dispatchers == nil {
		dispatchers = map[string]AgentDispatcher{"": &MockDispatcher{}}
	}
	deps := RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
	}
	ctrl, err := NewRunController(deps)
	if err != nil {
		t.Fatalf("makeController: NewRunController failed: %v", err)
	}
	return ctrl
}

// TestRunController_AlreadyTerminal verifies that when the entity is already
// in a terminal status, Run() returns immediately with outcome "already_terminal"
// and zero stages.
func TestRunController_AlreadyTerminal(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "completed",
				IsTerminal:    true,
			}, nil
		},
	}
	actionSvc := &MockActionService{}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "already_terminal" {
		t.Errorf("expected Outcome=already_terminal, got %s", result.Outcome)
	}
	if result.StagesCompleted != 0 {
		t.Errorf("expected StagesCompleted=0, got %d", result.StagesCompleted)
	}
	if result.FinalStatus != "completed" {
		t.Errorf("expected FinalStatus=completed, got %s", result.FinalStatus)
	}
}

// TestRunController_NoActionForStatus verifies that when no action is configured
// for the current status, Run() returns with outcome "no_action".
func TestRunController_NoActionForStatus(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return nil, nil // no action configured
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "no_action" {
		t.Errorf("expected Outcome=no_action, got %s", result.Outcome)
	}
}

// TestRunController_PauseAction verifies that a "pause" action type stops the loop
// and returns Outcome="paused".
func TestRunController_PauseAction(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "ready_for_approval",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionPause,
				Instruction: "Awaiting approval",
			}, nil
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "paused" {
		t.Errorf("expected Outcome=paused, got %s", result.Outcome)
	}
	if result.FinalStatus != "ready_for_approval" {
		t.Errorf("expected FinalStatus=ready_for_approval, got %s", result.FinalStatus)
	}
}

// TestRunController_WaitForTriageAction verifies wait_for_triage behaves like pause.
func TestRunController_WaitForTriageAction(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_triage",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionWaitForTriage,
				Instruction: "Waiting for triage",
			}, nil
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "paused" {
		t.Errorf("expected Outcome=paused, got %s", result.Outcome)
	}
}

// TestRunController_DispatcherSelection_DefaultProvider verifies that an empty
// provider selects the "" (default) dispatcher.
func TestRunController_DispatcherSelection_DefaultProvider(t *testing.T) {
	defaultDispatched := false

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "", // empty provider -> default dispatcher
				Instruction: "Do the work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				defaultDispatched = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if !defaultDispatched {
		t.Error("expected default dispatcher to be invoked, but it was not")
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed, got %s", result.Outcome)
	}
}

func TestRunController_SpawnAgentUsesRecommendedOutcome(t *testing.T) {
	getNextCalls := 0
	var transitionedTo string
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			getNextCalls++
			if getNextCalls > 2 {
				return &services.NextStatusInfo{CurrentStatus: "completed", IsTerminal: true}, nil
			}
			return &services.NextStatusInfo{
				CurrentStatus: "research",
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "specification"}},
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "task_generation"}},
				},
				Outcomes: map[string]string{"pass": "specification", "simple": "task_generation"},
			}, nil
		},
		TransitionStatusFunc: func(_ context.Context, _ string, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
			transitionedTo = target
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, Provider: "anthropic", Instruction: "research"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: "Completed research.\nRECOMMENDED OUTCOME: simple"}, nil
		}},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01", RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("Run() outcome = %q, want completed", result.Outcome)
	}
	if transitionedTo != "task_generation" {
		t.Fatalf("TransitionStatus() target = %q, want task_generation", transitionedTo)
	}
}

// TestRunController_SpawnAgentDispatchesAssembledPrompt verifies the run loop
// does not dispatch PopulatedAction.Instruction directly when a prompt
// assembler is configured. The CLI injects the same final assembly helper used
// by `shark next`, so this is the controller-level guard against regressing to
// raw orchestrator_action prompt dispatch.
func TestRunController_SpawnAgentDispatchesAssembledPrompt(t *testing.T) {
	var dispatched DispatchInput
	var assemblerInput PromptAssemblyInput
	getNextCalls := 0

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			getNextCalls++
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	placeholders := &MockPlaceholderGen{
		GenerateFunc: func(ctx context.Context, key string) (map[string]string, error) {
			return map[string]string{"task_key": key}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			if vars["task_id"] != "E07-F01-001" {
				t.Fatalf("expected augmented task_id alias before action rendering, got %q", vars["task_id"])
			}
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Model:       "sonnet",
				Instruction: "RAW ORCHESTRATOR INSTRUCTION",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatched = input
				return &DispatchResult{ExitCode: 0, Command: "mock-cmd"}, nil
			},
		},
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: placeholders,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
		PromptAssembler: PromptAssemblerFunc(func(ctx context.Context, input PromptAssemblyInput) (string, error) {
			assemblerInput = input
			return "SELF-CONTAINED PROMPT\n\n" + input.Instruction, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewRunController() returned unexpected error: %v", err)
	}

	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed, got %s", result.Outcome)
	}
	if getNextCalls < 2 {
		t.Errorf("expected at least two GetNextStatus calls, got %d", getNextCalls)
	}
	if assemblerInput.Instruction != "RAW ORCHESTRATOR INSTRUCTION" {
		t.Errorf("assembler saw instruction %q", assemblerInput.Instruction)
	}
	if assemblerInput.AgentType != "developer" {
		t.Errorf("assembler saw agent type %q", assemblerInput.AgentType)
	}
	if dispatched.Instruction == "RAW ORCHESTRATOR INSTRUCTION" {
		t.Fatal("dispatch received raw PopulatedAction.Instruction; expected assembled prompt")
	}
	if dispatched.Instruction != "SELF-CONTAINED PROMPT\n\nRAW ORCHESTRATOR INSTRUCTION" {
		t.Errorf("dispatch instruction mismatch: %q", dispatched.Instruction)
	}
}

// TestRunController_DispatcherSelection_UnknownProvider verifies that an unknown
// provider returns a descriptive error.
func TestRunController_DispatcherSelection_UnknownProvider(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "unknown_provider",
				Instruction: "Do stuff",
			}, nil
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("expected Outcome=failed, got %s", result.Outcome)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error in RunResult for unknown provider")
	}
}

// TestRunController_ContextCancellation verifies that context cancellation
// stops the loop and returns Outcome="failed" with the cancellation error.
func TestRunController_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				Instruction: "work",
			}, nil
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, nil)
	result, err := ctrl.Run(ctx, "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("expected Outcome=failed on cancellation, got %s", result.Outcome)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for cancelled context")
	}
}

// TestRunController_AgentFailure verifies that a non-zero exit code from the
// dispatcher stops the loop and returns Outcome="failed".
func TestRunController_AgentFailure(t *testing.T) {
	transitionCalled := false

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			transitionCalled = true
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "do work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 1, Stderr: "agent error"}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("expected Outcome=failed, got %s", result.Outcome)
	}
	if transitionCalled {
		t.Error("expected no status transition on agent failure, but TransitionStatus was called")
	}
}

// TestRunController_DispatchInputConstruction verifies that DispatchInput is
// populated correctly from PopulatedAction and entity context.
func TestRunController_DispatchInputConstruction(t *testing.T) {
	var capturedInput DispatchInput

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Model:       "claude-opus-4",
				Instruction: "Implement the feature",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				capturedInput = input
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	opts := RunOptions{WorkingDir: "/tmp/myproject"}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if capturedInput.Instruction != "Implement the feature" {
		t.Errorf("Instruction: expected 'Implement the feature', got %q", capturedInput.Instruction)
	}
	if capturedInput.EntityKey != "E07-F01-001" {
		t.Errorf("EntityKey: expected E07-F01-001, got %q", capturedInput.EntityKey)
	}
	if capturedInput.Status != "in_development" {
		t.Errorf("Status: expected in_development, got %q", capturedInput.Status)
	}
	if capturedInput.AgentType != "developer" {
		t.Errorf("AgentType: expected developer, got %q", capturedInput.AgentType)
	}
	if capturedInput.Model != "claude-opus-4" {
		t.Errorf("Model: expected claude-opus-4, got %q", capturedInput.Model)
	}
	if capturedInput.WorkingDir != "/tmp/myproject" {
		t.Errorf("WorkingDir: expected /tmp/myproject, got %q", capturedInput.WorkingDir)
	}
}

// ---------------------------------------------------------------------------
// Additional happy-path and advanced scenario tests
// ---------------------------------------------------------------------------

// TestRunController_HappyPath_SingleStage verifies a single-stage run where the
// entity starts non-terminal, an agent is dispatched successfully, the status is
// advanced to a terminal state, and the run completes with outcome "completed".
func TestRunController_HappyPath_SingleStage(t *testing.T) {
	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				// First call: non-terminal, so the loop runs
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			}
			// After transition: terminal
			return &services.NextStatusInfo{
				CurrentStatus: "completed",
				IsTerminal:    true,
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "implement the task",
			}, nil
		},
	}
	dispatched := false
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatched = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed, got %s", result.Outcome)
	}
	if result.StagesCompleted != 1 {
		t.Errorf("expected StagesCompleted=1, got %d", result.StagesCompleted)
	}
	if !dispatched {
		t.Error("expected dispatcher to be called, but it was not")
	}
}

// TestRunController_HappyPath_MultiStage verifies a multi-stage run where the entity
// transitions through two non-terminal states before reaching a terminal state.
//
// Controller flow for spawn_agent:
//   - GetNextStatus called once before loop (initial status).
//   - Per iteration: dispatch agent → GetNextStatus (post-dispatch) → TransitionStatus → check terminal.
//
// To get 2 stages we need: initial status A (non-terminal) → post-dispatch1 status B with
// transition to C (non-terminal) → TransitionStatus(C) → loop again → dispatch → post-dispatch2
// → no transitions → completed.
func TestRunController_HappyPath_MultiStage(t *testing.T) {
	// GetNextStatus call sequence:
	//   call 0 (pre-loop): in_development, non-terminal
	//   call 1 (post-dispatch1): in_development, non-terminal, next=ready_for_code_review
	//   call 2 (post-dispatch2): ready_for_code_review, non-terminal, no transitions → completed
	callIdx := 0
	getNextResponses := []struct {
		status      string
		isTerminal  bool
		nextTargets []string
	}{
		{"in_development", false, nil},                               // pre-loop
		{"in_development", false, []string{"ready_for_code_review"}}, // post-dispatch1
		{"ready_for_code_review", false, nil},                        // post-dispatch2: no transitions → completed
	}

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			r := getNextResponses[callIdx]
			callIdx++
			info := &services.NextStatusInfo{
				CurrentStatus: r.status,
				IsTerminal:    r.isTerminal,
			}
			for _, t := range r.nextTargets {
				info.AvailableTransitions = append(info.AvailableTransitions,
					services.TransitionInfoWithAction{TransitionInfo: workflow.TransitionInfo{TargetStatus: t}})
			}
			return info, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	dispatchCount := 0
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "do work for " + status,
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatchCount++
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed, got %s", result.Outcome)
	}
	if result.StagesCompleted != 2 {
		t.Errorf("expected StagesCompleted=2, got %d", result.StagesCompleted)
	}
	if dispatchCount != 2 {
		t.Errorf("expected 2 dispatcher calls, got %d", dispatchCount)
	}
}

// TestRunController_ArchiveAction verifies that an archive action type stops the
// loop and returns outcome "completed" without dispatching an agent.
func TestRunController_ArchiveAction(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "cancelled",
				IsTerminal:    false,
			}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action: config.ActionArchive,
			}, nil
		},
	}
	dispatched := false
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatched = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed for archive action, got %s", result.Outcome)
	}
	if dispatched {
		t.Error("expected no dispatcher call for archive action, but Dispatch was called")
	}
	if result.StagesCompleted != 1 {
		t.Errorf("expected StagesCompleted=1, got %d", result.StagesCompleted)
	}
}

// TestRunController_AdvanceStatusAction verifies that the advance_status action type
// advances the entity status without dispatching an agent and completes successfully.
//
// Controller flow for advance_status:
//   - GetNextStatus called once before loop (initial status).
//   - Inside loop: GetNextStatus again to get transitions, TransitionStatus called,
//     stage recorded, and if new status is terminal, returns "completed".
func TestRunController_AdvanceStatusAction(t *testing.T) {
	// GetNextStatus call sequence:
	//   call 0 (pre-loop): in_development, non-terminal
	//   call 1 (inside advance_status): in_development, non-terminal, next=completed
	//   After TransitionStatus("completed"), workflowSvc.IsTerminalStatus("completed")=true → return
	callIdx := 0
	getNextResponses := []struct {
		status      string
		isTerminal  bool
		nextTargets []string
	}{
		{"in_development", false, nil},                   // pre-loop
		{"in_development", false, []string{"completed"}}, // inside advance_status handler
	}

	transitionCalled := false
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			r := getNextResponses[callIdx]
			callIdx++
			info := &services.NextStatusInfo{
				CurrentStatus: r.status,
				IsTerminal:    r.isTerminal,
			}
			for _, tgt := range r.nextTargets {
				info.AvailableTransitions = append(info.AvailableTransitions,
					services.TransitionInfoWithAction{TransitionInfo: workflow.TransitionInfo{TargetStatus: tgt}})
			}
			return info, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			transitionCalled = true
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action: config.ActionAdvanceStatus,
			}, nil
		},
	}
	dispatched := false
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatched = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed for advance_status action, got %s", result.Outcome)
	}
	if dispatched {
		t.Error("expected no dispatcher call for advance_status action, but Dispatch was called")
	}
	if !transitionCalled {
		t.Error("expected TransitionStatus to be called for advance_status action, but it was not")
	}
	if result.StagesCompleted != 1 {
		t.Errorf("expected StagesCompleted=1, got %d", result.StagesCompleted)
	}
}

// TestRunController_ToolNotFound verifies that when a provider key has no registered
// dispatcher, the run fails with outcome "failed" containing a descriptive error.
func TestRunController_ToolNotFound(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:   config.ActionSpawnAgent,
				Provider: "unknown-provider", // not registered
			}, nil
		},
	}
	// Only the default "" dispatcher is registered; "unknown-provider" is missing.
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("expected Outcome=failed for unknown provider, got %s", result.Outcome)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for unknown provider, got empty string")
	}
}

// TestRunController_DryRun verifies that dry-run mode does not dispatch agents or
// advance status but still records stages and completes successfully.
//
// Controller dry-run flow for spawn_agent:
//   - GetNextStatus called once before loop (initial status).
//   - Inside loop: record stage, call GetNextStatus again to simulate advancement.
//   - If no transitions or IsTerminal, return "completed".
func TestRunController_DryRun(t *testing.T) {
	// GetNextStatus call sequence:
	//   call 0 (pre-loop): in_development, non-terminal (loop starts)
	//   call 1 (dry-run simulation): in_development, IsTerminal=true → completed
	callIdx := 0
	getNextResponses := []struct {
		status     string
		isTerminal bool
		hasTrans   bool
	}{
		{"in_development", false, true}, // pre-loop: non-terminal, has transitions
		{"in_development", true, false}, // dry-run simulation: treat as terminal → exit
	}

	transitionCalled := false
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			r := getNextResponses[callIdx]
			callIdx++
			info := &services.NextStatusInfo{
				CurrentStatus: r.status,
				IsTerminal:    r.isTerminal,
			}
			if r.hasTrans {
				info.AvailableTransitions = []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				}
			}
			return info, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			transitionCalled = true
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "implement",
			}, nil
		},
	}
	dispatched := false
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatched = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected no dispatcher call in dry-run mode, but Dispatch was called")
	}
	if transitionCalled {
		t.Error("expected no TransitionStatus call in dry-run mode, but it was called")
	}
	if result.Outcome == "" {
		t.Error("expected non-empty Outcome in dry-run mode")
	}
	if result.StagesCompleted < 1 {
		t.Errorf("expected at least 1 stage recorded in dry-run mode, got %d", result.StagesCompleted)
	}
}

func TestRunController_DryRunUsesSimulatedStatusTransitions(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "ready_for_review"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			t.Fatal("dry-run must not transition live status")
			return nil, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "do work for " + status,
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				t.Fatal("dry-run must not dispatch")
				return nil, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := ctrl.Run(ctx, "CC-001", RunOptions{DryRun: true})

	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed; error=%q", result.Outcome, result.Error)
	}
	if result.StagesCompleted != 2 {
		t.Fatalf("StagesCompleted = %d, want 2", result.StagesCompleted)
	}
	if result.Stages[0].Status != "development" || result.Stages[1].Status != "ready_for_review" {
		t.Fatalf("dry-run statuses = %q, %q; want development, ready_for_review",
			result.Stages[0].Status, result.Stages[1].Status)
	}
	if result.FinalStatus != "ready_for_review" {
		t.Fatalf("FinalStatus = %q, want ready_for_review", result.FinalStatus)
	}
}

// TestRunController_TransitionError verifies that a TransitionStatus error stops
// the loop and returns outcome "failed" with a descriptive error message.
func TestRunController_TransitionError(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "ready_for_code_review"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return nil, errors.New("transition rejected: workflow constraint violated")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "implement",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("expected Outcome=failed after transition error, got %s", result.Outcome)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error after transition error, got empty string")
	}
}

// TestRunController_DispatcherSelection_ExplicitProvider verifies that when an
// action specifies provider="anthropic", the anthropic dispatcher is selected
// rather than the default "" dispatcher.
func TestRunController_DispatcherSelection_ExplicitProvider(t *testing.T) {
	defaultCalled := false
	anthropicCalled := false

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic", // explicitly request anthropic
				Instruction: "implement",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				defaultCalled = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
		"anthropic": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				anthropicCalled = true
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	_, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if !anthropicCalled {
		t.Error("expected anthropic dispatcher to be called, but it was not")
	}
	if defaultCalled {
		t.Error("expected default dispatcher NOT to be called, but it was")
	}
}

// ---------------------------------------------------------------------------
// TC-F05-001: StageLog.OutputSummary population
// ---------------------------------------------------------------------------

// TestRunController_OutputSummary_PopulatedOnSuccess verifies that when a
// spawn_agent dispatch succeeds (exit code 0) with non-empty Stdout, the
// resulting StageLog.OutputSummary is set to that Stdout value.
func TestRunController_OutputSummary_PopulatedOnSuccess(t *testing.T) {
	const agentOutput = "agent completed the task successfully"

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: "completed"}, nil
		},
	}

	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:    config.ActionSpawnAgent,
				AgentType: "developer",
			}, nil
		},
	}

	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
					ExitCode: 0,
					Stdout:   agentOutput,
				}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E22-F05-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if len(result.Stages) == 0 {
		t.Fatal("expected at least one stage, got 0")
	}
	if result.Stages[0].OutputSummary != agentOutput {
		t.Errorf("OutputSummary = %q, want %q", result.Stages[0].OutputSummary, agentOutput)
	}
}

// TestRunController_OutputSummary_EmptyOnFailure verifies that when a
// spawn_agent dispatch fails (non-zero exit code), OutputSummary is NOT set.
func TestRunController_OutputSummary_EmptyOnFailure(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
	}

	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:    config.ActionSpawnAgent,
				AgentType: "developer",
			}, nil
		},
	}

	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
					ExitCode: 1,
					Stdout:   "some output before failure",
					Stderr:   "agent error",
				}, nil
			},
		},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E22-F05-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if len(result.Stages) == 0 {
		t.Fatal("expected at least one stage, got 0")
	}
	if result.Stages[0].OutputSummary != "" {
		t.Errorf("OutputSummary should be empty for failed stage, got %q", result.Stages[0].OutputSummary)
	}
	if result.Outcome != "failed" {
		t.Errorf("Outcome = %q, want %q", result.Outcome, "failed")
	}
}
