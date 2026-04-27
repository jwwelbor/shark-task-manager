package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockEpicRepo implements EpicRepository for testing.
type mockEpicRepo struct {
	getByKeyFn                        func(ctx context.Context, key string) (*models.Epic, error)
	updateFn                          func(ctx context.Context, epic *models.Epic) error
	updateStatusFn                    func(ctx context.Context, epicID int64, status models.EpicStatus) error
	listFn                            func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	getFeatureProgressDataByEpicFn    func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	getFeatureStatusBreakdownFn       func(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error)
	getFeatureStatusRollupFn          func(ctx context.Context, epicID int64) (map[string]int, error)
	getTaskStatusRollupFn             func(ctx context.Context, epicID int64) (map[string]int, error)
	createFn                          func(ctx context.Context, epic *models.Epic) error
	deleteFn                          func(ctx context.Context, id int64) error
	cascadeStatusToFeaturesAndTasksFn func(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error
	getEpicDisplayDataRawFn           func(ctx context.Context, epicID int64) (*repository.EpicDisplayDataRaw, error)
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

func (m *mockEpicRepo) Create(ctx context.Context, epic *models.Epic) error {
	if m.createFn != nil {
		return m.createFn(ctx, epic)
	}
	return nil
}

func (m *mockEpicRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockEpicRepo) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
	return nil, nil
}

func (m *mockEpicRepo) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error {
	return nil
}

func (m *mockEpicRepo) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	return nil
}

func (m *mockEpicRepo) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	return nil, nil
}

func (m *mockEpicRepo) GetFeatureStatusBreakdown(ctx context.Context, epicID int64) (map[models.FeatureStatus]int, error) {
	return nil, nil
}

func (m *mockEpicRepo) UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, epicID, status)
	}
	return nil
}

func (m *mockEpicRepo) CascadeStatusToFeaturesAndTasks(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error {
	if m.cascadeStatusToFeaturesAndTasksFn != nil {
		return m.cascadeStatusToFeaturesAndTasksFn(ctx, epicID, targetFeatureStatus, targetTaskStatus)
	}
	return nil
}

func (m *mockEpicRepo) CascadeStatusToFeaturesAndTasksWithTx(_ context.Context, _ *sql.Tx, _ int64, _ models.FeatureStatus, _ models.TaskStatus) error {
	return nil
}

func (m *mockEpicRepo) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (m *mockEpicRepo) GetEpicDisplayDataRaw(ctx context.Context, epicID int64) (*repository.EpicDisplayDataRaw, error) {
	if m.getEpicDisplayDataRawFn != nil {
		return m.getEpicDisplayDataRawFn(ctx, epicID)
	}
	return &repository.EpicDisplayDataRaw{
		FeaturesJSON:      "[]",
		TaskBreakdownJSON: "[]",
		BlockedTasksJSON:  "[]",
		DocumentsJSON:     "[]",
		NotesJSON:         "[]",
	}, nil
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
	statusUpdated := false
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key: "E16"}, Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			statusUpdated = true
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
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
	// Status update goes through UpdateStatus (via EntityService), not Update
	_ = statusUpdated
}

func TestEpicService_TransitionStatus_Invalid(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	ctx := context.Background()

	// Force should bypass validation - even arbitrary strings should work
	result, err := svc.TransitionStatus(ctx, "E16", "custom_status", TransitionOptions{Force: true, Reason: "test force override"})
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}
	if result.ToStatus != "custom_status" {
		t.Errorf("expected to_status 'custom_status', got %q", result.ToStatus)
	}
	// Status update goes through UpdateStatus (via EntityService), not Update
	_ = updatedEpic
}

func TestEpicService_TransitionStatus_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
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

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

func TestEpicService_TransitionStatus_UpdateError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key: "E16"}, Status: models.EpicStatusDraft,
			}, nil
		},
		updateStatusFn: func(ctx context.Context, epicID int64, status models.EpicStatus) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from update failure")
	}
}

