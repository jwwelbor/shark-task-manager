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
	# run-batch.sh sources lib/path-safety.sh relative to its own
	# $SCRIPT_DIR (NEW-1's shared-lib fix) -- the copied sibling deployment
	# needs that lib/ directory alongside it too.
	mkdir -p "$sibling_dir/lib"
	cp "$SCRIPTS_DIR/lib/path-safety.sh" "$sibling_dir/lib/path-safety.sh"

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

# ---------------------------------------------------------------------------
# TC-017g (F-2, UAT uat-20260808-E40-F03.md): a corpus.yaml whose second
# items: entry lacks an id key makes the embedded enumeration Python raise
# mid-stream. The batch must fail loud and name the enumeration error, never
# silently truncate the matrix to whatever partial output was captured
# before the raise (the pre-fix defect: 1-of-2 items enumerated, exit 0).
# ---------------------------------------------------------------------------
test_g() {
	local out_dir="$WORKDIR/g-out" log="$WORKDIR/g-run-one.log"
	local out="$WORKDIR/g.out" err="$WORKDIR/g.err"
	local corpus="$WORKDIR/g-corpus.yaml"

	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
  - not_an_id: beta
negative_items: []
YAML

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "g: run-batch.sh exited 0 over a malformed corpus.yaml (missing id key), want non-zero: $(cat "$out")"

	grep -qi "enumerat" "$err" || fail "g: stderr does not name the enumeration failure: $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "0" ]] || fail "g: run-one.sh was invoked $calls time(s) over a corpus that failed to enumerate, want exactly 0 (no partial dispatch over a truncated matrix)"

	[[ ! -s "$out" ]] || fail "g: a JSON summary was printed despite the enumeration failure, want none: $(cat "$out")"

	echo "TC-017g PASS"
}

# ---------------------------------------------------------------------------
# TC-017h (F-5, UAT uat-20260808-E40-F03.md): --items resolving to zero
# admitted ids (a typo, or a negative_items id) must fail loud and name the
# unresolved id(s) on stderr, not report a clean success over an empty
# batch. Negative: a --items list mixing one valid and one invalid id still
# runs the valid one AND warns about the invalid one on stderr.
# ---------------------------------------------------------------------------
test_h() {
	local out_dir="$WORKDIR/h-out" log="$WORKDIR/h-run-one.log"
	local out="$WORKDIR/h.out" err="$WORKDIR/h.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "does-not-exist" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "h: run-batch.sh exited 0 with --items resolving to zero admitted items, want non-zero: $(cat "$out")"

	grep -q "does-not-exist" "$err" || fail "h: stderr does not name the unresolved id 'does-not-exist': $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "0" ]] || fail "h: run-one.sh was invoked $calls time(s) over an empty --items match, want exactly 0"

	echo "TC-017h PASS"

	# --- Negative: a mix of one valid and one invalid id still runs the
	# valid one and warns per-id about the invalid one. ---
	local mix_out_dir="$WORKDIR/h-mix-out" mix_log="$WORKDIR/h-mix-run-one.log"
	local mix_out="$WORKDIR/h-mix.out" mix_err="$WORKDIR/h-mix.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$mix_log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$mix_out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length,also-does-not-exist" \
		</dev/null >"$mix_out" 2>"$mix_err"
	local mix_code=$?
	set -e
	[[ "$mix_code" -eq 0 ]] || fail "h(mix): run-batch.sh exited $mix_code, want 0 (one id resolved and its pair succeeded): $(cat "$mix_err")"

	grep -q "also-does-not-exist" "$mix_err" || fail "h(mix): stderr does not name the unresolved id 'also-does-not-exist' even though another id resolved: $(cat "$mix_err")"

	local mix_calls
	mix_calls="$(invocation_count "$mix_log")"
	[[ "$mix_calls" == "1" ]] || fail "h(mix): stub invocation count=$mix_calls, want exactly 1 (only the resolved id)"

	echo "TC-017h(mix) PASS"
}

