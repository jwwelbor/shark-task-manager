// TC-052 verifies the I-06 product-design replay contract E40-F07 produces
// for E40-F08 (spec.md "Produces: I-06"). Per REQ-NF-003/ADR-F07-10, this
// validator reads only in-repo artifacts -- bench/replay/i06-schema.yaml and
// tests/contracts/testdata/e40_i06/{valid,invalid}/** -- and never a
// populated fixture submodule, mirroring
// TestTC042_I05StageEvidenceContract's own submodule-independence
// discipline (bench/fixture-py and bench/fixture-repo are never read here).
//
// This task (T-E40-F07-001) covers AC-001's document-kind core only:
// REQ-F-001 (I-06 is two distinct, unambiguous document kinds) and
// REQ-F-002 (the replay bundle's schema_version/bundle_version/
// scenario_binding/entries[] field inventory). REQ-F-003's entry_digest
// recomputation (AC-002), REQ-F-010/REQ-F-011's artifact-record and
// interaction-proxy checks (AC-007, AC-008), and REQ-F-017/REQ-F-018/
// REQ-F-019's terminal-outcome mapping, malformed-field matrix, and
// vocabulary agreement (AC-013, AC-014, AC-015) are later tasks' extensions
// of this same test file (T-E40-F07-002, -008, -009).
package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// e40I06SupportedSchemaVersion is the i06-schema.yaml / bundle / result
// schema_version this validator understands (mirrors
// e40I05SupportedSchemaVersion).
const e40I06SupportedSchemaVersion = "1.0"

// e40I06DocumentKindSpec is one bench/replay/i06-schema.yaml
// document_kinds entry: the top-level field inventory a document of that
// kind MUST carry (REQ-F-001/REQ-F-002), plus the bundle-only
// entry-level field inventory.
type e40I06DocumentKindSpec struct {
	Description         string   `yaml:"description"`
	RequiredFields      []string `yaml:"required_fields"`
	EntryRequiredFields []string `yaml:"entry_required_fields"`
}

// e40I06Schema decodes bench/replay/i06-schema.yaml -- REQ-F-018's single
// machine-readable owner of I-06's schema_version and every closed
// vocabulary. The Go validator below reads every vocabulary from this
// struct; none of the vocab value lists are duplicated as Go constants.
type e40I06Schema struct {
	SchemaVersion                           string                            `yaml:"schema_version"`
	DocumentKinds                           map[string]e40I06DocumentKindSpec `yaml:"document_kinds"`
	Stage                                   []string                          `yaml:"stage"`
	RequestKind                             []string                          `yaml:"request_kind"`
	ArtifactType                            []string                          `yaml:"artifact_type"`
	EdgeKind                                []string                          `yaml:"edge_kind"`
	ReplayedInteractionProxiesDiscriminator string                            `yaml:"replayed_interaction_proxies_discriminator"`
	ReplayedInteractionProxiesFields        []string                          `yaml:"replayed_interaction_proxies_fields"`
	ReplayWaitCategory                      string                            `yaml:"replay_wait_category"`
	TerminalOutcome                         []string                          `yaml:"terminal_outcome"`
	ErrorKind                               []string                          `yaml:"error_kind"`
}

