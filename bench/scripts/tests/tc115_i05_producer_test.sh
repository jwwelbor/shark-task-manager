#!/usr/bin/env bash
# TC-115 / T-E40-F12-001/002: `--i05-bundle-dir` option/mode decision table
# (TC-001), the `bundle.json` triad + REQ-F-014 additive join fields
# (TC-002/TC-003), the stage-snapshot writer field completeness + required
# `provider` (TC-004), `snapshot_digest` immutability/mutation detection
# (TC-005), `candidate` partitioned by `stage_category` (TC-006), the
# `stage_category` closed-table exhaustive coverage (TC-007), `access.jsonl`
# existence/append-only (TC-010), the producer-owned reset and symlink
# refusal (TC-016), and the I-05 half of the I-07 evaluator join (TC-021,
# producer half). T-E40-F12-007 appends the feature-level final cases: the
# end-to-end `evaluate-lifecycle.sh` join proof (TC-021, final), the
# unmodified validator's real six-family acceptance (TC-023), its
# byte-identity since the pre-feature ref (TC-024), the spec.md sec 3.2 /
# Q008 disclosure (TC-025), and the three NFR guards -- per-dispatch write
# overhead (TC-026), no secret/prompt leakage (TC-027), and deterministic
# writes (TC-028).
#
# TC-019/TC-020 (T-E40-F12-006's own --i05-bundle-dir pass-through +
# guard-reorder AC coverage) and the T-E40-F12-006 kickback's REWORK guard
# cases live in tc117_i05_caller_pass_through_guard_test.sh, NOT here: this
# file's `set -euo pipefail` + shared `fail()` means one earlier case's
# failure aborts the whole process, and those cases used to sit downstream
# of TC-008, so a TC-008 failure silently prevented them from ever running.
# run-all.sh runs tc117 as its own independent subprocess instead.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
SCENARIO="$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml"
I05_SCHEMA="$BENCH_DIR/evidence/i05-schema.yaml"
STAGE_CATEGORY_MAP="$BENCH_DIR/evidence/stage-category-map.yaml"
WORKFLOW_FIXTURE_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow"
REPLAY_SCRIPT="$SCRIPTS_DIR/replay-stage-evidence.sh"
fail() { echo "TC-115 FAIL: $1" >&2; exit 1; }
[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable"
[[ -f "$SCENARIO" ]] || fail "scenario package missing: $SCENARIO"
[[ -d "$WORKFLOW_FIXTURE_DIR" ]] || fail "frozen workflow-phase fixtures missing: $WORKFLOW_FIXTURE_DIR"
[[ -x "$REPLAY_SCRIPT" ]] || fail "replay-stage-evidence.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

# A single-dispatch `shark` stub (TC-061's established convention): root
# ROOT-001 resolves directly to a spawn_agent dispatch for the given entity
# key (TASK-002 by default). Answers `admin workflow list <level> --json`
# from the frozen, real-captured fixtures under
# testdata/lifecycle/workflow/<level>.json (TC-002's precondition: "real
# installed workflow phase data" and "stubbed shark binary" are the same
# declared seam throughout this suite) when SHARK_WORKFLOW_DIR is exported.
write_single_dispatch_shark() {
	local bin_dir="$1"
	local entity_key="${2:-TASK-002}"
	cat >"$bin_dir/shark" <<SHARK
#!/usr/bin/env bash
set -euo pipefail
python3 - "\$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
if os.environ.get("SHARK_EVENTS"):
    with open(os.environ["SHARK_EVENTS"], "a") as f:
        f.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    response = json.load(open(os.environ["SHARK_RESPONSE"]))
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    fixture_dir = os.environ["SHARK_WORKFLOW_DIR"]
    with open(os.path.join(fixture_dir, level + ".json")) as f:
        print(f.read())
elif args[:2] == ["claim", "$entity_key"]:
    print('{"session_id":"SID-002"}')
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
	chmod +x "$bin_dir/shark"
}

write_adapter() {
	local path="$1" worker_id="$2"
	cat >"$path" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
python3 -c 'import json,sys; request=json.load(sys.stdin); print(json.dumps({"worker_id":"$worker_id","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":0.0,"evidence":{"summary":"fixture"}}))'
ADAPTER
	chmod +x "$path"
}

# ---------------------------------------------------------------------------
# TC-001: --i05-bundle-dir mode/option decision table (AC-001)
# ---------------------------------------------------------------------------

# (a) --mode live, no --i05-bundle-dir: exits non-zero before the first
# `shark next` call; stderr names the missing option; SHARK_BIN is never
# invoked.
WORKDIR_A="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_A"' EXIT
mkdir -p "$WORKDIR_A/bin" "$WORKDIR_A/scratch"
write_single_dispatch_shark "$WORKDIR_A/bin"
if PATH="$WORKDIR_A/bin:$PATH" SHARK_EVENTS="$WORKDIR_A/events.ndjson" \
	SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" "$RUNNER" \
	--scenario "$SCENARIO" --run-id tc115a --root ROOT-001 --scratch-root "$WORKDIR_A/scratch" \
	--mode live >/dev/null 2>"$WORKDIR_A/err"; then
	fail "(a) --mode live without --i05-bundle-dir unexpectedly succeeded"
fi
grep -q -- "--i05-bundle-dir is required" "$WORKDIR_A/err" || fail "(a) missing option was not named in stderr"
[[ -f "$WORKDIR_A/events.ndjson" ]] && fail "(a) shark was invoked despite the missing required option"
rm -rf "$WORKDIR_A"
trap - EXIT

# (b) --mode resolve-route --i05-bundle-dir <dir>: exits non-zero (2) naming
# the rejected option; <dir> is byte-for-byte untouched.
WORKDIR_B="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_B"' EXIT
mkdir -p "$WORKDIR_B/bin" "$WORKDIR_B/scratch" "$WORKDIR_B/i05"
write_single_dispatch_shark "$WORKDIR_B/bin"
before="$(find "$WORKDIR_B/i05" | sort)"
set +e
PATH="$WORKDIR_B/bin:$PATH" "$RUNNER" --scenario "$SCENARIO" --run-id tc115b --root ROOT-001 \
	--scratch-root "$WORKDIR_B/scratch" --mode resolve-route --i05-bundle-dir "$WORKDIR_B/i05" \
	>/dev/null 2>"$WORKDIR_B/err"
rc=$?
set -e
[[ "$rc" -eq 2 ]] || fail "(b) --mode resolve-route --i05-bundle-dir did not exit 2 (got $rc)"
grep -q -- "--i05-bundle-dir is not supported with --mode resolve-route" "$WORKDIR_B/err" || fail "(b) rejected option was not named in stderr"
after="$(find "$WORKDIR_B/i05" | sort)"
[[ "$before" == "$after" ]] || fail "(b) --i05-bundle-dir was written to despite resolve-route rejection"
rm -rf "$WORKDIR_B"
trap - EXIT

# (c) --mode resolve-route, no --i05-bundle-dir: regression guard only -- the
# new option check must never fire when the option is absent, whatever else
# the run does or does not accomplish.
WORKDIR_C="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_C"' EXIT
mkdir -p "$WORKDIR_C/scratch"
set +e
"$RUNNER" --scenario "$WORKDIR_C/missing.yaml" --run-id tc115c --root ROOT-001 \
	--scratch-root "$WORKDIR_C/scratch" --mode resolve-route >/dev/null 2>"$WORKDIR_C/err"
set -e
grep -q -- "--i05-bundle-dir" "$WORKDIR_C/err" && fail "(c) resolve-route without the option was rejected for it"
rm -rf "$WORKDIR_C"
trap - EXIT

# (d) --mode contract, no --i05-bundle-dir: unaffected, no-op (existing
# offline behavior regression guard).
WORKDIR_D="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_D"' EXIT
mkdir -p "$WORKDIR_D/bin" "$WORKDIR_D/scratch"
write_single_dispatch_shark "$WORKDIR_D/bin"
PATH="$WORKDIR_D/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115d --root ROOT-001 --scratch-root "$WORKDIR_D/scratch" \
	--mode contract --output "$WORKDIR_D/lifecycle.jsonl" || fail "(d) --mode contract without the option regressed"
rm -rf "$WORKDIR_D"
trap - EXIT

# (e) --mode dry-run, --i05-bundle-dir present: full bundle written
# (equivalence with the live-present case, one representative check).
WORKDIR_E="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_E"' EXIT
mkdir -p "$WORKDIR_E/bin" "$WORKDIR_E/scratch" "$WORKDIR_E/i05"
write_single_dispatch_shark "$WORKDIR_E/bin"
PATH="$WORKDIR_E/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115e --root ROOT-001 --scratch-root "$WORKDIR_E/scratch" \
	--mode dry-run --i05-bundle-dir "$WORKDIR_E/i05" --output "$WORKDIR_E/lifecycle.jsonl" \
	|| fail "(e) --mode dry-run with the option present failed"
[[ -f "$WORKDIR_E/i05/bundle.json" ]] || fail "(e) --mode dry-run with the option present did not write bundle.json"
rm -rf "$WORKDIR_E"
trap - EXIT

echo "TC-115 TC-001: pass (mode/option decision table)"

# ---------------------------------------------------------------------------
# TC-002/TC-003: bundle.json triad, REQ-F-014 additive fields, and stages[]
# index integrity vs I-07 dispatch ordinals -- a real >=2-dispatch live run.
# ---------------------------------------------------------------------------

WORKDIR_2="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_2"' EXIT
mkdir -p "$WORKDIR_2/bin" "$WORKDIR_2/scratch" "$WORKDIR_2/i05"

cat >"$WORKDIR_2/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
with open(os.environ["SHARK_EVENTS"], "a") as f:
    f.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"bug","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"TASK-002","entity_type":"task"},
                    {"entity_key":"TASK-001","entity_type":"task"}]}, separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":"task","status":"development",
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    fixture_dir = os.environ["SHARK_WORKFLOW_DIR"]
    with open(os.path.join(fixture_dir, level + ".json")) as f:
        print(f.read())
elif args[0] == "claim": print(json.dumps({"session_id":"SID-" + args[1]}))
elif args[0] == "heartbeat": print('{"ok":true}')
elif args[:2] == ["status", "advance"]: print('{"advanced":true}')
elif args[0] == "release": print('{"released":true}')
else: raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_2/bin/shark"
write_adapter "$WORKDIR_2/adapter.sh" "worker-115"

PATH="$WORKDIR_2/bin:$PATH" SHARK_EVENTS="$WORKDIR_2/events.ndjson" LIFECYCLE_ADAPTER="$WORKDIR_2/adapter.sh" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-002 --root ROOT-001 --scratch-root "$WORKDIR_2/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_2/i05" --output "$WORKDIR_2/lifecycle.jsonl" \
	|| fail "TC-002/003 real live run failed"

python3 - "$WORKDIR_2/i05" "$WORKDIR_2/lifecycle.jsonl" "$I05_SCHEMA" "$SCENARIO" <<'PY'
import hashlib
import json
import sys

import yaml

i05_dir, i07_path, schema_path, scenario_path = sys.argv[1:5]


def canonical_digest(value):
    encoded = json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def stage_snapshot_digest(snapshot):
    # Matches replay-stage-evidence.sh's own recompute_snapshot_digest()
    # byte-for-byte (run-lifecycle.sh's stage_snapshot_digest() docstring
    # explains why this is NOT canonical_digest()'s compact form).
    payload = {k: v for k, v in snapshot.items() if k != "snapshot_digest"}
    return "sha256:" + hashlib.sha256(json.dumps(payload, sort_keys=True).encode("utf-8")).hexdigest()


with open(f"{i05_dir}/bundle.json", encoding="utf-8") as f:
    bundle = json.load(f)
with open(i07_path, encoding="utf-8") as f:
    i07 = json.loads(f.readline())
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f)
with open(scenario_path, encoding="utf-8") as f:
    scenario = yaml.safe_load(f)

# AC-002: exactly the ten documented fields plus REQ-F-014's three additive
# fields, never a fourteenth.
allowed = {
    "schema_version", "scenario", "run_id", "roots", "stage_matrix_source",
    "stages", "terminal_status", "stop_outcome", "publication_eligible",
    "ineligibility_reasons", "scenario_id", "scenario_version", "dispatches",
}
assert set(bundle.keys()) <= allowed, f"unexpected top-level bundle.json keys: {set(bundle.keys()) - allowed}"
assert "stop_outcome" not in bundle, "clean run must not carry stop_outcome"

assert bundle["schema_version"] == schema["schema_version"]
assert bundle["scenario"] == {
    "scenario_id": scenario["scenario_id"],
    "scenario_version": str(scenario["scenario_version"]),
    "entity_family": scenario["entity_family"],
}
assert bundle["publication_eligible"] is True
assert bundle["ineligibility_reasons"] == []
assert bundle["terminal_status"]["reached"] is True
assert bundle["terminal_status"]["reached_at"]

# AC-003: one stages[] entry per dispatch; ordinal set equality vs I-07;
# every snapshot_digest reconciles against its real file.
dispatches = i07["dispatches"]
assert len(dispatches) == 2, f"expected 2 dispatches, got {len(dispatches)}"
stages = bundle["stages"]
assert len(stages) == len(dispatches)
assert {s["dispatch_ordinal"] for s in stages} == {d["ordinal"] for d in dispatches}
ordinals = [s["dispatch_ordinal"] for s in stages]
assert len(set(ordinals)) == len(ordinals), "dispatch_ordinal values in stages[] must be unique"
# TC-004 (AC-004): every stages/<ordinal>-<stage_key>.json file parses and
# carries every bench/README.md "Stage-snapshot field reference" field, plus
# a non-empty top-level `provider`. Both dispatches here are a task in
# "development" (phase "development" -> stage_category "code" per
# stage-category-map.yaml), so `candidate` is also asserted present.
REQUIRED_SNAPSHOT_FIELDS = {
    "dispatch_ordinal", "entity", "stage_key", "stage_category", "prompt_digest",
    "input_lineage", "artifacts", "usage", "time_ledger", "errors",
    "rework_count", "evaluator_access", "snapshot_digest", "provider", "candidate",
}
for stage in stages:
    with open(f"{i05_dir}/{stage['snapshot_path']}", encoding="utf-8") as f:
        snapshot = json.load(f)
    missing = REQUIRED_SNAPSHOT_FIELDS - set(snapshot.keys())
    assert not missing, f"snapshot {stage['snapshot_path']} missing fields: {missing}"
    assert isinstance(snapshot["provider"], str) and snapshot["provider"], "top-level provider must be a non-empty string"
    assert snapshot["stage_category"] == "code"
    assert stage["stage_category"] == "code", "bundle.json stages[] index must also carry stage_category"
    # The fixture's own "fixture" provider name is not a real usage-mapping.yaml
    # key, so the honest, fail-closed outcome is one unmapped_provider error
    # (X-09) -- never unknown_stage_category or missing_provider, since both
    # the category and the provider string itself resolved successfully.
    error_kinds = {e["kind"] for e in snapshot["errors"]}
    assert error_kinds <= {"unmapped_provider"}, f"unexpected errors on a fully-categorized/provided snapshot: {snapshot['errors']}"
    candidate = snapshot["candidate"]
    for field in ("base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest",
                  "dirty_untracked_manifest", "test_suite_digest", "test_suite_ids", "test_suite_dir"):
        assert field in candidate, f"candidate missing {field}"

    recorded_digest = snapshot.pop("snapshot_digest")
    assert recorded_digest == stage["snapshot_digest"]
    assert stage_snapshot_digest(snapshot) == recorded_digest, "snapshot_digest does not reconcile with its file"

# TC-010 partition 1: access.jsonl exists and is empty after a clean run
# with zero evaluator_access events.
access_path = f"{i05_dir}/access.jsonl"
import os as _os
assert _os.path.isfile(access_path), "access.jsonl must exist after every run"
assert _os.path.getsize(access_path) == 0, "access.jsonl must be empty when no evaluator_access events occurred"

