package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	questionrepo "github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

type mockQuestionRepository struct {
	createFn                func(context.Context, *models.Question) error
	getByKeyFn              func(context.Context, string) (*models.Question, error)
	getByIDFn               func(context.Context, int64) (*models.Question, error)
	deleteFn                func(context.Context, int64) error
	listFn                  func(context.Context, questionrepo.QuestionListFilter) ([]*models.Question, error)
	listOpenCandidatesFn    func(context.Context, int, int) ([]*models.Question, error)
	updateFn                func(context.Context, *models.Question) error
	statusFn                func(context.Context, int64, models.QuestionStatus) error
	configureWorkflowFn     func(context.Context, int64, models.QuestionStatus, *string, *string, string) error
	recordResponseFn        func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string) error
	followUpWorkExistsFn    func(context.Context, string) (bool, error)
	noteExistsFn            func(context.Context, string) (bool, error)
	resolveFn               func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error
	withdrawFn              func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error
	updateStatusIfCurrentFn func(context.Context, int64, models.QuestionStatus, models.QuestionStatus) (bool, error)
}

func (r *mockQuestionRepository) UpdateStatusIfCurrent(ctx context.Context, id int64, expected, next models.QuestionStatus) (bool, error) {
	if r.updateStatusIfCurrentFn != nil {
		return r.updateStatusIfCurrentFn(ctx, id, expected, next)
	}
	return false, errors.New("unexpected UpdateStatusIfCurrent call")
}

type focusedRelationshipReader struct {
	edges  []*models.EntityRelationship
	called bool
}

func (r *focusedRelationshipReader) GetIncomingPage(_ context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType, limit, offset int) ([]*models.EntityRelationship, error) {
	r.called = true
	if entityType != models.EntityTypeFeature || entityID != 8 || limit != 50 || offset != 0 || len(relTypes) != 1 || relTypes[0] != models.EntityRelQuestionBlocks {
		return nil, fmt.Errorf("unexpected focused relationship page arguments")
	}
	return r.edges, nil
}

func (r *mockQuestionRepository) ListOpenCandidates(ctx context.Context, limit, offset int) ([]*models.Question, error) {
	if r.listOpenCandidatesFn == nil {
		return nil, errors.New("unexpected ListOpenCandidates call")
	}
	return r.listOpenCandidatesFn(ctx, limit, offset)
}

func (r *mockQuestionRepository) FollowUpWorkExists(ctx context.Context, key string) (bool, error) {
	if r.followUpWorkExistsFn != nil {
		return r.followUpWorkExistsFn(ctx, key)
	}
	return true, nil
}
func (r *mockQuestionRepository) NoteExists(ctx context.Context, id string) (bool, error) {
	if r.noteExistsFn != nil {
		return r.noteExistsFn(ctx, id)
	}
	return true, nil
}

func (r *mockQuestionRepository) Resolve(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, kind string) error {
	if r.resolveFn == nil {
		return errors.New("unexpected Resolve call")
	}
	return r.resolveFn(ctx, id, expectedStatus, status, expectedContextData, contextData, owner, kind)
}

func (r *mockQuestionRepository) Withdraw(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, reason string) error {
	if r.withdrawFn == nil {
		return errors.New("unexpected Withdraw call")
	}
	return r.withdrawFn(ctx, id, expectedStatus, status, expectedContextData, contextData, owner, reason)
}

func (r *mockQuestionRepository) RecordResponse(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, responder string) error {
	if r.recordResponseFn == nil {
		return errors.New("unexpected RecordResponse call")
	}
	return r.recordResponseFn(ctx, id, expectedStatus, status, expectedContextData, contextData, responder)
}

func (r *mockQuestionRepository) ConfigureWorkflow(ctx context.Context, id int64, expectedStatus models.QuestionStatus, expectedContextData, contextData *string, resolutionOwner string) error {
	if r.configureWorkflowFn == nil {
		return errors.New("unexpected ConfigureWorkflow call")
	}
	return r.configureWorkflowFn(ctx, id, expectedStatus, expectedContextData, contextData, resolutionOwner)
}

func (r *mockQuestionRepository) List(ctx context.Context, filter questionrepo.QuestionListFilter) ([]*models.Question, error) {
	if r.listFn == nil {
		return nil, errors.New("unexpected List call")
	}
	return r.listFn(ctx, filter)
}

func (r *mockQuestionRepository) Update(ctx context.Context, question *models.Question) error {
	if r.updateFn == nil {
		return errors.New("unexpected Update call")
	}
	return r.updateFn(ctx, question)
}

func (r *mockQuestionRepository) UpdateStatus(ctx context.Context, id int64, status models.QuestionStatus) error {
	if r.statusFn == nil {
		return errors.New("unexpected UpdateStatus call")
	}
	return r.statusFn(ctx, id, status)
}

func (r *mockQuestionRepository) Create(ctx context.Context, question *models.Question) error {
	if r.createFn == nil {
		return errors.New("unexpected Create call")
	}
	return r.createFn(ctx, question)
}

func (r *mockQuestionRepository) GetByKey(ctx context.Context, key string) (*models.Question, error) {
	if r.getByKeyFn == nil {
		return nil, nil
	}
	return r.getByKeyFn(ctx, key)
}

func (r *mockQuestionRepository) GetByID(ctx context.Context, id int64) (*models.Question, error) {
	if r.getByIDFn == nil {
		return nil, nil
	}
	return r.getByIDFn(ctx, id)
}

func (r *mockQuestionRepository) Delete(ctx context.Context, id int64) error {
	if r.deleteFn == nil {
		return nil
	}
	return r.deleteFn(ctx, id)
}

func TestQuestionServiceGetQuestionByID(t *testing.T) {
	want := &models.Question{BaseEntity: models.BaseEntity{ID: 7, Key: "Q007"}}
	svc, err := NewQuestionService(&mockQuestionRepository{getByIDFn: func(_ context.Context, id int64) (*models.Question, error) {
		if id != 7 {
			t.Fatalf("GetByID() id = %d, want 7", id)
		}
		return want, nil
	}})
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	got, err := svc.GetQuestionByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetQuestionByID() error = %v", err)
	}
	if got != want {
		t.Fatalf("GetQuestionByID() = %p, want %p", got, want)
	}
}

