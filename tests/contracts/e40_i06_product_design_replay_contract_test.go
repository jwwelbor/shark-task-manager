// TC-052 verifies the I-06 product-design replay contract E40-F07 produces
// for E40-F08 (spec.md "Produces: I-06"). Per REQ-NF-003/ADR-F07-10, this
// validator reads only in-repo artifacts -- bench/replay/i06-schema.yaml and
// tests/contracts/testdata/e40_i06/{valid,invalid}/** -- and never a
// populated fixture submodule, mirroring
// TestTC042_I05StageEvidenceContract's own submodule-independence
// discipline (bench/fixture-py and bench/fixture-repo are never read here).
//
// T-E40-F07-001 covered AC-001's document-kind core: REQ-F-001 (I-06 is two
// distinct, unambiguous document kinds) and REQ-F-002 (the replay bundle's
// schema_version/bundle_version/scenario_binding/entries[] field
// inventory). This task (T-E40-F07-002) adds AC-002: REQ-F-003's
// entry_digest recomputation, the one-byte-mutation boundary case, and the
// consumed_entries[]/bundle join-key subset check. REQ-F-010/REQ-F-011's
// artifact-record and interaction-proxy checks (AC-007, AC-008), and
// REQ-F-017/REQ-F-018/REQ-F-019's terminal-outcome mapping, malformed-field
// matrix, and vocabulary agreement (AC-013, AC-014, AC-015) remain later
// tasks' extensions of this same test file (T-E40-F07-008, -009).
package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// e40I06EntryDigestSpec decodes bench/replay/i06-schema.yaml's
// entry_digest block (REQ-F-003, T-E40-F07-002): the machine-readable
// parameters of the single canonical-serialization rule this file's
// recompute helpers implement. The Go code never hand-derives a second,
// competing definition of the rule -- schema_self_check below asserts
// these schema-declared parameters still match what the helpers
// implement, so a future schema edit that changes the rule without a
// matching Go change fails loudly instead of silently diverging.
type e40I06EntryDigestSpec struct {
	Algorithm        string `yaml:"algorithm"`
	Encoding         string `yaml:"encoding"`
	DigestPrefix     string `yaml:"digest_prefix"`
	Canonicalization string `yaml:"canonicalization"`
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
	EntryDigest                             e40I06EntryDigestSpec             `yaml:"entry_digest"`
	ArtifactType                            []string                          `yaml:"artifact_type"`
	EdgeKind                                []string                          `yaml:"edge_kind"`
	ReplayedInteractionProxiesDiscriminator string                            `yaml:"replayed_interaction_proxies_discriminator"`
	ReplayedInteractionProxiesFields        []string                          `yaml:"replayed_interaction_proxies_fields"`
	ReplayWaitCategory                      string                            `yaml:"replay_wait_category"`
	// ReplayWaitNsPlausibilityCeiling is REQ-F-011/AC-008(e)'s plausibility
	// ceiling for replayed_interaction_proxies.replay_wait_ns
	// (T-E40-F07-008): a value strictly greater than this is rejected as a
	// synthesized delay. Schema-owned per REQ-F-018 -- never hardcoded in
	// the validator.
	ReplayWaitNsPlausibilityCeiling int64    `yaml:"replay_wait_ns_plausibility_ceiling"`
	TerminalOutcome                 []string `yaml:"terminal_outcome"`
	ErrorKind                       []string `yaml:"error_kind"`
}

