package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockEntityRepo implements EntityRepository for testing.
type mockEntityRepo struct {
	getByKeyFn     func(ctx context.Context, key string) (models.Entity, error)
	getByIDFn      func(ctx context.Context, id int64) (models.Entity, error)
	updateStatusFn func(ctx context.Context, id int64, status string) error
	updateFn       func(ctx context.Context, entity models.Entity) error
	getContextFn   func(ctx context.Context, id int64) (*string, error)
	setContextFn   func(ctx context.Context, id int64, data *string) error
}

func (m *mockEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockEntityRepo) Update(ctx context.Context, entity models.Entity) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, entity)
	}
	return nil
}

func (m *mockEntityRepo) GetContextData(ctx context.Context, id int64) (*string, error) {
	if m.getContextFn != nil {
		return m.getContextFn(ctx, id)
	}
	return nil, nil
}

func (m *mockEntityRepo) UpdateContextData(ctx context.Context, id int64, data *string) error {
	if m.setContextFn != nil {
		return m.setContextFn(ctx, id, data)
	}
	return nil
}

// newTestEntityService creates an EntityService scoped to epic level with default workflow config for testing.
func newTestEntityService(t *testing.T) *EntityService {
	t.Helper()
	wfSvc := workflow.NewService("")
	return NewEntityService(wfSvc).ForLevel(workflow.LevelEpic)
}

// --- TransitionStatus Tests ---

func TestEntityService_TransitionStatus_HappyPath(t *testing.T) {
	svc := newTestEntityService(t)
	var updatedID int64
	var updatedStatus string

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 42, Key: "E01", Status: "draft"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			updatedID = id
			updatedStatus = status
			return nil
		},
	}

	result, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "active", TransitionOptions{},
		DefaultTransitionFeatures(), nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.EntityType != models.EntityTypeEpic {
		t.Errorf("expected entity_type 'epic', got %q", result.EntityType)
	}
	if result.EntityKey != "E01" {
		t.Errorf("expected entity_key 'E01', got %q", result.EntityKey)
	}
	if result.EntityID != 42 {
		t.Errorf("expected entity_id 42, got %d", result.EntityID)
	}
	if result.FromStatus != "draft" {
		t.Errorf("expected from_status 'draft', got %q", result.FromStatus)
	}
	if !result.Transitioned {
		t.Error("expected Transitioned=true")
	}
	if updatedID != 42 {
		t.Errorf("expected repo UpdateStatus called with id=42, got %d", updatedID)
	}
	if updatedStatus == "" {
		t.Error("expected repo UpdateStatus called with non-empty status")
	}
}

func TestEntityService_TransitionStatus_EntityNotFound(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("epic not found: %s", key)
		},
	}

	_, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E99", "active", TransitionOptions{},
		DefaultTransitionFeatures(), nil,
	)

	if err == nil {
		t.Fatal("expected error for not-found entity")
	}
}

func TestEntityService_TransitionStatus_EntityNil(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, nil
		},
	}

	_, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E99", "active", TransitionOptions{},
		DefaultTransitionFeatures(), nil,
	)

	if err == nil {
		t.Fatal("expected error for nil entity")
	}
}

func TestEntityService_TransitionStatus_ForcedWithReason(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "completed"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return nil
		},
	}

	result, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "draft",
		TransitionOptions{Force: true, Reason: "emergency rollback"},
		DefaultTransitionFeatures(), nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsForced {
		t.Error("expected IsForced=true")
	}
	if result.Reason != "emergency rollback" {
		t.Errorf("expected reason 'emergency rollback', got %q", result.Reason)
	}
}

func TestEntityService_TransitionStatus_ForcedWithoutReason(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "completed"}, nil
		},
	}

	_, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "draft",
		TransitionOptions{Force: true}, // no reason
		DefaultTransitionFeatures(), nil,
	)

	if !errors.Is(err, ErrForceReasonRequired) {
		t.Errorf("expected ErrForceReasonRequired, got %v", err)
	}
}

func TestEntityService_TransitionStatus_BackwardWithReason(t *testing.T) {
	svc := newTestEntityServiceForBackward(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "active"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return nil
		},
	}

	result, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "draft",
		TransitionOptions{Reason: "requirements changed"},
		DefaultTransitionFeatures(), nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsBackward {
		t.Error("expected IsBackward=true")
	}
}

func TestEntityService_TransitionStatus_BackwardWithoutReason(t *testing.T) {
	svc := newTestEntityServiceForBackward(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "active"}, nil
		},
	}

	_, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "draft",
		TransitionOptions{}, // no reason
		DefaultTransitionFeatures(), nil,
	)

	var backErr *BackwardReasonError
	if !errors.As(err, &backErr) {
		t.Errorf("expected BackwardReasonError, got %T: %v", err, err)
	}
}

