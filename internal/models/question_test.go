package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProjectQuestionOmitsContextData(t *testing.T) {
	sentinel := `{"current_step":"must-not-project"}`
	projection := ProjectQuestion(&Question{
		BaseEntity: BaseEntity{Key: "Q001", Title: "Question", ContextData: &sentinel},
		Status:     QuestionStatusDraft,
		Summary:    "Summary",
		Requester:  "owner",
	})
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("Marshal(ProjectQuestion()) error = %v", err)
	}
	if strings.Contains(string(encoded), "must-not-project") || strings.Contains(string(encoded), "context_data") {
		t.Fatalf("ProjectQuestion() exposed context data: %s", encoded)
	}
}

func TestQuestion_Validate_TC003(t *testing.T) {
	tests := []struct {
		name     string
		question Question
		wantErr  bool
	}{
		{
			name: "valid base question",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q001", Title: "Which release?"},
				Status:     QuestionStatusDraft,
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
		},
		{
			name: "valid upper boundary key",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q999", Title: "Which release?"},
				Status:     QuestionStatus("archived"),
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
		},
		{
			name: "empty key",
			question: Question{
				BaseEntity: BaseEntity{Title: "Which release?"},
				Status:     QuestionStatusDraft,
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
			wantErr: true,
		},
		{
			name: "malformed key",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q000", Title: "Which release?"},
				Status:     QuestionStatusDraft,
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
			wantErr: true,
		},
		{
			name: "empty title",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q001"},
				Status:     QuestionStatusDraft,
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
			wantErr: true,
		},
		{
			name: "empty summary",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q001", Title: "Which release?"},
				Status:     QuestionStatusDraft,
				Requester:  "Product",
			},
			wantErr: true,
		},
		{
			name: "empty requester",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q001", Title: "Which release?"},
				Status:     QuestionStatusDraft,
				Summary:    "Choose the release window.",
			},
			wantErr: true,
		},
		{
			name: "empty status",
			question: Question{
				BaseEntity: BaseEntity{Key: "Q001", Title: "Which release?"},
				Summary:    "Choose the release window.",
				Requester:  "Product",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.question.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("TC-003 Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuestion_EntityConformance_TC003(t *testing.T) {
	var entity Entity = &Question{
		BaseEntity: BaseEntity{Key: "Q001", Title: "Which release?"},
		Status:     QuestionStatusDraft,
		Summary:    "Choose the release window.",
		Requester:  "Product",
	}

	if got := entity.GetEntityType(); got != EntityTypeQuestion {
		t.Errorf("TC-003 GetEntityType() = %q, want %q", got, EntityTypeQuestion)
	}
	if got := entity.GetStatus(); got != "draft" {
		t.Errorf("TC-003 GetStatus() = %q, want draft", got)
	}
}

// TestValidateQuestionBoundedTextDelimiterMarkers_TC107 keeps credential-label
// matching narrow: delimiter-permuted labels are rejected while ordinary
// prose that mentions the same words remains usable in Question fields.
func TestValidateQuestionBoundedTextDelimiterMarkers_TC107(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "bearer colon", value: "BEARER: credential", wantErr: true},
		{name: "password space equals", value: "password = credential", wantErr: true},
		{name: "authorization space colon", value: "Authorization : credential", wantErr: true},
		{name: "password policy prose", value: "The password policy requires twelve characters."},
		{name: "authorization model prose", value: "The authorization model has two roles."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuestionBoundedText("responses.summary", tt.value, 1, questionResponseMaxBytes)
			if (err != nil) != tt.wantErr {
				t.Fatalf("TC-107 ValidateQuestionBoundedText(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TC-102: Question workflow state is a bounded Question-owned value. The
// current responder is derived from the first pending responder rather than
// accepted as independently supplied routing input.
func TestQuestionStateValidateAndDeriveCurrentResponder_TC102(t *testing.T) {
	valid := QuestionState{
		ResolutionOwner: "release-owner",
		Responders: []QuestionResponder{
			{Identity: "alice", Status: QuestionResponderPending},
			{Identity: "bob", Status: QuestionResponderPending},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("TC-102 valid QuestionState.Validate() error = %v", err)
	}
	if got := valid.CurrentResponder(); got != "alice" {
		t.Fatalf("TC-102 CurrentResponder() = %q, want alice", got)
	}

	tests := []struct {
		name  string
		state QuestionState
		want  string
	}{
		{"empty owner", QuestionState{Responders: valid.Responders}, "resolution_owner"},
		{"eleven responders", QuestionState{ResolutionOwner: "owner", Responders: []QuestionResponder{{Identity: "r01", Status: QuestionResponderPending}, {Identity: "r02", Status: QuestionResponderPending}, {Identity: "r03", Status: QuestionResponderPending}, {Identity: "r04", Status: QuestionResponderPending}, {Identity: "r05", Status: QuestionResponderPending}, {Identity: "r06", Status: QuestionResponderPending}, {Identity: "r07", Status: QuestionResponderPending}, {Identity: "r08", Status: QuestionResponderPending}, {Identity: "r09", Status: QuestionResponderPending}, {Identity: "r10", Status: QuestionResponderPending}, {Identity: "r11", Status: QuestionResponderPending}}}, "responders"},
		{"duplicate responder", QuestionState{ResolutionOwner: "owner", Responders: []QuestionResponder{{Identity: "alice", Status: QuestionResponderPending}, {Identity: "alice", Status: QuestionResponderPending}}}, "duplicate"},
		{"completed without response", QuestionState{ResolutionOwner: "owner", Responders: []QuestionResponder{{Identity: "alice", Status: QuestionResponderCompleted}}}, "exactly one response"},
		{"completed response without session", QuestionState{ResolutionOwner: "owner", Responders: []QuestionResponder{{Identity: "alice", Status: QuestionResponderCompleted}}, Responses: []QuestionResponse{{Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}}}, "responses.session_id"},
		{"credential-shaped owner", QuestionState{ResolutionOwner: "api_key=secret", Responders: valid.Responders}, "credential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.want) {
				t.Fatalf("TC-102 QuestionState.Validate() error = %v, want %q", err, tt.want)
			}
		})
	}

	// TC-102: the 10-responder upper bound is inclusive on the accepted
	// side too -- "eleven responders" above only pins the rejected side.
	tenResponders := make([]QuestionResponder, 10)
	for i := range tenResponders {
		tenResponders[i] = QuestionResponder{Identity: fmt.Sprintf("r%02d", i+1), Status: QuestionResponderPending}
	}
	if err := (QuestionState{ResolutionOwner: "owner", Responders: tenResponders}).Validate(); err != nil {
		t.Fatalf("TC-102 QuestionState.Validate() with exactly 10 responders error = %v, want nil", err)
	}
}

// TC-106: resolved QuestionState is a durable I-02 value. Every consumer
// must be able to decode the exact state written by Resolve; validation cannot
// be limited to the pre-resolution phase.
func TestQuestionStateDecodeRoundTripsResolvedProvenance_TC106(t *testing.T) {
	recordedAt := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		kind, pointer string
	}{
		{"local_clarification", "note:Q001:42"},
		{"feature_change", "docs/spec.md"},
		{"product_decision", "docs/product/progress.md#decision-1"},
		{"architecture_decision", "docs/architecture/adr/ADR-001.md;docs/architecture/system.md"},
		{"follow_up_work", "E39-F02"},
		{"no_lasting_consequence", ""},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			state := QuestionState{
				ResolutionOwner: "release-owner",
				Responders:      []QuestionResponder{{Identity: "alice", Status: QuestionResponderCompleted}},
				Responses: []QuestionResponse{{
					SessionID: "session-a", Responder: "alice", Summary: "approved",
					EvidencePointer: "docs/spec.md", RecordedAt: recordedAt,
				}},
				ResolutionKind: tc.kind, ResolutionPointer: tc.pointer,
			}

			encoded, err := EncodeQuestionState(nil, state)
			if err != nil {
				t.Fatalf("TC-106 EncodeQuestionState() error = %v", err)
			}
			got, err := DecodeQuestionState(encoded)
			if err != nil {
				t.Fatalf("TC-106 DecodeQuestionState() error = %v", err)
			}
			if got == nil || !reflect.DeepEqual(*got, state) {
				t.Fatalf("TC-106 decoded QuestionState = %#v, want %#v", got, state)
			}
		})
	}
}

// TC-106: the same shared validator that permits durable resolved state must
// reject incomplete or malformed terminal provenance before serialization.
func TestQuestionStateRejectsInvalidResolvedProvenance_TC106(t *testing.T) {
	base := QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []QuestionResponder{{Identity: "alice", Status: QuestionResponderCompleted}},
		Responses: []QuestionResponse{{
			SessionID: "session-a", Responder: "alice", Summary: "approved",
			EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC(),
		}},
	}
	for _, tc := range []struct {
		name, kind, pointer string
		state               QuestionState
	}{
		{name: "unknown kind", kind: "manual_override", pointer: "docs/spec.md"},
		{name: "missing pointer", kind: "feature_change"},
		{name: "pointer forbidden for no lasting consequence", kind: "no_lasting_consequence", pointer: "docs/spec.md"},
		{name: "pending responder", kind: "feature_change", pointer: "docs/spec.md", state: QuestionState{ResolutionOwner: "release-owner", Responders: []QuestionResponder{{Identity: "alice", Status: QuestionResponderPending}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := base
			if tc.state.Responders != nil {
				state = tc.state
			}
			state.ResolutionKind, state.ResolutionPointer = tc.kind, tc.pointer
			if _, err := EncodeQuestionState(nil, state); err == nil {
				t.Fatal("TC-106 EncodeQuestionState() error = nil, want invalid terminal provenance rejection")
			}
		})
	}
}
