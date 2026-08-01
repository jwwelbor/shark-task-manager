package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type mockMutationServicer struct {
	updateEpicFn        func(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	updateFeatureFn     func(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	updateTaskFn        func(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	transitionEpicFn    func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	transitionFeatureFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	transitionTaskFn    func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	addNoteFn           func(ctx context.Context, entityKey, noteType, content, createdBy string) (*models.EntityNote, error)
	createRelFn         func(ctx context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error)
	deleteRelFn         func(ctx context.Context, fromKey, relType, toKey string) error

	updateEpicCalls        int
	updateFeatureCalls     int
	updateTaskCalls        int
	transitionEpicCalls    int
	transitionFeatureCalls int
	transitionTaskCalls    int
	addNoteCalls           int
	createRelCalls         int
	deleteRelCalls         int
}

func (m *mockMutationServicer) UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
	m.updateEpicCalls++
	if m.updateEpicFn != nil {
		return m.updateEpicFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockMutationServicer) UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
	m.updateFeatureCalls++
	if m.updateFeatureFn != nil {
		return m.updateFeatureFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockMutationServicer) UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
	m.updateTaskCalls++
	if m.updateTaskFn != nil {
		return m.updateTaskFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockMutationServicer) TransitionEpic(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionEpicCalls++
	if m.transitionEpicFn != nil {
		return m.transitionEpicFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockMutationServicer) TransitionFeature(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionFeatureCalls++
	if m.transitionFeatureFn != nil {
		return m.transitionFeatureFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockMutationServicer) TransitionTask(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	m.transitionTaskCalls++
	if m.transitionTaskFn != nil {
		return m.transitionTaskFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockMutationServicer) AddNote(ctx context.Context, entityKey, noteType, content, createdBy string) (*models.EntityNote, error) {
	m.addNoteCalls++
	if m.addNoteFn != nil {
		return m.addNoteFn(ctx, entityKey, noteType, content, createdBy)
	}
	return nil, nil
}

func (m *mockMutationServicer) CreateRelationship(ctx context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error) {
	m.createRelCalls++
	if m.createRelFn != nil {
		return m.createRelFn(ctx, fromKey, toKey, relType)
	}
	return nil, nil
}

func (m *mockMutationServicer) DeleteRelationship(ctx context.Context, fromKey, relType, toKey string) error {
	m.deleteRelCalls++
	if m.deleteRelFn != nil {
		return m.deleteRelFn(ctx, fromKey, relType, toKey)
	}
	return nil
}

func newMutationHandlerMux(svc MutationServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewMutationHandler(svc).RegisterRoutes(mux, "/api/v1/viewer")
	return mux
}

func TestMutationHandler_UpdatePaths(t *testing.T) {
	t.Run("epic", func(t *testing.T) {
		title := "Updated Epic"
		priority := "high"
		businessValue := "medium"
		size := 8
		mock := &mockMutationServicer{
			updateEpicFn: func(_ context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
				if key != "E07" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.Title == nil || *updates.Title != title {
					t.Fatalf("unexpected title: %#v", updates.Title)
				}
				if updates.Priority == nil || string(*updates.Priority) != priority {
					t.Fatalf("unexpected priority: %#v", updates.Priority)
				}
				if updates.BusinessValue == nil || string(*updates.BusinessValue) != businessValue {
					t.Fatalf("unexpected business value: %#v", updates.BusinessValue)
				}
				if updates.Size == nil || *updates.Size != size {
					t.Fatalf("unexpected size: %#v", updates.Size)
				}
				if !updates.ClearSize {
					t.Fatalf("expected clear_size to be true")
				}
				epic := &models.Epic{}
				epic.Key = key
				epic.Title = *updates.Title
				epic.Priority = *updates.Priority
				epic.BusinessValue = updates.BusinessValue
				epic.Size = updates.Size
				return epic, nil
			},
		}

		body := `{"title":"Updated Epic","priority":"high","business_value":"medium","size":8,"clear_size":true}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/epics/E07", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.updateEpicCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateEpicCalls)
		}

		var epic models.Epic
		if err := json.NewDecoder(rec.Body).Decode(&epic); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if epic.Title != title {
			t.Fatalf("unexpected title: %s", epic.Title)
		}
	})

	t.Run("feature", func(t *testing.T) {
		title := "Updated Feature"
		executionOrder := 9
		mock := &mockMutationServicer{
			updateFeatureFn: func(_ context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
				if key != "E07-F01" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.Title == nil || *updates.Title != title {
					t.Fatalf("unexpected title: %#v", updates.Title)
				}
				if updates.ExecutionOrder == nil || *updates.ExecutionOrder != executionOrder {
					t.Fatalf("unexpected execution order: %#v", updates.ExecutionOrder)
				}
				feature := &models.Feature{}
				feature.Key = key
				feature.Title = *updates.Title
				feature.ExecutionOrder = updates.ExecutionOrder
				return feature, nil
			},
		}

		body := `{"title":"Updated Feature","execution_order":9}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/features/E07-F01", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.updateFeatureCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateFeatureCalls)
		}

		var feature models.Feature
		if err := json.NewDecoder(rec.Body).Decode(&feature); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if feature.Title != title {
			t.Fatalf("unexpected title: %s", feature.Title)
		}
	})

	t.Run("task", func(t *testing.T) {
		agentType := "backend"
		priority := 7
		executionOrder := 3
		mock := &mockMutationServicer{
			updateTaskFn: func(_ context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
				if key != "T-E07-F01-001" {
					t.Fatalf("unexpected key: %s", key)
				}
				if updates.AgentType == nil || *updates.AgentType != agentType {
					t.Fatalf("unexpected agent type: %#v", updates.AgentType)
				}
				if updates.Priority == nil || *updates.Priority != priority {
					t.Fatalf("unexpected priority: %#v", updates.Priority)
				}
				if updates.ExecutionOrder == nil || *updates.ExecutionOrder != executionOrder {
					t.Fatalf("unexpected execution order: %#v", updates.ExecutionOrder)
				}
				task := &models.Task{}
				task.Key = key
				task.AgentType = updates.AgentType
				task.Priority = *updates.Priority
				task.ExecutionOrder = updates.ExecutionOrder
				return task, nil
			},
		}

		body := `{"agent_type":"backend","priority":7,"execution_order":3}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/tasks/T-E07-F01-001", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.updateTaskCalls != 1 {
			t.Fatalf("expected 1 update call, got %d", mock.updateTaskCalls)
		}

		var task models.Task
		if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if task.Priority != priority {
			t.Fatalf("unexpected priority: %d", task.Priority)
		}
	})
}

func TestMutationHandler_TransitionPaths(t *testing.T) {
	t.Run("epic", func(t *testing.T) {
		mock := &mockMutationServicer{
			transitionEpicFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "E07" {
					t.Fatalf("unexpected key: %s", key)
				}
				if targetStatus != "active" {
					t.Fatalf("unexpected target status: %s", targetStatus)
				}
				if !opts.Force || opts.Reason != "start" || opts.Agent != "viewer" {
					t.Fatalf("unexpected opts: %#v", opts)
				}
				return &services.TransitionResult{EntityType: models.EntityTypeEpic, EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}

		body := `{"target_status":"active","force":true,"reason":"start","agent":"viewer"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/epics/E07/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.transitionEpicCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionEpicCalls)
		}

		var result services.TransitionResult
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.EntityKey != "E07" || result.ToStatus != "active" || !result.Transitioned {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("feature", func(t *testing.T) {
		mock := &mockMutationServicer{
			transitionFeatureFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "E07-F01" {
					t.Fatalf("unexpected key: %s", key)
				}
				if targetStatus != "completed" {
					t.Fatalf("unexpected target status: %s", targetStatus)
				}
				if opts.Reason != "done" {
					t.Fatalf("unexpected opts: %#v", opts)
				}
				return &services.TransitionResult{EntityType: models.EntityTypeFeature, EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}

		body := `{"target_status":"completed","reason":"done"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/features/E07-F01/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.transitionFeatureCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionFeatureCalls)
		}
	})

	t.Run("task", func(t *testing.T) {
		mock := &mockMutationServicer{
			transitionTaskFn: func(_ context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
				if key != "T-E07-F01-001" {
					t.Fatalf("unexpected key: %s", key)
				}
				if targetStatus != "in_progress" {
					t.Fatalf("unexpected target status: %s", targetStatus)
				}
				return &services.TransitionResult{EntityType: models.EntityTypeTask, EntityKey: key, ToStatus: targetStatus, Transitioned: true}, nil
			},
		}

		body := `{"target_status":"in_progress"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/tasks/T-E07-F01-001/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.transitionTaskCalls != 1 {
			t.Fatalf("expected 1 transition call, got %d", mock.transitionTaskCalls)
		}
	})
}

func TestMutationHandler_NoteAndRelationshipPaths(t *testing.T) {
	t.Run("add note per entity", func(t *testing.T) {
		cases := []struct {
			name   string
			method string
			path   string
			key    string
		}{
			{"epic", http.MethodPost, "/api/v1/viewer/epics/E07/notes", "E07"},
			{"feature", http.MethodPost, "/api/v1/viewer/features/E07-F01/notes", "E07-F01"},
			{"task", http.MethodPost, "/api/v1/viewer/tasks/T-E07-F01-001/notes", "T-E07-F01-001"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				mock := &mockMutationServicer{
					addNoteFn: func(_ context.Context, entityKey, noteType, content, createdBy string) (*models.EntityNote, error) {
						if entityKey != tc.key {
							t.Fatalf("unexpected key: %s", entityKey)
						}
						if noteType != "decision" || content != "Use normalized writes" || createdBy != "viewer" {
							t.Fatalf("unexpected note payload: %s %s %s", noteType, content, createdBy)
						}
						return &models.EntityNote{EntityType: models.EntityTypeEpic, EntityID: 1, NoteType: models.NoteType(noteType), Content: content}, nil
					},
				}
				req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(`{"note_type":"decision","content":"Use normalized writes","created_by":"viewer"}`))
				rec := httptest.NewRecorder()
				newMutationHandlerMux(mock).ServeHTTP(rec, req)

				if rec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
				}
				if mock.addNoteCalls != 1 {
					t.Fatalf("expected 1 note call, got %d", mock.addNoteCalls)
				}

				var note models.EntityNote
				if err := json.NewDecoder(rec.Body).Decode(&note); err != nil {
					t.Fatalf("failed to decode note response: %v", err)
				}
				if note.NoteType != models.NoteType("decision") {
					t.Fatalf("unexpected note type: %s", note.NoteType)
				}
			})
		}
	})

	t.Run("create relationship", func(t *testing.T) {
		mock := &mockMutationServicer{
			createRelFn: func(_ context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error) {
				if fromKey != "E07-F01" || toKey != "T-E07-F01-001" || relType != "related_to" {
					t.Fatalf("unexpected relationship payload: %s %s %s", fromKey, toKey, relType)
				}
				return &models.EntityRelationship{FromEntityType: models.EntityTypeFeature, FromEntityID: 11, ToEntityType: models.EntityTypeTask, ToEntityID: 12, RelationshipType: models.EntityRelRelatedTo}, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/features/E07-F01/relationships", bytes.NewBufferString(`{"relationship_type":"related_to","to_key":"T-E07-F01-001"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.createRelCalls != 1 {
			t.Fatalf("expected 1 create call, got %d", mock.createRelCalls)
		}
	})

	t.Run("delete relationship", func(t *testing.T) {
		mock := &mockMutationServicer{
			deleteRelFn: func(_ context.Context, fromKey, relType, toKey string) error {
				if fromKey != "T-E07-F01-001" || relType != "depends_on" || toKey != "E07-F01" {
					t.Fatalf("unexpected delete payload: %s %s %s", fromKey, relType, toKey)
				}
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/viewer/tasks/T-E07-F01-001/relationships/depends_on/E07-F01", nil)
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if mock.deleteRelCalls != 1 {
			t.Fatalf("expected 1 delete call, got %d", mock.deleteRelCalls)
		}
	})
}

func TestMutationHandler_RejectsBadInput(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/epics/E07", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.updateEpicCalls != 0 {
			t.Fatalf("service should not be called on invalid json")
		}
	})

	t.Run("unknown fields", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/features/E07-F01", bytes.NewBufferString(`{"status":"active"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.updateFeatureCalls != 0 {
			t.Fatalf("service should not be called on unknown fields")
		}
	})

	t.Run("missing target_status", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/tasks/T-E07-F01-001/transition", bytes.NewBufferString(`{"reason":"missing target"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.transitionTaskCalls != 0 {
			t.Fatalf("service should not be called when target_status is missing")
		}
	})

	t.Run("wrong key type", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/tasks/E07-F01", bytes.NewBufferString(`{"title":"bad key"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.updateTaskCalls != 0 {
			t.Fatalf("service should not be called on wrong key type")
		}
	})

	t.Run("empty note content", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/epics/E07/notes", bytes.NewBufferString(`{"note_type":"comment","content":"   "}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.addNoteCalls != 0 {
			t.Fatalf("service should not be called on empty note content")
		}
	})

	t.Run("unsupported relationship type", func(t *testing.T) {
		mock := &mockMutationServicer{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/tasks/T-E07-F01-001/relationships", bytes.NewBufferString(`{"relationship_type":"bogus","to_key":"E07-F01"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if mock.createRelCalls != 0 {
			t.Fatalf("service should not be called on unsupported relationship type")
		}
	})

	t.Run("duplicate relationship maps to conflict", func(t *testing.T) {
		mock := &mockMutationServicer{
			createRelFn: func(_ context.Context, _, _, _ string) (*models.EntityRelationship, error) {
				return nil, fmt.Errorf("duplicate relationship: already exists")
			},
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/tasks/T-E07-F01-001/relationships", bytes.NewBufferString(`{"relationship_type":"depends_on","to_key":"E07-F01"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("missing note target entity maps to not found", func(t *testing.T) {
		mock := &mockMutationServicer{
			addNoteFn: func(_ context.Context, _ string, _, _, _ string) (*models.EntityNote, error) {
				return nil, fmt.Errorf("epic not found: E99")
			},
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/epics/E99/notes", bytes.NewBufferString(`{"note_type":"comment","content":"hello"}`))
		rec := httptest.NewRecorder()
		newMutationHandlerMux(mock).ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestMutationHandler_ServiceErrorsAreReturned(t *testing.T) {
	mock := &mockMutationServicer{
		updateEpicFn: func(_ context.Context, _ string, _ services.EpicUpdates) (*models.Epic, error) {
			return nil, fmt.Errorf("epic not found: E07")
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/epics/E07", bytes.NewBufferString(`{"title":"Updated"}`))
	rec := httptest.NewRecorder()
	newMutationHandlerMux(mock).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var errResp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp["message"] != "epic not found: E07" {
		t.Fatalf("expected service error to be returned, got %#v", errResp["message"])
	}
}

// TC-302: the existing generic relationship transport must accept the
// registered Question key as a source and leave direction enforcement to the
// normalized relationship service shared with the CLI.
func TestMutationHandler_QuestionRelationshipTransport_TC302(t *testing.T) {
	mock := &mockMutationServicer{
		createRelFn: func(_ context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error) {
			if fromKey != "Q001" || toKey != "E39-F03" || relType != "question_blocks" {
				t.Fatalf("TC-302 transport args = %q %q %q", fromKey, toKey, relType)
			}
			return &models.EntityRelationship{RelationshipType: models.EntityRelQuestionBlocks}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/questions/Q001/relationships", bytes.NewBufferString(`{"relationship_type":"question_blocks","to_key":"E39-F03"}`))
	rec := httptest.NewRecorder()
	newMutationHandlerMux(mock).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("TC-302 Question relationship status = %d body=%s", rec.Code, rec.Body.String())
	}
	if mock.createRelCalls != 1 {
		t.Fatalf("TC-302 relationship service calls = %d, want 1", mock.createRelCalls)
	}
}

// TC-302: every eligible Question gate target must survive the real viewer
// relationship transport validation. This is deliberately table driven so a
// future source-only route change cannot silently exclude a supported target
// key family before the shared relationship service checks direction.
func TestMutationHandler_QuestionRelationshipTransport_AllEligibleTargets_TC302(t *testing.T) {
	targets := []string{
		"E39",
		"E39-F03",
		"T-E39-F03-003",
		"B039",
		"CC-039",
		"TD-039",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			mock := &mockMutationServicer{
				createRelFn: func(_ context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error) {
					if fromKey != "Q001" || toKey != target || relType != "question_blocks" {
						t.Fatalf("TC-302 transport args = %q %q %q", fromKey, toKey, relType)
					}
					return &models.EntityRelationship{RelationshipType: models.EntityRelQuestionBlocks}, nil
				},
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/questions/Q001/relationships", bytes.NewBufferString(`{"relationship_type":"question_blocks","to_key":"`+target+`"}`))
			rec := httptest.NewRecorder()
			newMutationHandlerMux(mock).ServeHTTP(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("TC-302 Question -> %s status = %d body=%s", target, rec.Code, rec.Body.String())
			}
			if mock.createRelCalls != 1 {
				t.Fatalf("TC-302 Question -> %s relationship calls = %d, want 1", target, mock.createRelCalls)
			}
		})
	}
}

func TestMutationHandler_InternalErrorsAreGeneric(t *testing.T) {
	rawErr := "sqlite: UNIQUE constraint failed: secret_table.internal_column"
	mock := &mockMutationServicer{
		updateEpicFn: func(_ context.Context, _ string, _ services.EpicUpdates) (*models.Epic, error) {
			return nil, fmt.Errorf("%s", rawErr)
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/epics/E07", bytes.NewBufferString(`{"title":"Updated"}`))
	rec := httptest.NewRecorder()
	newMutationHandlerMux(mock).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var errResp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp["message"] == rawErr {
		t.Fatalf("expected generic 500 message, got raw backend error")
	}
}

func TestMutationHandler_RequestBodyTooLarge(t *testing.T) {
	called := false
	mock := &mockMutationServicer{
		updateEpicFn: func(_ context.Context, _ string, _ services.EpicUpdates) (*models.Epic, error) {
			called = true
			epic := &models.Epic{}
			epic.Key = "E07"
			epic.Title = "Updated"
			return epic, nil
		},
	}
	oversizedTitle := strings.Repeat("x", 2*1024*1024+1)
	body := `{"title":"` + oversizedTitle + `"}`

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/viewer/epics/E07", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	newMutationHandlerMux(mock).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	if called {
		t.Fatalf("service must not be called when request body is too large")
	}
}
