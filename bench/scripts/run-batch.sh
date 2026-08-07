#!/usr/bin/env bash
# run-batch.sh --out <artifact_root>
#              [--corpus <corpus.yaml>] [--variant <id>]
#              [--reps <n>] [--timeout <seconds>]
#              [--items <id[,id...]>] [--reclaim-incomplete]
#              [--dry-run]
#
# T-E40-F03-001 (spec.md ADR-F03-01, ADR-F03-07): the Phase 1 matrix driver.
# Enumerates every admitted corpus item (bench/corpus/corpus.yaml's `items:`
# ONLY -- REQ-F-005, `negative_items:` are never benched) x the given
# variant (default "default", Phase 1's only variant) x N reps, classifies
# each target directory, invokes run-one.sh for pending pairs only, and
# emits one machine-readable JSON summary to stdout naming every pair's
# classification. Never prompts (REQ-F-001) and never aborts on a single
# pair's failure (REQ-F-004).
#
# Classification (per pair, per ADR-F03-07 -- "directory present,
# record.jsonl absent" is its own explicit state, never folded into "not
# yet run"):
#   skipped_complete         -- record.jsonl already present; not
#                                re-invoked (REQ-F-002: a re-run over a
#                                fully-populated root performs zero
#                                run-one.sh calls and zero API spend)
#   incomplete_prior_attempt -- directory present and non-empty,
#                                record.jsonl absent (REQ-F-003). Default
#                                action: skip, name it, continue -- never
#                                re-invoked into the stale directory.
#   quarantined_and_rerun    -- same state, but --reclaim-incomplete first
#                                MOVED the stale directory under
#                                .incomplete/ (never deleted, REQ-F-006)
#                                before re-invoking into the now-absent
#                                (fresh) target path
#   pending_run              -- no prior directory (or an empty one);
#                                dispatched and run-one.sh exited 0
#   failed                   -- dispatched and run-one.sh exited non-zero
#                                (REQ-F-004: recorded, batch proceeds)
#
# Exit status: 0 when every pair is skipped_complete/pending_run/
# quarantined_and_rerun; non-zero when any pair is `failed` or an
# unreclaimed `incomplete_prior_attempt` was skipped.
#
# Testability (TD-077's defect class; binding decision note on this task):
# run-one.sh is resolved via ${RUN_ONE_BIN:-$SCRIPT_DIR/run-one.sh} -- the
# SIBLING path, never a bare PATH-resolved name -- so the self-test's stub
# (bench/scripts/testdata/stubs/run-one) can substitute it via one
# overridable variable, exactly the pattern run-one.sh itself uses for
# SHARK_BIN/CANARY_BIN.
#
# Record enumeration elsewhere (aggregate-runs.sh, a later task) is pinned
# to the glob "$root"/*/*/rep-*/record.jsonl, which never descends into the
# dot-prefixed .incomplete/ quarantine root -- this script's own quarantine
# path is what keeps that glob's exclusion correct.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RUN_ONE_BIN="${RUN_ONE_BIN:-$SCRIPT_DIR/run-one.sh}"

usage() {
	cat >&2 <<'EOF'
usage: run-batch.sh --out <artifact_root>
                     [--corpus <corpus.yaml>] [--variant <id>]
                     [--reps <n>] [--timeout <seconds>]
                     [--items <id[,id...]>] [--reclaim-incomplete]
                     [--dry-run]
EOF
	exit 2
}

