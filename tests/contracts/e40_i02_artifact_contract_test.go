// Package contracts — I-02 metric collection and artifact schema contract
// test, shared between E40-F02 (producer, this task) and E40-F03 (consumer,
// not yet dispatched). Shape source: architecture.md#metric-collection-and-
// artifact-schema, field reference: bench/README.md "I-02 record schema
// field reference" (T-E40-F02-007's authoritative enumeration — confirmed
// against real bench/scripts/collect-run.sh output over the committed
// bench/scripts/testdata/run/* fixtures, not against spec.md's earlier and
// now-partly-superseded "Data model changes" table: the six manifest fields
// spec.md names — fixture_base_sha, corpus_schema_version, p2p_set,
// variant_bundle_sha256, shark_version, shark_binary_sha256 — are not
// emitted by the shipped collector (bench/scripts/collect-run.sh's manifest
// copy loop lists only item_id/item_type/variant_id/rep/timeout_cap_s/
// seeded_keys) and are deliberately absent from the schema this file
// validates and from both golden records).
//
// Naming/placement/technique follow e40_i01_corpus_contract_test.go
// (package contracts, TestTC00N_... naming, repo-root-relative os.ReadFile,
// and malformed-fixture-through-the-real-parser negative subtests) per this
// task's own Brownfield Context pointer.
package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// e40I02SupportedSchemaVersions is the set of schema_version values this
// validator accepts (AC-15 / spec.md "schema_version | string | Pinned").
var e40I02SupportedSchemaVersions = map[string]bool{
	"1.0": true,
}

// e40I02OutcomeValues is the closed six-value set: the five RunResult
// values copied unchanged, plus the harness-assigned "timeout"
// (ADR-F02-04).
var e40I02OutcomeValues = map[string]bool{
	"completed":        true,
	"paused":           true,
	"failed":           true,
	"already_terminal": true,
	"no_action":        true,
	"timeout":          true,
}

// e40I02ErrorKinds is the closed SEVEN-value set bench/README.md pins
// (`errors[].kind` field reference row) — not the six-value wording in this
// task's own AC text, which predates T-E40-F02-007's addition of
// crosscheck_resolution_error. README is the authoritative, more recent
// enumeration per this task's Notes for Agent.
var e40I02ErrorKinds = map[string]bool{
	"envelope_parse_error":        true,
	"stage_join_error":            true,
	"transcript_missing":          true,
	"crosscheck_disagreement":     true,
	"crosscheck_resolution_error": true,
	"postrun_check_aborted":       true,
	"usage_unavailable":           true,
}

// e40I02SourceValues is REQ-N-007's closed five-value set for every
// `sources.<family>` entry.
var e40I02SourceValues = map[string]bool{
	"runresult":  true,
	"transcript": true,
	"scratch_db": true,
	"postrun":    true,
	"liveness":   true,
}

// e40I02KnownTopLevelFields is the full I-02 record's top-level field
// inventory (bench/README.md's field reference table). test-plan.md's
// AC-14 decision table pins this as part of TC-001's own schema check
// ("TC-001's schema check additionally asserts no non-listed field appears
// in the golden record" — the concrete counter-example given is a
// wall-clock "collected_at" field, which REQ-N-004 forbids).
var e40I02KnownTopLevelFields = map[string]bool{
	"schema_version": true,
	"manifest":       true,
	"outcome":        true,
	"timeout_detail": true,
	"runresult":      true,
	"stages":         true,
	"timing":         true,
	"rejections":     true,
	"oracle":         true,
	"quality":        true,
	"loc":            true,
	"errors":         true,
	"sources":        true,
}

// ---------------------------------------------------------------------------
// Malformed-fixture JSON constants used only by the four negative subtests
// test-plan.md's Caller-Path Contract names for TC-001, never by the golden
// assertion itself — mirroring e40_i01_corpus_contract_test.go's
// e40BrokenManifestYAML technique, fed through the same real decode+validate
// path as the golden records.
// ---------------------------------------------------------------------------

const e40I02MissingSchemaVersionJSON = `{"manifest":{"item_id":"x","item_type":"task","rep":1,"run_key":"x::default::rep1","timeout_cap_s":60,"variant_id":"default"},"outcome":"timeout","sources":{"stalled_stage":"liveness"},"timeout_detail":{"action":"spawn_agent","agent_type":"developer","provider":"anthropic","source":"liveness_stream","stage_index":1,"status":"in_development"},"timing":{"harness_wall_ns":1000}}`