// TestTC052_I06ProductDesignReplayContract is the shared contract test
// E40-F08 must reuse verbatim (spec.md Cross-feature interactions: "no
// twin test is created").
func TestTC052_I06ProductDesignReplayContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	schemaPath := filepath.Join(repoRoot, "bench", "replay", "i06-schema.yaml")
	testdataRoot := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i06")

	schema := e40I06ReadSchema(t, schemaPath)

	// AC-T1 (self-check): the schema file itself declares a supported
	// schema_version and both document kinds with a non-empty field
	// inventory, plus the stage/request_kind vocab this task's checks
	// read from it rather than a Go constant.
	t.Run("schema_self_check", func(t *testing.T) {
		if schema.SchemaVersion != e40I06SupportedSchemaVersion {
			t.Errorf("i06-schema.yaml schema_version = %q, want %q", schema.SchemaVersion, e40I06SupportedSchemaVersion)
		}
		for _, kind := range []string{"bundle", "result"} {
			spec, ok := schema.DocumentKinds[kind]
			if !ok {
				t.Fatalf("i06-schema.yaml document_kinds missing %q", kind)
			}
			if len(spec.RequiredFields) == 0 {
				t.Errorf("i06-schema.yaml document_kinds.%s.required_fields is empty", kind)
			}
		}
		if len(schema.DocumentKinds["bundle"].EntryRequiredFields) == 0 {
			t.Error("i06-schema.yaml document_kinds.bundle.entry_required_fields is empty")
		}
		wantStages := []string{"D01", "D02", "D03", "D04", "D05"}
		if !e40StringSlicesEqual(schema.Stage, wantStages) {
			t.Errorf("i06-schema.yaml stage = %v, want %v", schema.Stage, wantStages)
		}
		wantRequestKinds := []string{"human_question", "research_query"}
		if !e40StringSlicesEqual(schema.RequestKind, wantRequestKinds) {
			t.Errorf("i06-schema.yaml request_kind = %v, want %v", schema.RequestKind, wantRequestKinds)
		}
	})

	// AC-T2/AC-001: every valid bundle fixture's schema_version and
	// REQ-F-002 field inventory (bundle_version, scenario_binding,
	// entries[] with each entry's full field set) is present and
	// well-typed.
	t.Run("valid_bundle_field_inventory", func(t *testing.T) {
		for _, name := range []string{"bundle-minimal"} {
			name := name
			t.Run(name, func(t *testing.T) {
				dir := filepath.Join(testdataRoot, "valid")
				doc := e40I06ReadDocument(t, filepath.Join(dir, name+".json"))
				errs := e40I06ValidateDocument(doc, "bundle", schema)
				if len(errs) != 0 {
					t.Errorf("valid bundle fixture %s failed, want zero errors:\n%s", name, strings.Join(errs, "\n"))
				}
			})
		}
	})

	// AC-001: the valid result fixture declares schema_version and every
	// REQ-F-001 top-level field this task checks (deep result validation
	// -- artifact records, interaction proxies, terminal-outcome mapping
	// -- is added by T-E40-F07-008/-009).
	t.Run("valid_result_top_level_fields", func(t *testing.T) {
		dir := filepath.Join(testdataRoot, "valid")
		doc := e40I06ReadDocument(t, filepath.Join(dir, "result-minimal.json"))
		errs := e40I06ValidateDocument(doc, "result", schema)
		if len(errs) != 0 {
			t.Errorf("valid result fixture failed, want zero errors:\n%s", strings.Join(errs, "\n"))
		}
	})

	// AC-T3/AC-001: a result document supplied where a bundle is
	// expected, or the reverse, is rejected naming the expected kind
	// rather than silently half-validating (i.e. the validator must stop
	// at the kind mismatch, not fall through to a pile of unrelated
	// missing-field errors).
	t.Run("document_kind_mismatch", func(t *testing.T) {
		t.Run("result_supplied_where_bundle_expected", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", "result-as-bundle.json"))
			errs := e40I06ValidateDocument(doc, "bundle", schema)
			if len(errs) == 0 {
				t.Fatal("expected a document_kind_mismatch violation, got none")
			}
			// Assert the *ordered* phrase, not just that "bundle" appears
			// anywhere in the message: the reverse-direction message also
			// contains the substring "bundle" (as the detected kind), so a
			// validator that swapped expected/got would still pass a bare
			// substring check on "bundle" alone.
			if !e40ContainsErrorMatching(errs, "document_kind_mismatch", `expected "bundle"`) {
				t.Errorf("expected a document_kind_mismatch error naming \"bundle\" as the *expected* kind, got:\n%s", strings.Join(errs, "\n"))
			}
			if len(errs) != 1 {
				t.Errorf("expected exactly one kind-mismatch error (no half-validation fallthrough), got %d:\n%s", len(errs), strings.Join(errs, "\n"))
			}
		})

		t.Run("bundle_supplied_where_result_expected", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", "bundle-as-result.json"))
			errs := e40I06ValidateDocument(doc, "result", schema)
			if len(errs) == 0 {
				t.Fatal("expected a document_kind_mismatch violation, got none")
			}
			if !e40ContainsErrorMatching(errs, "document_kind_mismatch", `expected "result"`) {
				t.Errorf("expected a document_kind_mismatch error naming \"result\" as the *expected* kind, got:\n%s", strings.Join(errs, "\n"))
			}
			if len(errs) != 1 {
				t.Errorf("expected exactly one kind-mismatch error (no half-validation fallthrough), got %d:\n%s", len(errs), strings.Join(errs, "\n"))
			}
		})
	})

	// AC-001/AC-014 (this task's slice): named malformed-bundle cases.
	t.Run("malformed_bundle_cases", func(t *testing.T) {
		cases := []struct {
			name    string
			file    string
			wantAny []string
		}{
			{"unsupported_schema_version", "unsupported-schema-version.json", []string{"schema_version_unsupported"}},
			{"missing_bundle_version", "missing-bundle-version.json", []string{"field_missing", "bundle_version"}},
			{"duplicate_ordinal", "duplicate-ordinal.json", []string{"duplicate_ordinal", "D01"}},
			{"unknown_stage", "unknown-stage.json", []string{"D09"}},
			{"unknown_request_kind", "unknown-request-kind.json", []string{"chit_chat"}},
			{"bad_response_shape", "bad-response-shape.json", []string{"field_malformed", "response"}},
			{"entry_missing_field", "entry-missing-field.json", []string{"field_missing", "topic_key"}},
		}
		for _, c := range cases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", c.file))
				errs := e40I06ValidateDocument(doc, "bundle", schema)
				if len(errs) == 0 {
					t.Fatalf("case %s: expected validation errors, got none", c.name)
				}
				for _, want := range c.wantAny {
					if !e40ContainsErrorMatching(errs, want) {
						t.Errorf("case %s: expected an error naming %q, got:\n%s", c.name, want, strings.Join(errs, "\n"))
					}
				}
			})
		}
	})
}

