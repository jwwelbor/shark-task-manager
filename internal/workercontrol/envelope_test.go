package workercontrol

import (
	"strings"
	"testing"
)

func TestDecode_ValidFinalWithGateResult(t *testing.T) {
	raw := []byte(`{
		"kind": "final",
		"recommended_outcome": "deep_verify",
		"evidence": [
			{"kind": "test_run", "pointer": "artifacts/test.log", "summary": "all green", "command": "go test ./...", "working_directory": "/repo", "exit_code": 0}
		],
		"gate_result": {"schema_version": 1, "summary": "ok"}
	}`)
	env, err := Decode(raw)
	if err != nil {
		t.Fatalf("expected valid envelope to decode, got error: %v", err)
	}
	if env.Kind != KindFinal {
		t.Fatalf("expected kind final, got %q", env.Kind)
	}
	if env.RecommendedOutcome != "deep_verify" {
		t.Fatalf("expected recommended_outcome deep_verify, got %q", env.RecommendedOutcome)
	}
	if len(env.Evidence) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(env.Evidence))
	}
	if len(env.GateResult) == 0 {
		t.Fatal("expected nested gate_result to be captured")
	}
}

func TestDecode_ValidFinalWithoutGateResult(t *testing.T) {
	raw := []byte(`{"kind": "final", "recommended_outcome": "pass", "evidence": []}`)
	env, err := Decode(raw)
	if err != nil {
		t.Fatalf("expected valid legacy-shaped final envelope to decode, got error: %v", err)
	}
	if len(env.GateResult) != 0 {
		t.Fatal("expected no gate_result payload")
	}
}

func TestDecode_ValidNonFinalKinds(t *testing.T) {
	cases := []string{
		`{"kind": "needs_council", "evidence": []}`,
		`{"kind": "blocked_external", "evidence": []}`,
		`{"kind": "failed", "evidence": []}`,
		`{"kind": "question", "entity_key": "E01-F01-001", "category": "architecture", "question": "which approach?", "why_blocking": "cannot proceed without a decision", "evidence": []}`,
	}
	for _, raw := range cases {
		if _, err := Decode([]byte(raw)); err != nil {
			t.Fatalf("expected %s to decode, got error: %v", raw, err)
		}
	}
}

func TestDecode_UnknownKindRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "bogus", "evidence": []}`))
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Field != "kind" {
		t.Fatalf("expected field kind, got %q", ve.Field)
	}
}

func TestDecode_DuplicateJSONKeyRejected(t *testing.T) {
	// T-E34-F05-004 UAT rework: standard encoding/json.Unmarshal silently
	// accepts a duplicate object key (last one wins), the same class of
	// defect internal/gateresult.Decode rejects for the nested gate_result
	// payload. A hostile or malformed second value for, say,
	// recommended_outcome or a nested evidence field must not slip past this
	// package's checks undetected.
	cases := map[string]string{
		"top-level duplicate kind":                      `{"kind":"final","kind":"final","recommended_outcome":"pass","evidence":[]}`,
		"top-level duplicate recommended_outcome":       `{"kind":"final","recommended_outcome":"pass","recommended_outcome":"fail","evidence":[]}`,
		"duplicate key inside a nested evidence object": `{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"test_run","kind":"other","pointer":"artifacts/test.log"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(raw)); err == nil {
				t.Fatalf("expected duplicate JSON key to be rejected: %s", raw)
			}
		})
	}
}

func TestDecode_MalformedJSONRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "final",`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecode_UnknownTopLevelFieldRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "final", "recommended_outcome": "pass", "evidence": [], "extra": "nope"}`))
	if err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Class != ErrorClassUnknownField {
		t.Fatalf("expected unknown_field ValidationError, got %#v", err)
	}
}

func TestDecode_TrailingContentRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "evidence": []}{"kind": "failed", "evidence": []}`))
	if err == nil {
		t.Fatal("expected error for trailing content")
	}
}

func TestDecode_RecommendedOutcomeMissingForFinalRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "final", "evidence": []}`))
	if err == nil {
		t.Fatal("expected error when kind is final but recommended_outcome is absent")
	}
}

