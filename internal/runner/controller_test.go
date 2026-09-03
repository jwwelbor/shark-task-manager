package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
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
var _ CascadeChildrenService = (*MockCascadeChildrenService)(nil)

type mockQuestionBlocker struct {
	check func(context.Context, models.EntityType, string) (*services.QuestionBlock, error)
}

func (m mockQuestionBlocker) Check(ctx context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
	return m.check(ctx, entityType, key)
}

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

// MockCascadeChildrenService implements CascadeChildrenService for testing.
type MockCascadeChildrenService struct {
	DescribeDispatchableChildrenFunc func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error)
}

func (m *MockCascadeChildrenService) DescribeDispatchableChildren(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
	if m.DescribeDispatchableChildrenFunc != nil {
		return m.DescribeDispatchableChildrenFunc(ctx, entityType, key)
	}
	return services.CascadeChildrenState{}, nil
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

// TC-307/TC-308: The production RunController entrypoint must return the
// compact I-03 pause before either placeholder/action work or a dispatcher in
// both real and dry-run modes. CLI preflight protects the lease; this test
// proves direct controller callers retain the same safe contract.
func TestRunController_BlockedCandidatePausesBeforeDispatchWork_TC307_TC308(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "dry_run"}[dryRun], func(t *testing.T) {
			placeholderCalls := 0
			actionCalls := 0
			dispatchCalls := 0
			controller, err := NewRunController(RunControllerDeps{
				Transitioner: &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
					return &services.NextStatusInfo{CurrentStatus: "active"}, nil
				}},
				Placeholders: &MockPlaceholderGen{GenerateFunc: func(context.Context, string) (map[string]string, error) {
					placeholderCalls++
					return nil, errors.New("blocked run must not generate placeholders")
				}},
				ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
					actionCalls++
					return nil, errors.New("blocked run must not resolve action")
				}},
				WorkflowSvc: defaultWorkflowSvc(),
				Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
					dispatchCalls++
					return nil, errors.New("blocked run must not dispatch worker")
				}}},
				QuestionBlocker: mockQuestionBlocker{check: func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
					if entityType != models.EntityTypeFeature || key != "E39-F03" {
						t.Fatalf("blocker candidate = %s %s, want feature E39-F03", entityType, key)
					}
					return &services.QuestionBlock{QuestionKey: "Q001", Summary: "Choose", ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
				}},
			})
			if err != nil {
				t.Fatalf("NewRunController() error = %v", err)
			}

			got, err := controller.Run(context.Background(), "E39-F03", RunOptions{EntityType: "feature", DryRun: dryRun})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if got.Outcome != "paused" || got.FinalStatus != "active" || got.QuestionBlock == nil {
				t.Fatalf("Run() = %#v, want compact blocked pause", got)
			}
			if *got.QuestionBlock != (services.QuestionBlock{QuestionKey: "Q001", Summary: "Choose", ResolutionOwner: "owner", CurrentResponder: "alice"}) {
				t.Fatalf("question_block = %#v, want I-03 handoff", got.QuestionBlock)
			}
			if placeholderCalls != 0 || actionCalls != 0 || dispatchCalls != 0 {
				t.Fatalf("blocked run placeholder/action/dispatch calls = %d/%d/%d, want 0/0/0", placeholderCalls, actionCalls, dispatchCalls)
			}
		})
	}
}

// TC-103: a ready Question is a non-dispatching human checkpoint. The real
// runner must stop before asking the responder placeholder adapter or action
// renderer for a responder that no longer exists.
func TestRunController_ReadyQuestionPausesBeforeResponderRendering_TC103(t *testing.T) {
	transitioner := &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
		return &services.NextStatusInfo{CurrentStatus: "ready_for_resolution", IsTerminal: true}, nil
	}}
	placeholderCalls := 0
	actionCalls := 0
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: &MockPlaceholderGen{GenerateFunc: func(context.Context, string) (map[string]string, error) {
			placeholderCalls++
			return nil, errors.New("responder placeholders must not run")
		}},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			actionCalls++
			return nil, errors.New("workflow action lookup must not run")
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{}},
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	result, err := controller.Run(context.Background(), "Q001", RunOptions{EntityType: "question"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome != "paused" || result.FinalStatus != "ready_for_resolution" || len(result.Stages) != 0 {
		t.Fatalf("Run() = %+v, want paused ready checkpoint without stages", result)
	}
	if placeholderCalls != 0 || actionCalls != 0 {
		t.Fatalf("ready Question rendered placeholders=%d actions=%d, want neither", placeholderCalls, actionCalls)
	}
}

// TC-103/TC-104: Every Question status that carries the terminal dispatch
// signal because there is no responder is a pause checkpoint, not a completed
// Question. This includes the F01-compatible unconfigured draft and the
// migration-compatible open/answering statuses with no persisted responder
// state. Dry runs use the same controller path, so both modes must agree.
func TestRunController_QuestionNoResponderCheckpointsPause_TC103_TC104(t *testing.T) {
	for _, status := range []string{"draft", "open", "answering", "ready_for_resolution"} {
		t.Run(status, func(t *testing.T) {
			for _, dryRun := range []bool{false, true} {
				t.Run(map[bool]string{false: "run", true: "dry_run"}[dryRun], func(t *testing.T) {
					placeholderCalls := 0
					actionCalls := 0
					controller, err := NewRunController(RunControllerDeps{
						Transitioner: &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
							return &services.NextStatusInfo{CurrentStatus: status, IsTerminal: true}, nil
						}},
						Placeholders: &MockPlaceholderGen{GenerateFunc: func(context.Context, string) (map[string]string, error) {
							placeholderCalls++
							return nil, errors.New("Question responder placeholders must not run")
						}},
						ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
							actionCalls++
							return nil, errors.New("Question workflow action must not run")
						}},
						WorkflowSvc: defaultWorkflowSvc(),
						Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{}},
					})
					if err != nil {
						t.Fatalf("NewRunController() error = %v", err)
					}

					result, err := controller.Run(context.Background(), "Q001", RunOptions{EntityType: "question", DryRun: dryRun})
					if err != nil {
						t.Fatalf("Run() error = %v", err)
					}
					if result.Outcome != "paused" || result.FinalStatus != status || len(result.Stages) != 0 {
						t.Fatalf("Run() = %+v, want paused %s checkpoint without stages", result, status)
					}
					if placeholderCalls != 0 || actionCalls != 0 {
						t.Fatalf("Question %s rendered placeholders=%d actions=%d, want neither", status, placeholderCalls, actionCalls)
					}
				})
			}
		})
	}
}