// e40I06ReadSchema reads and parses a committed i06-schema.yaml file. Real
// filesystem read, per this task's Caller-Path Contract (TC-052 row): a
// validator reading a hand-built in-memory manifest would stay green even
// if the real committed schema were malformed.
func e40I06ReadSchema(t *testing.T, path string) *e40I06Schema {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read i06 schema %s: %v", path, err)
	}
	var schema e40I06Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse i06 schema %s: %v", path, err)
	}
	return &schema
}

// e40I06ReadDocument reads and JSON-decodes one committed bundle or result
// fixture file. Real filesystem read, per this task's Caller-Path Contract:
// a validator reading a hand-built in-memory manifest would stay green even
// if a real committed fixture were malformed.
func e40I06ReadDocument(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

// e40I06DetectKind scores doc's top-level keys against each declared
// document kind's required_fields and returns the better-matching kind, or
// "" if the two scores tie (including the zero/zero case).
func e40I06DetectKind(doc map[string]interface{}, schema *e40I06Schema) string {
	var bestKind string
	bestScore := -1
	tie := false
	for kind, spec := range schema.DocumentKinds {
		score := 0
		for _, field := range spec.RequiredFields {
			if _, ok := doc[field]; ok {
				score++
			}
		}
		switch {
		case score > bestScore:
			bestScore = score
			bestKind = kind
			tie = false
		case score == bestScore:
			tie = true
		}
	}
	if tie || bestScore <= 0 {
		return ""
	}
	return bestKind
}

// e40I06ValidateDocument applies REQ-F-001/REQ-F-002's field inventory to
// one decoded document, expected to be of expectedKind ("bundle" or
// "result"). It returns one description per violation, each naming the
// offending field or vocabulary value. If doc is detected as the *other*
// declared kind, validation stops at a single document_kind_mismatch
// error (AC-T3: "rejected naming the expected kind rather than silently
// half-validating") instead of falling through to unrelated field checks.
func e40I06ValidateDocument(doc map[string]interface{}, expectedKind string, schema *e40I06Schema) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	spec, ok := schema.DocumentKinds[expectedKind]
	if !ok {
		addf("i06-schema.yaml declares no document kind %q", expectedKind)
		return errs
	}

	if detected := e40I06DetectKind(doc, schema); detected != "" && detected != expectedKind {
		addf("document_kind_mismatch: expected %q document, got %q", expectedKind, detected)
		return errs
	}

	rawVersion, hasVersion := doc["schema_version"]
	version, versionIsString := rawVersion.(string)
	if !hasVersion {
		addf("field_missing: schema_version")
	} else if !versionIsString {
		addf("field_malformed: schema_version must be a string")
	} else if version != e40I06SupportedSchemaVersion {
		addf("schema_version_unsupported: %q (want %q)", version, e40I06SupportedSchemaVersion)
	}

	for _, field := range spec.RequiredFields {
		if field == "schema_version" {
			continue
		}
		if _, present := doc[field]; !present {
			addf("field_missing: %s", field)
		}
	}

	if expectedKind == "bundle" {
		errs = append(errs, e40I06ValidateBundleBody(doc, schema)...)
	}

	return errs
}