// TC-001: CreateQuestion sends the normalized direct-service creation shape to
// its typed backing repository, then reloads it so callers receive
// database-assigned identity and timestamps.
func TestQuestionServiceCreateQuestionNormalizesAndPersists(t *testing.T) {
	var created *models.Question
	createdAt := time.Date(2026, time.July, 30, 16, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Second)
	persisted := &models.Question{BaseEntity: models.BaseEntity{
		ID: 39, Key: "Q001", CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, Status: models.QuestionStatusDraft, Summary: "Confirm gate", Requester: "release-manager", Blocking: true}
	repo := &mockQuestionRepository{
		createFn: func(_ context.Context, question *models.Question) error {
			created = question
			question.ID = 39
			question.Key = "Q001"
			return nil
		},
		getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
			if key != "Q001" {
				t.Fatalf("reload key = %q, want Q001", key)
			}
			return persisted, nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	got, err := svc.CreateQuestion(context.Background(), CreateQuestionInput{
		Title:       "  Release gate  ",
		Summary:     "  Confirm gate  ",
		Requester:   "  release-manager  ",
		Description: "keep exact description",
		Blocking:    true,
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	if got != persisted || got.ID != 39 || got.Key != "Q001" {
		t.Fatalf("CreateQuestion() = %#v, want reloaded Q001 record", got)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Errorf("CreateQuestion() timestamps = (%v, %v), want (%v, %v)", got.CreatedAt, got.UpdatedAt, createdAt, updatedAt)
	}
	if created.Title != "Release gate" || created.Summary != "Confirm gate" || created.Requester != "release-manager" {
		t.Errorf("created normalized fields = %#v", created)
	}
	if created.Description == nil || *created.Description != "keep exact description" {
		t.Errorf("description = %#v, want exact optional description", created.Description)
	}
	if !created.Blocking || created.Status != models.QuestionStatusDraft {
		t.Errorf("created base record = %#v, want blocking draft", created)
	}
}

// TC-102: configuration validates the complete Question-owned state before
// the repository transaction is permitted to write state, note, or history.
func TestQuestionServiceConfigureWorkflow_TC102(t *testing.T) {
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001"}, Status: models.QuestionStatusOpen, Summary: "Confirm release", Requester: "alice"}
	var persistedState *string
	repo := &mockQuestionRepository{
		getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
			if key != "Q001" {
				t.Fatalf("GetByKey key = %q, want Q001", key)
			}
			return question, nil
		},
		configureWorkflowFn: func(_ context.Context, id int64, expectedStatus models.QuestionStatus, expectedContextData, contextData *string, owner string) error {
			if id != 39 || expectedStatus != models.QuestionStatusOpen || expectedContextData != nil || owner != "release-owner" {
				t.Fatalf("ConfigureWorkflow args = (%d,%q)", id, owner)
			}
			persistedState = contextData
			question.ContextData = contextData
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	got, err := svc.ConfigureWorkflow(context.Background(), ConfigureWorkflowInput{
		Key: "Q001", ResolutionOwner: "release-owner", Responders: []string{"alice", "bob", "carol"},
	})
	if err != nil {
		t.Fatalf("TC-102 ConfigureWorkflow() error = %v", err)
	}
	if got != question || persistedState == nil {
		t.Fatalf("TC-102 ConfigureWorkflow() = %#v, persisted state = %v", got, persistedState)
	}
	state, err := models.DecodeQuestionState(persistedState)
	if err != nil || state == nil || state.CurrentResponder() != "alice" {
		t.Fatalf("TC-102 persisted QuestionState = %#v, error = %v", state, err)
	}

	before := question.ContextData
	_, err = svc.ConfigureWorkflow(context.Background(), ConfigureWorkflowInput{Key: "Q001", ResolutionOwner: "release-owner", Responders: []string{"alice", "alice"}})
	if err == nil {
		t.Fatal("TC-102 duplicate configuration responder error = nil")
	}
	if question.ContextData != before {
		t.Fatal("TC-102 rejected configuration mutated context data")
	}
}

// TestQuestionServiceConfigureWorkflowRejectsDuplicateResponderIdentity_TC102
// exercises QuestionState.Validate's duplicate-identity rejection directly,
// on a freshly unconfigured Question. TestQuestionServiceConfigureWorkflow_TC102's
// second call reuses an already-configured Question, so it actually
// re-triggers the "already configured" guard (ContextData != nil) before
// state.Validate() is ever reached -- a regression that dropped or inverted
// the duplicate-identity check there would still pass.
func TestQuestionServiceConfigureWorkflowRejectsDuplicateResponderIdentity_TC102(t *testing.T) {
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001"}, Status: models.QuestionStatusOpen, Summary: "Confirm release", Requester: "alice"}
	configureWorkflowCalled := false
	repo := &mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
		configureWorkflowFn: func(context.Context, int64, models.QuestionStatus, *string, *string, string) error {
			configureWorkflowCalled = true
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConfigureWorkflow(context.Background(), ConfigureWorkflowInput{
		Key: "Q001", ResolutionOwner: "release-owner", Responders: []string{"alice", "alice"},
	})
	if err == nil {
		t.Fatal("ConfigureWorkflow() error = nil, want rejection of a duplicate responder identity")
	}
	if configureWorkflowCalled {
		t.Fatal("repository write called despite the duplicate responder identity")
	}
}

// TestQuestionServiceConfigureWorkflowRejectsDisallowedStatus_TC102 locks in
// ConfigureWorkflow's "must be draft or open" precondition: a Question that
// has already progressed past open (answering, ready_for_resolution, or any
// terminal status) must not be reconfigurable.
func TestQuestionServiceConfigureWorkflowRejectsDisallowedStatus_TC102(t *testing.T) {
	for _, status := range []models.QuestionStatus{
		models.QuestionStatusAnswering,
		models.QuestionStatusReadyForResolution,
		models.QuestionStatusResolved,
		models.QuestionStatusWithdrawn,
		models.QuestionStatusSuperseded,
		models.QuestionStatusArchived,
	} {
		t.Run(string(status), func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001"}, Status: status, Summary: "Confirm release", Requester: "alice"}
			configureWorkflowCalled := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				configureWorkflowFn: func(context.Context, int64, models.QuestionStatus, *string, *string, string) error {
					configureWorkflowCalled = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			_, err = svc.ConfigureWorkflow(context.Background(), ConfigureWorkflowInput{
				Key: "Q001", ResolutionOwner: "release-owner", Responders: []string{"alice"},
			})
			if err == nil {
				t.Fatalf("ConfigureWorkflow() error = nil, want rejection for status %q", status)
			}
			if configureWorkflowCalled {
				t.Fatalf("repository write called for disallowed status %q", status)
			}
		})
	}
}

// TC-105: the production service entrypoint accepts only the active claim
// holder and atomically delegates the completed, bounded response state.
func TestQuestionServiceRecordResponse_TC105(t *testing.T) {
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}, {Identity: "bob", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", Title: "Question", ContextData: encoded}, Status: models.QuestionStatus("answering"), Summary: "Summary", Requester: "owner"}
	var persisted *string
	repo := &mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
		recordResponseFn: func(_ context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, data *string, responder string) error {
			if id != 39 || expectedStatus != "answering" || expectedContextData == nil || status != "answering" || responder != "alice" {
				t.Fatalf("RecordResponse args = (%d,%q,%q)", id, status, responder)
			}
			persisted = data
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	svc.SetClaimReader(fakeQuestionClaimReader{claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-a"}})
	if _, err := svc.RecordResponse(context.Background(), RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/plan/spec.md"}); err != nil {
		t.Fatalf("TC-105 RecordResponse() error = %v", err)
	}
	if persisted == nil {
		t.Fatal("TC-105 no state was persisted")
	}
	got, err := models.DecodeQuestionState(persisted)
	if err != nil || got == nil || got.Responders[0].Status != models.QuestionResponderCompleted || got.CurrentResponder() != "bob" || len(got.Responses) != 1 {
		t.Fatalf("TC-105 persisted state = %#v, %v", got, err)
	}
}

// TestQuestionServiceRecordResponseRejectsAuthorizationGuardViolations_TC105
// locks in RecordResponse's five authorization/session-binding guards: no
// claim reader wired, an empty session or responder, a claim whose session
// or claimant doesn't match the caller, and a claim held by a responder who
// isn't the currently routed one. None of these were previously exercised
// by any test -- a regression in any one would ship undetected.
func TestQuestionServiceRecordResponseRejectsAuthorizationGuardViolations_TC105(t *testing.T) {
	newQuestion := func() (*models.Question, *string) {
		state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}, {Identity: "bob", Status: models.QuestionResponderPending}}}
		encoded, err := models.EncodeQuestionState(nil, state)
		if err != nil {
			t.Fatal(err)
		}
		return &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", Title: "Question", ContextData: encoded}, Status: models.QuestionStatus("answering"), Summary: "Summary", Requester: "owner"}, encoded
	}

	cases := []struct {
		name        string
		input       RecordQuestionResponseInput
		claim       *models.EntityClaim
		skipClaimer bool
	}{
		{
			name:        "no_claim_reader_wired",
			input:       RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md"},
			skipClaimer: true,
		},
		{
			name:  "empty_session_id",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-a"},
		},
		{
			name:  "empty_responder",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-a"},
		},
		{
			name:  "no_active_claim",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: nil,
		},
		{
			name:  "claim_session_mismatch",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-other"},
		},
		{
			name:  "claim_claimant_mismatch",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "someone-else", SessionID: "session-a"},
		},
		{
			name:  "claim_valid_but_not_current_responder",
			input: RecordQuestionResponseInput{Key: "Q001", SessionID: "session-b", Responder: "bob", Summary: "approved", EvidencePointer: "docs/spec.md"},
			claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "bob", SessionID: "session-b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			question, _ := newQuestion()
			recordResponseCalled := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				recordResponseFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string) error {
					recordResponseCalled = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.skipClaimer {
				svc.SetClaimReader(fakeQuestionClaimReader{claim: tc.claim})
			}
			if _, err := svc.RecordResponse(context.Background(), tc.input); err == nil {
				t.Fatal("RecordResponse() error = nil, want rejection")
			}
			if recordResponseCalled {
				t.Fatal("repository write called despite the authorization guard violation")
			}
		})
	}
}

// TC-105: An exact completed-response replay is a no-op even after the parent
// releases its single Question claim. Replay recognition must happen before
// the current-responder and active-claim checks that protect forward progress.
func TestQuestionServiceRecordResponseExactCompletedReplayIsIdempotent_TC105(t *testing.T) {
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}, {Identity: "bob", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", Title: "Question", ContextData: encoded}, Status: "open", Summary: "Summary", Requester: "owner"}
	records := 0
	repo := &mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
		recordResponseFn: func(_ context.Context, _ int64, _ models.QuestionStatus, status models.QuestionStatus, _ *string, data *string, _ string) error {
			records++
			question.ContextData, question.Status = data, status
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	claimReader := &fakeQuestionClaimReader{claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-a"}}
	svc.SetClaimReader(claimReader)
	input := RecordQuestionResponseInput{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/plan/spec.md"}
	if _, err := svc.RecordResponse(context.Background(), input); err != nil {
		t.Fatalf("TC-105 initial RecordResponse() error = %v", err)
	}
	claimReader.claim = nil // parent release: replay must not need a new claim.
	if _, err := svc.RecordResponse(context.Background(), input); err != nil {
		t.Fatalf("TC-105 exact completed replay error = %v", err)
	}
	if records != 1 {
		t.Fatalf("TC-105 response writes = %d, want 1", records)
	}
}

// TC-105: Idempotency is a four-field durable identity. Every partial match
// remains a rejected retry and cannot reach the repository audit transaction.
func TestQuestionServiceRecordResponseRejectsConflictingCompletedReplay_TC105(t *testing.T) {
	state := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders: []models.QuestionResponder{
			{Identity: "alice", Status: models.QuestionResponderCompleted},
			{Identity: "bob", Status: models.QuestionResponderPending},
		},
		Responses: []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/plan/spec.md", RecordedAt: time.Now().UTC()}},
	}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []RecordQuestionResponseInput{
		{Key: "Q001", SessionID: "session-b", Responder: "alice", Summary: "approved", EvidencePointer: "docs/plan/spec.md"},
		{Key: "Q001", SessionID: "session-a", Responder: "bob", Summary: "approved", EvidencePointer: "docs/plan/spec.md"},
		{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "changed", EvidencePointer: "docs/plan/spec.md"},
		{Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/plan/other.md"},
	} {
		t.Run(input.SessionID+input.Responder+input.Summary+input.EvidencePointer, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", Title: "Question", ContextData: encoded}, Status: "answering", Summary: "Summary", Requester: "owner"}
			writes := 0
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				recordResponseFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string) error {
					writes++
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			svc.SetClaimReader(fakeQuestionClaimReader{})
			if _, err := svc.RecordResponse(context.Background(), input); err == nil {
				t.Fatal("TC-105 conflicting completed replay error = nil")
			}
			if writes != 0 || question.ContextData != encoded {
				t.Fatalf("TC-105 conflicting replay writes=%d context=%v", writes, question.ContextData)
			}
		})
	}
}

// TC-106: Resolve calls the production service entrypoint, validates every
// classified destination before the atomic Question-only repository write, and
// never delegates a linked-record mutation.
func TestQuestionServiceResolve_TC106(t *testing.T) {
	readyState := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	encoded, err := models.EncodeQuestionState(nil, readyState)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ kind, pointer string }{
		{"local_clarification", "note:Q001:42"},
		{"feature_change", "docs/architecture/coding-standards.md"},
		{"product_decision", "docs/product/progress.md#decision-1"},
		{"architecture_decision", "docs/architecture/coding-standards.md;docs/architecture/architecture-overview.md"},
		{"follow_up_work", "E39-F02"},
		{"no_lasting_consequence", ""},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: encoded}, Status: "ready_for_resolution", Summary: "Summary", Requester: "owner"}
			resolved := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				resolveFn: func(_ context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, data *string, owner, kind string) error {
					if expectedStatus != "ready_for_resolution" || expectedContextData == nil {
						t.Fatalf("Resolve expected snapshot = (%q,%v)", expectedStatus, expectedContextData)
					}
					resolved = true
					if id != 39 || status != "resolved" || owner != "release-owner" || kind != tc.kind || data == nil {
						t.Fatalf("Resolve args = (%d,%q,%q,%q,%v)", id, status, owner, kind, data)
					}
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			svc.SetProjectRoot("../..")
			if _, err := svc.Resolve(context.Background(), ResolveQuestionInput{Key: "Q001", Owner: "release-owner", Kind: tc.kind, Pointer: tc.pointer}); err != nil {
				t.Fatalf("TC-106 Resolve() error = %v", err)
			}
			if !resolved {
				t.Fatal("TC-106 Resolve did not persist")
			}
		})
	}
}

// TestQuestionServiceResolveRejectsUnreadyQuestion_TC106 locks in Resolve's
// core precondition: status must be ready_for_resolution AND every responder
// must have completed. Without this guard, a caller could resolve a Question
// while responders are still pending or before any response was recorded.
func TestQuestionServiceResolveRejectsUnreadyQuestion_TC106(t *testing.T) {
	pendingState := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}, {Identity: "bob", Status: models.QuestionResponderPending}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	pendingEncoded, err := models.EncodeQuestionState(nil, pendingState)
	if err != nil {
		t.Fatal(err)
	}
	readyState := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	readyEncoded, err := models.EncodeQuestionState(nil, readyState)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		status models.QuestionStatus
		ctx    *string
	}{
		{"draft_status", models.QuestionStatusDraft, readyEncoded},
		{"open_status", models.QuestionStatusOpen, readyEncoded},
		{"answering_status", models.QuestionStatusAnswering, readyEncoded},
		{"ready_for_resolution_with_pending_responder", models.QuestionStatusReadyForResolution, pendingEncoded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: tc.ctx}, Status: tc.status, Summary: "Summary", Requester: "owner"}
			resolveCalled := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				resolveFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error {
					resolveCalled = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			svc.SetProjectRoot("../..")
			if _, err := svc.Resolve(context.Background(), ResolveQuestionInput{Key: "Q001", Owner: "release-owner", Kind: "no_lasting_consequence"}); err == nil {
				t.Fatal("Resolve() error = nil, want rejection for a Question that is not ready for resolution")
			}
			if resolveCalled {
				t.Fatal("Resolve() called the repository write despite failing its precondition")
			}
		})
	}
}

