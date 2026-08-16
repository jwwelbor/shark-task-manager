#!/usr/bin/env bash
# TC-048 (test-plan.md AC test matrix row AC-011; Caller-Path Contracts row
# tc048; T-E40-F06-007 task spec Test Cases, AC-T1 through AC-T4).
#
# Drives bench/scripts/verify-stage-evidence.sh's NEW --grant-access broker
# (T-E40-F06-007, REQ-F-012, ADR-F06-08) through the authorized-ordering
# state machine, plus a REAL bench/adapters/python/adapter.sh inject-tests
# subprocess invocation -- never stubbed (test-plan.md tc048 Caller-Path
# Contract: "adapter.sh inject-tests is I-04's real capability, invoked
# directly, not stubbed").
#
#   (a) inject-tests requested against a bundle whose terminal_status has
#       NOT been reached -> rejected as isolation_violation; the adapter is
#       never invoked and no file is placed (AC-T1).
#   (b) the same request against a bundle whose terminal_status HAS been
#       reached -> accepted; the real adapter places the oracle test file
#       into the checkout, and exactly one evaluator_access event is
#       appended to access.jsonl with every field populated (AC-T2).
#   (c) an in-place read of evaluator_only.reference_solution against a
#       post-terminal bundle, from a CLEAN checkout -> accepted with the
#       same one-event-append guarantee, and the digest matches the real
#       evaluator-only file's content -- never a copy left in the checkout
#       (AC-T3).
#   (d) the same read, but a byte-identical copy of the reference solution
#       was already planted in the checkout and backdated (via `touch -d`)
#       to BEFORE the bundle's own terminal_status.reached_at -> rejected
#       as isolation_violation naming the pre-completion copy, not the
#       read itself (AC-T4).
#
# A fifth check proves case (b)'s own accepted placement of a DIFFERENT
# (digest-distinct) file never trips case (d)'s copy-timing detector -- the
# task's Notes for Agent is explicit that case (d) is rejected for WHEN a
# copy of the SAME artifact happened, never merely because "some
# evaluator-only-shaped file exists in the checkout." Without this check, an
# implementation that flagged ANY evaluator-only file present in the
# checkout (rather than one matching the specific artifact under read)
# could pass AC-T4 while silently breaking AC-T2's legitimate placement.
#
# Caller-Path Contract (test-plan.md tc048): a real adapter.sh subprocess
# invocation for the post-terminal injection case (b), and a real file copy
# (not a metadata flag) for the negative pre-completion-copy case (d).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
FIXTURE="$SCRIPTS_DIR/testdata/evidence/access"
ORACLE_TEST_SRC="$FIXTURE/evaluator-only/oracle_tests/test_oracle_case.py"
REFERENCE_SOLUTION_SRC="$FIXTURE/evaluator-only/reference_solution.patch"

