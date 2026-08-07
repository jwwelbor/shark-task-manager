#!/usr/bin/env bash
# TC-017 (test-plan.md AC test matrix; T-E40-F03-001 task spec Test Cases),
# sub-cases a-f.
#
# Exercises AC-01..AC-05 (REQ-F-001..006) and the RUN_ONE_BIN default-
# resolution binding decision (TD-077's defect class): the matrix driver
# (bench/scripts/run-batch.sh) enumerates every admitted corpus item x the
# default variant x N reps, classifies each target directory, invokes
# run-one.sh for pending pairs only, quarantines stale attempts under
# --reclaim-incomplete, and never aborts the batch on one pair's failure.
#
# Caller-Path Contract (test-plan.md TC-017): real subprocess invocation of
# `bench/scripts/run-batch.sh` against the real corpus
# (bench/corpus/corpus.yaml, except TC-017e/f which use small synthetic
# corpora of their own), with RUN_ONE_BIN pointed at
# bench/scripts/testdata/stubs/run-one for the duration of each test --
# run-batch.sh's own enumeration, classification, and quarantine logic runs
# unmodified; only the external run-one.sh invocation is replaced.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

RUN_BATCH="$SCRIPTS_DIR/run-batch.sh"
STUB_RUN_ONE="$SCRIPTS_DIR/testdata/stubs/run-one"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
GOLDEN_RECORD="$REPO_ROOT/tests/contracts/testdata/e40_i02_golden_record.jsonl"

fail() {
	echo "TC-017 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_BATCH" ]] || fail "run-batch.sh missing or not executable: $RUN_BATCH"
[[ -x "$STUB_RUN_ONE" ]] || fail "stub run-one missing or not executable: $STUB_RUN_ONE"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -f "$GOLDEN_RECORD" ]] || fail "I-02 golden record missing: $GOLDEN_RECORD"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# invocation_count <log-file> -- 0 if the log was never created (no calls).
invocation_count() {
	local log="$1"
	if [[ -f "$log" ]]; then
		wc -l <"$log" | tr -d ' '
	else
		echo 0
	fi
}

# ---------------------------------------------------------------------------
# TC-017a (AC-01, REQ-F-001/REQ-F-005): a batch invocation over the real
# 10-item corpus x 3 reps runs to completion unattended (</dev/null) and
# prints a summary naming all 30 pairs' classifications; none of the 3
# negative_items ids ever appear. Negative: a negative_items id passed via
# --items is rejected/ignored, never dispatched.
# ---------------------------------------------------------------------------
test_a() {
	local out_dir="$WORKDIR/a-out" log="$WORKDIR/a-run-one.log"
	local out="$WORKDIR/a.out" err="$WORKDIR/a.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" \
		STUB_RUN_ONE_LOG="$log" \
		STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 3 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "a: run-batch.sh exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert len(d["pairs"]) == 30, "summary names %d pairs, want 30" % len(d["pairs"])
assert d["counts"].get("pending_run") == 30, "pending_run count=%r, want 30" % d["counts"].get("pending_run")
for neg in ("cart-new-cart-zero-subtotal", "cart-add-item-rejects-negative-quantity", "validate-sku-uppercase-only"):
    assert not any(p["item_id"] == neg for p in d["pairs"]), "negative_items id %s appeared in the summary" % neg
' "$out" || fail "a: summary check failed: $(cat "$out")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "30" ]] || fail "a: stub invocation count=$calls, want 30"

	echo "TC-017a PASS"

	# --- Negative (REQ-F-005): a negative_items id passed via --items is
	# rejected/ignored, never dispatched. ---
	local neg_out_dir="$WORKDIR/a-neg-out" neg_log="$WORKDIR/a-neg-run-one.log"
	local neg_out="$WORKDIR/a-neg.out" neg_err="$WORKDIR/a-neg.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" \
		STUB_RUN_ONE_LOG="$neg_log" \
		STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$neg_out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "cart-new-cart-zero-subtotal,validate-sku-max-length" \
		</dev/null >"$neg_out" 2>"$neg_err"
	local neg_code=$?
	set -e
	[[ "$neg_code" -eq 0 ]] || fail "a(negative): run-batch.sh exited $neg_code, want 0: $(cat "$neg_err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert len(d["pairs"]) == 1, "summary names %d pairs, want exactly 1 (only the admitted id): %r" % (len(d["pairs"]), d["pairs"])
assert d["pairs"][0]["item_id"] == "validate-sku-max-length", "unexpected pair dispatched: %r" % d["pairs"][0]
' "$neg_out" || fail "a(negative): summary check failed: $(cat "$neg_out")"

	local neg_calls
	neg_calls="$(invocation_count "$neg_log")"
	[[ "$neg_calls" == "1" ]] || fail "a(negative): stub invocation count=$neg_calls, want exactly 1 (the negative_items id was never dispatched)"

	echo "TC-017a(negative) PASS"
}

