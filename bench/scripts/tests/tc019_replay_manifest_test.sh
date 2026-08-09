#!/usr/bin/env bash
# TC-019 (test-plan.md AC test matrix; T-E40-F03-006 task spec Test Cases),
# sub-cases a, b, c, h.
#
# T-E40-F03-006's slice: `replay-manifest.sh`'s DISPATCH half only --
# REQ-F-027's three preconditions (checked before any API spend, every
# failing check reported, not just the first), the ADR-F03-02 synthetic
# single-item corpus construction, REQ-F-028's corpus_drift detection, and
# the RUN_ONE_BIN default-resolution binding decision (TD-077's defect
# class, same as TC-017f). T-E40-F03-007 extends this file with sub-cases
# d/e/f/g (variant_bundle_drift, model_version_drift, the three-valued
# verdict).
#
# Caller-Path Contract (test-plan.md TC-019): real subprocess invocation of
# `bench/scripts/replay-manifest.sh` against the real corpus
# (bench/corpus/corpus.yaml, except TC-019a(ii)/(multi) which pass a
# stripped temp copy, and TC-019c which passes a temp copy with one item's
# fixture_base_sha edited -- bench/corpus/corpus.yaml itself is NEVER
# edited), with RUN_ONE_BIN pointed at
# bench/scripts/testdata/stubs/run-one for the duration of each test --
# replay-manifest.sh's own precondition checks, synthetic-corpus
# construction, and corpus_drift detection all run unmodified; only the
# external run-one.sh invocation is replaced. Stored-record fixtures reuse
# a real corpus item id (`cart-remove-item-last-match`, already used by
# F02's own TC-014a) per test-plan.md's "Fixture derivation" #5 -- REQ-F-
# 027's item-resolution precondition needs a real id.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

REPLAY="$SCRIPTS_DIR/replay-manifest.sh"
AGGREGATE="$SCRIPTS_DIR/aggregate-runs.sh"
STUB_RUN_ONE="$SCRIPTS_DIR/testdata/stubs/run-one"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
GEN_FIXTURES="$SCRIPTS_DIR/testdata/aggregate/gen_fixtures.py"

REAL_ITEM_ID="cart-remove-item-last-match"

fail() {
	echo "TC-019 FAIL: $1" >&2
	exit 1
}

[[ -x "$REPLAY" ]] || fail "replay-manifest.sh missing or not executable: $REPLAY"
[[ -x "$AGGREGATE" ]] || fail "aggregate-runs.sh missing or not executable: $AGGREGATE"
[[ -x "$STUB_RUN_ONE" ]] || fail "stub run-one missing or not executable: $STUB_RUN_ONE"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -f "$GEN_FIXTURES" ]] || fail "gen_fixtures.py missing: $GEN_FIXTURES"
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

# place_record <root> <item_id> <rep> [gen_fixtures.py args...]
# Same convention as tc018's own place_record: one golden-derived fixture
# at <root>/<item_id>/default/rep-<rep>/record.jsonl.
place_record() {
	local root="$1" item_id="$2" rep="$3"
	shift 3
	local dir="$root/$item_id/default/rep-$rep"
	mkdir -p "$dir"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$item_id" --rep "$rep" "$@" >"$dir/record.jsonl"
}

# build_band <band_out_path> <item_id> [reps]
# Builds a REAL aggregate.json (never hand-authored -- ADR-F03-08) for
# <item_id> over [reps] (default 3) golden-derived records, by invoking
# the real aggregate-runs.sh. This is precondition (iii)'s only source: a
# hand-written aggregate would omit baseline_id, and REQ-F-027 (iii)'s
# check is keyed off it.
build_band() {
	local band_out="$1" item_id="$2" reps="${3:-3}"
	local root="$WORKDIR/band-root-$item_id-$RANDOM"
	local rep
	for ((rep = 1; rep <= reps; rep++)); do
		place_record "$root" "$item_id" "$rep"
	done
	local out err code
	out="$(mktemp -p "$WORKDIR")"
	err="$(mktemp -p "$WORKDIR")"
	set +e
	"$AGGREGATE" --root "$root" >"$out" 2>"$err"
	code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "build_band: aggregate-runs.sh exited $code for $item_id: $(cat "$err")"
	cp "$out" "$band_out"
}

# run_replay <record> <band> <out_root> <corpus> [replay-manifest.sh args...]
# Invokes replay-manifest.sh, capturing stdout/stderr/exit code into
# WORKDIR-scoped files whose paths are printed on the last three lines
# (mirrors tc017's/tc018's own run_* helpers).
run_replay() {
	local record="$1" band="$2" out_root="$3" corpus="$4"
	shift 4
	local out err code
	out="$(mktemp -p "$WORKDIR")"
	err="$(mktemp -p "$WORKDIR")"
	set +e
	"$REPLAY" --record "$record" --band "$band" --out "$out_root" --corpus "$corpus" "$@" >"$out" 2>"$err"
	code=$?
	set -e
	echo "$out"
	echo "$err"
	echo "$code"
}

# strip_item_from_corpus <dest_path> <item_id_to_remove>
# A temp copy of corpus.yaml with one item's entry removed entirely --
# never the tracked file (REQ-F-027 (ii)'s "does not resolve" case).
strip_item_from_corpus() {
	local dest="$1" remove_id="$2"
	python3 -c "
import sys
import yaml

with open('$CORPUS_YAML') as f:
    d = yaml.safe_load(f)
d['items'] = [it for it in d.get('items') or [] if it['id'] != '$remove_id']
with open('$dest', 'w') as f:
    yaml.safe_dump(d, f, sort_keys=False)
"
}

# edit_item_fixture_sha <dest_path> <item_id> <new_sha>
# A temp copy of corpus.yaml with one item's fixture_base_sha edited --
# the ordinary curator-edit REQ-F-028's corpus_drift exists to catch
# (never the tracked file).
edit_item_fixture_sha() {
	local dest="$1" item_id="$2" new_sha="$3"
	python3 -c "
import sys
import yaml

with open('$CORPUS_YAML') as f:
    d = yaml.safe_load(f)
for it in d.get('items') or []:
    if it['id'] == '$item_id':
        it['fixture_base_sha'] = '$new_sha'
with open('$dest', 'w') as f:
    yaml.safe_dump(d, f, sort_keys=False)
"
}

# real_fixture_base_sha <item_id> -- reads the item's CURRENT fixture_base_sha
# from the real, tracked corpus.yaml (never hardcoded, so a future fixture-
# repo commit that moves base_sha doesn't silently desync this file from
# the corpus it reads).
real_fixture_base_sha() {
	local item_id="$1"
	python3 -c "
import yaml

with open('$CORPUS_YAML') as f:
    d = yaml.safe_load(f)
for it in d.get('items') or []:
    if it['id'] == '$item_id':
        print(it['fixture_base_sha'])
        break
"
}

# force_band_metric_no_interval <band_path> <item_id> <metric_id> -- rewrites
# one metric's block in a REAL, already-built band (never hand-authors a
# whole band, ADR-F03-08 -- only mutates one block of a real one) so it
# carries `insufficient_reps: true` and no accept_set/accept_lo, the same
# shape aggregate-runs.sh itself produces when a metric's own contributing
# n<2 (compute_stats). Used by TC-019k (UAT R2-F-1) to reproduce "band
# published no interval for this one metric" without needing to contrive a
# real n<2 scenario across an entire multi-rep band.
force_band_metric_no_interval() {
	local band_path="$1" item_id="$2" metric_id="$3"
	python3 -c "
import json

with open('$band_path') as f:
    band = json.load(f)
task = next(t for t in band['tasks'] if t['item_id'] == '$item_id')
block = task['metrics']['$metric_id']
block.pop('accept_set', None)
block.pop('accept_lo', None)
block.pop('accept_hi', None)
block['insufficient_reps'] = True
block['n'] = 1
with open('$band_path', 'w') as f:
    json.dump(band, f)
"
}

# band_metric_count <band_path> <item_id> -- number of metrics the band
# publishes for item_id, read from the real band file itself (never
# hardcoded), so TC-019k's structural invariant check (every band-published
# metric must appear in metrics_out) doesn't silently desync from a future
# metric-registry change.
band_metric_count() {
	local band_path="$1" item_id="$2"
	python3 -c "
import json

with open('$band_path') as f:
    band = json.load(f)
task = next(t for t in band['tasks'] if t['item_id'] == '$item_id')
print(len(task['metrics']))
"
}

REAL_FIXTURE_SHA="$(real_fixture_base_sha "$REAL_ITEM_ID")"
[[ -n "$REAL_FIXTURE_SHA" ]] || fail "setup: could not resolve $REAL_ITEM_ID's fixture_base_sha from $CORPUS_YAML"

# Shared valid band (REQ-F-027 (iii)'s only source of a real baseline_id),
# built once and reused by every sub-case that needs preconditions (i)/(ii)/
# (iii) to all hold.
BAND_VALID="$WORKDIR/band-valid.json"
build_band "$BAND_VALID" "$REAL_ITEM_ID" 3