out_root=""
corpus_yaml=""
variant_id="default"
reps="3"
timeout_s="900"
items_filter=""
reclaim_incomplete="false"
dry_run="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--out)
		[[ $# -ge 2 ]] || usage
		out_root="$2"
		shift 2
		;;
	--corpus)
		[[ $# -ge 2 ]] || usage
		corpus_yaml="$2"
		shift 2
		;;
	--variant)
		[[ $# -ge 2 ]] || usage
		variant_id="$2"
		shift 2
		;;
	--reps)
		[[ $# -ge 2 ]] || usage
		reps="$2"
		shift 2
		;;
	--timeout)
		[[ $# -ge 2 ]] || usage
		timeout_s="$2"
		shift 2
		;;
	--items)
		[[ $# -ge 2 ]] || usage
		items_filter="$2"
		shift 2
		;;
	--reclaim-incomplete)
		reclaim_incomplete="true"
		shift
		;;
	--dry-run)
		dry_run="true"
		shift
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$out_root" ]] || usage
corpus_yaml="${corpus_yaml:-$BENCH_DIR/corpus/corpus.yaml}"

command -v python3 >/dev/null 2>&1 || {
	echo "run-batch: python3 not found on PATH" >&2
	exit 1
}
command -v "$RUN_ONE_BIN" >/dev/null 2>&1 || {
	echo "run-batch: run-one.sh binary not found (resolved as '$RUN_ONE_BIN'); build it or set RUN_ONE_BIN" >&2
	exit 1
}
[[ -f "$corpus_yaml" ]] || {
	echo "run-batch: corpus yaml not found: $corpus_yaml" >&2
	exit 1
}
[[ "$reps" =~ ^[0-9]+$ && "$reps" -ge 1 ]] || {
	echo "run-batch: --reps must be a positive integer, got '$reps'" >&2
	exit 1
}

# --- REQ-F-005: the item list comes from corpus.yaml's items: ONLY -- never
# negative_items:. Read once, in the file's own order (the enumeration
# order used below). ---
readarray -t corpus_item_ids < <(python3 - "$corpus_yaml" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

for it in data.get("items") or []:
    print(it["id"])
PYEOF
)
[[ "${#corpus_item_ids[@]}" -gt 0 ]] || {
	echo "run-batch: no items found in $corpus_yaml (items: is empty or missing)" >&2
	exit 1
}

# --items narrows the enumeration to the given ids, preserving corpus.yaml's
# own order. An id that does not resolve in items: -- including any
# negative_items: id -- is silently dropped, never dispatched (REQ-F-005):
# the filter names which ADMITTED items to run, it is not a second corpus.
selected_item_ids=()
if [[ -n "$items_filter" ]]; then
	IFS=',' read -r -a requested_ids <<<"$items_filter"
	for id in "${corpus_item_ids[@]}"; do
		for req in "${requested_ids[@]}"; do
			if [[ "$id" == "$req" ]]; then
				selected_item_ids+=("$id")
				break
			fi
		done
	done
else
	selected_item_ids=("${corpus_item_ids[@]}")
fi

[[ "$dry_run" == "true" ]] || mkdir -p "$out_root"

SUMMARY_TMP="$(mktemp)"
cleanup() { rm -f "$SUMMARY_TMP"; }
trap cleanup EXIT

overall_bad="false"

# append_summary <item_id> <variant_id> <rep> <run_key> <classification>
append_summary() {
	python3 -c '
import json
import sys

item_id, variant_id, rep, run_key, classification = sys.argv[1:6]
print(json.dumps({
    "item_id": item_id,
    "variant_id": variant_id,
    "rep": int(rep),
    "run_key": run_key,
    "classification": classification,
}))
' "$1" "$2" "$3" "$4" "$5" >>"$SUMMARY_TMP"
}

# log_batch_event <item_id> <variant_id> <rep> <run_key> <classification>
# Operator diagnostics only (artifact-root layout: batch-log.jsonl) --
# NEVER read by aggregate-runs.sh (REQ-F-007: it reads only record.jsonl
# files). Skipped entirely under --dry-run, which invokes nothing.
log_batch_event() {
	[[ "$dry_run" == "true" ]] && return 0
	python3 -c '
import json
import sys

item_id, variant_id, rep, run_key, classification = sys.argv[1:6]
print(json.dumps({
    "item_id": item_id,
    "variant_id": variant_id,
    "rep": int(rep),
    "run_key": run_key,
    "classification": classification,
}))
' "$1" "$2" "$3" "$4" "$5" >>"$out_root/batch-log.jsonl"
}

# classify_pair <dir> -- skipped_complete / incomplete_prior_attempt /
# pending.
classify_pair() {
	local dir="$1"
	if [[ -f "$dir/record.jsonl" ]]; then
		echo "skipped_complete"
	elif [[ -d "$dir" ]] && [[ -n "$(ls -A "$dir" 2>/dev/null)" ]]; then
		echo "incomplete_prior_attempt"
	else
		echo "pending"
	fi
}

# quarantine_pair <dir> <item_id> <variant_id> <rep> -- REQ-F-006: a MOVE,
# never a delete, to .incomplete/<item>/<variant>/rep-<n>-<seq>/. After this
# returns, <dir> no longer exists, so the runner never re-invokes the
# driver into a directory already holding a prior attempt's run/ or post/
# (REQ-F-003).
quarantine_pair() {
	local dir="$1" item_id="$2" variant_id="$3" rep="$4"
	local incomplete_root="$out_root/.incomplete/$item_id/$variant_id"
	mkdir -p "$incomplete_root"
	local seq=1
	while [[ -e "$incomplete_root/rep-$rep-$seq" ]]; do
		seq=$((seq + 1))
	done
	local dest="$incomplete_root/rep-$rep-$seq"
	mv "$dir" "$dest"
	echo "run-batch: quarantined incomplete prior attempt: $dir -> $dest" >&2
}

# dispatch_pair <item_id> <variant_id> <rep> <run_key> <success_label>
# Invokes RUN_ONE_BIN for a pending pair (REQ-F-001). A non-zero exit is
# recorded as `failed` and the batch proceeds to the next pair (REQ-F-004)
# -- one pair's failure never aborts the loop.
dispatch_pair() {
	local item_id="$1" variant_id="$2" rep="$3" run_key="$4" success_label="$5"

	if [[ "$dry_run" == "true" ]]; then
		append_summary "$item_id" "$variant_id" "$rep" "$run_key" "$success_label"
		return 0
	fi

	echo "run-batch: dispatching $run_key" >&2
	set +e
	"$RUN_ONE_BIN" --item "$item_id" --variant "$variant_id" --rep "$rep" \
		--timeout "$timeout_s" --out "$out_root" --corpus "$corpus_yaml" \
		</dev/null
	local rc=$?
	set -e

	if [[ "$rc" -eq 0 ]]; then
		append_summary "$item_id" "$variant_id" "$rep" "$run_key" "$success_label"
		log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "$success_label"
	else
		echo "run-batch: $run_key FAILED (RUN_ONE_BIN exit $rc)" >&2
		append_summary "$item_id" "$variant_id" "$rep" "$run_key" "failed"
		log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "failed"
		overall_bad="true"
	fi
}

for item_id in "${selected_item_ids[@]}"; do
	for ((rep = 1; rep <= reps; rep++)); do
		dir="$out_root/$item_id/$variant_id/rep-$rep"
		run_key="${item_id}::${variant_id}::rep${rep}"
		classification="$(classify_pair "$dir")"

		case "$classification" in
		skipped_complete)
			append_summary "$item_id" "$variant_id" "$rep" "$run_key" "skipped_complete"
			;;
		incomplete_prior_attempt)
			if [[ "$reclaim_incomplete" == "true" ]]; then
				[[ "$dry_run" == "true" ]] || quarantine_pair "$dir" "$item_id" "$variant_id" "$rep"
				dispatch_pair "$item_id" "$variant_id" "$rep" "$run_key" "quarantined_and_rerun"
			else
				echo "run-batch: $run_key is an incomplete prior attempt (directory present, record.jsonl absent); skipping (pass --reclaim-incomplete to quarantine and re-run)" >&2
				append_summary "$item_id" "$variant_id" "$rep" "$run_key" "incomplete_prior_attempt"
				log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "incomplete_prior_attempt"
				overall_bad="true"
			fi
			;;
		pending)
			dispatch_pair "$item_id" "$variant_id" "$rep" "$run_key" "pending_run"
			;;
		esac
	done
done

python3 - "$SUMMARY_TMP" "$corpus_yaml" "$out_root" "$variant_id" "$reps" "$dry_run" "$reclaim_incomplete" <<'PYEOF'
import json
import sys

tmp_path, corpus, out_root, variant_id, reps, dry_run, reclaim = sys.argv[1:8]

pairs = []
with open(tmp_path) as f:
    for line in f:
        line = line.strip()
        if line:
            pairs.append(json.loads(line))

counts = {}
for p in pairs:
    counts[p["classification"]] = counts.get(p["classification"], 0) + 1

summary = {
    "corpus": corpus,
    "counts": counts,
    "dry_run": dry_run == "true",
    "out_root": out_root,
    "pairs": pairs,
    "reclaim_incomplete": reclaim == "true",
    "reps": int(reps),
    "variant_id": variant_id,
}
print(json.dumps(summary, indent=2, sort_keys=True))
PYEOF

if [[ "$overall_bad" == "true" ]]; then
	exit 1
fi
exit 0
