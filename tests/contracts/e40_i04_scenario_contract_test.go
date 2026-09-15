// TC-030 verifies the I-04 lifecycle scenario package contract E40-F05
// produces for E40-F06, E40-F07, and E40-F08 (spec.md "Produces: I-04").
// Per REQ-NF-003/ADR-F05-07, this validator reads only in-repo artifacts --
// bench/scenarios/scenarios.yaml, bench/scenarios/packages/*/package.yaml,
// and bench/adapters/*/adapter.yaml -- and never bench/fixture-py or
// bench/fixture-repo content, so its result is identical whether either
// fixture submodule is populated or gitlink-only (AC-016, task AC-T1),
// exactly mirroring TestTC002_I01FixturePackageVisibilityContract's own
// submodule-independence discipline.
package contracts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// e40I04SupportedSchemaVersion is the package.yaml/scenarios.yaml
// schema_version this validator understands (mirrors e40I01SupportedSchemaVersion).
// Bumped 1.0 -> 1.1 by REQ-F-007/ADR-F11-01 alongside the six-family
// vocabulary widening; the validator supports exactly one version, so a
// package left at 1.0 is rejected.
const e40I04SupportedSchemaVersion = "1.1"

var e40I04ScenarioIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// e40I04FinalPredicateFamilies is the closed final_predicate.kind vocabulary
// (REQ-F-010) mapped to the one entity_family each kind is permitted for.
// "task_acceptance_tests" and "descendant_oracles_union" were added by
// T-E40-F11-005 (ADR-F11-04): the existing map is kind->exactly-one-family,
// so adding two entries preserves that 1:1 invariant and lets the existing
// case-14-predicate-kind-not-permitted-for-family.yaml guard cover the new
// kinds without new mechanism. Their operand fields (analogous to
// f2p_test_ids/acceptance_test_ids/etc.) were deliberately NOT added to the
// switch in e40I04ValidateFinalPredicate by T-005 -- the spec did not state
// their shape, and inventing one would have been schema authored by that
// task rather than by REQ-F-010's sibling curation tasks (T-E40-F11-008 for
// task_acceptance_tests, T-E40-F11-009 for descendant_oracles_union), which
// admit the first real task/epic packages and are accountable for that
// operand shape. T-E40-F11-008 admitted py-task-delete-task reusing
// acceptance_tests' own acceptance_test_ids field (already key-legal via
// e40I04CheckUnknownKeys) without adding a case for it. T-E40-F11-009
// admitted py-epic-task-organization reusing child_oracles_union's own
// integration_test_ids/child_oracles fields (likewise already key-legal)
// and DID add a `case "descendant_oracles_union":` below, mirroring
// child_oracles_union's own operand-presence check.
var e40I04FinalPredicateFamilies = map[string]string{
	"f2p_p2p":                  "bug",
	"acceptance_tests":         "change_card",
	"p2p_plus_rule_drop":       "tech_debt",
	"child_oracles_union":      "feature",
	"task_acceptance_tests":    "task",
	"descendant_oracles_union": "epic",
}

// e40I04ValidEntityFamilies is the closed six-family admission vocabulary
// (REQ-F-007, ADR-F11-02). This map is the single authoritative source;
// bench/scripts/admit-scenario.sh's validate_family_invariant is a consumer
// that must agree with it, never a second source of truth. "sprint" and
// "question" are deliberately excluded -- they are orchestration workflows,
// not delivery-scenario roots (feature.md REQ-F-007).
var e40I04ValidEntityFamilies = map[string]bool{
	"epic":        true,
	"feature":     true,
	"task":        true,
	"bug":         true,
	"change_card": true,
	"tech_debt":   true,
}

var e40I04PreludeStages = []string{"D01", "D02", "D03", "D04", "D05"}

// e40I04ScenarioFixtureReg / e40I04ScenarioAdapterReg / e40I04ScenarioIndex decode
// the I-04 index (bench/scenarios/scenarios.yaml). Unlike package.yaml, the
// index is not subject to REQ-F-015's unknown-field rejection cases, so a
// plain typed decode is sufficient here.
type e40I04ScenarioFixtureReg struct {
	SubmodulePath string `yaml:"submodule_path"`
}

type e40I04ScenarioAdapterReg struct {
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
}

type e40I04ScenarioIndex struct {
	SchemaVersion string                              `yaml:"schema_version"`
	Fixtures      map[string]e40I04ScenarioFixtureReg `yaml:"fixtures"`
	Adapters      map[string]e40I04ScenarioAdapterReg `yaml:"adapters"`
	Scenarios     []string                            `yaml:"scenarios"`
}