# ---------------------------------------------------------------------------
# TC-019a (AC-22, REQ-F-027): decision table over the three preconditions,
# each isolated, plus a multi-failure case (task spec's own "preconditions
# report every failing check, not just the first") and an all-pass negative
# (stub invoked exactly once).
# ---------------------------------------------------------------------------
test_a() {
	# --- (i): ledger dir absent -- fixture_base_sha overridden to a value
	# with no bench/corpus/ledgers/<sha>/ directory. Item id and band both
	# still resolve (isolated). ---
	local bogus_sha="1111111111111111111111111111111111111111"
	local rec_i="$WORKDIR/a-i-record.jsonl"
	place_record "$WORKDIR/a-i-root" "$REAL_ITEM_ID" 1 --set "manifest.fixture_base_sha=\"$bogus_sha\""
	cp "$WORKDIR/a-i-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec_i"

	local log_i="$WORKDIR/a-i-run-one.log"
	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_i" \
		run_replay "$rec_i" "$BAND_VALID" "$WORKDIR/a-i-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "a(i): exited 0, want non-zero (ledger dir absent)"
	grep -q "$bogus_sha" "$err" || fail "a(i): stderr does not name the missing fixture_base_sha: $(cat "$err")"
	grep -q "ledgers/$bogus_sha" "$err" || fail "a(i): stderr does not name the missing ledger path: $(cat "$err")"
	[[ "$(invocation_count "$log_i")" == "0" ]] || fail "a(i): RUN_ONE_BIN was invoked despite a failed precondition"
	echo "TC-019a(i, ledger dir absent) PASS"

	# --- (i-partial), UAT F-10 (docs/review/.../uat-20260808-E40-F03.md):
	# the ledger directory EXISTS but is missing tests.json and/or
	# lint.json -- REQ-F-027 (i) names the retained artifacts themselves,
	# not the container directory. A container-existence check alone
	# (os.path.isdir) would pass this and dispatch run-one.sh, defeating
	# REQ-F-027's "a replay that cannot be valid costs no API spend"
	# against the most likely real form of the failure: a partially-
	# pruned ledger dir, not a wholesale-deleted one. A sibling-copy of
	# replay-manifest.sh (mirrors test_h's technique) is used so this
	# directory can be constructed without touching the real,
	# fully-populated bench/corpus/ledgers/ tree -- item id and band both
	# still resolve against the real corpus/BAND_VALID (isolated). ---
	local sibling_ip="$WORKDIR/a-ip-sibling"
	mkdir -p "$sibling_ip"
	cp "$REPLAY" "$sibling_ip/replay-manifest.sh"
	chmod +x "$sibling_ip/replay-manifest.sh"
	cp "$STUB_RUN_ONE" "$sibling_ip/run-one.sh"
	chmod +x "$sibling_ip/run-one.sh"
	cp "$AGGREGATE" "$sibling_ip/aggregate-runs.sh"
	chmod +x "$sibling_ip/aggregate-runs.sh"
	mkdir -p "$sibling_ip/../corpus/ledgers/$REAL_FIXTURE_SHA"
	# tests.json present, lint.json ABSENT -- an empty ledger dir would
	# also fail (ii) below covers the fully-empty case implicitly, but a
	# PARTIAL prune is the case UAT F-10 calls out explicitly, and it is
	# the case a "does anything exist under the dir" check would miss.
	echo '{}' >"$sibling_ip/../corpus/ledgers/$REAL_FIXTURE_SHA/tests.json"

	local rec_ip="$WORKDIR/a-ip-record.jsonl"
	place_record "$WORKDIR/a-ip-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/a-ip-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec_ip"

	local log_ip="$WORKDIR/a-ip-run-one.log"
	local out_ip err_ip code_ip
	out_ip="$(mktemp -p "$WORKDIR")"
	err_ip="$(mktemp -p "$WORKDIR")"
	set +e
	env -u RUN_ONE_BIN STUB_RUN_ONE_LOG="$log_ip" \
		"$sibling_ip/replay-manifest.sh" --record "$rec_ip" --band "$BAND_VALID" --out "$WORKDIR/a-ip-out" --corpus "$CORPUS_YAML" \
		</dev/null >"$out_ip" 2>"$err_ip"
	code_ip=$?
	set -e
	[[ "$code_ip" -ne 0 ]] || fail "a(i-partial): exited 0, want non-zero (ledger dir exists but lint.json missing)"
	grep -q "lint.json" "$err_ip" || fail "a(i-partial): stderr does not name the missing lint.json: $(cat "$err_ip")"
	grep -q "$REAL_FIXTURE_SHA" "$err_ip" || fail "a(i-partial): stderr does not name the fixture_base_sha: $(cat "$err_ip")"
	! grep -q "tests.json" "$err_ip" || fail "a(i-partial): stderr names tests.json as missing, but it was present: $(cat "$err_ip")"
	[[ "$(invocation_count "$log_ip")" == "0" ]] || fail "a(i-partial): RUN_ONE_BIN was invoked despite a failed precondition (partially-pruned ledger dir)"
	echo "TC-019a(i-partial, ledger dir exists but lint.json missing) PASS"

	# --- (ii): item id does not resolve in --corpus -- a temp copy of
	# corpus.yaml with the item's entry removed. Ledger dir and band both
	# still resolve (isolated: the record itself still names the real,
	# unedited fixture_base_sha). ---
	local rec_ii="$WORKDIR/a-ii-record.jsonl"
	place_record "$WORKDIR/a-ii-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/a-ii-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec_ii"
	local stripped_corpus="$WORKDIR/a-ii-corpus.yaml"
	strip_item_from_corpus "$stripped_corpus" "$REAL_ITEM_ID"

	local log_ii="$WORKDIR/a-ii-run-one.log"
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_ii" \
		run_replay "$rec_ii" "$BAND_VALID" "$WORKDIR/a-ii-out" "$stripped_corpus")
	[[ "$code" -ne 0 ]] || fail "a(ii): exited 0, want non-zero (item id unresolved)"
	grep -q "$REAL_ITEM_ID" "$err" || fail "a(ii): stderr does not name the unresolved item id: $(cat "$err")"
	[[ "$(invocation_count "$log_ii")" == "0" ]] || fail "a(ii): RUN_ONE_BIN was invoked despite a failed precondition"
	echo "TC-019a(ii, item id unresolved) PASS"

	# --- (iii): the --band aggregate has no entry for (item_id, variant_id)
	# -- a real aggregate built for a DIFFERENT item. Ledger dir and item
	# resolution both still hold (isolated). ---
	local rec_iii="$rec_ii" # same valid record as (ii), reused
	local band_other="$WORKDIR/a-iii-band.json"
	build_band "$band_other" "validate-sku-max-length" 3

	local log_iii="$WORKDIR/a-iii-run-one.log"
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_iii" \
		run_replay "$rec_iii" "$band_other" "$WORKDIR/a-iii-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "a(iii): exited 0, want non-zero (no band entry)"
	grep -q "$REAL_ITEM_ID" "$err" || fail "a(iii): stderr does not name the item with no band entry: $(cat "$err")"
	[[ "$(invocation_count "$log_iii")" == "0" ]] || fail "a(iii): RUN_ONE_BIN was invoked despite a failed precondition"
	echo "TC-019a(iii, no band entry) PASS"

	# --- multi-failure: (i) and (ii) fail simultaneously -- BOTH named on
	# stderr, zero invocations (task spec: "preconditions report every
	# failing check, not just the first"). A short-circuiting
	# implementation would report only the first and pass every isolated
	# sub-case above while failing this one. ---
	local rec_multi="$WORKDIR/a-multi-record.jsonl"
	place_record "$WORKDIR/a-multi-root" "$REAL_ITEM_ID" 1 --set "manifest.fixture_base_sha=\"$bogus_sha\""
	cp "$WORKDIR/a-multi-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec_multi"

	local log_multi="$WORKDIR/a-multi-run-one.log"
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_multi" \
		run_replay "$rec_multi" "$BAND_VALID" "$WORKDIR/a-multi-out" "$stripped_corpus")
	[[ "$code" -ne 0 ]] || fail "a(multi): exited 0, want non-zero"
	grep -q "$bogus_sha" "$err" || fail "a(multi): stderr does not name the missing ledger sha: $(cat "$err")"
	grep -q "$REAL_ITEM_ID" "$err" || fail "a(multi): stderr does not name the unresolved item id: $(cat "$err")"
	[[ "$(invocation_count "$log_multi")" == "0" ]] || fail "a(multi): RUN_ONE_BIN was invoked despite failed preconditions"
	echo "TC-019a(multi, two preconditions fail together) PASS"

	# --- negative: all three preconditions hold -- stub invoked exactly
	# once (proves the checks are gating, not always-failing). ---
	local rec_ok="$WORKDIR/a-ok-record.jsonl"
	place_record "$WORKDIR/a-ok-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/a-ok-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec_ok"

	# STUB_RUN_ONE_RECORD_FILE reuses rec_ok itself as the fresh record
	# (gen_fixtures is deterministic, so this is byte-identical to a real
	# reproduced run) -- without this, the stub's own field-less default
	# synthetic record would trip T-E40-F03-007's post-dispatch
	# variant_bundle_sha256/model_ids comparison against rec_ok's real
	# golden-derived values, which is not what this sub-case (dispatch
	# gating only) is testing.
	local log_ok="$WORKDIR/a-ok-run-one.log"
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_ok" STUB_RUN_ONE_RECORD_FILE="$rec_ok" \
		run_replay "$rec_ok" "$BAND_VALID" "$WORKDIR/a-ok-out" "$CORPUS_YAML")
	[[ "$code" -eq 0 ]] || fail "a(negative): exited $code, want 0 (all preconditions hold): $(cat "$err")"
	[[ "$(invocation_count "$log_ok")" == "1" ]] || fail "a(negative): RUN_ONE_BIN invocation count=$(invocation_count "$log_ok"), want exactly 1"
	echo "TC-019a(negative, all preconditions hold) PASS"

	echo "TC-019a PASS"
}