const e40I02UnsupportedSchemaVersionJSON = `{"schema_version":"9.9","manifest":{"item_id":"x","item_type":"task","rep":1,"run_key":"x::default::rep1","timeout_cap_s":60,"variant_id":"default"},"outcome":"timeout","sources":{"stalled_stage":"liveness"},"timeout_detail":{"action":"spawn_agent","agent_type":"developer","provider":"anthropic","source":"liveness_stream","stage_index":1,"status":"in_development"},"timing":{"harness_wall_ns":1000}}`

const e40I02InvalidOutcomeJSON = `{"schema_version":"1.0","manifest":{"item_id":"x","item_type":"task","rep":1,"run_key":"x::default::rep1","timeout_cap_s":60,"variant_id":"default"},"outcome":"bogus_outcome","runresult":{"final_status":"bogus_outcome","stages_completed":1,"total_duration_ns":1000},"timing":{"harness_wall_ns":1000}}`

const e40I02ErrorMissingDetailJSON = `{"schema_version":"1.0","manifest":{"item_id":"x","item_type":"task","rep":1,"run_key":"x::default::rep1","timeout_cap_s":60,"variant_id":"default"},"outcome":"completed","runresult":{"final_status":"completed","stages_completed":1,"total_duration_ns":1000},"errors":[{"kind":"usage_unavailable"}],"timing":{"harness_wall_ns":1000}}`