// TestQuestionServiceResolveRejectsInvalidDestination_TC106 locks in
// validateResolutionDestination/validateResolutionDocument's negative paths,
// including the path-traversal guard (filepath.IsAbs / "../" escape) on
// feature_change/architecture_decision document pointers. Without this,
// stubbing either validator to always return nil would still pass the full
// suite -- these are the guards that gate every Resolve() call.
func TestQuestionServiceResolveRejectsInvalidDestination_TC106(t *testing.T) {
	readyState := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	encoded, err := models.EncodeQuestionState(nil, readyState)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name           string
		kind, pointer  string
		followUpExists bool
		noteExists     bool
	}{
		{name: "local_clarification_missing_note_prefix", kind: "local_clarification", pointer: "Q001:42", followUpExists: true, noteExists: true},
		{name: "local_clarification_nonexistent_note", kind: "local_clarification", pointer: "note:404", followUpExists: true, noteExists: false},
		{name: "follow_up_work_nonexistent_destination", kind: "follow_up_work", pointer: "E99-F99", followUpExists: false, noteExists: true},
		{name: "product_decision_missing_anchor", kind: "product_decision", pointer: "docs/product/progress.md", followUpExists: true, noteExists: true},
		{name: "product_decision_wrong_document", kind: "product_decision", pointer: "docs/other.md#decision-1", followUpExists: true, noteExists: true},
		{name: "architecture_decision_single_path", kind: "architecture_decision", pointer: "docs/architecture/coding-standards.md", followUpExists: true, noteExists: true},
		{name: "feature_change_nonexistent_file", kind: "feature_change", pointer: "docs/architecture/does-not-exist.md", followUpExists: true, noteExists: true},
		{name: "feature_change_absolute_path", kind: "feature_change", pointer: "/etc/passwd", followUpExists: true, noteExists: true},
		{name: "feature_change_path_traversal", kind: "feature_change", pointer: "../../../../etc/passwd", followUpExists: true, noteExists: true},
		{name: "architecture_decision_one_path_escapes", kind: "architecture_decision", pointer: "docs/architecture/coding-standards.md;../../../../etc/passwd", followUpExists: true, noteExists: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: encoded}, Status: "ready_for_resolution", Summary: "Summary", Requester: "owner"}
			resolveCalled := false
			repo := &mockQuestionRepository{
				getByKeyFn:           func(context.Context, string) (*models.Question, error) { return question, nil },
				followUpWorkExistsFn: func(context.Context, string) (bool, error) { return tc.followUpExists, nil },
				noteExistsFn:         func(context.Context, string) (bool, error) { return tc.noteExists, nil },
				resolveFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error {
					resolveCalled = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			svc.SetProjectRoot("../..")
			if _, err := svc.Resolve(context.Background(), ResolveQuestionInput{Key: "Q001", Owner: "release-owner", Kind: tc.kind, Pointer: tc.pointer}); err == nil {
				t.Fatalf("Resolve(kind=%q, pointer=%q) error = nil, want rejection", tc.kind, tc.pointer)
			}
			if resolveCalled {
				t.Fatalf("Resolve(kind=%q, pointer=%q) called the repository write despite invalid destination", tc.kind, tc.pointer)
			}
		})
	}
}