// TC-103: Responder-less checkpoints are pauses, but real Question terminal
// states retain the normal archive/already-terminal contract.
func TestRunController_QuestionDurableTerminalsRemainAlreadyTerminal_TC103(t *testing.T) {
	for _, status := range []string{"resolved", "withdrawn", "superseded", "archived"} {
		t.Run(status, func(t *testing.T) {
			controller := makeController(t,
				&MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
					return &services.NextStatusInfo{CurrentStatus: status, IsTerminal: true}, nil
				}},
				&MockActionService{}, nil,
			)
			result, err := controller.Run(context.Background(), "Q001", RunOptions{EntityType: "question"})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Outcome != "already_terminal" || result.FinalStatus != status {
				t.Fatalf("Run(%s) = %+v, want durable terminal result", status, result)
			}
		})
	}
}

// TC-104: A cascade child that is paused for lack of a Question responder
// must not count as progress or auto-advance its parent. This uses the real
// child RunController rather than returning a fabricated child outcome.
func TestRunController_CascadeQuestionNoResponderPausesParent_TC104(t *testing.T) {
	child := makeController(t,
		&MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "open", IsTerminal: true}, nil
		}},
		&MockActionService{}, nil,
	)
	parent, err := NewRunController(RunControllerDeps{
		Transitioner: &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "active"}, nil
		}},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{}},
		ChildrenSvc: &MockCascadeChildrenService{DescribeDispatchableChildrenFunc: func(context.Context, string, string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{Children: []services.CascadeChild{{Key: "Q001", EntityType: models.EntityTypeQuestion}}, TotalChildren: 1, NonTerminalChildren: 1}, nil
		}},
		RunChild: func(ctx context.Context, entityType, key string, opts RunOptions) (*RunResult, error) {
			opts.EntityType = entityType
			return child.Run(ctx, key, opts)
		},
	})
	if err != nil {
		t.Fatalf("NewRunController(parent) error = %v", err)
	}

	result, err := parent.Run(context.Background(), "E01-F01", RunOptions{EntityType: "feature"})
	if err != nil {
		t.Fatalf("parent Run() error = %v", err)
	}
	if result.Outcome != "paused" || result.StagesCompleted != 0 {
		t.Fatalf("parent Run() = %+v, want paused parent with no child progress", result)
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

// TestRunController_QuestionResponseHandoffPersistsBeforeReturning proves the
// parent-run lifecycle for a serial Question: the worker returns only bounded
// response data, the parent persists it under alice's lease, and the run stops
// before bob can be dispatched under that same lease. A later keyed dispatch
// can therefore route bob only after the parent has released alice's lease.
func TestRunController_QuestionResponseHandoffPersistsBeforeReturning(t *testing.T) {
	state := "open"
	var persisted []QuestionResponseHandoff
	transitionCalls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: state,
				AvailableTransitions: []services.TransitionInfoWithAction{{
					TransitionInfo: workflow.TransitionInfo{TargetStatus: "answering"},
				}},
			}, nil
		},
		TransitionStatusFunc: func(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error) {
			transitionCalls++
			return &services.TransitionResult{ToStatus: "answering"}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "responder"}, nil
		},
	}
	persister := QuestionResponsePersisterFunc(func(_ context.Context, handoff QuestionResponseHandoff) error {
		persisted = append(persisted, handoff)
		state = "answering" // Alice's committed response makes bob eligible after release.
		return nil
	})
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: &MockPlaceholderGen{GenerateFunc: func(context.Context, string) (map[string]string, error) {
			return map[string]string{"current_responder": "alice"}, nil
		}},
		ActionSvc:   actionSvc,
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: "QUESTION_RESPONSE_JSON: {\"summary\":\"approved\",\"evidence_pointer\":\"docs/spec.md\"}"}, nil
		}}},
		QuestionResponses: persister,
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	result, err := controller.Run(context.Background(), "Q001", RunOptions{EntityType: "question", SessionID: "session-alice"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome != "completed" || result.FinalStatus != "answering" {
		t.Fatalf("result = %+v, want completed at answering", result)
	}
	if len(persisted) != 1 {
		t.Fatalf("persisted handoffs = %d, want 1", len(persisted))
	}
	if got := persisted[0]; got.Key != "Q001" || got.SessionID != "session-alice" || got.Responder != "alice" || got.Summary != "approved" || got.EvidencePointer != "docs/spec.md" {
		t.Fatalf("handoff = %+v", got)
	}
	if transitionCalls != 0 {
		t.Fatalf("Question run transitioned %d times; response persistence must finish before lease release and later bob routing", transitionCalls)
	}
	if result.Stages[0].OutputSummary != "" {
		t.Fatalf("Question worker response leaked into stage output: %q", result.Stages[0].OutputSummary)
	}
}

// TestRunController_QuestionResponseLifecyclePersistsAliceThenRoutesBob uses
// the real Question, claim, and SQLite services through the runner's parent
// handoff seam. It proves the CLI runner lifecycle rather than a worker-side
// direct service call: alice's result commits under the parent lease, release
// happens afterwards, and the next read exposes bob as the sole responder.
func TestRunController_QuestionResponseLifecyclePersistsAliceThenRoutesBob(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.InitDB(filepath.Join(t.TempDir(), "question-runner.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	questionRepo := repository.NewQuestionRepository(repoDB)
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	question, err := questionSvc.CreateQuestion(ctx, services.CreateQuestionInput{Title: "Runner response", Summary: "Route one responder", Requester: "test"})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	if _, err := questionSvc.ConfigureWorkflow(ctx, services.ConfigureWorkflowInput{Key: question.Key, ResolutionOwner: "owner", Responders: []string{"alice", "bob"}}); err != nil {
		t.Fatalf("ConfigureWorkflow() error = %v", err)
	}
	claimSvc := services.NewClaimService(claimrepo.NewRepository(repoDB), nil)
	questionSvc.SetClaimReader(claimSvc)
	claim, err := claimSvc.Claim(ctx, services.ClaimInput{EntityType: string(models.EntityTypeQuestion), EntityKey: question.Key, ClaimedBy: "alice"})
	if err != nil {
		t.Fatalf("Claim(alice) error = %v", err)
	}

	controller, err := NewRunController(RunControllerDeps{
		Transitioner: questionSvc,
		Placeholders: &MockPlaceholderGen{GenerateFunc: func(context.Context, string) (map[string]string, error) {
			return map[string]string{"current_responder": "alice"}, nil
		}},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "responder"}, nil
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: "QUESTION_RESPONSE_JSON: {\"summary\":\"approved\",\"evidence_pointer\":\"docs/spec.md\"}"}, nil
		}}},
		QuestionResponses: QuestionResponsePersisterFunc(func(ctx context.Context, handoff QuestionResponseHandoff) error {
			_, err := questionSvc.RecordResponse(ctx, services.RecordQuestionResponseInput{Key: handoff.Key, SessionID: handoff.SessionID, Responder: handoff.Responder, Summary: handoff.Summary, EvidencePointer: handoff.EvidencePointer})
			return err
		}),
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}
	result, err := controller.Run(ctx, question.Key, RunOptions{EntityType: string(models.EntityTypeQuestion), SessionID: claim.SessionID})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome != "completed" || result.FinalStatus != "answering" {
		t.Fatalf("run result = %+v, want completed answering", result)
	}
	if _, err := claimSvc.Release(ctx, string(models.EntityTypeQuestion), question.Key, claim.SessionID, "completed", false); err != nil {
		t.Fatalf("Release(alice) error = %v", err)
	}
	persisted, err := questionSvc.GetQuestion(ctx, question.Key)
	if err != nil {
		t.Fatalf("GetQuestion() error = %v", err)
	}
	state, err := models.DecodeQuestionState(persisted.ContextData)
	if err != nil || state == nil {
		t.Fatalf("DecodeQuestionState() = %#v, %v", state, err)
	}
	if state.CurrentResponder() != "bob" || len(state.Responses) != 1 || state.Responses[0].Responder != "alice" {
		t.Fatalf("persisted Question state = %#v, want alice response and bob current", state)
	}
	next, err := questionSvc.GetNextStatus(ctx, question.Key)
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if next.IsClaimed || next.CurrentStatus != "answering" {
		t.Fatalf("next after alice release = %#v, want unclaimed answering for bob", next)
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

func TestRunController_ReportsIterationAndActionProgress(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_triage"}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionWaitForTriage, AgentType: "scrum", Provider: "codex"}, nil
		},
	}
	controller := makeController(t, transitioner, actionSvc, nil)
	var updates []RunProgress
	result, err := controller.Run(context.Background(), "E07-F01", RunOptions{
		Progress: func(update RunProgress) { updates = append(updates, update) },
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome != "paused" {
		t.Fatalf("Outcome = %q, want paused", result.Outcome)
	}
	if len(updates) != 2 {
		t.Fatalf("progress update count = %d, want 2", len(updates))
	}
	if got := updates[0]; got.Phase != "iteration" || got.Iteration != 1 || got.Status != "in_triage" || got.EntityKey != "E07-F01" {
		t.Errorf("iteration update = %#v", got)
	}
	if got := updates[1]; got.Phase != "action" || got.Action != config.ActionWaitForTriage || got.AgentType != "scrum" || got.Provider != "codex" {
		t.Errorf("action update = %#v", got)
	}
}

func TestRunController_CascadeAction_RunsDispatchableChildrenAndAutoAdvancesParent(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "active",
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
				Action:      config.ActionCascade,
				Instruction: "Cascade to ready children",
			}, nil
		},
	}
	runChildCalled := false
	cascadeSvc := &MockCascadeChildrenService{
		DescribeDispatchableChildrenFunc: func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			if runChildCalled {
				return services.CascadeChildrenState{
					TotalChildren:       1,
					NonTerminalChildren: 0,
				}, nil
			}
			return services.CascadeChildrenState{
				Children:            []services.CascadeChild{{Key: "E07-F01-T01", EntityType: "task"}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		},
	}

	runChild := func(ctx context.Context, childType, key string, opts RunOptions) (*RunResult, error) {
		runChildCalled = true
		if childType != "task" {
			t.Fatalf("runChild childType=%s, want task", childType)
		}
		if key != "E07-F01-T01" {
			t.Fatalf("runChild key=%s, want E07-F01-T01", key)
		}
		if opts.EntityType != "task" {
			t.Fatalf("runChild opts.EntityType=%s, want task", opts.EntityType)
		}
		return &RunResult{
			EntityKey:   key,
			FinalStatus: "done",
			Outcome:     "completed",
			Stages: []StageLog{{
				Status:   "active",
				Action:   config.ActionSpawnAgent,
				Duration: time.Millisecond,
			}},
			StagesCompleted: 1,
		}, nil
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: &MockPlaceholderGen{},
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{
			"": &MockDispatcher{},
		},
		ChildrenSvc: cascadeSvc,
		RunChild:    runChild,
	})
	if err != nil {
		t.Fatalf("makeController: NewRunController failed: %v", err)
	}
	result, err := ctrl.Run(context.Background(), "E07-F01", RunOptions{EntityType: "feature"})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if !runChildCalled {
		t.Fatal("expected child run to be invoked, but it was not")
	}
	if result.Outcome != "completed" {
		t.Errorf("expected Outcome=completed, got %s", result.Outcome)
	}
	if result.FinalStatus != "completed" {
		t.Errorf("expected FinalStatus=completed, got %s", result.FinalStatus)
	}
	if result.StagesCompleted != 2 {
		t.Fatalf("expected 2 stages completed, got %d", result.StagesCompleted)
	}
	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 recorded stages, got %d", len(result.Stages))
	}
	if result.Stages[0].Action != config.ActionSpawnAgent {
		t.Errorf("expected first stage action=%s, got %s", config.ActionSpawnAgent, result.Stages[0].Action)
	}
	if result.Stages[1].Action != config.ActionAdvanceStatus {
		t.Errorf("expected second stage action=%s, got %s", config.ActionAdvanceStatus, result.Stages[1].Action)
	}
}

