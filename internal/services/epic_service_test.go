package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockEpicRepo implements EpicRepository for testing.
type mockEpicRepo struct {
	getByKeyFn                     func(ctx context.Context, key string) (*models.Epic, error)
	updateFn                       func(ctx context.Context, epic *models.Epic) error
	listFn                         func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	getFeatureProgressDataByEpicFn func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	getFeatureStatusBreakdownFn    func(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error)
	getFeatureStatusRollupFn       func(ctx context.Context, epicID int64) (map[string]int, error)
	getTaskStatusRollupFn          func(ctx context.Context, epicID int64) (map[string]int, error)
}

func (m *mockEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockEpicRepo) Update(ctx context.Context, epic *models.Epic) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, epic)
	}
	return nil
}

func (m *mockEpicRepo) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	if m.listFn != nil {
		return m.listFn(ctx, status)
	}
	return nil, nil
}

func (m *mockEpicRepo) GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
	if m.getFeatureProgressDataByEpicFn != nil {
		return m.getFeatureProgressDataByEpicFn(ctx, epicID)
	}
	return nil, nil
}

func (m *mockEpicRepo) GetFeatureStatusBreakdownByKey(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error) {
	if m.getFeatureStatusBreakdownFn != nil {
		return m.getFeatureStatusBreakdownFn(ctx, epicKey)
	}
	return nil, nil
}

func (m *mockEpicRepo) GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
	if m.getFeatureStatusRollupFn != nil {
		return m.getFeatureStatusRollupFn(ctx, epicID)
	}
	return nil, nil
}

func (m *mockEpicRepo) GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
	if m.getTaskStatusRollupFn != nil {
		return m.getTaskStatusRollupFn(ctx, epicID)
	}
	return nil, nil
}

// newTestEpicWorkflowService creates a workflow.Service with default config for testing.
func newTestEpicWorkflowService() *workflow.Service {
	return workflow.NewService("")
}

// newTestEpicWorkflowServiceWithActions creates a workflow.Service backed by a temp config
// that has orchestrator_action defined on the "active" status of the epic workflow.
// The "draft" status has no orchestrator_action, allowing tests to verify both branches.
func newTestEpicWorkflowServiceWithActions(t *testing.T) *workflow.Service {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
		"epic_workflow": {
			"status_flow_version": "1.0",
			"status_flow": {
				"draft": ["active", "archived"],
				"active": ["completed", "archived"],
				"completed": ["archived"],
				"archived": []
			},
			"status_metadata": {
				"draft": {
					"color": "gray",
					"description": "Epic created, not yet started",
					"phase": "planning"
				},
				"active": {
					"color": "blue",
					"description": "Epic in progress",
					"phase": "execution",
					"orchestrator_action": {
						"action": "spawn_agent",
						"agent_type": "developer",
						"skills": ["implementation", "testing"],
						"instruction_template": "Work on epic {id} features"
					}
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

	// Clear workflow cache so fresh config is loaded
	config.ClearWorkflowCache()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return workflow.NewService(tmpDir)
}

func TestEpicService_TransitionStatus_Valid(t *testing.T) {
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", result.EntityType)
	}
	if result.EntityKey != "E16" {
		t.Errorf("expected entity_key 'E16', got %q", result.EntityKey)
	}
	if result.FromStatus != "draft" {
		t.Errorf("expected from_status 'draft', got %q", result.FromStatus)
	}
	if result.ToStatus != "active" {
		t.Errorf("expected to_status 'active', got %q", result.ToStatus)
	}
	if !result.Transitioned {
		t.Error("expected transitioned=true")
	}
	if updatedEpic == nil {
		t.Fatal("expected Update to be called")
	}
	if string(updatedEpic.Status) != "active" {
		t.Errorf("expected epic status 'active', got %q", updatedEpic.Status)
	}
}

func TestEpicService_TransitionStatus_Invalid(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	// "draft" -> "completed" is not a valid direct transition in default epic workflow
	_, err := svc.TransitionStatus(ctx, "E16", "completed", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid transition, got nil")
	}
}

func TestEpicService_TransitionStatus_Force(t *testing.T) {
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	// Force should bypass validation - even arbitrary strings should work
	result, err := svc.TransitionStatus(ctx, "E16", "custom_status", TransitionOptions{Force: true, Reason: "test force override"})
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}
	if result.ToStatus != "custom_status" {
		t.Errorf("expected to_status 'custom_status', got %q", result.ToStatus)
	}
	if updatedEpic == nil {
		t.Fatal("expected Update to be called")
	}
	if string(updatedEpic.Status) != "custom_status" {
		t.Errorf("expected epic status 'custom_status', got %q", updatedEpic.Status)
	}
}

func TestEpicService_TransitionStatus_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E99", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
	if err.Error() != "epic not found: E99" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEpicService_TransitionStatus_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

func TestEpicService_TransitionStatus_UpdateError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from update failure")
	}
}