// TC-107: terminal provenance accepts only the configured owner and leaves
// target validation and atomic terminal write at the typed repository seam.
func TestQuestionServiceWithdrawAndSupersede_TC107(t *testing.T) {
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		status models.QuestionStatus
		call   func(*QuestionService) error
	}{
		{name: "withdraw", status: "withdrawn", call: func(s *QuestionService) error {
			_, err := s.Withdraw(context.Background(), WithdrawQuestionInput{Key: "Q001", Owner: "release-owner", Reason: "no longer needed"})
			return err
		}},
		{name: "supersede", status: "superseded", call: func(s *QuestionService) error {
			_, err := s.Supersede(context.Background(), SupersedeQuestionInput{Key: "Q001", Owner: "release-owner", Reason: "replaced", SupersededBy: "Q002"})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: encoded}, Status: "open", Summary: "Summary", Requester: "owner"}
			called := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
					if key == "Q002" {
						return &models.Question{BaseEntity: models.BaseEntity{ID: 40, Key: "Q002"}}, nil
					}
					return question, nil
				},
				withdrawFn: func(_ context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, data *string, owner, reason string) error {
					if expectedStatus != "open" || expectedContextData == nil {
						t.Fatalf("Withdraw expected snapshot = (%q,%v)", expectedStatus, expectedContextData)
					}
					called = true
					if id != 39 || status != tc.status || data == nil || owner != "release-owner" || reason == "" {
						t.Fatalf("Withdraw args invalid")
					}
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.call(svc); err != nil {
				t.Fatalf("TC-107 %s error = %v", tc.name, err)
			}
			if !called {
				t.Fatalf("TC-107 %s did not call atomic repository write", tc.name)
			}
		})
	}
}