# ---------------------------------------------------------------------------
# TC-017b (AC-02): re-invoking a fully-populated batch performs zero
# run-one.sh invocations (the stub's own call count) and exits zero.
# ---------------------------------------------------------------------------
test_b() {
	local out_dir="$WORKDIR/b-out"
	local log1="$WORKDIR/b-run-one-1.log" log2="$WORKDIR/b-run-one-2.log"
	local out1="$WORKDIR/b1.out" err1="$WORKDIR/b1.err"
	local out2="$WORKDIR/b2.out" err2="$WORKDIR/b2.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log1" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 3 \
		</dev/null >"$out1" 2>"$err1"
	local code1=$?
	set -e
	[[ "$code1" -eq 0 ]] || fail "b: first (populating) run-batch.sh invocation exited $code1, want 0: $(cat "$err1")"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log2" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 3 \
		</dev/null >"$out2" 2>"$err2"
	local code2=$?
	set -e
	[[ "$code2" -eq 0 ]] || fail "b: second (resume) run-batch.sh invocation exited $code2, want 0: $(cat "$err2")"

	local calls2
	calls2="$(invocation_count "$log2")"
	[[ "$calls2" == "0" ]] || fail "b: second invocation made $calls2 RUN_ONE_BIN calls (stub's own call count), want exactly 0"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert d["counts"].get("skipped_complete") == 30, "skipped_complete count=%r, want 30" % d["counts"].get("skipped_complete")
' "$out2" || fail "b: second invocation summary check failed: $(cat "$out2")"

	echo "TC-017b PASS"
}

# ---------------------------------------------------------------------------
# TC-017c (AC-03): a directory holding run/ and post/ but no record.jsonl
# is its own incomplete_prior_attempt state -- skipped by default, named in
# the summary, every other pending pair still runs, exit non-zero. Negative:
# the stale directory's contents are byte-unchanged (no silent cleanup).
# ---------------------------------------------------------------------------
test_c() {
	local out_dir="$WORKDIR/c-out" log="$WORKDIR/c-run-one.log"
	local out="$WORKDIR/c.out" err="$WORKDIR/c.err"
	local stale_item="validate-sku-max-length" stale_rep=2
	local stale_dir="$out_dir/$stale_item/default/rep-$stale_rep"

	mkdir -p "$stale_dir/run" "$stale_dir/post"
	echo "stdout-from-a-crashed-attempt" >"$stale_dir/run/stdout.json"
	echo "toolchain-guard-partial" >"$stale_dir/post/toolchain-guard.json"

	local snapshot="$WORKDIR/c-stale-snapshot"
	cp -r "$stale_dir" "$snapshot"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 3 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "c: run-batch.sh exited 0, want non-zero (an incomplete_prior_attempt was skipped): $(cat "$out")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "29" ]] || fail "c: stub invocation count=$calls, want 29 (30 pairs minus the incomplete one): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item, rep = sys.argv[2], int(sys.argv[3])
matches = [p for p in d["pairs"] if p["item_id"] == item and p["rep"] == rep]
assert len(matches) == 1, "expected exactly one summary entry for %s rep %d, found %d" % (item, rep, len(matches))
assert matches[0]["classification"] == "incomplete_prior_attempt", "classification=%r, want incomplete_prior_attempt" % matches[0]["classification"]
assert d["counts"].get("pending_run") == 29, "pending_run count=%r, want 29" % d["counts"].get("pending_run")
' "$out" "$stale_item" "$stale_rep" || fail "c: summary classification check failed: $(cat "$out")"

	# Negative: the stale directory's contents are byte-unchanged (no
	# silent cleanup without --reclaim-incomplete).
	diff -r "$snapshot" "$stale_dir" >/dev/null || fail "c: stale directory contents changed without --reclaim-incomplete"
	[[ ! -f "$stale_dir/record.jsonl" ]] || fail "c: record.jsonl was written into the stale directory without --reclaim-incomplete"

	echo "TC-017c PASS"
}

