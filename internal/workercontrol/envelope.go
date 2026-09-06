// Package workercontrol implements the first Go type and parser for the
// canonical worker-control envelope described by
// skills/shark-attack/context/worker-control-schema.yaml (E38-F09, ADR-E34-01
// in docs/plan/E34-prompt-and-skill-improvements/architecture.md). That file
// is a schema/prose contract only: no Go type or processor for it existed
// before this package. Mirror it exactly here — do not add a second
// envelope, marker, or duplicate field list (T-E34-F05-004, REQ-F-005).
//
// This package owns only the outer envelope's shape: `kind`, the opaque
// `recommended_outcome` (kind: final only), and the bounded common
// `evidence[]`/EvidenceRef collection, plus the kind-specific fields for the
// other four kinds so the type is a faithful mirror of the schema. It
// deliberately does not own:
//   - The nested `gate_result` payload's shape or validation — that is
//     internal/gateresult (I-02 GateResult v1), consumed here only as opaque
//     json.RawMessage.
//   - Persistence, replay, or the sidecar transport — internal/gaterun /
//     internal/gatepersist.
//   - Interpreting recommended_outcome against a workflow's configured
//     outcomes — that remains `shark status advance --outcome`'s existing,
//     unrelated job (internal/cli/commands/status_group.go) plus, for
//     gate_result_v1 steps, the caller's outcome_roles lookup.
//   - The Question-response handoff (a worker's compact reply while *working
//     a Question entity*) — that is a different, pre-existing bounded
//     contract (internal/runner.parseQuestionResponseHandoff); `kind:
//     question` here is the schema's *other* concept, a worker escalating a
//     new question while working on some other entity.
package workercontrol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Bounds. Reused verbatim from internal/gateresult so the outer envelope and
// the nested GateResult payload share one set of bounds rather than each
// defining its own (architecture.md I-02: "These constants live in one
// GateResult model package"); this package does not import
// internal/gateresult (see the package doc's scope boundaries), so the
// values are restated here instead of aliased.
const (
	// SummaryMaxBytes bounds EvidenceRef.Summary.
	SummaryMaxBytes = 1000
	// PointerMaxBytes bounds EvidenceRef.Pointer and Command/WorkingDirectory.
	PointerMaxBytes = 2048
	// IdentityMaxBytes bounds short identity fields (Kind, RecommendedOutcome,
	// EvidenceRef.Kind, Question.Category).
	IdentityMaxBytes = 256
	// MaxEvidenceItems bounds the evidence[] collection.
	MaxEvidenceItems = 100
	// MaxEnvelopeBytes bounds the entire envelope (including any nested
	// gate_result payload) before it is even parsed.
	MaxEnvelopeBytes = 256 * 1024
	// OpaqueJSONMaxBytes bounds each of EvidenceRef's runner-native raw JSON
	// fields (Counts, ExpectedSkips, UnexpectedSkips) — these are
	// deliberately shapeless (architecture.md: "runner-native counts"), so
	// they cannot be bounded by boundedText's string-field contract; this is
	// the size ceiling applied before their content is walked for forbidden
	// markers (see validateOpaqueJSON).
	OpaqueJSONMaxBytes = 4096
)

// Kind is the closed set of worker-control envelope kinds.
type Kind string

const (
	KindFinal           Kind = "final"
	KindQuestion        Kind = "question"
	KindNeedsCouncil    Kind = "needs_council"
	KindBlockedExternal Kind = "blocked_external"
	KindFailed          Kind = "failed"
)

func (k Kind) valid() bool {
	switch k {
	case KindFinal, KindQuestion, KindNeedsCouncil, KindBlockedExternal, KindFailed:
		return true
	default:
		return false
	}
}

// ErrorClass classifies a validation failure without echoing rejected
// content, mirroring internal/gateresult.ErrorClass's taxonomy (REQ-NF-001).
type ErrorClass string

const (
	ErrorClassShape            ErrorClass = "shape"
	ErrorClassUnknownField     ErrorClass = "unknown_field"
	ErrorClassBounds           ErrorClass = "bounds"
	ErrorClassForbiddenContent ErrorClass = "forbidden_content"
	ErrorClassDuplicate        ErrorClass = "duplicate"
)