func TestEpicService_GetNextStatus(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusArchived}, nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
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

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_ValidateStatus(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	listBlockedTasksByEpicFn     func(ctx context.Context, epicKey string) ([]*models.Task, error)
	listByFeatureFn              func(ctx context.Context, featureID int64) ([]*models.Task, error)
	updateStatusForcedFn         func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	getStatusBreakdownMapBatchFn func(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error)
	getTaskCountsForFeaturesFn   func(ctx context.Context, featureIDs []int64) (map[int64]int, error)
}

func (m *mockEpicTaskLister) ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	if m.listBlockedTasksByEpicFn != nil {
		return m.listBlockedTasksByEpicFn(ctx, epicKey)
	}
	return nil, nil
}

func (m *mockEpicTaskLister) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureID)
	}
	return nil, nil
}

func (m *mockEpicTaskLister) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	if m.updateStatusForcedFn != nil {
		return m.updateStatusForcedFn(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	}
	return nil
}

func (m *mockEpicTaskLister) GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if m.getStatusBreakdownMapBatchFn != nil {
		return m.getStatusBreakdownMapBatchFn(ctx, featureIDs)
	}
	return make(map[int64]map[models.TaskStatus]int), nil
}

func (m *mockEpicTaskLister) GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	if m.getTaskCountsForFeaturesFn != nil {
		return m.getTaskCountsForFeaturesFn(ctx, featureIDs)
	}
	return make(map[int64]int), nil
}

// --- Tests for GetEpic ---

