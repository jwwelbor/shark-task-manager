#!/usr/bin/env bash
# TC-105 / T-E40-F11-007 (spec.md REQ-F-005, AC-F11-13/14/15; test-plan.md
# TC-13/14/15): route resolution (`run-lifecycle.sh --mode resolve-route`)
# is separated from live stage evidence.
#
# Caller-Path Contract (test-plan.md TC-13/14 row): the production
# entrypoint is `e40-benchmark.sh preflight --config <cfg>` -- the operator
# seam. This file never invokes `run-lifecycle.sh` or
# `run-lifecycle-batch.sh` directly for TC-13/14; `preflight`'s own,
# unmodified `_cmd_preflight_locked` invokes `run-lifecycle-batch.sh
# --mode preview`, which invokes `run-lifecycle.sh --mode resolve-route`
# once per selected scenario -- both real, undriven-around. The only stub
# is the `shark` binary itself (same precedent as tc061/tc079/tc094):
# `preflight_environment()`'s `trusted_shark_identity()` binds `SHARK_BIN`
# to the exact path/digest `operator/setup-result.json` records, so this
# file builds that scaffold with a real `e40-benchmark.sh setup` run, then
# substitutes only the `shark` binary underneath it (recomputing the
# recorded digest to match) -- provider/network-adjacent behavior is
# untouched; a stubbed `shark` is not a denied provider/network binary.
#
# TC-15's caller path is different: `verify-stage-evidence.sh <bundle_dir>`
# is itself a standalone, operator-invocable verifier (test-plan.md row
# TC-15), driven directly.
#
# Forbidden mocks (test-plan.md): do not mock the route resolver; do not
# invoke run-lifecycle.sh directly; do not mock verify-stage-evidence.sh's
# schema check.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
DIGEST_PATH_BIN="$SCRIPTS_DIR/lib/digest_path"