fail() {
	echo "TC-048 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $GUARD"
[[ -x "$ADAPTER" ]] || fail "python adapter missing or not executable: $ADAPTER"
[[ -f "$ORACLE_TEST_SRC" ]] || fail "fixture missing oracle test file: $ORACLE_TEST_SRC"
[[ -f "$REFERENCE_SOLUTION_SRC" ]] || fail "fixture missing reference solution: $REFERENCE_SOLUTION_SRC"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# Every case gets its own fresh copies -- the guard writes access.jsonl into
# the bundle dir and the adapter writes into the checkout, so running
# against the committed fixture in place would dirty the repo and make the
# test non-idempotent (unlike tc046, which only reads its fixture).
new_bundle() {
	local variant="$1" name="$2"
	local dest="$WORKDIR/$name"
	cp -r "$FIXTURE/$variant" "$dest"
	echo "$dest"
}

new_checkout() {
	local name="$1"
	local dest="$WORKDIR/$name"
	cp -r "$FIXTURE/worker-checkout" "$dest"
	echo "$dest"
}

count_lines() {
	[[ -f "$1" ]] || {
		echo 0
		return
	}
	wc -l <"$1" | tr -d ' '
}

# ---------------------------------------------------------------------------
echo "TC-048 (a): inject-tests before terminal status -> isolation_violation, nothing placed"

bundle_a="$(new_bundle pre-terminal bundle-a)"
checkout_a="$(new_checkout checkout-a)"

out_a="$WORKDIR/out-a.json"
err_a="$WORKDIR/err-a.txt"
set +e
"$GUARD" "$bundle_a" --grant-access inject-tests --accessor execution_oracle \
	--adapter "$ADAPTER" --checkout "$checkout_a" --files "$ORACLE_TEST_SRC" \
	>"$out_a" 2>"$err_a"
code_a=$?
set -e

[[ "$code_a" -eq 1 ]] || fail "case (a): guard exited $code_a, want 1 (isolation_violation); stderr: $(cat "$err_a")"
grep -q "isolation_violation" "$err_a" || fail "case (a): stderr does not name isolation_violation: $(cat "$err_a")"
[[ ! -s "$out_a" ]] || fail "case (a): guard printed stdout on a rejection: $(cat "$out_a")"
[[ "$(count_lines "$bundle_a/access.jsonl")" -eq 0 ]] || fail "case (a): access.jsonl gained an event on a rejected pre-terminal grant"
[[ ! -e "$checkout_a/tests/test_oracle_case.py" ]] || fail "case (a): adapter placed the oracle test into the checkout despite the pre-terminal rejection -- a rejected request must never reach adapter.sh"

echo "TC-048(AC-T1: pre-terminal inject-tests rejected as isolation_violation before any placement or event) PASS"

# ---------------------------------------------------------------------------
echo "TC-048 (b): inject-tests after terminal status -> accepted via real adapter.sh, one evaluator_access event"

bundle_b="$(new_bundle post-terminal bundle-b)"
checkout_b="$(new_checkout checkout-b)"

out_b="$WORKDIR/out-b.json"
err_b="$WORKDIR/err-b.txt"
set +e
"$GUARD" "$bundle_b" --grant-access inject-tests --accessor execution_oracle \
	--adapter "$ADAPTER" --checkout "$checkout_b" --files "$ORACLE_TEST_SRC" \
	>"$out_b" 2>"$err_b"
code_b=$?
set -e

[[ "$code_b" -eq 0 ]] || fail "case (b): guard exited $code_b, want 0; stderr: $(cat "$err_b")"
[[ -f "$checkout_b/tests/test_oracle_case.py" ]] || fail "case (b): adapter.sh did not place the oracle test into the checkout via the real inject-tests capability"
diff -q "$ORACLE_TEST_SRC" "$checkout_b/tests/test_oracle_case.py" >/dev/null || fail "case (b): placed file content differs from the source oracle test"
[[ "$(count_lines "$bundle_b/access.jsonl")" -eq 1 ]] || fail "case (b): access.jsonl has $(count_lines "$bundle_b/access.jsonl") line(s) after one grant, want exactly 1"

python3 -c "
import json
with open('$bundle_b/access.jsonl') as f:
    event = json.loads(f.readline())
for field in ('accessor', 'artifact_path', 'digest', 'phase', 'granted_at'):
    assert event.get(field), f'AC-T2 VIOLATION: evaluator_access event missing/empty field {field!r}: {event!r}'
assert event['phase'] == 'post_terminal', f'AC-T2 VIOLATION: event phase is {event[\"phase\"]!r}, want post_terminal'
assert event['accessor'] == 'execution_oracle', f'unexpected accessor: {event[\"accessor\"]!r}'
" || fail "case (b): appended event failed field assertions"

echo "TC-048(AC-T2: post-terminal inject-tests accepted through the real adapter, one evaluator_access event appended with all fields populated) PASS"

# ---------------------------------------------------------------------------
echo "TC-048 (c): in-place oracle read after terminal status, clean checkout -> accepted, one evaluator_access event"

bundle_c="$(new_bundle post-terminal bundle-c)"
checkout_c="$(new_checkout checkout-c)"

out_c="$WORKDIR/out-c.json"
err_c="$WORKDIR/err-c.txt"
set +e
"$GUARD" "$bundle_c" --grant-access in-place-read --accessor execution_oracle \
	--evaluator-root "$FIXTURE/evaluator-only" --artifact reference_solution.patch \
	--checkout "$checkout_c" \
	>"$out_c" 2>"$err_c"
code_c=$?
set -e

[[ "$code_c" -eq 0 ]] || fail "case (c): guard exited $code_c, want 0; stderr: $(cat "$err_c")"
[[ "$(count_lines "$bundle_c/access.jsonl")" -eq 1 ]] || fail "case (c): access.jsonl has $(count_lines "$bundle_c/access.jsonl") line(s) after one grant, want exactly 1"

python3 -c "
import hashlib, json
with open('$bundle_c/access.jsonl') as f:
    event = json.loads(f.readline())
for field in ('accessor', 'artifact_path', 'digest', 'phase', 'granted_at'):
    assert event.get(field), f'AC-T3 VIOLATION: evaluator_access event missing/empty field {field!r}: {event!r}'
assert event['phase'] == 'post_terminal', f'AC-T3 VIOLATION: event phase is {event[\"phase\"]!r}, want post_terminal'
with open('$REFERENCE_SOLUTION_SRC', 'rb') as f:
    want_digest = 'sha256:' + hashlib.sha256(f.read()).hexdigest()
assert event['digest'] == want_digest, f'AC-T3 VIOLATION: event digest {event[\"digest\"]!r} does not match the real evaluator-only file content {want_digest!r}'
" || fail "case (c): appended event failed field/digest assertions"

# The read is in place: no copy of the reference solution was ever written
# into the worker checkout by the guard itself.
leaked="$(find "$checkout_c" -type f ! -name 'README.md')"
[[ -z "$leaked" ]] || fail "case (c): file(s) other than the checkout's own README.md found after an in-place read -- the guard must not copy anything into the checkout: $leaked"

echo "TC-048(AC-T3: post-terminal in-place read accepted, digest matches the real evaluator-only file, one evaluator_access event appended, no copy left behind) PASS"

# ---------------------------------------------------------------------------
echo "TC-048 (d): same read, but a pre-completion copy already sits in the checkout -> rejected naming the violation"

bundle_d="$(new_bundle post-terminal bundle-d)"
checkout_d="$(new_checkout checkout-d)"

# Plant a byte-identical copy of the evaluator-only reference solution
# inside the agent-visible checkout, backdated (real `touch -d`, not a
# metadata flag) to before the bundle's own terminal_status.reached_at
# (2020-01-01T00:00:00Z) -- modeling a copy that happened DURING dispatch,
# before execution completed (task Notes for Agent: rejected for WHEN the
# copy happened, not merely that a copy exists).
leaked_copy="$checkout_d/leaked_reference_solution.patch"
cp "$REFERENCE_SOLUTION_SRC" "$leaked_copy"
touch -d "2019-06-01T00:00:00Z" "$leaked_copy"

out_d="$WORKDIR/out-d.json"
err_d="$WORKDIR/err-d.txt"
set +e
"$GUARD" "$bundle_d" --grant-access in-place-read --accessor execution_oracle \
	--evaluator-root "$FIXTURE/evaluator-only" --artifact reference_solution.patch \
	--checkout "$checkout_d" \
	>"$out_d" 2>"$err_d"
code_d=$?
set -e

[[ "$code_d" -eq 1 ]] || fail "case (d): guard exited $code_d, want 1 (isolation_violation); stderr: $(cat "$err_d")"
grep -q "isolation_violation" "$err_d" || fail "case (d): stderr does not name isolation_violation: $(cat "$err_d")"
grep -q "leaked_reference_solution.patch" "$err_d" || fail "case (d): stderr does not name the offending pre-completion copy path (must name the copy, not the read): $(cat "$err_d")"
[[ ! -s "$out_d" ]] || fail "case (d): guard printed stdout on a rejection: $(cat "$out_d")"
[[ "$(count_lines "$bundle_d/access.jsonl")" -eq 0 ]] || fail "case (d): access.jsonl gained an event on a rejected read"

echo "TC-048(AC-T4: pre-completion copy detected via its real backdated mtime and rejected naming the violation; no event appended) PASS"

# ---------------------------------------------------------------------------
echo "TC-048: case (b)'s accepted inject-tests placement of a different artifact must not trip an in-place-read grant"

out_bd="$WORKDIR/out-bd.json"
err_bd="$WORKDIR/err-bd.txt"
set +e
"$GUARD" "$bundle_b" --grant-access in-place-read --accessor execution_oracle \
	--evaluator-root "$FIXTURE/evaluator-only" --artifact reference_solution.patch \
	--checkout "$checkout_b" \
	>"$out_bd" 2>"$err_bd"
code_bd=$?
set -e
[[ "$code_bd" -eq 0 ]] || fail "case (b)+read: guard exited $code_bd, want 0 -- case (b)'s accepted, digest-DISTINCT oracle-test placement must never be mistaken for a reference-solution leak; stderr: $(cat "$err_bd")"
[[ "$(count_lines "$bundle_b/access.jsonl")" -eq 2 ]] || fail "case (b)+read: access.jsonl has $(count_lines "$bundle_b/access.jsonl") line(s) after two accepted grants, want exactly 2"

echo "TC-048: case (b) placement and this in-place read coexist without a false isolation_violation PASS"

echo "TC-048: PASS"
