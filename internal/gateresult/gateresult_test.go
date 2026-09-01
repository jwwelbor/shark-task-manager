package gateresult

import (
	"encoding/json"
	"strings"
	"testing"
)

// validPayload returns a canonical, schema-valid GateResult v1 JSON document.
// Each test mutates a copy so failures are isolated to the field under test.
func validPayload() map[string]interface{} {
	return map[string]interface{}{
		"schema_version": 1,
		"summary":        "code review found two findings",
		"findings": []interface{}{
			map[string]interface{}{
				"severity":        "major",
				"class_key":       "missing-input-validation",
				"class_statement": "Handlers accept unbounded free text without a length check",
				"fingerprint":     "fp-001",
				"disposition":     "open",
			},
		},
		"kickbacks": []interface{}{
			map[string]interface{}{
				"entity_key":    "T-E34-F05-002",
				"target_status": "todo",
				"reason":        "Missing input validation must be fixed before merge",
			},
		},
	}
}

func encode(t *testing.T, payload map[string]interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

func TestDecode_ValidPayloadAccepted(t *testing.T) {
	result, err := Decode(encode(t, validPayload()))
	if err != nil {
		t.Fatalf("expected valid payload to decode, got error: %v", err)
	}
	if result.SchemaVersion != 1 {
		t.Fatalf("expected schema_version 1, got %d", result.SchemaVersion)
	}
	if len(result.Findings) != 1 || len(result.Kickbacks) != 1 {
		t.Fatalf("expected one finding and one kickback, got %d findings, %d kickbacks", len(result.Findings), len(result.Kickbacks))
	}
}

func TestDecode_SchemaVersion(t *testing.T) {
	cases := []struct {
		name    string
		version interface{}
		wantErr bool
	}{
		{"exactly one accepted", 1, false},
		{"zero rejected", 0, true},
		{"two rejected as unknown version", 2, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			payload["schema_version"] = tc.version
			_, err := Decode(encode(t, payload))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for schema_version=%v", tc.version)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for schema_version=%v, got %v", tc.version, err)
			}
		})
	}
}

func TestDecode_SummaryBounds(t *testing.T) {
	cases := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"limit-1 accepted", SummaryMaxBytes - 1, false},
		{"limit accepted", SummaryMaxBytes, false},
		{"limit+1 rejected", SummaryMaxBytes + 1, true},
		{"empty rejected", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			payload["summary"] = strings.Repeat("a", tc.length)
			_, err := Decode(encode(t, payload))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for summary length %d", tc.length)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for summary length %d, got %v", tc.length, err)
			}
		})
	}
}

func TestDecode_CollectionBounds(t *testing.T) {
	buildFindings := func(n int) []interface{} {
		findings := make([]interface{}, 0, n)
		for i := 0; i < n; i++ {
			findings = append(findings, map[string]interface{}{
				"severity":        "minor",
				"class_key":       "class",
				"class_statement": "statement",
				"fingerprint":     stringsRepeatUnique("fp-", i),
				"disposition":     "fixed",
			})
		}
		return findings
	}
	cases := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{"limit-1 accepted", MaxCollectionItems - 1, false},
		{"limit accepted", MaxCollectionItems, false},
		{"limit+1 rejected", MaxCollectionItems + 1, true},
		{"empty accepted", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			payload["findings"] = buildFindings(tc.count)
			_, err := Decode(encode(t, payload))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for findings count %d", tc.count)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for findings count %d, got %v", tc.count, err)
			}
		})
	}
}

func stringsRepeatUnique(prefix string, i int) string {
	return prefix + string(rune('a'+(i%26))) + string(rune('A'+((i/26)%26)))
}

