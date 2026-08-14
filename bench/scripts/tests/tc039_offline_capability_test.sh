#!/usr/bin/env bash
# TC-039 (test-plan.md AC test matrix; T-E40-F05-015 task spec Test Cases).
#
# Exercises AC-018 (REQ-NF-004): with dependency caches warm, the admission
# gate (bench/scripts/admit-scenario.sh) completes offline over all four
# REQ-F-014 seed packages, run under TWO independent network-isolation
# mechanisms so neither is merely an assumed-equivalent, untested fallback --
#   1. `unshare --user --net` -- a real Linux network-namespace block. Works
#      unprivileged on this host without CAP_SYS_ADMIN in the parent
#      namespace, the same instantiation TC-013 (F01's offline test)
#      established in this repo for bare `unshare --net`, which requires a
#      privilege this environment does not have.
#   2. The portable `GOPROXY=off GOFLAGS=-mod=readonly PIP_NO_INDEX=1`
#      fallback, plus a poisoned `PIP_INDEX_URL`/`http_proxy`/`https_proxy`
#      pointed at a guaranteed-closed local port -- mirrors F01's TC-013
#      technique, extended to `pip` per test-plan.md's own note.
#
# Both mechanisms are exercised independently, proving the portable fallback
# genuinely works rather than merely existing as an assumed equivalent to
# the Linux-only namespace block.
#
# Cache-warm precondition: the Python environment under test already has
# the pinned pytest/ruff/black installed (this script's own PATH, or the
# caller's) -- bench/adapters/python/adapter.sh's own header states it
# "provisions nothing" at runtime (no pip install, no download), and
# checkout-scenario-fixture.sh clones the ALREADY-INITIALIZED local
# bench/fixture-py submodule directory (a local filesystem clone, never a
# remote URL) -- so REQ-NF-004's "once the fixtures and their dependency
# caches are present" is satisfied by this checkout's own state with no
# separate warming step required.
#
# Caller-Path Contract (test-plan.md TC-039): real subprocess execution
# with genuine network isolation (a real Linux namespace, and a real
# poisoned-proxy environment) -- never a code-level "offline mode" flag.
# Does not stub any network call; does not let a missing pinned
# interpreter/package fall back to a network install.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"
FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"

fail() {
	echo "TC-039 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"
[[ -e "$FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
python3 -m pytest --version >/dev/null 2>&1 || fail "pytest not runnable on PATH in the test environment (cache-warm precondition)"
command -v ruff >/dev/null 2>&1 || fail "ruff not found on PATH in the test environment (cache-warm precondition)"
command -v black >/dev/null 2>&1 || fail "black not found on PATH in the test environment (cache-warm precondition)"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

mapfile -t package_rel_dirs < <(python3 - "$SCENARIOS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for rel in sorted(data["scenarios"]):
    print(rel)
PYEOF
)
[[ "${#package_rel_dirs[@]}" -eq 4 ]] || fail "expected exactly 4 registered scenarios in scenarios.yaml, found ${#package_rel_dirs[@]}"

network_attempt_pattern='dial tcp|connection refused|no such host|network is unreachable|could not resolve|temporary failure in name resolution|NewConnectionError|ConnectTimeoutError|Failed to establish a new connection'

# run_offline <mech_label> <net_wrap_cmd...> -- runs admit-scenario.sh
# against a fresh candidate copy of each of the four seeds under the given
# isolation wrapper (a prefix argv array, possibly empty for the
# environment-variable-only portable fallback). Asserts every seed is
# admitted (exit 0) with no outbound-network-attempt pattern on stderr.
run_offline() {
	local mech_label="$1"
	shift
	local net_wrap=("$@")

	local rel_dir scenario_label candidate_dir out_file err_file code
	for rel_dir in "${package_rel_dirs[@]}"; do
		scenario_label="$(basename "$rel_dir")"
		candidate_dir="$WORKDIR/$mech_label/candidate-$scenario_label"
		mkdir -p "$(dirname "$candidate_dir")"
		cp -r "$BENCH_DIR/scenarios/$rel_dir" "$candidate_dir"

		out_file="$WORKDIR/$mech_label/$scenario_label.out"
		err_file="$WORKDIR/$mech_label/$scenario_label.err"
		code=0
		set +e
		if [[ "${#net_wrap[@]}" -gt 0 ]]; then
			"${net_wrap[@]}" "$ADMIT_SCRIPT" "$candidate_dir/package.yaml" >"$out_file" 2>"$err_file"
		else
			"$ADMIT_SCRIPT" "$candidate_dir/package.yaml" >"$out_file" 2>"$err_file"
		fi
		code=$?
		set -e

		[[ "$code" -eq 0 ]] || fail "$mech_label: $scenario_label: admit-scenario.sh exited $code (expected 0, admitted, offline); stderr: $(cat "$err_file")"
		if grep -qiE "$network_attempt_pattern" "$err_file"; then
			fail "$mech_label: $scenario_label: stderr shows an outbound network attempt: $(cat "$err_file")"
		fi
		echo "TC-039: $mech_label: $scenario_label admitted offline"
	done
}

echo "TC-039: mechanism 1 - unshare --user --net (real Linux network-namespace block)"
if unshare --user --net -- true >/dev/null 2>&1; then
	run_offline "unshare-net" unshare --user --net --
else
	fail "unshare --user --net is unavailable on this host -- TC-039 requires both isolation mechanisms to run independently, not a fallback to only one"
fi

echo "TC-039: mechanism 2 - portable GOPROXY=off/PIP_NO_INDEX=1 fallback with a poisoned proxy"
run_offline "portable-fallback" env \
	GOPROXY=off \
	GOFLAGS=-mod=readonly \
	GOSUMDB=off \
	PIP_NO_INDEX=1 \
	PIP_INDEX_URL="http://127.0.0.1:9/simple" \
	http_proxy="http://127.0.0.1:9" \
	https_proxy="http://127.0.0.1:9" \
	HTTP_PROXY="http://127.0.0.1:9" \
	HTTPS_PROXY="http://127.0.0.1:9"

echo "TC-039: both isolation mechanisms independently completed the admission gate over all four seeds, offline"
echo "TC-039: PASS"
