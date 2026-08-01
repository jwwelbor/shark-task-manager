package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type mockQuestionService struct {
	created       services.CreateQuestionInput
	createCalls   int
	question      *models.Question
	listed        bool
	updated       bool
	transitioned  bool
	nextStatusErr error
	transitionErr error
	configured    bool
	responded     bool
	resolved      bool
	withdrawn     bool
	superseded    bool
	openCalls     int
	blockingCalls int
	fullCalls     int
	openInput     struct {
		responder     string
		limit, offset int
	}
	blockingInput struct {
		targetType    models.EntityType
		targetKey     string
		limit, offset int
	}
	fullInput struct{ key, actor string }
	open      []models.QuestionProjection
	blocks    []*services.QuestionBlock
	full      *models.QuestionFullProjection
	fullErr   error
}

func (m *mockQuestionService) CreateQuestion(_ context.Context, in services.CreateQuestionInput) (*models.Question, error) {
	m.createCalls++
	m.created = in
	return m.question, nil
}
func (m *mockQuestionService) GetQuestion(context.Context, string) (*models.Question, error) {
	return m.question, nil
}
func (m *mockQuestionService) ListQuestions(context.Context, services.QuestionListFilter) ([]*models.Question, error) {
	m.listed = true
	return []*models.Question{m.question}, nil
}
func (m *mockQuestionService) UpdateQuestion(context.Context, string, services.QuestionUpdates) (*models.Question, error) {
	m.updated = true
	return m.question, nil
}
func (m *mockQuestionService) DeleteQuestion(context.Context, string) error { return nil }
func (m *mockQuestionService) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	if m.nextStatusErr != nil {
		return nil, m.nextStatusErr
	}
	return &services.NextStatusInfo{EntityType: models.EntityTypeQuestion, EntityKey: key, CurrentStatus: "draft"}, nil
}
func (m *mockQuestionService) TransitionStatus(_ context.Context, key, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	if m.transitionErr != nil {
		return nil, m.transitionErr
	}
	m.transitioned = true
	return &services.TransitionResult{EntityType: models.EntityTypeQuestion, EntityKey: key, FromStatus: "draft", ToStatus: target, Transitioned: true}, nil
}
func (m *mockQuestionService) ConfigureWorkflow(context.Context, services.ConfigureWorkflowInput) (*models.Question, error) {
	m.configured = true
	return m.question, nil
}
func (m *mockQuestionService) RecordResponse(context.Context, services.RecordQuestionResponseInput) (*models.Question, error) {
	m.responded = true
	return m.question, nil
}
func (m *mockQuestionService) Resolve(context.Context, services.ResolveQuestionInput) (*models.Question, error) {
	m.resolved = true
	return m.question, nil
}
func (m *mockQuestionService) Withdraw(context.Context, services.WithdrawQuestionInput) (*models.Question, error) {
	m.withdrawn = true
	return m.question, nil
}
func (m *mockQuestionService) Supersede(context.Context, services.SupersedeQuestionInput) (*models.Question, error) {
	m.superseded = true
	return m.question, nil
}
func (m *mockQuestionService) ListOpenQuestionsByResponder(_ context.Context, responder string, limit, offset int) ([]models.QuestionProjection, error) {
	m.openCalls++
	m.openInput.responder, m.openInput.limit, m.openInput.offset = responder, limit, offset
	return m.open, nil
}
func (m *mockQuestionService) ListQuestionsBlocking(_ context.Context, targetType models.EntityType, targetKey string, limit, offset int) ([]*services.QuestionBlock, error) {
	m.blockingCalls++
	m.blockingInput.targetType, m.blockingInput.targetKey = targetType, targetKey
	m.blockingInput.limit, m.blockingInput.offset = limit, offset
	return m.blocks, nil
}
func (m *mockQuestionService) ReadQuestionFull(_ context.Context, key, actor string) (*models.QuestionFullProjection, error) {
	m.fullCalls++
	m.fullInput.key, m.fullInput.actor = key, actor
	return m.full, m.fullErr
}
func questionMux(s QuestionServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewQuestionHandler(s).RegisterRoutes(mux)
	return mux
}

func TestQuestionHandlerCreateReturnsCanonicalLocation(t *testing.T) {
	svc := &mockQuestionService{question: &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Release"}, Status: "draft", Summary: "Summary", Requester: "owner"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"title":"Release","summary":"Summary","requester":"owner"}`))
	rec := httptest.NewRecorder()
	questionMux(svc).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/api/v1/questions/Q001" {
		t.Fatalf("Location=%q", got)
	}
	if svc.created.Requester != "owner" {
		t.Fatalf("input=%#v", svc.created)
	}
}