# ---------------------------------------------------------------------------
# TC-019b (AC-23, REQ-F-026/ADR-F03-02): the synthetic corpus file's actual
# field values, inspected via the stub's echo-capture mode -- not just the
# fact that dispatch happened.
# ---------------------------------------------------------------------------
test_b() {
	local rec="$WORKDIR/b-record.jsonl"
	place_record "$WORKDIR/b-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/b-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	# STUB_RUN_ONE_RECORD_FILE reuses rec itself (same reasoning as
	# test_a's a-ok sub-case above) so T-E40-F03-007's post-dispatch
	# identity comparison doesn't trip on the stub's field-less default
	# record -- this sub-case is about the synthetic corpus shape, not the
	# verdict.
	local captured="$WORKDIR/b-captured-corpus.yaml"
	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_ECHO_CORPUS_TO="$captured" STUB_RUN_ONE_RECORD_FILE="$rec" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/b-out" "$CORPUS_YAML")
	[[ "$code" -eq 0 ]] || fail "b: replay-manifest.sh exited $code, want 0: $(cat "$err")"
	[[ -f "$captured" ]] || fail "b: no synthetic corpus was captured -- RUN_ONE_BIN was never invoked with --corpus: $(cat "$err")"

	python3 -c "
import os
import sys

import yaml

with open('$captured') as f:
    synth = yaml.safe_load(f)

expected_sha = '$REAL_FIXTURE_SHA'
assert synth['fixture']['base_sha'] == expected_sha, 'fixture.base_sha=%r, want %r' % (synth['fixture']['base_sha'], expected_sha)
assert synth['schema_version'] == '1.0', 'schema_version=%r, want manifest.corpus_schema_version' % synth['schema_version']

items = synth.get('items') or []
assert len(items) == 1, 'items: holds %d entries, want exactly 1' % len(items)
item = items[0]
assert item['id'] == '$REAL_ITEM_ID', 'items[0].id=%r' % item['id']
assert item['fixture_base_sha'] == expected_sha, 'items[0].fixture_base_sha=%r, want %r (byte-identical to fixture.base_sha per corpus.yaml INVARIANT)' % (item['fixture_base_sha'], expected_sha)
assert item['fixture_base_sha'] == synth['fixture']['base_sha'], 'items[0].fixture_base_sha != fixture.base_sha -- violates corpus.yaml own INVARIANT'
assert item['p2p_set'] == 'default', 'items[0].p2p_set=%r' % item['p2p_set']

assert os.path.isabs(item['seed_path']), 'items[0].seed_path is not absolute: %r' % item['seed_path']
assert os.path.isfile(item['seed_path']), 'items[0].seed_path does not resolve to a real file: %r' % item['seed_path']

f2p_paths = item['f2p']['paths']
assert f2p_paths, 'items[0].f2p.paths is empty'
for p in f2p_paths:
    assert os.path.isabs(p), 'items[0].f2p.paths entry is not absolute: %r' % p
    assert os.path.isfile(p), 'items[0].f2p.paths entry does not resolve to a real file: %r' % p

assert 'negative_items' not in synth, 'negative_items key is present in the synthetic corpus, want omitted entirely'
" || fail "b: synthetic corpus shape check failed"

	echo "TC-019b PASS"
}

# ---------------------------------------------------------------------------
# TC-019c (AC-24, REQ-F-028/ADR-F03-02): a curator-edit simulation -- a temp
# copy of corpus.yaml with the target item's fixture_base_sha diverged from
# the stored record's manifest. Drift is recorded, not a precondition
# failure: the stub is still invoked, and the synthetic corpus dispatched
# still carries the MANIFEST's (not the live/edited) value.
# ---------------------------------------------------------------------------
test_c() {
	local rec="$WORKDIR/c-record.jsonl"
	place_record "$WORKDIR/c-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/c-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local drifted_sha="9999999999999999999999999999999999999999"
	local drifted_corpus="$WORKDIR/c-corpus.yaml"
	edit_item_fixture_sha "$drifted_corpus" "$REAL_ITEM_ID" "$drifted_sha"

	local log="$WORKDIR/c-run-one.log"
	local captured="$WORKDIR/c-captured-corpus.yaml"
	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_ECHO_CORPUS_TO="$captured" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/c-out" "$drifted_corpus")

	# AC-24 does not pin an exit code for this task's own (partial)
	# verdict computation -- T-E40-F03-007 finalizes pass/fail/invalid.
	# Only the three AC-24 assertions are checked here: dispatch still
	# happened, the drift is named with both values, and the synthetic
	# corpus still carries the manifest's SHA.
	[[ "$(invocation_count "$log")" == "1" ]] || fail "c: RUN_ONE_BIN invocation count=$(invocation_count "$log"), want exactly 1 (drift is recorded, not a precondition failure)"
	[[ -s "$out" ]] || fail "c: stdout is empty, want a verification document naming the drift: $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)

reasons = doc.get('reasons') or []
matches = [r for r in reasons if r.get('reason') == 'corpus_drift' and r.get('field') == 'fixture_base_sha']
assert matches, 'no corpus_drift reasons entry for fixture_base_sha in %r' % reasons
entry = matches[0]
assert entry['expected'] == '$REAL_FIXTURE_SHA', 'expected=%r, want manifest value %r' % (entry['expected'], '$REAL_FIXTURE_SHA')
assert entry['actual'] == '$drifted_sha', 'actual=%r, want live (edited) value %r' % (entry['actual'], '$drifted_sha')
" || fail "c: verification document's corpus_drift entry check failed: $(cat "$out")"

	[[ -f "$captured" ]] || fail "c: no synthetic corpus was captured"
	python3 -c "
import yaml

with open('$captured') as f:
    synth = yaml.safe_load(f)
expected_sha = '$REAL_FIXTURE_SHA'
assert synth['fixture']['base_sha'] == expected_sha, 'synthetic fixture.base_sha=%r, want the MANIFEST value %r, not the live/edited one' % (synth['fixture']['base_sha'], expected_sha)
assert synth['items'][0]['fixture_base_sha'] == expected_sha, 'synthetic items[0].fixture_base_sha=%r, want the MANIFEST value %r' % (synth['items'][0]['fixture_base_sha'], expected_sha)
" || fail "c: synthetic corpus dispatched the live/edited SHA instead of the manifest's"

	echo "TC-019c PASS"
}

# ---------------------------------------------------------------------------
# TC-019d (AC-25, REQ-F-029/ADR-F03-05): a stub-produced fresh record whose
# manifest.variant_bundle_sha256 differs from the stored record's,
# model_ids matching -- verdict invalid, reason variant_bundle_drift only,
# never fail.
# ---------------------------------------------------------------------------
test_d() {
	local rec="$WORKDIR/d-record.jsonl"
	place_record "$WORKDIR/d-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/d-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local fresh="$WORKDIR/d-fresh-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set 'manifest.variant_bundle_sha256="deadbeef0000000000000000000000000000000drifted"' >"$fresh"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/d-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "d: exited 0, want non-zero (variant_bundle_sha256 drift -> invalid)"
	[[ -s "$out" ]] || fail "d: stdout empty, want a verification document: $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
with open('$rec') as f:
    stored = json.loads(f.readline())
with open('$fresh') as f:
    replayed = json.loads(f.readline())

assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (never fail on drifted inputs)' % doc.get('verdict')
reasons = doc.get('reasons') or []
bundle_reasons = [r for r in reasons if r.get('reason') == 'variant_bundle_drift']
assert len(bundle_reasons) == 1, 'reasons=%r, want exactly one variant_bundle_drift entry' % reasons
model_reasons = [r for r in reasons if r.get('reason') == 'model_version_drift']
assert not model_reasons, 'model_version_drift present when only the bundle hash drifted: %r' % reasons
entry = bundle_reasons[0]
assert entry['expected'] == stored['manifest']['variant_bundle_sha256'], 'expected=%r, want the stored record value %r' % (entry['expected'], stored['manifest']['variant_bundle_sha256'])
assert entry['actual'] == replayed['manifest']['variant_bundle_sha256'], 'actual=%r, want the replayed record value %r' % (entry['actual'], replayed['manifest']['variant_bundle_sha256'])
assert entry['expected'] != entry['actual'], 'expected and actual are identical, want distinct hashes named'
" || fail "d: verification document check failed: $(cat "$out")"

	echo "TC-019d PASS"
}