// TestTC030_I04ScenarioPackageContract is the shared contract test E40-F06,
// E40-F07, and E40-F08 must reuse verbatim (spec.md Cross-feature
// interactions: "no twin test is created").
func TestTC030_I04ScenarioPackageContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	scenariosRoot := filepath.Join(repoRoot, "bench", "scenarios")
	indexPath := filepath.Join(scenariosRoot, "scenarios.yaml")

	t.Run("index_and_registered_packages", func(t *testing.T) {
		index := e40I04ReadScenarioIndex(t, indexPath)

		// AC-T1 requires unknown-field-rejecting unmarshalling for the index
		// too, not only package.yaml -- otherwise scenarios.yaml could grow
		// one of REQ-F-001's forbidden I-01 fields (or any other stray key)
		// one level above where case-16-forbidden-i01-fields.yaml proves the
		// package-level guard, and TC-030 would silently ignore it.
		indexData, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("read scenario index %s: %v", indexPath, err)
		}
		indexMap := e40I04ParseYAMLMap(t, indexData)
		if errs := e40I04ValidateIndexKnownFields(indexMap); len(errs) > 0 {
			t.Errorf("scenarios.yaml carries unknown fields:\n%s", strings.Join(errs, "\n"))
		}

		if index.SchemaVersion != e40I04SupportedSchemaVersion {
			t.Errorf("scenarios.yaml schema_version = %q, want %q", index.SchemaVersion, e40I04SupportedSchemaVersion)
		}
		if len(index.Fixtures) == 0 {
			t.Error("scenarios.yaml fixtures is empty")
		}
		if len(index.Adapters) == 0 {
			t.Error("scenarios.yaml adapters is empty")
		}
		for name, reg := range index.Adapters {
			adapterYAML := filepath.Join(repoRoot, reg.Path, "adapter.yaml")
			if !e40FileExists(adapterYAML) {
				t.Errorf("adapter %q registered path %q has no adapter.yaml at %s", name, reg.Path, adapterYAML)
				continue
			}
			// The index's version is the pin a package's admitted
			// adapter: {name, version} block gets diffed against (adapter.yaml's
			// own header comment). If the two ever disagree, that pin is
			// already stale.
			descriptor := e40I04ReadAdapterDescriptor(t, adapterYAML)
			if descriptor.Name != name {
				t.Errorf("adapter %q registered path %q has adapter.yaml name = %q, want %q", name, reg.Path, descriptor.Name, name)
			}
			if descriptor.Version != reg.Version {
				t.Errorf("adapter %q: scenarios.yaml version = %q, adapter.yaml version = %q, want agreement", name, reg.Version, descriptor.Version)
			}
		}
		for id, reg := range index.Fixtures {
			if strings.TrimSpace(reg.SubmodulePath) == "" {
				t.Errorf("fixture %q has empty submodule_path", id)
			}
		}

		// AC-T3: index/directory agreement. Every scenario_id in the index
		// must have a matching package.yaml, and every directory under
		// bench/scenarios/packages/ must be listed in the index -- so a
		// package directory absent from the index is never silently
		// invisible to nothing else. bench/scenarios/packages/ does not yet
		// exist at this task (real seed content lands in
		// T-E40-F05-008/010/011/013), so both sides are legitimately empty
		// right now; the check still runs so it fires the moment either
		// side gains an entry without the other.
		packagesRoot := filepath.Join(scenariosRoot, "packages")
		for _, rel := range index.Scenarios {
			pkgPath := filepath.Join(scenariosRoot, rel, "package.yaml")
			if !e40FileExists(pkgPath) {
				t.Errorf("scenarios.yaml scenarios lists %q but %s does not exist", rel, pkgPath)
			}
		}
		dirIDs := e40I04ListPackageDirs(t, packagesRoot)
		indexIDs := map[string]bool{}
		for _, rel := range index.Scenarios {
			indexIDs[filepath.Base(rel)] = true
		}
		for _, dirID := range dirIDs {
			if !indexIDs[dirID] {
				t.Errorf("bench/scenarios/packages/%s exists but is not listed in scenarios.yaml scenarios", dirID)
			}
		}

		// Validate every package the index actually registers. index.Scenarios
		// now lists the four committed seed packages, so this loop runs for
		// real and does exercise AC-019's admission-block cross-encoding
		// assertion inside e40I04ValidateScenarioPackage -- but only ever in
		// the matching-values direction, since every seed's admission block
		// mirrors its own top-level toolchain_identity exactly; there is no
		// negative-path testdata case that constructs a divergent
		// admission.toolchain_identity (tracked separately). AC-009 (load
		// twice, identical result) is proven here the same way TC-001 proves
		// it for I-01: parse the same committed bytes twice within one test
		// run and compare.
		var scenarioIDEntries []e40I04ScenarioIDEntry
		for _, rel := range index.Scenarios {
			pkgPath := filepath.Join(scenariosRoot, rel, "package.yaml")
			pkgDir := filepath.Dir(pkgPath)
			data, err := os.ReadFile(pkgPath)
			if err != nil {
				t.Errorf("read %s: %v", pkgPath, err)
				continue
			}
			first := e40I04ParseYAMLMap(t, data)
			second := e40I04ParseYAMLMap(t, data)
			if !reflect.DeepEqual(first, second) {
				t.Errorf("%s: loading the same package twice produced different results (AC-009)", rel)
			}
			// evaluatorRoot is packageDir/evaluator itself here -- a real
			// package's evaluator/ is always a genuine immediate child of
			// its own directory (see e40I04ValidateScenarioPackage's doc
			// comment; code review NEW-1).
			if errs := e40I04ValidateScenarioPackage(first, index, pkgDir, filepath.Join(pkgDir, "evaluator")); len(errs) > 0 {
				t.Errorf("%s violates the I-04 package contract:\n%s", rel, strings.Join(errs, "\n"))
			}
			if scenarioID, ok := first["scenario_id"].(string); ok && strings.TrimSpace(scenarioID) != "" {
				scenarioIDEntries = append(scenarioIDEntries, e40I04ScenarioIDEntry{Rel: rel, ScenarioID: scenarioID})
			}
		}
		// REQ-F-002: scenario_id must be unique across the whole corpus, not
		// merely well-formed within one package (UAT
		// uat-20260813T204821Z-E40-F05.md Finding 3).
		for _, msg := range e40I04FindDuplicateScenarioIDs(scenarioIDEntries) {
			t.Error(msg)
		}
	})

	// AC-005: table-driven malformed-package subtests. Each case is a real
	// committed fixture under tests/contracts/testdata/e40_i04/ (REQ-F-015's
	// enumerated rejection list), asserting the validator's error names the
	// one specific field that case violates -- not a generic message.
	t.Run("malformed_package_cases", func(t *testing.T) {
		index := e40I04ReadScenarioIndex(t, indexPath)
		testdataDir := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i04")

		cases := []struct {
			file    string
			wantAny []string // at least one of these substrings must appear in some error
			// family is the shared testdata baseline directory
			// (valid|valid-feature) this case's evaluator_only.* fields
			// resolve against for the requireEvaluator containment check
			// (code review NEW-1, code-review-20260813T232337Z-E40-F05.md).
			// Empty defaults to "valid" -- every case-*.yaml derives from
			// one of the two baselines and reuses its shared evaluator/
			// content via a "valid/" or "valid-feature/" path prefix (see
			// valid/package.yaml, valid-feature/package.yaml).
			family string
		}{
			{"case-01-unknown-entity-family.yaml", []string{"entity_family"}, ""},
			{"case-02-unregistered-fixture-id.yaml", []string{"fixture.fixture_id"}, ""},
			{"case-03-unregistered-adapter-name.yaml", []string{"adapter.name"}, ""},
			{"case-04-max-cost-usd-missing.yaml", []string{"resource_policy.max_cost_usd"}, ""},
			{"case-05-max-cost-usd-nonpositive.yaml", []string{"resource_policy.max_cost_usd"}, ""},
			{"case-06-max-wall-clock-seconds-missing.yaml", []string{"resource_policy.max_wall_clock_seconds"}, ""},
			{"case-07-max-wall-clock-seconds-nonpositive.yaml", []string{"resource_policy.max_wall_clock_seconds"}, ""},
			{"case-08-max-generated-tasks-missing.yaml", []string{"resource_policy.max_generated_tasks"}, ""},
			{"case-09-max-generated-tasks-nonpositive.yaml", []string{"resource_policy.max_generated_tasks"}, ""},
			{"case-10-prelude-stage-missing-boolean.yaml", []string{"stage_matrix.prelude.D03"}, ""},
			{"case-11-prelude-false-stage-missing-reason.yaml", []string{"stage_matrix.prelude.D02"}, ""},
			{"case-12-family-invariant-violation.yaml", []string{"D01"}, ""},
			{"case-13-unknown-predicate-kind.yaml", []string{"final_predicate.kind"}, ""},
			{"case-14-predicate-kind-not-permitted-for-family.yaml", []string{"final_predicate.kind"}, ""},
			{"case-15-predicate-operand-path-missing.yaml", []string{"evaluator_only.reference_solution"}, ""},
			{"case-16-forbidden-i01-fields.yaml", []string{"p2p_sets"}, ""},
			{"case-17-leak-surface-agent-visible-in-evaluator.yaml", []string{"input.agent_visible"}, ""},

			// Cases 18-24: rework of UAT report
			// uat-20260813T204821Z-E40-F05.md Findings 1 and 2. Finding 1
			// (HIGH): e40I04ResolvesInsideEvaluator was purely lexical --
			// it never resolved symlinks and never rejected absolute or
			// traversal paths. Cases 18-22 sweep every path-typed field
			// REQ-F-009/REQ-F-015 govern for the same defect class (lexical
			// or shape-only checks standing in for canonicalized-path
			// containment): input.agent_visible (18: raw absolute path,
			// 19: symlink resolving into evaluator/ without the literal
			// word "evaluator" anywhere in the raw string) and all three
			// evaluator_only fields, which previously had NO containment
			// check at all (20: reference_solution pointing at a real file
			// outside evaluator/, 21: oracle_tests escaping the package
			// root via ".." to an existing external file, 22: answer_keys
			// -- previously not validated in any way -- doing the same).
			// Finding 2 (MEDIUM): input.agent_visible and replay_reference
			// were checked for shape only, never resolved against the
			// filesystem (23, 24).
			{"case-18-agent-visible-absolute-path.yaml", []string{"input.agent_visible"}, ""},
			{"case-19-agent-visible-symlink-escapes-into-evaluator.yaml", []string{"input.agent_visible"}, ""},
			{"case-20-evaluator-reference-solution-outside-evaluator-subtree.yaml", []string{"evaluator_only.reference_solution"}, ""},
			{"case-21-evaluator-oracle-tests-escapes-package-root.yaml", []string{"evaluator_only.oracle_tests"}, ""},
			{"case-22-evaluator-answer-keys-path-traversal.yaml", []string{"evaluator_only.answer_keys"}, ""},
			{"case-23-agent-visible-missing-file.yaml", []string{"input.agent_visible"}, ""},
			// case-24 derives from valid-feature/, not valid/ -- every
			// evaluator_only.* / input.agent_visible field it sets uses a
			// "valid-feature/" prefix, so it needs the matching evaluatorRoot.
			{"case-24-replay-reference-missing-file.yaml", []string{"replay_reference"}, "valid-feature"},

			// Case 25: code review NEW-1 (code-review-20260813T232337Z-E40-F05.md).
			// evaluator_only.reference_solution resolves to
			// testdataDir/extras/evaluator/fake.patch -- a real, existing file
			// that contains an "evaluator" path segment but is not a
			// descendant of this case's actual evaluator/ subtree
			// (testdataDir/valid/evaluator/). The pre-fix segment-match
			// containment check (e40I04HasEvaluatorSegment used directly for
			// requireEvaluator) incorrectly accepts this; a genuine
			// subtree/prefix comparison against evaluatorRoot correctly
			// rejects it.
			{"case-25-evaluator-reference-solution-nested-evaluator-segment.yaml", []string{"evaluator_only.reference_solution"}, ""},

			// Cases 28-29: T-E40-F11-005 (REQ-F-009, AC-F11-25, ADR-F11-09).
			// expected_entity_graph is required-when-hierarchical (28: an
			// epic package with the block omitted) and
			// forbidden-when-flat (29: a bug package with the block
			// present).
			{"case-28-expected-entity-graph-required-for-epic.yaml", []string{"expected_entity_graph: required for family 'epic'"}, ""},
			{"case-29-expected-entity-graph-forbidden-for-flat-family.yaml", []string{"expected_entity_graph: forbidden for family \"bug\""}, ""},
		}

		for _, c := range cases {
			c := c
			t.Run(c.file, func(t *testing.T) {
				path := filepath.Join(testdataDir, c.file)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read fixture %s: %v", path, err)
				}
				pkg := e40I04ParseYAMLMap(t, data)
				family := c.family
				if family == "" {
					family = "valid"
				}
				evaluatorRoot := filepath.Join(testdataDir, family, "evaluator")
				errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
				if len(errs) == 0 {
					t.Fatalf("case %s: expected validation errors, got none", c.file)
				}
				for _, want := range c.wantAny {
					if !e40ContainsErrorMatching(errs, want) {
						t.Errorf("case %s: expected an error naming %q, got:\n%s", c.file, want, strings.Join(errs, "\n"))
					}
				}
			})
		}

		// AC-016's forbidden-field case (case-16) additionally proves each
		// of the four I-01 field names is unknown here, not just one.
		t.Run("case-16-forbidden-i01-fields.yaml_all_four_fields", func(t *testing.T) {
			path := filepath.Join(testdataDir, "case-16-forbidden-i01-fields.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s: %v", path, err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, filepath.Join(testdataDir, "valid", "evaluator"))
			// "fixture.toolchain" (not the bare "fixture" prefix, which
			// would also match an unrelated fixture.* error such as
			// fixture.fixture_id being empty) proves the nested unknown-key
			// detection specifically caught the injected toolchain block.
			for _, want := range []string{"fixture.toolchain", "p2p_sets", "reference_patch_path"} {
				if !e40ContainsErrorMatching(errs, want) {
					t.Errorf("expected an unknown-field error naming %q, got:\n%s", want, strings.Join(errs, "\n"))
				}
			}
		})

		// Positive control (AC-005's own requirement): the valid baseline
		// every case above derives from must itself pass, proving the
		// validator isn't rejecting the whole file for unrelated reasons.
		t.Run("valid_baseline_passes", func(t *testing.T) {
			path := filepath.Join(testdataDir, "valid", "package.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s: %v", path, err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, filepath.Join(testdataDir, "valid", "evaluator"))
			if len(errs) != 0 {
				t.Errorf("valid baseline fixture failed validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		// Second positive control: case-24 derives from this feature-family
		// baseline (replay_reference, all-true prelude, child_oracles_union),
		// which the bug-family valid/package.yaml above cannot cover. Proves
		// the new containment/existence checks (rework of UAT
		// uat-20260813T204821Z-E40-F05.md Finding 1) don't false-positive on
		// a legitimately-placed replay_reference (real committed seed
		// py-feature-recurring-tasks also places it under evaluator/,
		// consistent with this fixture's own placement).
		t.Run("valid_feature_baseline_passes", func(t *testing.T) {
			path := filepath.Join(testdataDir, "valid-feature", "package.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s: %v", path, err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, filepath.Join(testdataDir, "valid-feature", "evaluator"))
			if len(errs) != 0 {
				t.Errorf("valid feature baseline fixture failed validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})

		// TC-19/21 (AC-F11-19, AC-F11-21, REQ-F-007): the widened six-value
		// entity_family vocabulary accepts exactly epic|feature|task|bug|
		// change_card|tech_debt and rejects every other value, with the
		// diagnostic naming all six. Built by substituting entity_family in
		// the committed bug-family baseline rather than adding eight more
		// testdata files -- epic and task packages cannot yet be made fully
		// valid (their final_predicate.kind entries land with REQ-F-010's
		// sibling tasks), so this isolates the entity_family concern: a
		// valid family value must never itself produce an entity_family
		// error, whatever else in the package is still incomplete.
		t.Run("six_family_vocabulary", func(t *testing.T) {
			baseData, err := os.ReadFile(filepath.Join(testdataDir, "valid", "package.yaml"))
			if err != nil {
				t.Fatalf("read valid baseline: %v", err)
			}
			evaluatorRoot := filepath.Join(testdataDir, "valid", "evaluator")
			withFamily := func(literal string) map[string]interface{} {
				replaced := strings.Replace(string(baseData), "entity_family: bug", "entity_family: "+literal, 1)
				return e40I04ParseYAMLMap(t, []byte(replaced))
			}

			// e40I04HasEntityFamilyDiagnostic isolates the specific
			// "entity_family = ..." diagnostic addf emits for an unrecognized
			// family, as opposed to some other error (e.g.
			// final_predicate.kind's own message) that merely mentions the
			// substring "entity_family" in passing.
			hasEntityFamilyDiagnostic := func(errs []string) bool {
				for _, e := range errs {
					if strings.HasPrefix(e, "entity_family = ") {
						return true
					}
				}
				return false
			}

			validFamilies := []string{"epic", "feature", "task", "bug", "change_card", "tech_debt"}
			for _, family := range validFamilies {
				t.Run("valid_"+family, func(t *testing.T) {
					pkg := withFamily(family)
					errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
					if hasEntityFamilyDiagnostic(errs) {
						t.Errorf("family %q must not raise an entity_family error, got:\n%s", family, strings.Join(errs, "\n"))
					}
				})
			}

			// §7.3.5's 8-value negative set (AC-F11-19).
			negatives := []string{`sprint`, `question`, `Feature`, `""`, `null`, `123`, `[]`, `"epic "`}
			for _, literal := range negatives {
				t.Run("invalid_"+literal, func(t *testing.T) {
					pkg := withFamily(literal)
					errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
					if !hasEntityFamilyDiagnostic(errs) {
						t.Fatalf("literal %s: expected an entity_family error, got:\n%s", literal, strings.Join(errs, "\n"))
					}
					for _, want := range validFamilies {
						if !e40ContainsErrorMatching(errs, "entity_family = ", want) {
							t.Errorf("literal %s: diagnostic must name valid family %q, got:\n%s", literal, want, strings.Join(errs, "\n"))
						}
					}
				})
			}
		})

		// TC-19 (AC-F11-19, REQ-F-004/007): the family invariant's non-feature
		// branch widens from three families to five -- case-12 already proves
		// it for "bug"; this proves the two families REQ-F-007 adds (epic,
		// task) are covered by the same widened arm, not a second, forgotten
		// switch.
		t.Run("family_invariant_widened_to_epic_and_task", func(t *testing.T) {
			violatingStageMatrix := map[string]interface{}{
				"prelude": map[string]interface{}{
					"D01": map[string]interface{}{"applicable": true},
					"D02": map[string]interface{}{"applicable": false, "reason": "no product-design prelude"},
					"D03": map[string]interface{}{"applicable": false, "reason": "no product-design prelude"},
					"D04": map[string]interface{}{"applicable": false, "reason": "no product-design prelude"},
					"D05": map[string]interface{}{"applicable": false, "reason": "no product-design prelude"},
				},
				"lifecycle": map[string]interface{}{"mode": "all_dispatched", "evidence_required": true},
			}
			for _, family := range []string{"epic", "task"} {
				t.Run(family, func(t *testing.T) {
					errs := e40I04ValidateStageMatrix(violatingStageMatrix, family)
					want := fmt.Sprintf("family invariant violated: entity_family %q requires stage_matrix.prelude.D01.applicable = false (REQ-F-004)", family)
					found := false
					for _, e := range errs {
						if e == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("family %q: expected error %q, got:\n%s", family, want, strings.Join(errs, "\n"))
					}
				})
			}
		})

		// TC-20 (AC-F11-20, ADR-F11-01): the validator supports exactly one
		// schema_version; a package left at the superseded "1.0" is rejected
		// naming the field, and the current "1.1" is accepted (already
		// proven by valid_baseline_passes above, restated here as the
		// explicit positive control for this AC).
		t.Run("schema_version_single_supported_version", func(t *testing.T) {
			baseData, err := os.ReadFile(filepath.Join(testdataDir, "valid", "package.yaml"))
			if err != nil {
				t.Fatalf("read valid baseline: %v", err)
			}
			evaluatorRoot := filepath.Join(testdataDir, "valid", "evaluator")

			t.Run("old_version_rejected", func(t *testing.T) {
				old := strings.Replace(string(baseData), `schema_version: "1.1"`, `schema_version: "1.0"`, 1)
				pkg := e40I04ParseYAMLMap(t, []byte(old))
				errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
				if !e40ContainsErrorMatching(errs, "schema_version", `"1.0"`, `"1.1"`) {
					t.Errorf("expected a schema_version error naming both the found (1.0) and wanted (1.1) versions, got:\n%s", strings.Join(errs, "\n"))
				}
			})

			t.Run("third_version_rejected", func(t *testing.T) {
				third := strings.Replace(string(baseData), `schema_version: "1.1"`, `schema_version: "2.0"`, 1)
				pkg := e40I04ParseYAMLMap(t, []byte(third))
				errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
				if !e40ContainsErrorMatching(errs, "schema_version", `"2.0"`) {
					t.Errorf("expected a schema_version error naming the found version (2.0), got:\n%s", strings.Join(errs, "\n"))
				}
			})

			t.Run("current_version_accepted", func(t *testing.T) {
				pkg := e40I04ParseYAMLMap(t, baseData)
				errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
				if e40ContainsErrorMatching(errs, "schema_version") {
					t.Errorf("current schema_version must not raise an error, got:\n%s", strings.Join(errs, "\n"))
				}
			})
		})

		// Cases 26-27: round-4 code review NEW-5/NEW-6
		// (code-review-20260814T011910Z-E40-F05.md). Both are real,
		// self-contained package directories under testdata/e40_i04/ -- unlike
		// cases 01-25, which share testdataDir as packageDir and a
		// valid|valid-feature subdirectory as evaluatorRoot, these two set
		// packageDir to their own directory and evaluatorRoot to
		// filepath.Join(packageDir, "evaluator"), exactly mirroring the
		// index_and_registered_packages loop's real-package call convention
		// above -- required because the bug each case proves is specifically
		// about how a package's OWN evaluator/ entry resolves relative to its
		// OWN directory.
		t.Run("case-26-evaluator-symlink-collapses-onto-package-root", func(t *testing.T) {
			pkgDir := filepath.Join(testdataDir, "collapsed-evaluator-root")
			path := filepath.Join(pkgDir, "package.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s: %v", path, err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			// This fixture's evaluator/ entry is a committed `evaluator -> .`
			// symlink that resolves back onto pkgDir itself. Before the NEW-5
			// fix, e40I04IsWithin(resolved, evaluatorRoot) resolved
			// evaluatorRoot down to pkgDir and then treated every path in the
			// package as trivially "within evaluator/", so
			// evaluator_only.reference_solution (a real file at pkgDir's own
			// root, reached via the collapsed evaluator/ symlink) incorrectly
			// passed the requireEvaluator containment check.
			errs := e40I04ValidateScenarioPackage(pkg, index, pkgDir, filepath.Join(pkgDir, "evaluator"))
			if len(errs) == 0 {
				t.Fatalf("case-26: expected validation errors (evaluator/ collapses onto the package root), got none")
			}
			if !e40ContainsErrorMatching(errs, "evaluator_only.reference_solution") {
				t.Errorf("case-26: expected an error naming evaluator_only.reference_solution, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("case-27-agent-visible-symlink-redirect-into-evaluator", func(t *testing.T) {
			pkgDir := filepath.Join(testdataDir, "redirect-evaluator-root")
			path := filepath.Join(pkgDir, "package.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s: %v", path, err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			// This fixture's evaluator/ entry is a committed `evaluator ->
			// private` symlink to a differently-named real sibling directory.
			// input.agent_visible: "evaluator/secret" canonicalizes to
			// pkgDir/private/secret -- inside the same canonical subtree
			// evaluatorRoot resolves to. Before the NEW-6 fix, the
			// forbidEvaluator direction checked e40I04HasEvaluatorSegment on
			// the already-resolved path, where the literal segment
			// "evaluator" no longer appears (it reads .../private/secret), so
			// this leak silently passed.
			errs := e40I04ValidateScenarioPackage(pkg, index, pkgDir, filepath.Join(pkgDir, "evaluator"))
			if len(errs) == 0 {
				t.Fatalf("case-27: expected validation errors (agent_visible leaks into evaluator/ via a symlinked sibling), got none")
			}
			if !e40ContainsErrorMatching(errs, "input.agent_visible") {
				t.Errorf("case-27: expected an error naming input.agent_visible, got:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	// Rework of UAT report uat-20260813T204821Z-E40-F05.md Finding 3
	// (MEDIUM): REQ-F-002 requires scenario_id to be unique across the
	// whole corpus, but the validator previously checked only each
	// package's scenario_id independently against a kebab-case regex, never
	// accumulating seen ids across packages. e40I04FindDuplicateScenarioIDs
	// is the extracted, directly-testable unit the "index_and_registered_
	// packages" subtest above now also calls while iterating index.Scenarios.
	t.Run("scenario_id_uniqueness", func(t *testing.T) {
		t.Run("duplicate_across_packages_is_rejected", func(t *testing.T) {
			entries := []e40I04ScenarioIDEntry{
				{Rel: "packages/py-bug-a", ScenarioID: "py-bug-due-date-boundary"},
				{Rel: "packages/py-change-a", ScenarioID: "py-change-priority-scale"},
				{Rel: "packages/py-bug-b", ScenarioID: "py-bug-due-date-boundary"},
			}
			errs := e40I04FindDuplicateScenarioIDs(entries)
			if len(errs) != 1 {
				t.Fatalf("want exactly one duplicate-scenario_id violation, got %d: %v", len(errs), errs)
			}
			for _, want := range []string{"py-bug-due-date-boundary", "packages/py-bug-a", "packages/py-bug-b"} {
				if !strings.Contains(errs[0], want) {
					t.Errorf("duplicate scenario_id message missing %q, got: %s", want, errs[0])
				}
			}
		})

		t.Run("distinct_ids_pass", func(t *testing.T) {
			entries := []e40I04ScenarioIDEntry{
				{Rel: "packages/py-bug-a", ScenarioID: "py-bug-due-date-boundary"},
				{Rel: "packages/py-change-a", ScenarioID: "py-change-priority-scale"},
			}
			if errs := e40I04FindDuplicateScenarioIDs(entries); len(errs) != 0 {
				t.Errorf("want zero violations for distinct scenario_ids, got: %v", errs)
			}
		})
	})
}

// e40I04ReadScenarioIndex reads and parses the real committed scenarios.yaml
// into the typed struct field-access callers use.
func e40I04ReadScenarioIndex(t *testing.T, path string) *e40I04ScenarioIndex {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scenario index %s: %v", path, err)
	}
	var index e40I04ScenarioIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		t.Fatalf("parse scenario index %s: %v", path, err)
	}
	return &index
}

// e40I04ValidateIndexKnownFields applies AC-T1's unknown-field-rejecting
// unmarshalling to the index (scenarios.yaml), the same discipline
// e40I04ValidateScenarioPackage applies to every package.yaml. Field access
// stays on the typed e40I04ScenarioIndex struct (e40I04ReadScenarioIndex);
// this function only guards against a stray or forbidden field the typed
// decode would otherwise ignore.
func e40I04ValidateIndexKnownFields(index map[string]interface{}) []string {
	var errs []string
	e40I04CheckUnknownKeys(index, []string{"schema_version", "fixtures", "adapters", "scenarios"}, "", &errs)

	if fixtures, ok := e40I04AsMap(index["fixtures"]); ok {
		for id, v := range fixtures {
			if m, ok := e40I04AsMap(v); ok {
				e40I04CheckUnknownKeys(m, []string{"submodule_path"}, "fixtures."+id, &errs)
			}
		}
	}
	if adapters, ok := e40I04AsMap(index["adapters"]); ok {
		for name, v := range adapters {
			if m, ok := e40I04AsMap(v); ok {
				e40I04CheckUnknownKeys(m, []string{"path", "version"}, "adapters."+name, &errs)
			}
		}
	}
	return errs
}

// e40I04AdapterDescriptor decodes an adapter.yaml descriptor (REQ-F-006).
type e40I04AdapterDescriptor struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// e40I04ReadAdapterDescriptor reads and parses a real committed
// adapter.yaml, used to cross-check its name/version against the pin
// registered in scenarios.yaml's adapters map.
func e40I04ReadAdapterDescriptor(t *testing.T, path string) e40I04AdapterDescriptor {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read adapter descriptor %s: %v", path, err)
	}
	var descriptor e40I04AdapterDescriptor
	if err := yaml.Unmarshal(data, &descriptor); err != nil {
		t.Fatalf("parse adapter descriptor %s: %v", path, err)
	}
	return descriptor
}

// e40I04ParseYAMLMap decodes package.yaml content into a generic
// map[string]interface{} tree. Validation runs against this generic tree
// (rather than a typed struct) so e40I04CheckUnknownKeys can report exactly
// which field is unrecognized -- the mechanism behind AC-T1's
// "strict/unknown-field-rejecting unmarshalling" for every nested object in
// the schema, including REQ-F-001's forbidden I-01 fields.
func e40I04ParseYAMLMap(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse package YAML: %v", err)
	}
	return m
}

// e40I04ListPackageDirs lists the scenario_id directories under
// bench/scenarios/packages/. A missing directory (expected before any real
// seed content lands) is not an error -- it is treated as zero entries.
func e40I04ListPackageDirs(t *testing.T, packagesRoot string) []string {
	t.Helper()
	entries, err := os.ReadDir(packagesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", packagesRoot, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}

// e40I04CheckUnknownKeys appends one error per key in m that is not present in
// allowed, labeled with its dotted path. This is the manual equivalent of
// yaml.Decoder.KnownFields(true) applied to a generic map tree, giving full
// control over the reported field path (AC-T2's "names the specific field").
func e40I04CheckUnknownKeys(m map[string]interface{}, allowed []string, label string, errs *[]string) {
	allowedSet := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		allowedSet[k] = true
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if allowedSet[k] {
			continue
		}
		path := k
		if label != "" {
			path = label + "." + k
		}
		*errs = append(*errs, fmt.Sprintf("%s: unknown field", path))
	}
}

func e40I04AsMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

func e40I04AsStringSlice(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// e40I04ValidateStringSlice rejects mixed-type YAML arrays rather than
// silently dropping their non-string elements while retaining the strings.
func e40I04ValidateStringSlice(v interface{}, label string) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return []string{fmt.Sprintf("%s: must be an array of strings", label)}
	}
	var errs []string
	for i, item := range list {
		if _, ok := item.(string); !ok {
			errs = append(errs, fmt.Sprintf("%s[%d]: must be a string", label, i))
		}
	}
	return errs
}

// e40I04AsNumber reads a YAML scalar as a float64 regardless of whether the
// decoder produced an int or a float, returning ok=false if the key is
// absent or not numeric -- so callers can distinguish "missing" from
// "present but non-positive" when they need to (both are rejected here,
// naming the same field either way).
func e40I04AsNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// e40I04RealishPath resolves path (which must already be absolute or will
// be made absolute relative to the working directory) as far as the
// filesystem allows: it follows every symlink along the way via
// filepath.EvalSymlinks. When the full path does not exist yet -- a
// malformed-package case fixture deliberately references a missing file,
// or a containment check must run before anything is written -- it walks
// up to the nearest existing ancestor, resolves that ancestor's symlinks,
// and re-appends the non-existent trailing segments unresolved. This is
// what makes the containment checks below canonicalized-path checks
// (REQ-F-009's leak-surface rule, UAT report uat-20260813T204821Z-E40-F05.md
// Finding 1) rather than the lexical string checks they replace: a symlink
// anywhere in the path -- including one pointing outside the package
// entirely -- is followed to its real target before any containment
// comparison happens, and existence is verified separately by callers that
// need it.
func e40I04RealishPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	dir := filepath.Dir(abs)
	if dir == abs {
		return abs
	}
	return filepath.Join(e40I04RealishPath(dir), filepath.Base(abs))
}

// e40I04IsWithin reports whether resolved (already canonicalized via
// e40I04RealishPath) falls inside root, after canonicalizing root the same
// way. Used both for "must stay inside the package directory" (packageDir
// as root) and "must stay inside the package's evaluator/ subtree"
// (packageDir's evaluator/ segment as root) containment checks.
func e40I04IsWithin(resolved, root string) bool {
	rootResolved := e40I04RealishPath(root)
	rel, err := filepath.Rel(rootResolved, resolved)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Round-4 rework (code-review-20260814T011910Z-E40-F05.md NEW-5/NEW-6):
// e40I04HasEvaluatorSegment -- the any-segment literal-name check that used
// to back the forbidEvaluator direction below -- has been removed. It
// operated on segment names in the already-resolved path, so a package
// whose evaluator/ entry is itself a symlink to a differently-named real
// sibling (e.g. `evaluator -> private`) could redirect a declared
// input.agent_visible: "evaluator/secret" to a real path
// (packageDir/private/secret) that no longer contains the literal segment
// "evaluator" anywhere, silently defeating the leak check (NEW-6). Both
// directions -- requireEvaluator (evaluator_only.* must resolve INSIDE the
// package's own evaluator/ subtree) and forbidEvaluator
// (input.agent_visible must NOT resolve inside it, REQ-F-009's leak-surface
// rule) -- now go through the same canonical-identity comparison,
// e40I04IsWithin(resolved, evaluatorRootResolved), inside
// e40I04CheckPathField below, instead of two different heuristics.

// e40I04CheckPathField validates one package-relative path field end to
// end (REQ-F-009/REQ-F-015, rework of UAT report
// uat-20260813T204821Z-E40-F05.md Findings 1 and 2, code review
// NEW-1/NEW-4 of code-review-20260813T232337Z-E40-F05.md, and round-4 code
// review NEW-5/NEW-6 of code-review-20260814T011910Z-E40-F05.md):
//   - rejects rawPath given as an absolute path -- an absolute path would
//     otherwise sidestep every containment check below;
//   - resolves rawPath against packageDir following symlinks
//     (e40I04RealishPath) and rejects any resolution that escapes
//     packageDir, whether via lexical ".." traversal or a symlink pointing
//     outside the package -- a canonicalized containment check, not a
//     lexical one;
//   - when requireEvaluator or forbidEvaluator is true, resolves
//     evaluatorRoot the same symlink-following way and rejects outright
//     (as a caller/layout error, not a normal per-field violation) the
//     degenerate case where the resolved evaluatorRoot collapses onto the
//     resolved packageDir itself -- e.g. a literal `evaluator -> .`
//     symlink at the package root -- rather than silently letting that
//     collapsed root stand in as a containment boundary for every field
//     checked against it. Mirrors bench/scripts/admit-scenario.sh's
//     resolve_scoped() `root_real == base_real` check exactly (NEW-5:
//     without this, requireEvaluator would treat the whole package as
//     "inside evaluator/", trivially passing every evaluator_only.* field
//     regardless of where it actually points);
//   - when requireEvaluator is true (and evaluatorRoot did not collapse),
//     rejects a resolution that is not a genuine descendant of
//     evaluatorRoot (e40I04IsWithin(resolved, evaluatorRootResolved) -- a
//     real subtree/prefix comparison against the package's own evaluator/
//     directory, not merely "contains an 'evaluator' path segment
//     somewhere", which let evaluator_only.reference_solution:
//     extras/evaluator/fake.patch (a decoy evaluator/ directory elsewhere
//     in the package) incorrectly pass (NEW-1);
//   - when forbidEvaluator is true (and evaluatorRoot did not collapse),
//     rejects a resolution that IS a genuine descendant of evaluatorRoot
//     (input.agent_visible's leak-surface rule, REQ-F-009, AC-004) --
//     using the same e40I04IsWithin(resolved, evaluatorRootResolved)
//     canonical-identity comparison requireEvaluator uses, rather than the
//     literal any-"evaluator"-segment check this replaced (NEW-6: that
//     check operated on segment names in the resolved path, so a package
//     whose evaluator/ entry is a symlink to a differently-named real
//     sibling -- e.g. `evaluator -> private` -- could redirect
//     input.agent_visible: "evaluator/secret" to a real path
//     (packageDir/private/secret) that resolves inside the same canonical
//     subtree evaluatorRoot resolves to, yet no longer contains the
//     literal segment "evaluator" anywhere, silently defeating the leak
//     check). evaluatorRoot must be non-empty whenever either flag is true
//     -- omitting it is a caller bug, reported as a validation error
//     rather than silently skipped;
//   - when requireExists is true, rejects a resolution that does not exist
//     on disk (AC-001's "every referenced ... path exists").
//
// Returns nil when rawPath is empty -- callers check presence/emptiness
// separately, since "empty" and "escapes/missing" are different failure
// messages callers report independently.
//
// packageDir must be non-empty: every real call site (the
// index_and_registered_packages loop, the malformed_package_cases table,
// and both baseline positive controls) always resolves a real directory
// before calling this function. A caller that reaches this function with
// packageDir == "" gets a loud validation error naming the field instead
// of the pre-rework silent lexical-only fallback this replaced (NEW-4) --
// there is no real call path that needs or exercises that fallback.
func e40I04CheckPathField(fieldLabel, rawPath, packageDir, evaluatorRoot string, requireEvaluator, forbidEvaluator, requireExists bool) []string {
	if strings.TrimSpace(rawPath) == "" {
		return nil
	}
	if filepath.IsAbs(rawPath) {
		return []string{fmt.Sprintf("%s = %q must be a package-relative path, not absolute", fieldLabel, rawPath)}
	}
	if filepath.Clean(rawPath) == "." {
		return []string{fmt.Sprintf("%s = %q must name a file below its package directory", fieldLabel, rawPath)}
	}
	if packageDir == "" {
		return []string{fmt.Sprintf("%s: internal error: packageDir must not be empty to validate %q (cannot verify containment)", fieldLabel, rawPath)}
	}

	packageDirResolved := e40I04RealishPath(packageDir)
	resolved := e40I04RealishPath(filepath.Join(packageDir, rawPath))
	if !e40I04IsWithin(resolved, packageDirResolved) {
		return []string{fmt.Sprintf("%s = %q resolves outside its package directory (path traversal or symlink escape)", fieldLabel, rawPath)}
	}

	var errs []string
	if requireEvaluator || forbidEvaluator {
		if evaluatorRoot == "" {
			errs = append(errs, fmt.Sprintf("%s: internal error: evaluatorRoot must not be empty when requireEvaluator or forbidEvaluator is true", fieldLabel))
		} else if evaluatorRootResolved := e40I04RealishPath(evaluatorRoot); evaluatorRootResolved == packageDirResolved {
			// NEW-5/NEW-6: the package's evaluator/ entry collapses onto
			// the package directory itself (e.g. `evaluator -> .`).
			// Rejected here for both directions, mirroring resolve_scoped's
			// root_real == base_real check, instead of letting the
			// collapsed root serve as a trivially-satisfied (requireEvaluator)
			// or trivially-violated (forbidEvaluator) containment boundary.
			errs = append(errs, fmt.Sprintf("%s: internal error: the package's evaluator/ subtree collapses onto the package directory itself (%q resolves to %q) -- invalid package layout", fieldLabel, evaluatorRoot, evaluatorRootResolved))
		} else {
			if requireEvaluator && !e40I04IsWithin(resolved, evaluatorRootResolved) {
				errs = append(errs, fmt.Sprintf("%s = %q must resolve under the package's evaluator/ subtree", fieldLabel, rawPath))
			}
			if forbidEvaluator && e40I04IsWithin(resolved, evaluatorRootResolved) {
				errs = append(errs, fmt.Sprintf("%s = %q resolves inside evaluator/ (leak surface, REQ-F-009)", fieldLabel, rawPath))
			}
		}
	}
	if requireExists && !e40FileExists(resolved) {
		errs = append(errs, fmt.Sprintf("%s = %q does not exist", fieldLabel, rawPath))
	}
	return errs
}

// e40I04ScenarioIDEntry pairs a package's declared scenario_id with the
// index-relative package path that declared it, the input
// e40I04FindDuplicateScenarioIDs scans for corpus-wide duplicates.
type e40I04ScenarioIDEntry struct {
	Rel        string
	ScenarioID string
}

// e40I04FindDuplicateScenarioIDs scans entries in order and returns one
// message per duplicate scenario_id, naming the id and both colliding
// package paths (REQ-F-002's corpus-wide uniqueness invariant). The
// pre-rework validator checked each package's scenario_id independently
// against a kebab-case regex but never accumulated seen ids across the
// corpus, so a duplicate was never mechanically caught (UAT report
// uat-20260813T204821Z-E40-F05.md Finding 3).
func e40I04FindDuplicateScenarioIDs(entries []e40I04ScenarioIDEntry) []string {
	seen := make(map[string]string, len(entries))
	var errs []string
	for _, e := range entries {
		if e.ScenarioID == "" {
			continue
		}
		if prev, ok := seen[e.ScenarioID]; ok {
			errs = append(errs, fmt.Sprintf("scenario_id %q is declared by both %s and %s, want corpus-wide uniqueness (REQ-F-002)", e.ScenarioID, prev, e.Rel))
			continue
		}
		seen[e.ScenarioID] = e.Rel
	}
	return errs
}

// e40I04ValidateScenarioPackage applies the full I-04 package.yaml field
// inventory (REQ-F-002/003/004/005/006/008/009/010/011, REQ-F-015's
// rejection cases) to a generically-decoded package tree, returning one
// description per violation. index is the real, committed scenarios.yaml
// (REQ-F-001/005: fixture_id and adapter.name resolution). packageDir is
// the directory package-relative paths (input.agent_visible,
// evaluator_only.*) resolve against. evaluatorRoot is the directory
// evaluator_only.* fields must resolve inside (code review NEW-1,
// code-review-20260813T232337Z-E40-F05.md): for a real package this is
// always filepath.Join(packageDir, "evaluator") (evaluator/ is a genuine
// immediate child), but this feature's shared testdata layout reuses one
// of two baseline fixture directories (valid/, valid-feature/) for every
// malformed-case file, so callers validating testdata compute a different
// evaluatorRoot than packageDir/evaluator -- passed explicitly rather than
// derived here so this function never has to know about that test-only
// layout.
func e40I04ValidateScenarioPackage(pkg map[string]interface{}, index *e40I04ScenarioIndex, packageDir, evaluatorRoot string) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	e40I04CheckUnknownKeys(pkg, []string{
		"schema_version", "scenario_id", "scenario_version", "entity_family",
		"stage_matrix", "fixture", "adapter", "toolchain_identity", "input",
		"replay_reference", "evaluator_only", "final_predicate", "resource_policy",
		"admission", "expected_entity_graph",
	}, "", &errs)

	schemaVersion, _ := pkg["schema_version"].(string)
	if schemaVersion != e40I04SupportedSchemaVersion {
		addf("schema_version = %q, want %q", schemaVersion, e40I04SupportedSchemaVersion)
	}

	scenarioID, _ := pkg["scenario_id"].(string)
	if !e40I04ScenarioIDPattern.MatchString(scenarioID) {
		addf("scenario_id = %q, want a unique lowercase-kebab identity", scenarioID)
	}

	if v, ok := e40I04AsNumber(pkg["scenario_version"]); !ok || v < 1 {
		addf("scenario_version: missing or not a positive integer")
	}

	family, _ := pkg["entity_family"].(string)
	if !e40I04ValidEntityFamilies[family] {
		addf("entity_family = %q, want one of epic|feature|task|bug|change_card|tech_debt", family)
	}

	errs = append(errs, e40I04ValidateFixtureBlock(pkg["fixture"], index)...)
	errs = append(errs, e40I04ValidateAdapterBlock(pkg["adapter"], index)...)

	errs = append(errs, e40I04ValidateStageMatrix(pkg["stage_matrix"], family)...)

	topToolchain, tcErrs := e40I04ValidateToolchainIdentity(pkg["toolchain_identity"], "toolchain_identity")
	errs = append(errs, tcErrs...)

	var agentVisible string
	if inputMap, ok := e40I04AsMap(pkg["input"]); !ok {
		addf("input: missing or not an object")
	} else {
		e40I04CheckUnknownKeys(inputMap, []string{"agent_visible"}, "input", &errs)
		agentVisible, _ = inputMap["agent_visible"].(string)
		if strings.TrimSpace(agentVisible) == "" {
			addf("input.agent_visible: empty")
		}
	}

	replayRef, hasReplay := pkg["replay_reference"]
	if hasReplay {
		if family != "feature" {
			addf("replay_reference present but entity_family = %q, want \"feature\" (replay_reference is feature-only, REQ-F-009)", family)
		}
		if s, ok := replayRef.(string); !ok || strings.TrimSpace(s) == "" {
			addf("replay_reference: empty")
		} else {
			// Finding 2 (UAT report uat-20260813T204821Z-E40-F05.md):
			// replay_reference was previously checked for shape only,
			// never resolved against the filesystem.
			errs = append(errs, e40I04CheckPathField("replay_reference", s, packageDir, "", false, false, true)...)
		}
	} else if family == "feature" {
		addf("replay_reference: required for entity_family \"feature\"")
	}

	// REQ-F-009/ADR-F11-09: expected_entity_graph is required for
	// root_family: epic, optional for root_family: feature, and forbidden
	// (the key itself, not merely a populated block) for task, bug,
	// change_card, and tech_debt.
	graphRaw, hasGraph := pkg["expected_entity_graph"]
	errs = append(errs, e40I04ValidateExpectedEntityGraph(graphRaw, hasGraph, family)...)

	// Finding 1 (UAT report uat-20260813T204821Z-E40-F05.md): the leak-surface
	// check must be canonicalized-path containment (resolve symlinks, reject
	// absolute/traversal paths), not the lexical string check this replaced.
	// Finding 2: agent_visible existence was never verified either. evaluatorRoot
	// is passed through here (round-4 code review NEW-6,
	// code-review-20260814T011910Z-E40-F05.md) rather than "" -- the
	// forbidEvaluator direction now checks canonical-identity containment
	// against the package's real evaluator/ subtree, the same way the
	// requireEvaluator direction below does, instead of a literal
	// segment-name check that a symlinked evaluator/ could defeat.
	errs = append(errs, e40I04CheckPathField("input.agent_visible", agentVisible, packageDir, evaluatorRoot, false, true, true)...)

	errs = append(errs, e40I04ValidateEvaluatorOnly(pkg["evaluator_only"], packageDir, evaluatorRoot)...)
	errs = append(errs, e40I04ValidateFinalPredicate(pkg["final_predicate"], family)...)
	errs = append(errs, e40I04ValidateResourcePolicy(pkg["resource_policy"])...)

	if adm, present := pkg["admission"]; present {
		errs = append(errs, e40I04ValidateAdmission(adm, topToolchain)...)
	}

	return errs
}

func e40I04ValidateFixtureBlock(v interface{}, index *e40I04ScenarioIndex) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"fixture: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"fixture_id", "submodule_path", "base_sha"}, "fixture", &errs)

	fixtureID, _ := m["fixture_id"].(string)
	if strings.TrimSpace(fixtureID) == "" {
		errs = append(errs, "fixture.fixture_id: empty")
	} else if _, ok := index.Fixtures[fixtureID]; !ok {
		errs = append(errs, fmt.Sprintf("fixture.fixture_id = %q not registered in scenarios.yaml fixtures", fixtureID))
	}

	if s, _ := m["submodule_path"].(string); strings.TrimSpace(s) == "" {
		errs = append(errs, "fixture.submodule_path: empty")
	}
	if s, _ := m["base_sha"].(string); strings.TrimSpace(s) == "" {
		errs = append(errs, "fixture.base_sha: empty")
	}
	return errs
}

func e40I04ValidateAdapterBlock(v interface{}, index *e40I04ScenarioIndex) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"adapter: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"name", "version"}, "adapter", &errs)

	name, _ := m["name"].(string)
	if strings.TrimSpace(name) == "" {
		errs = append(errs, "adapter.name: empty")
	} else if _, ok := index.Adapters[name]; !ok {
		errs = append(errs, fmt.Sprintf("adapter.name = %q not registered in scenarios.yaml adapters", name))
	}
	if s, _ := m["version"].(string); strings.TrimSpace(s) == "" {
		errs = append(errs, "adapter.version: empty")
	}
	return errs
}

// e40I04ValidateStageMatrix checks REQ-F-003 (every prelude stage declares
// an explicit boolean, with a reason required when false) and REQ-F-004
// (the family invariant, checked here since it spans both the family and
// every prelude stage), returning one description per violation.
func e40I04ValidateStageMatrix(v interface{}, family string) []string {
	var errs []string
	preludeStates := map[string]bool{}

	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"stage_matrix: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"prelude", "lifecycle"}, "stage_matrix", &errs)

	prelude, ok := e40I04AsMap(m["prelude"])
	if !ok {
		errs = append(errs, "stage_matrix.prelude: missing or not an object")
	} else {
		e40I04CheckUnknownKeys(prelude, e40I04PreludeStages, "stage_matrix.prelude", &errs)
		for _, stageName := range e40I04PreludeStages {
			label := "stage_matrix.prelude." + stageName
			stage, ok := e40I04AsMap(prelude[stageName])
			if !ok {
				errs = append(errs, fmt.Sprintf("%s: missing or not an object (must declare an explicit applicable boolean)", label))
				continue
			}
			e40I04CheckUnknownKeys(stage, []string{"applicable", "reason"}, label, &errs)
			applicableRaw, present := stage["applicable"]
			if !present {
				errs = append(errs, fmt.Sprintf("%s: missing explicit applicable boolean (REQ-F-003)", label))
				continue
			}
			applicable, ok := applicableRaw.(bool)
			if !ok {
				errs = append(errs, fmt.Sprintf("%s: applicable is not a boolean", label))
				continue
			}
			preludeStates[stageName] = applicable
			if !applicable {
				reason, _ := stage["reason"].(string)
				if strings.TrimSpace(reason) == "" {
					errs = append(errs, fmt.Sprintf("%s.reason: required when applicable is false (REQ-F-003)", label))
				}
			}
		}

		if len(preludeStates) == len(e40I04PreludeStages) {
			switch family {
			case "feature":
				for _, stageName := range e40I04PreludeStages {
					if !preludeStates[stageName] {
						errs = append(errs, fmt.Sprintf("family invariant violated: entity_family \"feature\" requires stage_matrix.prelude.%s.applicable = true (REQ-F-004)", stageName))
					}
				}
			case "bug", "change_card", "tech_debt", "epic", "task":
				for _, stageName := range e40I04PreludeStages {
					if preludeStates[stageName] {
						errs = append(errs, fmt.Sprintf("family invariant violated: entity_family %q requires stage_matrix.prelude.%s.applicable = false (REQ-F-004)", family, stageName))
					}
				}
			}
		}
	}

	lifecycle, ok := e40I04AsMap(m["lifecycle"])
	if !ok {
		errs = append(errs, fmt.Sprintf("stage_matrix.lifecycle: must be an object with mode and evidence_required, not %T (ADR-F05-02: no enumerated status list)", m["lifecycle"]))
	} else {
		e40I04CheckUnknownKeys(lifecycle, []string{"mode", "evidence_required"}, "stage_matrix.lifecycle", &errs)
		mode, _ := lifecycle["mode"].(string)
		if mode != "all_dispatched" {
			errs = append(errs, fmt.Sprintf("stage_matrix.lifecycle.mode = %q, want \"all_dispatched\"", mode))
		}
		evReq, ok := lifecycle["evidence_required"].(bool)
		if !ok || !evReq {
			errs = append(errs, "stage_matrix.lifecycle.evidence_required must be true")
		}
	}

	return errs
}

// e40I04ValidateToolchainIdentity checks the REQ-F-008 shape (ordered list of
// {key, value} pairs) and returns the parsed list alongside any errors so
// callers (the top-level field and admission.toolchain_identity) can
// compare the two encodings for AC-019.
func e40I04ValidateToolchainIdentity(v interface{}, label string) ([][2]string, []string) {
	var errs []string
	list, ok := v.([]interface{})
	if !ok {
		return nil, []string{fmt.Sprintf("%s: missing or not a list", label)}
	}
	if len(list) == 0 {
		errs = append(errs, fmt.Sprintf("%s: empty, want at least one ordered key/value pair", label))
	}
	pairs := make([][2]string, 0, len(list))
	for i, item := range list {
		m, ok := e40I04AsMap(item)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s[%d]: not an object", label, i))
			continue
		}
		e40I04CheckUnknownKeys(m, []string{"key", "value"}, fmt.Sprintf("%s[%d]", label, i), &errs)
		key, _ := m["key"].(string)
		value, _ := m["value"].(string)
		if strings.TrimSpace(key) == "" {
			errs = append(errs, fmt.Sprintf("%s[%d].key: empty", label, i))
		}
		pairs = append(pairs, [2]string{key, value})
	}
	return pairs, errs
}

// e40I04ValidateEvaluatorOnly checks REQ-F-009's evaluator_only block: its
// three path-typed fields (reference_solution, oracle_tests[], answer_keys[])
// MUST all resolve under the package's own evaluator/ subtree (spec.md field
// table: "all under evaluator/"), and MUST exist on disk. This is
// deliberately where AC-001's "predicate-operand path" existence check
// lives: of the final_predicate operands (REQ-F-010's vocabulary table), the
// id-list fields (f2p_test_ids, acceptance_test_ids, integration_test_ids,
// child_oracles) are normalized test/oracle identifiers, not paths, and
// p2p_selection.include is explicitly fixture-relative (REQ-F-017) --
// checking its on-disk existence or containment would require a populated
// fixture submodule, which REQ-NF-003/AC-016 forbid (and which AC-T1's
// "identical results in both submodule states" makes structurally
// impossible for this validator to depend on) -- so it is deliberately left
// shape-only, unlike evaluator_only's own paths.
//
// Pre-rework (UAT report uat-20260813T204821Z-E40-F05.md Finding 1
// sibling-sweep), none of these three fields had any containment check at
// all -- reference_solution and oracle_tests were checked only for
// existence (so a real file anywhere on disk satisfied them), and
// answer_keys was not validated in any way beyond being a recognized field
// name. evaluatorRoot (see e40I04ValidateScenarioPackage's doc comment) is
// the directory each field must genuinely resolve inside (code review
// NEW-1, code-review-20260813T232337Z-E40-F05.md).
func e40I04ValidateEvaluatorOnly(v interface{}, packageDir, evaluatorRoot string) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"evaluator_only: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"reference_solution", "oracle_tests", "answer_keys"}, "evaluator_only", &errs)

	refSolution, _ := m["reference_solution"].(string)
	if strings.TrimSpace(refSolution) == "" {
		errs = append(errs, "evaluator_only.reference_solution: empty")
	} else {
		errs = append(errs, e40I04CheckPathField("evaluator_only.reference_solution", refSolution, packageDir, evaluatorRoot, true, false, true)...)
	}

	oracleTests := e40I04AsStringSlice(m["oracle_tests"])
	errs = append(errs, e40I04ValidateStringSlice(m["oracle_tests"], "evaluator_only.oracle_tests")...)
	if len(oracleTests) == 0 {
		errs = append(errs, "evaluator_only.oracle_tests: empty")
	}
	for i, p := range oracleTests {
		errs = append(errs, e40I04CheckPathField(fmt.Sprintf("evaluator_only.oracle_tests[%d]", i), p, packageDir, evaluatorRoot, true, false, true)...)
	}

	// answer_keys is permitted to be empty (a judge answer key isn't always
	// needed), but every entry that IS present is subject to the same
	// containment and existence discipline as reference_solution/
	// oracle_tests -- previously it had neither.
	answerKeys := e40I04AsStringSlice(m["answer_keys"])
	errs = append(errs, e40I04ValidateStringSlice(m["answer_keys"], "evaluator_only.answer_keys")...)
	for i, p := range answerKeys {
		errs = append(errs, e40I04CheckPathField(fmt.Sprintf("evaluator_only.answer_keys[%d]", i), p, packageDir, evaluatorRoot, true, false, true)...)
	}

	return errs
}

func e40I04ValidateFinalPredicate(v interface{}, family string) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"final_predicate: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{
		"kind", "p2p_selection", "f2p_test_ids", "acceptance_test_ids",
		"rule", "max_remaining", "integration_test_ids", "child_oracles",
	}, "final_predicate", &errs)

	kind, _ := m["kind"].(string)
	requiredFamily, known := e40I04FinalPredicateFamilies[kind]
	if !known {
		errs = append(errs, fmt.Sprintf("final_predicate.kind = %q, want one of f2p_p2p|acceptance_tests|p2p_plus_rule_drop|child_oracles_union|task_acceptance_tests|descendant_oracles_union (REQ-F-010, ADR-F11-04)", kind))
	} else if requiredFamily != family {
		errs = append(errs, fmt.Sprintf("final_predicate.kind = %q not permitted for entity_family %q, want %q", kind, family, requiredFamily))
	}

	p2p, ok := e40I04AsMap(m["p2p_selection"])
	if !ok {
		errs = append(errs, "final_predicate.p2p_selection: missing or not an object (REQ-F-017)")
	} else {
		e40I04CheckUnknownKeys(p2p, []string{"include", "exclude_test_ids"}, "final_predicate.p2p_selection", &errs)
		errs = append(errs, e40I04ValidateStringSlice(p2p["include"], "final_predicate.p2p_selection.include")...)
		errs = append(errs, e40I04ValidateStringSlice(p2p["exclude_test_ids"], "final_predicate.p2p_selection.exclude_test_ids")...)
		if len(e40I04AsStringSlice(p2p["include"])) == 0 {
			errs = append(errs, "final_predicate.p2p_selection.include: empty")
		}
	}

	switch kind {
	case "f2p_p2p":
		errs = append(errs, e40I04ValidateStringSlice(m["f2p_test_ids"], "final_predicate.f2p_test_ids")...)
		if len(e40I04AsStringSlice(m["f2p_test_ids"])) == 0 {
			errs = append(errs, "final_predicate.f2p_test_ids: empty, required for kind f2p_p2p")
		}
	case "acceptance_tests":
		errs = append(errs, e40I04ValidateStringSlice(m["acceptance_test_ids"], "final_predicate.acceptance_test_ids")...)
		if len(e40I04AsStringSlice(m["acceptance_test_ids"])) == 0 {
			errs = append(errs, "final_predicate.acceptance_test_ids: empty, required for kind acceptance_tests")
		}
	case "p2p_plus_rule_drop":
		rule, _ := m["rule"].(string)
		if strings.TrimSpace(rule) == "" {
			errs = append(errs, "final_predicate.rule: empty, required for kind p2p_plus_rule_drop")
		}
		if n, ok := e40I04AsNumber(m["max_remaining"]); !ok || n < 0 {
			errs = append(errs, "final_predicate.max_remaining: missing or negative, required for kind p2p_plus_rule_drop")
		}
	case "child_oracles_union":
		errs = append(errs, e40I04ValidateStringSlice(m["integration_test_ids"], "final_predicate.integration_test_ids")...)
		errs = append(errs, e40I04ValidateStringSlice(m["child_oracles"], "final_predicate.child_oracles")...)
		if len(e40I04AsStringSlice(m["integration_test_ids"])) == 0 {
			errs = append(errs, "final_predicate.integration_test_ids: empty, required for kind child_oracles_union")
		}
		if len(e40I04AsStringSlice(m["child_oracles"])) == 0 {
			errs = append(errs, "final_predicate.child_oracles: empty, required for kind child_oracles_union")
		}
	case "descendant_oracles_union":
		errs = append(errs, e40I04ValidateStringSlice(m["integration_test_ids"], "final_predicate.integration_test_ids")...)
		errs = append(errs, e40I04ValidateStringSlice(m["child_oracles"], "final_predicate.child_oracles")...)
		// T-E40-F11-009 (ADR-F11-04): reuses child_oracles_union's own
		// integration_test_ids/child_oracles operand shape verbatim, gated
		// to entity_family "epic" instead of "feature" -- the shape is
		// already key-legal (e40I04CheckUnknownKeys above already allows
		// both field names), so this case only adds the same
		// operand-presence check the child_oracles_union case above makes.
		if len(e40I04AsStringSlice(m["integration_test_ids"])) == 0 {
			errs = append(errs, "final_predicate.integration_test_ids: empty, required for kind descendant_oracles_union")
		}
		if len(e40I04AsStringSlice(m["child_oracles"])) == 0 {
			errs = append(errs, "final_predicate.child_oracles: empty, required for kind descendant_oracles_union")
		}
	}
	return errs
}

func e40I04ValidateResourcePolicy(v interface{}) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"resource_policy: missing or not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"max_cost_usd", "max_wall_clock_seconds", "max_generated_tasks"}, "resource_policy", &errs)

	for _, field := range []string{"max_cost_usd", "max_wall_clock_seconds", "max_generated_tasks"} {
		n, ok := e40I04AsNumber(m[field])
		if !ok || n <= 0 {
			errs = append(errs, fmt.Sprintf("resource_policy.%s: missing or non-positive, must be strictly positive (REQ-F-011)", field))
		}
	}
	return errs
}

func e40I04ValidateAdmission(v interface{}, topToolchain [][2]string) []string {
	var errs []string
	m, ok := e40I04AsMap(v)
	if !ok {
		return []string{"admission: not an object"}
	}
	e40I04CheckUnknownKeys(m, []string{"status", "base_outcome", "reference_outcome", "toolchain_identity"}, "admission", &errs)

	if status, _ := m["status"].(string); strings.TrimSpace(status) == "" {
		errs = append(errs, "admission.status: empty")
	}
	if _, ok := m["base_outcome"].(bool); !ok {
		errs = append(errs, "admission.base_outcome: missing or not a boolean")
	}
	if _, ok := m["reference_outcome"].(bool); !ok {
		errs = append(errs, "admission.reference_outcome: missing or not a boolean")
	}

	admToolchain, tcErrs := e40I04ValidateToolchainIdentity(m["toolchain_identity"], "admission.toolchain_identity")
	errs = append(errs, tcErrs...)

	// AC-019: the two encodings must agree element-for-element, in order.
	if len(tcErrs) == 0 && !reflect.DeepEqual(topToolchain, admToolchain) {
		errs = append(errs, "admission.toolchain_identity != top-level toolchain_identity (AC-019 cross-encoding)")
	}
	return errs
}

// e40I04ExpectedEntityGraphAllowedKeys is the complete declared field set
// for the expected_entity_graph block (§2.3.5, REQ-F-009). This set is
// disjoint from I-01's vocabulary (bench/corpus/corpus.yaml) by
// construction (ADR-F05-01/AC-F11-26) -- TestExpectedEntityGraph's
// i01_field_disjointness subtest injects each of I-01's 16 field names
// into the block and relies on e40I04CheckUnknownKeys to reject them here.
var e40I04ExpectedEntityGraphAllowedKeys = []string{
	"root_family", "descendants", "required_terminal_states",
	"unexpected_descendants", "required_artifacts", "required_provenance",
}

var e40I04DescendantAllowedKeys = []string{"family", "required", "min", "max"}

// e40I04ValidateExpectedEntityGraph checks the expected_entity_graph block
// (§2.3.5, REQ-F-009): required for root_family: epic, optional for
// root_family: feature, forbidden -- the key itself, not merely a
// populated block, per ADR-F11-09 -- for task, bug, change_card, and
// tech_debt, so a non-hierarchical package can never half-declare a
// descendant budget via a bare `expected_entity_graph: null`.
//
// present distinguishes "the key exists in the package" from raw being
// nil so a null value is treated the same as absence for every family
// (Y25: `expected_entity_graph: null` on an epic is exactly as invalid as
// omitting the key; a flat family cannot dodge the forbidden-key check by
// writing null instead of a real block).
func e40I04ValidateExpectedEntityGraph(raw interface{}, present bool, family string) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	isHierarchical := family == "epic" || family == "feature"
	hasContent := present && raw != nil

	if !isHierarchical {
		if present {
			addf("expected_entity_graph: forbidden for family %q (ADR-F11-09)", family)
		}
		return errs
	}

	if !hasContent {
		if family == "epic" {
			addf("expected_entity_graph: required for family 'epic', got null")
		}
		// optional for feature: absent (or null) is fine.
		return errs
	}

	m, ok := e40I04AsMap(raw)
	if !ok {
		return []string{fmt.Sprintf("expected_entity_graph: expected mapping, got %s", e40I04YAMLTypeName(raw))}
	}

	e40I04CheckUnknownKeys(m, e40I04ExpectedEntityGraphAllowedKeys, "expected_entity_graph", &errs)

	// root_family (Y1): string enum of 6, and must equal the package's own
	// entity_family -- a graph declared for a different family than the
	// package it lives in is never meaningful.
	rootFamilyRaw, hasRootFamily := m["root_family"]
	if !hasRootFamily || rootFamilyRaw == nil {
		addf("expected_entity_graph.root_family: missing")
	} else if rootFamily, ok := rootFamilyRaw.(string); !ok {
		addf("root_family: expected string, got %s", e40I04YAMLTypeName(rootFamilyRaw))
	} else if !e40I04ValidEntityFamilies[rootFamily] {
		addf("expected_entity_graph.root_family = %q, want one of epic|feature|task|bug|change_card|tech_debt", rootFamily)
	} else if rootFamily != family {
		addf("expected_entity_graph.root_family = %q, must equal entity_family %q", rootFamily, family)
	}

	// descendants (Y2, Y26, Y27): sequence of mappings, at least one entry,
	// no two entries naming the same family (otherwise §2.3.3a's summation
	// would double-count one family).
	declaredFamilySet := map[string]bool{}
	var descendantFamilies []string
	descendantsRaw, hasDescendants := m["descendants"]
	if !hasDescendants || descendantsRaw == nil {
		addf("expected_entity_graph.descendants: missing")
	} else if list, ok := descendantsRaw.([]interface{}); !ok {
		addf("descendants: expected sequence, got %s", e40I04YAMLTypeName(descendantsRaw))
	} else if len(list) == 0 {
		addf("descendants: must declare at least one family for root_family '%s'", family)
	} else {
		seenFamilies := map[string]bool{}
		for i, item := range list {
			label := fmt.Sprintf("descendants[%d]", i)
			dm, ok := e40I04AsMap(item)
			if !ok {
				addf("%s: expected mapping, got %s", label, e40I04YAMLTypeName(item))
				continue
			}
			e40I04CheckUnknownKeys(dm, e40I04DescendantAllowedKeys, label, &errs)

			var dFamily string
			dFamilyRaw, hasFamily := dm["family"]
			if !hasFamily || dFamilyRaw == nil {
				addf("%s.family: missing", label)
			} else if s, ok := dFamilyRaw.(string); !ok {
				addf("%s.family: expected string, got %s", label, e40I04YAMLTypeName(dFamilyRaw))
			} else if !e40I04ValidEntityFamilies[s] {
				addf("%s.family = %q, want one of epic|feature|task|bug|change_card|tech_debt", label, s)
			} else {
				dFamily = s
				if seenFamilies[dFamily] {
					addf("descendants: duplicate family '%s'", dFamily)
				}
				seenFamilies[dFamily] = true
				declaredFamilySet[dFamily] = true
				descendantFamilies = append(descendantFamilies, dFamily)
			}

			reqRaw, hasReq := dm["required"]
			if !hasReq || reqRaw == nil {
				addf("%s.required: required, got null", label)
			} else if _, ok := reqRaw.(bool); !ok {
				addf("%s.required: expected bool, got %s", label, e40I04YAMLTypeName(reqRaw))
			}

			minRaw, hasMin := dm["min"]
			minVal, minIsInt := e40I04AsStrictInt(minRaw)
			switch {
			case !hasMin || minRaw == nil:
				addf("%s.min: missing", label)
			case !minIsInt:
				addf("%s.min: expected int, got %s", label, e40I04YAMLTypeName(minRaw))
			case minVal < 0:
				addf("%s.min: must be >= 0, got %d", label, minVal)
			}

			maxRaw, hasMax := dm["max"]
			maxVal, maxIsInt := e40I04AsStrictInt(maxRaw)
			switch {
			case !hasMax || maxRaw == nil:
				addf("%s.max: missing", label)
			case !maxIsInt:
				addf("%s.max: expected int, got %s", label, e40I04YAMLTypeName(maxRaw))
			case maxVal < 0:
				addf("%s.max: must be >= 0, got %d", label, maxVal)
			case minIsInt && minVal > maxVal:
				addf("%s: min %d exceeds max %d", label, minVal, maxVal)
			}
		}
	}

	// required_terminal_states (Y7, Y8, ADR-F11-11, AC-F11-25b): a mapping
	// from descendant family to that family's OWN required terminal
	// status, key-complete against descendants[].family in both
	// directions -- never a flat literal, since terminal statuses are
	// per-family (tech_debt has no "completed" step at all).
	rtsRaw, hasRTS := m["required_terminal_states"]
	if !hasRTS || rtsRaw == nil {
		addf("expected_entity_graph.required_terminal_states: missing")
	} else if rtsMap, ok := e40I04AsMap(rtsRaw); !ok {
		addf("required_terminal_states: expected mapping, got %s", e40I04YAMLTypeName(rtsRaw))
	} else {
		for fam, statusRaw := range rtsMap {
			if !declaredFamilySet[fam] {
				addf("required_terminal_states: key '%s' is not a declared descendant family", fam)
				continue
			}
			if statusRaw == nil {
				addf("required_terminal_states.%s: required, got null", fam)
				continue
			}
			status, ok := statusRaw.(string)
			if !ok {
				addf("required_terminal_states.%s: expected string, got %s", fam, e40I04YAMLTypeName(statusRaw))
				continue
			}
			terminals, err := e40I04LoadTerminalStatuses(fam)
			if err != nil {
				addf("required_terminal_states.%s: %v", fam, err)
				continue
			}
			if !terminals[status] {
				addf("required_terminal_states.%s = %q is not a terminal status of family %q's own workflow (ADR-F11-11)", fam, status, fam)
			}
		}
		for _, fam := range descendantFamilies {
			if _, ok := rtsMap[fam]; !ok {
				addf("required_terminal_states: missing key '%s' declared in descendants[]", fam)
			}
		}
	}

	// unexpected_descendants (Y6): string enum invalidate|ignore.
	udRaw, hasUD := m["unexpected_descendants"]
	if !hasUD || udRaw == nil {
		addf("expected_entity_graph.unexpected_descendants: missing")
	} else if s, ok := udRaw.(string); !ok {
		addf("unexpected_descendants: expected string, got %s", e40I04YAMLTypeName(udRaw))
	} else if s != "invalidate" && s != "ignore" {
		addf("unexpected_descendants = %q, want invalidate|ignore", s)
	}

	// required_artifacts (Y12, Y13) / required_provenance (Y9, Y10):
	// sequences of strings. Both are required keys (an empty list is a
	// legal "no additional artifacts/provenance required" declaration --
	// Y-table's own thirteen-path inventory enumerates these as declared
	// fields with a fixed type, never as optional).
	for _, spec := range []struct {
		field string
	}{{"required_artifacts"}, {"required_provenance"}} {
		raw, has := m[spec.field]
		if !has || raw == nil {
			addf("expected_entity_graph.%s: missing", spec.field)
			continue
		}
		list, ok := raw.([]interface{})
		if !ok {
			addf("%s: expected sequence, got %s", spec.field, e40I04YAMLTypeName(raw))
			continue
		}
		for i, item := range list {
			if _, ok := item.(string); !ok {
				addf("%s[%d]: expected string, got %s", spec.field, i, e40I04YAMLTypeName(item))
			}
		}
	}

	return errs
}

// e40I04YAMLTypeName names the YAML-decoded Go type of v using the
// vocabulary spec.md §7.3.5's Y-table itself uses (string, int, float,
// bool, sequence, mapping, null), so a wrong-type diagnostic reads exactly
// as that table's "Expected rejection" column states it.
func e40I04YAMLTypeName(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case int, int64:
		return "int"
	case float64:
		return "float"
	case bool:
		return "bool"
	case []interface{}:
		return "sequence"
	case map[string]interface{}:
		return "mapping"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// e40I04AsStrictInt reports v as an int only when the YAML decoder
// produced an actual integer scalar -- unlike e40I04AsNumber (used
// elsewhere for resource_policy's cost/wall-clock fields, which
// deliberately accept a float), this rejects a float or a numeric string
// outright (Y4/Y5: descendants[].min/max must be int, not "2" or 2.5).
func e40I04AsStrictInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// e40I04WorkflowFileByFamily maps each I-04 delivery family to the
// embedded canonical workflow YAML that defines its OWN status steps.
// change_card -> change.yaml and tech_debt -> tech-debt.yaml are the two
// non-literal mappings (the workflow file basenames predate the I-04
// family vocabulary and don't use underscores).
var e40I04WorkflowFileByFamily = map[string]string{
	"epic":        "epic.yaml",
	"feature":     "feature.yaml",
	"task":        "task.yaml",
	"bug":         "bug.yaml",
	"change_card": "change.yaml",
	"tech_debt":   "tech-debt.yaml",
}

// e40I04WorkflowStepFile decodes just enough of a workflow YAML's steps:
// map to answer "is this step terminal" -- the only fact
// required_terminal_states validation needs from the real config.
type e40I04WorkflowStepFile struct {
	Steps map[string]struct {
		Terminal bool `yaml:"terminal"`
	} `yaml:"steps"`
}

// e40I04TerminalStatusesCache avoids re-reading and re-parsing the same
// workflow YAML for every descendant entry across every package a test run
// validates.
var e40I04TerminalStatusesCache = map[string]map[string]bool{}

// e40I04WorkflowRepoRoot resolves the repository root from this source
// file's own location (mirrors runSharkTC012/runSharkTC013 in
// e39_interactions_test.go), independent of any packageDir a caller of
// e40I04ValidateScenarioPackage happens to be using (which, for testdata
// cases, is tests/contracts/testdata/e40_i04/, not the repo root).
func e40I04WorkflowRepoRoot() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed while resolving repo root for workflow config lookup")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..")), nil
}