# TC-021 (producer half, AC-019): the I-05/I-07 join fields this task owns
# agree exactly -- run_id/scenario_id/scenario_version and the dispatches
# reference list (canonical-digest equality, the same comparison
# evaluate-lifecycle.sh's validate_identity_join() performs).
identity = i07["identity"]
assert bundle["run_id"] == identity["run_id"]
assert bundle["scenario_id"] == identity["scenario_id"]
assert bundle["scenario_version"] == identity["scenario_version"]
assert canonical_digest(bundle["dispatches"]) == canonical_digest(dispatches), "I-05/I-07 dispatches join disagrees"
assert isinstance(bundle["roots"], dict) and bundle["roots"]
PY

echo "TC-115 TC-002/TC-003/TC-004/TC-010(partition 1)/TC-021(producer half): pass"

rm -rf "$WORKDIR_2"
trap - EXIT

# ---------------------------------------------------------------------------
# TC-005: snapshot_digest immutability and mutation detection, via the real
# (unmodified) replay-stage-evidence.sh -- the script that actually owns
# recompute_snapshot_digest()/the snapshot_mutated verdict (Spec Drift item
# 2: spec.md's own AC-005 text misnames verify-stage-evidence.sh). Uses a
# `discovery`-category dispatch (bug/research) specifically so no
# `candidate` block is present -- replay-stage-evidence.sh's own drift
# checks require `--checkout`/`--adapter` only for a `code`/`review`
# snapshot carrying `candidate` (out of this test's scope; the file-drift
# and test-suite-drift checks against `candidate.test_suite_ids` are a
# later task's replay-guard concern, not this task's own AC-005/TC-005).
# ---------------------------------------------------------------------------

WORKDIR_5="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_5"' EXIT
mkdir -p "$WORKDIR_5/bin" "$WORKDIR_5/scratch" "$WORKDIR_5/i05"
write_single_dispatch_shark "$WORKDIR_5/bin" "BUG-900"
PATH="$WORKDIR_5/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-research.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-005 --root ROOT-001 --scratch-root "$WORKDIR_5/scratch" \
	--mode dry-run --i05-bundle-dir "$WORKDIR_5/i05" --output "$WORKDIR_5/lifecycle.jsonl" \
	|| fail "TC-005 fixture run failed"

python3 -c "
import json
b = json.load(open('$WORKDIR_5/i05/bundle.json'))
assert b['stages'][0]['stage_category'] == 'discovery', b['stages'][0]
with open('$WORKDIR_5/i05/' + b['stages'][0]['snapshot_path']) as f:
    s = json.load(f)
assert 'candidate' not in s, 'a discovery-category snapshot must carry no candidate block'
"

"$REPLAY_SCRIPT" "$WORKDIR_5/i05" >/dev/null 2>"$WORKDIR_5/replay-before.err" \
	|| fail "TC-005 replay-stage-evidence.sh rejected an untampered bundle: $(cat "$WORKDIR_5/replay-before.err")"

MUTATED_SNAPSHOT="$(python3 -c "
import json
bundle = json.load(open('$WORKDIR_5/i05/bundle.json'))
print('$WORKDIR_5/i05/' + bundle['stages'][0]['snapshot_path'])
")"
python3 -c "
path = '$MUTATED_SNAPSHOT'
with open(path, 'r+', encoding='utf-8') as f:
    text = f.read()
    # Flip one byte inside stage_key -- a field OTHER than snapshot_digest
    # itself, per AC-005's own scoping.
    mutated = text.replace('\"research\"', '\"researcH\"', 1)
    assert mutated != text, 'mutation did not change the file'
    f.seek(0)
    f.write(mutated)
    f.truncate()
"
set +e
"$REPLAY_SCRIPT" "$WORKDIR_5/i05" >"$WORKDIR_5/replay-after.out" 2>"$WORKDIR_5/replay-after.err"
rc=$?
set -e
[[ "$rc" -ne 0 ]] || fail "TC-005 replay-stage-evidence.sh accepted a mutated snapshot"
grep -q "snapshot_mutated" "$WORKDIR_5/replay-after.out" "$WORKDIR_5/replay-after.err" \
	|| fail "TC-005 mutation was not reported as snapshot_mutated"

echo "TC-115 TC-005: pass (snapshot_digest immutability + mutation detection)"

rm -rf "$WORKDIR_5"
trap - EXIT

# ---------------------------------------------------------------------------
# TC-006: `candidate` block partitioned by `stage_category` -- a fixture
# exercising one `code`-category (task/development), one `review`-category
# (bug/code_review), and one `discovery`-category (feature/research)
# dispatch in a single run (AC-006).
# ---------------------------------------------------------------------------

WORKDIR_6="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_6"' EXIT
mkdir -p "$WORKDIR_6/bin" "$WORKDIR_6/scratch" "$WORKDIR_6/i05"

cat >"$WORKDIR_6/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
ROWS = {
    "TASK-CODE": ("task", "development"),
    "BUG-REVIEW": ("bug", "code_review"),
    "FEATURE-DISCOVERY": ("feature", "research"),
}
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"bug","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":k,"entity_type":v[0]} for k, v in ROWS.items()]},
        separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; entity_type, status = ROWS[key]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":entity_type,"status":status,
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], level + ".json")) as f:
        print(f.read())
elif args[0] == "claim": print(json.dumps({"session_id":"SID-" + args[1]}))
elif args[0] == "heartbeat": print('{"ok":true}')
elif args[:2] == ["status", "advance"]: print('{"advanced":true}')
elif args[0] == "release": print('{"released":true}')
else: raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_6/bin/shark"
write_adapter "$WORKDIR_6/adapter.sh" "worker-006"

PATH="$WORKDIR_6/bin:$PATH" LIFECYCLE_ADAPTER="$WORKDIR_6/adapter.sh" SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-006 --root ROOT-001 --scratch-root "$WORKDIR_6/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_6/i05" --output "$WORKDIR_6/lifecycle.jsonl" \
	|| fail "TC-006 fixture run failed"

python3 -c "
import json
bundle = json.load(open('$WORKDIR_6/i05/bundle.json'))
by_category = {}
for stage in bundle['stages']:
    with open('$WORKDIR_6/i05/' + stage['snapshot_path']) as f:
        snapshot = json.load(f)
    by_category[snapshot['stage_category']] = snapshot
assert set(by_category) == {'code', 'review', 'discovery'}, by_category.keys()

REQ_F_006_FIELDS = (
    'base_commit', 'tree_digest', 'binary_diff_digest', 'changed_path_digest',
    'dirty_untracked_manifest', 'test_suite_digest', 'test_suite_ids', 'test_suite_dir',
)
for category in ('code', 'review'):
    candidate = by_category[category]['candidate']
    for field in REQ_F_006_FIELDS:
        assert field in candidate, f'{category} candidate missing {field}'

discovery_snapshot = by_category['discovery']
assert 'candidate' not in discovery_snapshot, 'discovery snapshot must carry no candidate key at all'
"

echo "TC-115 TC-006: pass (candidate partitioned by stage_category: code/review/discovery)"

rm -rf "$WORKDIR_6"
trap - EXIT

# Zero-dispatch edge case (REQ-F-002 "before the first dispatch"): a run
# whose root resolves directly to a non-spawn terminal action still writes
# bundle.json with an empty stages[] index.
WORKDIR_ZERO="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_ZERO"' EXIT
mkdir -p "$WORKDIR_ZERO/bin" "$WORKDIR_ZERO/scratch" "$WORKDIR_ZERO/i05"
cat >"$WORKDIR_ZERO/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    print('{"action":"archive","entity_key":"ROOT-001","error":"scenario archived"}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_ZERO/bin/shark"
PATH="$WORKDIR_ZERO/bin:$PATH" "$RUNNER" --scenario "$SCENARIO" --run-id tc115-zero --root ROOT-001 \
	--scratch-root "$WORKDIR_ZERO/scratch" --mode dry-run --i05-bundle-dir "$WORKDIR_ZERO/i05" \
	--output "$WORKDIR_ZERO/lifecycle.jsonl" >/dev/null 2>&1 || true
[[ -f "$WORKDIR_ZERO/i05/bundle.json" ]] || fail "zero-dispatch run did not write bundle.json"
python3 -c "import json; b = json.load(open('$WORKDIR_ZERO/i05/bundle.json')); assert b['stages'] == []" \
	|| fail "zero-dispatch bundle.json stages[] is not empty"
rm -rf "$WORKDIR_ZERO"
trap - EXIT

echo "TC-115 TC-002 (zero-dispatch edge case): pass"

# ---------------------------------------------------------------------------
# TC-016: producer-owned reset scope + symlink refusal (AC-015)
# ---------------------------------------------------------------------------

# (a) A repetition into an i05_bundle_dir already holding a prior
# repetition's residue: the four producer-owned entries are reset, an
# unrelated operator-placed file and directory survive untouched.
WORKDIR_R="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_R"' EXIT
mkdir -p "$WORKDIR_R/bin" "$WORKDIR_R/scratch" "$WORKDIR_R/i05/stages" "$WORKDIR_R/i05/custom"
echo '{"stale":true}' >"$WORKDIR_R/i05/bundle.json"
echo '{"stale":true}' >"$WORKDIR_R/i05/stages/99-stale.json"
echo '{"stale":true}' >"$WORKDIR_R/i05/access.jsonl"
echo "operator notes" >"$WORKDIR_R/i05/operator-notes.txt"
echo "operator file" >"$WORKDIR_R/i05/custom/keep.txt"
write_single_dispatch_shark "$WORKDIR_R/bin"
PATH="$WORKDIR_R/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115r --root ROOT-001 --scratch-root "$WORKDIR_R/scratch" \
	--mode contract --i05-bundle-dir "$WORKDIR_R/i05" --output "$WORKDIR_R/lifecycle.jsonl" \
	|| fail "(a) reset-repetition run failed"
[[ -f "$WORKDIR_R/i05/stages/99-stale.json" ]] && fail "(a) repetition-1 stage snapshot survived the reset"
# REQ-F-006 (T-E40-F12-002): access.jsonl now legitimately EXISTS after
# every run (created at run start) -- the reset property under test is that
# its STALE CONTENT is gone, not that the file itself is absent.
[[ -f "$WORKDIR_R/i05/access.jsonl" ]] || fail "(a) access.jsonl was not (re)created after the reset"
[[ -s "$WORKDIR_R/i05/access.jsonl" ]] && fail "(a) stale access.jsonl content survived the reset"
[[ -f "$WORKDIR_R/i05/operator-notes.txt" ]] || fail "(a) unrelated operator file was removed by the reset"
[[ "$(cat "$WORKDIR_R/i05/operator-notes.txt")" == "operator notes" ]] || fail "(a) unrelated operator file content changed"
[[ -f "$WORKDIR_R/i05/custom/keep.txt" ]] || fail "(a) unrelated operator directory was removed by the reset"
python3 -c "import json; b = json.load(open('$WORKDIR_R/i05/bundle.json')); assert b['stages'], 'bundle.json was not rewritten with real content'"
rm -rf "$WORKDIR_R"
trap - EXIT

# (b) A symlink at one of the four producer-owned entries fails the run
# before any write; the symlink target is left untouched.
WORKDIR_S="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_S"' EXIT
mkdir -p "$WORKDIR_S/bin" "$WORKDIR_S/scratch" "$WORKDIR_S/i05" "$WORKDIR_S/elsewhere"
echo "canary" >"$WORKDIR_S/elsewhere/canary.txt"
ln -s "$WORKDIR_S/elsewhere" "$WORKDIR_S/i05/stages"
write_single_dispatch_shark "$WORKDIR_S/bin"
set +e
PATH="$WORKDIR_S/bin:$PATH" SHARK_EVENTS="$WORKDIR_S/events.ndjson" \
	SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" "$RUNNER" \
	--scenario "$SCENARIO" --run-id tc115s --root ROOT-001 --scratch-root "$WORKDIR_S/scratch" \
	--mode contract --i05-bundle-dir "$WORKDIR_S/i05" --output "$WORKDIR_S/lifecycle.jsonl" \
	>/dev/null 2>"$WORKDIR_S/err"
rc=$?
set -e
[[ "$rc" -ne 0 ]] || fail "(b) symlinked producer-owned entry unexpectedly succeeded"
grep -q "symlink" "$WORKDIR_S/err" || fail "(b) symlink refusal was not named in stderr"
[[ -f "$WORKDIR_S/i05/bundle.json" ]] && fail "(b) bundle.json was written despite the symlink refusal"
[[ "$(cat "$WORKDIR_S/elsewhere/canary.txt")" == "canary" ]] || fail "(b) symlink target was modified despite the refusal"
[[ -f "$WORKDIR_S/events.ndjson" ]] && fail "(b) shark was invoked despite the symlink refusal (refusal must precede any write)"
rm -rf "$WORKDIR_S"
trap - EXIT

echo "TC-115 TC-016: pass (producer-owned reset scope + symlink refusal)"

# ---------------------------------------------------------------------------
# TC-007: stage_category closed-table exhaustive coverage (state-transition),
# per skills/quality/workflows/state-space-coverage.md. Exhaustive over the
# 14 real dispatchable phases (one representative (level, status) dispatch
# per phase, driven from the same frozen SHARK_BIN fixture used throughout
# this suite -- TC-002's declared seam), the 3 non-dispatchable phases
# (asserted by absence), the invalid-transition case (a fabricated,
# never-shipped phase name), the no-phase-at-all negative case, and the
# state-addition regression case against a REAL, unstubbed
# `shark admin workflow list` read across all six levels.
# ---------------------------------------------------------------------------

WORKDIR_7="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_7"' EXIT

python3 - "$RUNNER" "$SCENARIO" "$WORKFLOW_FIXTURE_DIR" "$WORKDIR_7" <<'TC007PY'
import hashlib
import json
import os
import subprocess
import sys
import tempfile

RUNNER, SCENARIO, WORKFLOW_DIR, ROOT = sys.argv[1:5]

STUB_TEMPLATE = '''#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    with open(os.environ["TC007_RESPONSE"]) as f:
        response = json.load(f)
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response))
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    with open(os.path.join(os.environ["TC007_WORKFLOW_DIR"], level + ".json")) as f:
        print(f.read())
elif args[:2] == ["claim", "ENTITY-001"]:
    print(\'{"session_id":"SID-001"}\')
elif args and args[0] == "heartbeat":
    print(\'{"ok":true}\')
elif args[:2] == ["status", "advance"]:
    print(\'{"advanced":true}\')
elif args and args[0] == "release":
    print(\'{"released":true}\')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
'''


def run_case(tag, entity_type, status, action, workflow_dir):
    workdir = tempfile.mkdtemp(dir=ROOT, prefix=f"{tag}-")
    bin_dir = os.path.join(workdir, "bin")
    os.makedirs(bin_dir)
    scratch = os.path.join(workdir, "scratch")
    os.makedirs(scratch)
    i05 = os.path.join(workdir, "i05")
    prompt = f"work {tag}\n"
    response = {
        "entity_key": "ENTITY-001", "entity_type": entity_type, "status": status,
        "action": action, "agent_type": "developer", "provider": "fixture",
        "model": "fixture-model", "effort": "medium", "prompt": prompt,
        "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes": len(prompt.encode()), "resolved_via": ["ROOT-001"],
        "unresolved_placeholders": [], "error": "" if action == "spawn_agent" else "terminal",
        "question_block": None, "current_responder": "",
    }
    if action != "spawn_agent":
        response = {"action": action, "entity_key": "ENTITY-001", "error": "terminal"}
    response_path = os.path.join(workdir, "response.json")
    with open(response_path, "w") as f:
        json.dump(response, f)
    shark_path = os.path.join(bin_dir, "shark")
    with open(shark_path, "w") as f:
        f.write(STUB_TEMPLATE)
    os.chmod(shark_path, 0o755)
    env = dict(os.environ)
    env["PATH"] = bin_dir + os.pathsep + env["PATH"]
    env["TC007_RESPONSE"] = response_path
    env["TC007_WORKFLOW_DIR"] = workflow_dir
    result = subprocess.run(
        [RUNNER, "--scenario", SCENARIO, "--run-id", tag, "--root", "ROOT-001",
         "--scratch-root", scratch, "--mode", "dry-run", "--i05-bundle-dir", i05,
         "--output", os.path.join(workdir, "lifecycle.jsonl")],
        env=env, capture_output=True, text=True,
    )
    return i05, result


