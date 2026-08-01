package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

type questionBlockerRegistry struct{ repo EntityRepository }

func (r questionBlockerRegistry) GetRepository(models.EntityType) (EntityRepository, error) {
	return r.repo, nil
}

type questionBlockerEntityRepo struct {
	entity models.Entity
	calls  int
}

func (r *questionBlockerEntityRepo) GetByKey(context.Context, string) (models.Entity, error) {
	r.calls++
	return r.entity, nil
}
func (*questionBlockerEntityRepo) GetByID(context.Context, int64) (models.Entity, error) {
	return nil, nil
}
func (*questionBlockerEntityRepo) UpdateStatus(context.Context, int64, string) error { return nil }
func (*questionBlockerEntityRepo) UpdateStatusIfCurrent(context.Context, int64, string, string) (bool, error) {
	return false, nil
}
func (*questionBlockerEntityRepo) Update(context.Context, models.Entity) error { return nil }
func (*questionBlockerEntityRepo) GetContextData(context.Context, int64) (*string, error) {
	return nil, nil
}
func (*questionBlockerEntityRepo) UpdateContextData(context.Context, int64, *string) error {
	return nil
}

type questionBlockerRelationships struct {
	incoming []*models.EntityRelationship
	calls    int
	filter   []models.EntityRelationshipType
}

func (r *questionBlockerRelationships) GetIncoming(_ context.Context, _ models.EntityType, _ int64, types []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	r.calls++
	r.filter = append([]models.EntityRelationshipType(nil), types...)
	return r.incoming, nil
}

type questionBlockerQuestions struct {
	byID  map[int64]*models.Question
	calls int
}

func (r *questionBlockerQuestions) GetQuestionByID(_ context.Context, id int64) (*models.Question, error) {
	r.calls++
	question, found := r.byID[id]
	if !found {
		return nil, fmt.Errorf("Question %d not found", id)
	}
	return question, nil
}

func questionBlockerQuestion(t *testing.T, key string, blocking bool, status models.QuestionStatus, owner, responder string) *models.Question {
	t.Helper()
	question := &models.Question{BaseEntity: models.BaseEntity{Key: key}, Status: status, Summary: "compact summary", Blocking: blocking}
	if owner == "" {
		return question
	}
	state := models.QuestionState{ResolutionOwner: owner, Responders: []models.QuestionResponder{{Identity: responder, Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatalf("encode question state: %v", err)
	}
	question.ContextData = encoded
	return question
}

func TestQuestionBlockerCheckTC303QualifiesOnlyDirectOpenBlockingQuestion(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	statuses := []models.QuestionStatus{"draft", "open", "answering", "ready_for_resolution", "resolved", "withdrawn", "superseded"}
	for _, status := range statuses {
		for _, blocking := range []bool{false, true} {
			name := fmt.Sprintf("%s-blocking-%t", status, blocking)
			t.Run(name, func(t *testing.T) {
				candidate := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "F001"}}
				entities := &questionBlockerEntityRepo{entity: candidate}
				relationships := &questionBlockerRelationships{incoming: []*models.EntityRelationship{{ID: 1, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: now}}}
				questions := &questionBlockerQuestions{byID: map[int64]*models.Question{1: questionBlockerQuestion(t, "Q001", blocking, status, "owner", "responder")}}
				blocker, err := NewQuestionBlocker(relationships, questionBlockerRegistry{repo: entities}, questions)
				if err != nil {
					t.Fatalf("NewQuestionBlocker() error = %v", err)
				}

				block, err := blocker.Check(context.Background(), models.EntityTypeFeature, "F001")
				if err != nil {
					t.Fatalf("TC-303 Check() error = %v", err)
				}
				wantMatch := blocking && (status == "open" || status == "answering")
				if (block != nil) != wantMatch {
					t.Fatalf("TC-303 Check() match = %v, want %v", block != nil, wantMatch)
				}
				if entities.calls != 1 || relationships.calls != 1 || questions.calls != 1 {
					t.Fatalf("TC-303 read counts = entity %d, relationships %d, questions %d; want exactly 1 each", entities.calls, relationships.calls, questions.calls)
				}
				if len(relationships.filter) != 1 || relationships.filter[0] != models.EntityRelQuestionBlocks {
					t.Fatalf("TC-303 incoming filter = %v, want question_blocks only", relationships.filter)
				}
			})
		}
	}
}

func TestQuestionBlockerCheckTC303ExcludesGenericAndIndirectRelationships(t *testing.T) {
	candidate := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "F001"}}
	questions := &questionBlockerQuestions{byID: map[int64]*models.Question{1: questionBlockerQuestion(t, "Q001", true, "open", "owner", "responder")}}
	for _, relationship := range []*models.EntityRelationship{
		{ID: 1, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelBlocks},
		{ID: 2, FromEntityType: models.EntityTypeTask, FromEntityID: 2, RelationshipType: models.EntityRelQuestionBlocks},
	} {
		relationships := &questionBlockerRelationships{incoming: []*models.EntityRelationship{relationship}}
		blocker, err := NewQuestionBlocker(relationships, questionBlockerRegistry{repo: &questionBlockerEntityRepo{entity: candidate}}, questions)
		if err != nil {
			t.Fatalf("NewQuestionBlocker() error = %v", err)
		}
		block, err := blocker.Check(context.Background(), models.EntityTypeFeature, "F001")
		if err != nil || block != nil {
			t.Fatalf("TC-303 Check() = (%v, %v), want no match", block, err)
		}
	}
}

