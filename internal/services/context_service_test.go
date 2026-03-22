package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// mockContextEntityRepo provides a configurable mock EntityRepository for ContextService tests.
type mockContextEntityRepo struct {
	getByKeyFunc          func(ctx context.Context, key string) (models.Entity, error)
	getByIDFunc           func(ctx context.Context, id int64) (models.Entity, error)
	getContextDataFunc    func(ctx context.Context, id int64) (*string, error)
	updateContextDataFunc func(ctx context.Context, id int64, data *string) error
}

func (m *mockContextEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented")
}

func (m *mockContextEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented")
}

func (m *mockContextEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockContextEntityRepo) Update(_ context.Context, _ models.Entity) error {
	return nil
}

func (m *mockContextEntityRepo) GetContextData(ctx context.Context, id int64) (*string, error) {
	if m.getContextDataFunc != nil {
		return m.getContextDataFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockContextEntityRepo) UpdateContextData(ctx context.Context, id int64, data *string) error {
	if m.updateContextDataFunc != nil {
		return m.updateContextDataFunc(ctx, id, data)
	}
	return nil
}

// newContextTestRegistry creates a basic registry with all 5 entity types for context tests.
func newContextTestRegistry() *EntityRegistry {
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 2, Key: key}}, nil
		},
	})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: key}}, nil
		},
	})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 4, Key: key}}, nil
		},
	})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 5, Key: key}}, nil
		},
	})
	return reg
}

func TestContextService_GetContext_Epic_NoContext(t *testing.T) {
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return nil, nil
		},
	})
	// Register remaining types to satisfy completeness (not used in this test)
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return &contextJSON, nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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

func TestContextService_GetContext_Task(t *testing.T) {
	contextJSON := `{"implementation_decisions":{"db":"sqlite"}}`
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return &contextJSON, nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, _ string) (models.Entity, error) {
			return nil, fmt.Errorf("not found")
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	_, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E999")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestContextService_SetContextField_Epic(t *testing.T) {
	var savedData *string
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return nil, nil // No existing context
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			return nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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

	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return &existingJSON, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			return nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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
	svc := NewContextService(newContextTestRegistry())

	err := svc.SetContextField(context.Background(), models.EntityTypeEpic, "E16", "invalid_field", "value")
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestContextService_SetContextField_Feature(t *testing.T) {
	var savedData *string
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 2, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return nil, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			return nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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
	var savedData *string
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return nil, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			return nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	err := svc.SetContextField(context.Background(), models.EntityTypeTask, "E16-F01-001", "implementation_decisions", `{"framework":"react"}`)
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	if savedData == nil {
		t.Fatal("expected context data to be saved")
	}
}

func TestContextService_ClearContext_Epic(t *testing.T) {
	clearCalled := false
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			clearCalled = true
			if data != nil {
				t.Error("expected nil context data for clear")
			}
			return nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	err := svc.ClearContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if !clearCalled {
		t.Error("expected UpdateContextData to be called")
	}
}

func TestContextService_ClearContext_Task(t *testing.T) {
	var savedData *string
	cleared := false
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: key}}, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			cleared = true
			return nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	err := svc.ClearContext(context.Background(), models.EntityTypeTask, "E16-F01-001")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if !cleared {
		t.Error("expected UpdateContextData to be called")
	}
	if savedData != nil {
		t.Error("expected context data to be nil after clear")
	}
}

func TestContextService_GetContext_EmptyJSON(t *testing.T) {
	emptyJSON := "{}"
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return &emptyJSON, nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	cd, err := svc.GetContext(context.Background(), models.EntityTypeEpic, "E16")
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if cd != nil {
		t.Error("expected nil for empty JSON context")
	}
}

func TestContextService_UnsupportedEntityType(t *testing.T) {
	svc := NewContextService(newContextTestRegistry())

	_, err := svc.GetContext(context.Background(), models.EntityType("unknown"), "key")
	if err == nil {
		t.Fatal("expected error for unsupported entity type")
	}
}

func TestContextService_GetContext_ChangeCard_WithContext(t *testing.T) {
	contextJSON := `{"progress":{"current_step":"Reviewing"},"open_questions":["Impact?"]}`
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 10, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return &contextJSON, nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})

	svc := NewContextService(reg)

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
	var savedData *string
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 10, Key: key}}, nil
		},
		getContextDataFunc: func(_ context.Context, _ int64) (*string, error) {
			return nil, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			return nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	err := svc.SetContextField(context.Background(), models.EntityTypeChange, "C001", "current_step", "Approved")
	if err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	if savedData == nil {
		t.Fatal("expected context data to be set on change-card")
	}
}

func TestContextService_ClearContext_ChangeCard(t *testing.T) {
	var savedData *string
	cleared := false
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeChange, &mockContextEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 10, Key: key}}, nil
		},
		updateContextDataFunc: func(_ context.Context, _ int64, data *string) error {
			savedData = data
			cleared = true
			return nil
		},
	})
	reg.Register(models.EntityTypeEpic, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockContextEntityRepo{})
	reg.Register(models.EntityTypeBug, &mockContextEntityRepo{})

	svc := NewContextService(reg)

	err := svc.ClearContext(context.Background(), models.EntityTypeChange, "C001")
	if err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if !cleared {
		t.Error("expected UpdateContextData to be called")
	}
	if savedData != nil {
		t.Error("expected context data to be nil after clear")
	}
}

func TestContextService_NilRegistry_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil registry")
		}
		msg, ok := r.(string)
		if !ok || msg != "ContextService: EntityRegistry must not be nil" {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	NewContextService(nil)
}

func TestIsValidContextField(t *testing.T) {
	validFields := []string{
		"current_step", "completed_steps", "remaining_steps",
		"implementation_decisions", "open_questions", "blockers",
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
		{"invalid_field", "invalid_field", "value", true},
		{"bad_json_completed_steps", "completed_steps", "not json", true},
		{"bad_json_remaining_steps", "remaining_steps", "not json", true},
		{"bad_json_decisions", "implementation_decisions", "not json", true},
		{"bad_json_questions", "open_questions", "not json", true},
		{"bad_json_blockers", "blockers", "not json", true},
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