// e40I04LoadTerminalStatuses returns the set of terminal status names for
// family's OWN workflow (ADR-F11-11: terminal statuses are per-family, so
// this must never be a hardcoded literal in this validator). It reads the
// embedded canonical workflow YAML,
// internal/sharkdata/default_data/workflow/<file>.yaml -- the source
// ADR-F11-11 itself cites (read 2026-09-04) and the one actually populated
// in this checkout. T-E40-F11-005's own task notes originally named
// shark-data/workflow/<entity>.yaml instead, but that deployed-copy
// directory is empty here (per this project's "edit the embedded
// canonical, shark-data/ is a deployed copy" convention) -- ADR-F11-11 was
// the more specific, evidence-cited source and won per this repo's
// "surface conflicts, pick one, don't blend" rule. T-E40-F11-014 (the docs
// task) reconciled T-E40-F11-005's notes to this path.
//
// A missing, unparseable, or terminal-step-free workflow file is a loud
// error, not a silently empty terminal set -- an empty set would make
// every required_terminal_states value for that family look invalid (or,
// if callers ignored the error, look like "nothing is terminal", which is
// worse than failing loudly).
func e40I04LoadTerminalStatuses(family string) (map[string]bool, error) {
	if cached, ok := e40I04TerminalStatusesCache[family]; ok {
		return cached, nil
	}
	filename, ok := e40I04WorkflowFileByFamily[family]
	if !ok {
		return nil, fmt.Errorf("no workflow file mapping for family %q", family)
	}
	repoRoot, err := e40I04WorkflowRepoRoot()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "workflow", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow config %s: %w", path, err)
	}
	var wf e40I04WorkflowStepFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow config %s: %w", path, err)
	}
	if len(wf.Steps) == 0 {
		return nil, fmt.Errorf("workflow config %s declares no steps", path)
	}
	terminals := map[string]bool{}
	for name, step := range wf.Steps {
		if step.Terminal {
			terminals[name] = true
		}
	}
	if len(terminals) == 0 {
		return nil, fmt.Errorf("workflow config %s declares no terminal step", path)
	}
	e40I04TerminalStatusesCache[family] = terminals
	return terminals, nil
}

