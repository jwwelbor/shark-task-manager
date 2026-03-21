package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// --- Mock typed repositories ---

type mockEpicAdapterRepo struct {
	getByKeyFunc          func(ctx context.Context, key string) (*models.Epic, error)
	getByIDFunc           func(ctx context.Context, id int64) (*models.Epic, error)
	updateFunc            func(ctx context.Context, epic *models.Epic) error
	updateStatusFunc      func(ctx context.Context, epicID int64, status models.EpicStatus) error
	getContextDataFunc    func(ctx context.Context, epicID int64) (*string, error)
	updateContextDataFunc func(ctx context.Context, epicID int64, data *string) error
}

func (m *mockEpicAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockEpicAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockEpicAdapterRepo) Update(ctx context.Context, epic *models.Epic) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, epic)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockEpicAdapterRepo) UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, epicID, status)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockEpicAdapterRepo) GetContextData(ctx context.Context, epicID int64) (*string, error) {
	if m.getContextDataFunc != nil {
		return m.getContextDataFunc(ctx, epicID)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockEpicAdapterRepo) UpdateContextData(ctx context.Context, epicID int64, data *string) error {
	if m.updateContextDataFunc != nil {
		return m.updateContextDataFunc(ctx, epicID, data)
	}
	return fmt.Errorf("not implemented")
}

type mockFeatureAdapterRepo struct {
	getByKeyFunc          func(ctx context.Context, key string) (*models.Feature, error)
	getByIDFunc           func(ctx context.Context, id int64) (*models.Feature, error)
	updateFunc            func(ctx context.Context, feature *models.Feature) error
	getContextDataFunc    func(ctx context.Context, featureID int64) (*string, error)
	updateContextDataFunc func(ctx context.Context, featureID int64, data *string) error
}

func (m *mockFeatureAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockFeatureAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockFeatureAdapterRepo) Update(ctx context.Context, feature *models.Feature) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, feature)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockFeatureAdapterRepo) GetContextData(ctx context.Context, featureID int64) (*string, error) {
	if m.getContextDataFunc != nil {
		return m.getContextDataFunc(ctx, featureID)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockFeatureAdapterRepo) UpdateContextData(ctx context.Context, featureID int64, data *string) error {
	if m.updateContextDataFunc != nil {
		return m.updateContextDataFunc(ctx, featureID, data)
	}
	return fmt.Errorf("not implemented")
}

type mockTaskAdapterRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
	getByIDFunc  func(ctx context.Context, id int64) (*models.Task, error)
	updateFunc   func(ctx context.Context, task *models.Task) error
}

func (m *mockTaskAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockTaskAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockTaskAdapterRepo) Update(ctx context.Context, task *models.Task) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, task)
	}
	return fmt.Errorf("not implemented")
}

type mockBugAdapterRepo struct {
	getByKeyFunc     func(ctx context.Context, key string) (*models.Bug, error)
	getByIDFunc      func(ctx context.Context, id int64) (*models.Bug, error)
	updateFunc       func(ctx context.Context, bug *models.Bug) error
	updateStatusFunc func(ctx context.Context, id int64, status models.BugStatus) error
}

func (m *mockBugAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockBugAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Bug, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockBugAdapterRepo) Update(ctx context.Context, bug *models.Bug) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, bug)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockBugAdapterRepo) UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return fmt.Errorf("not implemented")
}

type mockChangeCardAdapterRepo struct {
	getByKeyFunc          func(ctx context.Context, key string) (*models.ChangeCard, error)
	getByIDFunc           func(ctx context.Context, id int64) (*models.ChangeCard, error)
	updateFunc            func(ctx context.Context, card *models.ChangeCard) error
	updateStatusFunc      func(ctx context.Context, id int64, status models.ChangeCardStatus) error
	updateContextDataFunc func(ctx context.Context, id int64, data *string) error
}