// TestRunController_CascadeStagesCarryEntityKey verifies that after
// handleCascade flattens each child's Stages into the parent's result.Stages
// (controller.go's `result.Stages = append(result.Stages, childResult.Stages...)`),
// each flattened StageLog entry still identifies which cascade child produced
// it via StageLog.EntityKey. Before B052's fix, StageLog carried no entity
// attribution at all, so per-child stage metrics became unrecoverable once
// flattened into the parent's single Stages slice.
//
// Unlike TestRunController_CascadeAction_RunsDispatchableChildrenAndAutoAdvancesParent
// (which stubs RunChild to return a canned RunResult), this test drives each
// cascade child through a REAL child *RunController* — exactly as production
// wiring does in internal/cli/commands/run.go's runChild closure — so it
// exercises the actual StageLog{EntityKey: key, ...} construction sites in
// handleSpawnAgent, not just the flattening `append` itself.
func TestRunController_CascadeStagesCarryEntityKey(t *testing.T) {
	childKeys := []string{"E07-F01-T01", "E07-F01-T02"}

	dispatcher := &MockDispatcher{
		DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{
				ExitCode: 0,
				Stdout:   "output for " + input.EntityKey,
				Duration: time.Millisecond,
				Command:  "claude -p " + input.EntityKey,
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{"anthropic": dispatcher}

	childTransitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "active",
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
	childActionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do work",
			}, nil
		},
	}

	// runChild mirrors production wiring (internal/cli/commands/run.go): it
	// constructs a real child RunController per child and calls its Run().
	runChild := func(ctx context.Context, childType, key string, childOpts RunOptions) (*RunResult, error) {
		childCtrl, err := NewRunController(RunControllerDeps{
			Transitioner: childTransitioner,
			Placeholders: &MockPlaceholderGen{},
			ActionSvc:    childActionSvc,
			WorkflowSvc:  defaultWorkflowSvc(),
			Dispatchers:  dispatchers,
		})
		if err != nil {
			t.Fatalf("build child controller for %s: %v", key, err)
		}
		childOpts.EntityType = childType
		return childCtrl.Run(ctx, key, childOpts)
	}

	cascadeSvc := &MockCascadeChildrenService{
		DescribeDispatchableChildrenFunc: func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{
				Children: []services.CascadeChild{
					{Key: childKeys[0], EntityType: "task"},
					{Key: childKeys[1], EntityType: "task"},
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		},
	}

	parentTransitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "active", IsTerminal: false}, nil
		},
	}
	parentActionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		},
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: parentTransitioner,
		Placeholders: &MockPlaceholderGen{},
		ActionSvc:    parentActionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
		ChildrenSvc:  cascadeSvc,
		RunChild:     runChild,
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	result, err := ctrl.Run(context.Background(), "E07-F01", RunOptions{EntityType: "feature"})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	// Each child dispatches exactly one spawn_agent stage. After flattening,
	// result.Stages must contain one entry per child, each correctly tagged
	// with that child's own entity key (never empty, never the other child's,
	// never the parent's).
	byKey := map[string]int{}
	for _, stage := range result.Stages {
		if stage.Action != config.ActionSpawnAgent {
			continue
		}
		byKey[stage.EntityKey]++
	}
	for _, key := range childKeys {
		if byKey[key] != 1 {
			t.Errorf("expected exactly 1 spawn_agent stage tagged EntityKey=%s, got %d (all stages: %#v)", key, byKey[key], result.Stages)
		}
	}
	if got := byKey[""]; got != 0 {
		t.Errorf("expected 0 stages with empty EntityKey, got %d (all stages: %#v)", got, result.Stages)
	}
}