# ---------------------------------------------------------------------------
# TC-019e (AC-26, REQ-F-029/ADR-F03-05): a stub-produced fresh record whose
# manifest.model_ids differs from the stored record's, variant_bundle_sha256
# matching -- verdict invalid, reason model_version_drift only, both ID
# lists named in full.
# ---------------------------------------------------------------------------
test_e() {
	local rec="$WORKDIR/e-record.jsonl"
	place_record "$WORKDIR/e-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/e-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local fresh="$WORKDIR/e-fresh-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set 'manifest.model_ids=["claude-drifted-model"]' >"$fresh"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/e-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "e: exited 0, want non-zero (model_ids drift -> invalid)"
	[[ -s "$out" ]] || fail "e: stdout empty, want a verification document: $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
with open('$rec') as f:
    stored = json.loads(f.readline())
with open('$fresh') as f:
    replayed = json.loads(f.readline())

assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid' % doc.get('verdict')
reasons = doc.get('reasons') or []
model_reasons = [r for r in reasons if r.get('reason') == 'model_version_drift']
assert len(model_reasons) == 1, 'reasons=%r, want exactly one model_version_drift entry' % reasons
bundle_reasons = [r for r in reasons if r.get('reason') == 'variant_bundle_drift']
assert not bundle_reasons, 'variant_bundle_drift present when only model_ids drifted: %r' % reasons
entry = model_reasons[0]
assert entry['expected'] == stored['manifest']['model_ids'], 'expected=%r, want the stored record value %r (full list)' % (entry['expected'], stored['manifest']['model_ids'])
assert entry['actual'] == replayed['manifest']['model_ids'], 'actual=%r, want the replayed record value %r (full list)' % (entry['actual'], replayed['manifest']['model_ids'])
assert entry['expected'] != entry['actual'], 'expected and actual are identical, want distinct ID lists named'
" || fail "e: verification document check failed: $(cat "$out")"

	echo "TC-019e PASS"
}

# ---------------------------------------------------------------------------
# TC-019f (AC-27, REQ-F-025/030): (a) every headline metric inside its
# published interval -> pass; (b) exactly one metric's fresh value moved
# outside its interval -> fail, naming that metric/value/interval, every
# other metric still individually pass. Both sub-cases assert the stored
# --rep value was passed to RUN_ONE_BIN, --out is a distinct root, and the
# fresh record's manifest.run_key is byte-identical to the stored one's.
# ---------------------------------------------------------------------------
test_f() {
	local rec="$WORKDIR/f-record.jsonl"
	place_record "$WORKDIR/f-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/f-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	# --- (a) positive: fresh record identical to the golden-derived stored
	# one (same item_id/rep) -- every metric's fresh (n=1) value equals the
	# band's own min=median=max (BAND_VALID's 3 reps are themselves
	# identical golden-derived values), so every comparable metric lands
	# exactly inside its published interval. ---
	local fresh_ok="$WORKDIR/f-fresh-ok-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 >"$fresh_ok"

	local log_ok="$WORKDIR/f-ok-run-one.log"
	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log_ok" STUB_RUN_ONE_RECORD_FILE="$fresh_ok" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/f-ok-out" "$CORPUS_YAML")
	[[ "$code" -eq 0 ]] || fail "f(a): exited $code, want 0 (every metric inside its interval -> pass): $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'pass', 'verdict=%r, want pass: %r' % (doc.get('verdict'), doc)
metrics = doc.get('metrics') or []
assert metrics, 'metrics[] is empty, want at least one comparable headline metric'
# UAT R2-F-1: this real corpus item's own quality.tests_pass gate is
# gate_not_executed in the golden fixture (toolchain_guard=pass but
# tests_pass=null) -- BAND_VALID's own 3 reps therefore publish
# quality_tests_pass as insufficient_reps (band_no_interval), REAL data,
# not a fixture bug. band_no_interval is traced but must not by itself
# fail this all-pass assertion (AC-27 -- distinct from a genuine 'fail').
scored = [m for m in metrics if m.get('verdict') in ('pass', 'fail')]
assert scored, 'no metric in metrics[] carried a pass/fail score: %r' % metrics
bad = [m for m in scored if m.get('verdict') != 'pass']
assert not bad, 'scored metrics with a non-pass verdict in the all-pass case: %r' % bad
band_no_interval = [m for m in metrics if m.get('reason') == 'band_no_interval']
assert all(m.get('metric') == 'quality_tests_pass' for m in band_no_interval), 'unexpected band_no_interval metrics: %r' % band_no_interval
assert doc['stored']['run_key'] == doc['replayed']['run_key'], 'run_key mismatch: stored=%r replayed=%r' % (doc['stored']['run_key'], doc['replayed']['run_key'])
" || fail "f(a): verification document check failed: $(cat "$out")"

	python3 -c "
import json

argv = None
with open('$log_ok') as f:
    for line in f:
        argv = json.loads(line)['argv']
assert argv is not None, 'no invocation logged'
assert '--rep' in argv and argv[argv.index('--rep') + 1] == '1', 'argv=%r, want --rep 1 (the manifest STORED value)' % argv
assert '--out' in argv, 'argv=%r, missing --out' % argv
out_arg = argv[argv.index('--out') + 1]
assert out_arg != '$WORKDIR/f-root', 'RUN_ONE_BIN was invoked with --out equal to the stored record artifact root, want a distinct root'
" || fail "f(a): RUN_ONE_BIN invocation argv check failed"
	echo "TC-019f(a, every metric inside its interval) PASS"

	# --- (b) negative: exactly one metric (loc.prod_added, golden value 17)
	# moved far outside its published interval ([15.3, 18.7], since
	# BAND_VALID's 3 reps are identical -> r=0 -> r_eff=0.10*median). ---
	local fresh_bad="$WORKDIR/f-fresh-bad-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set loc.prod_added=100 >"$fresh_bad"

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh_bad" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/f-bad-out" "$CORPUS_YAML")
	[[ "$code2" -ne 0 ]] || fail "f(b): exited 0, want non-zero (one metric outside its interval -> fail)"

	python3 -c "
import json

with open('$out2') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'fail', 'verdict=%r, want fail' % doc.get('verdict')
metrics = doc.get('metrics') or []
moved = [m for m in metrics if m.get('metric') == 'loc_prod_added']
assert moved, 'metrics=%r, want a loc_prod_added entry' % metrics
entry = moved[0]
assert entry['verdict'] == 'fail', 'loc_prod_added verdict=%r, want fail' % entry['verdict']
assert entry['replayed_value'] == 100, 'loc_prod_added replayed_value=%r, want 100' % entry['replayed_value']
assert 'accept_lo' in entry and 'accept_hi' in entry, 'loc_prod_added entry missing its published interval: %r' % entry
others = [m for m in metrics if m.get('metric') != 'loc_prod_added']
assert others, 'no other metrics present -- fail should be metric-scoped, not blanket'
# UAT R2-F-1: quality_tests_pass is a real band_no_interval entry for this
# item (see f(a)'s comment) -- excluded from the pass-scoped check below,
# same as f(a).
scored_others = [m for m in others if m.get('verdict') in ('pass', 'fail')]
bad_others = [m for m in scored_others if m.get('verdict') != 'pass']
assert not bad_others, 'other scored metrics not individually pass despite only loc_prod_added moving: %r' % bad_others
assert doc['stored']['run_key'] == doc['replayed']['run_key'], 'run_key mismatch: stored=%r replayed=%r' % (doc['stored']['run_key'], doc['replayed']['run_key'])
" || fail "f(b): verification document check failed: $(cat "$out2")"
	echo "TC-019f(b, one metric outside its interval) PASS"

	echo "TC-019f PASS"
}

# ---------------------------------------------------------------------------
# TC-019g (AC-25/AC-26's uncovered both-drift cell, test-plan.md's pinned
# resolution): variant_bundle_sha256 AND model_ids both differ
# simultaneously -- verdict invalid, reasons[] contains BOTH
# variant_bundle_drift and model_version_drift, each with its own
# expected/actual pair, never one reason subsuming the other.
# ---------------------------------------------------------------------------
test_g() {
	local rec="$WORKDIR/g-record.jsonl"
	place_record "$WORKDIR/g-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/g-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local fresh="$WORKDIR/g-fresh-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set 'manifest.variant_bundle_sha256="deadbeef0000000000000000000000000000000drifted"' \
		--set 'manifest.model_ids=["claude-drifted-model"]' >"$fresh"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/g-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "g: exited 0, want non-zero (both fields drift -> invalid)"
	[[ -s "$out" ]] || fail "g: stdout empty, want a verification document: $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
with open('$rec') as f:
    stored = json.loads(f.readline())
with open('$fresh') as f:
    replayed = json.loads(f.readline())

assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid' % doc.get('verdict')
reasons = doc.get('reasons') or []
bundle_reasons = [r for r in reasons if r.get('reason') == 'variant_bundle_drift']
model_reasons = [r for r in reasons if r.get('reason') == 'model_version_drift']
assert len(bundle_reasons) == 1, 'reasons=%r, want exactly one variant_bundle_drift entry (both drifted simultaneously)' % reasons
assert len(model_reasons) == 1, 'reasons=%r, want exactly one model_version_drift entry (both drifted simultaneously)' % reasons
b, m = bundle_reasons[0], model_reasons[0]
assert b['expected'] == stored['manifest']['variant_bundle_sha256'] and b['actual'] == replayed['manifest']['variant_bundle_sha256'], 'variant_bundle_drift entry does not name its own expected/actual pair: %r' % b
assert m['expected'] == stored['manifest']['model_ids'] and m['actual'] == replayed['manifest']['model_ids'], 'model_version_drift entry does not name its own expected/actual pair: %r' % m
" || fail "g: verification document check failed: $(cat "$out")"

	echo "TC-019g PASS"
}