func TestEpicService_GetEpic(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			if key != "E01" {
				t.Errorf("expected key 'E01', got %q", key)
			}
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Test Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
				{BaseEntity: models.BaseEntity{Key: "E01", Title: "Epic 1"}},
				{BaseEntity: models.BaseEntity{Key: "E02", Title: "Epic 2"}},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
			return []*models.Epic{{BaseEntity: models.BaseEntity{Key: "E01"}}}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.GetProgress(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_GetProgress_CalculateError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
		getFeatureProgressDataByEpicFn: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
			return nil, fmt.Errorf("calculation failed")
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.CalculateProgress(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetFeatureRollup ---

func TestEpicService_GetFeatureRollup(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
		getFeatureStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			if epicID != 1 {
				t.Errorf("expected epicID 1, got %d", epicID)
			}
			return map[string]int{"active": 3, "completed": 2}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.GetFeatureRollup(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetTaskStatusRollup ---

func TestEpicService_GetTaskStatusRollup(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
		getTaskStatusRollupFn: func(ctx context.Context, epicID int64) (map[string]int, error) {
			return map[string]int{"todo": 5, "completed": 10}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
				{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "Blocked task"}, Status: "blocked", Priority: 5},
			}, nil
		},
	}
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

	_, err := svc.GetImpediments(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Tests for GetHealth ---

func TestEpicService_GetHealth_Healthy(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "Blocked"}, Status: "blocked", Priority: 5},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "Blocked 1"}, Status: "blocked", Priority: 5},
				{BaseEntity: models.BaseEntity{Key: "T-E01-F01-002", Title: "Blocked 2"}, Status: "blocked", Priority: 5},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listBlockedTasksByEpicFn: func(ctx context.Context, epicKey string) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "High-pri blocked"}, Status: "blocked", Priority: 2},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, taskLister)

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
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.GetHealth(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEpicService_GetHealth_NilTaskRepo(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)
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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusActive}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)
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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)
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
	// makeResolveActionFn callback should return nil gracefully
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E16", Title: "Test Epic"}, Status: "draft"}
	resolveFn := svc.makeResolveActionFn(context.Background())
	action := resolveFn(epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for default workflow (no actions defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_NonexistentStatus(t *testing.T) {
	// resolveAction with a status not in metadata should return nil
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E16", Title: "Test Epic"}, Status: "nonexistent_status"}
	resolveFn := svc.makeResolveActionFn(context.Background())
	action := resolveFn(epic, "nonexistent_status")
	if action != nil {
		t.Errorf("expected nil action for nonexistent status, got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithoutAction(t *testing.T) {
	// resolveAction for a status that exists but has no orchestrator_action
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E16", Title: "Test Epic"}, Status: "draft"}
	resolveFn := svc.makeResolveActionFn(context.Background())
	action := resolveFn(epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for 'draft' status (no action defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithAction(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E16", Title: "Test Epic"}, Status: "active"}
	resolveFn := svc.makeResolveActionFn(context.Background())
	action := resolveFn(epic, "active")
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
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E16"}, Status: models.EpicStatusDraft}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowServiceWithActions(t)), epicRepoAsEntityRepo(repo), nil, nil)
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

// --- Tests for CreateEpic ---

func TestEpicService_CreateEpic_Success(t *testing.T) {
	var capturedEpic *models.Epic
	repo := &mockEpicRepo{
		// getByKeyFn returns nil to indicate no existing epic with auto-generated key
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			// Called during nextEpicKey to list existing epics; also for custom key check
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			// Return empty list so nextEpicKey generates E01
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	input := CreateEpicInput{
		Title: "My New Epic",
	}
	epic, err := svc.CreateEpic(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if epic == nil {
		t.Fatal("expected non-nil epic")
	}
	if epic.Title != "My New Epic" {
		t.Errorf("expected title 'My New Epic', got %q", epic.Title)
	}
	if capturedEpic == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if capturedEpic.Title != "My New Epic" {
		t.Errorf("expected captured epic title 'My New Epic', got %q", capturedEpic.Title)
	}
}

func TestEpicService_CreateEpic_EmptyTitle(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.CreateEpic(context.Background(), CreateEpicInput{Title: "   "})
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestEpicService_CreateEpic_CustomKey(t *testing.T) {
	var capturedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			// Return nil to indicate key doesn't exist yet
			return nil, fmt.Errorf("not found")
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	input := CreateEpicInput{
		Title:     "Custom Key Epic",
		CustomKey: "E99",
	}
	epic, err := svc.CreateEpic(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if epic.Key != "E99" {
		t.Errorf("expected key 'E99', got %q", epic.Key)
	}
	if capturedEpic == nil {
		t.Fatal("expected repo.Create to be called")
	}
}

func TestEpicService_CreateEpic_DuplicateCustomKey(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			// Return existing epic for duplicate key check
			return &models.Epic{BaseEntity: models.BaseEntity{Key: "E99"}}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.CreateEpic(context.Background(), CreateEpicInput{
		Title:     "Duplicate Epic",
		CustomKey: "E99",
	})
	if err == nil {
		t.Fatal("expected error for duplicate key, got nil")
	}
}

func TestEpicService_CreateEpic_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			return fmt.Errorf("db write failed")
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.CreateEpic(context.Background(), CreateEpicInput{Title: "Failing Epic"})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// --- Tests for UpdateEpic ---

func TestEpicService_UpdateEpic_Success(t *testing.T) {
	existing := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Old Title"}, Status: models.EpicStatusActive, Priority: models.Priority("medium")}
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	newTitle := "New Title"
	result, err := svc.UpdateEpic(context.Background(), "E01", EpicUpdates{Title: &newTitle})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "New Title" {
		t.Errorf("expected updated title 'New Title', got %q", result.Title)
	}
	if updatedEpic == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if updatedEpic.Title != "New Title" {
		t.Errorf("expected updated epic title 'New Title', got %q", updatedEpic.Title)
	}
}

func TestEpicService_UpdateEpic_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil // Not found (nil, nil pattern)
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	newTitle := "Whatever"
	_, err := svc.UpdateEpic(context.Background(), "E99", EpicUpdates{Title: &newTitle})
	if err == nil {
		t.Fatal("expected error for non-existent epic")
	}
}

func TestEpicService_UpdateEpic_EmptyTitle(t *testing.T) {
	existing := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Original"}, Status: models.EpicStatusActive}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return existing, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	emptyTitle := "   "
	_, err := svc.UpdateEpic(context.Background(), "E01", EpicUpdates{Title: &emptyTitle})
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestEpicService_UpdateEpic_RepoError(t *testing.T) {
	existing := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Title"}, Status: models.EpicStatusActive}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return fmt.Errorf("db write failed")
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	newTitle := "New Title"
	_, err := svc.UpdateEpic(context.Background(), "E01", EpicUpdates{Title: &newTitle})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// --- Tests for DeleteEpic ---

func TestEpicService_DeleteEpic_Success(t *testing.T) {
	var deletedID int64
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 42, Key: "E01", Title: "To Delete"}, Status: models.EpicStatusActive}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.DeleteEpic(context.Background(), "E01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if deletedID != 42 {
		t.Errorf("expected deleted ID 42, got %d", deletedID)
	}
}

func TestEpicService_DeleteEpic_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.DeleteEpic(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for non-existent epic")
	}
}

func TestEpicService_DeleteEpic_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			return fmt.Errorf("db delete failed")
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.DeleteEpic(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// --- Tests for CompleteEpic ---

func TestEpicService_CompleteEpic_NoFeatures(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	featureCounter := &mockEpicFeatureCounter{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
	}
	taskLister := &mockEpicTaskLister{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), featureCounter, taskLister)

	result, err := svc.CompleteEpic(context.Background(), "E01", false, "agent1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.FeatureCount != 0 {
		t.Errorf("expected 0 features, got %d", result.FeatureCount)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected 0 total tasks, got %d", result.TotalCount)
	}
}

func TestEpicService_CompleteEpic_AllTasksComplete_NoForce(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}
	featureCounter := &mockEpicFeatureCounter{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Feature 1"}, Status: models.FeatureStatusActive},
			}, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: id, Key: "E01-F01"}, Status: models.FeatureStatusActive}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				models.TaskStatus("completed"): 3,
			}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{ID: 100, Key: "T-E01-F01-001"}, Status: models.TaskStatus("completed")},
				{BaseEntity: models.BaseEntity{ID: 101, Key: "T-E01-F01-002"}, Status: models.TaskStatus("completed")},
				{BaseEntity: models.BaseEntity{ID: 102, Key: "T-E01-F01-003"}, Status: models.TaskStatus("completed")},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), featureCounter, taskLister)

	result, err := svc.CompleteEpic(context.Background(), "E01", false, "agent1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.RequiresForce {
		t.Error("expected RequiresForce=false when all tasks complete")
	}
	if result.CompletedCount != 3 {
		t.Errorf("expected 3 completed tasks, got %d", result.CompletedCount)
	}
}

func TestEpicService_CompleteEpic_IncompleteTasksRequireForce(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	featureCounter := &mockEpicFeatureCounter{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Feature 1"}, Status: models.FeatureStatusActive},
			}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{ID: 100, Key: "T-E01-F01-001", Title: "Incomplete task"}, Status: models.TaskStatus("in_progress")},
				{BaseEntity: models.BaseEntity{ID: 101, Key: "T-E01-F01-002", Title: "Done task"}, Status: models.TaskStatus("completed")},
			}, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), featureCounter, taskLister)

	result, err := svc.CompleteEpic(context.Background(), "E01", false, "agent1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.RequiresForce {
		t.Error("expected RequiresForce=true when tasks are incomplete")
	}
}