// TC-308: a directly blocked cascade child is parked, rather than ending the
// parent run. The runner must continue to a later eligible sibling exactly as
// keyed next does; only an all-parked cascade returns the compact block pause.
func TestRunController_CascadeFallsThroughBlockedChild_TC308(t *testing.T) {
	secondChildRan := false
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			if secondChildRan {
				return &services.NextStatusInfo{CurrentStatus: "in_review"}, nil
			}
			return &services.NextStatusInfo{CurrentStatus: "active"}, nil
		},
	}
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: &MockPlaceholderGen{},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*config.PopulatedAction, error) {
			if status == "in_review" {
				return &config.PopulatedAction{Action: config.ActionPause}, nil
			}
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{
			"": &MockDispatcher{},
		},
		ChildrenSvc: &MockCascadeChildrenService{DescribeDispatchableChildrenFunc: func(context.Context, string, string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{Children: []services.CascadeChild{
				{Key: "E39-F03", EntityType: models.EntityTypeFeature},
				{Key: "E39-F04", EntityType: models.EntityTypeFeature},
			}, TotalChildren: 2, NonTerminalChildren: 2}, nil
		}},
		RunChild: func(_ context.Context, _ string, key string, _ RunOptions) (*RunResult, error) {
			switch key {
			case "E39-F03":
				return &RunResult{EntityKey: key, Outcome: "paused", QuestionBlock: &services.QuestionBlock{QuestionKey: "Q001", Summary: "Gate", ResolutionOwner: "owner", CurrentResponder: "alice"}}, nil
			case "E39-F04":
				secondChildRan = true
				return &RunResult{EntityKey: key, FinalStatus: "completed", Outcome: "completed"}, nil
			default:
				t.Fatalf("unexpected cascade child %q", key)
				return nil, nil
			}
		},
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	got, err := controller.Run(context.Background(), "E39", RunOptions{EntityType: "epic"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !secondChildRan {
		t.Fatal("blocked first child prevented the eligible sibling from running")
	}
	if got.Outcome != "paused" || got.QuestionBlock != nil {
		t.Fatalf("Run() = %#v, want sibling-driven pause without blocked-child handoff", got)
	}
}

func TestRunController_CascadeDoesNotAttributeMixedPauseToQuestion(t *testing.T) {
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "active"}, nil
		}},
		Placeholders: &MockPlaceholderGen{},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{}},
		ChildrenSvc: &MockCascadeChildrenService{DescribeDispatchableChildrenFunc: func(context.Context, string, string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{Children: []services.CascadeChild{{Key: "QUESTION", EntityType: models.EntityTypeFeature}, {Key: "OTHER", EntityType: models.EntityTypeFeature}}, TotalChildren: 2, NonTerminalChildren: 2}, nil
		}},
		RunChild: func(_ context.Context, _ string, key string, _ RunOptions) (*RunResult, error) {
			if key == "QUESTION" {
				return &RunResult{Outcome: "paused", QuestionBlock: &services.QuestionBlock{QuestionKey: "Q001"}}, nil
			}
			return &RunResult{Outcome: "paused"}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	got, err := controller.Run(context.Background(), "E39", RunOptions{EntityType: "epic"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Outcome != "paused" {
		t.Errorf("Outcome = %q, want paused", got.Outcome)
	}
	if got.QuestionBlock != nil {
		t.Errorf("QuestionBlock = %#v, want nil for a mixed pause", got.QuestionBlock)
	}
}

func TestRunController_CascadeAllQuestionBlockedRetainsFirstQuestionBlock(t *testing.T) {
	first := &services.QuestionBlock{QuestionKey: "Q001"}
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: &MockTransitioner{GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "active"}, nil
		}},
		Placeholders: &MockPlaceholderGen{},
		ActionSvc: &MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		}},
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"": &MockDispatcher{}},
		ChildrenSvc: &MockCascadeChildrenService{DescribeDispatchableChildrenFunc: func(context.Context, string, string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{Children: []services.CascadeChild{{Key: "FIRST", EntityType: models.EntityTypeFeature}, {Key: "SECOND", EntityType: models.EntityTypeFeature}}, TotalChildren: 2, NonTerminalChildren: 2}, nil
		}},
		RunChild: func(_ context.Context, _ string, key string, _ RunOptions) (*RunResult, error) {
			if key == "FIRST" {
				return &RunResult{Outcome: "paused", QuestionBlock: first}, nil
			}
			return &RunResult{Outcome: "paused", QuestionBlock: &services.QuestionBlock{QuestionKey: "Q002"}}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	got, err := controller.Run(context.Background(), "E39", RunOptions{EntityType: "epic"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Outcome != "paused" || got.QuestionBlock == nil || *got.QuestionBlock != *first {
		t.Fatalf("Run() = %#v, want first compact Question pause", got)
	}
}

func TestRunController_CascadeAction_PropagatesChildFailure(t *testing.T) {
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "active",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
	}

	// transitioner for this flow should never transition parent if child fails.
	var transitionToParentCalled bool
	transitioner.TransitionStatusFunc = func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
		transitionToParentCalled = true
		return &services.TransitionResult{ToStatus: target}, nil
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionCascade,
				Instruction: "Cascade to ready children",
			}, nil
		},
	}
	cascadeSvc := &MockCascadeChildrenService{
		DescribeDispatchableChildrenFunc: func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{
				Children:            []services.CascadeChild{{Key: "E07-F01-T01", EntityType: "task"}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		},
	}

	runChild := func(ctx context.Context, childType, key string, opts RunOptions) (*RunResult, error) {
		return &RunResult{
			EntityKey:       key,
			FinalStatus:     "blocked",
			Outcome:         "failed",
			Error:           "child dispatch failed",
			StagesCompleted: 0,
		}, nil
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: &MockPlaceholderGen{},
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{
			"": &MockDispatcher{},
		},
		ChildrenSvc: cascadeSvc,
		RunChild:    runChild,
	})
	if err != nil {
		t.Fatalf("makeController: NewRunController failed: %v", err)
	}

	result, err := ctrl.Run(context.Background(), "E07-F01", RunOptions{EntityType: "feature"})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected Outcome=failed, got %s", result.Outcome)
	}
	if result.Error == "" || result.Error != "cascade child task E07-F01-T01 failed: child dispatch failed" {
		t.Fatalf("expected child failure error message, got %q", result.Error)
	}
	if transitionToParentCalled {
		t.Fatal("parent transition should not happen after child failure")
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

// TestRunController_CheckOrResumeDispatchesAndTransitions verifies that a
// populated check_or_resume action follows the normal agent-dispatch path.
// Tech-debt uses this action at in_progress, so treating it as a pause would
// leave the entity stalled without invoking its declared developer agent.
func TestRunController_CheckOrResumeDispatchesAndTransitions(t *testing.T) {
	var dispatched DispatchInput
	var transitionedTo string

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_progress",
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
				Outcomes: map[string]string{"pass": "completed"},
			}, nil
		},
		TransitionStatusFunc: func(_ context.Context, _ string, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
			transitionedTo = target
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*config.PopulatedAction, error) {
			if status != "in_progress" {
				t.Fatalf("GetStatusActionPopulated() status = %q, want in_progress", status)
			}
			return &config.PopulatedAction{
				Action:      config.ActionCheckOrResume,
				AgentType:   "developer",
				Provider:    "anthropic",
				Model:       "sonnet",
				Instruction: "Resolve the tech debt",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(_ context.Context, input DispatchInput) (*DispatchResult, error) {
			dispatched = input
			return &DispatchResult{ExitCode: 0, Stdout: "RECOMMENDED OUTCOME: pass"}, nil
		}},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "TD-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outcome == "paused" {
		t.Fatalf("Run() outcome = paused, want dispatched check_or_resume action")
	}
	if result.Outcome != "completed" {
		t.Fatalf("Run() outcome = %q, want completed", result.Outcome)
	}
	if dispatched.Instruction != "Resolve the tech debt" || dispatched.AgentType != "developer" || dispatched.Model != "sonnet" {
		t.Fatalf("DispatchInput = %+v, want populated developer dispatch", dispatched)
	}
	if transitionedTo != "completed" {
		t.Fatalf("TransitionStatus() target = %q, want completed", transitionedTo)
	}
}

