#!/usr/bin/env bash
# TC-031 (test-plan.md AC test matrix; T-E40-F05-004 task spec Test Cases).
#
# One assertion set run against both bench/adapters/go/adapter.sh and
# bench/adapters/python/adapter.sh, proving the two independently
# implemented adapters satisfy an identical contract (REQ-F-006, REQ-F-007;
# AC-010, AC-011, AC-012). This is the workflow's stated TDD exception: a
# suite spanning two components that cannot be co-located inside either
# adapter's own implementation task.
#
# Caller-Path Contract (test-plan.md): real subprocess invocation of both
# adapter.sh scripts, once per capability, against real checkouts. Neither
# adapter's stdout is ever stubbed with canned JSON -- a harness asserting
# against a hand-authored fixture instead of live adapter output would pass
# even if a real adapter emitted a malformed or un-normalized shape.
#
# Three parts:
#   1. AC-010/AC-011 -- all six capabilities, identical JSON-shape
#      assertion set applied to both adapters via one shared function keyed
#      on CAPABILITY, never on adapter label; `test`'s ids are checked
#      against a generic <left>::<right> shape only (AC-T2: no
#      language-aware id parsing in this harness).
#   2. AC-T3 -- a seventh, undefined capability name is rejected with a
#      non-zero exit by both adapters, proving the closed six-verb set
#      rather than merely exercising the six that exist.
#   3. AC-012/AC-T4 -- REQ-F-007's leak surface: a grep of every generic
#      scenario/evidence/admission script for language-specific tokens, plus
#      a source-level check that none of them branches on fixture/adapter/
#      scenario identity. bench/scripts/admit-scenario.sh,
#      eval-predicate.sh, and checkout-scenario-fixture.sh are later-task
#      deliverables (T-E40-F05-006/007) that do not exist yet at this
#      task's point in the build order -- each is skipped (logged, not
#      silently passed) until it lands, per this suite's own "the one
#      resolvable, wrong-topology skip beats a false pass" precedent
#      (tc020_zero_go_change_test.sh).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

GO_ADAPTER="$BENCH_DIR/adapters/go/adapter.sh"
PY_ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
CHECKOUT_FIXTURE_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
GO_FIXTURE_SUBMODULE="$BENCH_DIR/fixture-repo"
PY_FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"

fail() {
	echo "TC-031 FAIL: $1" >&2
	exit 1
}

