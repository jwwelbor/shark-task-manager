#!/usr/bin/env bash
# TC-035 (test-plan.md AC test matrix; T-E40-F05-015 task spec Test Cases).
#
# Exercises AC-013 (REQ-F-008): two real, pinned, cache-warmed Python
# interpreter environments produce two genuinely different ordered
# `toolchain_identity` key/value lists, and a full
# `admit-scenario.sh py-bug-due-date-boundary` re-run under each records
# the matching identity end-to-end -- proving the pin propagates from the
# adapter's own `identity` capability all the way through admission, not
# merely at the capability boundary.
#
# The two environments here are Python 3.11 and Python 3.12 (both
# provisioned via `uv venv --python <ver>`, the same real dependency-lock
# mechanism the fixture uses -- never a PATH-stubbed `python3`/`pip`
# reporting a fabricated version string), each with the identical pinned
# `pytest`/`ruff`/`black` versions the committed seed packages were admitted
# under (test-plan.md's "e.g. a second Python minor version...alongside the
# pinned 3.12" is exemplary, not mandatory: the property under test is that
# the two environments are real, pinned, and produce genuinely different
# lists -- a differing python_version key alone already establishes that
# without needing a differing tool version too). bench/fixture-py's own
# pyproject.toml declares `requires-python = ">=3.11"`, so 3.11 is a
# legitimate second environment for the fixture, not merely for the
# adapter's own identity capability.
#
# This test provisions both venvs itself via `uv` at test time -- it does
# not depend on any venv the surrounding quality-gate invocation may already
# have on PATH -- so it is self-contained and re-runnable standalone.
#
# Caller-Path Contract (test-plan.md TC-035): real subprocess invocation of
# `bench/adapters/python/adapter.sh identity` and a full
# `admit-scenario.sh` re-run under each environment, reporting the actual
# live interpreter/tooling versions. Neither expected identity list is
# hardcoded; both are derived from live tool invocation. The comparison
# between the two lists is equality-only over the whole ordered list --
# never a read of a single named key (AC-013's own point).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
PYTHON_ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
REAL_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-bug-due-date-boundary"

fail() {
	echo "TC-035 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -x "$PYTHON_ADAPTER" ]] || fail "python adapter.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
[[ -f "$REAL_PACKAGE_DIR/package.yaml" ]] || fail "py-bug-due-date-boundary/package.yaml missing"
command -v uv >/dev/null 2>&1 || fail "uv not found on PATH (required to provision two real, pinned Python environments)"

# The pinned tool versions the committed seed packages were admitted under
# (bench/scenarios/packages/py-bug-due-date-boundary/package.yaml's own
# toolchain_identity block) -- read from the real committed file, never
# hand-duplicated, so a future re-pin is picked up automatically.
PINNED_VERSIONS="$(python3 - "$REAL_PACKAGE_DIR/package.yaml" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    package = yaml.safe_load(f)
identity = {e["key"]: e["value"] for e in package["toolchain_identity"]}
print(identity["pytest_version"])
print(identity["ruff_version"])
print(identity["black_version"])
PYEOF
)"
PYTEST_VERSION="$(sed -n '1p' <<<"$PINNED_VERSIONS")"
RUFF_VERSION="$(sed -n '2p' <<<"$PINNED_VERSIONS")"
BLACK_VERSION="$(sed -n '3p' <<<"$PINNED_VERSIONS")"
[[ -n "$PYTEST_VERSION" && -n "$RUFF_VERSION" && -n "$BLACK_VERSION" ]] || fail "could not derive pinned pytest/ruff/black versions from the committed package.yaml"

BASE_SHA="$(python3 - "$REAL_PACKAGE_DIR/package.yaml" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    print(yaml.safe_load(f)["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$BASE_SHA" ]] || fail "could not derive base_sha from the committed package.yaml"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# provision_env <label> <python_version> -> prints the venv's bin dir.
# Real `uv venv --python <ver>` provisioning, real `uv pip install` of the
# pinned tool versions -- never a stub binary standing in for the
# interpreter or its tooling.
provision_env() {
	local label="$1" pyver="$2"
	local venv_dir="$WORKDIR/venv-$label"
	uv venv --python "$pyver" "$venv_dir" >"$WORKDIR/uv-venv-$label.log" 2>&1 \
		|| fail "$label: uv venv --python $pyver failed: $(cat "$WORKDIR/uv-venv-$label.log")"
	uv pip install --python "$venv_dir/bin/python3" \
		"pytest==$PYTEST_VERSION" "ruff==$RUFF_VERSION" "black==$BLACK_VERSION" pyyaml \
		>"$WORKDIR/uv-pip-$label.log" 2>&1 \
		|| fail "$label: uv pip install failed: $(cat "$WORKDIR/uv-pip-$label.log")"
	echo "$venv_dir/bin"
}

echo "TC-035: provisioning two real, pinned Python environments (3.11, 3.12)"
ENV_A_BIN="$(provision_env a 3.11)"
ENV_B_BIN="$(provision_env b 3.12)"

