#!/usr/bin/env bash
# TC-068: the held-back oracle must refuse pre-terminal access before invoking
# an adapter and must use the existing I-05 access broker after terminality.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
! rg -q 'events\[-1\]' "$SCRIPT_DIR/../run-heldback-oracle.sh" || {
  echo "TC-068: oracle must not select provenance from an unrelated last event" >&2
  exit 1
}

REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ORACLE="$REPO_ROOT/bench/scripts/run-heldback-oracle.sh"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"

[[ -x "$ORACLE" ]] || { echo "TC-068: oracle missing or not executable" >&2; exit 1; }
grep -q 'adapter_calls' "$ORACLE" || { echo "TC-068: oracle must retain adapter call accounting" >&2; exit 1; }
grep -q 'restore_checkout' "$ORACLE" || { echo "TC-068: oracle must restore checkout on post-grant failure" >&2; exit 1; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bundle" "$tmp/checkout"
echo '{"terminal_status":{"reached":false}}' > "$tmp/bundle/bundle.json"
python3 - "$tmp/bundle" "$tmp/checkout" <<'PY'
import json, sys
path, checkout = sys.argv[1:]
with open(path + "/bundle.json", "w", encoding="utf-8") as stream:
    json.dump({"terminal_status": {"reached": False}, "roots": {"agent_fixture_checkout": checkout}}, stream)
    stream.write("\n")
PY
echo '{"identity":{"run_id":"tc068"},"outcome":{"terminal":"in_progress"}}' > "$tmp/i07.jsonl"
if "$ORACLE" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --i07 "$tmp/i07.jsonl" --stage-bundle "$tmp/bundle" --checkout "$tmp/checkout" --output "$tmp/oracle.json" >/dev/null 2>"$tmp/stderr"; then echo "TC-068: pre-terminal oracle unexpectedly passed" >&2; exit 1; fi
python3 - "$tmp/oracle.json" "$tmp/checkout" <<'PY'
import json, os, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "not_run"
assert record["adapter_calls"] == 0
assert record["invalidity_reasons"][0]["code"] == "pre_terminal"
assert os.listdir(sys.argv[2]) == []
PY
echo "TC-068: pre-terminal access is refused before adapter or filesystem use"

# Caller-path proof: the evaluator must own construction and invocation of the
# oracle, while retaining the same pre-terminal result and untouched roots.
eval_output="$tmp/evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/bundle" --i07 "$tmp/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$eval_output" >/dev/null 2>"$tmp/evaluator.stderr"; then
  echo "TC-068: evaluator pre-terminal oracle unexpectedly passed" >&2
  exit 1
fi
python3 - "$eval_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["execution_oracle"]["observed_result"] == "not_run"
assert any(item["code"] == "pre_terminal" for item in record["eligibility"]["invalidity_reasons"])
PY
echo "TC-068: evaluator entrypoint retains pre-terminal oracle refusal"

# ---------------------------------------------------------------------------
# QA1-003: the block above proves exactly 1 of test-plan.md's 10 declared
# TC-068 partitions (pre-terminal). The cases below add the other 9:
# terminal-but-unauthorized, authorized post-terminal (success), adapter-
# failure, hidden-test injection (a file introduced after admission),
# symlink, traversal, broken-link, renamed-root, and cleanup fixtures --
# through the real I-05 guard and the real, registered I-04 python adapter
# (test-plan.md's Caller-Path Contract for TC-068 permits only the adapter
# executable itself to be a deterministic fixture; the guard and filesystem
# roots stay real).
CHECKOUT_SCRIPT="$REPO_ROOT/bench/scripts/checkout-scenario-fixture.sh"
BUG_PACKAGE_YAML="$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml"
BUG_PACKAGE_DIR="$(dirname "$BUG_PACKAGE_YAML")"
BUG_REFERENCE_PATCH="$BUG_PACKAGE_DIR/evaluator/reference.patch"

[[ -x "$CHECKOUT_SCRIPT" ]] || { echo "TC-068: checkout-scenario-fixture.sh missing or not executable" >&2; exit 1; }

# provision_success_checkout <dest_dir>: a real fixture-py checkout at the
# scenario's admitted base_sha with the real reference patch applied, so the
# held-back predicate can genuinely pass -- never a hand-built adapter
# response standing in for the checkout state.
provision_success_checkout() {
	local dest="$1"
	"$CHECKOUT_SCRIPT" py 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9 "$dest" >/dev/null
	git -C "$dest" apply "$BUG_REFERENCE_PATCH"
}

# write_terminal_bundle <bundle_dir> <checkout_dir> [reached_at]: an I-05
# bundle.json agreeing the lifecycle is terminal. Omitting reached_at models
# a bundle that is terminal but cannot be authorized (terminal-but-
# unauthorized), distinct from the pre-terminal case above.
write_terminal_bundle() {
	local bundle_dir="$1" checkout_dir="$2" reached_at="${3-2020-01-01T00:00:00Z}"
	mkdir -p "$bundle_dir"
	python3 - "$bundle_dir/bundle.json" "$checkout_dir" "$reached_at" <<'PY'
import json, sys
bundle_path, checkout_dir, reached_at = sys.argv[1:4]
terminal = {"reached": True}
if reached_at:
    terminal["reached_at"] = reached_at
with open(bundle_path, "w", encoding="utf-8") as stream:
    json.dump({"terminal_status": terminal, "roots": {"agent_fixture_checkout": checkout_dir}}, stream)
PY
}

write_i07() {
	local path="$1" run_id="$2"
	printf '{"identity":{"run_id":"%s"},"outcome":{"terminal":"complete"}}\n' "$run_id" >"$path"
}

# ---------------------------------------------------------------------------
echo "TC-068: terminal-but-unauthorized -- both sides agree the run is terminal but the I-05 broker cannot authorize access"
case_dir="$tmp/terminal-unauthorized"
mkdir -p "$case_dir/checkout"
write_terminal_bundle "$case_dir/bundle" "$case_dir/checkout" ""
write_i07 "$case_dir/i07.jsonl" "tc068-terminal-unauthorized"
set +e
"$ORACLE" --scenario "$BUG_PACKAGE_YAML" --i07 "$case_dir/i07.jsonl" --stage-bundle "$case_dir/bundle" --checkout "$case_dir/checkout" --output "$case_dir/oracle.json" >/dev/null 2>"$case_dir/stderr"
code=$?
set -e
[[ "$code" -ne 0 ]] || { echo "TC-068: terminal-but-unauthorized unexpectedly passed" >&2; exit 1; }
python3 - "$case_dir/oracle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["adapter_calls"] == 0, record
assert record["observed_result"] == "not_run", record
codes = [item["code"] for item in record["invalidity_reasons"]]
assert codes == ["isolation_violation"], f"expected a distinct terminal-but-unauthorized reason, not pre_terminal: {codes}"
PY
echo "TC-068: a terminal-but-unauthorizable bundle is refused (not the pre_terminal code) with zero adapter calls"

# ---------------------------------------------------------------------------
echo "TC-068: authorized post-terminal -- the real I-04 adapter and I-05 guard produce a clean, successful oracle result"
case_dir="$tmp/authorized-success"
provision_success_checkout "$case_dir/checkout"
write_terminal_bundle "$case_dir/bundle" "$case_dir/checkout"
write_i07 "$case_dir/i07.jsonl" "tc068-authorized-success"
"$ORACLE" --scenario "$BUG_PACKAGE_YAML" --i07 "$case_dir/i07.jsonl" --stage-bundle "$case_dir/bundle" --checkout "$case_dir/checkout" --output "$case_dir/oracle.json"
python3 - "$case_dir/oracle.json" "$case_dir/checkout" <<'PY'
import json, os, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "pass", record
assert record["cleanup"] is True, record
assert record["invalidity_reasons"] == [], record
assert record["adapter_calls"] == 2, record
assert record["access_event"] and len(record["access_event"]) == 1, record
leftover = sorted(set(os.listdir(sys.argv[2])) - {".git", ".gitignore", "pyproject.toml", "README.md", "taskmanager", "tests"})
assert leftover == [], f"unexpected residue left in the checkout after an authorized run: {leftover}"
PY
echo "TC-068: an authorized post-terminal evaluation records one access event, passes, and leaves no residue"

# ---------------------------------------------------------------------------
echo "TC-068: adapter-failure -- the registered I-04 adapter itself fails when invoked for the predicate"
case_dir="$tmp/adapter-failure"
provision_success_checkout "$case_dir/checkout"
cp -r "$BUG_PACKAGE_DIR" "$case_dir/scenario"
# A held-back test id that cannot be resolved to a real file makes the real
# adapter.sh fail its own `test` invocation (non-JSON stdout, exit 1) -- the
# predicate-time adapter failure, distinct from the injection-time
# isolation_violation the cleanup-failure and collision cases below exercise.
python3 - "$case_dir/scenario/package.yaml" <<'PY'
import sys
import yaml
path = sys.argv[1]
with open(path, encoding="utf-8") as stream:
    package = yaml.safe_load(stream)
package["final_predicate"]["f2p_test_ids"] = ["tests.nonexistent_module::test_ghost"]
with open(path, "w", encoding="utf-8") as stream:
    yaml.safe_dump(package, stream)
PY
write_terminal_bundle "$case_dir/bundle" "$case_dir/checkout"
write_i07 "$case_dir/i07.jsonl" "tc068-adapter-failure"
set +e
"$ORACLE" --scenario "$case_dir/scenario/package.yaml" --i07 "$case_dir/i07.jsonl" --stage-bundle "$case_dir/bundle" --checkout "$case_dir/checkout" --output "$case_dir/oracle.json" >/dev/null 2>"$case_dir/stderr"
code=$?
set -e
[[ "$code" -ne 0 ]] || { echo "TC-068: adapter-failure case unexpectedly passed" >&2; exit 1; }
python3 - "$case_dir/oracle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["adapter_calls"] == 2, record
assert record["observed_result"] == "fail", record
assert any(item["code"] == "oracle_failure" for item in record["invalidity_reasons"]), record
PY
echo "TC-068: a real adapter failure during predicate invocation is recorded as a failed result, never a fabricated pass"

# ---------------------------------------------------------------------------
echo "TC-068: hidden-test injection -- material introduced into the checkout after admission is caught as residue"
case_dir="$tmp/hidden-file"
provision_success_checkout "$case_dir/checkout"
write_terminal_bundle "$case_dir/bundle" "$case_dir/checkout"
write_i07 "$case_dir/i07.jsonl" "tc068-hidden-file"
"$ORACLE" --scenario "$BUG_PACKAGE_YAML" --i07 "$case_dir/i07.jsonl" --stage-bundle "$case_dir/bundle" --checkout "$case_dir/checkout" --output "$case_dir/oracle.json" &
oracle_pid=$!
injected_marker="$case_dir/checkout/tests/test_due_date_boundary.py"
waited=0
while [[ ! -f "$injected_marker" && "$waited" -lt 500 ]]; do
	sleep 0.02
	waited=$((waited + 1))
done
if [[ ! -f "$injected_marker" ]]; then
	kill "$oracle_pid" 2>/dev/null || true
	wait "$oracle_pid" 2>/dev/null || true
	echo "TC-068: injected oracle test never appeared; cannot exercise hidden-test injection" >&2
	exit 1
fi
# Plant hidden material into the agent-visible checkout while the oracle's
# own predicate invocation is still running, after access was granted --
# the "introduced after admission" case, distinct from a pre-existing file.
echo "hidden material introduced after admission" >"$case_dir/checkout/hidden_after_admission.txt"
set +e
wait "$oracle_pid"
code=$?
set -e
[[ "$code" -ne 0 ]] || { echo "TC-068: hidden-test injection case unexpectedly passed" >&2; exit 1; }
python3 - "$case_dir/oracle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["cleanup"] is False, record
assert record["observed_result"] == "fail", record
assert any(item["code"] == "evaluator_only_residue" and "hidden_after_admission.txt" in item["detail"] for item in record["invalidity_reasons"]), record
PY
[[ ! -e "$case_dir/checkout/hidden_after_admission.txt" ]] || { echo "TC-068: hidden file survived checkout restoration" >&2; exit 1; }
echo "TC-068: a file introduced into the checkout after admission is named as residue and the checkout is restored"

# ---------------------------------------------------------------------------
echo "TC-068: cleanup-failure -- the adapter-resolved injection destination collides with pre-existing checkout content"
case_dir="$tmp/cleanup-failure"
provision_success_checkout "$case_dir/checkout"
mkdir -p "$case_dir/checkout/tests"
printf 'pre-existing content at the adapter-resolved destination\n' >"$case_dir/checkout/tests/test_due_date_boundary.py"
write_terminal_bundle "$case_dir/bundle" "$case_dir/checkout"
write_i07 "$case_dir/i07.jsonl" "tc068-cleanup-failure"
set +e
"$ORACLE" --scenario "$BUG_PACKAGE_YAML" --i07 "$case_dir/i07.jsonl" --stage-bundle "$case_dir/bundle" --checkout "$case_dir/checkout" --output "$case_dir/oracle.json" >/dev/null 2>"$case_dir/stderr"
code=$?
set -e
[[ "$code" -ne 0 ]] || { echo "TC-068: cleanup-failure case unexpectedly passed" >&2; exit 1; }
python3 - "$case_dir/oracle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "not_run", record
assert any(item["code"] == "cleanup_failure" for item in record["invalidity_reasons"]), record
PY
diff <(printf 'pre-existing content at the adapter-resolved destination\n') "$case_dir/checkout/tests/test_due_date_boundary.py" >/dev/null || { echo "TC-068: pre-existing colliding content was not restored byte-for-byte" >&2; exit 1; }
echo "TC-068: an adapter-resolved destination collision with pre-existing content is refused and restored byte-for-byte"

# ---------------------------------------------------------------------------
# Symlink, traversal, broken-link, and renamed-root all attack
# resolve_oracle_sources' evaluator-root containment check before any
# adapter or filesystem-write occurs -- a minimal scenario package is enough
# for each, no real predicate execution needed.
write_attack_package() {
	# write_attack_package <dir> <oracle_relative_path>
	local dir="$1" oracle_rel="$2"
	python3 - "$dir/package.yaml" "$oracle_rel" <<'PY'
import sys
import yaml
path, oracle_rel = sys.argv[1:3]
package = {
    "schema_version": "1.0",
    "scenario_id": "tc068-attack",
    "scenario_version": 1,
    "entity_family": "bug",
    "fixture": {"fixture_id": "py", "submodule_path": "bench/fixture-py", "base_sha": "964fa68e4c9e0c4e0f3756d9efd78b888c558fd9"},
    "adapter": {"name": "python", "version": "1.0.0"},
    "final_predicate": {"kind": "f2p_p2p", "f2p_test_ids": ["tests.test_due_date_boundary::test_is_overdue_true_for_task_due_today"], "p2p_selection": {"include": ["tests"]}},
    "evaluator_only": {"reference_solution": "evaluator/reference.patch", "oracle_tests": [oracle_rel]},
}
with open(path, "w", encoding="utf-8") as stream:
    yaml.safe_dump(package, stream)
PY
}

run_attack_case() {
	# run_attack_case <label> <dir>
	local label="$1" dir="$2"
	mkdir -p "$dir/checkout"
	write_terminal_bundle "$dir/bundle" "$dir/checkout"
	write_i07 "$dir/i07.jsonl" "tc068-attack-$label"
	set +e
	"$ORACLE" --scenario "$dir/package.yaml" --i07 "$dir/i07.jsonl" --stage-bundle "$dir/bundle" --checkout "$dir/checkout" --output "$dir/oracle.json" >/dev/null 2>"$dir/stderr"
	local run_code=$?
	set -e
	[[ "$run_code" -ne 0 ]] || { echo "TC-068: $label attack unexpectedly passed" >&2; exit 1; }
	python3 - "$dir/oracle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["adapter_calls"] == 0, record
assert record["observed_result"] == "not_run", record
assert record["invalidity_reasons"][0]["code"] == "isolation_violation", record
PY
}

echo "TC-068: symlink -- a symlink inside the evaluator root that escapes to outside content is refused with zero adapter calls"
attack_dir="$tmp/attack-symlink"
mkdir -p "$attack_dir/evaluator"
echo "outside secret" >"$attack_dir/outside_secret.py"
ln -s ../outside_secret.py "$attack_dir/evaluator/evil_link.py"
write_attack_package "$attack_dir" "evaluator/evil_link.py"
run_attack_case symlink "$attack_dir"
echo "TC-068: symlink escape refused"

echo "TC-068: traversal -- a declared oracle path that escapes the evaluator root via .. is refused with zero adapter calls"
attack_dir="$tmp/attack-traversal"
mkdir -p "$attack_dir/evaluator"
echo "outside secret" >"$attack_dir/outside_secret.py"
write_attack_package "$attack_dir" "evaluator/../outside_secret.py"
run_attack_case traversal "$attack_dir"
echo "TC-068: traversal escape refused"

echo "TC-068: broken-link -- a dangling symlink declared as an oracle test is refused as missing, never dereferenced blindly"
attack_dir="$tmp/attack-broken-link"
mkdir -p "$attack_dir/evaluator"
ln -s ./does_not_exist.py "$attack_dir/evaluator/broken_link.py"
write_attack_package "$attack_dir" "evaluator/broken_link.py"
run_attack_case broken-link "$attack_dir"
echo "TC-068: broken (dangling) symlink refused"

echo "TC-068: renamed-root -- a scenario whose declared evaluator root has been renamed/moved never falls back to another directory"
attack_dir="$tmp/attack-renamed-root"
mkdir -p "$attack_dir/evaluator-renamed"
echo "content" >"$attack_dir/evaluator-renamed/test_x.py"
write_attack_package "$attack_dir" "evaluator/test_x.py"
run_attack_case renamed-root "$attack_dir"
echo "TC-068: renamed evaluator root refused as missing, not silently substituted"

echo "TC-068: PASS"