func TestDecode_AggregateSizeBound(t *testing.T) {
	payload := validPayload()
	// Every field individually stays within its own bound, and every
	// collection stays within the 100-entry cap, but the combination
	// (100 findings x 2 max-size evidence pointers each, plus max-size
	// class_statement) blows the 256 KiB canonical aggregate bound.
	findings := make([]interface{}, 0, MaxCollectionItems)
	for i := 0; i < MaxCollectionItems; i++ {
		findings = append(findings, map[string]interface{}{
			"severity":        "minor",
			"class_key":       "class",
			"class_statement": strings.Repeat("b", SummaryMaxBytes),
			"fingerprint":     stringsRepeatUnique("agg-", i),
			"disposition":     "fixed",
			"evidence_pointers": []interface{}{
				strings.Repeat("p", PointerMaxBytes),
				strings.Repeat("q", PointerMaxBytes),
			},
		})
	}
	payload["findings"] = findings
	_, err := Decode(encode(t, payload))
	if err == nil {
		t.Fatalf("expected aggregate size bound to reject oversized result")
	}
}

func TestDecode_DuplicateFindingFingerprintRejected(t *testing.T) {
	payload := validPayload()
	finding := map[string]interface{}{
		"severity":        "minor",
		"class_key":       "class",
		"class_statement": "statement",
		"fingerprint":     "dup-fp",
		"disposition":     "fixed",
	}
	payload["findings"] = []interface{}{finding, finding}
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected duplicate finding fingerprint to be rejected")
	}
}

func TestDecode_DuplicateKickbackEntityKeyRejected(t *testing.T) {
	payload := validPayload()
	kickback := map[string]interface{}{
		"entity_key":    "T-E34-F05-002",
		"target_status": "todo",
		"reason":        "dup",
	}
	payload["kickbacks"] = []interface{}{kickback, kickback}
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected duplicate kickback entity_key to be rejected")
	}
}

func TestDecode_AliasedDuplicateKickbackEntityKeyRejected(t *testing.T) {
	// code-review round 12: raw string equality on entity_key let two
	// textually-different aliases of the SAME real entity ("T-E34-F05-002"
	// and its short form "E34-F05-002") both pass Validate()'s dedup check,
	// each getting its own independently-applied kickback transition on one
	// real entity within a single gate result. keys.KeyService.Normalize
	// folds the short/T-prefixed task alias pair to the same canonical form,
	// so this syntactic-layer dedup must reject it -- matching
	// ValidateRole's main-entity-kickback alias handling.
	payload := validPayload()
	payload["kickbacks"] = []interface{}{
		map[string]interface{}{
			"entity_key":    "T-E34-F05-002",
			"target_status": "todo",
			"reason":        "first",
		},
		map[string]interface{}{
			"entity_key":    "E34-F05-002",
			"target_status": "todo",
			"reason":        "second",
		},
	}
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected aliased duplicate kickback entity_key (short-form vs T-prefixed) to be rejected")
	}
}

func TestDecode_DuplicateSweepClassKeyRejected(t *testing.T) {
	payload := validPayload()
	sweep := map[string]interface{}{
		"class_key":       "dup-class",
		"class_statement": "statement",
		"searched_count":  1,
		"matching_count":  0,
		"guard": map[string]interface{}{
			"kind":                   "test",
			"implementation_pointer": "docs/g.md",
			"counterfactual_pointer": "docs/cf.md",
			"status":                 "verified",
		},
		"status": "open",
	}
	payload["remediation_sweeps"] = []interface{}{sweep, sweep}
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected duplicate remediation_sweeps class_key to be rejected")
	}
}

func TestDecode_DuplicateChangeImpactSourceRejected(t *testing.T) {
	payload := validPayload()
	impact := map[string]interface{}{
		"source_kind":    "question",
		"source_key":     "Q001",
		"source_pointer": "docs/q001.md",
		"change_summary": "summary",
		"status":         "accounted",
	}
	payload["change_impacts"] = []interface{}{impact, impact}
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected duplicate change_impacts source_kind+source_key to be rejected")
	}
}