# ---------------------------------------------------------------------------
# TC-017d (AC-04): same fixture as TC-017c with --reclaim-incomplete. The
# stale directory is MOVED (not copied, not deleted) to
# .incomplete/<item>/<variant>/rep-<n>-<seq>/, byte-identical there; the
# original target path is recreated fresh (no leftover run/post content)
# and dispatched exactly once.
# ---------------------------------------------------------------------------
test_d() {
	local out_dir="$WORKDIR/d-out" log="$WORKDIR/d-run-one.log"
	local out="$WORKDIR/d.out" err="$WORKDIR/d.err"
	local stale_item="validate-sku-max-length" stale_rep=2
	local stale_dir="$out_dir/$stale_item/default/rep-$stale_rep"

	mkdir -p "$stale_dir/run" "$stale_dir/post"
	echo "stdout-from-a-crashed-attempt" >"$stale_dir/run/stdout.json"
	echo "toolchain-guard-partial" >"$stale_dir/post/toolchain-guard.json"

	local snapshot="$WORKDIR/d-stale-snapshot"
	cp -r "$stale_dir" "$snapshot"
	# Captured BEFORE the batch runs, from the original file -- a `mv`
	# (rename, same filesystem) preserves the inode; a copy+delete would
	# not. This is what actually distinguishes "moved" from "copied then
	# deleted" (REQ-F-006) -- byte-identical contents alone (checked below
	# via diff) cannot.
	local orig_inode
	orig_inode="$(stat -c %i "$stale_dir/run/stdout.json")"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 3 --reclaim-incomplete \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "d: run-batch.sh exited $code, want 0 (--reclaim-incomplete quarantines and reruns): $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "30" ]] || fail "d: stub invocation count=$calls, want 30 (every pair, including the reclaimed one): $(cat "$err")"

	local quarantine_dir="$out_dir/.incomplete/$stale_item/default/rep-$stale_rep-1"
	[[ -d "$quarantine_dir" ]] || fail "d: quarantine directory not found at $quarantine_dir"
	diff -r "$snapshot" "$quarantine_dir" >/dev/null || fail "d: quarantined directory's contents are not byte-identical to the original stale directory"

	local quarantine_inode
	quarantine_inode="$(stat -c %i "$quarantine_dir/run/stdout.json")"
	[[ "$orig_inode" == "$quarantine_inode" ]] || fail "d: quarantine did not preserve the file's inode (orig=$orig_inode quarantine=$quarantine_inode) -- looks like a copy+delete rather than a move (REQ-F-006)"

	[[ -f "$stale_dir/record.jsonl" ]] || fail "d: fresh target path has no record.jsonl after quarantine+rerun"
	[[ ! -e "$stale_dir/run/stdout.json" ]] || fail "d: fresh target path still carries the old (quarantined) run/stdout.json -- the driver was invoked against a non-fresh path"
	[[ ! -e "$stale_dir/post/toolchain-guard.json" ]] || fail "d: fresh target path still carries the old (quarantined) post/toolchain-guard.json -- the driver was invoked against a non-fresh path"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item, rep = sys.argv[2], int(sys.argv[3])
matches = [p for p in d["pairs"] if p["item_id"] == item and p["rep"] == rep]
assert len(matches) == 1
assert matches[0]["classification"] == "quarantined_and_rerun", "classification=%r, want quarantined_and_rerun" % matches[0]["classification"]
' "$out" "$stale_item" "$stale_rep" || fail "d: summary classification check failed: $(cat "$out")"

	# Negative: exactly one invocation was logged for that pair's run_key --
	# the source .incomplete/ entry is never itself a target run-one.sh
	# writes back into.
	local match_count
	match_count="$(python3 -c '
import json
import sys

log_path, item, rep = sys.argv[1], sys.argv[2], sys.argv[3]
count = 0
with open(log_path) as f:
    for line in f:
        argv = json.loads(line)["argv"]
        if "--item" in argv and "--rep" in argv:
            if argv[argv.index("--item") + 1] == item and argv[argv.index("--rep") + 1] == rep:
                count += 1
print(count)
' "$log" "$stale_item" "$stale_rep")"
	[[ "$match_count" == "1" ]] || fail "d: RUN_ONE_BIN was invoked $match_count times for $stale_item rep $stale_rep, want exactly 1"

	echo "TC-017d PASS"
}

