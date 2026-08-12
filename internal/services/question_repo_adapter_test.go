package services

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// fakeQuestionAdapterRepository deliberately has no GetContextData/
// UpdateContextData methods, mirroring the real *QuestionRepository shape.
// That absence is what routes NewQuestionRepositoryAdapter's context-data
// calls through entityAdapter's emulated GetByID+SetContextData+Update path
// instead of a dedicated repo delegation -- the path production actually
// uses via `shark context set Q### ...`.
type fakeQuestionAdapterRepository struct {
	question           *models.Question
	updateCalls        int
	updatedID          int64
	updatedContextData *string
}

func (r *fakeQuestionAdapterRepository) GetByKey(context.Context, string) (*models.Question, error) {
	return r.question, nil
}
func (r *fakeQuestionAdapterRepository) GetByID(context.Context, int64) (*models.Question, error) {
	return r.question, nil
}
func (r *fakeQuestionAdapterRepository) Update(_ context.Context, question *models.Question) error {
	r.updateCalls++
	r.updatedID = question.ID
	if question.ContextData == nil {
		r.updatedContextData = nil
		return nil
	}
	captured := *question.ContextData
	r.updatedContextData = &captured
	return nil
}
func (r *fakeQuestionAdapterRepository) UpdateStatus(context.Context, int64, models.QuestionStatus) error {
	return nil
}
func (r *fakeQuestionAdapterRepository) UpdateStatusIfCurrent(context.Context, int64, models.QuestionStatus, models.QuestionStatus) (bool, error) {
	return true, nil
}

func TestQuestionRepositoryAdapterProvidesTypedContextSeam(t *testing.T) {
	repo := &fakeQuestionAdapterRepository{question: &models.Question{
		BaseEntity: models.BaseEntity{ID: 1, Key: "Q001", Title: "Question"},
		Status:     models.QuestionStatusDraft,
		Summary:    "Summary",
		Requester:  "requester",
	}}
	adapter := NewQuestionRepositoryAdapter(repo)

	entity, err := adapter.GetByKey(context.Background(), "Q001")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if entity.GetEntityType() != models.EntityTypeQuestion {
		t.Errorf("entity type = %q, want %q", entity.GetEntityType(), models.EntityTypeQuestion)
	}
	data := `{"progress":{"current_step":"collect"}}`
	if err := adapter.UpdateContextData(context.Background(), entity.GetID(), &data); err != nil {
		t.Fatalf("UpdateContextData() error = %v", err)
	}
	if repo.updateCalls != 1 || repo.updatedID != 1 || repo.updatedContextData == nil || *repo.updatedContextData != data {
		t.Fatalf("Update() = calls:%d id:%d context:%v, want one update for ID 1 with %q", repo.updateCalls, repo.updatedID, repo.updatedContextData, data)
	}
	got, err := adapter.GetContextData(context.Background(), entity.GetID())
	if err != nil || got == nil || *got != data {
		t.Fatalf("GetContextData() = %v, %v, want %q", got, err, data)
	}
}
