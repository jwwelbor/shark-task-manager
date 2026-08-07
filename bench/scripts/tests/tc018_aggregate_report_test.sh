#!/usr/bin/env bash
# TC-018 (test-plan.md AC test matrix; T-E40-F03-003 task spec Test Cases),
# sub-cases a, b, c, d, e, f, q, s, t.
#
# T-E40-F03-003's slice: `aggregate-runs.sh`'s core only -- pinned-glob
# enumeration, per-record structural validation (AC-07), classification
# (AC-08/AC-09/AC-10), family presence read from the family block itself
# and never from `sources` (TC-018q, TD-076's consumer-side consequence),
# five-field provenance uniformity (AC-11/TC-018f), the pinned-glob-vs-find
# quarantine exclusion (TC-018s), and `batch-log.jsonl` non-read (TC-018t).
#
# T-E40-F03-004's extension adds sub-cases g, h, i, j, k, l, m, r -- the
# metric registry, Class A/B/C statistics, ADR-F03-04's acceptance
# intervals, per-step keying, the regression-signal lock, `rejections.
# by_gate` zero-vs-excluded, the three flags, and `input_digest` -- and
# extends test_d (AC-09) and test_f (AC-11) in place, closing the IOUs
# those two left explicitly for this task (see their own comments below).
# `report-baseline.sh` didn't exist at that point, so TC-018i's own
# report-rendering half (REQ-F-023: the heading names "E40-F01" verbatim)
# was deferred here, to this task.
#
# T-E40-F03-005 (this extension) adds sub-cases n, o, p, u -- determinism/
# no-timestamp (TC-018n, AC-19), per-metric band+interval+derivation-rule
# content and the four REQ-F-024 caveats (TC-018o, AC-20), the provenance
# block plus the aggregate's own `baseline_id` format (TC-018p, AC-21), and
# `report-baseline.sh`'s own malformed/unsupported-schema_version exit
# behavior (TC-018u) -- all against `bench/scripts/report-baseline.sh`,
# which this task creates. It also closes test_i's own deferred half in
# place: TC-018i's report-rendering assertion (the corpus-feedback heading
# naming "E40-F01") now runs alongside its aggregate-side flag check.
# Mirrors tc015's own T-E40-F02-001/-002/-007 extension precedent (see that
# file's own header comment).
#
# Caller-Path Contract (test-plan.md TC-018): real subprocess invocation of
# `bench/scripts/aggregate-runs.sh --root <dir>` against fixture roots
# built entirely from `bench/scripts/testdata/aggregate/gen_fixtures.py`
# (T-E40-F03-002) -- never a hand-authored record (REQ-N-006, ADR-F03-08).
# Nothing in the aggregator is stubbed; this is the real I-02 consumer path
# (I-02: consumes, contract test tests/contracts/e40_i02_artifact_contract_
# test.go#TC-001, referenced not twinned, ADR-F03-09). Sub-cases n/o/p/u
# additionally invoke `bench/scripts/report-baseline.sh --aggregate <path>`
# as a real subprocess against the real `aggregate.json` the aggregator
# just produced -- never a hand-authored aggregate document.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

AGGREGATE="$SCRIPTS_DIR/aggregate-runs.sh"
REPORT="$SCRIPTS_DIR/report-baseline.sh"
GEN_FIXTURES="$SCRIPTS_DIR/testdata/aggregate/gen_fixtures.py"
README_PATH="$BENCH_DIR/README.md"

fail() {
	echo "TC-018 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATE" ]] || fail "aggregate-runs.sh missing or not executable: $AGGREGATE"
[[ -x "$REPORT" ]] || fail "report-baseline.sh missing or not executable: $REPORT"
[[ -f "$GEN_FIXTURES" ]] || fail "gen_fixtures.py missing: $GEN_FIXTURES"
[[ -f "$README_PATH" ]] || fail "bench/README.md missing: $README_PATH"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# place_record <root> <item_id> <rep> [gen_fixtures.py args...]
# Writes one golden-derived fixture at the canonical
# <root>/<item_id>/default/rep-<rep>/record.jsonl path (variant is always
# "default" -- the golden's own manifest.variant_id, which gen_fixtures.py
# never rewrites; see its rewrite/preserve contract). Defaults to
# `--golden completed`; a caller wanting the timeout golden passes its own
# `--golden timeout` in the extra args, which wins (argparse: last
# occurrence of a --golden flag takes effect).
place_record() {
	local root="$1" item_id="$2" rep="$3"
	shift 3
	local dir="$root/$item_id/default/rep-$rep"
	mkdir -p "$dir"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$item_id" --rep "$rep" "$@" >"$dir/record.jsonl"
}

# run_aggregate <root> [aggregate-runs.sh args...]
# Invokes aggregate-runs.sh, capturing stdout/stderr/exit code into
# WORKDIR-scoped files whose paths are printed on the last three lines (so
# callers can `read` them). stdout is the ONLY place the script ever
# "writes" the aggregate (ADR-F03-01, pure function to stdout) -- a caller
# wanting a persisted aggregate.json would redirect it there itself; this
# harness's own "did it write anything" checks (AC-07's "no aggregate.json"
# expectation) are therefore stdout emptiness checks, not file-existence
# checks against some third path.
run_aggregate() {
	local root="$1"
	shift
	local out err code
	out="$(mktemp -p "$WORKDIR")"
	err="$(mktemp -p "$WORKDIR")"
	set +e
	"$AGGREGATE" --root "$root" "$@" >"$out" 2>"$err"
	code=$?
	set -e
	echo "$out"
	echo "$err"
	echo "$code"
}

# run_report <aggregate.json path>
# Invokes report-baseline.sh the same way run_aggregate invokes
# aggregate-runs.sh -- stdout/stderr/exit code into WORKDIR-scoped files,
# their paths on the last three lines.
run_report() {
	local aggregate_path="$1"
	local out err code
	out="$(mktemp -p "$WORKDIR")"
	err="$(mktemp -p "$WORKDIR")"
	set +e
	"$REPORT" --aggregate "$aggregate_path" >"$out" 2>"$err"
	code=$?
	set -e
	echo "$out"
	echo "$err"
	echo "$code"
}

# build_ac12_aggregate <root_dir> <aggregate_out_path> <item_id>
# Builds the same 3-branch Class C fixture TC-018g (AC-12) uses -- (i)
# loc_prod_added varies (r>0), (ii) every other Class C metric held
# constant across reps (r==0, median!=0 -- wall_clock_ns is the one this
# file's report sub-cases key on), (iii) loc_test_deleted forced to 0 on
# every rep (identically-zero) -- then aggregates it into
# <aggregate_out_path>. AC-19/AC-20/AC-21 (TC-018n/o/p) all key their
# fixture off "the AC-12 fixture"; this is that fixture, not a fresh one,
# so the report's derivation-rule text is checked against the same
# three-branch shape TC-018g already proved the aggregator computes
# correctly.
build_ac12_aggregate() {
	local root="$1" aggregate_out="$2" item="$3"
	local rep prod_added
	for rep in 1 2 3; do
		case "$rep" in
		1) prod_added=10 ;;
		2) prod_added=12 ;;
		3) prod_added=15 ;;
		esac
		place_record "$root" "$item" "$rep" --set "loc.prod_added=$prod_added" --set loc.test_deleted=0
	done

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "build_ac12_aggregate: aggregate-runs.sh exited $code, want 0: $(cat "$err")"
	cp "$out" "$aggregate_out"
}