# ---------------------------------------------------------------------------
# TC-019h (RUN_ONE_BIN default resolution, TD-077's defect class -- same
# treatment as TC-017f): replay-manifest.sh invoked with RUN_ONE_BIN unset
# resolves $SCRIPT_DIR/run-one.sh (the sibling path), never a bare
# PATH-resolved name.
# ---------------------------------------------------------------------------
test_h() {
	local sibling_dir="$WORKDIR/h-sibling"
	mkdir -p "$sibling_dir"
	cp "$REPLAY" "$sibling_dir/replay-manifest.sh"
	chmod +x "$sibling_dir/replay-manifest.sh"
	cp "$STUB_RUN_ONE" "$sibling_dir/run-one.sh"
	chmod +x "$sibling_dir/run-one.sh"
	# T-E40-F03-007's own comparison primitive (AGGREGATE_BIN) is always the
	# sibling path, never overridable -- the copied replay-manifest.sh needs
	# its own aggregate-runs.sh sibling or its startup existence check fails
	# before RUN_ONE_BIN resolution is ever reached.
	cp "$AGGREGATE" "$sibling_dir/aggregate-runs.sh"
	chmod +x "$sibling_dir/aggregate-runs.sh"
	# replay-manifest.sh sources lib/path-safety.sh relative to its own
	# $SCRIPT_DIR (NEW-1's shared-lib fix) -- the copied sibling deployment
	# needs that lib/ directory alongside it too.
	mkdir -p "$sibling_dir/lib"
	cp "$SCRIPTS_DIR/lib/path-safety.sh" "$sibling_dir/lib/path-safety.sh"

	# replay-manifest.sh derives BENCH_DIR from ITS OWN location
	# ($SCRIPT_DIR/..) for the ledger-directory precondition (spec.md:
	# resolved from the script's own location, never --corpus's directory)
	# -- so the fake bench tree needs its own corpus/ledgers/<sha>/ sibling
	# to the copied script, or precondition (i) fails before this test ever
	# reaches RUN_ONE_BIN resolution. Precondition (i) requires the
	# retained tests.json/lint.json FILES, not just the directory (UAT
	# F-10) -- both are created here so this test (RUN_ONE_BIN resolution
	# only) isn't tripped up by the unrelated ledger-artifacts precondition.
	local fake_sha="2222222222222222222222222222222222222222"
	mkdir -p "$sibling_dir/../corpus/ledgers/$fake_sha"
	echo '{}' >"$sibling_dir/../corpus/ledgers/$fake_sha/tests.json"
	echo '{}' >"$sibling_dir/../corpus/ledgers/$fake_sha/lint.json"

	local decoy_dir="$WORKDIR/h-decoy" decoy_marker="$WORKDIR/h-decoy-invoked"
	mkdir -p "$decoy_dir"
	cat >"$decoy_dir/run-one.sh" <<DECOY
#!/usr/bin/env bash
touch "$decoy_marker"
exit 0
DECOY
	chmod +x "$decoy_dir/run-one.sh"

	local corpus="$WORKDIR/h-corpus.yaml"
	cat >"$corpus" <<YAML
schema_version: "1.0"
fixture:
  base_sha: "$fake_sha"
p2p_sets:
  default:
    packages: ["./..."]
items:
  - id: alpha
    type: task
    seed_path: seed.yaml
    f2p:
      paths: ["f2p_test.go"]
      test_names: ["pkg::TestAlpha"]
    p2p_set: default
    fixture_base_sha: "$fake_sha"
YAML

	local record="$WORKDIR/h-record.jsonl"
	cat >"$record" <<JSON
{"manifest": {"item_id": "alpha", "variant_id": "default", "rep": 1, "run_key": "alpha::default::rep1", "fixture_base_sha": "$fake_sha", "corpus_schema_version": "1.0", "p2p_set": "default"}, "outcome": "completed"}
JSON

	local band="$WORKDIR/h-band.json"
	python3 -c "
import json

band = {
    'baseline_id': 'default-${fake_sha:0:12}-r1',
    'provenance': {'fixture_base_sha': '$fake_sha', 'reps': 1},
    'tasks': [{'item_id': 'alpha'}],
}
json.dump(band, open('$band', 'w'))
"

	case ":$PATH:" in
	*":$sibling_dir:"*)
		fail "h: test setup accidentally put $sibling_dir on PATH -- this would test PATH-resolution, not sibling-path-default resolution"
		;;
	esac

	local out_dir="$WORKDIR/h-out" log="$WORKDIR/h-run-one.log"
	local out="$WORKDIR/h.out" err="$WORKDIR/h.err"
	set +e
	env -u RUN_ONE_BIN \
		PATH="$decoy_dir:$PATH" \
		STUB_RUN_ONE_LOG="$log" \
		"$sibling_dir/replay-manifest.sh" --record "$record" --band "$band" --out "$out_dir" --corpus "$corpus" \
		</dev/null >"$out" 2>"$err"
	local code=$?
	set -e
	# This test is about RUN_ONE_BIN's default-resolution seam only, not
	# the verdict -- the minimal synthetic band above (UAT F-1's own fix)
	# publishes no metrics for 'alpha' at all, so the run legitimately
	# verdicts invalid/non-zero (no_comparable_metrics, T-E40-F03-007).
	# Dispatch having happened at all (proven below) is what this test
	# actually needs.
	[[ -s "$out" ]] || fail "h: replay-manifest.sh produced no verification document: $(cat "$err")"

	[[ ! -f "$decoy_marker" ]] || fail "h: the PATH-resolved decoy run-one.sh was invoked -- RUN_ONE_BIN defaulted to a bare, PATH-resolved name instead of the sibling path"

	[[ -f "$log" ]] || fail "h: sibling-path run-one.sh stand-in was never invoked -- RUN_ONE_BIN did not default to \$SCRIPT_DIR/run-one.sh: $(cat "$err")"
	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "h: sibling-path run-one.sh stand-in invocation count=$calls, want exactly 1"

	echo "TC-019h PASS"
}

# ---------------------------------------------------------------------------
# TC-019i (UAT F-1, CRITICAL, docs/review/.../uat-20260808-E40-F03.md): the
# aggregator's own non-pass classification of the freshly replayed record
# must never be silently accepted as a comparison success, and a comparable
# band metric the fresh record cannot supply must never be silently
# dropped. Three sub-cases from the UAT reproduction table:
#   (a) anomaly (agg_rc != 0, full document still printed per REQ-F-009) ->
#       invalid, reason aggregator_anomaly, never pass.
#   (b) explained_absence (agg_rc == 0) but the band's comparable metrics
#       (e.g. oracle_f2p_resolved) are missing from the fresh record ->
#       invalid. THE TRAP: a fix that only checks agg_rc makes (a) invalid
#       and leaves this sub-case still returning pass.
#   (c) every family absent -> 0 comparable metrics -> invalid, never a
#       vacuous pass over an empty/all-not_comparable metrics[].
# ---------------------------------------------------------------------------
test_i() {
	local rec="$WORKDIR/i-record.jsonl"
	place_record "$WORKDIR/i-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/i-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	# --- (a) anomaly: oracle/quality/loc all unset, no explanation on the
	# record (same fixture shape as tc018c's full-anomaly case). ---
	local fresh_anomaly="$WORKDIR/i-fresh-anomaly.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--unset oracle --unset quality --unset loc \
		--unset sources.oracle --unset sources.quality --unset sources.loc >"$fresh_anomaly"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh_anomaly" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/i-a-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "i(a): exited 0, want non-zero (replayed record classifies anomaly -- UAT F-1 case a)"
	[[ -s "$out" ]] || fail "i(a): stdout empty, want a verification document: $(cat "$err")"
	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (UAT F-1 case a: aggregator classified the replayed run anomaly)' % doc.get('verdict')
reasons = doc.get('reasons') or []
assert any(r.get('reason') == 'aggregator_anomaly' for r in reasons), 'reasons=%r, want an aggregator_anomaly entry' % reasons
" || fail "i(a): verification document check failed: $(cat "$out")"
	echo "TC-019i(a, anomaly -> invalid, never pass) PASS"

	# --- (b) explained_absence (toolchain_guard abort, same fixture shape
	# as tc018e): agg_rc == 0, but oracle_f2p_resolved and other band
	# metrics are excluded family_absent by the fresh record itself. THE
	# TRAP: an agg_rc-only fix leaves this sub-case still 'pass'. ---
	local fresh_explained="$WORKDIR/i-fresh-explained.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set 'quality.toolchain_guard=go_version_mismatch' \
		--unset quality.fmt_clean --unset quality.vet_ok --unset quality.tests_pass \
		--unset quality.lint_new_issues --unset quality.lint_new_issues_count \
		--unset oracle --unset loc \
		--unset sources.oracle --unset sources.loc >"$fresh_explained"

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh_explained" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/i-b-out" "$CORPUS_YAML")
	[[ "$code2" -ne 0 ]] || fail "i(b): exited 0, want non-zero (UAT F-1 case b, the trap: explained_absence still drops comparable band metrics)"
	[[ -s "$out2" ]] || fail "i(b): stdout empty, want a verification document: $(cat "$err2")"
	python3 -c "