func (m *mockChangeCardAdapterRepo) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockChangeCardAdapterRepo) GetByID(ctx context.Context, id int64) (*models.ChangeCard, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockChangeCardAdapterRepo) Update(ctx context.Context, card *models.ChangeCard) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, card)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockChangeCardAdapterRepo) UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return fmt.Errorf("not implemented")
}
func (m *mockChangeCardAdapterRepo) UpdateContextData(ctx context.Context, id int64, data *string) error {
	if m.updateContextDataFunc != nil {
		return m.updateContextDataFunc(ctx, id, data)
	}
	return fmt.Errorf("not implemented")
}

// --- Epic Adapter Tests (thorough) ---

func TestEpicAdapter_GetByKey(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Test Epic"}, Status: "active"}
	mock := &mockEpicAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			if key != "E01" {
				t.Errorf("expected key E01, got %s", key)
			}
			return epic, nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	entity, err := adapter.GetByKey(context.Background(), "E01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetKey() != "E01" {
		t.Errorf("expected key E01, got %s", entity.GetKey())
	}
	if entity.GetEntityType() != models.EntityTypeEpic {
		t.Errorf("expected entity type epic, got %s", entity.GetEntityType())
	}
}

func TestEpicAdapter_GetByKey_Error(t *testing.T) {
	mock := &mockEpicAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	_, err := adapter.GetByKey(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEpicAdapter_GetByID(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 5, Key: "E05", Title: "Epic Five"}}
	mock := &mockEpicAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
			if id != 5 {
				t.Errorf("expected id 5, got %d", id)
			}
			return epic, nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	entity, err := adapter.GetByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetID() != 5 {
		t.Errorf("expected ID 5, got %d", entity.GetID())
	}
}

func TestEpicAdapter_UpdateStatus(t *testing.T) {
	var capturedStatus models.EpicStatus
	mock := &mockEpicAdapterRepo{
		updateStatusFunc: func(ctx context.Context, epicID int64, status models.EpicStatus) error {
			if epicID != 1 {
				t.Errorf("expected epicID 1, got %d", epicID)
			}
			capturedStatus = status
			return nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 1, "completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != "completed" {
		t.Errorf("expected status completed, got %s", capturedStatus)
	}
}

func TestEpicAdapter_Update_CorrectType(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Updated"}}
	var capturedEpic *models.Epic
	mock := &mockEpicAdapterRepo{
		updateFunc: func(ctx context.Context, e *models.Epic) error {
			capturedEpic = e
			return nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), epic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEpic != epic {
		t.Error("expected same epic pointer to be passed through")
	}
}

func TestEpicAdapter_Update_WrongType(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001"}}
	mock := &mockEpicAdapterRepo{}
	adapter := NewEpicRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for wrong type, got nil")
	}
	if expected := "EpicRepositoryAdapter.Update: expected *models.Epic, got *models.Task"; err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestEpicAdapter_GetContextData(t *testing.T) {
	data := `{"progress":{"current_step":"testing"}}`
	mock := &mockEpicAdapterRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			if epicID != 3 {
				t.Errorf("expected epicID 3, got %d", epicID)
			}
			return &data, nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || *result != data {
		t.Errorf("expected context data %q, got %v", data, result)
	}
}

func TestEpicAdapter_GetContextData_Nil(t *testing.T) {
	mock := &mockEpicAdapterRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return nil, nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil context data, got %v", result)
	}
}

func TestEpicAdapter_UpdateContextData(t *testing.T) {
	data := `{"key":"value"}`
	var capturedData *string
	mock := &mockEpicAdapterRepo{
		updateContextDataFunc: func(ctx context.Context, epicID int64, d *string) error {
			if epicID != 2 {
				t.Errorf("expected epicID 2, got %d", epicID)
			}
			capturedData = d
			return nil
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	err := adapter.UpdateContextData(context.Background(), 2, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedData == nil || *capturedData != data {
		t.Errorf("expected data %q, got %v", data, capturedData)
	}
}

// --- Feature Adapter Tests ---

func TestFeatureAdapter_GetByKey(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 2, Key: "E01-F01", Title: "Test Feature"}, Status: "active"}
	mock := &mockFeatureAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return feature, nil
		},
	}
	adapter := NewFeatureRepositoryAdapter(mock)
	entity, err := adapter.GetByKey(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetKey() != "E01-F01" {
		t.Errorf("expected key E01-F01, got %s", entity.GetKey())
	}
	if entity.GetEntityType() != models.EntityTypeFeature {
		t.Errorf("expected entity type feature, got %s", entity.GetEntityType())
	}
}

func TestFeatureAdapter_UpdateStatus_GetSetUpdate(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01"}, Status: "draft"}
	var updatedFeature *models.Feature
	mock := &mockFeatureAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
			if id != 10 {
				t.Errorf("expected id 10, got %d", id)
			}
			return feature, nil
		},
		updateFunc: func(ctx context.Context, f *models.Feature) error {
			updatedFeature = f
			return nil
		},
	}
	adapter := NewFeatureRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 10, "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected update to be called")
	}
	if updatedFeature.Status != "active" {
		t.Errorf("expected status active, got %s", updatedFeature.Status)
	}
}

