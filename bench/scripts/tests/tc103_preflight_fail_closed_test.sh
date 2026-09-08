#!/usr/bin/env bash
# TC-103 / T-E40-F11-010 (spec.md REQ-F-004, AC-F11-09, AC-F11-10,
# AC-F11-12; test-plan.md TC-09..12; §7.3.3's 9-case decision table).
#
# Caller-Path Contract (test-plan.md TC-09..12 row): production entrypoint
# is `e40-benchmark.sh preflight --config <cfg>` -- real ledger/config
# state, `resolution-ledger.jsonl` reads never mocked. The only stub is the
# `shark` binary itself (same precedent as tc061/tc079/tc094/tc105/tc109),
# used to drive controlled route-resolution outcomes.
#
# AC-F11-09 (§7.3.3): `status == "pass"` requires all seven of P1-P7 true;
# falsifying any one alone yields `status == "blocked"` with a `blockers[]`
# entry naming requirement/scenario/cause. This file proves:
#   - the all-true "pass" case (the load-bearing positive control every
#     negative case below is a single mutation away from);
#   - P2 falsified alone (an illegal transition -> route_defect);
#   - P3 falsified alone via `unresolved_gate` -- the exact X-13 case this
#     task exists to close: route resolution succeeds (`resolution:
#     "resolved"`) but the scenario is still reported blocked, never
#     provider-ready, because `unresolved_gate` is outside the declared
#     success set;
#   - P4 falsified alone (adapter unconfigured);
#   - P5 falsified alone (i05_bundle_dir unconfigured).
# P1 (`missing_ledger_record`) has no reachable seam through the real
# `run-lifecycle-batch.sh` preview driver -- by inspection, every one of
# its code paths (the resolve loop and both `continue` branches) calls
# `emit_ledger_record` unconditionally, so a selected scenario is never
# silently absent from `resolution-ledger.jsonl` (that absence is exactly
# what AC-F11-11 requires it not do, on the *other* two conditions it does
# reach). P1 is instead proven directly against
# `evaluate_ledger_conditions()`, the pure function `_cmd_preflight_locked`
# uses to derive `blockers[]`/`diagnostics[]` from an already-parsed
# ledger -- loaded from the real, unmodified module (same
# `importlib`-from-source-file technique tc094's own fixture harness
# already uses for internal-seam tests), never a re-implementation.
# P6/P7 (replay requirement, fixture/evaluator isolation) are exercised by
# this file's AC-F11-12 reproduction below, which trips both for real.
#
# AC-F11-10: `pass_with_dry_run_limitations` is asserted absent from the
# status vocabulary across every scaffold this file drives, and the
# diagnostics[] type vocabulary is asserted closed to its 5 named entries
# (structural checks, not a 30-case cartesian sweep -- test-plan.md's
# 5x2x3=30 count is a closure property, not 30 CLI invocations).
#
# AC-F11-12: replays the 2026-09-04 conditions -- a feature scenario
# without a replay result, two scenarios failing evaluator collection, and
# one scenario ending in a real route defect -- and asserts exactly four
# distinct blocker causes plus `provider_ready == false`.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
MODULE="$SCRIPTS_DIR/lib/e40_benchmark.py"
DIGEST_PATH_BIN="$SCRIPTS_DIR/lib/digest_path"