// ValidationError is the one error type this package returns for a
// structurally rejected envelope. Field and Class are always safe to log;
// Message never echoes the rejected value.
type ValidationError struct {
	Field   string
	Class   ErrorClass
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("worker-control envelope (%s): %s", e.Class, e.Message)
	}
	return fmt.Sprintf("worker-control envelope %s (%s): %s", e.Field, e.Class, e.Message)
}

func newValidationError(field string, class ErrorClass, message string) *ValidationError {
	return &ValidationError{Field: field, Class: class, Message: message}
}

// EvidenceRef is the outer envelope's common evidence collection entry
// (architecture.md "The outer final envelope's EvidenceRef contains kind,
// pointer, and an optional bounded summary. For executable evidence it also
// contains exact command, working_directory, exit_code, runner-native
// counts, expected_skips, and unexpected_skips.").
type EvidenceRef struct {
	Kind             string          `json:"kind"`
	Pointer          string          `json:"pointer"`
	Summary          string          `json:"summary,omitempty"`
	Command          string          `json:"command,omitempty"`
	WorkingDirectory string          `json:"working_directory,omitempty"`
	ExitCode         *int            `json:"exit_code,omitempty"`
	Counts           json.RawMessage `json:"counts,omitempty"`
	ExpectedSkips    json.RawMessage `json:"expected_skips,omitempty"`
	UnexpectedSkips  json.RawMessage `json:"unexpected_skips,omitempty"`
}