// TestQuestionServiceWithdrawAndSupersedeRejectGuardViolations_TC107 locks in
// the two guards closeWithReason/loadClosableQuestion enforce before Withdraw
// or Supersede may close a Question: the caller must be the configured
// resolution owner, and the Question must not already be terminal. Without
// these, any caller could re-close an already-resolved Question or close one
// they don't own.
func TestQuestionServiceWithdrawAndSupersedeRejectGuardViolations_TC107(t *testing.T) {
	openState := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	openEncoded, err := models.EncodeQuestionState(nil, openState)
	if err != nil {
		t.Fatal(err)
	}
	terminalState := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	terminalEncoded, err := models.EncodeQuestionState(nil, terminalState)
	if err != nil {
		t.Fatal(err)
	}

	for _, action := range []struct {
		name string
		call func(*QuestionService) error
	}{
		{name: "withdraw", call: func(s *QuestionService) error {
			_, err := s.Withdraw(context.Background(), WithdrawQuestionInput{Key: "Q001", Owner: "release-owner", Reason: "no longer needed"})
			return err
		}},
		{name: "supersede", call: func(s *QuestionService) error {
			_, err := s.Supersede(context.Background(), SupersedeQuestionInput{Key: "Q001", Owner: "release-owner", Reason: "replaced", SupersededBy: "Q002"})
			return err
		}},
	} {
		t.Run(action.name+"/wrong_owner", func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: openEncoded}, Status: "open", Summary: "Summary", Requester: "owner"}
			called := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
					if key == "Q002" {
						return &models.Question{BaseEntity: models.BaseEntity{ID: 40, Key: "Q002"}}, nil
					}
					return question, nil
				},
				withdrawFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error {
					called = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			// "someone-else" does not match the configured release-owner.
			var callErr error
			switch action.name {
			case "withdraw":
				_, callErr = svc.Withdraw(context.Background(), WithdrawQuestionInput{Key: "Q001", Owner: "someone-else", Reason: "no longer needed"})
			case "supersede":
				_, callErr = svc.Supersede(context.Background(), SupersedeQuestionInput{Key: "Q001", Owner: "someone-else", Reason: "replaced", SupersededBy: "Q002"})
			}
			if callErr == nil {
				t.Fatal("error = nil, want rejection for a caller that is not the configured resolution owner")
			}
			if called {
				t.Fatal("repository write called despite the owner mismatch")
			}
		})

		t.Run(action.name+"/already_terminal", func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: terminalEncoded}, Status: models.QuestionStatusResolved, Summary: "Summary", Requester: "owner"}
			called := false
			repo := &mockQuestionRepository{
				getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
					if key == "Q002" {
						return &models.Question{BaseEntity: models.BaseEntity{ID: 40, Key: "Q002"}}, nil
					}
					return question, nil
				},
				withdrawFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus, *string, *string, string, string) error {
					called = true
					return nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := action.call(svc); err == nil {
				t.Fatal("error = nil, want rejection for an already-terminal (resolved) Question")
			}
			if called {
				t.Fatal("repository write called despite the Question already being terminal")
			}
		})
	}
}

// TestValidateTerminalReasonEnforcesByteBoundaries locks in
// validateTerminalReason's byte-range check, used by Withdraw/Supersede to
// bound the caller-supplied close reason. Zero tests exercised this helper
// before -- an empty or oversized reason would previously go unchecked here.
func TestValidateTerminalReasonEnforcesByteBoundaries(t *testing.T) {
	if err := validateTerminalReason(""); err == nil {
		t.Error("validateTerminalReason(\"\") error = nil, want rejection of an empty reason")
	}
	if err := validateTerminalReason(strings.Repeat("a", 1001)); err == nil {
		t.Error("validateTerminalReason(1001 bytes) error = nil, want rejection over the 1000-byte bound")
	}
	if err := validateTerminalReason(strings.Repeat("a", 1000)); err != nil {
		t.Errorf("validateTerminalReason(1000 bytes) error = %v, want acceptance at the bound", err)
	}
	if err := validateTerminalReason("a"); err != nil {
		t.Errorf("validateTerminalReason(\"a\") error = %v, want acceptance of a minimal reason", err)
	}
}

type fakeQuestionClaimReader struct{ claim *models.EntityClaim }

func (f fakeQuestionClaimReader) Get(context.Context, string, string) (*models.EntityClaim, error) {
	return f.claim, nil
}

