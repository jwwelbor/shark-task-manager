#!/usr/bin/env bash
# checkout-scenario-fixture.sh <fixture_id> <base_sha> <dest_dir>
#
# Sibling of checkout-fixture.sh (REQ-NF-006, ADR-F05-06): resolves
# <fixture_id>'s submodule_path from bench/scenarios/scenarios.yaml and
# clones/checks out that submodule at <base_sha> into <dest_dir>. Multi-
# fixture support lives here, not as a new argument on checkout-fixture.sh
# -- that script's <base_sha> <dest_dir> interface is frozen once
# admit.sh, build-ledgers.sh, and E40-F02 depend on it, and this script
# changes no byte of it (task T-E40-F05-006 Scope Boundary).
#
# Generic over fixtures: which submodule to clone comes entirely from
# scenarios.yaml's fixtures: map, keyed by <fixture_id> -- this script
# never hardcodes "py" or "go" as a recognized identifier, only looks each
# one up.
#
# Intentionally does not call verify-clean-checkout.sh: that script's
# held-back-name leak check is derived from corpus.yaml's F2P holdouts
# (I-01's admitted-item mechanism), which I-04 does not use. I-04's
# held-back oracle material lives under each scenario package's own
# evaluator/ subtree (REQ-F-009), and its green-at-base invariant is
# enforced by admission check (b) (ADR-F05-10), not by grepping a fixture
# checkout.
#
# REQ-NF-003/REQ-NF-005: this script never touches the live shark
# database, .sharkconfig.json, or the live repository working tree, and
# never invokes shark project-initialisation commands -- all work happens
# inside the caller-supplied <dest_dir>.
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

usage() {
	echo "usage: checkout-scenario-fixture.sh <fixture_id> <base_sha> <dest_dir>" >&2
	exit 2
}

[[ $# -eq 3 ]] || usage
fixture_id="$1"
base_sha="$2"
dest_dir="$3"

scenarios_yaml="$BENCH_DIR/scenarios/scenarios.yaml"

[[ -f "$scenarios_yaml" ]] || {
	echo "checkout-scenario-fixture: scenarios yaml not found: $scenarios_yaml" >&2
	exit 1
}
command -v python3 >/dev/null 2>&1 || {
	echo "checkout-scenario-fixture: python3 not found on PATH (required to parse scenarios.yaml)" >&2
	exit 1
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "checkout-scenario-fixture: python3 module 'yaml' (PyYAML) not available (required to parse scenarios.yaml)" >&2
	exit 1
}

submodule_rel="$(python3 - "$scenarios_yaml" "$fixture_id" <<'PYEOF'
import sys
import yaml

scenarios_yaml_path, requested_id = sys.argv[1:3]
with open(scenarios_yaml_path) as f:
    data = yaml.safe_load(f)

fixtures = data.get("fixtures") or {}
entry = fixtures.get(requested_id)
path = (entry or {}).get("submodule_path")
if not path:
    sys.exit(1)
print(path)
PYEOF
)" || {
	echo "checkout-scenario-fixture: fixture_id not registered in $scenarios_yaml: $fixture_id" >&2
	exit 1
}

fixture_submodule="$REPO_ROOT/$submodule_rel"

[[ -e "$fixture_submodule/.git" ]] || {
	echo "checkout-scenario-fixture: $submodule_rel submodule not initialized; run 'git submodule update --init'" >&2
	exit 1
}

[[ -e "$dest_dir" ]] && {
	echo "checkout-scenario-fixture: dest_dir already exists: $dest_dir" >&2
	exit 1
}
mkdir -p "$(dirname "$dest_dir")"

git -c advice.detachedHead=false clone --quiet -- "$fixture_submodule" "$dest_dir"
git -C "$dest_dir" -c advice.detachedHead=false checkout --quiet "$base_sha"