func (e EvidenceRef) validate(index int) error {
	prefix := fmt.Sprintf("evidence[%d]", index)
	if err := boundedText(prefix+".kind", e.Kind, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".pointer", e.Pointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	if e.Summary != "" {
		if err := boundedText(prefix+".summary", e.Summary, 1, SummaryMaxBytes); err != nil {
			return err
		}
	}
	if e.Command != "" {
		if err := boundedText(prefix+".command", e.Command, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	if e.WorkingDirectory != "" {
		if err := boundedText(prefix+".working_directory", e.WorkingDirectory, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	if err := validateOpaqueJSON(prefix+".counts", e.Counts); err != nil {
		return err
	}
	if err := validateOpaqueJSON(prefix+".expected_skips", e.ExpectedSkips); err != nil {
		return err
	}
	if err := validateOpaqueJSON(prefix+".unexpected_skips", e.UnexpectedSkips); err != nil {
		return err
	}
	return nil
}

// Envelope is the canonical worker-control envelope. Only the fields
// relevant to the worker's declared Kind are populated; see
// worker-control-schema.yaml for the per-kind shape this mirrors.
type Envelope struct {
	Kind Kind `json:"kind"`

	// kind: final only.
	RecommendedOutcome string          `json:"recommended_outcome,omitempty"`
	GateResult         json.RawMessage `json:"gate_result,omitempty"`

	// kind: question only. QuestionID is deliberately absent from the schema
	// (D-005) and therefore from this type too.
	EntityKey      string   `json:"entity_key,omitempty"`
	Category       string   `json:"category,omitempty"`
	Question       string   `json:"question,omitempty"`
	WhyBlocking    string   `json:"why_blocking,omitempty"`
	Options        []string `json:"options,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`

	// Common to every kind.
	Evidence []EvidenceRef `json:"evidence"`
}

// Decode parses and structurally validates a candidate worker-control
// envelope. It rejects unknown top-level fields, malformed JSON shapes, and
// trailing content after the object, matching internal/gateresult.Decode's
// strictness so the outer and nested parsers apply the same "no second
// envelope shape" discipline.
func Decode(data []byte) (*Envelope, error) {
	if len(data) > MaxEnvelopeBytes {
		return nil, newValidationError("", ErrorClassBounds, "must not exceed the maximum envelope size")
	}

	if err := rejectDuplicateKeys(data); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var env Envelope
	if err := dec.Decode(&env); err != nil {
		if field, ok := unknownFieldName(err); ok {
			return nil, newValidationError(field, ErrorClassUnknownField, "is not a recognized worker-control envelope field")
		}
		return nil, newValidationError("", ErrorClassShape, "must be a well-formed JSON object matching the worker-control envelope shape")
	}
	if dec.More() {
		return nil, newValidationError("", ErrorClassShape, "must contain exactly one JSON object; a second top-level value is not permitted")
	}

	if err := env.Validate(); err != nil {
		return nil, err
	}
	return &env, nil
}

// Validate checks every structural invariant of the envelope shape: a known
// kind, kind-appropriate field presence, and bounded evidence.
func (e *Envelope) Validate() error {
	if !e.Kind.valid() {
		return newValidationError("kind", ErrorClassShape, "must be one of final, question, needs_council, blocked_external, failed")
	}

	if e.Kind == KindFinal {
		if strings.TrimSpace(e.RecommendedOutcome) == "" {
			return newValidationError("recommended_outcome", ErrorClassShape, "is required when kind is final")
		}
		if err := boundedText("recommended_outcome", e.RecommendedOutcome, 1, IdentityMaxBytes); err != nil {
			return err
		}
	} else if e.RecommendedOutcome != "" {
		return newValidationError("recommended_outcome", ErrorClassShape, "must be absent unless kind is final")
	}

	if e.Kind != KindFinal && len(e.GateResult) > 0 {
		return newValidationError("gate_result", ErrorClassShape, "must be absent unless kind is final")
	}

	if e.Kind == KindQuestion {
		if strings.TrimSpace(e.EntityKey) == "" || strings.TrimSpace(e.Category) == "" ||
			strings.TrimSpace(e.Question) == "" || strings.TrimSpace(e.WhyBlocking) == "" {
			return newValidationError("", ErrorClassShape, "entity_key, category, question, and why_blocking are required when kind is question")
		}
	} else if e.EntityKey != "" || e.Category != "" || e.Question != "" || e.WhyBlocking != "" ||
		len(e.Options) > 0 || e.Recommendation != "" {
		return newValidationError("", ErrorClassShape, "question fields must be absent unless kind is question")
	}

	if len(e.Evidence) > MaxEvidenceItems {
		return newValidationError("evidence", ErrorClassBounds, "must not exceed the maximum evidence collection size")
	}
	for i, ev := range e.Evidence {
		if err := ev.validate(i); err != nil {
			return err
		}
	}

	return nil
}

// rejectDuplicateKeys walks the raw JSON token stream and rejects a document
// that repeats a key within the same JSON object, at any nesting depth.
// encoding/json.Unmarshal silently accepts a duplicate key (the last value
// wins), which would let a hostile or malformed second value for, say,
// recommended_outcome or a nested evidence entry slip past every field-level
// check in this package undetected — the same class of defect
// internal/gateresult.rejectDuplicateKeys closes for the nested gate_result
// payload (T-E34-F05-001). internal/gateresult's implementation is
// unexported, so this is a local, deliberately identical port rather than an
// import (T-E34-F05-004 UAT rework, CRITICAL finding sibling #3) — this
// package's own doc comment already commits to restating gateresult's shared
// constants rather than importing it, for the same package-boundary reason.
func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		// Malformed JSON is reported by the typed decode that follows.
		return nil
	}
	return walkDuplicateKeys(dec, tok)
}

// walkDuplicateKeys checks tok (already read from dec) and, if it opens an
// object or array, recurses into dec to check every nested value as well.
// Any decode error encountered while walking is deliberately swallowed: the
// typed decode that runs after this pass is the single source of truth for
// shape/malformed-JSON errors, so this pass only ever reports duplicate keys.
func walkDuplicateKeys(dec *json.Decoder, tok json.Token) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		return walkDuplicateKeysObject(dec)
	case '[':
		return walkDuplicateKeysArray(dec)
	default:
		return nil
	}
}

func walkDuplicateKeysObject(dec *json.Decoder) error {
	seen := make(map[string]struct{})
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil
		}
		if _, exists := seen[key]; exists {
			return newValidationError(key, ErrorClassDuplicate, "must not appear more than once in the same JSON object")
		}
		seen[key] = struct{}{}

		valTok, err := dec.Token()
		if err != nil {
			return nil
		}
		if err := walkDuplicateKeys(dec, valTok); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // consume closing '}'
		return nil
	}
	return nil
}

func walkDuplicateKeysArray(dec *json.Decoder) error {
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		if err := walkDuplicateKeys(dec, tok); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // consume closing ']'
		return nil
	}
	return nil
}