func TestFeatureAdapter_UpdateStatus_GetError(t *testing.T) {
	mock := &mockFeatureAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
			return nil, fmt.Errorf("feature not found")
		},
	}
	adapter := NewFeatureRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 99, "active")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFeatureAdapter_Update_WrongType(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}
	mock := &mockFeatureAdapterRepo{}
	adapter := NewFeatureRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), epic)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestFeatureAdapter_GetContextData(t *testing.T) {
	data := `{"phase":"development"}`
	mock := &mockFeatureAdapterRepo{
		getContextDataFunc: func(ctx context.Context, featureID int64) (*string, error) {
			return &data, nil
		},
	}
	adapter := NewFeatureRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || *result != data {
		t.Errorf("expected %q, got %v", data, result)
	}
}

func TestFeatureAdapter_UpdateContextData(t *testing.T) {
	data := `{"phase":"done"}`
	var capturedData *string
	mock := &mockFeatureAdapterRepo{
		updateContextDataFunc: func(ctx context.Context, featureID int64, d *string) error {
			capturedData = d
			return nil
		},
	}
	adapter := NewFeatureRepositoryAdapter(mock)
	err := adapter.UpdateContextData(context.Background(), 2, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedData == nil || *capturedData != data {
		t.Errorf("expected data %q, got %v", data, capturedData)
	}
}

// --- Task Adapter Tests ---

func TestTaskAdapter_GetByKey(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: "T-E01-F01-001", Title: "Test Task"}, Status: "todo"}
	mock := &mockTaskAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return task, nil
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	entity, err := adapter.GetByKey(context.Background(), "T-E01-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetKey() != "T-E01-F01-001" {
		t.Errorf("expected key T-E01-F01-001, got %s", entity.GetKey())
	}
	if entity.GetEntityType() != models.EntityTypeTask {
		t.Errorf("expected entity type task, got %s", entity.GetEntityType())
	}
}

func TestTaskAdapter_UpdateStatus_GetSetUpdate(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: "T-E01-F01-001"}, Status: "todo"}
	var updatedTask *models.Task
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(ctx context.Context, t *models.Task) error {
			updatedTask = t
			return nil
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 3, "in_progress")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedTask == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTask.Status != "in_progress" {
		t.Errorf("expected status in_progress, got %s", updatedTask.Status)
	}
}