# (level, status, entity_type, expected stage_category) -- one representative
# dispatch per real dispatchable phase, spec.md §2.4.
DISPATCHABLE_ROWS = [
    ("bug", "research", "bug", "discovery"),
    ("feature", "assessment", "feature", "discovery"),
    ("epic", "refinement", "epic", "specification"),
    ("feature", "specification", "feature", "specification"),
    ("epic", "design", "epic", "specification"),
    ("epic", "decomposition", "epic", "planning"),
    ("task", "draft", "task", "planning"),
    ("feature", "test_planning", "feature", "planning"),
    ("task", "development", "task", "code"),
    ("bug", "code_review", "bug", "review"),
    ("feature", "task_review", "feature", "review"),
    ("bug", "qa", "bug", "qa"),
    ("feature", "approval", "feature", "uat"),
    ("feature", "active", "feature", "code"),
]
assert len(DISPATCHABLE_ROWS) == 14, len(DISPATCHABLE_ROWS)

for level, status, entity_type, expected_category in DISPATCHABLE_ROWS:
    tag = f"tc007-{level}-{status}"
    i05, result = run_case(tag, entity_type, status, "spawn_agent", WORKFLOW_DIR)
    assert result.returncode == 0, f"{tag}: run failed: {result.stderr}"
    with open(os.path.join(i05, "bundle.json")) as f:
        bundle = json.load(f)
    assert len(bundle["stages"]) == 1, f"{tag}: expected exactly one stage, got {bundle['stages']}"
    stage = bundle["stages"][0]
    assert stage["stage_category"] == expected_category, (
        f"{tag}: expected stage_category={expected_category!r}, got {stage['stage_category']!r}"
    )
    with open(os.path.join(i05, stage["snapshot_path"])) as f:
        snapshot = json.load(f)
    assert snapshot["stage_category"] == expected_category
    assert snapshot["errors"] == [] or {e["kind"] for e in snapshot["errors"]} <= {"unmapped_provider"}, (
        f"{tag}: unexpected errors {snapshot['errors']}"
    )

print(f"TC-007: {len(DISPATCHABLE_ROWS)} dispatchable phases -> stage_category: pass")

# 3 non-dispatchable phases (blocked/paused/done): shark next never returns
# spawn_agent for them, so no snapshot is ever written -- asserted here by
# having the stub return a non-spawn terminal action, matching that reality,
# and checking the bundle's stages[] stays empty.
NON_DISPATCHABLE_ROWS = [
    ("task", "blocked", "task"),
    ("task", "on_hold", "task"),
    ("task", "completed", "task"),
]
for level, status, entity_type in NON_DISPATCHABLE_ROWS:
    tag = f"tc007-nondispatch-{status}"
    i05, result = run_case(tag, entity_type, status, "archive", WORKFLOW_DIR)
    with open(os.path.join(i05, "bundle.json")) as f:
        bundle = json.load(f)
    assert bundle["stages"] == [], f"{tag}: a non-dispatchable phase must never produce a stage snapshot"

print(f"TC-007: {len(NON_DISPATCHABLE_ROWS)} non-dispatchable phases -> no snapshot written: pass")

# Invalid-transition case: a fabricated, never-shipped phase name, and the
# no-phase-at-all negative case -- both fall into the identical unmapped
# branch (REQ-F-004), never a substituted category.
invalid_workflow_dir = tempfile.mkdtemp(dir=ROOT, prefix="tc007-invalid-workflow-")
with open(os.path.join(invalid_workflow_dir, "task.json"), "w") as f:
    json.dump({
        "levels": [{
            "level": "task",
            "statuses": [
                {"name": "invented-status", "phase": "never-shipped-phase-xyz"},
                {"name": "phaseless-status"},
                {"name": "development", "phase": "development"},
            ],
        }],
    }, f)

for tag, status in (("tc007-fabricated-phase", "invented-status"), ("tc007-no-phase", "phaseless-status")):
    i05, result = run_case(tag, "task", status, "spawn_agent", invalid_workflow_dir)
    assert result.returncode == 0, f"{tag}: run failed: {result.stderr}"
    with open(os.path.join(i05, "bundle.json")) as f:
        bundle = json.load(f)
    stage = bundle["stages"][0]
    assert stage.get("stage_category") is None, f"{tag}: a category must never be substituted: {stage}"
    with open(os.path.join(i05, stage["snapshot_path"])) as f:
        snapshot = json.load(f)
    assert "stage_category" not in snapshot, f"{tag}: snapshot must carry no stage_category key when unmapped"
    unknown_errors = [e for e in snapshot["errors"] if e["kind"] == "unknown_stage_category"]
    assert len(unknown_errors) == 1, f"{tag}: expected exactly one unknown_stage_category error: {snapshot['errors']}"
    assert unknown_errors[0]["status"] == status
    real_category_values = {"discovery", "specification", "planning", "code", "review", "qa", "uat", "shipping"}
    assert unknown_errors[0].get("phase") not in real_category_values

print("TC-007: invalid-transition (fabricated phase) + no-phase negative case: pass")

# Recovery case: the unmapped-phase dispatch does not abort the run --
# subsequent dispatches in the same run still get correctly categorized.
recovery_response_1 = {
    "entity_key": "ENTITY-001", "entity_type": "task", "status": "invented-status",
    "action": "spawn_agent", "agent_type": "developer", "provider": "fixture",
    "model": "fixture-model", "effort": "medium", "prompt": "first\n",
    "prompt_sha256": hashlib.sha256(b"first\n").hexdigest(), "prompt_bytes": 6,
    "resolved_via": ["ROOT-001"], "unresolved_placeholders": [], "error": "",
    "question_block": None, "current_responder": "",
}
recovery_dir = tempfile.mkdtemp(dir=ROOT, prefix="tc007-recovery-")
bin_dir = os.path.join(recovery_dir, "bin")
os.makedirs(bin_dir)
scratch = os.path.join(recovery_dir, "scratch")
os.makedirs(scratch)
i05 = os.path.join(recovery_dir, "i05")
shark_path = os.path.join(bin_dir, "shark")
with open(shark_path, "w") as f:
    f.write('''#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"task","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"ENTITY-BAD","entity_type":"task"},
                    {"entity_key":"ENTITY-GOOD","entity_type":"task"}]}))
elif args[0] == "next":
    key = args[1]
    status = "invented-status" if key == "ENTITY-BAD" else "development"
    prompt = "work " + key + "\\n"
    print(json.dumps({"entity_key":key,"entity_type":"task","status":status,
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}))
    open(args[args.index("--prompt-out") + 1], "wb").write(prompt.encode())
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["TC007_WORKFLOW_DIR"], args[3] + ".json")) as f:
        print(f.read())
elif args[0] == "claim": print(json.dumps({"session_id":"SID-" + args[1]}))
elif args[0] == "heartbeat": print('{"ok":true}')
elif args[:2] == ["status", "advance"]: print('{"advanced":true}')
elif args[0] == "release": print('{"released":true}')
else: raise SystemExit("unexpected shark argv: " + repr(args))
PY
''')
os.chmod(shark_path, 0o755)
env = dict(os.environ)
env["PATH"] = bin_dir + os.pathsep + env["PATH"]
env["TC007_WORKFLOW_DIR"] = invalid_workflow_dir
result = subprocess.run(
    [RUNNER, "--scenario", SCENARIO, "--run-id", "tc007-recovery", "--root", "ROOT-001",
     "--scratch-root", scratch, "--mode", "dry-run", "--i05-bundle-dir", i05,
     "--output", os.path.join(recovery_dir, "lifecycle.jsonl")],
    env=env, capture_output=True, text=True,
)
assert result.returncode == 0, f"recovery run failed: {result.stderr}"
with open(os.path.join(i05, "bundle.json")) as f:
    bundle = json.load(f)
assert len(bundle["stages"]) == 2, bundle["stages"]
by_key = {}
for stage in bundle["stages"]:
    with open(os.path.join(i05, stage["snapshot_path"])) as f:
        by_key[json.load(f)["entity"]["entity_key"]] = stage
assert by_key["ENTITY-BAD"].get("stage_category") is None
assert by_key["ENTITY-GOOD"]["stage_category"] == "code", "an unmapped-phase dispatch must not derail a later dispatch's categorization"

print("TC-007: recovery case (unmapped phase does not derail later dispatches): pass")

# State-addition regression case: stage-category-map.yaml's declared phase
# set exactly equals the LIVE union of phases across all six levels -- a
# REAL, unstubbed `shark admin workflow list` read (never the frozen
# fixtures above, which would make this check self-consistent with itself
# rather than with the live binary).
TC007PY
echo "TC-115 TC-007 (phases + recovery): pass so far -- running state-addition regression against the real binary"

python3 - "$REPO_ROOT/bin/shark" "$STAGE_CATEGORY_MAP" <<'PY'
import subprocess
import sys

import yaml

shark_bin, map_path = sys.argv[1:3]

with open(map_path) as f:
    declared_phases = set(yaml.safe_load(f)["phases"].keys())

live_phases = set()
for level in ("task", "feature", "epic", "bug", "change", "tech_debt"):
    result = subprocess.run(
        [shark_bin, "admin", "workflow", "list", level, "--json"],
        capture_output=True, text=True, check=True,
    )
    import json
    display = json.loads(result.stdout)
    for entry in display["levels"]:
        if entry["level"] != level:
            continue
        for status in entry["statuses"]:
            phase = status.get("phase", "")
            if phase:
                live_phases.add(phase)

assert declared_phases == live_phases, (
    f"stage-category-map.yaml phases {declared_phases} != live union {live_phases} "
    f"(missing: {live_phases - declared_phases}, extra: {declared_phases - live_phases})"
)
print(f"TC-007: state-addition regression -- {len(live_phases)} live phases match stage-category-map.yaml exactly: pass")
PY

rm -rf "$WORKDIR_7"
trap - EXIT

echo "TC-115 TC-007: pass (stage_category closed-table exhaustive coverage)"

# ---------------------------------------------------------------------------
# TC-010 partition 2: `--grant-access inject-tests` (real, unmodified
# verify-stage-evidence.sh + real bench/adapters/python/adapter.sh) appends
# to the PRODUCER's own access.jsonl without recreating it.
# ---------------------------------------------------------------------------

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
ACCESS_FIXTURE="$SCRIPTS_DIR/testdata/evidence/access"
ORACLE_TEST_SRC="$ACCESS_FIXTURE/evaluator-only/oracle_tests/test_oracle_case.py"
[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable"
[[ -x "$ADAPTER" ]] || fail "python adapter missing or not executable: $ADAPTER"
[[ -f "$ORACLE_TEST_SRC" ]] || fail "fixture missing oracle test file: $ORACLE_TEST_SRC"

WORKDIR_10="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_10"' EXIT
mkdir -p "$WORKDIR_10/bin" "$WORKDIR_10/scratch" "$WORKDIR_10/i05"
write_single_dispatch_shark "$WORKDIR_10/bin" "BUG-900"
PATH="$WORKDIR_10/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-research.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-010 --root ROOT-001 --scratch-root "$WORKDIR_10/scratch" \
	--mode dry-run --i05-bundle-dir "$WORKDIR_10/i05" --output "$WORKDIR_10/lifecycle.jsonl" \
	|| fail "TC-010 fixture run failed"

[[ -f "$WORKDIR_10/i05/access.jsonl" ]] || fail "TC-010 access.jsonl missing after the run"
[[ ! -s "$WORKDIR_10/i05/access.jsonl" ]] || fail "TC-010 access.jsonl is not empty before any grant"

cp -r "$ACCESS_FIXTURE/worker-checkout" "$WORKDIR_10/checkout"

"$GUARD" "$WORKDIR_10/i05" --grant-access inject-tests --accessor execution_oracle \
	--adapter "$ADAPTER" --checkout "$WORKDIR_10/checkout" --files "$ORACLE_TEST_SRC" \
	>/dev/null || fail "TC-010 --grant-access inject-tests was rejected against a completed run's bundle"

[[ -f "$WORKDIR_10/i05/access.jsonl" ]] || fail "TC-010 --grant-access must append to the existing file, not recreate it"
LINE_COUNT="$(wc -l <"$WORKDIR_10/i05/access.jsonl" | tr -d ' ')"
[[ "$LINE_COUNT" -eq 1 ]] || fail "TC-010 access.jsonl has $LINE_COUNT line(s) after one grant, want exactly 1"

rm -rf "$WORKDIR_10"
trap - EXIT

echo "TC-115 TC-010: pass (access.jsonl existence + append-only via real --grant-access)"

# ---------------------------------------------------------------------------
# TC-008: time-ledger interval-category closed-table coverage + reconciliation
# (AC-008) -- a real two-dispatch live run exercising, across its dispatches,
# `tool_and_test`/`queue_or_claim_wait`/`unclassified` (every dispatch,
# real driver-observed windows), `retry_or_backoff` (a stubbed `heartbeat`
# that fails exactly once then succeeds), `provider_active` (a stubbed
# adapter envelope reporting one explicit interval), and
# `replay_or_human_gate_wait` (a second dispatch whose worker returns a
# question). T-E40-F12-003, REQ-F-005.
# ---------------------------------------------------------------------------

WORKDIR_8="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_8"' EXIT
mkdir -p "$WORKDIR_8/bin" "$WORKDIR_8/scratch" "$WORKDIR_8/i05"

cat >"$WORKDIR_8/bin/shark" <<SHARK
#!/usr/bin/env bash
set -euo pipefail
python3 - "\$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
if os.environ.get("SHARK_EVENTS"):
    with open(os.environ["SHARK_EVENTS"], "a") as f:
        f.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"task","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"TASK-LEDGER","entity_type":"task"},
                    {"entity_key":"TASK-QUESTION","entity_type":"task"}]}, separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":"task","status":"development",
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], args[3] + ".json")) as f:
        print(f.read())
elif args[0] == "claim":
    print(json.dumps({"session_id": "SID-" + args[1]}))
elif args and args[0] == "heartbeat":
    # T-E40-F12-003/TC-008: the FIRST heartbeat call in the whole run fails
    # once (sentinel file absent -> create it and exit 1); every later
    # heartbeat call (the immediate retry, and any subsequent ones) succeeds.
    sentinel = os.environ["TC008_HEARTBEAT_SENTINEL"]
    if not os.path.exists(sentinel):
        open(sentinel, "w").close()
        raise SystemExit("stub heartbeat: induced failure for retry_or_backoff coverage")
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args[0] == "release":
    print('{"released":true}')
elif args[:2] == ["question", "create"]:
    print('{"key":"Q-TC008-001"}')
elif args[:2] == ["question", "configure-workflow"]:
    print('{"ok":true}')
