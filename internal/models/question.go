package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// QuestionStatus represents a Question workflow status. Valid status values are
// supplied by workflow configuration rather than hardcoded in the model.
type QuestionStatus string

const (
	// QuestionStatusDraft is the base workflow state for a newly created Question.
	QuestionStatusDraft              QuestionStatus = "draft"
	QuestionStatusOpen               QuestionStatus = "open"
	QuestionStatusAnswering          QuestionStatus = "answering"
	QuestionStatusReadyForResolution QuestionStatus = "ready_for_resolution"
	QuestionStatusResolved           QuestionStatus = "resolved"
	QuestionStatusWithdrawn          QuestionStatus = "withdrawn"
	QuestionStatusSuperseded         QuestionStatus = "superseded"
	QuestionStatusArchived           QuestionStatus = "archived"
)

const (
	questionIdentityMaxBytes = 256
	questionResponseMaxBytes = 1000
	questionPointerMaxBytes  = 2048
)

var questionCredentialLabelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password\s*=`),
	regexp.MustCompile(`(?i)authorization\s*:`),
	regexp.MustCompile(`(?i)bearer(?:\s|:)`),
}

// QuestionResponderStatus is the bounded lifecycle state of one configured
// responder. It is private to QuestionState rather than generic context.
type QuestionResponderStatus string

const (
	QuestionResponderPending   QuestionResponderStatus = "pending"
	QuestionResponderCompleted QuestionResponderStatus = "completed"
)

// QuestionResponder is one ordered workflow participant.
type QuestionResponder struct {
	Identity string                  `json:"identity"`
	Status   QuestionResponderStatus `json:"status"`
}

// QuestionResponse is the bounded durable record for a completed responder.
type QuestionResponse struct {
	SessionID       string    `json:"session_id"`
	Responder       string    `json:"responder"`
	Summary         string    `json:"summary"`
	EvidencePointer string    `json:"evidence_pointer"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// QuestionState is the I-02 value stored only inside questions.context_data.
// CurrentResponder is deliberately derived and is never serialized as input.
type QuestionState struct {
	ResolutionOwner   string              `json:"resolution_owner"`
	Responders        []QuestionResponder `json:"responders"`
	Responses         []QuestionResponse  `json:"responses,omitempty"`
	ResolutionKind    string              `json:"resolution_kind,omitempty"`
	ResolutionPointer string              `json:"resolution_pointer,omitempty"`
}

// CurrentResponder derives the one serial routing target from persisted state.
func (s QuestionState) CurrentResponder() string {
	for _, responder := range s.Responders {
		if responder.Status == QuestionResponderPending {
			return responder.Identity
		}
	}
	return ""
}

// Validate checks the bounded state before it can be serialized or used for
// routing. Actionable errors name the rejected field and classify the durable
// alternative without echoing potentially sensitive input.
func (s QuestionState) Validate() error {
	if err := ValidateQuestionBoundedText("resolution_owner", s.ResolutionOwner, 1, questionIdentityMaxBytes); err != nil {
		return err
	}
	if len(s.Responders) < 1 || len(s.Responders) > 10 {
		return fmt.Errorf("responders must contain 1 through 10 ordered identities; use a typed note or authoritative record pointer for additional detail")
	}
	seenResponders := make(map[string]struct{}, len(s.Responders))
	responses := make(map[string]QuestionResponse, len(s.Responses))
	for _, response := range s.Responses {
		if err := ValidateQuestionBoundedText("responses.session_id", response.SessionID, 1, questionIdentityMaxBytes); err != nil {
			return err
		}
		if err := ValidateQuestionBoundedText("responses.responder", response.Responder, 1, questionIdentityMaxBytes); err != nil {
			return err
		}
		if _, exists := responses[response.Responder]; exists {
			return fmt.Errorf("responses contains duplicate responder; use a typed note or authoritative record pointer")
		}
		if err := ValidateQuestionBoundedText("responses.summary", response.Summary, 1, questionResponseMaxBytes); err != nil {
			return err
		}
		if err := ValidateQuestionBoundedText("responses.evidence_pointer", response.EvidencePointer, 1, questionPointerMaxBytes); err != nil {
			return err
		}
		if response.RecordedAt.IsZero() {
			return fmt.Errorf("responses.recorded_at is required; use a typed note or authoritative record pointer")
		}
		responses[response.Responder] = response
	}
	for _, responder := range s.Responders {
		if err := ValidateQuestionBoundedText("responders.identity", responder.Identity, 1, questionIdentityMaxBytes); err != nil {
			return err
		}
		if _, exists := seenResponders[responder.Identity]; exists {
			return fmt.Errorf("responders contains duplicate identity; use a typed note or authoritative record pointer")
		}
		seenResponders[responder.Identity] = struct{}{}
		_, hasResponse := responses[responder.Identity]
		switch responder.Status {
		case QuestionResponderPending:
			if hasResponse {
				return fmt.Errorf("responders pending responder has a response; use a typed note or authoritative record pointer")
			}
		case QuestionResponderCompleted:
			if !hasResponse {
				return fmt.Errorf("responders completed responder requires exactly one response; use a typed note or authoritative record pointer")
			}
		default:
			return fmt.Errorf("responders.status must be pending or completed; use a typed note or authoritative record pointer")
		}
	}
	for responder := range responses {
		if _, exists := seenResponders[responder]; !exists {
			return fmt.Errorf("responses has unknown responder; use a typed note or authoritative record pointer")
		}
	}
	if s.CurrentResponder() == "" && len(s.Responses) != len(s.Responders) {
		return fmt.Errorf("responders has no pending responder before all responders completed; use a typed note or authoritative record pointer")
	}
	return s.validateResolutionProvenance()
}