func TestDecode_RecommendedOutcomePresentForNonFinalRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "recommended_outcome": "pass", "evidence": []}`))
	if err == nil {
		t.Fatal("expected error when a non-final kind carries recommended_outcome")
	}
}

func TestDecode_GateResultPresentForNonFinalRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [], "gate_result": {"schema_version": 1, "summary": "x"}}`))
	if err == nil {
		t.Fatal("expected error when a non-final kind carries gate_result")
	}
}

func TestDecode_QuestionMissingRequiredFieldsRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "question", "evidence": []}`))
	if err == nil {
		t.Fatal("expected error when kind is question but required fields are absent")
	}
}

func TestDecode_QuestionFieldsPresentForNonQuestionRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [], "entity_key": "E01-F01-001"}`))
	if err == nil {
		t.Fatal("expected error when a non-question kind carries question fields")
	}
}

func TestDecode_EvidenceOverBoundsRejected(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"kind": "failed", "evidence": [`)
	for i := 0; i < MaxEvidenceItems+1; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"kind": "x", "pointer": "p"}`)
	}
	b.WriteString(`]}`)
	_, err := Decode([]byte(b.String()))
	if err == nil {
		t.Fatal("expected error for evidence collection over the max item bound")
	}
}

func TestDecode_EvidenceMissingKindRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [{"pointer": "p"}]}`))
	if err == nil {
		t.Fatal("expected error for evidence item with empty kind")
	}
}

func TestDecode_EvidenceForbiddenContentRejected(t *testing.T) {
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [{"kind": "x", "pointer": "p", "summary": "Authorization: Bearer abc123"}]}`))
	if err == nil {
		t.Fatal("expected error for evidence summary containing forbidden credential material")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Class != ErrorClassForbiddenContent {
		t.Fatalf("expected forbidden_content ValidationError, got %#v", err)
	}
}

// TestDecode_EvidenceOpaqueRawFieldsForbiddenContentRejected proves
// EvidenceRef.Counts/ExpectedSkips/UnexpectedSkips are not a bypass for
// REQ-NF-001's forbidden-marker guarantee: a forbidden marker embedded
// anywhere in these runner-native json.RawMessage fields — including nested
// inside an object value, not just a top-level string — must be rejected the
// same way a forbidden marker in a typed string field like summary is
// (code-review round 3 CRITICAL finding).
func TestDecode_EvidenceOpaqueRawFieldsForbiddenContentRejected(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "counts",
			json: `{"kind": "failed", "evidence": [{"kind": "x", "pointer": "p", "counts": {"passed": 3, "note": "Authorization: Bearer abc123"}}]}`,
		},
		{
			name: "expected_skips",
			json: `{"kind": "failed", "evidence": [{"kind": "x", "pointer": "p", "expected_skips": ["system prompt leak"]}]}`,
		},
		{
			name: "unexpected_skips",
			json: `{"kind": "failed", "evidence": [{"kind": "x", "pointer": "p", "unexpected_skips": {"reason": "api_key=abc123"}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.json))
			if err == nil {
				t.Fatalf("expected error for forbidden content embedded in %s", tt.name)
			}
			ve, ok := err.(*ValidationError)
			if !ok || ve.Class != ErrorClassForbiddenContent {
				t.Fatalf("expected forbidden_content ValidationError, got %#v", err)
			}
		})
	}
}

// TestDecode_EvidenceOpaqueRawFieldOversizedRejected proves the same three
// fields are size-bounded, not just content-checked, so an unbounded
// runner-native blob cannot inflate persisted note metadata without limit.
func TestDecode_EvidenceOpaqueRawFieldOversizedRejected(t *testing.T) {
	huge := `"` + strings.Repeat("a", OpaqueJSONMaxBytes+1) + `"`
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [{"kind": "x", "pointer": "p", "counts": ` + huge + `}]}`))
	if err == nil {
		t.Fatal("expected error for an oversized counts field")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Class != ErrorClassBounds {
		t.Fatalf("expected bounds ValidationError, got %#v", err)
	}
}

func TestDecode_OversizedEnvelopeRejected(t *testing.T) {
	huge := strings.Repeat("a", MaxEnvelopeBytes+1)
	_, err := Decode([]byte(`{"kind": "failed", "evidence": [], "pad": "` + huge + `"}`))
	if err == nil {
		t.Fatal("expected error for an oversized envelope")
	}
}