elif args[0] == "link":
    print('{"ok":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_8/bin/shark"

# NOTE: the adapter body is a SEPARATE .py file, never a `python3 -
# <<'PY'` heredoc -- that form feeds the heredoc to python3 as its own
# SOURCE (the `-` argument), which consumes stdin for the program text and
# leaves nothing for this script's own `json.load(sys.stdin)` read of the
# real request `adapter_result()` pipes in.
cat >"$WORKDIR_8/adapter_body.py" <<'PYBODY'
import json, sys, time
request = json.load(sys.stdin)
key = request["entity_key"]
if key == "TASK-LEDGER":
    # Long enough for >=1 heartbeat at LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS
    # below, so the induced first-heartbeat failure (and its retry) fall
    # genuinely inside this dispatch's adapter window.
    time.sleep(0.2)
    print(json.dumps({
        "worker_id": "worker-ledger", "session_id": request["session_id"],
        "kind": "final", "recommended_outcome": "pass", "cost_usd": 0.0,
        "evidence": {"summary": "ledger fixture"},
        # T-E40-F12-003 envelope-placement contract (spec.md sec 2.5):
        # offsets are ns relative to the adapter subprocess's own spawn.
        "time_ledger": {"provider_active": [[20000000, 80000000]]},
    }))
elif key == "TASK-QUESTION":
    print(json.dumps({
        "worker_id": "worker-question", "session_id": request["session_id"],
        "kind": "question", "category": "scope",
        "question": "Is this in scope?", "why_blocking": "ambiguous requirement",
        "recommendation": "ask the human",
    }))
else:
    raise SystemExit("unexpected entity_key: " + key)
PYBODY
cat >"$WORKDIR_8/adapter.sh" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
exec python3 "$WORKDIR_8/adapter_body.py"
ADAPTER
chmod +x "$WORKDIR_8/adapter.sh"

set +e
PATH="$WORKDIR_8/bin:$PATH" SHARK_EVENTS="$WORKDIR_8/events.ndjson" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_8/adapter.sh" \
	TC008_HEARTBEAT_SENTINEL="$WORKDIR_8/heartbeat-failed-once" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.02 \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-008 --root ROOT-001 --scratch-root "$WORKDIR_8/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_8/i05" --output "$WORKDIR_8/lifecycle.jsonl" \
	>"$WORKDIR_8/runner.out" 2>"$WORKDIR_8/runner.err"
rc=$?
set -e
# The question handoff pauses the run (exit 1) after dispatch 2 -- expected,
# not a failure of this fixture.
[[ "$rc" -eq 1 ]] || { cat "$WORKDIR_8/runner.err" >&2; fail "TC-008 fixture run exited $rc, want 1 (question pause)"; }
[[ -f "$WORKDIR_8/heartbeat-failed-once" ]] || fail "TC-008 the induced heartbeat failure never fired"

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR_8/i05" >"$WORKDIR_8/verify.out" 2>"$WORKDIR_8/verify.err" \
	|| fail "TC-008 verify-stage-evidence.sh rejected a real run's time_ledger: $(cat "$WORKDIR_8/verify.err")"
grep -qE "ledger_overlap|ledger_window_escape|ledger_non_reconciling" "$WORKDIR_8/verify.out" "$WORKDIR_8/verify.err" \
	&& fail "TC-008 a ledger rejection reason leaked into an accepted run: $(cat "$WORKDIR_8/verify.out")"

python3 - "$WORKDIR_8/i05" <<'PY'
import json, sys

i05_dir = sys.argv[1]
with open(f"{i05_dir}/bundle.json") as f:
    bundle = json.load(f)
assert len(bundle["stages"]) == 2, bundle["stages"]

snapshots = {}
for stage in bundle["stages"]:
    with open(f"{i05_dir}/{stage['snapshot_path']}") as f:
        snapshot = json.load(f)
    snapshots[snapshot["entity"]["entity_key"]] = snapshot

ledger_book = snapshots["TASK-LEDGER"]["time_ledger"]
ledger_question = snapshots["TASK-QUESTION"]["time_ledger"]

# State-addition regression case (T-E40-F12-003, corrected from codex
# red-team per test-plan.md TC-008): the producer's writer must only ever
# emit these SIX literal names -- pinned here, never read dynamically from
# i05-schema.yaml, so a real 7th-category addition cannot pass this check
# silently just because it also edited the schema.
FIXED_SIX = {"provider_active", "tool_and_test", "queue_or_claim_wait",
             "replay_or_human_gate_wait", "retry_or_backoff", "unclassified"}
for ledger in (ledger_book, ledger_question):
    assert set(ledger["intervals"].keys()) == FIXED_SIX, ledger["intervals"].keys()

# Closed-table coverage: every one of the six categories genuinely observed
# at least once across this real run's two dispatches.
observed = {name for ledger in (ledger_book, ledger_question)
            for name, spans in ledger["intervals"].items() if spans}
assert observed == FIXED_SIX, f"missing categories in a real run: {FIXED_SIX - observed}"

# tool_and_test / queue_or_claim_wait: every dispatch, real driver-observed
# windows (candidate_identity()/refresh_candidate(), claim/release).
for ledger in (ledger_book, ledger_question):
    assert len(ledger["intervals"]["tool_and_test"]) == 2, ledger["intervals"]["tool_and_test"]
    assert len(ledger["intervals"]["queue_or_claim_wait"]) == 2, ledger["intervals"]["queue_or_claim_wait"]

# retry_or_backoff / provider_active: only TASK-LEDGER.
assert ledger_book["intervals"]["retry_or_backoff"], "expected a recorded heartbeat retry window"
assert ledger_book["intervals"]["provider_active"], "expected the stubbed provider_active interval"
assert not ledger_question["intervals"]["retry_or_backoff"]
assert not ledger_question["intervals"]["provider_active"]

# replay_or_human_gate_wait: only TASK-QUESTION.
assert ledger_question["intervals"]["replay_or_human_gate_wait"], "expected the question-handoff window"
assert not ledger_book["intervals"]["replay_or_human_gate_wait"]

# Boundary case: the ledger's own last covered span always touches
# stage_end exactly (the gap-fill invariant reconcile_time_ledger() relies
# on) -- a half-open interval whose end lands exactly at stage_end.
for ledger in (ledger_book, ledger_question):
    all_ends = [span[1] for spans in ledger["intervals"].values() for span in spans]
    assert max(all_ends) == ledger["stage_end"], (max(all_ends), ledger["stage_end"])

print("TC-115 TC-008 (real run): six-category coverage + reconciliation: pass")
PY

echo "TC-115 TC-008 (real run): pass"

# ---------------------------------------------------------------------------
# TC-008 (continued): boundary/invalid-transition/recovery cases, all
# constructed from the SAME real bundle above by mutating a COPY of one
# stage snapshot's time_ledger -- never verify-stage-evidence.sh itself
# (unmodified, REQ-F-016). verify-stage-evidence.sh does not recompute
# snapshot_digest (that is replay-stage-evidence.sh's job, per TC-005's own
# note), so mutating a snapshot's time_ledger in place is safe here.
# ---------------------------------------------------------------------------

mutate_ledger_copy() {
	# mutate_ledger_copy <dest_bundle_dir> <mutator.py>: copies the real
	# TASK-LEDGER bundle, applies the given Python mutator (reads/writes the
	# TASK-LEDGER snapshot file named by env MUTATE_SNAPSHOT), returns via
	# stdout the mutated bundle dir.
	local dest="$1" mutator="$2"
	cp -r "$WORKDIR_8/i05" "$dest"
	local ledger_ordinal
	ledger_ordinal="$(python3 -c "
import json
b = json.load(open('$dest/bundle.json'))
for s in b['stages']:
    with open('$dest/' + s['snapshot_path']) as f:
        if json.load(f)['entity']['entity_key'] == 'TASK-LEDGER':
            print(s['snapshot_path']); break
")"
	MUTATE_SNAPSHOT="$dest/$ledger_ordinal" python3 -c "$mutator"
}

# (i) residual of exactly reconciliation_epsilon_ns (1,000,000 ns) is
# accepted. Constructed from synthetic, known-valid interval boundaries --
# NOT by shrinking whichever real captured interval happens to have the
# latest end. A real captured interval (e.g. a claim/release round trip
# under queue_or_claim_wait) can be sub-millisecond; shrinking IT by epsilon
# can invert it into start >= end, tripping the unrelated "not a valid
# half-open span" ScriptError before this case ever reaches the
# residual-tolerance check it exists to prove -- exactly the
# invalid-boundary-test-interval-mutation defect class. The stage window
# itself (stage_end - stage_start) is always far larger than epsilon (the
# fixture's adapter sleeps 0.2s per dispatch), so replacing every captured
# span with a single synthetic interval anchored to the window's own real
# boundaries is reproducible regardless of real execution speed/load.
BOUNDARY_EPS="$WORKDIR_8/boundary-eps"
mutate_ledger_copy "$BOUNDARY_EPS" "
import json, os
path = os.environ['MUTATE_SNAPSHOT']
snap = json.load(open(path))
ledger = snap['time_ledger']
stage_start = ledger['stage_start']
stage_end = ledger['stage_end']
epsilon = ledger['reconciliation_epsilon_ns']
end = stage_end - epsilon
assert end > stage_start, ('synthetic interval would be invalid', stage_start, stage_end, end)
ledger['intervals'] = {cat: [] for cat in ledger['intervals']}
ledger['intervals']['unclassified'] = [[stage_start, end]]
json.dump(snap, open(path, 'w'), sort_keys=True, separators=(',', ':'))
"
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$BOUNDARY_EPS" >/dev/null 2>"$BOUNDARY_EPS.err" \
	|| fail "TC-008 boundary: a residual of exactly reconciliation_epsilon_ns was rejected: $(cat "$BOUNDARY_EPS.err")"

# (ii) residual of epsilon + 1 is rejected, naming the magnitude. Same
# synthetic-boundary construction as (i), one ns narrower.
BOUNDARY_OVER="$WORKDIR_8/boundary-over"
mutate_ledger_copy "$BOUNDARY_OVER" "
import json, os
path = os.environ['MUTATE_SNAPSHOT']
snap = json.load(open(path))
ledger = snap['time_ledger']
stage_start = ledger['stage_start']
stage_end = ledger['stage_end']
epsilon = ledger['reconciliation_epsilon_ns']
end = stage_end - epsilon - 1
assert end > stage_start, ('synthetic interval would be invalid', stage_start, stage_end, end)
ledger['intervals'] = {cat: [] for cat in ledger['intervals']}
ledger['intervals']['unclassified'] = [[stage_start, end]]
json.dump(snap, open(path, 'w'), sort_keys=True, separators=(',', ':'))
"
set +e
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$BOUNDARY_OVER" >"$BOUNDARY_OVER.out" 2>"$BOUNDARY_OVER.err"
rc=$?
set -e
[[ "$rc" -ne 0 ]] || fail "TC-008 boundary: a residual of epsilon+1 was accepted"
grep -q "ledger_non_reconciling" "$BOUNDARY_OVER.out" "$BOUNDARY_OVER.err" \
	|| fail "TC-008 boundary: epsilon+1 residual was not reported as ledger_non_reconciling"

# Invalid-transition case: a 7th, invented interval_category name is
# rejected by the schema-driven closed vocabulary check (unmodified
# i05-schema.yaml/verify-stage-evidence.sh).
INVALID_CAT="$WORKDIR_8/invalid-category"
mutate_ledger_copy "$INVALID_CAT" "
import json, os
path = os.environ['MUTATE_SNAPSHOT']
snap = json.load(open(path))
ledger = snap['time_ledger']
ledger['intervals']['fabricated_seventh_category'] = ledger['intervals'].pop('unclassified')
json.dump(snap, open(path, 'w'), sort_keys=True, separators=(',', ':'))
"
set +e
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$INVALID_CAT" >"$INVALID_CAT.out" 2>"$INVALID_CAT.err"
rc=$?
set -e
[[ "$rc" -ne 0 ]] || fail "TC-008 invalid-transition: a fabricated 7th category was accepted"
grep -q "unknown_interval_category" "$INVALID_CAT.out" "$INVALID_CAT.err" \
	|| fail "TC-008 invalid-transition: fabricated category was not reported as unknown_interval_category"

echo "TC-115 TC-008 (boundary + invalid-transition): pass"

# Recovery case: one stage's ledger being invalid does not make an
# UNRELATED stage's own, independently-valid ledger fail (or an invalid
# one falsely pass) -- proven by evaluating each stage's snapshot in
# isolation (the real validator's own architecture stops at the first
# violation within one invocation, sorted by dispatch_ordinal -- see header
# comment above -- so "does not cascade" is proven by isolating each stage
# into its own bundle.json index and observing its OWN, unchanged verdict).
RECOVERY_BAD_ONLY="$WORKDIR_8/recovery-bad-only"
cp -r "$INVALID_CAT" "$RECOVERY_BAD_ONLY"
RECOVERY_GOOD_ONLY="$WORKDIR_8/recovery-good-only"
cp -r "$INVALID_CAT" "$RECOVERY_GOOD_ONLY"
python3 -c "
import json

def keep_only(bundle_dir, keep_key):
    bundle_path = bundle_dir + '/bundle.json'
    bundle = json.load(open(bundle_path))
    filtered = []
    for stage in bundle['stages']:
        with open(bundle_dir + '/' + stage['snapshot_path']) as f:
            snapshot = json.load(f)
        if snapshot['entity']['entity_key'] == keep_key:
            filtered.append(stage)
    bundle['stages'] = filtered
    json.dump(bundle, open(bundle_path, 'w'), sort_keys=True, separators=(',', ':'))

keep_only('$RECOVERY_BAD_ONLY', 'TASK-LEDGER')
keep_only('$RECOVERY_GOOD_ONLY', 'TASK-QUESTION')
"
set +e
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$RECOVERY_BAD_ONLY" >/dev/null 2>"$RECOVERY_BAD_ONLY.err"
bad_rc=$?
set -e
[[ "$bad_rc" -ne 0 ]] || fail "TC-008 recovery: the mutated TASK-LEDGER stage in isolation unexpectedly passed"
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$RECOVERY_GOOD_ONLY" >/dev/null 2>"$RECOVERY_GOOD_ONLY.err" \
	|| fail "TC-008 recovery: TASK-QUESTION's own, untouched ledger was rejected merely because TASK-LEDGER's (elsewhere) is invalid: $(cat "$RECOVERY_GOOD_ONLY.err")"

echo "TC-115 TC-008 (recovery -- one bad ledger does not cascade to an unrelated stage): pass"

rm -rf "$WORKDIR_8"
trap - EXIT

echo "TC-115 TC-008: pass (time-ledger closed-table coverage + reconciliation)"

# ---------------------------------------------------------------------------
# TC-009: `provider_active` sourcing partitions -- empty vs. populated
# envelope (AC-009).
# ---------------------------------------------------------------------------

run_tc009_case() {
	# run_tc009_case <workdir> <adapter_envelope_extra_json>
	local workdir="$1" extra="$2"
	mkdir -p "$workdir/bin" "$workdir/scratch" "$workdir/i05"
	write_single_dispatch_shark "$workdir/bin"
	cat >"$workdir/adapter.sh" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
python3 -c 'import json,sys; request=json.load(sys.stdin); print(json.dumps({"worker_id":"w","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":0.0,"evidence":{"summary":"tc009"}$extra}))'
ADAPTER
	chmod +x "$workdir/adapter.sh"
	PATH="$workdir/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
		SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$workdir/adapter.sh" \
		"$RUNNER" --scenario "$SCENARIO" --run-id "tc009-$(basename "$workdir")" --root ROOT-001 \
		--scratch-root "$workdir/scratch" --mode live --i05-bundle-dir "$workdir/i05" \
		--output "$workdir/lifecycle.jsonl" || fail "TC-009 fixture run failed ($workdir)"
}

# Partition 1: envelope reports no provider_active intervals at all.
WORKDIR_9A="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_9A"' EXIT
run_tc009_case "$WORKDIR_9A" ""
python3 -c "
import json
b = json.load(open('$WORKDIR_9A/i05/bundle.json'))
with open('$WORKDIR_9A/i05/' + b['stages'][0]['snapshot_path']) as f:
    ledger = json.load(f)['time_ledger']
assert ledger['intervals']['provider_active'] == [], ledger['intervals']['provider_active']
assert ledger['intervals']['unclassified'], 'expected the whole adapter window under unclassified'
"
rm -rf "$WORKDIR_9A"
trap - EXIT

# Partition 2: envelope reports one explicit provider_active interval.
WORKDIR_9B="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_9B"' EXIT
run_tc009_case "$WORKDIR_9B" ',"time_ledger":{"provider_active":[[1000000,3000000]]}'
python3 -c "
import json
b = json.load(open('$WORKDIR_9B/i05/bundle.json'))
with open('$WORKDIR_9B/i05/' + b['stages'][0]['snapshot_path']) as f:
    ledger = json.load(f)['time_ledger']
provider = ledger['intervals']['provider_active']
assert len(provider) == 1, provider
span = provider[0]
assert span[1] - span[0] == 2_000_000, span
# No leakage: the recorded provider_active span never appears verbatim
# inside any other category's list.
for name, spans in ledger['intervals'].items():
    if name == 'provider_active':
        continue
    assert span not in spans, f'{span} leaked into {name}'
"
rm -rf "$WORKDIR_9B"
trap - EXIT

# Negative case: envelope reports an interval extending far past the
# adapter's own real window -- must be clipped, never accepted verbatim.
WORKDIR_9C="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_9C"' EXIT
run_tc009_case "$WORKDIR_9C" ',"time_ledger":{"provider_active":[[0,999999999999]]}'
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR_9C/i05" >/dev/null 2>"$WORKDIR_9C/verify.err" \
	|| fail "TC-009 negative: an adversarial out-of-window provider_active interval was not clipped/rejected cleanly: $(cat "$WORKDIR_9C/verify.err")"
python3 -c "
import json
b = json.load(open('$WORKDIR_9C/i05/bundle.json'))
with open('$WORKDIR_9C/i05/' + b['stages'][0]['snapshot_path']) as f:
    ledger = json.load(f)['time_ledger']
for span in ledger['intervals']['provider_active']:
    assert span[1] <= ledger['stage_end'], f'provider_active span {span} escaped stage_end {ledger[\"stage_end\"]}'
"
rm -rf "$WORKDIR_9C"
trap - EXIT

echo "TC-115 TC-009: pass (provider_active sourcing partitions)"

# ---------------------------------------------------------------------------
# TC-011: `transcripts/` materialization + real `retain_pair` consumption
# (AC-011).
# ---------------------------------------------------------------------------

RETAIN_PAIR="$SCRIPTS_DIR/lib/retain_pair"
I07_FIXTURE="$REPO_ROOT/tests/contracts/testdata/e40_i07/valid/complete.jsonl"
I08_FIXTURE="$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl"
[[ -x "$RETAIN_PAIR" ]] || fail "bench/scripts/lib/retain_pair missing or not executable"
[[ -f "$I07_FIXTURE" ]] || fail "committed valid I-07 fixture missing: $I07_FIXTURE"
[[ -f "$I08_FIXTURE" ]] || fail "committed valid I-08 fixture missing: $I08_FIXTURE"
command -v jq >/dev/null 2>&1 || fail "jq not found on PATH"

WORKDIR_11="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_11"' EXIT
mkdir -p "$WORKDIR_11/bin" "$WORKDIR_11/scratch" "$WORKDIR_11/i05"
write_single_dispatch_shark "$WORKDIR_11/bin"
write_adapter "$WORKDIR_11/adapter.sh" "worker-011"
PATH="$WORKDIR_11/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_11/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-011 --root ROOT-001 --scratch-root "$WORKDIR_11/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_11/i05" --output "$WORKDIR_11/lifecycle.jsonl" \
	|| fail "TC-011 fixture run failed"

[[ -d "$WORKDIR_11/i05/transcripts" ]] || fail "TC-011 transcripts/ does not exist after a real run"
[[ ! -L "$WORKDIR_11/i05/transcripts" ]] || fail "TC-011 transcripts/ is a symlink, not a real directory"
[[ -n "$(ls -A "$WORKDIR_11/i05/transcripts")" ]] || fail "TC-011 transcripts/ has no artifact for the one dispatch"

# retain_pair's own positional-argument shape (tc082's build_golden()):
# <scenario_id> <rep> <package.yaml> <lifecycle.jsonl> <evaluation.jsonl>
# <entity_history.json> <i05_bundle_dir> <dest>.
EVAL_SOURCE="$WORKDIR_11/evaluation.jsonl"
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$I08_FIXTURE" >"$EVAL_SOURCE"
python3 -c "
import json
path = '$EVAL_SOURCE'
obj = json.load(open(path, encoding='utf-8'))
with open(path, 'w', encoding='utf-8') as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(',', ':')) + '\n')
"
ENTITY_HISTORY_SOURCE="$WORKDIR_11/entity-history.json"
python3 -c "
import json
obj = {'entity_type': 'task', 'entity_key': 'ROOT-TC115-011', 'history': [{'timestamp': '2026-01-01T00:00:00Z', 'old_status': 'todo', 'new_status': 'in_progress', 'agent': 'tc115-011-fixture'}], 'total': 1}
with open('$ENTITY_HISTORY_SOURCE', 'w', encoding='utf-8') as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(',', ':')) + '\n')
"
# retain_pair's own naming convention (evaluate-lifecycle.sh's
# "<evaluation.jsonl>.oracle.json" sidecar, UAT-R3-01): required alongside
# evaluation.jsonl or retain_pair refuses the whole pair.
python3 -c "
import json
obj = {'held_back': True, 'observed_result': 'pass'}
with open('$EVAL_SOURCE.oracle.json', 'w', encoding='utf-8') as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(',', ':')) + '\n')
"

