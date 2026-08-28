#!/usr/bin/env bash
# tests/lib/transient-run-selector-corpus.sh -- shared transient-corpus
# builder for tc096/tc097/tc098's `run_selector` regression tests (B053).
# Each of those tests needs a copy of the real, committed corpus.yaml with
# one addition -- a new p2p_set scoped to a single package glob and a
# caller-supplied run_selector -- and the already-admitted
# "pricing-negative-subtotal" item reassigned to that set (REQ-F-007: no
# committed corpus.yaml entry is touched; this is the same transient-item
# technique tc005 already uses). Extracted here once so three tests don't
# carry three near-identical copies of the same ~35-line Python heredoc
# (code review finding, B053 PR #203).
#
# Caller contract:
#   source ".../tests/lib/transient-run-selector-corpus.sh"
#   build_transient_run_selector_corpus \
#     "$CORPUS_YAML" "$TRANSIENT_CORPUS" "$TRANSIENT_ITEM" "$TRANSIENT_SET" \
#     "$RUN_SELECTOR" "$PACKAGE_GLOB"

build_transient_run_selector_corpus() {
	local src_corpus="$1" dst_corpus="$2" item_id="$3" set_name="$4"
	local run_selector="$5" package_glob="$6"

	python3 - "$src_corpus" "$dst_corpus" "$item_id" "$set_name" "$run_selector" "$package_glob" <<'PYEOF'
import os
import sys
import yaml

src_path, dst_path, item_id, set_name, run_selector, package_glob = sys.argv[1:7]

with open(src_path) as f:
    data = yaml.safe_load(f)

if item_id not in {it["id"] for it in data["items"]}:
    sys.exit(f"transient-run-selector-corpus setup: item not found in corpus.yaml: {item_id}")

data["p2p_sets"][set_name] = {
    "packages": [package_glob],
    "run_selector": run_selector,
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
}