// e40I06I05IntervalCategorySchema decodes only the interval_category
// vocabulary from I-05's own bench/evidence/i05-schema.yaml
// (T-E40-F06-004) -- read-only. Per this task's Integration Contracts ("F07
// reads this category name from bench/evidence/i05-schema.yaml at
// validation time and edits nothing under bench/evidence/"), this proves
// i06-schema.yaml's replay_wait_category is still a member of I-05's own
// declared interval_category set rather than a private copy that has
// silently drifted from it.
type e40I06I05IntervalCategorySchema struct {
	IntervalCategory []string `yaml:"interval_category"`
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
		// REQ-F-003 (T-E40-F07-002): the schema's declared entry_digest
		// parameters must match what e40I06RecomputeEntryDigest below
		// actually implements -- a schema edit that changes the rule
		// without a matching Go change fails here, not silently.
		if schema.EntryDigest.Algorithm != "sha256" {
			t.Errorf("i06-schema.yaml entry_digest.algorithm = %q, want %q", schema.EntryDigest.Algorithm, "sha256")
		}
		if schema.EntryDigest.Encoding != "hex" {
			t.Errorf("i06-schema.yaml entry_digest.encoding = %q, want %q", schema.EntryDigest.Encoding, "hex")
		}
		if schema.EntryDigest.DigestPrefix != "" {
			t.Errorf("i06-schema.yaml entry_digest.digest_prefix = %q, want empty", schema.EntryDigest.DigestPrefix)
		}
		if schema.EntryDigest.Canonicalization != "compact_json_sorted_keys" {
			t.Errorf("i06-schema.yaml entry_digest.canonicalization = %q, want %q", schema.EntryDigest.Canonicalization, "compact_json_sorted_keys")
		}

		// AC-T3/REQ-NF-007 (T-E40-F07-008): replay_wait_category MUST be a
		// member of I-05's own interval_category vocabulary -- a read-only
		// cross-check against bench/evidence/i05-schema.yaml, never a
		// private I-06 copy of I-05's set.
		i05SchemaPath := filepath.Join(repoRoot, "bench", "evidence", "i05-schema.yaml")
		i05Data, err := os.ReadFile(i05SchemaPath)
		if err != nil {
			t.Fatalf("read i05 schema %s: %v", i05SchemaPath, err)
		}
		var i05Schema e40I06I05IntervalCategorySchema
		if err := yaml.Unmarshal(i05Data, &i05Schema); err != nil {
			t.Fatalf("parse i05 schema %s: %v", i05SchemaPath, err)
		}
		if !e40I06StringSet(i05Schema.IntervalCategory)[schema.ReplayWaitCategory] {
			t.Errorf("i06-schema.yaml replay_wait_category %q is not a member of I-05's own interval_category set %v (bench/evidence/i05-schema.yaml)", schema.ReplayWaitCategory, i05Schema.IntervalCategory)
		}

		// T-E40-F07-008: the replay_wait_ns plausibility ceiling
		// (AC-008(e)) must be declared and positive -- an unset or zero
		// ceiling would silently disable the synthesized-delay check in
		// e40I06ValidateInteractionProxies below.
		if schema.ReplayWaitNsPlausibilityCeiling <= 0 {
			t.Errorf("i06-schema.yaml replay_wait_ns_plausibility_ceiling = %d, want a positive value", schema.ReplayWaitNsPlausibilityCeiling)
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

	// AC-002/REQ-F-003 (T-E40-F07-002): entry_digest is the single join
	// key to I-05 (ADR-F07-08) -- it MUST recompute from the stored bundle
	// alone, a one-byte edit to any entry field MUST be caught, and a
	// result's consumed_entries[].entry_digest values MUST be a subset of
	// the bundle's own freshly recomputed digest set.
	t.Run("entry_digest", func(t *testing.T) {
		// AC-T1: every entry's entry_digest recomputes from the stored
		// bundle, excluding the digest field itself. Asserting zero
		// errors (not merely "contains no replay_bundle_mutated") also
		// catches the self-defeating implementation bug of hashing the
		// entry_digest field into its own input, which would produce a
		// spurious mismatch here even with no real mutation.
		t.Run("recomputes_from_valid_bundle", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "bundle-minimal.json"))
			_, errs := e40I06RecomputeBundleEntryDigests(doc)
			if len(errs) != 0 {
				t.Errorf("valid bundle fixture: want zero recompute errors, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		// AC-T2: a one-byte edit to any entry field yields
		// replay_bundle_mutated naming the entry -- and only that entry,
		// not every entry in the bundle (a validator that flags every
		// entry regardless of which one changed would still pass a bare
		// "contains replay_bundle_mutated" check, so this asserts exactly
		// one violation naming the mutated entry_id).
		t.Run("one_byte_mutation_detected", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", "entry-digest-mutated.json"))
			_, errs := e40I06RecomputeBundleEntryDigests(doc)
			if len(errs) != 1 {
				t.Fatalf("mutated bundle fixture: want exactly one recompute error, got %d:\n%s", len(errs), strings.Join(errs, "\n"))
			}
			if !e40ContainsErrorMatching(errs, "replay_bundle_mutated", "d01-vision-01") {
				t.Errorf("expected a replay_bundle_mutated error naming \"d01-vision-01\", got:\n%s", strings.Join(errs, "\n"))
			}
		})

		// AC-T3: a valid result fixture's stages[].consumed_entries[]
		// .entry_digest values are all present in the bundle's own
		// recomputed digest set (the join-key subset property).
		t.Run("consumed_entries_subset_of_bundle", func(t *testing.T) {
			bundleDoc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "bundle-minimal.json"))
			resultDoc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-minimal.json"))
			digestSet, mutationErrs := e40I06RecomputeBundleEntryDigests(bundleDoc)
			if len(mutationErrs) != 0 {
				t.Fatalf("valid bundle fixture: want zero recompute errors, got:\n%s", strings.Join(mutationErrs, "\n"))
			}
			errs := e40I06ValidateConsumedEntriesSubset(resultDoc, digestSet)
			if len(errs) != 0 {
				t.Errorf("valid result fixture: want zero subset errors, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		// AC-T3 (negative): a result fixture whose one consumed-entry
		// digest is not in the bundle's recomputed set is rejected naming
		// the offending entry_digest -- the join-key spoof case. A
		// validator that trusts the result's recorded entry_digest
		// without checking it against the bundle's own recomputed set
		// would accept this fixture silently.
		t.Run("consumed_entry_digest_not_in_bundle", func(t *testing.T) {
			bundleDoc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "bundle-minimal.json"))
			resultDoc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", "consumed-entry-digest-not-in-bundle.json"))
			digestSet, mutationErrs := e40I06RecomputeBundleEntryDigests(bundleDoc)
			if len(mutationErrs) != 0 {
				t.Fatalf("valid bundle fixture: want zero recompute errors, got:\n%s", strings.Join(mutationErrs, "\n"))
			}
			errs := e40I06ValidateConsumedEntriesSubset(resultDoc, digestSet)
			if len(errs) == 0 {
				t.Fatal("expected a replay_bundle_mutated violation for the spoofed entry_digest, got none")
			}
			const spoofedDigest = "6c6e2f7d94638e37638aca64ec00c797fe09397b1566d9dd19883a906f6c2cd6"
			if !e40ContainsErrorMatching(errs, "replay_bundle_mutated", spoofedDigest) {
				t.Errorf("expected a replay_bundle_mutated error naming the offending digest %q, got:\n%s", spoofedDigest, strings.Join(errs, "\n"))
			}
		})
	})

	// AC-007/REQ-F-010 (T-E40-F07-008): every D01-D05 artifact record's
	// consumers[] empty-versus-absent distinction, adopted verbatim from
	// I-05's own rule (ADR-F06-07 inherited, REQ-F-010) -- the same
	// decoder trap F06's TC-046 was designed to catch (a
	// `.get("consumers", [])`-style default would collapse
	// consumption_evidence_missing into orphan).
	t.Run("artifact_records", func(t *testing.T) {
		doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-artifact-records.json"))

		t.Run("passes_document_validation", func(t *testing.T) {
			errs := e40I06ValidateDocument(doc, "result", schema)
			if len(errs) != 0 {
				t.Errorf("valid artifact-records fixture failed document validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		stages, _ := doc["stages"].([]interface{})

		t.Run("field_inventory_and_edge_kind", func(t *testing.T) {
			errs := e40I06ValidateArtifactRecords(stages, schema)
			if len(errs) != 0 {
				t.Errorf("valid artifact-records fixture failed artifact-record validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		// AC-T1/AC-007: consumers: [] yields orphan; the consumers key
		// entirely absent yields consumption_evidence_missing; neither
		// verdict applies to the other entry.
		t.Run("empty_consumers_is_orphan", func(t *testing.T) {
			artifact := e40I06FindArtifactByPath(stages, "docs/product/D01-vision-orphan.md")
			if artifact == nil {
				t.Fatal("fixture artifact docs/product/D01-vision-orphan.md not found")
			}
			if _, isList := artifact["consumers"].([]interface{}); !isList {
				t.Fatal("fixture artifact must carry consumers: [] (present, empty) for this case")
			}
			if v := e40I06ArtifactConsumersVerdict(artifact); v != "orphan" {
				t.Errorf("consumers: [] verdict = %q, want %q", v, "orphan")
			}
		})

		t.Run("absent_consumers_is_consumption_evidence_missing", func(t *testing.T) {
			artifact := e40I06FindArtifactByPath(stages, "docs/product/D02-personas-uncollected.md")
			if artifact == nil {
				t.Fatal("fixture artifact docs/product/D02-personas-uncollected.md not found")
			}
			if _, present := artifact["consumers"]; present {
				t.Fatal("fixture artifact must omit the consumers key entirely for this case")
			}
			if v := e40I06ArtifactConsumersVerdict(artifact); v != "consumption_evidence_missing" {
				t.Errorf("absent consumers verdict = %q, want %q", v, "consumption_evidence_missing")
			}
		})

		// A downstream edge from a later D0X stage to an earlier stage's
		// artifact is recorded with its edge_kind intact and readable --
		// the reused-versus-orphan distinction UAT-18 needs from E40-F10.
		t.Run("downstream_edge_recorded_intact", func(t *testing.T) {
			artifact := e40I06FindArtifactByPath(stages, "docs/product/D01-vision-reused.md")
			if artifact == nil {
				t.Fatal("fixture artifact docs/product/D01-vision-reused.md not found")
			}
			if v := e40I06ArtifactConsumersVerdict(artifact); v != "consumed" {
				t.Fatalf("consumers: [{...}] verdict = %q, want %q", v, "consumed")
			}
			consumers, _ := artifact["consumers"].([]interface{})
			if len(consumers) != 1 {
				t.Fatalf("expected exactly one consumer edge, got %d", len(consumers))
			}
			edge, _ := consumers[0].(map[string]interface{})
			if edge["consuming_stage"] != "D03" {
				t.Errorf("edge consuming_stage = %v, want %q", edge["consuming_stage"], "D03")
			}
			if edge["edge_kind"] != "referenced" {
				t.Errorf("edge edge_kind = %v, want %q", edge["edge_kind"], "referenced")
			}
			if s, _ := edge["observed_at"].(string); s == "" {
				t.Error("edge observed_at is missing or empty")
			}
		})
	})

	// AC-008/REQ-F-011 (T-E40-F07-008): replayed_interaction_proxies'
	// closed field set, required measurement_kind discriminator, and the
	// human-time-name prohibition; AC-T3/REQ-NF-007's replay_wait_category
	// and replay_wait_ns plausibility-ceiling checks.
	t.Run("interaction_proxies", func(t *testing.T) {
		t.Run("valid_proxy_block_at_boundary_ceiling", func(t *testing.T) {
			// replay_wait_ns sits exactly *at* the declared ceiling
			// (REQ-F-011 "exceeds"): the boundary value itself MUST pass.
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-proxies.json"))
			errs := e40I06ValidateInteractionProxies(doc, schema)
			if len(errs) != 0 {
				t.Errorf("valid proxies fixture failed, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
			proxies, _ := doc["replayed_interaction_proxies"].(map[string]interface{})
			if proxies["replay_wait_category"] != "replay_or_human_gate_wait" {
				t.Errorf("replay_wait_category = %v, want %q", proxies["replay_wait_category"], "replay_or_human_gate_wait")
			}
		})

		t.Run("valid_result_minimal_proxy_block", func(t *testing.T) {
			// result-minimal.json's own replayed_interaction_proxies block
			// (T-E40-F07-001) must also satisfy this task's closed-set
			// validation, proving the new checks don't regress the
			// already-committed fixture.
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-minimal.json"))
			errs := e40I06ValidateInteractionProxies(doc, schema)
			if len(errs) != 0 {
				t.Errorf("result-minimal.json proxies failed, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		cases := []struct {
			name    string
			file    string
			wantAny []string
		}{
			{"missing_measurement_kind", "proxy-missing-measurement-kind.json", []string{"field_missing", "measurement_kind"}},
			{"wrong_measurement_kind_value", "proxy-wrong-measurement-kind.json", []string{"proxy_measurement_kind_invalid"}},
			{"extra_field_outside_closed_set", "proxy-extra-field.json", []string{"proxy_field_unknown", "extra_debug_field"}},
			{"human_time_named_field", "proxy-human-time-field.json", []string{"proxy_field_unknown", "stakeholder_minutes"}},
			{"replay_wait_ns_exceeds_ceiling", "proxy-wait-implausible.json", []string{"proxy_wait_implausible", "replay_wait_ns"}},
		}
		for _, c := range cases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", c.file))
				errs := e40I06ValidateInteractionProxies(doc, schema)
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

	// AC-T4/REQ-F-008 (T-E40-F07-008): terminal_outcome: unresolved_gate
	// paired with unresolved_gate_count: 0 is a named contradiction -- the
	// counter obligation REQ-F-008 fixes, checked independently of the
	// proxy block's own field-inventory validity. This is deliberately
	// the narrow contradiction check AC-T4 fixes, not a literal
	// count-equality against a per-stage gate-event record: Document B's
	// stages[] shape is fixed by spec.md as
	// {stage, applicable, reason?, artifacts[], consumed_entries[]} with
	// no gate-event field, and REQ-F-001 forbids redefining that shape
	// here.
	t.Run("unresolved_gate_count_consistency", func(t *testing.T) {
		t.Run("nonzero_count_accepted", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-unresolved-gate.json"))
			errs := e40I06ValidateUnresolvedGateCountConsistency(doc)
			if len(errs) != 0 {
				t.Errorf("valid unresolved-gate fixture failed, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("complete_outcome_not_checked", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "valid", "result-minimal.json"))
			errs := e40I06ValidateUnresolvedGateCountConsistency(doc)
			if len(errs) != 0 {
				t.Errorf("terminal_outcome complete fixture must never be checked here, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("zero_count_rejected", func(t *testing.T) {
			doc := e40I06ReadDocument(t, filepath.Join(testdataRoot, "invalid", "unresolved-gate-count-zero.json"))
			errs := e40I06ValidateUnresolvedGateCountConsistency(doc)
			if len(errs) == 0 {
				t.Fatal("expected an unresolved_gate_count_inconsistent violation, got none")
			}
			if !e40ContainsErrorMatching(errs, "unresolved_gate_count_inconsistent") {
				t.Errorf("expected unresolved_gate_count_inconsistent, got:\n%s", strings.Join(errs, "\n"))
			}
		})
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

// e40I06CanonicalEntryJSON serializes entry (every field except
// entry_digest itself) per i06-schema.yaml's entry_digest block
// (REQ-F-003, T-E40-F07-002, "compact_json_sorted_keys"): a compact JSON
// object, keys sorted lexicographically at every nesting level, UTF-8, no
// \uXXXX-escaped non-ASCII characters. Go's encoding/json already sorts
// map[string]interface{} keys (recursively) when marshaling, so the only
// deviation from json.Marshal's default output this needs is disabling
// HTML-escaping (Go escapes '<', '>', '&' by default; Python's json.dumps
// and jq do not) to keep the two languages byte-identical.
func e40I06CanonicalEntryJSON(entry map[string]interface{}) ([]byte, error) {
	filtered := make(map[string]interface{}, len(entry))
	for k, v := range entry {
		if k == "entry_digest" {
			continue
		}
		filtered[k] = v
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(filtered); err != nil {
		return nil, err
	}
	// Encoder.Encode appends a trailing newline the canonical form must
	// not carry.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// e40I06RecomputeEntryDigest recomputes REQ-F-003's entry_digest for one
// bundle entry: sha256 hex digest (no prefix, per i06-schema.yaml's
// entry_digest.digest_prefix) of e40I06CanonicalEntryJSON's output.
func e40I06RecomputeEntryDigest(entry map[string]interface{}) (string, error) {
	canonical, err := e40I06CanonicalEntryJSON(entry)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// e40I06RecomputeBundleEntryDigests recomputes REQ-F-003's entry_digest
// for every entries[] element in a decoded bundle document. It returns
// the freshly recomputed digest set (the ground truth "bundle's
// recomputed digest set" REQ-F-003 names as the single join key to I-05,
// independent of whatever value happens to be stored in each entry's own
// entry_digest field) and one replay_bundle_mutated violation per entry
// whose recomputed digest does not match its stored entry_digest (AC-T2).
// Entries with a malformed or missing entry_id/entry_digest are skipped
// here -- those are e40I06ValidateBundleBody's field_missing/
// field_malformed violations, not this function's concern.
//
// Deliberately not called from e40I06ValidateDocument/
// e40I06ValidateBundleBody: several of T-E40-F07-001's existing invalid/
// fixtures (duplicate-ordinal.json, unknown-stage.json, etc.) carry a
// placeholder entry_digest that predates this task's canonicalization
// rule and would now also fail this check, which is harmless for those
// fixtures' own "wantAny" substring assertions but would conflate two
// unrelated defects in one document. TC-052 exercises this recompute path
// through its own dedicated "entry_digest" subtests instead. Wiring this
// into the general bundle-validation pipeline is left to a later task,
// which will also need to update those pre-existing fixtures' entry_digest
// values.
func e40I06RecomputeBundleEntryDigests(bundle map[string]interface{}) (map[string]bool, []string) {
	var errs []string
	digestSet := map[string]bool{}
	rawEntries, _ := bundle["entries"].([]interface{})
	for i, rawEntry := range rawEntries {
		entry, isMap := rawEntry.(map[string]interface{})
		if !isMap {
			continue
		}
		entryID, _ := entry["entry_id"].(string)
		storedDigest, hasStoredDigest := entry["entry_digest"].(string)
		got, err := e40I06RecomputeEntryDigest(entry)
		if err != nil {
			errs = append(errs, fmt.Sprintf("field_malformed: entries[%d] could not be canonically serialized: %v", i, err))
			continue
		}
		digestSet[got] = true
		if !hasStoredDigest || storedDigest == "" {
			continue
		}
		if got != storedDigest {
			errs = append(errs, fmt.Sprintf(
				"replay_bundle_mutated: entries[%d] (entry_id=%s) recomputed entry_digest %q does not match stored entry_digest %q",
				i, entryID, got, storedDigest,
			))
		}
	}
	return digestSet, errs
}

// e40I06ValidateConsumedEntriesSubset checks REQ-F-003's join-key
// invariant (AC-T3): every stages[].consumed_entries[].entry_digest value
// in a decoded result document must appear in bundleDigests, the bundle's
// own recomputed digest set (e40I06RecomputeBundleEntryDigests). A value
// absent from that set -- whether from a mutated bundle entry or a
// forged/spoofed digest the result never derived from any bundle entry --
// is rejected as replay_bundle_mutated naming the offending digest, per
// REQ-F-003 ("A result whose recorded entry_digest does not recompute
// from the bundle MUST be rejected as replay_bundle_mutated naming the
// entry").
func e40I06ValidateConsumedEntriesSubset(result map[string]interface{}, bundleDigests map[string]bool) []string {
	var errs []string
	rawStages, _ := result["stages"].([]interface{})
	for si, rawStage := range rawStages {
		stage, isMap := rawStage.(map[string]interface{})
		if !isMap {
			continue
		}
		rawConsumed, _ := stage["consumed_entries"].([]interface{})
		for ci, rawEntry := range rawConsumed {
			entry, isMap := rawEntry.(map[string]interface{})
			if !isMap {
				continue
			}
			digest, _ := entry["entry_digest"].(string)
			if digest == "" {
				errs = append(errs, fmt.Sprintf("field_missing: stages[%d].consumed_entries[%d].entry_digest", si, ci))
				continue
			}
			if !bundleDigests[digest] {
				errs = append(errs, fmt.Sprintf(
					"replay_bundle_mutated: stages[%d].consumed_entries[%d].entry_digest %q is not in the bundle's recomputed digest set",
					si, ci, digest,
				))
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

// e40I06AsNumber reports whether v decoded as a JSON number
// (encoding/json's map[string]interface{} decodes every JSON number as
// float64).
func e40I06AsNumber(v interface{}) (float64, bool) {
	n, ok := v.(float64)
	return n, ok
}

// e40I06ArtifactConsumersVerdict classifies REQ-F-010's empty-versus-absent
// distinction for one decoded artifact record's consumers field, adopted
// verbatim from I-05's own rule (ADR-F06-07 inherited): the verdict is
// derived strictly from Go's comma-ok map key-presence check on the raw
// decoded JSON object, never from a zero-value/omitempty-style default --
// the same discipline verify-stage-evidence.sh's Python implementation
// applies via `"consumers" not in artifact`.
func e40I06ArtifactConsumersVerdict(artifact map[string]interface{}) string {
	raw, present := artifact["consumers"]
	if !present {
		return "consumption_evidence_missing"
	}
	list, _ := raw.([]interface{})
	if len(list) == 0 {
		return "orphan"
	}
	return "consumed"
}

// e40I06ArtifactRecordRequiredFields is REQ-F-010's per-artifact field
// inventory, excluding consumers -- consumers' empty-vs-absent duality
// (e40I06ArtifactConsumersVerdict) is a distinguishable *state*, never a
// plain required-vs-missing field, the same "consumers is intentionally
// not required" discipline I-05's own e40I05ValidateArtifacts documents,
// adopted verbatim here per REQ-F-010/ADR-F06-07. Field name is `stage`
// (not I-05's `producer_stage`) -- I-06's own document shape, matching the
// already-committed result-minimal.json fixture (T-E40-F07-001).
var e40I06ArtifactRecordRequiredFields = []string{
	"stage", "artifact_type", "path", "digest", "size_bytes",
	"produced_at", "revision_index", "prompt_digest", "input_digests",
	"consumed_entries",
}

// e40I06ValidateArtifactRecords checks REQ-F-010's per-artifact field
// inventory (excluding consumers' own empty-vs-absent state, handled
// separately by e40I06ArtifactConsumersVerdict), artifact_type vocabulary
// membership, and every present consumers[] edge's
// {consuming_stage, edge_kind, observed_at} field set and edge_kind
// vocabulary membership.
func e40I06ValidateArtifactRecords(stages []interface{}, schema *e40I06Schema) []string {
	var errs []string
	artifactTypeSet := e40I06StringSet(schema.ArtifactType)
	edgeKindSet := e40I06StringSet(schema.EdgeKind)
	for si, rawStage := range stages {
		stage, ok := rawStage.(map[string]interface{})
		if !ok {
			continue
		}
		rawArtifacts, _ := stage["artifacts"].([]interface{})
		for ai, rawArtifact := range rawArtifacts {
			artifact, ok := rawArtifact.(map[string]interface{})
			if !ok {
				errs = append(errs, fmt.Sprintf("field_malformed: stages[%d].artifacts[%d] must be an object", si, ai))
				continue
			}
			for _, field := range e40I06ArtifactRecordRequiredFields {
				if _, present := artifact[field]; !present {
					errs = append(errs, fmt.Sprintf("field_missing: stages[%d].artifacts[%d].%s", si, ai, field))
				}
			}
			if aType, _ := artifact["artifact_type"].(string); aType != "" && !artifactTypeSet[aType] {
				errs = append(errs, fmt.Sprintf("vocabulary_value_unknown: stages[%d].artifacts[%d].artifact_type %q is not one of %v", si, ai, aType, schema.ArtifactType))
			}
			rawConsumers, present := artifact["consumers"]
			if !present {
				continue
			}
			consumers, isList := rawConsumers.([]interface{})
			if !isList {
				errs = append(errs, fmt.Sprintf("field_malformed: stages[%d].artifacts[%d].consumers must be an array when present", si, ai))
				continue
			}
			for ci, rawEdge := range consumers {
				edge, ok := rawEdge.(map[string]interface{})
				if !ok {
					errs = append(errs, fmt.Sprintf("field_malformed: stages[%d].artifacts[%d].consumers[%d] must be an object", si, ai, ci))
					continue
				}
				for _, f := range []string{"consuming_stage", "edge_kind", "observed_at"} {
					if _, present := edge[f]; !present {
						errs = append(errs, fmt.Sprintf("field_missing: stages[%d].artifacts[%d].consumers[%d].%s", si, ai, ci, f))
					}
				}
				if ek, _ := edge["edge_kind"].(string); ek != "" && !edgeKindSet[ek] {
					errs = append(errs, fmt.Sprintf("vocabulary_value_unknown: stages[%d].artifacts[%d].consumers[%d].edge_kind %q is not one of %v", si, ai, ci, ek, schema.EdgeKind))
				}
			}
		}
	}
	return errs
}

// e40I06FindArtifactByPath returns the first stages[].artifacts[] element
// whose path field equals path, or nil if none matches. Test-only lookup
// helper for AC-007's named fixture entries.
func e40I06FindArtifactByPath(stages []interface{}, path string) map[string]interface{} {
	for _, rawStage := range stages {
		stage, ok := rawStage.(map[string]interface{})
		if !ok {
			continue
		}
		rawArtifacts, _ := stage["artifacts"].([]interface{})
		for _, rawArtifact := range rawArtifacts {
			artifact, ok := rawArtifact.(map[string]interface{})
			if !ok {
				continue
			}
			if p, _ := artifact["path"].(string); p == path {
				return artifact
			}
		}
	}
	return nil
}

// e40I06ValidateInteractionProxies checks REQ-F-011's closed field set,
// required measurement_kind discriminator, and human-time-name
// prohibition (enforced structurally: any field name not in the closed
// set is rejected, so a human-attributed field name like
// stakeholder_minutes is caught the same way as any other unknown field),
// plus AC-T3/REQ-NF-007's replay_wait_category and replay_wait_ns
// plausibility-ceiling checks. Unknown-field violations are emitted in
// sorted order so two runs over the same fixture produce byte-identical
// output (REQ-NF-004), independent of Go map iteration order.
func e40I06ValidateInteractionProxies(doc map[string]interface{}, schema *e40I06Schema) []string {
	var errs []string

	raw, present := doc["replayed_interaction_proxies"]
	if !present {
		return []string{"field_missing: replayed_interaction_proxies"}
	}
	proxies, ok := raw.(map[string]interface{})
	if !ok {
		return []string{"field_malformed: replayed_interaction_proxies must be an object"}
	}

	if mk, hasMK := proxies["measurement_kind"]; !hasMK {
		errs = append(errs, "field_missing: replayed_interaction_proxies.measurement_kind")
	} else if s, isString := mk.(string); !isString || s != schema.ReplayedInteractionProxiesDiscriminator {
		errs = append(errs, fmt.Sprintf("proxy_measurement_kind_invalid: replayed_interaction_proxies.measurement_kind = %v, want %q", mk, schema.ReplayedInteractionProxiesDiscriminator))
	}

	fieldSet := e40I06StringSet(schema.ReplayedInteractionProxiesFields)
	var unknown []string
	for field := range proxies {
		if !fieldSet[field] {
			unknown = append(unknown, field)
		}
	}
	sort.Strings(unknown)
	for _, field := range unknown {
		errs = append(errs, fmt.Sprintf("proxy_field_unknown: replayed_interaction_proxies.%s is not in the closed field set %v", field, schema.ReplayedInteractionProxiesFields))
	}

	if cat, hasCat := proxies["replay_wait_category"].(string); hasCat && cat != schema.ReplayWaitCategory {
		errs = append(errs, fmt.Sprintf("vocabulary_value_unknown: replayed_interaction_proxies.replay_wait_category = %q, want %q", cat, schema.ReplayWaitCategory))
	}

	if schema.ReplayWaitNsPlausibilityCeiling <= 0 {
		// The ceiling is required to be schema-declared and positive
		// (asserted directly in schema_self_check); this defensive branch
		// keeps the check from silently no-op'ing if that assertion were
		// ever bypassed.
		errs = append(errs, "field_missing: i06-schema.yaml replay_wait_ns_plausibility_ceiling must be declared and positive")
	} else if waitNs, isNum := e40I06AsNumber(proxies["replay_wait_ns"]); isNum {
		if waitNs > float64(schema.ReplayWaitNsPlausibilityCeiling) {
			errs = append(errs, fmt.Sprintf("proxy_wait_implausible: replayed_interaction_proxies.replay_wait_ns = %v exceeds plausibility ceiling %d ns for a local file read (synthesized delay)", waitNs, schema.ReplayWaitNsPlausibilityCeiling))
		}
	}

	return errs
}

// e40I06ValidateUnresolvedGateCountConsistency checks REQ-F-008/AC-T4: a
// result whose terminal_outcome is unresolved_gate MUST report a nonzero
// replayed_interaction_proxies.unresolved_gate_count -- the counter that
// names how many times a missing bundle entry stopped the prelude cannot
// itself be zero for a result reporting exactly that stop. This is
// deliberately the narrow contradiction check AC-T4 fixes, not a literal
// count-equality against a per-stage gate-event record: Document B's
// stages[] shape is fixed by spec.md as
// {stage, applicable, reason?, artifacts[], consumed_entries[]} with no
// gate-event field, and REQ-F-001 forbids redefining that shape here.
func e40I06ValidateUnresolvedGateCountConsistency(doc map[string]interface{}) []string {
	outcome, _ := doc["terminal_outcome"].(string)
	if outcome != "unresolved_gate" {
		return nil
	}
	proxies, _ := doc["replayed_interaction_proxies"].(map[string]interface{})
	count, ok := e40I06AsNumber(proxies["unresolved_gate_count"])
	if !ok || count == 0 {
		return []string{"unresolved_gate_count_inconsistent: terminal_outcome is unresolved_gate but replayed_interaction_proxies.unresolved_gate_count is 0"}
	}
	return nil
}