# ---------------------------------------------------------------------------
# TC-018a (AC-06, REQ-F-007/REQ-N-004): determinism -- two consecutive
# invocations over a fixed fixture root are byte-identical under LC_ALL=C,
# and (when the locale is installed) under LC_ALL=en_US.UTF-8 too, and the
# two locales' outputs match each other. A container with only C installed
# skips that half cleanly (logged), never a silent pass.
# ---------------------------------------------------------------------------
test_a() {
	local root="$WORKDIR/a-root"
	place_record "$root" f03-fixture-tc018a-det 1
	place_record "$root" f03-fixture-tc018a-det 2
	place_record "$root" f03-fixture-tc018a-det 3
	# Locale-collation-discriminating pair (AC-06's own negative case: "an
	# implementation relying on the shell's default glob/sort collation...
	# would diverge between the two locale pairs when both run"). Under
	# LC_ALL=C, uppercase sorts before lowercase (Beta < alpha); under a
	# real en_US.UTF-8 collation it's the reverse -- a plain glob-order or
	# `strcoll`-based sort would therefore reorder these two between the
	# locale pairs, while an explicit codepoint sort (this script's own
	# `sorted()`/`sort_keys=True`) would not.
	place_record "$root" f03-fixture-tc018a-Beta 1
	place_record "$root" f03-fixture-tc018a-alpha 1

	local out1 err1 code1 out2 err2 code2
	{
		read -r out1
		read -r err1
		read -r code1
	} < <(LC_ALL=C run_aggregate "$root")
	[[ "$code1" -eq 0 ]] || fail "a: LC_ALL=C run 1 exited $code1, want 0: $(cat "$err1")"

	{
		read -r out2
		read -r err2
		read -r code2
	} < <(LC_ALL=C run_aggregate "$root")
	[[ "$code2" -eq 0 ]] || fail "a: LC_ALL=C run 2 exited $code2, want 0: $(cat "$err2")"

	diff "$out1" "$out2" >/dev/null || fail "a: LC_ALL=C two invocations are not byte-identical"
	echo "TC-018a(LC_ALL=C pair byte-identical) PASS"

	if locale -a 2>/dev/null | grep -Eiq '^en_us\.utf-?8$'; then
		# Self-validating precheck: if the two fixture names named above
		# don't actually collate differently between the two locales in
		# THIS environment, the byte-identity assertion below could pass
		# vacuously (nothing would have diverged even under a buggy,
		# collation-dependent implementation) -- fail loud instead of
		# silently proving nothing.
		local c_order u_order
		c_order="$(printf '%s\n' "f03-fixture-tc018a-Beta" "f03-fixture-tc018a-alpha" | LC_ALL=C sort)"
		u_order="$(printf '%s\n' "f03-fixture-tc018a-Beta" "f03-fixture-tc018a-alpha" | LC_ALL=en_US.UTF-8 sort)"
		[[ "$c_order" != "$u_order" ]] ||
			fail "a: fixture names are not locale-discriminating in this environment -- the locale comparison below would pass vacuously"

		local out3 err3 code3 out4 err4 code4
		{
			read -r out3
			read -r err3
			read -r code3
		} < <(LC_ALL=en_US.UTF-8 run_aggregate "$root")
		[[ "$code3" -eq 0 ]] || fail "a: LC_ALL=en_US.UTF-8 run 1 exited $code3, want 0: $(cat "$err3")"

		{
			read -r out4
			read -r err4
			read -r code4
		} < <(LC_ALL=en_US.UTF-8 run_aggregate "$root")
		[[ "$code4" -eq 0 ]] || fail "a: LC_ALL=en_US.UTF-8 run 2 exited $code4, want 0: $(cat "$err4")"

		diff "$out3" "$out4" >/dev/null || fail "a: LC_ALL=en_US.UTF-8 two invocations are not byte-identical"
		diff "$out1" "$out3" >/dev/null || fail "a: LC_ALL=C output differs from LC_ALL=en_US.UTF-8 output -- output depends on locale collation"
		echo "TC-018a(LC_ALL=en_US.UTF-8 pair byte-identical, matches C pair) PASS"
	else
		echo "TC-018a(en_US.UTF-8 not installed -- locale half skipped, logged not silently passed) SKIP" >&2
	fi

	echo "TC-018a PASS"
}

# ---------------------------------------------------------------------------
# TC-018b (AC-07, REQ-F-010/REQ-N-005): an unsupported schema_version makes
# the aggregator exit non-zero, name the file on stderr, and print NOTHING
# to stdout (this harness's "no aggregate.json" check -- see run_aggregate's
# comment).
# ---------------------------------------------------------------------------
test_b() {
	local root="$WORKDIR/b-root"
	place_record "$root" f03-fixture-tc018b 1 --schema-version 99.0

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")

	[[ "$code" -ne 0 ]] || fail "b: exited 0, want non-zero (unsupported schema_version)"
	[[ ! -s "$out" ]] || fail "b: stdout is non-empty on failure, want nothing written: $(cat "$out")"
	grep -q "record.jsonl" "$err" || fail "b: stderr does not name the offending record.jsonl file: $(cat "$err")"
	grep -q "99.0" "$err" || fail "b: stderr does not name the unsupported schema_version: $(cat "$err")"

	echo "TC-018b PASS"
}

# ---------------------------------------------------------------------------
# TC-018c (AC-08, REQ-F-008/REQ-F-009/REQ-N-005): the F-4 anomaly shape --
# outcome=completed, oracle/quality/loc (and their sources.* entries) all
# absent, errors[] empty -- classifies `anomaly`, names the run_key and all
# three missing families in anomalies[], and exits non-zero. Negative: a
# partial-absence fixture (only quality/loc unset, oracle left populated)
# still classifies `anomaly`, not silently downgraded to explained_absence.
# ---------------------------------------------------------------------------
test_c() {
	local root="$WORKDIR/c-root"
	place_record "$root" f03-fixture-tc018c-full 1 \
		--unset oracle --unset quality --unset loc \
		--unset sources.oracle --unset sources.quality --unset sources.loc

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "c: exited 0, want non-zero (anomaly present)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018c-full::default::rep1"
assert d["inventory"][run_key]["classification"] == "anomaly", d["inventory"][run_key]
matches = [a for a in d["anomalies"] if a["run_key"] == run_key]
assert len(matches) == 1, matches
assert matches[0]["missing_families"] == ["loc", "oracle", "quality"], matches[0]
' "$out" || fail "c: full-anomaly assertion failed: $(cat "$out")"
	echo "TC-018c PASS"

	# --- Negative: partial absence (oracle still present) is still anomaly. ---
	local root2="$WORKDIR/c-root2"
	place_record "$root2" f03-fixture-tc018c-partial 1 \
		--unset quality --unset loc \
		--unset sources.quality --unset sources.loc

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root2")
	[[ "$code2" -ne 0 ]] || fail "c(negative): exited 0, want non-zero"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018c-partial::default::rep1"
assert d["inventory"][run_key]["classification"] == "anomaly", d["inventory"][run_key]
assert "oracle" in d["inventory"][run_key]["families_present"], d["inventory"][run_key]
matches = [a for a in d["anomalies"] if a["run_key"] == run_key]
assert len(matches) == 1
assert matches[0]["missing_families"] == ["loc", "quality"], matches[0]
' "$out2" || fail "c(negative): partial-anomaly assertion failed: $(cat "$out2")"

	echo "TC-018c(negative, partial absence still anomaly) PASS"
}

# ---------------------------------------------------------------------------
# TC-018d (AC-09): a timeout record mixed with two completed reps of the
# same item contributes to `outcomes` counts and `timeout_rate` only,
# classifies explained_absence (never anomaly), and the batch exits zero
# (uniform provenance, no anomaly). T-E40-F03-004 closes the IOU this
# test's own comment left for the per-metric registry: the timeout rep
# appears in excluded[] with reason outcome_timeout for EVERY registry
# metric applicable to a `task` item -- including oracle_*/quality_*
# families the timeout record structurally never carried -- no band value
# equals the fixture's timeout cap, and wall_clock_ns's n is one lower
# than the rep count.
# ---------------------------------------------------------------------------
test_d() {
	local root="$WORKDIR/d-root"
	local item="f03-fixture-tc018d"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2
	place_record "$root" "$item" 3 --golden timeout

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "d: exited $code, want 0 (timeout is explained_absence, provenance uniform, no anomaly): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
rep1, rep2, rep3 = ("%s::default::rep%d" % (item, n) for n in (1, 2, 3))

assert d["inventory"][rep1]["classification"] == "complete", d["inventory"][rep1]
assert d["inventory"][rep2]["classification"] == "complete", d["inventory"][rep2]
assert d["inventory"][rep3]["classification"] == "explained_absence", d["inventory"][rep3]
assert d["inventory"][rep3]["outcome"] == "timeout", d["inventory"][rep3]

assert not any(a["run_key"] == rep3 for a in d["anomalies"]), "timeout record must never appear in anomalies[]: %r" % d["anomalies"]
assert d["outcomes"]["anomaly_count"] == 0, d["outcomes"]

assert d["outcomes"]["counts"].get("completed") == 2, d["outcomes"]["counts"]
assert d["outcomes"]["counts"].get("timeout") == 1, d["outcomes"]["counts"]
assert abs(d["outcomes"]["timeout_rate"] - (1.0 / 3.0)) < 1e-9, d["outcomes"]["timeout_rate"]

assert d["provenance"]["uniform"] is True, d["provenance"]

task = next(t for t in d["tasks"] if t["item_id"] == item)
assert task["item_type"] == "task", task

TIMEOUT_CAP_NS = 60 * 1_000_000_000  # the timeout golden own timeout_cap_s=60

for metric_id, block in task["metrics"].items():
    excluded_run_keys = {e["run_key"] for e in block["excluded"]}
    assert rep3 in excluded_run_keys, "metric %s excluded[] missing the timeout rep: %r" % (metric_id, block["excluded"])
    reason = next(e["reason"] for e in block["excluded"] if e["run_key"] == rep3)
    assert reason == "outcome_timeout", "metric %s: timeout rep excluded for %r, want outcome_timeout" % (metric_id, reason)
    for key in ("min", "max", "median", "mean"):
        assert block.get(key) != TIMEOUT_CAP_NS, "metric %s: %s equals the timeout cap %d -- the cap leaked into a band" % (metric_id, key, TIMEOUT_CAP_NS)

# Task item_type is "task" -- oracle_repro_confirmed (bug-only) must never
# be in this registry slice at all, not even excluded[].
assert "oracle_repro_confirmed" not in task["metrics"], task["metrics"].keys()

wall = task["metrics"]["wall_clock_ns"]
assert wall["n"] == 2, "wall_clock_ns n=%r, want 2 (3 reps minus the one timeout)" % wall["n"]
' "$out" "$item" || fail "d: assertion failed: $(cat "$out")"

	echo "TC-018d PASS"
}