// TC-013: Creating a Question establishes its initial status in the shared
// entity history stream so the runtime audit and delete cleanup paths have a
// durable record to operate on.
func TestQuestionServiceCreateQuestionRecordsInitialHistory(t *testing.T) {
	persisted := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001"}, Status: models.QuestionStatusDraft, Summary: "Confirm gate", Requester: "release-manager"}
	repo := &mockQuestionRepository{
		createFn: func(_ context.Context, question *models.Question) error {
			question.ID = persisted.ID
			question.Key = persisted.Key
			return nil
		},
		getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
			if key != persisted.Key {
				t.Fatalf("reload key = %q, want %q", key, persisted.Key)
			}
			return persisted, nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	historyRecorder := &mockEntityHistoryRecorder{}
	svc.SetHistoryRepo(historyRecorder)

	if _, err := svc.CreateQuestion(context.Background(), CreateQuestionInput{
		Title: "Release gate", Summary: "Confirm gate", Requester: "release-manager",
	}); err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	if len(historyRecorder.created) != 1 {
		t.Fatalf("history records = %d, want 1", len(historyRecorder.created))
	}
	history := historyRecorder.created[0]
	if history.EntityType != models.EntityTypeQuestion || history.EntityID != persisted.ID || history.FromStatus != nil || history.ToStatus != string(models.QuestionStatusDraft) || history.Forced {
		t.Fatalf("creation history = %#v, want Question initial draft record", history)
	}
}

func TestQuestionServiceCreateQuestionRejectsInvalidInputBeforeMutation(t *testing.T) {
	cases := []CreateQuestionInput{
		{Summary: "summary", Requester: "requester"},
		{Title: "title", Requester: "requester"},
		{Title: "title", Summary: "summary"},
	}
	for _, input := range cases {
		t.Run("invalid", func(t *testing.T) {
			called := false
			svc, err := NewQuestionService(&mockQuestionRepository{createFn: func(context.Context, *models.Question) error {
				called = true
				return nil
			}})
			if err != nil {
				t.Fatalf("NewQuestionService() error = %v", err)
			}
			if _, err := svc.CreateQuestion(context.Background(), input); err == nil {
				t.Fatal("CreateQuestion() error = nil, want validation error")
			}
			if called {
				t.Fatal("CreateQuestion() called repository for invalid input")
			}
		})
	}
}

func TestNewQuestionServiceRejectsNilRepository(t *testing.T) {
	_, err := NewQuestionService(nil)
	if err == nil || !strings.Contains(err.Error(), "QuestionRepository") {
		t.Fatalf("NewQuestionService(nil) error = %v, want non-nil repository error", err)
	}
}

func TestQuestionServiceReadListUpdateAndStatus(t *testing.T) {
	title := "  Updated title  "
	blocking := true
	stored := &models.Question{BaseEntity: models.BaseEntity{ID: 7, Key: "Q001", Title: "Original"}, Status: models.QuestionStatusDraft, Summary: "Summary", Requester: "Requester"}
	updated := false
	repo := &mockQuestionRepository{
		getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
			if key != "Q001" {
				t.Fatalf("key = %q", key)
			}
			return stored, nil
		},
		listFn: func(_ context.Context, filter questionrepo.QuestionListFilter) ([]*models.Question, error) {
			if filter.Limit != 50 {
				t.Fatalf("limit = %d", filter.Limit)
			}
			return []*models.Question{stored}, nil
		},
		updateFn: func(_ context.Context, question *models.Question) error {
			updated = true
			if question.Title != "Updated title" || !question.Blocking {
				t.Fatalf("update = %#v", question)
			}
			return nil
		},
		statusFn: func(_ context.Context, id int64, status models.QuestionStatus) error {
			if id != 7 || status != "archived" {
				t.Fatalf("status update = %d %q", id, status)
			}
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := svc.GetQuestion(context.Background(), "Q001"); err != nil || got != stored {
		t.Fatalf("GetQuestion() = %#v, %v", got, err)
	}
	if got, err := svc.ListQuestions(context.Background(), QuestionListFilter{Limit: 50}); err != nil || len(got) != 1 {
		t.Fatalf("ListQuestions() = %#v, %v", got, err)
	}
	if got, err := svc.UpdateQuestion(context.Background(), "Q001", QuestionUpdates{Title: &title, Blocking: &blocking}); err != nil || got != stored || !updated {
		t.Fatalf("UpdateQuestion() = %#v, %v", got, err)
	}
	if got, err := svc.SetQuestionStatus(context.Background(), "Q001", " ARCHIVED "); err != nil || got.Status != "archived" {
		t.Fatalf("SetQuestionStatus() = %#v, %v", got, err)
	}
}

// TC-402: the service derives the responder only from validated QuestionState
// and keeps the compact transport projection free of raw persisted context.
// TestNormalizeQuestionReadPageRejectsOutOfRangeBounds locks in
// normalizeQuestionReadPage's own boundary checks directly at the service
// layer -- previously this was only exercised indirectly through CLI/HTTP
// transports that pre-validate the same range before calling in.
func TestNormalizeQuestionReadPageRejectsOutOfRangeBounds(t *testing.T) {
	if _, err := normalizeQuestionReadPage(0, 0); err != nil {
		t.Errorf("normalizeQuestionReadPage(0, 0) error = %v, want the default limit accepted", err)
	}
	if limit, err := normalizeQuestionReadPage(100, 0); err != nil || limit != 100 {
		t.Errorf("normalizeQuestionReadPage(100, 0) = (%d, %v), want (100, nil) at the upper bound", limit, err)
	}
	if _, err := normalizeQuestionReadPage(101, 0); err == nil {
		t.Error("normalizeQuestionReadPage(101, 0) error = nil, want rejection over the 100 limit")
	}
	if _, err := normalizeQuestionReadPage(-1, 0); err == nil {
		t.Error("normalizeQuestionReadPage(-1, 0) error = nil, want rejection of a negative limit")
	}
	if _, err := normalizeQuestionReadPage(50, -1); err == nil {
		t.Error("normalizeQuestionReadPage(50, -1) error = nil, want rejection of a negative offset")
	}
	if limit, err := normalizeQuestionReadPage(50, 0); err != nil || limit != 50 {
		t.Errorf("normalizeQuestionReadPage(50, 0) = (%d, %v), want (50, nil)", limit, err)
	}
}

func TestQuestionServiceListOpenQuestionsByResponderTC402(t *testing.T) {
	state := models.QuestionState{ResolutionOwner: "owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	malformed := "{not-json"
	matching := &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "matching", ContextData: encoded}, Status: "open", Summary: "summary", Requester: "requester"}
	notMatching := &models.Question{BaseEntity: models.BaseEntity{Key: "Q002", Title: "not matching", ContextData: encoded}, Status: "ready_for_resolution", Summary: "summary", Requester: "requester"}
	repo := &mockQuestionRepository{listOpenCandidatesFn: func(_ context.Context, limit, offset int) ([]*models.Question, error) {
		if limit != 50 || offset != 0 {
			t.Fatalf("candidate page = (%d, %d), want (50, 0)", limit, offset)
		}
		return []*models.Question{matching, notMatching}, nil
	}}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.ListOpenQuestionsByResponder(context.Background(), "alice", 0, 0)
	if err != nil {
		t.Fatalf("TC-402 ListOpenQuestionsByResponder() error = %v", err)
	}
	if len(got) != 1 || got[0].Key != "Q001" {
		t.Fatalf("TC-402 page = %#v, want compact Q001 only", got)
	}
	encodedProjection, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedProjection), "context_data") {
		t.Fatalf("TC-402 compact projection leaked context: %s", encodedProjection)
	}

	matching.ContextData = &malformed
	if _, err := svc.ListOpenQuestionsByResponder(context.Background(), "alice", 50, 0); err == nil {
		t.Fatal("TC-402 malformed state error = nil, want actionable error")
	}
}

