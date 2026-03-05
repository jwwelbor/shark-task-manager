package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Mock implementations for ContextService dependencies

type mockContextEpicRepo struct {
	getByKeyFunc       func(ctx context.Context, key string) (*models.Epic, error)
	getContextDataFunc func(ctx context.Context, epicID int64) (*string, error)
	updateContextFunc  func(ctx context.Context, epicID int64, contextData *string) error
}

func (m *mockContextEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Epic{ID: 1, Key: key}, nil
}

func (m *mockContextEpicRepo) GetContextData(ctx context.Context, epicID int64) (*string, error) {
	if m.getContextDataFunc != nil {
		return m.getContextDataFunc(ctx, epicID)
	}
	return nil, nil
}

func (m *mockContextEpicRepo) UpdateContextData(ctx context.Context, epicID int64, contextData *string) error {
	if m.updateContextFunc != nil {
		return m.updateContextFunc(ctx, epicID, contextData)
	}
	return nil
}

type mockContextFeatureRepo struct {
	getByKeyFunc       func(ctx context.Context, key string) (*models.Feature, error)
	getContextDataFunc func(ctx context.Context, featureID int64) (*string, error)
	updateContextFunc  func(ctx context.Context, featureID int64, contextData *string) error
}

func (m *mockContextFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Feature{ID: 2, Key: key}, nil
}

func (m *mockContextFeatureRepo) GetContextData(ctx context.Context, featureID int64) (*string, error) {
	if m.getContextDataFunc != nil {
		return m.getContextDataFunc(ctx, featureID)
	}
	return nil, nil
}

func (m *mockContextFeatureRepo) UpdateContextData(ctx context.Context, featureID int64, contextData *string) error {
	if m.updateContextFunc != nil {
		return m.updateContextFunc(ctx, featureID, contextData)
	}
	return nil
}

type mockContextTaskRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
	updateFunc   func(ctx context.Context, task *models.Task) error
	storedTask   *models.Task // For tracking updates
}

func (m *mockContextTaskRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	if m.storedTask != nil {
		return m.storedTask, nil
	}
	return &models.Task{ID: 3, Key: key}, nil
}

func (m *mockContextTaskRepo) Update(ctx context.Context, task *models.Task) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, task)
	}
	m.storedTask = task
	return nil
}

func strPtr(s string) *string {
	return &s
}

func TestContextService_GetContext_Epic_NoContext(t *testing.T) {
	epicRepo := &mockContextEpicRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return nil, nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	cd, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd != nil {
		t.Error("expected nil context data for entity with no context")
	}
}

func TestContextService_GetContext_Epic_WithContext(t *testing.T) {
	contextJSON := `{"progress":{"current_step":"Implementing API"},"open_questions":["What about auth?"]}`
	epicRepo := &mockContextEpicRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return &contextJSON, nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	cd, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd == nil {
		t.Fatal("expected non-nil context data")
	}
	if cd.Progress == nil || cd.Progress.CurrentStep == nil || *cd.Progress.CurrentStep != "Implementing API" {
		t.Error("expected current_step to be 'Implementing API'")
	}
	if len(cd.OpenQuestions) != 1 || cd.OpenQuestions[0] != "What about auth?" {
		t.Error("expected open_questions to contain 'What about auth?'")
	}
}

// TestContextService_GetContext_Feature removed - related_tasks field removed from ContextData
// per commit f3235a8. Related tasks should use task_relationships table, not context_data JSON.

func TestContextService_GetContext_Task(t *testing.T) {
	contextJSON := `{"implementation_decisions":{"db":"sqlite"}}`
	taskRepo := &mockContextTaskRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{ID: 3, Key: key, ContextData: &contextJSON}, nil
		},
	}
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, taskRepo)

	cd, err := svc.GetContext(context.Background(), models.EntityTypeTask, "E16-F01-001")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd == nil {
		t.Fatal("expected non-nil context data")
	}
	if cd.ImplementationDecisions["db"] != "sqlite" {
		t.Error("expected implementation_decisions.db to be 'sqlite'")
	}
}

func TestContextService_GetContext_InvalidKey(t *testing.T) {
	epicRepo := &mockContextEpicRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	_, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E999")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestContextService_SetContextField_Epic(t *testing.T) {
	var savedData *string
	epicRepo := &mockContextEpicRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return nil, nil // No existing context
		},
		updateContextFunc: func(ctx context.Context, epicID int64, contextData *string) error {
			savedData = contextData
			return nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	err := svc.SetContextField(context.Background(), models.EntityTypeEpic, "E16", "current_step", "Design phase")
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}
	if savedData == nil {
		t.Fatal("expected context data to be saved")
	}

	// Parse back and verify
	cd, err := models.FromJSON(*savedData)
	if err != nil {
		t.Fatalf("failed to parse saved context: %v", err)
	}
	if cd.Progress == nil || cd.Progress.CurrentStep == nil || *cd.Progress.CurrentStep != "Design phase" {
		t.Error("expected current_step to be 'Design phase'")
	}
}