fail() {
	echo "TC-105 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh missing or not executable"
[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable"
[[ -x "$DIGEST_PATH_BIN" ]] || fail "lib/digest_path missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# TC-15 (AC-F11-15): verify-stage-evidence.sh accepts live evidence, and
# rejects any record -- bundle-level or per-snapshot -- carrying
# evidence_mode: "route_resolution_only".
# ===========================================================================
echo "TC-105: TC-15 -- verify-stage-evidence.sh live-vs-route_resolution_only partition"

LEDGER_FIXTURE="$SCRIPTS_DIR/testdata/evidence/ledger/valid-disjoint"
[[ -d "$LEDGER_FIXTURE" ]] || fail "shared live-evidence fixture missing: $LEDGER_FIXTURE"

set +e
"$GUARD" "$LEDGER_FIXTURE" >"$WORKDIR/tc15-positive.out" 2>"$WORKDIR/tc15-positive.err"
positive_rc=$?
set -e
[[ "$positive_rc" -eq 0 ]] \
	|| fail "live evidence (no evidence_mode marker) was rejected, want accepted: $(cat "$WORKDIR/tc15-positive.err")"
grep -q '"result": "accepted"' "$WORKDIR/tc15-positive.out" \
	|| fail "live evidence run did not report result=accepted: $(cat "$WORKDIR/tc15-positive.out")"
echo "TC-105(TC-15 positive control: live stage evidence accepted) PASS"

# Negative (bundle-level): bundle.json itself carries the marker.
mkdir -p "$WORKDIR/route-bundle-level"
cat >"$WORKDIR/route-bundle-level/bundle.json" <<'JSON'
{"evidence_mode": "route_resolution_only", "schema_version": "1.0"}
JSON
set +e
"$GUARD" "$WORKDIR/route-bundle-level" >"$WORKDIR/tc15-bundle.out" 2>"$WORKDIR/tc15-bundle.err"
bundle_rc=$?
set -e
[[ "$bundle_rc" -eq 1 ]] \
	|| fail "bundle-level route_resolution_only record was not rejected with exit 1 (got $bundle_rc): $(cat "$WORKDIR/tc15-bundle.err")"
grep -q "route_resolution_evidence_mode" "$WORKDIR/tc15-bundle.err" \
	|| fail "bundle-level rejection did not name route_resolution_evidence_mode: $(cat "$WORKDIR/tc15-bundle.err")"
grep -q "route_resolution_only" "$WORKDIR/tc15-bundle.err" \
	|| fail "bundle-level rejection did not name evidence_mode route_resolution_only: $(cat "$WORKDIR/tc15-bundle.err")"
[[ ! -s "$WORKDIR/tc15-bundle.out" ]] \
	|| fail "bundle-level rejection printed a success document to stdout: $(cat "$WORKDIR/tc15-bundle.out")"
echo "TC-105(TC-15 negative: bundle.json evidence_mode -> rejected) PASS"

# Negative (per-snapshot): bundle.json is otherwise ordinary, but the ONE
# indexed stage snapshot itself carries the marker -- proving AC-F11-15's
# "every record" is checked, not only the bundle's own top-level field.
mkdir -p "$WORKDIR/route-snapshot-level/stages"
cat >"$WORKDIR/route-snapshot-level/bundle.json" <<'JSON'
{
  "schema_version": "1.0",
  "stages": [
    {
      "dispatch_ordinal": 1,
      "stage_key": "research",
      "snapshot_path": "stages/1-research.json",
      "snapshot_digest": "sha256:tc105-route-snapshot"
    }
  ]
}
JSON
cat >"$WORKDIR/route-snapshot-level/stages/1-research.json" <<'JSON'
{
  "evidence_mode": "route_resolution_only",
  "dispatch_ordinal": 1,
  "stage_key": "research",
  "time_ledger": {"stage_start": 0, "stage_end": 1, "reconciliation_epsilon_ns": 1, "intervals": {}}
}
JSON
set +e
"$GUARD" "$WORKDIR/route-snapshot-level" >"$WORKDIR/tc15-snapshot.out" 2>"$WORKDIR/tc15-snapshot.err"
snapshot_rc=$?
set -e
[[ "$snapshot_rc" -eq 1 ]] \
	|| fail "per-snapshot route_resolution_only record was not rejected with exit 1 (got $snapshot_rc): $(cat "$WORKDIR/tc15-snapshot.err")"
grep -q "route_resolution_evidence_mode" "$WORKDIR/tc15-snapshot.err" \
	|| fail "per-snapshot rejection did not name route_resolution_evidence_mode: $(cat "$WORKDIR/tc15-snapshot.err")"
echo "TC-105(TC-15 negative: per-stage-snapshot evidence_mode -> rejected) PASS"

echo "TC-105: TC-15 -- live-vs-route_resolution_only acceptance partition -- PASS"

# ===========================================================================
# TC-13/14 shared scaffold: a real `e40-benchmark.sh setup`, with the
# recorded scratch Shark binary substituted for a deterministic stub so
# `next`/`claim`/`status advance`/`release` responses are controlled without
# mocking the route resolver or the operator seam itself.
# ===========================================================================
OPERATOR_ROOT="$WORKDIR/operator"
"$OPERATOR" setup --out "$OPERATOR_ROOT" >"$WORKDIR/setup.out" 2>"$WORKDIR/setup.err" \
	|| fail "real 'e40-benchmark.sh setup' failed: $(cat "$WORKDIR/setup.err")"

STUB_CALLS="$WORKDIR/stub-calls.log"
: >"$STUB_CALLS"
cat >"$OPERATOR_ROOT/scratch-template/shark" <<STUB
#!/usr/bin/env bash
set -euo pipefail
echo "\$*" >>"$STUB_CALLS"
python3 - "\$@" <<'PY'
import json, sys
args = sys.argv[1:]

# B001 (bug, py-bug-due-date-boundary): clean route, no defect, no deferred
# artifact -- TC-13's positive per-stage schema-completeness case.
if args[:2] == ["next", "B001"]:
    print(json.dumps({"entity_key": "B001", "status": "research", "action": "spawn_agent", "agent_type": "researcher", "provider": "anthropic", "model": "haiku", "effort": "low", "error": ""}))
elif args[:2] == ["claim", "B001"]:
    print(json.dumps({"session_id": "sess-b001"}))
elif args[:2] == ["status", "advance"] and args[2] == "B001":
    print(json.dumps({"advanced": True}))
elif args[:2] == ["release", "B001"]:
    print(json.dumps({"released": True}))

# CC-001 (change_card, py-change-priority-scale): a real workflow-routing
# failure at the "status advance" seam -- TC-14's route_defect case (AC-F11-14:
# "an outcome target naming an undefined step, a missing dispatch resolution").
elif args[:2] == ["next", "CC-001"]:
    print(json.dumps({"entity_key": "CC-001", "status": "research", "action": "spawn_agent", "agent_type": "researcher", "provider": "anthropic", "model": "haiku", "effort": "", "error": ""}))
elif args[:2] == ["claim", "CC-001"]:
    print(json.dumps({"session_id": "sess-cc001"}))
elif args[:2] == ["status", "advance"] and args[2] == "CC-001":
    sys.stderr.write("Error: outcome 'pass' has no configured target for this step\\n")
    sys.exit(1)
elif args[:2] == ["release", "CC-001"]:
    print(json.dumps({"released": True}))

# E01-F01 (feature, py-feature-recurring-tasks): also clean -- nothing here
# is dispatched twice per requested key (see run-lifecycle.sh resolve_route's
# one-shot-per-key convention).
elif args[:2] == ["next", "E01-F01"]:
    print(json.dumps({"entity_key": "E01-F01", "status": "assessment", "action": "spawn_agent", "agent_type": "researcher", "provider": "anthropic", "model": "haiku", "effort": "low", "error": ""}))
elif args[:2] == ["claim", "E01-F01"]:
    print(json.dumps({"session_id": "sess-e01f01"}))
elif args[:2] == ["status", "advance"] and args[2] == "E01-F01":
    print(json.dumps({"advanced": True}))
elif args[:2] == ["release", "E01-F01"]:
    print(json.dumps({"released": True}))

# TD-001 (tech_debt): exits 0 but prints human text, not JSON -- the real,
# pre-existing `shark release` defect this test discovered (it ignores
# --json unconditionally) reproduced deliberately. AC-F11-14 requires this
# to be told apart from a genuine route defect: a caller-path/output-
# contract failure is not "an outcome target naming an undefined step, a
# missing dispatch resolution" and must never be reported as such.
elif args[:2] == ["next", "TD-001"]:
    print(json.dumps({"entity_key": "TD-001", "status": "research", "action": "spawn_agent", "agent_type": "researcher", "provider": "anthropic", "model": "haiku", "effort": "", "error": ""}))
elif args[:2] == ["claim", "TD-001"]:
    print(json.dumps({"session_id": "sess-td001"}))
elif args[:2] == ["status", "advance"] and args[2] == "TD-001":
    print(json.dumps({"advanced": True}))
elif args[:2] == ["release", "TD-001"]:
    print("SUCCESS  Released claim on tech_debt TD-001")
else:
    sys.stderr.write("stub shark: unexpected argv: " + repr(args) + "\\n")
    sys.exit(2)
PY
STUB
chmod +x "$OPERATOR_ROOT/scratch-template/shark"

STUB_DIGEST="$("$DIGEST_PATH_BIN" "$OPERATOR_ROOT/scratch-template/shark")"
python3 - "$OPERATOR_ROOT/setup-result.json" "$STUB_DIGEST" <<'PY'
import json, sys
path, digest = sys.argv[1:3]
with open(path) as f:
    result = json.load(f)
result["shark_binary"]["sha256"] = digest
with open(path, "w") as f:
    json.dump(result, f)
PY

# ===========================================================================
# TC-13/TC-14 (route_defect half): drive `preflight` for real over the
# stubbed scratch template.
# ===========================================================================
echo "TC-105: TC-13/14 -- preflight over a stubbed shark (resolved + route_defect)"

set +e
"$OPERATOR" preflight --config "$OPERATOR_ROOT/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/preflight-a.out" 2>"$WORKDIR/preflight-a.err"
preflight_a_rc=$?
set -e
# T-E40-F11-010 (AC-F11-09): this scaffold configures no runtime adapter/
# i05_bundle_dir and includes a route_defect scenario, so the fail-closed
# gate reports status=="blocked" (exit 1), not 0 -- what this test actually
# needs is that the INNER preview batch driver itself ran to completion and
# produced a ledger, which `preview_exit_code` states directly.
[[ "$preflight_a_rc" -eq 1 ]] \
	|| fail "preflight (route_defect scaffold) exited $preflight_a_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-a.err")"
PREFLIGHT_A_RESULT="$(find "$OPERATOR_ROOT/preflight" -name "preflight-result.json" -print -quit)"
[[ -n "$PREFLIGHT_A_RESULT" ]] || fail "preflight (route_defect scaffold) did not write preflight-result.json"
python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if d['preview_exit_code']==0 else 1)" "$PREFLIGHT_A_RESULT" \
	|| fail "preflight (route_defect scaffold) inner preview batch driver did not exit 0"