# ---------------------------------------------------------------------------
# TC-018e (AC-10, REQ-F-009): a toolchain_guard-failed record -- quality
# present but toolchain_guard != "pass", oracle/loc absent, errors[] names
# postrun_check_aborted -- classifies explained_absence, and the aggregator
# exits zero when no anomaly exists elsewhere in the batch. Negative: the
# same fixture alongside a genuine anomaly record still exits non-zero
# overall (exit status is a whole-aggregation property, not per-record).
# ---------------------------------------------------------------------------
test_e() {
	local root="$WORKDIR/e-root"
	place_record "$root" f03-fixture-tc018e 1 \
		--set 'quality.toolchain_guard=go_version_mismatch' \
		--unset quality.fmt_clean --unset quality.vet_ok --unset quality.tests_pass \
		--unset quality.lint_new_issues --unset quality.lint_new_issues_count \
		--unset oracle --unset loc \
		--unset sources.oracle --unset sources.loc \
		--set 'errors=[{"kind":"postrun_check_aborted","detail":"go version mismatch (toolchain guard abort)"}]'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "e: exited $code, want 0 (no anomaly in this fixture set): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018e::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "explained_absence", inv
assert "quality" in inv["families_present"], inv
assert "oracle" not in inv["families_present"], inv
assert "loc" not in inv["families_present"], inv
assert d["outcomes"]["anomaly_count"] == 0, d["outcomes"]
' "$out" || fail "e: assertion failed: $(cat "$out")"
	echo "TC-018e PASS"

	# --- Negative: same record alongside a genuine anomaly -- overall exit
	# non-zero, but the toolchain_guard record is still explained_absence,
	# not flipped by the sibling anomaly. ---
	local root2="$WORKDIR/e-root2"
	place_record "$root2" f03-fixture-tc018e-guard 1 \
		--set 'quality.toolchain_guard=go_version_mismatch' \
		--unset quality.fmt_clean --unset quality.vet_ok --unset quality.tests_pass \
		--unset quality.lint_new_issues --unset quality.lint_new_issues_count \
		--unset oracle --unset loc \
		--unset sources.oracle --unset sources.loc \
		--set 'errors=[{"kind":"postrun_check_aborted","detail":"go version mismatch (toolchain guard abort)"}]'
	place_record "$root2" f03-fixture-tc018e-anomaly 1 \
		--unset oracle --unset quality --unset loc \
		--unset sources.oracle --unset sources.quality --unset sources.loc

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root2")
	[[ "$code2" -ne 0 ]] || fail "e(negative): exited 0, want non-zero (one genuine anomaly present)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
guard_key = "f03-fixture-tc018e-guard::default::rep1"
assert d["inventory"][guard_key]["classification"] == "explained_absence", d["inventory"][guard_key]
' "$out2" || fail "e(negative): guard record was affected by the sibling anomaly: $(cat "$out2")"

	echo "TC-018e(negative, whole-aggregation exit status) PASS"
}

# ---------------------------------------------------------------------------
# TC-018f (AC-11, REQ-F-011): five-field table-driven provenance uniformity.
# Each of the five fields, varied in isolation across a two-record fixture,
# is reported non-uniform, naming both differing values, with every other
# field still reported uniform (proving the check is scoped exactly to the
# varied field). Negative: a sixth pair differing only on manifest.rep (an
# unlisted, expected-to-vary field) is uniform.
# ---------------------------------------------------------------------------
test_f() {
	local field
	for field in model_ids fixture_base_sha variant_bundle_sha256 corpus_schema_version shark_version; do
		local root="$WORKDIR/f-root-$field"
		local item_a="f03-fixture-tc018f-${field}-a" item_b="f03-fixture-tc018f-${field}-b"

		place_record "$root" "$item_a" 1
		case "$field" in
		model_ids)
			place_record "$root" "$item_b" 1 --set 'manifest.model_ids=["claude-opus-5"]'
			;;
		fixture_base_sha)
			place_record "$root" "$item_b" 1 --set 'manifest.fixture_base_sha="0000000000000000000000000000000000000b"'
			;;
		variant_bundle_sha256)
			place_record "$root" "$item_b" 1 --set 'manifest.variant_bundle_sha256="00000000000000000000000000000000000000000000000000000000000b"'
			;;
		corpus_schema_version)
			place_record "$root" "$item_b" 1 --set 'manifest.corpus_schema_version="1.1"'
			;;
		shark_version)
			place_record "$root" "$item_b" 1 --set 'manifest.shark_version="shark version dev (deadbeef) built 2026-08-08"'
			;;
		esac

		local out err code
		{
			read -r out
			read -r err
			read -r code
		} < <(run_aggregate "$root")
		[[ "$code" -ne 0 ]] || fail "f($field): exited 0, want non-zero (non-uniform provenance)"

		python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
field = sys.argv[2]
prov = d["provenance"]
assert prov["uniform"] is False, prov
divs = prov.get("divergences", [])
assert len(divs) == 1, "expected exactly one divergent field, got %r" % divs
assert divs[0]["field"] == field, divs[0]
assert len(divs[0]["values"]) == 2, divs[0]
assert field not in prov, "divergent field %s must not also appear as a single agreed value: %r" % (field, prov)
for other in ("model_ids", "fixture_base_sha", "variant_bundle_sha256", "corpus_schema_version", "shark_version"):
    if other == field:
        continue
    assert other in prov, "non-varied field %s missing from provenance (should be uniform): %r" % (other, prov)

# T-E40-F03-004: non-uniform provenance must never publish the STATISTICAL
# blocks "as if the batch were valid" (AC-11) -- tasks[]/corpus/flags/
# baseline_id are entirely absent, even though this task/metric registry
# now exists. input_digest is NOT one of those -- it identifies the exact
# (possibly-invalid) input set, so it still publishes.
for stat_block_key in ("tasks", "corpus", "flags", "baseline_id"):
    assert stat_block_key not in d, "non-uniform provenance must omit %r entirely: %r" % (stat_block_key, d.get(stat_block_key))
assert "input_digest" in d, "input_digest must still publish even when provenance is non-uniform (it names the invalid input set)"
' "$out" "$field" || fail "f($field): assertion failed: $(cat "$out")"

		echo "TC-018f($field) PASS"
	done

	# --- Negative: differing only on manifest.rep (unlisted field) is uniform. ---
	local root_neg="$WORKDIR/f-root-negative"
	local item_neg="f03-fixture-tc018f-negative"
	place_record "$root_neg" "$item_neg" 1
	place_record "$root_neg" "$item_neg" 2

	local out_neg err_neg code_neg
	{
		read -r out_neg
		read -r err_neg
		read -r code_neg
	} < <(run_aggregate "$root_neg")
	[[ "$code_neg" -eq 0 ]] || fail "f(negative): exited $code_neg, want 0 (rep differing is expected, not a uniformity violation): $(cat "$err_neg")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert d["provenance"]["uniform"] is True, d["provenance"]
assert "divergences" not in d["provenance"], d["provenance"]
# Counterpart to the non-uniform suppression check above: a UNIFORM batch
# must actually publish the statistical blocks, not omit them universally.
for stat_block_key in ("tasks", "corpus", "flags", "baseline_id"):
    assert stat_block_key in d, "uniform provenance must publish %r: %r" % (stat_block_key, d)
' "$out_neg" || fail "f(negative): assertion failed: $(cat "$out_neg")"

	echo "TC-018f(negative, rep differing is not a violation) PASS"
}