func TestQuestionBlockerCheckTC304SelectsOldestEdgeAndReturnsOnlyCompactFields(t *testing.T) {
	equal := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	candidate := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "F001"}}
	relationships := &questionBlockerRelationships{incoming: []*models.EntityRelationship{
		{ID: 8, FromEntityType: models.EntityTypeQuestion, FromEntityID: 3, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: equal},
		{ID: 9, FromEntityType: models.EntityTypeQuestion, FromEntityID: 2, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: equal.Add(time.Second)},
		{ID: 7, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: equal},
	}}
	questions := &questionBlockerQuestions{byID: map[int64]*models.Question{
		1: questionBlockerQuestion(t, "Q001", true, "open", "owner-1", "responder-1"),
		2: questionBlockerQuestion(t, "Q002", true, "answering", "owner-2", "responder-2"),
		3: questionBlockerQuestion(t, "Q003", true, "open", "owner-3", "responder-3"),
	}}
	blocker, err := NewQuestionBlocker(relationships, questionBlockerRegistry{repo: &questionBlockerEntityRepo{entity: candidate}}, questions)
	if err != nil {
		t.Fatalf("NewQuestionBlocker() error = %v", err)
	}

	block, err := blocker.Check(context.Background(), models.EntityTypeFeature, "F001")
	if err != nil {
		t.Fatalf("TC-304 Check() error = %v", err)
	}
	if block == nil || block.QuestionKey != "Q001" || block.Summary != "compact summary" || block.ResolutionOwner != "owner-1" || block.CurrentResponder != "responder-1" {
		t.Fatalf("TC-304 Check() = %#v, want Q001 compact handoff", block)
	}
	encoded, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal handoff: %v", err)
	}
	if len(fields) != 4 {
		t.Fatalf("TC-304 compact handoff fields = %v, want exactly four", fields)
	}
}

func TestQuestionBlockerCheckTC304RejectsOpenBlockingQuestionWithoutValidState(t *testing.T) {
	candidate := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "F001"}}
	relationships := &questionBlockerRelationships{incoming: []*models.EntityRelationship{{ID: 1, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelQuestionBlocks}}}
	question := questionBlockerQuestion(t, "Q001", true, "open", "", "")
	blocker, err := NewQuestionBlocker(relationships, questionBlockerRegistry{repo: &questionBlockerEntityRepo{entity: candidate}}, &questionBlockerQuestions{byID: map[int64]*models.Question{1: question}})
	if err != nil {
		t.Fatalf("NewQuestionBlocker() error = %v", err)
	}
	if _, err := blocker.Check(context.Background(), models.EntityTypeFeature, "F001"); err == nil {
		t.Fatal("TC-304 Check() error = nil, want actionable missing-state error")
	}
}

func TestQuestionBlockerCheckRejectsNonCanonicalSourceIdentity(t *testing.T) {
	candidate := &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "F001"}}
	relationships := &questionBlockerRelationships{incoming: []*models.EntityRelationship{{ID: 1, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelQuestionBlocks}}}
	question := questionBlockerQuestion(t, "not-a-question", true, "open", "owner", "responder")
	blocker, err := NewQuestionBlocker(relationships, questionBlockerRegistry{repo: &questionBlockerEntityRepo{entity: candidate}}, &questionBlockerQuestions{byID: map[int64]*models.Question{1: question}})
	if err != nil {
		t.Fatalf("NewQuestionBlocker() error = %v", err)
	}
	if _, err := blocker.Check(context.Background(), models.EntityTypeFeature, "F001"); err == nil {
		t.Fatal("Check() error = nil, want canonical Question identity rejection")
	}
}

func TestQuestionBlockedErrorCarriesOnlyCompactHandoff(t *testing.T) {
	err := NewQuestionBlockedError(models.EntityTypeFeature, "F001", &QuestionBlock{QuestionKey: "Q001", Summary: "compact", ResolutionOwner: "owner", CurrentResponder: "responder"})
	if err.QuestionBlock == nil || err.QuestionBlock.QuestionKey != "Q001" {
		t.Fatalf("QuestionBlockedError handoff = %#v, want compact Q001 handoff", err.QuestionBlock)
	}
	if err.CandidateType != models.EntityTypeFeature || err.CandidateKey != "F001" {
		t.Fatalf("QuestionBlockedError candidate = %s/%s, want feature/F001", err.CandidateType, err.CandidateKey)
	}
}