// TestRunController_SpawnAgentUsesJSONRecommendedOutcome verifies the full
// dispatch loop routes to the workflow's "blocked" target when a worker's
// stdout is pure JSON (`{"outcome": "blocked"}`) with ExitCode 0 and no
// "RECOMMENDED OUTCOME:" text line. B046: this input shape previously fell
// through recommendedOutcome() to the pass-first target, reproducing the
// original bug (advancing past a blocked gate) for JSON-only worker output.
func TestRunController_SpawnAgentUsesJSONRecommendedOutcome(t *testing.T) {
	getNextCalls := 0
	var transitionedTo string
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			getNextCalls++
			if getNextCalls > 2 {
				return &services.NextStatusInfo{CurrentStatus: "on_hold", IsTerminal: true}, nil
			}
			return &services.NextStatusInfo{
				CurrentStatus: "in_progress",
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
				},
				Outcomes: map[string]string{"pass": "completed", "blocked": "on_hold"},
			}, nil
		},
		TransitionStatusFunc: func(_ context.Context, _ string, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
			transitionedTo = target
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, Provider: "anthropic", Instruction: "work"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: `{"outcome": "blocked"}`}, nil
		}},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	_, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if transitionedTo != "on_hold" {
		t.Fatalf("TransitionStatus() target = %q, want on_hold (blocked outcome), pass-first would incorrectly give completed", transitionedTo)
	}
}

