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

# --- R2-F-13 fix (code-review-20260808-235300-E40-F03.md /
# uat-20260808-231500-E40-F03.md): item_id (corpus.yaml-sourced, never
# typed by the operator at the point of use) and variant_id (directly
# operator-supplied) are both interpolated into constructed filesystem
# paths below -- reads (classify_pair's ls/-f checks) AND destructive
# moves (quarantine_pair's mv). Neither was validated. A safe-slug
# grammar, checked at parse time before either value reaches any path
# construction, structurally forbids '/' (no traversal component) and a
# leading '.' (so neither '.' nor '..' can ever match) -- REQ-F-006. ---
safe_slug_re='^[A-Za-z0-9][A-Za-z0-9._-]*$'
validate_slug() {
	local value="$1" label="$2"
	[[ "$value" =~ $safe_slug_re ]] || {
		echo "run-batch: $label '$value' is not a safe path identifier (must match $safe_slug_re -- no '/', no leading '.')" >&2
		return 1
	}
}
validate_slug "$variant_id" "--variant" || exit 1

# Belt-and-braces containment check (R2-F-13 fix guidance, in addition to
# the grammar check above): every path this script constructs from
# item_id/variant_id/rep is canonicalized with `realpath -m` and asserted
# beneath the canonicalized --out root before any mkdir/ls/mv/cp/dispatch.
# `realpath -m` never requires the path to exist, so this works even
# before $out_root itself has been created (--dry-run never creates it).
#
# assert_within_out_root() itself is lifted into lib/path-safety.sh, shared
# with replay-manifest.sh (NEW-1, UAT round 3) -- one implementation so the
# two callers cannot drift apart again.
out_root_canon="$(realpath -m -- "$out_root")"
# shellcheck source=lib/path-safety.sh
source "$SCRIPT_DIR/lib/path-safety.sh"

# --- REQ-F-005: the item list comes from corpus.yaml's items: ONLY -- never
# negative_items:. Read once, in the file's own order (the enumeration
# order used below).
#
# F-2 fix (UAT uat-20260808-E40-F03.md): a process-substitution
# (`< <(...)`) exit status is invisible to readarray -- a corpus.yaml with a
# malformed items: entry (e.g. missing id) makes the embedded Python raise
# partway through, and readarray silently keeps whatever partial output it
# already got, truncating the matrix while the batch still exits 0. Capture
# the enumeration to a temp file instead, check python3's own exit status
# explicitly, and independently assert the enumerated count equals
# corpus.yaml's declared items: length -- a second guard against any future
# enumeration path that could truncate silently while still exiting 0. ---
corpus_items_tmp="$(mktemp)"
set +e
python3 - "$corpus_yaml" >"$corpus_items_tmp" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

for it in data.get("items") or []:
    print(it["id"])
PYEOF
enumerate_rc=$?
set -e
if [[ "$enumerate_rc" -ne 0 ]]; then
	rm -f "$corpus_items_tmp"
	echo "run-batch: failed to enumerate corpus items from $corpus_yaml (python3 exited $enumerate_rc; corpus.yaml may be malformed, e.g. an items: entry missing 'id')" >&2
	exit 1
fi
readarray -t corpus_item_ids <"$corpus_items_tmp"
rm -f "$corpus_items_tmp"

declared_item_count="$(python3 -c '
import sys

import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

print(len(data.get("items") or []))
' "$corpus_yaml")"

[[ "${#corpus_item_ids[@]}" -eq "$declared_item_count" ]] || {
	echo "run-batch: enumerated ${#corpus_item_ids[@]} corpus item id(s) but corpus.yaml declares $declared_item_count under items: (enumeration truncated or malformed)" >&2
	exit 1
}

[[ "${#corpus_item_ids[@]}" -gt 0 ]] || {
	echo "run-batch: no items found in $corpus_yaml (items: is empty or missing)" >&2
	exit 1
}

# R2-F-13 fix: every enumerated corpus item id is validated against the
# safe-slug grammar as early as possible -- right after enumeration,
# before any of them reach --items intersection or path construction.
for id in "${corpus_item_ids[@]}"; do
	validate_slug "$id" "corpus.yaml items: id" || exit 1
done

