#!/usr/bin/env bash
# TC-112 / T-E40-F11-013: generalize pilot attestation, aggregation, and
# reports for six families (spec.md REQ-F-013; test-plan.md TC-42..44 full
# body; AC-F11-42, AC-F11-43, AC-F11-44).
#
# Two independent parts, matching this task's own file-scope split:
#
#   PART A (AC-F11-42, bench/scripts/lib/spend-gate.sh -- "no change
#   expected; add the tc112 assertion for six-family derivation"): the
#   pilot-ledger attestation gate's family set is derived, at runtime, from
#   the REAL admitted scenario index (bench/scenarios/scenarios.yaml) and
#   the REAL package.yaml files it points at -- never a private/hardcoded
#   family list. Reached through the real `run-lifecycle-batch.sh` CLI
#   (which sources the real lib/spend-gate.sh), the same Caller-Path
#   Contract tc081_pilot_ledger_gate_test.sh already established for this
#   exact mechanism (T-E40-F10-006), now driven by the real six-family
#   registry instead of a synthetic one.
#
#   PART B (AC-F11-43/AC-F11-44, bench/scripts/report-lifecycle.sh /
#   aggregate-lifecycle.sh): reached ONLY through `e40-benchmark.sh
#   baseline` (test-plan.md TC-42..44 Caller-Path Contract: "Do not mock
#   report-lifecycle.sh or aggregate-lifecycle.sh -- there is no `report`
#   subcommand; both are reached only through `baseline`"). The lowest mock
#   seam is the provider-backed batch driver itself
#   (`run-lifecycle-batch.sh`) -- intercepted the same way
#   tc094_e40_benchmark_operator_test.sh's own `fixture_run_process`
#   intercepts it, except this fixture ALSO populates real
#   scenarios/<id>/<rep>/{package.yaml,lifecycle.jsonl,evaluation.jsonl,
#   manifest.json} retained content, so the operator's own
#   `aggregate_and_report()` step invokes the REAL aggregate-lifecycle.sh
#   and the REAL report-lifecycle.sh (both views) against it, unmocked.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
BENCH_DIR="$SCRIPTS_DIR/.."
LIB="$SCRIPTS_DIR/lib/spend-gate.sh"
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
PILOT_LEDGER="$SCRIPTS_DIR/pilot-ledger.sh"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
E40_BENCHMARK_PY="$SCRIPTS_DIR/lib/e40_benchmark.py"
REAL_SCENARIO_INDEX="$BENCH_DIR/scenarios/scenarios.yaml"

fail() {
	echo "TC-112 FAIL: $1" >&2
	exit 1
}