// TestTC001_I02ArtifactContract is I-02's shared-contract evidence
// (spec.md AC-15, test-plan.md TC-001). It reads the two committed golden
// records via os.ReadFile and the real encoding/json decoder (Caller-Path
// Contract: "must parse the real committed file"), validates each against
// the documented schema, and proves the validator actually bites via four
// malformed-fixture negative subtests fed through the same decode+validate
// path. CI-safe: no submodule, no scratch project, no network, no API
// spend (ADR-F02-10 / matching F01's ADR-F01-05).
func TestTC001_I02ArtifactContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	happyPath := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i02_golden_record.jsonl")
	timeoutPath := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i02_golden_record_timeout.jsonl")

	t.Run("happy_path_golden_record_is_schema_valid", func(t *testing.T) {
		rec := e40I02ReadOneLineRecord(t, happyPath)
		if errs := e40I02Validate(rec); len(errs) > 0 {
			t.Fatalf("happy-path golden record violates the I-02 schema:\n%s", strings.Join(errs, "\n"))
		}

		if outcome, _ := rec["outcome"].(string); outcome != "completed" {
			t.Errorf("happy-path golden outcome = %v, want \"completed\"", rec["outcome"])
		}
		manifest, _ := rec["manifest"].(map[string]interface{})
		if itemType, _ := manifest["item_type"].(string); itemType != "task" {
			t.Errorf("happy-path golden manifest.item_type = %v, want \"task\" (scope requires a task item)", manifest["item_type"])
		}

		stages, _ := rec["stages"].([]interface{})
		if len(stages) != 3 {
			t.Fatalf("happy-path golden stages length = %d, want 3 (2 spawn_agent + 1 advance_status per scope)", len(stages))
		}
		spawnCount, advanceCount := 0, 0
		for _, raw := range stages {
			stage, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			switch stage["action"] {
			case "spawn_agent":
				spawnCount++
			case "advance_status":
				advanceCount++
			}
		}
		if spawnCount != 2 || advanceCount != 1 {
			t.Errorf("happy-path golden stages = %d spawn_agent + %d advance_status, want 2 + 1", spawnCount, advanceCount)
		}

		// AC-02: all six metric families present with concrete values.
		for _, family := range []string{"manifest", "runresult", "stages", "timing", "rejections", "oracle", "quality", "loc"} {
			if _, ok := rec[family]; !ok {
				t.Errorf("happy-path golden missing metric family block %q (AC-02 requires all six)", family)
			}
		}
		if errsField, ok := rec["errors"].([]interface{}); !ok || len(errsField) != 0 {
			t.Errorf("happy-path golden errors[] = %v, want an empty array", rec["errors"])
		}
	})

	t.Run("timeout_golden_record_is_schema_valid", func(t *testing.T) {
		rec := e40I02ReadOneLineRecord(t, timeoutPath)
		if errs := e40I02Validate(rec); len(errs) > 0 {
			t.Fatalf("timeout golden record violates the I-02 schema:\n%s", strings.Join(errs, "\n"))
		}

		if outcome, _ := rec["outcome"].(string); outcome != "timeout" {
			t.Errorf("timeout golden outcome = %v, want \"timeout\"", rec["outcome"])
		}
		if _, ok := rec["runresult"]; ok {
			t.Errorf("timeout golden unexpectedly carries runresult (must be genuinely absent — no RunResult was ever delivered on a killed run)")
		}
		if _, ok := rec["timeout_detail"]; !ok {
			t.Error("timeout golden missing timeout_detail (required when outcome == \"timeout\")")
		}
		sources, _ := rec["sources"].(map[string]interface{})
		if _, ok := sources["stalled_stage"]; !ok {
			t.Error(`timeout golden missing sources.stalled_stage (bench/README.md pins this exact key, not "an equivalent key")`)
		}
	})

	// test-plan.md's four malformed-fixture negative subtests, each fed
	// through the same e40I02ReadOneLineRecord + e40I02Validate path the
	// golden assertions above use.
	t.Run("missing_schema_version_is_rejected", func(t *testing.T) {
		rec := e40I02DecodeFixture(t, e40I02MissingSchemaVersionJSON)
		errs := e40I02Validate(rec)
		if !e40ContainsErrorMatching(errs, "schema_version", "missing") {
			t.Fatalf("expected a schema_version missing error, got: %v", errs)
		}
	})

	t.Run("unsupported_schema_version_is_rejected", func(t *testing.T) {
		rec := e40I02DecodeFixture(t, e40I02UnsupportedSchemaVersionJSON)
		errs := e40I02Validate(rec)
		if !e40ContainsErrorMatching(errs, "schema_version", "unsupported") {
			t.Fatalf("expected a schema_version unsupported error, got: %v", errs)
		}
	})

	t.Run("outcome_outside_closed_set_is_rejected", func(t *testing.T) {
		rec := e40I02DecodeFixture(t, e40I02InvalidOutcomeJSON)
		errs := e40I02Validate(rec)
		if !e40ContainsErrorMatching(errs, "outcome", "closed six-value set") {
			t.Fatalf("expected an outcome-not-in-closed-set error, got: %v", errs)
		}
	})

	t.Run("errors_entry_missing_detail_is_rejected", func(t *testing.T) {
		rec := e40I02DecodeFixture(t, e40I02ErrorMissingDetailJSON)
		errs := e40I02Validate(rec)
		if !e40ContainsErrorMatching(errs, "errors[0].detail", "missing") {
			t.Fatalf("expected an errors[0].detail missing error, got: %v", errs)
		}
	})
}

// e40I02ReadOneLineRecord reads the committed golden record at path (which
// REQ-N-004 pins as exactly one JSONL line), asserts that shape, and decodes
// it through e40I02Decode.
func e40I02ReadOneLineRecord(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden record %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 || strings.TrimSpace(lines[0]) == "" {
		t.Fatalf("golden record %s is not exactly one JSONL line, got %d line(s)", path, len(lines))
	}
	return e40I02Decode(t, []byte(lines[0]))
}

// e40I02DecodeFixture decodes a hand-authored malformed-fixture JSON string
// constant through the same real decoder e40I02ReadOneLineRecord uses.
func e40I02DecodeFixture(t *testing.T, jsonText string) map[string]interface{} {
	t.Helper()
	return e40I02Decode(t, []byte(jsonText))
}

// e40I02Decode parses raw JSON bytes into a map[string]interface{} using
// json.Number for numeric values (via Decoder.UseNumber) instead of
// float64. This is what makes an "is this field int64-shaped" check a real
// assertion against the JSON literal's own shape rather than a tautology
// that float64(3) and float64(3.0) can't distinguish.
func e40I02Decode(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var rec map[string]interface{}
	if err := dec.Decode(&rec); err != nil {
		t.Fatalf("decode I-02 record JSON: %v\ninput: %s", err, data)
	}
	return rec
}