RETAIN_DEST="$WORKDIR_11/retained"
mkdir -p "$RETAIN_DEST"
python3 "$RETAIN_PAIR" "py-bug-due-date-boundary" "1" "$SCENARIO" \
	"$WORKDIR_11/lifecycle.jsonl" "$EVAL_SOURCE" "$ENTITY_HISTORY_SOURCE" "$WORKDIR_11/i05" "$RETAIN_DEST" \
	>"$WORKDIR_11/retain.out" 2>"$WORKDIR_11/retain.err" \
	|| fail "TC-011 retain_pair rejected a real bundle carrying a real transcripts/: $(cat "$WORKDIR_11/retain.err")"
[[ -f "$RETAIN_DEST/manifest.json" ]] || fail "TC-011 retain_pair did not write manifest.json"
python3 -c "
import json
manifest = json.load(open('$RETAIN_DEST/manifest.json'))
for name in ('evidence', 'transcripts'):
    entry = manifest['artifacts'][name]
    digest = entry['sha256']
    assert isinstance(digest, str) and len(digest) == 64, f'{name} sha256 is not a real 64-hex digest: {digest!r}'
"

# Negative case (Finding 6's exact defect class): deleting transcripts/ from
# an otherwise-real produced bundle makes retain_pair refuse the WHOLE pair.
NO_TRANSCRIPTS_BUNDLE="$WORKDIR_11/i05-no-transcripts"
cp -r "$WORKDIR_11/i05" "$NO_TRANSCRIPTS_BUNDLE"
rm -rf "$NO_TRANSCRIPTS_BUNDLE/transcripts"
RETAIN_DEST_NEG="$WORKDIR_11/retained-negative"
mkdir -p "$RETAIN_DEST_NEG"
set +e
python3 "$RETAIN_PAIR" "py-bug-due-date-boundary" "1" "$SCENARIO" \
	"$WORKDIR_11/lifecycle.jsonl" "$EVAL_SOURCE" "$ENTITY_HISTORY_SOURCE" "$NO_TRANSCRIPTS_BUNDLE" "$RETAIN_DEST_NEG" \
	>"$WORKDIR_11/retain-negative.out" 2>"$WORKDIR_11/retain-negative.err"
rc=$?
set -e
[[ "$rc" -ne 0 ]] || fail "TC-011 negative: retain_pair accepted a bundle with no transcripts/"
[[ ! -f "$RETAIN_DEST_NEG/manifest.json" ]] || fail "TC-011 negative: manifest.json was written despite the missing transcripts/ source"

rm -rf "$WORKDIR_11"
trap - EXIT

echo "TC-115 TC-011: pass (transcripts/ materialization + real retain_pair consumption)"

# ---------------------------------------------------------------------------
# TC-012: three-root triad with a real `agent_fixture_checkout` (AC-012).
# ---------------------------------------------------------------------------

WORKDIR_12="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_12"' EXIT
mkdir -p "$WORKDIR_12/bin" "$WORKDIR_12/scratch" "$WORKDIR_12/i05"
write_single_dispatch_shark "$WORKDIR_12/bin"
write_adapter "$WORKDIR_12/adapter.sh" "worker-012"
# TC-012's CWD dependency is explicit, not inherited: candidate_identity()
# hard-assumes CWD is the Shark repo root (it runs real git commands there),
# which is also the base `scenario_identity()`'s CWD-relative
# `fixture.submodule_path` resolution relies on for `agent_fixture_checkout`
# -- both must run from $REPO_ROOT regardless of where this test file itself
# is invoked from.
( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_12/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_12/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-012 --root ROOT-001 --scratch-root "$WORKDIR_12/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_12/i05" --output "$WORKDIR_12/lifecycle.jsonl" ) \
	|| fail "TC-012 fixture run failed"

python3 -c "
import json, os
bundle = json.load(open('$WORKDIR_12/i05/bundle.json'))
roots = bundle['roots']
for name in ('agent_fixture_checkout', 'scratch_shark_project', 'evaluator_only'):
    assert name in roots, f'roots missing {name}'
    assert os.path.isabs(roots[name]), f'{name} is not an absolute path: {roots[name]!r}'
checkout = roots['agent_fixture_checkout']
assert os.path.isdir(checkout), f'agent_fixture_checkout is not a real, present directory: {checkout!r}'
paths = list(roots.values())
for i, a in enumerate(paths):
    for j, b in enumerate(paths):
        if i == j:
            continue
        # Pairwise non-nested: a's own path components are never a prefix
        # of b's (checked over every ordered pair, so both directions).
        a_parts = os.path.normpath(a).split(os.sep)
        b_parts = os.path.normpath(b).split(os.sep)
        assert a_parts != b_parts[: len(a_parts)], f'{b!r} is nested under {a!r}'
"

EVALUATOR="$SCRIPTS_DIR/evaluate-lifecycle.sh"
[[ -x "$EVALUATOR" ]] || fail "evaluate-lifecycle.sh missing or not executable"
EVAL_OUT_POS="$WORKDIR_12/evaluation-positive.json"
set +e
"$EVALUATOR" --i05 "$WORKDIR_12/i05" --i07 "$WORKDIR_12/lifecycle.jsonl" --scenario "$SCENARIO" --output "$EVAL_OUT_POS" \
	>/dev/null 2>"$WORKDIR_12/eval-positive.err"
set -e
[[ -f "$EVAL_OUT_POS" ]] || fail "TC-012 evaluate-lifecycle.sh did not write an output record: $(cat "$WORKDIR_12/eval-positive.err")"
python3 -c "
import json
record = json.load(open('$EVAL_OUT_POS'))
reasons = record['eligibility']['invalidity_reasons']
missing_oracle = [r for r in reasons if r['code'] == 'missing_oracle']
assert not missing_oracle, f'a real, present agent_fixture_checkout still produced missing_oracle: {missing_oracle}'
"

# Negative case: a mutated bundle whose agent_fixture_checkout does not
# exist reproduces the pre-fix defect (run_oracle() silently skipping the
# held-back oracle).
MISSING_CHECKOUT_BUNDLE="$WORKDIR_12/i05-missing-checkout"
cp -r "$WORKDIR_12/i05" "$MISSING_CHECKOUT_BUNDLE"
python3 -c "
import json
path = '$MISSING_CHECKOUT_BUNDLE/bundle.json'
bundle = json.load(open(path))
bundle['roots']['agent_fixture_checkout'] = '/nonexistent/tc115-012/checkout'
json.dump(bundle, open(path, 'w'), sort_keys=True, separators=(',', ':'))
"
EVAL_OUT_NEG="$WORKDIR_12/evaluation-negative.json"
set +e
"$EVALUATOR" --i05 "$MISSING_CHECKOUT_BUNDLE" --i07 "$WORKDIR_12/lifecycle.jsonl" --scenario "$SCENARIO" --output "$EVAL_OUT_NEG" \
	>/dev/null 2>"$WORKDIR_12/eval-negative.err"
set -e
[[ -f "$EVAL_OUT_NEG" ]] || fail "TC-012 negative: evaluate-lifecycle.sh did not write an output record: $(cat "$WORKDIR_12/eval-negative.err")"
python3 -c "
import json
record = json.load(open('$EVAL_OUT_NEG'))
reasons = record['eligibility']['invalidity_reasons']
missing_oracle = [r for r in reasons if r['code'] == 'missing_oracle']
assert missing_oracle, 'a missing agent_fixture_checkout did not reproduce missing_oracle'
"

rm -rf "$WORKDIR_12"
trap - EXIT

echo "TC-115 TC-012: pass (three-root triad with a real agent_fixture_checkout)"

# ---------------------------------------------------------------------------
# TC-013 / TC-014: stop-outcome eligibility triad (T-E40-F12-004, REQ-F-009,
# spec.md sec 2.6's closed 11-row table, AC-013). The clean-terminal row is
# already fully asserted above (TC-002/TC-003's own bundle.json assertions:
# stop_outcome absent, publication_eligible true, ineligibility_reasons ==
# []) -- re-run once more here anyway so this suite's own six-row table
# (spec.md AC-013's exact scope) is self-contained. TC-013 covers the five
# rows spec.md scopes to end-to-end induction; TC-014 covers the remaining
# five via direct-unit invocation of the real, source-extracted
# stop_outcome_triad() -- run-lifecycle.sh is a single embedded-Python
# heredoc script with no importable module, so TC-014 uses the same
# source-extraction-and-exec() technique tc075_content-identity_x12_test.sh
# already established for content_digest().
# ---------------------------------------------------------------------------

WORKDIR_13="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_13"' EXIT

python3 - "$RUNNER" "$SCENARIO" "$WORKFLOW_FIXTURE_DIR" "$WORKDIR_13" \
	"$SCRIPTS_DIR/verify-stage-evidence.sh" "$SCRIPTS_DIR/run-lifecycle.sh" <<'TC013PY'
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

RUNNER, SCENARIO, WORKFLOW_DIR, ROOT, VERIFY, RUNNER_SOURCE_PATH = sys.argv[1:7]

# A single-dispatch (ROOT-001 -> ENTITY-001) shark stub, env-toggled to
# induce exactly one failure point per TC-013 row: TC013_CLAIM_FAILS makes
# `claim` exit non-zero (the "error" row -- must NOT be the initial `next`
# call, since that would leave stages[] empty and verify-stage-evidence.sh
# requires a non-empty stages[] array even on a stop_outcome-carrying
# bundle).
SHARK_STUB = '''#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    with open(os.environ["TC013_RESPONSE"]) as f:
        response = json.load(f)
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response))
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    with open(os.path.join(os.environ["TC013_WORKFLOW_DIR"], level + ".json")) as f:
        print(f.read())
elif args[:2] == ["claim", "ENTITY-001"]:
    if os.environ.get("TC013_CLAIM_FAILS"):
        sys.stderr.write("stub shark: induced claim failure\\n")
        raise SystemExit(1)
    print('{"session_id":"SID-013"}')
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
elif args[:2] == ["question", "create"]:
    print(json.dumps({"key": "Q-TC013-001"}))
elif args[:2] == ["question", "configure-workflow"]:
    print('{"ok":true}')
elif args and args[0] == "link":
    print('{"ok":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
'''

