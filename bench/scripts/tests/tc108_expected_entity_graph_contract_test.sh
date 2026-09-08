#!/usr/bin/env bash
# TC-108 / T-E40-F11-005 (spec.md REQ-F-009, AC-F11-25/-25a/-25b/-26/-27;
# test-plan.md TC-25/26/27/25a/25b).
#
# Caller-Path Contract (test-plan.md's row for TC-25..27/25a/25b):
# production entrypoint is `go test ./tests/contracts/... -run
# TestExpectedEntityGraph` -- the expected_entity_graph package-contract
# validation lives in tests/contracts/e40_i04_scenario_contract_test.go
# (a Go contract test, ADR-F11-08: not production code), so this bash
# wrapper's job is to invoke exactly that command and prove it is a real,
# non-vacuous run -- not to re-implement any assertion the Go test already
# makes. Per test-plan.md's own §7.2 rule ("no test may call an internal
# Python function directly"), this file IS the check for the CLI-shape
# concern (the go test command), not a duplicate of the Go test's content.
#
# The single most likely way this file could silently prove nothing is a
# `-run` regexp typo: `go test -run <pattern>` exits 0 with "no tests to
# run" when the pattern matches zero tests, which looks identical to a
# real pass in a bare exit-code check. This file therefore additionally:
#   (i)  asserts the go test binary reports a NON-ZERO count of executed
#        tests (via `go test -v`'s own per-subtest PASS/FAIL lines), and
#   (ii) greps the output for a representative sample of the named
#        subtests the Go test itself declares (Y-cases, AC-F11-25a/-26/-27
#        subtests) -- proving real depth ran, not just that SOME test
#        named "TestExpectedEntityGraph" (however thin) passed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

fail() {
	echo "TC-108 FAIL: $1" >&2
	exit 1
}

command -v go >/dev/null 2>&1 || fail "go toolchain not found on PATH"
[[ -f "$REPO_ROOT/tests/contracts/e40_i04_scenario_contract_test.go" ]] ||
	fail "tests/contracts/e40_i04_scenario_contract_test.go is missing"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

OUT="$WORKDIR/go-test-output.txt"

echo "TC-108: driving the real Go contract test (go test ./tests/contracts/... -run TestExpectedEntityGraph -v)"

set +e
(cd "$REPO_ROOT" && go test ./tests/contracts/... -run 'TestExpectedEntityGraph' -v) >"$OUT" 2>&1
code=$?
set -e

[[ "$code" -eq 0 ]] || fail "go test exited $code, want 0: $(cat "$OUT")"

# (i) real, non-zero coverage: at least one PASS line for a SUBTEST (a
# "/" in the test name), not merely the top-level TestExpectedEntityGraph
# itself passing with zero children.
pass_subtest_count="$(grep -c -- '--- PASS: TestExpectedEntityGraph/' "$OUT" || true)"
[[ "$pass_subtest_count" -gt 0 ]] || fail "go test reported zero TestExpectedEntityGraph subtests -- a '-run' typo would look exactly like this false pass: $(cat "$OUT")"

fail_count="$(grep -c -- '--- FAIL:' "$OUT" || true)"
[[ "$fail_count" -eq 0 ]] || fail "go test reported $fail_count failing subtest(s): $(grep -- '--- FAIL:' "$OUT")"

# (ii) representative named subtests actually ran -- one per AC this task
# covers (spec.md AC-F11-25/-25a/-25b/-26/-27), so a stripped-down or
# accidentally-disabled subset of the Go test's own coverage is caught
# here rather than silently passing this wrapper.
declare -a want_substrings=(
	"TestExpectedEntityGraph/positive_control_valid_epic_graph"
	"TestExpectedEntityGraph/Y1_root_family_wrong_type"
	"TestExpectedEntityGraph/Y7_required_terminal_states_wrong_type"
	"TestExpectedEntityGraph/Y21_descendant_min_exceeds_max"
	"TestExpectedEntityGraph/Y26_descendants_empty_on_epic"
	"TestExpectedEntityGraph/Y27_duplicate_descendant_family"
	"TestExpectedEntityGraph/Y24_duplicate_yaml_key_in_descendant_rejected_at_parse_time"
	"TestExpectedEntityGraph/AC_F11_25b_key_completeness/missing_key_declared_in_descendants"
	"TestExpectedEntityGraph/AC_F11_25b_key_completeness/extra_key_not_a_declared_descendant"
	"TestExpectedEntityGraph/AC_F11_25b_terminal_status_read_from_own_workflow/tech_debt_completed_rejected_no_such_step"
	"TestExpectedEntityGraph/AC_F11_25b_terminal_status_read_from_own_workflow/tech_debt_resolved_accepted_real_terminal_step"
	"TestExpectedEntityGraph/requiredness_gating_across_all_six_families/epic_missing_block_rejected"
	"TestExpectedEntityGraph/requiredness_gating_across_all_six_families/task_forbidden_when_present"
	"TestExpectedEntityGraph/AC_F11_25a_four_existing_packages_remain_pure_version_bump/py-feature-recurring-tasks"
	"TestExpectedEntityGraph/AC_F11_26_no_field_borrowed_from_i01/p2p_sets"
	"TestExpectedEntityGraph/AC_F11_27_no_undeclared_reader"
)
for want in "${want_substrings[@]}"; do
	grep -q -- "--- PASS: ${want}" "$OUT" ||
		fail "expected go test output to report PASS for subtest '${want}', but it did not (see full output): $(grep -- "${want}" "$OUT" || echo '(no match at all)')"
done

echo "TC-108(go test ./tests/contracts/... -run TestExpectedEntityGraph: $pass_subtest_count subtests passed, 0 failed, all representative named subtests present) PASS"

# ---------------------------------------------------------------------------
# AC-F11-25b's key-completeness/terminal-status enforcement must ALSO gate
# the existing TC-030 committed-testdata cases (case-28/case-29,
# required-when-hierarchical and forbidden-when-flat) -- run alongside
# TestExpectedEntityGraph so a regression in either negative testdata case
# fails this same gate.
# ---------------------------------------------------------------------------
echo "TC-108: driving TestTC030_I04ScenarioPackageContract's new case-28/case-29 subtests"

OUT2="$WORKDIR/go-test-tc030-output.txt"
set +e
(cd "$REPO_ROOT" && go test ./tests/contracts/... -run 'TestTC030_I04ScenarioPackageContract' -v) >"$OUT2" 2>&1
code2=$?
set -e
[[ "$code2" -eq 0 ]] || fail "go test (TestTC030) exited $code2, want 0: $(cat "$OUT2")"

for want in \
	"case-28-expected-entity-graph-required-for-epic.yaml" \
	"case-29-expected-entity-graph-forbidden-for-flat-family.yaml"; do
	grep -q -- "--- PASS: TestTC030_I04ScenarioPackageContract/malformed_package_cases/${want}" "$OUT2" ||
		fail "expected TestTC030 to report PASS for case '${want}', but it did not: $(grep -- "${want}" "$OUT2" || echo '(no match at all)')"
done

echo "TC-108(TestTC030_I04ScenarioPackageContract: case-28/case-29 both PASS) PASS"

echo "TC-108: PASS"