// TC-405: only the current responder and resolution owner receive the
// deliberate full projection; no caller receives raw ContextData.
func TestQuestionServiceReadQuestionFullTC405(t *testing.T) {
	recordedAt := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	state := models.QuestionState{
		ResolutionOwner: "owner",
		Responders: []models.QuestionResponder{
			{Identity: "bob", Status: models.QuestionResponderCompleted},
			{Identity: "alice", Status: models.QuestionResponderPending},
		},
		Responses: []models.QuestionResponse{{SessionID: "session-bob", Responder: "bob", Summary: "answered", EvidencePointer: "note:1", RecordedAt: recordedAt}},
	}
	contextData, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	question := &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Question", ContextData: contextData}, Status: "answering", Summary: "summary", Requester: "requester"}
	svc, err := NewQuestionService(&mockQuestionRepository{getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
		if key != "Q001" {
			t.Fatalf("key = %q", key)
		}
		return question, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []string{"alice", "owner"} {
		full, readErr := svc.ReadQuestionFull(context.Background(), "Q001", actor)
		if readErr != nil || full.ResolutionOwner != "owner" || len(full.Responders) != 2 || len(full.Responses) != 1 {
			t.Fatalf("TC-405 actor %q full = %#v, %v", actor, full, readErr)
		}
		encodedFull, marshalErr := json.Marshal(full)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if strings.Contains(string(encodedFull), "context_data") {
			t.Fatalf("TC-405 full projection leaked raw context: %s", encodedFull)
		}
	}
	for _, actor := range []string{"", "bob", "mallory"} {
		if _, readErr := svc.ReadQuestionFull(context.Background(), "Q001", actor); readErr == nil {
			t.Fatalf("TC-405 actor %q error = nil, want denied/validation", actor)
		}
	}
}

// TC-404: focused blocking reads use the same direct F03 qualification and
// deterministic edge order as the dispatch preflight.
func TestQuestionServiceListQuestionsBlockingTC404(t *testing.T) {
	state := models.QuestionState{ResolutionOwner: "owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]*models.Question{
		1: {BaseEntity: models.BaseEntity{ID: 1, Key: "Q001", ContextData: encoded}, Status: "open", Summary: "first", Blocking: true},
		2: {BaseEntity: models.BaseEntity{ID: 2, Key: "Q002", ContextData: encoded}, Status: "answering", Summary: "second", Blocking: true},
		3: {BaseEntity: models.BaseEntity{ID: 3, Key: "Q003", ContextData: encoded}, Status: "open", Summary: "excluded", Blocking: false},
	}
	repo := &mockQuestionRepository{getByIDFn: func(_ context.Context, id int64) (*models.Question, error) { return byID[id], nil }}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	relationships := &focusedRelationshipReader{edges: []*models.EntityRelationship{
		{ID: 8, FromEntityType: models.EntityTypeQuestion, FromEntityID: 2, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)},
		{ID: 7, FromEntityType: models.EntityTypeQuestion, FromEntityID: 1, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)},
		{ID: 9, FromEntityType: models.EntityTypeQuestion, FromEntityID: 3, RelationshipType: models.EntityRelQuestionBlocks, CreatedAt: time.Date(2026, 7, 31, 12, 0, 1, 0, time.UTC)},
	}}
	svc.SetFocusedReadDependencies(relationships, questionBlockerRegistry{repo: &questionBlockerEntityRepo{entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 8, Key: "F001"}}}})
	got, err := svc.ListQuestionsBlocking(context.Background(), models.EntityTypeFeature, "F001", 50, 0)
	if err != nil {
		t.Fatalf("TC-404 ListQuestionsBlocking() error = %v", err)
	}
	if !relationships.called || len(got) != 2 || got[0].QuestionKey != "Q001" || got[1].QuestionKey != "Q002" {
		t.Fatalf("TC-404 blocks = %#v, want Q001/Q002 in F03 order", got)
	}
	if _, err := svc.ListQuestionsBlocking(context.Background(), models.EntityTypeQuestion, "Q001", 50, 0); err == nil {
		t.Fatal("TC-404 Question target error = nil, want validation")
	}
}

func TestQuestionServiceUpdateRejectsUnsupportedEmptyMutation(t *testing.T) {
	repo := &mockQuestionRepository{getByKeyFn: func(context.Context, string) (*models.Question, error) { t.Fatal("unexpected lookup"); return nil, nil }}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateQuestion(context.Background(), "Q001", QuestionUpdates{}); err == nil {
		t.Fatal("UpdateQuestion() error = nil")
	}
}

// TC-103: keyed next consumes QuestionService through runner.EntityTransitioner.
// A pre-configuration Question must pause rather than render a responder
// prompt: no identity may be invented before ConfigureWorkflow persists the
// bounded serial responder state.
func TestQuestionServiceGetNextStatusUnconfiguredPauses_TC103(t *testing.T) {
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 8, Key: "Q001", Title: "Question"}, Status: models.QuestionStatusDraft, Summary: "Summary", Requester: "owner"}
	repo := &mockQuestionRepository{getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil }}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	info, err := svc.GetNextStatus(context.Background(), "Q001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if info.EntityType != models.EntityTypeQuestion || info.EntityKey != "Q001" || info.CurrentStatus != "draft" || !info.IsTerminal || len(info.AvailableTransitions) != 0 {
		t.Fatalf("GetNextStatus() = %#v, want terminal compatibility pause for unconfigured draft", info)
	}

	question.Status = models.QuestionStatus("archived")
	info, err = svc.GetNextStatus(context.Background(), "Q001")
	if err != nil {
		t.Fatalf("GetNextStatus(archived) error = %v", err)
	}
	if !info.IsTerminal || info.CurrentStatus != "archived" {
		t.Fatalf("GetNextStatus(archived) = %#v, want terminal archived", info)
	}
}

// TestQuestionServiceGetNextStatusOpenWithNilContextDataIsTerminal locks in
// GetNextStatus's "state == nil || state.CurrentResponder() == \"\"" guard
// for the open/answering branch. Without it, a migrated-but-unconfigured
// Question sitting in open/answering (see migrateQuestionDraftsToOpen) would
// reach state.CurrentResponder() on a nil *QuestionState and panic instead
// of reporting a safe terminal pause.
func TestQuestionServiceGetNextStatusOpenWithNilContextDataIsTerminal(t *testing.T) {
	for _, status := range []models.QuestionStatus{models.QuestionStatusOpen, models.QuestionStatusAnswering} {
		t.Run(string(status), func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 8, Key: "Q001", ContextData: nil}, Status: status, Summary: "Summary", Requester: "owner"}
			repo := &mockQuestionRepository{getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil }}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			info, err := svc.GetNextStatus(context.Background(), "Q001")
			if err != nil {
				t.Fatalf("GetNextStatus() error = %v", err)
			}
			if !info.IsTerminal || len(info.AvailableTransitions) != 0 {
				t.Fatalf("GetNextStatus() = %#v, want terminal pause for nil context data", info)
			}
		})
	}
}

// TC-103: a fully answered Question has no responder to render. It remains a
// human resolution checkpoint, but every dispatch entry point must receive a
// non-dispatching result before attempting responder placeholder generation.
func TestQuestionServiceGetNextStatusReadyForResolutionStopsResponderDispatch_TC103(t *testing.T) {
	state := models.QuestionState{
		ResolutionOwner: "owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses: []models.QuestionResponse{{
			SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC(),
		}},
	}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	question := &models.Question{BaseEntity: models.BaseEntity{ID: 8, Key: "Q001", ContextData: encoded}, Status: "ready_for_resolution"}
	repo := &mockQuestionRepository{getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil }}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}

	info, err := svc.GetNextStatus(context.Background(), question.Key)
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if !info.IsTerminal || info.IsClaimed || len(info.AvailableTransitions) != 0 {
		t.Fatalf("GetNextStatus() = %#v, want non-dispatching ready-for-resolution checkpoint", info)
	}
}