# ---------------------------------------------------------------------------
# REQ-F-007 2nd sentence, TC-018q: family presence is read from the family
# block itself, never from `sources`. (a) oracle present, sources.oracle
# absent -- oracle still contributes (not excluded, not anomaly-triggering
# on its own). (b) whole `sources` object absent, every family present --
# not anomaly, no metric excluded on account of `sources` alone.
# ---------------------------------------------------------------------------
test_q() {
	local root_a="$WORKDIR/q-root-a"
	place_record "$root_a" f03-fixture-tc018q-a 1 --unset sources.oracle

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -eq 0 ]] || fail "q(a): exited $code_a, want 0: $(cat "$err_a")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018q-a::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "complete", inv
assert "oracle" in inv["families_present"], inv
' "$out_a" || fail "q(a): assertion failed: $(cat "$out_a")"
	echo "TC-018q(a, oracle present with sources.oracle absent still contributes) PASS"

	local root_b="$WORKDIR/q-root-b"
	place_record "$root_b" f03-fixture-tc018q-b 1 --unset sources

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -eq 0 ]] || fail "q(b): exited $code_b, want 0: $(cat "$err_b")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018q-b::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "complete", inv
assert d["outcomes"]["anomaly_count"] == 0, d["outcomes"]
assert not any(a["run_key"] == run_key for a in d["anomalies"]), d["anomalies"]
' "$out_b" || fail "q(b): assertion failed: $(cat "$out_b")"
	echo "TC-018q(b, whole sources block absent is not itself an anomaly signal) PASS"

	echo "TC-018q PASS"
}

# ---------------------------------------------------------------------------
# TC-018s: record enumeration is the pinned glob, never `find` -- a
# structurally identical, well-formed record.jsonl placed under
# <root>/.incomplete/<item>/default/rep-1-1/record.jsonl contributes to NO
# band, NO inventory entry, and NO count anywhere in the output.
# ---------------------------------------------------------------------------
test_s() {
	local root="$WORKDIR/s-root"
	local item="f03-fixture-tc018s"
	place_record "$root" "$item" 1

	local quarantine_dir="$root/.incomplete/$item/default/rep-1-1"
	mkdir -p "$quarantine_dir"
	cp "$root/$item/default/rep-1/record.jsonl" "$quarantine_dir/record.jsonl"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "s: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
assert len(d["inventory"]) == 1, "inventory has %d entries, want exactly 1 (the quarantined duplicate must never contribute): %r" % (len(d["inventory"]), d["inventory"])
total = sum(d["outcomes"]["counts"].values())
assert total == 1, "outcomes counts sum to %d, want 1: %r" % (total, d["outcomes"])
' "$out" || fail "s: assertion failed: $(cat "$out")"

	echo "TC-018s PASS"
}

# ---------------------------------------------------------------------------
# TC-018t: batch-log.jsonl is never read by the aggregator -- an unreadable
# (chmod 000) batch-log.jsonl produces byte-identical output to the same
# fixture root with batch-log.jsonl absent entirely.
# ---------------------------------------------------------------------------
test_t() {
	local root_with="$WORKDIR/t-root-with" root_without="$WORKDIR/t-root-without"
	place_record "$root_with" f03-fixture-tc018t 1
	place_record "$root_without" f03-fixture-tc018t 1

	echo '{"this": "is operator diagnostics only, never read by the aggregator"}' >"$root_with/batch-log.jsonl"

	if [[ "$(id -u)" -eq 0 ]]; then
		echo "TC-018t(running as root -- chmod 000 does not block root's own reads; skipping unreadable-file half, logged not silently passed) SKIP" >&2
	else
		chmod 000 "$root_with/batch-log.jsonl"
	fi

	local out_with err_with code_with out_without err_without code_without
	{
		read -r out_with
		read -r err_with
		read -r code_with
	} < <(run_aggregate "$root_with")
	{
		read -r out_without
		read -r err_without
		read -r code_without
	} < <(run_aggregate "$root_without")

	[[ "$code_with" -eq 0 ]] || fail "t: with unreadable batch-log.jsonl exited $code_with, want 0: $(cat "$err_with")"
	[[ "$code_without" -eq 0 ]] || fail "t: without batch-log.jsonl exited $code_without, want 0: $(cat "$err_without")"

	diff "$out_with" "$out_without" >/dev/null || fail "t: output differs depending on batch-log.jsonl presence/readability -- the aggregator must never read it (REQ-F-007)"

	[[ "$(id -u)" -eq 0 ]] || chmod 644 "$root_with/batch-log.jsonl"

	echo "TC-018t PASS"
}

# ---------------------------------------------------------------------------
# TC-018g (AC-12, ADR-F03-04): the three Class C acceptance-interval
# branches, all exercised in ONE 3-rep fixture root so "for every Class B/C
# metric" (AC-12's own invariant) is checked across the whole registry in
# a single aggregate, not just the one metric each branch varies:
#   (i)   loc_prod_added: 10, 12, 15            -- r=5>0
#   (ii)  every OTHER unvaried metric (wall_clock_ns among them)
#                                                -- r=0, median != 0
#   (iii) loc_test_deleted forced to 0 on every rep
#                                                -- r=0, median == 0
# Every value not explicitly varied is derived unchanged from the same
# golden on each of the 3 place_record calls, so it is automatically
# constant across reps -- branch (ii) coverage comes for free across
# nearly the entire registry, not just wall_clock_ns.
# ---------------------------------------------------------------------------
test_g() {
	local root="$WORKDIR/g-root"
	local item="f03-fixture-tc018g"
	local rep prod_added
	for rep in 1 2 3; do
		case "$rep" in
		1) prod_added=10 ;;
		2) prod_added=12 ;;
		3) prod_added=15 ;;
		esac
		place_record "$root" "$item" "$rep" --set "loc.prod_added=$prod_added" --set loc.test_deleted=0
	done

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "g: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import statistics
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
metrics = task["metrics"]

# (i) r > 0 branch, computed independently from the fixture own raw values
# (test-plan.md: never a literal copied from a prior run of the
# implementation).
values = [10, 12, 15]
mn, mx, median = min(values), max(values), statistics.median(values)
r_eff = mx - mn
block = metrics["loc_prod_added"]
assert block["n"] == 3, block
assert block["min"] == mn and block["max"] == mx and block["median"] == median, block
assert block["accept_lo"] == mn - r_eff, block
assert block["accept_hi"] == mx + r_eff, block

# (iii) identically-zero branch: accept_lo == accept_hi == 0 EXACTLY, and
# spread_rel is null (median == 0) -- the documented rule, not a formula
# coincidence.
block = metrics["loc_test_deleted"]
assert block["n"] == 3, block
assert block["min"] == 0 and block["max"] == 0 and block["median"] == 0, block
assert block["accept_lo"] == 0 and block["accept_hi"] == 0, block
assert block["spread_rel"] is None, block

# (ii) r == 0, median != 0 branch: wall_clock_ns is untouched by any --set,
# so every rep carries the same nonzero golden value.
block = metrics["wall_clock_ns"]
assert block["n"] == 3, block
assert block["spread_abs"] == 0, block
assert block["median"] != 0, block
r_eff = 0.10 * abs(block["median"])
assert abs(block["accept_lo"] - (block["min"] - r_eff)) < 1e-9, block
assert abs(block["accept_hi"] - (block["max"] + r_eff)) < 1e-9, block

# Class A metric in the SAME fixture set carries accept_set, never a
# numeric interval.
f2p = metrics["oracle_f2p_resolved"]
assert "accept_set" in f2p, f2p
assert "accept_lo" not in f2p and "accept_hi" not in f2p, f2p

# Blanket invariant (AC-12): for every Class B/C metric in this task,
# accept_lo <= min and accept_hi >= max.
no_class_c_or_b_without_interval = 0
for metric_id, block in metrics.items():
    if "accept_lo" not in block:
        continue  # Class A (accept_set) or insufficient_reps -- not this invariant
    assert block["accept_lo"] <= block["min"], "%s: accept_lo %r > min %r" % (metric_id, block["accept_lo"], block["min"])
    assert block["accept_hi"] >= block["max"], "%s: accept_hi %r < max %r" % (metric_id, block["accept_hi"], block["max"])
    no_class_c_or_b_without_interval += 1
assert no_class_c_or_b_without_interval > 5, "too few Class B/C metrics carried an interval to be a meaningful blanket check: %d" % no_class_c_or_b_without_interval

# Lightweight sanity check for the AC bullet this task also owns (the
# fuller TC-018p format assertion lands with T-E40-F03-005, which needs
# report-baseline.sh): baseline_id matches
# <variant_id>-<fixture_base_sha[:12]>-r<reps> exactly.
prov = d["provenance"]
expected_baseline_id = "%s-%s-r%d" % ("default", prov["fixture_base_sha"][:12], prov["reps"])
assert d["baseline_id"] == expected_baseline_id, (d["baseline_id"], expected_baseline_id)

assert isinstance(d.get("corpus"), dict) and d["corpus"], "corpus rollup block missing or empty"
' "$out" "$item" || fail "g: assertion failed: $(cat "$out")"

	echo "TC-018g PASS"
}