fail() {
	echo "TC-103 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh missing or not executable"
[[ -f "$MODULE" ]] || fail "lib/e40_benchmark.py is missing"
[[ -x "$DIGEST_PATH_BIN" ]] || fail "lib/digest_path missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# Shared scaffold: a single task-family scenario (py-task-delete-task --
# no D01-D05 prelude, so P6 is trivially satisfied; a real fixture
# checkout + real evaluator isolation check, so P7 is satisfied for real,
# not stubbed) with runtime.lifecycle_adapter/provider_command/
# i05_bundle_dir all configured, and a stub `shark` giving full control
# over the route-resolution outcome the ledger records.
# ===========================================================================
TASK_ONLY_INDEX="$WORKDIR/task-only-index.yaml"
cat >"$TASK_ONLY_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $BENCH_DIR/scenarios/packages/py-task-delete-task
EOF

READY_ROOT="$WORKDIR/ready-setup"
"$OPERATOR" setup --out "$READY_ROOT" --scenario-index "$TASK_ONLY_INDEX" \
	>"$WORKDIR/ready-setup.out" 2>&1 \
	|| fail "real 'e40-benchmark.sh setup' failed: $(cat "$WORKDIR/ready-setup.out")"
TASK_KEY="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['root_keys']['py-task-delete-task'])" "$READY_ROOT/setup-result.json")"
[[ "$TASK_KEY" =~ ^T-E[0-9]{2}-F[0-9]{2}-[0-9]{3}$ ]] || fail "seeded task root_key is not a task key: $TASK_KEY"

DUMMY_ADAPTER="$WORKDIR/dummy-adapter.sh"
cat >"$DUMMY_ADAPTER" <<'SH'
#!/usr/bin/env bash
# Never actually invoked -- preflight makes zero provider calls in every
# argument combination (AC-F11-08). This only needs to exist and be
# executable for P4's readiness check to pass.
exit 0
SH
chmod +x "$DUMMY_ADAPTER"
I05_DIR="$WORKDIR/i05"
mkdir -p "$I05_DIR"

configure_runtime() {
	# configure_runtime <config_path> <adapter-or-empty> <i05-dir-or-empty>
	local config_path="$1" adapter="$2" i05dir="$3"
	python3 - "$config_path" "$adapter" "$i05dir" <<'PY'
import pathlib
import sys
import yaml

path, adapter, i05dir = pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3]
config = yaml.safe_load(path.read_text(encoding="utf-8"))
if adapter:
    config["runtime"]["lifecycle_adapter"] = adapter
    config["runtime"]["provider_command"] = [adapter]
else:
    config["runtime"]["lifecycle_adapter"] = None
    config["runtime"]["provider_command"] = None
for entry in config["scenario_roots"].values():
    entry["i05_bundle_dir"] = i05dir or None
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY
}

write_stub() {
	# write_stub <scratch_template_dir> <next_action_json_or_default> <advance_ok(1/0)>
	local scratch="$1" next_body="$2" advance_ok="$3"
	cat >"$scratch/shark" <<STUB
#!/usr/bin/env bash
set -euo pipefail
python3 - "\$@" <<PY
import json, sys
args = sys.argv[1:]
task_key = "$TASK_KEY"
if args[:2] == ["next", task_key]:
    print(json.dumps($next_body))
elif args[:2] == ["claim", task_key]:
    print(json.dumps({"session_id": "sess-tc103"}))
elif args[:2] == ["status", "advance"] and args[2] == task_key:
    if $advance_ok:
        print(json.dumps({"advanced": True}))
    else:
        sys.stderr.write("Error: outcome 'pass' has no configured target for this step\\n")
        sys.exit(1)
elif args[:2] == ["release", task_key]:
    print(json.dumps({"released": True}))
else:
    sys.stderr.write("stub shark: unexpected argv: " + repr(args) + "\\n")
    sys.exit(2)
PY
STUB
	chmod +x "$scratch/shark"
	local digest
	digest="$("$DIGEST_PATH_BIN" "$scratch/shark")"
	python3 - "$scratch/../setup-result.json" "$digest" <<'PY'
import json
import sys

path, digest = sys.argv[1:3]
with open(path, encoding="utf-8") as f:
    result = json.load(f)
result["shark_binary"]["sha256"] = digest
with open(path, "w", encoding="utf-8") as f:
    json.dump(result, f)
PY
}

SPAWN_BODY='{"entity_key": "'"$TASK_KEY"'", "status": "development", "action": "spawn_agent", "agent_type": "developer", "provider": "anthropic", "model": "sonnet", "effort": "high", "error": ""}'
UNRESOLVED_GATE_BODY='{"entity_key": "'"$TASK_KEY"'", "status": "development", "action": "unresolved_gate", "error": "authorized human input required"}'