// TestQuestionServiceTransitionStatusRejectsWorkflowOwnedTargets locks in a
// shipped-and-caught regression: TransitionStatus satisfies
// runner.EntityTransitioner and is reachable directly via `shark status set
// <key> <status>` and the HTTP transition endpoint, NEITHER of which consult
// GetNextStatus's restricted AvailableTransitions/Outcomes first (unlike
// `shark status advance`). Delegating unconditionally to
// EntityService.TransitionStatus let a caller force a Question through
// ready_for_resolution/resolved/withdrawn/superseded via question.yaml's
// ordinary forward edges, completely bypassing RecordResponse/Resolve/
// Withdraw/Supersede's responder-completion, resolution-owner, and
// provenance checks -- and silently defeating the F03 blocking gate for
// anything linked to that Question. This test exercises exactly the
// delegated (entitySvc/entityRepo wired) path production uses via
// SetEntityTransitioner, not just the mock-backed fallback, since that's
// the boundary the bug shipped on.
func TestQuestionServiceTransitionStatusRejectsWorkflowOwnedTargets(t *testing.T) {
	state := models.QuestionState{
		ResolutionOwner: "owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}},
		Responses:       []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}},
	}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"ready_for_resolution", "resolved", "withdrawn", "superseded"} {
		t.Run(target, func(t *testing.T) {
			question := &models.Question{BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", ContextData: encoded}, Status: models.QuestionStatusOpen, Summary: "Summary", Requester: "owner"}
			repo := &mockQuestionRepository{
				getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
				statusFn: func(context.Context, int64, models.QuestionStatus) error {
					t.Fatal("repository status write called despite rejected workflow-owned target")
					return nil
				},
				updateStatusIfCurrentFn: func(context.Context, int64, models.QuestionStatus, models.QuestionStatus) (bool, error) {
					t.Fatal("repository conditional status write called despite rejected workflow-owned target")
					return false, nil
				},
			}
			svc, err := NewQuestionService(repo)
			if err != nil {
				t.Fatal(err)
			}
			entitySvc := NewEntityService(workflow.NewService(""))
			svc.SetEntityTransitioner(entitySvc, NewQuestionRepositoryAdapter(repo))

			if _, err := svc.TransitionStatus(context.Background(), "Q001", target, TransitionOptions{}); err == nil {
				t.Fatalf("TransitionStatus(target=%q) error = nil, want rejection", target)
			}
		})
	}
}

// TC-103/TC-104: A parent owns the Question lease and must be able to advance
// its worker stage while that lease remains live. A live claim blocks a second
// keyed dispatch, not the parent's status advance. The response write is
// intentionally explicit here: worker prompts cannot mutate Shark state.
func TestQuestionServiceParentLeaseLifecycleAdvancesThenRoutesNextResponder_TC103_TC104(t *testing.T) {
	state := models.QuestionState{
		ResolutionOwner: "owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}, {Identity: "bob", Status: models.QuestionResponderPending}},
	}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	question := &models.Question{
		BaseEntity: models.BaseEntity{ID: 39, Key: "Q001", Title: "Question", ContextData: encoded},
		Status:     models.QuestionStatusOpen,
		Summary:    "Summary",
		Requester:  "owner",
	}
	repo := &mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) { return question, nil },
		statusFn: func(_ context.Context, id int64, status models.QuestionStatus) error {
			if id != question.ID {
				t.Fatalf("UpdateStatus id = %d, want %d", id, question.ID)
			}
			question.Status = status
			return nil
		},
		recordResponseFn: func(_ context.Context, _ int64, _ models.QuestionStatus, status models.QuestionStatus, _ *string, data *string, responder string) error {
			if responder != "alice" {
				t.Fatalf("RecordResponse responder = %q, want alice", responder)
			}
			question.Status, question.ContextData = status, data
			return nil
		},
	}
	claimReader := &fakeQuestionClaimReader{claim: &models.EntityClaim{EntityType: "question", EntityKey: "Q001", ClaimedBy: "alice", SessionID: "session-a"}}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatal(err)
	}
	svc.SetClaimReader(claimReader)

	// next -> claim: the active lease suppresses competing dispatch only.
	info, err := svc.GetNextStatus(context.Background(), "Q001")
	if err != nil {
		t.Fatal(err)
	}
	if info.IsTerminal || !info.IsClaimed || len(info.AvailableTransitions) != 1 || info.AvailableTransitions[0].TargetStatus != "answering" {
		t.Fatalf("claimed Question next status = %#v, want non-terminal parent-advance state", info)
	}

	// Parent-owned status advance precedes the durable response write.
	if _, err := svc.TransitionStatus(context.Background(), "Q001", "answering", TransitionOptions{}); err != nil {
		t.Fatalf("parent status advance while claimed: %v", err)
	}
	if _, err := svc.RecordResponse(context.Background(), RecordQuestionResponseInput{
		Key: "Q001", SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md",
	}); err != nil {
		t.Fatalf("parent-mediated response write: %v", err)
	}

	// release -> next: alice remains completed and only bob is eligible.
	claimReader.claim = nil
	info, err = svc.GetNextStatus(context.Background(), "Q001")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := models.DecodeQuestionState(question.ContextData)
	if err != nil || decoded == nil || decoded.CurrentResponder() != "bob" || info.IsTerminal || info.IsClaimed {
		t.Fatalf("released Question next state = %#v, info=%#v, err=%v; want only bob pending", decoded, info, err)
	}
}

// TC-009: DeleteQuestion must resolve the canonical Question before asking the
// persistence owner to delete it. SQLite cleanup atomicity is covered by the
// repository integration test, not this mock-backed service test.
func TestQuestionServiceDeleteQuestionResolvesThenDeletes(t *testing.T) {
	var deletedID int64
	repo := &mockQuestionRepository{
		getByKeyFn: func(_ context.Context, key string) (*models.Question, error) {
			if key != "Q001" {
				t.Fatalf("GetByKey() key = %q, want Q001", key)
			}
			return &models.Question{BaseEntity: models.BaseEntity{ID: 42, Key: "Q001"}}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	svc, err := NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	if err := svc.DeleteQuestion(context.Background(), "Q001"); err != nil {
		t.Fatalf("DeleteQuestion() error = %v", err)
	}
	if deletedID != 42 {
		t.Errorf("DeleteQuestion() deleted ID %d, want 42", deletedID)
	}
}

func TestQuestionServiceDeleteQuestionDoesNotDeleteWhenLookupFails(t *testing.T) {
	lookupErr := errors.New("question missing")
	deleteCalled := false
	svc, err := NewQuestionService(&mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) { return nil, lookupErr },
		deleteFn: func(context.Context, int64) error {
			deleteCalled = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	err = svc.DeleteQuestion(context.Background(), "Q404")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("DeleteQuestion() error = %v, want wrapped lookup error", err)
	}
	if deleteCalled {
		t.Fatal("DeleteQuestion() called Delete after lookup failure")
	}
}

func TestQuestionServiceDeleteQuestionWrapsRepositoryDeleteFailure(t *testing.T) {
	cleanupErr := errors.New("cleanup trigger failed")
	svc, err := NewQuestionService(&mockQuestionRepository{
		getByKeyFn: func(context.Context, string) (*models.Question, error) {
			return &models.Question{BaseEntity: models.BaseEntity{ID: 42, Key: "Q001"}}, nil
		},
		deleteFn: func(context.Context, int64) error { return cleanupErr },
	})
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	err = svc.DeleteQuestion(context.Background(), "Q001")
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("DeleteQuestion() error = %v, want wrapped cleanup failure", err)
	}
}