# ---------------------------------------------------------------------------
# TC-017i (F-2 fix-guidance's independent count guard, UAT
# uat-20260808-E40-F03.md): the enumeration Python can exit 0 while still
# printing a line count that does not match corpus.yaml's declared items:
# length -- an id containing an embedded newline prints as two lines for one
# declared item. The independent "enumerated count == declared count" guard
# (not just the python3-exit-status guard TC-017g exercises) must catch this
# and fail loud rather than silently readarray-ing the extra line as if it
# were a second item id.
# ---------------------------------------------------------------------------
test_i() {
	local out_dir="$WORKDIR/i-out" log="$WORKDIR/i-run-one.log"
	local out="$WORKDIR/i.out" err="$WORKDIR/i.err"
	local corpus="$WORKDIR/i-corpus.yaml"

	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: "a\nb"
negative_items: []
YAML

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "i: run-batch.sh exited 0 over a corpus.yaml whose enumerated line count (2) does not match its declared items: length (1), want non-zero: $(cat "$out")"

	grep -qi "declares" "$err" || fail "i: stderr does not name the enumerated-vs-declared count mismatch: $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "0" ]] || fail "i: run-one.sh was invoked $calls time(s) over a count-mismatched enumeration, want exactly 0"

	echo "TC-017i PASS"
}

