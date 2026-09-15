#!/usr/bin/env bash
# TD-130: corpus-declared source and reference-patch paths must remain inside
# the corpus directory before admission can read or copy them.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CORPUS="$BENCH_DIR/corpus/corpus.yaml"

fail() { echo "TC-117 FAIL: $1" >&2; exit 1; }
[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing"

WORKDIR="$(mktemp -d)"
EXTERNAL_DIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR" "$EXTERNAL_DIR"' EXIT

printf 'outside patch\n' >"$EXTERNAL_DIR/external.patch"
printf 'outside test\n' >"$EXTERNAL_DIR/external.go"
ln -s "$EXTERNAL_DIR/external.patch" "$WORKDIR/escaped.patch"
ln -s "$EXTERNAL_DIR/external.go" "$WORKDIR/escaped.go"

for mode in traversal absolute symlink; do
	for field in reference_patch_path f2p_path; do
		variant="$WORKDIR/$mode-$field.yaml"
		python3 - "$CORPUS" "$variant" "$field" "$mode" "$WORKDIR" "$EXTERNAL_DIR" <<'PYEOF'
import sys
import yaml

source, target, field, mode, workdir, external_dir = sys.argv[1:]
with open(source, encoding="utf-8") as stream:
    corpus = yaml.safe_load(stream)
item = corpus["items"][0]
if mode == "traversal":
    value = "../" + external_dir.rsplit("/", 1)[-1] + ("/external.patch" if field == "reference_patch_path" else "/external.go")
elif mode == "absolute":
    value = external_dir + ("/external.patch" if field == "reference_patch_path" else "/external.go")
else:
    value = "escaped.patch" if field == "reference_patch_path" else "escaped.go"
if field == "reference_patch_path":
    item["reference_patch_path"] = value
else:
    item["f2p"]["paths"][0] = value
with open(target, "w", encoding="utf-8") as stream:
    yaml.safe_dump(corpus, stream, sort_keys=False)
PYEOF

	set +e
	"$ADMIT_SCRIPT" "$variant" --item "$(python3 - "$CORPUS" <<'PYEOF'
import sys, yaml
with open(sys.argv[1], encoding="utf-8") as stream:
    print(yaml.safe_load(stream)["items"][0]["id"])
PYEOF
)" >"$WORKDIR/$field.out" 2>"$WORKDIR/$field.err"
	rc=$?
	set -e
		[[ $rc -eq 2 ]] || fail "$mode $field escape exit=$rc, want 2: $(cat "$WORKDIR/$field.err")"
		[[ ! -s "$WORKDIR/$field.out" ]] || fail "$mode $field escape emitted a verdict"
		[[ -s "$WORKDIR/$field.err" ]] || fail "$mode $field escape produced no diagnostic"
	done
done

echo "TC-117: corpus path containment PASS"