// TC-401 and TC-403: focused routes are finite contracts and call the shared
// service seam only after transport validation. Static routes must win over
// the generic {key} route so a focused read cannot be mistaken for Q key.
func TestQuestionHandlerFocusedReadRoutes_TC401_TC403(t *testing.T) {
	t.Run("open by responder uses compact page and defaults", func(t *testing.T) {
		svc := &mockQuestionService{open: []models.QuestionProjection{{Key: "Q001", Summary: "safe"}}}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions/open-by-responder?responder=alice", nil))
		if rec.Code != http.StatusOK || svc.openCalls != 1 {
			t.Fatalf("status=%d calls=%d body=%s", rec.Code, svc.openCalls, rec.Body.String())
		}
		if svc.openInput.responder != "alice" || svc.openInput.limit != 50 || svc.openInput.offset != 0 {
			t.Fatalf("input=%+v", svc.openInput)
		}
		if strings.Contains(rec.Body.String(), "context_data") {
			t.Fatalf("focused compact response leaked durable context: %s", rec.Body.String())
		}
	})
	t.Run("blocking for normalizes target and passes finite page", func(t *testing.T) {
		svc := &mockQuestionService{blocks: []*services.QuestionBlock{{QuestionKey: "Q001", Summary: "safe", ResolutionOwner: "owner", CurrentResponder: "alice"}}}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions/blocking-for?entity_key=e39-f04&limit=1&offset=0", nil))
		if rec.Code != http.StatusOK || svc.blockingCalls != 1 {
			t.Fatalf("status=%d calls=%d body=%s", rec.Code, svc.blockingCalls, rec.Body.String())
		}
		if svc.blockingInput.targetType != models.EntityTypeFeature || svc.blockingInput.targetKey != "E39-F04" || svc.blockingInput.limit != 1 {
			t.Fatalf("input=%+v", svc.blockingInput)
		}
		if strings.Contains(rec.Body.String(), "relationship_id") {
			t.Fatalf("blocking response leaked relationship ID: %s", rec.Body.String())
		}
	})
}

// TC-405 and TC-406: the explicit full route has precedence over the generic
// GET by key route and rejects malformed query shape before the policy seam.
func TestQuestionHandlerFullReadRoute_TC405_TC406(t *testing.T) {
	svc := &mockQuestionService{full: &models.QuestionFullProjection{QuestionProjection: models.QuestionProjection{Key: "Q001", Summary: "safe"}}}
	rec := httptest.NewRecorder()
	questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions/Q001/full?actor=alice", nil))
	if rec.Code != http.StatusOK || svc.fullCalls != 1 || svc.fullInput.key != "Q001" || svc.fullInput.actor != "alice" {
		t.Fatalf("status=%d calls=%d input=%+v body=%s", rec.Code, svc.fullCalls, svc.fullInput, rec.Body.String())
	}

	for _, path := range []string{
		"/api/v1/questions/Q001/full",
		"/api/v1/questions/Q001/full?actor=%20",
		"/api/v1/questions/Q001/full?actor=%FF",
		"/api/v1/questions/Q001/full?actor=" + strings.Repeat("a", 257),
		"/api/v1/questions/Q001/full?actor=bearer%20token",
		"/api/v1/questions/Q001/full?actor=alice&unexpected=value",
		"/api/v1/questions/open-by-responder?responder=alice&unexpected=value",
		"/api/v1/questions/open-by-responder?responder=%FF",
		"/api/v1/questions/open-by-responder?responder=" + strings.Repeat("a", 257),
		"/api/v1/questions/open-by-responder?responder=bearer%20token",
		"/api/v1/questions/blocking-for?entity_key=Q001",
		"/api/v1/questions/blocking-for?entity_key=F04&limit=0",
	} {
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	if svc.fullCalls != 1 || svc.openCalls != 0 || svc.blockingCalls != 0 {
		t.Fatalf("invalid focused requests reached service: full=%d open=%d blocking=%d", svc.fullCalls, svc.openCalls, svc.blockingCalls)
	}

	denied := &mockQuestionService{fullErr: &services.QuestionFullReadDeniedError{Key: "Q001"}}
	rec = httptest.NewRecorder()
	questionMux(denied).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions/Q001/full?actor=mallory", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("denied full read status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TC-002: a POST body is exactly one JSON document; otherwise the handler
// must reject it before the creation service can allocate or write a Question.
func TestQuestionHandlerCreateRejectsTrailingJSONBeforeService_TC002(t *testing.T) {
	svc := &mockQuestionService{question: &models.Question{BaseEntity: models.BaseEntity{Key: "Q001"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"title":"Release","summary":"Summary","requester":"owner"} {"extra":true}`))
	rec := httptest.NewRecorder()
	questionMux(svc).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if svc.createCalls != 0 {
		t.Fatalf("CreateQuestion invoked with %#v", svc.created)
	}
}

// Question mutation bodies are finite contracts. Every mutating decoder must
// reject undeclared members before calling its service, otherwise an API client
// can believe a field was persisted while the service silently ignores it.
func TestQuestionHandlerMutationsRejectUnknownMembersBeforeService(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		svc := &mockQuestionService{question: &models.Question{}}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"title":"Release","summary":"Summary","requester":"owner","priority":1}`)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if svc.createCalls != 0 {
			t.Fatal("CreateQuestion called for an undeclared request member")
		}
	})

	t.Run("transition", func(t *testing.T) {
		svc := &mockQuestionService{}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/questions/Q001/transition", strings.NewReader(`{"target_status":"archived","priority":1}`)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if svc.transitioned {
			t.Fatal("TransitionStatus called for an undeclared request member")
		}
	})
}