# ---------------------------------------------------------------------------
# TC-018h (AC-13, REQ-F-016): exactly one metric family (loc) has only 1
# contributing rep -- reps 1-2 have loc unset with a postrun_check_aborted
# error (explained_absence, missing=["loc"] only, since oracle/quality stay
# present), rep 3 is untouched. loc_* metrics land at n=1, flagged
# insufficient_reps, and publish no accept_lo/accept_hi. Every OTHER
# metric in the same task stays at n=3 and publishes its interval normally
# -- proving the flag is per-metric, not per-task.
# ---------------------------------------------------------------------------
test_h() {
	local root="$WORKDIR/h-root"
	local item="f03-fixture-tc018h"
	local rep
	for rep in 1 2; do
		place_record "$root" "$item" "$rep" \
			--unset loc --unset sources.loc \
			--set 'errors=[{"kind":"postrun_check_aborted","detail":"loc computation aborted (synthetic TC-018h fixture)"}]'
	done
	place_record "$root" "$item" 3

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "h: exited $code, want 0 (explained_absence, no anomaly): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
metrics = task["metrics"]

loc_block = metrics["loc_files_touched"]
assert loc_block["n"] == 1, loc_block
assert loc_block.get("insufficient_reps") is True, loc_block
assert "accept_lo" not in loc_block and "accept_hi" not in loc_block and "accept_set" not in loc_block, loc_block
reasons = {e["run_key"]: e["reason"] for e in loc_block["excluded"]}
assert reasons.get(item + "::default::rep1") == "postrun_aborted", reasons
assert reasons.get(item + "::default::rep2") == "postrun_aborted", reasons

flagged = {(f["item_id"], f["metric"]) for f in d["flags"]["insufficient_reps"]}
assert (item, "loc_files_touched") in flagged, d["flags"]["insufficient_reps"]

# Negative: every OTHER metric in the same task stays at n=3 -- EXCEPT
# quality_tests_pass, which is n=0/insufficient_reps for an entirely
# different, unrelated reason: the golden own quality.tests_pass is
# `null` (README: "null means the gate could not be executed"), excluded
# with reason gate_not_executed on every rep regardless of this fixture
# own loc mutation. Asserted explicitly here so this loop stays a
# genuine "only loc was affected" check, not a loop that silently skips
# the one metric that would expose a regression in the null-guard.
for metric_id, block in metrics.items():
    if metric_id.startswith("loc_"):
        continue
    if metric_id == "quality_tests_pass":
        assert block["n"] == 0, block
        assert block.get("insufficient_reps") is True, block
        assert all(e["reason"] == "gate_not_executed" for e in block["excluded"]), block
        continue
    assert block["n"] == 3, "%s: n=%r, want 3 (only loc metrics should be affected by this fixture)" % (metric_id, block["n"])
    assert not block.get("insufficient_reps"), "%s unexpectedly flagged insufficient_reps: %r" % (metric_id, block)
' "$out" "$item" || fail "h: assertion failed: $(cat "$out")"

	echo "TC-018h PASS"
}

# ---------------------------------------------------------------------------
# TC-018i (AC-14/AC-23 REQ-F-017/REQ-F-023): both halves. Two 3-rep fixture
# sets for the same-shaped item: one with oracle.f2p_resolved true on every
# rep, one with false on every rep -- both flagged non_discriminative in
# flags.non_discriminative_tasks[] (aggregate-side, T-E40-F03-004's own
# slice). Negative: a mixed true/true/false set is NOT flagged.
#
# T-E40-F03-005 closes this test's own deferred half: the same aggregate,
# rendered via report-baseline.sh, carries a heading whose text names
# "E40-F01" verbatim (REQ-F-023's own wording -- not merely "a
# corpus-feedback section exists"), lists item_true/item_false under it,
# and omits item_mixed.
# ---------------------------------------------------------------------------
test_i() {
	local root="$WORKDIR/i-root"
	local item_true="f03-fixture-tc018i-true" item_false="f03-fixture-tc018i-false" item_mixed="f03-fixture-tc018i-mixed"

	local rep
	for rep in 1 2 3; do
		place_record "$root" "$item_true" "$rep" --set 'oracle.f2p_resolved=true'
	done
	for rep in 1 2 3; do
		place_record "$root" "$item_false" "$rep" --set 'oracle.f2p_resolved=false'
	done
	place_record "$root" "$item_mixed" 1 --set 'oracle.f2p_resolved=true'
	place_record "$root" "$item_mixed" 2 --set 'oracle.f2p_resolved=true'
	place_record "$root" "$item_mixed" 3 --set 'oracle.f2p_resolved=false'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "i: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_true, item_false, item_mixed = sys.argv[2], sys.argv[3], sys.argv[4]
flagged = set(d["flags"]["non_discriminative_tasks"])

assert item_true in flagged, d["flags"]["non_discriminative_tasks"]
assert item_false in flagged, d["flags"]["non_discriminative_tasks"]
assert item_mixed not in flagged, d["flags"]["non_discriminative_tasks"]

task_true = next(t for t in d["tasks"] if t["item_id"] == item_true)
task_mixed = next(t for t in d["tasks"] if t["item_id"] == item_mixed)
assert task_true["non_discriminative"] is True, task_true
assert task_mixed["non_discriminative"] is False, task_mixed
' "$out" "$item_true" "$item_false" "$item_mixed" || fail "i: assertion failed: $(cat "$out")"
	echo "TC-018i(aggregate-side flag) PASS"

	# --- report-rendering half (T-E40-F03-005): the corpus-feedback
	# heading names "E40-F01" verbatim, and lists exactly the two
	# non-discriminative items, never the mixed one. ---
	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -eq 0 ]] || fail "i(report): report-baseline.sh exited $rcode, want 0: $(cat "$rerr")"

	grep -Eq '^#+ .*E40-F01' "$rout" || fail "i(report): no heading names E40-F01 verbatim: $(cat "$rout")"

	# Scope the item-name checks to the corpus-feedback SECTION only --
	# every item's own noise-band heading appears elsewhere in the report
	# regardless of non_discriminative status, so a whole-document grep for
	# item_mixed would find it there and false-positive.
	python3 -c '
import re
import sys

report_text, item_true, item_false, item_mixed = open(sys.argv[1]).read(), sys.argv[2], sys.argv[3], sys.argv[4]
m = re.search(r"## Corpus feedback to E40-F01\n(.*?)(\n## |\Z)", report_text, re.DOTALL)
assert m, "no \"## Corpus feedback to E40-F01\" section found: %r" % report_text
section = m.group(1)
assert item_true in section, "corpus-feedback section does not list %r: %r" % (item_true, section)
assert item_false in section, "corpus-feedback section does not list %r: %r" % (item_false, section)
assert item_mixed not in section, "corpus-feedback section lists %r, which is NOT non-discriminative: %r" % (item_mixed, section)
' "$rout" "$item_true" "$item_false" "$item_mixed" || fail "i(report): corpus-feedback section content check failed"

	echo "TC-018i(report-rendering half: heading names E40-F01 verbatim) PASS"

	echo "TC-018i PASS"
}

# ---------------------------------------------------------------------------
# TC-018j (AC-15, REQ-F-018, strict >): loc_files_touched (Class B,
# standalone -- no companion array to desynchronize) at 1, 1, 100 ->
# spread_abs=99, mean=34 -> 99 > 34 -> flagged unusable. Negative: a
# boundary fixture at 1, 2, 3 -> spread_abs=2 == mean=2 exactly -> NOT
# flagged (proves the comparison is strict >, not >=).
# ---------------------------------------------------------------------------
test_j() {
	local root="$WORKDIR/j-root"
	local item="f03-fixture-tc018j"
	place_record "$root" "$item" 1 --set loc.files_touched=1
	place_record "$root" "$item" 2 --set loc.files_touched=1
	place_record "$root" "$item" 3 --set loc.files_touched=100

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "j: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["loc_files_touched"]
assert block["spread_abs"] == 99, block
assert abs(block["mean"] - 34.0) < 1e-9, block
assert block.get("unusable") is True, block

flagged = {(f["item_id"], f["metric"]) for f in d["flags"]["unusable_metrics"]}
assert (item, "loc_files_touched") in flagged, d["flags"]["unusable_metrics"]
' "$out" "$item" || fail "j: assertion failed: $(cat "$out")"
	echo "TC-018j PASS"

	# --- Negative: spread_abs == mean exactly is NOT flagged. ---
	local root_neg="$WORKDIR/j-root-negative"
	local item_neg="f03-fixture-tc018j-negative"
	place_record "$root_neg" "$item_neg" 1 --set loc.files_touched=1
	place_record "$root_neg" "$item_neg" 2 --set loc.files_touched=2
	place_record "$root_neg" "$item_neg" 3 --set loc.files_touched=3

	local out_neg err_neg code_neg
	{
		read -r out_neg
		read -r err_neg
		read -r code_neg
	} < <(run_aggregate "$root_neg")
	[[ "$code_neg" -eq 0 ]] || fail "j(negative): exited $code_neg, want 0: $(cat "$err_neg")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["loc_files_touched"]
assert block["spread_abs"] == 2, block
assert abs(block["mean"] - 2.0) < 1e-9, block
assert not block.get("unusable"), "spread_abs == mean must NOT be flagged unusable (strict >, not >=): %r" % block
assert not any(f["item_id"] == item for f in d["flags"]["unusable_metrics"]), d["flags"]["unusable_metrics"]
' "$out_neg" "$item_neg" || fail "j(negative): assertion failed: $(cat "$out_neg")"

	echo "TC-018j(negative, spread_abs == mean is not flagged) PASS"
}