# --items narrows the enumeration to the given ids, preserving corpus.yaml's
# own order. An id that does not resolve in items: -- including any
# negative_items: id -- is dropped, never dispatched (REQ-F-005): the filter
# names which ADMITTED items to run, it is not a second corpus. F-5 fix (UAT
# uat-20260808-E40-F03.md): a drop is no longer silent -- every unresolved
# id is named on stderr, and if none resolve the batch fails loud instead
# of reporting a clean success over an empty matrix.
#
# R2-F-5 fix (UAT uat-20260808-231500-E40-F03.md): the stderr warning above
# named the drop to a human, but a caller redirecting stdout to
# batch-summary.json (the documented usage) and discarding stderr never
# saw it -- requested_ids/unresolved_ids are captured here (declared before
# the if so they stay defined, as empty arrays, when --items is not given
# at all) and carried into the JSON summary object below.
selected_item_ids=()
requested_ids=()
unresolved_ids=()
if [[ -n "$items_filter" ]]; then
	IFS=',' read -r -a requested_ids <<<"$items_filter"

	# R2-F-15 fix (code-review-20260808-235300-E40-F03.md /
	# uat-20260808-231500-E40-F03.md): de-duplicate requested_ids BEFORE
	# intersecting against corpus_item_ids -- filter resolution intersects
	# as a SET, matching this comment block's own long-standing promise to
	# "narrow the enumeration ... preserving corpus.yaml's own order". A
	# repeated --items id must resolve to exactly one selected pair, never
	# two sharing one run_key (REQ-F-001: one declared pair costs at most
	# one dispatch).
	declare -A seen_requested=()
	deduped_requested_ids=()
	for req in "${requested_ids[@]}"; do
		if [[ -z "${seen_requested[$req]+x}" ]]; then
			seen_requested["$req"]=1
			deduped_requested_ids+=("$req")
		fi
	done
	requested_ids=("${deduped_requested_ids[@]}")

	for req in "${requested_ids[@]}"; do
		resolved="false"
		for id in "${corpus_item_ids[@]}"; do
			if [[ "$id" == "$req" ]]; then
				selected_item_ids+=("$id")
				resolved="true"
				break
			fi
		done
		[[ "$resolved" == "true" ]] || unresolved_ids+=("$req")
	done

	# F-5 fix (UAT uat-20260808-E40-F03.md): name every requested id that did
	# not resolve, even when others did -- a typo'd id sitting alongside
	# valid ones must not pass silently as a smaller-than-intended matrix.
	for req in "${unresolved_ids[@]}"; do
		echo "run-batch: --items id '$req' did not resolve against corpus.yaml's admitted items: (not present, or present only under negative_items:)" >&2
	done

	[[ "${#selected_item_ids[@]}" -gt 0 ]] || {
		echo "run-batch: --items '$items_filter' resolved to zero admitted items; nothing to run" >&2
		exit 1
	}
else
	selected_item_ids=("${corpus_item_ids[@]}")
fi

[[ "$dry_run" == "true" ]] || mkdir -p "$out_root"

SUMMARY_TMP="$(mktemp)"
cleanup() { rm -f "$SUMMARY_TMP"; }
trap cleanup EXIT

overall_bad="false"

# append_summary <item_id> <variant_id> <rep> <run_key> <classification>
#
# R2-F-8 sweep (UAT uat-20260808-231500-E40-F03.md fix guidance #2): a
# failing write here (e.g. $SUMMARY_TMP's filesystem going away mid-batch)
# is an unguarded plain statement at every call site -- same set -e abort
# exposure as classify_pair's bare command substitution. Guarded here, once,
# so every call site is covered without touching each one individually.
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
' "$1" "$2" "$3" "$4" "$5" >>"$SUMMARY_TMP" || {
		echo "run-batch: WARNING: failed to append summary entry for run_key=$4 classification=$5; batch proceeds" >&2
		overall_bad="true"
	}
}