// TestRunController_SpawnAgentMalformedJSONOutcomeFailsRunWithoutTransition
// verifies the full dispatch loop end-to-end for the fail-loud fix: when a
// worker's stdout looks like a JSON outcome object but fails to parse,
// TransitionStatus must never be called and the run must report a failed
// outcome, instead of silently advancing via the pass-first target. This
// pins down the actual guarantee B046 exists to provide, complementing the
// unit-level TestRecommendedOutcome_MalformedJSONFailsLoud coverage of
// recommendedOutcome() in isolation.
func TestRunController_SpawnAgentMalformedJSONOutcomeFailsRunWithoutTransition(t *testing.T) {
	transitionCalled := false
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(context.Context, string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_progress",
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
				},
				Outcomes: map[string]string{"pass": "completed", "blocked": "on_hold"},
			}, nil
		},
		TransitionStatusFunc: func(_ context.Context, _ string, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
			transitionCalled = true
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, Provider: "anthropic", Instruction: "work"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
			// Malformed JSON: looks like an outcome object but is truncated.
			return &DispatchResult{ExitCode: 0, Stdout: `{"outcome": "blocked"`}, nil
		}},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), "E07-F01-001", RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil (stage failures are reported via RunResult, not a Go error)", err)
	}
	if transitionCalled {
		t.Fatal("TransitionStatus() was called, want it never called — malformed JSON outcome must not silently advance the entity")
	}
	if result.Outcome != "failed" {
		t.Fatalf("RunResult.Outcome = %q, want %q", result.Outcome, "failed")
	}
	if result.Error == "" {
		t.Fatal("RunResult.Error is empty, want a parse-error message explaining the malformed JSON outcome")
	}
}