// unknownFieldName extracts the rejected field name from the stdlib
// DisallowUnknownFields error message. The message never contains rejected
// values, only the field name, so surfacing it does not violate REQ-NF-001.
func unknownFieldName(err error) (string, bool) {
	const marker = `json: unknown field "`
	msg := err.Error()
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return "", false
	}
	rest := msg[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// credentialLabelPatterns and forbiddenSubstrings mirror
// internal/models/question.go's ValidateQuestionBoundedText marker set
// exactly (REQ-NF-001: "Reuse the Question model's bounded-text and
// forbidden-marker approach"), matching internal/gateresult/text.go's own
// local reimplementation for the same reason: this package classifies the
// failure into its own ErrorClass taxonomy instead of parsing another
// package's error string.
var credentialLabelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password\s*=`),
	regexp.MustCompile(`(?i)authorization\s*:`),
	regexp.MustCompile(`(?i)bearer(?:\s|:)`),
}

var forbiddenSubstrings = []string{"api_key", "system prompt", "user prompt", "assistant:"}

func containsForbiddenContent(value string) bool {
	for _, pattern := range credentialLabelPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	lower := strings.ToLower(value)
	for _, marker := range forbiddenSubstrings {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func boundedText(field, value string, minBytes, maxBytes int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != value {
		return newValidationError(field, ErrorClassBounds, "must be trimmed")
	}
	if !utf8.ValidString(value) {
		return newValidationError(field, ErrorClassBounds, "must be valid UTF-8")
	}
	if n := len(value); n < minBytes || n > maxBytes {
		return newValidationError(field, ErrorClassBounds, "must contain "+strconv.Itoa(minBytes)+" through "+strconv.Itoa(maxBytes)+" UTF-8 bytes")
	}
	if containsForbiddenContent(value) {
		return newValidationError(field, ErrorClassForbiddenContent, "must not contain credential, rendered prompt, or transcript material")
	}
	return nil
}

// validateOpaqueJSON bounds and forbidden-content-checks one of EvidenceRef's
// runner-native raw JSON fields (Counts, ExpectedSkips, UnexpectedSkips).
// These fields are declared json.RawMessage rather than a typed struct
// because their shape is deliberately runner-native/opaque (architecture.md
// I-02), so boundedText's fixed-shape string contract does not apply
// directly. REQ-NF-001's forbidden-marker guarantee still must hold end to
// end: this walks the decoded JSON value irrespective of shape and applies
// the same containsForbiddenContent check boundedText uses to every string
// it finds (object keys and string values, at any nesting depth), closing
// the gap the code-review round-3 report identified — without this pass a
// forbidden marker embedded in, e.g., counts.notes would flow unchecked into
// durably persisted note metadata via gatepersist.Coordinator.Persist's
// writeNote (meta[metaEvidence]).
func validateOpaqueJSON(field string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > OpaqueJSONMaxBytes {
		return newValidationError(field, ErrorClassBounds, "must not exceed "+strconv.Itoa(OpaqueJSONMaxBytes)+" UTF-8 bytes")
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return newValidationError(field, ErrorClassShape, "must be a well-formed JSON value")
	}
	return walkOpaqueJSONForForbiddenContent(field, value)
}

// walkOpaqueJSONForForbiddenContent recursively visits every string in a
// decoded JSON value (object keys, string values, and array elements, at any
// depth) and rejects the first one that trips containsForbiddenContent.
func walkOpaqueJSONForForbiddenContent(field string, value interface{}) error {
	switch v := value.(type) {
	case string:
		if containsForbiddenContent(v) {
			return newValidationError(field, ErrorClassForbiddenContent, "must not contain credential, rendered prompt, or transcript material")
		}
	case map[string]interface{}:
		for key, nested := range v {
			if containsForbiddenContent(key) {
				return newValidationError(field, ErrorClassForbiddenContent, "must not contain credential, rendered prompt, or transcript material")
			}
			if err := walkOpaqueJSONForForbiddenContent(field, nested); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, nested := range v {
			if err := walkOpaqueJSONForForbiddenContent(field, nested); err != nil {
				return err
			}
		}
	}
	return nil
}