LEDGER_A=$(find "$OPERATOR_ROOT/preflight" -name "resolution-ledger.jsonl" -print -quit)
[[ -n "$LEDGER_A" && -f "$LEDGER_A" ]] \
	|| fail "preflight did not produce resolution-ledger.jsonl under its retention-preview root"

python3 - "$LEDGER_A" <<'PY'
import json
import sys

path = sys.argv[1]
records = {}
with open(path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        record = json.loads(line)
        records[record["scenario_id"]] = record

REQUIRED_STAGE_FIELDS = ("stage_id", "route", "provider", "model", "effort", "terminal", "artifact_dependencies_deferred")


def schema_complete(stage):
    """AC-F11-13's own 7-field schema -- the property this test proves is
    exhaustive over exactly these 7 keys, both for the real production
    record and for each single-field-dropped mutation below."""
    return all(field in stage for field in REQUIRED_STAGE_FIELDS)


bug = records.get("py-bug-due-date-boundary")
if bug is None:
    raise SystemExit("resolution-ledger.jsonl has no record for py-bug-due-date-boundary")
if bug["resolution"] != "resolved" or bug["cause_class"] is not None:
    raise SystemExit(f"expected py-bug-due-date-boundary resolved/null, got {bug['resolution']!r}/{bug['cause_class']!r}")
if bug["dispatch_count"] != 1 or bug["stage_count"] != 1:
    raise SystemExit(f"expected exactly 1 dispatch/1 stage for py-bug-due-date-boundary, got {bug!r}")
stage = bug["stages"][0]
if not schema_complete(stage):
    missing = [f for f in REQUIRED_STAGE_FIELDS if f not in stage]
    raise SystemExit(f"TC-13: real resolve-route stage record is missing required field(s): {missing}")
if stage["route"] != "pass":
    raise SystemExit(f"expected the traced route to be the configured success route (pass), got {stage['route']!r}")
if stage["artifact_dependencies_deferred"] is not True:
    raise SystemExit("artifact_dependencies_deferred must be true on every resolve-route stage")

# TC-13 negative: each of the 7 required fields absent in turn -> the same
# schema completeness check this test just proved true of the real record
# now correctly reports it incomplete. Never invents a field name outside
# AC-F11-13's own enumerated 7.
for field in REQUIRED_STAGE_FIELDS:
    mutated = dict(stage)
    del mutated[field]
    if schema_complete(mutated):
        raise SystemExit(f"TC-13 negative: dropping required field {field!r} was not detected as schema-incomplete")
if not schema_complete(stage):
    raise SystemExit("TC-13: the original (unmutated) record must remain schema-complete")

change_card = records.get("py-change-priority-scale")
if change_card is None:
    raise SystemExit("resolution-ledger.jsonl has no record for py-change-priority-scale")
if change_card["resolution"] != "failed" or change_card["cause_class"] != "route_defect":
    raise SystemExit(f"TC-14: expected py-change-priority-scale failed/route_defect, got {change_card['resolution']!r}/{change_card['cause_class']!r}")
if "outcome" in change_card and change_card["cause_class"] == "artifact_deferred":
    raise SystemExit("TC-14: route_defect record must never also read as artifact_deferred")

# TC-14 residual class: a `shark` subcommand that exits 0 but prints text
# `run-lifecycle.sh` cannot parse as JSON (TD-001's stub: the real,
# pre-existing `shark release --json` defect this task discovered,
# reproduced deliberately) is a caller-path/output-contract failure, never
# "an outcome target naming an undefined step" -- AC-F11-14's own route_defect
# definition. resolve_route must not mislabel it as route_defect (nor as
# artifact_deferred, which is reserved for a missing live-only artifact).
tech_debt = records.get("py-techdebt-consolidate-validation")
if tech_debt is None:
    raise SystemExit("resolution-ledger.jsonl has no record for py-techdebt-consolidate-validation")
if tech_debt["resolution"] != "failed":
    raise SystemExit(f"TC-14: a non-JSON CLI response must still fail resolution, got {tech_debt['resolution']!r}")
if tech_debt["cause_class"] is not None:
    raise SystemExit(f"TC-14: a caller-path/output-contract failure must not be labelled {tech_debt['cause_class']!r} (route_defect is reserved for a real routing failure)")

print("TC-105: TC-13/14 (route_defect half) ledger assertions OK")
PY

echo "TC-105(TC-13: real resolve-route stage record carries all 7 required fields; each field's absence is independently detected) PASS"
echo "TC-105(TC-14 case A: a real dispatch-seam failure classifies as resolution=failed, cause_class=route_defect) PASS"
echo "TC-105(TC-14 residual class: a caller-path/output-contract failure (exit 0, non-JSON) classifies as resolution=failed, cause_class=null -- never route_defect) PASS"

# ===========================================================================
# TC-14 (artifact_deferred half): a feature-family scenario whose replay
# bundle is absent resolves the route successfully -- AC-F11-14: "does not
# fail route resolution" -- and is never conflated with route_defect.
#
# Reuses the real, admitted py-feature-recurring-tasks package verbatim
# (fixture/toolchain/admission block unchanged, so I-04 admission and
# fixture resolution stay real) under a new scenario_id, with only its
# replay bundle file removed from the copy -- the real committed package is
# never touched.
# ===========================================================================
echo "TC-105: TC-14 (artifact_deferred half) -- feature scenario with an absent replay bundle"

REAL_FEATURE_PACKAGE="$SCRIPTS_DIR/../scenarios/packages/py-feature-recurring-tasks"
[[ -d "$REAL_FEATURE_PACKAGE" ]] || fail "real feature fixture package missing: $REAL_FEATURE_PACKAGE"

mkdir -p "$WORKDIR/scenarios/packages"
cp -a "$REAL_FEATURE_PACKAGE" "$WORKDIR/scenarios/packages/py-feature-deferred"
python3 - "$WORKDIR/scenarios/packages/py-feature-deferred/package.yaml" <<'PY'
import sys
path = sys.argv[1]
text = open(path).read()
needle = 'scenario_id: "py-feature-recurring-tasks"'
if needle not in text:
    raise SystemExit(f"expected {needle!r} in copied package.yaml")
open(path, "w").write(text.replace(needle, 'scenario_id: "py-feature-deferred"'))
PY
REPLAY_BUNDLE="$WORKDIR/scenarios/packages/py-feature-deferred/evaluator/replay/reference-bundle.json"
[[ -f "$REPLAY_BUNDLE" ]] || fail "copied feature package has no replay bundle to remove: $REPLAY_BUNDLE"
rm -f "$REPLAY_BUNDLE"

REAL_SCENARIO_INDEX="$SCRIPTS_DIR/../scenarios/scenarios.yaml"
python3 - "$REAL_SCENARIO_INDEX" "$WORKDIR/scenarios/scenarios.yaml" <<'PY'
import sys
import yaml

real_index_path, out_path = sys.argv[1:3]
with open(real_index_path) as f:
    real_index = yaml.safe_load(f)
custom_index = {
    "schema_version": real_index["schema_version"],
    "fixtures": real_index["fixtures"],
    "adapters": real_index["adapters"],
    "scenarios": ["packages/py-feature-deferred"],
}
with open(out_path, "w") as f:
    yaml.safe_dump(custom_index, f)
PY

DEFERRED_ROOT="$WORKDIR/operator-deferred"
mkdir -p "$DEFERRED_ROOT"
cp -a "$OPERATOR_ROOT/scratch-template" "$DEFERRED_ROOT/scratch-template"
python3 - "$OPERATOR_ROOT" "$DEFERRED_ROOT" "$WORKDIR/scenarios/scenarios.yaml" <<'PY'
import json
import sys
import yaml

src_root, dst_root, scenario_index = sys.argv[1:4]

with open(f"{src_root}/e40-demo.yaml") as f:
    config = yaml.safe_load(f)
config["operator_root"] = dst_root
config["run_store"] = f"{dst_root}/runs"
config["comparison_store"] = f"{dst_root}/comparisons"
config["scenario_index"] = scenario_index
config["scratch_template"] = f"{dst_root}/scratch-template"
config["shark_binary"] = f"{dst_root}/scratch-template/shark"
config["content_root"] = f"{dst_root}/scratch-template/shark-data"
config["prompt_root"] = f"{dst_root}/scratch-template/shark-data/prompts"
config["workflow_root"] = f"{dst_root}/scratch-template/shark-data/workflow"
config["policy_root"] = f"{dst_root}/scratch-template/shark-data/workflow"
config["scenario_roots"] = {
    "py-feature-deferred": {
        "root_key": "E01-F01",
        "scratch_root": f"{dst_root}/scratch-template",
        "i05_bundle_dir": None,
    }
}
with open(f"{dst_root}/e40-demo.yaml", "w") as f:
    yaml.safe_dump(config, f)

with open(f"{src_root}/setup-result.json") as f:
    result = json.load(f)
result["operator_root"] = dst_root
result["config"] = f"{dst_root}/e40-demo.yaml"
result["scratch_root"] = f"{dst_root}/scratch-template"
with open(f"{dst_root}/setup-result.json", "w") as f:
    json.dump(result, f)
PY

set +e
"$OPERATOR" preflight --config "$DEFERRED_ROOT/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/preflight-b.out" 2>"$WORKDIR/preflight-b.err"
preflight_b_rc=$?
set -e
# T-E40-F11-010 (AC-F11-09): P4/P5 are unconfigured here too, so the
# fail-closed gate reports status=="blocked" (exit 1) even though route
# resolution itself succeeds -- the property this test needs is that the
# inner preview batch driver ran to completion.
[[ "$preflight_b_rc" -eq 1 ]] \
	|| fail "preflight (artifact_deferred scaffold) exited $preflight_b_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-b.err")"