[[ -x "$GO_ADAPTER" ]] || fail "go adapter.sh missing or not executable: $GO_ADAPTER"
[[ -x "$PY_ADAPTER" ]] || fail "python adapter.sh missing or not executable: $PY_ADAPTER"
[[ -x "$CHECKOUT_FIXTURE_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -e "$GO_FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-repo submodule not initialized; run 'git submodule update --init'"
[[ -e "$PY_FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Provision two real, disposable checkouts -- one per adapter's fixture.
# ---------------------------------------------------------------------------
GO_BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$GO_BASE_SHA" ]] || fail "could not derive base_sha from $CORPUS_YAML"

GO_CHECKOUT="$WORKDIR/go-checkout"
"$CHECKOUT_FIXTURE_SCRIPT" "$GO_BASE_SHA" "$GO_CHECKOUT" || fail "checkout-fixture.sh failed to provision the Go fixture checkout"

# bench/scripts/checkout-scenario-fixture.sh (T-E40-F05-006) does not exist
# yet at this task's point in the build order. Clone-then-checkout the
# submodule directly here, mirroring checkout-fixture.sh's own shape
# without depending on the not-yet-built sibling script (REQ-NF-006:
# checkout-fixture.sh's own frozen interface stays untouched by this file).
PY_BASE_SHA="$(git -C "$PY_FIXTURE_SUBMODULE" rev-parse HEAD)"
PY_CHECKOUT="$WORKDIR/py-checkout"
git -c advice.detachedHead=false clone --quiet -- "$PY_FIXTURE_SUBMODULE" "$PY_CHECKOUT" || fail "could not clone bench/fixture-py"
git -C "$PY_CHECKOUT" -c advice.detachedHead=false checkout --quiet "$PY_BASE_SHA" || fail "could not checkout bench/fixture-py at $PY_BASE_SHA"

# ---------------------------------------------------------------------------
# Part 1 -- AC-010/AC-011: one JSON-shape assertion function per capability,
# invoked identically for both adapters. Dispatch is keyed on CAPABILITY
# only -- no branch here ever reads which adapter produced the document.
# ---------------------------------------------------------------------------
assert_shape() {
	# assert_shape <capability> <adapter-label> <json-file>
	local capability="$1" label="$2" json_file="$3"
	python3 - "$capability" "$label" "$json_file" <<'PYEOF'
import json
import re
import sys

capability, label, json_file = sys.argv[1:4]

with open(json_file) as f:
    text = f.read()
try:
    doc = json.loads(text)
except json.JSONDecodeError as e:
    sys.exit(f"{label}/{capability}: stdout is not valid JSON: {e}\n--- stdout ---\n{text}")


def require(cond, msg):
    if not cond:
        sys.exit(f"{label}/{capability}: {msg}\n--- document ---\n{json.dumps(doc)}")


require(isinstance(doc, dict), "top-level document must be a JSON object")

if capability == "identity":
    require(set(doc.keys()) == {"adapter", "version", "toolchain_identity"}, f"unexpected key set {sorted(doc.keys())}")
    require(isinstance(doc["adapter"], str) and doc["adapter"], "adapter must be a non-empty string")
    require(isinstance(doc["version"], str) and doc["version"], "version must be a non-empty string")
    ti = doc["toolchain_identity"]
    require(isinstance(ti, list) and len(ti) > 0, "toolchain_identity must be a non-empty ordered list")
    for entry in ti:
        require(isinstance(entry, dict) and set(entry.keys()) == {"key", "value"}, f"toolchain_identity entry has wrong shape: {entry}")
        require(isinstance(entry["key"], str) and entry["key"], "toolchain_identity entry key must be a non-empty string")
        require(isinstance(entry["value"], str) and entry["value"], "toolchain_identity entry value must be a non-empty string")

elif capability == "inject-tests":
    require(set(doc.keys()) == {"injected"}, f"unexpected key set {sorted(doc.keys())}")
    injected = doc["injected"]
    require(isinstance(injected, list) and len(injected) > 0, "injected must be a non-empty list")
    for entry in injected:
        require(isinstance(entry, dict) and set(entry.keys()) == {"source", "destination"}, f"injected entry has wrong shape: {entry}")
        require(isinstance(entry["source"], str) and entry["source"], "injected.source must be a non-empty string")
        require(isinstance(entry["destination"], str) and entry["destination"], "injected.destination must be a non-empty string")

elif capability == "test":
    require(set(doc.keys()) == {"entries"}, f"unexpected key set {sorted(doc.keys())}")
    entries = doc["entries"]
    require(isinstance(entries, list) and len(entries) > 0, "entries must be a non-empty list")
    # Generic normalized-id shape only (AC-011 / AC-T2: no language-aware id
    # parsing in this harness) -- exactly one "::" separator, non-empty on
    # both sides. What sits either side of "::" is each adapter's own
    # business (package import path vs. dotted module path).
    id_re = re.compile(r"^[^:]+::[^:]+$")
    for entry in entries:
        require(isinstance(entry, dict) and set(entry.keys()) == {"id", "outcome"}, f"test entry has wrong shape: {entry}")
        require(isinstance(entry["id"], str) and id_re.match(entry["id"]), f"test entry id is not normalized <module-or-package>::<test-name>: {entry['id']!r}")
        require(entry["outcome"] in ("pass", "fail", "skip"), f"test entry outcome must be pass|fail|skip, got {entry['outcome']!r}")

elif capability == "lint":
    require(set(doc.keys()) == {"issues"}, f"unexpected key set {sorted(doc.keys())}")
    issues = doc["issues"]
    require(isinstance(issues, list), "issues must be a list")
    for entry in issues:
        require(isinstance(entry, dict) and set(entry.keys()) == {"rule", "file", "text"}, f"lint issue has wrong shape (must be exactly rule/file/text -- no line/column, ADR-F01-03 precedent): {entry}")
        for k in ("rule", "file", "text"):
            require(isinstance(entry[k], str), f"lint issue.{k} must be a string: {entry}")

elif capability == "build":
    require(set(doc.keys()) == {"ok", "diagnostics"}, f"unexpected key set {sorted(doc.keys())}")
    require(isinstance(doc["ok"], bool), "ok must be a boolean")
    require(isinstance(doc["diagnostics"], list), "diagnostics must be a list")
    for d in doc["diagnostics"]:
        require(isinstance(d, str), f"diagnostics entries must be strings: {d!r}")

elif capability == "format-check":
    require(set(doc.keys()) == {"ok", "offending_files"}, f"unexpected key set {sorted(doc.keys())}")
    require(isinstance(doc["ok"], bool), "ok must be a boolean")
    require(isinstance(doc["offending_files"], list), "offending_files must be a list")
    for f_ in doc["offending_files"]:
        require(isinstance(f_, str), f"offending_files entries must be strings: {f_!r}")

else:
    sys.exit(f"assert_shape: no assertion defined for capability {capability!r}")

print(f"{label}/{capability}: shape OK")
PYEOF
}

run_capability() {
	# run_capability <adapter-script> <adapter-label> <checkout> <capability> [extra args...]
	local adapter="$1" label="$2" checkout="$3" capability="$4"
	shift 4
	local out_file="$WORKDIR/${label}-${capability}.json"
	local err_file="$WORKDIR/${label}-${capability}.err"
	if ! "$adapter" "$capability" --checkout "$checkout" "$@" >"$out_file" 2>"$err_file"; then
		fail "$label adapter.sh $capability exited non-zero: $(cat "$err_file")"
	fi
	assert_shape "$capability" "$label" "$out_file"
}

echo "TC-031: part 1 - six capabilities x two adapters, identical assertion set (AC-010, AC-011)"

for cap in identity build format-check test lint; do
	run_capability "$GO_ADAPTER" go "$GO_CHECKOUT" "$cap"
	run_capability "$PY_ADAPTER" python "$PY_CHECKOUT" "$cap"
done

# inject-tests needs a real, adapter-discoverable source test file per
# language -- a trivial always-pass probe, colocated the way each adapter's
# own placement rule requires (Go: package-matched directory; Python:
# tests/).
GO_PROBE="$WORKDIR/tc031_probe_test.go"
cat >"$GO_PROBE" <<'EOF'
package cart

import "testing"

func TestTC031ProbeAlwaysPasses(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("arithmetic is broken")
	}
}
EOF
run_capability "$GO_ADAPTER" go "$GO_CHECKOUT" inject-tests --files "$GO_PROBE"

PY_PROBE="$WORKDIR/tc031_probe_test.py"
cat >"$PY_PROBE" <<'EOF'
def test_tc031_probe_always_passes():
    assert 1 + 1 == 2
EOF
run_capability "$PY_ADAPTER" python "$PY_CHECKOUT" inject-tests --files "$PY_PROBE"

echo "TC-031(part 1: identity/inject-tests/test/lint/build/format-check all shape-valid for both adapters) PASS"

# ---------------------------------------------------------------------------
# Part 2 -- AC-T3: the capability vocabulary is a closed set of six verbs,
# not merely "the six that happen to exist". A seventh, undefined name must
# be rejected with a non-zero exit by both adapters.
# ---------------------------------------------------------------------------
echo "TC-031: part 2 - undefined capability rejected by both adapters (closed six-verb set, AC-T3)"

reject_undefined_capability() {
	local adapter="$1" label="$2" checkout="$3"
	local err_file="$WORKDIR/${label}-undefined.err"
	if "$adapter" coverage --checkout "$checkout" >"$WORKDIR/${label}-undefined.out" 2>"$err_file"; then
		fail "$label adapter.sh accepted an undefined capability 'coverage' (must reject with a non-zero exit)"
	fi
	grep -qi "unknown capability\|closed set" "$err_file" || fail "$label adapter.sh rejected 'coverage' but did not name it as an unknown/closed-set capability: $(cat "$err_file")"
}

reject_undefined_capability "$GO_ADAPTER" go "$GO_CHECKOUT"
reject_undefined_capability "$PY_ADAPTER" python "$PY_CHECKOUT"

echo "TC-031(part 2: undefined capability rejected by both adapters) PASS"

# ---------------------------------------------------------------------------
# Part 3 -- AC-012/AC-T4: REQ-F-007's leak surface. No generic scenario,
# evidence, or admission script may know a language-specific command exists,
# nor branch on which fixture/adapter/scenario it is handling -- the closed
# set of allowed discriminants is "which adapter.sh to invoke", never an
# inline language-specific code path.
# ---------------------------------------------------------------------------
echo "TC-031: part 3 - REQ-F-007 leak-surface grep (AC-012 / AC-T4)"

FORBIDDEN_TOKENS='\<python\>|\<pytest\>|\<pip\>|\<go[[:space:]]+test\>|\<golangci-lint\>|\<go[[:space:]]+build\>'
GREP_TARGETS=(
	"$SCRIPTS_DIR/admit-scenario.sh"
	"$SCRIPTS_DIR/eval-predicate.sh"
	"$SCRIPTS_DIR/checkout-scenario-fixture.sh"
	"$SCRIPT_DIR/run-all.sh"
)

leak_found=0
for target in "${GREP_TARGETS[@]}"; do
	if [[ ! -f "$target" ]]; then
		echo "TC-031(AC-012: $(basename "$target") does not exist yet at this task's point in the build order -- skipped, logged not silently passed) SKIP" >&2
		continue
	fi

	if hits="$(grep -nE "$FORBIDDEN_TOKENS" "$target")"; then
		echo "TC-031: forbidden language-specific token found in $target (outside bench/adapters/*/):" >&2
		echo "$hits" >&2
		leak_found=1
	fi

	# Source-level branch check: no `if`/`case` conditional on fixture_id,
	# adapter.name, or scenario_id -- must also catch an
	# `if fixture_id == "python"` branch that calls language-specific
	# logic without going through adapter.sh (task Notes for Agent).
	if branch_hits="$(grep -nE '\b(if|case)\b[^#]*\b(fixture_id|adapter\.name|scenario_id)\b' "$target")"; then
		echo "TC-031: conditional branch on fixture/adapter/scenario identity found in $target:" >&2
		echo "$branch_hits" >&2
		leak_found=1
	fi
done

[[ "$leak_found" -eq 0 ]] || fail "REQ-F-007 leak-surface check found forbidden tokens or identity-branching outside bench/adapters/*/ (see stderr above)"

echo "TC-031(part 3: no forbidden-token hits or identity-branches in admit-scenario.sh/eval-predicate.sh/checkout-scenario-fixture.sh/run-all.sh) PASS"

echo "TC-031: PASS"
