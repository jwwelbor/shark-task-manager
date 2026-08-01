package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// questionListQueryKeys is the complete public query contract for the
// Question collection. Keep this deliberately finite: accepting a parameter
// that the service cannot honor makes a request look filtered when it is not.
var questionListQueryKeys = map[string]struct{}{
	"status": {}, "requester": {}, "blocking": {}, "limit": {}, "offset": {},
}

// QuestionServicer is the finite service seam used by QuestionHandler.
type QuestionServicer interface {
	CreateQuestion(context.Context, services.CreateQuestionInput) (*models.Question, error)
	GetQuestion(context.Context, string) (*models.Question, error)
	ListQuestions(context.Context, services.QuestionListFilter) ([]*models.Question, error)
	UpdateQuestion(context.Context, string, services.QuestionUpdates) (*models.Question, error)
	DeleteQuestion(context.Context, string) error
	TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(context.Context, string) (*services.NextStatusInfo, error)
	ConfigureWorkflow(context.Context, services.ConfigureWorkflowInput) (*models.Question, error)
	RecordResponse(context.Context, services.RecordQuestionResponseInput) (*models.Question, error)
	Resolve(context.Context, services.ResolveQuestionInput) (*models.Question, error)
	Withdraw(context.Context, services.WithdrawQuestionInput) (*models.Question, error)
	Supersede(context.Context, services.SupersedeQuestionInput) (*models.Question, error)
	ListOpenQuestionsByResponder(context.Context, string, int, int) ([]models.QuestionProjection, error)
	ListQuestionsBlocking(context.Context, models.EntityType, string, int, int) ([]*services.QuestionBlock, error)
	ReadQuestionFull(context.Context, string, string) (*models.QuestionFullProjection, error)
}
type QuestionHandler struct{ svc QuestionServicer }

func NewQuestionHandler(svc QuestionServicer) *QuestionHandler {
	if svc == nil {
		panic("QuestionHandler: svc is required")
	}
	return &QuestionHandler{svc: svc}
}
func (h *QuestionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/questions", h.ListQuestions)
	mux.HandleFunc("POST /api/v1/questions", h.CreateQuestion)
	mux.HandleFunc("GET /api/v1/questions/open-by-responder", h.ListOpenQuestionsByResponder)
	mux.HandleFunc("GET /api/v1/questions/blocking-for", h.ListQuestionsBlocking)
	mux.HandleFunc("GET /api/v1/questions/{key}/full", h.GetQuestionFull)
	mux.HandleFunc("GET /api/v1/questions/{key}", h.GetQuestion)
	mux.HandleFunc("PATCH /api/v1/questions/{key}", h.UpdateQuestion)
	mux.HandleFunc("DELETE /api/v1/questions/{key}", h.DeleteQuestion)
	mux.HandleFunc("GET /api/v1/questions/{key}/next-status", h.GetNextStatus)
	mux.HandleFunc("POST /api/v1/questions/{key}/transition", h.TransitionStatus)
	mux.HandleFunc("POST /api/v1/questions/{key}/workflow", h.ConfigureWorkflow)
	mux.HandleFunc("POST /api/v1/questions/{key}/response", h.RecordResponse)
	mux.HandleFunc("POST /api/v1/questions/{key}/resolve", h.Resolve)
	mux.HandleFunc("POST /api/v1/questions/{key}/withdraw", h.Withdraw)
	mux.HandleFunc("POST /api/v1/questions/{key}/supersede", h.Supersede)
}

var (
	openQuestionQueryKeys     = map[string]struct{}{"responder": {}, "limit": {}, "offset": {}}
	blockingQuestionQueryKeys = map[string]struct{}{"entity_key": {}, "limit": {}, "offset": {}}
	fullQuestionQueryKeys     = map[string]struct{}{"actor": {}}
)

// ListOpenQuestionsByResponder exposes the bounded compact focused read.
func (h *QuestionHandler) ListOpenQuestionsByResponder(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if err := validateQuestionReadQuery(query, openQuestionQueryKeys); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	responder, err := requiredQuestionReadIdentity(query, "responder")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset, err := questionReadPage(query)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	questions, err := h.svc.ListOpenQuestionsByResponder(r.Context(), responder, limit, offset)
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, questions)
}

