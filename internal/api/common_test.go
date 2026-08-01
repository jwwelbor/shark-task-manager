package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleServiceErrorClassifiesQuestionBusinessRuleErrors locks in that
// QuestionService's well-known state-machine and validation rejection
// messages map to 409/400 instead of falling through to a generic 500 that
// hides the actionable message the service already built. Only errors
// routed with entityLabel "question" get this treatment -- other entity
// types keep the prior generic-500 fallback.
func TestHandleServiceErrorClassifiesQuestionBusinessRuleErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		entityType string
		wantStatus int
	}{
		{"draft_or_open_conflict", errors.New("configure Question workflow Q001: Question must be draft or open"), "question", http.StatusConflict},
		{"already_configured_conflict", errors.New("configure Question workflow Q001: Question is already configured"), "question", http.StatusConflict},
		{"claim_mismatch_conflict", errors.New("record Question response Q001: active claim does not match responder session"), "question", http.StatusConflict},
		{"not_ready_for_resolution_conflict", errors.New("resolve Question Q001: Question must be ready for resolution with all responders completed"), "question", http.StatusConflict},
		{"already_terminal_conflict", errors.New("withdrawn Question Q001: Question is already terminal"), "question", http.StatusConflict},
		{"required_field_bad_request", errors.New("record Question response: session and responder are required"), "question", http.StatusBadRequest},
		{"destination_missing_bad_request", errors.New("resolve Question Q001: validate destination: document destination \"docs/x.md\" does not exist: stat: no such file"), "question", http.StatusBadRequest},
		{"unclassified_falls_back_to_500", errors.New("boom: unexpected database failure"), "question", http.StatusInternalServerError},
		{"non_question_entity_keeps_generic_500", errors.New("must be draft or open"), "bug", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleServiceError(w, tc.err, tc.entityType)
			if w.Code != tc.wantStatus {
				t.Errorf("handleServiceError(%q, %q) status = %d, want %d", tc.err, tc.entityType, w.Code, tc.wantStatus)
			}
		})
	}
}