PREFLIGHT_B_RESULT="$(find "$DEFERRED_ROOT/preflight" -name "preflight-result.json" -print -quit)"
[[ -n "$PREFLIGHT_B_RESULT" ]] || fail "preflight (artifact_deferred scaffold) did not write preflight-result.json"
python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if d['preview_exit_code']==0 else 1)" "$PREFLIGHT_B_RESULT" \
	|| fail "preflight (artifact_deferred scaffold) inner preview batch driver did not exit 0"

LEDGER_B=$(find "$DEFERRED_ROOT/preflight" -name "resolution-ledger.jsonl" -print -quit)
[[ -n "$LEDGER_B" && -f "$LEDGER_B" ]] \
	|| fail "preflight (artifact_deferred scaffold) did not produce resolution-ledger.jsonl"

python3 - "$LEDGER_B" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    records = [json.loads(line) for line in f if line.strip()]
if len(records) != 1:
    raise SystemExit(f"expected exactly one ledger record, got {len(records)}: {records!r}")
record = records[0]
if record["scenario_id"] != "py-feature-deferred":
    raise SystemExit(f"unexpected scenario_id: {record['scenario_id']!r}")

# AC-F11-14: an absent live-only artifact is reported as cause_class ==
# "artifact_deferred" and does NOT fail route resolution -- the two
# classifications (this one and route_defect) must never be conflated in
# either direction.
if record["resolution"] != "resolved":
    raise SystemExit(f"an absent replay bundle must not fail route resolution; got resolution={record['resolution']!r}")