# ---------------------------------------------------------------------------
# TC-017e (AC-05): a 3-item synthetic corpus with one pair configured
# (env-var-selected) to fail -- every pending pair is still attempted, the
# failing pair is named with a failure classification, exit non-zero.
# Negative: the failure sits at the LAST enumerated pair, yet every prior
# pair still shows as completed -- the loop does not short-circuit.
# ---------------------------------------------------------------------------
test_e() {
	local out_dir="$WORKDIR/e-out" log="$WORKDIR/e-run-one.log"
	local out="$WORKDIR/e.out" err="$WORKDIR/e.err"
	local corpus="$WORKDIR/e-corpus.yaml"

	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
  - id: beta
  - id: gamma
negative_items: []
YAML

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		STUB_RUN_ONE_FAIL_RUN_KEY="gamma::default::rep1" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "e: run-batch.sh exited 0, want non-zero (one pair failed): $(cat "$out")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "3" ]] || fail "e: stub invocation count=$calls, want 3 (every pending pair attempted despite the failure): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
by_item = {p["item_id"]: p["classification"] for p in d["pairs"]}
assert by_item.get("gamma") == "failed", "gamma classification=%r, want failed" % by_item.get("gamma")
assert by_item.get("alpha") == "pending_run", "alpha classification=%r, want pending_run" % by_item.get("alpha")
assert by_item.get("beta") == "pending_run", "beta classification=%r, want pending_run" % by_item.get("beta")
' "$out" || fail "e: summary classification check failed: $(cat "$out")"

	echo "TC-017e PASS"

	# --- Negative: gamma must be the LAST-LOGGED invocation (proves the
	# failure genuinely sat at the last enumerated position and the loop
	# still reached it -- alpha/beta both succeeding is not, by itself,
	# proof of enumeration order, since it would hold under any order that
	# happened to run gamma last by chance). Checked against the stub's own
	# invocation log, not the summary (whose pairs[] order need not match
	# dispatch order). ---
	local last_item
	last_item="$(python3 -c '
import json
import sys

with open(sys.argv[1]) as f:
    lines = [line for line in f if line.strip()]
argv = json.loads(lines[-1])["argv"]
print(argv[argv.index("--item") + 1])
' "$log")"
	[[ "$last_item" == "gamma" ]] || fail "e: last logged invocation was for item '$last_item', want gamma -- the failing pair must be the last enumerated one to prove the loop did not short-circuit before reaching it"

	echo "TC-017e(negative, failure confirmed at the last-enumerated/last-logged position) PASS"
}

# ---------------------------------------------------------------------------
# TC-017f (RUN_ONE_BIN default resolution; TD-077's defect class): with
# RUN_ONE_BIN unset, run-batch.sh resolves and invokes the SIBLING path
# ($SCRIPT_DIR/run-one.sh), never a bare-PATH-resolved name. Per the
# Caller-Path Contract, the sibling directory is deliberately kept OFF
# PATH -- that would test PATH-resolution instead of sibling-path-default
# resolution, a different failure mode than TD-077's.
# ---------------------------------------------------------------------------
test_f() {
	local sibling_dir="$WORKDIR/f-sibling"
	mkdir -p "$sibling_dir"
	cp "$RUN_BATCH" "$sibling_dir/run-batch.sh"
	chmod +x "$sibling_dir/run-batch.sh"
	cp "$STUB_RUN_ONE" "$sibling_dir/run-one.sh"
	chmod +x "$sibling_dir/run-one.sh"

	# PATH decoy: a DIFFERENT directory (never the sibling dir, per the
	# Caller-Path Contract's own forbidden-mock rule) prepended to PATH,
	# holding its own script literally named run-one.sh that must NEVER be
	# invoked. A bare-name RUN_ONE_BIN default (TD-077's defect class)
	# would resolve to this decoy via PATH; a correct
	# ${RUN_ONE_BIN:-$SCRIPT_DIR/run-one.sh} default ignores PATH entirely
	# and reaches the sibling stand-in instead. This also removes the
	# environment-dependence a plain "unset RUN_ONE_BIN" run would have
	# (passing today only because no run-one.sh happens to sit on PATH).
	local decoy_dir="$WORKDIR/f-path-decoy" decoy_marker="$WORKDIR/f-decoy-invoked"
	mkdir -p "$decoy_dir"
	cat >"$decoy_dir/run-one.sh" <<DECOY
#!/usr/bin/env bash
touch "$decoy_marker"
exit 0
DECOY
	chmod +x "$decoy_dir/run-one.sh"

	local out_dir="$WORKDIR/f-out" log="$WORKDIR/f-run-one.log"
	local out="$WORKDIR/f.out" err="$WORKDIR/f.err"
	local corpus="$WORKDIR/f-corpus.yaml"
	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
negative_items: []
YAML

	case ":$PATH:" in
	*":$sibling_dir:"*)
		fail "f: test setup accidentally put $sibling_dir on PATH -- this would test PATH-resolution, not sibling-path-default resolution"
		;;
	esac

	set +e
	env -u RUN_ONE_BIN \
		PATH="$decoy_dir:$PATH" \
		STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$sibling_dir/run-batch.sh" --out "$out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "f: run-batch.sh (RUN_ONE_BIN unset) exited $code, want 0: $(cat "$err")"

	[[ ! -f "$decoy_marker" ]] || fail "f: the PATH-resolved decoy run-one.sh was invoked -- RUN_ONE_BIN defaulted to a bare, PATH-resolved name instead of the sibling path"

	[[ -f "$log" ]] || fail "f: sibling-path run-one.sh stand-in was never invoked -- RUN_ONE_BIN did not default to \$SCRIPT_DIR/run-one.sh: $(cat "$err")"
	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "f: sibling-path run-one.sh stand-in invocation count=$calls, want exactly 1"

	echo "TC-017f PASS"
}

test_a
test_b
test_c
test_d
test_e
test_f

echo "TC-017 PASS"