import json

with open('$out2') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (UAT F-1 case b: explained_absence still drops oracle_f2p_resolved etc)' % doc.get('verdict')
metrics = doc.get('metrics') or []
oracle = [m for m in metrics if m.get('metric') == 'oracle_f2p_resolved']
assert oracle, 'metrics=%r, want an oracle_f2p_resolved entry accounted for (never silently dropped)' % metrics
assert oracle[0].get('verdict') == 'not_comparable', 'oracle_f2p_resolved entry=%r, want verdict not_comparable' % oracle[0]
reasons = doc.get('reasons') or []
assert any(r.get('reason') == 'metric_not_comparable' for r in reasons), 'reasons=%r, want a metric_not_comparable entry' % reasons
" || fail "i(b): verification document check failed: $(cat "$out2")"
	echo "TC-019i(b, explained_absence still drops band metrics -> invalid, the trap) PASS"

	# --- (c) every family absent -- 0 comparable metrics. An empty (or
	# all-not_comparable) metrics_out must never yield pass. ---
	local fresh_empty="$WORKDIR/i-fresh-empty.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--unset oracle --unset quality --unset loc \
		--unset stages --unset timing --unset runresult --unset rejections \
		--unset sources.oracle --unset sources.quality --unset sources.loc >"$fresh_empty"

	local out3 err3 code3
	{
		read -r out3
		read -r err3
		read -r code3
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh_empty" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/i-c-out" "$CORPUS_YAML")
	[[ "$code3" -ne 0 ]] || fail "i(c): exited 0, want non-zero (UAT F-1 case c: all families absent -> 0 comparable metrics, still invalid)"
	[[ -s "$out3" ]] || fail "i(c): stdout empty, want a verification document: $(cat "$err3")"
	python3 -c "
import json

with open('$out3') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (UAT F-1 case c: 0 comparable metrics must never yield pass)' % doc.get('verdict')
" || fail "i(c): verification document check failed: $(cat "$out3")"
	echo "TC-019i(c, 0 comparable metrics -> invalid, never a vacuous pass) PASS"

	echo "TC-019i PASS"
}

# ---------------------------------------------------------------------------
# TC-019j (UAT F-6, HIGH): manifest.run_key is REQ-F-025's join key but was
# only ever displayed (identity()'s IDENTITY_FIELDS drives display only),
# never compared. A fresh record whose run_key differs from the stored
# one's -- e.g. a producer-side rep-renumbering bug, REQ-F-025's own named
# risk -- must verdict invalid with reason run_key_mismatch, both values
# named, even when every other identity field matches. Uses --rep 99 (never
# --set manifest.run_key, which gen_fixtures.py reserves -- run_key is
# always recomputed from --item-id/--rep) so the fresh record's own run_key
# is genuinely mismatched while the STUB_RUN_ONE_RECORD_FILE is still
# copied verbatim to the stored rep's computed output path.
# ---------------------------------------------------------------------------
test_j() {
	local rec="$WORKDIR/j-record.jsonl"
	place_record "$WORKDIR/j-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/j-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local fresh="$WORKDIR/j-fresh-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 99 >"$fresh"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh" \
		run_replay "$rec" "$BAND_VALID" "$WORKDIR/j-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "j: exited 0, want non-zero (run_key mismatch -> invalid)"
	[[ -s "$out" ]] || fail "j: stdout empty, want a verification document: $(cat "$err")"

	python3 -c "
import json

with open('$out') as f:
    doc = json.load(f)
with open('$rec') as f:
    stored = json.loads(f.readline())
with open('$fresh') as f:
    replayed = json.loads(f.readline())

assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (UAT F-6: run_key differs but every other identity field matches)' % doc.get('verdict')
reasons = doc.get('reasons') or []
mismatch = [r for r in reasons if r.get('reason') == 'run_key_mismatch']
assert len(mismatch) == 1, 'reasons=%r, want exactly one run_key_mismatch entry' % reasons
entry = mismatch[0]
assert entry['expected'] == stored['manifest']['run_key'], 'expected=%r, want the stored run_key %r' % (entry['expected'], stored['manifest']['run_key'])
assert entry['actual'] == replayed['manifest']['run_key'], 'actual=%r, want the replayed run_key %r' % (entry['actual'], replayed['manifest']['run_key'])
assert entry['expected'] != entry['actual'], 'expected and actual are identical, want distinct run_keys named'
" || fail "j: verification document check failed: $(cat "$out")"

	echo "TC-019j PASS"
}

# ---------------------------------------------------------------------------
# TC-019k (UAT R2-F-1, HIGH, docs/review/.../uat-20260808-231500-E40-F03.md):
# Class-1/Class-2 recurrence of round-1's CRITICAL F-1 fix -- that fix
# handled the fresh side unable to supply a band-published metric; the
# per-metric comparison loop's OTHER branch (the band side publishing no
# interval/accept_set at all, e.g. insufficient_reps on the band side) was
# left unhandled and silently dropped the metric from metrics_out. Uses a
# real, already-built 3-rep band (BAND_VALID-shaped, via build_band) with
# ONE metric (oracle_f2p_resolved, the Phase-1 discriminative metric named
# in the UAT repro) force-rewritten to insufficient_reps -- every other
# metric keeps its real accept_set/accept_lo -- compared against a clean
# fresh replay (the stored record itself, unmodified). Must verdict
# invalid, name oracle_f2p_resolved as not_comparable/band_no_interval, and
# the structural invariant (every band-published metric accounted for in
# metrics_out) must hold.
#
# Verdict note: band-side no-interval is accounted for (metrics[]/
# reasons[]) but must NOT by itself force `invalid` -- unlike the fresh-
# side-can't-supply-a-value case (round-1's F-1, still `metric_not_
# comparable` -> invalid), the band never published a claim about this
# metric, so a replay cannot have violated one (AC-13/REQ-F-016's
# insufficient_reps is already surfaced at publish time by report-
# baseline.sh's flags.insufficient_reps). AC-27's `pass` must stay
# reachable even when a band carries an insufficient_reps metric --
# otherwise `pass` is unreachable for any band with ANY such metric, which
# would make REQ-F-030's whole three-valued verdict dead code for a band
# like this one. Every OTHER metric (with a real interval) still gets its
# own pass/fail, so the fresh replay here still verdicts pass.
# ---------------------------------------------------------------------------
test_k() {
	local rec="$WORKDIR/k-record.jsonl"
	place_record "$WORKDIR/k-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/k-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local band="$WORKDIR/k-band.json"
	build_band "$band" "$REAL_ITEM_ID" 3
	local expected_count
	expected_count="$(band_metric_count "$band" "$REAL_ITEM_ID")"
	[[ "$expected_count" -gt 1 ]] || fail "k: setup: band publishes only $expected_count metric(s), need >1 to isolate one dropped metric"
	force_band_metric_no_interval "$band" "$REAL_ITEM_ID" "oracle_f2p_resolved"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$rec" \
		run_replay "$rec" "$band" "$WORKDIR/k-out" "$CORPUS_YAML")
	[[ "$code" -eq 0 ]] || fail "k: exited $code, want 0 (band-side insufficient_reps for one metric must not force invalid when every other metric still passes -- UAT R2-F-1): $(cat "$err")"
	[[ -s "$out" ]] || fail "k: stdout empty, want a verification document: $(cat "$err")"

	EXPECTED_COUNT="$expected_count" python3 -c "
import json
import os

expected_count = int(os.environ['EXPECTED_COUNT'])

with open('$out') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'pass', 'verdict=%r, want pass (band_no_interval traces the metric but must not force invalid -- AC-27)' % doc.get('verdict')
metrics = doc.get('metrics') or []
assert len(metrics) == expected_count, 'metrics has %d entries, want %d -- every band-published metric must be accounted for (structural invariant)' % (len(metrics), expected_count)
oracle = [m for m in metrics if m.get('metric') == 'oracle_f2p_resolved']
assert oracle, 'metrics=%r, want an oracle_f2p_resolved entry -- band_no_interval must never be silently dropped' % metrics
assert oracle[0].get('verdict') == 'not_comparable', 'oracle_f2p_resolved entry=%r, want verdict not_comparable' % oracle[0]
assert oracle[0].get('reason') == 'band_no_interval', 'oracle_f2p_resolved entry=%r, want reason band_no_interval' % oracle[0]
reasons = doc.get('reasons') or []
band_reasons = [r for r in reasons if r.get('reason') == 'band_no_interval']
assert len(band_reasons) == 1 and 'oracle_f2p_resolved' in band_reasons[0].get('metrics', []), 'reasons=%r, want a band_no_interval entry naming oracle_f2p_resolved' % reasons
assert not any(r.get('reason') == 'metric_not_comparable' for r in reasons), 'reasons=%r, want NO metric_not_comparable entry (band_no_interval is a distinct, non-invalidating reason)' % reasons
quality_reason = [r for r in band_reasons if 'quality_tests_pass' in r.get('metrics', [])]
assert quality_reason, 'reasons=%r, want quality_tests_pass folded into the same band_no_interval entry (real gate_not_executed data for this item, see f(a))' % reasons
# quality_tests_pass is a SECOND, genuinely real band_no_interval metric
# for this item (gate_not_executed in the golden fixture, see f(a)'s own
# comment) -- excluded here alongside oracle_f2p_resolved so this
# assertion is scoped to metrics this sub-case actually forced.
other_metrics = [m for m in metrics if m.get('metric') not in ('oracle_f2p_resolved', 'quality_tests_pass')]
assert other_metrics, 'metrics=%r, want at least one other metric still individually scored (band_no_interval must not swallow the whole comparison)' % metrics
assert all(m.get('verdict') in ('pass', 'fail') for m in other_metrics), 'other metrics=%r, want every non-affected metric still individually pass/fail-scored' % other_metrics
assert all(m.get('verdict') == 'pass' for m in other_metrics), 'other metrics=%r, want every non-affected metric to pass (clean fresh replay against its own band)' % other_metrics
" || fail "k: verification document check failed: $(cat "$out")"

	echo "TC-019k PASS"
}