// ---------------------------------------------------------------------------
// e40I02Validate and its per-block helpers: the schema validator itself.
// Each helper returns one description per violation; e40I02Validate
// concatenates them. No panics on malformed input — every type assertion is
// the comma-ok form, so a wrong-shaped fixture produces a named violation
// instead of a test crash.
// ---------------------------------------------------------------------------

func e40I02Validate(rec map[string]interface{}) []string {
	var errs []string

	for k := range rec {
		if !e40I02KnownTopLevelFields[k] {
			errs = append(errs, fmt.Sprintf("%s: unknown top-level field, not part of the I-02 schema", k))
		}
	}

	outcome := ""
	if sv, ok := rec["schema_version"]; !ok {
		errs = append(errs, "schema_version: missing")
	} else if s, ok := sv.(string); !ok {
		errs = append(errs, "schema_version: not a string")
	} else if !e40I02SupportedSchemaVersions[s] {
		errs = append(errs, fmt.Sprintf("schema_version: unsupported value %q", s))
	}

	if ov, ok := rec["outcome"]; !ok {
		errs = append(errs, "outcome: missing")
	} else if s, ok := ov.(string); !ok {
		errs = append(errs, "outcome: not a string")
	} else {
		outcome = s
		if !e40I02OutcomeValues[s] {
			errs = append(errs, fmt.Sprintf("outcome: value %q not in closed six-value set", s))
		}
	}

	errs = append(errs, e40I02ValidateErrors(rec)...)
	errs = append(errs, e40I02ValidateSources(rec)...)
	errs = append(errs, e40I02ValidateOutcomeConditional(rec, outcome)...)
	errs = append(errs, e40I02ValidateManifest(rec)...)
	errs = append(errs, e40I02ValidateStages(rec)...)
	errs = append(errs, e40I02ValidateRejections(rec)...)
	errs = append(errs, e40I02ValidateOracle(rec)...)
	errs = append(errs, e40I02ValidateQuality(rec)...)
	errs = append(errs, e40I02ValidateLOC(rec)...)
	errs = append(errs, e40I02ValidateTiming(rec)...)

	return errs
}

func e40I02ValidateErrors(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["errors"]
	if !ok {
		return errs
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return append(errs, fmt.Sprintf("errors: not an array (got %T)", raw))
	}
	for i, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Sprintf("errors[%d]: not an object", i))
			continue
		}
		if kv, ok := obj["kind"]; !ok {
			errs = append(errs, fmt.Sprintf("errors[%d].kind: missing", i))
		} else if s, ok := kv.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("errors[%d].kind: empty or not a string", i))
		} else if !e40I02ErrorKinds[s] {
			errs = append(errs, fmt.Sprintf("errors[%d].kind: value %q not in closed seven-value set", i, s))
		}
		if dv, ok := obj["detail"]; !ok {
			errs = append(errs, fmt.Sprintf("errors[%d].detail: missing", i))
		} else if s, ok := dv.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("errors[%d].detail: empty or not a string", i))
		}
		if si, ok := obj["stage_index"]; ok {
			if n, ok := si.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, fmt.Sprintf("errors[%d].stage_index: present but not an integer", i))
			}
		}
		if p, ok := obj["path"]; ok {
			if _, ok := p.(string); !ok {
				errs = append(errs, fmt.Sprintf("errors[%d].path: present but not a string", i))
			}
		}
	}
	return errs
}

func e40I02ValidateSources(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["sources"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, fmt.Sprintf("sources: not an object (got %T)", raw))
	}
	for k, v := range obj {
		s, ok := v.(string)
		if !ok {
			errs = append(errs, fmt.Sprintf("sources.%s: not a string", k))
			continue
		}
		if !e40I02SourceValues[s] {
			errs = append(errs, fmt.Sprintf("sources.%s: value %q not in closed five-value set", k, s))
		}
	}
	return errs
}