func TestTaskAdapter_Update_WrongType(t *testing.T) {
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001"}}
	mock := &mockTaskAdapterRepo{}
	adapter := NewTaskRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), bug)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestTaskAdapter_GetContextData(t *testing.T) {
	data := `{"progress":{"current_step":"writing tests"}}`
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: "T-E01-F01-001", ContextData: &data}}
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return task, nil
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || *result != data {
		t.Errorf("expected %q, got %v", data, result)
	}
}

func TestTaskAdapter_GetContextData_Nil(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: "T-E01-F01-001", ContextData: nil}}
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return task, nil
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestTaskAdapter_UpdateContextData(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: "T-E01-F01-001", ContextData: nil}}
	data := `{"progress":{"current_step":"implementing"}}`
	var updatedTask *models.Task
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(ctx context.Context, t *models.Task) error {
			updatedTask = t
			return nil
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	err := adapter.UpdateContextData(context.Background(), 3, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedTask == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTask.ContextData == nil || *updatedTask.ContextData != data {
		t.Errorf("expected context data %q, got %v", data, updatedTask.ContextData)
	}
}

// --- Bug Adapter Tests ---

func TestBugAdapter_GetByKey(t *testing.T) {
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 4, Key: "B001", Title: "Test Bug"}, Status: "open"}
	mock := &mockBugAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Bug, error) {
			return bug, nil
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	entity, err := adapter.GetByKey(context.Background(), "B001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetKey() != "B001" {
		t.Errorf("expected key B001, got %s", entity.GetKey())
	}
	if entity.GetEntityType() != models.EntityTypeBug {
		t.Errorf("expected entity type bug, got %s", entity.GetEntityType())
	}
}

func TestBugAdapter_UpdateStatus(t *testing.T) {
	var capturedStatus models.BugStatus
	mock := &mockBugAdapterRepo{
		updateStatusFunc: func(ctx context.Context, id int64, status models.BugStatus) error {
			capturedStatus = status
			return nil
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 4, "closed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != "closed" {
		t.Errorf("expected status closed, got %s", capturedStatus)
	}
}

func TestBugAdapter_Update_WrongType(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}
	mock := &mockBugAdapterRepo{}
	adapter := NewBugRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), epic)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestBugAdapter_GetContextData(t *testing.T) {
	data := `{"triage_notes":"needs investigation"}`
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 4, Key: "B001", ContextData: &data}}
	mock := &mockBugAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Bug, error) {
			return bug, nil
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || *result != data {
		t.Errorf("expected %q, got %v", data, result)
	}
}

func TestBugAdapter_UpdateContextData(t *testing.T) {
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 4, Key: "B001", ContextData: nil}}
	data := `{"triage_notes":"fixed"}`
	var updatedBug *models.Bug
	mock := &mockBugAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Bug, error) {
			return bug, nil
		},
		updateFunc: func(ctx context.Context, b *models.Bug) error {
			updatedBug = b
			return nil
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	err := adapter.UpdateContextData(context.Background(), 4, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedBug == nil {
		t.Fatal("expected update to be called")
	}
	if updatedBug.ContextData == nil || *updatedBug.ContextData != data {
		t.Errorf("expected context data %q, got %v", data, updatedBug.ContextData)
	}
}

// --- ChangeCard Adapter Tests ---

func TestChangeCardAdapter_GetByKey(t *testing.T) {
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 5, Key: "CC-001", Title: "Test Change"}, Status: "draft"}
	mock := &mockChangeCardAdapterRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return card, nil
		},
	}
	adapter := NewChangeCardRepositoryAdapter(mock)
	entity, err := adapter.GetByKey(context.Background(), "CC-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.GetKey() != "CC-001" {
		t.Errorf("expected key CC-001, got %s", entity.GetKey())
	}
	if entity.GetEntityType() != models.EntityTypeChange {
		t.Errorf("expected entity type change, got %s", entity.GetEntityType())
	}
}