if record["cause_class"] != "artifact_deferred":
    raise SystemExit(f"expected cause_class=artifact_deferred, got {record['cause_class']!r}")
if "replay bundle is missing" not in record["reason"]:
    raise SystemExit(f"reason does not name the missing replay bundle: {record['reason']!r}")
if record["dispatch_count"] < 1 or not record["stages"]:
    raise SystemExit("artifact_deferred record unexpectedly traced zero stages")

print("TC-105: TC-14 (artifact_deferred half) ledger assertions OK")
PY

echo "TC-105(TC-14 case B: a feature scenario with an absent replay bundle resolves as resolution=resolved, cause_class=artifact_deferred, never route_defect) PASS"
echo "TC-105(TC-14: route_defect, artifact_deferred, and the residual caller-path/output-contract class proven never conflated with one another) PASS"

# ===========================================================================
# TC-14 real-binary regression (code-review-20260906-210407-E40-F11 F1):
# everything above drives resolve_route() against a STUBBED shark (same
# precedent as tc061/tc079/tc094), which cannot exercise
# replay_artifact_deferred_reason()'s generalized half at all -- a stub
# never raises the real "cannot advance ... from research: research
# report: ..." error entity_service.go raises for a real bug/change_card/
# tech_debt scenario's real research step. Drive `preflight` for real, over
# a real `setup`-built `shark` binary (never substituted), for the three
# families whose first workable step is "research" (bug, change_card,
# tech_debt -- feature/epic's first workable step is "assessment", so
# resolve_route's one-hop-per-key tracing never reaches their research
# step; out of this task's scope, see run-lifecycle.sh's
# dispatch_failure_deferred_reason() comment). Before the fix
# (T-E40-F11-007 rework), all three misclassified as
# resolution=failed/cause_class=route_defect; after, all three must
# resolve as resolution=resolved/cause_class=artifact_deferred.
# ===========================================================================
echo "TC-105: TC-14 real-binary regression -- bug/change_card/tech_debt research-report deferral against the real shark binary"