# ---------------------------------------------------------------------------
# TC-019m (UAT R2-F-1 companion, REQ-N-005): TC-019k proves band_no_interval
# does NOT by itself force invalid when other metrics still score. This is
# the other half -- when EVERY band-published metric is band_no_interval
# (a single-rep band: n=1 for every metric -> insufficient_reps
# universally, compute_stats), metrics_out is non-empty (every metric is
# accounted for, per the structural invariant) but ZERO of them carry a
# pass/fail score. Retargeting the vacuous-pass guard from "metrics_out
# empty" to "no scored metric" (this task's fix) is what keeps this
# invalid -- the old `elif not metrics_out` guard would have missed this
# case entirely once band_no_interval entries started populating
# metrics_out, silently regressing REQ-N-005 into a vacuous pass.
# ---------------------------------------------------------------------------
test_m() {
	local rec="$WORKDIR/m-record.jsonl"
	place_record "$WORKDIR/m-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/m-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local band="$WORKDIR/m-band.json"
	build_band "$band" "$REAL_ITEM_ID" 1
	local expected_count
	expected_count="$(band_metric_count "$band" "$REAL_ITEM_ID")"
	[[ "$expected_count" -gt 0 ]] || fail "m: setup: band publishes 0 metrics, need at least 1"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$rec" \
		run_replay "$rec" "$band" "$WORKDIR/m-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "m: exited 0, want non-zero (every band metric is band_no_interval -- zero scored metrics must never be a vacuous pass, REQ-N-005)"
	[[ -s "$out" ]] || fail "m: stdout empty, want a verification document: $(cat "$err")"

	EXPECTED_COUNT="$expected_count" python3 -c "
import json
import os

expected_count = int(os.environ['EXPECTED_COUNT'])

with open('$out') as f:
    doc = json.load(f)
assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (zero scored metrics -- REQ-N-005 vacuous-pass guard)' % doc.get('verdict')
metrics = doc.get('metrics') or []
assert len(metrics) == expected_count, 'metrics has %d entries, want %d -- every band-published metric must be accounted for (structural invariant)' % (len(metrics), expected_count)
assert all(m.get('verdict') == 'not_comparable' for m in metrics), 'metrics=%r, want every entry not_comparable (single-rep band -- every metric is insufficient_reps)' % metrics
reasons = doc.get('reasons') or []
assert any(r.get('reason') == 'no_comparable_metrics' for r in reasons), 'reasons=%r, want a no_comparable_metrics entry' % reasons
" || fail "m: verification document check failed: $(cat "$out")"

	echo "TC-019m PASS"
}

# ---------------------------------------------------------------------------
# TC-019l (UAT R2-F-6, HIGH): round-1's F-6 fix compared only
# variant_bundle_sha256/model_ids (POST_DISPATCH_DRIFT_FIELDS) plus a
# special-cased run_key. fixture_base_sha, corpus_schema_version, and
# p2p_set were displayed by identity()'s IDENTITY_FIELDS but never
# enforced; shark_version was not even displayed. Each sub-case flips
# exactly ONE of these four fields on the freshly replayed record (rep
# stays 1, so run_key still matches -- isolates the field under test) and
# asserts verdict invalid with a named identity_mismatch reason naming the
# field and both values, and that the field is present in BOTH stored/
# replayed display blocks with the correct values (display must never
# outrun comparison -- identity()'s display set is now derived from the
# same IDENTITY_FIELDS the comparison loop iterates).
# ---------------------------------------------------------------------------
test_l() {
	local rec="$WORKDIR/l-record.jsonl"
	place_record "$WORKDIR/l-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/l-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	local field new_value
	for field in fixture_base_sha corpus_schema_version p2p_set shark_version; do
		case "$field" in
		fixture_base_sha) new_value='"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"' ;;
		corpus_schema_version) new_value='"99.0"' ;;
		p2p_set) new_value='"bogus-p2p-set"' ;;
		shark_version) new_value='"shark version bogus (0000000) built 1970-01-01"' ;;
		esac

		local fresh="$WORKDIR/l-fresh-$field.jsonl"
		python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
			--set "manifest.$field=$new_value" >"$fresh"

		local out err code
		{
			read -r out
			read -r err
			read -r code
		} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$fresh" \
			run_replay "$rec" "$BAND_VALID" "$WORKDIR/l-out-$field" "$CORPUS_YAML")
		[[ "$code" -ne 0 ]] || fail "l($field): exited 0, want non-zero ($field differs but was previously only displayed, never compared -- UAT R2-F-6)"
		[[ -s "$out" ]] || fail "l($field): stdout empty, want a verification document: $(cat "$err")"

		FIELD="$field" python3 -c "
import json
import os

field = os.environ['FIELD']

with open('$out') as f:
    doc = json.load(f)
with open('$rec') as f:
    stored = json.loads(f.readline())
with open('$fresh') as f:
    replayed = json.loads(f.readline())

assert doc.get('verdict') == 'invalid', 'verdict=%r, want invalid (%s differs but every other identity field matches)' % (doc.get('verdict'), field)
reasons = doc.get('reasons') or []
mismatch = [r for r in reasons if r.get('reason') == 'identity_mismatch' and r.get('field') == field]
assert len(mismatch) == 1, 'reasons=%r, want exactly one identity_mismatch entry for %s' % (reasons, field)
entry = mismatch[0]
assert entry['expected'] == stored['manifest'][field], 'expected=%r, want the stored %s %r' % (entry['expected'], field, stored['manifest'][field])
assert entry['actual'] == replayed['manifest'][field], 'actual=%r, want the replayed %s %r' % (entry['actual'], field, replayed['manifest'][field])
assert entry['expected'] != entry['actual'], 'expected and actual are identical, want distinct %s values named' % field
assert doc['stored'].get(field) == stored['manifest'][field], 'stored display block missing/wrong %s -- display must never outrun comparison' % field
assert doc['replayed'].get(field) == replayed['manifest'][field], 'replayed display block missing/wrong %s -- display must never outrun comparison' % field
other_reasons = [r for r in reasons if r.get('reason') != 'identity_mismatch']
assert not other_reasons, 'reasons=%r, want ONLY the identity_mismatch entry for %s (no spurious other-field drift)' % (reasons, field)
" || fail "l($field): verification document check failed: $(cat "$out")"

		echo "TC-019l($field) PASS"
	done

	echo "TC-019l PASS"
}

# ---------------------------------------------------------------------------
# TC-019n (R2-F-13, code-review-20260808-235300-E40-F03.md /
# uat-20260808-231500-E40-F03.md): manifest.item_id/manifest.variant_id
# come from the STORED RECORD, never the operator, and flow into
# fresh_record_path (out_root_abs/item_id/variant_id/rep-N/...) unless
# validated. A traversing value in either field must be rejected as a
# precondition failure -- before any mkdir/cp/dispatch -- never silently
# interpolated into a path that escapes --out.
# ---------------------------------------------------------------------------
test_n() {
	# --- (a) traversing manifest.item_id ---
	local rec_item="$WORKDIR/n-item-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "../../n-item-victim" --rep 1 >"$rec_item"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_replay "$rec_item" "$BAND_VALID" "$WORKDIR/n-item-out" "$CORPUS_YAML")
	[[ "$code" -ne 0 ]] || fail "n(item_id): exited 0 with a traversing manifest.item_id, want non-zero: $(cat "$out")"
	grep -qi "safe path identifier" "$err" || fail "n(item_id): stderr does not name the unsafe manifest.item_id: $(cat "$err")"
	[[ ! -d "$WORKDIR/n-item-out" ]] || fail "n(item_id): --out was created despite the traversing item_id being rejected before any filesystem operation"

	echo "TC-019n(item_id) PASS"

	# --- (b) traversing manifest.variant_id (not RESERVED_PATHS-guarded
	# in gen_fixtures.py, so settable directly via --set) ---
	local rec_variant="$WORKDIR/n-variant-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 \
		--set 'manifest.variant_id="../../n-variant-victim"' >"$rec_variant"

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_replay "$rec_variant" "$BAND_VALID" "$WORKDIR/n-variant-out" "$CORPUS_YAML")
	[[ "$code2" -ne 0 ]] || fail "n(variant_id): exited 0 with a traversing manifest.variant_id, want non-zero: $(cat "$out2")"
	grep -qi "safe path identifier" "$err2" || fail "n(variant_id): stderr does not name the unsafe manifest.variant_id: $(cat "$err2")"
	[[ ! -d "$WORKDIR/n-variant-out" ]] || fail "n(variant_id): --out was created despite the traversing variant_id being rejected before any filesystem operation"

	echo "TC-019n(variant_id) PASS"
}