# ---------------------------------------------------------------------------
# TC-018k (AC-16, REQ-F-013): a fixture whose stages[] contains two
# entries both with status "in_development" (a rework loop, built via
# gen_fixtures.py's --duplicate-stage), each carrying distinct nonzero
# usage. step.in_development.tokens_input/.tokens_output/.cost_usd/
# .duration_ns equal the SUM of both occurrences. Negative: the golden's
# third stage (status in_qa) is not folded into in_development's total.
# ---------------------------------------------------------------------------
test_k() {
	local root="$WORKDIR/k-root"
	local item="f03-fixture-tc018k"
	local dir="$root/$item/default/rep-1"
	mkdir -p "$dir"
	python3 "$GEN_FIXTURES" --golden completed --item-id "$item" --rep 1 \
		--duplicate-stage 0 \
		--set 'stages[-1].usage.input_tokens=500' \
		--set 'stages[-1].usage.output_tokens=600' \
		--set 'stages[-1].usage.total_cost_usd=0.09' \
		--set 'stages[-1].usage.duration_api_ms=700' \
		--set 'stages[-1].usage.cache_read_input_tokens=10' \
		--set 'stages[-1].usage.cache_creation_input_tokens=5' \
		--set 'stages[-1].duration_ns=900000000' \
		>"$dir/record.jsonl"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "k: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
metrics = task["metrics"]

# Golden stage 1 (in_development): input_tokens=100, output_tokens=120,
# total_cost_usd=0.0185, duration_ns=1800000000. Duplicated stage:
# input_tokens=500, output_tokens=600, total_cost_usd=0.09,
# duration_ns=900000000. Sums computed independently from those two known
# sources, not copied from the implementation own output.
assert metrics["step.in_development.tokens_input"]["min"] == 100 + 500, metrics["step.in_development.tokens_input"]
assert metrics["step.in_development.tokens_output"]["min"] == 120 + 600, metrics["step.in_development.tokens_output"]
assert abs(metrics["step.in_development.cost_usd"]["min"] - (0.0185 + 0.09)) < 1e-9, metrics["step.in_development.cost_usd"]
assert metrics["step.in_development.duration_ns"]["min"] == 1800000000 + 900000000, metrics["step.in_development.duration_ns"]

# Negative: the in_qa stage (golden stage 3, untouched) is its own,
# separate step -- not folded into in_development own total.
assert metrics["step.in_qa.tokens_input"]["min"] == 175, metrics["step.in_qa.tokens_input"]
' "$out" "$item" || fail "k: assertion failed: $(cat "$out")"

	echo "TC-018k PASS"
}

# ---------------------------------------------------------------------------
# TC-018l (AC-17, REQ-F-014/ADR-F03-06): oracle.p2p_regressions_count and
# quality.tests_pass set to deliberately uncorrelated values across reps
# (0, 1, 0 vs. false/false/false on every rep, T-004's real-world shape).
# The regression-tracking metric (p2p_regressions_count) tracks
# oracle.p2p_regressions_count exactly. Negative: re-aggregate with
# tests_pass flipped to true on every rep -- p2p_regressions_count's own
# block is byte-identical across both runs; only quality_tests_pass own
# block (which legitimately tracks tests_pass as ITS OWN registered
# metric) may differ.
# ---------------------------------------------------------------------------
test_l() {
	local root="$WORKDIR/l-root"
	local item="f03-fixture-tc018l"
	place_record "$root" "$item" 1 --set 'oracle.p2p_regressions_count=0' --set 'quality.tests_pass=false'
	place_record "$root" "$item" 2 --set 'oracle.p2p_regressions_count=1' --set 'quality.tests_pass=false'
	place_record "$root" "$item" 3 --set 'oracle.p2p_regressions_count=0' --set 'quality.tests_pass=false'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "l: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["p2p_regressions_count"]
assert block["min"] == 0 and block["max"] == 1 and block["median"] == 0, block
' "$out" "$item" || fail "l: assertion failed: $(cat "$out")"

	# --- Negative: flip tests_pass to true on every rep; regression field
	# must be byte-identical, only quality_tests_pass own block differs. ---
	local root2="$WORKDIR/l-root2"
	place_record "$root2" "$item" 1 --set 'oracle.p2p_regressions_count=0' --set 'quality.tests_pass=true'
	place_record "$root2" "$item" 2 --set 'oracle.p2p_regressions_count=1' --set 'quality.tests_pass=true'
	place_record "$root2" "$item" 3 --set 'oracle.p2p_regressions_count=0' --set 'quality.tests_pass=true'

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root2")
	[[ "$code2" -eq 0 ]] || fail "l(negative): exited $code2, want 0: $(cat "$err2")"

	python3 -c '
import json
import sys

d1 = json.load(open(sys.argv[1]))
d2 = json.load(open(sys.argv[2]))
item = sys.argv[3]

task1 = next(t for t in d1["tasks"] if t["item_id"] == item)
task2 = next(t for t in d2["tasks"] if t["item_id"] == item)

assert task1["metrics"]["p2p_regressions_count"] == task2["metrics"]["p2p_regressions_count"], (
    "the regression field changed when only quality.tests_pass changed -- "
    "REQ-F-014 violation: %r vs %r" % (task1["metrics"]["p2p_regressions_count"], task2["metrics"]["p2p_regressions_count"])
)
assert task1["metrics"]["quality_tests_pass"] != task2["metrics"]["quality_tests_pass"], (
    "quality_tests_pass own block should differ (it legitimately tracks tests_pass as its own metric): %r" % task1["metrics"]["quality_tests_pass"]
)
' "$out" "$out2" "$item" || fail "l(negative): assertion failed"

	echo "TC-018l PASS"
}

# ---------------------------------------------------------------------------
# TC-018m (AC-18, REQ-F-015): (a) 3 reps, rejections.by_gate present on
# every rep but one rep's by_gate is emptied (omitting the "in_qa" gate
# another rep carries) -- that rep contributes 0 for rejections_by_gate.
# in_qa, still counted in n (not excluded). (b) one rep's WHOLE rejections
# block absent -- contributes nothing to any rejections_* metric, listed
# in excluded[] with a named reason.
# ---------------------------------------------------------------------------
test_m() {
	local root_a="$WORKDIR/m-root-a"
	local item_a="f03-fixture-tc018m-a"
	place_record "$root_a" "$item_a" 1
	place_record "$root_a" "$item_a" 2 --set 'rejections.by_gate={}'
	place_record "$root_a" "$item_a" 3

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -eq 0 ]] || fail "m(a): exited $code_a, want 0: $(cat "$err_a")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["rejections_by_gate.in_qa"]
assert block["n"] == 3, "n=%r, want 3 -- the omitted-key rep still COUNTS (REQ-F-015 zero-fill, not excluded)" % block["n"]
assert block["excluded"] == [], "the omitted-key rep must not appear in excluded[]: %r" % block["excluded"]
assert block["min"] == 0, block  # golden own by_gate.in_qa=1; the emptied rep contributes 0
' "$out_a" "$item_a" || fail "m(a): assertion failed: $(cat "$out_a")"
	echo "TC-018m(a, omitted gate key within a present block zero-fills, still counted) PASS"

	local root_b="$WORKDIR/m-root-b"
	local item_b="f03-fixture-tc018m-b"
	place_record "$root_b" "$item_b" 1
	place_record "$root_b" "$item_b" 2 --unset rejections
	place_record "$root_b" "$item_b" 3

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -eq 0 ]] || fail "m(b): exited $code_b, want 0 (rejections is not one of the EXPECTED_FAMILIES that drive classification): $(cat "$err_b")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
rep2_key = item + "::default::rep2"