func TestEpicService_CompleteEpic_Force(t *testing.T) {
	forcedTaskIDs := make([]int64, 0)
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}
	featureCounter := &mockEpicFeatureCounter{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Feature 1"}, Status: models.FeatureStatusActive},
			}, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: id, Key: "E01-F01"}, Status: models.FeatureStatusActive}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{"completed": 2}, nil
		},
	}
	taskLister := &mockEpicTaskLister{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{ID: 100, Key: "T-E01-F01-001", Title: "Running task"}, Status: models.TaskStatus("in_progress")},
				{BaseEntity: models.BaseEntity{ID: 101, Key: "T-E01-F01-002", Title: "Done task"}, Status: models.TaskStatus("completed")},
			}, nil
		},
		updateStatusForcedFn: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
			forcedTaskIDs = append(forcedTaskIDs, taskID)
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), featureCounter, taskLister)

	result, err := svc.CompleteEpic(context.Background(), "E01", true, "agent1")
	if err != nil {
		t.Fatalf("expected no error with force=true, got: %v", err)
	}
	if result.RequiresForce {
		t.Error("expected RequiresForce=false after forced completion")
	}
	// Only the non-completed task (ID 100) should be force-completed
	if len(forcedTaskIDs) != 1 || forcedTaskIDs[0] != 100 {
		t.Errorf("expected only task 100 to be force-completed, got: %v", forcedTaskIDs)
	}
}