REAL_INDEX_ROOT="$SCRIPTS_DIR/.."
REAL_INDEX_DIR="$WORKDIR/real-binary-index"
mkdir -p "$REAL_INDEX_DIR"
cat >"$REAL_INDEX_DIR/scenarios.yaml" <<YAML
schema_version: "1.1"
fixtures:
  py:
    submodule_path: $REAL_INDEX_ROOT/fixture-py
adapters:
  python:
    path: $REAL_INDEX_ROOT/adapters/python
    version: "1.0.0"
scenarios:
  - $REAL_INDEX_ROOT/scenarios/packages/py-bug-due-date-boundary
  - $REAL_INDEX_ROOT/scenarios/packages/py-change-priority-scale
  - $REAL_INDEX_ROOT/scenarios/packages/py-techdebt-consolidate-validation
YAML

REAL_BINARY_ROOT="$WORKDIR/operator-real-binary"
"$OPERATOR" setup --out "$REAL_BINARY_ROOT" --scenario-index "$REAL_INDEX_DIR/scenarios.yaml" \
	>"$WORKDIR/real-binary-setup.out" 2>"$WORKDIR/real-binary-setup.err" \
	|| fail "real 'e40-benchmark.sh setup' (real-binary regression) failed: $(cat "$WORKDIR/real-binary-setup.err")"