func TestChangeCardAdapter_UpdateStatus(t *testing.T) {
	var capturedStatus models.ChangeCardStatus
	mock := &mockChangeCardAdapterRepo{
		updateStatusFunc: func(ctx context.Context, id int64, status models.ChangeCardStatus) error {
			capturedStatus = status
			return nil
		},
	}
	adapter := NewChangeCardRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 5, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != "approved" {
		t.Errorf("expected status approved, got %s", capturedStatus)
	}
}

func TestChangeCardAdapter_Update_WrongType(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001"}}
	mock := &mockChangeCardAdapterRepo{}
	adapter := NewChangeCardRepositoryAdapter(mock)
	err := adapter.Update(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestChangeCardAdapter_GetContextData(t *testing.T) {
	data := `{"impact":"low"}`
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 5, Key: "CC-001", ContextData: &data}}
	mock := &mockChangeCardAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.ChangeCard, error) {
			return card, nil
		},
	}
	adapter := NewChangeCardRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || *result != data {
		t.Errorf("expected %q, got %v", data, result)
	}
}

func TestChangeCardAdapter_UpdateContextData(t *testing.T) {
	data := `{"impact":"high"}`
	var capturedData *string
	mock := &mockChangeCardAdapterRepo{
		updateContextDataFunc: func(ctx context.Context, id int64, d *string) error {
			capturedData = d
			return nil
		},
	}
	adapter := NewChangeCardRepositoryAdapter(mock)
	err := adapter.UpdateContextData(context.Background(), 5, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedData == nil || *capturedData != data {
		t.Errorf("expected data %q, got %v", data, capturedData)
	}
}

// --- Cross-adapter: EntityRepository interface test ---

func TestAllAdapters_SatisfyEntityRepository(t *testing.T) {
	// Verify all adapters can be assigned to EntityRepository variable.
	// This is a compile-time check; if any adapter does not implement
	// EntityRepository, this function will not compile.
	var _ EntityRepository = NewEpicRepositoryAdapter(&mockEpicAdapterRepo{})
	var _ EntityRepository = NewFeatureRepositoryAdapter(&mockFeatureAdapterRepo{})
	var _ EntityRepository = NewTaskRepositoryAdapter(&mockTaskAdapterRepo{})
	var _ EntityRepository = NewBugRepositoryAdapter(&mockBugAdapterRepo{})
	var _ EntityRepository = NewChangeCardRepositoryAdapter(&mockChangeCardAdapterRepo{})
}

// --- Error propagation tests ---

func TestEpicAdapter_UpdateStatus_Error(t *testing.T) {
	mock := &mockEpicAdapterRepo{
		updateStatusFunc: func(ctx context.Context, epicID int64, status models.EpicStatus) error {
			return fmt.Errorf("database error")
		},
	}
	adapter := NewEpicRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 1, "completed")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTaskAdapter_UpdateStatus_GetError(t *testing.T) {
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return nil, fmt.Errorf("task not found")
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	err := adapter.UpdateStatus(context.Background(), 99, "in_progress")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBugAdapter_GetContextData_Error(t *testing.T) {
	mock := &mockBugAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found")
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error")
	}
}

func TestChangeCardAdapter_GetContextData_Error(t *testing.T) {
	mock := &mockChangeCardAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("card not found")
		},
	}
	adapter := NewChangeCardRepositoryAdapter(mock)
	result, err := adapter.GetContextData(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error")
	}
}

func TestTaskAdapter_UpdateContextData_GetError(t *testing.T) {
	mock := &mockTaskAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
			return nil, fmt.Errorf("task not found")
		},
	}
	adapter := NewTaskRepositoryAdapter(mock)
	data := `{"key":"value"}`
	err := adapter.UpdateContextData(context.Background(), 99, &data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBugAdapter_UpdateContextData_GetError(t *testing.T) {
	mock := &mockBugAdapterRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found")
		},
	}
	adapter := NewBugRepositoryAdapter(mock)
	data := `{"key":"value"}`
	err := adapter.UpdateContextData(context.Background(), 99, &data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
