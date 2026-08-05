#!/usr/bin/env bash
# verify-clean-checkout.sh <checkout_dir> <corpus_yaml_path>
#
# Proves held-back tests never leak into a checkout (REQ-F-004, AC-001).
# Derives every held-back F2P file name and test function name across all
# admitted (items) and negative (negative_items) candidates in
# <corpus_yaml_path>, greps <checkout_dir> for each, fails naming the first
# hit, and prints CLEAN on stdout on success.
#
# Names are always derived from corpus.yaml at call time -- never from a
# hardcoded list -- per REQ-F-004 and test-plan.md TC-003's Caller-Path
# Contract.
#
# Corpus-authoring constraint: an F2P file name or test function name that
# collides with a name already committed in the fixture repo would make
# this script report a false leak on every clean checkout -- and because
# checkout-fixture.sh calls this script unconditionally on its own exit
# path, that failure propagates to every caller (admit.sh, build-ledgers.sh,
# E40-F02), not just to a direct invocation. Future corpus items must pick
# F2P names that do not collide with any name already present in the
# fixture at fixture.base_sha.
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

usage() {
	echo "usage: verify-clean-checkout.sh <checkout_dir> <corpus_yaml_path>" >&2
	exit 2
}

[[ $# -eq 2 ]] || usage
checkout_dir="$1"
corpus_yaml_path="$2"

[[ -d "$checkout_dir" ]] || {
	echo "verify-clean-checkout: checkout dir not found: $checkout_dir" >&2
	exit 1
}
[[ -f "$corpus_yaml_path" ]] || {
	echo "verify-clean-checkout: corpus yaml not found: $corpus_yaml_path" >&2
	exit 1
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-clean-checkout: python3 not found on PATH (required to parse corpus.yaml)" >&2
	exit 1
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "verify-clean-checkout: python3 module 'yaml' (PyYAML) not available (required to parse corpus.yaml)" >&2
	exit 1
}

names="$(python3 - "$corpus_yaml_path" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

names = set()
for section in ("items", "negative_items"):
    for item in data.get(section) or []:
        f2p = item.get("f2p") or {}
        for path in f2p.get("paths") or []:
            names.add(path.rsplit("/", 1)[-1])
        for test_name in f2p.get("test_names") or []:
            names.add(test_name.rsplit("::", 1)[-1])

for name in sorted(names):
    print(name)
PYEOF
)"

if [[ -z "$names" ]]; then
	echo "verify-clean-checkout: no held-back names derived from $corpus_yaml_path" >&2
	exit 1
fi

while IFS= read -r name; do
	[[ -n "$name" ]] || continue
	hit="$(grep -rIl --exclude-dir=.git -F -- "$name" "$checkout_dir" 2>/dev/null | head -n1 || true)"
	if [[ -n "$hit" ]]; then
		echo "verify-clean-checkout: held-back name leaked: $name (found in $hit)" >&2
		exit 1
	fi
done <<<"$names"

echo "CLEAN"