func TestDecode_DuplicateJSONKeyRejected(t *testing.T) {
	// REQ-F-001 requires rejecting duplicate envelopes. Standard
	// encoding/json.Unmarshal silently accepts a duplicate object key
	// (last one wins), which lets a second, conflicting value for a field
	// like schema_version or summary slip past every other check in this
	// package undetected. This must be rejected at decode time regardless
	// of whether the duplicate values happen to also be individually valid.
	cases := map[string]string{
		"top-level duplicate summary":                  `{"schema_version":1,"summary":"first summary value","summary":"second summary value"}`,
		"top-level duplicate schema_version":           `{"schema_version":1,"schema_version":1,"summary":"a valid summary"}`,
		"duplicate gate_result-shaped key":             `{"schema_version":1,"summary":"ok","summary":"ok"}`,
		"duplicate key inside a nested finding object": `{"schema_version":1,"summary":"ok","findings":[{"severity":"minor","severity":"major","class_key":"k","class_statement":"s","fingerprint":"fp","disposition":"open"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(raw)); err == nil {
				t.Fatalf("expected duplicate JSON key to be rejected: %s", raw)
			}
		})
	}
}

func TestDecode_UnknownTopLevelFieldRejected(t *testing.T) {
	// This is the "second envelope"/alias case: gate, outcome, and evidence are
	// deliberately owned by the outer worker-control envelope and must never
	// be accepted as GateResult fields (architecture.md I-02).
	for _, alias := range []string{"gate", "outcome", "evidence", "recommended_outcome"} {
		t.Run(alias, func(t *testing.T) {
			payload := validPayload()
			payload[alias] = "unexpected"
			if _, err := Decode(encode(t, payload)); err == nil {
				t.Fatalf("expected unknown top-level field %q to be rejected", alias)
			}
		})
	}
}

func TestDecode_MalformedTopLevelShapeRejected(t *testing.T) {
	cases := map[string]string{
		"array":         `[]`,
		"string":        `"not an object"`,
		"number":        `42`,
		"truncated":     `{"schema_version":1,`,
		"trailing_data": `{"schema_version":1,"summary":"ok"} {}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(raw)); err == nil {
				t.Fatalf("expected malformed shape %q to be rejected", name)
			}
		})
	}
}

func TestDecode_ForbiddenCredentialContentRejected(t *testing.T) {
	cases := []string{
		"password=hunter2 was used to reproduce this",
		"Authorization: Bearer abc123",
		"api_key leaked in logs",
	}
	for _, summary := range cases {
		t.Run(summary, func(t *testing.T) {
			payload := validPayload()
			payload["summary"] = summary
			if _, err := Decode(encode(t, payload)); err == nil {
				t.Fatalf("expected forbidden content in summary to be rejected: %q", summary)
			}
		})
	}
}

func TestDecode_FindingDispositionEnum(t *testing.T) {
	valid := []string{"open", "fixed", "already_dispositioned", "severity_conflict", "not_reproducible"}
	for _, d := range valid {
		t.Run("valid_"+d, func(t *testing.T) {
			payload := validPayload()
			finding := payload["findings"].([]interface{})[0].(map[string]interface{})
			finding["disposition"] = d
			if d == "already_dispositioned" || d == "severity_conflict" {
				finding["disposition_pointer"] = "docs/decisions/DEC-001.md"
			}
			if _, err := Decode(encode(t, payload)); err != nil {
				t.Fatalf("expected disposition %q to be accepted, got %v", d, err)
			}
		})
	}
	t.Run("unknown_rejected", func(t *testing.T) {
		payload := validPayload()
		finding := payload["findings"].([]interface{})[0].(map[string]interface{})
		finding["disposition"] = "wontfix"
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected unknown disposition to be rejected")
		}
	})
}

func TestDecode_DispositionPointerRequiredForCitedDecisions(t *testing.T) {
	for _, d := range []string{"already_dispositioned", "severity_conflict"} {
		t.Run(d, func(t *testing.T) {
			payload := validPayload()
			finding := payload["findings"].([]interface{})[0].(map[string]interface{})
			finding["disposition"] = d
			delete(finding, "disposition_pointer")
			if _, err := Decode(encode(t, payload)); err == nil {
				t.Fatalf("expected missing disposition_pointer to be rejected for disposition %q", d)
			}
		})
	}
}

