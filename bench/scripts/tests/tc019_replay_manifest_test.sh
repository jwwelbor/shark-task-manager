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
bad = [m for m in metrics if m.get('verdict') != 'pass']
assert not bad, 'metrics with a non-pass verdict in the all-pass case: %r' % bad
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
bad_others = [m for m in others if m.get('verdict') != 'pass']
assert not bad_others, 'other metrics not individually pass despite only loc_prod_added moving: %r' % bad_others
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

	# replay-manifest.sh derives BENCH_DIR from ITS OWN location
	# ($SCRIPT_DIR/..) for the ledger-directory precondition (spec.md:
	# resolved from the script's own location, never --corpus's directory)
	# -- so the fake bench tree needs its own corpus/ledgers/<sha>/ sibling
	# to the copied script, or precondition (i) fails before this test ever
	# reaches RUN_ONE_BIN resolution.
	local fake_sha="2222222222222222222222222222222222222222"
	mkdir -p "$sibling_dir/../corpus/ledgers/$fake_sha"

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
	[[ "$code" -eq 0 ]] || fail "h: replay-manifest.sh (RUN_ONE_BIN unset) exited $code, want 0: $(cat "$err")"

	[[ ! -f "$decoy_marker" ]] || fail "h: the PATH-resolved decoy run-one.sh was invoked -- RUN_ONE_BIN defaulted to a bare, PATH-resolved name instead of the sibling path"

	[[ -f "$log" ]] || fail "h: sibling-path run-one.sh stand-in was never invoked -- RUN_ONE_BIN did not default to \$SCRIPT_DIR/run-one.sh: $(cat "$err")"
	local calls
	calls="$(invocation_count "$log")"
	[[ "$calls" == "1" ]] || fail "h: sibling-path run-one.sh stand-in invocation count=$calls, want exactly 1"

	echo "TC-019h PASS"
}

test_a
test_b
test_c
test_d
test_e
test_f
test_g
test_h

echo "TC-019 PASS"