func TestContextService_SetContextField_MergeSemantics(t *testing.T) {
	existingJSON := `{"progress":{"current_step":"Step 1"},"open_questions":["Q1"]}`
	var savedData *string

	epicRepo := &mockContextEpicRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return &existingJSON, nil
		},
		updateContextFunc: func(ctx context.Context, epicID int64, contextData *string) error {
			savedData = contextData
			return nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	// Update current_step - should preserve open_questions
	err := svc.SetContextField(context.Background(), models.EntityTypeEpic, "E16", "current_step", "Step 2")
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	cd, err := models.FromJSON(*savedData)
	if err != nil {
		t.Fatalf("failed to parse saved context: %v", err)
	}
	if cd.Progress == nil || cd.Progress.CurrentStep == nil || *cd.Progress.CurrentStep != "Step 2" {
		t.Error("expected current_step to be updated to 'Step 2'")
	}
	if len(cd.OpenQuestions) != 1 || cd.OpenQuestions[0] != "Q1" {
		t.Error("expected open_questions to be preserved")
	}
}

func TestContextService_SetContextField_InvalidField(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	err := svc.SetContextField(context.Background(), models.EntityTypeEpic, "E16", "invalid_field", "value")
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestContextService_SetContextField_Feature(t *testing.T) {
	var savedData *string
	featureRepo := &mockContextFeatureRepo{
		getContextDataFunc: func(ctx context.Context, featureID int64) (*string, error) {
			return nil, nil
		},
		updateContextFunc: func(ctx context.Context, featureID int64, contextData *string) error {
			savedData = contextData
			return nil
		},
	}
	svc := NewContextService(&mockContextEpicRepo{}, featureRepo, &mockContextTaskRepo{})

	err := svc.SetContextField(context.Background(), models.EntityTypeFeature, "E16-F01", "open_questions", `["How to handle auth?"]`)
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	cd, err := models.FromJSON(*savedData)
	if err != nil {
		t.Fatalf("failed to parse saved context: %v", err)
	}
	if len(cd.OpenQuestions) != 1 || cd.OpenQuestions[0] != "How to handle auth?" {
		t.Error("expected open_questions to contain 'How to handle auth?'")
	}
}

func TestContextService_SetContextField_Task(t *testing.T) {
	taskRepo := &mockContextTaskRepo{
		storedTask: &models.Task{ID: 3, Key: "E16-F01-001"},
	}
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, taskRepo)

	err := svc.SetContextField(context.Background(), models.EntityTypeTask, "E16-F01-001", "implementation_decisions", `{"framework":"react"}`)
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	if taskRepo.storedTask.ContextData == nil {
		t.Fatal("expected context data to be set on task")
	}
}

func TestContextService_ClearContext_Epic(t *testing.T) {
	clearCalled := false
	epicRepo := &mockContextEpicRepo{
		updateContextFunc: func(ctx context.Context, epicID int64, contextData *string) error {
			clearCalled = true
			if contextData != nil {
				t.Error("expected nil context data for clear")
			}
			return nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	err := svc.ClearContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if !clearCalled {
		t.Error("expected UpdateContextData to be called")
	}
}

func TestContextService_ClearContext_Task(t *testing.T) {
	taskRepo := &mockContextTaskRepo{
		storedTask: &models.Task{ID: 3, Key: "E16-F01-001", ContextData: strPtr(`{"progress":{"current_step":"test"}}`)},
	}
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, taskRepo)

	err := svc.ClearContext(context.Background(), models.EntityTypeTask, "E16-F01-001")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if taskRepo.storedTask.ContextData != nil {
		t.Error("expected context data to be nil after clear")
	}
}

func TestContextService_GetContext_EmptyJSON(t *testing.T) {
	emptyJSON := "{}"
	epicRepo := &mockContextEpicRepo{
		getContextDataFunc: func(ctx context.Context, epicID int64) (*string, error) {
			return &emptyJSON, nil
		},
	}
	svc := NewContextService(epicRepo, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	cd, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd != nil {
		t.Error("expected nil for empty JSON context")
	}
}

func TestContextService_UnsupportedEntityType(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})

	_, err := svc.GetContext(context.Background(), models.EntityType("unknown"), "key")
	if err == nil {
		t.Fatal("expected error for unsupported entity type")
	}
}

// ---- Change-card context tests ----

type mockContextChangeCardRepo struct {
	getByKeyFunc          func(ctx context.Context, key string) (*models.ChangeCard, error)
	updateContextDataFunc func(ctx context.Context, id int64, contextData *string) error
	storedCard            *models.ChangeCard
}

func (m *mockContextChangeCardRepo) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	if m.storedCard != nil {
		return m.storedCard, nil
	}
	return &models.ChangeCard{ID: 10, Key: key}, nil
}

func (m *mockContextChangeCardRepo) UpdateContextData(ctx context.Context, id int64, contextData *string) error {
	if m.updateContextDataFunc != nil {
		return m.updateContextDataFunc(ctx, id, contextData)
	}
	if m.storedCard != nil {
		m.storedCard.ContextData = contextData
	}
	return nil
}