func TestDecode_RemediationSweepCompleteInvariant(t *testing.T) {
	baseSweep := func() map[string]interface{} {
		return map[string]interface{}{
			"class_key":       "class",
			"class_statement": "statement",
			"searched_count":  3,
			"matching_count":  2,
			"instances": []interface{}{
				map[string]interface{}{"fingerprint": "i1", "site_pointer": "docs/a.md", "disposition": "fixed"},
				map[string]interface{}{"fingerprint": "i2", "site_pointer": "docs/b.md", "disposition": "already_dispositioned"},
			},
			"fixed_count":         1,
			"dispositioned_count": 1,
			"open_count":          0,
			"guard": map[string]interface{}{
				"kind":                   "test",
				"implementation_pointer": "docs/g.md",
				"counterfactual_pointer": "docs/cf.md",
				"status":                 "verified",
			},
			"status": "complete",
		}
	}

	t.Run("valid complete sweep accepted", func(t *testing.T) {
		payload := validPayload()
		payload["remediation_sweeps"] = []interface{}{baseSweep()}
		if _, err := Decode(encode(t, payload)); err != nil {
			t.Fatalf("expected valid complete sweep to be accepted, got %v", err)
		}
	})

	t.Run("complete sweep with open_count nonzero rejected", func(t *testing.T) {
		payload := validPayload()
		sweep := baseSweep()
		sweep["open_count"] = 1
		sweep["matching_count"] = 3
		payload["remediation_sweeps"] = []interface{}{sweep}
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected complete sweep with open_count > 0 to be rejected")
		}
	})

	t.Run("counts not summing to matching_count rejected", func(t *testing.T) {
		payload := validPayload()
		sweep := baseSweep()
		sweep["matching_count"] = 5
		payload["remediation_sweeps"] = []interface{}{sweep}
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected mismatched counts to be rejected")
		}
	})

	t.Run("complete sweep with unverified guard rejected", func(t *testing.T) {
		payload := validPayload()
		sweep := baseSweep()
		sweep["guard"].(map[string]interface{})["status"] = "pending"
		payload["remediation_sweeps"] = []interface{}{sweep}
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected complete sweep with unverified guard to be rejected")
		}
	})

	t.Run("complete sweep with mismatched instance count rejected", func(t *testing.T) {
		payload := validPayload()
		sweep := baseSweep()
		sweep["instances"] = []interface{}{
			map[string]interface{}{"fingerprint": "i1", "site_pointer": "docs/a.md", "disposition": "fixed"},
		}
		payload["remediation_sweeps"] = []interface{}{sweep}
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected complete sweep with fewer instances than matching_count to be rejected")
		}
	})
}

func TestDecode_ChangeImpactStatusEnum(t *testing.T) {
	for _, status := range []string{"accounted", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			payload := validPayload()
			payload["change_impacts"] = []interface{}{
				map[string]interface{}{
					"source_kind":    "question",
					"source_key":     "Q001",
					"source_pointer": "docs/q001.md",
					"change_summary": "summary",
					"status":         status,
				},
			}
			if _, err := Decode(encode(t, payload)); err != nil {
				t.Fatalf("expected status %q to be accepted, got %v", status, err)
			}
		})
	}
	t.Run("unknown_rejected", func(t *testing.T) {
		payload := validPayload()
		payload["change_impacts"] = []interface{}{
			map[string]interface{}{
				"source_kind":    "question",
				"source_key":     "Q001",
				"source_pointer": "docs/q001.md",
				"change_summary": "summary",
				"status":         "maybe",
			},
		}
		if _, err := Decode(encode(t, payload)); err == nil {
			t.Fatalf("expected unknown change_impacts status to be rejected")
		}
	})
}

func TestDecode_NoKickbackReasonBoundedWhenPresent(t *testing.T) {
	payload := validPayload()
	payload["kickbacks"] = []interface{}{}
	payload["no_kickback_reason"] = strings.Repeat("c", SummaryMaxBytes+1)
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected oversized no_kickback_reason to be rejected")
	}
}