func TestEpicService_CompleteEpic_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	_, err := svc.CompleteEpic(context.Background(), "E99", false, "agent1")
	if err == nil {
		t.Fatal("expected error for non-existent epic")
	}
}

// --- Tests for CascadeStatusToFeaturesAndTasks ---

func TestEpicService_CascadeStatusToFeaturesAndTasks_Success(t *testing.T) {
	var capturedEpicID int64
	var capturedFeatureStatus models.FeatureStatus
	var capturedTaskStatus models.TaskStatus

	repo := &mockEpicRepo{
		cascadeStatusToFeaturesAndTasksFn: func(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error {
			capturedEpicID = epicID
			capturedFeatureStatus = targetFeatureStatus
			capturedTaskStatus = targetTaskStatus
			return nil
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.CascadeStatusToFeaturesAndTasks(
		context.Background(), 1,
		models.FeatureStatus("completed"),
		models.TaskStatus("completed"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedEpicID != 1 {
		t.Errorf("expected epicID 1, got %d", capturedEpicID)
	}
	if capturedFeatureStatus != models.FeatureStatus("completed") {
		t.Errorf("expected featureStatus 'completed', got %q", capturedFeatureStatus)
	}
	if capturedTaskStatus != models.TaskStatus("completed") {
		t.Errorf("expected taskStatus 'completed', got %q", capturedTaskStatus)
	}
}

func TestEpicService_CascadeStatusToFeaturesAndTasks_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		cascadeStatusToFeaturesAndTasksFn: func(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error {
			return fmt.Errorf("db cascade failed")
		},
	}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.CascadeStatusToFeaturesAndTasks(
		context.Background(), 1,
		models.FeatureStatus("completed"),
		models.TaskStatus("completed"),
	)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ============================================================================
// MockEpicWritableDocumentRepository
// ============================================================================

// mockEpicDocRepo implements EntityDocumentRepository for epic-service doc tests.
type mockEpicDocRepo struct {
	createOrGetFn func(ctx context.Context, title, filePath string) (*models.Document, error)
	getByTitleFn  func(ctx context.Context, title string) (*models.Document, error)
}

func (m *mockEpicDocRepo) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	if m.createOrGetFn != nil {
		return m.createOrGetFn(ctx, title, filePath)
	}
	return nil, fmt.Errorf("CreateOrGet not implemented")
}

func (m *mockEpicDocRepo) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
	if m.getByTitleFn != nil {
		return m.getByTitleFn(ctx, title)
	}
	return nil, fmt.Errorf("GetByTitle not implemented")
}

// mockEpicLinkRepo implements EntityDocumentLinkRepository for epic-service doc tests.
type mockEpicLinkRepo struct {
	linkFn          func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error
	unlinkFn        func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error
	listForEntityFn func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}

func (m *mockEpicLinkRepo) Link(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
	if m.linkFn != nil {
		return m.linkFn(ctx, entityType, entityID, documentID, linkType)
	}
	return fmt.Errorf("Link not implemented")
}

func (m *mockEpicLinkRepo) Unlink(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
	if m.unlinkFn != nil {
		return m.unlinkFn(ctx, entityType, entityID, documentID)
	}
	return fmt.Errorf("Unlink not implemented")
}

func (m *mockEpicLinkRepo) ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
	if m.listForEntityFn != nil {
		return m.listForEntityFn(ctx, entityType, entityID)
	}
	return nil, fmt.Errorf("ListForEntity not implemented")
}

// ============================================================================
// EpicService.LinkDocument Tests
// ============================================================================

func TestEpicService_LinkDocument_Happy_Path(t *testing.T) {
	var capturedEpicID, capturedDocID int64

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			if title != "Design Doc" {
				t.Errorf("expected title 'Design Doc', got %q", title)
			}
			if filePath != "docs/design.md" {
				t.Errorf("expected filePath 'docs/design.md', got %q", filePath)
			}
			return &models.Document{ID: 42, Title: title, FilePath: filePath}, nil
		},
	}
	linkRepo := &mockEpicLinkRepo{
		linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
			capturedEpicID = entityID
			capturedDocID = documentID
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.LinkDocument(context.Background(), "E07", "Design Doc", "docs/design.md")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedEpicID != 7 {
		t.Errorf("expected epic ID 7, got %d", capturedEpicID)
	}
	if capturedDocID != 42 {
		t.Errorf("expected doc ID 42, got %d", capturedDocID)
	}
}

func TestEpicService_LinkDocument_NoWritableDocRepo(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	// writableDocRepo not set

	err := svc.LinkDocument(context.Background(), "E07", "Design Doc", "docs/design.md")

	if err == nil {
		t.Fatal("expected error when writable doc repo not configured, got nil")
	}
	if !strings.Contains(err.Error(), "writable document repository not configured") {
		t.Errorf("expected 'writable document repository not configured' in error, got: %v", err)
	}
}

func TestEpicService_LinkDocument_EpicNotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("epic not found")
		},
	}

	docRepo := &mockEpicDocRepo{}
	linkRepo := &mockEpicLinkRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.LinkDocument(context.Background(), "E99", "Design Doc", "docs/design.md")

	if err == nil {
		t.Fatal("expected error when epic not found, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestEpicService_LinkDocument_CreateOrGetError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	linkRepo := &mockEpicLinkRepo{}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.LinkDocument(context.Background(), "E07", "Design Doc", "docs/design.md")

	if err == nil {
		t.Fatal("expected error from CreateOrGet, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create or get document") {
		t.Errorf("expected 'failed to create or get document' in error, got: %v", err)
	}
}

func TestEpicService_LinkDocument_LinkError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title}, nil
		},
	}
	linkRepo := &mockEpicLinkRepo{
		linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
			return fmt.Errorf("link failed")
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.LinkDocument(context.Background(), "E07", "Design Doc", "docs/design.md")

	if err == nil {
		t.Fatal("expected error from LinkToEpic, got nil")
	}
	if !strings.Contains(err.Error(), "failed to link document to epic") {
		t.Errorf("expected 'failed to link document to epic' in error, got: %v", err)
	}
}