ADAPTER_BODY = '''
import json, sys, os
request = json.load(sys.stdin)
mode = os.environ.get("TC013_ADAPTER_MODE", "pass")
if mode == "fail":
    sys.stderr.write("stub adapter: induced worker_failure\\n")
    raise SystemExit(1)
base = {"worker_id": "worker-tc013", "session_id": request["session_id"], "cost_usd": 0.0}
if mode == "missing_outcome":
    print(json.dumps(dict(base, kind="final", evidence={})))
elif mode == "question":
    print(json.dumps(dict(base, kind="question", category="scope",
        question="Is this in scope?", why_blocking="ambiguous requirement",
        recommendation="ask the human")))
elif mode == "cost":
    print(json.dumps(dict(base, kind="final", recommended_outcome="pass", cost_usd=0.02, evidence={})))
else:
    print(json.dumps(dict(base, kind="final", recommended_outcome="pass", evidence={})))
'''


def run_case(tag, *, claim_fails=False, adapter_mode="pass", max_cost_usd=1.0):
    workdir = tempfile.mkdtemp(dir=ROOT, prefix=f"{tag}-")
    bin_dir = os.path.join(workdir, "bin")
    os.makedirs(bin_dir)
    scratch = os.path.join(workdir, "scratch")
    os.makedirs(scratch)
    i05 = os.path.join(workdir, "i05")
    prompt = f"work {tag}\n"
    response = {
        "entity_key": "ENTITY-001", "entity_type": "task", "status": "development",
        "action": "spawn_agent", "agent_type": "developer", "provider": "fixture",
        "model": "fixture-model", "effort": "medium", "prompt": prompt,
        "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes": len(prompt.encode()), "resolved_via": ["ROOT-001"],
        "unresolved_placeholders": [], "error": "", "question_block": None,
        "current_responder": "",
    }
    response_path = os.path.join(workdir, "response.json")
    with open(response_path, "w") as f:
        json.dump(response, f)
    shark_path = os.path.join(bin_dir, "shark")
    with open(shark_path, "w") as f:
        f.write(SHARK_STUB)
    os.chmod(shark_path, 0o755)
    adapter_body_path = os.path.join(workdir, "adapter_body.py")
    with open(adapter_body_path, "w") as f:
        f.write(ADAPTER_BODY)
    adapter_path = os.path.join(workdir, "adapter.sh")
    with open(adapter_path, "w") as f:
        f.write("#!/usr/bin/env bash\nset -euo pipefail\nexec python3 " + json.dumps(adapter_body_path) + "\n")
    os.chmod(adapter_path, 0o755)
    limits_path = os.path.join(workdir, "limits.yaml")
    with open(limits_path, "w") as f:
        f.write(f"max_cost_usd: {max_cost_usd}\nmax_wall_clock_seconds: 3600\nmax_generated_tasks: 100\n")
    env = dict(os.environ)
    env["PATH"] = bin_dir + os.pathsep + env["PATH"]
    env["TC013_RESPONSE"] = response_path
    env["TC013_WORKFLOW_DIR"] = WORKFLOW_DIR
    env["LIFECYCLE_ADAPTER"] = adapter_path
    env["TC013_ADAPTER_MODE"] = adapter_mode
    if claim_fails:
        env["TC013_CLAIM_FAILS"] = "1"
    else:
        env.pop("TC013_CLAIM_FAILS", None)
    output_path = os.path.join(workdir, "lifecycle.jsonl")
    result = subprocess.run(
        [RUNNER, "--scenario", SCENARIO, "--run-id", tag, "--root", "ROOT-001",
         "--scratch-root", scratch, "--mode", "live", "--i05-bundle-dir", i05,
         "--output", output_path, "--limits", limits_path],
        env=env, capture_output=True, text=True,
    )
    return i05, output_path, result


def load(i05, i07_path):
    with open(os.path.join(i05, "bundle.json")) as f:
        bundle = json.load(f)
    with open(i07_path) as f:
        i07 = json.loads(f.readline())
    return bundle, i07


def assert_verify_accepts(tag, i05):
    verify = subprocess.run([VERIFY, i05], capture_output=True, text=True)
    assert verify.returncode == 0, f"{tag}: verify-stage-evidence.sh rejected the bundle: {verify.stderr}"


# (tag, claim_fails, adapter_mode, max_cost_usd, expected stop_outcome
#  (None == clean), expected exit code, a substring the row's own
#  ineligibility_reasons[] entry must contain)
ROWS = [
    ("tc013-clean", False, "pass", 1.0, None, 0, None),
    ("tc013-resource-limit", False, "cost", 0.01, "resource_limit", 0, "max_cost_usd"),
    ("tc013-missing-outcome", False, "missing_outcome", 1.0, "missing_outcome", 1, "ENTITY-001"),
    ("tc013-error", True, "pass", 1.0, "error", 1, None),
    ("tc013-worker-failure", False, "fail", 1.0, "worker_failure", 1, None),
    ("tc013-pause", False, "question", 1.0, "pause", 1, "Q-TC013-001"),
]

clean_bundle_dir = None
for tag, claim_fails, adapter_mode, max_cost_usd, expected, expected_rc, reason_substr in ROWS:
    i05, output_path, result = run_case(
        tag, claim_fails=claim_fails, adapter_mode=adapter_mode, max_cost_usd=max_cost_usd,
    )
    assert result.returncode == expected_rc, f"{tag}: exit {result.returncode}, want {expected_rc}: {result.stderr}"
    bundle, i07 = load(i05, output_path)
    if expected is None:
        assert "stop_outcome" not in bundle, f"{tag}: unexpected stop_outcome {bundle.get('stop_outcome')!r}"
        assert bundle["publication_eligible"] is True, f"{tag}: publication_eligible {bundle['publication_eligible']!r}"
        assert bundle["ineligibility_reasons"] == [], f"{tag}: {bundle['ineligibility_reasons']}"
        assert i07["outcome"]["terminal"] == "complete", f"{tag}: I-07 terminal {i07['outcome']['terminal']!r}"
        clean_bundle_dir = i05
    else:
        assert bundle.get("stop_outcome") == expected, f"{tag}: stop_outcome={bundle.get('stop_outcome')!r}, want {expected!r}"
        assert bundle["publication_eligible"] is False, f"{tag}: publication_eligible {bundle['publication_eligible']!r}"
        assert bundle["ineligibility_reasons"], f"{tag}: ineligibility_reasons empty"
        # REQ-F-009's cross-record consistency: the bundle's stop_outcome
        # equals the same run's I-07 outcome.terminal, and the triad's
        # ineligibility_reasons[] is exactly that record's own outcome.reason
        # (every row's reason -- ceiling/entity/Question key/exception
        # message -- flows through the same local `reason` the I-07 record
        # itself is built from; never re-derived by the triad writer).
        assert i07["outcome"]["terminal"] == expected, (
            f"{tag}: bundle stop_outcome={expected!r} != I-07 outcome.terminal={i07['outcome']['terminal']!r}"
        )
        assert bundle["ineligibility_reasons"] == [i07["outcome"]["reason"]], (
            f"{tag}: ineligibility_reasons {bundle['ineligibility_reasons']} != [outcome.reason] {[i07['outcome']['reason']]}"
        )
        if reason_substr:
            assert any(reason_substr in r for r in bundle["ineligibility_reasons"]), (
                f"{tag}: {bundle['ineligibility_reasons']} missing {reason_substr!r}"
            )
    assert_verify_accepts(tag, i05)

print("TC-115 TC-013: pass (6-row end-to-end stop-outcome eligibility triad, I-05/I-07 cross-record consistency)")

# Invalid-transition case (state-transition technique requires it for a
# closed table): a stop_outcome mutated outside the 11-row closed set is
# rejected by verify-stage-evidence.sh's own closed vocabulary, constructed
# from a mutated copy of the real clean-run bundle produced above.
assert clean_bundle_dir, "no clean-run bundle captured for the invalid-transition case"
invalid_dir = tempfile.mkdtemp(dir=ROOT, prefix="tc013-invalid-")
shutil.copytree(clean_bundle_dir, invalid_dir, dirs_exist_ok=True)
bundle_path = os.path.join(invalid_dir, "bundle.json")
with open(bundle_path) as f:
    mutated = json.load(f)
mutated["stop_outcome"] = "not-a-real-stop-outcome"
mutated["publication_eligible"] = False
mutated["ineligibility_reasons"] = ["fabricated for TC-013's invalid-transition case"]
with open(bundle_path, "w") as f:
    json.dump(mutated, f, sort_keys=True, separators=(",", ":"))
verify = subprocess.run([VERIFY, invalid_dir], capture_output=True, text=True)
assert verify.returncode != 0, "TC-013 invalid-transition: verify-stage-evidence.sh accepted a fabricated stop_outcome"
assert "not one of the closed" in verify.stderr, (
    f"TC-013 invalid-transition: rejection did not name the closed vocabulary: {verify.stderr}"
)

print("TC-115 TC-013 (invalid-transition): pass")

# ---------------------------------------------------------------------------
# TC-014: the remaining five stop-outcome rows, direct-unit, via the real
# stop_outcome_triad() extracted out of run-lifecycle.sh's own source text
# (tc075's technique) -- never a hand-retyped reimplementation.
# ---------------------------------------------------------------------------
source_text = open(RUNNER_SOURCE_PATH, encoding="utf-8").read()

const_match = re.search(r"\nSTOP_OUTCOMES = \{.*?\n\}\n", source_text, re.S)
assert const_match, "STOP_OUTCOMES constant not found in run-lifecycle.sh"
fn_match = re.search(r"\ndef stop_outcome_triad\(terminal, reason\):.*?\n(?=\S)", source_text, re.S)
assert fn_match, (
    "stop_outcome_triad() source not found in run-lifecycle.sh -- must be a single, "
    "named, top-level function (implementation-contract requirement, test-plan.md TC-014)"
)

namespace = {}
exec(compile(const_match.group(0), "run-lifecycle.sh::STOP_OUTCOMES", "exec"), namespace)  # noqa: S102
exec(compile(fn_match.group(0), "run-lifecycle.sh::stop_outcome_triad", "exec"), namespace)  # noqa: S102
stop_outcome_triad = namespace["stop_outcome_triad"]
real_stop_outcomes = namespace["STOP_OUTCOMES"]

TC014_ROWS = [
    ("lease_loss", "lease lost for entity ENTITY-007"),
    ("unresolved_gate", "blocked by gate G-42"),
    ("archive", "archived entity ROOT-001"),
    ("cancellation", "terminated by signal SIGTERM"),
    ("timeout", "exceeded window max_wall_clock_seconds"),
]
for terminal, reason in TC014_ROWS:
    stop_outcome, publication_eligible, ineligibility_reasons = stop_outcome_triad(terminal, reason)
    assert stop_outcome == terminal, f"{terminal}: stop_outcome={stop_outcome!r}"
    assert publication_eligible is False, f"{terminal}: publication_eligible={publication_eligible!r}"
    assert ineligibility_reasons == [reason], f"{terminal}: ineligibility_reasons={ineligibility_reasons!r}"

print("TC-115 TC-014: pass (5 direct-unit stop-outcome rows via source-extracted stop_outcome_triad())")

# State-addition regression: TC-013's 5 rows + TC-014's 5 rows + the
# clean-terminal row == exactly the 11 rows spec.md sec 2.6 declares, and
# this set equals run-lifecycle.sh's own real STOP_OUTCOMES constant plus
# "complete" -- read from source, never hard-coded here.
tc013_rows = {expected for _, _, _, _, expected, _, _ in ROWS if expected is not None}
tc014_rows = {terminal for terminal, _ in TC014_ROWS}
covered = tc013_rows | tc014_rows | {"complete"}
assert tc013_rows.isdisjoint(tc014_rows), f"a row is covered by both TC-013 and TC-014: {tc013_rows & tc014_rows}"
assert len(covered) == 11, f"expected exactly 11 rows, covered {sorted(covered)}"
assert covered == real_stop_outcomes | {"complete"}, (
    f"covered rows {sorted(covered)} != real STOP_OUTCOMES+complete {sorted(real_stop_outcomes | {'complete'})} "
    "-- a STOP_OUTCOMES addition without a matching spec.md sec 2.6 row and test row"
)

print("TC-115 TC-014 (state-addition regression): pass")
TC013PY

rm -rf "$WORKDIR_13"
trap - EXIT

# ---------------------------------------------------------------------------
# TC-015: partial evidence survives an aborted run (SIGTERM mid-run) --
# T-E40-F12-004, REQ-F-010/AC-014. Exercises the real signal path
# release_on_signal() installs, and this task's own addition to it (the
# finalize() call ahead of its SystemExit) -- without it, bundle.json's
# triad would stay stuck at its stale per-dispatch "not reached yet" state.
# ---------------------------------------------------------------------------

WORKDIR_15="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_15"' EXIT
mkdir -p "$WORKDIR_15/bin" "$WORKDIR_15/scratch" "$WORKDIR_15/i05"

cat >"$WORKDIR_15/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"task","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"ENTITY-A","entity_type":"task"},
                    {"entity_key":"ENTITY-B","entity_type":"task"}]}, separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":"task","status":"development",
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], args[3] + ".json")) as f:
        print(f.read())
elif args[0] == "claim":
    print(json.dumps({"session_id": "SID-" + args[1]}))
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_15/bin/shark"

# NOTE: a separate .py file, never a `python3 - <<'PY'` heredoc (TC-008's own
# established convention) -- that form would consume stdin for the program
# text and leave nothing for this script's own json.load(sys.stdin) read of
# the real request adapter_result() pipes in.
cat >"$WORKDIR_15/adapter_body.py" <<'PYBODY'
import json, sys, time
request = json.load(sys.stdin)
key = request["entity_key"]
if key == "ENTITY-A":
    print(json.dumps({"worker_id": "worker-a", "session_id": request["session_id"],
        "kind": "final", "recommended_outcome": "pass", "cost_usd": 0.0,
        "evidence": {"summary": "fixture"}}))
elif key == "ENTITY-B":
    # Long enough for the test harness to poll for ENTITY-A's already-written
    # snapshot and deliver SIGTERM to the runner while this dispatch's
    # adapter call is still blocking run-lifecycle.sh's own subprocess.run()
    # -- the real interruption window release_on_signal() exists for.
    time.sleep(3)
    print(json.dumps({"worker_id": "worker-b", "session_id": request["session_id"],
        "kind": "final", "recommended_outcome": "pass", "cost_usd": 0.0,
        "evidence": {"summary": "fixture"}}))
else:
    raise SystemExit("unexpected entity_key: " + key)
PYBODY
cat >"$WORKDIR_15/adapter.sh" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
exec python3 "$WORKDIR_15/adapter_body.py"
ADAPTER
chmod +x "$WORKDIR_15/adapter.sh"

SNAPSHOT_A="$WORKDIR_15/i05/stages/1-development.json"

PATH="$WORKDIR_15/bin:$PATH" SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	LIFECYCLE_ADAPTER="$WORKDIR_15/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-015 --root ROOT-001 --scratch-root "$WORKDIR_15/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_15/i05" --output "$WORKDIR_15/lifecycle.jsonl" \
	>"$WORKDIR_15/runner.out" 2>"$WORKDIR_15/runner.err" &
RUNNER_PID=$!

# Poll for dispatch 1's snapshot -- proof its incremental write already
# landed on disk -- before delivering the signal. Never a blind sleep: this
# is a real wait-for-condition, not a race against the fixture's own timing.
SNAPSHOT_SEEN=0
for _ in $(seq 1 100); do
	if [[ -f "$SNAPSHOT_A" ]]; then
		SNAPSHOT_SEEN=1
		break
	fi
	sleep 0.05
done
if [[ "$SNAPSHOT_SEEN" -ne 1 ]]; then
	kill -KILL "$RUNNER_PID" 2>/dev/null || true
	wait "$RUNNER_PID" 2>/dev/null || true
	fail "TC-015 dispatch 1's snapshot never appeared before timeout"
fi

CHECKPOINT="$WORKDIR_15/checkpoint-1-development.json"
cp "$SNAPSHOT_A" "$CHECKPOINT"