func TestEntityService_TransitionStatus_SimpleFeatures_NoBackwardDetection(t *testing.T) {
	svc := newTestEntityServiceForBackward(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "active"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return nil
		},
	}

	// SimpleTransitionFeatures disables backward detection
	result, err := svc.TransitionStatus(
		context.Background(), repo, "bug", "B001", "draft",
		TransitionOptions{Reason: "fixing"},
		SimpleTransitionFeatures(), nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// IsBackward should be false because detection is disabled
	if result.IsBackward {
		t.Error("expected IsBackward=false with SimpleTransitionFeatures")
	}
}

func TestEntityService_TransitionStatus_RepoUpdateError(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "draft"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return fmt.Errorf("database error")
		},
	}

	_, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "active", TransitionOptions{},
		DefaultTransitionFeatures(), nil,
	)

	if err == nil {
		t.Fatal("expected error from repo update failure")
	}
}

func TestEntityService_TransitionStatus_WithResolveActionFn(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "draft"}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return nil
		},
	}

	resolveActionCalled := false
	resolveActionFn := func(entity models.Entity, status string) *config.PopulatedAction {
		resolveActionCalled = true
		return &config.PopulatedAction{Action: "test-action", AgentType: "developer"}
	}

	result, err := svc.TransitionStatus(
		context.Background(), repo, models.EntityTypeEpic, "E01", "active", TransitionOptions{},
		DefaultTransitionFeatures(), resolveActionFn,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolveActionCalled {
		t.Error("expected resolveActionFn to be called")
	}
	if result.OrchestratorAction == nil {
		t.Error("expected non-nil OrchestratorAction")
	}
}

// --- ValidateAndNormalize Tests ---

func TestEntityService_ValidateAndNormalize_Valid(t *testing.T) {
	svc := newTestEntityService(t)

	// "draft" -> "active" is a valid transition in the default epic workflow
	result, err := svc.ValidateAndNormalize("draft", "active", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty normalized status")
	}
}

func TestEntityService_ValidateAndNormalize_Invalid(t *testing.T) {
	svc := newTestEntityService(t)

	_, err := svc.ValidateAndNormalize("draft", "completed", false)
	if err == nil {
		t.Error("expected error for invalid transition")
	}
}

func TestEntityService_ValidateAndNormalize_ForcedSkipsValidation(t *testing.T) {
	svc := newTestEntityService(t)

	result, err := svc.ValidateAndNormalize("draft", "nonexistent_status", true)
	if err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}
	if result != "nonexistent_status" {
		t.Errorf("expected 'nonexistent_status', got %q", result)
	}
}

// --- DetectBackward Tests ---

func TestEntityService_DetectBackward_Forward(t *testing.T) {
	svc := newTestEntityService(t)

	isBackward, err := svc.DetectBackward("draft", "active", false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isBackward {
		t.Error("expected forward transition to not be backward")
	}
}

func TestEntityService_DetectBackward_Backward_WithReason(t *testing.T) {
	svc := newTestEntityServiceForBackward(t)

	isBackward, err := svc.DetectBackward("active", "draft", false, "requirements changed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isBackward {
		t.Error("expected backward transition to be detected")
	}
}

func TestEntityService_DetectBackward_Backward_WithoutReason(t *testing.T) {
	svc := newTestEntityServiceForBackward(t)

	_, err := svc.DetectBackward("active", "draft", false, "")
	var backErr *BackwardReasonError
	if !errors.As(err, &backErr) {
		t.Errorf("expected BackwardReasonError, got %T: %v", err, err)
	}
}

func TestEntityService_DetectBackward_ForceGraceful(t *testing.T) {
	svc := newTestEntityService(t)

	// Force=true with an unknown status that errors on IsBackwardTransition
	isBackward, err := svc.DetectBackward("active", "nonexistent", true, "forced")
	if err != nil {
		t.Fatalf("expected no error with force=true, got %v", err)
	}
	if isBackward {
		t.Error("expected isBackward=false when forced and error occurred")
	}
}

// --- ResolveActionForStatus Tests ---

func TestEntityService_ResolveActionForStatus_NilWorkflow(t *testing.T) {
	// NewService("") gives a default workflow; we can test missing status metadata
	svc := newTestEntityService(t)

	result := svc.ResolveActionForStatus("nonexistent_status", map[string]string{"key": "val"})
	if result != nil {
		t.Error("expected nil for nonexistent status")
	}
}

func TestEntityService_ResolveActionForStatus_ValidStatus(t *testing.T) {
	// Use a workflow that has orchestrator actions
	svc := newTestEntityServiceWithActions(t)

	placeholders := map[string]string{
		"epic_key":   "E01",
		"epic_title": "Test Epic",
	}

	result := svc.ResolveActionForStatus("active", placeholders)
	// Whether result is nil depends on whether the workflow config has actions
	// For this test, we just verify no panic
	_ = result
}

// --- GetNextStatus Tests ---

func TestEntityService_GetNextStatus_HappyPath(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "draft"}, nil
		},
	}

	info, err := svc.GetNextStatus(context.Background(), repo, models.EntityTypeEpic, "E01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil NextStatusInfo")
	}
	if info.EntityType != models.EntityTypeEpic {
		t.Errorf("expected entity_type 'epic', got %q", info.EntityType)
	}
	if info.CurrentStatus != "draft" {
		t.Errorf("expected current_status 'draft', got %q", info.CurrentStatus)
	}
	if len(info.AvailableTransitions) == 0 {
		t.Error("expected at least one available transition from 'draft'")
	}
}