// e40I04MustReplaceOnce is strings.Replace with a guard: it fails the test
// loudly if old does not occur in s exactly once, so a Y-case mutation
// that silently matches zero (or more than one) location can never pass
// by accident -- a test bug here would otherwise validate the unmodified
// baseline and still "pass" for the wrong reason.
func e40I04MustReplaceOnce(t *testing.T, s, old, new string) string {
	t.Helper()
	if n := strings.Count(s, old); n != 1 {
		t.Fatalf("test bug: substring %q occurs %d times in the fixture text, want exactly 1", old, n)
	}
	return strings.Replace(s, old, new, 1)
}

// TestExpectedEntityGraph is T-E40-F11-005's own contract test for the
// expected_entity_graph block (REQ-F-009, test-plan.md's Caller-Path
// Contract row for TC-25/26/27/25a/25b: "go test ./tests/contracts/...
// -run TestExpectedEntityGraph"). It covers:
//
//   - AC-F11-25/AC-F11-25b: the Y1-Y27 type-confusion and key-completeness
//     partition (spec.md §7.3.5), built by substitution over one epic
//     baseline -- the same technique TestTC030's six_family_vocabulary
//     subtest already uses for the entity_family enum -- rather than 27
//     separately committed testdata files. Y24 (a literal duplicate YAML
//     key inside one descendant mapping) is the one case NOT expressed as
//     a validator-level check: gopkg.in/yaml.v3's own Unmarshal already
//     rejects a duplicate mapping key as a parse error ("mapping key ...
//     already defined"), before e40I04ValidateScenarioPackage ever sees a
//     decoded value, so its own subtest asserts that parse-time rejection
//     directly rather than duplicating it in this validator.
//   - AC-F11-25a: the four pre-existing non-epic seed packages remain a
//     pure schema_version bump (none acquires the block). The call-plan
//     half of AC-F11-25a (bounded_descendant_calls falling back to
//     resource_policy.max_generated_tasks when the block is absent) is
//     REQ-F-006 territory landing with T-E40-F11-012/013's call planner --
//     not testable here since e40-benchmark.sh has no call_plan yet.
//   - AC-F11-26: no expected_entity_graph field name collides with I-01's
//     (bench/corpus/corpus.yaml's) 16-name vocabulary (ADR-F05-01).
//   - AC-F11-27: the block is read by no undeclared third file. Neither
//     declared consumer (bench/scripts/evaluate-lifecycle.sh, landing in
//     T-E40-F11-009; bench/scripts/lib/e40_benchmark.py, landing in
//     T-E40-F11-012/013) reads the block yet, so only the negative half of
//     AC-F11-27 ("no undeclared file reads it") is testable today -- the
//     positive half ("both declared consumers DO read it") lands with
//     those tasks.
//
// Y1-Y27 (structural graph-population correctness -- an actual run's
// descendant counts against min/max, terminal states, and provenance,
// spec.md §7.3.6's G1-G10) is out of scope: that requires the structural
// evaluator T-E40-F11-009 adds to bench/scripts/evaluate-lifecycle.sh, not
// this package-contract validator.
func TestExpectedEntityGraph(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	scenariosRoot := filepath.Join(repoRoot, "bench", "scenarios")
	testdataDir := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i04")
	index := e40I04ReadScenarioIndex(t, filepath.Join(scenariosRoot, "scenarios.yaml"))
	evaluatorRoot := filepath.Join(testdataDir, "valid", "evaluator")

	// epicBaselineBase is the committed valid/package.yaml (bug family)
	// with entity_family switched to epic and final_predicate.kind
	// switched to the epic-permitted kind (ADR-F11-04) -- fixture, adapter,
	// toolchain_identity, input/evaluator paths (still under valid/'s real,
	// committed assets), and resource_policy are all untouched.
	epicBaselineBase := func() string {
		data, err := os.ReadFile(filepath.Join(testdataDir, "valid", "package.yaml"))
		if err != nil {
			t.Fatalf("read valid baseline: %v", err)
		}
		s := string(data)
		s = e40I04MustReplaceOnce(t, s, "entity_family: bug", "entity_family: epic")
		s = e40I04MustReplaceOnce(t, s, "kind: f2p_p2p", "kind: descendant_oracles_union")
		return s
	}

	// validGraphBlock is §2.3.5's own worked example, reused as the single
	// positive baseline every Y-case below mutates exactly one field of.
	const validGraphBlock = `expected_entity_graph:
  root_family: epic
  descendants:
    - {family: feature, required: true, min: 2, max: 2}
    - {family: task, required: true, min: 2, max: 8}
  required_terminal_states:
    feature: completed
    task: completed
  unexpected_descendants: invalidate
  required_artifacts: [spec.md, test-plan.md]
  required_provenance: [candidate_identity, workflow_routing_identity]
`

	packageWithGraph := func(graph string) map[string]interface{} {
		return e40I04ParseYAMLMap(t, []byte(epicBaselineBase()+"\n"+graph))
	}
	validate := func(pkg map[string]interface{}) []string {
		return e40I04ValidateScenarioPackage(pkg, index, testdataDir, evaluatorRoot)
	}

	t.Run("positive_control_valid_epic_graph", func(t *testing.T) {
		errs := validate(packageWithGraph(validGraphBlock))
		if e40ContainsErrorMatching(errs, "expected_entity_graph") || e40ContainsErrorMatching(errs, "root_family") || e40ContainsErrorMatching(errs, "descendants") || e40ContainsErrorMatching(errs, "required_terminal_states") || e40ContainsErrorMatching(errs, "unexpected_descendants") || e40ContainsErrorMatching(errs, "required_artifacts") || e40ContainsErrorMatching(errs, "required_provenance") {
			t.Errorf("§2.3.5's own worked example must validate cleanly, got:\n%s", strings.Join(errs, "\n"))
		}
	})

	// ---------------------------------------------------------------------
	// Y1-Y27: spec.md §7.3.5's type-confusion and partition table. Each
	// case mutates exactly one field of validGraphBlock (guarded by
	// e40I04MustReplaceOnce so a mistyped target string fails loudly
	// instead of silently validating the unmodified baseline).
	// ---------------------------------------------------------------------
	type graphCase struct {
		name    string
		graph   string
		wantAny []string
	}
	ycases := []graphCase{
		{"Y1_root_family_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "root_family: epic", "root_family: 7"),
			[]string{"root_family: expected string, got int"}},
		{"Y2_descendants_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock,
				"descendants:\n    - {family: feature, required: true, min: 2, max: 2}\n    - {family: task, required: true, min: 2, max: 8}",
				"descendants: {}"),
			[]string{"descendants: expected sequence, got mapping"}},
		{"Y3_descendant_family_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", "{family: [task], required: true, min: 2, max: 2}"),
			[]string{"descendants[0].family: expected string, got sequence"}},
		{"Y4_descendant_min_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 2}", `min: "2", max: 2}`),
			[]string{"descendants[0].min: expected int, got string"}},
		{"Y5_descendant_max_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 8}", "min: 2, max: 2.5}"),
			[]string{"descendants[1].max: expected int, got float"}},
		{"Y6_unexpected_descendants_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "unexpected_descendants: invalidate", "unexpected_descendants: true"),
			[]string{"unexpected_descendants: expected string, got bool"}},
		{"Y7_required_terminal_states_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock,
				"required_terminal_states:\n    feature: completed\n    task: completed",
				`required_terminal_states: ["completed"]`),
			[]string{"required_terminal_states: expected mapping, got sequence"}},
		{"Y8_required_terminal_states_value_null",
			e40I04MustReplaceOnce(t, validGraphBlock, "feature: completed", "feature: null"),
			[]string{"required_terminal_states.feature: required, got null"}},
		{"Y9_required_provenance_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "required_provenance: [candidate_identity, workflow_routing_identity]", `required_provenance: "key"`),
			[]string{"required_provenance: expected sequence, got string"}},
		{"Y10_required_provenance_element_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "required_provenance: [candidate_identity, workflow_routing_identity]", "required_provenance: [12]"),
			[]string{"required_provenance[0]: expected string, got int"}},
		{"Y11_descendant_required_quoted_string",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", `{family: feature, required: "true", min: 2, max: 2}`),
			[]string{"descendants[0].required: expected bool, got string"}},
		{"Y12_required_artifacts_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "required_artifacts: [spec.md, test-plan.md]", `required_artifacts: "spec.md"`),
			[]string{"required_artifacts: expected sequence, got string"}},
		{"Y13_required_artifacts_element_wrong_type",
			e40I04MustReplaceOnce(t, validGraphBlock, "required_artifacts: [spec.md, test-plan.md]", "required_artifacts: [{name: spec.md}]"),
			[]string{"required_artifacts[0]: expected string, got mapping"}},
		// Y15: spec.md documents `yes`/`no` as YAML 1.1 booleans that must
		// be ACCEPTED. Empirically verified against this repo's actual
		// gopkg.in/yaml.v3 dependency (go run against a throwaway
		// map[string]interface{} decode): yaml.v3 follows the YAML 1.2
		// core schema, where only true/false/True/False/TRUE/FALSE are
		// booleans -- `yes`/`no` decode as the STRINGS "yes"/"no". This is
		// a spec/library discrepancy (Rule 7: surface it, don't silently
		// resolve it either way) -- flagged in this task's completion note
		// for correction, likely in T-E40-F11-014's docs pass. This test
		// asserts the REAL, empirically-verified behavior of the library
		// this validator actually uses, not the spec's stated expectation.
		{"Y15_yes_no_yaml_1_1_booleans_ACTUALLY_REJECTED_not_accepted",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", "{family: feature, required: yes, min: 2, max: 2}"),
			[]string{"descendants[0].required: expected bool, got string"}},
		{"Y16_descendant_required_quoted_string_single_quote",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", "{family: feature, required: 'true', min: 2, max: 2}"),
			[]string{"descendants[0].required: expected bool, got string"}},
		{"Y17_descendant_required_int",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", "{family: feature, required: 1, min: 2, max: 2}"),
			[]string{"descendants[0].required: expected bool, got int"}},
		{"Y18_descendant_required_null",
			e40I04MustReplaceOnce(t, validGraphBlock, "{family: feature, required: true, min: 2, max: 2}", "{family: feature, required: null, min: 2, max: 2}"),
			[]string{"descendants[0].required: required, got null"}},
		{"Y19_descendant_min_negative",
			e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 2}", "min: -1, max: 2}"),
			[]string{"descendants[0].min: must be >= 0, got -1"}},
		{"Y20_descendant_max_negative",
			e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 8}", "min: 2, max: -1}"),
			[]string{"descendants[1].max: must be >= 0, got -1"}},
		{"Y21_descendant_min_exceeds_max",
			e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 8}", "min: 3, max: 2}"),
			[]string{"descendants[1]: min 3 exceeds max 2"}},
		{"Y26_descendants_empty_on_epic",
			`expected_entity_graph:
  root_family: epic
  descendants: []
  required_terminal_states: {}
  unexpected_descendants: invalidate
  required_artifacts: []
  required_provenance: []
`,
			[]string{"descendants: must declare at least one family for root_family 'epic'"}},
		{"Y27_duplicate_descendant_family",
			`expected_entity_graph:
  root_family: epic
  descendants:
    - {family: task, required: true, min: 1, max: 2}
    - {family: task, required: true, min: 1, max: 2}
  required_terminal_states:
    task: completed
  unexpected_descendants: invalidate
  required_artifacts: []
  required_provenance: []
`,
			[]string{"descendants: duplicate family 'task'"}},
	}
	for _, c := range ycases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			errs := validate(packageWithGraph(c.graph))
			for _, want := range c.wantAny {
				if !e40ContainsErrorMatching(errs, want) {
					t.Errorf("case %s: expected an error containing %q, got:\n%s", c.name, want, strings.Join(errs, "\n"))
				}
			}
		})
	}

	// Y14/Y22/Y23: acceptance boundaries, not rejections -- each must
	// produce ZERO expected_entity_graph-related errors.
	t.Run("Y14_descendant_required_false_accepted", func(t *testing.T) {
		graph := e40I04MustReplaceOnce(t, validGraphBlock, "{family: task, required: true, min: 2, max: 8}", "{family: task, required: false, min: 2, max: 8}")
		errs := validate(packageWithGraph(graph))
		if e40ContainsErrorMatching(errs, "descendants[1].required") {
			t.Errorf("required: false must be accepted, got:\n%s", strings.Join(errs, "\n"))
		}
	})
	t.Run("Y22_min_zero_max_zero_accepted", func(t *testing.T) {
		graph := e40I04MustReplaceOnce(t, validGraphBlock, "min: 2, max: 8}", "min: 0, max: 0}")
		errs := validate(packageWithGraph(graph))
		if e40ContainsErrorMatching(errs, "descendants[1]") {
			t.Errorf("min: 0, max: 0 (a declared-but-never-realized descendant family) must be accepted, got:\n%s", strings.Join(errs, "\n"))
		}
	})
	t.Run("Y23_min_2_max_2_exact_count_boundary_accepted", func(t *testing.T) {
		// validGraphBlock's own feature entry is already the min==max==2
		// boundary -- proven by positive_control_valid_epic_graph above;
		// this subtest exists only to give Y23 its own named, traceable
		// case per the closed-inventory rule (spec.md §7.1a).
		errs := validate(packageWithGraph(validGraphBlock))
		if e40ContainsErrorMatching(errs, "descendants[0]") {
			t.Errorf("min: 2, max: 2 must be accepted, got:\n%s", strings.Join(errs, "\n"))
		}
	})

	// Y24: a literal duplicate YAML key inside one descendant mapping is a
	// gopkg.in/yaml.v3 PARSE-TIME rejection in this codebase (empirically
	// verified: Unmarshal into map[string]interface{} returns "mapping key
	// ... already defined"), not something e40I04ValidateExpectedEntityGraph
	// itself needs to detect -- by the time this validator sees a decoded
	// value, a duplicate key has already been rejected upstream.
	t.Run("Y24_duplicate_yaml_key_in_descendant_rejected_at_parse_time", func(t *testing.T) {
		graph := `expected_entity_graph:
  root_family: epic
  descendants:
    - family: task
      required: true
      min: 1
      min: 2
      max: 5
  required_terminal_states:
    task: completed
  unexpected_descendants: invalidate
  required_artifacts: []
  required_provenance: []
`
		full := []byte(epicBaselineBase() + "\n" + graph)
		var m map[string]interface{}
		parseErr := yaml.Unmarshal(full, &m)
		if parseErr == nil {
			t.Fatal("a descendant mapping declaring 'min' twice must be rejected, not last-wins; got no parse error")
		}
		if !strings.Contains(parseErr.Error(), "min") {
			t.Errorf("duplicate-key parse error does not name the offending key 'min': %v", parseErr)
		}
	})

	// Y25: `expected_entity_graph: null` on an epic is exactly as invalid
	// as omitting the key outright.
	t.Run("Y25_null_block_on_epic_rejected", func(t *testing.T) {
		pkg := e40I04ParseYAMLMap(t, []byte(epicBaselineBase()+"\nexpected_entity_graph: null\n"))
		errs := validate(pkg)
		if !e40ContainsErrorMatching(errs, "expected_entity_graph: required for family 'epic', got null") {
			t.Errorf("got:\n%s", strings.Join(errs, "\n"))
		}
	})

	// root_family must equal the package's own entity_family -- a graph
	// declared for a different family than the package it lives in is
	// never meaningful, and is not itself one of the Y-cases above.
	t.Run("root_family_must_equal_entity_family", func(t *testing.T) {
		graph := e40I04MustReplaceOnce(t, validGraphBlock, "root_family: epic", "root_family: feature")
		errs := validate(packageWithGraph(graph))
		if !e40ContainsErrorMatching(errs, "root_family", "must equal entity_family") {
			t.Errorf("got:\n%s", strings.Join(errs, "\n"))
		}
	})

	// -----------------------------------------------------------------
	// AC-F11-25b: required_terminal_states key-completeness (both
	// directions) and per-family terminal-status enforcement read from
	// each family's OWN workflow config, never a hardcoded literal --
	// proven by the tech_debt example ADR-F11-11 names explicitly
	// (tech_debt has no "completed" step at all).
	// -----------------------------------------------------------------
	t.Run("AC_F11_25b_key_completeness", func(t *testing.T) {
		t.Run("missing_key_declared_in_descendants", func(t *testing.T) {
			graph := e40I04MustReplaceOnce(t, validGraphBlock,
				"required_terminal_states:\n    feature: completed\n    task: completed",
				"required_terminal_states:\n    feature: completed")
			errs := validate(packageWithGraph(graph))
			if !e40ContainsErrorMatching(errs, "required_terminal_states: missing key 'task' declared in descendants[]") {
				t.Errorf("got:\n%s", strings.Join(errs, "\n"))
			}
		})
		t.Run("extra_key_not_a_declared_descendant", func(t *testing.T) {
			graph := e40I04MustReplaceOnce(t, validGraphBlock,
				"required_terminal_states:\n    feature: completed\n    task: completed",
				"required_terminal_states:\n    feature: completed\n    task: completed\n    bug: completed")
			errs := validate(packageWithGraph(graph))
			if !e40ContainsErrorMatching(errs, "required_terminal_states: key 'bug' is not a declared descendant family") {
				t.Errorf("got:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	t.Run("AC_F11_25b_terminal_status_read_from_own_workflow", func(t *testing.T) {
		techDebtGraph := func(status string) string {
			return fmt.Sprintf(`expected_entity_graph:
  root_family: epic
  descendants:
    - {family: tech_debt, required: true, min: 1, max: 1}
  required_terminal_states:
    tech_debt: %s
  unexpected_descendants: invalidate
  required_artifacts: []
  required_provenance: []
`, status)
		}
		t.Run("tech_debt_completed_rejected_no_such_step", func(t *testing.T) {
			errs := validate(packageWithGraph(techDebtGraph("completed")))
			if !e40ContainsErrorMatching(errs, "required_terminal_states.tech_debt", "not a terminal status") {
				t.Errorf("tech_debt's own workflow (internal/sharkdata/default_data/workflow/tech-debt.yaml) has no 'completed' step -- got:\n%s", strings.Join(errs, "\n"))
			}
		})
		t.Run("tech_debt_resolved_accepted_real_terminal_step", func(t *testing.T) {
			errs := validate(packageWithGraph(techDebtGraph("resolved")))
			if e40ContainsErrorMatching(errs, "required_terminal_states.tech_debt") {
				t.Errorf("tech_debt's own workflow declares 'resolved' as terminal -- got:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	// -----------------------------------------------------------------
	// AC-F11-25/ADR-F11-09: requiredness gating across all six families,
	// beyond the two committed testdata cases (case-28/case-29 in
	// TestTC030's malformed_package_cases table).
	// -----------------------------------------------------------------
	t.Run("requiredness_gating_across_all_six_families", func(t *testing.T) {
		t.Run("epic_missing_block_rejected", func(t *testing.T) {
			pkg := e40I04ParseYAMLMap(t, []byte(epicBaselineBase()))
			errs := validate(pkg)
			if !e40ContainsErrorMatching(errs, "expected_entity_graph: required for family 'epic'") {
				t.Errorf("got:\n%s", strings.Join(errs, "\n"))
			}
		})
		t.Run("epic_with_valid_block_accepted", func(t *testing.T) {
			errs := validate(packageWithGraph(validGraphBlock))
			if e40ContainsErrorMatching(errs, "expected_entity_graph") {
				t.Errorf("got:\n%s", strings.Join(errs, "\n"))
			}
		})
		t.Run("feature_optional_absent_accepted", func(t *testing.T) {
			// the real, committed valid-feature baseline has no
			// expected_entity_graph key at all -- "optional" means
			// absence is fine, not merely tolerated.
			data, err := os.ReadFile(filepath.Join(testdataDir, "valid-feature", "package.yaml"))
			if err != nil {
				t.Fatalf("read valid-feature baseline: %v", err)
			}
			pkg := e40I04ParseYAMLMap(t, data)
			errs := e40I04ValidateScenarioPackage(pkg, index, testdataDir, filepath.Join(testdataDir, "valid-feature", "evaluator"))
			if e40ContainsErrorMatching(errs, "expected_entity_graph") {
				t.Errorf("got:\n%s", strings.Join(errs, "\n"))
			}
		})
		for _, family := range []string{"task", "bug", "change_card", "tech_debt"} {
			family := family
			t.Run(family+"_forbidden_when_present", func(t *testing.T) {
				base, err := os.ReadFile(filepath.Join(testdataDir, "valid", "package.yaml"))
				if err != nil {
					t.Fatalf("read valid baseline: %v", err)
				}
				s := e40I04MustReplaceOnce(t, string(base), "entity_family: bug", "entity_family: "+family)
				s += "\n" + validGraphBlock
				pkg := e40I04ParseYAMLMap(t, []byte(s))
				errs := validate(pkg)
				want := fmt.Sprintf("expected_entity_graph: forbidden for family %q", family)
				if !e40ContainsErrorMatching(errs, want) {
					t.Errorf("family %q: got:\n%s", family, strings.Join(errs, "\n"))
				}
			})
			t.Run(family+"_forbidden_even_when_null", func(t *testing.T) {
				// ADR-F11-09's "cannot silently declare a descendant
				// budget" is best served by rejecting the KEY outright
				// (advisor-reviewed design choice), not by special-casing
				// a null value as equivalent to absence -- a flat package
				// must not carry the key at all, even set to null.
				base, err := os.ReadFile(filepath.Join(testdataDir, "valid", "package.yaml"))
				if err != nil {
					t.Fatalf("read valid baseline: %v", err)
				}
				s := e40I04MustReplaceOnce(t, string(base), "entity_family: bug", "entity_family: "+family)
				s += "\nexpected_entity_graph: null\n"
				pkg := e40I04ParseYAMLMap(t, []byte(s))
				errs := validate(pkg)
				want := fmt.Sprintf("expected_entity_graph: forbidden for family %q", family)
				if !e40ContainsErrorMatching(errs, want) {
					t.Errorf("family %q: a null expected_entity_graph must still be rejected as forbidden (the key itself is disallowed), got:\n%s", family, strings.Join(errs, "\n"))
				}
			})
		}
	})

	// -----------------------------------------------------------------
	// AC-F11-25a: the four pre-existing non-epic seed packages remain a
	// pure schema_version bump -- none acquires the new block.
	// -----------------------------------------------------------------
	t.Run("AC_F11_25a_four_existing_packages_remain_pure_version_bump", func(t *testing.T) {
		named := []string{
			"py-bug-due-date-boundary", "py-change-priority-scale",
			"py-feature-recurring-tasks", "py-techdebt-consolidate-validation",
		}
		for _, id := range named {
			id := id
			t.Run(id, func(t *testing.T) {
				path := filepath.Join(scenariosRoot, "packages", id, "package.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				pkg := e40I04ParseYAMLMap(t, data)
				if _, present := pkg["expected_entity_graph"]; present {
					t.Errorf("%s acquired an expected_entity_graph block; AC-F11-25a requires the four pre-existing packages remain a pure schema_version bump", id)
				}
			})
		}
	})

	// -----------------------------------------------------------------
	// AC-F11-26: expected_entity_graph's field vocabulary is disjoint from
	// I-01's (bench/corpus/corpus.yaml's) 16-name vocabulary (ADR-F05-01).
	// The 16 names below were read from bench/corpus/corpus.yaml's own
	// schema on 2026-09-04 (spec.md §7.1a row 26): top level
	// {schema_version, fixture, items, negative_items, p2p_sets}; items[]
	// {f2p, fixture_base_sha, id, p2p_set, prompt_path,
	// reference_patch_path, seed_path, type}; fixture {base_sha,
	// submodule_path, toolchain}.
	// -----------------------------------------------------------------
	t.Run("AC_F11_26_no_field_borrowed_from_i01", func(t *testing.T) {
		i01Names := []string{
			"schema_version", "fixture", "items", "negative_items", "p2p_sets",
			"f2p", "fixture_base_sha", "id", "p2p_set", "prompt_path",
			"reference_patch_path", "seed_path", "type",
			"base_sha", "submodule_path", "toolchain",
		}
		if len(i01Names) != 16 {
			t.Fatalf("test bug: I-01 name inventory must have 16 entries, got %d", len(i01Names))
		}

		// Static disjointness: the block's own declared vocabulary must
		// not itself contain any of the 16 names.
		for _, n := range i01Names {
			for _, own := range e40I04ExpectedEntityGraphAllowedKeys {
				if n == own {
					t.Errorf("expected_entity_graph field name %q collides with I-01 vocabulary", own)
				}
			}
		}

		// Dynamic disjointness: injecting each I-01 name as an additional
		// field inside a real block must be rejected as unknown, proving
		// the validator itself enforces the boundary rather than the two
		// vocabularies merely happening not to overlap today.
		for _, name := range i01Names {
			name := name
			t.Run(name, func(t *testing.T) {
				graph := e40I04MustReplaceOnce(t, validGraphBlock,
					"unexpected_descendants: invalidate",
					fmt.Sprintf("unexpected_descendants: invalidate\n  %s: injected", name))
				errs := validate(packageWithGraph(graph))
				want := fmt.Sprintf("expected_entity_graph.%s: unknown field", name)
				if !e40ContainsErrorMatching(errs, want) {
					t.Errorf("injecting I-01 field %q must be rejected as unknown, got:\n%s", name, strings.Join(errs, "\n"))
				}
			})
		}
	})

	// -----------------------------------------------------------------
	// AC-F11-27: no third private copy reads expected_entity_graph. Only
	// the negative half is testable today (see doc comment above) -- every
	// file the corpus-wide grep finds must be one of the two declared
	// consumers, this contract test file itself, or its testdata.
	// -----------------------------------------------------------------
	t.Run("AC_F11_27_no_undeclared_reader", func(t *testing.T) {
		cmd := exec.Command("grep", "-rl", "--exclude-dir=__pycache__", "expected_entity_graph", "bench", "tests")
		cmd.Dir = repoRoot
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			if exitErr, ok := cmdErr.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
				t.Fatalf("grep failed: %v\n%s", cmdErr, out)
			}
		}
		raw := strings.TrimSpace(string(out))
		var files []string
		if raw != "" {
			files = strings.Split(raw, "\n")
		}
		if len(files) == 0 {
			t.Fatal("grep found zero files mentioning expected_entity_graph -- expected at least this contract test and its testdata")
		}
		allowedFiles := map[string]bool{
			// declared consumers (spec.md §2.3.5/AC-F11-27) -- neither
			// reads the block yet (see doc comment above).
			"bench/scripts/evaluate-lifecycle.sh": true,
			"bench/scripts/lib/e40_benchmark.py":  true,
			// tc108 is this task's OWN bash wrapper around `go test ...
			// -run TestExpectedEntityGraph` (test-plan.md's Caller-Path
			// Contract row) -- it names the block in prose/comments to
			// explain what it drives, but never reads or parses the
			// block itself, so it is not a "reader" AC-F11-27 counts.
			// run-all.sh's own hit is purely its filename registration
			// line for tc108 (the literal string
			// "tc108_expected_entity_graph_contract_test.sh" as a path),
			// not a read of the block either.
			"bench/scripts/tests/tc108_expected_entity_graph_contract_test.sh": true,
			"bench/scripts/tests/run-all.sh":                                   true,
			// tc110 (T-E40-F11-009) drives the real evaluate-lifecycle.sh --
			// one of the two declared consumers -- against synthetic I-05/I-07
			// fixtures and reads the OUTPUT record's own "expected_entity_graph"
			// key back for assertions. That is consuming the declared
			// consumer's output, not a private re-implementation of the
			// block's own parsing/interpretation, so it is not a "reader"
			// AC-F11-27 counts either.
			"bench/scripts/tests/tc110_epic_root_scenario_test.sh": true,
			// tc080's UAT-R2-01 fixture stubs out evaluate-lifecycle.sh
			// itself (one of the two declared consumers) so the test can
			// force a deterministic ineligible verdict without a full real
			// evaluate-lifecycle.sh pass. To stay a faithful stand-in it
			// emits the required eligibility.expected_entity_graph_valid
			// boolean evaluate-lifecycle.sh always sets -- the one field
			// whose name substring-matches this grep -- but never the
			// four-field expected_entity_graph block itself, and never
			// reads or interprets either. That is carrying a declared
			// consumer's required output field, not a private
			// re-implementation of the block's parsing/interpretation, so
			// it is not a "reader" AC-F11-27 counts.
			"bench/scripts/tests/tc080_spend_gate_refusal_test.sh": true,
			// T-E40-F11-014 (docs task): bench/README.md's "I-04 scenario
			// package schema" table and its "E40-F11 six-family readiness
			// and cover additions" section document the block in prose --
			// a reader's map to spec.md §2.3.5 and the two declared
			// consumers, never a parser of the block itself.
			"bench/README.md": true,
		}
		// scenarioPackagePattern matches bench/scenarios/packages/<id>/package.yaml
		// -- a package.yaml that legitimately DECLARES an expected_entity_graph
		// block (T-E40-F11-009's py-epic-task-organization is the first) is a
		// data file, not a "reader" AC-F11-27 counts: it does not itself parse
		// or interpret the block, it only carries it as admitted content for
		// the two declared consumers to read. A path pattern (not one literal
		// path) so a later epic/feature-root curation task registering its
		// own expected_entity_graph-bearing package does not re-trip this gate.
		scenarioPackagePattern := regexp.MustCompile(`^bench/scenarios/packages/[^/]+/package\.yaml$`)
		for _, rel := range files {
			if allowedFiles[rel] {
				continue
			}
			if strings.HasPrefix(rel, "tests/contracts/") {
				continue // this contract test file and its own testdata
			}
			if scenarioPackagePattern.MatchString(rel) {
				continue // declarer, not reader (see comment above)
			}
			t.Errorf("unexpected reader of expected_entity_graph: %s (AC-F11-27: exactly two declared consumers, no third private copy)", rel)
		}
	})
}