json_field() {
	python3 -c "import json,sys; print(json.load(open(sys.argv[1])).get(sys.argv[2], ''))" "$1" "$2"
}

blockers_json() {
	python3 -c "import json,sys; print(json.dumps(json.load(open(sys.argv[1]))['blockers']))" "$1"
}

# ===========================================================================
# Positive control: all seven P1-P7 conditions true -> status == "pass".
# ===========================================================================
echo "TC-103: positive control -- all seven §7.3.3 conditions true"

write_stub "$READY_ROOT/scratch-template" "$SPAWN_BODY" 1
configure_runtime "$READY_ROOT/e40-demo.yaml" "$DUMMY_ADAPTER" "$I05_DIR"

set +e
"$OPERATOR" preflight --config "$READY_ROOT/e40-demo.yaml" --reps 1 --out "$WORKDIR/preflight-pass" \
	>"$WORKDIR/preflight-pass.out" 2>"$WORKDIR/preflight-pass.err"
pass_rc=$?
set -e
[[ "$pass_rc" -eq 0 ]] \
	|| fail "positive control exited $pass_rc, want 0: $(cat "$WORKDIR/preflight-pass.err")"
PASS_RESULT="$WORKDIR/preflight-pass/preflight-result.json"
[[ -f "$PASS_RESULT" ]] || fail "positive control did not write preflight-result.json"
[[ "$(json_field "$PASS_RESULT" status)" == "pass" ]] \
	|| fail "positive control status is not 'pass': $(cat "$PASS_RESULT")"
[[ "$(blockers_json "$PASS_RESULT")" == "[]" ]] \
	|| fail "positive control reported blockers: $(blockers_json "$PASS_RESULT")"
echo "TC-103(positive control: status=pass, blockers=[], exit 0) PASS"

# ===========================================================================
# P2 falsified alone: an illegal transition (no configured target for
# outcome pass) -> resolution=failed/cause_class=route_defect.
# ===========================================================================
echo "TC-103: P2 falsified alone -- route_defect"

write_stub "$READY_ROOT/scratch-template" "$SPAWN_BODY" 0
set +e
"$OPERATOR" preflight --config "$READY_ROOT/e40-demo.yaml" --reps 1 --out "$WORKDIR/preflight-p2" \
	>"$WORKDIR/preflight-p2.out" 2>"$WORKDIR/preflight-p2.err"
p2_rc=$?
set -e
[[ "$p2_rc" -eq 1 ]] || fail "P2 case exited $p2_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-p2.err")"
P2_RESULT="$WORKDIR/preflight-p2/preflight-result.json"
[[ "$(json_field "$P2_RESULT" status)" == "blocked" ]] || fail "P2 case status is not 'blocked'"
python3 - "$P2_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
blockers = result["blockers"]
if len(blockers) != 1:
    raise SystemExit(f"expected exactly 1 blocker (P2 alone), got {len(blockers)}: {blockers!r}")
blocker = blockers[0]
if blocker["requirement"] != "P2" or blocker["cause"] != "route_defect":
    raise SystemExit(f"expected P2/route_defect, got {blocker!r}")
if "no configured target" not in blocker["detail"]:
    raise SystemExit(f"blocker detail does not name the illegal-transition cause: {blocker!r}")
PY
echo "TC-103(P2 falsified alone: exactly 1 blocker, P2/route_defect) PASS"

# ===========================================================================
# P3 falsified alone via `unresolved_gate` -- the X-13 case: route
# resolution itself succeeds (resolution: "resolved"), yet the scenario
# must still read as blocked, never provider-ready.
# ===========================================================================
echo "TC-103: P3 falsified alone -- unresolved_gate is a hard blocker, never a silent pass"

write_stub "$READY_ROOT/scratch-template" "$UNRESOLVED_GATE_BODY" 1
set +e
"$OPERATOR" preflight --config "$READY_ROOT/e40-demo.yaml" --reps 1 --out "$WORKDIR/preflight-p3" \
	>"$WORKDIR/preflight-p3.out" 2>"$WORKDIR/preflight-p3.err"