# log_batch_event <item_id> <variant_id> <rep> <run_key> <classification>
# Operator diagnostics only (artifact-root layout: batch-log.jsonl) --
# NEVER read by aggregate-runs.sh (REQ-F-007: it reads only record.jsonl
# files). Skipped entirely under --dry-run, which invokes nothing.
#
# R2-F-8 sweep: guarded the same way as append_summary above -- a failing
# write here (e.g. $out_root itself has become unwritable) must not abort
# the batch; it is diagnostics-only, so it warns rather than setting
# overall_bad.
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
' "$1" "$2" "$3" "$4" "$5" >>"$out_root/batch-log.jsonl" || echo "run-batch: WARNING: failed to write batch-log.jsonl entry for run_key=$4 classification=$5 (diagnostics only; batch proceeds)" >&2
}

# classify_pair <dir> -- skipped_complete / incomplete_prior_attempt /
# classification_error / pending.
#
# Sweep fix (UAT uat-20260808-E40-F03.md, F-2/F-5 defect class): a bare
# `ls -A "$dir" 2>/dev/null` cannot distinguish "directory is empty" from
# "directory contents could not be listed" (e.g. a permission-denied dir) --
# both print nothing, and the swallowed stderr let the unreadable case fall
# through to `pending`, dispatching run-one.sh into a directory whose actual
# contents were never checked. List explicitly and name the unreadable case
# instead of defaulting to "empty".
#
# R2-F-8 fix (UAT uat-20260808-231500-E40-F03.md): the unreadable case used
# to `return 1`. That is reached ONLY via a bare command substitution
# (`classification="$(classify_pair "$dir")"` in the dispatch loop below),
# which under `set -euo pipefail` kills the entire script mid-loop --
# violating REQ-F-004 ("a single pair's non-zero exit never aborts the
# batch"). classify_pair now ALWAYS returns 0 and instead prints
# "classification_error" as the classification string for the caller to
# handle as a per-pair, summary-recorded outcome (never a script abort, and
# never a dispatch into the unreadable directory).
classify_pair() {
	local dir="$1"
	if [[ -f "$dir/record.jsonl" ]]; then
		echo "skipped_complete"
		return 0
	fi
	[[ -d "$dir" ]] || {
		echo "pending"
		return 0
	}
	local listing
	if ! listing="$(ls -A "$dir" 2>&1)"; then
		echo "run-batch: cannot list directory contents to classify $dir: $listing" >&2
		echo "classification_error"
		return 0
	fi
	if [[ -n "$listing" ]]; then
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
#
# R2-F-8 sweep (UAT uat-20260808-231500-E40-F03.md fix guidance #2): mkdir/mv
# failure here (e.g. --out has become unwritable) is surfaced via this
# function's own non-zero return; the caller below invokes it guarded by
# `|| quarantine_rc=$?` specifically so a failure is captured instead of
# tripping `set -e` on the plain statement.
#
# R2-F-8 sweep bug found (UAT uat-20260808-231500-E40-F03.md fix guidance
# #2): the mkdir -p and mv below used to run as un-return-checked plain
# statements followed by an unconditional trailing `echo`, whose OWN
# (always-successful) exit status became the function's return value --
# so a failing mkdir/mv was silently reported as success to the caller,
# regardless of `|| quarantine_rc=$?` at the call site. Both are now
# explicitly checked so a real failure propagates as this function's exit
# status, and the "quarantined" message only prints once the move actually
# succeeded.
quarantine_pair() {
	local dir="$1" item_id="$2" variant_id="$3" rep="$4"
	local incomplete_root="$out_root/.incomplete/$item_id/$variant_id"
	# R2-F-13 belt-and-braces: every path this function touches (the
	# quarantine root it mkdir's into, and the mv source/dest) is
	# containment-checked before the corresponding filesystem call.
	assert_within_out_root "$dir" || return 1
	assert_within_out_root "$incomplete_root" || return 1
	mkdir -p "$incomplete_root" || return 1
	local seq=1
	while [[ -e "$incomplete_root/rep-$rep-$seq" ]]; do
		seq=$((seq + 1))
	done
	local dest="$incomplete_root/rep-$rep-$seq"
	assert_within_out_root "$dest" || return 1
	mv "$dir" "$dest" || return 1
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

		# R2-F-13 belt-and-braces: item_id/variant_id are already
		# grammar-validated above, so this should never fire in practice
		# -- but the constructed path is still canonicalized and
		# containment-checked before classify_pair's own reads (ls/-f)
		# touch it, per the fix guidance's "before any mkdir/ls/mv/cp/
		# dispatch" requirement.
		if ! assert_within_out_root "$dir"; then
			echo "run-batch: $run_key skipped (path containment check failed, see above); batch proceeds" >&2
			append_summary "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
			log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
			overall_bad="true"
			continue
		fi

		classification="$(classify_pair "$dir")"

		case "$classification" in
		skipped_complete)
			append_summary "$item_id" "$variant_id" "$rep" "$run_key" "skipped_complete"
			;;
		classification_error)
			# R2-F-8 (UAT uat-20260808-231500-E40-F03.md): the directory
			# could not be listed (e.g. permission-denied). Recorded as a
			# per-pair error and the batch proceeds -- REQ-F-004 -- never
			# dispatched into a directory whose contents were never
			# confirmed safe to write into.
			echo "run-batch: $run_key skipped (classification error, see above); batch proceeds" >&2
			append_summary "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
			log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
			overall_bad="true"
			;;
		incomplete_prior_attempt)
			if [[ "$reclaim_incomplete" == "true" ]]; then
				quarantine_rc=0
				if [[ "$dry_run" != "true" ]]; then
					# R2-F-8 sweep: `|| quarantine_rc=$?` captures a
					# failing mkdir/mv without tripping `set -e` on this
					# plain statement (the same abort exposure
					# classify_pair's bare command substitution had).
					quarantine_pair "$dir" "$item_id" "$variant_id" "$rep" || quarantine_rc=$?
				fi
				if [[ "$quarantine_rc" -ne 0 ]]; then
					echo "run-batch: failed to quarantine $run_key (exit $quarantine_rc); skipping this pair, batch proceeds" >&2
					append_summary "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
					log_batch_event "$item_id" "$variant_id" "$rep" "$run_key" "classification_error"
					overall_bad="true"
				else
					dispatch_pair "$item_id" "$variant_id" "$rep" "$run_key" "quarantined_and_rerun"
				fi
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