// e40I02ValidateOutcomeConditional enforces the test-plan.md "Schema
// completeness gap" resolution: runresult.* is required exactly when
// outcome != "timeout"; timeout_detail is required exactly when
// outcome == "timeout" — never both, never neither.
func e40I02ValidateOutcomeConditional(rec map[string]interface{}, outcome string) []string {
	var errs []string
	_, hasRunresult := rec["runresult"]
	tdRaw, hasTimeoutDetail := rec["timeout_detail"]

	if outcome == "timeout" {
		if hasRunresult {
			errs = append(errs, `runresult: present but forbidden when outcome == "timeout" (no RunResult is ever delivered on a killed run)`)
		}
		if !hasTimeoutDetail {
			errs = append(errs, `timeout_detail: missing but required when outcome == "timeout"`)
		} else if tdObj, ok := tdRaw.(map[string]interface{}); !ok {
			errs = append(errs, "timeout_detail: not an object")
		} else {
			for _, key := range []string{"stage_index", "status", "action", "agent_type", "provider", "source"} {
				if _, ok := tdObj[key]; !ok {
					errs = append(errs, fmt.Sprintf("timeout_detail.%s: missing", key))
				}
			}
			if sv, ok := tdObj["status"]; ok {
				if s, ok := sv.(string); !ok || strings.TrimSpace(s) == "" {
					errs = append(errs, "timeout_detail.status: must be a non-empty string")
				}
			}
			if sv, ok := tdObj["source"]; ok {
				if s, ok := sv.(string); !ok || strings.TrimSpace(s) == "" {
					errs = append(errs, "timeout_detail.source: must be a non-empty string")
				}
			}
		}
		// bench/README.md: "This is not 'an equivalent key' -- collect-run.sh
		// emits exactly stalled_stage." Pinned here since this hand-authored
		// fixture is the only place a mismatch against that pin would
		// surface (this task's own Notes for Agent).
		if srcRaw, ok := rec["sources"]; ok {
			if srcObj, ok := srcRaw.(map[string]interface{}); ok {
				if _, ok := srcObj["stalled_stage"]; !ok {
					errs = append(errs, `sources.stalled_stage: missing on a timeout record (README pins this exact literal key)`)
				}
			}
		}
	} else {
		if !hasRunresult {
			errs = append(errs, `runresult: missing but required when outcome != "timeout"`)
		} else if rrObj, ok := rec["runresult"].(map[string]interface{}); !ok {
			errs = append(errs, "runresult: not an object")
		} else {
			if fs, ok := rrObj["final_status"]; !ok {
				errs = append(errs, "runresult.final_status: missing")
			} else if _, ok := fs.(string); !ok {
				errs = append(errs, "runresult.final_status: not a string")
			}
			if sc, ok := rrObj["stages_completed"]; !ok {
				errs = append(errs, "runresult.stages_completed: missing")
			} else if n, ok := sc.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, "runresult.stages_completed: not an integer")
			}
			if td, ok := rrObj["total_duration_ns"]; !ok {
				errs = append(errs, "runresult.total_duration_ns: missing")
			} else if n, ok := td.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, "runresult.total_duration_ns: not an int64-shaped integer")
			}
		}
		if hasTimeoutDetail {
			errs = append(errs, `timeout_detail: present but forbidden when outcome != "timeout" (never a zero-valued object)`)
		}
	}
	return errs
}