func TestEpicService_GetNextStatus(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if info.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", info.EntityType)
	}
	if info.CurrentStatus != "draft" {
		t.Errorf("expected current_status 'draft', got %q", info.CurrentStatus)
	}
	if info.IsTerminal {
		t.Error("expected IsTerminal=false for draft status")
	}
	if len(info.AvailableTransitions) == 0 {
		t.Error("expected available transitions for draft status")
	}
}

func TestEpicService_GetNextStatus_Terminal(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusArchived,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !info.IsTerminal {
		t.Error("expected IsTerminal=true for archived status")
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no transitions for terminal status, got %d", len(info.AvailableTransitions))
	}
}

func TestEpicService_GetNextStatus_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_ValidateStatus(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	// Valid epic statuses
	for _, status := range []string{"draft", "active", "completed", "archived"} {
		if err := svc.ValidateStatus(status); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", status, err)
		}
	}

	// Invalid status
	if err := svc.ValidateStatus("in_progress"); err == nil {
		t.Error("expected 'in_progress' to be invalid for epic workflow")
	}
}

func TestTransitionResult_JSONSerialization(t *testing.T) {
	result := &TransitionResult{
		EntityType:   "epic",
		EntityKey:    "E16",
		FromStatus:   "draft",
		ToStatus:     "active",
		Transitioned: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", parsed.EntityType)
	}
	if parsed.FromStatus != "draft" {
		t.Errorf("expected from_status 'draft', got %q", parsed.FromStatus)
	}
	if parsed.ToStatus != "active" {
		t.Errorf("expected to_status 'active', got %q", parsed.ToStatus)
	}
}

func TestNextStatusInfo_JSONSerialization(t *testing.T) {
	info := &NextStatusInfo{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "active",
					Phase:        "execution",
				},
			},
		},
		IsTerminal: false,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed NextStatusInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", parsed.EntityType)
	}
	if len(parsed.AvailableTransitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(parsed.AvailableTransitions))
	}
	if parsed.AvailableTransitions[0].TargetStatus != "active" {
		t.Errorf("expected target_status 'active', got %q", parsed.AvailableTransitions[0].TargetStatus)
	}
}

// --- mockEpicTaskLister implements EpicTaskLister for testing ---

type mockEpicTaskLister struct {
	listBlockedTasksByEpicFn func(ctx context.Context, epicKey string) ([]*models.Task, error)
}

func (m *mockEpicTaskLister) ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	if m.listBlockedTasksByEpicFn != nil {
		return m.listBlockedTasksByEpicFn(ctx, epicKey)
	}
	return nil, nil
}

// --- Tests for GetEpic ---