# ---------------------------------------------------------------------------
# TC-019o (NEW-1, HIGH, uat-20260809-013000-E40-F03.md): a SYMLINKED --out
# root must dispatch normally (G7 replay, the Phase 1 exit-criterion
# command, must not be unconditionally broken for any --out that resolves
# through a symlink). Round-3 UAT repro: replay-manifest.sh's own
# out_root_abs was `cd "$out_root" && pwd` (LOGICAL, symlinks preserved)
# while the fresh-record containment check canonicalized fully via
# `realpath -m` -- a mismatch that wrongly rejected a legitimate, fully-
# inside path. Fixed by canonicalizing out_root_abs itself and sharing
# run-batch.sh's own assert_within_out_root() via lib/path-safety.sh.
# ---------------------------------------------------------------------------
test_o() {
	local rec="$WORKDIR/o-record.jsonl"
	place_record "$WORKDIR/o-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/o-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	# Fresh record identical to the stored one (same pattern as f(a)) --
	# every metric lands inside its interval, so a correct fix exits 0.
	local fresh="$WORKDIR/o-fresh-record.jsonl"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$REAL_ITEM_ID" --rep 1 >"$fresh"

	local real_dir="$WORKDIR/o-real-out"
	local out_dir="$WORKDIR/o-symlink-out"
	mkdir -p "$real_dir"
	ln -s "$real_dir" "$out_dir"

	local log="$WORKDIR/o-run-one.log"
	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_LOG="$log" STUB_RUN_ONE_RECORD_FILE="$fresh" \
		run_replay "$rec" "$BAND_VALID" "$out_dir" "$CORPUS_YAML")
	[[ "$code" -eq 0 ]] || fail "o: exited $code with a symlinked --out root, want 0 (legitimate, fully-inside path, not 'outside --out'): $(cat "$err")"
	grep -qi "outside --out" "$err" && fail "o: stderr wrongly claims the symlinked --out resolves outside --out: $(cat "$err")"

	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "o: stub invocation count=$calls, want exactly 1"

	echo "TC-019o PASS"
}

# ---------------------------------------------------------------------------
# TC-019p (NEW-6, MEDIUM, uat-20260809-013000-E40-F03.md, round 3): the
# per-metric-comparison structural invariant ("metrics_out must account
# for every band-published metric") used to be a bare `assert`, which
# CPython strips from compiled bytecode entirely under PYTHONOPTIMIZE=1
# (or `-O`/`-OO`) -- silently turning the invariant into a no-op in that
# mode. It is now an explicit conditional writing to stderr and calling
# sys.exit(1) (replay-manifest.sh's own non-elidable hard-failure
# mechanism, the same one every load_one_line_json/precondition/--band
# failure already uses). Sub-cases (a)/(b) invoke the invariant check
# DIRECTLY with a deliberately mismatched metrics_out/band_metrics pair
# (simulating a future regression that silently drops a metric, since the
# real per-metric loop's if/else always appends and can never trip this in
# practice) -- proving the check itself, not merely its current callers,
# enforces the invariant in both the normal interpreter and under
# PYTHONOPTIMIZE=1. Sub-case (c) is the fix guidance's own literal ask: a
# real fixture run must produce byte-identical stdout and the same exit
# status whether or not PYTHONOPTIMIZE=1 is set.
# ---------------------------------------------------------------------------
extract_invariant_check() {
	# The invariant's own if/write/exit block, located by pattern (not a
	# hardcoded line range) so this stays correct as the file grows, and
	# dedented so it can be exec()'d as top-level code (the real file
	# nests it one level inside the outer `else:` block).
	awk '
		/^    if len\(metrics_out\) != len\(band_metrics\):/ { capture = 1 }
		capture { print }
		capture && /sys\.exit\(1\)/ { capture = 0 }
	' "$REPLAY" | sed 's/^    //' >"$1"
}

test_p() {
	local module_py="$WORKDIR/p-invariant-check.py"
	extract_invariant_check "$module_py"
	[[ -s "$module_py" ]] || fail "p: failed to extract the structural-invariant check from $REPLAY"
	grep -q "sys.exit(1)" "$module_py" || fail "p: extracted snippet does not contain sys.exit(1) -- extraction markers drifted from $REPLAY"

	local probe='
import sys

# Deliberately mismatched lengths -- simulates a future regression where
# the per-metric loop silently drops one band-published metric, which the
# real if/else branches can never do today (every branch appends).
metrics_out = [1, 2]
band_metrics = {"a": 1, "b": 1, "c": 1}

with open(sys.argv[1]) as f:
    snippet = f.read()

try:
    exec(compile(snippet, sys.argv[1], "exec"))
except SystemExit as e:
    print("FAIL_RAISED code=%r" % (e.code,))
    sys.exit(0 if e.code == 1 else 1)
else:
    print("NO_FAIL_RAISED -- invariant did not fire")
    sys.exit(1)
'

	local normal_out normal_code
	set +e
	normal_out="$(python3 -c "$probe" "$module_py" 2>&1)"
	normal_code=$?
	set -e
	[[ "$normal_code" -eq 0 ]] || fail "p(a): invariant check did not call sys.exit(1) on a mismatched metrics_out/band_metrics pair -- the invariant is not enforced by the check itself: $normal_out"
	[[ "$normal_out" == *FAIL_RAISED* ]] || fail "p(a): unexpected output: $normal_out"
	echo "TC-019p(a, invariant check fires on a direct mismatched-length call) PASS"

	local optimize_out optimize_code
	set +e
	optimize_out="$(PYTHONOPTIMIZE=1 python3 -c "$probe" "$module_py" 2>&1)"
	optimize_code=$?
	set -e
	[[ "$optimize_code" -eq 0 ]] || fail "p(b): under PYTHONOPTIMIZE=1, the invariant check did not call sys.exit(1) on a mismatched pair -- a bare 'assert' would be silently elided here: $optimize_out"
	[[ "$optimize_out" == *FAIL_RAISED* ]] || fail "p(b): unexpected output: $optimize_out"
	[[ "$normal_code" -eq "$optimize_code" ]] || fail "p(b): normal-mode and PYTHONOPTIMIZE=1 exit codes diverge: $normal_code vs $optimize_code"
	echo "TC-019p(b, invariant check is NOT elided under PYTHONOPTIMIZE=1 -- same failure mode as normal mode) PASS"

	# --- (c) full-pipeline "identical output/exit status" check over a
	# real, valid replay -- the fix guidance's own literal ask: PYTHONOPTIMIZE=1
	# must never change replay-manifest.sh behavior for any real
	# invocation, not just the direct invariant-check probe above. ---
	local rec="$WORKDIR/p-record.jsonl"
	place_record "$WORKDIR/p-root" "$REAL_ITEM_ID" 1
	cp "$WORKDIR/p-root/$REAL_ITEM_ID/default/rep-1/record.jsonl" "$rec"

	# Same --out path for both runs (removed and recreated between them) so
	# the document's own "replayed.path" field -- which embeds --out --
	# does not produce a spurious diff unrelated to PYTHONOPTIMIZE.
	local out_dir="$WORKDIR/p-out"

	local out_normal err_normal code_normal
	{
		read -r out_normal
		read -r err_normal
		read -r code_normal
	} < <(RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$rec" \
		run_replay "$rec" "$BAND_VALID" "$out_dir" "$CORPUS_YAML")
	[[ "$code_normal" -eq 0 ]] || fail "p(c): normal-mode run exited $code_normal, want 0: $(cat "$err_normal")"

	rm -rf "$out_dir"

	local out_pyopt err_pyopt code_pyopt
	out_pyopt="$(mktemp -p "$WORKDIR")"
	err_pyopt="$(mktemp -p "$WORKDIR")"
	set +e
	PYTHONOPTIMIZE=1 RUN_ONE_BIN="$STUB_RUN_ONE" STUB_RUN_ONE_RECORD_FILE="$rec" \
		"$REPLAY" --record "$rec" --band "$BAND_VALID" --out "$out_dir" --corpus "$CORPUS_YAML" \
		>"$out_pyopt" 2>"$err_pyopt"
	code_pyopt=$?
	set -e
	[[ "$code_pyopt" -eq 0 ]] || fail "p(c): PYTHONOPTIMIZE=1 run exited $code_pyopt, want 0: $(cat "$err_pyopt")"
	[[ "$code_normal" -eq "$code_pyopt" ]] || fail "p(c): exit status diverges -- normal=$code_normal PYTHONOPTIMIZE=1=$code_pyopt"
	diff -u "$out_normal" "$out_pyopt" >/dev/null || fail "p(c): PYTHONOPTIMIZE=1 output differs from normal-mode output: $(diff -u "$out_normal" "$out_pyopt")"
	echo "TC-019p(c, full-pipeline output/exit status identical under PYTHONOPTIMIZE=1) PASS"

	echo "TC-019p PASS"
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
test_m
test_l
test_n
test_o
test_p

echo "TC-019 PASS"