// ============================================================================
// EpicService.UnlinkDocument Tests
// ============================================================================

func TestEpicService_UnlinkDocument_Happy_Path(t *testing.T) {
	var capturedEpicID, capturedDocID int64

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			if title != "Design Doc" {
				t.Errorf("expected title 'Design Doc', got %q", title)
			}
			return &models.Document{ID: 42, Title: title}, nil
		},
	}
	linkRepo := &mockEpicLinkRepo{
		unlinkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
			capturedEpicID = entityID
			capturedDocID = documentID
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.UnlinkDocument(context.Background(), "E07", "Design Doc")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedEpicID != 7 {
		t.Errorf("expected epic ID 7, got %d", capturedEpicID)
	}
	if capturedDocID != 42 {
		t.Errorf("expected doc ID 42, got %d", capturedDocID)
	}
}

func TestEpicService_UnlinkDocument_NoWritableDocRepo(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	err := svc.UnlinkDocument(context.Background(), "E07", "Design Doc")

	if err == nil {
		t.Fatal("expected error when writable doc repo not configured, got nil")
	}
	if !strings.Contains(err.Error(), "writable document repository not configured") {
		t.Errorf("expected 'writable document repository not configured' in error, got: %v", err)
	}
}

func TestEpicService_UnlinkDocument_EpicNotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("epic not found")
		},
	}

	docRepo := &mockEpicDocRepo{}
	linkRepo := &mockEpicLinkRepo{}
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.UnlinkDocument(context.Background(), "E99", "Design Doc")

	if err == nil {
		t.Fatal("expected error when epic not found, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestEpicService_UnlinkDocument_DocumentNotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return nil, fmt.Errorf("document not found")
		},
	}
	linkRepo := &mockEpicLinkRepo{}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.UnlinkDocument(context.Background(), "E07", "Missing Doc")

	// EntityDocumentService treats document-not-found as idempotent success
	if err != nil {
		t.Fatalf("expected nil error (idempotent for missing document), got: %v", err)
	}
}