// validateResolutionProvenance makes QuestionState self-validating across its
// full durable lifecycle. A state without a kind is pre-resolution; a state
// with a kind is the resolved terminal representation and therefore requires
// every responder to be completed.
func (s QuestionState) validateResolutionProvenance() error {
	if s.ResolutionKind == "" {
		if s.ResolutionPointer != "" {
			return fmt.Errorf("resolution_pointer is unavailable before Question resolution; use a typed note or authoritative record pointer")
		}
		return nil
	}
	if s.CurrentResponder() != "" {
		return fmt.Errorf("resolution_kind requires every responder completed; use a typed note or authoritative record pointer")
	}
	switch s.ResolutionKind {
	case "local_clarification", "feature_change", "product_decision", "architecture_decision", "follow_up_work":
		return ValidateQuestionBoundedText("resolution_pointer", s.ResolutionPointer, 1, questionPointerMaxBytes)
	case "no_lasting_consequence":
		if s.ResolutionPointer != "" {
			return fmt.Errorf("no_lasting_consequence requires an empty resolution pointer; use a typed note or authoritative record pointer")
		}
		return nil
	default:
		return fmt.Errorf("resolution_kind is unsupported; use a typed note or authoritative record pointer")
	}
}

// ValidateQuestionBoundedText enforces the shared bounded-text contract used
// by every free-text Question field: trimmed, valid UTF-8, within
// [minBytes, maxBytes], and free of the forbidden credential/rendered-prompt/
// transcript markers. Shared by QuestionState validation (this package) and
// QuestionService's terminal-reason validation so the marker allowlist has a
// single source of truth.
func ValidateQuestionBoundedText(field, value string, minBytes, maxBytes int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != value {
		return fmt.Errorf("%s must be trimmed; use a typed note or authoritative record pointer", field)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8; use a typed note or authoritative record pointer", field)
	}
	if bytes := len(value); bytes < minBytes || bytes > maxBytes {
		return fmt.Errorf("%s must contain %d through %d UTF-8 bytes; use a typed note or authoritative record pointer", field, minBytes, maxBytes)
	}
	for _, pattern := range questionCredentialLabelPatterns {
		if pattern.MatchString(value) {
			return fmt.Errorf("%s contains forbidden credential, rendered prompt, or transcript material; use a typed note or authoritative record pointer", field)
		}
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"api_key", "system prompt", "user prompt", "assistant:"} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%s contains forbidden credential, rendered prompt, or transcript material; use a typed note or authoritative record pointer", field)
		}
	}
	return nil
}

// DecodeQuestionState reads only the Question-owned field while retaining the
// generic context representation for the caller to preserve on a later write.
func DecodeQuestionState(contextData *string) (*QuestionState, error) {
	if contextData == nil || strings.TrimSpace(*contextData) == "" {
		return nil, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*contextData), &fields); err != nil {
		return nil, fmt.Errorf("decode Question context data: %w", err)
	}
	if fields == nil {
		return nil, errors.New("decode Question context data: expected JSON object")
	}
	raw, found := fields["question_state"]
	if !found {
		return nil, nil
	}
	var state QuestionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode question_state: %w", err)
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("validate question_state: %w", err)
	}
	return &state, nil
}

// EncodeQuestionState stores the validated Question-owned state without
// expanding ContextData or dropping unrelated generic context keys.
func EncodeQuestionState(contextData *string, state QuestionState) (*string, error) {
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("validate question_state: %w", err)
	}
	fields := make(map[string]json.RawMessage)
	if contextData != nil && strings.TrimSpace(*contextData) != "" {
		if err := json.Unmarshal([]byte(*contextData), &fields); err != nil {
			return nil, fmt.Errorf("decode Question context data: %w", err)
		}
		if fields == nil {
			return nil, errors.New("decode Question context data: expected JSON object")
		}
	}
	encodedState, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("encode question_state: %w", err)
	}
	fields["question_state"] = encodedState
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode Question context data: %w", err)
	}
	value := string(encoded)
	return &value, nil
}