# R2-F-5 fix (UAT uat-20260808-231500-E40-F03.md): requested/unresolved/
# selected item ids are carried into the summary object itself, not just
# stderr -- written one-per-line to temp files since bash arrays can't be
# passed as argv to a heredoc script directly.
REQUESTED_ITEMS_TMP="$(mktemp)"
UNRESOLVED_ITEMS_TMP="$(mktemp)"
SELECTED_ITEMS_TMP="$(mktemp)"
cleanup_items_tmp() { rm -f "$REQUESTED_ITEMS_TMP" "$UNRESOLVED_ITEMS_TMP" "$SELECTED_ITEMS_TMP"; }
trap 'cleanup; cleanup_items_tmp' EXIT
printf '%s\n' "${requested_ids[@]-}" | sed '/^$/d' >"$REQUESTED_ITEMS_TMP" || true
printf '%s\n' "${unresolved_ids[@]-}" | sed '/^$/d' >"$UNRESOLVED_ITEMS_TMP" || true
printf '%s\n' "${selected_item_ids[@]-}" | sed '/^$/d' >"$SELECTED_ITEMS_TMP" || true

python3 - "$SUMMARY_TMP" "$corpus_yaml" "$out_root" "$variant_id" "$reps" "$dry_run" "$reclaim_incomplete" \
	"$REQUESTED_ITEMS_TMP" "$UNRESOLVED_ITEMS_TMP" "$SELECTED_ITEMS_TMP" <<'PYEOF'
import json
import sys

(tmp_path, corpus, out_root, variant_id, reps, dry_run, reclaim,
 requested_tmp, unresolved_tmp, selected_tmp) = sys.argv[1:11]


def read_lines(path):
    with open(path) as f:
        return [line.rstrip("\n") for line in f if line.strip()]


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
    "requested_items": read_lines(requested_tmp),
    "selected_items": read_lines(selected_tmp),
    "unresolved_items": read_lines(unresolved_tmp),
    "variant_id": variant_id,
}
print(json.dumps(summary, indent=2, sort_keys=True))
PYEOF

if [[ "$overall_bad" == "true" ]]; then
	exit 1
fi
exit 0