# Unlike the TC-13/14 stubbed scaffold above, the scratch `shark` under
# $REAL_BINARY_ROOT is never substituted -- `setup` copies the real
# bench-built `bin/shark` verbatim (cmd_setup, e40_benchmark.py) -- so this
# section proves the fix against production shark, not a stand-in.

set +e
"$OPERATOR" preflight --config "$REAL_BINARY_ROOT/e40-demo.yaml" --reps 1 \
	>"$WORKDIR/preflight-real.out" 2>"$WORKDIR/preflight-real.err"
preflight_real_rc=$?
set -e
# Same rationale as steps A/B above: no runtime adapter/i05_bundle_dir is
# configured, so the fail-closed gate reports status=="blocked" (exit 1);
# what this section needs is that the inner preview batch driver itself
# ran to completion and produced a resolution ledger.
[[ "$preflight_real_rc" -eq 1 ]] \
	|| fail "preflight (real-binary regression) exited $preflight_real_rc, want 1 (blocked): $(cat "$WORKDIR/preflight-real.err")"
PREFLIGHT_REAL_RESULT="$(find "$REAL_BINARY_ROOT/preflight" -name "preflight-result.json" -print -quit)"
[[ -n "$PREFLIGHT_REAL_RESULT" ]] || fail "preflight (real-binary regression) did not write preflight-result.json"
python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if d['preview_exit_code']==0 else 1)" "$PREFLIGHT_REAL_RESULT" \
	|| fail "preflight (real-binary regression) inner preview batch driver did not exit 0"