func TestEntityService_GetNextStatus_EntityNotFound(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("epic not found: %s", key)
		},
	}

	_, err := svc.GetNextStatus(context.Background(), repo, models.EntityTypeEpic, "E99", nil)
	if err == nil {
		t.Fatal("expected error for not-found entity")
	}
}

func TestEntityService_GetNextStatus_TerminalStatus(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "completed"}, nil
		},
	}

	info, err := svc.GetNextStatus(context.Background(), repo, models.EntityTypeEpic, "E01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsTerminal {
		t.Error("expected IsTerminal=true for 'completed' status")
	}
}

func TestEntityService_GetNextStatus_WithResolveActionFn(t *testing.T) {
	svc := newTestEntityService(t)

	repo := &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			return &models.Epic{ID: 1, Key: "E01", Status: "draft"}, nil
		},
	}

	callCount := 0
	resolveActionFn := func(entity models.Entity, status string) *config.PopulatedAction {
		callCount++
		return nil
	}

	info, err := svc.GetNextStatus(context.Background(), repo, models.EntityTypeEpic, "E01", resolveActionFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount == 0 && len(info.AvailableTransitions) > 0 {
		t.Error("expected resolveActionFn to be called for each transition")
	}
}

// --- TransitionFeatures Tests ---

func TestDefaultTransitionFeatures(t *testing.T) {
	f := DefaultTransitionFeatures()
	if !f.DetectBackward {
		t.Error("expected DetectBackward=true")
	}
	if !f.CreateRejectionNotes {
		t.Error("expected CreateRejectionNotes=true")
	}
	if !f.ResolveOrchestratorAction {
		t.Error("expected ResolveOrchestratorAction=true")
	}
}

func TestSimpleTransitionFeatures(t *testing.T) {
	f := SimpleTransitionFeatures()
	if f.DetectBackward {
		t.Error("expected DetectBackward=false")
	}
	if f.CreateRejectionNotes {
		t.Error("expected CreateRejectionNotes=false")
	}
	if !f.ResolveOrchestratorAction {
		t.Error("expected ResolveOrchestratorAction=true")
	}
}

// --- Helper: workflow service with orchestrator actions ---

func newTestEntityServiceWithActions(t *testing.T) *EntityService {
	t.Helper()
	wfSvc := workflow.NewService("")
	return NewEntityService(wfSvc).ForLevel(workflow.LevelEpic)
}

// newTestEntityServiceForBackward creates an EntityService with a workflow config
// that allows backward transitions (active -> draft) for testing backward detection.
func newTestEntityServiceForBackward(t *testing.T) *EntityService {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
		"epic_workflow": {
			"status_flow_version": "1.0",
			"require_rejection_reason": true,
			"status_flow": {
				"draft": ["active"],
				"active": ["draft", "completed", "archived"],
				"completed": ["archived"],
				"archived": []
			},
			"status_metadata": {
				"draft": {
					"color": "gray",
					"description": "Epic created",
					"phase": "planning"
				},
				"active": {
					"color": "blue",
					"description": "Epic in progress",
					"phase": "development"
				},
				"completed": {
					"color": "green",
					"description": "All features complete",
					"phase": "done"
				},
				"archived": {
					"color": "gray",
					"description": "Epic archived",
					"phase": "done"
				}
			},
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed", "archived"]
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	config.ClearWorkflowCache()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	wfSvc := workflow.NewService(tmpDir)
	return NewEntityService(wfSvc).ForLevel(workflow.LevelEpic)
}
