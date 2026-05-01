package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// mockTaskRepo is a minimal mock for TaskRepository used in tracing tests.
type mockTaskRepo struct {
	getByKeyFn          func(ctx context.Context, key string) (*models.Task, error)
	getByIDFn           func(ctx context.Context, id int64) (*models.Task, error)
	createFn            func(ctx context.Context, task *models.Task) error
	updateFn            func(ctx context.Context, task *models.Task) error
	deleteFn            func(ctx context.Context, id int64) error
	listFn              func(ctx context.Context) ([]*models.Task, error)
	listByFeatureFn     func(ctx context.Context, featureID int64) ([]*models.Task, error)
	listByFeatureKeyFn  func(ctx context.Context, featureKey string) ([]*models.Task, error)
	listByEpicFn        func(ctx context.Context, epicKey string) ([]*models.Task, error)
	getTaskDependentsFn func(ctx context.Context, taskKey string) ([]*models.Task, error)
	listByKeyPrefixFn   func(ctx context.Context, prefix string) ([]*models.Task, error)
}

func (m *mockTaskRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockTaskRepo) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}
func (m *mockTaskRepo) Create(ctx context.Context, task *models.Task) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}
func (m *mockTaskRepo) Update(ctx context.Context, task *models.Task) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}
func (m *mockTaskRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockTaskRepo) List(ctx context.Context) ([]*models.Task, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}
func (m *mockTaskRepo) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureID)
	}
	return nil, nil
}
func (m *mockTaskRepo) ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error) {
	if m.listByFeatureKeyFn != nil {
		return m.listByFeatureKeyFn(ctx, featureKey)
	}
	return nil, nil
}
func (m *mockTaskRepo) ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	if m.listByEpicFn != nil {
		return m.listByEpicFn(ctx, epicKey)
	}
	return nil, nil
}
func (m *mockTaskRepo) GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error) {
	if m.getTaskDependentsFn != nil {
		return m.getTaskDependentsFn(ctx, taskKey)
	}
	return nil, nil
}
func (m *mockTaskRepo) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
	return nil
}
func (m *mockTaskRepo) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	return nil
}
func (m *mockTaskRepo) UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	return nil, nil
}
func (m *mockTaskRepo) StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
	return nil, nil
}
func (m *mockTaskRepo) StatusUpdateRawWithTx(_ context.Context, _ *sql.Tx, _ models.StatusUpdateParams) ([]string, error) {
	return nil, nil
}
func (m *mockTaskRepo) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil
}
func (m *mockTaskRepo) FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error) {
	if m.listByKeyPrefixFn != nil {
		return m.listByKeyPrefixFn(ctx, prefix)
	}
	return nil, nil
}
func (m *mockTaskRepo) GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error) {
	return nil, nil
}
func (m *mockTaskRepo) GetRejectionCounts(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
	// Default for tracing tests: no rejections.
	return map[int64]int{}, map[int64]*time.Time{}, nil
}

// newTestTracer creates an in-memory tracer for testing span output.
func newTestTracer(name string) (trace.Tracer, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	return tp.Tracer(name), exporter
}

// findSpanByName returns the first span with the given name or nil.
func findSpanByName(spans tracetest.SpanStubs, name string) *tracetest.SpanStub {
	for i := range spans {
		if spans[i].Name == name {
			return &spans[i]
		}
	}
	return nil
}

// spanHasAttribute checks if a span has an attribute with the given key and string value.
func spanHasAttribute(span *tracetest.SpanStub, key, value string) bool {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return true
		}
	}
	return false
}

func TestTaskService_GetTask_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	mockRepo := &mockTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{Key: key, Title: "Test Task"}}, nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	task, err := svc.GetTask(context.Background(), "E07-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Key != "E07-F01-001" {
		t.Errorf("expected key E07-F01-001, got %s", task.Key)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.GetTask")
	if span == nil {
		t.Fatal("expected span TaskService.GetTask not found")
	}
	if !spanHasAttribute(span, "task.key", "E07-F01-001") {
		t.Error("expected span to have attribute task.key=E07-F01-001")
	}
}

func TestTaskService_GetTask_Tracing_Error(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	mockRepo := &mockTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	_, err := svc.GetTask(context.Background(), "E07-F01-999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.GetTask")
	if span == nil {
		t.Fatal("expected span TaskService.GetTask not found")
	}
	// codes.Error in Go OTel SDK is 1
	if span.Status.Code != 1 {
		t.Errorf("expected span status Error (1), got %d", span.Status.Code)
	}
}

func TestTaskService_GetTask_NilTracer(t *testing.T) {
	mockRepo := &mockTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{Key: key, Title: "Test"}}, nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	// Do NOT call SetTracer -- tracer is nil

	task, err := svc.GetTask(context.Background(), "E07-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
}

func TestTaskService_CreateTask_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	mockRepo := &mockTaskRepo{
		listByKeyPrefixFn: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	_, _, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "New Task",
		AgentType:  "developer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.CreateTask")
	if span == nil {
		t.Fatal("expected span TaskService.CreateTask not found")
	}
	if !spanHasAttribute(span, "task.epic_key", "E07") {
		t.Error("expected span to have attribute task.epic_key=E07")
	}
	if !spanHasAttribute(span, "task.title", "New Task") {
		t.Error("expected span to have attribute task.title=New Task")
	}
}

func TestTaskService_DeleteTask_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	mockRepo := &mockTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getTaskDependentsFn: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			return nil, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			return nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	err := svc.DeleteTask(context.Background(), "E07-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.DeleteTask")
	if span == nil {
		t.Fatal("expected span TaskService.DeleteTask not found")
	}
	if !spanHasAttribute(span, "task.key", "E07-F01-001") {
		t.Error("expected span to have attribute task.key=E07-F01-001")
	}
}

func TestTaskService_ListTasks_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	mockRepo := &mockTaskRepo{
		listFn: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001", Title: "Task 1"}, Status: "todo"},
			}, nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{ShowAll: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.ListTasks")
	if span == nil {
		t.Fatal("expected span TaskService.ListTasks not found")
	}
}

func TestTaskService_GetNextStatus_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	// GetNextStatus needs an entity repo adapter; it reads the entity by key.
	// We skip deep testing since it delegates to entitySvc.GetNextStatus which
	// requires a full EntityRepository setup. The test verifies span creation.
	mockRepo := &mockTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "todo",
			}, nil
		},
	}
	entitySvc := NewEntityService(newMockWorkflowService())
	svc := NewTaskService(mockRepo, entitySvc, nil)
	svc.SetTracer(tracer)

	// This will likely fail because entitySvc.GetNextStatus needs a full adapter,
	// but the span should still be created before the error.
	_, _ = svc.GetNextStatus(context.Background(), "E07-F01-001")

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "TaskService.GetNextStatus")
	if span == nil {
		t.Fatal("expected span TaskService.GetNextStatus not found")
	}
	if !spanHasAttribute(span, "task.key", "E07-F01-001") {
		t.Error("expected span to have attribute task.key=E07-F01-001")
	}
}