LEDGER_REAL=$(find "$REAL_BINARY_ROOT/preflight" -name "resolution-ledger.jsonl" -print -quit)
[[ -n "$LEDGER_REAL" && -f "$LEDGER_REAL" ]] \
	|| fail "preflight (real-binary regression) did not produce resolution-ledger.jsonl"

python3 - "$LEDGER_REAL" <<'PY'
import json
import sys

path = sys.argv[1]
records = {}
with open(path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        record = json.loads(line)
        records[record["scenario_id"]] = record

# AC-F11-14, generalized (code-review F1): a real research-report.md
# absence at ANY family's research step must classify as
# resolution=resolved/cause_class=artifact_deferred, never
# resolution=failed/cause_class=route_defect -- proven here against the
# real shark binary, not a stub, for all three families the review found
# misclassified.
for scenario_id in ("py-bug-due-date-boundary", "py-change-priority-scale", "py-techdebt-consolidate-validation"):
    record = records.get(scenario_id)
    if record is None:
        raise SystemExit(f"resolution-ledger.jsonl has no record for {scenario_id}")
    if record["resolution"] != "resolved":
        raise SystemExit(f"{scenario_id}: expected resolution=resolved against the real shark binary, got {record['resolution']!r} (cause_class={record['cause_class']!r}, reason={record['reason']!r})")
    if record["cause_class"] != "artifact_deferred":
        raise SystemExit(f"{scenario_id}: expected cause_class=artifact_deferred, got {record['cause_class']!r}")
    if "research report:" not in record["reason"]:
        raise SystemExit(f"{scenario_id}: reason does not name the missing research report: {record['reason']!r}")
    if record["dispatch_count"] < 1 or not record["stages"]:
        raise SystemExit(f"{scenario_id}: artifact_deferred record unexpectedly traced zero stages")

print("TC-105: TC-14 real-binary regression ledger assertions OK (bug/change_card/tech_debt all artifact_deferred, none route_defect)")
PY

echo "TC-105(TC-14 real-binary regression: bug/change_card/tech_debt each resolve as resolution=resolved/cause_class=artifact_deferred against the real shark binary, never route_defect) PASS"

echo "TC-105: pass (route resolution schema completeness; route_defect vs artifact_deferred vs caller-path/output-contract failure never conflated; route_resolution_only never accepted as I-05 stage evidence; artifact-deferral generalized across families and proven against the real shark binary)"
