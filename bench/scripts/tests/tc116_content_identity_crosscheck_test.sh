#!/usr/bin/env bash
# TC-116 / T-E40-F12-005: TD-132's content_root + walk-based digest scheme
# crosscheck (AC-016/AC-017, ADR-F12-03), driven from a REAL live-produced
# I-07 record -- complementing tc075_content-identity_x12_test.sh, which can
# only reach the crosscheck branch by hand-injecting a synthetic
# content_root/identity pair (its own header says so). Here, content_root
# comes from run-lifecycle.sh's own scenario_identity(), against a real
# scratch project with the installed Shark-data canonical content tree
# copied into it (ADR-F12-03's own precondition).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
EVALUATOR="$SCRIPTS_DIR/evaluate-lifecycle.sh"
SCENARIO="$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml"
CANONICAL_TREE="$REPO_ROOT/internal/sharkdata/default_data"
WORKFLOW_FIXTURE_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow"
fail() { echo "TC-116 FAIL: $1" >&2; exit 1; }
[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable"
[[ -x "$EVALUATOR" ]] || fail "evaluate-lifecycle.sh missing or not executable"
[[ -f "$SCENARIO" ]] || fail "scenario package missing: $SCENARIO"
[[ -d "$CANONICAL_TREE" ]] || fail "canonical Shark-data tree missing: $CANONICAL_TREE"
[[ -d "$WORKFLOW_FIXTURE_DIR" ]] || fail "frozen workflow-phase fixtures missing: $WORKFLOW_FIXTURE_DIR"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

# tc115's established single-dispatch `shark` stub: root ROOT-001 resolves
# directly to the scripted SHARK_RESPONSE dispatch; `admin workflow list
# <level>` answers from the frozen real-captured fixtures.
write_single_dispatch_shark() {
	local bin_dir="$1"
	local entity_key="${2:-TASK-002}"
	cat >"$bin_dir/shark" <<SHARK
#!/usr/bin/env bash
set -euo pipefail
python3 - "\$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
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
    print('{"session_id":"SID-116"}')
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

# tc115's established adapter stub: a `final`/`pass` worker result, so the
# single dispatch reaches a clean terminal in `--mode live` (TC-017's own
# Caller-Path Contract entrypoint, not `--mode dry-run`'s synthetic
# adapter_result() short-circuit).
write_adapter() {
	local path="$1" worker_id="$2"
	cat >"$path" <<ADAPTER
#!/usr/bin/env bash
set -euo pipefail
python3 -c 'import json,sys; request=json.load(sys.stdin); print(json.dumps({"worker_id":"$worker_id","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":0.0,"evidence":{"summary":"fixture"}}))'
ADAPTER
	chmod +x "$path"
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch" "$WORKDIR/i05"

# ADR-F12-03 precondition: "content_root is the installed Shark-data
# canonical content tree inside the scratch Shark project" -- copied at the
# conventional `shark admin install-shark-data` default location
# (<project_root>/shark-data), never the repo's own
# internal/sharkdata/default_data path directly.
cp -a "$CANONICAL_TREE" "$WORKDIR/scratch/shark-data"

write_single_dispatch_shark "$WORKDIR/bin" "BUG-900"
write_adapter "$WORKDIR/adapter.sh" "worker-116"

PATH="$WORKDIR/bin:$PATH" SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-research.json" \
	SHARK_WORKFLOW_DIR="$WORKFLOW_FIXTURE_DIR" LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" \
	"$RUNNER" --scenario "$SCENARIO" --run-id tc116 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
	--mode live --i05-bundle-dir "$WORKDIR/i05" --output "$WORKDIR/lifecycle.jsonl" \
	|| fail "real live run failed"

# Extract the REAL content_digest() source out of evaluate-lifecycle.sh
# (tc075's established technique) -- never a hand-retyped reimplementation
# that could silently drift from the production function.
EXPECTED_DIGEST="$(python3 - "$EVALUATOR" "$WORKDIR/scratch/shark-data" <<'PY'
import pathlib
import re
import sys

evaluator, content_root = sys.argv[1:3]
source_text = pathlib.Path(evaluator).read_text(encoding="utf-8")
match = re.search(r"\ndef content_digest\(root\):.*?\n(?=\S)", source_text, re.S)
assert match, "content_digest() source not found in evaluate-lifecycle.sh"
namespace = {"os": __import__("os"), "hashlib": __import__("hashlib")}
exec(compile(match.group(0), "evaluate-lifecycle.sh::content_digest", "exec"), namespace)  # noqa: S102
reference_content_digest = namespace["content_digest"]
digest = reference_content_digest(content_root)
assert digest, "reference content_digest() produced no value for the installed canonical tree"
print(digest)
PY
)"
[[ -n "$EXPECTED_DIGEST" ]] || fail "could not compute reference content digest"

# --- TC-017 (AC-016): identity.content_root/content_digest_scheme/
# shark_content_digest on the REAL produced record.
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR/scratch/shark-data" "$EXPECTED_DIGEST" <<'PY'
import json
import sys

i07_path, content_root, expected_digest = sys.argv[1:4]
with open(i07_path, encoding="utf-8") as f:
    record = json.loads(f.readline())
identity = record["identity"]
assert identity["content_root"] == content_root, (identity["content_root"], content_root)
assert identity["content_digest_scheme"] == "walk_v1", identity["content_digest_scheme"]
assert identity["shark_content_digest"] == expected_digest, (identity["shark_content_digest"], expected_digest)
PY
echo "TC-116 TC-017 (identity fields): pass"

# --- TC-017 (AC-016), real evaluator half: the crosscheck runs against the
# real produced bundle.json/lifecycle.jsonl pair and does NOT report
# identity_mismatch at /identity/shark_content_digest (declared and
# independently re-derived digests agree).
EVAL_OUT_MATCH="$WORKDIR/evaluation-match.jsonl"
"$EVALUATOR" --i05 "$WORKDIR/i05" --i07 "$WORKDIR/lifecycle.jsonl" --scenario "$SCENARIO" --output "$EVAL_OUT_MATCH" >/dev/null 2>&1 || true
python3 -c "
import json
record = json.load(open('$EVAL_OUT_MATCH'))
reasons = record['eligibility']['invalidity_reasons']
mismatch = [r for r in reasons if r['code'] == 'identity_mismatch' and r['path'] == '/identity/shark_content_digest']
assert not mismatch, ('matching content_root falsely flagged as a mismatch', mismatch)
"
echo "TC-116 TC-017 (evaluator crosscheck, matching case): pass"

# --- TC-017 (AC-016) negative case, equivalence partitioning: a record
# lacking content_digest_scheme/content_root entirely (a pre-feature legacy
# record, simulated by stripping both fields from the real produced record)
# is treated as the legacy whole-repo-tree scheme -- the crosscheck has
# nothing to derive a content_root from, so it never fires, and the record
# is never silently compared against a walk_v1 record as if equivalent.
LEGACY_I07="$WORKDIR/lifecycle-legacy.jsonl"
python3 -c "
import json
with open('$WORKDIR/lifecycle.jsonl', encoding='utf-8') as f:
    record = json.loads(f.readline())
del record['identity']['content_root']
del record['identity']['content_digest_scheme']
with open('$LEGACY_I07', 'w', encoding='utf-8') as f:
    f.write(json.dumps(record) + '\n')
"
EVAL_OUT_LEGACY="$WORKDIR/evaluation-legacy.jsonl"
"$EVALUATOR" --i05 "$WORKDIR/i05" --i07 "$LEGACY_I07" --scenario "$SCENARIO" --output "$EVAL_OUT_LEGACY" >/dev/null 2>&1 || true
python3 -c "
import json
record = json.load(open('$EVAL_OUT_LEGACY'))
reasons = record['eligibility']['invalidity_reasons']
mismatch = [r for r in reasons if r['code'] == 'identity_mismatch' and r['path'] == '/identity/shark_content_digest']
assert not mismatch, ('a legacy-shaped record (no content_root) must never trip the content-digest crosscheck', mismatch)
"
echo "TC-116 TC-017 (scheme-marker-absent legacy partition): pass"

# --- TC-018 (AC-017): corrupt one byte under content_root; the crosscheck
# now fires against the SAME (unchanged) real produced record.
CORRUPT_TARGET="$WORKDIR/scratch/shark-data/workflow/task.yaml"
[[ -f "$CORRUPT_TARGET" ]] || fail "expected canonical file missing: $CORRUPT_TARGET"
printf '\n# TC-116 corruption\n' >>"$CORRUPT_TARGET"

EVAL_OUT_CORRUPT="$WORKDIR/evaluation-corrupt.jsonl"
"$EVALUATOR" --i05 "$WORKDIR/i05" --i07 "$WORKDIR/lifecycle.jsonl" --scenario "$SCENARIO" --output "$EVAL_OUT_CORRUPT" >/dev/null 2>&1 || true
python3 -c "
import json
record = json.load(open('$EVAL_OUT_CORRUPT'))
reasons = record['eligibility']['invalidity_reasons']
mismatch = [r for r in reasons if r['code'] == 'identity_mismatch' and r['path'] == '/identity/shark_content_digest']
assert mismatch, ('corrupting content_root must trip the content-digest crosscheck', reasons)
"
echo "TC-116 TC-018 (corruption fires): pass"

echo "TC-116: all cases pass"