func TestContextService_GetContext_ChangeCard_NoRepo(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	// No change-card repo wired in

	_, err := svc.GetContext(context.Background(), models.EntityTypeChange, "C001")
	if err == nil {
		t.Fatal("expected error when change-card repo not configured")
	}
	if err.Error() != "change-card repository not configured for context operations" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestContextService_GetContext_ChangeCard_NotFound(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	ccRepo := &mockContextChangeCardRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	svc.SetChangeCardRepo(ccRepo)

	_, err := svc.GetContext(context.Background(), models.EntityTypeChange, "C999")
	if err == nil {
		t.Fatal("expected error for non-existent change-card")
	}
}

func TestContextService_GetContext_ChangeCard_NoContext(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	ccRepo := &mockContextChangeCardRepo{
		storedCard: &models.ChangeCard{ID: 10, Key: "C001", ContextData: nil},
	}
	svc.SetChangeCardRepo(ccRepo)

	cd, err := svc.GetContext(context.Background(), models.EntityTypeChange, "C001")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd != nil {
		t.Error("expected nil context data for change-card with no context")
	}
}

func TestContextService_GetContext_ChangeCard_WithContext(t *testing.T) {
	contextJSON := `{"progress":{"current_step":"Reviewing"},"open_questions":["Impact?"]}`
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	ccRepo := &mockContextChangeCardRepo{
		storedCard: &models.ChangeCard{ID: 10, Key: "C001", ContextData: &contextJSON},
	}
	svc.SetChangeCardRepo(ccRepo)

	cd, err := svc.GetContext(context.Background(), models.EntityTypeChange, "C001")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd == nil {
		t.Fatal("expected non-nil context data")
	}
	if cd.Progress == nil || cd.Progress.CurrentStep == nil || *cd.Progress.CurrentStep != "Reviewing" {
		t.Error("expected current_step to be 'Reviewing'")
	}
}

func TestContextService_SetContextField_ChangeCard(t *testing.T) {
	card := &models.ChangeCard{ID: 10, Key: "C001"}
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	ccRepo := &mockContextChangeCardRepo{storedCard: card}
	svc.SetChangeCardRepo(ccRepo)

	err := svc.SetContextField(context.Background(), models.EntityTypeChange, "C001", "current_step", "Approved")
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	if card.ContextData == nil {
		t.Fatal("expected context data to be set on change-card")
	}
}

func TestContextService_ClearContext_ChangeCard(t *testing.T) {
	existingJSON := `{"progress":{"current_step":"step"}}`
	card := &models.ChangeCard{ID: 10, Key: "C001", ContextData: &existingJSON}
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	ccRepo := &mockContextChangeCardRepo{storedCard: card}
	svc.SetChangeCardRepo(ccRepo)

	err := svc.ClearContext(context.Background(), models.EntityTypeChange, "C001")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if card.ContextData != nil {
		t.Error("expected context data to be nil after clear")
	}
}

func TestContextService_SetContextField_ChangeCard_NoRepo(t *testing.T) {
	svc := NewContextService(&mockContextEpicRepo{}, &mockContextFeatureRepo{}, &mockContextTaskRepo{})
	// No change-card repo

	err := svc.SetContextField(context.Background(), models.EntityTypeChange, "C001", "current_step", "Step")
	if err == nil {
		t.Fatal("expected error when change-card repo not configured")
	}
}

func TestIsValidContextField(t *testing.T) {
	validFields := []string{
		"current_step", "completed_steps", "remaining_steps",
		"implementation_decisions", "open_questions", "blockers",
		"acceptance_criteria_status",
	}

	for _, f := range validFields {
		if !isValidContextField(f) {
			t.Errorf("expected %s to be valid", f)
		}
	}

	invalidFields := []string{"invalid", "status", "title", "key", "related_tasks"}
	for _, f := range invalidFields {
		if isValidContextField(f) {
			t.Errorf("expected %s to be invalid", f)
		}
	}
}

func TestUpdateContextField_AllFields(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"current_step", "current_step", "Design phase", false},
		{"completed_steps", "completed_steps", `["Step 1","Step 2"]`, false},
		{"remaining_steps", "remaining_steps", `["Step 3"]`, false},
		{"implementation_decisions", "implementation_decisions", `{"db":"sqlite"}`, false},
		{"open_questions", "open_questions", `["Q1?"]`, false},
		{"blockers", "blockers", `[]`, false},
		{"acceptance_criteria_status", "acceptance_criteria_status", `[]`, false},
		{"invalid_field", "invalid_field", "value", true},
		{"bad_json_completed_steps", "completed_steps", "not json", true},
		{"bad_json_remaining_steps", "remaining_steps", "not json", true},
		{"bad_json_decisions", "implementation_decisions", "not json", true},
		{"bad_json_questions", "open_questions", "not json", true},
		{"bad_json_blockers", "blockers", "not json", true},
		{"bad_json_criteria", "acceptance_criteria_status", "not json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &models.ContextData{}
			err := updateContextField(cd, tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateContextField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