# run-lifecycle.sh's shebang is bash, and its own body invokes the embedded
# Python interpreter (which installs the real signal.signal() handlers) as a
# plain foreground child, never `exec`-replaced -- so the two are separate
# processes sharing no signal disposition. A signal delivered to $RUNNER_PID
# hits only the bash wrapper, whose own unhandled-SIGTERM default action
# (immediate death, never forwarded to its child) would kill the wrapper
# without ever reaching release_on_signal() at all. The real target -- the
# same one any supervisor delivering SIGTERM to "the python3 process running
# this dispatch loop" would hit -- is that embedded interpreter, so signal
# its PID directly.
PYTHON_PID="$(pgrep -P "$RUNNER_PID" | head -1)"
[[ -n "$PYTHON_PID" ]] || { kill -KILL "$RUNNER_PID" 2>/dev/null || true; wait "$RUNNER_PID" 2>/dev/null || true; fail "TC-015 could not find the embedded python3 child of $RUNNER_PID"; }

kill -TERM "$PYTHON_PID"
set +e
wait "$RUNNER_PID"
rc=$?
set -e
[[ "$rc" -eq 143 ]] || fail "TC-015 runner did not exit 128+SIGTERM (143), got $rc: $(cat "$WORKDIR_15/runner.err")"

[[ -f "$SNAPSHOT_A" ]] || fail "TC-015 dispatch 1's snapshot was removed by the abort"
cmp -s "$SNAPSHOT_A" "$CHECKPOINT" || fail "TC-015 dispatch 1's snapshot was truncated/rewritten by the abort (REQ-F-010 negative case)"

python3 -c "
import json
bundle = json.load(open('$WORKDIR_15/i05/bundle.json'))
assert len(bundle['stages']) == 1, f\"expected exactly 1 indexed stage, got {bundle['stages']}\"
assert bundle['stages'][0]['stage_key'] == 'development'
assert bundle.get('stop_outcome') == 'cancellation', f\"stop_outcome={bundle.get('stop_outcome')!r}\"
assert bundle['publication_eligible'] is False
assert bundle['ineligibility_reasons'], 'ineligibility_reasons is empty'
assert any('SIGTERM' in r for r in bundle['ineligibility_reasons']), bundle['ineligibility_reasons']
" || fail "TC-015 bundle.json triad assertions failed"

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR_15/i05" >"$WORKDIR_15/verify.out" 2>"$WORKDIR_15/verify.err" \
	|| fail "TC-015 verify-stage-evidence.sh rejected the partial bundle: $(cat "$WORKDIR_15/verify.err")"

rm -rf "$WORKDIR_15"
trap - EXIT

echo "TC-115 TC-015: pass (partial evidence survives SIGTERM mid-run)"

# ---------------------------------------------------------------------------
# TC-022: producer failure is loud (mid-run write failure) -- T-E40-F12-004,
# REQ-F-015/AC-020. Only `stages/` is made read-only (mode 0555) between two
# dispatches, isolating exactly one artifact-write failure while
# `i05_bundle_dir`'s top level (bundle.json) and access.jsonl/transcripts/
# stay writable, per test-plan.md TC-022's corrected fault-injection design
# (deterministic: the second dispatch's own adapter performs the chmod as a
# synchronous side effect, never a race against a concurrent writer).
# ---------------------------------------------------------------------------

WORKDIR_22="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_22"' EXIT
mkdir -p "$WORKDIR_22/bin" "$WORKDIR_22/scratch" "$WORKDIR_22/i05"

cat >"$WORKDIR_22/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"task","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"ENTITY-A","entity_type":"task"},
                    {"entity_key":"ENTITY-B","entity_type":"task"}]}, separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":"task","status":"development",
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], args[3] + ".json")) as f:
        print(f.read())
elif args[0] == "claim":
    print(json.dumps({"session_id": "SID-" + args[1]}))
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR_22/bin/shark"

cat >"$WORKDIR_22/adapter_body.py" <<'PYBODY'
import json, os, sys
request = json.load(sys.stdin)
key = request["entity_key"]
if key == "ENTITY-B":
    # TC-022 (REQ-F-015, AC-020): make ONLY stages/ read-only, between the
    # two dispatches -- the *next* stage-snapshot write (ENTITY-B's own)
    # fails while bundle.json/access.jsonl/transcripts/ stay writable.
    os.chmod(os.environ["TC022_STAGES_DIR"], 0o555)
print(json.dumps({"worker_id": "worker-" + key.lower(), "session_id": request["session_id"],
    "kind": "final", "recommended_outcome": "pass", "cost_usd": 0.0,
    "evidence": {"summary": "fixture"}}))
PYBODY
cat >"$WORKDIR_22/adapter.sh" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
exec python3 "$WORKDIR_22/adapter_body.py"
ADAPTER
chmod +x "$WORKDIR_22/adapter.sh"

STAGES_DIR="$WORKDIR_22/i05/stages"
SNAPSHOT_A="$STAGES_DIR/1-development.json"

set +e
PATH="$WORKDIR_22/bin:$PATH" SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" \
	LIFECYCLE_ADAPTER="$WORKDIR_22/adapter.sh" TC022_STAGES_DIR="$STAGES_DIR" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-022 --root ROOT-001 --scratch-root "$WORKDIR_22/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_22/i05" --output "$WORKDIR_22/lifecycle.jsonl" \
	>"$WORKDIR_22/runner.out" 2>"$WORKDIR_22/runner.err"
rc=$?
set -e

# Restore write permission so the EXIT trap's rm -rf can clean up.
chmod u+w "$STAGES_DIR" 2>/dev/null || true

[[ "$rc" -ne 0 ]] || fail "TC-022 run unexpectedly exited 0 despite the induced write failure"
[[ -f "$SNAPSHOT_A" ]] || fail "TC-022 dispatch 1's own snapshot (written before the fault) is missing"

python3 -c "
import json
i07 = json.loads(open('$WORKDIR_22/lifecycle.jsonl').readline())
assert i07['outcome']['terminal'] == 'error', i07['outcome']
assert i07['outcome']['reason'], 'I-07 outcome.reason is empty'
assert i07['outcome']['publication_eligible'] is False
bundle = json.load(open('$WORKDIR_22/i05/bundle.json'))
assert bundle.get('stop_outcome') == 'error', bundle.get('stop_outcome')
assert bundle['publication_eligible'] is False
assert bundle['ineligibility_reasons'] == [i07['outcome']['reason']], (bundle['ineligibility_reasons'], i07['outcome']['reason'])
assert any('stage' in r for r in bundle['ineligibility_reasons']), bundle['ineligibility_reasons']
" || fail "TC-022 outcome/triad assertions failed: $(cat "$WORKDIR_22/runner.err")"

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR_22/i05" >"$WORKDIR_22/verify.out" 2>"$WORKDIR_22/verify.err" \
	|| fail "TC-022 verify-stage-evidence.sh rejected the bundle after the induced write failure: $(cat "$WORKDIR_22/verify.err")"

rm -rf "$WORKDIR_22"
trap - EXIT

echo "TC-115 TC-022: pass (mid-run write failure is loud, not swallowed)"

# ---------------------------------------------------------------------------
# TC-019/TC-020 (T-E40-F12-006, REQ-F-013, ADR-F12-07, AC-018): run-lifecycle-
# batch.sh dispatch_pair() / run-review-comparison.sh dispatch_gate()
# --i05-bundle-dir pass-through + guard reordering. Round-2 kickback sweep:
# these cases used to live here but sat after TC-008 in this fail-fast,
# single-process script, so a TC-008 failure (owned by T-E40-F12-003, out of
# scope for this task) silently prevented them from ever running. Extracted
# to tc117_i05_caller_pass_through_guard_test.sh, which run-all.sh invokes
# as an independent subprocess -- see that file for the cases and full
# rationale.
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# TC-021 (final, AC-019): the real, unmodified evaluate-lifecycle.sh binary,
# run against a real produced I-05/I-07 pair, emits no missing_join/
# contradictory_join reason at /join/run_id, /join/scenario_id,
# /join/scenario_version, or /join/dispatches -- and DOES emit missing_join
# at /join/dispatch_id and /join/dispatch_ordinal, per AC-019's own required
# negative assertion (these two are Q008's subject, not this feature's own
# defect: spec.md sec 1.4 item 4, sec 3.2). Caller-Path Contract (TC-021
# row, test-plan.md): the real evaluate-lifecycle.sh binary, zero mocks at
# the evaluator itself; the upstream run-lifecycle.sh invocation reuses the
# same SHARK_BIN/LIFECYCLE_ADAPTER seams as TC-012.
# ---------------------------------------------------------------------------

WORKDIR_21="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_21"' EXIT
mkdir -p "$WORKDIR_21/bin" "$WORKDIR_21/scratch" "$WORKDIR_21/i05"
write_single_dispatch_shark "$WORKDIR_21/bin"
write_adapter "$WORKDIR_21/adapter.sh" "worker-021"

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_21/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_21/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-021 --root ROOT-001 --scratch-root "$WORKDIR_21/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_21/i05" --output "$WORKDIR_21/lifecycle.jsonl" ) \
	|| fail "TC-021 (final) fixture run failed"

EVALUATOR_21="$SCRIPTS_DIR/evaluate-lifecycle.sh"
[[ -x "$EVALUATOR_21" ]] || fail "evaluate-lifecycle.sh missing or not executable"
EVAL_OUT_21="$WORKDIR_21/evaluation.json"
set +e
"$EVALUATOR_21" --i05 "$WORKDIR_21/i05" --i07 "$WORKDIR_21/lifecycle.jsonl" --scenario "$SCENARIO" --output "$EVAL_OUT_21" \
	>/dev/null 2>"$WORKDIR_21/eval.err"
set -e
[[ -f "$EVAL_OUT_21" ]] || fail "TC-021 (final) evaluate-lifecycle.sh did not write an output record: $(cat "$WORKDIR_21/eval.err")"

python3 -c "
import json
record = json.load(open('$EVAL_OUT_21'))
reasons = record['eligibility']['invalidity_reasons']
join_paths = {r['path'] for r in reasons if r['code'] in ('missing_join', 'contradictory_join')}
required_clean = {'/join/run_id', '/join/scenario_id', '/join/scenario_version', '/join/dispatches'}
dirty = join_paths & required_clean
assert not dirty, f'AC-019 defect: missing_join/contradictory_join present at {dirty}'
# AC-019's own required negative assertion (test-plan.md TC-021): these two
# ARE expected to be missing_join -- Q008's subject, not this feature's own
# defect (spec.md sec 1.4 item 4, sec 3.2).
expected_dirty = {'/join/dispatch_id', '/join/dispatch_ordinal'}
assert expected_dirty <= join_paths, f'TC-021 negative case: expected missing_join at {expected_dirty}, saw {join_paths}'
" || fail "TC-021 (final) AC-019 join assertions failed"

rm -rf "$WORKDIR_21"
trap - EXIT

echo "TC-115 TC-021 (final): pass (evaluate-lifecycle.sh real join proof, AC-019 positive+negative scope)"

# ---------------------------------------------------------------------------
# TC-023 (AC-021): the unmodified verify-stage-evidence.sh exits 0 on a
# bundle from a real --mode live run of one of the six lifecycle-v2
# families ($SCENARIO is the "bug" family package, satisfying AC-021's own
# "at least one" scope), printing its fixed-order JSON summary.
# Caller-Path Contract (TC-023 row): the real binary over a genuinely
# produced bundle -- never a hand-assembled directory -- enforced
# structurally: this block never constructs bundle.json/stages/*.json
# content itself, only points the validator at the real
# run-lifecycle.sh --mode live invocation's own output directory.
# ---------------------------------------------------------------------------

WORKDIR_23="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_23"' EXIT
mkdir -p "$WORKDIR_23/bin" "$WORKDIR_23/scratch" "$WORKDIR_23/i05"
write_single_dispatch_shark "$WORKDIR_23/bin"
write_adapter "$WORKDIR_23/adapter.sh" "worker-023"

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_23/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_23/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-023 --root ROOT-001 --scratch-root "$WORKDIR_23/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_23/i05" --output "$WORKDIR_23/lifecycle.jsonl" ) \
	|| fail "TC-023 fixture run failed"

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR_23/i05" >"$WORKDIR_23/verify.out" 2>"$WORKDIR_23/verify.err" \
	|| fail "TC-023 verify-stage-evidence.sh rejected the real six-family live-run bundle: $(cat "$WORKDIR_23/verify.err")"

python3 -c "
import json
summary = json.loads(open('$WORKDIR_23/verify.out').read())
assert 'bundle_dir' in summary and 'stages' in summary, f'unexpected verify-stage-evidence.sh summary shape: {sorted(summary.keys())}'
" || fail "TC-023 verify-stage-evidence.sh summary was not the expected fixed-order JSON shape"

rm -rf "$WORKDIR_23"
trap - EXIT

echo "TC-115 TC-023: pass (verify-stage-evidence.sh accepts a real six-family live-run bundle)"

# ---------------------------------------------------------------------------
# TC-024 (AC-022): verify-stage-evidence.sh is byte-identical since the
# pre-feature ref -- REQ-F-016's own "no parallel/relaxed validator, no
# vocabulary of its own" requirement. Same unresolvable-base skip discipline
# as tc020/tc038: a vacuous pass is worse than no check, so a base that
# fails to resolve is logged and skipped, never silently treated as
# "unchanged". Caller-Path Contract (TC-024 row): content-only,
# `git diff <pre-feature-ref> -- bench/scripts/verify-stage-evidence.sh`.
# ---------------------------------------------------------------------------

tc024_base=""
if ! tc024_base="$(cd "$REPO_ROOT" && git merge-base HEAD origin/main 2>/dev/null)"; then
	tc024_base=""
fi

if [[ -z "$tc024_base" ]]; then
	echo "TC-115 TC-024 (no merge-base resolved against origin/main -- skipped, logged not silently passed) SKIP" >&2
else
	tc024_diff="$(cd "$REPO_ROOT" && git diff "$tc024_base" -- bench/scripts/verify-stage-evidence.sh)"
	[[ -z "$tc024_diff" ]] || fail "verify-stage-evidence.sh differs from pre-feature ref $tc024_base (REQ-F-016, AC-022):
$tc024_diff"
	echo "TC-115 TC-024: pass (verify-stage-evidence.sh byte-unchanged since $tc024_base)"
fi

# ---------------------------------------------------------------------------
# TC-025 (AC-023): spec.md sec 3.2 enumerates every I-07 identity and
# workflow-policy field the evaluator requires and the live producer omits,
# and Q008 exists in Shark carrying that enumeration, open and blocking.
# Caller-Path Contract (TC-025 row): direct file read of spec.md sec 3.2
# plus a real `shark get Q008 --json` CLI read, zero mocks.
#
# NOTE for future maintainers (mirroring TC-021's own forward-looking note):
# this asserts Q008 is currently open/blocking because that is its state as
# of this task. A future run that finds Q008 resolved should prompt
# checking whether spec.md sec 3.2 was updated to match the resolution,
# not be treated as this test regressing.
# ---------------------------------------------------------------------------

SPEC_MD="$REPO_ROOT/docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v/spec.md"
[[ -f "$SPEC_MD" ]] || fail "TC-025 spec.md missing: $SPEC_MD"
grep -q "^### 3.2 Adjacent gap this feature does NOT close" "$SPEC_MD" \
	|| fail "TC-025 spec.md sec 3.2 heading missing"
for field in "identity.dispatch_id" "identity.toolchain_identity" "identity.rendered_prompt_digests" \
	"identity.provider_identity" "identity.judge_identity" "identity.reference_digests" \
	"identity.resource_policy_digest" "workflow_policy.enabled_gates" "workflow_policy.reviewer.effort" \
	"workflow_policy.rendered_prompt_digest" "workflow_policy.deep_review_bundle_digest" \
	"workflow_policy.workflow_policy_identity_digest"; do
	grep -qF "$field" "$SPEC_MD" || fail "TC-025 spec.md sec 3.2 does not enumerate $field"
