#!/usr/bin/env bash
# TC-118 / TD-148: the real Shark CLI, lifecycle producer, and I-07 verifier
# must agree on one dry-run record. No PATH stub is permitted at this boundary.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
SMOKE="$SCRIPTS_DIR/smoke-lifecycle.sh"
REAL_SHARK="$REPO_ROOT/bin/shark"
CHECKOUT_FIXTURE="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
VERIFIER="$SCRIPTS_DIR/verify-lifecycle-run.sh"
PACKAGE="$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml"
LIFECYCLE_ADAPTER="$SCRIPTS_DIR/lifecycle-worker-adapter.sh"

fail() { echo "TC-118 FAIL: $1" >&2; exit 1; }
[[ -x "$SMOKE" ]] || fail "smoke-lifecycle.sh missing or not executable"
[[ -x "$REAL_SHARK" ]] || fail "real shark binary missing at $REAL_SHARK (run 'make shark' first)"
[[ -x "$CHECKOUT_FIXTURE" && -x "$VERIFIER" && -f "$PACKAGE" ]] || fail "required lifecycle fixture tooling is missing"
[[ -x "$LIFECYCLE_ADAPTER" ]] || fail "lifecycle worker adapter is missing"

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc118.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

read -r FIXTURE_ID FIXTURE_SHA < <(python3 - "$PACKAGE" <<'PY'
import sys
import yaml
fixture = (yaml.safe_load(open(sys.argv[1], encoding="utf-8")) or {})["fixture"]
print(fixture["fixture_id"], fixture["base_sha"])
PY
)
FIXTURE_CHECKOUT="$WORKDIR/fixture"
"$CHECKOUT_FIXTURE" "$FIXTURE_ID" "$FIXTURE_SHA" "$FIXTURE_CHECKOUT"
[[ "$(git -C "$FIXTURE_CHECKOUT" rev-parse HEAD)" == "$FIXTURE_SHA" ]] || fail "fixture checkout does not match scenario-pinned base SHA"

# A deterministic local provider envelope supplies the mapped usage evidence
# required by a live lifecycle record. Shark itself remains the real binary;
# this command only prevents an external provider call or spend.
PROVIDER="$WORKDIR/provider.sh"
cat >"$PROVIDER" <<'EOF'
#!/usr/bin/env bash
cat >/dev/null
printf '%s\n' '{"kind":"final","recommended_outcome":"pass","evidence":[],"total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"modelUsage":{"fixture-model":{}},"duration_api_ms":1,"num_turns":1,"session_id":"tc118-provider"}'
EOF
chmod +x "$PROVIDER"

smoke_rc=0
LIFECYCLE_PROVIDER_COMMAND="[\"$PROVIDER\"]" "$SMOKE" --live --adapter "$LIFECYCLE_ADAPTER" --out "$WORKDIR/smoke" --run-id tc118-real-contract --fixture-root "$FIXTURE_CHECKOUT" </dev/null || smoke_rc=$?
RECORD="$WORKDIR/smoke/runs/tc118-real-contract-live/lifecycle.jsonl"
[[ -s "$RECORD" ]] || fail "smoke-lifecycle did not produce lifecycle.jsonl"
"$VERIFIER" "$RECORD" --schema "$REPO_ROOT/bench/runs/i07-schema.yaml" || fail "production verifier rejected the real lifecycle record"

python3 - "$RECORD" <<'PY'
import json
import hashlib
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    record = json.loads(stream.readline())
assert record["identity"]["shark_binary_digest"], record
assert record["dispatches"], record
assert record["stages"], record
candidate = record["stages"][0]["candidate"]
fields = (
    "base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest",
    "dirty_untracked_manifest", "test_suite_digest", "scratch_content_digest",
)
assert all(candidate.get(field) not in (None, "") for field in fields), candidate
expected = hashlib.sha256(json.dumps({field: candidate[field] for field in fields}, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()
assert candidate["identity_digest"] == expected, candidate
PY

echo "TC-118: real Shark lifecycle record at the scenario-pinned fixture SHA passes the production I-07 verifier (smoke exit=$smoke_rc)"