for metric_id in ("rejections_by_gate.in_qa", "rejections_rework_loops"):
    block = task["metrics"][metric_id]
    assert block["n"] == 2, "%s: n=%r, want 2 (the whole-block-absent rep contributes nothing)" % (metric_id, block["n"])
    excluded_run_keys = {e["run_key"] for e in block["excluded"]}
    assert rep2_key in excluded_run_keys, "%s: rep2 (whole rejections absent) missing from excluded[]: %r" % (metric_id, block["excluded"])

# The record itself is still "complete" (rejections is not one of the
# three EXPECTED_FAMILIES that drive classification).
assert d["inventory"][rep2_key]["classification"] == "complete", d["inventory"][rep2_key]
' "$out_b" "$item_b" || fail "m(b): assertion failed: $(cat "$out_b")"
	echo "TC-018m(b, whole rejections block absent excludes, never zero-fills) PASS"

	echo "TC-018m PASS"
}

# ---------------------------------------------------------------------------
# TC-018r (REQ-F-019, input_digest is COMPUTED, not echoed): (a) mutation
# sensitivity -- one byte of one contributing record.jsonl mutated ->
# input_digest differs. (b) order independence -- the same fixture
# root's files touched in reverse-mtime order -> input_digest identical.
# (c) location independence -- the same fixture content copied to a
# second absolute path -> input_digest identical (relpath, not absolute
# path, is what the digest is rooted at).
# ---------------------------------------------------------------------------
test_r() {
	local root="$WORKDIR/r-root"
	local item="f03-fixture-tc018r"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2
	place_record "$root" "$item" 3

	local out1 err1 code1
	{
		read -r out1
		read -r err1
		read -r code1
	} < <(run_aggregate "$root")
	[[ "$code1" -eq 0 ]] || fail "r: run 1 exited $code1, want 0: $(cat "$err1")"
	local digest1
	digest1="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["input_digest"])' "$out1")"

	# --- (a) mutation sensitivity ---
	python3 -c '
import json
import sys

path = sys.argv[1]
rec = json.loads(open(path).read().strip())
rec["loc"]["prod_added"] = rec["loc"]["prod_added"] + 1
open(path, "w").write(json.dumps(rec, sort_keys=True) + "\n")
' "$root/$item/default/rep-1/record.jsonl"

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root")
	[[ "$code2" -eq 0 ]] || fail "r(a): run after mutation exited $code2, want 0: $(cat "$err2")"
	local digest2
	digest2="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["input_digest"])' "$out2")"
	[[ "$digest1" != "$digest2" ]] || fail "r(a): input_digest unchanged after mutating one contributing record.jsonl byte -- REQ-F-019 violation"
	echo "TC-018r(a, mutation sensitivity) PASS"

	# --- (b) order independence: touch files in reverse-mtime order ---
	local root_ord="$WORKDIR/r-root-order"
	place_record "$root_ord" "$item" 1
	place_record "$root_ord" "$item" 2
	place_record "$root_ord" "$item" 3
	touch -d "2020-01-03" "$root_ord/$item/default/rep-1/record.jsonl"
	touch -d "2020-01-02" "$root_ord/$item/default/rep-2/record.jsonl"
	touch -d "2020-01-01" "$root_ord/$item/default/rep-3/record.jsonl"

	local out3 err3 code3
	{
		read -r out3
		read -r err3
		read -r code3
	} < <(run_aggregate "$root_ord")
	[[ "$code3" -eq 0 ]] || fail "r(b): exited $code3, want 0: $(cat "$err3")"
	local digest3
	digest3="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["input_digest"])' "$out3")"
	[[ "$digest1" == "$digest3" ]] || fail "r(b): input_digest differs after only reordering on-disk mtimes -- REQ-F-019's sorted-lines rule violated"
	echo "TC-018r(b, order independence) PASS"

	# --- (c) location independence: same content, different absolute root ---
	local root_copy="$WORKDIR/r-root-copy"
	cp -r "$root" "$root_copy"

	local out4 err4 code4
	{
		read -r out4
		read -r err4
		read -r code4
	} < <(run_aggregate "$root_copy")
	[[ "$code4" -eq 0 ]] || fail "r(c): exited $code4, want 0: $(cat "$err4")"
	local digest4
	digest4="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["input_digest"])' "$out4")"
	[[ "$digest2" == "$digest4" ]] || fail "r(c): input_digest differs when the identical artifact set is aggregated from a different absolute --root -- relpath rooting violated"
	echo "TC-018r(c, location independence) PASS"

	echo "TC-018r PASS"
}

# ---------------------------------------------------------------------------
# TC-018n (AC-19, REQ-N-004): report-baseline.sh determinism -- the AC-12
# fixture's aggregate.json, rendered twice as two fully separate process
# invocations, is byte-identical, and carries no wall-clock value the
# script itself introduced. The "no timestamp" property is checked more
# precisely than a blind date/time grep would allow: `manifest.
# shark_version` (verbatim in the aggregate's provenance, REQ-F-022) is a
# free-text producer string that legitimately embeds a build date (see the
# committed I-02 golden), so a literal date/time-shaped substring CAN
# appear in a correct report. What must never happen is the SCRIPT adding
# one that isn't already in the input. So: find every date/time-shaped
# substring in the report, and require each one to already appear
# verbatim in the source aggregate.json's own text -- proving it is
# reproduced provenance, not a fabricated "generated at" stamp (the
# counter-factual AC-19 names).
# ---------------------------------------------------------------------------
test_n() {
	local root="$WORKDIR/n-root"
	local aggregate_path="$WORKDIR/n-aggregate.json"
	build_ac12_aggregate "$root" "$aggregate_path" "f03-fixture-tc018n"

	local out1 err1 code1
	{
		read -r out1
		read -r err1
		read -r code1
	} < <(run_report "$aggregate_path")
	[[ "$code1" -eq 0 ]] || fail "n: run 1 exited $code1, want 0: $(cat "$err1")"

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_report "$aggregate_path")
	[[ "$code2" -eq 0 ]] || fail "n: run 2 exited $code2, want 0: $(cat "$err2")"

	diff "$out1" "$out2" >/dev/null || fail "n: two invocations of report-baseline.sh are not byte-identical"
	echo "TC-018n(byte-identical across two invocations) PASS"

	python3 -c '
import re
import sys

report_path, aggregate_path = sys.argv[1], sys.argv[2]
report_text = open(report_path).read()
aggregate_text = open(aggregate_path).read()

pattern = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}|[0-9]{2}:[0-9]{2}:[0-9]{2}")
matches = pattern.findall(report_text)
for m in matches:
    assert m in aggregate_text, (
        "report contains a date/time-shaped substring %r that does not "
        "appear anywhere in the source aggregate.json -- the script "
        "fabricated it rather than reproducing input provenance" % m
    )
' "$out1" "$aggregate_path" || fail "n: date/time-shaped substring check failed"

	echo "TC-018n(no fabricated timestamp -- any date/time-shaped substring traces to the aggregate itself) PASS"
	echo "TC-018n PASS"
}

# ---------------------------------------------------------------------------
# TC-018o (AC-20, REQ-F-021/REQ-F-024): the AC-12 aggregate, rendered via
# report-baseline.sh. For each of the three ADR-F03-04 Class C branches
# TC-018g's own fixture exercises (loc_prod_added: r>0; wall_clock_ns:
# r==0, median!=0; loc_test_deleted: identically zero), the report states
# both the observed band and the acceptance interval, and the interval
# values are recomputed HERE from the aggregate's own min/median/max
# (never a literal copied from report-baseline.sh's own implementation --
# test-plan.md's own admonition for this fixture) and checked against what
# the report actually printed. Also asserts the closed 4-caveat set by
# content (TD-079, TD-081, T-004, and the timeout-exclusion rule) -- a
# report missing even one fails, per test-plan.md's own "closed-set check"
# framing.
# ---------------------------------------------------------------------------
test_o() {
	local root="$WORKDIR/o-root"
	local aggregate_path="$WORKDIR/o-aggregate.json"
	build_ac12_aggregate "$root" "$aggregate_path" "f03-fixture-tc018o"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_report "$aggregate_path")
	[[ "$code" -eq 0 ]] || fail "o: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import re
import sys

aggregate_path, report_path, item = sys.argv[1], sys.argv[2], sys.argv[3]
agg = json.load(open(aggregate_path))
report_text = open(report_path).read()

task = next(t for t in agg["tasks"] if t["item_id"] == item)
metrics = task["metrics"]


def metric_block_text(metric_id):
    marker = "#### `%s`" % metric_id
    start = report_text.index(marker)
    rest = report_text[start:]
    nxt = re.search(r"\n#### ", rest[1:])
    return rest[: nxt.start() + 1] if nxt else rest


def interval_from_report(text):
    m = re.search(r"Acceptance interval: \[([^,]+), ([^\]]+)\]", text)
    assert m, "no Acceptance interval line found: %r" % text
    return float(m.group(1)), float(m.group(2))