done
grep -q "\*\*Q008\*\*" "$SPEC_MD" || fail "TC-025 spec.md does not name Q008 as the tracking Question"

REAL_SHARK_25="$REPO_ROOT/bin/shark"
[[ -x "$REAL_SHARK_25" ]] || fail "TC-025 bin/shark missing; run 'make shark' first"
WORKDIR_25="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_25"' EXIT
( cd "$REPO_ROOT" && "$REAL_SHARK_25" get Q008 --json ) >"$WORKDIR_25/q008.json" \
	|| fail "TC-025 'shark get Q008 --json' failed: $(cat "$WORKDIR_25/q008.json")"
python3 -c "
import json
record = json.load(open('$WORKDIR_25/q008.json'))
assert record.get('status') == 'open', f\"Q008 status is {record.get('status')!r}, want 'open'\"
assert record.get('blocking') is True, f\"Q008 blocking is {record.get('blocking')!r}, want True\"
" || fail "TC-025 Q008 record did not carry status=open/blocking=true: $(cat "$WORKDIR_25/q008.json")"

rm -rf "$WORKDIR_25"
trap - EXIT

echo "TC-115 TC-025: pass (spec.md sec 3.2 discloses the I-07 gap; Q008 exists, open, blocking)"

# ---------------------------------------------------------------------------
# TC-026 (REQ-NF-002): the producer's per-dispatch write path (serialize +
# write stages/<n>.json + transcript + access.jsonl append + bundle.json
# rewrite) stays under 1s per dispatch on the reference fixture, and at
# most one `shark admin workflow list` call is made per workflow level per
# run. Caller-Path Contract (TC-026 row): wall-clock instrumentation
# wrapped around the real write path inside a real run-lifecycle.sh
# --mode live run (same entrypoint as TC-002). Implementation-contract note
# (test-plan.md TC-026): this task adds the one lightweight, explicit
# timing seam this measurement requires -- a single `i05_write_ms=<n>`
# stderr line emitted once per dispatch when LIFECYCLE_BENCH_TIMING=1 is
# set, never on by default.
# ---------------------------------------------------------------------------

WORKDIR_26="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_26"' EXIT
mkdir -p "$WORKDIR_26/bin" "$WORKDIR_26/scratch" "$WORKDIR_26/i05"
write_single_dispatch_shark "$WORKDIR_26/bin"
write_adapter "$WORKDIR_26/adapter.sh" "worker-026"

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_26/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_26/adapter.sh" \
	SHARK_EVENTS="$WORKDIR_26/events.ndjson" LIFECYCLE_BENCH_TIMING=1 \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-026 --root ROOT-001 --scratch-root "$WORKDIR_26/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_26/i05" --output "$WORKDIR_26/lifecycle.jsonl" \
	>"$WORKDIR_26/runner.out" 2>"$WORKDIR_26/runner.err" ) \
	|| fail "TC-026 fixture run failed: $(cat "$WORKDIR_26/runner.err")"

TC026_TIMING_LINES="$(grep -c '^i05_write_ms=' "$WORKDIR_26/runner.err" || true)"
[[ "$TC026_TIMING_LINES" -eq 1 ]] || fail "TC-026 expected exactly 1 i05_write_ms= line for 1 dispatch, got $TC026_TIMING_LINES: $(cat "$WORKDIR_26/runner.err")"

python3 -c "
line = [l for l in open('$WORKDIR_26/runner.err') if l.startswith('i05_write_ms=')][0]
ms = float(line.strip().split('=', 1)[1])
assert ms >= 0, f'negative write duration: {ms}'
assert ms < 1000, f'REQ-NF-002: per-dispatch write path took {ms}ms, want < 1000ms'
" || fail "TC-026 timing assertion failed"

python3 -c "
import json
events = [json.loads(l) for l in open('$WORKDIR_26/events.ndjson') if l.strip()]
workflow_calls = [e['argv'] for e in events if e['argv'][:3] == ['admin', 'workflow', 'list']]
levels = [call[3] for call in workflow_calls]
assert len(levels) == len(set(levels)), f'shark admin workflow list called more than once for a level: {levels}'
" || fail "TC-026 at-most-once-per-level shark admin workflow list assertion failed"

echo "TC-115 TC-026 (timing + call-count): pass"

# Negative case: without LIFECYCLE_BENCH_TIMING set, the seam must stay
# silent (opt-in only, never a default-on stderr line).
WORKDIR_26B="$(mktemp -d)"
mkdir -p "$WORKDIR_26B/bin" "$WORKDIR_26B/scratch" "$WORKDIR_26B/i05"
write_single_dispatch_shark "$WORKDIR_26B/bin"
write_adapter "$WORKDIR_26B/adapter.sh" "worker-026b"
( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_26B/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_26B/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-026b --root ROOT-001 --scratch-root "$WORKDIR_26B/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_26B/i05" --output "$WORKDIR_26B/lifecycle.jsonl" \
	>"$WORKDIR_26B/runner.out" 2>"$WORKDIR_26B/runner.err" ) \
	|| fail "TC-026 (opt-out) fixture run failed"
! grep -q '^i05_write_ms=' "$WORKDIR_26B/runner.err" \
	|| fail "TC-026 negative: i05_write_ms= line appeared without LIFECYCLE_BENCH_TIMING=1"
rm -rf "$WORKDIR_26B"

rm -rf "$WORKDIR_26"
trap - EXIT

echo "TC-115 TC-026 (opt-in-only negative case): pass"

# ---------------------------------------------------------------------------
# TC-027 (REQ-NF-003): no secret or raw prompt material lands in the
# bundle. Grepping every file under i05_bundle_dir (including transcripts/)
# for the literal rendered-prompt text of the same run finds no match, and
# the existing, unmodified verify-evidence-roots.sh stays green against the
# same run's roots. Caller-Path Contract (TC-027 row): grep over real
# produced output, driven from the same real run as TC-002, no mocks for
# the assertion itself.
#
# `run-lifecycle.sh` resolves `agent_fixture_checkout` straight to the
# scenario's declared `fixture.submodule_path` (line 341) -- the AMBIENT
# submodule checkout, whatever commit it happens to be sitting at, not a
# base_sha-pinned copy. `verify-evidence-roots.sh` fail-closed checks that
# checkout's HEAD against the package's declared `fixture.base_sha`
# (REQ-F-010/011's drift guard). To give this guard a checkout it can
# actually accept without depending on the ambient dev checkout's current
# submodule commit (an environment precondition this test does not
# control), this block clones a base_sha-pinned copy via the existing,
# unmodified `checkout-scenario-fixture.sh` (the same production tool
# T-E40-F05-006 shipped for exactly this) and points a scratch copy of the
# scenario package at it -- never a hand-edited i05 bundle, never a mocked
# guard function.
# ---------------------------------------------------------------------------

WORKDIR_27="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_27"' EXIT
mkdir -p "$WORKDIR_27/bin" "$WORKDIR_27/scratch" "$WORKDIR_27/i05"
write_single_dispatch_shark "$WORKDIR_27/bin"
write_adapter "$WORKDIR_27/adapter.sh" "worker-027"

python3 -c "
import shutil, yaml
src_pkg = '$SCENARIO'
with open(src_pkg) as f:
    package = yaml.safe_load(f)
fixture = package['fixture']
print(fixture['fixture_id'])
print(fixture['base_sha'])
"  >"$WORKDIR_27/fixture-info.txt" || fail "TC-027 could not read fixture info from $SCENARIO"
mapfile -t TC27_FIXTURE_INFO <"$WORKDIR_27/fixture-info.txt"
TC27_FIXTURE_ID="${TC27_FIXTURE_INFO[0]}"
TC27_BASE_SHA="${TC27_FIXTURE_INFO[1]}"

CHECKOUT_SCENARIO_FIXTURE_27="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
[[ -x "$CHECKOUT_SCENARIO_FIXTURE_27" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
"$CHECKOUT_SCENARIO_FIXTURE_27" "$TC27_FIXTURE_ID" "$TC27_BASE_SHA" "$WORKDIR_27/pinned-fixture" \
	>"$WORKDIR_27/checkout.out" 2>"$WORKDIR_27/checkout.err" \
	|| fail "TC-027 checkout-scenario-fixture.sh failed: $(cat "$WORKDIR_27/checkout.err")"

TC27_SCENARIO_DIR="$(dirname "$SCENARIO")"
cp -r "$TC27_SCENARIO_DIR" "$WORKDIR_27/scenario-pkg"
TC27_SCENARIO="$WORKDIR_27/scenario-pkg/package.yaml"
python3 -c "
import yaml
path = '$TC27_SCENARIO'
with open(path) as f:
    package = yaml.safe_load(f)
package['fixture']['submodule_path'] = '$WORKDIR_27/pinned-fixture'
with open(path, 'w') as f:
    yaml.safe_dump(package, f)
" || fail "TC-027 could not repoint the scratch package's fixture.submodule_path"

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_27/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_27/adapter.sh" \
	"$RUNNER" --scenario "$TC27_SCENARIO" --run-id tc115-027 --root ROOT-001 --scratch-root "$WORKDIR_27/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_27/i05" --output "$WORKDIR_27/lifecycle.jsonl" ) \
	|| fail "TC-027 fixture run failed"

PROMPT_TEXT_27="$(python3 -c "import json; print(json.load(open('$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json'))['prompt'].strip())")"
[[ -n "$PROMPT_TEXT_27" ]] || fail "TC-027 could not derive the rendered-prompt text fixture"
if grep -R -l -F -- "$PROMPT_TEXT_27" "$WORKDIR_27/i05" >"$WORKDIR_27/leak.out" 2>/dev/null; then
	fail "TC-027 rendered-prompt bytes leaked into the bundle: $(cat "$WORKDIR_27/leak.out")"
fi

ROOTS_GUARD_27="$SCRIPTS_DIR/verify-evidence-roots.sh"
[[ -x "$ROOTS_GUARD_27" ]] || fail "verify-evidence-roots.sh missing or not executable"
python3 -c "
import json, os
bundle = json.load(open('$WORKDIR_27/i05/bundle.json'))
roots = bundle['roots']
print(roots['agent_fixture_checkout'])
print(roots['scratch_shark_project'])
# roots['evaluator_only'] is <scenario_dir>/evaluator (run-lifecycle.sh line
# 343); verify-evidence-roots.sh's own <evaluator_root> parameter is the
# scenario PACKAGE directory it resolves package.yaml's declared
# evaluator/... paths against, i.e. evaluator_only's parent.
print(os.path.dirname(roots['evaluator_only']))
" >"$WORKDIR_27/roots.txt"
mapfile -t TC27_ROOTS <"$WORKDIR_27/roots.txt"
( cd "$REPO_ROOT" && "$ROOTS_GUARD_27" "$TC27_SCENARIO" "${TC27_ROOTS[0]}" "${TC27_ROOTS[1]}" "${TC27_ROOTS[2]}" ) \
	>"$WORKDIR_27/roots-guard.out" 2>"$WORKDIR_27/roots-guard.err" \
	|| fail "TC-027 verify-evidence-roots.sh rejected this run's roots: $(cat "$WORKDIR_27/roots-guard.err")"

rm -rf "$WORKDIR_27"
trap - EXIT

echo "TC-115 TC-027: pass (no prompt-byte leakage; verify-evidence-roots.sh stays green)"

# ---------------------------------------------------------------------------
# TC-028 (REQ-NF-004): two independent real run-lifecycle.sh --mode live
# invocations with identical scenario/run-id/stub inputs produce
# byte-identical bundle.json/stages/*.json apart from timestamps, digests
# derived from real per-dispatch timing (time_ledger, and snapshot_digest
# which hashes over it), and workdir-specific absolute paths. Canonical
# serialization is sort_keys=True, separators=(",", ":") throughout.
# Caller-Path Contract (TC-028 row): two real run-lifecycle.sh --mode live
# invocations, same entrypoint/seams as TC-002.
# ---------------------------------------------------------------------------

WORKDIR_28A="$(mktemp -d)"
WORKDIR_28B="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_28A" "$WORKDIR_28B"' EXIT
for wd in "$WORKDIR_28A" "$WORKDIR_28B"; do
	mkdir -p "$wd/bin" "$wd/scratch" "$wd/i05"
	write_single_dispatch_shark "$wd/bin"
	write_adapter "$wd/adapter.sh" "worker-028"
done

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_28A/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_28A/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-028 --root ROOT-001 --scratch-root "$WORKDIR_28A/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_28A/i05" --output "$WORKDIR_28A/lifecycle.jsonl" ) \
	|| fail "TC-028 run A failed"

( cd "$REPO_ROOT" && \
  PATH="$WORKDIR_28B/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR_28B/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc115-028 --root ROOT-001 --scratch-root "$WORKDIR_28B/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR_28B/i05" --output "$WORKDIR_28B/lifecycle.jsonl" ) \
	|| fail "TC-028 run B failed"

python3 -c "
import glob
import json
import os

def normalize(obj, replace_from):
    if isinstance(obj, dict):
        out = {}
        for k, v in obj.items():
            if k == 'time_ledger' or k == 'snapshot_digest' or k.endswith('_at'):
                continue
            out[k] = normalize(v, replace_from)
        return out
    if isinstance(obj, list):
        return [normalize(v, replace_from) for v in obj]
    if isinstance(obj, str):
        for old, new in replace_from:
            obj = obj.replace(old, new)
        return obj
    return obj

def load_run(i05_dir, workdir):
    replace_from = [(workdir, '<WORKDIR>')]
    with open(f'{i05_dir}/bundle.json') as f:
        bundle = normalize(json.load(f), replace_from)
    stages = {}
    for path in sorted(glob.glob(f'{i05_dir}/stages/*.json')):
        with open(path) as f:
            stages[os.path.basename(path)] = normalize(json.load(f), replace_from)
    return bundle, stages

bundle_a, stages_a = load_run('$WORKDIR_28A/i05', '$WORKDIR_28A')
bundle_b, stages_b = load_run('$WORKDIR_28B/i05', '$WORKDIR_28B')

dump = lambda o: json.dumps(o, sort_keys=True, separators=(',', ':'))
assert dump(bundle_a) == dump(bundle_b), f'bundle.json diverged between identical-input runs:\n{dump(bundle_a)}\n!=\n{dump(bundle_b)}'
assert set(stages_a) == set(stages_b), f'stage filenames diverged: {sorted(stages_a)} vs {sorted(stages_b)}'
for name in stages_a:
    assert dump(stages_a[name]) == dump(stages_b[name]), f'stage {name} diverged between identical-input runs'
" || fail "TC-028 determinism assertions failed"

rm -rf "$WORKDIR_28A" "$WORKDIR_28B"
trap - EXIT

echo "TC-115 TC-028: pass (deterministic bundle writes across identical-input runs)"

# ---------------------------------------------------------------------------
# REWORK T-E40-F12-006 kickback guard cases (defect class
# "cli-contract-change-missed-caller": smoke-lifecycle.sh pass-through proof
# + the structural static-scan guard over every bench/scripts/*.sh
# run-lifecycle.sh invocation site). Round-2 kickback sweep: these two cases
# used to live here but sat after TC-008 in this fail-fast, single-process
# script, so a TC-008 failure (owned by T-E40-F12-003, out of scope for this
# task) silently prevented them from ever running -- they had never once
# executed in the checked-in suite. Extracted to
# tc117_i05_caller_pass_through_guard_test.sh, which run-all.sh invokes as
# an independent subprocess, so their pass/fail state is no longer coupled
# to TC-008 or to any of tc115's other ~28 cases. See that file for the
# cases and full rationale.
# ---------------------------------------------------------------------------


echo "TC-115: pass"