func e40I02ValidateManifest(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["manifest"]
	if !ok {
		return append(errs, "manifest: missing")
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "manifest: not an object")
	}

	for _, key := range []string{"item_id", "item_type", "variant_id", "run_key"} {
		v, ok := obj[key]
		if !ok {
			errs = append(errs, fmt.Sprintf("manifest.%s: missing", key))
			continue
		}
		if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("manifest.%s: empty or not a string", key))
		}
	}
	if it, ok := obj["item_type"].(string); ok && it != "task" && it != "bug" {
		errs = append(errs, fmt.Sprintf(`manifest.item_type: value %q, want "task" or "bug"`, it))
	}

	for _, key := range []string{"rep", "timeout_cap_s"} {
		v, ok := obj[key]
		if !ok {
			errs = append(errs, fmt.Sprintf("manifest.%s: missing", key))
			continue
		}
		if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
			errs = append(errs, fmt.Sprintf("manifest.%s: not an integer", key))
		}
	}

	// REQ-F-018: run_key is the derived <item_id>::<variant_id>::rep<rep> key.
	if id, ok := obj["item_id"].(string); ok {
		if variant, ok := obj["variant_id"].(string); ok {
			if repN, ok := obj["rep"].(json.Number); ok {
				want := fmt.Sprintf("%s::%s::rep%s", id, variant, repN.String())
				if rk, ok := obj["run_key"].(string); ok && rk != want {
					errs = append(errs, fmt.Sprintf("manifest.run_key: %q, want %q (REQ-F-018)", rk, want))
				}
			}
		}
	}

	// README: model_ids and model_id_source are present together or absent
	// together; the only valid non-absent model_id_source value is
	// "modelUsage" -- the shipped parser never falls back to a top-level
	// "model" field.
	_, hasModelIDs := obj["model_ids"]
	modelSourceRaw, hasModelSource := obj["model_id_source"]
	if hasModelIDs != hasModelSource {
		errs = append(errs, "manifest.model_ids/model_id_source: must both be present or both be absent")
	}
	if hasModelIDs {
		if arr, ok := obj["model_ids"].([]interface{}); !ok {
			errs = append(errs, "manifest.model_ids: not an array")
		} else {
			for i, item := range arr {
				if _, ok := item.(string); !ok {
					errs = append(errs, fmt.Sprintf("manifest.model_ids[%d]: not a string", i))
				}
			}
		}
	}
	if hasModelSource {
		if s, ok := modelSourceRaw.(string); !ok {
			errs = append(errs, "manifest.model_id_source: not a string")
		} else if s != "modelUsage" {
			errs = append(errs, fmt.Sprintf(`manifest.model_id_source: value %q, want "modelUsage" (never "model" -- the shipped parser has no model fallback, T-E40-F02-007)`, s))
		}
	}

	return errs
}

func e40I02ValidateStages(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["stages"]
	if !ok {
		return errs
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return append(errs, "stages: not an array")
	}
	for i, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Sprintf("stages[%d]: not an object", i))
			continue
		}
		for _, key := range []string{"index", "duration_ns", "exit_code"} {
			v, ok := obj[key]
			if !ok {
				errs = append(errs, fmt.Sprintf("stages[%d].%s: missing", i, key))
				continue
			}
			if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, fmt.Sprintf("stages[%d].%s: not an integer", i, key))
			}
		}
		for _, key := range []string{"status", "action"} {
			v, ok := obj[key]
			if !ok {
				errs = append(errs, fmt.Sprintf("stages[%d].%s: missing", i, key))
				continue
			}
			if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
				errs = append(errs, fmt.Sprintf("stages[%d].%s: empty or not a string", i, key))
			}
		}
		if usageRaw, ok := obj["usage"]; ok {
			errs = append(errs, e40I02ValidateUsage(i, usageRaw)...)
		}
	}
	return errs
}

func e40I02ValidateUsage(stageIdx int, raw interface{}) []string {
	var errs []string
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, fmt.Sprintf("stages[%d].usage: not an object", stageIdx))
	}
	for _, key := range []string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "duration_api_ms", "num_turns"} {
		if v, ok := obj[key]; ok {
			if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, fmt.Sprintf("stages[%d].usage.%s: not an integer", stageIdx, key))
			}
		}
	}
	if v, ok := obj["total_cost_usd"]; ok {
		if _, ok := v.(json.Number); !ok {
			errs = append(errs, fmt.Sprintf("stages[%d].usage.total_cost_usd: not a number", stageIdx))
		}
	}
	if v, ok := obj["model_ids"]; ok {
		arr, ok := v.([]interface{})
		if !ok {
			errs = append(errs, fmt.Sprintf("stages[%d].usage.model_ids: not an array", stageIdx))
		} else {
			for i, item := range arr {
				if _, ok := item.(string); !ok {
					errs = append(errs, fmt.Sprintf("stages[%d].usage.model_ids[%d]: not a string", stageIdx, i))
				}
			}
		}
	}
	return errs
}