// TC-008: registered Question status routes call the same finite service
// contract as the generic CLI surface, including missing and invalid cases.
func TestQuestionHandlerStatusRoutes_TC008(t *testing.T) {
	t.Run("draft can transition to archived", func(t *testing.T) {
		svc := &mockQuestionService{}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/questions/Q001/transition", bytes.NewBufferString(`{"target_status":"archived"}`)))
		if rec.Code != http.StatusOK || !svc.transitioned {
			t.Fatalf("status=%d transitioned=%v body=%s", rec.Code, svc.transitioned, rec.Body.String())
		}
	})
	t.Run("invalid transition does not invoke service", func(t *testing.T) {
		svc := &mockQuestionService{}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/questions/Q001/transition", bytes.NewBufferString(`{}`)))
		if rec.Code != http.StatusBadRequest || svc.transitioned {
			t.Fatalf("status=%d transitioned=%v", rec.Code, svc.transitioned)
		}
	})
	t.Run("trailing transition JSON does not invoke service", func(t *testing.T) {
		svc := &mockQuestionService{}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/questions/Q001/transition", bytes.NewBufferString(`{"target_status":"archived"} {"extra":true}`)))
		if rec.Code != http.StatusBadRequest || svc.transitioned {
			t.Fatalf("status=%d transitioned=%v", rec.Code, svc.transitioned)
		}
	})
	t.Run("missing Question maps to not found", func(t *testing.T) {
		svc := &mockQuestionService{nextStatusErr: fmt.Errorf("question not found")}
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions/Q999/next-status", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
func TestQuestionHandlerRejectsMalformedAndUnsupportedStatusUpdate(t *testing.T) {
	svc := &mockQuestionService{question: &models.Question{}}
	for _, tc := range []struct{ method, path, body string }{{http.MethodPost, "/api/v1/questions", "{"}, {http.MethodPatch, "/api/v1/questions/Q001", `{"status":"archived"}`}} {
		rec := httptest.NewRecorder()
		questionMux(svc).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", tc.method, rec.Code)
		}
	}
}

// PATCH is a finite transport contract. It must reject both fields reserved
// for workflow ownership and fields it does not declare before the service is
// invoked, so a malformed client request cannot cause a write.
func TestQuestionHandlerUpdateRejectsForbiddenAndUnknownMembersBeforeService(t *testing.T) {
	for name, body := range map[string]string{
		"forbidden status":       `{"status":"archived"}`,
		"forbidden empty status": `{"status":""}`,
		"unknown priority":       `{"priority":1}`,
		"unknown nested":         `{"metadata":{"owner":"release"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			svc := &mockQuestionService{question: &models.Question{}}
			rec := httptest.NewRecorder()
			questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/api/v1/questions/Q001", strings.NewReader(body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if svc.updated {
				t.Fatal("UpdateQuestion called for forbidden or unknown request member")
			}
		})
	}
}

// The Question list is intentionally a finite query contract. A silently
// ignored parameter would make a caller believe a filtered page was returned.
func TestQuestionHandlerListRejectsUndeclaredQueryParameters(t *testing.T) {
	for _, name := range []string{
		"sort", "sort_by", "order", "filter", "title", "agent", "epic",
		"feature", "show_all", "all", "blocked", "tag", "unexpected",
	} {
		t.Run(name, func(t *testing.T) {
			svc := &mockQuestionService{question: &models.Question{}}
			query := url.Values{name: {"value"}}
			rec := httptest.NewRecorder()
			questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/questions?"+query.Encode(), nil))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Message != `unsupported Question list query parameter "`+name+`"` {
				t.Fatalf("message=%q", body.Message)
			}
			if svc.listed {
				t.Fatal("ListQuestions called for an undeclared query parameter")
			}
		})
	}
}

// Question metadata transports must not serialize the stored generic context.
// The service returns the persistence model, so this covers every Question
// route that returns a Question rather than a hand-built response body.
func TestQuestionHandlerQuestionResponsesRedactContextData(t *testing.T) {
	sentinel := `{"current_step":"question-context-must-not-leak"}`
	question := &models.Question{
		BaseEntity: models.BaseEntity{Key: "Q001", Title: "Release", ContextData: &sentinel},
		Status:     models.QuestionStatusDraft,
		Summary:    "Summary",
		Requester:  "owner",
	}
	svc := &mockQuestionService{question: question}
	for _, tc := range []struct {
		name, method, path, body string
	}{
		{"create", http.MethodPost, "/api/v1/questions", `{"title":"Release","summary":"Summary","requester":"owner"}`},
		{"get", http.MethodGet, "/api/v1/questions/Q001", ""},
		{"list", http.MethodGet, "/api/v1/questions", ""},
		{"update", http.MethodPatch, "/api/v1/questions/Q001", `{"title":"Release"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			questionMux(svc).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if rec.Code < http.StatusOK || rec.Code >= http.StatusMultipleChoices {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "question-context-must-not-leak") {
				t.Fatalf("response exposed Question context: %s", rec.Body.String())
			}
			var payload any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
		})
	}
}

// TC-109: F02 workflow transports are finite POST routes. A valid operation
// returns the ordinary metadata projection, never the stored state payload.
func TestQuestionHandlerWorkflowRoutesReturnMetadataOnly_TC109(t *testing.T) {
	sentinel := `{"question_state":{"responses":[{"summary":"secret"}]}}`
	for _, tc := range []struct {
		name, path, body string
		called           func(*mockQuestionService) bool
	}{
		{"workflow", "/api/v1/questions/Q001/workflow", `{"resolution_owner":"owner","responders":["alice"]}`, func(s *mockQuestionService) bool { return s.configured }},
		{"response", "/api/v1/questions/Q001/response", `{"session":"session-a","responder":"alice","summary":"approved","evidence_pointer":"docs/spec.md"}`, func(s *mockQuestionService) bool { return s.responded }},
		{"resolve", "/api/v1/questions/Q001/resolve", `{"owner":"owner","resolution_kind":"no_lasting_consequence"}`, func(s *mockQuestionService) bool { return s.resolved }},
		{"withdraw", "/api/v1/questions/Q001/withdraw", `{"owner":"owner","reason":"obsolete"}`, func(s *mockQuestionService) bool { return s.withdrawn }},
		{"supersede", "/api/v1/questions/Q001/supersede", `{"owner":"owner","reason":"obsolete","superseded_by":"Q002"}`, func(s *mockQuestionService) bool { return s.superseded }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockQuestionService{question: &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Question", ContextData: &sentinel}}}
			rec := httptest.NewRecorder()
			questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
			if rec.Code != http.StatusOK || !tc.called(svc) {
				t.Fatalf("status=%d called=%v body=%s", rec.Code, tc.called(svc), rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Fatalf("response exposed Question context: %s", rec.Body.String())
			}
		})
	}
}

// TC-109: strict decoding and required fields are transport responsibilities;
// a rejected request must not reach any Question workflow mutation method.
func TestQuestionHandlerWorkflowRoutesRejectInvalidBodiesBeforeService_TC109(t *testing.T) {
	for _, tc := range []struct{ name, path, body string }{
		{"workflow unknown", "/api/v1/questions/Q001/workflow", `{"resolution_owner":"owner","responders":["alice"],"extra":true}`},
		{"response malformed", "/api/v1/questions/Q001/response", `{`},
		{"resolve missing pointer", "/api/v1/questions/Q001/resolve", `{"owner":"owner","resolution_kind":"feature_change"}`},
		{"withdraw missing reason", "/api/v1/questions/Q001/withdraw", `{"owner":"owner"}`},
		{"supersede missing target", "/api/v1/questions/Q001/supersede", `{"owner":"owner","reason":"obsolete"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockQuestionService{question: &models.Question{}}
			rec := httptest.NewRecorder()
			questionMux(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if svc.configured || svc.responded || svc.resolved || svc.withdrawn || svc.superseded {
				t.Fatal("workflow service invoked for invalid request")
			}
		})
	}
}