func TestEpicService_GetEpic(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			if key != "E01" {
				t.Errorf("expected key 'E01', got %q", key)
			}
			return &models.Epic{ID: 1, Key: "E01", Title: "Test Epic", Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	epic, err := svc.GetEpic(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if epic.Key != "E01" {
		t.Errorf("expected key 'E01', got %q", epic.Key)
	}
}

func TestEpicService_GetEpic_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetEpic(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_GetEpic_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetEpic(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for ListEpics ---

func TestEpicService_ListEpics(t *testing.T) {
	repo := &mockEpicRepo{
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			if status != nil {
				t.Errorf("expected nil status filter, got %v", *status)
			}
			return []*models.Epic{
				{Key: "E01", Title: "Epic 1"},
				{Key: "E02", Title: "Epic 2"},
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	epics, err := svc.ListEpics(context.Background(), EpicFilters{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(epics))
	}
}

func TestEpicService_ListEpics_WithStatusFilter(t *testing.T) {
	repo := &mockEpicRepo{
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			if status == nil || *status != models.EpicStatusActive {
				t.Errorf("expected active status filter")
			}
			return []*models.Epic{{Key: "E01"}}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	epics, err := svc.ListEpics(context.Background(), EpicFilters{Status: "active"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(epics))
	}
}

func TestEpicService_ListEpics_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.ListEpics(context.Background(), EpicFilters{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetProgress ---

func TestEpicService_GetProgress(t *testing.T) {
	// 3 features: 2 active (at 26.665% each), 1 completed (treated as 100%)
	// Progress = (26.665 + 26.665 + 100) / 3 = 51.11
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return []repository.FeatureProgressData{
				{Status: "active", ProgressPct: 26.665},
				{Status: "active", ProgressPct: 26.665},
				{Status: "completed", ProgressPct: 50.0}, // stored value ignored; treated as 100%
			}, nil
		},
		getFeatureStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			return map[string]int{"active": 2, "completed": 1}, nil
		},
		getTaskStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			return map[string]int{"todo": 3, "in_development": 2, "completed": 5}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	progress, err := svc.GetProgress(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress.EpicKey != "E01" {
		t.Errorf("expected epic_key 'E01', got %q", progress.EpicKey)
	}
	if progress.TotalFeatures != 3 {
		t.Errorf("expected 3 total features, got %d", progress.TotalFeatures)
	}
	// (26.665 + 26.665 + 100) / 3 = 51.11
	if progress.ProgressPct != 51.11 {
		t.Errorf("expected progress_pct 51.11, got %v", progress.ProgressPct)
	}
	if progress.TaskRollup["completed"] != 5 {
		t.Errorf("expected 5 completed tasks, got %d", progress.TaskRollup["completed"])
	}
}

func TestEpicService_GetProgress_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetProgress(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_GetProgress_CalculateError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return nil, fmt.Errorf("calculation failed")
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetProgress(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for CalculateProgress ---

func TestEpicService_CalculateProgress_NoFeatures(t *testing.T) {
	repo := &mockEpicRepo{
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return []repository.FeatureProgressData{}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	progress, err := svc.CalculateProgress(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress != 0 {
		t.Errorf("expected 0 progress for no features, got %v", progress)
	}
}

func TestEpicService_CalculateProgress_NilFeatures(t *testing.T) {
	repo := &mockEpicRepo{
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	progress, err := svc.CalculateProgress(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress != 0 {
		t.Errorf("expected 0 progress for nil features, got %v", progress)
	}
}

func TestEpicService_CalculateProgress_AllCompleted(t *testing.T) {
	repo := &mockEpicRepo{
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return []repository.FeatureProgressData{
				{Status: "completed", ProgressPct: 80.0},
				{Status: "completed", ProgressPct: 0.0},
				{Status: "archived", ProgressPct: 50.0},
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	progress, err := svc.CalculateProgress(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress != 100.0 {
		t.Errorf("expected 100.0 progress for all completed/archived, got %v", progress)
	}
}

func TestEpicService_CalculateProgress_MixedStatuses(t *testing.T) {
	repo := &mockEpicRepo{
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return []repository.FeatureProgressData{
				{Status: "active", ProgressPct: 50.0},
				{Status: "completed", ProgressPct: 75.0}, // treated as 100%
				{Status: "draft", ProgressPct: 0.0},
				{Status: "archived", ProgressPct: 10.0}, // treated as 100%
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	progress, err := svc.CalculateProgress(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// (50 + 100 + 0 + 100) / 4 = 62.5
	if progress != 62.5 {
		t.Errorf("expected 62.5 progress, got %v", progress)
	}
}

func TestEpicService_CalculateProgress_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.CalculateProgress(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetFeatureRollup ---

func TestEpicService_GetFeatureRollup(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
		getFeatureStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			if epicID != 1 {
				t.Errorf("expected epicID 1, got %d", epicID)
			}
			return map[string]int{"active": 3, "completed": 2}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	rollup, err := svc.GetFeatureRollup(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if rollup.TotalFeatures != 5 {
		t.Errorf("expected 5 total features, got %d", rollup.TotalFeatures)
	}
	if rollup.StatusCounts["active"] != 3 {
		t.Errorf("expected 3 active, got %d", rollup.StatusCounts["active"])
	}
}

func TestEpicService_GetFeatureRollup_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetFeatureRollup(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetTaskStatusRollup ---

func TestEpicService_GetTaskStatusRollup(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
		getTaskStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			return map[string]int{"todo": 5, "completed": 10}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	rollup, err := svc.GetTaskStatusRollup(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if rollup["todo"] != 5 {
		t.Errorf("expected 5 todo tasks, got %d", rollup["todo"])
	}
	if rollup["completed"] != 10 {
		t.Errorf("expected 10 completed tasks, got %d", rollup["completed"])
	}
}

func TestEpicService_GetTaskStatusRollup_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetTaskStatusRollup(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetImpediments ---

func TestEpicService_GetImpediments(t *testing.T) {
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			if epicKey != "E01" {
				t.Errorf("expected epicKey 'E01', got %q", epicKey)
			}
			return []*models.Task{
				{Key: "T-E01-F01-001", Title: "Blocked task", Status: "blocked", Priority: 5, UpdatedAt: time.Now().Add(-48 * time.Hour)},
			}, nil
		},
	}
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	impediments, err := svc.GetImpediments(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(impediments) != 1 {
		t.Fatalf("expected 1 impediment, got %d", len(impediments))
	}
	if impediments[0].TaskKey != "T-E01-F01-001" {
		t.Errorf("expected task key 'T-E01-F01-001', got %q", impediments[0].TaskKey)
	}
	if impediments[0].AgeDays < 1 {
		t.Errorf("expected age_days >= 1, got %d", impediments[0].AgeDays)
	}
}

func TestEpicService_GetImpediments_NilTaskRepo(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	impediments, err := svc.GetImpediments(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(impediments) != 0 {
		t.Errorf("expected empty impediments when taskRepo is nil, got %d", len(impediments))
	}
}

func TestEpicService_GetImpediments_RepoError(t *testing.T) {
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	_, err := svc.GetImpediments(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetHealth ---

func TestEpicService_GetHealth_Healthy(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	health, err := svc.GetHealth(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected 'healthy', got %q", health.Status)
	}
	if len(health.Reasons) != 0 {
		t.Errorf("expected no reasons, got %v", health.Reasons)
	}
}

func TestEpicService_GetHealth_Warning(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E01-F01-001", Title: "Blocked", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	health, err := svc.GetHealth(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "warning" {
		t.Errorf("expected 'warning', got %q", health.Status)
	}
}

func TestEpicService_GetHealth_Critical_MultipleBlocked(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E01-F01-001", Title: "Blocked 1", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
				{Key: "T-E01-F01-002", Title: "Blocked 2", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	health, err := svc.GetHealth(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "critical" {
		t.Errorf("expected 'critical', got %q", health.Status)
	}
}

func TestEpicService_GetHealth_Critical_HighPriority(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E01-F01-001", Title: "High-pri blocked", Status: "blocked", Priority: 2, UpdatedAt: time.Now()},
			}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, taskLister)

	health, err := svc.GetHealth(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "critical" {
		t.Errorf("expected 'critical' for high-priority blocked task, got %q", health.Status)
	}
}

func TestEpicService_GetHealth_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	_, err := svc.GetHealth(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEpicService_GetHealth_NilTaskRepo(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01"}, nil
		},
	}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	health, err := svc.GetHealth(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected 'healthy' when taskRepo is nil, got %q", health.Status)
	}
}

// --- Tests for resolveAction integration ---

func TestEpicService_TransitionStatus_WithAction(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	// Transition to "active" which has an orchestrator_action defined
	result, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction to be populated for 'active' status")
	}
	if result.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", result.OrchestratorAction.Action)
	}
	if result.OrchestratorAction.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", result.OrchestratorAction.AgentType)
	}
	if len(result.OrchestratorAction.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(result.OrchestratorAction.Skills))
	}
	// Verify template was populated with entity key
	expectedInstruction := "Work on epic E16 features"
	if result.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, result.OrchestratorAction.Instruction)
	}
}

func TestEpicService_TransitionStatus_WithoutAction(t *testing.T) {
	// Start from "active" to transition to "completed" (no action defined on completed)
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusActive,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	// Transition to "completed" which has no orchestrator_action
	result, err := svc.TransitionStatus(ctx, "E16", "completed", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction for 'completed' status, got %+v", result.OrchestratorAction)
	}
}

func TestEpicService_GetNextStatus_WithActions(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// From draft, transitions are: active (has action), archived (no action)
	if len(info.AvailableTransitions) == 0 {
		t.Fatal("expected available transitions for draft status")
	}

	// Find the "active" transition
	var activeTransition *TransitionInfoWithAction
	var archivedTransition *TransitionInfoWithAction
	for i := range info.AvailableTransitions {
		if info.AvailableTransitions[i].TargetStatus == "active" {
			activeTransition = &info.AvailableTransitions[i]
		}
		if info.AvailableTransitions[i].TargetStatus == "archived" {
			archivedTransition = &info.AvailableTransitions[i]
		}
	}

	if activeTransition == nil {
		t.Fatal("expected transition to 'active' status")
	}
	if activeTransition.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction on 'active' transition")
	}
	if activeTransition.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", activeTransition.OrchestratorAction.Action)
	}
	// Template should be populated with the entity key
	expectedInstruction := "Work on epic E16 features"
	if activeTransition.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, activeTransition.OrchestratorAction.Instruction)
	}

	if archivedTransition == nil {
		t.Fatal("expected transition to 'archived' status")
	}
	if archivedTransition.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction on 'archived' transition, got %+v", archivedTransition.OrchestratorAction)
	}
}

func TestEpicService_resolveAction_NilWorkflow(t *testing.T) {
	// Create an EpicService with a default workflow (no actions defined)
	// resolveAction should return nil gracefully
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService(), nil, nil, nil)

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "draft"}
	action := svc.resolveAction(context.Background(), epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for default workflow (no actions defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_NonexistentStatus(t *testing.T) {
	// resolveAction with a status not in metadata should return nil
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "nonexistent_status"}
	action := svc.resolveAction(context.Background(), epic, "nonexistent_status")
	if action != nil {
		t.Errorf("expected nil action for nonexistent status, got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithoutAction(t *testing.T) {
	// resolveAction for a status that exists but has no orchestrator_action
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "draft"}
	action := svc.resolveAction(context.Background(), epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for 'draft' status (no action defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithAction(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "active"}
	action := svc.resolveAction(context.Background(), epic, "active")
	if action == nil {
		t.Fatal("expected non-nil action for 'active' status")
	}
	if action.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", action.Action)
	}
	if action.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", action.AgentType)
	}
	if len(action.Skills) != 2 || action.Skills[0] != "implementation" || action.Skills[1] != "testing" {
		t.Errorf("expected skills [implementation, testing], got %v", action.Skills)
	}
	expectedInstruction := "Work on epic E16 features"
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, action.Instruction)
	}
}

func TestEpicService_TransitionStatus_ActionJSON(t *testing.T) {
	// Verify the action is properly serialized in JSON output
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.OrchestratorAction == nil {
		t.Fatal("expected orchestrator_action in JSON output")
	}
	if parsed.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent' in JSON, got %q", parsed.OrchestratorAction.Action)
	}
}