[[ -f "$LIB" ]] || fail "bench/scripts/lib/spend-gate.sh missing"
[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"
[[ -x "$PILOT_LEDGER" ]] || fail "bench/scripts/pilot-ledger.sh missing or not executable"
[[ -x "$OPERATOR" ]] || fail "bench/scripts/e40-benchmark.sh missing or not executable"
[[ -f "$REAL_SCENARIO_INDEX" ]] || fail "bench/scenarios/scenarios.yaml missing"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ---------------------------------------------------------------------------
# Ground truth: read the REAL scenario_id -> entity_family mapping directly
# from the real admitted registry -- this is the test's own oracle, never a
# literal family list baked into this file. AC-F11-42's own §7.1a closure
# rule: "Closed by asserting set equality between the attestation family set
# and the batch policy's selected set -- never against the literal 6."
# ---------------------------------------------------------------------------
REAL_FAMILIES_JSON="$WORKDIR/real-families.json"
python3 - "$BENCH_DIR" "$REAL_SCENARIO_INDEX" "$REAL_FAMILIES_JSON" <<'PYEOF'
import json
import os
import sys

import yaml

bench_dir, index_path, out_path = sys.argv[1:4]
with open(index_path, encoding="utf-8") as f:
    index = yaml.safe_load(f) or {}
index_dir = os.path.dirname(index_path)

mapping = {}
for rel in index.get("scenarios") or []:
    pkg_path = os.path.join(index_dir, rel, "package.yaml")
    with open(pkg_path, encoding="utf-8") as f:
        pkg = yaml.safe_load(f) or {}
    scenario_id = str(pkg["scenario_id"])
    family = str(pkg["entity_family"])
    mapping[scenario_id] = family

if len(mapping) < 2:
    raise SystemExit(f"real scenario index resolved fewer than 2 scenarios: {mapping}")

with open(out_path, "w", encoding="utf-8") as f:
    json.dump(mapping, f, sort_keys=True)
print(f"TC-112: real registry resolves {len(mapping)} scenario(s) across "
      f"{len(set(mapping.values()))} distinct famil(y/ies): {sorted(set(mapping.values()))}")
PYEOF

REAL_FAMILY_COUNT="$(python3 -c "import json,sys; print(len(set(json.load(open(sys.argv[1])).values())))" "$REAL_FAMILIES_JSON")"
[[ "$REAL_FAMILY_COUNT" -ge 2 ]] || fail "real registry does not carry enough distinct families to exercise per-family sectioning"

# ===========================================================================
# PART A (AC-F11-42): pilot-ledger attestation gate, derived from the real
# registry, reached through the real run-lifecycle-batch.sh CLI.
# ===========================================================================

ROOT_A="$WORKDIR/retention-a"
mkdir -p "$ROOT_A"

cat >"$WORKDIR/batch-policy-a.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$REAL_SCENARIO_INDEX"
EOF

# --- A1: _spend_gate_families_from_batch's own derived set matches the
# ground-truth family set exactly (set equality, never against a literal
# count) -- source the library directly, the same seam T-E40-F10-005
# already names as this file's own mechanism.
# shellcheck source=/dev/null
DERIVED_FAMILIES_A1="$(bash -c 'source "'"$LIB"'"; _spend_gate_families_from_batch "'"$WORKDIR/batch-policy-a.yaml"'" ""')"
python3 - "$REAL_FAMILIES_JSON" <<PYEOF
import json
import sys

mapping = json.load(open(sys.argv[1] if len(sys.argv) > 1 else "$REAL_FAMILIES_JSON"))
expected = sorted(set(mapping.values()))
observed = sorted(l for l in """$DERIVED_FAMILIES_A1""".splitlines() if l)
if observed != expected:
    raise SystemExit(f"TC-112 FAIL: _spend_gate_families_from_batch derived {observed}, expected (from the real registry) {expected}")
print("TC-112 (A1): _spend_gate_families_from_batch derives exactly the real registry's family set -- never a hardcoded literal")
PYEOF

# --- A2: no line in spend-gate.sh names two or more of the real family
# values together -- the structural signature of a hardcoded family list
# (AC-F11-42: "a hardcoded family list anywhere -> the derivation test
# fails"). A single incidental prose mention of one family name is fine;
# co-occurrence of two or more on one line is not.
python3 - "$REAL_FAMILIES_JSON" "$LIB" <<'PYEOF'
import json
import re
import sys

families_path, lib_path = sys.argv[1:3]
families = sorted(set(json.load(open(families_path)).values()))
with open(lib_path, encoding="utf-8") as f:
    lines = f.readlines()

for lineno, line in enumerate(lines, start=1):
    hits = [fam for fam in families if re.search(r"\b%s\b" % re.escape(fam), line)]
    if len(hits) >= 2:
        raise SystemExit(
            f"TC-112 FAIL (A2): {lib_path}:{lineno} names {len(hits)} real family "
            f"values together ({hits}) -- looks like a hardcoded family list: {line!r}"
        )
print("TC-112 (A2): no line in spend-gate.sh co-mentions 2+ real family values -- no hardcoded family list")
PYEOF

# --- Build minimal real-registry-driven retention fixtures for A3-A5 below.
retain_fixture_a() {
	# retain_fixture_a <scenario_id> <family> -- all eight canonical
	# retention artifacts pilot-ledger.sh requires (spec.md Data model),
	# same shape as tc081_pilot_ledger_gate_test.sh's own retain_fixture().
	local scenario_id="$1" family="$2"
	local dest="$ROOT_A/scenarios/$scenario_id/1"
	mkdir -p "$dest/evidence" "$dest/transcripts"
	cat >"$dest/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$scenario_id"
scenario_version: "1"
entity_family: "$family"
EOF
	echo '{"stage": "code", "note": "fixture evidence"}' >"$dest/evidence/stage.json"
	echo "fixture transcript for $scenario_id" >"$dest/transcripts/stage.txt"
	echo '{"root_key": "ROOT-001", "entries": []}' >"$dest/entity-history.json"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/lifecycle.jsonl"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/evaluation.jsonl"
	echo '{"held_back": true}' >"$dest/oracle.json"
	python3 - "$dest" <<'PY'
import hashlib
import json
import os
import sys

dest = sys.argv[1]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


artifacts = {}
for name in ("package.yaml", "evidence", "transcripts", "entity-history.json", "lifecycle.jsonl", "evaluation.jsonl", "oracle.json"):
    artifacts[name] = {
        "source_path": f"/fixture-source/{name}",
        "sha256": digest_of_path(os.path.join(dest, name)),
    }

manifest = {
    "scenario_id": os.path.basename(os.path.dirname(dest)),
    "rep": 1,
    "artifacts": artifacts,
}
with open(os.path.join(dest, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
PY
}

mapfile -t REAL_SCENARIO_IDS < <(python3 -c "import json; m=json.load(open('$REAL_FAMILIES_JSON')); print('\n'.join(sorted(m)))")
mapfile -t REAL_FAMILY_VALUES < <(python3 -c "
import json
m = json.load(open('$REAL_FAMILIES_JSON'))
for sid in sorted(m):
    print(m[sid])
")

CHECKLIST="$WORKDIR/checklist.json"
echo '{"items": [{"id": "structural-spotcheck", "result": "pass"}]}' >"$CHECKLIST"

WITHHELD_SCENARIO="${REAL_SCENARIO_IDS[-1]}"
WITHHELD_FAMILY="${REAL_FAMILY_VALUES[-1]}"

for idx in "${!REAL_SCENARIO_IDS[@]}"; do
	sid="${REAL_SCENARIO_IDS[$idx]}"
	fam="${REAL_FAMILY_VALUES[$idx]}"
	retain_fixture_a "$sid" "$fam"
	if [[ "$sid" != "$WITHHELD_SCENARIO" ]]; then
		"$PILOT_LEDGER" --retention-root "$ROOT_A" --record --scenario "$sid" --rep 1 \
			--operator "operator-tc112@example.com" --checklist "$CHECKLIST" \
			>/dev/null || fail "A: precondition --record for $sid/$fam failed"
	fi
done

GOOD_CEILINGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 --max-provider-calls 15)

# --- A3: with one family (the withheld one) never attested, a whole-batch
# --mode baseline invocation over the ENTIRE real registry (no --scenarios
# filter -- the operator default, which resolves to every real family) must
# refuse as a whole command, pre-dispatch, naming the withheld family and
# the schema-owned missing_pilot_attestation reason. This is the direct
# "5 attestations -> publication refused" observable AC-F11-42/spec.md row
# 42 names, scaled to however many families the real registry carries.
out_a3="$("$BATCH" --batch "$WORKDIR/batch-policy-a.yaml" --retention-root "$ROOT_A" --mode baseline \
	"${GOOD_CEILINGS[@]}" 2>&1)" && rc_a3=0 || rc_a3=$?
[[ "$rc_a3" -eq 3 ]] || fail "A3: expected spend-gate refusal exit 3 for the withheld family, got $rc_a3: $out_a3"
echo "$out_a3" | grep -q "$WITHHELD_FAMILY" || fail "A3: refusal did not name the withheld family '$WITHHELD_FAMILY': $out_a3"
echo "$out_a3" | grep -q "missing_pilot_attestation" || fail "A3: refusal did not carry missing_pilot_attestation: $out_a3"
echo "$out_a3" | grep -q '"classification"' && fail "A3: output contains dispatch/classification evidence -- refusal was not pre-dispatch: $out_a3"
[[ ! -f "$ROOT_A/batch.json" ]] || fail "A3: batch.json was written despite a whole-command refusal"

# --- A4: attesting the withheld family too makes the gate pass for the
# WHOLE real registry -- "attestation count == selected-family count, both
# computed from the policy" (AC-F11-42's own observable). Every scenario
# already has a retained evaluation.jsonl (built above, not via a real
# dispatch), so classify_pair honestly reports skipped_complete once the
# gate is reached, proving the gate was passed rather than bypassed.
"$PILOT_LEDGER" --retention-root "$ROOT_A" --record --scenario "$WITHHELD_SCENARIO" --rep 1 \
	--operator "operator-tc112@example.com" --checklist "$CHECKLIST" \
	>/dev/null || fail "A4: --record for the withheld family failed"

out_a4="$("$BATCH" --batch "$WORKDIR/batch-policy-a.yaml" --retention-root "$ROOT_A" --mode baseline \
	"${GOOD_CEILINGS[@]}" 2>&1)" && rc_a4=0 || rc_a4=$?
[[ "$rc_a4" -eq 0 ]] || fail "A4: expected the gate to pass once every real family is attested, got $rc_a4: $out_a4"
echo "$out_a4" | grep -q '"classification": "skipped_complete"' || fail "A4: expected classification to be reached and report skipped_complete: $out_a4"
[[ -f "$ROOT_A/batch.json" ]] || fail "A4: batch.json was not written despite a gate-passing invocation"

echo "TC-112 (Part A / AC-F11-42): pilot-ledger attestation is required per SELECTED family, derived at runtime from the real admitted registry -- a family withheld from attestation refuses the whole batch pre-dispatch by name, and attesting every real family lets the batch proceed"

# ===========================================================================
# PART B (AC-F11-43 / AC-F11-44): report + aggregate generalization, reached
# only through `e40-benchmark.sh baseline`.
# ===========================================================================

OPROOT="$WORKDIR/operator"
"$OPERATOR" setup --out "$OPROOT" >"$WORKDIR/setup.out" 2>&1 || fail "operator setup failed: $(cat "$WORKDIR/setup.out")"
CONFIG="$OPROOT/e40-demo.yaml"
[[ -f "$CONFIG" ]] || fail "operator setup did not produce e40-demo.yaml"

# --readiness placeholders (never actually invoked -- run-lifecycle-batch.sh
# itself is fully intercepted below by the fixture harness before any
# adapter/i05-bundle read happens): a real, executable lifecycle_adapter
# path and a real i05_bundle_dir per scenario_root, the same way
# tc094_e40_benchmark_operator_test.sh configures its own test-local
# runtime placeholders.
PROVIDER_SPY="$WORKDIR/provider-spy.sh"
cat >"$PROVIDER_SPY" <<'SH'
#!/usr/bin/env bash
echo "provider spy invoked unexpectedly" >&2
exit 97
SH
chmod +x "$PROVIDER_SPY"
mkdir -p "$WORKDIR/i05"
python3 - "$CONFIG" "$PROVIDER_SPY" "$WORKDIR/i05" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
config = yaml.safe_load(path.read_text(encoding="utf-8"))
config["runtime"]["lifecycle_adapter"] = sys.argv[2]
for scenario_id, entry in config["scenario_roots"].items():
    bundle = pathlib.Path(sys.argv[3]) / scenario_id
    bundle.mkdir(parents=True, exist_ok=True)
    entry["i05_bundle_dir"] = str(bundle)
path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY

FIXTURE_HARNESS="$WORKDIR/fixture-harness.py"
cat >"$FIXTURE_HARNESS" <<'PY'
import hashlib
import importlib.util
import json
import os
import pathlib
import subprocess
import sys

module_path, real_families_path, findings_path, gate_spec_path = sys.argv[1:5]

spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
real_run_process = module.run_process

with open(real_families_path, encoding="utf-8") as f:
    FAMILY_BY_SCENARIO = json.load(f)
with open(findings_path, encoding="utf-8") as f:
    FINDINGS_BY_SCENARIO = json.load(f)
with open(gate_spec_path, encoding="utf-8") as f:
    GATE_SPEC_BY_SCENARIO = json.load(f)

INSUFFICIENT_SCENARIO = os.environ.get("TC112_INSUFFICIENT_SCENARIO") or None


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def write_json(path, obj):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")


def base_stage():
    return {
        "dispatch_ordinal": 1, "stage": "code", "category": "code",
        "snapshot_digest": "a" * 64, "prompt_digest": "b" * 64,
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": {"provider": "fixture", "model": "fixture-model"},
        "cost_usd": 1.0, "elapsed_seconds": 1.0, "errors": [], "rework": False,
        "intervals": [{"category": "provider_active", "start": 0, "end": 1}],
        "candidate": {}, "artifacts": [], "access_events": [], "evidence_refs": {},
    }


def review_gates_for(scenario_id, rep):
    """AC-F11-43/AC-F11-44 (advisor-flagged gap): a real per-family
    review_gates[] entry, keyed by a scenario-scoped gate_id, for the
    scenarios this test names -- proves review_value.gates[] survives
    independently per family in aggregate.json itself (not just in the
    rendered reports), and that one family's zero_findings gate never
    absorbs a different family's findings gate (aggregate-lifecycle.sh's
    own gate_id-scoped-to-one-scenario resolution, design decision 8).
    Attached only on rep 1: aggregate-lifecycle.sh SUMS review_measures
    across every contributing rep (the normal multi-rep behavior), so
    seeding the same gate on every rep would double/triple its counts --
    rep 1 alone keeps this fixture's asserted raw counts exact."""
    if rep != 1:
        return []
    gate_spec = GATE_SPEC_BY_SCENARIO.get(scenario_id)
    if not gate_spec:
        return []
    return [{"gate_id": gate_spec["gate_id"], "state": gate_spec["state"], "round": gate_spec.get("round")}]


def review_measures_for(scenario_id, rep):
    if rep != 1:
        return []
    gate_spec = GATE_SPEC_BY_SCENARIO.get(scenario_id)
    if not gate_spec or not gate_spec.get("measures"):
        return []
    return [
        {
            "gate": gate_spec["gate_id"],
            "severity": "high",
            "defect_class": "logic",
            "measures": {k: {"value": v, "available": True} for k, v in gate_spec["measures"].items()},
        }
    ]


def build_pair(write_root, scenario_id, family, rep):
    pair_dir = write_root / "scenarios" / scenario_id / str(rep)
    package_yaml = pair_dir / "package.yaml"
    pair_dir.mkdir(parents=True, exist_ok=True)
    package_yaml.write_text(
        'schema_version: "1.0"\n'
        f'scenario_id: "{scenario_id}"\n'
        'scenario_version: "1"\n'
        f'entity_family: "{family}"\n',
        encoding="utf-8",
    )
    lc = {
        "identity": {
            "schema_version": "1.0", "run_id": f"run-{scenario_id}-{rep}", "scenario_id": scenario_id,
            "scenario_version": "1", "fixture_id": f"fixture-{scenario_id}", "fixture_digest": "c" * 64,
            "adapter_id": "fixture-adapter", "adapter_version": "1",
            "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
        },
        "entity_graph": {}, "dispatches": [], "stages": [base_stage()],
        "workflow_policy": {}, "review_gates": review_gates_for(scenario_id, rep), "questions": [],
        "limits": {
            "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
            "observed_cost_usd": 1.0, "observed_wall_clock_seconds": 1.0,
            "observed_generated_tasks": 1, "first_exceeded": None,
        },
        "outcome": {"terminal": "complete", "reason": f"{scenario_id} fixture", "partial_evidence": False, "publication_eligible": True},
    }
    write_json(pair_dir / "lifecycle.jsonl", lc)

    confirmed_findings = FINDINGS_BY_SCENARIO[scenario_id]
    quality_metrics = {
        "confirmed_findings": (
            {"value": confirmed_findings, "available": True}
            if confirmed_findings != "unavailable"
            else {"value": None, "available": False}
        ),
        "unconfirmed_findings": {"value": 0, "available": True},
        "precision": {"value": None, "available": False},
        "recall": {"value": None, "available": False},
        "truth_set_status": "truth-set-unavailable",
        "review_measures": review_measures_for(scenario_id, rep),
        "aggregate_eligible": {"value": None, "available": False, "detail": "eligibility is reported separately"},
    }
    ev = {
        "schema_version": "1.0", "evaluation_id": f"eval-{scenario_id}-{rep}", "identity": {}, "source_artifacts": {},
        "structural": {}, "judge": {}, "execution_oracle": {},
        "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
        "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
        "metrics": {
            "elapsed_time": {"value": 1.0, "available": True},
            "provider_cost": {"value": 1.0, "available": True},
            "rework": {"value": 0, "available": True},
            "quality": quality_metrics,
        },
    }
    write_json(pair_dir / "evaluation.jsonl", ev)

    manifest = {
        "scenario_id": scenario_id, "rep": rep,
        "artifacts": {
            "package.yaml": {"source_path": str(package_yaml), "sha256": sha(package_yaml)},
            "lifecycle.jsonl": {"source_path": str(pair_dir / "lifecycle.jsonl"), "sha256": sha(pair_dir / "lifecycle.jsonl")},
            "evaluation.jsonl": {"source_path": str(pair_dir / "evaluation.jsonl"), "sha256": sha(pair_dir / "evaluation.jsonl")},
        },
    }
    write_json(pair_dir / "manifest.json", manifest)


def fixture_run_process(command, **kwargs):
    if pathlib.Path(command[0]) == module.SCRIPTS_DIR / "run-lifecycle-batch.sh":
        values = {}
        for index, value in enumerate(command[:-1]):
            if value.startswith("--"):
                values[value] = command[index + 1]
        real_root = pathlib.Path(values["--retention-root"])
        write_root = pathlib.Path(
            f"/proc/self/fd/{values['--retention-root-fd']}"
            if "--retention-root-fd" in values
            else values["--retention-root"]
        )
        policy_path = pathlib.Path(values["--batch"])
        import yaml

        policy = yaml.safe_load(policy_path.read_text(encoding="utf-8")) or {}
        declared_min_reps = int(values["--reps"])
        for scenario_id, scenario_policy in (policy.get("scenarios") or {}).items():
            family = FAMILY_BY_SCENARIO[scenario_id]
            reps_to_write = 1 if scenario_id == INSUFFICIENT_SCENARIO else declared_min_reps
            for rep in range(1, reps_to_write + 1):
                build_pair(write_root, scenario_id, family, rep)

        batch = {
            "phase": "lifecycle_v2",
            "batch_id": f"batch-{real_root.name}-{values['--mode']}",
            "mode": values["--mode"],
            "retention_root": str(real_root),
            "batch_policy_digest": hashlib.sha256(policy_path.read_bytes()).hexdigest(),
            "min_reps": declared_min_reps,
            "ceilings": {
                "max_cost_usd": values["--max-cost-usd"],
                "max_wall_clock_seconds": values["--max-wall-clock-seconds"],
                "max_generated_tasks": values["--max-generated-tasks"],
                "max_provider_calls": values["--max-provider-calls"],
            },
            "acknowledgement_ref": {
                "flag": "--acknowledge-provider-spend",
                "present": "--acknowledge-provider-spend" in command,
            },
        }
        write_json(write_root / "batch.json", batch)
        return subprocess.CompletedProcess(command, 0, '{"fixture_driver":true}\n', "")
    return real_run_process(command, **kwargs)


module.run_process = fixture_run_process
sys.argv = ["e40-benchmark.sh", *sys.argv[5:]]
raise SystemExit(module.main())
PY

SIX_FAMILY_CEILINGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 900 --max-generated-tasks 20 --max-provider-calls 30)

# --- Choose, from the real registry, one scenario for a genuine measured
# "0 findings" clean pass, one for a genuine "not measured" gap, one for an
# insufficient-repetition case, and (arbitrary) small nonzero counts for the
# rest -- AC-F11-43's own precondition ("one family with 0 findings and one
# not measured").
ALL_SCENARIOS_JSON="$WORKDIR/all-scenarios-6.json"
python3 - "$REAL_FAMILIES_JSON" "$ALL_SCENARIOS_JSON" "$WORKDIR/zero-findings-scenario.txt" \
	"$WORKDIR/not-measured-scenario.txt" "$WORKDIR/insufficient-scenario.txt" <<'PYEOF'
import json
import sys

families_path, out_path, zero_path, not_measured_path, insufficient_path = sys.argv[1:6]
mapping = json.load(open(families_path))
scenario_ids = sorted(mapping)

findings = {}
for idx, sid in enumerate(scenario_ids):
    if idx == 0:
        findings[sid] = 0
    elif idx == 1:
        findings[sid] = "unavailable"
    else:
        findings[sid] = idx  # small distinct nonzero counts for the rest

with open(out_path, "w", encoding="utf-8") as f:
    json.dump(findings, f, sort_keys=True)
open(zero_path, "w", encoding="utf-8").write(scenario_ids[0])
open(not_measured_path, "w", encoding="utf-8").write(scenario_ids[1])
# Insufficient-repetition scenario: reuse the 3rd scenario if there is one,
# else fall back to the 0-findings scenario (still a real, distinct check).
open(insufficient_path, "w", encoding="utf-8").write(scenario_ids[2] if len(scenario_ids) > 2 else scenario_ids[0])
PYEOF

ZERO_FINDINGS_SCENARIO="$(cat "$WORKDIR/zero-findings-scenario.txt")"
NOT_MEASURED_SCENARIO="$(cat "$WORKDIR/not-measured-scenario.txt")"
INSUFFICIENT_SCENARIO="$(cat "$WORKDIR/insufficient-scenario.txt")"
ZERO_FINDINGS_FAMILY="$(python3 -c "import json; print(json.load(open('$REAL_FAMILIES_JSON'))['$ZERO_FINDINGS_SCENARIO'])")"
NOT_MEASURED_FAMILY="$(python3 -c "import json; print(json.load(open('$REAL_FAMILIES_JSON'))['$NOT_MEASURED_SCENARIO'])")"

# --- Real per-family review_gates[] (advisor-flagged gap): the zero-
# findings scenario's own gate is state=zero_findings; the not-measured
# scenario's own gate is a DIFFERENT gate_id, state=findings, with real
# seeded measures -- proves review_value.gates[] in aggregate.json itself
# carries both rows independently (never one family's state/counts
# absorbing the other's), not just the reports built from it. Used
# identically by both the all-real-families run and the reference-subset
# run below, so AC-F11-44's shape comparison stays meaningful.
GATE_SPEC_JSON="$WORKDIR/gate-spec.json"
python3 - "$ZERO_FINDINGS_SCENARIO" "$NOT_MEASURED_SCENARIO" "$GATE_SPEC_JSON" <<'PYEOF'
import json
import sys

zero_scenario, not_measured_scenario, out_path = sys.argv[1:4]
spec = {
    zero_scenario: {"gate_id": f"gate-{zero_scenario}", "state": "zero_findings", "round": 1},
    not_measured_scenario: {
        "gate_id": f"gate-{not_measured_scenario}",
        "state": "findings",
        "round": 1,
        "measures": {
            "emitted": 5, "normalized_unique": 4, "duplicate": 1,
            "recurrent": 0, "confirmed": 3, "unconfirmed": 1, "downstream_escape": 0,
        },
    },
}
json.dump(spec, open(out_path, "w", encoding="utf-8"), sort_keys=True)
PYEOF

RUN6_ID="tc112-run-all"
TC112_INSUFFICIENT_SCENARIO="$INSUFFICIENT_SCENARIO" python3 "$FIXTURE_HARNESS" "$E40_BENCHMARK_PY" \
	"$REAL_FAMILIES_JSON" "$ALL_SCENARIOS_JSON" "$GATE_SPEC_JSON" \
	baseline --config "$CONFIG" --run-id "$RUN6_ID" --reps 2 \
	"${SIX_FAMILY_CEILINGS[@]}" \
	>"$WORKDIR/run6.out" 2>"$WORKDIR/run6.err" && RUN6_RC=0 || RUN6_RC=$?
[[ "$RUN6_RC" -eq 0 ]] || fail "baseline (all real families) exited $RUN6_RC: $(cat "$WORKDIR/run6.out" "$WORKDIR/run6.err")"

RUN6_ROOT="$OPROOT/runs/$RUN6_ID"
AGG6="$RUN6_ROOT/aggregate.json"
HEADLINE6="$RUN6_ROOT/reports/headline.md"
STAGE6="$RUN6_ROOT/reports/stage-diagnostic.md"
[[ -f "$AGG6" ]] || fail "aggregate.json was not retained for the all-real-families baseline run"
[[ -f "$HEADLINE6" ]] || fail "headline.md was not retained for the all-real-families baseline run"
[[ -f "$STAGE6" ]] || fail "stage-diagnostic.md was not retained for the all-real-families baseline run"

# --- AC-F11-43: one independent "### `<family>`" section per SELECTED
# family in BOTH reports; the count matches the real family count derived
# from the aggregate itself (never a hardcoded 6). The zero-findings family
# renders the exact, full line "- Findings: 0 findings"; the not-measured
# family renders the exact, full line "- Findings: not measured" plus a
# named upstream_contract_gap on the very next line. Full-line matching
# (not naive substring search) so "10 findings" can never be mistaken for
# "0 findings".
python3 - "$AGG6" "$HEADLINE6" "$STAGE6" "$REAL_FAMILIES_JSON" \
	"$ZERO_FINDINGS_FAMILY" "$NOT_MEASURED_FAMILY" <<'PYEOF'
import json
import sys

agg_path, headline_path, stage_path, families_path, zero_family, not_measured_family = sys.argv[1:7]
agg = json.load(open(agg_path, encoding="utf-8"))
families_map = json.load(open(families_path, encoding="utf-8"))
expected_families = sorted(set(families_map.values()))


def fail(msg):
    raise SystemExit("TC-112 FAIL (B, AC-F11-43): " + msg)


agg_families = sorted({s["family"] for s in agg.get("scenarios") or []})
if agg_families != expected_families:
    fail(f"aggregate.json /scenarios[]/family set = {agg_families}, expected the real registry's set {expected_families}")


def check_report(path, view_label):
    text = open(path, encoding="utf-8").read()
    if "## Findings by family" not in text:
        fail(f"{view_label}: missing the '## Findings by family' section")
    lines = text.splitlines()
    heading_families = sorted(
        line.strip()[len("### `"):-1]
        for line in lines
        if line.strip().startswith("### `") and line.strip().endswith("`")
    )
    # Restrict to headings that actually match a real family name (the
    # per-scenario "### `sid` rep N" headings elsewhere in the report use a
    # different backtick-quoted token shape and are excluded by the
    # family-set membership check below).
    heading_families = sorted(h for h in heading_families if h in expected_families)
    if sorted(set(heading_families)) != expected_families:
        fail(
            f"{view_label}: 'Findings by family' section headings = {sorted(set(heading_families))}, "
            f"expected one independent section per selected family {expected_families}"
        )

    def section_lines(family):
        marker = f"### `{family}`"
        start = text.index(marker)
        rest = text[start:]
        end = rest.find("\n### `", 1)
        return rest[:end] if end != -1 else rest

    zero_section = section_lines(zero_family)
    if "- Findings: 0 findings" not in zero_section.splitlines():
        fail(f"{view_label}: family '{zero_family}' section does not render the exact full line '- Findings: 0 findings': {zero_section!r}")

    not_measured_section = section_lines(not_measured_family)
    section_lines_list = not_measured_section.splitlines()
    if "- Findings: not measured" not in section_lines_list:
        fail(f"{view_label}: family '{not_measured_family}' section does not render the exact full line '- Findings: not measured': {not_measured_section!r}")
    idx = section_lines_list.index("- Findings: not measured")
    gap_line = section_lines_list[idx + 1] if idx + 1 < len(section_lines_list) else ""
    if "upstream_contract_gap" not in gap_line:
        fail(f"{view_label}: 'not measured' verdict for '{not_measured_family}' was not followed by a named upstream_contract_gap: {gap_line!r}")

    # The two verdicts are never collapsed into one string.
    if "- Findings: 0 findings" in not_measured_section.splitlines():
        fail(f"{view_label}: '{not_measured_family}' section ALSO renders '0 findings' -- collapsed with the not-measured verdict")
    if "- Findings: not measured" in zero_section.splitlines():
        fail(f"{view_label}: '{zero_family}' section ALSO renders 'not measured' -- collapsed with the 0-findings verdict")


check_report(headline_path, "headline.md")
check_report(stage_path, "stage-diagnostic.md")
print(
    f"TC-112 (B, AC-F11-43): both reports carry one independent section per "
    f"selected family ({expected_families}); '0 findings' ({zero_family}) and "
    f"'not measured' ({not_measured_family}) render as distinct, never-collapsed, "
    f"full-line-matched strings, with a named upstream contract gap"
)
PYEOF

# --- AC-F11-44 (part 1): insufficient-repetition warning survives the
# six-family expansion -- the scenario given fewer reps than the batch's
# declared min_reps carries insufficient_reps: true on its own
# confirmed_findings noise band, never silently suppressed.
python3 - "$AGG6" "$INSUFFICIENT_SCENARIO" <<'PYEOF'
import json
import sys

agg_path, insufficient_scenario = sys.argv[1:3]
agg = json.load(open(agg_path, encoding="utf-8"))
bands = [
    b for b in agg.get("noise_bands") or []
    if b.get("scenario_id") == insufficient_scenario and b.get("metric") == "confirmed_findings"
]
if not bands:
    raise SystemExit(f"TC-112 FAIL (B, AC-F11-44): no confirmed_findings noise band for {insufficient_scenario}")
band = bands[0]
if band.get("insufficient_reps") is not True:
    raise SystemExit(
        f"TC-112 FAIL (B, AC-F11-44): {insufficient_scenario}'s confirmed_findings band "
        f"insufficient_reps = {band.get('insufficient_reps')!r}, expected True "
        f"(rep_count={band.get('rep_count')!r} < declared min_reps)"
    )
print(f"TC-112 (B, AC-F11-44 part 1): insufficient-repetition warning survives the six-family expansion for {insufficient_scenario}")
PYEOF

# --- AC-F11-43 (aggregate.json half, not just the rendered reports): the
# zero-findings scenario's own review_gates[] row and the not-measured
# scenario's own row -- two DISTINCT gate_ids, one per family -- both
# survive independently in aggregate.json's /review_value/gates[], never
# merged/absorbed into each other's state or counts. This is the direct
# "collapsed across families" hazard aggregate-lifecycle.sh's own gate_id-
# scoped-to-one-scenario resolution (design decision 8) exists to prevent;
# also confirms the noise_bands[] values for the zero-findings and
# not-measured scenarios stay scoped to their own scenario (no cross-
# family bleed), using the distinct seeded values this fixture builds.
python3 - "$AGG6" "$ZERO_FINDINGS_SCENARIO" "$NOT_MEASURED_SCENARIO" <<'PYEOF'
import json
import sys

agg_path, zero_scenario, not_measured_scenario = sys.argv[1:4]
agg = json.load(open(agg_path, encoding="utf-8"))


def fail(msg):
    raise SystemExit("TC-112 FAIL (B, AC-F11-43, aggregate.json half): " + msg)


gates = {g["gate_id"]: g for g in (agg.get("review_value") or {}).get("gates") or []}
zero_gate_id = f"gate-{zero_scenario}"
findings_gate_id = f"gate-{not_measured_scenario}"
if zero_gate_id not in gates:
    fail(f"expected gate {zero_gate_id!r} (the zero-findings family's own gate) in review_value.gates[], got {sorted(gates)}")
if findings_gate_id not in gates:
    fail(f"expected gate {findings_gate_id!r} (the not-measured family's own gate) in review_value.gates[], got {sorted(gates)}")

zero_gate = gates[zero_gate_id]
findings_gate = gates[findings_gate_id]
if zero_gate["state"] != "zero_findings":
    fail(f"{zero_gate_id}.state = {zero_gate['state']!r}, expected 'zero_findings' -- not absorbed by the other family's 'findings' gate")
if findings_gate["state"] != "findings":
    fail(f"{findings_gate_id}.state = {findings_gate['state']!r}, expected 'findings' -- not absorbed by the other family's 'zero_findings' gate")
zero_counts = zero_gate["counts"]
if any(zero_counts.values()):
    fail(f"{zero_gate_id}.counts = {zero_counts} -- expected all-zero (its own family's gate never absorbed the other family's nonzero counts)")
findings_counts = findings_gate["counts"]
if findings_counts.get("emitted") != 5 or findings_counts.get("confirmed") != 3:
    fail(f"{findings_gate_id}.counts = {findings_counts} -- expected this family's own seeded counts (emitted=5, confirmed=3), not merged with the zero-findings family's")

# noise_bands[] independence: each scenario's own confirmed_findings band
# carries only that scenario's own seeded value(s), never a neighbor's.
bands_by_scenario = {
    (b["scenario_id"], b["metric"]): b for b in agg.get("noise_bands") or []
}
zero_band = bands_by_scenario.get((zero_scenario, "confirmed_findings"))
not_measured_band = bands_by_scenario.get((not_measured_scenario, "confirmed_findings"))
if not zero_band or zero_band.get("max") != 0:
    fail(f"{zero_scenario}'s own confirmed_findings band = {zero_band} -- expected max=0, not bled from another family's seeded value")
if not not_measured_band or not_measured_band.get("min") != "unavailable":
    fail(f"{not_measured_scenario}'s own confirmed_findings band = {not_measured_band} -- expected 'unavailable', not bled from another family's seeded value")

print(
    "TC-112 (B, AC-F11-43, aggregate.json half): review_value.gates[] carries both families' "
    "gate rows independently (distinct gate_id, state, and counts, never merged), and each "
    "family's own noise_bands[] confirmed_findings value stays scoped to that family alone"
)
PYEOF

# --- Reference run: a 4-family subset (or fewer, if the real registry ever
# has fewer than 4) of the SAME real registry, structurally compared against
# the six-family (or however-many-real) run above -- AC-F11-44: "Counts,
# bands, and warnings unchanged in shape" as the matrix grows.
# The operator's own `--scenario` flag accepts exactly ONE scenario id, not
# a comma-joined set (build_matrix's `selected = [args.scenario]`), and
# `selected_matrix` walks the config's `scenario_index` file directly --
# trimming `scenario_roots` alone does not shrink the matrix. A reduced-
# family reference run is built by pointing a config COPY at a TRUNCATED
# scenario_index file (4, or fewer if the real registry ever has fewer,
# of the same real scenario package directories -- no new/synthetic
# package content) and a matching, trimmed `scenario_roots` map.
REFERENCE_CONFIG="$WORKDIR/e40-demo-reference.yaml"
REFERENCE_INDEX="$WORKDIR/reference-scenarios.yaml"
python3 - "$CONFIG" "$REFERENCE_CONFIG" "$REAL_SCENARIO_INDEX" "$REFERENCE_INDEX" \
	"$REAL_FAMILIES_JSON" "$WORKDIR/reference-findings.json" <<'PYEOF'
import json
import os
import sys

import yaml

config_path, reference_config_path, real_index_path, reference_index_path, families_path, findings_path = sys.argv[1:7]
config = yaml.safe_load(open(config_path, encoding="utf-8").read())
real_index = yaml.safe_load(open(real_index_path, encoding="utf-8").read())
mapping = json.load(open(families_path, encoding="utf-8"))
scenario_ids = sorted(mapping)
reference_ids = set(scenario_ids[: min(4, len(scenario_ids))])

real_index_dir = os.path.dirname(real_index_path)
reference_rel_dirs = [
    rel for rel in real_index["scenarios"]
    if yaml.safe_load(open(os.path.join(real_index_dir, rel, "package.yaml"), encoding="utf-8"))["scenario_id"]
    in reference_ids
]
reference_index = dict(real_index)
reference_index["scenarios"] = [os.path.join(real_index_dir, rel) for rel in reference_rel_dirs]
with open(reference_index_path, "w", encoding="utf-8") as f:
    yaml.safe_dump(reference_index, f, sort_keys=False)

config["scenario_index"] = reference_index_path
config["scenario_roots"] = {
    sid: entry for sid, entry in config["scenario_roots"].items() if sid in reference_ids
}
with open(reference_config_path, "w", encoding="utf-8") as f:
    yaml.safe_dump(config, f, sort_keys=False)

findings = {sid: (idx + 1) for idx, sid in enumerate(sorted(reference_ids))}
json.dump(findings, open(findings_path, "w", encoding="utf-8"), sort_keys=True)
PYEOF

RUN4_ID="tc112-run-reference"
python3 "$FIXTURE_HARNESS" "$E40_BENCHMARK_PY" "$REAL_FAMILIES_JSON" "$WORKDIR/reference-findings.json" "$GATE_SPEC_JSON" \
	baseline --config "$REFERENCE_CONFIG" --run-id "$RUN4_ID" --reps 1 \
	--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 900 \
	--max-generated-tasks 20 --max-provider-calls 30 \
	>"$WORKDIR/run4.out" 2>"$WORKDIR/run4.err" && RUN4_RC=0 || RUN4_RC=$?
[[ "$RUN4_RC" -eq 0 ]] || fail "baseline (reference subset) exited $RUN4_RC: $(cat "$WORKDIR/run4.out" "$WORKDIR/run4.err")"
AGG4="$OPROOT/runs/$RUN4_ID/aggregate.json"
[[ -f "$AGG4" ]] || fail "aggregate.json was not retained for the reference-subset baseline run"

# --- AC-F11-44 (part 2): structural shape equality between the reference
# subset's aggregate and the all-real-families aggregate -- same field
# names/types throughout, growing the family matrix adds LIST ITEMS, never
# new/removed/retyped fields. Then the four named mutations, each detected
# by name (spec.md row 44's own enumerated set).
python3 - "$AGG4" "$AGG6" "$ZERO_FINDINGS_FAMILY" "$NOT_MEASURED_FAMILY" "$INSUFFICIENT_SCENARIO" \
	"$SCRIPTS_DIR/report-lifecycle.sh" <<'PYEOF'
import copy
import json
import subprocess
import sys
import tempfile

agg4_path, agg6_path, zero_family, not_measured_family, insufficient_scenario, reporter = sys.argv[1:7]
agg4 = json.load(open(agg4_path, encoding="utf-8"))
agg6 = json.load(open(agg6_path, encoding="utf-8"))


def fail(msg):
    raise SystemExit("TC-112 FAIL (B, AC-F11-44): " + msg)


def shape(agg):
    """A structural fingerprint: top-level key set, plus the sorted key set
    of each item in every list-of-object block this aggregate carries --
    deliberately blind to CARDINALITY (how many families/scenarios/bands
    exist) and to VALUES, sensitive only to which fields exist. Counts,
    bands, and warnings surviving the six-family expansion 'unchanged in
    shape' means exactly this: identical field vocabulary at every level."""
    result = {"top_level": sorted(agg.keys())}
    for list_field in ("scenarios", "noise_bands", "comparisons", "invalid"):
        items = agg.get(list_field) or []
        result[list_field] = sorted({tuple(sorted(item.keys())) for item in items if isinstance(item, dict)})
    review_gates = (agg.get("review_value") or {}).get("gates") or []
    result["review_value.gates"] = sorted({tuple(sorted(g.keys())) for g in review_gates if isinstance(g, dict)})
    artifact_use = agg.get("artifact_use") or {}
    result["artifact_use"] = sorted(artifact_use.keys())
    result["time"] = sorted((agg.get("time") or {}).keys())
    result["cost"] = sorted((agg.get("cost") or {}).keys())

    # X-09 preservation (this task's own Cross-epic row): the six-family
    # aggregate must carry the SAME provider-usage field name/type set as
    # the four-family aggregate -- cost.ceiling_consumption's own field
    # names/types, the time/cost partition value TYPES, and each review
    # gate's own provider_cost_usd/elapsed_seconds/resolution_cost_usd
    # field TYPE (these are the upstream-contract-gap string in both
    # aggregates today, per report-lifecycle.sh's own
    # GATE_TIME_COST_UPSTREAM_GAP precedent -- their TYPE, not their
    # value, is what must not silently change shape as families grow).
    ceiling_consumption = (agg.get("cost") or {}).get("ceiling_consumption") or {}
    result["cost.ceiling_consumption"] = sorted(
        (k, type(v).__name__) for k, v in ceiling_consumption.items()
    )
    for label, block_name in (("time", "time"), ("cost", "cost")):
        block = agg.get(block_name) or {}
        result[f"{label}.partition_value_types"] = sorted(
            {
                type(v).__name__
                for partition_name in ("stage_category", "interval_category", "share_partition")
                for v in (block.get(partition_name) or {}).values()
            }
        )
    result["review_value.gates_usage_field_types"] = sorted(
        {
            (field, type(g.get(field)).__name__)
            for g in review_gates
            if isinstance(g, dict)
            for field in ("provider_cost_usd", "elapsed_seconds", "resolution_cost_usd")
        }
    )
    return result


shape4 = shape(agg4)
shape6 = shape(agg6)
if shape4 != shape6:
    fail(f"reference-subset shape != all-real-families shape: {shape4} vs {shape6}")
print("TC-112 (B, AC-F11-44 part 2): the 4-scenario reference aggregate and the all-real-families aggregate share the identical structural shape")

# --- Mutation 1: "a family section dropped" -- remove one family's own
# scenarios[] and noise_bands[] entries from a copy; the family-section
# count in a re-rendered headline report must drop below the real family
# count, and the dropped family's own heading must disappear.
mutated_drop = copy.deepcopy(agg6)
dropped_scenario_ids = {s["scenario_id"] for s in mutated_drop["scenarios"] if s["family"] == not_measured_family}
mutated_drop["scenarios"] = [s for s in mutated_drop["scenarios"] if s["family"] != not_measured_family]
mutated_drop["noise_bands"] = [b for b in mutated_drop["noise_bands"] if b["scenario_id"] not in dropped_scenario_ids]
with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as fh:
    json.dump(mutated_drop, fh)
    mutated_drop_path = fh.name
rendered = subprocess.run([reporter, "--aggregate", mutated_drop_path, "--view", "headline"], capture_output=True, text=True, check=True)
if f"### `{not_measured_family}`" in rendered.stdout:
    fail(f"mutation 'family section dropped' not detected: '{not_measured_family}' heading still present after removing its scenarios/noise_bands entries")
print("TC-112 (B, AC-F11-44 mutation 'family section dropped'): detected -- the dropped family's own section disappears from the re-rendered report")

# --- Mutation 2: "zero-findings collapsed into not-measured" -- corrupt the
# zero-findings family's own noise band to look unmeasured; the re-rendered
# report must switch from the '0 findings' line to the 'not measured' line
# for that family (a real, functional detection, not just a JSON diff).
mutated_collapse = copy.deepcopy(agg6)
zero_family_scenario_ids = {s["scenario_id"] for s in agg6["scenarios"] if s["family"] == zero_family}
for band in mutated_collapse["noise_bands"]:
    if band["scenario_id"] in zero_family_scenario_ids and band["metric"] == "confirmed_findings":
        band["min"] = "unavailable"
        band["median"] = "unavailable"
        band["max"] = "unavailable"
        band["rep_count"] = 0
        band["insufficient_reps"] = True
with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as fh:
    json.dump(mutated_collapse, fh)
    mutated_collapse_path = fh.name
rendered = subprocess.run([reporter, "--aggregate", mutated_collapse_path, "--view", "headline"], capture_output=True, text=True, check=True)
start = rendered.stdout.index(f"### `{zero_family}`")
rest = rendered.stdout[start:]
end = rest.find("\n### `", 1)
section = rest[:end] if end != -1 else rest
if "- Findings: 0 findings" in section.splitlines():
    fail("mutation 'zero-findings collapsed into not-measured' not detected: the corrupted family's section still renders '0 findings'")
if "- Findings: not measured" not in section.splitlines():
    fail(f"mutation 'zero-findings collapsed into not-measured': expected the corrupted family's section to now render 'not measured', got: {section!r}")
print("TC-112 (B, AC-F11-44 mutation 'zero-findings collapsed into not-measured'): detected -- corrupting the family's own noise band flips its rendered verdict from '0 findings' to 'not measured'")

# --- Mutation 3: "a noise band omitted" -- delete the not-measured family's
# own confirmed_findings noise_bands[] entry ENTIRELY (not just its values)
# from a copy; aggregate-lifecycle.sh always publishes exactly
# len(distinct scenario_ids) * 3 noise-band metrics (confirmed_findings,
# elapsed_time, provider_cost), so a missing entry is detected by recount.
mutated_omit = copy.deepcopy(agg6)
not_measured_scenario_ids = {s["scenario_id"] for s in agg6["scenarios"] if s["family"] == not_measured_family}
before_count = len(mutated_omit["noise_bands"])
mutated_omit["noise_bands"] = [
    b for b in mutated_omit["noise_bands"]
    if not (b["scenario_id"] in not_measured_scenario_ids and b["metric"] == "confirmed_findings")
]
after_count = len(mutated_omit["noise_bands"])
distinct_scenarios = {s["scenario_id"] for s in mutated_omit["scenarios"]}
expected_band_count = len(distinct_scenarios) * 3
if after_count == expected_band_count:
    fail(f"mutation 'noise band omitted' not detected: expected band count mismatch, got before={before_count} after={after_count} expected={expected_band_count}")
print("TC-112 (B, AC-F11-44 mutation 'noise band omitted'): detected -- deleting one family's confirmed_findings band entry breaks the closed per-scenario x per-metric count invariant")

# --- Mutation 4: "insufficient-repetition warning suppressed" -- flip the
# insufficient scenario's own insufficient_reps flag from True to False in a
# copy; a recomputation against the published rep_count/min_reps catches it.
mutated_suppress = copy.deepcopy(agg6)
min_reps = mutated_suppress["identity"]["min_reps"]
suppressed = False
for band in mutated_suppress["noise_bands"]:
    if band["scenario_id"] == insufficient_scenario and band["metric"] == "confirmed_findings":
        if band.get("insufficient_reps") is not True:
            fail("mutation setup error: insufficient scenario's band was not True before suppression")
        band["insufficient_reps"] = False
        suppressed = True
if not suppressed:
    fail("mutation setup error: could not find the insufficient scenario's confirmed_findings band to suppress")

recomputed_bad = [
    b for b in mutated_suppress["noise_bands"]
    if b["scenario_id"] == insufficient_scenario
    and b["metric"] == "confirmed_findings"
    and isinstance(b.get("rep_count"), int)
    and b["rep_count"] < min_reps
    and b.get("insufficient_reps") is not True
]
if not recomputed_bad:
    fail("mutation 'insufficient-repetition warning suppressed' not detected: recomputation (rep_count < min_reps) did not flag the suppressed band")
print("TC-112 (B, AC-F11-44 mutation 'insufficient-repetition warning suppressed'): detected -- recomputing rep_count < min_reps catches the suppressed flag")

print("TC-112 (B, AC-F11-44): all four named shape mutations are individually detected against the reference shape")
PYEOF

echo "TC-112: PASS (pilot attestation, aggregation, and reports generalize to every family the real registry admits, with zero private family list, and '0 findings'/'not measured' never collapsed)"
