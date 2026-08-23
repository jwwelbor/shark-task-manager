#!/usr/bin/env bash
# TC-092: Full regression and complete F10 suite registration (AC-015,
# test-plan.md "TC-092: Full regression and complete F10 suite registration").
#
# This test is itself registered in run-all.sh (self-referentially), so it
# MUST NOT invoke run-all.sh -- that would recurse. Instead:
#
#   - AC-T2 (registration order) and the TC-078..TC-092 registration audit
#     are always performed here, statically, against the current
#     run-all.sh source -- the same discipline tc077 established for F09.
#   - AC-T3 (each pre-F10 test's own documented pass marker present in a
#     REAL captured run-all.sh run) is performed only when the caller
#     supplies TC092_RUN_LOG pointing at a log already captured from a
#     separate, real `run-all.sh` invocation (see bench/README.md's
#     "Full regression verification" section for the two-step sequence).
#     Without it, this test fails: AC-T3 is a required quality-gate check.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_ALL="$SCRIPT_DIR/run-all.sh"
BASELINE="$SCRIPT_DIR/testdata/pre-f10-registration.txt"

[[ -f "$BASELINE" ]] || { echo "TC-092: fixed baseline $BASELINE missing" >&2; exit 1; }

mapfile -t baseline_names <"$BASELINE"

# --- AC-T2: pre-F10 registrations still present, in unchanged relative order ---

mapfile -t current_names < <(grep -oE 'tc[0-9]+_[A-Za-z0-9_.-]+_test\.sh' "$RUN_ALL")

filtered=()
for n in "${current_names[@]}"; do
	for b in "${baseline_names[@]}"; do
		if [[ "$n" == "$b" ]]; then
			filtered+=("$n")
			break
		fi
	done
done

if [[ "${#filtered[@]}" -ne "${#baseline_names[@]}" ]]; then
	echo "TC-092: AC-T2 FAIL - pre-F10 registration count changed (baseline ${#baseline_names[@]}, found ${#filtered[@]})" >&2
	diff <(printf '%s\n' "${baseline_names[@]}") <(printf '%s\n' "${filtered[@]}") >&2 || true
	exit 1
fi
for i in "${!baseline_names[@]}"; do
	if [[ "${filtered[$i]}" != "${baseline_names[$i]}" ]]; then
		echo "TC-092: AC-T2 FAIL - pre-F10 relative order diverged at position $i (expected ${baseline_names[$i]}, found ${filtered[$i]})" >&2
		diff <(printf '%s\n' "${baseline_names[@]}") <(printf '%s\n' "${filtered[@]}") >&2 || true
		exit 1
	fi
done
echo "TC-092: AC-T2 pass (${#baseline_names[@]} pre-F10 tests present in run-all.sh, relative order unchanged)"

# --- Every pre-F10 test script still exists and still declares a discoverable pass marker ---
# (independent of whether AC-T3's captured-output check runs -- a deleted or
# marker-stripped test is caught here even without a captured log.)

extract_marker() {
	local file="$1" line
	line="$(grep -nE '(echo|print)\(? *"TC-[0-9]' "$file" 2>/dev/null | tail -1 || true)"
	[[ -n "$line" ]] || return 1
	line="${line#*:}"
	printf '%s\n' "$line" | grep -oE '"[^"]*"' | head -1 | sed -e 's/^"//' -e 's/"$//'
}

declare -A markers
fail=0
for name in "${baseline_names[@]}"; do
	script_path="$SCRIPT_DIR/$name"
	if [[ ! -f "$script_path" ]]; then
		echo "TC-092: AC-T3 FAIL - pre-F10 test file removed: $name" >&2
		fail=1
		continue
	fi
	marker="$(extract_marker "$script_path" || true)"
	if [[ -z "$marker" ]]; then
		echo "TC-092: AC-T3 FAIL - no discoverable pass marker in $name (weakened or removed)" >&2
		fail=1
		continue
	fi
	markers["$name"]="$marker"
done
[[ "$fail" -eq 0 ]] || exit 1
echo "TC-092: every pre-F10 test still declares a discoverable pass marker"

# --- AC-T3: markers (and run-all.sh's own wrapper line) present in a REAL captured run ---

if [[ -n "${TC092_RUN_LOG:-}" ]]; then
	[[ -f "$TC092_RUN_LOG" ]] || { echo "TC-092: AC-T3 FAIL - TC092_RUN_LOG=$TC092_RUN_LOG does not exist" >&2; exit 1; }
	fail=0
	for name in "${baseline_names[@]}"; do
		marker="${markers[$name]}"
		if ! grep -qF -- "$marker" "$TC092_RUN_LOG"; then
			echo "TC-092: AC-T3 FAIL - $name's own pass marker not found in captured run output: $marker" >&2
			fail=1
		fi
		if ! grep -qF -- "PASS: $name" "$TC092_RUN_LOG"; then
			echo "TC-092: AC-T3 FAIL - run-all.sh wrapper line 'PASS: $name' not found in captured run output" >&2
			fail=1
		fi
	done
	[[ "$fail" -eq 0 ]] || exit 1
	echo "TC-092: AC-T3 pass (every pre-F10 test's own marker AND run-all.sh's wrapper PASS line present in $TC092_RUN_LOG)"
else
	echo "TC-092: AC-T3 FAIL - TC092_RUN_LOG was not supplied; captured full-regression evidence is required." >&2
	exit 1
fi

# --- TC-078 through TC-092 registered in run-all.sh, deterministic order ---
# TC-078 is the Go contract test (tests/contracts/e40_f10_operator_baseline_contract_test.go),
# run under `make test`, not a bench/scripts/tests/tc0NN_*.sh entry -- REQ-F-018's schema
# validator has no shell counterpart registered here (architecture.md Component-changes row).
assert_executable_registration() {
	local id="$1"
	if [[ "$id" == "092" ]]; then
		grep -qE 'TC092_RUN_LOG=.*tc092_full-regression-registration_test\.sh' "$RUN_ALL" || {
			echo "TC-092: TC-${id} is not invoked as a quality-gate command" >&2
			exit 1
		}
		return
	fi
	grep -qE "^[[:space:]]*\\\"\\\$SCRIPT_DIR/tc${id}_[A-Za-z0-9_.-]+_test\\.sh\\\"[[:space:]]*$" "$RUN_ALL" || {
		echo "TC-092: TC-${id} has no executable test-array registration" >&2
		exit 1
	}
}
for id in 079 080 081 082 083 084 085 086 087 088 089 090 091 092 093; do
	assert_executable_registration "$id"
done
echo "TC-092: complete F10 suite (TC-079 through TC-092) registered in run-all.sh"

for id in 003 004 005 006 007 008 009 010 011 013 014 015 016 017 018 019 020 031 032 033 034 035 036 037 038 039 040 041 043 044 045 046 047 048 049 050 051 053 054 055 056 057 058 059 060 061 062 063 064 065 066 067 068 069 070 071 072 073 074 075 076 077; do
	assert_executable_registration "$id"
done
echo "TC-092: F01-F09 and complete F10 registration pass"