# (i) loc_prod_added: r > 0 branch -- interval independently recomputed
# from the aggregate own min/max, never copied from the implementation.
block = metrics["loc_prod_added"]
text = metric_block_text("loc_prod_added")
mn, mx = block["min"], block["max"]
r = mx - mn
expected_lo, expected_hi = mn - r, mx + r
lo, hi = interval_from_report(text)
assert abs(lo - expected_lo) < 1e-6 and abs(hi - expected_hi) < 1e-6, (lo, hi, expected_lo, expected_hi)
assert str(mn) in text and str(mx) in text and str(block["median"]) in text, text
assert "r = max - min" in text or "max - min" in text, "no derivation rule sentence for the r>0 branch: %r" % text

# (ii) wall_clock_ns: r == 0, median != 0 branch.
block = metrics["wall_clock_ns"]
text = metric_block_text("wall_clock_ns")
assert block["spread_abs"] == 0 and block["median"] != 0, block
r_eff = 0.10 * abs(block["median"])
expected_lo, expected_hi = block["min"] - r_eff, block["max"] + r_eff
lo, hi = interval_from_report(text)
assert abs(lo - expected_lo) < 1e-3 and abs(hi - expected_hi) < 1e-3, (lo, hi, expected_lo, expected_hi)
assert "10%" in text, "no 10%%-of-median derivation sentence for the zero-spread/nonzero-median branch: %r" % text

# (iii) loc_test_deleted: identically-zero branch -- exact [0, 0].
block = metrics["loc_test_deleted"]
text = metric_block_text("loc_test_deleted")
assert block["min"] == 0 and block["max"] == 0 and block["median"] == 0, block
lo, hi = interval_from_report(text)
assert lo == 0.0 and hi == 0.0, (lo, hi)
assert "identically zero" in text, "no identically-zero derivation sentence: %r" % text

# Class A metric: accept_set, never accept_lo/accept_hi.
text = metric_block_text("oracle_f2p_resolved")
assert "Acceptance set" in text, text
assert "Acceptance interval" not in text, text

# Closed 4-caveat set, by content -- a report missing even one fails.
assert "TD-079" in report_text, "TD-079 caveat missing"
assert "TD-081" in report_text, "TD-081 caveat missing"
assert "T-004" in report_text, "T-004 caveat missing"
assert "timeout" in report_text.lower() and "band" in report_text.lower(), "timeout-exclusion caveat missing"
' "$aggregate_path" "$out" "f03-fixture-tc018o" || fail "o: assertion failed: $(cat "$out")"

	echo "TC-018o PASS"
}

# ---------------------------------------------------------------------------
# TC-018p (AC-21, REQ-F-022): the AC-12 aggregate's provenance block
# (model_ids, fixture_base_sha, variant_bundle_sha256, corpus_schema_
# version, shark_version, reps, input_digest), reproduced verbatim in the
# report -- values read from the aggregate itself, never hardcoded.
# Separately, the aggregate's OWN `baseline_id` field is checked against
# the documented format `<variant_id>-<fixture_base_sha[:12]>-r<reps>`
# directly (not through the report) -- the format-string gap test-plan.md
# names explicitly (no AC states it).
# ---------------------------------------------------------------------------
test_p() {
	local root="$WORKDIR/p-root"
	local aggregate_path="$WORKDIR/p-aggregate.json"
	build_ac12_aggregate "$root" "$aggregate_path" "f03-fixture-tc018p"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_report "$aggregate_path")
	[[ "$code" -eq 0 ]] || fail "p: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import re
import sys

aggregate_path, report_path = sys.argv[1], sys.argv[2]
agg = json.load(open(aggregate_path))
report_text = open(report_path).read()
prov = agg["provenance"]

for value in (
    prov["model_ids"][0],
    prov["fixture_base_sha"],
    prov["variant_bundle_sha256"],
    prov["corpus_schema_version"],
    prov["shark_version"],
    str(prov["reps"]),
    agg["input_digest"],
):
    assert value in report_text, "provenance value %r not reproduced verbatim in the report" % value

# The aggregate own baseline_id format: <variant_id>-<fixture_base_sha[:12]>-r<reps>.
expected_baseline_id = "%s-%s-r%d" % ("default", prov["fixture_base_sha"][:12], prov["reps"])
assert agg["baseline_id"] == expected_baseline_id, (agg["baseline_id"], expected_baseline_id)
assert re.match(r"^[^-]+-[0-9a-f]{12}-r[0-9]+$", agg["baseline_id"]), agg["baseline_id"]
' "$aggregate_path" "$out" || fail "p: assertion failed: $(cat "$out")"

	echo "TC-018p PASS"
}

# ---------------------------------------------------------------------------
# TC-018u (Interface-contracts exit table, report-baseline.sh's non-zero
# row: "Aggregate unreadable or unsupported schema_version" -- distinct
# from AC-07, which is the RECORD's schema_version, not the aggregate's).
# Three sub-cases: (a) an aggregate.json with an unsupported schema_
# version, (b) unparseable JSON, (c) a nonexistent --aggregate path. All
# three exit non-zero with NOTHING written to stdout -- the whole input is
# validated before any output line is built (REQ-F-020's pure-function
# framing), so a caller redirecting stdout to report.md is never left with
# a partial file.
# ---------------------------------------------------------------------------
test_u() {
	# --- (a) unsupported schema_version ---
	local bad_schema="$WORKDIR/u-bad-schema.json"
	python3 -c 'import json,sys; json.dump({"schema_version":"99.0"}, open(sys.argv[1],"w"))' "$bad_schema"

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_report "$bad_schema")
	[[ "$code" -ne 0 ]] || fail "u(a): exited 0, want non-zero (unsupported schema_version)"
	[[ ! -s "$out" ]] || fail "u(a): stdout is non-empty on failure, want nothing written: $(cat "$out")"
	grep -q "99.0" "$err" || fail "u(a): stderr does not name the unsupported schema_version: $(cat "$err")"
	echo "TC-018u(a, unsupported schema_version) PASS"

	# --- (b) unparseable JSON ---
	local bad_json="$WORKDIR/u-bad-json.json"
	printf '{not valid json' >"$bad_json"

	{
		read -r out
		read -r err
		read -r code
	} < <(run_report "$bad_json")
	[[ "$code" -ne 0 ]] || fail "u(b): exited 0, want non-zero (unparseable JSON)"
	[[ ! -s "$out" ]] || fail "u(b): stdout is non-empty on failure, want nothing written: $(cat "$out")"
	echo "TC-018u(b, unparseable JSON) PASS"

	# --- (c) nonexistent --aggregate path ---
	local missing_path="$WORKDIR/u-does-not-exist.json"

	{
		read -r out
		read -r err
		read -r code
	} < <(run_report "$missing_path")
	[[ "$code" -ne 0 ]] || fail "u(c): exited 0, want non-zero (missing aggregate file)"
	[[ ! -s "$out" ]] || fail "u(c): stdout is non-empty on failure, want nothing written: $(cat "$out")"
	echo "TC-018u(c, unreadable/missing aggregate) PASS"

	echo "TC-018u PASS"
}

# ---------------------------------------------------------------------------
# TC-018v (REQ-N-007/Q005, T-E40-F03-006's task spec): a content-only read
# of bench/README.md's "Baseline aggregation, noise band, and replay"
# section -- both REQ-N-007 preconditions must be stated in substance.
# Distinctive substrings, not whole sentences, so a copy edit doesn't break
# this while a missing section still fails (matches F02's own TC-017
# treatment of its analogous documentation AC; no decision-table or
# mutation test simulates the meaning of the prose itself).
# ---------------------------------------------------------------------------
test_v() {
	grep -q "Baseline aggregation, noise band, and replay" "$README_PATH" ||
		fail "v: bench/README.md has no 'Baseline aggregation, noise band, and replay' section"

	# (i) ledger retention: never delete bench/corpus/ledgers/<sha>/ for any
	# SHA a published manifest references.
	grep -q "ledgers/" "$README_PATH" || fail "v: README does not mention bench/corpus/ledgers/"
	grep -qi "never.*delet" "$README_PATH" || fail "v: README does not state the ledger-retention rule (never delete)"

	# (ii) corpus item immutability: seed file and held-back F2P files are
	# immutable for any SHA a published manifest references.
	grep -qi "immutable" "$README_PATH" || fail "v: README does not state the corpus-item-immutability precondition"
	grep -qi "seed" "$README_PATH" || fail "v: README's immutability precondition does not mention the seed file"
	grep -qi "F2P" "$README_PATH" || fail "v: README's immutability precondition does not mention F2P test files"

	echo "TC-018v PASS"
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
test_q
test_r
test_s
test_t
test_n
test_o
test_p
test_u
test_v

echo "TC-018 PASS"