// e40I06ValidateBundleBody validates REQ-F-002's bundle-specific field
// inventory: bundle_version's type, scenario_binding's shape, and every
// entries[] element's full field set, stage/request_kind vocabulary
// membership, response polymorphism, and per-stage ordinal uniqueness.
func e40I06ValidateBundleBody(doc map[string]interface{}, schema *e40I06Schema) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	if raw, present := doc["bundle_version"]; present {
		if _, isString := raw.(string); !isString {
			addf("field_malformed: bundle_version must be a string")
		}
	}

	if raw, present := doc["scenario_binding"]; present {
		binding, isMap := raw.(map[string]interface{})
		if !isMap {
			addf("field_malformed: scenario_binding must be an object")
		} else {
			if _, ok := binding["scenario_id"]; !ok {
				addf("field_missing: scenario_binding.scenario_id")
			}
			if _, ok := binding["scenario_version"]; !ok {
				addf("field_missing: scenario_binding.scenario_version")
			}
		}
	}

	entrySpec, ok := schema.DocumentKinds["bundle"]
	if !ok {
		return errs
	}
	stageSet := e40I06StringSet(schema.Stage)
	requestKindSet := e40I06StringSet(schema.RequestKind)

	rawEntries, present := doc["entries"]
	if !present {
		// Already reported as a missing top-level field by the caller.
		return errs
	}
	entryList, isList := rawEntries.([]interface{})
	if !isList {
		addf("field_malformed: entries must be an array")
		return errs
	}

	seenOrdinalsByStage := map[string]map[float64]bool{}
	for i, rawEntry := range entryList {
		entry, isMap := rawEntry.(map[string]interface{})
		if !isMap {
			addf("field_malformed: entries[%d] must be an object", i)
			continue
		}

		for _, field := range entrySpec.EntryRequiredFields {
			if _, present := entry[field]; !present {
				addf("field_missing: entries[%d].%s", i, field)
			}
		}

		// Stage vocabulary membership and per-stage ordinal uniqueness are
		// deliberately independent checks: an entry with a missing or
		// unknown stage still has its ordinal type-checked and grouped
		// (keyed on its raw stage value) for duplicate detection, so a
		// bad stage value can never mask a duplicate-ordinal violation.
		var stageKey string
		hasStageKey := false
		if rawStage, present := entry["stage"]; present {
			stage, isString := rawStage.(string)
			if !isString {
				addf("field_malformed: entries[%d].stage must be a string", i)
			} else {
				stageKey = stage
				hasStageKey = true
				if !stageSet[stage] {
					addf("vocabulary_value_unknown: entries[%d].stage %q is not one of %v", i, stage, schema.Stage)
				}
			}
		}

		if hasStageKey {
			if rawOrdinal, hasOrdinal := entry["ordinal"]; hasOrdinal {
				if ordinal, isNum := rawOrdinal.(float64); isNum {
					if seenOrdinalsByStage[stageKey] == nil {
						seenOrdinalsByStage[stageKey] = map[float64]bool{}
					}
					if seenOrdinalsByStage[stageKey][ordinal] {
						addf("duplicate_ordinal: stage %s ordinal %v is used by more than one entry", stageKey, ordinal)
					}
					seenOrdinalsByStage[stageKey][ordinal] = true
				} else {
					addf("field_malformed: entries[%d].ordinal must be a number", i)
				}
			}
		}

		if rawKind, present := entry["request_kind"]; present {
			kind, isString := rawKind.(string)
			if !isString {
				addf("field_malformed: entries[%d].request_kind must be a string", i)
			} else if !requestKindSet[kind] {
				addf("vocabulary_value_unknown: entries[%d].request_kind %q is not one of %v", i, kind, schema.RequestKind)
			}
		}

		if rawResponse, present := entry["response"]; present {
			switch v := rawResponse.(type) {
			case string:
				// Inline response text -- well-typed.
			case map[string]interface{}:
				if _, ok := v["path"].(string); !ok {
					addf("field_malformed: entries[%d].response.path must be a string", i)
				}
				if _, ok := v["digest"].(string); !ok {
					addf("field_malformed: entries[%d].response.digest must be a string", i)
				}
			default:
				addf("field_malformed: entries[%d].response must be a string or a {path, digest} object", i)
			}
		}
	}

	return errs
}

func e40I06StringSet(values []string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

func e40StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