func TestEpicService_UnlinkDocument_UnlinkError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}

	docRepo := &mockEpicDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title}, nil
		},
	}
	linkRepo := &mockEpicLinkRepo{
		unlinkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
			return fmt.Errorf("unlink failed")
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetWritableDocRepo(docRepo, linkRepo)

	err := svc.UnlinkDocument(context.Background(), "E07", "Design Doc")

	if err == nil {
		t.Fatal("expected error from UnlinkFromEpic, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unlink document from epic") {
		t.Errorf("expected 'failed to unlink document from epic' in error, got: %v", err)
	}
}

// ============================================================================
// TC-SVC-A through TC-SVC-E: Size field propagation (E07-F42-005)
// ============================================================================

func TestEpicService_CreateEpic_PropagatesSize(t *testing.T) {
	// TC-SVC-A: CreateEpic propagates Size to repository.
	var capturedEpic *models.Epic

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found") // custom key collision check
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			epic.ID = 1
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	size := 5
	epic, err := svc.CreateEpic(context.Background(), CreateEpicInput{
		Title: "Epic with size",
		Size:  &size,
	})

	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if capturedEpic == nil {
		t.Fatal("repo Create was not called")
	}
	if capturedEpic.Size == nil {
		t.Fatal("expected capturedEpic.Size to be non-nil")
	}
	if *capturedEpic.Size != 5 {
		t.Errorf("expected capturedEpic.Size=5, got %d", *capturedEpic.Size)
	}
}

func TestEpicService_CreateEpic_NilSizePropagated(t *testing.T) {
	// TC-SVC-E: CreateEpic passes Size=nil when not provided.
	var capturedEpic *models.Epic

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			epic.ID = 1
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	epic, err := svc.CreateEpic(context.Background(), CreateEpicInput{
		Title: "Epic without size",
	})

	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if capturedEpic == nil {
		t.Fatal("repo Create was not called")
	}
	if capturedEpic.Size != nil {
		t.Errorf("expected capturedEpic.Size=nil, got %d", *capturedEpic.Size)
	}
}

func TestEpicService_UpdateEpic_SetsSize(t *testing.T) {
	// TC-SVC-C: UpdateEpic with Size=ptr(8) updates the field.
	var capturedEpic *models.Epic

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status: "draft", Priority: "medium"}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	size := 8
	epic, err := svc.UpdateEpic(context.Background(), "E07", EpicUpdates{Size: &size})
	if err != nil {
		t.Fatalf("UpdateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if capturedEpic == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedEpic.Size == nil {
		t.Fatal("expected capturedEpic.Size to be non-nil")
	}
	if *capturedEpic.Size != 8 {
		t.Errorf("expected capturedEpic.Size=8, got %d", *capturedEpic.Size)
	}
}

func TestEpicService_UpdateEpic_ClearSize(t *testing.T) {
	// TC-SVC-B: UpdateEpic with ClearSize=true sets model.Size = nil.
	var capturedEpic *models.Epic

	existingSize := 5
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test", Size: &existingSize},
				Status: "draft", Priority: "medium"}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	epic, err := svc.UpdateEpic(context.Background(), "E07", EpicUpdates{ClearSize: true})
	if err != nil {
		t.Fatalf("UpdateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if capturedEpic == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedEpic.Size != nil {
		t.Errorf("expected capturedEpic.Size=nil (ClearSize=true), got %d", *capturedEpic.Size)
	}
}

func TestEpicService_UpdateEpic_NoSizeChange(t *testing.T) {
	// TC-SVC-D: UpdateEpic with neither Size nor ClearSize leaves size unchanged.
	var capturedEpic *models.Epic

	existingSize := 3
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test", Size: &existingSize},
				Status: "draft", Priority: "medium"}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			capturedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)

	epic, err := svc.UpdateEpic(context.Background(), "E07", EpicUpdates{})
	if err != nil {
		t.Fatalf("UpdateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if capturedEpic == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedEpic.Size == nil {
		t.Fatal("expected capturedEpic.Size to remain non-nil")
	}
	if *capturedEpic.Size != 3 {
		t.Errorf("expected capturedEpic.Size=3 (unchanged), got %d", *capturedEpic.Size)
	}
}