p3_rc=$?
set -e
[[ "$p3_rc" -eq 1 ]] || fail "P3 (unresolved_gate) case exited $p3_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-p3.err")"
P3_RESULT="$WORKDIR/preflight-p3/preflight-result.json"
[[ "$(json_field "$P3_RESULT" status)" == "blocked" ]] || fail "P3 case status is not 'blocked'"
LEDGER_P3="$(find "$WORKDIR/preflight-p3" -name "resolution-ledger.jsonl" -print -quit)"
[[ -n "$LEDGER_P3" ]] || fail "P3 case did not produce resolution-ledger.jsonl"
python3 - "$LEDGER_P3" "$P3_RESULT" <<'PY'
import json
import sys

ledger_path, result_path = sys.argv[1:3]
with open(ledger_path, encoding="utf-8") as f:
    records = [json.loads(line) for line in f if line.strip()]
if len(records) != 1:
    raise SystemExit(f"expected exactly one ledger record, got {len(records)}: {records!r}")
record = records[0]
# The exact X-13 property: route resolution succeeded ("resolved"), so this
# is NOT a route defect -- the benchmark never invents a decision or hides
# the blocking Question in transcript-only state, but it also never reports
# the scenario provider-ready while an authorized human input is missing.
if record["resolution"] != "resolved":
    raise SystemExit(f"expected resolution=resolved (unresolved_gate is a real, non-error terminal), got {record['resolution']!r}")
if record["terminal"] != "unresolved_gate":
    raise SystemExit(f"expected terminal=unresolved_gate, got {record['terminal']!r}")

result = json.load(open(result_path, encoding="utf-8"))
blockers = result["blockers"]
if len(blockers) != 1:
    raise SystemExit(f"expected exactly 1 blocker (P3 alone), got {len(blockers)}: {blockers!r}")
blocker = blockers[0]
if blocker["requirement"] != "P3" or blocker["cause"] != "terminal_not_in_success_set":
    raise SystemExit(f"expected P3/terminal_not_in_success_set, got {blocker!r}")
if "unresolved_gate" not in blocker["detail"]:
    raise SystemExit(f"blocker detail does not name unresolved_gate: {blocker!r}")
PY
echo "TC-103(X-13/P3: unresolved_gate resolves successfully but is still a hard blocker, never reported provider-ready) PASS"

# ===========================================================================
# P4 falsified alone: adapter unconfigured.
# ===========================================================================
echo "TC-103: P4 falsified alone -- provider_not_ready"

write_stub "$READY_ROOT/scratch-template" "$SPAWN_BODY" 1
configure_runtime "$READY_ROOT/e40-demo.yaml" "" "$I05_DIR"
set +e
"$OPERATOR" preflight --config "$READY_ROOT/e40-demo.yaml" --reps 1 --out "$WORKDIR/preflight-p4" \
	>"$WORKDIR/preflight-p4.out" 2>"$WORKDIR/preflight-p4.err"
p4_rc=$?
set -e
[[ "$p4_rc" -eq 1 ]] || fail "P4 case exited $p4_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-p4.err")"
P4_RESULT="$WORKDIR/preflight-p4/preflight-result.json"
[[ "$(json_field "$P4_RESULT" status)" == "blocked" ]] || fail "P4 case status is not 'blocked'"
[[ "$(json_field "$P4_RESULT" provider_ready)" == "False" ]] || fail "P4 case provider_ready is not False"
python3 - "$P4_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
blockers = result["blockers"]
if len(blockers) != 1:
    raise SystemExit(f"expected exactly 1 blocker (P4 alone), got {len(blockers)}: {blockers!r}")
blocker = blockers[0]
if blocker["requirement"] != "P4" or blocker["cause"] != "provider_not_ready":
    raise SystemExit(f"expected P4/provider_not_ready, got {blocker!r}")
PY
echo "TC-103(P4 falsified alone: exactly 1 blocker, P4/provider_not_ready) PASS"