// TestRecommendedOutcome_JSONBody verifies a worker that returns a pure JSON
// object like {"outcome": "blocked"} (no "RECOMMENDED OUTCOME:" text line) is
// still recognized. B046: without this, workers using the JSON-only shape of
// the shark-rider worker-return contract fell through to the pass-first
// target, silently advancing past a blocked/failed gate.
func TestRecommendedOutcome_JSONBody(t *testing.T) {
	outcome, specified, err := recommendedOutcome(`{"outcome": "blocked"}`)
	if err != nil {
		t.Fatalf("recommendedOutcome() error = %v, want nil", err)
	}
	if !specified {
		t.Fatal("recommendedOutcome() specified = false, want true for JSON body")
	}
	if outcome != "blocked" {
		t.Fatalf("recommendedOutcome() outcome = %q, want %q", outcome, "blocked")
	}
}

// TestRecommendedOutcome_JSONBodyWithWhitespace verifies the whole-stdout JSON
// match tolerates surrounding whitespace/newlines, matching how real process
// output is captured.
func TestRecommendedOutcome_JSONBodyWithWhitespace(t *testing.T) {
	outcome, specified, err := recommendedOutcome("\n  {\"outcome\": \"simple\"}  \n")
	if err != nil {
		t.Fatalf("recommendedOutcome() error = %v, want nil", err)
	}
	if !specified {
		t.Fatal("recommendedOutcome() specified = false, want true for whitespace-padded JSON body")
	}
	if outcome != "simple" {
		t.Fatalf("recommendedOutcome() outcome = %q, want %q", outcome, "simple")
	}
}