# capture_identity <label> <env_bin_dir> -> writes the identity capability's
# JSON to a file and echoes its path. Real subprocess invocation of the
# adapter under each environment's own PATH -- the environment switch is a
# real venv, not a PATH-stubbed binary reporting a fabricated version.
capture_identity() {
	local label="$1" env_bin="$2"
	local checkout="$WORKDIR/identity-checkout-$label"
	"$CHECKOUT_SCRIPT" py "$BASE_SHA" "$checkout" >/dev/null
	local out="$WORKDIR/identity-$label.json"
	PATH="$env_bin:$PATH" "$PYTHON_ADAPTER" identity --checkout "$checkout" >"$out" \
		|| fail "$label: adapter.sh identity failed under this environment"
	echo "$out"
}

echo "TC-035: capturing identity under environment A (Python 3.11)"
IDENTITY_A="$(capture_identity a "$ENV_A_BIN")"
echo "TC-035: capturing identity under environment B (Python 3.12)"
IDENTITY_B="$(capture_identity b "$ENV_B_BIN")"

python3 - "$IDENTITY_A" "$IDENTITY_B" <<'PYEOF'
import json
import sys

path_a, path_b = sys.argv[1:3]
with open(path_a) as f:
    doc_a = json.load(f)
with open(path_b) as f:
    doc_b = json.load(f)

list_a = doc_a["toolchain_identity"]
list_b = doc_b["toolchain_identity"]

# Equality-only comparison of the WHOLE ordered list -- never a read of a
# single named key (AC-013's own point: a consumer indexing
# toolchain_identity[0].value would still "work" by accident here, which is
# exactly why the comparison must be over the whole list).
if list_a == list_b:
    sys.exit(f"TC-035 FAIL: the two real environments produced IDENTICAL toolchain_identity lists: {list_a} -- the environments are not genuinely different, or the adapter never actually interrogated the live toolchain")
print(f"TC-035: environment A identity: {list_a}")
print(f"TC-035: environment B identity: {list_b}")
print("TC-035: the two real, pinned environments produced genuinely different ordered toolchain_identity lists")
PYEOF

echo "TC-035: full admit-scenario.sh re-run under environment A"
CAND_A="$WORKDIR/candidate-a/package.yaml"
mkdir -p "$(dirname "$CAND_A")"
cp -r "$REAL_PACKAGE_DIR"/. "$(dirname "$CAND_A")"
PATH="$ENV_A_BIN:$PATH" "$ADMIT_SCRIPT" "$CAND_A" >"$WORKDIR/admit-a.out" 2>"$WORKDIR/admit-a.err" \
	|| fail "environment A: admit-scenario.sh did not admit the candidate (expected 0): $(cat "$WORKDIR/admit-a.err")"

echo "TC-035: full admit-scenario.sh re-run under environment B"
CAND_B="$WORKDIR/candidate-b/package.yaml"
mkdir -p "$(dirname "$CAND_B")"
cp -r "$REAL_PACKAGE_DIR"/. "$(dirname "$CAND_B")"
PATH="$ENV_B_BIN:$PATH" "$ADMIT_SCRIPT" "$CAND_B" >"$WORKDIR/admit-b.out" 2>"$WORKDIR/admit-b.err" \
	|| fail "environment B: admit-scenario.sh did not admit the candidate (expected 0): $(cat "$WORKDIR/admit-b.err")"

# assert_end_to_end_identity <verdict_out> <candidate_package_yaml>
# <identity_json> <label> -- shared by both environments below so the
# assertion logic itself is written once, not duplicated per environment.
assert_end_to_end_identity() {
	local verdict_path="$1" package_path="$2" identity_path="$3" label="$4"
	python3 - "$verdict_path" "$package_path" "$identity_path" "$label" <<'PYEOF'
import json
import sys

import yaml

verdict_path, package_path, identity_path, label = sys.argv[1:5]

with open(verdict_path) as f:
    verdict = json.loads(f.read().strip())
with open(package_path) as f:
    package = yaml.safe_load(f)
with open(identity_path) as f:
    expected_identity = json.load(f)["toolchain_identity"]

if verdict["toolchain_identity"] != expected_identity:
    sys.exit(f"TC-035 FAIL: environment {label}: admit-scenario.sh's own verdict toolchain_identity does not match the identity capability's own list: {verdict['toolchain_identity']} != {expected_identity}")
if package.get("admission", {}).get("toolchain_identity") != expected_identity:
    sys.exit(f"TC-035 FAIL: environment {label}: package.yaml's written admission.toolchain_identity does not match environment {label}'s own identity list")
print(f"TC-035: environment {label}: admission.toolchain_identity end-to-end matches the environment's own identity capability output")
PYEOF
}

assert_end_to_end_identity "$WORKDIR/admit-a.out" "$CAND_A" "$IDENTITY_A" "a"
assert_end_to_end_identity "$WORKDIR/admit-b.out" "$CAND_B" "$IDENTITY_B" "b"

echo "TC-035: PASS"