// ListQuestionsBlocking exposes the direct I-03 compact handoff. Target
// existence and F03 qualification remain service responsibilities.
func (h *QuestionHandler) ListQuestionsBlocking(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if err := validateQuestionReadQuery(query, blockingQuestionQueryKeys); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	rawKey, err := requiredQuestionReadText(query, "entity_key")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	targetType, targetKey, err := focusedQuestionTarget(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset, err := questionReadPage(query)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	blocks, err := h.svc.ListQuestionsBlocking(r.Context(), targetType, targetKey, limit, offset)
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, blocks)
}

// GetQuestionFull is separate from GetQuestion so generic reads remain compact.
func (h *QuestionHandler) GetQuestionFull(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if err := validateQuestionReadQuery(query, fullQuestionQueryKeys); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	actor, err := requiredQuestionReadIdentity(query, "actor")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	full, err := h.svc.ReadQuestionFull(r.Context(), pathParam(r, "key"), actor)
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, full)
}

func validateQuestionReadQuery(query url.Values, allowed map[string]struct{}) error {
	for name, values := range query {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("unsupported Question query parameter %q", name)
		}
		if len(values) != 1 {
			return fmt.Errorf("Question query parameter %q must appear exactly once", name)
		}
	}
	return nil
}

func requiredQuestionReadText(query url.Values, field string) (string, error) {
	values, ok := query[field]
	if !ok || len(values) != 1 || values[0] == "" || strings.TrimSpace(values[0]) != values[0] {
		return "", fmt.Errorf("%s is required and must be trimmed", field)
	}
	return values[0], nil
}

func requiredQuestionReadIdentity(query url.Values, field string) (string, error) {
	identity, err := requiredQuestionReadText(query, field)
	if err != nil {
		return "", err
	}
	if err := services.ValidateQuestionReadIdentity(field, identity); err != nil {
		return "", err
	}
	return identity, nil
}

func questionReadPage(query url.Values) (int, int, error) {
	limit, offset := 50, 0
	for _, input := range []struct {
		name        string
		destination *int
	}{{"limit", &limit}, {"offset", &offset}} {
		if _, present := query[input.name]; !present {
			continue
		}
		value := query.Get(input.name)
		parsed, err := strconv.Atoi(value)
		if value == "" || err != nil {
			return 0, 0, fmt.Errorf("%s must be an integer", input.name)
		}
		*input.destination = parsed
	}
	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must be zero or greater")
	}
	return limit, offset, nil
}

func focusedQuestionTarget(rawKey string) (models.EntityType, string, error) {
	upper := strings.ToUpper(rawKey)
	keyService := keys.NewKeyService()
	switch {
	case keys.IsEpicKey(upper):
		return models.EntityTypeEpic, keyService.Normalize(upper), nil
	case keys.IsFeatureKey(upper):
		return models.EntityTypeFeature, keyService.Normalize(upper), nil
	case keys.IsShortTaskKey(upper), keys.IsTaskKey(upper):
		return models.EntityTypeTask, keyService.NormalizeTaskKey(upper), nil
	case keys.IsBugKey(upper):
		return models.EntityTypeBug, keyService.Normalize(upper), nil
	case keys.IsChangeKey(upper):
		return models.EntityTypeChange, keyService.Normalize(upper), nil
	case keys.IsTechDebtKey(upper):
		return models.EntityTypeTechDebt, upper, nil
	case keyService.DetectEntityType(upper) == keys.EntityTypeQuestion:
		return "", "", fmt.Errorf("entity_key must not identify a Question")
	default:
		return "", "", fmt.Errorf("entity_key is invalid")
	}
}

type questionRequest struct {
	Title       *string `json:"title,omitempty"`
	Summary     *string `json:"summary,omitempty"`
	Requester   *string `json:"requester,omitempty"`
	Description *string `json:"description,omitempty"`
	Blocking    *bool   `json:"blocking,omitempty"`
	Status      string  `json:"status,omitempty"`
}