// TestRecommendedOutcome_TextLineStillWorks verifies the existing
// "RECOMMENDED OUTCOME:" text-line format keeps working unchanged — this
// asserts the common, pre-existing case is unaffected by the new JSON path.
func TestRecommendedOutcome_TextLineStillWorks(t *testing.T) {
	outcome, specified, err := recommendedOutcome("Did the work.\nRECOMMENDED OUTCOME: pass")
	if err != nil {
		t.Fatalf("recommendedOutcome() error = %v, want nil", err)
	}
	if !specified {
		t.Fatal("recommendedOutcome() specified = false, want true for text-line format")
	}
	if outcome != "pass" {
		t.Fatalf("recommendedOutcome() outcome = %q, want %q", outcome, "pass")
	}
}

// TestRecommendedOutcome_ProseMentioningJSONIsIgnored verifies the safety
// property carried over from the text-line format: outcome-shaped JSON that
// appears embedded in a longer message (not as the SOLE stdout content) must
// not be misparsed as an explicit outcome. This mirrors the existing
// safeguard against prose merely mentioning "RECOMMENDED OUTCOME".
func TestRecommendedOutcome_ProseMentioningJSONIsIgnored(t *testing.T) {
	stdout := `I considered returning {"outcome": "blocked"} but decided to keep going.`
	outcome, specified, err := recommendedOutcome(stdout)
	if err != nil {
		t.Fatalf("recommendedOutcome() error = %v, want nil", err)
	}
	if specified {
		t.Fatalf("recommendedOutcome() specified = true, outcome = %q, want false (embedded JSON in prose must not trigger routing)", outcome)
	}
}

// TestRecommendedOutcome_MalformedJSONFailsLoud verifies that stdout shaped
// like a JSON outcome object but malformed (unterminated / invalid JSON)
// surfaces a parse error instead of silently falling through to the
// pass-first target. B046 was filed to stop exactly this failure class
// (silent advance past a gate); recommendedOutcome must fail loud here to
// match parseQuestionResponseHandoff's behavior on invalid JSON.
func TestRecommendedOutcome_MalformedJSONFailsLoud(t *testing.T) {
	outcome, specified, err := recommendedOutcome(`{"outcome": "blocked"`)
	if err == nil {
		t.Fatalf("recommendedOutcome() error = nil, want error for malformed JSON; got outcome=%q specified=%v", outcome, specified)
	}
}

// TestRecommendedOutcome_WrongTypedOutcomeFieldFailsLoud verifies a
// wrong-typed `outcome` field (e.g. a number instead of a string) surfaces a
// parse error rather than silently falling through to pass-first.
func TestRecommendedOutcome_WrongTypedOutcomeFieldFailsLoud(t *testing.T) {
	outcome, specified, err := recommendedOutcome(`{"outcome": 5}`)
	if err == nil {
		t.Fatalf("recommendedOutcome() error = nil, want error for wrong-typed outcome field; got outcome=%q specified=%v", outcome, specified)
	}
}

// TestRecommendedOutcome_JSONEmptyOutcomeIsSpecified verifies JSON
// `{"outcome":""}` is reported as specified (not silently dropped to
// pass-first), aligning empty-outcome semantics with the text-line format —
// both now defer to the unknown-outcome hard error in
// targetStatusForDispatch rather than diverging.
func TestRecommendedOutcome_JSONEmptyOutcomeIsSpecified(t *testing.T) {
	outcome, specified, err := recommendedOutcome(`{"outcome": ""}`)
	if err != nil {
		t.Fatalf("recommendedOutcome() error = %v, want nil", err)
	}
	if !specified {
		t.Fatal("recommendedOutcome() specified = false, want true for empty JSON outcome (must not silently fall back to pass-first)")
	}
	if outcome != "" {
		t.Fatalf("recommendedOutcome() outcome = %q, want empty string", outcome)
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
//
// GetNextStatusFunc is driven off the transitioner's OWN live status (updated
// by TransitionStatusFunc) rather than a fixed call-count sequence: Run()'s
// loop refreshes nextInfo with an extra GetNextStatus call between stages
// whenever a stage's stageOutcome doesn't already carry one (code-review
// round-7 Finding 1 sweep — see controller.go's Run() loop, the `else if
// !opts.DryRun` branch), so pinning an exact call count here would be
// coupled to an implementation detail rather than this test's actual intent
// (a 2-stage transition sequence completing correctly).
func TestRunController_HappyPath_MultiStage(t *testing.T) {
	status := "in_development"
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			info := &services.NextStatusInfo{CurrentStatus: status, IsTerminal: false}
			if status == "in_development" {
				info.AvailableTransitions = []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "ready_for_code_review"}},
				}
			}
			return info, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			status = target
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