# ===========================================================================
# P5 falsified alone: scenario_roots.<id>.i05_bundle_dir unconfigured.
# ===========================================================================
echo "TC-103: P5 falsified alone -- missing_runtime_input"

configure_runtime "$READY_ROOT/e40-demo.yaml" "$DUMMY_ADAPTER" ""
set +e
"$OPERATOR" preflight --config "$READY_ROOT/e40-demo.yaml" --reps 1 --out "$WORKDIR/preflight-p5" \
	>"$WORKDIR/preflight-p5.out" 2>"$WORKDIR/preflight-p5.err"
p5_rc=$?
set -e
[[ "$p5_rc" -eq 1 ]] || fail "P5 case exited $p5_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-p5.err")"
P5_RESULT="$WORKDIR/preflight-p5/preflight-result.json"
[[ "$(json_field "$P5_RESULT" status)" == "blocked" ]] || fail "P5 case status is not 'blocked'"
python3 - "$P5_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
blockers = result["blockers"]
if len(blockers) != 1:
    raise SystemExit(f"expected exactly 1 blocker (P5 alone), got {len(blockers)}: {blockers!r}")
blocker = blockers[0]
if blocker["requirement"] != "P5" or blocker["cause"] != "missing_runtime_input":
    raise SystemExit(f"expected P5/missing_runtime_input, got {blocker!r}")
if not result["missing_real_runtime_inputs"]:
    raise SystemExit("missing_real_runtime_inputs did not carry the P5 finding")
diagnostics = result["diagnostics"]
matches = [d for d in diagnostics if d["type"] == "missing_runtime_input"]
if len(matches) != 1 or matches[0]["spend_eligible"] is not False:
    raise SystemExit(f"expected exactly 1 missing_runtime_input diagnostic with spend_eligible=False, got {diagnostics!r}")
PY
echo "TC-103(P5 falsified alone: exactly 1 blocker, P5/missing_runtime_input, mirrored into diagnostics[]) PASS"

# restore the shared scaffold to its ready state so nothing downstream is
# left half-configured
configure_runtime "$READY_ROOT/e40-demo.yaml" "$DUMMY_ADAPTER" "$I05_DIR"
write_stub "$READY_ROOT/scratch-template" "$SPAWN_BODY" 1

# ===========================================================================
# AC-F11-10: `pass_with_dry_run_limitations` is never producible; every
# emitted `status` is one of exactly {pass, blocked, preview_failed}.
# ===========================================================================
echo "TC-103: AC-F11-10 -- status vocabulary is closed to 3 values"

for result_file in "$PASS_RESULT" "$P2_RESULT" "$P3_RESULT" "$P4_RESULT" "$P5_RESULT"; do
	status="$(json_field "$result_file" status)"
	case "$status" in
	pass | blocked | preview_failed) ;;
	*) fail "status $status (from $result_file) is outside the closed 3-value vocabulary" ;;
	esac
done
echo "TC-103(status vocabulary closed to pass|blocked|preview_failed across every scaffold above) PASS"

# ===========================================================================
# AC-F11-09 P1 (missing_ledger_record): unreachable via the real preview
# driver (see file header) -- proven directly against the pure
# evaluate_ledger_conditions() function the real _cmd_preflight_locked uses,
# loaded from the unmodified module source, never re-implemented.
# ===========================================================================
echo "TC-103: P1 (missing_ledger_record) -- direct proof against evaluate_ledger_conditions()"

python3 - "$MODULE" <<'PY'
import importlib.util
import sys