// questionUpdateRequest deliberately has a separate transport shape from
// creation. Status remains present only so PATCH can report it as an explicit
// forbidden field; all other undeclared members are rejected by the strict
// decoder below before the service can read or write a Question.
type questionUpdateRequest struct {
	Title       *string `json:"title,omitempty"`
	Summary     *string `json:"summary,omitempty"`
	Requester   *string `json:"requester,omitempty"`
	Description *string `json:"description,omitempty"`
	Blocking    *bool   `json:"blocking,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type configureQuestionWorkflowRequest struct {
	ResolutionOwner *string  `json:"resolution_owner"`
	Responders      []string `json:"responders"`
}
type questionResponseRequest struct {
	Session         *string `json:"session"`
	Responder       *string `json:"responder"`
	Summary         *string `json:"summary"`
	EvidencePointer *string `json:"evidence_pointer"`
}
type resolveQuestionRequest struct {
	Owner             *string `json:"owner"`
	ResolutionKind    *string `json:"resolution_kind"`
	ResolutionPointer *string `json:"resolution_pointer"`
}
type closeQuestionRequest struct {
	Owner        *string `json:"owner"`
	Reason       *string `json:"reason"`
	SupersededBy *string `json:"superseded_by,omitempty"`
}

func decodeQuestionOperation(w http.ResponseWriter, r *http.Request, request any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil || rejectTrailingJSON(decoder) != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func requiredQuestionRequestField(w http.ResponseWriter, field string, value *string) (string, bool) {
	if value == nil || *value == "" {
		respondError(w, http.StatusBadRequest, field+" is required")
		return "", false
	}
	return *value, true
}

func (h *QuestionHandler) ConfigureWorkflow(w http.ResponseWriter, r *http.Request) {
	var req configureQuestionWorkflowRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	owner, ok := requiredQuestionRequestField(w, "resolution_owner", req.ResolutionOwner)
	if !ok {
		return
	}
	if len(req.Responders) == 0 {
		respondError(w, http.StatusBadRequest, "responders are required")
		return
	}
	question, err := h.svc.ConfigureWorkflow(r.Context(), services.ConfigureWorkflowInput{Key: pathParam(r, "key"), ResolutionOwner: owner, Responders: req.Responders})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(question))
}

func (h *QuestionHandler) RecordResponse(w http.ResponseWriter, r *http.Request) {
	var req questionResponseRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	session, ok := requiredQuestionRequestField(w, "session", req.Session)
	if !ok {
		return
	}
	responder, ok := requiredQuestionRequestField(w, "responder", req.Responder)
	if !ok {
		return
	}
	summary, ok := requiredQuestionRequestField(w, "summary", req.Summary)
	if !ok {
		return
	}
	evidencePointer, ok := requiredQuestionRequestField(w, "evidence_pointer", req.EvidencePointer)
	if !ok {
		return
	}
	question, err := h.svc.RecordResponse(r.Context(), services.RecordQuestionResponseInput{Key: pathParam(r, "key"), SessionID: session, Responder: responder, Summary: summary, EvidencePointer: evidencePointer})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(question))
}

func (h *QuestionHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req resolveQuestionRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	owner, ok := requiredQuestionRequestField(w, "owner", req.Owner)
	if !ok {
		return
	}
	kind, ok := requiredQuestionRequestField(w, "resolution_kind", req.ResolutionKind)
	if !ok {
		return
	}
	pointer := ""
	if req.ResolutionPointer != nil {
		pointer = *req.ResolutionPointer
	}
	if kind != "no_lasting_consequence" && pointer == "" {
		respondError(w, http.StatusBadRequest, "resolution_pointer is required")
		return
	}
	question, err := h.svc.Resolve(r.Context(), services.ResolveQuestionInput{Key: pathParam(r, "key"), Owner: owner, Kind: kind, Pointer: pointer})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(question))
}

func (h *QuestionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var req closeQuestionRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	owner, ok := requiredQuestionRequestField(w, "owner", req.Owner)
	if !ok {
		return
	}
	reason, ok := requiredQuestionRequestField(w, "reason", req.Reason)
	if !ok {
		return
	}
	question, err := h.svc.Withdraw(r.Context(), services.WithdrawQuestionInput{Key: pathParam(r, "key"), Owner: owner, Reason: reason})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(question))
}

func (h *QuestionHandler) Supersede(w http.ResponseWriter, r *http.Request) {
	var req closeQuestionRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	owner, ok := requiredQuestionRequestField(w, "owner", req.Owner)
	if !ok {
		return
	}
	reason, ok := requiredQuestionRequestField(w, "reason", req.Reason)
	if !ok {
		return
	}
	supersededBy, ok := requiredQuestionRequestField(w, "superseded_by", req.SupersededBy)
	if !ok {
		return
	}
	question, err := h.svc.Supersede(r.Context(), services.SupersedeQuestionInput{Key: pathParam(r, "key"), Owner: owner, Reason: reason, SupersededBy: supersededBy})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(question))
}

func (h *QuestionHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var req questionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || req.Title == nil || req.Summary == nil || req.Requester == nil {
		respondError(w, http.StatusBadRequest, "title, summary, and requester are required")
		return
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		respondError(w, http.StatusBadRequest, "title, summary, and requester are required")
		return
	}
	q, err := h.svc.CreateQuestion(r.Context(), services.CreateQuestionInput{Title: *req.Title, Summary: *req.Summary, Requester: *req.Requester, Description: valueOrEmpty(req.Description), Blocking: boolOrFalse(req.Blocking), Status: req.Status})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	w.Header().Set("Location", "/api/v1/questions/"+q.Key)
	respondJSON(w, http.StatusCreated, models.ProjectQuestion(q))
}

// GetNextStatus returns the available transitions for a Question.
// GET /api/v1/questions/{key}/next-status
func (h *QuestionHandler) GetNextStatus(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.GetNextStatus(r.Context(), pathParam(r, "key"))
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, info)
}

// TransitionStatus applies the finite Question draft-to-archived transition.
// POST /api/v1/questions/{key}/transition
func (h *QuestionHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	var req TransitionStatusRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	if req.TargetStatus == "" {
		respondError(w, http.StatusBadRequest, "target_status is required")
		return
	}
	result, err := h.svc.TransitionStatus(r.Context(), pathParam(r, "key"), req.TargetStatus, services.TransitionOptions{
		Force: req.Force, Reason: req.Reason, Agent: req.Agent, SessionID: req.SessionID,
		FromStatus: req.FromStatus, Outcome: req.Outcome, ForceRepeat: req.ForceRepeat, GuardAdvance: req.GuardAdvance,
	})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
func (h *QuestionHandler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	q, err := h.svc.GetQuestion(r.Context(), pathParam(r, "key"))
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(q))
}
func (h *QuestionHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if err := validateQuestionListQuery(q); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	f := services.QuestionListFilter{Limit: 50}
	if v := q.Get("status"); v != "" {
		s := models.QuestionStatus(v)
		f.Status = &s
	}
	if v := q.Get("requester"); v != "" {
		f.Requester = &v
	}
	if v := q.Get("blocking"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "blocking must be boolean")
			return
		}
		f.Blocking = &b
	}
	for _, pair := range []struct {
		name string
		dest *int
	}{{"limit", &f.Limit}, {"offset", &f.Offset}} {
		if v := q.Get(pair.name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				respondError(w, http.StatusBadRequest, pair.name+" must be an integer")
				return
			}
			*pair.dest = n
		}
	}
	qs, err := h.svc.ListQuestions(r.Context(), f)
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestions(qs))
}

func validateQuestionListQuery(query url.Values) error {
	for name := range query {
		if _, ok := questionListQueryKeys[name]; !ok {
			return fmt.Errorf("unsupported Question list query parameter %q", name)
		}
	}
	return nil
}
func (h *QuestionHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	var req questionUpdateRequest
	if !decodeQuestionOperation(w, r, &req) {
		return
	}
	if req.Status != nil {
		respondError(w, http.StatusBadRequest, "status is not an updatable Question field")
		return
	}
	q, err := h.svc.UpdateQuestion(r.Context(), pathParam(r, "key"), services.QuestionUpdates{Title: req.Title, Summary: req.Summary, Requester: req.Requester, Description: req.Description, Blocking: req.Blocking})
	if err != nil {
		handleServiceError(w, err, "question")
		return
	}
	respondJSON(w, http.StatusOK, models.ProjectQuestion(q))
}

// rejectTrailingJSON makes PATCH a single-document contract. Decoding only
// the first value would otherwise accept a valid update followed by arbitrary
// JSON, weakening the boundary before service invocation.
func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}
func (h *QuestionHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteQuestion(r.Context(), pathParam(r, "key")); err != nil {
		handleServiceError(w, err, "question")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func boolOrFalse(v *bool) bool { return v != nil && *v }

var _ QuestionServicer = (*services.QuestionService)(nil)
