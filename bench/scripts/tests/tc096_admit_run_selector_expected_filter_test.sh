#!/usr/bin/env bash
# TC-096 (B053 regression). check_p2p_green() in admit.sh passes a
# p2p_set's run_selector to `go test -run` (so only selected tests
# actually execute) but, before this fix, computed `expected` from every
# testenum-enumerated top-level test in the package, never filtered by
# that same run_selector. A valid non-empty selector then left every
# OTHER top-level test in `expected`, reported as a spurious missing
# terminal event, and deterministically rejected an otherwise-clean P2P
# set with P2P-red -- see docs/plan/bugs/B053.md.
#
# This test builds a transient corpus.yaml (task spec REQ-F-007 pattern
# already used by tc005's branches (c)/(e)): a copy of the real,
# committed corpus.yaml with one addition -- a new p2p_set
# "pricing_taxamount_only" scoped to ./pkg/pricing/... with
# run_selector: "^TestTaxAmount$" -- and reassigns the already-admitted
# "pricing-negative-subtotal" item to that set. No committed corpus.yaml
# entry is touched (REQ-F-007); this is exactly the same transient-item
# technique tc005 already uses.
#
# Caller-Path Contract: same as tc004/tc005 -- the real admit.sh
# entrypoint driving real `go test`, `git apply`, and testenum. Nothing
# about check_p2p_green's own control flow is stubbed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-096 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

TRANSIENT_ITEM="pricing-negative-subtotal"
TRANSIENT_SET="pricing_taxamount_only"
TRANSIENT_CORPUS="$WORKDIR/corpus-run-selector.yaml"

python3 - "$CORPUS_YAML" "$TRANSIENT_CORPUS" "$TRANSIENT_ITEM" "$TRANSIENT_SET" <<'PYEOF'
import os
import sys
import yaml

src_path, dst_path, item_id, set_name = sys.argv[1:5]

with open(src_path) as f:
    data = yaml.safe_load(f)

if item_id not in {it["id"] for it in data["items"]}:
    sys.exit(f"TC-096 setup: item not found in corpus.yaml: {item_id}")

data["p2p_sets"][set_name] = {
    "packages": ["./pkg/pricing/..."],
    "run_selector": "^TestTaxAmount$",
    "exclude_tests": [],
}

corpus_dir = os.path.dirname(os.path.abspath(src_path))

for item in data["items"]:
    if item["id"] == item_id:
        item["p2p_set"] = set_name
        # admit.sh resolves every path field in an item relative to the
        # corpus.yaml file it was given (os.path.dirname(corpus_yaml_path)).
        # This transient copy lives in a different directory, so rewrite
        # every path field to an absolute path pointing back at the real,
        # committed corpus/ tree it was copied from.
        for key in ("prompt_path", "seed_path", "reference_patch_path"):
            if key in item:
                item[key] = os.path.join(corpus_dir, item[key])
        for i, p in enumerate(item["f2p"]["paths"]):
            item["f2p"]["paths"][i] = os.path.join(corpus_dir, p)
        break

with open(dst_path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF

echo "TC-096: running admit.sh against a p2p_set with a non-empty run_selector"
set +e
OUT_FILE="$WORKDIR/verdict.json"
"$ADMIT_SCRIPT" "$TRANSIENT_CORPUS" --item "$TRANSIENT_ITEM" >"$OUT_FILE"
CODE=$?
set -e

python3 - "$OUT_FILE" "$CODE" <<'PYEOF'
import json
import sys

verdict_path, code_str = sys.argv[1:3]
code = int(code_str)

with open(verdict_path) as f:
    line = f.read().strip()

try:
    verdict = json.loads(line)
except json.JSONDecodeError as exc:
    sys.exit(f"TC-096 FAIL: admit.sh did not print a single JSON verdict line: {line!r} ({exc})")

if verdict["status"] != "admitted":
    sys.exit(
        "TC-096 FAIL: expected status 'admitted' for a p2p_set whose "
        f"run_selector selects only a passing subset, got {verdict['status']!r} "
        f"(failing_check={verdict.get('failing_check')!r}); a valid non-empty "
        "run_selector must not report unselected tests as missing terminal "
        f"events (B053): {line}"
    )
if code != 0:
    sys.exit(f"TC-096 FAIL: admit.sh exited {code} for an admitted item (expected 0): {line}")

print("TC-096: p2p_set with non-empty run_selector admitted cleanly (B053 fixed)")
PYEOF

echo "TC-096: PASS"