// Question is the bounded base record for the Q001-Q999 Question domain.
// Later workflow, provenance, response, and gate fields intentionally do not
// belong to this model.
type Question struct {
	BaseEntity
	Status    QuestionStatus `json:"status" db:"status"`
	Summary   string         `json:"summary" db:"summary"`
	Blocking  bool           `json:"blocking" db:"blocking"`
	Requester string         `json:"requester" db:"requester"`
}

// QuestionProjection is the metadata-only transport representation of a
// Question. ContextData, file paths, and sizing are persistence concerns; they
// must not cross Question CLI or HTTP read boundaries.
//
// Keep this type explicit rather than changing Question's JSON tags: the
// persisted model is also the typed I-01 context carrier used by services.
type QuestionProjection struct {
	ID          int64          `json:"id"`
	Key         string         `json:"key"`
	Title       string         `json:"title"`
	Slug        *string        `json:"slug,omitempty"`
	Description *string        `json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Status      QuestionStatus `json:"status"`
	Summary     string         `json:"summary"`
	Blocking    bool           `json:"blocking"`
	Requester   string         `json:"requester"`
}

// QuestionFullProjection is the deliberately separate, policy-gated Question
// read shape. It embeds the existing compact projection and exposes only the
// bounded, validated workflow fields needed by the assigned responder or
// resolution owner; it never transports raw ContextData.
type QuestionFullProjection struct {
	QuestionProjection
	ResolutionOwner   string              `json:"resolution_owner"`
	Responders        []QuestionResponder `json:"responders"`
	Responses         []QuestionResponse  `json:"responses"`
	ResolutionKind    string              `json:"resolution_kind,omitempty"`
	ResolutionPointer string              `json:"resolution_pointer,omitempty"`
}

// ProjectQuestion returns the only Question shape allowed on user-visible
// metadata transports. It deliberately does not copy ContextData.
func ProjectQuestion(question *Question) QuestionProjection {
	return QuestionProjection{
		ID:          question.ID,
		Key:         question.Key,
		Title:       question.Title,
		Slug:        question.Slug,
		Description: question.Description,
		CreatedAt:   question.CreatedAt,
		UpdatedAt:   question.UpdatedAt,
		Status:      question.Status,
		Summary:     question.Summary,
		Blocking:    question.Blocking,
		Requester:   question.Requester,
	}
}

// ProjectQuestions projects a slice of Questions to their user-visible
// metadata transport shape, in order.
func ProjectQuestions(questions []*Question) []QuestionProjection {
	projected := make([]QuestionProjection, 0, len(questions))
	for _, question := range questions {
		projected = append(projected, ProjectQuestion(question))
	}
	return projected
}

// ProjectQuestionFull returns the explicit bounded full-read representation.
// Callers own state validation and authorization before invoking it.
func ProjectQuestionFull(question *Question, state QuestionState) QuestionFullProjection {
	return QuestionFullProjection{
		QuestionProjection: ProjectQuestion(question),
		ResolutionOwner:    state.ResolutionOwner,
		Responders:         append([]QuestionResponder(nil), state.Responders...),
		Responses:          append([]QuestionResponse(nil), state.Responses...),
		ResolutionKind:     state.ResolutionKind,
		ResolutionPointer:  state.ResolutionPointer,
	}
}

var questionKeyPattern = regexp.MustCompile(`^Q[0-9]{3}$`)

// ErrInvalidQuestionKey is returned when a Question key is outside Q001-Q999.
var ErrInvalidQuestionKey = errors.New("invalid question key format: must match Q001 through Q999")

// GetEntityType returns EntityTypeQuestion.
func (q *Question) GetEntityType() EntityType { return EntityTypeQuestion }

// GetStatus returns the Question status as a string.
func (q *Question) GetStatus() string { return string(q.Status) }

// SetStatus updates the Question status. Workflow validation happens above the model.
func (q *Question) SetStatus(status string) { q.Status = QuestionStatus(status) }

// Validate checks the bounded Question base record. It intentionally leaves
// workflow-specific status validation to the workflow-aware service layer.
func (q *Question) Validate() error {
	if err := ValidateQuestionKey(q.Key); err != nil {
		return err
	}
	if strings.TrimSpace(q.Title) == "" {
		return ErrEmptyTitle
	}
	if strings.TrimSpace(q.Summary) == "" {
		return errors.New("question summary cannot be empty")
	}
	if strings.TrimSpace(q.Requester) == "" {
		return errors.New("question requester cannot be empty")
	}
	if strings.TrimSpace(string(q.Status)) == "" {
		return errors.New("question status cannot be empty")
	}
	if q.Size != nil {
		if err := ValidateSize(*q.Size); err != nil {
			return err
		}
	}
	return nil
}

// ValidateQuestionKey validates the strict uppercase Q001-Q999 key grammar.
func ValidateQuestionKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if !questionKeyPattern.MatchString(key) || key == "Q000" {
		return fmt.Errorf("%w: got %q", ErrInvalidQuestionKey, key)
	}
	return nil
}