# ---------------------------------------------------------------------------
# TC-017j (R2-F-8, UAT uat-20260808-231500-E40-F03.md): classify_pair's
# unreadable-directory guard must be a per-pair classification_error, never
# a script-fatal abort. One unreadable rep dir among several dispatchable
# pairs must not stop the batch: every other pair is still attempted and a
# parseable JSON summary is still printed, naming the unreadable pair as
# classification_error, exit non-zero.
# ---------------------------------------------------------------------------
test_j() {
	local out_dir="$WORKDIR/j-out" log="$WORKDIR/j-run-one.log"
	local out="$WORKDIR/j.out" err="$WORKDIR/j.err"
	local corpus="$WORKDIR/j-corpus.yaml"
	local unreadable_item="alpha" unreadable_rep=1
	local unreadable_dir="$out_dir/$unreadable_item/default/rep-$unreadable_rep"

	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
  - id: beta
  - id: gamma
negative_items: []
YAML

	mkdir -p "$unreadable_dir"
	chmod 000 "$unreadable_dir"
	restore_perms() { chmod 755 "$unreadable_dir" 2>/dev/null || true; }
	trap 'restore_perms; cleanup' EXIT

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	restore_perms
	trap cleanup EXIT

	[[ "$code" -ne 0 ]] || fail "j: run-batch.sh exited 0 with an unreadable pair directory, want non-zero: $(cat "$out")"

	[[ -s "$out" ]] || fail "j: no JSON summary was printed despite an unreadable pair directory (script aborted mid-loop instead of recording a per-pair error): $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "2" ]] || fail "j: stub invocation count=$calls, want 2 (beta and gamma both still attempted despite alpha's unreadable directory): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
by_item = {p["item_id"]: p["classification"] for p in d["pairs"]}
assert by_item.get("alpha") == "classification_error", "alpha classification=%r, want classification_error" % by_item.get("alpha")
assert by_item.get("beta") == "pending_run", "beta classification=%r, want pending_run" % by_item.get("beta")
assert by_item.get("gamma") == "pending_run", "gamma classification=%r, want pending_run" % by_item.get("gamma")
' "$out" || fail "j: summary classification check failed: $(cat "$out")"

	echo "TC-017j PASS"
}

# ---------------------------------------------------------------------------
# TC-017k (R2-F-5, UAT uat-20260808-231500-E40-F03.md): the machine-readable
# summary object itself (not just stderr) must record requested_items,
# unresolved_items, and selected_items when --items names an id that does
# not resolve -- including under --dry-run, which never touches stderr's
# audience differently but must still populate the same fields.
# ---------------------------------------------------------------------------
test_k() {
	local out_dir="$WORKDIR/k-out" log="$WORKDIR/k-run-one.log"
	local out="$WORKDIR/k.out" err="$WORKDIR/k.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length,also-does-not-exist" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "k: run-batch.sh exited $code, want 0 (one id resolved): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert sorted(d.get("requested_items", [])) == sorted(["validate-sku-max-length", "also-does-not-exist"]), "requested_items=%r" % d.get("requested_items")
assert d.get("unresolved_items") == ["also-does-not-exist"], "unresolved_items=%r, want [\"also-does-not-exist\"]" % d.get("unresolved_items")
assert d.get("selected_items") == ["validate-sku-max-length"], "selected_items=%r, want [\"validate-sku-max-length\"]" % d.get("selected_items")
' "$out" || fail "k: summary did not record requested/unresolved/selected items: $(cat "$out")"

	echo "TC-017k PASS"

	# --- Same check under --dry-run, which invokes nothing but must still
	# populate the same fields. ---
	local dry_out_dir="$WORKDIR/k-dry-out"
	local dry_out="$WORKDIR/k-dry.out" dry_err="$WORKDIR/k-dry.err"

	set +e
	"$RUN_BATCH" --out "$dry_out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length,also-does-not-exist" --dry-run \
		</dev/null >"$dry_out" 2>"$dry_err"
	local dry_code=$?
	set -e
	[[ "$dry_code" -eq 0 ]] || fail "k(dry-run): run-batch.sh exited $dry_code, want 0: $(cat "$dry_err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert d.get("unresolved_items") == ["also-does-not-exist"], "unresolved_items=%r under --dry-run" % d.get("unresolved_items")
assert d.get("selected_items") == ["validate-sku-max-length"], "selected_items=%r under --dry-run" % d.get("selected_items")
' "$dry_out" || fail "k(dry-run): summary did not record unresolved/selected items under --dry-run: $(cat "$dry_out")"

	echo "TC-017k(dry-run) PASS"
}

# ---------------------------------------------------------------------------
# TC-017l (R2-F-8 sweep, UAT uat-20260808-231500-E40-F03.md fix guidance #2):
# quarantine_pair's mv is called as a plain statement inside the dispatch
# loop, not inside a command substitution -- but a failing mv (e.g. the
# quarantine root cannot be created because --out is unwritable) has the
# identical set -e abort exposure as classify_pair's bare command
# substitution. It must not abort the batch either: the failure is recorded
# as a classification_error and every other pending pair still runs.
# ---------------------------------------------------------------------------
test_l() {
	local out_dir="$WORKDIR/l-out" log="$WORKDIR/l-run-one.log"
	local out="$WORKDIR/l.out" err="$WORKDIR/l.err"
	local corpus="$WORKDIR/l-corpus.yaml"
	local stale_item="alpha" stale_rep=1
	local stale_dir="$out_dir/$stale_item/default/rep-$stale_rep"

	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
  - id: beta
negative_items: []
YAML

	mkdir -p "$stale_dir/run" "$out_dir/.incomplete"
	echo "stale" >"$stale_dir/run/stdout.json"
	# Block quarantine_pair's `mkdir -p "$out_root/.incomplete/$item_id/..."`
	# (and therefore its subsequent mv) by making the pre-existing
	# .incomplete/ root unwritable -- $out_root itself stays writable so
	# beta's own dispatch is unaffected, isolating the quarantine failure
	# from a broader out_root-unwritable scenario.
	chmod 000 "$out_dir/.incomplete"
	restore_perms() { chmod 755 "$out_dir/.incomplete" 2>/dev/null || true; }
	trap 'restore_perms; cleanup' EXIT

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$corpus" --reps 1 --reclaim-incomplete \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	restore_perms
	trap cleanup EXIT

	[[ "$code" -ne 0 ]] || fail "l: run-batch.sh exited 0 despite a failed quarantine mv, want non-zero: $(cat "$out")"

	[[ -s "$out" ]] || fail "l: no JSON summary was printed despite a failed quarantine mv (script aborted mid-loop instead of recording a per-pair error): $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "l: stub invocation count=$calls, want 1 (beta still attempted despite alpha's failed quarantine): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
by_item = {p["item_id"]: p["classification"] for p in d["pairs"]}
assert by_item.get("alpha") == "classification_error", "alpha classification=%r, want classification_error" % by_item.get("alpha")
assert by_item.get("beta") == "pending_run", "beta classification=%r, want pending_run" % by_item.get("beta")
' "$out" || fail "l: summary classification check failed: $(cat "$out")"

	echo "TC-017l PASS"
}

# ---------------------------------------------------------------------------
# TC-017m (R2-F-13, code-review-20260808-235300-E40-F03.md /
# uat-20260808-231500-E40-F03.md): item_id/variant_id must be validated
# against a safe-slug grammar before either reaches path construction --
# neither an operator-supplied traversing --variant nor a traversing
# corpus.yaml item id may move or write anything outside --out.
# ---------------------------------------------------------------------------
test_m() {
	# --- (a) traversing --variant: must be rejected before any dispatch,
	# and the directory it would have escaped to (--out's grandparent)
	# must never come into existence. ---
	local out_dir="$WORKDIR/m-variant-out" log="$WORKDIR/m-variant-run-one.log"
	local out="$WORKDIR/m-variant.out" err="$WORKDIR/m-variant.err"
	local victim_marker="$WORKDIR/m-variant-victim-marker"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length" --variant "../../m-variant-victim-marker" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "m(variant): run-batch.sh exited 0 with a traversing --variant, want non-zero: $(cat "$out")"

	grep -qi "safe path identifier" "$err" || fail "m(variant): stderr does not name the unsafe --variant identifier: $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "0" ]] || fail "m(variant): run-one.sh was invoked $calls time(s) with a traversing --variant, want exactly 0"

	[[ ! -e "$victim_marker" ]] || fail "m(variant): a path escaped --out and touched $victim_marker"
	[[ ! -d "$out_dir" ]] || fail "m(variant): --out was created despite the traversing --variant being rejected before any filesystem operation"

	echo "TC-017m(variant) PASS"

	# --- (b) traversing corpus.yaml item id: must be rejected at
	# enumeration time, before any --items intersection or dispatch. ---
	local corpus="$WORKDIR/m-item-corpus.yaml"
	cat >"$corpus" <<'YAML'
schema_version: "1.0"
items:
  - id: alpha
  - id: "../../m-item-victim"
negative_items: []
YAML

	local item_out_dir="$WORKDIR/m-item-out" item_log="$WORKDIR/m-item-run-one.log"
	local item_out="$WORKDIR/m-item.out" item_err="$WORKDIR/m-item.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$item_log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$item_out_dir" --corpus "$corpus" --reps 1 \
		</dev/null >"$item_out" 2>"$item_err"
	local item_code=$?
	set -e
	[[ "$item_code" -ne 0 ]] || fail "m(item): run-batch.sh exited 0 over a corpus.yaml with a traversing item id, want non-zero: $(cat "$item_out")"

	grep -qi "safe path identifier" "$item_err" || fail "m(item): stderr does not name the unsafe corpus item id: $(cat "$item_err")"

	local item_calls
	item_calls="$(invocation_count "$item_log")"
	[[ "$item_calls" == "0" ]] || fail "m(item): run-one.sh was invoked $item_calls time(s) over a corpus with a traversing item id, want exactly 0 (rejected before any --items intersection or dispatch)"

	[[ ! -d "$item_out_dir" ]] || fail "m(item): --out was created despite the traversing item id being rejected before any filesystem operation"

	echo "TC-017m(item) PASS"
}

# ---------------------------------------------------------------------------
# TC-017n (R2-F-15, code-review-20260808-235300-E40-F03.md /
# uat-20260808-231500-E40-F03.md): a duplicate id in --items must resolve
# to exactly ONE selected pair, never two sharing one run_key -- checked
# under both --dry-run and real dispatch (where a double-dispatch would
# mean two API-spending invocations for one declared pair).
# ---------------------------------------------------------------------------
test_n() {
	local dry_out_dir="$WORKDIR/n-dry-out"
	local dry_out="$WORKDIR/n-dry.out" dry_err="$WORKDIR/n-dry.err"

	set +e
	"$RUN_BATCH" --out "$dry_out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length,validate-sku-max-length" --dry-run \
		</dev/null >"$dry_out" 2>"$dry_err"
	local dry_code=$?
	set -e
	[[ "$dry_code" -eq 0 ]] || fail "n(dry-run): run-batch.sh exited $dry_code, want 0: $(cat "$dry_err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert d["counts"].get("pending_run") == 1, "counts.pending_run=%r, want 1 (duplicate --items id must not double the matrix)" % d["counts"].get("pending_run")
assert d["selected_items"] == ["validate-sku-max-length"], "selected_items=%r, want a single entry (deduplicated)" % d["selected_items"]
assert len(d["pairs"]) == 1, "pairs has %d entries, want exactly 1" % len(d["pairs"])
run_keys = [p["run_key"] for p in d["pairs"]]
assert len(run_keys) == len(set(run_keys)), "duplicate run_key(s) in pairs: %r" % run_keys
' "$dry_out" || fail "n(dry-run): summary check failed: $(cat "$dry_out")"

	echo "TC-017n(dry-run) PASS"

	# --- Real dispatch: a duplicate --items id must cost exactly ONE
	# RUN_ONE_BIN invocation, never two. ---
	local out_dir="$WORKDIR/n-out" log="$WORKDIR/n-run-one.log"
	local out="$WORKDIR/n.out" err="$WORKDIR/n.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length,validate-sku-max-length" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "n: run-batch.sh exited $code, want 0: $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "n: stub invocation count=$calls, want exactly 1 (a duplicate --items id must cost at most one dispatch, not two)"

	echo "TC-017n PASS"
}

# ---------------------------------------------------------------------------
# TC-017o (NEW-1, HIGH, uat-20260809-013000-E40-F03.md; mirrors TC-019o):
# a SYMLINKED --out root must dispatch normally, never be rejected as
# "outside --out". run-batch.sh's own assert_within_out_root() already
# canonicalizes both sides of its containment comparison (R2-F-13), so
# this pins that correct behavior in place -- the sibling regression this
# UAT finding actually hit was in replay-manifest.sh's independent
# reimplementation (TC-019o), not here.
# ---------------------------------------------------------------------------
test_o() {
	local real_dir="$WORKDIR/o-real-out"
	local out_dir="$WORKDIR/o-symlink-out"
	mkdir -p "$real_dir"
	ln -s "$real_dir" "$out_dir"

	local log="$WORKDIR/o-run-one.log"
	local out="$WORKDIR/o.out" err="$WORKDIR/o.err"

	set +e
	RUN_ONE_BIN="$STUB_RUN_ONE" \
		STUB_RUN_ONE_LOG="$log" \
		STUB_RUN_ONE_RECORD_FILE="$GOLDEN_RECORD" \
		"$RUN_BATCH" --out "$out_dir" --corpus "$CORPUS_YAML" --reps 1 \
		--items "validate-sku-max-length" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "o: run-batch.sh exited $code with a symlinked --out root, want 0 (legitimate, fully-inside path): $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "o: stub invocation count=$calls, want exactly 1"

	echo "TC-017o PASS"
}

test_a
test_b
test_c
test_d
test_e
test_f
test_g
test_h
test_i
test_j
test_k
test_l
test_m
test_n
test_o

echo "TC-017 PASS"