func e40I02ValidateRejections(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["rejections"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "rejections: not an object")
	}
	if v, ok := obj["by_gate"]; ok {
		if _, ok := v.(map[string]interface{}); !ok {
			errs = append(errs, "rejections.by_gate: not an object")
		}
	}
	if v, ok := obj["rework_loops"]; ok {
		if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
			errs = append(errs, "rejections.rework_loops: not an integer")
		}
	}
	if v, ok := obj["crosscheck"]; ok {
		ccObj, ok := v.(map[string]interface{})
		if !ok {
			errs = append(errs, "rejections.crosscheck: not an object")
		} else {
			for _, key := range []string{"entity_history_backward_transitions", "work_session_outcomes"} {
				cv, ok := ccObj[key]
				if !ok {
					errs = append(errs, fmt.Sprintf("rejections.crosscheck.%s: missing", key))
					continue
				}
				if n, ok := cv.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
					errs = append(errs, fmt.Sprintf("rejections.crosscheck.%s: not an integer", key))
				}
			}
			if av, ok := ccObj["agrees"]; !ok {
				errs = append(errs, "rejections.crosscheck.agrees: missing")
			} else if _, ok := av.(bool); !ok {
				errs = append(errs, "rejections.crosscheck.agrees: not a bool")
			}
		}
	}
	return errs
}

func e40I02ValidateOracle(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["oracle"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "oracle: not an object")
	}
	for _, key := range []string{"f2p_resolved", "repro_confirmed"} {
		if v, ok := obj[key]; ok && !e40I02IsBoolOrNull(v) {
			errs = append(errs, fmt.Sprintf("oracle.%s: not bool or null", key))
		}
	}
	for _, key := range []string{"p2p_regressions_count", "removed_count"} {
		if v, ok := obj[key]; ok {
			if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, fmt.Sprintf("oracle.%s: not an integer", key))
			}
		}
	}
	for _, key := range []string{"p2p_regressions", "removed"} {
		if v, ok := obj[key]; ok {
			if _, ok := v.([]interface{}); !ok {
				errs = append(errs, fmt.Sprintf("oracle.%s: not an array", key))
			}
		}
	}
	return errs
}

func e40I02ValidateQuality(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["quality"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "quality: not an object")
	}
	for _, key := range []string{"fmt_clean", "vet_ok", "tests_pass"} {
		if v, ok := obj[key]; ok && !e40I02IsBoolOrNull(v) {
			errs = append(errs, fmt.Sprintf("quality.%s: not bool or null", key))
		}
	}
	if v, ok := obj["lint_new_issues"]; ok {
		if _, ok := v.([]interface{}); !ok {
			errs = append(errs, "quality.lint_new_issues: not an array")
		}
	}
	if v, ok := obj["lint_new_issues_count"]; ok {
		if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
			errs = append(errs, "quality.lint_new_issues_count: not an integer")
		}
	}
	if v, ok := obj["toolchain_guard"]; ok {
		if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, "quality.toolchain_guard: empty or not a string")
		}
	}
	return errs
}

func e40I02ValidateLOC(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["loc"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "loc: not an object")
	}
	for _, key := range []string{"prod_added", "prod_deleted", "test_added", "test_deleted", "files_touched"} {
		if v, ok := obj[key]; ok {
			if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
				errs = append(errs, fmt.Sprintf("loc.%s: not an integer", key))
			}
		}
	}
	return errs
}

func e40I02ValidateTiming(rec map[string]interface{}) []string {
	var errs []string
	raw, ok := rec["timing"]
	if !ok {
		return errs
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return append(errs, "timing: not an object")
	}
	v, ok := obj["harness_wall_ns"]
	if !ok {
		return append(errs, "timing.harness_wall_ns: missing")
	}
	if n, ok := v.(json.Number); !ok || !e40I02IsIntegerLiteral(n) {
		errs = append(errs, "timing.harness_wall_ns: not an int64-shaped integer")
	}
	return errs
}

// e40I02IsIntegerLiteral reports whether n's own JSON literal text (not its
// mathematical value) is integer-shaped -- no ".", "e", or "E" -- so
// duration_ns: 3.0 (a float literal that happens to be a whole number)
// fails this check even though 3.0 == 3 numerically. This is what makes
// "duration_ns is int64" a real assertion against the wire shape rather
// than a tautology float64 comparison can't distinguish.
func e40I02IsIntegerLiteral(n json.Number) bool {
	return !strings.ContainsAny(n.String(), ".eE")
}

// e40I02IsBoolOrNull reports whether v is Go's decoded form of a JSON `true`,
// `false`, or `null` -- the bool|null type REQ-F-016 pins for the
// quality.*/oracle.* gate fields (null means the gate could not be
// executed, never a silent pass).
func e40I02IsBoolOrNull(v interface{}) bool {
	if v == nil {
		return true
	}
	_, ok := v.(bool)
	return ok
}