spec = importlib.util.spec_from_file_location("e40_benchmark", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

matrix = [{"scenario_id": "present-scenario"}, {"scenario_id": "absent-scenario"}]
ledger = {
    "present-scenario": {
        "resolution": "resolved",
        "cause_class": None,
        "terminal": "complete",
        "reason": "",
    }
}
blockers, diagnostics = module.evaluate_ledger_conditions(matrix, ledger)
matches = [b for b in blockers if b["scenario_id"] == "absent-scenario"]
if len(matches) != 1:
    raise SystemExit(f"expected exactly 1 blocker for the absent scenario, got {matches!r}")
blocker = matches[0]
if blocker["requirement"] != "P1" or blocker["cause"] != "missing_ledger_record":
    raise SystemExit(f"expected P1/missing_ledger_record, got {blocker!r}")
if any(b["scenario_id"] == "present-scenario" for b in blockers):
    raise SystemExit("the present, resolved scenario must not also blocker")
diag_matches = [d for d in diagnostics if d["type"] == "missing_ledger_record"]
if len(diag_matches) != 1 or diag_matches[0]["spend_eligible"] is not False:
    raise SystemExit(f"expected exactly 1 missing_ledger_record diagnostic with spend_eligible=False, got {diagnostics!r}")
print("P1 unit proof OK")
PY
echo "TC-103(P1 falsified alone: evaluate_ledger_conditions() blockers a selected scenario absent from the ledger, never a silent omission) PASS"

# ===========================================================================
# AC-F11-12: reproduce the exact 2026-09-04 conditions -- a feature scenario
# without a replay result, two scenarios failing evaluator collection, and
# one scenario ending in a real route defect -- and assert exactly four
# distinct blocker causes plus provider_ready == false.
# ===========================================================================
echo "TC-103: AC-F11-12 -- reproduce the 2026-09-04 four-blocker state"

REPRO_INDEX="$WORKDIR/repro-index.yaml"
REPRO_PACKAGES="$WORKDIR/repro-packages"
mkdir -p "$REPRO_PACKAGES"

# (1) A feature scenario whose OWN replay_reference bundle no longer covers
# every applicable D01-D05 stage, and for which no prepare-replay attempt
# has ever been run -- P6/missing_replay.
cp -a "$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks" "$REPRO_PACKAGES/py-feature-no-replay"
python3 - "$REPRO_PACKAGES/py-feature-no-replay/package.yaml" <<'PY'
import sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
needle = 'scenario_id: "py-feature-recurring-tasks"'
if needle not in text:
    raise SystemExit(f"expected {needle!r} in copied package.yaml")
open(path, "w", encoding="utf-8").write(text.replace(needle, 'scenario_id: "py-feature-no-replay"'))
PY
rm -f "$REPRO_PACKAGES/py-feature-no-replay/evaluator/replay/reference-bundle.json"

# (2)/(3) Two scenarios (bug, change_card families -- distinct fixtures not
# required, only distinct scenario_ids) each declaring an evaluator_only
# entry that does not exist under their own evaluator/ subtree -- a real,
# unmocked verify-evidence-roots.sh rejection for each -- P7/
# evaluator_collection_failure, twice.
for src in py-bug-due-date-boundary py-change-priority-scale; do
	dest="repro-leak-${src}"
	cp -a "$BENCH_DIR/scenarios/packages/$src" "$REPRO_PACKAGES/$dest"
	python3 - "$REPRO_PACKAGES/$dest/package.yaml" "$src" "$dest" <<'PY'
import sys
import yaml

path, old_id, new_id = sys.argv[1:4]
with open(path, encoding="utf-8") as f:
    text = f.read()
pkg = yaml.safe_load(text)
text = text.replace(f'scenario_id: "{old_id}"', f'scenario_id: "{new_id}"')
evaluator_only = pkg.get("evaluator_only") or {}
missing_ref = "evaluator/does-not-exist.patch"
if evaluator_only.get("reference_solution"):
    old_ref = evaluator_only["reference_solution"]
    text = text.replace(f"reference_solution: {old_ref}", f"reference_solution: {missing_ref}")
open(path, "w", encoding="utf-8").write(text)
PY
done

# (4) A scenario reaching a genuine route defect via a stubbed shark --
# P2/route_defect (reproducing "one scenario ending terminal=error": the
# resolve_route code path that observes an explicit `action: "error"`
# response sets `route_defect_reason` before ever reaching the terminal
# assignment, so this is the real, reachable shape of "ends in error").
cp -a "$BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation" "$REPRO_PACKAGES/repro-error-scenario"
python3 - "$REPRO_PACKAGES/repro-error-scenario/package.yaml" <<'PY'
import sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
needle = 'scenario_id: "py-techdebt-consolidate-validation"'
if needle not in text:
    raise SystemExit(f"expected {needle!r} in copied package.yaml")
open(path, "w", encoding="utf-8").write(text.replace(needle, 'scenario_id: "repro-error-scenario"'))
PY

cat >"$REPRO_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $REPRO_PACKAGES/py-feature-no-replay
  - $REPRO_PACKAGES/repro-leak-py-bug-due-date-boundary
  - $REPRO_PACKAGES/repro-leak-py-change-priority-scale
  - $REPRO_PACKAGES/repro-error-scenario
EOF

REPRO_ROOT="$WORKDIR/repro-setup"
"$OPERATOR" setup --out "$REPRO_ROOT" --scenario-index "$REPRO_INDEX" \
	>"$WORKDIR/repro-setup.out" 2>&1 \
	|| fail "real 'e40-benchmark.sh setup' (repro) failed: $(cat "$WORKDIR/repro-setup.out")"

# P5 satisfied for every scenario here (i05_bundle_dir configured) -- the
# reproduction targets exactly the four 2026-09-04 causes, not a fifth.
# P4 (provider_ready) is deliberately left unconfigured: AC-F11-12 requires
# provider_ready == false alongside the four named causes.
configure_runtime "$REPRO_ROOT/e40-demo.yaml" "" "$I05_DIR"

python3 -c "
import json,sys
root_keys = json.load(open(sys.argv[1]))['root_keys']
for k, v in root_keys.items():
    print(k, v)
" "$REPRO_ROOT/setup-result.json" >"$WORKDIR/repro-root-keys.txt"
ERROR_KEY="$(awk '$1=="repro-error-scenario"{print $2}' "$WORKDIR/repro-root-keys.txt")"
CLEAN_KEYS="$(awk '$1!="repro-error-scenario"{print $2}' "$WORKDIR/repro-root-keys.txt")"
[[ -n "$ERROR_KEY" ]] || fail "could not resolve root_key for repro-error-scenario"
[[ "$(echo "$CLEAN_KEYS" | wc -l)" -eq 3 ]] || fail "expected 3 clean-route root keys, got: $CLEAN_KEYS"

# Every scenario other than the deliberate route-defect one gets a clean,
# real-JSON claim/next/status-advance/release cycle -- P6 (missing_replay)
# and P7 (evaluator_collection_failure) are the only findings a clean route
# should be able to contribute, never a spurious P2.
CLEAN_KEYS_PY="$(python3 -c "import json,sys; print(json.dumps(sys.argv[1].split()))" "$CLEAN_KEYS")"
cat >"$REPRO_ROOT/scratch-template/shark" <<STUB
#!/usr/bin/env bash
set -euo pipefail
python3 - "\$@" <<PY
import json, sys
args = sys.argv[1:]
error_key = "$ERROR_KEY"
clean_keys = set(json.loads('$CLEAN_KEYS_PY'))
if args[:2] == ["next", error_key]:
    print(json.dumps({"entity_key": error_key, "status": "research", "action": "error", "error": "terminal route failure"}))
elif len(args) >= 2 and args[0] == "next" and args[1] in clean_keys:
    key = args[1]
    print(json.dumps({"entity_key": key, "status": "research", "action": "spawn_agent", "agent_type": "researcher", "provider": "anthropic", "model": "haiku", "effort": "low", "error": ""}))
elif len(args) >= 2 and args[0] == "claim" and args[1] in clean_keys:
    print(json.dumps({"session_id": "sess-" + args[1]}))
elif len(args) >= 3 and args[0] == "status" and args[1] == "advance" and args[2] in clean_keys:
    print(json.dumps({"advanced": True}))
elif len(args) >= 2 and args[0] == "release" and args[1] in clean_keys:
    print(json.dumps({"released": True}))
else:
    sys.stderr.write("stub shark: unexpected argv: " + repr(args) + "\\n")
    sys.exit(2)
PY
STUB
chmod +x "$REPRO_ROOT/scratch-template/shark"
REPRO_DIGEST="$("$DIGEST_PATH_BIN" "$REPRO_ROOT/scratch-template/shark")"
python3 - "$REPRO_ROOT/setup-result.json" "$REPRO_DIGEST" <<'PY'
import json
import sys

path, digest = sys.argv[1:3]
with open(path, encoding="utf-8") as f:
    result = json.load(f)
result["shark_binary"]["sha256"] = digest
with open(path, "w", encoding="utf-8") as f:
    json.dump(result, f)
PY

set +e
"$OPERATOR" preflight --config "$REPRO_ROOT/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/repro-preflight.out" 2>"$WORKDIR/repro-preflight.err"
repro_rc=$?
set -e
[[ "$repro_rc" -eq 1 ]] \
	|| fail "AC-F11-12 reproduction exited $repro_rc, want 1 (blocked): $(cat "$WORKDIR/repro-preflight.err")"

REPRO_RESULT="$(find "$REPRO_ROOT/preflight" -name "preflight-result.json" -print -quit)"
[[ -n "$REPRO_RESULT" ]] || fail "AC-F11-12 reproduction did not write preflight-result.json"

python3 - "$REPRO_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
if result["status"] != "blocked":
    raise SystemExit(f"expected status=blocked, got {result['status']!r}")
if result["provider_ready"] is not False:
    raise SystemExit("expected provider_ready == false")

blockers = result["blockers"]
causes = {b["cause"] for b in blockers}
expected = {"missing_replay", "evaluator_collection_failure", "route_defect", "provider_not_ready"}
if causes != expected:
    raise SystemExit(f"expected exactly the four 2026-09-04 causes {sorted(expected)}, got {sorted(causes)}: {blockers!r}")

evaluator_failures = [b for b in blockers if b["cause"] == "evaluator_collection_failure"]
if len(evaluator_failures) != 2:
    raise SystemExit(f"expected exactly 2 evaluator_collection_failure blockers, got {len(evaluator_failures)}: {evaluator_failures!r}")
evaluator_scenarios = {b["scenario_id"] for b in evaluator_failures}
if evaluator_scenarios != {"repro-leak-py-bug-due-date-boundary", "repro-leak-py-change-priority-scale"}:
    raise SystemExit(f"evaluator_collection_failure did not name both leak scenarios: {evaluator_scenarios!r}")

replay_blockers = [b for b in blockers if b["cause"] == "missing_replay"]
if len(replay_blockers) != 1 or replay_blockers[0]["scenario_id"] != "py-feature-no-replay":
    raise SystemExit(f"expected exactly 1 missing_replay blocker naming py-feature-no-replay, got {replay_blockers!r}")

route_defect_blockers = [b for b in blockers if b["cause"] == "route_defect"]
if len(route_defect_blockers) != 1 or route_defect_blockers[0]["scenario_id"] != "repro-error-scenario":
    raise SystemExit(f"expected exactly 1 route_defect blocker naming repro-error-scenario, got {route_defect_blockers!r}")

# Negative half of row 12 (§7.1 row 12: "each of the 4 causes removed one
# at a time -> blocker count drops to 3"): a synthetic removal of any one
# named cause leaves exactly 3 distinct causes among the rest.
for removed in sorted(expected):
    remaining = {c for c in causes if c != removed}
    if len(remaining) != 3:
        raise SystemExit(f"removing cause {removed!r} did not leave exactly 3 distinct causes: {remaining!r}")
print("AC-F11-12 reproduction OK: 4 distinct causes, provider_ready=false")
PY
echo "TC-103(AC-F11-12: 2026-09-04 conditions reproduced -- 4 distinct blocker causes (missing_replay, evaluator_collection_failure x2, route_defect, provider_not_ready), provider_ready=false) PASS"

echo "TC-103: pass (fail-closed preflight decision table: positive control, P1-P5 falsified alone, status vocabulary closed, 2026-09-04 four-cause reproduction)"
